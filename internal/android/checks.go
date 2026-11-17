package android

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/inspect"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

type (
	Check  = inspect.Check
	Status = inspect.Status
)

const (
	pass     = inspect.Pass
	warn     = inspect.Warn
	fail     = inspect.Fail
	info     = inspect.Info
	skip     = inspect.Skip
	critical = inspect.Critical
	major    = inspect.Major
	minor    = inspect.Minor
)

// Result is the outcome of an Android scan (same JSON shape as the iPhone one).
type Result struct {
	Snapshot    *Snapshot              `json:"snapshot"`
	Spec        any                    `json:"spec"`
	ModelName   string                 `json:"modelName"`
	Identity    []inspect.KV           `json:"identity"`
	Checks      []Check                `json:"checks"`
	Battery     inspect.BatteryReport  `json:"battery"`
	Score       int                    `json:"score"`
	Verdict     string                 `json:"verdict"`
	VerdictText string                 `json:"verdictText"`
	Counts      map[inspect.Status]int `json:"counts"`
	Storage     map[string]any         `json:"storage"`
	GeneratedAt time.Time              `json:"generatedAt"`
	AppVersion  string                 `json:"appVersion"`
	Mode        string                 `json:"mode"`
	Lang        string                 `json:"lang,omitempty"`
	Platform    string                 `json:"platform"`
}

type ev struct {
	s      *Snapshot
	spec   *specs.AndroidDevice
	db     *specs.DB
	mode   string
	lang   string
	now    time.Time
	checks []Check
}

func (e *ev) t(en, ar string) string {
	if e.lang == "ar" {
		return ar
	}
	return en
}

func (e *ev) add(c Check) {
	if c.Severity == "" {
		c.Severity = inspect.Note
	}
	e.checks = append(e.checks, c)
}

func (e *ev) p(k string) string { return strings.TrimSpace(e.s.Props[k]) }

// ManualResult is a hands-on test outcome reported by the UI.
type ManualResult struct {
	ID     string         `json:"id"`
	Status inspect.Status `json:"status"`
	Note   string         `json:"note,omitempty"`
}

// Evaluate runs every check and folds in hands-on results.
func Evaluate(s *Snapshot, db *specs.DB, appVersion, mode, lang string, manual map[string]ManualResult) *Result {
	if mode != inspect.ModeNew {
		mode = inspect.ModeUsed
	}
	if s.Props == nil {
		s.Props = map[string]string{}
	}
	e := &ev{s: s, db: db, mode: mode, lang: lang, now: time.Now()}
	e.spec = db.AndroidByModel(e.p("ro.product.model"), e.p("ro.product.device"))

	e.identityChecks()
	e.ownershipChecks()
	e.softwareChecks()
	b := e.batteryChecks()
	e.hardwareChecks()
	if mode == inspect.ModeNew {
		e.newDeviceChecks(b)
	}
	e.manualChecks(manual)

	r := &Result{Snapshot: s, Checks: e.checks, Battery: b, GeneratedAt: e.now, AppVersion: appVersion,
		Mode: mode, Lang: lang, Platform: "android", Storage: e.storage()}
	if e.spec != nil {
		r.Spec = e.spec
		r.ModelName = e.spec.Name
	} else {
		r.ModelName = e.marketingName()
	}
	r.Identity = e.identity(r.ModelName)
	o := inspect.Evaluate(r.Checks)
	r.Counts, r.Score = o.Counts, o.Score
	r.Verdict, r.VerdictText = inspect.VerdictText(o.Verdict, lang)
	return r
}

func (e *ev) marketingName() string {
	for _, k := range []string{"ro.product.marketname", "ro.product.vendor.marketname", "ro.vendor.product.display", "ro.config.marketing_name"} {
		if v := e.p(k); v != "" {
			return v
		}
	}
	brand := e.p("ro.product.brand")
	if brand != "" {
		brand = strings.ToUpper(brand[:1]) + brand[1:] + " "
	}
	return strings.TrimSpace(brand + e.p("ro.product.model"))
}

// ---------------------------------------------------------------- identity

func generic(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "" || v == "mainline" || v == "generic" || strings.HasPrefix(v, "aosp") || v == "unknown"
}

