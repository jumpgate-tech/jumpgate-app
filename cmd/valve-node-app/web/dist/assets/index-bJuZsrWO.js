var Le=Object.defineProperty;var Ie=(t,a,n)=>a in t?Le(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var ve=(t,a,n)=>Ie(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const p of document.querySelectorAll('link[rel="modulepreload"]'))e(p);new MutationObserver(p=>{for(const h of p)if(h.type==="childList")for(const y of h.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(p){const h={};return p.integrity&&(h.integrity=p.integrity),p.referrerPolicy&&(h.referrerPolicy=p.referrerPolicy),p.crossOrigin==="use-credentials"?h.credentials="include":p.crossOrigin==="anonymous"?h.credentials="omit":h.credentials="same-origin",h}function e(p){if(p.ep)return;p.ep=!0;const h=n(p);fetch(p.href,h)}})();function Be(){return H("/api/host")}function Y(){return H("/api/catalog")}function V(){return H("/api/targets")}function be(t){return H("/api/targets",{method:"POST",headers:te,body:JSON.stringify(t)})}function He(t){return H(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Re(t,a){return H(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(a)}`)}function Ae(t,a){return H(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:te,body:JSON.stringify(a)})}function De(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Ue(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Ne(t,a=200){return H(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function Me(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function ye(t,a){const n=a===void 0?{}:{lines:a};return H(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:te,body:JSON.stringify(n)})}function qe(t,a,n){return H(`/api/targets/${encodeURIComponent(t)}/services/${a}/${n}`,{method:"POST"})}function Fe(t,a){return H(`/api/targets/${encodeURIComponent(t)}/services/${a}/clear`,{method:"POST",headers:te,body:JSON.stringify({Confirm:a})})}function Oe(t){return H(`/api/targets/${encodeURIComponent(t)}/du`)}function ze(t){return H(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function _e(t){return H(`/api/targets/${encodeURIComponent(t)}/firewall`)}function je(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Ke(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function We(){return H("/api/settings")}function Je(t){return H("/api/settings",{method:"PUT",headers:te,body:JSON.stringify(t)})}class ue extends Error{constructor(n,e){super(e);ve(this,"status");this.name="ApiError",this.status=n}}const te={"Content-Type":"application/json"};async function H(t,a){const n=await fetch(t,a);if(!n.ok){let p=n.statusText||`HTTP ${n.status}`;try{const h=await n.json();h&&typeof h.error=="string"&&h.error&&(p=h.error)}catch{}throw new ue(n.status,p)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const $e="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function U(t,a){const n=t&&a&&a!==$e?` <span class="footer-sep">·</span> <a href="${s(a)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${s($e)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function Ge(t){t.innerHTML=`
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
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:p=>{for(const h of n)h.classList.toggle("active",h.dataset.nav===p)}}}function K(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ye(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ve(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const p=Math.floor(n/24),h=n%24;return`~${p}d ${h}h`}function B(t,a){return`<span class="badge badge-${a}">${s(t)}</span>`}function we(t){return`<span class="dot dot-${t}"></span>`}const ke=["B","KB","MB","GB","TB","PB"];function J(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let a=t,n=0;for(;a>=1024&&n<ke.length-1;)a/=1024,n++;const e=a<10?2:a<100?1:0;return`${a.toFixed(e)} ${ke[n]}`}async function pe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function X(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const p=e.dataset.action;p&&a(p,e,n)})}function de(t,a,n){const e=a.find(h=>h.value===n),p=a.map(h=>`
      <li class="dropdown-option${h.value===n?" selected":""}" role="option"
          aria-selected="${h.value===n}" data-value="${s(h.value)}">
        ${s(h.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${s(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${s(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${p}</ul>
    </div>
  `}function ee(t){t.querySelectorAll(".dropdown.open").forEach(a=>{var n;a.classList.remove("open"),(n=a.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function Se(t,a){t.addEventListener("click",p=>{const h=p.target,y=h.closest(".dropdown-trigger");if(y&&t.contains(y)){const S=y.closest(".dropdown"),C=!!S&&!S.classList.contains("open");ee(t),S&&C&&(S.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const v=h.closest(".dropdown-option");if(v&&t.contains(v)){const S=v.closest(".dropdown");ee(t),a((S==null?void 0:S.dataset.dropdown)??"",v.dataset.value??"");return}ee(t)});const n=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const h=p.target;(!h.closest(".dropdown")||!t.contains(h))&&ee(t)},e=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}p.key==="Escape"&&ee(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const Xe=85,ce={exec:"Execution",beacon:"Beacon"};function Ze(t,a){let n=!1,e=null,p=null,h=null,y=null,v=null,S=null,C=null,E=null;const L={exec:null,beacon:null};let w=null;t.innerHTML=`<h1>Dashboard: ${s(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${U()}</div>`;const T=t.querySelector("#dash-body"),b=t.querySelector("#dash-footer");T.addEventListener("click",r=>{const d=r.target.closest("[data-action]");if(!d||!T.contains(d))return;const g=d.dataset.action;if(g==="svc-action"){const f=d.dataset.svc,P=d.dataset.kind;f&&P&&_(f,P)}else if(g==="open-clear"){const f=d.dataset.svc;f&&ie(f)}else if(g==="copy"){const f=d.dataset.copy;f&&oe(d,f)}else g==="retry-du"?c():g==="retry-endpoints"&&u()}),l();async function l(){let r,d;try{const[f,P]=await Promise.all([V(),Y()]);r=f.find(I=>I.id===a),d=P}catch(f){if(n)return;T.innerHTML=`<p class="error">Failed to load target: ${s(String(f))}</p>`;return}if(n)return;if(!r){T.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!r.wire){T.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const g=d==null?void 0:d.networks.find(f=>f.ChainID===r.wire.ChainID);g&&(b.innerHTML=U(g.Name,g.LearnURL)),T.innerHTML='<p class="muted">Connecting…</p>',e=Ue(a,f=>{n||($(f),p=f,h=f,x())}),c(),u()}async function c(){S=null;try{v=await Oe(a)}catch(r){v=null,S=String(r instanceof Error?r.message:r)}n||x()}async function u(){E=null;try{C=await ze(a)}catch(r){C=null,E=String(r instanceof Error?r.message:r)}n||x()}function $(r){if(!p)return;const d=(new Date(r.at).getTime()-new Date(p.at).getTime())/1e3,g=r.execHead-p.execHead;if(d>0&&g>=0){const f=g/d;y=y===null?f:y*.7+f*.3}}function x(){if(!h)return;const r=h;T.innerHTML=`
      <div class="card-grid">
        ${M(r)}
        ${q(r)}
        ${z(r)}
        ${Z(r)}
        ${Q(r)}
        ${j()}
        ${O(r)}
      </div>
      <p class="muted small">Last updated ${s(new Date(r.at).toLocaleTimeString())}</p>
    `}function R(r){const g=r.refHead>0?r.refHead-r.execHead:null,f=g!==null&&g>0&&y&&y>0?Ve(g/y):g!==null&&g<=0?"caught up":"—";return{lag:g,eta:f}}function M(r){const{lag:d,eta:g}=R(r);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${r.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${K(r.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${d!==null?K(r.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${d!==null?K(Math.max(d,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${g}</dd></div>
        </dl>
      </div>
    `}function q(r){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${r.beaconDistance===0?B("synced","ok"):B("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${K(r.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${K(r.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function z(r){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${K(r.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${K(r.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Z(r){const d=r.diskUsedPct>=Xe;return`
      <div class="card ${d?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${d?"meter-warn":""}" style="width:${Math.min(r.diskUsedPct,100)}%"></div></div>
        <p>${Ye(r.diskUsedPct)} used</p>
      </div>
    `}function Q(r){if(S)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${s(S)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!v)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const d=v.ExpectedExecBytes>0?Math.min(v.ExecBytes/v.ExpectedExecBytes*100,100):0,g=v.ExpectedBeaconBytes>0?Math.min(v.BeaconBytes/v.ExpectedBeaconBytes*100,100):0,{lag:f,eta:P}=R(r),I=f!==null&&f>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${J(v.ExecBytes)} of ~${J(v.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${d}%"></div></div>
        ${I?`<p class="muted small">Estimated time remaining: ${s(P)}</p>`:""}
        <p class="muted small">Beacon — ${J(v.BeaconBytes)} of ~${J(v.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${J(v.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(v.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(v.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function j(){if(E)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(E)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!C)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const r=C,d=r.ExecReachable&&!r.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",g=r.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${s(r.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${we(r.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${we(r.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.BeaconHTTP)}">Copy</button>
        </div>
        ${d}
        ${g}
      </div>
    `}function A(r,d){const g=ce[r],f=L[r],P=(I,k,N)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${r}" data-kind="${I}" ${f!==null||N?"disabled":""}>${f===I?G():s(k)}</button>`;return`
      <div class="service-row">
        <span>${s(g)} ${d?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${P("start","Start",d)}
          ${P("stop","Stop",!d)}
          ${P("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${r}" ${f!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function O(r){return`
      <div class="card">
        <h3>Services</h3>
        ${A("exec",r.execActive)}
        ${A("beacon",r.beaconActive)}
        ${w?`<p class="error small">${s(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(a)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(a)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(a)}">Diagnostics →</a>
        </p>
      </div>
    `}function G(){return'<span class="spinner" aria-label="working"></span>'}async function _(r,d){if(L[r]===null){L[r]=d,w=null,x();try{await qe(a,r,d)}catch(g){w=`${ce[r]} ${d} failed: ${g instanceof Error?g.message:String(g)}`}L[r]=null,n||x()}}async function oe(r,d){const g=await pe(d),f=r.textContent;r.textContent=g?"Copied!":"Copy failed",setTimeout(()=>{n||(r.textContent=f)},1500)}function ie(r){const d=ce[r],g=v?J(r==="exec"?v.ExecBytes:v.BeaconBytes):"unknown (disk usage hasn't loaded)";o(`
        <h2>Clear ${s(d)} data</h2>
        <p class="error">
          This stops the ${s(d.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(g)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(r)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,I=>{if(I==="cancel"){m();return}I==="confirm"&&i(r)});const f=document.getElementById("clear-confirm-input"),P=document.getElementById("clear-confirm-btn");f==null||f.addEventListener("input",()=>{P&&(P.disabled=f.value.trim()!==r)}),f==null||f.focus()}async function i(r){const d=document.getElementById("clear-confirm-btn");d&&(d.disabled=!0,d.textContent="Clearing…");try{await Fe(a,r),m(),c()}catch(g){const f=document.querySelector("#clear-modal .modal");if(f){const P=document.createElement("p");P.className="error small",P.textContent=`Clear failed: ${g instanceof Error?g.message:String(g)}`,f.appendChild(P)}d&&(d.disabled=!1,d.textContent="Clear and resync")}}function o(r,d){m();const g=document.createElement("div");g.className="modal-overlay",g.id="clear-modal",g.innerHTML=`<div class="modal">${r}</div>`,g.addEventListener("click",f=>{const P=f.target.closest("[data-modal-action]");P!=null&&P.dataset.modalAction&&d(P.dataset.modalAction),f.target===g&&d("cancel")}),document.body.appendChild(g)}function m(){var r;(r=document.getElementById("clear-modal"))==null||r.remove()}return()=>{n=!0,e==null||e(),m()}}const xe=500,Te="valve-node-app.explain-consent";function Qe(t,a){let n=!1,e=null;const p=[];t.innerHTML=`
    <h1>Logs: ${s(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${U()}</div>
  `;const h=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");X(t,l=>{l==="explain"&&E()}),v();async function v(){let l,c;try{const[$,x]=await Promise.all([V(),Y()]);l=$.find(R=>R.id===a),c=x}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${s(String($))}</p>`;return}if(n)return;if(!l){h.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const u=c==null?void 0:c.networks.find($=>$.ChainID===l.wire.ChainID);u&&(y.innerHTML=U(u.Name,u.LearnURL));try{const $=await Ne(a,200);if(n)return;p.push(...$)}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load logs: ${s(String($))}</p>`;return}S(),e=Me(a,$=>{n||(p.push($),p.length>xe&&p.splice(0,p.length-xe),S())})}function S(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical");h.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${p.map(C).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${B(String(l.length),l.length?"bad":"neutral")}</h2>
          <div class="log-lines">${l.length?l.slice().reverse().map(C).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const c=h.querySelector(".log-lines");c&&(c.scrollTop=c.scrollHeight)}function C(l){const c=l.severity||"info",u=l.learnUrl?` <a href="${s(l.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(c)}">
        <span class="log-time">${s(new Date(l.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(l.unit)}</span>
        <span class="log-sev">${s(c)}</span>
        <span class="log-text">${s(l.line)}</span>
        ${l.explain?`<div class="log-explain">${s(l.explain)}${u}</div>`:""}
      </div>
    `}async function E(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical").map(u=>u.line).slice(-40);if(!(localStorage.getItem(Te)==="1")){L(l);return}await w(l)}function L(l){const c=l.length?`<pre class="explain-excerpt">${l.map(u=>s(u)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';T(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${c}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,u=>{u==="proceed"?(localStorage.setItem(Te,"1"),b(),w(l)):b()})}async function w(l){T('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const c=l.length?await ye(a,l):await ye(a);if(n)return;T(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(c.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${c.sentExcerpt.map(u=>s(u)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&b()})}catch(c){if(n)return;if(c instanceof ue&&c.status===409){T(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,u=>{u==="close"&&b()});return}T(`
        <h2>Explain failed</h2>
        <p class="error">${s(c instanceof Error?c.message:String(c))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&b()})}}function T(l,c){b();const u=document.createElement("div");u.className="modal-overlay",u.id="explain-modal",u.innerHTML=`<div class="modal">${l}</div>`,u.addEventListener("click",$=>{const x=$.target.closest("[data-modal-action]");x!=null&&x.dataset.modalAction&&c(x.dataset.modalAction),$.target===u&&c("cancel")}),document.body.appendChild(u)}function b(){var l;(l=document.getElementById("explain-modal"))==null||l.remove()}return()=>{n=!0,e==null||e(),b()}}function et(t,a){let n=!1,e=null,p=null,h=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${s(a)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${U()}</div>`;const v=t.querySelector("#diag-body"),S=t.querySelector("#diag-footer");X(t,(c,u)=>{var $;if(c==="run")E();else if(c==="toggle")($=u.closest(".check-item"))==null||$.classList.toggle("expanded");else if(c==="copy"){const x=u.dataset.copy;x&&l(u,x)}}),C();async function C(){let c,u;try{const[x,R]=await Promise.all([V(),Y()]);c=x.find(M=>M.id===a),u=R}catch(x){if(n)return;v.innerHTML=`<p class="error">Failed to load target: ${s(String(x))}</p>`;return}if(n)return;if(!c){v.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const $=u==null?void 0:u.networks.find(x=>x.ChainID===c.wire.ChainID);$&&(S.innerHTML=U($.Name,$.LearnURL));try{e=await Ke(a),y=!0}catch(x){p=String(x instanceof Error?x.message:x)}n||L()}async function E(){h=!0,p=null,L();try{e=await je(a),y=!0}catch(c){p=String(c instanceof Error?c.message:c)}h=!1,n||L()}function L(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${h?"disabled":""}>${h?"Running…":"Run diagnostics"}</button>
      </div>
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${w()}
    `}function w(){if(!y&&!p)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const c=new Date(e.at).toLocaleString(),u=e.failedId?`<p><strong>Failed at: ${s(T(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(c)} — trigger: ${s(e.trigger)}</p>
      ${u}
      <ul class="check-list">${e.items.map(b).join("")}</ul>
    `}function T(c){var u;return((u=e==null?void 0:e.items.find($=>$.ID===c))==null?void 0:u.Title)??c}function b(c){const u=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral",$=c.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${$?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${B($?"failed here":c.Status,u)}
          <strong>${s(c.Title)}</strong>
          <span class="muted small check-detail-inline">${s(c.Detail)}</span>
        </button>
        <div class="check-body">
          <details${$?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${s(c.Why)}</p>
          </details>
          ${c.Fix?`
                <details${$?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(c.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(c.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function l(c,u){const $=await pe(u),x=c.textContent;c.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=x)},1500)}return()=>{n=!0}}function tt(t,a){let n=!1,e=[],p=null,h=!1,y=!1;t.innerHTML=`<h1>Security: ${s(a)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${U()}</div>`;const v=t.querySelector("#sec-body"),S=t.querySelector("#sec-footer");X(t,(b,l)=>{var c;if(b==="rerun")E();else if(b==="toggle")(c=l.closest(".check-item"))==null||c.classList.toggle("expanded");else if(b==="copy"){const u=l.dataset.copy;u&&T(l,u)}}),C();async function C(){let b,l;try{const[u,$]=await Promise.all([V(),Y()]);b=u.find(x=>x.id===a),l=$}catch(u){if(n)return;v.innerHTML=`<p class="error">Failed to load target: ${s(String(u))}</p>`;return}if(n)return;if(!b){v.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!b.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const c=l==null?void 0:l.networks.find(u=>u.ChainID===b.wire.ChainID);c&&(S.innerHTML=U(c.Name,c.LearnURL)),await E()}async function E(){h=!0,p=null,L();try{e=await _e(a),y=!0}catch(b){p=String(b instanceof Error?b.message:b)}h=!1,n||L()}function L(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${h?"disabled":""}>${h?"Re-running…":"Re-run checks"}</button>
      </div>
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${!y&&h?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function w(b){const l=b.Status==="pass"?"ok":b.Status==="fail"?"bad":b.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(b.Status,l)}
          <strong>${s(b.Title)}</strong>
          <span class="muted small check-detail-inline">${s(b.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${s(b.Why)}</p>
          </details>
          ${b.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(b.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(b.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(b,l){const c=await pe(l),u=b.textContent;b.textContent=c?"Copied!":"Copy failed",setTimeout(()=>{n||(b.textContent=u)},1500)}return()=>{n=!0}}const nt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function at(t){let a=!1,n=!1,e=!1,p=null,h=!1,y=null,v=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${U()}`;const S=t.querySelector("#settings-body");X(t,w=>{if(w==="save"&&L(),w==="clear-key"){if(!y)return;n=!0;const T=t.querySelector("#ai-key");T&&(T.value=""),E(y)}}),Se(t,(w,T)=>{w!=="ai-provider"||!y||(v=T,h=!1,E(y))}),C();async function C(){try{const w=await We();if(a)return;y=w,E(w)}catch(w){if(a)return;S.innerHTML=`<p class="error">Failed to load settings: ${s(String(w))}</p>`}}function E(w){var l;const T=v??w.aiProvider;S.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${de("ai-provider",nt.map(c=>({value:c.value,label:c.label})),T)}
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
        ${p?`<p class="error">${s(p)}</p>`:""}
        ${h?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const b=t.querySelector("#ai-key");b==null||b.addEventListener("input",()=>{n=!0,h=!1}),(l=t.querySelector("#ref-rpc-base"))==null||l.addEventListener("input",()=>{h=!1})}async function L(){const w=t.querySelector("#ai-key"),T=t.querySelector("#ref-rpc-base");if(!w||!T||!y)return;const b={aiProvider:v??y.aiProvider,refRpcBase:T.value.trim()};n&&(b.aiKey=w.value),e=!0,p=null,h=!1,E(y);try{const l=await Je(b);if(a)return;y=l,n=!1,e=!1,h=!0,E(l)}catch(l){if(a)return;e=!1,p=String(l instanceof Error?l.message:l),E(y)}}return()=>{a=!0}}const rt="local";function st(t){let a=!1,n=!1,e="",p=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${U()}
  `;const h=t.querySelector("#targets-body");X(t,(l,c)=>{C(l,c)}),y();async function y(){try{const[l,c,u]=await Promise.all([V(),Y(),Be()]);if(a)return;e=u.os,S(l,c)}catch(l){if(a)return;h.innerHTML=`<p class="error">Failed to load machines: ${s(String(l))}</p>`}}function v(){p&&S(p.targets,p.catalog)}function S(l,c){p={targets:l,catalog:c};const u=e==="linux",$=[...l].sort((q,z)=>(q.mode==="local"?-1:0)-(z.mode==="local"?-1:0)),x=$.length?`<div class="card-grid">${$.map(q=>ot(q,c)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${u?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node-app is running here as your <strong>controller</strong> — add a Linux server over SSH to run a node, or add this machine to walk the setup (it will need a Linux host to finish)."}
          </p>
        </div>
      `,M=`
      <div class="add-actions">
        ${u?'<button class="btn" data-action="add-local">Add this machine</button>':`<button class="btn btn-ghost" data-action="add-local" title="Setup needs a Linux host — this machine can drive remote nodes; local setup won't complete here">Add this machine</button>`}
        <button class="btn${u?" btn-ghost":""}" data-action="toggle-ssh">
          ${n?"Cancel":"Add a server (SSH)"}
        </button>
      </div>
    `;h.innerHTML=`
      <section class="section">
        <div class="section-head">
          <h2>Your machines</h2>
          ${M}
        </div>
        ${!u&&e?`<p class="muted small">This machine (${s(e)}) runs valve-node-app as a <strong>controller</strong>. Node hosts must be Linux — "Add this machine" is available to walk the flow, but setup only completes on a Linux host.</p>`:""}
        ${n?it():""}
        ${x}
      </section>
    `}async function C(l,c){var u;if(l==="add-local"){await E();return}if(l==="delete-target"){const $=c.dataset.id;if(!$||!confirm(`Remove target "${$}"? This does not touch anything already running on it.`))return;await L($);return}if(l==="toggle-ssh"){n=!n,b(),v(),n&&((u=t.querySelector("#ssh-host"))==null||u.focus());return}l==="add-ssh"&&await w()}async function E(){if(b(),!(e!=="linux"&&!confirm("This machine ("+e+") isn't a Linux host, so node setup can't complete here — it's only useful for previewing the setup wizard. Add it anyway?")))try{await be({id:rt,mode:"local"}),await y()}catch(l){T(l)}}async function L(l){try{await He(l),await y()}catch(c){T(c)}}async function w(){const l=t.querySelector("#ssh-host"),c=t.querySelector("#ssh-user"),u=t.querySelector("#ssh-key"),$=t.querySelector("#ssh-port"),x=t.querySelector("#ssh-id");if(!l||!c||!u||!$||!x)return;const R=l.value.trim(),M=c.value.trim(),q=u.value.trim(),z=$.value.trim(),Z=x.value.trim();if(b(),!R||!M||!q){T(new Error("host, user, and key path are required"));return}const Q=Z||ct(R),j={Host:R,User:M,KeyPath:q};if(z){const O=Number.parseInt(z,10);if(!Number.isFinite(O)||O<=0){T(new Error("port must be a positive number"));return}j.Port=O}const A=t.querySelector("#ssh-submit");A&&(A.disabled=!0,A.textContent="Connecting…");try{await be({id:Q,mode:"ssh",ssh:j}),n=!1,await y()}catch(O){T(O),A&&(A.disabled=!1,A.textContent="Add server")}}function T(l){let c=t.querySelector("#targets-error");c||(h.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),c=t.querySelector("#targets-error")),c.textContent=String(l instanceof Error?l.message:l)}function b(){var l;(l=t.querySelector("#targets-error"))==null||l.remove()}return()=>{a=!0}}function ot(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",p=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let h,y;if(!n)h=B("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const v=a.networks.find(C=>C.ChainID===n.ChainID),S=v?v.Name:`chain ${n.ChainID}`;h=`${B(S,"ok")} ${B(n.ExecID,"neutral")} ${B(n.BeaconID,"neutral")}${n.Archive?" "+B("archive","warn"):""}`,y=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${s(t.id)}</h2>
      <p class="muted">${p}</p>
      <p>${h}</p>
      <div class="card-actions">
        ${y}
        <button class="btn btn-danger" data-action="delete-target" data-id="${s(t.id)}">Remove</button>
      </div>
    </div>
  `}function it(){return`
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
  `}function ct(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const le=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ae=8545,re=5052,se=30303,lt=[369,943,1],Pe={369:"default",943:"practise here first"};function dt(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${U()}</div>`;const p=t.querySelector("#wizard-body"),h=t.querySelector("#wizard-footer");X(t,(i,o)=>{Q(i,o)}),Se(t,(i,o)=>{i==="exec-select"?e.execId=o:i==="beacon-select"&&(e.beaconId=o),v()}),t.addEventListener("change",i=>{const o=i.target;o instanceof HTMLInputElement&&(o.id==="data-dir-input"?(j(),u()):o.id==="checkpoint-toggle"?(e.checkpoint=o.checked,v()):o.id==="exec-snapshot-toggle"&&(e.execSnapshot=o.checked,v()))}),y();async function y(){try{const[i,o]=await Promise.all([Y(),V()]);if(n)return;e.catalog=i;const m=o.find(r=>r.id===a);m!=null&&m.wire&&(e.chainId=m.wire.ChainID,e.execId=m.wire.ExecID,e.beaconId=m.wire.BeaconID,e.archive=m.wire.Archive,m.wire.ExecHTTPPort&&(e.execHTTPPort=String(m.wire.ExecHTTPPort)),m.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(m.wire.BeaconHTTPPort)),m.wire.ExecP2PPort&&(e.execP2PPort=String(m.wire.ExecP2PPort)),m.wire.RPCBindAddr&&(e.rpcBindAddr=m.wire.RPCBindAddr)),v()}catch(i){if(n)return;e.loadError=String(i instanceof Error?i.message:i),v()}}function v(){if(e.loadError){p.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(p.innerHTML=`
      ${ie(e.step)}
      ${C()}
    `,S())}function S(){var o;const i=(o=e.catalog)==null?void 0:o.networks.find(m=>m.ChainID===e.chainId);h.innerHTML=i?U(i.Name,i.LearnURL):U()}function C(){switch(e.step){case"network":return E();case"clients":return L();case"mode":return q();case"review":return z();case"run":return Z()}}function E(){const i=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${lt.map(m=>{const r=i.networks.find(f=>f.ChainID===m);if(!r)return"";const d=e.chainId===m,g=Pe[m]?B(Pe[m],m===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${d?"selected":""}" data-action="pick-network" data-chain-id="${m}" type="button">
          <h3>${s(r.Name)} <span class="muted">(chain ${m})</span></h3>
          ${g}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function L(){const i=e.catalog,o=i.networks.find(d=>d.ChainID===e.chainId);if(!o)return'<p class="error">Unknown network.</p>';(e.execId===null||!o.ExecClients.includes(e.execId))&&(e.execId=o.ExecClients[0]??null),(e.beaconId===null||!o.BeaconClients.includes(e.beaconId))&&(e.beaconId=o.BeaconClients[0]??null);const m=o.ExecClients.map(d=>x(d,i)),r=o.BeaconClients.map(d=>x(d,i));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${s(o.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${de("exec-select",m,e.execId)}
        </label>
        ${M(e.execId,i)}
        <label>
          Beacon client
          ${de("beacon-select",r,e.beaconId)}
        </label>
        ${M(e.beaconId,i)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(i){return i<=0?"—":i>=1?`~${i.toFixed(1)} TB`:`~${Math.round(i*1e3)} GB`}const T=1.1;function b(i){const o=i.ArchiveSizeTB*1e12*T;return{archive:o,full:o/2}}function l(i,o){if(!i)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${s(o)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${s(o)}</code>: ${s(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==o)return"";const m=b(i),r=e.freeBytes>=m.archive,d=e.freeBytes>=m.full,g=`<p class="muted small">Free at <code>${s(o)}</code>: <strong>${J(e.freeBytes)}</strong> — archive ${r?"fits":"won't fit"} (${w(i.ArchiveSizeTB)}), full ${d?"fits":"won't fit"} (${w(i.ArchiveSizeTB/2)}).</p>`;let f="";return e.downgradeNote?f=`<p class="banner banner-warn">${s(e.downgradeNote)}</p>`:d||(f=`<p class="banner banner-warn">Neither full (${w(i.ArchiveSizeTB/2)}) nor archive (${w(i.ArchiveSizeTB)}) fits the free space here — choose a location with more room.</p>`),g+f}function c(i,o){if(e.downgradeNote=null,!i||e.freeBytes===null)return;const m=b(i);e.archive&&e.freeBytes<m.archive&&e.freeBytes>=m.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${o} for archive (${w(i.ArchiveSizeTB)}) — switched to Full (${w(i.ArchiveSizeTB/2)}). Pick a location with more room to run archive.`)}async function u(){var m;if(e.chainId===null)return;const i=(m=e.catalog)==null?void 0:m.networks.find(r=>r.ChainID===e.chainId),o=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,v();try{const{freeBytes:r}=await Re(e.targetId,o);if(n)return;e.freeBytes=r,e.probedPath=o,c(i,o)}catch(r){if(n)return;e.freeBytes=null,e.probedPath=o,e.diskError=String(r instanceof Error?r.message:r)}e.diskProbing=!1,v()}function $(i){return i?/^https?:\/\/.+/i.test(i)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function x(i,o){const m=o.clients.find(r=>r.id===i);return{value:i,label:m?`${m.id} — ${R(m.repo)}`:i}}function R(i){const o=i.split("/");return o.length>=4?o[3]:i}function M(i,o){const m=i?o.clients.find(d=>d.id===i):void 0;if(!m)return"";const r=m.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(m.repo)}" target="_blank" rel="noopener noreferrer">${s(r)}</a></p>`}function q(){var I,k,N;const i=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",o=(I=e.catalog)==null?void 0:I.networks.find(D=>D.ChainID===e.chainId),m=((N=(k=e.catalog)==null?void 0:k.clients.find(D=>D.id===e.execId))==null?void 0:N.snapshotSupported)??!1,r=(o==null?void 0:o.ArchiveSizeTB)??0,d=o?w(r/2):"Smaller",g=o?w(r):"Much larger",f=o?` on ${s(o.Name)}`:"",P=o?e.checkpoint?o.SyncLabel:o.GenesisSyncLabel:"";return`
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
          ${o?`<p class="sync-estimate">⏱ Estimated initial sync${f}: <strong>${s(P)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${s((o==null?void 0:o.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${s((o==null?void 0:o.CheckpointURL)??"")}" value="${s(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${s(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${m?`
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
              <tr><th>Approx. disk footprint${f}</th><td class="yes">${d}</td><td class="limited">${g}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          <p class="muted small">Disk sizes are rough baselines — they vary by client and setup.</p>
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${g}${o?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${d}${o?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${s(i)})</span>
            <input id="data-dir-input" type="text" placeholder="${s(i)}" value="${s(e.dataDir)}" />
          </label>
          ${l(o,e.dataDir||i)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${s(i)}/jwt.hex" value="${s(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${ae})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${ae}" value="${s(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${s(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${re})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${re}" value="${s(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${s(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${se})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${se}" value="${s(e.execP2PPort)}" />
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
    `}function z(){const o=e.catalog.networks.find(D=>D.ChainID===e.chainId),m=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,r=e.jwtPath||`${m}/jwt.hex`,d=le.map(D=>`<li>${s(D.title)}</li>`).join(""),g=_(e.execHTTPPort,ae),f=_(e.beaconHTTPPort,re),P=_(e.execP2PPort,se),I=g||f||P?`<tr><th>Non-default ports</th><td>${[g?`exec HTTP ${g}`:null,f?`beacon HTTP ${f}`:null,P?`exec p2p ${P}`:null].filter(D=>D!==null).map(s).join(", ")}</td></tr>`:"",{addr:k}=A(e.rpcBindAddr),N=k?`<tr><th>RPC bind address</th><td><code>${s(k)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${s(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${s((o==null?void 0:o.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${s(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${s(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${s(m)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${s(r)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${s(e.checkpointUrl||(o==null?void 0:o.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${I}
            ${N}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${d}</ol>
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function Z(){const o=e.catalog.networks.find(k=>k.ChainID===e.chainId),m=o==null?void 0:o.LearnURL,r=new Set(e.events.filter(k=>k.done).map(k=>k.stepId)),d=new Set(e.events.filter(k=>k.err).map(k=>k.stepId)),g=new Map;for(const k of e.events){if(!k.line)continue;const N=g.get(k.stepId)??[];N.push(k.line),g.set(k.stepId,N)}const f=le.map(k=>{var ge;const N=r.has(k.id),D=d.has(k.id),he=D?B("failed","bad"):N?B("done","ok"):B("pending","neutral"),fe=(g.get(k.id)??[]).slice(-5),me=(ge=e.events.find(ne=>ne.stepId===k.id&&ne.err))==null?void 0:ge.err,Ce=k.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${m?` <a href="${s(m)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${N?"step-done":""} ${D?"step-error":""}">
          <div class="step-head">${he} <strong>${s(k.title)}</strong></div>
          ${Ce}
          ${fe.length?`<pre class="step-log">${fe.map(ne=>s(ne)).join(`
`)}</pre>`:""}
          ${me?`<p class="error small">${s(me)}</p>`:""}
        </li>
      `}).join(""),P=e.events.some(k=>k.err),I=le.every(k=>r.has(k.id))||e.events.some(k=>k.stepId==="handshake"&&k.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${f}</ol>
        ${I&&!P?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        ${P?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function Q(i,o){switch(i){case"pick-network":e.chainId=Number(o.dataset.chainId),e.execId=null,e.beaconId=null,v();break;case"goto-network":e.step="network",v();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",v();break;case"goto-mode":e.step="mode",v(),u();break;case"goto-review":if(j(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){v();break}e.step="review",v();break;case"start-setup":oe();break}}function j(){const i=t.querySelectorAll('input[name="mode"]');for(const k of Array.from(i))k.checked&&(e.archive=k.value==="archive");const o=t.querySelector("#data-dir-input"),m=t.querySelector("#jwt-path-input");o&&(e.dataDir=o.value.trim()),m&&(e.jwtPath=m.value.trim());const r=t.querySelector("#exec-http-port-input"),d=t.querySelector("#beacon-http-port-input"),g=t.querySelector("#exec-p2p-port-input");r&&(e.execHTTPPort=r.value.trim()),d&&(e.beaconHTTPPort=d.value.trim()),g&&(e.execP2PPort=g.value.trim());const f=t.querySelector("#rpc-bind-addr-input");f&&(e.rpcBindAddr=f.value.trim());const P=t.querySelector("#checkpoint-url-input");P&&(e.checkpointUrl=P.value.trim());const I=t.querySelector("#snapshot-key-input");I&&(e.snapshotKey=I.value.trim()),e.execHTTPPortError=G(e.execHTTPPort).error??null,e.beaconHTTPPortError=G(e.beaconHTTPPort).error??null,e.execP2PPortError=G(e.execP2PPort).error??null,e.rpcBindAddrError=A(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?$(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function A(i){if(!i)return{};const o=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(i);return o?o.slice(1).every(m=>Number(m)<=255)?{addr:i}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(i)&&i.includes(":")?{addr:i}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const O=/^\d+$/;function G(i){if(!i)return{};if(!O.test(i))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const o=Number(i);return!Number.isInteger(o)||o<1||o>65535?{error:"Port must be between 1 and 65535."}:{port:o}}function _(i,o){const{port:m}=G(i);if(!(m===void 0||m===o))return m}async function oe(){var g;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(g=e.streamStop)==null||g.call(e),e.streamStop=null,v();const i={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(i.DataDir=e.dataDir),e.jwtPath&&(i.JWTPath=e.jwtPath);const o=_(e.execHTTPPort,ae),m=_(e.beaconHTTPPort,re),r=_(e.execP2PPort,se);o!==void 0&&(i.ExecHTTPPort=o),m!==void 0&&(i.BeaconHTTPPort=m),r!==void 0&&(i.ExecP2PPort=r);const{addr:d}=A(e.rpcBindAddr);d!==void 0&&(i.RPCBindAddr=d),e.checkpoint?e.checkpointUrl&&(i.CheckpointURL=e.checkpointUrl):i.NoCheckpoint=!0,e.execSnapshot&&(i.ExecSnapshot=!0,i.SnapshotKey=e.snapshotKey);try{await Ae(e.targetId,i)}catch(f){if(!(f instanceof ue&&f.status===409)){e.starting=!1,e.startError=String(f instanceof Error?f.message:f),v();return}}e.starting=!1,e.step="run",v(),e.streamStop=De(e.targetId,f=>{n||(e.events.push(f),e.step==="run"&&v())})}function ie(i){const o=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],r=o.map(d=>d.id).indexOf(i);return`
      <ol class="wizard-progress">
        ${o.map((d,g)=>`<li class="${g===r?"current":g<r?"past":"future"}">${s(d.label)}</li>`).join("")}
      </ol>
    `}return()=>{var i;n=!0,(i=e.streamStop)==null||i.call(e)}}const ut=document.querySelector("#app"),{contentEl:pt,setActiveNav:ht}=Ge(ut);let F=null;function ft(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function W(t){const a=document.createElement("div");return pt.replaceChildren(a),t(a)}function Ee(){if(F){try{F()}catch{}F=null}const{screen:t,id:a}=ft();switch(ht(t),t){case"setup":if(!a){location.hash="#/targets";return}F=W(n=>dt(n,a));break;case"dash":if(!a){location.hash="#/targets";return}F=W(n=>Ze(n,a));break;case"logs":if(!a){location.hash="#/targets";return}F=W(n=>Qe(n,a));break;case"security":if(!a){location.hash="#/targets";return}F=W(n=>tt(n,a));break;case"diag":if(!a){location.hash="#/targets";return}F=W(n=>et(n,a));break;case"settings":F=W(n=>at(n));break;case"targets":default:F=W(n=>st(n));break}}window.addEventListener("hashchange",Ee);Ee();
