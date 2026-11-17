# Roadmap

## v0.1 — iPhone route (done)
- Portable Windows `.exe`: native window, offline UI, platform picker (iPhone / Android)
- Detects the Apple driver and devices; handles the trust/pairing flow
- About 35 automated checks across identity, ownership, software integrity, battery and hardware
- 15 guided hands-on tests, score and verdict, HTML/JSON report export
- Embedded spec database (48 iPhones) with an external override folder
- Demo mode (healthy and tampered sample devices)

## v0.2 — iPhone depth
- Battery manufacture date vs device production date (detects replaced batteries)
- Display/camera/Face ID component serials from IORegistry where iOS exposes them
- Optional online lookups (coverage, blacklist) with the user's consent
- Colour code to marketing colour mapping per model
- Spec data refresh: per-region model numbers and battery variants
- Code-signing the executable

## v0.3 — Android route (first version done)
Done: ADB client, Google platform-tools download, clone/lock/root/battery/hardware checks, hands-on tests,
Arabic, simulator and tests. Next:
- Bundle `adb` (platform-tools) inside the executable; guide the user to enable USB debugging
- Checks: FRP/Google account lock, root (su binaries, Magisk), bootloader lock state / verified boot,
  Knox warranty bit (Samsung), battery health and cycles (`dumpsys battery`, `/sys/class/power_supply`),
  IMEI validation, build fingerprint vs model, storage and RAM vs spec, MDM / device owner, sideloaded apps
- Spec database: Samsung Galaxy S/A/Z, Google Pixel, Xiaomi, OnePlus (`specs/<brand>/android/*.json`)

## Later
- Single universal flow that auto-detects the platform when a phone is plugged in
- Multi-language UI (English / Arabic)
- Shop mode: batch inspection and a printable certificate with QR code
