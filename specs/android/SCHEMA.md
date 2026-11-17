# Android device spec files

One JSON file per model family under `specs/android/<brand>/<id>.json`, embedded in the executable
(a `specs` folder next to the `.exe` can add or override files with the same `id`).
The app matches a connected phone by `ro.product.model` (`models`), then `ro.product.device` (`devices`).

| field | type | notes |
|---|---|---|
| `id` | string | kebab-case, e.g. `galaxy-s24-ultra` (also the file name) |
| `name` | string | marketing name, e.g. `Galaxy S24 Ultra` |
| `brand` | string | `Samsung`, `Google`, `Xiaomi`, ... |
| `platform` | string | `android` |
| `releaseDate` | string | ISO date |
| `models` | string[] | every `ro.product.model` value, e.g. `["SM-S928B","SM-S928B/DS","SM-S928U","SM-S928U1","SM-S928W","SM-S928N","SM-S9280","SM-S928E"]` |
| `devices` | string[] | `ro.product.device` / codename values, e.g. `["e3q"]` |
| `boardPlatforms` | string[] | `ro.board.platform` values, e.g. `["pineapple"]` (Snapdragon 8 Gen 3), `["s5e9945"]` (Exynos 2400) |
| `socModels` | string[] | `ro.soc.model` values when known, e.g. `["SM8650"]` |
| `chip` | string | e.g. `Snapdragon 8 Gen 3 for Galaxy` (list regional variants in `notes`) |
| `ramGB` | number[] | RAM options |
| `storageOptionsGB` | number[] | storage options (1 TB = 1024) |
| `display` | object | `sizeInches`, `technology`, `resolution` (native, "1440x3120" width x height), `maxRefreshHz` |
| `battery` | object | `designCapacityMah` (number[], rated/minimum capacity), `typicalMah` (number, optional), `wiredChargingW`, `wirelessCharging` |
| `biometrics` | string | e.g. `Ultrasonic in-display fingerprint, face unlock` |
| `port` | string | e.g. `USB-C (USB 3.2 Gen 1)` |
| `cellular` | object | `generation`, `sim` (description, e.g. "Nano-SIM + eSIM") |
| `cameras` | object | `rear` (string[]), `front` (string) |
| `features` | string[] | e.g. `S Pen`, `IP68`, `Wireless charging`, `Reverse wireless charging` |
| `launchAndroid` | string | Android version shipped, e.g. `14` |
| `updatePolicy` | string | e.g. `7 OS upgrades / 7 years security` |
| `maxAndroid` | string\|null | last major Android it gets per the policy if computable (e.g. `21`), else null |
| `weightGrams` | number | |
| `sources` | string[] | URLs used |
| `notes` | string | optional |
