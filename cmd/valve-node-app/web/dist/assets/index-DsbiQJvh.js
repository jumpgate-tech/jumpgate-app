var Wn=Object.defineProperty;var _n=(t,s,o)=>s in t?Wn(t,s,{enumerable:!0,configurable:!0,writable:!0,value:o}):t[s]=o;var Xe=(t,s,o)=>_n(t,typeof s!="symbol"?s+"":s,o);(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))e(d);new MutationObserver(d=>{for(const l of d)if(l.type==="childList")for(const v of l.addedNodes)v.tagName==="LINK"&&v.rel==="modulepreload"&&e(v)}).observe(document,{childList:!0,subtree:!0});function o(d){const l={};return d.integrity&&(l.integrity=d.integrity),d.referrerPolicy&&(l.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?l.credentials="include":d.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function e(d){if(d.ep)return;d.ep=!0;const l=o(d);fetch(d.href,l)}})();const qt="valve-theme",Wt=window.matchMedia("(prefers-color-scheme: dark)");function vt(){const t=localStorage.getItem(qt);return t==="light"||t==="dark"||t==="system"?t:"system"}function Kn(t=vt()){return t==="system"?Wt.matches?"dark":"light":t}function dt(){document.documentElement.dataset.theme=Kn()}function Vn(t){localStorage.setItem(qt,t),dt()}function Gn(){dt(),Wt.addEventListener("change",()=>{vt()==="system"&&dt()})}function _t(){return se("/api/host")}function He(){return se("/api/catalog")}function Me(){return se("/api/targets")}function ut(t){return se("/api/targets",{method:"POST",headers:Ie,body:JSON.stringify(t)})}function zn(t){return se(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Jn(t,s){return se(`/api/targets/${encodeURIComponent(t)}/disk?path=${encodeURIComponent(s)}`)}function Yn(t,s){return se(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:Ie,body:JSON.stringify(s)})}function Ze(t,s){const o=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function Zn(t,s){const o=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function Xn(t,s=200){return se(`/api/targets/${encodeURIComponent(t)}/logs?n=${s}`)}function Qn(t,s){const o=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return o.onmessage=e=>{try{s(JSON.parse(e.data))}catch{}},()=>o.close()}function Lt(t,s){const o=s===void 0?{}:{lines:s};return se(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:Ie,body:JSON.stringify(o)})}function ea(t,s,o){return se(`/api/targets/${encodeURIComponent(t)}/services/${s}/${o}`,{method:"POST"})}function ta(t,s){return se(`/api/targets/${encodeURIComponent(t)}/services/${s}/clear`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:s})})}function na(t){return se(`/api/targets/${encodeURIComponent(t)}/du`)}function aa(t){return se(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function sa(t){return se(`/api/targets/${encodeURIComponent(t)}/firewall`)}function oa(t){return se(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function ra(t){return se(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function Kt(t){return se(`/api/targets/${encodeURIComponent(t)}/containers`)}function ia(t,s,o){return se(`/api/targets/${encodeURIComponent(t)}/containers/${s}/${o}`,{method:"POST"})}async function ca(t,s){const o=await fetch(`/api/targets/${encodeURIComponent(t)}/containers/${s}/wipe`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:s})}),e=await o.text();let d=null;try{d=e?JSON.parse(e):null}catch{}if(d&&typeof d=="object"&&"report"in d)return d;const l=d&&typeof d=="object"&&typeof d.error=="string"?d.error:o.statusText||`HTTP ${o.status}`;throw new Le(o.status,l)}function la(t,s){return se(`/api/targets/${encodeURIComponent(t)}/containers/${s}/provision`,{method:"POST"})}async function da(t){const s=await fetch(`/api/targets/${encodeURIComponent(t)}/containers/devnet/reset`,{method:"POST",headers:Ie}),o=await s.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Le(s.status,d)}function ua(t,s,o){return se(`/api/targets/${encodeURIComponent(t)}/containers/${s}/config`,{method:"PUT",headers:Ie,body:JSON.stringify(o)})}function yt(){return se("/api/gateways")}async function pa(t){await se(`/api/orphans/${encodeURIComponent(t)}`,{method:"DELETE"})}function Vt(t){return se("/api/gateways",{method:"POST",headers:Ie,body:JSON.stringify(t)})}function Gt(t){return se(`/api/gateways/${encodeURIComponent(t)}/tls/verify`)}function ha(t){return se(`/api/gateways/${encodeURIComponent(t)}/traffic`)}function pt(t){return se(`/api/gateways/${encodeURIComponent(t)}/analytics`)}function zt(t,s=!1){const o=s?"?refresh=1":"";return se(`/api/gateways/${encodeURIComponent(t)}/capabilities${o}`)}function fa(t){return se(`/api/gateways/${encodeURIComponent(t)}`,{method:"DELETE"})}function Ae(t,s){return se(`/api/gateways/${encodeURIComponent(t)}/config`,{method:"PUT",headers:Ie,body:JSON.stringify(s)})}function Jt(t,s){return se(`/api/gateways/${encodeURIComponent(t)}/${s}`,{method:"POST"})}function ma(t){return se(`/api/gateways/${encodeURIComponent(t)}/trust-cert`,{method:"POST"})}function ht(t){return se(`/api/gateways/${encodeURIComponent(t)}/provision`,{method:"POST"})}async function Yt(t){const s=await fetch(`/api/gateways/${encodeURIComponent(t)}/wipe`,{method:"POST",headers:Ie,body:JSON.stringify({Confirm:t})}),o=await s.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const d=e&&typeof e=="object"&&typeof e.error=="string"?e.error:s.statusText||`HTTP ${s.status}`;throw new Le(s.status,d)}function ba(t){return se(`/api/chainlist/${t}`)}function ft(t,s){return se(`/api/gateways/${encodeURIComponent(t)}/knownset/${s}`)}function va(){return se("/api/settings")}function ya(t){return se("/api/settings",{method:"PUT",headers:Ie,body:JSON.stringify(t)})}class Le extends Error{constructor(o,e,d,l){super(e);Xe(this,"status");Xe(this,"hint");Xe(this,"code");this.name="ApiError",this.status=o,this.hint=d,this.code=l}}const Ie={"Content-Type":"application/json"};async function se(t,s){const o=await fetch(t,s);if(!o.ok){let d=o.statusText||`HTTP ${o.status}`,l,v;try{const p=await o.json();p&&typeof p.error=="string"&&p.error&&(d=p.error),p&&typeof p.hint=="string"&&p.hint&&(l=p.hint),p&&typeof p.code=="string"&&p.code&&(v=p.code)}catch{}throw new Le(o.status,d,l,v)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const Nt="https://learn.valve.city/rpc";function a(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function $e(t,s){const o=t&&s&&s!==Nt?` <span class="footer-sep">·</span> <a href="${a(s)}" target="_blank" rel="noopener noreferrer">${a(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(Nt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function ga(t){t.innerHTML=`
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
  `;const s=t.querySelector("#content"),o=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:s,setActiveNav:d=>{const l=d==="machine"?"targets":d==="home"||d==="panel"?"rpc":d;for(const v of o)v.classList.toggle("active",v.dataset.nav===l)}}}function ge(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function $a(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function wa(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const s=Math.round(t/60),o=Math.floor(s/60),e=s%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const d=Math.floor(o/24),l=o%24;return`~${d}d ${l}h`}function Y(t,s){return`<span class="badge badge-${s}">${a(t)}</span>`}function Be(t){return`<span class="dot dot-${t}"></span>`}const At=["B","KB","MB","GB","TB","PB"];function je(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let s=t,o=0;for(;s>=1024&&o<At.length-1;)s/=1024,o++;const e=s<10?2:s<100?1:0;return`${s.toFixed(e)} ${At[o]}`}async function ze(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Pe(t,s){t.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!t.contains(e))return;const d=e.dataset.action;d&&s(d,e,o)})}function mt(t,s,o){const e=s.find(l=>l.value===o),d=s.map(l=>`
      <li class="dropdown-option${l.value===o?" selected":""}" role="option"
          aria-selected="${l.value===o}" data-value="${a(l.value)}">
        ${a(l.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(t)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function Je(t){t.querySelectorAll(".dropdown.open").forEach(s=>{var o;s.classList.remove("open"),(o=s.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function gt(t,s){t.addEventListener("click",d=>{const l=d.target,v=l.closest(".dropdown-trigger");if(v&&t.contains(v)){const E=v.closest(".dropdown"),F=!!E&&!E.classList.contains("open");Je(t),E&&F&&(E.classList.add("open"),v.setAttribute("aria-expanded","true"));return}const p=l.closest(".dropdown-option");if(p&&t.contains(p)){const E=p.closest(".dropdown");Je(t),s((E==null?void 0:E.dataset.dropdown)??"",p.dataset.value??"");return}Je(t)});const o=d=>{if(!t.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const l=d.target;(!l.closest(".dropdown")||!t.contains(l))&&Je(t)},e=d=>{if(!t.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}d.key==="Escape"&&Je(t)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const at="app-modal";let nt=null;function fe(t,s){ne();const o=document.createElement("div");o.className="modal-overlay",o.id=at,o.innerHTML=`<div class="modal">${t}</div>`,o.addEventListener("click",d=>{const l=d.target.closest("[data-modal-action]");l!=null&&l.dataset.modalAction?s(l.dataset.modalAction):d.target===o&&s("cancel")});const e=d=>{d.key==="Escape"&&s("cancel")};document.addEventListener("keydown",e),nt=e,document.body.appendChild(o)}function ne(){var t;(t=document.getElementById(at))==null||t.remove(),nt&&(document.removeEventListener("keydown",nt),nt=null)}function Ve(){return document.querySelector(`#${at} .modal`)}function De(t){return new Promise(s=>{var d;let o=!1;const e=l=>{o||(o=!0,ne(),s(l))};fe(`
        <h2>${a(t.title)}</h2>
        <p>${a(t.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${t.danger?" btn-danger":""}" data-modal-action="confirm">${a(t.confirmLabel)}</button>
        </div>
      `,l=>e(l==="confirm")),(d=document.querySelector(`#${at} [data-modal-action="confirm"]`))==null||d.focus()})}const rt=5e3,ka=60;function Ca(t,s){let o=!1,e=null,d=null,l=null,v=null;const p=[];t.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${$e()}</div>`;const E=t.querySelector("#an-body");Pe(t,($,h)=>{var x;$==="toggle-endpoint"&&((x=h.closest(".an-endpoint"))==null||x.classList.toggle("expanded"))}),F();async function F(){try{e=((await yt()).gateways??[]).find(h=>h.id===s)??null}catch($){if(o)return;l=String($ instanceof Error?$.message:$),_();return}if(!o){if(!e){_();return}await V(),v=window.setInterval(()=>void V(),rt)}}async function V(){try{const $=await pt(s);if(o)return;A($),d=$,l=null}catch($){if(o)return;l=String($ instanceof Error?$.message:$)}_()}function A($){if(!$.enabled||$.error)return;const h=p[p.length-1];h&&h.since!==$.since&&(p.length=0);const x=new Map;for(const H of $.networks??[])x.set(H.chainId,H.received);p.push({t:Date.now(),since:$.since,received:x}),p.length>ka&&p.shift()}function _(){o||(E.innerHTML=G())}function G(){return l&&!d?`<h1>Analytics</h1><p class="error">${a(l)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
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
        <div class="an-head-right muted small">${I()}</div>
      </div>
    `}function I(){if(!d)return"";if(!d.enabled)return"counters off";if(d.error)return"could not be read";const $=d.since?new Date(d.since):null;return $&&!Number.isNaN($.getTime())?`totals since the gateway started, ${a($.toLocaleString())}<br />re-read every ${rt/1e3}s`:`re-read every ${rt/1e3}s`}function f($){return $.enabled?$.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a($.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:g($)+ve($):`
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
          ${U("Received",ge($.received),"what clients asked this chain for")}
          ${U("Answered",ge($.answered),"returned by one of your endpoints")}
          ${U("From cache",ge($.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${U("Failed",ge($.failed),"asked for and never answered",$.failed>0?"bad":"")}
        </div>
        ${ce($.chainId)}
        ${H?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':le("Method",h.map(W=>({label:W.method,l:W})))+le("Endpoint",x.map(W=>({label:W.upstream,l:W})))+J($)}
      </div>
    `}function U($,h,x,H=""){return`
      <div class="an-stat${H?" an-stat-"+H:""}" title="${a(x)}">
        <span class="an-stat-n">${a(h)}</span>
        <span class="an-stat-l">${a($)}</span>
      </div>
    `}function z($){const h=ee($.chainId);if(h===null)return'<span class="an-rate muted small">measuring rate…</span>';const x=Math.round((p[p.length-1].t-p[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${x}s apart.">
      ${a(h.toFixed(h<10?2:0))} req/s <span class="muted">over the last ${x}s</span>
    </span>`}function ee($){if(p.length<2)return null;const h=p[0],x=p[p.length-1],H=(x.t-h.t)/1e3;if(H<=0)return null;const W=(x.received.get($)??0)-(h.received.get($)??0);return W<0?null:W/H}function ce($){if(p.length<3)return"";const h=[];for(let w=1;w<p.length;w++){const D=p[w-1],X=p[w],u=(X.t-D.t)/1e3,y=(X.received.get($)??0)-(D.received.get($)??0);h.push(u>0&&y>=0?y/u:0)}const x=Math.max(...h);if(x<=0)return"";const H=240,W=28,Q=h.length>1?H/(h.length-1):H,b=h.map((w,D)=>`${(D*Q).toFixed(1)},${(W-w/x*W).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${x.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${H} ${W}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(x.toFixed(2))} req/s</span>
      </div>
    `}function J($){const h=[];return $.cached.count>0&&h.push(`${a(ge($.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${$.cached.mean===null?"":`, in ${a(Ye($.cached.mean))} on average`}.`),$.failedLatency.count>0&&$.failedLatency.mean!==null&&h.push(`The ${a(ge($.failedLatency.count))} that failed took
         ${a(Ye($.failedLatency.mean))} on average to fail.`),h.length===0?"":`<p class="muted small">${h.join(" ")}</p>`}function le($,h){return h.length===0?"":`
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
        <td class="an-num">${h.mean===null?'<span class="muted">—</span>':a(Ye(h.mean))}</td>
        <td>${j(h)}</td>
      </tr>
    `}function j($){const h=$.buckets??[];if(h.length===0||$.count===0)return'<span class="muted small">—</span>';let x=0;const H=[];for(const Q of h){const b=Q.count-x;x=Q.count,H.push({label:de(Q.le),n:Math.max(0,b)})}return H.reduce((Q,b)=>Q+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(H.filter(Q=>Q.n>0).map(Q=>`${Q.n} ${Q.label}`).join(" · "))}">
        ${H.map((Q,b)=>Q.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${Q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(pe(H))}</span>
    `}function pe($){for(let h=$.length-1;h>=0;h--)if($[h].n>0)return`slowest ${$[h].label}`;return""}function de($){if($==="+Inf")return"30s or more";const h=Number($);return Number.isFinite(h)?`under ${Ye(h)}`:`under ${$}`}function ve($){const h=$.endpoints??[];return`
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
    `}function we($){const h=[];return $.scored?(h.push($.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String($.position))}</span>`),h.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a($.score.toFixed(3))}</span>`),$.primarySwitches>1&&h.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ge($.primarySwitches)} switches</span>`),$.excludedSeconds>0&&h.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(Ye($.excludedSeconds))}</span>`),`<span class="an-selection">${h.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function Ce($,h){return`
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
    `}return()=>{o=!0,v!==null&&window.clearInterval(v)}}function Ye(t){return!Number.isFinite(t)||t<0?"—":t>0&&t<5e-4?"<1ms":t<1?`${Math.round(t*1e3)}ms`:t<60?`${t<10?t.toFixed(1):Math.round(t)}s`:`${Math.round(t/60)}m`}function Sa(t,s){let o=!1,e=null,d=null,l=!1,v=!1;t.innerHTML=`<h1>Network diagnostics: ${a(s)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${$e()}</div>`;const p=t.querySelector("#diag-body"),E=t.querySelector("#diag-footer");Pe(t,(f,g)=>{var S;if(f==="run")V();else if(f==="toggle")(S=g.closest(".check-item"))==null||S.classList.toggle("expanded");else if(f==="copy"){const U=g.dataset.copy;U&&I(g,U)}}),F();async function F(){let f,g;try{const[U,z]=await Promise.all([Me(),He()]);f=U.find(ee=>ee.id===s),g=z}catch(U){if(o)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(U))}</p>`;return}if(o)return;if(!f){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const S=g==null?void 0:g.networks.find(U=>U.ChainID===f.wire.ChainID);S&&(E.innerHTML=$e(S.Name,S.LearnURL));try{e=await ra(s),v=!0}catch(U){d=String(U instanceof Error?U.message:U)}o||A()}async function V(){l=!0,d=null,A();try{e=await oa(s),v=!0}catch(f){d=String(f instanceof Error?f.message:f)}l=!1,o||A()}function A(){p.innerHTML=`
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
    `}function _(){if(!v&&!d)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const f=new Date(e.at).toLocaleString(),g=e.failedId?`<p><strong>Failed at: ${a(G(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
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
    `}async function I(f,g){const S=await ze(g),U=f.textContent;f.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{o||(f.textContent=U)},1500)}return()=>{o=!0}}const xa=85,it={exec:"Execution",beacon:"Beacon"};function Ta(t,s){let o=!1,e=null,d=null,l=null,v=null,p=null,E=null,F=null,V=null;const A={exec:null,beacon:null};let _=null;t.innerHTML=`<h1>Dashboard: ${a(s)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${$e()}</div>`;const G=t.querySelector("#dash-body"),B=t.querySelector("#dash-footer");G.addEventListener("click",h=>{const x=h.target.closest("[data-action]");if(!x||!G.contains(x))return;const H=x.dataset.action;if(H==="svc-action"){const W=x.dataset.svc,Q=x.dataset.kind;W&&Q&&oe(W,Q)}else if(H==="open-clear"){const W=x.dataset.svc;W&&Ce(W)}else if(H==="copy"){const W=x.dataset.copy;W&&we(x,W)}else H==="retry-du"?f():H==="retry-endpoints"&&g()}),I();async function I(){let h,x;try{const[W,Q]=await Promise.all([Me(),He()]);h=W.find(b=>b.id===s),x=Q}catch(W){if(o)return;G.innerHTML=`<p class="error">Failed to load target: ${a(String(W))}</p>`;return}if(o)return;if(!h){G.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){G.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const H=x==null?void 0:x.networks.find(W=>W.ChainID===h.wire.ChainID);H&&(B.innerHTML=$e(H.Name,H.LearnURL)),G.innerHTML='<p class="muted">Connecting…</p>',e=Zn(s,W=>{o||(S(W),d=W,l=W,U())}),f(),g()}async function f(){E=null;try{p=await na(s)}catch(h){p=null,E=String(h instanceof Error?h.message:h)}o||U()}async function g(){V=null;try{F=await aa(s)}catch(h){F=null,V=String(h instanceof Error?h.message:h)}o||U()}function S(h){if(!d)return;const x=(new Date(h.at).getTime()-new Date(d.at).getTime())/1e3,H=h.execHead-d.execHead;if(x>0&&H>=0){const W=H/x;v=v===null?W:v*.7+W*.3}}function U(){if(!l)return;const h=l;G.innerHTML=`
      <p class="dash-status">${z(h)}</p>
      <div class="card-grid">
        ${de(h)}
        ${ce(h)}
        ${J(h)}
        ${le(h)}
        ${Z(h)}
        ${j()}
      </div>
      <p class="muted small">Last updated ${a(new Date(h.at).toLocaleTimeString())}</p>
    `}function z(h){return!h.execActive&&!h.beaconActive?Y("Node not running","bad"):h.execSyncing||h.beaconDistance>0?Y("Syncing","warn"):Y("Running · synced","ok")}function ee(h){const H=h.refHead>0?h.refHead-h.execHead:null,W=H!==null&&H>0&&v&&v>0?wa(H/v):H!==null&&H<=0?"caught up":"—";return{lag:H,eta:W}}function ce(h){const{lag:x,eta:H}=ee(h);return`
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
    `}function Z(h){const x=h.diskUsedPct>=xa,H=`
      <div class="meter"><div class="meter-fill ${x?"meter-warn":""}" style="width:${Math.min(h.diskUsedPct,100)}%"></div></div>
      <p>${$a(h.diskUsedPct)} used</p>
    `;if(E)return`
        <div class="card ${x?"card-warn":""}">
          <h3>Storage</h3>
          ${H}
          <p class="error small">${a(E)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!p)return`
        <div class="card ${x?"card-warn":""}">
          <h3>Storage</h3>
          ${H}
          <p class="muted">Loading…</p>
        </div>
      `;const W=p.ExpectedExecBytes>0?Math.min(p.ExecBytes/p.ExpectedExecBytes*100,100):0,Q=p.ExpectedBeaconBytes>0?Math.min(p.BeaconBytes/p.ExpectedBeaconBytes*100,100):0,{lag:b,eta:w}=ee(h),D=b!==null&&b>0&&v!==null&&v>0;return`
      <div class="card ${x?"card-warn":""}">
        <h3>Storage</h3>
        ${H}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${je(p.ExecBytes)} of ~${je(p.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${W}%"></div></div>
        ${D?`<p class="muted small">Estimated time remaining: ${a(w)}</p>`:""}
        <p class="muted small">Beacon — ${je(p.BeaconBytes)} of ~${je(p.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${Q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${je(p.DiskFreeBytes)}</dd></div>
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
    `}function pe(h,x){const H=it[h],W=A[h],Q=(b,w,D)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${h}" data-kind="${b}" ${W!==null||D?"disabled":""}>${W===b?ve():a(w)}</button>`;return`
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
    `}function ve(){return'<span class="spinner" aria-label="working"></span>'}async function oe(h,x){if(A[h]===null){A[h]=x,_=null,U();try{await ea(s,h,x)}catch(H){_=`${it[h]} ${x} failed: ${H instanceof Error?H.message:String(H)}`}A[h]=null,o||U()}}async function we(h,x){const H=await ze(x),W=h.textContent;h.textContent=H?"Copied!":"Copy failed",setTimeout(()=>{o||(h.textContent=W)},1500)}function Ce(h){const x=it[h],H=p?je(h==="exec"?p.ExecBytes:p.BeaconBytes):"unknown (disk usage hasn't loaded)";fe(`
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
      `,b=>{if(b==="cancel"){ne();return}b==="confirm"&&$(h)});const W=document.getElementById("clear-confirm-input"),Q=document.getElementById("clear-confirm-btn");W==null||W.addEventListener("input",()=>{Q&&(Q.disabled=W.value.trim()!==h)}),W==null||W.focus()}async function $(h){const x=document.getElementById("clear-confirm-btn");x&&(x.disabled=!0,x.textContent="Clearing…");try{await ta(s,h),ne(),f()}catch(H){const W=Ve();if(W){const Q=document.createElement("p");Q.className="error small",Q.textContent=`Clear failed: ${H instanceof Error?H.message:String(H)}`,W.appendChild(Q)}x&&(x.disabled=!1,x.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),ne()}}const Bt=500,Dt="valve-node-app.explain-consent";function Ea(t,s){let o=!1,e=null;const d=[];t.innerHTML=`
    <h1>Logs: ${a(s)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${$e()}</div>
  `;const l=t.querySelector("#logs-body"),v=t.querySelector("#logs-footer");Pe(t,I=>{I==="explain"&&V()}),p();async function p(){let I,f;try{const[S,U]=await Promise.all([Me(),He()]);I=S.find(z=>z.id===s),f=U}catch(S){if(o)return;l.innerHTML=`<p class="error">Failed to load target: ${a(String(S))}</p>`;return}if(o)return;if(!I){l.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!I.wire){l.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const g=f==null?void 0:f.networks.find(S=>S.ChainID===I.wire.ChainID);g&&(v.innerHTML=$e(g.Name,g.LearnURL));try{const S=await Xn(s,200);if(o)return;d.push(...S)}catch(S){if(o)return;l.innerHTML=`<p class="error">Failed to load logs: ${a(String(S))}</p>`;return}E(),e=Qn(s,S=>{o||(d.push(S),d.length>Bt&&d.splice(0,d.length-Bt),E())})}function E(){const I=d.filter(g=>g.severity==="error"||g.severity==="critical");l.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(F).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${Y(String(I.length),I.length?"bad":"neutral")}</h2>
          <div class="log-lines">${I.length?I.slice().reverse().map(F).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const f=l.querySelector(".log-lines");f&&(f.scrollTop=f.scrollHeight)}function F(I){const f=I.severity||"info",g=I.learnUrl?` <a href="${a(I.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(f)}">
        <span class="log-time">${a(new Date(I.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(I.unit)}</span>
        <span class="log-sev">${a(f)}</span>
        <span class="log-text">${a(I.line)}</span>
        ${I.explain?`<div class="log-explain">${a(I.explain)}${g}</div>`:""}
      </div>
    `}async function V(){const I=d.filter(g=>g.severity==="error"||g.severity==="critical").map(g=>g.line).slice(-40);if(!(localStorage.getItem(Dt)==="1")){A(I);return}await _(I)}function A(I){const f=I.length?`<pre class="explain-excerpt">${I.map(g=>a(g)).join(`
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
    `,g=>{g==="proceed"?(localStorage.setItem(Dt,"1"),B(),_(I)):B()})}async function _(I){G('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const f=I.length?await Lt(s,I):await Lt(s);if(o)return;G(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(f.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${f.sentExcerpt.map(g=>a(g)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&B()})}catch(f){if(o)return;if(f instanceof Le&&f.status===409){G(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,g=>{g==="close"&&B()});return}G(`
        <h2>Explain failed</h2>
        <p class="error">${a(f instanceof Error?f.message:String(f))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&B()})}}function G(I,f){B();const g=document.createElement("div");g.className="modal-overlay",g.id="explain-modal",g.innerHTML=`<div class="modal">${I}</div>`,g.addEventListener("click",S=>{const U=S.target.closest("[data-modal-action]");U!=null&&U.dataset.modalAction&&f(U.dataset.modalAction),S.target===g&&f("cancel")}),document.body.appendChild(g)}function B(){var I;(I=document.getElementById("explain-modal"))==null||I.remove()}return()=>{o=!0,e==null||e(),B()}}const Ia="run",Pa={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Ra={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function La(t,s){let o=!1,e=null,d=null;const l={devnet:null},v={devnet:null},p={devnet:[]};let E=null;const F={devnet:!1};let V=null;const A={devnet:null},_={devnet:null};t.innerHTML=`
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
  `;const G=t.querySelector("#services-body");Pe(t,(u,y)=>{Ce(u,y)}),B();async function B(){try{const u=await Kt(s);if(o)return;e=u,d=null}catch(u){if(o)return;e=null,d=D(u)}f()}function I(u){return e==null?void 0:e.services.find(y=>y.id===u)}function f(){if(!o){if(d){G.innerHTML=`<p class="error">Could not read this machine's services: ${a(d)}</p>`;return}if(!e){G.innerHTML='<p class="muted">Loading…</p>';return}G.innerHTML=`
      ${g(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(S).join("")}
      </div>
    `}}function g(u){if(u.present&&u.reachable&&!u.hint)return`<p class="muted small">Docker: ${a(u.flavor)}${u.serverVersion?` ${a(u.serverVersion)}`:""} · reachable</p>`;const y=u.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(y)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${u.detail?`<div class="small">${a(u.detail)}</div>`:""}
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function S(u){const y=u.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(u.label)}</h2>
          ${U(u)}
        </div>
        <p class="muted small">${a(Pa[u.id]??"")}</p>

        ${u.error?z(u):""}
        ${u.blocked?`<div class="banner banner-warn">${a(u.blocked)}</div>`:""}
        ${y.map(O=>`<div class="banner banner-warn">${a(O)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(u.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${u.status.Image?`<code>${a(u.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${ee(u)}

        ${ce(u)}

        <div class="card-actions">
          ${(u.actions??[]).map(O=>J(u,O)).join("")}
        </div>
        ${v[u.id]?`<p class="error small">${a(v[u.id])}</p>`:""}
        ${le(u)}

        ${Z(u)}
      </div>
    `}function U(u){switch(u.status.State){case"running":return Y("running","ok");case"created-but-stopped":return Y("stopped","warn");case"not-created":return Y("not created","neutral");default:return Y("unknown","bad")}}function z(u){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(u.error??"")}</div>
        ${u.hint?`<div class="small">${a(u.hint)}</div>`:""}
      </div>
    `}function ee(u){if(u.status.State!=="created-but-stopped"||u.status.ExitCode===0)return"";const y=u.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${u.status.ExitCode}${y}.</p>`}function ce(u){const y=u.endpoints??[];return y.length===0?u.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":y.map(O=>`
        <div class="endpoint-row">
          ${Be("ok")}
          <span class="muted small">${a(O.label)}</span>
          <code class="endpoint-url">${a(O.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(O.url)}">Copy</button>
        </div>`).join("")}function J(u,y){const O=Ra[y];if(!O)return"";const te=l[u.id],ie=y==="create"?`Create ${u.id==="devnet"?"devnet":"gateway"}`:O.label;return`
      <button class="${O.className}" data-action="svc-${y}" data-svc="${a(u.id)}"
              title="${a(O.title)}" ${te?"disabled":""}>
        ${te===y?'<span class="spinner" aria-label="working"></span>':a(ie)}
      </button>
    `}function le(u){const y=p[u.id]??[];return y.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(y.join(`
`))}</pre>
      </div>
    `}function Z(u){const y=F[u.id],O=j(u);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${u.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(u.id)}">
            ${y?"Close":"Edit"}
          </button>
        </div>
        ${y?pe():`<p class="small">${O}</p>`}
        ${A[u.id]?`<p class="error small">${a(A[u.id])}</p>`:""}
        ${_[u.id]?`<p class="muted small">${a(_[u.id])}</p>`:""}
      </div>
    `}function j(u){const y=u.devnet;return y?`Chain ${y.ChainID} · a block every ${a(y.BlockTime)} · JSON-RPC on ${a(y.BindAddr)}:${y.HTTPPort} · WebSocket on ${a(y.BindAddr)}:${y.WSPort}`:"—"}function pe(u){return de()}function de(){const u=V;return u?`
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
    `:""}function ve(){F.devnet&&V&&(V.BlockTime=oe("#dev-blocktime",V.BlockTime),V.HTTPPort=we("#dev-http",V.HTTPPort),V.WSPort=we("#dev-ws",V.WSPort),V.BindAddr=oe("#dev-bind",V.BindAddr))}function oe(u,y){const O=t.querySelector(u);return O?O.value.trim():y}function we(u,y){const O=t.querySelector(u);if(!O)return y;const te=Number.parseInt(O.value.trim(),10);return Number.isFinite(te)?te:y}async function Ce(u,y){const O=y.dataset.svc??"";switch(u){case"refresh":await B();return;case"copy":y.dataset.copy&&await w(y,y.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await $(O,u.slice(4));return;case"svc-create":case"svc-recreate":await h(O);return;case"svc-wipe":W(O);return;case"toggle-config":x(O);return;case"save-config":await H(O);return;default:return}}async function $(u,y){if(!l[u]){l[u]=y,v[u]=null,f();try{await ia(s,u,y)}catch(O){v[u]=`${y} failed: ${D(O)}${X(O)}`}l[u]=null,await B()}}async function h(u){if(!l[u]){l[u]="create",v[u]=null,p[u]=["starting…"],f();try{await la(s,u)}catch(y){v[u]=`${D(y)}${X(y)}`,p[u]=[],l[u]=null,f();return}E==null||E(),E=Ze(s,y=>{if(o)return;const O=y.err?`${y.stepId}: ${y.err}`:y.line?`${y.stepId}: ${y.line}`:`${y.stepId}: done`;if(p[u]=[...(p[u]??[]).filter(ie=>ie!=="starting…"),O],!!y.err||y.stepId===Ia&&!!y.done){E==null||E(),E=null,l[u]=null,y.err&&(v[u]="Provisioning failed — see the log below."),B();return}f()})}}function x(u){if(ve(),F[u]=!F[u],A[u]=null,_[u]=null,F[u]){const y=I(u);y!=null&&y.devnet&&(V={...y.devnet})}f()}async function H(u){var te;ve(),A[u]=null,_[u]=null;const y=V;if(!y)return;if(y.HTTPPort===y.WSPort){A[u]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",f();return}try{await ua(s,u,y)}catch(ie){A[u]=D(ie),f();return}const O=((te=I(u))==null?void 0:te.status.State)==="running";F[u]=!1,_[u]=O?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await B()}function W(u){const y=I(u);if(!y)return;const O=(y.restartsOnWipe??[]).map(K=>{var be;return((be=I(K))==null?void 0:be.label)??K});fe(`
        <h2>Wipe ${a(y.label)}</h2>
        <p class="error">This deletes ${a(y.wipeDiscards)}</p>
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
      `,K=>{if(K==="cancel"||K==="close"){ne(),B();return}K==="confirm"&&Q(u)});const te=document.getElementById("wipe-confirm-input"),ie=document.getElementById("wipe-confirm-btn");te==null||te.addEventListener("input",()=>{ie&&(ie.disabled=te.value.trim()!==u)}),te==null||te.focus()}async function Q(u){const y=document.getElementById("wipe-confirm-btn");y&&(y.disabled=!0,y.textContent="Wiping…");let O;try{O=await ca(s,u)}catch(te){const ie=Ve();if(ie){const K=document.createElement("p");K.className="error small",K.textContent=`Wipe failed: ${D(te)}${X(te)}`,ie.appendChild(K)}y&&(y.disabled=!1,y.textContent=`Wipe ${u}`);return}b(u,O)}function b(u,y){const O=I(u),te=he=>{var Ne;return((Ne=I(he))==null?void 0:Ne.label)??he},ie=[];ie.push(y.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const he of y.report.VolumesRemoved??[])ie.push(`Volume ${he} deleted.`);for(const he of y.report.VolumesAbsent??[])ie.push(`Volume ${he} was already gone.`);y.report.Recreated&&ie.push("Container re-created from your saved configuration.");const K=(y.report.Cascaded??[]).map(te),be=(y.report.CascadeSkipped??[]).map(te);fe(`
        <h2>${a((O==null?void 0:O.label)??u)} wiped</h2>
        <ul class="plain-list">${ie.map(he=>`<li>${a(he)}</li>`).join("")}</ul>
        ${K.length?`<p class="ok">Restarted in front of it: ${a(K.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${be.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(be.join(", "))}.</p>`:""}
        ${y.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(y.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,he=>{(he==="close"||he==="cancel")&&(ne(),B())})}async function w(u,y){const O=await ze(y),te=u.textContent;u.textContent=O?"Copied!":"Copy failed",setTimeout(()=>{o||(u.textContent=te)},1500)}function D(u){return u instanceof Error?u.message:String(u)}function X(u){return u instanceof Le&&u.hint?` — ${u.hint}`:""}return()=>{o=!0,E==null||E(),ne()}}const ct=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Qe=8545,et=5052,tt=30303,Na=[369,943,1],Ht={369:"default",943:"practise here first"};function Aa(t,s){let o=!1;const e={targetId:s,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${a(s)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${$e()}</div>`;const d=t.querySelector("#wizard-body"),l=t.querySelector("#wizard-footer");Pe(t,(b,w)=>{we(b,w)}),gt(t,(b,w)=>{b==="exec-select"?e.execId=w:b==="beacon-select"&&(e.beaconId=w),p()}),t.addEventListener("change",b=>{const w=b.target;w instanceof HTMLInputElement&&(w.id==="data-dir-input"?(Ce(),J()):w.id==="checkpoint-toggle"?(e.checkpoint=w.checked,p()):w.id==="exec-snapshot-toggle"&&(e.execSnapshot=w.checked,p()))}),v();async function v(){try{const[b,w]=await Promise.all([He(),Me()]);if(o)return;e.catalog=b;const D=w.find(X=>X.id===s);D!=null&&D.wire&&(e.chainId=D.wire.ChainID,e.execId=D.wire.ExecID,e.beaconId=D.wire.BeaconID,e.archive=D.wire.Archive,D.wire.ExecHTTPPort&&(e.execHTTPPort=String(D.wire.ExecHTTPPort)),D.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(D.wire.BeaconHTTPPort)),D.wire.ExecP2PPort&&(e.execP2PPort=String(D.wire.ExecP2PPort)),D.wire.RPCBindAddr&&(e.rpcBindAddr=D.wire.RPCBindAddr)),p()}catch(b){if(o)return;e.loadError=String(b instanceof Error?b.message:b),p()}}function p(){if(e.loadError){d.innerHTML=`<p class="error">Failed to load: ${a(e.loadError)}</p>`;return}e.catalog&&(d.innerHTML=`
      ${Q(e.step)}
      ${F()}
    `,E())}function E(){var w;const b=(w=e.catalog)==null?void 0:w.networks.find(D=>D.ChainID===e.chainId);l.innerHTML=b?$e(b.Name,b.LearnURL):$e()}function F(){switch(e.step){case"network":return V();case"clients":return A();case"mode":return de();case"review":return ve();case"run":return oe()}}function V(){const b=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Na.map(D=>{const X=b.networks.find(O=>O.ChainID===D);if(!X)return"";const u=e.chainId===D,y=Ht[D]?Y(Ht[D],D===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${u?"selected":""}" data-action="pick-network" data-chain-id="${D}" type="button">
          <h3>${a(X.Name)} <span class="muted">(chain ${D})</span></h3>
          ${y}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function A(){const b=e.catalog,w=b.networks.find(u=>u.ChainID===e.chainId);if(!w)return'<p class="error">Unknown network.</p>';(e.execId===null||!w.ExecClients.includes(e.execId))&&(e.execId=w.ExecClients[0]??null),(e.beaconId===null||!w.BeaconClients.includes(e.beaconId))&&(e.beaconId=w.BeaconClients[0]??null);const D=w.ExecClients.map(u=>Z(u,b)),X=w.BeaconClients.map(u=>Z(u,b));return`
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
          ${mt("exec-select",D,e.execId)}
        </label>
        ${pe(e.execId,b)}
        <label>
          Beacon client
          ${mt("beacon-select",X,e.beaconId)}
        </label>
        ${pe(e.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function _(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const G=1.1,B=.5,I="Valve reth snapshot",f="rough estimate";function g(b){return b.SnapshotSizeTB}function S(b){return b.SnapshotSizeTB*B}function U(b){return`<p class="muted small">${_(g(b))} is the measured size of Valve's reth snapshot for ${a(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function z(b){return{archive:g(b)*1e12*G,full:S(b)*1e12*G}}function ee(b,w){if(!b)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${a(w)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${a(w)}</code>: ${a(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==w)return"";const D=z(b),X=e.freeBytes>=D.archive,u=e.freeBytes>=D.full,y=`<p class="muted small">Free at <code>${a(w)}</code>: <strong>${je(e.freeBytes)}</strong> — archive ${X?"fits":"won't fit"} (${_(g(b))}, ${I}), full ${u?"fits":"won't fit"} (${_(S(b))}, ${f}).</p>`;let O="";return e.downgradeNote?O=`<p class="banner banner-warn">${a(e.downgradeNote)}</p>`:u||(O=`<p class="banner banner-warn">Neither full (${_(S(b))}, ${f}) nor archive (${_(g(b))}, ${I}) fits the free space here — choose a location with more room.</p>`),y+O}function ce(b,w){if(e.downgradeNote=null,!b||e.freeBytes===null)return;const D=z(b);e.archive&&e.freeBytes<D.archive&&e.freeBytes>=D.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${w} for archive (${_(g(b))}, ${I}) — switched to Full (${_(S(b))}, ${f}). Pick a location with more room to run archive.`)}async function J(){var D;if(e.chainId===null)return;const b=(D=e.catalog)==null?void 0:D.networks.find(X=>X.ChainID===e.chainId),w=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,p();try{const{freeBytes:X}=await Jn(e.targetId,w);if(o)return;e.freeBytes=X,e.probedPath=w,ce(b,w)}catch(X){if(o)return;e.freeBytes=null,e.probedPath=w,e.diskError=String(X instanceof Error?X.message:X)}e.diskProbing=!1,p()}function le(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function Z(b,w){const D=w.clients.find(X=>X.id===b);return{value:b,label:D?`${D.id} — ${j(D.repo)}`:b}}function j(b){const w=b.split("/");return w.length>=4?w[3]:b}function pe(b,w){const D=b?w.clients.find(u=>u.id===b):void 0;if(!D)return"";const X=D.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(D.repo)}" target="_blank" rel="noopener noreferrer">${a(X)}</a></p>`}function de(){var te,ie,K;const b=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",w=(te=e.catalog)==null?void 0:te.networks.find(be=>be.ChainID===e.chainId),D=((K=(ie=e.catalog)==null?void 0:ie.clients.find(be=>be.id===e.execId))==null?void 0:K.snapshotSupported)??!1,X=w?`${_(S(w))} (${f})`:"Smaller",u=w?`${_(g(w))} (${I})`:"Much larger",y=w?` on ${a(w.Name)}`:"",O=w?e.checkpoint?w.SyncLabel:w.GenesisSyncLabel:"";return`
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
          ${w?`<p class="sync-estimate">⏱ Estimated initial sync${y}: <strong>${a(O)}</strong></p>
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
              <tr><th>Approx. disk footprint${y}</th><td class="yes">${X}</td><td class="limited">${u}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${w?U(w):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
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
            Execution HTTP port <span class="muted">(default: ${Qe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Qe}" value="${a(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${a(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${et})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${et}" value="${a(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${a(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${tt})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${tt}" value="${a(e.execP2PPort)}" />
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
    `}function ve(){const w=e.catalog.networks.find(he=>he.ChainID===e.chainId),D=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,X=e.jwtPath||`${D}/jwt.hex`,u=ct.map(he=>`<li>${a(he.title)}</li>`).join(""),y=H(e.execHTTPPort,Qe),O=H(e.beaconHTTPPort,et),te=H(e.execP2PPort,tt),ie=y||O||te?`<tr><th>Non-default ports</th><td>${[y?`exec HTTP ${y}`:null,O?`beacon HTTP ${O}`:null,te?`exec p2p ${te}`:null].filter(he=>he!==null).map(a).join(", ")}</td></tr>`:"",{addr:K}=$(e.rpcBindAddr),be=K?`<tr><th>RPC bind address</th><td><code>${a(K)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
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
            ${ie}
            ${be}
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
    `}function oe(){const w=e.catalog.networks.find(K=>K.ChainID===e.chainId),D=w==null?void 0:w.LearnURL,X=new Set(e.events.filter(K=>K.done).map(K=>K.stepId)),u=new Set(e.events.filter(K=>K.err).map(K=>K.stepId)),y=new Map;for(const K of e.events){if(!K.line)continue;const be=y.get(K.stepId)??[];be.push(K.line),y.set(K.stepId,be)}const O=ct.map(K=>{var T;const be=X.has(K.id),he=u.has(K.id),Ne=he?Y("failed","bad"):be?Y("done","ok"):Y("pending","neutral"),Ue=(y.get(K.id)??[]).slice(-5),L=(T=e.events.find(q=>q.stepId===K.id&&q.err))==null?void 0:T.err,M=K.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${D?` <a href="${a(D)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${be?"step-done":""} ${he?"step-error":""}">
          <div class="step-head">${Ne} <strong>${a(K.title)}</strong></div>
          ${M}
          ${Ue.length?`<pre class="step-log">${Ue.map(q=>a(q)).join(`
`)}</pre>`:""}
          ${L?`<p class="error small">${a(L)}</p>`:""}
        </li>
      `}).join(""),te=e.events.some(K=>K.err),ie=ct.every(K=>X.has(K.id))||e.events.some(K=>K.stepId==="handshake"&&K.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${O}</ol>
        ${ie&&!te?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${a(e.startError)}</p>`:""}
        ${te?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function we(b,w){switch(b){case"pick-network":e.chainId=Number(w.dataset.chainId),e.execId=null,e.beaconId=null,p();break;case"goto-network":e.step="network",p();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",p();break;case"goto-mode":e.step="mode",p(),J();break;case"goto-review":if(Ce(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){p();break}e.step="review",p();break;case"start-setup":W();break}}function Ce(){const b=t.querySelectorAll('input[name="mode"]');for(const K of Array.from(b))K.checked&&(e.archive=K.value==="archive");const w=t.querySelector("#data-dir-input"),D=t.querySelector("#jwt-path-input");w&&(e.dataDir=w.value.trim()),D&&(e.jwtPath=D.value.trim());const X=t.querySelector("#exec-http-port-input"),u=t.querySelector("#beacon-http-port-input"),y=t.querySelector("#exec-p2p-port-input");X&&(e.execHTTPPort=X.value.trim()),u&&(e.beaconHTTPPort=u.value.trim()),y&&(e.execP2PPort=y.value.trim());const O=t.querySelector("#rpc-bind-addr-input");O&&(e.rpcBindAddr=O.value.trim());const te=t.querySelector("#checkpoint-url-input");te&&(e.checkpointUrl=te.value.trim());const ie=t.querySelector("#snapshot-key-input");ie&&(e.snapshotKey=ie.value.trim()),e.execHTTPPortError=x(e.execHTTPPort).error??null,e.beaconHTTPPortError=x(e.beaconHTTPPort).error??null,e.execP2PPortError=x(e.execP2PPort).error??null,e.rpcBindAddrError=$(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?le(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function $(b){if(!b)return{};const w=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return w?w.slice(1).every(D=>Number(D)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const h=/^\d+$/;function x(b){if(!b)return{};if(!h.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const w=Number(b);return!Number.isInteger(w)||w<1||w>65535?{error:"Port must be between 1 and 65535."}:{port:w}}function H(b,w){const{port:D}=x(b);if(!(D===void 0||D===w))return D}async function W(){var y;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(y=e.streamStop)==null||y.call(e),e.streamStop=null,p();const b={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(b.DataDir=e.dataDir),e.jwtPath&&(b.JWTPath=e.jwtPath);const w=H(e.execHTTPPort,Qe),D=H(e.beaconHTTPPort,et),X=H(e.execP2PPort,tt);w!==void 0&&(b.ExecHTTPPort=w),D!==void 0&&(b.BeaconHTTPPort=D),X!==void 0&&(b.ExecP2PPort=X);const{addr:u}=$(e.rpcBindAddr);u!==void 0&&(b.RPCBindAddr=u),e.checkpoint?e.checkpointUrl&&(b.CheckpointURL=e.checkpointUrl):b.NoCheckpoint=!0,e.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=e.snapshotKey);try{await Yn(e.targetId,b)}catch(O){if(!(O instanceof Le&&O.status===409)){e.starting=!1,e.startError=String(O instanceof Error?O.message:O),p();return}}e.starting=!1,e.step="run",p(),e.streamStop=Ze(e.targetId,O=>{o||(e.events.push(O),e.step==="run"&&p())})}function Q(b){const w=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],X=w.map(u=>u.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${w.map((u,y)=>`<li class="${y===X?"current":y<X?"past":"future"}">${a(u.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;o=!0,(b=e.streamStop)==null||b.call(e)}}function Ba(t,s){let o=!1;const e=new Map;t.innerHTML=`<h1>${a(s)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const d=t.querySelector("#machine-body");Pe(t,(A,_)=>{A==="toggle-section"&&F(_.dataset.section??"")}),l();async function l(){let A,_;try{const[G,B]=await Promise.all([Me(),He()]);A=G.find(I=>I.id===s),_=B}catch(G){if(o)return;d.innerHTML=`<p class="error">Failed to load machine: ${a(String(G))}</p>`;return}if(!o){if(!A){location.hash="#/targets";return}v(A,_)}}function v(A,_){const G=A.mode==="local"?"this machine":"SSH",B=A.mode==="ssh"&&A.ssh?`${a(A.ssh.User)}@${a(A.ssh.Host)}`:G;d.innerHTML=`
      <p class="muted">${B}</p>
      <p>${p(A,_)}</p>
      <div class="machine-sections">
        ${V.map(I=>E(I,A,_)).join("")}
      </div>
      ${$e()}
    `}function p(A,_){const G=A.wire;if(!G)return Y("not set up","neutral");const B=_.networks.find(f=>f.ChainID===G.ChainID),I=B?B.Name:`chain ${G.ChainID}`;return`${Y(I,"ok")} ${Y(G.ExecID,"neutral")} ${Y(G.BeaconID,"neutral")}${G.Archive?" "+Y("archive","warn"):""}`}function E(A,_,G){return`
      <section class="card machine-section" data-section-card="${a(A.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(A.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(A.title)}</span>
          <span class="machine-section-status">${A.status(_,G)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(A.key)}" hidden></div>
      </section>
    `}function F(A){const _=V.find(g=>g.key===A);if(!_)return;const G=t.querySelector(`[data-section-card="${A}"]`),B=t.querySelector(`[data-section-body="${A}"]`),I=t.querySelector(`.machine-section-head[data-section="${A}"]`);if(!G||!B||!I)return;const f=B.hidden;if(f&&!e.has(A)){const g=document.createElement("div");B.appendChild(g),e.set(A,_.mount(g))}B.hidden=!f,G.classList.toggle("open",f),I.setAttribute("aria-expanded",String(f))}const V=[{key:"setup",title:"Setup",status:A=>A.wire?Y("set up","ok"):Y("not set up","neutral"),mount:A=>Aa(A,s)},{key:"dashboard",title:"Dashboard",status:A=>A.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:A=>Ta(A,s)},{key:"logs",title:"Logs",status:A=>A.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:A=>Ea(A,s)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:A=>La(A,s)}];return()=>{o=!0;for(const A of e.values())try{A()}catch{}e.clear()}}function Da(t){let s;try{s=new URL(t).hostname}catch{return"endpoint"}if(!s)return"endpoint";if(s==="localhost"||/^[0-9.]+$/.test(s)||/^\[.*\]$/.test(s))return s;const o=s.split(".").filter(Boolean);return o.length<=1?s:o[o.length-2]}function Zt(t){var e;if(!t)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const s=t.actions??[];if(t.blocked)return{tone:"blocked",label:"Unavailable",sub:t.blocked,actions:s,blocked:t.blocked};const o=((e=t.networks)==null?void 0:e.length)??0;return t.status.State==="running"?{tone:"on",label:"Running",sub:`${o} network${o===1?"":"s"} served`,actions:s}:{tone:"off",label:"Stopped",sub:o?`${o} network${o===1?"":"s"} configured`:"Press to start",actions:s}}function Ge(t){if(!t.running)return"off";if(!t.serviceable)return"frequent";const s=t.slowRate??0;return s>.4?"frequent":s>=.1?"occasional":"stable"}const Ha="0.5";function Xt(t){if(!t||t.count<=0||!t.buckets||t.buckets.length===0)return;const s=t.buckets.find(e=>e.le===Ha);if(!s)return;const o=t.count-s.count;return Math.max(0,Math.min(1,o/t.count))}function Ma(t){if(!t||t.length===0)return null;let s=0;const o=new Map;for(const e of t){s+=e.count;for(const d of e.buckets??[])o.set(d.le,(o.get(d.le)??0)+d.count)}return{count:s,mean:null,buckets:[...o.entries()].map(([e,d])=>({le:e,count:d}))}}function $t(t){const s=Xt(Ma(t.methods));if(s!==void 0)return s;if(t.received>0)return Math.max(0,Math.min(1,t.failed/t.received))}function wt(t,s){var e;const o=(e=t==null?void 0:t.endpoints)==null?void 0:e.find(d=>d.upstream===s);return Xt(o??null)}const Ua=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function kt(t){return Ua.map(({key:s,label:o,hot:e})=>{const d=t[s]==="supported";return{key:s,label:o,lit:d,hot:!!e&&d}})}function Oa(t,s,o){const e=t.Networks??[],d=e.findIndex(p=>p.ChainID===s),l={ChainID:s,Upstreams:o},v=d===-1?[...e,l]:e.map((p,E)=>E===d?l:p);return{...t,Networks:v}}function Fa(t,s){const o=t.Networks??[];return{...t,Networks:o.filter(e=>e.ChainID!==s)}}function lt(t,s,o){const e=t.Networks??[],d=e.findIndex(F=>F.ChainID===s);if(d===-1)return{...t,Networks:[...e,{ChainID:s,Upstreams:[o]}]};const l=e[d],v=l.Upstreams.findIndex(F=>F.ID===o.ID),p=v===-1?[...l.Upstreams,o]:l.Upstreams.map((F,V)=>V===v?o:F),E={...l,Upstreams:p};return{...t,Networks:e.map((F,V)=>V===d?E:F)}}function ja(t,s,o){const e=t.Networks??[],d=e.findIndex(p=>p.ChainID===s);if(d===-1)return{...t,Networks:e};const l=e[d],v={...l,Upstreams:l.Upstreams.filter(p=>p.ID!==o)};return{...t,Networks:e.map((p,E)=>E===d?v:p)}}function qa(t,s){if(t.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const o=t.filter(p=>!p.wire);if(o.length>0){const p=o.map(F=>F.id);return{level:"attention",sentence:p.length===1?"1 machine still needs setup.":`${p.length} machines still need setup.`,machines:p}}const e=s.networks??[],d=p=>{const E=e.find(F=>F.ChainID===p);return E?E.Name:`chain ${p}`},l=_a(t.map(p=>d(p.wire.ChainID))),v=t.length===1?"machine":"machines";return{level:"ok",sentence:`All ${t.length} ${v} healthy — ${Ka(l)}.`,machines:[]}}function Wa(t,s){const o=s.machines.length?` <span class="verdict-machines">${s.machines.map(e=>`<a href="#/setup/${encodeURIComponent(e)}">${a(e)}</a>`).join(" ")}</span>`:"";t.innerHTML=`
    <div class="verdict-line verdict-${s.level}">
      ${Y(s.level==="ok"?"OK":"Attention",s.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(s.sentence)}</strong>${o}
    </div>
  `}function _a(t){return[...new Set(t)]}function Ka(t){return t.length<=1?t[0]??"":t.length===2?`${t[0]} and ${t[1]}`:`${t.slice(0,-1).join(", ")} and ${t[t.length-1]}`}const Va=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function Mt(t){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:t,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Ga=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
  <symbol id="p-gear" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3.3"/><path d="M12 2.2v3.1M12 18.7v3.1M2.2 12h3.1M18.7 12h3.1M5 5l2.2 2.2M16.8 16.8 19 19M19 5l-2.2 2.2M7.2 16.8 5 19"/></symbol>
</defs></svg>`,me=t=>`<svg class="p-i"><use href="#p-${t}"/></svg>`,Ut="run",Ot=1337;function Ft(t){let s=null,o={name:"list"},e=null,d=null,l=null,v=null,p=null,E=[],F=null,V=null,A=!1,_=null,G=null,B=!1,I=null,f=!1,g=null,S=null,U=null,z=null,ee=null,ce=!1,J="";t.innerHTML=Ga+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const le=t.querySelector("#p-card");async function Z(){try{const L=await yt();s=za(L.gateways),e=null}catch(L){e=ye(L)}j()}function j(){le.innerHTML=pe()}function pe(){return e?Ja(e):o.name==="network"?os(s,o.chainId,{caps:F,capsBusy:A,capsErr:_,tls:G,tlsBusy:B,tlsErr:I,copyFlash:f,error:g,netHealth:ee,busy:l}):o.name==="endpoint"?is(s,o.chainId,o.upstreamId,{caps:F,capsBusy:A,capsErr:_,health:U,copyFlash:f,error:S,netHealth:ee}):Ya(s,l,v,E,ee)}async function de(L,M){A=!0,j();try{F=await zt(L,M),V=L,_=null}catch(T){F=null,V=L,_=ye(T)}A=!1,j()}async function ve(L,M){if(!(!M&&z===L&&U)){j();try{U=await pt(L),z=L}catch{U=null,z=L}j()}}function oe(){var T;if(!s)return"";const L=s.status.State==="running",M=[];for(const q of s.networks??[]){const P=(T=ee==null?void 0:ee.networks)==null?void 0:T.find(re=>re.chainId===q.chainId),N=P?$t(P):void 0;M.push(`n${q.chainId}:${Ge({running:L,serviceable:q.serviceable,slowRate:N})}`);for(const re of q.upstreams??[]){const ke=P?wt(P,re.id):void 0;M.push(`u${q.chainId}/${re.id}:${Ge({running:L,serviceable:!re.problem,slowRate:ke})}`)}}return M.join("|")}let we=null;async function Ce(L=!1){if(!s)return;if(ce){if(!L)return;await we,await Ce(!0);return}ce=!0;const M=s.id;we=(async()=>{try{ee=await pt(M)}catch{}})(),await we,ce=!1,we=null;const T=oe();T!==J&&(J=T,j())}async function $(L,M){var P;const T=(P=L.networks)==null?void 0:P.find(N=>N.chainId===M);if(await De({title:"Remove network",body:`Stop serving ${(T==null?void 0:T.name)??`chain ${M}`}?`,confirmLabel:"Remove",danger:!0})){g=null,j();try{await Ae(L.id,Fa(L.config,M))}catch(N){g=`Could not remove the network: ${ye(N)}`,j();return}o={name:"list"},j(),await K(L.id)}}async function h(L,M,T){var re;const q=(re=L.networks)==null?void 0:re.find(ke=>ke.chainId===M),P=H(L,M,T);if(await De({title:"Remove endpoint",body:`Stop routing to ${(P==null?void 0:P.label)??"this endpoint"}? The gateway keeps balancing across whatever else remains on ${(q==null?void 0:q.name)??`chain ${M}`}.`,confirmLabel:"Remove",danger:!0})){S=null,j();try{await Ae(L.id,ja(L.config,M,T))}catch(ke){S=`Could not remove the endpoint: ${ye(ke)}`,j();return}o={name:"network",chainId:M},j(),await K(L.id)}}Pe(le,(L,M)=>{x(L,M)});async function x(L,M){if(L==="setup"){if(l)return;await be();return}if(L==="open-settings"){he();return}if(L==="power"){if(!s||l)return;const T=Zt(s);if(T.tone==="blocked")return;if(s.status.State==="running"&&T.actions.includes("stop")){await ie(s.id,"stop");return}if(T.actions.includes("start")){await ie(s.id,"start");return}if(T.actions.includes("create")){await K(s.id);return}return}if(L==="open-network"){o={name:"network",chainId:Number(M.dataset.chainId)},g=null,G=null,I=null,f=!1,j(),s&&V!==s.id&&de(s.id,!1);return}if(L==="back"){o={name:"list"},f=!1,j();return}if(L==="back-to-network"){const T=Number(M.dataset.chainId);o=Number.isFinite(T)?{name:"network",chainId:T}:{name:"list"},S=null,f=!1,j();return}if(L==="add-network"){if(!s||l)return;await O(s);return}switch(L){case"gw-start":case"gw-stop":case"gw-restart":s&&!l&&await ie(s.id,L.slice(3));return;case"gw-create":case"gw-recreate":s&&!l&&await K(s.id);return;case"copy-url":{const T=M.dataset.url??"";if(!T)return;await ze(T)&&(f=!0,j(),window.setTimeout(()=>{f=!1,j()},1200));return}case"verify-tls":{if(!s||B)return;B=!0,I=null,j();try{G=await Gt(s.id)}catch(T){I=ye(T)}B=!1,j();return}case"open-endpoint":{const T=Number(M.dataset.chainId),q=M.dataset.upstreamId??"";if(!Number.isFinite(T)||!q)return;o={name:"endpoint",chainId:T,upstreamId:q},S=null,f=!1,j(),s&&V!==s.id&&de(s.id,!1),s&&z!==s.id&&ve(s.id,!1);return}case"add-endpoint":{if(!s||l||o.name!=="network")return;u(s,o.chainId);return}case"remove-network":{if(!s||l||o.name!=="network")return;await $(s,o.chainId);return}case"rename-endpoint":{if(!s||l||o.name!=="endpoint")return;const T=H(s,o.chainId,o.upstreamId);if(!T)return;b(s.id,o.chainId,T.id,T.label);return}case"edit-address":{if(!s||l||o.name!=="endpoint")return;const T=H(s,o.chainId,o.upstreamId);if(!T||T.kind!=="external")return;D(s.id,o.chainId,T.id,T.endpoint);return}case"remove-endpoint":{if(!s||l||o.name!=="endpoint")return;await h(s,o.chainId,o.upstreamId);return}case"recheck":{if(!s)return;const T=[de(s.id,!0),Z(),Ce(!0)];o.name==="endpoint"&&T.push(ve(s.id,!0)),await Promise.all(T);return}default:return}}function H(L,M,T){var q,P,N;return(N=(P=(q=L.networks)==null?void 0:q.find(re=>re.chainId===M))==null?void 0:P.upstreams)==null?void 0:N.find(re=>re.id===T)}function W(L,M,T){var q,P;return(P=(q=L.config.Networks)==null?void 0:q.find(N=>N.ChainID===M))==null?void 0:P.Upstreams.find(N=>N.ID===T)}function Q(L){const M=Ve();if(!M)return;const T=document.createElement("p");T.className="error small",T.textContent=L,M.appendChild(T)}function b(L,M,T,q){fe(`
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
      `,N=>{if(N==="cancel"){ne();return}N==="save"&&w(L,M,T)});const P=document.getElementById("ep-rename-input");P==null||P.focus(),P==null||P.select()}async function w(L,M,T){if(!s)return;const q=W(s,M,T);if(!q){ne();return}const P=document.getElementById("ep-rename-input"),N=document.getElementById("ep-rename-save"),re=(P==null?void 0:P.value.trim())??"";P&&(P.disabled=!0),N&&(N.disabled=!0,N.textContent="Saving…");const ke={...q,Name:re||void 0};try{await Ae(L,lt(s.config,M,ke))}catch(Ee){Q(`Could not rename the endpoint: ${ye(Ee)}`),P&&(P.disabled=!1),N&&(N.disabled=!1,N.textContent="Save");return}ne(),await K(L)}function D(L,M,T,q){fe(`
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
      `,N=>{if(N==="cancel"){ne();return}N==="save"&&X(L,M,T)});const P=document.getElementById("ep-addr-input");P==null||P.focus(),P==null||P.select()}async function X(L,M,T){if(!s)return;const q=document.getElementById("ep-addr-input"),P=document.getElementById("ep-addr-save"),N=(q==null?void 0:q.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(N)){Q("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}const re=W(s,M,T);if(!re){ne();return}q&&(q.disabled=!0),P&&(P.disabled=!0,P.textContent="Saving…");const ke={...re,Endpoint:N};try{await Ae(L,lt(s.config,M,ke))}catch(Ee){Q(`Could not save the address: ${ye(Ee)}`),q&&(q.disabled=!1),P&&(P.disabled=!1,P.textContent="Save");return}ne(),await K(L)}function u(L,M){var T;fe(`
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
      `,q=>{if(q==="cancel"){ne();return}q==="add"&&y(L.id,M)}),(T=document.getElementById("ep-add-input"))==null||T.focus()}async function y(L,M){if(!s)return;const T=document.getElementById("ep-add-input"),q=document.getElementById("ep-add-save"),P=(T==null?void 0:T.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(P)){Q("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}T&&(T.disabled=!0),q&&(q.disabled=!0,q.textContent="Adding…");const N={ID:crypto.randomUUID(),Kind:"external",Endpoint:P,Local:!1,RecentOnly:!1,Name:Da(P)};try{await Ae(L,lt(s.config,M,N))}catch(re){Q(`Could not add the endpoint: ${ye(re)}`),T&&(T.disabled=!1),q&&(q.disabled=!1,q.textContent="Add endpoint");return}ne(),await K(L,()=>void de(L,!0))}async function O(L){l="add-network",v=null,j();let M;try{M=(await He()).networks??[]}catch(P){l=null,v=`Could not load the network catalog: ${ye(P)}`,j();return}l=null,j();const T=new Set((L.networks??[]).map(P=>P.chainId)),q=M.filter(P=>!T.has(P.ChainID)).map(P=>({chainId:P.ChainID,name:P.Name}));if(T.has(Ot)||q.push({chainId:Ot,name:"Devnet"}),q.length===0){v="Every network valve's catalog knows about is already configured on this gateway.",j();return}fe(`
        <h2>Add a network</h2>
        <ul class="plain-list rpc-picker">
          ${q.map(P=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="pick:${P.chainId}">
                <span>${a(P.name)}</span>
                <span class="muted small">chain ${P.chainId}</span>
              </button>
            </li>`).join("")}
        </ul>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,P=>{if(P==="cancel"){ne();return}if(P.startsWith("pick:")){const N=Number.parseInt(P.slice(5),10);if(!Number.isFinite(N))return;ne(),te(L.id,N)}})}async function te(L,M){if(!s||l)return;l="create",v=null,j();let T;try{T=((await ft(L,M)).endpoints??[]).filter(N=>!N.alreadyAdded).map(N=>N.url)}catch(P){l=null,v=`Could not read valve's known set for chain ${M}: ${ye(P)}`,j();return}if(T.length===0){l=null,v=`valve has no measured endpoints for chain ${M} yet, so there was nothing to add.`,j();return}const q=T.map((P,N)=>({ID:`public-${M}-${N+1}`,Kind:"external",Endpoint:P,Local:!1,RecentOnly:!1}));try{await Ae(L,Oa(s.config,M,q))}catch(P){l=null,v=`Could not add the network: ${ye(P)}`,j();return}l=null,await K(L,()=>{o={name:"network",chainId:M},j(),de(L,!0)})}async function ie(L,M){if(!l){l=M,v=null,j();try{await Jt(L,M)}catch(T){v=`${M} failed: ${ye(T)}`}l=null,await Z()}}async function K(L,M){if(l)return;l="create",v=null,j();let T;try{T=await ht(L)}catch(q){v=ye(q),l=null,j();return}p==null||p(),p=Ze(T.targetId,q=>{(q.err||q.stepId===Ut&&q.done)&&(p==null||p(),p=null,l=null,q.err&&(v=`Provisioning failed: ${q.err}`),Z().then(()=>{q.err||M==null||M()}))})}async function be(){if(l)return;l="setup",v=null,E=[],j();const L=N=>{E=[...E,N],j()},M=(N,re)=>{l=null,v=re?`${N} — ${re}`:N,j()};L("Preparing your endpoint…");try{(await Me()).some(re=>re.id==="local")||await ut({id:"local",mode:"local"})}catch(N){M(`Could not register this machine: ${ye(N)}`,_e(N));return}try{const N=await Kt("local");if(!N.docker.reachable){M(N.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",N.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(N){M(`Could not check Docker on this machine: ${ye(N)}`,_e(N));return}L("Creating the gateway…");let T="default";try{T=(await Vt({id:T,placement:{targetId:"local",backend:"docker"},config:Mt([])})).id}catch(N){M(`Could not create the gateway: ${ye(N)}`,_e(N));return}L("Adding Ethereum and PulseChain endpoints…");const q=[];for(const{chainId:N}of Va)try{const ke=((await ft(T,N)).endpoints??[]).filter(Ee=>!Ee.alreadyAdded).map(Ee=>Ee.url);if(ke.length===0)continue;q.push({ChainID:N,Upstreams:ke.map((Ee,st)=>({ID:`public-${N}-${st+1}`,Kind:"external",Endpoint:Ee,Local:!1,RecentOnly:!1}))})}catch(re){M(`Could not read valve's set for chain ${N}: ${ye(re)}`,_e(re));return}if(q.length===0){M("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await Ae(T,Mt(q))}catch(N){M(`Could not save the endpoints: ${ye(N)}`,_e(N));return}L("Starting the gateway… the first run pulls the eRPC and Caddy images.");let P;try{P=await ht(T)}catch(N){M(`Could not start the gateway: ${ye(N)}`,_e(N));return}p==null||p(),p=Ze(P.targetId,N=>{const re=N.err?`${N.stepId}: ${N.err}`:N.line?`${N.stepId}: ${N.line}`:`${N.stepId}: done`;L(re),(N.err||N.stepId===Ut&&N.done)&&(p==null||p(),p=null,l=null,N.err&&(v=`Provisioning failed: ${N.err}`),E=[],Z())})}function he(){const L=vt(),M=["system","light","dark"].map(P=>`<button type="button" class="theme-opt${P===L?" active":""}" data-modal-action="theme:${P}">${P[0].toUpperCase()}${P.slice(1)}</button>`).join(""),q=!!s&&(s.actions??[]).includes("wipe")?`
        <div class="set-group">
          <div class="set-label danger">Danger zone</div>
          <button class="btn btn-danger set-wipe" data-modal-action="wipe"${l?" disabled":""}>${me("trash")} Wipe gateway</button>
          <p class="muted small">Destroys the gateway container and its stored config. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, devnet or public endpoint — is touched.</p>
        </div>`:"";fe(`
        <h2>Settings</h2>
        <div class="set-group">
          <div class="set-label">Appearance</div>
          <div class="theme-seg" role="group" aria-label="Appearance">${M}</div>
        </div>
        ${q}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Done</button>
        </div>
      `,P=>{if(P==="cancel"){ne();return}if(P.startsWith("theme:")){const N=P.slice(6);Vn(N),document.querySelectorAll(".theme-opt").forEach(re=>{re.classList.toggle("active",re.dataset.modalAction===`theme:${N}`)});return}if(P==="wipe"){if(!s||l)return;const N=s;ne(),Ne(N)}})}async function Ne(L){if(await De({title:`Wipe ${L.label}`,body:`This destroys ${L.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){l="wipe",v=null,j();try{const T=await Yt(L.id);T.error&&(v=T.error)}catch(T){v=`wipe failed: ${ye(T)}`}l=null,await Z()}}let Ue=!1;return Z().then(()=>{Ue||(J=oe(),d=window.setInterval(()=>{Ce()},5e3))}),()=>{Ue=!0,d&&window.clearInterval(d),p==null||p()}}function za(t){return!t||t.length===0?null:t.find(s=>s.placement.targetId==="local")??t[0]}function ye(t){return t instanceof Error?t.message:String(t)}function _e(t){return t instanceof Le?t.hint:void 0}function Ja(t){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(t)}</div>`}function Ya(t,s,o,e,d){var p;if(t===null)return Za(s,o,e);const l=Zt(t),v=(p=t==null?void 0:t.networks)!=null&&p.length?t.networks.map((E,F)=>as(t,E,F>0,d)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-hright">
        <span class="p-sum">${a(l.sub)}</span>
        <button type="button" class="p-gear" data-action="open-settings" title="Settings" aria-label="Settings">${me("gear")}</button>
      </span>
    </div>
    <div class="p-band">
      ${es(t,l,s,o)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${v}
      <div class="p-row p-rowdiv addr${s?" p-disabled":""}" data-action="add-network"${s?' aria-disabled="true"':""}>
        <span class="p-lead">${me("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function Za(t,s,o){const e=t==="setup",d=s?`<div class="p-emptyerr">${a(s)}</div>`:"",l=o.length?`<div class="p-setup-log" aria-live="polite">${o.map(v=>`<div>${a(v)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-hright">
        <button type="button" class="p-gear" data-action="open-settings" title="Settings" aria-label="Settings">${me("gear")}</button>
      </span>
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
  `}function Xa(t,s){return s.tone==="blocked"?null:t.status.State==="running"&&s.actions.includes("stop")?"stop":s.actions.includes("start")?"start":s.actions.includes("create")?"create":null}const Qa={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},jt={restart:"refresh",recreate:"refresh",wipe:"trash"};function es(t,s,o,e){const d=s.tone==="blocked"?s.blocked??"":s.sub,l=o?" busy":"",v=e?`<div class="p-ps" style="color:var(--red)">${a(e)}</div>`:"",p=s.tone==="blocked"&&(t!=null&&t.hint)?`<div class="p-ps">${a(t.hint)}</div>`:"",E=`
    <div class="p-power${l}" data-action="power">
      <div class="p-pbtn ${s.tone}">${me("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(s.label)}</div>
        <div class="p-ps"${s.tone==="blocked"?' style="color:var(--red)"':""}>${a(d)}</div>
        ${p}
        ${v}
      </div>
    </div>
  `,F=t?ts(t,s,o):"";return E+F}function ts(t,s,o){const e=Xa(t,s),d=(t.actions??[]).filter(v=>v!==e&&v!=="wipe");return d.length===0?"":`<div class="p-chips">${d.map(v=>{const p=Qa[v]??v,E=jt[v]?me(jt[v]):"";return`<button type="button" class="p-chip" data-action="gw-${v}" data-gid="${a(t.id)}"${o?" disabled":""}>${E}${a(p)}</button>`}).join("")}</div>`}const Ct={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function ns(t){return t.map(s=>`<svg class="p-i${s.hot?" hot":s.lit?" on":""}"><use href="#p-${Ct[s.key]}"/></svg>`).join("")}function as(t,s,o,e){var E;const d=(E=e==null?void 0:e.networks)==null?void 0:E.find(F=>F.chainId===s.chainId),l=d?$t(d):void 0,v=Ge({running:t.status.State==="running",serviceable:s.serviceable,slowRate:l}),p=kt({});return`
    <div class="p-row${o?" p-rowdiv":""}" data-action="open-network" data-chain-id="${s.chainId}">
      <span class="p-lead"><span class="p-dot ${v}"></span></span>
      <span class="p-nm">${a(s.name)}</span>
      <span class="p-caps">${ns(p)}</span>
      <span class="p-chev">${me("chevR")}</span>
    </div>
  `}function Qt(t,s){var o;return s==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(o=(t.capabilities??[]).find(e=>e.key===s))==null?void 0:o.status}function ss(t,s,o){const e=((t==null?void 0:t.endpoints)??[]).filter(l=>l.chainId===s&&o.includes(l.upstream)),d={};for(const l of["http","ws","archive","trace"])e.some(v=>Qt(v,l)==="supported")&&(d[l]="supported");return d}function os(t,s,o){var ce,J,le;const e=(ce=t==null?void 0:t.networks)==null?void 0:ce.find(Z=>Z.chainId===s);if(!t||!e)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${me("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${s}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;const d=t.status.State==="running",l=(le=(J=o.netHealth)==null?void 0:J.networks)==null?void 0:le.find(Z=>Z.chainId===s),v=l?$t(l):void 0,p=Ge({running:d,serviceable:e.serviceable,slowRate:v}),E=e.upstreams??[],F=o.tls??t.tls.verification??null,V=(F==null?void 0:F.ok)===!0,A=o.tlsBusy?"Verifying…":V?`Verified ${F?new Date(F.at).toLocaleString():""}`:"Verify HTTPS now",_=o.tlsErr?`<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${a(o.tlsErr)}</div>`:"",G=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${V?"green":"dim"}" data-action="verify-tls" title="${a(A)}">${me("lock")}</span>
          <span class="p-ic ${o.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(e.url??"")}" title="Copy the gateway URL">${me("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${a(e.url||"—")}</div>
      ${_}
    </div>
  `,B=E.map((Z,j)=>{const pe=l?wt(l,Z.id):void 0,de=Ge({running:d,serviceable:!Z.problem,slowRate:pe});return`
        <div class="p-row${j>0?" p-rowdiv":""}" data-action="open-endpoint" data-chain-id="${e.chainId}" data-upstream-id="${a(Z.id)}">
          <span class="p-lead"><span class="p-dot ${de}"></span></span>
          <span class="p-nm">${a(Z.label)}</span>
          <span class="p-chev">${me("chevR")}</span>
        </div>
      `}).join(""),I=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${E.length}</span></div>
      ${B}
      <div class="p-row${E.length>0?" p-rowdiv":""} addr${o.busy?" p-disabled":""}" data-action="add-endpoint"${o.busy?' aria-disabled="true"':""}>
        <span class="p-lead">${me("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `,f=ss(o.caps,s,E.map(Z=>Z.id)),g=kt(f),S=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${o.capsBusy&&!o.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':o.capsErr&&!o.caps?`<div class="p-caprow p-caperr">Couldn't check capabilities — ${a(o.capsErr)}</div>`:`<div class="p-caprow">${g.map(Z=>`<span class="p-capitem${Z.lit?" lit":""}">${me(Ct[Z.key])}${a(Z.label)}</span>`).join("")}</div>`}
    </div>
  `,U=d?e.serviceable?"Healthy":"Unserviceable":"Stopped",z=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${me("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${p}"></span> ${a(U)}</span></div>
    </div>
  `,ee=o.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(o.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${me("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${p}"></span> <span class="p-nmtxt">${a(e.name)}</span></span>
    </div>
    ${G}
    ${I}
    ${S}
    ${z}
    ${ee}
    <div class="p-band p-remove" data-action="remove-network">${me("trash")} Remove network</div>
  `}function rs(t,s,o){const e=((t==null?void 0:t.endpoints)??[]).find(l=>l.chainId===s&&l.upstream===o);if(!e)return{};const d={};for(const l of["http","ws","archive","trace"])Qt(e,l)==="supported"&&(d[l]="supported");return d}function is(t,s,o,e){var z,ee,ce,J,le;const d=(z=t==null?void 0:t.networks)==null?void 0:z.find(Z=>Z.chainId===s),l=(ee=d==null?void 0:d.upstreams)==null?void 0:ee.find(Z=>Z.id===o);if(!t||!d||!l)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${me("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Endpoint</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This endpoint is no longer configured.</div>
    `;const v=t.status.State==="running",p=(J=(ce=e.netHealth)==null?void 0:ce.networks)==null?void 0:J.find(Z=>Z.chainId===s),E=p?wt(p,o):void 0,F=Ge({running:v,serviceable:!l.problem,slowRate:E}),V=l.kind==="external",A=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Address</span>
        <span class="p-acts"><span class="p-ic ${e.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(l.endpoint)}" title="Copy the endpoint URL">${me("copy")}</span></span>
      </div>
      <div class="p-gwurl"${V?' data-action="edit-address" style="cursor:text"':""}>${a(l.endpoint||"—")}</div>
    </div>
  `,_=rs(e.caps,s,o),G=kt(_),B=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${e.capsBusy&&!e.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':e.capsErr&&!e.caps?`<div class="p-caprow p-caperr">Couldn't check capabilities — ${a(e.capsErr)}</div>`:`<div class="p-caprow">${G.map(Z=>`<span class="p-capitem${Z.lit?" lit":""}">${me(Ct[Z.key])}${a(Z.label)}</span>`).join("")}</div>`}
    </div>
  `,I=v?l.problem?l.problem:"Healthy":"Stopped",f=(((le=e.health)==null?void 0:le.endpoints)??[]).find(Z=>Z.chainId===s&&Z.upstream===o),g=f&&f.scored&&f.headLag>0?`<div class="p-srow"><span class="p-k">Chain head</span><span class="p-v">behind ${ge(f.headLag)} block${f.headLag===1?"":"s"}</span></div>`:"",S=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${me("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${F}"></span> ${a(I)}</span></div>
      ${g}
    </div>
  `,U=e.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(e.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${me("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${F}"></span> <span class="p-nmtxt">${a(l.label)}</span> <span class="p-pen" data-action="rename-endpoint">${me("pencil")}</span></span>
    </div>
    ${A}
    ${B}
    ${S}
    ${U}
    <div class="p-band p-remove" data-action="remove-endpoint">${me("trash")} Remove endpoint</div>
  `}function cs(t,s){let o=!1,e=[],d=null,l=!1,v=!1;t.innerHTML=`<h1>Security: ${a(s)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${$e()}</div>`;const p=t.querySelector("#sec-body"),E=t.querySelector("#sec-footer");Pe(t,(B,I)=>{var f;if(B==="rerun")V();else if(B==="toggle")(f=I.closest(".check-item"))==null||f.classList.toggle("expanded");else if(B==="copy"){const g=I.dataset.copy;g&&G(I,g)}}),F();async function F(){let B,I;try{const[g,S]=await Promise.all([Me(),He()]);B=g.find(U=>U.id===s),I=S}catch(g){if(o)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(g))}</p>`;return}if(o)return;if(!B){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!B.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const f=I==null?void 0:I.networks.find(g=>g.ChainID===B.wire.ChainID);f&&(E.innerHTML=$e(f.Name,f.LearnURL)),await V()}async function V(){l=!0,d=null,A();try{e=await sa(s),v=!0}catch(B){d=String(B instanceof Error?B.message:B)}l=!1,o||A()}function A(){p.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${l?"disabled":""}>${l?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${!v&&l?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(_).join("")}</ul>`:v?'<p class="muted">No checks returned.</p>':""}
    `}function _(B){const I=B.Status==="pass"?"ok":B.Status==="fail"?"bad":B.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${Y(B.Status,I)}
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
    `}async function G(B,I){const f=await ze(I),g=B.textContent;B.textContent=f?"Copied!":"Copy failed",setTimeout(()=>{o||(B.textContent=g)},1500)}return()=>{o=!0}}const ls=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],bt="VALVE_API_KEY";function ds(t){return t===bt?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(t)}}</code> slot wherever an endpoint URL carries one.`}function us(t){let s=!1,o=!1,e=!1,d=null,l=!1,v=null,p=null;const E=new Set,F=new Map;let V="",A="";t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${$e()}`;const _=t.querySelector("#settings-body");Pe(t,(S,U)=>{if(S==="save"&&g(),S==="clear-key"){if(!v)return;o=!0;const z=t.querySelector("#ai-key");z&&(z.value=""),f(v)}if(S==="clear-provider-key"){const z=U.dataset.key;if(!v||!z)return;E.add(z),F.set(z,""),l=!1,f(v)}}),gt(t,(S,U)=>{S!=="ai-provider"||!v||(p=U,l=!1,f(v))}),G();async function G(){try{const S=await va();if(s)return;v=S,f(S)}catch(S){if(s)return;_.innerHTML=`<p class="error">Failed to load settings: ${a(String(S))}</p>`}}function B(S){const z=(Array.isArray(S.providerKeysSet)?S.providerKeysSet:[]).filter(ee=>ee!==bt).sort();return[bt,...z]}function I(S,U){const z=a(S);return`
      <div class="pk-row">
        <label>
          <code>${z}</code>
          <input class="provider-key" data-key="${z}" type="password" autocomplete="off"
                 placeholder="${U?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${ds(S)}</p>
        ${U?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${z}">Clear saved key</button>`:""}
      </div>`}function f(S){var Z;const U=p??S.aiProvider,z=Array.isArray(S.providerKeysSet)?S.providerKeysSet:[],ee=B(S).map(j=>I(j,z.includes(j))).join("");_.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${mt("ai-provider",ls.map(j=>({value:j.value,label:j.label})),U)}
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
    `;const ce=t.querySelector("#ai-key");ce==null||ce.addEventListener("input",()=>{o=!0,l=!1}),(Z=t.querySelector("#ref-rpc-base"))==null||Z.addEventListener("input",()=>{l=!1}),t.querySelectorAll("input.provider-key").forEach(j=>{const pe=j.dataset.key;if(!pe)return;const de=F.get(pe);de!==void 0&&(j.value=de),j.addEventListener("input",()=>{E.add(pe),F.set(pe,j.value),l=!1})});const J=t.querySelector("#pk-new-value");J&&(J.value=A),J==null||J.addEventListener("input",()=>{A=J.value,l=!1});const le=t.querySelector("#pk-new-name");le==null||le.addEventListener("input",()=>{V=le.value,l=!1})}async function g(){const S=t.querySelector("#ai-key"),U=t.querySelector("#ref-rpc-base");if(!S||!U||!v)return;const z={aiProvider:p??v.aiProvider,refRpcBase:U.value.trim()};o&&(z.aiKey=S.value);const ee={};for(const J of E)ee[J]=F.get(J)??"";const ce=V.trim();ce&&(ee[ce]=A),Object.keys(ee).length>0&&(z.providerKeys=ee),e=!0,d=null,l=!1,f(v);try{const J=await ya(z);if(s)return;v=J,o=!1,E.clear(),F.clear(),V="",A="",e=!1,l=!0,f(J)}catch(J){if(s)return;e=!1,d=String(J instanceof Error?J.message:J),f(v)}}return()=>{s=!0}}const ps=["http","ws","archive","trace"],hs={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Ke=1337,fs="run",ms={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function bs(t){let s=!1,o=null,e=null;const d={},l={},v={},p={},E={},F={},V={},A={},_={},G={},B={},I={},f={},g={},S={};let U="",z=null;t.innerHTML=`
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
  `;const ee=t.querySelector("#rpc-body");Pe(t,(n,r)=>{bn(n,r)}),gt(t,()=>{}),J(),ce();async function ce(){try{const n=await _t();if(s)return;U=n.os,oe()}catch{}}async function J(){try{const n=await yt();if(s)return;o=n,e=null}catch(n){if(s)return;o=null,e=Se(n)}oe();for(const n of(o==null?void 0:o.gateways)??[])le(n.id),Z(n.id,!1)}async function le(n){try{const r=await ha(n);if(s)return;d[n]=r}catch{if(s)return;d[n]=null}oe()}async function Z(n,r){v[n]=r,r&&oe();try{const i=await zt(n,r);if(s)return;l[n]=i}catch{if(s)return;l[n]=null}v[n]=!1,oe()}function j(n){return((o==null?void 0:o.gateways)??[]).find(r=>r.id===n)}function pe(n,r){return(n.networks??[]).find(i=>i.chainId===r)}function de(n,r,i){var m;const c=(((m=d[n])==null?void 0:m.networks)??[]).find(C=>C.chainId===r);return((c==null?void 0:c.upstreams)??[]).find(C=>C.upstream===i)}function ve(n,r,i){var c;return(((c=l[n])==null?void 0:c.endpoints)??[]).find(m=>m.chainId===r&&m.upstream===i)}function oe(){if(s)return;if(e){ee.innerHTML=`<p class="error">Could not read the gateways: ${a(e)}</p>`;return}if(!o){ee.innerHTML='<p class="muted">Loading…</p>';return}const n=o.gateways??[],r=n.length>1,i=(o.targets??[]).some(C=>Rt(C.id,n)),c=new Set(n.map(C=>C.placement.targetId)),m=(o.orphans??[]).filter(C=>!c.has(C.targetId));ee.innerHTML=`
      ${n.map(C=>$(C,r)).join("")}
      ${n.length===0?Ce():""}
      ${m.map(we).join("")}
      ${i?`<div class="card-actions rpc-add-gateway">
               <button class="btn${n.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${n.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function we(n){const r=`docker rm -f ${n.containerName}`,i=f[n.containerName];return`
      <div class="strip">
        ${O({tone:"warn",text:`${n.containerName} is still running on ${n.targetId}. Its chains were folded into ${n.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:r})}
        ${i?O({tone:"bad",text:i}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(n.containerName)}">Dismiss this record</button>
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
    `}function $(n,r){return`
      ${r?`<h2 class="rpc-machine">${a(n.placement.targetId)}</h2>`:""}
      ${h(n)}
      ${y(n)}
      ${he(n)}
      ${Ne(n)}
      ${b(n)}
    `}function h(n){const r=n.status.State==="running",i=n.tls,c=[`on <strong>${a(n.placement.targetId)}</strong>`];return n.status.Image&&c.push(`<code>${a(n.status.Image)}</code>`),c.push(i!=null&&i.enabled?`HTTPS front <code>${a(i.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${K(n)}
        <strong>${a(n.label)}</strong>
        ${ie(n)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${r?`base <code>${a(n.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function x(n){const r=n.tls;return r!=null&&r.enabled&&r.rootCaPath&&r.effectiveCertSource==="internal"?r.rootCaPath:null}function H(n){var r;return((r=((o==null?void 0:o.targets)??[]).find(i=>i.id===n.placement.targetId))==null?void 0:r.mode)??""}function W(n){switch(n){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return n||"this device"}}function Q(n,r,i){switch(n){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${r}"`;case"windows":return`certutil -addstore -f ROOT "${r}"`;case"linux":default:return`sudo cp "${r}" /usr/local/share/ca-certificates/valve-node-app-${i}.crt && sudo update-ca-certificates`}}function b(n){const r=_[n.id]??!1,i=((o==null?void 0:o.orphans)??[]).filter(c=>c.targetId===n.placement.targetId);return`
      <section class="card manage-section${r?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(n.id)}" aria-expanded="${r}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${w(n,i.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${r?D(n,i):""}
      </section>
    `}function w(n,r){const i=[];return n.status.State!=="running"&&i.push("gateway not running"),r>0&&i.push(`${r} leftover container${r===1?"":"s"}`),i.length===0?"container, settings, certificate":i.join(" · ")}function D(n,r){var i;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(n.actions??[]).map(c=>be(n,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(n.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${a(n.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${v[n.id]?"disabled":""}>
            ${v[n.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${a(n.id)}">
            ${V[n.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${a(n.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${n.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${a(n.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(n.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${a(((i=(n.networks??[])[0])==null?void 0:i.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${a(n.baseUrl)}</code> once it is running.</p>`}
        ${X(n)}
        ${r.map(we).join("")}
        ${V[n.id]?ln(n):""}
      </div>
    `}function X(n){const r=x(n);if(!r)return"";const i=H(n)==="local",c=Q(U,r,n.id),m=S[n.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(n.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(r)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(r)}">Copy path</button>
        </div>
        ${i?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(n.id)}" ${g[n.id]?"disabled":""}>
                   ${g[n.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${m?u(m):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${i?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${U?` (${a(W(U))})`:""}:</span>
          <code class="strip-cmd">${a(c)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(c)}">Copy command</button>
        </div>
      </div>
    `}function u(n){return n.ok?`<div class="strip-line strip-note"><span class="strip-text">${a(n.message)}</span></div>`:`
      <div class="strip-line strip-warn">
        <span class="strip-text">${a(n.message)}</span>
        ${n.ranCommand?`<code class="strip-cmd">${a(n.ranCommand)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(n.ranCommand)}">Copy</button>`:""}
      </div>
    `}function y(n){const r=[];n.error&&r.push({tone:"bad",text:`This gateway could not be read: ${n.error}${n.hint?` — ${n.hint}`:""}`}),n.blocked&&r.push({tone:"warn",text:n.blocked});for(const c of n.warnings??[])r.push({tone:"warn",text:c});r.push(...te(n));const i=E[n.id];return i&&r.push({tone:"bad",text:i}),r.length===0?"":`<div class="strip">${r.map(O).join("")}</div>`}function O(n){return`
      <div class="strip-line strip-${n.tone}">
        <span class="strip-text">${a(n.text)}</span>
        ${n.cmd?`<code class="strip-cmd">${a(n.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(n.cmd)}">Copy</button>`:""}
      </div>
    `}function te(n){var m,C;const r=n.tls;if(!(r!=null&&r.enabled))return[];const i=[];r.fallback&&i.push({tone:"warn",text:r.fallback}),r.error?i.push({tone:"warn",text:`HTTPS front: ${r.error}`}):((m=r.status)==null?void 0:m.State)!=="running"&&i.push({tone:"warn",text:`The HTTPS front is ${((C=r.status)==null?void 0:C.State)??"unknown"}, so nothing answers on ${r.url??"its https URL"} even if the gateway itself is up.`,cmd:r.containerName?`docker start ${r.containerName}`:void 0});const c=G[n.id]??r.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&i.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&i.push({tone:"warn",text:c.expiryWarning}),i}function ie(n){switch(n.status.State){case"running":return Y("running","ok");case"created-but-stopped":return Y("stopped","warn");case"not-created":return Y("not created","neutral");default:return Y("unknown","bad")}}function K(n){return n.status.State==="running"?Be("ok"):n.status.State==="unknown"?Be("bad"):Be("neutral")}function be(n,r){const i=ms[r];if(!i)return"";const c=p[n.id];return`
      <button class="${i.className}" data-action="gw-${r}" data-gid="${a(n.id)}"
              title="${a(i.title)}" ${c?"disabled":""}>
        ${c===r?'<span class="spinner" aria-label="working"></span>':a(i.label)}
      </button>
    `}function he(n){const r=F[n.id]??[];return r.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(n.placement.targetId)}</p>
        <pre class="step-log">${a(r.join(`
`))}</pre>
      </div>
    `}function Ne(n){const r=Ue(n.networks??[]),i=r.some(c=>c.chainId===Ke);return r.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(n.id)}">Add a network</button>
            ${P(n,i)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${r.map(c=>L(n,c)).join("")}
        </div>
        ${q(n,i)}
        ${cn(n)}
      </div>
    `}function Ue(n){const r=n.filter(c=>c.chainId!==Ke),i=n.filter(c=>c.chainId===Ke);return[...r,...i]}function L(n,r){const i=re(r),c=r.chainId===Ke,m=`${n.id}:${r.chainId}`,C=A[m]??!1,R=i.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${i.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(r.name)}</span>
          <code class="chain-key">evm:${r.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${Y(R,i.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(m)}" aria-expanded="${C}">
              ${C?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${M(n,r)}
        ${C?T(n,r,i):""}
      </section>
    `}function M(n,r){if(!r.url)return`<p class="chain-connect-none muted small">${n.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const i=x(n);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(r.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(r.url)}"
                title="Copy ${a(r.url)}">Copy URL</button>
        ${i?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${H(n)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(n.id)}" ${g[n.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${g[n.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(i)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(n.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${S[n.id]?`<span class="chain-cert muted small">${a(S[n.id].ok?"Trusted — reload your wallet or browser.":S[n.id].message)}</span>`:""}`:""}
      </div>
    `}function T(n,r,i){const c=r.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${i.why?" chain-verdict-why":""}"${i.why?` title="${a(i.why)}"`:""}>${i.html}</p>
        <div class="chain-detail-bar">
          ${N(c.length,i.tone,r.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(n.id)}" data-chain="${r.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(n.id)}" data-chain="${r.chainId}">Remove</button>
        </div>
        ${st(n,r)}
        ${(r.warnings??[]).map(m=>`<p class="chain-note">${a(m)}</p>`).join("")}
      </div>
    `}function q(n,r){const i=l[n.id],c=i!=null&&i.at?`probed ${a(xt(i.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(n.id)}">+ Network</button>
        ${P(n,r)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(n.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${v[n.id]?"disabled":""}>
          ${v[n.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function P(n,r){return r?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(n.id)}"
                    title="Add a throwaway local test chain (evm:${Ke}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function N(n,r,i){const c=i>0,m=c?i:n,C=Math.min(n,m);let R="";for(let We=0;We<m;We++)R+=`<span class="seg${We<C?` seg-on seg-${r}`:""}"></span>`;const k=c&&n>i,ae=c?k?`${n} (set is ${i})`:`${n} of ${i}`:`${n}`,ue=`${n} upstream${n===1?"":"s"} configured`,xe=c?`${ue}${k?`, ${n-i} beyond the set`:""}. valve's set for this chain is ${i}.`:`${ue}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(xe)}">${R}</span>
      <span class="segs-n">${ae}</span>
    `}function re(n){const r=n.upstreams??[];if(r.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!n.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!r.some(ke)){const c=Ee(r);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(C=>`<code>${a(C)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(r.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!r.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const i=r.filter(c=>!!c.problem);if(i.length>0){const c=r.length-i.length;return{tone:"warn",html:`${i.length} of these ${r.length} endpoints ${i.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${r.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function ke(n){return/^wss?:\/\//i.test((n.endpoint??"").trim())}function Ee(n){const r=new Set;for(const i of n){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((i.endpoint??"").trim());c&&r.add(c[1].toLowerCase())}return[...r].sort()}function st(n,r){const i=r.upstreams??[];return i.length===0?"":`<ul class="ups">${i.map(c=>tn(n,r,c)).join("")}</ul>`}function tn(n,r,i){const c=`${n.id}|${r.chainId}|${i.id}`,m=i.actions??[];return`
      <li class="up${i.problem?" up-bad":""}">
        <div class="up-what">
          ${i.problem?Be("bad"):Be("ok")}
          <span class="up-label">${a(i.label)}</span>
          ${nn(i)}
        </div>
        <code class="up-url">${a(i.endpoint||"—")}</code>
        <div class="up-caps">${an(n,r,i)}</div>
        <div class="up-share">${rn(n,r,i)}</div>
        <div class="up-acts">
          ${m.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${a(c)}"
                         data-target="${a(i.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${p[n.id]?"disabled":""}>
                   ${p[n.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${a(c)}">Remove</button>
        </div>
        ${i.problem?`<div class="up-problem error small">${a(i.problem)}</div>`:""}
      </li>
    `}function nn(n){return n.problem?Y("unusable","bad"):n.recentOnly?Y("recent blocks","warn"):n.local?Y("yours","ok"):Y("public","neutral")}function St(n,r){var i;if(n)return r==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(i=(n.capabilities??[]).find(c=>c.key===r))==null?void 0:i.status}function an(n,r,i){const c=ve(n.id,r.chainId,i.id);return c?c.unprobeable?`<span class="caps-none" title="${a(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${ps.map(m=>sn(n,r,c,m)).join("")}</span>`:`<span class="muted small">${l[n.id]===void 0?"probing…":"—"}</span>`}function sn(n,r,i,c){const m=(i.capabilities??[]).find(ue=>ue.key===c),C=St(i,c)??"inconclusive",R=hs[c]??c.toUpperCase();let k="cap";C==="unsupported"?k=on(n,r,c)?"cap missing":"cap off":C==="inconclusive"?k="cap unknown":C==="inconsistent"&&(k="cap mixed");const ae=m!=null&&m.detail?`${m.label}: ${m.detail}`:c==="http"&&i.reachDetail?`Answers JSON-RPC over HTTP: ${i.reachDetail}`:`${R}: no verdict`;return`<span class="${k}" title="${a(ae)}">${a(R)}</span>`}function on(n,r,i){const c=(r.upstreams??[]).map(m=>ve(n.id,r.chainId,m.id)).filter(m=>!!m&&!m.unprobeable);return c.length>0&&c.every(m=>St(m,i)==="unsupported")}function rn(n,r,i){const c=d[n.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const m=de(n.id,r.chainId,i.id),C=(c.networks??[]).find(xe=>xe.chainId===r.chainId);if(!m||!C||C.attributed===0)return'<span class="muted small">no traffic yet</span>';const R=Math.round(m.actual*100),k=Math.round(m.intended*100),ae=m.diverged?i.local?"warn":"":"ok",ue=`${m.succeeded.toLocaleString()} of ${C.attributed.toLocaleString()} answered requests · routing intends ${k}%`+(m.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(ue)}">
        <span class="bar">
          <span class="fill${ae?" "+ae:""}" style="width:${R}%"></span>
          <span class="tick" style="left:${k}%"></span>
        </span>
        <span class="share-n${m.diverged?" warn":""}">${R}%</span>
        ${m.unconfigured?Y("not in config","warn"):""}
      </span>
    `}function cn(n){const r=d[n.id];return r?r.enabled?r.error?`<p class="muted small">The request counters could not be read: ${a(r.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${r.since?` (${a(xt(r.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function xt(n){const r=new Date(n);return Number.isNaN(r.getTime())?n:r.toLocaleString()}function ln(n){const r=n.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${a(n.id)}-port" value="${r.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${a(n.id)}-bind" value="${a(r.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${a(r.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${dn(n)}
        ${un(n)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(n.id)}">Save settings</button>
        </div>
      </div>
    `}function dn(n){const r=!n.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${a(n.id)}-metrics" ${r?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function un(n){var R;const r=a(n.id),i=n.config.TLS??null,c=(i==null?void 0:i.Enabled)??!1,m=(i==null?void 0:i.CertSource)||"internal",C=((R=n.tls)==null?void 0:R.suggestedHostname)??"";return`
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
      ${pn(n)}
    `}function pn(n){var R,k;const r=a(n.id),i=((R=n.config.TLS)==null?void 0:R.Enabled)??!1,c=G[n.id]??((k=n.tls)==null?void 0:k.verification)??null,m=B[n.id]??!1,C=I[n.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${r}" ${i&&!m?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${m?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${i?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${C?`<p class="error small">${a(C)}</p>`:""}
      ${c?hn(c):""}
    `}function hn(n){const r=(n.assertions??[]).map(i=>`
          <li class="small">
            ${fn(i.status)}
            <strong>${a(i.title)}</strong>
            <div class="muted">${a(i.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${n.ok?n.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${a(n.summary)}
      </div>
      <ul class="verify-list">${r}</ul>
      <p class="muted small">
        Checked ${a(new Date(n.at).toLocaleString())} against <code>${a(n.address)}</code>
        ${n.notAfter?`· certificate valid until <code>${a(new Date(n.notAfter).toLocaleString())}</code> (${a(n.expiresIn??"")})`:""}
      </p>
      ${n.expiryWarning?`<div class="banner banner-warn">${a(n.expiryWarning)}</div>`:""}
    `}function fn(n){switch(n){case"pass":return Y("pass","ok");case"fail":return Y("fail","bad");case"unavailable":return Y("unavailable","warn");default:return Y("skipped","neutral")}}async function mn(n){B[n]=!0,I[n]=null,oe();try{G[n]=await Gt(n)}catch(r){I[n]=`${Se(r)}${qe(r)}`}finally{B[n]=!1,oe()}}function Oe(n){return{...n.config,Networks:(n.config.Networks??[]).map(r=>({ChainID:r.ChainID,Upstreams:r.Upstreams.map(i=>({...i}))}))}}async function Fe(n,r,i){E[n]=null;try{await Ae(n,r)}catch(c){return E[n]=`${i?i+": ":""}${Se(c)}`,oe(),!1}return await J(),!0}async function bn(n,r){const i=r.dataset.gid??"";switch(n){case"refresh":await J();return;case"copy":r.dataset.copy&&await jn(r,r.dataset.copy);return;case"reprobe":await Z(i,!0);return;case"toggle-settings":V[i]=!V[i],oe();return;case"toggle-manage":_[i]=!_[i],oe();return;case"toggle-chain-detail":{const c=r.dataset.key??"";c&&(A[c]=!A[c]),oe();return}case"save-settings":await vn(i);return;case"verify-tls":await mn(i);return;case"trust-cert":await $n(i);return;case"gw-start":case"gw-stop":case"gw-restart":await wn(i,n.slice(3));return;case"gw-create":case"gw-recreate":await kn(i);return;case"gw-wipe":Mn(i);return;case"add-gateway":On();return;case"forget-gateway":await Cn(i);return;case"dismiss-orphan":await Sn(r.dataset.name??"");return;case"add-chain":xn(i);return;case"add-devnet":{const c=j(i);if(c){const m=((o==null?void 0:o.targets)??[]).some(C=>C.id===c.placement.targetId&&C.hasDevnet);Et(i,Ke,m)}return}case"remove-chain":await In(i,Number.parseInt(r.dataset.chain??"",10));return;case"add-endpoint":Pt(i,Number.parseInt(r.dataset.chain??"",10));return;case"remove-endpoint":await Pn(r.dataset.key??"");return;case"reset-devnet":await Dn(r.dataset.key??"",r.dataset.target??"");return;default:return}}async function vn(n){const r=j(n);if(!r)return;const i=Oe(r),c=t.querySelector(`#gw-${CSS.escape(n)}-port`),m=t.querySelector(`#gw-${CSS.escape(n)}-bind`);if(c){const k=Number.parseInt(c.value.trim(),10);Number.isFinite(k)&&(i.Port=k)}m&&(i.BindAddr=m.value.trim());const C=t.querySelector(`#gw-${CSS.escape(n)}-metrics`);C&&(i.MetricsOff=!C.checked),i.TLS=yn(n,r);const R=r.status.State==="running";await Fe(n,i,"Saving settings")&&(V[n]=!1,R&&(E[n]=null,gn(n,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),oe())}function yn(n,r){var C,R,k,ae,ue,xe,We;const i=qn=>t.querySelector(`#gw-${CSS.escape(n)}-${qn}`),c=i("tls");if(!c)return r.config.TLS??null;const m=Number.parseInt(((C=i("tls-port"))==null?void 0:C.value.trim())??"",10);return{Enabled:c.checked,Hostname:((R=i("tls-host"))==null?void 0:R.value.trim())??"",CertSource:((k=i("tls-source"))==null?void 0:k.value)??"internal",CertFile:((ae=i("tls-cert"))==null?void 0:ae.value.trim())??"",KeyFile:((ue=i("tls-key"))==null?void 0:ue.value.trim())??"",HTTPSPort:Number.isFinite(m)?m:443,BindAddr:((xe=r.config.TLS)==null?void 0:xe.BindAddr)??"",ImageRef:((We=r.config.TLS)==null?void 0:We.ImageRef)??""}}function gn(n,r){F[n]=[r]}async function $n(n){if(!g[n]){g[n]=!0,S[n]=null,oe();try{S[n]=await ma(n)}catch(r){S[n]={ok:!1,message:`${Se(r)}${qe(r)}`}}g[n]=!1,oe()}}async function wn(n,r){if(!p[n]){p[n]=r,E[n]=null,oe();try{await Jt(n,r)}catch(i){E[n]=`${r} failed: ${Se(i)}${qe(i)}`}p[n]=null,await J()}}async function kn(n){if(p[n])return;p[n]="create",E[n]=null,F[n]=["starting…"],oe();let r;try{r=await ht(n)}catch(i){E[n]=`${Se(i)}${qe(i)}`,F[n]=[],p[n]=null,oe();return}z==null||z(),z=Ze(r.targetId,i=>{if(s)return;const c=i.err?`${i.stepId}: ${i.err}`:i.line?`${i.stepId}: ${i.line}`:`${i.stepId}: done`;if(F[n]=[...(F[n]??[]).filter(C=>C!=="starting…"),c],!!i.err||i.stepId===fs&&!!i.done){z==null||z(),z=null,p[n]=null,i.err&&(E[n]="Provisioning failed — see the log below."),J();return}oe()})}async function Cn(n){const r=j(n);if(!(!r||!await De({title:`Forget ${r.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${r.containerName}" on ${r.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await fa(n)}catch(c){E[n]=Se(c),oe();return}await J()}}async function Sn(n){if(n){f[n]=null;try{await pa(n)}catch(r){f[n]=Se(r),oe();return}await J()}}function xn(n){const r=j(n);if(!r)return;const i=new Set((r.networks??[]).map(k=>k.chainId)),c=(o==null?void 0:o.presets)??[],m=c.filter(k=>!i.has(k.chainId)),C=c.filter(k=>i.has(k.chainId)),R=((o==null?void 0:o.targets)??[]).some(k=>k.id===r.placement.targetId&&k.hasDevnet);fe(`
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
                <span class="muted small">chain ${k.chainId}${k.devnet?R?" · uses the devnet on "+a(r.placement.targetId):" · will create a devnet on "+a(r.placement.targetId):""}</span>
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
      `,k=>{if(k==="cancel"){ne();return}if(k==="custom"){Tn(n);return}if(k.startsWith("preset:")){const ae=Number.parseInt(k.slice(7),10),ue=c.find(xe=>xe.chainId===ae);ne(),ue!=null&&ue.devnet?Et(n,ae,R):Tt(n,ae)}})}function Tn(n){var r;fe(`
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
      `,i=>{if(i==="cancel"){ne();return}if(i!=="add")return;const c=document.getElementById("custom-chain-id"),m=document.getElementById("custom-chain-err"),C=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(C)||C<=0){m&&(m.className="error small"),m&&(m.textContent="A chain id is a positive whole number.");return}ne(),Tt(n,C)}),(r=document.getElementById("custom-chain-id"))==null||r.focus()}async function Tt(n,r){const i=j(n);if(!i)return;const c=Oe(i),m=c.Networks??[];m.some(C=>C.ChainID===r)||(m.push({ChainID:r,Upstreams:[]}),c.Networks=m,await En(n,c)&&(oe(),Pt(n,r)))}async function En(n,r){var C;const i={...r,Networks:(r.Networks??[]).filter(R=>R.Upstreams.length>0)};if(!await Fe(n,i))return!1;const m=j(n);if(m)for(const R of r.Networks??[])R.Upstreams.length===0&&!(m.networks??[]).some(k=>k.chainId===R.ChainID)&&(m.config.Networks=[...m.config.Networks??[],{ChainID:R.ChainID,Upstreams:[]}],m.networks=[...m.networks??[],{chainId:R.ChainID,name:((C=((o==null?void 0:o.presets)??[]).find(k=>k.chainId===R.ChainID))==null?void 0:C.name)??`Chain ${R.ChainID}`,path:`/${m.config.ProjectID}/evm/${R.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function Et(n,r,i){const c=j(n);if(!c)return;if(!i){fe(`
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
        `,()=>ne());return}const m=Oe(c),C=m.Networks??[],R={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},k=C.find(ae=>ae.ChainID===r);k?k.Upstreams.push(R):C.push({ChainID:r,Upstreams:[R]}),m.Networks=C,await Fe(n,m,"Adding the devnet")}async function In(n,r){const i=j(n);if(!i||!Number.isFinite(r))return;const c=pe(i,r);if(!await De({title:`Remove ${(c==null?void 0:c.name)??`chain ${r}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${r}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const C=Oe(i);C.Networks=(C.Networks??[]).filter(R=>R.ChainID!==r),await Fe(n,C,"Removing the network")}function It(n){const r=n.split("|");return r.length!==3?null:{gid:r[0],chainId:Number.parseInt(r[1],10),upstreamId:r[2]}}async function Pn(n){const r=It(n);if(!r)return;const i=j(r.gid);if(!i)return;const c=Oe(i),m=(c.Networks??[]).find(k=>k.ChainID===r.chainId);if(!m)return;const C=m.Upstreams.findIndex((k,ae)=>(k.ID||`${r.chainId}-${ae}`)===r.upstreamId);C<0||!await De({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(m.Upstreams.splice(C,1),await Fe(r.gid,c,"Removing the endpoint"))}function Pt(n,r){const i=j(n);if(!i||!Number.isFinite(r))return;const c=((o==null?void 0:o.sources)??[]).filter(k=>k.chainId===r),m=pe(i,r),C=new Set(((m==null?void 0:m.upstreams)??[]).filter(k=>k.kind!=="external").map(k=>`${k.kind}|${k.targetId??""}`)),R=c.filter(k=>!C.has(`${k.kind}|${k.targetId}`));fe(`
        <h2>Add an endpoint for ${a((m==null?void 0:m.name)??`chain ${r}`)}</h2>
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
               </ul>`:`<p class="muted small">No machine you manage serves chain ${r}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn" data-modal-action="known-set">Add valve's set…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,k=>{if(k==="cancel"){ne();return}if(k==="known-set"){Nn(n,r);return}if(k==="manual"){Bn(n,r);return}if(k.startsWith("source:")){const[,ae,ue]=k.split(":");ne(),Rn(n,r,ae,ue)}})}async function Rn(n,r,i,c){const m=j(n);if(!m)return;const C=Oe(m),R=C.Networks??[],k={ID:`${i==="managed-devnet"?"devnet":"node"}-${c}`,Kind:i,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},ae=R.find(ue=>ue.ChainID===r);ae?ae.Upstreams.push(k):R.push({ChainID:r,Upstreams:[k]}),C.Networks=R,await Fe(n,C,"Adding the endpoint")}function Ln(n){const r=[...n].sort((m,C)=>(m.latencyMs??1e9)-(C.latencyMs??1e9)),i=r.slice(0,3),c=r.find(m=>m.url.startsWith("wss://")||m.url.startsWith("ws://"));return c&&!i.some(m=>m.url===c.url)&&(i.length===3&&i.pop(),i.push(c)),new Set(i.map(m=>m.url))}async function Nn(n,r){let i;try{i=await ft(n,r)}catch(k){fe(`<h2>Endpoints for chain ${r}</h2>
         <p class="error small">Could not read the set: ${a(Se(k))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>ne());return}if(s)return;const c=i.endpoints??[],m=c.filter(k=>!k.alreadyAdded).map(k=>k.url),C=new Set(c.map(k=>k.provider)).size,R=c.map(k=>{const ae=[k.websocket?'<span class="t ws">websocket</span>':"",k.archive?'<span class="t ar">archive</span>':"",k.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(k.url)}</code>
                  <span class="muted small">${a(k.provider)}</span> ${ae}</li>`}).join("");fe(`<h2>Endpoints for chain ${r}</h2>
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
       </div>`,k=>{ne(),k==="add"&&ot(n,r,m),k==="discover"&&An(n,r)})}async function An(n,r){fe(`
        <h2>Public endpoints for chain ${r}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,R=>{R==="cancel"&&ne()});let i;try{i=await ba(r)}catch(R){const k=Ve();if(k){const ae=document.createElement("p");ae.className="error small",ae.textContent=`Could not discover endpoints: ${Se(R)}`,k.appendChild(ae)}return}if(s)return;const c=(i.endpoints??[]).filter(R=>R.status==="live"||R.status==="unprobed"),m=(i.endpoints??[]).filter(R=>R.status==="rejected"),C=Ln(c);fe(`
        <h2>Public endpoints for chain ${r}</h2>
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
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${r} right now.</p>`}
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
      `,R=>{if(R==="cancel"){ne();return}if(R==="add"){const k=Ve(),ae=k?Array.from(k.querySelectorAll('input[type="checkbox"]:checked')).map(ue=>ue.value):[];ne(),ot(n,r,ae);return}})}function Bn(n,r){var i;fe(`
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
      `,c=>{if(c==="cancel"){ne();return}if(c!=="add")return;const m=document.getElementById("manual-endpoint"),C=document.getElementById("manual-recent"),R=document.getElementById("manual-err"),k=(m==null?void 0:m.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(k)){R&&(R.className="error small",R.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}ne(),ot(n,r,[k],(C==null?void 0:C.checked)??!1)}),(i=document.getElementById("manual-endpoint"))==null||i.focus()}async function ot(n,r,i,c=!1){if(!i.length)return;const m=j(n);if(!m)return;const C=Oe(m),R=C.Networks??[];let k=R.find(ue=>ue.ChainID===r);k||(k={ChainID:r,Upstreams:[]},R.push(k));let ae=1;for(const ue of k.Upstreams){const xe=/^public-\d+-(\d+)$/.exec(ue.ID??"");xe&&(ae=Math.max(ae,Number(xe[1])+1))}for(const ue of i)k.Upstreams.some(xe=>xe.Endpoint===ue)||k.Upstreams.push({ID:`public-${r}-${ae++}`,Kind:"external",Endpoint:ue,Local:!1,RecentOnly:c});C.Networks=R,await Fe(n,C,i.length===1?"Adding the endpoint":`Adding ${i.length} endpoints`)}async function Dn(n,r){const i=It(n);if(!i||!r||!await De({title:"Reset this devnet",body:`The chain on ${r} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;p[i.gid]="reset",E[i.gid]=null,oe();let m;try{m=await da(r)}catch(C){E[i.gid]=`Reset failed: ${Se(C)}${qe(C)}`,p[i.gid]=null,oe();return}p[i.gid]=null,Hn(r,m),await J()}function Hn(n,r){const i=[];i.push(r.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),r.report.Recreated&&i.push("A fresh chain was started from genesis.");const c=r.report.Cascaded??[],m=r.report.CascadeSkipped??[];fe(`
        <h2>Devnet on ${a(n)} reset</h2>
        <ul class="plain-list">${i.map(C=>`<li>${a(C)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${a(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${m.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(m.join(", "))}.</p>`:""}
        ${r.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>ne())}function Mn(n){const r=j(n);if(!r)return;fe(`
        <h2>Wipe ${a(r.label)}</h2>
        <p class="error">This destroys ${a(r.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(n)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(n)}</button>
        </div>
      `,m=>{if(m==="cancel"||m==="close"){ne(),J();return}m==="confirm"&&Un(n)});const i=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");i==null||i.addEventListener("input",()=>{c&&(c.disabled=i.value.trim()!==n)}),i==null||i.focus()}async function Un(n){const r=document.getElementById("wipe-confirm-btn");r&&(r.disabled=!0,r.textContent="Wiping…");let i;try{i=await Yt(n)}catch(c){const m=Ve();if(m){const C=document.createElement("p");C.className="error small",C.textContent=`Wipe failed: ${Se(c)}${qe(c)}`,m.appendChild(C)}r&&(r.disabled=!1,r.textContent=`Wipe ${n}`);return}fe(`
        <h2>${a(n)} wiped</h2>
        <ul class="plain-list">
          <li>${i.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${i.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${i.error?`<p class="error small">${a(i.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{ne(),J()})}function Rt(n,r){return!r.some(i=>{var c;return((c=i.placement)==null?void 0:c.targetId)===n})}function On(){var C;const n=(o==null?void 0:o.targets)??[],r=(o==null?void 0:o.gateways)??[],i=n.filter(R=>Rt(R.id,r)),c=new Set(r.map(R=>R.id));if(n.length===0){fe(`
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
      `,R=>{if(R==="cancel"){ne();return}R==="create"&&Fn()}),(C=document.getElementById("new-gw-id"))==null||C.focus()}async function Fn(){const n=document.getElementById("new-gw-id"),r=document.getElementById("new-gw-target"),i=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),m=(n==null?void 0:n.value.trim())??"",C=(r==null?void 0:r.value)??"",R=Number.parseInt((i==null?void 0:i.value.trim())??"",10),k=ae=>{c&&(c.className="error small",c.textContent=ae)};if(!m){k("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!C){k("Pick the machine it runs on.");return}try{await Vt({id:m,placement:{targetId:C,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(R)?R:4e3,Networks:[]}})}catch(ae){k(Se(ae));return}ne(),await J()}async function jn(n,r){const i=await ze(r),c=n.textContent;n.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{s||(n.textContent=c)},1500)}function Se(n){return n instanceof Error?n.message:String(n)}function qe(n){return n instanceof Le&&n.hint?` — ${n.hint}`:""}return()=>{s=!0,z==null||z(),ne()}}const vs="local";function ys(t){let s=!1,o=!1,e="",d=null;t.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${$e()}
  `;const l=t.querySelector("#targets-body");Pe(t,(f,g)=>{V(f,g)}),v();async function v(){try{const[f,g,S]=await Promise.all([Me(),He(),_t()]);if(s)return;e=S.os,E(f,g)}catch(f){if(s)return;l.innerHTML=`<p class="error">Failed to load machines: ${a(String(f))}</p>`}}function p(){d&&E(d.targets,d.catalog)}function E(f,g){d={targets:f,catalog:g};const S=e==="linux",U=[...f].sort((J,le)=>(J.mode==="local"?-1:0)-(le.mode==="local"?-1:0)),z=U.length?`<div class="card-grid">${U.map(J=>gs(J,g,J.mode!=="local"||S,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',ee=f.some(J=>J.mode==="local");l.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${z}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${F(S,ee)}
        ${o?$s():""}
      </section>
    `;const ce=l.querySelector("#fleet-verdict");ce&&Wa(ce,qa(f,g))}function F(f,g){const S=`
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
    `,U=f?`
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
      `;return g?`<div class="card-grid card-grid-wide">${S}</div>`:`<div class="card-grid card-grid-wide">${f?U+S:S+U}</div>`}async function V(f,g){var S;if(f==="add-local"){await A();return}if(f==="delete-target"){const U=g.dataset.id;if(!U||!await De({title:"Remove machine",body:`Remove "${U}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await _(U);return}if(f==="toggle-ssh"){o=!o,I(),p(),o&&((S=t.querySelector("#ssh-host"))==null||S.focus());return}f==="add-ssh"&&await G()}async function A(){I();try{await ut({id:vs,mode:"local"}),await v()}catch(f){B(f)}}async function _(f){try{await zn(f),await v()}catch(g){B(g)}}async function G(){const f=t.querySelector("#ssh-host"),g=t.querySelector("#ssh-user"),S=t.querySelector("#ssh-key"),U=t.querySelector("#ssh-port"),z=t.querySelector("#ssh-id");if(!f||!g||!S||!U||!z)return;const ee=f.value.trim(),ce=g.value.trim(),J=S.value.trim(),le=U.value.trim(),Z=z.value.trim();if(I(),!ee||!ce||!J){B(new Error("host, user, and key path are required"));return}const j=Z||ws(ee),pe={Host:ee,User:ce,KeyPath:J};if(le){const ve=Number.parseInt(le,10);if(!Number.isFinite(ve)||ve<=0){B(new Error("port must be a positive number"));return}pe.Port=ve}const de=t.querySelector("#ssh-submit");de&&(de.disabled=!0,de.textContent="Connecting…");try{await ut({id:j,mode:"ssh",ssh:pe}),o=!1,await v()}catch(ve){B(ve),de&&(de.disabled=!1,de.textContent="Add server")}}function B(f){let g=t.querySelector("#targets-error");g||(l.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),g=t.querySelector("#targets-error")),g.textContent=String(f instanceof Error?f.message:f)}function I(){var f;(f=t.querySelector("#targets-error"))==null||f.remove()}return()=>{s=!0}}function gs(t,s,o,e){const d=t.wire,l=t.mode==="local"?"this machine":"SSH",v=t.mode==="ssh"&&t.ssh?`${a(t.ssh.User)}@${a(t.ssh.Host)}`:l;let p;if(!d&&!o)p=`${Y("can't run a node","warn")} ${Y(e||"not Linux","neutral")}`;else if(!d)p=Y("not set up","neutral");else{const E=s.networks.find(V=>V.ChainID===d.ChainID),F=E?E.Name:`chain ${d.ChainID}`;p=`${Y(F,"ok")} ${Y(d.ExecID,"neutral")} ${Y(d.BeaconID,"neutral")}${d.Archive?" "+Y("archive","warn"):""}`}return`
    <div class="card">
      <h2>${a(t.id)}</h2>
      <p class="muted">${v}</p>
      <p>${p}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(t.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(t.id)}">Remove</button>
      </div>
    </div>
  `}function $s(){return`
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
  `}function ws(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}Gn();const ks=document.querySelector("#app"),{contentEl:Cs,setActiveNav:Ss}=ga(ks);let Te=null;function xs(){const s=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(s.length===0)return{screen:"home"};const[o,e]=s;return o==="machine"||o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"||o==="analytics"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function Re(t){const s=document.createElement("div");return Cs.replaceChildren(s),t(s)}function en(){if(Te){try{Te()}catch{}Te=null}const{screen:t,id:s}=xs();switch(Ss(t),t){case"machine":if(!s){location.hash="#/targets";return}Te=Re(o=>Ba(o,s));break;case"setup":case"dash":case"logs":case"services":if(!s){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(s)}`;return;case"security":if(!s){location.hash="#/targets";return}Te=Re(o=>cs(o,s));break;case"diag":if(!s){location.hash="#/targets";return}Te=Re(o=>Sa(o,s));break;case"analytics":if(!s){location.hash="#/rpc";return}Te=Re(o=>Ca(o,s));break;case"rpc":Te=Re(o=>bs(o));break;case"settings":Te=Re(o=>us(o));break;case"targets":Te=Re(o=>ys(o));break;case"panel":Te=Re(o=>Ft(o));break;case"home":default:Te=Re(o=>Ft(o));break}}window.addEventListener("hashchange",en);en();
