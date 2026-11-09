// Sync core: same format + merge rules as sync.go (Windows) and docs/app/sync.js (web).
// Gist file ssh-servers.json = {app,v,iv,ct}: AES-256-GCM over
// {v, servers:[{id,name,user,host,port,color,legacy,u}], deleted:{id:ms}, orderU}.
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:http/http.dart' as http;
import 'package:pointycastle/export.dart';

const gistFile = 'ssh-servers.json';
const githubClientId = 'Ov23lixhLgGnIlMRameI';
const tombLife = Duration(days: 180);

class Server {
  String id, name, user, host, color;
  int port;
  bool legacy;
  int u;
  Server({required this.id, required this.name, required this.user, required this.host, this.port = 22, this.color = '#495057', this.legacy = false, this.u = 0});

  factory Server.fromJson(Map<String, dynamic> j) => Server(
        id: '${j['id'] ?? ''}',
        name: '${j['name'] ?? ''}',
        user: '${j['user'] ?? ''}',
        host: '${j['host'] ?? ''}',
        port: (j['port'] is num) ? (j['port'] as num).toInt() : int.tryParse('${j['port']}') ?? 22,
        color: '${j['color'] ?? '#495057'}',
        legacy: j['legacy'] == true,
        u: (j['u'] is num) ? (j['u'] as num).toInt() : 0,
      );

  Map<String, dynamic> toJson() => {
        'id': id, 'name': name, 'user': user, 'host': host, 'port': port, 'color': color,
        if (legacy) 'legacy': true,
        'u': u,
      };

  Server copy() => Server.fromJson(toJson());
  String get addr => '${user.isNotEmpty ? '$user@' : ''}$host${port != 22 ? ':$port' : ''}';
}

class SyncDoc {
  List<Server> servers;
  Map<String, int> deleted;
  int orderU;
  SyncDoc({List<Server>? servers, Map<String, int>? deleted, this.orderU = 0})
      : servers = servers ?? [],
        deleted = deleted ?? {};

  factory SyncDoc.fromJson(Map<String, dynamic> j) => SyncDoc(
        servers: ((j['servers'] as List?) ?? []).map((e) => Server.fromJson(Map<String, dynamic>.from(e as Map))).toList(),
        deleted: ((j['deleted'] as Map?) ?? {}).map((k, v) => MapEntry('$k', (v as num).toInt())),
        orderU: (j['orderU'] is num) ? (j['orderU'] as num).toInt() : 0,
      );

  Map<String, dynamic> toJson() => {
        'v': 1,
        'servers': servers.map((s) => s.toJson()).toList(),
        'deleted': deleted,
        'orderU': orderU,
      };

  /// canonical string so two docs compare equal wherever they came from
  String canon() {
    final keys = deleted.keys.toList()..sort();
    return jsonEncode({
      'servers': servers.map((s) => [s.id, s.name, s.user, s.host, s.port, s.color, s.legacy, s.u]).toList(),
      'deleted': [for (final k in keys) [k, deleted[k]]],
      'orderU': orderU,
    });
  }
}

bool sameDoc(SyncDoc a, SyncDoc b) => a.canon() == b.canon();

/// Newest edit of each server wins, deletions beat older edits,
/// order comes from the side that reordered last. Ties go to a.
SyncDoc mergeDocs(SyncDoc a, SyncDoc b) {
  final del = <String, int>{};
  for (final m in [a.deleted, b.deleted]) {
    m.forEach((k, v) {
      if (!del.containsKey(k) || v > del[k]!) del[k] = v;
    });
  }
  final win = <String, Server>{};
  for (final list in [a.servers, b.servers]) {
    for (final s in list) {
      final cur = win[s.id];
      if (cur == null || s.u > cur.u) win[s.id] = s;
    }
  }
  final first = b.orderU > a.orderU ? b : a;
  final second = identical(first, a) ? b : a;
  final out = <Server>[];
  final seen = <String>{};
  for (final list in [first.servers, second.servers]) {
    for (final s in list) {
      if (!seen.add(s.id)) continue;
      final w = win[s.id]!;
      final d = del[s.id];
      if (d != null && d >= w.u) continue;
      out.add(w.copy());
    }
  }
  final cut = DateTime.now().millisecondsSinceEpoch - tombLife.inMilliseconds;
  del.removeWhere((k, v) => v < cut);
  return SyncDoc(servers: out, deleted: del, orderU: max(a.orderU, b.orderU));
}