func (e *ev) identityChecks() {
	model := e.p("ro.product.model")
	device := e.p("ro.product.device")
	brand := strings.ToLower(e.p("ro.product.brand"))

	if e.spec != nil {
		e.add(Check{ID: "model", Category: inspect.CatIdentity, Title: e.t("Model recognised", "الموديل معروف"), Status: pass, Actual: model,
			Summary: e.t(fmt.Sprintf("%s (%s) found in the spec database.", e.spec.Name, model), fmt.Sprintf("%s (%s) موجود في قاعدة المواصفات.", e.spec.Name, model))})
	} else {
		e.add(Check{ID: "model", Category: inspect.CatIdentity, Title: e.t("Model recognised", "الموديل معروف"), Status: warn, Severity: minor, Actual: model,
			Summary: e.t("This model is not in the spec database, so spec cross-checks were skipped.", "الموديل ده مش موجود في قاعدة المواصفات، فمش هنقدر نقارن مواصفاته."),
			Advice:  e.t("Add a spec file for this model to the specs folder next to the app.", "ضيف ملف مواصفات للموديل ده في فولدر specs جنب البرنامج.")})
	}

	// Fingerprint & partition consistency (edited build.prop / clones).
	var mism []string
	fp := e.p("ro.build.fingerprint")
	if fpBrand, fpDevice, ok := parseFingerprint(fp); ok {
		if brand != "" && !strings.EqualFold(fpBrand, brand) {
			mism = append(mism, "fingerprint brand="+fpBrand+" vs brand="+brand)
		}
		if device != "" && !strings.EqualFold(fpDevice, device) {
			mism = append(mism, "fingerprint device="+fpDevice+" vs device="+device)
		}
	}
	for _, k := range []string{"ro.product.vendor.model", "ro.product.odm.model"} {
		if v := e.p(k); !generic(v) && model != "" && !strings.EqualFold(v, model) {
			mism = append(mism, k+"="+v)
		}
	}
	if vfp := e.p("ro.vendor.build.fingerprint"); vfp != "" {
		if vb, vd, ok := parseFingerprint(vfp); ok && brand != "" && (!strings.EqualFold(vb, brand) || (device != "" && !strings.EqualFold(vd, device))) {
			mism = append(mism, "vendor fingerprint "+vb+"/"+vd)
		}
	}
	if len(mism) > 0 {
		e.add(Check{ID: "consistency", Category: inspect.CatIdentity, Title: e.t("Identity consistency", "تطابق بيانات الجهاز"), Status: fail, Severity: critical,
			Summary: e.t("System partitions describe different phones — the identity was edited or this is a clone.", "أجزاء النظام بتوصف موبايلات مختلفة — بيانات الجهاز متعدّلة أو الموبايل تقليد."),
			Actual:  strings.Join(mism, "; ")})
	} else if fp != "" {
		e.add(Check{ID: "consistency", Category: inspect.CatIdentity, Title: e.t("Identity consistency", "تطابق بيانات الجهاز"), Status: pass,
			Summary: e.t("Build fingerprint, brand, device and partitions all agree.", "بصمة النظام والماركة والموديل كلهم متطابقين."), Actual: fp})
	}

	// Chipset vs spec (fake / clone phones).
	if e.spec != nil && (len(e.spec.BoardPlatforms) > 0 || len(e.spec.SocModels) > 0) {
		board := strings.ToLower(firstNonEmpty(e.p("ro.board.platform"), e.p("ro.hardware.chipname"), e.p("ro.hardware")))
		soc := e.p("ro.soc.model")
		ok := containsFold(e.spec.BoardPlatforms, board) || (soc != "" && containsFold(e.spec.SocModels, soc))
		actual := strings.Trim(board+" "+soc, " ")
		if actual != "" {
			exp := strings.Join(append(append([]string{}, e.spec.BoardPlatforms...), e.spec.SocModels...), " / ")
			if ok {
				e.add(Check{ID: "chip", Category: inspect.CatIdentity, Title: e.t("Processor", "المعالج"), Status: pass, Expected: exp, Actual: actual,
					Summary: e.t("Chipset matches "+e.spec.Chip+".", "المعالج مطابق: "+e.spec.Chip+".")})
			} else {
				e.add(Check{ID: "chip", Category: inspect.CatIdentity, Title: e.t("Processor", "المعالج"), Status: fail, Severity: critical, Expected: exp, Actual: actual,
					Summary: e.t("The processor is not the one this model uses — this is very likely a fake/clone phone.", "المعالج مش هو اللي في الموديل ده — غالباً الموبايل تقليد."),
					Advice:  e.t("Clones copy the model name but run on cheap chips. Do not buy.", "الموبايلات التقليد بتنسخ اسم الموديل بس بمعالج رخيص. ماتشتريش.")})
			}
		}
	}

	// RAM
	if e.s.MemTotalKB > 0 && e.spec != nil && len(e.spec.RAMGB) > 0 {
		gb := float64(e.s.MemTotalKB) / 1024 / 1024
		match := false
		for _, o := range e.spec.RAMGB {
			if gb >= o*0.78 && gb <= o+0.5 {
				match = true
			}
		}
		exp := joinFloats(e.spec.RAMGB, " GB")
		act := fmt.Sprintf("%.1f GB", gb)
		if match {
			e.add(Check{ID: "ram", Category: inspect.CatIdentity, Title: e.t("Memory (RAM)", "الرامات"), Status: pass, Expected: exp, Actual: act,
				Summary: e.t("RAM matches an official configuration.", "الرامات مطابقة لنسخة رسمية.")})
		} else {
			e.add(Check{ID: "ram", Category: inspect.CatIdentity, Title: e.t("Memory (RAM)", "الرامات"), Status: fail, Severity: major, Expected: exp, Actual: act,
				Summary: e.t("RAM does not match any official version of this model — fake specs or a clone.", "الرامات مش مطابقة لأي نسخة رسمية — مواصفات مزيّفة أو تقليد.")})
		}
	}

	// Storage
	if e.s.DataTotalKB > 0 && e.spec != nil && len(e.spec.StorageOptionsGB) > 0 {
		gb := float64(e.s.DataTotalKB) * 1024 / 1e9
		best := 0
		for _, o := range e.spec.StorageOptionsGB {
			m := float64(o)
			if o >= 1024 {
				m = float64(o) / 1024 * 1000
			}
			if gb >= m*0.72 && gb <= m*1.01 {
				best = o
				break
			}
		}
		exp := joinGB(e.spec.StorageOptionsGB)
		act := fmt.Sprintf("%.1f GB", gb)
		if best > 0 {
			e.add(Check{ID: "storage", Category: inspect.CatIdentity, Title: e.t("Storage capacity", "مساحة التخزين"), Status: pass, Expected: exp, Actual: act,
				Summary: e.t(gbLabel(best)+" model — capacity matches an official configuration.", "نسخة "+gbLabel(best)+" — المساحة رسمية وصحيحة.")})
		} else {
			e.add(Check{ID: "storage", Category: inspect.CatIdentity, Title: e.t("Storage capacity", "مساحة التخزين"), Status: fail, Severity: major, Expected: exp, Actual: act,
				Summary: e.t("Storage does not match any official version — fake storage or a clone.", "المساحة مش مطابقة لأي نسخة رسمية — تخزين مزيّف أو تقليد.")})
		}
	}

	// Screen resolution
	if e.s.ScreenSize != "" && e.spec != nil && e.spec.Display.Resolution != "" {
		if sameRes(e.s.ScreenSize, e.spec.Display.Resolution) {
			e.add(Check{ID: "screen", Category: inspect.CatIdentity, Title: e.t("Screen resolution", "دقة الشاشة"), Status: pass, Expected: e.spec.Display.Resolution, Actual: e.s.ScreenSize,
				Summary: e.t("Screen panel resolution matches the model.", "دقة الشاشة مطابقة للموديل.")})
		} else {
			e.add(Check{ID: "screen", Category: inspect.CatIdentity, Title: e.t("Screen resolution", "دقة الشاشة"), Status: warn, Severity: major, Expected: e.spec.Display.Resolution, Actual: e.s.ScreenSize,
				Summary: e.t("Screen resolution differs from this model — replaced screen or not the claimed phone.", "دقة الشاشة مختلفة عن الموديل — شاشة متغيّرة أو الموبايل مش هو اللي مكتوب.")})
		}
	}

	// Serial
	serial := firstNonEmpty(e.p("ro.serialno"), e.p("ro.boot.serialno"), e.s.Serial)
	switch strings.ToUpper(serial) {
	case "", "UNKNOWN", "0123456789ABCDEF", "0123456789", "1234567890ABCDEF":
		e.add(Check{ID: "serial", Category: inspect.CatIdentity, Title: e.t("Serial number", "الرقم التسلسلي"), Status: fail, Severity: major, Actual: serial,
			Summary: e.t("Missing or generic serial number — typical of clones and tampered phones.", "الرقم التسلسلي ناقص أو عام — ده بيحصل في الموبايلات التقليد أو المتلاعب فيها.")})
	default:
		e.add(Check{ID: "serial", Category: inspect.CatIdentity, Title: e.t("Serial number", "الرقم التسلسلي"), Status: pass, Actual: serial,
			Summary: e.t("Serial number present.", "الرقم التسلسلي موجود.")})
	}

	// IMEI
	if len(e.s.IMEIs) == 0 {
		e.add(Check{ID: "imei", Category: inspect.CatIdentity, Title: "IMEI", Status: skip,
			Summary: e.t("Android does not allow reading the IMEI over USB on this phone.", "أندرويد مش بيسمح بقراءة الـ IMEI بالكابل على الموبايل ده."),
			Advice:  e.t("Dial *#06# and compare the IMEI with the box and the SIM tray.", "اطلب ‎*#06#‎ وقارن الـ IMEI باللي على العلبة ودرج الشريحة.")})
	} else {
		bad := []string{}
		for _, im := range e.s.IMEIs {
			if !luhnValid(im) {
				bad = append(bad, im)
			}
		}
		if len(bad) > 0 {
			e.add(Check{ID: "imei", Category: inspect.CatIdentity, Title: "IMEI", Status: fail, Severity: critical, Actual: strings.Join(e.s.IMEIs, " · "),
				Summary: e.t("IMEI fails the checksum — invalid or altered.", "رقم IMEI غير صحيح — متغيّر أو متلاعب فيه.")})
		} else {
			e.add(Check{ID: "imei", Category: inspect.CatIdentity, Title: "IMEI", Status: pass, Actual: strings.Join(e.s.IMEIs, " · "),
				Summary: e.t("Valid IMEI (checksum OK).", "رقم IMEI صحيح."),
				Advice:  e.t("Compare with *#06#, the box, and make sure it is not blacklisted.", "قارنه بـ ‎*#06#‎ وبالعلبة، واتأكد إنه مش مبلّغ عنه.")})
		}
	}

	// Android version plausibility
	rel := majorVersion(e.p("ro.build.version.release"))
	if e.spec != nil && rel > 0 && e.spec.LaunchAndroid != "" {
		launch := majorVersion(e.spec.LaunchAndroid)
		max := 0
		if e.spec.MaxAndroid != nil {
			max = majorVersion(*e.spec.MaxAndroid)
		}
		rng := fmt.Sprintf("Android %d+", launch)
		if max > 0 {
			rng = fmt.Sprintf("Android %d – %d", launch, max)
		}
		act := "Android " + e.p("ro.build.version.release")
		switch {
		case rel < launch:
			e.add(Check{ID: "android-version", Category: inspect.CatSoftware, Title: e.t("Android version", "نسخة أندرويد"), Status: fail, Severity: critical, Expected: rng, Actual: act,
				Summary: e.t("This Android version is older than the one the model shipped with — the readings are fake.", "نسخة أندرويد أقدم من اللي الموبايل نزل بيها — البيانات مزيّفة.")})
		case max > 0 && rel > max:
			e.add(Check{ID: "android-version", Category: inspect.CatSoftware, Title: e.t("Android version", "نسخة أندرويد"), Status: warn, Severity: major, Expected: rng, Actual: act,
				Summary: e.t("Newer than any official update for this model — likely a custom ROM.", "أحدث من أي تحديث رسمي للموديل ده — غالباً روم معدّل.")})
		default:
			e.add(Check{ID: "android-version", Category: inspect.CatSoftware, Title: e.t("Android version", "نسخة أندرويد"), Status: pass, Expected: rng, Actual: act,
				Summary: e.t("Android version is valid for this model.", "نسخة أندرويد مناسبة للموديل.")})
		}
	}
}

