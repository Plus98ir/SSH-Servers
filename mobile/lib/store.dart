// App state: server list, sync, device-only data (passwords, host keys, key login flags).
import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'l10n.dart';
import 'sync.dart';

const colors = ['#C00000', '#E8590C', '#F2B705', '#008000', '#12B886', '#1C7ED6', '#7048E8', '#D6336C', '#495057'];

class AppState extends ChangeNotifier {
  static final AppState I = AppState._();
  AppState._();

  late SharedPreferences _p;
  final _sec = const FlutterSecureStorage(aOptions: AndroidOptions(encryptedSharedPreferences: true));

  SyncDoc doc = SyncDoc();
  SyncCfg? sync;
  int lastSync = 0;
  String syncErr = '';
  bool busy = false;
  Map<String, bool> keyOK = {}; // server id -> key login works (this phone only)
  Map<String, String> hostKeys = {}; // host:port -> fingerprint
  double fontSize = 13;
  bool keepAwake = true;

  /// shown as a toast by the UI
  final messages = StreamController<(String, bool)>.broadcast();

  Future<void> load() async {
    _p = await SharedPreferences.getInstance();
    lang = _p.getString('lang') ?? (PlatformDispatcher.instance.locale.languageCode == 'fa' ? 'fa' : 'en');
    final d = _p.getString('doc');
    if (d != null) {
      try {
        doc = SyncDoc.fromJson(jsonDecode(d) as Map<String, dynamic>);
      } catch (_) {}
    }
    keyOK = Map<String, bool>.from(jsonDecode(_p.getString('keyOK') ?? '{}') as Map);
    hostKeys = Map<String, String>.from(jsonDecode(_p.getString('hostKeys') ?? '{}') as Map);
    lastSync = _p.getInt('lastSync') ?? 0;
    fontSize = _p.getDouble('fontSize') ?? 13;
    keepAwake = _p.getBool('keepAwake') ?? true;
    final s = await _sec.read(key: 'sync');
    if (s != null) sync = SyncCfg.fromJson(jsonDecode(s) as Map<String, dynamic>);
    Timer.periodic(const Duration(minutes: 1), (_) {
      if (sync != null && _foreground) runSync();
    });
    if (sync != null) runSync();
  }

  bool _foreground = true;
  void setForeground(bool v) {
    _foreground = v;
    if (v) syncSoon(const Duration(milliseconds: 300));
  }

  void setLang(String l) {
    lang = l;
    _p.setString('lang', l);
    notifyListeners();
  }

  void setFont(double v) {
    fontSize = v;
    _p.setDouble('fontSize', v);
    notifyListeners();
  }

  void setKeepAwake(bool v) {
    keepAwake = v;
    _p.setBool('keepAwake', v);
    notifyListeners();
  }

  void _saveDoc() => _p.setString('doc', jsonEncode(doc.toJson()));

  // ---------------------------------------------------------------- edits (every edit is stamped; newest wins everywhere)
  int get _now => DateTime.now().millisecondsSinceEpoch;

  void saveServer(Server s) {
    s.u = _now;
    final i = doc.servers.indexWhere((x) => x.id == s.id);
    if (i >= 0) {
      final old = doc.servers[i];
      if (old.host != s.host || old.user != s.user || old.port != s.port) setKeyOK(s.id, false);
      doc.servers[i] = s;
    } else {
      doc.servers.add(s);
    }
    _changed();
  }

  void deleteServer(String id) {
    doc.servers.removeWhere((x) => x.id == id);
    doc.deleted[id] = _now;
    forgetPassword(id);
    _changed();
  }

  void move(String id, int d) {
    final i = doc.servers.indexWhere((x) => x.id == id), j = i + d;
    if (i < 0 || j < 0 || j >= doc.servers.length) return;
    final tmp = doc.servers[i];
    doc.servers[i] = doc.servers[j];
    doc.servers[j] = tmp;
    doc.orderU = _now;
    _changed();
  }

  void _changed() {
    _saveDoc();
    notifyListeners();
    syncSoon();
  }

  // ---------------------------------------------------------------- device-only data
  void setKeyOK(String id, bool v) {
    keyOK[id] = v;
    _p.setString('keyOK', jsonEncode(keyOK));
    notifyListeners();
  }

  void setHostKey(String hp, String fp) {
    hostKeys[hp] = fp;
    _p.setString('hostKeys', jsonEncode(hostKeys));
  }

  Future<String?> password(String id) => _sec.read(key: 'pw.$id');
  Future<void> savePassword(String id, String pw) => _sec.write(key: 'pw.$id', value: pw);
  Future<void> forgetPassword(String id) => _sec.delete(key: 'pw.$id');
  Future<String?> secret(String k) => _sec.read(key: k);
  Future<void> setSecret(String k, String v) => _sec.write(key: k, value: v);

  // ---------------------------------------------------------------- sync
  Timer? _soon;
  bool _again = false;

  void syncSoon([Duration d = const Duration(milliseconds: 900)]) {
    if (sync == null) return;
    _soon?.cancel();
    _soon = Timer(d, runSync);
  }

  String errText(Object e) {
    if (e is SyncError) {
      switch (e.code) {
        case 'auth':
          return t('eAuth');
        case 'notfound':
          return t('eNotFound');
        case 'badkey':
          return t('eKey');
        case 'net':
          return t('eNet');
      }
      return e.msg.isEmpty ? e.code : e.msg;
    }
    return '$e';
  }

  Future<void> runSync({bool manual = false}) async {
    final sc = sync;
    if (sc == null) return;
    if (busy) {
      _again = true;
      return;
    }
    busy = true;
    notifyListeners();
    try {
      final before = doc.canon();
      final (merged, changed) = await syncOnce(sc, doc);
      if (doc.canon() != before) {
        _again = true; // edited while syncing: keep both
        doc = mergeDocs(doc, merged);
      } else {
        doc = merged;
      }
      _saveDoc();
      lastSync = _now;
      _p.setInt('lastSync', lastSync);
      syncErr = '';
      if (changed) {
        messages.add((t('got'), false));
      } else if (manual) {
        messages.add((t('synced'), false));
      }
    } catch (e) {
      syncErr = errText(e);
      if (manual || !(e is SyncError && e.code == 'net')) messages.add((syncErr, true));
    } finally {
      busy = false;
      notifyListeners();
      if (_again) {
        _again = false;
        syncSoon(const Duration(milliseconds: 300));
      }
    }
  }

  Future<void> setSync(SyncCfg sc) async {
    sync = sc;
    syncErr = '';
    await _sec.write(key: 'sync', value: jsonEncode(sc.toJson()));
    notifyListeners();
    await runSync(manual: true);
  }

  /// After "Sign in with GitHub" or a pasted token. Returns true when a sync already exists (needs QR or replace).
  Future<bool> useToken(String token, {bool replace = false}) async {
    final r = await fromToken(token, doc, replace: replace);
    if (r.exists) return true;
    sync = r.cfg;
    syncErr = '';
    lastSync = _now;
    _p.setInt('lastSync', lastSync);
    await _sec.write(key: 'sync', value: jsonEncode(r.cfg!.toJson()));
    notifyListeners();
    messages.add((t('synced'), false));
    return false;
  }

  Future<void> stopSync() async {
    sync = null;
    syncErr = '';
    await _sec.delete(key: 'sync');
    notifyListeners();
  }
}
