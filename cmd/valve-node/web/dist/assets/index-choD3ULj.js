var Le=Object.defineProperty;var Ie=(t,r,n)=>r in t?Le(t,r,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[r]=n;var ve=(t,r,n)=>Ie(t,typeof r!="symbol"?r+"":r,n);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const f of document.querySelectorAll('link[rel="modulepreload"]'))e(f);new MutationObserver(f=>{for(const h of f)if(h.type==="childList")for(const y of h.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(f){const h={};return f.integrity&&(h.integrity=f.integrity),f.referrerPolicy&&(h.referrerPolicy=f.referrerPolicy),f.crossOrigin==="use-credentials"?h.credentials="include":f.crossOrigin==="anonymous"?h.credentials="omit":h.credentials="same-origin",h}function e(f){if(f.ep)return;f.ep=!0;const h=n(f);fetch(f.href,h)}})();function G(){return H("/api/catalog")}function Y(){return H("/api/targets")}function be(t){return H("/api/targets",{method:"POST",headers:ee,body:JSON.stringify(t)})}function He(t){return H(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Be(t,r){return H(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(r)}`)}function Re(t,r){return H(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:ee,body:JSON.stringify(r)})}function Ae(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function De(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function Ue(t,r=200){return H(`/api/targets/${encodeURIComponent(t)}/logs?n=${r}`)}function Ne(t,r){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{r(JSON.parse(e.data))}catch{}},()=>n.close()}function ye(t,r){const n=r===void 0?{}:{lines:r};return H(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:ee,body:JSON.stringify(n)})}function Me(t,r,n){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/${n}`,{method:"POST"})}function qe(t,r){return H(`/api/targets/${encodeURIComponent(t)}/services/${r}/clear`,{method:"POST",headers:ee,body:JSON.stringify({Confirm:r})})}function Fe(t){return H(`/api/targets/${encodeURIComponent(t)}/du`)}function Oe(t){return H(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function je(t){return H(`/api/targets/${encodeURIComponent(t)}/firewall`)}function ze(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function _e(t){return H(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function We(){return H("/api/settings")}function Je(t){return H("/api/settings",{method:"PUT",headers:ee,body:JSON.stringify(t)})}class ue extends Error{constructor(n,e){super(e);ve(this,"status");this.name="ApiError",this.status=n}}const ee={"Content-Type":"application/json"};async function H(t,r){const n=await fetch(t,r);if(!n.ok){let f=n.statusText||`HTTP ${n.status}`;try{const h=await n.json();h&&typeof h.error=="string"&&h.error&&(f=h.error)}catch{}throw new ue(n.status,f)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const $e="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function A(t,r){const n=t&&r&&r!==$e?` <span class="footer-sep">·</span> <a href="${s(r)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${s($e)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function Ke(t){t.innerHTML=`
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
  `;const r=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:f=>{for(const h of n)h.classList.toggle("active",h.dataset.nav===f)}}}function j(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ge(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ye(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const r=Math.round(t/60),n=Math.floor(r/60),e=r%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const f=Math.floor(n/24),h=n%24;return`~${f}d ${h}h`}function I(t,r){return`<span class="badge badge-${r}">${s(t)}</span>`}function we(t){return`<span class="dot dot-${t}"></span>`}const ke=["B","KB","MB","GB","TB","PB"];function _(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let r=t,n=0;for(;r>=1024&&n<ke.length-1;)r/=1024,n++;const e=r<10?2:r<100?1:0;return`${r.toFixed(e)} ${ke[n]}`}async function pe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function V(t,r){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const f=e.dataset.action;f&&r(f,e,n)})}function de(t,r,n){const e=r.find(h=>h.value===n),f=r.map(h=>`
      <li class="dropdown-option${h.value===n?" selected":""}" role="option"
          aria-selected="${h.value===n}" data-value="${s(h.value)}">
        ${s(h.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${s(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${s(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${f}</ul>
    </div>
  `}function Q(t){t.querySelectorAll(".dropdown.open").forEach(r=>{var n;r.classList.remove("open"),(n=r.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function Ee(t,r){t.addEventListener("click",f=>{const h=f.target,y=h.closest(".dropdown-trigger");if(y&&t.contains(y)){const E=y.closest(".dropdown"),C=!!E&&!E.classList.contains("open");Q(t),E&&C&&(E.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const b=h.closest(".dropdown-option");if(b&&t.contains(b)){const E=b.closest(".dropdown");Q(t),r((E==null?void 0:E.dataset.dropdown)??"",b.dataset.value??"");return}Q(t)});const n=f=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const h=f.target;(!h.closest(".dropdown")||!t.contains(h))&&Q(t)},e=f=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}f.key==="Escape"&&Q(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const Ve=85,ce={exec:"Execution",beacon:"Beacon"};function Xe(t,r){let n=!1,e=null,f=null,h=null,y=null,b=null,E=null,C=null,S=null;const L={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${s(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${A()}</div>`;const T=t.querySelector("#dash-body"),u=t.querySelector("#dash-footer");T.addEventListener("click",a=>{const d=a.target.closest("[data-action]");if(!d||!T.contains(d))return;const v=d.dataset.action;if(v==="svc-action"){const m=d.dataset.svc,P=d.dataset.kind;m&&P&&O(m,P)}else if(v==="open-clear"){const m=d.dataset.svc;m&&ie(m)}else if(v==="copy"){const m=d.dataset.copy;m&&oe(d,m)}else v==="retry-du"?c():v==="retry-endpoints"&&p()}),l();async function l(){let a,d;try{const[m,P]=await Promise.all([Y(),G()]);a=m.find(B=>B.id===r),d=P}catch(m){if(n)return;T.innerHTML=`<p class="error">Failed to load target: ${s(String(m))}</p>`;return}if(n)return;if(!a){T.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!a.wire){T.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const v=d==null?void 0:d.networks.find(m=>m.ChainID===a.wire.ChainID);v&&(u.innerHTML=A(v.Name,v.LearnURL)),T.innerHTML='<p class="muted">Connecting…</p>',e=De(r,m=>{n||(w(m),f=m,h=m,k())}),c(),p()}async function c(){E=null;try{b=await Fe(r)}catch(a){b=null,E=String(a instanceof Error?a.message:a)}n||k()}async function p(){S=null;try{C=await Oe(r)}catch(a){C=null,S=String(a instanceof Error?a.message:a)}n||k()}function w(a){if(!f)return;const d=(new Date(a.at).getTime()-new Date(f.at).getTime())/1e3,v=a.execHead-f.execHead;if(d>0&&v>=0){const m=v/d;y=y===null?m:y*.7+m*.3}}function k(){if(!h)return;const a=h;T.innerHTML=`
      <div class="card-grid">
        ${U(a)}
        ${W(a)}
        ${X(a)}
        ${Z(a)}
        ${J(a)}
        ${N()}
        ${se(a)}
      </div>
      <p class="muted small">Last updated ${s(new Date(a.at).toLocaleTimeString())}</p>
    `}function R(a){const v=a.refHead>0?a.refHead-a.execHead:null,m=v!==null&&v>0&&y&&y>0?Ye(v/y):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:m}}function U(a){const{lag:d,eta:v}=R(a);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${a.execSyncing?I("syncing","warn"):I("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${j(a.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${d!==null?j(a.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${d!==null?j(Math.max(d,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function W(a){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${a.beaconDistance===0?I("synced","ok"):I("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${j(a.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${j(a.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function X(a){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${j(a.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${j(a.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Z(a){const d=a.diskUsedPct>=Ve;return`
      <div class="card ${d?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${d?"meter-warn":""}" style="width:${Math.min(a.diskUsedPct,100)}%"></div></div>
        <p>${Ge(a.diskUsedPct)} used</p>
      </div>
    `}function J(a){if(E)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${s(E)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!b)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const d=b.ExpectedExecBytes>0?Math.min(b.ExecBytes/b.ExpectedExecBytes*100,100):0,v=b.ExpectedBeaconBytes>0?Math.min(b.BeaconBytes/b.ExpectedBeaconBytes*100,100):0,{lag:m,eta:P}=R(a),B=m!==null&&m>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${_(b.ExecBytes)} of ~${_(b.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${d}%"></div></div>
        ${B?`<p class="muted small">Estimated time remaining: ${s(P)}</p>`:""}
        <p class="muted small">Beacon — ${_(b.BeaconBytes)} of ~${_(b.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${_(b.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(b.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(b.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function N(){if(S)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(S)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!C)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const a=C,d=a.ExecReachable&&!a.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=a.Access==="ssh"?`
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
        ${v}
      </div>
    `}function D(a,d){const v=ce[a],m=L[a],P=(B,x,q)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${a}" data-kind="${B}" ${m!==null||q?"disabled":""}>${m===B?K():s(x)}</button>`;return`
      <div class="service-row">
        <span>${s(v)} ${d?I("active","ok"):I("down","bad")}</span>
        <div class="service-actions">
          ${P("start","Start",d)}
          ${P("stop","Stop",!d)}
          ${P("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${a}" ${m!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function se(a){return`
      <div class="card">
        <h3>Services</h3>
        ${D("exec",a.execActive)}
        ${D("beacon",a.beaconActive)}
        ${$?`<p class="error small">${s($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function K(){return'<span class="spinner" aria-label="working"></span>'}async function O(a,d){if(L[a]===null){L[a]=d,$=null,k();try{await Me(r,a,d)}catch(v){$=`${ce[a]} ${d} failed: ${v instanceof Error?v.message:String(v)}`}L[a]=null,n||k()}}async function oe(a,d){const v=await pe(d),m=a.textContent;a.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{n||(a.textContent=m)},1500)}function ie(a){const d=ce[a],v=b?_(a==="exec"?b.ExecBytes:b.BeaconBytes):"unknown (disk usage hasn't loaded)";o(`
        <h2>Clear ${s(d)} data</h2>
        <p class="error">
          This stops the ${s(d.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(v)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(a)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,B=>{if(B==="cancel"){g();return}B==="confirm"&&i(a)});const m=document.getElementById("clear-confirm-input"),P=document.getElementById("clear-confirm-btn");m==null||m.addEventListener("input",()=>{P&&(P.disabled=m.value.trim()!==a)}),m==null||m.focus()}async function i(a){const d=document.getElementById("clear-confirm-btn");d&&(d.disabled=!0,d.textContent="Clearing…");try{await qe(r,a),g(),c()}catch(v){const m=document.querySelector("#clear-modal .modal");if(m){const P=document.createElement("p");P.className="error small",P.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,m.appendChild(P)}d&&(d.disabled=!1,d.textContent="Clear and resync")}}function o(a,d){g();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${a}</div>`,v.addEventListener("click",m=>{const P=m.target.closest("[data-modal-action]");P!=null&&P.dataset.modalAction&&d(P.dataset.modalAction),m.target===v&&d("cancel")}),document.body.appendChild(v)}function g(){var a;(a=document.getElementById("clear-modal"))==null||a.remove()}return()=>{n=!0,e==null||e(),g()}}const Pe=500,Te="valve-node.explain-consent";function Ze(t,r){let n=!1,e=null;const f=[];t.innerHTML=`
    <h1>Logs: ${s(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${A()}</div>
  `;const h=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");V(t,l=>{l==="explain"&&S()}),b();async function b(){let l,c;try{const[w,k]=await Promise.all([Y(),G()]);l=w.find(R=>R.id===r),c=k}catch(w){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${s(String(w))}</p>`;return}if(n)return;if(!l){h.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const p=c==null?void 0:c.networks.find(w=>w.ChainID===l.wire.ChainID);p&&(y.innerHTML=A(p.Name,p.LearnURL));try{const w=await Ue(r,200);if(n)return;f.push(...w)}catch(w){if(n)return;h.innerHTML=`<p class="error">Failed to load logs: ${s(String(w))}</p>`;return}E(),e=Ne(r,w=>{n||(f.push(w),f.length>Pe&&f.splice(0,f.length-Pe),E())})}function E(){const l=f.filter(p=>p.severity==="error"||p.severity==="critical");h.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${f.map(C).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${I(String(l.length),l.length?"bad":"neutral")}</h2>
          <div class="log-lines">${l.length?l.slice().reverse().map(C).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const c=h.querySelector(".log-lines");c&&(c.scrollTop=c.scrollHeight)}function C(l){const c=l.severity||"info",p=l.learnUrl?` <a href="${s(l.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(c)}">
        <span class="log-time">${s(new Date(l.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(l.unit)}</span>
        <span class="log-sev">${s(c)}</span>
        <span class="log-text">${s(l.line)}</span>
        ${l.explain?`<div class="log-explain">${s(l.explain)}${p}</div>`:""}
      </div>
    `}async function S(){const l=f.filter(p=>p.severity==="error"||p.severity==="critical").map(p=>p.line).slice(-40);if(!(localStorage.getItem(Te)==="1")){L(l);return}await $(l)}function L(l){const c=l.length?`<pre class="explain-excerpt">${l.map(p=>s(p)).join(`
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
    `,p=>{p==="proceed"?(localStorage.setItem(Te,"1"),u(),$(l)):u()})}async function $(l){T('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const c=l.length?await ye(r,l):await ye(r);if(n)return;T(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(c.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${c.sentExcerpt.map(p=>s(p)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,p=>{p==="close"&&u()})}catch(c){if(n)return;if(c instanceof ue&&c.status===409){T(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,p=>{p==="close"&&u()});return}T(`
        <h2>Explain failed</h2>
        <p class="error">${s(c instanceof Error?c.message:String(c))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,p=>{p==="close"&&u()})}}function T(l,c){u();const p=document.createElement("div");p.className="modal-overlay",p.id="explain-modal",p.innerHTML=`<div class="modal">${l}</div>`,p.addEventListener("click",w=>{const k=w.target.closest("[data-modal-action]");k!=null&&k.dataset.modalAction&&c(k.dataset.modalAction),w.target===p&&c("cancel")}),document.body.appendChild(p)}function u(){var l;(l=document.getElementById("explain-modal"))==null||l.remove()}return()=>{n=!0,e==null||e(),u()}}function Qe(t,r){let n=!1,e=null,f=null,h=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${s(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${A()}</div>`;const b=t.querySelector("#diag-body"),E=t.querySelector("#diag-footer");V(t,(c,p)=>{var w;if(c==="run")S();else if(c==="toggle")(w=p.closest(".check-item"))==null||w.classList.toggle("expanded");else if(c==="copy"){const k=p.dataset.copy;k&&l(p,k)}}),C();async function C(){let c,p;try{const[k,R]=await Promise.all([Y(),G()]);c=k.find(U=>U.id===r),p=R}catch(k){if(n)return;b.innerHTML=`<p class="error">Failed to load target: ${s(String(k))}</p>`;return}if(n)return;if(!c){b.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const w=p==null?void 0:p.networks.find(k=>k.ChainID===c.wire.ChainID);w&&(E.innerHTML=A(w.Name,w.LearnURL));try{e=await _e(r),y=!0}catch(k){f=String(k instanceof Error?k.message:k)}n||L()}async function S(){h=!0,f=null,L();try{e=await ze(r),y=!0}catch(c){f=String(c instanceof Error?c.message:c)}h=!1,n||L()}function L(){b.innerHTML=`
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
      ${f?`<p class="error">${s(f)}</p>`:""}
      ${$()}
    `}function $(){if(!y&&!f)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const c=new Date(e.at).toLocaleString(),p=e.failedId?`<p><strong>Failed at: ${s(T(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(c)} — trigger: ${s(e.trigger)}</p>
      ${p}
      <ul class="check-list">${e.items.map(u).join("")}</ul>
    `}function T(c){var p;return((p=e==null?void 0:e.items.find(w=>w.ID===c))==null?void 0:p.Title)??c}function u(c){const p=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral",w=c.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${w?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${I(w?"failed here":c.Status,p)}
          <strong>${s(c.Title)}</strong>
          <span class="muted small check-detail-inline">${s(c.Detail)}</span>
        </button>
        <div class="check-body">
          <details${w?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${s(c.Why)}</p>
          </details>
          ${c.Fix?`
                <details${w?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(c.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(c.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function l(c,p){const w=await pe(p),k=c.textContent;c.textContent=w?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=k)},1500)}return()=>{n=!0}}function et(t,r){let n=!1,e=[],f=null,h=!1,y=!1;t.innerHTML=`<h1>Security: ${s(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${A()}</div>`;const b=t.querySelector("#sec-body"),E=t.querySelector("#sec-footer");V(t,(u,l)=>{var c;if(u==="rerun")S();else if(u==="toggle")(c=l.closest(".check-item"))==null||c.classList.toggle("expanded");else if(u==="copy"){const p=l.dataset.copy;p&&T(l,p)}}),C();async function C(){let u,l;try{const[p,w]=await Promise.all([Y(),G()]);u=p.find(k=>k.id===r),l=w}catch(p){if(n)return;b.innerHTML=`<p class="error">Failed to load target: ${s(String(p))}</p>`;return}if(n)return;if(!u){b.innerHTML=`<p class="error">Target "${s(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const c=l==null?void 0:l.networks.find(p=>p.ChainID===u.wire.ChainID);c&&(E.innerHTML=A(c.Name,c.LearnURL)),await S()}async function S(){h=!0,f=null,L();try{e=await je(r),y=!0}catch(u){f=String(u instanceof Error?u.message:u)}h=!1,n||L()}function L(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${h?"disabled":""}>${h?"Re-running…":"Re-run checks"}</button>
      </div>
      ${f?`<p class="error">${s(f)}</p>`:""}
      ${!y&&h?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function $(u){const l=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${I(u.Status,l)}
          <strong>${s(u.Title)}</strong>
          <span class="muted small check-detail-inline">${s(u.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${s(u.Why)}</p>
          </details>
          ${u.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(u.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(u.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(u,l){const c=await pe(l),p=u.textContent;u.textContent=c?"Copied!":"Copy failed",setTimeout(()=>{n||(u.textContent=p)},1500)}return()=>{n=!0}}const tt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function nt(t){let r=!1,n=!1,e=!1,f=null,h=!1,y=null,b=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${A()}`;const E=t.querySelector("#settings-body");V(t,$=>{if($==="save"&&L(),$==="clear-key"){if(!y)return;n=!0;const T=t.querySelector("#ai-key");T&&(T.value=""),S(y)}}),Ee(t,($,T)=>{$!=="ai-provider"||!y||(b=T,h=!1,S(y))}),C();async function C(){try{const $=await We();if(r)return;y=$,S($)}catch($){if(r)return;E.innerHTML=`<p class="error">Failed to load settings: ${s(String($))}</p>`}}function S($){var l;const T=b??$.aiProvider;E.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${de("ai-provider",tt.map(c=>({value:c.value,label:c.label})),T)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${$.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${$.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${s($.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${f?`<p class="error">${s(f)}</p>`:""}
        ${h?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const u=t.querySelector("#ai-key");u==null||u.addEventListener("input",()=>{n=!0,h=!1}),(l=t.querySelector("#ref-rpc-base"))==null||l.addEventListener("input",()=>{h=!1})}async function L(){const $=t.querySelector("#ai-key"),T=t.querySelector("#ref-rpc-base");if(!$||!T||!y)return;const u={aiProvider:b??y.aiProvider,refRpcBase:T.value.trim()};n&&(u.aiKey=$.value),e=!0,f=null,h=!1,S(y);try{const l=await Je(u);if(r)return;y=l,n=!1,e=!1,h=!0,S(l)}catch(l){if(r)return;e=!1,f=String(l instanceof Error?l.message:l),S(y)}}return()=>{r=!0}}const rt="local";function at(t){let r=!1,n=!1,e=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${A()}
  `;const f=t.querySelector("#targets-body");V(t,(u,l)=>{E(u,l)}),h();async function h(){try{const[u,l]=await Promise.all([Y(),G()]);if(r)return;b(u,l)}catch(u){if(r)return;f.innerHTML=`<p class="error">Failed to load machines: ${s(String(u))}</p>`}}function y(){e&&b(e.targets,e.catalog)}function b(u,l){e={targets:u,catalog:l};const c=!it(),p=[...u].sort((R,U)=>(R.mode==="local"?-1:0)-(U.mode==="local"?-1:0)),w=p.length?`<div class="card-grid">${p.map(R=>st(R,l)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${c?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node is running here as your <strong>controller</strong> — it drives nodes but doesn't host them. Add a Linux server over SSH to run one."}
          </p>
        </div>
      `,k=`
      <div class="add-actions">
        ${c?'<button class="btn" data-action="add-local">Add this machine</button>':""}
        <button class="btn${c?" btn-ghost":""}" data-action="toggle-ssh">
          ${n?"Cancel":"Add a server (SSH)"}
        </button>
      </div>
    `;f.innerHTML=`
      <section class="section">
        <div class="section-head">
          <h2>Your machines</h2>
          ${k}
        </div>
        ${n?ot():""}
        ${w}
      </section>
    `}async function E(u,l){var c;if(u==="add-local"){await C();return}if(u==="delete-target"){const p=l.dataset.id;if(!p||!confirm(`Remove target "${p}"? This does not touch anything already running on it.`))return;await S(p);return}if(u==="toggle-ssh"){n=!n,T(),y(),n&&((c=t.querySelector("#ssh-host"))==null||c.focus());return}u==="add-ssh"&&await L()}async function C(){T();try{await be({id:rt,mode:"local"}),await h()}catch(u){$(u)}}async function S(u){try{await He(u),await h()}catch(l){$(l)}}async function L(){const u=t.querySelector("#ssh-host"),l=t.querySelector("#ssh-user"),c=t.querySelector("#ssh-key"),p=t.querySelector("#ssh-port"),w=t.querySelector("#ssh-id");if(!u||!l||!c||!p||!w)return;const k=u.value.trim(),R=l.value.trim(),U=c.value.trim(),W=p.value.trim(),X=w.value.trim();if(T(),!k||!R||!U){$(new Error("host, user, and key path are required"));return}const Z=X||ct(k),J={Host:k,User:R,KeyPath:U};if(W){const D=Number.parseInt(W,10);if(!Number.isFinite(D)||D<=0){$(new Error("port must be a positive number"));return}J.Port=D}const N=t.querySelector("#ssh-submit");N&&(N.disabled=!0,N.textContent="Connecting…");try{await be({id:Z,mode:"ssh",ssh:J}),n=!1,await h()}catch(D){$(D),N&&(N.disabled=!1,N.textContent="Add server")}}function $(u){let l=t.querySelector("#targets-error");l||(f.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),l=t.querySelector("#targets-error")),l.textContent=String(u instanceof Error?u.message:u)}function T(){var u;(u=t.querySelector("#targets-error"))==null||u.remove()}return()=>{r=!0}}function st(t,r){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",f=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let h,y;if(!n)h=I("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const b=r.networks.find(C=>C.ChainID===n.ChainID),E=b?b.Name:`chain ${n.ChainID}`;h=`${I(E,"ok")} ${I(n.ExecID,"neutral")} ${I(n.BeaconID,"neutral")}${n.Archive?" "+I("archive","warn"):""}`,y=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${s(t.id)}</h2>
      <p class="muted">${f}</p>
      <p>${h}</p>
      <div class="card-actions">
        ${y}
        <button class="btn btn-danger" data-action="delete-target" data-id="${s(t.id)}">Remove</button>
      </div>
    </div>
  `}function ot(){return`
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
  `}function it(){const t=navigator.userAgentData,r=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(r)&&!/linux|android/i.test(r)}function ct(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const le=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ne=8545,re=5052,ae=30303,lt=[369,943,1],xe={369:"default",943:"practise here first"};function dt(t,r){let n=!1;const e={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${A()}</div>`;const f=t.querySelector("#wizard-body"),h=t.querySelector("#wizard-footer");V(t,(i,o)=>{J(i,o)}),Ee(t,(i,o)=>{i==="exec-select"?e.execId=o:i==="beacon-select"&&(e.beaconId=o),b()}),t.addEventListener("change",i=>{const o=i.target;o instanceof HTMLInputElement&&(o.id==="data-dir-input"?(N(),p()):o.id==="checkpoint-toggle"&&(e.checkpoint=o.checked,b()))}),y();async function y(){try{const[i,o]=await Promise.all([G(),Y()]);if(n)return;e.catalog=i;const g=o.find(a=>a.id===r);g!=null&&g.wire&&(e.chainId=g.wire.ChainID,e.execId=g.wire.ExecID,e.beaconId=g.wire.BeaconID,e.archive=g.wire.Archive,g.wire.ExecHTTPPort&&(e.execHTTPPort=String(g.wire.ExecHTTPPort)),g.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(g.wire.BeaconHTTPPort)),g.wire.ExecP2PPort&&(e.execP2PPort=String(g.wire.ExecP2PPort)),g.wire.RPCBindAddr&&(e.rpcBindAddr=g.wire.RPCBindAddr)),b()}catch(i){if(n)return;e.loadError=String(i instanceof Error?i.message:i),b()}}function b(){if(e.loadError){f.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(f.innerHTML=`
      ${ie(e.step)}
      ${C()}
    `,E())}function E(){var o;const i=(o=e.catalog)==null?void 0:o.networks.find(g=>g.ChainID===e.chainId);h.innerHTML=i?A(i.Name,i.LearnURL):A()}function C(){switch(e.step){case"network":return S();case"clients":return L();case"mode":return W();case"review":return X();case"run":return Z()}}function S(){const i=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${lt.map(g=>{const a=i.networks.find(m=>m.ChainID===g);if(!a)return"";const d=e.chainId===g,v=xe[g]?I(xe[g],g===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${d?"selected":""}" data-action="pick-network" data-chain-id="${g}" type="button">
          <h3>${s(a.Name)} <span class="muted">(chain ${g})</span></h3>
          ${v}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function L(){const i=e.catalog,o=i.networks.find(d=>d.ChainID===e.chainId);if(!o)return'<p class="error">Unknown network.</p>';(e.execId===null||!o.ExecClients.includes(e.execId))&&(e.execId=o.ExecClients[0]??null),(e.beaconId===null||!o.BeaconClients.includes(e.beaconId))&&(e.beaconId=o.BeaconClients[0]??null);const g=o.ExecClients.map(d=>k(d,i)),a=o.BeaconClients.map(d=>k(d,i));return`
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
    `}function $(i){return i<=0?"—":i>=1?`~${i.toFixed(1)} TB`:`~${Math.round(i*1e3)} GB`}const T=1.1;function u(i){const o=i.ArchiveSizeTB*1e12*T;return{archive:o,full:o/2}}function l(i,o){if(!i)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${s(o)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${s(o)}</code>: ${s(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==o)return"";const g=u(i),a=e.freeBytes>=g.archive,d=e.freeBytes>=g.full,v=`<p class="muted small">Free at <code>${s(o)}</code>: <strong>${_(e.freeBytes)}</strong> — archive ${a?"fits":"won't fit"} (~${$(i.ArchiveSizeTB)}), full ${d?"fits":"won't fit"} (~${$(i.ArchiveSizeTB/2)}).</p>`;let m="";return e.downgradeNote?m=`<p class="banner banner-warn">${s(e.downgradeNote)}</p>`:d||(m=`<p class="banner banner-warn">Neither mode fits at this location (full needs ~${$(i.ArchiveSizeTB/2)}). Choose a location with more space.</p>`),v+m}function c(i,o){if(e.downgradeNote=null,!i||e.freeBytes===null)return;const g=u(i);e.archive&&e.freeBytes<g.archive&&e.freeBytes>=g.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${o} for archive (~${$(i.ArchiveSizeTB)}) — switched to Full (~${$(i.ArchiveSizeTB/2)}). Pick a location with more room to run archive.`)}async function p(){var g;if(e.chainId===null)return;const i=(g=e.catalog)==null?void 0:g.networks.find(a=>a.ChainID===e.chainId),o=(e.dataDir||`/var/lib/valve-node/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,b();try{const{freeBytes:a}=await Be(e.targetId,o);if(n)return;e.freeBytes=a,e.probedPath=o,c(i,o)}catch(a){if(n)return;e.freeBytes=null,e.probedPath=o,e.diskError=String(a instanceof Error?a.message:a)}e.diskProbing=!1,b()}function w(i){return i?/^https?:\/\/.+/i.test(i)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function k(i,o){const g=o.clients.find(a=>a.id===i);return{value:i,label:g?`${g.id} — ${R(g.repo)}`:i}}function R(i){const o=i.split("/");return o.length>=4?o[3]:i}function U(i,o){const g=i?o.clients.find(d=>d.id===i):void 0;if(!g)return"";const a=g.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(g.repo)}" target="_blank" rel="noopener noreferrer">${s(a)}</a></p>`}function W(){var m;const i=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"",o=(m=e.catalog)==null?void 0:m.networks.find(P=>P.ChainID===e.chainId),g=(o==null?void 0:o.ArchiveSizeTB)??0,a=o?$(g/2):"Smaller",d=o?$(g):"Much larger",v=o?` on ${s(o.Name)}`:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <p class="muted">
          Both modes run a fully-validating node — same security, same current-state RPC.
          The difference is how much <strong>historical</strong> state is kept.
        </p>
        <table class="compare-table">
          <thead>
            <tr><th>What you get</th><th>Full</th><th>Archive</th></tr>
          </thead>
          <tbody>
            <tr><th>Current state &amp; recent blocks</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Send transactions, normal RPC</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Historical state (balances, <code>eth_call</code>) at any past block</th><td class="limited">Recent only (~128 blocks)</td><td class="yes">Full history</td></tr>
            <tr><th>Tracing / <code>debug_trace</code> on old blocks</th><td class="limited">Recent only</td><td class="yes">Full history</td></tr>
            <tr><th>Approx. disk footprint${v}</th><td class="yes">${a}</td><td class="limited">${d}</td></tr>
            <tr><th>Initial sync time${v}</th><td class="yes">${o?s(o.SyncLabel):"Faster"}</td><td class="limited">${o?s(o.GenesisSyncLabel):"Much slower"}</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <p class="muted small">
          Disk sizes and sync times are rough baselines — both vary by client and scale with the
          target's CPU and disk speed.
        </p>
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

        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="checkpoint-toggle" ${e.checkpoint?"checked":""} />
            <span><strong>Checkpoint sync</strong> — start near the chain head in minutes (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.</span>
          </label>
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${s((o==null?void 0:o.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${s((o==null?void 0:o.CheckpointURL)??"")}" value="${s(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${s(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${s(i)}/jwt.hex" value="${s(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${ne})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${ne}" value="${s(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${s(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${re})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${re}" value="${s(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${s(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${ae})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${ae}" value="${s(e.execP2PPort)}" />
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
    `}function X(){const o=e.catalog.networks.find(F=>F.ChainID===e.chainId),g=e.dataDir||`/var/lib/valve-node/${e.chainId}`,a=e.jwtPath||`${g}/jwt.hex`,d=le.map(F=>`<li>${s(F.title)}</li>`).join(""),v=O(e.execHTTPPort,ne),m=O(e.beaconHTTPPort,re),P=O(e.execP2PPort,ae),B=v||m||P?`<tr><th>Non-default ports</th><td>${[v?`exec HTTP ${v}`:null,m?`beacon HTTP ${m}`:null,P?`exec p2p ${P}`:null].filter(F=>F!==null).map(s).join(", ")}</td></tr>`:"",{addr:x}=D(e.rpcBindAddr),q=x?`<tr><th>RPC bind address</th><td><code>${s(x)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
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
            ${B}
            ${q}
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
    `}function Z(){const o=e.catalog.networks.find(x=>x.ChainID===e.chainId),g=o==null?void 0:o.LearnURL,a=new Set(e.events.filter(x=>x.done).map(x=>x.stepId)),d=new Set(e.events.filter(x=>x.err).map(x=>x.stepId)),v=new Map;for(const x of e.events){if(!x.line)continue;const q=v.get(x.stepId)??[];q.push(x.line),v.set(x.stepId,q)}const m=le.map(x=>{var ge;const q=a.has(x.id),F=d.has(x.id),he=F?I("failed","bad"):q?I("done","ok"):I("pending","neutral"),fe=(v.get(x.id)??[]).slice(-5),me=(ge=e.events.find(te=>te.stepId===x.id&&te.err))==null?void 0:ge.err,Ce=x.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${g?` <a href="${s(g)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${q?"step-done":""} ${F?"step-error":""}">
          <div class="step-head">${he} <strong>${s(x.title)}</strong></div>
          ${Ce}
          ${fe.length?`<pre class="step-log">${fe.map(te=>s(te)).join(`
`)}</pre>`:""}
          ${me?`<p class="error small">${s(me)}</p>`:""}
        </li>
      `}).join(""),P=e.events.some(x=>x.err),B=le.every(x=>a.has(x.id))||e.events.some(x=>x.stepId==="handshake"&&x.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${m}</ol>
        ${B&&!P?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${P?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function J(i,o){switch(i){case"pick-network":e.chainId=Number(o.dataset.chainId),e.execId=null,e.beaconId=null,b();break;case"goto-network":e.step="network",b();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",b();break;case"goto-mode":e.step="mode",b(),p();break;case"goto-review":if(N(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError){b();break}e.step="review",b();break;case"start-setup":oe();break}}function N(){const i=t.querySelectorAll('input[name="mode"]');for(const B of Array.from(i))B.checked&&(e.archive=B.value==="archive");const o=t.querySelector("#data-dir-input"),g=t.querySelector("#jwt-path-input");o&&(e.dataDir=o.value.trim()),g&&(e.jwtPath=g.value.trim());const a=t.querySelector("#exec-http-port-input"),d=t.querySelector("#beacon-http-port-input"),v=t.querySelector("#exec-p2p-port-input");a&&(e.execHTTPPort=a.value.trim()),d&&(e.beaconHTTPPort=d.value.trim()),v&&(e.execP2PPort=v.value.trim());const m=t.querySelector("#rpc-bind-addr-input");m&&(e.rpcBindAddr=m.value.trim());const P=t.querySelector("#checkpoint-url-input");P&&(e.checkpointUrl=P.value.trim()),e.execHTTPPortError=K(e.execHTTPPort).error??null,e.beaconHTTPPortError=K(e.beaconHTTPPort).error??null,e.execP2PPortError=K(e.execP2PPort).error??null,e.rpcBindAddrError=D(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?w(e.checkpointUrl):null}function D(i){if(!i)return{};const o=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(i);return o?o.slice(1).every(g=>Number(g)<=255)?{addr:i}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(i)&&i.includes(":")?{addr:i}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const se=/^\d+$/;function K(i){if(!i)return{};if(!se.test(i))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const o=Number(i);return!Number.isInteger(o)||o<1||o>65535?{error:"Port must be between 1 and 65535."}:{port:o}}function O(i,o){const{port:g}=K(i);if(!(g===void 0||g===o))return g}async function oe(){var v;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,b();const i={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(i.DataDir=e.dataDir),e.jwtPath&&(i.JWTPath=e.jwtPath);const o=O(e.execHTTPPort,ne),g=O(e.beaconHTTPPort,re),a=O(e.execP2PPort,ae);o!==void 0&&(i.ExecHTTPPort=o),g!==void 0&&(i.BeaconHTTPPort=g),a!==void 0&&(i.ExecP2PPort=a);const{addr:d}=D(e.rpcBindAddr);d!==void 0&&(i.RPCBindAddr=d),e.checkpoint?e.checkpointUrl&&(i.CheckpointURL=e.checkpointUrl):i.NoCheckpoint=!0;try{await Re(e.targetId,i)}catch(m){if(!(m instanceof ue&&m.status===409)){e.starting=!1,e.startError=String(m instanceof Error?m.message:m),b();return}}e.starting=!1,e.step="run",e.events=[],b(),(v=e.streamStop)==null||v.call(e),e.streamStop=Ae(e.targetId,m=>{n||(e.events.push(m),e.step==="run"&&b())})}function ie(i){const o=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],a=o.map(d=>d.id).indexOf(i);return`
      <ol class="wizard-progress">
        ${o.map((d,v)=>`<li class="${v===a?"current":v<a?"past":"future"}">${s(d.label)}</li>`).join("")}
      </ol>
    `}return()=>{var i;n=!0,(i=e.streamStop)==null||i.call(e)}}const ut=document.querySelector("#app"),{contentEl:pt,setActiveNav:ht}=Ke(ut);let M=null;function ft(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"targets"};const[n,e]=r;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function z(t){const r=document.createElement("div");return pt.replaceChildren(r),t(r)}function Se(){if(M){try{M()}catch{}M=null}const{screen:t,id:r}=ft();switch(ht(t),t){case"setup":if(!r){location.hash="#/targets";return}M=z(n=>dt(n,r));break;case"dash":if(!r){location.hash="#/targets";return}M=z(n=>Xe(n,r));break;case"logs":if(!r){location.hash="#/targets";return}M=z(n=>Ze(n,r));break;case"security":if(!r){location.hash="#/targets";return}M=z(n=>et(n,r));break;case"diag":if(!r){location.hash="#/targets";return}M=z(n=>Qe(n,r));break;case"settings":M=z(n=>nt(n));break;case"targets":default:M=z(n=>at(n));break}}window.addEventListener("hashchange",Se);Se();
