// Package specs loads the device specification database. Specs are embedded in
// the executable and can be extended/overridden at runtime by a "specs" folder
// placed next to the executable (same folder layout as the built-in one).
package specs

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Display describes the screen of a device.
type Display struct {
	SizeInches         float64 `json:"sizeInches,omitempty"`
	Technology         string  `json:"technology,omitempty"`
	Resolution         string  `json:"resolution,omitempty"`
	PPI                int     `json:"ppi,omitempty"`
	MaxRefreshHz       int     `json:"maxRefreshHz,omitempty"`
	AlwaysOn           bool    `json:"alwaysOn,omitempty"`
	PeakBrightnessNits int     `json:"peakBrightnessNits,omitempty"`
}

// Battery describes the factory battery.
type Battery struct {
	DesignCapacityMah []int  `json:"designCapacityMah,omitempty"`
	RatedCycles       int    `json:"ratedCycles,omitempty"`
	WiredChargingW    any    `json:"wiredChargingW,omitempty"`
	WirelessCharging  string `json:"wirelessCharging,omitempty"`
}

// Cellular describes radio capabilities.
type Cellular struct {
	Generation      string   `json:"generation,omitempty"`
	MmWave          any      `json:"mmWave,omitempty"`
	SIM             string   `json:"sim,omitempty"`
	ESIMOnlyRegions []string `json:"esimOnlyRegions,omitempty"`
}

// Cameras describes the camera system.
type Cameras struct {
	Rear  []string `json:"rear,omitempty"`
	Front string   `json:"front,omitempty"`
	Lidar bool     `json:"lidar,omitempty"`
}

// Device is one device model in the specification database.
type Device struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Brand            string   `json:"brand"`
	Platform         string   `json:"platform"`
	Series           string   `json:"series,omitempty"`
	Tier             string   `json:"tier,omitempty"`
	ReleaseDate      string   `json:"releaseDate,omitempty"`
	ProductTypes     []string `json:"productTypes"`
	HardwareModels   []string `json:"hardwareModels,omitempty"`
	RegulatoryModels []string `json:"regulatoryModels,omitempty"`
	Chip             string   `json:"chip,omitempty"`
	CPUArchitecture  string   `json:"cpuArchitecture,omitempty"`
	RAMGB            float64  `json:"ramGB,omitempty"`
	StorageOptionsGB []int    `json:"storageOptionsGB,omitempty"`
	Colors           []string `json:"colors,omitempty"`
	Display          Display  `json:"display"`
	Battery          Battery  `json:"battery"`
	Biometrics       string   `json:"biometrics,omitempty"`
	Port             string   `json:"port,omitempty"`
	USBSpec          string   `json:"usbSpec,omitempty"`
	Cellular         Cellular `json:"cellular"`
	Cameras          Cameras  `json:"cameras"`
	Features         []string `json:"features,omitempty"`
	IPRating         string   `json:"ipRating,omitempty"`
	WeightGrams      float64  `json:"weightGrams,omitempty"`
	LaunchIOS        string   `json:"launchIOS,omitempty"`
	MaxIOS           *string  `json:"maxIOS"`
	Sources          []string `json:"sources,omitempty"`
	Notes            string   `json:"notes,omitempty"`

	// Origin tells whether the spec came from the built-in database or an
	// external override folder.
	Origin string `json:"origin,omitempty"`
}

// Region maps an Apple region suffix (e.g. "LL/A") to a market.
type Region struct {
	Code      string   `json:"code"`
	Countries []string `json:"countries"`
	Notes     string   `json:"notes,omitempty"`
}

// Chip maps an Apple SoC to its lockdown ChipID.
type Chip struct {
	Name   string   `json:"name"`
	ChipID []string `json:"chipIds"`
}

// Reference holds auxiliary lookup tables used by the checks.
type Reference struct {
	Regions         []Region          `json:"regions"`
	PartPrefixes    map[string]string `json:"partNumberPrefixes"`
	Chips           []Chip            `json:"chips"`
	JailbreakApps   []JailbreakApp    `json:"jailbreakApps"`
	SideloadApps    []JailbreakApp    `json:"sideloadApps"`
	SuspiciousNames []string          `json:"suspiciousAppNameFragments"`
}

