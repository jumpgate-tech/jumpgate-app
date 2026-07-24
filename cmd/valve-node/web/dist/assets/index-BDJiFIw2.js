var we=Object.defineProperty;var xe=(t,a,n)=>a in t?we(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var de=(t,a,n)=>xe(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const d of u)if(d.type==="childList")for(const y of d.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(u){const d={};return u.integrity&&(d.integrity=u.integrity),u.referrerPolicy&&(d.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?d.credentials="include":u.crossOrigin==="anonymous"?d.credentials="omit":d.credentials="same-origin",d}function e(u){if(u.ep)return;u.ep=!0;const d=n(u);fetch(u.href,d)}})();function G(){return D("/api/catalog")}function Y(){return D("/api/targets")}function ue(t){return D("/api/targets",{method:"POST",headers:te,body:JSON.stringify(t)})}function Pe(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,a){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:te,body:JSON.stringify(a)})}function ke(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Ee(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Se(t,a=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function Ce(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function pe(t,a){const n=a===void 0?{}:{lines:a};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:te,body:JSON.stringify(n)})}function Le(t,a,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${a}/${n}`,{method:"POST"})}function Ie(t,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${a}/clear`,{method:"POST",headers:te,body:JSON.stringify({Confirm:a})})}function He(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Re(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Be(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Ae(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function De(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Me(){return D("/api/settings")}function Ne(t){return D("/api/settings",{method:"PUT",headers:te,body:JSON.stringify(t)})}class ce extends Error{constructor(n,e){super(e);de(this,"status");this.name="ApiError",this.status=n}}const te={"Content-Type":"application/json"};async function D(t,a){const n=await fetch(t,a);if(!n.ok){let u=n.statusText||`HTTP ${n.status}`;try{const d=await n.json();d&&typeof d.error=="string"&&d.error&&(u=d.error)}catch{}throw new ce(n.status,u)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const Ue="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function q(t,a){const n=t&&a?` <span class="footer-sep">·</span> <a href="${r(a)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(Ue)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function qe(t){t.innerHTML=`
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
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:u=>{for(const d of n)d.classList.toggle("active",d.dataset.nav===u)}}}function z(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Oe(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Fe(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const u=Math.floor(n/24),d=n%24;return`~${u}d ${d}h`}function A(t,a){return`<span class="badge badge-${a}">${r(t)}</span>`}function he(t){return`<span class="dot dot-${t}"></span>`}const fe=["B","KB","MB","GB","TB","PB"];function K(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let a=t,n=0;for(;a>=1024&&n<fe.length-1;)a/=1024,n++;const e=a<10?2:a<100?1:0;return`${a.toFixed(e)} ${fe[n]}`}async function le(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function V(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const u=e.dataset.action;u&&a(u,e,n)})}function ie(t,a,n){const e=a.find(d=>d.value===n),u=a.map(d=>`
      <li class="dropdown-option${d.value===n?" selected":""}" role="option"
          aria-selected="${d.value===n}" data-value="${r(d.value)}">
        ${r(d.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${r(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${r(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${u}</ul>
    </div>
  `}function ee(t){t.querySelectorAll(".dropdown.open").forEach(a=>{var n;a.classList.remove("open"),(n=a.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function be(t,a){t.addEventListener("click",u=>{const d=u.target,y=d.closest(".dropdown-trigger");if(y&&t.contains(y)){const k=y.closest(".dropdown"),H=!!k&&!k.classList.contains("open");ee(t),k&&H&&(k.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const m=d.closest(".dropdown-option");if(m&&t.contains(m)){const k=m.closest(".dropdown");ee(t),a((k==null?void 0:k.dataset.dropdown)??"",m.dataset.value??"");return}ee(t)});const n=u=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const d=u.target;(!d.closest(".dropdown")||!t.contains(d))&&ee(t)},e=u=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}u.key==="Escape"&&ee(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const je=85,se={exec:"Execution",beacon:"Beacon"};function _e(t,a){let n=!1,e=null,u=null,d=null,y=null,m=null,k=null,H=null,I=null;const B={exec:null,beacon:null};let x=null;t.innerHTML=`<h1>Dashboard: ${r(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${q()}</div>`;const P=t.querySelector("#dash-body"),c=t.querySelector("#dash-footer");P.addEventListener("click",s=>{const g=s.target.closest("[data-action]");if(!g||!P.contains(g))return;const v=g.dataset.action;if(v==="svc-action"){const b=g.dataset.svc,L=g.dataset.kind;b&&L&&R(b,L)}else if(v==="open-clear"){const b=g.dataset.svc;b&&U(b)}else if(v==="copy"){const b=g.dataset.copy;b&&C(g,b)}else v==="retry-du"?o():v==="retry-endpoints"&&l()}),i();async function i(){let s,g;try{const[b,L]=await Promise.all([Y(),G()]);s=b.find(F=>F.id===a),g=L}catch(b){if(n)return;P.innerHTML=`<p class="error">Failed to load target: ${r(String(b))}</p>`;return}if(n)return;if(!s){P.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const v=g==null?void 0:g.networks.find(b=>b.ChainID===s.wire.ChainID);v&&(c.innerHTML=q(v.Name,v.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=Ee(a,b=>{n||($(b),u=b,d=b,w())}),o(),l()}async function o(){k=null;try{m=await He(a)}catch(s){m=null,k=String(s instanceof Error?s.message:s)}n||w()}async function l(){I=null;try{H=await Re(a)}catch(s){H=null,I=String(s instanceof Error?s.message:s)}n||w()}function $(s){if(!u)return;const g=(new Date(s.at).getTime()-new Date(u.at).getTime())/1e3,v=s.execHead-u.execHead;if(g>0&&v>=0){const b=v/g;y=y===null?b:y*.7+b*.3}}function w(){if(!d)return;const s=d;P.innerHTML=`
      <div class="card-grid">
        ${N(s)}
        ${j(s)}
        ${X(s)}
        ${Z(s)}
        ${h(s)}
        ${p()}
        ${E(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function M(s){const v=s.refHead>0?s.refHead-s.execHead:null,b=v!==null&&v>0&&y&&y>0?Fe(v/y):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function N(s){const{lag:g,eta:v}=M(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?A("syncing","warn"):A("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${z(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${g!==null?z(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${g!==null?z(Math.max(g,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function j(s){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${s.beaconDistance===0?A("synced","ok"):A("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${z(s.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${z(s.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function X(s){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${z(s.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${z(s.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Z(s){const g=s.diskUsedPct>=je;return`
      <div class="card ${g?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${g?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Oe(s.diskUsedPct)} used</p>
      </div>
    `}function h(s){if(k)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(k)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const g=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,v=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:b,eta:L}=M(s),F=b!==null&&b>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${K(m.ExecBytes)} of ~${K(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        ${F?`<p class="muted small">Estimated time remaining: ${r(L)}</p>`:""}
        <p class="muted small">Beacon — ${K(m.BeaconBytes)} of ~${K(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${K(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function p(){if(I)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(I)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!H)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=H,g=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=s.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${r(s.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${he(s.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${he(s.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.BeaconHTTP)}">Copy</button>
        </div>
        ${g}
        ${v}
      </div>
    `}function f(s,g){const v=se[s],b=B[s],L=(F,J,$e)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${F}" ${b!==null||$e?"disabled":""}>${b===F?S():r(J)}</button>`;return`
      <div class="service-row">
        <span>${r(v)} ${g?A("active","ok"):A("down","bad")}</span>
        <div class="service-actions">
          ${L("start","Start",g)}
          ${L("stop","Stop",!g)}
          ${L("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function E(s){return`
      <div class="card">
        <h3>Services</h3>
        ${f("exec",s.execActive)}
        ${f("beacon",s.beaconActive)}
        ${x?`<p class="error small">${r(x)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(a)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(a)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(a)}">Diagnostics →</a>
        </p>
      </div>
    `}function S(){return'<span class="spinner" aria-label="working"></span>'}async function R(s,g){if(B[s]===null){B[s]=g,x=null,w();try{await Le(a,s,g)}catch(v){x=`${se[s]} ${g} failed: ${v instanceof Error?v.message:String(v)}`}B[s]=null,n||w()}}async function C(s,g){const v=await le(g),b=s.textContent;s.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{n||(s.textContent=b)},1500)}function U(s){const g=se[s],v=m?K(s==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";T(`
        <h2>Clear ${r(g)} data</h2>
        <p class="error">
          This stops the ${r(g.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${r(v)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${r(s)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,F=>{if(F==="cancel"){O();return}F==="confirm"&&Q(s)});const b=document.getElementById("clear-confirm-input"),L=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{L&&(L.disabled=b.value.trim()!==s)}),b==null||b.focus()}async function Q(s){const g=document.getElementById("clear-confirm-btn");g&&(g.disabled=!0,g.textContent="Clearing…");try{await Ie(a,s),O(),o()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const L=document.createElement("p");L.className="error small",L.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(L)}g&&(g.disabled=!1,g.textContent="Clear and resync")}}function T(s,g){O();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${s}</div>`,v.addEventListener("click",b=>{const L=b.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&g(L.dataset.modalAction),b.target===v&&g("cancel")}),document.body.appendChild(v)}function O(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{n=!0,e==null||e(),O()}}const me=500,ge="valve-node.explain-consent";function ze(t,a){let n=!1,e=null;const u=[];t.innerHTML=`
    <h1>Logs: ${r(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${q()}</div>
  `;const d=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");V(t,i=>{i==="explain"&&I()}),m();async function m(){let i,o;try{const[$,w]=await Promise.all([Y(),G()]);i=$.find(M=>M.id===a),o=w}catch($){if(n)return;d.innerHTML=`<p class="error">Failed to load target: ${r(String($))}</p>`;return}if(n)return;if(!i){d.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){d.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const l=o==null?void 0:o.networks.find($=>$.ChainID===i.wire.ChainID);l&&(y.innerHTML=q(l.Name,l.LearnURL));try{const $=await Se(a,200);if(n)return;u.push(...$)}catch($){if(n)return;d.innerHTML=`<p class="error">Failed to load logs: ${r(String($))}</p>`;return}k(),e=Ce(a,$=>{n||(u.push($),u.length>me&&u.splice(0,u.length-me),k())})}function k(){const i=u.filter(l=>l.severity==="error"||l.severity==="critical");d.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(H).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${A(String(i.length),i.length?"bad":"neutral")}</h2>
          <div class="log-lines">${i.length?i.slice().reverse().map(H).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=d.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function H(i){const o=i.severity||"info",l=i.learnUrl?` <a href="${r(i.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(i.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(i.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(i.line)}</span>
        ${i.explain?`<div class="log-explain">${r(i.explain)}${l}</div>`:""}
      </div>
    `}async function I(){const i=u.filter(l=>l.severity==="error"||l.severity==="critical").map(l=>l.line).slice(-40);if(!(localStorage.getItem(ge)==="1")){B(i);return}await x(i)}function B(i){const o=i.length?`<pre class="explain-excerpt">${i.map(l=>r(l)).join(`
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
    `,l=>{l==="proceed"?(localStorage.setItem(ge,"1"),c(),x(i)):c()})}async function x(i){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=i.length?await pe(a,i):await pe(a);if(n)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(l=>r(l)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&c()})}catch(o){if(n)return;if(o instanceof ce&&o.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,l=>{l==="close"&&c()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&c()})}}function P(i,o){c();const l=document.createElement("div");l.className="modal-overlay",l.id="explain-modal",l.innerHTML=`<div class="modal">${i}</div>`,l.addEventListener("click",$=>{const w=$.target.closest("[data-modal-action]");w!=null&&w.dataset.modalAction&&o(w.dataset.modalAction),$.target===l&&o("cancel")}),document.body.appendChild(l)}function c(){var i;(i=document.getElementById("explain-modal"))==null||i.remove()}return()=>{n=!0,e==null||e(),c()}}function We(t,a){let n=!1,e=null,u=null,d=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${r(a)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${q()}</div>`;const m=t.querySelector("#diag-body"),k=t.querySelector("#diag-footer");V(t,(o,l)=>{var $;if(o==="run")I();else if(o==="toggle")($=l.closest(".check-item"))==null||$.classList.toggle("expanded");else if(o==="copy"){const w=l.dataset.copy;w&&i(l,w)}}),H();async function H(){let o,l;try{const[w,M]=await Promise.all([Y(),G()]);o=w.find(N=>N.id===a),l=M}catch(w){if(n)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(w))}</p>`;return}if(n)return;if(!o){m.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const $=l==null?void 0:l.networks.find(w=>w.ChainID===o.wire.ChainID);$&&(k.innerHTML=q($.Name,$.LearnURL));try{e=await De(a),y=!0}catch(w){u=String(w instanceof Error?w.message:w)}n||B()}async function I(){d=!0,u=null,B();try{e=await Ae(a),y=!0}catch(o){u=String(o instanceof Error?o.message:o)}d=!1,n||B()}function B(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${d?"disabled":""}>${d?"Running…":"Run diagnostics"}</button>
      </div>
      ${u?`<p class="error">${r(u)}</p>`:""}
      ${x()}
    `}function x(){if(!y&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),l=e.failedId?`<p><strong>Failed at: ${r(P(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${l}
      <ul class="check-list">${e.items.map(c).join("")}</ul>
    `}function P(o){var l;return((l=e==null?void 0:e.items.find($=>$.ID===o))==null?void 0:l.Title)??o}function c(o){const l=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",$=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${$?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${A($?"failed here":o.Status,l)}
          <strong>${r(o.Title)}</strong>
          <span class="muted small check-detail-inline">${r(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${$?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${r(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${$?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function i(o,l){const $=await le(l),w=o.textContent;o.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(o.textContent=w)},1500)}return()=>{n=!0}}function Je(t,a){let n=!1,e=[],u=null,d=!1,y=!1;t.innerHTML=`<h1>Security: ${r(a)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${q()}</div>`;const m=t.querySelector("#sec-body"),k=t.querySelector("#sec-footer");V(t,(c,i)=>{var o;if(c==="rerun")I();else if(c==="toggle")(o=i.closest(".check-item"))==null||o.classList.toggle("expanded");else if(c==="copy"){const l=i.dataset.copy;l&&P(i,l)}}),H();async function H(){let c,i;try{const[l,$]=await Promise.all([Y(),G()]);c=l.find(w=>w.id===a),i=$}catch(l){if(n)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(l))}</p>`;return}if(n)return;if(!c){m.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const o=i==null?void 0:i.networks.find(l=>l.ChainID===c.wire.ChainID);o&&(k.innerHTML=q(o.Name,o.LearnURL)),await I()}async function I(){d=!0,u=null,B();try{e=await Be(a),y=!0}catch(c){u=String(c instanceof Error?c.message:c)}d=!1,n||B()}function B(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${d?"disabled":""}>${d?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${r(u)}</p>`:""}
      ${!y&&d?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(x).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function x(c){const i=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${A(c.Status,i)}
          <strong>${r(c.Title)}</strong>
          <span class="muted small check-detail-inline">${r(c.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(c.Why)}</p>
          </details>
          ${c.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(c.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(c.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(c,i){const o=await le(i),l=c.textContent;c.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=l)},1500)}return()=>{n=!0}}const Ke=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Ge(t){let a=!1,n=!1,e=!1,u=null,d=!1,y=null,m=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${q()}`;const k=t.querySelector("#settings-body");V(t,x=>{if(x==="save"&&B(),x==="clear-key"){if(!y)return;n=!0;const P=t.querySelector("#ai-key");P&&(P.value=""),I(y)}}),be(t,(x,P)=>{x!=="ai-provider"||!y||(m=P,d=!1,I(y))}),H();async function H(){try{const x=await Me();if(a)return;y=x,I(x)}catch(x){if(a)return;k.innerHTML=`<p class="error">Failed to load settings: ${r(String(x))}</p>`}}function I(x){var i;const P=m??x.aiProvider;k.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${ie("ai-provider",Ke.map(o=>({value:o.value,label:o.label})),P)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${x.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${x.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${r(x.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${u?`<p class="error">${r(u)}</p>`:""}
        ${d?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const c=t.querySelector("#ai-key");c==null||c.addEventListener("input",()=>{n=!0,d=!1}),(i=t.querySelector("#ref-rpc-base"))==null||i.addEventListener("input",()=>{d=!1})}async function B(){const x=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!x||!P||!y)return;const c={aiProvider:m??y.aiProvider,refRpcBase:P.value.trim()};n&&(c.aiKey=x.value),e=!0,u=null,d=!1,I(y);try{const i=await Ne(c);if(a)return;y=i,n=!1,e=!1,d=!0,I(i)}catch(i){if(a)return;e=!1,u=String(i instanceof Error?i.message:i),I(y)}}return()=>{a=!0}}const Ye="local";function Ve(t){let a=!1,n=!1,e=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${q()}
  `;const u=t.querySelector("#targets-body");V(t,(c,i)=>{k(c,i)}),d();async function d(){try{const[c,i]=await Promise.all([Y(),G()]);if(a)return;m(c,i)}catch(c){if(a)return;u.innerHTML=`<p class="error">Failed to load machines: ${r(String(c))}</p>`}}function y(){e&&m(e.targets,e.catalog)}function m(c,i){e={targets:c,catalog:i};const o=!Qe(),l=[...c].sort((M,N)=>(M.mode==="local"?-1:0)-(N.mode==="local"?-1:0)),$=l.length?`<div class="card-grid">${l.map(M=>Xe(M,i)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${o?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node is running here as your <strong>controller</strong> — it drives nodes but doesn't host them. Add a Linux server over SSH to run one."}
          </p>
        </div>
      `,w=`
      <div class="add-actions">
        ${o?'<button class="btn" data-action="add-local">Add this machine</button>':""}
        <button class="btn${o?" btn-ghost":""}" data-action="toggle-ssh">
          ${n?"Cancel":"Add a server (SSH)"}
        </button>
      </div>
    `;u.innerHTML=`
      <section class="section">
        <div class="section-head">
          <h2>Your machines</h2>
          ${w}
        </div>
        ${n?Ze():""}
        ${$}
      </section>
    `}async function k(c,i){var o;if(c==="add-local"){await H();return}if(c==="delete-target"){const l=i.dataset.id;if(!l||!confirm(`Remove target "${l}"? This does not touch anything already running on it.`))return;await I(l);return}if(c==="toggle-ssh"){n=!n,P(),y(),n&&((o=t.querySelector("#ssh-host"))==null||o.focus());return}c==="add-ssh"&&await B()}async function H(){P();try{await ue({id:Ye,mode:"local"}),await d()}catch(c){x(c)}}async function I(c){try{await Pe(c),await d()}catch(i){x(i)}}async function B(){const c=t.querySelector("#ssh-host"),i=t.querySelector("#ssh-user"),o=t.querySelector("#ssh-key"),l=t.querySelector("#ssh-port"),$=t.querySelector("#ssh-id");if(!c||!i||!o||!l||!$)return;const w=c.value.trim(),M=i.value.trim(),N=o.value.trim(),j=l.value.trim(),X=$.value.trim();if(P(),!w||!M||!N){x(new Error("host, user, and key path are required"));return}const Z=X||et(w),h={Host:w,User:M,KeyPath:N};if(j){const f=Number.parseInt(j,10);if(!Number.isFinite(f)||f<=0){x(new Error("port must be a positive number"));return}h.Port=f}const p=t.querySelector("#ssh-submit");p&&(p.disabled=!0,p.textContent="Connecting…");try{await ue({id:Z,mode:"ssh",ssh:h}),n=!1,await d()}catch(f){x(f),p&&(p.disabled=!1,p.textContent="Add server")}}function x(c){let i=t.querySelector("#targets-error");i||(u.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),i=t.querySelector("#targets-error")),i.textContent=String(c instanceof Error?c.message:c)}function P(){var c;(c=t.querySelector("#targets-error"))==null||c.remove()}return()=>{a=!0}}function Xe(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",u=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let d,y;if(!n)d=A("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const m=a.networks.find(H=>H.ChainID===n.ChainID),k=m?m.Name:`chain ${n.ChainID}`;d=`${A(k,"ok")} ${A(n.ExecID,"neutral")} ${A(n.BeaconID,"neutral")}${n.Archive?" "+A("archive","warn"):""}`,y=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${u}</p>
      <p>${d}</p>
      <div class="card-actions">
        ${y}
        <button class="btn btn-danger" data-action="delete-target" data-id="${r(t.id)}">Remove</button>
      </div>
    </div>
  `}function Ze(){return`
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
  `}function Qe(){const t=navigator.userAgentData,a=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(a)&&!/linux|android/i.test(a)}function et(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const oe=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ne=8545,ae=5052,re=30303,tt=[369,943,1],ve={369:"default",943:"practise here first"};function nt(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${q()}</div>`;const u=t.querySelector("#wizard-body"),d=t.querySelector("#wizard-footer");V(t,(h,p)=>{l(h,p)}),be(t,(h,p)=>{h==="exec-select"?e.execId=p:h==="beacon-select"&&(e.beaconId=p),m()}),y();async function y(){try{const[h,p]=await Promise.all([G(),Y()]);if(n)return;e.catalog=h;const f=p.find(E=>E.id===a);f!=null&&f.wire&&(e.chainId=f.wire.ChainID,e.execId=f.wire.ExecID,e.beaconId=f.wire.BeaconID,e.archive=f.wire.Archive,f.wire.ExecHTTPPort&&(e.execHTTPPort=String(f.wire.ExecHTTPPort)),f.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(f.wire.BeaconHTTPPort)),f.wire.ExecP2PPort&&(e.execP2PPort=String(f.wire.ExecP2PPort)),f.wire.RPCBindAddr&&(e.rpcBindAddr=f.wire.RPCBindAddr)),m()}catch(h){if(n)return;e.loadError=String(h instanceof Error?h.message:h),m()}}function m(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${Z(e.step)}
      ${H()}
    `,k())}function k(){var p;const h=(p=e.catalog)==null?void 0:p.networks.find(f=>f.ChainID===e.chainId);d.innerHTML=h?q(h.Name,h.LearnURL):q()}function H(){switch(e.step){case"network":return I();case"clients":return B();case"mode":return c();case"review":return i();case"run":return o()}}function I(){const h=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${tt.map(f=>{const E=h.networks.find(C=>C.ChainID===f);if(!E)return"";const S=e.chainId===f,R=ve[f]?A(ve[f],f===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${S?"selected":""}" data-action="pick-network" data-chain-id="${f}" type="button">
          <h3>${r(E.Name)} <span class="muted">(chain ${f})</span></h3>
          ${R}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function B(){const h=e.catalog,p=h.networks.find(S=>S.ChainID===e.chainId);if(!p)return'<p class="error">Unknown network.</p>';(e.execId===null||!p.ExecClients.includes(e.execId))&&(e.execId=p.ExecClients[0]??null),(e.beaconId===null||!p.BeaconClients.includes(e.beaconId))&&(e.beaconId=p.BeaconClients[0]??null);const f=p.ExecClients.map(S=>P(S,h)),E=p.BeaconClients.map(S=>P(S,h));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(p.Name)} are offered.</p>
        <label>
          Execution client
          ${ie("exec-select",f,e.execId)}
        </label>
        <label>
          Beacon client
          ${ie("beacon-select",E,e.beaconId)}
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function x(h){return h<=0?"—":h>=1?`~${h.toFixed(1)} TB`:`~${Math.round(h*1e3)} GB`}function P(h,p){const f=p.clients.find(E=>E.id===h);return{value:h,label:f?`${f.id} (${f.toolchain})`:h}}function c(){var C;const h=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"",p=(C=e.catalog)==null?void 0:C.networks.find(U=>U.ChainID===e.chainId),f=(p==null?void 0:p.ArchiveSizeTB)??0,E=p?x(f/2):"Smaller",S=p?x(f):"Much larger";return`
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
            <tr><th>Approx. disk footprint${p?` on ${r(p.Name)}`:""}</th><td class="yes">${E}</td><td class="limited">${S}</td></tr>
            <tr><th>Initial sync time</th><td class="yes">Faster</td><td class="limited">Much slower</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${S}${p?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${E}${p?"":" disk"}</span>
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${r(h)})</span>
            <input id="data-dir-input" type="text" placeholder="${r(h)}" value="${r(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${r(h)}/jwt.hex" value="${r(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${ne})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${ne}" value="${r(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${r(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${ae})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ae}" value="${r(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${r(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${re})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${re}" value="${r(e.execP2PPort)}" />
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
    `}function i(){const p=e.catalog.networks.find(s=>s.ChainID===e.chainId),f=e.dataDir||`/var/lib/valve-node/${e.chainId}`,E=e.jwtPath||`${f}/jwt.hex`,S=oe.map(s=>`<li>${r(s.title)}</li>`).join(""),R=j(e.execHTTPPort,ne),C=j(e.beaconHTTPPort,ae),U=j(e.execP2PPort,re),Q=R||C||U?`<tr><th>Non-default ports</th><td>${[R?`exec HTTP ${R}`:null,C?`beacon HTTP ${C}`:null,U?`exec p2p ${U}`:null].filter(s=>s!==null).map(r).join(", ")}</td></tr>`:"",{addr:T}=w(e.rpcBindAddr),O=T?`<tr><th>RPC bind address</th><td><code>${r(T)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((p==null?void 0:p.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(f)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(E)}</code></td></tr>
            ${p?`<tr><th>Checkpoint sync</th><td><code>${r(p.CheckpointURL)}</code></td></tr>`:""}
            ${Q}
            ${O}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${S}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function o(){const p=e.catalog.networks.find(T=>T.ChainID===e.chainId),f=p==null?void 0:p.LearnURL,E=new Set(e.events.filter(T=>T.done).map(T=>T.stepId)),S=new Set(e.events.filter(T=>T.err).map(T=>T.stepId)),R=new Map;for(const T of e.events){if(!T.line)continue;const O=R.get(T.stepId)??[];O.push(T.line),R.set(T.stepId,O)}const C=oe.map(T=>{var F;const O=E.has(T.id),s=S.has(T.id),g=s?A("failed","bad"):O?A("done","ok"):A("pending","neutral"),v=(R.get(T.id)??[]).slice(-5),b=(F=e.events.find(J=>J.stepId===T.id&&J.err))==null?void 0:F.err,L=T.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${f?` <a href="${r(f)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${O?"step-done":""} ${s?"step-error":""}">
          <div class="step-head">${g} <strong>${r(T.title)}</strong></div>
          ${L}
          ${v.length?`<pre class="step-log">${v.map(J=>r(J)).join(`
`)}</pre>`:""}
          ${b?`<p class="error small">${r(b)}</p>`:""}
        </li>
      `}).join(""),U=e.events.some(T=>T.err),Q=oe.every(T=>E.has(T.id))||e.events.some(T=>T.stepId==="handshake"&&T.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${C}</ol>
        ${Q&&!U?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${U?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function l(h,p){switch(h){case"pick-network":e.chainId=Number(p.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m();break;case"goto-review":if($(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError){m();break}e.step="review",m();break;case"start-setup":X();break}}function $(){const h=t.querySelectorAll('input[name="mode"]');for(const U of Array.from(h))U.checked&&(e.archive=U.value==="archive");const p=t.querySelector("#data-dir-input"),f=t.querySelector("#jwt-path-input");p&&(e.dataDir=p.value.trim()),f&&(e.jwtPath=f.value.trim());const E=t.querySelector("#exec-http-port-input"),S=t.querySelector("#beacon-http-port-input"),R=t.querySelector("#exec-p2p-port-input");E&&(e.execHTTPPort=E.value.trim()),S&&(e.beaconHTTPPort=S.value.trim()),R&&(e.execP2PPort=R.value.trim());const C=t.querySelector("#rpc-bind-addr-input");C&&(e.rpcBindAddr=C.value.trim()),e.execHTTPPortError=N(e.execHTTPPort).error??null,e.beaconHTTPPortError=N(e.beaconHTTPPort).error??null,e.execP2PPortError=N(e.execP2PPort).error??null,e.rpcBindAddrError=w(e.rpcBindAddr).error??null}function w(h){if(!h)return{};const p=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(h);return p?p.slice(1).every(f=>Number(f)<=255)?{addr:h}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(h)&&h.includes(":")?{addr:h}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const M=/^\d+$/;function N(h){if(!h)return{};if(!M.test(h))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const p=Number(h);return!Number.isInteger(p)||p<1||p>65535?{error:"Port must be between 1 and 65535."}:{port:p}}function j(h,p){const{port:f}=N(h);if(!(f===void 0||f===p))return f}async function X(){var R;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,m();const h={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(h.DataDir=e.dataDir),e.jwtPath&&(h.JWTPath=e.jwtPath);const p=j(e.execHTTPPort,ne),f=j(e.beaconHTTPPort,ae),E=j(e.execP2PPort,re);p!==void 0&&(h.ExecHTTPPort=p),f!==void 0&&(h.BeaconHTTPPort=f),E!==void 0&&(h.ExecP2PPort=E);const{addr:S}=w(e.rpcBindAddr);S!==void 0&&(h.RPCBindAddr=S);try{await Te(e.targetId,h)}catch(C){if(!(C instanceof ce&&C.status===409)){e.starting=!1,e.startError=String(C instanceof Error?C.message:C),m();return}}e.starting=!1,e.step="run",e.events=[],m(),(R=e.streamStop)==null||R.call(e),e.streamStop=ke(e.targetId,C=>{n||(e.events.push(C),e.step==="run"&&m())})}function Z(h){const p=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],E=p.map(S=>S.id).indexOf(h);return`
      <ol class="wizard-progress">
        ${p.map((S,R)=>`<li class="${R===E?"current":R<E?"past":"future"}">${r(S.label)}</li>`).join("")}
      </ol>
    `}return()=>{var h;n=!0,(h=e.streamStop)==null||h.call(e)}}const at=document.querySelector("#app"),{contentEl:rt,setActiveNav:st}=qe(at);let _=null;function ot(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function W(t){const a=document.createElement("div");return rt.replaceChildren(a),t(a)}function ye(){if(_){try{_()}catch{}_=null}const{screen:t,id:a}=ot();switch(st(t),t){case"setup":if(!a){location.hash="#/targets";return}_=W(n=>nt(n,a));break;case"dash":if(!a){location.hash="#/targets";return}_=W(n=>_e(n,a));break;case"logs":if(!a){location.hash="#/targets";return}_=W(n=>ze(n,a));break;case"security":if(!a){location.hash="#/targets";return}_=W(n=>Je(n,a));break;case"diag":if(!a){location.hash="#/targets";return}_=W(n=>We(n,a));break;case"settings":_=W(n=>Ge(n));break;case"targets":default:_=W(n=>Ve(n));break}}window.addEventListener("hashchange",ye);ye();