func parseFingerprint(fp string) (brand, device string, ok bool) {
	// brand/product/device:release/id/incremental:type/tags
	parts := strings.Split(fp, "/")
	if len(parts) < 3 {
		return "", "", false
	}
	dev, _, _ := strings.Cut(parts[2], ":")
	return parts[0], dev, true
}

// ---------------------------------------------------------------- ownership

var lockAccounts = map[string][2]string{
	"com.google":           {"Google account (Factory Reset Protection)", "حساب جوجل (حماية إعادة الضبط FRP)"},
	"com.osp.app.signin":   {"Samsung account (Reactivation Lock)", "حساب سامسونج (قفل إعادة التفعيل)"},
	"com.xiaomi":           {"Xiaomi account (Find Device lock)", "حساب شاومي (قفل البحث عن الجهاز)"},
	"com.huawei.hwid":      {"Huawei ID (activation lock)", "حساب هواوي (قفل التفعيل)"},
	"com.oplus.account":    {"OPPO / realme / OnePlus account", "حساب أوبو / ريلمي / ون بلس"},
	"com.heytap.account":   {"OPPO / realme / OnePlus account", "حساب أوبو / ريلمي / ون بلس"},
	"com.hihonor.id":       {"HONOR ID (activation lock)", "حساب هونر (قفل التفعيل)"},
	"com.motorola.account": {"Motorola account", "حساب موتورولا"},
	"com.vivo.account":     {"vivo account", "حساب فيفو"},
}

func (e *ev) ownershipChecks() {
	// Accounts / FRP
	if e.s.AccountsRead {
		var locks []string
		for _, a := range e.s.Accounts {
			if l, ok := lockAccounts[a.Type]; ok {
				locks = append(locks, e.t(l[0], l[1])+": "+a.Name)
			}
		}
		if len(locks) > 0 {
			e.add(Check{ID: "accounts", Category: inspect.CatOwner, Title: e.t("Accounts & reset lock (FRP)", "الحسابات وقفل إعادة الضبط (FRP)"), Status: fail, Severity: critical,
				Summary: e.t(fmt.Sprintf("%d lock-enabled account(s) are signed in — after a reset the phone will ask for the owner's password.", len(locks)),
					fmt.Sprintf("فيه %d حساب مسجّل يقفل الموبايل — بعد إعادة الضبط هيطلب باسورد صاحبه.", len(locks))),
				Actual: strings.Join(locks, " · "),
				Advice: e.t("Ask the seller to remove every account (Settings › Accounts), then factory-reset the phone in front of you.", "خلّي البائع يمسح كل الحسابات (الإعدادات › الحسابات) ويعمل ضبط مصنع قدامك.")})
		} else {
			e.add(Check{ID: "accounts", Category: inspect.CatOwner, Title: e.t("Accounts & reset lock (FRP)", "الحسابات وقفل إعادة الضبط (FRP)"), Status: pass,
				Summary: e.t("No Google or manufacturer accounts — no reset lock.", "مفيش حساب جوجل أو حساب شركة — مفيش قفل.")})
		}
	} else {
		e.add(Check{ID: "accounts", Category: inspect.CatOwner, Title: e.t("Accounts & reset lock (FRP)", "الحسابات وقفل إعادة الضبط (FRP)"), Status: warn, Severity: major,
			Summary: e.t("Accounts could not be read.", "ماقدرناش نقرا الحسابات."),
			Advice:  e.t("Check Settings › Accounts manually — it must be empty.", "افتح الإعدادات › الحسابات بنفسك — لازم تكون فاضية.")})
	}

	// MDM
	if e.s.PolicyRead {
		switch {
		case e.s.DeviceOwner != "":
			e.add(Check{ID: "mdm", Category: inspect.CatOwner, Title: e.t("Device management (MDM)", "إدارة الجهاز عن بُعد (MDM)"), Status: fail, Severity: critical, Actual: e.s.DeviceOwner,
				Summary: e.t("The phone is managed by a company or school (device owner) — it can be locked or wiped remotely.", "الموبايل تابع لشركة أو مدرسة — ممكن يتقفل أو يتمسح من بعيد."),
				Advice:  e.t("Only buy with written release from the organisation.", "ماتشتريش إلا بخطاب رسمي من الشركة.")})
		case len(e.s.ProfileOwners) > 0:
			e.add(Check{ID: "mdm", Category: inspect.CatOwner, Title: e.t("Device management (MDM)", "إدارة الجهاز عن بُعد (MDM)"), Status: warn, Severity: major, Actual: strings.Join(e.s.ProfileOwners, ", "),
				Summary: e.t("A work profile or management app is installed.", "فيه بروفايل شغل أو برنامج إدارة متسطّب.")})
		default:
			e.add(Check{ID: "mdm", Category: inspect.CatOwner, Title: e.t("Device management (MDM)", "إدارة الجهاز عن بُعد (MDM)"), Status: pass,
				Summary: e.t("Not managed by any company.", "الموبايل مش تابع لأي شركة.")})
		}
	}

	// Samsung Knox Guard (financed / remotely lockable)
	if kg := e.p("knox.kg.state"); kg != "" {
		switch strings.ToLower(kg) {
		case "active", "locked", "prenormal", "checking", "blink":
			e.add(Check{ID: "knox-guard", Category: inspect.CatOwner, Title: e.t("Samsung Knox Guard", "قفل Knox Guard من سامسونج"), Status: fail, Severity: critical, Actual: kg,
				Summary: e.t("Knox Guard is active — the phone is financed/leased and can be locked remotely by the lender.", "Knox Guard شغّال — الموبايل متقسّط أو متأجّر ويقدر صاحب القسط يقفله من بعيد.")})
		default:
			e.add(Check{ID: "knox-guard", Category: inspect.CatOwner, Title: e.t("Samsung Knox Guard", "قفل Knox Guard من سامسونج"), Status: pass, Actual: kg,
				Summary: e.t("No active Knox Guard lock.", "مفيش قفل Knox Guard.")})
		}
	}

	// Carrier lock
	sim := strings.ToUpper(e.p("gsm.sim.state"))
	if strings.Contains(sim, "NETWORK_LOCKED") || strings.Contains(sim, "PERSO_LOCKED") {
		e.add(Check{ID: "carrier-lock", Category: inspect.CatOwner, Title: e.t("Carrier lock", "قفل الشبكة"), Status: fail, Severity: major, Actual: sim,
			Summary: e.t("The phone is locked to a carrier and rejects this SIM.", "الموبايل مقفول على شركة اتصالات ومش قابل الشريحة دي.")})
	}
}

