import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';

import 'l10n.dart';
import 'main.dart';
import 'ssh.dart';
import 'store.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});
  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  String _pub = '';

  @override
  void initState() {
    super.initState();
    DeviceKey.get().then((k) {
      if (mounted) setState(() => _pub = k.authorizedLine);
    });
  }

  Widget _sec(String s) => Padding(
        padding: const EdgeInsets.fromLTRB(4, 18, 4, 8),
        child: Text(s, style: const TextStyle(color: cMuted, fontSize: 12.5)),
      );

  @override
  Widget build(BuildContext context) {
    final st = AppState.I;
    return Scaffold(
      appBar: AppBar(backgroundColor: cBg, title: Text(t('settings'), style: const TextStyle(fontSize: 17))),
      body: ListView(padding: const EdgeInsets.fromLTRB(16, 0, 16, 30), children: [
        _sec(t('language')),
        SegmentedButton<String>(
          segments: const [ButtonSegment(value: 'fa', label: Text('فارسی')), ButtonSegment(value: 'en', label: Text('English'))],
          selected: {lang},
          onSelectionChanged: (v) {
            st.setLang(v.first);
            setState(() {});
          },
        ),
        _sec('${t('fontSize')}: ${faNum(st.fontSize.round())}'),
        Slider(
          value: st.fontSize,
          min: 9,
          max: 20,
          divisions: 11,
          onChanged: (v) {
            st.setFont(v);
            setState(() {});
          },
        ),
        SwitchListTile(
          value: st.keepAwake,
          onChanged: (v) {
            st.setKeepAwake(v);
            setState(() {});
          },
          title: Text(t('keepAwake')),
          contentPadding: EdgeInsets.zero,
        ),
        _sec(t('myKey')),
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(color: const Color(0xFF0E1218), borderRadius: BorderRadius.circular(12)),
          child: SelectableText(_pub, textDirection: TextDirection.ltr, style: const TextStyle(fontFamily: 'monospace', fontSize: 11.5, color: cMuted)),
        ),
        const SizedBox(height: 8),
        OutlinedButton.icon(
          onPressed: () async {
            await Clipboard.setData(ClipboardData(text: _pub));
            toast(t('copied'));
          },
          icon: const Icon(Icons.copy_rounded, size: 18),
          label: Text(t('copyKey')),
        ),
        _sec(t('guide')),
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(color: cCard, borderRadius: BorderRadius.circular(14), border: Border.all(color: cLine)),
          child: Text(t('guideBody'), style: const TextStyle(color: cMuted, height: 1.9, fontSize: 13.5)),
        ),
        const SizedBox(height: 18),
        Row(children: [
          Expanded(
            child: OutlinedButton.icon(
              onPressed: () => launchUrl(Uri.parse('https://corenetbase.com/#support'), mode: LaunchMode.externalApplication),
              icon: const Text('☕'),
              label: Text(t('support')),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: OutlinedButton.icon(
              onPressed: () => launchUrl(Uri.parse('https://github.com/Plus98ir/SSH-Servers'), mode: LaunchMode.externalApplication),
              icon: const Icon(Icons.code_rounded, size: 18),
              label: const Text('GitHub'),
            ),
          ),
        ]),
      ]),
    );
  }
}
