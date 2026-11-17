package iphone

import (
	"regexp"
	"strings"
)

// Arabic presentation of results. The check engine produces English text;
// Localize maps it to simple Arabic (exact phrases + patterns for phrases
// that contain numbers). Anything not found stays in English, and the tests
// make sure every phrase the engine can produce has a translation.

var arExact = map[string]string{
	// ---- titles
	"Model recognised":             "الموديل معروف",
	"Model identity consistency":   "تطابق بيانات الموديل",
	"Logic board model":            "موديل اللوحة الأم",
	"Regulatory model (A-number)":  "رقم الموديل (A-number)",
	"Processor (SoC)":              "المعالج",
	"Hardware ID integrity":        "سلامة رقم تعريف الجهاز",
	"Serial number":                "الرقم التسلسلي (Serial)",
	"Production hardware":          "جهاز تجاري (مش نموذج تجريبي)",
	"Sales model (part number)":    "نوع الجهاز (رقم القطعة)",
	"Sales region":                 "منطقة البيع",
	"IMEI":                         "رقم IMEI",
	"Storage capacity":             "مساحة التخزين",
	"Activation":                   "التفعيل",
	"Find My / Activation Lock":    "Find My / قفل التفعيل",
	"Remote Management (MDM)":      "الإدارة عن بُعد (MDM)",
	"Configuration profiles":       "ملفات التعريف (Profiles)",
	"Device state":                 "حالة الجهاز",
	"Passcode":                     "رمز القفل",
	"iOS version plausibility":     "نسخة iOS مناسبة للموديل",
	"Firmware build consistency":   "تطابق نسخة النظام",
	"Release firmware":             "نسخة نظام رسمية",
	"Root filesystem service":      "خدمة ملفات الجيلبريك",
	"Jailbreak tools":              "برامج الجيلبريك",
	"Sideloading":                  "تطبيقات من خارج المتجر",
	"Developer Mode":               "وضع المطوّر",
	"Battery health":               "صحة البطارية",
	"Charge cycles":                "دورات الشحن",
	"Battery matches factory spec": "البطارية أصلية (زي المصنع)",
	"Battery temperature":          "حرارة البطارية",
	"Battery gauge identity":       "شريحة البطارية",
	"Kernel panics":                "سجل الأعطال (إعادة تشغيل مفاجئة)",
	"Cellular modem":               "شريحة الشبكة (المودم)",
	"Wi-Fi & Bluetooth":            "الواي فاي والبلوتوث",
	"Storage controller":           "وحدة التخزين",
	"SIM":                          "الشريحة (SIM)",
	"New retail unit":              "جهاز جديد من المصنع",
	"Battery never used":           "البطارية لم تُستخدم",
	"Battery at full capacity":     "البطارية بكامل سعتها",
	"No previous owner's apps":     "لا توجد تطبيقات لمالك سابق",
	"No stored user data":          "لا توجد بيانات مستخدم",
	"Clean crash history":          "لا توجد أعطال مسجّلة",

	// ---- summaries
	"Different system services report different models — the device readings have been tampered with.": "أجزاء النظام تعطي موديلات مختلفة — بيانات الجهاز متلاعب بها.",
	"Board identifier matches the model.":                                                                                 "اللوحة الأم مطابقة للموديل.",
	"Board identifier does not match the expected value for this model.":                                                  "اللوحة الأم لا تطابق هذا الموديل.",
	"A-number belongs to this model.":                                                                                     "رقم الموديل صحيح لهذا الجهاز.",
	"A-number is not one of the known numbers for this model.":                                                            "رقم الموديل ليس من أرقام هذا الجهاز.",
	"UDID is consistent with the chip ID and ECID burned into the processor.":                                             "رقم تعريف الجهاز مطابق للرقم المحفور في المعالج.",
	"UDID does not match the processor's chip ID / ECID — identity has been altered.":                                     "رقم تعريف الجهاز لا يطابق المعالج — هوية الجهاز تم تغييرها.",
	"Legacy UDID is consistent with serial number, ECID and MAC addresses.":                                               "رقم تعريف الجهاز مطابق للرقم التسلسلي والمعالج.",
	"Legacy-format UDID; cannot be fully re-derived on this model.":                                                       "جهاز قديم — لا يمكن التحقق الكامل من رقم التعريف.",
	"The device did not report a serial number.":                                                                          "الجهاز لم يُظهر رقماً تسلسلياً.",
	"Serial number has an invalid format.":                                                                                "شكل الرقم التسلسلي غير صحيح.",
	"Serial number differs between system services — readings were modified.":                                             "الرقم التسلسلي مختلف داخل النظام — البيانات متلاعب بها.",
	"Valid serial number format.":                                                                                         "الرقم التسلسلي سليم.",
	"Retail (production-fused) processor.":                                                                                "معالج تجاري عادي.",
	"Development / prototype processor — not a retail unit.":                                                              "معالج تجريبي — الجهاز ليس نسخة تجارية.",
	"New retail unit (sold new by Apple or an authorised reseller)":                                                       "جهاز جديد من المصنع (اتباع جديد من Apple أو موزّع معتمد)",
	"Refurbished unit (Apple Certified Refurbished or carrier refurbished)":                                               "جهاز مجدّد (Refurbished)",
	"Replacement unit (issued by Apple / an authorised service provider as a warranty or service replacement)":            "جهاز بديل من الصيانة (Apple أعطته بدل جهاز آخر)",
	"Personalised unit (engraved / custom order)":                                                                         "جهاز بطلب خاص (محفور عليه اسم)",
	"Demo unit (in-store display device)":                                                                                 "جهاز عرض من محل",
	"Unusual part-number prefix.":                                                                                         "رقم القطعة غير معتاد.",
	"No IMEI reported — the cellular modem may be damaged or the baseband is not responding.":                             "لا يوجد IMEI — شريحة الشبكة قد تكون تالفة.",
	"IMEI fails the checksum — it is invalid or has been altered.":                                                        "رقم IMEI غير صحيح — تم التلاعب به.",
	"Valid IMEI (checksum OK).":                                                                                           "رقم IMEI صحيح.",
	"Second IMEI fails the checksum — altered or invalid.":                                                                "رقم IMEI الثاني غير صحيح — تم التلاعب به.",
	"Reported capacity does not match any official configuration — storage may have been modified.":                       "المساحة لا تطابق أي مساحة رسمية — قد يكون التخزين معدّلاً.",
	"Device is activated.":                                                                                                "الجهاز مفعّل.",
	"Device is not activated.":                                                                                            "الجهاز غير مفعّل.",
	"Not activated yet — expected for a sealed phone. Activate it in front of the seller.":                                "لم يُفعّل بعد — طبيعي لجهاز جديد. فعّله أمام البائع.",
	"Find My is ON — the phone is locked to someone's Apple Account (Activation Lock).":                                   "Find My شغّال — الجهاز مقفول على حساب شخص آخر (قفل التفعيل).",
	"Find My is off — no Activation Lock.":                                                                                "Find My مقفول — لا يوجد قفل تفعيل.",
	"Find My status could not be read.":                                                                                   "لم نتمكن من قراءة حالة Find My.",
	"This iPhone is enrolled in an organisation's device management — it belongs to a company or school.":                 "الجهاز تابع لشركة أو مدرسة وتديره عن بُعد.",
	"Device is supervised (managed configuration).":                                                                       "الجهاز تحت إشراف (إعدادات مُدارة).",
	"Not enrolled in device management and not supervised.":                                                               "الجهاز غير تابع لأي شركة.",
	"Management status could not be read.":                                                                                "لم نتمكن من قراءة حالة الإدارة.",
	"No configuration profiles installed.":                                                                                "لا توجد ملفات تعريف.",
	"Device reports a bricked (unusable) state.":                                                                          "الجهاز في حالة عطل كامل.",
	"No passcode is set.":                                                                                                 "لا يوجد رمز قفل.",
	"A passcode is set. Ask the seller to erase the phone (Settings › General › Transfer or Reset) before handover.":      "يوجد رمز قفل. اطلب من البائع مسح الجهاز قبل الاستلام (الإعدادات › عام › نقل أو إعادة تعيين).",
	"This iOS version cannot officially run on this model — the version/model readings were spoofed.":                     "نسخة iOS هذه لا تعمل على هذا الموديل — البيانات مزوّرة.",
	"iOS version is valid for this model.":                                                                                "نسخة iOS مناسبة للموديل.",
	"System services disagree about the installed iOS build — firmware readings were modified.":                           "النظام يعطي نسخاً مختلفة — بيانات النظام متلاعب بها.",
	"A beta (pre-release) iOS build is installed.":                                                                        "مثبّت نسخة تجريبية (بيتا) من iOS.",
	"Public release build of iOS.":                                                                                        "نسخة iOS رسمية.",
	"A root-filesystem access service (afc2) is running — the device is jailbroken.":                                      "توجد خدمة جيلبريك (afc2) — الجهاز معمول له جيلبريك.",
	"No jailbreak filesystem services exposed.":                                                                           "لا توجد خدمات جيلبريك.",
	"Jailbreak / exploit tools are installed — the system has been modified.":                                             "برامج جيلبريك مثبّتة — النظام معدّل.",
	"Non-Apple apps are registered as system apps — typical of TrollStore or a jailbreak.":                                "تطبيقات ليست من Apple مسجّلة كتطبيقات نظام — علامة على TrollStore أو جيلبريك.",
	"Apps with names associated with jailbreak/sideloading tools were found.":                                             "توجد تطبيقات بأسماء برامج جيلبريك.",
	"Installed apps could not be listed.":                                                                                 "لم نتمكن من قراءة التطبيقات.",
	"Sideloading tools are installed (apps from outside the App Store).":                                                  "مثبّت برامج لتنزيل تطبيقات من خارج المتجر.",
	"Developer Mode is enabled.":                                                                                          "وضع المطوّر مفعّل.",
	"Battery gauge could not be read on this iOS version.":                                                                "لم نتمكن من قراءة البطارية في هذه النسخة.",
	"Maximum capacity not reported.":                                                                                      "سعة البطارية غير متاحة.",
	"0 cycles reported but capacity is worn — cycle counter may have been reset (battery swapped or gauge reprogrammed).": "عداد الدورات صفر لكن البطارية مستهلكة — العداد تم تصفيره (بطارية مغيّرة أو متلاعب بها).",
	"Design capacity matches the original battery for this model.":                                                        "سعة البطارية مطابقة لبطارية المصنع.",
	"Design capacity differs from the factory battery — likely a replacement or third-party battery.":                     "سعة البطارية مختلفة عن بطارية المصنع — غالباً بطارية مغيّرة أو غير أصلية.",
	"Battery is running hot.":                                                                                             "البطارية سخنة جداً.",
	"Battery is warm (charging or heavy use?).":                                                                           "البطارية دافئة (ربما بسبب الشحن).",
	"Normal temperature.":                                                                                  "الحرارة طبيعية.",
	"Battery management chip reports a serial number.":                                                     "شريحة البطارية لها رقم تسلسلي.",
	"Crash reports could not be read.":                                                                     "لم نتمكن من قراءة سجل الأعطال.",
	"Modem firmware is loaded and responding.":                                                             "شريحة الشبكة تعمل.",
	"No modem firmware reported — cellular hardware may be faulty (no service).":                           "شريحة الشبكة لا تستجيب — قد لا تعمل الشبكة.",
	"Wi-Fi and Bluetooth chips report valid hardware addresses.":                                           "الواي فاي والبلوتوث يعملان.",
	"Wi-Fi or Bluetooth hardware address missing — the wireless module may be faulty.":                     "الواي فاي أو البلوتوث لا يستجيب — قد يكون به عطل.",
	"Storage self-test passed.":                                                                            "فحص وحدة التخزين ناجح.",
	"Storage self-test did not report success.":                                                            "فحص وحدة التخزين لم ينجح.",
	"A SIM is inserted and ready.":                                                                         "الشريحة موجودة وتعمل.",
	"No SIM inserted — insert yours to test calls and data.":                                               "لا توجد شريحة — ضع شريحتك لتجربة المكالمات والإنترنت.",
	"SIM is PIN-locked.":                                                                                   "الشريحة عليها رمز PIN.",
	"Battery has (almost) never been cycled — consistent with a new phone.":                                "البطارية تقريباً لم تُشحن قبل كده — طبيعي لجهاز جديد.",
	"More charge cycles than a new phone should have — it may have been used as a display or for testing.": "دورات شحن أكثر من جهاز جديد — ربما استُخدم كجهاز عرض أو للتجربة.",
	"This battery has been through many charge cycles — the phone has been used before.":                   "البطارية مشحونة مرات كثيرة — الجهاز مستعمل.",
	"Battery is at full factory capacity.":                                                                 "البطارية بكامل سعة المصنع.",
	"A new battery should be at 100% — this one is already worn.":                                          "البطارية الجديدة لازم تكون 100% — هذه مستهلكة.",
	"Only Apple's pre-installed apps are present.":                                                         "فقط تطبيقات Apple الأصلية موجودة.",
	"Storage usage is consistent with a fresh phone.":                                                      "المساحة المستخدمة طبيعية لجهاز جديد.",
	"A lot of storage is already used — photos, apps or data from someone else.":                           "مساحة كبيرة مستخدمة — صور أو تطبيقات أو بيانات لشخص آخر.",
	"No kernel panics recorded.":                                                                           "لا توجد أعطال مسجّلة.",
	"Never activated — as expected for a sealed phone. Activate it now in front of the seller.":            "لم يُفعّل أبداً — طبيعي لجهاز متبرشم. فعّله الآن أمام البائع.",
	"Already activated. Fine if you just set it up yourself; otherwise confirm on checkcoverage.apple.com that the warranty starts today.": "الجهاز مفعّل. لو أنت اللي فعّلته الآن فلا مشكلة، وإلا تأكد من موقع checkcoverage.apple.com أن الضمان يبدأ اليوم.",
	"Battery gauge not readable — check Settings › Battery › Battery Health (100%, cycle count 0).":                                        "لم نتمكن من قراءة البطارية — افتح الإعدادات › البطارية › صحة البطارية (لازم 100% ودورات 0).",
	"Tested — OK.":            "تم الفحص — سليم.",
	"Tested — problem found.": "تم الفحص — توجد مشكلة.",
	"Not tested.":             "لم يتم الفحص.",

	// ---- advice
	"Add a spec file for this model to the specs folder next to the app.":                                                                                         "أضف ملف مواصفات لهذا الموديل في مجلد specs بجانب البرنامج.",
	"Possible logic-board swap or spoofed readings. Inspect carefully.":                                                                                           "قد تكون اللوحة الأم مغيّرة أو البيانات مزوّرة. افحص الجهاز جيداً.",
	"Compare with the A-number printed on the SIM tray/box and Settings › General › About.":                                                                       "قارنه بالرقم المكتوب على العلبة وفي الإعدادات › عام › حول.",
	"Verify coverage at checkcoverage.apple.com and that it matches the box / Settings.":                                                                          "افحص الضمان على checkcoverage.apple.com وتأكد أنه نفس الرقم على العلبة.",
	"Refurbished units are fine if priced accordingly — make sure the seller disclosed it.":                                                                       "الجهاز المجدّد مقبول لو سعره أقل — تأكد أن البائع قال لك ذلك.",
	"A replacement unit issued by Apple service. Usually fine, but not the original retail device.":                                                               "جهاز بديل من صيانة Apple. غالباً سليم لكنه ليس الجهاز الأصلي.",
	"Ex-demo units have been on display 24/7; expect heavier battery wear.":                                                                                       "أجهزة العرض كانت شغالة طول الوقت في المحل؛ البطارية غالباً مستهلكة أكثر.",
	"This region's model is eSIM-only (no physical SIM tray). Make sure your carrier supports eSIM.":                                                              "هذه النسخة eSIM فقط (بدون مكان للشريحة العادية). تأكد أن شركة الاتصالات عندك تدعم eSIM.",
	"Check Settings › General › About. A missing IMEI usually means no cellular service.":                                                                         "افتح الإعدادات › عام › حول. عدم وجود IMEI يعني غالباً أن الشبكة لن تعمل.",
	"Check it against the box / SIM tray and a blacklist (lost/stolen) database before paying.":                                                                   "قارنه بالرقم على العلبة، وتأكد أنه غير مسروق أو مبلّغ عنه قبل الدفع.",
	"Activate it in front of you (insert a SIM / connect to Wi-Fi). If it asks for the previous owner's Apple Account, it is Activation Locked — do not buy.":     "فعّله أمامك (ضع شريحة أو اتصل بالواي فاي). لو طلب حساب Apple لصاحبه السابق فهو مقفول — لا تشتريه.",
	"Ask the seller to sign out of their Apple Account (Settings › [name] › Sign Out), turn off Find My and erase the phone in front of you. Re-scan afterwards.": "اطلب من البائع تسجيل الخروج من حساب Apple (الإعدادات › اسمه › تسجيل الخروج) وقفل Find My ومسح الجهاز أمامك. ثم افحصه مرة أخرى.",
	"Check Settings › [name] › Find My manually. It must be OFF before you pay.":                                                                                  "افتح الإعدادات › الاسم › Find My بنفسك. لازم يكون مقفولاً قبل الدفع.",
	"Company-owned devices can be remotely locked or wiped. Only buy with written release from the organisation.":                                                 "أجهزة الشركات ممكن تتقفل أو تتمسح من بعيد. لا تشتريه إلا بخطاب رسمي من الشركة.",
	"Check Settings › General › VPN & Device Management.":                                                                                                         "افتح الإعدادات › عام › VPN وإدارة الجهاز.",
	"Profiles can install certificates, VPNs or restrictions. Ask the seller to remove them (Settings › General › VPN & Device Management).":                      "ملفات التعريف ممكن تضيف قيوداً أو تراقب الجهاز. اطلب من البائع حذفها (الإعدادات › عام › VPN وإدارة الجهاز).",
	"Beta firmware can be unstable. Ask the seller to leave the beta programme and update to a public release.":                                                   "النسخة التجريبية قد تكون غير مستقرة. اطلب من البائع تثبيت نسخة رسمية.",
	"A jailbroken phone can hide modifications and fake readings. Ask for a full erase + restore, then re-scan.":                                                  "الجهاز المعمول له جيلبريك ممكن يخفي تعديلات ويزوّر البيانات. اطلب مسح كامل وإعادة تثبيت النظام، ثم افحصه مرة أخرى.",
	"Not harmful by itself, but unusual for a personal phone.":                                                                                                    "ليس ضاراً بحد ذاته، لكنه غير معتاد في جهاز شخصي.",
	"Check Settings › Battery › Battery Health manually.":                                                                                                         "افتح الإعدادات › البطارية › صحة البطارية بنفسك.",
	"Check Settings › Battery › Battery Health.":                                                                                                                  "افتح الإعدادات › البطارية › صحة البطارية.",
	"Negotiate the price to cover a battery replacement.":                                                                                                         "فاوض على السعر لتغطية تغيير البطارية.",
	"Expect poor battery life and possible performance throttling. Factor a battery replacement into the price.":                                                  "البطارية ستفصل بسرعة وقد يبطئ الجهاز. احسب تكلفة تغيير البطارية من السعر.",
	"Look for \"Unknown Part\" / \"Service\" messages in Settings › Battery › Battery Health and Settings › General › About.":                                     "ابحث عن رسالة \"Unknown Part\" أو \"Service\" في الإعدادات › البطارية، وفي الإعدادات › عام › حول.",
	"Typical after a bad repair (e.g. charging port flex, microphone, proximity sensor). Sensor names: mic=microphone, prs=pressure/barometer, tg=thermal/gauge.": "يحدث غالباً بعد صيانة سيئة (مثل منفذ الشحن أو المايك أو الحساس). mic = مايك، prs = حساس الضغط، tg = حساس الحرارة.",
	"Check Settings › General › About › Carrier Lock shows \"No SIM restrictions\".":                                                                              "افتح الإعدادات › عام › حول › قفل الشبكة، لازم يكون مكتوب \"No SIM restrictions\".",
	"A phone sold as \"new\" must have a part number starting with M. Ask for a refund or a lower refurbished price.":                                             "الجهاز الجديد لازم رقم القطعة يبدأ بحرف M. اطلب استرجاع الفلوس أو سعر جهاز مجدّد.",
	"Ignore this only if you installed these apps yourself after setting the phone up.":                                                                           "تجاهل هذا فقط لو أنت اللي نزّلت هذه التطبيقات بعد تشغيل الجهاز.",

	// ---- expected values
	"0 – 5 cycles": "0 – 5 دورات",

	// ---- verdicts
	"Do not buy — a critical problem was found": "لا تشتري — توجد مشكلة خطيرة",
	"Looks genuine and healthy":                 "الجهاز أصلي وسليم",
	"Buy with caution — review the warnings":    "اشترِ بحذر — راجع التحذيرات",
	"High risk — several problems were found":   "خطر — توجد مشاكل كثيرة",
}