// ---------------------------------------------------------------- software

var rootPackages = map[string]string{
	"com.topjohnwu.magisk": "Magisk", "io.github.vvb2060.magisk": "Magisk Alpha", "io.github.huskydg.magisk": "Magisk Delta",
	"me.weishu.kernelsu": "KernelSU", "me.bmax.apatch": "APatch", "eu.chainfire.supersu": "SuperSU",
	"com.koushikdutta.superuser": "Superuser", "com.noshufou.android.su": "Superuser", "com.kingroot.kinguser": "KingRoot",
	"com.kingo.root": "KingoRoot", "com.zhiqupk.root.global": "One Click Root", "org.lsposed.manager": "LSPosed",
	"de.robv.android.xposed.installer": "Xposed", "org.meowcat.edxposed.manager": "EdXposed",
	"com.chelpus.lackypatch": "Lucky Patcher", "com.dimonvideo.luckypatcher": "Lucky Patcher",
	"com.android.vending.billing.InAppBillingService.LUCK": "Lucky Patcher", "com.forpda.lp": "Lucky Patcher",
}

var storeInstallers = map[string]bool{
	"com.android.vending": true, "com.sec.android.app.samsungapps": true, "com.huawei.appmarket": true,
	"com.xiaomi.market": true, "com.xiaomi.mipicks": true, "com.heytap.market": true, "com.oppo.market": true,
	"com.bbk.appstore": true, "com.amazon.venezia": true, "com.hihonor.appmarket": true, "com.google.android.packageinstaller": false,
}

func (e *ev) softwareChecks() {
	// Bootloader / verified boot
	vbs := strings.ToLower(e.p("ro.boot.verifiedbootstate"))
	flash := e.p("ro.boot.flash.locked")
	vbdev := strings.ToLower(e.p("ro.boot.vbmeta.device_state"))
	act := strings.Trim(strings.Join([]string{vbs, "flash.locked=" + flash, vbdev}, " "), " ")
	switch {
	case vbs == "red":
		e.add(Check{ID: "bootloader", Category: inspect.CatSoftware, Title: e.t("Bootloader & verified boot", "البوت لودر والتحقق من النظام"), Status: fail, Severity: critical, Actual: act,
			Summary: e.t("Verified boot failed — the system has been modified.", "التحقق من النظام فشل — النظام متعدّل.")})
	case vbs == "orange" || flash == "0" || vbdev == "unlocked":
		e.add(Check{ID: "bootloader", Category: inspect.CatSoftware, Title: e.t("Bootloader & verified boot", "البوت لودر والتحقق من النظام"), Status: fail, Severity: major, Actual: act,
			Summary: e.t("The bootloader is unlocked — the system can be modified and banking/pay apps may refuse to work.", "البوت لودر مفتوح — النظام ممكن يتعدّل، وتطبيقات البنوك والدفع ممكن ماتشتغلش."),
			Advice:  e.t("Ask the seller to relock the bootloader with the official firmware, or treat the phone as modified.", "اطلب من البائع يقفل البوت لودر بالسوفتوير الرسمي، أو اعتبر الموبايل متعدّل.")})
	case vbs == "yellow":
		e.add(Check{ID: "bootloader", Category: inspect.CatSoftware, Title: e.t("Bootloader & verified boot", "البوت لودر والتحقق من النظام"), Status: warn, Severity: major, Actual: act,
			Summary: e.t("Verified boot uses a custom key (yellow state).", "التحقق من النظام بمفتاح غير رسمي.")})
	case vbs == "green":
		e.add(Check{ID: "bootloader", Category: inspect.CatSoftware, Title: e.t("Bootloader & verified boot", "البوت لودر والتحقق من النظام"), Status: pass, Actual: act,
			Summary: e.t("Bootloader locked and verified boot is green — official, untouched system.", "البوت لودر مقفول والنظام رسمي ومش متعدّل.")})
	}

	// Root
	var rootSigns []string
	rootSigns = append(rootSigns, e.s.SuPaths...)
	if e.s.AppsScanned {
		for _, a := range e.s.Apps {
			if n, ok := rootPackages[a.BundleID]; ok {
				rootSigns = append(rootSigns, n+" ("+a.BundleID+")")
			}
		}
	}
	if len(rootSigns) > 0 {
		e.add(Check{ID: "root", Category: inspect.CatSoftware, Title: e.t("Root", "الروت"), Status: fail, Severity: critical, Actual: strings.Join(rootSigns, ", "),
			Summary: e.t("The phone is rooted — the system is modified and readings can be faked.", "الموبايل معمول له روت — النظام متعدّل والبيانات ممكن تتزيّف."),
			Advice:  e.t("Ask for the official firmware to be flashed and the phone reset, then scan again.", "اطلب تسطيب السوفتوير الرسمي وعمل ضبط مصنع، وبعدين افحصه تاني.")})
	} else {
		e.add(Check{ID: "root", Category: inspect.CatSoftware, Title: e.t("Root", "الروت"), Status: pass,
			Summary: e.t("No root tools found.", "مفيش أي أدوات روت.")})
	}

	// SELinux
	if se := strings.ToLower(e.s.SELinux); se != "" {
		if se == "enforcing" {
			e.add(Check{ID: "selinux", Category: inspect.CatSoftware, Title: e.t("Security policy (SELinux)", "سياسة الأمان (SELinux)"), Status: pass, Actual: e.s.SELinux,
				Summary: e.t("Security policy is enforced.", "سياسة الأمان شغّالة.")})
		} else {
			e.add(Check{ID: "selinux", Category: inspect.CatSoftware, Title: e.t("Security policy (SELinux)", "سياسة الأمان (SELinux)"), Status: fail, Severity: major, Actual: e.s.SELinux,
				Summary: e.t("Security policy is disabled — only happens on modified systems.", "سياسة الأمان مقفولة — ده مابيحصلش غير في الأنظمة المعدّلة.")})
		}
	}

	// Official build
	btype, tags := e.p("ro.build.type"), e.p("ro.build.tags")
	custom := []string{}
	for _, k := range []string{"ro.lineage.version", "ro.modversion", "ro.crdroid.version", "ro.evolution.version", "ro.pixelexperience.version", "ro.derp.version"} {
		if v := e.p(k); v != "" {
			custom = append(custom, k+"="+v)
		}
	}
	disp := strings.ToLower(e.p("ro.build.display.id"))
	for _, w := range []string{"lineage", "crdroid", "evolution", "pixelexperience", "arrow", "derpfest"} {
		if strings.Contains(disp, w) {
			custom = append(custom, "display="+disp)
			break
		}
	}
	switch {
	case btype != "" && btype != "user" || strings.Contains(tags, "test-keys"):
		e.add(Check{ID: "build", Category: inspect.CatSoftware, Title: e.t("Official firmware", "سوفتوير رسمي"), Status: fail, Severity: major, Actual: btype + " / " + tags,
			Summary: e.t("Not an official retail build (developer/test firmware).", "مش سوفتوير رسمي للبيع (نسخة تجريبية أو للمطورين).")})
	case len(custom) > 0:
		e.add(Check{ID: "build", Category: inspect.CatSoftware, Title: e.t("Official firmware", "سوفتوير رسمي"), Status: warn, Severity: major, Actual: strings.Join(custom, ", "),
			Summary: e.t("A custom ROM is installed instead of the manufacturer's software.", "متسطّب روم معدّل بدل سوفتوير الشركة.")})
	default:
		e.add(Check{ID: "build", Category: inspect.CatSoftware, Title: e.t("Official firmware", "سوفتوير رسمي"), Status: pass, Actual: firstNonEmpty(e.p("ro.build.display.id"), btype+" / "+tags),
			Summary: e.t("Official manufacturer firmware (release keys).", "سوفتوير الشركة الرسمي.")})
	}

	// Samsung Knox warranty bit
	if wb := firstNonEmpty(e.p("ro.boot.warranty_bit"), e.p("ro.warranty_bit")); wb != "" {
		if wb == "0" || strings.EqualFold(wb, "0x0") {
			e.add(Check{ID: "knox-warranty", Category: inspect.CatSoftware, Title: e.t("Samsung Knox warranty bit", "علامة ضمان Knox"), Status: pass, Actual: wb,
				Summary: e.t("Knox never tripped — the phone was never rooted or flashed with unofficial software.", "Knox سليم — الموبايل ما اتعملوش روت ولا اتسطّب عليه سوفتوير غير رسمي.")})
		} else {
			e.add(Check{ID: "knox-warranty", Category: inspect.CatSoftware, Title: e.t("Samsung Knox warranty bit", "علامة ضمان Knox"), Status: fail, Severity: major, Actual: wb,
				Summary: e.t("Knox is tripped — the phone was rooted or flashed before. Samsung Pay, Secure Folder and some banking apps are permanently disabled.", "Knox اتكسر — الموبايل اتعمله روت أو اتسطّب عليه سوفتوير غير رسمي قبل كده. Samsung Pay والمجلد الآمن وبعض تطبيقات البنوك مش هتشتغل تاني.")})
		}
	}

	// Security patch age
	if sp := e.p("ro.build.version.security_patch"); sp != "" {
		if t, err := time.Parse("2006-01-02", sp); err == nil {
			months := int(e.now.Sub(t).Hours() / 24 / 30)
			c := Check{ID: "patch", Category: inspect.CatSoftware, Title: e.t("Security updates", "تحديثات الأمان"), Actual: sp}
			switch {
			case months <= 6:
				c.Status, c.Summary = pass, e.t("Security patches are recent.", "تحديثات الأمان حديثة.")
			case months <= 18:
				c.Status, c.Severity, c.Summary = warn, minor, e.t(fmt.Sprintf("Last security update is %d months old.", months), fmt.Sprintf("آخر تحديث أمان من %d شهر.", months))
				c.Advice = e.t("Install pending updates (Settings › Software update).", "نزّل التحديثات (الإعدادات › تحديث البرامج).")
			default:
				c.Status, c.Severity, c.Summary = warn, major, e.t(fmt.Sprintf("Security updates are %d months old — the phone may no longer be supported.", months), fmt.Sprintf("تحديثات الأمان قديمة من %d شهر — ممكن الموبايل مابقاش بيتحدّث.", months))
			}
			e.add(c)
		}
	}

	// Apps from outside the store
	if e.s.AppsScanned {
		var side []string
		for _, a := range e.s.Apps {
			if a.Type == "User" && !storeInstallers[a.Installer] {
				if _, isRoot := rootPackages[a.BundleID]; !isRoot {
					side = append(side, a.BundleID)
				}
			}
		}
		if len(side) > 0 {
			shown := side
			if len(shown) > 8 {
				shown = shown[:8]
			}
			e.add(Check{ID: "sideload", Category: inspect.CatSoftware, Title: e.t("Apps from outside the store", "تطبيقات من خارج المتجر"), Status: warn, Severity: minor, Actual: strings.Join(shown, ", "),
				Summary: e.t(fmt.Sprintf("%d app(s) were installed from outside an official store.", len(side)), fmt.Sprintf("فيه %d تطبيق متسطّب من برّه المتجر الرسمي.", len(side))),
				Advice:  e.t("A factory reset removes them. Unknown apps can include spyware.", "ضبط المصنع بيمسحهم. التطبيقات المجهولة ممكن يكون فيها برامج تجسس.")})
		}
	}

	// USB debugging reminder
	e.add(Check{ID: "usb-debugging", Category: inspect.CatSoftware, Title: e.t("USB debugging", "تصحيح USB"), Status: info,
		Summary: e.t("USB debugging was enabled for this inspection. Turn it off afterwards (Settings › Developer options).", "تصحيح USB اتفتح علشان الفحص. اقفله بعد ما تخلص (الإعدادات › خيارات المطور).")})
}

