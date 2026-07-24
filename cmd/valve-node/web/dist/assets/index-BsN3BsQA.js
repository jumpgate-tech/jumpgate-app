var Pe=Object.defineProperty;var Te=(t,a,n)=>a in t?Pe(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var pe=(t,a,n)=>Te(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const p of document.querySelectorAll('link[rel="modulepreload"]'))e(p);new MutationObserver(p=>{for(const u of p)if(u.type==="childList")for(const y of u.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function n(p){const u={};return p.integrity&&(u.integrity=p.integrity),p.referrerPolicy&&(u.referrerPolicy=p.referrerPolicy),p.crossOrigin==="use-credentials"?u.credentials="include":p.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(p){if(p.ep)return;p.ep=!0;const u=n(p);fetch(p.href,u)}})();function V(){return A("/api/catalog")}function X(){return A("/api/targets")}function he(t){return A("/api/targets",{method:"POST",headers:te,body:JSON.stringify(t)})}function ke(t){return A(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Se(t,a){return A(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:te,body:JSON.stringify(a)})}function Ee(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Ce(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Le(t,a=200){return A(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function Ie(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function fe(t,a){const n=a===void 0?{}:{lines:a};return A(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:te,body:JSON.stringify(n)})}function He(t,a,n){return A(`/api/targets/${encodeURIComponent(t)}/services/${a}/${n}`,{method:"POST"})}function Re(t,a){return A(`/api/targets/${encodeURIComponent(t)}/services/${a}/clear`,{method:"POST",headers:te,body:JSON.stringify({Confirm:a})})}function Be(t){return A(`/api/targets/${encodeURIComponent(t)}/du`)}function Ae(t){return A(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function De(t){return A(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Me(t){return A(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Ne(t){return A(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Ue(){return A("/api/settings")}function qe(t){return A("/api/settings",{method:"PUT",headers:te,body:JSON.stringify(t)})}class de extends Error{constructor(n,e){super(e);pe(this,"status");this.name="ApiError",this.status=n}}const te={"Content-Type":"application/json"};async function A(t,a){const n=await fetch(t,a);if(!n.ok){let p=n.statusText||`HTTP ${n.status}`;try{const u=await n.json();u&&typeof u.error=="string"&&u.error&&(p=u.error)}catch{}throw new de(n.status,p)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const me="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function N(t,a){const n=t&&a&&a!==me?` <span class="footer-sep">·</span> <a href="${s(a)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${s(me)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function Oe(t){t.innerHTML=`
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
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:p=>{for(const u of n)u.classList.toggle("active",u.dataset.nav===p)}}}function W(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Fe(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function je(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const p=Math.floor(n/24),u=n%24;return`~${p}d ${u}h`}function B(t,a){return`<span class="badge badge-${a}">${s(t)}</span>`}function ge(t){return`<span class="dot dot-${t}"></span>`}const ve=["B","KB","MB","GB","TB","PB"];function Y(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let a=t,n=0;for(;a>=1024&&n<ve.length-1;)a/=1024,n++;const e=a<10?2:a<100?1:0;return`${a.toFixed(e)} ${ve[n]}`}async function ue(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Z(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const p=e.dataset.action;p&&a(p,e,n)})}function le(t,a,n){const e=a.find(u=>u.value===n),p=a.map(u=>`
      <li class="dropdown-option${u.value===n?" selected":""}" role="option"
          aria-selected="${u.value===n}" data-value="${s(u.value)}">
        ${s(u.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${s(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${s(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${p}</ul>
    </div>
  `}function ee(t){t.querySelectorAll(".dropdown.open").forEach(a=>{var n;a.classList.remove("open"),(n=a.querySelector(".dropdown-trigger"))==null||n.setAttribute("aria-expanded","false")})}function we(t,a){t.addEventListener("click",p=>{const u=p.target,y=u.closest(".dropdown-trigger");if(y&&t.contains(y)){const T=y.closest(".dropdown"),H=!!T&&!T.classList.contains("open");ee(t),T&&H&&(T.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const f=u.closest(".dropdown-option");if(f&&t.contains(f)){const T=f.closest(".dropdown");ee(t),a((T==null?void 0:T.dataset.dropdown)??"",f.dataset.value??"");return}ee(t)});const n=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}const u=p.target;(!u.closest(".dropdown")||!t.contains(u))&&ee(t)},e=p=>{if(!t.isConnected){document.removeEventListener("click",n),document.removeEventListener("keydown",e);return}p.key==="Escape"&&ee(t)};document.addEventListener("click",n),document.addEventListener("keydown",e)}const _e=85,ie={exec:"Execution",beacon:"Beacon"};function ze(t,a){let n=!1,e=null,p=null,u=null,y=null,f=null,T=null,H=null,I=null;const R={exec:null,beacon:null};let w=null;t.innerHTML=`<h1>Dashboard: ${s(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${N()}</div>`;const P=t.querySelector("#dash-body"),l=t.querySelector("#dash-footer");P.addEventListener("click",r=>{const g=r.target.closest("[data-action]");if(!g||!P.contains(g))return;const v=g.dataset.action;if(v==="svc-action"){const b=g.dataset.svc,E=g.dataset.kind;b&&E&&k(b,E)}else if(v==="open-clear"){const b=g.dataset.svc;b&&L(b)}else if(v==="copy"){const b=g.dataset.copy;b&&S(g,b)}else v==="retry-du"?o():v==="retry-endpoints"&&d()}),i();async function i(){let r,g;try{const[b,E]=await Promise.all([X(),V()]);r=b.find(q=>q.id===a),g=E}catch(b){if(n)return;P.innerHTML=`<p class="error">Failed to load target: ${s(String(b))}</p>`;return}if(n)return;if(!r){P.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!r.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const v=g==null?void 0:g.networks.find(b=>b.ChainID===r.wire.ChainID);v&&(l.innerHTML=N(v.Name,v.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=Ce(a,b=>{n||($(b),p=b,u=b,x())}),o(),d()}async function o(){T=null;try{f=await Be(a)}catch(r){f=null,T=String(r instanceof Error?r.message:r)}n||x()}async function d(){I=null;try{H=await Ae(a)}catch(r){H=null,I=String(r instanceof Error?r.message:r)}n||x()}function $(r){if(!p)return;const g=(new Date(r.at).getTime()-new Date(p.at).getTime())/1e3,v=r.execHead-p.execHead;if(g>0&&v>=0){const b=v/g;y=y===null?b:y*.7+b*.3}}function x(){if(!u)return;const r=u;P.innerHTML=`
      <div class="card-grid">
        ${U(r)}
        ${K(r)}
        ${_(r)}
        ${O(r)}
        ${G(r)}
        ${F()}
        ${h(r)}
      </div>
      <p class="muted small">Last updated ${s(new Date(r.at).toLocaleTimeString())}</p>
    `}function D(r){const v=r.refHead>0?r.refHead-r.execHead:null,b=v!==null&&v>0&&y&&y>0?je(v/y):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function U(r){const{lag:g,eta:v}=D(r);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${r.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${W(r.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${g!==null?W(r.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${g!==null?W(Math.max(g,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function K(r){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${r.beaconDistance===0?B("synced","ok"):B("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${W(r.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${W(r.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function _(r){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${W(r.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${W(r.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function O(r){const g=r.diskUsedPct>=_e;return`
      <div class="card ${g?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${g?"meter-warn":""}" style="width:${Math.min(r.diskUsedPct,100)}%"></div></div>
        <p>${Fe(r.diskUsedPct)} used</p>
      </div>
    `}function G(r){if(T)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${s(T)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const g=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,v=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:b,eta:E}=D(r),q=b!==null&&b>0&&y!==null&&y>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Y(f.ExecBytes)} of ~${Y(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        ${q?`<p class="muted small">Estimated time remaining: ${s(E)}</p>`:""}
        <p class="muted small">Beacon — ${Y(f.BeaconBytes)} of ~${Y(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Y(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function F(){if(I)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(I)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!H)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const r=H,g=r.ExecReachable&&!r.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=r.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${s(r.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ge(r.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ge(r.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.BeaconHTTP)}">Copy</button>
        </div>
        ${g}
        ${v}
      </div>
    `}function c(r,g){const v=ie[r],b=R[r],E=(q,oe,ne)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${r}" data-kind="${q}" ${b!==null||ne?"disabled":""}>${b===q?m():s(oe)}</button>`;return`
      <div class="service-row">
        <span>${s(v)} ${g?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${E("start","Start",g)}
          ${E("stop","Stop",!g)}
          ${E("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${r}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function h(r){return`
      <div class="card">
        <h3>Services</h3>
        ${c("exec",r.execActive)}
        ${c("beacon",r.beaconActive)}
        ${w?`<p class="error small">${s(w)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(a)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(a)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(a)}">Diagnostics →</a>
        </p>
      </div>
    `}function m(){return'<span class="spinner" aria-label="working"></span>'}async function k(r,g){if(R[r]===null){R[r]=g,w=null,x();try{await He(a,r,g)}catch(v){w=`${ie[r]} ${g} failed: ${v instanceof Error?v.message:String(v)}`}R[r]=null,n||x()}}async function S(r,g){const v=await ue(g),b=r.textContent;r.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{n||(r.textContent=b)},1500)}function L(r){const g=ie[r],v=f?Y(r==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";M(`
        <h2>Clear ${s(g)} data</h2>
        <p class="error">
          This stops the ${s(g.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(v)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(r)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,q=>{if(q==="cancel"){z();return}q==="confirm"&&C(r)});const b=document.getElementById("clear-confirm-input"),E=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{E&&(E.disabled=b.value.trim()!==r)}),b==null||b.focus()}async function C(r){const g=document.getElementById("clear-confirm-btn");g&&(g.disabled=!0,g.textContent="Clearing…");try{await Re(a,r),z(),o()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const E=document.createElement("p");E.className="error small",E.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(E)}g&&(g.disabled=!1,g.textContent="Clear and resync")}}function M(r,g){z();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${r}</div>`,v.addEventListener("click",b=>{const E=b.target.closest("[data-modal-action]");E!=null&&E.dataset.modalAction&&g(E.dataset.modalAction),b.target===v&&g("cancel")}),document.body.appendChild(v)}function z(){var r;(r=document.getElementById("clear-modal"))==null||r.remove()}return()=>{n=!0,e==null||e(),z()}}const be=500,ye="valve-node.explain-consent";function We(t,a){let n=!1,e=null;const p=[];t.innerHTML=`
    <h1>Logs: ${s(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${N()}</div>
  `;const u=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");Z(t,i=>{i==="explain"&&I()}),f();async function f(){let i,o;try{const[$,x]=await Promise.all([X(),V()]);i=$.find(D=>D.id===a),o=x}catch($){if(n)return;u.innerHTML=`<p class="error">Failed to load target: ${s(String($))}</p>`;return}if(n)return;if(!i){u.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const d=o==null?void 0:o.networks.find($=>$.ChainID===i.wire.ChainID);d&&(y.innerHTML=N(d.Name,d.LearnURL));try{const $=await Le(a,200);if(n)return;p.push(...$)}catch($){if(n)return;u.innerHTML=`<p class="error">Failed to load logs: ${s(String($))}</p>`;return}T(),e=Ie(a,$=>{n||(p.push($),p.length>be&&p.splice(0,p.length-be),T())})}function T(){const i=p.filter(d=>d.severity==="error"||d.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${p.map(H).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${B(String(i.length),i.length?"bad":"neutral")}</h2>
          <div class="log-lines">${i.length?i.slice().reverse().map(H).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=u.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function H(i){const o=i.severity||"info",d=i.learnUrl?` <a href="${s(i.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(o)}">
        <span class="log-time">${s(new Date(i.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(i.unit)}</span>
        <span class="log-sev">${s(o)}</span>
        <span class="log-text">${s(i.line)}</span>
        ${i.explain?`<div class="log-explain">${s(i.explain)}${d}</div>`:""}
      </div>
    `}async function I(){const i=p.filter(d=>d.severity==="error"||d.severity==="critical").map(d=>d.line).slice(-40);if(!(localStorage.getItem(ye)==="1")){R(i);return}await w(i)}function R(i){const o=i.length?`<pre class="explain-excerpt">${i.map(d=>s(d)).join(`
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
    `,d=>{d==="proceed"?(localStorage.setItem(ye,"1"),l(),w(i)):l()})}async function w(i){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=i.length?await fe(a,i):await fe(a);if(n)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(d=>s(d)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,d=>{d==="close"&&l()})}catch(o){if(n)return;if(o instanceof de&&o.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,d=>{d==="close"&&l()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${s(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,d=>{d==="close"&&l()})}}function P(i,o){l();const d=document.createElement("div");d.className="modal-overlay",d.id="explain-modal",d.innerHTML=`<div class="modal">${i}</div>`,d.addEventListener("click",$=>{const x=$.target.closest("[data-modal-action]");x!=null&&x.dataset.modalAction&&o(x.dataset.modalAction),$.target===d&&o("cancel")}),document.body.appendChild(d)}function l(){var i;(i=document.getElementById("explain-modal"))==null||i.remove()}return()=>{n=!0,e==null||e(),l()}}function Je(t,a){let n=!1,e=null,p=null,u=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${s(a)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${N()}</div>`;const f=t.querySelector("#diag-body"),T=t.querySelector("#diag-footer");Z(t,(o,d)=>{var $;if(o==="run")I();else if(o==="toggle")($=d.closest(".check-item"))==null||$.classList.toggle("expanded");else if(o==="copy"){const x=d.dataset.copy;x&&i(d,x)}}),H();async function H(){let o,d;try{const[x,D]=await Promise.all([X(),V()]);o=x.find(U=>U.id===a),d=D}catch(x){if(n)return;f.innerHTML=`<p class="error">Failed to load target: ${s(String(x))}</p>`;return}if(n)return;if(!o){f.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const $=d==null?void 0:d.networks.find(x=>x.ChainID===o.wire.ChainID);$&&(T.innerHTML=N($.Name,$.LearnURL));try{e=await Ne(a),y=!0}catch(x){p=String(x instanceof Error?x.message:x)}n||R()}async function I(){u=!0,p=null,R();try{e=await Me(a),y=!0}catch(o){p=String(o instanceof Error?o.message:o)}u=!1,n||R()}function R(){f.innerHTML=`
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
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${w()}
    `}function w(){if(!y&&!p)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),d=e.failedId?`<p><strong>Failed at: ${s(P(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${s(o)} — trigger: ${s(e.trigger)}</p>
      ${d}
      <ul class="check-list">${e.items.map(l).join("")}</ul>
    `}function P(o){var d;return((d=e==null?void 0:e.items.find($=>$.ID===o))==null?void 0:d.Title)??o}function l(o){const d=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",$=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${$?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${B($?"failed here":o.Status,d)}
          <strong>${s(o.Title)}</strong>
          <span class="muted small check-detail-inline">${s(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${$?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${s(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${$?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function i(o,d){const $=await ue(d),x=o.textContent;o.textContent=$?"Copied!":"Copy failed",setTimeout(()=>{n||(o.textContent=x)},1500)}return()=>{n=!0}}function Ke(t,a){let n=!1,e=[],p=null,u=!1,y=!1;t.innerHTML=`<h1>Security: ${s(a)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${N()}</div>`;const f=t.querySelector("#sec-body"),T=t.querySelector("#sec-footer");Z(t,(l,i)=>{var o;if(l==="rerun")I();else if(l==="toggle")(o=i.closest(".check-item"))==null||o.classList.toggle("expanded");else if(l==="copy"){const d=i.dataset.copy;d&&P(i,d)}}),H();async function H(){let l,i;try{const[d,$]=await Promise.all([X(),V()]);l=d.find(x=>x.id===a),i=$}catch(d){if(n)return;f.innerHTML=`<p class="error">Failed to load target: ${s(String(d))}</p>`;return}if(n)return;if(!l){f.innerHTML=`<p class="error">Target "${s(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const o=i==null?void 0:i.networks.find(d=>d.ChainID===l.wire.ChainID);o&&(T.innerHTML=N(o.Name,o.LearnURL)),await I()}async function I(){u=!0,p=null,R();try{e=await De(a),y=!0}catch(l){p=String(l instanceof Error?l.message:l)}u=!1,n||R()}function R(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(a)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${p?`<p class="error">${s(p)}</p>`:""}
      ${!y&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(w).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function w(l){const i=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(l.Status,i)}
          <strong>${s(l.Title)}</strong>
          <span class="muted small check-detail-inline">${s(l.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${s(l.Why)}</p>
          </details>
          ${l.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(l.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(l.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(l,i){const o=await ue(i),d=l.textContent;l.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{n||(l.textContent=d)},1500)}return()=>{n=!0}}const Ge=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Ye(t){let a=!1,n=!1,e=!1,p=null,u=!1,y=null,f=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${N()}`;const T=t.querySelector("#settings-body");Z(t,w=>{if(w==="save"&&R(),w==="clear-key"){if(!y)return;n=!0;const P=t.querySelector("#ai-key");P&&(P.value=""),I(y)}}),we(t,(w,P)=>{w!=="ai-provider"||!y||(f=P,u=!1,I(y))}),H();async function H(){try{const w=await Ue();if(a)return;y=w,I(w)}catch(w){if(a)return;T.innerHTML=`<p class="error">Failed to load settings: ${s(String(w))}</p>`}}function I(w){var i;const P=f??w.aiProvider;T.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${le("ai-provider",Ge.map(o=>({value:o.value,label:o.label})),P)}
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
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const l=t.querySelector("#ai-key");l==null||l.addEventListener("input",()=>{n=!0,u=!1}),(i=t.querySelector("#ref-rpc-base"))==null||i.addEventListener("input",()=>{u=!1})}async function R(){const w=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!w||!P||!y)return;const l={aiProvider:f??y.aiProvider,refRpcBase:P.value.trim()};n&&(l.aiKey=w.value),e=!0,p=null,u=!1,I(y);try{const i=await qe(l);if(a)return;y=i,n=!1,e=!1,u=!0,I(i)}catch(i){if(a)return;e=!1,p=String(i instanceof Error?i.message:i),I(y)}}return()=>{a=!0}}const Ve="local";function Xe(t){let a=!1,n=!1,e=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${N()}
  `;const p=t.querySelector("#targets-body");Z(t,(l,i)=>{T(l,i)}),u();async function u(){try{const[l,i]=await Promise.all([X(),V()]);if(a)return;f(l,i)}catch(l){if(a)return;p.innerHTML=`<p class="error">Failed to load machines: ${s(String(l))}</p>`}}function y(){e&&f(e.targets,e.catalog)}function f(l,i){e={targets:l,catalog:i};const o=!et(),d=[...l].sort((D,U)=>(D.mode==="local"?-1:0)-(U.mode==="local"?-1:0)),$=d.length?`<div class="card-grid">${d.map(D=>Ze(D,i)).join("")}</div>`:`
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
    `;p.innerHTML=`
      <section class="section">
        <div class="section-head">
          <h2>Your machines</h2>
          ${x}
        </div>
        ${n?Qe():""}
        ${$}
      </section>
    `}async function T(l,i){var o;if(l==="add-local"){await H();return}if(l==="delete-target"){const d=i.dataset.id;if(!d||!confirm(`Remove target "${d}"? This does not touch anything already running on it.`))return;await I(d);return}if(l==="toggle-ssh"){n=!n,P(),y(),n&&((o=t.querySelector("#ssh-host"))==null||o.focus());return}l==="add-ssh"&&await R()}async function H(){P();try{await he({id:Ve,mode:"local"}),await u()}catch(l){w(l)}}async function I(l){try{await ke(l),await u()}catch(i){w(i)}}async function R(){const l=t.querySelector("#ssh-host"),i=t.querySelector("#ssh-user"),o=t.querySelector("#ssh-key"),d=t.querySelector("#ssh-port"),$=t.querySelector("#ssh-id");if(!l||!i||!o||!d||!$)return;const x=l.value.trim(),D=i.value.trim(),U=o.value.trim(),K=d.value.trim(),_=$.value.trim();if(P(),!x||!D||!U){w(new Error("host, user, and key path are required"));return}const O=_||tt(x),G={Host:x,User:D,KeyPath:U};if(K){const c=Number.parseInt(K,10);if(!Number.isFinite(c)||c<=0){w(new Error("port must be a positive number"));return}G.Port=c}const F=t.querySelector("#ssh-submit");F&&(F.disabled=!0,F.textContent="Connecting…");try{await he({id:O,mode:"ssh",ssh:G}),n=!1,await u()}catch(c){w(c),F&&(F.disabled=!1,F.textContent="Add server")}}function w(l){let i=t.querySelector("#targets-error");i||(p.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),i=t.querySelector("#targets-error")),i.textContent=String(l instanceof Error?l.message:l)}function P(){var l;(l=t.querySelector("#targets-error"))==null||l.remove()}return()=>{a=!0}}function Ze(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",p=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let u,y;if(!n)u=B("not set up","neutral"),y=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const f=a.networks.find(H=>H.ChainID===n.ChainID),T=f?f.Name:`chain ${n.ChainID}`;u=`${B(T,"ok")} ${B(n.ExecID,"neutral")} ${B(n.BeaconID,"neutral")}${n.Archive?" "+B("archive","warn"):""}`,y=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${s(t.id)}</h2>
      <p class="muted">${p}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${y}
        <button class="btn btn-danger" data-action="delete-target" data-id="${s(t.id)}">Remove</button>
      </div>
    </div>
  `}function Qe(){return`
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
  `}function et(){const t=navigator.userAgentData,a=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(a)&&!/linux|android/i.test(a)}function tt(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ce=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ae=8545,re=5052,se=30303,nt=[369,943,1],$e={369:"default",943:"practise here first"};function at(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${N()}</div>`;const p=t.querySelector("#wizard-body"),u=t.querySelector("#wizard-footer");Z(t,(c,h)=>{x(c,h)}),we(t,(c,h)=>{c==="exec-select"?e.execId=h:c==="beacon-select"&&(e.beaconId=h),f()}),y();async function y(){try{const[c,h]=await Promise.all([V(),X()]);if(n)return;e.catalog=c;const m=h.find(k=>k.id===a);m!=null&&m.wire&&(e.chainId=m.wire.ChainID,e.execId=m.wire.ExecID,e.beaconId=m.wire.BeaconID,e.archive=m.wire.Archive,m.wire.ExecHTTPPort&&(e.execHTTPPort=String(m.wire.ExecHTTPPort)),m.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(m.wire.BeaconHTTPPort)),m.wire.ExecP2PPort&&(e.execP2PPort=String(m.wire.ExecP2PPort)),m.wire.RPCBindAddr&&(e.rpcBindAddr=m.wire.RPCBindAddr)),f()}catch(c){if(n)return;e.loadError=String(c instanceof Error?c.message:c),f()}}function f(){if(e.loadError){p.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(p.innerHTML=`
      ${F(e.step)}
      ${H()}
    `,T())}function T(){var h;const c=(h=e.catalog)==null?void 0:h.networks.find(m=>m.ChainID===e.chainId);u.innerHTML=c?N(c.Name,c.LearnURL):N()}function H(){switch(e.step){case"network":return I();case"clients":return R();case"mode":return o();case"review":return d();case"run":return $()}}function I(){const c=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${nt.map(m=>{const k=c.networks.find(C=>C.ChainID===m);if(!k)return"";const S=e.chainId===m,L=$e[m]?B($e[m],m===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${S?"selected":""}" data-action="pick-network" data-chain-id="${m}" type="button">
          <h3>${s(k.Name)} <span class="muted">(chain ${m})</span></h3>
          ${L}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function R(){const c=e.catalog,h=c.networks.find(S=>S.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const m=h.ExecClients.map(S=>P(S,c)),k=h.BeaconClients.map(S=>P(S,c));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${s(h.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${le("exec-select",m,e.execId)}
        </label>
        ${i(e.execId,c)}
        <label>
          Beacon client
          ${le("beacon-select",k,e.beaconId)}
        </label>
        ${i(e.beaconId,c)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function w(c){return c<=0?"—":c>=1?`~${c.toFixed(1)} TB`:`~${Math.round(c*1e3)} GB`}function P(c,h){const m=h.clients.find(k=>k.id===c);return{value:c,label:m?`${m.id} — ${l(m.repo)}`:c}}function l(c){const h=c.split("/");return h.length>=4?h[3]:c}function i(c,h){const m=c?h.clients.find(S=>S.id===c):void 0;if(!m)return"";const k=m.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${s(m.repo)}" target="_blank" rel="noopener noreferrer">${s(k)}</a></p>`}function o(){var C;const c=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"",h=(C=e.catalog)==null?void 0:C.networks.find(M=>M.ChainID===e.chainId),m=(h==null?void 0:h.ArchiveSizeTB)??0,k=h?w(m/2):"Smaller",S=h?w(m):"Much larger",L=h?` on ${s(h.Name)}`:"";return`
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
            <tr><th>Approx. disk footprint${L}</th><td class="yes">${k}</td><td class="limited">${S}</td></tr>
            <tr><th>Initial sync time${L}</th><td class="yes">${h?s(h.SyncLabel):"Faster"}</td><td class="limited">${h?s(h.GenesisSyncLabel):"Much slower"}</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <p class="muted small">
          Disk sizes and sync times are rough baselines — both vary by client and scale with the
          target's CPU and disk speed.
        </p>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${S}${h?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${k}${h?"":" disk"}</span>
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${s(c)})</span>
            <input id="data-dir-input" type="text" placeholder="${s(c)}" value="${s(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${s(c)}/jwt.hex" value="${s(e.jwtPath)}" />
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
    `}function d(){const h=e.catalog.networks.find(v=>v.ChainID===e.chainId),m=e.dataDir||`/var/lib/valve-node/${e.chainId}`,k=e.jwtPath||`${m}/jwt.hex`,S=ce.map(v=>`<li>${s(v.title)}</li>`).join(""),L=O(e.execHTTPPort,ae),C=O(e.beaconHTTPPort,re),M=O(e.execP2PPort,se),z=L||C||M?`<tr><th>Non-default ports</th><td>${[L?`exec HTTP ${L}`:null,C?`beacon HTTP ${C}`:null,M?`exec p2p ${M}`:null].filter(v=>v!==null).map(s).join(", ")}</td></tr>`:"",{addr:r}=U(e.rpcBindAddr),g=r?`<tr><th>RPC bind address</th><td><code>${s(r)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${s(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${s((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${s(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${s(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${s(m)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${s(k)}</code></td></tr>
            ${h?`<tr><th>Checkpoint sync</th><td><code>${s(h.CheckpointURL)}</code></td></tr>`:""}
            ${z}
            ${g}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${S}</ol>
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function $(){const h=e.catalog.networks.find(r=>r.ChainID===e.chainId),m=h==null?void 0:h.LearnURL,k=new Set(e.events.filter(r=>r.done).map(r=>r.stepId)),S=new Set(e.events.filter(r=>r.err).map(r=>r.stepId)),L=new Map;for(const r of e.events){if(!r.line)continue;const g=L.get(r.stepId)??[];g.push(r.line),L.set(r.stepId,g)}const C=ce.map(r=>{var ne;const g=k.has(r.id),v=S.has(r.id),b=v?B("failed","bad"):g?B("done","ok"):B("pending","neutral"),E=(L.get(r.id)??[]).slice(-5),q=(ne=e.events.find(Q=>Q.stepId===r.id&&Q.err))==null?void 0:ne.err,oe=r.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${m?` <a href="${s(m)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${g?"step-done":""} ${v?"step-error":""}">
          <div class="step-head">${b} <strong>${s(r.title)}</strong></div>
          ${oe}
          ${E.length?`<pre class="step-log">${E.map(Q=>s(Q)).join(`
`)}</pre>`:""}
          ${q?`<p class="error small">${s(q)}</p>`:""}
        </li>
      `}).join(""),M=e.events.some(r=>r.err),z=ce.every(r=>k.has(r.id))||e.events.some(r=>r.stepId==="handshake"&&r.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${C}</ol>
        ${z&&!M?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${M?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function x(c,h){switch(c){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,f();break;case"goto-network":e.step="network",f();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",f();break;case"goto-mode":e.step="mode",f();break;case"goto-review":if(D(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError){f();break}e.step="review",f();break;case"start-setup":G();break}}function D(){const c=t.querySelectorAll('input[name="mode"]');for(const M of Array.from(c))M.checked&&(e.archive=M.value==="archive");const h=t.querySelector("#data-dir-input"),m=t.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),m&&(e.jwtPath=m.value.trim());const k=t.querySelector("#exec-http-port-input"),S=t.querySelector("#beacon-http-port-input"),L=t.querySelector("#exec-p2p-port-input");k&&(e.execHTTPPort=k.value.trim()),S&&(e.beaconHTTPPort=S.value.trim()),L&&(e.execP2PPort=L.value.trim());const C=t.querySelector("#rpc-bind-addr-input");C&&(e.rpcBindAddr=C.value.trim()),e.execHTTPPortError=_(e.execHTTPPort).error??null,e.beaconHTTPPortError=_(e.beaconHTTPPort).error??null,e.execP2PPortError=_(e.execP2PPort).error??null,e.rpcBindAddrError=U(e.rpcBindAddr).error??null}function U(c){if(!c)return{};const h=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(c);return h?h.slice(1).every(m=>Number(m)<=255)?{addr:c}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(c)&&c.includes(":")?{addr:c}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const K=/^\d+$/;function _(c){if(!c)return{};if(!K.test(c))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const h=Number(c);return!Number.isInteger(h)||h<1||h>65535?{error:"Port must be between 1 and 65535."}:{port:h}}function O(c,h){const{port:m}=_(c);if(!(m===void 0||m===h))return m}async function G(){var L;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,f();const c={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(c.DataDir=e.dataDir),e.jwtPath&&(c.JWTPath=e.jwtPath);const h=O(e.execHTTPPort,ae),m=O(e.beaconHTTPPort,re),k=O(e.execP2PPort,se);h!==void 0&&(c.ExecHTTPPort=h),m!==void 0&&(c.BeaconHTTPPort=m),k!==void 0&&(c.ExecP2PPort=k);const{addr:S}=U(e.rpcBindAddr);S!==void 0&&(c.RPCBindAddr=S);try{await Se(e.targetId,c)}catch(C){if(!(C instanceof de&&C.status===409)){e.starting=!1,e.startError=String(C instanceof Error?C.message:C),f();return}}e.starting=!1,e.step="run",e.events=[],f(),(L=e.streamStop)==null||L.call(e),e.streamStop=Ee(e.targetId,C=>{n||(e.events.push(C),e.step==="run"&&f())})}function F(c){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],k=h.map(S=>S.id).indexOf(c);return`
      <ol class="wizard-progress">
        ${h.map((S,L)=>`<li class="${L===k?"current":L<k?"past":"future"}">${s(S.label)}</li>`).join("")}
      </ol>
    `}return()=>{var c;n=!0,(c=e.streamStop)==null||c.call(e)}}const rt=document.querySelector("#app"),{contentEl:st,setActiveNav:ot}=Oe(rt);let j=null;function it(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"||n==="security"||n==="diag"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function J(t){const a=document.createElement("div");return st.replaceChildren(a),t(a)}function xe(){if(j){try{j()}catch{}j=null}const{screen:t,id:a}=it();switch(ot(t),t){case"setup":if(!a){location.hash="#/targets";return}j=J(n=>at(n,a));break;case"dash":if(!a){location.hash="#/targets";return}j=J(n=>ze(n,a));break;case"logs":if(!a){location.hash="#/targets";return}j=J(n=>We(n,a));break;case"security":if(!a){location.hash="#/targets";return}j=J(n=>Ke(n,a));break;case"diag":if(!a){location.hash="#/targets";return}j=J(n=>Je(n,a));break;case"settings":j=J(n=>Ye(n));break;case"targets":default:j=J(n=>Xe(n));break}}window.addEventListener("hashchange",xe);xe();
