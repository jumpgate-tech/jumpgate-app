var be=Object.defineProperty;var ye=(t,n,a)=>n in t?be(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var ce=(t,n,a)=>ye(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const p of u)if(p.type==="childList")for(const w of p.addedNodes)w.tagName==="LINK"&&w.rel==="modulepreload"&&e(w)}).observe(document,{childList:!0,subtree:!0});function a(u){const p={};return u.integrity&&(p.integrity=u.integrity),u.referrerPolicy&&(p.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?p.credentials="include":u.crossOrigin==="anonymous"?p.credentials="omit":p.credentials="same-origin",p}function e(u){if(u.ep)return;u.ep=!0;const p=a(u);fetch(u.href,p)}})();function G(){return D("/api/catalog")}function V(){return D("/api/targets")}function le(t){return D("/api/targets",{method:"POST",headers:ee,body:JSON.stringify(t)})}function $e(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function we(t,n){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:ee,body:JSON.stringify(n)})}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Te(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Pe(t,n=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Se(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function de(t,n){const a=n===void 0?{}:{lines:n};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:ee,body:JSON.stringify(a)})}function Ee(t,n,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function ke(t,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:ee,body:JSON.stringify({Confirm:n})})}function Ce(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Le(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Ie(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function He(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Re(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Be(){return D("/api/settings")}function Ae(t){return D("/api/settings",{method:"PUT",headers:ee,body:JSON.stringify(t)})}class oe extends Error{constructor(a,e){super(e);ce(this,"status");this.name="ApiError",this.status=a}}const ee={"Content-Type":"application/json"};async function D(t,n){const a=await fetch(t,n);if(!a.ok){let u=a.statusText||`HTTP ${a.status}`;try{const p=await a.json();p&&typeof p.error=="string"&&p.error&&(u=p.error)}catch{}throw new oe(a.status,u)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const De="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function U(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${r(n)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(De)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function Ne(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:u=>{for(const p of a)p.classList.toggle("active",p.dataset.nav===u)}}}function z(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Me(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ue(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const u=Math.floor(a/24),p=a%24;return`~${u}d ${p}h`}function B(t,n){return`<span class="badge badge-${n}">${r(t)}</span>`}function ue(t){return`<span class="dot dot-${t}"></span>`}const pe=["B","KB","MB","GB","TB","PB"];function K(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<pe.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${pe[a]}`}async function ie(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Y(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const u=e.dataset.action;u&&n(u,e,a)})}const qe=85,re={exec:"Execution",beacon:"Beacon"};function Oe(t,n){let a=!1,e=null,u=null,p=null,w=null,m=null,R=null,L=null,A=null;const x={exec:null,beacon:null};let S=null;t.innerHTML=`<h1>Dashboard: ${r(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${U()}</div>`;const P=t.querySelector("#dash-body"),l=t.querySelector("#dash-footer");P.addEventListener("click",s=>{const g=s.target.closest("[data-action]");if(!g||!P.contains(g))return;const v=g.dataset.action;if(v==="svc-action"){const b=g.dataset.svc,C=g.dataset.kind;b&&C&&I(b,C)}else if(v==="open-clear"){const b=g.dataset.svc;b&&j(b)}else if(v==="copy"){const b=g.dataset.copy;b&&H(g,b)}else v==="retry-du"?o():v==="retry-endpoints"&&c()}),i();async function i(){let s,g;try{const[b,C]=await Promise.all([V(),G()]);s=b.find(O=>O.id===n),g=C}catch(b){if(a)return;P.innerHTML=`<p class="error">Failed to load target: ${r(String(b))}</p>`;return}if(a)return;if(!s){P.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const v=g==null?void 0:g.networks.find(b=>b.ChainID===s.wire.ChainID);v&&(l.innerHTML=U(v.Name,v.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=Te(n,b=>{a||(y(b),u=b,p=b,$())}),o(),c()}async function o(){R=null;try{m=await Ce(n)}catch(s){m=null,R=String(s instanceof Error?s.message:s)}a||$()}async function c(){A=null;try{L=await Le(n)}catch(s){L=null,A=String(s instanceof Error?s.message:s)}a||$()}function y(s){if(!u)return;const g=(new Date(s.at).getTime()-new Date(u.at).getTime())/1e3,v=s.execHead-u.execHead;if(g>0&&v>=0){const b=v/g;w=w===null?b:w*.7+b*.3}}function $(){if(!p)return;const s=p;P.innerHTML=`
      <div class="card-grid">
        ${M(s)}
        ${F(s)}
        ${X(s)}
        ${Z(s)}
        ${d(s)}
        ${h()}
        ${E(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function N(s){const v=s.refHead>0?s.refHead-s.execHead:null,b=v!==null&&v>0&&w&&w>0?Ue(v/w):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function M(s){const{lag:g,eta:v}=N(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${z(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${g!==null?z(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${g!==null?z(Math.max(g,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function F(s){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${s.beaconDistance===0?B("synced","ok"):B("syncing","warn")}</p>
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
    `}function Z(s){const g=s.diskUsedPct>=qe;return`
      <div class="card ${g?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${g?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Me(s.diskUsedPct)} used</p>
      </div>
    `}function d(s){if(R)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(R)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const g=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,v=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:b,eta:C}=N(s),O=b!==null&&b>0&&w!==null&&w>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${K(m.ExecBytes)} of ~${K(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${g}%"></div></div>
        ${O?`<p class="muted small">Estimated time remaining: ${r(C)}</p>`:""}
        <p class="muted small">Beacon — ${K(m.BeaconBytes)} of ~${K(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${K(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function h(){if(A)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(A)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!L)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=L,g=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=s.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${r(s.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ue(s.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ue(s.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.BeaconHTTP)}">Copy</button>
        </div>
        ${g}
        ${v}
      </div>
    `}function f(s,g){const v=re[s],b=x[s],C=(O,W,ve)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${O}" ${b!==null||ve?"disabled":""}>${b===O?k():r(W)}</button>`;return`
      <div class="service-row">
        <span>${r(v)} ${g?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${C("start","Start",g)}
          ${C("stop","Stop",!g)}
          ${C("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function E(s){return`
      <div class="card">
        <h3>Services</h3>
        ${f("exec",s.execActive)}
        ${f("beacon",s.beaconActive)}
        ${S?`<p class="error small">${r(S)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(n)}">Diagnostics →</a>
        </p>
      </div>
    `}function k(){return'<span class="spinner" aria-label="working"></span>'}async function I(s,g){if(x[s]===null){x[s]=g,S=null,$();try{await Ee(n,s,g)}catch(v){S=`${re[s]} ${g} failed: ${v instanceof Error?v.message:String(v)}`}x[s]=null,a||$()}}async function H(s,g){const v=await ie(g),b=s.textContent;s.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{a||(s.textContent=b)},1500)}function j(s){const g=re[s],v=m?K(s==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";T(`
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
      `,O=>{if(O==="cancel"){q();return}O==="confirm"&&Q(s)});const b=document.getElementById("clear-confirm-input"),C=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{C&&(C.disabled=b.value.trim()!==s)}),b==null||b.focus()}async function Q(s){const g=document.getElementById("clear-confirm-btn");g&&(g.disabled=!0,g.textContent="Clearing…");try{await ke(n,s),q(),o()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const C=document.createElement("p");C.className="error small",C.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(C)}g&&(g.disabled=!1,g.textContent="Clear and resync")}}function T(s,g){q();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${s}</div>`,v.addEventListener("click",b=>{const C=b.target.closest("[data-modal-action]");C!=null&&C.dataset.modalAction&&g(C.dataset.modalAction),b.target===v&&g("cancel")}),document.body.appendChild(v)}function q(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{a=!0,e==null||e(),q()}}const he=500,fe="valve-node.explain-consent";function Fe(t,n){let a=!1,e=null;const u=[];t.innerHTML=`
    <h1>Logs: ${r(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${U()}</div>
  `;const p=t.querySelector("#logs-body"),w=t.querySelector("#logs-footer");Y(t,i=>{i==="explain"&&A()}),m();async function m(){let i,o;try{const[y,$]=await Promise.all([V(),G()]);i=y.find(N=>N.id===n),o=$}catch(y){if(a)return;p.innerHTML=`<p class="error">Failed to load target: ${r(String(y))}</p>`;return}if(a)return;if(!i){p.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const c=o==null?void 0:o.networks.find(y=>y.ChainID===i.wire.ChainID);c&&(w.innerHTML=U(c.Name,c.LearnURL));try{const y=await Pe(n,200);if(a)return;u.push(...y)}catch(y){if(a)return;p.innerHTML=`<p class="error">Failed to load logs: ${r(String(y))}</p>`;return}R(),e=Se(n,y=>{a||(u.push(y),u.length>he&&u.splice(0,u.length-he),R())})}function R(){const i=u.filter(c=>c.severity==="error"||c.severity==="critical");p.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(L).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${B(String(i.length),i.length?"bad":"neutral")}</h2>
          <div class="log-lines">${i.length?i.slice().reverse().map(L).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=p.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function L(i){const o=i.severity||"info",c=i.learnUrl?` <a href="${r(i.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(i.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(i.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(i.line)}</span>
        ${i.explain?`<div class="log-explain">${r(i.explain)}${c}</div>`:""}
      </div>
    `}async function A(){const i=u.filter(c=>c.severity==="error"||c.severity==="critical").map(c=>c.line).slice(-40);if(!(localStorage.getItem(fe)==="1")){x(i);return}await S(i)}function x(i){const o=i.length?`<pre class="explain-excerpt">${i.map(c=>r(c)).join(`
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
    `,c=>{c==="proceed"?(localStorage.setItem(fe,"1"),l(),S(i)):l()})}async function S(i){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=i.length?await de(n,i):await de(n);if(a)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(c=>r(c)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&l()})}catch(o){if(a)return;if(o instanceof oe&&o.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,c=>{c==="close"&&l()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&l()})}}function P(i,o){l();const c=document.createElement("div");c.className="modal-overlay",c.id="explain-modal",c.innerHTML=`<div class="modal">${i}</div>`,c.addEventListener("click",y=>{const $=y.target.closest("[data-modal-action]");$!=null&&$.dataset.modalAction&&o($.dataset.modalAction),y.target===c&&o("cancel")}),document.body.appendChild(c)}function l(){var i;(i=document.getElementById("explain-modal"))==null||i.remove()}return()=>{a=!0,e==null||e(),l()}}function je(t,n){let a=!1,e=null,u=null,p=!1,w=!1;t.innerHTML=`<h1>Network diagnostics: ${r(n)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${U()}</div>`;const m=t.querySelector("#diag-body"),R=t.querySelector("#diag-footer");Y(t,(o,c)=>{var y;if(o==="run")A();else if(o==="toggle")(y=c.closest(".check-item"))==null||y.classList.toggle("expanded");else if(o==="copy"){const $=c.dataset.copy;$&&i(c,$)}}),L();async function L(){let o,c;try{const[$,N]=await Promise.all([V(),G()]);o=$.find(M=>M.id===n),c=N}catch($){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String($))}</p>`;return}if(a)return;if(!o){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const y=c==null?void 0:c.networks.find($=>$.ChainID===o.wire.ChainID);y&&(R.innerHTML=U(y.Name,y.LearnURL));try{e=await Re(n),w=!0}catch($){u=String($ instanceof Error?$.message:$)}a||x()}async function A(){p=!0,u=null,x();try{e=await He(n),w=!0}catch(o){u=String(o instanceof Error?o.message:o)}p=!1,a||x()}function x(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${p?"disabled":""}>${p?"Running…":"Run diagnostics"}</button>
      </div>
      ${u?`<p class="error">${r(u)}</p>`:""}
      ${S()}
    `}function S(){if(!w&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),c=e.failedId?`<p><strong>Failed at: ${r(P(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${c}
      <ul class="check-list">${e.items.map(l).join("")}</ul>
    `}function P(o){var c;return((c=e==null?void 0:e.items.find(y=>y.ID===o))==null?void 0:c.Title)??o}function l(o){const c=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",y=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${y?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${B(y?"failed here":o.Status,c)}
          <strong>${r(o.Title)}</strong>
          <span class="muted small check-detail-inline">${r(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${y?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${r(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${y?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function i(o,c){const y=await ie(c),$=o.textContent;o.textContent=y?"Copied!":"Copy failed",setTimeout(()=>{a||(o.textContent=$)},1500)}return()=>{a=!0}}function _e(t,n){let a=!1,e=[],u=null,p=!1,w=!1;t.innerHTML=`<h1>Security: ${r(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${U()}</div>`;const m=t.querySelector("#sec-body"),R=t.querySelector("#sec-footer");Y(t,(l,i)=>{var o;if(l==="rerun")A();else if(l==="toggle")(o=i.closest(".check-item"))==null||o.classList.toggle("expanded");else if(l==="copy"){const c=i.dataset.copy;c&&P(i,c)}}),L();async function L(){let l,i;try{const[c,y]=await Promise.all([V(),G()]);l=c.find($=>$.id===n),i=y}catch(c){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(c))}</p>`;return}if(a)return;if(!l){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const o=i==null?void 0:i.networks.find(c=>c.ChainID===l.wire.ChainID);o&&(R.innerHTML=U(o.Name,o.LearnURL)),await A()}async function A(){p=!0,u=null,x();try{e=await Ie(n),w=!0}catch(l){u=String(l instanceof Error?l.message:l)}p=!1,a||x()}function x(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${p?"disabled":""}>${p?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${r(u)}</p>`:""}
      ${!w&&p?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(S).join("")}</ul>`:w?'<p class="muted">No checks returned.</p>':""}
    `}function S(l){const i=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(l.Status,i)}
          <strong>${r(l.Title)}</strong>
          <span class="muted small check-detail-inline">${r(l.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(l.Why)}</p>
          </details>
          ${l.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(l.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(l.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(l,i){const o=await ie(i),c=l.textContent;l.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{a||(l.textContent=c)},1500)}return()=>{a=!0}}const ze=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Je(t){let n=!1,a=!1,e=!1,u=null,p=!1,w=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${U()}`;const m=t.querySelector("#settings-body");Y(t,x=>{if(x==="save"&&A(),x==="clear-key"){if(!w)return;a=!0;const S=t.querySelector("#ai-key");S&&(S.value=""),L(w)}}),R();async function R(){try{const x=await Be();if(n)return;w=x,L(x)}catch(x){if(n)return;m.innerHTML=`<p class="error">Failed to load settings: ${r(String(x))}</p>`}}function L(x){var l,i;const S=ze.map(o=>`<option value="${o.value}" ${x.aiProvider===o.value?"selected":""}>${r(o.label)}</option>`).join("");m.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${S}</select>
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
        ${p?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const P=t.querySelector("#ai-key");P==null||P.addEventListener("input",()=>{a=!0,p=!1}),(l=t.querySelector("#ai-provider"))==null||l.addEventListener("change",()=>{p=!1}),(i=t.querySelector("#ref-rpc-base"))==null||i.addEventListener("input",()=>{p=!1})}async function A(){const x=t.querySelector("#ai-provider"),S=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!x||!S||!P||!w)return;const l={aiProvider:x.value,refRpcBase:P.value.trim()};a&&(l.aiKey=S.value),e=!0,u=null,p=!1,L(w);try{const i=await Ae(l);if(n)return;w=i,a=!1,e=!1,p=!0,L(i)}catch(i){if(n)return;e=!1,u=String(i instanceof Error?i.message:i),L(w)}}return()=>{n=!0}}const We="local";function Ke(t){let n=!1,a=!1,e=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${U()}
  `;const u=t.querySelector("#targets-body");Y(t,(l,i)=>{R(l,i)}),p();async function p(){try{const[l,i]=await Promise.all([V(),G()]);if(n)return;m(l,i)}catch(l){if(n)return;u.innerHTML=`<p class="error">Failed to load machines: ${r(String(l))}</p>`}}function w(){e&&m(e.targets,e.catalog)}function m(l,i){e={targets:l,catalog:i};const o=!Ye(),c=[...l].sort((N,M)=>(N.mode==="local"?-1:0)-(M.mode==="local"?-1:0)),y=c.length?`<div class="card-grid">${c.map(N=>Ge(N,i)).join("")}</div>`:`
        <div class="card empty-state">
          <p>No machines yet.</p>
          <p class="muted small">
            ${o?"Add this machine to run a node here, or add a remote Linux server over SSH.":"valve-node is running here as your <strong>controller</strong> — it drives nodes but doesn't host them. Add a Linux server over SSH to run one."}
          </p>
        </div>
      `,$=`
      <div class="add-actions">
        ${o?'<button class="btn" data-action="add-local">Add this machine</button>':""}
        <button class="btn${o?" btn-ghost":""}" data-action="toggle-ssh">
          ${a?"Cancel":"Add a server (SSH)"}
        </button>
      </div>
    `;u.innerHTML=`
      <section class="section">
        <div class="section-head">
          <h2>Your machines</h2>
          ${$}
        </div>
        ${a?Ve():""}
        ${y}
      </section>
    `}async function R(l,i){var o;if(l==="add-local"){await L();return}if(l==="delete-target"){const c=i.dataset.id;if(!c||!confirm(`Remove target "${c}"? This does not touch anything already running on it.`))return;await A(c);return}if(l==="toggle-ssh"){a=!a,P(),w(),a&&((o=t.querySelector("#ssh-host"))==null||o.focus());return}l==="add-ssh"&&await x()}async function L(){P();try{await le({id:We,mode:"local"}),await p()}catch(l){S(l)}}async function A(l){try{await $e(l),await p()}catch(i){S(i)}}async function x(){const l=t.querySelector("#ssh-host"),i=t.querySelector("#ssh-user"),o=t.querySelector("#ssh-key"),c=t.querySelector("#ssh-port"),y=t.querySelector("#ssh-id");if(!l||!i||!o||!c||!y)return;const $=l.value.trim(),N=i.value.trim(),M=o.value.trim(),F=c.value.trim(),X=y.value.trim();if(P(),!$||!N||!M){S(new Error("host, user, and key path are required"));return}const Z=X||Xe($),d={Host:$,User:N,KeyPath:M};if(F){const f=Number.parseInt(F,10);if(!Number.isFinite(f)||f<=0){S(new Error("port must be a positive number"));return}d.Port=f}const h=t.querySelector("#ssh-submit");h&&(h.disabled=!0,h.textContent="Connecting…");try{await le({id:Z,mode:"ssh",ssh:d}),a=!1,await p()}catch(f){S(f),h&&(h.disabled=!1,h.textContent="Add server")}}function S(l){let i=t.querySelector("#targets-error");i||(u.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),i=t.querySelector("#targets-error")),i.textContent=String(l instanceof Error?l.message:l)}function P(){var l;(l=t.querySelector("#targets-error"))==null||l.remove()}return()=>{n=!0}}function Ge(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",u=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let p,w;if(!a)p=B("not set up","neutral"),w=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const m=n.networks.find(L=>L.ChainID===a.ChainID),R=m?m.Name:`chain ${a.ChainID}`;p=`${B(R,"ok")} ${B(a.ExecID,"neutral")} ${B(a.BeaconID,"neutral")}${a.Archive?" "+B("archive","warn"):""}`,w=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${u}</p>
      <p>${p}</p>
      <div class="card-actions">
        ${w}
        <button class="btn btn-danger" data-action="delete-target" data-id="${r(t.id)}">Remove</button>
      </div>
    </div>
  `}function Ve(){return`
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
  `}function Ye(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Xe(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const se=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],te=8545,ne=5052,ae=30303,Ze=[369,943,1],me={369:"default",943:"practise here first"};function Qe(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${U()}</div>`;const u=t.querySelector("#wizard-body"),p=t.querySelector("#wizard-footer");Y(t,(d,h)=>{o(d,h)}),w();async function w(){try{const[d,h]=await Promise.all([G(),V()]);if(a)return;e.catalog=d;const f=h.find(E=>E.id===n);f!=null&&f.wire&&(e.chainId=f.wire.ChainID,e.execId=f.wire.ExecID,e.beaconId=f.wire.BeaconID,e.archive=f.wire.Archive,f.wire.ExecHTTPPort&&(e.execHTTPPort=String(f.wire.ExecHTTPPort)),f.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(f.wire.BeaconHTTPPort)),f.wire.ExecP2PPort&&(e.execP2PPort=String(f.wire.ExecP2PPort)),f.wire.RPCBindAddr&&(e.rpcBindAddr=f.wire.RPCBindAddr)),m()}catch(d){if(a)return;e.loadError=String(d instanceof Error?d.message:d),m()}}function m(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${Z(e.step)}
      ${L()}
    `,R())}function R(){var h;const d=(h=e.catalog)==null?void 0:h.networks.find(f=>f.ChainID===e.chainId);p.innerHTML=d?U(d.Name,d.LearnURL):U()}function L(){switch(e.step){case"network":return A();case"clients":return x();case"mode":return P();case"review":return l();case"run":return i()}}function A(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ze.map(f=>{const E=d.networks.find(H=>H.ChainID===f);if(!E)return"";const k=e.chainId===f,I=me[f]?B(me[f],f===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${k?"selected":""}" data-action="pick-network" data-chain-id="${f}" type="button">
          <h3>${r(E.Name)} <span class="muted">(chain ${f})</span></h3>
          ${I}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function x(){const d=e.catalog,h=d.networks.find(k=>k.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const f=h.ExecClients.map(k=>S(k,d,e.execId)).join(""),E=h.BeaconClients.map(k=>S(k,d,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(h.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${f}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${E}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function S(d,h,f){const E=h.clients.find(I=>I.id===d),k=E?`${E.id} (${E.toolchain})`:d;return`<option value="${r(d)}" ${d===f?"selected":""}>${r(k)}</option>`}function P(){const d=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          Full — prune old state, smaller disk footprint
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          Archive — keep full history, needs much more disk
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${r(d)})</span>
            <input id="data-dir-input" type="text" placeholder="${r(d)}" value="${r(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${r(d)}/jwt.hex" value="${r(e.jwtPath)}" />
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
    `}function l(){const h=e.catalog.networks.find(s=>s.ChainID===e.chainId),f=e.dataDir||`/var/lib/valve-node/${e.chainId}`,E=e.jwtPath||`${f}/jwt.hex`,k=se.map(s=>`<li>${r(s.title)}</li>`).join(""),I=F(e.execHTTPPort,te),H=F(e.beaconHTTPPort,ne),j=F(e.execP2PPort,ae),Q=I||H||j?`<tr><th>Non-default ports</th><td>${[I?`exec HTTP ${I}`:null,H?`beacon HTTP ${H}`:null,j?`exec p2p ${j}`:null].filter(s=>s!==null).map(r).join(", ")}</td></tr>`:"",{addr:T}=$(e.rpcBindAddr),q=T?`<tr><th>RPC bind address</th><td><code>${r(T)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(f)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(E)}</code></td></tr>
            ${h?`<tr><th>Checkpoint sync</th><td><code>${r(h.CheckpointURL)}</code></td></tr>`:""}
            ${Q}
            ${q}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${k}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function i(){const h=e.catalog.networks.find(T=>T.ChainID===e.chainId),f=h==null?void 0:h.LearnURL,E=new Set(e.events.filter(T=>T.done).map(T=>T.stepId)),k=new Set(e.events.filter(T=>T.err).map(T=>T.stepId)),I=new Map;for(const T of e.events){if(!T.line)continue;const q=I.get(T.stepId)??[];q.push(T.line),I.set(T.stepId,q)}const H=se.map(T=>{var O;const q=E.has(T.id),s=k.has(T.id),g=s?B("failed","bad"):q?B("done","ok"):B("pending","neutral"),v=(I.get(T.id)??[]).slice(-5),b=(O=e.events.find(W=>W.stepId===T.id&&W.err))==null?void 0:O.err,C=T.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${f?` <a href="${r(f)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${q?"step-done":""} ${s?"step-error":""}">
          <div class="step-head">${g} <strong>${r(T.title)}</strong></div>
          ${C}
          ${v.length?`<pre class="step-log">${v.map(W=>r(W)).join(`
`)}</pre>`:""}
          ${b?`<p class="error small">${r(b)}</p>`:""}
        </li>
      `}).join(""),j=e.events.some(T=>T.err),Q=se.every(T=>E.has(T.id))||e.events.some(T=>T.stepId==="handshake"&&T.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${H}</ol>
        ${Q&&!j?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${j?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function o(d,h){switch(d){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":c(),e.step="mode",m();break;case"goto-review":if(y(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError){m();break}e.step="review",m();break;case"start-setup":X();break}}function c(){const d=t.querySelector("#exec-select"),h=t.querySelector("#beacon-select");d&&(e.execId=d.value),h&&(e.beaconId=h.value)}function y(){const d=t.querySelectorAll('input[name="mode"]');for(const j of Array.from(d))j.checked&&(e.archive=j.value==="archive");const h=t.querySelector("#data-dir-input"),f=t.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),f&&(e.jwtPath=f.value.trim());const E=t.querySelector("#exec-http-port-input"),k=t.querySelector("#beacon-http-port-input"),I=t.querySelector("#exec-p2p-port-input");E&&(e.execHTTPPort=E.value.trim()),k&&(e.beaconHTTPPort=k.value.trim()),I&&(e.execP2PPort=I.value.trim());const H=t.querySelector("#rpc-bind-addr-input");H&&(e.rpcBindAddr=H.value.trim()),e.execHTTPPortError=M(e.execHTTPPort).error??null,e.beaconHTTPPortError=M(e.beaconHTTPPort).error??null,e.execP2PPortError=M(e.execP2PPort).error??null,e.rpcBindAddrError=$(e.rpcBindAddr).error??null}function $(d){if(!d)return{};const h=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(d);return h?h.slice(1).every(f=>Number(f)<=255)?{addr:d}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(d)&&d.includes(":")?{addr:d}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const N=/^\d+$/;function M(d){if(!d)return{};if(!N.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const h=Number(d);return!Number.isInteger(h)||h<1||h>65535?{error:"Port must be between 1 and 65535."}:{port:h}}function F(d,h){const{port:f}=M(d);if(!(f===void 0||f===h))return f}async function X(){var I;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,m();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const h=F(e.execHTTPPort,te),f=F(e.beaconHTTPPort,ne),E=F(e.execP2PPort,ae);h!==void 0&&(d.ExecHTTPPort=h),f!==void 0&&(d.BeaconHTTPPort=f),E!==void 0&&(d.ExecP2PPort=E);const{addr:k}=$(e.rpcBindAddr);k!==void 0&&(d.RPCBindAddr=k);try{await we(e.targetId,d)}catch(H){if(!(H instanceof oe&&H.status===409)){e.starting=!1,e.startError=String(H instanceof Error?H.message:H),m();return}}e.starting=!1,e.step="run",e.events=[],m(),(I=e.streamStop)==null||I.call(e),e.streamStop=xe(e.targetId,H=>{a||(e.events.push(H),e.step==="run"&&m())})}function Z(d){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],E=h.map(k=>k.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${h.map((k,I)=>`<li class="${I===E?"current":I<E?"past":"future"}">${r(k.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;a=!0,(d=e.streamStop)==null||d.call(e)}}const et=document.querySelector("#app"),{contentEl:tt,setActiveNav:nt}=Ne(et);let _=null;function at(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"||a==="diag"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function J(t){const n=document.createElement("div");return tt.replaceChildren(n),t(n)}function ge(){if(_){try{_()}catch{}_=null}const{screen:t,id:n}=at();switch(nt(t),t){case"setup":if(!n){location.hash="#/targets";return}_=J(a=>Qe(a,n));break;case"dash":if(!n){location.hash="#/targets";return}_=J(a=>Oe(a,n));break;case"logs":if(!n){location.hash="#/targets";return}_=J(a=>Fe(a,n));break;case"security":if(!n){location.hash="#/targets";return}_=J(a=>_e(a,n));break;case"diag":if(!n){location.hash="#/targets";return}_=J(a=>je(a,n));break;case"settings":_=J(a=>Je(a));break;case"targets":default:_=J(a=>Ke(a));break}}window.addEventListener("hashchange",ge);ge();