// JailbreakApp is a known bundle identifier associated with jailbreaking,
// TrollStore or sideloading.
type JailbreakApp struct {
	BundleID string `json:"bundleId"`
	Name     string `json:"name"`
}

// DB is the loaded specification database.
type DB struct {
	mu        sync.RWMutex
	devices   []*Device
	byType    map[string]*Device
	Reference Reference
	Warnings  []string
	External  string

	android         []*AndroidDevice
	androidByModel  map[string]*AndroidDevice
	androidByDevice map[string]*AndroidDevice
}

// AndroidDevice is one Android model family in the specification database.
type AndroidDevice struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Brand            string    `json:"brand"`
	Platform         string    `json:"platform"`
	ReleaseDate      string    `json:"releaseDate,omitempty"`
	Models           []string  `json:"models"`
	Devices          []string  `json:"devices,omitempty"`
	BoardPlatforms   []string  `json:"boardPlatforms,omitempty"`
	SocModels        []string  `json:"socModels,omitempty"`
	Chip             string    `json:"chip,omitempty"`
	RAMGB            []float64 `json:"ramGB,omitempty"`
	StorageOptionsGB []int     `json:"storageOptionsGB,omitempty"`
	Display          Display   `json:"display"`
	Battery          struct {
		DesignCapacityMah []int `json:"designCapacityMah,omitempty"`
		TypicalMah        int   `json:"typicalMah,omitempty"`
		RatedCycles       int   `json:"ratedCycles,omitempty"`
		WiredChargingW    any   `json:"wiredChargingW,omitempty"`
		WirelessCharging  any   `json:"wirelessCharging,omitempty"`
	} `json:"battery"`
	Biometrics    string   `json:"biometrics,omitempty"`
	Port          string   `json:"port,omitempty"`
	Cellular      Cellular `json:"cellular"`
	Cameras       Cameras  `json:"cameras"`
	Features      []string `json:"features,omitempty"`
	LaunchAndroid string   `json:"launchAndroid,omitempty"`
	UpdatePolicy  string   `json:"updatePolicy,omitempty"`
	MaxAndroid    *string  `json:"maxAndroid"`
	WeightGrams   float64  `json:"weightGrams,omitempty"`
	Sources       []string `json:"sources,omitempty"`
	Notes         string   `json:"notes,omitempty"`
	Origin        string   `json:"origin,omitempty"`
}

// normModel normalises a model number for lookup ("SM-S928B/DS" -> "sm-s928b").
func normModel(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	if i := strings.Index(m, "/"); i > 0 {
		m = m[:i]
	}
	return m
}

// AndroidByModel finds the spec for ro.product.model, falling back to the
// device codename (ro.product.device).
func (db *DB) AndroidByModel(model, device string) *AndroidDevice {
	if a := db.androidByModel[normModel(model)]; a != nil {
		return a
	}
	if a := db.androidByDevice[strings.ToLower(strings.TrimSpace(device))]; a != nil {
		return a
	}
	return nil
}

// Android returns every Android model in the database.
func (db *DB) Android() []*AndroidDevice {
	out := make([]*AndroidDevice, len(db.android))
	copy(out, db.android)
	return out
}

