// Update check from GitHub releases (tags android-v*). Download opens the APK in the browser.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import 'l10n.dart';
import 'main.dart';

class Update {
  final String latest, url, notes;
  Update(this.latest, this.url, this.notes);
}

String? _current;
Future<String> appVersion() async => _current ??= (await PackageInfo.fromPlatform()).version;

List<int> _parts(String v) => v.split('.').map((p) => int.tryParse(p.replaceAll(RegExp(r'[^0-9]'), '')) ?? 0).toList();

bool newer(String a, String b) {
  final x = _parts(a), y = _parts(b);
  for (var i = 0; i < x.length || i < y.length; i++) {
    final p = i < x.length ? x[i] : 0, q = i < y.length ? y[i] : 0;
    if (p != q) return p > q;
  }
  return false;
}

/// null = up to date; throws on network errors.
Future<Update?> findUpdate() async {
  final r = await http.get(Uri.parse('https://api.github.com/repos/Plus98ir/SSH-Servers/releases?per_page=30'),
      headers: {'Accept': 'application/vnd.github+json', 'User-Agent': 'SSH-Servers-Android'}).timeout(const Duration(seconds: 20));
  if (r.statusCode != 200) throw Exception('GitHub ${r.statusCode}');
  final cur = await appVersion();
  for (final rel in jsonDecode(r.body) as List) {
    final m = rel as Map<String, dynamic>;
    final tag = '${m['tag_name']}';
    if (m['draft'] == true || m['prerelease'] == true || !tag.startsWith('android-v')) continue;
    final v = tag.substring('android-v'.length);
    String? url;
    for (final a in (m['assets'] as List? ?? [])) {
      if ('${a['name']}'.endsWith('.apk')) url = '${a['browser_download_url']}';
    }
    if (url == null || !newer(v, cur)) return null;
    return Update(v, url, '${m['body'] ?? ''}');
  }
  return null;
}

/// manual: shows "you're up to date" / errors too. Auto checks run at most every 12 hours.
Future<void> checkUpdate(BuildContext context, {bool manual = false}) async {
  final p = await SharedPreferences.getInstance();
  if (!manual) {
    final last = p.getInt('updCheck') ?? 0;
    if (DateTime.now().millisecondsSinceEpoch - last < 12 * 3600 * 1000) return;
  }
  Update? u;
  try {
    u = await findUpdate();
    p.setInt('updCheck', DateTime.now().millisecondsSinceEpoch);
  } catch (_) {
    if (manual) toast(t('upErr'), err: true);
    return;
  }
  if (u == null) {
    if (manual) toast(t('upLatest'));
    return;
  }
  final cur = await appVersion();
  if (!context.mounted) return;
  final up = u;
  showDialog(
    context: context,
    builder: (d) => AlertDialog(
      title: Text('${t('upTitle')} · v${up.latest}'),
      content: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(t('upMsg', 'v$cur'), style: const TextStyle(color: cMuted)),
        if (up.notes.trim().isNotEmpty) ...[
          const SizedBox(height: 10),
          Container(
            constraints: const BoxConstraints(maxHeight: 180),
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(color: const Color(0xFF0E1218), borderRadius: BorderRadius.circular(10)),
            child: SingleChildScrollView(child: Text(up.notes, textDirection: TextDirection.ltr, style: const TextStyle(fontSize: 12.5, color: cMuted))),
          ),
        ],
        const SizedBox(height: 10),
        Text(t('upHow'), style: const TextStyle(fontSize: 12.5, color: cDim, height: 1.6)),
      ]),
      actions: [
        TextButton(onPressed: () => Navigator.pop(d), child: Text(t('upLater'))),
        FilledButton(
          onPressed: () {
            Navigator.pop(d);
            launchUrl(Uri.parse(up.url), mode: LaunchMode.externalApplication);
          },
          child: Text(t('upNow')),
        ),
      ],
    ),
  );
}
