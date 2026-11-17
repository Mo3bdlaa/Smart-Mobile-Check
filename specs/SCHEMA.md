# Device spec files

Every file under `specs/<brand>/<category>/` describes **one device model** and is
embedded into the executable at build time. A `specs` folder placed next to the
`.exe` is also loaded at runtime (same layout) and overrides/extends the built-in
files with the same `id`, so new models can be added without rebuilding.

## Fields (Apple iPhone)

| field | type | notes |
|---|---|---|
| `id` | string | kebab-case unique id, e.g. `iphone-17-pro-max` (also the file name) |
| `name` | string | marketing name, e.g. `iPhone 17 Pro Max` |
| `brand` | string | `Apple` |
| `platform` | string | `ios` |
| `series` | string | e.g. `17`, `SE`, `Air` |
| `tier` | string | `standard` \| `plus` \| `mini` \| `pro` \| `pro-max` \| `se` \| `e` \| `air` \| `max` |
| `releaseDate` | string | ISO date of retail availability |
| `productTypes` | string[] | lockdown `ProductType` identifiers, e.g. `["iPhone18,2"]` |
| `hardwareModels` | string[] | lockdown `HardwareModel` board ids, e.g. `["D94AP"]` (may be empty if unknown) |
| `regulatoryModels` | string[] | A-numbers, e.g. `["A3106","A3105"]` |
| `chip` | string | e.g. `A19 Pro` |
| `cpuArchitecture` | string | `arm64` or `arm64e` |
| `ramGB` | number | |
| `storageOptionsGB` | number[] | marketed capacities in GB (1 TB = 1024) |
| `colors` | string[] | marketing colour names |
| `display` | object | `sizeInches`, `technology`, `resolution` ("2868x1320"), `ppi`, `maxRefreshHz`, `alwaysOn` (bool), `peakBrightnessNits` |
| `battery` | object | `designCapacityMah` (number[] – all known variants), `ratedCycles` (cycles to 80%: 500 or 1000), `wiredChargingW`, `wirelessCharging` (string) |
| `biometrics` | string | `Face ID` \| `Touch ID` |
| `port` | string | `Lightning` \| `USB-C` |
| `usbSpec` | string | e.g. `USB 2 (480 Mb/s)` |
| `cellular` | object | `generation` (`4G`/`5G`), `mmWave` (bool or "US only"), `sim` (string description), `esimOnlyRegions` (string[] country names) |
| `cameras` | object | `rear` (string[]), `front` (string), `lidar` (bool) |
| `features` | string[] | e.g. `Dynamic Island`, `Action button`, `Camera Control`, `MagSafe`, `Emergency SOS via satellite`, `Crash Detection` |
| `ipRating` | string | e.g. `IP68 (6 m, 30 min)` |
| `weightGrams` | number | |
| `launchIOS` | string | iOS version shipped, e.g. `26.0` |
| `maxIOS` | string\|null | last supported major iOS, `null` if still current |
| `sources` | string[] | URLs used to compile the data |
| `notes` | string | optional free text |
