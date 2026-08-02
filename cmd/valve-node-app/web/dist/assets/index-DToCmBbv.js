var $n=Object.defineProperty;var wn=(a,r,i)=>r in a?$n(a,r,{enumerable:!0,configurable:!0,writable:!0,value:i}):a[r]=i;var je=(a,r,i)=>wn(a,typeof r!="symbol"?r+"":r,i);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const u of document.querySelectorAll('link[rel="modulepreload"]'))t(u);new MutationObserver(u=>{for(const y of u)if(y.type==="childList")for(const S of y.addedNodes)S.tagName==="LINK"&&S.rel==="modulepreload"&&t(S)}).observe(document,{childList:!0,subtree:!0});function i(u){const y={};return u.integrity&&(y.integrity=u.integrity),u.referrerPolicy&&(y.referrerPolicy=u.referrerPolicy),u.crossOrigin==="use-credentials"?y.credentials="include":u.crossOrigin==="anonymous"?y.credentials="omit":y.credentials="same-origin",y}function t(u){if(u.ep)return;u.ep=!0;const y=i(u);fetch(u.href,y)}})();function wt(){return z("/api/host")}function Ce(){return z("/api/catalog")}function xe(){return z("/api/targets")}function pt(a){return z("/api/targets",{method:"POST",headers:be,body:JSON.stringify(a)})}function kn(a){return z(`/api/targets/${encodeURIComponent(a)}`,{method:"DELETE"})}function Tn(a,r){return z(`/api/targets/${encodeURIComponent(a)}/disk?path=${encodeURIComponent(r)}`)}function Sn(a,r){return z(`/api/targets/${encodeURIComponent(a)}/setup`,{method:"POST",headers:be,body:JSON.stringify(r)})}function Ge(a,r){const i=new EventSource(`/api/targets/${encodeURIComponent(a)}/setup/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function Cn(a,r){const i=new EventSource(`/api/targets/${encodeURIComponent(a)}/monitor/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function xn(a,r=200){return z(`/api/targets/${encodeURIComponent(a)}/logs?n=${r}`)}function Pn(a,r){const i=new EventSource(`/api/targets/${encodeURIComponent(a)}/logs/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function ht(a,r){const i=r===void 0?{}:{lines:r};return z(`/api/targets/${encodeURIComponent(a)}/explain`,{method:"POST",headers:be,body:JSON.stringify(i)})}function En(a,r,i){return z(`/api/targets/${encodeURIComponent(a)}/services/${r}/${i}`,{method:"POST"})}function In(a,r){return z(`/api/targets/${encodeURIComponent(a)}/services/${r}/clear`,{method:"POST",headers:be,body:JSON.stringify({Confirm:r})})}function Rn(a){return z(`/api/targets/${encodeURIComponent(a)}/du`)}function Ln(a){return z(`/api/targets/${encodeURIComponent(a)}/endpoints`)}function An(a){return z(`/api/targets/${encodeURIComponent(a)}/firewall`)}function Nn(a){return z(`/api/targets/${encodeURIComponent(a)}/diagnostics`)}function Bn(a){return z(`/api/targets/${encodeURIComponent(a)}/diagnostics/latest`)}function Hn(a){return z(`/api/targets/${encodeURIComponent(a)}/containers`)}function Dn(a,r,i){return z(`/api/targets/${encodeURIComponent(a)}/containers/${r}/${i}`,{method:"POST"})}async function Mn(a,r){const i=await fetch(`/api/targets/${encodeURIComponent(a)}/containers/${r}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:r})}),t=await i.text();let u=null;try{u=t?JSON.parse(t):null}catch{}if(u&&typeof u=="object"&&"report"in u)return u;const y=u&&typeof u=="object"&&typeof u.error=="string"?u.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,y)}function Un(a,r){return z(`/api/targets/${encodeURIComponent(a)}/containers/${r}/provision`,{method:"POST"})}async function On(a){const r=await fetch(`/api/targets/${encodeURIComponent(a)}/containers/devnet/reset`,{method:"POST",headers:be}),i=await r.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const u=t&&typeof t=="object"&&typeof t.error=="string"?t.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,u)}function qn(a,r,i){return z(`/api/targets/${encodeURIComponent(a)}/containers/${r}/config`,{method:"PUT",headers:be,body:JSON.stringify(i)})}function nt(){return z("/api/gateways")}async function jn(a){await z(`/api/orphans/${encodeURIComponent(a)}`,{method:"DELETE"})}function Fn(a){return z("/api/gateways",{method:"POST",headers:be,body:JSON.stringify(a)})}function Wn(a){return z(`/api/gateways/${encodeURIComponent(a)}/tls/verify`)}function _n(a){return z(`/api/gateways/${encodeURIComponent(a)}/traffic`)}function Kn(a){return z(`/api/gateways/${encodeURIComponent(a)}/analytics`)}function Vn(a,r=!1){const i=r?"?refresh=1":"";return z(`/api/gateways/${encodeURIComponent(a)}/capabilities${i}`)}function Gn(a){return z(`/api/gateways/${encodeURIComponent(a)}`,{method:"DELETE"})}function zn(a,r){return z(`/api/gateways/${encodeURIComponent(a)}/config`,{method:"PUT",headers:be,body:JSON.stringify(r)})}function kt(a,r){return z(`/api/gateways/${encodeURIComponent(a)}/${r}`,{method:"POST"})}function Jn(a){return z(`/api/gateways/${encodeURIComponent(a)}/trust-cert`,{method:"POST"})}function Tt(a){return z(`/api/gateways/${encodeURIComponent(a)}/provision`,{method:"POST"})}async function St(a){const r=await fetch(`/api/gateways/${encodeURIComponent(a)}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:a})}),i=await r.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const u=t&&typeof t=="object"&&typeof t.error=="string"?t.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,u)}function Yn(a){return z(`/api/chainlist/${a}`)}function Zn(a,r){return z(`/api/gateways/${encodeURIComponent(a)}/knownset/${r}`)}function Xn(){return z("/api/settings")}function Qn(a){return z("/api/settings",{method:"PUT",headers:be,body:JSON.stringify(a)})}class we extends Error{constructor(i,t,u,y){super(t);je(this,"status");je(this,"hint");je(this,"code");this.name="ApiError",this.status=i,this.hint=u,this.code=y}}const be={"Content-Type":"application/json"};async function z(a,r){const i=await fetch(a,r);if(!i.ok){let u=i.statusText||`HTTP ${i.status}`,y,S;try{const f=await i.json();f&&typeof f.error=="string"&&f.error&&(u=f.error),f&&typeof f.hint=="string"&&f.hint&&(y=f.hint),f&&typeof f.code=="string"&&f.code&&(S=f.code)}catch{}throw new we(i.status,u,y,S)}if(i.status===204)return;const t=await i.text();return t?JSON.parse(t):void 0}const ft="https://learn.valve.city/rpc";function n(a){return a.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function de(a,r){const i=a&&r&&r!==ft?` <span class="footer-sep">·</span> <a href="${n(r)}" target="_blank" rel="noopener noreferrer">${n(a)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(ft)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${i}
    </footer>
  `}function ea(a){a.innerHTML=`
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
  `;const r=a.querySelector("#content"),i=Array.from(a.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:u=>{const y=u==="machine"?"targets":u==="home"||u==="panel"?"rpc":u;for(const S of i)S.classList.toggle("active",S.dataset.nav===y)}}}function ce(a){return Number.isFinite(a)?a.toLocaleString("en-US"):"—"}function ta(a){return Number.isFinite(a)?`${a.toFixed(1)}%`:"—"}function na(a){if(!Number.isFinite(a)||a<0)return"—";if(a<60)return`~${Math.round(a)}s`;const r=Math.round(a/60),i=Math.floor(r/60),t=r%60;if(i===0)return`~${t}m`;if(i<48)return`~${i}h ${t}m`;const u=Math.floor(i/24),y=i%24;return`~${u}d ${y}h`}function U(a,r){return`<span class="badge badge-${r}">${n(a)}</span>`}function $e(a){return`<span class="dot dot-${a}"></span>`}const mt=["B","KB","MB","GB","TB","PB"];function Se(a){if(!Number.isFinite(a)||a<0)return"—";if(a===0)return"0 B";let r=a,i=0;for(;r>=1024&&i<mt.length-1;)r/=1024,i++;const t=r<10?2:r<100?1:0;return`${r.toFixed(t)} ${mt[i]}`}async function De(a){try{return await navigator.clipboard.writeText(a),!0}catch{return!1}}function ye(a,r){a.addEventListener("click",i=>{const t=i.target.closest("[data-action]");if(!t||!a.contains(t))return;const u=t.dataset.action;u&&r(u,t,i)})}function et(a,r,i){const t=r.find(y=>y.value===i),u=r.map(y=>`
      <li class="dropdown-option${y.value===i?" selected":""}" role="option"
          aria-selected="${y.value===i}" data-value="${n(y.value)}">
        ${n(y.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(a)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${u}</ul>
    </div>
  `}function Ne(a){a.querySelectorAll(".dropdown.open").forEach(r=>{var i;r.classList.remove("open"),(i=r.querySelector(".dropdown-trigger"))==null||i.setAttribute("aria-expanded","false")})}function at(a,r){a.addEventListener("click",u=>{const y=u.target,S=y.closest(".dropdown-trigger");if(S&&a.contains(S)){const A=S.closest(".dropdown"),M=!!A&&!A.classList.contains("open");Ne(a),A&&M&&(A.classList.add("open"),S.setAttribute("aria-expanded","true"));return}const f=y.closest(".dropdown-option");if(f&&a.contains(f)){const A=f.closest(".dropdown");Ne(a),r((A==null?void 0:A.dataset.dropdown)??"",f.dataset.value??"");return}Ne(a)});const i=u=>{if(!a.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}const y=u.target;(!y.closest(".dropdown")||!a.contains(y))&&Ne(a)},t=u=>{if(!a.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}u.key==="Escape"&&Ne(a)};document.addEventListener("click",i),document.addEventListener("keydown",t)}const ze="app-modal";let Ve=null;function ie(a,r){Z();const i=document.createElement("div");i.className="modal-overlay",i.id=ze,i.innerHTML=`<div class="modal">${a}</div>`,i.addEventListener("click",u=>{const y=u.target.closest("[data-modal-action]");y!=null&&y.dataset.modalAction?r(y.dataset.modalAction):u.target===i&&r("cancel")});const t=u=>{u.key==="Escape"&&r("cancel")};document.addEventListener("keydown",t),Ve=t,document.body.appendChild(i)}function Z(){var a;(a=document.getElementById(ze))==null||a.remove(),Ve&&(document.removeEventListener("keydown",Ve),Ve=null)}function He(){return document.querySelector(`#${ze} .modal`)}function Ae(a){return new Promise(r=>{var u;let i=!1;const t=y=>{i||(i=!0,Z(),r(y))};ie(`
        <h2>${n(a.title)}</h2>
        <p>${n(a.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${a.danger?" btn-danger":""}" data-modal-action="confirm">${n(a.confirmLabel)}</button>
        </div>
      `,y=>t(y==="confirm")),(u=document.querySelector(`#${ze} [data-modal-action="confirm"]`))==null||u.focus()})}const Ze=5e3,aa=60;function sa(a,r){let i=!1,t=null,u=null,y=null,S=null;const f=[];a.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${de()}</div>`;const A=a.querySelector("#an-body");ye(a,(v,d)=>{var C;v==="toggle-endpoint"&&((C=d.closest(".an-endpoint"))==null||C.classList.toggle("expanded"))}),M();async function M(){try{t=((await nt()).gateways??[]).find(d=>d.id===r)??null}catch(v){if(i)return;y=String(v instanceof Error?v.message:v),D();return}if(!i){if(!t){D();return}await j(),S=window.setInterval(()=>void j(),Ze)}}async function j(){try{const v=await Kn(r);if(i)return;E(v),u=v,y=null}catch(v){if(i)return;y=String(v instanceof Error?v.message:v)}D()}function E(v){if(!v.enabled||v.error)return;const d=f[f.length-1];d&&d.since!==v.since&&(f.length=0);const C=new Map;for(const R of v.networks??[])C.set(R.chainId,R.received);f.push({t:Date.now(),since:v.since,received:C}),f.length>aa&&f.shift()}function D(){i||(A.innerHTML=O())}function O(){return y&&!u?`<h1>Analytics</h1><p class="error">${n(y)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${L(t)}
      ${u?p(u):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(r)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function L(v){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${n(v.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${T()}</div>
      </div>
    `}function T(){if(!u)return"";if(!u.enabled)return"counters off";if(u.error)return"could not be read";const v=u.since?new Date(u.since):null;return v&&!Number.isNaN(v.getTime())?`totals since the gateway started, ${n(v.toLocaleString())}<br />re-read every ${Ze/1e3}s`:`re-read every ${Ze/1e3}s`}function p(v){return v.enabled?v.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(v.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:m(v)+le(v):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function m(v){const d=v.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(C=>x(C)).join("")}
      </section>
    `}function x(v){const d=v.methods??[],C=v.endpoints??[],R=v.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${v.chainId}</span>
          <span class="band-name">${n(v.name)}</span>
          ${F(v)}
        </div>
        <div class="an-stats">
          ${B("Received",ce(v.received),"what clients asked this chain for")}
          ${B("Answered",ce(v.answered),"returned by one of your endpoints")}
          ${B("From cache",ce(v.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${B("Failed",ce(v.failed),"asked for and never answered",v.failed>0?"bad":"")}
        </div>
        ${te(v.chainId)}
        ${R?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ae("Method",d.map(H=>({label:H.method,l:H})))+ae("Endpoint",C.map(H=>({label:H.upstream,l:H})))+W(v)}
      </div>
    `}function B(v,d,C,R=""){return`
      <div class="an-stat${R?" an-stat-"+R:""}" title="${n(C)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(v)}</span>
      </div>
    `}function F(v){const d=X(v.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const C=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${C}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${C}s</span>
    </span>`}function X(v){if(f.length<2)return null;const d=f[0],C=f[f.length-1],R=(C.t-d.t)/1e3;if(R<=0)return null;const H=(C.received.get(v)??0)-(d.received.get(v)??0);return H<0?null:H/R}function te(v){if(f.length<3)return"";const d=[];for(let $=1;$<f.length;$++){const I=f[$-1],_=f[$],l=(_.t-I.t)/1e3,g=(_.received.get(v)??0)-(I.received.get(v)??0);d.push(l>0&&g>=0?g/l:0)}const C=Math.max(...d);if(C<=0)return"";const R=240,H=28,K=d.length>1?R/(d.length-1):R,b=d.map(($,I)=>`${(I*K).toFixed(1)},${(H-$/C*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${C.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${R} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(C.toFixed(2))} req/s</span>
      </div>
    `}function W(v){const d=[];return v.cached.count>0&&d.push(`${n(ce(v.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${v.cached.mean===null?"":`, in ${n(Be(v.cached.mean))} on average`}.`),v.failedLatency.count>0&&v.failedLatency.mean!==null&&d.push(`The ${n(ce(v.failedLatency.count))} that failed took
         ${n(Be(v.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function ae(v,d){return d.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${n(v)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${d.map(C=>ue(C.label,C.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function ue(v,d){return`
      <tr>
        <td><code>${n(v)}</code></td>
        <td class="an-num">${ce(d.count)}</td>
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(Be(d.mean))}</td>
        <td>${J(d)}</td>
      </tr>
    `}function J(v){const d=v.buckets??[];if(d.length===0||v.count===0)return'<span class="muted small">—</span>';let C=0;const R=[];for(const K of d){const b=K.count-C;C=K.count,R.push({label:oe(K.le),n:Math.max(0,b)})}return R.reduce((K,b)=>K+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(R.filter(K=>K.n>0).map(K=>`${K.n} ${K.label}`).join(" · "))}">
        ${R.map((K,b)=>K.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${K.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(se(R))}</span>
    `}function se(v){for(let d=v.length-1;d>=0;d--)if(v[d].n>0)return`slowest ${v[d].label}`;return""}function oe(v){if(v==="+Inf")return"30s or more";const d=Number(v);return Number.isFinite(d)?`under ${Be(d)}`:`under ${v}`}function le(v){const d=v.endpoints??[];return`
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
                     <tbody>${d.map(C=>Y(C)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function Y(v){const d=v.errors??[],C=d.reduce((H,K)=>H+K.count,0),R=d.length>0;return`
      <tr class="an-endpoint${R?" expandable":""}" ${R?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(v.upstream)}</code>
          ${v.chainId?`<span class="muted small">chain ${v.chainId}</span>`:""}
          ${v.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ce(v.requests)}</td>
        <td class="an-num${C>0?" bad":""}">${C>0?ce(C):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${v.headLag>0?ce(v.headLag):'<span class="muted">0</span>'}</td>
        <td>${me(v)}</td>
      </tr>
      ${R?ge(v,d):""}
    `}function me(v){const d=[];return v.scored?(d.push(v.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(v.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(v.score.toFixed(3))}</span>`),v.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ce(v.primarySwitches)} switches</span>`),v.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Be(v.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(v,d){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${d.map(C=>`
                    <tr>
                      <td class="an-num">${ce(C.count)}</td>
                      <td><code>${n(C.class)}</code></td>
                      <td>${C.severity?`<span class="badge badge-${C.severity==="critical"?"bad":"warn"}">${n(C.severity)}</span>`:""}</td>
                      <td class="muted small">${n(C.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${n(v.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{i=!0,S!==null&&window.clearInterval(S)}}function Be(a){return!Number.isFinite(a)||a<0?"—":a>0&&a<5e-4?"<1ms":a<1?`${Math.round(a*1e3)}ms`:a<60?`${a<10?a.toFixed(1):Math.round(a)}s`:`${Math.round(a/60)}m`}function oa(a,r){let i=!1,t=null,u=null,y=!1,S=!1;a.innerHTML=`<h1>Network diagnostics: ${n(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${de()}</div>`;const f=a.querySelector("#diag-body"),A=a.querySelector("#diag-footer");ye(a,(p,m)=>{var x;if(p==="run")j();else if(p==="toggle")(x=m.closest(".check-item"))==null||x.classList.toggle("expanded");else if(p==="copy"){const B=m.dataset.copy;B&&T(m,B)}}),M();async function M(){let p,m;try{const[B,F]=await Promise.all([xe(),Ce()]);p=B.find(X=>X.id===r),m=F}catch(B){if(i)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(B))}</p>`;return}if(i)return;if(!p){f.innerHTML=`<p class="error">Target "${n(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const x=m==null?void 0:m.networks.find(B=>B.ChainID===p.wire.ChainID);x&&(A.innerHTML=de(x.Name,x.LearnURL));try{t=await Bn(r),S=!0}catch(B){u=String(B instanceof Error?B.message:B)}i||E()}async function j(){y=!0,u=null,E();try{t=await Nn(r),S=!0}catch(p){u=String(p instanceof Error?p.message:p)}y=!1,i||E()}function E(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${y?"disabled":""}>${y?"Running…":"Run diagnostics"}</button>
      </div>
      ${u?`<p class="error">${n(u)}</p>`:""}
      ${D()}
    `}function D(){if(!S&&!u)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const p=new Date(t.at).toLocaleString(),m=t.failedId?`<p><strong>Failed at: ${n(O(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(p)} — trigger: ${n(t.trigger)}</p>
      ${m}
      <ul class="check-list">${t.items.map(L).join("")}</ul>
    `}function O(p){var m;return((m=t==null?void 0:t.items.find(x=>x.ID===p))==null?void 0:m.Title)??p}function L(p){const m=p.Status==="pass"?"ok":p.Status==="fail"?"bad":p.Status==="warn"?"warn":"neutral",x=p.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${x?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${U(x?"failed here":p.Status,m)}
          <strong>${n(p.Title)}</strong>
          <span class="muted small check-detail-inline">${n(p.Detail)}</span>
        </button>
        <div class="check-body">
          <details${x?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(p.Why)}</p>
          </details>
          ${p.Fix?`
                <details${x?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(p.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(p.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(p,m){const x=await De(m),B=p.textContent;p.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{i||(p.textContent=B)},1500)}return()=>{i=!0}}const ra=85,Xe={exec:"Execution",beacon:"Beacon"};function ia(a,r){let i=!1,t=null,u=null,y=null,S=null,f=null,A=null,M=null,j=null;const E={exec:null,beacon:null};let D=null;a.innerHTML=`<h1>Dashboard: ${n(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${de()}</div>`;const O=a.querySelector("#dash-body"),L=a.querySelector("#dash-footer");O.addEventListener("click",d=>{const C=d.target.closest("[data-action]");if(!C||!O.contains(C))return;const R=C.dataset.action;if(R==="svc-action"){const H=C.dataset.svc,K=C.dataset.kind;H&&K&&Y(H,K)}else if(R==="open-clear"){const H=C.dataset.svc;H&&ge(H)}else if(R==="copy"){const H=C.dataset.copy;H&&me(C,H)}else R==="retry-du"?p():R==="retry-endpoints"&&m()}),T();async function T(){let d,C;try{const[H,K]=await Promise.all([xe(),Ce()]);d=H.find(b=>b.id===r),C=K}catch(H){if(i)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(i)return;if(!d){O.innerHTML=`<p class="error">Target "${n(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const R=C==null?void 0:C.networks.find(H=>H.ChainID===d.wire.ChainID);R&&(L.innerHTML=de(R.Name,R.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',t=Cn(r,H=>{i||(x(H),u=H,y=H,B())}),p(),m()}async function p(){A=null;try{f=await Rn(r)}catch(d){f=null,A=String(d instanceof Error?d.message:d)}i||B()}async function m(){j=null;try{M=await Ln(r)}catch(d){M=null,j=String(d instanceof Error?d.message:d)}i||B()}function x(d){if(!u)return;const C=(new Date(d.at).getTime()-new Date(u.at).getTime())/1e3,R=d.execHead-u.execHead;if(C>0&&R>=0){const H=R/C;S=S===null?H:S*.7+H*.3}}function B(){if(!y)return;const d=y;O.innerHTML=`
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
    `}function F(d){return!d.execActive&&!d.beaconActive?U("Node not running","bad"):d.execSyncing||d.beaconDistance>0?U("Syncing","warn"):U("Running · synced","ok")}function X(d){const R=d.refHead>0?d.refHead-d.execHead:null,H=R!==null&&R>0&&S&&S>0?na(R/S):R!==null&&R<=0?"caught up":"—";return{lag:R,eta:H}}function te(d){const{lag:C,eta:R}=X(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?U("syncing","warn"):d.execHead===0?U("no data","neutral"):U("synced","ok"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ce(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${C!==null?ce(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${C!==null?ce(Math.max(C,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${R}</dd></div>
        </dl>
      </div>
    `}function W(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?U("no data","neutral"):d.beaconDistance===0?U("synced","ok"):U("syncing","warn"):U("stopped","bad")}</p>
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
    `}function ue(d){const C=d.diskUsedPct>=ra,R=`
      <div class="meter"><div class="meter-fill ${C?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${ta(d.diskUsedPct)} used</p>
    `;if(A)return`
        <div class="card ${C?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="error small">${n(A)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${C?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="muted">Loading…</p>
        </div>
      `;const H=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,K=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:b,eta:$}=X(d),I=b!==null&&b>0&&S!==null&&S>0;return`
      <div class="card ${C?"card-warn":""}">
        <h3>Storage</h3>
        ${R}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Se(f.ExecBytes)} of ~${Se(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${I?`<p class="muted small">Estimated time remaining: ${n($)}</p>`:""}
        <p class="muted small">Beacon — ${Se(f.BeaconBytes)} of ~${Se(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${K}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Se(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function J(){if(j)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(j)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!M)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=M,C=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",R=d.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(d.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${$e(d.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${$e(d.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.BeaconHTTP)}">Copy</button>
        </div>
        ${C}
        ${R}
      </div>
    `}function se(d,C){const R=Xe[d],H=E[d],K=(b,$,I)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${b}" ${H!==null||I?"disabled":""}>${H===b?le():n($)}</button>`;return`
      <div class="service-row">
        <span>${n(R)} ${C?U("active","ok"):U("down","bad")}</span>
        <div class="service-actions">
          ${K("start","Start",C)}
          ${K("stop","Stop",!C)}
          ${K("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${d}" ${H!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(d){return`
      <div class="card">
        <h3>Services</h3>
        ${se("exec",d.execActive)}
        ${se("beacon",d.beaconActive)}
        ${D?`<p class="error small">${n(D)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function le(){return'<span class="spinner" aria-label="working"></span>'}async function Y(d,C){if(E[d]===null){E[d]=C,D=null,B();try{await En(r,d,C)}catch(R){D=`${Xe[d]} ${C} failed: ${R instanceof Error?R.message:String(R)}`}E[d]=null,i||B()}}async function me(d,C){const R=await De(C),H=d.textContent;d.textContent=R?"Copied!":"Copy failed",setTimeout(()=>{i||(d.textContent=H)},1500)}function ge(d){const C=Xe[d],R=f?Se(d==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";ie(`
        <h2>Clear ${n(C)} data</h2>
        <p class="error">
          This stops the ${n(C.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(R)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,b=>{if(b==="cancel"){Z();return}b==="confirm"&&v(d)});const H=document.getElementById("clear-confirm-input"),K=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{K&&(K.disabled=H.value.trim()!==d)}),H==null||H.focus()}async function v(d){const C=document.getElementById("clear-confirm-btn");C&&(C.disabled=!0,C.textContent="Clearing…");try{await In(r,d),Z(),p()}catch(R){const H=He();if(H){const K=document.createElement("p");K.className="error small",K.textContent=`Clear failed: ${R instanceof Error?R.message:String(R)}`,H.appendChild(K)}C&&(C.disabled=!1,C.textContent="Clear and resync")}}return()=>{i=!0,t==null||t(),Z()}}const bt=500,yt="valve-node-app.explain-consent";function ca(a,r){let i=!1,t=null;const u=[];a.innerHTML=`
    <h1>Logs: ${n(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${de()}</div>
  `;const y=a.querySelector("#logs-body"),S=a.querySelector("#logs-footer");ye(a,T=>{T==="explain"&&j()}),f();async function f(){let T,p;try{const[x,B]=await Promise.all([xe(),Ce()]);T=x.find(F=>F.id===r),p=B}catch(x){if(i)return;y.innerHTML=`<p class="error">Failed to load target: ${n(String(x))}</p>`;return}if(i)return;if(!T){y.innerHTML=`<p class="error">Target "${n(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!T.wire){y.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const m=p==null?void 0:p.networks.find(x=>x.ChainID===T.wire.ChainID);m&&(S.innerHTML=de(m.Name,m.LearnURL));try{const x=await xn(r,200);if(i)return;u.push(...x)}catch(x){if(i)return;y.innerHTML=`<p class="error">Failed to load logs: ${n(String(x))}</p>`;return}A(),t=Pn(r,x=>{i||(u.push(x),u.length>bt&&u.splice(0,u.length-bt),A())})}function A(){const T=u.filter(m=>m.severity==="error"||m.severity==="critical");y.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${u.map(M).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${U(String(T.length),T.length?"bad":"neutral")}</h2>
          <div class="log-lines">${T.length?T.slice().reverse().map(M).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const p=y.querySelector(".log-lines");p&&(p.scrollTop=p.scrollHeight)}function M(T){const p=T.severity||"info",m=T.learnUrl?` <a href="${n(T.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(p)}">
        <span class="log-time">${n(new Date(T.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(T.unit)}</span>
        <span class="log-sev">${n(p)}</span>
        <span class="log-text">${n(T.line)}</span>
        ${T.explain?`<div class="log-explain">${n(T.explain)}${m}</div>`:""}
      </div>
    `}async function j(){const T=u.filter(m=>m.severity==="error"||m.severity==="critical").map(m=>m.line).slice(-40);if(!(localStorage.getItem(yt)==="1")){E(T);return}await D(T)}function E(T){const p=T.length?`<pre class="explain-excerpt">${T.map(m=>n(m)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';O(`
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
    `,m=>{m==="proceed"?(localStorage.setItem(yt,"1"),L(),D(T)):L()})}async function D(T){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const p=T.length?await ht(r,T):await ht(r);if(i)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(p.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${p.sentExcerpt.map(m=>n(m)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,m=>{m==="close"&&L()})}catch(p){if(i)return;if(p instanceof we&&p.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,m=>{m==="close"&&L()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(p instanceof Error?p.message:String(p))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,m=>{m==="close"&&L()})}}function O(T,p){L();const m=document.createElement("div");m.className="modal-overlay",m.id="explain-modal",m.innerHTML=`<div class="modal">${T}</div>`,m.addEventListener("click",x=>{const B=x.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&p(B.dataset.modalAction),x.target===m&&p("cancel")}),document.body.appendChild(m)}function L(){var T;(T=document.getElementById("explain-modal"))==null||T.remove()}return()=>{i=!0,t==null||t(),L()}}const la="run",da={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},ua={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function pa(a,r){let i=!1,t=null,u=null;const y={devnet:null},S={devnet:null},f={devnet:[]};let A=null;const M={devnet:!1};let j=null;const E={devnet:null},D={devnet:null};a.innerHTML=`
    <div class="page-head">
      <h1>Services: ${n(r)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${de()}
  `;const O=a.querySelector("#services-body");ye(a,(l,g)=>{ge(l,g)}),L();async function L(){try{const l=await Hn(r);if(i)return;t=l,u=null}catch(l){if(i)return;t=null,u=I(l)}p()}function T(l){return t==null?void 0:t.services.find(g=>g.id===l)}function p(){if(!i){if(u){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(u)}</p>`;return}if(!t){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${m(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(x).join("")}
      </div>
    `}}function m(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${n(l.flavor)}${l.serverVersion?` ${n(l.serverVersion)}`:""} · reachable</p>`;const g=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(g)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${n(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function x(l){const g=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(l.label)}</h2>
          ${B(l)}
        </div>
        <p class="muted small">${n(da[l.id]??"")}</p>

        ${l.error?F(l):""}
        ${l.blocked?`<div class="banner banner-warn">${n(l.blocked)}</div>`:""}
        ${g.map(N=>`<div class="banner banner-warn">${n(N)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${n(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${X(l)}

        ${te(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(N=>W(l,N)).join("")}
        </div>
        ${S[l.id]?`<p class="error small">${n(S[l.id])}</p>`:""}
        ${ae(l)}

        ${ue(l)}
      </div>
    `}function B(l){switch(l.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function F(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(l.error??"")}</div>
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function X(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const g=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${g}.</p>`}function te(l){const g=l.endpoints??[];return g.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":g.map(N=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(N.label)}</span>
          <code class="endpoint-url">${n(N.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(N.url)}">Copy</button>
        </div>`).join("")}function W(l,g){const N=ua[g];if(!N)return"";const V=y[l.id],Q=g==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:N.label;return`
      <button class="${N.className}" data-action="svc-${g}" data-svc="${n(l.id)}"
              title="${n(N.title)}" ${V?"disabled":""}>
        ${V===g?'<span class="spinner" aria-label="working"></span>':n(Q)}
      </button>
    `}function ae(l){const g=f[l.id]??[];return g.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(g.join(`
`))}</pre>
      </div>
    `}function ue(l){const g=M[l.id],N=J(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(l.id)}">
            ${g?"Close":"Edit"}
          </button>
        </div>
        ${g?se():`<p class="small">${N}</p>`}
        ${E[l.id]?`<p class="error small">${n(E[l.id])}</p>`:""}
        ${D[l.id]?`<p class="muted small">${n(D[l.id])}</p>`:""}
      </div>
    `}function J(l){const g=l.devnet;return g?`Chain ${g.ChainID} · a block every ${n(g.BlockTime)} · JSON-RPC on ${n(g.BindAddr)}:${g.HTTPPort} · WebSocket on ${n(g.BindAddr)}:${g.WSPort}`:"—"}function se(l){return oe()}function oe(){const l=j;return l?`
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
    `:""}function le(){M.devnet&&j&&(j.BlockTime=Y("#dev-blocktime",j.BlockTime),j.HTTPPort=me("#dev-http",j.HTTPPort),j.WSPort=me("#dev-ws",j.WSPort),j.BindAddr=Y("#dev-bind",j.BindAddr))}function Y(l,g){const N=a.querySelector(l);return N?N.value.trim():g}function me(l,g){const N=a.querySelector(l);if(!N)return g;const V=Number.parseInt(N.value.trim(),10);return Number.isFinite(V)?V:g}async function ge(l,g){const N=g.dataset.svc??"";switch(l){case"refresh":await L();return;case"copy":g.dataset.copy&&await $(g,g.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await v(N,l.slice(4));return;case"svc-create":case"svc-recreate":await d(N);return;case"svc-wipe":H(N);return;case"toggle-config":C(N);return;case"save-config":await R(N);return;default:return}}async function v(l,g){if(!y[l]){y[l]=g,S[l]=null,p();try{await Dn(r,l,g)}catch(N){S[l]=`${g} failed: ${I(N)}${_(N)}`}y[l]=null,await L()}}async function d(l){if(!y[l]){y[l]="create",S[l]=null,f[l]=["starting…"],p();try{await Un(r,l)}catch(g){S[l]=`${I(g)}${_(g)}`,f[l]=[],y[l]=null,p();return}A==null||A(),A=Ge(r,g=>{if(i)return;const N=g.err?`${g.stepId}: ${g.err}`:g.line?`${g.stepId}: ${g.line}`:`${g.stepId}: done`;if(f[l]=[...(f[l]??[]).filter(Q=>Q!=="starting…"),N],!!g.err||g.stepId===la&&!!g.done){A==null||A(),A=null,y[l]=null,g.err&&(S[l]="Provisioning failed — see the log below."),L();return}p()})}}function C(l){if(le(),M[l]=!M[l],E[l]=null,D[l]=null,M[l]){const g=T(l);g!=null&&g.devnet&&(j={...g.devnet})}p()}async function R(l){var V;le(),E[l]=null,D[l]=null;const g=j;if(!g)return;if(g.HTTPPort===g.WSPort){E[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",p();return}try{await qn(r,l,g)}catch(Q){E[l]=I(Q),p();return}const N=((V=T(l))==null?void 0:V.status.State)==="running";M[l]=!1,D[l]=N?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function H(l){const g=T(l);if(!g)return;const N=(g.restartsOnWipe??[]).map(q=>{var re;return((re=T(q))==null?void 0:re.label)??q});ie(`
        <h2>Wipe ${n(g.label)}</h2>
        <p class="error">This deletes ${n(g.wipeDiscards)}</p>
        ${N.length?`<p>It also restarts what sits in front of it: ${n(N.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(l)}</button>
        </div>
      `,q=>{if(q==="cancel"||q==="close"){Z(),L();return}q==="confirm"&&K(l)});const V=document.getElementById("wipe-confirm-input"),Q=document.getElementById("wipe-confirm-btn");V==null||V.addEventListener("input",()=>{Q&&(Q.disabled=V.value.trim()!==l)}),V==null||V.focus()}async function K(l){const g=document.getElementById("wipe-confirm-btn");g&&(g.disabled=!0,g.textContent="Wiping…");let N;try{N=await Mn(r,l)}catch(V){const Q=He();if(Q){const q=document.createElement("p");q.className="error small",q.textContent=`Wipe failed: ${I(V)}${_(V)}`,Q.appendChild(q)}g&&(g.disabled=!1,g.textContent=`Wipe ${l}`);return}b(l,N)}function b(l,g){const N=T(l),V=ne=>{var Pe;return((Pe=T(ne))==null?void 0:Pe.label)??ne},Q=[];Q.push(g.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ne of g.report.VolumesRemoved??[])Q.push(`Volume ${ne} deleted.`);for(const ne of g.report.VolumesAbsent??[])Q.push(`Volume ${ne} was already gone.`);g.report.Recreated&&Q.push("Container re-created from your saved configuration.");const q=(g.report.Cascaded??[]).map(V),re=(g.report.CascadeSkipped??[]).map(V);ie(`
        <h2>${n((N==null?void 0:N.label)??l)} wiped</h2>
        <ul class="plain-list">${Q.map(ne=>`<li>${n(ne)}</li>`).join("")}</ul>
        ${q.length?`<p class="ok">Restarted in front of it: ${n(q.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${re.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(re.join(", "))}.</p>`:""}
        ${g.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(g.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ne=>{(ne==="close"||ne==="cancel")&&(Z(),L())})}async function $(l,g){const N=await De(g),V=l.textContent;l.textContent=N?"Copied!":"Copy failed",setTimeout(()=>{i||(l.textContent=V)},1500)}function I(l){return l instanceof Error?l.message:String(l)}function _(l){return l instanceof we&&l.hint?` — ${l.hint}`:""}return()=>{i=!0,A==null||A(),Z()}}const Qe=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Fe=8545,We=5052,_e=30303,ha=[369,943,1],gt={369:"default",943:"practise here first"};function fa(a,r){let i=!1;const t={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};a.innerHTML=`<h1>Setup: ${n(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${de()}</div>`;const u=a.querySelector("#wizard-body"),y=a.querySelector("#wizard-footer");ye(a,(b,$)=>{me(b,$)}),at(a,(b,$)=>{b==="exec-select"?t.execId=$:b==="beacon-select"&&(t.beaconId=$),f()}),a.addEventListener("change",b=>{const $=b.target;$ instanceof HTMLInputElement&&($.id==="data-dir-input"?(ge(),W()):$.id==="checkpoint-toggle"?(t.checkpoint=$.checked,f()):$.id==="exec-snapshot-toggle"&&(t.execSnapshot=$.checked,f()))}),S();async function S(){try{const[b,$]=await Promise.all([Ce(),xe()]);if(i)return;t.catalog=b;const I=$.find(_=>_.id===r);I!=null&&I.wire&&(t.chainId=I.wire.ChainID,t.execId=I.wire.ExecID,t.beaconId=I.wire.BeaconID,t.archive=I.wire.Archive,I.wire.ExecHTTPPort&&(t.execHTTPPort=String(I.wire.ExecHTTPPort)),I.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(I.wire.BeaconHTTPPort)),I.wire.ExecP2PPort&&(t.execP2PPort=String(I.wire.ExecP2PPort)),I.wire.RPCBindAddr&&(t.rpcBindAddr=I.wire.RPCBindAddr)),f()}catch(b){if(i)return;t.loadError=String(b instanceof Error?b.message:b),f()}}function f(){if(t.loadError){u.innerHTML=`<p class="error">Failed to load: ${n(t.loadError)}</p>`;return}t.catalog&&(u.innerHTML=`
      ${K(t.step)}
      ${M()}
    `,A())}function A(){var $;const b=($=t.catalog)==null?void 0:$.networks.find(I=>I.ChainID===t.chainId);y.innerHTML=b?de(b.Name,b.LearnURL):de()}function M(){switch(t.step){case"network":return j();case"clients":return E();case"mode":return oe();case"review":return le();case"run":return Y()}}function j(){const b=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${ha.map(I=>{const _=b.networks.find(N=>N.ChainID===I);if(!_)return"";const l=t.chainId===I,g=gt[I]?U(gt[I],I===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${I}" type="button">
          <h3>${n(_.Name)} <span class="muted">(chain ${I})</span></h3>
          ${g}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function E(){const b=t.catalog,$=b.networks.find(l=>l.ChainID===t.chainId);if(!$)return'<p class="error">Unknown network.</p>';(t.execId===null||!$.ExecClients.includes(t.execId))&&(t.execId=$.ExecClients[0]??null),(t.beaconId===null||!$.BeaconClients.includes(t.beaconId))&&(t.beaconId=$.BeaconClients[0]??null);const I=$.ExecClients.map(l=>ue(l,b)),_=$.BeaconClients.map(l=>ue(l,b));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${n($.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${et("exec-select",I,t.execId)}
        </label>
        ${se(t.execId,b)}
        <label>
          Beacon client
          ${et("beacon-select",_,t.beaconId)}
        </label>
        ${se(t.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function D(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const O=1.1,L=.5,T="Valve reth snapshot",p="rough estimate";function m(b){return b.SnapshotSizeTB}function x(b){return b.SnapshotSizeTB*L}function B(b){return`<p class="muted small">${D(m(b))} is the measured size of Valve's reth snapshot for ${n(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function F(b){return{archive:m(b)*1e12*O,full:x(b)*1e12*O}}function X(b,$){if(!b)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${n($)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${n($)}</code>: ${n(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==$)return"";const I=F(b),_=t.freeBytes>=I.archive,l=t.freeBytes>=I.full,g=`<p class="muted small">Free at <code>${n($)}</code>: <strong>${Se(t.freeBytes)}</strong> — archive ${_?"fits":"won't fit"} (${D(m(b))}, ${T}), full ${l?"fits":"won't fit"} (${D(x(b))}, ${p}).</p>`;let N="";return t.downgradeNote?N=`<p class="banner banner-warn">${n(t.downgradeNote)}</p>`:l||(N=`<p class="banner banner-warn">Neither full (${D(x(b))}, ${p}) nor archive (${D(m(b))}, ${T}) fits the free space here — choose a location with more room.</p>`),g+N}function te(b,$){if(t.downgradeNote=null,!b||t.freeBytes===null)return;const I=F(b);t.archive&&t.freeBytes<I.archive&&t.freeBytes>=I.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${$} for archive (${D(m(b))}, ${T}) — switched to Full (${D(x(b))}, ${p}). Pick a location with more room to run archive.`)}async function W(){var I;if(t.chainId===null)return;const b=(I=t.catalog)==null?void 0:I.networks.find(_=>_.ChainID===t.chainId),$=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,f();try{const{freeBytes:_}=await Tn(t.targetId,$);if(i)return;t.freeBytes=_,t.probedPath=$,te(b,$)}catch(_){if(i)return;t.freeBytes=null,t.probedPath=$,t.diskError=String(_ instanceof Error?_.message:_)}t.diskProbing=!1,f()}function ae(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ue(b,$){const I=$.clients.find(_=>_.id===b);return{value:b,label:I?`${I.id} — ${J(I.repo)}`:b}}function J(b){const $=b.split("/");return $.length>=4?$[3]:b}function se(b,$){const I=b?$.clients.find(l=>l.id===b):void 0;if(!I)return"";const _=I.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(I.repo)}" target="_blank" rel="noopener noreferrer">${n(_)}</a></p>`}function oe(){var V,Q,q;const b=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",$=(V=t.catalog)==null?void 0:V.networks.find(re=>re.ChainID===t.chainId),I=((q=(Q=t.catalog)==null?void 0:Q.clients.find(re=>re.id===t.execId))==null?void 0:q.snapshotSupported)??!1,_=$?`${D(x($))} (${p})`:"Smaller",l=$?`${D(m($))} (${T})`:"Much larger",g=$?` on ${n($.Name)}`:"",N=$?t.checkpoint?$.SyncLabel:$.GenesisSyncLabel:"";return`
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
          ${$?`<p class="sync-estimate">⏱ Estimated initial sync${g}: <strong>${n(N)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n(($==null?void 0:$.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n(($==null?void 0:$.CheckpointURL)??"")}" value="${n(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${n(t.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${I?`
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
          ${$?B($):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${$?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${_}${$?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(b)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(b)}" value="${n(t.dataDir)}" />
          </label>
          ${X($,t.dataDir||b)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(b)}/jwt.hex" value="${n(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Fe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Fe}" value="${n(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${n(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${We})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${We}" value="${n(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${n(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${_e})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${_e}" value="${n(t.execP2PPort)}" />
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
    `}function le(){const $=t.catalog.networks.find(ne=>ne.ChainID===t.chainId),I=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,_=t.jwtPath||`${I}/jwt.hex`,l=Qe.map(ne=>`<li>${n(ne.title)}</li>`).join(""),g=R(t.execHTTPPort,Fe),N=R(t.beaconHTTPPort,We),V=R(t.execP2PPort,_e),Q=g||N||V?`<tr><th>Non-default ports</th><td>${[g?`exec HTTP ${g}`:null,N?`beacon HTTP ${N}`:null,V?`exec p2p ${V}`:null].filter(ne=>ne!==null).map(n).join(", ")}</td></tr>`:"",{addr:q}=v(t.rpcBindAddr),re=q?`<tr><th>RPC bind address</th><td><code>${n(q)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${n(($==null?void 0:$.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(I)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(_)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${n(t.checkpointUrl||($==null?void 0:$.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
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
    `}function Y(){const $=t.catalog.networks.find(q=>q.ChainID===t.chainId),I=$==null?void 0:$.LearnURL,_=new Set(t.events.filter(q=>q.done).map(q=>q.stepId)),l=new Set(t.events.filter(q=>q.err).map(q=>q.stepId)),g=new Map;for(const q of t.events){if(!q.line)continue;const re=g.get(q.stepId)??[];re.push(q.line),g.set(q.stepId,re)}const N=Qe.map(q=>{var qe;const re=_.has(q.id),ne=l.has(q.id),Pe=ne?U("failed","bad"):re?U("done","ok"):U("pending","neutral"),Ue=(g.get(q.id)??[]).slice(-5),Oe=(qe=t.events.find(Ee=>Ee.stepId===q.id&&Ee.err))==null?void 0:qe.err,Je=q.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${I?` <a href="${n(I)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${re?"step-done":""} ${ne?"step-error":""}">
          <div class="step-head">${Pe} <strong>${n(q.title)}</strong></div>
          ${Je}
          ${Ue.length?`<pre class="step-log">${Ue.map(Ee=>n(Ee)).join(`
`)}</pre>`:""}
          ${Oe?`<p class="error small">${n(Oe)}</p>`:""}
        </li>
      `}).join(""),V=t.events.some(q=>q.err),Q=Qe.every(q=>_.has(q.id))||t.events.some(q=>q.stepId==="handshake"&&q.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${Q&&!V?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        ${V?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function me(b,$){switch(b){case"pick-network":t.chainId=Number($.dataset.chainId),t.execId=null,t.beaconId=null,f();break;case"goto-network":t.step="network",f();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",f();break;case"goto-mode":t.step="mode",f(),W();break;case"goto-review":if(ge(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){f();break}t.step="review",f();break;case"start-setup":H();break}}function ge(){const b=a.querySelectorAll('input[name="mode"]');for(const q of Array.from(b))q.checked&&(t.archive=q.value==="archive");const $=a.querySelector("#data-dir-input"),I=a.querySelector("#jwt-path-input");$&&(t.dataDir=$.value.trim()),I&&(t.jwtPath=I.value.trim());const _=a.querySelector("#exec-http-port-input"),l=a.querySelector("#beacon-http-port-input"),g=a.querySelector("#exec-p2p-port-input");_&&(t.execHTTPPort=_.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),g&&(t.execP2PPort=g.value.trim());const N=a.querySelector("#rpc-bind-addr-input");N&&(t.rpcBindAddr=N.value.trim());const V=a.querySelector("#checkpoint-url-input");V&&(t.checkpointUrl=V.value.trim());const Q=a.querySelector("#snapshot-key-input");Q&&(t.snapshotKey=Q.value.trim()),t.execHTTPPortError=C(t.execHTTPPort).error??null,t.beaconHTTPPortError=C(t.beaconHTTPPort).error??null,t.execP2PPortError=C(t.execP2PPort).error??null,t.rpcBindAddrError=v(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?ae(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function v(b){if(!b)return{};const $=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return $?$.slice(1).every(I=>Number(I)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function C(b){if(!b)return{};if(!d.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const $=Number(b);return!Number.isInteger($)||$<1||$>65535?{error:"Port must be between 1 and 65535."}:{port:$}}function R(b,$){const{port:I}=C(b);if(!(I===void 0||I===$))return I}async function H(){var g;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(g=t.streamStop)==null||g.call(t),t.streamStop=null,f();const b={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(b.DataDir=t.dataDir),t.jwtPath&&(b.JWTPath=t.jwtPath);const $=R(t.execHTTPPort,Fe),I=R(t.beaconHTTPPort,We),_=R(t.execP2PPort,_e);$!==void 0&&(b.ExecHTTPPort=$),I!==void 0&&(b.BeaconHTTPPort=I),_!==void 0&&(b.ExecP2PPort=_);const{addr:l}=v(t.rpcBindAddr);l!==void 0&&(b.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(b.CheckpointURL=t.checkpointUrl):b.NoCheckpoint=!0,t.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=t.snapshotKey);try{await Sn(t.targetId,b)}catch(N){if(!(N instanceof we&&N.status===409)){t.starting=!1,t.startError=String(N instanceof Error?N.message:N),f();return}}t.starting=!1,t.step="run",f(),t.streamStop=Ge(t.targetId,N=>{i||(t.events.push(N),t.step==="run"&&f())})}function K(b){const $=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],_=$.map(l=>l.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${$.map((l,g)=>`<li class="${g===_?"current":g<_?"past":"future"}">${n(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;i=!0,(b=t.streamStop)==null||b.call(t)}}function ma(a,r){let i=!1;const t=new Map;a.innerHTML=`<h1>${n(r)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const u=a.querySelector("#machine-body");ye(a,(E,D)=>{E==="toggle-section"&&M(D.dataset.section??"")}),y();async function y(){let E,D;try{const[O,L]=await Promise.all([xe(),Ce()]);E=O.find(T=>T.id===r),D=L}catch(O){if(i)return;u.innerHTML=`<p class="error">Failed to load machine: ${n(String(O))}</p>`;return}if(!i){if(!E){location.hash="#/targets";return}S(E,D)}}function S(E,D){const O=E.mode==="local"?"this machine":"SSH",L=E.mode==="ssh"&&E.ssh?`${n(E.ssh.User)}@${n(E.ssh.Host)}`:O;u.innerHTML=`
      <p class="muted">${L}</p>
      <p>${f(E,D)}</p>
      <div class="machine-sections">
        ${j.map(T=>A(T,E,D)).join("")}
      </div>
      ${de()}
    `}function f(E,D){const O=E.wire;if(!O)return U("not set up","neutral");const L=D.networks.find(p=>p.ChainID===O.ChainID),T=L?L.Name:`chain ${O.ChainID}`;return`${U(T,"ok")} ${U(O.ExecID,"neutral")} ${U(O.BeaconID,"neutral")}${O.Archive?" "+U("archive","warn"):""}`}function A(E,D,O){return`
      <section class="card machine-section" data-section-card="${n(E.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(E.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(E.title)}</span>
          <span class="machine-section-status">${E.status(D,O)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(E.key)}" hidden></div>
      </section>
    `}function M(E){const D=j.find(m=>m.key===E);if(!D)return;const O=a.querySelector(`[data-section-card="${E}"]`),L=a.querySelector(`[data-section-body="${E}"]`),T=a.querySelector(`.machine-section-head[data-section="${E}"]`);if(!O||!L||!T)return;const p=L.hidden;if(p&&!t.has(E)){const m=document.createElement("div");L.appendChild(m),t.set(E,D.mount(m))}L.hidden=!p,O.classList.toggle("open",p),T.setAttribute("aria-expanded",String(p))}const j=[{key:"setup",title:"Setup",status:E=>E.wire?U("set up","ok"):U("not set up","neutral"),mount:E=>fa(E,r)},{key:"dashboard",title:"Dashboard",status:E=>E.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>ia(E,r)},{key:"logs",title:"Logs",status:E=>E.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>ca(E,r)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:E=>pa(E,r)}];return()=>{i=!0;for(const E of t.values())try{E()}catch{}t.clear()}}function Ct(a){var t;if(!a)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const r=a.actions??[];if(a.blocked)return{tone:"blocked",label:"Unavailable",sub:a.blocked,actions:r,blocked:a.blocked};const i=((t=a.networks)==null?void 0:t.length)??0;return a.status.State==="running"?{tone:"on",label:"Running",sub:`${i} network${i===1?"":"s"} served`,actions:r}:{tone:"off",label:"Stopped",sub:i?`${i} network${i===1?"":"s"} configured`:"Press to start",actions:r}}function ba(a){if(!a.running)return"off";if(!a.serviceable)return"frequent";const r=a.slowRate??0;return r>.4?"frequent":r>=.1?"occasional":"stable"}const ya=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function ga(a){return ya.map(({key:r,label:i,hot:t})=>{const u=a[r]==="supported";return{key:r,label:i,lit:u,hot:!!t&&u}})}const va=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
  <symbol id="p-power" viewBox="0 0 24 24"><line x1="12" y1="3.5" x2="12" y2="11.5"/><path d="M7.5 7a7 7 0 1 0 9 0"/></symbol>
  <symbol id="p-globe" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.5 2.4 2.5 14.6 0 17M12 3.5c-2.5 2.4-2.5 14.6 0 17"/></symbol>
  <symbol id="p-ws" viewBox="0 0 24 24"><path d="M4 9h13l-3.5-3.5M20 15H7l3.5 3.5"/></symbol>
  <symbol id="p-archive" viewBox="0 0 24 24"><path d="M12 3 3 7.5l9 4.5 9-4.5L12 3ZM3 12l9 4.5 9-4.5M3 16.5 12 21l9-4.5"/></symbol>
  <symbol id="p-trace" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5.5"/><path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3"/></symbol>
  <symbol id="p-lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9.5" rx="2.2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></symbol>
  <symbol id="p-pencil" viewBox="0 0 24 24"><path d="M14 5.5l4.5 4.5M4 20l1.2-4.4L16 4.8a2 2 0 0 1 2.8 0l.4.4a2 2 0 0 1 0 2.8L8.4 18.8 4 20Z"/></symbol>
  <symbol id="p-trash" viewBox="0 0 24 24"><path d="M4 6.5h16M9.5 6.5V5a2 2 0 0 1 2-2h1a2 2 0 0 1 2 2v1.5M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6"/></symbol>
  <symbol id="p-copy" viewBox="0 0 24 24"><rect x="9" y="9" width="11" height="11" rx="2.2"/><path d="M5 15V5a2 2 0 0 1 2-2h8"/></symbol>
  <symbol id="p-scale" viewBox="0 0 24 24"><path d="M12 3v18M7 21h10M12 5 5 8m7-3 7 3M5 8l-3 6a3 3 0 0 0 6 0L5 8Zm14 0-3 6a3 3 0 0 0 6 0l-3-6Z"/></symbol>
  <symbol id="p-refresh" viewBox="0 0 24 24"><path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"/></symbol>
  <symbol id="p-chevR" viewBox="0 0 24 24"><path d="M9.5 5.5l6.5 6.5-6.5 6.5"/></symbol>
  <symbol id="p-chevL" viewBox="0 0 24 24"><path d="M14.5 5.5 8 12l6.5 6.5"/></symbol>
  <symbol id="p-plus" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></symbol>
</defs></svg>`,Me=a=>`<svg class="p-i"><use href="#p-${a}"/></svg>`,$a="run";function vt(a){let r=null,i={name:"list"},t=null,u=null,y=null,S=null;a.innerHTML=va+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const f=a.querySelector("#p-card");async function A(){try{const T=await nt();r=wa(T.gateways),t=null}catch(T){t=Ke(T)}M()}function M(){f.innerHTML=j()}function j(){return t?ka(t):i.name==="network"?La(r,i.chainId):i.name==="endpoint"?Aa(r,i.chainId,i.upstreamId):Ta(r,u,y)}ye(f,(T,p)=>{E(T,p)});async function E(T,p){if(T==="power"){if(!r||u)return;const m=Ct(r);if(m.tone==="blocked")return;if(r.status.State==="running"&&m.actions.includes("stop")){await D(r.id,"stop");return}if(m.actions.includes("start")){await D(r.id,"start");return}if(m.actions.includes("create")){await O(r.id);return}return}if(T==="open-network"){i={name:"network",chainId:Number(p.dataset.chainId)},M();return}if(T==="back-to-list"){i={name:"list"},M();return}if(T!=="add-network")switch(T){case"gw-start":case"gw-stop":case"gw-restart":r&&!u&&await D(r.id,T.slice(3));return;case"gw-create":case"gw-recreate":r&&!u&&await O(r.id);return;case"gw-wipe":r&&!u&&await L(r);return;default:return}}async function D(T,p){if(!u){u=p,y=null,M();try{await kt(T,p)}catch(m){y=`${p} failed: ${Ke(m)}`}u=null,await A()}}async function O(T){if(u)return;u="create",y=null,M();let p;try{p=await Tt(T)}catch(m){y=Ke(m),u=null,M();return}S==null||S(),S=Ge(p.targetId,m=>{(m.err||m.stepId===$a&&m.done)&&(S==null||S(),S=null,u=null,m.err&&(y=`Provisioning failed: ${m.err}`),A())})}async function L(T){if(await Ae({title:`Wipe ${T.label}`,body:`This destroys ${T.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){u="wipe",y=null,M();try{const m=await St(T.id);m.error&&(y=m.error)}catch(m){y=`wipe failed: ${Ke(m)}`}u=null,await A()}}return A(),()=>{S==null||S()}}function wa(a){return!a||a.length===0?null:a.find(r=>r.placement.targetId==="local")??a[0]}function Ke(a){return a instanceof Error?a.message:String(a)}function ka(a){return`<div class="p-band" style="padding:16px;color:var(--red)">${n(a)}</div>`}function Ta(a,r,i){var y;const t=Ct(a),u=(y=a==null?void 0:a.networks)!=null&&y.length?a.networks.map((S,f)=>Ra(a,S,f>0)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${n(t.sub)}</span>
    </div>
    <div class="p-band">
      ${xa(a,t,r,i)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${u}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${Me("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function Sa(a,r){return r.tone==="blocked"?null:a.status.State==="running"&&r.actions.includes("stop")?"stop":r.actions.includes("start")?"start":r.actions.includes("create")?"create":null}const Ca={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},$t={restart:"refresh",recreate:"refresh",wipe:"trash"};function xa(a,r,i,t){const u=r.tone==="blocked"?r.blocked??"":r.sub,y=i?" busy":"",S=t?`<div class="p-ps" style="color:var(--red)">${n(t)}</div>`:"",f=`
    <div class="p-power${y}" data-action="power">
      <div class="p-pbtn ${r.tone}">${Me("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${n(r.label)}</div>
        <div class="p-ps"${r.tone==="blocked"?' style="color:var(--red)"':""}>${n(u)}</div>
        ${S}
      </div>
    </div>
  `,A=a?Pa(a,r,i):"";return f+A}function Pa(a,r,i){const t=Sa(a,r),u=(a.actions??[]).filter(S=>S!==t);return u.length===0?"":`<div class="p-chips">${u.map(S=>{const f=Ca[S]??S,A=$t[S]?Me($t[S]):"";return`<button type="button" class="p-chip${S==="wipe"?" danger":""}" data-action="gw-${S}" data-gid="${n(a.id)}"${i?" disabled":""}>${A}${n(f)}</button>`}).join("")}</div>`}const Ea={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function Ia(a){return a.map(r=>`<svg class="p-i${r.hot?" hot":r.lit?" on":""}"><use href="#p-${Ea[r.key]}"/></svg>`).join("")}function Ra(a,r,i){const t=ba({running:a.status.State==="running",serviceable:r.serviceable}),u=ga({});return`
    <div class="p-row${i?" p-rowdiv":""}" data-action="open-network" data-chain-id="${r.chainId}">
      <span class="p-lead"><span class="p-dot ${t}"></span></span>
      <span class="p-nm">${n(r.name)}</span>
      <span class="p-caps">${Ia(u)}</span>
      <span class="p-chev">${Me("chevR")}</span>
    </div>
  `}function La(a,r){var t;const i=(t=a==null?void 0:a.networks)==null?void 0:t.find(u=>u.chainId===r);return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-list">${Me("chevL")}</span>
      <span class="p-dtitle"><span class="p-nmtxt">${n((i==null?void 0:i.name)??`Chain ${r}`)}</span></span>
    </div>
    <div class="p-band" style="padding:16px;color:var(--dim)">Network detail is coming soon.</div>
  `}function Aa(a,r,i){return""}function Na(a,r){let i=!1,t=[],u=null,y=!1,S=!1;a.innerHTML=`<h1>Security: ${n(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${de()}</div>`;const f=a.querySelector("#sec-body"),A=a.querySelector("#sec-footer");ye(a,(L,T)=>{var p;if(L==="rerun")j();else if(L==="toggle")(p=T.closest(".check-item"))==null||p.classList.toggle("expanded");else if(L==="copy"){const m=T.dataset.copy;m&&O(T,m)}}),M();async function M(){let L,T;try{const[m,x]=await Promise.all([xe(),Ce()]);L=m.find(B=>B.id===r),T=x}catch(m){if(i)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(m))}</p>`;return}if(i)return;if(!L){f.innerHTML=`<p class="error">Target "${n(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const p=T==null?void 0:T.networks.find(m=>m.ChainID===L.wire.ChainID);p&&(A.innerHTML=de(p.Name,p.LearnURL)),await j()}async function j(){y=!0,u=null,E();try{t=await An(r),S=!0}catch(L){u=String(L instanceof Error?L.message:L)}y=!1,i||E()}function E(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${y?"disabled":""}>${y?"Re-running…":"Re-run checks"}</button>
      </div>
      ${u?`<p class="error">${n(u)}</p>`:""}
      ${!S&&y?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(D).join("")}</ul>`:S?'<p class="muted">No checks returned.</p>':""}
    `}function D(L){const T=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${U(L.Status,T)}
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
    `}async function O(L,T){const p=await De(T),m=L.textContent;L.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{i||(L.textContent=m)},1500)}return()=>{i=!0}}const Ba=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],tt="VALVE_API_KEY";function Ha(a){return a===tt?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(a)}}</code> slot wherever an endpoint URL carries one.`}function Da(a){let r=!1,i=!1,t=!1,u=null,y=!1,S=null,f=null;const A=new Set,M=new Map;let j="",E="";a.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${de()}`;const D=a.querySelector("#settings-body");ye(a,(x,B)=>{if(x==="save"&&m(),x==="clear-key"){if(!S)return;i=!0;const F=a.querySelector("#ai-key");F&&(F.value=""),p(S)}if(x==="clear-provider-key"){const F=B.dataset.key;if(!S||!F)return;A.add(F),M.set(F,""),y=!1,p(S)}}),at(a,(x,B)=>{x!=="ai-provider"||!S||(f=B,y=!1,p(S))}),O();async function O(){try{const x=await Xn();if(r)return;S=x,p(x)}catch(x){if(r)return;D.innerHTML=`<p class="error">Failed to load settings: ${n(String(x))}</p>`}}function L(x){const F=(Array.isArray(x.providerKeysSet)?x.providerKeysSet:[]).filter(X=>X!==tt).sort();return[tt,...F]}function T(x,B){const F=n(x);return`
      <div class="pk-row">
        <label>
          <code>${F}</code>
          <input class="provider-key" data-key="${F}" type="password" autocomplete="off"
                 placeholder="${B?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Ha(x)}</p>
        ${B?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${F}">Clear saved key</button>`:""}
      </div>`}function p(x){var ue;const B=f??x.aiProvider,F=Array.isArray(x.providerKeysSet)?x.providerKeysSet:[],X=L(x).map(J=>T(J,F.includes(J))).join("");D.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${et("ai-provider",Ba.map(J=>({value:J.value,label:J.label})),B)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${x.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${x.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${X}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${n(j)}" />
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
            <input id="ref-rpc-base" type="text" value="${n(x.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${u?`<p class="error">${n(u)}</p>`:""}
        ${y?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const te=a.querySelector("#ai-key");te==null||te.addEventListener("input",()=>{i=!0,y=!1}),(ue=a.querySelector("#ref-rpc-base"))==null||ue.addEventListener("input",()=>{y=!1}),a.querySelectorAll("input.provider-key").forEach(J=>{const se=J.dataset.key;if(!se)return;const oe=M.get(se);oe!==void 0&&(J.value=oe),J.addEventListener("input",()=>{A.add(se),M.set(se,J.value),y=!1})});const W=a.querySelector("#pk-new-value");W&&(W.value=E),W==null||W.addEventListener("input",()=>{E=W.value,y=!1});const ae=a.querySelector("#pk-new-name");ae==null||ae.addEventListener("input",()=>{j=ae.value,y=!1})}async function m(){const x=a.querySelector("#ai-key"),B=a.querySelector("#ref-rpc-base");if(!x||!B||!S)return;const F={aiProvider:f??S.aiProvider,refRpcBase:B.value.trim()};i&&(F.aiKey=x.value);const X={};for(const W of A)X[W]=M.get(W)??"";const te=j.trim();te&&(X[te]=E),Object.keys(X).length>0&&(F.providerKeys=X),t=!0,u=null,y=!1,p(S);try{const W=await Qn(F);if(r)return;S=W,i=!1,A.clear(),M.clear(),j="",E="",t=!1,y=!0,p(W)}catch(W){if(r)return;t=!1,u=String(W instanceof Error?W.message:W),p(S)}}return()=>{r=!0}}const Ma=["http","ws","archive","trace"],Ua={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Le=1337,Oa="run",qa={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function ja(a){let r=!1,i=null,t=null;const u={},y={},S={},f={},A={},M={},j={},E={},D={},O={},L={},T={},p={},m={},x={};let B="",F=null;a.innerHTML=`
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
    ${de()}
  `;const X=a.querySelector("#rpc-body");ye(a,(e,s)=>{Vt(e,s)}),at(a,()=>{}),W(),te();async function te(){try{const e=await wt();if(r)return;B=e.os,Y()}catch{}}async function W(){try{const e=await nt();if(r)return;i=e,t=null}catch(e){if(r)return;i=null,t=pe(e)}Y();for(const e of(i==null?void 0:i.gateways)??[])ae(e.id),ue(e.id,!1)}async function ae(e){try{const s=await _n(e);if(r)return;u[e]=s}catch{if(r)return;u[e]=null}Y()}async function ue(e,s){S[e]=s,s&&Y();try{const o=await Vn(e,s);if(r)return;y[e]=o}catch{if(r)return;y[e]=null}S[e]=!1,Y()}function J(e){return((i==null?void 0:i.gateways)??[]).find(s=>s.id===e)}function se(e,s){return(e.networks??[]).find(o=>o.chainId===s)}function oe(e,s,o){var h;const c=(((h=u[e])==null?void 0:h.networks)??[]).find(k=>k.chainId===s);return((c==null?void 0:c.upstreams)??[]).find(k=>k.upstream===o)}function le(e,s,o){var c;return(((c=y[e])==null?void 0:c.endpoints)??[]).find(h=>h.chainId===s&&h.upstream===o)}function Y(){if(r)return;if(t){X.innerHTML=`<p class="error">Could not read the gateways: ${n(t)}</p>`;return}if(!i){X.innerHTML='<p class="muted">Loading…</p>';return}const e=i.gateways??[],s=e.length>1,o=(i.targets??[]).some(k=>ut(k.id,e)),c=new Set(e.map(k=>k.placement.targetId)),h=(i.orphans??[]).filter(k=>!c.has(k.targetId));X.innerHTML=`
      ${e.map(k=>v(k,s)).join("")}
      ${e.length===0?ge():""}
      ${h.map(me).join("")}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function me(e){const s=`docker rm -f ${e.containerName}`,o=p[e.containerName];return`
      <div class="strip">
        ${N({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:s})}
        ${o?N({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function ge(){return((i==null?void 0:i.targets)??[]).length===0?`
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
    `}function v(e,s){return`
      ${s?`<h2 class="rpc-machine">${n(e.placement.targetId)}</h2>`:""}
      ${d(e)}
      ${g(e)}
      ${ne(e)}
      ${Pe(e)}
      ${b(e)}
    `}function d(e){const s=e.status.State==="running",o=e.tls,c=[`on <strong>${n(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${n(e.status.Image)}</code>`),c.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${q(e)}
        <strong>${n(e.label)}</strong>
        ${Q(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${s?`base <code>${n(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function C(e){const s=e.tls;return s!=null&&s.enabled&&s.rootCaPath&&s.effectiveCertSource==="internal"?s.rootCaPath:null}function R(e){var s;return((s=((i==null?void 0:i.targets)??[]).find(o=>o.id===e.placement.targetId))==null?void 0:s.mode)??""}function H(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function K(e,s,o){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${s}"`;case"windows":return`certutil -addstore -f ROOT "${s}"`;case"linux":default:return`sudo cp "${s}" /usr/local/share/ca-certificates/valve-node-app-${o}.crt && sudo update-ca-certificates`}}function b(e){const s=D[e.id]??!1,o=((i==null?void 0:i.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${s?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${n(e.id)}" aria-expanded="${s}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${$(e,o.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${s?I(e,o):""}
      </section>
    `}function $(e,s){const o=[];return e.status.State!=="running"&&o.push("gateway not running"),s>0&&o.push(`${s} leftover container${s===1?"":"s"}`),o.length===0?"container, settings, certificate":o.join(" · ")}function I(e,s){var o;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>re(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${S[e.id]?"disabled":""}>
            ${S[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(e.id)}">
            ${j[e.id]?"Close settings":"Settings"}
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
        ${s.map(me).join("")}
        ${j[e.id]?Ot(e):""}
      </div>
    `}function _(e){const s=C(e);if(!s)return"";const o=R(e)==="local",c=K(B,s,e.id),h=x[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${n(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${n(s)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(s)}">Copy path</button>
        </div>
        ${o?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${m[e.id]?"disabled":""}>
                   ${m[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${h?l(h):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${o?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${B?` (${n(H(B))})`:""}:</span>
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
    `}function g(e){const s=[];e.error&&s.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&s.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])s.push({tone:"warn",text:c});s.push(...V(e));const o=A[e.id];return o&&s.push({tone:"bad",text:o}),s.length===0?"":`<div class="strip">${s.map(N).join("")}</div>`}function N(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${n(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${n(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.cmd)}">Copy</button>`:""}
      </div>
    `}function V(e){var h,k;const s=e.tls;if(!(s!=null&&s.enabled))return[];const o=[];s.fallback&&o.push({tone:"warn",text:s.fallback}),s.error?o.push({tone:"warn",text:`HTTPS front: ${s.error}`}):((h=s.status)==null?void 0:h.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((k=s.status)==null?void 0:k.State)??"unknown"}, so nothing answers on ${s.url??"its https URL"} even if the gateway itself is up.`,cmd:s.containerName?`docker start ${s.containerName}`:void 0});const c=O[e.id]??s.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&o.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&o.push({tone:"warn",text:c.expiryWarning}),o}function Q(e){switch(e.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function q(e){return e.status.State==="running"?$e("ok"):e.status.State==="unknown"?$e("bad"):$e("neutral")}function re(e,s){const o=qa[s];if(!o)return"";const c=f[e.id];return`
      <button class="${o.className}" data-action="gw-${s}" data-gid="${n(e.id)}"
              title="${n(o.title)}" ${c?"disabled":""}>
        ${c===s?'<span class="spinner" aria-label="working"></span>':n(o.label)}
      </button>
    `}function ne(e){const s=M[e.id]??[];return s.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(e.placement.targetId)}</p>
        <pre class="step-log">${n(s.join(`
`))}</pre>
      </div>
    `}function Pe(e){const s=Ue(e.networks??[]),o=s.some(c=>c.chainId===Le);return s.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(e.id)}">Add a network</button>
            ${st(e,o)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${s.map(c=>Oe(e,c)).join("")}
        </div>
        ${Ee(e,o)}
        ${Ut(e)}
      </div>
    `}function Ue(e){const s=e.filter(c=>c.chainId!==Le),o=e.filter(c=>c.chainId===Le);return[...s,...o]}function Oe(e,s){const o=Et(s),c=s.chainId===Le,h=`${e.id}:${s.chainId}`,k=E[h]??!1,P=o.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${o.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${n(s.name)}</span>
          <code class="chain-key">evm:${s.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${U(P,o.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${n(h)}" aria-expanded="${k}">
              ${k?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Je(e,s)}
        ${k?qe(e,s,o):""}
      </section>
    `}function Je(e,s){if(!s.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const o=C(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${n(s.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${n(s.url)}"
                title="Copy ${n(s.url)}">Copy URL</button>
        ${o?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${R(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${m[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${m[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(o)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${n(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${x[e.id]?`<span class="chain-cert muted small">${n(x[e.id].ok?"Trusted — reload your wallet or browser.":x[e.id].message)}</span>`:""}`:""}
      </div>
    `}function qe(e,s,o){const c=s.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${o.why?" chain-verdict-why":""}"${o.why?` title="${n(o.why)}"`:""}>${o.html}</p>
        <div class="chain-detail-bar">
          ${Pt(c.length,o.tone,s.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${n(e.id)}" data-chain="${s.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${n(e.id)}" data-chain="${s.chainId}">Remove</button>
        </div>
        ${Lt(e,s)}
        ${(s.warnings??[]).map(h=>`<p class="chain-note">${n(h)}</p>`).join("")}
      </div>
    `}function Ee(e,s){const o=y[e.id],c=o!=null&&o.at?`probed ${n(rt(o.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${n(e.id)}">+ Network</button>
        ${st(e,s)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${n(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${S[e.id]?"disabled":""}>
          ${S[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function st(e,s){return s?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${n(e.id)}"
                    title="Add a throwaway local test chain (evm:${Le}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Pt(e,s,o){const c=o>0,h=c?o:e,k=Math.min(e,h);let P="";for(let Re=0;Re<h;Re++)P+=`<span class="seg${Re<k?` seg-on seg-${s}`:""}"></span>`;const w=c&&e>o,G=c?w?`${e} (set is ${o})`:`${e} of ${o}`:`${e}`,ee=`${e} upstream${e===1?"":"s"} configured`,he=c?`${ee}${w?`, ${e-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${ee}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(he)}">${P}</span>
      <span class="segs-n">${G}</span>
    `}function Et(e){const s=e.upstreams??[];if(s.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!s.some(It)){const c=Rt(s);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(k=>`<code>${n(k)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(s.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!s.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=s.filter(c=>!!c.problem);if(o.length>0){const c=s.length-o.length;return{tone:"warn",html:`${o.length} of these ${s.length} endpoints ${o.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${s.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function It(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Rt(e){const s=new Set;for(const o of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());c&&s.add(c[1].toLowerCase())}return[...s].sort()}function Lt(e,s){const o=s.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(c=>At(e,s,c)).join("")}</ul>`}function At(e,s,o){const c=`${e.id}|${s.chainId}|${o.id}`,h=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?$e("bad"):$e("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${Nt(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${Bt(e,s,o)}</div>
        <div class="up-share">${Mt(e,s,o)}</div>
        <div class="up-acts">
          ${h.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(c)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[e.id]?"disabled":""}>
                   ${f[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(c)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function Nt(e){return e.problem?U("unusable","bad"):e.recentOnly?U("recent blocks","warn"):e.local?U("yours","ok"):U("public","neutral")}function ot(e,s){var o;if(e)return s==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(c=>c.key===s))==null?void 0:o.status}function Bt(e,s,o){const c=le(e.id,s.chainId,o.id);return c?c.unprobeable?`<span class="caps-none" title="${n(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Ma.map(h=>Ht(e,s,c,h)).join("")}</span>`:`<span class="muted small">${y[e.id]===void 0?"probing…":"—"}</span>`}function Ht(e,s,o,c){const h=(o.capabilities??[]).find(ee=>ee.key===c),k=ot(o,c)??"inconclusive",P=Ua[c]??c.toUpperCase();let w="cap";k==="unsupported"?w=Dt(e,s,c)?"cap missing":"cap off":k==="inconclusive"?w="cap unknown":k==="inconsistent"&&(w="cap mixed");const G=h!=null&&h.detail?`${h.label}: ${h.detail}`:c==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${P}: no verdict`;return`<span class="${w}" title="${n(G)}">${n(P)}</span>`}function Dt(e,s,o){const c=(s.upstreams??[]).map(h=>le(e.id,s.chainId,h.id)).filter(h=>!!h&&!h.unprobeable);return c.length>0&&c.every(h=>ot(h,o)==="unsupported")}function Mt(e,s,o){const c=u[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const h=oe(e.id,s.chainId,o.id),k=(c.networks??[]).find(he=>he.chainId===s.chainId);if(!h||!k||k.attributed===0)return'<span class="muted small">no traffic yet</span>';const P=Math.round(h.actual*100),w=Math.round(h.intended*100),G=h.diverged?o.local?"warn":"":"ok",ee=`${h.succeeded.toLocaleString()} of ${k.attributed.toLocaleString()} answered requests · routing intends ${w}%`+(h.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ee)}">
        <span class="bar">
          <span class="fill${G?" "+G:""}" style="width:${P}%"></span>
          <span class="tick" style="left:${w}%"></span>
        </span>
        <span class="share-n${h.diverged?" warn":""}">${P}%</span>
        ${h.unconfigured?U("not in config","warn"):""}
      </span>
    `}function Ut(e){const s=u[e.id];return s?s.enabled?s.error?`<p class="muted small">The request counters could not be read: ${n(s.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${s.since?` (${n(rt(s.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function rt(e){const s=new Date(e);return Number.isNaN(s.getTime())?e:s.toLocaleString()}function Ot(e){const s=e.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${n(e.id)}-port" value="${s.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${n(e.id)}-bind" value="${n(s.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${n(s.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${qt(e)}
        ${jt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(e.id)}">Save settings</button>
        </div>
      </div>
    `}function qt(e){const s=!e.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${n(e.id)}-metrics" ${s?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function jt(e){var P;const s=n(e.id),o=e.config.TLS??null,c=(o==null?void 0:o.Enabled)??!1,h=(o==null?void 0:o.CertSource)||"internal",k=((P=e.tls)==null?void 0:P.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${s}-tls" ${c?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${s}-tls-host" value="${n((o==null?void 0:o.Hostname)??k)}"
               placeholder="${n(k||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${k?`<p class="muted small">
               The default is <code>${n(k)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${s}-tls-port" value="${(o==null?void 0:o.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${s}-tls-source">
          <option value="internal" ${h==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${h==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${s}-tls-cert" value="${n((o==null?void 0:o.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${s}-tls-key" value="${n((o==null?void 0:o.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${Ft(e)}
    `}function Ft(e){var P,w;const s=n(e.id),o=((P=e.config.TLS)==null?void 0:P.Enabled)??!1,c=O[e.id]??((w=e.tls)==null?void 0:w.verification)??null,h=L[e.id]??!1,k=T[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${s}" ${o&&!h?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${h?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${k?`<p class="error small">${n(k)}</p>`:""}
      ${c?Wt(c):""}
    `}function Wt(e){const s=(e.assertions??[]).map(o=>`
          <li class="small">
            ${_t(o.status)}
            <strong>${n(o.title)}</strong>
            <div class="muted">${n(o.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${e.ok?e.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${n(e.summary)}
      </div>
      <ul class="verify-list">${s}</ul>
      <p class="muted small">
        Checked ${n(new Date(e.at).toLocaleString())} against <code>${n(e.address)}</code>
        ${e.notAfter?`· certificate valid until <code>${n(new Date(e.notAfter).toLocaleString())}</code> (${n(e.expiresIn??"")})`:""}
      </p>
      ${e.expiryWarning?`<div class="banner banner-warn">${n(e.expiryWarning)}</div>`:""}
    `}function _t(e){switch(e){case"pass":return U("pass","ok");case"fail":return U("fail","bad");case"unavailable":return U("unavailable","warn");default:return U("skipped","neutral")}}async function Kt(e){L[e]=!0,T[e]=null,Y();try{O[e]=await Wn(e)}catch(s){T[e]=`${pe(s)}${Ie(s)}`}finally{L[e]=!1,Y()}}function ke(e){return{...e.config,Networks:(e.config.Networks??[]).map(s=>({ChainID:s.ChainID,Upstreams:s.Upstreams.map(o=>({...o}))}))}}async function Te(e,s,o){A[e]=null;try{await zn(e,s)}catch(c){return A[e]=`${o?o+": ":""}${pe(c)}`,Y(),!1}return await W(),!0}async function Vt(e,s){const o=s.dataset.gid??"";switch(e){case"refresh":await W();return;case"copy":s.dataset.copy&&await gn(s,s.dataset.copy);return;case"reprobe":await ue(o,!0);return;case"toggle-settings":j[o]=!j[o],Y();return;case"toggle-manage":D[o]=!D[o],Y();return;case"toggle-chain-detail":{const c=s.dataset.key??"";c&&(E[c]=!E[c]),Y();return}case"save-settings":await Gt(o);return;case"verify-tls":await Kt(o);return;case"trust-cert":await Yt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await Zt(o,e.slice(3));return;case"gw-create":case"gw-recreate":await Xt(o);return;case"gw-wipe":fn(o);return;case"add-gateway":bn();return;case"forget-gateway":await Qt(o);return;case"dismiss-orphan":await en(s.dataset.name??"");return;case"add-chain":tn(o);return;case"add-devnet":{const c=J(o);if(c){const h=((i==null?void 0:i.targets)??[]).some(k=>k.id===c.placement.targetId&&k.hasDevnet);ct(o,Le,h)}return}case"remove-chain":await sn(o,Number.parseInt(s.dataset.chain??"",10));return;case"add-endpoint":dt(o,Number.parseInt(s.dataset.chain??"",10));return;case"remove-endpoint":await on(s.dataset.key??"");return;case"reset-devnet":await pn(s.dataset.key??"",s.dataset.target??"");return;default:return}}async function Gt(e){const s=J(e);if(!s)return;const o=ke(s),c=a.querySelector(`#gw-${CSS.escape(e)}-port`),h=a.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const w=Number.parseInt(c.value.trim(),10);Number.isFinite(w)&&(o.Port=w)}h&&(o.BindAddr=h.value.trim());const k=a.querySelector(`#gw-${CSS.escape(e)}-metrics`);k&&(o.MetricsOff=!k.checked),o.TLS=zt(e,s);const P=s.status.State==="running";await Te(e,o,"Saving settings")&&(j[e]=!1,P&&(A[e]=null,Jt(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),Y())}function zt(e,s){var k,P,w,G,ee,he,Re;const o=vn=>a.querySelector(`#gw-${CSS.escape(e)}-${vn}`),c=o("tls");if(!c)return s.config.TLS??null;const h=Number.parseInt(((k=o("tls-port"))==null?void 0:k.value.trim())??"",10);return{Enabled:c.checked,Hostname:((P=o("tls-host"))==null?void 0:P.value.trim())??"",CertSource:((w=o("tls-source"))==null?void 0:w.value)??"internal",CertFile:((G=o("tls-cert"))==null?void 0:G.value.trim())??"",KeyFile:((ee=o("tls-key"))==null?void 0:ee.value.trim())??"",HTTPSPort:Number.isFinite(h)?h:443,BindAddr:((he=s.config.TLS)==null?void 0:he.BindAddr)??"",ImageRef:((Re=s.config.TLS)==null?void 0:Re.ImageRef)??""}}function Jt(e,s){M[e]=[s]}async function Yt(e){if(!m[e]){m[e]=!0,x[e]=null,Y();try{x[e]=await Jn(e)}catch(s){x[e]={ok:!1,message:`${pe(s)}${Ie(s)}`}}m[e]=!1,Y()}}async function Zt(e,s){if(!f[e]){f[e]=s,A[e]=null,Y();try{await kt(e,s)}catch(o){A[e]=`${s} failed: ${pe(o)}${Ie(o)}`}f[e]=null,await W()}}async function Xt(e){if(f[e])return;f[e]="create",A[e]=null,M[e]=["starting…"],Y();let s;try{s=await Tt(e)}catch(o){A[e]=`${pe(o)}${Ie(o)}`,M[e]=[],f[e]=null,Y();return}F==null||F(),F=Ge(s.targetId,o=>{if(r)return;const c=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(M[e]=[...(M[e]??[]).filter(k=>k!=="starting…"),c],!!o.err||o.stepId===Oa&&!!o.done){F==null||F(),F=null,f[e]=null,o.err&&(A[e]="Provisioning failed — see the log below."),W();return}Y()})}async function Qt(e){const s=J(e);if(!(!s||!await Ae({title:`Forget ${s.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${s.containerName}" on ${s.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await Gn(e)}catch(c){A[e]=pe(c),Y();return}await W()}}async function en(e){if(e){p[e]=null;try{await jn(e)}catch(s){p[e]=pe(s),Y();return}await W()}}function tn(e){const s=J(e);if(!s)return;const o=new Set((s.networks??[]).map(w=>w.chainId)),c=(i==null?void 0:i.presets)??[],h=c.filter(w=>!o.has(w.chainId)),k=c.filter(w=>o.has(w.chainId)),P=((i==null?void 0:i.targets)??[]).some(w=>w.id===s.placement.targetId&&w.hasDevnet);ie(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(s.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${h.map(w=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${w.chainId}">
                <span>${n(w.name)}</span>
                <span class="muted small">chain ${w.chainId}${w.devnet?P?" · uses the devnet on "+n(s.placement.targetId):" · will create a devnet on "+n(s.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${k.length?`<p class="muted small">Already fronted: ${n(k.map(w=>w.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,w=>{if(w==="cancel"){Z();return}if(w==="custom"){nn(e);return}if(w.startsWith("preset:")){const G=Number.parseInt(w.slice(7),10),ee=c.find(he=>he.chainId===G);Z(),ee!=null&&ee.devnet?ct(e,G,P):it(e,G)}})}function nn(e){var s;ie(`
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
      `,o=>{if(o==="cancel"){Z();return}if(o!=="add")return;const c=document.getElementById("custom-chain-id"),h=document.getElementById("custom-chain-err"),k=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(k)||k<=0){h&&(h.className="error small"),h&&(h.textContent="A chain id is a positive whole number.");return}Z(),it(e,k)}),(s=document.getElementById("custom-chain-id"))==null||s.focus()}async function it(e,s){const o=J(e);if(!o)return;const c=ke(o),h=c.Networks??[];h.some(k=>k.ChainID===s)||(h.push({ChainID:s,Upstreams:[]}),c.Networks=h,await an(e,c)&&(Y(),dt(e,s)))}async function an(e,s){var k;const o={...s,Networks:(s.Networks??[]).filter(P=>P.Upstreams.length>0)};if(!await Te(e,o))return!1;const h=J(e);if(h)for(const P of s.Networks??[])P.Upstreams.length===0&&!(h.networks??[]).some(w=>w.chainId===P.ChainID)&&(h.config.Networks=[...h.config.Networks??[],{ChainID:P.ChainID,Upstreams:[]}],h.networks=[...h.networks??[],{chainId:P.ChainID,name:((k=((i==null?void 0:i.presets)??[]).find(w=>w.chainId===P.ChainID))==null?void 0:k.name)??`Chain ${P.ChainID}`,path:`/${h.config.ProjectID}/evm/${P.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function ct(e,s,o){const c=J(e);if(!c)return;if(!o){ie(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${n(c.placement.targetId)}</code>, so adding chain ${s} here
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
        `,()=>Z());return}const h=ke(c),k=h.Networks??[],P={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},w=k.find(G=>G.ChainID===s);w?w.Upstreams.push(P):k.push({ChainID:s,Upstreams:[P]}),h.Networks=k,await Te(e,h,"Adding the devnet")}async function sn(e,s){const o=J(e);if(!o||!Number.isFinite(s))return;const c=se(o,s);if(!await Ae({title:`Remove ${(c==null?void 0:c.name)??`chain ${s}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${s}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const k=ke(o);k.Networks=(k.Networks??[]).filter(P=>P.ChainID!==s),await Te(e,k,"Removing the network")}function lt(e){const s=e.split("|");return s.length!==3?null:{gid:s[0],chainId:Number.parseInt(s[1],10),upstreamId:s[2]}}async function on(e){const s=lt(e);if(!s)return;const o=J(s.gid);if(!o)return;const c=ke(o),h=(c.Networks??[]).find(w=>w.ChainID===s.chainId);if(!h)return;const k=h.Upstreams.findIndex((w,G)=>(w.ID||`${s.chainId}-${G}`)===s.upstreamId);k<0||!await Ae({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(h.Upstreams.splice(k,1),await Te(s.gid,c,"Removing the endpoint"))}function dt(e,s){const o=J(e);if(!o||!Number.isFinite(s))return;const c=((i==null?void 0:i.sources)??[]).filter(w=>w.chainId===s),h=se(o,s),k=new Set(((h==null?void 0:h.upstreams)??[]).filter(w=>w.kind!=="external").map(w=>`${w.kind}|${w.targetId??""}`)),P=c.filter(w=>!k.has(`${w.kind}|${w.targetId}`));ie(`
        <h2>Add an endpoint for ${n((h==null?void 0:h.name)??`chain ${s}`)}</h2>
        ${P.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${P.map(w=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(w.kind)}:${n(w.targetId)}">
                       <span>${n(w.label)}</span>
                       <span class="muted small">${n(w.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${s}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn" data-modal-action="known-set">Add valve's set…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,w=>{if(w==="cancel"){Z();return}if(w==="known-set"){ln(e,s);return}if(w==="manual"){un(e,s);return}if(w.startsWith("source:")){const[,G,ee]=w.split(":");Z(),rn(e,s,G,ee)}})}async function rn(e,s,o,c){const h=J(e);if(!h)return;const k=ke(h),P=k.Networks??[],w={ID:`${o==="managed-devnet"?"devnet":"node"}-${c}`,Kind:o,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},G=P.find(ee=>ee.ChainID===s);G?G.Upstreams.push(w):P.push({ChainID:s,Upstreams:[w]}),k.Networks=P,await Te(e,k,"Adding the endpoint")}function cn(e){const s=[...e].sort((h,k)=>(h.latencyMs??1e9)-(k.latencyMs??1e9)),o=s.slice(0,3),c=s.find(h=>h.url.startsWith("wss://")||h.url.startsWith("ws://"));return c&&!o.some(h=>h.url===c.url)&&(o.length===3&&o.pop(),o.push(c)),new Set(o.map(h=>h.url))}async function ln(e,s){let o;try{o=await Zn(e,s)}catch(w){ie(`<h2>Endpoints for chain ${s}</h2>
         <p class="error small">Could not read the set: ${n(pe(w))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Z());return}if(r)return;const c=o.endpoints??[],h=c.filter(w=>!w.alreadyAdded).map(w=>w.url),k=new Set(c.map(w=>w.provider)).size,P=c.map(w=>{const G=[w.websocket?'<span class="t ws">websocket</span>':"",w.archive?'<span class="t ar">archive</span>':"",w.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n(w.url)}</code>
                  <span class="muted small">${n(w.provider)}</span> ${G}</li>`}).join("");ie(`<h2>Endpoints for chain ${s}</h2>
       ${c.length?`<p class="muted small">${k} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${P}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${o.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${h.length?"":" disabled"}>
           ${h.length?`Add ${h.length}`:"Nothing to add"}</button>
       </div>`,w=>{Z(),w==="add"&&Ye(e,s,h),w==="discover"&&dn(e,s)})}async function dn(e,s){ie(`
        <h2>Public endpoints for chain ${s}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,P=>{P==="cancel"&&Z()});let o;try{o=await Yn(s)}catch(P){const w=He();if(w){const G=document.createElement("p");G.className="error small",G.textContent=`Could not discover endpoints: ${pe(P)}`,w.appendChild(G)}return}if(r)return;const c=(o.endpoints??[]).filter(P=>P.status==="live"||P.status==="unprobed"),h=(o.endpoints??[]).filter(P=>P.status==="rejected"),k=cn(c);ie(`
        <h2>Public endpoints for chain ${s}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(P=>{const w=k.has(P.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(P.url)}"${w}>
                       <span><code>${n(P.url)}</code></span>
                       <span class="muted small">${P.status==="live"?`answered in ${P.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${s} right now.</p>`}
        ${h.length?`<details class="rpc-rejected">
                 <summary class="muted small">${h.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${h.map(P=>`<li class="muted small"><code>${n(P.url)}</code> — ${n(P.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,P=>{if(P==="cancel"){Z();return}if(P==="add"){const w=He(),G=w?Array.from(w.querySelectorAll('input[type="checkbox"]:checked')).map(ee=>ee.value):[];Z(),Ye(e,s,G);return}})}function un(e,s){var o;ie(`
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
      `,c=>{if(c==="cancel"){Z();return}if(c!=="add")return;const h=document.getElementById("manual-endpoint"),k=document.getElementById("manual-recent"),P=document.getElementById("manual-err"),w=(h==null?void 0:h.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(w)){P&&(P.className="error small",P.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Z(),Ye(e,s,[w],(k==null?void 0:k.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ye(e,s,o,c=!1){if(!o.length)return;const h=J(e);if(!h)return;const k=ke(h),P=k.Networks??[];let w=P.find(ee=>ee.ChainID===s);w||(w={ChainID:s,Upstreams:[]},P.push(w));let G=1;for(const ee of w.Upstreams){const he=/^public-\d+-(\d+)$/.exec(ee.ID??"");he&&(G=Math.max(G,Number(he[1])+1))}for(const ee of o)w.Upstreams.some(he=>he.Endpoint===ee)||w.Upstreams.push({ID:`public-${s}-${G++}`,Kind:"external",Endpoint:ee,Local:!1,RecentOnly:c});k.Networks=P,await Te(e,k,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function pn(e,s){const o=lt(e);if(!o||!s||!await Ae({title:"Reset this devnet",body:`The chain on ${s} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[o.gid]="reset",A[o.gid]=null,Y();let h;try{h=await On(s)}catch(k){A[o.gid]=`Reset failed: ${pe(k)}${Ie(k)}`,f[o.gid]=null,Y();return}f[o.gid]=null,hn(s,h),await W()}function hn(e,s){const o=[];o.push(s.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),s.report.Recreated&&o.push("A fresh chain was started from genesis.");const c=s.report.Cascaded??[],h=s.report.CascadeSkipped??[];ie(`
        <h2>Devnet on ${n(e)} reset</h2>
        <ul class="plain-list">${o.map(k=>`<li>${n(k)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${n(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${h.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(h.join(", "))}.</p>`:""}
        ${s.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(s.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Z())}function fn(e){const s=J(e);if(!s)return;ie(`
        <h2>Wipe ${n(s.label)}</h2>
        <p class="error">This destroys ${n(s.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(e)}</button>
        </div>
      `,h=>{if(h==="cancel"||h==="close"){Z(),W();return}h==="confirm"&&mn(e)});const o=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{c&&(c.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function mn(e){const s=document.getElementById("wipe-confirm-btn");s&&(s.disabled=!0,s.textContent="Wiping…");let o;try{o=await St(e)}catch(c){const h=He();if(h){const k=document.createElement("p");k.className="error small",k.textContent=`Wipe failed: ${pe(c)}${Ie(c)}`,h.appendChild(k)}s&&(s.disabled=!1,s.textContent=`Wipe ${e}`);return}ie(`
        <h2>${n(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Z(),W()})}function ut(e,s){return!s.some(o=>{var c;return((c=o.placement)==null?void 0:c.targetId)===e})}function bn(){var k;const e=(i==null?void 0:i.targets)??[],s=(i==null?void 0:i.gateways)??[],o=e.filter(P=>ut(P.id,s)),c=new Set(s.map(P=>P.id));if(e.length===0){ie(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Z());return}if(o.length===0){ie(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Z());return}const h=c.has("default")?"":"default";ie(`
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
            ${o.map(P=>`<option value="${n(P.id)}">${n(P.id)} (${n(P.mode)})</option>`).join("")}
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
      `,P=>{if(P==="cancel"){Z();return}P==="create"&&yn()}),(k=document.getElementById("new-gw-id"))==null||k.focus()}async function yn(){const e=document.getElementById("new-gw-id"),s=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),h=(e==null?void 0:e.value.trim())??"",k=(s==null?void 0:s.value)??"",P=Number.parseInt((o==null?void 0:o.value.trim())??"",10),w=G=>{c&&(c.className="error small",c.textContent=G)};if(!h){w("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!k){w("Pick the machine it runs on.");return}try{await Fn({id:h,placement:{targetId:k,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(P)?P:4e3,Networks:[]}})}catch(G){w(pe(G));return}Z(),await W()}async function gn(e,s){const o=await De(s),c=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{r||(e.textContent=c)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Ie(e){return e instanceof we&&e.hint?` — ${e.hint}`:""}return()=>{r=!0,F==null||F(),Z()}}function Fa(a,r){if(a.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const i=a.filter(f=>!f.wire);if(i.length>0){const f=i.map(M=>M.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const t=r.networks??[],u=f=>{const A=t.find(M=>M.ChainID===f);return A?A.Name:`chain ${f}`},y=_a(a.map(f=>u(f.wire.ChainID))),S=a.length===1?"machine":"machines";return{level:"ok",sentence:`All ${a.length} ${S} healthy — ${Ka(y)}.`,machines:[]}}function Wa(a,r){const i=r.machines.length?` <span class="verdict-machines">${r.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${n(t)}</a>`).join(" ")}</span>`:"";a.innerHTML=`
    <div class="verdict-line verdict-${r.level}">
      ${U(r.level==="ok"?"OK":"Attention",r.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(r.sentence)}</strong>${i}
    </div>
  `}function _a(a){return[...new Set(a)]}function Ka(a){return a.length<=1?a[0]??"":a.length===2?`${a[0]} and ${a[1]}`:`${a.slice(0,-1).join(", ")} and ${a[a.length-1]}`}const Va="local";function Ga(a){let r=!1,i=!1,t="",u=null;a.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${de()}
  `;const y=a.querySelector("#targets-body");ye(a,(p,m)=>{j(p,m)}),S();async function S(){try{const[p,m,x]=await Promise.all([xe(),Ce(),wt()]);if(r)return;t=x.os,A(p,m)}catch(p){if(r)return;y.innerHTML=`<p class="error">Failed to load machines: ${n(String(p))}</p>`}}function f(){u&&A(u.targets,u.catalog)}function A(p,m){u={targets:p,catalog:m};const x=t==="linux",B=[...p].sort((W,ae)=>(W.mode==="local"?-1:0)-(ae.mode==="local"?-1:0)),F=B.length?`<div class="card-grid">${B.map(W=>za(W,m,W.mode!=="local"||x,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=p.some(W=>W.mode==="local");y.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${F}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${M(x,X)}
        ${i?Ja():""}
      </section>
    `;const te=y.querySelector("#fleet-verdict");te&&Wa(te,Fa(p,m))}function M(p,m){const x=`
      <div class="card">
        <h3>A server over SSH ${U("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${p?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${p?" btn-ghost":""}" data-action="toggle-ssh">
            ${i?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,B=p?`
        <div class="card">
          <h3>This machine ${U("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${n(t)})`:""} ${U("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return m?`<div class="card-grid card-grid-wide">${x}</div>`:`<div class="card-grid card-grid-wide">${p?B+x:x+B}</div>`}async function j(p,m){var x;if(p==="add-local"){await E();return}if(p==="delete-target"){const B=m.dataset.id;if(!B||!await Ae({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await D(B);return}if(p==="toggle-ssh"){i=!i,T(),f(),i&&((x=a.querySelector("#ssh-host"))==null||x.focus());return}p==="add-ssh"&&await O()}async function E(){T();try{await pt({id:Va,mode:"local"}),await S()}catch(p){L(p)}}async function D(p){try{await kn(p),await S()}catch(m){L(m)}}async function O(){const p=a.querySelector("#ssh-host"),m=a.querySelector("#ssh-user"),x=a.querySelector("#ssh-key"),B=a.querySelector("#ssh-port"),F=a.querySelector("#ssh-id");if(!p||!m||!x||!B||!F)return;const X=p.value.trim(),te=m.value.trim(),W=x.value.trim(),ae=B.value.trim(),ue=F.value.trim();if(T(),!X||!te||!W){L(new Error("host, user, and key path are required"));return}const J=ue||Ya(X),se={Host:X,User:te,KeyPath:W};if(ae){const le=Number.parseInt(ae,10);if(!Number.isFinite(le)||le<=0){L(new Error("port must be a positive number"));return}se.Port=le}const oe=a.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await pt({id:J,mode:"ssh",ssh:se}),i=!1,await S()}catch(le){L(le),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function L(p){let m=a.querySelector("#targets-error");m||(y.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),m=a.querySelector("#targets-error")),m.textContent=String(p instanceof Error?p.message:p)}function T(){var p;(p=a.querySelector("#targets-error"))==null||p.remove()}return()=>{r=!0}}function za(a,r,i,t){const u=a.wire,y=a.mode==="local"?"this machine":"SSH",S=a.mode==="ssh"&&a.ssh?`${n(a.ssh.User)}@${n(a.ssh.Host)}`:y;let f;if(!u&&!i)f=`${U("can't run a node","warn")} ${U(t||"not Linux","neutral")}`;else if(!u)f=U("not set up","neutral");else{const A=r.networks.find(j=>j.ChainID===u.ChainID),M=A?A.Name:`chain ${u.ChainID}`;f=`${U(M,"ok")} ${U(u.ExecID,"neutral")} ${U(u.BeaconID,"neutral")}${u.Archive?" "+U("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(a.id)}</h2>
      <p class="muted">${S}</p>
      <p>${f}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(a.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(a.id)}">Remove</button>
      </div>
    </div>
  `}function Ja(){return`
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
  `}function Ya(a){return a.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Za=document.querySelector("#app"),{contentEl:Xa,setActiveNav:Qa}=ea(Za);let fe=null;function es(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"home"};const[i,t]=r;return i==="machine"||i==="setup"||i==="dash"||i==="logs"||i==="security"||i==="diag"||i==="services"||i==="analytics"?{screen:i,id:t?decodeURIComponent(t):void 0}:{screen:i??"targets"}}function ve(a){const r=document.createElement("div");return Xa.replaceChildren(r),a(r)}function xt(){if(fe){try{fe()}catch{}fe=null}const{screen:a,id:r}=es();switch(Qa(a),a){case"machine":if(!r){location.hash="#/targets";return}fe=ve(i=>ma(i,r));break;case"setup":case"dash":case"logs":case"services":if(!r){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(r)}`;return;case"security":if(!r){location.hash="#/targets";return}fe=ve(i=>Na(i,r));break;case"diag":if(!r){location.hash="#/targets";return}fe=ve(i=>oa(i,r));break;case"analytics":if(!r){location.hash="#/rpc";return}fe=ve(i=>sa(i,r));break;case"rpc":fe=ve(i=>ja(i));break;case"settings":fe=ve(i=>Da(i));break;case"targets":fe=ve(i=>Ga(i));break;case"panel":fe=ve(i=>vt(i));break;case"home":default:fe=ve(i=>vt(i));break}}window.addEventListener("hashchange",xt);xt();
