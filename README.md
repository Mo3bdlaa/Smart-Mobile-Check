# Smart Mobile Check

A **portable Windows app** (single `.exe`, no installer) that inspects a used phone before you buy it.
Plug the phone into a PC with a USB cable, and the app reads the hardware identity, locks, battery wear and
diagnostic logs, flags tampering, and walks you through hands-on tests. It then produces a score, a verdict and a
printable report.

Both routes work end to end: **iPhone** (over Apple's USB service) and **Android** (over USB debugging / ADB).
See [docs/ROADMAP.md](docs/ROADMAP.md) for what's next.

**Languages:** Arabic is the default. The interface uses simple Arabic with a right-to-left layout, and the check
results, hands-on tests, new-iPhone checklist and saved reports are all in Arabic too. Switch to English with the
button in the top bar; the choice is remembered.

## Download & run

1. Download [`releases/SmartMobileCheck-0.3.0-windows-x64.exe`](releases/) (SHA-256 in `releases/SHA256SUMS.txt`).
2. Install Apple's USB driver **once**. Use the **Apple Devices** app from the Microsoft Store (recommended) or **iTunes**.
   The app detects whether it is missing and links to it.
3. Double-click the `.exe`. Windows SmartScreen may warn because the file is not code-signed: click *More info → Run anyway*.
4. Connect the iPhone, unlock it, tap **Trust**, then click **Start full scan**.

To explore without a phone, use the **demo** buttons on the home screen.

Requirements: Windows 10/11 x64. The window uses the Microsoft Edge WebView2 runtime, which ships with Windows 10/11.
If WebView2 is missing, the app opens in your default browser instead. Reports are saved to
`Documents\Smart Mobile Check\Reports`.

## What the iPhone scan checks

| Area | Automated checks |
|---|---|
| **Identity & authenticity** | Model recognised in the spec DB · model reported consistently by three independent services (lockdown, file service, MobileGestalt) · logic board ID · regulatory A-number · SoC chip ID · **UDID re-derived from chip ID + ECID** (catches spoofed identities) · serial number format and cross-service match · production vs prototype silicon · part-number prefix (new / refurbished / replacement / demo unit) · sales region and eSIM-only notice · IMEI and IMEI2 Luhn validation · storage capacity vs official configurations |
| **Ownership & locks** | Find My / **Activation Lock** · activation state · **Remote Management (DEP/MDM)** enrollment and supervision · installed configuration profiles, including non-removable ones · bricked state · passcode |
| **Software integrity** | **Jailbreak**: `afc2` root-filesystem service, known jailbreak/TrollStore bundle IDs, non-Apple apps registered as *system* apps, suspicious app names · sideloading tools · iOS version plausible for the hardware (catches spoofed version/model) · build consistency · beta firmware · Developer Mode |
| **Battery** | Health % (max capacity) · cycle count vs rated cycles (500, or 1000 for iPhone 15 and later) · **design capacity vs factory battery** (detects replaced or third-party batteries) · cycle counter reset with worn capacity (gauge tampering) · temperature · gauge serial |
| **Hardware & diagnostics** | Kernel **panic logs** parsed for "Missing sensor(s)", which points to disconnected or non-original parts after a bad repair · cellular modem (baseband) · Wi-Fi/Bluetooth chips · storage controller self-test · SIM state |
| **Hands-on tests (guided)** | Parts & Service History · Battery Health screen · True Tone (screen originality) · dead pixels / burn-in · touch · Face ID / Touch ID · cameras · speakers & microphones · buttons & haptics · charging · cellular/Wi-Fi/BT/GPS · carrier lock · sensors · body & water exposure · Apple Account signed out |

## What the Android scan checks

On the phone, enable **Developer options** (tap *Build number* 7 times) and **USB debugging**, connect the cable
and tap **Allow** on the phone. On first use the app downloads Google's official ADB tool (about 9 MB, from
dl.google.com) into `%LOCALAPPDATA%\SmartMobileCheck\platform-tools`. All commands are read-only.

| Area | Automated checks |
|---|---|
| **Identity & clones** | Model recognised · build fingerprint, brand, device and vendor/odm partitions agree (catches edited props) · **chipset vs the model's real chip** (catches fake/clone phones) · RAM and storage vs official configurations · screen resolution vs spec · serial number (generic clone serials) · IMEI checksum (when Android exposes it) |
| **Ownership & locks** | **Google / Samsung / Xiaomi / Huawei / OPPO accounts** (Factory Reset Protection) · **company management (device owner / work profile)** · **Samsung Knox Guard** (financed or leased phones) · carrier lock |
| **Software integrity** | Bootloader lock and verified-boot state · **root** (su binaries, Magisk, KernelSU, APatch, LSPosed, Lucky Patcher…) · SELinux · official vs test/custom ROM · **Samsung Knox warranty bit** · security-patch age · Android version plausible for the model · apps installed from outside a store |
| **Battery** | Android battery health state · capacity % (fuel gauge or full ÷ design) · cycle count · original-battery capacity vs spec · temperature (what each phone exposes varies; the rest is covered by a hands-on test) |
| **Hardware** | Sensors (accelerometer, gyroscope, compass, proximity, light) · camera count · cellular modem · kernel panics and system crashes · SIM state |
| **Hands-on tests** | Manufacturer diagnostics (Samsung Members, `*#0*#`, Pixel diagnostics, Xiaomi CIT) · battery screen · display & burn-in · touch · fingerprint/face · cameras · audio · buttons & S Pen · charging · connectivity · sensors · body · accounts removed & FRP after reset · IMEI |

