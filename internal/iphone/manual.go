package iphone

// ManualTest is a hands-on test the buyer performs on the phone while the app
// guides them. Results are folded into the score and the report.
type ManualTest struct {
	ID       string   `json:"id"`
	Group    string   `json:"group"`
	Title    string   `json:"title"`
	Steps    []string `json:"steps"`
	Severity Severity `json:"severity"`
	Requires string   `json:"requires,omitempty"`
}

// ManualTests is the guided checklist, in order.
var ManualTests = []ManualTest{
	{ID: "parts-history", Group: "Genuine parts", Title: "Parts and Service History", Severity: Major, Steps: []string{
		"Open Settings › General › About.",
		"Look for a \"Parts and Service History\" section (iOS 15.2+, more detail on iOS 17.4+).",
		"PASS if it is absent or lists only \"Genuine Apple Part\". FAIL if any item shows \"Unknown Part\".",
	}},
	{ID: "battery-settings", Group: "Genuine parts", Title: "Battery Health screen", Severity: Major, Steps: []string{
		"Open Settings › Battery › Battery Health (& Charging).",
		"Compare Maximum Capacity with the value shown by this app.",
		"FAIL if it says \"Unknown Part\", \"Service\" or \"Unable to verify this iPhone has a genuine Apple battery\".",
	}},
	{ID: "true-tone", Group: "Display", Title: "True Tone & display origin", Severity: Major, Requires: "Face ID models / iPhone 8 and later", Steps: []string{
		"Open Control Center and long-press the brightness slider.",
		"PASS if the True Tone button is present and can be toggled.",
		"A missing True Tone toggle is a classic sign of a replaced (often non-original) screen.",
	}},
	{ID: "display-pixels", Group: "Display", Title: "Dead pixels, burn-in & tint", Severity: Major, Steps: []string{
		"Set brightness to maximum. Open a plain white page, then a black image, then red/green/blue images.",
		"Look for dead or stuck pixels, yellow/pink tint, shadows, lines or burned-in keyboard/status-bar ghosts.",
	}},
	{ID: "touch", Group: "Display", Title: "Touch screen", Severity: Major, Steps: []string{
		"Long-press an app icon until it jiggles, then drag it slowly to every corner and edge of the screen.",
		"FAIL if the icon drops or jumps anywhere (dead touch zones). Also test pinch-to-zoom in Photos.",
	}},
	{ID: "faceid", Group: "Biometrics", Title: "Face ID / Touch ID", Severity: Major, Steps: []string{
		"Settings › Face ID & Passcode (or Touch ID & Passcode) › Set Up / Add an alternate appearance.",
		"FAIL if you see \"Face ID is not available\" / \"Unable to activate Touch ID\" or enrollment fails.",
	}},
	{ID: "cameras", Group: "Cameras", Title: "All cameras, focus & flash", Severity: Major, Steps: []string{
		"Open Camera. Switch through every lens (0.5×, 1×, 2×/telephoto) and the front camera.",
		"Check focus on near and far objects, video recording, flash/torch, and listen for rattling or buzzing while moving the phone.",
		"Look for dust or spots inside the lenses and cracks on the lens covers.",
	}},
	{ID: "audio", Group: "Audio", Title: "Speakers & microphones", Severity: Major, Steps: []string{
		"Play music at full volume from both the bottom speaker and the earpiece (hold to your ear during a call).",
		"Record a Voice Memo, a selfie video and a rear video; play them back to check all microphones.",
		"Make a short call and confirm the other side hears you clearly.",
	}},
	{ID: "buttons", Group: "Buttons & haptics", Title: "Buttons, switch & vibration", Severity: Minor, Steps: []string{
		"Press volume up/down, side button, Action button / mute switch and Camera Control (if present).",
		"Toggle silent mode and confirm the phone vibrates (Taptic Engine).",
	}},
	{ID: "charging", Group: "Power", Title: "Wired & wireless charging", Severity: Major, Steps: []string{
		"Connect the charging cable: the lightning bolt must appear and the connection must hold when the cable is wiggled.",
		"If supported, place the phone on a MagSafe/Qi charger and confirm it charges.",
	}},
	{ID: "connectivity", Group: "Connectivity", Title: "Cellular, Wi-Fi, Bluetooth & GPS", Severity: Major, Steps: []string{
		"Insert your SIM (or eSIM): confirm signal bars, a phone call and mobile data.",
		"Join a Wi-Fi network, pair a Bluetooth device and open Maps to confirm your location is found.",
	}},
	{ID: "carrier-lock", Group: "Connectivity", Title: "Carrier lock", Severity: Major, Steps: []string{
		"Open Settings › General › About › Carrier Lock.",
		"PASS only if it shows \"No SIM restrictions\".",
	}},
	{ID: "sensors", Group: "Sensors", Title: "Proximity, light & motion sensors", Severity: Minor, Steps: []string{
		"During a call, cover the top of the screen: the display must turn off.",
		"With Auto-Brightness on, cover the top sensor: brightness should drop.",
		"Rotate the phone in Photos/Compass: rotation and compass must respond.",
	}},
	{ID: "body", Group: "Physical condition", Title: "Body, frame & water exposure", Severity: Minor, Steps: []string{
		"Inspect frame and back glass for cracks, dents or bent edges. Check the bottom screws for scratches (sign of opening).",
		"Eject the SIM tray (if present) and look inside: a red/pink liquid contact indicator means water exposure.",
		"Make sure the IMEI on the SIM tray / box matches the IMEI shown in this app.",
	}},
	{ID: "account", Group: "Ownership", Title: "Apple Account signed out", Severity: Critical, Steps: []string{
		"Open Settings: the top must show \"Sign in to your iPhone\", not a person's name.",
		"If someone is signed in, ask them to sign out (Settings › [name] › Sign Out) and erase the phone in front of you.",
	}},
}

// ManualResult is the outcome of a hands-on test as reported by the UI.
type ManualResult struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	Note   string `json:"note,omitempty"`
}

// ApplyManual returns a copy of the result with hands-on tests merged in and
// the score recomputed.
func ApplyManual(base *Result, manual map[string]ManualResult) *Result {
	if base == nil {
		return nil
	}
	r := *base
	r.Checks = nil
	for _, c := range base.Checks {
		if c.Category != CatHands {
			r.Checks = append(r.Checks, c)
		}
	}
	for _, t := range ManualTests {
		m, ok := manual[t.ID]
		if !ok || (m.Status != Pass && m.Status != Fail && m.Status != Skip) {
			continue
		}
		c := Check{ID: "manual-" + t.ID, Category: CatHands, Title: t.Title, Status: m.Status, Severity: t.Severity, Actual: m.Note}
		switch m.Status {
		case Pass:
			c.Summary = "Tested — OK."
		case Fail:
			c.Summary = "Tested — problem found."
		case Skip:
			c.Summary = "Not tested."
			c.Severity = Note
		}
		r.Checks = append(r.Checks, c)
	}
	Score(&r)
	return &r
}
