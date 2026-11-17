package android

import "github.com/mo3bdlaa/smart-mobile-check/internal/inspect"

// ManualTest is a guided hands-on test (same JSON shape as the iPhone one).
type ManualTest struct {
	ID       string           `json:"id"`
	Modes    []string         `json:"modes"`
	Severity inspect.Severity `json:"severity"`
	Link     string           `json:"link,omitempty"`
	Group    string           `json:"group"`
	Title    string           `json:"title"`
	Steps    []string         `json:"steps"`
	GroupAr  string           `json:"groupAr"`
	TitleAr  string           `json:"titleAr"`
	StepsAr  []string         `json:"stepsAr"`
}

func (t ManualTest) forMode(m string) bool {
	for _, x := range t.Modes {
		if x == m {
			return true
		}
	}
	return false
}

var (
	both     = []string{inspect.ModeUsed, inspect.ModeNew}
	usedOnly = []string{inspect.ModeUsed}
	newOnly  = []string{inspect.ModeNew}
)

// ManualTests is the Android hands-on checklist, in order.
var ManualTests = []ManualTest{
	// ---- brand-new phone
	{ID: "box-seal", Modes: newOnly, Severity: inspect.Critical,
		Group: "Box & seal", Title: "Factory seal intact",
		Steps:   []string{"The box seal stickers / shrink-wrap must be untouched and not re-glued.", "No dents, crushed corners or signs the box was opened before."},
		GroupAr: "العلبة والبرشام", TitleAr: "برشام المصنع سليم",
		StepsAr: []string{"ستيكرات البرشام أو السلوفان لازم تكون سليمة ومش متلزّقة تاني.", "العلبة مفيهاش خبطات ولا أركان مضغوطة ولا علامات إنها اتفتحت."}},
	{ID: "box-imei", Modes: newOnly, Severity: inspect.Critical,
		Group: "Box & seal", Title: "Box IMEI & model match the phone",
		Steps:   []string{"Dial *#06# on the phone.", "Compare the IMEI(s) and the model number with the label on the box and with this app's Device tab.", "FAIL if anything differs — the phone was swapped into another box."},
		GroupAr: "العلبة والبرشام", TitleAr: "IMEI والموديل على العلبة نفس الموبايل",
		StepsAr: []string{"اطلب ‎*#06#‎ على الموبايل.", "قارن الـ IMEI ورقم الموديل باللي على ملصق العلبة وبتاب \"الجهاز والمواصفات\" في البرنامج.", "لو أي حاجة مختلفة = فشل، يعني الموبايل اتحط في علبة مش بتاعته."}},
	{ID: "welcome", Modes: newOnly, Severity: inspect.Critical,
		Group: "First start", Title: "Starts at the Welcome setup screen",
		Steps:   []string{"Power the phone on for the first time.", "It must show the Welcome / Start setup wizard. FAIL if it goes straight to a home screen."},
		GroupAr: "أول تشغيل", TitleAr: "بيفتح على شاشة الترحيب",
		StepsAr: []string{"شغّل الموبايل لأول مرة.", "لازم تظهر شاشة الترحيب وخطوات الإعداد. لو فتح على الشاشة الرئيسية على طول = فشل."}},
	{ID: "no-frp", Modes: newOnly, Severity: inspect.Critical,
		Group: "First start", Title: "No previous account during setup",
		Steps:   []string{"Go through setup.", "FAIL if it asks to \"verify the Google account that was previously synced\" or for a Samsung/Xiaomi account — return it."},
		GroupAr: "أول تشغيل", TitleAr: "مفيش حساب قديم وقت الإعداد",
		StepsAr: []string{"كمّل خطوات الإعداد.", "لو طلب \"تأكيد حساب جوجل اللي كان متزامن قبل كده\" أو حساب سامسونج/شاومي = فشل، رجّعه."}},
	{ID: "warranty", Modes: newOnly, Severity: inspect.Major,
		Group: "Warranty", Title: "Official warranty & local version",
		Steps:   []string{"Ask for the invoice and the official warranty card.", "Samsung: Samsung Members app › Support › Warranty information shows the start date.", "Make sure the phone is registered / customs paid for your country (e.g. Egypt) so the network is not blocked later."},
		GroupAr: "الضمان", TitleAr: "ضمان رسمي ونسخة البلد",
		StepsAr: []string{"اطلب الفاتورة وكارت الضمان الرسمي.", "سامسونج: تطبيق Samsung Members › الدعم › معلومات الضمان بيوضّح تاريخ البداية.", "اتأكد إن الموبايل متسجّل وجماركه مدفوعة في بلدك (زي مصر) علشان الشبكة ماتتقفلش بعدين."}},
	{ID: "pristine", Modes: newOnly, Severity: inspect.Minor,
		Group: "Physical condition", Title: "Pristine body, complete box",
		Steps:   []string{"No scratches on the frame, screen, back or cameras; protective films in place.", "Box contents complete (cable, SIM tool, documents, S Pen if applicable)."},
		GroupAr: "الشكل", TitleAr: "الموبايل نضيف والعلبة كاملة",
		StepsAr: []string{"مفيش أي خدش على الإطار أو الشاشة أو الضهر أو الكاميرات، والاستيكرات الحماية موجودة.", "محتويات العلبة كاملة (الكابل، دبوس الشريحة، الورق، والقلم لو موجود)."}},

	// ---- used phone: ownership
	{ID: "accounts-removed", Modes: usedOnly, Severity: inspect.Critical,
		Group: "Ownership", Title: "All accounts removed & factory reset",
		Steps:   []string{"Settings › Accounts must be empty (no Google, Samsung, Xiaomi account).", "Ask the seller to factory-reset the phone in front of you.", "After the reset, setup must NOT ask for the previous owner's account (FRP)."},
		GroupAr: "الملكية", TitleAr: "الحسابات اتمسحت واتعمل ضبط مصنع",
		StepsAr: []string{"الإعدادات › الحسابات لازم تكون فاضية (مفيش حساب جوجل أو سامسونج أو شاومي).", "خلّي البائع يعمل ضبط مصنع قدامك.", "بعد الضبط، الإعداد لازم ما يطلبش حساب صاحبه القديم (FRP)."}},
	{ID: "imei-match", Modes: usedOnly, Severity: inspect.Major,
		Group: "Ownership", Title: "IMEI matches box & is clean",
		Steps:   []string{"Dial *#06# and compare with the box / SIM tray.", "Check the IMEI is not reported lost/stolen and that the phone is registered for your country."},
		GroupAr: "الملكية", TitleAr: "الـ IMEI مطابق ونضيف",
		StepsAr: []string{"اطلب ‎*#06#‎ وقارن باللي على العلبة أو درج الشريحة.", "اتأكد إن الـ IMEI مش مبلّغ عنه مسروق أو ضايع، وإن الموبايل متسجّل في بلدك."}},

	// ---- functional tests (both)
	{ID: "diagnostics", Modes: both, Severity: inspect.Major,
		Group: "Built-in diagnostics", Title: "Manufacturer diagnostics",
		Steps:   []string{"Samsung: open Samsung Members › Support › Phone diagnostics › Test all. (Or dial *#0*# for the test menu.)", "Pixel: Settings › System › Diagnostics (or the Pixel Diagnostic app).", "Xiaomi: Settings › My device › All specs › tap \"Kernel version\" 5 times (CIT menu).", "FAIL if any test fails."},
		GroupAr: "فحص الشركة", TitleAr: "الفحص الداخلي من الشركة",
		StepsAr: []string{"سامسونج: افتح Samsung Members › الدعم › تشخيص الهاتف › اختبار الكل. (أو اطلب ‎*#0*#‎ لقائمة الاختبار.)", "بيكسل: الإعدادات › النظام › التشخيص.", "شاومي: الإعدادات › جهازي › كل المواصفات › دوس 5 مرات على \"إصدار النواة\".", "لو أي اختبار فشل = فشل."}},
	{ID: "battery-screen", Modes: usedOnly, Severity: inspect.Major,
		Group: "Battery", Title: "Battery status in Settings",
		Steps:   []string{"Samsung: Samsung Members › Diagnostics › Battery status (shows Normal/Weak and capacity).", "Pixel: Settings › Battery › Battery health.", "FAIL if it says Weak / Bad / needs service."},
		GroupAr: "البطارية", TitleAr: "حالة البطارية في الإعدادات",
		StepsAr: []string{"سامسونج: Samsung Members › التشخيص › حالة البطارية (بيقول طبيعي أو ضعيف).", "بيكسل: الإعدادات › البطارية › صحة البطارية.", "لو مكتوب ضعيفة أو محتاجة صيانة = فشل."}},
	{ID: "display", Modes: both, Severity: inspect.Major,
		Group: "Display", Title: "Screen: dead pixels, burn-in & tint",
		Steps:   []string{"Max brightness. Show a white, black, red, green and blue full-screen image.", "Look for dead pixels, green/pink lines, burn-in ghosts (keyboard, status bar) or yellow tint — common on AMOLED screens."},
		GroupAr: "الشاشة", TitleAr: "الشاشة: نقط ميتة وحرق وألوان",
		StepsAr: []string{"علّي الإضاءة للآخر. اعرض صورة بيضا وسودا وحمرا وخضرا وزرقا مالية الشاشة.", "دوّر على نقط ميتة، أو خطوط خضرا/بمبي، أو أثر كيبورد محروق، أو لون مصفر — منتشرة في شاشات AMOLED."}},
	{ID: "touch", Modes: both, Severity: inspect.Major,
		Group: "Display", Title: "Touch screen",
		Steps:   []string{"Drag an app icon slowly to every corner and edge.", "FAIL if it drops or jumps anywhere."},
		GroupAr: "الشاشة", TitleAr: "اللمس",
		StepsAr: []string{"اسحب أيقونة بالراحة لكل الأركان والحواف.", "لو وقعت أو نطّت في أي مكان = فشل."}},
	{ID: "biometrics", Modes: both, Severity: inspect.Major,
		Group: "Biometrics", Title: "Fingerprint & face unlock",
		Steps:   []string{"Register a fingerprint and unlock with it several times.", "Set up face unlock if available.", "An in-display fingerprint that fails often can mean a non-original screen."},
		GroupAr: "البصمة", TitleAr: "بصمة الإصبع والوجه",
		StepsAr: []string{"سجّل بصمة وافتح بيها كذا مرة.", "جرّب بصمة الوجه لو موجودة.", "لو بصمة الشاشة بتفشل كتير ممكن الشاشة مش أصلية."}},
	{ID: "cameras", Modes: both, Severity: inspect.Major,
		Group: "Cameras", Title: "All cameras, zoom & flash",
		Steps:   []string{"Switch through every lens (ultra-wide, main, telephoto/zoom) and the front camera.", "Check focus, video, flash, and look for dust or spots inside lenses."},
		GroupAr: "الكاميرات", TitleAr: "كل الكاميرات والزوم والفلاش",
		StepsAr: []string{"جرّب كل العدسات (الواسعة والأساسية والزوم) والكاميرا الأمامية.", "جرّب الفوكس والفيديو والفلاش، وشوف لو فيه تراب أو بقع جوه العدسات."}},
	{ID: "audio", Modes: both, Severity: inspect.Major,
		Group: "Audio", Title: "Speakers & microphones",
		Steps:   []string{"Play music at full volume (both speakers).", "Record a voice note and a video, play them back.", "Make a call — both sides must hear clearly."},
		GroupAr: "الصوت", TitleAr: "السماعات والمايكات",
		StepsAr: []string{"شغّل أغنية على أعلى صوت (السماعتين).", "سجّل فويس وفيديو واسمعهم.", "اعمل مكالمة — لازم الطرفين يسمعوا كويس."}},
	{ID: "buttons", Modes: both, Severity: inspect.Minor,
		Group: "Buttons", Title: "Buttons, vibration & S Pen",
		Steps:   []string{"Press power and volume buttons.", "Check vibration.", "If the phone has an S Pen: take it out, write and use Air actions."},
		GroupAr: "الزراير", TitleAr: "الزراير والاهتزاز والقلم",
		StepsAr: []string{"جرّب زرار الباور وزراير الصوت.", "اتأكد من الاهتزاز.", "لو فيه قلم S Pen: طلّعه واكتب بيه وجرّب الأوامر."}},
	{ID: "charging", Modes: both, Severity: inspect.Major,
		Group: "Power", Title: "Wired, fast & wireless charging",
		Steps:   []string{"Connect a charger: it must show \"Fast charging\" with a fast charger and hold the connection when wiggled.", "Try wireless charging if supported."},
		GroupAr: "الشحن", TitleAr: "الشحن العادي والسريع واللاسلكي",
		StepsAr: []string{"وصّل شاحن: لازم يكتب \"شحن سريع\" مع الشاحن السريع ومايفصلش لما تحرّك الكابل.", "جرّب الشحن اللاسلكي لو بيدعمه."}},
	{ID: "connectivity", Modes: both, Severity: inspect.Major,
		Group: "Connectivity", Title: "Calls, data, Wi-Fi, Bluetooth, GPS & NFC",
		Steps:   []string{"Insert your SIM: signal, a call and mobile data.", "Wi-Fi, Bluetooth, Maps location and NFC (if present)."},
		GroupAr: "الاتصال", TitleAr: "المكالمات والنت والواي فاي والبلوتوث والـ GPS",
		StepsAr: []string{"حط شريحتك: الشبكة ومكالمة والنت.", "الواي فاي والبلوتوث والخرائط والـ NFC (لو موجود)."}},
	{ID: "sensors", Modes: both, Severity: inspect.Minor,
		Group: "Sensors", Title: "Proximity, light & rotation",
		Steps:   []string{"During a call, cover the top of the screen: it must turn off.", "Auto-rotate and auto-brightness must work."},
		GroupAr: "الحساسات", TitleAr: "حساس القرب والإضاءة واللف",
		StepsAr: []string{"وانت في مكالمة غطّي أعلى الشاشة: لازم تطفي.", "لف الشاشة التلقائي والإضاءة التلقائية لازم يشتغلوا."}},
	{ID: "body", Modes: usedOnly, Severity: inspect.Minor,
		Group: "Physical condition", Title: "Body, frame & water damage",
		Steps:   []string{"Check frame and back glass for cracks, bends, or a lifted back (swollen battery!).", "Look at the screen edges for gaps — a sign of a cheap screen replacement."},
		GroupAr: "الشكل", TitleAr: "الجسم والإطار وآثار المية",
		StepsAr: []string{"شوف الإطار والضهر: شروخ أو انحناء أو ضهر مرفوع (بطارية منفوخة!).", "شوف حواف الشاشة لو فيها فراغات — علامة إن الشاشة اتغيّرت بشاشة رخيصة."}},
}
