# Smart Mobile Check

A **portable Windows app** (single `.exe`, no installer) that inspects a used phone before you buy it.
Plug the phone into a PC with a USB cable, and the app reads the hardware identity, locks, battery wear and
diagnostic logs, flags tampering, and walks you through hands-on tests. It then produces a score, a verdict and a
printable report.

The **iPhone route works end to end**. The Android route is on the roadmap (see [docs/ROADMAP.md](docs/ROADMAP.md)).

## Download & run

1. Download [`releases/SmartMobileCheck-0.1.2-windows-x64.exe`](releases/) (SHA-256 in `releases/SHA256SUMS.txt`).
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
./build.sh 0.1.2                                 # -> dist/SmartMobileCheck-0.1.2-windows-x64.exe
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
  ```
  On Windows, stop the *Apple Mobile Device Service* first. The simulator uses the same port, so the `.exe` connects
  to it directly.

## Project layout

```
cmd/smartcheck/       entry point, native WebView2 window (Windows), icon/manifest (winres/)
internal/iphone/      device collector (USB/lockdown services), check engine, demo data, hands-on tests
internal/specs/       spec database loader (embedded + external override folder)
internal/server/      local HTTP API on 127.0.0.1 (loopback only)
internal/report/      standalone HTML report renderer
internal/fakedevice/  iPhone + usbmuxd protocol simulator used by tests and tools/fakeiphone
web/                  user interface (HTML/CSS/JS, embedded, works offline)
specs/                device specification database (embedded)
releases/             prebuilt Windows executables
```
