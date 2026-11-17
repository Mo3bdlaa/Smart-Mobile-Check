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
	// Modes lists the purchase modes the test applies to ("used", "new").
	Modes []string `json:"modes"`
	// Link is an optional app link target (see /api/open), e.g. "coverage".
	Link string `json:"link,omitempty"`

	// Arabic presentation.
	GroupAr string   `json:"groupAr,omitempty"`
	TitleAr string   `json:"titleAr,omitempty"`
	StepsAr []string `json:"stepsAr,omitempty"`
}

var (
	both     = []string{ModeUsed, ModeNew}
	usedOnly = []string{ModeUsed}
	newOnly  = []string{ModeNew}
)

// ForMode reports whether the test applies to a purchase mode.
func (t ManualTest) ForMode(mode string) bool {
	for _, m := range t.Modes {
		if m == mode {
			return true
		}
	}
	return false
}

// ManualTests is the guided checklist, in order.
var ManualTests = []ManualTest{
	// Brand-new phone checklist.
	{ID: "box-seal", Modes: newOnly, Group: "Box & seal", Title: "Factory seal intact", Severity: Critical, Steps: []string{
		"iPhone 13 and later: the box has no shrink-wrap — check the two paper pull-tab strips on the bottom; they must be untouched and not re-glued.",
		"Older models: the shrink-wrap must be tight with Apple's neat factory fold, not loose or heat-sealed by hand.",
		"The box should be crisp: no dents, crushed corners or signs it was opened and closed again.",
	}},
	{ID: "box-label", Modes: newOnly, Group: "Box & seal", Title: "Box label matches the phone", Severity: Critical, Steps: []string{
		"Read the Serial No. and IMEI on the label on the back of the box.",
		"Compare them with this app's Device & specs tab and with Settings › General › About.",
		"FAIL if any number differs — the phone was swapped into another box.",
	}},
	{ID: "box-part", Modes: newOnly, Group: "Box & seal", Title: "Part number starts with M", Severity: Critical, Steps: []string{
		"On the box label, find the Part No. / Model (e.g. MYW63AB/A).",
		"M = new retail. F = refurbished, N = service replacement, 3 = shop demo unit — none of those is \"new\".",
	}},
	{ID: "coverage", Modes: newOnly, Group: "Warranty", Title: "Apple coverage shows a new phone", Severity: Critical, Link: "coverage", Steps: []string{
		"Open checkcoverage.apple.com and enter the serial number from the box.",
		"PASS if it says the phone still needs activation / purchase date not validated, or the 1-year limited warranty starts today.",
		"FAIL if the warranty has already been running for weeks or months — the phone was activated and used before.",
	}},
	{ID: "hello", Modes: newOnly, Group: "First start", Title: "Starts at the \"Hello\" setup screen", Severity: Critical, Steps: []string{
		"Power the phone on for the first time.",
		"It must show the \"Hello\" screen and the full setup assistant.",
		"FAIL if it goes straight to a Home Screen or a lock screen — it has been set up before.",
	}},
	{ID: "no-lock", Modes: newOnly, Group: "First start", Title: "No Activation Lock during setup", Severity: Critical, Steps: []string{
		"Go through setup until the phone is activated.",
		"FAIL if it shows \"iPhone Locked to Owner\" or asks for someone else's Apple Account — return it immediately.",
	}},
	{ID: "battery-new", Modes: newOnly, Group: "Battery", Title: "Battery Health 100% and 0 cycles", Severity: Major, Steps: []string{
		"Open Settings › Battery › Battery Health (& Charging).",
		"Maximum Capacity must be 100%. On iPhone 15 and later, Cycle Count must be 0 – 5.",
	}},
	{ID: "no-parts-history", Modes: newOnly, Group: "Battery", Title: "No Parts and Service History", Severity: Major, Steps: []string{
		"Open Settings › General › About.",
		"A new phone has no \"Parts and Service History\" section at all. FAIL if it exists.",
	}},
	{ID: "sim-region", Modes: newOnly, Group: "Your country", Title: "SIM type works for you", Severity: Major, Steps: []string{
		"US models (part number ending LL/A) of iPhone 14 and later have no SIM tray — eSIM only.",
		"Make sure your carrier supports eSIM, or buy a model with a physical SIM tray.",
		"Insert your SIM / set up your eSIM and make a test call before paying.",
	}},
	{ID: "registration", Modes: newOnly, Group: "Your country", Title: "Registration / customs paid", Severity: Major, Steps: []string{
		"In some countries (e.g. Egypt) imported phones must be registered and customs paid, or the network is blocked after a grace period.",
		"Ask for the official invoice and confirm the phone is registered / tax-paid for your country.",
	}},
	{ID: "pristine", Modes: newOnly, Group: "Physical condition", Title: "Pristine body, complete box", Severity: Minor, Steps: []string{
		"No scratches or marks on the frame, screen, back glass or camera lenses.",
		"The box contains the phone, the USB-C cable (Lightning on older models) and documentation. No charger since iPhone 12.",
	}},
	// Functional and used-phone tests.
	{ID: "parts-history", Modes: usedOnly, Group: "Genuine parts", Title: "Parts and Service History", Severity: Major, Steps: []string{
		"Open Settings › General › About.",
		"Look for a \"Parts and Service History\" section (iOS 15.2+, more detail on iOS 17.4+).",
		"PASS if it is absent or lists only \"Genuine Apple Part\". FAIL if any item shows \"Unknown Part\".",
	}},
	{ID: "battery-settings", Modes: usedOnly, Group: "Genuine parts", Title: "Battery Health screen", Severity: Major, Steps: []string{
		"Open Settings › Battery › Battery Health (& Charging).",
		"Compare Maximum Capacity with the value shown by this app.",
		"FAIL if it says \"Unknown Part\", \"Service\" or \"Unable to verify this iPhone has a genuine Apple battery\".",
	}},
	{ID: "true-tone", Modes: both, Group: "Display", Title: "True Tone & display origin", Severity: Major, Requires: "Face ID models / iPhone 8 and later", Steps: []string{
		"Open Control Center and long-press the brightness slider.",
		"PASS if the True Tone button is present and can be toggled.",
		"A missing True Tone toggle is a classic sign of a replaced (often non-original) screen.",
	}},
	{ID: "display-pixels", Modes: both, Group: "Display", Title: "Dead pixels, burn-in & tint", Severity: Major, Steps: []string{
		"Set brightness to maximum. Open a plain white page, then a black image, then red/green/blue images.",
		"Look for dead or stuck pixels, yellow/pink tint, shadows, lines or burned-in keyboard/status-bar ghosts.",
	}},
	{ID: "touch", Modes: both, Group: "Display", Title: "Touch screen", Severity: Major, Steps: []string{
		"Long-press an app icon until it jiggles, then drag it slowly to every corner and edge of the screen.",
		"FAIL if the icon drops or jumps anywhere (dead touch zones). Also test pinch-to-zoom in Photos.",
	}},
	{ID: "faceid", Modes: both, Group: "Biometrics", Title: "Face ID / Touch ID", Severity: Major, Steps: []string{
		"Settings › Face ID & Passcode (or Touch ID & Passcode) › Set Up / Add an alternate appearance.",
		"FAIL if you see \"Face ID is not available\" / \"Unable to activate Touch ID\" or enrollment fails.",
	}},
	{ID: "cameras", Modes: both, Group: "Cameras", Title: "All cameras, focus & flash", Severity: Major, Steps: []string{
		"Open Camera. Switch through every lens (0.5×, 1×, 2×/telephoto) and the front camera.",
		"Check focus on near and far objects, video recording, flash/torch, and listen for rattling or buzzing while moving the phone.",
		"Look for dust or spots inside the lenses and cracks on the lens covers.",
	}},
	{ID: "audio", Modes: both, Group: "Audio", Title: "Speakers & microphones", Severity: Major, Steps: []string{
		"Play music at full volume from both the bottom speaker and the earpiece (hold to your ear during a call).",
		"Record a Voice Memo, a selfie video and a rear video; play them back to check all microphones.",
		"Make a short call and confirm the other side hears you clearly.",
	}},
	{ID: "buttons", Modes: both, Group: "Buttons & haptics", Title: "Buttons, switch & vibration", Severity: Minor, Steps: []string{
		"Press volume up/down, side button, Action button / mute switch and Camera Control (if present).",
		"Toggle silent mode and confirm the phone vibrates (Taptic Engine).",
	}},
	{ID: "charging", Modes: both, Group: "Power", Title: "Wired & wireless charging", Severity: Major, Steps: []string{
		"Connect the charging cable: the lightning bolt must appear and the connection must hold when the cable is wiggled.",
		"If supported, place the phone on a MagSafe/Qi charger and confirm it charges.",
	}},
	{ID: "connectivity", Modes: both, Group: "Connectivity", Title: "Cellular, Wi-Fi, Bluetooth & GPS", Severity: Major, Steps: []string{
		"Insert your SIM (or eSIM): confirm signal bars, a phone call and mobile data.",
		"Join a Wi-Fi network, pair a Bluetooth device and open Maps to confirm your location is found.",
	}},
	{ID: "carrier-lock", Modes: both, Group: "Connectivity", Title: "Carrier lock", Severity: Major, Steps: []string{
		"Open Settings › General › About › Carrier Lock.",
		"PASS only if it shows \"No SIM restrictions\".",
	}},
	{ID: "sensors", Modes: both, Group: "Sensors", Title: "Proximity, light & motion sensors", Severity: Minor, Steps: []string{
		"During a call, cover the top of the screen: the display must turn off.",
		"With Auto-Brightness on, cover the top sensor: brightness should drop.",
		"Rotate the phone in Photos/Compass: rotation and compass must respond.",
	}},
	{ID: "body", Modes: usedOnly, Group: "Physical condition", Title: "Body, frame & water exposure", Severity: Minor, Steps: []string{
		"Inspect frame and back glass for cracks, dents or bent edges. Check the bottom screws for scratches (sign of opening).",
		"Eject the SIM tray (if present) and look inside: a red/pink liquid contact indicator means water exposure.",
		"Make sure the IMEI on the SIM tray / box matches the IMEI shown in this app.",
	}},
	{ID: "account", Modes: usedOnly, Group: "Ownership", Title: "Apple Account signed out", Severity: Critical, Steps: []string{
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
		if !t.ForMode(r.Mode) && r.Mode != "" {
			continue
		}
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