type arPattern struct {
	re  *regexp.Regexp
	tpl string
}

func pat(re, tpl string) arPattern { return arPattern{regexp.MustCompile("^" + re + "$"), tpl} }

var arPatterns = []arPattern{
	pat(`(.+) \((.+)\) found in the spec database\.`, `$1 ($2) موجود في قاعدة المواصفات.`),
	pat(`Identifier "(.+)" is not in the spec database, so spec cross-checks were skipped\.`, `الموديل "$1" غير موجود في قاعدة المواصفات، لذلك لم تتم مقارنة المواصفات.`),
	pat(`(\d+) independent services agree on the model\.`, `$1 أجزاء من النظام تعطي نفس الموديل.`),
	pat(`Chip ID matches (.+)\.`, `المعالج مطابق: $1.`),
	pat(`Chip ID does not match the (.+) expected for this model\.`, `المعالج لا يطابق $1 المفروض في هذا الموديل.`),
	pat(`Part number starts with "(.)" — sold new by Apple or an authorised reseller\.`, `رقم القطعة يبدأ بحرف "$1" — جهاز جديد من Apple أو موزّع معتمد.`),
	pat(`Part number (.+)`, `رقم القطعة $1`),
	pat(`Sold for: (.+)`, `مُباع في: $1`),
	pat(`Region code (.+)`, `كود المنطقة $1`),
	pat(`(.+) model — capacity matches an official configuration\.`, `نسخة $1 — المساحة رسمية وصحيحة.`),
	pat(`(\d+) profile\(s\) installed\. At least one cannot be removed by the user\.`, `يوجد $1 ملف تعريف، وواحد منها على الأقل لا يمكن حذفه.`),
	pat(`(\d+) profile\(s\) installed\.`, `يوجد $1 ملف تعريف مثبّت.`),
	pat(`Scanned (\d+) apps — no jailbreak tools found\.`, `تم فحص $1 تطبيق — لا توجد برامج جيلبريك.`),
	pat(`Maximum capacity (\d+)% — good condition\.`, `أقصى سعة $1% — حالة جيدة.`),
	pat(`Maximum capacity (\d+)% — worn, replacement will be needed soon\.`, `أقصى سعة $1% — مستهلكة، ستحتاج تغيير قريباً.`),
	pat(`Maximum capacity (\d+)% — below 80%, battery is degraded and needs service\.`, `أقصى سعة $1% — أقل من 80%، البطارية ضعيفة وتحتاج تغيير.`),
	pat(`(\d+) cycles — (\d+)% of rated life used\.`, `$1 دورة شحن — تم استهلاك $2% من عمر البطارية.`),
	pat(`(\d+) cycles — beyond the rated cycle life\.`, `$1 دورة شحن — أكثر من العمر الرسمي للبطارية.`),
	pat(`rated for (\d+) cycles at 80% capacity`, `العمر الرسمي $1 دورة (بسعة 80%)`),
	pat(`(\d+) cycles`, `$1 دورة`),
	pat(`(\d+) panic report\(s\); the system reports missing sensors — a component is disconnected, faulty or non-original\.`, `$1 سجل أعطال، والنظام يقول إن هناك حساسات مفقودة — قطعة مفصولة أو تالفة أو غير أصلية.`),
	pat(`(\d+) panic reports found — the phone has been restarting unexpectedly\.`, `$1 سجلات أعطال — الجهاز كان يعيد التشغيل فجأة.`),
	pat(`(\d+) panic report\(s\) found\.`, `يوجد $1 سجل عطل.`),
	pat(`No kernel panics in (\d+) stored diagnostic reports\.`, `لا توجد أعطال في $1 سجل محفوظ.`),
	pat(`Missing: (.+)`, `مفقود: $1`),
	pat(`SIM state: (.+)`, `حالة الشريحة: $1`),
	pat(`Not a new retail unit: (.+)\.`, `الجهاز ليس جديداً من المصنع: $1.`),
	pat(`(\d+) third-party app\(s\) installed — the phone has been set up and used\.`, `يوجد $1 تطبيق من خارج Apple — الجهاز تم تشغيله واستخدامه.`),
	pat(`(\d+) kernel panic report\(s\) on a phone sold as new\.`, `يوجد $1 سجل عطل في جهاز يُباع على أنه جديد.`),
	pat(`(.+) used`, `$1 مستخدمة`),
	pat(`iOS (.+) or later`, `iOS $1 أو أحدث`),
}

