// SSH sessions: this phone's own ed25519 key, host-key check, password fallback, key install.
import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dartssh2/dartssh2.dart';
import 'package:flutter/foundation.dart';
import 'package:pinenacl/tweetnacl.dart';
import 'package:xterm/xterm.dart';

import 'store.dart';
import 'sync.dart';

// ---------------------------------------------------------------- device key

class DeviceKey {
  final Uint8List pub; // 32
  final Uint8List priv; // 64 = seed || pub
  DeviceKey(this.pub, this.priv);

  static DeviceKey? _cached;

  static Future<DeviceKey> get() async {
    if (_cached != null) return _cached!;
    final st = AppState.I;
    final saved = await st.secret('devkey');
    if (saved != null) {
      final sk = base64.decode(saved);
      _cached = DeviceKey(Uint8List.fromList(sk.sublist(32)), Uint8List.fromList(sk));
    } else {
      final pk = Uint8List(32), sk = Uint8List(64);
      TweetNaCl.crypto_sign_keypair(pk, sk, randomBytes(32));
      await st.setSecret('devkey', base64.encode(sk));
      _cached = DeviceKey(pk, sk);
    }
    return _cached!;
  }

  static void _str(BytesBuilder b, List<int> v) {
    b.add(_u32(v.length));
    b.add(v);
  }

  static Uint8List _u32(int n) => Uint8List(4)..buffer.asByteData().setUint32(0, n);

  Uint8List get blob {
    final b = BytesBuilder();
    _str(b, utf8.encode('ssh-ed25519'));
    _str(b, pub);
    return b.toBytes();
  }

  String get authorizedLine => 'ssh-ed25519 ${base64.encode(blob)} ssh-servers-android';

  /// OpenSSH private key (openssh-key-v1, unencrypted) so dartssh2 can load it.
  String get pem {
    final check = randomBytes(4);
    final p = BytesBuilder()
      ..add(check)
      ..add(check);
    _str(p, utf8.encode('ssh-ed25519'));
    _str(p, pub);
    _str(p, priv);
    _str(p, utf8.encode('ssh-servers-android'));
    var i = 1;
    while (p.length % 8 != 0) {
      p.addByte(i++);
    }
    final b = BytesBuilder()..add(utf8.encode('openssh-key-v1\x00'));
    _str(b, utf8.encode('none'));
    _str(b, utf8.encode('none'));
    _str(b, const []);
    b.add(_u32(1));
    _str(b, blob);
    _str(b, p.toBytes());
    final s = base64.encode(b.toBytes());
    final lines = [for (var j = 0; j < s.length; j += 70) s.substring(j, j + 70 > s.length ? s.length : j + 70)];
    return '-----BEGIN OPENSSH PRIVATE KEY-----\n${lines.join('\n')}\n-----END OPENSSH PRIVATE KEY-----\n';
  }

  List<SSHKeyPair> get identities => SSHKeyPair.fromPem(pem);
}

/// dartssh2 hands over the fingerprint as text ("SHA256:...") or raw bytes; show it like ssh does.
String fpText(Uint8List fp) {
  if (fp.every((b) => b >= 0x20 && b < 0x7f)) return String.fromCharCodes(fp);
  return fp.map((b) => b.toRadixString(16).padLeft(2, '0')).join(':');
}

// ---------------------------------------------------------------- session

enum SessState { connecting, connected, closed }

class AuthNeeded implements Exception {}

class HostKeyRejected implements Exception {}

/// UI hooks a session needs while connecting.
abstract class SessUI {
  /// first == true: never seen; false: changed. Return true to accept.
  Future<bool> confirmHostKey(Server s, String fp, bool first);
}

class Sess extends ChangeNotifier {
  final Server server;
  final Terminal term = Terminal(maxLines: 5000);
  SSHClient? _client;
  SSHSession? _shell;
  SessState state = SessState.connecting;

