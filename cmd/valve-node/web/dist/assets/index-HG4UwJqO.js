var ye=Object.defineProperty;var $e=(t,n,a)=>n in t?ye(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var ce=(t,n,a)=>$e(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))e(l);new MutationObserver(l=>{for(const u of l)if(u.type==="childList")for(const T of u.addedNodes)T.tagName==="LINK"&&T.rel==="modulepreload"&&e(T)}).observe(document,{childList:!0,subtree:!0});function a(l){const u={};return l.integrity&&(u.integrity=l.integrity),l.referrerPolicy&&(u.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?u.credentials="include":l.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(l){if(l.ep)return;l.ep=!0;const u=a(l);fetch(l.href,u)}})();function X(){return D("/api/catalog")}function Y(){return D("/api/targets")}function le(t){return D("/api/targets",{method:"POST",headers:ee,body:JSON.stringify(t)})}function we(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,n){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:ee,body:JSON.stringify(n)})}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Pe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Se(t,n=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Ee(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function de(t,n){const a=n===void 0?{}:{lines:n};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:ee,body:JSON.stringify(a)})}function ke(t,n,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function Ce(t,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:ee,body:JSON.stringify({Confirm:n})})}function Le(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Ie(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function He(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Re(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Be(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Ae(){return D("/api/settings")}function De(t){return D("/api/settings",{method:"PUT",headers:ee,body:JSON.stringify(t)})}class oe extends Error{constructor(a,e){super(e);ce(this,"status");this.name="ApiError",this.status=a}}const ee={"Content-Type":"application/json"};async function D(t,n){const a=await fetch(t,n);if(!a.ok){let l=a.statusText||`HTTP ${a.status}`;try{const u=await a.json();u&&typeof u.error=="string"&&u.error&&(l=u.error)}catch{}throw new oe(a.status,l)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const Ne="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function N(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${r(n)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(Ne)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function Me(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:l=>{for(const u of a)u.classList.toggle("active",u.dataset.nav===l)}}}function W(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ue(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function qe(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const l=Math.floor(a/24),u=a%24;return`~${l}d ${u}h`}function A(t,n){return`<span class="badge badge-${n}">${r(t)}</span>`}function ue(t){return`<span class="dot dot-${t}"></span>`}const pe=["B","KB","MB","GB","TB","PB"];function V(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<pe.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${pe[a]}`}async function ie(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Z(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const l=e.dataset.action;l&&n(l,e,a)})}const Oe=85,re={exec:"Execution",beacon:"Beacon"};function Fe(t,n){let a=!1,e=null,l=null,u=null,T=null,g=null,R=null,E=null,B=null;const p={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${r(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${N()}</div>`;const x=t.querySelector("#dash-body"),f=t.querySelector("#dash-footer");x.addEventListener("click",s=>{const m=s.target.closest("[data-action]");if(!m||!x.contains(m))return;const b=m.dataset.action;if(b==="svc-action"){const y=m.dataset.svc,L=m.dataset.kind;y&&L&&I(y,L)}else if(b==="open-clear"){const y=m.dataset.svc;y&&O(y)}else if(b==="copy"){const y=m.dataset.copy;y&&H(m,y)}else b==="retry-du"?o():b==="retry-endpoints"&&i()}),c();async function c(){let s,m;try{const[y,L]=await Promise.all([Y(),X()]);s=y.find(U=>U.id===n),m=L}catch(y){if(a)return;x.innerHTML=`<p class="error">Failed to load target: ${r(String(y))}</p>`;return}if(a)return;if(!s){x.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){x.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const b=m==null?void 0:m.networks.find(y=>y.ChainID===s.wire.ChainID);b&&(f.innerHTML=N(b.Name,b.LearnURL)),x.innerHTML='<p class="muted">Connecting…</p>',e=Pe(n,y=>{a||(w(y),l=y,u=y,P())}),o(),i()}async function o(){R=null;try{g=await Le(n)}catch(s){g=null,R=String(s instanceof Error?s.message:s)}a||P()}async function i(){B=null;try{E=await Ie(n)}catch(s){E=null,B=String(s instanceof Error?s.message:s)}a||P()}function w(s){if(!l)return;const m=(new Date(s.at).getTime()-new Date(l.at).getTime())/1e3,b=s.execHead-l.execHead;if(m>0&&b>=0){const y=b/m;T=T===null?y:T*.7+y*.3}}function P(){if(!u)return;const s=u;x.innerHTML=`
      <div class="card-grid">
        ${j(s)}
        ${q(s)}
        ${_(s)}
        ${J(s)}
        ${d(s)}
        ${h()}
        ${k(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function F(s){const b=s.refHead>0?s.refHead-s.execHead:null,y=b!==null&&b>0&&T&&T>0?qe(b/T):b!==null&&b<=0?"caught up":"—";return{lag:b,eta:y}}function j(s){const{lag:m,eta:b}=F(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?A("syncing","warn"):A("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${W(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${m!==null?W(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${m!==null?W(Math.max(m,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${b}</dd></div>
        </dl>
      </div>
    `}function q(s){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${s.beaconDistance===0?A("synced","ok"):A("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${W(s.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${W(s.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function _(s){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${W(s.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${W(s.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function J(s){const m=s.diskUsedPct>=Oe;return`
      <div class="card ${m?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${m?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Ue(s.diskUsedPct)} used</p>
      </div>
    `}function d(s){if(R)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(R)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!g)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const m=g.ExpectedExecBytes>0?Math.min(g.ExecBytes/g.ExpectedExecBytes*100,100):0,b=g.ExpectedBeaconBytes>0?Math.min(g.BeaconBytes/g.ExpectedBeaconBytes*100,100):0,{lag:y,eta:L}=F(s),U=y!==null&&y>0&&T!==null&&T>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${V(g.ExecBytes)} of ~${V(g.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${m}%"></div></div>
        ${U?`<p class="muted small">Estimated time remaining: ${r(L)}</p>`:""}
        <p class="muted small">Beacon — ${V(g.BeaconBytes)} of ~${V(g.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${b}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${V(g.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(g.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(g.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function h(){if(B)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(B)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!E)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=E,m=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",b=s.Access==="ssh"?`
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
        ${m}
        ${b}
      </div>
    `}function v(s,m){const b=re[s],y=p[s],L=(U,G,be)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${U}" ${y!==null||be?"disabled":""}>${y===U?C():r(G)}</button>`;return`
      <div class="service-row">
        <span>${r(b)} ${m?A("active","ok"):A("down","bad")}</span>
        <div class="service-actions">
          ${L("start","Start",m)}
          ${L("stop","Stop",!m)}
          ${L("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${y!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function k(s){return`
      <div class="card">
        <h3>Services</h3>
        ${v("exec",s.execActive)}
        ${v("beacon",s.beaconActive)}
        ${$?`<p class="error small">${r($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(n)}">Diagnostics →</a>
        </p>
      </div>
    `}function C(){return'<span class="spinner" aria-label="working"></span>'}async function I(s,m){if(p[s]===null){p[s]=m,$=null,P();try{await ke(n,s,m)}catch(b){$=`${re[s]} ${m} failed: ${b instanceof Error?b.message:String(b)}`}p[s]=null,a||P()}}async function H(s,m){const b=await ie(m),y=s.textContent;s.textContent=b?"Copied!":"Copy failed",setTimeout(()=>{a||(s.textContent=y)},1500)}function O(s){const m=re[s],b=g?V(s==="exec"?g.ExecBytes:g.BeaconBytes):"unknown (disk usage hasn't loaded)";S(`
        <h2>Clear ${r(m)} data</h2>
        <p class="error">
          This stops the ${r(m.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${r(b)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${r(s)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,U=>{if(U==="cancel"){M();return}U==="confirm"&&Q(s)});const y=document.getElementById("clear-confirm-input"),L=document.getElementById("clear-confirm-btn");y==null||y.addEventListener("input",()=>{L&&(L.disabled=y.value.trim()!==s)}),y==null||y.focus()}async function Q(s){const m=document.getElementById("clear-confirm-btn");m&&(m.disabled=!0,m.textContent="Clearing…");try{await Ce(n,s),M(),o()}catch(b){const y=document.querySelector("#clear-modal .modal");if(y){const L=document.createElement("p");L.className="error small",L.textContent=`Clear failed: ${b instanceof Error?b.message:String(b)}`,y.appendChild(L)}m&&(m.disabled=!1,m.textContent="Clear and resync")}}function S(s,m){M();const b=document.createElement("div");b.className="modal-overlay",b.id="clear-modal",b.innerHTML=`<div class="modal">${s}</div>`,b.addEventListener("click",y=>{const L=y.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&m(L.dataset.modalAction),y.target===b&&m("cancel")}),document.body.appendChild(b)}function M(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{a=!0,e==null||e(),M()}}const he=500,fe="valve-node.explain-consent";function je(t,n){let a=!1,e=null;const l=[];t.innerHTML=`
    <h1>Logs: ${r(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${N()}</div>
  `;const u=t.querySelector("#logs-body"),T=t.querySelector("#logs-footer");Z(t,c=>{c==="explain"&&B()}),g();async function g(){let c,o;try{const[w,P]=await Promise.all([Y(),X()]);c=w.find(F=>F.id===n),o=P}catch(w){if(a)return;u.innerHTML=`<p class="error">Failed to load target: ${r(String(w))}</p>`;return}if(a)return;if(!c){u.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const i=o==null?void 0:o.networks.find(w=>w.ChainID===c.wire.ChainID);i&&(T.innerHTML=N(i.Name,i.LearnURL));try{const w=await Se(n,200);if(a)return;l.push(...w)}catch(w){if(a)return;u.innerHTML=`<p class="error">Failed to load logs: ${r(String(w))}</p>`;return}R(),e=Ee(n,w=>{a||(l.push(w),l.length>he&&l.splice(0,l.length-he),R())})}function R(){const c=l.filter(i=>i.severity==="error"||i.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(E).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${A(String(c.length),c.length?"bad":"neutral")}</h2>
          <div class="log-lines">${c.length?c.slice().reverse().map(E).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=u.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function E(c){const o=c.severity||"info",i=c.learnUrl?` <a href="${r(c.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(c.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(c.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(c.line)}</span>
        ${c.explain?`<div class="log-explain">${r(c.explain)}${i}</div>`:""}
      </div>
    `}async function B(){const c=l.filter(i=>i.severity==="error"||i.severity==="critical").map(i=>i.line).slice(-40);if(!(localStorage.getItem(fe)==="1")){p(c);return}await $(c)}function p(c){const o=c.length?`<pre class="explain-excerpt">${c.map(i=>r(i)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';x(`
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
    `,i=>{i==="proceed"?(localStorage.setItem(fe,"1"),f(),$(c)):f()})}async function $(c){x('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=c.length?await de(n,c):await de(n);if(a)return;x(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(i=>r(i)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&f()})}catch(o){if(a)return;if(o instanceof oe&&o.status===409){x(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,i=>{i==="close"&&f()});return}x(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&f()})}}function x(c,o){f();const i=document.createElement("div");i.className="modal-overlay",i.id="explain-modal",i.innerHTML=`<div class="modal">${c}</div>`,i.addEventListener("click",w=>{const P=w.target.closest("[data-modal-action]");P!=null&&P.dataset.modalAction&&o(P.dataset.modalAction),w.target===i&&o("cancel")}),document.body.appendChild(i)}function f(){var c;(c=document.getElementById("explain-modal"))==null||c.remove()}return()=>{a=!0,e==null||e(),f()}}function _e(t,n){let a=!1,e=null,l=null,u=!1,T=!1;t.innerHTML=`<h1>Network diagnostics: ${r(n)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${N()}</div>`;const g=t.querySelector("#diag-body"),R=t.querySelector("#diag-footer");Z(t,(o,i)=>{var w;if(o==="run")B();else if(o==="toggle")(w=i.closest(".check-item"))==null||w.classList.toggle("expanded");else if(o==="copy"){const P=i.dataset.copy;P&&c(i,P)}}),E();async function E(){let o,i;try{const[P,F]=await Promise.all([Y(),X()]);o=P.find(j=>j.id===n),i=F}catch(P){if(a)return;g.innerHTML=`<p class="error">Failed to load target: ${r(String(P))}</p>`;return}if(a)return;if(!o){g.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){g.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const w=i==null?void 0:i.networks.find(P=>P.ChainID===o.wire.ChainID);w&&(R.innerHTML=N(w.Name,w.LearnURL));try{e=await Be(n),T=!0}catch(P){l=String(P instanceof Error?P.message:P)}a||p()}async function B(){u=!0,l=null,p();try{e=await Re(n),T=!0}catch(o){l=String(o instanceof Error?o.message:o)}u=!1,a||p()}function p(){g.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${u?"disabled":""}>${u?"Running…":"Run diagnostics"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${$()}
    `}function $(){if(!T&&!l)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),i=e.failedId?`<p><strong>Failed at: ${r(x(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${i}
      <ul class="check-list">${e.items.map(f).join("")}</ul>
    `}function x(o){var i;return((i=e==null?void 0:e.items.find(w=>w.ID===o))==null?void 0:i.Title)??o}function f(o){const i=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",w=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${w?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${A(w?"failed here":o.Status,i)}
          <strong>${r(o.Title)}</strong>
          <span class="muted small check-detail-inline">${r(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${w?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${r(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${w?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function c(o,i){const w=await ie(i),P=o.textContent;o.textContent=w?"Copied!":"Copy failed",setTimeout(()=>{a||(o.textContent=P)},1500)}return()=>{a=!0}}function ze(t,n){let a=!1,e=[],l=null,u=!1,T=!1;t.innerHTML=`<h1>Security: ${r(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${N()}</div>`;const g=t.querySelector("#sec-body"),R=t.querySelector("#sec-footer");Z(t,(f,c)=>{var o;if(f==="rerun")B();else if(f==="toggle")(o=c.closest(".check-item"))==null||o.classList.toggle("expanded");else if(f==="copy"){const i=c.dataset.copy;i&&x(c,i)}}),E();async function E(){let f,c;try{const[i,w]=await Promise.all([Y(),X()]);f=i.find(P=>P.id===n),c=w}catch(i){if(a)return;g.innerHTML=`<p class="error">Failed to load target: ${r(String(i))}</p>`;return}if(a)return;if(!f){g.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){g.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const o=c==null?void 0:c.networks.find(i=>i.ChainID===f.wire.ChainID);o&&(R.innerHTML=N(o.Name,o.LearnURL)),await B()}async function B(){u=!0,l=null,p();try{e=await He(n),T=!0}catch(f){l=String(f instanceof Error?f.message:f)}u=!1,a||p()}function p(){g.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${!T&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:T?'<p class="muted">No checks returned.</p>':""}
    `}function $(f){const c=f.Status==="pass"?"ok":f.Status==="fail"?"bad":f.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${A(f.Status,c)}
          <strong>${r(f.Title)}</strong>
          <span class="muted small check-detail-inline">${r(f.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(f.Why)}</p>
          </details>
          ${f.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(f.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(f.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function x(f,c){const o=await ie(c),i=f.textContent;f.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{a||(f.textContent=i)},1500)}return()=>{a=!0}}const Je=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function We(t){let n=!1,a=!1,e=!1,l=null,u=!1,T=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${N()}`;const g=t.querySelector("#settings-body");Z(t,p=>{if(p==="save"&&B(),p==="clear-key"){if(!T)return;a=!0;const $=t.querySelector("#ai-key");$&&($.value=""),E(T)}}),R();async function R(){try{const p=await Ae();if(n)return;T=p,E(p)}catch(p){if(n)return;g.innerHTML=`<p class="error">Failed to load settings: ${r(String(p))}</p>`}}function E(p){var f,c;const $=Je.map(o=>`<option value="${o.value}" ${p.aiProvider===o.value?"selected":""}>${r(o.label)}</option>`).join("");g.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${$}</select>
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${p.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${p.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${r(p.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${l?`<p class="error">${r(l)}</p>`:""}
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const x=t.querySelector("#ai-key");x==null||x.addEventListener("input",()=>{a=!0,u=!1}),(f=t.querySelector("#ai-provider"))==null||f.addEventListener("change",()=>{u=!1}),(c=t.querySelector("#ref-rpc-base"))==null||c.addEventListener("input",()=>{u=!1})}async function B(){const p=t.querySelector("#ai-provider"),$=t.querySelector("#ai-key"),x=t.querySelector("#ref-rpc-base");if(!p||!$||!x||!T)return;const f={aiProvider:p.value,refRpcBase:x.value.trim()};a&&(f.aiKey=$.value),e=!0,l=null,u=!1,E(T);try{const c=await De(f);if(n)return;T=c,a=!1,e=!1,u=!0,E(c)}catch(c){if(n)return;e=!1,l=String(c instanceof Error?c.message:c),E(T)}}return()=>{n=!0}}const Ke="local";function Ge(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${N()}
  `;const a=t.querySelector("#targets-body");Z(t,(p,$)=>{u(p,$)}),e();async function e(){try{const[p,$]=await Promise.all([Y(),X()]);if(n)return;l(p,$)}catch(p){if(n)return;a.innerHTML=`<p class="error">Failed to load targets: ${r(String(p))}</p>`}}function l(p,$){const x=p.find(i=>i.mode==="local"),f=p.filter(i=>i.mode==="ssh"),c=x?me(x,$):Xe()?`
        <div class="card">
          <h2>This machine</h2>
          <p class="muted">
            valve-node is running here as your <strong>controller</strong> — it drives your nodes but
            doesn't host them. Node hosts must be Linux, so add a server over SSH below.
          </p>
        </div>
      `:`
        <div class="card">
          <h2>This machine</h2>
          <p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,o=f.length?f.map(i=>me(i,$)).join(""):'<p class="muted">No SSH targets yet.</p>';a.innerHTML=`
      <section class="section">${c}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${o}</div>
        ${Ve()}
      </section>
    `}async function u(p,$){if(p==="add-local"){await T();return}if(p==="delete-target"){const x=$.dataset.id;if(!x||!confirm(`Remove target "${x}"? This does not touch anything already running on it.`))return;await g(x);return}p==="add-ssh"&&await R()}async function T(){B();try{await le({id:Ke,mode:"local"}),await e()}catch(p){E(p)}}async function g(p){try{await we(p),await e()}catch($){E($)}}async function R(){const p=t.querySelector("#ssh-host"),$=t.querySelector("#ssh-user"),x=t.querySelector("#ssh-key"),f=t.querySelector("#ssh-port"),c=t.querySelector("#ssh-id");if(!p||!$||!x||!f||!c)return;const o=p.value.trim(),i=$.value.trim(),w=x.value.trim(),P=f.value.trim(),F=c.value.trim();if(B(),!o||!i||!w){E(new Error("host, user, and key path are required"));return}const j=F||Ye(o),q={Host:o,User:i,KeyPath:w};if(P){const J=Number.parseInt(P,10);if(!Number.isFinite(J)||J<=0){E(new Error("port must be a positive number"));return}q.Port=J}const _=t.querySelector("#ssh-submit");_&&(_.disabled=!0,_.textContent="Connecting…");try{await le({id:j,mode:"ssh",ssh:q}),await e()}catch(J){E(J),_&&(_.disabled=!1,_.textContent="Add server")}}function E(p){let $=t.querySelector("#targets-error");$||(a.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=t.querySelector("#targets-error")),$.textContent=String(p instanceof Error?p.message:p)}function B(){var p;(p=t.querySelector("#targets-error"))==null||p.remove()}return()=>{n=!0}}function me(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",l=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let u,T;if(!a)u=A("not set up","neutral"),T=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const g=n.networks.find(E=>E.ChainID===a.ChainID),R=g?g.Name:`chain ${a.ChainID}`;u=`${A(R,"ok")} ${A(a.ExecID,"neutral")} ${A(a.BeaconID,"neutral")}${a.Archive?" "+A("archive","warn"):""}`,T=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${l}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${T}
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
  `}function Xe(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Ye(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const se=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],te=8545,ne=5052,ae=30303,Ze=[369,943,1],ge={369:"default",943:"practise here first"};function Qe(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${N()}</div>`;const l=t.querySelector("#wizard-body"),u=t.querySelector("#wizard-footer");Z(t,(d,h)=>{o(d,h)}),T();async function T(){try{const[d,h]=await Promise.all([X(),Y()]);if(a)return;e.catalog=d;const v=h.find(k=>k.id===n);v!=null&&v.wire&&(e.chainId=v.wire.ChainID,e.execId=v.wire.ExecID,e.beaconId=v.wire.BeaconID,e.archive=v.wire.Archive,v.wire.ExecHTTPPort&&(e.execHTTPPort=String(v.wire.ExecHTTPPort)),v.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(v.wire.BeaconHTTPPort)),v.wire.ExecP2PPort&&(e.execP2PPort=String(v.wire.ExecP2PPort)),v.wire.RPCBindAddr&&(e.rpcBindAddr=v.wire.RPCBindAddr)),g()}catch(d){if(a)return;e.loadError=String(d instanceof Error?d.message:d),g()}}function g(){if(e.loadError){l.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(l.innerHTML=`
      ${J(e.step)}
      ${E()}
    `,R())}function R(){var h;const d=(h=e.catalog)==null?void 0:h.networks.find(v=>v.ChainID===e.chainId);u.innerHTML=d?N(d.Name,d.LearnURL):N()}function E(){switch(e.step){case"network":return B();case"clients":return p();case"mode":return x();case"review":return f();case"run":return c()}}function B(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ze.map(v=>{const k=d.networks.find(H=>H.ChainID===v);if(!k)return"";const C=e.chainId===v,I=ge[v]?A(ge[v],v===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${C?"selected":""}" data-action="pick-network" data-chain-id="${v}" type="button">
          <h3>${r(k.Name)} <span class="muted">(chain ${v})</span></h3>
          ${I}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function p(){const d=e.catalog,h=d.networks.find(C=>C.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const v=h.ExecClients.map(C=>$(C,d,e.execId)).join(""),k=h.BeaconClients.map(C=>$(C,d,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(h.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${v}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${k}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function $(d,h,v){const k=h.clients.find(I=>I.id===d),C=k?`${k.id} (${k.toolchain})`:d;return`<option value="${r(d)}" ${d===v?"selected":""}>${r(C)}</option>`}function x(){const d=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
    `}function f(){const h=e.catalog.networks.find(s=>s.ChainID===e.chainId),v=e.dataDir||`/var/lib/valve-node/${e.chainId}`,k=e.jwtPath||`${v}/jwt.hex`,C=se.map(s=>`<li>${r(s.title)}</li>`).join(""),I=q(e.execHTTPPort,te),H=q(e.beaconHTTPPort,ne),O=q(e.execP2PPort,ae),Q=I||H||O?`<tr><th>Non-default ports</th><td>${[I?`exec HTTP ${I}`:null,H?`beacon HTTP ${H}`:null,O?`exec p2p ${O}`:null].filter(s=>s!==null).map(r).join(", ")}</td></tr>`:"",{addr:S}=P(e.rpcBindAddr),M=S?`<tr><th>RPC bind address</th><td><code>${r(S)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(v)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(k)}</code></td></tr>
            ${h?`<tr><th>Checkpoint sync</th><td><code>${r(h.CheckpointURL)}</code></td></tr>`:""}
            ${Q}
            ${M}
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
    `}function c(){const h=e.catalog.networks.find(S=>S.ChainID===e.chainId),v=h==null?void 0:h.LearnURL,k=new Set(e.events.filter(S=>S.done).map(S=>S.stepId)),C=new Set(e.events.filter(S=>S.err).map(S=>S.stepId)),I=new Map;for(const S of e.events){if(!S.line)continue;const M=I.get(S.stepId)??[];M.push(S.line),I.set(S.stepId,M)}const H=se.map(S=>{var U;const M=k.has(S.id),s=C.has(S.id),m=s?A("failed","bad"):M?A("done","ok"):A("pending","neutral"),b=(I.get(S.id)??[]).slice(-5),y=(U=e.events.find(G=>G.stepId===S.id&&G.err))==null?void 0:U.err,L=S.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${v?` <a href="${r(v)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${M?"step-done":""} ${s?"step-error":""}">
          <div class="step-head">${m} <strong>${r(S.title)}</strong></div>
          ${L}
          ${b.length?`<pre class="step-log">${b.map(G=>r(G)).join(`
`)}</pre>`:""}
          ${y?`<p class="error small">${r(y)}</p>`:""}
        </li>
      `}).join(""),O=e.events.some(S=>S.err),Q=se.every(S=>k.has(S.id))||e.events.some(S=>S.stepId==="handshake"&&S.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${H}</ol>
        ${Q&&!O?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${O?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function o(d,h){switch(d){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,g();break;case"goto-network":e.step="network",g();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",g();break;case"goto-mode":i(),e.step="mode",g();break;case"goto-review":if(w(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError){g();break}e.step="review",g();break;case"start-setup":_();break}}function i(){const d=t.querySelector("#exec-select"),h=t.querySelector("#beacon-select");d&&(e.execId=d.value),h&&(e.beaconId=h.value)}function w(){const d=t.querySelectorAll('input[name="mode"]');for(const O of Array.from(d))O.checked&&(e.archive=O.value==="archive");const h=t.querySelector("#data-dir-input"),v=t.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),v&&(e.jwtPath=v.value.trim());const k=t.querySelector("#exec-http-port-input"),C=t.querySelector("#beacon-http-port-input"),I=t.querySelector("#exec-p2p-port-input");k&&(e.execHTTPPort=k.value.trim()),C&&(e.beaconHTTPPort=C.value.trim()),I&&(e.execP2PPort=I.value.trim());const H=t.querySelector("#rpc-bind-addr-input");H&&(e.rpcBindAddr=H.value.trim()),e.execHTTPPortError=j(e.execHTTPPort).error??null,e.beaconHTTPPortError=j(e.beaconHTTPPort).error??null,e.execP2PPortError=j(e.execP2PPort).error??null,e.rpcBindAddrError=P(e.rpcBindAddr).error??null}function P(d){if(!d)return{};const h=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(d);return h?h.slice(1).every(v=>Number(v)<=255)?{addr:d}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(d)&&d.includes(":")?{addr:d}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const F=/^\d+$/;function j(d){if(!d)return{};if(!F.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const h=Number(d);return!Number.isInteger(h)||h<1||h>65535?{error:"Port must be between 1 and 65535."}:{port:h}}function q(d,h){const{port:v}=j(d);if(!(v===void 0||v===h))return v}async function _(){var I;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,g();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const h=q(e.execHTTPPort,te),v=q(e.beaconHTTPPort,ne),k=q(e.execP2PPort,ae);h!==void 0&&(d.ExecHTTPPort=h),v!==void 0&&(d.BeaconHTTPPort=v),k!==void 0&&(d.ExecP2PPort=k);const{addr:C}=P(e.rpcBindAddr);C!==void 0&&(d.RPCBindAddr=C);try{await Te(e.targetId,d)}catch(H){if(!(H instanceof oe&&H.status===409)){e.starting=!1,e.startError=String(H instanceof Error?H.message:H),g();return}}e.starting=!1,e.step="run",e.events=[],g(),(I=e.streamStop)==null||I.call(e),e.streamStop=xe(e.targetId,H=>{a||(e.events.push(H),e.step==="run"&&g())})}function J(d){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],k=h.map(C=>C.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${h.map((C,I)=>`<li class="${I===k?"current":I<k?"past":"future"}">${r(C.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;a=!0,(d=e.streamStop)==null||d.call(e)}}const et=document.querySelector("#app"),{contentEl:tt,setActiveNav:nt}=Me(et);let z=null;function at(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"||a==="diag"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function K(t){const n=document.createElement("div");return tt.replaceChildren(n),t(n)}function ve(){if(z){try{z()}catch{}z=null}const{screen:t,id:n}=at();switch(nt(t),t){case"setup":if(!n){location.hash="#/targets";return}z=K(a=>Qe(a,n));break;case"dash":if(!n){location.hash="#/targets";return}z=K(a=>Fe(a,n));break;case"logs":if(!n){location.hash="#/targets";return}z=K(a=>je(a,n));break;case"security":if(!n){location.hash="#/targets";return}z=K(a=>ze(a,n));break;case"diag":if(!n){location.hash="#/targets";return}z=K(a=>_e(a,n));break;case"settings":z=K(a=>We(a));break;case"targets":default:z=K(a=>Ge(a));break}}window.addEventListener("hashchange",ve);ve();
