import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';

import 'l10n.dart';
import 'main.dart';
import 'store.dart';
import 'sync.dart';

const syncPage = 'https://corenetbase.com/SSH-Servers/app/';

void openSync(BuildContext context) => Navigator.push(context, MaterialPageRoute(builder: (_) => const SyncScreen()));

class SyncScreen extends StatefulWidget {
  const SyncScreen({super.key});
  @override
  State<SyncScreen> createState() => _SyncScreenState();
}

class _SyncScreenState extends State<SyncScreen> {
  final _link = TextEditingController();
  bool _working = false;

  @override
  void initState() {
    super.initState();
    AppState.I.addListener(_re);
  }

  @override
  void dispose() {
    AppState.I.removeListener(_re);
    super.dispose();
  }

  void _re() {
    if (mounted) setState(() {});
  }

  String _ago(int ms) {
    if (ms == 0) return t('never');
    final m = DateTime.now().difference(DateTime.fromMillisecondsSinceEpoch(ms)).inMinutes;
    if (m < 1) return t('justNow');
    if (m < 60) return t('minAgo', faNum(m));
    final d = DateTime.fromMillisecondsSinceEpoch(ms);
    return faNum('${d.year}/${d.month}/${d.day} ${d.hour}:${d.minute.toString().padLeft(2, '0')}');
  }

  Future<void> _useInput(String s, {bool replace = false}) async {
    s = s.trim();
    final sc = parseLink(s);
    if (sc != null) {
      setState(() => _working = true);
      await AppState.I.setSync(sc);
      if (mounted) setState(() => _working = false);
      return;
    }
    if (looksLikeToken(s)) {
      await _withToken(s, replace: replace);
      return;
    }
    toast(t('badLink'), err: true);
  }

  Future<void> _withToken(String token, {bool replace = false}) async {
    setState(() => _working = true);
    try {
      final exists = await AppState.I.useToken(token, replace: replace);
      if (exists && mounted) {
        final r = await showDialog<String>(
          context: context,
          builder: (d) => AlertDialog(
            title: Text(t('sync')),
            content: Text(t('exists'), style: const TextStyle(height: 1.7)),
            actions: [
              TextButton(onPressed: () => Navigator.pop(d), child: Text(t('cancel'))),
              TextButton(onPressed: () => Navigator.pop(d, 'restart'), child: Text(t('restart'), style: const TextStyle(color: cDanger))),
              FilledButton(onPressed: () => Navigator.pop(d, 'scan'), child: Text(t('scanQr'))),
            ],
          ),
        );
        if (r == 'restart') {
          await _withToken(token, replace: true);
        } else if (r == 'scan') {
          await _scan();
        }
      }
    } catch (e) {
      toast(AppState.I.errText(e), err: true);
    } finally {
      if (mounted) setState(() => _working = false);
    }
  }

  Future<void> _scan() async {
    final v = await Navigator.push<String>(context, MaterialPageRoute(builder: (_) => const ScanScreen()));
    if (v != null) await _useInput(v);
  }