// ---------------------------------------------------------------- battery

func atoi(s string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	return v, err == nil
}

func (e *ev) batteryChecks() inspect.BatteryReport {
	b := inspect.BatteryReport{}
	bat, sys := e.s.Battery, e.s.BatterySys
	if len(bat) == 0 && len(sys) == 0 {
		e.add(Check{ID: "battery", Category: inspect.CatBattery, Title: e.t("Battery", "البطارية"), Status: skip,
			Summary: e.t("Battery data could not be read.", "ماقدرناش نقرا البطارية.")})
		return b
	}
	b.Available = true
	if v, ok := atoi(bat["level"]); ok {
		b.ChargePercent = v
	}
	if v, ok := atoi(bat["temperature"]); ok {
		b.TemperatureC = float64(v) / 10
	}
	if v, ok := atoi(bat["voltage"]); ok {
		b.VoltageV = float64(v) / 1000
	}
	b.IsCharging = bat["status"] == "2"
	b.ExternalConnected = bat["AC powered"] == "true" || bat["USB powered"] == "true" || bat["Wireless powered"] == "true"
	b.RatedCycles = 800
	if e.spec != nil {
		b.ExpectedDesign = append([]int{}, e.spec.Battery.DesignCapacityMah...)
		if e.spec.Battery.TypicalMah > 0 {
			b.ExpectedDesign = append(b.ExpectedDesign, e.spec.Battery.TypicalMah)
		}
		if e.spec.Battery.RatedCycles > 0 {
			b.RatedCycles = e.spec.Battery.RatedCycles
		}
	}
	// Newer Android releases also report these in "dumpsys battery".
	for k, v := range bat {
		lk := strings.ToLower(k)
		switch {
		case strings.Contains(lk, "cycle") && sys["cycle_count"] == "" && sys["battery_cycle"] == "":
			if n, ok := atoi(v); ok {
				if sys == nil {
					sys = map[string]string{}
				}
				sys["cycle_count"] = strconv.Itoa(n)
			}
		case (strings.Contains(lk, "state of health") || strings.Contains(lk, "health percentage") || lk == "battery health") && sys["fg_asoc"] == "":
			if n, ok := atoi(strings.TrimSuffix(v, "%")); ok && n > 0 && n <= 110 {
				if sys == nil {
					sys = map[string]string{}
				}
				sys["fg_asoc"] = strconv.Itoa(n)
			}
		}
	}
	if v, ok := atoi(firstNonEmpty(sys["cycle_count"], sys["battery_cycle"])); ok && v >= 0 {
		if v > 20000 {
			v /= 100
		}
		b.CycleCount, b.CyclesKnown = v, true
	}
	if v, ok := atoi(sys["charge_full_design"]); ok && v > 0 {
		b.DesignCapacity = v / 1000
	}
	if v, ok := atoi(sys["charge_full"]); ok && v > 0 {
		b.FullChargeCap = v / 1000
	}
	if v, ok := atoi(sys["fg_asoc"]); ok && v > 0 && v <= 110 {
		b.HealthPercent, b.HealthSource = math.Min(100, float64(v)), "fuel gauge (state of health)"
	} else if b.DesignCapacity > 0 && b.FullChargeCap > 0 {
		b.HealthPercent = math.Min(100, math.Round(float64(b.FullChargeCap)/float64(b.DesignCapacity)*1000)/10)
		b.HealthSource = "full charge ÷ design capacity"
	}

	// Android health code
	switch bat["health"] {
	case "2":
		e.add(Check{ID: "battery-state", Category: inspect.CatBattery, Title: e.t("Battery condition", "حالة البطارية"), Status: pass,
			Summary: e.t("The phone reports the battery as good.", "الموبايل بيقول إن البطارية كويسة.")})
	case "3":
		e.add(Check{ID: "battery-state", Category: inspect.CatBattery, Title: e.t("Battery condition", "حالة البطارية"), Status: warn, Severity: major,
			Summary: e.t("Battery is overheating.", "البطارية سخنة زيادة.")})
	case "4", "5", "6":
		e.add(Check{ID: "battery-state", Category: inspect.CatBattery, Title: e.t("Battery condition", "حالة البطارية"), Status: fail, Severity: major, Actual: bat["health"],
			Summary: e.t("The phone reports the battery as dead or faulty.", "الموبايل بيقول إن البطارية بايظة أو فيها عطل."),
			Advice:  e.t("Battery replacement needed — factor it into the price.", "محتاجة تتغيّر — احسبها من السعر.")})
	case "7":
		e.add(Check{ID: "battery-state", Category: inspect.CatBattery, Title: e.t("Battery condition", "حالة البطارية"), Status: warn, Severity: minor,
			Summary: e.t("Battery is too cold.", "البطارية باردة جداً.")})
	}

	// Health %
	if h := b.HealthPercent; h > 0 {
		c := Check{ID: "battery-health", Category: inspect.CatBattery, Title: e.t("Battery health", "صحة البطارية"), Actual: fmt.Sprintf("%.0f%%", h)}
		switch {
		case h >= 85:
			c.Status, c.Summary = pass, e.t(fmt.Sprintf("Capacity %.0f%% — good condition.", h), fmt.Sprintf("السعة %.0f%% — حالة كويسة.", h))
		case h >= 80:
			c.Status, c.Severity, c.Summary = warn, major, e.t(fmt.Sprintf("Capacity %.0f%% — worn, replacement soon.", h), fmt.Sprintf("السعة %.0f%% — مستهلكة وهتحتاج تتغيّر قريب.", h))
		default:
			c.Status, c.Severity, c.Summary = fail, major, e.t(fmt.Sprintf("Capacity %.0f%% — below 80%%, needs replacement.", h), fmt.Sprintf("السعة %.0f%% — أقل من 80%%، محتاجة تتغيّر.", h))
		}
		e.add(c)
	} else {
		e.add(Check{ID: "battery-health", Category: inspect.CatBattery, Title: e.t("Battery health", "صحة البطارية"), Status: skip,
			Summary: e.t("This phone does not share its battery capacity over USB.", "الموبايل ده مش بيدّي سعة البطارية بالكابل."),
			Advice:  e.t("Samsung: Samsung Members › Diagnostics › Battery status. Pixel: Settings › Battery › Battery health. Xiaomi: Settings › Battery.", "سامسونج: Samsung Members › التشخيص › حالة البطارية. بيكسل: الإعدادات › البطارية › صحة البطارية. شاومي: الإعدادات › البطارية.")})
	}

	// Cycles
	if b.CyclesKnown {
		ratio := float64(b.CycleCount) / float64(b.RatedCycles)
		c := Check{ID: "battery-cycles", Category: inspect.CatBattery, Title: e.t("Charge cycles", "دورات الشحن"), Actual: fmt.Sprintf("%d", b.CycleCount)}
		switch {
		case ratio < 0.6:
			c.Status, c.Summary = pass, e.t(fmt.Sprintf("%d cycles — %.0f%% of rated life used.", b.CycleCount, ratio*100), fmt.Sprintf("%d دورة شحن — اتستهلك %.0f%% من عمر البطارية.", b.CycleCount, ratio*100))
		case ratio < 1:
			c.Status, c.Severity, c.Summary = warn, minor, e.t(fmt.Sprintf("%d cycles — %.0f%% of rated life used.", b.CycleCount, ratio*100), fmt.Sprintf("%d دورة شحن — اتستهلك %.0f%% من عمر البطارية.", b.CycleCount, ratio*100))
		default:
			c.Status, c.Severity, c.Summary = warn, major, e.t(fmt.Sprintf("%d cycles — beyond the rated battery life.", b.CycleCount), fmt.Sprintf("%d دورة شحن — أكتر من العمر الرسمي للبطارية.", b.CycleCount))
		}
		e.add(c)
	}

	// Original battery
	if b.DesignCapacity > 0 && len(b.ExpectedDesign) > 0 {
		match := false
		for _, x := range b.ExpectedDesign {
			if math.Abs(float64(b.DesignCapacity-x))/float64(x) <= 0.07 {
				match = true
			}
		}
		exp := joinInts(b.ExpectedDesign, " mAh")
		act := fmt.Sprintf("%d mAh", b.DesignCapacity)
		if match {
			e.add(Check{ID: "battery-original", Category: inspect.CatBattery, Title: e.t("Battery matches factory spec", "البطارية أصلية (زي المصنع)"), Status: pass, Expected: exp, Actual: act,
				Summary: e.t("Design capacity matches the original battery.", "سعة البطارية مطابقة لبطارية المصنع.")})
		} else {
			e.add(Check{ID: "battery-original", Category: inspect.CatBattery, Title: e.t("Battery matches factory spec", "البطارية أصلية (زي المصنع)"), Status: warn, Severity: major, Expected: exp, Actual: act,
				Summary: e.t("Design capacity differs from the factory battery — likely a replacement battery.", "سعة البطارية مختلفة عن بطارية المصنع — غالباً بطارية متغيّرة.")})
		}
	}

	// Temperature
	if b.TemperatureC > 0 {
		c := Check{ID: "battery-temp", Category: inspect.CatBattery, Title: e.t("Battery temperature", "حرارة البطارية"), Actual: fmt.Sprintf("%.1f °C", b.TemperatureC)}
		switch {
		case b.TemperatureC > 45:
			c.Status, c.Severity, c.Summary = warn, major, e.t("Battery is running hot.", "البطارية سخنة جداً.")
		case b.TemperatureC > 38:
			c.Status, c.Severity, c.Summary = warn, minor, e.t("Battery is warm (charging or heavy use?).", "البطارية دافية (ممكن من الشحن).")
		default:
			c.Status, c.Summary = pass, e.t("Normal temperature.", "الحرارة طبيعية.")
		}
		e.add(c)
	}
	return b
}

