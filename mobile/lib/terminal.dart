import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:wakelock_plus/wakelock_plus.dart';
import 'package:xterm/xterm.dart';

import 'l10n.dart';
import 'main.dart';
import 'ssh.dart';
import 'store.dart';
import 'sync.dart';

class TerminalScreen extends StatefulWidget {
  final Sess sess;
  final bool fresh;
  const TerminalScreen({super.key, required this.sess, this.fresh = false});
  @override
  State<TerminalScreen> createState() => _TerminalScreenState();
}

class _TerminalScreenState extends State<TerminalScreen> implements SessUI {
  Sess get s => widget.sess;
  final _focus = FocusNode();
  final _tc = TerminalController();

  @override
  void initState() {
    super.initState();
    s.addListener(_re);
    if (AppState.I.keepAwake) WakelockPlus.enable();
    if (widget.fresh) WidgetsBinding.instance.addPostFrameCallback((_) => _connect());
  }

  @override
  void dispose() {
    s.removeListener(_re);
    WakelockPlus.disable();
    _focus.dispose();
    super.dispose();
  }

  void _re() {
    if (mounted) setState(() {});
  }

  Server get server => s.server;

  Future<void> _connect({String? password, bool askedPassword = false, bool install = false, bool remember = false}) async {
    s.term.write('\x1b[90m${t('connecting')} ${server.addr}\x1b[0m\r\n');
    try {
      await s.connect(this, password: password);
      _focus.requestFocus();
      if (askedPassword && password != null) {
        if (remember) await AppState.I.savePassword(server.id, password);
        if (install) {
          try {
            await s.installKey();
            toast(t('keyInstalled'));
          } catch (e) {
            toast(t('keyFail', e), err: true);
          }
        }
      }
    } on AuthNeeded {
      if (askedPassword) toast(t('authFail'), err: true);
      AppState.I.setKeyOK(server.id, false);
      final r = await _askPassword();
      if (r == null) {
        s.close();
        if (mounted) Navigator.pop(context);
        return;
      }
      await _connect(password: r.$1, askedPassword: true, install: r.$2, remember: r.$3);
    } on HostKeyRejected {
      s.close();
      if (mounted) Navigator.pop(context);
    } catch (e) {
      s.state = SessState.closed;
      s.error = '$e';
      s.term.write('\r\n\x1b[31m${t('connFail', e)}\x1b[0m\r\n');
      _re();
    }
  }

