var Le=Object.defineProperty;var Ie=(t,r,n)=>r in t?Le(t,r,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[r]=n;var be=(t,r,n)=>Ie(t,typeof r!="symbol"?r+"":r,n);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const p of document.querySelectorAll('link[rel="modulepreload"]'))e(p);new MutationObserver(p=>{for(const h of p)if(h.type==="childList")for(const y of h.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(p){const h={};return p.integrity&&(h.integrity=p.integrity),p.referrerPolicy&&(h.referrerPolicy=p.referrerPolicy),p.crossOrigin==="use-credentials"?h.credentials="include":p.crossOrigin==="anonymous"?h.credentials="omit":h.credentials="same-origin",h}function e(p){if(p.ep)return;p.ep=!0;const h=n(p);fetch(p.href,h)}})();function Be(){return H("/api/host")}function Y(){return H("/api/catalog")}function V(){return H("/api/targets")}function ve(t){return H("/api/targets",{method:"POST",headers:te,body:JSON.stringify(t)})}function He(t){return H(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Re(t,r){return H(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(r)}`)}function Ae(t,r){return H(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:te,body:JSON.stringify(r)})}function De(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function Ue(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function Ne(t,r=200){return H(`/api/targets/${encodeURIComponent(t)}/logs?n=${r}`)}function Me(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function ye(t,r){const n=r===void 0?{}:{lines:r};return H(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:te,body:JSON.stringify(n)})}function qe(t,r,n){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/${n}`,{method:"POST"})}function Fe(t,r){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/clear`,{method:"POST",headers:te,body:JSON.stringify({Confirm:r})})}function Oe(t){return H(`/api/targets/${encodeURIComponent(t)}/du`)}function je(t){return H(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function ze(t){return H(`/api/targets/${encodeURIComponent(t)}/firewall`)}function _e(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function We(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Je(){return H("/api/settings")}function Ke(t){return H("/api/settings",{method:"PUT",headers:te,body:JSON.stringify(t)})}class ue extends Error{constructor(n,e){super(e);be(this,"status");this.name="ApiError",this.status=n}}const te={"Content-Type":"application/json"};async function H(t,r){const n=await fetch(t,r);if(!n.ok){let p=n.statusText||`HTTP ${n.status}`;try{const h=await n.json();h&&typeof h.error=="string"&&h.error&&(p=h.error)}catch{}throw new ue(n.status,p)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const $e="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function D(t,r){const n=t&&r&&r!==$e?` <span class="footer-sep">·</span> <a href="${s(r)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${s($e)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function Ge(t){t.innerHTML=`
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/targets">valve-node</a>
        <nav class="nav">
          <a href="#/targets" data-nav="targets">Targets</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;const r=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:p=>{for(const h of n)h.classList.toggle("active",h.dataset.nav===p)}}}function W(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ye(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ve(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const r=Math.round(t/60),n=Math.floor(r/60),e=r%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const p=Math.floor(n/24),h=n%24;return`~${p}d ${h}h`}function B(t,r){return`<span class="badge badge-${r}">${s(t)}</span>`}function we(t){return`<span class="dot dot-${t}"></span>`}const ke=["B","KB","MB","GB","TB","PB"];function K(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let r=t,n=0;for(;r>=1024&&n<ke.length-1;)r/=1024,n++;const e=r<10?2:r<100?1:0;return`${r.toFixed(e)} ${ke[n]}`}async function pe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function X(t,r){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const p=e.dataset.action;p&&r(p,e,n)})}function de(t,r,n){const e=r.find(h=>h.value===n),p=r.map(h=>`
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
  `}function ee(t){t.querySelectorAll(".dropdown.open").forEach(r=>{var n;r.classList.remove("open"),(n=r.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function Ee(t,r){t.addEventListener("click",p=>{const h=p.target,y=h.closest(".dropdown-trigger");if(y&&t.contains(y)){const E=y.closest(".dropdown"),C=!!E&&!E.classList.contains("open");ee(t),E&&C&&(E.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const b=h.closest(".dropdown-option");if(b&&t.contains(b)){const E=b.closest(".dropdown");ee(t),r((E==null?void 0:E.dataset.dropdown)??"",b.dataset.value??"");return}ee(t)});const n=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const h=p.target;(!h.closest(".dropdown")||!t.contains(h))&&ee(t)},e=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}p.key==="Escape"&&ee(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const Xe=85,ce={exec:"Execution",beacon:"Beacon"};function Ze(t,r){let n=!1,e=null,p=null,h=null,y=null,b=null,E=null,C=null,S=null;const L={exec:null,beacon:null};let w=null;t.innerHTML=`<h1>Dashboard: ${s(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${D()}</div>`;const x=t.querySelector("#dash-body"),v=t.querySelector("#dash-footer");x.addEventListener("click",a=>{const d=a.target.closest("[data-action]");if(!d||!x.contains(d))return;const g=d.dataset.action;if(g==="svc-action"){const f=d.dataset.svc,P=d.dataset.kind;f&&P&&z(f,P)}else if(g==="open-clear"){const f=d.dataset.svc;f&&ie(f)}else if(g==="copy"){const f=d.dataset.copy;f&&oe(d,f)}else g==="retry-du"?c():g==="retry-endpoints"&&u()}),l();async function l(){let a,d;try{const[f,P]=await Promise.all([V(),Y()]);a=f.find(I=>I.id===r),d=P}catch(f){if(n)return;x.innerHTML=`<p class="error">Failed to load target: ${s(String(f))}</p>`;return}if(n)return;if(!a){x.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!a.wire){x.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const g=d==null?void 0:d.networks.find(f=>f.ChainID===a.wire.ChainID);g&&(v.innerHTML=D(g.Name,g.LearnURL)),x.innerHTML='<p class="muted">Connecting…</p>',e=Ue(r,f=>{n||($(f),p=f,h=f,k())}),c(),u()}async function c(){E=null;try{b=await Oe(r)}catch(a){b=null,E=String(a instanceof Error?a.message:a)}n||k()}async function u(){S=null;try{C=await je(r)}catch(a){C=null,S=String(a instanceof Error?a.message:a)}n||k()}function $(a){if(!p)return;const d=(new Date(a.at).getTime()-new Date(p.at).getTime())/1e3,g=a.execHead-p.execHead;if(d>0&&g>=0){const f=g/d;y=y===null?f:y*.7+f*.3}}function k(){if(!h)return;const a=h;x.innerHTML=`
      <div class="card-grid">
        ${U(a)}
        ${N(a)}
        ${j(a)}
        ${Z(a)}
        ${Q(a)}
        ${_()}
        ${q(a)}
      </div>
      <p class="muted small">Last updated ${s(new Date(a.at).toLocaleTimeString())}</p>
    `}function R(a){const g=a.refHead>0?a.refHead-a.execHead:null,f=g!==null&&g>0&&y&&y>0?Ve(g/y):g!==null&&g<=0?"caught up":"—";return{lag:g,eta:f}}function U(a){const{lag:d,eta:g}=R(a);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${a.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${W(a.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${d!==null?W(a.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${d!==null?W(Math.max(d,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${g}</dd></div>
        </dl>
      </div>
    `}function N(a){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${a.beaconDistance===0?B("synced","ok"):B("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${W(a.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${W(a.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function j(a){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${W(a.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${W(a.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Z(a){const d=a.diskUsedPct>=Xe;return`
      <div class="card ${d?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${d?"meter-warn":""}" style="width:${Math.min(a.diskUsedPct,100)}%"></div></div>
        <p>${Ye(a.diskUsedPct)} used</p>
      </div>
    `}function Q(a){if(E)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${s(E)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!b)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const d=b.ExpectedExecBytes>0?Math.min(b.ExecBytes/b.ExpectedExecBytes*100,100):0,g=b.ExpectedBeaconBytes>0?Math.min(b.BeaconBytes/b.ExpectedBeaconBytes*100,100):0,{lag:f,eta:P}=R(a),I=f!==null&&f>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${K(b.ExecBytes)} of ~${K(b.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${d}%"></div></div>
        ${I?`<p class="muted small">Estimated time remaining: ${s(P)}</p>`:""}
        <p class="muted small">Beacon — ${K(b.BeaconBytes)} of ~${K(b.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${K(b.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(b.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(b.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function _(){if(S)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(S)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!C)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const a=C,d=a.ExecReachable&&!a.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",g=a.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${s(a.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${s(a.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${we(a.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${s(a.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(a.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${we(a.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${s(a.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(a.BeaconHTTP)}">Copy</button>
        </div>
        ${d}
        ${g}
      </div>
    `}function A(a,d){const g=ce[a],f=L[a],P=(I,T,F)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${a}" data-kind="${I}" ${f!==null||F?"disabled":""}>${f===I?G():s(T)}</button>`;return`
      <div class="service-row">
        <span>${s(g)} ${d?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${P("start","Start",d)}
          ${P("stop","Stop",!d)}
          ${P("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${a}" ${f!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function q(a){return`
      <div class="card">
        <h3>Services</h3>
        ${A("exec",a.execActive)}
        ${A("beacon",a.beaconActive)}
        ${w?`<p class="error small">${s(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function G(){return'<span class="spinner" aria-label="working"></span>'}async function z(a,d){if(L[a]===null){L[a]=d,w=null,k();try{await qe(r,a,d)}catch(g){w=`${ce[a]} ${d} failed: ${g instanceof Error?g.message:String(g)}`}L[a]=null,n||k()}}async function oe(a,d){const g=await pe(d),f=a.textContent;a.textContent=g?"Copied!":"Copy failed",setTimeout(()=>{n||(a.textContent=f)},1500)}function ie(a){const d=ce[a],g=b?K(a==="exec"?b.ExecBytes:b.BeaconBytes):"unknown (disk usage hasn't loaded)";o(`
        <h2>Clear ${s(d)} data</h2>
        <p class="error">
          This stops the ${s(d.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(g)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(a)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,I=>{if(I==="cancel"){m();return}I==="confirm"&&i(a)});const f=document.getElementById("clear-confirm-input"),P=document.getElementById("clear-confirm-btn");f==null||f.addEventListener("input",()=>{P&&(P.disabled=f.value.trim()!==a)}),f==null||f.focus()}async function i(a){const d=document.getElementById("clear-confirm-btn");d&&(d.disabled=!0,d.textContent="Clearing…");try{await Fe(r,a),m(),c()}catch(g){const f=document.querySelector("#clear-modal .modal");if(f){const P=document.createElement("p");P.className="error small",P.textContent=`Clear failed: ${g instanceof Error?g.message:String(g)}`,f.appendChild(P)}d&&(d.disabled=!1,d.textContent="Clear and resync")}}function o(a,d){m();const g=document.createElement("div");g.className="modal-overlay",g.id="clear-modal",g.innerHTML=`<div class="modal">${a}</div>`,g.addEventListener("click",f=>{const P=f.target.closest("[data-modal-action]");P!=null&&P.dataset.modalAction&&d(P.dataset.modalAction),f.target===g&&d("cancel")}),document.body.appendChild(g)}function m(){var a;(a=document.getElementById("clear-modal"))==null||a.remove()}return()=>{n=!0,e==null||e(),m()}}const xe=500,Pe="valve-node.explain-consent";function Qe(t,r){let n=!1,e=null;const p=[];t.innerHTML=`
    <h1>Logs: ${s(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${D()}</div>
  `;const h=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");X(t,l=>{l==="explain"&&S()}),b();async function b(){let l,c;try{const[$,k]=await Promise.all([V(),Y()]);l=$.find(R=>R.id===r),c=k}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${s(String($))}</p>`;return}if(n)return;if(!l){h.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const u=c==null?void 0:c.networks.find($=>$.ChainID===l.wire.ChainID);u&&(y.innerHTML=D(u.Name,u.LearnURL));try{const $=await Ne(r,200);if(n)return;p.push(...$)}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load logs: ${s(String($))}</p>`;return}E(),e=Me(r,$=>{n||(p.push($),p.length>xe&&p.splice(0,p.length-xe),E())})}function E(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical");h.innerHTML=`
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
    `}async function S(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical").map(u=>u.line).slice(-40);if(!(localStorage.getItem(Pe)==="1")){L(l);return}await w(l)}function L(l){const c=l.length?`<pre class="explain-excerpt">${l.map(u=>s(u)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';x(`
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
    `,u=>{u==="proceed"?(localStorage.setItem(Pe,"1"),v(),w(l)):v()})}async function w(l){x('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const c=l.length?await ye(r,l):await ye(r);if(n)return;x(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(c.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${c.sentExcerpt.map(u=>s(u)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&v()})}catch(c){if(n)return;if(c instanceof ue&&c.status===409){x(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,u=>{u==="close"&&v()});return}x(`
        <h2>Explain failed</h2>
        <p class="error">${s(c instanceof Error?c.message:String(c))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&v()})}}function x(l,c){v();const u=document.createElement("div");u.className="modal-overlay",u.id="explain-modal",u.innerHTML=`<div class="modal">${l}</div>`,u.addEventListener("click",$=>{const k=$.target.closest("[data-modal-action]");k!=null&&k.dataset.modalAction&&c(k.dataset.modalAction),$.target===u&&c("cancel")}),document.body.appendChild(u)}function v(){var l;(l=document.getElementById("explain-modal"))==null||l.remove()}return()=>{n=!0,e==null||e(),v()}}function et(t,r){let n=!1,e=null,p=null,h=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${s(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${D()}</div>`;const b=t.querySelector("#diag-body"),E=t.querySelector("#diag-footer");X(t,(c,u)=>{var $;if(c==="run")S();else if(c==="toggle")($=u.closest(".check-item"))==null||$.classList.toggle("expanded");else if(c==="copy"){const k=u.dataset.copy;k&&l(u,k)}}),C();async function C(){let c,u;try{const[k,R]=await Promise.all([V(),Y()]);c=k.find(U=>U.id===r),u=R}catch(k){if(n)return;b.innerHTML=`<p class="error">Failed to load target: ${s(String(k))}</p>`;return}if(n)return;if(!c){b.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const $=u==null?void 0:u.networks.find(k=>k.ChainID===c.wire.ChainID);$&&(E.innerHTML=D($.Name,$.LearnURL));try{e=await We(r),y=!0}catch(k){p=String(k instanceof Error?k.message:k)}n||L()}async function S(){h=!0,p=null,L();try{e=await _e(r),y=!0}catch(c){p=String(c instanceof Error?c.message:c)}h=!1,n||L()}function L(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
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
    `}function w(){if(!y&&!p)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const c=new Date(e.at).toLocaleString(),u=e.failedId?`<p><strong>Failed at: ${s(x(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(c)} — trigger: ${s(e.trigger)}</p>
      ${u}
      <ul class="check-list">${e.items.map(v).join("")}</ul>
    `}function x(c){var u;return((u=e==null?void 0:e.items.find($=>$.ID===c))==null?void 0:u.Title)??c}function v(c){const u=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral",$=c.ID===(e==null?void 0:e.failedId);return`
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
    `}async function l(c,u){const $=await pe(u),k=c.textContent;c.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=k)},1500)}return()=>{n=!0}}function tt(t,r){let n=!1,e=[],p=null,h=!1,y=!1;t.innerHTML=`<h1>Security: ${s(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${D()}</div>`;const b=t.querySelector("#sec-body"),E=t.querySelector("#sec-footer");X(t,(v,l)=>{var c;if(v==="rerun")S();else if(v==="toggle")(c=l.closest(".check-item"))==null||c.classList.toggle("expanded");else if(v==="copy"){const u=l.dataset.copy;u&&x(l,u)}}),C();async function C(){let v,l;try{const[u,$]=await Promise.all([V(),Y()]);v=u.find(k=>k.id===r),l=$}catch(u){if(n)return;b.innerHTML=`<p class="error">Failed to load target: ${s(String(u))}</p>`;return}if(n)return;if(!v){b.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!v.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const c=l==null?void 0:l.networks.find(u=>u.ChainID===v.wire.ChainID);c&&(E.innerHTML=D(c.Name,c.LearnURL)),await S()}async function S(){h=!0,p=null,L();try{e=await ze(r),y=!0}catch(v){p=String(v instanceof Error?v.message:v)}h=!1,n||L()}function L(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${h?"disabled":""}>${h?"Re-running…":"Re-run checks"}</button>
      </div>
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${!y&&h?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function w(v){const l=v.Status==="pass"?"ok":v.Status==="fail"?"bad":v.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(v.Status,l)}
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
    `}async function x(v,l){const c=await pe(l),u=v.textContent;v.textContent=c?"Copied!":"Copy failed",setTimeout(()=>{n||(v.textContent=u)},1500)}return()=>{n=!0}}const nt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function rt(t){let r=!1,n=!1,e=!1,p=null,h=!1,y=null,b=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${D()}`;const E=t.querySelector("#settings-body");X(t,w=>{if(w==="save"&&L(),w==="clear-key"){if(!y)return;n=!0;const x=t.querySelector("#ai-key");x&&(x.value=""),S(y)}}),Ee(t,(w,x)=>{w!=="ai-provider"||!y||(b=x,h=!1,S(y))}),C();async function C(){try{const w=await Je();if(r)return;y=w,S(w)}catch(w){if(r)return;E.innerHTML=`<p class="error">Failed to load settings: ${s(String(w))}</p>`}}function S(w){var l;const x=b??w.aiProvider;E.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${de("ai-provider",nt.map(c=>({value:c.value,label:c.label})),x)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${w.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${w.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
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
    `;const v=t.querySelector("#ai-key");v==null||v.addEventListener("input",()=>{n=!0,h=!1}),(l=t.querySelector("#ref-rpc-base"))==null||l.addEventListener("input",()=>{h=!1})}async function L(){const w=t.querySelector("#ai-key"),x=t.querySelector("#ref-rpc-base");if(!w||!x||!y)return;const v={aiProvider:b??y.aiProvider,refRpcBase:x.value.trim()};n&&(v.aiKey=w.value),e=!0,p=null,h=!1,S(y);try{const l=await Ke(v);if(r)return;y=l,n=!1,e=!1,h=!0,S(l)}catch(l){if(r)return;e=!1,p=String(l instanceof Error?l.message:l),S(y)}}return()=>{r=!0}}const at="local";function st(t){let r=!1,n=!1,e="",p=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${D()}
  `;const h=t.querySelector("#targets-body");X(t,(l,c)=>{C(l,c)}),y();async function y(){try{const[l,c,u]=await Promise.all([V(),Y(),Be()]);if(r)return;e=u.os,E(l,c)}catch(l){if(r)return;h.innerHTML=`<p class="error">Failed to load machines: ${s(String(l))}</p>`}}function b(){p&&E(p.targets,p.catalog)}function E(l,c){p={targets:l,catalog:c};const u=e==="linux",$=[...l].sort((N,j)=>(N.mode==="local"?-1:0)-(j.mode==="local"?-1:0)),k=$.length?`<div class="card-grid">${$.map(N=>ot(N,c)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${u?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node is running here as your <strong>controller</strong> — add a Linux server over SSH to run a node, or add this machine to walk the setup (it will need a Linux host to finish)."}
          </p>
        </div>
      `,U=`
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
          ${U}
        </div>
        ${!u&&e?`<p class="muted small">This machine (${s(e)}) runs valve-node as a <strong>controller</strong>. Node hosts must be Linux — "Add this machine" is available to walk the flow, but setup only completes on a Linux host.</p>`:""}
        ${n?it():""}
        ${k}
      </section>
    `}async function C(l,c){var u;if(l==="add-local"){await S();return}if(l==="delete-target"){const $=c.dataset.id;if(!$||!confirm(`Remove target "${$}"? This does not touch anything already running on it.`))return;await L($);return}if(l==="toggle-ssh"){n=!n,v(),b(),n&&((u=t.querySelector("#ssh-host"))==null||u.focus());return}l==="add-ssh"&&await w()}async function S(){v();try{await ve({id:at,mode:"local"}),await y()}catch(l){x(l)}}async function L(l){try{await He(l),await y()}catch(c){x(c)}}async function w(){const l=t.querySelector("#ssh-host"),c=t.querySelector("#ssh-user"),u=t.querySelector("#ssh-key"),$=t.querySelector("#ssh-port"),k=t.querySelector("#ssh-id");if(!l||!c||!u||!$||!k)return;const R=l.value.trim(),U=c.value.trim(),N=u.value.trim(),j=$.value.trim(),Z=k.value.trim();if(v(),!R||!U||!N){x(new Error("host, user, and key path are required"));return}const Q=Z||ct(R),_={Host:R,User:U,KeyPath:N};if(j){const q=Number.parseInt(j,10);if(!Number.isFinite(q)||q<=0){x(new Error("port must be a positive number"));return}_.Port=q}const A=t.querySelector("#ssh-submit");A&&(A.disabled=!0,A.textContent="Connecting…");try{await ve({id:Q,mode:"ssh",ssh:_}),n=!1,await y()}catch(q){x(q),A&&(A.disabled=!1,A.textContent="Add server")}}function x(l){let c=t.querySelector("#targets-error");c||(h.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),c=t.querySelector("#targets-error")),c.textContent=String(l instanceof Error?l.message:l)}function v(){var l;(l=t.querySelector("#targets-error"))==null||l.remove()}return()=>{r=!0}}function ot(t,r){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",p=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let h,y;if(!n)h=B("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const b=r.networks.find(C=>C.ChainID===n.ChainID),E=b?b.Name:`chain ${n.ChainID}`;h=`${B(E,"ok")} ${B(n.ExecID,"neutral")} ${B(n.BeaconID,"neutral")}${n.Archive?" "+B("archive","warn"):""}`,y=`
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
  `}function ct(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const le=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],re=8545,ae=5052,se=30303,lt=[369,943,1],Te={369:"default",943:"practise here first"};function dt(t,r){let n=!1;const e={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${D()}</div>`;const p=t.querySelector("#wizard-body"),h=t.querySelector("#wizard-footer");X(t,(i,o)=>{Q(i,o)}),Ee(t,(i,o)=>{i==="exec-select"?e.execId=o:i==="beacon-select"&&(e.beaconId=o),b()}),t.addEventListener("change",i=>{const o=i.target;o instanceof HTMLInputElement&&(o.id==="data-dir-input"?(_(),u()):o.id==="checkpoint-toggle"&&(e.checkpoint=o.checked,b()))}),y();async function y(){try{const[i,o]=await Promise.all([Y(),V()]);if(n)return;e.catalog=i;const m=o.find(a=>a.id===r);m!=null&&m.wire&&(e.chainId=m.wire.ChainID,e.execId=m.wire.ExecID,e.beaconId=m.wire.BeaconID,e.archive=m.wire.Archive,m.wire.ExecHTTPPort&&(e.execHTTPPort=String(m.wire.ExecHTTPPort)),m.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(m.wire.BeaconHTTPPort)),m.wire.ExecP2PPort&&(e.execP2PPort=String(m.wire.ExecP2PPort)),m.wire.RPCBindAddr&&(e.rpcBindAddr=m.wire.RPCBindAddr)),b()}catch(i){if(n)return;e.loadError=String(i instanceof Error?i.message:i),b()}}function b(){if(e.loadError){p.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(p.innerHTML=`
      ${ie(e.step)}
      ${C()}
    `,E())}function E(){var o;const i=(o=e.catalog)==null?void 0:o.networks.find(m=>m.ChainID===e.chainId);h.innerHTML=i?D(i.Name,i.LearnURL):D()}function C(){switch(e.step){case"network":return S();case"clients":return L();case"mode":return N();case"review":return j();case"run":return Z()}}function S(){const i=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${lt.map(m=>{const a=i.networks.find(f=>f.ChainID===m);if(!a)return"";const d=e.chainId===m,g=Te[m]?B(Te[m],m===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${d?"selected":""}" data-action="pick-network" data-chain-id="${m}" type="button">
          <h3>${s(a.Name)} <span class="muted">(chain ${m})</span></h3>
          ${g}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function L(){const i=e.catalog,o=i.networks.find(d=>d.ChainID===e.chainId);if(!o)return'<p class="error">Unknown network.</p>';(e.execId===null||!o.ExecClients.includes(e.execId))&&(e.execId=o.ExecClients[0]??null),(e.beaconId===null||!o.BeaconClients.includes(e.beaconId))&&(e.beaconId=o.BeaconClients[0]??null);const m=o.ExecClients.map(d=>k(d,i)),a=o.BeaconClients.map(d=>k(d,i));return`
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
        ${U(e.execId,i)}
        <label>
          Beacon client
          ${de("beacon-select",a,e.beaconId)}
        </label>
        ${U(e.beaconId,i)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(i){return i<=0?"—":i>=1?`~${i.toFixed(1)} TB`:`~${Math.round(i*1e3)} GB`}const x=1.1;function v(i){const o=i.ArchiveSizeTB*1e12*x;return{archive:o,full:o/2}}function l(i,o){if(!i)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${s(o)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${s(o)}</code>: ${s(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==o)return"";const m=v(i),a=e.freeBytes>=m.archive,d=e.freeBytes>=m.full,g=`<p class="muted small">Free at <code>${s(o)}</code>: <strong>${K(e.freeBytes)}</strong> — archive ${a?"fits":"won't fit"} (~${w(i.ArchiveSizeTB)}), full ${d?"fits":"won't fit"} (~${w(i.ArchiveSizeTB/2)}).</p>`;let f="";return e.downgradeNote?f=`<p class="banner banner-warn">${s(e.downgradeNote)}</p>`:d||(f=`<p class="banner banner-warn">Neither mode fits at this location (full needs ~${w(i.ArchiveSizeTB/2)}). Choose a location with more space.</p>`),g+f}function c(i,o){if(e.downgradeNote=null,!i||e.freeBytes===null)return;const m=v(i);e.archive&&e.freeBytes<m.archive&&e.freeBytes>=m.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${o} for archive (~${w(i.ArchiveSizeTB)}) — switched to Full (~${w(i.ArchiveSizeTB/2)}). Pick a location with more room to run archive.`)}async function u(){var m;if(e.chainId===null)return;const i=(m=e.catalog)==null?void 0:m.networks.find(a=>a.ChainID===e.chainId),o=(e.dataDir||`/var/lib/valve-node/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,b();try{const{freeBytes:a}=await Re(e.targetId,o);if(n)return;e.freeBytes=a,e.probedPath=o,c(i,o)}catch(a){if(n)return;e.freeBytes=null,e.probedPath=o,e.diskError=String(a instanceof Error?a.message:a)}e.diskProbing=!1,b()}function $(i){return i?/^https?:\/\/.+/i.test(i)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function k(i,o){const m=o.clients.find(a=>a.id===i);return{value:i,label:m?`${m.id} — ${R(m.repo)}`:i}}function R(i){const o=i.split("/");return o.length>=4?o[3]:i}function U(i,o){const m=i?o.clients.find(d=>d.id===i):void 0;if(!m)return"";const a=m.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(m.repo)}" target="_blank" rel="noopener noreferrer">${s(a)}</a></p>`}function N(){var P;const i=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"",o=(P=e.catalog)==null?void 0:P.networks.find(I=>I.ChainID===e.chainId),m=(o==null?void 0:o.ArchiveSizeTB)??0,a=o?w(m/2):"Smaller",d=o?w(m):"Much larger",g=o?` on ${s(o.Name)}`:"",f=o?e.checkpoint?o.SyncLabel:o.GenesisSyncLabel:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <p class="muted">
          Both modes run a fully-validating node — same security, same current-state RPC.
          The difference is how much <strong>historical</strong> state is kept.
        </p>

        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="checkpoint-toggle" ${e.checkpoint?"checked":""} />
            <span><strong>Checkpoint sync</strong> — start near the chain head in minutes (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.</span>
          </label>
          ${o?`<p class="sync-estimate">⏱ Estimated initial sync${g}: <strong>${s(f)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${s((o==null?void 0:o.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${s((o==null?void 0:o.CheckpointURL)??"")}" value="${s(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${s(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        <table class="compare-table">
          <thead>
            <tr><th>What you get</th><th>Full</th><th>Archive</th></tr>
          </thead>
          <tbody>
            <tr><th>Current state &amp; recent blocks</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Send transactions, normal RPC</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Historical state (balances, <code>eth_call</code>) at any past block</th><td class="limited">Recent only (~128 blocks)</td><td class="yes">Full history</td></tr>
            <tr><th>Tracing / <code>debug_trace</code> on old blocks</th><td class="limited">Recent only</td><td class="yes">Full history</td></tr>
            <tr><th>Approx. disk footprint${g}</th><td class="yes">${a}</td><td class="limited">${d}</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <p class="muted small">Disk sizes are rough baselines — they vary by client and setup.</p>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${d}${o?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${a}${o?"":" disk"}</span>
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
            Execution HTTP port <span class="muted">(default: ${re})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${re}" value="${s(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${s(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${ae})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ae}" value="${s(e.beaconHTTPPort)}" />
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
    `}function j(){const o=e.catalog.networks.find(O=>O.ChainID===e.chainId),m=e.dataDir||`/var/lib/valve-node/${e.chainId}`,a=e.jwtPath||`${m}/jwt.hex`,d=le.map(O=>`<li>${s(O.title)}</li>`).join(""),g=z(e.execHTTPPort,re),f=z(e.beaconHTTPPort,ae),P=z(e.execP2PPort,se),I=g||f||P?`<tr><th>Non-default ports</th><td>${[g?`exec HTTP ${g}`:null,f?`beacon HTTP ${f}`:null,P?`exec p2p ${P}`:null].filter(O=>O!==null).map(s).join(", ")}</td></tr>`:"",{addr:T}=A(e.rpcBindAddr),F=T?`<tr><th>RPC bind address</th><td><code>${s(T)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
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
            <tr><th>JWT secret path</th><td><code>${s(a)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${s(e.checkpointUrl||(o==null?void 0:o.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${I}
            ${F}
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
    `}function Z(){const o=e.catalog.networks.find(T=>T.ChainID===e.chainId),m=o==null?void 0:o.LearnURL,a=new Set(e.events.filter(T=>T.done).map(T=>T.stepId)),d=new Set(e.events.filter(T=>T.err).map(T=>T.stepId)),g=new Map;for(const T of e.events){if(!T.line)continue;const F=g.get(T.stepId)??[];F.push(T.line),g.set(T.stepId,F)}const f=le.map(T=>{var ge;const F=a.has(T.id),O=d.has(T.id),he=O?B("failed","bad"):F?B("done","ok"):B("pending","neutral"),fe=(g.get(T.id)??[]).slice(-5),me=(ge=e.events.find(ne=>ne.stepId===T.id&&ne.err))==null?void 0:ge.err,Ce=T.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${m?` <a href="${s(m)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${F?"step-done":""} ${O?"step-error":""}">
          <div class="step-head">${he} <strong>${s(T.title)}</strong></div>
          ${Ce}
          ${fe.length?`<pre class="step-log">${fe.map(ne=>s(ne)).join(`
`)}</pre>`:""}
          ${me?`<p class="error small">${s(me)}</p>`:""}
        </li>
      `}).join(""),P=e.events.some(T=>T.err),I=le.every(T=>a.has(T.id))||e.events.some(T=>T.stepId==="handshake"&&T.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${f}</ol>
        ${I&&!P?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${P?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function Q(i,o){switch(i){case"pick-network":e.chainId=Number(o.dataset.chainId),e.execId=null,e.beaconId=null,b();break;case"goto-network":e.step="network",b();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",b();break;case"goto-mode":e.step="mode",b(),u();break;case"goto-review":if(_(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError){b();break}e.step="review",b();break;case"start-setup":oe();break}}function _(){const i=t.querySelectorAll('input[name="mode"]');for(const I of Array.from(i))I.checked&&(e.archive=I.value==="archive");const o=t.querySelector("#data-dir-input"),m=t.querySelector("#jwt-path-input");o&&(e.dataDir=o.value.trim()),m&&(e.jwtPath=m.value.trim());const a=t.querySelector("#exec-http-port-input"),d=t.querySelector("#beacon-http-port-input"),g=t.querySelector("#exec-p2p-port-input");a&&(e.execHTTPPort=a.value.trim()),d&&(e.beaconHTTPPort=d.value.trim()),g&&(e.execP2PPort=g.value.trim());const f=t.querySelector("#rpc-bind-addr-input");f&&(e.rpcBindAddr=f.value.trim());const P=t.querySelector("#checkpoint-url-input");P&&(e.checkpointUrl=P.value.trim()),e.execHTTPPortError=G(e.execHTTPPort).error??null,e.beaconHTTPPortError=G(e.beaconHTTPPort).error??null,e.execP2PPortError=G(e.execP2PPort).error??null,e.rpcBindAddrError=A(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?$(e.checkpointUrl):null}function A(i){if(!i)return{};const o=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(i);return o?o.slice(1).every(m=>Number(m)<=255)?{addr:i}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(i)&&i.includes(":")?{addr:i}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const q=/^\d+$/;function G(i){if(!i)return{};if(!q.test(i))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const o=Number(i);return!Number.isInteger(o)||o<1||o>65535?{error:"Port must be between 1 and 65535."}:{port:o}}function z(i,o){const{port:m}=G(i);if(!(m===void 0||m===o))return m}async function oe(){var g;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,b();const i={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(i.DataDir=e.dataDir),e.jwtPath&&(i.JWTPath=e.jwtPath);const o=z(e.execHTTPPort,re),m=z(e.beaconHTTPPort,ae),a=z(e.execP2PPort,se);o!==void 0&&(i.ExecHTTPPort=o),m!==void 0&&(i.BeaconHTTPPort=m),a!==void 0&&(i.ExecP2PPort=a);const{addr:d}=A(e.rpcBindAddr);d!==void 0&&(i.RPCBindAddr=d),e.checkpoint?e.checkpointUrl&&(i.CheckpointURL=e.checkpointUrl):i.NoCheckpoint=!0;try{await Ae(e.targetId,i)}catch(f){if(!(f instanceof ue&&f.status===409)){e.starting=!1,e.startError=String(f instanceof Error?f.message:f),b();return}}e.starting=!1,e.step="run",e.events=[],b(),(g=e.streamStop)==null||g.call(e),e.streamStop=De(e.targetId,f=>{n||(e.events.push(f),e.step==="run"&&b())})}function ie(i){const o=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],a=o.map(d=>d.id).indexOf(i);return`
      <ol class="wizard-progress">
        ${o.map((d,g)=>`<li class="${g===a?"current":g<a?"past":"future"}">${s(d.label)}</li>`).join("")}
      </ol>
    `}return()=>{var i;n=!0,(i=e.streamStop)==null||i.call(e)}}const ut=document.querySelector("#app"),{contentEl:pt,setActiveNav:ht}=Ge(ut);let M=null;function ft(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"targets"};const[n,e]=r;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function J(t){const r=document.createElement("div");return pt.replaceChildren(r),t(r)}function Se(){if(M){try{M()}catch{}M=null}const{screen:t,id:r}=ft();switch(ht(t),t){case"setup":if(!r){location.hash="#/targets";return}M=J(n=>dt(n,r));break;case"dash":if(!r){location.hash="#/targets";return}M=J(n=>Ze(n,r));break;case"logs":if(!r){location.hash="#/targets";return}M=J(n=>Qe(n,r));break;case"security":if(!r){location.hash="#/targets";return}M=J(n=>tt(n,r));break;case"diag":if(!r){location.hash="#/targets";return}M=J(n=>et(n,r));break;case"settings":M=J(n=>rt(n));break;case"targets":default:M=J(n=>st(n));break}}window.addEventListener("hashchange",Se);Se();
