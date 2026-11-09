import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'home.dart';
import 'l10n.dart';
import 'store.dart';

const cBg = Color(0xFF0C0F13);
const cPanel = Color(0xFF141820);
const cCard = Color(0xFF191E27);
const cLine = Color(0xFF262C38);
const cLineH = Color(0xFF384154);
const cText = Color(0xFFECEEF4);
const cMuted = Color(0xFF8A91A6);
const cDim = Color(0xFF5D6479);
const cAccent = Color(0xFF22C38E);
const cAccent2 = Color(0xFF1A9BD7);
const cOk = Color(0xFF2FD27A);
const cDanger = Color(0xFFFF5B6E);
const cWarn = Color(0xFFFFB648);

Color hexColor(String h) {
  final v = int.tryParse(h.replaceFirst('#', ''), radix: 16);
  return v == null ? const Color(0xFF495057) : Color(0xFF000000 | v);
}

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
    statusBarColor: Colors.transparent,
    systemNavigationBarColor: cBg,
    statusBarIconBrightness: Brightness.light,
  ));
  await AppState.I.load();
  runApp(const App());
}

final navKey = GlobalKey<NavigatorState>();
final messengerKey = GlobalKey<ScaffoldMessengerState>();

void toast(String m, {bool err = false}) {
  messengerKey.currentState
    ?..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(
      content: Text(m, style: const TextStyle(fontFamily: 'Vazirmatn', color: cText)),
      backgroundColor: const Color(0xFF1E2430),
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: BorderSide(color: err ? cDanger.withValues(alpha: .6) : cLineH)),
      duration: Duration(seconds: err ? 5 : 3),
    ));
}

class App extends StatefulWidget {
  const App({super.key});
  @override
  State<App> createState() => _AppState();
}

class _AppState extends State<App> with WidgetsBindingObserver {
  StreamSubscription? _sub;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    AppState.I.addListener(_re);
    _sub = AppState.I.messages.stream.listen((m) => toast(m.$1, err: m.$2));
  }

  void _re() => setState(() {});

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    AppState.I.removeListener(_re);
    _sub?.cancel();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState s) => AppState.I.setForeground(s == AppLifecycleState.resumed);

  @override
  Widget build(BuildContext context) {
    final base = ThemeData.dark(useMaterial3: true);
    return MaterialApp(
      title: 'SSH Servers',
      debugShowCheckedModeBanner: false,
      navigatorKey: navKey,
      scaffoldMessengerKey: messengerKey,
      locale: Locale(lang),
      supportedLocales: const [Locale('fa'), Locale('en')],
      localizationsDelegates: const [GlobalMaterialLocalizations.delegate, GlobalWidgetsLocalizations.delegate, GlobalCupertinoLocalizations.delegate],
      theme: base.copyWith(
        scaffoldBackgroundColor: cBg,
        colorScheme: base.colorScheme.copyWith(primary: cAccent, secondary: cAccent2, surface: cPanel, error: cDanger),
        textTheme: base.textTheme.apply(fontFamily: 'Vazirmatn', bodyColor: cText, displayColor: cText),
        dialogTheme: const DialogThemeData(backgroundColor: Color(0xFF161A23)),
        bottomSheetTheme: const BottomSheetThemeData(backgroundColor: Color(0xFF161A23), showDragHandle: true),
        inputDecorationTheme: InputDecorationTheme(
          filled: true,
          fillColor: const Color(0xFF0E1218),
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: const BorderSide(color: cLineH)),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: const BorderSide(color: cLineH)),
          focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: const BorderSide(color: cAccent)),
          labelStyle: const TextStyle(color: cMuted),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        ),
        filledButtonTheme: FilledButtonThemeData(
          style: FilledButton.styleFrom(
            backgroundColor: cAccent,
            foregroundColor: const Color(0xFF04140E),
            textStyle: const TextStyle(fontFamily: 'Vazirmatn', fontWeight: FontWeight.w700),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
          ),
        ),
        outlinedButtonTheme: OutlinedButtonThemeData(
          style: OutlinedButton.styleFrom(
            foregroundColor: cText,
            side: const BorderSide(color: cLineH),
            textStyle: const TextStyle(fontFamily: 'Vazirmatn', fontWeight: FontWeight.w500),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          ),
        ),
      ),
      builder: (context, child) => Directionality(textDirection: rtl ? TextDirection.rtl : TextDirection.ltr, child: child!),
      home: const HomeScreen(),
    );
  }
}
