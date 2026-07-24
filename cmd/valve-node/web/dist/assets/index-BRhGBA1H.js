var we=Object.defineProperty;var xe=(t,a,n)=>a in t?we(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var le=(t,a,n)=>xe(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const d of u)if(d.type==="childList")for(const y of d.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(u){const d={};return u.integrity&&(d.integrity=u.integrity),u.referrerPolicy&&(d.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?d.credentials="include":u.crossOrigin==="anonymous"?d.credentials="omit":d.credentials="same-origin",d}function e(u){if(u.ep)return;u.ep=!0;const d=n(u);fetch(u.href,d)}})();function Y(){return M("/api/catalog")}function G(){return M("/api/targets")}function de(t){return M("/api/targets",{method:"POST",headers:ee,body:JSON.stringify(t)})}function Pe(t){return M(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,a){return M(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:ee,body:JSON.stringify(a)})}function Ee(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function ke(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Se(t,a=200){return M(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function Ce(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function ue(t,a){const n=a===void 0?{}:{lines:a};return M(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:ee,body:JSON.stringify(n)})}function Le(t,a,n){return M(`/api/targets/${encodeURIComponent(t)}/services/${a}/${n}`,{method:"POST"})}function Ie(t,a){return M(`/api/targets/${encodeURIComponent(t)}/services/${a}/clear`,{method:"POST",headers:ee,body:JSON.stringify({Confirm:a})})}function He(t){return M(`/api/targets/${encodeURIComponent(t)}/du`)}function Re(t){return M(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Be(t){return M(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Ae(t){return M(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function De(t){return M(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Me(){return M("/api/settings")}function Ne(t){return M("/api/settings",{method:"PUT",headers:ee,body:JSON.stringify(t)})}class ie extends Error{constructor(n,e){super(e);le(this,"status");this.name="ApiError",this.status=n}}const ee={"Content-Type":"application/json"};async function M(t,a){const n=await fetch(t,a);if(!n.ok){let u=n.statusText||`HTTP ${n.status}`;try{const d=await n.json();d&&typeof d.error=="string"&&d.error&&(u=d.error)}catch{}throw new ie(n.status,u)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const Ue="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function O(t,a){const n=t&&a?` <span class="footer-sep">·</span> <a href="${r(a)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
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
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:u=>{for(const d of n)d.classList.toggle("active",d.dataset.nav===u)}}}function z(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Oe(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Fe(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const u=Math.floor(n/24),d=n%24;return`~${u}d ${d}h`}function D(t,a){return`<span class="badge badge-${a}">${r(t)}</span>`}function pe(t){return`<span class="dot dot-${t}"></span>`}const he=["B","KB","MB","GB","TB","PB"];function K(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let a=t,n=0;for(;a>=1024&&n<he.length-1;)a/=1024,n++;const e=a<10?2:a<100?1:0;return`${a.toFixed(e)} ${he[n]}`}async function ce(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function V(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const u=e.dataset.action;u&&a(u,e,n)})}function oe(t,a,n){const e=a.find(d=>d.value===n),u=a.map(d=>`
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
  `}function Q(t){t.querySelectorAll(".dropdown.open").forEach(a=>{var n;a.classList.remove("open"),(n=a.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function ve(t,a){t.addEventListener("click",u=>{const d=u.target,y=d.closest(".dropdown-trigger");if(y&&t.contains(y)){const k=y.closest(".dropdown"),I=!!k&&!k.classList.contains("open");Q(t),k&&I&&(k.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const h=d.closest(".dropdown-option");if(h&&t.contains(h)){const k=h.closest(".dropdown");Q(t),a((k==null?void 0:k.dataset.dropdown)??"",h.dataset.value??"");return}Q(t)});const n=u=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const d=u.target;(!d.closest(".dropdown")||!t.contains(d))&&Q(t)},e=u=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}u.key==="Escape"&&Q(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const je=85,re={exec:"Execution",beacon:"Beacon"};function _e(t,a){let n=!1,e=null,u=null,d=null,y=null,h=null,k=null,I=null,L=null;const B={exec:null,beacon:null};let w=null;t.innerHTML=`<h1>Dashboard: ${r(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${O()}</div>`;const P=t.querySelector("#dash-body"),c=t.querySelector("#dash-footer");P.addEventListener("click",s=>{const g=s.target.closest("[data-action]");if(!g||!P.contains(g))return;const v=g.dataset.action;if(v==="svc-action"){const b=g.dataset.svc,S=g.dataset.kind;b&&S&&H(b,S)}else if(v==="open-clear"){const b=g.dataset.svc;b&&Z(b)}else if(v==="copy"){const b=g.dataset.copy;b&&F(g,b)}else v==="retry-du"?o():v==="retry-endpoints"&&l()}),i();async function i(){let s,g;try{const[b,S]=await Promise.all([G(),Y()]);s=b.find(U=>U.id===a),g=S}catch(b){if(n)return;P.innerHTML=`<p class="error">Failed to load target: ${r(String(b))}</p>`;return}if(n)return;if(!s){P.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const v=g==null?void 0:g.networks.find(b=>b.ChainID===s.wire.ChainID);v&&(c.innerHTML=O(v.Name,v.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=ke(a,b=>{n||($(b),u=b,d=b,x())}),o(),l()}async function o(){k=null;try{h=await He(a)}catch(s){h=null,k=String(s instanceof Error?s.message:s)}n||x()}async function l(){L=null;try{I=await Re(a)}catch(s){I=null,L=String(s instanceof Error?s.message:s)}n||x()}function $(s){if(!u)return;const g=(new Date(s.at).getTime()-new Date(u.at).getTime())/1e3,v=s.execHead-u.execHead;if(g>0&&v>=0){const b=v/g;y=y===null?b:y*.7+b*.3}}function x(){if(!d)return;const s=d;P.innerHTML=`
      <div class="card-grid">
        ${N(s)}
        ${J(s)}
        ${X(s)}
        ${p(s)}
        ${m(s)}
        ${f()}
        ${C(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function A(s){const v=s.refHead>0?s.refHead-s.execHead:null,b=v!==null&&v>0&&y&&y>0?Fe(v/y):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function N(s){const{lag:g,eta:v}=A(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?D("syncing","warn"):D("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${z(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${g!==null?z(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${g!==null?z(Math.max(g,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function J(s){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${s.beaconDistance===0?D("synced","ok"):D("syncing","warn")}</p>
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
    `}function p(s){const g=s.diskUsedPct>=je;return`
      <div class="card ${g?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${g?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Oe(s.diskUsedPct)} used</p>
      </div>
    `}function m(s){if(k)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(k)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!h)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const g=h.ExpectedExecBytes>0?Math.min(h.ExecBytes/h.ExpectedExecBytes*100,100):0,v=h.ExpectedBeaconBytes>0?Math.min(h.BeaconBytes/h.ExpectedBeaconBytes*100,100):0,{lag:b,eta:S}=A(s),U=b!==null&&b>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${K(h.ExecBytes)} of ~${K(h.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        ${U?`<p class="muted small">Estimated time remaining: ${r(S)}</p>`:""}
        <p class="muted small">Beacon — ${K(h.BeaconBytes)} of ~${K(h.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${K(h.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(h.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(h.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function f(){if(L)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(L)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!I)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=I,g=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=s.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${r(s.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${pe(s.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${pe(s.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.BeaconHTTP)}">Copy</button>
        </div>
        ${g}
        ${v}
      </div>
    `}function E(s,g){const v=re[s],b=B[s],S=(U,ye,$e)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${U}" ${b!==null||$e?"disabled":""}>${b===U?R():r(ye)}</button>`;return`
      <div class="service-row">
        <span>${r(v)} ${g?D("active","ok"):D("down","bad")}</span>
        <div class="service-actions">
          ${S("start","Start",g)}
          ${S("stop","Stop",!g)}
          ${S("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function C(s){return`
      <div class="card">
        <h3>Services</h3>
        ${E("exec",s.execActive)}
        ${E("beacon",s.beaconActive)}
        ${w?`<p class="error small">${r(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(a)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(a)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(a)}">Diagnostics →</a>
        </p>
      </div>
    `}function R(){return'<span class="spinner" aria-label="working"></span>'}async function H(s,g){if(B[s]===null){B[s]=g,w=null,x();try{await Le(a,s,g)}catch(v){w=`${re[s]} ${g} failed: ${v instanceof Error?v.message:String(v)}`}B[s]=null,n||x()}}async function F(s,g){const v=await ce(g),b=s.textContent;s.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{n||(s.textContent=b)},1500)}function Z(s){const g=re[s],v=h?K(s==="exec"?h.ExecBytes:h.BeaconBytes):"unknown (disk usage hasn't loaded)";_(`
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
      `,U=>{if(U==="cancel"){q();return}U==="confirm"&&T(s)});const b=document.getElementById("clear-confirm-input"),S=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{S&&(S.disabled=b.value.trim()!==s)}),b==null||b.focus()}async function T(s){const g=document.getElementById("clear-confirm-btn");g&&(g.disabled=!0,g.textContent="Clearing…");try{await Ie(a,s),q(),o()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const S=document.createElement("p");S.className="error small",S.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(S)}g&&(g.disabled=!1,g.textContent="Clear and resync")}}function _(s,g){q();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${s}</div>`,v.addEventListener("click",b=>{const S=b.target.closest("[data-modal-action]");S!=null&&S.dataset.modalAction&&g(S.dataset.modalAction),b.target===v&&g("cancel")}),document.body.appendChild(v)}function q(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{n=!0,e==null||e(),q()}}const fe=500,me="valve-node.explain-consent";function ze(t,a){let n=!1,e=null;const u=[];t.innerHTML=`
    <h1>Logs: ${r(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${O()}</div>
  `;const d=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");V(t,i=>{i==="explain"&&L()}),h();async function h(){let i,o;try{const[$,x]=await Promise.all([G(),Y()]);i=$.find(A=>A.id===a),o=x}catch($){if(n)return;d.innerHTML=`<p class="error">Failed to load target: ${r(String($))}</p>`;return}if(n)return;if(!i){d.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){d.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const l=o==null?void 0:o.networks.find($=>$.ChainID===i.wire.ChainID);l&&(y.innerHTML=O(l.Name,l.LearnURL));try{const $=await Se(a,200);if(n)return;u.push(...$)}catch($){if(n)return;d.innerHTML=`<p class="error">Failed to load logs: ${r(String($))}</p>`;return}k(),e=Ce(a,$=>{n||(u.push($),u.length>fe&&u.splice(0,u.length-fe),k())})}function k(){const i=u.filter(l=>l.severity==="error"||l.severity==="critical");d.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(I).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${D(String(i.length),i.length?"bad":"neutral")}</h2>
          <div class="log-lines">${i.length?i.slice().reverse().map(I).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=d.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function I(i){const o=i.severity||"info",l=i.learnUrl?` <a href="${r(i.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(i.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(i.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(i.line)}</span>
        ${i.explain?`<div class="log-explain">${r(i.explain)}${l}</div>`:""}
      </div>
    `}async function L(){const i=u.filter(l=>l.severity==="error"||l.severity==="critical").map(l=>l.line).slice(-40);if(!(localStorage.getItem(me)==="1")){B(i);return}await w(i)}function B(i){const o=i.length?`<pre class="explain-excerpt">${i.map(l=>r(l)).join(`
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
    `,l=>{l==="proceed"?(localStorage.setItem(me,"1"),c(),w(i)):c()})}async function w(i){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=i.length?await ue(a,i):await ue(a);if(n)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(l=>r(l)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&c()})}catch(o){if(n)return;if(o instanceof ie&&o.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,l=>{l==="close"&&c()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,l=>{l==="close"&&c()})}}function P(i,o){c();const l=document.createElement("div");l.className="modal-overlay",l.id="explain-modal",l.innerHTML=`<div class="modal">${i}</div>`,l.addEventListener("click",$=>{const x=$.target.closest("[data-modal-action]");x!=null&&x.dataset.modalAction&&o(x.dataset.modalAction),$.target===l&&o("cancel")}),document.body.appendChild(l)}function c(){var i;(i=document.getElementById("explain-modal"))==null||i.remove()}return()=>{n=!0,e==null||e(),c()}}function We(t,a){let n=!1,e=null,u=null,d=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${r(a)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${O()}</div>`;const h=t.querySelector("#diag-body"),k=t.querySelector("#diag-footer");V(t,(o,l)=>{var $;if(o==="run")L();else if(o==="toggle")($=l.closest(".check-item"))==null||$.classList.toggle("expanded");else if(o==="copy"){const x=l.dataset.copy;x&&i(l,x)}}),I();async function I(){let o,l;try{const[x,A]=await Promise.all([G(),Y()]);o=x.find(N=>N.id===a),l=A}catch(x){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${r(String(x))}</p>`;return}if(n)return;if(!o){h.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const $=l==null?void 0:l.networks.find(x=>x.ChainID===o.wire.ChainID);$&&(k.innerHTML=O($.Name,$.LearnURL));try{e=await De(a),y=!0}catch(x){u=String(x instanceof Error?x.message:x)}n||B()}async function L(){d=!0,u=null,B();try{e=await Ae(a),y=!0}catch(o){u=String(o instanceof Error?o.message:o)}d=!1,n||B()}function B(){h.innerHTML=`
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
      ${w()}
    `}function w(){if(!y&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),l=e.failedId?`<p><strong>Failed at: ${r(P(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${l}
      <ul class="check-list">${e.items.map(c).join("")}</ul>
    `}function P(o){var l;return((l=e==null?void 0:e.items.find($=>$.ID===o))==null?void 0:l.Title)??o}function c(o){const l=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",$=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${$?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${D($?"failed here":o.Status,l)}
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
    `}async function i(o,l){const $=await ce(l),x=o.textContent;o.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(o.textContent=x)},1500)}return()=>{n=!0}}function Je(t,a){let n=!1,e=[],u=null,d=!1,y=!1;t.innerHTML=`<h1>Security: ${r(a)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${O()}</div>`;const h=t.querySelector("#sec-body"),k=t.querySelector("#sec-footer");V(t,(c,i)=>{var o;if(c==="rerun")L();else if(c==="toggle")(o=i.closest(".check-item"))==null||o.classList.toggle("expanded");else if(c==="copy"){const l=i.dataset.copy;l&&P(i,l)}}),I();async function I(){let c,i;try{const[l,$]=await Promise.all([G(),Y()]);c=l.find(x=>x.id===a),i=$}catch(l){if(n)return;h.innerHTML=`<p class="error">Failed to load target: ${r(String(l))}</p>`;return}if(n)return;if(!c){h.innerHTML=`<p class="error">Target "${r(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const o=i==null?void 0:i.networks.find(l=>l.ChainID===c.wire.ChainID);o&&(k.innerHTML=O(o.Name,o.LearnURL)),await L()}async function L(){d=!0,u=null,B();try{e=await Be(a),y=!0}catch(c){u=String(c instanceof Error?c.message:c)}d=!1,n||B()}function B(){h.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${d?"disabled":""}>${d?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${r(u)}</p>`:""}
      ${!y&&d?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function w(c){const i=c.Status==="pass"?"ok":c.Status==="fail"?"bad":c.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${D(c.Status,i)}
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
    `}async function P(c,i){const o=await ce(i),l=c.textContent;c.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{n||(c.textContent=l)},1500)}return()=>{n=!0}}const Ke=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Ye(t){let a=!1,n=!1,e=!1,u=null,d=!1,y=null,h=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${O()}`;const k=t.querySelector("#settings-body");V(t,w=>{if(w==="save"&&B(),w==="clear-key"){if(!y)return;n=!0;const P=t.querySelector("#ai-key");P&&(P.value=""),L(y)}}),ve(t,(w,P)=>{w!=="ai-provider"||!y||(h=P,d=!1,L(y))}),I();async function I(){try{const w=await Me();if(a)return;y=w,L(w)}catch(w){if(a)return;k.innerHTML=`<p class="error">Failed to load settings: ${r(String(w))}</p>`}}function L(w){var i;const P=h??w.aiProvider;k.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${oe("ai-provider",Ke.map(o=>({value:o.value,label:o.label})),P)}
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
            <input id="ref-rpc-base" type="text" value="${r(w.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${u?`<p class="error">${r(u)}</p>`:""}
        ${d?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const c=t.querySelector("#ai-key");c==null||c.addEventListener("input",()=>{n=!0,d=!1}),(i=t.querySelector("#ref-rpc-base"))==null||i.addEventListener("input",()=>{d=!1})}async function B(){const w=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!w||!P||!y)return;const c={aiProvider:h??y.aiProvider,refRpcBase:P.value.trim()};n&&(c.aiKey=w.value),e=!0,u=null,d=!1,L(y);try{const i=await Ne(c);if(a)return;y=i,n=!1,e=!1,d=!0,L(i)}catch(i){if(a)return;e=!1,u=String(i instanceof Error?i.message:i),L(y)}}return()=>{a=!0}}const Ge="local";function Ve(t){let a=!1,n=!1,e=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${O()}
  `;const u=t.querySelector("#targets-body");V(t,(c,i)=>{k(c,i)}),d();async function d(){try{const[c,i]=await Promise.all([G(),Y()]);if(a)return;h(c,i)}catch(c){if(a)return;u.innerHTML=`<p class="error">Failed to load machines: ${r(String(c))}</p>`}}function y(){e&&h(e.targets,e.catalog)}function h(c,i){e={targets:c,catalog:i};const o=!Qe(),l=[...c].sort((A,N)=>(A.mode==="local"?-1:0)-(N.mode==="local"?-1:0)),$=l.length?`<div class="card-grid">${l.map(A=>Xe(A,i)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${o?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node is running here as your <strong>controller</strong> — it drives nodes but doesn't host them. Add a Linux server over SSH to run one."}
          </p>
        </div>
      `,x=`
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
          ${x}
        </div>
        ${n?Ze():""}
        ${$}
      </section>
    `}async function k(c,i){var o;if(c==="add-local"){await I();return}if(c==="delete-target"){const l=i.dataset.id;if(!l||!confirm(`Remove target "${l}"? This does not touch anything already running on it.`))return;await L(l);return}if(c==="toggle-ssh"){n=!n,P(),y(),n&&((o=t.querySelector("#ssh-host"))==null||o.focus());return}c==="add-ssh"&&await B()}async function I(){P();try{await de({id:Ge,mode:"local"}),await d()}catch(c){w(c)}}async function L(c){try{await Pe(c),await d()}catch(i){w(i)}}async function B(){const c=t.querySelector("#ssh-host"),i=t.querySelector("#ssh-user"),o=t.querySelector("#ssh-key"),l=t.querySelector("#ssh-port"),$=t.querySelector("#ssh-id");if(!c||!i||!o||!l||!$)return;const x=c.value.trim(),A=i.value.trim(),N=o.value.trim(),J=l.value.trim(),X=$.value.trim();if(P(),!x||!A||!N){w(new Error("host, user, and key path are required"));return}const p=X||et(x),m={Host:x,User:A,KeyPath:N};if(J){const E=Number.parseInt(J,10);if(!Number.isFinite(E)||E<=0){w(new Error("port must be a positive number"));return}m.Port=E}const f=t.querySelector("#ssh-submit");f&&(f.disabled=!0,f.textContent="Connecting…");try{await de({id:p,mode:"ssh",ssh:m}),n=!1,await d()}catch(E){w(E),f&&(f.disabled=!1,f.textContent="Add server")}}function w(c){let i=t.querySelector("#targets-error");i||(u.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),i=t.querySelector("#targets-error")),i.textContent=String(c instanceof Error?c.message:c)}function P(){var c;(c=t.querySelector("#targets-error"))==null||c.remove()}return()=>{a=!0}}function Xe(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",u=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let d,y;if(!n)d=D("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const h=a.networks.find(I=>I.ChainID===n.ChainID),k=h?h.Name:`chain ${n.ChainID}`;d=`${D(k,"ok")} ${D(n.ExecID,"neutral")} ${D(n.BeaconID,"neutral")}${n.Archive?" "+D("archive","warn"):""}`,y=`
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
  `}function Qe(){const t=navigator.userAgentData,a=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(a)&&!/linux|android/i.test(a)}function et(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const se=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],te=8545,ne=5052,ae=30303,tt=[369,943,1],ge={369:"default",943:"practise here first"};function nt(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${O()}</div>`;const u=t.querySelector("#wizard-body"),d=t.querySelector("#wizard-footer");V(t,(p,m)=>{o(p,m)}),ve(t,(p,m)=>{p==="exec-select"?e.execId=m:p==="beacon-select"&&(e.beaconId=m),h()}),y();async function y(){try{const[p,m]=await Promise.all([Y(),G()]);if(n)return;e.catalog=p;const f=m.find(E=>E.id===a);f!=null&&f.wire&&(e.chainId=f.wire.ChainID,e.execId=f.wire.ExecID,e.beaconId=f.wire.BeaconID,e.archive=f.wire.Archive,f.wire.ExecHTTPPort&&(e.execHTTPPort=String(f.wire.ExecHTTPPort)),f.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(f.wire.BeaconHTTPPort)),f.wire.ExecP2PPort&&(e.execP2PPort=String(f.wire.ExecP2PPort)),f.wire.RPCBindAddr&&(e.rpcBindAddr=f.wire.RPCBindAddr)),h()}catch(p){if(n)return;e.loadError=String(p instanceof Error?p.message:p),h()}}function h(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${X(e.step)}
      ${I()}
    `,k())}function k(){var m;const p=(m=e.catalog)==null?void 0:m.networks.find(f=>f.ChainID===e.chainId);d.innerHTML=p?O(p.Name,p.LearnURL):O()}function I(){switch(e.step){case"network":return L();case"clients":return B();case"mode":return P();case"review":return c();case"run":return i()}}function L(){const p=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${tt.map(f=>{const E=p.networks.find(H=>H.ChainID===f);if(!E)return"";const C=e.chainId===f,R=ge[f]?D(ge[f],f===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${C?"selected":""}" data-action="pick-network" data-chain-id="${f}" type="button">
          <h3>${r(E.Name)} <span class="muted">(chain ${f})</span></h3>
          ${R}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function B(){const p=e.catalog,m=p.networks.find(C=>C.ChainID===e.chainId);if(!m)return'<p class="error">Unknown network.</p>';(e.execId===null||!m.ExecClients.includes(e.execId))&&(e.execId=m.ExecClients[0]??null),(e.beaconId===null||!m.BeaconClients.includes(e.beaconId))&&(e.beaconId=m.BeaconClients[0]??null);const f=m.ExecClients.map(C=>w(C,p)),E=m.BeaconClients.map(C=>w(C,p));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(m.Name)} are offered.</p>
        <label>
          Execution client
          ${oe("exec-select",f,e.execId)}
        </label>
        <label>
          Beacon client
          ${oe("beacon-select",E,e.beaconId)}
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(p,m){const f=m.clients.find(E=>E.id===p);return{value:p,label:f?`${f.id} (${f.toolchain})`:p}}function P(){const p=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
            <tr><th>Disk footprint</th><td class="yes">Smaller</td><td class="limited">Much larger</td></tr>
            <tr><th>Initial sync time</th><td class="yes">Faster</td><td class="limited">Much slower</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — recommended for most nodes</span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — only if you need full historical state</span>
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${r(p)})</span>
            <input id="data-dir-input" type="text" placeholder="${r(p)}" value="${r(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${r(p)}/jwt.hex" value="${r(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${te})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${te}" value="${r(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${r(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${ne})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ne}" value="${r(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${r(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${ae})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${ae}" value="${r(e.execP2PPort)}" />
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
    `}function c(){const m=e.catalog.networks.find(q=>q.ChainID===e.chainId),f=e.dataDir||`/var/lib/valve-node/${e.chainId}`,E=e.jwtPath||`${f}/jwt.hex`,C=se.map(q=>`<li>${r(q.title)}</li>`).join(""),R=N(e.execHTTPPort,te),H=N(e.beaconHTTPPort,ne),F=N(e.execP2PPort,ae),Z=R||H||F?`<tr><th>Non-default ports</th><td>${[R?`exec HTTP ${R}`:null,H?`beacon HTTP ${H}`:null,F?`exec p2p ${F}`:null].filter(q=>q!==null).map(r).join(", ")}</td></tr>`:"",{addr:T}=$(e.rpcBindAddr),_=T?`<tr><th>RPC bind address</th><td><code>${r(T)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((m==null?void 0:m.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(f)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(E)}</code></td></tr>
            ${m?`<tr><th>Checkpoint sync</th><td><code>${r(m.CheckpointURL)}</code></td></tr>`:""}
            ${Z}
            ${_}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${C}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function i(){const m=e.catalog.networks.find(T=>T.ChainID===e.chainId),f=m==null?void 0:m.LearnURL,E=new Set(e.events.filter(T=>T.done).map(T=>T.stepId)),C=new Set(e.events.filter(T=>T.err).map(T=>T.stepId)),R=new Map;for(const T of e.events){if(!T.line)continue;const _=R.get(T.stepId)??[];_.push(T.line),R.set(T.stepId,_)}const H=se.map(T=>{var S;const _=E.has(T.id),q=C.has(T.id),s=q?D("failed","bad"):_?D("done","ok"):D("pending","neutral"),g=(R.get(T.id)??[]).slice(-5),v=(S=e.events.find(U=>U.stepId===T.id&&U.err))==null?void 0:S.err,b=T.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${f?` <a href="${r(f)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${_?"step-done":""} ${q?"step-error":""}">
          <div class="step-head">${s} <strong>${r(T.title)}</strong></div>
          ${b}
          ${g.length?`<pre class="step-log">${g.map(U=>r(U)).join(`
`)}</pre>`:""}
          ${v?`<p class="error small">${r(v)}</p>`:""}
        </li>
      `}).join(""),F=e.events.some(T=>T.err),Z=se.every(T=>E.has(T.id))||e.events.some(T=>T.stepId==="handshake"&&T.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${H}</ol>
        ${Z&&!F?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${F?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function o(p,m){switch(p){case"pick-network":e.chainId=Number(m.dataset.chainId),e.execId=null,e.beaconId=null,h();break;case"goto-network":e.step="network",h();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",h();break;case"goto-mode":e.step="mode",h();break;case"goto-review":if(l(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError){h();break}e.step="review",h();break;case"start-setup":J();break}}function l(){const p=t.querySelectorAll('input[name="mode"]');for(const F of Array.from(p))F.checked&&(e.archive=F.value==="archive");const m=t.querySelector("#data-dir-input"),f=t.querySelector("#jwt-path-input");m&&(e.dataDir=m.value.trim()),f&&(e.jwtPath=f.value.trim());const E=t.querySelector("#exec-http-port-input"),C=t.querySelector("#beacon-http-port-input"),R=t.querySelector("#exec-p2p-port-input");E&&(e.execHTTPPort=E.value.trim()),C&&(e.beaconHTTPPort=C.value.trim()),R&&(e.execP2PPort=R.value.trim());const H=t.querySelector("#rpc-bind-addr-input");H&&(e.rpcBindAddr=H.value.trim()),e.execHTTPPortError=A(e.execHTTPPort).error??null,e.beaconHTTPPortError=A(e.beaconHTTPPort).error??null,e.execP2PPortError=A(e.execP2PPort).error??null,e.rpcBindAddrError=$(e.rpcBindAddr).error??null}function $(p){if(!p)return{};const m=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(p);return m?m.slice(1).every(f=>Number(f)<=255)?{addr:p}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(p)&&p.includes(":")?{addr:p}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const x=/^\d+$/;function A(p){if(!p)return{};if(!x.test(p))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const m=Number(p);return!Number.isInteger(m)||m<1||m>65535?{error:"Port must be between 1 and 65535."}:{port:m}}function N(p,m){const{port:f}=A(p);if(!(f===void 0||f===m))return f}async function J(){var R;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,h();const p={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(p.DataDir=e.dataDir),e.jwtPath&&(p.JWTPath=e.jwtPath);const m=N(e.execHTTPPort,te),f=N(e.beaconHTTPPort,ne),E=N(e.execP2PPort,ae);m!==void 0&&(p.ExecHTTPPort=m),f!==void 0&&(p.BeaconHTTPPort=f),E!==void 0&&(p.ExecP2PPort=E);const{addr:C}=$(e.rpcBindAddr);C!==void 0&&(p.RPCBindAddr=C);try{await Te(e.targetId,p)}catch(H){if(!(H instanceof ie&&H.status===409)){e.starting=!1,e.startError=String(H instanceof Error?H.message:H),h();return}}e.starting=!1,e.step="run",e.events=[],h(),(R=e.streamStop)==null||R.call(e),e.streamStop=Ee(e.targetId,H=>{n||(e.events.push(H),e.step==="run"&&h())})}function X(p){const m=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],E=m.map(C=>C.id).indexOf(p);return`
      <ol class="wizard-progress">
        ${m.map((C,R)=>`<li class="${R===E?"current":R<E?"past":"future"}">${r(C.label)}</li>`).join("")}
      </ol>
    `}return()=>{var p;n=!0,(p=e.streamStop)==null||p.call(e)}}const at=document.querySelector("#app"),{contentEl:rt,setActiveNav:st}=qe(at);let j=null;function ot(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function W(t){const a=document.createElement("div");return rt.replaceChildren(a),t(a)}function be(){if(j){try{j()}catch{}j=null}const{screen:t,id:a}=ot();switch(st(t),t){case"setup":if(!a){location.hash="#/targets";return}j=W(n=>nt(n,a));break;case"dash":if(!a){location.hash="#/targets";return}j=W(n=>_e(n,a));break;case"logs":if(!a){location.hash="#/targets";return}j=W(n=>ze(n,a));break;case"security":if(!a){location.hash="#/targets";return}j=W(n=>Je(n,a));break;case"diag":if(!a){location.hash="#/targets";return}j=W(n=>We(n,a));break;case"settings":j=W(n=>Ye(n));break;case"targets":default:j=W(n=>Ve(n));break}}window.addEventListener("hashchange",be);be();