// ---------------------------------------------------------------- hardware

var requiredSensors = []struct{ key, en, ar string }{
	{"accelerometer", "accelerometer", "حساس الحركة"},
	{"gyroscope", "gyroscope", "الجيروسكوب"},
	{"magnetic_field", "compass", "البوصلة"},
	{"proximity", "proximity", "حساس القرب"},
	{"light", "light", "حساس الإضاءة"},
}

func (e *ev) hardwareChecks() {
	if e.s.SensorsRead {
		var missing []string
		for _, r := range requiredSensors {
			if !contains(e.s.Sensors, r.key) {
				missing = append(missing, e.t(r.en, r.ar))
			}
		}
		if len(missing) > 0 {
			e.add(Check{ID: "sensors", Category: inspect.CatHardware, Title: e.t("Sensors", "الحساسات"), Status: fail, Severity: major, Actual: strings.Join(missing, ", "),
				Summary: e.t("Some sensors are missing — a part is broken or disconnected (often after a screen/flex repair).", "فيه حساسات مش موجودة — قطعة بايظة أو مفصولة (غالباً بعد تغيير شاشة أو فلاتة)."),
				Advice:  e.t("Test auto-rotate, screen-off during calls and auto-brightness.", "جرّب لف الشاشة، وإن الشاشة تطفي في المكالمة، والإضاءة التلقائية.")})
		} else {
			e.add(Check{ID: "sensors", Category: inspect.CatHardware, Title: e.t("Sensors", "الحساسات"), Status: pass, Actual: fmt.Sprintf("%d", len(e.s.Sensors)),
				Summary: e.t("Motion, compass, proximity and light sensors all respond.", "حساسات الحركة والبوصلة والقرب والإضاءة كلها شغّالة.")})
		}
	}
	if e.s.CamerasRead {
		if e.s.CameraCount == 0 {
			e.add(Check{ID: "cameras", Category: inspect.CatHardware, Title: e.t("Cameras", "الكاميرات"), Status: fail, Severity: major,
				Summary: e.t("No camera was detected — the camera module is faulty or disconnected.", "مفيش كاميرا ظاهرة — الكاميرا بايظة أو مفصولة.")})
		} else {
			e.add(Check{ID: "cameras", Category: inspect.CatHardware, Title: e.t("Cameras", "الكاميرات"), Status: pass, Actual: fmt.Sprintf("%d", e.s.CameraCount),
				Summary: e.t(fmt.Sprintf("%d camera(s) detected by the system.", e.s.CameraCount), fmt.Sprintf("النظام شايف %d كاميرا.", e.s.CameraCount))})
		}
	}
	if bb := e.p("gsm.version.baseband"); bb != "" && bb != "unknown" {
		e.add(Check{ID: "baseband", Category: inspect.CatHardware, Title: e.t("Cellular modem", "شريحة الشبكة (المودم)"), Status: pass, Actual: bb,
			Summary: e.t("Modem firmware is loaded and responding.", "شريحة الشبكة شغّالة.")})
	} else if len(e.s.Props) > 0 {
		e.add(Check{ID: "baseband", Category: inspect.CatHardware, Title: e.t("Cellular modem", "شريحة الشبكة (المودم)"), Status: fail, Severity: major,
			Summary: e.t("No modem firmware reported — calls and mobile data may not work.", "شريحة الشبكة مش بترد — المكالمات والنت ممكن مايشتغلوش.")})
	}
	if e.s.DropboxRead {
		panics := e.s.Dropbox["SYSTEM_LAST_KMSG"] + e.s.Dropbox["kernel_panic"] + e.s.Dropbox["SYSTEM_RECOVERY_KMSG"]
		crashes := e.s.Dropbox["system_server_crash"] + e.s.Dropbox["system_server_watchdog"] + e.s.Dropbox["SYSTEM_TOMBSTONE"]
		c := Check{ID: "crashes", Category: inspect.CatHardware, Title: e.t("Crash history", "سجل الأعطال"), Actual: fmt.Sprintf("panics=%d, system crashes=%d", panics, crashes)}
		switch {
		case panics >= 3:
			c.Status, c.Severity, c.Summary = warn, major, e.t(fmt.Sprintf("%d kernel panics — the phone has been restarting by itself.", panics), fmt.Sprintf("%d مرة الموبايل عمل ريستارت لوحده بسبب عطل.", panics))
		case panics > 0 || crashes >= 10:
			c.Status, c.Severity, c.Summary = warn, minor, e.t("A few crashes were recorded recently.", "فيه شوية أعطال متسجّلة مؤخراً.")
		default:
			c.Status, c.Summary = pass, e.t("No serious crashes recorded.", "مفيش أعطال خطيرة متسجّلة.")
		}
		e.add(c)
	}
	sim := strings.ToUpper(e.p("gsm.sim.state"))
	if sim != "" && !strings.Contains(sim, "LOCKED") {
		c := Check{ID: "sim", Category: inspect.CatHardware, Title: e.t("SIM", "الشريحة"), Status: inspect.Info, Actual: sim}
		if strings.Contains(sim, "READY") || strings.Contains(sim, "LOADED") {
			c.Summary = e.t("A SIM is inserted and ready.", "الشريحة موجودة وشغّالة.")
		} else {
			c.Summary = e.t("No SIM inserted — insert yours to test calls and data.", "مفيش شريحة — حط شريحتك وجرّب المكالمات والنت.")
		}
		e.add(c)
	}
}