The Android spec database has **95 models**: Samsung Galaxy S22–S26, A05s–A57 and Z Fold/Flip 5–8, Google Pixel 7–11, Xiaomi / Redmi / POCO, OnePlus, OPPO, HONOR and realme. Specs live in `specs/android/<brand>/*.json` (schema in [`specs/android/SCHEMA.md`](specs/android/SCHEMA.md)).

### Buying a brand-new iPhone

Choose **New / sealed iPhone** on the connect screen, or open **"Buying a brand-new iPhone?"** on the home screen.
You can start this checklist before the phone is even unboxed.

- **Automatic new-device checks:**
  - Retail part number (M), with no refurbished, replacement or demo units.
  - Battery at 0–5 cycles and 100% capacity.
  - No previous owner's apps or stored data.
  - Clean crash history.
  - Activation state.
- **New-iPhone checklist:**
  - Factory seal and pull-tabs.
  - Box label serial and IMEI match the phone.
  - Part number on the box starts with M.
  - Apple coverage shows the phone was never activated.
  - First start shows the "Hello" screen.
  - No Activation Lock.
  - Battery Health 100% with 0 cycles.
  - No Parts & Service History.
  - eSIM / region suitability.
  - Local registration or customs.
  - Pristine body and complete box.
  - The functional tests.

A used phone that is sold as new is flagged **"Do not buy"**.

Scoring: each failed or warned check deducts points by severity (critical / major / minor). Any **critical** failure,
such as Activation Lock, MDM, jailbreak or spoofed identity, forces the verdict **"Do not buy"**.

### Limits

Everything is read from what the phone itself reports over Apple's USB protocols. iOS does not expose some
information to a PC, for example per-part genuineness ("Unknown Part") and carrier-lock status. Those items are
covered by the guided hands-on tests. Always check the IMEI against a blacklist and the serial number at
checkcoverage.apple.com before paying.

## Device spec database

`specs/apple/iphone/*.json` holds one file per model, currently **48 iPhones** from iPhone 5s to iPhone 18 Pro Max /
iPhone Duo. Each file has identifiers, board IDs, A-numbers, chip, RAM, storage options, battery capacities, display,
cameras, iOS support range and sources. The schema is in [`specs/SCHEMA.md`](specs/SCHEMA.md).
`specs/apple/reference/ios-reference.json` holds region codes, part-number prefixes, chip IDs and jailbreak app signatures.

The files are **embedded in the `.exe`**. To add or correct models **without rebuilding**, put a `specs` folder next
to the `.exe` with the same layout. Files with the same `id` override the built-in ones, and new files are added.

## Building

Builds cross-compile from any OS. No CGO and no Windows machine are required.

```bash
go install github.com/tc-hib/go-winres@latest   # optional: icon + manifest + version info
./build.sh 0.3.0                                 # -> dist/SmartMobileCheck-0.3.0-windows-x64.exe
go test ./...                                    # check-engine and report tests
go run ./cmd/smartcheck --browser                # run locally (UI in the browser)
```

## Testing without an iPhone

`internal/fakedevice` is a protocol-level iPhone simulator. It emulates Apple's USB service (usbmuxd) and a connected
iPhone: pairing with the Trust prompt, encrypted lockdown sessions, diagnostics, installed apps, management profiles,
the file service and crash logs. The real scanning code runs against it unchanged.

- `go test ./...` runs a full trust → scan → evaluate cycle against a genuine and a tampered simulated iPhone.
- Interactive use:
  ```bash
  go run ./tools/fakeiphone -profile genuine      # or -profile tampered; listens on 127.0.0.1:27015
  USBMUXD_SOCKET_ADDRESS=tcp://127.0.0.1:27015 go run ./cmd/smartcheck --browser
  go run ./tools/fakeandroid -profile a-healthy   # or a-clone / a-new; a simulated adb server on 127.0.0.1:5037
  ```
- `internal/fakedevice` also emulates an **adb server with an Android phone** (including the "Allow USB debugging"
  prompt). The Android tests run the real ADB client and collector against it.
  On Windows, stop the *Apple Mobile Device Service* first. The simulator uses the same port, so the `.exe` connects
  to it directly.

## Project layout

```
cmd/smartcheck/       entry point, native WebView2 window (Windows), icon/manifest (winres/)
internal/iphone/      device collector (USB/lockdown services), check engine, demo data, hands-on tests
internal/specs/       spec database loader (embedded + external override folder)
internal/server/      local HTTP API on 127.0.0.1 (loopback only)
internal/report/      standalone HTML report renderer
internal/android/     Android collector (ADB shell, read-only), check engine, demo data, hands-on tests
internal/adb/         ADB server-protocol client + download of Google's platform-tools
internal/inspect/     shared check/score/verdict types
internal/fakedevice/  iPhone (usbmuxd) and Android (adb) protocol simulators used by tests and tools/
web/                  user interface (HTML/CSS/JS, embedded, works offline)
specs/                device specification database (embedded)
releases/             prebuilt Windows executables
```