  Future<void> _github() async {
    DeviceCode dc;
    setState(() => _working = true);
    try {
      dc = await deviceStart();
    } catch (e) {
      toast(AppState.I.errText(e), err: true);
      if (mounted) setState(() => _working = false);
      return;
    }
    await Clipboard.setData(ClipboardData(text: dc.userCode));
    var alive = true;
    if (!mounted) return;
    // code dialog; polling runs alongside
    final dialog = showDialog(
      context: context,
      barrierDismissible: false,
      builder: (d) => AlertDialog(
        title: Text(t('ghLogin')),
        content: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Text(t('ghCode'), style: const TextStyle(color: cMuted, height: 1.7)),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.symmetric(vertical: 14),
            decoration: BoxDecoration(color: const Color(0xFF0E1218), borderRadius: BorderRadius.circular(12)),
            child: SelectableText(dc.userCode,
                textAlign: TextAlign.center,
                textDirection: TextDirection.ltr,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 28, fontWeight: FontWeight.w700, letterSpacing: 4)),
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: () => launchUrl(Uri.parse(dc.uri), mode: LaunchMode.externalApplication),
            icon: const Icon(Icons.open_in_new_rounded, size: 18),
            label: Text(t('ghOpen')),
          ),
          const SizedBox(height: 12),
          Row(children: [
            const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
            const SizedBox(width: 10),
            Expanded(child: Text(t('ghWait'), style: const TextStyle(color: cMuted, fontSize: 12.5))),
          ]),
        ]),
        actions: [
          TextButton(
            onPressed: () {
              alive = false;
              Navigator.pop(d);
            },
            child: Text(t('cancel')),
          ),
        ],
      ),
    );
    launchUrl(Uri.parse(dc.uri), mode: LaunchMode.externalApplication);
    final token = await devicePoll(dc, () => alive);
    if (alive && mounted) Navigator.of(context, rootNavigator: true).pop();
    await dialog;
    if (!mounted) return;
    setState(() => _working = false);
    if (token == null) {
      if (alive) toast(t('ghCancel'), err: true);
      return;
    }
    await _withToken(token);
  }

  @override
  Widget build(BuildContext context) {
    final st = AppState.I;
    final sc = st.sync;
    return Scaffold(
      appBar: AppBar(backgroundColor: cBg, title: Text(t('syncTitle'), style: const TextStyle(fontSize: 17))),
      body: ListView(padding: const EdgeInsets.fromLTRB(18, 8, 18, 30), children: [
        Text(t('syncIntro'), style: const TextStyle(color: cMuted, height: 1.8, fontSize: 13.5)),
        const SizedBox(height: 18),
        if (_working) const LinearProgressIndicator(minHeight: 2),
        if (sc == null) ..._off() else ..._on(sc),
      ]),
    );
  }

  List<Widget> _off() => [
        FilledButton.icon(
          style: FilledButton.styleFrom(backgroundColor: const Color(0xFFF0F3F6), foregroundColor: const Color(0xFF0D1117)),
          onPressed: _working ? null : _github,
          icon: const Icon(Icons.login_rounded),
          label: Text(t('ghLogin')),
        ),
        const SizedBox(height: 10),
        OutlinedButton.icon(onPressed: _working ? null : _scan, icon: const Icon(Icons.qr_code_scanner_rounded), label: Text(t('scanQr'))),
        const SizedBox(height: 22),
        Row(children: [
          Expanded(
            child: TextField(
              controller: _link,
              textDirection: TextDirection.ltr,
              autocorrect: false,
              decoration: InputDecoration(hintText: t('pasteLink')),
            ),
          ),
          const SizedBox(width: 8),
          FilledButton(onPressed: _working ? null : () => _useInput(_link.text), child: Text(t('connect'))),
        ]),
      ];

  List<Widget> _on(SyncCfg sc) {
    final st = AppState.I;
    final err = st.syncErr;
    final link = sc.link(syncPage);
    return [
      Row(children: [
        Icon(Icons.circle, size: 10, color: err.isEmpty ? cOk : cDanger),
        const SizedBox(width: 8),
        Expanded(child: Text(err.isEmpty ? t('syncOn', _ago(st.lastSync)) : err, style: const TextStyle(color: cMuted, height: 1.6))),
      ]),
      const SizedBox(height: 14),
      FilledButton.icon(
        onPressed: st.busy ? null : () => st.runSync(manual: true),
        icon: const Icon(Icons.sync_rounded),
        label: Text(t('syncNow')),
      ),
      const SizedBox(height: 22),
      Text(t('myLink'), style: const TextStyle(color: cMuted, fontSize: 12.5, height: 1.7)),
      const SizedBox(height: 10),
      Center(
        child: Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14)),
          child: QrImageView(data: link, size: 200, backgroundColor: Colors.white),
        ),
      ),
      const SizedBox(height: 10),
      OutlinedButton.icon(
        onPressed: () async {
          await Clipboard.setData(ClipboardData(text: link));
          toast(t('copied'));
        },
        icon: const Icon(Icons.copy_rounded, size: 18),
        label: Text(t('copyLink')),
      ),
      const SizedBox(height: 26),
      TextButton(
        onPressed: () async {
          final ok = await showDialog<bool>(
            context: context,
            builder: (d) => AlertDialog(
              title: Text(t('syncOff')),
              content: Text(t('syncOffQ')),
              actions: [
                TextButton(onPressed: () => Navigator.pop(d, false), child: Text(t('cancel'))),
                FilledButton(style: FilledButton.styleFrom(backgroundColor: cDanger, foregroundColor: Colors.white), onPressed: () => Navigator.pop(d, true), child: Text(t('syncOff'))),
              ],
            ),
          );
          if (ok == true) await st.stopSync();
        },
        child: Text(t('syncOff'), style: const TextStyle(color: cDanger)),
      ),
    ];
  }
}

class ScanScreen extends StatefulWidget {
  const ScanScreen({super.key});
  @override
  State<ScanScreen> createState() => _ScanScreenState();
}

class _ScanScreenState extends State<ScanScreen> {
  bool _done = false;

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(backgroundColor: cBg, title: Text(t('scanTitle'), style: const TextStyle(fontSize: 17))),
        body: Stack(children: [
          MobileScanner(
            onDetect: (cap) {
              if (_done) return;
              for (final b in cap.barcodes) {
                final v = b.rawValue;
                if (v != null && parseLink(v) != null) {
                  _done = true;
                  Navigator.pop(context, v);
                  return;
                }
              }
            },
          ),
          Positioned(
            left: 16,
            right: 16,
            bottom: 30,
            child: Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(color: Colors.black.withValues(alpha: .6), borderRadius: BorderRadius.circular(12)),
              child: Text(t('scanHint'), textAlign: TextAlign.center, style: const TextStyle(color: Colors.white)),
            ),
          ),
        ]),
      );
}
