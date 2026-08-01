var Gt=Object.defineProperty;var Vt=(s,i,r)=>i in s?Gt(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var Oe=(s,i,r)=>Vt(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const p of document.querySelectorAll('link[rel="modulepreload"]'))e(p);new MutationObserver(p=>{for(const g of p)if(g.type==="childList")for(const E of g.addedNodes)E.tagName==="LINK"&&E.rel==="modulepreload"&&e(E)}).observe(document,{childList:!0,subtree:!0});function r(p){const g={};return p.integrity&&(g.integrity=p.integrity),p.referrerPolicy&&(g.referrerPolicy=p.referrerPolicy),p.crossOrigin==="use-credentials"?g.credentials="include":p.crossOrigin==="anonymous"?g.credentials="omit":g.credentials="same-origin",g}function e(p){if(p.ep)return;p.ep=!0;const g=r(p);fetch(p.href,g)}})();function ut(){return J("/api/host")}function we(){return J("/api/catalog")}function ke(){return J("/api/targets")}function st(s){return J("/api/targets",{method:"POST",headers:me,body:JSON.stringify(s)})}function zt(s){return J(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Jt(s,i){return J(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function Yt(s,i){return J(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:me,body:JSON.stringify(i)})}function Xe(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function Zt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function Xt(s,i=200){return J(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function Qt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function ot(s,i){const r=i===void 0?{}:{lines:i};return J(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:me,body:JSON.stringify(r)})}function en(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function tn(s,i){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})})}function nn(s){return J(`/api/targets/${encodeURIComponent(s)}/du`)}function an(s){return J(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function sn(s){return J(`/api/targets/${encodeURIComponent(s)}/firewall`)}function on(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function rn(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function cn(s){return J(`/api/targets/${encodeURIComponent(s)}/containers`)}function ln(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function dn(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})}),e=await r.text();let p=null;try{p=e?JSON.parse(e):null}catch{}if(p&&typeof p=="object"&&"report"in p)return p;const g=p&&typeof p=="object"&&typeof p.error=="string"?p.error:r.statusText||`HTTP ${r.status}`;throw new Se(r.status,g)}function un(s,i){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function pn(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:me}),r=await i.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const p=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new Se(i.status,p)}function hn(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:me,body:JSON.stringify(r)})}function pt(){return J("/api/gateways")}async function fn(s){await J(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function mn(s){return J("/api/gateways",{method:"POST",headers:me,body:JSON.stringify(s)})}function bn(s){return J(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function gn(s){return J(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function yn(s){return J(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function vn(s,i=!1){const r=i?"?refresh=1":"";return J(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function $n(s){return J(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function wn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:me,body:JSON.stringify(i)})}function kn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function Sn(s){return J(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function Tn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:s})}),r=await i.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const p=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new Se(i.status,p)}function Cn(s){return J(`/api/chainlist/${s}`)}function xn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function Pn(){return J("/api/settings")}function En(s){return J("/api/settings",{method:"PUT",headers:me,body:JSON.stringify(s)})}class Se extends Error{constructor(r,e,p,g){super(e);Oe(this,"status");Oe(this,"hint");Oe(this,"code");this.name="ApiError",this.status=r,this.hint=p,this.code=g}}const me={"Content-Type":"application/json"};async function J(s,i){const r=await fetch(s,i);if(!r.ok){let p=r.statusText||`HTTP ${r.status}`,g,E;try{const f=await r.json();f&&typeof f.error=="string"&&f.error&&(p=f.error),f&&typeof f.hint=="string"&&f.hint&&(g=f.hint),f&&typeof f.code=="string"&&f.code&&(E=f.code)}catch{}throw new Se(r.status,p,g,E)}if(r.status===204)return;const e=await r.text();return e?JSON.parse(e):void 0}const rt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ce(s,i){const r=s&&i&&i!==rt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(rt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function In(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:p=>{const g=p==="machine"?"targets":p==="home"?"rpc":p;for(const E of r)E.classList.toggle("active",E.dataset.nav===g)}}}function ie(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function Rn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function Ln(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),e=i%60;if(r===0)return`~${e}m`;if(r<48)return`~${r}h ${e}m`;const p=Math.floor(r/24),g=r%24;return`~${p}d ${g}h`}function D(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function $e(s){return`<span class="dot dot-${s}"></span>`}const it=["B","KB","MB","GB","TB","PB"];function xe(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<it.length-1;)i/=1024,r++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${it[r]}`}async function De(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ye(s,i){s.addEventListener("click",r=>{const e=r.target.closest("[data-action]");if(!e||!s.contains(e))return;const p=e.dataset.action;p&&i(p,e,r)})}function Ye(s,i,r){const e=i.find(g=>g.value===r),p=i.map(g=>`
      <li class="dropdown-option${g.value===r?" selected":""}" role="option"
          aria-selected="${g.value===r}" data-value="${n(g.value)}">
        ${n(g.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${p}</ul>
    </div>
  `}function Ae(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function Qe(s,i){s.addEventListener("click",p=>{const g=p.target,E=g.closest(".dropdown-trigger");if(E&&s.contains(E)){const N=E.closest(".dropdown"),F=!!N&&!N.classList.contains("open");Ae(s),N&&F&&(N.classList.add("open"),E.setAttribute("aria-expanded","true"));return}const f=g.closest(".dropdown-option");if(f&&s.contains(f)){const N=f.closest(".dropdown");Ae(s),i((N==null?void 0:N.dataset.dropdown)??"",f.dataset.value??"");return}Ae(s)});const r=p=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}const g=p.target;(!g.closest(".dropdown")||!s.contains(g))&&Ae(s)},e=p=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}p.key==="Escape"&&Ae(s)};document.addEventListener("click",r),document.addEventListener("keydown",e)}const _e="app-modal";let We=null;function re(s,i){Y();const r=document.createElement("div");r.className="modal-overlay",r.id=_e,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",p=>{const g=p.target.closest("[data-modal-action]");g!=null&&g.dataset.modalAction?i(g.dataset.modalAction):p.target===r&&i("cancel")});const e=p=>{p.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),We=e,document.body.appendChild(r)}function Y(){var s;(s=document.getElementById(_e))==null||s.remove(),We&&(document.removeEventListener("keydown",We),We=null)}function He(){return document.querySelector(`#${_e} .modal`)}function Be(s){return new Promise(i=>{var p;let r=!1;const e=g=>{r||(r=!0,Y(),i(g))};re(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,g=>e(g==="confirm")),(p=document.querySelector(`#${_e} [data-modal-action="confirm"]`))==null||p.focus()})}const Ve=5e3,An=60;function Nn(s,i){let r=!1,e=null,p=null,g=null,E=null;const f=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ce()}</div>`;const N=s.querySelector("#an-body");ye(s,(v,d)=>{var k;v==="toggle-endpoint"&&((k=d.closest(".an-endpoint"))==null||k.classList.toggle("expanded"))}),F();async function F(){try{e=((await pt()).gateways??[]).find(d=>d.id===i)??null}catch(v){if(r)return;g=String(v instanceof Error?v.message:v),U();return}if(!r){if(!e){U();return}await j(),E=window.setInterval(()=>void j(),Ve)}}async function j(){try{const v=await yn(i);if(r)return;I(v),p=v,g=null}catch(v){if(r)return;g=String(v instanceof Error?v.message:v)}U()}function I(v){if(!v.enabled||v.error)return;const d=f[f.length-1];d&&d.since!==v.since&&(f.length=0);const k=new Map;for(const R of v.networks??[])k.set(R.chainId,R.received);f.push({t:Date.now(),since:v.since,received:k}),f.length>An&&f.shift()}function U(){r||(N.innerHTML=O())}function O(){return g&&!p?`<h1>Analytics</h1><p class="error">${n(g)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${L(e)}
      ${p?u(p):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(i)}”.</p>
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
    `}function T(){if(!p)return"";if(!p.enabled)return"counters off";if(p.error)return"could not be read";const v=p.since?new Date(p.since):null;return v&&!Number.isNaN(v.getTime())?`totals since the gateway started, ${n(v.toLocaleString())}<br />re-read every ${Ve/1e3}s`:`re-read every ${Ve/1e3}s`}function u(v){return v.enabled?v.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(v.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:y(v)+le(v):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function y(v){const d=v.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(k=>x(k)).join("")}
      </section>
    `}function x(v){const d=v.methods??[],k=v.endpoints??[],R=v.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${v.chainId}</span>
          <span class="band-name">${n(v.name)}</span>
          ${q(v)}
        </div>
        <div class="an-stats">
          ${B("Received",ie(v.received),"what clients asked this chain for")}
          ${B("Answered",ie(v.answered),"returned by one of your endpoints")}
          ${B("From cache",ie(v.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${B("Failed",ie(v.failed),"asked for and never answered",v.failed>0?"bad":"")}
        </div>
        ${ee(v.chainId)}
        ${R?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':z("Method",d.map(H=>({label:H.method,l:H})))+z("Endpoint",k.map(H=>({label:H.upstream,l:H})))+G(v)}
      </div>
    `}function B(v,d,k,R=""){return`
      <div class="an-stat${R?" an-stat-"+R:""}" title="${n(k)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(v)}</span>
      </div>
    `}function q(v){const d=X(v.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const k=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${k}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${k}s</span>
    </span>`}function X(v){if(f.length<2)return null;const d=f[0],k=f[f.length-1],R=(k.t-d.t)/1e3;if(R<=0)return null;const H=(k.received.get(v)??0)-(d.received.get(v)??0);return H<0?null:H/R}function ee(v){if(f.length<3)return"";const d=[];for(let $=1;$<f.length;$++){const P=f[$-1],W=f[$],c=(W.t-P.t)/1e3,b=(W.received.get(v)??0)-(P.received.get(v)??0);d.push(c>0&&b>=0?b/c:0)}const k=Math.max(...d);if(k<=0)return"";const R=240,H=28,_=d.length>1?R/(d.length-1):R,m=d.map(($,P)=>`${(P*_).toFixed(1)},${(H-$/k*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${k.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${R} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${m}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(k.toFixed(2))} req/s</span>
      </div>
    `}function G(v){const d=[];return v.cached.count>0&&d.push(`${n(ie(v.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${v.cached.mean===null?"":`, in ${n(Ne(v.cached.mean))} on average`}.`),v.failedLatency.count>0&&v.failedLatency.mean!==null&&d.push(`The ${n(ie(v.failedLatency.count))} that failed took
         ${n(Ne(v.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function z(v,d){return d.length===0?"":`
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
            ${d.map(k=>de(k.label,k.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function de(v,d){return`
      <tr>
        <td><code>${n(v)}</code></td>
        <td class="an-num">${ie(d.count)}</td>
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(Ne(d.mean))}</td>
        <td>${te(d)}</td>
      </tr>
    `}function te(v){const d=v.buckets??[];if(d.length===0||v.count===0)return'<span class="muted small">—</span>';let k=0;const R=[];for(const _ of d){const m=_.count-k;k=_.count,R.push({label:se(_.le),n:Math.max(0,m)})}return R.reduce((_,m)=>_+m.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(R.filter(_=>_.n>0).map(_=>`${_.n} ${_.label}`).join(" · "))}">
        ${R.map((_,m)=>_.n===0?"":`<span class="an-band an-band-${Math.min(m,4)}" style="flex:${_.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ae(R))}</span>
    `}function ae(v){for(let d=v.length-1;d>=0;d--)if(v[d].n>0)return`slowest ${v[d].label}`;return""}function se(v){if(v==="+Inf")return"30s or more";const d=Number(v);return Number.isFinite(d)?`under ${Ne(d)}`:`under ${v}`}function le(v){const d=v.endpoints??[];return`
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
                     <tbody>${d.map(k=>pe(k)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function pe(v){const d=v.errors??[],k=d.reduce((H,_)=>H+_.count,0),R=d.length>0;return`
      <tr class="an-endpoint${R?" expandable":""}" ${R?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(v.upstream)}</code>
          ${v.chainId?`<span class="muted small">chain ${v.chainId}</span>`:""}
          ${v.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ie(v.requests)}</td>
        <td class="an-num${k>0?" bad":""}">${k>0?ie(k):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${v.headLag>0?ie(v.headLag):'<span class="muted">0</span>'}</td>
        <td>${be(v)}</td>
      </tr>
      ${R?ge(v,d):""}
    `}function be(v){const d=[];return v.scored?(d.push(v.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(v.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(v.score.toFixed(3))}</span>`),v.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ie(v.primarySwitches)} switches</span>`),v.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Ne(v.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(v,d){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${d.map(k=>`
                    <tr>
                      <td class="an-num">${ie(k.count)}</td>
                      <td><code>${n(k.class)}</code></td>
                      <td>${k.severity?`<span class="badge badge-${k.severity==="critical"?"bad":"warn"}">${n(k.severity)}</span>`:""}</td>
                      <td class="muted small">${n(k.method||"")}</td>
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
    `}return()=>{r=!0,E!==null&&window.clearInterval(E)}}function Ne(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}function Bn(s,i){let r=!1,e=null,p=null,g=!1,E=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ce()}</div>`;const f=s.querySelector("#diag-body"),N=s.querySelector("#diag-footer");ye(s,(u,y)=>{var x;if(u==="run")j();else if(u==="toggle")(x=y.closest(".check-item"))==null||x.classList.toggle("expanded");else if(u==="copy"){const B=y.dataset.copy;B&&T(y,B)}}),F();async function F(){let u,y;try{const[B,q]=await Promise.all([ke(),we()]);u=B.find(X=>X.id===i),y=q}catch(B){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(B))}</p>`;return}if(r)return;if(!u){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const x=y==null?void 0:y.networks.find(B=>B.ChainID===u.wire.ChainID);x&&(N.innerHTML=ce(x.Name,x.LearnURL));try{e=await rn(i),E=!0}catch(B){p=String(B instanceof Error?B.message:B)}r||I()}async function j(){g=!0,p=null,I();try{e=await on(i),E=!0}catch(u){p=String(u instanceof Error?u.message:u)}g=!1,r||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${g?"disabled":""}>${g?"Running…":"Run diagnostics"}</button>
      </div>
      ${p?`<p class="error">${n(p)}</p>`:""}
      ${U()}
    `}function U(){if(!E&&!p)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const u=new Date(e.at).toLocaleString(),y=e.failedId?`<p><strong>Failed at: ${n(O(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(u)} — trigger: ${n(e.trigger)}</p>
      ${y}
      <ul class="check-list">${e.items.map(L).join("")}</ul>
    `}function O(u){var y;return((y=e==null?void 0:e.items.find(x=>x.ID===u))==null?void 0:y.Title)??u}function L(u){const y=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral",x=u.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${x?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${D(x?"failed here":u.Status,y)}
          <strong>${n(u.Title)}</strong>
          <span class="muted small check-detail-inline">${n(u.Detail)}</span>
        </button>
        <div class="check-body">
          <details${x?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(u.Why)}</p>
          </details>
          ${u.Fix?`
                <details${x?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(u.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(u.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(u,y){const x=await De(y),B=u.textContent;u.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{r||(u.textContent=B)},1500)}return()=>{r=!0}}function ht(s,i){if(s.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=s.filter(f=>!f.wire);if(r.length>0){const f=r.map(F=>F.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const e=i.networks??[],p=f=>{const N=e.find(F=>F.ChainID===f);return N?N.Name:`chain ${f}`},g=Hn(s.map(f=>p(f.wire.ChainID))),E=s.length===1?"machine":"machines";return{level:"ok",sentence:`All ${s.length} ${E} healthy — ${Dn(g)}.`,machines:[]}}function ft(s,i){const r=i.machines.length?` <span class="verdict-machines">${i.machines.map(e=>`<a href="#/setup/${encodeURIComponent(e)}">${n(e)}</a>`).join(" ")}</span>`:"";s.innerHTML=`
    <div class="verdict-line verdict-${i.level}">
      ${D(i.level==="ok"?"OK":"Attention",i.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(i.sentence)}</strong>${r}
    </div>
  `}function Hn(s){return[...new Set(s)]}function Dn(s){return s.length<=1?s[0]??"":s.length===2?`${s[0]} and ${s[1]}`:`${s.slice(0,-1).join(", ")} and ${s[s.length-1]}`}function Un(s,i){const r=i==="linux";return s.some(p=>p.mode==="ssh"||p.mode==="local"&&r)||r}function Mn(s){let i=!1;s.innerHTML='<div id="home-body"><p class="muted">Loading…</p></div>';const r=s.querySelector("#home-body");e();async function e(){let g,E,f;try{const[N,F,j]=await Promise.all([ke(),we(),ut()]);g=N,E=F,f=j.os}catch(N){if(i)return;r.innerHTML=`<p class="error">Failed to load: ${n(String(N))}</p>`;return}if(!i){if(Un(g,f)){location.hash="#/targets";return}p(g,E)}}function p(g,E){r.innerHTML=`
      <h1>valve-node-app</h1>
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your RPC endpoint</h2></div>
        <div class="card hero-card">
          <h3>Get an RPC endpoint — no node required ${D("recommended","ok")}</h3>
          <p class="muted">
            eRPC is a managed endpoint that aggregates Valve — via the shared
            <code>vk_demo</code> key — and the chain's known-set public upstreams behind one
            URL, with automatic failover between them. It runs as a container here; you never
            run, sync or babysit a node.
          </p>
          <div class="card-actions">
            <a class="btn btn-primary" href="#/rpc">Set up my endpoint →</a>
          </div>
        </div>
      </section>
      <section class="section">
        <div class="section-head"><h2>Run your own node</h2></div>
        <div class="card card-warn">
          <h3>Run your own node ${D("needs a Linux server","warn")}</h3>
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
      ${ce()}
    `;const f=r.querySelector("#fleet-verdict");f&&ft(f,ht(g,E))}return()=>{i=!0}}const On=85,ze={exec:"Execution",beacon:"Beacon"};function qn(s,i){let r=!1,e=null,p=null,g=null,E=null,f=null,N=null,F=null,j=null;const I={exec:null,beacon:null};let U=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ce()}</div>`;const O=s.querySelector("#dash-body"),L=s.querySelector("#dash-footer");O.addEventListener("click",d=>{const k=d.target.closest("[data-action]");if(!k||!O.contains(k))return;const R=k.dataset.action;if(R==="svc-action"){const H=k.dataset.svc,_=k.dataset.kind;H&&_&&pe(H,_)}else if(R==="open-clear"){const H=k.dataset.svc;H&&ge(H)}else if(R==="copy"){const H=k.dataset.copy;H&&be(k,H)}else R==="retry-du"?u():R==="retry-endpoints"&&y()}),T();async function T(){let d,k;try{const[H,_]=await Promise.all([ke(),we()]);d=H.find(m=>m.id===i),k=_}catch(H){if(r)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(r)return;if(!d){O.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const R=k==null?void 0:k.networks.find(H=>H.ChainID===d.wire.ChainID);R&&(L.innerHTML=ce(R.Name,R.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',e=Zt(i,H=>{r||(x(H),p=H,g=H,B())}),u(),y()}async function u(){N=null;try{f=await nn(i)}catch(d){f=null,N=String(d instanceof Error?d.message:d)}r||B()}async function y(){j=null;try{F=await an(i)}catch(d){F=null,j=String(d instanceof Error?d.message:d)}r||B()}function x(d){if(!p)return;const k=(new Date(d.at).getTime()-new Date(p.at).getTime())/1e3,R=d.execHead-p.execHead;if(k>0&&R>=0){const H=R/k;E=E===null?H:E*.7+H*.3}}function B(){if(!g)return;const d=g;O.innerHTML=`
      <p class="dash-status">${q(d)}</p>
      <div class="card-grid">
        ${se(d)}
        ${ee(d)}
        ${G(d)}
        ${z(d)}
        ${de(d)}
        ${te()}
      </div>
      <p class="muted small">Last updated ${n(new Date(d.at).toLocaleTimeString())}</p>
    `}function q(d){return!d.execActive&&!d.beaconActive?D("Node not running","bad"):d.execSyncing||d.beaconDistance>0?D("Syncing","warn"):D("Running · synced","ok")}function X(d){const R=d.refHead>0?d.refHead-d.execHead:null,H=R!==null&&R>0&&E&&E>0?Ln(R/E):R!==null&&R<=0?"caught up":"—";return{lag:R,eta:H}}function ee(d){const{lag:k,eta:R}=X(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?D("syncing","warn"):d.execHead===0?D("no data","neutral"):D("synced","ok"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ie(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${k!==null?ie(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${k!==null?ie(Math.max(k,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${R}</dd></div>
        </dl>
      </div>
    `}function G(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?D("no data","neutral"):d.beaconDistance===0?D("synced","ok"):D("syncing","warn"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ie(d.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ie(d.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function z(d){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ie(d.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ie(d.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(d){const k=d.diskUsedPct>=On,R=`
      <div class="meter"><div class="meter-fill ${k?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${Rn(d.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="error small">${n(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="muted">Loading…</p>
        </div>
      `;const H=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,_=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:m,eta:$}=X(d),P=m!==null&&m>0&&E!==null&&E>0;return`
      <div class="card ${k?"card-warn":""}">
        <h3>Storage</h3>
        ${R}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${xe(f.ExecBytes)} of ~${xe(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${P?`<p class="muted small">Estimated time remaining: ${n($)}</p>`:""}
        <p class="muted small">Beacon — ${xe(f.BeaconBytes)} of ~${xe(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${_}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${xe(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function te(){if(j)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(j)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!F)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=F,k=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",R=d.Access==="ssh"?`
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
        ${k}
        ${R}
      </div>
    `}function ae(d,k){const R=ze[d],H=I[d],_=(m,$,P)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${m}" ${H!==null||P?"disabled":""}>${H===m?le():n($)}</button>`;return`
      <div class="service-row">
        <span>${n(R)} ${k?D("active","ok"):D("down","bad")}</span>
        <div class="service-actions">
          ${_("start","Start",k)}
          ${_("stop","Stop",!k)}
          ${_("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${d}" ${H!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function se(d){return`
      <div class="card">
        <h3>Services</h3>
        ${ae("exec",d.execActive)}
        ${ae("beacon",d.beaconActive)}
        ${U?`<p class="error small">${n(U)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function le(){return'<span class="spinner" aria-label="working"></span>'}async function pe(d,k){if(I[d]===null){I[d]=k,U=null,B();try{await en(i,d,k)}catch(R){U=`${ze[d]} ${k} failed: ${R instanceof Error?R.message:String(R)}`}I[d]=null,r||B()}}async function be(d,k){const R=await De(k),H=d.textContent;d.textContent=R?"Copied!":"Copy failed",setTimeout(()=>{r||(d.textContent=H)},1500)}function ge(d){const k=ze[d],R=f?xe(d==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";re(`
        <h2>Clear ${n(k)} data</h2>
        <p class="error">
          This stops the ${n(k.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(R)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,m=>{if(m==="cancel"){Y();return}m==="confirm"&&v(d)});const H=document.getElementById("clear-confirm-input"),_=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{_&&(_.disabled=H.value.trim()!==d)}),H==null||H.focus()}async function v(d){const k=document.getElementById("clear-confirm-btn");k&&(k.disabled=!0,k.textContent="Clearing…");try{await tn(i,d),Y(),u()}catch(R){const H=He();if(H){const _=document.createElement("p");_.className="error small",_.textContent=`Clear failed: ${R instanceof Error?R.message:String(R)}`,H.appendChild(_)}k&&(k.disabled=!1,k.textContent="Clear and resync")}}return()=>{r=!0,e==null||e(),Y()}}const ct=500,lt="valve-node-app.explain-consent";function Fn(s,i){let r=!1,e=null;const p=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ce()}</div>
  `;const g=s.querySelector("#logs-body"),E=s.querySelector("#logs-footer");ye(s,T=>{T==="explain"&&j()}),f();async function f(){let T,u;try{const[x,B]=await Promise.all([ke(),we()]);T=x.find(q=>q.id===i),u=B}catch(x){if(r)return;g.innerHTML=`<p class="error">Failed to load target: ${n(String(x))}</p>`;return}if(r)return;if(!T){g.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!T.wire){g.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const y=u==null?void 0:u.networks.find(x=>x.ChainID===T.wire.ChainID);y&&(E.innerHTML=ce(y.Name,y.LearnURL));try{const x=await Xt(i,200);if(r)return;p.push(...x)}catch(x){if(r)return;g.innerHTML=`<p class="error">Failed to load logs: ${n(String(x))}</p>`;return}N(),e=Qt(i,x=>{r||(p.push(x),p.length>ct&&p.splice(0,p.length-ct),N())})}function N(){const T=p.filter(y=>y.severity==="error"||y.severity==="critical");g.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${p.map(F).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${D(String(T.length),T.length?"bad":"neutral")}</h2>
          <div class="log-lines">${T.length?T.slice().reverse().map(F).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=g.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function F(T){const u=T.severity||"info",y=T.learnUrl?` <a href="${n(T.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(u)}">
        <span class="log-time">${n(new Date(T.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(T.unit)}</span>
        <span class="log-sev">${n(u)}</span>
        <span class="log-text">${n(T.line)}</span>
        ${T.explain?`<div class="log-explain">${n(T.explain)}${y}</div>`:""}
      </div>
    `}async function j(){const T=p.filter(y=>y.severity==="error"||y.severity==="critical").map(y=>y.line).slice(-40);if(!(localStorage.getItem(lt)==="1")){I(T);return}await U(T)}function I(T){const u=T.length?`<pre class="explain-excerpt">${T.map(y=>n(y)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';O(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${u}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,y=>{y==="proceed"?(localStorage.setItem(lt,"1"),L(),U(T)):L()})}async function U(T){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=T.length?await ot(i,T):await ot(i);if(r)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map(y=>n(y)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,y=>{y==="close"&&L()})}catch(u){if(r)return;if(u instanceof Se&&u.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,y=>{y==="close"&&L()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,y=>{y==="close"&&L()})}}function O(T,u){L();const y=document.createElement("div");y.className="modal-overlay",y.id="explain-modal",y.innerHTML=`<div class="modal">${T}</div>`,y.addEventListener("click",x=>{const B=x.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&u(B.dataset.modalAction),x.target===y&&u("cancel")}),document.body.appendChild(y)}function L(){var T;(T=document.getElementById("explain-modal"))==null||T.remove()}return()=>{r=!0,e==null||e(),L()}}const jn="run",Wn={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},_n={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Kn(s,i){let r=!1,e=null,p=null;const g={devnet:null},E={devnet:null},f={devnet:[]};let N=null;const F={devnet:!1};let j=null;const I={devnet:null},U={devnet:null};s.innerHTML=`
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
    ${ce()}
  `;const O=s.querySelector("#services-body");ye(s,(c,b)=>{ge(c,b)}),L();async function L(){try{const c=await cn(i);if(r)return;e=c,p=null}catch(c){if(r)return;e=null,p=P(c)}u()}function T(c){return e==null?void 0:e.services.find(b=>b.id===c)}function u(){if(!r){if(p){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(p)}</p>`;return}if(!e){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${y(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(x).join("")}
      </div>
    `}}function y(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const b=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(b)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function x(c){const b=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${B(c)}
        </div>
        <p class="muted small">${n(Wn[c.id]??"")}</p>

        ${c.error?q(c):""}
        ${c.blocked?`<div class="banner banner-warn">${n(c.blocked)}</div>`:""}
        ${b.map(A=>`<div class="banner banner-warn">${n(A)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${n(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${X(c)}

        ${ee(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(A=>G(c,A)).join("")}
        </div>
        ${E[c.id]?`<p class="error small">${n(E[c.id])}</p>`:""}
        ${z(c)}

        ${de(c)}
      </div>
    `}function B(c){switch(c.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function q(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function X(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const b=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${b}.</p>`}function ee(c){const b=c.endpoints??[];return b.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":b.map(A=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(A.label)}</span>
          <code class="endpoint-url">${n(A.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(A.url)}">Copy</button>
        </div>`).join("")}function G(c,b){const A=_n[b];if(!A)return"";const K=g[c.id],Z=b==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:A.label;return`
      <button class="${A.className}" data-action="svc-${b}" data-svc="${n(c.id)}"
              title="${n(A.title)}" ${K?"disabled":""}>
        ${K===b?'<span class="spinner" aria-label="working"></span>':n(Z)}
      </button>
    `}function z(c){const b=f[c.id]??[];return b.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(b.join(`
`))}</pre>
      </div>
    `}function de(c){const b=F[c.id],A=te(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(c.id)}">
            ${b?"Close":"Edit"}
          </button>
        </div>
        ${b?ae():`<p class="small">${A}</p>`}
        ${I[c.id]?`<p class="error small">${n(I[c.id])}</p>`:""}
        ${U[c.id]?`<p class="muted small">${n(U[c.id])}</p>`:""}
      </div>
    `}function te(c){const b=c.devnet;return b?`Chain ${b.ChainID} · a block every ${n(b.BlockTime)} · JSON-RPC on ${n(b.BindAddr)}:${b.HTTPPort} · WebSocket on ${n(b.BindAddr)}:${b.WSPort}`:"—"}function ae(c){return se()}function se(){const c=j;return c?`
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
    `:""}function le(){F.devnet&&j&&(j.BlockTime=pe("#dev-blocktime",j.BlockTime),j.HTTPPort=be("#dev-http",j.HTTPPort),j.WSPort=be("#dev-ws",j.WSPort),j.BindAddr=pe("#dev-bind",j.BindAddr))}function pe(c,b){const A=s.querySelector(c);return A?A.value.trim():b}function be(c,b){const A=s.querySelector(c);if(!A)return b;const K=Number.parseInt(A.value.trim(),10);return Number.isFinite(K)?K:b}async function ge(c,b){const A=b.dataset.svc??"";switch(c){case"refresh":await L();return;case"copy":b.dataset.copy&&await $(b,b.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await v(A,c.slice(4));return;case"svc-create":case"svc-recreate":await d(A);return;case"svc-wipe":H(A);return;case"toggle-config":k(A);return;case"save-config":await R(A);return;default:return}}async function v(c,b){if(!g[c]){g[c]=b,E[c]=null,u();try{await ln(i,c,b)}catch(A){E[c]=`${b} failed: ${P(A)}${W(A)}`}g[c]=null,await L()}}async function d(c){if(!g[c]){g[c]="create",E[c]=null,f[c]=["starting…"],u();try{await un(i,c)}catch(b){E[c]=`${P(b)}${W(b)}`,f[c]=[],g[c]=null,u();return}N==null||N(),N=Xe(i,b=>{if(r)return;const A=b.err?`${b.stepId}: ${b.err}`:b.line?`${b.stepId}: ${b.line}`:`${b.stepId}: done`;if(f[c]=[...(f[c]??[]).filter(Z=>Z!=="starting…"),A],!!b.err||b.stepId===jn&&!!b.done){N==null||N(),N=null,g[c]=null,b.err&&(E[c]="Provisioning failed — see the log below."),L();return}u()})}}function k(c){if(le(),F[c]=!F[c],I[c]=null,U[c]=null,F[c]){const b=T(c);b!=null&&b.devnet&&(j={...b.devnet})}u()}async function R(c){var K;le(),I[c]=null,U[c]=null;const b=j;if(!b)return;if(b.HTTPPort===b.WSPort){I[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await hn(i,c,b)}catch(Z){I[c]=P(Z),u();return}const A=((K=T(c))==null?void 0:K.status.State)==="running";F[c]=!1,U[c]=A?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function H(c){const b=T(c);if(!b)return;const A=(b.restartsOnWipe??[]).map(M=>{var oe;return((oe=T(M))==null?void 0:oe.label)??M});re(`
        <h2>Wipe ${n(b.label)}</h2>
        <p class="error">This deletes ${n(b.wipeDiscards)}</p>
        ${A.length?`<p>It also restarts what sits in front of it: ${n(A.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(c)}</button>
        </div>
      `,M=>{if(M==="cancel"||M==="close"){Y(),L();return}M==="confirm"&&_(c)});const K=document.getElementById("wipe-confirm-input"),Z=document.getElementById("wipe-confirm-btn");K==null||K.addEventListener("input",()=>{Z&&(Z.disabled=K.value.trim()!==c)}),K==null||K.focus()}async function _(c){const b=document.getElementById("wipe-confirm-btn");b&&(b.disabled=!0,b.textContent="Wiping…");let A;try{A=await dn(i,c)}catch(K){const Z=He();if(Z){const M=document.createElement("p");M.className="error small",M.textContent=`Wipe failed: ${P(K)}${W(K)}`,Z.appendChild(M)}b&&(b.disabled=!1,b.textContent=`Wipe ${c}`);return}m(c,A)}function m(c,b){const A=T(c),K=ne=>{var Pe;return((Pe=T(ne))==null?void 0:Pe.label)??ne},Z=[];Z.push(b.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ne of b.report.VolumesRemoved??[])Z.push(`Volume ${ne} deleted.`);for(const ne of b.report.VolumesAbsent??[])Z.push(`Volume ${ne} was already gone.`);b.report.Recreated&&Z.push("Container re-created from your saved configuration.");const M=(b.report.Cascaded??[]).map(K),oe=(b.report.CascadeSkipped??[]).map(K);re(`
        <h2>${n((A==null?void 0:A.label)??c)} wiped</h2>
        <ul class="plain-list">${Z.map(ne=>`<li>${n(ne)}</li>`).join("")}</ul>
        ${M.length?`<p class="ok">Restarted in front of it: ${n(M.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${oe.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(oe.join(", "))}.</p>`:""}
        ${b.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(b.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ne=>{(ne==="close"||ne==="cancel")&&(Y(),L())})}async function $(c,b){const A=await De(b),K=c.textContent;c.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{r||(c.textContent=K)},1500)}function P(c){return c instanceof Error?c.message:String(c)}function W(c){return c instanceof Se&&c.hint?` — ${c.hint}`:""}return()=>{r=!0,N==null||N(),Y()}}const Je=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],qe=8545,Fe=5052,je=30303,Gn=[369,943,1],dt={369:"default",943:"practise here first"};function Vn(s,i){let r=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ce()}</div>`;const p=s.querySelector("#wizard-body"),g=s.querySelector("#wizard-footer");ye(s,(m,$)=>{be(m,$)}),Qe(s,(m,$)=>{m==="exec-select"?e.execId=$:m==="beacon-select"&&(e.beaconId=$),f()}),s.addEventListener("change",m=>{const $=m.target;$ instanceof HTMLInputElement&&($.id==="data-dir-input"?(ge(),G()):$.id==="checkpoint-toggle"?(e.checkpoint=$.checked,f()):$.id==="exec-snapshot-toggle"&&(e.execSnapshot=$.checked,f()))}),E();async function E(){try{const[m,$]=await Promise.all([we(),ke()]);if(r)return;e.catalog=m;const P=$.find(W=>W.id===i);P!=null&&P.wire&&(e.chainId=P.wire.ChainID,e.execId=P.wire.ExecID,e.beaconId=P.wire.BeaconID,e.archive=P.wire.Archive,P.wire.ExecHTTPPort&&(e.execHTTPPort=String(P.wire.ExecHTTPPort)),P.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(P.wire.BeaconHTTPPort)),P.wire.ExecP2PPort&&(e.execP2PPort=String(P.wire.ExecP2PPort)),P.wire.RPCBindAddr&&(e.rpcBindAddr=P.wire.RPCBindAddr)),f()}catch(m){if(r)return;e.loadError=String(m instanceof Error?m.message:m),f()}}function f(){if(e.loadError){p.innerHTML=`<p class="error">Failed to load: ${n(e.loadError)}</p>`;return}e.catalog&&(p.innerHTML=`
      ${_(e.step)}
      ${F()}
    `,N())}function N(){var $;const m=($=e.catalog)==null?void 0:$.networks.find(P=>P.ChainID===e.chainId);g.innerHTML=m?ce(m.Name,m.LearnURL):ce()}function F(){switch(e.step){case"network":return j();case"clients":return I();case"mode":return se();case"review":return le();case"run":return pe()}}function j(){const m=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Gn.map(P=>{const W=m.networks.find(A=>A.ChainID===P);if(!W)return"";const c=e.chainId===P,b=dt[P]?D(dt[P],P===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${P}" type="button">
          <h3>${n(W.Name)} <span class="muted">(chain ${P})</span></h3>
          ${b}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function I(){const m=e.catalog,$=m.networks.find(c=>c.ChainID===e.chainId);if(!$)return'<p class="error">Unknown network.</p>';(e.execId===null||!$.ExecClients.includes(e.execId))&&(e.execId=$.ExecClients[0]??null),(e.beaconId===null||!$.BeaconClients.includes(e.beaconId))&&(e.beaconId=$.BeaconClients[0]??null);const P=$.ExecClients.map(c=>de(c,m)),W=$.BeaconClients.map(c=>de(c,m));return`
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
          ${Ye("exec-select",P,e.execId)}
        </label>
        ${ae(e.execId,m)}
        <label>
          Beacon client
          ${Ye("beacon-select",W,e.beaconId)}
        </label>
        ${ae(e.beaconId,m)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function U(m){return m<=0?"—":m>=1?`~${m.toFixed(1)} TB`:`~${Math.round(m*1e3)} GB`}const O=1.1,L=.5,T="Valve reth snapshot",u="rough estimate";function y(m){return m.SnapshotSizeTB}function x(m){return m.SnapshotSizeTB*L}function B(m){return`<p class="muted small">${U(y(m))} is the measured size of Valve's reth snapshot for ${n(m.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function q(m){return{archive:y(m)*1e12*O,full:x(m)*1e12*O}}function X(m,$){if(!m)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${n($)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${n($)}</code>: ${n(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==$)return"";const P=q(m),W=e.freeBytes>=P.archive,c=e.freeBytes>=P.full,b=`<p class="muted small">Free at <code>${n($)}</code>: <strong>${xe(e.freeBytes)}</strong> — archive ${W?"fits":"won't fit"} (${U(y(m))}, ${T}), full ${c?"fits":"won't fit"} (${U(x(m))}, ${u}).</p>`;let A="";return e.downgradeNote?A=`<p class="banner banner-warn">${n(e.downgradeNote)}</p>`:c||(A=`<p class="banner banner-warn">Neither full (${U(x(m))}, ${u}) nor archive (${U(y(m))}, ${T}) fits the free space here — choose a location with more room.</p>`),b+A}function ee(m,$){if(e.downgradeNote=null,!m||e.freeBytes===null)return;const P=q(m);e.archive&&e.freeBytes<P.archive&&e.freeBytes>=P.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${$} for archive (${U(y(m))}, ${T}) — switched to Full (${U(x(m))}, ${u}). Pick a location with more room to run archive.`)}async function G(){var P;if(e.chainId===null)return;const m=(P=e.catalog)==null?void 0:P.networks.find(W=>W.ChainID===e.chainId),$=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,f();try{const{freeBytes:W}=await Jt(e.targetId,$);if(r)return;e.freeBytes=W,e.probedPath=$,ee(m,$)}catch(W){if(r)return;e.freeBytes=null,e.probedPath=$,e.diskError=String(W instanceof Error?W.message:W)}e.diskProbing=!1,f()}function z(m){return m?/^https?:\/\/.+/i.test(m)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(m,$){const P=$.clients.find(W=>W.id===m);return{value:m,label:P?`${P.id} — ${te(P.repo)}`:m}}function te(m){const $=m.split("/");return $.length>=4?$[3]:m}function ae(m,$){const P=m?$.clients.find(c=>c.id===m):void 0;if(!P)return"";const W=P.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(P.repo)}" target="_blank" rel="noopener noreferrer">${n(W)}</a></p>`}function se(){var K,Z,M;const m=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",$=(K=e.catalog)==null?void 0:K.networks.find(oe=>oe.ChainID===e.chainId),P=((M=(Z=e.catalog)==null?void 0:Z.clients.find(oe=>oe.id===e.execId))==null?void 0:M.snapshotSupported)??!1,W=$?`${U(x($))} (${u})`:"Smaller",c=$?`${U(y($))} (${T})`:"Much larger",b=$?` on ${n($.Name)}`:"",A=$?e.checkpoint?$.SyncLabel:$.GenesisSyncLabel:"";return`
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
          ${$?`<p class="sync-estimate">⏱ Estimated initial sync${b}: <strong>${n(A)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n(($==null?void 0:$.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n(($==null?void 0:$.CheckpointURL)??"")}" value="${n(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${n(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${P?`
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
              <tr><th>Approx. disk footprint${b}</th><td class="yes">${W}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${$?B($):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${$?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${W}${$?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(m)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(m)}" value="${n(e.dataDir)}" />
          </label>
          ${X($,e.dataDir||m)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(m)}/jwt.hex" value="${n(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${qe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${qe}" value="${n(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${n(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Fe})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Fe}" value="${n(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${n(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${je})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${je}" value="${n(e.execP2PPort)}" />
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
    `}function le(){const $=e.catalog.networks.find(ne=>ne.ChainID===e.chainId),P=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,W=e.jwtPath||`${P}/jwt.hex`,c=Je.map(ne=>`<li>${n(ne.title)}</li>`).join(""),b=R(e.execHTTPPort,qe),A=R(e.beaconHTTPPort,Fe),K=R(e.execP2PPort,je),Z=b||A||K?`<tr><th>Non-default ports</th><td>${[b?`exec HTTP ${b}`:null,A?`beacon HTTP ${A}`:null,K?`exec p2p ${K}`:null].filter(ne=>ne!==null).map(n).join(", ")}</td></tr>`:"",{addr:M}=v(e.rpcBindAddr),oe=M?`<tr><th>RPC bind address</th><td><code>${n(M)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${n(($==null?void 0:$.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(P)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(W)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${n(e.checkpointUrl||($==null?void 0:$.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${Z}
            ${oe}
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
    `}function pe(){const $=e.catalog.networks.find(M=>M.ChainID===e.chainId),P=$==null?void 0:$.LearnURL,W=new Set(e.events.filter(M=>M.done).map(M=>M.stepId)),c=new Set(e.events.filter(M=>M.err).map(M=>M.stepId)),b=new Map;for(const M of e.events){if(!M.line)continue;const oe=b.get(M.stepId)??[];oe.push(M.line),b.set(M.stepId,oe)}const A=Je.map(M=>{var Me;const oe=W.has(M.id),ne=c.has(M.id),Pe=ne?D("failed","bad"):oe?D("done","ok"):D("pending","neutral"),Re=(b.get(M.id)??[]).slice(-5),Ue=(Me=e.events.find(Ee=>Ee.stepId===M.id&&Ee.err))==null?void 0:Me.err,Ke=M.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${P?` <a href="${n(P)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${oe?"step-done":""} ${ne?"step-error":""}">
          <div class="step-head">${Pe} <strong>${n(M.title)}</strong></div>
          ${Ke}
          ${Re.length?`<pre class="step-log">${Re.map(Ee=>n(Ee)).join(`
`)}</pre>`:""}
          ${Ue?`<p class="error small">${n(Ue)}</p>`:""}
        </li>
      `}).join(""),K=e.events.some(M=>M.err),Z=Je.every(M=>W.has(M.id))||e.events.some(M=>M.stepId==="handshake"&&M.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${A}</ol>
        ${Z&&!K?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        ${K?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(m,$){switch(m){case"pick-network":e.chainId=Number($.dataset.chainId),e.execId=null,e.beaconId=null,f();break;case"goto-network":e.step="network",f();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",f();break;case"goto-mode":e.step="mode",f(),G();break;case"goto-review":if(ge(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){f();break}e.step="review",f();break;case"start-setup":H();break}}function ge(){const m=s.querySelectorAll('input[name="mode"]');for(const M of Array.from(m))M.checked&&(e.archive=M.value==="archive");const $=s.querySelector("#data-dir-input"),P=s.querySelector("#jwt-path-input");$&&(e.dataDir=$.value.trim()),P&&(e.jwtPath=P.value.trim());const W=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),b=s.querySelector("#exec-p2p-port-input");W&&(e.execHTTPPort=W.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),b&&(e.execP2PPort=b.value.trim());const A=s.querySelector("#rpc-bind-addr-input");A&&(e.rpcBindAddr=A.value.trim());const K=s.querySelector("#checkpoint-url-input");K&&(e.checkpointUrl=K.value.trim());const Z=s.querySelector("#snapshot-key-input");Z&&(e.snapshotKey=Z.value.trim()),e.execHTTPPortError=k(e.execHTTPPort).error??null,e.beaconHTTPPortError=k(e.beaconHTTPPort).error??null,e.execP2PPortError=k(e.execP2PPort).error??null,e.rpcBindAddrError=v(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?z(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function v(m){if(!m)return{};const $=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(m);return $?$.slice(1).every(P=>Number(P)<=255)?{addr:m}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(m)&&m.includes(":")?{addr:m}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function k(m){if(!m)return{};if(!d.test(m))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const $=Number(m);return!Number.isInteger($)||$<1||$>65535?{error:"Port must be between 1 and 65535."}:{port:$}}function R(m,$){const{port:P}=k(m);if(!(P===void 0||P===$))return P}async function H(){var b;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(b=e.streamStop)==null||b.call(e),e.streamStop=null,f();const m={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(m.DataDir=e.dataDir),e.jwtPath&&(m.JWTPath=e.jwtPath);const $=R(e.execHTTPPort,qe),P=R(e.beaconHTTPPort,Fe),W=R(e.execP2PPort,je);$!==void 0&&(m.ExecHTTPPort=$),P!==void 0&&(m.BeaconHTTPPort=P),W!==void 0&&(m.ExecP2PPort=W);const{addr:c}=v(e.rpcBindAddr);c!==void 0&&(m.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(m.CheckpointURL=e.checkpointUrl):m.NoCheckpoint=!0,e.execSnapshot&&(m.ExecSnapshot=!0,m.SnapshotKey=e.snapshotKey);try{await Yt(e.targetId,m)}catch(A){if(!(A instanceof Se&&A.status===409)){e.starting=!1,e.startError=String(A instanceof Error?A.message:A),f();return}}e.starting=!1,e.step="run",f(),e.streamStop=Xe(e.targetId,A=>{r||(e.events.push(A),e.step==="run"&&f())})}function _(m){const $=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],W=$.map(c=>c.id).indexOf(m);return`
      <ol class="wizard-progress">
        ${$.map((c,b)=>`<li class="${b===W?"current":b<W?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var m;r=!0,(m=e.streamStop)==null||m.call(e)}}function zn(s,i){let r=!1;const e=new Map;s.innerHTML=`<h1>${n(i)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const p=s.querySelector("#machine-body");ye(s,(I,U)=>{I==="toggle-section"&&F(U.dataset.section??"")}),g();async function g(){let I,U;try{const[O,L]=await Promise.all([ke(),we()]);I=O.find(T=>T.id===i),U=L}catch(O){if(r)return;p.innerHTML=`<p class="error">Failed to load machine: ${n(String(O))}</p>`;return}if(!r){if(!I){location.hash="#/targets";return}E(I,U)}}function E(I,U){const O=I.mode==="local"?"this machine":"SSH",L=I.mode==="ssh"&&I.ssh?`${n(I.ssh.User)}@${n(I.ssh.Host)}`:O;p.innerHTML=`
      <p class="muted">${L}</p>
      <p>${f(I,U)}</p>
      <div class="machine-sections">
        ${j.map(T=>N(T,I,U)).join("")}
      </div>
      ${ce()}
    `}function f(I,U){const O=I.wire;if(!O)return D("not set up","neutral");const L=U.networks.find(u=>u.ChainID===O.ChainID),T=L?L.Name:`chain ${O.ChainID}`;return`${D(T,"ok")} ${D(O.ExecID,"neutral")} ${D(O.BeaconID,"neutral")}${O.Archive?" "+D("archive","warn"):""}`}function N(I,U,O){return`
      <section class="card machine-section" data-section-card="${n(I.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(I.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(I.title)}</span>
          <span class="machine-section-status">${I.status(U,O)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(I.key)}" hidden></div>
      </section>
    `}function F(I){const U=j.find(y=>y.key===I);if(!U)return;const O=s.querySelector(`[data-section-card="${I}"]`),L=s.querySelector(`[data-section-body="${I}"]`),T=s.querySelector(`.machine-section-head[data-section="${I}"]`);if(!O||!L||!T)return;const u=L.hidden;if(u&&!e.has(I)){const y=document.createElement("div");L.appendChild(y),e.set(I,U.mount(y))}L.hidden=!u,O.classList.toggle("open",u),T.setAttribute("aria-expanded",String(u))}const j=[{key:"setup",title:"Setup",status:I=>I.wire?D("set up","ok"):D("not set up","neutral"),mount:I=>Vn(I,i)},{key:"dashboard",title:"Dashboard",status:I=>I.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>qn(I,i)},{key:"logs",title:"Logs",status:I=>I.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>Fn(I,i)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:I=>Kn(I,i)}];return()=>{r=!0;for(const I of e.values())try{I()}catch{}e.clear()}}function Jn(s,i){let r=!1,e=[],p=null,g=!1,E=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ce()}</div>`;const f=s.querySelector("#sec-body"),N=s.querySelector("#sec-footer");ye(s,(L,T)=>{var u;if(L==="rerun")j();else if(L==="toggle")(u=T.closest(".check-item"))==null||u.classList.toggle("expanded");else if(L==="copy"){const y=T.dataset.copy;y&&O(T,y)}}),F();async function F(){let L,T;try{const[y,x]=await Promise.all([ke(),we()]);L=y.find(B=>B.id===i),T=x}catch(y){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(y))}</p>`;return}if(r)return;if(!L){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=T==null?void 0:T.networks.find(y=>y.ChainID===L.wire.ChainID);u&&(N.innerHTML=ce(u.Name,u.LearnURL)),await j()}async function j(){g=!0,p=null,I();try{e=await sn(i),E=!0}catch(L){p=String(L instanceof Error?L.message:L)}g=!1,r||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${g?"disabled":""}>${g?"Re-running…":"Re-run checks"}</button>
      </div>
      ${p?`<p class="error">${n(p)}</p>`:""}
      ${!E&&g?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(U).join("")}</ul>`:E?'<p class="muted">No checks returned.</p>':""}
    `}function U(L){const T=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${D(L.Status,T)}
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
    `}async function O(L,T){const u=await De(T),y=L.textContent;L.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{r||(L.textContent=y)},1500)}return()=>{r=!0}}const Yn=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],Ze="VALVE_API_KEY";function Zn(s){return s===Ze?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(s)}}</code> slot wherever an endpoint URL carries one.`}function Xn(s){let i=!1,r=!1,e=!1,p=null,g=!1,E=null,f=null;const N=new Set,F=new Map;let j="",I="";s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ce()}`;const U=s.querySelector("#settings-body");ye(s,(x,B)=>{if(x==="save"&&y(),x==="clear-key"){if(!E)return;r=!0;const q=s.querySelector("#ai-key");q&&(q.value=""),u(E)}if(x==="clear-provider-key"){const q=B.dataset.key;if(!E||!q)return;N.add(q),F.set(q,""),g=!1,u(E)}}),Qe(s,(x,B)=>{x!=="ai-provider"||!E||(f=B,g=!1,u(E))}),O();async function O(){try{const x=await Pn();if(i)return;E=x,u(x)}catch(x){if(i)return;U.innerHTML=`<p class="error">Failed to load settings: ${n(String(x))}</p>`}}function L(x){const q=(Array.isArray(x.providerKeysSet)?x.providerKeysSet:[]).filter(X=>X!==Ze).sort();return[Ze,...q]}function T(x,B){const q=n(x);return`
      <div class="pk-row">
        <label>
          <code>${q}</code>
          <input class="provider-key" data-key="${q}" type="password" autocomplete="off"
                 placeholder="${B?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Zn(x)}</p>
        ${B?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${q}">Clear saved key</button>`:""}
      </div>`}function u(x){var de;const B=f??x.aiProvider,q=Array.isArray(x.providerKeysSet)?x.providerKeysSet:[],X=L(x).map(te=>T(te,q.includes(te))).join("");U.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ye("ai-provider",Yn.map(te=>({value:te.value,label:te.label})),B)}
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
        ${p?`<p class="error">${n(p)}</p>`:""}
        ${g?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const ee=s.querySelector("#ai-key");ee==null||ee.addEventListener("input",()=>{r=!0,g=!1}),(de=s.querySelector("#ref-rpc-base"))==null||de.addEventListener("input",()=>{g=!1}),s.querySelectorAll("input.provider-key").forEach(te=>{const ae=te.dataset.key;if(!ae)return;const se=F.get(ae);se!==void 0&&(te.value=se),te.addEventListener("input",()=>{N.add(ae),F.set(ae,te.value),g=!1})});const G=s.querySelector("#pk-new-value");G&&(G.value=I),G==null||G.addEventListener("input",()=>{I=G.value,g=!1});const z=s.querySelector("#pk-new-name");z==null||z.addEventListener("input",()=>{j=z.value,g=!1})}async function y(){const x=s.querySelector("#ai-key"),B=s.querySelector("#ref-rpc-base");if(!x||!B||!E)return;const q={aiProvider:f??E.aiProvider,refRpcBase:B.value.trim()};r&&(q.aiKey=x.value);const X={};for(const G of N)X[G]=F.get(G)??"";const ee=j.trim();ee&&(X[ee]=I),Object.keys(X).length>0&&(q.providerKeys=X),e=!0,p=null,g=!1,u(E);try{const G=await En(q);if(i)return;E=G,r=!1,N.clear(),F.clear(),j="",I="",e=!1,g=!0,u(G)}catch(G){if(i)return;e=!1,p=String(G instanceof Error?G.message:G),u(E)}}return()=>{i=!0}}const Qn=["http","ws","archive","trace"],ea={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},ta="run",na={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function aa(s){let i=!1,r=null,e=null;const p={},g={},E={},f={},N={},F={},j={},I={},U={},O={},L={};let T=null;s.innerHTML=`
    <div class="page-head">
      <h1>RPC</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      A machine runs one gateway, and that gateway fronts as many chains as you list.
      Each chain below leads with how many ways it can still be answered: the filled
      segments are the endpoints it has, the hollow ones are endpoints it could have.
    </p>
    <div id="rpc-body"><p class="muted">Loading…</p></div>
    ${ce()}
  `;const u=s.querySelector("#rpc-body");ye(s,(t,a)=>{vt(t,a)}),Qe(s,()=>{}),y();async function y(){try{const t=await pt();if(i)return;r=t,e=null}catch(t){if(i)return;r=null,e=he(t)}z();for(const t of(r==null?void 0:r.gateways)??[])x(t.id),B(t.id,!1)}async function x(t){try{const a=await gn(t);if(i)return;p[t]=a}catch{if(i)return;p[t]=null}z()}async function B(t,a){E[t]=a,a&&z();try{const o=await vn(t,a);if(i)return;g[t]=o}catch{if(i)return;g[t]=null}E[t]=!1,z()}function q(t){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===t)}function X(t,a){return(t.networks??[]).find(o=>o.chainId===a)}function ee(t,a,o){var h;const l=(((h=p[t])==null?void 0:h.networks)??[]).find(S=>S.chainId===a);return((l==null?void 0:l.upstreams)??[]).find(S=>S.upstream===o)}function G(t,a,o){var l;return(((l=g[t])==null?void 0:l.endpoints)??[]).find(h=>h.chainId===a&&h.upstream===o)}function z(){if(i)return;if(e){u.innerHTML=`<p class="error">Could not read the gateways: ${n(e)}</p>`;return}if(!r){u.innerHTML='<p class="muted">Loading…</p>';return}const t=r.gateways??[],a=t.length>1,o=(r.targets??[]).some(l=>at(l.id,t));u.innerHTML=`
      ${(r.orphans??[]).map(de).join("")}
      ${t.map(l=>ae(l,a)).join("")}
      ${t.length===0?te():""}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${t.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${t.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function de(t){const a=`docker rm -f ${t.containerName}`,o=L[t.containerName];return`
      <div class="strip">
        ${pe({tone:"warn",text:`${t.containerName} is still running on ${t.targetId}. Its chains were folded into ${t.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:a})}
        ${o?pe({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(t.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function te(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function ae(t,a){return`
      ${a?`<h2 class="rpc-machine">${n(t.placement.targetId)}</h2>`:""}
      ${se(t)}
      ${le(t)}
      ${k(t)}
      ${j[t.id]?Ue(t):""}
      ${R(t)}
    `}function se(t){var h;const a=t.status.State==="running",o=t.tls,l=[`on <strong>${n(t.placement.targetId)}</strong>`];return t.status.Image&&l.push(`<code>${n(t.status.Image)}</code>`),l.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-head">
        <div class="rpc-head-id">
          ${v(t)}
          <strong>${n(t.label)}</strong>
          ${ge(t)}
          <span class="muted small">${l.join(" · ")}</span>
        </div>
        <div class="rpc-head-actions">
          ${(t.actions??[]).map(S=>d(t,S)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(t.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(t.id)}">
            ${j[t.id]?"Close":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(t.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        <div class="rpc-head-url">
          ${a?`<code class="endpoint-url">${n(t.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(t.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((h=(t.networks??[])[0])==null?void 0:h.path)??"/main/evm/<chainId>")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${n(t.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function le(t){const a=[];t.error&&a.push({tone:"bad",text:`This gateway could not be read: ${t.error}${t.hint?` — ${t.hint}`:""}`}),t.blocked&&a.push({tone:"warn",text:t.blocked});for(const l of t.warnings??[])a.push({tone:"warn",text:l});a.push(...be(t));const o=N[t.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(pe).join("")}</div>`}function pe(t){return`
      <div class="strip-line strip-${t.tone}">
        <span class="strip-text">${n(t.text)}</span>
        ${t.cmd?`<code class="strip-cmd">${n(t.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(t.cmd)}">Copy</button>`:""}
      </div>
    `}function be(t){var h,S;const a=t.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((h=a.status)==null?void 0:h.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((S=a.status)==null?void 0:S.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const l=I[t.id]??a.verification??null;return l&&(!l.ok||!l.subscriptionsOk)&&o.push({tone:l.ok?"warn":"bad",text:`${l.summary} Checked ${new Date(l.at).toLocaleString()} — open Settings for the full check.`}),l!=null&&l.expiryWarning&&o.push({tone:"warn",text:l.expiryWarning}),a.rootCaPath&&a.effectiveCertSource==="internal"&&o.push({tone:"note",text:`Served by Caddy's own certificate authority. Install this file (on ${t.placement.targetId}) into the trust store of every device that will call it and the browser warning goes away:`,cmd:a.rootCaPath}),o}function ge(t){switch(t.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function v(t){return t.status.State==="running"?$e("ok"):t.status.State==="unknown"?$e("bad"):$e("neutral")}function d(t,a){const o=na[a];if(!o)return"";const l=f[t.id];return`
      <button class="${o.className}" data-action="gw-${a}" data-gid="${n(t.id)}"
              title="${n(o.title)}" ${l?"disabled":""}>
        ${l===a?'<span class="spinner" aria-label="working"></span>':n(o.label)}
      </button>
    `}function k(t){const a=F[t.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(t.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function R(t){const a=t.networks??[];return a.length===0?`
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
        ${W(t)}
        <div class="chains">
          ${a.map(o=>H(t,o)).join("")}
        </div>
        ${Pe(t)}
      </div>
    `}function H(t,a){const o=a.upstreams??[],l=m(a);return`
      <section class="chain chain-${l.tone}">
        <div class="chain-head">
          <span class="chain-name">${n(a.name)}</span>
          <code class="chain-key">evm:${a.chainId}</code>
          <code class="chain-path">${n(a.path)}</code>
          ${a.url?`<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                         title="Copy ${n(a.url)}">Copy URL</button>`:""}
          <span class="chain-right">
            ${_(o.length,l.tone,a.knownSetSize)}
            <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                    data-gid="${n(t.id)}" data-chain="${a.chainId}">+ Endpoint</button>
            <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                    data-gid="${n(t.id)}" data-chain="${a.chainId}">Remove</button>
          </span>
        </div>
        <p class="chain-verdict${l.why?" chain-verdict-why":""}"${l.why?` title="${n(l.why)}"`:""}>${l.html}</p>
        ${c(t,a)}
        ${(a.warnings??[]).map(h=>`<p class="chain-note">${n(h)}</p>`).join("")}
      </section>
    `}function _(t,a,o){const l=o>0,h=l?o:t,S=Math.min(t,h);let C="";for(let Ie=0;Ie<h;Ie++)C+=`<span class="seg${Ie<S?` seg-on seg-${a}`:""}"></span>`;const w=l&&t>o,V=l?w?`${t} (set is ${o})`:`${t} of ${o}`:`${t}`,Q=`${t} upstream${t===1?"":"s"} configured`,ue=l?`${Q}${w?`, ${t-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${Q}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(ue)}">${C}</span>
      <span class="segs-n">${V}</span>
    `}function m(t){const a=t.upstreams??[];if(a.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!t.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!a.some($)){const l=P(a);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${l.length?` — every upstream here is configured as ${l.map(S=>`<code>${n(S)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(a.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!a.some(l=>l.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=a.filter(l=>!!l.problem);if(o.length>0){const l=a.length-o.length;return{tone:"warn",html:`${o.length} of these ${a.length} endpoints ${o.length===1?"is":"are"} unusable, so ${l===1?"only one can":`only ${l} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function $(t){return/^wss?:\/\//i.test((t.endpoint??"").trim())}function P(t){const a=new Set;for(const o of t){const l=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());l&&a.add(l[1].toLowerCase())}return[...a].sort()}function W(t){const a=g[t.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(Re(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${E[t.id]?"disabled":""}>
          ${E[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(t.id)}">+ Network</button>
      </div>
    `}function c(t,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(l=>b(t,a,l)).join("")}</ul>`}function b(t,a,o){const l=`${t.id}|${a.chainId}|${o.id}`,h=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?$e("bad"):$e("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${A(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${Z(t,a,o)}</div>
        <div class="up-share">${ne(t,a,o)}</div>
        <div class="up-acts">
          ${h.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(l)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[t.id]?"disabled":""}>
                   ${f[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(l)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function A(t){return t.problem?D("unusable","bad"):t.recentOnly?D("recent blocks","warn"):t.local?D("yours","ok"):D("public","neutral")}function K(t,a){var o;if(t)return a==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(o=(t.capabilities??[]).find(l=>l.key===a))==null?void 0:o.status}function Z(t,a,o){const l=G(t.id,a.chainId,o.id);return l?l.unprobeable?`<span class="caps-none" title="${n(l.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Qn.map(h=>M(t,a,l,h)).join("")}</span>`:`<span class="muted small">${g[t.id]===void 0?"probing…":"—"}</span>`}function M(t,a,o,l){const h=(o.capabilities??[]).find(Q=>Q.key===l),S=K(o,l)??"inconclusive",C=ea[l]??l.toUpperCase();let w="cap";S==="unsupported"?w=oe(t,a,l)?"cap missing":"cap off":S==="inconclusive"?w="cap unknown":S==="inconsistent"&&(w="cap mixed");const V=h!=null&&h.detail?`${h.label}: ${h.detail}`:l==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${C}: no verdict`;return`<span class="${w}" title="${n(V)}">${n(C)}</span>`}function oe(t,a,o){const l=(a.upstreams??[]).map(h=>G(t.id,a.chainId,h.id)).filter(h=>!!h&&!h.unprobeable);return l.length>0&&l.every(h=>K(h,o)==="unsupported")}function ne(t,a,o){const l=p[t.id];if(l===void 0)return'<span class="muted small">reading…</span>';if(l===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!l.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const h=ee(t.id,a.chainId,o.id),S=(l.networks??[]).find(ue=>ue.chainId===a.chainId);if(!h||!S||S.attributed===0)return'<span class="muted small">no traffic yet</span>';const C=Math.round(h.actual*100),w=Math.round(h.intended*100),V=h.diverged?o.local?"warn":"":"ok",Q=`${h.succeeded.toLocaleString()} of ${S.attributed.toLocaleString()} answered requests · routing intends ${w}%`+(h.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(Q)}">
        <span class="bar">
          <span class="fill${V?" "+V:""}" style="width:${C}%"></span>
          <span class="tick" style="left:${w}%"></span>
        </span>
        <span class="share-n${h.diverged?" warn":""}">${C}%</span>
        ${h.unconfigured?D("not in config","warn"):""}
      </span>
    `}function Pe(t){const a=p[t.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(Re(a.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function Re(t){const a=new Date(t);return Number.isNaN(a.getTime())?t:a.toLocaleString()}function Ue(t){const a=t.config;return`
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
        ${Ke(t)}
        ${Me(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(t.id)}">Save settings</button>
        </div>
      </div>
    `}function Ke(t){const a=!t.config.MetricsOff;return`
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
    `}function Me(t){var C;const a=n(t.id),o=t.config.TLS??null,l=(o==null?void 0:o.Enabled)??!1,h=(o==null?void 0:o.CertSource)||"internal",S=((C=t.tls)==null?void 0:C.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${a}-tls" ${l?"checked":""} />
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
      ${Ee(t)}
    `}function Ee(t){var C,w;const a=n(t.id),o=((C=t.config.TLS)==null?void 0:C.Enabled)??!1,l=I[t.id]??((w=t.tls)==null?void 0:w.verification)??null,h=U[t.id]??!1,S=O[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!h?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${h?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${S?`<p class="error small">${n(S)}</p>`:""}
      ${l?bt(l):""}
    `}function bt(t){const a=(t.assertions??[]).map(o=>`
          <li class="small">
            ${gt(o.status)}
            <strong>${n(o.title)}</strong>
            <div class="muted">${n(o.detail)}</div>
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
    `}function gt(t){switch(t){case"pass":return D("pass","ok");case"fail":return D("fail","bad");case"unavailable":return D("unavailable","warn");default:return D("skipped","neutral")}}async function yt(t){U[t]=!0,O[t]=null,z();try{I[t]=await bn(t)}catch(a){O[t]=`${he(a)}${Le(a)}`}finally{U[t]=!1,z()}}function Te(t){return{...t.config,Networks:(t.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function Ce(t,a,o){N[t]=null;try{await wn(t,a)}catch(l){return N[t]=`${o?o+": ":""}${he(l)}`,z(),!1}return await y(),!0}async function vt(t,a){const o=a.dataset.gid??"";switch(t){case"refresh":await y();return;case"copy":a.dataset.copy&&await _t(a,a.dataset.copy);return;case"reprobe":await B(o,!0);return;case"toggle-settings":j[o]=!j[o],z();return;case"save-settings":await $t(o);return;case"verify-tls":await yt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await St(o,t.slice(3));return;case"gw-create":case"gw-recreate":await Tt(o);return;case"gw-wipe":qt(o);return;case"add-gateway":jt();return;case"forget-gateway":await Ct(o);return;case"dismiss-orphan":await xt(a.dataset.name??"");return;case"add-chain":Pt(o);return;case"remove-chain":await Lt(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":nt(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await At(a.dataset.key??"");return;case"reset-devnet":await Mt(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function $t(t){const a=q(t);if(!a)return;const o=Te(a),l=s.querySelector(`#gw-${CSS.escape(t)}-port`),h=s.querySelector(`#gw-${CSS.escape(t)}-bind`);if(l){const w=Number.parseInt(l.value.trim(),10);Number.isFinite(w)&&(o.Port=w)}h&&(o.BindAddr=h.value.trim());const S=s.querySelector(`#gw-${CSS.escape(t)}-metrics`);S&&(o.MetricsOff=!S.checked),o.TLS=wt(t,a);const C=a.status.State==="running";await Ce(t,o,"Saving settings")&&(j[t]=!1,C&&(N[t]=null,kt(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),z())}function wt(t,a){var S,C,w,V,Q,ue,Ie;const o=Kt=>s.querySelector(`#gw-${CSS.escape(t)}-${Kt}`),l=o("tls");if(!l)return a.config.TLS??null;const h=Number.parseInt(((S=o("tls-port"))==null?void 0:S.value.trim())??"",10);return{Enabled:l.checked,Hostname:((C=o("tls-host"))==null?void 0:C.value.trim())??"",CertSource:((w=o("tls-source"))==null?void 0:w.value)??"internal",CertFile:((V=o("tls-cert"))==null?void 0:V.value.trim())??"",KeyFile:((Q=o("tls-key"))==null?void 0:Q.value.trim())??"",HTTPSPort:Number.isFinite(h)?h:443,BindAddr:((ue=a.config.TLS)==null?void 0:ue.BindAddr)??"",ImageRef:((Ie=a.config.TLS)==null?void 0:Ie.ImageRef)??""}}function kt(t,a){F[t]=[a]}async function St(t,a){if(!f[t]){f[t]=a,N[t]=null,z();try{await kn(t,a)}catch(o){N[t]=`${a} failed: ${he(o)}${Le(o)}`}f[t]=null,await y()}}async function Tt(t){if(f[t])return;f[t]="create",N[t]=null,F[t]=["starting…"],z();let a;try{a=await Sn(t)}catch(o){N[t]=`${he(o)}${Le(o)}`,F[t]=[],f[t]=null,z();return}T==null||T(),T=Xe(a.targetId,o=>{if(i)return;const l=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(F[t]=[...(F[t]??[]).filter(S=>S!=="starting…"),l],!!o.err||o.stepId===ta&&!!o.done){T==null||T(),T=null,f[t]=null,o.err&&(N[t]="Provisioning failed — see the log below."),y();return}z()})}async function Ct(t){const a=q(t);if(!(!a||!await Be({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await $n(t)}catch(l){N[t]=he(l),z();return}await y()}}async function xt(t){if(t){L[t]=null;try{await fn(t)}catch(a){L[t]=he(a),z();return}await y()}}function Pt(t){const a=q(t);if(!a)return;const o=new Set((a.networks??[]).map(w=>w.chainId)),l=(r==null?void 0:r.presets)??[],h=l.filter(w=>!o.has(w.chainId)),S=l.filter(w=>o.has(w.chainId)),C=((r==null?void 0:r.targets)??[]).some(w=>w.id===a.placement.targetId&&w.hasDevnet);re(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${h.map(w=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${w.chainId}">
                <span>${n(w.name)}</span>
                <span class="muted small">chain ${w.chainId}${w.devnet?C?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${S.length?`<p class="muted small">Already fronted: ${n(S.map(w=>w.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,w=>{if(w==="cancel"){Y();return}if(w==="custom"){Et(t);return}if(w.startsWith("preset:")){const V=Number.parseInt(w.slice(7),10),Q=l.find(ue=>ue.chainId===V);Y(),Q!=null&&Q.devnet?Rt(t,V,C):et(t,V)}})}function Et(t){var a;re(`
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
      `,o=>{if(o==="cancel"){Y();return}if(o!=="add")return;const l=document.getElementById("custom-chain-id"),h=document.getElementById("custom-chain-err"),S=Number.parseInt((l==null?void 0:l.value.trim())??"",10);if(!Number.isFinite(S)||S<=0){h&&(h.className="error small"),h&&(h.textContent="A chain id is a positive whole number.");return}Y(),et(t,S)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function et(t,a){const o=q(t);if(!o)return;const l=Te(o),h=l.Networks??[];h.some(S=>S.ChainID===a)||(h.push({ChainID:a,Upstreams:[]}),l.Networks=h,await It(t,l)&&(z(),nt(t,a)))}async function It(t,a){var S;const o={...a,Networks:(a.Networks??[]).filter(C=>C.Upstreams.length>0)};if(!await Ce(t,o))return!1;const h=q(t);if(h)for(const C of a.Networks??[])C.Upstreams.length===0&&!(h.networks??[]).some(w=>w.chainId===C.ChainID)&&(h.config.Networks=[...h.config.Networks??[],{ChainID:C.ChainID,Upstreams:[]}],h.networks=[...h.networks??[],{chainId:C.ChainID,name:((S=((r==null?void 0:r.presets)??[]).find(w=>w.chainId===C.ChainID))==null?void 0:S.name)??`Chain ${C.ChainID}`,path:`/${h.config.ProjectID}/evm/${C.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function Rt(t,a,o){const l=q(t);if(!l)return;if(!o){re(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${n(l.placement.targetId)}</code>, so adding chain ${a} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(l.placement.targetId)}" data-modal-action="go">Create a devnet on ${n(l.placement.targetId)}</a>
          </div>
        `,()=>Y());return}const h=Te(l),S=h.Networks??[],C={ID:"devnet",Kind:"managed-devnet",TargetID:l.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},w=S.find(V=>V.ChainID===a);w?w.Upstreams.push(C):S.push({ChainID:a,Upstreams:[C]}),h.Networks=S,await Ce(t,h,"Adding the devnet")}async function Lt(t,a){const o=q(t);if(!o||!Number.isFinite(a))return;const l=X(o,a);if(!await Be({title:`Remove ${(l==null?void 0:l.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(l==null?void 0:l.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const S=Te(o);S.Networks=(S.Networks??[]).filter(C=>C.ChainID!==a),await Ce(t,S,"Removing the network")}function tt(t){const a=t.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function At(t){const a=tt(t);if(!a)return;const o=q(a.gid);if(!o)return;const l=Te(o),h=(l.Networks??[]).find(w=>w.ChainID===a.chainId);if(!h)return;const S=h.Upstreams.findIndex((w,V)=>(w.ID||`${a.chainId}-${V}`)===a.upstreamId);S<0||!await Be({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(h.Upstreams.splice(S,1),await Ce(a.gid,l,"Removing the endpoint"))}function nt(t,a){const o=q(t);if(!o||!Number.isFinite(a))return;const l=((r==null?void 0:r.sources)??[]).filter(w=>w.chainId===a),h=X(o,a),S=new Set(((h==null?void 0:h.upstreams)??[]).filter(w=>w.kind!=="external").map(w=>`${w.kind}|${w.targetId??""}`)),C=l.filter(w=>!S.has(`${w.kind}|${w.targetId}`));re(`
        <h2>Add an endpoint for ${n((h==null?void 0:h.name)??`chain ${a}`)}</h2>
        ${C.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${C.map(w=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(w.kind)}:${n(w.targetId)}">
                       <span>${n(w.label)}</span>
                       <span class="muted small">${n(w.endpoint)}</span>
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
      `,w=>{if(w==="cancel"){Y();return}if(w==="known-set"){Ht(t,a);return}if(w==="manual"){Ut(t,a);return}if(w.startsWith("source:")){const[,V,Q]=w.split(":");Y(),Nt(t,a,V,Q)}})}async function Nt(t,a,o,l){const h=q(t);if(!h)return;const S=Te(h),C=S.Networks??[],w={ID:`${o==="managed-devnet"?"devnet":"node"}-${l}`,Kind:o,TargetID:l,Endpoint:"",Local:!0,RecentOnly:!1},V=C.find(Q=>Q.ChainID===a);V?V.Upstreams.push(w):C.push({ChainID:a,Upstreams:[w]}),S.Networks=C,await Ce(t,S,"Adding the endpoint")}function Bt(t){const a=[...t].sort((h,S)=>(h.latencyMs??1e9)-(S.latencyMs??1e9)),o=a.slice(0,3),l=a.find(h=>h.url.startsWith("wss://")||h.url.startsWith("ws://"));return l&&!o.some(h=>h.url===l.url)&&(o.length===3&&o.pop(),o.push(l)),new Set(o.map(h=>h.url))}async function Ht(t,a){let o;try{o=await xn(t,a)}catch(w){re(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(he(w))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Y());return}if(i)return;const l=o.endpoints??[],h=l.filter(w=>!w.alreadyAdded).map(w=>w.url),S=new Set(l.map(w=>w.provider)).size,C=l.map(w=>{const V=[w.websocket?'<span class="t ws">websocket</span>':"",w.archive?'<span class="t ar">archive</span>':"",w.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n(w.url)}</code>
                  <span class="muted small">${n(w.provider)}</span> ${V}</li>`}).join("");re(`<h2>Endpoints for chain ${a}</h2>
       ${l.length?`<p class="muted small">${S} providers valve has measured, in the order the gateway
                should prefer them — ${l.length} entries, because a provider that serves both schemes
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
       </div>`,w=>{Y(),w==="add"&&Ge(t,a,h),w==="discover"&&Dt(t,a)})}async function Dt(t,a){re(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,C=>{C==="cancel"&&Y()});let o;try{o=await Cn(a)}catch(C){const w=He();if(w){const V=document.createElement("p");V.className="error small",V.textContent=`Could not discover endpoints: ${he(C)}`,w.appendChild(V)}return}if(i)return;const l=(o.endpoints??[]).filter(C=>C.status==="live"||C.status==="unprobed"),h=(o.endpoints??[]).filter(C=>C.status==="rejected"),S=Bt(l);re(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${l.length?`<p class="muted small">${l.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${l.map(C=>{const w=S.has(C.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(C.url)}"${w}>
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
          ${l.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,C=>{if(C==="cancel"){Y();return}if(C==="add"){const w=He(),V=w?Array.from(w.querySelectorAll('input[type="checkbox"]:checked')).map(Q=>Q.value):[];Y(),Ge(t,a,V);return}})}function Ut(t,a){var o;re(`
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
      `,l=>{if(l==="cancel"){Y();return}if(l!=="add")return;const h=document.getElementById("manual-endpoint"),S=document.getElementById("manual-recent"),C=document.getElementById("manual-err"),w=(h==null?void 0:h.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(w)){C&&(C.className="error small",C.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Y(),Ge(t,a,[w],(S==null?void 0:S.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ge(t,a,o,l=!1){if(!o.length)return;const h=q(t);if(!h)return;const S=Te(h),C=S.Networks??[];let w=C.find(Q=>Q.ChainID===a);w||(w={ChainID:a,Upstreams:[]},C.push(w));let V=1;for(const Q of w.Upstreams){const ue=/^public-\d+-(\d+)$/.exec(Q.ID??"");ue&&(V=Math.max(V,Number(ue[1])+1))}for(const Q of o)w.Upstreams.some(ue=>ue.Endpoint===Q)||w.Upstreams.push({ID:`public-${a}-${V++}`,Kind:"external",Endpoint:Q,Local:!1,RecentOnly:l});S.Networks=C,await Ce(t,S,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function Mt(t,a){const o=tt(t);if(!o||!a||!await Be({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[o.gid]="reset",N[o.gid]=null,z();let h;try{h=await pn(a)}catch(S){N[o.gid]=`Reset failed: ${he(S)}${Le(S)}`,f[o.gid]=null,z();return}f[o.gid]=null,Ot(a,h),await y()}function Ot(t,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const l=a.report.Cascaded??[],h=a.report.CascadeSkipped??[];re(`
        <h2>Devnet on ${n(t)} reset</h2>
        <ul class="plain-list">${o.map(S=>`<li>${n(S)}</li>`).join("")}</ul>
        ${l.length?`<p class="ok">Restarted in front of it: ${n(l.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${h.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(h.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Y())}function qt(t){const a=q(t);if(!a)return;re(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(t)}</button>
        </div>
      `,h=>{if(h==="cancel"||h==="close"){Y(),y();return}h==="confirm"&&Ft(t)});const o=document.getElementById("wipe-confirm-input"),l=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{l&&(l.disabled=o.value.trim()!==t)}),o==null||o.focus()}async function Ft(t){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await Tn(t)}catch(l){const h=He();if(h){const S=document.createElement("p");S.className="error small",S.textContent=`Wipe failed: ${he(l)}${Le(l)}`,h.appendChild(S)}a&&(a.disabled=!1,a.textContent=`Wipe ${t}`);return}re(`
        <h2>${n(t)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Y(),y()})}function at(t,a){return!a.some(o=>{var l;return((l=o.placement)==null?void 0:l.targetId)===t})}function jt(){var S;const t=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=t.filter(C=>at(C.id,a)),l=new Set(a.map(C=>C.id));if(t.length===0){re(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Y());return}if(o.length===0){re(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Y());return}const h=l.has("default")?"":"default";re(`
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
      `,C=>{if(C==="cancel"){Y();return}C==="create"&&Wt()}),(S=document.getElementById("new-gw-id"))==null||S.focus()}async function Wt(){const t=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),l=document.getElementById("new-gw-err"),h=(t==null?void 0:t.value.trim())??"",S=(a==null?void 0:a.value)??"",C=Number.parseInt((o==null?void 0:o.value.trim())??"",10),w=V=>{l&&(l.className="error small",l.textContent=V)};if(!h){w("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!S){w("Pick the machine it runs on.");return}try{await mn({id:h,placement:{targetId:S,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(C)?C:4e3,Networks:[]}})}catch(V){w(he(V));return}Y(),await y()}async function _t(t,a){const o=await De(a),l=t.textContent;t.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(t.textContent=l)},1500)}function he(t){return t instanceof Error?t.message:String(t)}function Le(t){return t instanceof Se&&t.hint?` — ${t.hint}`:""}return()=>{i=!0,T==null||T(),Y()}}const sa="local";function oa(s){let i=!1,r=!1,e="",p=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ce()}
  `;const g=s.querySelector("#targets-body");ye(s,(u,y)=>{j(u,y)}),E();async function E(){try{const[u,y,x]=await Promise.all([ke(),we(),ut()]);if(i)return;e=x.os,N(u,y)}catch(u){if(i)return;g.innerHTML=`<p class="error">Failed to load machines: ${n(String(u))}</p>`}}function f(){p&&N(p.targets,p.catalog)}function N(u,y){p={targets:u,catalog:y};const x=e==="linux",B=[...u].sort((G,z)=>(G.mode==="local"?-1:0)-(z.mode==="local"?-1:0)),q=B.length?`<div class="card-grid">${B.map(G=>ra(G,y,G.mode!=="local"||x,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=u.some(G=>G.mode==="local");g.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${q}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${F(x,X)}
        ${r?ia():""}
      </section>
    `;const ee=g.querySelector("#fleet-verdict");ee&&ft(ee,ht(u,y))}function F(u,y){const x=`
      <div class="card">
        <h3>A server over SSH ${D("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${u?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${u?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,B=u?`
        <div class="card">
          <h3>This machine ${D("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${n(e)})`:""} ${D("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return y?`<div class="card-grid card-grid-wide">${x}</div>`:`<div class="card-grid card-grid-wide">${u?B+x:x+B}</div>`}async function j(u,y){var x;if(u==="add-local"){await I();return}if(u==="delete-target"){const B=y.dataset.id;if(!B||!await Be({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await U(B);return}if(u==="toggle-ssh"){r=!r,T(),f(),r&&((x=s.querySelector("#ssh-host"))==null||x.focus());return}u==="add-ssh"&&await O()}async function I(){T();try{await st({id:sa,mode:"local"}),await E()}catch(u){L(u)}}async function U(u){try{await zt(u),await E()}catch(y){L(y)}}async function O(){const u=s.querySelector("#ssh-host"),y=s.querySelector("#ssh-user"),x=s.querySelector("#ssh-key"),B=s.querySelector("#ssh-port"),q=s.querySelector("#ssh-id");if(!u||!y||!x||!B||!q)return;const X=u.value.trim(),ee=y.value.trim(),G=x.value.trim(),z=B.value.trim(),de=q.value.trim();if(T(),!X||!ee||!G){L(new Error("host, user, and key path are required"));return}const te=de||ca(X),ae={Host:X,User:ee,KeyPath:G};if(z){const le=Number.parseInt(z,10);if(!Number.isFinite(le)||le<=0){L(new Error("port must be a positive number"));return}ae.Port=le}const se=s.querySelector("#ssh-submit");se&&(se.disabled=!0,se.textContent="Connecting…");try{await st({id:te,mode:"ssh",ssh:ae}),r=!1,await E()}catch(le){L(le),se&&(se.disabled=!1,se.textContent="Add server")}}function L(u){let y=s.querySelector("#targets-error");y||(g.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),y=s.querySelector("#targets-error")),y.textContent=String(u instanceof Error?u.message:u)}function T(){var u;(u=s.querySelector("#targets-error"))==null||u.remove()}return()=>{i=!0}}function ra(s,i,r,e){const p=s.wire,g=s.mode==="local"?"this machine":"SSH",E=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:g;let f;if(!p&&!r)f=`${D("can't run a node","warn")} ${D(e||"not Linux","neutral")}`;else if(!p)f=D("not set up","neutral");else{const N=i.networks.find(j=>j.ChainID===p.ChainID),F=N?N.Name:`chain ${p.ChainID}`;f=`${D(F,"ok")} ${D(p.ExecID,"neutral")} ${D(p.BeaconID,"neutral")}${p.Archive?" "+D("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${E}</p>
      <p>${f}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(s.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function ia(){return`
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
  `}function ca(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const la=document.querySelector("#app"),{contentEl:da,setActiveNav:ua}=In(la);let fe=null;function pa(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"home"};const[r,e]=i;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:e?decodeURIComponent(e):void 0}:{screen:r??"targets"}}function ve(s){const i=document.createElement("div");return da.replaceChildren(i),s(i)}function mt(){if(fe){try{fe()}catch{}fe=null}const{screen:s,id:i}=pa();switch(ua(s),s){case"machine":if(!i){location.hash="#/targets";return}fe=ve(r=>zn(r,i));break;case"setup":case"dash":case"logs":case"services":if(!i){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(i)}`;return;case"security":if(!i){location.hash="#/targets";return}fe=ve(r=>Jn(r,i));break;case"diag":if(!i){location.hash="#/targets";return}fe=ve(r=>Bn(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}fe=ve(r=>Nn(r,i));break;case"rpc":fe=ve(r=>aa(r));break;case"settings":fe=ve(r=>Xn(r));break;case"targets":fe=ve(r=>oa(r));break;case"home":default:fe=ve(r=>Mn(r));break}}window.addEventListener("hashchange",mt);mt();
