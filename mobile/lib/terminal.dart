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
  Offset _pan = Offset.zero;
  bool _gesture = false;
  DateTime _lastTap = DateTime(0);
  Offset _downAt = Offset.zero, _lastTapAt = Offset.zero;

  /// double tap anywhere on the terminal = Tab (raw pointer events, so the terminal's own gestures keep working)
  void _pointerDown(PointerDownEvent e) => _downAt = e.position;
  void _pointerUp(PointerUpEvent e) {
    if ((e.position - _downAt).distance > 18) return; // a drag, not a tap
    final now = DateTime.now();
    if (now.difference(_lastTap).inMilliseconds < 320 && (e.position - _lastTapAt).distance < 40) {
      _tc.clearSelection();
      s.term.keyInput(TerminalKey.tab);
      HapticFeedback.selectionClick();
      _lastTap = DateTime(0);
    } else {
      _lastTap = now;
      _lastTapAt = e.position;
    }
  }

  /// gesture mode: left/right moves the cursor between characters, up/down walks the command history
  void _onPan(DragUpdateDetails d) {
    _pan += d.delta;
    const stepX = 16.0, stepY = 34.0;
    while (_pan.dx.abs() >= stepX || _pan.dy.abs() >= stepY) {
      if (_pan.dx.abs() / stepX >= _pan.dy.abs() / stepY) {
        s.term.keyInput(_pan.dx > 0 ? TerminalKey.arrowRight : TerminalKey.arrowLeft);
        _pan = Offset(_pan.dx - stepX * _pan.dx.sign, 0);
      } else {
        s.term.keyInput(_pan.dy > 0 ? TerminalKey.arrowDown : TerminalKey.arrowUp);
        _pan = Offset(0, _pan.dy - stepY * _pan.dy.sign);
      }
      HapticFeedback.selectionClick();
    }
  }

  @override
  void initState() {
    super.initState();
    s.addListener(_re);
    s.ctrl.addListener(_re);
    if (AppState.I.keepAwake) WakelockPlus.enable();
    if (widget.fresh) WidgetsBinding.instance.addPostFrameCallback((_) => _connect());
  }

  @override
  void dispose() {
    s.removeListener(_re);
    s.ctrl.removeListener(_re);
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
              textInputAction: TextInputAction.done,
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

  // ---------------------------------------------------------------- sessions (+ and grid)

  void _openSession(Sess other, {bool fresh = false}) {
    Navigator.pushReplacement(context, PageRouteBuilder(
      pageBuilder: (_, __, ___) => TerminalScreen(sess: other, fresh: fresh),
      transitionDuration: Duration.zero,
    ));
  }

  void _newSession() {
    final list = AppState.I.doc.servers;
    showModalBottomSheet(
      context: context,
      builder: (c) => SafeArea(
        child: ListView(shrinkWrap: true, padding: const EdgeInsets.fromLTRB(12, 0, 12, 12), children: [
          for (final sv in list)
            ListTile(
              leading: CircleAvatar(radius: 16, backgroundColor: hexColor(sv.color), child: Text(sv.name.isEmpty ? '?' : sv.name.characters.first.toUpperCase(), style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700))),
              title: Text(sv.name),
              subtitle: Text(sv.addr, textDirection: TextDirection.ltr, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: cMuted)),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
              onTap: () {
                Navigator.pop(c);
                final n = Sess(sv);
                Sess.open.add(n);
                Sess.changes.value++;
                _openSession(n, fresh: true);
              },
            ),
        ]),
      ),
    );
  }

  void _sessions() {
    showModalBottomSheet(
      context: context,
      builder: (c) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Text(t('sessions'), style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            GridView.count(
              crossAxisCount: 2,
              shrinkWrap: true,
              mainAxisSpacing: 10,
              crossAxisSpacing: 10,
              childAspectRatio: 2.4,
              children: [
                for (final o in Sess.open)
                  Material(
                    color: identical(o, s) ? cAccent.withValues(alpha: .15) : cCard,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14), side: BorderSide(color: identical(o, s) ? cAccent : cLine)),
                    child: InkWell(
                      borderRadius: BorderRadius.circular(14),
                      onTap: () {
                        Navigator.pop(c);
                        if (!identical(o, s)) _openSession(o);
                      },
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        child: Row(children: [
                          Icon(Icons.circle, size: 9, color: o.state == SessState.connected ? cOk : (o.state == SessState.closed ? cDanger : cWarn)),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.start, children: [
                              Text(o.server.name, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w700)),
                              Text(o.server.host, maxLines: 1, textDirection: TextDirection.ltr, style: const TextStyle(fontSize: 11, color: cMuted, fontFamily: 'monospace')),
                            ]),
                          ),
                        ]),
                      ),
                    ),
                  ),
              ],
            ),
          ]),
        ),
      ),
    );
  }

  // ---------------------------------------------------------------- UI

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0E0E0E),
      resizeToAvoidBottomInset: true,
      body: SafeArea(
        child: Column(children: [
          Expanded(
            child: Directionality(
              textDirection: TextDirection.ltr,
              child: Listener(
                onPointerDown: _pointerDown,
                onPointerUp: _pointerUp,
                child: Stack(children: [
                Positioned.fill(
                  child: TerminalView(
                    s.term,
                    controller: _tc,
                    focusNode: _focus,
                    autofocus: true,
                    // no suggestions / composing: one Enter press is enough on Samsung & Gboard
                    keyboardType: TextInputType.visiblePassword,
                    backgroundOpacity: 0,
                    padding: const EdgeInsets.fromLTRB(6, 4, 6, 4),
                    textStyle: TerminalStyle(fontSize: AppState.I.fontSize, fontFamily: 'monospace'),
                    theme: _theme,
                  ),
                ),
                if (_gesture)
                  Positioned.fill(
                    child: GestureDetector(
                      behavior: HitTestBehavior.opaque,
                      onPanStart: (_) => _pan = Offset.zero,
                      onPanUpdate: _onPan,
                      onTap: () => _focus.requestFocus(),
                    ),
                  ),
                if (_gesture)
                  Positioned(
                    top: 8,
                    right: 8,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      decoration: BoxDecoration(color: cAccent.withValues(alpha: .18), borderRadius: BorderRadius.circular(10), border: Border.all(color: cAccent.withValues(alpha: .5))),
                      child: Row(mainAxisSize: MainAxisSize.min, children: [
                        const Icon(Icons.swipe_rounded, size: 16, color: cAccent),
                        const SizedBox(width: 6),
                        Text(t('gestureOn'), style: const TextStyle(fontSize: 12, color: cAccent, fontFamily: 'Vazirmatn')),
                      ]),
                    ),
                  ),
                ]),
              ),
            ),
          ),
          _bar(),
          _keys(),
        ]),
      ),
    );
  }

  static const _theme = TerminalTheme(
    cursor: Color(0xFFFF9F1C),
    selection: Color(0x5522C38E),
    foreground: Color(0xFFE8E8E8),
    background: Color(0xFF0E0E0E),
    black: Color(0xFF1C1C1C),
    red: Color(0xFFFF5B6E),
    green: Color(0xFF2FD27A),
    yellow: Color(0xFFFFC857),
    blue: Color(0xFF4DA3FF),
    magenta: Color(0xFFC77DFF),
    cyan: Color(0xFF22D3C5),
    white: Color(0xFFE8E8E8),
    brightBlack: Color(0xFF6B7287),
    brightRed: Color(0xFFFF7A8A),
    brightGreen: Color(0xFF5EE59A),
    brightYellow: Color(0xFFFFD98A),
    brightBlue: Color(0xFF7CBCFF),
    brightMagenta: Color(0xFFD9A3FF),
    brightCyan: Color(0xFF5BE7DB),
    brightWhite: Color(0xFFFFFFFF),
    searchHitBackground: Color(0xFFFFC857),
    searchHitBackgroundCurrent: Color(0xFFFF9F1C),
    searchHitForeground: Color(0xFF000000),
  );

  Widget _round(Widget child, VoidCallback onTap, {double w = 46}) => Material(
        color: const Color(0xFF22262E),
        borderRadius: BorderRadius.circular(16),
        child: InkWell(borderRadius: BorderRadius.circular(16), onTap: onTap, child: SizedBox(width: w, height: 46, child: Center(child: child))),
      );

  Widget _bar() {
    final st = s.state;
    final dot = st == SessState.connected ? cOk : (st == SessState.closed ? cDanger : cWarn);
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 6, 12, 6),
        child: Row(children: [
          _round(const Icon(Icons.chevron_left_rounded, size: 28, color: cText), () => Navigator.pop(context)),
          const SizedBox(width: 10),
          Expanded(
            child: Material(
              color: const Color(0xFF22262E),
              borderRadius: BorderRadius.circular(16),
              child: InkWell(
                borderRadius: BorderRadius.circular(16),
                onTap: st == SessState.closed ? () => _connect() : null,
                child: SizedBox(
                  height: 46,
                  child: Row(children: [
                    const SizedBox(width: 12),
                    st == SessState.connecting
                        ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2, color: cWarn))
                        : Icon(st == SessState.closed ? Icons.refresh_rounded : Icons.circle, size: st == SessState.closed ? 18 : 10, color: dot),
                    const SizedBox(width: 10),
                    Container(width: 10, height: 10, decoration: BoxDecoration(color: hexColor(server.color), borderRadius: BorderRadius.circular(3))),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(st == SessState.closed ? t('reconnect') : server.name,
                          maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 16, color: cText, fontWeight: FontWeight.w500)),
                    ),
                    IconButton(
                      tooltip: t('disconnect'),
                      icon: const Icon(Icons.close_rounded, color: cMuted),
                      onPressed: () {
                        s.close();
                        Navigator.pop(context);
                      },
                    ),
                  ]),
                ),
              ),
            ),
          ),
          const SizedBox(width: 10),
          _round(const Icon(Icons.add_rounded, size: 26, color: cText), _newSession),
          const SizedBox(width: 10),
          _round(const Icon(Icons.grid_view_rounded, size: 22, color: cText), _sessions),
        ]),
      ),
    );
  }

  Widget _keys() {
    Widget k(Widget child, VoidCallback onTap, {bool on = false}) => InkWell(
          onTap: () {
            HapticFeedback.selectionClick();
            onTap();
          },
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 13),
            alignment: Alignment.center,
            decoration: on ? BoxDecoration(color: cAccent.withValues(alpha: .22), borderRadius: BorderRadius.circular(10)) : null,
            child: child,
          ),
        );
    Widget txt(String s, {Color c = cText}) => Text(s, style: TextStyle(fontSize: 17, color: c, fontWeight: FontWeight.w500));
    Widget ic(IconData i) => Icon(i, size: 24, color: cText);
    void key(TerminalKey key) => s.term.keyInput(key);

    // gesture mode toggle: while on, swiping over the terminal sends arrow keys
    final pad = k(Icon(Icons.open_with_rounded, size: 24, color: _gesture ? cAccent : cText), () => setState(() => _gesture = !_gesture), on: _gesture);

    return Directionality(
      textDirection: TextDirection.ltr,
      child: Container(
        height: 50,
        color: const Color(0xFF16181D),
        child: ListView(scrollDirection: Axis.horizontal, padding: const EdgeInsets.symmetric(horizontal: 4), children: [
          k(txt('^C'), () => s.send('\x03')),
          k(txt('^X'), () => s.send('\x18')),
          k(ic(Icons.keyboard_return_rounded), () => s.send('\r')),
          pad,
          k(ic(Icons.arrow_back_rounded), () => key(TerminalKey.arrowLeft)),
          k(ic(Icons.arrow_forward_rounded), () => key(TerminalKey.arrowRight)),
          k(ic(Icons.arrow_upward_rounded), () => key(TerminalKey.arrowUp)),
          k(ic(Icons.arrow_downward_rounded), () => key(TerminalKey.arrowDown)),
          k(txt('tab'), () => key(TerminalKey.tab)),
          k(txt('esc'), () => key(TerminalKey.escape)),
          k(txt('ctrl', c: s.ctrl.value ? cAccent : cText), () => s.ctrl.value = !s.ctrl.value, on: s.ctrl.value),
          k(txt('paste'), () async {
            final d = await Clipboard.getData('text/plain');
            if (d?.text != null) s.send(d!.text!);
          }),
          k(txt('/'), () => s.send('/')),
          k(txt('|'), () => s.send('|')),
          k(txt('-'), () => s.send('-')),
          k(txt('~'), () => s.send('~')),
        ]),
      ),
    );
  }
}