// ---------------------------------------------------------------- new device

func (e *ev) newDeviceChecks(b inspect.BatteryReport) {
	if b.CyclesKnown {
		c := Check{ID: "new-cycles", Category: inspect.CatNew, Title: e.t("Battery never used", "البطارية ما اتستخدمتش"), Expected: "0 – 5", Actual: fmt.Sprintf("%d", b.CycleCount)}
		switch {
		case b.CycleCount <= 5:
			c.Status, c.Summary = pass, e.t("Battery has (almost) never been cycled — consistent with a new phone.", "البطارية تقريباً ما اتشحنتش — طبيعي لموبايل جديد.")
		case b.CycleCount <= 20:
			c.Status, c.Severity, c.Summary = warn, major, e.t("More charge cycles than a new phone should have.", "دورات شحن أكتر من موبايل جديد.")
		default:
			c.Status, c.Severity, c.Summary = fail, critical, e.t("The battery has many charge cycles — the phone has been used before.", "البطارية اتشحنت كتير — الموبايل مستعمل.")
		}
		e.add(c)
	}
	if b.HealthPercent > 0 {
		c := Check{ID: "new-health", Category: inspect.CatNew, Title: e.t("Battery at full capacity", "البطارية بكامل سعتها"), Expected: "100%", Actual: fmt.Sprintf("%.0f%%", b.HealthPercent)}
		if b.HealthPercent >= 98 {
			c.Status, c.Summary = pass, e.t("Battery is at full factory capacity.", "البطارية بكامل سعة المصنع.")
		} else {
			c.Status, c.Severity, c.Summary = fail, major, e.t("A new battery should be at 100% — this one is worn.", "البطارية الجديدة لازم تبقى 100% — دي مستهلكة.")
		}
		e.add(c)
	}
	if e.s.AppsScanned {
		var store []string
		for _, a := range e.s.Apps {
			if a.Type == "User" && a.Installer == "com.android.vending" {
				store = append(store, a.BundleID)
			}
		}
		if len(store) == 0 {
			e.add(Check{ID: "new-apps", Category: inspect.CatNew, Title: e.t("No previous owner's apps", "مفيش تطبيقات لحد قبلك"), Status: pass,
				Summary: e.t("No apps downloaded from the Play Store.", "مفيش تطبيقات متنزّلة من Play Store.")})
		} else {
			e.add(Check{ID: "new-apps", Category: inspect.CatNew, Title: e.t("No previous owner's apps", "مفيش تطبيقات لحد قبلك"), Status: warn, Severity: major, Actual: strings.Join(store, ", "),
				Summary: e.t(fmt.Sprintf("%d app(s) were downloaded from the Play Store — someone has used the phone.", len(store)), fmt.Sprintf("فيه %d تطبيق متنزّل من Play Store — حد استعمل الموبايل.", len(store))),
				Advice:  e.t("Ignore only if you installed them yourself.", "تجاهل ده بس لو إنت اللي نزّلتهم.")})
		}
	}
	if e.s.DataUsedKB > 0 {
		gb := float64(e.s.DataUsedKB) * 1024 / 1e9
		c := Check{ID: "new-data", Category: inspect.CatNew, Title: e.t("No stored user data", "مفيش بيانات لحد"), Actual: fmt.Sprintf("%.1f GB", gb)}
		if gb < 30 {
			c.Status, c.Summary = pass, e.t("Storage usage is consistent with a fresh phone.", "المساحة المستخدمة طبيعية لموبايل جديد.")
		} else {
			c.Status, c.Severity, c.Summary = warn, major, e.t("A lot of storage is already used — someone else's data.", "مساحة كبيرة مستخدمة — بيانات لحد تاني.")
		}
		e.add(c)
	}
	if e.s.DropboxRead {
		panics := e.s.Dropbox["SYSTEM_LAST_KMSG"] + e.s.Dropbox["kernel_panic"]
		if panics == 0 {
			e.add(Check{ID: "new-crashes", Category: inspect.CatNew, Title: e.t("Clean crash history", "مفيش أعطال متسجّلة"), Status: pass,
				Summary: e.t("No kernel panics recorded.", "مفيش أعطال متسجّلة.")})
		} else {
			e.add(Check{ID: "new-crashes", Category: inspect.CatNew, Title: e.t("Clean crash history", "مفيش أعطال متسجّلة"), Status: warn, Severity: major,
				Summary: e.t("Kernel panics recorded on a phone sold as new.", "فيه أعطال متسجّلة في موبايل بيتباع على إنه جديد.")})
		}
	}
}

