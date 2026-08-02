var On=Object.defineProperty;var Fn=(n,s,r)=>s in n?On(n,s,{enumerable:!0,configurable:!0,writable:!0,value:r}):n[s]=r;var Je=(n,s,r)=>Fn(n,typeof s!="symbol"?s+"":s,r);(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))e(d);new MutationObserver(d=>{for(const l of d)if(l.type==="childList")for(const y of l.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function r(d){const l={};return d.integrity&&(l.integrity=d.integrity),d.referrerPolicy&&(l.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?l.credentials="include":d.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function e(d){if(d.ep)return;d.ep=!0;const l=r(d);fetch(d.href,l)}})();function Ut(){return oe("/api/host")}function Be(){return oe("/api/catalog")}function De(){return oe("/api/targets")}function it(n){return oe("/api/targets",{method:"POST",headers:Ie,body:JSON.stringify(n)})}function jn(n){return oe(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function qn(n,s){return oe(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(s)}`)}function Wn(n,s){return oe(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:Ie,body:JSON.stringify(s)})}function ze(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function _n(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function Kn(n,s=200){return oe(`/api/targets/${encodeURIComponent(n)}/logs?n=${s}`)}function Vn(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return r.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>r.close()}function Tt(n,s){const r=s===void 0?{}:{lines:s};return oe(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:Ie,body:JSON.stringify(r)})}function Gn(n,s,r){return oe(`/api/targets/${encodeURIComponent(n)}/services/${s}/${r}`,{method:"POST"})}function zn(n,s){return oe(`/api/targets/${encodeURIComponent(n)}/services/${s}/clear`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:s})})}function Jn(n){return oe(`/api/targets/${encodeURIComponent(n)}/du`)}function Yn(n){return oe(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function Zn(n){return oe(`/api/targets/${encodeURIComponent(n)}/firewall`)}function Xn(n){return oe(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function Qn(n){return oe(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function Mt(n){return oe(`/api/targets/${encodeURIComponent(n)}/containers`)}function ea(n,s,r){return oe(`/api/targets/${encodeURIComponent(n)}/containers/${s}/${r}`,{method:"POST"})}async function ta(n,s){const r=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${s}/wipe`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:s})}),e=await r.text();let d=null;try{d=e?JSON.parse(e):null}catch{}if(d&&typeof d=="object"&&"report"in d)return d;const l=d&&typeof d=="object"&&typeof d.error=="string"?d.error:r.statusText||`HTTP ${r.status}`;throw new Re(r.status,l)}function na(n,s){return oe(`/api/targets/${encodeURIComponent(n)}/containers/${s}/provision`,{method:"POST"})}async function aa(n){const s=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:Ie}),r=await s.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Re(s.status,d)}function sa(n,s,r){return oe(`/api/targets/${encodeURIComponent(n)}/containers/${s}/config`,{method:"PUT",headers:Ie,body:JSON.stringify(r)})}function ht(){return oe("/api/gateways")}async function oa(n){await oe(`/api/orphans/${encodeURIComponent(n)}`,{method:"DELETE"})}function Ot(n){return oe("/api/gateways",{method:"POST",headers:Ie,body:JSON.stringify(n)})}function Ft(n){return oe(`/api/gateways/${encodeURIComponent(n)}/tls/verify`)}function ra(n){return oe(`/api/gateways/${encodeURIComponent(n)}/traffic`)}function ct(n){return oe(`/api/gateways/${encodeURIComponent(n)}/analytics`)}function jt(n,s=!1){const r=s?"?refresh=1":"";return oe(`/api/gateways/${encodeURIComponent(n)}/capabilities${r}`)}function ia(n){return oe(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function Le(n,s){return oe(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:Ie,body:JSON.stringify(s)})}function qt(n,s){return oe(`/api/gateways/${encodeURIComponent(n)}/${s}`,{method:"POST"})}function ca(n){return oe(`/api/gateways/${encodeURIComponent(n)}/trust-cert`,{method:"POST"})}function lt(n){return oe(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Wt(n){const s=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:n})}),r=await s.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Re(s.status,d)}function la(n){return oe(`/api/chainlist/${n}`)}function dt(n,s){return oe(`/api/gateways/${encodeURIComponent(n)}/knownset/${s}`)}function da(){return oe("/api/settings")}function ua(n){return oe("/api/settings",{method:"PUT",headers:Ie,body:JSON.stringify(n)})}class Re extends Error{constructor(r,e,d,l){super(e);Je(this,"status");Je(this,"hint");Je(this,"code");this.name="ApiError",this.status=r,this.hint=d,this.code=l}}const Ie={"Content-Type":"application/json"};async function oe(n,s){const r=await fetch(n,s);if(!r.ok){let d=r.statusText||`HTTP ${r.status}`,l,y;try{const p=await r.json();p&&typeof p.error=="string"&&p.error&&(d=p.error),p&&typeof p.hint=="string"&&p.hint&&(l=p.hint),p&&typeof p.code=="string"&&p.code&&(y=p.code)}catch{}throw new Re(r.status,d,l,y)}if(r.status===204)return;const e=await r.text();return e?JSON.parse(e):void 0}const It="https://learn.valve.city/rpc";function a(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ge(n,s){const r=n&&s&&s!==It?` <span class="footer-sep">·</span> <a href="${a(s)}" target="_blank" rel="noopener noreferrer">${a(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(It)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function pa(n){n.innerHTML=`
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
  `;const s=n.querySelector("#content"),r=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:s,setActiveNav:d=>{const l=d==="machine"?"targets":d==="home"||d==="panel"?"rpc":d;for(const y of r)y.classList.toggle("active",y.dataset.nav===l)}}}function ye(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function ha(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function fa(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const s=Math.round(n/60),r=Math.floor(s/60),e=s%60;if(r===0)return`~${e}m`;if(r<48)return`~${r}h ${e}m`;const d=Math.floor(r/24),l=r%24;return`~${d}d ${l}h`}function J(n,s){return`<span class="badge badge-${s}">${a(n)}</span>`}function Ne(n){return`<span class="dot dot-${n}"></span>`}const Et=["B","KB","MB","GB","TB","PB"];function Me(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let s=n,r=0;for(;s>=1024&&r<Et.length-1;)s/=1024,r++;const e=s<10?2:s<100?1:0;return`${s.toFixed(e)} ${Et[r]}`}async function Ke(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function Ee(n,s){n.addEventListener("click",r=>{const e=r.target.closest("[data-action]");if(!e||!n.contains(e))return;const d=e.dataset.action;d&&s(d,e,r)})}function ut(n,s,r){const e=s.find(l=>l.value===r),d=s.map(l=>`
      <li class="dropdown-option${l.value===r?" selected":""}" role="option"
          aria-selected="${l.value===r}" data-value="${a(l.value)}">
        ${a(l.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function Ve(n){n.querySelectorAll(".dropdown.open").forEach(s=>{var r;s.classList.remove("open"),(r=s.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function ft(n,s){n.addEventListener("click",d=>{const l=d.target,y=l.closest(".dropdown-trigger");if(y&&n.contains(y)){const I=y.closest(".dropdown"),j=!!I&&!I.classList.contains("open");Ve(n),I&&j&&(I.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const p=l.closest(".dropdown-option");if(p&&n.contains(p)){const I=p.closest(".dropdown");Ve(n),s((I==null?void 0:I.dataset.dropdown)??"",p.dataset.value??"");return}Ve(n)});const r=d=>{if(!n.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}const l=d.target;(!l.closest(".dropdown")||!n.contains(l))&&Ve(n)},e=d=>{if(!n.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}d.key==="Escape"&&Ve(n)};document.addEventListener("click",r),document.addEventListener("keydown",e)}const et="app-modal";let Qe=null;function he(n,s){ee();const r=document.createElement("div");r.className="modal-overlay",r.id=et,r.innerHTML=`<div class="modal">${n}</div>`,r.addEventListener("click",d=>{const l=d.target.closest("[data-modal-action]");l!=null&&l.dataset.modalAction?s(l.dataset.modalAction):d.target===r&&s("cancel")});const e=d=>{d.key==="Escape"&&s("cancel")};document.addEventListener("keydown",e),Qe=e,document.body.appendChild(r)}function ee(){var n;(n=document.getElementById(et))==null||n.remove(),Qe&&(document.removeEventListener("keydown",Qe),Qe=null)}function We(){return document.querySelector(`#${et} .modal`)}function Ae(n){return new Promise(s=>{var d;let r=!1;const e=l=>{r||(r=!0,ee(),s(l))};he(`
        <h2>${a(n.title)}</h2>
        <p>${a(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${a(n.confirmLabel)}</button>
        </div>
      `,l=>e(l==="confirm")),(d=document.querySelector(`#${et} [data-modal-action="confirm"]`))==null||d.focus()})}const at=5e3,ma=60;function ba(n,s){let r=!1,e=null,d=null,l=null,y=null;const p=[];n.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ge()}</div>`;const I=n.querySelector("#an-body");Ee(n,($,h)=>{var T;$==="toggle-endpoint"&&((T=h.closest(".an-endpoint"))==null||T.classList.toggle("expanded"))}),j();async function j(){try{e=((await ht()).gateways??[]).find(h=>h.id===s)??null}catch($){if(r)return;l=String($ instanceof Error?$.message:$),_();return}if(!r){if(!e){_();return}await V(),y=window.setInterval(()=>void V(),at)}}async function V(){try{const $=await ct(s);if(r)return;L($),d=$,l=null}catch($){if(r)return;l=String($ instanceof Error?$.message:$)}_()}function L($){if(!$.enabled||$.error)return;const h=p[p.length-1];h&&h.since!==$.since&&(p.length=0);const T=new Map;for(const B of $.networks??[])T.set(B.chainId,B.received);p.push({t:Date.now(),since:$.since,received:T}),p.length>ma&&p.shift()}function _(){r||(I.innerHTML=G())}function G(){return l&&!d?`<h1>Analytics</h1><p class="error">${a(l)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${A(e)}
      ${d?f(d):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${a(s)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function A($){return`
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
    `}function E(){if(!d)return"";if(!d.enabled)return"counters off";if(d.error)return"could not be read";const $=d.since?new Date(d.since):null;return $&&!Number.isNaN($.getTime())?`totals since the gateway started, ${a($.toLocaleString())}<br />re-read every ${at/1e3}s`:`re-read every ${at/1e3}s`}function f($){return $.enabled?$.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a($.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:g($)+be($):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function g($){const h=$.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${h.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':h.map(T=>x(T)).join("")}
      </section>
    `}function x($){const h=$.methods??[],T=$.endpoints??[],B=$.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${$.chainId}</span>
          <span class="band-name">${a($.name)}</span>
          ${K($)}
        </div>
        <div class="an-stats">
          ${F("Received",ye($.received),"what clients asked this chain for")}
          ${F("Answered",ye($.answered),"returned by one of your endpoints")}
          ${F("From cache",ye($.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${F("Failed",ye($.failed),"asked for and never answered",$.failed>0?"bad":"")}
        </div>
        ${ce($.chainId)}
        ${B?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ie("Method",h.map(W=>({label:W.method,l:W})))+ie("Endpoint",T.map(W=>({label:W.upstream,l:W})))+z($)}
      </div>
    `}function F($,h,T,B=""){return`
      <div class="an-stat${B?" an-stat-"+B:""}" title="${a(T)}">
        <span class="an-stat-n">${a(h)}</span>
        <span class="an-stat-l">${a($)}</span>
      </div>
    `}function K($){const h=te($.chainId);if(h===null)return'<span class="an-rate muted small">measuring rate…</span>';const T=Math.round((p[p.length-1].t-p[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${T}s apart.">
      ${a(h.toFixed(h<10?2:0))} req/s <span class="muted">over the last ${T}s</span>
    </span>`}function te($){if(p.length<2)return null;const h=p[0],T=p[p.length-1],B=(T.t-h.t)/1e3;if(B<=0)return null;const W=(T.received.get($)??0)-(h.received.get($)??0);return W<0?null:W/B}function ce($){if(p.length<3)return"";const h=[];for(let w=1;w<p.length;w++){const D=p[w-1],Z=p[w],u=(Z.t-D.t)/1e3,v=(Z.received.get($)??0)-(D.received.get($)??0);h.push(u>0&&v>=0?v/u:0)}const T=Math.max(...h);if(T<=0)return"";const B=240,W=28,Q=h.length>1?B/(h.length-1):B,b=h.map((w,D)=>`${(D*Q).toFixed(1)},${(W-w/T*W).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${T.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${B} ${W}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(T.toFixed(2))} req/s</span>
      </div>
    `}function z($){const h=[];return $.cached.count>0&&h.push(`${a(ye($.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${$.cached.mean===null?"":`, in ${a(Ge($.cached.mean))} on average`}.`),$.failedLatency.count>0&&$.failedLatency.mean!==null&&h.push(`The ${a(ye($.failedLatency.count))} that failed took
         ${a(Ge($.failedLatency.mean))} on average to fail.`),h.length===0?"":`<p class="muted small">${h.join(" ")}</p>`}function ie($,h){return h.length===0?"":`
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
            ${h.map(T=>U(T.label,T.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function U($,h){return`
      <tr>
        <td><code>${a($)}</code></td>
        <td class="an-num">${ye(h.count)}</td>
        <td class="an-num">${h.mean===null?'<span class="muted">—</span>':a(Ge(h.mean))}</td>
        <td>${ne(h)}</td>
      </tr>
    `}function ne($){const h=$.buckets??[];if(h.length===0||$.count===0)return'<span class="muted small">—</span>';let T=0;const B=[];for(const Q of h){const b=Q.count-T;T=Q.count,B.push({label:pe(Q.le),n:Math.max(0,b)})}return B.reduce((Q,b)=>Q+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(B.filter(Q=>Q.n>0).map(Q=>`${Q.n} ${Q.label}`).join(" · "))}">
        ${B.map((Q,b)=>Q.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${Q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(de(B))}</span>
    `}function de($){for(let h=$.length-1;h>=0;h--)if($[h].n>0)return`slowest ${$[h].label}`;return""}function pe($){if($==="+Inf")return"30s or more";const h=Number($);return Number.isFinite(h)?`under ${Ge(h)}`:`under ${$}`}function be($){const h=$.endpoints??[];return`
      <section class="an-section">
        <h2>What the gateway sees from your endpoints</h2>
        <p class="muted small">
          The gateway's own view, not a client's. Every count here <strong>includes the
          gateway's block-tracking poller</strong>, which calls each endpoint on a timer
          whether or not anyone is using it — on a quiet gateway it is nearly all of this.
          That is why these numbers are much larger than the ones above, and why they are
          not a measure of your traffic.
        </p>
        ${h.length===0?'<div class="card"><p class="muted">The gateway has not talked to any endpoint yet.</p></div>':`<div class="card">
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
                     <tbody>${h.map(T=>re(T)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function re($){const h=$.errors??[],T=h.reduce((W,Q)=>W+Q.count,0),B=h.length>0;return`
      <tr class="an-endpoint${B?" expandable":""}" ${B?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${a($.upstream)}</code>
          ${$.chainId?`<span class="muted small">chain ${$.chainId}</span>`:""}
          ${$.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ye($.requests)}</td>
        <td class="an-num${T>0?" bad":""}">${T>0?ye(T):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${$.headLag>0?ye($.headLag):'<span class="muted">0</span>'}</td>
        <td>${we($)}</td>
      </tr>
      ${B?Se($,h):""}
    `}function we($){const h=[];return $.scored?(h.push($.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String($.position))}</span>`),h.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a($.score.toFixed(3))}</span>`),$.primarySwitches>1&&h.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ye($.primarySwitches)} switches</span>`),$.excludedSeconds>0&&h.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(Ge($.excludedSeconds))}</span>`),`<span class="an-selection">${h.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function Se($,h){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${h.map(T=>`
                    <tr>
                      <td class="an-num">${ye(T.count)}</td>
                      <td><code>${a(T.class)}</code></td>
                      <td>${T.severity?`<span class="badge badge-${T.severity==="critical"?"bad":"warn"}">${a(T.severity)}</span>`:""}</td>
                      <td class="muted small">${a(T.method||"")}</td>
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
    `}return()=>{r=!0,y!==null&&window.clearInterval(y)}}function Ge(n){return!Number.isFinite(n)||n<0?"—":n>0&&n<5e-4?"<1ms":n<1?`${Math.round(n*1e3)}ms`:n<60?`${n<10?n.toFixed(1):Math.round(n)}s`:`${Math.round(n/60)}m`}function ya(n,s){let r=!1,e=null,d=null,l=!1,y=!1;n.innerHTML=`<h1>Network diagnostics: ${a(s)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ge()}</div>`;const p=n.querySelector("#diag-body"),I=n.querySelector("#diag-footer");Ee(n,(f,g)=>{var x;if(f==="run")V();else if(f==="toggle")(x=g.closest(".check-item"))==null||x.classList.toggle("expanded");else if(f==="copy"){const F=g.dataset.copy;F&&E(g,F)}}),j();async function j(){let f,g;try{const[F,K]=await Promise.all([De(),Be()]);f=F.find(te=>te.id===s),g=K}catch(F){if(r)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(F))}</p>`;return}if(r)return;if(!f){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const x=g==null?void 0:g.networks.find(F=>F.ChainID===f.wire.ChainID);x&&(I.innerHTML=ge(x.Name,x.LearnURL));try{e=await Qn(s),y=!0}catch(F){d=String(F instanceof Error?F.message:F)}r||L()}async function V(){l=!0,d=null,L();try{e=await Xn(s),y=!0}catch(f){d=String(f instanceof Error?f.message:f)}l=!1,r||L()}function L(){p.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${l?"disabled":""}>${l?"Running…":"Run diagnostics"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${_()}
    `}function _(){if(!y&&!d)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const f=new Date(e.at).toLocaleString(),g=e.failedId?`<p><strong>Failed at: ${a(G(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${a(f)} — trigger: ${a(e.trigger)}</p>
      ${g}
      <ul class="check-list">${e.items.map(A).join("")}</ul>
    `}function G(f){var g;return((g=e==null?void 0:e.items.find(x=>x.ID===f))==null?void 0:g.Title)??f}function A(f){const g=f.Status==="pass"?"ok":f.Status==="fail"?"bad":f.Status==="warn"?"warn":"neutral",x=f.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${x?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${J(x?"failed here":f.Status,g)}
          <strong>${a(f.Title)}</strong>
          <span class="muted small check-detail-inline">${a(f.Detail)}</span>
        </button>
        <div class="check-body">
          <details${x?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(f.Why)}</p>
          </details>
          ${f.Fix?`
                <details${x?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(f.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(f.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function E(f,g){const x=await Ke(g),F=f.textContent;f.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{r||(f.textContent=F)},1500)}return()=>{r=!0}}const va=85,st={exec:"Execution",beacon:"Beacon"};function ga(n,s){let r=!1,e=null,d=null,l=null,y=null,p=null,I=null,j=null,V=null;const L={exec:null,beacon:null};let _=null;n.innerHTML=`<h1>Dashboard: ${a(s)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ge()}</div>`;const G=n.querySelector("#dash-body"),A=n.querySelector("#dash-footer");G.addEventListener("click",h=>{const T=h.target.closest("[data-action]");if(!T||!G.contains(T))return;const B=T.dataset.action;if(B==="svc-action"){const W=T.dataset.svc,Q=T.dataset.kind;W&&Q&&re(W,Q)}else if(B==="open-clear"){const W=T.dataset.svc;W&&Se(W)}else if(B==="copy"){const W=T.dataset.copy;W&&we(T,W)}else B==="retry-du"?f():B==="retry-endpoints"&&g()}),E();async function E(){let h,T;try{const[W,Q]=await Promise.all([De(),Be()]);h=W.find(b=>b.id===s),T=Q}catch(W){if(r)return;G.innerHTML=`<p class="error">Failed to load target: ${a(String(W))}</p>`;return}if(r)return;if(!h){G.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){G.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const B=T==null?void 0:T.networks.find(W=>W.ChainID===h.wire.ChainID);B&&(A.innerHTML=ge(B.Name,B.LearnURL)),G.innerHTML='<p class="muted">Connecting…</p>',e=_n(s,W=>{r||(x(W),d=W,l=W,F())}),f(),g()}async function f(){I=null;try{p=await Jn(s)}catch(h){p=null,I=String(h instanceof Error?h.message:h)}r||F()}async function g(){V=null;try{j=await Yn(s)}catch(h){j=null,V=String(h instanceof Error?h.message:h)}r||F()}function x(h){if(!d)return;const T=(new Date(h.at).getTime()-new Date(d.at).getTime())/1e3,B=h.execHead-d.execHead;if(T>0&&B>=0){const W=B/T;y=y===null?W:y*.7+W*.3}}function F(){if(!l)return;const h=l;G.innerHTML=`
      <p class="dash-status">${K(h)}</p>
      <div class="card-grid">
        ${pe(h)}
        ${ce(h)}
        ${z(h)}
        ${ie(h)}
        ${U(h)}
        ${ne()}
      </div>
      <p class="muted small">Last updated ${a(new Date(h.at).toLocaleTimeString())}</p>
    `}function K(h){return!h.execActive&&!h.beaconActive?J("Node not running","bad"):h.execSyncing||h.beaconDistance>0?J("Syncing","warn"):J("Running · synced","ok")}function te(h){const B=h.refHead>0?h.refHead-h.execHead:null,W=B!==null&&B>0&&y&&y>0?fa(B/y):B!==null&&B<=0?"caught up":"—";return{lag:B,eta:W}}function ce(h){const{lag:T,eta:B}=te(h);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${h.execActive?h.execSyncing?J("syncing","warn"):h.execHead===0?J("no data","neutral"):J("synced","ok"):J("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ye(h.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${T!==null?ye(h.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${T!==null?ye(Math.max(T,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${B}</dd></div>
        </dl>
      </div>
    `}function z(h){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${h.beaconActive?h.beaconSlot===0?J("no data","neutral"):h.beaconDistance===0?J("synced","ok"):J("syncing","warn"):J("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ye(h.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ye(h.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ie(h){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ye(h.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ye(h.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function U(h){const T=h.diskUsedPct>=va,B=`
      <div class="meter"><div class="meter-fill ${T?"meter-warn":""}" style="width:${Math.min(h.diskUsedPct,100)}%"></div></div>
      <p>${ha(h.diskUsedPct)} used</p>
    `;if(I)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${B}
          <p class="error small">${a(I)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!p)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${B}
          <p class="muted">Loading…</p>
        </div>
      `;const W=p.ExpectedExecBytes>0?Math.min(p.ExecBytes/p.ExpectedExecBytes*100,100):0,Q=p.ExpectedBeaconBytes>0?Math.min(p.BeaconBytes/p.ExpectedBeaconBytes*100,100):0,{lag:b,eta:w}=te(h),D=b!==null&&b>0&&y!==null&&y>0;return`
      <div class="card ${T?"card-warn":""}">
        <h3>Storage</h3>
        ${B}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Me(p.ExecBytes)} of ~${Me(p.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${W}%"></div></div>
        ${D?`<p class="muted small">Estimated time remaining: ${a(w)}</p>`:""}
        <p class="muted small">Beacon — ${Me(p.BeaconBytes)} of ~${Me(p.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${Q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Me(p.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(p.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(p.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function ne(){if(V)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(V)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const h=j,T=h.ExecReachable&&!h.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",B=h.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(h.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${Ne(h.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${Ne(h.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.BeaconHTTP)}">Copy</button>
        </div>
        ${T}
        ${B}
      </div>
    `}function de(h,T){const B=st[h],W=L[h],Q=(b,w,D)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${h}" data-kind="${b}" ${W!==null||D?"disabled":""}>${W===b?be():a(w)}</button>`;return`
      <div class="service-row">
        <span>${a(B)} ${T?J("active","ok"):J("down","bad")}</span>
        <div class="service-actions">
          ${Q("start","Start",T)}
          ${Q("stop","Stop",!T)}
          ${Q("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${h}" ${W!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function pe(h){return`
      <div class="card">
        <h3>Services</h3>
        ${de("exec",h.execActive)}
        ${de("beacon",h.beaconActive)}
        ${_?`<p class="error small">${a(_)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(s)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(s)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(s)}">Diagnostics →</a>
        </p>
      </div>
    `}function be(){return'<span class="spinner" aria-label="working"></span>'}async function re(h,T){if(L[h]===null){L[h]=T,_=null,F();try{await Gn(s,h,T)}catch(B){_=`${st[h]} ${T} failed: ${B instanceof Error?B.message:String(B)}`}L[h]=null,r||F()}}async function we(h,T){const B=await Ke(T),W=h.textContent;h.textContent=B?"Copied!":"Copy failed",setTimeout(()=>{r||(h.textContent=W)},1500)}function Se(h){const T=st[h],B=p?Me(h==="exec"?p.ExecBytes:p.BeaconBytes):"unknown (disk usage hasn't loaded)";he(`
        <h2>Clear ${a(T)} data</h2>
        <p class="error">
          This stops the ${a(T.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(B)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(h)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,b=>{if(b==="cancel"){ee();return}b==="confirm"&&$(h)});const W=document.getElementById("clear-confirm-input"),Q=document.getElementById("clear-confirm-btn");W==null||W.addEventListener("input",()=>{Q&&(Q.disabled=W.value.trim()!==h)}),W==null||W.focus()}async function $(h){const T=document.getElementById("clear-confirm-btn");T&&(T.disabled=!0,T.textContent="Clearing…");try{await zn(s,h),ee(),f()}catch(B){const W=We();if(W){const Q=document.createElement("p");Q.className="error small",Q.textContent=`Clear failed: ${B instanceof Error?B.message:String(B)}`,W.appendChild(Q)}T&&(T.disabled=!1,T.textContent="Clear and resync")}}return()=>{r=!0,e==null||e(),ee()}}const Pt=500,Rt="valve-node-app.explain-consent";function $a(n,s){let r=!1,e=null;const d=[];n.innerHTML=`
    <h1>Logs: ${a(s)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ge()}</div>
  `;const l=n.querySelector("#logs-body"),y=n.querySelector("#logs-footer");Ee(n,E=>{E==="explain"&&V()}),p();async function p(){let E,f;try{const[x,F]=await Promise.all([De(),Be()]);E=x.find(K=>K.id===s),f=F}catch(x){if(r)return;l.innerHTML=`<p class="error">Failed to load target: ${a(String(x))}</p>`;return}if(r)return;if(!E){l.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!E.wire){l.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const g=f==null?void 0:f.networks.find(x=>x.ChainID===E.wire.ChainID);g&&(y.innerHTML=ge(g.Name,g.LearnURL));try{const x=await Kn(s,200);if(r)return;d.push(...x)}catch(x){if(r)return;l.innerHTML=`<p class="error">Failed to load logs: ${a(String(x))}</p>`;return}I(),e=Vn(s,x=>{r||(d.push(x),d.length>Pt&&d.splice(0,d.length-Pt),I())})}function I(){const E=d.filter(g=>g.severity==="error"||g.severity==="critical");l.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${J(String(E.length),E.length?"bad":"neutral")}</h2>
          <div class="log-lines">${E.length?E.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const f=l.querySelector(".log-lines");f&&(f.scrollTop=f.scrollHeight)}function j(E){const f=E.severity||"info",g=E.learnUrl?` <a href="${a(E.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(f)}">
        <span class="log-time">${a(new Date(E.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(E.unit)}</span>
        <span class="log-sev">${a(f)}</span>
        <span class="log-text">${a(E.line)}</span>
        ${E.explain?`<div class="log-explain">${a(E.explain)}${g}</div>`:""}
      </div>
    `}async function V(){const E=d.filter(g=>g.severity==="error"||g.severity==="critical").map(g=>g.line).slice(-40);if(!(localStorage.getItem(Rt)==="1")){L(E);return}await _(E)}function L(E){const f=E.length?`<pre class="explain-excerpt">${E.map(g=>a(g)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';G(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${f}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,g=>{g==="proceed"?(localStorage.setItem(Rt,"1"),A(),_(E)):A()})}async function _(E){G('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const f=E.length?await Tt(s,E):await Tt(s);if(r)return;G(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(f.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${f.sentExcerpt.map(g=>a(g)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&A()})}catch(f){if(r)return;if(f instanceof Re&&f.status===409){G(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,g=>{g==="close"&&A()});return}G(`
        <h2>Explain failed</h2>
        <p class="error">${a(f instanceof Error?f.message:String(f))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&A()})}}function G(E,f){A();const g=document.createElement("div");g.className="modal-overlay",g.id="explain-modal",g.innerHTML=`<div class="modal">${E}</div>`,g.addEventListener("click",x=>{const F=x.target.closest("[data-modal-action]");F!=null&&F.dataset.modalAction&&f(F.dataset.modalAction),x.target===g&&f("cancel")}),document.body.appendChild(g)}function A(){var E;(E=document.getElementById("explain-modal"))==null||E.remove()}return()=>{r=!0,e==null||e(),A()}}const wa="run",ka={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Ca={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Sa(n,s){let r=!1,e=null,d=null;const l={devnet:null},y={devnet:null},p={devnet:[]};let I=null;const j={devnet:!1};let V=null;const L={devnet:null},_={devnet:null};n.innerHTML=`
    <div class="page-head">
      <h1>Services: ${a(s)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${ge()}
  `;const G=n.querySelector("#services-body");Ee(n,(u,v)=>{Se(u,v)}),A();async function A(){try{const u=await Mt(s);if(r)return;e=u,d=null}catch(u){if(r)return;e=null,d=D(u)}f()}function E(u){return e==null?void 0:e.services.find(v=>v.id===u)}function f(){if(!r){if(d){G.innerHTML=`<p class="error">Could not read this machine's services: ${a(d)}</p>`;return}if(!e){G.innerHTML='<p class="muted">Loading…</p>';return}G.innerHTML=`
      ${g(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(x).join("")}
      </div>
    `}}function g(u){if(u.present&&u.reachable&&!u.hint)return`<p class="muted small">Docker: ${a(u.flavor)}${u.serverVersion?` ${a(u.serverVersion)}`:""} · reachable</p>`;const v=u.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(v)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${u.detail?`<div class="small">${a(u.detail)}</div>`:""}
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function x(u){const v=u.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(u.label)}</h2>
          ${F(u)}
        </div>
        <p class="muted small">${a(ka[u.id]??"")}</p>

        ${u.error?K(u):""}
        ${u.blocked?`<div class="banner banner-warn">${a(u.blocked)}</div>`:""}
        ${v.map(O=>`<div class="banner banner-warn">${a(O)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(u.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${u.status.Image?`<code>${a(u.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${te(u)}

        ${ce(u)}

        <div class="card-actions">
          ${(u.actions??[]).map(O=>z(u,O)).join("")}
        </div>
        ${y[u.id]?`<p class="error small">${a(y[u.id])}</p>`:""}
        ${ie(u)}

        ${U(u)}
      </div>
    `}function F(u){switch(u.status.State){case"running":return J("running","ok");case"created-but-stopped":return J("stopped","warn");case"not-created":return J("not created","neutral");default:return J("unknown","bad")}}function K(u){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(u.error??"")}</div>
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function te(u){if(u.status.State!=="created-but-stopped"||u.status.ExitCode===0)return"";const v=u.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${u.status.ExitCode}${v}.</p>`}function ce(u){const v=u.endpoints??[];return v.length===0?u.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":v.map(O=>`
        <div class="endpoint-row">
          ${Ne("ok")}
          <span class="muted small">${a(O.label)}</span>
          <code class="endpoint-url">${a(O.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(O.url)}">Copy</button>
        </div>`).join("")}function z(u,v){const O=Ca[v];if(!O)return"";const X=l[u.id],le=v==="create"?`Create ${u.id==="devnet"?"devnet":"gateway"}`:O.label;return`
      <button class="${O.className}" data-action="svc-${v}" data-svc="${a(u.id)}"
              title="${a(O.title)}" ${X?"disabled":""}>
        ${X===v?'<span class="spinner" aria-label="working"></span>':a(le)}
      </button>
    `}function ie(u){const v=p[u.id]??[];return v.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(v.join(`
`))}</pre>
      </div>
    `}function U(u){const v=j[u.id],O=ne(u);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${u.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(u.id)}">
            ${v?"Close":"Edit"}
          </button>
        </div>
        ${v?de():`<p class="small">${O}</p>`}
        ${L[u.id]?`<p class="error small">${a(L[u.id])}</p>`:""}
        ${_[u.id]?`<p class="muted small">${a(_[u.id])}</p>`:""}
      </div>
    `}function ne(u){const v=u.devnet;return v?`Chain ${v.ChainID} · a block every ${a(v.BlockTime)} · JSON-RPC on ${a(v.BindAddr)}:${v.HTTPPort} · WebSocket on ${a(v.BindAddr)}:${v.WSPort}`:"—"}function de(u){return pe()}function pe(){const u=V;return u?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${a(u.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${u.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${u.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${a(u.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${u.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function be(){j.devnet&&V&&(V.BlockTime=re("#dev-blocktime",V.BlockTime),V.HTTPPort=we("#dev-http",V.HTTPPort),V.WSPort=we("#dev-ws",V.WSPort),V.BindAddr=re("#dev-bind",V.BindAddr))}function re(u,v){const O=n.querySelector(u);return O?O.value.trim():v}function we(u,v){const O=n.querySelector(u);if(!O)return v;const X=Number.parseInt(O.value.trim(),10);return Number.isFinite(X)?X:v}async function Se(u,v){const O=v.dataset.svc??"";switch(u){case"refresh":await A();return;case"copy":v.dataset.copy&&await w(v,v.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await $(O,u.slice(4));return;case"svc-create":case"svc-recreate":await h(O);return;case"svc-wipe":W(O);return;case"toggle-config":T(O);return;case"save-config":await B(O);return;default:return}}async function $(u,v){if(!l[u]){l[u]=v,y[u]=null,f();try{await ea(s,u,v)}catch(O){y[u]=`${v} failed: ${D(O)}${Z(O)}`}l[u]=null,await A()}}async function h(u){if(!l[u]){l[u]="create",y[u]=null,p[u]=["starting…"],f();try{await na(s,u)}catch(v){y[u]=`${D(v)}${Z(v)}`,p[u]=[],l[u]=null,f();return}I==null||I(),I=ze(s,v=>{if(r)return;const O=v.err?`${v.stepId}: ${v.err}`:v.line?`${v.stepId}: ${v.line}`:`${v.stepId}: done`;if(p[u]=[...(p[u]??[]).filter(le=>le!=="starting…"),O],!!v.err||v.stepId===wa&&!!v.done){I==null||I(),I=null,l[u]=null,v.err&&(y[u]="Provisioning failed — see the log below."),A();return}f()})}}function T(u){if(be(),j[u]=!j[u],L[u]=null,_[u]=null,j[u]){const v=E(u);v!=null&&v.devnet&&(V={...v.devnet})}f()}async function B(u){var X;be(),L[u]=null,_[u]=null;const v=V;if(!v)return;if(v.HTTPPort===v.WSPort){L[u]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",f();return}try{await sa(s,u,v)}catch(le){L[u]=D(le),f();return}const O=((X=E(u))==null?void 0:X.status.State)==="running";j[u]=!1,_[u]=O?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await A()}function W(u){const v=E(u);if(!v)return;const O=(v.restartsOnWipe??[]).map(Y=>{var fe;return((fe=E(Y))==null?void 0:fe.label)??Y});he(`
        <h2>Wipe ${a(v.label)}</h2>
        <p class="error">This deletes ${a(v.wipeDiscards)}</p>
        ${O.length?`<p>It also restarts what sits in front of it: ${a(O.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${a(u)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(u)}</button>
        </div>
      `,Y=>{if(Y==="cancel"||Y==="close"){ee(),A();return}Y==="confirm"&&Q(u)});const X=document.getElementById("wipe-confirm-input"),le=document.getElementById("wipe-confirm-btn");X==null||X.addEventListener("input",()=>{le&&(le.disabled=X.value.trim()!==u)}),X==null||X.focus()}async function Q(u){const v=document.getElementById("wipe-confirm-btn");v&&(v.disabled=!0,v.textContent="Wiping…");let O;try{O=await ta(s,u)}catch(X){const le=We();if(le){const Y=document.createElement("p");Y.className="error small",Y.textContent=`Wipe failed: ${D(X)}${Z(X)}`,le.appendChild(Y)}v&&(v.disabled=!1,v.textContent=`Wipe ${u}`);return}b(u,O)}function b(u,v){const O=E(u),X=S=>{var M;return((M=E(S))==null?void 0:M.label)??S},le=[];le.push(v.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const S of v.report.VolumesRemoved??[])le.push(`Volume ${S} deleted.`);for(const S of v.report.VolumesAbsent??[])le.push(`Volume ${S} was already gone.`);v.report.Recreated&&le.push("Container re-created from your saved configuration.");const Y=(v.report.Cascaded??[]).map(X),fe=(v.report.CascadeSkipped??[]).map(X);he(`
        <h2>${a((O==null?void 0:O.label)??u)} wiped</h2>
        <ul class="plain-list">${le.map(S=>`<li>${a(S)}</li>`).join("")}</ul>
        ${Y.length?`<p class="ok">Restarted in front of it: ${a(Y.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${fe.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(fe.join(", "))}.</p>`:""}
        ${v.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(v.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,S=>{(S==="close"||S==="cancel")&&(ee(),A())})}async function w(u,v){const O=await Ke(v),X=u.textContent;u.textContent=O?"Copied!":"Copy failed",setTimeout(()=>{r||(u.textContent=X)},1500)}function D(u){return u instanceof Error?u.message:String(u)}function Z(u){return u instanceof Re&&u.hint?` — ${u.hint}`:""}return()=>{r=!0,I==null||I(),ee()}}const ot=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ye=8545,Ze=5052,Xe=30303,xa=[369,943,1],Lt={369:"default",943:"practise here first"};function Ta(n,s){let r=!1;const e={targetId:s,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${a(s)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ge()}</div>`;const d=n.querySelector("#wizard-body"),l=n.querySelector("#wizard-footer");Ee(n,(b,w)=>{we(b,w)}),ft(n,(b,w)=>{b==="exec-select"?e.execId=w:b==="beacon-select"&&(e.beaconId=w),p()}),n.addEventListener("change",b=>{const w=b.target;w instanceof HTMLInputElement&&(w.id==="data-dir-input"?(Se(),z()):w.id==="checkpoint-toggle"?(e.checkpoint=w.checked,p()):w.id==="exec-snapshot-toggle"&&(e.execSnapshot=w.checked,p()))}),y();async function y(){try{const[b,w]=await Promise.all([Be(),De()]);if(r)return;e.catalog=b;const D=w.find(Z=>Z.id===s);D!=null&&D.wire&&(e.chainId=D.wire.ChainID,e.execId=D.wire.ExecID,e.beaconId=D.wire.BeaconID,e.archive=D.wire.Archive,D.wire.ExecHTTPPort&&(e.execHTTPPort=String(D.wire.ExecHTTPPort)),D.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(D.wire.BeaconHTTPPort)),D.wire.ExecP2PPort&&(e.execP2PPort=String(D.wire.ExecP2PPort)),D.wire.RPCBindAddr&&(e.rpcBindAddr=D.wire.RPCBindAddr)),p()}catch(b){if(r)return;e.loadError=String(b instanceof Error?b.message:b),p()}}function p(){if(e.loadError){d.innerHTML=`<p class="error">Failed to load: ${a(e.loadError)}</p>`;return}e.catalog&&(d.innerHTML=`
      ${Q(e.step)}
      ${j()}
    `,I())}function I(){var w;const b=(w=e.catalog)==null?void 0:w.networks.find(D=>D.ChainID===e.chainId);l.innerHTML=b?ge(b.Name,b.LearnURL):ge()}function j(){switch(e.step){case"network":return V();case"clients":return L();case"mode":return pe();case"review":return be();case"run":return re()}}function V(){const b=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${xa.map(D=>{const Z=b.networks.find(O=>O.ChainID===D);if(!Z)return"";const u=e.chainId===D,v=Lt[D]?J(Lt[D],D===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${u?"selected":""}" data-action="pick-network" data-chain-id="${D}" type="button">
          <h3>${a(Z.Name)} <span class="muted">(chain ${D})</span></h3>
          ${v}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function L(){const b=e.catalog,w=b.networks.find(u=>u.ChainID===e.chainId);if(!w)return'<p class="error">Unknown network.</p>';(e.execId===null||!w.ExecClients.includes(e.execId))&&(e.execId=w.ExecClients[0]??null),(e.beaconId===null||!w.BeaconClients.includes(e.beaconId))&&(e.beaconId=w.BeaconClients[0]??null);const D=w.ExecClients.map(u=>U(u,b)),Z=w.BeaconClients.map(u=>U(u,b));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${a(w.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${ut("exec-select",D,e.execId)}
        </label>
        ${de(e.execId,b)}
        <label>
          Beacon client
          ${ut("beacon-select",Z,e.beaconId)}
        </label>
        ${de(e.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function _(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const G=1.1,A=.5,E="Valve reth snapshot",f="rough estimate";function g(b){return b.SnapshotSizeTB}function x(b){return b.SnapshotSizeTB*A}function F(b){return`<p class="muted small">${_(g(b))} is the measured size of Valve's reth snapshot for ${a(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function K(b){return{archive:g(b)*1e12*G,full:x(b)*1e12*G}}function te(b,w){if(!b)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${a(w)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${a(w)}</code>: ${a(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==w)return"";const D=K(b),Z=e.freeBytes>=D.archive,u=e.freeBytes>=D.full,v=`<p class="muted small">Free at <code>${a(w)}</code>: <strong>${Me(e.freeBytes)}</strong> — archive ${Z?"fits":"won't fit"} (${_(g(b))}, ${E}), full ${u?"fits":"won't fit"} (${_(x(b))}, ${f}).</p>`;let O="";return e.downgradeNote?O=`<p class="banner banner-warn">${a(e.downgradeNote)}</p>`:u||(O=`<p class="banner banner-warn">Neither full (${_(x(b))}, ${f}) nor archive (${_(g(b))}, ${E}) fits the free space here — choose a location with more room.</p>`),v+O}function ce(b,w){if(e.downgradeNote=null,!b||e.freeBytes===null)return;const D=K(b);e.archive&&e.freeBytes<D.archive&&e.freeBytes>=D.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${w} for archive (${_(g(b))}, ${E}) — switched to Full (${_(x(b))}, ${f}). Pick a location with more room to run archive.`)}async function z(){var D;if(e.chainId===null)return;const b=(D=e.catalog)==null?void 0:D.networks.find(Z=>Z.ChainID===e.chainId),w=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,p();try{const{freeBytes:Z}=await qn(e.targetId,w);if(r)return;e.freeBytes=Z,e.probedPath=w,ce(b,w)}catch(Z){if(r)return;e.freeBytes=null,e.probedPath=w,e.diskError=String(Z instanceof Error?Z.message:Z)}e.diskProbing=!1,p()}function ie(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function U(b,w){const D=w.clients.find(Z=>Z.id===b);return{value:b,label:D?`${D.id} — ${ne(D.repo)}`:b}}function ne(b){const w=b.split("/");return w.length>=4?w[3]:b}function de(b,w){const D=b?w.clients.find(u=>u.id===b):void 0;if(!D)return"";const Z=D.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(D.repo)}" target="_blank" rel="noopener noreferrer">${a(Z)}</a></p>`}function pe(){var X,le,Y;const b=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",w=(X=e.catalog)==null?void 0:X.networks.find(fe=>fe.ChainID===e.chainId),D=((Y=(le=e.catalog)==null?void 0:le.clients.find(fe=>fe.id===e.execId))==null?void 0:Y.snapshotSupported)??!1,Z=w?`${_(x(w))} (${f})`:"Smaller",u=w?`${_(g(w))} (${E})`:"Much larger",v=w?` on ${a(w.Name)}`:"",O=w?e.checkpoint?w.SyncLabel:w.GenesisSyncLabel:"";return`
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
          ${w?`<p class="sync-estimate">⏱ Estimated initial sync${v}: <strong>${a(O)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${a((w==null?void 0:w.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${a((w==null?void 0:w.CheckpointURL)??"")}" value="${a(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${a(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${D?`
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
              <tr><th>Approx. disk footprint${v}</th><td class="yes">${Z}</td><td class="limited">${u}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${w?F(w):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${u}${w?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${Z}${w?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(b)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(b)}" value="${a(e.dataDir)}" />
          </label>
          ${te(w,e.dataDir||b)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(b)}/jwt.hex" value="${a(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Ye})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Ye}" value="${a(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${a(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ze})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ze}" value="${a(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${a(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Xe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Xe}" value="${a(e.execP2PPort)}" />
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
    `}function be(){const w=e.catalog.networks.find(S=>S.ChainID===e.chainId),D=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,Z=e.jwtPath||`${D}/jwt.hex`,u=ot.map(S=>`<li>${a(S.title)}</li>`).join(""),v=B(e.execHTTPPort,Ye),O=B(e.beaconHTTPPort,Ze),X=B(e.execP2PPort,Xe),le=v||O||X?`<tr><th>Non-default ports</th><td>${[v?`exec HTTP ${v}`:null,O?`beacon HTTP ${O}`:null,X?`exec p2p ${X}`:null].filter(S=>S!==null).map(a).join(", ")}</td></tr>`:"",{addr:Y}=$(e.rpcBindAddr),fe=Y?`<tr><th>RPC bind address</th><td><code>${a(Y)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${a(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${a((w==null?void 0:w.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${a(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${a(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${a(D)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${a(Z)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${a(e.checkpointUrl||(w==null?void 0:w.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${le}
            ${fe}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${u}</ol>
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function re(){const w=e.catalog.networks.find(Y=>Y.ChainID===e.chainId),D=w==null?void 0:w.LearnURL,Z=new Set(e.events.filter(Y=>Y.done).map(Y=>Y.stepId)),u=new Set(e.events.filter(Y=>Y.err).map(Y=>Y.stepId)),v=new Map;for(const Y of e.events){if(!Y.line)continue;const fe=v.get(Y.stepId)??[];fe.push(Y.line),v.set(Y.stepId,fe)}const O=ot.map(Y=>{var N;const fe=Z.has(Y.id),S=u.has(Y.id),M=S?J("failed","bad"):fe?J("done","ok"):J("pending","neutral"),P=(v.get(Y.id)??[]).slice(-5),q=(N=e.events.find(se=>se.stepId===Y.id&&se.err))==null?void 0:N.err,H=Y.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${D?` <a href="${a(D)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${fe?"step-done":""} ${S?"step-error":""}">
          <div class="step-head">${M} <strong>${a(Y.title)}</strong></div>
          ${H}
          ${P.length?`<pre class="step-log">${P.map(se=>a(se)).join(`
`)}</pre>`:""}
          ${q?`<p class="error small">${a(q)}</p>`:""}
        </li>
      `}).join(""),X=e.events.some(Y=>Y.err),le=ot.every(Y=>Z.has(Y.id))||e.events.some(Y=>Y.stepId==="handshake"&&Y.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${O}</ol>
        ${le&&!X?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        ${X?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function we(b,w){switch(b){case"pick-network":e.chainId=Number(w.dataset.chainId),e.execId=null,e.beaconId=null,p();break;case"goto-network":e.step="network",p();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",p();break;case"goto-mode":e.step="mode",p(),z();break;case"goto-review":if(Se(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){p();break}e.step="review",p();break;case"start-setup":W();break}}function Se(){const b=n.querySelectorAll('input[name="mode"]');for(const Y of Array.from(b))Y.checked&&(e.archive=Y.value==="archive");const w=n.querySelector("#data-dir-input"),D=n.querySelector("#jwt-path-input");w&&(e.dataDir=w.value.trim()),D&&(e.jwtPath=D.value.trim());const Z=n.querySelector("#exec-http-port-input"),u=n.querySelector("#beacon-http-port-input"),v=n.querySelector("#exec-p2p-port-input");Z&&(e.execHTTPPort=Z.value.trim()),u&&(e.beaconHTTPPort=u.value.trim()),v&&(e.execP2PPort=v.value.trim());const O=n.querySelector("#rpc-bind-addr-input");O&&(e.rpcBindAddr=O.value.trim());const X=n.querySelector("#checkpoint-url-input");X&&(e.checkpointUrl=X.value.trim());const le=n.querySelector("#snapshot-key-input");le&&(e.snapshotKey=le.value.trim()),e.execHTTPPortError=T(e.execHTTPPort).error??null,e.beaconHTTPPortError=T(e.beaconHTTPPort).error??null,e.execP2PPortError=T(e.execP2PPort).error??null,e.rpcBindAddrError=$(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?ie(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function $(b){if(!b)return{};const w=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return w?w.slice(1).every(D=>Number(D)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const h=/^\d+$/;function T(b){if(!b)return{};if(!h.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const w=Number(b);return!Number.isInteger(w)||w<1||w>65535?{error:"Port must be between 1 and 65535."}:{port:w}}function B(b,w){const{port:D}=T(b);if(!(D===void 0||D===w))return D}async function W(){var v;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(v=e.streamStop)==null||v.call(e),e.streamStop=null,p();const b={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(b.DataDir=e.dataDir),e.jwtPath&&(b.JWTPath=e.jwtPath);const w=B(e.execHTTPPort,Ye),D=B(e.beaconHTTPPort,Ze),Z=B(e.execP2PPort,Xe);w!==void 0&&(b.ExecHTTPPort=w),D!==void 0&&(b.BeaconHTTPPort=D),Z!==void 0&&(b.ExecP2PPort=Z);const{addr:u}=$(e.rpcBindAddr);u!==void 0&&(b.RPCBindAddr=u),e.checkpoint?e.checkpointUrl&&(b.CheckpointURL=e.checkpointUrl):b.NoCheckpoint=!0,e.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=e.snapshotKey);try{await Wn(e.targetId,b)}catch(O){if(!(O instanceof Re&&O.status===409)){e.starting=!1,e.startError=String(O instanceof Error?O.message:O),p();return}}e.starting=!1,e.step="run",p(),e.streamStop=ze(e.targetId,O=>{r||(e.events.push(O),e.step==="run"&&p())})}function Q(b){const w=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],Z=w.map(u=>u.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${w.map((u,v)=>`<li class="${v===Z?"current":v<Z?"past":"future"}">${a(u.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;r=!0,(b=e.streamStop)==null||b.call(e)}}function Ia(n,s){let r=!1;const e=new Map;n.innerHTML=`<h1>${a(s)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const d=n.querySelector("#machine-body");Ee(n,(L,_)=>{L==="toggle-section"&&j(_.dataset.section??"")}),l();async function l(){let L,_;try{const[G,A]=await Promise.all([De(),Be()]);L=G.find(E=>E.id===s),_=A}catch(G){if(r)return;d.innerHTML=`<p class="error">Failed to load machine: ${a(String(G))}</p>`;return}if(!r){if(!L){location.hash="#/targets";return}y(L,_)}}function y(L,_){const G=L.mode==="local"?"this machine":"SSH",A=L.mode==="ssh"&&L.ssh?`${a(L.ssh.User)}@${a(L.ssh.Host)}`:G;d.innerHTML=`
      <p class="muted">${A}</p>
      <p>${p(L,_)}</p>
      <div class="machine-sections">
        ${V.map(E=>I(E,L,_)).join("")}
      </div>
      ${ge()}
    `}function p(L,_){const G=L.wire;if(!G)return J("not set up","neutral");const A=_.networks.find(f=>f.ChainID===G.ChainID),E=A?A.Name:`chain ${G.ChainID}`;return`${J(E,"ok")} ${J(G.ExecID,"neutral")} ${J(G.BeaconID,"neutral")}${G.Archive?" "+J("archive","warn"):""}`}function I(L,_,G){return`
      <section class="card machine-section" data-section-card="${a(L.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(L.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(L.title)}</span>
          <span class="machine-section-status">${L.status(_,G)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(L.key)}" hidden></div>
      </section>
    `}function j(L){const _=V.find(g=>g.key===L);if(!_)return;const G=n.querySelector(`[data-section-card="${L}"]`),A=n.querySelector(`[data-section-body="${L}"]`),E=n.querySelector(`.machine-section-head[data-section="${L}"]`);if(!G||!A||!E)return;const f=A.hidden;if(f&&!e.has(L)){const g=document.createElement("div");A.appendChild(g),e.set(L,_.mount(g))}A.hidden=!f,G.classList.toggle("open",f),E.setAttribute("aria-expanded",String(f))}const V=[{key:"setup",title:"Setup",status:L=>L.wire?J("set up","ok"):J("not set up","neutral"),mount:L=>Ta(L,s)},{key:"dashboard",title:"Dashboard",status:L=>L.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:L=>ga(L,s)},{key:"logs",title:"Logs",status:L=>L.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:L=>$a(L,s)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:L=>Sa(L,s)}];return()=>{r=!0;for(const L of e.values())try{L()}catch{}e.clear()}}function Ea(n){let s;try{s=new URL(n).hostname}catch{return"endpoint"}if(!s)return"endpoint";if(s==="localhost"||/^[0-9.]+$/.test(s)||/^\[.*\]$/.test(s))return s;const r=s.split(".").filter(Boolean);return r.length<=1?s:r[r.length-2]}function _t(n){var e;if(!n)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const s=n.actions??[];if(n.blocked)return{tone:"blocked",label:"Unavailable",sub:n.blocked,actions:s,blocked:n.blocked};const r=((e=n.networks)==null?void 0:e.length)??0;return n.status.State==="running"?{tone:"on",label:"Running",sub:`${r} network${r===1?"":"s"} served`,actions:s}:{tone:"off",label:"Stopped",sub:r?`${r} network${r===1?"":"s"} configured`:"Press to start",actions:s}}function _e(n){if(!n.running)return"off";if(!n.serviceable)return"frequent";const s=n.slowRate??0;return s>.4?"frequent":s>=.1?"occasional":"stable"}const Pa="0.5";function Kt(n){if(!n||n.count<=0||!n.buckets||n.buckets.length===0)return;const s=n.buckets.find(e=>e.le===Pa);if(!s)return;const r=n.count-s.count;return Math.max(0,Math.min(1,r/n.count))}function Ra(n){if(!n||n.length===0)return null;let s=0;const r=new Map;for(const e of n){s+=e.count;for(const d of e.buckets??[])r.set(d.le,(r.get(d.le)??0)+d.count)}return{count:s,mean:null,buckets:[...r.entries()].map(([e,d])=>({le:e,count:d}))}}function mt(n){const s=Kt(Ra(n.methods));if(s!==void 0)return s;if(n.received>0)return Math.max(0,Math.min(1,n.failed/n.received))}function bt(n,s){var e;const r=(e=n==null?void 0:n.endpoints)==null?void 0:e.find(d=>d.upstream===s);return Kt(r??null)}const La=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function yt(n){return La.map(({key:s,label:r,hot:e})=>{const d=n[s]==="supported";return{key:s,label:r,lit:d,hot:!!e&&d}})}function Na(n,s,r){const e=n.Networks??[],d=e.findIndex(p=>p.ChainID===s),l={ChainID:s,Upstreams:r},y=d===-1?[...e,l]:e.map((p,I)=>I===d?l:p);return{...n,Networks:y}}function Aa(n,s){const r=n.Networks??[];return{...n,Networks:r.filter(e=>e.ChainID!==s)}}function rt(n,s,r){const e=n.Networks??[],d=e.findIndex(j=>j.ChainID===s);if(d===-1)return{...n,Networks:[...e,{ChainID:s,Upstreams:[r]}]};const l=e[d],y=l.Upstreams.findIndex(j=>j.ID===r.ID),p=y===-1?[...l.Upstreams,r]:l.Upstreams.map((j,V)=>V===y?r:j),I={...l,Upstreams:p};return{...n,Networks:e.map((j,V)=>V===d?I:j)}}function Ba(n,s,r){const e=n.Networks??[],d=e.findIndex(p=>p.ChainID===s);if(d===-1)return{...n,Networks:e};const l=e[d],y={...l,Upstreams:l.Upstreams.filter(p=>p.ID!==r)};return{...n,Networks:e.map((p,I)=>I===d?y:p)}}function Da(n,s){if(n.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=n.filter(p=>!p.wire);if(r.length>0){const p=r.map(j=>j.id);return{level:"attention",sentence:p.length===1?"1 machine still needs setup.":`${p.length} machines still need setup.`,machines:p}}const e=s.networks??[],d=p=>{const I=e.find(j=>j.ChainID===p);return I?I.Name:`chain ${p}`},l=Ua(n.map(p=>d(p.wire.ChainID))),y=n.length===1?"machine":"machines";return{level:"ok",sentence:`All ${n.length} ${y} healthy — ${Ma(l)}.`,machines:[]}}function Ha(n,s){const r=s.machines.length?` <span class="verdict-machines">${s.machines.map(e=>`<a href="#/setup/${encodeURIComponent(e)}">${a(e)}</a>`).join(" ")}</span>`:"";n.innerHTML=`
    <div class="verdict-line verdict-${s.level}">
      ${J(s.level==="ok"?"OK":"Attention",s.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(s.sentence)}</strong>${r}
    </div>
  `}function Ua(n){return[...new Set(n)]}function Ma(n){return n.length<=1?n[0]??"":n.length===2?`${n[0]} and ${n[1]}`:`${n.slice(0,-1).join(", ")} and ${n[n.length-1]}`}const Oa=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function Nt(n){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:n,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Fa=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`,me=n=>`<svg class="p-i"><use href="#p-${n}"/></svg>`,At="run",Bt=1337;function Dt(n){let s=null,r={name:"list"},e=null,d=null,l=null,y=null,p=null,I=[],j=null,V=null,L=!1,_=null,G=!1,A=null,E=!1,f=null,g=null,x=null,F=null,K=null,te=!1,ce="";n.innerHTML=Fa+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const z=n.querySelector("#p-card");async function ie(){try{const S=await ht();s=ja(S.gateways),e=null}catch(S){e=ve(S)}U()}function U(){z.innerHTML=ne()}function ne(){return e?qa(e):r.name==="network"?Xa(s,r.chainId,{caps:j,capsBusy:L,tls:_,tlsBusy:G,tlsErr:A,copyFlash:E,error:f,netHealth:K}):r.name==="endpoint"?es(s,r.chainId,r.upstreamId,{caps:j,capsBusy:L,health:x,copyFlash:E,error:g,netHealth:K}):Wa(s,l,y,I,K)}async function de(S,M){L=!0,U();try{j=await jt(S,M),V=S}catch{j=null,V=S}L=!1,U()}async function pe(S,M){if(!(!M&&F===S&&x)){U();try{x=await ct(S),F=S}catch{x=null,F=S}U()}}function be(){var P;if(!s)return"";const S=s.status.State==="running",M=[];for(const q of s.networks??[]){const H=(P=K==null?void 0:K.networks)==null?void 0:P.find(se=>se.chainId===q.chainId),N=H?mt(H):void 0;M.push(`n${q.chainId}:${_e({running:S,serviceable:q.serviceable,slowRate:N})}`);for(const se of q.upstreams??[]){const $e=H?bt(H,se.id):void 0;M.push(`u${q.chainId}/${se.id}:${_e({running:S,serviceable:!se.problem,slowRate:$e})}`)}}return M.join("|")}async function re(){if(!s||te)return;te=!0;try{K=await ct(s.id)}catch{}te=!1;const S=be();S!==ce&&(ce=S,U())}async function we(S,M){var H;const P=(H=S.networks)==null?void 0:H.find(N=>N.chainId===M);if(await Ae({title:"Remove network",body:`Stop serving ${(P==null?void 0:P.name)??`chain ${M}`}?`,confirmLabel:"Remove",danger:!0})){f=null,U();try{await Le(S.id,Aa(S.config,M))}catch(N){f=`Could not remove the network: ${ve(N)}`,U();return}r={name:"list"},U(),await X(S.id)}}async function Se(S,M,P){var se;const q=(se=S.networks)==null?void 0:se.find($e=>$e.chainId===M),H=h(S,M,P);if(await Ae({title:"Remove endpoint",body:`Stop routing to ${(H==null?void 0:H.label)??"this endpoint"}? The gateway keeps balancing across whatever else remains on ${(q==null?void 0:q.name)??`chain ${M}`}.`,confirmLabel:"Remove",danger:!0})){g=null,U();try{await Le(S.id,Ba(S.config,M,P))}catch($e){g=`Could not remove the endpoint: ${ve($e)}`,U();return}r={name:"network",chainId:M},U(),await X(S.id)}}Ee(z,(S,M)=>{$(S,M)});async function $(S,M){if(S==="setup"){if(l)return;await le();return}if(S==="power"){if(!s||l)return;const P=_t(s);if(P.tone==="blocked")return;if(s.status.State==="running"&&P.actions.includes("stop")){await O(s.id,"stop");return}if(P.actions.includes("start")){await O(s.id,"start");return}if(P.actions.includes("create")){await X(s.id);return}return}if(S==="open-network"){r={name:"network",chainId:Number(M.dataset.chainId)},f=null,_=null,A=null,U(),s&&V!==s.id&&de(s.id,!1);return}if(S==="back"){r={name:"list"},U();return}if(S==="back-to-network"){const P=Number(M.dataset.chainId);r=Number.isFinite(P)?{name:"network",chainId:P}:{name:"list"},g=null,U();return}if(S==="add-network"){if(!s||l)return;await u(s);return}switch(S){case"gw-start":case"gw-stop":case"gw-restart":s&&!l&&await O(s.id,S.slice(3));return;case"gw-create":case"gw-recreate":s&&!l&&await X(s.id);return;case"gw-wipe":s&&!l&&await Y(s);return;case"copy-url":{const P=M.dataset.url??"";if(!P)return;await Ke(P)&&(E=!0,U(),window.setTimeout(()=>{E=!1,U()},1200));return}case"verify-tls":{if(!s||G)return;G=!0,A=null,U();try{_=await Ft(s.id)}catch(P){A=ve(P)}G=!1,U();return}case"open-endpoint":{const P=Number(M.dataset.chainId),q=M.dataset.upstreamId??"";if(!Number.isFinite(P)||!q)return;r={name:"endpoint",chainId:P,upstreamId:q},g=null,U(),s&&V!==s.id&&de(s.id,!1),s&&F!==s.id&&pe(s.id,!1);return}case"add-endpoint":{if(!s||l||r.name!=="network")return;D(s,r.chainId);return}case"remove-network":{if(!s||l||r.name!=="network")return;await we(s,r.chainId);return}case"rename-endpoint":{if(!s||l||r.name!=="endpoint")return;const P=h(s,r.chainId,r.upstreamId);if(!P)return;W(s.id,r.chainId,P.id,P.label);return}case"edit-address":{if(!s||l||r.name!=="endpoint")return;const P=h(s,r.chainId,r.upstreamId);if(!P||P.kind!=="external")return;b(s.id,r.chainId,P.id,P.endpoint);return}case"remove-endpoint":{if(!s||l||r.name!=="endpoint")return;await Se(s,r.chainId,r.upstreamId);return}case"recheck":{if(!s)return;const P=[de(s.id,!0),ie(),re()];r.name==="endpoint"&&P.push(pe(s.id,!0)),await Promise.all(P);return}default:return}}function h(S,M,P){var q,H,N;return(N=(H=(q=S.networks)==null?void 0:q.find(se=>se.chainId===M))==null?void 0:H.upstreams)==null?void 0:N.find(se=>se.id===P)}function T(S,M,P){var q,H;return(H=(q=S.config.Networks)==null?void 0:q.find(N=>N.ChainID===M))==null?void 0:H.Upstreams.find(N=>N.ID===P)}function B(S){const M=We();if(!M)return;const P=document.createElement("p");P.className="error small",P.textContent=S,M.appendChild(P)}function W(S,M,P,q){he(`
        <h2>Rename endpoint</h2>
        <label>
          Name
          <input type="text" id="ep-rename-input" autocomplete="off" spellcheck="false" value="${a(q)}" />
        </label>
        <p class="muted small">Clear it to fall back to the automatic name.</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-rename-save">Save</button>
        </div>
      `,N=>{if(N==="cancel"){ee();return}N==="save"&&Q(S,M,P)});const H=document.getElementById("ep-rename-input");H==null||H.focus(),H==null||H.select()}async function Q(S,M,P){if(!s)return;const q=T(s,M,P);if(!q){ee();return}const H=document.getElementById("ep-rename-input"),N=document.getElementById("ep-rename-save"),se=(H==null?void 0:H.value.trim())??"";H&&(H.disabled=!0),N&&(N.disabled=!0,N.textContent="Saving…");const $e={...q,Name:se||void 0};try{await Le(S,rt(s.config,M,$e))}catch(Te){B(`Could not rename the endpoint: ${ve(Te)}`),H&&(H.disabled=!1),N&&(N.disabled=!1,N.textContent="Save");return}ee(),await X(S)}function b(S,M,P,q){he(`
        <h2>Edit endpoint address</h2>
        <p class="muted small">http://, https://, ws:// or wss://.</p>
        <label>
          URL
          <input type="text" id="ep-addr-input" autocomplete="off" spellcheck="false" value="${a(q)}" placeholder="https://rpc.example.com" />
        </label>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-addr-save">Save</button>
        </div>
      `,N=>{if(N==="cancel"){ee();return}N==="save"&&w(S,M,P)});const H=document.getElementById("ep-addr-input");H==null||H.focus(),H==null||H.select()}async function w(S,M,P){if(!s)return;const q=document.getElementById("ep-addr-input"),H=document.getElementById("ep-addr-save"),N=(q==null?void 0:q.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(N)){B("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}const se=T(s,M,P);if(!se){ee();return}q&&(q.disabled=!0),H&&(H.disabled=!0,H.textContent="Saving…");const $e={...se,Endpoint:N};try{await Le(S,rt(s.config,M,$e))}catch(Te){B(`Could not save the address: ${ve(Te)}`),q&&(q.disabled=!1),H&&(H.disabled=!1,H.textContent="Save");return}ee(),await X(S)}function D(S,M){var P;he(`
        <h2>Add an endpoint by URL</h2>
        <p class="muted small">http://, https://, ws:// or wss://.</p>
        <label>
          Endpoint
          <input type="text" id="ep-add-input" autocomplete="off" spellcheck="false" placeholder="https://rpc.example.com" />
        </label>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add" id="ep-add-save">Add endpoint</button>
        </div>
      `,q=>{if(q==="cancel"){ee();return}q==="add"&&Z(S.id,M)}),(P=document.getElementById("ep-add-input"))==null||P.focus()}async function Z(S,M){if(!s)return;const P=document.getElementById("ep-add-input"),q=document.getElementById("ep-add-save"),H=(P==null?void 0:P.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(H)){B("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}P&&(P.disabled=!0),q&&(q.disabled=!0,q.textContent="Adding…");const N={ID:crypto.randomUUID(),Kind:"external",Endpoint:H,Local:!1,RecentOnly:!1,Name:Ea(H)};try{await Le(S,rt(s.config,M,N))}catch(se){B(`Could not add the endpoint: ${ve(se)}`),P&&(P.disabled=!1),q&&(q.disabled=!1,q.textContent="Add endpoint");return}ee(),await X(S)}async function u(S){l="add-network",y=null,U();let M;try{M=(await Be()).networks??[]}catch(H){l=null,y=`Could not load the network catalog: ${ve(H)}`,U();return}l=null,U();const P=new Set((S.networks??[]).map(H=>H.chainId)),q=M.filter(H=>!P.has(H.ChainID)).map(H=>({chainId:H.ChainID,name:H.Name}));if(P.has(Bt)||q.push({chainId:Bt,name:"Devnet"}),q.length===0){y="Every network valve's catalog knows about is already configured on this gateway.",U();return}he(`
        <h2>Add a network</h2>
        <ul class="plain-list rpc-picker">
          ${q.map(H=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="pick:${H.chainId}">
                <span>${a(H.name)}</span>
                <span class="muted small">chain ${H.chainId}</span>
              </button>
            </li>`).join("")}
        </ul>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,H=>{if(H==="cancel"){ee();return}if(H.startsWith("pick:")){const N=Number.parseInt(H.slice(5),10);if(!Number.isFinite(N))return;ee(),v(S.id,N)}})}async function v(S,M){if(!s||l)return;l="create",y=null,U();let P;try{P=((await dt(S,M)).endpoints??[]).filter(N=>!N.alreadyAdded).map(N=>N.url)}catch(H){l=null,y=`Could not read valve's known set for chain ${M}: ${ve(H)}`,U();return}if(P.length===0){l=null,y=`valve has no measured endpoints for chain ${M} yet, so there was nothing to add.`,U();return}const q=P.map((H,N)=>({ID:`public-${M}-${N+1}`,Kind:"external",Endpoint:H,Local:!1,RecentOnly:!1}));try{await Le(S,Na(s.config,M,q))}catch(H){l=null,y=`Could not add the network: ${ve(H)}`,U();return}l=null,await X(S,()=>{r={name:"network",chainId:M},U()})}async function O(S,M){if(!l){l=M,y=null,U();try{await qt(S,M)}catch(P){y=`${M} failed: ${ve(P)}`}l=null,await ie()}}async function X(S,M){if(l)return;l="create",y=null,U();let P;try{P=await lt(S)}catch(q){y=ve(q),l=null,U();return}p==null||p(),p=ze(P.targetId,q=>{(q.err||q.stepId===At&&q.done)&&(p==null||p(),p=null,l=null,q.err&&(y=`Provisioning failed: ${q.err}`),ie().then(()=>{q.err||M==null||M()}))})}async function le(){if(l)return;l="setup",y=null,I=[],U();const S=N=>{I=[...I,N],U()},M=(N,se)=>{l=null,y=se?`${N} — ${se}`:N,U()};S("Preparing your endpoint…");try{(await De()).some(se=>se.id==="local")||await it({id:"local",mode:"local"})}catch(N){M(`Could not register this machine: ${ve(N)}`,je(N));return}try{const N=await Mt("local");if(!N.docker.reachable){M(N.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",N.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(N){M(`Could not check Docker on this machine: ${ve(N)}`,je(N));return}S("Creating the gateway…");let P="default";try{P=(await Ot({id:P,placement:{targetId:"local",backend:"docker"},config:Nt([])})).id}catch(N){M(`Could not create the gateway: ${ve(N)}`,je(N));return}S("Adding Ethereum and PulseChain endpoints…");const q=[];for(const{chainId:N}of Oa)try{const $e=((await dt(P,N)).endpoints??[]).filter(Te=>!Te.alreadyAdded).map(Te=>Te.url);if($e.length===0)continue;q.push({ChainID:N,Upstreams:$e.map((Te,tt)=>({ID:`public-${N}-${tt+1}`,Kind:"external",Endpoint:Te,Local:!1,RecentOnly:!1}))})}catch(se){M(`Could not read valve's set for chain ${N}: ${ve(se)}`,je(se));return}if(q.length===0){M("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await Le(P,Nt(q))}catch(N){M(`Could not save the endpoints: ${ve(N)}`,je(N));return}S("Starting the gateway… the first run pulls the eRPC and Caddy images.");let H;try{H=await lt(P)}catch(N){M(`Could not start the gateway: ${ve(N)}`,je(N));return}p==null||p(),p=ze(H.targetId,N=>{const se=N.err?`${N.stepId}: ${N.err}`:N.line?`${N.stepId}: ${N.line}`:`${N.stepId}: done`;S(se),(N.err||N.stepId===At&&N.done)&&(p==null||p(),p=null,l=null,N.err&&(y=`Provisioning failed: ${N.err}`),I=[],ie())})}async function Y(S){if(await Ae({title:`Wipe ${S.label}`,body:`This destroys ${S.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){l="wipe",y=null,U();try{const P=await Wt(S.id);P.error&&(y=P.error)}catch(P){y=`wipe failed: ${ve(P)}`}l=null,await ie()}}let fe=!1;return ie().then(()=>{fe||(ce=be(),d=window.setInterval(()=>{re()},5e3))}),()=>{fe=!0,d&&window.clearInterval(d),p==null||p()}}function ja(n){return!n||n.length===0?null:n.find(s=>s.placement.targetId==="local")??n[0]}function ve(n){return n instanceof Error?n.message:String(n)}function je(n){return n instanceof Re?n.hint:void 0}function qa(n){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(n)}</div>`}function Wa(n,s,r,e,d){var p;if(n===null)return _a(s,r,e);const l=_t(n),y=(p=n==null?void 0:n.networks)!=null&&p.length?n.networks.map((I,j)=>Ya(n,I,j>0,d)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${a(l.sub)}</span>
    </div>
    <div class="p-band">
      ${Ga(n,l,s,r)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${y}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${me("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function _a(n,s,r){const e=n==="setup",d=s?`<div class="p-emptyerr">${a(s)}</div>`:"",l=r.length?`<div class="p-setup-log" aria-live="polite">${r.map(y=>`<div>${a(y)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${e?" disabled":""}>
        <div class="p-pbtn off big${e?" busy":""}">${me("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${d}
      ${l}
    </div>
  `}function Ka(n,s){return s.tone==="blocked"?null:n.status.State==="running"&&s.actions.includes("stop")?"stop":s.actions.includes("start")?"start":s.actions.includes("create")?"create":null}const Va={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},Ht={restart:"refresh",recreate:"refresh",wipe:"trash"};function Ga(n,s,r,e){const d=s.tone==="blocked"?s.blocked??"":s.sub,l=r?" busy":"",y=e?`<div class="p-ps" style="color:var(--red)">${a(e)}</div>`:"",p=s.tone==="blocked"&&(n!=null&&n.hint)?`<div class="p-ps">${a(n.hint)}</div>`:"",I=`
    <div class="p-power${l}" data-action="power">
      <div class="p-pbtn ${s.tone}">${me("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(s.label)}</div>
        <div class="p-ps"${s.tone==="blocked"?' style="color:var(--red)"':""}>${a(d)}</div>
        ${p}
        ${y}
      </div>
    </div>
  `,j=n?za(n,s,r):"";return I+j}function za(n,s,r){const e=Ka(n,s),d=(n.actions??[]).filter(y=>y!==e);return d.length===0?"":`<div class="p-chips">${d.map(y=>{const p=Va[y]??y,I=Ht[y]?me(Ht[y]):"";return`<button type="button" class="p-chip${y==="wipe"?" danger":""}" data-action="gw-${y}" data-gid="${a(n.id)}"${r?" disabled":""}>${I}${a(p)}</button>`}).join("")}</div>`}const vt={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function Ja(n){return n.map(s=>`<svg class="p-i${s.hot?" hot":s.lit?" on":""}"><use href="#p-${vt[s.key]}"/></svg>`).join("")}function Ya(n,s,r,e){var I;const d=(I=e==null?void 0:e.networks)==null?void 0:I.find(j=>j.chainId===s.chainId),l=d?mt(d):void 0,y=_e({running:n.status.State==="running",serviceable:s.serviceable,slowRate:l}),p=yt({});return`
    <div class="p-row${r?" p-rowdiv":""}" data-action="open-network" data-chain-id="${s.chainId}">
      <span class="p-lead"><span class="p-dot ${y}"></span></span>
      <span class="p-nm">${a(s.name)}</span>
      <span class="p-caps">${Ja(p)}</span>
      <span class="p-chev">${me("chevR")}</span>
    </div>
  `}function Vt(n,s){var r;return s==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(r=(n.capabilities??[]).find(e=>e.key===s))==null?void 0:r.status}function Za(n,s,r){const e=((n==null?void 0:n.endpoints)??[]).filter(l=>l.chainId===s&&r.includes(l.upstream)),d={};for(const l of["http","ws","archive","trace"])e.some(y=>Vt(y,l)==="supported")&&(d[l]="supported");return d}function Xa(n,s,r){var ce,z,ie;const e=(ce=n==null?void 0:n.networks)==null?void 0:ce.find(U=>U.chainId===s);if(!n||!e)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${me("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${s}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;const d=n.status.State==="running",l=(ie=(z=r.netHealth)==null?void 0:z.networks)==null?void 0:ie.find(U=>U.chainId===s),y=l?mt(l):void 0,p=_e({running:d,serviceable:e.serviceable,slowRate:y}),I=e.upstreams??[],j=r.tls??n.tls.verification??null,V=(j==null?void 0:j.ok)===!0,L=r.tlsBusy?"Verifying…":V?`Verified ${j?new Date(j.at).toLocaleString():""}`:"Verify HTTPS now",_=r.tlsErr?`<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${a(r.tlsErr)}</div>`:"",G=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${V?"green":"dim"}" data-action="verify-tls" title="${a(L)}">${me("lock")}</span>
          <span class="p-ic ${r.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(e.url??"")}" title="Copy the gateway URL">${me("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${a(e.url||"—")}</div>
      ${_}
    </div>
  `,A=I.map((U,ne)=>{const de=l?bt(l,U.id):void 0,pe=_e({running:d,serviceable:!U.problem,slowRate:de});return`
        <div class="p-row${ne>0?" p-rowdiv":""}" data-action="open-endpoint" data-chain-id="${e.chainId}" data-upstream-id="${a(U.id)}">
          <span class="p-lead"><span class="p-dot ${pe}"></span></span>
          <span class="p-nm">${a(U.label)}</span>
          <span class="p-chev">${me("chevR")}</span>
        </div>
      `}).join(""),E=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${I.length}</span></div>
      ${A}
      <div class="p-row${I.length>0?" p-rowdiv":""} addr" data-action="add-endpoint">
        <span class="p-lead">${me("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `,f=Za(r.caps,s,I.map(U=>U.id)),g=yt(f),x=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${r.capsBusy&&!r.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':`<div class="p-caprow">${g.map(U=>`<span class="p-capitem${U.lit?" lit":""}">${me(vt[U.key])}${a(U.label)}</span>`).join("")}</div>`}
    </div>
  `,F=d?e.serviceable?"Healthy":"Unserviceable":"Stopped",K=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${me("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${p}"></span> ${a(F)}</span></div>
    </div>
  `,te=r.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(r.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${me("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${p}"></span> <span class="p-nmtxt">${a(e.name)}</span></span>
    </div>
    ${G}
    ${E}
    ${x}
    ${K}
    ${te}
    <div class="p-band p-remove" data-action="remove-network">${me("trash")} Remove network</div>
  `}function Qa(n,s,r){const e=((n==null?void 0:n.endpoints)??[]).find(l=>l.chainId===s&&l.upstream===r);if(!e)return{};const d={};for(const l of["http","ws","archive","trace"])Vt(e,l)==="supported"&&(d[l]="supported");return d}function es(n,s,r,e){var K,te,ce,z,ie;const d=(K=n==null?void 0:n.networks)==null?void 0:K.find(U=>U.chainId===s),l=(te=d==null?void 0:d.upstreams)==null?void 0:te.find(U=>U.id===r);if(!n||!d||!l)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${me("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Endpoint</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This endpoint is no longer configured.</div>
    `;const y=n.status.State==="running",p=(z=(ce=e.netHealth)==null?void 0:ce.networks)==null?void 0:z.find(U=>U.chainId===s),I=p?bt(p,r):void 0,j=_e({running:y,serviceable:!l.problem,slowRate:I}),V=l.kind==="external",L=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Address</span>
        <span class="p-acts"><span class="p-ic ${e.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(l.endpoint)}" title="Copy the endpoint URL">${me("copy")}</span></span>
      </div>
      <div class="p-gwurl"${V?' data-action="edit-address" style="cursor:text"':""}>${a(l.endpoint||"—")}</div>
    </div>
  `,_=Qa(e.caps,s,r),G=yt(_),A=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${e.capsBusy&&!e.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':`<div class="p-caprow">${G.map(U=>`<span class="p-capitem${U.lit?" lit":""}">${me(vt[U.key])}${a(U.label)}</span>`).join("")}</div>`}
    </div>
  `,E=y?l.problem?l.problem:"Healthy":"Stopped",f=(((ie=e.health)==null?void 0:ie.endpoints)??[]).find(U=>U.chainId===s&&U.upstream===r),g=f&&f.scored&&f.headLag>0?`<div class="p-srow"><span class="p-k">Chain head</span><span class="p-v">behind ${ye(f.headLag)} block${f.headLag===1?"":"s"}</span></div>`:"",x=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${me("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${j}"></span> ${a(E)}</span></div>
      ${g}
    </div>
  `,F=e.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(e.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${me("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${j}"></span> <span class="p-nmtxt">${a(l.label)}</span> <span class="p-pen" data-action="rename-endpoint">${me("pencil")}</span></span>
    </div>
    ${L}
    ${A}
    ${x}
    ${F}
    <div class="p-band p-remove" data-action="remove-endpoint">${me("trash")} Remove endpoint</div>
  `}function ts(n,s){let r=!1,e=[],d=null,l=!1,y=!1;n.innerHTML=`<h1>Security: ${a(s)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ge()}</div>`;const p=n.querySelector("#sec-body"),I=n.querySelector("#sec-footer");Ee(n,(A,E)=>{var f;if(A==="rerun")V();else if(A==="toggle")(f=E.closest(".check-item"))==null||f.classList.toggle("expanded");else if(A==="copy"){const g=E.dataset.copy;g&&G(E,g)}}),j();async function j(){let A,E;try{const[g,x]=await Promise.all([De(),Be()]);A=g.find(F=>F.id===s),E=x}catch(g){if(r)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(g))}</p>`;return}if(r)return;if(!A){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!A.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const f=E==null?void 0:E.networks.find(g=>g.ChainID===A.wire.ChainID);f&&(I.innerHTML=ge(f.Name,f.LearnURL)),await V()}async function V(){l=!0,d=null,L();try{e=await Zn(s),y=!0}catch(A){d=String(A instanceof Error?A.message:A)}l=!1,r||L()}function L(){p.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${l?"disabled":""}>${l?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${!y&&l?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(_).join("")}</ul>`:y?'<p class="muted">No checks returned.</p>':""}
    `}function _(A){const E=A.Status==="pass"?"ok":A.Status==="fail"?"bad":A.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${J(A.Status,E)}
          <strong>${a(A.Title)}</strong>
          <span class="muted small check-detail-inline">${a(A.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${a(A.Why)}</p>
          </details>
          ${A.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(A.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(A.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function G(A,E){const f=await Ke(E),g=A.textContent;A.textContent=f?"Copied!":"Copy failed",setTimeout(()=>{r||(A.textContent=g)},1500)}return()=>{r=!0}}const ns=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],pt="VALVE_API_KEY";function as(n){return n===pt?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(n)}}</code> slot wherever an endpoint URL carries one.`}function ss(n){let s=!1,r=!1,e=!1,d=null,l=!1,y=null,p=null;const I=new Set,j=new Map;let V="",L="";n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ge()}`;const _=n.querySelector("#settings-body");Ee(n,(x,F)=>{if(x==="save"&&g(),x==="clear-key"){if(!y)return;r=!0;const K=n.querySelector("#ai-key");K&&(K.value=""),f(y)}if(x==="clear-provider-key"){const K=F.dataset.key;if(!y||!K)return;I.add(K),j.set(K,""),l=!1,f(y)}}),ft(n,(x,F)=>{x!=="ai-provider"||!y||(p=F,l=!1,f(y))}),G();async function G(){try{const x=await da();if(s)return;y=x,f(x)}catch(x){if(s)return;_.innerHTML=`<p class="error">Failed to load settings: ${a(String(x))}</p>`}}function A(x){const K=(Array.isArray(x.providerKeysSet)?x.providerKeysSet:[]).filter(te=>te!==pt).sort();return[pt,...K]}function E(x,F){const K=a(x);return`
      <div class="pk-row">
        <label>
          <code>${K}</code>
          <input class="provider-key" data-key="${K}" type="password" autocomplete="off"
                 placeholder="${F?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${as(x)}</p>
        ${F?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${K}">Clear saved key</button>`:""}
      </div>`}function f(x){var U;const F=p??x.aiProvider,K=Array.isArray(x.providerKeysSet)?x.providerKeysSet:[],te=A(x).map(ne=>E(ne,K.includes(ne))).join("");_.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${ut("ai-provider",ns.map(ne=>({value:ne.value,label:ne.label})),F)}
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
          ${te}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${a(V)}" />
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
            <input id="ref-rpc-base" type="text" value="${a(x.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${a(d)}</p>`:""}
        ${l?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const ce=n.querySelector("#ai-key");ce==null||ce.addEventListener("input",()=>{r=!0,l=!1}),(U=n.querySelector("#ref-rpc-base"))==null||U.addEventListener("input",()=>{l=!1}),n.querySelectorAll("input.provider-key").forEach(ne=>{const de=ne.dataset.key;if(!de)return;const pe=j.get(de);pe!==void 0&&(ne.value=pe),ne.addEventListener("input",()=>{I.add(de),j.set(de,ne.value),l=!1})});const z=n.querySelector("#pk-new-value");z&&(z.value=L),z==null||z.addEventListener("input",()=>{L=z.value,l=!1});const ie=n.querySelector("#pk-new-name");ie==null||ie.addEventListener("input",()=>{V=ie.value,l=!1})}async function g(){const x=n.querySelector("#ai-key"),F=n.querySelector("#ref-rpc-base");if(!x||!F||!y)return;const K={aiProvider:p??y.aiProvider,refRpcBase:F.value.trim()};r&&(K.aiKey=x.value);const te={};for(const z of I)te[z]=j.get(z)??"";const ce=V.trim();ce&&(te[ce]=L),Object.keys(te).length>0&&(K.providerKeys=te),e=!0,d=null,l=!1,f(y);try{const z=await ua(K);if(s)return;y=z,r=!1,I.clear(),j.clear(),V="",L="",e=!1,l=!0,f(z)}catch(z){if(s)return;e=!1,d=String(z instanceof Error?z.message:z),f(y)}}return()=>{s=!0}}const os=["http","ws","archive","trace"],rs={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},qe=1337,is="run",cs={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function ls(n){let s=!1,r=null,e=null;const d={},l={},y={},p={},I={},j={},V={},L={},_={},G={},A={},E={},f={},g={},x={};let F="",K=null;n.innerHTML=`
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
    ${ge()}
  `;const te=n.querySelector("#rpc-body");Ee(n,(t,o)=>{pn(t,o)}),ft(n,()=>{}),z(),ce();async function ce(){try{const t=await Ut();if(s)return;F=t.os,re()}catch{}}async function z(){try{const t=await ht();if(s)return;r=t,e=null}catch(t){if(s)return;r=null,e=ke(t)}re();for(const t of(r==null?void 0:r.gateways)??[])ie(t.id),U(t.id,!1)}async function ie(t){try{const o=await ra(t);if(s)return;d[t]=o}catch{if(s)return;d[t]=null}re()}async function U(t,o){y[t]=o,o&&re();try{const i=await jt(t,o);if(s)return;l[t]=i}catch{if(s)return;l[t]=null}y[t]=!1,re()}function ne(t){return((r==null?void 0:r.gateways)??[]).find(o=>o.id===t)}function de(t,o){return(t.networks??[]).find(i=>i.chainId===o)}function pe(t,o,i){var m;const c=(((m=d[t])==null?void 0:m.networks)??[]).find(C=>C.chainId===o);return((c==null?void 0:c.upstreams)??[]).find(C=>C.upstream===i)}function be(t,o,i){var c;return(((c=l[t])==null?void 0:c.endpoints)??[]).find(m=>m.chainId===o&&m.upstream===i)}function re(){if(s)return;if(e){te.innerHTML=`<p class="error">Could not read the gateways: ${a(e)}</p>`;return}if(!r){te.innerHTML='<p class="muted">Loading…</p>';return}const t=r.gateways??[],o=t.length>1,i=(r.targets??[]).some(C=>xt(C.id,t)),c=new Set(t.map(C=>C.placement.targetId)),m=(r.orphans??[]).filter(C=>!c.has(C.targetId));te.innerHTML=`
      ${t.map(C=>$(C,o)).join("")}
      ${t.length===0?Se():""}
      ${m.map(we).join("")}
      ${i?`<div class="card-actions rpc-add-gateway">
               <button class="btn${t.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${t.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function we(t){const o=`docker rm -f ${t.containerName}`,i=f[t.containerName];return`
      <div class="strip">
        ${O({tone:"warn",text:`${t.containerName} is still running on ${t.targetId}. Its chains were folded into ${t.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:o})}
        ${i?O({tone:"bad",text:i}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(t.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function Se(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function $(t,o){return`
      ${o?`<h2 class="rpc-machine">${a(t.placement.targetId)}</h2>`:""}
      ${h(t)}
      ${v(t)}
      ${S(t)}
      ${M(t)}
      ${b(t)}
    `}function h(t){const o=t.status.State==="running",i=t.tls,c=[`on <strong>${a(t.placement.targetId)}</strong>`];return t.status.Image&&c.push(`<code>${a(t.status.Image)}</code>`),c.push(i!=null&&i.enabled?`HTTPS front <code>${a(i.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${Y(t)}
        <strong>${a(t.label)}</strong>
        ${le(t)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${o?`base <code>${a(t.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function T(t){const o=t.tls;return o!=null&&o.enabled&&o.rootCaPath&&o.effectiveCertSource==="internal"?o.rootCaPath:null}function B(t){var o;return((o=((r==null?void 0:r.targets)??[]).find(i=>i.id===t.placement.targetId))==null?void 0:o.mode)??""}function W(t){switch(t){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return t||"this device"}}function Q(t,o,i){switch(t){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${o}"`;case"windows":return`certutil -addstore -f ROOT "${o}"`;case"linux":default:return`sudo cp "${o}" /usr/local/share/ca-certificates/valve-node-app-${i}.crt && sudo update-ca-certificates`}}function b(t){const o=_[t.id]??!1,i=((r==null?void 0:r.orphans)??[]).filter(c=>c.targetId===t.placement.targetId);return`
      <section class="card manage-section${o?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(t.id)}" aria-expanded="${o}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${w(t,i.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${o?D(t,i):""}
      </section>
    `}function w(t,o){const i=[];return t.status.State!=="running"&&i.push("gateway not running"),o>0&&i.push(`${o} leftover container${o===1?"":"s"}`),i.length===0?"container, settings, certificate":i.join(" · ")}function D(t,o){var i;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(t.actions??[]).map(c=>fe(t,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(t.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${a(t.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${y[t.id]?"disabled":""}>
            ${y[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${a(t.id)}">
            ${V[t.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${a(t.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${t.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${a(t.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(t.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${a(((i=(t.networks??[])[0])==null?void 0:i.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${a(t.baseUrl)}</code> once it is running.</p>`}
        ${Z(t)}
        ${o.map(we).join("")}
        ${V[t.id]?sn(t):""}
      </div>
    `}function Z(t){const o=T(t);if(!o)return"";const i=B(t)==="local",c=Q(F,o,t.id),m=x[t.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(t.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(o)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(o)}">Copy path</button>
        </div>
        ${i?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(t.id)}" ${g[t.id]?"disabled":""}>
                   ${g[t.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${m?u(m):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${i?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${F?` (${a(W(F))})`:""}:</span>
          <code class="strip-cmd">${a(c)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(c)}">Copy command</button>
        </div>
      </div>
    `}function u(t){return t.ok?`<div class="strip-line strip-note"><span class="strip-text">${a(t.message)}</span></div>`:`
      <div class="strip-line strip-warn">
        <span class="strip-text">${a(t.message)}</span>
        ${t.ranCommand?`<code class="strip-cmd">${a(t.ranCommand)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(t.ranCommand)}">Copy</button>`:""}
      </div>
    `}function v(t){const o=[];t.error&&o.push({tone:"bad",text:`This gateway could not be read: ${t.error}${t.hint?` — ${t.hint}`:""}`}),t.blocked&&o.push({tone:"warn",text:t.blocked});for(const c of t.warnings??[])o.push({tone:"warn",text:c});o.push(...X(t));const i=I[t.id];return i&&o.push({tone:"bad",text:i}),o.length===0?"":`<div class="strip">${o.map(O).join("")}</div>`}function O(t){return`
      <div class="strip-line strip-${t.tone}">
        <span class="strip-text">${a(t.text)}</span>
        ${t.cmd?`<code class="strip-cmd">${a(t.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(t.cmd)}">Copy</button>`:""}
      </div>
    `}function X(t){var m,C;const o=t.tls;if(!(o!=null&&o.enabled))return[];const i=[];o.fallback&&i.push({tone:"warn",text:o.fallback}),o.error?i.push({tone:"warn",text:`HTTPS front: ${o.error}`}):((m=o.status)==null?void 0:m.State)!=="running"&&i.push({tone:"warn",text:`The HTTPS front is ${((C=o.status)==null?void 0:C.State)??"unknown"}, so nothing answers on ${o.url??"its https URL"} even if the gateway itself is up.`,cmd:o.containerName?`docker start ${o.containerName}`:void 0});const c=G[t.id]??o.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&i.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&i.push({tone:"warn",text:c.expiryWarning}),i}function le(t){switch(t.status.State){case"running":return J("running","ok");case"created-but-stopped":return J("stopped","warn");case"not-created":return J("not created","neutral");default:return J("unknown","bad")}}function Y(t){return t.status.State==="running"?Ne("ok"):t.status.State==="unknown"?Ne("bad"):Ne("neutral")}function fe(t,o){const i=cs[o];if(!i)return"";const c=p[t.id];return`
      <button class="${i.className}" data-action="gw-${o}" data-gid="${a(t.id)}"
              title="${a(i.title)}" ${c?"disabled":""}>
        ${c===o?'<span class="spinner" aria-label="working"></span>':a(i.label)}
      </button>
    `}function S(t){const o=j[t.id]??[];return o.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(t.placement.targetId)}</p>
        <pre class="step-log">${a(o.join(`
`))}</pre>
      </div>
    `}function M(t){const o=P(t.networks??[]),i=o.some(c=>c.chainId===qe);return o.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(t.id)}">Add a network</button>
            ${$e(t,i)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${o.map(c=>q(t,c)).join("")}
        </div>
        ${se(t,i)}
        ${an(t)}
      </div>
    `}function P(t){const o=t.filter(c=>c.chainId!==qe),i=t.filter(c=>c.chainId===qe);return[...o,...i]}function q(t,o){const i=tt(o),c=o.chainId===qe,m=`${t.id}:${o.chainId}`,C=L[m]??!1,R=i.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${i.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(o.name)}</span>
          <code class="chain-key">evm:${o.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${J(R,i.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(m)}" aria-expanded="${C}">
              ${C?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${H(t,o)}
        ${C?N(t,o,i):""}
      </section>
    `}function H(t,o){if(!o.url)return`<p class="chain-connect-none muted small">${t.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const i=T(t);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(o.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(o.url)}"
                title="Copy ${a(o.url)}">Copy URL</button>
        ${i?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${B(t)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(t.id)}" ${g[t.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${g[t.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(i)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(t.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${x[t.id]?`<span class="chain-cert muted small">${a(x[t.id].ok?"Trusted — reload your wallet or browser.":x[t.id].message)}</span>`:""}`:""}
      </div>
    `}function N(t,o,i){const c=o.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${i.why?" chain-verdict-why":""}"${i.why?` title="${a(i.why)}"`:""}>${i.html}</p>
        <div class="chain-detail-bar">
          ${Te(c.length,i.tone,o.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(t.id)}" data-chain="${o.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(t.id)}" data-chain="${o.chainId}">Remove</button>
        </div>
        ${Yt(t,o)}
        ${(o.warnings??[]).map(m=>`<p class="chain-note">${a(m)}</p>`).join("")}
      </div>
    `}function se(t,o){const i=l[t.id],c=i!=null&&i.at?`probed ${a($t(i.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(t.id)}">+ Network</button>
        ${$e(t,o)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${y[t.id]?"disabled":""}>
          ${y[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function $e(t,o){return o?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(t.id)}"
                    title="Add a throwaway local test chain (evm:${qe}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Te(t,o,i){const c=i>0,m=c?i:t,C=Math.min(t,m);let R="";for(let Fe=0;Fe<m;Fe++)R+=`<span class="seg${Fe<C?` seg-on seg-${o}`:""}"></span>`;const k=c&&t>i,ae=c?k?`${t} (set is ${i})`:`${t} of ${i}`:`${t}`,ue=`${t} upstream${t===1?"":"s"} configured`,Ce=c?`${ue}${k?`, ${t-i} beyond the set`:""}. valve's set for this chain is ${i}.`:`${ue}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(Ce)}">${R}</span>
      <span class="segs-n">${ae}</span>
    `}function tt(t){const o=t.upstreams??[];if(o.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!t.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!o.some(zt)){const c=Jt(o);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(C=>`<code>${a(C)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(o.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!o.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const i=o.filter(c=>!!c.problem);if(i.length>0){const c=o.length-i.length;return{tone:"warn",html:`${i.length} of these ${o.length} endpoints ${i.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${o.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function zt(t){return/^wss?:\/\//i.test((t.endpoint??"").trim())}function Jt(t){const o=new Set;for(const i of t){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((i.endpoint??"").trim());c&&o.add(c[1].toLowerCase())}return[...o].sort()}function Yt(t,o){const i=o.upstreams??[];return i.length===0?"":`<ul class="ups">${i.map(c=>Zt(t,o,c)).join("")}</ul>`}function Zt(t,o,i){const c=`${t.id}|${o.chainId}|${i.id}`,m=i.actions??[];return`
      <li class="up${i.problem?" up-bad":""}">
        <div class="up-what">
          ${i.problem?Ne("bad"):Ne("ok")}
          <span class="up-label">${a(i.label)}</span>
          ${Xt(i)}
        </div>
        <code class="up-url">${a(i.endpoint||"—")}</code>
        <div class="up-caps">${Qt(t,o,i)}</div>
        <div class="up-share">${nn(t,o,i)}</div>
        <div class="up-acts">
          ${m.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${a(c)}"
                         data-target="${a(i.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${p[t.id]?"disabled":""}>
                   ${p[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${a(c)}">Remove</button>
        </div>
        ${i.problem?`<div class="up-problem error small">${a(i.problem)}</div>`:""}
      </li>
    `}function Xt(t){return t.problem?J("unusable","bad"):t.recentOnly?J("recent blocks","warn"):t.local?J("yours","ok"):J("public","neutral")}function gt(t,o){var i;if(t)return o==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(i=(t.capabilities??[]).find(c=>c.key===o))==null?void 0:i.status}function Qt(t,o,i){const c=be(t.id,o.chainId,i.id);return c?c.unprobeable?`<span class="caps-none" title="${a(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${os.map(m=>en(t,o,c,m)).join("")}</span>`:`<span class="muted small">${l[t.id]===void 0?"probing…":"—"}</span>`}function en(t,o,i,c){const m=(i.capabilities??[]).find(ue=>ue.key===c),C=gt(i,c)??"inconclusive",R=rs[c]??c.toUpperCase();let k="cap";C==="unsupported"?k=tn(t,o,c)?"cap missing":"cap off":C==="inconclusive"?k="cap unknown":C==="inconsistent"&&(k="cap mixed");const ae=m!=null&&m.detail?`${m.label}: ${m.detail}`:c==="http"&&i.reachDetail?`Answers JSON-RPC over HTTP: ${i.reachDetail}`:`${R}: no verdict`;return`<span class="${k}" title="${a(ae)}">${a(R)}</span>`}function tn(t,o,i){const c=(o.upstreams??[]).map(m=>be(t.id,o.chainId,m.id)).filter(m=>!!m&&!m.unprobeable);return c.length>0&&c.every(m=>gt(m,i)==="unsupported")}function nn(t,o,i){const c=d[t.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const m=pe(t.id,o.chainId,i.id),C=(c.networks??[]).find(Ce=>Ce.chainId===o.chainId);if(!m||!C||C.attributed===0)return'<span class="muted small">no traffic yet</span>';const R=Math.round(m.actual*100),k=Math.round(m.intended*100),ae=m.diverged?i.local?"warn":"":"ok",ue=`${m.succeeded.toLocaleString()} of ${C.attributed.toLocaleString()} answered requests · routing intends ${k}%`+(m.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(ue)}">
        <span class="bar">
          <span class="fill${ae?" "+ae:""}" style="width:${R}%"></span>
          <span class="tick" style="left:${k}%"></span>
        </span>
        <span class="share-n${m.diverged?" warn":""}">${R}%</span>
        ${m.unconfigured?J("not in config","warn"):""}
      </span>
    `}function an(t){const o=d[t.id];return o?o.enabled?o.error?`<p class="muted small">The request counters could not be read: ${a(o.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${o.since?` (${a($t(o.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function $t(t){const o=new Date(t);return Number.isNaN(o.getTime())?t:o.toLocaleString()}function sn(t){const o=t.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${a(t.id)}-port" value="${o.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${a(t.id)}-bind" value="${a(o.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${a(o.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${on(t)}
        ${rn(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(t.id)}">Save settings</button>
        </div>
      </div>
    `}function on(t){const o=!t.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${a(t.id)}-metrics" ${o?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function rn(t){var R;const o=a(t.id),i=t.config.TLS??null,c=(i==null?void 0:i.Enabled)??!1,m=(i==null?void 0:i.CertSource)||"internal",C=((R=t.tls)==null?void 0:R.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${o}-tls" ${c?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${o}-tls-host" value="${a((i==null?void 0:i.Hostname)??C)}"
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
        <input type="text" inputmode="numeric" id="gw-${o}-tls-port" value="${(i==null?void 0:i.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${o}-tls-source">
          <option value="internal" ${m==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${m==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${o}-tls-cert" value="${a((i==null?void 0:i.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${o}-tls-key" value="${a((i==null?void 0:i.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${cn(t)}
    `}function cn(t){var R,k;const o=a(t.id),i=((R=t.config.TLS)==null?void 0:R.Enabled)??!1,c=G[t.id]??((k=t.tls)==null?void 0:k.verification)??null,m=A[t.id]??!1,C=E[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${o}" ${i&&!m?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${m?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${i?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${C?`<p class="error small">${a(C)}</p>`:""}
      ${c?ln(c):""}
    `}function ln(t){const o=(t.assertions??[]).map(i=>`
          <li class="small">
            ${dn(i.status)}
            <strong>${a(i.title)}</strong>
            <div class="muted">${a(i.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${t.ok?t.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${a(t.summary)}
      </div>
      <ul class="verify-list">${o}</ul>
      <p class="muted small">
        Checked ${a(new Date(t.at).toLocaleString())} against <code>${a(t.address)}</code>
        ${t.notAfter?`· certificate valid until <code>${a(new Date(t.notAfter).toLocaleString())}</code> (${a(t.expiresIn??"")})`:""}
      </p>
      ${t.expiryWarning?`<div class="banner banner-warn">${a(t.expiryWarning)}</div>`:""}
    `}function dn(t){switch(t){case"pass":return J("pass","ok");case"fail":return J("fail","bad");case"unavailable":return J("unavailable","warn");default:return J("skipped","neutral")}}async function un(t){A[t]=!0,E[t]=null,re();try{G[t]=await Ft(t)}catch(o){E[t]=`${ke(o)}${Oe(o)}`}finally{A[t]=!1,re()}}function He(t){return{...t.config,Networks:(t.config.Networks??[]).map(o=>({ChainID:o.ChainID,Upstreams:o.Upstreams.map(i=>({...i}))}))}}async function Ue(t,o,i){I[t]=null;try{await Le(t,o)}catch(c){return I[t]=`${i?i+": ":""}${ke(c)}`,re(),!1}return await z(),!0}async function pn(t,o){const i=o.dataset.gid??"";switch(t){case"refresh":await z();return;case"copy":o.dataset.copy&&await Un(o,o.dataset.copy);return;case"reprobe":await U(i,!0);return;case"toggle-settings":V[i]=!V[i],re();return;case"toggle-manage":_[i]=!_[i],re();return;case"toggle-chain-detail":{const c=o.dataset.key??"";c&&(L[c]=!L[c]),re();return}case"save-settings":await hn(i);return;case"verify-tls":await un(i);return;case"trust-cert":await bn(i);return;case"gw-start":case"gw-stop":case"gw-restart":await yn(i,t.slice(3));return;case"gw-create":case"gw-recreate":await vn(i);return;case"gw-wipe":An(i);return;case"add-gateway":Dn();return;case"forget-gateway":await gn(i);return;case"dismiss-orphan":await $n(o.dataset.name??"");return;case"add-chain":wn(i);return;case"add-devnet":{const c=ne(i);if(c){const m=((r==null?void 0:r.targets)??[]).some(C=>C.id===c.placement.targetId&&C.hasDevnet);kt(i,qe,m)}return}case"remove-chain":await Sn(i,Number.parseInt(o.dataset.chain??"",10));return;case"add-endpoint":St(i,Number.parseInt(o.dataset.chain??"",10));return;case"remove-endpoint":await xn(o.dataset.key??"");return;case"reset-devnet":await Ln(o.dataset.key??"",o.dataset.target??"");return;default:return}}async function hn(t){const o=ne(t);if(!o)return;const i=He(o),c=n.querySelector(`#gw-${CSS.escape(t)}-port`),m=n.querySelector(`#gw-${CSS.escape(t)}-bind`);if(c){const k=Number.parseInt(c.value.trim(),10);Number.isFinite(k)&&(i.Port=k)}m&&(i.BindAddr=m.value.trim());const C=n.querySelector(`#gw-${CSS.escape(t)}-metrics`);C&&(i.MetricsOff=!C.checked),i.TLS=fn(t,o);const R=o.status.State==="running";await Ue(t,i,"Saving settings")&&(V[t]=!1,R&&(I[t]=null,mn(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),re())}function fn(t,o){var C,R,k,ae,ue,Ce,Fe;const i=Mn=>n.querySelector(`#gw-${CSS.escape(t)}-${Mn}`),c=i("tls");if(!c)return o.config.TLS??null;const m=Number.parseInt(((C=i("tls-port"))==null?void 0:C.value.trim())??"",10);return{Enabled:c.checked,Hostname:((R=i("tls-host"))==null?void 0:R.value.trim())??"",CertSource:((k=i("tls-source"))==null?void 0:k.value)??"internal",CertFile:((ae=i("tls-cert"))==null?void 0:ae.value.trim())??"",KeyFile:((ue=i("tls-key"))==null?void 0:ue.value.trim())??"",HTTPSPort:Number.isFinite(m)?m:443,BindAddr:((Ce=o.config.TLS)==null?void 0:Ce.BindAddr)??"",ImageRef:((Fe=o.config.TLS)==null?void 0:Fe.ImageRef)??""}}function mn(t,o){j[t]=[o]}async function bn(t){if(!g[t]){g[t]=!0,x[t]=null,re();try{x[t]=await ca(t)}catch(o){x[t]={ok:!1,message:`${ke(o)}${Oe(o)}`}}g[t]=!1,re()}}async function yn(t,o){if(!p[t]){p[t]=o,I[t]=null,re();try{await qt(t,o)}catch(i){I[t]=`${o} failed: ${ke(i)}${Oe(i)}`}p[t]=null,await z()}}async function vn(t){if(p[t])return;p[t]="create",I[t]=null,j[t]=["starting…"],re();let o;try{o=await lt(t)}catch(i){I[t]=`${ke(i)}${Oe(i)}`,j[t]=[],p[t]=null,re();return}K==null||K(),K=ze(o.targetId,i=>{if(s)return;const c=i.err?`${i.stepId}: ${i.err}`:i.line?`${i.stepId}: ${i.line}`:`${i.stepId}: done`;if(j[t]=[...(j[t]??[]).filter(C=>C!=="starting…"),c],!!i.err||i.stepId===is&&!!i.done){K==null||K(),K=null,p[t]=null,i.err&&(I[t]="Provisioning failed — see the log below."),z();return}re()})}async function gn(t){const o=ne(t);if(!(!o||!await Ae({title:`Forget ${o.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${o.containerName}" on ${o.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await ia(t)}catch(c){I[t]=ke(c),re();return}await z()}}async function $n(t){if(t){f[t]=null;try{await oa(t)}catch(o){f[t]=ke(o),re();return}await z()}}function wn(t){const o=ne(t);if(!o)return;const i=new Set((o.networks??[]).map(k=>k.chainId)),c=(r==null?void 0:r.presets)??[],m=c.filter(k=>!i.has(k.chainId)),C=c.filter(k=>i.has(k.chainId)),R=((r==null?void 0:r.targets)??[]).some(k=>k.id===o.placement.targetId&&k.hasDevnet);he(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${a(o.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${m.map(k=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${k.chainId}">
                <span>${a(k.name)}</span>
                <span class="muted small">chain ${k.chainId}${k.devnet?R?" · uses the devnet on "+a(o.placement.targetId):" · will create a devnet on "+a(o.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${C.length?`<p class="muted small">Already fronted: ${a(C.map(k=>k.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,k=>{if(k==="cancel"){ee();return}if(k==="custom"){kn(t);return}if(k.startsWith("preset:")){const ae=Number.parseInt(k.slice(7),10),ue=c.find(Ce=>Ce.chainId===ae);ee(),ue!=null&&ue.devnet?kt(t,ae,R):wt(t,ae)}})}function kn(t){var o;he(`
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
      `,i=>{if(i==="cancel"){ee();return}if(i!=="add")return;const c=document.getElementById("custom-chain-id"),m=document.getElementById("custom-chain-err"),C=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(C)||C<=0){m&&(m.className="error small"),m&&(m.textContent="A chain id is a positive whole number.");return}ee(),wt(t,C)}),(o=document.getElementById("custom-chain-id"))==null||o.focus()}async function wt(t,o){const i=ne(t);if(!i)return;const c=He(i),m=c.Networks??[];m.some(C=>C.ChainID===o)||(m.push({ChainID:o,Upstreams:[]}),c.Networks=m,await Cn(t,c)&&(re(),St(t,o)))}async function Cn(t,o){var C;const i={...o,Networks:(o.Networks??[]).filter(R=>R.Upstreams.length>0)};if(!await Ue(t,i))return!1;const m=ne(t);if(m)for(const R of o.Networks??[])R.Upstreams.length===0&&!(m.networks??[]).some(k=>k.chainId===R.ChainID)&&(m.config.Networks=[...m.config.Networks??[],{ChainID:R.ChainID,Upstreams:[]}],m.networks=[...m.networks??[],{chainId:R.ChainID,name:((C=((r==null?void 0:r.presets)??[]).find(k=>k.chainId===R.ChainID))==null?void 0:C.name)??`Chain ${R.ChainID}`,path:`/${m.config.ProjectID}/evm/${R.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function kt(t,o,i){const c=ne(t);if(!c)return;if(!i){he(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${a(c.placement.targetId)}</code>, so adding chain ${o} here
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
        `,()=>ee());return}const m=He(c),C=m.Networks??[],R={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},k=C.find(ae=>ae.ChainID===o);k?k.Upstreams.push(R):C.push({ChainID:o,Upstreams:[R]}),m.Networks=C,await Ue(t,m,"Adding the devnet")}async function Sn(t,o){const i=ne(t);if(!i||!Number.isFinite(o))return;const c=de(i,o);if(!await Ae({title:`Remove ${(c==null?void 0:c.name)??`chain ${o}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${o}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const C=He(i);C.Networks=(C.Networks??[]).filter(R=>R.ChainID!==o),await Ue(t,C,"Removing the network")}function Ct(t){const o=t.split("|");return o.length!==3?null:{gid:o[0],chainId:Number.parseInt(o[1],10),upstreamId:o[2]}}async function xn(t){const o=Ct(t);if(!o)return;const i=ne(o.gid);if(!i)return;const c=He(i),m=(c.Networks??[]).find(k=>k.ChainID===o.chainId);if(!m)return;const C=m.Upstreams.findIndex((k,ae)=>(k.ID||`${o.chainId}-${ae}`)===o.upstreamId);C<0||!await Ae({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(m.Upstreams.splice(C,1),await Ue(o.gid,c,"Removing the endpoint"))}function St(t,o){const i=ne(t);if(!i||!Number.isFinite(o))return;const c=((r==null?void 0:r.sources)??[]).filter(k=>k.chainId===o),m=de(i,o),C=new Set(((m==null?void 0:m.upstreams)??[]).filter(k=>k.kind!=="external").map(k=>`${k.kind}|${k.targetId??""}`)),R=c.filter(k=>!C.has(`${k.kind}|${k.targetId}`));he(`
        <h2>Add an endpoint for ${a((m==null?void 0:m.name)??`chain ${o}`)}</h2>
        ${R.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${R.map(k=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${a(k.kind)}:${a(k.targetId)}">
                       <span>${a(k.label)}</span>
                       <span class="muted small">${a(k.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${o}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn" data-modal-action="known-set">Add valve's set…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,k=>{if(k==="cancel"){ee();return}if(k==="known-set"){En(t,o);return}if(k==="manual"){Rn(t,o);return}if(k.startsWith("source:")){const[,ae,ue]=k.split(":");ee(),Tn(t,o,ae,ue)}})}async function Tn(t,o,i,c){const m=ne(t);if(!m)return;const C=He(m),R=C.Networks??[],k={ID:`${i==="managed-devnet"?"devnet":"node"}-${c}`,Kind:i,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},ae=R.find(ue=>ue.ChainID===o);ae?ae.Upstreams.push(k):R.push({ChainID:o,Upstreams:[k]}),C.Networks=R,await Ue(t,C,"Adding the endpoint")}function In(t){const o=[...t].sort((m,C)=>(m.latencyMs??1e9)-(C.latencyMs??1e9)),i=o.slice(0,3),c=o.find(m=>m.url.startsWith("wss://")||m.url.startsWith("ws://"));return c&&!i.some(m=>m.url===c.url)&&(i.length===3&&i.pop(),i.push(c)),new Set(i.map(m=>m.url))}async function En(t,o){let i;try{i=await dt(t,o)}catch(k){he(`<h2>Endpoints for chain ${o}</h2>
         <p class="error small">Could not read the set: ${a(ke(k))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>ee());return}if(s)return;const c=i.endpoints??[],m=c.filter(k=>!k.alreadyAdded).map(k=>k.url),C=new Set(c.map(k=>k.provider)).size,R=c.map(k=>{const ae=[k.websocket?'<span class="t ws">websocket</span>':"",k.archive?'<span class="t ar">archive</span>':"",k.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(k.url)}</code>
                  <span class="muted small">${a(k.provider)}</span> ${ae}</li>`}).join("");he(`<h2>Endpoints for chain ${o}</h2>
       ${c.length?`<p class="muted small">${C} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${R}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${i.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${m.length?"":" disabled"}>
           ${m.length?`Add ${m.length}`:"Nothing to add"}</button>
       </div>`,k=>{ee(),k==="add"&&nt(t,o,m),k==="discover"&&Pn(t,o)})}async function Pn(t,o){he(`
        <h2>Public endpoints for chain ${o}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,R=>{R==="cancel"&&ee()});let i;try{i=await la(o)}catch(R){const k=We();if(k){const ae=document.createElement("p");ae.className="error small",ae.textContent=`Could not discover endpoints: ${ke(R)}`,k.appendChild(ae)}return}if(s)return;const c=(i.endpoints??[]).filter(R=>R.status==="live"||R.status==="unprobed"),m=(i.endpoints??[]).filter(R=>R.status==="rejected"),C=In(c);he(`
        <h2>Public endpoints for chain ${o}</h2>
        ${i.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${i.fetchError?`<div class="small">${a(i.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(R=>{const k=C.has(R.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${a(R.url)}"${k}>
                       <span><code>${a(R.url)}</code></span>
                       <span class="muted small">${R.status==="live"?`answered in ${R.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${o} right now.</p>`}
        ${m.length?`<details class="rpc-rejected">
                 <summary class="muted small">${m.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${m.map(R=>`<li class="muted small"><code>${a(R.url)}</code> — ${a(R.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,R=>{if(R==="cancel"){ee();return}if(R==="add"){const k=We(),ae=k?Array.from(k.querySelectorAll('input[type="checkbox"]:checked')).map(ue=>ue.value):[];ee(),nt(t,o,ae);return}})}function Rn(t,o){var i;he(`
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
      `,c=>{if(c==="cancel"){ee();return}if(c!=="add")return;const m=document.getElementById("manual-endpoint"),C=document.getElementById("manual-recent"),R=document.getElementById("manual-err"),k=(m==null?void 0:m.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(k)){R&&(R.className="error small",R.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}ee(),nt(t,o,[k],(C==null?void 0:C.checked)??!1)}),(i=document.getElementById("manual-endpoint"))==null||i.focus()}async function nt(t,o,i,c=!1){if(!i.length)return;const m=ne(t);if(!m)return;const C=He(m),R=C.Networks??[];let k=R.find(ue=>ue.ChainID===o);k||(k={ChainID:o,Upstreams:[]},R.push(k));let ae=1;for(const ue of k.Upstreams){const Ce=/^public-\d+-(\d+)$/.exec(ue.ID??"");Ce&&(ae=Math.max(ae,Number(Ce[1])+1))}for(const ue of i)k.Upstreams.some(Ce=>Ce.Endpoint===ue)||k.Upstreams.push({ID:`public-${o}-${ae++}`,Kind:"external",Endpoint:ue,Local:!1,RecentOnly:c});C.Networks=R,await Ue(t,C,i.length===1?"Adding the endpoint":`Adding ${i.length} endpoints`)}async function Ln(t,o){const i=Ct(t);if(!i||!o||!await Ae({title:"Reset this devnet",body:`The chain on ${o} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;p[i.gid]="reset",I[i.gid]=null,re();let m;try{m=await aa(o)}catch(C){I[i.gid]=`Reset failed: ${ke(C)}${Oe(C)}`,p[i.gid]=null,re();return}p[i.gid]=null,Nn(o,m),await z()}function Nn(t,o){const i=[];i.push(o.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),o.report.Recreated&&i.push("A fresh chain was started from genesis.");const c=o.report.Cascaded??[],m=o.report.CascadeSkipped??[];he(`
        <h2>Devnet on ${a(t)} reset</h2>
        <ul class="plain-list">${i.map(C=>`<li>${a(C)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${a(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${m.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(m.join(", "))}.</p>`:""}
        ${o.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>ee())}function An(t){const o=ne(t);if(!o)return;he(`
        <h2>Wipe ${a(o.label)}</h2>
        <p class="error">This destroys ${a(o.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(t)}</button>
        </div>
      `,m=>{if(m==="cancel"||m==="close"){ee(),z();return}m==="confirm"&&Bn(t)});const i=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");i==null||i.addEventListener("input",()=>{c&&(c.disabled=i.value.trim()!==t)}),i==null||i.focus()}async function Bn(t){const o=document.getElementById("wipe-confirm-btn");o&&(o.disabled=!0,o.textContent="Wiping…");let i;try{i=await Wt(t)}catch(c){const m=We();if(m){const C=document.createElement("p");C.className="error small",C.textContent=`Wipe failed: ${ke(c)}${Oe(c)}`,m.appendChild(C)}o&&(o.disabled=!1,o.textContent=`Wipe ${t}`);return}he(`
        <h2>${a(t)} wiped</h2>
        <ul class="plain-list">
          <li>${i.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${i.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${i.error?`<p class="error small">${a(i.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{ee(),z()})}function xt(t,o){return!o.some(i=>{var c;return((c=i.placement)==null?void 0:c.targetId)===t})}function Dn(){var C;const t=(r==null?void 0:r.targets)??[],o=(r==null?void 0:r.gateways)??[],i=t.filter(R=>xt(R.id,o)),c=new Set(o.map(R=>R.id));if(t.length===0){he(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>ee());return}if(i.length===0){he(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>ee());return}const m=c.has("default")?"":"default";he(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${a(m)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${i.map(R=>`<option value="${a(R.id)}">${a(R.id)} (${a(R.mode)})</option>`).join("")}
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
      `,R=>{if(R==="cancel"){ee();return}R==="create"&&Hn()}),(C=document.getElementById("new-gw-id"))==null||C.focus()}async function Hn(){const t=document.getElementById("new-gw-id"),o=document.getElementById("new-gw-target"),i=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),m=(t==null?void 0:t.value.trim())??"",C=(o==null?void 0:o.value)??"",R=Number.parseInt((i==null?void 0:i.value.trim())??"",10),k=ae=>{c&&(c.className="error small",c.textContent=ae)};if(!m){k("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!C){k("Pick the machine it runs on.");return}try{await Ot({id:m,placement:{targetId:C,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(R)?R:4e3,Networks:[]}})}catch(ae){k(ke(ae));return}ee(),await z()}async function Un(t,o){const i=await Ke(o),c=t.textContent;t.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{s||(t.textContent=c)},1500)}function ke(t){return t instanceof Error?t.message:String(t)}function Oe(t){return t instanceof Re&&t.hint?` — ${t.hint}`:""}return()=>{s=!0,K==null||K(),ee()}}const ds="local";function us(n){let s=!1,r=!1,e="",d=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ge()}
  `;const l=n.querySelector("#targets-body");Ee(n,(f,g)=>{V(f,g)}),y();async function y(){try{const[f,g,x]=await Promise.all([De(),Be(),Ut()]);if(s)return;e=x.os,I(f,g)}catch(f){if(s)return;l.innerHTML=`<p class="error">Failed to load machines: ${a(String(f))}</p>`}}function p(){d&&I(d.targets,d.catalog)}function I(f,g){d={targets:f,catalog:g};const x=e==="linux",F=[...f].sort((z,ie)=>(z.mode==="local"?-1:0)-(ie.mode==="local"?-1:0)),K=F.length?`<div class="card-grid">${F.map(z=>ps(z,g,z.mode!=="local"||x,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',te=f.some(z=>z.mode==="local");l.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${K}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(x,te)}
        ${r?hs():""}
      </section>
    `;const ce=l.querySelector("#fleet-verdict");ce&&Ha(ce,Da(f,g))}function j(f,g){const x=`
      <div class="card">
        <h3>A server over SSH ${J("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${f?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${f?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,F=f?`
        <div class="card">
          <h3>This machine ${J("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${a(e)})`:""} ${J("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return g?`<div class="card-grid card-grid-wide">${x}</div>`:`<div class="card-grid card-grid-wide">${f?F+x:x+F}</div>`}async function V(f,g){var x;if(f==="add-local"){await L();return}if(f==="delete-target"){const F=g.dataset.id;if(!F||!await Ae({title:"Remove machine",body:`Remove "${F}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await _(F);return}if(f==="toggle-ssh"){r=!r,E(),p(),r&&((x=n.querySelector("#ssh-host"))==null||x.focus());return}f==="add-ssh"&&await G()}async function L(){E();try{await it({id:ds,mode:"local"}),await y()}catch(f){A(f)}}async function _(f){try{await jn(f),await y()}catch(g){A(g)}}async function G(){const f=n.querySelector("#ssh-host"),g=n.querySelector("#ssh-user"),x=n.querySelector("#ssh-key"),F=n.querySelector("#ssh-port"),K=n.querySelector("#ssh-id");if(!f||!g||!x||!F||!K)return;const te=f.value.trim(),ce=g.value.trim(),z=x.value.trim(),ie=F.value.trim(),U=K.value.trim();if(E(),!te||!ce||!z){A(new Error("host, user, and key path are required"));return}const ne=U||fs(te),de={Host:te,User:ce,KeyPath:z};if(ie){const be=Number.parseInt(ie,10);if(!Number.isFinite(be)||be<=0){A(new Error("port must be a positive number"));return}de.Port=be}const pe=n.querySelector("#ssh-submit");pe&&(pe.disabled=!0,pe.textContent="Connecting…");try{await it({id:ne,mode:"ssh",ssh:de}),r=!1,await y()}catch(be){A(be),pe&&(pe.disabled=!1,pe.textContent="Add server")}}function A(f){let g=n.querySelector("#targets-error");g||(l.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),g=n.querySelector("#targets-error")),g.textContent=String(f instanceof Error?f.message:f)}function E(){var f;(f=n.querySelector("#targets-error"))==null||f.remove()}return()=>{s=!0}}function ps(n,s,r,e){const d=n.wire,l=n.mode==="local"?"this machine":"SSH",y=n.mode==="ssh"&&n.ssh?`${a(n.ssh.User)}@${a(n.ssh.Host)}`:l;let p;if(!d&&!r)p=`${J("can't run a node","warn")} ${J(e||"not Linux","neutral")}`;else if(!d)p=J("not set up","neutral");else{const I=s.networks.find(V=>V.ChainID===d.ChainID),j=I?I.Name:`chain ${d.ChainID}`;p=`${J(j,"ok")} ${J(d.ExecID,"neutral")} ${J(d.BeaconID,"neutral")}${d.Archive?" "+J("archive","warn"):""}`}return`
    <div class="card">
      <h2>${a(n.id)}</h2>
      <p class="muted">${y}</p>
      <p>${p}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(n.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(n.id)}">Remove</button>
      </div>
    </div>
  `}function hs(){return`
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
  `}function fs(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ms=document.querySelector("#app"),{contentEl:bs,setActiveNav:ys}=pa(ms);let xe=null;function vs(){const s=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(s.length===0)return{screen:"home"};const[r,e]=s;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:e?decodeURIComponent(e):void 0}:{screen:r??"targets"}}function Pe(n){const s=document.createElement("div");return bs.replaceChildren(s),n(s)}function Gt(){if(xe){try{xe()}catch{}xe=null}const{screen:n,id:s}=vs();switch(ys(n),n){case"machine":if(!s){location.hash="#/targets";return}xe=Pe(r=>Ia(r,s));break;case"setup":case"dash":case"logs":case"services":if(!s){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(s)}`;return;case"security":if(!s){location.hash="#/targets";return}xe=Pe(r=>ts(r,s));break;case"diag":if(!s){location.hash="#/targets";return}xe=Pe(r=>ya(r,s));break;case"analytics":if(!s){location.hash="#/rpc";return}xe=Pe(r=>ba(r,s));break;case"rpc":xe=Pe(r=>ls(r));break;case"settings":xe=Pe(r=>ss(r));break;case"targets":xe=Pe(r=>us(r));break;case"panel":xe=Pe(r=>Dt(r));break;case"home":default:xe=Pe(r=>Dt(r));break}}window.addEventListener("hashchange",Gt);Gt();
