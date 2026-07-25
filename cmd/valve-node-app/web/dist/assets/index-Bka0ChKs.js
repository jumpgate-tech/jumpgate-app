var pt=Object.defineProperty;var ht=(a,c,o)=>c in a?pt(a,c,{enumerable:!0,configurable:!0,writable:!0,value:o}):a[c]=o;var He=(a,c,o)=>ht(a,typeof c!="symbol"?c+"":c,o);(function(){const c=document.createElement("link").relList;if(c&&c.supports&&c.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const b of u)if(b.type==="childList")for(const x of b.addedNodes)x.tagName==="LINK"&&x.rel==="modulepreload"&&e(x)}).observe(document,{childList:!0,subtree:!0});function o(u){const b={};return u.integrity&&(b.integrity=u.integrity),u.referrerPolicy&&(b.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?b.credentials="include":u.crossOrigin==="anonymous"?b.credentials="omit":b.credentials="same-origin",b}function e(u){if(u.ep)return;u.ep=!0;const b=o(u);fetch(u.href,b)}})();function ft(){return W("/api/host")}function Se(){return W("/api/catalog")}function xe(){return W("/api/targets")}function Ve(a){return W("/api/targets",{method:"POST",headers:ce,body:JSON.stringify(a)})}function mt(a){return W(`/api/targets/${encodeURIComponent(a)}`,{method:"DELETE"})}function bt(a,c){return W(`/api/targets/${encodeURIComponent(a)}/disk?path=${encodeURIComponent(c)}`)}function gt(a,c){return W(`/api/targets/${encodeURIComponent(a)}/setup`,{method:"POST",headers:ce,body:JSON.stringify(c)})}function Ke(a,c){const o=new EventSource(`/api/targets/${encodeURIComponent(a)}/setup/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function vt(a,c){const o=new EventSource(`/api/targets/${encodeURIComponent(a)}/monitor/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function yt(a,c=200){return W(`/api/targets/${encodeURIComponent(a)}/logs?n=${c}`)}function $t(a,c){const o=new EventSource(`/api/targets/${encodeURIComponent(a)}/logs/stream`);return o.onmessage=e=>{try{c(JSON.parse(e.data))}catch{}},()=>o.close()}function Ye(a,c){const o=c===void 0?{}:{lines:c};return W(`/api/targets/${encodeURIComponent(a)}/explain`,{method:"POST",headers:ce,body:JSON.stringify(o)})}function wt(a,c,o){return W(`/api/targets/${encodeURIComponent(a)}/services/${c}/${o}`,{method:"POST"})}function kt(a,c){return W(`/api/targets/${encodeURIComponent(a)}/services/${c}/clear`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:c})})}function Tt(a){return W(`/api/targets/${encodeURIComponent(a)}/du`)}function Ct(a){return W(`/api/targets/${encodeURIComponent(a)}/endpoints`)}function St(a){return W(`/api/targets/${encodeURIComponent(a)}/firewall`)}function xt(a){return W(`/api/targets/${encodeURIComponent(a)}/diagnostics`)}function Pt(a){return W(`/api/targets/${encodeURIComponent(a)}/diagnostics/latest`)}function Et(a){return W(`/api/targets/${encodeURIComponent(a)}/containers`)}function It(a,c,o){return W(`/api/targets/${encodeURIComponent(a)}/containers/${c}/${o}`,{method:"POST"})}async function Rt(a,c){const o=await fetch(`/api/targets/${encodeURIComponent(a)}/containers/${c}/wipe`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:c})}),e=await o.text();let u=null;try{u=e?JSON.parse(e):null}catch{}if(u&&typeof u=="object"&&"report"in u)return u;const b=u&&typeof u=="object"&&typeof u.error=="string"?u.error:o.statusText||`HTTP ${o.status}`;throw new ve(o.status,b)}function Lt(a,c){return W(`/api/targets/${encodeURIComponent(a)}/containers/${c}/provision`,{method:"POST"})}async function Nt(a){const c=await fetch(`/api/targets/${encodeURIComponent(a)}/containers/devnet/reset`,{method:"POST",headers:ce}),o=await c.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:c.statusText||`HTTP ${c.status}`;throw new ve(c.status,u)}function Bt(a,c,o){return W(`/api/targets/${encodeURIComponent(a)}/containers/${c}/config`,{method:"PUT",headers:ce,body:JSON.stringify(o)})}function At(){return W("/api/gateways")}function Ht(a){return W("/api/gateways",{method:"POST",headers:ce,body:JSON.stringify(a)})}function Dt(a){return W(`/api/gateways/${encodeURIComponent(a)}/tls/verify`)}function Ut(a){return W(`/api/gateways/${encodeURIComponent(a)}`,{method:"DELETE"})}function Mt(a,c){return W(`/api/gateways/${encodeURIComponent(a)}/config`,{method:"PUT",headers:ce,body:JSON.stringify(c)})}function Ot(a,c){return W(`/api/gateways/${encodeURIComponent(a)}/${c}`,{method:"POST"})}function Ft(a){return W(`/api/gateways/${encodeURIComponent(a)}/provision`,{method:"POST"})}async function jt(a){const c=await fetch(`/api/gateways/${encodeURIComponent(a)}/wipe`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:a})}),o=await c.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:c.statusText||`HTTP ${c.status}`;throw new ve(c.status,u)}function qt(a){return W(`/api/chainlist/${a}`)}function Wt(){return W("/api/settings")}function _t(a){return W("/api/settings",{method:"PUT",headers:ce,body:JSON.stringify(a)})}class ve extends Error{constructor(o,e,u,b){super(e);He(this,"status");He(this,"hint");He(this,"code");this.name="ApiError",this.status=o,this.hint=u,this.code=b}}const ce={"Content-Type":"application/json"};async function W(a,c){const o=await fetch(a,c);if(!o.ok){let u=o.statusText||`HTTP ${o.status}`,b,x;try{const w=await o.json();w&&typeof w.error=="string"&&w.error&&(u=w.error),w&&typeof w.hint=="string"&&w.hint&&(b=w.hint),w&&typeof w.code=="string"&&w.code&&(x=w.code)}catch{}throw new ve(o.status,u,b,x)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const Ze="https://learn.valve.city/rpc";function t(a){return a.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ee(a,c){const o=a&&c&&c!==Ze?` <span class="footer-sep">·</span> <a href="${t(c)}" target="_blank" rel="noopener noreferrer">${t(a)}</a>`:"";return`
    <footer class="footer">
      <a href="${t(Ze)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function Kt(a){a.innerHTML=`
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
  `;const c=a.querySelector("#content"),o=Array.from(a.querySelectorAll("[data-nav]"));return{contentEl:c,setActiveNav:u=>{for(const b of o)b.classList.toggle("active",b.dataset.nav===u)}}}function $e(a){return Number.isFinite(a)?a.toLocaleString("en-US"):"—"}function zt(a){return Number.isFinite(a)?`${a.toFixed(1)}%`:"—"}function Jt(a){if(!Number.isFinite(a)||a<0)return"—";if(a<60)return`~${Math.round(a)}s`;const c=Math.round(a/60),o=Math.floor(c/60),e=c%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const u=Math.floor(o/24),b=o%24;return`~${u}d ${b}h`}function D(a,c){return`<span class="badge badge-${c}">${t(a)}</span>`}function ie(a){return`<span class="dot dot-${a}"></span>`}const Xe=["B","KB","MB","GB","TB","PB"];function we(a){if(!Number.isFinite(a)||a<0)return"—";if(a===0)return"0 B";let c=a,o=0;for(;c>=1024&&o<Xe.length-1;)c/=1024,o++;const e=c<10?2:c<100?1:0;return`${c.toFixed(e)} ${Xe[o]}`}async function Le(a){try{return await navigator.clipboard.writeText(a),!0}catch{return!1}}function ye(a,c){a.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!a.contains(e))return;const u=e.dataset.action;u&&c(u,e,o)})}function Fe(a,c,o){const e=c.find(b=>b.value===o),u=c.map(b=>`
      <li class="dropdown-option${b.value===o?" selected":""}" role="option"
          aria-selected="${b.value===o}" data-value="${t(b.value)}">
        ${t(b.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${t(a)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${t(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${u}</ul>
    </div>
  `}function Ie(a){a.querySelectorAll(".dropdown.open").forEach(c=>{var o;c.classList.remove("open"),(o=c.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function ze(a,c){a.addEventListener("click",u=>{const b=u.target,x=b.closest(".dropdown-trigger");if(x&&a.contains(x)){const B=x.closest(".dropdown"),j=!!B&&!B.classList.contains("open");Ie(a),B&&j&&(B.classList.add("open"),x.setAttribute("aria-expanded","true"));return}const w=b.closest(".dropdown-option");if(w&&a.contains(w)){const B=w.closest(".dropdown");Ie(a),c((B==null?void 0:B.dataset.dropdown)??"",w.dataset.value??"");return}Ie(a)});const o=u=>{if(!a.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const b=u.target;(!b.closest(".dropdown")||!a.contains(b))&&Ie(a)},e=u=>{if(!a.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}u.key==="Escape"&&Ie(a)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const qe="app-modal";let Oe=null;function Q(a,c){z();const o=document.createElement("div");o.className="modal-overlay",o.id=qe,o.innerHTML=`<div class="modal">${a}</div>`,o.addEventListener("click",u=>{const b=u.target.closest("[data-modal-action]");b!=null&&b.dataset.modalAction?c(b.dataset.modalAction):u.target===o&&c("cancel")});const e=u=>{u.key==="Escape"&&c("cancel")};document.addEventListener("keydown",e),Oe=e,document.body.appendChild(o)}function z(){var a;(a=document.getElementById(qe))==null||a.remove(),Oe&&(document.removeEventListener("keydown",Oe),Oe=null)}function je(){return document.querySelector(`#${qe} .modal`)}function Re(a){return new Promise(c=>{var u;let o=!1;const e=b=>{o||(o=!0,z(),c(b))};Q(`
        <h2>${t(a.title)}</h2>
        <p>${t(a.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${a.danger?" btn-danger":""}" data-modal-action="confirm">${t(a.confirmLabel)}</button>
        </div>
      `,b=>e(b==="confirm")),(u=document.querySelector(`#${qe} [data-modal-action="confirm"]`))==null||u.focus()})}const Gt=85,We={exec:"Execution",beacon:"Beacon"};function Vt(a,c){let o=!1,e=null,u=null,b=null,x=null,w=null,B=null,j=null,M=null;const q={exec:null,beacon:null};let P=null;a.innerHTML=`<h1>Dashboard: ${t(c)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ee()}</div>`;const O=a.querySelector("#dash-body"),C=a.querySelector("#dash-footer");O.addEventListener("click",g=>{const I=g.target.closest("[data-action]");if(!I||!O.contains(I))return;const N=I.dataset.action;if(N==="svc-action"){const R=I.dataset.svc,_=I.dataset.kind;R&&_&&me(R,_)}else if(N==="open-clear"){const R=I.dataset.svc;R&&ge(R)}else if(N==="copy"){const R=I.dataset.copy;R&&be(I,R)}else N==="retry-du"?l():N==="retry-endpoints"&&f()}),$();async function $(){let g,I;try{const[R,_]=await Promise.all([xe(),Se()]);g=R.find(d=>d.id===c),I=_}catch(R){if(o)return;O.innerHTML=`<p class="error">Failed to load target: ${t(String(R))}</p>`;return}if(o)return;if(!g){O.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!g.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const N=I==null?void 0:I.networks.find(R=>R.ChainID===g.wire.ChainID);N&&(C.innerHTML=ee(N.Name,N.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',e=vt(c,R=>{o||(E(R),u=R,b=R,H())}),l(),f()}async function l(){B=null;try{w=await Tt(c)}catch(g){w=null,B=String(g instanceof Error?g.message:g)}o||H()}async function f(){M=null;try{j=await Ct(c)}catch(g){j=null,M=String(g instanceof Error?g.message:g)}o||H()}function E(g){if(!u)return;const I=(new Date(g.at).getTime()-new Date(u.at).getTime())/1e3,N=g.execHead-u.execHead;if(I>0&&N>=0){const R=N/I;x=x===null?R:x*.7+R*.3}}function H(){if(!b)return;const g=b;O.innerHTML=`
      <p class="dash-status">${Y(g)}</p>
      <div class="card-grid">
        ${te(g)}
        ${Z(g)}
        ${ae(g)}
        ${le(g)}
        ${de(g)}
        ${pe()}
      </div>
      <p class="muted small">Last updated ${t(new Date(g.at).toLocaleTimeString())}</p>
    `}function Y(g){return!g.execActive&&!g.beaconActive?D("Node not running","bad"):g.execSyncing||g.beaconDistance>0?D("Syncing","warn"):D("Running · synced","ok")}function X(g){const N=g.refHead>0?g.refHead-g.execHead:null,R=N!==null&&N>0&&x&&x>0?Jt(N/x):N!==null&&N<=0?"caught up":"—";return{lag:N,eta:R}}function Z(g){const{lag:I,eta:N}=X(g);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${g.execActive?g.execSyncing?D("syncing","warn"):g.execHead===0?D("no data","neutral"):D("synced","ok"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${$e(g.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${I!==null?$e(g.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${I!==null?$e(Math.max(I,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${N}</dd></div>
        </dl>
      </div>
    `}function ae(g){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${g.beaconActive?g.beaconSlot===0?D("no data","neutral"):g.beaconDistance===0?D("synced","ok"):D("syncing","warn"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${$e(g.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${$e(g.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function le(g){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${$e(g.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${$e(g.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(g){const I=g.diskUsedPct>=Gt,N=`
      <div class="meter"><div class="meter-fill ${I?"meter-warn":""}" style="width:${Math.min(g.diskUsedPct,100)}%"></div></div>
      <p>${zt(g.diskUsedPct)} used</p>
    `;if(B)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${N}
          <p class="error small">${t(B)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!w)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${N}
          <p class="muted">Loading…</p>
        </div>
      `;const R=w.ExpectedExecBytes>0?Math.min(w.ExecBytes/w.ExpectedExecBytes*100,100):0,_=w.ExpectedBeaconBytes>0?Math.min(w.BeaconBytes/w.ExpectedBeaconBytes*100,100):0,{lag:d,eta:m}=X(g),T=d!==null&&d>0&&x!==null&&x>0;return`
      <div class="card ${I?"card-warn":""}">
        <h3>Storage</h3>
        ${N}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${we(w.ExecBytes)} of ~${we(w.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${R}%"></div></div>
        ${T?`<p class="muted small">Estimated time remaining: ${t(m)}</p>`:""}
        <p class="muted small">Beacon — ${we(w.BeaconBytes)} of ~${we(w.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${_}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${we(w.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${t(w.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${t(w.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function pe(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${t(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const g=j,I=g.ExecReachable&&!g.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",N=g.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${t(g.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${t(g.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ie(g.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${t(g.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(g.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ie(g.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${t(g.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(g.BeaconHTTP)}">Copy</button>
        </div>
        ${I}
        ${N}
      </div>
    `}function oe(g,I){const N=We[g],R=q[g],_=(d,m,T)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${g}" data-kind="${d}" ${R!==null||T?"disabled":""}>${R===d?ne():t(m)}</button>`;return`
      <div class="service-row">
        <span>${t(N)} ${I?D("active","ok"):D("down","bad")}</span>
        <div class="service-actions">
          ${_("start","Start",I)}
          ${_("stop","Stop",!I)}
          ${_("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${g}" ${R!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function te(g){return`
      <div class="card">
        <h3>Services</h3>
        ${oe("exec",g.execActive)}
        ${oe("beacon",g.beaconActive)}
        ${P?`<p class="error small">${t(P)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(c)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(c)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(c)}">Diagnostics →</a>
        </p>
      </div>
    `}function ne(){return'<span class="spinner" aria-label="working"></span>'}async function me(g,I){if(q[g]===null){q[g]=I,P=null,H();try{await wt(c,g,I)}catch(N){P=`${We[g]} ${I} failed: ${N instanceof Error?N.message:String(N)}`}q[g]=null,o||H()}}async function be(g,I){const N=await Le(I),R=g.textContent;g.textContent=N?"Copied!":"Copy failed",setTimeout(()=>{o||(g.textContent=R)},1500)}function ge(g){const I=We[g],N=w?we(g==="exec"?w.ExecBytes:w.BeaconBytes):"unknown (disk usage hasn't loaded)";Q(`
        <h2>Clear ${t(I)} data</h2>
        <p class="error">
          This stops the ${t(I.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${t(N)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${t(g)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,d=>{if(d==="cancel"){z();return}d==="confirm"&&he(g)});const R=document.getElementById("clear-confirm-input"),_=document.getElementById("clear-confirm-btn");R==null||R.addEventListener("input",()=>{_&&(_.disabled=R.value.trim()!==g)}),R==null||R.focus()}async function he(g){const I=document.getElementById("clear-confirm-btn");I&&(I.disabled=!0,I.textContent="Clearing…");try{await kt(c,g),z(),l()}catch(N){const R=je();if(R){const _=document.createElement("p");_.className="error small",_.textContent=`Clear failed: ${N instanceof Error?N.message:String(N)}`,R.appendChild(_)}I&&(I.disabled=!1,I.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),z()}}const Qe=500,et="valve-node-app.explain-consent";function Yt(a,c){let o=!1,e=null;const u=[];a.innerHTML=`
    <h1>Logs: ${t(c)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ee()}</div>
  `;const b=a.querySelector("#logs-body"),x=a.querySelector("#logs-footer");ye(a,$=>{$==="explain"&&M()}),w();async function w(){let $,l;try{const[E,H]=await Promise.all([xe(),Se()]);$=E.find(Y=>Y.id===c),l=H}catch(E){if(o)return;b.innerHTML=`<p class="error">Failed to load target: ${t(String(E))}</p>`;return}if(o)return;if(!$){b.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!$.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const f=l==null?void 0:l.networks.find(E=>E.ChainID===$.wire.ChainID);f&&(x.innerHTML=ee(f.Name,f.LearnURL));try{const E=await yt(c,200);if(o)return;u.push(...E)}catch(E){if(o)return;b.innerHTML=`<p class="error">Failed to load logs: ${t(String(E))}</p>`;return}B(),e=$t(c,E=>{o||(u.push(E),u.length>Qe&&u.splice(0,u.length-Qe),B())})}function B(){const $=u.filter(f=>f.severity==="error"||f.severity==="critical");b.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${D(String($.length),$.length?"bad":"neutral")}</h2>
          <div class="log-lines">${$.length?$.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const l=b.querySelector(".log-lines");l&&(l.scrollTop=l.scrollHeight)}function j($){const l=$.severity||"info",f=$.learnUrl?` <a href="${t($.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${t(l)}">
        <span class="log-time">${t(new Date($.at).toLocaleTimeString())}</span>
        <span class="log-unit">${t($.unit)}</span>
        <span class="log-sev">${t(l)}</span>
        <span class="log-text">${t($.line)}</span>
        ${$.explain?`<div class="log-explain">${t($.explain)}${f}</div>`:""}
      </div>
    `}async function M(){const $=u.filter(f=>f.severity==="error"||f.severity==="critical").map(f=>f.line).slice(-40);if(!(localStorage.getItem(et)==="1")){q($);return}await P($)}function q($){const l=$.length?`<pre class="explain-excerpt">${$.map(f=>t(f)).join(`
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
    `,f=>{f==="proceed"?(localStorage.setItem(et,"1"),C(),P($)):C()})}async function P($){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const l=$.length?await Ye(c,$):await Ye(c);if(o)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${t(l.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${l.sentExcerpt.map(f=>t(f)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,f=>{f==="close"&&C()})}catch(l){if(o)return;if(l instanceof ve&&l.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,f=>{f==="close"&&C()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${t(l instanceof Error?l.message:String(l))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,f=>{f==="close"&&C()})}}function O($,l){C();const f=document.createElement("div");f.className="modal-overlay",f.id="explain-modal",f.innerHTML=`<div class="modal">${$}</div>`,f.addEventListener("click",E=>{const H=E.target.closest("[data-modal-action]");H!=null&&H.dataset.modalAction&&l(H.dataset.modalAction),E.target===f&&l("cancel")}),document.body.appendChild(f)}function C(){var $;($=document.getElementById("explain-modal"))==null||$.remove()}return()=>{o=!0,e==null||e(),C()}}function Zt(a,c){let o=!1,e=null,u=null,b=!1,x=!1;a.innerHTML=`<h1>Network diagnostics: ${t(c)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ee()}</div>`;const w=a.querySelector("#diag-body"),B=a.querySelector("#diag-footer");ye(a,(l,f)=>{var E;if(l==="run")M();else if(l==="toggle")(E=f.closest(".check-item"))==null||E.classList.toggle("expanded");else if(l==="copy"){const H=f.dataset.copy;H&&$(f,H)}}),j();async function j(){let l,f;try{const[H,Y]=await Promise.all([xe(),Se()]);l=H.find(X=>X.id===c),f=Y}catch(H){if(o)return;w.innerHTML=`<p class="error">Failed to load target: ${t(String(H))}</p>`;return}if(o)return;if(!l){w.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const E=f==null?void 0:f.networks.find(H=>H.ChainID===l.wire.ChainID);E&&(B.innerHTML=ee(E.Name,E.LearnURL));try{e=await Pt(c),x=!0}catch(H){u=String(H instanceof Error?H.message:H)}o||q()}async function M(){b=!0,u=null,q();try{e=await xt(c),x=!0}catch(l){u=String(l instanceof Error?l.message:l)}b=!1,o||q()}function q(){w.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(c)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${b?"disabled":""}>${b?"Running…":"Run diagnostics"}</button>
      </div>
      ${u?`<p class="error">${t(u)}</p>`:""}
      ${P()}
    `}function P(){if(!x&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const l=new Date(e.at).toLocaleString(),f=e.failedId?`<p><strong>Failed at: ${t(O(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${t(l)} — trigger: ${t(e.trigger)}</p>
      ${f}
      <ul class="check-list">${e.items.map(C).join("")}</ul>
    `}function O(l){var f;return((f=e==null?void 0:e.items.find(E=>E.ID===l))==null?void 0:f.Title)??l}function C(l){const f=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral",E=l.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${E?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${D(E?"failed here":l.Status,f)}
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
    `}async function $(l,f){const E=await Le(f),H=l.textContent;l.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=H)},1500)}return()=>{o=!0}}function Xt(a,c){let o=!1,e=[],u=null,b=!1,x=!1;a.innerHTML=`<h1>Security: ${t(c)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ee()}</div>`;const w=a.querySelector("#sec-body"),B=a.querySelector("#sec-footer");ye(a,(C,$)=>{var l;if(C==="rerun")M();else if(C==="toggle")(l=$.closest(".check-item"))==null||l.classList.toggle("expanded");else if(C==="copy"){const f=$.dataset.copy;f&&O($,f)}}),j();async function j(){let C,$;try{const[f,E]=await Promise.all([xe(),Se()]);C=f.find(H=>H.id===c),$=E}catch(f){if(o)return;w.innerHTML=`<p class="error">Failed to load target: ${t(String(f))}</p>`;return}if(o)return;if(!C){w.innerHTML=`<p class="error">Target "${t(c)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(c)}">Run the setup wizard →</a></p>`;return}const l=$==null?void 0:$.networks.find(f=>f.ChainID===C.wire.ChainID);l&&(B.innerHTML=ee(l.Name,l.LearnURL)),await M()}async function M(){b=!0,u=null,q();try{e=await St(c),x=!0}catch(C){u=String(C instanceof Error?C.message:C)}b=!1,o||q()}function q(){w.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(c)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${b?"disabled":""}>${b?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${t(u)}</p>`:""}
      ${!x&&b?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(P).join("")}</ul>`:x?'<p class="muted">No checks returned.</p>':""}
    `}function P(C){const $=C.Status==="pass"?"ok":C.Status==="fail"?"bad":C.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${D(C.Status,$)}
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
    `}async function O(C,$){const l=await Le($),f=C.textContent;C.textContent=l?"Copied!":"Copy failed",setTimeout(()=>{o||(C.textContent=f)},1500)}return()=>{o=!0}}const Qt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function en(a){let c=!1,o=!1,e=!1,u=null,b=!1,x=null,w=null;a.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ee()}`;const B=a.querySelector("#settings-body");ye(a,P=>{if(P==="save"&&q(),P==="clear-key"){if(!x)return;o=!0;const O=a.querySelector("#ai-key");O&&(O.value=""),M(x)}}),ze(a,(P,O)=>{P!=="ai-provider"||!x||(w=O,b=!1,M(x))}),j();async function j(){try{const P=await Wt();if(c)return;x=P,M(P)}catch(P){if(c)return;B.innerHTML=`<p class="error">Failed to load settings: ${t(String(P))}</p>`}}function M(P){var $;const O=w??P.aiProvider;B.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Fe("ai-provider",Qt.map(l=>({value:l.value,label:l.label})),O)}
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
        ${b?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const C=a.querySelector("#ai-key");C==null||C.addEventListener("input",()=>{o=!0,b=!1}),($=a.querySelector("#ref-rpc-base"))==null||$.addEventListener("input",()=>{b=!1})}async function q(){const P=a.querySelector("#ai-key"),O=a.querySelector("#ref-rpc-base");if(!P||!O||!x)return;const C={aiProvider:w??x.aiProvider,refRpcBase:O.value.trim()};o&&(C.aiKey=P.value),e=!0,u=null,b=!1,M(x);try{const $=await _t(C);if(c)return;x=$,o=!1,e=!1,b=!0,M($)}catch($){if(c)return;e=!1,u=String($ instanceof Error?$.message:$),M(x)}}return()=>{c=!0}}const tn=6,nn="run",an={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function sn(a){let c=!1,o=null,e=null;const u={},b={},x={},w={},B={},j={},M={},q={};let P=null;a.innerHTML=`
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
  `;const O=a.querySelector("#rpc-body");ye(a,(n,s)=>{d(n,s)}),ze(a,(n,s)=>{if(n.startsWith("chain-")){const r=n.slice(6);u[r]=Number.parseInt(s,10),f()}}),C();async function C(){try{const n=await At();if(c)return;o=n,e=null;for(const s of n.gateways??[]){const r=s.networks??[],h=u[s.id];(h==null||!r.some(v=>v.chainId===h))&&(u[s.id]=r.length?r[0].chainId:null)}}catch(n){if(c)return;o=null,e=ue(n)}f()}function $(n){return((o==null?void 0:o.gateways)??[]).find(s=>s.id===n)}function l(n,s){if(s!=null)return(n.networks??[]).find(r=>r.chainId===s)}function f(){if(c)return;if(e){O.innerHTML=`<p class="error">Could not read the gateways: ${t(e)}</p>`;return}if(!o){O.innerHTML='<p class="muted">Loading…</p>';return}const n=o.gateways??[];O.innerHTML=`
      ${n.map(H).join("")}
      ${n.length===0?E():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${n.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function E(){return((o==null?void 0:o.targets)??[]).length===0?`
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
    `}function H(n){const s=l(n,u[n.id]??null);return`
      <section class="rpc-gateway">
        ${Y(n)}
        ${n.error?ae(n):""}
        ${n.blocked?`<div class="banner banner-warn">${t(n.blocked)}</div>`:""}
        ${(n.warnings??[]).map(r=>`<div class="banner banner-warn">${t(r)}</div>`).join("")}
        ${N(n)}
        ${x[n.id]?`<p class="error small">${t(x[n.id])}</p>`:""}
        ${de(n)}
        ${B[n.id]?me(n):""}
        ${te(n,s)}
      </section>
    `}function Y(n){var r;const s=n.status.State==="running";return`
      <div class="rpc-bar${s?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${Z(n)}
            <strong>${t(n.label)}</strong>
            ${X(n)}
            <span class="muted small">on ${t(n.placement.targetId)} · ${t(n.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(n.actions??[]).map(h=>le(n,h)).join("")}
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${t(n.id)}">
              ${B[n.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${t(n.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${s?`<code class="endpoint-url">${t(n.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(n.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${t(((r=(n.networks??[])[0])==null?void 0:r.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${t(n.baseUrl)}</code> once it is running.</span>`}
        </div>
        ${pe(n)}
      </div>
    `}function X(n){switch(n.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function Z(n){return n.status.State==="running"?ie("ok"):n.status.State==="unknown"?ie("bad"):ie("neutral")}function ae(n){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${t(n.error??"")}</div>
        ${n.hint?`<div class="small">${t(n.hint)}</div>`:""}
      </div>
    `}function le(n,s){const r=an[s];if(!r)return"";const h=b[n.id];return`
      <button class="${r.className}" data-action="gw-${s}" data-gid="${t(n.id)}"
              title="${t(r.title)}" ${h?"disabled":""}>
        ${h===s?'<span class="spinner" aria-label="working"></span>':t(r.label)}
      </button>
    `}function de(n){const s=w[n.id]??[];return s.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${t(n.placement.targetId)}</p>
        <pre class="step-log">${t(s.join(`
`))}</pre>
      </div>
    `}function pe(n){const s=n.networks??[],r=u[n.id]??null,h=`
      <button class="chip chip-add" data-action="add-chain" data-gid="${t(n.id)}"
              title="Add a network for this gateway to front">+ Network</button>
    `;if(s.length===0)return`
        <div class="rpc-chiprow">
          <span class="muted small">No networks yet — eRPC refuses a configuration with none, so add one before creating the gateway.</span>
          ${h}
        </div>
      `;if(s.length>tn){const v=s.map(y=>({value:String(y.chainId),label:`${y.name} (${y.chainId})${y.serviceable?"":" — no working endpoint"}`}));return`
        <div class="rpc-chiprow">
          <span class="muted small">Fronting ${s.length} networks</span>
          ${Fe(`chain-${n.id}`,v,r==null?null:String(r))}
          ${h}
        </div>
      `}return`
      <div class="rpc-chiprow">
        ${s.map(v=>oe(n,v,v.chainId===r)).join("")}
        ${h}
      </div>
    `}function oe(n,s,r){const h=!s.serviceable;return`
      <button class="chip card-selectable${r?" selected":""}${h?" chip-bad":""}"
              data-action="select-chain" data-gid="${t(n.id)}" data-chain="${s.chainId}"
              title="${t(h?`${s.name}: no endpoint on this chain can be used right now`:`${s.name} · ${s.path}`)}">
        <span class="chip-dot">${ie(h?"bad":"ok")}</span>
        <span class="chip-name">${t(s.name)}</span>
        <span class="chip-id">${s.chainId}</span>
      </button>
    `}function te(n,s){if(!s)return'<div class="card rpc-upstreams"><p class="muted small">Pick a network above to see the servers behind it.</p></div>';const r=s.upstreams??[];return`
      <div class="card rpc-upstreams">
        <div class="service-head">
          <h2>${t(s.name)} <span class="muted">· chain ${s.chainId}</span></h2>
          <div class="card-actions">
            <button class="btn" data-action="add-endpoint" data-gid="${t(n.id)}" data-chain="${s.chainId}">Add an endpoint</button>
            <button class="btn btn-ghost" data-action="remove-chain" data-gid="${t(n.id)}" data-chain="${s.chainId}">Remove network</button>
          </div>
        </div>
        ${s.url?`<div class="endpoint-row">${ie("ok")}<span class="muted small">callers dial</span>
                 <code class="endpoint-url">${t(s.url)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(s.url)}">Copy</button></div>`:`<p class="muted small">Path <code>${t(s.path)}</code> — the full URL appears once the gateway is running.</p>`}
        ${(s.warnings??[]).map(h=>`<div class="banner banner-warn">${t(h)}</div>`).join("")}
        ${r.map(h=>ne(n,s,h)).join("")}
        ${r.length===0?'<p class="muted small">No endpoint yet, so there is nowhere for calls on this path to go.</p>':""}
      </div>
    `}function ne(n,s,r){const h=`${n.id}|${s.chainId}|${r.id}`,v=r.actions??[];return`
      <div class="upstream-row${r.problem?" upstream-row-bad":""}">
        <span class="upstream-state">${r.problem?ie("bad"):ie("ok")}</span>
        <div class="upstream-what">
          <div class="upstream-label">
            ${t(r.label)}
            ${r.local?D("preferred","ok"):D("fallback","neutral")}
            ${r.recentOnly?D("recent blocks only","warn"):""}
          </div>
          <code class="endpoint-url">${t(r.endpoint||"—")}</code>
          ${r.problem?`<div class="error small">${t(r.problem)}</div>`:""}
        </div>
        <div class="card-actions">
          ${v.includes("reset")?`<button class="btn" data-action="reset-devnet" data-key="${t(h)}" data-target="${t(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${b[n.id]?"disabled":""}>
                   ${b[n.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost" data-action="remove-endpoint" data-key="${t(h)}">Remove</button>
        </div>
      </div>
    `}function me(n){const s=n.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${t(n.id)}-port" value="${s.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${t(n.id)}-bind" value="${t(s.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${t(s.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${be(n)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${t(n.id)}">Save settings</button>
        </div>
      </div>
    `}function be(n){var A;const s=t(n.id),r=n.config.TLS??null,h=(r==null?void 0:r.Enabled)??!1,v=(r==null?void 0:r.CertSource)||"internal",y=((A=n.tls)==null?void 0:A.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${s}-tls" ${h?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${s}-tls-host" value="${t((r==null?void 0:r.Hostname)??y)}"
               placeholder="${t(y||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${y?`<p class="muted small">
               The default is <code>${t(y)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
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
      ${ge(n)}
    `}function ge(n){var A,k;const s=t(n.id),r=((A=n.config.TLS)==null?void 0:A.Enabled)??!1,h=j[n.id]??((k=n.tls)==null?void 0:k.verification)??null,v=M[n.id]??!1,y=q[n.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${s}" ${r&&!v?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${v?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${y?`<p class="error small">${t(y)}</p>`:""}
      ${h?he(h):""}
    `}function he(n){const s=(n.assertions??[]).map(r=>`
          <li class="small">
            ${g(r.status)}
            <strong>${t(r.title)}</strong>
            <div class="muted">${t(r.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${n.ok?n.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${t(n.summary)}
      </div>
      <ul class="verify-list">${s}</ul>
      <p class="muted small">
        Checked ${t(new Date(n.at).toLocaleString())} against <code>${t(n.address)}</code>
        ${n.notAfter?`· certificate valid until <code>${t(new Date(n.notAfter).toLocaleString())}</code> (${t(n.expiresIn??"")})`:""}
      </p>
      ${n.expiryWarning?`<div class="banner banner-warn">${t(n.expiryWarning)}</div>`:""}
    `}function g(n){switch(n){case"pass":return D("pass","ok");case"fail":return D("fail","bad");case"unavailable":return D("unavailable","warn");default:return D("skipped","neutral")}}async function I(n){M[n]=!0,q[n]=null,f();try{j[n]=await Dt(n)}catch(s){q[n]=`${ue(s)}${Ee(s)}`}finally{M[n]=!1,f()}}function N(n){var v,y;const s=n.tls;if(!(s!=null&&s.enabled))return"";const r=[];s.fallback&&r.push(`<div class="banner banner-warn">${t(s.fallback)}</div>`),s.error?r.push(`<div class="banner banner-warn">HTTPS front: ${t(s.error)}</div>`):((v=s.status)==null?void 0:v.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${t(s.containerName??"")}</code>) is
         ${t(((y=s.status)==null?void 0:y.State)??"unknown")}, so nothing is answering on
         <code>${t(s.url??"")}</code> even if the gateway itself is up.</div>`);const h=j[n.id]??s.verification??null;return h&&(!h.ok||!h.subscriptionsOk)&&r.push(`<div class="banner ${h.ok?"banner-warn":"banner-bad"}">${t(h.summary)}
         <div class="small">Checked ${t(new Date(h.at).toLocaleString())} — open Settings for the full check.</div></div>`),h!=null&&h.expiryWarning&&r.push(`<div class="banner banner-warn">${t(h.expiryWarning)}</div>`),s.rootCaPath&&s.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${t(s.rootCaPath)}</code> (on ${t(n.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function R(n){return{...n.config,Networks:(n.config.Networks??[]).map(s=>({ChainID:s.ChainID,Upstreams:s.Upstreams.map(r=>({...r}))}))}}async function _(n,s,r){x[n]=null;try{await Mt(n,s)}catch(h){return x[n]=`${r?r+": ":""}${ue(h)}`,f(),!1}return await C(),!0}async function d(n,s){const r=s.dataset.gid??"";switch(n){case"refresh":await C();return;case"copy":s.dataset.copy&&await dt(s,s.dataset.copy);return;case"select-chain":u[r]=Number.parseInt(s.dataset.chain??"",10),f();return;case"toggle-settings":B[r]=!B[r],f();return;case"save-settings":await m(r);return;case"verify-tls":await I(r);return;case"gw-start":case"gw-stop":case"gw-restart":await i(r,n.slice(3));return;case"gw-create":case"gw-recreate":await p(r);return;case"gw-wipe":rt(r);return;case"add-gateway":ct();return;case"forget-gateway":await S(r);return;case"add-chain":F(r);return;case"remove-chain":await ke(r,Number.parseInt(s.dataset.chain??"",10));return;case"add-endpoint":Be(r,Number.parseInt(s.dataset.chain??"",10));return;case"remove-endpoint":await Ne(s.dataset.key??"");return;case"reset-devnet":await st(s.dataset.key??"",s.dataset.target??"");return;default:return}}async function m(n){const s=$(n);if(!s)return;const r=R(s),h=a.querySelector(`#gw-${CSS.escape(n)}-port`),v=a.querySelector(`#gw-${CSS.escape(n)}-bind`);if(h){const A=Number.parseInt(h.value.trim(),10);Number.isFinite(A)&&(r.Port=A)}v&&(r.BindAddr=v.value.trim()),r.TLS=T(n,s);const y=s.status.State==="running";await _(n,r,"Saving settings")&&(B[n]=!1,y&&(x[n]=null,U(n,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),f())}function T(n,s){var y,A,k,G,se,Ce,Ge;const r=ut=>a.querySelector(`#gw-${CSS.escape(n)}-${ut}`),h=r("tls");if(!h)return s.config.TLS??null;const v=Number.parseInt(((y=r("tls-port"))==null?void 0:y.value.trim())??"",10);return{Enabled:h.checked,Hostname:((A=r("tls-host"))==null?void 0:A.value.trim())??"",CertSource:((k=r("tls-source"))==null?void 0:k.value)??"internal",CertFile:((G=r("tls-cert"))==null?void 0:G.value.trim())??"",KeyFile:((se=r("tls-key"))==null?void 0:se.value.trim())??"",HTTPSPort:Number.isFinite(v)?v:443,BindAddr:((Ce=s.config.TLS)==null?void 0:Ce.BindAddr)??"",ImageRef:((Ge=s.config.TLS)==null?void 0:Ge.ImageRef)??""}}function U(n,s){w[n]=[s]}async function i(n,s){if(!b[n]){b[n]=s,x[n]=null,f();try{await Ot(n,s)}catch(r){x[n]=`${s} failed: ${ue(r)}${Ee(r)}`}b[n]=null,await C()}}async function p(n){if(b[n])return;b[n]="create",x[n]=null,w[n]=["starting…"],f();let s;try{s=await Ft(n)}catch(r){x[n]=`${ue(r)}${Ee(r)}`,w[n]=[],b[n]=null,f();return}P==null||P(),P=Ke(s.targetId,r=>{if(c)return;const h=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(w[n]=[...(w[n]??[]).filter(y=>y!=="starting…"),h],!!r.err||r.stepId===nn&&!!r.done){P==null||P(),P=null,b[n]=null,r.err&&(x[n]="Provisioning failed — see the log below."),C();return}f()})}async function S(n){const s=$(n);if(!(!s||!await Re({title:`Forget ${s.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${s.containerName}" on ${s.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await Ut(n)}catch(h){x[n]=ue(h),f();return}await C()}}function F(n){const s=$(n);if(!s)return;const r=new Set((s.networks??[]).map(k=>k.chainId)),h=(o==null?void 0:o.presets)??[],v=h.filter(k=>!r.has(k.chainId)),y=h.filter(k=>r.has(k.chainId)),A=((o==null?void 0:o.targets)??[]).some(k=>k.id===s.placement.targetId&&k.hasDevnet);Q(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${t(s.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${v.map(k=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${k.chainId}">
                <span>${t(k.name)}</span>
                <span class="muted small">chain ${k.chainId}${k.devnet?A?" · uses the devnet on "+t(s.placement.targetId):" · will create a devnet on "+t(s.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${y.length?`<p class="muted small">Already fronted: ${t(y.map(k=>k.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,k=>{if(k==="cancel"){z();return}if(k==="custom"){K(n);return}if(k.startsWith("preset:")){const G=Number.parseInt(k.slice(7),10),se=h.find(Ce=>Ce.chainId===G);z(),se!=null&&se.devnet?J(n,G,A):L(n,G)}})}function K(n){var s;Q(`
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
      `,r=>{if(r==="cancel"){z();return}if(r!=="add")return;const h=document.getElementById("custom-chain-id"),v=document.getElementById("custom-chain-err"),y=Number.parseInt((h==null?void 0:h.value.trim())??"",10);if(!Number.isFinite(y)||y<=0){v&&(v.className="error small"),v&&(v.textContent="A chain id is a positive whole number.");return}z(),L(n,y)}),(s=document.getElementById("custom-chain-id"))==null||s.focus()}async function L(n,s){const r=$(n);if(!r)return;const h=R(r),v=h.Networks??[];v.some(y=>y.ChainID===s)||(v.push({ChainID:s,Upstreams:[]}),h.Networks=v,u[n]=s,await V(n,h)&&(u[n]=s,f(),Be(n,s)))}async function V(n,s){var y;const r={...s,Networks:(s.Networks??[]).filter(A=>A.Upstreams.length>0)};if(!await _(n,r))return!1;const v=$(n);if(v)for(const A of s.Networks??[])A.Upstreams.length===0&&!(v.networks??[]).some(k=>k.chainId===A.ChainID)&&(v.config.Networks=[...v.config.Networks??[],{ChainID:A.ChainID,Upstreams:[]}],v.networks=[...v.networks??[],{chainId:A.ChainID,name:((y=((o==null?void 0:o.presets)??[]).find(k=>k.chainId===A.ChainID))==null?void 0:y.name)??`Chain ${A.ChainID}`,path:`/${v.config.ProjectID}/evm/${A.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function J(n,s,r){const h=$(n);if(!h)return;if(!r){Q(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${t(h.placement.targetId)}</code>, so adding chain ${s} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(h.placement.targetId)}" data-modal-action="go">Create a devnet on ${t(h.placement.targetId)}</a>
          </div>
        `,()=>z());return}const v=R(h),y=v.Networks??[],A={ID:"devnet",Kind:"managed-devnet",TargetID:h.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},k=y.find(G=>G.ChainID===s);k?k.Upstreams.push(A):y.push({ChainID:s,Upstreams:[A]}),v.Networks=y,u[n]=s,await _(n,v,"Adding the devnet")}async function ke(n,s){const r=$(n);if(!r||!Number.isFinite(s))return;const h=l(r,s);if(!await Re({title:`Remove ${(h==null?void 0:h.name)??`chain ${s}`}`,body:`This gateway will stop serving ${(h==null?void 0:h.path)??`chain ${s}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const y=R(r);y.Networks=(y.Networks??[]).filter(A=>A.ChainID!==s),u[n]=null,await _(n,y,"Removing the network")}function Pe(n){const s=n.split("|");return s.length!==3?null:{gid:s[0],chainId:Number.parseInt(s[1],10),upstreamId:s[2]}}async function Ne(n){const s=Pe(n);if(!s)return;const r=$(s.gid);if(!r)return;const h=R(r),v=(h.Networks??[]).find(k=>k.ChainID===s.chainId);if(!v)return;const y=v.Upstreams.findIndex((k,G)=>(k.ID||`${s.chainId}-${G}`)===s.upstreamId);y<0||!await Re({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(v.Upstreams.splice(y,1),await _(s.gid,h,"Removing the endpoint"))}function Be(n,s){const r=$(n);if(!r||!Number.isFinite(s))return;const h=((o==null?void 0:o.sources)??[]).filter(k=>k.chainId===s),v=l(r,s),y=new Set(((v==null?void 0:v.upstreams)??[]).filter(k=>k.kind!=="external").map(k=>`${k.kind}|${k.targetId??""}`)),A=h.filter(k=>!y.has(`${k.kind}|${k.targetId}`));Q(`
        <h2>Add an endpoint for ${t((v==null?void 0:v.name)??`chain ${s}`)}</h2>
        ${A.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${A.map(k=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${t(k.kind)}:${t(k.targetId)}">
                       <span>${t(k.label)}</span>
                       <span class="muted small">${t(k.endpoint)}</span>
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
      `,k=>{if(k==="cancel"){z();return}if(k==="discover"){Te(n,s);return}if(k==="manual"){at(n,s);return}if(k.startsWith("source:")){const[,G,se]=k.split(":");z(),Ae(n,s,G,se)}})}async function Ae(n,s,r,h){const v=$(n);if(!v)return;const y=R(v),A=y.Networks??[],k={ID:`${r==="managed-devnet"?"devnet":"node"}-${h}`,Kind:r,TargetID:h,Endpoint:"",Local:!0,RecentOnly:!1},G=A.find(se=>se.ChainID===s);G?G.Upstreams.push(k):A.push({ChainID:s,Upstreams:[k]}),y.Networks=A,await _(n,y,"Adding the endpoint")}async function Te(n,s){Q(`
        <h2>Public endpoints for chain ${s}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,y=>{y==="cancel"&&z()});let r;try{r=await qt(s)}catch(y){const A=je();if(A){const k=document.createElement("p");k.className="error small",k.textContent=`Could not discover endpoints: ${ue(y)}`,A.appendChild(k)}return}if(c)return;const h=(r.endpoints??[]).filter(y=>y.status==="live"||y.status==="unprobed"),v=(r.endpoints??[]).filter(y=>y.status==="rejected");Q(`
        <h2>Public endpoints for chain ${s}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${t(r.fetchError)}</div>`:""}</div>`:""}
        ${h.length?`<p class="muted small">${h.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${h.map(y=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(y.url)}">
                       <span><code>${t(y.url)}</code></span>
                       <span class="muted small">${y.status==="live"?`answered in ${y.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${s} right now.</p>`}
        ${v.length?`<details class="rpc-rejected">
                 <summary class="muted small">${v.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${v.map(y=>`<li class="muted small"><code>${t(y.url)}</code> — ${t(y.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,y=>{if(y==="cancel"){z();return}y.startsWith("add:")&&(z(),Je(n,s,decodeURIComponent(y.slice(4))))})}function at(n,s){var r;Q(`
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
      `,h=>{if(h==="cancel"){z();return}if(h!=="add")return;const v=document.getElementById("manual-endpoint"),y=document.getElementById("manual-recent"),A=document.getElementById("manual-err"),k=(v==null?void 0:v.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(k)){A&&(A.className="error small",A.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}z(),Je(n,s,k,(y==null?void 0:y.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Je(n,s,r,h=!1){const v=$(n);if(!v)return;const y=R(v),A=y.Networks??[],k=A.find(Ce=>Ce.ChainID===s),G=((k==null?void 0:k.Upstreams.length)??0)+1,se={ID:`public-${s}-${G}`,Kind:"external",Endpoint:r,Local:!1,RecentOnly:h};k?k.Upstreams.push(se):A.push({ChainID:s,Upstreams:[se]}),y.Networks=A,await _(n,y,"Adding the endpoint")}async function st(n,s){const r=Pe(n);if(!r||!s||!await Re({title:"Reset this devnet",body:`The chain on ${s} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;b[r.gid]="reset",x[r.gid]=null,f();let v;try{v=await Nt(s)}catch(y){x[r.gid]=`Reset failed: ${ue(y)}${Ee(y)}`,b[r.gid]=null,f();return}b[r.gid]=null,ot(s,v),await C()}function ot(n,s){const r=[];r.push(s.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),s.report.Recreated&&r.push("A fresh chain was started from genesis.");const h=s.report.Cascaded??[],v=s.report.CascadeSkipped??[];Q(`
        <h2>Devnet on ${t(n)} reset</h2>
        <ul class="plain-list">${r.map(y=>`<li>${t(y)}</li>`).join("")}</ul>
        ${h.length?`<p class="ok">Restarted in front of it: ${t(h.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${v.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(v.join(", "))}.</p>`:""}
        ${s.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(s.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>z())}function rt(n){const s=$(n);if(!s)return;Q(`
        <h2>Wipe ${t(s.label)}</h2>
        <p class="error">This destroys ${t(s.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${t(n)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(n)}</button>
        </div>
      `,v=>{if(v==="cancel"||v==="close"){z(),C();return}v==="confirm"&&it(n)});const r=document.getElementById("wipe-confirm-input"),h=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{h&&(h.disabled=r.value.trim()!==n)}),r==null||r.focus()}async function it(n){const s=document.getElementById("wipe-confirm-btn");s&&(s.disabled=!0,s.textContent="Wiping…");let r;try{r=await jt(n)}catch(h){const v=je();if(v){const y=document.createElement("p");y.className="error small",y.textContent=`Wipe failed: ${ue(h)}${Ee(h)}`,v.appendChild(y)}s&&(s.disabled=!1,s.textContent=`Wipe ${n}`);return}Q(`
        <h2>${t(n)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${t(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{z(),C()})}function ct(){var h;const n=(o==null?void 0:o.targets)??[],s=new Set(((o==null?void 0:o.gateways)??[]).map(v=>v.id));if(n.length===0){Q(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>z());return}const r=s.has("default")?"":"default";Q(`
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
            ${n.map(v=>`<option value="${t(v.id)}">${t(v.id)} (${t(v.mode)})</option>`).join("")}
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
      `,v=>{if(v==="cancel"){z();return}v==="create"&&lt()}),(h=document.getElementById("new-gw-id"))==null||h.focus()}async function lt(){const n=document.getElementById("new-gw-id"),s=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),h=document.getElementById("new-gw-err"),v=(n==null?void 0:n.value.trim())??"",y=(s==null?void 0:s.value)??"",A=Number.parseInt((r==null?void 0:r.value.trim())??"",10),k=G=>{h&&(h.className="error small",h.textContent=G)};if(!v){k("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!y){k("Pick the machine it runs on.");return}try{await Ht({id:v,placement:{targetId:y,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(A)?A:4e3,Networks:[]}})}catch(G){k(ue(G));return}z(),await C()}async function dt(n,s){const r=await Le(s),h=n.textContent;n.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{c||(n.textContent=h)},1500)}function ue(n){return n instanceof Error?n.message:String(n)}function Ee(n){return n instanceof ve&&n.hint?` — ${n.hint}`:""}return()=>{c=!0,P==null||P(),z()}}const on="run",rn={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},cn={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function ln(a,c){let o=!1,e=null,u=null;const b={devnet:null},x={devnet:null},w={devnet:[]};let B=null;const j={devnet:!1};let M=null;const q={devnet:null},P={devnet:null};a.innerHTML=`
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
  `;const O=a.querySelector("#services-body");ye(a,(i,p)=>{ge(i,p)}),C();async function C(){try{const i=await Et(c);if(o)return;e=i,u=null}catch(i){if(o)return;e=null,u=T(i)}l()}function $(i){return e==null?void 0:e.services.find(p=>p.id===i)}function l(){if(!o){if(u){O.innerHTML=`<p class="error">Could not read this machine's services: ${t(u)}</p>`;return}if(!e){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${f(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(E).join("")}
      </div>
    `}}function f(i){if(i.present&&i.reachable&&!i.hint)return`<p class="muted small">Docker: ${t(i.flavor)}${i.serverVersion?` ${t(i.serverVersion)}`:""} · reachable</p>`;const p=i.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${t(p)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${i.detail?`<div class="small">${t(i.detail)}</div>`:""}
        ${i.hint?`<div class="small">${t(i.hint)}</div>`:""}
      </div>
    `}function E(i){const p=i.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${t(i.label)}</h2>
          ${H(i)}
        </div>
        <p class="muted small">${t(rn[i.id]??"")}</p>

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
          ${(i.actions??[]).map(S=>ae(i,S)).join("")}
        </div>
        ${x[i.id]?`<p class="error small">${t(x[i.id])}</p>`:""}
        ${le(i)}

        ${de(i)}
      </div>
    `}function H(i){switch(i.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function Y(i){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${t(i.error??"")}</div>
        ${i.hint?`<div class="small">${t(i.hint)}</div>`:""}
      </div>
    `}function X(i){if(i.status.State!=="created-but-stopped"||i.status.ExitCode===0)return"";const p=i.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${i.status.ExitCode}${p}.</p>`}function Z(i){const p=i.endpoints??[];return p.length===0?i.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":p.map(S=>`
        <div class="endpoint-row">
          ${ie("ok")}
          <span class="muted small">${t(S.label)}</span>
          <code class="endpoint-url">${t(S.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(S.url)}">Copy</button>
        </div>`).join("")}function ae(i,p){const S=cn[p];if(!S)return"";const F=b[i.id],K=p==="create"?`Create ${i.id==="devnet"?"devnet":"gateway"}`:S.label;return`
      <button class="${S.className}" data-action="svc-${p}" data-svc="${t(i.id)}"
              title="${t(S.title)}" ${F?"disabled":""}>
        ${F===p?'<span class="spinner" aria-label="working"></span>':t(K)}
      </button>
    `}function le(i){const p=w[i.id]??[];return p.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${t(p.join(`
`))}</pre>
      </div>
    `}function de(i){const p=j[i.id],S=pe(i);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${i.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${t(i.id)}">
            ${p?"Close":"Edit"}
          </button>
        </div>
        ${p?oe():`<p class="small">${S}</p>`}
        ${q[i.id]?`<p class="error small">${t(q[i.id])}</p>`:""}
        ${P[i.id]?`<p class="muted small">${t(P[i.id])}</p>`:""}
      </div>
    `}function pe(i){const p=i.devnet;return p?`Chain ${p.ChainID} · a block every ${t(p.BlockTime)} · JSON-RPC on ${t(p.BindAddr)}:${p.HTTPPort} · WebSocket on ${t(p.BindAddr)}:${p.WSPort}`:"—"}function oe(i){return te()}function te(){const i=M;return i?`
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
    `:""}function ne(){j.devnet&&M&&(M.BlockTime=me("#dev-blocktime",M.BlockTime),M.HTTPPort=be("#dev-http",M.HTTPPort),M.WSPort=be("#dev-ws",M.WSPort),M.BindAddr=me("#dev-bind",M.BindAddr))}function me(i,p){const S=a.querySelector(i);return S?S.value.trim():p}function be(i,p){const S=a.querySelector(i);if(!S)return p;const F=Number.parseInt(S.value.trim(),10);return Number.isFinite(F)?F:p}async function ge(i,p){const S=p.dataset.svc??"";switch(i){case"refresh":await C();return;case"copy":p.dataset.copy&&await m(p,p.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await he(S,i.slice(4));return;case"svc-create":case"svc-recreate":await g(S);return;case"svc-wipe":R(S);return;case"toggle-config":I(S);return;case"save-config":await N(S);return;default:return}}async function he(i,p){if(!b[i]){b[i]=p,x[i]=null,l();try{await It(c,i,p)}catch(S){x[i]=`${p} failed: ${T(S)}${U(S)}`}b[i]=null,await C()}}async function g(i){if(!b[i]){b[i]="create",x[i]=null,w[i]=["starting…"],l();try{await Lt(c,i)}catch(p){x[i]=`${T(p)}${U(p)}`,w[i]=[],b[i]=null,l();return}B==null||B(),B=Ke(c,p=>{if(o)return;const S=p.err?`${p.stepId}: ${p.err}`:p.line?`${p.stepId}: ${p.line}`:`${p.stepId}: done`;if(w[i]=[...(w[i]??[]).filter(K=>K!=="starting…"),S],!!p.err||p.stepId===on&&!!p.done){B==null||B(),B=null,b[i]=null,p.err&&(x[i]="Provisioning failed — see the log below."),C();return}l()})}}function I(i){if(ne(),j[i]=!j[i],q[i]=null,P[i]=null,j[i]){const p=$(i);p!=null&&p.devnet&&(M={...p.devnet})}l()}async function N(i){var F;ne(),q[i]=null,P[i]=null;const p=M;if(!p)return;if(p.HTTPPort===p.WSPort){q[i]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",l();return}try{await Bt(c,i,p)}catch(K){q[i]=T(K),l();return}const S=((F=$(i))==null?void 0:F.status.State)==="running";j[i]=!1,P[i]=S?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await C()}function R(i){const p=$(i);if(!p)return;const S=(p.restartsOnWipe??[]).map(L=>{var V;return((V=$(L))==null?void 0:V.label)??L});Q(`
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
      `,L=>{if(L==="cancel"||L==="close"){z(),C();return}L==="confirm"&&_(i)});const F=document.getElementById("wipe-confirm-input"),K=document.getElementById("wipe-confirm-btn");F==null||F.addEventListener("input",()=>{K&&(K.disabled=F.value.trim()!==i)}),F==null||F.focus()}async function _(i){const p=document.getElementById("wipe-confirm-btn");p&&(p.disabled=!0,p.textContent="Wiping…");let S;try{S=await Rt(c,i)}catch(F){const K=je();if(K){const L=document.createElement("p");L.className="error small",L.textContent=`Wipe failed: ${T(F)}${U(F)}`,K.appendChild(L)}p&&(p.disabled=!1,p.textContent=`Wipe ${i}`);return}d(i,S)}function d(i,p){const S=$(i),F=J=>{var ke;return((ke=$(J))==null?void 0:ke.label)??J},K=[];K.push(p.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const J of p.report.VolumesRemoved??[])K.push(`Volume ${J} deleted.`);for(const J of p.report.VolumesAbsent??[])K.push(`Volume ${J} was already gone.`);p.report.Recreated&&K.push("Container re-created from your saved configuration.");const L=(p.report.Cascaded??[]).map(F),V=(p.report.CascadeSkipped??[]).map(F);Q(`
        <h2>${t((S==null?void 0:S.label)??i)} wiped</h2>
        <ul class="plain-list">${K.map(J=>`<li>${t(J)}</li>`).join("")}</ul>
        ${L.length?`<p class="ok">Restarted in front of it: ${t(L.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${V.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(V.join(", "))}.</p>`:""}
        ${p.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(p.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,J=>{(J==="close"||J==="cancel")&&(z(),C())})}async function m(i,p){const S=await Le(p),F=i.textContent;i.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{o||(i.textContent=F)},1500)}function T(i){return i instanceof Error?i.message:String(i)}function U(i){return i instanceof ve&&i.hint?` — ${i.hint}`:""}return()=>{o=!0,B==null||B(),z()}}const dn="local";function un(a){let c=!1,o=!1,e="",u=null;a.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ee()}
  `;const b=a.querySelector("#targets-body");ye(a,(l,f)=>{M(l,f)}),x();async function x(){try{const[l,f,E]=await Promise.all([xe(),Se(),ft()]);if(c)return;e=E.os,B(l,f)}catch(l){if(c)return;b.innerHTML=`<p class="error">Failed to load machines: ${t(String(l))}</p>`}}function w(){u&&B(u.targets,u.catalog)}function B(l,f){u={targets:l,catalog:f};const E=e==="linux",H=[...l].sort((Z,ae)=>(Z.mode==="local"?-1:0)-(ae.mode==="local"?-1:0)),Y=H.length?`<div class="card-grid">${H.map(Z=>pn(Z,f,Z.mode!=="local"||E,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=l.some(Z=>Z.mode==="local");b.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${Y}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(E,X)}
        ${o?hn():""}
      </section>
    `}function j(l,f){const E=`
      <div class="card">
        <h3>A server over SSH ${D("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${l?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${l?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,H=l?`
        <div class="card">
          <h3>This machine ${D("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${t(e)})`:""} ${D("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return f?`<div class="card-grid card-grid-wide">${E}</div>`:`<div class="card-grid card-grid-wide">${l?H+E:E+H}</div>`}async function M(l,f){var E;if(l==="add-local"){await q();return}if(l==="delete-target"){const H=f.dataset.id;if(!H||!await Re({title:"Remove machine",body:`Remove "${H}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await P(H);return}if(l==="toggle-ssh"){o=!o,$(),w(),o&&((E=a.querySelector("#ssh-host"))==null||E.focus());return}l==="add-ssh"&&await O()}async function q(){$();try{await Ve({id:dn,mode:"local"}),await x()}catch(l){C(l)}}async function P(l){try{await mt(l),await x()}catch(f){C(f)}}async function O(){const l=a.querySelector("#ssh-host"),f=a.querySelector("#ssh-user"),E=a.querySelector("#ssh-key"),H=a.querySelector("#ssh-port"),Y=a.querySelector("#ssh-id");if(!l||!f||!E||!H||!Y)return;const X=l.value.trim(),Z=f.value.trim(),ae=E.value.trim(),le=H.value.trim(),de=Y.value.trim();if($(),!X||!Z||!ae){C(new Error("host, user, and key path are required"));return}const pe=de||fn(X),oe={Host:X,User:Z,KeyPath:ae};if(le){const ne=Number.parseInt(le,10);if(!Number.isFinite(ne)||ne<=0){C(new Error("port must be a positive number"));return}oe.Port=ne}const te=a.querySelector("#ssh-submit");te&&(te.disabled=!0,te.textContent="Connecting…");try{await Ve({id:pe,mode:"ssh",ssh:oe}),o=!1,await x()}catch(ne){C(ne),te&&(te.disabled=!1,te.textContent="Add server")}}function C(l){let f=a.querySelector("#targets-error");f||(b.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),f=a.querySelector("#targets-error")),f.textContent=String(l instanceof Error?l.message:l)}function $(){var l;(l=a.querySelector("#targets-error"))==null||l.remove()}return()=>{c=!0}}function pn(a,c,o,e){const u=a.wire,b=a.mode==="local"?"this machine":"SSH",x=a.mode==="ssh"&&a.ssh?`${t(a.ssh.User)}@${t(a.ssh.Host)}`:b,w=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(a.id)}">Devnet</a>`;let B,j;if(!u&&!o)B=`${D("can't run a node","warn")} ${D(e||"not Linux","neutral")}`,j=`
      ${w}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(a.id)}">Preview setup wizard</a>
    `;else if(!u)B=D("not set up","neutral"),j=`
      <a class="btn" href="#/setup/${encodeURIComponent(a.id)}">Run setup wizard</a>
      ${w}
    `;else{const M=c.networks.find(P=>P.ChainID===u.ChainID),q=M?M.Name:`chain ${u.ChainID}`;B=`${D(q,"ok")} ${D(u.ExecID,"neutral")} ${D(u.BeaconID,"neutral")}${u.Archive?" "+D("archive","warn"):""}`,j=`
      <a class="btn" href="#/dash/${encodeURIComponent(a.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(a.id)}">Logs</a>
      ${w}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(a.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${t(a.id)}</h2>
      <p class="muted">${x}</p>
      <p>${B}</p>
      <div class="card-actions">
        ${j}
        <button class="btn btn-danger" data-action="delete-target" data-id="${t(a.id)}">Remove</button>
      </div>
    </div>
  `}function hn(){return`
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
  `}function fn(a){return a.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const _e=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],De=8545,Ue=5052,Me=30303,mn=[369,943,1],tt={369:"default",943:"practise here first"};function bn(a,c){let o=!1;const e={targetId:c,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};a.innerHTML=`<h1>Setup: ${t(c)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ee()}</div>`;const u=a.querySelector("#wizard-body"),b=a.querySelector("#wizard-footer");ye(a,(d,m)=>{be(d,m)}),ze(a,(d,m)=>{d==="exec-select"?e.execId=m:d==="beacon-select"&&(e.beaconId=m),w()}),a.addEventListener("change",d=>{const m=d.target;m instanceof HTMLInputElement&&(m.id==="data-dir-input"?(ge(),ae()):m.id==="checkpoint-toggle"?(e.checkpoint=m.checked,w()):m.id==="exec-snapshot-toggle"&&(e.execSnapshot=m.checked,w()))}),x();async function x(){try{const[d,m]=await Promise.all([Se(),xe()]);if(o)return;e.catalog=d;const T=m.find(U=>U.id===c);T!=null&&T.wire&&(e.chainId=T.wire.ChainID,e.execId=T.wire.ExecID,e.beaconId=T.wire.BeaconID,e.archive=T.wire.Archive,T.wire.ExecHTTPPort&&(e.execHTTPPort=String(T.wire.ExecHTTPPort)),T.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(T.wire.BeaconHTTPPort)),T.wire.ExecP2PPort&&(e.execP2PPort=String(T.wire.ExecP2PPort)),T.wire.RPCBindAddr&&(e.rpcBindAddr=T.wire.RPCBindAddr)),w()}catch(d){if(o)return;e.loadError=String(d instanceof Error?d.message:d),w()}}function w(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${t(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${_(e.step)}
      ${j()}
    `,B())}function B(){var m;const d=(m=e.catalog)==null?void 0:m.networks.find(T=>T.ChainID===e.chainId);b.innerHTML=d?ee(d.Name,d.LearnURL):ee()}function j(){switch(e.step){case"network":return M();case"clients":return q();case"mode":return te();case"review":return ne();case"run":return me()}}function M(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${mn.map(T=>{const U=d.networks.find(S=>S.ChainID===T);if(!U)return"";const i=e.chainId===T,p=tt[T]?D(tt[T],T===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${i?"selected":""}" data-action="pick-network" data-chain-id="${T}" type="button">
          <h3>${t(U.Name)} <span class="muted">(chain ${T})</span></h3>
          ${p}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function q(){const d=e.catalog,m=d.networks.find(i=>i.ChainID===e.chainId);if(!m)return'<p class="error">Unknown network.</p>';(e.execId===null||!m.ExecClients.includes(e.execId))&&(e.execId=m.ExecClients[0]??null),(e.beaconId===null||!m.BeaconClients.includes(e.beaconId))&&(e.beaconId=m.BeaconClients[0]??null);const T=m.ExecClients.map(i=>de(i,d)),U=m.BeaconClients.map(i=>de(i,d));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${t(m.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${Fe("exec-select",T,e.execId)}
        </label>
        ${oe(e.execId,d)}
        <label>
          Beacon client
          ${Fe("beacon-select",U,e.beaconId)}
        </label>
        ${oe(e.beaconId,d)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function P(d){return d<=0?"—":d>=1?`~${d.toFixed(1)} TB`:`~${Math.round(d*1e3)} GB`}const O=1.1,C=.5,$="Valve reth snapshot",l="rough estimate";function f(d){return d.SnapshotSizeTB}function E(d){return d.SnapshotSizeTB*C}function H(d){return`<p class="muted small">${P(f(d))} is the measured size of Valve's reth snapshot for ${t(d.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function Y(d){return{archive:f(d)*1e12*O,full:E(d)*1e12*O}}function X(d,m){if(!d)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${t(m)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${t(m)}</code>: ${t(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==m)return"";const T=Y(d),U=e.freeBytes>=T.archive,i=e.freeBytes>=T.full,p=`<p class="muted small">Free at <code>${t(m)}</code>: <strong>${we(e.freeBytes)}</strong> — archive ${U?"fits":"won't fit"} (${P(f(d))}, ${$}), full ${i?"fits":"won't fit"} (${P(E(d))}, ${l}).</p>`;let S="";return e.downgradeNote?S=`<p class="banner banner-warn">${t(e.downgradeNote)}</p>`:i||(S=`<p class="banner banner-warn">Neither full (${P(E(d))}, ${l}) nor archive (${P(f(d))}, ${$}) fits the free space here — choose a location with more room.</p>`),p+S}function Z(d,m){if(e.downgradeNote=null,!d||e.freeBytes===null)return;const T=Y(d);e.archive&&e.freeBytes<T.archive&&e.freeBytes>=T.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${m} for archive (${P(f(d))}, ${$}) — switched to Full (${P(E(d))}, ${l}). Pick a location with more room to run archive.`)}async function ae(){var T;if(e.chainId===null)return;const d=(T=e.catalog)==null?void 0:T.networks.find(U=>U.ChainID===e.chainId),m=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,w();try{const{freeBytes:U}=await bt(e.targetId,m);if(o)return;e.freeBytes=U,e.probedPath=m,Z(d,m)}catch(U){if(o)return;e.freeBytes=null,e.probedPath=m,e.diskError=String(U instanceof Error?U.message:U)}e.diskProbing=!1,w()}function le(d){return d?/^https?:\/\/.+/i.test(d)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(d,m){const T=m.clients.find(U=>U.id===d);return{value:d,label:T?`${T.id} — ${pe(T.repo)}`:d}}function pe(d){const m=d.split("/");return m.length>=4?m[3]:d}function oe(d,m){const T=d?m.clients.find(i=>i.id===d):void 0;if(!T)return"";const U=T.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${t(T.repo)}" target="_blank" rel="noopener noreferrer">${t(U)}</a></p>`}function te(){var F,K,L;const d=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",m=(F=e.catalog)==null?void 0:F.networks.find(V=>V.ChainID===e.chainId),T=((L=(K=e.catalog)==null?void 0:K.clients.find(V=>V.id===e.execId))==null?void 0:L.snapshotSupported)??!1,U=m?`${P(E(m))} (${l})`:"Smaller",i=m?`${P(f(m))} (${$})`:"Much larger",p=m?` on ${t(m.Name)}`:"",S=m?e.checkpoint?m.SyncLabel:m.GenesisSyncLabel:"";return`
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
          ${m?`<p class="sync-estimate">⏱ Estimated initial sync${p}: <strong>${t(S)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${t((m==null?void 0:m.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${t((m==null?void 0:m.CheckpointURL)??"")}" value="${t(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${t(e.checkpointUrlError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${p}</th><td class="yes">${U}</td><td class="limited">${i}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${m?H(m):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${i}${m?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${U}${m?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${t(d)})</span>
            <input id="data-dir-input" type="text" placeholder="${t(d)}" value="${t(e.dataDir)}" />
          </label>
          ${X(m,e.dataDir||d)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${t(d)}/jwt.hex" value="${t(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${De})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${De}" value="${t(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${t(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ue})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ue}" value="${t(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${t(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Me})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Me}" value="${t(e.execP2PPort)}" />
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
    `}function ne(){const m=e.catalog.networks.find(J=>J.ChainID===e.chainId),T=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,U=e.jwtPath||`${T}/jwt.hex`,i=_e.map(J=>`<li>${t(J.title)}</li>`).join(""),p=N(e.execHTTPPort,De),S=N(e.beaconHTTPPort,Ue),F=N(e.execP2PPort,Me),K=p||S||F?`<tr><th>Non-default ports</th><td>${[p?`exec HTTP ${p}`:null,S?`beacon HTTP ${S}`:null,F?`exec p2p ${F}`:null].filter(J=>J!==null).map(t).join(", ")}</td></tr>`:"",{addr:L}=he(e.rpcBindAddr),V=L?`<tr><th>RPC bind address</th><td><code>${t(L)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${t(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${t((m==null?void 0:m.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${t(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${t(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${t(T)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${t(U)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${t(e.checkpointUrl||(m==null?void 0:m.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${K}
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
    `}function me(){const m=e.catalog.networks.find(L=>L.ChainID===e.chainId),T=m==null?void 0:m.LearnURL,U=new Set(e.events.filter(L=>L.done).map(L=>L.stepId)),i=new Set(e.events.filter(L=>L.err).map(L=>L.stepId)),p=new Map;for(const L of e.events){if(!L.line)continue;const V=p.get(L.stepId)??[];V.push(L.line),p.set(L.stepId,V)}const S=_e.map(L=>{var Ae;const V=U.has(L.id),J=i.has(L.id),ke=J?D("failed","bad"):V?D("done","ok"):D("pending","neutral"),Pe=(p.get(L.id)??[]).slice(-5),Ne=(Ae=e.events.find(Te=>Te.stepId===L.id&&Te.err))==null?void 0:Ae.err,Be=L.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${T?` <a href="${t(T)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${V?"step-done":""} ${J?"step-error":""}">
          <div class="step-head">${ke} <strong>${t(L.title)}</strong></div>
          ${Be}
          ${Pe.length?`<pre class="step-log">${Pe.map(Te=>t(Te)).join(`
`)}</pre>`:""}
          ${Ne?`<p class="error small">${t(Ne)}</p>`:""}
        </li>
      `}).join(""),F=e.events.some(L=>L.err),K=_e.every(L=>U.has(L.id))||e.events.some(L=>L.stepId==="handshake"&&L.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${S}</ol>
        ${K&&!F?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        ${F?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(d,m){switch(d){case"pick-network":e.chainId=Number(m.dataset.chainId),e.execId=null,e.beaconId=null,w();break;case"goto-network":e.step="network",w();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",w();break;case"goto-mode":e.step="mode",w(),ae();break;case"goto-review":if(ge(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){w();break}e.step="review",w();break;case"start-setup":R();break}}function ge(){const d=a.querySelectorAll('input[name="mode"]');for(const L of Array.from(d))L.checked&&(e.archive=L.value==="archive");const m=a.querySelector("#data-dir-input"),T=a.querySelector("#jwt-path-input");m&&(e.dataDir=m.value.trim()),T&&(e.jwtPath=T.value.trim());const U=a.querySelector("#exec-http-port-input"),i=a.querySelector("#beacon-http-port-input"),p=a.querySelector("#exec-p2p-port-input");U&&(e.execHTTPPort=U.value.trim()),i&&(e.beaconHTTPPort=i.value.trim()),p&&(e.execP2PPort=p.value.trim());const S=a.querySelector("#rpc-bind-addr-input");S&&(e.rpcBindAddr=S.value.trim());const F=a.querySelector("#checkpoint-url-input");F&&(e.checkpointUrl=F.value.trim());const K=a.querySelector("#snapshot-key-input");K&&(e.snapshotKey=K.value.trim()),e.execHTTPPortError=I(e.execHTTPPort).error??null,e.beaconHTTPPortError=I(e.beaconHTTPPort).error??null,e.execP2PPortError=I(e.execP2PPort).error??null,e.rpcBindAddrError=he(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?le(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function he(d){if(!d)return{};const m=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(d);return m?m.slice(1).every(T=>Number(T)<=255)?{addr:d}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(d)&&d.includes(":")?{addr:d}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const g=/^\d+$/;function I(d){if(!d)return{};if(!g.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const m=Number(d);return!Number.isInteger(m)||m<1||m>65535?{error:"Port must be between 1 and 65535."}:{port:m}}function N(d,m){const{port:T}=I(d);if(!(T===void 0||T===m))return T}async function R(){var p;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(p=e.streamStop)==null||p.call(e),e.streamStop=null,w();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const m=N(e.execHTTPPort,De),T=N(e.beaconHTTPPort,Ue),U=N(e.execP2PPort,Me);m!==void 0&&(d.ExecHTTPPort=m),T!==void 0&&(d.BeaconHTTPPort=T),U!==void 0&&(d.ExecP2PPort=U);const{addr:i}=he(e.rpcBindAddr);i!==void 0&&(d.RPCBindAddr=i),e.checkpoint?e.checkpointUrl&&(d.CheckpointURL=e.checkpointUrl):d.NoCheckpoint=!0,e.execSnapshot&&(d.ExecSnapshot=!0,d.SnapshotKey=e.snapshotKey);try{await gt(e.targetId,d)}catch(S){if(!(S instanceof ve&&S.status===409)){e.starting=!1,e.startError=String(S instanceof Error?S.message:S),w();return}}e.starting=!1,e.step="run",w(),e.streamStop=Ke(e.targetId,S=>{o||(e.events.push(S),e.step==="run"&&w())})}function _(d){const m=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],U=m.map(i=>i.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${m.map((i,p)=>`<li class="${p===U?"current":p<U?"past":"future"}">${t(i.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;o=!0,(d=e.streamStop)==null||d.call(e)}}const gn=document.querySelector("#app"),{contentEl:vn,setActiveNav:yn}=Kt(gn);let re=null;function $n(){const c=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(c.length===0)return{screen:"targets"};const[o,e]=c;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function fe(a){const c=document.createElement("div");return vn.replaceChildren(c),a(c)}function nt(){if(re){try{re()}catch{}re=null}const{screen:a,id:c}=$n();switch(yn(a),a){case"setup":if(!c){location.hash="#/targets";return}re=fe(o=>bn(o,c));break;case"dash":if(!c){location.hash="#/targets";return}re=fe(o=>Vt(o,c));break;case"logs":if(!c){location.hash="#/targets";return}re=fe(o=>Yt(o,c));break;case"security":if(!c){location.hash="#/targets";return}re=fe(o=>Xt(o,c));break;case"diag":if(!c){location.hash="#/targets";return}re=fe(o=>Zt(o,c));break;case"services":if(!c){location.hash="#/targets";return}re=fe(o=>ln(o,c));break;case"rpc":re=fe(o=>sn(o));break;case"settings":re=fe(o=>en(o));break;case"targets":default:re=fe(o=>un(o));break}}window.addEventListener("hashchange",nt);nt();