// ---------------------------------------------------------------- manual

func (e *ev) manualChecks(manual map[string]ManualResult) {
	for _, t := range ManualTests {
		if !t.forMode(e.mode) {
			continue
		}
		m, ok := manual[t.ID]
		if !ok || (m.Status != pass && m.Status != fail && m.Status != skip) {
			continue
		}
		c := Check{ID: "manual-" + t.ID, Category: inspect.CatHands, Title: e.t(t.Title, t.TitleAr), Status: m.Status, Severity: t.Severity, Actual: m.Note}
		switch m.Status {
		case pass:
			c.Summary = e.t("Tested — OK.", "تم الفحص — سليم.")
		case fail:
			c.Summary = e.t("Tested — problem found.", "تم الفحص — فيه مشكلة.")
		default:
			c.Summary = e.t("Not tested.", "ماتفحصش.")
			c.Severity = inspect.Note
		}
		e.add(c)
	}
}

// ---------------------------------------------------------------- display

func (e *ev) storage() map[string]any {
	if e.s.DataTotalKB <= 0 {
		return nil
	}
	total := float64(e.s.DataTotalKB) * 1024
	used := float64(e.s.DataUsedKB) * 1024
	out := map[string]any{"totalBytes": total, "usedBytes": used, "totalLabel": humanBytes(total), "usedLabel": humanBytes(used)}
	if e.spec != nil {
		gb := total / 1e9
		for _, o := range e.spec.StorageOptionsGB {
			m := float64(o)
			if o >= 1024 {
				m = float64(o) / 1024 * 1000
			}
			if gb >= m*0.72 && gb <= m*1.01 {
				out["totalLabel"] = gbLabel(o)
				out["totalBytes"] = m * 1e9
				break
			}
		}
	}
	return out
}

func (e *ev) identity(model string) []inspect.KV {
	var kv []inspect.KV
	add := func(key, ar, v string, mono bool) {
		if strings.TrimSpace(v) != "" {
			kv = append(kv, inspect.KV{Key: key, Label: e.t(key, ar), Value: v, Mono: mono})
		}
	}
	add("Model", "الموديل", model, false)
	add("Brand", "الماركة", e.p("ro.product.manufacturer"), false)
	add("Model number", "رقم الموديل", e.p("ro.product.model"), true)
	add("Codename", "الاسم الكودي", e.p("ro.product.device"), true)
	if v := e.p("ro.build.version.release"); v != "" {
		add("Android", "أندرويد", v+" (SDK "+e.p("ro.build.version.sdk")+")", false)
	}
	add("Build", "رقم البناء", e.p("ro.build.display.id"), true)
	add("Security patch", "تحديث الأمان", e.p("ro.build.version.security_patch"), false)
	chip := strings.Trim(e.p("ro.board.platform")+" "+e.p("ro.soc.model"), " ")
	if e.spec != nil && e.spec.Chip != "" {
		chip = e.spec.Chip + " (" + chip + ")"
	}
	add("Chip", "المعالج", chip, false)
	if e.s.MemTotalKB > 0 {
		add("RAM", "الرامات", fmt.Sprintf("%.1f GB", float64(e.s.MemTotalKB)/1024/1024), false)
	}
	if st := e.storage(); st != nil {
		add("Capacity", "المساحة", st["totalLabel"].(string), false)
	}
	add("Screen", "الشاشة", e.s.ScreenSize, true)
	add("Serial number", "الرقم التسلسلي", firstNonEmpty(e.p("ro.serialno"), e.p("ro.boot.serialno"), e.s.Serial), true)
	for i, im := range e.s.IMEIs {
		lbl := "IMEI"
		if i > 0 {
			lbl = fmt.Sprintf("IMEI %d", i+1)
		}
		add(lbl, lbl, im, true)
	}
	add("Baseband", "شريحة الشبكة", e.p("gsm.version.baseband"), true)
	add("Region / CSC", "المنطقة / CSC", firstNonEmpty(e.p("ro.csc.sales_code"), e.p("ro.boot.carrierid"), e.p("ro.miui.region")), true)
	add("Bootloader", "البوت لودر", e.p("ro.boot.verifiedbootstate"), true)
	add("Fingerprint", "بصمة النظام", e.p("ro.build.fingerprint"), true)
	return kv
}

// ---------------------------------------------------------------- helpers

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func containsFold(list []string, v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return false
	}
	for _, x := range list {
		if strings.ToLower(strings.TrimSpace(x)) == v {
			return true
		}
	}
	return false
}

func sameRes(a, b string) bool {
	pa, pb := strings.Split(strings.ToLower(a), "x"), strings.Split(strings.ToLower(b), "x")
	if len(pa) != 2 || len(pb) != 2 {
		return strings.EqualFold(a, b)
	}
	return (pa[0] == pb[0] && pa[1] == pb[1]) || (pa[0] == pb[1] && pa[1] == pb[0])
}

func majorVersion(v string) int {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, ". "); i > 0 {
		v = v[:i]
	}
	n, _ := strconv.Atoi(v)
	return n
}

func luhnValid(s string) bool {
	sum, dbl := 0, false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if dbl {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		dbl = !dbl
	}
	return len(s) > 1 && sum%10 == 0
}

func gbLabel(gb int) string {
	if gb >= 1024 && gb%1024 == 0 {
		return fmt.Sprintf("%d TB", gb/1024)
	}
	return fmt.Sprintf("%d GB", gb)
}

func joinGB(v []int) string {
	var s []string
	for _, x := range v {
		s = append(s, gbLabel(x))
	}
	return strings.Join(s, " / ")
}

func joinInts(v []int, suffix string) string {
	var s []string
	for _, x := range v {
		s = append(s, fmt.Sprintf("%d%s", x, suffix))
	}
	return strings.Join(s, " / ")
}

func joinFloats(v []float64, suffix string) string {
	var s []string
	for _, x := range v {
		s = append(s, strconv.FormatFloat(x, 'f', -1, 64)+suffix)
	}
	return strings.Join(s, " / ")
}

func humanBytes(b float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for b >= 1000 && i < len(units)-1 {
		b /= 1000
		i++
	}
	return fmt.Sprintf("%.1f %s", b, units[i])
}
