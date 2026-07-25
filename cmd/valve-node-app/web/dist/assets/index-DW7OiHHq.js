var We=Object.defineProperty;var ze=(t,s,r)=>s in t?We(t,s,{enumerable:!0,configurable:!0,writable:!0,value:r}):t[s]=r;var $e=(t,s,r)=>ze(t,typeof s!="symbol"?s+"":s,r);(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))e(l);new MutationObserver(l=>{for(const h of l)if(h.type==="childList")for(const y of h.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function r(l){const h={};return l.integrity&&(h.integrity=l.integrity),l.referrerPolicy&&(h.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?h.credentials="include":l.crossOrigin==="anonymous"?h.credentials="omit":h.credentials="same-origin",h}function e(l){if(l.ep)return;l.ep=!0;const h=r(l);fetch(l.href,h)}})();function Ke(){return O("/api/host")}function he(){return O("/api/catalog")}function fe(){return O("/api/targets")}function Be(t){return O("/api/targets",{method:"POST",headers:de,body:JSON.stringify(t)})}function Je(t){return O(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Ve(t,s){return O(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(s)}`)}function Ge(t,s){return O(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:de,body:JSON.stringify(s)})}function Oe(t,s){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function Ye(t,s){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function Ze(t,s=200){return O(`/api/targets/${encodeURIComponent(t)}/logs?n=${s}`)}function Xe(t,s){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function He(t,s){const r=s===void 0?{}:{lines:s};return O(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:de,body:JSON.stringify(r)})}function Qe(t,s,r){return O(`/api/targets/${encodeURIComponent(t)}/services/${s}/${r}`,{method:"POST"})}function et(t,s){return O(`/api/targets/${encodeURIComponent(t)}/services/${s}/clear`,{method:"POST",headers:de,body:JSON.stringify({Confirm:s})})}function tt(t){return O(`/api/targets/${encodeURIComponent(t)}/du`)}function nt(t){return O(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function at(t){return O(`/api/targets/${encodeURIComponent(t)}/firewall`)}function rt(t){return O(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function st(t){return O(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function ot(t){return O(`/api/targets/${encodeURIComponent(t)}/containers`)}function it(t,s,r){return O(`/api/targets/${encodeURIComponent(t)}/containers/${s}/${r}`,{method:"POST"})}async function ct(t,s){const r=await fetch(`/api/targets/${encodeURIComponent(t)}/containers/${s}/wipe`,{method:"POST",headers:de,body:JSON.stringify({Confirm:s})}),e=await r.text();let l=null;try{l=e?JSON.parse(e):null}catch{}if(l&&typeof l=="object"&&"report"in l)return l;const h=l&&typeof l=="object"&&typeof l.error=="string"?l.error:r.statusText||`HTTP ${r.status}`;throw new ye(r.status,h)}function lt(t,s){return O(`/api/targets/${encodeURIComponent(t)}/containers/${s}/provision`,{method:"POST"})}function dt(t,s,r){return O(`/api/targets/${encodeURIComponent(t)}/containers/${s}/config`,{method:"PUT",headers:de,body:JSON.stringify(r)})}function ut(){return O("/api/settings")}function pt(t){return O("/api/settings",{method:"PUT",headers:de,body:JSON.stringify(t)})}class ye extends Error{constructor(r,e,l,h){super(e);$e(this,"status");$e(this,"hint");$e(this,"code");this.name="ApiError",this.status=r,this.hint=l,this.code=h}}const de={"Content-Type":"application/json"};async function O(t,s){const r=await fetch(t,s);if(!r.ok){let l=r.statusText||`HTTP ${r.status}`,h,y;try{const m=await r.json();m&&typeof m.error=="string"&&m.error&&(l=m.error),m&&typeof m.hint=="string"&&m.hint&&(h=m.hint),m&&typeof m.code=="string"&&m.code&&(y=m.code)}catch{}throw new ye(r.status,l,h,y)}if(r.status===204)return;const e=await r.text();return e?JSON.parse(e):void 0}const Ae="https://learn.valve.city/rpc";function a(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function G(t,s){const r=t&&s&&s!==Ae?` <span class="footer-sep">·</span> <a href="${a(s)}" target="_blank" rel="noopener noreferrer">${a(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(Ae)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function ht(t){t.innerHTML=`
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
  `;const s=t.querySelector("#content"),r=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:s,setActiveNav:l=>{for(const h of r)h.classList.toggle("active",h.dataset.nav===l)}}}function ce(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function ft(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function mt(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const s=Math.round(t/60),r=Math.floor(s/60),e=s%60;if(r===0)return`~${e}m`;if(r<48)return`~${r}h ${e}m`;const l=Math.floor(r/24),h=r%24;return`~${l}d ${h}h`}function H(t,s){return`<span class="badge badge-${s}">${a(t)}</span>`}function Le(t){return`<span class="dot dot-${t}"></span>`}const De=["B","KB","MB","GB","TB","PB"];function le(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let s=t,r=0;for(;s>=1024&&r<De.length-1;)s/=1024,r++;const e=s<10?2:s<100?1:0;return`${s.toFixed(e)} ${De[r]}`}async function Se(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function ue(t,s){t.addEventListener("click",r=>{const e=r.target.closest("[data-action]");if(!e||!t.contains(e))return;const l=e.dataset.action;l&&s(l,e,r)})}function Re(t,s,r){const e=s.find(h=>h.value===r),l=s.map(h=>`
      <li class="dropdown-option${h.value===r?" selected":""}" role="option"
          aria-selected="${h.value===r}" data-value="${a(h.value)}">
        ${a(h.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${l}</ul>
    </div>
  `}function ve(t){t.querySelectorAll(".dropdown.open").forEach(s=>{var r;s.classList.remove("open"),(r=s.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function Fe(t,s){t.addEventListener("click",l=>{const h=l.target,y=h.closest(".dropdown-trigger");if(y&&t.contains(y)){const E=y.closest(".dropdown"),A=!!E&&!E.classList.contains("open");ve(t),E&&A&&(E.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const m=h.closest(".dropdown-option");if(m&&t.contains(m)){const E=m.closest(".dropdown");ve(t),s((E==null?void 0:E.dataset.dropdown)??"",m.dataset.value??"");return}ve(t)});const r=l=>{if(!t.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}const h=l.target;(!h.closest(".dropdown")||!t.contains(h))&&ve(t)},e=l=>{if(!t.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}l.key==="Escape"&&ve(t)};document.addEventListener("click",r),document.addEventListener("keydown",e)}const Ee="app-modal";let Te=null;function Pe(t,s){se();const r=document.createElement("div");r.className="modal-overlay",r.id=Ee,r.innerHTML=`<div class="modal">${t}</div>`,r.addEventListener("click",l=>{const h=l.target.closest("[data-modal-action]");h!=null&&h.dataset.modalAction?s(h.dataset.modalAction):l.target===r&&s("cancel")});const e=l=>{l.key==="Escape"&&s("cancel")};document.addEventListener("keydown",e),Te=e,document.body.appendChild(r)}function se(){var t;(t=document.getElementById(Ee))==null||t.remove(),Te&&(document.removeEventListener("keydown",Te),Te=null)}function qe(){return document.querySelector(`#${Ee} .modal`)}function bt(t){return new Promise(s=>{var l;let r=!1;const e=h=>{r||(r=!0,se(),s(h))};Pe(`
        <h2>${a(t.title)}</h2>
        <p>${a(t.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm">${a(t.confirmLabel)}</button>
        </div>
      `,h=>e(h==="confirm")),(l=document.querySelector(`#${Ee} [data-modal-action="confirm"]`))==null||l.focus()})}const gt=85,Ce={exec:"Execution",beacon:"Beacon"};function vt(t,s){let r=!1,e=null,l=null,h=null,y=null,m=null,E=null,A=null,I=null;const B={exec:null,beacon:null};let T=null;t.innerHTML=`<h1>Dashboard: ${a(s)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${G()}</div>`;const R=t.querySelector("#dash-body"),$=t.querySelector("#dash-footer");R.addEventListener("click",p=>{const v=p.target.closest("[data-action]");if(!v||!R.contains(v))return;const x=v.dataset.action;if(x==="svc-action"){const P=v.dataset.svc,F=v.dataset.kind;P&&F&&me(P,F)}else if(x==="open-clear"){const P=v.dataset.svc;P&&pe(P)}else if(x==="copy"){const P=v.dataset.copy;P&&be(v,P)}else x==="retry-du"?i():x==="retry-endpoints"&&u()}),b();async function b(){let p,v;try{const[P,F]=await Promise.all([fe(),he()]);p=P.find(c=>c.id===s),v=F}catch(P){if(r)return;R.innerHTML=`<p class="error">Failed to load target: ${a(String(P))}</p>`;return}if(r)return;if(!p){R.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){R.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const x=v==null?void 0:v.networks.find(P=>P.ChainID===p.wire.ChainID);x&&($.innerHTML=G(x.Name,x.LearnURL)),R.innerHTML='<p class="muted">Connecting…</p>',e=Ye(s,P=>{r||(w(P),l=P,h=P,S())}),i(),u()}async function i(){E=null;try{m=await tt(s)}catch(p){m=null,E=String(p instanceof Error?p.message:p)}r||S()}async function u(){I=null;try{A=await nt(s)}catch(p){A=null,I=String(p instanceof Error?p.message:p)}r||S()}function w(p){if(!l)return;const v=(new Date(p.at).getTime()-new Date(l.at).getTime())/1e3,x=p.execHead-l.execHead;if(v>0&&x>=0){const P=x/v;y=y===null?P:y*.7+P*.3}}function S(){if(!h)return;const p=h;R.innerHTML=`
      <p class="dash-status">${j(p)}</p>
      <div class="card-grid">
        ${Y(p)}
        ${W(p)}
        ${Z(p)}
        ${ne(p)}
        ${ae(p)}
        ${oe()}
      </div>
      <p class="muted small">Last updated ${a(new Date(p.at).toLocaleTimeString())}</p>
    `}function j(p){return!p.execActive&&!p.beaconActive?H("Node not running","bad"):p.execSyncing||p.beaconDistance>0?H("Syncing","warn"):H("Running · synced","ok")}function K(p){const x=p.refHead>0?p.refHead-p.execHead:null,P=x!==null&&x>0&&y&&y>0?mt(x/y):x!==null&&x<=0?"caught up":"—";return{lag:x,eta:P}}function W(p){const{lag:v,eta:x}=K(p);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${p.execActive?p.execSyncing?H("syncing","warn"):p.execHead===0?H("no data","neutral"):H("synced","ok"):H("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ce(p.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${v!==null?ce(p.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${v!==null?ce(Math.max(v,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${x}</dd></div>
        </dl>
      </div>
    `}function Z(p){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${p.beaconActive?p.beaconSlot===0?H("no data","neutral"):p.beaconDistance===0?H("synced","ok"):H("syncing","warn"):H("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ce(p.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ce(p.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ne(p){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ce(p.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ce(p.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function ae(p){const v=p.diskUsedPct>=gt,x=`
      <div class="meter"><div class="meter-fill ${v?"meter-warn":""}" style="width:${Math.min(p.diskUsedPct,100)}%"></div></div>
      <p>${ft(p.diskUsedPct)} used</p>
    `;if(E)return`
        <div class="card ${v?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="error small">${a(E)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${v?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="muted">Loading…</p>
        </div>
      `;const P=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,F=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:c,eta:d}=K(p),g=c!==null&&c>0&&y!==null&&y>0;return`
      <div class="card ${v?"card-warn":""}">
        <h3>Storage</h3>
        ${x}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${le(m.ExecBytes)} of ~${le(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${P}%"></div></div>
        ${g?`<p class="muted small">Estimated time remaining: ${a(d)}</p>`:""}
        <p class="muted small">Beacon — ${le(m.BeaconBytes)} of ~${le(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${F}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${le(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function oe(){if(I)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(I)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!A)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const p=A,v=p.ExecReachable&&!p.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",x=p.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(p.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(p.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${Le(p.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(p.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(p.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${Le(p.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(p.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(p.BeaconHTTP)}">Copy</button>
        </div>
        ${v}
        ${x}
      </div>
    `}function te(p,v){const x=Ce[p],P=B[p],F=(c,d,g)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${p}" data-kind="${c}" ${P!==null||g?"disabled":""}>${P===c?X():a(d)}</button>`;return`
      <div class="service-row">
        <span>${a(x)} ${v?H("active","ok"):H("down","bad")}</span>
        <div class="service-actions">
          ${F("start","Start",v)}
          ${F("stop","Stop",!v)}
          ${F("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${p}" ${P!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function Y(p){return`
      <div class="card">
        <h3>Services</h3>
        ${te("exec",p.execActive)}
        ${te("beacon",p.beaconActive)}
        ${T?`<p class="error small">${a(T)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(s)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(s)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(s)}">Diagnostics →</a>
        </p>
      </div>
    `}function X(){return'<span class="spinner" aria-label="working"></span>'}async function me(p,v){if(B[p]===null){B[p]=v,T=null,S();try{await Qe(s,p,v)}catch(x){T=`${Ce[p]} ${v} failed: ${x instanceof Error?x.message:String(x)}`}B[p]=null,r||S()}}async function be(p,v){const x=await Se(v),P=p.textContent;p.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{r||(p.textContent=P)},1500)}function pe(p){const v=Ce[p],x=m?le(p==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";Pe(`
        <h2>Clear ${a(v)} data</h2>
        <p class="error">
          This stops the ${a(v.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(x)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(p)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,c=>{if(c==="cancel"){se();return}c==="confirm"&&ie(p)});const P=document.getElementById("clear-confirm-input"),F=document.getElementById("clear-confirm-btn");P==null||P.addEventListener("input",()=>{F&&(F.disabled=P.value.trim()!==p)}),P==null||P.focus()}async function ie(p){const v=document.getElementById("clear-confirm-btn");v&&(v.disabled=!0,v.textContent="Clearing…");try{await et(s,p),se(),i()}catch(x){const P=qe();if(P){const F=document.createElement("p");F.className="error small",F.textContent=`Clear failed: ${x instanceof Error?x.message:String(x)}`,P.appendChild(F)}v&&(v.disabled=!1,v.textContent="Clear and resync")}}return()=>{r=!0,e==null||e(),se()}}const Ne=500,Ue="valve-node-app.explain-consent";function yt(t,s){let r=!1,e=null;const l=[];t.innerHTML=`
    <h1>Logs: ${a(s)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${G()}</div>
  `;const h=t.querySelector("#logs-body"),y=t.querySelector("#logs-footer");ue(t,b=>{b==="explain"&&I()}),m();async function m(){let b,i;try{const[w,S]=await Promise.all([fe(),he()]);b=w.find(j=>j.id===s),i=S}catch(w){if(r)return;h.innerHTML=`<p class="error">Failed to load target: ${a(String(w))}</p>`;return}if(r)return;if(!b){h.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!b.wire){h.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const u=i==null?void 0:i.networks.find(w=>w.ChainID===b.wire.ChainID);u&&(y.innerHTML=G(u.Name,u.LearnURL));try{const w=await Ze(s,200);if(r)return;l.push(...w)}catch(w){if(r)return;h.innerHTML=`<p class="error">Failed to load logs: ${a(String(w))}</p>`;return}E(),e=Xe(s,w=>{r||(l.push(w),l.length>Ne&&l.splice(0,l.length-Ne),E())})}function E(){const b=l.filter(u=>u.severity==="error"||u.severity==="critical");h.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(A).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${H(String(b.length),b.length?"bad":"neutral")}</h2>
          <div class="log-lines">${b.length?b.slice().reverse().map(A).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const i=h.querySelector(".log-lines");i&&(i.scrollTop=i.scrollHeight)}function A(b){const i=b.severity||"info",u=b.learnUrl?` <a href="${a(b.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(i)}">
        <span class="log-time">${a(new Date(b.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(b.unit)}</span>
        <span class="log-sev">${a(i)}</span>
        <span class="log-text">${a(b.line)}</span>
        ${b.explain?`<div class="log-explain">${a(b.explain)}${u}</div>`:""}
      </div>
    `}async function I(){const b=l.filter(u=>u.severity==="error"||u.severity==="critical").map(u=>u.line).slice(-40);if(!(localStorage.getItem(Ue)==="1")){B(b);return}await T(b)}function B(b){const i=b.length?`<pre class="explain-excerpt">${b.map(u=>a(u)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';R(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${i}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,u=>{u==="proceed"?(localStorage.setItem(Ue,"1"),$(),T(b)):$()})}async function T(b){R('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const i=b.length?await He(s,b):await He(s);if(r)return;R(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(i.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${i.sentExcerpt.map(u=>a(u)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&$()})}catch(i){if(r)return;if(i instanceof ye&&i.status===409){R(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,u=>{u==="close"&&$()});return}R(`
        <h2>Explain failed</h2>
        <p class="error">${a(i instanceof Error?i.message:String(i))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&$()})}}function R(b,i){$();const u=document.createElement("div");u.className="modal-overlay",u.id="explain-modal",u.innerHTML=`<div class="modal">${b}</div>`,u.addEventListener("click",w=>{const S=w.target.closest("[data-modal-action]");S!=null&&S.dataset.modalAction&&i(S.dataset.modalAction),w.target===u&&i("cancel")}),document.body.appendChild(u)}function $(){var b;(b=document.getElementById("explain-modal"))==null||b.remove()}return()=>{r=!0,e==null||e(),$()}}function $t(t,s){let r=!1,e=null,l=null,h=!1,y=!1;t.innerHTML=`<h1>Network diagnostics: ${a(s)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${G()}</div>`;const m=t.querySelector("#diag-body"),E=t.querySelector("#diag-footer");ue(t,(i,u)=>{var w;if(i==="run")I();else if(i==="toggle")(w=u.closest(".check-item"))==null||w.classList.toggle("expanded");else if(i==="copy"){const S=u.dataset.copy;S&&b(u,S)}}),A();async function A(){let i,u;try{const[S,j]=await Promise.all([fe(),he()]);i=S.find(K=>K.id===s),u=j}catch(S){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${a(String(S))}</p>`;return}if(r)return;if(!i){m.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const w=u==null?void 0:u.networks.find(S=>S.ChainID===i.wire.ChainID);w&&(E.innerHTML=G(w.Name,w.LearnURL));try{e=await st(s),y=!0}catch(S){l=String(S instanceof Error?S.message:S)}r||B()}async function I(){h=!0,l=null,B();try{e=await rt(s),y=!0}catch(i){l=String(i instanceof Error?i.message:i)}h=!1,r||B()}function B(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${h?"disabled":""}>${h?"Running…":"Run diagnostics"}</button>
      </div>
      ${l?`<p class="error">${a(l)}</p>`:""}
      ${T()}
    `}function T(){if(!y&&!l)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const i=new Date(e.at).toLocaleString(),u=e.failedId?`<p><strong>Failed at: ${a(R(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${a(i)} — trigger: ${a(e.trigger)}</p>
      ${u}
      <ul class="check-list">${e.items.map($).join("")}</ul>
    `}function R(i){var u;return((u=e==null?void 0:e.items.find(w=>w.ID===i))==null?void 0:u.Title)??i}function $(i){const u=i.Status==="pass"?"ok":i.Status==="fail"?"bad":i.Status==="warn"?"warn":"neutral",w=i.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${w?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${H(w?"failed here":i.Status,u)}
          <strong>${a(i.Title)}</strong>
          <span class="muted small check-detail-inline">${a(i.Detail)}</span>
        </button>
        <div class="check-body">
          <details${w?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(i.Why)}</p>
          </details>
          ${i.Fix?`
                <details${w?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(i.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(i.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function b(i,u){const w=await Se(u),S=i.textContent;i.textContent=w?"Copied!":"Copy failed",setTimeout(()=>{r||(i.textContent=S)},1500)}return()=>{r=!0}}function wt(t,s){let r=!1,e=[],l=null,h=!1,y=!1;t.innerHTML=`<h1>Security: ${a(s)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${G()}</div>`;const m=t.querySelector("#sec-body"),E=t.querySelector("#sec-footer");ue(t,($,b)=>{var i;if($==="rerun")I();else if($==="toggle")(i=b.closest(".check-item"))==null||i.classList.toggle("expanded");else if($==="copy"){const u=b.dataset.copy;u&&R(b,u)}}),A();async function A(){let $,b;try{const[u,w]=await Promise.all([fe(),he()]);$=u.find(S=>S.id===s),b=w}catch(u){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${a(String(u))}</p>`;return}if(r)return;if(!$){m.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!$.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const i=b==null?void 0:b.networks.find(u=>u.ChainID===$.wire.ChainID);i&&(E.innerHTML=G(i.Name,i.LearnURL)),await I()}async function I(){h=!0,l=null,B();try{e=await at(s),y=!0}catch($){l=String($ instanceof Error?$.message:$)}h=!1,r||B()}function B(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${h?"disabled":""}>${h?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${a(l)}</p>`:""}
      ${!y&&h?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(T).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function T($){const b=$.Status==="pass"?"ok":$.Status==="fail"?"bad":$.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${H($.Status,b)}
          <strong>${a($.Title)}</strong>
          <span class="muted small check-detail-inline">${a($.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${a($.Why)}</p>
          </details>
          ${$.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a($.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a($.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function R($,b){const i=await Se(b),u=$.textContent;$.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{r||($.textContent=u)},1500)}return()=>{r=!0}}const kt=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function xt(t){let s=!1,r=!1,e=!1,l=null,h=!1,y=null,m=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${G()}`;const E=t.querySelector("#settings-body");ue(t,T=>{if(T==="save"&&B(),T==="clear-key"){if(!y)return;r=!0;const R=t.querySelector("#ai-key");R&&(R.value=""),I(y)}}),Fe(t,(T,R)=>{T!=="ai-provider"||!y||(m=R,h=!1,I(y))}),A();async function A(){try{const T=await ut();if(s)return;y=T,I(T)}catch(T){if(s)return;E.innerHTML=`<p class="error">Failed to load settings: ${a(String(T))}</p>`}}function I(T){var b;const R=m??T.aiProvider;E.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Re("ai-provider",kt.map(i=>({value:i.value,label:i.label})),R)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${T.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${T.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${a(T.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${l?`<p class="error">${a(l)}</p>`:""}
        ${h?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const $=t.querySelector("#ai-key");$==null||$.addEventListener("input",()=>{r=!0,h=!1}),(b=t.querySelector("#ref-rpc-base"))==null||b.addEventListener("input",()=>{h=!1})}async function B(){const T=t.querySelector("#ai-key"),R=t.querySelector("#ref-rpc-base");if(!T||!R||!y)return;const $={aiProvider:m??y.aiProvider,refRpcBase:R.value.trim()};r&&($.aiKey=T.value),e=!0,l=null,h=!1,I(y);try{const b=await pt($);if(s)return;y=b,r=!1,e=!1,h=!0,I(b)}catch(b){if(s)return;e=!1,l=String(b instanceof Error?b.message:b),I(y)}}return()=>{s=!0}}const Tt="run",Pt={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container.",erpc:"One eRPC instance in front of however many chains you list. It addresses a chain by URL path, so a single port serves all of them — and the same path serves WebSocket."},St={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Et(t,s){let r=!1,e=null,l=null;const h={devnet:null,erpc:null},y={devnet:null,erpc:null},m={devnet:[],erpc:[]};let E=null;const A={devnet:!1,erpc:!1};let I=null,B=null;const T={devnet:null,erpc:null},R={devnet:null,erpc:null};t.innerHTML=`
    <div class="page-head">
      <h1>Services: ${a(s)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      Containers this machine hosts. They are independent of any node setup —
      a machine can run a devnet, a gateway, both, or neither.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${G()}
  `;const $=t.querySelector("#services-body");ue(t,(n,o)=>{F(n,o)}),b();async function b(){try{const n=await ot(s);if(r)return;e=n,l=null}catch(n){if(r)return;e=null,l=C(n)}u()}function i(n){return e==null?void 0:e.services.find(o=>o.id===n)}function u(){if(!r){if(l){$.innerHTML=`<p class="error">Could not read this machine's services: ${a(l)}</p>`;return}if(!e){$.innerHTML='<p class="muted">Loading…</p>';return}$.innerHTML=`
      ${w(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(S).join("")}
      </div>
    `}}function w(n){if(n.present&&n.reachable&&!n.hint)return`<p class="muted small">Docker: ${a(n.flavor)}${n.serverVersion?` ${a(n.serverVersion)}`:""} · reachable</p>`;const o=n.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(o)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${n.detail?`<div class="small">${a(n.detail)}</div>`:""}
        ${n.hint?`<div class="small">${a(n.hint)}</div>`:""}
      </div>
    `}function S(n){const o=n.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(n.label)}</h2>
          ${j(n)}
        </div>
        <p class="muted small">${a(Pt[n.id]??"")}</p>

        ${n.error?K(n):""}
        ${n.blocked?`<div class="banner banner-warn">${a(n.blocked)}</div>`:""}
        ${o.map(f=>`<div class="banner banner-warn">${a(f)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(n.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${n.status.Image?`<code>${a(n.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${W(n)}

        ${Z(n)}

        <div class="card-actions">
          ${(n.actions??[]).map(f=>ne(n,f)).join("")}
        </div>
        ${y[n.id]?`<p class="error small">${a(y[n.id])}</p>`:""}
        ${ae(n)}

        ${oe(n)}
      </div>
    `}function j(n){switch(n.status.State){case"running":return H("running","ok");case"created-but-stopped":return H("stopped","warn");case"not-created":return H("not created","neutral");default:return H("unknown","bad")}}function K(n){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(n.error??"")}</div>
        ${n.hint?`<div class="small">${a(n.hint)}</div>`:""}
      </div>
    `}function W(n){if(n.status.State!=="created-but-stopped"||n.status.ExitCode===0)return"";const o=n.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${n.status.ExitCode}${o}.</p>`}function Z(n){const o=n.endpoints??[];return o.length===0?n.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":o.map(f=>`
        <div class="endpoint-row">
          ${Le("ok")}
          <span class="muted small">${a(f.label)}</span>
          <code class="endpoint-url">${a(f.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(f.url)}">Copy</button>
        </div>`).join("")}function ne(n,o){const f=St[o];if(!f)return"";const k=h[n.id],U=o==="create"?`Create ${n.id==="devnet"?"devnet":"gateway"}`:f.label;return`
      <button class="${f.className}" data-action="svc-${o}" data-svc="${a(n.id)}"
              title="${a(f.title)}" ${k?"disabled":""}>
        ${k===o?'<span class="spinner" aria-label="working"></span>':a(U)}
      </button>
    `}function ae(n){const o=m[n.id]??[];return o.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(o.join(`
`))}</pre>
      </div>
    `}function oe(n){const o=A[n.id],f=n.id==="devnet"?te(n):Y(n);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${n.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(n.id)}">
            ${o?"Close":"Edit"}
          </button>
        </div>
        ${o?X(n):`<p class="small">${f}</p>`}
        ${T[n.id]?`<p class="error small">${a(T[n.id])}</p>`:""}
        ${R[n.id]?`<p class="muted small">${a(R[n.id])}</p>`:""}
      </div>
    `}function te(n){const o=n.devnet;return o?`Chain ${o.ChainID} · a block every ${a(o.BlockTime)} · JSON-RPC on ${a(o.BindAddr)}:${o.HTTPPort} · WebSocket on ${a(o.BindAddr)}:${o.WSPort}`:"—"}function Y(n){const o=n.gateway;if(!o)return"—";const f=o.Networks??[];return f.length===0?`Listening on ${a(o.BindAddr)}:${o.Port} · no chains yet`:`Listening on ${a(o.BindAddr)}:${o.Port} · ${f.map(k=>`chain ${k.ChainID} (${k.Upstreams.length} upstream${k.Upstreams.length===1?"":"s"})`).join(", ")}`}function X(n){return n.id==="devnet"?me():be()}function me(){const n=I;return n?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${a(n.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${n.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${n.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${a(n.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${n.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function be(){var k;const n=B;if(!n)return"";const o=n.Networks??[],f=(k=i("devnet"))==null?void 0:k.devnet;return`
      <label>
        Listen port
        <input type="text" inputmode="numeric" id="gw-port" value="${n.Port}" autocomplete="off" />
      </label>
      <label>
        Bind address
        <input type="text" id="gw-bind" value="${a(n.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        Requests are addressed by path: <code>/${a(n.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
        chain below, and the same path serves WebSocket with a <code>ws://</code> scheme.
      </p>
      ${o.map(pe).join("")}
      ${o.length===0?'<p class="muted small">No chains yet. A gateway with no chain has nothing to serve, so it cannot be created until you add one.</p>':""}
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="add-network">Add a chain</button>
        ${f?`<button class="btn btn-ghost" data-action="front-devnet" title="Add this machine's devnet as chain ${f.ChainID}'s upstream">Front the local devnet</button>`:""}
        <button class="btn" data-action="save-config" data-svc="erpc">Save configuration</button>
      </div>
    `}function pe(n,o){return`
      <div class="config-block">
        <div class="service-head">
          <label class="inline-label">
            Chain id
            <input type="text" inputmode="numeric" id="gw-net-${o}-chain" value="${n.ChainID}" autocomplete="off" />
          </label>
          <button class="btn btn-ghost" data-action="remove-network" data-net="${o}">Remove chain</button>
        </div>
        ${n.Upstreams.map((f,k)=>ie(f,o,k)).join("")}
        ${n.Upstreams.length===0?'<p class="muted small">This chain has no upstream, so the gateway has nowhere to send its calls.</p>':""}
        <button class="btn btn-ghost" data-action="add-upstream" data-net="${o}">Add an upstream</button>
      </div>
    `}function ie(n,o,f){return`
      <div class="upstream-row">
        <label class="inline-label">
          Name
          <input type="text" id="gw-up-${o}-${f}-id" value="${a(n.ID)}" autocomplete="off" spellcheck="false" />
        </label>
        <label class="inline-label grow">
          Endpoint
          <input type="text" id="gw-up-${o}-${f}-endpoint" value="${a(n.Endpoint)}" autocomplete="off" spellcheck="false" placeholder="http://127.0.0.1:8545" />
        </label>
        <label class="radio">
          <input type="checkbox" id="gw-up-${o}-${f}-local" ${n.Local?"checked":""} />
          Mine — prefer it
        </label>
        <label class="radio">
          <input type="checkbox" id="gw-up-${o}-${f}-recent" ${n.RecentOnly?"checked":""} />
          Recent blocks only
        </label>
        <button class="btn btn-ghost" data-action="remove-upstream" data-net="${o}" data-up="${f}">Remove</button>
      </div>
    `}function p(){A.devnet&&I&&(I.BlockTime=v("#dev-blocktime",I.BlockTime),I.HTTPPort=x("#dev-http",I.HTTPPort),I.WSPort=x("#dev-ws",I.WSPort),I.BindAddr=v("#dev-bind",I.BindAddr)),A.erpc&&B&&(B.Port=x("#gw-port",B.Port),B.BindAddr=v("#gw-bind",B.BindAddr),(B.Networks??[]).forEach((o,f)=>{o.ChainID=x(`#gw-net-${f}-chain`,o.ChainID),o.Upstreams.forEach((k,U)=>{k.ID=v(`#gw-up-${f}-${U}-id`,k.ID),k.Endpoint=v(`#gw-up-${f}-${U}-endpoint`,k.Endpoint),k.Local=P(`#gw-up-${f}-${U}-local`,k.Local),k.RecentOnly=P(`#gw-up-${f}-${U}-recent`,k.RecentOnly)})}))}function v(n,o){const f=t.querySelector(n);return f?f.value.trim():o}function x(n,o){const f=t.querySelector(n);if(!f)return o;const k=Number.parseInt(f.value.trim(),10);return Number.isFinite(k)?k:o}function P(n,o){const f=t.querySelector(n);return f?f.checked:o}async function F(n,o){const f=o.dataset.svc??"";switch(n){case"refresh":await b();return;case"copy":o.dataset.copy&&await Q(o,o.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await c(f,n.slice(4));return;case"svc-create":case"svc-recreate":await d(f);return;case"svc-wipe":M(f);return;case"toggle-config":g(f);return;case"save-config":await D(f);return;case"add-network":case"remove-network":case"add-upstream":case"remove-upstream":case"front-devnet":L(n,o);return;default:return}}async function c(n,o){if(!h[n]){h[n]=o,y[n]=null,u();try{await it(s,n,o)}catch(f){y[n]=`${o} failed: ${C(f)}${z(f)}`}h[n]=null,await b()}}async function d(n){if(!h[n]){h[n]="create",y[n]=null,m[n]=["starting…"],u();try{await lt(s,n)}catch(o){y[n]=`${C(o)}${z(o)}`,m[n]=[],h[n]=null,u();return}E==null||E(),E=Oe(s,o=>{if(r)return;const f=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(m[n]=[...(m[n]??[]).filter(U=>U!=="starting…"),f],!!o.err||o.stepId===Tt&&!!o.done){E==null||E(),E=null,h[n]=null,o.err&&(y[n]="Provisioning failed — see the log below."),b();return}u()})}}function g(n){if(p(),A[n]=!A[n],T[n]=null,R[n]=null,A[n]){const o=i(n);n==="devnet"&&(o!=null&&o.devnet)&&(I={...o.devnet}),n==="erpc"&&(o!=null&&o.gateway)&&(B={...o.gateway,Networks:(o.gateway.Networks??[]).map(f=>({ChainID:f.ChainID,Upstreams:f.Upstreams.map(k=>({...k}))}))})}u()}function L(n,o){var q;if(!B)return;p();const f=B.Networks??[],k=Number.parseInt(o.dataset.net??"",10),U=Number.parseInt(o.dataset.up??"",10);switch(n){case"add-network":f.push({ChainID:1,Upstreams:[{ID:"",Endpoint:"",Local:!1,RecentOnly:!1}]});break;case"remove-network":Number.isFinite(k)&&f.splice(k,1);break;case"add-upstream":Number.isFinite(k)&&f[k]&&f[k].Upstreams.push({ID:"",Endpoint:"",Local:!1,RecentOnly:!1});break;case"remove-upstream":Number.isFinite(k)&&Number.isFinite(U)&&f[k]&&f[k].Upstreams.splice(U,1);break;case"front-devnet":{const _=(q=i("devnet"))==null?void 0:q.devnet;if(!_)break;const V=`http://${_.BindAddr==="0.0.0.0"?"127.0.0.1":_.BindAddr}:${_.HTTPPort}`,ge=f.find(je=>je.ChainID===_.ChainID);ge?ge.Upstreams=[{ID:"devnet",Endpoint:V,Local:!0,RecentOnly:!1}]:f.push({ChainID:_.ChainID,Upstreams:[{ID:"devnet",Endpoint:V,Local:!0,RecentOnly:!1}]});break}default:return}B.Networks=f,u()}async function D(n){var k;p(),T[n]=null,R[n]=null;const o=n==="devnet"?I:B;if(!o)return;if(n==="devnet"&&I&&I.HTTPPort===I.WSPort){T[n]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await dt(s,n,o)}catch(U){T[n]=C(U),u();return}const f=((k=i(n))==null?void 0:k.status.State)==="running";A[n]=!1,R[n]=f?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await b()}function M(n){const o=i(n);if(!o)return;const f=(o.restartsOnWipe??[]).map(q=>{var _;return((_=i(q))==null?void 0:_.label)??q});Pe(`
        <h2>Wipe ${a(o.label)}</h2>
        <p class="error">This deletes ${a(o.wipeDiscards)}</p>
        ${f.length?`<p>It also restarts what sits in front of it: ${a(f.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${a(n)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(n)}</button>
        </div>
      `,q=>{if(q==="cancel"||q==="close"){se(),b();return}q==="confirm"&&N(n)});const k=document.getElementById("wipe-confirm-input"),U=document.getElementById("wipe-confirm-btn");k==null||k.addEventListener("input",()=>{U&&(U.disabled=k.value.trim()!==n)}),k==null||k.focus()}async function N(n){const o=document.getElementById("wipe-confirm-btn");o&&(o.disabled=!0,o.textContent="Wiping…");let f;try{f=await ct(s,n)}catch(k){const U=qe();if(U){const q=document.createElement("p");q.className="error small",q.textContent=`Wipe failed: ${C(k)}${z(k)}`,U.appendChild(q)}o&&(o.disabled=!1,o.textContent=`Wipe ${n}`);return}J(n,f)}function J(n,o){const f=i(n),k=V=>{var ge;return((ge=i(V))==null?void 0:ge.label)??V},U=[];U.push(o.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const V of o.report.VolumesRemoved??[])U.push(`Volume ${V} deleted.`);for(const V of o.report.VolumesAbsent??[])U.push(`Volume ${V} was already gone.`);o.report.Recreated&&U.push("Container re-created from your saved configuration.");const q=(o.report.Cascaded??[]).map(k),_=(o.report.CascadeSkipped??[]).map(k);Pe(`
        <h2>${a((f==null?void 0:f.label)??n)} wiped</h2>
        <ul class="plain-list">${U.map(V=>`<li>${a(V)}</li>`).join("")}</ul>
        ${q.length?`<p class="ok">Restarted in front of it: ${a(q.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${_.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(_.join(", "))}.</p>`:""}
        ${o.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(o.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,V=>{(V==="close"||V==="cancel")&&(se(),b())})}async function Q(n,o){const f=await Se(o),k=n.textContent;n.textContent=f?"Copied!":"Copy failed",setTimeout(()=>{r||(n.textContent=k)},1500)}function C(n){return n instanceof Error?n.message:String(n)}function z(n){return n instanceof ye&&n.hint?` — ${n.hint}`:""}return()=>{r=!0,E==null||E(),se()}}const Ct="local";function It(t){let s=!1,r=!1,e="",l=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${G()}
  `;const h=t.querySelector("#targets-body");ue(t,(i,u)=>{I(i,u)}),y();async function y(){try{const[i,u,w]=await Promise.all([fe(),he(),Ke()]);if(s)return;e=w.os,E(i,u)}catch(i){if(s)return;h.innerHTML=`<p class="error">Failed to load machines: ${a(String(i))}</p>`}}function m(){l&&E(l.targets,l.catalog)}function E(i,u){l={targets:i,catalog:u};const w=e==="linux",S=[...i].sort((W,Z)=>(W.mode==="local"?-1:0)-(Z.mode==="local"?-1:0)),j=S.length?`<div class="card-grid">${S.map(W=>Lt(W,u,W.mode!=="local"||w,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',K=i.some(W=>W.mode==="local");h.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${j}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${A(w,K)}
        ${r?Rt():""}
      </section>
    `}function A(i,u){const w=`
      <div class="card">
        <h3>A server over SSH ${H("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${i?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${i?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,S=i?`
        <div class="card">
          <h3>This machine ${H("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${a(e)})`:""} ${H("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return u?`<div class="card-grid card-grid-wide">${w}</div>`:`<div class="card-grid card-grid-wide">${i?S+w:w+S}</div>`}async function I(i,u){var w;if(i==="add-local"){await B();return}if(i==="delete-target"){const S=u.dataset.id;if(!S||!await bt({title:"Remove machine",body:`Remove "${S}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove"}))return;await T(S);return}if(i==="toggle-ssh"){r=!r,b(),m(),r&&((w=t.querySelector("#ssh-host"))==null||w.focus());return}i==="add-ssh"&&await R()}async function B(){b();try{await Be({id:Ct,mode:"local"}),await y()}catch(i){$(i)}}async function T(i){try{await Je(i),await y()}catch(u){$(u)}}async function R(){const i=t.querySelector("#ssh-host"),u=t.querySelector("#ssh-user"),w=t.querySelector("#ssh-key"),S=t.querySelector("#ssh-port"),j=t.querySelector("#ssh-id");if(!i||!u||!w||!S||!j)return;const K=i.value.trim(),W=u.value.trim(),Z=w.value.trim(),ne=S.value.trim(),ae=j.value.trim();if(b(),!K||!W||!Z){$(new Error("host, user, and key path are required"));return}const oe=ae||Bt(K),te={Host:K,User:W,KeyPath:Z};if(ne){const X=Number.parseInt(ne,10);if(!Number.isFinite(X)||X<=0){$(new Error("port must be a positive number"));return}te.Port=X}const Y=t.querySelector("#ssh-submit");Y&&(Y.disabled=!0,Y.textContent="Connecting…");try{await Be({id:oe,mode:"ssh",ssh:te}),r=!1,await y()}catch(X){$(X),Y&&(Y.disabled=!1,Y.textContent="Add server")}}function $(i){let u=t.querySelector("#targets-error");u||(h.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),u=t.querySelector("#targets-error")),u.textContent=String(i instanceof Error?i.message:i)}function b(){var i;(i=t.querySelector("#targets-error"))==null||i.remove()}return()=>{s=!0}}function Lt(t,s,r,e){const l=t.wire,h=t.mode==="local"?"this machine":"SSH",y=t.mode==="ssh"&&t.ssh?`${a(t.ssh.User)}@${a(t.ssh.Host)}`:h,m=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(t.id)}">Devnet &amp; gateway</a>`;let E,A;if(!l&&!r)E=`${H("can't run a node","warn")} ${H(e||"not Linux","neutral")}`,A=`
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Preview setup wizard</a>
    `;else if(!l)E=H("not set up","neutral"),A=`
      <a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>
      ${m}
    `;else{const I=s.networks.find(T=>T.ChainID===l.ChainID),B=I?I.Name:`chain ${l.ChainID}`;E=`${H(B,"ok")} ${H(l.ExecID,"neutral")} ${H(l.BeaconID,"neutral")}${l.Archive?" "+H("archive","warn"):""}`,A=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${a(t.id)}</h2>
      <p class="muted">${y}</p>
      <p>${E}</p>
      <div class="card-actions">
        ${A}
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(t.id)}">Remove</button>
      </div>
    </div>
  `}function Rt(){return`
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
  `}function Bt(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Ie=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],we=8545,ke=5052,xe=30303,Ht=[369,943,1],Me={369:"default",943:"practise here first"};function At(t,s){let r=!1;const e={targetId:s,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${a(s)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${G()}</div>`;const l=t.querySelector("#wizard-body"),h=t.querySelector("#wizard-footer");ue(t,(c,d)=>{be(c,d)}),Fe(t,(c,d)=>{c==="exec-select"?e.execId=d:c==="beacon-select"&&(e.beaconId=d),m()}),t.addEventListener("change",c=>{const d=c.target;d instanceof HTMLInputElement&&(d.id==="data-dir-input"?(pe(),Z()):d.id==="checkpoint-toggle"?(e.checkpoint=d.checked,m()):d.id==="exec-snapshot-toggle"&&(e.execSnapshot=d.checked,m()))}),y();async function y(){try{const[c,d]=await Promise.all([he(),fe()]);if(r)return;e.catalog=c;const g=d.find(L=>L.id===s);g!=null&&g.wire&&(e.chainId=g.wire.ChainID,e.execId=g.wire.ExecID,e.beaconId=g.wire.BeaconID,e.archive=g.wire.Archive,g.wire.ExecHTTPPort&&(e.execHTTPPort=String(g.wire.ExecHTTPPort)),g.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(g.wire.BeaconHTTPPort)),g.wire.ExecP2PPort&&(e.execP2PPort=String(g.wire.ExecP2PPort)),g.wire.RPCBindAddr&&(e.rpcBindAddr=g.wire.RPCBindAddr)),m()}catch(c){if(r)return;e.loadError=String(c instanceof Error?c.message:c),m()}}function m(){if(e.loadError){l.innerHTML=`<p class="error">Failed to load: ${a(e.loadError)}</p>`;return}e.catalog&&(l.innerHTML=`
      ${F(e.step)}
      ${A()}
    `,E())}function E(){var d;const c=(d=e.catalog)==null?void 0:d.networks.find(g=>g.ChainID===e.chainId);h.innerHTML=c?G(c.Name,c.LearnURL):G()}function A(){switch(e.step){case"network":return I();case"clients":return B();case"mode":return Y();case"review":return X();case"run":return me()}}function I(){const c=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ht.map(g=>{const L=c.networks.find(N=>N.ChainID===g);if(!L)return"";const D=e.chainId===g,M=Me[g]?H(Me[g],g===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${D?"selected":""}" data-action="pick-network" data-chain-id="${g}" type="button">
          <h3>${a(L.Name)} <span class="muted">(chain ${g})</span></h3>
          ${M}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function B(){const c=e.catalog,d=c.networks.find(D=>D.ChainID===e.chainId);if(!d)return'<p class="error">Unknown network.</p>';(e.execId===null||!d.ExecClients.includes(e.execId))&&(e.execId=d.ExecClients[0]??null),(e.beaconId===null||!d.BeaconClients.includes(e.beaconId))&&(e.beaconId=d.BeaconClients[0]??null);const g=d.ExecClients.map(D=>ae(D,c)),L=d.BeaconClients.map(D=>ae(D,c));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${a(d.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${Re("exec-select",g,e.execId)}
        </label>
        ${te(e.execId,c)}
        <label>
          Beacon client
          ${Re("beacon-select",L,e.beaconId)}
        </label>
        ${te(e.beaconId,c)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function T(c){return c<=0?"—":c>=1?`~${c.toFixed(1)} TB`:`~${Math.round(c*1e3)} GB`}const R=1.1,$=.5,b="Valve reth snapshot",i="rough estimate";function u(c){return c.SnapshotSizeTB}function w(c){return c.SnapshotSizeTB*$}function S(c){return`<p class="muted small">${T(u(c))} is the measured size of Valve's reth snapshot for ${a(c.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function j(c){return{archive:u(c)*1e12*R,full:w(c)*1e12*R}}function K(c,d){if(!c)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${a(d)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${a(d)}</code>: ${a(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==d)return"";const g=j(c),L=e.freeBytes>=g.archive,D=e.freeBytes>=g.full,M=`<p class="muted small">Free at <code>${a(d)}</code>: <strong>${le(e.freeBytes)}</strong> — archive ${L?"fits":"won't fit"} (${T(u(c))}, ${b}), full ${D?"fits":"won't fit"} (${T(w(c))}, ${i}).</p>`;let N="";return e.downgradeNote?N=`<p class="banner banner-warn">${a(e.downgradeNote)}</p>`:D||(N=`<p class="banner banner-warn">Neither full (${T(w(c))}, ${i}) nor archive (${T(u(c))}, ${b}) fits the free space here — choose a location with more room.</p>`),M+N}function W(c,d){if(e.downgradeNote=null,!c||e.freeBytes===null)return;const g=j(c);e.archive&&e.freeBytes<g.archive&&e.freeBytes>=g.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${d} for archive (${T(u(c))}, ${b}) — switched to Full (${T(w(c))}, ${i}). Pick a location with more room to run archive.`)}async function Z(){var g;if(e.chainId===null)return;const c=(g=e.catalog)==null?void 0:g.networks.find(L=>L.ChainID===e.chainId),d=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,m();try{const{freeBytes:L}=await Ve(e.targetId,d);if(r)return;e.freeBytes=L,e.probedPath=d,W(c,d)}catch(L){if(r)return;e.freeBytes=null,e.probedPath=d,e.diskError=String(L instanceof Error?L.message:L)}e.diskProbing=!1,m()}function ne(c){return c?/^https?:\/\/.+/i.test(c)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ae(c,d){const g=d.clients.find(L=>L.id===c);return{value:c,label:g?`${g.id} — ${oe(g.repo)}`:c}}function oe(c){const d=c.split("/");return d.length>=4?d[3]:c}function te(c,d){const g=c?d.clients.find(D=>D.id===c):void 0;if(!g)return"";const L=g.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(g.repo)}" target="_blank" rel="noopener noreferrer">${a(L)}</a></p>`}function Y(){var J,Q,C;const c=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",d=(J=e.catalog)==null?void 0:J.networks.find(z=>z.ChainID===e.chainId),g=((C=(Q=e.catalog)==null?void 0:Q.clients.find(z=>z.id===e.execId))==null?void 0:C.snapshotSupported)??!1,L=d?`${T(w(d))} (${i})`:"Smaller",D=d?`${T(u(d))} (${b})`:"Much larger",M=d?` on ${a(d.Name)}`:"",N=d?e.checkpoint?d.SyncLabel:d.GenesisSyncLabel:"";return`
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
          ${d?`<p class="sync-estimate">⏱ Estimated initial sync${M}: <strong>${a(N)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${a((d==null?void 0:d.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${a((d==null?void 0:d.CheckpointURL)??"")}" value="${a(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${a(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${g?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${e.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${e.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${a(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${a(e.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${M}</th><td class="yes">${L}</td><td class="limited">${D}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${d?S(d):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${D}${d?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${L}${d?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(c)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(c)}" value="${a(e.dataDir)}" />
          </label>
          ${K(d,e.dataDir||c)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(c)}/jwt.hex" value="${a(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${we})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${we}" value="${a(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${a(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${ke})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ke}" value="${a(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${a(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${xe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${xe}" value="${a(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${a(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${a(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${a(e.rpcBindAddrError)}</p>`:""}
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
    `}function X(){const d=e.catalog.networks.find(n=>n.ChainID===e.chainId),g=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,L=e.jwtPath||`${g}/jwt.hex`,D=Ie.map(n=>`<li>${a(n.title)}</li>`).join(""),M=x(e.execHTTPPort,we),N=x(e.beaconHTTPPort,ke),J=x(e.execP2PPort,xe),Q=M||N||J?`<tr><th>Non-default ports</th><td>${[M?`exec HTTP ${M}`:null,N?`beacon HTTP ${N}`:null,J?`exec p2p ${J}`:null].filter(n=>n!==null).map(a).join(", ")}</td></tr>`:"",{addr:C}=ie(e.rpcBindAddr),z=C?`<tr><th>RPC bind address</th><td><code>${a(C)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${a(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${a((d==null?void 0:d.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${a(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${a(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${a(g)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${a(L)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${a(e.checkpointUrl||(d==null?void 0:d.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${Q}
            ${z}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${D}</ol>
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function me(){const d=e.catalog.networks.find(C=>C.ChainID===e.chainId),g=d==null?void 0:d.LearnURL,L=new Set(e.events.filter(C=>C.done).map(C=>C.stepId)),D=new Set(e.events.filter(C=>C.err).map(C=>C.stepId)),M=new Map;for(const C of e.events){if(!C.line)continue;const z=M.get(C.stepId)??[];z.push(C.line),M.set(C.stepId,z)}const N=Ie.map(C=>{var q;const z=L.has(C.id),n=D.has(C.id),o=n?H("failed","bad"):z?H("done","ok"):H("pending","neutral"),f=(M.get(C.id)??[]).slice(-5),k=(q=e.events.find(_=>_.stepId===C.id&&_.err))==null?void 0:q.err,U=C.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${g?` <a href="${a(g)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${z?"step-done":""} ${n?"step-error":""}">
          <div class="step-head">${o} <strong>${a(C.title)}</strong></div>
          ${U}
          ${f.length?`<pre class="step-log">${f.map(_=>a(_)).join(`
`)}</pre>`:""}
          ${k?`<p class="error small">${a(k)}</p>`:""}
        </li>
      `}).join(""),J=e.events.some(C=>C.err),Q=Ie.every(C=>L.has(C.id))||e.events.some(C=>C.stepId==="handshake"&&C.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${Q&&!J?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        ${J?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(c,d){switch(c){case"pick-network":e.chainId=Number(d.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m(),Z();break;case"goto-review":if(pe(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){m();break}e.step="review",m();break;case"start-setup":P();break}}function pe(){const c=t.querySelectorAll('input[name="mode"]');for(const C of Array.from(c))C.checked&&(e.archive=C.value==="archive");const d=t.querySelector("#data-dir-input"),g=t.querySelector("#jwt-path-input");d&&(e.dataDir=d.value.trim()),g&&(e.jwtPath=g.value.trim());const L=t.querySelector("#exec-http-port-input"),D=t.querySelector("#beacon-http-port-input"),M=t.querySelector("#exec-p2p-port-input");L&&(e.execHTTPPort=L.value.trim()),D&&(e.beaconHTTPPort=D.value.trim()),M&&(e.execP2PPort=M.value.trim());const N=t.querySelector("#rpc-bind-addr-input");N&&(e.rpcBindAddr=N.value.trim());const J=t.querySelector("#checkpoint-url-input");J&&(e.checkpointUrl=J.value.trim());const Q=t.querySelector("#snapshot-key-input");Q&&(e.snapshotKey=Q.value.trim()),e.execHTTPPortError=v(e.execHTTPPort).error??null,e.beaconHTTPPortError=v(e.beaconHTTPPort).error??null,e.execP2PPortError=v(e.execP2PPort).error??null,e.rpcBindAddrError=ie(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?ne(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function ie(c){if(!c)return{};const d=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(c);return d?d.slice(1).every(g=>Number(g)<=255)?{addr:c}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(c)&&c.includes(":")?{addr:c}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const p=/^\d+$/;function v(c){if(!c)return{};if(!p.test(c))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const d=Number(c);return!Number.isInteger(d)||d<1||d>65535?{error:"Port must be between 1 and 65535."}:{port:d}}function x(c,d){const{port:g}=v(c);if(!(g===void 0||g===d))return g}async function P(){var M;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(M=e.streamStop)==null||M.call(e),e.streamStop=null,m();const c={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(c.DataDir=e.dataDir),e.jwtPath&&(c.JWTPath=e.jwtPath);const d=x(e.execHTTPPort,we),g=x(e.beaconHTTPPort,ke),L=x(e.execP2PPort,xe);d!==void 0&&(c.ExecHTTPPort=d),g!==void 0&&(c.BeaconHTTPPort=g),L!==void 0&&(c.ExecP2PPort=L);const{addr:D}=ie(e.rpcBindAddr);D!==void 0&&(c.RPCBindAddr=D),e.checkpoint?e.checkpointUrl&&(c.CheckpointURL=e.checkpointUrl):c.NoCheckpoint=!0,e.execSnapshot&&(c.ExecSnapshot=!0,c.SnapshotKey=e.snapshotKey);try{await Ge(e.targetId,c)}catch(N){if(!(N instanceof ye&&N.status===409)){e.starting=!1,e.startError=String(N instanceof Error?N.message:N),m();return}}e.starting=!1,e.step="run",m(),e.streamStop=Oe(e.targetId,N=>{r||(e.events.push(N),e.step==="run"&&m())})}function F(c){const d=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],L=d.map(D=>D.id).indexOf(c);return`
      <ol class="wizard-progress">
        ${d.map((D,M)=>`<li class="${M===L?"current":M<L?"past":"future"}">${a(D.label)}</li>`).join("")}
      </ol>
    `}return()=>{var c;r=!0,(c=e.streamStop)==null||c.call(e)}}const Dt=document.querySelector("#app"),{contentEl:Nt,setActiveNav:Ut}=ht(Dt);let ee=null;function Mt(){const s=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(s.length===0)return{screen:"targets"};const[r,e]=s;return r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"?{screen:r,id:e?decodeURIComponent(e):void 0}:{screen:r??"targets"}}function re(t){const s=document.createElement("div");return Nt.replaceChildren(s),t(s)}function _e(){if(ee){try{ee()}catch{}ee=null}const{screen:t,id:s}=Mt();switch(Ut(t),t){case"setup":if(!s){location.hash="#/targets";return}ee=re(r=>At(r,s));break;case"dash":if(!s){location.hash="#/targets";return}ee=re(r=>vt(r,s));break;case"logs":if(!s){location.hash="#/targets";return}ee=re(r=>yt(r,s));break;case"security":if(!s){location.hash="#/targets";return}ee=re(r=>wt(r,s));break;case"diag":if(!s){location.hash="#/targets";return}ee=re(r=>$t(r,s));break;case"services":if(!s){location.hash="#/targets";return}ee=re(r=>Et(r,s));break;case"settings":ee=re(r=>xt(r));break;case"targets":default:ee=re(r=>It(r));break}}window.addEventListener("hashchange",_e);_e();