// ---------------------------------------------------------------- crypto

Uint8List _b64url(String s) {
  var t = s.replaceAll('-', '+').replaceAll('_', '/');
  while (t.length % 4 != 0) {
    t += '=';
  }
  return base64.decode(t);
}

String _b64urlEnc(List<int> b) => base64Url.encode(b).replaceAll('=', '');

final _rnd = Random.secure();
Uint8List randomBytes(int n) => Uint8List.fromList(List.generate(n, (_) => _rnd.nextInt(256)));

GCMBlockCipher _gcm(bool enc, Uint8List key, Uint8List iv) =>
    GCMBlockCipher(AESEngine())..init(enc, AEADParameters(KeyParameter(key), 128, iv, Uint8List(0)));

String seal(String k, SyncDoc doc) {
  final key = _b64url(k);
  final iv = randomBytes(12);
  final ct = _gcm(true, key, iv).process(Uint8List.fromList(utf8.encode(jsonEncode(doc.toJson()))));
  return const JsonEncoder.withIndent(' ').convert({'app': 'ssh-servers', 'v': 1, 'iv': base64.encode(iv), 'ct': base64.encode(ct)});
}

class SyncError implements Exception {
  final String code; // auth | notfound | badkey | net | other
  final String msg;
  SyncError(this.code, [this.msg = '']);
  @override
  String toString() => msg.isEmpty ? code : '$code: $msg';
}

SyncDoc unseal(String k, String content) {
  try {
    final env = jsonDecode(content) as Map<String, dynamic>;
    if (env['app'] != 'ssh-servers') throw SyncError('badkey');
    final plain = _gcm(false, _b64url(k), base64.decode(env['iv'] as String)).process(base64.decode(env['ct'] as String));
    return SyncDoc.fromJson(jsonDecode(utf8.decode(plain)) as Map<String, dynamic>);
  } on SyncError {
    rethrow;
  } catch (_) {
    throw SyncError('badkey');
  }
}

// ---------------------------------------------------------------- sync config + link

class SyncCfg {
  final String gist, key, token;
  SyncCfg(this.gist, this.key, this.token);
  Map<String, String> toJson() => {'g': gist, 'k': key, 't': token};
  static SyncCfg? fromJson(Map<String, dynamic>? j) =>
      j == null || j['g'] == null ? null : SyncCfg('${j['g']}', '${j['k']}', '${j['t']}');
  String link(String base) =>
      '$base#g=${Uri.encodeQueryComponent(gist)}&k=${Uri.encodeQueryComponent(key)}&t=${Uri.encodeQueryComponent(token)}';
}

SyncCfg? parseLink(String s) {
  final i = s.indexOf('#');
  if (i < 0) return null;
  final q = Uri.splitQueryString(s.substring(i + 1));
  final g = q['g'], k = q['k'], t = q['t'];
  if (g == null || k == null || t == null || g.isEmpty || t.isEmpty) return null;
  try {
    if (_b64url(k).length != 32) return null;
  } catch (_) {
    return null;
  }
  return SyncCfg(g, k, t);
}

bool looksLikeToken(String s) => RegExp(r'^[A-Za-z0-9_]{20,255}$').hasMatch(s);

// ---------------------------------------------------------------- GitHub

const _api = 'https://api.github.com';

Future<dynamic> _gh(String method, String path, String token, [Object? body]) async {
  final req = http.Request(method, Uri.parse('$_api$path'))
    ..headers.addAll({
      'Authorization': 'Bearer $token',
      'Accept': 'application/vnd.github+json',
      'User-Agent': 'SSH-Servers-Android',
      if (body != null) 'Content-Type': 'application/json',
    });
  if (body != null) req.body = jsonEncode(body);
  http.StreamedResponse res;
  try {
    res = await req.send().timeout(const Duration(seconds: 25));
  } catch (e) {
    throw SyncError('net', '$e');
  }
  final text = await res.stream.bytesToString();
  if (res.statusCode == 401 || res.statusCode == 403) throw SyncError('auth');
  if (res.statusCode == 404) throw SyncError('notfound');
  if (res.statusCode >= 300) throw SyncError('other', 'GitHub ${res.statusCode}');
  return text.isEmpty ? null : jsonDecode(text);
}

