var On=Object.defineProperty;var Fn=(n,s,o)=>s in n?On(n,s,{enumerable:!0,configurable:!0,writable:!0,value:o}):n[s]=o;var Ze=(n,s,o)=>Fn(n,typeof s!="symbol"?s+"":s,o);(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))e(d);new MutationObserver(d=>{for(const l of d)if(l.type==="childList")for(const y of l.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&e(y)}).observe(document,{childList:!0,subtree:!0});function o(d){const l={};return d.integrity&&(l.integrity=d.integrity),d.referrerPolicy&&(l.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?l.credentials="include":d.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function e(d){if(d.ep)return;d.ep=!0;const l=o(d);fetch(d.href,l)}})();function Ot(){return se("/api/host")}function He(){return se("/api/catalog")}function Ue(){return se("/api/targets")}function lt(n){return se("/api/targets",{method:"POST",headers:Ee,body:JSON.stringify(n)})}function jn(n){return se(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function qn(n,s){return se(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(s)}`)}function Wn(n,s){return se(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:Ee,body:JSON.stringify(s)})}function Ye(n,s){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function _n(n,s){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function Kn(n,s=200){return se(`/api/targets/${encodeURIComponent(n)}/logs?n=${s}`)}function Vn(n,s){const o=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function Et(n,s){const o=s===void 0?{}:{lines:s};return se(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:Ee,body:JSON.stringify(o)})}function Gn(n,s,o){return se(`/api/targets/${encodeURIComponent(n)}/services/${s}/${o}`,{method:"POST"})}function zn(n,s){return se(`/api/targets/${encodeURIComponent(n)}/services/${s}/clear`,{method:"POST",headers:Ee,body:JSON.stringify({Confirm:s})})}function Jn(n){return se(`/api/targets/${encodeURIComponent(n)}/du`)}function Yn(n){return se(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function Zn(n){return se(`/api/targets/${encodeURIComponent(n)}/firewall`)}function Xn(n){return se(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function Qn(n){return se(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function Ft(n){return se(`/api/targets/${encodeURIComponent(n)}/containers`)}function ea(n,s,o){return se(`/api/targets/${encodeURIComponent(n)}/containers/${s}/${o}`,{method:"POST"})}async function ta(n,s){const o=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${s}/wipe`,{method:"POST",headers:Ee,body:JSON.stringify({Confirm:s})}),e=await o.text();let d=null;try{d=e?JSON.parse(e):null}catch{}if(d&&typeof d=="object"&&"report"in d)return d;const l=d&&typeof d=="object"&&typeof d.error=="string"?d.error:o.statusText||`HTTP ${o.status}`;throw new Ne(o.status,l)}function na(n,s){return se(`/api/targets/${encodeURIComponent(n)}/containers/${s}/provision`,{method:"POST"})}async function aa(n){const s=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:Ee}),o=await s.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Ne(s.status,d)}function sa(n,s,o){return se(`/api/targets/${encodeURIComponent(n)}/containers/${s}/config`,{method:"PUT",headers:Ee,body:JSON.stringify(o)})}function mt(){return se("/api/gateways")}async function oa(n){await se(`/api/orphans/${encodeURIComponent(n)}`,{method:"DELETE"})}function jt(n){return se("/api/gateways",{method:"POST",headers:Ee,body:JSON.stringify(n)})}function qt(n){return se(`/api/gateways/${encodeURIComponent(n)}/tls/verify`)}function ra(n){return se(`/api/gateways/${encodeURIComponent(n)}/traffic`)}function dt(n){return se(`/api/gateways/${encodeURIComponent(n)}/analytics`)}function Wt(n,s=!1){const o=s?"?refresh=1":"";return se(`/api/gateways/${encodeURIComponent(n)}/capabilities${o}`)}function ia(n){return se(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function Ae(n,s){return se(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:Ee,body:JSON.stringify(s)})}function _t(n,s){return se(`/api/gateways/${encodeURIComponent(n)}/${s}`,{method:"POST"})}function ca(n){return se(`/api/gateways/${encodeURIComponent(n)}/trust-cert`,{method:"POST"})}function ut(n){return se(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Kt(n){const s=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:Ee,body:JSON.stringify({Confirm:n})}),o=await s.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Ne(s.status,d)}function la(n){return se(`/api/chainlist/${n}`)}function pt(n,s){return se(`/api/gateways/${encodeURIComponent(n)}/knownset/${s}`)}function da(){return se("/api/settings")}function ua(n){return se("/api/settings",{method:"PUT",headers:Ee,body:JSON.stringify(n)})}class Ne extends Error{constructor(o,e,d,l){super(e);Ze(this,"status");Ze(this,"hint");Ze(this,"code");this.name="ApiError",this.status=o,this.hint=d,this.code=l}}const Ee={"Content-Type":"application/json"};async function se(n,s){const o=await fetch(n,s);if(!o.ok){let d=o.statusText||`HTTP ${o.status}`,l,y;try{const p=await o.json();p&&typeof p.error=="string"&&p.error&&(d=p.error),p&&typeof p.hint=="string"&&p.hint&&(l=p.hint),p&&typeof p.code=="string"&&p.code&&(y=p.code)}catch{}throw new Ne(o.status,d,l,y)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const Pt="https://learn.valve.city/rpc";function a(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function $e(n,s){const o=n&&s&&s!==Pt?` <span class="footer-sep">·</span> <a href="${a(s)}" target="_blank" rel="noopener noreferrer">${a(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(Pt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
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
  `;const s=n.querySelector("#content"),o=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:s,setActiveNav:d=>{const l=d==="machine"?"targets":d==="home"||d==="panel"?"rpc":d;for(const y of o)y.classList.toggle("active",y.dataset.nav===l)}}}function ge(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function ha(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function fa(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const s=Math.round(n/60),o=Math.floor(s/60),e=s%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const d=Math.floor(o/24),l=o%24;return`~${d}d ${l}h`}function Y(n,s){return`<span class="badge badge-${s}">${a(n)}</span>`}function Be(n){return`<span class="dot dot-${n}"></span>`}const Rt=["B","KB","MB","GB","TB","PB"];function Fe(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let s=n,o=0;for(;s>=1024&&o<Rt.length-1;)s/=1024,o++;const e=s<10?2:s<100?1:0;return`${s.toFixed(e)} ${Rt[o]}`}async function Ge(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function Pe(n,s){n.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!n.contains(e))return;const d=e.dataset.action;d&&s(d,e,o)})}function ht(n,s,o){const e=s.find(l=>l.value===o),d=s.map(l=>`
      <li class="dropdown-option${l.value===o?" selected":""}" role="option"
          aria-selected="${l.value===o}" data-value="${a(l.value)}">
        ${a(l.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function ze(n){n.querySelectorAll(".dropdown.open").forEach(s=>{var o;s.classList.remove("open"),(o=s.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function bt(n,s){n.addEventListener("click",d=>{const l=d.target,y=l.closest(".dropdown-trigger");if(y&&n.contains(y)){const I=y.closest(".dropdown"),F=!!I&&!I.classList.contains("open");ze(n),I&&F&&(I.classList.add("open"),y.setAttribute("aria-expanded","true"));return}const p=l.closest(".dropdown-option");if(p&&n.contains(p)){const I=p.closest(".dropdown");ze(n),s((I==null?void 0:I.dataset.dropdown)??"",p.dataset.value??"");return}ze(n)});const o=d=>{if(!n.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const l=d.target;(!l.closest(".dropdown")||!n.contains(l))&&ze(n)},e=d=>{if(!n.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}d.key==="Escape"&&ze(n)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const nt="app-modal";let tt=null;function fe(n,s){ne();const o=document.createElement("div");o.className="modal-overlay",o.id=nt,o.innerHTML=`<div class="modal">${n}</div>`,o.addEventListener("click",d=>{const l=d.target.closest("[data-modal-action]");l!=null&&l.dataset.modalAction?s(l.dataset.modalAction):d.target===o&&s("cancel")});const e=d=>{d.key==="Escape"&&s("cancel")};document.addEventListener("keydown",e),tt=e,document.body.appendChild(o)}function ne(){var n;(n=document.getElementById(nt))==null||n.remove(),tt&&(document.removeEventListener("keydown",tt),tt=null)}function Ke(){return document.querySelector(`#${nt} .modal`)}function De(n){return new Promise(s=>{var d;let o=!1;const e=l=>{o||(o=!0,ne(),s(l))};fe(`
        <h2>${a(n.title)}</h2>
        <p>${a(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${a(n.confirmLabel)}</button>
        </div>
      `,l=>e(l==="confirm")),(d=document.querySelector(`#${nt} [data-modal-action="confirm"]`))==null||d.focus()})}const ot=5e3,ma=60;function ba(n,s){let o=!1,e=null,d=null,l=null,y=null;const p=[];n.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${$e()}</div>`;const I=n.querySelector("#an-body");Pe(n,($,h)=>{var x;$==="toggle-endpoint"&&((x=h.closest(".an-endpoint"))==null||x.classList.toggle("expanded"))}),F();async function F(){try{e=((await mt()).gateways??[]).find(h=>h.id===s)??null}catch($){if(o)return;l=String($ instanceof Error?$.message:$),_();return}if(!o){if(!e){_();return}await V(),y=window.setInterval(()=>void V(),ot)}}async function V(){try{const $=await dt(s);if(o)return;R($),d=$,l=null}catch($){if(o)return;l=String($ instanceof Error?$.message:$)}_()}function R($){if(!$.enabled||$.error)return;const h=p[p.length-1];h&&h.since!==$.since&&(p.length=0);const x=new Map;for(const H of $.networks??[])x.set(H.chainId,H.received);p.push({t:Date.now(),since:$.since,received:x}),p.length>ma&&p.shift()}function _(){o||(I.innerHTML=G())}function G(){return l&&!d?`<h1>Analytics</h1><p class="error">${a(l)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${B(e)}
      ${d?f(d):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${a(s)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function B($){return`
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
    `}function E(){if(!d)return"";if(!d.enabled)return"counters off";if(d.error)return"could not be read";const $=d.since?new Date(d.since):null;return $&&!Number.isNaN($.getTime())?`totals since the gateway started, ${a($.toLocaleString())}<br />re-read every ${ot/1e3}s`:`re-read every ${ot/1e3}s`}function f($){return $.enabled?$.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a($.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:g($)+ye($):`
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
        ${h.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':h.map(x=>S(x)).join("")}
      </section>
    `}function S($){const h=$.methods??[],x=$.endpoints??[],H=$.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${$.chainId}</span>
          <span class="band-name">${a($.name)}</span>
          ${z($)}
        </div>
        <div class="an-stats">
          ${M("Received",ge($.received),"what clients asked this chain for")}
          ${M("Answered",ge($.answered),"returned by one of your endpoints")}
          ${M("From cache",ge($.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${M("Failed",ge($.failed),"asked for and never answered",$.failed>0?"bad":"")}
        </div>
        ${ie($.chainId)}
        ${H?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':le("Method",h.map(W=>({label:W.method,l:W})))+le("Endpoint",x.map(W=>({label:W.upstream,l:W})))+J($)}
      </div>
    `}function M($,h,x,H=""){return`
      <div class="an-stat${H?" an-stat-"+H:""}" title="${a(x)}">
        <span class="an-stat-n">${a(h)}</span>
        <span class="an-stat-l">${a($)}</span>
      </div>
    `}function z($){const h=ee($.chainId);if(h===null)return'<span class="an-rate muted small">measuring rate…</span>';const x=Math.round((p[p.length-1].t-p[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${x}s apart.">
      ${a(h.toFixed(h<10?2:0))} req/s <span class="muted">over the last ${x}s</span>
    </span>`}function ee($){if(p.length<2)return null;const h=p[0],x=p[p.length-1],H=(x.t-h.t)/1e3;if(H<=0)return null;const W=(x.received.get($)??0)-(h.received.get($)??0);return W<0?null:W/H}function ie($){if(p.length<3)return"";const h=[];for(let w=1;w<p.length;w++){const D=p[w-1],X=p[w],u=(X.t-D.t)/1e3,v=(X.received.get($)??0)-(D.received.get($)??0);h.push(u>0&&v>=0?v/u:0)}const x=Math.max(...h);if(x<=0)return"";const H=240,W=28,Q=h.length>1?H/(h.length-1):H,b=h.map((w,D)=>`${(D*Q).toFixed(1)},${(W-w/x*W).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${x.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${H} ${W}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(x.toFixed(2))} req/s</span>
      </div>
    `}function J($){const h=[];return $.cached.count>0&&h.push(`${a(ge($.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${$.cached.mean===null?"":`, in ${a(Je($.cached.mean))} on average`}.`),$.failedLatency.count>0&&$.failedLatency.mean!==null&&h.push(`The ${a(ge($.failedLatency.count))} that failed took
         ${a(Je($.failedLatency.mean))} on average to fail.`),h.length===0?"":`<p class="muted small">${h.join(" ")}</p>`}function le($,h){return h.length===0?"":`
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
            ${h.map(x=>Z(x.label,x.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function Z($,h){return`
      <tr>
        <td><code>${a($)}</code></td>
        <td class="an-num">${ge(h.count)}</td>
        <td class="an-num">${h.mean===null?'<span class="muted">—</span>':a(Je(h.mean))}</td>
        <td>${j(h)}</td>
      </tr>
    `}function j($){const h=$.buckets??[];if(h.length===0||$.count===0)return'<span class="muted small">—</span>';let x=0;const H=[];for(const Q of h){const b=Q.count-x;x=Q.count,H.push({label:de(Q.le),n:Math.max(0,b)})}return H.reduce((Q,b)=>Q+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(H.filter(Q=>Q.n>0).map(Q=>`${Q.n} ${Q.label}`).join(" · "))}">
        ${H.map((Q,b)=>Q.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${Q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(pe(H))}</span>
    `}function pe($){for(let h=$.length-1;h>=0;h--)if($[h].n>0)return`slowest ${$[h].label}`;return""}function de($){if($==="+Inf")return"30s or more";const h=Number($);return Number.isFinite(h)?`under ${Je(h)}`:`under ${$}`}function ye($){const h=$.endpoints??[];return`
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
                     <tbody>${h.map(x=>oe(x)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function oe($){const h=$.errors??[],x=h.reduce((W,Q)=>W+Q.count,0),H=h.length>0;return`
      <tr class="an-endpoint${H?" expandable":""}" ${H?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${a($.upstream)}</code>
          ${$.chainId?`<span class="muted small">chain ${$.chainId}</span>`:""}
          ${$.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ge($.requests)}</td>
        <td class="an-num${x>0?" bad":""}">${x>0?ge(x):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${$.headLag>0?ge($.headLag):'<span class="muted">0</span>'}</td>
        <td>${we($)}</td>
      </tr>
      ${H?Ce($,h):""}
    `}function we($){const h=[];return $.scored?(h.push($.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String($.position))}</span>`),h.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a($.score.toFixed(3))}</span>`),$.primarySwitches>1&&h.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ge($.primarySwitches)} switches</span>`),$.excludedSeconds>0&&h.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(Je($.excludedSeconds))}</span>`),`<span class="an-selection">${h.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function Ce($,h){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${h.map(x=>`
                    <tr>
                      <td class="an-num">${ge(x.count)}</td>
                      <td><code>${a(x.class)}</code></td>
                      <td>${x.severity?`<span class="badge badge-${x.severity==="critical"?"bad":"warn"}">${a(x.severity)}</span>`:""}</td>
                      <td class="muted small">${a(x.method||"")}</td>
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
    `}return()=>{o=!0,y!==null&&window.clearInterval(y)}}function Je(n){return!Number.isFinite(n)||n<0?"—":n>0&&n<5e-4?"<1ms":n<1?`${Math.round(n*1e3)}ms`:n<60?`${n<10?n.toFixed(1):Math.round(n)}s`:`${Math.round(n/60)}m`}function ya(n,s){let o=!1,e=null,d=null,l=!1,y=!1;n.innerHTML=`<h1>Network diagnostics: ${a(s)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${$e()}</div>`;const p=n.querySelector("#diag-body"),I=n.querySelector("#diag-footer");Pe(n,(f,g)=>{var S;if(f==="run")V();else if(f==="toggle")(S=g.closest(".check-item"))==null||S.classList.toggle("expanded");else if(f==="copy"){const M=g.dataset.copy;M&&E(g,M)}}),F();async function F(){let f,g;try{const[M,z]=await Promise.all([Ue(),He()]);f=M.find(ee=>ee.id===s),g=z}catch(M){if(o)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(M))}</p>`;return}if(o)return;if(!f){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const S=g==null?void 0:g.networks.find(M=>M.ChainID===f.wire.ChainID);S&&(I.innerHTML=$e(S.Name,S.LearnURL));try{e=await Qn(s),y=!0}catch(M){d=String(M instanceof Error?M.message:M)}o||R()}async function V(){l=!0,d=null,R();try{e=await Xn(s),y=!0}catch(f){d=String(f instanceof Error?f.message:f)}l=!1,o||R()}function R(){p.innerHTML=`
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
      <ul class="check-list">${e.items.map(B).join("")}</ul>
    `}function G(f){var g;return((g=e==null?void 0:e.items.find(S=>S.ID===f))==null?void 0:g.Title)??f}function B(f){const g=f.Status==="pass"?"ok":f.Status==="fail"?"bad":f.Status==="warn"?"warn":"neutral",S=f.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${S?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${Y(S?"failed here":f.Status,g)}
          <strong>${a(f.Title)}</strong>
          <span class="muted small check-detail-inline">${a(f.Detail)}</span>
        </button>
        <div class="check-body">
          <details${S?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(f.Why)}</p>
          </details>
          ${f.Fix?`
                <details${S?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(f.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(f.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function E(f,g){const S=await Ge(g),M=f.textContent;f.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{o||(f.textContent=M)},1500)}return()=>{o=!0}}const va=85,rt={exec:"Execution",beacon:"Beacon"};function ga(n,s){let o=!1,e=null,d=null,l=null,y=null,p=null,I=null,F=null,V=null;const R={exec:null,beacon:null};let _=null;n.innerHTML=`<h1>Dashboard: ${a(s)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${$e()}</div>`;const G=n.querySelector("#dash-body"),B=n.querySelector("#dash-footer");G.addEventListener("click",h=>{const x=h.target.closest("[data-action]");if(!x||!G.contains(x))return;const H=x.dataset.action;if(H==="svc-action"){const W=x.dataset.svc,Q=x.dataset.kind;W&&Q&&oe(W,Q)}else if(H==="open-clear"){const W=x.dataset.svc;W&&Ce(W)}else if(H==="copy"){const W=x.dataset.copy;W&&we(x,W)}else H==="retry-du"?f():H==="retry-endpoints"&&g()}),E();async function E(){let h,x;try{const[W,Q]=await Promise.all([Ue(),He()]);h=W.find(b=>b.id===s),x=Q}catch(W){if(o)return;G.innerHTML=`<p class="error">Failed to load target: ${a(String(W))}</p>`;return}if(o)return;if(!h){G.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){G.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const H=x==null?void 0:x.networks.find(W=>W.ChainID===h.wire.ChainID);H&&(B.innerHTML=$e(H.Name,H.LearnURL)),G.innerHTML='<p class="muted">Connecting…</p>',e=_n(s,W=>{o||(S(W),d=W,l=W,M())}),f(),g()}async function f(){I=null;try{p=await Jn(s)}catch(h){p=null,I=String(h instanceof Error?h.message:h)}o||M()}async function g(){V=null;try{F=await Yn(s)}catch(h){F=null,V=String(h instanceof Error?h.message:h)}o||M()}function S(h){if(!d)return;const x=(new Date(h.at).getTime()-new Date(d.at).getTime())/1e3,H=h.execHead-d.execHead;if(x>0&&H>=0){const W=H/x;y=y===null?W:y*.7+W*.3}}function M(){if(!l)return;const h=l;G.innerHTML=`
      <p class="dash-status">${z(h)}</p>
      <div class="card-grid">
        ${de(h)}
        ${ie(h)}
        ${J(h)}
        ${le(h)}
        ${Z(h)}
        ${j()}
      </div>
      <p class="muted small">Last updated ${a(new Date(h.at).toLocaleTimeString())}</p>
    `}function z(h){return!h.execActive&&!h.beaconActive?Y("Node not running","bad"):h.execSyncing||h.beaconDistance>0?Y("Syncing","warn"):Y("Running · synced","ok")}function ee(h){const H=h.refHead>0?h.refHead-h.execHead:null,W=H!==null&&H>0&&y&&y>0?fa(H/y):H!==null&&H<=0?"caught up":"—";return{lag:H,eta:W}}function ie(h){const{lag:x,eta:H}=ee(h);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${h.execActive?h.execSyncing?Y("syncing","warn"):h.execHead===0?Y("no data","neutral"):Y("synced","ok"):Y("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ge(h.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${x!==null?ge(h.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${x!==null?ge(Math.max(x,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${H}</dd></div>
        </dl>
      </div>
    `}function J(h){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${h.beaconActive?h.beaconSlot===0?Y("no data","neutral"):h.beaconDistance===0?Y("synced","ok"):Y("syncing","warn"):Y("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ge(h.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ge(h.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function le(h){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ge(h.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ge(h.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function Z(h){const x=h.diskUsedPct>=va,H=`
      <div class="meter"><div class="meter-fill ${x?"meter-warn":""}" style="width:${Math.min(h.diskUsedPct,100)}%"></div></div>
      <p>${ha(h.diskUsedPct)} used</p>
    `;if(I)return`
        <div class="card ${x?"card-warn":""}">
          <h3>Storage</h3>
          ${H}
          <p class="error small">${a(I)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!p)return`
        <div class="card ${x?"card-warn":""}">
          <h3>Storage</h3>
          ${H}
          <p class="muted">Loading…</p>
        </div>
      `;const W=p.ExpectedExecBytes>0?Math.min(p.ExecBytes/p.ExpectedExecBytes*100,100):0,Q=p.ExpectedBeaconBytes>0?Math.min(p.BeaconBytes/p.ExpectedBeaconBytes*100,100):0,{lag:b,eta:w}=ee(h),D=b!==null&&b>0&&y!==null&&y>0;return`
      <div class="card ${x?"card-warn":""}">
        <h3>Storage</h3>
        ${H}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Fe(p.ExecBytes)} of ~${Fe(p.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${W}%"></div></div>
        ${D?`<p class="muted small">Estimated time remaining: ${a(w)}</p>`:""}
        <p class="muted small">Beacon — ${Fe(p.BeaconBytes)} of ~${Fe(p.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${Q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Fe(p.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(p.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(p.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function j(){if(V)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(V)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!F)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const h=F,x=h.ExecReachable&&!h.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",H=h.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(h.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${Be(h.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${Be(h.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.BeaconHTTP)}">Copy</button>
        </div>
        ${x}
        ${H}
      </div>
    `}function pe(h,x){const H=rt[h],W=R[h],Q=(b,w,D)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${h}" data-kind="${b}" ${W!==null||D?"disabled":""}>${W===b?ye():a(w)}</button>`;return`
      <div class="service-row">
        <span>${a(H)} ${x?Y("active","ok"):Y("down","bad")}</span>
        <div class="service-actions">
          ${Q("start","Start",x)}
          ${Q("stop","Stop",!x)}
          ${Q("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${h}" ${W!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function de(h){return`
      <div class="card">
        <h3>Services</h3>
        ${pe("exec",h.execActive)}
        ${pe("beacon",h.beaconActive)}
        ${_?`<p class="error small">${a(_)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(s)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(s)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(s)}">Diagnostics →</a>
        </p>
      </div>
    `}function ye(){return'<span class="spinner" aria-label="working"></span>'}async function oe(h,x){if(R[h]===null){R[h]=x,_=null,M();try{await Gn(s,h,x)}catch(H){_=`${rt[h]} ${x} failed: ${H instanceof Error?H.message:String(H)}`}R[h]=null,o||M()}}async function we(h,x){const H=await Ge(x),W=h.textContent;h.textContent=H?"Copied!":"Copy failed",setTimeout(()=>{o||(h.textContent=W)},1500)}function Ce(h){const x=rt[h],H=p?Fe(h==="exec"?p.ExecBytes:p.BeaconBytes):"unknown (disk usage hasn't loaded)";fe(`
        <h2>Clear ${a(x)} data</h2>
        <p class="error">
          This stops the ${a(x.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(H)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(h)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,b=>{if(b==="cancel"){ne();return}b==="confirm"&&$(h)});const W=document.getElementById("clear-confirm-input"),Q=document.getElementById("clear-confirm-btn");W==null||W.addEventListener("input",()=>{Q&&(Q.disabled=W.value.trim()!==h)}),W==null||W.focus()}async function $(h){const x=document.getElementById("clear-confirm-btn");x&&(x.disabled=!0,x.textContent="Clearing…");try{await zn(s,h),ne(),f()}catch(H){const W=Ke();if(W){const Q=document.createElement("p");Q.className="error small",Q.textContent=`Clear failed: ${H instanceof Error?H.message:String(H)}`,W.appendChild(Q)}x&&(x.disabled=!1,x.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),ne()}}const Lt=500,Nt="valve-node-app.explain-consent";function $a(n,s){let o=!1,e=null;const d=[];n.innerHTML=`
    <h1>Logs: ${a(s)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${$e()}</div>
  `;const l=n.querySelector("#logs-body"),y=n.querySelector("#logs-footer");Pe(n,E=>{E==="explain"&&V()}),p();async function p(){let E,f;try{const[S,M]=await Promise.all([Ue(),He()]);E=S.find(z=>z.id===s),f=M}catch(S){if(o)return;l.innerHTML=`<p class="error">Failed to load target: ${a(String(S))}</p>`;return}if(o)return;if(!E){l.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!E.wire){l.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const g=f==null?void 0:f.networks.find(S=>S.ChainID===E.wire.ChainID);g&&(y.innerHTML=$e(g.Name,g.LearnURL));try{const S=await Kn(s,200);if(o)return;d.push(...S)}catch(S){if(o)return;l.innerHTML=`<p class="error">Failed to load logs: ${a(String(S))}</p>`;return}I(),e=Vn(s,S=>{o||(d.push(S),d.length>Lt&&d.splice(0,d.length-Lt),I())})}function I(){const E=d.filter(g=>g.severity==="error"||g.severity==="critical");l.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(F).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${Y(String(E.length),E.length?"bad":"neutral")}</h2>
          <div class="log-lines">${E.length?E.slice().reverse().map(F).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const f=l.querySelector(".log-lines");f&&(f.scrollTop=f.scrollHeight)}function F(E){const f=E.severity||"info",g=E.learnUrl?` <a href="${a(E.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(f)}">
        <span class="log-time">${a(new Date(E.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(E.unit)}</span>
        <span class="log-sev">${a(f)}</span>
        <span class="log-text">${a(E.line)}</span>
        ${E.explain?`<div class="log-explain">${a(E.explain)}${g}</div>`:""}
      </div>
    `}async function V(){const E=d.filter(g=>g.severity==="error"||g.severity==="critical").map(g=>g.line).slice(-40);if(!(localStorage.getItem(Nt)==="1")){R(E);return}await _(E)}function R(E){const f=E.length?`<pre class="explain-excerpt">${E.map(g=>a(g)).join(`
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
    `,g=>{g==="proceed"?(localStorage.setItem(Nt,"1"),B(),_(E)):B()})}async function _(E){G('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const f=E.length?await Et(s,E):await Et(s);if(o)return;G(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(f.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${f.sentExcerpt.map(g=>a(g)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&B()})}catch(f){if(o)return;if(f instanceof Ne&&f.status===409){G(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,g=>{g==="close"&&B()});return}G(`
        <h2>Explain failed</h2>
        <p class="error">${a(f instanceof Error?f.message:String(f))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&B()})}}function G(E,f){B();const g=document.createElement("div");g.className="modal-overlay",g.id="explain-modal",g.innerHTML=`<div class="modal">${E}</div>`,g.addEventListener("click",S=>{const M=S.target.closest("[data-modal-action]");M!=null&&M.dataset.modalAction&&f(M.dataset.modalAction),S.target===g&&f("cancel")}),document.body.appendChild(g)}function B(){var E;(E=document.getElementById("explain-modal"))==null||E.remove()}return()=>{o=!0,e==null||e(),B()}}const wa="run",ka={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Ca={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Sa(n,s){let o=!1,e=null,d=null;const l={devnet:null},y={devnet:null},p={devnet:[]};let I=null;const F={devnet:!1};let V=null;const R={devnet:null},_={devnet:null};n.innerHTML=`
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
    ${$e()}
  `;const G=n.querySelector("#services-body");Pe(n,(u,v)=>{Ce(u,v)}),B();async function B(){try{const u=await Ft(s);if(o)return;e=u,d=null}catch(u){if(o)return;e=null,d=D(u)}f()}function E(u){return e==null?void 0:e.services.find(v=>v.id===u)}function f(){if(!o){if(d){G.innerHTML=`<p class="error">Could not read this machine's services: ${a(d)}</p>`;return}if(!e){G.innerHTML='<p class="muted">Loading…</p>';return}G.innerHTML=`
      ${g(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(S).join("")}
      </div>
    `}}function g(u){if(u.present&&u.reachable&&!u.hint)return`<p class="muted small">Docker: ${a(u.flavor)}${u.serverVersion?` ${a(u.serverVersion)}`:""} · reachable</p>`;const v=u.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(v)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${u.detail?`<div class="small">${a(u.detail)}</div>`:""}
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function S(u){const v=u.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(u.label)}</h2>
          ${M(u)}
        </div>
        <p class="muted small">${a(ka[u.id]??"")}</p>

        ${u.error?z(u):""}
        ${u.blocked?`<div class="banner banner-warn">${a(u.blocked)}</div>`:""}
        ${v.map(O=>`<div class="banner banner-warn">${a(O)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(u.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${u.status.Image?`<code>${a(u.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${ee(u)}

        ${ie(u)}

        <div class="card-actions">
          ${(u.actions??[]).map(O=>J(u,O)).join("")}
        </div>
        ${y[u.id]?`<p class="error small">${a(y[u.id])}</p>`:""}
        ${le(u)}

        ${Z(u)}
      </div>
    `}function M(u){switch(u.status.State){case"running":return Y("running","ok");case"created-but-stopped":return Y("stopped","warn");case"not-created":return Y("not created","neutral");default:return Y("unknown","bad")}}function z(u){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(u.error??"")}</div>
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function ee(u){if(u.status.State!=="created-but-stopped"||u.status.ExitCode===0)return"";const v=u.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${u.status.ExitCode}${v}.</p>`}function ie(u){const v=u.endpoints??[];return v.length===0?u.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":v.map(O=>`
        <div class="endpoint-row">
          ${Be("ok")}
          <span class="muted small">${a(O.label)}</span>
          <code class="endpoint-url">${a(O.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(O.url)}">Copy</button>
        </div>`).join("")}function J(u,v){const O=Ca[v];if(!O)return"";const te=l[u.id],re=v==="create"?`Create ${u.id==="devnet"?"devnet":"gateway"}`:O.label;return`
      <button class="${O.className}" data-action="svc-${v}" data-svc="${a(u.id)}"
              title="${a(O.title)}" ${te?"disabled":""}>
        ${te===v?'<span class="spinner" aria-label="working"></span>':a(re)}
      </button>
    `}function le(u){const v=p[u.id]??[];return v.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(v.join(`
`))}</pre>
      </div>
    `}function Z(u){const v=F[u.id],O=j(u);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${u.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(u.id)}">
            ${v?"Close":"Edit"}
          </button>
        </div>
        ${v?pe():`<p class="small">${O}</p>`}
        ${R[u.id]?`<p class="error small">${a(R[u.id])}</p>`:""}
        ${_[u.id]?`<p class="muted small">${a(_[u.id])}</p>`:""}
      </div>
    `}function j(u){const v=u.devnet;return v?`Chain ${v.ChainID} · a block every ${a(v.BlockTime)} · JSON-RPC on ${a(v.BindAddr)}:${v.HTTPPort} · WebSocket on ${a(v.BindAddr)}:${v.WSPort}`:"—"}function pe(u){return de()}function de(){const u=V;return u?`
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
    `:""}function ye(){F.devnet&&V&&(V.BlockTime=oe("#dev-blocktime",V.BlockTime),V.HTTPPort=we("#dev-http",V.HTTPPort),V.WSPort=we("#dev-ws",V.WSPort),V.BindAddr=oe("#dev-bind",V.BindAddr))}function oe(u,v){const O=n.querySelector(u);return O?O.value.trim():v}function we(u,v){const O=n.querySelector(u);if(!O)return v;const te=Number.parseInt(O.value.trim(),10);return Number.isFinite(te)?te:v}async function Ce(u,v){const O=v.dataset.svc??"";switch(u){case"refresh":await B();return;case"copy":v.dataset.copy&&await w(v,v.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await $(O,u.slice(4));return;case"svc-create":case"svc-recreate":await h(O);return;case"svc-wipe":W(O);return;case"toggle-config":x(O);return;case"save-config":await H(O);return;default:return}}async function $(u,v){if(!l[u]){l[u]=v,y[u]=null,f();try{await ea(s,u,v)}catch(O){y[u]=`${v} failed: ${D(O)}${X(O)}`}l[u]=null,await B()}}async function h(u){if(!l[u]){l[u]="create",y[u]=null,p[u]=["starting…"],f();try{await na(s,u)}catch(v){y[u]=`${D(v)}${X(v)}`,p[u]=[],l[u]=null,f();return}I==null||I(),I=Ye(s,v=>{if(o)return;const O=v.err?`${v.stepId}: ${v.err}`:v.line?`${v.stepId}: ${v.line}`:`${v.stepId}: done`;if(p[u]=[...(p[u]??[]).filter(re=>re!=="starting…"),O],!!v.err||v.stepId===wa&&!!v.done){I==null||I(),I=null,l[u]=null,v.err&&(y[u]="Provisioning failed — see the log below."),B();return}f()})}}function x(u){if(ye(),F[u]=!F[u],R[u]=null,_[u]=null,F[u]){const v=E(u);v!=null&&v.devnet&&(V={...v.devnet})}f()}async function H(u){var te;ye(),R[u]=null,_[u]=null;const v=V;if(!v)return;if(v.HTTPPort===v.WSPort){R[u]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",f();return}try{await sa(s,u,v)}catch(re){R[u]=D(re),f();return}const O=((te=E(u))==null?void 0:te.status.State)==="running";F[u]=!1,_[u]=O?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await B()}function W(u){const v=E(u);if(!v)return;const O=(v.restartsOnWipe??[]).map(K=>{var me;return((me=E(K))==null?void 0:me.label)??K});fe(`
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
      `,K=>{if(K==="cancel"||K==="close"){ne(),B();return}K==="confirm"&&Q(u)});const te=document.getElementById("wipe-confirm-input"),re=document.getElementById("wipe-confirm-btn");te==null||te.addEventListener("input",()=>{re&&(re.disabled=te.value.trim()!==u)}),te==null||te.focus()}async function Q(u){const v=document.getElementById("wipe-confirm-btn");v&&(v.disabled=!0,v.textContent="Wiping…");let O;try{O=await ta(s,u)}catch(te){const re=Ke();if(re){const K=document.createElement("p");K.className="error small",K.textContent=`Wipe failed: ${D(te)}${X(te)}`,re.appendChild(K)}v&&(v.disabled=!1,v.textContent=`Wipe ${u}`);return}b(u,O)}function b(u,v){const O=E(u),te=he=>{var Re;return((Re=E(he))==null?void 0:Re.label)??he},re=[];re.push(v.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const he of v.report.VolumesRemoved??[])re.push(`Volume ${he} deleted.`);for(const he of v.report.VolumesAbsent??[])re.push(`Volume ${he} was already gone.`);v.report.Recreated&&re.push("Container re-created from your saved configuration.");const K=(v.report.Cascaded??[]).map(te),me=(v.report.CascadeSkipped??[]).map(te);fe(`
        <h2>${a((O==null?void 0:O.label)??u)} wiped</h2>
        <ul class="plain-list">${re.map(he=>`<li>${a(he)}</li>`).join("")}</ul>
        ${K.length?`<p class="ok">Restarted in front of it: ${a(K.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${me.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(me.join(", "))}.</p>`:""}
        ${v.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(v.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,he=>{(he==="close"||he==="cancel")&&(ne(),B())})}async function w(u,v){const O=await Ge(v),te=u.textContent;u.textContent=O?"Copied!":"Copy failed",setTimeout(()=>{o||(u.textContent=te)},1500)}function D(u){return u instanceof Error?u.message:String(u)}function X(u){return u instanceof Ne&&u.hint?` — ${u.hint}`:""}return()=>{o=!0,I==null||I(),ne()}}const it=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Xe=8545,Qe=5052,et=30303,xa=[369,943,1],At={369:"default",943:"practise here first"};function Ta(n,s){let o=!1;const e={targetId:s,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${a(s)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${$e()}</div>`;const d=n.querySelector("#wizard-body"),l=n.querySelector("#wizard-footer");Pe(n,(b,w)=>{we(b,w)}),bt(n,(b,w)=>{b==="exec-select"?e.execId=w:b==="beacon-select"&&(e.beaconId=w),p()}),n.addEventListener("change",b=>{const w=b.target;w instanceof HTMLInputElement&&(w.id==="data-dir-input"?(Ce(),J()):w.id==="checkpoint-toggle"?(e.checkpoint=w.checked,p()):w.id==="exec-snapshot-toggle"&&(e.execSnapshot=w.checked,p()))}),y();async function y(){try{const[b,w]=await Promise.all([He(),Ue()]);if(o)return;e.catalog=b;const D=w.find(X=>X.id===s);D!=null&&D.wire&&(e.chainId=D.wire.ChainID,e.execId=D.wire.ExecID,e.beaconId=D.wire.BeaconID,e.archive=D.wire.Archive,D.wire.ExecHTTPPort&&(e.execHTTPPort=String(D.wire.ExecHTTPPort)),D.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(D.wire.BeaconHTTPPort)),D.wire.ExecP2PPort&&(e.execP2PPort=String(D.wire.ExecP2PPort)),D.wire.RPCBindAddr&&(e.rpcBindAddr=D.wire.RPCBindAddr)),p()}catch(b){if(o)return;e.loadError=String(b instanceof Error?b.message:b),p()}}function p(){if(e.loadError){d.innerHTML=`<p class="error">Failed to load: ${a(e.loadError)}</p>`;return}e.catalog&&(d.innerHTML=`
      ${Q(e.step)}
      ${F()}
    `,I())}function I(){var w;const b=(w=e.catalog)==null?void 0:w.networks.find(D=>D.ChainID===e.chainId);l.innerHTML=b?$e(b.Name,b.LearnURL):$e()}function F(){switch(e.step){case"network":return V();case"clients":return R();case"mode":return de();case"review":return ye();case"run":return oe()}}function V(){const b=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${xa.map(D=>{const X=b.networks.find(O=>O.ChainID===D);if(!X)return"";const u=e.chainId===D,v=At[D]?Y(At[D],D===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${u?"selected":""}" data-action="pick-network" data-chain-id="${D}" type="button">
          <h3>${a(X.Name)} <span class="muted">(chain ${D})</span></h3>
          ${v}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function R(){const b=e.catalog,w=b.networks.find(u=>u.ChainID===e.chainId);if(!w)return'<p class="error">Unknown network.</p>';(e.execId===null||!w.ExecClients.includes(e.execId))&&(e.execId=w.ExecClients[0]??null),(e.beaconId===null||!w.BeaconClients.includes(e.beaconId))&&(e.beaconId=w.BeaconClients[0]??null);const D=w.ExecClients.map(u=>Z(u,b)),X=w.BeaconClients.map(u=>Z(u,b));return`
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
          ${ht("exec-select",D,e.execId)}
        </label>
        ${pe(e.execId,b)}
        <label>
          Beacon client
          ${ht("beacon-select",X,e.beaconId)}
        </label>
        ${pe(e.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function _(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const G=1.1,B=.5,E="Valve reth snapshot",f="rough estimate";function g(b){return b.SnapshotSizeTB}function S(b){return b.SnapshotSizeTB*B}function M(b){return`<p class="muted small">${_(g(b))} is the measured size of Valve's reth snapshot for ${a(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function z(b){return{archive:g(b)*1e12*G,full:S(b)*1e12*G}}function ee(b,w){if(!b)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${a(w)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${a(w)}</code>: ${a(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==w)return"";const D=z(b),X=e.freeBytes>=D.archive,u=e.freeBytes>=D.full,v=`<p class="muted small">Free at <code>${a(w)}</code>: <strong>${Fe(e.freeBytes)}</strong> — archive ${X?"fits":"won't fit"} (${_(g(b))}, ${E}), full ${u?"fits":"won't fit"} (${_(S(b))}, ${f}).</p>`;let O="";return e.downgradeNote?O=`<p class="banner banner-warn">${a(e.downgradeNote)}</p>`:u||(O=`<p class="banner banner-warn">Neither full (${_(S(b))}, ${f}) nor archive (${_(g(b))}, ${E}) fits the free space here — choose a location with more room.</p>`),v+O}function ie(b,w){if(e.downgradeNote=null,!b||e.freeBytes===null)return;const D=z(b);e.archive&&e.freeBytes<D.archive&&e.freeBytes>=D.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${w} for archive (${_(g(b))}, ${E}) — switched to Full (${_(S(b))}, ${f}). Pick a location with more room to run archive.`)}async function J(){var D;if(e.chainId===null)return;const b=(D=e.catalog)==null?void 0:D.networks.find(X=>X.ChainID===e.chainId),w=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,p();try{const{freeBytes:X}=await qn(e.targetId,w);if(o)return;e.freeBytes=X,e.probedPath=w,ie(b,w)}catch(X){if(o)return;e.freeBytes=null,e.probedPath=w,e.diskError=String(X instanceof Error?X.message:X)}e.diskProbing=!1,p()}function le(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function Z(b,w){const D=w.clients.find(X=>X.id===b);return{value:b,label:D?`${D.id} — ${j(D.repo)}`:b}}function j(b){const w=b.split("/");return w.length>=4?w[3]:b}function pe(b,w){const D=b?w.clients.find(u=>u.id===b):void 0;if(!D)return"";const X=D.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(D.repo)}" target="_blank" rel="noopener noreferrer">${a(X)}</a></p>`}function de(){var te,re,K;const b=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",w=(te=e.catalog)==null?void 0:te.networks.find(me=>me.ChainID===e.chainId),D=((K=(re=e.catalog)==null?void 0:re.clients.find(me=>me.id===e.execId))==null?void 0:K.snapshotSupported)??!1,X=w?`${_(S(w))} (${f})`:"Smaller",u=w?`${_(g(w))} (${E})`:"Much larger",v=w?` on ${a(w.Name)}`:"",O=w?e.checkpoint?w.SyncLabel:w.GenesisSyncLabel:"";return`
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
              <tr><th>Approx. disk footprint${v}</th><td class="yes">${X}</td><td class="limited">${u}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${w?M(w):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${u}${w?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${X}${w?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(b)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(b)}" value="${a(e.dataDir)}" />
          </label>
          ${ee(w,e.dataDir||b)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(b)}/jwt.hex" value="${a(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Xe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Xe}" value="${a(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${a(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Qe})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Qe}" value="${a(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${a(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${et})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${et}" value="${a(e.execP2PPort)}" />
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
    `}function ye(){const w=e.catalog.networks.find(he=>he.ChainID===e.chainId),D=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,X=e.jwtPath||`${D}/jwt.hex`,u=it.map(he=>`<li>${a(he.title)}</li>`).join(""),v=H(e.execHTTPPort,Xe),O=H(e.beaconHTTPPort,Qe),te=H(e.execP2PPort,et),re=v||O||te?`<tr><th>Non-default ports</th><td>${[v?`exec HTTP ${v}`:null,O?`beacon HTTP ${O}`:null,te?`exec p2p ${te}`:null].filter(he=>he!==null).map(a).join(", ")}</td></tr>`:"",{addr:K}=$(e.rpcBindAddr),me=K?`<tr><th>RPC bind address</th><td><code>${a(K)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
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
            <tr><th>JWT secret path</th><td><code>${a(X)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${a(e.checkpointUrl||(w==null?void 0:w.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${re}
            ${me}
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
    `}function oe(){const w=e.catalog.networks.find(K=>K.ChainID===e.chainId),D=w==null?void 0:w.LearnURL,X=new Set(e.events.filter(K=>K.done).map(K=>K.stepId)),u=new Set(e.events.filter(K=>K.err).map(K=>K.stepId)),v=new Map;for(const K of e.events){if(!K.line)continue;const me=v.get(K.stepId)??[];me.push(K.line),v.set(K.stepId,me)}const O=it.map(K=>{var q;const me=X.has(K.id),he=u.has(K.id),Re=he?Y("failed","bad"):me?Y("done","ok"):Y("pending","neutral"),L=(v.get(K.id)??[]).slice(-5),U=(q=e.events.find(A=>A.stepId===K.id&&A.err))==null?void 0:q.err,T=K.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${D?` <a href="${a(D)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${me?"step-done":""} ${he?"step-error":""}">
          <div class="step-head">${Re} <strong>${a(K.title)}</strong></div>
          ${T}
          ${L.length?`<pre class="step-log">${L.map(A=>a(A)).join(`
`)}</pre>`:""}
          ${U?`<p class="error small">${a(U)}</p>`:""}
        </li>
      `}).join(""),te=e.events.some(K=>K.err),re=it.every(K=>X.has(K.id))||e.events.some(K=>K.stepId==="handshake"&&K.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${O}</ol>
        ${re&&!te?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        ${te?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function we(b,w){switch(b){case"pick-network":e.chainId=Number(w.dataset.chainId),e.execId=null,e.beaconId=null,p();break;case"goto-network":e.step="network",p();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",p();break;case"goto-mode":e.step="mode",p(),J();break;case"goto-review":if(Ce(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){p();break}e.step="review",p();break;case"start-setup":W();break}}function Ce(){const b=n.querySelectorAll('input[name="mode"]');for(const K of Array.from(b))K.checked&&(e.archive=K.value==="archive");const w=n.querySelector("#data-dir-input"),D=n.querySelector("#jwt-path-input");w&&(e.dataDir=w.value.trim()),D&&(e.jwtPath=D.value.trim());const X=n.querySelector("#exec-http-port-input"),u=n.querySelector("#beacon-http-port-input"),v=n.querySelector("#exec-p2p-port-input");X&&(e.execHTTPPort=X.value.trim()),u&&(e.beaconHTTPPort=u.value.trim()),v&&(e.execP2PPort=v.value.trim());const O=n.querySelector("#rpc-bind-addr-input");O&&(e.rpcBindAddr=O.value.trim());const te=n.querySelector("#checkpoint-url-input");te&&(e.checkpointUrl=te.value.trim());const re=n.querySelector("#snapshot-key-input");re&&(e.snapshotKey=re.value.trim()),e.execHTTPPortError=x(e.execHTTPPort).error??null,e.beaconHTTPPortError=x(e.beaconHTTPPort).error??null,e.execP2PPortError=x(e.execP2PPort).error??null,e.rpcBindAddrError=$(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?le(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function $(b){if(!b)return{};const w=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return w?w.slice(1).every(D=>Number(D)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const h=/^\d+$/;function x(b){if(!b)return{};if(!h.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const w=Number(b);return!Number.isInteger(w)||w<1||w>65535?{error:"Port must be between 1 and 65535."}:{port:w}}function H(b,w){const{port:D}=x(b);if(!(D===void 0||D===w))return D}async function W(){var v;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(v=e.streamStop)==null||v.call(e),e.streamStop=null,p();const b={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(b.DataDir=e.dataDir),e.jwtPath&&(b.JWTPath=e.jwtPath);const w=H(e.execHTTPPort,Xe),D=H(e.beaconHTTPPort,Qe),X=H(e.execP2PPort,et);w!==void 0&&(b.ExecHTTPPort=w),D!==void 0&&(b.BeaconHTTPPort=D),X!==void 0&&(b.ExecP2PPort=X);const{addr:u}=$(e.rpcBindAddr);u!==void 0&&(b.RPCBindAddr=u),e.checkpoint?e.checkpointUrl&&(b.CheckpointURL=e.checkpointUrl):b.NoCheckpoint=!0,e.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=e.snapshotKey);try{await Wn(e.targetId,b)}catch(O){if(!(O instanceof Ne&&O.status===409)){e.starting=!1,e.startError=String(O instanceof Error?O.message:O),p();return}}e.starting=!1,e.step="run",p(),e.streamStop=Ye(e.targetId,O=>{o||(e.events.push(O),e.step==="run"&&p())})}function Q(b){const w=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],X=w.map(u=>u.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${w.map((u,v)=>`<li class="${v===X?"current":v<X?"past":"future"}">${a(u.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;o=!0,(b=e.streamStop)==null||b.call(e)}}function Ia(n,s){let o=!1;const e=new Map;n.innerHTML=`<h1>${a(s)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const d=n.querySelector("#machine-body");Pe(n,(R,_)=>{R==="toggle-section"&&F(_.dataset.section??"")}),l();async function l(){let R,_;try{const[G,B]=await Promise.all([Ue(),He()]);R=G.find(E=>E.id===s),_=B}catch(G){if(o)return;d.innerHTML=`<p class="error">Failed to load machine: ${a(String(G))}</p>`;return}if(!o){if(!R){location.hash="#/targets";return}y(R,_)}}function y(R,_){const G=R.mode==="local"?"this machine":"SSH",B=R.mode==="ssh"&&R.ssh?`${a(R.ssh.User)}@${a(R.ssh.Host)}`:G;d.innerHTML=`
      <p class="muted">${B}</p>
      <p>${p(R,_)}</p>
      <div class="machine-sections">
        ${V.map(E=>I(E,R,_)).join("")}
      </div>
      ${$e()}
    `}function p(R,_){const G=R.wire;if(!G)return Y("not set up","neutral");const B=_.networks.find(f=>f.ChainID===G.ChainID),E=B?B.Name:`chain ${G.ChainID}`;return`${Y(E,"ok")} ${Y(G.ExecID,"neutral")} ${Y(G.BeaconID,"neutral")}${G.Archive?" "+Y("archive","warn"):""}`}function I(R,_,G){return`
      <section class="card machine-section" data-section-card="${a(R.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(R.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(R.title)}</span>
          <span class="machine-section-status">${R.status(_,G)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(R.key)}" hidden></div>
      </section>
    `}function F(R){const _=V.find(g=>g.key===R);if(!_)return;const G=n.querySelector(`[data-section-card="${R}"]`),B=n.querySelector(`[data-section-body="${R}"]`),E=n.querySelector(`.machine-section-head[data-section="${R}"]`);if(!G||!B||!E)return;const f=B.hidden;if(f&&!e.has(R)){const g=document.createElement("div");B.appendChild(g),e.set(R,_.mount(g))}B.hidden=!f,G.classList.toggle("open",f),E.setAttribute("aria-expanded",String(f))}const V=[{key:"setup",title:"Setup",status:R=>R.wire?Y("set up","ok"):Y("not set up","neutral"),mount:R=>Ta(R,s)},{key:"dashboard",title:"Dashboard",status:R=>R.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:R=>ga(R,s)},{key:"logs",title:"Logs",status:R=>R.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:R=>$a(R,s)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:R=>Sa(R,s)}];return()=>{o=!0;for(const R of e.values())try{R()}catch{}e.clear()}}function Ea(n){let s;try{s=new URL(n).hostname}catch{return"endpoint"}if(!s)return"endpoint";if(s==="localhost"||/^[0-9.]+$/.test(s)||/^\[.*\]$/.test(s))return s;const o=s.split(".").filter(Boolean);return o.length<=1?s:o[o.length-2]}function Vt(n){var e;if(!n)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const s=n.actions??[];if(n.blocked)return{tone:"blocked",label:"Unavailable",sub:n.blocked,actions:s,blocked:n.blocked};const o=((e=n.networks)==null?void 0:e.length)??0;return n.status.State==="running"?{tone:"on",label:"Running",sub:`${o} network${o===1?"":"s"} served`,actions:s}:{tone:"off",label:"Stopped",sub:o?`${o} network${o===1?"":"s"} configured`:"Press to start",actions:s}}function Ve(n){if(!n.running)return"off";if(!n.serviceable)return"frequent";const s=n.slowRate??0;return s>.4?"frequent":s>=.1?"occasional":"stable"}const Pa="0.5";function Gt(n){if(!n||n.count<=0||!n.buckets||n.buckets.length===0)return;const s=n.buckets.find(e=>e.le===Pa);if(!s)return;const o=n.count-s.count;return Math.max(0,Math.min(1,o/n.count))}function Ra(n){if(!n||n.length===0)return null;let s=0;const o=new Map;for(const e of n){s+=e.count;for(const d of e.buckets??[])o.set(d.le,(o.get(d.le)??0)+d.count)}return{count:s,mean:null,buckets:[...o.entries()].map(([e,d])=>({le:e,count:d}))}}function yt(n){const s=Gt(Ra(n.methods));if(s!==void 0)return s;if(n.received>0)return Math.max(0,Math.min(1,n.failed/n.received))}function vt(n,s){var e;const o=(e=n==null?void 0:n.endpoints)==null?void 0:e.find(d=>d.upstream===s);return Gt(o??null)}const La=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function gt(n){return La.map(({key:s,label:o,hot:e})=>{const d=n[s]==="supported";return{key:s,label:o,lit:d,hot:!!e&&d}})}function Na(n,s,o){const e=n.Networks??[],d=e.findIndex(p=>p.ChainID===s),l={ChainID:s,Upstreams:o},y=d===-1?[...e,l]:e.map((p,I)=>I===d?l:p);return{...n,Networks:y}}function Aa(n,s){const o=n.Networks??[];return{...n,Networks:o.filter(e=>e.ChainID!==s)}}function ct(n,s,o){const e=n.Networks??[],d=e.findIndex(F=>F.ChainID===s);if(d===-1)return{...n,Networks:[...e,{ChainID:s,Upstreams:[o]}]};const l=e[d],y=l.Upstreams.findIndex(F=>F.ID===o.ID),p=y===-1?[...l.Upstreams,o]:l.Upstreams.map((F,V)=>V===y?o:F),I={...l,Upstreams:p};return{...n,Networks:e.map((F,V)=>V===d?I:F)}}function Ba(n,s,o){const e=n.Networks??[],d=e.findIndex(p=>p.ChainID===s);if(d===-1)return{...n,Networks:e};const l=e[d],y={...l,Upstreams:l.Upstreams.filter(p=>p.ID!==o)};return{...n,Networks:e.map((p,I)=>I===d?y:p)}}function Da(n,s){if(n.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const o=n.filter(p=>!p.wire);if(o.length>0){const p=o.map(F=>F.id);return{level:"attention",sentence:p.length===1?"1 machine still needs setup.":`${p.length} machines still need setup.`,machines:p}}const e=s.networks??[],d=p=>{const I=e.find(F=>F.ChainID===p);return I?I.Name:`chain ${p}`},l=Ua(n.map(p=>d(p.wire.ChainID))),y=n.length===1?"machine":"machines";return{level:"ok",sentence:`All ${n.length} ${y} healthy — ${Ma(l)}.`,machines:[]}}function Ha(n,s){const o=s.machines.length?` <span class="verdict-machines">${s.machines.map(e=>`<a href="#/setup/${encodeURIComponent(e)}">${a(e)}</a>`).join(" ")}</span>`:"";n.innerHTML=`
    <div class="verdict-line verdict-${s.level}">
      ${Y(s.level==="ok"?"OK":"Attention",s.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(s.sentence)}</strong>${o}
    </div>
  `}function Ua(n){return[...new Set(n)]}function Ma(n){return n.length<=1?n[0]??"":n.length===2?`${n[0]} and ${n[1]}`:`${n.slice(0,-1).join(", ")} and ${n[n.length-1]}`}const Oa=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function Bt(n){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:n,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Fa=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`,be=n=>`<svg class="p-i"><use href="#p-${n}"/></svg>`,Dt="run",Ht=1337;function Ut(n){let s=null,o={name:"list"},e=null,d=null,l=null,y=null,p=null,I=[],F=null,V=null,R=!1,_=null,G=null,B=!1,E=null,f=!1,g=null,S=null,M=null,z=null,ee=null,ie=!1,J="";n.innerHTML=Fa+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const le=n.querySelector("#p-card");async function Z(){try{const L=await mt();s=ja(L.gateways),e=null}catch(L){e=ve(L)}j()}function j(){le.innerHTML=pe()}function pe(){return e?qa(e):o.name==="network"?Xa(s,o.chainId,{caps:F,capsBusy:R,capsErr:_,tls:G,tlsBusy:B,tlsErr:E,copyFlash:f,error:g,netHealth:ee,busy:l}):o.name==="endpoint"?es(s,o.chainId,o.upstreamId,{caps:F,capsBusy:R,capsErr:_,health:M,copyFlash:f,error:S,netHealth:ee}):Wa(s,l,y,I,ee)}async function de(L,U){R=!0,j();try{F=await Wt(L,U),V=L,_=null}catch(T){F=null,V=L,_=ve(T)}R=!1,j()}async function ye(L,U){if(!(!U&&z===L&&M)){j();try{M=await dt(L),z=L}catch{M=null,z=L}j()}}function oe(){var T;if(!s)return"";const L=s.status.State==="running",U=[];for(const q of s.networks??[]){const A=(T=ee==null?void 0:ee.networks)==null?void 0:T.find(ce=>ce.chainId===q.chainId),N=A?yt(A):void 0;U.push(`n${q.chainId}:${Ve({running:L,serviceable:q.serviceable,slowRate:N})}`);for(const ce of q.upstreams??[]){const ke=A?vt(A,ce.id):void 0;U.push(`u${q.chainId}/${ce.id}:${Ve({running:L,serviceable:!ce.problem,slowRate:ke})}`)}}return U.join("|")}let we=null;async function Ce(L=!1){if(!s)return;if(ie){if(!L)return;await we,await Ce(!0);return}ie=!0;const U=s.id;we=(async()=>{try{ee=await dt(U)}catch{}})(),await we,ie=!1,we=null;const T=oe();T!==J&&(J=T,j())}async function $(L,U){var A;const T=(A=L.networks)==null?void 0:A.find(N=>N.chainId===U);if(await De({title:"Remove network",body:`Stop serving ${(T==null?void 0:T.name)??`chain ${U}`}?`,confirmLabel:"Remove",danger:!0})){g=null,j();try{await Ae(L.id,Aa(L.config,U))}catch(N){g=`Could not remove the network: ${ve(N)}`,j();return}o={name:"list"},j(),await K(L.id)}}async function h(L,U,T){var ce;const q=(ce=L.networks)==null?void 0:ce.find(ke=>ke.chainId===U),A=H(L,U,T);if(await De({title:"Remove endpoint",body:`Stop routing to ${(A==null?void 0:A.label)??"this endpoint"}? The gateway keeps balancing across whatever else remains on ${(q==null?void 0:q.name)??`chain ${U}`}.`,confirmLabel:"Remove",danger:!0})){S=null,j();try{await Ae(L.id,Ba(L.config,U,T))}catch(ke){S=`Could not remove the endpoint: ${ve(ke)}`,j();return}o={name:"network",chainId:U},j(),await K(L.id)}}Pe(le,(L,U)=>{x(L,U)});async function x(L,U){if(L==="setup"){if(l)return;await me();return}if(L==="power"){if(!s||l)return;const T=Vt(s);if(T.tone==="blocked")return;if(s.status.State==="running"&&T.actions.includes("stop")){await re(s.id,"stop");return}if(T.actions.includes("start")){await re(s.id,"start");return}if(T.actions.includes("create")){await K(s.id);return}return}if(L==="open-network"){o={name:"network",chainId:Number(U.dataset.chainId)},g=null,G=null,E=null,f=!1,j(),s&&V!==s.id&&de(s.id,!1);return}if(L==="back"){o={name:"list"},f=!1,j();return}if(L==="back-to-network"){const T=Number(U.dataset.chainId);o=Number.isFinite(T)?{name:"network",chainId:T}:{name:"list"},S=null,f=!1,j();return}if(L==="add-network"){if(!s||l)return;await O(s);return}switch(L){case"gw-start":case"gw-stop":case"gw-restart":s&&!l&&await re(s.id,L.slice(3));return;case"gw-create":case"gw-recreate":s&&!l&&await K(s.id);return;case"gw-wipe":s&&!l&&await he(s);return;case"copy-url":{const T=U.dataset.url??"";if(!T)return;await Ge(T)&&(f=!0,j(),window.setTimeout(()=>{f=!1,j()},1200));return}case"verify-tls":{if(!s||B)return;B=!0,E=null,j();try{G=await qt(s.id)}catch(T){E=ve(T)}B=!1,j();return}case"open-endpoint":{const T=Number(U.dataset.chainId),q=U.dataset.upstreamId??"";if(!Number.isFinite(T)||!q)return;o={name:"endpoint",chainId:T,upstreamId:q},S=null,f=!1,j(),s&&V!==s.id&&de(s.id,!1),s&&z!==s.id&&ye(s.id,!1);return}case"add-endpoint":{if(!s||l||o.name!=="network")return;u(s,o.chainId);return}case"remove-network":{if(!s||l||o.name!=="network")return;await $(s,o.chainId);return}case"rename-endpoint":{if(!s||l||o.name!=="endpoint")return;const T=H(s,o.chainId,o.upstreamId);if(!T)return;b(s.id,o.chainId,T.id,T.label);return}case"edit-address":{if(!s||l||o.name!=="endpoint")return;const T=H(s,o.chainId,o.upstreamId);if(!T||T.kind!=="external")return;D(s.id,o.chainId,T.id,T.endpoint);return}case"remove-endpoint":{if(!s||l||o.name!=="endpoint")return;await h(s,o.chainId,o.upstreamId);return}case"recheck":{if(!s)return;const T=[de(s.id,!0),Z(),Ce(!0)];o.name==="endpoint"&&T.push(ye(s.id,!0)),await Promise.all(T);return}default:return}}function H(L,U,T){var q,A,N;return(N=(A=(q=L.networks)==null?void 0:q.find(ce=>ce.chainId===U))==null?void 0:A.upstreams)==null?void 0:N.find(ce=>ce.id===T)}function W(L,U,T){var q,A;return(A=(q=L.config.Networks)==null?void 0:q.find(N=>N.ChainID===U))==null?void 0:A.Upstreams.find(N=>N.ID===T)}function Q(L){const U=Ke();if(!U)return;const T=document.createElement("p");T.className="error small",T.textContent=L,U.appendChild(T)}function b(L,U,T,q){fe(`
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
      `,N=>{if(N==="cancel"){ne();return}N==="save"&&w(L,U,T)});const A=document.getElementById("ep-rename-input");A==null||A.focus(),A==null||A.select()}async function w(L,U,T){if(!s)return;const q=W(s,U,T);if(!q){ne();return}const A=document.getElementById("ep-rename-input"),N=document.getElementById("ep-rename-save"),ce=(A==null?void 0:A.value.trim())??"";A&&(A.disabled=!0),N&&(N.disabled=!0,N.textContent="Saving…");const ke={...q,Name:ce||void 0};try{await Ae(L,ct(s.config,U,ke))}catch(Ie){Q(`Could not rename the endpoint: ${ve(Ie)}`),A&&(A.disabled=!1),N&&(N.disabled=!1,N.textContent="Save");return}ne(),await K(L)}function D(L,U,T,q){fe(`
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
      `,N=>{if(N==="cancel"){ne();return}N==="save"&&X(L,U,T)});const A=document.getElementById("ep-addr-input");A==null||A.focus(),A==null||A.select()}async function X(L,U,T){if(!s)return;const q=document.getElementById("ep-addr-input"),A=document.getElementById("ep-addr-save"),N=(q==null?void 0:q.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(N)){Q("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}const ce=W(s,U,T);if(!ce){ne();return}q&&(q.disabled=!0),A&&(A.disabled=!0,A.textContent="Saving…");const ke={...ce,Endpoint:N};try{await Ae(L,ct(s.config,U,ke))}catch(Ie){Q(`Could not save the address: ${ve(Ie)}`),q&&(q.disabled=!1),A&&(A.disabled=!1,A.textContent="Save");return}ne(),await K(L)}function u(L,U){var T;fe(`
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
      `,q=>{if(q==="cancel"){ne();return}q==="add"&&v(L.id,U)}),(T=document.getElementById("ep-add-input"))==null||T.focus()}async function v(L,U){if(!s)return;const T=document.getElementById("ep-add-input"),q=document.getElementById("ep-add-save"),A=(T==null?void 0:T.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(A)){Q("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}T&&(T.disabled=!0),q&&(q.disabled=!0,q.textContent="Adding…");const N={ID:crypto.randomUUID(),Kind:"external",Endpoint:A,Local:!1,RecentOnly:!1,Name:Ea(A)};try{await Ae(L,ct(s.config,U,N))}catch(ce){Q(`Could not add the endpoint: ${ve(ce)}`),T&&(T.disabled=!1),q&&(q.disabled=!1,q.textContent="Add endpoint");return}ne(),await K(L,()=>void de(L,!0))}async function O(L){l="add-network",y=null,j();let U;try{U=(await He()).networks??[]}catch(A){l=null,y=`Could not load the network catalog: ${ve(A)}`,j();return}l=null,j();const T=new Set((L.networks??[]).map(A=>A.chainId)),q=U.filter(A=>!T.has(A.ChainID)).map(A=>({chainId:A.ChainID,name:A.Name}));if(T.has(Ht)||q.push({chainId:Ht,name:"Devnet"}),q.length===0){y="Every network valve's catalog knows about is already configured on this gateway.",j();return}fe(`
        <h2>Add a network</h2>
        <ul class="plain-list rpc-picker">
          ${q.map(A=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="pick:${A.chainId}">
                <span>${a(A.name)}</span>
                <span class="muted small">chain ${A.chainId}</span>
              </button>
            </li>`).join("")}
        </ul>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,A=>{if(A==="cancel"){ne();return}if(A.startsWith("pick:")){const N=Number.parseInt(A.slice(5),10);if(!Number.isFinite(N))return;ne(),te(L.id,N)}})}async function te(L,U){if(!s||l)return;l="create",y=null,j();let T;try{T=((await pt(L,U)).endpoints??[]).filter(N=>!N.alreadyAdded).map(N=>N.url)}catch(A){l=null,y=`Could not read valve's known set for chain ${U}: ${ve(A)}`,j();return}if(T.length===0){l=null,y=`valve has no measured endpoints for chain ${U} yet, so there was nothing to add.`,j();return}const q=T.map((A,N)=>({ID:`public-${U}-${N+1}`,Kind:"external",Endpoint:A,Local:!1,RecentOnly:!1}));try{await Ae(L,Na(s.config,U,q))}catch(A){l=null,y=`Could not add the network: ${ve(A)}`,j();return}l=null,await K(L,()=>{o={name:"network",chainId:U},j(),de(L,!0)})}async function re(L,U){if(!l){l=U,y=null,j();try{await _t(L,U)}catch(T){y=`${U} failed: ${ve(T)}`}l=null,await Z()}}async function K(L,U){if(l)return;l="create",y=null,j();let T;try{T=await ut(L)}catch(q){y=ve(q),l=null,j();return}p==null||p(),p=Ye(T.targetId,q=>{(q.err||q.stepId===Dt&&q.done)&&(p==null||p(),p=null,l=null,q.err&&(y=`Provisioning failed: ${q.err}`),Z().then(()=>{q.err||U==null||U()}))})}async function me(){if(l)return;l="setup",y=null,I=[],j();const L=N=>{I=[...I,N],j()},U=(N,ce)=>{l=null,y=ce?`${N} — ${ce}`:N,j()};L("Preparing your endpoint…");try{(await Ue()).some(ce=>ce.id==="local")||await lt({id:"local",mode:"local"})}catch(N){U(`Could not register this machine: ${ve(N)}`,We(N));return}try{const N=await Ft("local");if(!N.docker.reachable){U(N.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",N.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(N){U(`Could not check Docker on this machine: ${ve(N)}`,We(N));return}L("Creating the gateway…");let T="default";try{T=(await jt({id:T,placement:{targetId:"local",backend:"docker"},config:Bt([])})).id}catch(N){U(`Could not create the gateway: ${ve(N)}`,We(N));return}L("Adding Ethereum and PulseChain endpoints…");const q=[];for(const{chainId:N}of Oa)try{const ke=((await pt(T,N)).endpoints??[]).filter(Ie=>!Ie.alreadyAdded).map(Ie=>Ie.url);if(ke.length===0)continue;q.push({ChainID:N,Upstreams:ke.map((Ie,at)=>({ID:`public-${N}-${at+1}`,Kind:"external",Endpoint:Ie,Local:!1,RecentOnly:!1}))})}catch(ce){U(`Could not read valve's set for chain ${N}: ${ve(ce)}`,We(ce));return}if(q.length===0){U("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await Ae(T,Bt(q))}catch(N){U(`Could not save the endpoints: ${ve(N)}`,We(N));return}L("Starting the gateway… the first run pulls the eRPC and Caddy images.");let A;try{A=await ut(T)}catch(N){U(`Could not start the gateway: ${ve(N)}`,We(N));return}p==null||p(),p=Ye(A.targetId,N=>{const ce=N.err?`${N.stepId}: ${N.err}`:N.line?`${N.stepId}: ${N.line}`:`${N.stepId}: done`;L(ce),(N.err||N.stepId===Dt&&N.done)&&(p==null||p(),p=null,l=null,N.err&&(y=`Provisioning failed: ${N.err}`),I=[],Z())})}async function he(L){if(await De({title:`Wipe ${L.label}`,body:`This destroys ${L.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){l="wipe",y=null,j();try{const T=await Kt(L.id);T.error&&(y=T.error)}catch(T){y=`wipe failed: ${ve(T)}`}l=null,await Z()}}let Re=!1;return Z().then(()=>{Re||(J=oe(),d=window.setInterval(()=>{Ce()},5e3))}),()=>{Re=!0,d&&window.clearInterval(d),p==null||p()}}function ja(n){return!n||n.length===0?null:n.find(s=>s.placement.targetId==="local")??n[0]}function ve(n){return n instanceof Error?n.message:String(n)}function We(n){return n instanceof Ne?n.hint:void 0}function qa(n){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(n)}</div>`}function Wa(n,s,o,e,d){var p;if(n===null)return _a(s,o,e);const l=Vt(n),y=(p=n==null?void 0:n.networks)!=null&&p.length?n.networks.map((I,F)=>Ya(n,I,F>0,d)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${a(l.sub)}</span>
    </div>
    <div class="p-band">
      ${Ga(n,l,s,o)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${y}
      <div class="p-row p-rowdiv addr${s?" p-disabled":""}" data-action="add-network"${s?' aria-disabled="true"':""}>
        <span class="p-lead">${be("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function _a(n,s,o){const e=n==="setup",d=s?`<div class="p-emptyerr">${a(s)}</div>`:"",l=o.length?`<div class="p-setup-log" aria-live="polite">${o.map(y=>`<div>${a(y)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${e?" disabled":""}>
        <div class="p-pbtn off big${e?" busy":""}">${be("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${d}
      ${l}
    </div>
  `}function Ka(n,s){return s.tone==="blocked"?null:n.status.State==="running"&&s.actions.includes("stop")?"stop":s.actions.includes("start")?"start":s.actions.includes("create")?"create":null}const Va={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},Mt={restart:"refresh",recreate:"refresh",wipe:"trash"};function Ga(n,s,o,e){const d=s.tone==="blocked"?s.blocked??"":s.sub,l=o?" busy":"",y=e?`<div class="p-ps" style="color:var(--red)">${a(e)}</div>`:"",p=s.tone==="blocked"&&(n!=null&&n.hint)?`<div class="p-ps">${a(n.hint)}</div>`:"",I=`
    <div class="p-power${l}" data-action="power">
      <div class="p-pbtn ${s.tone}">${be("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(s.label)}</div>
        <div class="p-ps"${s.tone==="blocked"?' style="color:var(--red)"':""}>${a(d)}</div>
        ${p}
        ${y}
      </div>
    </div>
  `,F=n?za(n,s,o):"";return I+F}function za(n,s,o){const e=Ka(n,s),d=(n.actions??[]).filter(y=>y!==e);return d.length===0?"":`<div class="p-chips">${d.map(y=>{const p=Va[y]??y,I=Mt[y]?be(Mt[y]):"";return`<button type="button" class="p-chip${y==="wipe"?" danger":""}" data-action="gw-${y}" data-gid="${a(n.id)}"${o?" disabled":""}>${I}${a(p)}</button>`}).join("")}</div>`}const $t={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function Ja(n){return n.map(s=>`<svg class="p-i${s.hot?" hot":s.lit?" on":""}"><use href="#p-${$t[s.key]}"/></svg>`).join("")}function Ya(n,s,o,e){var I;const d=(I=e==null?void 0:e.networks)==null?void 0:I.find(F=>F.chainId===s.chainId),l=d?yt(d):void 0,y=Ve({running:n.status.State==="running",serviceable:s.serviceable,slowRate:l}),p=gt({});return`
    <div class="p-row${o?" p-rowdiv":""}" data-action="open-network" data-chain-id="${s.chainId}">
      <span class="p-lead"><span class="p-dot ${y}"></span></span>
      <span class="p-nm">${a(s.name)}</span>
      <span class="p-caps">${Ja(p)}</span>
      <span class="p-chev">${be("chevR")}</span>
    </div>
  `}function zt(n,s){var o;return s==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(o=(n.capabilities??[]).find(e=>e.key===s))==null?void 0:o.status}function Za(n,s,o){const e=((n==null?void 0:n.endpoints)??[]).filter(l=>l.chainId===s&&o.includes(l.upstream)),d={};for(const l of["http","ws","archive","trace"])e.some(y=>zt(y,l)==="supported")&&(d[l]="supported");return d}function Xa(n,s,o){var ie,J,le;const e=(ie=n==null?void 0:n.networks)==null?void 0:ie.find(Z=>Z.chainId===s);if(!n||!e)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${be("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${s}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;const d=n.status.State==="running",l=(le=(J=o.netHealth)==null?void 0:J.networks)==null?void 0:le.find(Z=>Z.chainId===s),y=l?yt(l):void 0,p=Ve({running:d,serviceable:e.serviceable,slowRate:y}),I=e.upstreams??[],F=o.tls??n.tls.verification??null,V=(F==null?void 0:F.ok)===!0,R=o.tlsBusy?"Verifying…":V?`Verified ${F?new Date(F.at).toLocaleString():""}`:"Verify HTTPS now",_=o.tlsErr?`<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${a(o.tlsErr)}</div>`:"",G=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${V?"green":"dim"}" data-action="verify-tls" title="${a(R)}">${be("lock")}</span>
          <span class="p-ic ${o.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(e.url??"")}" title="Copy the gateway URL">${be("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${a(e.url||"—")}</div>
      ${_}
    </div>
  `,B=I.map((Z,j)=>{const pe=l?vt(l,Z.id):void 0,de=Ve({running:d,serviceable:!Z.problem,slowRate:pe});return`
        <div class="p-row${j>0?" p-rowdiv":""}" data-action="open-endpoint" data-chain-id="${e.chainId}" data-upstream-id="${a(Z.id)}">
          <span class="p-lead"><span class="p-dot ${de}"></span></span>
          <span class="p-nm">${a(Z.label)}</span>
          <span class="p-chev">${be("chevR")}</span>
        </div>
      `}).join(""),E=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${I.length}</span></div>
      ${B}
      <div class="p-row${I.length>0?" p-rowdiv":""} addr${o.busy?" p-disabled":""}" data-action="add-endpoint"${o.busy?' aria-disabled="true"':""}>
        <span class="p-lead">${be("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `,f=Za(o.caps,s,I.map(Z=>Z.id)),g=gt(f),S=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${o.capsBusy&&!o.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':o.capsErr&&!o.caps?`<div class="p-caprow p-caperr">Couldn't check capabilities — ${a(o.capsErr)}</div>`:`<div class="p-caprow">${g.map(Z=>`<span class="p-capitem${Z.lit?" lit":""}">${be($t[Z.key])}${a(Z.label)}</span>`).join("")}</div>`}
    </div>
  `,M=d?e.serviceable?"Healthy":"Unserviceable":"Stopped",z=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${be("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${p}"></span> ${a(M)}</span></div>
    </div>
  `,ee=o.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(o.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${be("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${p}"></span> <span class="p-nmtxt">${a(e.name)}</span></span>
    </div>
    ${G}
    ${E}
    ${S}
    ${z}
    ${ee}
    <div class="p-band p-remove" data-action="remove-network">${be("trash")} Remove network</div>
  `}function Qa(n,s,o){const e=((n==null?void 0:n.endpoints)??[]).find(l=>l.chainId===s&&l.upstream===o);if(!e)return{};const d={};for(const l of["http","ws","archive","trace"])zt(e,l)==="supported"&&(d[l]="supported");return d}function es(n,s,o,e){var z,ee,ie,J,le;const d=(z=n==null?void 0:n.networks)==null?void 0:z.find(Z=>Z.chainId===s),l=(ee=d==null?void 0:d.upstreams)==null?void 0:ee.find(Z=>Z.id===o);if(!n||!d||!l)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${be("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Endpoint</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This endpoint is no longer configured.</div>
    `;const y=n.status.State==="running",p=(J=(ie=e.netHealth)==null?void 0:ie.networks)==null?void 0:J.find(Z=>Z.chainId===s),I=p?vt(p,o):void 0,F=Ve({running:y,serviceable:!l.problem,slowRate:I}),V=l.kind==="external",R=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Address</span>
        <span class="p-acts"><span class="p-ic ${e.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(l.endpoint)}" title="Copy the endpoint URL">${be("copy")}</span></span>
      </div>
      <div class="p-gwurl"${V?' data-action="edit-address" style="cursor:text"':""}>${a(l.endpoint||"—")}</div>
    </div>
  `,_=Qa(e.caps,s,o),G=gt(_),B=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${e.capsBusy&&!e.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':e.capsErr&&!e.caps?`<div class="p-caprow p-caperr">Couldn't check capabilities — ${a(e.capsErr)}</div>`:`<div class="p-caprow">${G.map(Z=>`<span class="p-capitem${Z.lit?" lit":""}">${be($t[Z.key])}${a(Z.label)}</span>`).join("")}</div>`}
    </div>
  `,E=y?l.problem?l.problem:"Healthy":"Stopped",f=(((le=e.health)==null?void 0:le.endpoints)??[]).find(Z=>Z.chainId===s&&Z.upstream===o),g=f&&f.scored&&f.headLag>0?`<div class="p-srow"><span class="p-k">Chain head</span><span class="p-v">behind ${ge(f.headLag)} block${f.headLag===1?"":"s"}</span></div>`:"",S=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${be("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${F}"></span> ${a(E)}</span></div>
      ${g}
    </div>
  `,M=e.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(e.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${be("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${F}"></span> <span class="p-nmtxt">${a(l.label)}</span> <span class="p-pen" data-action="rename-endpoint">${be("pencil")}</span></span>
    </div>
    ${R}
    ${B}
    ${S}
    ${M}
    <div class="p-band p-remove" data-action="remove-endpoint">${be("trash")} Remove endpoint</div>
  `}function ts(n,s){let o=!1,e=[],d=null,l=!1,y=!1;n.innerHTML=`<h1>Security: ${a(s)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${$e()}</div>`;const p=n.querySelector("#sec-body"),I=n.querySelector("#sec-footer");Pe(n,(B,E)=>{var f;if(B==="rerun")V();else if(B==="toggle")(f=E.closest(".check-item"))==null||f.classList.toggle("expanded");else if(B==="copy"){const g=E.dataset.copy;g&&G(E,g)}}),F();async function F(){let B,E;try{const[g,S]=await Promise.all([Ue(),He()]);B=g.find(M=>M.id===s),E=S}catch(g){if(o)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(g))}</p>`;return}if(o)return;if(!B){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!B.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const f=E==null?void 0:E.networks.find(g=>g.ChainID===B.wire.ChainID);f&&(I.innerHTML=$e(f.Name,f.LearnURL)),await V()}async function V(){l=!0,d=null,R();try{e=await Zn(s),y=!0}catch(B){d=String(B instanceof Error?B.message:B)}l=!1,o||R()}function R(){p.innerHTML=`
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
    `}function _(B){const E=B.Status==="pass"?"ok":B.Status==="fail"?"bad":B.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${Y(B.Status,E)}
          <strong>${a(B.Title)}</strong>
          <span class="muted small check-detail-inline">${a(B.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${a(B.Why)}</p>
          </details>
          ${B.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(B.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(B.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function G(B,E){const f=await Ge(E),g=B.textContent;B.textContent=f?"Copied!":"Copy failed",setTimeout(()=>{o||(B.textContent=g)},1500)}return()=>{o=!0}}const ns=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],ft="VALVE_API_KEY";function as(n){return n===ft?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(n)}}</code> slot wherever an endpoint URL carries one.`}function ss(n){let s=!1,o=!1,e=!1,d=null,l=!1,y=null,p=null;const I=new Set,F=new Map;let V="",R="";n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${$e()}`;const _=n.querySelector("#settings-body");Pe(n,(S,M)=>{if(S==="save"&&g(),S==="clear-key"){if(!y)return;o=!0;const z=n.querySelector("#ai-key");z&&(z.value=""),f(y)}if(S==="clear-provider-key"){const z=M.dataset.key;if(!y||!z)return;I.add(z),F.set(z,""),l=!1,f(y)}}),bt(n,(S,M)=>{S!=="ai-provider"||!y||(p=M,l=!1,f(y))}),G();async function G(){try{const S=await da();if(s)return;y=S,f(S)}catch(S){if(s)return;_.innerHTML=`<p class="error">Failed to load settings: ${a(String(S))}</p>`}}function B(S){const z=(Array.isArray(S.providerKeysSet)?S.providerKeysSet:[]).filter(ee=>ee!==ft).sort();return[ft,...z]}function E(S,M){const z=a(S);return`
      <div class="pk-row">
        <label>
          <code>${z}</code>
          <input class="provider-key" data-key="${z}" type="password" autocomplete="off"
                 placeholder="${M?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${as(S)}</p>
        ${M?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${z}">Clear saved key</button>`:""}
      </div>`}function f(S){var Z;const M=p??S.aiProvider,z=Array.isArray(S.providerKeysSet)?S.providerKeysSet:[],ee=B(S).map(j=>E(j,z.includes(j))).join("");_.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${ht("ai-provider",ns.map(j=>({value:j.value,label:j.label})),M)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${S.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${S.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${ee}
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
            <input id="ref-rpc-base" type="text" value="${a(S.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${a(d)}</p>`:""}
        ${l?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const ie=n.querySelector("#ai-key");ie==null||ie.addEventListener("input",()=>{o=!0,l=!1}),(Z=n.querySelector("#ref-rpc-base"))==null||Z.addEventListener("input",()=>{l=!1}),n.querySelectorAll("input.provider-key").forEach(j=>{const pe=j.dataset.key;if(!pe)return;const de=F.get(pe);de!==void 0&&(j.value=de),j.addEventListener("input",()=>{I.add(pe),F.set(pe,j.value),l=!1})});const J=n.querySelector("#pk-new-value");J&&(J.value=R),J==null||J.addEventListener("input",()=>{R=J.value,l=!1});const le=n.querySelector("#pk-new-name");le==null||le.addEventListener("input",()=>{V=le.value,l=!1})}async function g(){const S=n.querySelector("#ai-key"),M=n.querySelector("#ref-rpc-base");if(!S||!M||!y)return;const z={aiProvider:p??y.aiProvider,refRpcBase:M.value.trim()};o&&(z.aiKey=S.value);const ee={};for(const J of I)ee[J]=F.get(J)??"";const ie=V.trim();ie&&(ee[ie]=R),Object.keys(ee).length>0&&(z.providerKeys=ee),e=!0,d=null,l=!1,f(y);try{const J=await ua(z);if(s)return;y=J,o=!1,I.clear(),F.clear(),V="",R="",e=!1,l=!0,f(J)}catch(J){if(s)return;e=!1,d=String(J instanceof Error?J.message:J),f(y)}}return()=>{s=!0}}const os=["http","ws","archive","trace"],rs={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},_e=1337,is="run",cs={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function ls(n){let s=!1,o=null,e=null;const d={},l={},y={},p={},I={},F={},V={},R={},_={},G={},B={},E={},f={},g={},S={};let M="",z=null;n.innerHTML=`
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
    ${$e()}
  `;const ee=n.querySelector("#rpc-body");Pe(n,(t,r)=>{pn(t,r)}),bt(n,()=>{}),J(),ie();async function ie(){try{const t=await Ot();if(s)return;M=t.os,oe()}catch{}}async function J(){try{const t=await mt();if(s)return;o=t,e=null}catch(t){if(s)return;o=null,e=Se(t)}oe();for(const t of(o==null?void 0:o.gateways)??[])le(t.id),Z(t.id,!1)}async function le(t){try{const r=await ra(t);if(s)return;d[t]=r}catch{if(s)return;d[t]=null}oe()}async function Z(t,r){y[t]=r,r&&oe();try{const i=await Wt(t,r);if(s)return;l[t]=i}catch{if(s)return;l[t]=null}y[t]=!1,oe()}function j(t){return((o==null?void 0:o.gateways)??[]).find(r=>r.id===t)}function pe(t,r){return(t.networks??[]).find(i=>i.chainId===r)}function de(t,r,i){var m;const c=(((m=d[t])==null?void 0:m.networks)??[]).find(C=>C.chainId===r);return((c==null?void 0:c.upstreams)??[]).find(C=>C.upstream===i)}function ye(t,r,i){var c;return(((c=l[t])==null?void 0:c.endpoints)??[]).find(m=>m.chainId===r&&m.upstream===i)}function oe(){if(s)return;if(e){ee.innerHTML=`<p class="error">Could not read the gateways: ${a(e)}</p>`;return}if(!o){ee.innerHTML='<p class="muted">Loading…</p>';return}const t=o.gateways??[],r=t.length>1,i=(o.targets??[]).some(C=>It(C.id,t)),c=new Set(t.map(C=>C.placement.targetId)),m=(o.orphans??[]).filter(C=>!c.has(C.targetId));ee.innerHTML=`
      ${t.map(C=>$(C,r)).join("")}
      ${t.length===0?Ce():""}
      ${m.map(we).join("")}
      ${i?`<div class="card-actions rpc-add-gateway">
               <button class="btn${t.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${t.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function we(t){const r=`docker rm -f ${t.containerName}`,i=f[t.containerName];return`
      <div class="strip">
        ${O({tone:"warn",text:`${t.containerName} is still running on ${t.targetId}. Its chains were folded into ${t.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:r})}
        ${i?O({tone:"bad",text:i}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(t.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function Ce(){return((o==null?void 0:o.targets)??[]).length===0?`
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
    `}function $(t,r){return`
      ${r?`<h2 class="rpc-machine">${a(t.placement.targetId)}</h2>`:""}
      ${h(t)}
      ${v(t)}
      ${he(t)}
      ${Re(t)}
      ${b(t)}
    `}function h(t){const r=t.status.State==="running",i=t.tls,c=[`on <strong>${a(t.placement.targetId)}</strong>`];return t.status.Image&&c.push(`<code>${a(t.status.Image)}</code>`),c.push(i!=null&&i.enabled?`HTTPS front <code>${a(i.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${K(t)}
        <strong>${a(t.label)}</strong>
        ${re(t)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${r?`base <code>${a(t.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function x(t){const r=t.tls;return r!=null&&r.enabled&&r.rootCaPath&&r.effectiveCertSource==="internal"?r.rootCaPath:null}function H(t){var r;return((r=((o==null?void 0:o.targets)??[]).find(i=>i.id===t.placement.targetId))==null?void 0:r.mode)??""}function W(t){switch(t){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return t||"this device"}}function Q(t,r,i){switch(t){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${r}"`;case"windows":return`certutil -addstore -f ROOT "${r}"`;case"linux":default:return`sudo cp "${r}" /usr/local/share/ca-certificates/valve-node-app-${i}.crt && sudo update-ca-certificates`}}function b(t){const r=_[t.id]??!1,i=((o==null?void 0:o.orphans)??[]).filter(c=>c.targetId===t.placement.targetId);return`
      <section class="card manage-section${r?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(t.id)}" aria-expanded="${r}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${w(t,i.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${r?D(t,i):""}
      </section>
    `}function w(t,r){const i=[];return t.status.State!=="running"&&i.push("gateway not running"),r>0&&i.push(`${r} leftover container${r===1?"":"s"}`),i.length===0?"container, settings, certificate":i.join(" · ")}function D(t,r){var i;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(t.actions??[]).map(c=>me(t,c)).join("")}
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
        ${X(t)}
        ${r.map(we).join("")}
        ${V[t.id]?sn(t):""}
      </div>
    `}function X(t){const r=x(t);if(!r)return"";const i=H(t)==="local",c=Q(M,r,t.id),m=S[t.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(t.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(r)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(r)}">Copy path</button>
        </div>
        ${i?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(t.id)}" ${g[t.id]?"disabled":""}>
                   ${g[t.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${m?u(m):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${i?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${M?` (${a(W(M))})`:""}:</span>
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
    `}function v(t){const r=[];t.error&&r.push({tone:"bad",text:`This gateway could not be read: ${t.error}${t.hint?` — ${t.hint}`:""}`}),t.blocked&&r.push({tone:"warn",text:t.blocked});for(const c of t.warnings??[])r.push({tone:"warn",text:c});r.push(...te(t));const i=I[t.id];return i&&r.push({tone:"bad",text:i}),r.length===0?"":`<div class="strip">${r.map(O).join("")}</div>`}function O(t){return`
      <div class="strip-line strip-${t.tone}">
        <span class="strip-text">${a(t.text)}</span>
        ${t.cmd?`<code class="strip-cmd">${a(t.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(t.cmd)}">Copy</button>`:""}
      </div>
    `}function te(t){var m,C;const r=t.tls;if(!(r!=null&&r.enabled))return[];const i=[];r.fallback&&i.push({tone:"warn",text:r.fallback}),r.error?i.push({tone:"warn",text:`HTTPS front: ${r.error}`}):((m=r.status)==null?void 0:m.State)!=="running"&&i.push({tone:"warn",text:`The HTTPS front is ${((C=r.status)==null?void 0:C.State)??"unknown"}, so nothing answers on ${r.url??"its https URL"} even if the gateway itself is up.`,cmd:r.containerName?`docker start ${r.containerName}`:void 0});const c=G[t.id]??r.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&i.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&i.push({tone:"warn",text:c.expiryWarning}),i}function re(t){switch(t.status.State){case"running":return Y("running","ok");case"created-but-stopped":return Y("stopped","warn");case"not-created":return Y("not created","neutral");default:return Y("unknown","bad")}}function K(t){return t.status.State==="running"?Be("ok"):t.status.State==="unknown"?Be("bad"):Be("neutral")}function me(t,r){const i=cs[r];if(!i)return"";const c=p[t.id];return`
      <button class="${i.className}" data-action="gw-${r}" data-gid="${a(t.id)}"
              title="${a(i.title)}" ${c?"disabled":""}>
        ${c===r?'<span class="spinner" aria-label="working"></span>':a(i.label)}
      </button>
    `}function he(t){const r=F[t.id]??[];return r.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(t.placement.targetId)}</p>
        <pre class="step-log">${a(r.join(`
`))}</pre>
      </div>
    `}function Re(t){const r=L(t.networks??[]),i=r.some(c=>c.chainId===_e);return r.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(t.id)}">Add a network</button>
            ${N(t,i)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${r.map(c=>U(t,c)).join("")}
        </div>
        ${A(t,i)}
        ${an(t)}
      </div>
    `}function L(t){const r=t.filter(c=>c.chainId!==_e),i=t.filter(c=>c.chainId===_e);return[...r,...i]}function U(t,r){const i=ke(r),c=r.chainId===_e,m=`${t.id}:${r.chainId}`,C=R[m]??!1,P=i.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${i.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(r.name)}</span>
          <code class="chain-key">evm:${r.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${Y(P,i.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(m)}" aria-expanded="${C}">
              ${C?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${T(t,r)}
        ${C?q(t,r,i):""}
      </section>
    `}function T(t,r){if(!r.url)return`<p class="chain-connect-none muted small">${t.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const i=x(t);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(r.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(r.url)}"
                title="Copy ${a(r.url)}">Copy URL</button>
        ${i?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${H(t)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(t.id)}" ${g[t.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${g[t.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(i)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(t.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${S[t.id]?`<span class="chain-cert muted small">${a(S[t.id].ok?"Trusted — reload your wallet or browser.":S[t.id].message)}</span>`:""}`:""}
      </div>
    `}function q(t,r,i){const c=r.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${i.why?" chain-verdict-why":""}"${i.why?` title="${a(i.why)}"`:""}>${i.html}</p>
        <div class="chain-detail-bar">
          ${ce(c.length,i.tone,r.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(t.id)}" data-chain="${r.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(t.id)}" data-chain="${r.chainId}">Remove</button>
        </div>
        ${Yt(t,r)}
        ${(r.warnings??[]).map(m=>`<p class="chain-note">${a(m)}</p>`).join("")}
      </div>
    `}function A(t,r){const i=l[t.id],c=i!=null&&i.at?`probed ${a(kt(i.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(t.id)}">+ Network</button>
        ${N(t,r)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${y[t.id]?"disabled":""}>
          ${y[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function N(t,r){return r?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(t.id)}"
                    title="Add a throwaway local test chain (evm:${_e}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function ce(t,r,i){const c=i>0,m=c?i:t,C=Math.min(t,m);let P="";for(let qe=0;qe<m;qe++)P+=`<span class="seg${qe<C?` seg-on seg-${r}`:""}"></span>`;const k=c&&t>i,ae=c?k?`${t} (set is ${i})`:`${t} of ${i}`:`${t}`,ue=`${t} upstream${t===1?"":"s"} configured`,xe=c?`${ue}${k?`, ${t-i} beyond the set`:""}. valve's set for this chain is ${i}.`:`${ue}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(xe)}">${P}</span>
      <span class="segs-n">${ae}</span>
    `}function ke(t){const r=t.upstreams??[];if(r.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!t.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!r.some(Ie)){const c=at(r);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(C=>`<code>${a(C)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(r.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!r.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const i=r.filter(c=>!!c.problem);if(i.length>0){const c=r.length-i.length;return{tone:"warn",html:`${i.length} of these ${r.length} endpoints ${i.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${r.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function Ie(t){return/^wss?:\/\//i.test((t.endpoint??"").trim())}function at(t){const r=new Set;for(const i of t){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((i.endpoint??"").trim());c&&r.add(c[1].toLowerCase())}return[...r].sort()}function Yt(t,r){const i=r.upstreams??[];return i.length===0?"":`<ul class="ups">${i.map(c=>Zt(t,r,c)).join("")}</ul>`}function Zt(t,r,i){const c=`${t.id}|${r.chainId}|${i.id}`,m=i.actions??[];return`
      <li class="up${i.problem?" up-bad":""}">
        <div class="up-what">
          ${i.problem?Be("bad"):Be("ok")}
          <span class="up-label">${a(i.label)}</span>
          ${Xt(i)}
        </div>
        <code class="up-url">${a(i.endpoint||"—")}</code>
        <div class="up-caps">${Qt(t,r,i)}</div>
        <div class="up-share">${nn(t,r,i)}</div>
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
    `}function Xt(t){return t.problem?Y("unusable","bad"):t.recentOnly?Y("recent blocks","warn"):t.local?Y("yours","ok"):Y("public","neutral")}function wt(t,r){var i;if(t)return r==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(i=(t.capabilities??[]).find(c=>c.key===r))==null?void 0:i.status}function Qt(t,r,i){const c=ye(t.id,r.chainId,i.id);return c?c.unprobeable?`<span class="caps-none" title="${a(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${os.map(m=>en(t,r,c,m)).join("")}</span>`:`<span class="muted small">${l[t.id]===void 0?"probing…":"—"}</span>`}function en(t,r,i,c){const m=(i.capabilities??[]).find(ue=>ue.key===c),C=wt(i,c)??"inconclusive",P=rs[c]??c.toUpperCase();let k="cap";C==="unsupported"?k=tn(t,r,c)?"cap missing":"cap off":C==="inconclusive"?k="cap unknown":C==="inconsistent"&&(k="cap mixed");const ae=m!=null&&m.detail?`${m.label}: ${m.detail}`:c==="http"&&i.reachDetail?`Answers JSON-RPC over HTTP: ${i.reachDetail}`:`${P}: no verdict`;return`<span class="${k}" title="${a(ae)}">${a(P)}</span>`}function tn(t,r,i){const c=(r.upstreams??[]).map(m=>ye(t.id,r.chainId,m.id)).filter(m=>!!m&&!m.unprobeable);return c.length>0&&c.every(m=>wt(m,i)==="unsupported")}function nn(t,r,i){const c=d[t.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const m=de(t.id,r.chainId,i.id),C=(c.networks??[]).find(xe=>xe.chainId===r.chainId);if(!m||!C||C.attributed===0)return'<span class="muted small">no traffic yet</span>';const P=Math.round(m.actual*100),k=Math.round(m.intended*100),ae=m.diverged?i.local?"warn":"":"ok",ue=`${m.succeeded.toLocaleString()} of ${C.attributed.toLocaleString()} answered requests · routing intends ${k}%`+(m.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(ue)}">
        <span class="bar">
          <span class="fill${ae?" "+ae:""}" style="width:${P}%"></span>
          <span class="tick" style="left:${k}%"></span>
        </span>
        <span class="share-n${m.diverged?" warn":""}">${P}%</span>
        ${m.unconfigured?Y("not in config","warn"):""}
      </span>
    `}function an(t){const r=d[t.id];return r?r.enabled?r.error?`<p class="muted small">The request counters could not be read: ${a(r.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${r.since?` (${a(kt(r.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function kt(t){const r=new Date(t);return Number.isNaN(r.getTime())?t:r.toLocaleString()}function sn(t){const r=t.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${a(t.id)}-port" value="${r.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${a(t.id)}-bind" value="${a(r.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${a(r.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${on(t)}
        ${rn(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(t.id)}">Save settings</button>
        </div>
      </div>
    `}function on(t){const r=!t.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${a(t.id)}-metrics" ${r?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function rn(t){var P;const r=a(t.id),i=t.config.TLS??null,c=(i==null?void 0:i.Enabled)??!1,m=(i==null?void 0:i.CertSource)||"internal",C=((P=t.tls)==null?void 0:P.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${r}-tls" ${c?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${r}-tls-host" value="${a((i==null?void 0:i.Hostname)??C)}"
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
        <input type="text" inputmode="numeric" id="gw-${r}-tls-port" value="${(i==null?void 0:i.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${r}-tls-source">
          <option value="internal" ${m==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${m==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${r}-tls-cert" value="${a((i==null?void 0:i.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${r}-tls-key" value="${a((i==null?void 0:i.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${cn(t)}
    `}function cn(t){var P,k;const r=a(t.id),i=((P=t.config.TLS)==null?void 0:P.Enabled)??!1,c=G[t.id]??((k=t.tls)==null?void 0:k.verification)??null,m=B[t.id]??!1,C=E[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${r}" ${i&&!m?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${m?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${i?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${C?`<p class="error small">${a(C)}</p>`:""}
      ${c?ln(c):""}
    `}function ln(t){const r=(t.assertions??[]).map(i=>`
          <li class="small">
            ${dn(i.status)}
            <strong>${a(i.title)}</strong>
            <div class="muted">${a(i.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${t.ok?t.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${a(t.summary)}
      </div>
      <ul class="verify-list">${r}</ul>
      <p class="muted small">
        Checked ${a(new Date(t.at).toLocaleString())} against <code>${a(t.address)}</code>
        ${t.notAfter?`· certificate valid until <code>${a(new Date(t.notAfter).toLocaleString())}</code> (${a(t.expiresIn??"")})`:""}
      </p>
      ${t.expiryWarning?`<div class="banner banner-warn">${a(t.expiryWarning)}</div>`:""}
    `}function dn(t){switch(t){case"pass":return Y("pass","ok");case"fail":return Y("fail","bad");case"unavailable":return Y("unavailable","warn");default:return Y("skipped","neutral")}}async function un(t){B[t]=!0,E[t]=null,oe();try{G[t]=await qt(t)}catch(r){E[t]=`${Se(r)}${je(r)}`}finally{B[t]=!1,oe()}}function Me(t){return{...t.config,Networks:(t.config.Networks??[]).map(r=>({ChainID:r.ChainID,Upstreams:r.Upstreams.map(i=>({...i}))}))}}async function Oe(t,r,i){I[t]=null;try{await Ae(t,r)}catch(c){return I[t]=`${i?i+": ":""}${Se(c)}`,oe(),!1}return await J(),!0}async function pn(t,r){const i=r.dataset.gid??"";switch(t){case"refresh":await J();return;case"copy":r.dataset.copy&&await Un(r,r.dataset.copy);return;case"reprobe":await Z(i,!0);return;case"toggle-settings":V[i]=!V[i],oe();return;case"toggle-manage":_[i]=!_[i],oe();return;case"toggle-chain-detail":{const c=r.dataset.key??"";c&&(R[c]=!R[c]),oe();return}case"save-settings":await hn(i);return;case"verify-tls":await un(i);return;case"trust-cert":await bn(i);return;case"gw-start":case"gw-stop":case"gw-restart":await yn(i,t.slice(3));return;case"gw-create":case"gw-recreate":await vn(i);return;case"gw-wipe":An(i);return;case"add-gateway":Dn();return;case"forget-gateway":await gn(i);return;case"dismiss-orphan":await $n(r.dataset.name??"");return;case"add-chain":wn(i);return;case"add-devnet":{const c=j(i);if(c){const m=((o==null?void 0:o.targets)??[]).some(C=>C.id===c.placement.targetId&&C.hasDevnet);St(i,_e,m)}return}case"remove-chain":await Sn(i,Number.parseInt(r.dataset.chain??"",10));return;case"add-endpoint":Tt(i,Number.parseInt(r.dataset.chain??"",10));return;case"remove-endpoint":await xn(r.dataset.key??"");return;case"reset-devnet":await Ln(r.dataset.key??"",r.dataset.target??"");return;default:return}}async function hn(t){const r=j(t);if(!r)return;const i=Me(r),c=n.querySelector(`#gw-${CSS.escape(t)}-port`),m=n.querySelector(`#gw-${CSS.escape(t)}-bind`);if(c){const k=Number.parseInt(c.value.trim(),10);Number.isFinite(k)&&(i.Port=k)}m&&(i.BindAddr=m.value.trim());const C=n.querySelector(`#gw-${CSS.escape(t)}-metrics`);C&&(i.MetricsOff=!C.checked),i.TLS=fn(t,r);const P=r.status.State==="running";await Oe(t,i,"Saving settings")&&(V[t]=!1,P&&(I[t]=null,mn(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),oe())}function fn(t,r){var C,P,k,ae,ue,xe,qe;const i=Mn=>n.querySelector(`#gw-${CSS.escape(t)}-${Mn}`),c=i("tls");if(!c)return r.config.TLS??null;const m=Number.parseInt(((C=i("tls-port"))==null?void 0:C.value.trim())??"",10);return{Enabled:c.checked,Hostname:((P=i("tls-host"))==null?void 0:P.value.trim())??"",CertSource:((k=i("tls-source"))==null?void 0:k.value)??"internal",CertFile:((ae=i("tls-cert"))==null?void 0:ae.value.trim())??"",KeyFile:((ue=i("tls-key"))==null?void 0:ue.value.trim())??"",HTTPSPort:Number.isFinite(m)?m:443,BindAddr:((xe=r.config.TLS)==null?void 0:xe.BindAddr)??"",ImageRef:((qe=r.config.TLS)==null?void 0:qe.ImageRef)??""}}function mn(t,r){F[t]=[r]}async function bn(t){if(!g[t]){g[t]=!0,S[t]=null,oe();try{S[t]=await ca(t)}catch(r){S[t]={ok:!1,message:`${Se(r)}${je(r)}`}}g[t]=!1,oe()}}async function yn(t,r){if(!p[t]){p[t]=r,I[t]=null,oe();try{await _t(t,r)}catch(i){I[t]=`${r} failed: ${Se(i)}${je(i)}`}p[t]=null,await J()}}async function vn(t){if(p[t])return;p[t]="create",I[t]=null,F[t]=["starting…"],oe();let r;try{r=await ut(t)}catch(i){I[t]=`${Se(i)}${je(i)}`,F[t]=[],p[t]=null,oe();return}z==null||z(),z=Ye(r.targetId,i=>{if(s)return;const c=i.err?`${i.stepId}: ${i.err}`:i.line?`${i.stepId}: ${i.line}`:`${i.stepId}: done`;if(F[t]=[...(F[t]??[]).filter(C=>C!=="starting…"),c],!!i.err||i.stepId===is&&!!i.done){z==null||z(),z=null,p[t]=null,i.err&&(I[t]="Provisioning failed — see the log below."),J();return}oe()})}async function gn(t){const r=j(t);if(!(!r||!await De({title:`Forget ${r.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${r.containerName}" on ${r.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await ia(t)}catch(c){I[t]=Se(c),oe();return}await J()}}async function $n(t){if(t){f[t]=null;try{await oa(t)}catch(r){f[t]=Se(r),oe();return}await J()}}function wn(t){const r=j(t);if(!r)return;const i=new Set((r.networks??[]).map(k=>k.chainId)),c=(o==null?void 0:o.presets)??[],m=c.filter(k=>!i.has(k.chainId)),C=c.filter(k=>i.has(k.chainId)),P=((o==null?void 0:o.targets)??[]).some(k=>k.id===r.placement.targetId&&k.hasDevnet);fe(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${a(r.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${m.map(k=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${k.chainId}">
                <span>${a(k.name)}</span>
                <span class="muted small">chain ${k.chainId}${k.devnet?P?" · uses the devnet on "+a(r.placement.targetId):" · will create a devnet on "+a(r.placement.targetId):""}</span>
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
      `,k=>{if(k==="cancel"){ne();return}if(k==="custom"){kn(t);return}if(k.startsWith("preset:")){const ae=Number.parseInt(k.slice(7),10),ue=c.find(xe=>xe.chainId===ae);ne(),ue!=null&&ue.devnet?St(t,ae,P):Ct(t,ae)}})}function kn(t){var r;fe(`
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
      `,i=>{if(i==="cancel"){ne();return}if(i!=="add")return;const c=document.getElementById("custom-chain-id"),m=document.getElementById("custom-chain-err"),C=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(C)||C<=0){m&&(m.className="error small"),m&&(m.textContent="A chain id is a positive whole number.");return}ne(),Ct(t,C)}),(r=document.getElementById("custom-chain-id"))==null||r.focus()}async function Ct(t,r){const i=j(t);if(!i)return;const c=Me(i),m=c.Networks??[];m.some(C=>C.ChainID===r)||(m.push({ChainID:r,Upstreams:[]}),c.Networks=m,await Cn(t,c)&&(oe(),Tt(t,r)))}async function Cn(t,r){var C;const i={...r,Networks:(r.Networks??[]).filter(P=>P.Upstreams.length>0)};if(!await Oe(t,i))return!1;const m=j(t);if(m)for(const P of r.Networks??[])P.Upstreams.length===0&&!(m.networks??[]).some(k=>k.chainId===P.ChainID)&&(m.config.Networks=[...m.config.Networks??[],{ChainID:P.ChainID,Upstreams:[]}],m.networks=[...m.networks??[],{chainId:P.ChainID,name:((C=((o==null?void 0:o.presets)??[]).find(k=>k.chainId===P.ChainID))==null?void 0:C.name)??`Chain ${P.ChainID}`,path:`/${m.config.ProjectID}/evm/${P.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function St(t,r,i){const c=j(t);if(!c)return;if(!i){fe(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${a(c.placement.targetId)}</code>, so adding chain ${r} here
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
        `,()=>ne());return}const m=Me(c),C=m.Networks??[],P={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},k=C.find(ae=>ae.ChainID===r);k?k.Upstreams.push(P):C.push({ChainID:r,Upstreams:[P]}),m.Networks=C,await Oe(t,m,"Adding the devnet")}async function Sn(t,r){const i=j(t);if(!i||!Number.isFinite(r))return;const c=pe(i,r);if(!await De({title:`Remove ${(c==null?void 0:c.name)??`chain ${r}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${r}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const C=Me(i);C.Networks=(C.Networks??[]).filter(P=>P.ChainID!==r),await Oe(t,C,"Removing the network")}function xt(t){const r=t.split("|");return r.length!==3?null:{gid:r[0],chainId:Number.parseInt(r[1],10),upstreamId:r[2]}}async function xn(t){const r=xt(t);if(!r)return;const i=j(r.gid);if(!i)return;const c=Me(i),m=(c.Networks??[]).find(k=>k.ChainID===r.chainId);if(!m)return;const C=m.Upstreams.findIndex((k,ae)=>(k.ID||`${r.chainId}-${ae}`)===r.upstreamId);C<0||!await De({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(m.Upstreams.splice(C,1),await Oe(r.gid,c,"Removing the endpoint"))}function Tt(t,r){const i=j(t);if(!i||!Number.isFinite(r))return;const c=((o==null?void 0:o.sources)??[]).filter(k=>k.chainId===r),m=pe(i,r),C=new Set(((m==null?void 0:m.upstreams)??[]).filter(k=>k.kind!=="external").map(k=>`${k.kind}|${k.targetId??""}`)),P=c.filter(k=>!C.has(`${k.kind}|${k.targetId}`));fe(`
        <h2>Add an endpoint for ${a((m==null?void 0:m.name)??`chain ${r}`)}</h2>
        ${P.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${P.map(k=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${a(k.kind)}:${a(k.targetId)}">
                       <span>${a(k.label)}</span>
                       <span class="muted small">${a(k.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${r}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn" data-modal-action="known-set">Add valve's set…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,k=>{if(k==="cancel"){ne();return}if(k==="known-set"){En(t,r);return}if(k==="manual"){Rn(t,r);return}if(k.startsWith("source:")){const[,ae,ue]=k.split(":");ne(),Tn(t,r,ae,ue)}})}async function Tn(t,r,i,c){const m=j(t);if(!m)return;const C=Me(m),P=C.Networks??[],k={ID:`${i==="managed-devnet"?"devnet":"node"}-${c}`,Kind:i,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},ae=P.find(ue=>ue.ChainID===r);ae?ae.Upstreams.push(k):P.push({ChainID:r,Upstreams:[k]}),C.Networks=P,await Oe(t,C,"Adding the endpoint")}function In(t){const r=[...t].sort((m,C)=>(m.latencyMs??1e9)-(C.latencyMs??1e9)),i=r.slice(0,3),c=r.find(m=>m.url.startsWith("wss://")||m.url.startsWith("ws://"));return c&&!i.some(m=>m.url===c.url)&&(i.length===3&&i.pop(),i.push(c)),new Set(i.map(m=>m.url))}async function En(t,r){let i;try{i=await pt(t,r)}catch(k){fe(`<h2>Endpoints for chain ${r}</h2>
         <p class="error small">Could not read the set: ${a(Se(k))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>ne());return}if(s)return;const c=i.endpoints??[],m=c.filter(k=>!k.alreadyAdded).map(k=>k.url),C=new Set(c.map(k=>k.provider)).size,P=c.map(k=>{const ae=[k.websocket?'<span class="t ws">websocket</span>':"",k.archive?'<span class="t ar">archive</span>':"",k.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(k.url)}</code>
                  <span class="muted small">${a(k.provider)}</span> ${ae}</li>`}).join("");fe(`<h2>Endpoints for chain ${r}</h2>
       ${c.length?`<p class="muted small">${C} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${P}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${i.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${m.length?"":" disabled"}>
           ${m.length?`Add ${m.length}`:"Nothing to add"}</button>
       </div>`,k=>{ne(),k==="add"&&st(t,r,m),k==="discover"&&Pn(t,r)})}async function Pn(t,r){fe(`
        <h2>Public endpoints for chain ${r}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,P=>{P==="cancel"&&ne()});let i;try{i=await la(r)}catch(P){const k=Ke();if(k){const ae=document.createElement("p");ae.className="error small",ae.textContent=`Could not discover endpoints: ${Se(P)}`,k.appendChild(ae)}return}if(s)return;const c=(i.endpoints??[]).filter(P=>P.status==="live"||P.status==="unprobed"),m=(i.endpoints??[]).filter(P=>P.status==="rejected"),C=In(c);fe(`
        <h2>Public endpoints for chain ${r}</h2>
        ${i.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${i.fetchError?`<div class="small">${a(i.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(P=>{const k=C.has(P.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${a(P.url)}"${k}>
                       <span><code>${a(P.url)}</code></span>
                       <span class="muted small">${P.status==="live"?`answered in ${P.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${r} right now.</p>`}
        ${m.length?`<details class="rpc-rejected">
                 <summary class="muted small">${m.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${m.map(P=>`<li class="muted small"><code>${a(P.url)}</code> — ${a(P.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,P=>{if(P==="cancel"){ne();return}if(P==="add"){const k=Ke(),ae=k?Array.from(k.querySelectorAll('input[type="checkbox"]:checked')).map(ue=>ue.value):[];ne(),st(t,r,ae);return}})}function Rn(t,r){var i;fe(`
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
      `,c=>{if(c==="cancel"){ne();return}if(c!=="add")return;const m=document.getElementById("manual-endpoint"),C=document.getElementById("manual-recent"),P=document.getElementById("manual-err"),k=(m==null?void 0:m.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(k)){P&&(P.className="error small",P.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}ne(),st(t,r,[k],(C==null?void 0:C.checked)??!1)}),(i=document.getElementById("manual-endpoint"))==null||i.focus()}async function st(t,r,i,c=!1){if(!i.length)return;const m=j(t);if(!m)return;const C=Me(m),P=C.Networks??[];let k=P.find(ue=>ue.ChainID===r);k||(k={ChainID:r,Upstreams:[]},P.push(k));let ae=1;for(const ue of k.Upstreams){const xe=/^public-\d+-(\d+)$/.exec(ue.ID??"");xe&&(ae=Math.max(ae,Number(xe[1])+1))}for(const ue of i)k.Upstreams.some(xe=>xe.Endpoint===ue)||k.Upstreams.push({ID:`public-${r}-${ae++}`,Kind:"external",Endpoint:ue,Local:!1,RecentOnly:c});C.Networks=P,await Oe(t,C,i.length===1?"Adding the endpoint":`Adding ${i.length} endpoints`)}async function Ln(t,r){const i=xt(t);if(!i||!r||!await De({title:"Reset this devnet",body:`The chain on ${r} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;p[i.gid]="reset",I[i.gid]=null,oe();let m;try{m=await aa(r)}catch(C){I[i.gid]=`Reset failed: ${Se(C)}${je(C)}`,p[i.gid]=null,oe();return}p[i.gid]=null,Nn(r,m),await J()}function Nn(t,r){const i=[];i.push(r.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),r.report.Recreated&&i.push("A fresh chain was started from genesis.");const c=r.report.Cascaded??[],m=r.report.CascadeSkipped??[];fe(`
        <h2>Devnet on ${a(t)} reset</h2>
        <ul class="plain-list">${i.map(C=>`<li>${a(C)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${a(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${m.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(m.join(", "))}.</p>`:""}
        ${r.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>ne())}function An(t){const r=j(t);if(!r)return;fe(`
        <h2>Wipe ${a(r.label)}</h2>
        <p class="error">This destroys ${a(r.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(t)}</button>
        </div>
      `,m=>{if(m==="cancel"||m==="close"){ne(),J();return}m==="confirm"&&Bn(t)});const i=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");i==null||i.addEventListener("input",()=>{c&&(c.disabled=i.value.trim()!==t)}),i==null||i.focus()}async function Bn(t){const r=document.getElementById("wipe-confirm-btn");r&&(r.disabled=!0,r.textContent="Wiping…");let i;try{i=await Kt(t)}catch(c){const m=Ke();if(m){const C=document.createElement("p");C.className="error small",C.textContent=`Wipe failed: ${Se(c)}${je(c)}`,m.appendChild(C)}r&&(r.disabled=!1,r.textContent=`Wipe ${t}`);return}fe(`
        <h2>${a(t)} wiped</h2>
        <ul class="plain-list">
          <li>${i.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${i.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${i.error?`<p class="error small">${a(i.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{ne(),J()})}function It(t,r){return!r.some(i=>{var c;return((c=i.placement)==null?void 0:c.targetId)===t})}function Dn(){var C;const t=(o==null?void 0:o.targets)??[],r=(o==null?void 0:o.gateways)??[],i=t.filter(P=>It(P.id,r)),c=new Set(r.map(P=>P.id));if(t.length===0){fe(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>ne());return}if(i.length===0){fe(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>ne());return}const m=c.has("default")?"":"default";fe(`
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
            ${i.map(P=>`<option value="${a(P.id)}">${a(P.id)} (${a(P.mode)})</option>`).join("")}
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
      `,P=>{if(P==="cancel"){ne();return}P==="create"&&Hn()}),(C=document.getElementById("new-gw-id"))==null||C.focus()}async function Hn(){const t=document.getElementById("new-gw-id"),r=document.getElementById("new-gw-target"),i=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),m=(t==null?void 0:t.value.trim())??"",C=(r==null?void 0:r.value)??"",P=Number.parseInt((i==null?void 0:i.value.trim())??"",10),k=ae=>{c&&(c.className="error small",c.textContent=ae)};if(!m){k("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!C){k("Pick the machine it runs on.");return}try{await jt({id:m,placement:{targetId:C,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(P)?P:4e3,Networks:[]}})}catch(ae){k(Se(ae));return}ne(),await J()}async function Un(t,r){const i=await Ge(r),c=t.textContent;t.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{s||(t.textContent=c)},1500)}function Se(t){return t instanceof Error?t.message:String(t)}function je(t){return t instanceof Ne&&t.hint?` — ${t.hint}`:""}return()=>{s=!0,z==null||z(),ne()}}const ds="local";function us(n){let s=!1,o=!1,e="",d=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${$e()}
  `;const l=n.querySelector("#targets-body");Pe(n,(f,g)=>{V(f,g)}),y();async function y(){try{const[f,g,S]=await Promise.all([Ue(),He(),Ot()]);if(s)return;e=S.os,I(f,g)}catch(f){if(s)return;l.innerHTML=`<p class="error">Failed to load machines: ${a(String(f))}</p>`}}function p(){d&&I(d.targets,d.catalog)}function I(f,g){d={targets:f,catalog:g};const S=e==="linux",M=[...f].sort((J,le)=>(J.mode==="local"?-1:0)-(le.mode==="local"?-1:0)),z=M.length?`<div class="card-grid">${M.map(J=>ps(J,g,J.mode!=="local"||S,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',ee=f.some(J=>J.mode==="local");l.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${z}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${F(S,ee)}
        ${o?hs():""}
      </section>
    `;const ie=l.querySelector("#fleet-verdict");ie&&Ha(ie,Da(f,g))}function F(f,g){const S=`
      <div class="card">
        <h3>A server over SSH ${Y("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${f?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${f?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,M=f?`
        <div class="card">
          <h3>This machine ${Y("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${a(e)})`:""} ${Y("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return g?`<div class="card-grid card-grid-wide">${S}</div>`:`<div class="card-grid card-grid-wide">${f?M+S:S+M}</div>`}async function V(f,g){var S;if(f==="add-local"){await R();return}if(f==="delete-target"){const M=g.dataset.id;if(!M||!await De({title:"Remove machine",body:`Remove "${M}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await _(M);return}if(f==="toggle-ssh"){o=!o,E(),p(),o&&((S=n.querySelector("#ssh-host"))==null||S.focus());return}f==="add-ssh"&&await G()}async function R(){E();try{await lt({id:ds,mode:"local"}),await y()}catch(f){B(f)}}async function _(f){try{await jn(f),await y()}catch(g){B(g)}}async function G(){const f=n.querySelector("#ssh-host"),g=n.querySelector("#ssh-user"),S=n.querySelector("#ssh-key"),M=n.querySelector("#ssh-port"),z=n.querySelector("#ssh-id");if(!f||!g||!S||!M||!z)return;const ee=f.value.trim(),ie=g.value.trim(),J=S.value.trim(),le=M.value.trim(),Z=z.value.trim();if(E(),!ee||!ie||!J){B(new Error("host, user, and key path are required"));return}const j=Z||fs(ee),pe={Host:ee,User:ie,KeyPath:J};if(le){const ye=Number.parseInt(le,10);if(!Number.isFinite(ye)||ye<=0){B(new Error("port must be a positive number"));return}pe.Port=ye}const de=n.querySelector("#ssh-submit");de&&(de.disabled=!0,de.textContent="Connecting…");try{await lt({id:j,mode:"ssh",ssh:pe}),o=!1,await y()}catch(ye){B(ye),de&&(de.disabled=!1,de.textContent="Add server")}}function B(f){let g=n.querySelector("#targets-error");g||(l.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),g=n.querySelector("#targets-error")),g.textContent=String(f instanceof Error?f.message:f)}function E(){var f;(f=n.querySelector("#targets-error"))==null||f.remove()}return()=>{s=!0}}function ps(n,s,o,e){const d=n.wire,l=n.mode==="local"?"this machine":"SSH",y=n.mode==="ssh"&&n.ssh?`${a(n.ssh.User)}@${a(n.ssh.Host)}`:l;let p;if(!d&&!o)p=`${Y("can't run a node","warn")} ${Y(e||"not Linux","neutral")}`;else if(!d)p=Y("not set up","neutral");else{const I=s.networks.find(V=>V.ChainID===d.ChainID),F=I?I.Name:`chain ${d.ChainID}`;p=`${Y(F,"ok")} ${Y(d.ExecID,"neutral")} ${Y(d.BeaconID,"neutral")}${d.Archive?" "+Y("archive","warn"):""}`}return`
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
  `}function fs(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ms=document.querySelector("#app"),{contentEl:bs,setActiveNav:ys}=pa(ms);let Te=null;function vs(){const s=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(s.length===0)return{screen:"home"};const[o,e]=s;return o==="machine"||o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"||o==="analytics"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function Le(n){const s=document.createElement("div");return bs.replaceChildren(s),n(s)}function Jt(){if(Te){try{Te()}catch{}Te=null}const{screen:n,id:s}=vs();switch(ys(n),n){case"machine":if(!s){location.hash="#/targets";return}Te=Le(o=>Ia(o,s));break;case"setup":case"dash":case"logs":case"services":if(!s){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(s)}`;return;case"security":if(!s){location.hash="#/targets";return}Te=Le(o=>ts(o,s));break;case"diag":if(!s){location.hash="#/targets";return}Te=Le(o=>ya(o,s));break;case"analytics":if(!s){location.hash="#/rpc";return}Te=Le(o=>ba(o,s));break;case"rpc":Te=Le(o=>ls(o));break;case"settings":Te=Le(o=>ss(o));break;case"targets":Te=Le(o=>us(o));break;case"panel":Te=Le(o=>Ut(o));break;case"home":default:Te=Le(o=>Ut(o));break}}window.addEventListener("hashchange",Jt);Jt();
