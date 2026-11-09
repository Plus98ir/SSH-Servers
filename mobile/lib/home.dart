import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'l10n.dart';
import 'main.dart';
import 'settings.dart';
import 'ssh.dart';
import 'store.dart';
import 'sync.dart';
import 'sync_screen.dart';
import 'terminal.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
    Sess.changes.addListener(_re);
    AppState.I.addListener(_re);
  }

  @override
  void dispose() {
    Sess.changes.removeListener(_re);
    AppState.I.removeListener(_re);
    super.dispose();
  }

  void _re() {
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final st = AppState.I;
    final list = st.doc.servers;
    return Scaffold(
      body: SafeArea(
        child: RefreshIndicator(
          color: cAccent,
          onRefresh: () => st.sync == null ? Future.value() : st.runSync(manual: true),
          child: CustomScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: [
              SliverToBoxAdapter(child: _header(context)),
              if (Sess.open.isNotEmpty) SliverToBoxAdapter(child: _sessions(context)),
              if (st.sync == null)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
                    child: _syncBanner(context),
                  ),
                ),
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                sliver: SliverGrid(
                  gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(maxCrossAxisExtent: 240, mainAxisSpacing: 10, crossAxisSpacing: 10, childAspectRatio: 1.05),
                  delegate: SliverChildBuilderDelegate(
                    (c, i) => i < list.length ? _card(c, list[i]) : _addCard(c),
                    childCount: list.length + 1,
                  ),
                ),
              ),
              if (list.isEmpty)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 24),
                    child: Text(t('empty'), textAlign: TextAlign.center, style: const TextStyle(color: cDim, fontSize: 13)),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _header(BuildContext context) {
    final st = AppState.I;
    final dot = st.sync == null ? cDim : (st.syncErr.isNotEmpty ? cDanger : cOk);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
      child: Row(children: [
        Container(
          width: 42,
          height: 42,
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            gradient: const LinearGradient(colors: [cAccent, cAccent2], begin: Alignment.topLeft, end: Alignment.bottomRight),
          ),
          child: const Icon(Icons.terminal_rounded, color: Colors.white),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(t('title'), style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
            Text(t('subtitle'), style: const TextStyle(fontSize: 12.5, color: cMuted), overflow: TextOverflow.ellipsis),
          ]),
        ),
        _iconBtn(
          Stack(clipBehavior: Clip.none, children: [
            st.busy
                ? const SizedBox(width: 19, height: 19, child: CircularProgressIndicator(strokeWidth: 2, color: cText))
                : const Icon(Icons.sync_rounded, size: 20),
            Positioned(top: -3, right: -3, child: Container(width: 8, height: 8, decoration: BoxDecoration(color: dot, shape: BoxShape.circle))),
          ]),
          () => openSync(context),
        ),
        const SizedBox(width: 8),
        _iconBtn(const Icon(Icons.settings_outlined, size: 20), () => Navigator.push(context, MaterialPageRoute(builder: (_) => const SettingsScreen()))),
      ]),
    );
  }

  Widget _iconBtn(Widget child, VoidCallback onTap) => Material(
        color: cPanel,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: const BorderSide(color: cLine)),
        child: InkWell(borderRadius: BorderRadius.circular(12), onTap: onTap, child: SizedBox(width: 42, height: 42, child: Center(child: child))),
      );

  Widget _syncBanner(BuildContext context) => Material(
        color: cCard,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16), side: const BorderSide(color: cLine)),
        child: InkWell(
          borderRadius: BorderRadius.circular(16),
          onTap: () => openSync(context),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Row(children: [
              const Icon(Icons.cloud_sync_outlined, color: cAccent),
              const SizedBox(width: 12),
              Expanded(child: Text(t('syncTitle'), style: const TextStyle(fontWeight: FontWeight.w500))),
              Text(t('ghLogin'), style: const TextStyle(color: cAccent, fontSize: 13)),
            ]),
          ),
        ),
      );

  Widget _sessions(BuildContext context) => SizedBox(
        height: 46,
        child: ListView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 10),
          children: [
            for (final s in Sess.open)
              Padding(
                padding: const EdgeInsetsDirectional.only(end: 8),
                child: ActionChip(
                  avatar: Icon(Icons.circle, size: 10, color: s.state == SessState.connected ? cOk : (s.state == SessState.closed ? cDanger : cWarn)),
                  label: Text(s.server.name),
                  backgroundColor: cPanel,
                  side: const BorderSide(color: cLine),
                  onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => TerminalScreen(sess: s))),
                ),
              ),
          ],
        ),
      );

  Widget _card(BuildContext context, Server s) {
    final c = hexColor(s.color);
    final hsl = HSLColor.fromColor(c);
    final keyOk = AppState.I.keyOK[s.id] == true;
    return Material(
      color: cCard,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18), side: const BorderSide(color: cLine)),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: () => openTerminal(context, s),
        onLongPress: () => _menu(context, s),
        child: Stack(children: [
          PositionedDirectional(
            top: 2,
            end: 2,
            child: IconButton(icon: const Icon(Icons.more_horiz, color: cDim, size: 20), onPressed: () => _menu(context, s)),
          ),
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: Column(mainAxisSize: MainAxisSize.min, children: [
                Container(
                  width: 52,
                  height: 52,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(15),
                    gradient: LinearGradient(
                      colors: [hsl.withLightness((hsl.lightness + .12).clamp(0, 1)).toColor(), hsl.withLightness((hsl.lightness - .08).clamp(0, 1)).toColor()],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                  ),
                  child: Center(
                    child: Text(s.name.isEmpty ? '?' : s.name.characters.first.toUpperCase(),
                        style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.w700)),
                  ),
                ),
                const SizedBox(height: 10),
                Text(s.name, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 15)),
                const SizedBox(height: 2),
                Text(s.addr,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    textDirection: TextDirection.ltr,
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 11.5, color: cMuted)),
                if (keyOk) ...[
                  const SizedBox(height: 6),
                  Row(mainAxisSize: MainAxisSize.min, children: [
                    const Icon(Icons.key_rounded, size: 13, color: cAccent),
                    const SizedBox(width: 4),
                    Text(t('noPass'), style: const TextStyle(fontSize: 11, color: cAccent)),
                  ]),
                ],
              ]),
            ),
          ),
        ]),
      ),
    );
  }

  Widget _addCard(BuildContext context) => Material(
        color: Colors.transparent,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18), side: const BorderSide(color: cLineH)),
        child: InkWell(
          borderRadius: BorderRadius.circular(18),
          onTap: () => editServer(context, null),
          child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(color: Colors.white.withValues(alpha: .05), borderRadius: BorderRadius.circular(14)),
              child: const Icon(Icons.add, color: cMuted),
            ),
            const SizedBox(height: 8),
            Text(t('newServer'), style: const TextStyle(color: cMuted)),
          ]),
        ),
      );

  void _menu(BuildContext context, Server s) {
    final st = AppState.I;
    showModalBottomSheet(
      context: context,
      builder: (c) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
          child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: Text(s.name, style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700)),
            ),
            _item(Icons.terminal_rounded, t('connect'), () {
              Navigator.pop(c);
              openTerminal(context, s);
            }),
            _item(Icons.edit_outlined, t('edit'), () {
              Navigator.pop(c);
              editServer(context, s);
            }),
            _item(Icons.arrow_upward_rounded, t('up'), () {
              Navigator.pop(c);
              st.move(s.id, -1);
            }),
            _item(Icons.arrow_downward_rounded, t('down'), () {
              Navigator.pop(c);
              st.move(s.id, 1);
            }),
            _item(Icons.password_rounded, t('forgetPass'), () {
              Navigator.pop(c);
              st.forgetPassword(s.id);
              toast(t('ok'));
            }),
            _item(Icons.delete_outline, t('del'), () async {
              Navigator.pop(c);
              final ok = await showDialog<bool>(
                context: context,
                builder: (d) => AlertDialog(
                  title: Text(t('del')),
                  content: Text(t('delQ', s.name)),
                  actions: [
                    TextButton(onPressed: () => Navigator.pop(d, false), child: Text(t('cancel'))),
                    FilledButton(style: FilledButton.styleFrom(backgroundColor: cDanger, foregroundColor: Colors.white), onPressed: () => Navigator.pop(d, true), child: Text(t('del'))),
                  ],
                ),
              );
              if (ok == true) st.deleteServer(s.id);
            }, danger: true),
          ]),
        ),
      ),
    );
  }

  Widget _item(IconData i, String label, VoidCallback onTap, {bool danger = false}) => ListTile(
        leading: Icon(i, color: danger ? cDanger : cMuted),
        title: Text(label, style: TextStyle(color: danger ? cDanger : cText)),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        onTap: onTap,
      );
}

