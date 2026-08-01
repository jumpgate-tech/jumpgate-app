var wn=Object.defineProperty;var kn=(s,i,r)=>i in s?wn(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var je=(s,i,r)=>kn(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const f of document.querySelectorAll('link[rel="modulepreload"]'))t(f);new MutationObserver(f=>{for(const w of f)if(w.type==="childList")for(const R of w.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&t(R)}).observe(document,{childList:!0,subtree:!0});function r(f){const w={};return f.integrity&&(w.integrity=f.integrity),f.referrerPolicy&&(w.referrerPolicy=f.referrerPolicy),f.crossOrigin==="use-credentials"?w.credentials="include":f.crossOrigin==="anonymous"?w.credentials="omit":w.credentials="same-origin",w}function t(f){if(f.ep)return;f.ep=!0;const w=r(f);fetch(f.href,w)}})();function at(){return z("/api/host")}function Se(){return z("/api/catalog")}function ve(){return z("/api/targets")}function et(s){return z("/api/targets",{method:"POST",headers:be,body:JSON.stringify(s)})}function Sn(s){return z(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Tn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function Cn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:be,body:JSON.stringify(i)})}function Ve(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function xn(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function Pn(s,i=200){return z(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function En(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function ft(s,i){const r=i===void 0?{}:{lines:i};return z(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:be,body:JSON.stringify(r)})}function In(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function Rn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:be,body:JSON.stringify({Confirm:i})})}function Ln(s){return z(`/api/targets/${encodeURIComponent(s)}/du`)}function An(s){return z(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function Nn(s){return z(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Bn(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function Hn(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function $t(s){return z(`/api/targets/${encodeURIComponent(s)}/containers`)}function Dn(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function Un(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:i})}),t=await r.text();let f=null;try{f=t?JSON.parse(t):null}catch{}if(f&&typeof f=="object"&&"report"in f)return f;const w=f&&typeof f=="object"&&typeof f.error=="string"?f.error:r.statusText||`HTTP ${r.status}`;throw new $e(r.status,w)}function Mn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function On(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:be}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const f=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new $e(i.status,f)}function qn(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:be,body:JSON.stringify(r)})}function st(){return z("/api/gateways")}async function Fn(s){await z(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function wt(s){return z("/api/gateways",{method:"POST",headers:be,body:JSON.stringify(s)})}function jn(s){return z(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function Wn(s){return z(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function _n(s){return z(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function Kn(s,i=!1){const r=i?"?refresh=1":"";return z(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function Gn(s){return z(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function kt(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:be,body:JSON.stringify(i)})}function Vn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function zn(s){return z(`/api/gateways/${encodeURIComponent(s)}/trust-cert`,{method:"POST"})}function St(s){return z(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function Jn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:s})}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const f=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new $e(i.status,f)}function Yn(s){return z(`/api/chainlist/${s}`)}function Tt(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function Zn(){return z("/api/settings")}function Xn(s){return z("/api/settings",{method:"PUT",headers:be,body:JSON.stringify(s)})}class $e extends Error{constructor(r,t,f,w){super(t);je(this,"status");je(this,"hint");je(this,"code");this.name="ApiError",this.status=r,this.hint=f,this.code=w}}const be={"Content-Type":"application/json"};async function z(s,i){const r=await fetch(s,i);if(!r.ok){let f=r.statusText||`HTTP ${r.status}`,w,R;try{const m=await r.json();m&&typeof m.error=="string"&&m.error&&(f=m.error),m&&typeof m.hint=="string"&&m.hint&&(w=m.hint),m&&typeof m.code=="string"&&m.code&&(R=m.code)}catch{}throw new $e(r.status,f,w,R)}if(r.status===204)return;const t=await r.text();return t?JSON.parse(t):void 0}const mt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function le(s,i){const r=s&&i&&i!==mt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(mt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function Qn(s){s.innerHTML=`
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/">valve-node-app</a>
        <nav class="nav">
          <a href="#/rpc" data-nav="rpc">RPC</a>
          <a href="#/targets" data-nav="targets">Machines</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:f=>{const w=f==="machine"?"targets":f==="home"?"rpc":f;for(const R of r)R.classList.toggle("active",R.dataset.nav===w)}}}function ce(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function ea(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function ta(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),t=i%60;if(r===0)return`~${t}m`;if(r<48)return`~${r}h ${t}m`;const f=Math.floor(r/24),w=r%24;return`~${f}d ${w}h`}function M(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function ke(s){return`<span class="dot dot-${s}"></span>`}const bt=["B","KB","MB","GB","TB","PB"];function xe(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<bt.length-1;)i/=1024,r++;const t=i<10?2:i<100?1:0;return`${i.toFixed(t)} ${bt[r]}`}async function Me(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ge(s,i){s.addEventListener("click",r=>{const t=r.target.closest("[data-action]");if(!t||!s.contains(t))return;const f=t.dataset.action;f&&i(f,t,r)})}function tt(s,i,r){const t=i.find(w=>w.value===r),f=i.map(w=>`
      <li class="dropdown-option${w.value===r?" selected":""}" role="option"
          aria-selected="${w.value===r}" data-value="${n(w.value)}">
        ${n(w.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${f}</ul>
    </div>
  `}function Be(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function ot(s,i){s.addEventListener("click",f=>{const w=f.target,R=w.closest(".dropdown-trigger");if(R&&s.contains(R)){const I=R.closest(".dropdown"),q=!!I&&!I.classList.contains("open");Be(s),I&&q&&(I.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const m=w.closest(".dropdown-option");if(m&&s.contains(m)){const I=m.closest(".dropdown");Be(s),i((I==null?void 0:I.dataset.dropdown)??"",m.dataset.value??"");return}Be(s)});const r=f=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}const w=f.target;(!w.closest(".dropdown")||!s.contains(w))&&Be(s)},t=f=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}f.key==="Escape"&&Be(s)};document.addEventListener("click",r),document.addEventListener("keydown",t)}const ze="app-modal";let Ge=null;function ie(s,i){X();const r=document.createElement("div");r.className="modal-overlay",r.id=ze,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",f=>{const w=f.target.closest("[data-modal-action]");w!=null&&w.dataset.modalAction?i(w.dataset.modalAction):f.target===r&&i("cancel")});const t=f=>{f.key==="Escape"&&i("cancel")};document.addEventListener("keydown",t),Ge=t,document.body.appendChild(r)}function X(){var s;(s=document.getElementById(ze))==null||s.remove(),Ge&&(document.removeEventListener("keydown",Ge),Ge=null)}function Ue(){return document.querySelector(`#${ze} .modal`)}function De(s){return new Promise(i=>{var f;let r=!1;const t=w=>{r||(r=!0,X(),i(w))};ie(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,w=>t(w==="confirm")),(f=document.querySelector(`#${ze} [data-modal-action="confirm"]`))==null||f.focus()})}const Ze=5e3,na=60;function aa(s,i){let r=!1,t=null,f=null,w=null,R=null;const m=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${le()}</div>`;const I=s.querySelector("#an-body");ge(s,(y,d)=>{var T;y==="toggle-endpoint"&&((T=d.closest(".an-endpoint"))==null||T.classList.toggle("expanded"))}),q();async function q(){try{t=((await st()).gateways??[]).find(d=>d.id===i)??null}catch(y){if(r)return;w=String(y instanceof Error?y.message:y),H();return}if(!r){if(!t){H();return}await O(),R=window.setInterval(()=>void O(),Ze)}}async function O(){try{const y=await _n(i);if(r)return;x(y),f=y,w=null}catch(y){if(r)return;w=String(y instanceof Error?y.message:y)}H()}function x(y){if(!y.enabled||y.error)return;const d=m[m.length-1];d&&d.since!==y.since&&(m.length=0);const T=new Map;for(const A of y.networks??[])T.set(A.chainId,A.received);m.push({t:Date.now(),since:y.since,received:T}),m.length>na&&m.shift()}function H(){r||(I.innerHTML=U())}function U(){return w&&!f?`<h1>Analytics</h1><p class="error">${n(w)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${L(t)}
      ${f?p(f):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(i)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function L(y){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${n(y.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${P()}</div>
      </div>
    `}function P(){if(!f)return"";if(!f.enabled)return"counters off";if(f.error)return"could not be read";const y=f.since?new Date(f.since):null;return y&&!Number.isNaN(y.getTime())?`totals since the gateway started, ${n(y.toLocaleString())}<br />re-read every ${Ze/1e3}s`:`re-read every ${Ze/1e3}s`}function p(y){return y.enabled?y.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(y.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:u(y)+de(y):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function u(y){const d=y.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(T=>k(T)).join("")}
      </section>
    `}function k(y){const d=y.methods??[],T=y.endpoints??[],A=y.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${y.chainId}</span>
          <span class="band-name">${n(y.name)}</span>
          ${F(y)}
        </div>
        <div class="an-stats">
          ${N("Received",ce(y.received),"what clients asked this chain for")}
          ${N("Answered",ce(y.answered),"returned by one of your endpoints")}
          ${N("From cache",ce(y.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${N("Failed",ce(y.failed),"asked for and never answered",y.failed>0?"bad":"")}
        </div>
        ${te(y.chainId)}
        ${A?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ae("Method",d.map(D=>({label:D.method,l:D})))+ae("Endpoint",T.map(D=>({label:D.upstream,l:D})))+W(y)}
      </div>
    `}function N(y,d,T,A=""){return`
      <div class="an-stat${A?" an-stat-"+A:""}" title="${n(T)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(y)}</span>
      </div>
    `}function F(y){const d=Z(y.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const T=Math.round((m[m.length-1].t-m[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${T}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${T}s</span>
    </span>`}function Z(y){if(m.length<2)return null;const d=m[0],T=m[m.length-1],A=(T.t-d.t)/1e3;if(A<=0)return null;const D=(T.received.get(y)??0)-(d.received.get(y)??0);return D<0?null:D/A}function te(y){if(m.length<3)return"";const d=[];for(let v=1;v<m.length;v++){const E=m[v-1],_=m[v],l=(_.t-E.t)/1e3,g=(_.received.get(y)??0)-(E.received.get(y)??0);d.push(l>0&&g>=0?g/l:0)}const T=Math.max(...d);if(T<=0)return"";const A=240,D=28,K=d.length>1?A/(d.length-1):A,b=d.map((v,E)=>`${(E*K).toFixed(1)},${(D-v/T*D).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${T.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${A} ${D}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(T.toFixed(2))} req/s</span>
      </div>
    `}function W(y){const d=[];return y.cached.count>0&&d.push(`${n(ce(y.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${y.cached.mean===null?"":`, in ${n(He(y.cached.mean))} on average`}.`),y.failedLatency.count>0&&y.failedLatency.mean!==null&&d.push(`The ${n(ce(y.failedLatency.count))} that failed took
         ${n(He(y.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function ae(y,d){return d.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${n(y)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${d.map(T=>ue(T.label,T.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function ue(y,d){return`
      <tr>
        <td><code>${n(y)}</code></td>
        <td class="an-num">${ce(d.count)}</td>
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(He(d.mean))}</td>
        <td>${J(d)}</td>
      </tr>
    `}function J(y){const d=y.buckets??[];if(d.length===0||y.count===0)return'<span class="muted small">—</span>';let T=0;const A=[];for(const K of d){const b=K.count-T;T=K.count,A.push({label:oe(K.le),n:Math.max(0,b)})}return A.reduce((K,b)=>K+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(A.filter(K=>K.n>0).map(K=>`${K.n} ${K.label}`).join(" · "))}">
        ${A.map((K,b)=>K.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${K.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(se(A))}</span>
    `}function se(y){for(let d=y.length-1;d>=0;d--)if(y[d].n>0)return`slowest ${y[d].label}`;return""}function oe(y){if(y==="+Inf")return"30s or more";const d=Number(y);return Number.isFinite(d)?`under ${He(d)}`:`under ${y}`}function de(y){const d=y.endpoints??[];return`
      <section class="an-section">
        <h2>What the gateway sees from your endpoints</h2>
        <p class="muted small">
          The gateway's own view, not a client's. Every count here <strong>includes the
          gateway's block-tracking poller</strong>, which calls each endpoint on a timer
          whether or not anyone is using it — on a quiet gateway it is nearly all of this.
          That is why these numbers are much larger than the ones above, and why they are
          not a measure of your traffic.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">The gateway has not talked to any endpoint yet.</p></div>':`<div class="card">
                 <div class="surface-scroll">
                   <table class="surface an-endpoints">
                     <thead>
                       <tr>
                         <th>Endpoint</th>
                         <th class="an-num">Asked</th>
                         <th class="an-num">Errors</th>
                         <th class="an-num">Head lag</th>
                         <th>Selection</th>
                       </tr>
                     </thead>
                     <tbody>${d.map(T=>Y(T)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function Y(y){const d=y.errors??[],T=d.reduce((D,K)=>D+K.count,0),A=d.length>0;return`
      <tr class="an-endpoint${A?" expandable":""}" ${A?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(y.upstream)}</code>
          ${y.chainId?`<span class="muted small">chain ${y.chainId}</span>`:""}
          ${y.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ce(y.requests)}</td>
        <td class="an-num${T>0?" bad":""}">${T>0?ce(T):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${y.headLag>0?ce(y.headLag):'<span class="muted">0</span>'}</td>
        <td>${fe(y)}</td>
      </tr>
      ${A?ye(y,d):""}
    `}function fe(y){const d=[];return y.scored?(d.push(y.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(y.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(y.score.toFixed(3))}</span>`),y.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ce(y.primarySwitches)} switches</span>`),y.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(He(y.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ye(y,d){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${d.map(T=>`
                    <tr>
                      <td class="an-num">${ce(T.count)}</td>
                      <td><code>${n(T.class)}</code></td>
                      <td>${T.severity?`<span class="badge badge-${T.severity==="critical"?"bad":"warn"}">${n(T.severity)}</span>`:""}</td>
                      <td class="muted small">${n(T.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${n(y.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{r=!0,R!==null&&window.clearInterval(R)}}function He(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}function sa(s,i){let r=!1,t=null,f=null,w=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${le()}</div>`;const m=s.querySelector("#diag-body"),I=s.querySelector("#diag-footer");ge(s,(p,u)=>{var k;if(p==="run")O();else if(p==="toggle")(k=u.closest(".check-item"))==null||k.classList.toggle("expanded");else if(p==="copy"){const N=u.dataset.copy;N&&P(u,N)}}),q();async function q(){let p,u;try{const[N,F]=await Promise.all([ve(),Se()]);p=N.find(Z=>Z.id===i),u=F}catch(N){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String(N))}</p>`;return}if(r)return;if(!p){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const k=u==null?void 0:u.networks.find(N=>N.ChainID===p.wire.ChainID);k&&(I.innerHTML=le(k.Name,k.LearnURL));try{t=await Hn(i),R=!0}catch(N){f=String(N instanceof Error?N.message:N)}r||x()}async function O(){w=!0,f=null,x();try{t=await Bn(i),R=!0}catch(p){f=String(p instanceof Error?p.message:p)}w=!1,r||x()}function x(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${w?"disabled":""}>${w?"Running…":"Run diagnostics"}</button>
      </div>
      ${f?`<p class="error">${n(f)}</p>`:""}
      ${H()}
    `}function H(){if(!R&&!f)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const p=new Date(t.at).toLocaleString(),u=t.failedId?`<p><strong>Failed at: ${n(U(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(p)} — trigger: ${n(t.trigger)}</p>
      ${u}
      <ul class="check-list">${t.items.map(L).join("")}</ul>
    `}function U(p){var u;return((u=t==null?void 0:t.items.find(k=>k.ID===p))==null?void 0:u.Title)??p}function L(p){const u=p.Status==="pass"?"ok":p.Status==="fail"?"bad":p.Status==="warn"?"warn":"neutral",k=p.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${k?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${M(k?"failed here":p.Status,u)}
          <strong>${n(p.Title)}</strong>
          <span class="muted small check-detail-inline">${n(p.Detail)}</span>
        </button>
        <div class="check-body">
          <details${k?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(p.Why)}</p>
          </details>
          ${p.Fix?`
                <details${k?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(p.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(p.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(p,u){const k=await Me(u),N=p.textContent;p.textContent=k?"Copied!":"Copy failed",setTimeout(()=>{r||(p.textContent=N)},1500)}return()=>{r=!0}}function Ct(s,i){if(s.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=s.filter(m=>!m.wire);if(r.length>0){const m=r.map(q=>q.id);return{level:"attention",sentence:m.length===1?"1 machine still needs setup.":`${m.length} machines still need setup.`,machines:m}}const t=i.networks??[],f=m=>{const I=t.find(q=>q.ChainID===m);return I?I.Name:`chain ${m}`},w=oa(s.map(m=>f(m.wire.ChainID))),R=s.length===1?"machine":"machines";return{level:"ok",sentence:`All ${s.length} ${R} healthy — ${ra(w)}.`,machines:[]}}function xt(s,i){const r=i.machines.length?` <span class="verdict-machines">${i.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${n(t)}</a>`).join(" ")}</span>`:"";s.innerHTML=`
    <div class="verdict-line verdict-${i.level}">
      ${M(i.level==="ok"?"OK":"Attention",i.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(i.sentence)}</strong>${r}
    </div>
  `}function oa(s){return[...new Set(s)]}function ra(s){return s.length<=1?s[0]??"":s.length===2?`${s[0]} and ${s[1]}`:`${s.slice(0,-1).join(", ")} and ${s[s.length-1]}`}const ia="run",ca=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function la(s,i){const r=i==="linux";return s.some(f=>f.mode==="ssh"||f.mode==="local"&&r)||r}function da(s){let i=!1;s.innerHTML='<div id="home-body"><p class="muted">Loading…</p></div>';const r=s.querySelector("#home-body");let t=!1;ge(s,I=>{I==="setup-endpoint"&&R()}),f();async function f(){let I,q,O;try{const[x,H,U]=await Promise.all([ve(),Se(),at()]);I=x,q=H,O=U.os}catch(x){if(i)return;r.innerHTML=`<p class="error">Failed to load: ${n(String(x))}</p>`;return}if(!i){if(la(I,O)){location.hash="#/targets";return}w(I,q)}}function w(I,q){r.innerHTML=`
      <h1>valve-node-app</h1>
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your RPC endpoint</h2></div>
        <div class="card hero-card">
          <h3>Get an RPC endpoint — no node required ${M("recommended","ok")}</h3>
          <p class="muted">
            eRPC is a managed endpoint that aggregates Valve — via the shared
            <code>vk_demo</code> key — and the chain's known-set public upstreams behind one
            URL, with automatic failover between them. It runs as a container here; you never
            run, sync or babysit a node.
          </p>
          <div class="card-actions">
            <button class="btn btn-primary" data-action="setup-endpoint">Set up my endpoint →</button>
          </div>
          <div id="setup-progress" aria-live="polite"></div>
        </div>
      </section>
      <section class="section">
        <div class="section-head"><h2>Run your own node</h2></div>
        <div class="card card-warn">
          <h3>Run your own node ${M("needs a Linux server","warn")}</h3>
          <p class="muted small">
            Node setup installs systemd units, uses apt and needs root, so it only completes on
            a Linux server — not on this machine. Add one over SSH and valve-node-app will drive
            the node on it from here.
          </p>
          <div class="card-actions">
            <a class="btn btn-ghost" href="#/targets">Add a Linux server →</a>
          </div>
        </div>
      </section>
      ${le()}
    `;const O=r.querySelector("#fleet-verdict");O&&xt(O,Ct(I,q))}async function R(){if(t)return;t=!0;const I=r.querySelector('[data-action="setup-endpoint"]'),q=r.querySelector("#setup-progress");I&&(I.disabled=!0);const O=u=>{q&&(q.innerHTML=u)},x=u=>O(`<p class="muted small"><span class="spinner" aria-label="working"></span> ${n(u)}</p>`),H=(u,k)=>{t=!1,I&&(I.disabled=!1),O(`<p class="error small">${n(u)}${k?` — ${n(k)}`:""}</p>`)};x("Preparing your endpoint…");try{(await ve()).some(k=>k.id==="local")||await et({id:"local",mode:"local"})}catch(u){H(`Could not register this machine: ${Le(u)}`,Ae(u));return}if(i)return;try{const u=await $t("local");if(!u.docker.reachable){H(u.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",u.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(u){H(`Could not check Docker on this machine: ${Le(u)}`,Ae(u));return}if(i)return;let U="default";try{if(((await st()).gateways??[]).find(k=>{var N;return((N=k.placement)==null?void 0:N.targetId)==="local"})){location.hash="#/rpc";return}}catch{}if(i)return;x("Creating the gateway…");try{U=(await wt({id:U,placement:{targetId:"local",backend:"docker"},config:m([])})).id}catch(u){H(`Could not create the gateway: ${Le(u)}`,Ae(u));return}if(i)return;x("Adding Ethereum and PulseChain endpoints…");const L=[];for(const{chainId:u}of ca)try{const N=((await Tt(U,u)).endpoints??[]).filter(F=>!F.alreadyAdded).map(F=>F.url);if(N.length===0)continue;L.push({ChainID:u,Upstreams:N.map((F,Z)=>({ID:`public-${u}-${Z+1}`,Kind:"external",Endpoint:F,Local:!1,RecentOnly:!1}))})}catch(k){H(`Could not read valve's set for chain ${u}: ${Le(k)}`,Ae(k));return}if(i)return;if(L.length===0){H("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await kt(U,m(L))}catch(u){H(`Could not save the endpoints: ${Le(u)}`,Ae(u));return}if(i)return;x("Starting the gateway… the first run pulls the eRPC and Caddy images.");let P;try{P=await St(U)}catch(u){H(`Could not start the gateway: ${Le(u)}`,Ae(u));return}const p=Ve(P.targetId,u=>{if(i){p();return}const k=u.err?`${u.stepId}: ${u.err}`:u.line?`${u.stepId}: ${u.line}`:`${u.stepId}: done`;if(x(k),!!u.err||u.stepId===ia&&!!u.done){if(p(),u.err){H("The gateway was created but did not start — open RPC to see the log and retry."),setTimeout(()=>{i||(location.hash="#/rpc")},1500);return}location.hash="#/rpc"}})}function m(I){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:I,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}return()=>{i=!0}}function Le(s){return s instanceof Error?s.message:String(s)}function Ae(s){return s instanceof $e?s.hint:void 0}const ua=85,Xe={exec:"Execution",beacon:"Beacon"};function pa(s,i){let r=!1,t=null,f=null,w=null,R=null,m=null,I=null,q=null,O=null;const x={exec:null,beacon:null};let H=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${le()}</div>`;const U=s.querySelector("#dash-body"),L=s.querySelector("#dash-footer");U.addEventListener("click",d=>{const T=d.target.closest("[data-action]");if(!T||!U.contains(T))return;const A=T.dataset.action;if(A==="svc-action"){const D=T.dataset.svc,K=T.dataset.kind;D&&K&&Y(D,K)}else if(A==="open-clear"){const D=T.dataset.svc;D&&ye(D)}else if(A==="copy"){const D=T.dataset.copy;D&&fe(T,D)}else A==="retry-du"?p():A==="retry-endpoints"&&u()}),P();async function P(){let d,T;try{const[D,K]=await Promise.all([ve(),Se()]);d=D.find(b=>b.id===i),T=K}catch(D){if(r)return;U.innerHTML=`<p class="error">Failed to load target: ${n(String(D))}</p>`;return}if(r)return;if(!d){U.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){U.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const A=T==null?void 0:T.networks.find(D=>D.ChainID===d.wire.ChainID);A&&(L.innerHTML=le(A.Name,A.LearnURL)),U.innerHTML='<p class="muted">Connecting…</p>',t=xn(i,D=>{r||(k(D),f=D,w=D,N())}),p(),u()}async function p(){I=null;try{m=await Ln(i)}catch(d){m=null,I=String(d instanceof Error?d.message:d)}r||N()}async function u(){O=null;try{q=await An(i)}catch(d){q=null,O=String(d instanceof Error?d.message:d)}r||N()}function k(d){if(!f)return;const T=(new Date(d.at).getTime()-new Date(f.at).getTime())/1e3,A=d.execHead-f.execHead;if(T>0&&A>=0){const D=A/T;R=R===null?D:R*.7+D*.3}}function N(){if(!w)return;const d=w;U.innerHTML=`
      <p class="dash-status">${F(d)}</p>
      <div class="card-grid">
        ${oe(d)}
        ${te(d)}
        ${W(d)}
        ${ae(d)}
        ${ue(d)}
        ${J()}
      </div>
      <p class="muted small">Last updated ${n(new Date(d.at).toLocaleTimeString())}</p>
    `}function F(d){return!d.execActive&&!d.beaconActive?M("Node not running","bad"):d.execSyncing||d.beaconDistance>0?M("Syncing","warn"):M("Running · synced","ok")}function Z(d){const A=d.refHead>0?d.refHead-d.execHead:null,D=A!==null&&A>0&&R&&R>0?ta(A/R):A!==null&&A<=0?"caught up":"—";return{lag:A,eta:D}}function te(d){const{lag:T,eta:A}=Z(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?M("syncing","warn"):d.execHead===0?M("no data","neutral"):M("synced","ok"):M("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ce(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${T!==null?ce(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${T!==null?ce(Math.max(T,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${A}</dd></div>
        </dl>
      </div>
    `}function W(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?M("no data","neutral"):d.beaconDistance===0?M("synced","ok"):M("syncing","warn"):M("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ce(d.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ce(d.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ae(d){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ce(d.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ce(d.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function ue(d){const T=d.diskUsedPct>=ua,A=`
      <div class="meter"><div class="meter-fill ${T?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${ea(d.diskUsedPct)} used</p>
    `;if(I)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="error small">${n(I)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="muted">Loading…</p>
        </div>
      `;const D=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,K=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:b,eta:v}=Z(d),E=b!==null&&b>0&&R!==null&&R>0;return`
      <div class="card ${T?"card-warn":""}">
        <h3>Storage</h3>
        ${A}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${xe(m.ExecBytes)} of ~${xe(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${D}%"></div></div>
        ${E?`<p class="muted small">Estimated time remaining: ${n(v)}</p>`:""}
        <p class="muted small">Beacon — ${xe(m.BeaconBytes)} of ~${xe(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${K}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${xe(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function J(){if(O)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(O)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!q)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=q,T=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",A=d.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(d.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ke(d.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ke(d.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.BeaconHTTP)}">Copy</button>
        </div>
        ${T}
        ${A}
      </div>
    `}function se(d,T){const A=Xe[d],D=x[d],K=(b,v,E)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${b}" ${D!==null||E?"disabled":""}>${D===b?de():n(v)}</button>`;return`
      <div class="service-row">
        <span>${n(A)} ${T?M("active","ok"):M("down","bad")}</span>
        <div class="service-actions">
          ${K("start","Start",T)}
          ${K("stop","Stop",!T)}
          ${K("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${d}" ${D!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(d){return`
      <div class="card">
        <h3>Services</h3>
        ${se("exec",d.execActive)}
        ${se("beacon",d.beaconActive)}
        ${H?`<p class="error small">${n(H)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function de(){return'<span class="spinner" aria-label="working"></span>'}async function Y(d,T){if(x[d]===null){x[d]=T,H=null,N();try{await In(i,d,T)}catch(A){H=`${Xe[d]} ${T} failed: ${A instanceof Error?A.message:String(A)}`}x[d]=null,r||N()}}async function fe(d,T){const A=await Me(T),D=d.textContent;d.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{r||(d.textContent=D)},1500)}function ye(d){const T=Xe[d],A=m?xe(d==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";ie(`
        <h2>Clear ${n(T)} data</h2>
        <p class="error">
          This stops the ${n(T.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(A)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,b=>{if(b==="cancel"){X();return}b==="confirm"&&y(d)});const D=document.getElementById("clear-confirm-input"),K=document.getElementById("clear-confirm-btn");D==null||D.addEventListener("input",()=>{K&&(K.disabled=D.value.trim()!==d)}),D==null||D.focus()}async function y(d){const T=document.getElementById("clear-confirm-btn");T&&(T.disabled=!0,T.textContent="Clearing…");try{await Rn(i,d),X(),p()}catch(A){const D=Ue();if(D){const K=document.createElement("p");K.className="error small",K.textContent=`Clear failed: ${A instanceof Error?A.message:String(A)}`,D.appendChild(K)}T&&(T.disabled=!1,T.textContent="Clear and resync")}}return()=>{r=!0,t==null||t(),X()}}const gt=500,yt="valve-node-app.explain-consent";function ha(s,i){let r=!1,t=null;const f=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${le()}</div>
  `;const w=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");ge(s,P=>{P==="explain"&&O()}),m();async function m(){let P,p;try{const[k,N]=await Promise.all([ve(),Se()]);P=k.find(F=>F.id===i),p=N}catch(k){if(r)return;w.innerHTML=`<p class="error">Failed to load target: ${n(String(k))}</p>`;return}if(r)return;if(!P){w.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!P.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=p==null?void 0:p.networks.find(k=>k.ChainID===P.wire.ChainID);u&&(R.innerHTML=le(u.Name,u.LearnURL));try{const k=await Pn(i,200);if(r)return;f.push(...k)}catch(k){if(r)return;w.innerHTML=`<p class="error">Failed to load logs: ${n(String(k))}</p>`;return}I(),t=En(i,k=>{r||(f.push(k),f.length>gt&&f.splice(0,f.length-gt),I())})}function I(){const P=f.filter(u=>u.severity==="error"||u.severity==="critical");w.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${f.map(q).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${M(String(P.length),P.length?"bad":"neutral")}</h2>
          <div class="log-lines">${P.length?P.slice().reverse().map(q).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const p=w.querySelector(".log-lines");p&&(p.scrollTop=p.scrollHeight)}function q(P){const p=P.severity||"info",u=P.learnUrl?` <a href="${n(P.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(p)}">
        <span class="log-time">${n(new Date(P.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(P.unit)}</span>
        <span class="log-sev">${n(p)}</span>
        <span class="log-text">${n(P.line)}</span>
        ${P.explain?`<div class="log-explain">${n(P.explain)}${u}</div>`:""}
      </div>
    `}async function O(){const P=f.filter(u=>u.severity==="error"||u.severity==="critical").map(u=>u.line).slice(-40);if(!(localStorage.getItem(yt)==="1")){x(P);return}await H(P)}function x(P){const p=P.length?`<pre class="explain-excerpt">${P.map(u=>n(u)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';U(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${p}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,u=>{u==="proceed"?(localStorage.setItem(yt,"1"),L(),H(P)):L()})}async function H(P){U('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const p=P.length?await ft(i,P):await ft(i);if(r)return;U(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(p.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${p.sentExcerpt.map(u=>n(u)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&L()})}catch(p){if(r)return;if(p instanceof $e&&p.status===409){U(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,u=>{u==="close"&&L()});return}U(`
        <h2>Explain failed</h2>
        <p class="error">${n(p instanceof Error?p.message:String(p))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,u=>{u==="close"&&L()})}}function U(P,p){L();const u=document.createElement("div");u.className="modal-overlay",u.id="explain-modal",u.innerHTML=`<div class="modal">${P}</div>`,u.addEventListener("click",k=>{const N=k.target.closest("[data-modal-action]");N!=null&&N.dataset.modalAction&&p(N.dataset.modalAction),k.target===u&&p("cancel")}),document.body.appendChild(u)}function L(){var P;(P=document.getElementById("explain-modal"))==null||P.remove()}return()=>{r=!0,t==null||t(),L()}}const fa="run",ma={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},ba={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function ga(s,i){let r=!1,t=null,f=null;const w={devnet:null},R={devnet:null},m={devnet:[]};let I=null;const q={devnet:!1};let O=null;const x={devnet:null},H={devnet:null};s.innerHTML=`
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
    ${le()}
  `;const U=s.querySelector("#services-body");ge(s,(l,g)=>{ye(l,g)}),L();async function L(){try{const l=await $t(i);if(r)return;t=l,f=null}catch(l){if(r)return;t=null,f=E(l)}p()}function P(l){return t==null?void 0:t.services.find(g=>g.id===l)}function p(){if(!r){if(f){U.innerHTML=`<p class="error">Could not read this machine's services: ${n(f)}</p>`;return}if(!t){U.innerHTML='<p class="muted">Loading…</p>';return}U.innerHTML=`
      ${u(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(k).join("")}
      </div>
    `}}function u(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${n(l.flavor)}${l.serverVersion?` ${n(l.serverVersion)}`:""} · reachable</p>`;const g=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(g)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${n(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function k(l){const g=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(l.label)}</h2>
          ${N(l)}
        </div>
        <p class="muted small">${n(ma[l.id]??"")}</p>

        ${l.error?F(l):""}
        ${l.blocked?`<div class="banner banner-warn">${n(l.blocked)}</div>`:""}
        ${g.map(B=>`<div class="banner banner-warn">${n(B)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${n(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Z(l)}

        ${te(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(B=>W(l,B)).join("")}
        </div>
        ${R[l.id]?`<p class="error small">${n(R[l.id])}</p>`:""}
        ${ae(l)}

        ${ue(l)}
      </div>
    `}function N(l){switch(l.status.State){case"running":return M("running","ok");case"created-but-stopped":return M("stopped","warn");case"not-created":return M("not created","neutral");default:return M("unknown","bad")}}function F(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(l.error??"")}</div>
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function Z(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const g=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${g}.</p>`}function te(l){const g=l.endpoints??[];return g.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":g.map(B=>`
        <div class="endpoint-row">
          ${ke("ok")}
          <span class="muted small">${n(B.label)}</span>
          <code class="endpoint-url">${n(B.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(B.url)}">Copy</button>
        </div>`).join("")}function W(l,g){const B=ba[g];if(!B)return"";const G=w[l.id],Q=g==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:B.label;return`
      <button class="${B.className}" data-action="svc-${g}" data-svc="${n(l.id)}"
              title="${n(B.title)}" ${G?"disabled":""}>
        ${G===g?'<span class="spinner" aria-label="working"></span>':n(Q)}
      </button>
    `}function ae(l){const g=m[l.id]??[];return g.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(g.join(`
`))}</pre>
      </div>
    `}function ue(l){const g=q[l.id],B=J(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(l.id)}">
            ${g?"Close":"Edit"}
          </button>
        </div>
        ${g?se():`<p class="small">${B}</p>`}
        ${x[l.id]?`<p class="error small">${n(x[l.id])}</p>`:""}
        ${H[l.id]?`<p class="muted small">${n(H[l.id])}</p>`:""}
      </div>
    `}function J(l){const g=l.devnet;return g?`Chain ${g.ChainID} · a block every ${n(g.BlockTime)} · JSON-RPC on ${n(g.BindAddr)}:${g.HTTPPort} · WebSocket on ${n(g.BindAddr)}:${g.WSPort}`:"—"}function se(l){return oe()}function oe(){const l=O;return l?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${n(l.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${l.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${l.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${n(l.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${l.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function de(){q.devnet&&O&&(O.BlockTime=Y("#dev-blocktime",O.BlockTime),O.HTTPPort=fe("#dev-http",O.HTTPPort),O.WSPort=fe("#dev-ws",O.WSPort),O.BindAddr=Y("#dev-bind",O.BindAddr))}function Y(l,g){const B=s.querySelector(l);return B?B.value.trim():g}function fe(l,g){const B=s.querySelector(l);if(!B)return g;const G=Number.parseInt(B.value.trim(),10);return Number.isFinite(G)?G:g}async function ye(l,g){const B=g.dataset.svc??"";switch(l){case"refresh":await L();return;case"copy":g.dataset.copy&&await v(g,g.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await y(B,l.slice(4));return;case"svc-create":case"svc-recreate":await d(B);return;case"svc-wipe":D(B);return;case"toggle-config":T(B);return;case"save-config":await A(B);return;default:return}}async function y(l,g){if(!w[l]){w[l]=g,R[l]=null,p();try{await Dn(i,l,g)}catch(B){R[l]=`${g} failed: ${E(B)}${_(B)}`}w[l]=null,await L()}}async function d(l){if(!w[l]){w[l]="create",R[l]=null,m[l]=["starting…"],p();try{await Mn(i,l)}catch(g){R[l]=`${E(g)}${_(g)}`,m[l]=[],w[l]=null,p();return}I==null||I(),I=Ve(i,g=>{if(r)return;const B=g.err?`${g.stepId}: ${g.err}`:g.line?`${g.stepId}: ${g.line}`:`${g.stepId}: done`;if(m[l]=[...(m[l]??[]).filter(Q=>Q!=="starting…"),B],!!g.err||g.stepId===fa&&!!g.done){I==null||I(),I=null,w[l]=null,g.err&&(R[l]="Provisioning failed — see the log below."),L();return}p()})}}function T(l){if(de(),q[l]=!q[l],x[l]=null,H[l]=null,q[l]){const g=P(l);g!=null&&g.devnet&&(O={...g.devnet})}p()}async function A(l){var G;de(),x[l]=null,H[l]=null;const g=O;if(!g)return;if(g.HTTPPort===g.WSPort){x[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",p();return}try{await qn(i,l,g)}catch(Q){x[l]=E(Q),p();return}const B=((G=P(l))==null?void 0:G.status.State)==="running";q[l]=!1,H[l]=B?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function D(l){const g=P(l);if(!g)return;const B=(g.restartsOnWipe??[]).map(j=>{var re;return((re=P(j))==null?void 0:re.label)??j});ie(`
        <h2>Wipe ${n(g.label)}</h2>
        <p class="error">This deletes ${n(g.wipeDiscards)}</p>
        ${B.length?`<p>It also restarts what sits in front of it: ${n(B.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(l)}</button>
        </div>
      `,j=>{if(j==="cancel"||j==="close"){X(),L();return}j==="confirm"&&K(l)});const G=document.getElementById("wipe-confirm-input"),Q=document.getElementById("wipe-confirm-btn");G==null||G.addEventListener("input",()=>{Q&&(Q.disabled=G.value.trim()!==l)}),G==null||G.focus()}async function K(l){const g=document.getElementById("wipe-confirm-btn");g&&(g.disabled=!0,g.textContent="Wiping…");let B;try{B=await Un(i,l)}catch(G){const Q=Ue();if(Q){const j=document.createElement("p");j.className="error small",j.textContent=`Wipe failed: ${E(G)}${_(G)}`,Q.appendChild(j)}g&&(g.disabled=!1,g.textContent=`Wipe ${l}`);return}b(l,B)}function b(l,g){const B=P(l),G=ne=>{var Pe;return((Pe=P(ne))==null?void 0:Pe.label)??ne},Q=[];Q.push(g.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ne of g.report.VolumesRemoved??[])Q.push(`Volume ${ne} deleted.`);for(const ne of g.report.VolumesAbsent??[])Q.push(`Volume ${ne} was already gone.`);g.report.Recreated&&Q.push("Container re-created from your saved configuration.");const j=(g.report.Cascaded??[]).map(G),re=(g.report.CascadeSkipped??[]).map(G);ie(`
        <h2>${n((B==null?void 0:B.label)??l)} wiped</h2>
        <ul class="plain-list">${Q.map(ne=>`<li>${n(ne)}</li>`).join("")}</ul>
        ${j.length?`<p class="ok">Restarted in front of it: ${n(j.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${re.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(re.join(", "))}.</p>`:""}
        ${g.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(g.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ne=>{(ne==="close"||ne==="cancel")&&(X(),L())})}async function v(l,g){const B=await Me(g),G=l.textContent;l.textContent=B?"Copied!":"Copy failed",setTimeout(()=>{r||(l.textContent=G)},1500)}function E(l){return l instanceof Error?l.message:String(l)}function _(l){return l instanceof $e&&l.hint?` — ${l.hint}`:""}return()=>{r=!0,I==null||I(),X()}}const Qe=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],We=8545,_e=5052,Ke=30303,ya=[369,943,1],vt={369:"default",943:"practise here first"};function va(s,i){let r=!1;const t={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${le()}</div>`;const f=s.querySelector("#wizard-body"),w=s.querySelector("#wizard-footer");ge(s,(b,v)=>{fe(b,v)}),ot(s,(b,v)=>{b==="exec-select"?t.execId=v:b==="beacon-select"&&(t.beaconId=v),m()}),s.addEventListener("change",b=>{const v=b.target;v instanceof HTMLInputElement&&(v.id==="data-dir-input"?(ye(),W()):v.id==="checkpoint-toggle"?(t.checkpoint=v.checked,m()):v.id==="exec-snapshot-toggle"&&(t.execSnapshot=v.checked,m()))}),R();async function R(){try{const[b,v]=await Promise.all([Se(),ve()]);if(r)return;t.catalog=b;const E=v.find(_=>_.id===i);E!=null&&E.wire&&(t.chainId=E.wire.ChainID,t.execId=E.wire.ExecID,t.beaconId=E.wire.BeaconID,t.archive=E.wire.Archive,E.wire.ExecHTTPPort&&(t.execHTTPPort=String(E.wire.ExecHTTPPort)),E.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(E.wire.BeaconHTTPPort)),E.wire.ExecP2PPort&&(t.execP2PPort=String(E.wire.ExecP2PPort)),E.wire.RPCBindAddr&&(t.rpcBindAddr=E.wire.RPCBindAddr)),m()}catch(b){if(r)return;t.loadError=String(b instanceof Error?b.message:b),m()}}function m(){if(t.loadError){f.innerHTML=`<p class="error">Failed to load: ${n(t.loadError)}</p>`;return}t.catalog&&(f.innerHTML=`
      ${K(t.step)}
      ${q()}
    `,I())}function I(){var v;const b=(v=t.catalog)==null?void 0:v.networks.find(E=>E.ChainID===t.chainId);w.innerHTML=b?le(b.Name,b.LearnURL):le()}function q(){switch(t.step){case"network":return O();case"clients":return x();case"mode":return oe();case"review":return de();case"run":return Y()}}function O(){const b=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${ya.map(E=>{const _=b.networks.find(B=>B.ChainID===E);if(!_)return"";const l=t.chainId===E,g=vt[E]?M(vt[E],E===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${E}" type="button">
          <h3>${n(_.Name)} <span class="muted">(chain ${E})</span></h3>
          ${g}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function x(){const b=t.catalog,v=b.networks.find(l=>l.ChainID===t.chainId);if(!v)return'<p class="error">Unknown network.</p>';(t.execId===null||!v.ExecClients.includes(t.execId))&&(t.execId=v.ExecClients[0]??null),(t.beaconId===null||!v.BeaconClients.includes(t.beaconId))&&(t.beaconId=v.BeaconClients[0]??null);const E=v.ExecClients.map(l=>ue(l,b)),_=v.BeaconClients.map(l=>ue(l,b));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${n(v.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${tt("exec-select",E,t.execId)}
        </label>
        ${se(t.execId,b)}
        <label>
          Beacon client
          ${tt("beacon-select",_,t.beaconId)}
        </label>
        ${se(t.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function H(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const U=1.1,L=.5,P="Valve reth snapshot",p="rough estimate";function u(b){return b.SnapshotSizeTB}function k(b){return b.SnapshotSizeTB*L}function N(b){return`<p class="muted small">${H(u(b))} is the measured size of Valve's reth snapshot for ${n(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function F(b){return{archive:u(b)*1e12*U,full:k(b)*1e12*U}}function Z(b,v){if(!b)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${n(v)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${n(v)}</code>: ${n(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==v)return"";const E=F(b),_=t.freeBytes>=E.archive,l=t.freeBytes>=E.full,g=`<p class="muted small">Free at <code>${n(v)}</code>: <strong>${xe(t.freeBytes)}</strong> — archive ${_?"fits":"won't fit"} (${H(u(b))}, ${P}), full ${l?"fits":"won't fit"} (${H(k(b))}, ${p}).</p>`;let B="";return t.downgradeNote?B=`<p class="banner banner-warn">${n(t.downgradeNote)}</p>`:l||(B=`<p class="banner banner-warn">Neither full (${H(k(b))}, ${p}) nor archive (${H(u(b))}, ${P}) fits the free space here — choose a location with more room.</p>`),g+B}function te(b,v){if(t.downgradeNote=null,!b||t.freeBytes===null)return;const E=F(b);t.archive&&t.freeBytes<E.archive&&t.freeBytes>=E.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${v} for archive (${H(u(b))}, ${P}) — switched to Full (${H(k(b))}, ${p}). Pick a location with more room to run archive.`)}async function W(){var E;if(t.chainId===null)return;const b=(E=t.catalog)==null?void 0:E.networks.find(_=>_.ChainID===t.chainId),v=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,m();try{const{freeBytes:_}=await Tn(t.targetId,v);if(r)return;t.freeBytes=_,t.probedPath=v,te(b,v)}catch(_){if(r)return;t.freeBytes=null,t.probedPath=v,t.diskError=String(_ instanceof Error?_.message:_)}t.diskProbing=!1,m()}function ae(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ue(b,v){const E=v.clients.find(_=>_.id===b);return{value:b,label:E?`${E.id} — ${J(E.repo)}`:b}}function J(b){const v=b.split("/");return v.length>=4?v[3]:b}function se(b,v){const E=b?v.clients.find(l=>l.id===b):void 0;if(!E)return"";const _=E.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(E.repo)}" target="_blank" rel="noopener noreferrer">${n(_)}</a></p>`}function oe(){var G,Q,j;const b=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",v=(G=t.catalog)==null?void 0:G.networks.find(re=>re.ChainID===t.chainId),E=((j=(Q=t.catalog)==null?void 0:Q.clients.find(re=>re.id===t.execId))==null?void 0:j.snapshotSupported)??!1,_=v?`${H(k(v))} (${p})`:"Smaller",l=v?`${H(u(v))} (${P})`:"Much larger",g=v?` on ${n(v.Name)}`:"",B=v?t.checkpoint?v.SyncLabel:v.GenesisSyncLabel:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <p class="muted">
          Both modes run a fully-validating node — same security, same current-state RPC.
          The difference is how much <strong>historical</strong> state is kept.
        </p>

        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="checkpoint-toggle" ${t.checkpoint?"checked":""} />
            <span><strong>Consensus checkpoint sync (beacon client)</strong> — start near the chain head in minutes (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.</span>
          </label>
          <p class="muted small">This applies to the beacon/consensus client (e.g. lighthouse-pulse) — not the execution client, which uses a snapshot below.</p>
          ${v?`<p class="sync-estimate">⏱ Estimated initial sync${g}: <strong>${n(B)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((v==null?void 0:v.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((v==null?void 0:v.CheckpointURL)??"")}" value="${n(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${n(t.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${E?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${t.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${t.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${n(t.snapshotKey)}" />
                 </label>
                 ${t.snapshotKeyError?`<p class="error small">${n(t.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${g}</th><td class="yes">${_}</td><td class="limited">${l}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${v?N(v):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${v?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${_}${v?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(b)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(b)}" value="${n(t.dataDir)}" />
          </label>
          ${Z(v,t.dataDir||b)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(b)}/jwt.hex" value="${n(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${We})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${We}" value="${n(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${n(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${_e})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${_e}" value="${n(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${n(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Ke})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Ke}" value="${n(t.execP2PPort)}" />
          </label>
          ${t.execP2PPortError?`<p class="error small">${n(t.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${n(t.rpcBindAddr)}" />
          </label>
          ${t.rpcBindAddrError?`<p class="error small">${n(t.rpcBindAddrError)}</p>`:""}
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
    `}function de(){const v=t.catalog.networks.find(ne=>ne.ChainID===t.chainId),E=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,_=t.jwtPath||`${E}/jwt.hex`,l=Qe.map(ne=>`<li>${n(ne.title)}</li>`).join(""),g=A(t.execHTTPPort,We),B=A(t.beaconHTTPPort,_e),G=A(t.execP2PPort,Ke),Q=g||B||G?`<tr><th>Non-default ports</th><td>${[g?`exec HTTP ${g}`:null,B?`beacon HTTP ${B}`:null,G?`exec p2p ${G}`:null].filter(ne=>ne!==null).map(n).join(", ")}</td></tr>`:"",{addr:j}=y(t.rpcBindAddr),re=j?`<tr><th>RPC bind address</th><td><code>${n(j)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((v==null?void 0:v.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(E)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(_)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${n(t.checkpointUrl||(v==null?void 0:v.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${Q}
            ${re}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${l}</ol>
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${t.starting?"disabled":""}>
            ${t.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function Y(){const v=t.catalog.networks.find(j=>j.ChainID===t.chainId),E=v==null?void 0:v.LearnURL,_=new Set(t.events.filter(j=>j.done).map(j=>j.stepId)),l=new Set(t.events.filter(j=>j.err).map(j=>j.stepId)),g=new Map;for(const j of t.events){if(!j.line)continue;const re=g.get(j.stepId)??[];re.push(j.line),g.set(j.stepId,re)}const B=Qe.map(j=>{var Fe;const re=_.has(j.id),ne=l.has(j.id),Pe=ne?M("failed","bad"):re?M("done","ok"):M("pending","neutral"),Oe=(g.get(j.id)??[]).slice(-5),qe=(Fe=t.events.find(Ee=>Ee.stepId===j.id&&Ee.err))==null?void 0:Fe.err,Je=j.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${E?` <a href="${n(E)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${re?"step-done":""} ${ne?"step-error":""}">
          <div class="step-head">${Pe} <strong>${n(j.title)}</strong></div>
          ${Je}
          ${Oe.length?`<pre class="step-log">${Oe.map(Ee=>n(Ee)).join(`
`)}</pre>`:""}
          ${qe?`<p class="error small">${n(qe)}</p>`:""}
        </li>
      `}).join(""),G=t.events.some(j=>j.err),Q=Qe.every(j=>_.has(j.id))||t.events.some(j=>j.stepId==="handshake"&&j.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${B}</ol>
        ${Q&&!G?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        ${G?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function fe(b,v){switch(b){case"pick-network":t.chainId=Number(v.dataset.chainId),t.execId=null,t.beaconId=null,m();break;case"goto-network":t.step="network",m();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",m();break;case"goto-mode":t.step="mode",m(),W();break;case"goto-review":if(ye(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){m();break}t.step="review",m();break;case"start-setup":D();break}}function ye(){const b=s.querySelectorAll('input[name="mode"]');for(const j of Array.from(b))j.checked&&(t.archive=j.value==="archive");const v=s.querySelector("#data-dir-input"),E=s.querySelector("#jwt-path-input");v&&(t.dataDir=v.value.trim()),E&&(t.jwtPath=E.value.trim());const _=s.querySelector("#exec-http-port-input"),l=s.querySelector("#beacon-http-port-input"),g=s.querySelector("#exec-p2p-port-input");_&&(t.execHTTPPort=_.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),g&&(t.execP2PPort=g.value.trim());const B=s.querySelector("#rpc-bind-addr-input");B&&(t.rpcBindAddr=B.value.trim());const G=s.querySelector("#checkpoint-url-input");G&&(t.checkpointUrl=G.value.trim());const Q=s.querySelector("#snapshot-key-input");Q&&(t.snapshotKey=Q.value.trim()),t.execHTTPPortError=T(t.execHTTPPort).error??null,t.beaconHTTPPortError=T(t.beaconHTTPPort).error??null,t.execP2PPortError=T(t.execP2PPort).error??null,t.rpcBindAddrError=y(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?ae(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function y(b){if(!b)return{};const v=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return v?v.slice(1).every(E=>Number(E)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function T(b){if(!b)return{};if(!d.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const v=Number(b);return!Number.isInteger(v)||v<1||v>65535?{error:"Port must be between 1 and 65535."}:{port:v}}function A(b,v){const{port:E}=T(b);if(!(E===void 0||E===v))return E}async function D(){var g;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(g=t.streamStop)==null||g.call(t),t.streamStop=null,m();const b={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(b.DataDir=t.dataDir),t.jwtPath&&(b.JWTPath=t.jwtPath);const v=A(t.execHTTPPort,We),E=A(t.beaconHTTPPort,_e),_=A(t.execP2PPort,Ke);v!==void 0&&(b.ExecHTTPPort=v),E!==void 0&&(b.BeaconHTTPPort=E),_!==void 0&&(b.ExecP2PPort=_);const{addr:l}=y(t.rpcBindAddr);l!==void 0&&(b.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(b.CheckpointURL=t.checkpointUrl):b.NoCheckpoint=!0,t.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=t.snapshotKey);try{await Cn(t.targetId,b)}catch(B){if(!(B instanceof $e&&B.status===409)){t.starting=!1,t.startError=String(B instanceof Error?B.message:B),m();return}}t.starting=!1,t.step="run",m(),t.streamStop=Ve(t.targetId,B=>{r||(t.events.push(B),t.step==="run"&&m())})}function K(b){const v=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],_=v.map(l=>l.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${v.map((l,g)=>`<li class="${g===_?"current":g<_?"past":"future"}">${n(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;r=!0,(b=t.streamStop)==null||b.call(t)}}function $a(s,i){let r=!1;const t=new Map;s.innerHTML=`<h1>${n(i)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const f=s.querySelector("#machine-body");ge(s,(x,H)=>{x==="toggle-section"&&q(H.dataset.section??"")}),w();async function w(){let x,H;try{const[U,L]=await Promise.all([ve(),Se()]);x=U.find(P=>P.id===i),H=L}catch(U){if(r)return;f.innerHTML=`<p class="error">Failed to load machine: ${n(String(U))}</p>`;return}if(!r){if(!x){location.hash="#/targets";return}R(x,H)}}function R(x,H){const U=x.mode==="local"?"this machine":"SSH",L=x.mode==="ssh"&&x.ssh?`${n(x.ssh.User)}@${n(x.ssh.Host)}`:U;f.innerHTML=`
      <p class="muted">${L}</p>
      <p>${m(x,H)}</p>
      <div class="machine-sections">
        ${O.map(P=>I(P,x,H)).join("")}
      </div>
      ${le()}
    `}function m(x,H){const U=x.wire;if(!U)return M("not set up","neutral");const L=H.networks.find(p=>p.ChainID===U.ChainID),P=L?L.Name:`chain ${U.ChainID}`;return`${M(P,"ok")} ${M(U.ExecID,"neutral")} ${M(U.BeaconID,"neutral")}${U.Archive?" "+M("archive","warn"):""}`}function I(x,H,U){return`
      <section class="card machine-section" data-section-card="${n(x.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(x.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(x.title)}</span>
          <span class="machine-section-status">${x.status(H,U)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(x.key)}" hidden></div>
      </section>
    `}function q(x){const H=O.find(u=>u.key===x);if(!H)return;const U=s.querySelector(`[data-section-card="${x}"]`),L=s.querySelector(`[data-section-body="${x}"]`),P=s.querySelector(`.machine-section-head[data-section="${x}"]`);if(!U||!L||!P)return;const p=L.hidden;if(p&&!t.has(x)){const u=document.createElement("div");L.appendChild(u),t.set(x,H.mount(u))}L.hidden=!p,U.classList.toggle("open",p),P.setAttribute("aria-expanded",String(p))}const O=[{key:"setup",title:"Setup",status:x=>x.wire?M("set up","ok"):M("not set up","neutral"),mount:x=>va(x,i)},{key:"dashboard",title:"Dashboard",status:x=>x.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:x=>pa(x,i)},{key:"logs",title:"Logs",status:x=>x.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:x=>ha(x,i)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:x=>ga(x,i)}];return()=>{r=!0;for(const x of t.values())try{x()}catch{}t.clear()}}function wa(s,i){let r=!1,t=[],f=null,w=!1,R=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${le()}</div>`;const m=s.querySelector("#sec-body"),I=s.querySelector("#sec-footer");ge(s,(L,P)=>{var p;if(L==="rerun")O();else if(L==="toggle")(p=P.closest(".check-item"))==null||p.classList.toggle("expanded");else if(L==="copy"){const u=P.dataset.copy;u&&U(P,u)}}),q();async function q(){let L,P;try{const[u,k]=await Promise.all([ve(),Se()]);L=u.find(N=>N.id===i),P=k}catch(u){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String(u))}</p>`;return}if(r)return;if(!L){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const p=P==null?void 0:P.networks.find(u=>u.ChainID===L.wire.ChainID);p&&(I.innerHTML=le(p.Name,p.LearnURL)),await O()}async function O(){w=!0,f=null,x();try{t=await Nn(i),R=!0}catch(L){f=String(L instanceof Error?L.message:L)}w=!1,r||x()}function x(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${w?"disabled":""}>${w?"Re-running…":"Re-run checks"}</button>
      </div>
      ${f?`<p class="error">${n(f)}</p>`:""}
      ${!R&&w?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(H).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function H(L){const P=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${M(L.Status,P)}
          <strong>${n(L.Title)}</strong>
          <span class="muted small check-detail-inline">${n(L.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${n(L.Why)}</p>
          </details>
          ${L.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(L.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(L.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function U(L,P){const p=await Me(P),u=L.textContent;L.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{r||(L.textContent=u)},1500)}return()=>{r=!0}}const ka=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],nt="VALVE_API_KEY";function Sa(s){return s===nt?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(s)}}</code> slot wherever an endpoint URL carries one.`}function Ta(s){let i=!1,r=!1,t=!1,f=null,w=!1,R=null,m=null;const I=new Set,q=new Map;let O="",x="";s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${le()}`;const H=s.querySelector("#settings-body");ge(s,(k,N)=>{if(k==="save"&&u(),k==="clear-key"){if(!R)return;r=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),p(R)}if(k==="clear-provider-key"){const F=N.dataset.key;if(!R||!F)return;I.add(F),q.set(F,""),w=!1,p(R)}}),ot(s,(k,N)=>{k!=="ai-provider"||!R||(m=N,w=!1,p(R))}),U();async function U(){try{const k=await Zn();if(i)return;R=k,p(k)}catch(k){if(i)return;H.innerHTML=`<p class="error">Failed to load settings: ${n(String(k))}</p>`}}function L(k){const F=(Array.isArray(k.providerKeysSet)?k.providerKeysSet:[]).filter(Z=>Z!==nt).sort();return[nt,...F]}function P(k,N){const F=n(k);return`
      <div class="pk-row">
        <label>
          <code>${F}</code>
          <input class="provider-key" data-key="${F}" type="password" autocomplete="off"
                 placeholder="${N?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Sa(k)}</p>
        ${N?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${F}">Clear saved key</button>`:""}
      </div>`}function p(k){var ue;const N=m??k.aiProvider,F=Array.isArray(k.providerKeysSet)?k.providerKeysSet:[],Z=L(k).map(J=>P(J,F.includes(J))).join("");H.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${tt("ai-provider",ka.map(J=>({value:J.value,label:J.label})),N)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${k.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${k.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${Z}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${n(O)}" />
            </label>
            <label>
              <span class="muted small">its key</span>
              <input id="pk-new-value" type="password" autocomplete="off" placeholder="no key set" />
            </label>
            <p class="muted small">Use the exact name the rejection quotes. Letters, digits and underscores only.</p>
          </div>
        </section>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${n(k.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${f?`<p class="error">${n(f)}</p>`:""}
        ${w?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const te=s.querySelector("#ai-key");te==null||te.addEventListener("input",()=>{r=!0,w=!1}),(ue=s.querySelector("#ref-rpc-base"))==null||ue.addEventListener("input",()=>{w=!1}),s.querySelectorAll("input.provider-key").forEach(J=>{const se=J.dataset.key;if(!se)return;const oe=q.get(se);oe!==void 0&&(J.value=oe),J.addEventListener("input",()=>{I.add(se),q.set(se,J.value),w=!1})});const W=s.querySelector("#pk-new-value");W&&(W.value=x),W==null||W.addEventListener("input",()=>{x=W.value,w=!1});const ae=s.querySelector("#pk-new-name");ae==null||ae.addEventListener("input",()=>{O=ae.value,w=!1})}async function u(){const k=s.querySelector("#ai-key"),N=s.querySelector("#ref-rpc-base");if(!k||!N||!R)return;const F={aiProvider:m??R.aiProvider,refRpcBase:N.value.trim()};r&&(F.aiKey=k.value);const Z={};for(const W of I)Z[W]=q.get(W)??"";const te=O.trim();te&&(Z[te]=x),Object.keys(Z).length>0&&(F.providerKeys=Z),t=!0,f=null,w=!1,p(R);try{const W=await Xn(F);if(i)return;R=W,r=!1,I.clear(),q.clear(),O="",x="",t=!1,w=!0,p(W)}catch(W){if(i)return;t=!1,f=String(W instanceof Error?W.message:W),p(R)}}return()=>{i=!0}}const Ca=["http","ws","archive","trace"],xa={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Ne=1337,Pa="run",Ea={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Ia(s){let i=!1,r=null,t=null;const f={},w={},R={},m={},I={},q={},O={},x={},H={},U={},L={},P={},p={},u={},k={};let N="",F=null;s.innerHTML=`
    <div class="page-head">
      <h1>RPC</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      A machine runs one gateway, and that gateway fronts as many chains as you list.
      Each chain below leads with the URL you point a wallet or dApp at and whether it
      is healthy. The operator detail — every upstream, its capabilities and share —
      is one click away under each chain's “Details”.
    </p>
    <div id="rpc-body"><p class="muted">Loading…</p></div>
    ${le()}
  `;const Z=s.querySelector("#rpc-body");ge(s,(e,a)=>{Vt(e,a)}),ot(s,()=>{}),W(),te();async function te(){try{const e=await at();if(i)return;N=e.os,Y()}catch{}}async function W(){try{const e=await st();if(i)return;r=e,t=null}catch(e){if(i)return;r=null,t=pe(e)}Y();for(const e of(r==null?void 0:r.gateways)??[])ae(e.id),ue(e.id,!1)}async function ae(e){try{const a=await Wn(e);if(i)return;f[e]=a}catch{if(i)return;f[e]=null}Y()}async function ue(e,a){R[e]=a,a&&Y();try{const o=await Kn(e,a);if(i)return;w[e]=o}catch{if(i)return;w[e]=null}R[e]=!1,Y()}function J(e){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===e)}function se(e,a){return(e.networks??[]).find(o=>o.chainId===a)}function oe(e,a,o){var h;const c=(((h=f[e])==null?void 0:h.networks)??[]).find(S=>S.chainId===a);return((c==null?void 0:c.upstreams)??[]).find(S=>S.upstream===o)}function de(e,a,o){var c;return(((c=w[e])==null?void 0:c.endpoints)??[]).find(h=>h.chainId===a&&h.upstream===o)}function Y(){if(i)return;if(t){Z.innerHTML=`<p class="error">Could not read the gateways: ${n(t)}</p>`;return}if(!r){Z.innerHTML='<p class="muted">Loading…</p>';return}const e=r.gateways??[],a=e.length>1,o=(r.targets??[]).some(S=>ht(S.id,e)),c=new Set(e.map(S=>S.placement.targetId)),h=(r.orphans??[]).filter(S=>!c.has(S.targetId));Z.innerHTML=`
      ${e.map(S=>y(S,a)).join("")}
      ${e.length===0?ye():""}
      ${h.map(fe).join("")}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function fe(e){const a=`docker rm -f ${e.containerName}`,o=p[e.containerName];return`
      <div class="strip">
        ${B({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:a})}
        ${o?B({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function ye(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function y(e,a){return`
      ${a?`<h2 class="rpc-machine">${n(e.placement.targetId)}</h2>`:""}
      ${d(e)}
      ${g(e)}
      ${ne(e)}
      ${Pe(e)}
      ${b(e)}
    `}function d(e){const a=e.status.State==="running",o=e.tls,c=[`on <strong>${n(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${n(e.status.Image)}</code>`),c.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${j(e)}
        <strong>${n(e.label)}</strong>
        ${Q(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${a?`base <code>${n(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function T(e){const a=e.tls;return a!=null&&a.enabled&&a.rootCaPath&&a.effectiveCertSource==="internal"?a.rootCaPath:null}function A(e){var a;return((a=((r==null?void 0:r.targets)??[]).find(o=>o.id===e.placement.targetId))==null?void 0:a.mode)??""}function D(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function K(e,a,o){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${a}"`;case"windows":return`certutil -addstore -f ROOT "${a}"`;case"linux":default:return`sudo cp "${a}" /usr/local/share/ca-certificates/valve-node-app-${o}.crt && sudo update-ca-certificates`}}function b(e){const a=H[e.id]??!1,o=((r==null?void 0:r.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${a?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${n(e.id)}" aria-expanded="${a}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${v(e,o.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${a?E(e,o):""}
      </section>
    `}function v(e,a){const o=[];return e.status.State!=="running"&&o.push("gateway not running"),a>0&&o.push(`${a} leftover container${a===1?"":"s"}`),o.length===0?"container, settings, certificate":o.join(" · ")}function E(e,a){var o;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>re(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${R[e.id]?"disabled":""}>
            ${R[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(e.id)}">
            ${O[e.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(e.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${e.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${n(e.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((o=(e.networks??[])[0])==null?void 0:o.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${n(e.baseUrl)}</code> once it is running.</p>`}
        ${_(e)}
        ${a.map(fe).join("")}
        ${O[e.id]?qt(e):""}
      </div>
    `}function _(e){const a=T(e);if(!a)return"";const o=A(e)==="local",c=K(N,a,e.id),h=k[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${n(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${n(a)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a)}">Copy path</button>
        </div>
        ${o?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${u[e.id]?"disabled":""}>
                   ${u[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${h?l(h):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${o?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${N?` (${n(D(N))})`:""}:</span>
          <code class="strip-cmd">${n(c)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(c)}">Copy command</button>
        </div>
      </div>
    `}function l(e){return e.ok?`<div class="strip-line strip-note"><span class="strip-text">${n(e.message)}</span></div>`:`
      <div class="strip-line strip-warn">
        <span class="strip-text">${n(e.message)}</span>
        ${e.ranCommand?`<code class="strip-cmd">${n(e.ranCommand)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.ranCommand)}">Copy</button>`:""}
      </div>
    `}function g(e){const a=[];e.error&&a.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&a.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])a.push({tone:"warn",text:c});a.push(...G(e));const o=I[e.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(B).join("")}</div>`}function B(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${n(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${n(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.cmd)}">Copy</button>`:""}
      </div>
    `}function G(e){var h,S;const a=e.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((h=a.status)==null?void 0:h.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((S=a.status)==null?void 0:S.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const c=U[e.id]??a.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&o.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&o.push({tone:"warn",text:c.expiryWarning}),o}function Q(e){switch(e.status.State){case"running":return M("running","ok");case"created-but-stopped":return M("stopped","warn");case"not-created":return M("not created","neutral");default:return M("unknown","bad")}}function j(e){return e.status.State==="running"?ke("ok"):e.status.State==="unknown"?ke("bad"):ke("neutral")}function re(e,a){const o=Ea[a];if(!o)return"";const c=m[e.id];return`
      <button class="${o.className}" data-action="gw-${a}" data-gid="${n(e.id)}"
              title="${n(o.title)}" ${c?"disabled":""}>
        ${c===a?'<span class="spinner" aria-label="working"></span>':n(o.label)}
      </button>
    `}function ne(e){const a=q[e.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(e.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function Pe(e){const a=Oe(e.networks??[]),o=a.some(c=>c.chainId===Ne);return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(e.id)}">Add a network</button>
            ${rt(e,o)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${a.map(c=>qe(e,c)).join("")}
        </div>
        ${Ee(e,o)}
        ${Ot(e)}
      </div>
    `}function Oe(e){const a=e.filter(c=>c.chainId!==Ne),o=e.filter(c=>c.chainId===Ne);return[...a,...o]}function qe(e,a){const o=It(a),c=a.chainId===Ne,h=`${e.id}:${a.chainId}`,S=x[h]??!1,C=o.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${o.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${n(a.name)}</span>
          <code class="chain-key">evm:${a.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${M(C,o.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${n(h)}" aria-expanded="${S}">
              ${S?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Je(e,a)}
        ${S?Fe(e,a,o):""}
      </section>
    `}function Je(e,a){if(!a.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const o=T(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${n(a.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                title="Copy ${n(a.url)}">Copy URL</button>
        ${o?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${A(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${u[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${u[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(o)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${n(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${k[e.id]?`<span class="chain-cert muted small">${n(k[e.id].ok?"Trusted — reload your wallet or browser.":k[e.id].message)}</span>`:""}`:""}
      </div>
    `}function Fe(e,a,o){const c=a.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${o.why?" chain-verdict-why":""}"${o.why?` title="${n(o.why)}"`:""}>${o.html}</p>
        <div class="chain-detail-bar">
          ${Et(c.length,o.tone,a.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">Remove</button>
        </div>
        ${At(e,a)}
        ${(a.warnings??[]).map(h=>`<p class="chain-note">${n(h)}</p>`).join("")}
      </div>
    `}function Ee(e,a){const o=w[e.id],c=o!=null&&o.at?`probed ${n(ct(o.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${n(e.id)}">+ Network</button>
        ${rt(e,a)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${n(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[e.id]?"disabled":""}>
          ${R[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function rt(e,a){return a?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${n(e.id)}"
                    title="Add a throwaway local test chain (evm:${Ne}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Et(e,a,o){const c=o>0,h=c?o:e,S=Math.min(e,h);let C="";for(let Re=0;Re<h;Re++)C+=`<span class="seg${Re<S?` seg-on seg-${a}`:""}"></span>`;const $=c&&e>o,V=c?$?`${e} (set is ${o})`:`${e} of ${o}`:`${e}`,ee=`${e} upstream${e===1?"":"s"} configured`,he=c?`${ee}${$?`, ${e-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${ee}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(he)}">${C}</span>
      <span class="segs-n">${V}</span>
    `}function It(e){const a=e.upstreams??[];if(a.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!a.some(Rt)){const c=Lt(a);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(S=>`<code>${n(S)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(a.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!a.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=a.filter(c=>!!c.problem);if(o.length>0){const c=a.length-o.length;return{tone:"warn",html:`${o.length} of these ${a.length} endpoints ${o.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function Rt(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Lt(e){const a=new Set;for(const o of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());c&&a.add(c[1].toLowerCase())}return[...a].sort()}function At(e,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(c=>Nt(e,a,c)).join("")}</ul>`}function Nt(e,a,o){const c=`${e.id}|${a.chainId}|${o.id}`,h=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?ke("bad"):ke("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${Bt(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${Ht(e,a,o)}</div>
        <div class="up-share">${Mt(e,a,o)}</div>
        <div class="up-acts">
          ${h.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(c)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${m[e.id]?"disabled":""}>
                   ${m[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(c)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function Bt(e){return e.problem?M("unusable","bad"):e.recentOnly?M("recent blocks","warn"):e.local?M("yours","ok"):M("public","neutral")}function it(e,a){var o;if(e)return a==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(c=>c.key===a))==null?void 0:o.status}function Ht(e,a,o){const c=de(e.id,a.chainId,o.id);return c?c.unprobeable?`<span class="caps-none" title="${n(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Ca.map(h=>Dt(e,a,c,h)).join("")}</span>`:`<span class="muted small">${w[e.id]===void 0?"probing…":"—"}</span>`}function Dt(e,a,o,c){const h=(o.capabilities??[]).find(ee=>ee.key===c),S=it(o,c)??"inconclusive",C=xa[c]??c.toUpperCase();let $="cap";S==="unsupported"?$=Ut(e,a,c)?"cap missing":"cap off":S==="inconclusive"?$="cap unknown":S==="inconsistent"&&($="cap mixed");const V=h!=null&&h.detail?`${h.label}: ${h.detail}`:c==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${C}: no verdict`;return`<span class="${$}" title="${n(V)}">${n(C)}</span>`}function Ut(e,a,o){const c=(a.upstreams??[]).map(h=>de(e.id,a.chainId,h.id)).filter(h=>!!h&&!h.unprobeable);return c.length>0&&c.every(h=>it(h,o)==="unsupported")}function Mt(e,a,o){const c=f[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const h=oe(e.id,a.chainId,o.id),S=(c.networks??[]).find(he=>he.chainId===a.chainId);if(!h||!S||S.attributed===0)return'<span class="muted small">no traffic yet</span>';const C=Math.round(h.actual*100),$=Math.round(h.intended*100),V=h.diverged?o.local?"warn":"":"ok",ee=`${h.succeeded.toLocaleString()} of ${S.attributed.toLocaleString()} answered requests · routing intends ${$}%`+(h.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ee)}">
        <span class="bar">
          <span class="fill${V?" "+V:""}" style="width:${C}%"></span>
          <span class="tick" style="left:${$}%"></span>
        </span>
        <span class="share-n${h.diverged?" warn":""}">${C}%</span>
        ${h.unconfigured?M("not in config","warn"):""}
      </span>
    `}function Ot(e){const a=f[e.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(ct(a.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function ct(e){const a=new Date(e);return Number.isNaN(a.getTime())?e:a.toLocaleString()}function qt(e){const a=e.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${n(e.id)}-port" value="${a.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${n(e.id)}-bind" value="${n(a.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${n(a.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${Ft(e)}
        ${jt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(e.id)}">Save settings</button>
        </div>
      </div>
    `}function Ft(e){const a=!e.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${n(e.id)}-metrics" ${a?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function jt(e){var C;const a=n(e.id),o=e.config.TLS??null,c=(o==null?void 0:o.Enabled)??!1,h=(o==null?void 0:o.CertSource)||"internal",S=((C=e.tls)==null?void 0:C.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${a}-tls" ${c?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${a}-tls-host" value="${n((o==null?void 0:o.Hostname)??S)}"
               placeholder="${n(S||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${S?`<p class="muted small">
               The default is <code>${n(S)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${a}-tls-port" value="${(o==null?void 0:o.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${a}-tls-source">
          <option value="internal" ${h==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${h==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${a}-tls-cert" value="${n((o==null?void 0:o.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${a}-tls-key" value="${n((o==null?void 0:o.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${Wt(e)}
    `}function Wt(e){var C,$;const a=n(e.id),o=((C=e.config.TLS)==null?void 0:C.Enabled)??!1,c=U[e.id]??(($=e.tls)==null?void 0:$.verification)??null,h=L[e.id]??!1,S=P[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!h?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${h?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${S?`<p class="error small">${n(S)}</p>`:""}
      ${c?_t(c):""}
    `}function _t(e){const a=(e.assertions??[]).map(o=>`
          <li class="small">
            ${Kt(o.status)}
            <strong>${n(o.title)}</strong>
            <div class="muted">${n(o.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${e.ok?e.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${n(e.summary)}
      </div>
      <ul class="verify-list">${a}</ul>
      <p class="muted small">
        Checked ${n(new Date(e.at).toLocaleString())} against <code>${n(e.address)}</code>
        ${e.notAfter?`· certificate valid until <code>${n(new Date(e.notAfter).toLocaleString())}</code> (${n(e.expiresIn??"")})`:""}
      </p>
      ${e.expiryWarning?`<div class="banner banner-warn">${n(e.expiryWarning)}</div>`:""}
    `}function Kt(e){switch(e){case"pass":return M("pass","ok");case"fail":return M("fail","bad");case"unavailable":return M("unavailable","warn");default:return M("skipped","neutral")}}async function Gt(e){L[e]=!0,P[e]=null,Y();try{U[e]=await jn(e)}catch(a){P[e]=`${pe(a)}${Ie(a)}`}finally{L[e]=!1,Y()}}function Te(e){return{...e.config,Networks:(e.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function Ce(e,a,o){I[e]=null;try{await kt(e,a)}catch(c){return I[e]=`${o?o+": ":""}${pe(c)}`,Y(),!1}return await W(),!0}async function Vt(e,a){const o=a.dataset.gid??"";switch(e){case"refresh":await W();return;case"copy":a.dataset.copy&&await vn(a,a.dataset.copy);return;case"reprobe":await ue(o,!0);return;case"toggle-settings":O[o]=!O[o],Y();return;case"toggle-manage":H[o]=!H[o],Y();return;case"toggle-chain-detail":{const c=a.dataset.key??"";c&&(x[c]=!x[c]),Y();return}case"save-settings":await zt(o);return;case"verify-tls":await Gt(o);return;case"trust-cert":await Zt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await Xt(o,e.slice(3));return;case"gw-create":case"gw-recreate":await Qt(o);return;case"gw-wipe":mn(o);return;case"add-gateway":gn();return;case"forget-gateway":await en(o);return;case"dismiss-orphan":await tn(a.dataset.name??"");return;case"add-chain":nn(o);return;case"add-devnet":{const c=J(o);if(c){const h=((r==null?void 0:r.targets)??[]).some(S=>S.id===c.placement.targetId&&S.hasDevnet);dt(o,Ne,h)}return}case"remove-chain":await on(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":pt(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await rn(a.dataset.key??"");return;case"reset-devnet":await hn(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function zt(e){const a=J(e);if(!a)return;const o=Te(a),c=s.querySelector(`#gw-${CSS.escape(e)}-port`),h=s.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const $=Number.parseInt(c.value.trim(),10);Number.isFinite($)&&(o.Port=$)}h&&(o.BindAddr=h.value.trim());const S=s.querySelector(`#gw-${CSS.escape(e)}-metrics`);S&&(o.MetricsOff=!S.checked),o.TLS=Jt(e,a);const C=a.status.State==="running";await Ce(e,o,"Saving settings")&&(O[e]=!1,C&&(I[e]=null,Yt(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),Y())}function Jt(e,a){var S,C,$,V,ee,he,Re;const o=$n=>s.querySelector(`#gw-${CSS.escape(e)}-${$n}`),c=o("tls");if(!c)return a.config.TLS??null;const h=Number.parseInt(((S=o("tls-port"))==null?void 0:S.value.trim())??"",10);return{Enabled:c.checked,Hostname:((C=o("tls-host"))==null?void 0:C.value.trim())??"",CertSource:(($=o("tls-source"))==null?void 0:$.value)??"internal",CertFile:((V=o("tls-cert"))==null?void 0:V.value.trim())??"",KeyFile:((ee=o("tls-key"))==null?void 0:ee.value.trim())??"",HTTPSPort:Number.isFinite(h)?h:443,BindAddr:((he=a.config.TLS)==null?void 0:he.BindAddr)??"",ImageRef:((Re=a.config.TLS)==null?void 0:Re.ImageRef)??""}}function Yt(e,a){q[e]=[a]}async function Zt(e){if(!u[e]){u[e]=!0,k[e]=null,Y();try{k[e]=await zn(e)}catch(a){k[e]={ok:!1,message:`${pe(a)}${Ie(a)}`}}u[e]=!1,Y()}}async function Xt(e,a){if(!m[e]){m[e]=a,I[e]=null,Y();try{await Vn(e,a)}catch(o){I[e]=`${a} failed: ${pe(o)}${Ie(o)}`}m[e]=null,await W()}}async function Qt(e){if(m[e])return;m[e]="create",I[e]=null,q[e]=["starting…"],Y();let a;try{a=await St(e)}catch(o){I[e]=`${pe(o)}${Ie(o)}`,q[e]=[],m[e]=null,Y();return}F==null||F(),F=Ve(a.targetId,o=>{if(i)return;const c=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(q[e]=[...(q[e]??[]).filter(S=>S!=="starting…"),c],!!o.err||o.stepId===Pa&&!!o.done){F==null||F(),F=null,m[e]=null,o.err&&(I[e]="Provisioning failed — see the log below."),W();return}Y()})}async function en(e){const a=J(e);if(!(!a||!await De({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await Gn(e)}catch(c){I[e]=pe(c),Y();return}await W()}}async function tn(e){if(e){p[e]=null;try{await Fn(e)}catch(a){p[e]=pe(a),Y();return}await W()}}function nn(e){const a=J(e);if(!a)return;const o=new Set((a.networks??[]).map($=>$.chainId)),c=(r==null?void 0:r.presets)??[],h=c.filter($=>!o.has($.chainId)),S=c.filter($=>o.has($.chainId)),C=((r==null?void 0:r.targets)??[]).some($=>$.id===a.placement.targetId&&$.hasDevnet);ie(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${h.map($=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${$.chainId}">
                <span>${n($.name)}</span>
                <span class="muted small">chain ${$.chainId}${$.devnet?C?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${S.length?`<p class="muted small">Already fronted: ${n(S.map($=>$.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,$=>{if($==="cancel"){X();return}if($==="custom"){an(e);return}if($.startsWith("preset:")){const V=Number.parseInt($.slice(7),10),ee=c.find(he=>he.chainId===V);X(),ee!=null&&ee.devnet?dt(e,V,C):lt(e,V)}})}function an(e){var a;ie(`
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
      `,o=>{if(o==="cancel"){X();return}if(o!=="add")return;const c=document.getElementById("custom-chain-id"),h=document.getElementById("custom-chain-err"),S=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(S)||S<=0){h&&(h.className="error small"),h&&(h.textContent="A chain id is a positive whole number.");return}X(),lt(e,S)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function lt(e,a){const o=J(e);if(!o)return;const c=Te(o),h=c.Networks??[];h.some(S=>S.ChainID===a)||(h.push({ChainID:a,Upstreams:[]}),c.Networks=h,await sn(e,c)&&(Y(),pt(e,a)))}async function sn(e,a){var S;const o={...a,Networks:(a.Networks??[]).filter(C=>C.Upstreams.length>0)};if(!await Ce(e,o))return!1;const h=J(e);if(h)for(const C of a.Networks??[])C.Upstreams.length===0&&!(h.networks??[]).some($=>$.chainId===C.ChainID)&&(h.config.Networks=[...h.config.Networks??[],{ChainID:C.ChainID,Upstreams:[]}],h.networks=[...h.networks??[],{chainId:C.ChainID,name:((S=((r==null?void 0:r.presets)??[]).find($=>$.chainId===C.ChainID))==null?void 0:S.name)??`Chain ${C.ChainID}`,path:`/${h.config.ProjectID}/evm/${C.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function dt(e,a,o){const c=J(e);if(!c)return;if(!o){ie(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${n(c.placement.targetId)}</code>, so adding chain ${a} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(c.placement.targetId)}" data-modal-action="go">Create a devnet on ${n(c.placement.targetId)}</a>
          </div>
        `,()=>X());return}const h=Te(c),S=h.Networks??[],C={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},$=S.find(V=>V.ChainID===a);$?$.Upstreams.push(C):S.push({ChainID:a,Upstreams:[C]}),h.Networks=S,await Ce(e,h,"Adding the devnet")}async function on(e,a){const o=J(e);if(!o||!Number.isFinite(a))return;const c=se(o,a);if(!await De({title:`Remove ${(c==null?void 0:c.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const S=Te(o);S.Networks=(S.Networks??[]).filter(C=>C.ChainID!==a),await Ce(e,S,"Removing the network")}function ut(e){const a=e.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function rn(e){const a=ut(e);if(!a)return;const o=J(a.gid);if(!o)return;const c=Te(o),h=(c.Networks??[]).find($=>$.ChainID===a.chainId);if(!h)return;const S=h.Upstreams.findIndex(($,V)=>($.ID||`${a.chainId}-${V}`)===a.upstreamId);S<0||!await De({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(h.Upstreams.splice(S,1),await Ce(a.gid,c,"Removing the endpoint"))}function pt(e,a){const o=J(e);if(!o||!Number.isFinite(a))return;const c=((r==null?void 0:r.sources)??[]).filter($=>$.chainId===a),h=se(o,a),S=new Set(((h==null?void 0:h.upstreams)??[]).filter($=>$.kind!=="external").map($=>`${$.kind}|${$.targetId??""}`)),C=c.filter($=>!S.has(`${$.kind}|${$.targetId}`));ie(`
        <h2>Add an endpoint for ${n((h==null?void 0:h.name)??`chain ${a}`)}</h2>
        ${C.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${C.map($=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n($.kind)}:${n($.targetId)}">
                       <span>${n($.label)}</span>
                       <span class="muted small">${n($.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${a}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn" data-modal-action="known-set">Add valve's set…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,$=>{if($==="cancel"){X();return}if($==="known-set"){dn(e,a);return}if($==="manual"){pn(e,a);return}if($.startsWith("source:")){const[,V,ee]=$.split(":");X(),cn(e,a,V,ee)}})}async function cn(e,a,o,c){const h=J(e);if(!h)return;const S=Te(h),C=S.Networks??[],$={ID:`${o==="managed-devnet"?"devnet":"node"}-${c}`,Kind:o,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},V=C.find(ee=>ee.ChainID===a);V?V.Upstreams.push($):C.push({ChainID:a,Upstreams:[$]}),S.Networks=C,await Ce(e,S,"Adding the endpoint")}function ln(e){const a=[...e].sort((h,S)=>(h.latencyMs??1e9)-(S.latencyMs??1e9)),o=a.slice(0,3),c=a.find(h=>h.url.startsWith("wss://")||h.url.startsWith("ws://"));return c&&!o.some(h=>h.url===c.url)&&(o.length===3&&o.pop(),o.push(c)),new Set(o.map(h=>h.url))}async function dn(e,a){let o;try{o=await Tt(e,a)}catch($){ie(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(pe($))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>X());return}if(i)return;const c=o.endpoints??[],h=c.filter($=>!$.alreadyAdded).map($=>$.url),S=new Set(c.map($=>$.provider)).size,C=c.map($=>{const V=[$.websocket?'<span class="t ws">websocket</span>':"",$.archive?'<span class="t ar">archive</span>':"",$.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n($.url)}</code>
                  <span class="muted small">${n($.provider)}</span> ${V}</li>`}).join("");ie(`<h2>Endpoints for chain ${a}</h2>
       ${c.length?`<p class="muted small">${S} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${C}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${o.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${h.length?"":" disabled"}>
           ${h.length?`Add ${h.length}`:"Nothing to add"}</button>
       </div>`,$=>{X(),$==="add"&&Ye(e,a,h),$==="discover"&&un(e,a)})}async function un(e,a){ie(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,C=>{C==="cancel"&&X()});let o;try{o=await Yn(a)}catch(C){const $=Ue();if($){const V=document.createElement("p");V.className="error small",V.textContent=`Could not discover endpoints: ${pe(C)}`,$.appendChild(V)}return}if(i)return;const c=(o.endpoints??[]).filter(C=>C.status==="live"||C.status==="unprobed"),h=(o.endpoints??[]).filter(C=>C.status==="rejected"),S=ln(c);ie(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(C=>{const $=S.has(C.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(C.url)}"${$}>
                       <span><code>${n(C.url)}</code></span>
                       <span class="muted small">${C.status==="live"?`answered in ${C.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${h.length?`<details class="rpc-rejected">
                 <summary class="muted small">${h.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${h.map(C=>`<li class="muted small"><code>${n(C.url)}</code> — ${n(C.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,C=>{if(C==="cancel"){X();return}if(C==="add"){const $=Ue(),V=$?Array.from($.querySelectorAll('input[type="checkbox"]:checked')).map(ee=>ee.value):[];X(),Ye(e,a,V);return}})}function pn(e,a){var o;ie(`
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
      `,c=>{if(c==="cancel"){X();return}if(c!=="add")return;const h=document.getElementById("manual-endpoint"),S=document.getElementById("manual-recent"),C=document.getElementById("manual-err"),$=(h==null?void 0:h.value.trim())??"";if(!/^(https?|wss?):\/\//i.test($)){C&&(C.className="error small",C.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}X(),Ye(e,a,[$],(S==null?void 0:S.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ye(e,a,o,c=!1){if(!o.length)return;const h=J(e);if(!h)return;const S=Te(h),C=S.Networks??[];let $=C.find(ee=>ee.ChainID===a);$||($={ChainID:a,Upstreams:[]},C.push($));let V=1;for(const ee of $.Upstreams){const he=/^public-\d+-(\d+)$/.exec(ee.ID??"");he&&(V=Math.max(V,Number(he[1])+1))}for(const ee of o)$.Upstreams.some(he=>he.Endpoint===ee)||$.Upstreams.push({ID:`public-${a}-${V++}`,Kind:"external",Endpoint:ee,Local:!1,RecentOnly:c});S.Networks=C,await Ce(e,S,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function hn(e,a){const o=ut(e);if(!o||!a||!await De({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;m[o.gid]="reset",I[o.gid]=null,Y();let h;try{h=await On(a)}catch(S){I[o.gid]=`Reset failed: ${pe(S)}${Ie(S)}`,m[o.gid]=null,Y();return}m[o.gid]=null,fn(a,h),await W()}function fn(e,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const c=a.report.Cascaded??[],h=a.report.CascadeSkipped??[];ie(`
        <h2>Devnet on ${n(e)} reset</h2>
        <ul class="plain-list">${o.map(S=>`<li>${n(S)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${n(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${h.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(h.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>X())}function mn(e){const a=J(e);if(!a)return;ie(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(e)}</button>
        </div>
      `,h=>{if(h==="cancel"||h==="close"){X(),W();return}h==="confirm"&&bn(e)});const o=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{c&&(c.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function bn(e){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await Jn(e)}catch(c){const h=Ue();if(h){const S=document.createElement("p");S.className="error small",S.textContent=`Wipe failed: ${pe(c)}${Ie(c)}`,h.appendChild(S)}a&&(a.disabled=!1,a.textContent=`Wipe ${e}`);return}ie(`
        <h2>${n(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{X(),W()})}function ht(e,a){return!a.some(o=>{var c;return((c=o.placement)==null?void 0:c.targetId)===e})}function gn(){var S;const e=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=e.filter(C=>ht(C.id,a)),c=new Set(a.map(C=>C.id));if(e.length===0){ie(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>X());return}if(o.length===0){ie(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>X());return}const h=c.has("default")?"":"default";ie(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(h)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${o.map(C=>`<option value="${n(C.id)}">${n(C.id)} (${n(C.mode)})</option>`).join("")}
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
      `,C=>{if(C==="cancel"){X();return}C==="create"&&yn()}),(S=document.getElementById("new-gw-id"))==null||S.focus()}async function yn(){const e=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),h=(e==null?void 0:e.value.trim())??"",S=(a==null?void 0:a.value)??"",C=Number.parseInt((o==null?void 0:o.value.trim())??"",10),$=V=>{c&&(c.className="error small",c.textContent=V)};if(!h){$("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!S){$("Pick the machine it runs on.");return}try{await wt({id:h,placement:{targetId:S,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(C)?C:4e3,Networks:[]}})}catch(V){$(pe(V));return}X(),await W()}async function vn(e,a){const o=await Me(a),c=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(e.textContent=c)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Ie(e){return e instanceof $e&&e.hint?` — ${e.hint}`:""}return()=>{i=!0,F==null||F(),X()}}const Ra="local";function La(s){let i=!1,r=!1,t="",f=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${le()}
  `;const w=s.querySelector("#targets-body");ge(s,(p,u)=>{O(p,u)}),R();async function R(){try{const[p,u,k]=await Promise.all([ve(),Se(),at()]);if(i)return;t=k.os,I(p,u)}catch(p){if(i)return;w.innerHTML=`<p class="error">Failed to load machines: ${n(String(p))}</p>`}}function m(){f&&I(f.targets,f.catalog)}function I(p,u){f={targets:p,catalog:u};const k=t==="linux",N=[...p].sort((W,ae)=>(W.mode==="local"?-1:0)-(ae.mode==="local"?-1:0)),F=N.length?`<div class="card-grid">${N.map(W=>Aa(W,u,W.mode!=="local"||k,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Z=p.some(W=>W.mode==="local");w.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${F}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${q(k,Z)}
        ${r?Na():""}
      </section>
    `;const te=w.querySelector("#fleet-verdict");te&&xt(te,Ct(p,u))}function q(p,u){const k=`
      <div class="card">
        <h3>A server over SSH ${M("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${p?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${p?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,N=p?`
        <div class="card">
          <h3>This machine ${M("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${n(t)})`:""} ${M("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return u?`<div class="card-grid card-grid-wide">${k}</div>`:`<div class="card-grid card-grid-wide">${p?N+k:k+N}</div>`}async function O(p,u){var k;if(p==="add-local"){await x();return}if(p==="delete-target"){const N=u.dataset.id;if(!N||!await De({title:"Remove machine",body:`Remove "${N}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await H(N);return}if(p==="toggle-ssh"){r=!r,P(),m(),r&&((k=s.querySelector("#ssh-host"))==null||k.focus());return}p==="add-ssh"&&await U()}async function x(){P();try{await et({id:Ra,mode:"local"}),await R()}catch(p){L(p)}}async function H(p){try{await Sn(p),await R()}catch(u){L(u)}}async function U(){const p=s.querySelector("#ssh-host"),u=s.querySelector("#ssh-user"),k=s.querySelector("#ssh-key"),N=s.querySelector("#ssh-port"),F=s.querySelector("#ssh-id");if(!p||!u||!k||!N||!F)return;const Z=p.value.trim(),te=u.value.trim(),W=k.value.trim(),ae=N.value.trim(),ue=F.value.trim();if(P(),!Z||!te||!W){L(new Error("host, user, and key path are required"));return}const J=ue||Ba(Z),se={Host:Z,User:te,KeyPath:W};if(ae){const de=Number.parseInt(ae,10);if(!Number.isFinite(de)||de<=0){L(new Error("port must be a positive number"));return}se.Port=de}const oe=s.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await et({id:J,mode:"ssh",ssh:se}),r=!1,await R()}catch(de){L(de),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function L(p){let u=s.querySelector("#targets-error");u||(w.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),u=s.querySelector("#targets-error")),u.textContent=String(p instanceof Error?p.message:p)}function P(){var p;(p=s.querySelector("#targets-error"))==null||p.remove()}return()=>{i=!0}}function Aa(s,i,r,t){const f=s.wire,w=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:w;let m;if(!f&&!r)m=`${M("can't run a node","warn")} ${M(t||"not Linux","neutral")}`;else if(!f)m=M("not set up","neutral");else{const I=i.networks.find(O=>O.ChainID===f.ChainID),q=I?I.Name:`chain ${f.ChainID}`;m=`${M(q,"ok")} ${M(f.ExecID,"neutral")} ${M(f.BeaconID,"neutral")}${f.Archive?" "+M("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${R}</p>
      <p>${m}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(s.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function Na(){return`
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
  `}function Ba(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Ha=document.querySelector("#app"),{contentEl:Da,setActiveNav:Ua}=Qn(Ha);let me=null;function Ma(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"home"};const[r,t]=i;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:t?decodeURIComponent(t):void 0}:{screen:r??"targets"}}function we(s){const i=document.createElement("div");return Da.replaceChildren(i),s(i)}function Pt(){if(me){try{me()}catch{}me=null}const{screen:s,id:i}=Ma();switch(Ua(s),s){case"machine":if(!i){location.hash="#/targets";return}me=we(r=>$a(r,i));break;case"setup":case"dash":case"logs":case"services":if(!i){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(i)}`;return;case"security":if(!i){location.hash="#/targets";return}me=we(r=>wa(r,i));break;case"diag":if(!i){location.hash="#/targets";return}me=we(r=>sa(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}me=we(r=>aa(r,i));break;case"rpc":me=we(r=>Ia(r));break;case"settings":me=we(r=>Ta(r));break;case"targets":me=we(r=>La(r));break;case"home":default:me=we(r=>da(r));break}}window.addEventListener("hashchange",Pt);Pt();