  @override
  Future<bool> confirmHostKey(Server sv, String fp, bool first) async {
    if (!mounted) return false;
    final ok = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (d) => AlertDialog(
        title: Text(first ? t('hkNewT') : t('hkChgT'), style: TextStyle(color: first ? cText : cWarn)),
        content: Text(first ? t('hkNew', fp) : t('hkChg', fp), style: const TextStyle(height: 1.7)),
        actions: [
          TextButton(onPressed: () => Navigator.pop(d, false), child: Text(t('cancel'))),
          FilledButton(onPressed: () => Navigator.pop(d, true), child: Text(t('accept'))),
        ],
      ),
    );
    return ok == true;
  }

  /// (password, installKey, remember)
  Future<(String, bool, bool)?> _askPassword() async {
    if (!mounted) return null;
    final pw = TextEditingController();
    var install = true, remember = false, hide = true;
    return showDialog<(String, bool, bool)>(
      context: context,
      barrierDismissible: false,
      builder: (d) => StatefulBuilder(
        builder: (d, setS) => AlertDialog(
          title: Text(t('passFor', server.addr), style: const TextStyle(fontSize: 16)),
          content: Column(mainAxisSize: MainAxisSize.min, children: [
            TextField(
              controller: pw,
              autofocus: true,
              obscureText: hide,
              textDirection: TextDirection.ltr,
              decoration: InputDecoration(
                suffixIcon: IconButton(icon: Icon(hide ? Icons.visibility_outlined : Icons.visibility_off_outlined), onPressed: () => setS(() => hide = !hide)),
              ),
              onSubmitted: (_) => Navigator.pop(d, (pw.text, install, remember)),
            ),
            const SizedBox(height: 8),
            CheckboxListTile(
              value: install,
              onChanged: (v) => setS(() => install = v ?? false),
              title: Text(t('installKey'), style: const TextStyle(fontSize: 13.5)),
              contentPadding: EdgeInsets.zero,
              controlAffinity: ListTileControlAffinity.leading,
            ),
            CheckboxListTile(
              value: remember,
              onChanged: (v) => setS(() => remember = v ?? false),
              title: Text(t('remember'), style: const TextStyle(fontSize: 13.5)),
              contentPadding: EdgeInsets.zero,
              controlAffinity: ListTileControlAffinity.leading,
            ),
          ]),
          actions: [
            TextButton(onPressed: () => Navigator.pop(d), child: Text(t('cancel'))),
            FilledButton(onPressed: () => Navigator.pop(d, (pw.text, install, remember)), child: Text(t('login'))),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final st = s.state;
    final dot = st == SessState.connected ? cOk : (st == SessState.closed ? cDanger : cWarn);
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: cBg,
        titleSpacing: 0,
        title: Row(children: [
          Icon(Icons.circle, size: 10, color: dot),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(server.name, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
              Text(server.addr, textDirection: TextDirection.ltr, style: const TextStyle(fontSize: 11.5, color: cMuted, fontFamily: 'monospace')),
            ]),
          ),
        ]),
        actions: [
          if (st == SessState.closed)
            IconButton(tooltip: t('reconnect'), icon: const Icon(Icons.refresh_rounded), onPressed: () => _connect()),
          IconButton(
            tooltip: t('paste'),
            icon: const Icon(Icons.content_paste_rounded),
            onPressed: () async {
              final d = await Clipboard.getData('text/plain');
              if (d?.text != null) s.send(d!.text!);
            },
          ),
          IconButton(
            tooltip: t('disconnect'),
            icon: const Icon(Icons.close_rounded, color: cDanger),
            onPressed: () {
              s.close();
              Navigator.pop(context);
            },
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(children: [
          Expanded(
            child: Directionality(
              textDirection: TextDirection.ltr,
              child: TerminalView(
                s.term,
                controller: _tc,
                focusNode: _focus,
                autofocus: true,
                backgroundOpacity: 1,
                padding: const EdgeInsets.all(4),
                textStyle: TerminalStyle(fontSize: AppState.I.fontSize, fontFamily: 'monospace'),
                theme: TerminalThemes.defaultTheme,
              ),
            ),
          ),
          _keys(),
        ]),
      ),
    );
  }

  Widget _keys() {
    Widget k(String label, VoidCallback onTap) => InkWell(
          onTap: () {
            HapticFeedback.selectionClick();
            onTap();
          },
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            alignment: Alignment.center,
            child: Text(label, style: const TextStyle(fontFamily: 'monospace', fontSize: 14, color: cText)),
          ),
        );
    void key(TerminalKey key) => s.term.keyInput(key);
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Container(
        height: 44,
        decoration: const BoxDecoration(color: cPanel, border: Border(top: BorderSide(color: cLine))),
        child: ListView(scrollDirection: Axis.horizontal, children: [
          k('Esc', () => key(TerminalKey.escape)),
          k('Tab', () => key(TerminalKey.tab)),
          k('^C', () => s.send('\x03')),
          k('^D', () => s.send('\x04')),
          k('^Z', () => s.send('\x1a')),
          k('^L', () => s.send('\x0c')),
          k('←', () => key(TerminalKey.arrowLeft)),
          k('↑', () => key(TerminalKey.arrowUp)),
          k('↓', () => key(TerminalKey.arrowDown)),
          k('→', () => key(TerminalKey.arrowRight)),
          k('|', () => s.send('|')),
          k('/', () => s.send('/')),
          k('-', () => s.send('-')),
          k('~', () => s.send('~')),
          k('Home', () => key(TerminalKey.home)),
          k('End', () => key(TerminalKey.end)),
          k('PgUp', () => key(TerminalKey.pageUp)),
          k('PgDn', () => key(TerminalKey.pageDown)),
        ]),
      ),
    );
  }
}
