/* Smart Mobile Check — user interface (Arabic / English) */
(() => {
  "use strict";

  // ------------------------------------------------------------ language
  const loadLang = () => { try { return localStorage.getItem("smc-lang") || "ar"; } catch (e) { return "ar"; } };
  const state = {
    lang: loadLang(),
    env: null,
    devices: [],
    devError: "",
    pairing: {},      // udid -> {state, busy}
    scan: null,
    result: null,
    manualTests: [],
    manual: {},       // id -> {status, note}
    notes: "",
    tab: "summary",
    specs: null,
    specFilter: "",
    appFilter: "",
    demoProfiles: [],
    mode: (() => { try { return localStorage.getItem("smc-mode") || "used"; } catch (e) { return "used"; } })(),
    manualFor: null,  // "pending" when the checklist was started before scanning
    platform: "ios",  // ios | android
    mtPlatform: "",
    aDemo: [],
    aDevices: [],
    aError: "",
  };
  /** L returns the text for the current language. */
  const L = (en, ar) => (state.lang === "ar" ? ar : en);
  const isAr = () => state.lang === "ar";
  const applyLang = () => {
    document.documentElement.lang = state.lang;
    document.documentElement.dir = isAr() ? "rtl" : "ltr";
    document.getElementById("specsLink").textContent = L("Spec database", "قاعدة المواصفات");
    document.getElementById("quitBtn").textContent = L("Quit", "خروج");
    document.getElementById("langBtn").textContent = L("العربية", "English");
  };

  // ------------------------------------------------------------ helpers
  const $app = document.getElementById("app");
  const $crumbs = document.getElementById("crumbs");
  const bdi = (v) => `<bdi>${esc(v)}</bdi>`;
  const esc = (v) => String(v ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const api = async (path, body) => {
    const opt = body === undefined ? {} : { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
    const r = await fetch(path, opt);
    if (!r.ok) throw new Error((await r.text()) || r.statusText);
    return r.json();
  };
  const withLang = (path) => path + (path.includes("?") ? "&" : "?") + "lang=" + state.lang;
  let toastTimer;
  const toast = (msg) => {
    const t = document.getElementById("toast");
    t.textContent = msg;
    t.classList.add("show");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => t.classList.remove("show"), 3200);
  };
  const ICON = { pass: "✓", warn: "!", fail: "✕", info: "i", skip: "–" };
  const ORDER = { fail: 0, warn: 1, info: 2, skip: 3, pass: 4 };
  const SEV = { critical: 0, major: 1, minor: 2, info: 3 };
  const CAT_ICON = {
    "Identity & authenticity": "🪪", "Ownership & locks": "🔐", "Software integrity": "🛡️",
    "Battery": "🔋", "Hardware & diagnostics": "🧪", "Hands-on tests": "✋", "New-device checks": "📦",
  };
  const CAT_AR = {
    "Identity & authenticity": "الهوية والأصالة", "Ownership & locks": "الملكية والأقفال", "Software integrity": "سلامة النظام",
    "Battery": "البطارية", "Hardware & diagnostics": "الهاردوير والأعطال", "Hands-on tests": "الاختبارات اليدوية", "New-device checks": "فحوصات الجهاز الجديد",
  };
  const catName = (cat, mode) => {
    if (cat === "Hands-on tests" && mode === "new") return state.platform === "android" ? L("New-phone checklist", "قائمة الموبايل الجديد") : L("New-iPhone checklist", "قائمة الآيفون الجديد");
    return L(cat, CAT_AR[cat] || cat);
  };
  const SEV_AR = { critical: "خطير", major: "مهم", minor: "بسيط", info: "معلومة" };
  const sevName = (s) => L(s, SEV_AR[s] || s);
  const COLOR = { good: "#22c55e", caution: "#f59e0b", risk: "#ef4444" };
  const ARROW = () => (isAr() ? "←" : "→");
  const NEXT = () => (isAr() ? "‹" : "›");

  let timers = [];
  const every = (fn, ms) => { fn(); timers.push(setInterval(fn, ms)); };
  const clearTimers = () => { timers.forEach(clearInterval); timers = []; };

  const P = () => (state.platform === "android" ? "/api/android" : "/api/ios");
  const connectHash = () => (state.platform === "android" ? "#/android" : "#/iphone");
  const platformName = () => (state.platform === "android" ? "Android" : "iPhone");
  async function ensureTests() {
    if (state.mtPlatform !== state.platform || !state.manualTests.length) {
      state.manualTests = await api(P() + "/manual-tests").catch(() => []);
      state.mtPlatform = state.platform;
    }
  }
  const setMode = (m) => { state.mode = m; try { localStorage.setItem("smc-mode", m); } catch (e) { /* ignore */ } };
  const testsFor = (mode) => state.manualTests.filter((t) => (t.modes || ["used", "new"]).includes(mode));
  const doneCount = (tests) => tests.filter((t) => state.manual[t.id] && state.manual[t.id].status).length;
  const tTitle = (t) => (isAr() && t.titleAr ? t.titleAr : t.title);
  const tGroup = (t) => (isAr() && t.groupAr ? t.groupAr : t.group);
  const tSteps = (t) => (isAr() && t.stepsAr && t.stepsAr.length ? t.stepsAr : t.steps);
  const modeBar = () => `<div class="mode-bar"><span class="muted">${L("I'm buying a", "أنا بشتري")}</span>
      <div class="seg mode-seg">${["used", "new"].map((m) => `<button class="${state.mode === m ? "on mode" : ""}" data-mode="${m}">${state.platform === "android" ? (m === "new" ? L("New / sealed phone", "موبايل جديد متبرشم") : L("Used phone", "موبايل مستعمل")) : (m === "new" ? L("New / sealed iPhone", "آيفون جديد متبرشم") : L("Used iPhone", "آيفون مستعمل"))}</button>`).join("")}</div>
      <span class="dim" style="font-size:12px">${state.mode === "new"
        ? L("Adds checks that prove the phone was never used, plus the new-phone checklist.", "بيضيف فحوصات بتتأكد إن الموبايل ما اتستعملش قبل كده، ومعاها قائمة الموبايل الجديد.")
        : L("Full inspection for a second-hand phone.", "فحص كامل لموبايل مستعمل.")}</span></div>`;
  const bindModeBar = (root, after) => root.querySelectorAll("[data-mode]").forEach((b) => b.addEventListener("click", () => { setMode(b.dataset.mode); after(); }));

  const crumbs = (items) => {
    $crumbs.innerHTML = items.map((it, i) => (i ? `<span class="sep">${NEXT()}</span>` : "") + (it.href ? `<a href="${it.href}">${esc(it.label)}</a>` : `<span>${esc(it.label)}</span>`)).join("");
  };
  const HOME = () => ({ label: L("Home", "الرئيسية"), href: "#/" });

  const ring = (pct, size, stroke, color, inner) => {
    const r = (size - stroke) / 2;
    const c = 2 * Math.PI * r;
    const off = c * (1 - Math.max(0, Math.min(100, pct)) / 100);
    return `<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
      <circle cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="rgba(148,163,184,.15)" stroke-width="${stroke}"/>
      <circle cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="${color}" stroke-width="${stroke}" stroke-linecap="round"
        stroke-dasharray="${c}" stroke-dashoffset="${off}" transform="rotate(-90 ${size / 2} ${size / 2})" style="transition:stroke-dashoffset .5s ease"/>
      ${inner || ""}</svg>`;
  };

  const phoneSVG = (w = 150, glow = true) => `
    <svg width="${w}" height="${w * 2}" viewBox="0 0 150 300" aria-hidden="true">
      <defs>
        <linearGradient id="pg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6366f1"/><stop offset="1" stop-color="#22d3ee"/></linearGradient>
        <linearGradient id="sc" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1e293b"/><stop offset="1" stop-color="#0f172a"/></linearGradient>
      </defs>
      <rect x="6" y="6" width="138" height="288" rx="30" fill="#0b1222" stroke="url(#pg)" stroke-width="3"/>
      <rect x="14" y="14" width="122" height="272" rx="23" fill="url(#sc)"/>
      <rect x="55" y="24" width="40" height="11" rx="5.5" fill="#000"/>
      ${glow ? '<circle cx="75" cy="150" r="26" fill="none" stroke="url(#pg)" stroke-width="3" opacity=".9"/><path d="M64 150l8 8 15-17" fill="none" stroke="url(#pg)" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>' : ""}
      <rect x="146" y="80" width="3" height="34" rx="1.5" fill="#334155"/>
      <rect x="1" y="70" width="3" height="20" rx="1.5" fill="#334155"/>
      <rect x="1" y="98" width="3" height="34" rx="1.5" fill="#334155"/>
      <rect x="62" y="296" width="26" height="4" rx="2" fill="#334155"/>
    </svg>`;

  const DEMO_AR = {
    healthy: ["آيفون 17 برو ماكس مستعمل — سليم", "موبايل مستعمل أصلي وسليم مع ملاحظات بسيطة."],
    tampered: ["آيفون 13 برو مستعمل — فيه مشاكل", "قفل تفعيل، جيلبريك، بطارية متغيّرة وبيانات مزوّرة."],
    new: ["آيفون 17 برو ماكس جديد — متبرشم", "موبايل جديد متفحوص في وضع الجهاز الجديد."],
  };
  const demoTitle = (p) => (isAr() && DEMO_AR[p.id] ? DEMO_AR[p.id][0] : p.title);
  const demoDesc = (p) => (isAr() && DEMO_AR[p.id] ? DEMO_AR[p.id][1] : p.desc);

  // ------------------------------------------------------------ router
  const routes = {
    "": viewHome,
    "iphone": viewConnect,
    "scan": viewScan,
    "result": viewResult,
    "specs": viewSpecs,
    "checklist": viewChecklist,
    "android": viewAndroid,
  };
  function route() {
    clearTimers();
    applyLang();
    const [name, arg] = location.hash.replace(/^#\/?/, "").split("/");
    (routes[name] || viewHome)(arg ? decodeURIComponent(arg) : undefined);
    window.scrollTo(0, 0);
  }
  window.addEventListener("hashchange", route);
  const go = (h) => { if (location.hash === h) route(); else location.hash = h; };

  // ------------------------------------------------------------ home
  async function viewHome() {
    crumbs([{ label: L("Home", "الرئيسية") }]);
    $app.innerHTML = `
      <section class="hero fade-in">
        <span class="badge">${L("Used & new phone inspection", "فحص الموبايلات المستعملة والجديدة")}</span>
        <h1 style="margin-top:14px">${L("Know exactly what you're buying.", "اعرف بالظبط إنت بتشتري إيه.")}<br><span class="grad">${L("Before you pay.", "قبل ما تدفع.")}</span></h1>
        <p>${L("Connect the phone with a USB cable. Smart Mobile Check reads the hardware identity, locks, battery wear and diagnostic logs, flags tampering, then guides you through the hands-on tests.",
          "وصّل الموبايل بالكمبيوتر بكابل USB. البرنامج هيفحص هل الموبايل أصلي، وعليه قفل ولا لأ، وحالة البطارية، وسجل الأعطال، وهيقولك لو حد لعب فيه. وبعدين هيمشّيك خطوة بخطوة في الاختبارات اليدوية.")}</p>
      </section>
      <section class="platforms fade-in">
        <div class="platform" data-go="#/iphone" tabindex="0" role="button">
          <div class="glow" style="background:#6366f1"></div>
          <span class="badge live"><span class="dot ok"></span> ${L("Available", "متاح")}</span>
          <span class="go btn small primary">${L("Start ›", "ابدأ ‹")}</span>
          <svg width="38" height="38" viewBox="0 0 24 24" fill="#e8edf7" style="margin-top:18px"><path d="M16.37 1.43c0 1.14-.42 2.2-1.25 3.04-.86.87-1.9 1.37-3.03 1.28a3.4 3.4 0 0 1-.02-.42c0-1.1.48-2.27 1.33-3.1.43-.42.97-.77 1.63-1.05.65-.28 1.27-.43 1.850-.46.02.24.03.48.03.71zM20.6 17.2c-.42.97-.92 1.87-1.5 2.68-.79 1.11-1.43 1.88-1.93 2.3-.77.7-1.6 1.050-2.48 1.07-.63 0-1.4-.18-2.29-.54-.9-.36-1.71-.54-2.46-.54-.78 0-1.62.18-2.52.54-.9.36-1.62.55-2.17.57-.85.04-1.7-.33-2.54-1.1-.54-.46-1.2-1.26-2.01-2.4-.86-1.2-1.57-2.6-2.12-4.2C-.02 13.85-.3 12.2-.3 10.6c0-1.83.4-3.4 1.18-4.73a6.9 6.9 0 0 1 2.48-2.5 6.66 6.66 0 0 1 3.35-.95c.66 0 1.52.2 2.6.6 1.07.4 1.75.6 2.05.6.23 0 1-.24 2.31-.71 1.24-.44 2.28-.62 3.14-.55 2.32.19 4.06 1.1 5.22 2.74-2.08 1.26-3.1 3.02-3.08 5.28.02 1.76.66 3.22 1.9 4.38.57.54 1.2.95 1.9 1.25-.15.44-.32.86-.49 1.27z" transform="translate(1.5 0)"/></svg>
          <h2>iPhone</h2>
          <p>${L("Full automated inspection over USB plus guided hands-on tests.", "فحص أوتوماتيك كامل بالكابل، ومعاه اختبارات يدوية سهلة.")}</p>
          <ul>
            <li>${L("Activation Lock, MDM & profile detection", "كشف قفل التفعيل وأجهزة الشركات")}</li>
            <li>${L("Jailbreak & tampered-reading detection", "كشف الجيلبريك والبيانات المزوّرة")}</li>
            <li>${L("Battery health, cycles & original-battery check", "صحة البطارية ودورات الشحن وهل هي أصلية")}</li>
            <li>${L("Model, storage, IMEI & serial authenticity", "التأكد من الموديل والمساحة والـ IMEI والسيريال")}</li>
            <li>${L("Kernel panic & missing-sensor analysis", "قراءة سجل الأعطال والقطع المفصولة")}</li>
          </ul>
        </div>
        <div class="platform" data-go="#/android" tabindex="0" role="button">
          <div class="glow" style="background:#22c55e;opacity:.35"></div>
          <span class="badge live"><span class="dot ok"></span> ${L("Available", "متاح")}</span>
          <span class="go btn small primary">${L("Start ›", "ابدأ ‹")}</span>
          <svg width="38" height="38" viewBox="0 0 24 24" fill="#3ddc84" style="margin-top:18px"><path d="M17.6 9.48l1.84-3.18a.38.38 0 0 0-.66-.38l-1.86 3.22a11.4 11.4 0 0 0-9.84 0L5.22 5.92a.38.38 0 0 0-.66.38L6.4 9.48A10.8 10.8 0 0 0 1 18h22a10.8 10.8 0 0 0-5.4-8.52zM7 15.25a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5zm10 0a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5z"/></svg>
          <h2>Android</h2>
          <p>${L("Samsung, Google Pixel, Xiaomi and more — over USB debugging.", "سامسونج وجوجل بيكسل وشاومي وغيرهم — عن طريق تصحيح USB.")}</p>
          <ul>
            <li>${L("Google / Samsung / Xiaomi account lock (FRP)", "قفل حساب جوجل وسامسونج وشاومي (FRP)")}</li>
            <li>${L("Fake / clone phone detection (chip, RAM, storage)", "كشف الموبايلات التقليد (المعالج والرامات والمساحة)")}</li>
            <li>${L("Root, unlocked bootloader & Knox status", "الروت والبوت لودر المفتوح وحالة Knox")}</li>
            <li>${L("Company management (MDM) & Knox Guard", "إدارة الشركات (MDM) وقفل Knox Guard")}</li>
            <li>${L("Battery, sensors, cameras & crash history", "البطارية والحساسات والكاميرات وسجل الأعطال")}</li>
          </ul>
        </div>
      </section>
      <a class="new-banner fade-in" href="#/checklist">
        <span class="nb-icon">📦</span>
        <span><b>${L("Buying a brand-new iPhone?", "هتشتري آيفون جديد؟")}</b><br><span class="muted">${L("Open the new-iPhone checklist: box seal, serial & IMEI match, warranty status, first start, battery and more.",
          "افتح قائمة الآيفون الجديد: برشام العلبة، والسيريال والـ IMEI، والضمان، وأول تشغيل، والبطارية وحاجات تانية.")}</span></span>
        <span class="btn small">${L("Open checklist ›", "افتح القائمة ‹")}</span>
      </a>
      <div class="status-strip" id="envStrip"><span class="pill"><span class="spinner"></span> ${L("Checking Apple device service…", "بنتأكد من تعريف Apple…")}</span></div>
      <div class="status-strip" id="demoStrip"></div>`;
    $app.querySelectorAll("[data-go]").forEach((el) => {
      const h = () => go(el.dataset.go);
      el.addEventListener("click", h);
      el.addEventListener("keydown", (e) => { if (e.key === "Enter") h(); });
    });
    await loadEnv();
    renderEnvStrip();
    if (!state.demoProfiles.length) state.demoProfiles = await api("/api/demo/profiles").catch(() => []);
    if (!state.aDemo.length) state.aDemo = await api("/api/android/demo/profiles").catch(() => []);
    const ds = document.getElementById("demoStrip");
    if (!ds) return;
    ds.innerHTML = `<span class="muted" style="align-self:center">${L("No phone at hand? Try a demo:", "مفيش موبايل معاك؟ جرّب مثال:")}</span>` +
      state.demoProfiles.map((p) => `<button class="btn small" data-demo="${esc(p.id)}" title="${esc(demoDesc(p))}">${esc(demoTitle(p))}</button>`).join("") +
      state.aDemo.map((p) => `<button class="btn small" data-platform="android" data-demo="${esc(p.id)}" title="${esc(p.desc)}">🤖 ${esc(isAr() ? p.titleAr : p.title)}</button>`).join("");
    bindDemo();
  }

  async function loadEnv() {
    try { state.env = await api("/api/env"); } catch (e) { state.env = { serviceOK: false, serviceError: String(e) }; }
  }
  function renderEnvStrip() {
    const el = document.getElementById("envStrip");
    if (!el || !state.env) return;
    const e = state.env;
    el.innerHTML = `
      <span class="pill"><span class="dot ${e.serviceOK ? "ok" : "bad"}"></span>${L("Apple device service", "تعريف Apple")}: ${e.serviceOK ? L("running", "شغّال") : L("not found", "مش موجود")}</span>
      <a class="pill" href="#/specs"><span class="dot ok"></span>${e.specCount || 0} ${L("device models in database", "موديل في قاعدة البيانات")}</a>
      <span class="pill">v${esc(e.version || "")}</span>`;
  }
  function bindDemo() {
    $app.querySelectorAll("[data-demo]").forEach((b) => b.addEventListener("click", () => {
      state.platform = b.dataset.platform || "ios";
      const list = state.platform === "android" ? state.aDemo : state.demoProfiles;
      const p = list.find((x) => x.id === b.dataset.demo);
      if (p && p.mode) setMode(p.mode);
      startScan({ demo: b.dataset.demo });
    }));
  }

  // ------------------------------------------------------------ Android connect
  const androidSVG = () => phoneSVG(150).replace('<rect x="55" y="24" width="40" height="11" rx="5.5" fill="#000"/>', '<circle cx="75" cy="30" r="6" fill="#000"/>').replace(/#6366f1/g, "#22c55e");

  async function viewAndroid() {
    state.platform = "android";
    crumbs([HOME(), { label: "Android" }, { label: L("Connect", "التوصيل") }]);
    $app.innerHTML = `<div class="connect fade-in">
        <div class="card phone-art">${androidSVG()}</div>
        <div id="connectPane"><div class="card"><div class="row"><span class="spinner"></span> ${L("Preparing Android tools…", "بنجهّز أدوات أندرويد…")}</div></div></div>
      </div>`;
    if (!state.aDemo.length) state.aDemo = await api("/api/android/demo/profiles").catch(() => []);
    const t = await api("/api/android/tools").catch((e) => ({ error: String(e) }));
    if (!t.serverRunning) return renderNoTools(t);
    every(pollAndroid, 2000);
  }

  function aDemoButtons() {
    return state.aDemo.map((p) => `<button class="btn small" data-platform="android" data-demo="${esc(p.id)}">${esc(isAr() ? p.titleAr : p.title)}</button>`).join("");
  }

  function renderNoTools(t, busy, err) {
    const pane = document.getElementById("connectPane");
    if (!pane) return;
    pane.innerHTML = `<div class="notice" style="border-color:rgba(245,158,11,.4);background:rgba(245,158,11,.07)">
        <h3 style="color:#fde68a">${L("One-time setup: Android tools", "إعداد لمرة واحدة: أدوات أندرويد")}</h3>
        <p class="muted" style="margin:6px 0 0">${L("To talk to Android phones the app uses Google's official ADB tool (free, about 9 MB). Download it once — it is saved on this computer for next time.",
          "علشان البرنامج يكلّم موبايلات أندرويد بيستخدم أداة ADB الرسمية من جوجل (مجانية، حوالي 9 ميجا). نزّلها مرة واحدة بس وهتتحفظ على الكمبيوتر.")}</p>
      </div>
      <div class="card" style="margin-top:14px">
        <div class="row">
          <button class="btn primary" id="dlTools" ${busy ? "disabled" : ""}>${busy ? `<span class="spinner"></span> ${L("Downloading…", "جاري التحميل…")}` : L("Download from Google", "نزّلها من جوجل")}</button>
          <button class="btn" id="recheckTools">${L("Check again", "اتأكد تاني")}</button>
        </div>
        ${err ? `<p class="trust-box" style="margin-top:12px">${esc(err)}</p>` : ""}
        <p class="dim" style="margin-top:12px;font-size:12px">${L("Source", "المصدر")}: <span class="mono">${esc(t.downloadUrl || "dl.google.com")}</span>${t.error ? ` · <span class="mono">${esc(t.error)}</span>` : ""}</p>
      </div>
      <div class="row" style="margin-top:14px"><span class="muted">${L("Just exploring?", "عايز تجرّب بس؟")}</span>${aDemoButtons()}</div>`;
    document.getElementById("dlTools").onclick = async () => {
      renderNoTools(t, true);
      try {
        const r = await api("/api/android/tools/download", {});
        if (r.ok) { toast(L("Android tools ready", "أدوات أندرويد جاهزة")); return viewAndroid(); }
        renderNoTools(t, false, r.error || L("Download failed", "التحميل فشل"));
      } catch (e) { renderNoTools(t, false, String(e)); }
    };
    document.getElementById("recheckTools").onclick = () => viewAndroid();
    bindDemo();
  }

  async function pollAndroid() {
    try {
      const r = await api("/api/android/devices");
      state.aError = r.ok ? "" : r.error;
      state.aDevices = r.devices || [];
    } catch (e) { state.aError = String(e); }
    renderAndroidDevices();
  }

  function androidSteps() {
    return `<ol class="steps">
      <li><div>${L('Open <b>Settings › About phone › Software information</b> and tap <b>Build number</b> 7 times until it says "Developer mode turned on". (Xiaomi: tap <b>OS version</b> 7 times. Pixel: <b>Settings › About phone › Build number</b>.)',
        'افتح <b>الإعدادات › حول الهاتف › معلومات البرنامج</b> ودوس على <b>رقم الإصدار</b> 7 مرات لحد ما يقول "تم تفعيل وضع المطوّر". (شاومي: دوس على <b>إصدار النظام</b> 7 مرات. بيكسل: <b>الإعدادات › حول الهاتف › رقم الإصدار</b>.)')}</div></li>
      <li><div>${L("Go to <b>Settings › Developer options</b> and turn on <b>USB debugging</b>.", "ارجع لـ<b>الإعدادات › خيارات المطوّر</b> وفعّل <b>تصحيح USB</b>.")}</div></li>
      <li><div>${L('Connect the phone with a <b>USB data cable</b>. If asked, choose <b>File transfer</b>.', 'وصّل الموبايل بـ<b>كابل USB بينقل بيانات</b>. لو سألك، اختار <b>نقل الملفات</b>.')}</div></li>
      <li><div>${L('On the phone, tap <b>Allow</b> on "Allow USB debugging?" (tick <b>Always allow</b>).', 'هيظهر على الموبايل "السماح بتصحيح USB؟" — علّم <b>السماح دائماً</b> ودوس <b>سماح</b>.')}</div></li>
    </ol>`;
  }

  function renderAndroidDevices() {
    const pane = document.getElementById("connectPane");
    if (!pane) return;
    const devs = state.aDevices;
    const drivers = `<div class="row" style="margin-top:14px"><span class="muted" style="font-size:12.5px">${L("Computer doesn't see the phone? Install its USB driver:", "الكمبيوتر مش شايف الموبايل؟ نزّل تعريف USB:")}</span>
      <button class="btn small" data-open="samsung-driver">${L("Samsung driver", "تعريف سامسونج")}</button>
      <button class="btn small" data-open="google-driver">${L("Google / Pixel driver", "تعريف جوجل / بيكسل")}</button>
      <button class="btn small" data-open="oem-drivers">${L("Other brands", "شركات تانية")}</button></div>`;
    if (!devs.length) {
      pane.innerHTML = `${modeBar()}<div class="card">
          <div class="row" style="justify-content:space-between"><h2 style="font-size:22px">${L("Connect the Android phone", "وصّل موبايل الأندرويد")}</h2><span class="pill"><span class="dot wait"></span>${L("Waiting for device", "مستنيين الموبايل")}</span></div>
          ${androidSteps()}
          ${drivers}
          ${state.aError ? `<p class="dim" style="font-size:12px;margin-top:12px">${esc(state.aError)}</p>` : ""}
          <p class="dim" style="font-size:12px;margin-top:10px">${L("Read-only: the app only reads information, it does not change anything on the phone. Turn USB debugging off when you're done.", "البرنامج بيقرا بس، مش بيغيّر أي حاجة في الموبايل. اقفل تصحيح USB بعد ما تخلص.")}</p>
        </div>
        <div class="row" style="margin-top:14px"><span class="muted">${L("Just exploring?", "عايز تجرّب بس؟")}</span>${aDemoButtons()}</div>`;
      bindModeBar(pane, renderAndroidDevices);
      bindOpen(pane);
      bindDemo();
      return;
    }
    pane.innerHTML = `${modeBar()}<div class="card"><h2 style="font-size:22px">${devs.length > 1 ? L("Choose a device", "اختار الموبايل") : L("Android phone detected", "تم العثور على الموبايل")}</h2>
      ${devs.map(androidCard).join("")}</div>${drivers}`;
    bindModeBar(pane, renderAndroidDevices);
    bindOpen(pane);
    pane.querySelectorAll("[data-scan]").forEach((b) => b.addEventListener("click", () => { state.platform = "android"; startScan({ udid: b.dataset.scan }); }));
  }

  function androidCard(d) {
    let action = "", note = "", ok = false, label = "";
    switch (d.state) {
      case "device":
        ok = true;
        label = L("Connected — ready to scan", "متوصّل — جاهز للفحص");
        action = `<button class="btn primary big" data-scan="${esc(d.serial)}">${L("Start full scan", "ابدأ الفحص الكامل")}</button>`;
        break;
      case "unauthorized":
        label = L("Waiting for permission", "مستنيين السماح");
        note = `<div class="trust-box">${L('Look at the phone and tap <b>Allow</b> on "Allow USB debugging?" (tick "Always allow"). If it doesn\'t appear, unplug and reconnect the cable.', 'بص على الموبايل ودوس <b>سماح</b> على "السماح بتصحيح USB؟" (علّم "السماح دائماً"). لو ماظهرش، شيل الكابل ووصّله تاني.')}</div>`;
        action = `<button class="btn" disabled><span class="spinner"></span> ${L("Waiting…", "مستنيين…")}</button>`;
        break;
      case "offline":
        label = L("Offline", "غير متصل");
        note = `<div class="trust-box">${L("Unplug and reconnect the cable, then unlock the phone.", "شيل الكابل ووصّله تاني وافتح قفل الموبايل.")}</div>`;
        break;
      default:
        label = d.state;
        note = `<div class="trust-box">${L("The phone is in a special mode or needs a USB driver.", "الموبايل في وضع خاص أو محتاج تعريف USB.")}</div>`;
    }
    const name = d.modelName || d.model || "Android";
    return `<div class="device-card">
        ${androidSVG().replace('width="150" height="300"', 'width="38" height="76"')}
        <div class="meta">
          <h3>${esc(name)}</h3>
          <div class="sub">${d.model ? esc(d.model) + " · " : ""}${d.device ? esc(d.device) + " · " : ""}<span class="mono">${esc(d.serial)}</span></div>
          <div class="sub" style="margin-top:4px"><span class="dot ${ok ? "ok" : "wait"}" style="display:inline-block;margin-inline-end:6px"></span>${esc(label)}</div>
          ${note}
        </div>
        ${action}
      </div>`;
  }

  // ------------------------------------------------------------ connect
  async function viewConnect() {
    state.platform = "ios";
    crumbs([HOME(), { label: "iPhone" }, { label: L("Connect", "التوصيل") }]);
    $app.innerHTML = `<div class="connect fade-in">
        <div class="card phone-art">${phoneSVG(150)}</div>
        <div id="connectPane"><div class="card"><div class="row"><span class="spinner"></span> ${L("Looking for the Apple device service…", "بندوّر على تعريف Apple…")}</div></div></div>
      </div>`;
    await loadEnv();
    if (!state.demoProfiles.length) state.demoProfiles = await api("/api/demo/profiles").catch(() => []);
    if (!state.env.serviceOK) return renderNoService();
    every(pollDevices, 2000);
  }

  function renderNoService() {
    const pane = document.getElementById("connectPane");
    if (!pane) return;
    pane.innerHTML = `<div class="notice">
        <h3>${L("Apple device service not found", "تعريف Apple مش موجود على الكمبيوتر")}</h3>
        <p class="muted" style="margin:6px 0 0">${L("Windows needs Apple's USB driver to talk to an iPhone. Install <b>one</b> of these (free, from Apple), then click <i>Check again</i>:",
          "ويندوز محتاج تعريف Apple علشان يكلّم الآيفون. نزّل <b>واحد</b> من دول (مجاني من Apple)، وبعدين دوس <i>اتأكد تاني</i>:")}</p>
      </div>
      <div class="card" style="margin-top:14px">
        <ol class="steps">
          <li><div>${L("<b>Apple Devices</b> app from the Microsoft Store (recommended on Windows 10/11), or <b>iTunes</b>.", "برنامج <b>Apple Devices</b> من Microsoft Store (الأفضل على ويندوز 10 و11)، أو <b>iTunes</b>.")}</div></li>
          <li><div>${L("Open it once so the driver finishes installing, then close it.", "افتحه مرة واحدة علشان التعريف يتسطّب، وبعدين اقفله.")}</div></li>
          <li><div>${L("Reconnect the iPhone and come back here.", "وصّل الآيفون تاني وارجع هنا.")}</div></li>
        </ol>
        <div class="row" style="margin-top:18px">
          <button class="btn primary" data-open="apple-devices">${L("Get Apple Devices", "نزّل Apple Devices")}</button>
          <button class="btn" data-open="itunes">${L("Get iTunes", "نزّل iTunes")}</button>
          <button class="btn" id="recheck">${L("Check again", "اتأكد تاني")}</button>
        </div>
        <p class="dim" style="margin-top:14px;font-size:12px">${L("Details", "تفاصيل")}: <span class="mono">${esc(state.env.serviceError || "")}</span></p>
      </div>
      <div class="row" style="margin-top:14px"><span class="muted">${L("Just exploring?", "عايز تجرّب بس؟")}</span>${state.demoProfiles.map((p) => `<button class="btn small" data-demo="${esc(p.id)}">${esc(demoTitle(p))}</button>`).join("")}</div>`;
    document.getElementById("recheck").onclick = () => viewConnect();
    bindOpen(pane);
    bindDemo();
  }

  function bindOpen(root) {
    root.querySelectorAll("[data-open]").forEach((b) => b.addEventListener("click", async () => {
      try {
        const r = await api("/api/open", { target: b.dataset.open });
        if (!r.ok && b.dataset.open === "apple-devices") await api("/api/open", { target: "apple-devices-web" });
      } catch (e) { toast(L("Could not open: ", "ماقدرناش نفتح: ") + e.message); }
    }));
  }

  let lastOthers = [];
  async function pollDevices() {
    try {
      const r = await api("/api/ios/devices");
      state.devError = r.ok ? "" : r.error;
      state.devices = (r.devices || []).filter((d) => !d.deviceClass || d.deviceClass === "iPhone" || d.deviceClass === "");
      lastOthers = (r.devices || []).filter((d) => d.deviceClass && d.deviceClass !== "iPhone");
    } catch (e) {
      state.devError = String(e);
    }
    renderDevices();
  }

  function renderDevices() {
    const pane = document.getElementById("connectPane");
    if (!pane) return;
    if (!state.devices.length) {
      pane.innerHTML = `${modeBar()}<div class="card">
          <div class="row" style="justify-content:space-between"><h2 style="font-size:22px">${L("Connect the iPhone", "وصّل الآيفون")}</h2><span class="pill"><span class="dot wait"></span>${L("Waiting for device", "مستنيين الموبايل")}</span></div>
          <ol class="steps">
            <li><div>${L("Plug the iPhone into this computer with a <b>USB cable</b> (use a data cable, not a charge-only one).", "وصّل الآيفون بالكمبيوتر بـ<b>كابل USB</b> (كابل بينقل بيانات، مش كابل شحن بس).")}</div></li>
            <li><div>${L("<b>Unlock</b> the iPhone and keep it on the Home Screen.", "<b>افتح قفل</b> الآيفون وسيبه على الشاشة الرئيسية.")}</div></li>
            <li><div>${L('If asked <b>"Trust This Computer?"</b>, tap <b>Trust</b> and enter the passcode.', 'لو ظهر سؤال <b>"Trust This Computer?"</b> دوس <b>Trust</b> واكتب الباسكود.')}</div></li>
          </ol>
          ${lastOthers.length ? `<p class="trust-box">${lastOthers.map((o) => esc(o.deviceClass)).join(", ")} ${L("detected — only iPhones are supported for now.", "متوصّل — البرنامج بيدعم الآيفون بس دلوقتي.")}</p>` : ""}
          ${state.devError ? `<p class="dim" style="font-size:12px;margin-top:12px">${esc(state.devError)}</p>` : ""}
        </div>`;
      bindModeBar(pane, renderDevices);
      return;
    }
    pane.innerHTML = `${modeBar()}<div class="card"><h2 style="font-size:22px">${state.devices.length > 1 ? L("Choose a device", "اختار الموبايل") : L("iPhone detected", "تم العثور على الآيفون")}</h2>
      ${state.devices.map(deviceCard).join("")}</div>`;
    bindModeBar(pane, renderDevices);
    pane.querySelectorAll("[data-pair]").forEach((b) => b.addEventListener("click", () => pairLoop(b.dataset.pair)));
    pane.querySelectorAll("[data-scan]").forEach((b) => b.addEventListener("click", () => startScan({ udid: b.dataset.scan })));
  }

  const pairText = (st, fallback) => ({
    waiting: L('Tap "Trust" on the iPhone and enter its passcode.', 'دوس "Trust" على الآيفون واكتب الباسكود.'),
    locked: L("Unlock the iPhone (enter its passcode), then keep it on the Home Screen.", "افتح قفل الآيفون واكتب الباسكود، وسيبه على الشاشة الرئيسية."),
    denied: L('"Don\'t Trust" was selected. Unplug and reconnect the cable, then tap "Trust".', 'اتداس "Don\'t Trust". شيل الكابل ووصّله تاني، ودوس "Trust".'),
    unpaired: L("The iPhone must trust this computer before it can be scanned.", "لازم الآيفون يوثق في الكمبيوتر ده قبل الفحص."),
    error: fallback,
  }[st] || fallback);

  function deviceCard(d) {
    const p = state.pairing[d.udid];
    const paired = d.paired || (p && p.state === "paired");
    const pairing = p && p.busy;
    let action, note = "";
    if (paired) {
      action = `<button class="btn primary big" data-scan="${esc(d.udid)}">${L("Start full scan", "ابدأ الفحص الكامل")}</button>`;
    } else if (pairing) {
      action = `<button class="btn" disabled><span class="spinner"></span> ${L("Waiting…", "مستنيين…")}</button>`;
      note = `<div class="trust-box">${esc(pairText(p.state || "waiting", p.message))}</div>`;
    } else {
      action = `<button class="btn primary" data-pair="${esc(d.udid)}">${L("Trust this computer", "وثّق في الكمبيوتر")}</button>`;
      note = `<div class="trust-box">${esc(pairText((p && p.state) || d.pairState || "unpaired", (p && p.message) || d.pairMessage))}</div>`;
    }
    return `<div class="device-card">
        ${phoneSVG(38, false)}
        <div class="meta">
          <h3>${esc(d.modelName || d.productType || "iPhone")}</h3>
          <div class="sub">${d.productVersion ? "iOS " + esc(d.productVersion) + " · " : ""}${esc(d.connectionType || "USB")} · <span class="mono">${esc(d.udid)}</span></div>
          <div class="sub" style="margin-top:4px"><span class="dot ${paired ? "ok" : "wait"}" style="display:inline-block;margin-inline-end:6px"></span>${paired ? L("Trusted — ready to scan", "موثوق — جاهز للفحص") : L("Not trusted yet", "لسه مش موثوق")}</div>
          ${note}
        </div>
        ${action}
      </div>`;
  }

  async function pairLoop(udid) {
    const p = (state.pairing[udid] = { busy: true, state: "waiting" });
    renderDevices();
    for (let i = 0; i < 45 && location.hash.startsWith("#/iphone"); i++) {
      try {
        const r = await api("/api/ios/pair", { udid });
        p.state = r.state;
        p.message = r.message;
        if (r.state === "paired") { p.busy = false; toast(L("iPhone trusts this computer", "الآيفون وثق في الكمبيوتر")); break; }
        if (r.state === "denied" || r.state === "error") { p.busy = false; break; }
      } catch (e) { p.message = String(e); }
      renderDevices();
      await new Promise((res) => setTimeout(res, 2500));
    }
    p.busy = false;
    pollDevices();
  }

  // ------------------------------------------------------------ scan
  async function startScan(req) {
    try {
      await ensureTests();
      await api(P() + "/scan", Object.assign({ mode: state.mode }, req));
      if (state.manualFor !== "pending") { state.manual = {}; state.notes = ""; }
      state.manualFor = req.udid || req.demo;
      state.tab = "summary";
      go("#/scan");
    } catch (e) { toast(L("Could not start scan: ", "ماقدرناش نبدأ الفحص: ") + e.message); }
  }

  const STEP_LABELS = () => state.platform === "android" ? (isAr()
    ? ["قراءة بيانات الجهاز", "قراءة البطارية", "فحص الرامات والمساحة والشاشة", "فحص الحسابات والأقفال", "فحص التطبيقات", "البحث عن روت وبوت لودر مفتوح", "فحص الحساسات والكاميرات", "قراءة سجل الأعطال", "تجهيز النتيجة"]
    : ["Reading device identity", "Reading battery", "Checking memory, storage & screen", "Checking accounts & locks", "Auditing installed apps", "Checking for root & bootloader unlock", "Testing sensors & cameras", "Reading crash history", "Evaluating results"])
    : isAr()
    ? ["قراءة بيانات الجهاز", "قراءة البطارية", "فحص الهاردوير", "قياس المساحة", "فحص التطبيقات", "فحص الإدارة وملفات التعريف", "البحث عن جيلبريك", "قراءة سجل الأعطال", "تجهيز النتيجة"]
    : ["Reading device identity", "Reading battery gauge", "Running hardware diagnostics", "Measuring storage",
      "Auditing installed apps", "Checking management & profiles", "Probing for jailbreak services", "Analysing crash & panic logs", "Evaluating results"];

  function viewScan() {
    crumbs([HOME(), { label: platformName(), href: connectHash() }, { label: L("Scanning", "جاري الفحص") }]);
    $app.innerHTML = `<div class="scan fade-in">
        <div class="scan-ring" id="scanRing"></div>
        <div><h2 style="font-size:24px">${L("Scanning the iPhone", "جاري فحص الآيفون")}</h2>
          <p class="muted" style="margin:6px 0 16px">${L("Keep the phone connected and unlocked. This usually takes under a minute.", "سيب الموبايل متوصّل ومفتوح. الفحص بياخد أقل من دقيقة.")}</p>
          <div class="step-list" id="stepList"></div>
          <div id="scanErr"></div></div>
      </div>`;
    every(pollScan, 450);
  }

  async function pollScan() {
    let s;
    try { s = await api(withLang(P() + "/scan")); } catch (e) { return; }
    state.scan = s;
    const labels = STEP_LABELS();
    const total = s.total || labels.length;
    const pct = s.status === "done" ? 100 : Math.round(((s.step || 0) - 0.5) / total * 100);
    const rEl = document.getElementById("scanRing");
    if (!rEl) return;
    const cur = s.status === "done" ? labels[labels.length - 1] : labels[Math.max(0, (s.step || 1) - 1)] || "";
    rEl.innerHTML = ring(Math.max(pct, 3), 260, 16, "url(#scanGrad)",
      `<defs><linearGradient id="scanGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6366f1"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></defs>`) +
      `<div class="num"><div><b>${Math.max(0, pct)}%</b><span>${esc(cur)}</span></div></div>`;
    document.getElementById("stepList").innerHTML = labels.map((l, i) => {
      const n = i + 1;
      const cls = s.status === "done" || n < s.step ? "done" : n === s.step ? "active" : "";
      const ic = cls === "done" ? "✓" : cls === "active" ? '<span class="spinner"></span>' : "○";
      return `<div class="step ${cls}"><span class="ic">${ic}</span>${esc(l)}</div>`;
    }).join("");
    if (s.status === "done") {
      clearTimers();
      state.result = s.result;
      if (Object.values(state.manual).some((m) => m.status) || state.notes) {
        try {
          const results = Object.entries(state.manual).filter(([, m]) => m.status).map(([id, m]) => ({ id, status: m.status, note: m.note || "" }));
          const res = await api(withLang(P() + "/manual"), { results, notes: state.notes });
          if (res) state.result = res;
        } catch (e) { /* keep automatic result */ }
      }
      setTimeout(() => go("#/result"), 350);
    } else if (s.status === "error") {
      clearTimers();
      document.getElementById("scanErr").innerHTML = `<div class="notice" style="margin-top:16px"><h3>${L("Scan failed", "الفحص ماكملش")}</h3>
        <p class="muted mono">${esc(s.error)}</p><p class="muted">${L("Make sure the iPhone is unlocked, trusted and still connected, then try again.", "اتأكد إن الآيفون مفتوح وموثوق ولسه متوصّل، وجرّب تاني.")}</p>
        <div class="row" style="margin-top:10px"><a class="btn primary" href="${connectHash()}">${L("Back to device", "رجوع للموبايل")}</a></div></div>`;
    }
  }

  // ------------------------------------------------------------ result
  async function viewResult() {
    if (!state.result || state.result.lang !== (isAr() ? "ar" : undefined)) {
      try { const s = await api(withLang(P() + "/scan")); if (s.result) state.result = s.result; } catch (e) { /* ignore */ }
    }
    if (!state.result) return go(connectHash());
    if (state.result.platform) state.platform = state.result.platform;
    await ensureTests();
    crumbs([HOME(), { label: platformName(), href: connectHash() }, { label: state.result.modelName }]);
    renderResult();
  }

  function idv(key) {
    const it = (state.result.identity || []).find((k) => (k.key || k.label) === key);
    return it ? it.value : "";
  }

  function renderResult() {
    const r = state.result;
    const c = r.counts || {};
    const color = COLOR[r.verdict] || COLOR.risk;
    const checks = r.checks || [];
    const autoN = checks.filter((x) => x.category !== "Hands-on tests").length;
    const fails = checks.filter((x) => x.status === "fail").length;
    const handTests = testsFor(r.mode || "used");
    const handsDone = doneCount(handTests);
    const handsName = r.mode === "new" ? (r.platform === "android" ? L("New-phone checklist", "قائمة الموبايل الجديد") : L("New-iPhone checklist", "قائمة الآيفون الجديد")) : L("Hands-on tests", "الاختبارات اليدوية");
    const sub = (r.platform === "android"
      ? [idv("Android") && "Android " + idv("Android").split(" ")[0], idv("Capacity"), idv("RAM") && idv("RAM") + " RAM", idv("Model number"), idv("Region / CSC")]
      : [idv("iOS") && "iOS " + idv("iOS"), idv("Capacity"), idv("Model number"), idv("Part number")]).filter(Boolean).join(" · ");
    $app.innerHTML = `
      <section class="card res-head fade-in">
        <div style="position:relative;width:150px;height:150px">${ring(r.score, 150, 13, color)}
          <div style="position:absolute;inset:0;display:grid;place-items:center;text-align:center"><div><b style="font-size:40px;line-height:1">${r.score}</b><div class="muted" style="font-size:12px">${L("score", "الدرجة")}</div></div></div></div>
        <div>
          <h1>${esc(r.modelName)}${r.snapshot && r.snapshot.demo ? `<span class="demo-flag">${L("DEMO", "مثال")}</span>` : ""}<span class="mode-flag ${r.mode === "new" ? "new" : ""}">${r.mode === "new" ? L("NEW-DEVICE MODE", "وضع الجهاز الجديد") : L("USED-DEVICE MODE", "وضع الجهاز المستعمل")}</span></h1>
          <div class="sub">${bdi(sub)}</div>
          <div class="verdict ${esc(r.verdict)}">${r.verdict === "good" ? "✓" : r.verdict === "caution" ? "!" : "✕"} ${esc(r.verdictText)}</div>
          <div class="counts">
            <span class="count"><i class="st pass" style="width:10px;height:10px"></i><b>${c.pass || 0}</b> ${L("passed", "سليم")}</span>
            <span class="count"><i class="st warn" style="width:10px;height:10px"></i><b>${c.warn || 0}</b> ${L("warnings", "تحذير")}</span>
            <span class="count"><i class="st fail" style="width:10px;height:10px"></i><b>${c.fail || 0}</b> ${L("failed", "مشكلة")}</span>
            <span class="count"><i class="st info" style="width:10px;height:10px"></i><b>${c.info || 0}</b> ${L("info", "معلومة")}</span>
            <span class="count">✋ <b>${handsDone}/${handTests.length}</b> ${r.mode === "new" ? L("checklist", "من القائمة") : L("hands-on", "يدوي")}</span>
          </div>
        </div>
        <div class="head-actions">
          <button class="btn primary" id="saveHtml">${L("Save report", "احفظ التقرير")}</button>
          <div class="row"><button class="btn small" id="saveJson">${L("Export JSON", "تصدير JSON")}</button><button class="btn small" data-open="reports">${L("Reports folder", "فولدر التقارير")}</button></div>
          <a class="btn small" href="${connectHash()}">${L("New scan", "فحص جديد")}</a>
        </div>
      </section>
      <nav class="tabs" id="tabs">
        ${[["summary", L("Summary", "الملخص")], ["checks", L("All checks", "كل الفحوصات"), autoN, fails], ["battery", L("Battery", "البطارية")], ["hands", handsName, `${handsDone}/${handTests.length}`], ["device", L("Device & specs", "الجهاز والمواصفات")], ["apps", L("Apps", "التطبيقات"), (r.snapshot.apps || []).length], ["raw", L("Raw data", "البيانات الخام")]]
          .map(([id, label, n, f]) => `<button class="tab ${state.tab === id ? "active" : ""}" data-tab="${id}">${label}${n !== undefined ? `<span class="n ${f ? "f" : ""}">${n}</span>` : ""}</button>`).join("")}
      </nav>
      <section id="tabBody" class="fade-in"></section>`;
    document.getElementById("tabs").addEventListener("click", (e) => {
      const b = e.target.closest("[data-tab]");
      if (!b) return;
      state.tab = b.dataset.tab;
      document.querySelectorAll(".tab").forEach((t) => t.classList.toggle("active", t.dataset.tab === state.tab));
      renderTab();
    });
    document.getElementById("saveHtml").onclick = () => saveReport("html");
    document.getElementById("saveJson").onclick = () => saveReport("json");
    bindOpen($app.querySelector(".head-actions"));
    renderTab();
  }

  async function saveReport(format) {
    try {
      const r = await api("/api/report/save", { format, open: format === "html", lang: state.lang, platform: state.platform });
      if (r.ok) toast(L("Saved: ", "اتحفظ: ") + r.path); else toast(L("Could not save: ", "ماقدرناش نحفظ: ") + r.error);
    } catch (e) { toast(L("Could not save: ", "ماقدرناش نحفظ: ") + e.message); }
  }

  function renderTab() {
    const body = document.getElementById("tabBody");
    const fn = { summary: tabSummary, checks: tabChecks, battery: tabBattery, hands: tabHands, device: tabDevice, apps: tabApps, raw: tabRaw }[state.tab] || tabSummary;
    body.innerHTML = fn();
    body.classList.remove("fade-in"); void body.offsetWidth; body.classList.add("fade-in");
    state.rerender = renderResult;
    if (state.tab === "hands") bindHands();
    if (state.tab === "apps") bindApps();
    if (state.tab === "summary") {
      body.querySelectorAll("[data-cat]").forEach((el) => el.addEventListener("click", () => {
        state.tab = el.dataset.cat === "Hands-on tests" ? "hands" : "checks";
        renderResult();
        setTimeout(() => { const t = document.getElementById("cat-" + slug(el.dataset.cat)); if (t) t.scrollIntoView({ behavior: "smooth" }); }, 50);
      }));
      bindOpen(body);
      const oh = document.getElementById("openHands");
      if (oh) oh.onclick = () => document.querySelector("[data-tab=hands]").click();
    }
    body.querySelectorAll("[data-copy]").forEach((b) => b.addEventListener("click", () => {
      navigator.clipboard.writeText(b.dataset.copy).then(() => toast(L("Copied", "اتنسخ")), () => toast(L("Copy failed", "النسخ ماشتغلش")));
    }));
  }
  const slug = (s) => s.toLowerCase().replace(/[^a-z0-9]+/g, "-");

  function checkRow(c, open) {
    const hasBody = c.expected || c.actual || c.advice;
    return `<details class="check" ${open ? "open" : ""}>
      <summary><span class="st ${esc(c.status)}">${ICON[c.status] || "i"}</span>
        <div class="grow"><div class="t">${esc(c.title)}</div><div class="s">${esc(c.summary)}</div></div>
        ${c.status === "fail" || c.status === "warn" ? `<span class="sev">${esc(sevName(c.severity))}</span>` : ""}
      </summary>
      ${hasBody ? `<div class="body">
        ${c.expected || c.actual ? `<div class="ea">${c.expected ? `<span>${L("Expected", "المفروض")}</span><span>${bdi(c.expected)}</span>` : ""}${c.actual ? `<span>${L("Found", "الموجود")}</span><span class="mono">${esc(c.actual)}</span>` : ""}</div>` : ""}
        ${c.advice ? `<div class="advice">${esc(c.advice)}</div>` : ""}</div>` : ""}
    </details>`;
  }

  function tabSummary() {
    const r = state.result;
    const checks = r.checks || [];
    const issues = checks.filter((c) => c.status === "fail" || c.status === "warn")
      .sort((a, b) => ORDER[a.status] - ORDER[b.status] || SEV[a.severity] - SEV[b.severity]);
    const cats = (r.mode === "new" ? ["New-device checks"] : []).concat(["Identity & authenticity", "Ownership & locks", "Software integrity", "Battery", "Hardware & diagnostics", "Hands-on tests"]);
    const b = r.battery || {};
    const st = r.storage || {};
    const usedPct = st.totalBytes && st.usedBytes ? Math.round(st.usedBytes / st.totalBytes * 100) : 0;
    const hTests = testsFor(r.mode || "used");
    const handsLeft = hTests.length - doneCount(hTests);
    const isNew = r.mode === "new";
    return `
      <div class="grid2">
        <div class="card">
          <h3>${issues.length ? L("Things to look at", "حاجات لازم تاخد بالك منها") : L("No problems found", "مفيش أي مشاكل")}</h3>
          <p class="muted" style="margin:0 0 6px">${issues.length ? L("Sorted by importance. Click a category below for every check.", "مترتبة حسب الأهمية. دوس على أي قسم تحت علشان تشوف كل الفحوصات.") : L("All automated checks passed. Complete the hands-on tests before you pay.", "كل الفحوصات الأوتوماتيك سليمة. كمّل الاختبارات اليدوية قبل ما تدفع.")}</p>
          ${issues.slice(0, 8).map((c) => `<div class="issue"><span class="st ${c.status}">${ICON[c.status]}</span><div><div class="t">${esc(c.title)}</div><div class="s">${esc(c.summary)}</div>${c.advice ? `<div class="s" style="color:#bfdbfe;margin-top:4px">${ARROW()} ${esc(c.advice)}</div>` : ""}</div></div>`).join("")}
          ${issues.length > 8 ? `<p class="muted">+ ${issues.length - 8} ${L("more in “All checks”.", "كمان في \"كل الفحوصات\".")}</p>` : ""}
        </div>
        <div style="display:grid;gap:18px;align-content:start">
          <div class="grid2">
            <div class="stat"><span>${L("Battery health", "صحة البطارية")}</span><b>${b.available && b.healthPercent ? Math.round(b.healthPercent) + "%" : "—"}</b><small>${!b.available ? L("not readable", "مش متاحة") : r.platform === "android" && !b.cyclesKnown ? L("cycles not shared by phone", "الدورات مش متاحة") : b.cycleCount + " " + L("cycles", "دورة شحن")}</small>
              <div class="meter"><div style="width:${b.healthPercent || 0}%;background:${(b.healthPercent || 0) >= 85 ? "var(--pass)" : (b.healthPercent || 0) >= 80 ? "var(--warn)" : "var(--fail)"}"></div></div></div>
            <div class="stat"><span>${L("Storage", "المساحة")}</span><b>${bdi(st.totalLabel || "—")}</b><small>${st.usedLabel ? bdi(st.usedLabel) + " " + L("used", "مستخدمة") : ""}</small>
              <div class="meter"><div style="width:${usedPct}%;background:var(--grad)"></div></div></div>
          </div>
          ${handsLeft > 0 ? `<div class="card" style="border-color:rgba(129,140,248,.4)"><h3>${isNew ? "📦" : "✋"} ${isNew ? (r.platform === "android" ? L(`${handsLeft} new-phone checklist items left`, `فاضل ${handsLeft} بند في قائمة الموبايل الجديد`) : L(`${handsLeft} new-iPhone checklist items left`, `فاضل ${handsLeft} بند في قائمة الآيفون الجديد`)) : L(`${handsLeft} hands-on tests left`, `فاضل ${handsLeft} اختبار يدوي`)}</h3>
            <p class="muted" style="margin:0 0 12px">${isNew ? L("Box seal, box label vs phone, Apple warranty status, first start and more can only be verified by you.", "برشام العلبة، وبيانات العلبة، والضمان، وأول تشغيل — حاجات لازم تتأكد منها بنفسك.") : (r.platform === "android" ? L("Some things (screen, cameras, fingerprint, speakers, accounts after reset) can only be verified on the phone itself.", "في حاجات (الشاشة، الكاميرات، البصمة، السماعات، الحسابات بعد الضبط) لازم تجرّبها بنفسك على الموبايل.") : L("Some things (screen, cameras, Face ID, speakers, parts history) can only be verified on the phone itself.", "في حاجات (الشاشة، الكاميرات، Face ID، السماعات، سجل القطع) لازم تجرّبها بنفسك على الموبايل."))}</p>
            <button class="btn small primary" id="openHands">${isNew ? L("Open checklist", "افتح القائمة") : L("Start hands-on tests", "ابدأ الاختبارات اليدوية")}</button></div>` : ""}
          <div class="card"><h3>${L("Before you pay", "قبل ما تدفع")}</h3><ul class="muted" style="margin:6px 0 0;padding-inline-start:18px;display:grid;gap:4px;font-size:13px">
            ${r.platform === "android" ? `<li>${L("Dial *#06# and compare the IMEI with the box.", "اطلب ‎*#06#‎ وقارن الـ IMEI بالعلبة.")}</li>
            <li>${L("Make sure all accounts are removed and the phone is factory-reset in front of you.", "اتأكد إن كل الحسابات اتمسحت والموبايل اتعمله ضبط مصنع قدامك.")}</li>` : `<li>${L("Check the serial number's warranty/coverage status with Apple.", "افحص الضمان بالرقم التسلسلي على موقع Apple.")}</li>`}
            <li>${L("Check the IMEI is not blacklisted (lost/stolen) with your carrier.", "اتأكد إن الـ IMEI مش مبلّغ عنه (مسروق أو ضايع).")}</li>
            ${r.platform === "android" ? "" : `<li>${L("Make sure the seller has signed out of their Apple Account and erased the phone.", "اتأكد إن البائع عمل تسجيل خروج من حساب Apple ومسح الموبايل.")}</li>`}</ul>
            <div class="row" style="margin-top:12px">${r.platform === "android" ? "" : `<button class="btn small" data-open="coverage">${L("Apple coverage check", "افحص الضمان")}</button>`}${idv("Serial number") ? `<button class="btn small" data-copy="${esc(idv("Serial number"))}">${L("Copy serial", "انسخ السيريال")}</button>` : ""}${idv("IMEI") ? `<button class="btn small" data-copy="${esc(idv("IMEI"))}">${L("Copy IMEI", "انسخ الـ IMEI")}</button>` : ""}</div></div>
        </div>
      </div>
      <div class="section-title">${L("Categories", "الأقسام")}</div>
      <div class="grid3">${cats.map((cat) => {
        const list = checks.filter((c) => c.category === cat);
        const worst = list.reduce((w, c) => (ORDER[c.status] < ORDER[w] ? c.status : w), "pass");
        const passed = list.filter((c) => c.status === "pass").length;
        const sub = cat === "Hands-on tests" && !list.length ? L("Not started", "لسه مابدأتش") : L(`${passed}/${list.length} passed`, `${passed} من ${list.length} سليم`);
        return `<div class="card cat-card" data-cat="${esc(cat)}"><div class="cat-icon">${CAT_ICON[cat] || "•"}</div>
          <div style="flex:1;min-width:0"><div class="row" style="justify-content:space-between"><b>${esc(catName(cat, r.mode))}</b>${list.length ? `<span class="st ${worst}" style="width:20px;height:20px;font-size:10px">${ICON[worst]}</span>` : ""}</div>
          <div class="muted" style="font-size:12.5px">${sub}</div><div class="bar">${list.map((c) => `<i class="${c.status}"></i>`).join("")}</div></div></div>`;
      }).join("")}</div>`;
  }

  function tabChecks() {
    const checks = state.result.checks || [];
    const cats = [...new Set(checks.map((c) => c.category))];
    const errs = state.result.snapshot.errors || {};
    return cats.map((cat) => {
      const list = checks.filter((c) => c.category === cat).sort((a, b) => ORDER[a.status] - ORDER[b.status]);
      return `<div class="cat-heading" id="cat-${slug(cat)}">${CAT_ICON[cat] || ""} ${esc(catName(cat, state.result.mode))}</div><div class="check-list">${list.map((c) => checkRow(c, c.status === "fail")).join("")}</div>`;
    }).join("") + (Object.keys(errs).length ? `<div class="cat-heading">${L("Could not be read", "حاجات ماقدرناش نقراها")}</div><div class="card dim mono">${Object.entries(errs).map(([k, v]) => `<div>${esc(k)}: ${esc(v)}</div>`).join("")}</div>` : "");
  }

  function tabBattery() {
    const b = state.result.battery || {};
    if (!b.available) return `<div class="card"><h3>${L("Battery data not available", "بيانات البطارية مش متاحة")}</h3><p class="muted">${L("The battery gauge could not be read on this device/iOS version. Check <b>Settings › Battery › Battery Health</b> on the phone and record it in the hands-on tests.", "ماقدرناش نقرا البطارية في الموبايل ده. افتح <b>الإعدادات › البطارية › صحة البطارية</b> على الموبايل وسجّلها في الاختبارات اليدوية.")}</p></div>`;
    const h = b.healthPercent || 0;
    const col = h >= 85 ? "#22c55e" : h >= 80 ? "#f59e0b" : "#ef4444";
    const cyc = b.ratedCycles ? Math.min(100, b.cycleCount / b.ratedCycles * 100) : 0;
    const bc = (state.result.checks || []).filter((c) => c.category === "Battery");
    return `<div class="card gauge-wrap">
        <div style="position:relative;width:220px;height:220px;margin:auto">${ring(h, 220, 18, col)}
          <div style="position:absolute;inset:0;display:grid;place-items:center;text-align:center"><div><b style="font-size:44px;line-height:1">${h ? Math.round(h) + "%" : "—"}</b><div class="muted">${L("maximum capacity", "أقصى سعة")}</div></div></div></div>
        <div>
          <div class="grid2">
            <div class="stat"><span>${L("Charge cycles", "دورات الشحن")}</span><b>${state.result.platform === "android" && !b.cyclesKnown ? "—" : b.cycleCount}</b><small>${L(`rated ${b.ratedCycles} cycles to 80%`, `عمرها الرسمي ${b.ratedCycles} دورة لحد 80%`)}</small><div class="meter"><div style="width:${cyc}%;background:${cyc < 60 ? "var(--pass)" : cyc < 100 ? "var(--warn)" : "var(--fail)"}"></div></div></div>
            <div class="stat"><span>${L("Full charge capacity", "السعة الحالية")}</span><b><bdi>${b.fullChargeCapacity || "—"} mAh</bdi></b><small>${L("design", "سعة المصنع")} ${bdi((b.designCapacity || "—") + " mAh")}${b.expectedDesign ? " · " + L("factory spec", "المواصفات") + " " + bdi(b.expectedDesign.join("/") + " mAh") : ""}</small></div>
            <div class="stat"><span>${L("Temperature", "الحرارة")}</span><b>${bdi(b.temperatureC ? b.temperatureC.toFixed(1) + " °C" : "—")}</b><small>${bdi(b.voltageV ? b.voltageV.toFixed(2) + " V" : "")}</small></div>
            <div class="stat"><span>${L("Current charge", "الشحن الحالي")}</span><b>${b.chargePercent ? b.chargePercent + "%" : "—"}</b><small>${b.isCharging ? L("charging", "بيشحن") : b.externalConnected ? L("plugged in", "متوصّل بالشاحن") : L("on battery", "على البطارية")}</small></div>
          </div>
          ${b.serial ? `<p class="muted" style="margin-top:12px;font-size:12.5px">${L("Battery gauge serial", "رقم شريحة البطارية")}: <span class="mono">${esc(b.serial)}</span></p>` : ""}
          <p class="dim" style="font-size:12px">${state.result.platform === "android"
            ? L("Some Android phones do not share cycles or capacity over USB — check the battery screen in the hands-on tests.", "بعض موبايلات أندرويد مش بتدّي الدورات أو السعة بالكابل — شوف شاشة البطارية في الاختبارات اليدوية.")
            : L("Apple rates iPhone 15 and later at 1000 cycles, earlier models at 500 cycles, to retain 80% capacity.", "حسب Apple: من آيفون 15 وطالع البطارية بتستحمل 1000 دورة، والأقدم 500 دورة، قبل ما تنزل لـ 80%.")}</p>
        </div></div>
      <div class="section-title">${L("Battery checks", "فحوصات البطارية")}</div><div class="check-list">${bc.map((c) => checkRow(c, c.status !== "pass")).join("")}</div>`;
  }

  // hands-on / checklist
  function checklistHTML(tests, opts) {
    const done = doneCount(tests);
    let lastGroup = "";
    return `<div class="card" style="margin-bottom:16px"><div class="row" style="justify-content:space-between"><div><h3>${esc(opts.title)}</h3><p class="muted" style="margin:0">${esc(opts.intro)}</p></div>
        <div class="row"><b style="font-size:18px">${done}/${tests.length}</b><button class="btn small" id="resetChecklist">${L("Reset", "امسح")}</button></div></div>
      <div class="progress-line" style="margin-top:12px"><div style="width:${done / Math.max(1, tests.length) * 100}%"></div></div></div>
      <div class="check-list">${tests.map((t) => {
        const m = state.manual[t.id] || {};
        const grp = tGroup(t);
        const g = grp !== lastGroup ? `<div class="group-label" style="margin-top:${lastGroup ? 14 : 0}px">${esc(grp)}</div>` : "";
        lastGroup = grp;
        return `${g}<div class="card mt ${m.status ? "marked-" + m.status : ""}" data-mt="${esc(t.id)}"><div><b>${esc(tTitle(t))}</b> <span class="dim" style="font-size:11px;text-transform:uppercase;letter-spacing:.08em;margin-inline-start:6px">${esc(sevName(t.severity))}</span>
          <ol>${tSteps(t).map((st) => `<li>${esc(st)}</li>`).join("")}</ol>
          ${t.link ? `<button class="btn small" data-open="${esc(t.link)}" style="margin-top:8px">${L("Open Apple coverage check ↗", "افتح موقع الضمان ↗")}</button>` : ""}
          <input type="text" placeholder="${L("Note (optional)", "ملاحظة (اختياري)")}" value="${esc(m.note || "")}" data-note="${esc(t.id)}"></div>
          <div class="seg">${["pass", "fail", "skip"].map((st) => `<button class="${st} ${m.status === st ? "on" : ""}" data-set="${st}">${st === "pass" ? L("✓ Pass", "✓ سليم") : st === "fail" ? L("✕ Fail", "✕ فيه مشكلة") : L("Skip", "تخطّي")}</button>`).join("")}</div></div>`;
      }).join("")}</div>
      <div class="section-title">${L("Your notes for the report", "ملاحظاتك للتقرير")}</div>
      <textarea id="notes" placeholder="${L("e.g. small scratch on the back glass, original box included, agreed price…", "مثلاً: خدش صغير في الضهر، العلبة الأصلية موجودة، السعر المتفق عليه…")}">${esc(state.notes)}</textarea>`;
  }

  function tabHands() {
    const r = state.result;
    return checklistHTML(testsFor(r.mode || "used"), r.mode === "new"
      ? { title: r.platform === "android" ? L("New-phone checklist", "قائمة الموبايل الجديد") : L("New-iPhone checklist", "قائمة الآيفون الجديد"), intro: L("Everything to confirm before paying for a phone sold as brand new. Results are added to the score and the report.", "كل اللي لازم تتأكد منه قبل ما تدفع في موبايل جديد. النتيجة بتتحسب في الدرجة وفي التقرير.") }
      : { title: L("Guided hands-on tests", "الاختبارات اليدوية"), intro: L("Do each test on the phone and mark the result. Results are added to the score and the report.", "اعمل كل اختبار على الموبايل وعلّم النتيجة. النتيجة بتتحسب في الدرجة وفي التقرير.") });
  }

  let manualTimer;
  function pushManual(rerender) {
    clearTimeout(manualTimer);
    manualTimer = setTimeout(async () => {
      try {
        const results = Object.entries(state.manual).filter(([, m]) => m.status).map(([id, m]) => ({ id, status: m.status, note: m.note || "" }));
        const res = state.result ? await api(withLang(P() + "/manual"), { results, notes: state.notes }) : null;
        if (res) state.result = res;
        if (rerender) (state.rerender || renderResult)();
      } catch (e) { toast(L("Could not save test result: ", "ماقدرناش نحفظ النتيجة: ") + e.message); }
    }, rerender ? 0 : 400);
  }

  function bindHands() {
    document.querySelectorAll("[data-mt]").forEach((card) => {
      const id = card.dataset.mt;
      card.querySelectorAll("[data-set]").forEach((b) => b.addEventListener("click", () => {
        const m = (state.manual[id] = state.manual[id] || {});
        m.status = m.status === b.dataset.set ? "" : b.dataset.set;
        const y = window.scrollY;
        pushManual(true);
        setTimeout(() => window.scrollTo(0, y), 60);
      }));
    });
    document.querySelectorAll("[data-note]").forEach((inp) => inp.addEventListener("input", () => {
      const m = (state.manual[inp.dataset.note] = state.manual[inp.dataset.note] || {});
      m.note = inp.value;
      pushManual(false);
    }));
    const n = document.getElementById("notes");
    if (n) n.addEventListener("input", () => { state.notes = n.value; pushManual(false); });
    const rs = document.getElementById("resetChecklist");
    if (rs) rs.addEventListener("click", () => { if (confirm(L("Clear all checklist results?", "تمسح كل النتايج؟"))) { state.manual = {}; state.notes = ""; pushManual(true); } });
    bindOpen(document.getElementById("app"));
  }

  // device & specs
  function tabDevice() {
    const r = state.result;
    const kv = (r.identity || []).map((k) => `<div class="kv"><span>${esc(k.label)}</span><span class="${k.mono ? "mono" : ""}">${bdi(k.value)}${k.mono ? `<button class="copy" title="${L("Copy", "نسخ")}" data-copy="${esc(k.value)}">⧉</button>` : ""}</span></div>`).join("");
    const s = r.spec;
    let specHtml = `<div class="card"><h3>${L("Not in the spec database", "الموديل مش موجود في قاعدة المواصفات")}</h3><p class="muted">${L("Add a JSON file for", "ضيف ملف JSON لـ")} <span class="mono">${esc(idv("Identifier") || idv("Model number"))}</span> ${L("to a <span class=\"mono\">specs</span> folder next to the app to enable spec cross-checks.", "في فولدر <span class=\"mono\">specs</span> جنب البرنامج علشان يقارن المواصفات.")}</p></div>`;
    if (s && r.platform === "android") {
      const pr = r.snapshot.props || {};
      const rows = [
        [L("Model numbers", "أرقام الموديل"), (s.models || []).join(", "), pr["ro.product.model"]],
        [L("Codename", "الاسم الكودي"), (s.devices || []).join(", "), pr["ro.product.device"]],
        [L("Chip", "المعالج"), [s.chip, (s.boardPlatforms || []).join("/")].filter(Boolean).join(" · "), [pr["ro.board.platform"], pr["ro.soc.model"]].filter(Boolean).join(" · ")],
        [L("RAM", "الرامات"), (s.ramGB || []).map((g) => g + " GB").join(" / "), idv("RAM")],
        [L("Storage options", "المساحات المتاحة"), (s.storageOptionsGB || []).map((g) => (g >= 1024 ? g / 1024 + " TB" : g + " GB")).join(" / "), idv("Capacity")],
        [L("Display", "الشاشة"), [s.display?.sizeInches && s.display.sizeInches + '"', s.display?.technology, s.display?.resolution, s.display?.maxRefreshHz && s.display.maxRefreshHz + " Hz"].filter(Boolean).join(" · "), r.snapshot.screenSize],
        [L("Battery (design)", "البطارية (سعة المصنع)"), [(s.battery?.designCapacityMah || []).map((x) => x + " mAh").join(" / "), s.battery?.typicalMah && "typ. " + s.battery.typicalMah + " mAh"].filter(Boolean).join(" · "), r.battery?.designCapacity ? r.battery.designCapacity + " mAh" : ""],
        [L("Android", "أندرويد"), (s.launchAndroid ? "Android " + s.launchAndroid : "") + (s.maxAndroid ? " – " + s.maxAndroid : L(" – current", " – لحد دلوقتي")), idv("Android")],
        [L("Update policy", "سياسة التحديثات"), s.updatePolicy, ""],
        [L("Biometrics", "البصمة"), s.biometrics, ""],
        [L("Port", "المنفذ"), s.port, ""],
        [L("Cellular", "الشبكة"), [s.cellular?.generation, s.cellular?.sim].filter(Boolean).join(" · "), ""],
        [L("Rear cameras", "الكاميرات الخلفية"), (s.cameras?.rear || []).join(" · "), ""],
        [L("Front camera", "الكاميرا الأمامية"), s.cameras?.front, ""],
        [L("Released", "تاريخ النزول"), s.releaseDate, ""],
        [L("Features", "المميزات"), (s.features || []).join(", "), ""],
      ].filter((x) => x[1]);
      specHtml = `<div class="card"><div class="row" style="justify-content:space-between"><h3>${esc(s.name)} — ${L("factory specification", "مواصفات المصنع")}</h3><span class="badge">${esc(s.origin || "")}</span></div>
        <table class="cmp"><thead><tr><th>${L("Item", "البند")}</th><th>${L("Factory spec", "المواصفات")}</th><th>${L("Read from device", "من الموبايل")}</th></tr></thead><tbody>
        ${rows.map(([a, b, c]) => `<tr><td>${esc(a)}</td><td>${bdi(b)}</td><td class="mono">${esc(c || "")}</td></tr>`).join("")}</tbody></table></div>`;
    } else if (s) {
      const ld = r.snapshot.lockdown || {};
      const rows = [
        [L("Identifier", "المعرّف"), (s.productTypes || []).join(", "), ld.ProductType],
        [L("Board", "اللوحة الأم"), (s.hardwareModels || []).join(", "), ld.HardwareModel],
        [L("Model numbers", "أرقام الموديل"), (s.regulatoryModels || []).join(", "), ld.RegulatoryModelNumber],
        [L("Chip", "المعالج"), s.chip, ld.ChipID !== undefined ? "0x" + Number(ld.ChipID).toString(16).toUpperCase() : ""],
        [L("Storage options", "المساحات المتاحة"), (s.storageOptionsGB || []).map((g) => (g >= 1024 ? g / 1024 + " TB" : g + " GB")).join(" / "), idv("Capacity")],
        [L("Battery (design)", "البطارية (سعة المصنع)"), (s.battery?.designCapacityMah || []).map((x) => x + " mAh").join(" / "), r.battery?.designCapacity ? r.battery.designCapacity + " mAh" : ""],
        [L("iOS support", "نسخ iOS المدعومة"), (s.launchIOS ? "iOS " + s.launchIOS : "") + (s.maxIOS ? " – " + s.maxIOS + ".x" : L(" – current", " – لحد دلوقتي")), idv("iOS")],
        [L("RAM", "الرامات"), s.ramGB ? s.ramGB + " GB" : "", ""],
        [L("Display", "الشاشة"), [s.display?.sizeInches && s.display.sizeInches + '"', s.display?.technology, s.display?.resolution, s.display?.maxRefreshHz && s.display.maxRefreshHz + " Hz"].filter(Boolean).join(" · "), ""],
        [L("Biometrics", "البصمة"), s.biometrics, ""],
        [L("Port", "المنفذ"), [s.port, s.usbSpec].filter(Boolean).join(" · "), ""],
        [L("Cellular", "الشبكة"), [s.cellular?.generation, s.cellular?.sim].filter(Boolean).join(" · "), ""],
        [L("Rear cameras", "الكاميرات الخلفية"), (s.cameras?.rear || []).join(" · "), ""],
        [L("Front camera", "الكاميرا الأمامية"), s.cameras?.front, ""],
        [L("Colours", "الألوان"), (s.colors || []).join(", "), idv("Colour code") ? L("code ", "كود ") + idv("Colour code") : ""],
        [L("Released", "تاريخ النزول"), s.releaseDate, ""],
        [L("Features", "المميزات"), (s.features || []).join(", "), ""],
      ].filter((x) => x[1]);
      specHtml = `<div class="card"><div class="row" style="justify-content:space-between"><h3>${esc(s.name)} — ${L("factory specification", "مواصفات المصنع")}</h3><span class="badge">${esc(s.origin || "")}</span></div>
        <table class="cmp"><thead><tr><th>${L("Item", "البند")}</th><th>${L("Factory spec", "المواصفات")}</th><th>${L("Read from device", "من الموبايل")}</th></tr></thead><tbody>
        ${rows.map(([a, b, c]) => `<tr><td>${esc(a)}</td><td>${bdi(b)}</td><td class="mono">${esc(c || "")}</td></tr>`).join("")}</tbody></table></div>`;
    }
    return `<div class="card"><h3>${L("Device identity", "بيانات الجهاز")}</h3><div class="kv-grid">${kv}</div></div><div class="section-title">${L("Specification", "المواصفات")}</div>${specHtml}`;
  }

  function tabApps() {
    const apps = state.result.snapshot.apps || [];
    if (!state.result.snapshot.appsScanned) return `<div class="card muted">${L("Installed apps could not be listed.", "ماقدرناش نقرا التطبيقات.")}</div>`;
    return `<div class="card"><div class="row" style="justify-content:space-between;margin-bottom:10px"><h3>${apps.length} ${L("installed apps", "تطبيق متسطّب")}</h3><input type="search" id="appSearch" placeholder="${L("Search apps…", "دوّر على تطبيق…")}" value="${esc(state.appFilter)}" style="width:260px"></div>
      <div id="appRows"></div></div>`;
  }
  function bindApps() {
    const flagged = new Set();
    (state.result.checks || []).filter((c) => (c.id === "jb-apps" || c.id === "sideload") && c.status !== "pass").forEach((c) => (c.actual || "").split(",").forEach((x) => flagged.add(x.trim())));
    const draw = () => {
      const q = state.appFilter.toLowerCase();
      const apps = (state.result.snapshot.apps || []).filter((a) => !q || (a.name + " " + a.bundleId).toLowerCase().includes(q));
      document.getElementById("appRows").innerHTML = apps.map((a) => {
        const flag = [...flagged].some((f) => f.includes(a.bundleId));
        return `<div class="app-row ${flag ? "flag" : ""}"><div><b>${esc(a.name || a.bundleId)}</b> ${flag ? "⚠" : ""}<div class="mono dim">${esc(a.bundleId)}</div></div><div class="dim">${esc(a.version || "")}</div><span class="tag">${esc(a.type || "")}</span></div>`;
      }).join("") || `<p class="muted">${L("No apps match.", "مفيش تطبيقات بالاسم ده.")}</p>`;
    };
    const s = document.getElementById("appSearch");
    s.addEventListener("input", () => { state.appFilter = s.value; draw(); });
    draw();
  }

  function tabRaw() {
    return `<div class="card"><div class="row" style="justify-content:space-between;margin-bottom:10px"><h3>${L("Raw data read from the device", "البيانات الخام من الموبايل")}</h3><button class="btn small" data-copy="${esc(JSON.stringify(state.result.snapshot, null, 2))}">${L("Copy JSON", "انسخ JSON")}</button></div>
      <pre class="raw" dir="ltr">${esc(JSON.stringify(state.result.snapshot, null, 2))}</pre></div>`;
  }

  // ------------------------------------------------------------ new-iPhone checklist (no scan needed)
  async function viewChecklist() {
    crumbs([HOME(), { label: L("New iPhone checklist", "قائمة الآيفون الجديد") }]);
    state.platform = "ios";
    await ensureTests();
    if (state.result && (state.result.mode !== "new" || state.result.platform === "android")) { state.result = null; state.manual = {}; state.notes = ""; }
    if (!state.result) state.manualFor = "pending";
    const draw = () => {
      const tests = testsFor("new");
      $app.innerHTML = `<div class="fade-in">
        <div class="card new-hero"><div><span class="badge live">📦 ${L("New-device mode", "وضع الجهاز الجديد")}</span>
          <h2 style="font-size:26px;margin-top:10px">${L("Buying a brand-new iPhone", "بتشتري آيفون جديد")}</h2>
          <p class="muted" style="margin:6px 0 0;max-width:720px">${L("Go through this list at the shop or with the seller. Start with the box before it is opened. Then connect the phone with a USB cable: the app adds automatic checks that prove it was never used (battery cycles, retail part number, no previous apps or data, clean crash history), and everything you tick here goes into the same report.",
            "امشي على القائمة دي في المحل أو مع البائع. ابدأ بالعلبة قبل ما تتفتح. بعدين وصّل الموبايل بالكابل، والبرنامج هيعمل فحوصات أوتوماتيك تثبت إن الموبايل ما اتستعملش قبل كده (دورات البطارية، رقم القطعة، مفيش تطبيقات أو بيانات لحد قبلك، مفيش أعطال). وكل اللي هتعلّم عليه هنا هيدخل في نفس التقرير.")}</p></div>
          <button class="btn primary big" id="goScan">${L("Connect iPhone & run automatic checks ›", "وصّل الآيفون وابدأ الفحص الأوتوماتيك ‹")}</button></div>
        <div style="margin-top:18px">${checklistHTML(tests, { title: L("New-iPhone checklist", "قائمة الآيفون الجديد"), intro: L("Mark each item Pass or Fail as you check it.", "علّم على كل بند سليم أو فيه مشكلة وانت بتفحصه.") })}</div></div>`;
      document.getElementById("goScan").onclick = () => { setMode("new"); state.manualFor = state.result ? state.manualFor : "pending"; go("#/iphone"); };
      bindHands();
    };
    state.rerender = draw;
    draw();
  }

  // ------------------------------------------------------------ specs browser
  async function viewSpecs(id) {
    crumbs([HOME(), { label: L("Spec database", "قاعدة المواصفات"), href: "#/specs" }].concat(id ? [{ label: id }] : []));
    if (!state.specs) {
      $app.innerHTML = `<div class="card"><div class="row"><span class="spinner"></span>${L("Loading…", "جاري التحميل…")}</div></div>`;
      state.specs = await api("/api/specs");
      const a = await api("/api/android/specs").catch(() => ({ devices: [] }));
      state.specs.android = (a.devices || []).map((d) => Object.assign({ productTypes: d.models, regulatoryModels: [] }, d));
    }
    const devs = (state.specs.devices || []).concat(state.specs.android || []);
    if (id) {
      const d = devs.find((x) => x.id === id);
      if (!d) return go("#/specs");
      $app.innerHTML = `<div class="card fade-in"><div class="row" style="justify-content:space-between"><h2 style="font-size:24px">${esc(d.name)}</h2><span class="badge">${esc(d.origin)}</span></div>
        <p class="muted">${esc((d.productTypes || []).join(", "))} · ${esc(d.chip || "")} · ${L("released", "نزل")} ${esc(d.releaseDate || "?")}</p>
        <pre class="raw" dir="ltr">${esc(JSON.stringify(d, null, 2))}</pre><a class="btn" href="#/specs">${L("‹ Back", "رجوع ›")}</a></div>`;
      return;
    }
    const draw = () => {
      const q = state.specFilter.toLowerCase();
      document.getElementById("specGrid").innerHTML = devs.filter((d) => !q || JSON.stringify([d.name, d.productTypes, d.regulatoryModels, d.chip]).toLowerCase().includes(q))
        .map((d) => `<a class="spec-item" href="#/specs/${encodeURIComponent(d.id)}"><b>${esc(d.name)}</b><span>${esc((d.productTypes || []).join(", "))} · ${esc(d.chip || "")}</span><br><span>${esc((d.releaseDate || "").slice(0, 4))} · ${esc((d.storageOptionsGB || []).map((g) => (g >= 1024 ? g / 1024 + "TB" : g + "GB")).join("/"))}</span></a>`).join("");
    };
    $app.innerHTML = `<div class="fade-in"><div class="row" style="justify-content:space-between;margin-bottom:16px"><div><h2 style="font-size:26px">${L("Device spec database", "قاعدة مواصفات الأجهزة")}</h2>
      <p class="muted" style="margin:4px 0 0">${(state.specs.devices || []).length} iPhone · ${(state.specs.android || []).length} Android · ${L("models. Built into the app; add or override models by placing JSON files in a <span class=\"mono\">specs</span> folder next to the .exe", "موديل جوه البرنامج. تقدر تضيف أو تعدّل موديلات بملفات JSON في فولدر <span class=\"mono\">specs</span> جنب البرنامج")}${state.specs.external ? ` (${L("loaded", "متحمّل")}: <span class="mono">${esc(state.specs.external)}</span>)` : ""}.</p></div>
      <input type="search" id="specSearch" placeholder="${L("Search name, identifier, A-number, chip…", "دوّر بالاسم أو رقم الموديل أو المعالج…")}" style="width:320px" value="${esc(state.specFilter)}"></div>
      ${(state.specs.warnings || []).length ? `<div class="trust-box" style="margin-bottom:14px">${state.specs.warnings.map(esc).join("<br>")}</div>` : ""}
      <div class="spec-list" id="specGrid"></div></div>`;
    const s = document.getElementById("specSearch");
    s.addEventListener("input", () => { state.specFilter = s.value; draw(); });
    draw();
  }

  // ------------------------------------------------------------ global
  document.getElementById("quitBtn").addEventListener("click", async () => {
    if (!confirm(L("Quit Smart Mobile Check?", "تقفل البرنامج؟"))) return;
    try { await api("/api/quit", {}); } catch (e) { /* ignore */ }
    document.body.innerHTML = `<div style="display:grid;place-items:center;height:100vh;color:#93a0b8;font:16px Segoe UI,Tahoma,sans-serif">${L("Smart Mobile Check has been closed. You can close this window.", "البرنامج اتقفل. تقدر تقفل الشباك ده.")}</div>`;
  });
  document.getElementById("langBtn").addEventListener("click", async () => {
    state.lang = isAr() ? "en" : "ar";
    try { localStorage.setItem("smc-lang", state.lang); } catch (e) { /* ignore */ }
    if (state.result) {
      try { const s = await api(withLang(P() + "/scan")); if (s.result) state.result = s.result; } catch (e) { /* ignore */ }
    }
    route();
  });

  route();
})();
