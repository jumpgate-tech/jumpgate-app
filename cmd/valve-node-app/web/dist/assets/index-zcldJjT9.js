var ot=Object.defineProperty;var rt=(n,c,o)=>c in n?ot(n,c,{enumerable:!0,configurable:!0,writable:!0,value:o}):n[c]=o;var Ae=(n,c,o)=>rt(n,typeof c!="symbol"?c+"":c,o);(function(){const c=document.createElement("link").relList;if(c&&c.supports&&c.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const f of u)if(f.type==="childList")for(const x of f.addedNodes)x.tagName==="LINK"&&x.rel==="modulepreload"&&e(x)}).observe(document,{childList:!0,subtree:!0});function o(u){const f={};return u.integrity&&(f.integrity=u.integrity),u.referrerPolicy&&(f.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?f.credentials="include":u.crossOrigin==="anonymous"?f.credentials="omit":f.credentials="same-origin",f}function e(u){if(u.ep)return;u.ep=!0;const f=o(u);fetch(u.href,f)}})();function it(){return _("/api/host")}function Se(){return _("/api/catalog")}function xe(){return _("/api/targets")}function Ge(n){return _("/api/targets",{method:"POST",headers:de,body:JSON.stringify(n)})}function ct(n){return _(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function lt(n,c){return _(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(c)}`)}function dt(n,c){return _(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:de,body:JSON.stringify(c)})}function Ke(n,c){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function ut(n,c){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function pt(n,c=200){return _(`/api/targets/${encodeURIComponent(n)}/logs?n=${c}`)}function ht(n,c){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function Ve(n,c){const o=c===void 0?{}:{lines:c};return _(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:de,body:JSON.stringify(o)})}function ft(n,c,o){return _(`/api/targets/${encodeURIComponent(n)}/services/${c}/${o}`,{method:"POST"})}function mt(n,c){return _(`/api/targets/${encodeURIComponent(n)}/services/${c}/clear`,{method:"POST",headers:de,body:JSON.stringify({Confirm:c})})}function bt(n){return _(`/api/targets/${encodeURIComponent(n)}/du`)}function gt(n){return _(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function vt(n){return _(`/api/targets/${encodeURIComponent(n)}/firewall`)}function yt(n){return _(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function $t(n){return _(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function wt(n){return _(`/api/targets/${encodeURIComponent(n)}/containers`)}function kt(n,c,o){return _(`/api/targets/${encodeURIComponent(n)}/containers/${c}/${o}`,{method:"POST"})}async function Ct(n,c){const o=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${c}/wipe`,{method:"POST",headers:de,body:JSON.stringify({Confirm:c})}),e=await o.text();let u=null;try{u=e?JSON.parse(e):null}catch{}if(u&&typeof u=="object"&&"report"in u)return u;const f=u&&typeof u=="object"&&typeof u.error=="string"?u.error:o.statusText||`HTTP ${o.status}`;throw new ve(o.status,f)}function Tt(n,c){return _(`/api/targets/${encodeURIComponent(n)}/containers/${c}/provision`,{method:"POST"})}async function St(n){const c=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:de}),o=await c.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:c.statusText||`HTTP ${c.status}`;throw new ve(c.status,u)}function xt(n,c,o){return _(`/api/targets/${encodeURIComponent(n)}/containers/${c}/config`,{method:"PUT",headers:de,body:JSON.stringify(o)})}function Pt(){return _("/api/gateways")}function Et(n){return _("/api/gateways",{method:"POST",headers:de,body:JSON.stringify(n)})}function It(n){return _(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function Rt(n,c){return _(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:de,body:JSON.stringify(c)})}function Lt(n,c){return _(`/api/gateways/${encodeURIComponent(n)}/${c}`,{method:"POST"})}function Nt(n){return _(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Bt(n){const c=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:de,body:JSON.stringify({Confirm:n})}),o=await c.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:c.statusText||`HTTP ${c.status}`;throw new ve(c.status,u)}function At(n){return _(`/api/chainlist/${n}`)}function Ht(){return _("/api/settings")}function Dt(n){return _("/api/settings",{method:"PUT",headers:de,body:JSON.stringify(n)})}class ve extends Error{constructor(o,e,u,f){super(e);Ae(this,"status");Ae(this,"hint");Ae(this,"code");this.name="ApiError",this.status=o,this.hint=u,this.code=f}}const de={"Content-Type":"application/json"};async function _(n,c){const o=await fetch(n,c);if(!o.ok){let u=o.statusText||`HTTP ${o.status}`,f,x;try{const y=await o.json();y&&typeof y.error=="string"&&y.error&&(u=y.error),y&&typeof y.hint=="string"&&y.hint&&(f=y.hint),y&&typeof y.code=="string"&&y.code&&(x=y.code)}catch{}throw new ve(o.status,u,f,x)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const Ye="https://learn.valve.city/rpc";function t(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ee(n,c){const o=n&&c&&c!==Ye?` <span class="footer-sep">·</span> <a href="${t(c)}" target="_blank" rel="noopener noreferrer">${t(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${t(Ye)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function Ut(n){n.innerHTML=`
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/targets">valve-node-app</a>
        <nav class="nav">
          <a href="#/targets" data-nav="targets">Targets</a>
          <a href="#/rpc" data-nav="rpc">RPC</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;const c=n.querySelector("#content"),o=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:c,setActiveNav:u=>{for(const f of o)f.classList.toggle("active",f.dataset.nav===u)}}}function we(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function Mt(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function Ot(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const c=Math.round(n/60),o=Math.floor(c/60),e=c%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const u=Math.floor(o/24),f=o%24;return`~${u}d ${f}h`}function O(n,c){return`<span class="badge badge-${c}">${t(n)}</span>`}function le(n){return`<span class="dot dot-${n}"></span>`}const Ze=["B","KB","MB","GB","TB","PB"];function ke(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let c=n,o=0;for(;c>=1024&&o<Ze.length-1;)c/=1024,o++;const e=c<10?2:c<100?1:0;return`${c.toFixed(e)} ${Ze[o]}`}async function Ie(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function ye(n,c){n.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!n.contains(e))return;const u=e.dataset.action;u&&c(u,e,o)})}function Oe(n,c,o){const e=c.find(f=>f.value===o),u=c.map(f=>`
      <li class="dropdown-option${f.value===o?" selected":""}" role="option"
          aria-selected="${f.value===o}" data-value="${t(f.value)}">
        ${t(f.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${t(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${t(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${u}</ul>
    </div>
  `}function Pe(n){n.querySelectorAll(".dropdown.open").forEach(c=>{var o;c.classList.remove("open"),(o=c.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function ze(n,c){n.addEventListener("click",u=>{const f=u.target,x=f.closest(".dropdown-trigger");if(x&&n.contains(x)){const N=x.closest(".dropdown"),U=!!N&&!N.classList.contains("open");Pe(n),N&&U&&(N.classList.add("open"),x.setAttribute("aria-expanded","true"));return}const y=f.closest(".dropdown-option");if(y&&n.contains(y)){const N=y.closest(".dropdown");Pe(n),c((N==null?void 0:N.dataset.dropdown)??"",y.dataset.value??"");return}Pe(n)});const o=u=>{if(!n.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const f=u.target;(!f.closest(".dropdown")||!n.contains(f))&&Pe(n)},e=u=>{if(!n.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}u.key==="Escape"&&Pe(n)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const je="app-modal";let Me=null;function Q(n,c){K();const o=document.createElement("div");o.className="modal-overlay",o.id=je,o.innerHTML=`<div class="modal">${n}</div>`,o.addEventListener("click",u=>{const f=u.target.closest("[data-modal-action]");f!=null&&f.dataset.modalAction?c(f.dataset.modalAction):u.target===o&&c("cancel")});const e=u=>{u.key==="Escape"&&c("cancel")};document.addEventListener("keydown",e),Me=e,document.body.appendChild(o)}function K(){var n;(n=document.getElementById(je))==null||n.remove(),Me&&(document.removeEventListener("keydown",Me),Me=null)}function Fe(){return document.querySelector(`#${je} .modal`)}function Ee(n){return new Promise(c=>{var u;let o=!1;const e=f=>{o||(o=!0,K(),c(f))};Q(`
        <h2>${t(n.title)}</h2>
        <p>${t(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${t(n.confirmLabel)}</button>
        </div>
      `,f=>e(f==="confirm")),(u=document.querySelector(`#${je} [data-modal-action="confirm"]`))==null||u.focus()})}const Ft=85,We={exec:"Execution",beacon:"Beacon"};function jt(n,c){let o=!1,e=null,u=null,f=null,x=null,y=null,N=null,U=null,M=null;const q={exec:null,beacon:null};let P=null;n.innerHTML=`<h1>Dashboard: ${t(c)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ee()}</div>`;const F=n.querySelector("#dash-body"),C=n.querySelector("#dash-footer");F.addEventListener("click",b=>{const I=b.target.closest("[data-action]");if(!I||!F.contains(I))return;const L=I.dataset.action;if(L==="svc-action"){const A=I.dataset.svc,z=I.dataset.kind;A&&z&&ae(A,z)}else if(L==="open-clear"){const A=I.dataset.svc;A&&ge(A)}else if(L==="copy"){const A=I.dataset.copy;A&&se(I,A)}else L==="retry-du"?l():L==="retry-endpoints"&&g()}),w();async function w(){let b,I;try{const[A,z]=await Promise.all([xe(),Se()]);b=A.find(d=>d.id===c),I=z}catch(A){if(o)return;F.innerHTML=`<p class="error">Failed to load target: ${t(String(A))}</p>`;return}if(o)return;if(!b){F.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!b.wire){F.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const L=I==null?void 0:I.networks.find(A=>A.ChainID===b.wire.ChainID);L&&(C.innerHTML=ee(L.Name,L.LearnURL)),F.innerHTML='<p class="muted">Connecting…</p>',e=ut(c,A=>{o||(E(A),u=A,f=A,B())}),l(),g()}async function l(){N=null;try{y=await bt(c)}catch(b){y=null,N=String(b instanceof Error?b.message:b)}o||B()}async function g(){M=null;try{U=await gt(c)}catch(b){U=null,M=String(b instanceof Error?b.message:b)}o||B()}function E(b){if(!u)return;const I=(new Date(b.at).getTime()-new Date(u.at).getTime())/1e3,L=b.execHead-u.execHead;if(I>0&&L>=0){const A=L/I;x=x===null?A:x*.7+A*.3}}function B(){if(!f)return;const b=f;F.innerHTML=`
      <p class="dash-status">${Y(b)}</p>
      <div class="card-grid">
        ${te(b)}
        ${Z(b)}
        ${oe(b)}
        ${ue(b)}
        ${pe(b)}
        ${he()}
      </div>
      <p class="muted small">Last updated ${t(new Date(b.at).toLocaleTimeString())}</p>
    `}function Y(b){return!b.execActive&&!b.beaconActive?O("Node not running","bad"):b.execSyncing||b.beaconDistance>0?O("Syncing","warn"):O("Running · synced","ok")}function X(b){const L=b.refHead>0?b.refHead-b.execHead:null,A=L!==null&&L>0&&x&&x>0?Ot(L/x):L!==null&&L<=0?"caught up":"—";return{lag:L,eta:A}}function Z(b){const{lag:I,eta:L}=X(b);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${b.execActive?b.execSyncing?O("syncing","warn"):b.execHead===0?O("no data","neutral"):O("synced","ok"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${we(b.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${I!==null?we(b.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${I!==null?we(Math.max(I,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${L}</dd></div>
        </dl>
      </div>
    `}function oe(b){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${b.beaconActive?b.beaconSlot===0?O("no data","neutral"):b.beaconDistance===0?O("synced","ok"):O("syncing","warn"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${we(b.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${we(b.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ue(b){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${we(b.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${we(b.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function pe(b){const I=b.diskUsedPct>=Ft,L=`
      <div class="meter"><div class="meter-fill ${I?"meter-warn":""}" style="width:${Math.min(b.diskUsedPct,100)}%"></div></div>
      <p>${Mt(b.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="error small">${t(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!y)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="muted">Loading…</p>
        </div>
      `;const A=y.ExpectedExecBytes>0?Math.min(y.ExecBytes/y.ExpectedExecBytes*100,100):0,z=y.ExpectedBeaconBytes>0?Math.min(y.BeaconBytes/y.ExpectedBeaconBytes*100,100):0,{lag:d,eta:h}=X(b),k=d!==null&&d>0&&x!==null&&x>0;return`
      <div class="card ${I?"card-warn":""}">
        <h3>Storage</h3>
        ${L}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${ke(y.ExecBytes)} of ~${ke(y.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${A}%"></div></div>
        ${k?`<p class="muted small">Estimated time remaining: ${t(h)}</p>`:""}
        <p class="muted small">Beacon — ${ke(y.BeaconBytes)} of ~${ke(y.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${z}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${ke(y.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${t(y.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${t(y.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function he(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${t(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!U)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const b=U,I=b.ExecReachable&&!b.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",L=b.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${t(b.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${t(b.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${le(b.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${t(b.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(b.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${le(b.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${t(b.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(b.BeaconHTTP)}">Copy</button>
        </div>
        ${I}
        ${L}
      </div>
    `}function ie(b,I){const L=We[b],A=q[b],z=(d,h,k)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${b}" data-kind="${d}" ${A!==null||k?"disabled":""}>${A===d?ne():t(h)}</button>`;return`
      <div class="service-row">
        <span>${t(L)} ${I?O("active","ok"):O("down","bad")}</span>
        <div class="service-actions">
          ${z("start","Start",I)}
          ${z("stop","Stop",!I)}
          ${z("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${b}" ${A!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function te(b){return`
      <div class="card">
        <h3>Services</h3>
        ${ie("exec",b.execActive)}
        ${ie("beacon",b.beaconActive)}
        ${P?`<p class="error small">${t(P)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(c)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(c)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(c)}">Diagnostics →</a>
        </p>
      </div>
    `}function ne(){return'<span class="spinner" aria-label="working"></span>'}async function ae(b,I){if(q[b]===null){q[b]=I,P=null,B();try{await ft(c,b,I)}catch(L){P=`${We[b]} ${I} failed: ${L instanceof Error?L.message:String(L)}`}q[b]=null,o||B()}}async function se(b,I){const L=await Ie(I),A=b.textContent;b.textContent=L?"Copied!":"Copy failed",setTimeout(()=>{o||(b.textContent=A)},1500)}function ge(b){const I=We[b],L=y?ke(b==="exec"?y.ExecBytes:y.BeaconBytes):"unknown (disk usage hasn't loaded)";Q(`
        <h2>Clear ${t(I)} data</h2>
        <p class="error">
          This stops the ${t(I.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${t(L)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${t(b)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,d=>{if(d==="cancel"){K();return}d==="confirm"&&fe(b)});const A=document.getElementById("clear-confirm-input"),z=document.getElementById("clear-confirm-btn");A==null||A.addEventListener("input",()=>{z&&(z.disabled=A.value.trim()!==b)}),A==null||A.focus()}async function fe(b){const I=document.getElementById("clear-confirm-btn");I&&(I.disabled=!0,I.textContent="Clearing…");try{await mt(c,b),K(),l()}catch(L){const A=Fe();if(A){const z=document.createElement("p");z.className="error small",z.textContent=`Clear failed: ${L instanceof Error?L.message:String(L)}`,A.appendChild(z)}I&&(I.disabled=!1,I.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),K()}}const Xe=500,Qe="valve-node-app.explain-consent";function qt(n,c){let o=!1,e=null;const u=[];n.innerHTML=`
    <h1>Logs: ${t(c)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ee()}</div>
  `;const f=n.querySelector("#logs-body"),x=n.querySelector("#logs-footer");ye(n,w=>{w==="explain"&&M()}),y();async function y(){let w,l;try{const[E,B]=await Promise.all([xe(),Se()]);w=E.find(Y=>Y.id===c),l=B}catch(E){if(o)return;f.innerHTML=`<p class="error">Failed to load target: ${t(String(E))}</p>`;return}if(o)return;if(!w){f.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!w.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const g=l==null?void 0:l.networks.find(E=>E.ChainID===w.wire.ChainID);g&&(x.innerHTML=ee(g.Name,g.LearnURL));try{const E=await pt(c,200);if(o)return;u.push(...E)}catch(E){if(o)return;f.innerHTML=`<p class="error">Failed to load logs: ${t(String(E))}</p>`;return}N(),e=ht(c,E=>{o||(u.push(E),u.length>Xe&&u.splice(0,u.length-Xe),N())})}function N(){const w=u.filter(g=>g.severity==="error"||g.severity==="critical");f.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(U).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${O(String(w.length),w.length?"bad":"neutral")}</h2>
          <div class="log-lines">${w.length?w.slice().reverse().map(U).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const l=f.querySelector(".log-lines");l&&(l.scrollTop=l.scrollHeight)}function U(w){const l=w.severity||"info",g=w.learnUrl?` <a href="${t(w.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${t(l)}">
        <span class="log-time">${t(new Date(w.at).toLocaleTimeString())}</span>
        <span class="log-unit">${t(w.unit)}</span>
        <span class="log-sev">${t(l)}</span>
        <span class="log-text">${t(w.line)}</span>
        ${w.explain?`<div class="log-explain">${t(w.explain)}${g}</div>`:""}
      </div>
    `}async function M(){const w=u.filter(g=>g.severity==="error"||g.severity==="critical").map(g=>g.line).slice(-40);if(!(localStorage.getItem(Qe)==="1")){q(w);return}await P(w)}function q(w){const l=w.length?`<pre class="explain-excerpt">${w.map(g=>t(g)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';F(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${l}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,g=>{g==="proceed"?(localStorage.setItem(Qe,"1"),C(),P(w)):C()})}async function P(w){F('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const l=w.length?await Ve(c,w):await Ve(c);if(o)return;F(`
        <h2>Explanation</h2>
        <div class="explain-text">${t(l.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${l.sentExcerpt.map(g=>t(g)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&C()})}catch(l){if(o)return;if(l instanceof ve&&l.status===409){F(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,g=>{g==="close"&&C()});return}F(`
        <h2>Explain failed</h2>
        <p class="error">${t(l instanceof Error?l.message:String(l))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&C()})}}function F(w,l){C();const g=document.createElement("div");g.className="modal-overlay",g.id="explain-modal",g.innerHTML=`<div class="modal">${w}</div>`,g.addEventListener("click",E=>{const B=E.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&l(B.dataset.modalAction),E.target===g&&l("cancel")}),document.body.appendChild(g)}function C(){var w;(w=document.getElementById("explain-modal"))==null||w.remove()}return()=>{o=!0,e==null||e(),C()}}function Wt(n,c){let o=!1,e=null,u=null,f=!1,x=!1;n.innerHTML=`<h1>Network diagnostics: ${t(c)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ee()}</div>`;const y=n.querySelector("#diag-body"),N=n.querySelector("#diag-footer");ye(n,(l,g)=>{var E;if(l==="run")M();else if(l==="toggle")(E=g.closest(".check-item"))==null||E.classList.toggle("expanded");else if(l==="copy"){const B=g.dataset.copy;B&&w(g,B)}}),U();async function U(){let l,g;try{const[B,Y]=await Promise.all([xe(),Se()]);l=B.find(X=>X.id===c),g=Y}catch(B){if(o)return;y.innerHTML=`<p class="error">Failed to load target: ${t(String(B))}</p>`;return}if(o)return;if(!l){y.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){y.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const E=g==null?void 0:g.networks.find(B=>B.ChainID===l.wire.ChainID);E&&(N.innerHTML=ee(E.Name,E.LearnURL));try{e=await $t(c),x=!0}catch(B){u=String(B instanceof Error?B.message:B)}o||q()}async function M(){f=!0,u=null,q();try{e=await yt(c),x=!0}catch(l){u=String(l instanceof Error?l.message:l)}f=!1,o||q()}function q(){y.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(c)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${f?"disabled":""}>${f?"Running…":"Run diagnostics"}</button>
      </div>
      ${u?`<p class="error">${t(u)}</p>`:""}
      ${P()}
    `}function P(){if(!x&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const l=new Date(e.at).toLocaleString(),g=e.failedId?`<p><strong>Failed at: ${t(F(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${t(l)} — trigger: ${t(e.trigger)}</p>
      ${g}
      <ul class="check-list">${e.items.map(C).join("")}</ul>
    `}function F(l){var g;return((g=e==null?void 0:e.items.find(E=>E.ID===l))==null?void 0:g.Title)??l}function C(l){const g=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral",E=l.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${E?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${O(E?"failed here":l.Status,g)}
          <strong>${t(l.Title)}</strong>
          <span class="muted small check-detail-inline">${t(l.Detail)}</span>
        </button>
        <div class="check-body">
          <details${E?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${t(l.Why)}</p>
          </details>
          ${l.Fix?`
                <details${E?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${t(l.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${t(l.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function w(l,g){const E=await Ie(g),B=l.textContent;l.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=B)},1500)}return()=>{o=!0}}function _t(n,c){let o=!1,e=[],u=null,f=!1,x=!1;n.innerHTML=`<h1>Security: ${t(c)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ee()}</div>`;const y=n.querySelector("#sec-body"),N=n.querySelector("#sec-footer");ye(n,(C,w)=>{var l;if(C==="rerun")M();else if(C==="toggle")(l=w.closest(".check-item"))==null||l.classList.toggle("expanded");else if(C==="copy"){const g=w.dataset.copy;g&&F(w,g)}}),U();async function U(){let C,w;try{const[g,E]=await Promise.all([xe(),Se()]);C=g.find(B=>B.id===c),w=E}catch(g){if(o)return;y.innerHTML=`<p class="error">Failed to load target: ${t(String(g))}</p>`;return}if(o)return;if(!C){y.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){y.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const l=w==null?void 0:w.networks.find(g=>g.ChainID===C.wire.ChainID);l&&(N.innerHTML=ee(l.Name,l.LearnURL)),await M()}async function M(){f=!0,u=null,q();try{e=await vt(c),x=!0}catch(C){u=String(C instanceof Error?C.message:C)}f=!1,o||q()}function q(){y.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(c)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${f?"disabled":""}>${f?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${t(u)}</p>`:""}
      ${!x&&f?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(P).join("")}</ul>`:x?'<p class="muted">No checks returned.</p>':""}
    `}function P(C){const w=C.Status==="pass"?"ok":C.Status==="fail"?"bad":C.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${O(C.Status,w)}
          <strong>${t(C.Title)}</strong>
          <span class="muted small check-detail-inline">${t(C.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${t(C.Why)}</p>
          </details>
          ${C.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${t(C.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${t(C.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function F(C,w){const l=await Ie(w),g=C.textContent;C.textContent=l?"Copied!":"Copy failed",setTimeout(()=>{o||(C.textContent=g)},1500)}return()=>{o=!0}}const Kt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function zt(n){let c=!1,o=!1,e=!1,u=null,f=!1,x=null,y=null;n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ee()}`;const N=n.querySelector("#settings-body");ye(n,P=>{if(P==="save"&&q(),P==="clear-key"){if(!x)return;o=!0;const F=n.querySelector("#ai-key");F&&(F.value=""),M(x)}}),ze(n,(P,F)=>{P!=="ai-provider"||!x||(y=F,f=!1,M(x))}),U();async function U(){try{const P=await Ht();if(c)return;x=P,M(P)}catch(P){if(c)return;N.innerHTML=`<p class="error">Failed to load settings: ${t(String(P))}</p>`}}function M(P){var w;const F=y??P.aiProvider;N.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Oe("ai-provider",Kt.map(l=>({value:l.value,label:l.label})),F)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${P.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${P.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${t(P.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${u?`<p class="error">${t(u)}</p>`:""}
        ${f?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const C=n.querySelector("#ai-key");C==null||C.addEventListener("input",()=>{o=!0,f=!1}),(w=n.querySelector("#ref-rpc-base"))==null||w.addEventListener("input",()=>{f=!1})}async function q(){const P=n.querySelector("#ai-key"),F=n.querySelector("#ref-rpc-base");if(!P||!F||!x)return;const C={aiProvider:y??x.aiProvider,refRpcBase:F.value.trim()};o&&(C.aiKey=P.value),e=!0,u=null,f=!1,M(x);try{const w=await Dt(C);if(c)return;x=w,o=!1,e=!1,f=!0,M(w)}catch(w){if(c)return;e=!1,u=String(w instanceof Error?w.message:w),M(x)}}return()=>{c=!0}}const Jt=6,Gt="run",Vt={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Yt(n){let c=!1,o=null,e=null;const u={},f={},x={},y={},N={};let U=null;n.innerHTML=`
    <div class="page-head">
      <h1>RPC</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      eRPC sits above everything else here. One gateway fronts as many chains as you
      list, and each chain can be served by a devnet on this machine, a node on any
      machine you manage, or a public endpoint — a gateway names the machine it runs
      on, it does not belong to it.
    </p>
    <div id="rpc-body"><p class="muted">Loading…</p></div>
    ${ee()}
  `;const M=n.querySelector("#rpc-body");ye(n,(a,s)=>{ge(a,s)}),ze(n,(a,s)=>{if(a.startsWith("chain-")){const r=a.slice(6);u[r]=Number.parseInt(s,10),C()}}),q();async function q(){try{const a=await Pt();if(c)return;o=a,e=null;for(const s of a.gateways??[]){const r=s.networks??[],m=u[s.id];(m==null||!r.some(v=>v.chainId===m))&&(u[s.id]=r.length?r[0].chainId:null)}}catch(a){if(c)return;o=null,e=me(a)}C()}function P(a){return((o==null?void 0:o.gateways)??[]).find(s=>s.id===a)}function F(a,s){if(s!=null)return(a.networks??[]).find(r=>r.chainId===s)}function C(){if(c)return;if(e){M.innerHTML=`<p class="error">Could not read the gateways: ${t(e)}</p>`;return}if(!o){M.innerHTML='<p class="muted">Loading…</p>';return}const a=o.gateways??[];M.innerHTML=`
      ${a.map(l).join("")}
      ${a.length===0?w():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${a.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function w(){return((o==null?void 0:o.targets)??[]).length===0?`
        <div class="card empty-state">
          <p class="muted">
            No machines yet. A gateway is a container, so it has to run somewhere —
            add a machine on <a href="#/targets">Machines</a> first.
          </p>
        </div>
      `:`
      <div class="card empty-state">
        <p class="muted">
          No gateway yet. A gateway is one eRPC instance fronting however many chains you
          list; it addresses a chain by URL path, so a single port serves all of them —
          and the same path serves WebSocket.
        </p>
      </div>
    `}function l(a){const s=F(a,u[a.id]??null);return`
      <section class="rpc-gateway">
        ${g(a)}
        ${a.error?Y(a):""}
        ${a.blocked?`<div class="banner banner-warn">${t(a.blocked)}</div>`:""}
        ${(a.warnings??[]).map(r=>`<div class="banner banner-warn">${t(r)}</div>`).join("")}
        ${ne(a)}
        ${x[a.id]?`<p class="error small">${t(x[a.id])}</p>`:""}
        ${Z(a)}
        ${N[a.id]?ie(a):""}
        ${pe(a,s)}
      </section>
    `}function g(a){var r;const s=a.status.State==="running";return`
      <div class="rpc-bar${s?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${B(a)}
            <strong>${t(a.label)}</strong>
            ${E(a)}
            <span class="muted small">on ${t(a.placement.targetId)} · ${t(a.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(a.actions??[]).map(m=>X(a,m)).join("")}
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${t(a.id)}">
              ${N[a.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${t(a.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${s?`<code class="endpoint-url">${t(a.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(a.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${t(((r=(a.networks??[])[0])==null?void 0:r.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${t(a.baseUrl)}</code> once it is running.</span>`}
        </div>
        ${oe(a)}
      </div>
    `}function E(a){switch(a.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function B(a){return a.status.State==="running"?le("ok"):a.status.State==="unknown"?le("bad"):le("neutral")}function Y(a){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${t(a.error??"")}</div>
        ${a.hint?`<div class="small">${t(a.hint)}</div>`:""}
      </div>
    `}function X(a,s){const r=Vt[s];if(!r)return"";const m=f[a.id];return`
      <button class="${r.className}" data-action="gw-${s}" data-gid="${t(a.id)}"
              title="${t(r.title)}" ${m?"disabled":""}>
        ${m===s?'<span class="spinner" aria-label="working"></span>':t(r.label)}
      </button>
    `}function Z(a){const s=y[a.id]??[];return s.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${t(a.placement.targetId)}</p>
        <pre class="step-log">${t(s.join(`
`))}</pre>
      </div>
    `}function oe(a){const s=a.networks??[],r=u[a.id]??null,m=`
      <button class="chip chip-add" data-action="add-chain" data-gid="${t(a.id)}"
              title="Add a network for this gateway to front">+ Network</button>
    `;if(s.length===0)return`
        <div class="rpc-chiprow">
          <span class="muted small">No networks yet — eRPC refuses a configuration with none, so add one before creating the gateway.</span>
          ${m}
        </div>
      `;if(s.length>Jt){const v=s.map($=>({value:String($.chainId),label:`${$.name} (${$.chainId})${$.serviceable?"":" — no working endpoint"}`}));return`
        <div class="rpc-chiprow">
          <span class="muted small">Fronting ${s.length} networks</span>
          ${Oe(`chain-${a.id}`,v,r==null?null:String(r))}
          ${m}
        </div>
      `}return`
      <div class="rpc-chiprow">
        ${s.map(v=>ue(a,v,v.chainId===r)).join("")}
        ${m}
      </div>
    `}function ue(a,s,r){const m=!s.serviceable;return`
      <button class="chip card-selectable${r?" selected":""}${m?" chip-bad":""}"
              data-action="select-chain" data-gid="${t(a.id)}" data-chain="${s.chainId}"
              title="${t(m?`${s.name}: no endpoint on this chain can be used right now`:`${s.name} · ${s.path}`)}">
        <span class="chip-dot">${le(m?"bad":"ok")}</span>
        <span class="chip-name">${t(s.name)}</span>
        <span class="chip-id">${s.chainId}</span>
      </button>
    `}function pe(a,s){if(!s)return'<div class="card rpc-upstreams"><p class="muted small">Pick a network above to see the servers behind it.</p></div>';const r=s.upstreams??[];return`
      <div class="card rpc-upstreams">
        <div class="service-head">
          <h2>${t(s.name)} <span class="muted">· chain ${s.chainId}</span></h2>
          <div class="card-actions">
            <button class="btn" data-action="add-endpoint" data-gid="${t(a.id)}" data-chain="${s.chainId}">Add an endpoint</button>
            <button class="btn btn-ghost" data-action="remove-chain" data-gid="${t(a.id)}" data-chain="${s.chainId}">Remove network</button>
          </div>
        </div>
        ${s.url?`<div class="endpoint-row">${le("ok")}<span class="muted small">callers dial</span>
                 <code class="endpoint-url">${t(s.url)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(s.url)}">Copy</button></div>`:`<p class="muted small">Path <code>${t(s.path)}</code> — the full URL appears once the gateway is running.</p>`}
        ${(s.warnings??[]).map(m=>`<div class="banner banner-warn">${t(m)}</div>`).join("")}
        ${r.map(m=>he(a,s,m)).join("")}
        ${r.length===0?'<p class="muted small">No endpoint yet, so there is nowhere for calls on this path to go.</p>':""}
      </div>
    `}function he(a,s,r){const m=`${a.id}|${s.chainId}|${r.id}`,v=r.actions??[];return`
      <div class="upstream-row${r.problem?" upstream-row-bad":""}">
        <span class="upstream-state">${r.problem?le("bad"):le("ok")}</span>
        <div class="upstream-what">
          <div class="upstream-label">
            ${t(r.label)}
            ${r.local?O("preferred","ok"):O("fallback","neutral")}
            ${r.recentOnly?O("recent blocks only","warn"):""}
          </div>
          <code class="endpoint-url">${t(r.endpoint||"—")}</code>
          ${r.problem?`<div class="error small">${t(r.problem)}</div>`:""}
        </div>
        <div class="card-actions">
          ${v.includes("reset")?`<button class="btn" data-action="reset-devnet" data-key="${t(m)}" data-target="${t(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[a.id]?"disabled":""}>
                   ${f[a.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost" data-action="remove-endpoint" data-key="${t(m)}">Remove</button>
        </div>
      </div>
    `}function ie(a){const s=a.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${t(a.id)}-port" value="${s.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${t(a.id)}-bind" value="${t(s.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${t(s.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${te(a)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${t(a.id)}">Save settings</button>
        </div>
      </div>
    `}function te(a){const s=t(a.id),r=a.config.TLS??null,m=(r==null?void 0:r.Enabled)??!1,v=(r==null?void 0:r.CertSource)||"internal";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${s}-tls" ${m?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${s}-tls-host" value="${t((r==null?void 0:r.Hostname)??"")}"
               placeholder="gateway.example.com" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${s}-tls-port" value="${(r==null?void 0:r.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${s}-tls-source">
          <option value="internal" ${v==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${v==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${s}-tls-cert" value="${t((r==null?void 0:r.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${s}-tls-key" value="${t((r==null?void 0:r.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
    `}function ne(a){var m,v;const s=a.tls;if(!(s!=null&&s.enabled))return"";const r=[];return s.fallback&&r.push(`<div class="banner banner-warn">${t(s.fallback)}</div>`),s.error?r.push(`<div class="banner banner-warn">HTTPS front: ${t(s.error)}</div>`):((m=s.status)==null?void 0:m.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${t(s.containerName??"")}</code>) is
         ${t(((v=s.status)==null?void 0:v.State)??"unknown")}, so nothing is answering on
         <code>${t(s.url??"")}</code> even if the gateway itself is up.</div>`),s.rootCaPath&&s.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${t(s.rootCaPath)}</code> (on ${t(a.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function ae(a){return{...a.config,Networks:(a.config.Networks??[]).map(s=>({ChainID:s.ChainID,Upstreams:s.Upstreams.map(r=>({...r}))}))}}async function se(a,s,r){x[a]=null;try{await Rt(a,s)}catch(m){return x[a]=`${r?r+": ":""}${me(m)}`,C(),!1}return await q(),!0}async function ge(a,s){const r=s.dataset.gid??"";switch(a){case"refresh":await q();return;case"copy":s.dataset.copy&&await at(s,s.dataset.copy);return;case"select-chain":u[r]=Number.parseInt(s.dataset.chain??"",10),C();return;case"toggle-settings":N[r]=!N[r],C();return;case"save-settings":await fe(r);return;case"gw-start":case"gw-stop":case"gw-restart":await L(r,a.slice(3));return;case"gw-create":case"gw-recreate":await A(r);return;case"gw-wipe":qe(r);return;case"add-gateway":Ce();return;case"forget-gateway":await z(r);return;case"add-chain":d(r);return;case"remove-chain":await p(r,Number.parseInt(s.dataset.chain??"",10));return;case"add-endpoint":W(r,Number.parseInt(s.dataset.chain??"",10));return;case"remove-endpoint":await j(s.dataset.key??"");return;case"reset-devnet":await Re(s.dataset.key??"",s.dataset.target??"");return;default:return}}async function fe(a){const s=P(a);if(!s)return;const r=ae(s),m=n.querySelector(`#gw-${CSS.escape(a)}-port`),v=n.querySelector(`#gw-${CSS.escape(a)}-bind`);if(m){const H=Number.parseInt(m.value.trim(),10);Number.isFinite(H)&&(r.Port=H)}v&&(r.BindAddr=v.value.trim()),r.TLS=b(a,s);const $=s.status.State==="running";await se(a,r,"Saving settings")&&(N[a]=!1,$&&(x[a]=null,I(a,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),C())}function b(a,s){var $,H,T,G,re,Te,Je;const r=st=>n.querySelector(`#gw-${CSS.escape(a)}-${st}`),m=r("tls");if(!m)return s.config.TLS??null;const v=Number.parseInt((($=r("tls-port"))==null?void 0:$.value.trim())??"",10);return{Enabled:m.checked,Hostname:((H=r("tls-host"))==null?void 0:H.value.trim())??"",CertSource:((T=r("tls-source"))==null?void 0:T.value)??"internal",CertFile:((G=r("tls-cert"))==null?void 0:G.value.trim())??"",KeyFile:((re=r("tls-key"))==null?void 0:re.value.trim())??"",HTTPSPort:Number.isFinite(v)?v:443,BindAddr:((Te=s.config.TLS)==null?void 0:Te.BindAddr)??"",ImageRef:((Je=s.config.TLS)==null?void 0:Je.ImageRef)??""}}function I(a,s){y[a]=[s]}async function L(a,s){if(!f[a]){f[a]=s,x[a]=null,C();try{await Lt(a,s)}catch(r){x[a]=`${s} failed: ${me(r)}${Be(r)}`}f[a]=null,await q()}}async function A(a){if(f[a])return;f[a]="create",x[a]=null,y[a]=["starting…"],C();let s;try{s=await Nt(a)}catch(r){x[a]=`${me(r)}${Be(r)}`,y[a]=[],f[a]=null,C();return}U==null||U(),U=Ke(s.targetId,r=>{if(c)return;const m=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(y[a]=[...(y[a]??[]).filter($=>$!=="starting…"),m],!!r.err||r.stepId===Gt&&!!r.done){U==null||U(),U=null,f[a]=null,r.err&&(x[a]="Provisioning failed — see the log below."),q();return}C()})}async function z(a){const s=P(a);if(!(!s||!await Ee({title:`Forget ${s.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${s.containerName}" on ${s.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await It(a)}catch(m){x[a]=me(m),C();return}await q()}}function d(a){const s=P(a);if(!s)return;const r=new Set((s.networks??[]).map(T=>T.chainId)),m=(o==null?void 0:o.presets)??[],v=m.filter(T=>!r.has(T.chainId)),$=m.filter(T=>r.has(T.chainId)),H=((o==null?void 0:o.targets)??[]).some(T=>T.id===s.placement.targetId&&T.hasDevnet);Q(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${t(s.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${v.map(T=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${T.chainId}">
                <span>${t(T.name)}</span>
                <span class="muted small">chain ${T.chainId}${T.devnet?H?" · uses the devnet on "+t(s.placement.targetId):" · will create a devnet on "+t(s.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${$.length?`<p class="muted small">Already fronted: ${t($.map(T=>T.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,T=>{if(T==="cancel"){K();return}if(T==="custom"){h(a);return}if(T.startsWith("preset:")){const G=Number.parseInt(T.slice(7),10),re=m.find(Te=>Te.chainId===G);K(),re!=null&&re.devnet?i(a,G,H):k(a,G)}})}function h(a){var s;Q(`
        <h2>Add a custom network</h2>
        <p class="muted small">
          Any EVM chain id. Nothing here restricts it to the chains this app can run a node for —
          fronting somebody else's chain is a perfectly good use of a gateway.
        </p>
        <label>
          Chain id
          <input type="text" inputmode="numeric" id="custom-chain-id" autocomplete="off" placeholder="8453" />
        </label>
        <p class="muted small" id="custom-chain-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add">Add network</button>
        </div>
      `,r=>{if(r==="cancel"){K();return}if(r!=="add")return;const m=document.getElementById("custom-chain-id"),v=document.getElementById("custom-chain-err"),$=Number.parseInt((m==null?void 0:m.value.trim())??"",10);if(!Number.isFinite($)||$<=0){v&&(v.className="error small"),v&&(v.textContent="A chain id is a positive whole number.");return}K(),k(a,$)}),(s=document.getElementById("custom-chain-id"))==null||s.focus()}async function k(a,s){const r=P(a);if(!r)return;const m=ae(r),v=m.Networks??[];v.some($=>$.ChainID===s)||(v.push({ChainID:s,Upstreams:[]}),m.Networks=v,u[a]=s,await D(a,m)&&(u[a]=s,C(),W(a,s)))}async function D(a,s){var $;const r={...s,Networks:(s.Networks??[]).filter(H=>H.Upstreams.length>0)};if(!await se(a,r))return!1;const v=P(a);if(v)for(const H of s.Networks??[])H.Upstreams.length===0&&!(v.networks??[]).some(T=>T.chainId===H.ChainID)&&(v.config.Networks=[...v.config.Networks??[],{ChainID:H.ChainID,Upstreams:[]}],v.networks=[...v.networks??[],{chainId:H.ChainID,name:(($=((o==null?void 0:o.presets)??[]).find(T=>T.chainId===H.ChainID))==null?void 0:$.name)??`Chain ${H.ChainID}`,path:`/${v.config.ProjectID}/evm/${H.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function i(a,s,r){const m=P(a);if(!m)return;if(!r){Q(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${t(m.placement.targetId)}</code>, so adding chain ${s} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(m.placement.targetId)}" data-modal-action="go">Create a devnet on ${t(m.placement.targetId)}</a>
          </div>
        `,()=>K());return}const v=ae(m),$=v.Networks??[],H={ID:"devnet",Kind:"managed-devnet",TargetID:m.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},T=$.find(G=>G.ChainID===s);T?T.Upstreams.push(H):$.push({ChainID:s,Upstreams:[H]}),v.Networks=$,u[a]=s,await se(a,v,"Adding the devnet")}async function p(a,s){const r=P(a);if(!r||!Number.isFinite(s))return;const m=F(r,s);if(!await Ee({title:`Remove ${(m==null?void 0:m.name)??`chain ${s}`}`,body:`This gateway will stop serving ${(m==null?void 0:m.path)??`chain ${s}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const $=ae(r);$.Networks=($.Networks??[]).filter(H=>H.ChainID!==s),u[a]=null,await se(a,$,"Removing the network")}function S(a){const s=a.split("|");return s.length!==3?null:{gid:s[0],chainId:Number.parseInt(s[1],10),upstreamId:s[2]}}async function j(a){const s=S(a);if(!s)return;const r=P(s.gid);if(!r)return;const m=ae(r),v=(m.Networks??[]).find(T=>T.ChainID===s.chainId);if(!v)return;const $=v.Upstreams.findIndex((T,G)=>(T.ID||`${s.chainId}-${G}`)===s.upstreamId);$<0||!await Ee({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(v.Upstreams.splice($,1),await se(s.gid,m,"Removing the endpoint"))}function W(a,s){const r=P(a);if(!r||!Number.isFinite(s))return;const m=((o==null?void 0:o.sources)??[]).filter(T=>T.chainId===s),v=F(r,s),$=new Set(((v==null?void 0:v.upstreams)??[]).filter(T=>T.kind!=="external").map(T=>`${T.kind}|${T.targetId??""}`)),H=m.filter(T=>!$.has(`${T.kind}|${T.targetId}`));Q(`
        <h2>Add an endpoint for ${t((v==null?void 0:v.name)??`chain ${s}`)}</h2>
        ${H.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${H.map(T=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${t(T.kind)}:${t(T.targetId)}">
                       <span>${t(T.label)}</span>
                       <span class="muted small">${t(T.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${s}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn btn-ghost" data-modal-action="discover">Find public endpoints…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,T=>{if(T==="cancel"){K();return}if(T==="discover"){V(a,s);return}if(T==="manual"){J(a,s);return}if(T.startsWith("source:")){const[,G,re]=T.split(":");K(),R(a,s,G,re)}})}async function R(a,s,r,m){const v=P(a);if(!v)return;const $=ae(v),H=$.Networks??[],T={ID:`${r==="managed-devnet"?"devnet":"node"}-${m}`,Kind:r,TargetID:m,Endpoint:"",Local:!0,RecentOnly:!1},G=H.find(re=>re.ChainID===s);G?G.Upstreams.push(T):H.push({ChainID:s,Upstreams:[T]}),$.Networks=H,await se(a,$,"Adding the endpoint")}async function V(a,s){Q(`
        <h2>Public endpoints for chain ${s}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,$=>{$==="cancel"&&K()});let r;try{r=await At(s)}catch($){const H=Fe();if(H){const T=document.createElement("p");T.className="error small",T.textContent=`Could not discover endpoints: ${me($)}`,H.appendChild(T)}return}if(c)return;const m=(r.endpoints??[]).filter($=>$.status==="live"||$.status==="unprobed"),v=(r.endpoints??[]).filter($=>$.status==="rejected");Q(`
        <h2>Public endpoints for chain ${s}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${t(r.fetchError)}</div>`:""}</div>`:""}
        ${m.length?`<p class="muted small">${m.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${m.map($=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent($.url)}">
                       <span><code>${t($.url)}</code></span>
                       <span class="muted small">${$.status==="live"?`answered in ${$.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${s} right now.</p>`}
        ${v.length?`<details class="rpc-rejected">
                 <summary class="muted small">${v.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${v.map($=>`<li class="muted small"><code>${t($.url)}</code> — ${t($.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,$=>{if($==="cancel"){K();return}$.startsWith("add:")&&(K(),$e(a,s,decodeURIComponent($.slice(4))))})}function J(a,s){var r;Q(`
        <h2>Add an endpoint by URL</h2>
        <p class="muted small">
          http://, https://, ws:// or wss://. eRPC infers WebSocket from the scheme — there is no
          separate setting — and a ws upstream also serves ordinary calls.
        </p>
        <label>
          Endpoint
          <input type="text" id="manual-endpoint" autocomplete="off" spellcheck="false" placeholder="https://rpc.example.com" />
        </label>
        <label class="radio">
          <input type="checkbox" id="manual-recent" />
          Recent blocks only <span class="muted">— tick for a pruned node that cannot answer historical state</span>
        </label>
        <p class="muted small" id="manual-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add">Add endpoint</button>
        </div>
      `,m=>{if(m==="cancel"){K();return}if(m!=="add")return;const v=document.getElementById("manual-endpoint"),$=document.getElementById("manual-recent"),H=document.getElementById("manual-err"),T=(v==null?void 0:v.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(T)){H&&(H.className="error small",H.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}K(),$e(a,s,T,($==null?void 0:$.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function $e(a,s,r,m=!1){const v=P(a);if(!v)return;const $=ae(v),H=$.Networks??[],T=H.find(Te=>Te.ChainID===s),G=((T==null?void 0:T.Upstreams.length)??0)+1,re={ID:`public-${s}-${G}`,Kind:"external",Endpoint:r,Local:!1,RecentOnly:m};T?T.Upstreams.push(re):H.push({ChainID:s,Upstreams:[re]}),$.Networks=H,await se(a,$,"Adding the endpoint")}async function Re(a,s){const r=S(a);if(!r||!s||!await Ee({title:"Reset this devnet",body:`The chain on ${s} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[r.gid]="reset",x[r.gid]=null,C();let v;try{v=await St(s)}catch($){x[r.gid]=`Reset failed: ${me($)}${Be($)}`,f[r.gid]=null,C();return}f[r.gid]=null,Le(s,v),await q()}function Le(a,s){const r=[];r.push(s.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),s.report.Recreated&&r.push("A fresh chain was started from genesis.");const m=s.report.Cascaded??[],v=s.report.CascadeSkipped??[];Q(`
        <h2>Devnet on ${t(a)} reset</h2>
        <ul class="plain-list">${r.map($=>`<li>${t($)}</li>`).join("")}</ul>
        ${m.length?`<p class="ok">Restarted in front of it: ${t(m.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${v.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(v.join(", "))}.</p>`:""}
        ${s.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(s.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>K())}function qe(a){const s=P(a);if(!s)return;Q(`
        <h2>Wipe ${t(s.label)}</h2>
        <p class="error">This destroys ${t(s.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${t(a)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(a)}</button>
        </div>
      `,v=>{if(v==="cancel"||v==="close"){K(),q();return}v==="confirm"&&Ne(a)});const r=document.getElementById("wipe-confirm-input"),m=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{m&&(m.disabled=r.value.trim()!==a)}),r==null||r.focus()}async function Ne(a){const s=document.getElementById("wipe-confirm-btn");s&&(s.disabled=!0,s.textContent="Wiping…");let r;try{r=await Bt(a)}catch(m){const v=Fe();if(v){const $=document.createElement("p");$.className="error small",$.textContent=`Wipe failed: ${me(m)}${Be(m)}`,v.appendChild($)}s&&(s.disabled=!1,s.textContent=`Wipe ${a}`);return}Q(`
        <h2>${t(a)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${t(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{K(),q()})}function Ce(){var m;const a=(o==null?void 0:o.targets)??[],s=new Set(((o==null?void 0:o.gateways)??[]).map(v=>v.id));if(a.length===0){Q(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>K());return}const r=s.has("default")?"":"default";Q(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${t(r)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${a.map(v=>`<option value="${t(v.id)}">${t(v.id)} (${t(v.mode)})</option>`).join("")}
          </select>
        </label>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="new-gw-port" value="4000" autocomplete="off" />
        </label>
        <p class="muted small" id="new-gw-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="create">Create gateway</button>
        </div>
      `,v=>{if(v==="cancel"){K();return}v==="create"&&nt()}),(m=document.getElementById("new-gw-id"))==null||m.focus()}async function nt(){const a=document.getElementById("new-gw-id"),s=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),m=document.getElementById("new-gw-err"),v=(a==null?void 0:a.value.trim())??"",$=(s==null?void 0:s.value)??"",H=Number.parseInt((r==null?void 0:r.value.trim())??"",10),T=G=>{m&&(m.className="error small",m.textContent=G)};if(!v){T("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!$){T("Pick the machine it runs on.");return}try{await Et({id:v,placement:{targetId:$,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(H)?H:4e3,Networks:[]}})}catch(G){T(me(G));return}K(),await q()}async function at(a,s){const r=await Ie(s),m=a.textContent;a.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{c||(a.textContent=m)},1500)}function me(a){return a instanceof Error?a.message:String(a)}function Be(a){return a instanceof ve&&a.hint?` — ${a.hint}`:""}return()=>{c=!0,U==null||U(),K()}}const Zt="run",Xt={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Qt={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function en(n,c){let o=!1,e=null,u=null;const f={devnet:null},x={devnet:null},y={devnet:[]};let N=null;const U={devnet:!1};let M=null;const q={devnet:null},P={devnet:null};n.innerHTML=`
    <div class="page-head">
      <h1>Services: ${t(c)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${ee()}
  `;const F=n.querySelector("#services-body");ye(n,(i,p)=>{ge(i,p)}),C();async function C(){try{const i=await wt(c);if(o)return;e=i,u=null}catch(i){if(o)return;e=null,u=k(i)}l()}function w(i){return e==null?void 0:e.services.find(p=>p.id===i)}function l(){if(!o){if(u){F.innerHTML=`<p class="error">Could not read this machine's services: ${t(u)}</p>`;return}if(!e){F.innerHTML='<p class="muted">Loading…</p>';return}F.innerHTML=`
      ${g(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(E).join("")}
      </div>
    `}}function g(i){if(i.present&&i.reachable&&!i.hint)return`<p class="muted small">Docker: ${t(i.flavor)}${i.serverVersion?` ${t(i.serverVersion)}`:""} · reachable</p>`;const p=i.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${t(p)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${i.detail?`<div class="small">${t(i.detail)}</div>`:""}
        ${i.hint?`<div class="small">${t(i.hint)}</div>`:""}
      </div>
    `}function E(i){const p=i.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${t(i.label)}</h2>
          ${B(i)}
        </div>
        <p class="muted small">${t(Xt[i.id]??"")}</p>

        ${i.error?Y(i):""}
        ${i.blocked?`<div class="banner banner-warn">${t(i.blocked)}</div>`:""}
        ${p.map(S=>`<div class="banner banner-warn">${t(S)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${t(i.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${i.status.Image?`<code>${t(i.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${X(i)}

        ${Z(i)}

        <div class="card-actions">
          ${(i.actions??[]).map(S=>oe(i,S)).join("")}
        </div>
        ${x[i.id]?`<p class="error small">${t(x[i.id])}</p>`:""}
        ${ue(i)}

        ${pe(i)}
      </div>
    `}function B(i){switch(i.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function Y(i){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${t(i.error??"")}</div>
        ${i.hint?`<div class="small">${t(i.hint)}</div>`:""}
      </div>
    `}function X(i){if(i.status.State!=="created-but-stopped"||i.status.ExitCode===0)return"";const p=i.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${i.status.ExitCode}${p}.</p>`}function Z(i){const p=i.endpoints??[];return p.length===0?i.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":p.map(S=>`
        <div class="endpoint-row">
          ${le("ok")}
          <span class="muted small">${t(S.label)}</span>
          <code class="endpoint-url">${t(S.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(S.url)}">Copy</button>
        </div>`).join("")}function oe(i,p){const S=Qt[p];if(!S)return"";const j=f[i.id],W=p==="create"?`Create ${i.id==="devnet"?"devnet":"gateway"}`:S.label;return`
      <button class="${S.className}" data-action="svc-${p}" data-svc="${t(i.id)}"
              title="${t(S.title)}" ${j?"disabled":""}>
        ${j===p?'<span class="spinner" aria-label="working"></span>':t(W)}
      </button>
    `}function ue(i){const p=y[i.id]??[];return p.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${t(p.join(`
`))}</pre>
      </div>
    `}function pe(i){const p=U[i.id],S=he(i);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${i.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${t(i.id)}">
            ${p?"Close":"Edit"}
          </button>
        </div>
        ${p?ie():`<p class="small">${S}</p>`}
        ${q[i.id]?`<p class="error small">${t(q[i.id])}</p>`:""}
        ${P[i.id]?`<p class="muted small">${t(P[i.id])}</p>`:""}
      </div>
    `}function he(i){const p=i.devnet;return p?`Chain ${p.ChainID} · a block every ${t(p.BlockTime)} · JSON-RPC on ${t(p.BindAddr)}:${p.HTTPPort} · WebSocket on ${t(p.BindAddr)}:${p.WSPort}`:"—"}function ie(i){return te()}function te(){const i=M;return i?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${t(i.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${i.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${i.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${t(i.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${i.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function ne(){U.devnet&&M&&(M.BlockTime=ae("#dev-blocktime",M.BlockTime),M.HTTPPort=se("#dev-http",M.HTTPPort),M.WSPort=se("#dev-ws",M.WSPort),M.BindAddr=ae("#dev-bind",M.BindAddr))}function ae(i,p){const S=n.querySelector(i);return S?S.value.trim():p}function se(i,p){const S=n.querySelector(i);if(!S)return p;const j=Number.parseInt(S.value.trim(),10);return Number.isFinite(j)?j:p}async function ge(i,p){const S=p.dataset.svc??"";switch(i){case"refresh":await C();return;case"copy":p.dataset.copy&&await h(p,p.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await fe(S,i.slice(4));return;case"svc-create":case"svc-recreate":await b(S);return;case"svc-wipe":A(S);return;case"toggle-config":I(S);return;case"save-config":await L(S);return;default:return}}async function fe(i,p){if(!f[i]){f[i]=p,x[i]=null,l();try{await kt(c,i,p)}catch(S){x[i]=`${p} failed: ${k(S)}${D(S)}`}f[i]=null,await C()}}async function b(i){if(!f[i]){f[i]="create",x[i]=null,y[i]=["starting…"],l();try{await Tt(c,i)}catch(p){x[i]=`${k(p)}${D(p)}`,y[i]=[],f[i]=null,l();return}N==null||N(),N=Ke(c,p=>{if(o)return;const S=p.err?`${p.stepId}: ${p.err}`:p.line?`${p.stepId}: ${p.line}`:`${p.stepId}: done`;if(y[i]=[...(y[i]??[]).filter(W=>W!=="starting…"),S],!!p.err||p.stepId===Zt&&!!p.done){N==null||N(),N=null,f[i]=null,p.err&&(x[i]="Provisioning failed — see the log below."),C();return}l()})}}function I(i){if(ne(),U[i]=!U[i],q[i]=null,P[i]=null,U[i]){const p=w(i);p!=null&&p.devnet&&(M={...p.devnet})}l()}async function L(i){var j;ne(),q[i]=null,P[i]=null;const p=M;if(!p)return;if(p.HTTPPort===p.WSPort){q[i]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",l();return}try{await xt(c,i,p)}catch(W){q[i]=k(W),l();return}const S=((j=w(i))==null?void 0:j.status.State)==="running";U[i]=!1,P[i]=S?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await C()}function A(i){const p=w(i);if(!p)return;const S=(p.restartsOnWipe??[]).map(R=>{var V;return((V=w(R))==null?void 0:V.label)??R});Q(`
        <h2>Wipe ${t(p.label)}</h2>
        <p class="error">This deletes ${t(p.wipeDiscards)}</p>
        ${S.length?`<p>It also restarts what sits in front of it: ${t(S.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${t(i)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(i)}</button>
        </div>
      `,R=>{if(R==="cancel"||R==="close"){K(),C();return}R==="confirm"&&z(i)});const j=document.getElementById("wipe-confirm-input"),W=document.getElementById("wipe-confirm-btn");j==null||j.addEventListener("input",()=>{W&&(W.disabled=j.value.trim()!==i)}),j==null||j.focus()}async function z(i){const p=document.getElementById("wipe-confirm-btn");p&&(p.disabled=!0,p.textContent="Wiping…");let S;try{S=await Ct(c,i)}catch(j){const W=Fe();if(W){const R=document.createElement("p");R.className="error small",R.textContent=`Wipe failed: ${k(j)}${D(j)}`,W.appendChild(R)}p&&(p.disabled=!1,p.textContent=`Wipe ${i}`);return}d(i,S)}function d(i,p){const S=w(i),j=J=>{var $e;return(($e=w(J))==null?void 0:$e.label)??J},W=[];W.push(p.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const J of p.report.VolumesRemoved??[])W.push(`Volume ${J} deleted.`);for(const J of p.report.VolumesAbsent??[])W.push(`Volume ${J} was already gone.`);p.report.Recreated&&W.push("Container re-created from your saved configuration.");const R=(p.report.Cascaded??[]).map(j),V=(p.report.CascadeSkipped??[]).map(j);Q(`
        <h2>${t((S==null?void 0:S.label)??i)} wiped</h2>
        <ul class="plain-list">${W.map(J=>`<li>${t(J)}</li>`).join("")}</ul>
        ${R.length?`<p class="ok">Restarted in front of it: ${t(R.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${V.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(V.join(", "))}.</p>`:""}
        ${p.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(p.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,J=>{(J==="close"||J==="cancel")&&(K(),C())})}async function h(i,p){const S=await Ie(p),j=i.textContent;i.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{o||(i.textContent=j)},1500)}function k(i){return i instanceof Error?i.message:String(i)}function D(i){return i instanceof ve&&i.hint?` — ${i.hint}`:""}return()=>{o=!0,N==null||N(),K()}}const tn="local";function nn(n){let c=!1,o=!1,e="",u=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ee()}
  `;const f=n.querySelector("#targets-body");ye(n,(l,g)=>{M(l,g)}),x();async function x(){try{const[l,g,E]=await Promise.all([xe(),Se(),it()]);if(c)return;e=E.os,N(l,g)}catch(l){if(c)return;f.innerHTML=`<p class="error">Failed to load machines: ${t(String(l))}</p>`}}function y(){u&&N(u.targets,u.catalog)}function N(l,g){u={targets:l,catalog:g};const E=e==="linux",B=[...l].sort((Z,oe)=>(Z.mode==="local"?-1:0)-(oe.mode==="local"?-1:0)),Y=B.length?`<div class="card-grid">${B.map(Z=>an(Z,g,Z.mode!=="local"||E,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=l.some(Z=>Z.mode==="local");f.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${Y}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${U(E,X)}
        ${o?sn():""}
      </section>
    `}function U(l,g){const E=`
      <div class="card">
        <h3>A server over SSH ${O("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${l?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${l?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,B=l?`
        <div class="card">
          <h3>This machine ${O("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${t(e)})`:""} ${O("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return g?`<div class="card-grid card-grid-wide">${E}</div>`:`<div class="card-grid card-grid-wide">${l?B+E:E+B}</div>`}async function M(l,g){var E;if(l==="add-local"){await q();return}if(l==="delete-target"){const B=g.dataset.id;if(!B||!await Ee({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await P(B);return}if(l==="toggle-ssh"){o=!o,w(),y(),o&&((E=n.querySelector("#ssh-host"))==null||E.focus());return}l==="add-ssh"&&await F()}async function q(){w();try{await Ge({id:tn,mode:"local"}),await x()}catch(l){C(l)}}async function P(l){try{await ct(l),await x()}catch(g){C(g)}}async function F(){const l=n.querySelector("#ssh-host"),g=n.querySelector("#ssh-user"),E=n.querySelector("#ssh-key"),B=n.querySelector("#ssh-port"),Y=n.querySelector("#ssh-id");if(!l||!g||!E||!B||!Y)return;const X=l.value.trim(),Z=g.value.trim(),oe=E.value.trim(),ue=B.value.trim(),pe=Y.value.trim();if(w(),!X||!Z||!oe){C(new Error("host, user, and key path are required"));return}const he=pe||on(X),ie={Host:X,User:Z,KeyPath:oe};if(ue){const ne=Number.parseInt(ue,10);if(!Number.isFinite(ne)||ne<=0){C(new Error("port must be a positive number"));return}ie.Port=ne}const te=n.querySelector("#ssh-submit");te&&(te.disabled=!0,te.textContent="Connecting…");try{await Ge({id:he,mode:"ssh",ssh:ie}),o=!1,await x()}catch(ne){C(ne),te&&(te.disabled=!1,te.textContent="Add server")}}function C(l){let g=n.querySelector("#targets-error");g||(f.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),g=n.querySelector("#targets-error")),g.textContent=String(l instanceof Error?l.message:l)}function w(){var l;(l=n.querySelector("#targets-error"))==null||l.remove()}return()=>{c=!0}}function an(n,c,o,e){const u=n.wire,f=n.mode==="local"?"this machine":"SSH",x=n.mode==="ssh"&&n.ssh?`${t(n.ssh.User)}@${t(n.ssh.Host)}`:f,y=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(n.id)}">Devnet</a>`;let N,U;if(!u&&!o)N=`${O("can't run a node","warn")} ${O(e||"not Linux","neutral")}`,U=`
      ${y}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(n.id)}">Preview setup wizard</a>
    `;else if(!u)N=O("not set up","neutral"),U=`
      <a class="btn" href="#/setup/${encodeURIComponent(n.id)}">Run setup wizard</a>
      ${y}
    `;else{const M=c.networks.find(P=>P.ChainID===u.ChainID),q=M?M.Name:`chain ${u.ChainID}`;N=`${O(q,"ok")} ${O(u.ExecID,"neutral")} ${O(u.BeaconID,"neutral")}${u.Archive?" "+O("archive","warn"):""}`,U=`
      <a class="btn" href="#/dash/${encodeURIComponent(n.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(n.id)}">Logs</a>
      ${y}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(n.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${t(n.id)}</h2>
      <p class="muted">${x}</p>
      <p>${N}</p>
      <div class="card-actions">
        ${U}
        <button class="btn btn-danger" data-action="delete-target" data-id="${t(n.id)}">Remove</button>
      </div>
    </div>
  `}function sn(){return`
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
  `}function on(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const _e=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],He=8545,De=5052,Ue=30303,rn=[369,943,1],et={369:"default",943:"practise here first"};function cn(n,c){let o=!1;const e={targetId:c,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${t(c)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ee()}</div>`;const u=n.querySelector("#wizard-body"),f=n.querySelector("#wizard-footer");ye(n,(d,h)=>{se(d,h)}),ze(n,(d,h)=>{d==="exec-select"?e.execId=h:d==="beacon-select"&&(e.beaconId=h),y()}),n.addEventListener("change",d=>{const h=d.target;h instanceof HTMLInputElement&&(h.id==="data-dir-input"?(ge(),oe()):h.id==="checkpoint-toggle"?(e.checkpoint=h.checked,y()):h.id==="exec-snapshot-toggle"&&(e.execSnapshot=h.checked,y()))}),x();async function x(){try{const[d,h]=await Promise.all([Se(),xe()]);if(o)return;e.catalog=d;const k=h.find(D=>D.id===c);k!=null&&k.wire&&(e.chainId=k.wire.ChainID,e.execId=k.wire.ExecID,e.beaconId=k.wire.BeaconID,e.archive=k.wire.Archive,k.wire.ExecHTTPPort&&(e.execHTTPPort=String(k.wire.ExecHTTPPort)),k.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(k.wire.BeaconHTTPPort)),k.wire.ExecP2PPort&&(e.execP2PPort=String(k.wire.ExecP2PPort)),k.wire.RPCBindAddr&&(e.rpcBindAddr=k.wire.RPCBindAddr)),y()}catch(d){if(o)return;e.loadError=String(d instanceof Error?d.message:d),y()}}function y(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${t(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${z(e.step)}
      ${U()}
    `,N())}function N(){var h;const d=(h=e.catalog)==null?void 0:h.networks.find(k=>k.ChainID===e.chainId);f.innerHTML=d?ee(d.Name,d.LearnURL):ee()}function U(){switch(e.step){case"network":return M();case"clients":return q();case"mode":return te();case"review":return ne();case"run":return ae()}}function M(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${rn.map(k=>{const D=d.networks.find(S=>S.ChainID===k);if(!D)return"";const i=e.chainId===k,p=et[k]?O(et[k],k===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${i?"selected":""}" data-action="pick-network" data-chain-id="${k}" type="button">
          <h3>${t(D.Name)} <span class="muted">(chain ${k})</span></h3>
          ${p}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function q(){const d=e.catalog,h=d.networks.find(i=>i.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const k=h.ExecClients.map(i=>pe(i,d)),D=h.BeaconClients.map(i=>pe(i,d));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${t(h.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${Oe("exec-select",k,e.execId)}
        </label>
        ${ie(e.execId,d)}
        <label>
          Beacon client
          ${Oe("beacon-select",D,e.beaconId)}
        </label>
        ${ie(e.beaconId,d)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function P(d){return d<=0?"—":d>=1?`~${d.toFixed(1)} TB`:`~${Math.round(d*1e3)} GB`}const F=1.1,C=.5,w="Valve reth snapshot",l="rough estimate";function g(d){return d.SnapshotSizeTB}function E(d){return d.SnapshotSizeTB*C}function B(d){return`<p class="muted small">${P(g(d))} is the measured size of Valve's reth snapshot for ${t(d.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function Y(d){return{archive:g(d)*1e12*F,full:E(d)*1e12*F}}function X(d,h){if(!d)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${t(h)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${t(h)}</code>: ${t(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==h)return"";const k=Y(d),D=e.freeBytes>=k.archive,i=e.freeBytes>=k.full,p=`<p class="muted small">Free at <code>${t(h)}</code>: <strong>${ke(e.freeBytes)}</strong> — archive ${D?"fits":"won't fit"} (${P(g(d))}, ${w}), full ${i?"fits":"won't fit"} (${P(E(d))}, ${l}).</p>`;let S="";return e.downgradeNote?S=`<p class="banner banner-warn">${t(e.downgradeNote)}</p>`:i||(S=`<p class="banner banner-warn">Neither full (${P(E(d))}, ${l}) nor archive (${P(g(d))}, ${w}) fits the free space here — choose a location with more room.</p>`),p+S}function Z(d,h){if(e.downgradeNote=null,!d||e.freeBytes===null)return;const k=Y(d);e.archive&&e.freeBytes<k.archive&&e.freeBytes>=k.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${h} for archive (${P(g(d))}, ${w}) — switched to Full (${P(E(d))}, ${l}). Pick a location with more room to run archive.`)}async function oe(){var k;if(e.chainId===null)return;const d=(k=e.catalog)==null?void 0:k.networks.find(D=>D.ChainID===e.chainId),h=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,y();try{const{freeBytes:D}=await lt(e.targetId,h);if(o)return;e.freeBytes=D,e.probedPath=h,Z(d,h)}catch(D){if(o)return;e.freeBytes=null,e.probedPath=h,e.diskError=String(D instanceof Error?D.message:D)}e.diskProbing=!1,y()}function ue(d){return d?/^https?:\/\/.+/i.test(d)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function pe(d,h){const k=h.clients.find(D=>D.id===d);return{value:d,label:k?`${k.id} — ${he(k.repo)}`:d}}function he(d){const h=d.split("/");return h.length>=4?h[3]:d}function ie(d,h){const k=d?h.clients.find(i=>i.id===d):void 0;if(!k)return"";const D=k.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${t(k.repo)}" target="_blank" rel="noopener noreferrer">${t(D)}</a></p>`}function te(){var j,W,R;const d=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",h=(j=e.catalog)==null?void 0:j.networks.find(V=>V.ChainID===e.chainId),k=((R=(W=e.catalog)==null?void 0:W.clients.find(V=>V.id===e.execId))==null?void 0:R.snapshotSupported)??!1,D=h?`${P(E(h))} (${l})`:"Smaller",i=h?`${P(g(h))} (${w})`:"Much larger",p=h?` on ${t(h.Name)}`:"",S=h?e.checkpoint?h.SyncLabel:h.GenesisSyncLabel:"";return`
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
          ${h?`<p class="sync-estimate">⏱ Estimated initial sync${p}: <strong>${t(S)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${t((h==null?void 0:h.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${t((h==null?void 0:h.CheckpointURL)??"")}" value="${t(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${t(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${k?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${e.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${e.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${t(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${t(e.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${p}</th><td class="yes">${D}</td><td class="limited">${i}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${h?B(h):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${i}${h?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${D}${h?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${t(d)})</span>
            <input id="data-dir-input" type="text" placeholder="${t(d)}" value="${t(e.dataDir)}" />
          </label>
          ${X(h,e.dataDir||d)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${t(d)}/jwt.hex" value="${t(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${He})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${He}" value="${t(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${t(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${De})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${De}" value="${t(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${t(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Ue})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Ue}" value="${t(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${t(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${t(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${t(e.rpcBindAddrError)}</p>`:""}
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
    `}function ne(){const h=e.catalog.networks.find(J=>J.ChainID===e.chainId),k=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,D=e.jwtPath||`${k}/jwt.hex`,i=_e.map(J=>`<li>${t(J.title)}</li>`).join(""),p=L(e.execHTTPPort,He),S=L(e.beaconHTTPPort,De),j=L(e.execP2PPort,Ue),W=p||S||j?`<tr><th>Non-default ports</th><td>${[p?`exec HTTP ${p}`:null,S?`beacon HTTP ${S}`:null,j?`exec p2p ${j}`:null].filter(J=>J!==null).map(t).join(", ")}</td></tr>`:"",{addr:R}=fe(e.rpcBindAddr),V=R?`<tr><th>RPC bind address</th><td><code>${t(R)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${t(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${t((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${t(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${t(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${t(k)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${t(D)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${t(e.checkpointUrl||(h==null?void 0:h.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${W}
            ${V}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${i}</ol>
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function ae(){const h=e.catalog.networks.find(R=>R.ChainID===e.chainId),k=h==null?void 0:h.LearnURL,D=new Set(e.events.filter(R=>R.done).map(R=>R.stepId)),i=new Set(e.events.filter(R=>R.err).map(R=>R.stepId)),p=new Map;for(const R of e.events){if(!R.line)continue;const V=p.get(R.stepId)??[];V.push(R.line),p.set(R.stepId,V)}const S=_e.map(R=>{var Ne;const V=D.has(R.id),J=i.has(R.id),$e=J?O("failed","bad"):V?O("done","ok"):O("pending","neutral"),Re=(p.get(R.id)??[]).slice(-5),Le=(Ne=e.events.find(Ce=>Ce.stepId===R.id&&Ce.err))==null?void 0:Ne.err,qe=R.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${k?` <a href="${t(k)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${V?"step-done":""} ${J?"step-error":""}">
          <div class="step-head">${$e} <strong>${t(R.title)}</strong></div>
          ${qe}
          ${Re.length?`<pre class="step-log">${Re.map(Ce=>t(Ce)).join(`
`)}</pre>`:""}
          ${Le?`<p class="error small">${t(Le)}</p>`:""}
        </li>
      `}).join(""),j=e.events.some(R=>R.err),W=_e.every(R=>D.has(R.id))||e.events.some(R=>R.stepId==="handshake"&&R.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${S}</ol>
        ${W&&!j?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        ${j?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function se(d,h){switch(d){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,y();break;case"goto-network":e.step="network",y();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",y();break;case"goto-mode":e.step="mode",y(),oe();break;case"goto-review":if(ge(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){y();break}e.step="review",y();break;case"start-setup":A();break}}function ge(){const d=n.querySelectorAll('input[name="mode"]');for(const R of Array.from(d))R.checked&&(e.archive=R.value==="archive");const h=n.querySelector("#data-dir-input"),k=n.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),k&&(e.jwtPath=k.value.trim());const D=n.querySelector("#exec-http-port-input"),i=n.querySelector("#beacon-http-port-input"),p=n.querySelector("#exec-p2p-port-input");D&&(e.execHTTPPort=D.value.trim()),i&&(e.beaconHTTPPort=i.value.trim()),p&&(e.execP2PPort=p.value.trim());const S=n.querySelector("#rpc-bind-addr-input");S&&(e.rpcBindAddr=S.value.trim());const j=n.querySelector("#checkpoint-url-input");j&&(e.checkpointUrl=j.value.trim());const W=n.querySelector("#snapshot-key-input");W&&(e.snapshotKey=W.value.trim()),e.execHTTPPortError=I(e.execHTTPPort).error??null,e.beaconHTTPPortError=I(e.beaconHTTPPort).error??null,e.execP2PPortError=I(e.execP2PPort).error??null,e.rpcBindAddrError=fe(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?ue(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function fe(d){if(!d)return{};const h=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(d);return h?h.slice(1).every(k=>Number(k)<=255)?{addr:d}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(d)&&d.includes(":")?{addr:d}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const b=/^\d+$/;function I(d){if(!d)return{};if(!b.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const h=Number(d);return!Number.isInteger(h)||h<1||h>65535?{error:"Port must be between 1 and 65535."}:{port:h}}function L(d,h){const{port:k}=I(d);if(!(k===void 0||k===h))return k}async function A(){var p;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(p=e.streamStop)==null||p.call(e),e.streamStop=null,y();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const h=L(e.execHTTPPort,He),k=L(e.beaconHTTPPort,De),D=L(e.execP2PPort,Ue);h!==void 0&&(d.ExecHTTPPort=h),k!==void 0&&(d.BeaconHTTPPort=k),D!==void 0&&(d.ExecP2PPort=D);const{addr:i}=fe(e.rpcBindAddr);i!==void 0&&(d.RPCBindAddr=i),e.checkpoint?e.checkpointUrl&&(d.CheckpointURL=e.checkpointUrl):d.NoCheckpoint=!0,e.execSnapshot&&(d.ExecSnapshot=!0,d.SnapshotKey=e.snapshotKey);try{await dt(e.targetId,d)}catch(S){if(!(S instanceof ve&&S.status===409)){e.starting=!1,e.startError=String(S instanceof Error?S.message:S),y();return}}e.starting=!1,e.step="run",y(),e.streamStop=Ke(e.targetId,S=>{o||(e.events.push(S),e.step==="run"&&y())})}function z(d){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],D=h.map(i=>i.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${h.map((i,p)=>`<li class="${p===D?"current":p<D?"past":"future"}">${t(i.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;o=!0,(d=e.streamStop)==null||d.call(e)}}const ln=document.querySelector("#app"),{contentEl:dn,setActiveNav:un}=Ut(ln);let ce=null;function pn(){const c=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(c.length===0)return{screen:"targets"};const[o,e]=c;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function be(n){const c=document.createElement("div");return dn.replaceChildren(c),n(c)}function tt(){if(ce){try{ce()}catch{}ce=null}const{screen:n,id:c}=pn();switch(un(n),n){case"setup":if(!c){location.hash="#/targets";return}ce=be(o=>cn(o,c));break;case"dash":if(!c){location.hash="#/targets";return}ce=be(o=>jt(o,c));break;case"logs":if(!c){location.hash="#/targets";return}ce=be(o=>qt(o,c));break;case"security":if(!c){location.hash="#/targets";return}ce=be(o=>_t(o,c));break;case"diag":if(!c){location.hash="#/targets";return}ce=be(o=>Wt(o,c));break;case"services":if(!c){location.hash="#/targets";return}ce=be(o=>en(o,c));break;case"rpc":ce=be(o=>Yt(o));break;case"settings":ce=be(o=>zt(o));break;case"targets":default:ce=be(o=>nn(o));break}}window.addEventListener("hashchange",tt);tt();
