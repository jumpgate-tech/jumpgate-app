var qe=Object.defineProperty;var Fe=(t,a,n)=>a in t?qe(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var Pe=(t,a,n)=>Fe(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))e(d);new MutationObserver(d=>{for(const u of d)if(u.type==="childList")for(const b of u.addedNodes)b.tagName==="LINK"&&b.rel==="modulepreload"&&e(b)}).observe(document,{childList:!0,subtree:!0});function n(d){const u={};return d.integrity&&(u.integrity=d.integrity),d.referrerPolicy&&(u.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?u.credentials="include":d.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(d){if(d.ep)return;d.ep=!0;const u=n(d);fetch(d.href,u)}})();function Oe(){return D("/api/host")}function ee(){return D("/api/catalog")}function te(){return D("/api/targets")}function Se(t){return D("/api/targets",{method:"POST",headers:ie,body:JSON.stringify(t)})}function _e(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function je(t,a){return D(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(a)}`)}function ze(t,a){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:ie,body:JSON.stringify(a)})}function Ke(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function We(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Je(t,a=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function Ge(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Ee(t,a){const n=a===void 0?{}:{lines:a};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:ie,body:JSON.stringify(n)})}function Ye(t,a,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${a}/${n}`,{method:"POST"})}function Ve(t,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${a}/clear`,{method:"POST",headers:ie,body:JSON.stringify({Confirm:a})})}function Ze(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Xe(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Qe(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function et(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function tt(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function nt(){return D("/api/settings")}function at(t){return D("/api/settings",{method:"PUT",headers:ie,body:JSON.stringify(t)})}class $e extends Error{constructor(n,e){super(e);Pe(this,"status");this.name="ApiError",this.status=n}}const ie={"Content-Type":"application/json"};async function D(t,a){const n=await fetch(t,a);if(!n.ok){let d=n.statusText||`HTTP ${n.status}`;try{const u=await n.json();u&&typeof u.error=="string"&&u.error&&(d=u.error)}catch{}throw new $e(n.status,d)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const Ce="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function M(t,a){const n=t&&a&&a!==Ce?` <span class="footer-sep">·</span> <a href="${r(a)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(Ce)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function rt(t){t.innerHTML=`
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
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:d=>{for(const u of n)u.classList.toggle("active",u.dataset.nav===d)}}}function Y(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function st(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function ot(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const d=Math.floor(n/24),u=n%24;return`~${d}d ${u}h`}function C(t,a){return`<span class="badge badge-${a}">${r(t)}</span>`}function Le(t){return`<span class="dot dot-${t}"></span>`}const Ie=["B","KB","MB","GB","TB","PB"];function Z(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let a=t,n=0;for(;a>=1024&&n<Ie.length-1;)a/=1024,n++;const e=a<10?2:a<100?1:0;return`${a.toFixed(e)} ${Ie[n]}`}async function we(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function ne(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const d=e.dataset.action;d&&a(d,e,n)})}function ye(t,a,n){const e=a.find(u=>u.value===n),d=a.map(u=>`
      <li class="dropdown-option${u.value===n?" selected":""}" role="option"
          aria-selected="${u.value===n}" data-value="${r(u.value)}">
        ${r(u.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${r(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${r(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function se(t){t.querySelectorAll(".dropdown.open").forEach(a=>{var n;a.classList.remove("open"),(n=a.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function Ae(t,a){t.addEventListener("click",d=>{const u=d.target,b=u.closest(".dropdown-trigger");if(b&&t.contains(b)){const S=b.closest(".dropdown"),H=!!S&&!S.classList.contains("open");se(t),S&&H&&(S.classList.add("open"),b.setAttribute("aria-expanded","true"));return}const f=u.closest(".dropdown-option");if(f&&t.contains(f)){const S=f.closest(".dropdown");se(t),a((S==null?void 0:S.dataset.dropdown)??"",f.dataset.value??"");return}se(t)});const n=d=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const u=d.target;(!u.closest(".dropdown")||!t.contains(u))&&se(t)},e=d=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}d.key==="Escape"&&se(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const fe="app-modal";let he=null;function De(t,a){oe();const n=document.createElement("div");n.className="modal-overlay",n.id=fe,n.innerHTML=`<div class="modal">${t}</div>`,n.addEventListener("click",d=>{const u=d.target.closest("[data-modal-action]");u!=null&&u.dataset.modalAction?a(u.dataset.modalAction):d.target===n&&a("cancel")});const e=d=>{d.key==="Escape"&&a("cancel")};document.addEventListener("keydown",e),he=e,document.body.appendChild(n)}function oe(){var t;(t=document.getElementById(fe))==null||t.remove(),he&&(document.removeEventListener("keydown",he),he=null)}function it(){return document.querySelector(`#${fe} .modal`)}function ct(t){return new Promise(a=>{var d;let n=!1;const e=u=>{n||(n=!0,oe(),a(u))};De(`
        <h2>${r(t.title)}</h2>
        <p>${r(t.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm">${r(t.confirmLabel)}</button>
        </div>
      `,u=>e(u==="confirm")),(d=document.querySelector(`#${fe} [data-modal-action="confirm"]`))==null||d.focus()})}const lt=85,ve={exec:"Execution",beacon:"Beacon"};function dt(t,a){let n=!1,e=null,d=null,u=null,b=null,f=null,S=null,H=null,R=null;const B={exec:null,beacon:null};let w=null;t.innerHTML=`<h1>Dashboard: ${r(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${M()}</div>`;const P=t.querySelector("#dash-body"),m=t.querySelector("#dash-footer");P.addEventListener("click",c=>{const v=c.target.closest("[data-action]");if(!v||!P.contains(v))return;const y=v.dataset.action;if(y==="svc-action"){const $=v.dataset.svc,A=v.dataset.kind;$&&A&&me($,A)}else if(y==="open-clear"){const $=v.dataset.svc;$&&ce($)}else if(y==="copy"){const $=v.dataset.copy;$&&ge(v,$)}else y==="retry-du"?o():y==="retry-endpoints"&&l()}),p();async function p(){let c,v;try{const[$,A]=await Promise.all([te(),ee()]);c=$.find(s=>s.id===a),v=A}catch($){if(n)return;P.innerHTML=`<p class="error">Failed to load target: ${r(String($))}</p>`;return}if(n)return;if(!c){P.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const y=v==null?void 0:v.networks.find($=>$.ChainID===c.wire.ChainID);y&&(m.innerHTML=M(y.Name,y.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=We(a,$=>{n||(g($),d=$,u=$,k())}),o(),l()}async function o(){S=null;try{f=await Ze(a)}catch(c){f=null,S=String(c instanceof Error?c.message:c)}n||k()}async function l(){R=null;try{H=await Xe(a)}catch(c){H=null,R=String(c instanceof Error?c.message:c)}n||k()}function g(c){if(!d)return;const v=(new Date(c.at).getTime()-new Date(d.at).getTime())/1e3,y=c.execHead-d.execHead;if(v>0&&y>=0){const $=y/v;b=b===null?$:b*.7+$*.3}}function k(){if(!u)return;const c=u;P.innerHTML=`
      <p class="dash-status">${N(c)}</p>
      <div class="card-grid">
        ${O(c)}
        ${K(c)}
        ${G(c)}
        ${X(c)}
        ${Q(c)}
        ${ae()}
      </div>
      <p class="muted small">Last updated ${r(new Date(c.at).toLocaleTimeString())}</p>
    `}function N(c){return!c.execActive&&!c.beaconActive?C("Node not running","bad"):c.execSyncing||c.beaconDistance>0?C("Syncing","warn"):C("Running · synced","ok")}function U(c){const y=c.refHead>0?c.refHead-c.execHead:null,$=y!==null&&y>0&&b&&b>0?ot(y/b):y!==null&&y<=0?"caught up":"—";return{lag:y,eta:$}}function K(c){const{lag:v,eta:y}=U(c);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${c.execActive?c.execSyncing?C("syncing","warn"):c.execHead===0?C("no data","neutral"):C("synced","ok"):C("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${Y(c.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${v!==null?Y(c.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${v!==null?Y(Math.max(v,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${y}</dd></div>
        </dl>
      </div>
    `}function G(c){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${c.beaconActive?c.beaconSlot===0?C("no data","neutral"):c.beaconDistance===0?C("synced","ok"):C("syncing","warn"):C("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${Y(c.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${Y(c.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function X(c){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${Y(c.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${Y(c.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Q(c){const v=c.diskUsedPct>=lt,y=`
      <div class="meter"><div class="meter-fill ${v?"meter-warn":""}" style="width:${Math.min(c.diskUsedPct,100)}%"></div></div>
      <p>${st(c.diskUsedPct)} used</p>
    `;if(S)return`
        <div class="card ${v?"card-warn":""}">
          <h3>Storage</h3>
          ${y}
          <p class="error small">${r(S)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${v?"card-warn":""}">
          <h3>Storage</h3>
          ${y}
          <p class="muted">Loading…</p>
        </div>
      `;const $=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,A=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:s,eta:i}=U(c),h=s!==null&&s>0&&b!==null&&b>0;return`
      <div class="card ${v?"card-warn":""}">
        <h3>Storage</h3>
        ${y}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Z(f.ExecBytes)} of ~${Z(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${$}%"></div></div>
        ${h?`<p class="muted small">Estimated time remaining: ${r(i)}</p>`:""}
        <p class="muted small">Beacon — ${Z(f.BeaconBytes)} of ~${Z(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${A}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Z(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function ae(){if(R)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(R)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!H)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const c=H,v=c.ExecReachable&&!c.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",y=c.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${r(c.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${r(c.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${Le(c.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${r(c.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(c.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${Le(c.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${r(c.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(c.BeaconHTTP)}">Copy</button>
        </div>
        ${v}
        ${y}
      </div>
    `}function W(c,v){const y=ve[c],$=B[c],A=(s,i,h)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${c}" data-kind="${s}" ${$!==null||h?"disabled":""}>${$===s?j():r(i)}</button>`;return`
      <div class="service-row">
        <span>${r(y)} ${v?C("active","ok"):C("down","bad")}</span>
        <div class="service-actions">
          ${A("start","Start",v)}
          ${A("stop","Stop",!v)}
          ${A("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${c}" ${$!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function O(c){return`
      <div class="card">
        <h3>Services</h3>
        ${W("exec",c.execActive)}
        ${W("beacon",c.beaconActive)}
        ${w?`<p class="error small">${r(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(a)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(a)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(a)}">Diagnostics →</a>
        </p>
      </div>
    `}function j(){return'<span class="spinner" aria-label="working"></span>'}async function me(c,v){if(B[c]===null){B[c]=v,w=null,k();try{await Ye(a,c,v)}catch(y){w=`${ve[c]} ${v} failed: ${y instanceof Error?y.message:String(y)}`}B[c]=null,n||k()}}async function ge(c,v){const y=await we(v),$=c.textContent;c.textContent=y?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=$)},1500)}function ce(c){const v=ve[c],y=f?Z(c==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";De(`
        <h2>Clear ${r(v)} data</h2>
        <p class="error">
          This stops the ${r(v.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${r(y)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${r(c)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,s=>{if(s==="cancel"){oe();return}s==="confirm"&&re(c)});const $=document.getElementById("clear-confirm-input"),A=document.getElementById("clear-confirm-btn");$==null||$.addEventListener("input",()=>{A&&(A.disabled=$.value.trim()!==c)}),$==null||$.focus()}async function re(c){const v=document.getElementById("clear-confirm-btn");v&&(v.disabled=!0,v.textContent="Clearing…");try{await Ve(a,c),oe(),o()}catch(y){const $=it();if($){const A=document.createElement("p");A.className="error small",A.textContent=`Clear failed: ${y instanceof Error?y.message:String(y)}`,$.appendChild(A)}v&&(v.disabled=!1,v.textContent="Clear and resync")}}return()=>{n=!0,e==null||e(),oe()}}const Re=500,He="valve-node-app.explain-consent";function ut(t,a){let n=!1,e=null;const d=[];t.innerHTML=`
    <h1>Logs: ${r(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${M()}</div>
  `;const u=t.querySelector("#logs-body"),b=t.querySelector("#logs-footer");ne(t,p=>{p==="explain"&&R()}),f();async function f(){let p,o;try{const[g,k]=await Promise.all([te(),ee()]);p=g.find(N=>N.id===a),o=k}catch(g){if(n)return;u.innerHTML=`<p class="error">Failed to load target: ${r(String(g))}</p>`;return}if(n)return;if(!p){u.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const l=o==null?void 0:o.networks.find(g=>g.ChainID===p.wire.ChainID);l&&(b.innerHTML=M(l.Name,l.LearnURL));try{const g=await Je(a,200);if(n)return;d.push(...g)}catch(g){if(n)return;u.innerHTML=`<p class="error">Failed to load logs: ${r(String(g))}</p>`;return}S(),e=Ge(a,g=>{n||(d.push(g),d.length>Re&&d.splice(0,d.length-Re),S())})}function S(){const p=d.filter(l=>l.severity==="error"||l.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(H).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${C(String(p.length),p.length?"bad":"neutral")}</h2>
          <div class="log-lines">${p.length?p.slice().reverse().map(H).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=u.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function H(p){const o=p.severity||"info",l=p.learnUrl?` <a href="${r(p.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(p.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(p.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(p.line)}</span>
        ${p.explain?`<div class="log-explain">${r(p.explain)}${l}</div>`:""}
      </div>
    `}async function R(){const p=d.filter(l=>l.severity==="error"||l.severity==="critical").map(l=>l.line).slice(-40);if(!(localStorage.getItem(He)==="1")){B(p);return}await w(p)}function B(p){const o=p.length?`<pre class="explain-excerpt">${p.map(l=>r(l)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';P(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${o}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,l=>{l==="proceed"?(localStorage.setItem(He,"1"),m(),w(p)):m()})}async function w(p){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=p.length?await Ee(a,p):await Ee(a);if(n)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(l=>r(l)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&m()})}catch(o){if(n)return;if(o instanceof $e&&o.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,l=>{l==="close"&&m()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&m()})}}function P(p,o){m();const l=document.createElement("div");l.className="modal-overlay",l.id="explain-modal",l.innerHTML=`<div class="modal">${p}</div>`,l.addEventListener("click",g=>{const k=g.target.closest("[data-modal-action]");k!=null&&k.dataset.modalAction&&o(k.dataset.modalAction),g.target===l&&o("cancel")}),document.body.appendChild(l)}function m(){var p;(p=document.getElementById("explain-modal"))==null||p.remove()}return()=>{n=!0,e==null||e(),m()}}function pt(t,a){let n=!1,e=null,d=null,u=!1,b=!1;t.innerHTML=`<h1>Network diagnostics: ${r(a)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${M()}</div>`;const f=t.querySelector("#diag-body"),S=t.querySelector("#diag-footer");ne(t,(o,l)=>{var g;if(o==="run")R();else if(o==="toggle")(g=l.closest(".check-item"))==null||g.classList.toggle("expanded");else if(o==="copy"){const k=l.dataset.copy;k&&p(l,k)}}),H();async function H(){let o,l;try{const[k,N]=await Promise.all([te(),ee()]);o=k.find(U=>U.id===a),l=N}catch(k){if(n)return;f.innerHTML=`<p class="error">Failed to load target: ${r(String(k))}</p>`;return}if(n)return;if(!o){f.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const g=l==null?void 0:l.networks.find(k=>k.ChainID===o.wire.ChainID);g&&(S.innerHTML=M(g.Name,g.LearnURL));try{e=await tt(a),b=!0}catch(k){d=String(k instanceof Error?k.message:k)}n||B()}async function R(){u=!0,d=null,B();try{e=await et(a),b=!0}catch(o){d=String(o instanceof Error?o.message:o)}u=!1,n||B()}function B(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${u?"disabled":""}>${u?"Running…":"Run diagnostics"}</button>
      </div>
      ${d?`<p class="error">${r(d)}</p>`:""}
      ${w()}
    `}function w(){if(!b&&!d)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),l=e.failedId?`<p><strong>Failed at: ${r(P(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${l}
      <ul class="check-list">${e.items.map(m).join("")}</ul>
    `}function P(o){var l;return((l=e==null?void 0:e.items.find(g=>g.ID===o))==null?void 0:l.Title)??o}function m(o){const l=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",g=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${g?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${C(g?"failed here":o.Status,l)}
          <strong>${r(o.Title)}</strong>
          <span class="muted small check-detail-inline">${r(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${g?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${r(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${g?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function p(o,l){const g=await we(l),k=o.textContent;o.textContent=g?"Copied!":"Copy failed",setTimeout(()=>{n||(o.textContent=k)},1500)}return()=>{n=!0}}function ht(t,a){let n=!1,e=[],d=null,u=!1,b=!1;t.innerHTML=`<h1>Security: ${r(a)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${M()}</div>`;const f=t.querySelector("#sec-body"),S=t.querySelector("#sec-footer");ne(t,(m,p)=>{var o;if(m==="rerun")R();else if(m==="toggle")(o=p.closest(".check-item"))==null||o.classList.toggle("expanded");else if(m==="copy"){const l=p.dataset.copy;l&&P(p,l)}}),H();async function H(){let m,p;try{const[l,g]=await Promise.all([te(),ee()]);m=l.find(k=>k.id===a),p=g}catch(l){if(n)return;f.innerHTML=`<p class="error">Failed to load target: ${r(String(l))}</p>`;return}if(n)return;if(!m){f.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!m.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const o=p==null?void 0:p.networks.find(l=>l.ChainID===m.wire.ChainID);o&&(S.innerHTML=M(o.Name,o.LearnURL)),await R()}async function R(){u=!0,d=null,B();try{e=await Qe(a),b=!0}catch(m){d=String(m instanceof Error?m.message:m)}u=!1,n||B()}function B(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${r(d)}</p>`:""}
      ${!b&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:b?'<p class="muted">No checks returned.</p>':""}
    `}function w(m){const p=m.Status==="pass"?"ok":m.Status==="fail"?"bad":m.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${C(m.Status,p)}
          <strong>${r(m.Title)}</strong>
          <span class="muted small check-detail-inline">${r(m.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(m.Why)}</p>
          </details>
          ${m.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(m.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(m.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(m,p){const o=await we(p),l=m.textContent;m.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{n||(m.textContent=l)},1500)}return()=>{n=!0}}const ft=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function mt(t){let a=!1,n=!1,e=!1,d=null,u=!1,b=null,f=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${M()}`;const S=t.querySelector("#settings-body");ne(t,w=>{if(w==="save"&&B(),w==="clear-key"){if(!b)return;n=!0;const P=t.querySelector("#ai-key");P&&(P.value=""),R(b)}}),Ae(t,(w,P)=>{w!=="ai-provider"||!b||(f=P,u=!1,R(b))}),H();async function H(){try{const w=await nt();if(a)return;b=w,R(w)}catch(w){if(a)return;S.innerHTML=`<p class="error">Failed to load settings: ${r(String(w))}</p>`}}function R(w){var p;const P=f??w.aiProvider;S.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${ye("ai-provider",ft.map(o=>({value:o.value,label:o.label})),P)}
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
            <input id="ref-rpc-base" type="text" value="${r(w.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${r(d)}</p>`:""}
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const m=t.querySelector("#ai-key");m==null||m.addEventListener("input",()=>{n=!0,u=!1}),(p=t.querySelector("#ref-rpc-base"))==null||p.addEventListener("input",()=>{u=!1})}async function B(){const w=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!w||!P||!b)return;const m={aiProvider:f??b.aiProvider,refRpcBase:P.value.trim()};n&&(m.aiKey=w.value),e=!0,d=null,u=!1,R(b);try{const p=await at(m);if(a)return;b=p,n=!1,e=!1,u=!0,R(p)}catch(p){if(a)return;e=!1,d=String(p instanceof Error?p.message:p),R(b)}}return()=>{a=!0}}const gt="local";function vt(t){let a=!1,n=!1,e="",d=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${M()}
  `;const u=t.querySelector("#targets-body");ne(t,(o,l)=>{R(o,l)}),b();async function b(){try{const[o,l,g]=await Promise.all([te(),ee(),Oe()]);if(a)return;e=g.os,S(o,l)}catch(o){if(a)return;u.innerHTML=`<p class="error">Failed to load machines: ${r(String(o))}</p>`}}function f(){d&&S(d.targets,d.catalog)}function S(o,l){d={targets:o,catalog:l};const g=e==="linux",k=[...o].sort((U,K)=>(U.mode==="local"?-1:0)-(K.mode==="local"?-1:0)),N=k.length?`<div class="card-grid">${k.map(U=>bt(U,l)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>';u.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${N}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${H(g)}
        ${n?yt():""}
      </section>
    `}function H(o){const l=`
      <div class="card">
        <h3>A server over SSH ${C("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${o?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${o?" btn-ghost":""}" data-action="toggle-ssh">
            ${n?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,g=o?`
        <div class="card">
          <h3>This machine ${C("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${r(e)})`:""} ${C("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return`<div class="card-grid">${o?g+l:l+g}</div>`}async function R(o,l){var g;if(o==="add-local"){await B();return}if(o==="delete-target"){const k=l.dataset.id;if(!k||!await ct({title:"Remove machine",body:`Remove "${k}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove"}))return;await w(k);return}if(o==="toggle-ssh"){n=!n,p(),f(),n&&((g=t.querySelector("#ssh-host"))==null||g.focus());return}o==="add-ssh"&&await P()}async function B(){p();try{await Se({id:gt,mode:"local"}),await b()}catch(o){m(o)}}async function w(o){try{await _e(o),await b()}catch(l){m(l)}}async function P(){const o=t.querySelector("#ssh-host"),l=t.querySelector("#ssh-user"),g=t.querySelector("#ssh-key"),k=t.querySelector("#ssh-port"),N=t.querySelector("#ssh-id");if(!o||!l||!g||!k||!N)return;const U=o.value.trim(),K=l.value.trim(),G=g.value.trim(),X=k.value.trim(),Q=N.value.trim();if(p(),!U||!K||!G){m(new Error("host, user, and key path are required"));return}const ae=Q||$t(U),W={Host:U,User:K,KeyPath:G};if(X){const j=Number.parseInt(X,10);if(!Number.isFinite(j)||j<=0){m(new Error("port must be a positive number"));return}W.Port=j}const O=t.querySelector("#ssh-submit");O&&(O.disabled=!0,O.textContent="Connecting…");try{await Se({id:ae,mode:"ssh",ssh:W}),n=!1,await b()}catch(j){m(j),O&&(O.disabled=!1,O.textContent="Add server")}}function m(o){let l=t.querySelector("#targets-error");l||(u.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),l=t.querySelector("#targets-error")),l.textContent=String(o instanceof Error?o.message:o)}function p(){var o;(o=t.querySelector("#targets-error"))==null||o.remove()}return()=>{a=!0}}function bt(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",d=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let u,b;if(!n)u=C("not set up","neutral"),b=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const f=a.networks.find(H=>H.ChainID===n.ChainID),S=f?f.Name:`chain ${n.ChainID}`;u=`${C(S,"ok")} ${C(n.ExecID,"neutral")} ${C(n.BeaconID,"neutral")}${n.Archive?" "+C("archive","warn"):""}`,b=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${d}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${b}
        <button class="btn btn-danger" data-action="delete-target" data-id="${r(t.id)}">Remove</button>
      </div>
    </div>
  `}function yt(){return`
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
  `}function $t(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const be=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],de=8545,ue=5052,pe=30303,wt=[369,943,1],Be={369:"default",943:"practise here first"};function kt(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${M()}</div>`;const d=t.querySelector("#wizard-body"),u=t.querySelector("#wizard-footer");ne(t,(s,i)=>{ge(s,i)}),Ae(t,(s,i)=>{s==="exec-select"?e.execId=i:s==="beacon-select"&&(e.beaconId=i),f()}),t.addEventListener("change",s=>{const i=s.target;i instanceof HTMLInputElement&&(i.id==="data-dir-input"?(ce(),G()):i.id==="checkpoint-toggle"?(e.checkpoint=i.checked,f()):i.id==="exec-snapshot-toggle"&&(e.execSnapshot=i.checked,f()))}),b();async function b(){try{const[s,i]=await Promise.all([ee(),te()]);if(n)return;e.catalog=s;const h=i.find(x=>x.id===a);h!=null&&h.wire&&(e.chainId=h.wire.ChainID,e.execId=h.wire.ExecID,e.beaconId=h.wire.BeaconID,e.archive=h.wire.Archive,h.wire.ExecHTTPPort&&(e.execHTTPPort=String(h.wire.ExecHTTPPort)),h.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(h.wire.BeaconHTTPPort)),h.wire.ExecP2PPort&&(e.execP2PPort=String(h.wire.ExecP2PPort)),h.wire.RPCBindAddr&&(e.rpcBindAddr=h.wire.RPCBindAddr)),f()}catch(s){if(n)return;e.loadError=String(s instanceof Error?s.message:s),f()}}function f(){if(e.loadError){d.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(d.innerHTML=`
      ${A(e.step)}
      ${H()}
    `,S())}function S(){var i;const s=(i=e.catalog)==null?void 0:i.networks.find(h=>h.ChainID===e.chainId);u.innerHTML=s?M(s.Name,s.LearnURL):M()}function H(){switch(e.step){case"network":return R();case"clients":return B();case"mode":return O();case"review":return j();case"run":return me()}}function R(){const s=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${wt.map(h=>{const x=s.networks.find(L=>L.ChainID===h);if(!x)return"";const E=e.chainId===h,I=Be[h]?C(Be[h],h===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${E?"selected":""}" data-action="pick-network" data-chain-id="${h}" type="button">
          <h3>${r(x.Name)} <span class="muted">(chain ${h})</span></h3>
          ${I}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function B(){const s=e.catalog,i=s.networks.find(E=>E.ChainID===e.chainId);if(!i)return'<p class="error">Unknown network.</p>';(e.execId===null||!i.ExecClients.includes(e.execId))&&(e.execId=i.ExecClients[0]??null),(e.beaconId===null||!i.BeaconClients.includes(e.beaconId))&&(e.beaconId=i.BeaconClients[0]??null);const h=i.ExecClients.map(E=>Q(E,s)),x=i.BeaconClients.map(E=>Q(E,s));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(i.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${ye("exec-select",h,e.execId)}
        </label>
        ${W(e.execId,s)}
        <label>
          Beacon client
          ${ye("beacon-select",x,e.beaconId)}
        </label>
        ${W(e.beaconId,s)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(s){return s<=0?"—":s>=1?`~${s.toFixed(1)} TB`:`~${Math.round(s*1e3)} GB`}const P=1.1,m=.5,p="Valve reth snapshot",o="rough estimate";function l(s){return s.SnapshotSizeTB}function g(s){return s.SnapshotSizeTB*m}function k(s){return`<p class="muted small">${w(l(s))} is the measured size of Valve's reth snapshot for ${r(s.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function N(s){return{archive:l(s)*1e12*P,full:g(s)*1e12*P}}function U(s,i){if(!s)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${r(i)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${r(i)}</code>: ${r(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==i)return"";const h=N(s),x=e.freeBytes>=h.archive,E=e.freeBytes>=h.full,I=`<p class="muted small">Free at <code>${r(i)}</code>: <strong>${Z(e.freeBytes)}</strong> — archive ${x?"fits":"won't fit"} (${w(l(s))}, ${p}), full ${E?"fits":"won't fit"} (${w(g(s))}, ${o}).</p>`;let L="";return e.downgradeNote?L=`<p class="banner banner-warn">${r(e.downgradeNote)}</p>`:E||(L=`<p class="banner banner-warn">Neither full (${w(g(s))}, ${o}) nor archive (${w(l(s))}, ${p}) fits the free space here — choose a location with more room.</p>`),I+L}function K(s,i){if(e.downgradeNote=null,!s||e.freeBytes===null)return;const h=N(s);e.archive&&e.freeBytes<h.archive&&e.freeBytes>=h.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${i} for archive (${w(l(s))}, ${p}) — switched to Full (${w(g(s))}, ${o}). Pick a location with more room to run archive.`)}async function G(){var h;if(e.chainId===null)return;const s=(h=e.catalog)==null?void 0:h.networks.find(x=>x.ChainID===e.chainId),i=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,f();try{const{freeBytes:x}=await je(e.targetId,i);if(n)return;e.freeBytes=x,e.probedPath=i,K(s,i)}catch(x){if(n)return;e.freeBytes=null,e.probedPath=i,e.diskError=String(x instanceof Error?x.message:x)}e.diskProbing=!1,f()}function X(s){return s?/^https?:\/\/.+/i.test(s)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function Q(s,i){const h=i.clients.find(x=>x.id===s);return{value:s,label:h?`${h.id} — ${ae(h.repo)}`:s}}function ae(s){const i=s.split("/");return i.length>=4?i[3]:s}function W(s,i){const h=s?i.clients.find(E=>E.id===s):void 0;if(!h)return"";const x=h.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${r(h.repo)}" target="_blank" rel="noopener noreferrer">${r(x)}</a></p>`}function O(){var q,z,T;const s=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",i=(q=e.catalog)==null?void 0:q.networks.find(F=>F.ChainID===e.chainId),h=((T=(z=e.catalog)==null?void 0:z.clients.find(F=>F.id===e.execId))==null?void 0:T.snapshotSupported)??!1,x=i?`${w(g(i))} (${o})`:"Smaller",E=i?`${w(l(i))} (${p})`:"Much larger",I=i?` on ${r(i.Name)}`:"",L=i?e.checkpoint?i.SyncLabel:i.GenesisSyncLabel:"";return`
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
          ${i?`<p class="sync-estimate">⏱ Estimated initial sync${I}: <strong>${r(L)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${r((i==null?void 0:i.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${r((i==null?void 0:i.CheckpointURL)??"")}" value="${r(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${r(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${h?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${e.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${e.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${r(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${r(e.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${I}</th><td class="yes">${x}</td><td class="limited">${E}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${i?k(i):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${E}${i?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${x}${i?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${r(s)})</span>
            <input id="data-dir-input" type="text" placeholder="${r(s)}" value="${r(e.dataDir)}" />
          </label>
          ${U(i,e.dataDir||s)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${r(s)}/jwt.hex" value="${r(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${de})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${de}" value="${r(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${r(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${ue})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ue}" value="${r(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${r(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${pe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${pe}" value="${r(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${r(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${r(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${r(e.rpcBindAddrError)}</p>`:""}
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
    `}function j(){const i=e.catalog.networks.find(J=>J.ChainID===e.chainId),h=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,x=e.jwtPath||`${h}/jwt.hex`,E=be.map(J=>`<li>${r(J.title)}</li>`).join(""),I=y(e.execHTTPPort,de),L=y(e.beaconHTTPPort,ue),q=y(e.execP2PPort,pe),z=I||L||q?`<tr><th>Non-default ports</th><td>${[I?`exec HTTP ${I}`:null,L?`beacon HTTP ${L}`:null,q?`exec p2p ${q}`:null].filter(J=>J!==null).map(r).join(", ")}</td></tr>`:"",{addr:T}=re(e.rpcBindAddr),F=T?`<tr><th>RPC bind address</th><td><code>${r(T)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((i==null?void 0:i.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(h)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(x)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${r(e.checkpointUrl||(i==null?void 0:i.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${z}
            ${F}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${E}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function me(){const i=e.catalog.networks.find(T=>T.ChainID===e.chainId),h=i==null?void 0:i.LearnURL,x=new Set(e.events.filter(T=>T.done).map(T=>T.stepId)),E=new Set(e.events.filter(T=>T.err).map(T=>T.stepId)),I=new Map;for(const T of e.events){if(!T.line)continue;const F=I.get(T.stepId)??[];F.push(T.line),I.set(T.stepId,F)}const L=be.map(T=>{var Te;const F=x.has(T.id),J=E.has(T.id),Ue=J?C("failed","bad"):F?C("done","ok"):C("pending","neutral"),ke=(I.get(T.id)??[]).slice(-5),xe=(Te=e.events.find(le=>le.stepId===T.id&&le.err))==null?void 0:Te.err,Me=T.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${h?` <a href="${r(h)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${F?"step-done":""} ${J?"step-error":""}">
          <div class="step-head">${Ue} <strong>${r(T.title)}</strong></div>
          ${Me}
          ${ke.length?`<pre class="step-log">${ke.map(le=>r(le)).join(`
`)}</pre>`:""}
          ${xe?`<p class="error small">${r(xe)}</p>`:""}
        </li>
      `}).join(""),q=e.events.some(T=>T.err),z=be.every(T=>x.has(T.id))||e.events.some(T=>T.stepId==="handshake"&&T.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${L}</ol>
        ${z&&!q?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        ${q?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function ge(s,i){switch(s){case"pick-network":e.chainId=Number(i.dataset.chainId),e.execId=null,e.beaconId=null,f();break;case"goto-network":e.step="network",f();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",f();break;case"goto-mode":e.step="mode",f(),G();break;case"goto-review":if(ce(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){f();break}e.step="review",f();break;case"start-setup":$();break}}function ce(){const s=t.querySelectorAll('input[name="mode"]');for(const T of Array.from(s))T.checked&&(e.archive=T.value==="archive");const i=t.querySelector("#data-dir-input"),h=t.querySelector("#jwt-path-input");i&&(e.dataDir=i.value.trim()),h&&(e.jwtPath=h.value.trim());const x=t.querySelector("#exec-http-port-input"),E=t.querySelector("#beacon-http-port-input"),I=t.querySelector("#exec-p2p-port-input");x&&(e.execHTTPPort=x.value.trim()),E&&(e.beaconHTTPPort=E.value.trim()),I&&(e.execP2PPort=I.value.trim());const L=t.querySelector("#rpc-bind-addr-input");L&&(e.rpcBindAddr=L.value.trim());const q=t.querySelector("#checkpoint-url-input");q&&(e.checkpointUrl=q.value.trim());const z=t.querySelector("#snapshot-key-input");z&&(e.snapshotKey=z.value.trim()),e.execHTTPPortError=v(e.execHTTPPort).error??null,e.beaconHTTPPortError=v(e.beaconHTTPPort).error??null,e.execP2PPortError=v(e.execP2PPort).error??null,e.rpcBindAddrError=re(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?X(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function re(s){if(!s)return{};const i=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(s);return i?i.slice(1).every(h=>Number(h)<=255)?{addr:s}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(s)&&s.includes(":")?{addr:s}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const c=/^\d+$/;function v(s){if(!s)return{};if(!c.test(s))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const i=Number(s);return!Number.isInteger(i)||i<1||i>65535?{error:"Port must be between 1 and 65535."}:{port:i}}function y(s,i){const{port:h}=v(s);if(!(h===void 0||h===i))return h}async function $(){var I;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(I=e.streamStop)==null||I.call(e),e.streamStop=null,f();const s={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(s.DataDir=e.dataDir),e.jwtPath&&(s.JWTPath=e.jwtPath);const i=y(e.execHTTPPort,de),h=y(e.beaconHTTPPort,ue),x=y(e.execP2PPort,pe);i!==void 0&&(s.ExecHTTPPort=i),h!==void 0&&(s.BeaconHTTPPort=h),x!==void 0&&(s.ExecP2PPort=x);const{addr:E}=re(e.rpcBindAddr);E!==void 0&&(s.RPCBindAddr=E),e.checkpoint?e.checkpointUrl&&(s.CheckpointURL=e.checkpointUrl):s.NoCheckpoint=!0,e.execSnapshot&&(s.ExecSnapshot=!0,s.SnapshotKey=e.snapshotKey);try{await ze(e.targetId,s)}catch(L){if(!(L instanceof $e&&L.status===409)){e.starting=!1,e.startError=String(L instanceof Error?L.message:L),f();return}}e.starting=!1,e.step="run",f(),e.streamStop=Ke(e.targetId,L=>{n||(e.events.push(L),e.step==="run"&&f())})}function A(s){const i=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],x=i.map(E=>E.id).indexOf(s);return`
      <ol class="wizard-progress">
        ${i.map((E,I)=>`<li class="${I===x?"current":I<x?"past":"future"}">${r(E.label)}</li>`).join("")}
      </ol>
    `}return()=>{var s;n=!0,(s=e.streamStop)==null||s.call(e)}}const xt=document.querySelector("#app"),{contentEl:Tt,setActiveNav:Pt}=rt(xt);let _=null;function St(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function V(t){const a=document.createElement("div");return Tt.replaceChildren(a),t(a)}function Ne(){if(_){try{_()}catch{}_=null}const{screen:t,id:a}=St();switch(Pt(t),t){case"setup":if(!a){location.hash="#/targets";return}_=V(n=>kt(n,a));break;case"dash":if(!a){location.hash="#/targets";return}_=V(n=>dt(n,a));break;case"logs":if(!a){location.hash="#/targets";return}_=V(n=>ut(n,a));break;case"security":if(!a){location.hash="#/targets";return}_=V(n=>ht(n,a));break;case"diag":if(!a){location.hash="#/targets";return}_=V(n=>pt(n,a));break;case"settings":_=V(n=>mt(n));break;case"targets":default:_=V(n=>vt(n));break}}window.addEventListener("hashchange",Ne);Ne();