var arCountries = map[string]string{
	"United States": "الولايات المتحدة", "Canada": "كندا", "Mexico": "المكسيك", "Brazil": "البرازيل", "Argentina": "الأرجنتين",
	"United Kingdom": "بريطانيا", "Ireland": "أيرلندا", "Germany": "ألمانيا", "France": "فرنسا", "Italy": "إيطاليا", "Spain": "إسبانيا",
	"Portugal": "البرتغال", "Netherlands": "هولندا", "Belgium": "بلجيكا", "Austria": "النمسا", "Switzerland": "سويسرا", "Poland": "بولندا",
	"Greece": "اليونان", "Turkey": "تركيا", "Russia": "روسيا", "Ukraine": "أوكرانيا", "Kazakhstan": "كازاخستان",
	"United Arab Emirates": "الإمارات", "Bahrain": "البحرين", "Kuwait": "الكويت", "Israel": "إسرائيل", "South Africa": "جنوب أفريقيا",
	"India": "الهند", "Singapore": "سنغافورة", "Hong Kong": "هونج كونج", "Macau": "ماكاو", "China (mainland)": "الصين", "Taiwan": "تايوان",
	"Japan": "اليابان", "South Korea": "كوريا الجنوبية", "Thailand": "تايلاند", "Malaysia": "ماليزيا", "Philippines": "الفلبين",
	"Indonesia": "إندونيسيا", "Vietnam": "فيتنام", "Australia": "أستراليا", "New Zealand": "نيوزيلندا", "Denmark": "الدنمارك",
	"Norway": "النرويج", "Sweden": "السويد", "Finland": "فنلندا", "Czech Republic": "التشيك", "Hungary": "المجر", "Romania": "رومانيا",
	"Middle East (Saudi Arabia, UAE, Qatar, Jordan, Egypt, Oman, Bahrain, Kuwait)": "الشرق الأوسط (السعودية، الإمارات، قطر، الأردن، مصر، عمان، البحرين، الكويت)",
}

