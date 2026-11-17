/* Smart Mobile Check — user interface */
(() => {
  "use strict";

  // ------------------------------------------------------------ helpers
  const $app = document.getElementById("app");
  const $crumbs = document.getElementById("crumbs");
  const esc = (v) => String(v ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const api = async (path, body) => {
    const opt = body === undefined ? {} : { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
    const r = await fetch(path, opt);
    if (!r.ok) throw new Error((await r.text()) || r.statusText);
    return r.json();
  };
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
  const COLOR = { good: "#22c55e", caution: "#f59e0b", risk: "#ef4444" };

  const state = {
    env: null,
    devices: [],
    devError: "",
    pairing: {},      // udid -> {state, message}
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
  };
  let timers = [];
  const every = (fn, ms) => { fn(); timers.push(setInterval(fn, ms)); };
  const clearTimers = () => { timers.forEach(clearInterval); timers = []; };

  const setMode = (m) => { state.mode = m; try { localStorage.setItem("smc-mode", m); } catch (e) { /* ignore */ } };
  const testsFor = (mode) => state.manualTests.filter((t) => (t.modes || ["used", "new"]).includes(mode));
  const doneCount = (tests) => tests.filter((t) => state.manual[t.id] && state.manual[t.id].status).length;
  const modeLabel = (m) => (m === "new" ? "New / sealed iPhone" : "Used iPhone");
  const modeBar = () => `<div class="mode-bar"><span class="muted">I'm buying a</span>
      <div class="seg mode-seg">${["used", "new"].map((m) => `<button class="${state.mode === m ? "on mode" : ""}" data-mode="${m}">${m === "new" ? "New / sealed iPhone" : "Used iPhone"}</button>`).join("")}</div>
      <span class="dim" style="font-size:12px">${state.mode === "new" ? "Adds checks that prove the phone was never used, plus the new-iPhone checklist." : "Full inspection for a second-hand phone."}</span></div>`;
  const bindModeBar = (root, after) => root.querySelectorAll("[data-mode]").forEach((b) => b.addEventListener("click", () => { setMode(b.dataset.mode); after(); }));

  const crumbs = (items) => {
    $crumbs.innerHTML = items.map((it, i) => (i ? '<span class="sep">›</span>' : "") + (it.href ? `<a href="${it.href}">${esc(it.label)}</a>` : `<span>${esc(it.label)}</span>`)).join("");
  };

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
    const [name, arg] = location.hash.replace(/^#\/?/, "").split("/");
    (routes[name] || viewHome)(arg ? decodeURIComponent(arg) : undefined);
    window.scrollTo(0, 0);
  }
  window.addEventListener("hashchange", route);
  const go = (h) => { if (location.hash === h) route(); else location.hash = h; };

  // ------------------------------------------------------------ home
  async function viewHome() {
    crumbs([{ label: "Home" }]);
    $app.innerHTML = `
      <section class="hero fade-in">
        <span class="badge">Used & new phone inspection</span>
        <h1 style="margin-top:14px">Know exactly what you're buying.<br><span class="grad">Before you pay.</span></h1>
        <p>Connect the phone with a USB cable. Smart Mobile Check reads the hardware identity, locks, battery wear and diagnostic logs, flags tampering, then guides you through the hands-on tests.</p>
      </section>
      <section class="platforms fade-in">
        <div class="platform" data-go="#/iphone" tabindex="0" role="button">
          <div class="glow" style="background:#6366f1"></div>
          <span class="badge live"><span class="dot ok"></span> Available</span>
          <span class="go btn small primary">Start ›</span>
          <svg width="38" height="38" viewBox="0 0 24 24" fill="#e8edf7" style="margin-top:18px"><path d="M16.37 1.43c0 1.14-.42 2.2-1.25 3.04-.86.87-1.9 1.37-3.03 1.28a3.4 3.4 0 0 1-.02-.42c0-1.1.48-2.27 1.33-3.1.43-.42.97-.77 1.63-1.05.65-.28 1.27-.43 1.85-.46.02.24.03.48.03.71zM20.6 17.2c-.42.97-.92 1.87-1.5 2.68-.79 1.11-1.43 1.88-1.93 2.3-.77.7-1.6 1.05-2.48 1.07-.63 0-1.4-.18-2.29-.54-.9-.36-1.71-.54-2.46-.54-.78 0-1.62.18-2.52.54-.9.36-1.62.55-2.17.57-.85.04-1.7-.33-2.54-1.1-.54-.46-1.2-1.26-2.01-2.4-.86-1.2-1.57-2.6-2.12-4.2C-.02 13.85-.3 12.2-.3 10.6c0-1.83.4-3.4 1.18-4.73a6.9 6.9 0 0 1 2.48-2.5 6.66 6.66 0 0 1 3.35-.95c.66 0 1.52.2 2.6.6 1.07.4 1.75.6 2.05.6.23 0 1-.24 2.31-.71 1.24-.44 2.28-.62 3.14-.55 2.32.19 4.06 1.1 5.22 2.74-2.08 1.26-3.1 3.02-3.08 5.28.02 1.76.66 3.22 1.9 4.38.57.54 1.2.95 1.9 1.25-.15.44-.32.86-.49 1.27z" transform="translate(1.5 0)"/></svg>
          <h2>iPhone</h2>
          <p>Full automated inspection over USB plus guided hands-on tests.</p>
          <ul>
            <li>Activation Lock, MDM & profile detection</li>
            <li>Jailbreak & tampered-reading detection</li>
            <li>Battery health, cycles & original-battery check</li>
            <li>Model, storage, IMEI & serial authenticity</li>
            <li>Kernel panic & missing-sensor analysis</li>
          </ul>
        </div>
        <div class="platform disabled" data-go="#/android" tabindex="0" role="button">
          <div class="glow" style="background:#22c55e;opacity:.15"></div>
          <span class="badge soon">Coming soon</span>
          <svg width="38" height="38" viewBox="0 0 24 24" fill="#93a0b8" style="margin-top:18px"><path d="M17.6 9.48l1.84-3.18a.38.38 0 0 0-.66-.38l-1.86 3.22a11.4 11.4 0 0 0-9.84 0L5.22 5.92a.38.38 0 0 0-.66.38L6.4 9.48A10.8 10.8 0 0 0 1 18h22a10.8 10.8 0 0 0-5.4-8.52zM7 15.25a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5zm10 0a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5z"/></svg>
          <h2>Android</h2>
          <p>Samsung, Google Pixel, Xiaomi and more over ADB.</p>
          <ul><li>FRP / account lock</li><li>Root & bootloader state</li><li>Battery & hardware diagnostics</li></ul>
        </div>
      </section>
      <a class="new-banner fade-in" href="#/checklist">
        <span class="nb-icon">📦</span>
        <span><b>Buying a brand-new iPhone?</b><br><span class="muted">Open the new-iPhone checklist: box seal, serial & IMEI match, warranty status, first start, battery and more.</span></span>
        <span class="btn small">Open checklist ›</span>
      </a>
      <div class="status-strip" id="envStrip"><span class="pill"><span class="spinner"></span> Checking Apple device service…</span></div>
      <div class="status-strip" id="demoStrip"></div>`;
    $app.querySelectorAll("[data-go]").forEach((el) => {
      const h = () => go(el.dataset.go);
      el.addEventListener("click", h);
      el.addEventListener("keydown", (e) => { if (e.key === "Enter") h(); });
    });
    await loadEnv();
    renderEnvStrip();
    if (!state.demoProfiles.length) state.demoProfiles = await api("/api/demo/profiles").catch(() => []);
    document.getElementById("demoStrip").innerHTML = `<span class="muted" style="align-self:center">No phone at hand? Try a demo:</span>` +
      state.demoProfiles.map((p) => `<button class="btn small" data-demo="${esc(p.id)}" title="${esc(p.desc)}">${esc(p.title)}</button>`).join("");
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
      <span class="pill"><span class="dot ${e.serviceOK ? "ok" : "bad"}"></span>Apple device service: ${e.serviceOK ? "running" : "not found"}</span>
      <a class="pill" href="#/specs"><span class="dot ok"></span>${e.specCount || 0} device models in database</a>
      <span class="pill">v${esc(e.version || "")}</span>`;
  }
  function bindDemo() {
    $app.querySelectorAll("[data-demo]").forEach((b) => b.addEventListener("click", () => {
      const p = state.demoProfiles.find((x) => x.id === b.dataset.demo);
      if (p && p.mode) setMode(p.mode);
      startScan({ demo: b.dataset.demo });
    }));
  }

  function viewAndroid() {
    crumbs([{ label: "Home", href: "#/" }, { label: "Android" }]);
    $app.innerHTML = `<div class="card fade-in" style="max-width:640px;margin:40px auto;text-align:center">
      <h2 style="font-size:22px">Android support is on the way</h2>
      <p class="muted">The Android route (ADB-based checks for FRP lock, root, bootloader, battery and hardware) will be added in a future version. The iPhone route is fully available now.</p>
      <div class="row" style="justify-content:center;margin-top:14px"><a class="btn primary" href="#/iphone">Inspect an iPhone</a><a class="btn" href="#/">Back</a></div></div>`;
  }

  // ------------------------------------------------------------ connect
  async function viewConnect() {
    crumbs([{ label: "Home", href: "#/" }, { label: "iPhone" }, { label: "Connect" }]);
    $app.innerHTML = `<div class="connect fade-in">
        <div class="card phone-art">${phoneSVG(150)}</div>
        <div id="connectPane"><div class="card"><div class="row"><span class="spinner"></span> Looking for the Apple device service…</div></div></div>
      </div>`;
    await loadEnv();
    if (!state.demoProfiles.length) state.demoProfiles = await api("/api/demo/profiles").catch(() => []);
    if (!state.env.serviceOK) return renderNoService();
    every(pollDevices, 2000);
  }

  function renderNoService() {
    const pane = document.getElementById("connectPane");
    pane.innerHTML = `<div class="notice">
        <h3>Apple device service not found</h3>
        <p class="muted" style="margin:6px 0 0">Windows needs Apple's USB driver to talk to an iPhone. Install <b>one</b> of these (free, from Apple), then click <i>Check again</i>:</p>
      </div>
      <div class="card" style="margin-top:14px">
        <ol class="steps">
          <li><div><b>Apple Devices</b> app from the Microsoft Store (recommended on Windows 10/11), or <b>iTunes</b>.</div></li>
          <li><div>Open it once so the driver finishes installing, then close it.</div></li>
          <li><div>Reconnect the iPhone and come back here.</div></li>
        </ol>
        <div class="row" style="margin-top:18px">
          <button class="btn primary" data-open="apple-devices">Get Apple Devices</button>
          <button class="btn" data-open="itunes">Get iTunes</button>
          <button class="btn" id="recheck">Check again</button>
        </div>
        <p class="dim" style="margin-top:14px;font-size:12px">Details: ${esc(state.env.serviceError || "")}</p>
      </div>
      <div class="row" style="margin-top:14px"><span class="muted">Just exploring?</span>${state.demoProfiles.map((p) => `<button class="btn small" data-demo="${esc(p.id)}">${esc(p.title)}</button>`).join("")}</div>`;
    document.getElementById("recheck").onclick = () => viewConnect();
    bindOpen(pane);
    bindDemo();
  }

  function bindOpen(root) {
    root.querySelectorAll("[data-open]").forEach((b) => b.addEventListener("click", async () => {
      try {
        const r = await api("/api/open", { target: b.dataset.open });
        if (!r.ok && b.dataset.open === "apple-devices") await api("/api/open", { target: "apple-devices-web" });
      } catch (e) { toast("Could not open: " + e.message); }
    }));
  }

  async function pollDevices() {
    try {
      const r = await api("/api/ios/devices");
      state.devError = r.ok ? "" : r.error;
      state.devices = (r.devices || []).filter((d) => !d.deviceClass || d.deviceClass === "iPhone" || d.deviceClass === "");
      const others = (r.devices || []).filter((d) => d.deviceClass && d.deviceClass !== "iPhone");
      renderDevices(others);
    } catch (e) {
      state.devError = String(e);
      renderDevices([]);
    }
  }

  function renderDevices(others) {
    const pane = document.getElementById("connectPane");
    if (!pane) return;
    if (!state.devices.length) {
      pane.innerHTML = `${modeBar()}<div class="card">
          <div class="row" style="justify-content:space-between"><h2 style="font-size:22px">Connect the iPhone</h2><span class="pill"><span class="dot wait"></span>Waiting for device</span></div>
          <ol class="steps">
            <li><div>Plug the iPhone into this computer with a <b>USB cable</b> (use a data cable, not a charge-only one).</div></li>
            <li><div><b>Unlock</b> the iPhone and keep it on the Home Screen.</div></li>
            <li><div>If asked <b>"Trust This Computer?"</b>, tap <b>Trust</b> and enter the passcode.</div></li>
          </ol>
          ${others.length ? `<p class="trust-box">${others.map((o) => esc(o.deviceClass)).join(", ")} detected — only iPhones are supported for now.</p>` : ""}
          ${state.devError ? `<p class="dim" style="font-size:12px;margin-top:12px">${esc(state.devError)}</p>` : ""}
        </div>`;
      bindModeBar(pane, () => renderDevices(others));
      return;
    }
    pane.innerHTML = `${modeBar()}<div class="card"><h2 style="font-size:22px">${state.devices.length > 1 ? "Choose a device" : "iPhone detected"}</h2>
      ${state.devices.map(deviceCard).join("")}</div>`;
    bindModeBar(pane, () => renderDevices(others));
    pane.querySelectorAll("[data-pair]").forEach((b) => b.addEventListener("click", () => pairLoop(b.dataset.pair)));
    pane.querySelectorAll("[data-scan]").forEach((b) => b.addEventListener("click", () => startScan({ udid: b.dataset.scan })));
  }

  function deviceCard(d) {
    const p = state.pairing[d.udid];
    const paired = d.paired || (p && p.state === "paired");
    const pairing = p && p.busy;
    let action, note = "";
    if (paired) {
      action = `<button class="btn primary big" data-scan="${esc(d.udid)}">Start full scan</button>`;
    } else if (pairing) {
      action = `<button class="btn" disabled><span class="spinner"></span> Waiting…</button>`;
      note = `<div class="trust-box">${esc(p.message || "Unlock the iPhone and tap Trust.")}</div>`;
    } else {
      action = `<button class="btn primary" data-pair="${esc(d.udid)}">Trust this computer</button>`;
      note = `<div class="trust-box">${esc((p && p.message) || d.pairMessage || "The iPhone must trust this computer before it can be scanned.")}</div>`;
    }
    return `<div class="device-card">
        ${phoneSVG(38, false)}
        <div class="meta">
          <h3>${esc(d.modelName || d.productType || "iPhone")}</h3>
          <div class="sub">${d.productVersion ? "iOS " + esc(d.productVersion) + " · " : ""}${esc(d.connectionType || "USB")} · <span class="mono">${esc(d.udid)}</span></div>
          <div class="sub" style="margin-top:4px"><span class="dot ${paired ? "ok" : "wait"}" style="display:inline-block;margin-right:6px"></span>${paired ? "Trusted — ready to scan" : "Not trusted yet"}</div>
          ${note}
        </div>
        ${action}
      </div>`;
  }

  async function pairLoop(udid) {
    const p = (state.pairing[udid] = { busy: true, message: "Check the iPhone: tap Trust and enter the passcode." });
    renderDevices([]);
    for (let i = 0; i < 45 && location.hash.startsWith("#/iphone"); i++) {
      try {
        const r = await api("/api/ios/pair", { udid });
        p.state = r.state;
        p.message = r.message;
        if (r.state === "paired") { p.busy = false; toast("iPhone trusts this computer"); break; }
        if (r.state === "denied" || r.state === "error") { p.busy = false; break; }
      } catch (e) { p.message = String(e); }
      renderDevices([]);
      await new Promise((res) => setTimeout(res, 2500));
    }
    p.busy = false;
    pollDevices();
  }

  // ------------------------------------------------------------ scan
  async function startScan(req) {
    try {
      if (!state.manualTests.length) state.manualTests = await api("/api/ios/manual-tests").catch(() => []);
      await api("/api/ios/scan", Object.assign({ mode: state.mode }, req));
      if (state.manualFor !== "pending") { state.manual = {}; state.notes = ""; }
      state.manualFor = req.udid || req.demo;
      state.tab = "summary";
      go("#/scan");
    } catch (e) { toast("Could not start scan: " + e.message); }
  }

  const STEP_LABELS = ["Reading device identity", "Reading battery gauge", "Running hardware diagnostics", "Measuring storage",
    "Auditing installed apps", "Checking management & profiles", "Probing for jailbreak services", "Analysing crash & panic logs", "Evaluating results"];

  function viewScan() {
    crumbs([{ label: "Home", href: "#/" }, { label: "iPhone", href: "#/iphone" }, { label: "Scanning" }]);
    $app.innerHTML = `<div class="scan fade-in">
        <div class="scan-ring" id="scanRing"></div>
        <div><h2 style="font-size:24px">Scanning the iPhone</h2>
          <p class="muted" style="margin:6px 0 16px">Keep the phone connected and unlocked. This usually takes under a minute.</p>
          <div class="step-list" id="stepList"></div>
          <div id="scanErr"></div></div>
      </div>`;
    every(pollScan, 450);
  }

  async function pollScan() {
    let s;
    try { s = await api("/api/ios/scan"); } catch (e) { return; }
    state.scan = s;
    const total = s.total || STEP_LABELS.length;
    const pct = s.status === "done" ? 100 : Math.round(((s.step || 0) - 0.5) / total * 100);
    const rEl = document.getElementById("scanRing");
    if (!rEl) return;
    rEl.innerHTML = ring(Math.max(pct, 3), 260, 16, "url(#scanGrad)",
      `<defs><linearGradient id="scanGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6366f1"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></defs>`) +
      `<div class="num"><div><b>${Math.max(0, pct)}%</b><span>${esc(s.label || "")}</span></div></div>`;
    document.getElementById("stepList").innerHTML = STEP_LABELS.map((l, i) => {
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
          const res = await api("/api/ios/manual", { results, notes: state.notes });
          if (res) state.result = res;
        } catch (e) { /* keep automatic result */ }
      }
      setTimeout(() => go("#/result"), 350);
    } else if (s.status === "error") {
      clearTimers();
      document.getElementById("scanErr").innerHTML = `<div class="notice" style="margin-top:16px"><h3>Scan failed</h3>
        <p class="muted">${esc(s.error)}</p><p class="muted">Make sure the iPhone is unlocked, trusted and still connected, then try again.</p>
        <div class="row" style="margin-top:10px"><a class="btn primary" href="#/iphone">Back to device</a></div></div>`;
    }
  }

  // ------------------------------------------------------------ result
  async function viewResult() {
    if (!state.result) {
      try { const s = await api("/api/ios/scan"); state.result = s.result; } catch (e) { /* ignore */ }
    }
    if (!state.result) return go("#/iphone");
    if (!state.manualTests.length) state.manualTests = await api("/api/ios/manual-tests").catch(() => []);
    const r = state.result;
    crumbs([{ label: "Home", href: "#/" }, { label: "iPhone", href: "#/iphone" }, { label: r.modelName }]);
    renderResult();
  }

  function idv(label) {
    const it = (state.result.identity || []).find((k) => k.label === label);
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
    const handsName = r.mode === "new" ? "New-iPhone checklist" : "Hands-on tests";
    const sub = [idv("iOS") && "iOS " + idv("iOS"), idv("Capacity"), idv("Model number"), idv("Part number")].filter(Boolean).join(" · ");
    $app.innerHTML = `
      <section class="card res-head fade-in">
        <div style="position:relative;width:150px;height:150px">${ring(r.score, 150, 13, color)}
          <div style="position:absolute;inset:0;display:grid;place-items:center;text-align:center"><div><b style="font-size:40px;line-height:1">${r.score}</b><div class="muted" style="font-size:12px">score</div></div></div></div>
        <div>
          <h1>${esc(r.modelName)}${r.snapshot && r.snapshot.demo ? '<span class="demo-flag">DEMO</span>' : ""}<span class="mode-flag ${r.mode === "new" ? "new" : ""}">${r.mode === "new" ? "NEW-DEVICE MODE" : "USED-DEVICE MODE"}</span></h1>
          <div class="sub">${esc(sub)}</div>
          <div class="verdict ${esc(r.verdict)}">${r.verdict === "good" ? "✓" : r.verdict === "caution" ? "!" : "✕"} ${esc(r.verdictText)}</div>
          <div class="counts">
            <span class="count"><i class="st pass" style="width:10px;height:10px"></i><b>${c.pass || 0}</b> passed</span>
            <span class="count"><i class="st warn" style="width:10px;height:10px"></i><b>${c.warn || 0}</b> warnings</span>
            <span class="count"><i class="st fail" style="width:10px;height:10px"></i><b>${c.fail || 0}</b> failed</span>
            <span class="count"><i class="st info" style="width:10px;height:10px"></i><b>${c.info || 0}</b> info</span>
            <span class="count">✋ <b>${handsDone}/${handTests.length}</b> ${r.mode === "new" ? "checklist" : "hands-on"}</span>
          </div>
        </div>
        <div class="head-actions">
          <button class="btn primary" id="saveHtml">Save report</button>
          <div class="row"><button class="btn small" id="saveJson">Export JSON</button><button class="btn small" data-open="reports">Reports folder</button></div>
          <a class="btn small" href="#/iphone">New scan</a>
        </div>
      </section>
      <nav class="tabs" id="tabs">
        ${[["summary", "Summary"], ["checks", "All checks", autoN, fails], ["battery", "Battery"], ["hands", handsName, `${handsDone}/${handTests.length}`], ["device", "Device & specs"], ["apps", "Apps", (r.snapshot.apps || []).length], ["raw", "Raw data"]]
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
      const r = await api("/api/report/save", { format, open: format === "html" });
      if (r.ok) toast("Saved: " + r.path); else toast("Could not save: " + r.error);
    } catch (e) { toast("Could not save: " + e.message); }
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
    }
    body.querySelectorAll("[data-copy]").forEach((b) => b.addEventListener("click", () => {
      navigator.clipboard.writeText(b.dataset.copy).then(() => toast("Copied"), () => toast("Copy failed"));
    }));
  }
  const slug = (s) => s.toLowerCase().replace(/[^a-z0-9]+/g, "-");

  function checkRow(c, open) {
    const hasBody = c.expected || c.actual || c.advice;
    return `<details class="check" ${open ? "open" : ""}>
      <summary><span class="st ${esc(c.status)}">${ICON[c.status] || "i"}</span>
        <div class="grow"><div class="t">${esc(c.title)}</div><div class="s">${esc(c.summary)}</div></div>
        ${c.status === "fail" || c.status === "warn" ? `<span class="sev">${esc(c.severity)}</span>` : ""}
      </summary>
      ${hasBody ? `<div class="body">
        ${c.expected || c.actual ? `<div class="ea">${c.expected ? `<span>Expected</span><span>${esc(c.expected)}</span>` : ""}${c.actual ? `<span>Found</span><span class="mono">${esc(c.actual)}</span>` : ""}</div>` : ""}
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
    return `
      <div class="grid2">
        <div class="card">
          <h3>${issues.length ? "Things to look at" : "No problems found"}</h3>
          <p class="muted" style="margin:0 0 6px">${issues.length ? "Sorted by importance. Click a category below for every check." : "All automated checks passed. Complete the hands-on tests before you pay."}</p>
          ${issues.slice(0, 8).map((c) => `<div class="issue"><span class="st ${c.status}">${ICON[c.status]}</span><div><div class="t">${esc(c.title)}</div><div class="s">${esc(c.summary)}</div>${c.advice ? `<div class="s" style="color:#bfdbfe;margin-top:4px">→ ${esc(c.advice)}</div>` : ""}</div></div>`).join("")}
          ${issues.length > 8 ? `<p class="muted">+ ${issues.length - 8} more in “All checks”.</p>` : ""}
        </div>
        <div style="display:grid;gap:18px;align-content:start">
          <div class="grid2">
            <div class="stat"><span>Battery health</span><b>${b.available && b.healthPercent ? Math.round(b.healthPercent) + "%" : "—"}</b><small>${b.available ? b.cycleCount + " cycles" : "not readable"}</small>
              <div class="meter"><div style="width:${b.healthPercent || 0}%;background:${(b.healthPercent || 0) >= 85 ? "var(--pass)" : (b.healthPercent || 0) >= 80 ? "var(--warn)" : "var(--fail)"}"></div></div></div>
            <div class="stat"><span>Storage</span><b>${esc(st.totalLabel || "—")}</b><small>${st.usedLabel ? esc(st.usedLabel) + " used" : ""}</small>
              <div class="meter"><div style="width:${usedPct}%;background:var(--grad)"></div></div></div>
          </div>
          ${handsLeft > 0 ? `<div class="card" style="border-color:rgba(129,140,248,.4)"><h3>${r.mode === "new" ? "📦" : "✋"} ${handsLeft} ${r.mode === "new" ? "new-iPhone checklist items" : "hands-on tests"} left</h3><p class="muted" style="margin:0 0 12px">${r.mode === "new" ? "Box seal, box label vs phone, Apple warranty status, first start and more can only be verified by you." : "Some things (screen, cameras, Face ID, speakers, parts history) can only be verified on the phone itself."}</p><button class="btn small primary" onclick="document.querySelector('[data-tab=hands]').click()">${r.mode === "new" ? "Open checklist" : "Start hands-on tests"}</button></div>` : ""}
          <div class="card"><h3>Before you pay</h3><ul class="muted" style="margin:6px 0 0;padding-left:18px;display:grid;gap:4px;font-size:13px">
            <li>Check the serial number's warranty/coverage status with Apple.</li>
            <li>Check the IMEI is not blacklisted (lost/stolen) with your carrier.</li>
            <li>Make sure the seller has signed out of their Apple Account and erased the phone.</li></ul>
            <div class="row" style="margin-top:12px"><button class="btn small" data-open="coverage">Apple coverage check</button>${idv("Serial number") ? `<button class="btn small" data-copy="${esc(idv("Serial number"))}">Copy serial</button>` : ""}${idv("IMEI") ? `<button class="btn small" data-copy="${esc(idv("IMEI"))}">Copy IMEI</button>` : ""}</div></div>
        </div>
      </div>
      <div class="section-title">Categories</div>
      <div class="grid3">${cats.map((cat) => {
        const list = checks.filter((c) => c.category === cat);
        const worst = list.reduce((w, c) => (ORDER[c.status] < ORDER[w] ? c.status : w), "pass");
        const label = cat === "Hands-on tests" && r.mode === "new" ? "New-iPhone checklist" : cat;
        const sub = cat === "Hands-on tests" && !list.length ? "Not started" : `${list.filter((c) => c.status === "pass").length}/${list.length} passed`;
        return `<div class="card cat-card" data-cat="${esc(cat)}"><div class="cat-icon">${CAT_ICON[cat] || "•"}</div>
          <div style="flex:1;min-width:0"><div class="row" style="justify-content:space-between"><b>${esc(label)}</b>${list.length ? `<span class="st ${worst}" style="width:20px;height:20px;font-size:10px">${ICON[worst]}</span>` : ""}</div>
          <div class="muted" style="font-size:12.5px">${sub}</div><div class="bar">${list.map((c) => `<i class="${c.status}"></i>`).join("")}</div></div></div>`;
      }).join("")}</div>`;
  }

  function tabChecks() {
    const checks = state.result.checks || [];
    const cats = [...new Set(checks.map((c) => c.category))];
    return cats.map((cat) => {
      const list = checks.filter((c) => c.category === cat).sort((a, b) => ORDER[a.status] - ORDER[b.status]);
      const label = cat === "Hands-on tests" && state.result.mode === "new" ? "New-iPhone checklist" : cat;
      return `<div class="cat-heading" id="cat-${slug(cat)}">${CAT_ICON[cat] || ""} ${esc(label)}</div><div class="check-list">${list.map((c) => checkRow(c, c.status === "fail")).join("")}</div>`;
    }).join("") + (Object.keys(state.result.snapshot.errors || {}).length ? `<div class="cat-heading">Could not be read</div><div class="card dim mono">${Object.entries(state.result.snapshot.errors).map(([k, v]) => `<div>${esc(k)}: ${esc(v)}</div>`).join("")}</div>` : "");
  }

  function tabBattery() {
    const b = state.result.battery || {};
    if (!b.available) return `<div class="card"><h3>Battery data not available</h3><p class="muted">The battery gauge could not be read on this device/iOS version. Check <b>Settings › Battery › Battery Health</b> on the phone and record it in the hands-on tests.</p></div>`;
    const h = b.healthPercent || 0;
    const col = h >= 85 ? "#22c55e" : h >= 80 ? "#f59e0b" : "#ef4444";
    const cyc = b.ratedCycles ? Math.min(100, b.cycleCount / b.ratedCycles * 100) : 0;
    const bc = (state.result.checks || []).filter((c) => c.category === "Battery");
    return `<div class="card gauge-wrap">
        <div style="position:relative;width:220px;height:220px;margin:auto">${ring(h, 220, 18, col)}
          <div style="position:absolute;inset:0;display:grid;place-items:center;text-align:center"><div><b style="font-size:44px;line-height:1">${h ? Math.round(h) + "%" : "—"}</b><div class="muted">maximum capacity</div></div></div></div>
        <div>
          <div class="grid2">
            <div class="stat"><span>Charge cycles</span><b>${b.cycleCount}</b><small>rated ${b.ratedCycles} cycles to 80%</small><div class="meter"><div style="width:${cyc}%;background:${cyc < 60 ? "var(--pass)" : cyc < 100 ? "var(--warn)" : "var(--fail)"}"></div></div></div>
            <div class="stat"><span>Full charge capacity</span><b>${b.fullChargeCapacity || "—"} <small>mAh</small></b><small>design ${b.designCapacity || "—"} mAh${b.expectedDesign ? " · factory spec " + b.expectedDesign.join("/") + " mAh" : ""}</small></div>
            <div class="stat"><span>Temperature</span><b>${b.temperatureC ? b.temperatureC.toFixed(1) + " °C" : "—"}</b><small>${b.voltageV ? b.voltageV.toFixed(2) + " V" : ""}</small></div>
            <div class="stat"><span>Current charge</span><b>${b.chargePercent ? b.chargePercent + "%" : "—"}</b><small>${b.isCharging ? "charging" : b.externalConnected ? "plugged in" : "on battery"}</small></div>
          </div>
          ${b.serial ? `<p class="muted" style="margin-top:12px;font-size:12.5px">Battery gauge serial: <span class="mono">${esc(b.serial)}</span></p>` : ""}
          <p class="dim" style="font-size:12px">Health method: ${esc(b.healthSource || "—")}. Apple rates iPhone 15 and later at 1000 cycles, earlier models at 500 cycles, to retain 80% capacity.</p>
        </div></div>
      <div class="section-title">Battery checks</div><div class="check-list">${bc.map((c) => checkRow(c, c.status !== "pass")).join("")}</div>`;
  }

  // hands-on
  function checklistHTML(tests, opts) {
    const done = doneCount(tests);
    let lastGroup = "";
    return `<div class="card" style="margin-bottom:16px"><div class="row" style="justify-content:space-between"><div><h3>${esc(opts.title)}</h3><p class="muted" style="margin:0">${esc(opts.intro)}</p></div>
        <div class="row"><b style="font-size:18px">${done}/${tests.length}</b><button class="btn small" id="resetChecklist">Reset</button></div></div>
      <div class="progress-line" style="margin-top:12px"><div style="width:${done / Math.max(1, tests.length) * 100}%"></div></div></div>
      <div class="check-list">${tests.map((t) => {
        const m = state.manual[t.id] || {};
        const g = t.group !== lastGroup ? `<div class="group-label" style="margin-top:${lastGroup ? 14 : 0}px">${esc(t.group)}</div>` : "";
        lastGroup = t.group;
        return `${g}<div class="card mt ${m.status ? "marked-" + m.status : ""}" data-mt="${esc(t.id)}"><div><b>${esc(t.title)}</b> <span class="dim" style="font-size:11px;text-transform:uppercase;letter-spacing:.08em;margin-left:6px">${esc(t.severity)}</span>
          <ol>${t.steps.map((st) => `<li>${esc(st)}</li>`).join("")}</ol>
          ${t.link ? `<button class="btn small" data-open="${esc(t.link)}" style="margin-top:8px">Open Apple coverage check ↗</button>` : ""}
          <input type="text" placeholder="Note (optional)" value="${esc(m.note || "")}" data-note="${esc(t.id)}"></div>
          <div class="seg">${["pass", "fail", "skip"].map((st) => `<button class="${st} ${m.status === st ? "on" : ""}" data-set="${st}">${st === "pass" ? "✓ Pass" : st === "fail" ? "✕ Fail" : "Skip"}</button>`).join("")}</div></div>`;
      }).join("")}</div>
      <div class="section-title">Your notes for the report</div>
      <textarea id="notes" placeholder="e.g. small scratch on the back glass, original box included, agreed price…">${esc(state.notes)}</textarea>`;
  }

  function tabHands() {
    const r = state.result;
    const isNew = r.mode === "new";
    return checklistHTML(testsFor(r.mode || "used"), isNew
      ? { title: "New-iPhone checklist", intro: "Everything to confirm before paying for a phone sold as brand new. Results are added to the score and the report." }
      : { title: "Guided hands-on tests", intro: "Do each test on the phone and mark the result. Results are added to the score and the report." });
  }

  let manualTimer;
  function pushManual(rerender) {
    clearTimeout(manualTimer);
    manualTimer = setTimeout(async () => {
      try {
        const results = Object.entries(state.manual).filter(([, m]) => m.status).map(([id, m]) => ({ id, status: m.status, note: m.note || "" }));
        const res = state.result ? await api("/api/ios/manual", { results, notes: state.notes }) : null;
        if (res) state.result = res;
        if (rerender) (state.rerender || renderResult)();
      } catch (e) { toast("Could not save test result: " + e.message); }
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
    if (rs) rs.addEventListener("click", () => { if (confirm("Clear all checklist results?")) { state.manual = {}; state.notes = ""; pushManual(true); } });
    bindOpen(document.getElementById("app"));
  }

  // device & specs
  function tabDevice() {
    const r = state.result;
    const kv = (r.identity || []).map((k) => `<div class="kv"><span>${esc(k.label)}</span><span class="${k.mono ? "mono" : ""}">${esc(k.value)}${k.mono ? `<button class="copy" title="Copy" data-copy="${esc(k.value)}">⧉</button>` : ""}</span></div>`).join("");
    const s = r.spec;
    let specHtml = `<div class="card"><h3>Not in the spec database</h3><p class="muted">Add a JSON file for <span class="mono">${esc(idv("Identifier"))}</span> to a <span class="mono">specs</span> folder next to the app to enable spec cross-checks.</p></div>`;
    if (s) {
      const ld = r.snapshot.lockdown || {};
      const rows = [
        ["Identifier", (s.productTypes || []).join(", "), ld.ProductType],
        ["Board", (s.hardwareModels || []).join(", "), ld.HardwareModel],
        ["Model numbers", (s.regulatoryModels || []).join(", "), ld.RegulatoryModelNumber],
        ["Chip", s.chip, ld.ChipID !== undefined ? "0x" + Number(ld.ChipID).toString(16).toUpperCase() : ""],
        ["Storage options", (s.storageOptionsGB || []).map((g) => (g >= 1024 ? g / 1024 + " TB" : g + " GB")).join(" / "), idv("Capacity")],
        ["Battery (design)", (s.battery?.designCapacityMah || []).map((x) => x + " mAh").join(" / "), r.battery?.designCapacity ? r.battery.designCapacity + " mAh" : ""],
        ["iOS support", (s.launchIOS ? "iOS " + s.launchIOS : "") + (s.maxIOS ? " – " + s.maxIOS + ".x" : " – current"), idv("iOS")],
        ["RAM", s.ramGB ? s.ramGB + " GB" : "", ""],
        ["Display", [s.display?.sizeInches && s.display.sizeInches + '"', s.display?.technology, s.display?.resolution, s.display?.maxRefreshHz && s.display.maxRefreshHz + " Hz"].filter(Boolean).join(" · "), ""],
        ["Biometrics", s.biometrics, ""],
        ["Port", [s.port, s.usbSpec].filter(Boolean).join(" · "), ""],
        ["Cellular", [s.cellular?.generation, s.cellular?.sim].filter(Boolean).join(" · "), ""],
        ["Rear cameras", (s.cameras?.rear || []).join(" · "), ""],
        ["Front camera", s.cameras?.front, ""],
        ["Colours", (s.colors || []).join(", "), idv("Colour code") ? "code " + idv("Colour code") : ""],
        ["Released", s.releaseDate, ""],
        ["Features", (s.features || []).join(", "), ""],
      ].filter((x) => x[1]);
      specHtml = `<div class="card"><div class="row" style="justify-content:space-between"><h3>${esc(s.name)} — factory specification</h3><span class="badge">${esc(s.origin || "")}</span></div>
        <table class="cmp"><thead><tr><th>Item</th><th>Factory spec</th><th>Read from device</th></tr></thead><tbody>
        ${rows.map(([a, b, c]) => `<tr><td>${esc(a)}</td><td>${esc(b)}</td><td class="mono">${esc(c || "")}</td></tr>`).join("")}</tbody></table></div>`;
    }
    return `<div class="card"><h3>Device identity</h3><div class="kv-grid">${kv}</div></div><div class="section-title">Specification</div>${specHtml}`;
  }

  function tabApps() {
    const apps = state.result.snapshot.apps || [];
    if (!state.result.snapshot.appsScanned) return `<div class="card muted">Installed apps could not be listed.</div>`;
    return `<div class="card"><div class="row" style="justify-content:space-between;margin-bottom:10px"><h3>${apps.length} installed apps</h3><input type="search" id="appSearch" placeholder="Search apps…" value="${esc(state.appFilter)}" style="width:260px"></div>
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
      }).join("") || '<p class="muted">No apps match.</p>';
    };
    const s = document.getElementById("appSearch");
    s.addEventListener("input", () => { state.appFilter = s.value; draw(); });
    draw();
  }

  function tabRaw() {
    return `<div class="card"><div class="row" style="justify-content:space-between;margin-bottom:10px"><h3>Raw data read from the device</h3><button class="btn small" data-copy="${esc(JSON.stringify(state.result.snapshot, null, 2))}">Copy JSON</button></div>
      <pre class="raw">${esc(JSON.stringify(state.result.snapshot, null, 2))}</pre></div>`;
  }

  // ------------------------------------------------------------ new-iPhone checklist (no scan needed)
  async function viewChecklist() {
    crumbs([{ label: "Home", href: "#/" }, { label: "New iPhone checklist" }]);
    if (!state.manualTests.length) state.manualTests = await api("/api/ios/manual-tests").catch(() => []);
    if (state.result && state.result.mode !== "new") { state.result = null; state.manual = {}; state.notes = ""; }
    if (!state.result) state.manualFor = "pending";
    const draw = () => {
      const tests = testsFor("new");
      $app.innerHTML = `<div class="fade-in">
        <div class="card new-hero"><div><span class="badge live">📦 New-device mode</span>
          <h2 style="font-size:26px;margin-top:10px">Buying a brand-new iPhone</h2>
          <p class="muted" style="margin:6px 0 0;max-width:720px">Go through this list at the shop or with the seller. Start with the box before it is opened. Then connect the phone with a USB cable: the app adds automatic checks that prove it was never used (battery cycles, retail part number, no previous apps or data, clean crash history), and everything you tick here goes into the same report.</p></div>
          <button class="btn primary big" id="goScan">Connect iPhone & run automatic checks ›</button></div>
        <div style="margin-top:18px">${checklistHTML(tests, { title: "New-iPhone checklist", intro: "Mark each item Pass or Fail as you check it." })}</div></div>`;
      document.getElementById("goScan").onclick = () => { setMode("new"); state.manualFor = state.result ? state.manualFor : "pending"; go("#/iphone"); };
      bindHands();
    };
    state.rerender = draw;
    draw();
  }

  // ------------------------------------------------------------ specs browser
  async function viewSpecs(id) {
    crumbs([{ label: "Home", href: "#/" }, { label: "Spec database", href: "#/specs" }].concat(id ? [{ label: id }] : []));
    if (!state.specs) {
      $app.innerHTML = `<div class="card"><div class="row"><span class="spinner"></span>Loading…</div></div>`;
      state.specs = await api("/api/specs");
    }
    const devs = state.specs.devices || [];
    if (id) {
      const d = devs.find((x) => x.id === id);
      if (!d) return go("#/specs");
      $app.innerHTML = `<div class="card fade-in"><div class="row" style="justify-content:space-between"><h2 style="font-size:24px">${esc(d.name)}</h2><span class="badge">${esc(d.origin)}</span></div>
        <p class="muted">${esc((d.productTypes || []).join(", "))} · ${esc(d.chip || "")} · released ${esc(d.releaseDate || "?")}</p>
        <pre class="raw">${esc(JSON.stringify(d, null, 2))}</pre><a class="btn" href="#/specs">‹ Back</a></div>`;
      return;
    }
    const draw = () => {
      const q = state.specFilter.toLowerCase();
      document.getElementById("specGrid").innerHTML = devs.filter((d) => !q || JSON.stringify([d.name, d.productTypes, d.regulatoryModels, d.chip]).toLowerCase().includes(q))
        .map((d) => `<a class="spec-item" href="#/specs/${encodeURIComponent(d.id)}"><b>${esc(d.name)}</b><span>${esc((d.productTypes || []).join(", "))} · ${esc(d.chip || "")}</span><br><span>${esc((d.releaseDate || "").slice(0, 4))} · ${esc((d.storageOptionsGB || []).map((g) => (g >= 1024 ? g / 1024 + "TB" : g + "GB")).join("/"))}</span></a>`).join("");
    };
    $app.innerHTML = `<div class="fade-in"><div class="row" style="justify-content:space-between;margin-bottom:16px"><div><h2 style="font-size:26px">Device spec database</h2>
      <p class="muted" style="margin:4px 0 0">${devs.length} models. Built into the app; add or override models by placing JSON files in a <span class="mono">specs</span> folder next to the .exe${state.specs.external ? ` (loaded: <span class="mono">${esc(state.specs.external)}</span>)` : ""}.</p></div>
      <input type="search" id="specSearch" placeholder="Search name, identifier, A-number, chip…" style="width:320px" value="${esc(state.specFilter)}"></div>
      ${(state.specs.warnings || []).length ? `<div class="trust-box" style="margin-bottom:14px">${state.specs.warnings.map(esc).join("<br>")}</div>` : ""}
      <div class="spec-list" id="specGrid"></div></div>`;
    const s = document.getElementById("specSearch");
    s.addEventListener("input", () => { state.specFilter = s.value; draw(); });
    draw();
  }

  // ------------------------------------------------------------ global
  document.querySelector('[data-action="quit"]').addEventListener("click", async () => {
    if (!confirm("Quit Smart Mobile Check?")) return;
    try { await api("/api/quit", {}); } catch (e) { /* ignore */ }
    document.body.innerHTML = '<div style="display:grid;place-items:center;height:100vh;color:#93a0b8;font:16px Segoe UI,sans-serif">Smart Mobile Check has been closed. You can close this window.</div>';
  });

  route();
})();
