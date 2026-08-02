var Pn=Object.defineProperty;var En=(n,r,i)=>r in n?Pn(n,r,{enumerable:!0,configurable:!0,writable:!0,value:i}):n[r]=i;var We=(n,r,i)=>En(n,typeof r!="symbol"?r+"":r,i);(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))t(d);new MutationObserver(d=>{for(const b of d)if(b.type==="childList")for(const w of b.addedNodes)w.tagName==="LINK"&&w.rel==="modulepreload"&&t(w)}).observe(document,{childList:!0,subtree:!0});function i(d){const b={};return d.integrity&&(b.integrity=d.integrity),d.referrerPolicy&&(b.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?b.credentials="include":d.crossOrigin==="anonymous"?b.credentials="omit":b.credentials="same-origin",b}function t(d){if(d.ep)return;d.ep=!0;const b=i(d);fetch(d.href,b)}})();function St(){return Y("/api/host")}function Pe(){return Y("/api/catalog")}function Te(){return Y("/api/targets")}function tt(n){return Y("/api/targets",{method:"POST",headers:be,body:JSON.stringify(n)})}function In(n){return Y(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function Rn(n,r){return Y(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(r)}`)}function Ln(n,r){return Y(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:be,body:JSON.stringify(r)})}function Oe(n,r){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function An(n,r){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function Nn(n,r=200){return Y(`/api/targets/${encodeURIComponent(n)}/logs?n=${r}`)}function Bn(n,r){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return i.onmessage=t=>{try{r(JSON.parse(t.data))}catch{}},()=>i.close()}function mt(n,r){const i=r===void 0?{}:{lines:r};return Y(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:be,body:JSON.stringify(i)})}function Hn(n,r,i){return Y(`/api/targets/${encodeURIComponent(n)}/services/${r}/${i}`,{method:"POST"})}function Dn(n,r){return Y(`/api/targets/${encodeURIComponent(n)}/services/${r}/clear`,{method:"POST",headers:be,body:JSON.stringify({Confirm:r})})}function Mn(n){return Y(`/api/targets/${encodeURIComponent(n)}/du`)}function Un(n){return Y(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function On(n){return Y(`/api/targets/${encodeURIComponent(n)}/firewall`)}function qn(n){return Y(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function jn(n){return Y(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function xt(n){return Y(`/api/targets/${encodeURIComponent(n)}/containers`)}function Fn(n,r,i){return Y(`/api/targets/${encodeURIComponent(n)}/containers/${r}/${i}`,{method:"POST"})}async function _n(n,r){const i=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${r}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:r})}),t=await i.text();let d=null;try{d=t?JSON.parse(t):null}catch{}if(d&&typeof d=="object"&&"report"in d)return d;const b=d&&typeof d=="object"&&typeof d.error=="string"?d.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,b)}function Wn(n,r){return Y(`/api/targets/${encodeURIComponent(n)}/containers/${r}/provision`,{method:"POST"})}async function Kn(n){const r=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:be}),i=await r.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const d=t&&typeof t=="object"&&typeof t.error=="string"?t.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,d)}function Vn(n,r,i){return Y(`/api/targets/${encodeURIComponent(n)}/containers/${r}/config`,{method:"PUT",headers:be,body:JSON.stringify(i)})}function ot(){return Y("/api/gateways")}async function Gn(n){await Y(`/api/orphans/${encodeURIComponent(n)}`,{method:"DELETE"})}function Pt(n){return Y("/api/gateways",{method:"POST",headers:be,body:JSON.stringify(n)})}function zn(n){return Y(`/api/gateways/${encodeURIComponent(n)}/tls/verify`)}function Jn(n){return Y(`/api/gateways/${encodeURIComponent(n)}/traffic`)}function Yn(n){return Y(`/api/gateways/${encodeURIComponent(n)}/analytics`)}function Zn(n,r=!1){const i=r?"?refresh=1":"";return Y(`/api/gateways/${encodeURIComponent(n)}/capabilities${i}`)}function Xn(n){return Y(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function Et(n,r){return Y(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:be,body:JSON.stringify(r)})}function It(n,r){return Y(`/api/gateways/${encodeURIComponent(n)}/${r}`,{method:"POST"})}function Qn(n){return Y(`/api/gateways/${encodeURIComponent(n)}/trust-cert`,{method:"POST"})}function nt(n){return Y(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Rt(n){const r=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:n})}),i=await r.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const d=t&&typeof t=="object"&&typeof t.error=="string"?t.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,d)}function ea(n){return Y(`/api/chainlist/${n}`)}function Lt(n,r){return Y(`/api/gateways/${encodeURIComponent(n)}/knownset/${r}`)}function ta(){return Y("/api/settings")}function na(n){return Y("/api/settings",{method:"PUT",headers:be,body:JSON.stringify(n)})}class we extends Error{constructor(i,t,d,b){super(t);We(this,"status");We(this,"hint");We(this,"code");this.name="ApiError",this.status=i,this.hint=d,this.code=b}}const be={"Content-Type":"application/json"};async function Y(n,r){const i=await fetch(n,r);if(!i.ok){let d=i.statusText||`HTTP ${i.status}`,b,w;try{const f=await i.json();f&&typeof f.error=="string"&&f.error&&(d=f.error),f&&typeof f.hint=="string"&&f.hint&&(b=f.hint),f&&typeof f.code=="string"&&f.code&&(w=f.code)}catch{}throw new we(i.status,d,b,w)}if(i.status===204)return;const t=await i.text();return t?JSON.parse(t):void 0}const bt="https://learn.valve.city/rpc";function a(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ue(n,r){const i=n&&r&&r!==bt?` <span class="footer-sep">·</span> <a href="${a(r)}" target="_blank" rel="noopener noreferrer">${a(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(bt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${i}
    </footer>
  `}function aa(n){n.innerHTML=`
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
  `;const r=n.querySelector("#content"),i=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:r,setActiveNav:d=>{const b=d==="machine"?"targets":d==="home"||d==="panel"?"rpc":d;for(const w of i)w.classList.toggle("active",w.dataset.nav===b)}}}function le(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function sa(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function oa(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const r=Math.round(n/60),i=Math.floor(r/60),t=r%60;if(i===0)return`~${t}m`;if(i<48)return`~${i}h ${t}m`;const d=Math.floor(i/24),b=i%24;return`~${d}d ${b}h`}function F(n,r){return`<span class="badge badge-${r}">${a(n)}</span>`}function ke(n){return`<span class="dot dot-${n}"></span>`}const yt=["B","KB","MB","GB","TB","PB"];function xe(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let r=n,i=0;for(;r>=1024&&i<yt.length-1;)r/=1024,i++;const t=r<10?2:r<100?1:0;return`${r.toFixed(t)} ${yt[i]}`}async function qe(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function ye(n,r){n.addEventListener("click",i=>{const t=i.target.closest("[data-action]");if(!t||!n.contains(t))return;const d=t.dataset.action;d&&r(d,t,i)})}function at(n,r,i){const t=r.find(b=>b.value===i),d=r.map(b=>`
      <li class="dropdown-option${b.value===i?" selected":""}" role="option"
          aria-selected="${b.value===i}" data-value="${a(b.value)}">
        ${a(b.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function De(n){n.querySelectorAll(".dropdown.open").forEach(r=>{var i;r.classList.remove("open"),(i=r.querySelector(".dropdown-trigger"))==null||i.setAttribute("aria-expanded","false")})}function rt(n,r){n.addEventListener("click",d=>{const b=d.target,w=b.closest(".dropdown-trigger");if(w&&n.contains(w)){const N=w.closest(".dropdown"),q=!!N&&!N.classList.contains("open");De(n),N&&q&&(N.classList.add("open"),w.setAttribute("aria-expanded","true"));return}const f=b.closest(".dropdown-option");if(f&&n.contains(f)){const N=f.closest(".dropdown");De(n),r((N==null?void 0:N.dataset.dropdown)??"",f.dataset.value??"");return}De(n)});const i=d=>{if(!n.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}const b=d.target;(!b.closest(".dropdown")||!n.contains(b))&&De(n)},t=d=>{if(!n.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}d.key==="Escape"&&De(n)};document.addEventListener("click",i),document.addEventListener("keydown",t)}const Je="app-modal";let ze=null;function ce(n,r){Q();const i=document.createElement("div");i.className="modal-overlay",i.id=Je,i.innerHTML=`<div class="modal">${n}</div>`,i.addEventListener("click",d=>{const b=d.target.closest("[data-modal-action]");b!=null&&b.dataset.modalAction?r(b.dataset.modalAction):d.target===i&&r("cancel")});const t=d=>{d.key==="Escape"&&r("cancel")};document.addEventListener("keydown",t),ze=t,document.body.appendChild(i)}function Q(){var n;(n=document.getElementById(Je))==null||n.remove(),ze&&(document.removeEventListener("keydown",ze),ze=null)}function Ue(){return document.querySelector(`#${Je} .modal`)}function Be(n){return new Promise(r=>{var d;let i=!1;const t=b=>{i||(i=!0,Q(),r(b))};ce(`
        <h2>${a(n.title)}</h2>
        <p>${a(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${a(n.confirmLabel)}</button>
        </div>
      `,b=>t(b==="confirm")),(d=document.querySelector(`#${Je} [data-modal-action="confirm"]`))==null||d.focus()})}const Xe=5e3,ra=60;function ia(n,r){let i=!1,t=null,d=null,b=null,w=null;const f=[];n.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ue()}</div>`;const N=n.querySelector("#an-body");ye(n,($,u)=>{var S;$==="toggle-endpoint"&&((S=u.closest(".an-endpoint"))==null||S.classList.toggle("expanded"))}),q();async function q(){try{t=((await ot()).gateways??[]).find(u=>u.id===r)??null}catch($){if(i)return;b=String($ instanceof Error?$.message:$),O();return}if(!i){if(!t){O();return}await M(),w=window.setInterval(()=>void M(),Xe)}}async function M(){try{const $=await Yn(r);if(i)return;I($),d=$,b=null}catch($){if(i)return;b=String($ instanceof Error?$.message:$)}O()}function I($){if(!$.enabled||$.error)return;const u=f[f.length-1];u&&u.since!==$.since&&(f.length=0);const S=new Map;for(const A of $.networks??[])S.set(A.chainId,A.received);f.push({t:Date.now(),since:$.since,received:S}),f.length>ra&&f.shift()}function O(){i||(N.innerHTML=j())}function j(){return b&&!d?`<h1>Analytics</h1><p class="error">${a(b)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${L(t)}
      ${d?p(d):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${a(r)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function L($){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${a($.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${E()}</div>
      </div>
    `}function E(){if(!d)return"";if(!d.enabled)return"counters off";if(d.error)return"could not be read";const $=d.since?new Date(d.since):null;return $&&!Number.isNaN($.getTime())?`totals since the gateway started, ${a($.toLocaleString())}<br />re-read every ${Xe/1e3}s`:`re-read every ${Xe/1e3}s`}function p($){return $.enabled?$.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a($.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:m($)+de($):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function m($){const u=$.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${u.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':u.map(S=>v(S)).join("")}
      </section>
    `}function v($){const u=$.methods??[],S=$.endpoints??[],A=$.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${$.chainId}</span>
          <span class="band-name">${a($.name)}</span>
          ${_($)}
        </div>
        <div class="an-stats">
          ${x("Received",le($.received),"what clients asked this chain for")}
          ${x("Answered",le($.answered),"returned by one of your endpoints")}
          ${x("From cache",le($.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${x("Failed",le($.failed),"asked for and never answered",$.failed>0?"bad":"")}
        </div>
        ${H($.chainId)}
        ${A?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ne("Method",u.map(D=>({label:D.method,l:D})))+ne("Endpoint",S.map(D=>({label:D.upstream,l:D})))+U($)}
      </div>
    `}function x($,u,S,A=""){return`
      <div class="an-stat${A?" an-stat-"+A:""}" title="${a(S)}">
        <span class="an-stat-n">${a(u)}</span>
        <span class="an-stat-l">${a($)}</span>
      </div>
    `}function _($){const u=Z($.chainId);if(u===null)return'<span class="an-rate muted small">measuring rate…</span>';const S=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${S}s apart.">
      ${a(u.toFixed(u<10?2:0))} req/s <span class="muted">over the last ${S}s</span>
    </span>`}function Z($){if(f.length<2)return null;const u=f[0],S=f[f.length-1],A=(S.t-u.t)/1e3;if(A<=0)return null;const D=(S.received.get($)??0)-(u.received.get($)??0);return D<0?null:D/A}function H($){if(f.length<3)return"";const u=[];for(let k=1;k<f.length;k++){const R=f[k-1],K=f[k],l=(K.t-R.t)/1e3,g=(K.received.get($)??0)-(R.received.get($)??0);u.push(l>0&&g>=0?g/l:0)}const S=Math.max(...u);if(S<=0)return"";const A=240,D=28,V=u.length>1?A/(u.length-1):A,y=u.map((k,R)=>`${(R*V).toFixed(1)},${(D-k/S*D).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${S.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${A} ${D}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${y}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(S.toFixed(2))} req/s</span>
      </div>
    `}function U($){const u=[];return $.cached.count>0&&u.push(`${a(le($.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${$.cached.mean===null?"":`, in ${a(Me($.cached.mean))} on average`}.`),$.failedLatency.count>0&&$.failedLatency.mean!==null&&u.push(`The ${a(le($.failedLatency.count))} that failed took
         ${a(Me($.failedLatency.mean))} on average to fail.`),u.length===0?"":`<p class="muted small">${u.join(" ")}</p>`}function ne($,u){return u.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${a($)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${u.map(S=>ae(S.label,S.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function ae($,u){return`
      <tr>
        <td><code>${a($)}</code></td>
        <td class="an-num">${le(u.count)}</td>
        <td class="an-num">${u.mean===null?'<span class="muted">—</span>':a(Me(u.mean))}</td>
        <td>${J(u)}</td>
      </tr>
    `}function J($){const u=$.buckets??[];if(u.length===0||$.count===0)return'<span class="muted small">—</span>';let S=0;const A=[];for(const V of u){const y=V.count-S;S=V.count,A.push({label:re(V.le),n:Math.max(0,y)})}return A.reduce((V,y)=>V+y.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(A.filter(V=>V.n>0).map(V=>`${V.n} ${V.label}`).join(" · "))}">
        ${A.map((V,y)=>V.n===0?"":`<span class="an-band an-band-${Math.min(y,4)}" style="flex:${V.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(oe(A))}</span>
    `}function oe($){for(let u=$.length-1;u>=0;u--)if($[u].n>0)return`slowest ${$[u].label}`;return""}function re($){if($==="+Inf")return"30s or more";const u=Number($);return Number.isFinite(u)?`under ${Me(u)}`:`under ${$}`}function de($){const u=$.endpoints??[];return`
      <section class="an-section">
        <h2>What the gateway sees from your endpoints</h2>
        <p class="muted small">
          The gateway's own view, not a client's. Every count here <strong>includes the
          gateway's block-tracking poller</strong>, which calls each endpoint on a timer
          whether or not anyone is using it — on a quiet gateway it is nearly all of this.
          That is why these numbers are much larger than the ones above, and why they are
          not a measure of your traffic.
        </p>
        ${u.length===0?'<div class="card"><p class="muted">The gateway has not talked to any endpoint yet.</p></div>':`<div class="card">
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
                     <tbody>${u.map(S=>X(S)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function X($){const u=$.errors??[],S=u.reduce((D,V)=>D+V.count,0),A=u.length>0;return`
      <tr class="an-endpoint${A?" expandable":""}" ${A?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${a($.upstream)}</code>
          ${$.chainId?`<span class="muted small">chain ${$.chainId}</span>`:""}
          ${$.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${le($.requests)}</td>
        <td class="an-num${S>0?" bad":""}">${S>0?le(S):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${$.headLag>0?le($.headLag):'<span class="muted">0</span>'}</td>
        <td>${me($)}</td>
      </tr>
      ${A?ge($,u):""}
    `}function me($){const u=[];return $.scored?(u.push($.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String($.position))}</span>`),u.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a($.score.toFixed(3))}</span>`),$.primarySwitches>1&&u.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${le($.primarySwitches)} switches</span>`),$.excludedSeconds>0&&u.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(Me($.excludedSeconds))}</span>`),`<span class="an-selection">${u.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge($,u){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${u.map(S=>`
                    <tr>
                      <td class="an-num">${le(S.count)}</td>
                      <td><code>${a(S.class)}</code></td>
                      <td>${S.severity?`<span class="badge badge-${S.severity==="critical"?"bad":"warn"}">${a(S.severity)}</span>`:""}</td>
                      <td class="muted small">${a(S.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${a($.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{i=!0,w!==null&&window.clearInterval(w)}}function Me(n){return!Number.isFinite(n)||n<0?"—":n>0&&n<5e-4?"<1ms":n<1?`${Math.round(n*1e3)}ms`:n<60?`${n<10?n.toFixed(1):Math.round(n)}s`:`${Math.round(n/60)}m`}function ca(n,r){let i=!1,t=null,d=null,b=!1,w=!1;n.innerHTML=`<h1>Network diagnostics: ${a(r)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ue()}</div>`;const f=n.querySelector("#diag-body"),N=n.querySelector("#diag-footer");ye(n,(p,m)=>{var v;if(p==="run")M();else if(p==="toggle")(v=m.closest(".check-item"))==null||v.classList.toggle("expanded");else if(p==="copy"){const x=m.dataset.copy;x&&E(m,x)}}),q();async function q(){let p,m;try{const[x,_]=await Promise.all([Te(),Pe()]);p=x.find(Z=>Z.id===r),m=_}catch(x){if(i)return;f.innerHTML=`<p class="error">Failed to load target: ${a(String(x))}</p>`;return}if(i)return;if(!p){f.innerHTML=`<p class="error">Target "${a(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const v=m==null?void 0:m.networks.find(x=>x.ChainID===p.wire.ChainID);v&&(N.innerHTML=ue(v.Name,v.LearnURL));try{t=await jn(r),w=!0}catch(x){d=String(x instanceof Error?x.message:x)}i||I()}async function M(){b=!0,d=null,I();try{t=await qn(r),w=!0}catch(p){d=String(p instanceof Error?p.message:p)}b=!1,i||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${b?"disabled":""}>${b?"Running…":"Run diagnostics"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${O()}
    `}function O(){if(!w&&!d)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const p=new Date(t.at).toLocaleString(),m=t.failedId?`<p><strong>Failed at: ${a(j(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${a(p)} — trigger: ${a(t.trigger)}</p>
      ${m}
      <ul class="check-list">${t.items.map(L).join("")}</ul>
    `}function j(p){var m;return((m=t==null?void 0:t.items.find(v=>v.ID===p))==null?void 0:m.Title)??p}function L(p){const m=p.Status==="pass"?"ok":p.Status==="fail"?"bad":p.Status==="warn"?"warn":"neutral",v=p.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${v?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${F(v?"failed here":p.Status,m)}
          <strong>${a(p.Title)}</strong>
          <span class="muted small check-detail-inline">${a(p.Detail)}</span>
        </button>
        <div class="check-body">
          <details${v?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(p.Why)}</p>
          </details>
          ${p.Fix?`
                <details${v?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(p.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(p.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function E(p,m){const v=await qe(m),x=p.textContent;p.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{i||(p.textContent=x)},1500)}return()=>{i=!0}}const la=85,Qe={exec:"Execution",beacon:"Beacon"};function da(n,r){let i=!1,t=null,d=null,b=null,w=null,f=null,N=null,q=null,M=null;const I={exec:null,beacon:null};let O=null;n.innerHTML=`<h1>Dashboard: ${a(r)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ue()}</div>`;const j=n.querySelector("#dash-body"),L=n.querySelector("#dash-footer");j.addEventListener("click",u=>{const S=u.target.closest("[data-action]");if(!S||!j.contains(S))return;const A=S.dataset.action;if(A==="svc-action"){const D=S.dataset.svc,V=S.dataset.kind;D&&V&&X(D,V)}else if(A==="open-clear"){const D=S.dataset.svc;D&&ge(D)}else if(A==="copy"){const D=S.dataset.copy;D&&me(S,D)}else A==="retry-du"?p():A==="retry-endpoints"&&m()}),E();async function E(){let u,S;try{const[D,V]=await Promise.all([Te(),Pe()]);u=D.find(y=>y.id===r),S=V}catch(D){if(i)return;j.innerHTML=`<p class="error">Failed to load target: ${a(String(D))}</p>`;return}if(i)return;if(!u){j.innerHTML=`<p class="error">Target "${a(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){j.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const A=S==null?void 0:S.networks.find(D=>D.ChainID===u.wire.ChainID);A&&(L.innerHTML=ue(A.Name,A.LearnURL)),j.innerHTML='<p class="muted">Connecting…</p>',t=An(r,D=>{i||(v(D),d=D,b=D,x())}),p(),m()}async function p(){N=null;try{f=await Mn(r)}catch(u){f=null,N=String(u instanceof Error?u.message:u)}i||x()}async function m(){M=null;try{q=await Un(r)}catch(u){q=null,M=String(u instanceof Error?u.message:u)}i||x()}function v(u){if(!d)return;const S=(new Date(u.at).getTime()-new Date(d.at).getTime())/1e3,A=u.execHead-d.execHead;if(S>0&&A>=0){const D=A/S;w=w===null?D:w*.7+D*.3}}function x(){if(!b)return;const u=b;j.innerHTML=`
      <p class="dash-status">${_(u)}</p>
      <div class="card-grid">
        ${re(u)}
        ${H(u)}
        ${U(u)}
        ${ne(u)}
        ${ae(u)}
        ${J()}
      </div>
      <p class="muted small">Last updated ${a(new Date(u.at).toLocaleTimeString())}</p>
    `}function _(u){return!u.execActive&&!u.beaconActive?F("Node not running","bad"):u.execSyncing||u.beaconDistance>0?F("Syncing","warn"):F("Running · synced","ok")}function Z(u){const A=u.refHead>0?u.refHead-u.execHead:null,D=A!==null&&A>0&&w&&w>0?oa(A/w):A!==null&&A<=0?"caught up":"—";return{lag:A,eta:D}}function H(u){const{lag:S,eta:A}=Z(u);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${u.execActive?u.execSyncing?F("syncing","warn"):u.execHead===0?F("no data","neutral"):F("synced","ok"):F("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${le(u.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${S!==null?le(u.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${S!==null?le(Math.max(S,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${A}</dd></div>
        </dl>
      </div>
    `}function U(u){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${u.beaconActive?u.beaconSlot===0?F("no data","neutral"):u.beaconDistance===0?F("synced","ok"):F("syncing","warn"):F("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${le(u.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${le(u.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ne(u){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${le(u.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${le(u.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function ae(u){const S=u.diskUsedPct>=la,A=`
      <div class="meter"><div class="meter-fill ${S?"meter-warn":""}" style="width:${Math.min(u.diskUsedPct,100)}%"></div></div>
      <p>${sa(u.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="error small">${a(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${A}
          <p class="muted">Loading…</p>
        </div>
      `;const D=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,V=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:y,eta:k}=Z(u),R=y!==null&&y>0&&w!==null&&w>0;return`
      <div class="card ${S?"card-warn":""}">
        <h3>Storage</h3>
        ${A}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${xe(f.ExecBytes)} of ~${xe(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${D}%"></div></div>
        ${R?`<p class="muted small">Estimated time remaining: ${a(k)}</p>`:""}
        <p class="muted small">Beacon — ${xe(f.BeaconBytes)} of ~${xe(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${V}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${xe(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function J(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!q)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const u=q,S=u.ExecReachable&&!u.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",A=u.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(u.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ke(u.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(u.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ke(u.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(u.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.BeaconHTTP)}">Copy</button>
        </div>
        ${S}
        ${A}
      </div>
    `}function oe(u,S){const A=Qe[u],D=I[u],V=(y,k,R)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${u}" data-kind="${y}" ${D!==null||R?"disabled":""}>${D===y?de():a(k)}</button>`;return`
      <div class="service-row">
        <span>${a(A)} ${S?F("active","ok"):F("down","bad")}</span>
        <div class="service-actions">
          ${V("start","Start",S)}
          ${V("stop","Stop",!S)}
          ${V("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${u}" ${D!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function re(u){return`
      <div class="card">
        <h3>Services</h3>
        ${oe("exec",u.execActive)}
        ${oe("beacon",u.beaconActive)}
        ${O?`<p class="error small">${a(O)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(r)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(r)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(r)}">Diagnostics →</a>
        </p>
      </div>
    `}function de(){return'<span class="spinner" aria-label="working"></span>'}async function X(u,S){if(I[u]===null){I[u]=S,O=null,x();try{await Hn(r,u,S)}catch(A){O=`${Qe[u]} ${S} failed: ${A instanceof Error?A.message:String(A)}`}I[u]=null,i||x()}}async function me(u,S){const A=await qe(S),D=u.textContent;u.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{i||(u.textContent=D)},1500)}function ge(u){const S=Qe[u],A=f?xe(u==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";ce(`
        <h2>Clear ${a(S)} data</h2>
        <p class="error">
          This stops the ${a(S.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(A)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(u)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,y=>{if(y==="cancel"){Q();return}y==="confirm"&&$(u)});const D=document.getElementById("clear-confirm-input"),V=document.getElementById("clear-confirm-btn");D==null||D.addEventListener("input",()=>{V&&(V.disabled=D.value.trim()!==u)}),D==null||D.focus()}async function $(u){const S=document.getElementById("clear-confirm-btn");S&&(S.disabled=!0,S.textContent="Clearing…");try{await Dn(r,u),Q(),p()}catch(A){const D=Ue();if(D){const V=document.createElement("p");V.className="error small",V.textContent=`Clear failed: ${A instanceof Error?A.message:String(A)}`,D.appendChild(V)}S&&(S.disabled=!1,S.textContent="Clear and resync")}}return()=>{i=!0,t==null||t(),Q()}}const gt=500,vt="valve-node-app.explain-consent";function ua(n,r){let i=!1,t=null;const d=[];n.innerHTML=`
    <h1>Logs: ${a(r)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ue()}</div>
  `;const b=n.querySelector("#logs-body"),w=n.querySelector("#logs-footer");ye(n,E=>{E==="explain"&&M()}),f();async function f(){let E,p;try{const[v,x]=await Promise.all([Te(),Pe()]);E=v.find(_=>_.id===r),p=x}catch(v){if(i)return;b.innerHTML=`<p class="error">Failed to load target: ${a(String(v))}</p>`;return}if(i)return;if(!E){b.innerHTML=`<p class="error">Target "${a(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!E.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const m=p==null?void 0:p.networks.find(v=>v.ChainID===E.wire.ChainID);m&&(w.innerHTML=ue(m.Name,m.LearnURL));try{const v=await Nn(r,200);if(i)return;d.push(...v)}catch(v){if(i)return;b.innerHTML=`<p class="error">Failed to load logs: ${a(String(v))}</p>`;return}N(),t=Bn(r,v=>{i||(d.push(v),d.length>gt&&d.splice(0,d.length-gt),N())})}function N(){const E=d.filter(m=>m.severity==="error"||m.severity==="critical");b.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(q).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${F(String(E.length),E.length?"bad":"neutral")}</h2>
          <div class="log-lines">${E.length?E.slice().reverse().map(q).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const p=b.querySelector(".log-lines");p&&(p.scrollTop=p.scrollHeight)}function q(E){const p=E.severity||"info",m=E.learnUrl?` <a href="${a(E.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(p)}">
        <span class="log-time">${a(new Date(E.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(E.unit)}</span>
        <span class="log-sev">${a(p)}</span>
        <span class="log-text">${a(E.line)}</span>
        ${E.explain?`<div class="log-explain">${a(E.explain)}${m}</div>`:""}
      </div>
    `}async function M(){const E=d.filter(m=>m.severity==="error"||m.severity==="critical").map(m=>m.line).slice(-40);if(!(localStorage.getItem(vt)==="1")){I(E);return}await O(E)}function I(E){const p=E.length?`<pre class="explain-excerpt">${E.map(m=>a(m)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';j(`
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
    `,m=>{m==="proceed"?(localStorage.setItem(vt,"1"),L(),O(E)):L()})}async function O(E){j('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const p=E.length?await mt(r,E):await mt(r);if(i)return;j(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(p.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${p.sentExcerpt.map(m=>a(m)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,m=>{m==="close"&&L()})}catch(p){if(i)return;if(p instanceof we&&p.status===409){j(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,m=>{m==="close"&&L()});return}j(`
        <h2>Explain failed</h2>
        <p class="error">${a(p instanceof Error?p.message:String(p))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,m=>{m==="close"&&L()})}}function j(E,p){L();const m=document.createElement("div");m.className="modal-overlay",m.id="explain-modal",m.innerHTML=`<div class="modal">${E}</div>`,m.addEventListener("click",v=>{const x=v.target.closest("[data-modal-action]");x!=null&&x.dataset.modalAction&&p(x.dataset.modalAction),v.target===m&&p("cancel")}),document.body.appendChild(m)}function L(){var E;(E=document.getElementById("explain-modal"))==null||E.remove()}return()=>{i=!0,t==null||t(),L()}}const pa="run",ha={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},fa={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function ma(n,r){let i=!1,t=null,d=null;const b={devnet:null},w={devnet:null},f={devnet:[]};let N=null;const q={devnet:!1};let M=null;const I={devnet:null},O={devnet:null};n.innerHTML=`
    <div class="page-head">
      <h1>Services: ${a(r)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${ue()}
  `;const j=n.querySelector("#services-body");ye(n,(l,g)=>{ge(l,g)}),L();async function L(){try{const l=await xt(r);if(i)return;t=l,d=null}catch(l){if(i)return;t=null,d=R(l)}p()}function E(l){return t==null?void 0:t.services.find(g=>g.id===l)}function p(){if(!i){if(d){j.innerHTML=`<p class="error">Could not read this machine's services: ${a(d)}</p>`;return}if(!t){j.innerHTML='<p class="muted">Loading…</p>';return}j.innerHTML=`
      ${m(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(v).join("")}
      </div>
    `}}function m(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${a(l.flavor)}${l.serverVersion?` ${a(l.serverVersion)}`:""} · reachable</p>`;const g=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(g)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${a(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${a(l.hint)}</div>`:""}
      </div>
    `}function v(l){const g=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(l.label)}</h2>
          ${x(l)}
        </div>
        <p class="muted small">${a(ha[l.id]??"")}</p>

        ${l.error?_(l):""}
        ${l.blocked?`<div class="banner banner-warn">${a(l.blocked)}</div>`:""}
        ${g.map(B=>`<div class="banner banner-warn">${a(B)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${a(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Z(l)}

        ${H(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(B=>U(l,B)).join("")}
        </div>
        ${w[l.id]?`<p class="error small">${a(w[l.id])}</p>`:""}
        ${ne(l)}

        ${ae(l)}
      </div>
    `}function x(l){switch(l.status.State){case"running":return F("running","ok");case"created-but-stopped":return F("stopped","warn");case"not-created":return F("not created","neutral");default:return F("unknown","bad")}}function _(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(l.error??"")}</div>
        ${l.hint?`<div class="small">${a(l.hint)}</div>`:""}
      </div>
    `}function Z(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const g=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${g}.</p>`}function H(l){const g=l.endpoints??[];return g.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":g.map(B=>`
        <div class="endpoint-row">
          ${ke("ok")}
          <span class="muted small">${a(B.label)}</span>
          <code class="endpoint-url">${a(B.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(B.url)}">Copy</button>
        </div>`).join("")}function U(l,g){const B=fa[g];if(!B)return"";const G=b[l.id],ee=g==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:B.label;return`
      <button class="${B.className}" data-action="svc-${g}" data-svc="${a(l.id)}"
              title="${a(B.title)}" ${G?"disabled":""}>
        ${G===g?'<span class="spinner" aria-label="working"></span>':a(ee)}
      </button>
    `}function ne(l){const g=f[l.id]??[];return g.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(g.join(`
`))}</pre>
      </div>
    `}function ae(l){const g=q[l.id],B=J(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(l.id)}">
            ${g?"Close":"Edit"}
          </button>
        </div>
        ${g?oe():`<p class="small">${B}</p>`}
        ${I[l.id]?`<p class="error small">${a(I[l.id])}</p>`:""}
        ${O[l.id]?`<p class="muted small">${a(O[l.id])}</p>`:""}
      </div>
    `}function J(l){const g=l.devnet;return g?`Chain ${g.ChainID} · a block every ${a(g.BlockTime)} · JSON-RPC on ${a(g.BindAddr)}:${g.HTTPPort} · WebSocket on ${a(g.BindAddr)}:${g.WSPort}`:"—"}function oe(l){return re()}function re(){const l=M;return l?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${a(l.BlockTime)}" autocomplete="off" spellcheck="false" />
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
        <input type="text" id="dev-bind" value="${a(l.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${l.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function de(){q.devnet&&M&&(M.BlockTime=X("#dev-blocktime",M.BlockTime),M.HTTPPort=me("#dev-http",M.HTTPPort),M.WSPort=me("#dev-ws",M.WSPort),M.BindAddr=X("#dev-bind",M.BindAddr))}function X(l,g){const B=n.querySelector(l);return B?B.value.trim():g}function me(l,g){const B=n.querySelector(l);if(!B)return g;const G=Number.parseInt(B.value.trim(),10);return Number.isFinite(G)?G:g}async function ge(l,g){const B=g.dataset.svc??"";switch(l){case"refresh":await L();return;case"copy":g.dataset.copy&&await k(g,g.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await $(B,l.slice(4));return;case"svc-create":case"svc-recreate":await u(B);return;case"svc-wipe":D(B);return;case"toggle-config":S(B);return;case"save-config":await A(B);return;default:return}}async function $(l,g){if(!b[l]){b[l]=g,w[l]=null,p();try{await Fn(r,l,g)}catch(B){w[l]=`${g} failed: ${R(B)}${K(B)}`}b[l]=null,await L()}}async function u(l){if(!b[l]){b[l]="create",w[l]=null,f[l]=["starting…"],p();try{await Wn(r,l)}catch(g){w[l]=`${R(g)}${K(g)}`,f[l]=[],b[l]=null,p();return}N==null||N(),N=Oe(r,g=>{if(i)return;const B=g.err?`${g.stepId}: ${g.err}`:g.line?`${g.stepId}: ${g.line}`:`${g.stepId}: done`;if(f[l]=[...(f[l]??[]).filter(ee=>ee!=="starting…"),B],!!g.err||g.stepId===pa&&!!g.done){N==null||N(),N=null,b[l]=null,g.err&&(w[l]="Provisioning failed — see the log below."),L();return}p()})}}function S(l){if(de(),q[l]=!q[l],I[l]=null,O[l]=null,q[l]){const g=E(l);g!=null&&g.devnet&&(M={...g.devnet})}p()}async function A(l){var G;de(),I[l]=null,O[l]=null;const g=M;if(!g)return;if(g.HTTPPort===g.WSPort){I[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",p();return}try{await Vn(r,l,g)}catch(ee){I[l]=R(ee),p();return}const B=((G=E(l))==null?void 0:G.status.State)==="running";q[l]=!1,O[l]=B?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function D(l){const g=E(l);if(!g)return;const B=(g.restartsOnWipe??[]).map(W=>{var ie;return((ie=E(W))==null?void 0:ie.label)??W});ce(`
        <h2>Wipe ${a(g.label)}</h2>
        <p class="error">This deletes ${a(g.wipeDiscards)}</p>
        ${B.length?`<p>It also restarts what sits in front of it: ${a(B.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${a(l)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(l)}</button>
        </div>
      `,W=>{if(W==="cancel"||W==="close"){Q(),L();return}W==="confirm"&&V(l)});const G=document.getElementById("wipe-confirm-input"),ee=document.getElementById("wipe-confirm-btn");G==null||G.addEventListener("input",()=>{ee&&(ee.disabled=G.value.trim()!==l)}),G==null||G.focus()}async function V(l){const g=document.getElementById("wipe-confirm-btn");g&&(g.disabled=!0,g.textContent="Wiping…");let B;try{B=await _n(r,l)}catch(G){const ee=Ue();if(ee){const W=document.createElement("p");W.className="error small",W.textContent=`Wipe failed: ${R(G)}${K(G)}`,ee.appendChild(W)}g&&(g.disabled=!1,g.textContent=`Wipe ${l}`);return}y(l,B)}function y(l,g){const B=E(l),G=se=>{var Ee;return((Ee=E(se))==null?void 0:Ee.label)??se},ee=[];ee.push(g.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const se of g.report.VolumesRemoved??[])ee.push(`Volume ${se} deleted.`);for(const se of g.report.VolumesAbsent??[])ee.push(`Volume ${se} was already gone.`);g.report.Recreated&&ee.push("Container re-created from your saved configuration.");const W=(g.report.Cascaded??[]).map(G),ie=(g.report.CascadeSkipped??[]).map(G);ce(`
        <h2>${a((B==null?void 0:B.label)??l)} wiped</h2>
        <ul class="plain-list">${ee.map(se=>`<li>${a(se)}</li>`).join("")}</ul>
        ${W.length?`<p class="ok">Restarted in front of it: ${a(W.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${ie.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(ie.join(", "))}.</p>`:""}
        ${g.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(g.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,se=>{(se==="close"||se==="cancel")&&(Q(),L())})}async function k(l,g){const B=await qe(g),G=l.textContent;l.textContent=B?"Copied!":"Copy failed",setTimeout(()=>{i||(l.textContent=G)},1500)}function R(l){return l instanceof Error?l.message:String(l)}function K(l){return l instanceof we&&l.hint?` — ${l.hint}`:""}return()=>{i=!0,N==null||N(),Q()}}const et=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ke=8545,Ve=5052,Ge=30303,ba=[369,943,1],$t={369:"default",943:"practise here first"};function ya(n,r){let i=!1;const t={targetId:r,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${a(r)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ue()}</div>`;const d=n.querySelector("#wizard-body"),b=n.querySelector("#wizard-footer");ye(n,(y,k)=>{me(y,k)}),rt(n,(y,k)=>{y==="exec-select"?t.execId=k:y==="beacon-select"&&(t.beaconId=k),f()}),n.addEventListener("change",y=>{const k=y.target;k instanceof HTMLInputElement&&(k.id==="data-dir-input"?(ge(),U()):k.id==="checkpoint-toggle"?(t.checkpoint=k.checked,f()):k.id==="exec-snapshot-toggle"&&(t.execSnapshot=k.checked,f()))}),w();async function w(){try{const[y,k]=await Promise.all([Pe(),Te()]);if(i)return;t.catalog=y;const R=k.find(K=>K.id===r);R!=null&&R.wire&&(t.chainId=R.wire.ChainID,t.execId=R.wire.ExecID,t.beaconId=R.wire.BeaconID,t.archive=R.wire.Archive,R.wire.ExecHTTPPort&&(t.execHTTPPort=String(R.wire.ExecHTTPPort)),R.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(R.wire.BeaconHTTPPort)),R.wire.ExecP2PPort&&(t.execP2PPort=String(R.wire.ExecP2PPort)),R.wire.RPCBindAddr&&(t.rpcBindAddr=R.wire.RPCBindAddr)),f()}catch(y){if(i)return;t.loadError=String(y instanceof Error?y.message:y),f()}}function f(){if(t.loadError){d.innerHTML=`<p class="error">Failed to load: ${a(t.loadError)}</p>`;return}t.catalog&&(d.innerHTML=`
      ${V(t.step)}
      ${q()}
    `,N())}function N(){var k;const y=(k=t.catalog)==null?void 0:k.networks.find(R=>R.ChainID===t.chainId);b.innerHTML=y?ue(y.Name,y.LearnURL):ue()}function q(){switch(t.step){case"network":return M();case"clients":return I();case"mode":return re();case"review":return de();case"run":return X()}}function M(){const y=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${ba.map(R=>{const K=y.networks.find(B=>B.ChainID===R);if(!K)return"";const l=t.chainId===R,g=$t[R]?F($t[R],R===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${R}" type="button">
          <h3>${a(K.Name)} <span class="muted">(chain ${R})</span></h3>
          ${g}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function I(){const y=t.catalog,k=y.networks.find(l=>l.ChainID===t.chainId);if(!k)return'<p class="error">Unknown network.</p>';(t.execId===null||!k.ExecClients.includes(t.execId))&&(t.execId=k.ExecClients[0]??null),(t.beaconId===null||!k.BeaconClients.includes(t.beaconId))&&(t.beaconId=k.BeaconClients[0]??null);const R=k.ExecClients.map(l=>ae(l,y)),K=k.BeaconClients.map(l=>ae(l,y));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${a(k.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${at("exec-select",R,t.execId)}
        </label>
        ${oe(t.execId,y)}
        <label>
          Beacon client
          ${at("beacon-select",K,t.beaconId)}
        </label>
        ${oe(t.beaconId,y)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function O(y){return y<=0?"—":y>=1?`~${y.toFixed(1)} TB`:`~${Math.round(y*1e3)} GB`}const j=1.1,L=.5,E="Valve reth snapshot",p="rough estimate";function m(y){return y.SnapshotSizeTB}function v(y){return y.SnapshotSizeTB*L}function x(y){return`<p class="muted small">${O(m(y))} is the measured size of Valve's reth snapshot for ${a(y.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function _(y){return{archive:m(y)*1e12*j,full:v(y)*1e12*j}}function Z(y,k){if(!y)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${a(k)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${a(k)}</code>: ${a(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==k)return"";const R=_(y),K=t.freeBytes>=R.archive,l=t.freeBytes>=R.full,g=`<p class="muted small">Free at <code>${a(k)}</code>: <strong>${xe(t.freeBytes)}</strong> — archive ${K?"fits":"won't fit"} (${O(m(y))}, ${E}), full ${l?"fits":"won't fit"} (${O(v(y))}, ${p}).</p>`;let B="";return t.downgradeNote?B=`<p class="banner banner-warn">${a(t.downgradeNote)}</p>`:l||(B=`<p class="banner banner-warn">Neither full (${O(v(y))}, ${p}) nor archive (${O(m(y))}, ${E}) fits the free space here — choose a location with more room.</p>`),g+B}function H(y,k){if(t.downgradeNote=null,!y||t.freeBytes===null)return;const R=_(y);t.archive&&t.freeBytes<R.archive&&t.freeBytes>=R.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${k} for archive (${O(m(y))}, ${E}) — switched to Full (${O(v(y))}, ${p}). Pick a location with more room to run archive.`)}async function U(){var R;if(t.chainId===null)return;const y=(R=t.catalog)==null?void 0:R.networks.find(K=>K.ChainID===t.chainId),k=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,f();try{const{freeBytes:K}=await Rn(t.targetId,k);if(i)return;t.freeBytes=K,t.probedPath=k,H(y,k)}catch(K){if(i)return;t.freeBytes=null,t.probedPath=k,t.diskError=String(K instanceof Error?K.message:K)}t.diskProbing=!1,f()}function ne(y){return y?/^https?:\/\/.+/i.test(y)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ae(y,k){const R=k.clients.find(K=>K.id===y);return{value:y,label:R?`${R.id} — ${J(R.repo)}`:y}}function J(y){const k=y.split("/");return k.length>=4?k[3]:y}function oe(y,k){const R=y?k.clients.find(l=>l.id===y):void 0;if(!R)return"";const K=R.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(R.repo)}" target="_blank" rel="noopener noreferrer">${a(K)}</a></p>`}function re(){var G,ee,W;const y=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",k=(G=t.catalog)==null?void 0:G.networks.find(ie=>ie.ChainID===t.chainId),R=((W=(ee=t.catalog)==null?void 0:ee.clients.find(ie=>ie.id===t.execId))==null?void 0:W.snapshotSupported)??!1,K=k?`${O(v(k))} (${p})`:"Smaller",l=k?`${O(m(k))} (${E})`:"Much larger",g=k?` on ${a(k.Name)}`:"",B=k?t.checkpoint?k.SyncLabel:k.GenesisSyncLabel:"";return`
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
          ${k?`<p class="sync-estimate">⏱ Estimated initial sync${g}: <strong>${a(B)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${a((k==null?void 0:k.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${a((k==null?void 0:k.CheckpointURL)??"")}" value="${a(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${a(t.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${R?`
        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="exec-snapshot-toggle" ${t.execSnapshot?"checked":""} />
            <span><strong>Restore from Valve's execution snapshot</strong> — fast sync (~hours) instead of syncing from genesis (~days).</span>
          </label>
          ${t.execSnapshot?`<label>
                   Snapshot key
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${a(t.snapshotKey)}" />
                 </label>
                 ${t.snapshotKeyError?`<p class="error small">${a(t.snapshotKeyError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${g}</th><td class="yes">${K}</td><td class="limited">${l}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${k?x(k):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${k?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${K}${k?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(y)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(y)}" value="${a(t.dataDir)}" />
          </label>
          ${Z(k,t.dataDir||y)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(y)}/jwt.hex" value="${a(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Ke})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Ke}" value="${a(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${a(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ve})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ve}" value="${a(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${a(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Ge})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Ge}" value="${a(t.execP2PPort)}" />
          </label>
          ${t.execP2PPortError?`<p class="error small">${a(t.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${a(t.rpcBindAddr)}" />
          </label>
          ${t.rpcBindAddrError?`<p class="error small">${a(t.rpcBindAddrError)}</p>`:""}
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
    `}function de(){const k=t.catalog.networks.find(se=>se.ChainID===t.chainId),R=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,K=t.jwtPath||`${R}/jwt.hex`,l=et.map(se=>`<li>${a(se.title)}</li>`).join(""),g=A(t.execHTTPPort,Ke),B=A(t.beaconHTTPPort,Ve),G=A(t.execP2PPort,Ge),ee=g||B||G?`<tr><th>Non-default ports</th><td>${[g?`exec HTTP ${g}`:null,B?`beacon HTTP ${B}`:null,G?`exec p2p ${G}`:null].filter(se=>se!==null).map(a).join(", ")}</td></tr>`:"",{addr:W}=$(t.rpcBindAddr),ie=W?`<tr><th>RPC bind address</th><td><code>${a(W)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${a(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${a((k==null?void 0:k.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${a(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${a(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${a(R)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${a(K)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${a(t.checkpointUrl||(k==null?void 0:k.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${ee}
            ${ie}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${l}</ol>
        ${t.startError?`<p class="error">${a(t.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${t.starting?"disabled":""}>
            ${t.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function X(){const k=t.catalog.networks.find(W=>W.ChainID===t.chainId),R=k==null?void 0:k.LearnURL,K=new Set(t.events.filter(W=>W.done).map(W=>W.stepId)),l=new Set(t.events.filter(W=>W.err).map(W=>W.stepId)),g=new Map;for(const W of t.events){if(!W.line)continue;const ie=g.get(W.stepId)??[];ie.push(W.line),g.set(W.stepId,ie)}const B=et.map(W=>{var _e;const ie=K.has(W.id),se=l.has(W.id),Ee=se?F("failed","bad"):ie?F("done","ok"):F("pending","neutral"),je=(g.get(W.id)??[]).slice(-5),Fe=(_e=t.events.find(Ie=>Ie.stepId===W.id&&Ie.err))==null?void 0:_e.err,Ye=W.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${R?` <a href="${a(R)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${ie?"step-done":""} ${se?"step-error":""}">
          <div class="step-head">${Ee} <strong>${a(W.title)}</strong></div>
          ${Ye}
          ${je.length?`<pre class="step-log">${je.map(Ie=>a(Ie)).join(`
`)}</pre>`:""}
          ${Fe?`<p class="error small">${a(Fe)}</p>`:""}
        </li>
      `}).join(""),G=t.events.some(W=>W.err),ee=et.every(W=>K.has(W.id))||t.events.some(W=>W.stepId==="handshake"&&W.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${B}</ol>
        ${ee&&!G?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${a(t.startError)}</p>`:""}
        ${G?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function me(y,k){switch(y){case"pick-network":t.chainId=Number(k.dataset.chainId),t.execId=null,t.beaconId=null,f();break;case"goto-network":t.step="network",f();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",f();break;case"goto-mode":t.step="mode",f(),U();break;case"goto-review":if(ge(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){f();break}t.step="review",f();break;case"start-setup":D();break}}function ge(){const y=n.querySelectorAll('input[name="mode"]');for(const W of Array.from(y))W.checked&&(t.archive=W.value==="archive");const k=n.querySelector("#data-dir-input"),R=n.querySelector("#jwt-path-input");k&&(t.dataDir=k.value.trim()),R&&(t.jwtPath=R.value.trim());const K=n.querySelector("#exec-http-port-input"),l=n.querySelector("#beacon-http-port-input"),g=n.querySelector("#exec-p2p-port-input");K&&(t.execHTTPPort=K.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),g&&(t.execP2PPort=g.value.trim());const B=n.querySelector("#rpc-bind-addr-input");B&&(t.rpcBindAddr=B.value.trim());const G=n.querySelector("#checkpoint-url-input");G&&(t.checkpointUrl=G.value.trim());const ee=n.querySelector("#snapshot-key-input");ee&&(t.snapshotKey=ee.value.trim()),t.execHTTPPortError=S(t.execHTTPPort).error??null,t.beaconHTTPPortError=S(t.beaconHTTPPort).error??null,t.execP2PPortError=S(t.execP2PPort).error??null,t.rpcBindAddrError=$(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?ne(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function $(y){if(!y)return{};const k=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(y);return k?k.slice(1).every(R=>Number(R)<=255)?{addr:y}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(y)&&y.includes(":")?{addr:y}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const u=/^\d+$/;function S(y){if(!y)return{};if(!u.test(y))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const k=Number(y);return!Number.isInteger(k)||k<1||k>65535?{error:"Port must be between 1 and 65535."}:{port:k}}function A(y,k){const{port:R}=S(y);if(!(R===void 0||R===k))return R}async function D(){var g;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(g=t.streamStop)==null||g.call(t),t.streamStop=null,f();const y={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(y.DataDir=t.dataDir),t.jwtPath&&(y.JWTPath=t.jwtPath);const k=A(t.execHTTPPort,Ke),R=A(t.beaconHTTPPort,Ve),K=A(t.execP2PPort,Ge);k!==void 0&&(y.ExecHTTPPort=k),R!==void 0&&(y.BeaconHTTPPort=R),K!==void 0&&(y.ExecP2PPort=K);const{addr:l}=$(t.rpcBindAddr);l!==void 0&&(y.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(y.CheckpointURL=t.checkpointUrl):y.NoCheckpoint=!0,t.execSnapshot&&(y.ExecSnapshot=!0,y.SnapshotKey=t.snapshotKey);try{await Ln(t.targetId,y)}catch(B){if(!(B instanceof we&&B.status===409)){t.starting=!1,t.startError=String(B instanceof Error?B.message:B),f();return}}t.starting=!1,t.step="run",f(),t.streamStop=Oe(t.targetId,B=>{i||(t.events.push(B),t.step==="run"&&f())})}function V(y){const k=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],K=k.map(l=>l.id).indexOf(y);return`
      <ol class="wizard-progress">
        ${k.map((l,g)=>`<li class="${g===K?"current":g<K?"past":"future"}">${a(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var y;i=!0,(y=t.streamStop)==null||y.call(t)}}function ga(n,r){let i=!1;const t=new Map;n.innerHTML=`<h1>${a(r)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const d=n.querySelector("#machine-body");ye(n,(I,O)=>{I==="toggle-section"&&q(O.dataset.section??"")}),b();async function b(){let I,O;try{const[j,L]=await Promise.all([Te(),Pe()]);I=j.find(E=>E.id===r),O=L}catch(j){if(i)return;d.innerHTML=`<p class="error">Failed to load machine: ${a(String(j))}</p>`;return}if(!i){if(!I){location.hash="#/targets";return}w(I,O)}}function w(I,O){const j=I.mode==="local"?"this machine":"SSH",L=I.mode==="ssh"&&I.ssh?`${a(I.ssh.User)}@${a(I.ssh.Host)}`:j;d.innerHTML=`
      <p class="muted">${L}</p>
      <p>${f(I,O)}</p>
      <div class="machine-sections">
        ${M.map(E=>N(E,I,O)).join("")}
      </div>
      ${ue()}
    `}function f(I,O){const j=I.wire;if(!j)return F("not set up","neutral");const L=O.networks.find(p=>p.ChainID===j.ChainID),E=L?L.Name:`chain ${j.ChainID}`;return`${F(E,"ok")} ${F(j.ExecID,"neutral")} ${F(j.BeaconID,"neutral")}${j.Archive?" "+F("archive","warn"):""}`}function N(I,O,j){return`
      <section class="card machine-section" data-section-card="${a(I.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(I.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(I.title)}</span>
          <span class="machine-section-status">${I.status(O,j)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(I.key)}" hidden></div>
      </section>
    `}function q(I){const O=M.find(m=>m.key===I);if(!O)return;const j=n.querySelector(`[data-section-card="${I}"]`),L=n.querySelector(`[data-section-body="${I}"]`),E=n.querySelector(`.machine-section-head[data-section="${I}"]`);if(!j||!L||!E)return;const p=L.hidden;if(p&&!t.has(I)){const m=document.createElement("div");L.appendChild(m),t.set(I,O.mount(m))}L.hidden=!p,j.classList.toggle("open",p),E.setAttribute("aria-expanded",String(p))}const M=[{key:"setup",title:"Setup",status:I=>I.wire?F("set up","ok"):F("not set up","neutral"),mount:I=>ya(I,r)},{key:"dashboard",title:"Dashboard",status:I=>I.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>da(I,r)},{key:"logs",title:"Logs",status:I=>I.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>ua(I,r)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:I=>ma(I,r)}];return()=>{i=!0;for(const I of t.values())try{I()}catch{}t.clear()}}function At(n){var t;if(!n)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const r=n.actions??[];if(n.blocked)return{tone:"blocked",label:"Unavailable",sub:n.blocked,actions:r,blocked:n.blocked};const i=((t=n.networks)==null?void 0:t.length)??0;return n.status.State==="running"?{tone:"on",label:"Running",sub:`${i} network${i===1?"":"s"} served`,actions:r}:{tone:"off",label:"Stopped",sub:i?`${i} network${i===1?"":"s"} configured`:"Press to start",actions:r}}function va(n){if(!n.running)return"off";if(!n.serviceable)return"frequent";const r=n.slowRate??0;return r>.4?"frequent":r>=.1?"occasional":"stable"}const $a=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function wa(n){return $a.map(({key:r,label:i,hot:t})=>{const d=n[r]==="supported";return{key:r,label:i,lit:d,hot:!!t&&d}})}function ka(n,r){if(n.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const i=n.filter(f=>!f.wire);if(i.length>0){const f=i.map(q=>q.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const t=r.networks??[],d=f=>{const N=t.find(q=>q.ChainID===f);return N?N.Name:`chain ${f}`},b=Ca(n.map(f=>d(f.wire.ChainID))),w=n.length===1?"machine":"machines";return{level:"ok",sentence:`All ${n.length} ${w} healthy — ${Sa(b)}.`,machines:[]}}function Ta(n,r){const i=r.machines.length?` <span class="verdict-machines">${r.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${a(t)}</a>`).join(" ")}</span>`:"";n.innerHTML=`
    <div class="verdict-line verdict-${r.level}">
      ${F(r.level==="ok"?"OK":"Attention",r.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(r.sentence)}</strong>${i}
    </div>
  `}function Ca(n){return[...new Set(n)]}function Sa(n){return n.length<=1?n[0]??"":n.length===2?`${n[0]} and ${n[1]}`:`${n.slice(0,-1).join(", ")} and ${n[n.length-1]}`}const xa=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function wt(n){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:n,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Pa=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`,He=n=>`<svg class="p-i"><use href="#p-${n}"/></svg>`,kt="run";function Tt(n){let r=null,i={name:"list"},t=null,d=null,b=null,w=null,f=[];n.innerHTML=Pa+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const N=n.querySelector("#p-card");async function q(){try{const m=await ot();r=Ea(m.gateways),t=null}catch(m){t=ve(m)}M()}function M(){N.innerHTML=I()}function I(){return t?Ia(t):i.name==="network"?Oa(r,i.chainId):i.name==="endpoint"?qa(r,i.chainId,i.upstreamId):Ra(r,d,b,f)}ye(N,(m,v)=>{O(m,v)});async function O(m,v){if(m==="setup"){if(d)return;await E();return}if(m==="power"){if(!r||d)return;const x=At(r);if(x.tone==="blocked")return;if(r.status.State==="running"&&x.actions.includes("stop")){await j(r.id,"stop");return}if(x.actions.includes("start")){await j(r.id,"start");return}if(x.actions.includes("create")){await L(r.id);return}return}if(m==="open-network"){i={name:"network",chainId:Number(v.dataset.chainId)},M();return}if(m==="back-to-list"){i={name:"list"},M();return}if(m!=="add-network")switch(m){case"gw-start":case"gw-stop":case"gw-restart":r&&!d&&await j(r.id,m.slice(3));return;case"gw-create":case"gw-recreate":r&&!d&&await L(r.id);return;case"gw-wipe":r&&!d&&await p(r);return;default:return}}async function j(m,v){if(!d){d=v,b=null,M();try{await It(m,v)}catch(x){b=`${v} failed: ${ve(x)}`}d=null,await q()}}async function L(m){if(d)return;d="create",b=null,M();let v;try{v=await nt(m)}catch(x){b=ve(x),d=null,M();return}w==null||w(),w=Oe(v.targetId,x=>{(x.err||x.stepId===kt&&x.done)&&(w==null||w(),w=null,d=null,x.err&&(b=`Provisioning failed: ${x.err}`),q())})}async function E(){if(d)return;d="setup",b=null,f=[],M();const m=H=>{f=[...f,H],M()},v=(H,U)=>{d=null,b=U?`${H} — ${U}`:H,M()};m("Preparing your endpoint…");try{(await Te()).some(U=>U.id==="local")||await tt({id:"local",mode:"local"})}catch(H){v(`Could not register this machine: ${ve(H)}`,Ae(H));return}try{const H=await xt("local");if(!H.docker.reachable){v(H.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",H.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(H){v(`Could not check Docker on this machine: ${ve(H)}`,Ae(H));return}m("Creating the gateway…");let x="default";try{x=(await Pt({id:x,placement:{targetId:"local",backend:"docker"},config:wt([])})).id}catch(H){v(`Could not create the gateway: ${ve(H)}`,Ae(H));return}m("Adding Ethereum and PulseChain endpoints…");const _=[];for(const{chainId:H}of xa)try{const ne=((await Lt(x,H)).endpoints??[]).filter(ae=>!ae.alreadyAdded).map(ae=>ae.url);if(ne.length===0)continue;_.push({ChainID:H,Upstreams:ne.map((ae,J)=>({ID:`public-${H}-${J+1}`,Kind:"external",Endpoint:ae,Local:!1,RecentOnly:!1}))})}catch(U){v(`Could not read valve's set for chain ${H}: ${ve(U)}`,Ae(U));return}if(_.length===0){v("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await Et(x,wt(_))}catch(H){v(`Could not save the endpoints: ${ve(H)}`,Ae(H));return}m("Starting the gateway… the first run pulls the eRPC and Caddy images.");let Z;try{Z=await nt(x)}catch(H){v(`Could not start the gateway: ${ve(H)}`,Ae(H));return}w==null||w(),w=Oe(Z.targetId,H=>{const U=H.err?`${H.stepId}: ${H.err}`:H.line?`${H.stepId}: ${H.line}`:`${H.stepId}: done`;if(m(U),!!H.err||H.stepId===kt&&!!H.done){if(w==null||w(),w=null,d=null,H.err){b=`Provisioning failed: ${H.err}`,M();return}f=[],q()}})}async function p(m){if(await Be({title:`Wipe ${m.label}`,body:`This destroys ${m.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){d="wipe",b=null,M();try{const x=await Rt(m.id);x.error&&(b=x.error)}catch(x){b=`wipe failed: ${ve(x)}`}d=null,await q()}}return q(),()=>{w==null||w()}}function Ea(n){return!n||n.length===0?null:n.find(r=>r.placement.targetId==="local")??n[0]}function ve(n){return n instanceof Error?n.message:String(n)}function Ae(n){return n instanceof we?n.hint:void 0}function Ia(n){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(n)}</div>`}function Ra(n,r,i,t){var w;if(n===null)return La(r,i,t);const d=At(n),b=(w=n==null?void 0:n.networks)!=null&&w.length?n.networks.map((f,N)=>Ua(n,f,N>0)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${a(d.sub)}</span>
    </div>
    <div class="p-band">
      ${Ba(n,d,r,i)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${b}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${He("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function La(n,r,i){const t=n==="setup",d=r?`<div class="p-emptyerr">${a(r)}</div>`:"",b=i.length?`<div class="p-setup-log" aria-live="polite">${i.map(w=>`<div>${a(w)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${t?" disabled":""}>
        <div class="p-pbtn off big${t?" busy":""}">${He("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${d}
      ${b}
    </div>
  `}function Aa(n,r){return r.tone==="blocked"?null:n.status.State==="running"&&r.actions.includes("stop")?"stop":r.actions.includes("start")?"start":r.actions.includes("create")?"create":null}const Na={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},Ct={restart:"refresh",recreate:"refresh",wipe:"trash"};function Ba(n,r,i,t){const d=r.tone==="blocked"?r.blocked??"":r.sub,b=i?" busy":"",w=t?`<div class="p-ps" style="color:var(--red)">${a(t)}</div>`:"",f=r.tone==="blocked"&&(n!=null&&n.hint)?`<div class="p-ps">${a(n.hint)}</div>`:"",N=`
    <div class="p-power${b}" data-action="power">
      <div class="p-pbtn ${r.tone}">${He("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(r.label)}</div>
        <div class="p-ps"${r.tone==="blocked"?' style="color:var(--red)"':""}>${a(d)}</div>
        ${f}
        ${w}
      </div>
    </div>
  `,q=n?Ha(n,r,i):"";return N+q}function Ha(n,r,i){const t=Aa(n,r),d=(n.actions??[]).filter(w=>w!==t);return d.length===0?"":`<div class="p-chips">${d.map(w=>{const f=Na[w]??w,N=Ct[w]?He(Ct[w]):"";return`<button type="button" class="p-chip${w==="wipe"?" danger":""}" data-action="gw-${w}" data-gid="${a(n.id)}"${i?" disabled":""}>${N}${a(f)}</button>`}).join("")}</div>`}const Da={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function Ma(n){return n.map(r=>`<svg class="p-i${r.hot?" hot":r.lit?" on":""}"><use href="#p-${Da[r.key]}"/></svg>`).join("")}function Ua(n,r,i){const t=va({running:n.status.State==="running",serviceable:r.serviceable}),d=wa({});return`
    <div class="p-row${i?" p-rowdiv":""}" data-action="open-network" data-chain-id="${r.chainId}">
      <span class="p-lead"><span class="p-dot ${t}"></span></span>
      <span class="p-nm">${a(r.name)}</span>
      <span class="p-caps">${Ma(d)}</span>
      <span class="p-chev">${He("chevR")}</span>
    </div>
  `}function Oa(n,r){var t;const i=(t=n==null?void 0:n.networks)==null?void 0:t.find(d=>d.chainId===r);return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-list">${He("chevL")}</span>
      <span class="p-dtitle"><span class="p-nmtxt">${a((i==null?void 0:i.name)??`Chain ${r}`)}</span></span>
    </div>
    <div class="p-band" style="padding:16px;color:var(--dim)">Network detail is coming soon.</div>
  `}function qa(n,r,i){return""}function ja(n,r){let i=!1,t=[],d=null,b=!1,w=!1;n.innerHTML=`<h1>Security: ${a(r)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ue()}</div>`;const f=n.querySelector("#sec-body"),N=n.querySelector("#sec-footer");ye(n,(L,E)=>{var p;if(L==="rerun")M();else if(L==="toggle")(p=E.closest(".check-item"))==null||p.classList.toggle("expanded");else if(L==="copy"){const m=E.dataset.copy;m&&j(E,m)}}),q();async function q(){let L,E;try{const[m,v]=await Promise.all([Te(),Pe()]);L=m.find(x=>x.id===r),E=v}catch(m){if(i)return;f.innerHTML=`<p class="error">Failed to load target: ${a(String(m))}</p>`;return}if(i)return;if(!L){f.innerHTML=`<p class="error">Target "${a(r)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(r)}">Run the setup wizard →</a></p>`;return}const p=E==null?void 0:E.networks.find(m=>m.ChainID===L.wire.ChainID);p&&(N.innerHTML=ue(p.Name,p.LearnURL)),await M()}async function M(){b=!0,d=null,I();try{t=await On(r),w=!0}catch(L){d=String(L instanceof Error?L.message:L)}b=!1,i||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(r)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${b?"disabled":""}>${b?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${!w&&b?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(O).join("")}</ul>`:w?'<p class="muted">No checks returned.</p>':""}
    `}function O(L){const E=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${F(L.Status,E)}
          <strong>${a(L.Title)}</strong>
          <span class="muted small check-detail-inline">${a(L.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${a(L.Why)}</p>
          </details>
          ${L.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(L.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(L.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function j(L,E){const p=await qe(E),m=L.textContent;L.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{i||(L.textContent=m)},1500)}return()=>{i=!0}}const Fa=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],st="VALVE_API_KEY";function _a(n){return n===st?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(n)}}</code> slot wherever an endpoint URL carries one.`}function Wa(n){let r=!1,i=!1,t=!1,d=null,b=!1,w=null,f=null;const N=new Set,q=new Map;let M="",I="";n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ue()}`;const O=n.querySelector("#settings-body");ye(n,(v,x)=>{if(v==="save"&&m(),v==="clear-key"){if(!w)return;i=!0;const _=n.querySelector("#ai-key");_&&(_.value=""),p(w)}if(v==="clear-provider-key"){const _=x.dataset.key;if(!w||!_)return;N.add(_),q.set(_,""),b=!1,p(w)}}),rt(n,(v,x)=>{v!=="ai-provider"||!w||(f=x,b=!1,p(w))}),j();async function j(){try{const v=await ta();if(r)return;w=v,p(v)}catch(v){if(r)return;O.innerHTML=`<p class="error">Failed to load settings: ${a(String(v))}</p>`}}function L(v){const _=(Array.isArray(v.providerKeysSet)?v.providerKeysSet:[]).filter(Z=>Z!==st).sort();return[st,..._]}function E(v,x){const _=a(v);return`
      <div class="pk-row">
        <label>
          <code>${_}</code>
          <input class="provider-key" data-key="${_}" type="password" autocomplete="off"
                 placeholder="${x?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${_a(v)}</p>
        ${x?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${_}">Clear saved key</button>`:""}
      </div>`}function p(v){var ae;const x=f??v.aiProvider,_=Array.isArray(v.providerKeysSet)?v.providerKeysSet:[],Z=L(v).map(J=>E(J,_.includes(J))).join("");O.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${at("ai-provider",Fa.map(J=>({value:J.value,label:J.label})),x)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${v.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${v.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
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
                     placeholder="INFURA_API_KEY" value="${a(M)}" />
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
            <input id="ref-rpc-base" type="text" value="${a(v.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${a(d)}</p>`:""}
        ${b?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const H=n.querySelector("#ai-key");H==null||H.addEventListener("input",()=>{i=!0,b=!1}),(ae=n.querySelector("#ref-rpc-base"))==null||ae.addEventListener("input",()=>{b=!1}),n.querySelectorAll("input.provider-key").forEach(J=>{const oe=J.dataset.key;if(!oe)return;const re=q.get(oe);re!==void 0&&(J.value=re),J.addEventListener("input",()=>{N.add(oe),q.set(oe,J.value),b=!1})});const U=n.querySelector("#pk-new-value");U&&(U.value=I),U==null||U.addEventListener("input",()=>{I=U.value,b=!1});const ne=n.querySelector("#pk-new-name");ne==null||ne.addEventListener("input",()=>{M=ne.value,b=!1})}async function m(){const v=n.querySelector("#ai-key"),x=n.querySelector("#ref-rpc-base");if(!v||!x||!w)return;const _={aiProvider:f??w.aiProvider,refRpcBase:x.value.trim()};i&&(_.aiKey=v.value);const Z={};for(const U of N)Z[U]=q.get(U)??"";const H=M.trim();H&&(Z[H]=I),Object.keys(Z).length>0&&(_.providerKeys=Z),t=!0,d=null,b=!1,p(w);try{const U=await na(_);if(r)return;w=U,i=!1,N.clear(),q.clear(),M="",I="",t=!1,b=!0,p(U)}catch(U){if(r)return;t=!1,d=String(U instanceof Error?U.message:U),p(w)}}return()=>{r=!0}}const Ka=["http","ws","archive","trace"],Va={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Ne=1337,Ga="run",za={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Ja(n){let r=!1,i=null,t=null;const d={},b={},w={},f={},N={},q={},M={},I={},O={},j={},L={},E={},p={},m={},v={};let x="",_=null;n.innerHTML=`
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
    ${ue()}
  `;const Z=n.querySelector("#rpc-body");ye(n,(e,s)=>{Qt(e,s)}),rt(n,()=>{}),U(),H();async function H(){try{const e=await St();if(r)return;x=e.os,X()}catch{}}async function U(){try{const e=await ot();if(r)return;i=e,t=null}catch(e){if(r)return;i=null,t=pe(e)}X();for(const e of(i==null?void 0:i.gateways)??[])ne(e.id),ae(e.id,!1)}async function ne(e){try{const s=await Jn(e);if(r)return;d[e]=s}catch{if(r)return;d[e]=null}X()}async function ae(e,s){w[e]=s,s&&X();try{const o=await Zn(e,s);if(r)return;b[e]=o}catch{if(r)return;b[e]=null}w[e]=!1,X()}function J(e){return((i==null?void 0:i.gateways)??[]).find(s=>s.id===e)}function oe(e,s){return(e.networks??[]).find(o=>o.chainId===s)}function re(e,s,o){var h;const c=(((h=d[e])==null?void 0:h.networks)??[]).find(C=>C.chainId===s);return((c==null?void 0:c.upstreams)??[]).find(C=>C.upstream===o)}function de(e,s,o){var c;return(((c=b[e])==null?void 0:c.endpoints)??[]).find(h=>h.chainId===s&&h.upstream===o)}function X(){if(r)return;if(t){Z.innerHTML=`<p class="error">Could not read the gateways: ${a(t)}</p>`;return}if(!i){Z.innerHTML='<p class="muted">Loading…</p>';return}const e=i.gateways??[],s=e.length>1,o=(i.targets??[]).some(C=>ft(C.id,e)),c=new Set(e.map(C=>C.placement.targetId)),h=(i.orphans??[]).filter(C=>!c.has(C.targetId));Z.innerHTML=`
      ${e.map(C=>$(C,s)).join("")}
      ${e.length===0?ge():""}
      ${h.map(me).join("")}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function me(e){const s=`docker rm -f ${e.containerName}`,o=p[e.containerName];return`
      <div class="strip">
        ${B({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:s})}
        ${o?B({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(e.containerName)}">Dismiss this record</button>
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
    `}function $(e,s){return`
      ${s?`<h2 class="rpc-machine">${a(e.placement.targetId)}</h2>`:""}
      ${u(e)}
      ${g(e)}
      ${se(e)}
      ${Ee(e)}
      ${y(e)}
    `}function u(e){const s=e.status.State==="running",o=e.tls,c=[`on <strong>${a(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${a(e.status.Image)}</code>`),c.push(o!=null&&o.enabled?`HTTPS front <code>${a(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${W(e)}
        <strong>${a(e.label)}</strong>
        ${ee(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${s?`base <code>${a(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function S(e){const s=e.tls;return s!=null&&s.enabled&&s.rootCaPath&&s.effectiveCertSource==="internal"?s.rootCaPath:null}function A(e){var s;return((s=((i==null?void 0:i.targets)??[]).find(o=>o.id===e.placement.targetId))==null?void 0:s.mode)??""}function D(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function V(e,s,o){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${s}"`;case"windows":return`certutil -addstore -f ROOT "${s}"`;case"linux":default:return`sudo cp "${s}" /usr/local/share/ca-certificates/valve-node-app-${o}.crt && sudo update-ca-certificates`}}function y(e){const s=O[e.id]??!1,o=((i==null?void 0:i.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${s?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(e.id)}" aria-expanded="${s}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${k(e,o.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${s?R(e,o):""}
      </section>
    `}function k(e,s){const o=[];return e.status.State!=="running"&&o.push("gateway not running"),s>0&&o.push(`${s} leftover container${s===1?"":"s"}`),o.length===0?"container, settings, certificate":o.join(" · ")}function R(e,s){var o;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>ie(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${a(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${w[e.id]?"disabled":""}>
            ${w[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${a(e.id)}">
            ${M[e.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${a(e.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${e.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${a(e.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${a(((o=(e.networks??[])[0])==null?void 0:o.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${a(e.baseUrl)}</code> once it is running.</p>`}
        ${K(e)}
        ${s.map(me).join("")}
        ${M[e.id]?Vt(e):""}
      </div>
    `}function K(e){const s=S(e);if(!s)return"";const o=A(e)==="local",c=V(x,s,e.id),h=v[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(s)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(s)}">Copy path</button>
        </div>
        ${o?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${m[e.id]?"disabled":""}>
                   ${m[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${h?l(h):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${o?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${x?` (${a(D(x))})`:""}:</span>
          <code class="strip-cmd">${a(c)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(c)}">Copy command</button>
        </div>
      </div>
    `}function l(e){return e.ok?`<div class="strip-line strip-note"><span class="strip-text">${a(e.message)}</span></div>`:`
      <div class="strip-line strip-warn">
        <span class="strip-text">${a(e.message)}</span>
        ${e.ranCommand?`<code class="strip-cmd">${a(e.ranCommand)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.ranCommand)}">Copy</button>`:""}
      </div>
    `}function g(e){const s=[];e.error&&s.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&s.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])s.push({tone:"warn",text:c});s.push(...G(e));const o=N[e.id];return o&&s.push({tone:"bad",text:o}),s.length===0?"":`<div class="strip">${s.map(B).join("")}</div>`}function B(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${a(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${a(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.cmd)}">Copy</button>`:""}
      </div>
    `}function G(e){var h,C;const s=e.tls;if(!(s!=null&&s.enabled))return[];const o=[];s.fallback&&o.push({tone:"warn",text:s.fallback}),s.error?o.push({tone:"warn",text:`HTTPS front: ${s.error}`}):((h=s.status)==null?void 0:h.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((C=s.status)==null?void 0:C.State)??"unknown"}, so nothing answers on ${s.url??"its https URL"} even if the gateway itself is up.`,cmd:s.containerName?`docker start ${s.containerName}`:void 0});const c=j[e.id]??s.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&o.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&o.push({tone:"warn",text:c.expiryWarning}),o}function ee(e){switch(e.status.State){case"running":return F("running","ok");case"created-but-stopped":return F("stopped","warn");case"not-created":return F("not created","neutral");default:return F("unknown","bad")}}function W(e){return e.status.State==="running"?ke("ok"):e.status.State==="unknown"?ke("bad"):ke("neutral")}function ie(e,s){const o=za[s];if(!o)return"";const c=f[e.id];return`
      <button class="${o.className}" data-action="gw-${s}" data-gid="${a(e.id)}"
              title="${a(o.title)}" ${c?"disabled":""}>
        ${c===s?'<span class="spinner" aria-label="working"></span>':a(o.label)}
      </button>
    `}function se(e){const s=q[e.id]??[];return s.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(e.placement.targetId)}</p>
        <pre class="step-log">${a(s.join(`
`))}</pre>
      </div>
    `}function Ee(e){const s=je(e.networks??[]),o=s.some(c=>c.chainId===Ne);return s.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(e.id)}">Add a network</button>
            ${it(e,o)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${s.map(c=>Fe(e,c)).join("")}
        </div>
        ${Ie(e,o)}
        ${Kt(e)}
      </div>
    `}function je(e){const s=e.filter(c=>c.chainId!==Ne),o=e.filter(c=>c.chainId===Ne);return[...s,...o]}function Fe(e,s){const o=Ht(s),c=s.chainId===Ne,h=`${e.id}:${s.chainId}`,C=I[h]??!1,P=o.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${o.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(s.name)}</span>
          <code class="chain-key">evm:${s.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${F(P,o.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(h)}" aria-expanded="${C}">
              ${C?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Ye(e,s)}
        ${C?_e(e,s,o):""}
      </section>
    `}function Ye(e,s){if(!s.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const o=S(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(s.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(s.url)}"
                title="Copy ${a(s.url)}">Copy URL</button>
        ${o?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${A(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${m[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${m[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(o)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${v[e.id]?`<span class="chain-cert muted small">${a(v[e.id].ok?"Trusted — reload your wallet or browser.":v[e.id].message)}</span>`:""}`:""}
      </div>
    `}function _e(e,s,o){const c=s.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${o.why?" chain-verdict-why":""}"${o.why?` title="${a(o.why)}"`:""}>${o.html}</p>
        <div class="chain-detail-bar">
          ${Bt(c.length,o.tone,s.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(e.id)}" data-chain="${s.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(e.id)}" data-chain="${s.chainId}">Remove</button>
        </div>
        ${Ut(e,s)}
        ${(s.warnings??[]).map(h=>`<p class="chain-note">${a(h)}</p>`).join("")}
      </div>
    `}function Ie(e,s){const o=b[e.id],c=o!=null&&o.at?`probed ${a(lt(o.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(e.id)}">+ Network</button>
        ${it(e,s)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${w[e.id]?"disabled":""}>
          ${w[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function it(e,s){return s?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(e.id)}"
                    title="Add a throwaway local test chain (evm:${Ne}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Bt(e,s,o){const c=o>0,h=c?o:e,C=Math.min(e,h);let P="";for(let Le=0;Le<h;Le++)P+=`<span class="seg${Le<C?` seg-on seg-${s}`:""}"></span>`;const T=c&&e>o,z=c?T?`${e} (set is ${o})`:`${e} of ${o}`:`${e}`,te=`${e} upstream${e===1?"":"s"} configured`,he=c?`${te}${T?`, ${e-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${te}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(he)}">${P}</span>
      <span class="segs-n">${z}</span>
    `}function Ht(e){const s=e.upstreams??[];if(s.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!s.some(Dt)){const c=Mt(s);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(C=>`<code>${a(C)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(s.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!s.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=s.filter(c=>!!c.problem);if(o.length>0){const c=s.length-o.length;return{tone:"warn",html:`${o.length} of these ${s.length} endpoints ${o.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${s.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function Dt(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Mt(e){const s=new Set;for(const o of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());c&&s.add(c[1].toLowerCase())}return[...s].sort()}function Ut(e,s){const o=s.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(c=>Ot(e,s,c)).join("")}</ul>`}function Ot(e,s,o){const c=`${e.id}|${s.chainId}|${o.id}`,h=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?ke("bad"):ke("ok")}
          <span class="up-label">${a(o.label)}</span>
          ${qt(o)}
        </div>
        <code class="up-url">${a(o.endpoint||"—")}</code>
        <div class="up-caps">${jt(e,s,o)}</div>
        <div class="up-share">${Wt(e,s,o)}</div>
        <div class="up-acts">
          ${h.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${a(c)}"
                         data-target="${a(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[e.id]?"disabled":""}>
                   ${f[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${a(c)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${a(o.problem)}</div>`:""}
      </li>
    `}function qt(e){return e.problem?F("unusable","bad"):e.recentOnly?F("recent blocks","warn"):e.local?F("yours","ok"):F("public","neutral")}function ct(e,s){var o;if(e)return s==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(c=>c.key===s))==null?void 0:o.status}function jt(e,s,o){const c=de(e.id,s.chainId,o.id);return c?c.unprobeable?`<span class="caps-none" title="${a(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Ka.map(h=>Ft(e,s,c,h)).join("")}</span>`:`<span class="muted small">${b[e.id]===void 0?"probing…":"—"}</span>`}function Ft(e,s,o,c){const h=(o.capabilities??[]).find(te=>te.key===c),C=ct(o,c)??"inconclusive",P=Va[c]??c.toUpperCase();let T="cap";C==="unsupported"?T=_t(e,s,c)?"cap missing":"cap off":C==="inconclusive"?T="cap unknown":C==="inconsistent"&&(T="cap mixed");const z=h!=null&&h.detail?`${h.label}: ${h.detail}`:c==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${P}: no verdict`;return`<span class="${T}" title="${a(z)}">${a(P)}</span>`}function _t(e,s,o){const c=(s.upstreams??[]).map(h=>de(e.id,s.chainId,h.id)).filter(h=>!!h&&!h.unprobeable);return c.length>0&&c.every(h=>ct(h,o)==="unsupported")}function Wt(e,s,o){const c=d[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const h=re(e.id,s.chainId,o.id),C=(c.networks??[]).find(he=>he.chainId===s.chainId);if(!h||!C||C.attributed===0)return'<span class="muted small">no traffic yet</span>';const P=Math.round(h.actual*100),T=Math.round(h.intended*100),z=h.diverged?o.local?"warn":"":"ok",te=`${h.succeeded.toLocaleString()} of ${C.attributed.toLocaleString()} answered requests · routing intends ${T}%`+(h.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(te)}">
        <span class="bar">
          <span class="fill${z?" "+z:""}" style="width:${P}%"></span>
          <span class="tick" style="left:${T}%"></span>
        </span>
        <span class="share-n${h.diverged?" warn":""}">${P}%</span>
        ${h.unconfigured?F("not in config","warn"):""}
      </span>
    `}function Kt(e){const s=d[e.id];return s?s.enabled?s.error?`<p class="muted small">The request counters could not be read: ${a(s.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${s.since?` (${a(lt(s.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function lt(e){const s=new Date(e);return Number.isNaN(s.getTime())?e:s.toLocaleString()}function Vt(e){const s=e.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${a(e.id)}-port" value="${s.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${a(e.id)}-bind" value="${a(s.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${a(s.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${Gt(e)}
        ${zt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(e.id)}">Save settings</button>
        </div>
      </div>
    `}function Gt(e){const s=!e.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${a(e.id)}-metrics" ${s?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function zt(e){var P;const s=a(e.id),o=e.config.TLS??null,c=(o==null?void 0:o.Enabled)??!1,h=(o==null?void 0:o.CertSource)||"internal",C=((P=e.tls)==null?void 0:P.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${s}-tls-host" value="${a((o==null?void 0:o.Hostname)??C)}"
               placeholder="${a(C||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${C?`<p class="muted small">
               The default is <code>${a(C)}</code>. That whole domain's wildcard resolves to
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
        <input type="text" id="gw-${s}-tls-cert" value="${a((o==null?void 0:o.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${s}-tls-key" value="${a((o==null?void 0:o.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${Jt(e)}
    `}function Jt(e){var P,T;const s=a(e.id),o=((P=e.config.TLS)==null?void 0:P.Enabled)??!1,c=j[e.id]??((T=e.tls)==null?void 0:T.verification)??null,h=L[e.id]??!1,C=E[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${s}" ${o&&!h?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${h?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${C?`<p class="error small">${a(C)}</p>`:""}
      ${c?Yt(c):""}
    `}function Yt(e){const s=(e.assertions??[]).map(o=>`
          <li class="small">
            ${Zt(o.status)}
            <strong>${a(o.title)}</strong>
            <div class="muted">${a(o.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${e.ok?e.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${a(e.summary)}
      </div>
      <ul class="verify-list">${s}</ul>
      <p class="muted small">
        Checked ${a(new Date(e.at).toLocaleString())} against <code>${a(e.address)}</code>
        ${e.notAfter?`· certificate valid until <code>${a(new Date(e.notAfter).toLocaleString())}</code> (${a(e.expiresIn??"")})`:""}
      </p>
      ${e.expiryWarning?`<div class="banner banner-warn">${a(e.expiryWarning)}</div>`:""}
    `}function Zt(e){switch(e){case"pass":return F("pass","ok");case"fail":return F("fail","bad");case"unavailable":return F("unavailable","warn");default:return F("skipped","neutral")}}async function Xt(e){L[e]=!0,E[e]=null,X();try{j[e]=await zn(e)}catch(s){E[e]=`${pe(s)}${Re(s)}`}finally{L[e]=!1,X()}}function Ce(e){return{...e.config,Networks:(e.config.Networks??[]).map(s=>({ChainID:s.ChainID,Upstreams:s.Upstreams.map(o=>({...o}))}))}}async function Se(e,s,o){N[e]=null;try{await Et(e,s)}catch(c){return N[e]=`${o?o+": ":""}${pe(c)}`,X(),!1}return await U(),!0}async function Qt(e,s){const o=s.dataset.gid??"";switch(e){case"refresh":await U();return;case"copy":s.dataset.copy&&await Sn(s,s.dataset.copy);return;case"reprobe":await ae(o,!0);return;case"toggle-settings":M[o]=!M[o],X();return;case"toggle-manage":O[o]=!O[o],X();return;case"toggle-chain-detail":{const c=s.dataset.key??"";c&&(I[c]=!I[c]),X();return}case"save-settings":await en(o);return;case"verify-tls":await Xt(o);return;case"trust-cert":await an(o);return;case"gw-start":case"gw-stop":case"gw-restart":await sn(o,e.slice(3));return;case"gw-create":case"gw-recreate":await on(o);return;case"gw-wipe":wn(o);return;case"add-gateway":Tn();return;case"forget-gateway":await rn(o);return;case"dismiss-orphan":await cn(s.dataset.name??"");return;case"add-chain":ln(o);return;case"add-devnet":{const c=J(o);if(c){const h=((i==null?void 0:i.targets)??[]).some(C=>C.id===c.placement.targetId&&C.hasDevnet);ut(o,Ne,h)}return}case"remove-chain":await pn(o,Number.parseInt(s.dataset.chain??"",10));return;case"add-endpoint":ht(o,Number.parseInt(s.dataset.chain??"",10));return;case"remove-endpoint":await hn(s.dataset.key??"");return;case"reset-devnet":await vn(s.dataset.key??"",s.dataset.target??"");return;default:return}}async function en(e){const s=J(e);if(!s)return;const o=Ce(s),c=n.querySelector(`#gw-${CSS.escape(e)}-port`),h=n.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const T=Number.parseInt(c.value.trim(),10);Number.isFinite(T)&&(o.Port=T)}h&&(o.BindAddr=h.value.trim());const C=n.querySelector(`#gw-${CSS.escape(e)}-metrics`);C&&(o.MetricsOff=!C.checked),o.TLS=tn(e,s);const P=s.status.State==="running";await Se(e,o,"Saving settings")&&(M[e]=!1,P&&(N[e]=null,nn(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),X())}function tn(e,s){var C,P,T,z,te,he,Le;const o=xn=>n.querySelector(`#gw-${CSS.escape(e)}-${xn}`),c=o("tls");if(!c)return s.config.TLS??null;const h=Number.parseInt(((C=o("tls-port"))==null?void 0:C.value.trim())??"",10);return{Enabled:c.checked,Hostname:((P=o("tls-host"))==null?void 0:P.value.trim())??"",CertSource:((T=o("tls-source"))==null?void 0:T.value)??"internal",CertFile:((z=o("tls-cert"))==null?void 0:z.value.trim())??"",KeyFile:((te=o("tls-key"))==null?void 0:te.value.trim())??"",HTTPSPort:Number.isFinite(h)?h:443,BindAddr:((he=s.config.TLS)==null?void 0:he.BindAddr)??"",ImageRef:((Le=s.config.TLS)==null?void 0:Le.ImageRef)??""}}function nn(e,s){q[e]=[s]}async function an(e){if(!m[e]){m[e]=!0,v[e]=null,X();try{v[e]=await Qn(e)}catch(s){v[e]={ok:!1,message:`${pe(s)}${Re(s)}`}}m[e]=!1,X()}}async function sn(e,s){if(!f[e]){f[e]=s,N[e]=null,X();try{await It(e,s)}catch(o){N[e]=`${s} failed: ${pe(o)}${Re(o)}`}f[e]=null,await U()}}async function on(e){if(f[e])return;f[e]="create",N[e]=null,q[e]=["starting…"],X();let s;try{s=await nt(e)}catch(o){N[e]=`${pe(o)}${Re(o)}`,q[e]=[],f[e]=null,X();return}_==null||_(),_=Oe(s.targetId,o=>{if(r)return;const c=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(q[e]=[...(q[e]??[]).filter(C=>C!=="starting…"),c],!!o.err||o.stepId===Ga&&!!o.done){_==null||_(),_=null,f[e]=null,o.err&&(N[e]="Provisioning failed — see the log below."),U();return}X()})}async function rn(e){const s=J(e);if(!(!s||!await Be({title:`Forget ${s.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${s.containerName}" on ${s.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await Xn(e)}catch(c){N[e]=pe(c),X();return}await U()}}async function cn(e){if(e){p[e]=null;try{await Gn(e)}catch(s){p[e]=pe(s),X();return}await U()}}function ln(e){const s=J(e);if(!s)return;const o=new Set((s.networks??[]).map(T=>T.chainId)),c=(i==null?void 0:i.presets)??[],h=c.filter(T=>!o.has(T.chainId)),C=c.filter(T=>o.has(T.chainId)),P=((i==null?void 0:i.targets)??[]).some(T=>T.id===s.placement.targetId&&T.hasDevnet);ce(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${a(s.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${h.map(T=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${T.chainId}">
                <span>${a(T.name)}</span>
                <span class="muted small">chain ${T.chainId}${T.devnet?P?" · uses the devnet on "+a(s.placement.targetId):" · will create a devnet on "+a(s.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${C.length?`<p class="muted small">Already fronted: ${a(C.map(T=>T.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,T=>{if(T==="cancel"){Q();return}if(T==="custom"){dn(e);return}if(T.startsWith("preset:")){const z=Number.parseInt(T.slice(7),10),te=c.find(he=>he.chainId===z);Q(),te!=null&&te.devnet?ut(e,z,P):dt(e,z)}})}function dn(e){var s;ce(`
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
      `,o=>{if(o==="cancel"){Q();return}if(o!=="add")return;const c=document.getElementById("custom-chain-id"),h=document.getElementById("custom-chain-err"),C=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(C)||C<=0){h&&(h.className="error small"),h&&(h.textContent="A chain id is a positive whole number.");return}Q(),dt(e,C)}),(s=document.getElementById("custom-chain-id"))==null||s.focus()}async function dt(e,s){const o=J(e);if(!o)return;const c=Ce(o),h=c.Networks??[];h.some(C=>C.ChainID===s)||(h.push({ChainID:s,Upstreams:[]}),c.Networks=h,await un(e,c)&&(X(),ht(e,s)))}async function un(e,s){var C;const o={...s,Networks:(s.Networks??[]).filter(P=>P.Upstreams.length>0)};if(!await Se(e,o))return!1;const h=J(e);if(h)for(const P of s.Networks??[])P.Upstreams.length===0&&!(h.networks??[]).some(T=>T.chainId===P.ChainID)&&(h.config.Networks=[...h.config.Networks??[],{ChainID:P.ChainID,Upstreams:[]}],h.networks=[...h.networks??[],{chainId:P.ChainID,name:((C=((i==null?void 0:i.presets)??[]).find(T=>T.chainId===P.ChainID))==null?void 0:C.name)??`Chain ${P.ChainID}`,path:`/${h.config.ProjectID}/evm/${P.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function ut(e,s,o){const c=J(e);if(!c)return;if(!o){ce(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${a(c.placement.targetId)}</code>, so adding chain ${s} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(c.placement.targetId)}" data-modal-action="go">Create a devnet on ${a(c.placement.targetId)}</a>
          </div>
        `,()=>Q());return}const h=Ce(c),C=h.Networks??[],P={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},T=C.find(z=>z.ChainID===s);T?T.Upstreams.push(P):C.push({ChainID:s,Upstreams:[P]}),h.Networks=C,await Se(e,h,"Adding the devnet")}async function pn(e,s){const o=J(e);if(!o||!Number.isFinite(s))return;const c=oe(o,s);if(!await Be({title:`Remove ${(c==null?void 0:c.name)??`chain ${s}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${s}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const C=Ce(o);C.Networks=(C.Networks??[]).filter(P=>P.ChainID!==s),await Se(e,C,"Removing the network")}function pt(e){const s=e.split("|");return s.length!==3?null:{gid:s[0],chainId:Number.parseInt(s[1],10),upstreamId:s[2]}}async function hn(e){const s=pt(e);if(!s)return;const o=J(s.gid);if(!o)return;const c=Ce(o),h=(c.Networks??[]).find(T=>T.ChainID===s.chainId);if(!h)return;const C=h.Upstreams.findIndex((T,z)=>(T.ID||`${s.chainId}-${z}`)===s.upstreamId);C<0||!await Be({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(h.Upstreams.splice(C,1),await Se(s.gid,c,"Removing the endpoint"))}function ht(e,s){const o=J(e);if(!o||!Number.isFinite(s))return;const c=((i==null?void 0:i.sources)??[]).filter(T=>T.chainId===s),h=oe(o,s),C=new Set(((h==null?void 0:h.upstreams)??[]).filter(T=>T.kind!=="external").map(T=>`${T.kind}|${T.targetId??""}`)),P=c.filter(T=>!C.has(`${T.kind}|${T.targetId}`));ce(`
        <h2>Add an endpoint for ${a((h==null?void 0:h.name)??`chain ${s}`)}</h2>
        ${P.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${P.map(T=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${a(T.kind)}:${a(T.targetId)}">
                       <span>${a(T.label)}</span>
                       <span class="muted small">${a(T.endpoint)}</span>
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
      `,T=>{if(T==="cancel"){Q();return}if(T==="known-set"){bn(e,s);return}if(T==="manual"){gn(e,s);return}if(T.startsWith("source:")){const[,z,te]=T.split(":");Q(),fn(e,s,z,te)}})}async function fn(e,s,o,c){const h=J(e);if(!h)return;const C=Ce(h),P=C.Networks??[],T={ID:`${o==="managed-devnet"?"devnet":"node"}-${c}`,Kind:o,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},z=P.find(te=>te.ChainID===s);z?z.Upstreams.push(T):P.push({ChainID:s,Upstreams:[T]}),C.Networks=P,await Se(e,C,"Adding the endpoint")}function mn(e){const s=[...e].sort((h,C)=>(h.latencyMs??1e9)-(C.latencyMs??1e9)),o=s.slice(0,3),c=s.find(h=>h.url.startsWith("wss://")||h.url.startsWith("ws://"));return c&&!o.some(h=>h.url===c.url)&&(o.length===3&&o.pop(),o.push(c)),new Set(o.map(h=>h.url))}async function bn(e,s){let o;try{o=await Lt(e,s)}catch(T){ce(`<h2>Endpoints for chain ${s}</h2>
         <p class="error small">Could not read the set: ${a(pe(T))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Q());return}if(r)return;const c=o.endpoints??[],h=c.filter(T=>!T.alreadyAdded).map(T=>T.url),C=new Set(c.map(T=>T.provider)).size,P=c.map(T=>{const z=[T.websocket?'<span class="t ws">websocket</span>':"",T.archive?'<span class="t ar">archive</span>':"",T.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(T.url)}</code>
                  <span class="muted small">${a(T.provider)}</span> ${z}</li>`}).join("");ce(`<h2>Endpoints for chain ${s}</h2>
       ${c.length?`<p class="muted small">${C} providers valve has measured, in the order the gateway
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
       </div>`,T=>{Q(),T==="add"&&Ze(e,s,h),T==="discover"&&yn(e,s)})}async function yn(e,s){ce(`
        <h2>Public endpoints for chain ${s}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,P=>{P==="cancel"&&Q()});let o;try{o=await ea(s)}catch(P){const T=Ue();if(T){const z=document.createElement("p");z.className="error small",z.textContent=`Could not discover endpoints: ${pe(P)}`,T.appendChild(z)}return}if(r)return;const c=(o.endpoints??[]).filter(P=>P.status==="live"||P.status==="unprobed"),h=(o.endpoints??[]).filter(P=>P.status==="rejected"),C=mn(c);ce(`
        <h2>Public endpoints for chain ${s}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${a(o.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(P=>{const T=C.has(P.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${a(P.url)}"${T}>
                       <span><code>${a(P.url)}</code></span>
                       <span class="muted small">${P.status==="live"?`answered in ${P.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${s} right now.</p>`}
        ${h.length?`<details class="rpc-rejected">
                 <summary class="muted small">${h.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${h.map(P=>`<li class="muted small"><code>${a(P.url)}</code> — ${a(P.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,P=>{if(P==="cancel"){Q();return}if(P==="add"){const T=Ue(),z=T?Array.from(T.querySelectorAll('input[type="checkbox"]:checked')).map(te=>te.value):[];Q(),Ze(e,s,z);return}})}function gn(e,s){var o;ce(`
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
      `,c=>{if(c==="cancel"){Q();return}if(c!=="add")return;const h=document.getElementById("manual-endpoint"),C=document.getElementById("manual-recent"),P=document.getElementById("manual-err"),T=(h==null?void 0:h.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(T)){P&&(P.className="error small",P.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Q(),Ze(e,s,[T],(C==null?void 0:C.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ze(e,s,o,c=!1){if(!o.length)return;const h=J(e);if(!h)return;const C=Ce(h),P=C.Networks??[];let T=P.find(te=>te.ChainID===s);T||(T={ChainID:s,Upstreams:[]},P.push(T));let z=1;for(const te of T.Upstreams){const he=/^public-\d+-(\d+)$/.exec(te.ID??"");he&&(z=Math.max(z,Number(he[1])+1))}for(const te of o)T.Upstreams.some(he=>he.Endpoint===te)||T.Upstreams.push({ID:`public-${s}-${z++}`,Kind:"external",Endpoint:te,Local:!1,RecentOnly:c});C.Networks=P,await Se(e,C,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function vn(e,s){const o=pt(e);if(!o||!s||!await Be({title:"Reset this devnet",body:`The chain on ${s} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[o.gid]="reset",N[o.gid]=null,X();let h;try{h=await Kn(s)}catch(C){N[o.gid]=`Reset failed: ${pe(C)}${Re(C)}`,f[o.gid]=null,X();return}f[o.gid]=null,$n(s,h),await U()}function $n(e,s){const o=[];o.push(s.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),s.report.Recreated&&o.push("A fresh chain was started from genesis.");const c=s.report.Cascaded??[],h=s.report.CascadeSkipped??[];ce(`
        <h2>Devnet on ${a(e)} reset</h2>
        <ul class="plain-list">${o.map(C=>`<li>${a(C)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${a(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${h.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(h.join(", "))}.</p>`:""}
        ${s.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(s.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Q())}function wn(e){const s=J(e);if(!s)return;ce(`
        <h2>Wipe ${a(s.label)}</h2>
        <p class="error">This destroys ${a(s.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(e)}</button>
        </div>
      `,h=>{if(h==="cancel"||h==="close"){Q(),U();return}h==="confirm"&&kn(e)});const o=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{c&&(c.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function kn(e){const s=document.getElementById("wipe-confirm-btn");s&&(s.disabled=!0,s.textContent="Wiping…");let o;try{o=await Rt(e)}catch(c){const h=Ue();if(h){const C=document.createElement("p");C.className="error small",C.textContent=`Wipe failed: ${pe(c)}${Re(c)}`,h.appendChild(C)}s&&(s.disabled=!1,s.textContent=`Wipe ${e}`);return}ce(`
        <h2>${a(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${a(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Q(),U()})}function ft(e,s){return!s.some(o=>{var c;return((c=o.placement)==null?void 0:c.targetId)===e})}function Tn(){var C;const e=(i==null?void 0:i.targets)??[],s=(i==null?void 0:i.gateways)??[],o=e.filter(P=>ft(P.id,s)),c=new Set(s.map(P=>P.id));if(e.length===0){ce(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Q());return}if(o.length===0){ce(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Q());return}const h=c.has("default")?"":"default";ce(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${a(h)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${o.map(P=>`<option value="${a(P.id)}">${a(P.id)} (${a(P.mode)})</option>`).join("")}
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
      `,P=>{if(P==="cancel"){Q();return}P==="create"&&Cn()}),(C=document.getElementById("new-gw-id"))==null||C.focus()}async function Cn(){const e=document.getElementById("new-gw-id"),s=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),h=(e==null?void 0:e.value.trim())??"",C=(s==null?void 0:s.value)??"",P=Number.parseInt((o==null?void 0:o.value.trim())??"",10),T=z=>{c&&(c.className="error small",c.textContent=z)};if(!h){T("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!C){T("Pick the machine it runs on.");return}try{await Pt({id:h,placement:{targetId:C,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(P)?P:4e3,Networks:[]}})}catch(z){T(pe(z));return}Q(),await U()}async function Sn(e,s){const o=await qe(s),c=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{r||(e.textContent=c)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Re(e){return e instanceof we&&e.hint?` — ${e.hint}`:""}return()=>{r=!0,_==null||_(),Q()}}const Ya="local";function Za(n){let r=!1,i=!1,t="",d=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ue()}
  `;const b=n.querySelector("#targets-body");ye(n,(p,m)=>{M(p,m)}),w();async function w(){try{const[p,m,v]=await Promise.all([Te(),Pe(),St()]);if(r)return;t=v.os,N(p,m)}catch(p){if(r)return;b.innerHTML=`<p class="error">Failed to load machines: ${a(String(p))}</p>`}}function f(){d&&N(d.targets,d.catalog)}function N(p,m){d={targets:p,catalog:m};const v=t==="linux",x=[...p].sort((U,ne)=>(U.mode==="local"?-1:0)-(ne.mode==="local"?-1:0)),_=x.length?`<div class="card-grid">${x.map(U=>Xa(U,m,U.mode!=="local"||v,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Z=p.some(U=>U.mode==="local");b.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${_}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${q(v,Z)}
        ${i?Qa():""}
      </section>
    `;const H=b.querySelector("#fleet-verdict");H&&Ta(H,ka(p,m))}function q(p,m){const v=`
      <div class="card">
        <h3>A server over SSH ${F("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${p?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${p?" btn-ghost":""}" data-action="toggle-ssh">
            ${i?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,x=p?`
        <div class="card">
          <h3>This machine ${F("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${a(t)})`:""} ${F("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return m?`<div class="card-grid card-grid-wide">${v}</div>`:`<div class="card-grid card-grid-wide">${p?x+v:v+x}</div>`}async function M(p,m){var v;if(p==="add-local"){await I();return}if(p==="delete-target"){const x=m.dataset.id;if(!x||!await Be({title:"Remove machine",body:`Remove "${x}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await O(x);return}if(p==="toggle-ssh"){i=!i,E(),f(),i&&((v=n.querySelector("#ssh-host"))==null||v.focus());return}p==="add-ssh"&&await j()}async function I(){E();try{await tt({id:Ya,mode:"local"}),await w()}catch(p){L(p)}}async function O(p){try{await In(p),await w()}catch(m){L(m)}}async function j(){const p=n.querySelector("#ssh-host"),m=n.querySelector("#ssh-user"),v=n.querySelector("#ssh-key"),x=n.querySelector("#ssh-port"),_=n.querySelector("#ssh-id");if(!p||!m||!v||!x||!_)return;const Z=p.value.trim(),H=m.value.trim(),U=v.value.trim(),ne=x.value.trim(),ae=_.value.trim();if(E(),!Z||!H||!U){L(new Error("host, user, and key path are required"));return}const J=ae||es(Z),oe={Host:Z,User:H,KeyPath:U};if(ne){const de=Number.parseInt(ne,10);if(!Number.isFinite(de)||de<=0){L(new Error("port must be a positive number"));return}oe.Port=de}const re=n.querySelector("#ssh-submit");re&&(re.disabled=!0,re.textContent="Connecting…");try{await tt({id:J,mode:"ssh",ssh:oe}),i=!1,await w()}catch(de){L(de),re&&(re.disabled=!1,re.textContent="Add server")}}function L(p){let m=n.querySelector("#targets-error");m||(b.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),m=n.querySelector("#targets-error")),m.textContent=String(p instanceof Error?p.message:p)}function E(){var p;(p=n.querySelector("#targets-error"))==null||p.remove()}return()=>{r=!0}}function Xa(n,r,i,t){const d=n.wire,b=n.mode==="local"?"this machine":"SSH",w=n.mode==="ssh"&&n.ssh?`${a(n.ssh.User)}@${a(n.ssh.Host)}`:b;let f;if(!d&&!i)f=`${F("can't run a node","warn")} ${F(t||"not Linux","neutral")}`;else if(!d)f=F("not set up","neutral");else{const N=r.networks.find(M=>M.ChainID===d.ChainID),q=N?N.Name:`chain ${d.ChainID}`;f=`${F(q,"ok")} ${F(d.ExecID,"neutral")} ${F(d.BeaconID,"neutral")}${d.Archive?" "+F("archive","warn"):""}`}return`
    <div class="card">
      <h2>${a(n.id)}</h2>
      <p class="muted">${w}</p>
      <p>${f}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(n.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(n.id)}">Remove</button>
      </div>
    </div>
  `}function Qa(){return`
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
  `}function es(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ts=document.querySelector("#app"),{contentEl:ns,setActiveNav:as}=aa(ts);let fe=null;function ss(){const r=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(r.length===0)return{screen:"home"};const[i,t]=r;return i==="machine"||i==="setup"||i==="dash"||i==="logs"||i==="security"||i==="diag"||i==="services"||i==="analytics"?{screen:i,id:t?decodeURIComponent(t):void 0}:{screen:i??"targets"}}function $e(n){const r=document.createElement("div");return ns.replaceChildren(r),n(r)}function Nt(){if(fe){try{fe()}catch{}fe=null}const{screen:n,id:r}=ss();switch(as(n),n){case"machine":if(!r){location.hash="#/targets";return}fe=$e(i=>ga(i,r));break;case"setup":case"dash":case"logs":case"services":if(!r){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(r)}`;return;case"security":if(!r){location.hash="#/targets";return}fe=$e(i=>ja(i,r));break;case"diag":if(!r){location.hash="#/targets";return}fe=$e(i=>ca(i,r));break;case"analytics":if(!r){location.hash="#/rpc";return}fe=$e(i=>ia(i,r));break;case"rpc":fe=$e(i=>Ja(i));break;case"settings":fe=$e(i=>Wa(i));break;case"targets":fe=$e(i=>Za(i));break;case"panel":fe=$e(i=>Tt(i));break;case"home":default:fe=$e(i=>Tt(i));break}}window.addEventListener("hashchange",Nt);Nt();