var arIdentityLabels = map[string]string{
	"Model": "الموديل", "Device name": "اسم الجهاز", "Identifier": "المعرّف", "Model number": "رقم الموديل",
	"Part number": "رقم القطعة", "Region": "المنطقة", "Serial number": "الرقم التسلسلي", "IMEI": "IMEI", "IMEI 2": "IMEI 2",
	"MEID": "MEID", "iOS": "iOS", "Chip": "المعالج", "Board": "اللوحة الأم", "Capacity": "المساحة", "Colour code": "كود اللون",
	"Activation": "التفعيل", "Baseband": "شريحة الشبكة", "Wi-Fi address": "عنوان الواي فاي", "Bluetooth": "البلوتوث",
	"ECID": "ECID", "UDID": "UDID", "ICCID": "ICCID", "Phone number": "رقم الهاتف",
}

var arActivation = map[string]string{"Activated": "مفعّل", "Unactivated": "غير مفعّل", "FactoryActivated": "مفعّل من المصنع"}

// Arabic reports whether a string already contains Arabic letters.
func hasArabic(s string) bool {
	for _, r := range s {
		if r >= 0x0600 && r <= 0x06FF {
			return true
		}
	}
	return false
}

// TranslateAr translates one English phrase; ok is false when unknown.
func TranslateAr(s string) (string, bool) {
	if s == "" {
		return s, true
	}
	if v, ok := arExact[s]; ok {
		return v, true
	}
	for _, p := range arPatterns {
		if p.re.MatchString(s) {
			out := p.re.ReplaceAllString(s, p.tpl)
			// translate country lists and part-prefix meanings embedded in patterns
			for en, ar := range arCountries {
				out = strings.ReplaceAll(out, en, ar)
			}
			if m := regexp.MustCompile(`: (.+)\.$`).FindStringSubmatch(out); m != nil {
				for en, ar := range arExact {
					if strings.EqualFold(m[1], en) {
						out = strings.Replace(out, m[1], ar, 1)
					}
				}
			}
			return out, true
		}
	}
	return s, false
}

