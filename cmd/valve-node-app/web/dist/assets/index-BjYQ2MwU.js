var et=Object.defineProperty;var tt=(n,i,s)=>i in n?et(n,i,{enumerable:!0,configurable:!0,writable:!0,value:s}):n[i]=s;var Ne=(n,i,s)=>tt(n,typeof i!="symbol"?i+"":i,s);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))e(u);new MutationObserver(u=>{for(const f of u)if(f.type==="childList")for(const x of f.addedNodes)x.tagName==="LINK"&&x.rel==="modulepreload"&&e(x)}).observe(document,{childList:!0,subtree:!0});function s(u){const f={};return u.integrity&&(f.integrity=u.integrity),u.referrerPolicy&&(f.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?f.credentials="include":u.crossOrigin==="anonymous"?f.credentials="omit":f.credentials="same-origin",f}function e(u){if(u.ep)return;u.ep=!0;const f=s(u);fetch(u.href,f)}})();function nt(){return _("/api/host")}function Ce(){return _("/api/catalog")}function Te(){return _("/api/targets")}function Ke(n){return _("/api/targets",{method:"POST",headers:ce,body:JSON.stringify(n)})}function at(n){return _(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function st(n,i){return _(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(i)}`)}function ot(n,i){return _(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:ce,body:JSON.stringify(i)})}function _e(n,i){const s=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return s.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>s.close()}function rt(n,i){const s=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return s.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>s.close()}function it(n,i=200){return _(`/api/targets/${encodeURIComponent(n)}/logs?n=${i}`)}function ct(n,i){const s=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return s.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>s.close()}function Je(n,i){const s=i===void 0?{}:{lines:i};return _(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:ce,body:JSON.stringify(s)})}function lt(n,i,s){return _(`/api/targets/${encodeURIComponent(n)}/services/${i}/${s}`,{method:"POST"})}function dt(n,i){return _(`/api/targets/${encodeURIComponent(n)}/services/${i}/clear`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:i})})}function ut(n){return _(`/api/targets/${encodeURIComponent(n)}/du`)}function pt(n){return _(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function ht(n){return _(`/api/targets/${encodeURIComponent(n)}/firewall`)}function ft(n){return _(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function mt(n){return _(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function bt(n){return _(`/api/targets/${encodeURIComponent(n)}/containers`)}function gt(n,i,s){return _(`/api/targets/${encodeURIComponent(n)}/containers/${i}/${s}`,{method:"POST"})}async function vt(n,i){const s=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${i}/wipe`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:i})}),e=await s.text();let u=null;try{u=e?JSON.parse(e):null}catch{}if(u&&typeof u=="object"&&"report"in u)return u;const f=u&&typeof u=="object"&&typeof u.error=="string"?u.error:s.statusText||`HTTP ${s.status}`;throw new ve(s.status,f)}function yt(n,i){return _(`/api/targets/${encodeURIComponent(n)}/containers/${i}/provision`,{method:"POST"})}async function $t(n){const i=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:ce}),s=await i.text();let e=null;try{e=s?JSON.parse(s):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ve(i.status,u)}function wt(n,i,s){return _(`/api/targets/${encodeURIComponent(n)}/containers/${i}/config`,{method:"PUT",headers:ce,body:JSON.stringify(s)})}function kt(){return _("/api/gateways")}function Ct(n){return _("/api/gateways",{method:"POST",headers:ce,body:JSON.stringify(n)})}function Tt(n){return _(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function xt(n,i){return _(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:ce,body:JSON.stringify(i)})}function St(n,i){return _(`/api/gateways/${encodeURIComponent(n)}/${i}`,{method:"POST"})}function Pt(n){return _(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Et(n){const i=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:ce,body:JSON.stringify({Confirm:n})}),s=await i.text();let e=null;try{e=s?JSON.parse(s):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const u=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ve(i.status,u)}function It(n){return _(`/api/chainlist/${n}`)}function Rt(){return _("/api/settings")}function Lt(n){return _("/api/settings",{method:"PUT",headers:ce,body:JSON.stringify(n)})}class ve extends Error{constructor(s,e,u,f){super(e);Ne(this,"status");Ne(this,"hint");Ne(this,"code");this.name="ApiError",this.status=s,this.hint=u,this.code=f}}const ce={"Content-Type":"application/json"};async function _(n,i){const s=await fetch(n,i);if(!s.ok){let u=s.statusText||`HTTP ${s.status}`,f,x;try{const v=await s.json();v&&typeof v.error=="string"&&v.error&&(u=v.error),v&&typeof v.hint=="string"&&v.hint&&(f=v.hint),v&&typeof v.code=="string"&&v.code&&(x=v.code)}catch{}throw new ve(s.status,u,f,x)}if(s.status===204)return;const e=await s.text();return e?JSON.parse(e):void 0}const Ge="https://learn.valve.city/rpc";function t(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ne(n,i){const s=n&&i&&i!==Ge?` <span class="footer-sep">·</span> <a href="${t(i)}" target="_blank" rel="noopener noreferrer">${t(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${t(Ge)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${s}
    </footer>
  `}function Nt(n){n.innerHTML=`
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
  `;const i=n.querySelector("#content"),s=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:u=>{for(const f of s)f.classList.toggle("active",f.dataset.nav===u)}}}function $e(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function Bt(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function At(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const i=Math.round(n/60),s=Math.floor(i/60),e=i%60;if(s===0)return`~${e}m`;if(s<48)return`~${s}h ${e}m`;const u=Math.floor(s/24),f=s%24;return`~${u}d ${f}h`}function O(n,i){return`<span class="badge badge-${i}">${t(n)}</span>`}function ie(n){return`<span class="dot dot-${n}"></span>`}const Ve=["B","KB","MB","GB","TB","PB"];function we(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let i=n,s=0;for(;i>=1024&&s<Ve.length-1;)i/=1024,s++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${Ve[s]}`}async function Pe(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function ye(n,i){n.addEventListener("click",s=>{const e=s.target.closest("[data-action]");if(!e||!n.contains(e))return;const u=e.dataset.action;u&&i(u,e,s)})}function Ue(n,i,s){const e=i.find(f=>f.value===s),u=i.map(f=>`
      <li class="dropdown-option${f.value===s?" selected":""}" role="option"
          aria-selected="${f.value===s}" data-value="${t(f.value)}">
        ${t(f.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${t(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${t(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${u}</ul>
    </div>
  `}function xe(n){n.querySelectorAll(".dropdown.open").forEach(i=>{var s;i.classList.remove("open"),(s=i.querySelector(".dropdown-trigger"))==null||s.setAttribute("aria-expanded","false")})}function ze(n,i){n.addEventListener("click",u=>{const f=u.target,x=f.closest(".dropdown-trigger");if(x&&n.contains(x)){const N=x.closest(".dropdown"),H=!!N&&!N.classList.contains("open");xe(n),N&&H&&(N.classList.add("open"),x.setAttribute("aria-expanded","true"));return}const v=f.closest(".dropdown-option");if(v&&n.contains(v)){const N=v.closest(".dropdown");xe(n),i((N==null?void 0:N.dataset.dropdown)??"",v.dataset.value??"");return}xe(n)});const s=u=>{if(!n.isConnected){document.removeEventListener("click",s),document.removeEventListener("keydown",e);return}const f=u.target;(!f.closest(".dropdown")||!n.contains(f))&&xe(n)},e=u=>{if(!n.isConnected){document.removeEventListener("click",s),document.removeEventListener("keydown",e);return}u.key==="Escape"&&xe(n)};document.addEventListener("click",s),document.addEventListener("keydown",e)}const Oe="app-modal";let He=null;function te(n,i){z();const s=document.createElement("div");s.className="modal-overlay",s.id=Oe,s.innerHTML=`<div class="modal">${n}</div>`,s.addEventListener("click",u=>{const f=u.target.closest("[data-modal-action]");f!=null&&f.dataset.modalAction?i(f.dataset.modalAction):u.target===s&&i("cancel")});const e=u=>{u.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),He=e,document.body.appendChild(s)}function z(){var n;(n=document.getElementById(Oe))==null||n.remove(),He&&(document.removeEventListener("keydown",He),He=null)}function Me(){return document.querySelector(`#${Oe} .modal`)}function Se(n){return new Promise(i=>{var u;let s=!1;const e=f=>{s||(s=!0,z(),i(f))};te(`
        <h2>${t(n.title)}</h2>
        <p>${t(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${t(n.confirmLabel)}</button>
        </div>
      `,f=>e(f==="confirm")),(u=document.querySelector(`#${Oe} [data-modal-action="confirm"]`))==null||u.focus()})}const Dt=85,qe={exec:"Execution",beacon:"Beacon"};function Ht(n,i){let s=!1,e=null,u=null,f=null,x=null,v=null,N=null,H=null,U=null;const q={exec:null,beacon:null};let P=null;n.innerHTML=`<h1>Dashboard: ${t(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ne()}</div>`;const j=n.querySelector("#dash-body"),k=n.querySelector("#dash-footer");j.addEventListener("click",m=>{const I=m.target.closest("[data-action]");if(!I||!j.contains(I))return;const L=I.dataset.action;if(L==="svc-action"){const A=I.dataset.svc,K=I.dataset.kind;A&&K&&me(A,K)}else if(L==="open-clear"){const A=I.dataset.svc;A&&ge(A)}else if(L==="copy"){const A=I.dataset.copy;A&&be(I,A)}else L==="retry-du"?l():L==="retry-endpoints"&&b()}),$();async function $(){let m,I;try{const[A,K]=await Promise.all([Te(),Ce()]);m=A.find(d=>d.id===i),I=K}catch(A){if(s)return;j.innerHTML=`<p class="error">Failed to load target: ${t(String(A))}</p>`;return}if(s)return;if(!m){j.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!m.wire){j.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const L=I==null?void 0:I.networks.find(A=>A.ChainID===m.wire.ChainID);L&&(k.innerHTML=ne(L.Name,L.LearnURL)),j.innerHTML='<p class="muted">Connecting…</p>',e=rt(i,A=>{s||(E(A),u=A,f=A,B())}),l(),b()}async function l(){N=null;try{v=await ut(i)}catch(m){v=null,N=String(m instanceof Error?m.message:m)}s||B()}async function b(){U=null;try{H=await pt(i)}catch(m){H=null,U=String(m instanceof Error?m.message:m)}s||B()}function E(m){if(!u)return;const I=(new Date(m.at).getTime()-new Date(u.at).getTime())/1e3,L=m.execHead-u.execHead;if(I>0&&L>=0){const A=L/I;x=x===null?A:x*.7+A*.3}}function B(){if(!f)return;const m=f;j.innerHTML=`
      <p class="dash-status">${Z(m)}</p>
      <div class="card-grid">
        ${G(m)}
        ${Q(m)}
        ${se(m)}
        ${le(m)}
        ${de(m)}
        ${pe()}
      </div>
      <p class="muted small">Last updated ${t(new Date(m.at).toLocaleTimeString())}</p>
    `}function Z(m){return!m.execActive&&!m.beaconActive?O("Node not running","bad"):m.execSyncing||m.beaconDistance>0?O("Syncing","warn"):O("Running · synced","ok")}function ee(m){const L=m.refHead>0?m.refHead-m.execHead:null,A=L!==null&&L>0&&x&&x>0?At(L/x):L!==null&&L<=0?"caught up":"—";return{lag:L,eta:A}}function Q(m){const{lag:I,eta:L}=ee(m);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${m.execActive?m.execSyncing?O("syncing","warn"):m.execHead===0?O("no data","neutral"):O("synced","ok"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${$e(m.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${I!==null?$e(m.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${I!==null?$e(Math.max(I,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${L}</dd></div>
        </dl>
      </div>
    `}function se(m){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${m.beaconActive?m.beaconSlot===0?O("no data","neutral"):m.beaconDistance===0?O("synced","ok"):O("syncing","warn"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${$e(m.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${$e(m.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function le(m){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${$e(m.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${$e(m.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(m){const I=m.diskUsedPct>=Dt,L=`
      <div class="meter"><div class="meter-fill ${I?"meter-warn":""}" style="width:${Math.min(m.diskUsedPct,100)}%"></div></div>
      <p>${Bt(m.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="error small">${t(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!v)return`
        <div class="card ${I?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="muted">Loading…</p>
        </div>
      `;const A=v.ExpectedExecBytes>0?Math.min(v.ExecBytes/v.ExpectedExecBytes*100,100):0,K=v.ExpectedBeaconBytes>0?Math.min(v.BeaconBytes/v.ExpectedBeaconBytes*100,100):0,{lag:d,eta:h}=ee(m),C=d!==null&&d>0&&x!==null&&x>0;return`
      <div class="card ${I?"card-warn":""}">
        <h3>Storage</h3>
        ${L}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${we(v.ExecBytes)} of ~${we(v.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${A}%"></div></div>
        ${C?`<p class="muted small">Estimated time remaining: ${t(h)}</p>`:""}
        <p class="muted small">Beacon — ${we(v.BeaconBytes)} of ~${we(v.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${K}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${we(v.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${t(v.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${t(v.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function pe(){if(U)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${t(U)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!H)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const m=H,I=m.ExecReachable&&!m.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",L=m.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${t(m.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${t(m.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ie(m.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${t(m.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(m.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ie(m.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${t(m.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(m.BeaconHTTP)}">Copy</button>
        </div>
        ${I}
        ${L}
      </div>
    `}function oe(m,I){const L=qe[m],A=q[m],K=(d,h,C)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${m}" data-kind="${d}" ${A!==null||C?"disabled":""}>${A===d?V():t(h)}</button>`;return`
      <div class="service-row">
        <span>${t(L)} ${I?O("active","ok"):O("down","bad")}</span>
        <div class="service-actions">
          ${K("start","Start",I)}
          ${K("stop","Stop",!I)}
          ${K("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${m}" ${A!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function G(m){return`
      <div class="card">
        <h3>Services</h3>
        ${oe("exec",m.execActive)}
        ${oe("beacon",m.beaconActive)}
        ${P?`<p class="error small">${t(P)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function V(){return'<span class="spinner" aria-label="working"></span>'}async function me(m,I){if(q[m]===null){q[m]=I,P=null,B();try{await lt(i,m,I)}catch(L){P=`${qe[m]} ${I} failed: ${L instanceof Error?L.message:String(L)}`}q[m]=null,s||B()}}async function be(m,I){const L=await Pe(I),A=m.textContent;m.textContent=L?"Copied!":"Copy failed",setTimeout(()=>{s||(m.textContent=A)},1500)}function ge(m){const I=qe[m],L=v?we(m==="exec"?v.ExecBytes:v.BeaconBytes):"unknown (disk usage hasn't loaded)";te(`
        <h2>Clear ${t(I)} data</h2>
        <p class="error">
          This stops the ${t(I.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${t(L)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${t(m)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,d=>{if(d==="cancel"){z();return}d==="confirm"&&he(m)});const A=document.getElementById("clear-confirm-input"),K=document.getElementById("clear-confirm-btn");A==null||A.addEventListener("input",()=>{K&&(K.disabled=A.value.trim()!==m)}),A==null||A.focus()}async function he(m){const I=document.getElementById("clear-confirm-btn");I&&(I.disabled=!0,I.textContent="Clearing…");try{await dt(i,m),z(),l()}catch(L){const A=Me();if(A){const K=document.createElement("p");K.className="error small",K.textContent=`Clear failed: ${L instanceof Error?L.message:String(L)}`,A.appendChild(K)}I&&(I.disabled=!1,I.textContent="Clear and resync")}}return()=>{s=!0,e==null||e(),z()}}const Ye=500,Ze="valve-node-app.explain-consent";function Ut(n,i){let s=!1,e=null;const u=[];n.innerHTML=`
    <h1>Logs: ${t(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ne()}</div>
  `;const f=n.querySelector("#logs-body"),x=n.querySelector("#logs-footer");ye(n,$=>{$==="explain"&&U()}),v();async function v(){let $,l;try{const[E,B]=await Promise.all([Te(),Ce()]);$=E.find(Z=>Z.id===i),l=B}catch(E){if(s)return;f.innerHTML=`<p class="error">Failed to load target: ${t(String(E))}</p>`;return}if(s)return;if(!$){f.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!$.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const b=l==null?void 0:l.networks.find(E=>E.ChainID===$.wire.ChainID);b&&(x.innerHTML=ne(b.Name,b.LearnURL));try{const E=await it(i,200);if(s)return;u.push(...E)}catch(E){if(s)return;f.innerHTML=`<p class="error">Failed to load logs: ${t(String(E))}</p>`;return}N(),e=ct(i,E=>{s||(u.push(E),u.length>Ye&&u.splice(0,u.length-Ye),N())})}function N(){const $=u.filter(b=>b.severity==="error"||b.severity==="critical");f.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(H).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${O(String($.length),$.length?"bad":"neutral")}</h2>
          <div class="log-lines">${$.length?$.slice().reverse().map(H).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const l=f.querySelector(".log-lines");l&&(l.scrollTop=l.scrollHeight)}function H($){const l=$.severity||"info",b=$.learnUrl?` <a href="${t($.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${t(l)}">
        <span class="log-time">${t(new Date($.at).toLocaleTimeString())}</span>
        <span class="log-unit">${t($.unit)}</span>
        <span class="log-sev">${t(l)}</span>
        <span class="log-text">${t($.line)}</span>
        ${$.explain?`<div class="log-explain">${t($.explain)}${b}</div>`:""}
      </div>
    `}async function U(){const $=u.filter(b=>b.severity==="error"||b.severity==="critical").map(b=>b.line).slice(-40);if(!(localStorage.getItem(Ze)==="1")){q($);return}await P($)}function q($){const l=$.length?`<pre class="explain-excerpt">${$.map(b=>t(b)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';j(`
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
    `,b=>{b==="proceed"?(localStorage.setItem(Ze,"1"),k(),P($)):k()})}async function P($){j('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const l=$.length?await Je(i,$):await Je(i);if(s)return;j(`
        <h2>Explanation</h2>
        <div class="explain-text">${t(l.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${l.sentExcerpt.map(b=>t(b)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,b=>{b==="close"&&k()})}catch(l){if(s)return;if(l instanceof ve&&l.status===409){j(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,b=>{b==="close"&&k()});return}j(`
        <h2>Explain failed</h2>
        <p class="error">${t(l instanceof Error?l.message:String(l))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,b=>{b==="close"&&k()})}}function j($,l){k();const b=document.createElement("div");b.className="modal-overlay",b.id="explain-modal",b.innerHTML=`<div class="modal">${$}</div>`,b.addEventListener("click",E=>{const B=E.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&l(B.dataset.modalAction),E.target===b&&l("cancel")}),document.body.appendChild(b)}function k(){var $;($=document.getElementById("explain-modal"))==null||$.remove()}return()=>{s=!0,e==null||e(),k()}}function Mt(n,i){let s=!1,e=null,u=null,f=!1,x=!1;n.innerHTML=`<h1>Network diagnostics: ${t(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ne()}</div>`;const v=n.querySelector("#diag-body"),N=n.querySelector("#diag-footer");ye(n,(l,b)=>{var E;if(l==="run")U();else if(l==="toggle")(E=b.closest(".check-item"))==null||E.classList.toggle("expanded");else if(l==="copy"){const B=b.dataset.copy;B&&$(b,B)}}),H();async function H(){let l,b;try{const[B,Z]=await Promise.all([Te(),Ce()]);l=B.find(ee=>ee.id===i),b=Z}catch(B){if(s)return;v.innerHTML=`<p class="error">Failed to load target: ${t(String(B))}</p>`;return}if(s)return;if(!l){v.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const E=b==null?void 0:b.networks.find(B=>B.ChainID===l.wire.ChainID);E&&(N.innerHTML=ne(E.Name,E.LearnURL));try{e=await mt(i),x=!0}catch(B){u=String(B instanceof Error?B.message:B)}s||q()}async function U(){f=!0,u=null,q();try{e=await ft(i),x=!0}catch(l){u=String(l instanceof Error?l.message:l)}f=!1,s||q()}function q(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
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
    `}function P(){if(!x&&!u)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const l=new Date(e.at).toLocaleString(),b=e.failedId?`<p><strong>Failed at: ${t(j(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${t(l)} — trigger: ${t(e.trigger)}</p>
      ${b}
      <ul class="check-list">${e.items.map(k).join("")}</ul>
    `}function j(l){var b;return((b=e==null?void 0:e.items.find(E=>E.ID===l))==null?void 0:b.Title)??l}function k(l){const b=l.Status==="pass"?"ok":l.Status==="fail"?"bad":l.Status==="warn"?"warn":"neutral",E=l.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${E?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${O(E?"failed here":l.Status,b)}
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
    `}async function $(l,b){const E=await Pe(b),B=l.textContent;l.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{s||(l.textContent=B)},1500)}return()=>{s=!0}}function Ot(n,i){let s=!1,e=[],u=null,f=!1,x=!1;n.innerHTML=`<h1>Security: ${t(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ne()}</div>`;const v=n.querySelector("#sec-body"),N=n.querySelector("#sec-footer");ye(n,(k,$)=>{var l;if(k==="rerun")U();else if(k==="toggle")(l=$.closest(".check-item"))==null||l.classList.toggle("expanded");else if(k==="copy"){const b=$.dataset.copy;b&&j($,b)}}),H();async function H(){let k,$;try{const[b,E]=await Promise.all([Te(),Ce()]);k=b.find(B=>B.id===i),$=E}catch(b){if(s)return;v.innerHTML=`<p class="error">Failed to load target: ${t(String(b))}</p>`;return}if(s)return;if(!k){v.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!k.wire){v.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const l=$==null?void 0:$.networks.find(b=>b.ChainID===k.wire.ChainID);l&&(N.innerHTML=ne(l.Name,l.LearnURL)),await U()}async function U(){f=!0,u=null,q();try{e=await ht(i),x=!0}catch(k){u=String(k instanceof Error?k.message:k)}f=!1,s||q()}function q(){v.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${f?"disabled":""}>${f?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${t(u)}</p>`:""}
      ${!x&&f?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(P).join("")}</ul>`:x?'<p class="muted">No checks returned.</p>':""}
    `}function P(k){const $=k.Status==="pass"?"ok":k.Status==="fail"?"bad":k.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${O(k.Status,$)}
          <strong>${t(k.Title)}</strong>
          <span class="muted small check-detail-inline">${t(k.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${t(k.Why)}</p>
          </details>
          ${k.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${t(k.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${t(k.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function j(k,$){const l=await Pe($),b=k.textContent;k.textContent=l?"Copied!":"Copy failed",setTimeout(()=>{s||(k.textContent=b)},1500)}return()=>{s=!0}}const jt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Ft(n){let i=!1,s=!1,e=!1,u=null,f=!1,x=null,v=null;n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ne()}`;const N=n.querySelector("#settings-body");ye(n,P=>{if(P==="save"&&q(),P==="clear-key"){if(!x)return;s=!0;const j=n.querySelector("#ai-key");j&&(j.value=""),U(x)}}),ze(n,(P,j)=>{P!=="ai-provider"||!x||(v=j,f=!1,U(x))}),H();async function H(){try{const P=await Rt();if(i)return;x=P,U(P)}catch(P){if(i)return;N.innerHTML=`<p class="error">Failed to load settings: ${t(String(P))}</p>`}}function U(P){var $;const j=v??P.aiProvider;N.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ue("ai-provider",jt.map(l=>({value:l.value,label:l.label})),j)}
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
    `;const k=n.querySelector("#ai-key");k==null||k.addEventListener("input",()=>{s=!0,f=!1}),($=n.querySelector("#ref-rpc-base"))==null||$.addEventListener("input",()=>{f=!1})}async function q(){const P=n.querySelector("#ai-key"),j=n.querySelector("#ref-rpc-base");if(!P||!j||!x)return;const k={aiProvider:v??x.aiProvider,refRpcBase:j.value.trim()};s&&(k.aiKey=P.value),e=!0,u=null,f=!1,U(x);try{const $=await Lt(k);if(i)return;x=$,s=!1,e=!1,f=!0,U($)}catch($){if(i)return;e=!1,u=String($ instanceof Error?$.message:$),U(x)}}return()=>{i=!0}}const qt=6,Wt="run",_t={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function zt(n){let i=!1,s=null,e=null;const u={},f={},x={},v={},N={};let H=null;n.innerHTML=`
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
  `;const U=n.querySelector("#rpc-body");ye(n,(a,o)=>{me(a,o)}),ze(n,(a,o)=>{if(a.startsWith("chain-")){const c=a.slice(6);u[c]=Number.parseInt(o,10),k()}}),q();async function q(){try{const a=await kt();if(i)return;s=a,e=null;for(const o of a.gateways??[]){const c=o.networks??[],g=u[o.id];(g==null||!c.some(y=>y.chainId===g))&&(u[o.id]=c.length?c[0].chainId:null)}}catch(a){if(i)return;s=null,e=ae(a)}k()}function P(a){return((s==null?void 0:s.gateways)??[]).find(o=>o.id===a)}function j(a,o){if(o!=null)return(a.networks??[]).find(c=>c.chainId===o)}function k(){if(i)return;if(e){U.innerHTML=`<p class="error">Could not read the gateways: ${t(e)}</p>`;return}if(!s){U.innerHTML='<p class="muted">Loading…</p>';return}const a=s.gateways??[];U.innerHTML=`
      ${a.map(l).join("")}
      ${a.length===0?$():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${a.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function $(){return((s==null?void 0:s.targets)??[]).length===0?`
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
    `}function l(a){const o=j(a,u[a.id]??null);return`
      <section class="rpc-gateway">
        ${b(a)}
        ${a.error?Z(a):""}
        ${a.blocked?`<div class="banner banner-warn">${t(a.blocked)}</div>`:""}
        ${(a.warnings??[]).map(c=>`<div class="banner banner-warn">${t(c)}</div>`).join("")}
        ${x[a.id]?`<p class="error small">${t(x[a.id])}</p>`:""}
        ${Q(a)}
        ${N[a.id]?oe(a):""}
        ${de(a,o)}
      </section>
    `}function b(a){var c;const o=a.status.State==="running";return`
      <div class="rpc-bar${o?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${B(a)}
            <strong>${t(a.label)}</strong>
            ${E(a)}
            <span class="muted small">on ${t(a.placement.targetId)} · ${t(a.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(a.actions??[]).map(g=>ee(a,g)).join("")}
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${t(a.id)}">
              ${N[a.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${t(a.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${o?`<code class="endpoint-url">${t(a.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(a.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${t(((c=(a.networks??[])[0])==null?void 0:c.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${t(a.baseUrl)}</code> once it is running.</span>`}
        </div>
        ${se(a)}
      </div>
    `}function E(a){switch(a.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function B(a){return a.status.State==="running"?ie("ok"):a.status.State==="unknown"?ie("bad"):ie("neutral")}function Z(a){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${t(a.error??"")}</div>
        ${a.hint?`<div class="small">${t(a.hint)}</div>`:""}
      </div>
    `}function ee(a,o){const c=_t[o];if(!c)return"";const g=f[a.id];return`
      <button class="${c.className}" data-action="gw-${o}" data-gid="${t(a.id)}"
              title="${t(c.title)}" ${g?"disabled":""}>
        ${g===o?'<span class="spinner" aria-label="working"></span>':t(c.label)}
      </button>
    `}function Q(a){const o=v[a.id]??[];return o.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${t(a.placement.targetId)}</p>
        <pre class="step-log">${t(o.join(`
`))}</pre>
      </div>
    `}function se(a){const o=a.networks??[],c=u[a.id]??null,g=`
      <button class="chip chip-add" data-action="add-chain" data-gid="${t(a.id)}"
              title="Add a network for this gateway to front">+ Network</button>
    `;if(o.length===0)return`
        <div class="rpc-chiprow">
          <span class="muted small">No networks yet — eRPC refuses a configuration with none, so add one before creating the gateway.</span>
          ${g}
        </div>
      `;if(o.length>qt){const y=o.map(w=>({value:String(w.chainId),label:`${w.name} (${w.chainId})${w.serviceable?"":" — no working endpoint"}`}));return`
        <div class="rpc-chiprow">
          <span class="muted small">Fronting ${o.length} networks</span>
          ${Ue(`chain-${a.id}`,y,c==null?null:String(c))}
          ${g}
        </div>
      `}return`
      <div class="rpc-chiprow">
        ${o.map(y=>le(a,y,y.chainId===c)).join("")}
        ${g}
      </div>
    `}function le(a,o,c){const g=!o.serviceable;return`
      <button class="chip card-selectable${c?" selected":""}${g?" chip-bad":""}"
              data-action="select-chain" data-gid="${t(a.id)}" data-chain="${o.chainId}"
              title="${t(g?`${o.name}: no endpoint on this chain can be used right now`:`${o.name} · ${o.path}`)}">
        <span class="chip-dot">${ie(g?"bad":"ok")}</span>
        <span class="chip-name">${t(o.name)}</span>
        <span class="chip-id">${o.chainId}</span>
      </button>
    `}function de(a,o){if(!o)return'<div class="card rpc-upstreams"><p class="muted small">Pick a network above to see the servers behind it.</p></div>';const c=o.upstreams??[];return`
      <div class="card rpc-upstreams">
        <div class="service-head">
          <h2>${t(o.name)} <span class="muted">· chain ${o.chainId}</span></h2>
          <div class="card-actions">
            <button class="btn" data-action="add-endpoint" data-gid="${t(a.id)}" data-chain="${o.chainId}">Add an endpoint</button>
            <button class="btn btn-ghost" data-action="remove-chain" data-gid="${t(a.id)}" data-chain="${o.chainId}">Remove network</button>
          </div>
        </div>
        ${o.url?`<div class="endpoint-row">${ie("ok")}<span class="muted small">callers dial</span>
                 <code class="endpoint-url">${t(o.url)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(o.url)}">Copy</button></div>`:`<p class="muted small">Path <code>${t(o.path)}</code> — the full URL appears once the gateway is running.</p>`}
        ${(o.warnings??[]).map(g=>`<div class="banner banner-warn">${t(g)}</div>`).join("")}
        ${c.map(g=>pe(a,o,g)).join("")}
        ${c.length===0?'<p class="muted small">No endpoint yet, so there is nowhere for calls on this path to go.</p>':""}
      </div>
    `}function pe(a,o,c){const g=`${a.id}|${o.chainId}|${c.id}`,y=c.actions??[];return`
      <div class="upstream-row${c.problem?" upstream-row-bad":""}">
        <span class="upstream-state">${c.problem?ie("bad"):ie("ok")}</span>
        <div class="upstream-what">
          <div class="upstream-label">
            ${t(c.label)}
            ${c.local?O("preferred","ok"):O("fallback","neutral")}
            ${c.recentOnly?O("recent blocks only","warn"):""}
          </div>
          <code class="endpoint-url">${t(c.endpoint||"—")}</code>
          ${c.problem?`<div class="error small">${t(c.problem)}</div>`:""}
        </div>
        <div class="card-actions">
          ${y.includes("reset")?`<button class="btn" data-action="reset-devnet" data-key="${t(g)}" data-target="${t(c.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[a.id]?"disabled":""}>
                   ${f[a.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost" data-action="remove-endpoint" data-key="${t(g)}">Remove</button>
        </div>
      </div>
    `}function oe(a){const o=a.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${t(a.id)}-port" value="${o.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${t(a.id)}-bind" value="${t(o.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${t(o.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${t(a.id)}">Save settings</button>
        </div>
      </div>
    `}function G(a){return{...a.config,Networks:(a.config.Networks??[]).map(o=>({ChainID:o.ChainID,Upstreams:o.Upstreams.map(c=>({...c}))}))}}async function V(a,o,c){x[a]=null;try{await xt(a,o)}catch(g){return x[a]=`${c?c+": ":""}${ae(g)}`,k(),!1}return await q(),!0}async function me(a,o){const c=o.dataset.gid??"";switch(a){case"refresh":await q();return;case"copy":o.dataset.copy&&await Re(o,o.dataset.copy);return;case"select-chain":u[c]=Number.parseInt(o.dataset.chain??"",10),k();return;case"toggle-settings":N[c]=!N[c],k();return;case"save-settings":await be(c);return;case"gw-start":case"gw-stop":case"gw-restart":await he(c,a.slice(3));return;case"gw-create":case"gw-recreate":await m(c);return;case"gw-wipe":ke(c);return;case"add-gateway":Ie();return;case"forget-gateway":await I(c);return;case"add-chain":L(c);return;case"remove-chain":await C(c,Number.parseInt(o.dataset.chain??"",10));return;case"add-endpoint":p(c,Number.parseInt(o.dataset.chain??"",10));return;case"remove-endpoint":await r(o.dataset.key??"");return;case"reset-devnet":await Y(o.dataset.key??"",o.dataset.target??"");return;default:return}}async function be(a){const o=P(a);if(!o)return;const c=G(o),g=n.querySelector(`#gw-${CSS.escape(a)}-port`),y=n.querySelector(`#gw-${CSS.escape(a)}-bind`);if(g){const M=Number.parseInt(g.value.trim(),10);Number.isFinite(M)&&(c.Port=M)}y&&(c.BindAddr=y.value.trim());const w=o.status.State==="running";await V(a,c,"Saving settings")&&(N[a]=!1,w&&(x[a]=null,ge(a,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),k())}function ge(a,o){v[a]=[o]}async function he(a,o){if(!f[a]){f[a]=o,x[a]=null,k();try{await St(a,o)}catch(c){x[a]=`${o} failed: ${ae(c)}${Le(c)}`}f[a]=null,await q()}}async function m(a){if(f[a])return;f[a]="create",x[a]=null,v[a]=["starting…"],k();let o;try{o=await Pt(a)}catch(c){x[a]=`${ae(c)}${Le(c)}`,v[a]=[],f[a]=null,k();return}H==null||H(),H=_e(o.targetId,c=>{if(i)return;const g=c.err?`${c.stepId}: ${c.err}`:c.line?`${c.stepId}: ${c.line}`:`${c.stepId}: done`;if(v[a]=[...(v[a]??[]).filter(w=>w!=="starting…"),g],!!c.err||c.stepId===Wt&&!!c.done){H==null||H(),H=null,f[a]=null,c.err&&(x[a]="Provisioning failed — see the log below."),q();return}k()})}async function I(a){const o=P(a);if(!(!o||!await Se({title:`Forget ${o.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${o.containerName}" on ${o.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await Tt(a)}catch(g){x[a]=ae(g),k();return}await q()}}function L(a){const o=P(a);if(!o)return;const c=new Set((o.networks??[]).map(S=>S.chainId)),g=(s==null?void 0:s.presets)??[],y=g.filter(S=>!c.has(S.chainId)),w=g.filter(S=>c.has(S.chainId)),M=((s==null?void 0:s.targets)??[]).some(S=>S.id===o.placement.targetId&&S.hasDevnet);te(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${t(o.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${y.map(S=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${S.chainId}">
                <span>${t(S.name)}</span>
                <span class="muted small">chain ${S.chainId}${S.devnet?M?" · uses the devnet on "+t(o.placement.targetId):" · will create a devnet on "+t(o.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${w.length?`<p class="muted small">Already fronted: ${t(w.map(S=>S.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,S=>{if(S==="cancel"){z();return}if(S==="custom"){A(a);return}if(S.startsWith("preset:")){const X=Number.parseInt(S.slice(7),10),ue=g.find(Fe=>Fe.chainId===X);z(),ue!=null&&ue.devnet?h(a,X,M):K(a,X)}})}function A(a){var o;te(`
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
      `,c=>{if(c==="cancel"){z();return}if(c!=="add")return;const g=document.getElementById("custom-chain-id"),y=document.getElementById("custom-chain-err"),w=Number.parseInt((g==null?void 0:g.value.trim())??"",10);if(!Number.isFinite(w)||w<=0){y&&(y.className="error small"),y&&(y.textContent="A chain id is a positive whole number.");return}z(),K(a,w)}),(o=document.getElementById("custom-chain-id"))==null||o.focus()}async function K(a,o){const c=P(a);if(!c)return;const g=G(c),y=g.Networks??[];y.some(w=>w.ChainID===o)||(y.push({ChainID:o,Upstreams:[]}),g.Networks=y,u[a]=o,await d(a,g)&&(u[a]=o,k(),p(a,o)))}async function d(a,o){var w;const c={...o,Networks:(o.Networks??[]).filter(M=>M.Upstreams.length>0)};if(!await V(a,c))return!1;const y=P(a);if(y)for(const M of o.Networks??[])M.Upstreams.length===0&&!(y.networks??[]).some(S=>S.chainId===M.ChainID)&&(y.config.Networks=[...y.config.Networks??[],{ChainID:M.ChainID,Upstreams:[]}],y.networks=[...y.networks??[],{chainId:M.ChainID,name:((w=((s==null?void 0:s.presets)??[]).find(S=>S.chainId===M.ChainID))==null?void 0:w.name)??`Chain ${M.ChainID}`,path:`/${y.config.ProjectID}/evm/${M.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function h(a,o,c){const g=P(a);if(!g)return;if(!c){te(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${t(g.placement.targetId)}</code>, so adding chain ${o} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(g.placement.targetId)}" data-modal-action="go">Create a devnet on ${t(g.placement.targetId)}</a>
          </div>
        `,()=>z());return}const y=G(g),w=y.Networks??[],M={ID:"devnet",Kind:"managed-devnet",TargetID:g.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},S=w.find(X=>X.ChainID===o);S?S.Upstreams.push(M):w.push({ChainID:o,Upstreams:[M]}),y.Networks=w,u[a]=o,await V(a,y,"Adding the devnet")}async function C(a,o){const c=P(a);if(!c||!Number.isFinite(o))return;const g=j(c,o);if(!await Se({title:`Remove ${(g==null?void 0:g.name)??`chain ${o}`}`,body:`This gateway will stop serving ${(g==null?void 0:g.path)??`chain ${o}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const w=G(c);w.Networks=(w.Networks??[]).filter(M=>M.ChainID!==o),u[a]=null,await V(a,w,"Removing the network")}function D(a){const o=a.split("|");return o.length!==3?null:{gid:o[0],chainId:Number.parseInt(o[1],10),upstreamId:o[2]}}async function r(a){const o=D(a);if(!o)return;const c=P(o.gid);if(!c)return;const g=G(c),y=(g.Networks??[]).find(S=>S.ChainID===o.chainId);if(!y)return;const w=y.Upstreams.findIndex((S,X)=>(S.ID||`${o.chainId}-${X}`)===o.upstreamId);w<0||!await Se({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(y.Upstreams.splice(w,1),await V(o.gid,g,"Removing the endpoint"))}function p(a,o){const c=P(a);if(!c||!Number.isFinite(o))return;const g=((s==null?void 0:s.sources)??[]).filter(S=>S.chainId===o),y=j(c,o),w=new Set(((y==null?void 0:y.upstreams)??[]).filter(S=>S.kind!=="external").map(S=>`${S.kind}|${S.targetId??""}`)),M=g.filter(S=>!w.has(`${S.kind}|${S.targetId}`));te(`
        <h2>Add an endpoint for ${t((y==null?void 0:y.name)??`chain ${o}`)}</h2>
        ${M.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${M.map(S=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${t(S.kind)}:${t(S.targetId)}">
                       <span>${t(S.label)}</span>
                       <span class="muted small">${t(S.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${o}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn btn-ghost" data-modal-action="discover">Find public endpoints…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,S=>{if(S==="cancel"){z();return}if(S==="discover"){F(a,o);return}if(S==="manual"){W(a,o);return}if(S.startsWith("source:")){const[,X,ue]=S.split(":");z(),T(a,o,X,ue)}})}async function T(a,o,c,g){const y=P(a);if(!y)return;const w=G(y),M=w.Networks??[],S={ID:`${c==="managed-devnet"?"devnet":"node"}-${g}`,Kind:c,TargetID:g,Endpoint:"",Local:!0,RecentOnly:!1},X=M.find(ue=>ue.ChainID===o);X?X.Upstreams.push(S):M.push({ChainID:o,Upstreams:[S]}),w.Networks=M,await V(a,w,"Adding the endpoint")}async function F(a,o){te(`
        <h2>Public endpoints for chain ${o}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,w=>{w==="cancel"&&z()});let c;try{c=await It(o)}catch(w){const M=Me();if(M){const S=document.createElement("p");S.className="error small",S.textContent=`Could not discover endpoints: ${ae(w)}`,M.appendChild(S)}return}if(i)return;const g=(c.endpoints??[]).filter(w=>w.status==="live"||w.status==="unprobed"),y=(c.endpoints??[]).filter(w=>w.status==="rejected");te(`
        <h2>Public endpoints for chain ${o}</h2>
        ${c.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${c.fetchError?`<div class="small">${t(c.fetchError)}</div>`:""}</div>`:""}
        ${g.length?`<p class="muted small">${g.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${g.map(w=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(w.url)}">
                       <span><code>${t(w.url)}</code></span>
                       <span class="muted small">${w.status==="live"?`answered in ${w.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${o} right now.</p>`}
        ${y.length?`<details class="rpc-rejected">
                 <summary class="muted small">${y.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${y.map(w=>`<li class="muted small"><code>${t(w.url)}</code> — ${t(w.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,w=>{if(w==="cancel"){z();return}w.startsWith("add:")&&(z(),R(a,o,decodeURIComponent(w.slice(4))))})}function W(a,o){var c;te(`
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
      `,g=>{if(g==="cancel"){z();return}if(g!=="add")return;const y=document.getElementById("manual-endpoint"),w=document.getElementById("manual-recent"),M=document.getElementById("manual-err"),S=(y==null?void 0:y.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(S)){M&&(M.className="error small",M.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}z(),R(a,o,S,(w==null?void 0:w.checked)??!1)}),(c=document.getElementById("manual-endpoint"))==null||c.focus()}async function R(a,o,c,g=!1){const y=P(a);if(!y)return;const w=G(y),M=w.Networks??[],S=M.find(Fe=>Fe.ChainID===o),X=((S==null?void 0:S.Upstreams.length)??0)+1,ue={ID:`public-${o}-${X}`,Kind:"external",Endpoint:c,Local:!1,RecentOnly:g};S?S.Upstreams.push(ue):M.push({ChainID:o,Upstreams:[ue]}),w.Networks=M,await V(a,w,"Adding the endpoint")}async function Y(a,o){const c=D(a);if(!c||!o||!await Se({title:"Reset this devnet",body:`The chain on ${o} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[c.gid]="reset",x[c.gid]=null,k();let y;try{y=await $t(o)}catch(w){x[c.gid]=`Reset failed: ${ae(w)}${Le(w)}`,f[c.gid]=null,k();return}f[c.gid]=null,J(o,y),await q()}function J(a,o){const c=[];c.push(o.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),o.report.Recreated&&c.push("A fresh chain was started from genesis.");const g=o.report.Cascaded??[],y=o.report.CascadeSkipped??[];te(`
        <h2>Devnet on ${t(a)} reset</h2>
        <ul class="plain-list">${c.map(w=>`<li>${t(w)}</li>`).join("")}</ul>
        ${g.length?`<p class="ok">Restarted in front of it: ${t(g.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${y.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(y.join(", "))}.</p>`:""}
        ${o.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>z())}function ke(a){const o=P(a);if(!o)return;te(`
        <h2>Wipe ${t(o.label)}</h2>
        <p class="error">This destroys ${t(o.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${t(a)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(a)}</button>
        </div>
      `,y=>{if(y==="cancel"||y==="close"){z(),q();return}y==="confirm"&&Ee(a)});const c=document.getElementById("wipe-confirm-input"),g=document.getElementById("wipe-confirm-btn");c==null||c.addEventListener("input",()=>{g&&(g.disabled=c.value.trim()!==a)}),c==null||c.focus()}async function Ee(a){const o=document.getElementById("wipe-confirm-btn");o&&(o.disabled=!0,o.textContent="Wiping…");let c;try{c=await Et(a)}catch(g){const y=Me();if(y){const w=document.createElement("p");w.className="error small",w.textContent=`Wipe failed: ${ae(g)}${Le(g)}`,y.appendChild(w)}o&&(o.disabled=!1,o.textContent=`Wipe ${a}`);return}te(`
        <h2>${t(a)} wiped</h2>
        <ul class="plain-list">
          <li>${c.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${c.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${c.error?`<p class="error small">${t(c.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{z(),q()})}function Ie(){var g;const a=(s==null?void 0:s.targets)??[],o=new Set(((s==null?void 0:s.gateways)??[]).map(y=>y.id));if(a.length===0){te(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>z());return}const c=o.has("default")?"":"default";te(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${t(c)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${a.map(y=>`<option value="${t(y.id)}">${t(y.id)} (${t(y.mode)})</option>`).join("")}
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
      `,y=>{if(y==="cancel"){z();return}y==="create"&&je()}),(g=document.getElementById("new-gw-id"))==null||g.focus()}async function je(){const a=document.getElementById("new-gw-id"),o=document.getElementById("new-gw-target"),c=document.getElementById("new-gw-port"),g=document.getElementById("new-gw-err"),y=(a==null?void 0:a.value.trim())??"",w=(o==null?void 0:o.value)??"",M=Number.parseInt((c==null?void 0:c.value.trim())??"",10),S=X=>{g&&(g.className="error small",g.textContent=X)};if(!y){S("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!w){S("Pick the machine it runs on.");return}try{await Ct({id:y,placement:{targetId:w,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(M)?M:4e3,Networks:[]}})}catch(X){S(ae(X));return}z(),await q()}async function Re(a,o){const c=await Pe(o),g=a.textContent;a.textContent=c?"Copied!":"Copy failed",setTimeout(()=>{i||(a.textContent=g)},1500)}function ae(a){return a instanceof Error?a.message:String(a)}function Le(a){return a instanceof ve&&a.hint?` — ${a.hint}`:""}return()=>{i=!0,H==null||H(),z()}}const Kt="run",Jt={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Gt={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Vt(n,i){let s=!1,e=null,u=null;const f={devnet:null},x={devnet:null},v={devnet:[]};let N=null;const H={devnet:!1};let U=null;const q={devnet:null},P={devnet:null};n.innerHTML=`
    <div class="page-head">
      <h1>Services: ${t(i)}</h1>
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
  `;const j=n.querySelector("#services-body");ye(n,(r,p)=>{ge(r,p)}),k();async function k(){try{const r=await bt(i);if(s)return;e=r,u=null}catch(r){if(s)return;e=null,u=C(r)}l()}function $(r){return e==null?void 0:e.services.find(p=>p.id===r)}function l(){if(!s){if(u){j.innerHTML=`<p class="error">Could not read this machine's services: ${t(u)}</p>`;return}if(!e){j.innerHTML='<p class="muted">Loading…</p>';return}j.innerHTML=`
      ${b(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(E).join("")}
      </div>
    `}}function b(r){if(r.present&&r.reachable&&!r.hint)return`<p class="muted small">Docker: ${t(r.flavor)}${r.serverVersion?` ${t(r.serverVersion)}`:""} · reachable</p>`;const p=r.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${t(p)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${r.detail?`<div class="small">${t(r.detail)}</div>`:""}
        ${r.hint?`<div class="small">${t(r.hint)}</div>`:""}
      </div>
    `}function E(r){const p=r.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${t(r.label)}</h2>
          ${B(r)}
        </div>
        <p class="muted small">${t(Jt[r.id]??"")}</p>

        ${r.error?Z(r):""}
        ${r.blocked?`<div class="banner banner-warn">${t(r.blocked)}</div>`:""}
        ${p.map(T=>`<div class="banner banner-warn">${t(T)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${t(r.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${r.status.Image?`<code>${t(r.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${ee(r)}

        ${Q(r)}

        <div class="card-actions">
          ${(r.actions??[]).map(T=>se(r,T)).join("")}
        </div>
        ${x[r.id]?`<p class="error small">${t(x[r.id])}</p>`:""}
        ${le(r)}

        ${de(r)}
      </div>
    `}function B(r){switch(r.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function Z(r){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${t(r.error??"")}</div>
        ${r.hint?`<div class="small">${t(r.hint)}</div>`:""}
      </div>
    `}function ee(r){if(r.status.State!=="created-but-stopped"||r.status.ExitCode===0)return"";const p=r.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${r.status.ExitCode}${p}.</p>`}function Q(r){const p=r.endpoints??[];return p.length===0?r.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":p.map(T=>`
        <div class="endpoint-row">
          ${ie("ok")}
          <span class="muted small">${t(T.label)}</span>
          <code class="endpoint-url">${t(T.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(T.url)}">Copy</button>
        </div>`).join("")}function se(r,p){const T=Gt[p];if(!T)return"";const F=f[r.id],W=p==="create"?`Create ${r.id==="devnet"?"devnet":"gateway"}`:T.label;return`
      <button class="${T.className}" data-action="svc-${p}" data-svc="${t(r.id)}"
              title="${t(T.title)}" ${F?"disabled":""}>
        ${F===p?'<span class="spinner" aria-label="working"></span>':t(W)}
      </button>
    `}function le(r){const p=v[r.id]??[];return p.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${t(p.join(`
`))}</pre>
      </div>
    `}function de(r){const p=H[r.id],T=pe(r);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${r.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${t(r.id)}">
            ${p?"Close":"Edit"}
          </button>
        </div>
        ${p?oe():`<p class="small">${T}</p>`}
        ${q[r.id]?`<p class="error small">${t(q[r.id])}</p>`:""}
        ${P[r.id]?`<p class="muted small">${t(P[r.id])}</p>`:""}
      </div>
    `}function pe(r){const p=r.devnet;return p?`Chain ${p.ChainID} · a block every ${t(p.BlockTime)} · JSON-RPC on ${t(p.BindAddr)}:${p.HTTPPort} · WebSocket on ${t(p.BindAddr)}:${p.WSPort}`:"—"}function oe(r){return G()}function G(){const r=U;return r?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${t(r.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${r.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${r.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${t(r.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${r.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function V(){H.devnet&&U&&(U.BlockTime=me("#dev-blocktime",U.BlockTime),U.HTTPPort=be("#dev-http",U.HTTPPort),U.WSPort=be("#dev-ws",U.WSPort),U.BindAddr=me("#dev-bind",U.BindAddr))}function me(r,p){const T=n.querySelector(r);return T?T.value.trim():p}function be(r,p){const T=n.querySelector(r);if(!T)return p;const F=Number.parseInt(T.value.trim(),10);return Number.isFinite(F)?F:p}async function ge(r,p){const T=p.dataset.svc??"";switch(r){case"refresh":await k();return;case"copy":p.dataset.copy&&await h(p,p.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await he(T,r.slice(4));return;case"svc-create":case"svc-recreate":await m(T);return;case"svc-wipe":A(T);return;case"toggle-config":I(T);return;case"save-config":await L(T);return;default:return}}async function he(r,p){if(!f[r]){f[r]=p,x[r]=null,l();try{await gt(i,r,p)}catch(T){x[r]=`${p} failed: ${C(T)}${D(T)}`}f[r]=null,await k()}}async function m(r){if(!f[r]){f[r]="create",x[r]=null,v[r]=["starting…"],l();try{await yt(i,r)}catch(p){x[r]=`${C(p)}${D(p)}`,v[r]=[],f[r]=null,l();return}N==null||N(),N=_e(i,p=>{if(s)return;const T=p.err?`${p.stepId}: ${p.err}`:p.line?`${p.stepId}: ${p.line}`:`${p.stepId}: done`;if(v[r]=[...(v[r]??[]).filter(W=>W!=="starting…"),T],!!p.err||p.stepId===Kt&&!!p.done){N==null||N(),N=null,f[r]=null,p.err&&(x[r]="Provisioning failed — see the log below."),k();return}l()})}}function I(r){if(V(),H[r]=!H[r],q[r]=null,P[r]=null,H[r]){const p=$(r);p!=null&&p.devnet&&(U={...p.devnet})}l()}async function L(r){var F;V(),q[r]=null,P[r]=null;const p=U;if(!p)return;if(p.HTTPPort===p.WSPort){q[r]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",l();return}try{await wt(i,r,p)}catch(W){q[r]=C(W),l();return}const T=((F=$(r))==null?void 0:F.status.State)==="running";H[r]=!1,P[r]=T?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await k()}function A(r){const p=$(r);if(!p)return;const T=(p.restartsOnWipe??[]).map(R=>{var Y;return((Y=$(R))==null?void 0:Y.label)??R});te(`
        <h2>Wipe ${t(p.label)}</h2>
        <p class="error">This deletes ${t(p.wipeDiscards)}</p>
        ${T.length?`<p>It also restarts what sits in front of it: ${t(T.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${t(r)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(r)}</button>
        </div>
      `,R=>{if(R==="cancel"||R==="close"){z(),k();return}R==="confirm"&&K(r)});const F=document.getElementById("wipe-confirm-input"),W=document.getElementById("wipe-confirm-btn");F==null||F.addEventListener("input",()=>{W&&(W.disabled=F.value.trim()!==r)}),F==null||F.focus()}async function K(r){const p=document.getElementById("wipe-confirm-btn");p&&(p.disabled=!0,p.textContent="Wiping…");let T;try{T=await vt(i,r)}catch(F){const W=Me();if(W){const R=document.createElement("p");R.className="error small",R.textContent=`Wipe failed: ${C(F)}${D(F)}`,W.appendChild(R)}p&&(p.disabled=!1,p.textContent=`Wipe ${r}`);return}d(r,T)}function d(r,p){const T=$(r),F=J=>{var ke;return((ke=$(J))==null?void 0:ke.label)??J},W=[];W.push(p.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const J of p.report.VolumesRemoved??[])W.push(`Volume ${J} deleted.`);for(const J of p.report.VolumesAbsent??[])W.push(`Volume ${J} was already gone.`);p.report.Recreated&&W.push("Container re-created from your saved configuration.");const R=(p.report.Cascaded??[]).map(F),Y=(p.report.CascadeSkipped??[]).map(F);te(`
        <h2>${t((T==null?void 0:T.label)??r)} wiped</h2>
        <ul class="plain-list">${W.map(J=>`<li>${t(J)}</li>`).join("")}</ul>
        ${R.length?`<p class="ok">Restarted in front of it: ${t(R.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${Y.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(Y.join(", "))}.</p>`:""}
        ${p.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(p.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,J=>{(J==="close"||J==="cancel")&&(z(),k())})}async function h(r,p){const T=await Pe(p),F=r.textContent;r.textContent=T?"Copied!":"Copy failed",setTimeout(()=>{s||(r.textContent=F)},1500)}function C(r){return r instanceof Error?r.message:String(r)}function D(r){return r instanceof ve&&r.hint?` — ${r.hint}`:""}return()=>{s=!0,N==null||N(),z()}}const Yt="local";function Zt(n){let i=!1,s=!1,e="",u=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ne()}
  `;const f=n.querySelector("#targets-body");ye(n,(l,b)=>{U(l,b)}),x();async function x(){try{const[l,b,E]=await Promise.all([Te(),Ce(),nt()]);if(i)return;e=E.os,N(l,b)}catch(l){if(i)return;f.innerHTML=`<p class="error">Failed to load machines: ${t(String(l))}</p>`}}function v(){u&&N(u.targets,u.catalog)}function N(l,b){u={targets:l,catalog:b};const E=e==="linux",B=[...l].sort((Q,se)=>(Q.mode==="local"?-1:0)-(se.mode==="local"?-1:0)),Z=B.length?`<div class="card-grid">${B.map(Q=>Xt(Q,b,Q.mode!=="local"||E,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',ee=l.some(Q=>Q.mode==="local");f.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${Z}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${H(E,ee)}
        ${s?Qt():""}
      </section>
    `}function H(l,b){const E=`
      <div class="card">
        <h3>A server over SSH ${O("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${l?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${l?" btn-ghost":""}" data-action="toggle-ssh">
            ${s?"Cancel":"Add a server"}
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
      `;return b?`<div class="card-grid card-grid-wide">${E}</div>`:`<div class="card-grid card-grid-wide">${l?B+E:E+B}</div>`}async function U(l,b){var E;if(l==="add-local"){await q();return}if(l==="delete-target"){const B=b.dataset.id;if(!B||!await Se({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await P(B);return}if(l==="toggle-ssh"){s=!s,$(),v(),s&&((E=n.querySelector("#ssh-host"))==null||E.focus());return}l==="add-ssh"&&await j()}async function q(){$();try{await Ke({id:Yt,mode:"local"}),await x()}catch(l){k(l)}}async function P(l){try{await at(l),await x()}catch(b){k(b)}}async function j(){const l=n.querySelector("#ssh-host"),b=n.querySelector("#ssh-user"),E=n.querySelector("#ssh-key"),B=n.querySelector("#ssh-port"),Z=n.querySelector("#ssh-id");if(!l||!b||!E||!B||!Z)return;const ee=l.value.trim(),Q=b.value.trim(),se=E.value.trim(),le=B.value.trim(),de=Z.value.trim();if($(),!ee||!Q||!se){k(new Error("host, user, and key path are required"));return}const pe=de||en(ee),oe={Host:ee,User:Q,KeyPath:se};if(le){const V=Number.parseInt(le,10);if(!Number.isFinite(V)||V<=0){k(new Error("port must be a positive number"));return}oe.Port=V}const G=n.querySelector("#ssh-submit");G&&(G.disabled=!0,G.textContent="Connecting…");try{await Ke({id:pe,mode:"ssh",ssh:oe}),s=!1,await x()}catch(V){k(V),G&&(G.disabled=!1,G.textContent="Add server")}}function k(l){let b=n.querySelector("#targets-error");b||(f.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),b=n.querySelector("#targets-error")),b.textContent=String(l instanceof Error?l.message:l)}function $(){var l;(l=n.querySelector("#targets-error"))==null||l.remove()}return()=>{i=!0}}function Xt(n,i,s,e){const u=n.wire,f=n.mode==="local"?"this machine":"SSH",x=n.mode==="ssh"&&n.ssh?`${t(n.ssh.User)}@${t(n.ssh.Host)}`:f,v=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(n.id)}">Devnet</a>`;let N,H;if(!u&&!s)N=`${O("can't run a node","warn")} ${O(e||"not Linux","neutral")}`,H=`
      ${v}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(n.id)}">Preview setup wizard</a>
    `;else if(!u)N=O("not set up","neutral"),H=`
      <a class="btn" href="#/setup/${encodeURIComponent(n.id)}">Run setup wizard</a>
      ${v}
    `;else{const U=i.networks.find(P=>P.ChainID===u.ChainID),q=U?U.Name:`chain ${u.ChainID}`;N=`${O(q,"ok")} ${O(u.ExecID,"neutral")} ${O(u.BeaconID,"neutral")}${u.Archive?" "+O("archive","warn"):""}`,H=`
      <a class="btn" href="#/dash/${encodeURIComponent(n.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(n.id)}">Logs</a>
      ${v}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(n.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${t(n.id)}</h2>
      <p class="muted">${x}</p>
      <p>${N}</p>
      <div class="card-actions">
        ${H}
        <button class="btn btn-danger" data-action="delete-target" data-id="${t(n.id)}">Remove</button>
      </div>
    </div>
  `}function Qt(){return`
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
  `}function en(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const We=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Be=8545,Ae=5052,De=30303,tn=[369,943,1],Xe={369:"default",943:"practise here first"};function nn(n,i){let s=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${t(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ne()}</div>`;const u=n.querySelector("#wizard-body"),f=n.querySelector("#wizard-footer");ye(n,(d,h)=>{be(d,h)}),ze(n,(d,h)=>{d==="exec-select"?e.execId=h:d==="beacon-select"&&(e.beaconId=h),v()}),n.addEventListener("change",d=>{const h=d.target;h instanceof HTMLInputElement&&(h.id==="data-dir-input"?(ge(),se()):h.id==="checkpoint-toggle"?(e.checkpoint=h.checked,v()):h.id==="exec-snapshot-toggle"&&(e.execSnapshot=h.checked,v()))}),x();async function x(){try{const[d,h]=await Promise.all([Ce(),Te()]);if(s)return;e.catalog=d;const C=h.find(D=>D.id===i);C!=null&&C.wire&&(e.chainId=C.wire.ChainID,e.execId=C.wire.ExecID,e.beaconId=C.wire.BeaconID,e.archive=C.wire.Archive,C.wire.ExecHTTPPort&&(e.execHTTPPort=String(C.wire.ExecHTTPPort)),C.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(C.wire.BeaconHTTPPort)),C.wire.ExecP2PPort&&(e.execP2PPort=String(C.wire.ExecP2PPort)),C.wire.RPCBindAddr&&(e.rpcBindAddr=C.wire.RPCBindAddr)),v()}catch(d){if(s)return;e.loadError=String(d instanceof Error?d.message:d),v()}}function v(){if(e.loadError){u.innerHTML=`<p class="error">Failed to load: ${t(e.loadError)}</p>`;return}e.catalog&&(u.innerHTML=`
      ${K(e.step)}
      ${H()}
    `,N())}function N(){var h;const d=(h=e.catalog)==null?void 0:h.networks.find(C=>C.ChainID===e.chainId);f.innerHTML=d?ne(d.Name,d.LearnURL):ne()}function H(){switch(e.step){case"network":return U();case"clients":return q();case"mode":return G();case"review":return V();case"run":return me()}}function U(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${tn.map(C=>{const D=d.networks.find(T=>T.ChainID===C);if(!D)return"";const r=e.chainId===C,p=Xe[C]?O(Xe[C],C===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${r?"selected":""}" data-action="pick-network" data-chain-id="${C}" type="button">
          <h3>${t(D.Name)} <span class="muted">(chain ${C})</span></h3>
          ${p}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function q(){const d=e.catalog,h=d.networks.find(r=>r.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const C=h.ExecClients.map(r=>de(r,d)),D=h.BeaconClients.map(r=>de(r,d));return`
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
          ${Ue("exec-select",C,e.execId)}
        </label>
        ${oe(e.execId,d)}
        <label>
          Beacon client
          ${Ue("beacon-select",D,e.beaconId)}
        </label>
        ${oe(e.beaconId,d)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function P(d){return d<=0?"—":d>=1?`~${d.toFixed(1)} TB`:`~${Math.round(d*1e3)} GB`}const j=1.1,k=.5,$="Valve reth snapshot",l="rough estimate";function b(d){return d.SnapshotSizeTB}function E(d){return d.SnapshotSizeTB*k}function B(d){return`<p class="muted small">${P(b(d))} is the measured size of Valve's reth snapshot for ${t(d.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function Z(d){return{archive:b(d)*1e12*j,full:E(d)*1e12*j}}function ee(d,h){if(!d)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${t(h)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${t(h)}</code>: ${t(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==h)return"";const C=Z(d),D=e.freeBytes>=C.archive,r=e.freeBytes>=C.full,p=`<p class="muted small">Free at <code>${t(h)}</code>: <strong>${we(e.freeBytes)}</strong> — archive ${D?"fits":"won't fit"} (${P(b(d))}, ${$}), full ${r?"fits":"won't fit"} (${P(E(d))}, ${l}).</p>`;let T="";return e.downgradeNote?T=`<p class="banner banner-warn">${t(e.downgradeNote)}</p>`:r||(T=`<p class="banner banner-warn">Neither full (${P(E(d))}, ${l}) nor archive (${P(b(d))}, ${$}) fits the free space here — choose a location with more room.</p>`),p+T}function Q(d,h){if(e.downgradeNote=null,!d||e.freeBytes===null)return;const C=Z(d);e.archive&&e.freeBytes<C.archive&&e.freeBytes>=C.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${h} for archive (${P(b(d))}, ${$}) — switched to Full (${P(E(d))}, ${l}). Pick a location with more room to run archive.`)}async function se(){var C;if(e.chainId===null)return;const d=(C=e.catalog)==null?void 0:C.networks.find(D=>D.ChainID===e.chainId),h=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,v();try{const{freeBytes:D}=await st(e.targetId,h);if(s)return;e.freeBytes=D,e.probedPath=h,Q(d,h)}catch(D){if(s)return;e.freeBytes=null,e.probedPath=h,e.diskError=String(D instanceof Error?D.message:D)}e.diskProbing=!1,v()}function le(d){return d?/^https?:\/\/.+/i.test(d)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(d,h){const C=h.clients.find(D=>D.id===d);return{value:d,label:C?`${C.id} — ${pe(C.repo)}`:d}}function pe(d){const h=d.split("/");return h.length>=4?h[3]:d}function oe(d,h){const C=d?h.clients.find(r=>r.id===d):void 0;if(!C)return"";const D=C.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${t(C.repo)}" target="_blank" rel="noopener noreferrer">${t(D)}</a></p>`}function G(){var F,W,R;const d=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",h=(F=e.catalog)==null?void 0:F.networks.find(Y=>Y.ChainID===e.chainId),C=((R=(W=e.catalog)==null?void 0:W.clients.find(Y=>Y.id===e.execId))==null?void 0:R.snapshotSupported)??!1,D=h?`${P(E(h))} (${l})`:"Smaller",r=h?`${P(b(h))} (${$})`:"Much larger",p=h?` on ${t(h.Name)}`:"",T=h?e.checkpoint?h.SyncLabel:h.GenesisSyncLabel:"";return`
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
          ${h?`<p class="sync-estimate">⏱ Estimated initial sync${p}: <strong>${t(T)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${t((h==null?void 0:h.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${t((h==null?void 0:h.CheckpointURL)??"")}" value="${t(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${t(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${C?`
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
              <tr><th>Approx. disk footprint${p}</th><td class="yes">${D}</td><td class="limited">${r}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${h?B(h):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${r}${h?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
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
          ${ee(h,e.dataDir||d)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${t(d)}/jwt.hex" value="${t(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Be})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Be}" value="${t(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${t(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ae})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ae}" value="${t(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${t(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${De})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${De}" value="${t(e.execP2PPort)}" />
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
    `}function V(){const h=e.catalog.networks.find(J=>J.ChainID===e.chainId),C=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,D=e.jwtPath||`${C}/jwt.hex`,r=We.map(J=>`<li>${t(J.title)}</li>`).join(""),p=L(e.execHTTPPort,Be),T=L(e.beaconHTTPPort,Ae),F=L(e.execP2PPort,De),W=p||T||F?`<tr><th>Non-default ports</th><td>${[p?`exec HTTP ${p}`:null,T?`beacon HTTP ${T}`:null,F?`exec p2p ${F}`:null].filter(J=>J!==null).map(t).join(", ")}</td></tr>`:"",{addr:R}=he(e.rpcBindAddr),Y=R?`<tr><th>RPC bind address</th><td><code>${t(R)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${t(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${t((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${t(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${t(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${t(C)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${t(D)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${t(e.checkpointUrl||(h==null?void 0:h.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${W}
            ${Y}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${r}</ol>
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function me(){const h=e.catalog.networks.find(R=>R.ChainID===e.chainId),C=h==null?void 0:h.LearnURL,D=new Set(e.events.filter(R=>R.done).map(R=>R.stepId)),r=new Set(e.events.filter(R=>R.err).map(R=>R.stepId)),p=new Map;for(const R of e.events){if(!R.line)continue;const Y=p.get(R.stepId)??[];Y.push(R.line),p.set(R.stepId,Y)}const T=We.map(R=>{var Re;const Y=D.has(R.id),J=r.has(R.id),ke=J?O("failed","bad"):Y?O("done","ok"):O("pending","neutral"),Ee=(p.get(R.id)??[]).slice(-5),Ie=(Re=e.events.find(ae=>ae.stepId===R.id&&ae.err))==null?void 0:Re.err,je=R.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${C?` <a href="${t(C)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${Y?"step-done":""} ${J?"step-error":""}">
          <div class="step-head">${ke} <strong>${t(R.title)}</strong></div>
          ${je}
          ${Ee.length?`<pre class="step-log">${Ee.map(ae=>t(ae)).join(`
`)}</pre>`:""}
          ${Ie?`<p class="error small">${t(Ie)}</p>`:""}
        </li>
      `}).join(""),F=e.events.some(R=>R.err),W=We.every(R=>D.has(R.id))||e.events.some(R=>R.stepId==="handshake"&&R.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${T}</ol>
        ${W&&!F?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        ${F?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(d,h){switch(d){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,v();break;case"goto-network":e.step="network",v();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",v();break;case"goto-mode":e.step="mode",v(),se();break;case"goto-review":if(ge(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){v();break}e.step="review",v();break;case"start-setup":A();break}}function ge(){const d=n.querySelectorAll('input[name="mode"]');for(const R of Array.from(d))R.checked&&(e.archive=R.value==="archive");const h=n.querySelector("#data-dir-input"),C=n.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),C&&(e.jwtPath=C.value.trim());const D=n.querySelector("#exec-http-port-input"),r=n.querySelector("#beacon-http-port-input"),p=n.querySelector("#exec-p2p-port-input");D&&(e.execHTTPPort=D.value.trim()),r&&(e.beaconHTTPPort=r.value.trim()),p&&(e.execP2PPort=p.value.trim());const T=n.querySelector("#rpc-bind-addr-input");T&&(e.rpcBindAddr=T.value.trim());const F=n.querySelector("#checkpoint-url-input");F&&(e.checkpointUrl=F.value.trim());const W=n.querySelector("#snapshot-key-input");W&&(e.snapshotKey=W.value.trim()),e.execHTTPPortError=I(e.execHTTPPort).error??null,e.beaconHTTPPortError=I(e.beaconHTTPPort).error??null,e.execP2PPortError=I(e.execP2PPort).error??null,e.rpcBindAddrError=he(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?le(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function he(d){if(!d)return{};const h=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(d);return h?h.slice(1).every(C=>Number(C)<=255)?{addr:d}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(d)&&d.includes(":")?{addr:d}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const m=/^\d+$/;function I(d){if(!d)return{};if(!m.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const h=Number(d);return!Number.isInteger(h)||h<1||h>65535?{error:"Port must be between 1 and 65535."}:{port:h}}function L(d,h){const{port:C}=I(d);if(!(C===void 0||C===h))return C}async function A(){var p;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(p=e.streamStop)==null||p.call(e),e.streamStop=null,v();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const h=L(e.execHTTPPort,Be),C=L(e.beaconHTTPPort,Ae),D=L(e.execP2PPort,De);h!==void 0&&(d.ExecHTTPPort=h),C!==void 0&&(d.BeaconHTTPPort=C),D!==void 0&&(d.ExecP2PPort=D);const{addr:r}=he(e.rpcBindAddr);r!==void 0&&(d.RPCBindAddr=r),e.checkpoint?e.checkpointUrl&&(d.CheckpointURL=e.checkpointUrl):d.NoCheckpoint=!0,e.execSnapshot&&(d.ExecSnapshot=!0,d.SnapshotKey=e.snapshotKey);try{await ot(e.targetId,d)}catch(T){if(!(T instanceof ve&&T.status===409)){e.starting=!1,e.startError=String(T instanceof Error?T.message:T),v();return}}e.starting=!1,e.step="run",v(),e.streamStop=_e(e.targetId,T=>{s||(e.events.push(T),e.step==="run"&&v())})}function K(d){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],D=h.map(r=>r.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${h.map((r,p)=>`<li class="${p===D?"current":p<D?"past":"future"}">${t(r.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;s=!0,(d=e.streamStop)==null||d.call(e)}}const an=document.querySelector("#app"),{contentEl:sn,setActiveNav:on}=Nt(an);let re=null;function rn(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[s,e]=i;return s==="setup"||s==="dash"||s==="logs"||s==="security"||s==="diag"||s==="services"?{screen:s,id:e?decodeURIComponent(e):void 0}:{screen:s??"targets"}}function fe(n){const i=document.createElement("div");return sn.replaceChildren(i),n(i)}function Qe(){if(re){try{re()}catch{}re=null}const{screen:n,id:i}=rn();switch(on(n),n){case"setup":if(!i){location.hash="#/targets";return}re=fe(s=>nn(s,i));break;case"dash":if(!i){location.hash="#/targets";return}re=fe(s=>Ht(s,i));break;case"logs":if(!i){location.hash="#/targets";return}re=fe(s=>Ut(s,i));break;case"security":if(!i){location.hash="#/targets";return}re=fe(s=>Ot(s,i));break;case"diag":if(!i){location.hash="#/targets";return}re=fe(s=>Mt(s,i));break;case"services":if(!i){location.hash="#/targets";return}re=fe(s=>Vt(s,i));break;case"rpc":re=fe(s=>zt(s));break;case"settings":re=fe(s=>Ft(s));break;case"targets":default:re=fe(s=>Zt(s));break}}window.addEventListener("hashchange",Qe);Qe();
