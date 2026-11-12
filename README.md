<h1 align="center">SSH Servers</h1>

<p align="center"><a href="https://hits.sh/github.com/Plus98ir/SSH-Servers/"><img src="https://hits.sh/github.com/Plus98ir/SSH-Servers.svg?style=for-the-badge&label=views&color=2563eb" alt="Views"></a> <a href="https://corenetbase.com/#support"><img src="https://img.shields.io/badge/support-buy%20me%20a%20coffee-e11d2a?style=for-the-badge&logo=buymeacoffee&logoColor=white" alt="Support"></a></p>

<details>
<summary><b>☕ Support · حمایت</b> — USDT (TRC20) / TRX</summary>
<br>
<p align="center"><img src="https://raw.githubusercontent.com/Plus98ir/.github/main/tron-qr.png" width="200" alt="TRON address QR"></p>

```
TA53aFyGNfSAcrCaJi9QmA9x5W47nPJcf7
```

> [!WARNING]
> **TRON network only.** Sending from ERC20 / BEP20 will lose the funds.<br>
> **فقط شبکه‌ی TRON.** ارسال از ERC20 یا BEP20 پول را از بین می‌برد.

</details>

<div align="center">

[🇬🇧 English](#english) | [🇮🇷 فارسی](#persian)

</div>

---

<a name="english"></a>

## 🇬🇧 English

<div align="center">

One click into your SSH servers on Windows. The same list is on your phone and stays in sync.

<img src="SSH-Servers.png" alt="SSH Servers" width="860">

</div>

## Download

- **Windows:** [`SSH Servers.exe`](https://github.com/Plus98ir/SSH-Servers/releases/latest/download/SSH.Servers.exe). It's one file with nothing to install. It needs the WebView2 Runtime, which Windows 10/11 already has.
- **Android:** [`SSH-Servers.apk`](https://github.com/Plus98ir/SSH-Servers/releases/download/android-v1.0.0/SSH-Servers.apk). A real SSH client with a built-in terminal, synced with the Windows app.
- **Website:** https://corenetbase.com/SSH-Servers/
- **Web version (any phone):** https://corenetbase.com/SSH-Servers/app/. Open it and use **Add to Home screen** to install it like an app.

## What it does

**On Windows**
- Each server is a card. Click one (or press `1`–`9`) and `ssh` opens straight in Windows Terminal, in a new tab or its own window.
- **Multiple servers:** open several servers side by side (split panes) or in tabs, or run one command on all of them and see each result.
- **Files (SFTP):** a built-in file manager for every server. Browse folders, upload by dragging files or folders from Explorer, download to your Downloads folder, rename, delete, change permissions, and edit config files right in the app. Transfers show progress and can be cancelled.
- **Passwordless login:** enter the password once and the app puts your SSH key on the server. The password is never saved.
- **Rebuilt server?** When a server's host key changes, the app shows the new fingerprint and fixes `known_hosts` in one click. If the key login broke too, it asks for the password once and reinstalls the key.
- Live ping on every card, old routers and modems (`ssh-rsa`) supported, FA / EN.

**On the phone**
- The same cards. Tap one and it opens in your SSH app ([Termius](https://termius.com), [ConnectBot](https://connectbot.org) or any app that handles `ssh://` links). If no app picks the link up, the `ssh` command is copied for you.
- You can add, edit, delete and reorder servers. Changes reach your PC too.
- Works offline with the last list it saw.

<p align="center">
<img src="screenshots/pc-sync.png" width="520" alt="Sync on the PC">&nbsp;
<img src="screenshots/phone.png" width="250" alt="Phone app">
</p>

## Sync

The server list (name, address, port, user and color) is stored in a **secret Gist on your own GitHub account**, encrypted with AES-256-GCM. There's no server of ours in between, and every user keeps their own data on their own GitHub. Passwords and SSH keys never leave the PC.

1. On the PC press the **sync button** (top bar), then **Sign in with GitHub**. GitHub opens in the browser: enter the short code shown in the app and press Authorize. That's it, no token to create. A personal token with the `gist` scope still works too.
2. A QR code appears. Scan it with your phone camera. The phone app opens and is connected.
3. To add another PC, paste the same **sync link** into its sync box instead of a token.

No PC? Paste a GitHub token on the phone and start there. You can add the PC later with the sync link.

**The newest change always wins.** Every server carries the time of its last edit, and every device fetches the latest list before sending its own. A device you open after a few days never brings old entries back, and a server deleted on one device stays deleted everywhere.

> [!IMPORTANT]
> The sync link holds your token and the encryption key. It works like a password, so don't share it.<br>
> The app only asks for the `gist` scope. You can revoke access any time in GitHub → Settings → Applications.

## Build

```
go build -trimpath -ldflags "-H windowsgui -s -w" -o "SSH Servers.exe" .
```

Go 1.24+ and Windows (WebView2 via [go-webview2](https://github.com/jchv/go-webview2)). The website ([`docs/`](docs/)) and the phone app ([`docs/app/`](docs/app/)) are static HTML/JS served by GitHub Pages.

The server list lives in `servers.json` next to the exe.

---

<a name="persian"></a>

## 🇮🇷 فارسی

<div align="center">

[🇬🇧 English](#english) | [🇮🇷 فارسی](#persian)

</div>

<div align="center" dir="rtl">

با یک کلیک روی ویندوز به سرورهای SSH خودت وصل شو. همین لیست روی گوشیت هم هست و همیشه سینک می‌مونه.

<img src="SSH-Servers.png" alt="SSH Servers" width="860">

</div>

<div dir="rtl">

<details>
<summary><b>☕ حمایت · Support</b> — USDT (TRC20) / TRX</summary>
<br>
<p align="center"><img src="https://raw.githubusercontent.com/Plus98ir/.github/main/tron-qr.png" width="200" alt="TRON address QR"></p>

```
TA53aFyGNfSAcrCaJi9QmA9x5W47nPJcf7
```

> [!WARNING]
> **فقط شبکه‌ی TRON.** ارسال از ERC20 یا BEP20 پول را از بین می‌برد.

</details>

## دانلود

- **ویندوز:** [`SSH Servers.exe`](https://github.com/Plus98ir/SSH-Servers/releases/latest/download/SSH.Servers.exe). فقط یک فایله و نصب لازم نداره. به WebView2 Runtime نیاز داره که ویندوز ۱۰ و ۱۱ خودشون دارن.
- **اندروید:** [`SSH-Servers.apk`](https://github.com/Plus98ir/SSH-Servers/releases/download/android-v1.0.0/SSH-Servers.apk). یه کلاینت SSH واقعی با ترمینال داخلی که با برنامه‌ی ویندوز سینک می‌شه.
- **سایت:** https://corenetbase.com/SSH-Servers/
- **نسخه‌ی وب (هر گوشی):** https://corenetbase.com/SSH-Servers/app/. بازش کن و از منوی مرورگر **Add to Home screen** رو بزن تا مثل یه اپ نصب بشه.

## چه کار می‌کنه

**روی ویندوز**
- هر سرور یه کارته. روش کلیک کن (یا کلیدهای `1` تا `9` رو بزن) تا `ssh` مستقیم توی Windows Terminal باز بشه، توی تب جدید یا پنجره‌ی جدا.
- **فایل‌ها (SFTP):** فایل‌منیجر داخلی برای هر سرور. بین پوشه‌ها بگرد، فایل یا پوشه رو از Explorer بکش و رها کن تا آپلود بشه، دانلود کن توی پوشهٔ Downloads، تغییر نام بده، حذف کن، دسترسی‌ها رو عوض کن و فایل‌های کانفیگ رو همون‌جا ویرایش کن. هر انتقال درصد پیشرفت داره و می‌شه لغوش کرد.
- **چند سرور با هم:** چند سرور رو کنار هم (Split) یا توی تب‌ها باز کن، یا یه دستور رو روی همه‌شون اجرا کن و نتیجه‌ی هر کدوم رو ببین.
- **ورود بدون رمز:** یک بار پسورد رو وارد می‌کنی و برنامه کلید SSH رو روی سرور نصب می‌کنه. پسورد هیچ‌جا ذخیره نمی‌شه.
- **سرور از نو نصب شده؟** اگه host key سرور عوض بشه، برنامه اثر انگشت جدید رو نشون می‌ده و با یک کلیک `known_hosts` رو درست می‌کنه. اگه ورود با کلید هم از کار افتاده باشه، یک بار پسورد می‌پرسه و کلید رو دوباره نصب می‌کنه.
- پینگ زنده روی هر کارت، پشتیبانی از مودم و روترهای قدیمی (`ssh-rsa`)، فارسی و انگلیسی.

**روی گوشی**
- همون کارت‌ها. روی هر سرور بزنی توی اپ SSH گوشیت باز می‌شه ([Termius](https://termius.com)، [ConnectBot](https://connectbot.org) یا هر اپی که لینک `ssh://` رو باز کنه). اگه هیچ اپی لینک رو باز نکرد، دستور `ssh` برات کپی می‌شه.
- می‌تونی سرور اضافه، ویرایش، حذف و جابه‌جا کنی. تغییرات به کامپیوتر هم می‌رسه.
- بدون اینترنت هم با آخرین لیستی که دیده کار می‌کنه.

<p align="center">
<img src="screenshots/pc-fa.png" width="520" alt="برنامه‌ی ویندوز">&nbsp;
<img src="screenshots/phone-fa.png" width="250" alt="اپ گوشی">
</p>

## سینک

لیست سرورها (اسم، آدرس، پورت، یوزر و رنگ) توی یک **Gist مخفی روی حساب GitHub خودت** ذخیره می‌شه و با AES-256-GCM رمزنگاری شده. هیچ سرور واسطه‌ای از طرف ما وجود نداره و دیتای هر کس روی GitHub خودشه. پسورد و کلید SSH هیچ‌وقت از کامپیوتر بیرون نمی‌ره.

۱. روی کامپیوتر **دکمه‌ی سینک** (بالای برنامه) و بعد **ورود با GitHub** رو بزن. صفحه‌ی GitHub توی مرورگر باز می‌شه؛ کد کوتاهی که برنامه نشون می‌ده رو وارد کن و Authorize رو بزن. همین؛ لازم نیست توکنی بسازی. با توکن شخصی (دسترسی `gist`) هم هنوز می‌شه وصل شد.<br>
۲. یه QR کد نشون داده می‌شه. با دوربین گوشی اسکنش کن تا اپ گوشی باز بشه و وصل بشه.<br>
۳. برای اضافه کردن یه کامپیوتر دیگه، به جای توکن همون **لینک سینک** رو توی بخش سینکش بذار.

کامپیوتر نداری؟ توکن GitHub رو روی گوشی وارد کن و از همون‌جا شروع کن. بعداً با لینک سینک می‌تونی کامپیوتر رو هم اضافه کنی.

**همیشه آخرین تغییر برنده‌ست.** هر سرور زمان آخرین ویرایشش رو با خودش داره و هر دستگاه قبل از فرستادن، اول آخرین لیست رو می‌گیره. پس اگه یه دستگاه رو بعد از چند روز باز کنی، اطلاعات قدیمی برنمی‌گرده. سروری هم که روی یه دستگاه حذف بشه، روی همه‌ی دستگاه‌ها حذف می‌مونه.

> [!IMPORTANT]
> لینک سینک توکن و کلید رمزنگاری رو با خودش داره. مثل رمز می‌مونه، به کسی نده.<br>
> برنامه فقط دسترسی `gist` می‌خواد و هر وقت خواستی می‌تونی از GitHub → Settings → Applications باطلش کنی.

## ساختن از سورس

```
go build -trimpath -ldflags "-H windowsgui -s -w" -o "SSH Servers.exe" .
```

Go نسخه‌ی ۱.۲۴ به بالا و ویندوز لازمه. سایت (پوشه‌ی [`docs/`](docs/)) و اپ گوشی (پوشه‌ی [`docs/app/`](docs/app/)) HTML/JS ساده‌ان و GitHub Pages منتشرشون می‌کنه.

لیست سرورها توی فایل `servers.json` کنار exe ذخیره می‌شه.

</div>
