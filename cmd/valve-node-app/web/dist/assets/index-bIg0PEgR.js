var He=Object.defineProperty;var Ae=(n,r,a)=>r in n?He(n,r,{enumerable:!0,configurable:!0,writable:!0,value:a}):n[r]=a;var ye=(n,r,a)=>Ae(n,typeof r!="symbol"?r+"":r,a);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))e(d);new MutationObserver(d=>{for(const u of d)if(u.type==="childList")for(const g of u.addedNodes)g.tagName==="LINK"&&g.rel==="modulepreload"&&e(g)}).observe(document,{childList:!0,subtree:!0});function a(d){const u={};return d.integrity&&(u.integrity=d.integrity),d.referrerPolicy&&(u.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?u.credentials="include":d.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(d){if(d.ep)return;d.ep=!0;const u=a(d);fetch(d.href,u)}})();function De(){return I("/api/host")}function V(){return I("/api/catalog")}function X(){return I("/api/targets")}function $e(n){return I("/api/targets",{method:"POST",headers:ne,body:JSON.stringify(n)})}function Ue(n){return I(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function Ne(n,r){return I(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(r)}`)}function Me(n,r){return I(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:ne,body:JSON.stringify(r)})}function qe(n,r){const a=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return a.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>a.close()}function Fe(n,r){const a=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return a.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>a.close()}function Oe(n,r=200){return I(`/api/targets/${encodeURIComponent(n)}/logs?n=${r}`)}function _e(n,r){const a=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return a.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>a.close()}function we(n,r){const a=r===void 0?{}:{lines:r};return I(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:ne,body:JSON.stringify(a)})}function ze(n,r,a){return I(`/api/targets/${encodeURIComponent(n)}/services/${r}/${a}`,{method:"POST"})}function je(n,r){return I(`/api/targets/${encodeURIComponent(n)}/services/${r}/clear`,{method:"POST",headers:ne,body:JSON.stringify({Confirm:r})})}function Ke(n){return I(`/api/targets/${encodeURIComponent(n)}/du`)}function We(n){return I(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function Je(n){return I(`/api/targets/${encodeURIComponent(n)}/firewall`)}function Ge(n){return I(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function Ye(n){return I(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function Ve(){return I("/api/settings")}function Xe(n){return I("/api/settings",{method:"PUT",headers:ne,body:JSON.stringify(n)})}class fe extends Error{constructor(a,e){super(e);ye(this,"status");this.name="ApiError",this.status=a}}const ne={"Content-Type":"application/json"};async function I(n,r){const a=await fetch(n,r);if(!a.ok){let d=a.statusText||`HTTP ${a.status}`;try{const u=await a.json();u&&typeof u.error=="string"&&u.error&&(d=u.error)}catch{}throw new fe(a.status,d)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const ke="https://learn.valve.city/rpc";function s(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function D(n,r){const a=n&&r&&r!==ke?` <span class="footer-sep">·</span> <a href="${s(r)}" target="_blank" rel="noopener noreferrer">${s(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${s(ke)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function Ze(n){n.innerHTML=`
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/targets">valve-node-app</a>
        <nav class="nav">
          <a href="#/targets" data-nav="targets">Targets</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;const r=n.querySelector("#content"),a=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:d=>{for(const u of a)u.classList.toggle("active",u.dataset.nav===d)}}}function j(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function Qe(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function et(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const r=Math.round(n/60),a=Math.floor(r/60),e=r%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const d=Math.floor(a/24),u=a%24;return`~${d}d ${u}h`}function S(n,r){return`<span class="badge badge-${r}">${s(n)}</span>`}function xe(n){return`<span class="dot dot-${n}"></span>`}const Te=["B","KB","MB","GB","TB","PB"];function W(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let r=n,a=0;for(;r>=1024&&a<Te.length-1;)r/=1024,a++;const e=r<10?2:r<100?1:0;return`${r.toFixed(e)} ${Te[a]}`}async function me(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function Z(n,r){n.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!n.contains(e))return;const d=e.dataset.action;d&&r(d,e,a)})}function he(n,r,a){const e=r.find(u=>u.value===a),d=r.map(u=>`
      <li class="dropdown-option${u.value===a?" selected":""}" role="option"
          aria-selected="${u.value===a}" data-value="${s(u.value)}">
        ${s(u.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${s(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${s(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function ee(n){n.querySelectorAll(".dropdown.open").forEach(r=>{var a;r.classList.remove("open"),(a=r.querySelector(".dropdown-trigger"))==null||a.setAttribute("aria-expanded","false")})}function Ce(n,r){n.addEventListener("click",d=>{const u=d.target,g=u.closest(".dropdown-trigger");if(g&&n.contains(g)){const P=g.closest(".dropdown"),C=!!P&&!P.classList.contains("open");ee(n),P&&C&&(P.classList.add("open"),g.setAttribute("aria-expanded","true"));return}const m=u.closest(".dropdown-option");if(m&&n.contains(m)){const P=m.closest(".dropdown");ee(n),r((P==null?void 0:P.dataset.dropdown)??"",m.dataset.value??"");return}ee(n)});const a=d=>{if(!n.isConnected){document.removeEventListener("click",a),document.removeEventListener("keydown",e);return}const u=d.target;(!u.closest(".dropdown")||!n.contains(u))&&ee(n)},e=d=>{if(!n.isConnected){document.removeEventListener("click",a),document.removeEventListener("keydown",e);return}d.key==="Escape"&&ee(n)};document.addEventListener("click",a),document.addEventListener("keydown",e)}const ce="app-modal";let ie=null;function Le(n,r){te();const a=document.createElement("div");a.className="modal-overlay",a.id=ce,a.innerHTML=`<div class="modal">${n}</div>`,a.addEventListener("click",d=>{const u=d.target.closest("[data-modal-action]");u!=null&&u.dataset.modalAction?r(u.dataset.modalAction):d.target===a&&r("cancel")});const e=d=>{d.key==="Escape"&&r("cancel")};document.addEventListener("keydown",e),ie=e,document.body.appendChild(a)}function te(){var n;(n=document.getElementById(ce))==null||n.remove(),ie&&(document.removeEventListener("keydown",ie),ie=null)}function tt(){return document.querySelector(`#${ce} .modal`)}function nt(n){return new Promise(r=>{var d;let a=!1;const e=u=>{a||(a=!0,te(),r(u))};Le(`
        <h2>${s(n.title)}</h2>
        <p>${s(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm">${s(n.confirmLabel)}</button>
        </div>
      `,u=>e(u==="confirm")),(d=document.querySelector(`#${ce} [data-modal-action="confirm"]`))==null||d.focus()})}const at=85,ue={exec:"Execution",beacon:"Beacon"};function rt(n,r){let a=!1,e=null,d=null,u=null,g=null,m=null,P=null,C=null,E=null;const L={exec:null,beacon:null};let w=null;n.innerHTML=`<h1>Dashboard: ${s(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${D()}</div>`;const T=n.querySelector("#dash-body"),v=n.querySelector("#dash-footer");T.addEventListener("click",t=>{const o=t.target.closest("[data-action]");if(!o||!T.contains(o))return;const p=o.dataset.action;if(p==="svc-action"){const h=o.dataset.svc,$=o.dataset.kind;h&&$&&z(h,$)}else if(p==="open-clear"){const h=o.dataset.svc;h&&de(h)}else if(p==="copy"){const h=o.dataset.copy;h&&le(o,h)}else p==="retry-du"?i():p==="retry-endpoints"&&l()}),f();async function f(){let t,o;try{const[h,$]=await Promise.all([X(),V()]);t=h.find(x=>x.id===r),o=$}catch(h){if(a)return;T.innerHTML=`<p class="error">Failed to load target: ${s(String(h))}</p>`;return}if(a)return;if(!t){T.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!t.wire){T.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const p=o==null?void 0:o.networks.find(h=>h.ChainID===t.wire.ChainID);p&&(v.innerHTML=D(p.Name,p.LearnURL)),T.innerHTML='<p class="muted">Connecting…</p>',e=Fe(r,h=>{a||(b(h),d=h,u=h,y())}),i(),l()}async function i(){P=null;try{m=await Ke(r)}catch(t){m=null,P=String(t instanceof Error?t.message:t)}a||y()}async function l(){E=null;try{C=await We(r)}catch(t){C=null,E=String(t instanceof Error?t.message:t)}a||y()}function b(t){if(!d)return;const o=(new Date(t.at).getTime()-new Date(d.at).getTime())/1e3,p=t.execHead-d.execHead;if(o>0&&p>=0){const h=p/o;g=g===null?h:g*.7+h*.3}}function y(){if(!u)return;const t=u;T.innerHTML=`
      <p class="dash-status">${H(t)}</p>
      <div class="card-grid">
        ${q(t)}
        ${_(t)}
        ${J(t)}
        ${G(t)}
        ${Q(t)}
        ${Y()}
      </div>
      <p class="muted small">Last updated ${s(new Date(t.at).toLocaleTimeString())}</p>
    `}function H(t){return!t.execActive&&!t.beaconActive?S("Node not running","bad"):t.execSyncing||t.beaconDistance>0?S("Syncing","warn"):S("Running · synced","ok")}function R(t){const p=t.refHead>0?t.refHead-t.execHead:null,h=p!==null&&p>0&&g&&g>0?et(p/g):p!==null&&p<=0?"caught up":"—";return{lag:p,eta:h}}function _(t){const{lag:o,eta:p}=R(t);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${t.execActive?t.execSyncing?S("syncing","warn"):t.execHead===0?S("no data","neutral"):S("synced","ok"):S("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${j(t.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${o!==null?j(t.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${o!==null?j(Math.max(o,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${p}</dd></div>
        </dl>
      </div>
    `}function J(t){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${t.beaconActive?t.beaconSlot===0?S("no data","neutral"):t.beaconDistance===0?S("synced","ok"):S("syncing","warn"):S("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${j(t.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${j(t.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function G(t){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${j(t.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${j(t.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Q(t){const o=t.diskUsedPct>=at,p=`
      <div class="meter"><div class="meter-fill ${o?"meter-warn":""}" style="width:${Math.min(t.diskUsedPct,100)}%"></div></div>
      <p>${Qe(t.diskUsedPct)} used</p>
    `;if(P)return`
        <div class="card ${o?"card-warn":""}">
          <h3>Storage</h3>
          ${p}
          <p class="error small">${s(P)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${o?"card-warn":""}">
          <h3>Storage</h3>
          ${p}
          <p class="muted">Loading…</p>
        </div>
      `;const h=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,$=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:x,eta:B}=R(t),A=x!==null&&x>0&&g!==null&&g>0;return`
      <div class="card ${o?"card-warn":""}">
        <h3>Storage</h3>
        ${p}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${W(m.ExecBytes)} of ~${W(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${h}%"></div></div>
        ${A?`<p class="muted small">Estimated time remaining: ${s(B)}</p>`:""}
        <p class="muted small">Beacon — ${W(m.BeaconBytes)} of ~${W(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${$}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${W(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function Y(){if(E)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(E)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!C)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const t=C,o=t.ExecReachable&&!t.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",p=t.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${s(t.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${s(t.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${xe(t.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${s(t.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(t.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${xe(t.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${s(t.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(t.BeaconHTTP)}">Copy</button>
        </div>
        ${o}
        ${p}
      </div>
    `}function O(t,o){const p=ue[t],h=L[t],$=(x,B,A)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${t}" data-kind="${x}" ${h!==null||A?"disabled":""}>${h===x?U():s(B)}</button>`;return`
      <div class="service-row">
        <span>${s(p)} ${o?S("active","ok"):S("down","bad")}</span>
        <div class="service-actions">
          ${$("start","Start",o)}
          ${$("stop","Stop",!o)}
          ${$("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${t}" ${h!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function q(t){return`
      <div class="card">
        <h3>Services</h3>
        ${O("exec",t.execActive)}
        ${O("beacon",t.beaconActive)}
        ${w?`<p class="error small">${s(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function U(){return'<span class="spinner" aria-label="working"></span>'}async function z(t,o){if(L[t]===null){L[t]=o,w=null,y();try{await ze(r,t,o)}catch(p){w=`${ue[t]} ${o} failed: ${p instanceof Error?p.message:String(p)}`}L[t]=null,a||y()}}async function le(t,o){const p=await me(o),h=t.textContent;t.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{a||(t.textContent=h)},1500)}function de(t){const o=ue[t],p=m?W(t==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";Le(`
        <h2>Clear ${s(o)} data</h2>
        <p class="error">
          This stops the ${s(o.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(p)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(t)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,x=>{if(x==="cancel"){te();return}x==="confirm"&&c(t)});const h=document.getElementById("clear-confirm-input"),$=document.getElementById("clear-confirm-btn");h==null||h.addEventListener("input",()=>{$&&($.disabled=h.value.trim()!==t)}),h==null||h.focus()}async function c(t){const o=document.getElementById("clear-confirm-btn");o&&(o.disabled=!0,o.textContent="Clearing…");try{await je(r,t),te(),i()}catch(p){const h=tt();if(h){const $=document.createElement("p");$.className="error small",$.textContent=`Clear failed: ${p instanceof Error?p.message:String(p)}`,h.appendChild($)}o&&(o.disabled=!1,o.textContent="Clear and resync")}}return()=>{a=!0,e==null||e(),te()}}const Pe=500,Se="valve-node-app.explain-consent";function st(n,r){let a=!1,e=null;const d=[];n.innerHTML=`
    <h1>Logs: ${s(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${D()}</div>
  `;const u=n.querySelector("#logs-body"),g=n.querySelector("#logs-footer");Z(n,f=>{f==="explain"&&E()}),m();async function m(){let f,i;try{const[b,y]=await Promise.all([X(),V()]);f=b.find(H=>H.id===r),i=y}catch(b){if(a)return;u.innerHTML=`<p class="error">Failed to load target: ${s(String(b))}</p>`;return}if(a)return;if(!f){u.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const l=i==null?void 0:i.networks.find(b=>b.ChainID===f.wire.ChainID);l&&(g.innerHTML=D(l.Name,l.LearnURL));try{const b=await Oe(r,200);if(a)return;d.push(...b)}catch(b){if(a)return;u.innerHTML=`<p class="error">Failed to load logs: ${s(String(b))}</p>`;return}P(),e=_e(r,b=>{a||(d.push(b),d.length>Pe&&d.splice(0,d.length-Pe),P())})}function P(){const f=d.filter(l=>l.severity==="error"||l.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(C).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${S(String(f.length),f.length?"bad":"neutral")}</h2>
          <div class="log-lines">${f.length?f.slice().reverse().map(C).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const i=u.querySelector(".log-lines");i&&(i.scrollTop=i.scrollHeight)}function C(f){const i=f.severity||"info",l=f.learnUrl?` <a href="${s(f.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(i)}">
        <span class="log-time">${s(new Date(f.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(f.unit)}</span>
        <span class="log-sev">${s(i)}</span>
        <span class="log-text">${s(f.line)}</span>
        ${f.explain?`<div class="log-explain">${s(f.explain)}${l}</div>`:""}
      </div>
    `}async function E(){const f=d.filter(l=>l.severity==="error"||l.severity==="critical").map(l=>l.line).slice(-40);if(!(localStorage.getItem(Se)==="1")){L(f);return}await w(f)}function L(f){const i=f.length?`<pre class="explain-excerpt">${f.map(l=>s(l)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';T(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${i}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,l=>{l==="proceed"?(localStorage.setItem(Se,"1"),v(),w(f)):v()})}async function w(f){T('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const i=f.length?await we(r,f):await we(r);if(a)return;T(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(i.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${i.sentExcerpt.map(l=>s(l)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&v()})}catch(i){if(a)return;if(i instanceof fe&&i.status===409){T(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,l=>{l==="close"&&v()});return}T(`
        <h2>Explain failed</h2>
        <p class="error">${s(i instanceof Error?i.message:String(i))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&v()})}}function T(f,i){v();const l=document.createElement("div");l.className="modal-overlay",l.id="explain-modal",l.innerHTML=`<div class="modal">${f}</div>`,l.addEventListener("click",b=>{const y=b.target.closest("[data-modal-action]");y!=null&&y.dataset.modalAction&&i(y.dataset.modalAction),b.target===l&&i("cancel")}),document.body.appendChild(l)}function v(){var f;(f=document.getElementById("explain-modal"))==null||f.remove()}return()=>{a=!0,e==null||e(),v()}}function ot(n,r){let a=!1,e=null,d=null,u=!1,g=!1;n.innerHTML=`<h1>Network diagnostics: ${s(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${D()}</div>`;const m=n.querySelector("#diag-body"),P=n.querySelector("#diag-footer");Z(n,(i,l)=>{var b;if(i==="run")E();else if(i==="toggle")(b=l.closest(".check-item"))==null||b.classList.toggle("expanded");else if(i==="copy"){const y=l.dataset.copy;y&&f(l,y)}}),C();async function C(){let i,l;try{const[y,H]=await Promise.all([X(),V()]);i=y.find(R=>R.id===r),l=H}catch(y){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${s(String(y))}</p>`;return}if(a)return;if(!i){m.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const b=l==null?void 0:l.networks.find(y=>y.ChainID===i.wire.ChainID);b&&(P.innerHTML=D(b.Name,b.LearnURL));try{e=await Ye(r),g=!0}catch(y){d=String(y instanceof Error?y.message:y)}a||L()}async function E(){u=!0,d=null,L();try{e=await Ge(r),g=!0}catch(i){d=String(i instanceof Error?i.message:i)}u=!1,a||L()}function L(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${u?"disabled":""}>${u?"Running…":"Run diagnostics"}</button>
      </div>
      ${d?`<p class="error">${s(d)}</p>`:""}
      ${w()}
    `}function w(){if(!g&&!d)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const i=new Date(e.at).toLocaleString(),l=e.failedId?`<p><strong>Failed at: ${s(T(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(i)} — trigger: ${s(e.trigger)}</p>
      ${l}
      <ul class="check-list">${e.items.map(v).join("")}</ul>
    `}function T(i){var l;return((l=e==null?void 0:e.items.find(b=>b.ID===i))==null?void 0:l.Title)??i}function v(i){const l=i.Status==="pass"?"ok":i.Status==="fail"?"bad":i.Status==="warn"?"warn":"neutral",b=i.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${b?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${S(b?"failed here":i.Status,l)}
          <strong>${s(i.Title)}</strong>
          <span class="muted small check-detail-inline">${s(i.Detail)}</span>
        </button>
        <div class="check-body">
          <details${b?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${s(i.Why)}</p>
          </details>
          ${i.Fix?`
                <details${b?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(i.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(i.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function f(i,l){const b=await me(l),y=i.textContent;i.textContent=b?"Copied!":"Copy failed",setTimeout(()=>{a||(i.textContent=y)},1500)}return()=>{a=!0}}function it(n,r){let a=!1,e=[],d=null,u=!1,g=!1;n.innerHTML=`<h1>Security: ${s(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${D()}</div>`;const m=n.querySelector("#sec-body"),P=n.querySelector("#sec-footer");Z(n,(v,f)=>{var i;if(v==="rerun")E();else if(v==="toggle")(i=f.closest(".check-item"))==null||i.classList.toggle("expanded");else if(v==="copy"){const l=f.dataset.copy;l&&T(f,l)}}),C();async function C(){let v,f;try{const[l,b]=await Promise.all([X(),V()]);v=l.find(y=>y.id===r),f=b}catch(l){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${s(String(l))}</p>`;return}if(a)return;if(!v){m.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!v.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const i=f==null?void 0:f.networks.find(l=>l.ChainID===v.wire.ChainID);i&&(P.innerHTML=D(i.Name,i.LearnURL)),await E()}async function E(){u=!0,d=null,L();try{e=await Je(r),g=!0}catch(v){d=String(v instanceof Error?v.message:v)}u=!1,a||L()}function L(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${s(d)}</p>`:""}
      ${!g&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:g?'<p class="muted">No checks returned.</p>':""}
    `}function w(v){const f=v.Status==="pass"?"ok":v.Status==="fail"?"bad":v.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${S(v.Status,f)}
          <strong>${s(v.Title)}</strong>
          <span class="muted small check-detail-inline">${s(v.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${s(v.Why)}</p>
          </details>
          ${v.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(v.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(v.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(v,f){const i=await me(f),l=v.textContent;v.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{a||(v.textContent=l)},1500)}return()=>{a=!0}}const ct=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function lt(n){let r=!1,a=!1,e=!1,d=null,u=!1,g=null,m=null;n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${D()}`;const P=n.querySelector("#settings-body");Z(n,w=>{if(w==="save"&&L(),w==="clear-key"){if(!g)return;a=!0;const T=n.querySelector("#ai-key");T&&(T.value=""),E(g)}}),Ce(n,(w,T)=>{w!=="ai-provider"||!g||(m=T,u=!1,E(g))}),C();async function C(){try{const w=await Ve();if(r)return;g=w,E(w)}catch(w){if(r)return;P.innerHTML=`<p class="error">Failed to load settings: ${s(String(w))}</p>`}}function E(w){var f;const T=m??w.aiProvider;P.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${he("ai-provider",ct.map(i=>({value:i.value,label:i.label})),T)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${w.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${w.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${s(w.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${s(d)}</p>`:""}
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const v=n.querySelector("#ai-key");v==null||v.addEventListener("input",()=>{a=!0,u=!1}),(f=n.querySelector("#ref-rpc-base"))==null||f.addEventListener("input",()=>{u=!1})}async function L(){const w=n.querySelector("#ai-key"),T=n.querySelector("#ref-rpc-base");if(!w||!T||!g)return;const v={aiProvider:m??g.aiProvider,refRpcBase:T.value.trim()};a&&(v.aiKey=w.value),e=!0,d=null,u=!1,E(g);try{const f=await Xe(v);if(r)return;g=f,a=!1,e=!1,u=!0,E(f)}catch(f){if(r)return;e=!1,d=String(f instanceof Error?f.message:f),E(g)}}return()=>{r=!0}}const dt="local";function ut(n){let r=!1,a=!1,e="",d=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${D()}
  `;const u=n.querySelector("#targets-body");Z(n,(i,l)=>{E(i,l)}),g();async function g(){try{const[i,l,b]=await Promise.all([X(),V(),De()]);if(r)return;e=b.os,P(i,l)}catch(i){if(r)return;u.innerHTML=`<p class="error">Failed to load machines: ${s(String(i))}</p>`}}function m(){d&&P(d.targets,d.catalog)}function P(i,l){d={targets:i,catalog:l};const b=e==="linux",y=[...i].sort((R,_)=>(R.mode==="local"?-1:0)-(_.mode==="local"?-1:0)),H=y.length?`<div class="card-grid">${y.map(R=>pt(R,l)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>';u.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${H}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${C(b)}
        ${a?ht():""}
      </section>
    `}function C(i){const l=`
      <div class="card">
        <h3>A server over SSH ${S("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${i?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${i?" btn-ghost":""}" data-action="toggle-ssh">
            ${a?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,b=i?`
        <div class="card">
          <h3>This machine ${S("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${s(e)})`:""} ${S("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return`<div class="card-grid">${i?b+l:l+b}</div>`}async function E(i,l){var b;if(i==="add-local"){await L();return}if(i==="delete-target"){const y=l.dataset.id;if(!y||!await nt({title:"Remove machine",body:`Remove "${y}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove"}))return;await w(y);return}if(i==="toggle-ssh"){a=!a,f(),m(),a&&((b=n.querySelector("#ssh-host"))==null||b.focus());return}i==="add-ssh"&&await T()}async function L(){f();try{await $e({id:dt,mode:"local"}),await g()}catch(i){v(i)}}async function w(i){try{await Ue(i),await g()}catch(l){v(l)}}async function T(){const i=n.querySelector("#ssh-host"),l=n.querySelector("#ssh-user"),b=n.querySelector("#ssh-key"),y=n.querySelector("#ssh-port"),H=n.querySelector("#ssh-id");if(!i||!l||!b||!y||!H)return;const R=i.value.trim(),_=l.value.trim(),J=b.value.trim(),G=y.value.trim(),Q=H.value.trim();if(f(),!R||!_||!J){v(new Error("host, user, and key path are required"));return}const Y=Q||ft(R),O={Host:R,User:_,KeyPath:J};if(G){const U=Number.parseInt(G,10);if(!Number.isFinite(U)||U<=0){v(new Error("port must be a positive number"));return}O.Port=U}const q=n.querySelector("#ssh-submit");q&&(q.disabled=!0,q.textContent="Connecting…");try{await $e({id:Y,mode:"ssh",ssh:O}),a=!1,await g()}catch(U){v(U),q&&(q.disabled=!1,q.textContent="Add server")}}function v(i){let l=n.querySelector("#targets-error");l||(u.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),l=n.querySelector("#targets-error")),l.textContent=String(i instanceof Error?i.message:i)}function f(){var i;(i=n.querySelector("#targets-error"))==null||i.remove()}return()=>{r=!0}}function pt(n,r){const a=n.wire,e=n.mode==="local"?"this machine":"SSH",d=n.mode==="ssh"&&n.ssh?`${s(n.ssh.User)}@${s(n.ssh.Host)}`:e;let u,g;if(!a)u=S("not set up","neutral"),g=`<a class="btn" href="#/setup/${encodeURIComponent(n.id)}">Run setup wizard</a>`;else{const m=r.networks.find(C=>C.ChainID===a.ChainID),P=m?m.Name:`chain ${a.ChainID}`;u=`${S(P,"ok")} ${S(a.ExecID,"neutral")} ${S(a.BeaconID,"neutral")}${a.Archive?" "+S("archive","warn"):""}`,g=`
      <a class="btn" href="#/dash/${encodeURIComponent(n.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(n.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(n.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${s(n.id)}</h2>
      <p class="muted">${d}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${g}
        <button class="btn btn-danger" data-action="delete-target" data-id="${s(n.id)}">Remove</button>
      </div>
    </div>
  `}function ht(){return`
    <form class="card" id="ssh-add-form" onsubmit="return false">
      <h3>Add server over SSH</h3>
      <label>
        Host
        <input id="ssh-host" type="text" placeholder="203.0.113.10" autocomplete="off" />
      </label>
      <label>
        User
        <input id="ssh-user" type="text" placeholder="root" autocomplete="off" />
      </label>
      <label>
        Private key path
        <input id="ssh-key" type="text" placeholder="/home/me/.ssh/id_ed25519" autocomplete="off" />
      </label>
      <label>
        Port <span class="muted">(optional, default 22)</span>
        <input id="ssh-port" type="text" inputmode="numeric" placeholder="22" autocomplete="off" />
      </label>
      <label>
        Target name <span class="muted">(optional, defaults to the host)</span>
        <input id="ssh-id" type="text" placeholder="my-node" autocomplete="off" />
      </label>
      <p class="muted small">
        The key never leaves this machine — only its path is stored, and the
        connection is dialed immediately so the host key can be pinned
        (trust-on-first-use) before it's saved.
      </p>
      <button class="btn" type="button" id="ssh-submit" data-action="add-ssh">Add server</button>
    </form>
  `}function ft(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const pe=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],re=8545,se=5052,oe=30303,mt=[369,943,1],Ee={369:"default",943:"practise here first"};function vt(n,r){let a=!1;const e={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${s(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${D()}</div>`;const d=n.querySelector("#wizard-body"),u=n.querySelector("#wizard-footer");Z(n,(c,t)=>{Q(c,t)}),Ce(n,(c,t)=>{c==="exec-select"?e.execId=t:c==="beacon-select"&&(e.beaconId=t),m()}),n.addEventListener("change",c=>{const t=c.target;t instanceof HTMLInputElement&&(t.id==="data-dir-input"?(Y(),l()):t.id==="checkpoint-toggle"?(e.checkpoint=t.checked,m()):t.id==="exec-snapshot-toggle"&&(e.execSnapshot=t.checked,m()))}),g();async function g(){try{const[c,t]=await Promise.all([V(),X()]);if(a)return;e.catalog=c;const o=t.find(p=>p.id===r);o!=null&&o.wire&&(e.chainId=o.wire.ChainID,e.execId=o.wire.ExecID,e.beaconId=o.wire.BeaconID,e.archive=o.wire.Archive,o.wire.ExecHTTPPort&&(e.execHTTPPort=String(o.wire.ExecHTTPPort)),o.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(o.wire.BeaconHTTPPort)),o.wire.ExecP2PPort&&(e.execP2PPort=String(o.wire.ExecP2PPort)),o.wire.RPCBindAddr&&(e.rpcBindAddr=o.wire.RPCBindAddr)),m()}catch(c){if(a)return;e.loadError=String(c instanceof Error?c.message:c),m()}}function m(){if(e.loadError){d.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(d.innerHTML=`
      ${de(e.step)}
      ${C()}
    `,P())}function P(){var t;const c=(t=e.catalog)==null?void 0:t.networks.find(o=>o.ChainID===e.chainId);u.innerHTML=c?D(c.Name,c.LearnURL):D()}function C(){switch(e.step){case"network":return E();case"clients":return L();case"mode":return _();case"review":return J();case"run":return G()}}function E(){const c=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${mt.map(o=>{const p=c.networks.find(x=>x.ChainID===o);if(!p)return"";const h=e.chainId===o,$=Ee[o]?S(Ee[o],o===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${h?"selected":""}" data-action="pick-network" data-chain-id="${o}" type="button">
          <h3>${s(p.Name)} <span class="muted">(chain ${o})</span></h3>
          ${$}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function L(){const c=e.catalog,t=c.networks.find(h=>h.ChainID===e.chainId);if(!t)return'<p class="error">Unknown network.</p>';(e.execId===null||!t.ExecClients.includes(e.execId))&&(e.execId=t.ExecClients[0]??null),(e.beaconId===null||!t.BeaconClients.includes(e.beaconId))&&(e.beaconId=t.BeaconClients[0]??null);const o=t.ExecClients.map(h=>y(h,c)),p=t.BeaconClients.map(h=>y(h,c));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${s(t.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${he("exec-select",o,e.execId)}
        </label>
        ${R(e.execId,c)}
        <label>
          Beacon client
          ${he("beacon-select",p,e.beaconId)}
        </label>
        ${R(e.beaconId,c)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(c){return c<=0?"—":c>=1?`~${c.toFixed(1)} TB`:`~${Math.round(c*1e3)} GB`}const T=1.1;function v(c){const t=c.ArchiveSizeTB*1e12*T;return{archive:t,full:t/2}}function f(c,t){if(!c)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${s(t)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${s(t)}</code>: ${s(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==t)return"";const o=v(c),p=e.freeBytes>=o.archive,h=e.freeBytes>=o.full,$=`<p class="muted small">Free at <code>${s(t)}</code>: <strong>${W(e.freeBytes)}</strong> — archive ${p?"fits":"won't fit"} (${w(c.ArchiveSizeTB)}), full ${h?"fits":"won't fit"} (${w(c.ArchiveSizeTB/2)}).</p>`;let x="";return e.downgradeNote?x=`<p class="banner banner-warn">${s(e.downgradeNote)}</p>`:h||(x=`<p class="banner banner-warn">Neither full (${w(c.ArchiveSizeTB/2)}) nor archive (${w(c.ArchiveSizeTB)}) fits the free space here — choose a location with more room.</p>`),$+x}function i(c,t){if(e.downgradeNote=null,!c||e.freeBytes===null)return;const o=v(c);e.archive&&e.freeBytes<o.archive&&e.freeBytes>=o.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${t} for archive (${w(c.ArchiveSizeTB)}) — switched to Full (${w(c.ArchiveSizeTB/2)}). Pick a location with more room to run archive.`)}async function l(){var o;if(e.chainId===null)return;const c=(o=e.catalog)==null?void 0:o.networks.find(p=>p.ChainID===e.chainId),t=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,m();try{const{freeBytes:p}=await Ne(e.targetId,t);if(a)return;e.freeBytes=p,e.probedPath=t,i(c,t)}catch(p){if(a)return;e.freeBytes=null,e.probedPath=t,e.diskError=String(p instanceof Error?p.message:p)}e.diskProbing=!1,m()}function b(c){return c?/^https?:\/\/.+/i.test(c)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function y(c,t){const o=t.clients.find(p=>p.id===c);return{value:c,label:o?`${o.id} — ${H(o.repo)}`:c}}function H(c){const t=c.split("/");return t.length>=4?t[3]:c}function R(c,t){const o=c?t.clients.find(h=>h.id===c):void 0;if(!o)return"";const p=o.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(o.repo)}" target="_blank" rel="noopener noreferrer">${s(p)}</a></p>`}function _(){var A,k,M;const c=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",t=(A=e.catalog)==null?void 0:A.networks.find(N=>N.ChainID===e.chainId),o=((M=(k=e.catalog)==null?void 0:k.clients.find(N=>N.id===e.execId))==null?void 0:M.snapshotSupported)??!1,p=(t==null?void 0:t.ArchiveSizeTB)??0,h=t?w(p/2):"Smaller",$=t?w(p):"Much larger",x=t?` on ${s(t.Name)}`:"",B=t?e.checkpoint?t.SyncLabel:t.GenesisSyncLabel:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <p class="muted">
          Both modes run a fully-validating node — same security, same current-state RPC.
          The difference is how much <strong>historical</strong> state is kept.
        </p>

        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="checkpoint-toggle" ${e.checkpoint?"checked":""} />
            <span><strong>Consensus checkpoint sync (beacon client)</strong> — start near the chain head in minutes (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.</span>
          </label>
          <p class="muted small">This applies to the beacon/consensus client (e.g. lighthouse-pulse) — not the execution client, which uses a snapshot below.</p>
          ${t?`<p class="sync-estimate">⏱ Estimated initial sync${x}: <strong>${s(B)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${s((t==null?void 0:t.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${s((t==null?void 0:t.CheckpointURL)??"")}" value="${s(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${s(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${o?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${e.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${e.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${s(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${s(e.snapshotKeyError)}</p>`:""}
                 <p class="muted small">Get a free key at <a href="https://valve.city" target="_blank" rel="noopener noreferrer">valve.city</a>.</p>`:""}
        </div>`:""}

        <details class="advanced">
          <summary>Full — current-state lookups (recent blocks) · Archive — full historical state &amp; indexing</summary>
          <table class="compare-table">
            <thead>
              <tr><th>What you get</th><th>Full</th><th>Archive</th></tr>
            </thead>
            <tbody>
              <tr><th>Current state &amp; recent blocks</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
              <tr><th>Send transactions, normal RPC</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
              <tr><th>Historical state (balances, <code>eth_call</code>) at any past block</th><td class="limited">Recent only (~128 blocks)</td><td class="yes">Full history</td></tr>
              <tr><th>Tracing / <code>debug_trace</code> on old blocks</th><td class="limited">Recent only</td><td class="yes">Full history</td></tr>
              <tr><th>Approx. disk footprint${x}</th><td class="yes">${h}</td><td class="limited">${$}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          <p class="muted small">Disk sizes are rough baselines — they vary by client and setup.</p>
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${$}${t?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${h}${t?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${s(c)})</span>
            <input id="data-dir-input" type="text" placeholder="${s(c)}" value="${s(e.dataDir)}" />
          </label>
          ${f(t,e.dataDir||c)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${s(c)}/jwt.hex" value="${s(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${re})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${re}" value="${s(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${s(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${se})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${se}" value="${s(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${s(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${oe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${oe}" value="${s(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${s(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${s(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${s(e.rpcBindAddrError)}</p>`:""}
          <p class="muted small">
            Leave any of these blank to use the default. The engine API port (8551) is fixed and
            loopback-only — it isn't configurable. Set the RPC bind address to this box's
            <strong>Tailscale IP</strong> (or another trusted overlay address) to reach the node's
            exec/beacon RPC from your own machine without an SSH tunnel. Note: the RPC is
            <strong>unauthenticated</strong>, so anyone on that network can drive the node — only
            bind to a trusted, private overlay, never a public address.
          </p>
        </details>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-clients">Back</button>
          <button class="btn" data-action="goto-review">Next: review</button>
        </div>
      </section>
    `}function J(){const t=e.catalog.networks.find(N=>N.ChainID===e.chainId),o=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,p=e.jwtPath||`${o}/jwt.hex`,h=pe.map(N=>`<li>${s(N.title)}</li>`).join(""),$=z(e.execHTTPPort,re),x=z(e.beaconHTTPPort,se),B=z(e.execP2PPort,oe),A=$||x||B?`<tr><th>Non-default ports</th><td>${[$?`exec HTTP ${$}`:null,x?`beacon HTTP ${x}`:null,B?`exec p2p ${B}`:null].filter(N=>N!==null).map(s).join(", ")}</td></tr>`:"",{addr:k}=O(e.rpcBindAddr),M=k?`<tr><th>RPC bind address</th><td><code>${s(k)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${s(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${s((t==null?void 0:t.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${s(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${s(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${s(o)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${s(p)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${s(e.checkpointUrl||(t==null?void 0:t.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${A}
            ${M}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${h}</ol>
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function G(){const t=e.catalog.networks.find(k=>k.ChainID===e.chainId),o=t==null?void 0:t.LearnURL,p=new Set(e.events.filter(k=>k.done).map(k=>k.stepId)),h=new Set(e.events.filter(k=>k.err).map(k=>k.stepId)),$=new Map;for(const k of e.events){if(!k.line)continue;const M=$.get(k.stepId)??[];M.push(k.line),$.set(k.stepId,M)}const x=pe.map(k=>{var be;const M=p.has(k.id),N=h.has(k.id),Re=N?S("failed","bad"):M?S("done","ok"):S("pending","neutral"),ve=($.get(k.id)??[]).slice(-5),ge=(be=e.events.find(ae=>ae.stepId===k.id&&ae.err))==null?void 0:be.err,Be=k.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${o?` <a href="${s(o)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${M?"step-done":""} ${N?"step-error":""}">
          <div class="step-head">${Re} <strong>${s(k.title)}</strong></div>
          ${Be}
          ${ve.length?`<pre class="step-log">${ve.map(ae=>s(ae)).join(`
`)}</pre>`:""}
          ${ge?`<p class="error small">${s(ge)}</p>`:""}
        </li>
      `}).join(""),B=e.events.some(k=>k.err),A=pe.every(k=>p.has(k.id))||e.events.some(k=>k.stepId==="handshake"&&k.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${x}</ol>
        ${A&&!B?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        ${B?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function Q(c,t){switch(c){case"pick-network":e.chainId=Number(t.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m(),l();break;case"goto-review":if(Y(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){m();break}e.step="review",m();break;case"start-setup":le();break}}function Y(){const c=n.querySelectorAll('input[name="mode"]');for(const k of Array.from(c))k.checked&&(e.archive=k.value==="archive");const t=n.querySelector("#data-dir-input"),o=n.querySelector("#jwt-path-input");t&&(e.dataDir=t.value.trim()),o&&(e.jwtPath=o.value.trim());const p=n.querySelector("#exec-http-port-input"),h=n.querySelector("#beacon-http-port-input"),$=n.querySelector("#exec-p2p-port-input");p&&(e.execHTTPPort=p.value.trim()),h&&(e.beaconHTTPPort=h.value.trim()),$&&(e.execP2PPort=$.value.trim());const x=n.querySelector("#rpc-bind-addr-input");x&&(e.rpcBindAddr=x.value.trim());const B=n.querySelector("#checkpoint-url-input");B&&(e.checkpointUrl=B.value.trim());const A=n.querySelector("#snapshot-key-input");A&&(e.snapshotKey=A.value.trim()),e.execHTTPPortError=U(e.execHTTPPort).error??null,e.beaconHTTPPortError=U(e.beaconHTTPPort).error??null,e.execP2PPortError=U(e.execP2PPort).error??null,e.rpcBindAddrError=O(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?b(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function O(c){if(!c)return{};const t=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(c);return t?t.slice(1).every(o=>Number(o)<=255)?{addr:c}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(c)&&c.includes(":")?{addr:c}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const q=/^\d+$/;function U(c){if(!c)return{};if(!q.test(c))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const t=Number(c);return!Number.isInteger(t)||t<1||t>65535?{error:"Port must be between 1 and 65535."}:{port:t}}function z(c,t){const{port:o}=U(c);if(!(o===void 0||o===t))return o}async function le(){var $;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],($=e.streamStop)==null||$.call(e),e.streamStop=null,m();const c={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(c.DataDir=e.dataDir),e.jwtPath&&(c.JWTPath=e.jwtPath);const t=z(e.execHTTPPort,re),o=z(e.beaconHTTPPort,se),p=z(e.execP2PPort,oe);t!==void 0&&(c.ExecHTTPPort=t),o!==void 0&&(c.BeaconHTTPPort=o),p!==void 0&&(c.ExecP2PPort=p);const{addr:h}=O(e.rpcBindAddr);h!==void 0&&(c.RPCBindAddr=h),e.checkpoint?e.checkpointUrl&&(c.CheckpointURL=e.checkpointUrl):c.NoCheckpoint=!0,e.execSnapshot&&(c.ExecSnapshot=!0,c.SnapshotKey=e.snapshotKey);try{await Me(e.targetId,c)}catch(x){if(!(x instanceof fe&&x.status===409)){e.starting=!1,e.startError=String(x instanceof Error?x.message:x),m();return}}e.starting=!1,e.step="run",m(),e.streamStop=qe(e.targetId,x=>{a||(e.events.push(x),e.step==="run"&&m())})}function de(c){const t=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],p=t.map(h=>h.id).indexOf(c);return`
      <ol class="wizard-progress">
        ${t.map((h,$)=>`<li class="${$===p?"current":$<p?"past":"future"}">${s(h.label)}</li>`).join("")}
      </ol>
    `}return()=>{var c;a=!0,(c=e.streamStop)==null||c.call(e)}}const gt=document.querySelector("#app"),{contentEl:bt,setActiveNav:yt}=Ze(gt);let F=null;function $t(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"targets"};const[a,e]=r;return a==="setup"||a==="dash"||a==="logs"||a==="security"||a==="diag"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function K(n){const r=document.createElement("div");return bt.replaceChildren(r),n(r)}function Ie(){if(F){try{F()}catch{}F=null}const{screen:n,id:r}=$t();switch(yt(n),n){case"setup":if(!r){location.hash="#/targets";return}F=K(a=>vt(a,r));break;case"dash":if(!r){location.hash="#/targets";return}F=K(a=>rt(a,r));break;case"logs":if(!r){location.hash="#/targets";return}F=K(a=>st(a,r));break;case"security":if(!r){location.hash="#/targets";return}F=K(a=>it(a,r));break;case"diag":if(!r){location.hash="#/targets";return}F=K(a=>ot(a,r));break;case"settings":F=K(a=>lt(a));break;case"targets":default:F=K(a=>ut(a));break}}window.addEventListener("hashchange",Ie);Ie();