  /// sticky Ctrl from the key bar: applies to the next typed key
  final ctrl = ValueNotifier<bool>(false);
  String error = '';

  Sess(this.server);

  static final List<Sess> open = [];
  static final changes = ValueNotifier<int>(0);

  /// Connects; throws AuthNeeded when the key didn't work and no password is known.
  Future<void> connect(SessUI ui, {String? password}) async {
    state = SessState.connecting;
    error = '';
    notifyListeners();
    final st = AppState.I;
    final key = await DeviceKey.get();
    password ??= await st.password(server.id);
    final hp = '${server.host}:${server.port}';
    SSHClient? client;
    try {
      final socket = await SSHSocket.connect(server.host, server.port, timeout: const Duration(seconds: 12));
      client = SSHClient(
        socket,
        username: server.user.isEmpty ? 'root' : server.user,
        identities: key.identities,
        onPasswordRequest: () => password,
        onUserInfoRequest: (req) => password == null ? null : [for (final _ in req.prompts) password],
        onVerifyHostKey: (type, fp) async {
          final f = fpText(fp);
          final known = st.hostKeys[hp];
          if (known == f) return true;
          final ok = await ui.confirmHostKey(server, f, known == null);
          if (ok) st.setHostKey(hp, f);
          return ok;
        },
      );
      await client.authenticated;
    } on SSHAuthFailError {
      client?.close();
      throw AuthNeeded();
    } on SSHHostkeyError {
      client?.close();
      throw HostKeyRejected();
    } catch (e) {
      client?.close();
      final m = '$e';
      if (m.contains('auth') && m.toLowerCase().contains('fail')) throw AuthNeeded();
      rethrow;
    }
    _client = client;
    final shell = await client.shell(pty: SSHPtyConfig(width: term.viewWidth, height: term.viewHeight));
    _shell = shell;
    state = SessState.connected;
    notifyListeners();
    term.onOutput = (data) {
      if (ctrl.value && data.length == 1) {
        final c = data.toLowerCase().codeUnitAt(0);
        if (c >= 0x61 && c <= 0x7a) data = String.fromCharCode(c - 0x60); // ctrl+letter
        if (data == ' ' || data == '2') data = '\x00';
        if (data == '[') data = '\x1b';
        ctrl.value = false;
      }
      shell.write(Uint8List.fromList(utf8.encode(data)));
    };
    term.onResize = (w, h, pw, ph) => shell.resizeTerminal(w, h, pw, ph);
    shell.stdout.cast<List<int>>().transform(const Utf8Decoder(allowMalformed: true)).listen(term.write);
    shell.stderr.cast<List<int>>().transform(const Utf8Decoder(allowMalformed: true)).listen(term.write);
    shell.done.whenComplete(() {
      state = SessState.closed;
      notifyListeners();
      changes.value++;
    });
  }

  /// Puts this phone's key into ~/.ssh/authorized_keys.
  Future<void> installKey() async {
    final c = _client;
    if (c == null) return;
    final line = (await DeviceKey.get()).authorizedLine;
    final cmd = "umask 077; mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys && "
        "(grep -qxF '$line' ~/.ssh/authorized_keys || echo '$line' >> ~/.ssh/authorized_keys) && echo SSHSERVERS_OK";
    final out = utf8.decode(await c.run(cmd), allowMalformed: true);
    if (!out.contains('SSHSERVERS_OK')) throw Exception(out.trim().isEmpty ? 'failed' : out.trim());
    AppState.I.setKeyOK(server.id, true);
  }

  void send(String s) {
    final sh = _shell;
    if (sh != null && state == SessState.connected) sh.write(Uint8List.fromList(utf8.encode(s)));
  }

  void close() {
    try {
      _shell?.close();
      _client?.close();
    } catch (_) {}
    state = SessState.closed;
    open.remove(this);
    changes.value++;
    notifyListeners();
  }
}
