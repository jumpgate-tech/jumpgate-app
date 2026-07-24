var Le=Object.defineProperty;var Ie=(t,r,n)=>r in t?Le(t,r,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[r]=n;var ve=(t,r,n)=>Ie(t,typeof r!="symbol"?r+"":r,n);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const p of document.querySelectorAll('link[rel="modulepreload"]'))e(p);new MutationObserver(p=>{for(const h of p)if(h.type==="childList")for(const y of h.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(p){const h={};return p.integrity&&(h.integrity=p.integrity),p.referrerPolicy&&(h.referrerPolicy=p.referrerPolicy),p.crossOrigin==="use-credentials"?h.credentials="include":p.crossOrigin==="anonymous"?h.credentials="omit":h.credentials="same-origin",h}function e(p){if(p.ep)return;p.ep=!0;const h=n(p);fetch(p.href,h)}})();function Be(){return H("/api/host")}function Y(){return H("/api/catalog")}function V(){return H("/api/targets")}function be(t){return H("/api/targets",{method:"POST",headers:te,body:JSON.stringify(t)})}function He(t){return H(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Re(t,r){return H(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(r)}`)}function Ae(t,r){return H(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:te,body:JSON.stringify(r)})}function De(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function Ne(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function Ue(t,r=200){return H(`/api/targets/${encodeURIComponent(t)}/logs?n=${r}`)}function Me(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function ye(t,r){const n=r===void 0?{}:{lines:r};return H(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:te,body:JSON.stringify(n)})}function qe(t,r,n){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/${n}`,{method:"POST"})}function Fe(t,r){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/clear`,{method:"POST",headers:te,body:JSON.stringify({Confirm:r})})}function Oe(t){return H(`/api/targets/${encodeURIComponent(t)}/du`)}function ze(t){return H(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function _e(t){return H(`/api/targets/${encodeURIComponent(t)}/firewall`)}function je(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Ke(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function We(){return H("/api/settings")}function Je(t){return H("/api/settings",{method:"PUT",headers:te,body:JSON.stringify(t)})}class ue extends Error{constructor(n,e){super(e);ve(this,"status");this.name="ApiError",this.status=n}}const te={"Content-Type":"application/json"};async function H(t,r){const n=await fetch(t,r);if(!n.ok){let p=n.statusText||`HTTP ${n.status}`;try{const h=await n.json();h&&typeof h.error=="string"&&h.error&&(p=h.error)}catch{}throw new ue(n.status,p)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const $e="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function N(t,r){const n=t&&r&&r!==$e?` <span class="footer-sep">·</span> <a href="${s(r)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
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
  `;const r=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:p=>{for(const h of n)h.classList.toggle("active",h.dataset.nav===p)}}}function K(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ye(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ve(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const r=Math.round(t/60),n=Math.floor(r/60),e=r%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const p=Math.floor(n/24),h=n%24;return`~${p}d ${h}h`}function E(t,r){return`<span class="badge badge-${r}">${s(t)}</span>`}function we(t){return`<span class="dot dot-${t}"></span>`}const ke=["B","KB","MB","GB","TB","PB"];function J(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let r=t,n=0;for(;r>=1024&&n<ke.length-1;)r/=1024,n++;const e=r<10?2:r<100?1:0;return`${r.toFixed(e)} ${ke[n]}`}async function pe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function X(t,r){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const p=e.dataset.action;p&&r(p,e,n)})}function de(t,r,n){const e=r.find(h=>h.value===n),p=r.map(h=>`
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
  `}function ee(t){t.querySelectorAll(".dropdown.open").forEach(r=>{var n;r.classList.remove("open"),(n=r.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function Se(t,r){t.addEventListener("click",p=>{const h=p.target,y=h.closest(".dropdown-trigger");if(y&&t.contains(y)){const S=y.closest(".dropdown"),L=!!S&&!S.classList.contains("open");ee(t),S&&L&&(S.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const v=h.closest(".dropdown-option");if(v&&t.contains(v)){const S=v.closest(".dropdown");ee(t),r((S==null?void 0:S.dataset.dropdown)??"",v.dataset.value??"");return}ee(t)});const n=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const h=p.target;(!h.closest(".dropdown")||!t.contains(h))&&ee(t)},e=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}p.key==="Escape"&&ee(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const Xe=85,ce={exec:"Execution",beacon:"Beacon"};function Ze(t,r){let n=!1,e=null,p=null,h=null,y=null,v=null,S=null,L=null,C=null;const B={exec:null,beacon:null};let k=null;t.innerHTML=`<h1>Dashboard: ${s(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${N()}</div>`;const T=t.querySelector("#dash-body"),b=t.querySelector("#dash-footer");T.addEventListener("click",a=>{const d=a.target.closest("[data-action]");if(!d||!T.contains(d))return;const f=d.dataset.action;if(f==="svc-action"){const m=d.dataset.svc,P=d.dataset.kind;m&&P&&_(m,P)}else if(f==="open-clear"){const m=d.dataset.svc;m&&ie(m)}else if(f==="copy"){const m=d.dataset.copy;m&&oe(d,m)}else f==="retry-du"?c():f==="retry-endpoints"&&u()}),l();async function l(){let a,d;try{const[m,P]=await Promise.all([V(),Y()]);a=m.find(I=>I.id===r),d=P}catch(m){if(n)return;T.innerHTML=`<p class="error">Failed to load target: ${s(String(m))}</p>`;return}if(n)return;if(!a){T.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!a.wire){T.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const f=d==null?void 0:d.networks.find(m=>m.ChainID===a.wire.ChainID);f&&(b.innerHTML=N(f.Name,f.LearnURL)),T.innerHTML='<p class="muted">Connecting…</p>',e=Ne(r,m=>{n||($(m),p=m,h=m,x())}),c(),u()}async function c(){S=null;try{v=await Oe(r)}catch(a){v=null,S=String(a instanceof Error?a.message:a)}n||x()}async function u(){C=null;try{L=await ze(r)}catch(a){L=null,C=String(a instanceof Error?a.message:a)}n||x()}function $(a){if(!p)return;const d=(new Date(a.at).getTime()-new Date(p.at).getTime())/1e3,f=a.execHead-p.execHead;if(d>0&&f>=0){const m=f/d;y=y===null?m:y*.7+m*.3}}function x(){if(!h)return;const a=h;T.innerHTML=`
      <p class="dash-status">${U(a)}</p>
      <div class="card-grid">
        ${O(a)}
        ${q(a)}
        ${z(a)}
        ${Z(a)}
        ${Q(a)}
        ${j()}
      </div>
      <p class="muted small">Last updated ${s(new Date(a.at).toLocaleTimeString())}</p>
    `}function U(a){return!a.execActive&&!a.beaconActive?E("Node not running","bad"):a.execSyncing||a.beaconDistance>0?E("Syncing","warn"):E("Running · synced","ok")}function M(a){const f=a.refHead>0?a.refHead-a.execHead:null,m=f!==null&&f>0&&y&&y>0?Ve(f/y):f!==null&&f<=0?"caught up":"—";return{lag:f,eta:m}}function q(a){const{lag:d,eta:f}=M(a);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${a.execActive?a.execSyncing?E("syncing","warn"):a.execHead===0?E("no data","neutral"):E("synced","ok"):E("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${K(a.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${d!==null?K(a.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${d!==null?K(Math.max(d,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${f}</dd></div>
        </dl>
      </div>
    `}function z(a){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${a.beaconActive?a.beaconSlot===0?E("no data","neutral"):a.beaconDistance===0?E("synced","ok"):E("syncing","warn"):E("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${K(a.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${K(a.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function Z(a){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${K(a.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${K(a.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Q(a){const d=a.diskUsedPct>=Xe,f=`
      <div class="meter"><div class="meter-fill ${d?"meter-warn":""}" style="width:${Math.min(a.diskUsedPct,100)}%"></div></div>
      <p>${Ye(a.diskUsedPct)} used</p>
    `;if(S)return`
        <div class="card ${d?"card-warn":""}">
          <h3>Storage</h3>
          ${f}
          <p class="error small">${s(S)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!v)return`
        <div class="card ${d?"card-warn":""}">
          <h3>Storage</h3>
          ${f}
          <p class="muted">Loading…</p>
        </div>
      `;const m=v.ExpectedExecBytes>0?Math.min(v.ExecBytes/v.ExpectedExecBytes*100,100):0,P=v.ExpectedBeaconBytes>0?Math.min(v.BeaconBytes/v.ExpectedBeaconBytes*100,100):0,{lag:I,eta:w}=M(a),R=I!==null&&I>0&&y!==null&&y>0;return`
      <div class="card ${d?"card-warn":""}">
        <h3>Storage</h3>
        ${f}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${J(v.ExecBytes)} of ~${J(v.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${m}%"></div></div>
        ${R?`<p class="muted small">Estimated time remaining: ${s(w)}</p>`:""}
        <p class="muted small">Beacon — ${J(v.BeaconBytes)} of ~${J(v.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${P}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${J(v.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(v.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(v.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function j(){if(C)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(C)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!L)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const a=L,d=a.ExecReachable&&!a.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",f=a.Access==="ssh"?`
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
        ${f}
      </div>
    `}function A(a,d){const f=ce[a],m=B[a],P=(I,w,R)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${a}" data-kind="${I}" ${m!==null||R?"disabled":""}>${m===I?G():s(w)}</button>`;return`
      <div class="service-row">
        <span>${s(f)} ${d?E("active","ok"):E("down","bad")}</span>
        <div class="service-actions">
          ${P("start","Start",d)}
          ${P("stop","Stop",!d)}
          ${P("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${a}" ${m!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function O(a){return`
      <div class="card">
        <h3>Services</h3>
        ${A("exec",a.execActive)}
        ${A("beacon",a.beaconActive)}
        ${k?`<p class="error small">${s(k)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function G(){return'<span class="spinner" aria-label="working"></span>'}async function _(a,d){if(B[a]===null){B[a]=d,k=null,x();try{await qe(r,a,d)}catch(f){k=`${ce[a]} ${d} failed: ${f instanceof Error?f.message:String(f)}`}B[a]=null,n||x()}}async function oe(a,d){const f=await pe(d),m=a.textContent;a.textContent=f?"Copied!":"Copy failed",setTimeout(()=>{n||(a.textContent=m)},1500)}function ie(a){const d=ce[a],f=v?J(a==="exec"?v.ExecBytes:v.BeaconBytes):"unknown (disk usage hasn't loaded)";o(`
        <h2>Clear ${s(d)} data</h2>
        <p class="error">
          This stops the ${s(d.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(f)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(a)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,I=>{if(I==="cancel"){g();return}I==="confirm"&&i(a)});const m=document.getElementById("clear-confirm-input"),P=document.getElementById("clear-confirm-btn");m==null||m.addEventListener("input",()=>{P&&(P.disabled=m.value.trim()!==a)}),m==null||m.focus()}async function i(a){const d=document.getElementById("clear-confirm-btn");d&&(d.disabled=!0,d.textContent="Clearing…");try{await Fe(r,a),g(),c()}catch(f){const m=document.querySelector("#clear-modal .modal");if(m){const P=document.createElement("p");P.className="error small",P.textContent=`Clear failed: ${f instanceof Error?f.message:String(f)}`,m.appendChild(P)}d&&(d.disabled=!1,d.textContent="Clear and resync")}}function o(a,d){g();const f=document.createElement("div");f.className="modal-overlay",f.id="clear-modal",f.innerHTML=`<div class="modal">${a}</div>`,f.addEventListener("click",m=>{const P=m.target.closest("[data-modal-action]");P!=null&&P.dataset.modalAction&&d(P.dataset.modalAction),m.target===f&&d("cancel")}),document.body.appendChild(f)}function g(){var a;(a=document.getElementById("clear-modal"))==null||a.remove()}return()=>{n=!0,e==null||e(),g()}}const xe=500,Te="valve-node-app.explain-consent";function Qe(t,r){let n=!1,e=null;const p=[];t.innerHTML=`
    <h1>Logs: ${s(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${N()}</div>
  `;const h=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");X(t,l=>{l==="explain"&&C()}),v();async function v(){let l,c;try{const[$,x]=await Promise.all([V(),Y()]);l=$.find(U=>U.id===r),c=x}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${s(String($))}</p>`;return}if(n)return;if(!l){h.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const u=c==null?void 0:c.networks.find($=>$.ChainID===l.wire.ChainID);u&&(y.innerHTML=N(u.Name,u.LearnURL));try{const $=await Ue(r,200);if(n)return;p.push(...$)}catch($){if(n)return;h.innerHTML=`<p class="error">Failed to load logs: ${s(String($))}</p>`;return}S(),e=Me(r,$=>{n||(p.push($),p.length>xe&&p.splice(0,p.length-xe),S())})}function S(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical");h.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${p.map(L).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${E(String(l.length),l.length?"bad":"neutral")}</h2>
          <div class="log-lines">${l.length?l.slice().reverse().map(L).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const c=h.querySelector(".log-lines");c&&(c.scrollTop=c.scrollHeight)}function L(l){const c=l.severity||"info",u=l.learnUrl?` <a href="${s(l.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(c)}">
        <span class="log-time">${s(new Date(l.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(l.unit)}</span>
        <span class="log-sev">${s(c)}</span>
        <span class="log-text">${s(l.line)}</span>
        ${l.explain?`<div class="log-explain">${s(l.explain)}${u}</div>`:""}
      </div>
    `}async function C(){const l=p.filter(u=>u.severity==="error"||u.severity==="critical").map(u=>u.line).slice(-40);if(!(localStorage.getItem(Te)==="1")){B(l);return}await k(l)}function B(l){const c=l.length?`<pre class="explain-excerpt">${l.map(u=>s(u)).join(`
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
    `,u=>{u==="proceed"?(localStorage.setItem(Te,"1"),b(),k(l)):b()})}async function k(l){T('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const c=l.length?await ye(r,l):await ye(r);if(n)return;T(`
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
      `,u=>{u==="close"&&b()})}}function T(l,c){b();const u=document.createElement("div");u.className="modal-overlay",u.id="explain-modal",u.innerHTML=`<div class="modal">${l}</div>`,u.addEventListener("click",$=>{const x=$.target.closest("[data-modal-action]");x!=null&&x.dataset.modalAction&&c(x.dataset.modalAction),$.target===u&&c("cancel")}),document.body.appendChild(u)}function b(){var l;(l=document.getElementById("explain-modal"))==null||l.remove()}return()=>{n=!0,e==null||e(),b()}}function et(t,r){let n=!1,e=null,p=null,h=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${s(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${N()}</div>`;const v=t.querySelector("#diag-body"),S=t.querySelector("#diag-footer");X(t,(c,u)=>{var $;if(c==="run")C();else if(c==="toggle")($=u.closest(".check-item"))==null||$.classList.toggle("expanded");else if(c==="copy"){const x=u.dataset.copy;x&&l(u,x)}}),L();async function L(){let c,u;try{const[x,U]=await Promise.all([V(),Y()]);c=x.find(M=>M.id===r),u=U}catch(x){if(n)return;v.innerHTML=`<p class="error">Failed to load target: ${s(String(x))}</p>`;return}if(n)return;if(!c){v.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const $=u==null?void 0:u.networks.find(x=>x.ChainID===c.wire.ChainID);$&&(S.innerHTML=N($.Name,$.LearnURL));try{e=await Ke(r),y=!0}catch(x){p=String(x instanceof Error?x.message:x)}n||B()}async function C(){h=!0,p=null,B();try{e=await je(r),y=!0}catch(c){p=String(c instanceof Error?c.message:c)}h=!1,n||B()}function B(){v.innerHTML=`
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
      ${k()}
    `}function k(){if(!y&&!p)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const c=new Date(e.at).toLocaleString(),u=e.failedId?`<p><strong>Failed at: ${s(T(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(c)} — trigger: ${s(e.trigger)}</p>
      ${u}
      <ul class="check-list">${e.items.map(b).join("")}</ul>
    `}function T(c){var u;return((u=e==null?void 0:e.items.find($=>$.ID===c))==null?void 0:u.Title)??c}function b(c){const u=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral",$=c.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${$?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${E($?"failed here":c.Status,u)}
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
    `}async function l(c,u){const $=await pe(u),x=c.textContent;c.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=x)},1500)}return()=>{n=!0}}function tt(t,r){let n=!1,e=[],p=null,h=!1,y=!1;t.innerHTML=`<h1>Security: ${s(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${N()}</div>`;const v=t.querySelector("#sec-body"),S=t.querySelector("#sec-footer");X(t,(b,l)=>{var c;if(b==="rerun")C();else if(b==="toggle")(c=l.closest(".check-item"))==null||c.classList.toggle("expanded");else if(b==="copy"){const u=l.dataset.copy;u&&T(l,u)}}),L();async function L(){let b,l;try{const[u,$]=await Promise.all([V(),Y()]);b=u.find(x=>x.id===r),l=$}catch(u){if(n)return;v.innerHTML=`<p class="error">Failed to load target: ${s(String(u))}</p>`;return}if(n)return;if(!b){v.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!b.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const c=l==null?void 0:l.networks.find(u=>u.ChainID===b.wire.ChainID);c&&(S.innerHTML=N(c.Name,c.LearnURL)),await C()}async function C(){h=!0,p=null,B();try{e=await _e(r),y=!0}catch(b){p=String(b instanceof Error?b.message:b)}h=!1,n||B()}function B(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${h?"disabled":""}>${h?"Re-running…":"Re-run checks"}</button>
      </div>
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${!y&&h?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(k).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function k(b){const l=b.Status==="pass"?"ok":b.Status==="fail"?"bad":b.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${E(b.Status,l)}
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
    `}async function T(b,l){const c=await pe(l),u=b.textContent;b.textContent=c?"Copied!":"Copy failed",setTimeout(()=>{n||(b.textContent=u)},1500)}return()=>{n=!0}}const nt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function at(t){let r=!1,n=!1,e=!1,p=null,h=!1,y=null,v=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${N()}`;const S=t.querySelector("#settings-body");X(t,k=>{if(k==="save"&&B(),k==="clear-key"){if(!y)return;n=!0;const T=t.querySelector("#ai-key");T&&(T.value=""),C(y)}}),Se(t,(k,T)=>{k!=="ai-provider"||!y||(v=T,h=!1,C(y))}),L();async function L(){try{const k=await We();if(r)return;y=k,C(k)}catch(k){if(r)return;S.innerHTML=`<p class="error">Failed to load settings: ${s(String(k))}</p>`}}function C(k){var l;const T=v??k.aiProvider;S.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${de("ai-provider",nt.map(c=>({value:c.value,label:c.label})),T)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${k.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${k.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${s(k.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${p?`<p class="error">${s(p)}</p>`:""}
        ${h?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const b=t.querySelector("#ai-key");b==null||b.addEventListener("input",()=>{n=!0,h=!1}),(l=t.querySelector("#ref-rpc-base"))==null||l.addEventListener("input",()=>{h=!1})}async function B(){const k=t.querySelector("#ai-key"),T=t.querySelector("#ref-rpc-base");if(!k||!T||!y)return;const b={aiProvider:v??y.aiProvider,refRpcBase:T.value.trim()};n&&(b.aiKey=k.value),e=!0,p=null,h=!1,C(y);try{const l=await Je(b);if(r)return;y=l,n=!1,e=!1,h=!0,C(l)}catch(l){if(r)return;e=!1,p=String(l instanceof Error?l.message:l),C(y)}}return()=>{r=!0}}const rt="local";function st(t){let r=!1,n=!1,e="",p=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${N()}
  `;const h=t.querySelector("#targets-body");X(t,(l,c)=>{L(l,c)}),y();async function y(){try{const[l,c,u]=await Promise.all([V(),Y(),Be()]);if(r)return;e=u.os,S(l,c)}catch(l){if(r)return;h.innerHTML=`<p class="error">Failed to load machines: ${s(String(l))}</p>`}}function v(){p&&S(p.targets,p.catalog)}function S(l,c){p={targets:l,catalog:c};const u=e==="linux",$=[...l].sort((q,z)=>(q.mode==="local"?-1:0)-(z.mode==="local"?-1:0)),x=$.length?`<div class="card-grid">${$.map(q=>ot(q,c)).join("")}</div>`:`
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
    `}async function L(l,c){var u;if(l==="add-local"){await C();return}if(l==="delete-target"){const $=c.dataset.id;if(!$||!confirm(`Remove target "${$}"? This does not touch anything already running on it.`))return;await B($);return}if(l==="toggle-ssh"){n=!n,b(),v(),n&&((u=t.querySelector("#ssh-host"))==null||u.focus());return}l==="add-ssh"&&await k()}async function C(){if(b(),!(e!=="linux"&&!confirm("This machine ("+e+") isn't a Linux host, so node setup can't complete here — it's only useful for previewing the setup wizard. Add it anyway?")))try{await be({id:rt,mode:"local"}),await y()}catch(l){T(l)}}async function B(l){try{await He(l),await y()}catch(c){T(c)}}async function k(){const l=t.querySelector("#ssh-host"),c=t.querySelector("#ssh-user"),u=t.querySelector("#ssh-key"),$=t.querySelector("#ssh-port"),x=t.querySelector("#ssh-id");if(!l||!c||!u||!$||!x)return;const U=l.value.trim(),M=c.value.trim(),q=u.value.trim(),z=$.value.trim(),Z=x.value.trim();if(b(),!U||!M||!q){T(new Error("host, user, and key path are required"));return}const Q=Z||ct(U),j={Host:U,User:M,KeyPath:q};if(z){const O=Number.parseInt(z,10);if(!Number.isFinite(O)||O<=0){T(new Error("port must be a positive number"));return}j.Port=O}const A=t.querySelector("#ssh-submit");A&&(A.disabled=!0,A.textContent="Connecting…");try{await be({id:Q,mode:"ssh",ssh:j}),n=!1,await y()}catch(O){T(O),A&&(A.disabled=!1,A.textContent="Add server")}}function T(l){let c=t.querySelector("#targets-error");c||(h.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),c=t.querySelector("#targets-error")),c.textContent=String(l instanceof Error?l.message:l)}function b(){var l;(l=t.querySelector("#targets-error"))==null||l.remove()}return()=>{r=!0}}function ot(t,r){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",p=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let h,y;if(!n)h=E("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const v=r.networks.find(L=>L.ChainID===n.ChainID),S=v?v.Name:`chain ${n.ChainID}`;h=`${E(S,"ok")} ${E(n.ExecID,"neutral")} ${E(n.BeaconID,"neutral")}${n.Archive?" "+E("archive","warn"):""}`,y=`
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
  `}function ct(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const le=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ae=8545,re=5052,se=30303,lt=[369,943,1],Pe={369:"default",943:"practise here first"};function dt(t,r){let n=!1;const e={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${N()}</div>`;const p=t.querySelector("#wizard-body"),h=t.querySelector("#wizard-footer");X(t,(i,o)=>{Q(i,o)}),Se(t,(i,o)=>{i==="exec-select"?e.execId=o:i==="beacon-select"&&(e.beaconId=o),v()}),t.addEventListener("change",i=>{const o=i.target;o instanceof HTMLInputElement&&(o.id==="data-dir-input"?(j(),u()):o.id==="checkpoint-toggle"?(e.checkpoint=o.checked,v()):o.id==="exec-snapshot-toggle"&&(e.execSnapshot=o.checked,v()))}),y();async function y(){try{const[i,o]=await Promise.all([Y(),V()]);if(n)return;e.catalog=i;const g=o.find(a=>a.id===r);g!=null&&g.wire&&(e.chainId=g.wire.ChainID,e.execId=g.wire.ExecID,e.beaconId=g.wire.BeaconID,e.archive=g.wire.Archive,g.wire.ExecHTTPPort&&(e.execHTTPPort=String(g.wire.ExecHTTPPort)),g.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(g.wire.BeaconHTTPPort)),g.wire.ExecP2PPort&&(e.execP2PPort=String(g.wire.ExecP2PPort)),g.wire.RPCBindAddr&&(e.rpcBindAddr=g.wire.RPCBindAddr)),v()}catch(i){if(n)return;e.loadError=String(i instanceof Error?i.message:i),v()}}function v(){if(e.loadError){p.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(p.innerHTML=`
      ${ie(e.step)}
      ${L()}
    `,S())}function S(){var o;const i=(o=e.catalog)==null?void 0:o.networks.find(g=>g.ChainID===e.chainId);h.innerHTML=i?N(i.Name,i.LearnURL):N()}function L(){switch(e.step){case"network":return C();case"clients":return B();case"mode":return q();case"review":return z();case"run":return Z()}}function C(){const i=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${lt.map(g=>{const a=i.networks.find(m=>m.ChainID===g);if(!a)return"";const d=e.chainId===g,f=Pe[g]?E(Pe[g],g===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${d?"selected":""}" data-action="pick-network" data-chain-id="${g}" type="button">
          <h3>${s(a.Name)} <span class="muted">(chain ${g})</span></h3>
          ${f}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function B(){const i=e.catalog,o=i.networks.find(d=>d.ChainID===e.chainId);if(!o)return'<p class="error">Unknown network.</p>';(e.execId===null||!o.ExecClients.includes(e.execId))&&(e.execId=o.ExecClients[0]??null),(e.beaconId===null||!o.BeaconClients.includes(e.beaconId))&&(e.beaconId=o.BeaconClients[0]??null);const g=o.ExecClients.map(d=>x(d,i)),a=o.BeaconClients.map(d=>x(d,i));return`
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
          ${de("exec-select",g,e.execId)}
        </label>
        ${M(e.execId,i)}
        <label>
          Beacon client
          ${de("beacon-select",a,e.beaconId)}
        </label>
        ${M(e.beaconId,i)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function k(i){return i<=0?"—":i>=1?`~${i.toFixed(1)} TB`:`~${Math.round(i*1e3)} GB`}const T=1.1;function b(i){const o=i.ArchiveSizeTB*1e12*T;return{archive:o,full:o/2}}function l(i,o){if(!i)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${s(o)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${s(o)}</code>: ${s(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==o)return"";const g=b(i),a=e.freeBytes>=g.archive,d=e.freeBytes>=g.full,f=`<p class="muted small">Free at <code>${s(o)}</code>: <strong>${J(e.freeBytes)}</strong> — archive ${a?"fits":"won't fit"} (${k(i.ArchiveSizeTB)}), full ${d?"fits":"won't fit"} (${k(i.ArchiveSizeTB/2)}).</p>`;let m="";return e.downgradeNote?m=`<p class="banner banner-warn">${s(e.downgradeNote)}</p>`:d||(m=`<p class="banner banner-warn">Neither full (${k(i.ArchiveSizeTB/2)}) nor archive (${k(i.ArchiveSizeTB)}) fits the free space here — choose a location with more room.</p>`),f+m}function c(i,o){if(e.downgradeNote=null,!i||e.freeBytes===null)return;const g=b(i);e.archive&&e.freeBytes<g.archive&&e.freeBytes>=g.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${o} for archive (${k(i.ArchiveSizeTB)}) — switched to Full (${k(i.ArchiveSizeTB/2)}). Pick a location with more room to run archive.`)}async function u(){var g;if(e.chainId===null)return;const i=(g=e.catalog)==null?void 0:g.networks.find(a=>a.ChainID===e.chainId),o=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,v();try{const{freeBytes:a}=await Re(e.targetId,o);if(n)return;e.freeBytes=a,e.probedPath=o,c(i,o)}catch(a){if(n)return;e.freeBytes=null,e.probedPath=o,e.diskError=String(a instanceof Error?a.message:a)}e.diskProbing=!1,v()}function $(i){return i?/^https?:\/\/.+/i.test(i)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function x(i,o){const g=o.clients.find(a=>a.id===i);return{value:i,label:g?`${g.id} — ${U(g.repo)}`:i}}function U(i){const o=i.split("/");return o.length>=4?o[3]:i}function M(i,o){const g=i?o.clients.find(d=>d.id===i):void 0;if(!g)return"";const a=g.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(g.repo)}" target="_blank" rel="noopener noreferrer">${s(a)}</a></p>`}function q(){var I,w,R;const i=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",o=(I=e.catalog)==null?void 0:I.networks.find(D=>D.ChainID===e.chainId),g=((R=(w=e.catalog)==null?void 0:w.clients.find(D=>D.id===e.execId))==null?void 0:R.snapshotSupported)??!1,a=(o==null?void 0:o.ArchiveSizeTB)??0,d=o?k(a/2):"Smaller",f=o?k(a):"Much larger",m=o?` on ${s(o.Name)}`:"",P=o?e.checkpoint?o.SyncLabel:o.GenesisSyncLabel:"";return`
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
          ${o?`<p class="sync-estimate">⏱ Estimated initial sync${m}: <strong>${s(P)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${s((o==null?void 0:o.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${s((o==null?void 0:o.CheckpointURL)??"")}" value="${s(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${s(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${g?`
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
              <tr><th>Approx. disk footprint${m}</th><td class="yes">${d}</td><td class="limited">${f}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          <p class="muted small">Disk sizes are rough baselines — they vary by client and setup.</p>
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${f}${o?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
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
    `}function z(){const o=e.catalog.networks.find(D=>D.ChainID===e.chainId),g=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,a=e.jwtPath||`${g}/jwt.hex`,d=le.map(D=>`<li>${s(D.title)}</li>`).join(""),f=_(e.execHTTPPort,ae),m=_(e.beaconHTTPPort,re),P=_(e.execP2PPort,se),I=f||m||P?`<tr><th>Non-default ports</th><td>${[f?`exec HTTP ${f}`:null,m?`beacon HTTP ${m}`:null,P?`exec p2p ${P}`:null].filter(D=>D!==null).map(s).join(", ")}</td></tr>`:"",{addr:w}=A(e.rpcBindAddr),R=w?`<tr><th>RPC bind address</th><td><code>${s(w)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${s(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${s((o==null?void 0:o.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${s(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${s(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${s(g)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${s(a)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${s(e.checkpointUrl||(o==null?void 0:o.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${I}
            ${R}
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
    `}function Z(){const o=e.catalog.networks.find(w=>w.ChainID===e.chainId),g=o==null?void 0:o.LearnURL,a=new Set(e.events.filter(w=>w.done).map(w=>w.stepId)),d=new Set(e.events.filter(w=>w.err).map(w=>w.stepId)),f=new Map;for(const w of e.events){if(!w.line)continue;const R=f.get(w.stepId)??[];R.push(w.line),f.set(w.stepId,R)}const m=le.map(w=>{var ge;const R=a.has(w.id),D=d.has(w.id),he=D?E("failed","bad"):R?E("done","ok"):E("pending","neutral"),fe=(f.get(w.id)??[]).slice(-5),me=(ge=e.events.find(ne=>ne.stepId===w.id&&ne.err))==null?void 0:ge.err,Ce=w.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${g?` <a href="${s(g)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${R?"step-done":""} ${D?"step-error":""}">
          <div class="step-head">${he} <strong>${s(w.title)}</strong></div>
          ${Ce}
          ${fe.length?`<pre class="step-log">${fe.map(ne=>s(ne)).join(`
`)}</pre>`:""}
          ${me?`<p class="error small">${s(me)}</p>`:""}
        </li>
      `}).join(""),P=e.events.some(w=>w.err),I=le.every(w=>a.has(w.id))||e.events.some(w=>w.stepId==="handshake"&&w.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${m}</ol>
        ${I&&!P?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        ${P?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function Q(i,o){switch(i){case"pick-network":e.chainId=Number(o.dataset.chainId),e.execId=null,e.beaconId=null,v();break;case"goto-network":e.step="network",v();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",v();break;case"goto-mode":e.step="mode",v(),u();break;case"goto-review":if(j(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){v();break}e.step="review",v();break;case"start-setup":oe();break}}function j(){const i=t.querySelectorAll('input[name="mode"]');for(const w of Array.from(i))w.checked&&(e.archive=w.value==="archive");const o=t.querySelector("#data-dir-input"),g=t.querySelector("#jwt-path-input");o&&(e.dataDir=o.value.trim()),g&&(e.jwtPath=g.value.trim());const a=t.querySelector("#exec-http-port-input"),d=t.querySelector("#beacon-http-port-input"),f=t.querySelector("#exec-p2p-port-input");a&&(e.execHTTPPort=a.value.trim()),d&&(e.beaconHTTPPort=d.value.trim()),f&&(e.execP2PPort=f.value.trim());const m=t.querySelector("#rpc-bind-addr-input");m&&(e.rpcBindAddr=m.value.trim());const P=t.querySelector("#checkpoint-url-input");P&&(e.checkpointUrl=P.value.trim());const I=t.querySelector("#snapshot-key-input");I&&(e.snapshotKey=I.value.trim()),e.execHTTPPortError=G(e.execHTTPPort).error??null,e.beaconHTTPPortError=G(e.beaconHTTPPort).error??null,e.execP2PPortError=G(e.execP2PPort).error??null,e.rpcBindAddrError=A(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?$(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function A(i){if(!i)return{};const o=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(i);return o?o.slice(1).every(g=>Number(g)<=255)?{addr:i}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(i)&&i.includes(":")?{addr:i}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const O=/^\d+$/;function G(i){if(!i)return{};if(!O.test(i))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const o=Number(i);return!Number.isInteger(o)||o<1||o>65535?{error:"Port must be between 1 and 65535."}:{port:o}}function _(i,o){const{port:g}=G(i);if(!(g===void 0||g===o))return g}async function oe(){var f;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(f=e.streamStop)==null||f.call(e),e.streamStop=null,v();const i={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(i.DataDir=e.dataDir),e.jwtPath&&(i.JWTPath=e.jwtPath);const o=_(e.execHTTPPort,ae),g=_(e.beaconHTTPPort,re),a=_(e.execP2PPort,se);o!==void 0&&(i.ExecHTTPPort=o),g!==void 0&&(i.BeaconHTTPPort=g),a!==void 0&&(i.ExecP2PPort=a);const{addr:d}=A(e.rpcBindAddr);d!==void 0&&(i.RPCBindAddr=d),e.checkpoint?e.checkpointUrl&&(i.CheckpointURL=e.checkpointUrl):i.NoCheckpoint=!0,e.execSnapshot&&(i.ExecSnapshot=!0,i.SnapshotKey=e.snapshotKey);try{await Ae(e.targetId,i)}catch(m){if(!(m instanceof ue&&m.status===409)){e.starting=!1,e.startError=String(m instanceof Error?m.message:m),v();return}}e.starting=!1,e.step="run",v(),e.streamStop=De(e.targetId,m=>{n||(e.events.push(m),e.step==="run"&&v())})}function ie(i){const o=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],a=o.map(d=>d.id).indexOf(i);return`
      <ol class="wizard-progress">
        ${o.map((d,f)=>`<li class="${f===a?"current":f<a?"past":"future"}">${s(d.label)}</li>`).join("")}
      </ol>
    `}return()=>{var i;n=!0,(i=e.streamStop)==null||i.call(e)}}const ut=document.querySelector("#app"),{contentEl:pt,setActiveNav:ht}=Ge(ut);let F=null;function ft(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"targets"};const[n,e]=r;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function W(t){const r=document.createElement("div");return pt.replaceChildren(r),t(r)}function Ee(){if(F){try{F()}catch{}F=null}const{screen:t,id:r}=ft();switch(ht(t),t){case"setup":if(!r){location.hash="#/targets";return}F=W(n=>dt(n,r));break;case"dash":if(!r){location.hash="#/targets";return}F=W(n=>Ze(n,r));break;case"logs":if(!r){location.hash="#/targets";return}F=W(n=>Qe(n,r));break;case"security":if(!r){location.hash="#/targets";return}F=W(n=>tt(n,r));break;case"diag":if(!r){location.hash="#/targets";return}F=W(n=>et(n,r));break;case"settings":F=W(n=>at(n));break;case"targets":default:F=W(n=>st(n));break}}window.addEventListener("hashchange",Ee);Ee();
