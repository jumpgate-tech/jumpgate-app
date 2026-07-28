var It=Object.defineProperty;var Rt=(s,i,o)=>i in s?It(s,i,{enumerable:!0,configurable:!0,writable:!0,value:o}):s[i]=o;var He=(s,i,o)=>Rt(s,typeof i!="symbol"?i+"":i,o);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const f of document.querySelectorAll('link[rel="modulepreload"]'))e(f);new MutationObserver(f=>{for(const y of f)if(y.type==="childList")for(const x of y.addedNodes)x.tagName==="LINK"&&x.rel==="modulepreload"&&e(x)}).observe(document,{childList:!0,subtree:!0});function o(f){const y={};return f.integrity&&(y.integrity=f.integrity),f.referrerPolicy&&(y.referrerPolicy=f.referrerPolicy),f.crossOrigin==="use-credentials"?y.credentials="include":f.crossOrigin==="anonymous"?y.credentials="omit":y.credentials="same-origin",y}function e(f){if(f.ep)return;f.ep=!0;const y=o(f);fetch(f.href,y)}})();function Lt(){return W("/api/host")}function Pe(){return W("/api/catalog")}function Ee(){return W("/api/targets")}function Xe(s){return W("/api/targets",{method:"POST",headers:ie,body:JSON.stringify(s)})}function Nt(s){return W(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Bt(s,i){return W(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function At(s,i){return W(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:ie,body:JSON.stringify(i)})}function Ke(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Ht(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Dt(s,i=200){return W(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function Ut(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Qe(s,i){const o=i===void 0?{}:{lines:i};return W(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:ie,body:JSON.stringify(o)})}function Mt(s,i,o){return W(`/api/targets/${encodeURIComponent(s)}/services/${i}/${o}`,{method:"POST"})}function Ot(s,i){return W(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:ie,body:JSON.stringify({Confirm:i})})}function Ft(s){return W(`/api/targets/${encodeURIComponent(s)}/du`)}function qt(s){return W(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function jt(s){return W(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Wt(s){return W(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function _t(s){return W(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function Kt(s){return W(`/api/targets/${encodeURIComponent(s)}/containers`)}function zt(s,i,o){return W(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${o}`,{method:"POST"})}async function Jt(s,i){const o=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:ie,body:JSON.stringify({Confirm:i})}),e=await o.text();let f=null;try{f=e?JSON.parse(e):null}catch{}if(f&&typeof f=="object"&&"report"in f)return f;const y=f&&typeof f=="object"&&typeof f.error=="string"?f.error:o.statusText||`HTTP ${o.status}`;throw new we(o.status,y)}function Gt(s,i){return W(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function Vt(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:ie}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const f=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,f)}function Yt(s,i,o){return W(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:ie,body:JSON.stringify(o)})}function Zt(){return W("/api/gateways")}function Xt(s){return W("/api/gateways",{method:"POST",headers:ie,body:JSON.stringify(s)})}function Qt(s){return W(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function en(s){return W(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function tn(s,i=!1){const o=i?"?refresh=1":"";return W(`/api/gateways/${encodeURIComponent(s)}/capabilities${o}`)}function nn(s){return W(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function an(s,i){return W(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:ie,body:JSON.stringify(i)})}function sn(s,i){return W(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function rn(s){return W(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function on(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:ie,body:JSON.stringify({Confirm:s})}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const f=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,f)}function cn(s){return W(`/api/chainlist/${s}`)}function ln(){return W("/api/settings")}function dn(s){return W("/api/settings",{method:"PUT",headers:ie,body:JSON.stringify(s)})}class we extends Error{constructor(o,e,f,y){super(e);He(this,"status");He(this,"hint");He(this,"code");this.name="ApiError",this.status=o,this.hint=f,this.code=y}}const ie={"Content-Type":"application/json"};async function W(s,i){const o=await fetch(s,i);if(!o.ok){let f=o.statusText||`HTTP ${o.status}`,y,x;try{const v=await o.json();v&&typeof v.error=="string"&&v.error&&(f=v.error),v&&typeof v.hint=="string"&&v.hint&&(y=v.hint),v&&typeof v.code=="string"&&v.code&&(x=v.code)}catch{}throw new we(o.status,f,y,x)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const et="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ne(s,i){const o=s&&i&&i!==et?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(et)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function un(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),o=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:f=>{for(const y of o)y.classList.toggle("active",y.dataset.nav===f)}}}function Te(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function pn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function hn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),o=Math.floor(i/60),e=i%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const f=Math.floor(o/24),y=o%24;return`~${f}d ${y}h`}function H(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function $e(s){return`<span class="dot dot-${s}"></span>`}const tt=["B","KB","MB","GB","TB","PB"];function Ce(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,o=0;for(;i>=1024&&o<tt.length-1;)i/=1024,o++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${tt[o]}`}async function Ne(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ke(s,i){s.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!s.contains(e))return;const f=e.dataset.action;f&&i(f,e,o)})}function _e(s,i,o){const e=i.find(y=>y.value===o),f=i.map(y=>`
      <li class="dropdown-option${y.value===o?" selected":""}" role="option"
          aria-selected="${y.value===o}" data-value="${n(y.value)}">
        ${n(y.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${f}</ul>
    </div>
  `}function Re(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var o;i.classList.remove("open"),(o=i.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function ze(s,i){s.addEventListener("click",f=>{const y=f.target,x=y.closest(".dropdown-trigger");if(x&&s.contains(x)){const E=x.closest(".dropdown"),q=!!E&&!E.classList.contains("open");Re(s),E&&q&&(E.classList.add("open"),x.setAttribute("aria-expanded","true"));return}const v=y.closest(".dropdown-option");if(v&&s.contains(v)){const E=v.closest(".dropdown");Re(s),i((E==null?void 0:E.dataset.dropdown)??"",v.dataset.value??"");return}Re(s)});const o=f=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const y=f.target;(!y.closest(".dropdown")||!s.contains(y))&&Re(s)},e=f=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}f.key==="Escape"&&Re(s)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const qe="app-modal";let Oe=null;function te(s,i){z();const o=document.createElement("div");o.className="modal-overlay",o.id=qe,o.innerHTML=`<div class="modal">${s}</div>`,o.addEventListener("click",f=>{const y=f.target.closest("[data-modal-action]");y!=null&&y.dataset.modalAction?i(y.dataset.modalAction):f.target===o&&i("cancel")});const e=f=>{f.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),Oe=e,document.body.appendChild(o)}function z(){var s;(s=document.getElementById(qe))==null||s.remove(),Oe&&(document.removeEventListener("keydown",Oe),Oe=null)}function Fe(){return document.querySelector(`#${qe} .modal`)}function Le(s){return new Promise(i=>{var f;let o=!1;const e=y=>{o||(o=!0,z(),i(y))};te(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,y=>e(y==="confirm")),(f=document.querySelector(`#${qe} [data-modal-action="confirm"]`))==null||f.focus()})}const fn=85,je={exec:"Execution",beacon:"Beacon"};function mn(s,i){let o=!1,e=null,f=null,y=null,x=null,v=null,E=null,q=null,U=null;const j={exec:null,beacon:null};let N=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ne()}</div>`;const O=s.querySelector("#dash-body"),C=s.querySelector("#dash-footer");O.addEventListener("click",g=>{const L=g.target.closest("[data-action]");if(!L||!O.contains(L))return;const A=L.dataset.action;if(A==="svc-action"){const D=L.dataset.svc,G=L.dataset.kind;D&&G&&be(D,G)}else if(A==="open-clear"){const D=L.dataset.svc;D&&ye(D)}else if(A==="copy"){const D=L.dataset.copy;D&&ge(L,D)}else A==="retry-du"?l():A==="retry-endpoints"&&$()}),k();async function k(){let g,L;try{const[D,G]=await Promise.all([Ee(),Pe()]);g=D.find(u=>u.id===i),L=G}catch(D){if(o)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(D))}</p>`;return}if(o)return;if(!g){O.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!g.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const A=L==null?void 0:L.networks.find(D=>D.ChainID===g.wire.ChainID);A&&(C.innerHTML=ne(A.Name,A.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',e=Ht(i,D=>{o||(P(D),f=D,y=D,I())}),l(),$()}async function l(){E=null;try{v=await Ft(i)}catch(g){v=null,E=String(g instanceof Error?g.message:g)}o||I()}async function $(){U=null;try{q=await qt(i)}catch(g){q=null,U=String(g instanceof Error?g.message:g)}o||I()}function P(g){if(!f)return;const L=(new Date(g.at).getTime()-new Date(f.at).getTime())/1e3,A=g.execHead-f.execHead;if(L>0&&A>=0){const D=A/L;x=x===null?D:x*.7+D*.3}}function I(){if(!y)return;const g=y;O.innerHTML=`
      <p class="dash-status">${Y(g)}</p>
      <div class="card-grid">
        ${ae(g)}
        ${Z(g)}
        ${_(g)}
        ${ce(g)}
        ${le(g)}
        ${pe()}
      </div>
      <p class="muted small">Last updated ${n(new Date(g.at).toLocaleTimeString())}</p>
    `}function Y(g){return!g.execActive&&!g.beaconActive?H("Node not running","bad"):g.execSyncing||g.beaconDistance>0?H("Syncing","warn"):H("Running · synced","ok")}function Q(g){const A=g.refHead>0?g.refHead-g.execHead:null,D=A!==null&&A>0&&x&&x>0?hn(A/x):A!==null&&A<=0?"caught up":"—";return{lag:A,eta:D}}function Z(g){const{lag:L,eta:A}=Q(g);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${g.execActive?g.execSyncing?H("syncing","warn"):g.execHead===0?H("no data","neutral"):H("synced","ok"):H("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${Te(g.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${L!==null?Te(g.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${L!==null?Te(Math.max(L,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${A}</dd></div>
        </dl>
      </div>
    `}function _(g){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${g.beaconActive?g.beaconSlot===0?H("no data","neutral"):g.beaconDistance===0?H("synced","ok"):H("syncing","warn"):H("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${Te(g.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${Te(g.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ce(g){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${Te(g.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${Te(g.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function le(g){const L=g.diskUsedPct>=fn,A=`
      <div class="meter"><div class="meter-fill ${L?"meter-warn":""}" style="width:${Math.min(g.diskUsedPct,100)}%"></div></div>
      <p>${pn(g.diskUsedPct)} used</p>
    `;if(E)return`
        <div class="card ${L?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="error small">${n(E)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!v)return`
        <div class="card ${L?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="muted">Loading…</p>
        </div>
      `;const D=v.ExpectedExecBytes>0?Math.min(v.ExecBytes/v.ExpectedExecBytes*100,100):0,G=v.ExpectedBeaconBytes>0?Math.min(v.BeaconBytes/v.ExpectedBeaconBytes*100,100):0,{lag:u,eta:b}=Q(g),T=u!==null&&u>0&&x!==null&&x>0;return`
      <div class="card ${L?"card-warn":""}">
        <h3>Storage</h3>
        ${A}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Ce(v.ExecBytes)} of ~${Ce(v.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${D}%"></div></div>
        ${T?`<p class="muted small">Estimated time remaining: ${n(b)}</p>`:""}
        <p class="muted small">Beacon — ${Ce(v.BeaconBytes)} of ~${Ce(v.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${G}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Ce(v.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(v.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(v.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function pe(){if(U)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(U)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!q)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const g=q,L=g.ExecReachable&&!g.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",A=g.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(g.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(g.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${$e(g.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(g.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(g.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${$e(g.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(g.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(g.BeaconHTTP)}">Copy</button>
        </div>
        ${L}
        ${A}
      </div>
    `}function re(g,L){const A=je[g],D=j[g],G=(u,b,T)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${g}" data-kind="${u}" ${D!==null||T?"disabled":""}>${D===u?se():n(b)}</button>`;return`
      <div class="service-row">
        <span>${n(A)} ${L?H("active","ok"):H("down","bad")}</span>
        <div class="service-actions">
          ${G("start","Start",L)}
          ${G("stop","Stop",!L)}
          ${G("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${g}" ${D!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function ae(g){return`
      <div class="card">
        <h3>Services</h3>
        ${re("exec",g.execActive)}
        ${re("beacon",g.beaconActive)}
        ${N?`<p class="error small">${n(N)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function se(){return'<span class="spinner" aria-label="working"></span>'}async function be(g,L){if(j[g]===null){j[g]=L,N=null,I();try{await Mt(i,g,L)}catch(A){N=`${je[g]} ${L} failed: ${A instanceof Error?A.message:String(A)}`}j[g]=null,o||I()}}async function ge(g,L){const A=await Ne(L),D=g.textContent;g.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{o||(g.textContent=D)},1500)}function ye(g){const L=je[g],A=v?Ce(g==="exec"?v.ExecBytes:v.BeaconBytes):"unknown (disk usage hasn't loaded)";te(`
        <h2>Clear ${n(L)} data</h2>
        <p class="error">
          This stops the ${n(L.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(A)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(g)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,u=>{if(u==="cancel"){z();return}u==="confirm"&&he(g)});const D=document.getElementById("clear-confirm-input"),G=document.getElementById("clear-confirm-btn");D==null||D.addEventListener("input",()=>{G&&(G.disabled=D.value.trim()!==g)}),D==null||D.focus()}async function he(g){const L=document.getElementById("clear-confirm-btn");L&&(L.disabled=!0,L.textContent="Clearing…");try{await Ot(i,g),z(),l()}catch(A){const D=Fe();if(D){const G=document.createElement("p");G.className="error small",G.textContent=`Clear failed: ${A instanceof Error?A.message:String(A)}`,D.appendChild(G)}L&&(L.disabled=!1,L.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),z()}}const nt=500,at="valve-node-app.explain-consent";function bn(s,i){let o=!1,e=null;const f=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ne()}</div>
  `;const y=s.querySelector("#logs-body"),x=s.querySelector("#logs-footer");ke(s,k=>{k==="explain"&&U()}),v();async function v(){let k,l;try{const[P,I]=await Promise.all([Ee(),Pe()]);k=P.find(Y=>Y.id===i),l=I}catch(P){if(o)return;y.innerHTML=`<p class="error">Failed to load target: ${n(String(P))}</p>`;return}if(o)return;if(!k){y.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!k.wire){y.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const $=l==null?void 0:l.networks.find(P=>P.ChainID===k.wire.ChainID);$&&(x.innerHTML=ne($.Name,$.LearnURL));try{const P=await Dt(i,200);if(o)return;f.push(...P)}catch(P){if(o)return;y.innerHTML=`<p class="error">Failed to load logs: ${n(String(P))}</p>`;return}E(),e=Ut(i,P=>{o||(f.push(P),f.length>nt&&f.splice(0,f.length-nt),E())})}function E(){const k=f.filter($=>$.severity==="error"||$.severity==="critical");y.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${f.map(q).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${H(String(k.length),k.length?"bad":"neutral")}</h2>
          <div class="log-lines">${k.length?k.slice().reverse().map(q).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const l=y.querySelector(".log-lines");l&&(l.scrollTop=l.scrollHeight)}function q(k){const l=k.severity||"info",$=k.learnUrl?` <a href="${n(k.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(l)}">
        <span class="log-time">${n(new Date(k.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(k.unit)}</span>
        <span class="log-sev">${n(l)}</span>
        <span class="log-text">${n(k.line)}</span>
        ${k.explain?`<div class="log-explain">${n(k.explain)}${$}</div>`:""}
      </div>
    `}async function U(){const k=f.filter($=>$.severity==="error"||$.severity==="critical").map($=>$.line).slice(-40);if(!(localStorage.getItem(at)==="1")){j(k);return}await N(k)}function j(k){const l=k.length?`<pre class="explain-excerpt">${k.map($=>n($)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';O(`
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
    `,$=>{$==="proceed"?(localStorage.setItem(at,"1"),C(),N(k)):C()})}async function N(k){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const l=k.length?await Qe(i,k):await Qe(i);if(o)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(l.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${l.sentExcerpt.map($=>n($)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&C()})}catch(l){if(o)return;if(l instanceof we&&l.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,$=>{$==="close"&&C()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(l instanceof Error?l.message:String(l))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&C()})}}function O(k,l){C();const $=document.createElement("div");$.className="modal-overlay",$.id="explain-modal",$.innerHTML=`<div class="modal">${k}</div>`,$.addEventListener("click",P=>{const I=P.target.closest("[data-modal-action]");I!=null&&I.dataset.modalAction&&l(I.dataset.modalAction),P.target===$&&l("cancel")}),document.body.appendChild($)}function C(){var k;(k=document.getElementById("explain-modal"))==null||k.remove()}return()=>{o=!0,e==null||e(),C()}}function gn(s,i){let o=!1,e=null,f=null,y=!1,x=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ne()}</div>`;const v=s.querySelector("#diag-body"),E=s.querySelector("#diag-footer");ke(s,(l,$)=>{var P;if(l==="run")U();else if(l==="toggle")(P=$.closest(".check-item"))==null||P.classList.toggle("expanded");else if(l==="copy"){const I=$.dataset.copy;I&&k($,I)}}),q();async function q(){let l,$;try{const[I,Y]=await Promise.all([Ee(),Pe()]);l=I.find(Q=>Q.id===i),$=Y}catch(I){if(o)return;v.innerHTML=`<p class="error">Failed to load target: ${n(String(I))}</p>`;return}if(o)return;if(!l){v.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const P=$==null?void 0:$.networks.find(I=>I.ChainID===l.wire.ChainID);P&&(E.innerHTML=ne(P.Name,P.LearnURL));try{e=await _t(i),x=!0}catch(I){f=String(I instanceof Error?I.message:I)}o||j()}async function U(){y=!0,f=null,j();try{e=await Wt(i),x=!0}catch(l){f=String(l instanceof Error?l.message:l)}y=!1,o||j()}function j(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${y?"disabled":""}>${y?"Running…":"Run diagnostics"}</button>
      </div>
      ${f?`<p class="error">${n(f)}</p>`:""}
      ${N()}
    `}function N(){if(!x&&!f)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const l=new Date(e.at).toLocaleString(),$=e.failedId?`<p><strong>Failed at: ${n(O(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(l)} — trigger: ${n(e.trigger)}</p>
      ${$}
      <ul class="check-list">${e.items.map(C).join("")}</ul>
    `}function O(l){var $;return(($=e==null?void 0:e.items.find(P=>P.ID===l))==null?void 0:$.Title)??l}function C(l){const $=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral",P=l.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${P?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${H(P?"failed here":l.Status,$)}
          <strong>${n(l.Title)}</strong>
          <span class="muted small check-detail-inline">${n(l.Detail)}</span>
        </button>
        <div class="check-body">
          <details${P?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(l.Why)}</p>
          </details>
          ${l.Fix?`
                <details${P?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(l.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function k(l,$){const P=await Ne($),I=l.textContent;l.textContent=P?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=I)},1500)}return()=>{o=!0}}function yn(s,i){let o=!1,e=[],f=null,y=!1,x=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ne()}</div>`;const v=s.querySelector("#sec-body"),E=s.querySelector("#sec-footer");ke(s,(C,k)=>{var l;if(C==="rerun")U();else if(C==="toggle")(l=k.closest(".check-item"))==null||l.classList.toggle("expanded");else if(C==="copy"){const $=k.dataset.copy;$&&O(k,$)}}),q();async function q(){let C,k;try{const[$,P]=await Promise.all([Ee(),Pe()]);C=$.find(I=>I.id===i),k=P}catch($){if(o)return;v.innerHTML=`<p class="error">Failed to load target: ${n(String($))}</p>`;return}if(o)return;if(!C){v.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const l=k==null?void 0:k.networks.find($=>$.ChainID===C.wire.ChainID);l&&(E.innerHTML=ne(l.Name,l.LearnURL)),await U()}async function U(){y=!0,f=null,j();try{e=await jt(i),x=!0}catch(C){f=String(C instanceof Error?C.message:C)}y=!1,o||j()}function j(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${y?"disabled":""}>${y?"Re-running…":"Re-run checks"}</button>
      </div>
      ${f?`<p class="error">${n(f)}</p>`:""}
      ${!x&&y?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(N).join("")}</ul>`:x?'<p class="muted">No checks returned.</p>':""}
    `}function N(C){const k=C.Status==="pass"?"ok":C.Status==="fail"?"bad":C.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${H(C.Status,k)}
          <strong>${n(C.Title)}</strong>
          <span class="muted small check-detail-inline">${n(C.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${n(C.Why)}</p>
          </details>
          ${C.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(C.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(C.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function O(C,k){const l=await Ne(k),$=C.textContent;C.textContent=l?"Copied!":"Copy failed",setTimeout(()=>{o||(C.textContent=$)},1500)}return()=>{o=!0}}const vn=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function $n(s){let i=!1,o=!1,e=!1,f=null,y=!1,x=null,v=null;s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ne()}`;const E=s.querySelector("#settings-body");ke(s,N=>{if(N==="save"&&j(),N==="clear-key"){if(!x)return;o=!0;const O=s.querySelector("#ai-key");O&&(O.value=""),U(x)}}),ze(s,(N,O)=>{N!=="ai-provider"||!x||(v=O,y=!1,U(x))}),q();async function q(){try{const N=await ln();if(i)return;x=N,U(N)}catch(N){if(i)return;E.innerHTML=`<p class="error">Failed to load settings: ${n(String(N))}</p>`}}function U(N){var k;const O=v??N.aiProvider;E.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${_e("ai-provider",vn.map(l=>({value:l.value,label:l.label})),O)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${N.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${N.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${n(N.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${f?`<p class="error">${n(f)}</p>`:""}
        ${y?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const C=s.querySelector("#ai-key");C==null||C.addEventListener("input",()=>{o=!0,y=!1}),(k=s.querySelector("#ref-rpc-base"))==null||k.addEventListener("input",()=>{y=!1})}async function j(){const N=s.querySelector("#ai-key"),O=s.querySelector("#ref-rpc-base");if(!N||!O||!x)return;const C={aiProvider:v??x.aiProvider,refRpcBase:O.value.trim()};o&&(C.aiKey=N.value),e=!0,f=null,y=!1,U(x);try{const k=await dn(C);if(i)return;x=k,o=!1,e=!1,y=!0,U(k)}catch(k){if(i)return;e=!1,f=String(k instanceof Error?k.message:k),U(x)}}return()=>{i=!0}}const wn=["http","ws","archive","trace"],kn={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Tn="run",Cn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Sn(s){let i=!1,o=null,e=null;const f={},y={},x={},v={},E={},q={},U={},j={},N={},O={};let C=null;s.innerHTML=`
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
    ${ne()}
  `;const k=s.querySelector("#rpc-body");ke(s,(t,a)=>{xe(t,a)}),ze(s,()=>{}),l();async function l(){try{const t=await Zt();if(i)return;o=t,e=null}catch(t){if(i)return;o=null,e=ue(t)}_();for(const t of(o==null?void 0:o.gateways)??[])$(t.id),P(t.id,!1)}async function $(t){try{const a=await en(t);if(i)return;f[t]=a}catch{if(i)return;f[t]=null}_()}async function P(t,a){x[t]=a,a&&_();try{const r=await tn(t,a);if(i)return;y[t]=r}catch{if(i)return;y[t]=null}x[t]=!1,_()}function I(t){return((o==null?void 0:o.gateways)??[]).find(a=>a.id===t)}function Y(t,a){return(t.networks??[]).find(r=>r.chainId===a)}function Q(t,a,r){var p;const d=(((p=f[t])==null?void 0:p.networks)??[]).find(m=>m.chainId===a);return((d==null?void 0:d.upstreams)??[]).find(m=>m.upstream===r)}function Z(t,a,r){var d;return(((d=y[t])==null?void 0:d.endpoints)??[]).find(p=>p.chainId===a&&p.upstream===r)}function _(){if(i)return;if(e){k.innerHTML=`<p class="error">Could not read the gateways: ${n(e)}</p>`;return}if(!o){k.innerHTML='<p class="muted">Loading…</p>';return}const t=o.gateways??[];k.innerHTML=`
      ${t.map(le).join("")}
      ${t.length===0?ce():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${t.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function ce(){return((o==null?void 0:o.targets)??[]).length===0?`
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
    `}function le(t){return`
      <section class="rpc-gateway">
        ${pe(t)}
        ${t.error?se(t):""}
        ${t.blocked?`<div class="banner banner-warn">${n(t.blocked)}</div>`:""}
        ${(t.warnings??[]).map(a=>`<div class="banner banner-warn">${n(a)}</div>`).join("")}
        ${Ae(t)}
        ${E[t.id]?`<p class="error small">${n(E[t.id])}</p>`:""}
        ${ge(t)}
        ${U[t.id]?F(t):""}
        ${ye(t)}
      </section>
    `}function pe(t){var r;const a=t.status.State==="running";return`
      <div class="rpc-bar${a?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${ae(t)}
            <strong>${n(t.label)}</strong>
            ${re(t)}
            <span class="muted small">on ${n(t.placement.targetId)} · ${n(t.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(t.actions??[]).map(d=>be(t,d)).join("")}
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(t.id)}">
              ${U[t.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(t.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${a?`<code class="endpoint-url">${n(t.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${n(t.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((r=(t.networks??[])[0])==null?void 0:r.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${n(t.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function re(t){switch(t.status.State){case"running":return H("running","ok");case"created-but-stopped":return H("stopped","warn");case"not-created":return H("not created","neutral");default:return H("unknown","bad")}}function ae(t){return t.status.State==="running"?$e("ok"):t.status.State==="unknown"?$e("bad"):$e("neutral")}function se(t){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${n(t.error??"")}</div>
        ${t.hint?`<div class="small">${n(t.hint)}</div>`:""}
      </div>
    `}function be(t,a){const r=Cn[a];if(!r)return"";const d=v[t.id];return`
      <button class="${r.className}" data-action="gw-${a}" data-gid="${n(t.id)}"
              title="${n(r.title)}" ${d?"disabled":""}>
        ${d===a?'<span class="spinner" aria-label="working"></span>':n(r.label)}
      </button>
    `}function ge(t){const a=q[t.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(t.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function ye(t){const a=t.networks??[];return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(t.id)}">Add a network</button>
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        ${he(t)}
        <div class="surface-scroll">
          <table class="surface">
            <thead>
              <tr>
                <th class="col-endpoint">Endpoint</th>
                <th>Role</th>
                <th>State</th>
                <th>Capabilities</th>
                <th class="col-share">Share of traffic</th>
                <th class="col-act"></th>
              </tr>
            </thead>
            <tbody>
              ${a.map(r=>g(t,r)+A(t,r)).join("")}
            </tbody>
          </table>
        </div>
        ${h(t)}
      </div>
    `}function he(t){const a=y[t.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(S(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${x[t.id]?"disabled":""}>
          ${x[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(t.id)}">+ Network</button>
      </div>
    `}function g(t,a){return`
      <tr class="band${!a.serviceable?" band-bad":""}">
        <td colspan="6">
          <div class="band-inner">
            <span class="band-id">${a.chainId}</span>
            <span class="band-name">${n(a.name)}</span>
            <code class="band-path">${n(a.path)}</code>
            ${a.url?`<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                           title="Copy ${n(a.url)}">Copy URL</button>`:""}
            <span class="band-right">
              ${L(t,a)}
              <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                      data-gid="${n(t.id)}" data-chain="${a.chainId}">+ Endpoint</button>
              <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                      data-gid="${n(t.id)}" data-chain="${a.chainId}">Remove</button>
            </span>
          </div>
          ${(a.warnings??[]).map(d=>`<div class="band-warn">${n(d)}</div>`).join("")}
        </td>
      </tr>
    `}function L(t,a){if(!a.serviceable)return H("no usable endpoint","bad");const r=a.upstreams??[],d=r.map(m=>Z(t.id,a.chainId,m.id)).filter(m=>!!m&&!m.unprobeable);return d.length>0&&d.every(m=>u(m,"ws")==="unsupported")?H("subscriptions unavailable","bad"):r.map(m=>Q(t.id,a.chainId,m.id)).some((m,R)=>{var w;return m&&m.diverged&&(((w=r[R])==null?void 0:w.local)??!1)})?H("your endpoint is under-used","warn"):H(`${r.length} endpoint${r.length===1?"":"s"}`,"ok")}function A(t,a){const r=a.upstreams??[];return r.length===0?`
        <tr class="ep"><td colspan="6" class="muted small">
          No endpoint yet, so there is nowhere for calls on this path to go.
        </td></tr>
      `:r.map(d=>D(t,a,d)).join("")}function D(t,a,r){const d=`${t.id}|${a.chainId}|${r.id}`,p=r.actions??[];return`
      <tr class="ep${r.problem?" ep-bad":""}">
        <td class="col-endpoint">
          <div class="ep-what">
            ${r.problem?$e("bad"):$e("ok")}
            <span class="ep-label">${n(r.label)}</span>
          </div>
          <code class="ep-url">${n(r.endpoint||"—")}</code>
          ${r.problem?`<div class="error small">${n(r.problem)}</div>`:""}
        </td>
        <td>${r.local?"Yours":"Public"}</td>
        <td>${G(r)}</td>
        <td>${b(t,a,r)}</td>
        <td class="col-share">${c(t,a,r)}</td>
        <td class="col-act">
          ${p.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(d)}"
                         data-target="${n(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${v[t.id]?"disabled":""}>
                   ${v[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(d)}">Remove</button>
        </td>
      </tr>
    `}function G(t){return t.problem?H("unusable","bad"):t.recentOnly?H("recent blocks","warn"):t.local?H("serving","ok"):H("fallback","neutral")}function u(t,a){var r;if(t)return a==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(r=(t.capabilities??[]).find(d=>d.key===a))==null?void 0:r.status}function b(t,a,r){const d=Z(t.id,a.chainId,r.id);return d?d.unprobeable?`<span class="caps-none" title="${n(d.unprobeable)}">not probeable from here</span>`:`<span class="caps">${wn.map(p=>T(t,a,d,p)).join("")}</span>`:`<span class="muted small">${y[t.id]===void 0?"probing…":"—"}</span>`}function T(t,a,r,d){const p=(r.capabilities??[]).find(ee=>ee.key===d),m=u(r,d)??"inconclusive",R=kn[d]??d.toUpperCase();let w="cap";m==="unsupported"?w=M(t,a,d)?"cap missing":"cap off":m==="inconclusive"?w="cap unknown":m==="inconsistent"&&(w="cap mixed");const J=p!=null&&p.detail?`${p.label}: ${p.detail}`:d==="http"&&r.reachDetail?`Answers JSON-RPC over HTTP: ${r.reachDetail}`:`${R}: no verdict`;return`<span class="${w}" title="${n(J)}">${n(R)}</span>`}function M(t,a,r){const d=(a.upstreams??[]).map(p=>Z(t.id,a.chainId,p.id)).filter(p=>!!p&&!p.unprobeable);return d.length>0&&d.every(p=>u(p,r)==="unsupported")}function c(t,a,r){const d=f[t.id];if(d===void 0)return'<span class="muted small">reading…</span>';if(d===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!d.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const p=Q(t.id,a.chainId,r.id),m=(d.networks??[]).find(ve=>ve.chainId===a.chainId);if(!p||!m||m.attributed===0)return'<span class="muted small">no traffic yet</span>';const R=Math.round(p.actual*100),w=Math.round(p.intended*100),J=p.diverged?r.local?"warn":"":"ok",ee=`${p.succeeded.toLocaleString()} of ${m.attributed.toLocaleString()} answered requests · routing intends ${w}%`+(p.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ee)}">
        <span class="bar">
          <span class="fill${J?" "+J:""}" style="width:${R}%"></span>
          <span class="tick" style="left:${w}%"></span>
        </span>
        <span class="share-n${p.diverged?" warn":""}">${R}%</span>
        ${p.unconfigured?H("not in config","warn"):""}
      </span>
    `}function h(t){const a=f[t.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(S(a.since))})`:""}. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function S(t){const a=new Date(t);return Number.isNaN(a.getTime())?t:a.toLocaleString()}function F(t){const a=t.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${n(t.id)}-port" value="${a.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${n(t.id)}-bind" value="${n(a.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${n(a.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${K(t)}
        ${B(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(t.id)}">Save settings</button>
        </div>
      </div>
    `}function K(t){const a=!t.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${n(t.id)}-metrics" ${a?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function B(t){var R;const a=n(t.id),r=t.config.TLS??null,d=(r==null?void 0:r.Enabled)??!1,p=(r==null?void 0:r.CertSource)||"internal",m=((R=t.tls)==null?void 0:R.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${a}-tls" ${d?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${a}-tls-host" value="${n((r==null?void 0:r.Hostname)??m)}"
               placeholder="${n(m||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${m?`<p class="muted small">
               The default is <code>${n(m)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${a}-tls-port" value="${(r==null?void 0:r.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${a}-tls-source">
          <option value="internal" ${p==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${p==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${a}-tls-cert" value="${n((r==null?void 0:r.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${a}-tls-key" value="${n((r==null?void 0:r.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${X(t)}
    `}function X(t){var R,w;const a=n(t.id),r=((R=t.config.TLS)==null?void 0:R.Enabled)??!1,d=j[t.id]??((w=t.tls)==null?void 0:w.verification)??null,p=N[t.id]??!1,m=O[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${r&&!p?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${p?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${m?`<p class="error small">${n(m)}</p>`:""}
      ${d?V(d):""}
    `}function V(t){const a=(t.assertions??[]).map(r=>`
          <li class="small">
            ${Se(r.status)}
            <strong>${n(r.title)}</strong>
            <div class="muted">${n(r.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${t.ok?t.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${n(t.summary)}
      </div>
      <ul class="verify-list">${a}</ul>
      <p class="muted small">
        Checked ${n(new Date(t.at).toLocaleString())} against <code>${n(t.address)}</code>
        ${t.notAfter?`· certificate valid until <code>${n(new Date(t.notAfter).toLocaleString())}</code> (${n(t.expiresIn??"")})`:""}
      </p>
      ${t.expiryWarning?`<div class="banner banner-warn">${n(t.expiryWarning)}</div>`:""}
    `}function Se(t){switch(t){case"pass":return H("pass","ok");case"fail":return H("fail","bad");case"unavailable":return H("unavailable","warn");default:return H("skipped","neutral")}}async function Be(t){N[t]=!0,O[t]=null,_();try{j[t]=await Qt(t)}catch(a){O[t]=`${ue(a)}${Ie(a)}`}finally{N[t]=!1,_()}}function Ae(t){var p,m;const a=t.tls;if(!(a!=null&&a.enabled))return"";const r=[];a.fallback&&r.push(`<div class="banner banner-warn">${n(a.fallback)}</div>`),a.error?r.push(`<div class="banner banner-warn">HTTPS front: ${n(a.error)}</div>`):((p=a.status)==null?void 0:p.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${n(a.containerName??"")}</code>) is
         ${n(((m=a.status)==null?void 0:m.State)??"unknown")}, so nothing is answering on
         <code>${n(a.url??"")}</code> even if the gateway itself is up.</div>`);const d=j[t.id]??a.verification??null;return d&&(!d.ok||!d.subscriptionsOk)&&r.push(`<div class="banner ${d.ok?"banner-warn":"banner-bad"}">${n(d.summary)}
         <div class="small">Checked ${n(new Date(d.at).toLocaleString())} — open Settings for the full check.</div></div>`),d!=null&&d.expiryWarning&&r.push(`<div class="banner banner-warn">${n(d.expiryWarning)}</div>`),a.rootCaPath&&a.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${n(a.rootCaPath)}</code> (on ${n(t.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function fe(t){return{...t.config,Networks:(t.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(r=>({...r}))}))}}async function de(t,a,r){E[t]=null;try{await an(t,a)}catch(d){return E[t]=`${r?r+": ":""}${ue(d)}`,_(),!1}return await l(),!0}async function xe(t,a){const r=a.dataset.gid??"";switch(t){case"refresh":await l();return;case"copy":a.dataset.copy&&await Pt(a,a.dataset.copy);return;case"reprobe":await P(r,!0);return;case"toggle-settings":U[r]=!U[r],_();return;case"save-settings":await ot(r);return;case"verify-tls":await Be(r);return;case"gw-start":case"gw-stop":case"gw-restart":await lt(r,t.slice(3));return;case"gw-create":case"gw-recreate":await dt(r);return;case"gw-wipe":Tt(r);return;case"add-gateway":St();return;case"forget-gateway":await ut(r);return;case"add-chain":pt(r);return;case"remove-chain":await bt(r,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":Ve(r,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await gt(a.dataset.key??"");return;case"reset-devnet":await wt(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function ot(t){const a=I(t);if(!a)return;const r=fe(a),d=s.querySelector(`#gw-${CSS.escape(t)}-port`),p=s.querySelector(`#gw-${CSS.escape(t)}-bind`);if(d){const w=Number.parseInt(d.value.trim(),10);Number.isFinite(w)&&(r.Port=w)}p&&(r.BindAddr=p.value.trim());const m=s.querySelector(`#gw-${CSS.escape(t)}-metrics`);m&&(r.MetricsOff=!m.checked),r.TLS=it(t,a);const R=a.status.State==="running";await de(t,r,"Saving settings")&&(U[t]=!1,R&&(E[t]=null,ct(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),_())}function it(t,a){var m,R,w,J,ee,ve,Ze;const r=Et=>s.querySelector(`#gw-${CSS.escape(t)}-${Et}`),d=r("tls");if(!d)return a.config.TLS??null;const p=Number.parseInt(((m=r("tls-port"))==null?void 0:m.value.trim())??"",10);return{Enabled:d.checked,Hostname:((R=r("tls-host"))==null?void 0:R.value.trim())??"",CertSource:((w=r("tls-source"))==null?void 0:w.value)??"internal",CertFile:((J=r("tls-cert"))==null?void 0:J.value.trim())??"",KeyFile:((ee=r("tls-key"))==null?void 0:ee.value.trim())??"",HTTPSPort:Number.isFinite(p)?p:443,BindAddr:((ve=a.config.TLS)==null?void 0:ve.BindAddr)??"",ImageRef:((Ze=a.config.TLS)==null?void 0:Ze.ImageRef)??""}}function ct(t,a){q[t]=[a]}async function lt(t,a){if(!v[t]){v[t]=a,E[t]=null,_();try{await sn(t,a)}catch(r){E[t]=`${a} failed: ${ue(r)}${Ie(r)}`}v[t]=null,await l()}}async function dt(t){if(v[t])return;v[t]="create",E[t]=null,q[t]=["starting…"],_();let a;try{a=await rn(t)}catch(r){E[t]=`${ue(r)}${Ie(r)}`,q[t]=[],v[t]=null,_();return}C==null||C(),C=Ke(a.targetId,r=>{if(i)return;const d=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(q[t]=[...(q[t]??[]).filter(m=>m!=="starting…"),d],!!r.err||r.stepId===Tn&&!!r.done){C==null||C(),C=null,v[t]=null,r.err&&(E[t]="Provisioning failed — see the log below."),l();return}_()})}async function ut(t){const a=I(t);if(!(!a||!await Le({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await nn(t)}catch(d){E[t]=ue(d),_();return}await l()}}function pt(t){const a=I(t);if(!a)return;const r=new Set((a.networks??[]).map(w=>w.chainId)),d=(o==null?void 0:o.presets)??[],p=d.filter(w=>!r.has(w.chainId)),m=d.filter(w=>r.has(w.chainId)),R=((o==null?void 0:o.targets)??[]).some(w=>w.id===a.placement.targetId&&w.hasDevnet);te(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${p.map(w=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${w.chainId}">
                <span>${n(w.name)}</span>
                <span class="muted small">chain ${w.chainId}${w.devnet?R?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${m.length?`<p class="muted small">Already fronted: ${n(m.map(w=>w.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,w=>{if(w==="cancel"){z();return}if(w==="custom"){ht(t);return}if(w.startsWith("preset:")){const J=Number.parseInt(w.slice(7),10),ee=d.find(ve=>ve.chainId===J);z(),ee!=null&&ee.devnet?mt(t,J,R):Je(t,J)}})}function ht(t){var a;te(`
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
      `,r=>{if(r==="cancel"){z();return}if(r!=="add")return;const d=document.getElementById("custom-chain-id"),p=document.getElementById("custom-chain-err"),m=Number.parseInt((d==null?void 0:d.value.trim())??"",10);if(!Number.isFinite(m)||m<=0){p&&(p.className="error small"),p&&(p.textContent="A chain id is a positive whole number.");return}z(),Je(t,m)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function Je(t,a){const r=I(t);if(!r)return;const d=fe(r),p=d.Networks??[];p.some(m=>m.ChainID===a)||(p.push({ChainID:a,Upstreams:[]}),d.Networks=p,await ft(t,d)&&(_(),Ve(t,a)))}async function ft(t,a){var m;const r={...a,Networks:(a.Networks??[]).filter(R=>R.Upstreams.length>0)};if(!await de(t,r))return!1;const p=I(t);if(p)for(const R of a.Networks??[])R.Upstreams.length===0&&!(p.networks??[]).some(w=>w.chainId===R.ChainID)&&(p.config.Networks=[...p.config.Networks??[],{ChainID:R.ChainID,Upstreams:[]}],p.networks=[...p.networks??[],{chainId:R.ChainID,name:((m=((o==null?void 0:o.presets)??[]).find(w=>w.chainId===R.ChainID))==null?void 0:m.name)??`Chain ${R.ChainID}`,path:`/${p.config.ProjectID}/evm/${R.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function mt(t,a,r){const d=I(t);if(!d)return;if(!r){te(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${n(d.placement.targetId)}</code>, so adding chain ${a} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(d.placement.targetId)}" data-modal-action="go">Create a devnet on ${n(d.placement.targetId)}</a>
          </div>
        `,()=>z());return}const p=fe(d),m=p.Networks??[],R={ID:"devnet",Kind:"managed-devnet",TargetID:d.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},w=m.find(J=>J.ChainID===a);w?w.Upstreams.push(R):m.push({ChainID:a,Upstreams:[R]}),p.Networks=m,await de(t,p,"Adding the devnet")}async function bt(t,a){const r=I(t);if(!r||!Number.isFinite(a))return;const d=Y(r,a);if(!await Le({title:`Remove ${(d==null?void 0:d.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(d==null?void 0:d.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const m=fe(r);m.Networks=(m.Networks??[]).filter(R=>R.ChainID!==a),await de(t,m,"Removing the network")}function Ge(t){const a=t.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function gt(t){const a=Ge(t);if(!a)return;const r=I(a.gid);if(!r)return;const d=fe(r),p=(d.Networks??[]).find(w=>w.ChainID===a.chainId);if(!p)return;const m=p.Upstreams.findIndex((w,J)=>(w.ID||`${a.chainId}-${J}`)===a.upstreamId);m<0||!await Le({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(p.Upstreams.splice(m,1),await de(a.gid,d,"Removing the endpoint"))}function Ve(t,a){const r=I(t);if(!r||!Number.isFinite(a))return;const d=((o==null?void 0:o.sources)??[]).filter(w=>w.chainId===a),p=Y(r,a),m=new Set(((p==null?void 0:p.upstreams)??[]).filter(w=>w.kind!=="external").map(w=>`${w.kind}|${w.targetId??""}`)),R=d.filter(w=>!m.has(`${w.kind}|${w.targetId}`));te(`
        <h2>Add an endpoint for ${n((p==null?void 0:p.name)??`chain ${a}`)}</h2>
        ${R.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${R.map(w=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(w.kind)}:${n(w.targetId)}">
                       <span>${n(w.label)}</span>
                       <span class="muted small">${n(w.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${a}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn btn-ghost" data-modal-action="discover">Find public endpoints…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,w=>{if(w==="cancel"){z();return}if(w==="discover"){vt(t,a);return}if(w==="manual"){$t(t,a);return}if(w.startsWith("source:")){const[,J,ee]=w.split(":");z(),yt(t,a,J,ee)}})}async function yt(t,a,r,d){const p=I(t);if(!p)return;const m=fe(p),R=m.Networks??[],w={ID:`${r==="managed-devnet"?"devnet":"node"}-${d}`,Kind:r,TargetID:d,Endpoint:"",Local:!0,RecentOnly:!1},J=R.find(ee=>ee.ChainID===a);J?J.Upstreams.push(w):R.push({ChainID:a,Upstreams:[w]}),m.Networks=R,await de(t,m,"Adding the endpoint")}async function vt(t,a){te(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,m=>{m==="cancel"&&z()});let r;try{r=await cn(a)}catch(m){const R=Fe();if(R){const w=document.createElement("p");w.className="error small",w.textContent=`Could not discover endpoints: ${ue(m)}`,R.appendChild(w)}return}if(i)return;const d=(r.endpoints??[]).filter(m=>m.status==="live"||m.status==="unprobed"),p=(r.endpoints??[]).filter(m=>m.status==="rejected");te(`
        <h2>Public endpoints for chain ${a}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${n(r.fetchError)}</div>`:""}</div>`:""}
        ${d.length?`<p class="muted small">${d.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${d.map(m=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(m.url)}">
                       <span><code>${n(m.url)}</code></span>
                       <span class="muted small">${m.status==="live"?`answered in ${m.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${p.length?`<details class="rpc-rejected">
                 <summary class="muted small">${p.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${p.map(m=>`<li class="muted small"><code>${n(m.url)}</code> — ${n(m.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,m=>{if(m==="cancel"){z();return}m.startsWith("add:")&&(z(),Ye(t,a,decodeURIComponent(m.slice(4))))})}function $t(t,a){var r;te(`
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
      `,d=>{if(d==="cancel"){z();return}if(d!=="add")return;const p=document.getElementById("manual-endpoint"),m=document.getElementById("manual-recent"),R=document.getElementById("manual-err"),w=(p==null?void 0:p.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(w)){R&&(R.className="error small",R.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}z(),Ye(t,a,w,(m==null?void 0:m.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Ye(t,a,r,d=!1){const p=I(t);if(!p)return;const m=fe(p),R=m.Networks??[],w=R.find(ve=>ve.ChainID===a),J=((w==null?void 0:w.Upstreams.length)??0)+1,ee={ID:`public-${a}-${J}`,Kind:"external",Endpoint:r,Local:!1,RecentOnly:d};w?w.Upstreams.push(ee):R.push({ChainID:a,Upstreams:[ee]}),m.Networks=R,await de(t,m,"Adding the endpoint")}async function wt(t,a){const r=Ge(t);if(!r||!a||!await Le({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;v[r.gid]="reset",E[r.gid]=null,_();let p;try{p=await Vt(a)}catch(m){E[r.gid]=`Reset failed: ${ue(m)}${Ie(m)}`,v[r.gid]=null,_();return}v[r.gid]=null,kt(a,p),await l()}function kt(t,a){const r=[];r.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&r.push("A fresh chain was started from genesis.");const d=a.report.Cascaded??[],p=a.report.CascadeSkipped??[];te(`
        <h2>Devnet on ${n(t)} reset</h2>
        <ul class="plain-list">${r.map(m=>`<li>${n(m)}</li>`).join("")}</ul>
        ${d.length?`<p class="ok">Restarted in front of it: ${n(d.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${p.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(p.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>z())}function Tt(t){const a=I(t);if(!a)return;te(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(t)}</button>
        </div>
      `,p=>{if(p==="cancel"||p==="close"){z(),l();return}p==="confirm"&&Ct(t)});const r=document.getElementById("wipe-confirm-input"),d=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{d&&(d.disabled=r.value.trim()!==t)}),r==null||r.focus()}async function Ct(t){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let r;try{r=await on(t)}catch(d){const p=Fe();if(p){const m=document.createElement("p");m.className="error small",m.textContent=`Wipe failed: ${ue(d)}${Ie(d)}`,p.appendChild(m)}a&&(a.disabled=!1,a.textContent=`Wipe ${t}`);return}te(`
        <h2>${n(t)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${n(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{z(),l()})}function St(){var d;const t=(o==null?void 0:o.targets)??[],a=new Set(((o==null?void 0:o.gateways)??[]).map(p=>p.id));if(t.length===0){te(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>z());return}const r=a.has("default")?"":"default";te(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(r)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${t.map(p=>`<option value="${n(p.id)}">${n(p.id)} (${n(p.mode)})</option>`).join("")}
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
      `,p=>{if(p==="cancel"){z();return}p==="create"&&xt()}),(d=document.getElementById("new-gw-id"))==null||d.focus()}async function xt(){const t=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),d=document.getElementById("new-gw-err"),p=(t==null?void 0:t.value.trim())??"",m=(a==null?void 0:a.value)??"",R=Number.parseInt((r==null?void 0:r.value.trim())??"",10),w=J=>{d&&(d.className="error small",d.textContent=J)};if(!p){w("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!m){w("Pick the machine it runs on.");return}try{await Xt({id:p,placement:{targetId:m,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(R)?R:4e3,Networks:[]}})}catch(J){w(ue(J));return}z(),await l()}async function Pt(t,a){const r=await Ne(a),d=t.textContent;t.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{i||(t.textContent=d)},1500)}function ue(t){return t instanceof Error?t.message:String(t)}function Ie(t){return t instanceof we&&t.hint?` — ${t.hint}`:""}return()=>{i=!0,C==null||C(),z()}}const xn="run",Pn={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},En={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function In(s,i){let o=!1,e=null,f=null;const y={devnet:null},x={devnet:null},v={devnet:[]};let E=null;const q={devnet:!1};let U=null;const j={devnet:null},N={devnet:null};s.innerHTML=`
    <div class="page-head">
      <h1>Services: ${n(i)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${ne()}
  `;const O=s.querySelector("#services-body");ke(s,(c,h)=>{ye(c,h)}),C();async function C(){try{const c=await Kt(i);if(o)return;e=c,f=null}catch(c){if(o)return;e=null,f=T(c)}l()}function k(c){return e==null?void 0:e.services.find(h=>h.id===c)}function l(){if(!o){if(f){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(f)}</p>`;return}if(!e){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${$(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(P).join("")}
      </div>
    `}}function $(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const h=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(h)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function P(c){const h=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${I(c)}
        </div>
        <p class="muted small">${n(Pn[c.id]??"")}</p>

        ${c.error?Y(c):""}
        ${c.blocked?`<div class="banner banner-warn">${n(c.blocked)}</div>`:""}
        ${h.map(S=>`<div class="banner banner-warn">${n(S)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${n(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Q(c)}

        ${Z(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(S=>_(c,S)).join("")}
        </div>
        ${x[c.id]?`<p class="error small">${n(x[c.id])}</p>`:""}
        ${ce(c)}

        ${le(c)}
      </div>
    `}function I(c){switch(c.status.State){case"running":return H("running","ok");case"created-but-stopped":return H("stopped","warn");case"not-created":return H("not created","neutral");default:return H("unknown","bad")}}function Y(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function Q(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const h=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${h}.</p>`}function Z(c){const h=c.endpoints??[];return h.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":h.map(S=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(S.label)}</span>
          <code class="endpoint-url">${n(S.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(S.url)}">Copy</button>
        </div>`).join("")}function _(c,h){const S=En[h];if(!S)return"";const F=y[c.id],K=h==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:S.label;return`
      <button class="${S.className}" data-action="svc-${h}" data-svc="${n(c.id)}"
              title="${n(S.title)}" ${F?"disabled":""}>
        ${F===h?'<span class="spinner" aria-label="working"></span>':n(K)}
      </button>
    `}function ce(c){const h=v[c.id]??[];return h.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(h.join(`
`))}</pre>
      </div>
    `}function le(c){const h=q[c.id],S=pe(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(c.id)}">
            ${h?"Close":"Edit"}
          </button>
        </div>
        ${h?re():`<p class="small">${S}</p>`}
        ${j[c.id]?`<p class="error small">${n(j[c.id])}</p>`:""}
        ${N[c.id]?`<p class="muted small">${n(N[c.id])}</p>`:""}
      </div>
    `}function pe(c){const h=c.devnet;return h?`Chain ${h.ChainID} · a block every ${n(h.BlockTime)} · JSON-RPC on ${n(h.BindAddr)}:${h.HTTPPort} · WebSocket on ${n(h.BindAddr)}:${h.WSPort}`:"—"}function re(c){return ae()}function ae(){const c=U;return c?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${n(c.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${c.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${c.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${n(c.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${c.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function se(){q.devnet&&U&&(U.BlockTime=be("#dev-blocktime",U.BlockTime),U.HTTPPort=ge("#dev-http",U.HTTPPort),U.WSPort=ge("#dev-ws",U.WSPort),U.BindAddr=be("#dev-bind",U.BindAddr))}function be(c,h){const S=s.querySelector(c);return S?S.value.trim():h}function ge(c,h){const S=s.querySelector(c);if(!S)return h;const F=Number.parseInt(S.value.trim(),10);return Number.isFinite(F)?F:h}async function ye(c,h){const S=h.dataset.svc??"";switch(c){case"refresh":await C();return;case"copy":h.dataset.copy&&await b(h,h.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await he(S,c.slice(4));return;case"svc-create":case"svc-recreate":await g(S);return;case"svc-wipe":D(S);return;case"toggle-config":L(S);return;case"save-config":await A(S);return;default:return}}async function he(c,h){if(!y[c]){y[c]=h,x[c]=null,l();try{await zt(i,c,h)}catch(S){x[c]=`${h} failed: ${T(S)}${M(S)}`}y[c]=null,await C()}}async function g(c){if(!y[c]){y[c]="create",x[c]=null,v[c]=["starting…"],l();try{await Gt(i,c)}catch(h){x[c]=`${T(h)}${M(h)}`,v[c]=[],y[c]=null,l();return}E==null||E(),E=Ke(i,h=>{if(o)return;const S=h.err?`${h.stepId}: ${h.err}`:h.line?`${h.stepId}: ${h.line}`:`${h.stepId}: done`;if(v[c]=[...(v[c]??[]).filter(K=>K!=="starting…"),S],!!h.err||h.stepId===xn&&!!h.done){E==null||E(),E=null,y[c]=null,h.err&&(x[c]="Provisioning failed — see the log below."),C();return}l()})}}function L(c){if(se(),q[c]=!q[c],j[c]=null,N[c]=null,q[c]){const h=k(c);h!=null&&h.devnet&&(U={...h.devnet})}l()}async function A(c){var F;se(),j[c]=null,N[c]=null;const h=U;if(!h)return;if(h.HTTPPort===h.WSPort){j[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",l();return}try{await Yt(i,c,h)}catch(K){j[c]=T(K),l();return}const S=((F=k(c))==null?void 0:F.status.State)==="running";q[c]=!1,N[c]=S?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await C()}function D(c){const h=k(c);if(!h)return;const S=(h.restartsOnWipe??[]).map(B=>{var X;return((X=k(B))==null?void 0:X.label)??B});te(`
        <h2>Wipe ${n(h.label)}</h2>
        <p class="error">This deletes ${n(h.wipeDiscards)}</p>
        ${S.length?`<p>It also restarts what sits in front of it: ${n(S.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(c)}</button>
        </div>
      `,B=>{if(B==="cancel"||B==="close"){z(),C();return}B==="confirm"&&G(c)});const F=document.getElementById("wipe-confirm-input"),K=document.getElementById("wipe-confirm-btn");F==null||F.addEventListener("input",()=>{K&&(K.disabled=F.value.trim()!==c)}),F==null||F.focus()}async function G(c){const h=document.getElementById("wipe-confirm-btn");h&&(h.disabled=!0,h.textContent="Wiping…");let S;try{S=await Jt(i,c)}catch(F){const K=Fe();if(K){const B=document.createElement("p");B.className="error small",B.textContent=`Wipe failed: ${T(F)}${M(F)}`,K.appendChild(B)}h&&(h.disabled=!1,h.textContent=`Wipe ${c}`);return}u(c,S)}function u(c,h){const S=k(c),F=V=>{var Se;return((Se=k(V))==null?void 0:Se.label)??V},K=[];K.push(h.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const V of h.report.VolumesRemoved??[])K.push(`Volume ${V} deleted.`);for(const V of h.report.VolumesAbsent??[])K.push(`Volume ${V} was already gone.`);h.report.Recreated&&K.push("Container re-created from your saved configuration.");const B=(h.report.Cascaded??[]).map(F),X=(h.report.CascadeSkipped??[]).map(F);te(`
        <h2>${n((S==null?void 0:S.label)??c)} wiped</h2>
        <ul class="plain-list">${K.map(V=>`<li>${n(V)}</li>`).join("")}</ul>
        ${B.length?`<p class="ok">Restarted in front of it: ${n(B.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${X.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(X.join(", "))}.</p>`:""}
        ${h.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(h.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,V=>{(V==="close"||V==="cancel")&&(z(),C())})}async function b(c,h){const S=await Ne(h),F=c.textContent;c.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{o||(c.textContent=F)},1500)}function T(c){return c instanceof Error?c.message:String(c)}function M(c){return c instanceof we&&c.hint?` — ${c.hint}`:""}return()=>{o=!0,E==null||E(),z()}}const Rn="local";function Ln(s){let i=!1,o=!1,e="",f=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ne()}
  `;const y=s.querySelector("#targets-body");ke(s,(l,$)=>{U(l,$)}),x();async function x(){try{const[l,$,P]=await Promise.all([Ee(),Pe(),Lt()]);if(i)return;e=P.os,E(l,$)}catch(l){if(i)return;y.innerHTML=`<p class="error">Failed to load machines: ${n(String(l))}</p>`}}function v(){f&&E(f.targets,f.catalog)}function E(l,$){f={targets:l,catalog:$};const P=e==="linux",I=[...l].sort((Z,_)=>(Z.mode==="local"?-1:0)-(_.mode==="local"?-1:0)),Y=I.length?`<div class="card-grid">${I.map(Z=>Nn(Z,$,Z.mode!=="local"||P,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Q=l.some(Z=>Z.mode==="local");y.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${Y}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${q(P,Q)}
        ${o?Bn():""}
      </section>
    `}function q(l,$){const P=`
      <div class="card">
        <h3>A server over SSH ${H("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${l?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${l?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,I=l?`
        <div class="card">
          <h3>This machine ${H("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${n(e)})`:""} ${H("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return $?`<div class="card-grid card-grid-wide">${P}</div>`:`<div class="card-grid card-grid-wide">${l?I+P:P+I}</div>`}async function U(l,$){var P;if(l==="add-local"){await j();return}if(l==="delete-target"){const I=$.dataset.id;if(!I||!await Le({title:"Remove machine",body:`Remove "${I}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await N(I);return}if(l==="toggle-ssh"){o=!o,k(),v(),o&&((P=s.querySelector("#ssh-host"))==null||P.focus());return}l==="add-ssh"&&await O()}async function j(){k();try{await Xe({id:Rn,mode:"local"}),await x()}catch(l){C(l)}}async function N(l){try{await Nt(l),await x()}catch($){C($)}}async function O(){const l=s.querySelector("#ssh-host"),$=s.querySelector("#ssh-user"),P=s.querySelector("#ssh-key"),I=s.querySelector("#ssh-port"),Y=s.querySelector("#ssh-id");if(!l||!$||!P||!I||!Y)return;const Q=l.value.trim(),Z=$.value.trim(),_=P.value.trim(),ce=I.value.trim(),le=Y.value.trim();if(k(),!Q||!Z||!_){C(new Error("host, user, and key path are required"));return}const pe=le||An(Q),re={Host:Q,User:Z,KeyPath:_};if(ce){const se=Number.parseInt(ce,10);if(!Number.isFinite(se)||se<=0){C(new Error("port must be a positive number"));return}re.Port=se}const ae=s.querySelector("#ssh-submit");ae&&(ae.disabled=!0,ae.textContent="Connecting…");try{await Xe({id:pe,mode:"ssh",ssh:re}),o=!1,await x()}catch(se){C(se),ae&&(ae.disabled=!1,ae.textContent="Add server")}}function C(l){let $=s.querySelector("#targets-error");$||(y.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=s.querySelector("#targets-error")),$.textContent=String(l instanceof Error?l.message:l)}function k(){var l;(l=s.querySelector("#targets-error"))==null||l.remove()}return()=>{i=!0}}function Nn(s,i,o,e){const f=s.wire,y=s.mode==="local"?"this machine":"SSH",x=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:y,v=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(s.id)}">Devnet</a>`;let E,q;if(!f&&!o)E=`${H("can't run a node","warn")} ${H(e||"not Linux","neutral")}`,q=`
      ${v}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Preview setup wizard</a>
    `;else if(!f)E=H("not set up","neutral"),q=`
      <a class="btn" href="#/setup/${encodeURIComponent(s.id)}">Run setup wizard</a>
      ${v}
    `;else{const U=i.networks.find(N=>N.ChainID===f.ChainID),j=U?U.Name:`chain ${f.ChainID}`;E=`${H(j,"ok")} ${H(f.ExecID,"neutral")} ${H(f.BeaconID,"neutral")}${f.Archive?" "+H("archive","warn"):""}`,q=`
      <a class="btn" href="#/dash/${encodeURIComponent(s.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(s.id)}">Logs</a>
      ${v}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${x}</p>
      <p>${E}</p>
      <div class="card-actions">
        ${q}
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function Bn(){return`
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
  `}function An(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const We=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],De=8545,Ue=5052,Me=30303,Hn=[369,943,1],st={369:"default",943:"practise here first"};function Dn(s,i){let o=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ne()}</div>`;const f=s.querySelector("#wizard-body"),y=s.querySelector("#wizard-footer");ke(s,(u,b)=>{ge(u,b)}),ze(s,(u,b)=>{u==="exec-select"?e.execId=b:u==="beacon-select"&&(e.beaconId=b),v()}),s.addEventListener("change",u=>{const b=u.target;b instanceof HTMLInputElement&&(b.id==="data-dir-input"?(ye(),_()):b.id==="checkpoint-toggle"?(e.checkpoint=b.checked,v()):b.id==="exec-snapshot-toggle"&&(e.execSnapshot=b.checked,v()))}),x();async function x(){try{const[u,b]=await Promise.all([Pe(),Ee()]);if(o)return;e.catalog=u;const T=b.find(M=>M.id===i);T!=null&&T.wire&&(e.chainId=T.wire.ChainID,e.execId=T.wire.ExecID,e.beaconId=T.wire.BeaconID,e.archive=T.wire.Archive,T.wire.ExecHTTPPort&&(e.execHTTPPort=String(T.wire.ExecHTTPPort)),T.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(T.wire.BeaconHTTPPort)),T.wire.ExecP2PPort&&(e.execP2PPort=String(T.wire.ExecP2PPort)),T.wire.RPCBindAddr&&(e.rpcBindAddr=T.wire.RPCBindAddr)),v()}catch(u){if(o)return;e.loadError=String(u instanceof Error?u.message:u),v()}}function v(){if(e.loadError){f.innerHTML=`<p class="error">Failed to load: ${n(e.loadError)}</p>`;return}e.catalog&&(f.innerHTML=`
      ${G(e.step)}
      ${q()}
    `,E())}function E(){var b;const u=(b=e.catalog)==null?void 0:b.networks.find(T=>T.ChainID===e.chainId);y.innerHTML=u?ne(u.Name,u.LearnURL):ne()}function q(){switch(e.step){case"network":return U();case"clients":return j();case"mode":return ae();case"review":return se();case"run":return be()}}function U(){const u=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Hn.map(T=>{const M=u.networks.find(S=>S.ChainID===T);if(!M)return"";const c=e.chainId===T,h=st[T]?H(st[T],T===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${T}" type="button">
          <h3>${n(M.Name)} <span class="muted">(chain ${T})</span></h3>
          ${h}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function j(){const u=e.catalog,b=u.networks.find(c=>c.ChainID===e.chainId);if(!b)return'<p class="error">Unknown network.</p>';(e.execId===null||!b.ExecClients.includes(e.execId))&&(e.execId=b.ExecClients[0]??null),(e.beaconId===null||!b.BeaconClients.includes(e.beaconId))&&(e.beaconId=b.BeaconClients[0]??null);const T=b.ExecClients.map(c=>le(c,u)),M=b.BeaconClients.map(c=>le(c,u));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${n(b.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${_e("exec-select",T,e.execId)}
        </label>
        ${re(e.execId,u)}
        <label>
          Beacon client
          ${_e("beacon-select",M,e.beaconId)}
        </label>
        ${re(e.beaconId,u)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function N(u){return u<=0?"—":u>=1?`~${u.toFixed(1)} TB`:`~${Math.round(u*1e3)} GB`}const O=1.1,C=.5,k="Valve reth snapshot",l="rough estimate";function $(u){return u.SnapshotSizeTB}function P(u){return u.SnapshotSizeTB*C}function I(u){return`<p class="muted small">${N($(u))} is the measured size of Valve's reth snapshot for ${n(u.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function Y(u){return{archive:$(u)*1e12*O,full:P(u)*1e12*O}}function Q(u,b){if(!u)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${n(b)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${n(b)}</code>: ${n(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==b)return"";const T=Y(u),M=e.freeBytes>=T.archive,c=e.freeBytes>=T.full,h=`<p class="muted small">Free at <code>${n(b)}</code>: <strong>${Ce(e.freeBytes)}</strong> — archive ${M?"fits":"won't fit"} (${N($(u))}, ${k}), full ${c?"fits":"won't fit"} (${N(P(u))}, ${l}).</p>`;let S="";return e.downgradeNote?S=`<p class="banner banner-warn">${n(e.downgradeNote)}</p>`:c||(S=`<p class="banner banner-warn">Neither full (${N(P(u))}, ${l}) nor archive (${N($(u))}, ${k}) fits the free space here — choose a location with more room.</p>`),h+S}function Z(u,b){if(e.downgradeNote=null,!u||e.freeBytes===null)return;const T=Y(u);e.archive&&e.freeBytes<T.archive&&e.freeBytes>=T.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${b} for archive (${N($(u))}, ${k}) — switched to Full (${N(P(u))}, ${l}). Pick a location with more room to run archive.`)}async function _(){var T;if(e.chainId===null)return;const u=(T=e.catalog)==null?void 0:T.networks.find(M=>M.ChainID===e.chainId),b=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,v();try{const{freeBytes:M}=await Bt(e.targetId,b);if(o)return;e.freeBytes=M,e.probedPath=b,Z(u,b)}catch(M){if(o)return;e.freeBytes=null,e.probedPath=b,e.diskError=String(M instanceof Error?M.message:M)}e.diskProbing=!1,v()}function ce(u){return u?/^https?:\/\/.+/i.test(u)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function le(u,b){const T=b.clients.find(M=>M.id===u);return{value:u,label:T?`${T.id} — ${pe(T.repo)}`:u}}function pe(u){const b=u.split("/");return b.length>=4?b[3]:u}function re(u,b){const T=u?b.clients.find(c=>c.id===u):void 0;if(!T)return"";const M=T.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(T.repo)}" target="_blank" rel="noopener noreferrer">${n(M)}</a></p>`}function ae(){var F,K,B;const u=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",b=(F=e.catalog)==null?void 0:F.networks.find(X=>X.ChainID===e.chainId),T=((B=(K=e.catalog)==null?void 0:K.clients.find(X=>X.id===e.execId))==null?void 0:B.snapshotSupported)??!1,M=b?`${N(P(b))} (${l})`:"Smaller",c=b?`${N($(b))} (${k})`:"Much larger",h=b?` on ${n(b.Name)}`:"",S=b?e.checkpoint?b.SyncLabel:b.GenesisSyncLabel:"";return`
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
          ${b?`<p class="sync-estimate">⏱ Estimated initial sync${h}: <strong>${n(S)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((b==null?void 0:b.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((b==null?void 0:b.CheckpointURL)??"")}" value="${n(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${n(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${T?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${e.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${e.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${n(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${n(e.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${h}</th><td class="yes">${M}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${b?I(b):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${b?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${M}${b?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(u)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(u)}" value="${n(e.dataDir)}" />
          </label>
          ${Q(b,e.dataDir||u)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(u)}/jwt.hex" value="${n(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${De})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${De}" value="${n(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${n(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ue})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ue}" value="${n(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${n(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Me})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Me}" value="${n(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${n(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${n(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${n(e.rpcBindAddrError)}</p>`:""}
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
    `}function se(){const b=e.catalog.networks.find(V=>V.ChainID===e.chainId),T=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,M=e.jwtPath||`${T}/jwt.hex`,c=We.map(V=>`<li>${n(V.title)}</li>`).join(""),h=A(e.execHTTPPort,De),S=A(e.beaconHTTPPort,Ue),F=A(e.execP2PPort,Me),K=h||S||F?`<tr><th>Non-default ports</th><td>${[h?`exec HTTP ${h}`:null,S?`beacon HTTP ${S}`:null,F?`exec p2p ${F}`:null].filter(V=>V!==null).map(n).join(", ")}</td></tr>`:"",{addr:B}=he(e.rpcBindAddr),X=B?`<tr><th>RPC bind address</th><td><code>${n(B)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((b==null?void 0:b.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(T)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(M)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${n(e.checkpointUrl||(b==null?void 0:b.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${K}
            ${X}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${c}</ol>
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function be(){const b=e.catalog.networks.find(B=>B.ChainID===e.chainId),T=b==null?void 0:b.LearnURL,M=new Set(e.events.filter(B=>B.done).map(B=>B.stepId)),c=new Set(e.events.filter(B=>B.err).map(B=>B.stepId)),h=new Map;for(const B of e.events){if(!B.line)continue;const X=h.get(B.stepId)??[];X.push(B.line),h.set(B.stepId,X)}const S=We.map(B=>{var de;const X=M.has(B.id),V=c.has(B.id),Se=V?H("failed","bad"):X?H("done","ok"):H("pending","neutral"),Be=(h.get(B.id)??[]).slice(-5),Ae=(de=e.events.find(xe=>xe.stepId===B.id&&xe.err))==null?void 0:de.err,fe=B.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${T?` <a href="${n(T)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${X?"step-done":""} ${V?"step-error":""}">
          <div class="step-head">${Se} <strong>${n(B.title)}</strong></div>
          ${fe}
          ${Be.length?`<pre class="step-log">${Be.map(xe=>n(xe)).join(`
`)}</pre>`:""}
          ${Ae?`<p class="error small">${n(Ae)}</p>`:""}
        </li>
      `}).join(""),F=e.events.some(B=>B.err),K=We.every(B=>M.has(B.id))||e.events.some(B=>B.stepId==="handshake"&&B.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${S}</ol>
        ${K&&!F?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        ${F?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function ge(u,b){switch(u){case"pick-network":e.chainId=Number(b.dataset.chainId),e.execId=null,e.beaconId=null,v();break;case"goto-network":e.step="network",v();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",v();break;case"goto-mode":e.step="mode",v(),_();break;case"goto-review":if(ye(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){v();break}e.step="review",v();break;case"start-setup":D();break}}function ye(){const u=s.querySelectorAll('input[name="mode"]');for(const B of Array.from(u))B.checked&&(e.archive=B.value==="archive");const b=s.querySelector("#data-dir-input"),T=s.querySelector("#jwt-path-input");b&&(e.dataDir=b.value.trim()),T&&(e.jwtPath=T.value.trim());const M=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),h=s.querySelector("#exec-p2p-port-input");M&&(e.execHTTPPort=M.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),h&&(e.execP2PPort=h.value.trim());const S=s.querySelector("#rpc-bind-addr-input");S&&(e.rpcBindAddr=S.value.trim());const F=s.querySelector("#checkpoint-url-input");F&&(e.checkpointUrl=F.value.trim());const K=s.querySelector("#snapshot-key-input");K&&(e.snapshotKey=K.value.trim()),e.execHTTPPortError=L(e.execHTTPPort).error??null,e.beaconHTTPPortError=L(e.beaconHTTPPort).error??null,e.execP2PPortError=L(e.execP2PPort).error??null,e.rpcBindAddrError=he(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?ce(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function he(u){if(!u)return{};const b=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(u);return b?b.slice(1).every(T=>Number(T)<=255)?{addr:u}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(u)&&u.includes(":")?{addr:u}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const g=/^\d+$/;function L(u){if(!u)return{};if(!g.test(u))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const b=Number(u);return!Number.isInteger(b)||b<1||b>65535?{error:"Port must be between 1 and 65535."}:{port:b}}function A(u,b){const{port:T}=L(u);if(!(T===void 0||T===b))return T}async function D(){var h;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(h=e.streamStop)==null||h.call(e),e.streamStop=null,v();const u={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(u.DataDir=e.dataDir),e.jwtPath&&(u.JWTPath=e.jwtPath);const b=A(e.execHTTPPort,De),T=A(e.beaconHTTPPort,Ue),M=A(e.execP2PPort,Me);b!==void 0&&(u.ExecHTTPPort=b),T!==void 0&&(u.BeaconHTTPPort=T),M!==void 0&&(u.ExecP2PPort=M);const{addr:c}=he(e.rpcBindAddr);c!==void 0&&(u.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(u.CheckpointURL=e.checkpointUrl):u.NoCheckpoint=!0,e.execSnapshot&&(u.ExecSnapshot=!0,u.SnapshotKey=e.snapshotKey);try{await At(e.targetId,u)}catch(S){if(!(S instanceof we&&S.status===409)){e.starting=!1,e.startError=String(S instanceof Error?S.message:S),v();return}}e.starting=!1,e.step="run",v(),e.streamStop=Ke(e.targetId,S=>{o||(e.events.push(S),e.step==="run"&&v())})}function G(u){const b=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],M=b.map(c=>c.id).indexOf(u);return`
      <ol class="wizard-progress">
        ${b.map((c,h)=>`<li class="${h===M?"current":h<M?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var u;o=!0,(u=e.streamStop)==null||u.call(e)}}const Un=document.querySelector("#app"),{contentEl:Mn,setActiveNav:On}=un(Un);let oe=null;function Fn(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[o,e]=i;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function me(s){const i=document.createElement("div");return Mn.replaceChildren(i),s(i)}function rt(){if(oe){try{oe()}catch{}oe=null}const{screen:s,id:i}=Fn();switch(On(s),s){case"setup":if(!i){location.hash="#/targets";return}oe=me(o=>Dn(o,i));break;case"dash":if(!i){location.hash="#/targets";return}oe=me(o=>mn(o,i));break;case"logs":if(!i){location.hash="#/targets";return}oe=me(o=>bn(o,i));break;case"security":if(!i){location.hash="#/targets";return}oe=me(o=>yn(o,i));break;case"diag":if(!i){location.hash="#/targets";return}oe=me(o=>gn(o,i));break;case"services":if(!i){location.hash="#/targets";return}oe=me(o=>In(o,i));break;case"rpc":oe=me(o=>Sn(o));break;case"settings":oe=me(o=>$n(o));break;case"targets":default:oe=me(o=>Ln(o));break}}window.addEventListener("hashchange",rt);rt();