func trAr(s string) string {
	out, _ := TranslateAr(s)
	return out
}

// Localize returns a copy of the result presented in the given language.
func Localize(r *Result, lang string) *Result {
	if r == nil || lang != "ar" {
		return r
	}
	c := *r
	c.Lang = "ar"
	c.VerdictText = trAr(r.VerdictText)
	c.Checks = make([]Check, len(r.Checks))
	for i, ck := range r.Checks {
		if strings.HasPrefix(ck.ID, "manual-") {
			if t := manualByID(strings.TrimPrefix(ck.ID, "manual-")); t != nil && t.TitleAr != "" {
				ck.Title = t.TitleAr
			}
		} else {
			ck.Title = trAr(ck.Title)
		}
		ck.Summary = trAr(ck.Summary)
		ck.Advice = trAr(ck.Advice)
		ck.Expected = trAr(ck.Expected)
		if out, ok := TranslateAr(ck.Actual); ok && hasArabic(out) && !strings.Contains(ck.Actual, "/") {
			ck.Actual = out
		}
		c.Checks[i] = ck
	}
	c.Identity = make([]KV, len(r.Identity))
	for i, kv := range r.Identity {
		if v, ok := arIdentityLabels[kv.Label]; ok {
			kv.Label = v
		}
		if kv.Label == "المنطقة" {
			for en, ar := range arCountries {
				kv.Value = strings.ReplaceAll(kv.Value, en, ar)
			}
		}
		if v, ok := arActivation[kv.Value]; ok && kv.Label == "التفعيل" {
			kv.Value = v
		}
		c.Identity[i] = kv
	}
	return &c
}

func manualByID(id string) *ManualTest {
	for i := range ManualTests {
		if ManualTests[i].ID == id {
			return &ManualTests[i]
		}
	}
	return nil
}