void openTerminal(BuildContext context, Server s) {
  final sess = Sess(s);
  Sess.open.add(sess);
  Sess.changes.value++;
  Navigator.push(context, MaterialPageRoute(builder: (_) => TerminalScreen(sess: sess, fresh: true)));
}

Future<void> editServer(BuildContext context, Server? s) async {
  final st = AppState.I;
  final isNew = s == null;
  final cur = s?.copy() ??
      Server(id: '', name: '', user: 'root', host: '', color: colors[st.doc.servers.length % colors.length]);
  final name = TextEditingController(text: cur.name),
      user = TextEditingController(text: cur.user),
      host = TextEditingController(text: cur.host),
      port = TextEditingController(text: '${cur.port}');
  var color = cur.color;
  await showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (c) => StatefulBuilder(
      builder: (c, setS) => Padding(
        padding: EdgeInsets.fromLTRB(18, 0, 18, MediaQuery.of(c).viewInsets.bottom + 18),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Text(isNew ? t('newServer') : t('edit'), style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700)),
          const SizedBox(height: 14),
          TextField(controller: name, maxLength: 32, decoration: InputDecoration(labelText: t('name'), hintText: t('namePh'), counterText: '')),
          const SizedBox(height: 10),
          TextField(
            controller: host,
            autofocus: isNew,
            keyboardType: TextInputType.url,
            autocorrect: false,
            textDirection: TextDirection.ltr,
            decoration: InputDecoration(labelText: t('host'), hintText: '1.2.3.4'),
          ),
          const SizedBox(height: 10),
          Row(children: [
            Expanded(
              flex: 3,
              child: TextField(controller: user, autocorrect: false, textDirection: TextDirection.ltr, decoration: InputDecoration(labelText: t('user'))),
            ),
            const SizedBox(width: 8),
            Expanded(
              flex: 2,
              child: TextField(
                controller: port,
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                textDirection: TextDirection.ltr,
                decoration: InputDecoration(labelText: t('port')),
              ),
            ),
          ]),
          const SizedBox(height: 14),
          Text(t('color'), style: const TextStyle(color: cMuted, fontSize: 12.5)),
          const SizedBox(height: 8),
          Wrap(spacing: 9, runSpacing: 9, children: [
            for (final col in colors)
              GestureDetector(
                onTap: () => setS(() => color = col),
                child: Container(
                  width: 34,
                  height: 34,
                  decoration: BoxDecoration(
                    color: hexColor(col),
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(color: col == color ? Colors.white : Colors.transparent, width: 2),
                  ),
                ),
              ),
          ]),
          const SizedBox(height: 18),
          Row(children: [
            Expanded(
              child: FilledButton(
                onPressed: () {
                  final h = host.text.trim(), u = user.text.trim();
                  if (h.isEmpty || RegExp(r'''[\s@;"']''').hasMatch(h)) {
                    toast(t('badHost'), err: true);
                    return;
                  }
                  if (RegExp(r'''[\s@;"']''').hasMatch(u)) {
                    toast(t('badUser'), err: true);
                    return;
                  }
                  var p = int.tryParse(port.text) ?? 22;
                  if (p <= 0 || p > 65535) p = 22;
                  final out = Server(
                    id: cur.id.isEmpty ? _newId() : cur.id,
                    name: name.text.trim().isEmpty ? h : name.text.trim(),
                    user: u,
                    host: h,
                    port: p,
                    color: color,
                    legacy: !isNew && cur.host == h && cur.user == u && cur.port == p && cur.legacy,
                  );
                  Navigator.pop(c);
                  st.saveServer(out);
                },
                child: Text(isNew ? t('add') : t('save')),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(child: OutlinedButton(onPressed: () => Navigator.pop(c), child: Text(t('cancel')))),
          ]),
        ]),
      ),
    ),
  );
}

String _newId() {
  const abc = '0123456789abcdefghijklmnopqrstuvwxyz';
  final r = randomBytes(12);
  return String.fromCharCodes(r.map((b) => abc.codeUnitAt(b % 36)));
}
