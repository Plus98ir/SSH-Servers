// FA / EN strings.
String lang = 'fa';

const _s = <String, List<String>>{
  // key: [fa, en]
  'title': ['سرورهای SSH', 'SSH Servers'],
  'subtitle': ['روی سرور بزن تا وصل بشی', 'Tap a server to connect'],
  'newServer': ['سرور جدید', 'New server'],
  'empty': ['هنوز سروری نداری. از منوی ⋮ «افزودن سرور» رو بزن یا سینک رو روشن کن.', 'No servers yet. Use ⋮ → “Add server”, or turn on sync.'],
  'addServer': ['افزودن سرور', 'Add server'],
  'viewTiles': ['نمایش کاشی', 'Tile view'],
  'viewList': ['نمایش ستونی', 'List view'],
  'gestureOn': ['حالت ژست: بکش برای فلش‌ها', 'Gesture mode: swipe for arrows'],
  'edit': ['ویرایش', 'Edit'],
  'del': ['حذف', 'Delete'],
  'delQ': ['«{n}» حذف بشه؟ از بقیهٔ دستگاه‌ها هم حذف می‌شه.', 'Delete “{n}”? It is removed from your other devices too.'],
  'up': ['جابه‌جایی به قبل', 'Move earlier'],
  'down': ['جابه‌جایی به بعد', 'Move later'],
  'cancel': ['انصراف', 'Cancel'],
  'save': ['ذخیره', 'Save'],
  'add': ['اضافه کن', 'Add'],
  'close': ['بستن', 'Close'],
  'ok': ['باشه', 'OK'],
  'name': ['اسم', 'Name'],
  'namePh': ['مثلاً سرور آلمان', 'e.g. Germany server'],
  'user': ['یوزر', 'User'],
  'host': ['آدرس (IP یا دامنه)', 'Address (IP or domain)'],
  'port': ['پورت', 'Port'],
  'color': ['رنگ', 'Color'],
  'badHost': ['آدرس سرور رو درست بنویس', 'Enter a valid server address'],
  'badUser': ['یوزر نباید فاصله یا @ داشته باشه', 'The user can’t contain spaces or @'],
  // connect
  'connecting': ['در حال وصل شدن…', 'Connecting…'],
  'passFor': ['پسورد {n}', 'Password for {n}'],
  'remember': ['پسورد رو روی این گوشی نگه دار', 'Remember the password on this phone'],
  'installKey': ['ورود بدون رمز (کلید این گوشی روی سرور نصب بشه)', 'Passwordless next time (install this phone’s key on the server)'],
  'login': ['ورود', 'Log in'],
  'authFail': ['یوزر یا پسورد اشتباهه', 'Wrong user or password'],
  'connFail': ['وصل نشد: {n}', 'Couldn’t connect: {n}'],
  'keyInstalled': ['کلید نصب شد؛ دفعهٔ بعد بدون رمز وارد می‌شی', 'Key installed — no password next time'],
  'keyFail': ['نصب کلید نشد: {n}', 'Couldn’t install the key: {n}'],
  'hkNewT': ['اولین اتصال به این سرور', 'First connection to this server'],
  'hkNew': ['اثر انگشت سرور:\n{n}\nاگه سرور خودته، قبول کن.', 'Server fingerprint:\n{n}\nAccept if this is your server.'],
  'hkChgT': ['کلید سرور عوض شده!', 'The server’s key changed!'],
  'hkChg': ['اگه سرور رو از نو نصب کردی عادیه. اثر انگشت جدید:\n{n}\nوگرنه وصل نشو.', 'Normal if you rebuilt the server. New fingerprint:\n{n}\nOtherwise don’t connect.'],
  'accept': ['قبول و اتصال', 'Accept and connect'],
  'disconnect': ['قطع اتصال', 'Disconnect'],
  'closed': ['اتصال بسته شد', 'Connection closed'],
  'reconnect': ['اتصال دوباره', 'Reconnect'],
  'sessions': ['سشن‌های باز', 'Open sessions'],
  'noPass': ['بدون رمز', 'No password'],
  'forgetPass': ['فراموش کردن پسورد ذخیره‌شده', 'Forget saved password'],
  'copied': ['کپی شد', 'Copied'],
  'paste': ['پیست', 'Paste'],
  // sync
  'sync': ['سینک', 'Sync'],
  'syncTitle': ['سینک لیست سرورها', 'Sync the server list'],
  'syncIntro': ['لیست سرورها (اسم، آدرس، پورت، یوزر و رنگ) رمزنگاری‌شده روی GitHub خودت نگه داشته می‌شه و با برنامهٔ ویندوز و گوشی‌های دیگه یکی می‌شه. پسوردها هیچ‌وقت سینک نمی‌شن.',
      'The server list (name, address, port, user and color) is kept encrypted on your own GitHub and shared with the Windows app and your other phones. Passwords are never synced.'],
  'ghLogin': ['ورود با GitHub', 'Sign in with GitHub'],
  'scanQr': ['اسکن QR برنامهٔ کامپیوتر', 'Scan the PC app’s QR'],
  'pasteLink': ['لینک سینک یا توکن', 'Sync link or token'],
  'connect': ['وصل کن', 'Connect'],
  'ghCode': ['این کد رو توی صفحهٔ GitHub وارد کن و Authorize رو بزن (کد کپی شد):', 'Enter this code on the GitHub page and press Authorize (it’s copied):'],
  'ghOpen': ['باز کردن GitHub', 'Open GitHub'],
  'ghWait': ['منتظر تأیید GitHub… حساب نداری؟ توی همون صفحه رایگان بساز.', 'Waiting for GitHub… No account? Create one for free on that page.'],
  'ghCancel': ['ورود با GitHub لغو شد یا وقتش تموم شد', 'GitHub sign-in was cancelled or timed out'],
  'exists': ['روی این حساب GitHub قبلاً سینک ساخته شده. QR برنامهٔ کامپیوتر (یا گوشی دیگه) رو اسکن کن، یا از نو بساز (لیست این گوشی جای قبلی رو می‌گیره).',
      'Sync already exists on this GitHub account. Scan the QR from the PC app (or another phone), or start over (this phone’s list replaces the old one).'],
  'restart': ['از نو بساز', 'Start over'],
  'badLink': ['این نه لینک سینک درسته، نه توکن GitHub', 'That’s neither a sync link nor a GitHub token'],
  'syncOn': ['سینک روشنه · آخرین سینک: {n}', 'Sync is on · last sync: {n}'],
  'syncNow': ['همین الان سینک کن', 'Sync now'],
  'syncOff': ['قطع سینک روی این گوشی', 'Turn sync off on this phone'],
  'syncOffQ': ['سینک روی این گوشی قطع بشه؟ لیست همین‌جا می‌مونه.', 'Turn sync off on this phone? The list stays here.'],
  'synced': ['سینک شد', 'Synced'],
  'got': ['لیست از دستگاه دیگه به‌روز شد', 'List updated from another device'],
  'myLink': ['QR این گوشی برای اضافه کردن دستگاه دیگه (مثل رمزه، به کسی نشون نده):', 'This phone’s QR to add another device (it works like a password — don’t show it around):'],
  'copyLink': ['کپی لینک سینک', 'Copy sync link'],
  'eAuth': ['توکن GitHub قبول نشد (منقضی یا پاک شده). دوباره وارد شو.', 'GitHub didn’t accept the token (expired or revoked). Sign in again.'],
  'eNotFound': ['فایل سینک روی GitHub پیدا نشد', 'The sync file wasn’t found on GitHub'],
  'eKey': ['کلید سینک با فایل نمی‌خونه؛ QR رو دوباره اسکن کن', 'The sync key doesn’t match; scan the QR again'],
  'eNet': ['به GitHub وصل نشد؛ دفعهٔ بعد سینک می‌شه', 'Couldn’t reach GitHub; it will sync next time'],
  'never': ['هنوز نه', 'not yet'],
  'justNow': ['همین الان', 'just now'],
  'minAgo': ['{n} دقیقه پیش', '{n} min ago'],
  'scanTitle': ['QR سینک رو اسکن کن', 'Scan the sync QR'],
  'scanHint': ['توی برنامهٔ ویندوز: دکمهٔ سینک (فلش‌های گرد بالا)', 'In the Windows app: the sync button (round arrows, top)'],
  // settings
  'settings': ['تنظیمات', 'Settings'],
  'language': ['زبان', 'Language'],
  'fontSize': ['اندازهٔ متن ترمینال', 'Terminal text size'],
  'keepAwake': ['صفحه موقع اتصال خاموش نشه', 'Keep the screen on while connected'],
  'myKey': ['کلید عمومی این گوشی', 'This phone’s public key'],
  'copyKey': ['کپی کلید عمومی', 'Copy public key'],
  'guide': ['راهنما', 'Guide'],
  'guideBody': [
    '۱. سرورهات رو با + اضافه کن، یا سینک رو روشن کن تا لیست برنامهٔ ویندوز بیاد.\n'
        '۲. سینک: «ورود با GitHub» ← کد رو توی GitHub بزن ← Authorize. اگه روی کامپیوتر قبلاً روشنش کردی، به جاش QR برنامهٔ ویندوز رو اسکن کن.\n'
        '۳. روی کارت سرور بزن تا ترمینال باز بشه. بار اول پسورد می‌پرسه؛ تیک «ورود بدون رمز» رو بذار تا دفعه‌های بعد پسورد نخواد.\n'
        '۴. نوار پایین ترمینال: Ctrl+C، فلش‌ها، ctrl چسبنده و paste. دکمهٔ ✥ حالت ژست رو روشن می‌کنه: روی ترمینال به چپ/راست بکش تا مکان‌نما جابه‌جا بشه، بالا/پایین برای دستورهای قبلی. دوبار ضربه روی ترمینال = Tab. با دکمهٔ برگشت سشن باز می‌مونه و از بالای لیست برمی‌گردی بهش.\n'
        '۵. سینک خودکاره: موقع باز کردن، بعد از هر تغییر و هر ۱ دقیقه. همیشه آخرین تغییر برنده‌ست.',
    '1. Add your servers with +, or turn on sync to get the Windows app’s list.\n'
        '2. Sync: “Sign in with GitHub” → enter the code on GitHub → Authorize. If sync is already on in the PC app, scan its QR instead.\n'
        '3. Tap a server card to open the terminal. The first time it asks for the password; tick “Passwordless next time” so it won’t ask again.\n'
        '4. Terminal key bar: Ctrl+C, arrows, sticky ctrl and paste. The ✥ button turns on gesture mode: swipe left/right on the terminal to move the cursor, up/down for previous commands. Double tap on the terminal = Tab. Back keeps the session open; return to it from the top of the list.\n'
        '5. Sync is automatic: on open, after every change and every minute. The newest change always wins.'
  ],
  'support': ['حمایت', 'Support'],
};

String t(String k, [Object? n]) {
  final v = _s[k];
  final s = v == null ? k : (lang == 'en' ? v[1] : v[0]);
  return n == null ? s : s.replaceAll('{n}', '$n');
}

bool get rtl => lang == 'fa';

String faNum(Object n) => lang == 'fa' ? '$n'.replaceAllMapped(RegExp(r'\d'), (m) => '۰۱۲۳۴۵۶۷۸۹'[int.parse(m[0]!)]) : '$n';