// Load reads the built-in specs and then the optional external folder.
func Load(builtin fs.FS, externalDir string) *DB {
	db := &DB{byType: map[string]*Device{}}
	byID := map[string]*Device{}
	var order []string
	androidByID := map[string]*AndroidDevice{}
	var androidOrder []string

	add := func(d *Device) {
		if _, ok := byID[d.ID]; !ok {
			order = append(order, d.ID)
		}
		byID[d.ID] = d
	}

	readTree := func(fsys fs.FS, origin string) {
		_ = fs.WalkDir(fsys, ".", func(path string, de fs.DirEntry, err error) error {
			if err != nil || de.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
				return nil
			}
			raw, err := fs.ReadFile(fsys, path)
			if err != nil {
				db.Warnings = append(db.Warnings, fmt.Sprintf("%s: %v", path, err))
				return nil
			}
			if strings.HasPrefix(filepath.ToSlash(path), "android/") {
				var a AndroidDevice
				if err := json.Unmarshal(raw, &a); err != nil {
					db.Warnings = append(db.Warnings, fmt.Sprintf("%s (%s): %v", path, origin, err))
					return nil
				}
				if a.ID == "" || (len(a.Models) == 0 && len(a.Devices) == 0) {
					db.Warnings = append(db.Warnings, fmt.Sprintf("%s (%s): missing id or models", path, origin))
					return nil
				}
				a.Origin = origin
				if _, ok := androidByID[a.ID]; !ok {
					androidOrder = append(androidOrder, a.ID)
				}
				androidByID[a.ID] = &a
				return nil
			}
			if strings.Contains(filepath.ToSlash(path), "reference/") {
				var ref Reference
				if err := json.Unmarshal(raw, &ref); err != nil {
					db.Warnings = append(db.Warnings, fmt.Sprintf("%s: %v", path, err))
					return nil
				}
				db.mergeReference(ref)
				return nil
			}
			var d Device
			if err := json.Unmarshal(raw, &d); err != nil {
				db.Warnings = append(db.Warnings, fmt.Sprintf("%s (%s): %v", path, origin, err))
				return nil
			}
			if d.ID == "" || len(d.ProductTypes) == 0 {
				db.Warnings = append(db.Warnings, fmt.Sprintf("%s (%s): missing id or productTypes", path, origin))
				return nil
			}
			d.Origin = origin
			add(&d)
			return nil
		})
	}

	if sub, err := fs.Sub(builtin, "specs"); err == nil {
		readTree(sub, "built-in")
	}
	if externalDir != "" {
		if st, err := os.Stat(externalDir); err == nil && st.IsDir() {
			db.External = externalDir
			readTree(os.DirFS(externalDir), "external")
		}
	}

	for _, id := range order {
		d := byID[id]
		db.devices = append(db.devices, d)
		for _, pt := range d.ProductTypes {
			db.byType[strings.ToLower(pt)] = d
		}
	}
	sort.SliceStable(db.devices, func(i, j int) bool {
		return db.devices[i].ReleaseDate > db.devices[j].ReleaseDate
	})
	db.androidByModel = map[string]*AndroidDevice{}
	db.androidByDevice = map[string]*AndroidDevice{}
	for _, id := range androidOrder {
		a := androidByID[id]
		db.android = append(db.android, a)
		for _, m := range a.Models {
			db.androidByModel[normModel(m)] = a
		}
		for _, d := range a.Devices {
			db.androidByDevice[strings.ToLower(strings.TrimSpace(d))] = a
		}
	}
	sort.SliceStable(db.android, func(i, j int) bool {
		if db.android[i].Brand != db.android[j].Brand {
			return db.android[i].Brand < db.android[j].Brand
		}
		return db.android[i].ReleaseDate > db.android[j].ReleaseDate
	})
	return db
}

func (db *DB) mergeReference(r Reference) {
	db.Reference.Regions = append(db.Reference.Regions, r.Regions...)
	if db.Reference.PartPrefixes == nil {
		db.Reference.PartPrefixes = map[string]string{}
	}
	for k, v := range r.PartPrefixes {
		db.Reference.PartPrefixes[k] = v
	}
	db.Reference.Chips = append(db.Reference.Chips, r.Chips...)
	db.Reference.JailbreakApps = append(db.Reference.JailbreakApps, r.JailbreakApps...)
	db.Reference.SideloadApps = append(db.Reference.SideloadApps, r.SideloadApps...)
	db.Reference.SuspiciousNames = append(db.Reference.SuspiciousNames, r.SuspiciousNames...)
}

// ByProductType returns the spec for a lockdown ProductType (e.g. "iPhone18,2").
func (db *DB) ByProductType(pt string) *Device {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.byType[strings.ToLower(pt)]
}

// All returns every device in the database (newest first).
func (db *DB) All() []*Device {
	db.mu.RLock()
	defer db.mu.RUnlock()
	out := make([]*Device, len(db.devices))
	copy(out, db.devices)
	return out
}

// Region looks up a region suffix such as "LL/A".
func (db *DB) Region(code string) *Region {
	code = strings.ToUpper(strings.TrimSpace(code))
	for i := range db.Reference.Regions {
		if strings.ToUpper(db.Reference.Regions[i].Code) == code {
			return &db.Reference.Regions[i]
		}
	}
	return nil
}

// ChipIDs returns the expected lockdown ChipIDs for a chip name.
func (db *DB) ChipIDs(chip string) []string {
	for _, c := range db.Reference.Chips {
		if strings.EqualFold(c.Name, chip) {
			return c.ChipID
		}
	}
	return nil
}