Future<SyncDoc> pull(SyncCfg sc) async {
  final g = await _gh('GET', '/gists/${sc.gist}', sc.token) as Map<String, dynamic>;
  final f = (g['files'] as Map?)?[gistFile] as Map?;
  if (f == null) throw SyncError('notfound');
  var content = '${f['content']}';
  if (f['truncated'] == true) {
    content = (await http.get(Uri.parse('${f['raw_url']}'))).body;
  }
  return unseal(sc.key, content);
}

Future<void> push(SyncCfg sc, SyncDoc doc) => _gh('PATCH', '/gists/${sc.gist}', sc.token, {
      'files': {gistFile: {'content': seal(sc.key, doc)}}
    });

/// pull -> merge -> push if needed
Future<(SyncDoc, bool)> syncOnce(SyncCfg sc, SyncDoc local) async {
  final remote = await pull(sc);
  final merged = mergeDocs(local, remote);
  if (!sameDoc(merged, remote)) await push(sc, merged);
  return (merged, !sameDoc(merged, local));
}

/// Start from a GitHub token: finds an existing sync (returns null cfg + exists) or creates one.
Future<({SyncCfg? cfg, bool exists})> fromToken(String token, SyncDoc doc, {bool replace = false}) async {
  final list = (await _gh('GET', '/gists?per_page=100', token) as List?) ?? [];
  String? found;
  for (final g in list) {
    final files = (g as Map)['files'] as Map?;
    if (files != null && files.containsKey(gistFile)) {
      found = '${g['id']}';
      break;
    }
  }
  if (found != null && !replace) return (cfg: null, exists: true);
  final k = _b64urlEnc(randomBytes(32));
  final content = seal(k, doc);
  String id;
  if (found != null) {
    id = found;
    await _gh('PATCH', '/gists/$id', token, {'files': {gistFile: {'content': content}}});
  } else {
    final g = await _gh('POST', '/gists', token, {
      'description': 'SSH Servers sync (encrypted)',
      'public': false,
      'files': {gistFile: {'content': content}},
    }) as Map<String, dynamic>;
    id = '${g['id']}';
  }
  return (cfg: SyncCfg(id, k, token), exists: false);
}

// ---------------------------------------------------------------- Sign in with GitHub (device flow; no CORS on Android)

class DeviceCode {
  final String deviceCode, userCode, uri;
  final int interval, expiresIn;
  DeviceCode(this.deviceCode, this.userCode, this.uri, this.interval, this.expiresIn);
}

Future<Map<String, dynamic>> _form(String url, Map<String, String> body) async {
  try {
    final r = await http
        .post(Uri.parse(url), headers: {'Accept': 'application/json', 'User-Agent': 'SSH-Servers-Android'}, body: body)
        .timeout(const Duration(seconds: 25));
    return jsonDecode(r.body) as Map<String, dynamic>;
  } catch (e) {
    throw SyncError('net', '$e');
  }
}

Future<DeviceCode> deviceStart() async {
  final j = await _form('https://github.com/login/device/code', {'client_id': githubClientId, 'scope': 'gist'});
  if (j['device_code'] == null) throw SyncError('other', '${j['error_description'] ?? j['error'] ?? 'no device code'}');
  return DeviceCode('${j['device_code']}', '${j['user_code']}', '${j['verification_uri']}',
      (j['interval'] as num?)?.toInt() ?? 5, (j['expires_in'] as num?)?.toInt() ?? 900);
}

/// Polls until the user authorizes. Returns the token, or null when cancelled/expired.
Future<String?> devicePoll(DeviceCode dc, bool Function() alive) async {
  var iv = Duration(seconds: max(5, dc.interval));
  final end = DateTime.now().add(Duration(seconds: dc.expiresIn));
  while (alive() && DateTime.now().isBefore(end)) {
    await Future.delayed(iv);
    if (!alive()) return null;
    Map<String, dynamic> j;
    try {
      j = await _form('https://github.com/login/oauth/access_token', {
        'client_id': githubClientId,
        'device_code': dc.deviceCode,
        'grant_type': 'urn:ietf:params:oauth:grant-type:device_code',
      });
    } catch (_) {
      continue;
    }
    if (j['access_token'] != null) return '${j['access_token']}';
    if (j['error'] == 'slow_down') {
      iv += const Duration(seconds: 5);
      continue;
    }
    if (j['error'] == 'authorization_pending') continue;
    return null;
  }
  return null;
}
