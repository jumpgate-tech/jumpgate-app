var Wt=Object.defineProperty;var _t=(s,i,r)=>i in s?Wt(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var Oe=(s,i,r)=>_t(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))e(h);new MutationObserver(h=>{for(const w of h)if(w.type==="childList")for(const R of w.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&e(R)}).observe(document,{childList:!0,subtree:!0});function r(h){const w={};return h.integrity&&(w.integrity=h.integrity),h.referrerPolicy&&(w.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?w.credentials="include":h.crossOrigin==="anonymous"?w.credentials="omit":w.credentials="same-origin",w}function e(h){if(h.ep)return;h.ep=!0;const w=r(h);fetch(h.href,w)}})();function Kt(){return J("/api/host")}function Ce(){return J("/api/catalog")}function xe(){return J("/api/targets")}function st(s){return J("/api/targets",{method:"POST",headers:fe,body:JSON.stringify(s)})}function Gt(s){return J(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Vt(s,i){return J(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function zt(s,i){return J(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:fe,body:JSON.stringify(i)})}function Xe(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function Jt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function Yt(s,i=200){return J(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function Zt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>r.close()}function ot(s,i){const r=i===void 0?{}:{lines:i};return J(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:fe,body:JSON.stringify(r)})}function Xt(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function Qt(s,i){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:i})})}function en(s){return J(`/api/targets/${encodeURIComponent(s)}/du`)}function tn(s){return J(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function nn(s){return J(`/api/targets/${encodeURIComponent(s)}/firewall`)}function an(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function sn(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function on(s){return J(`/api/targets/${encodeURIComponent(s)}/containers`)}function rn(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function cn(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:i})}),e=await r.text();let h=null;try{h=e?JSON.parse(e):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const w=h&&typeof h=="object"&&typeof h.error=="string"?h.error:r.statusText||`HTTP ${r.status}`;throw new $e(r.status,w)}function ln(s,i){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function dn(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:fe}),r=await i.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new $e(i.status,h)}function un(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:fe,body:JSON.stringify(r)})}function ut(){return J("/api/gateways")}async function pn(s){await J(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function hn(s){return J("/api/gateways",{method:"POST",headers:fe,body:JSON.stringify(s)})}function fn(s){return J(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function mn(s){return J(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function bn(s){return J(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function gn(s,i=!1){const r=i?"?refresh=1":"";return J(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function yn(s){return J(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function vn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:fe,body:JSON.stringify(i)})}function $n(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function wn(s){return J(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function kn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:s})}),r=await i.text();let e=null;try{e=r?JSON.parse(r):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new $e(i.status,h)}function Tn(s){return J(`/api/chainlist/${s}`)}function Sn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function Cn(){return J("/api/settings")}function xn(s){return J("/api/settings",{method:"PUT",headers:fe,body:JSON.stringify(s)})}class $e extends Error{constructor(r,e,h,w){super(e);Oe(this,"status");Oe(this,"hint");Oe(this,"code");this.name="ApiError",this.status=r,this.hint=h,this.code=w}}const fe={"Content-Type":"application/json"};async function J(s,i){const r=await fetch(s,i);if(!r.ok){let h=r.statusText||`HTTP ${r.status}`,w,R;try{const m=await r.json();m&&typeof m.error=="string"&&m.error&&(h=m.error),m&&typeof m.hint=="string"&&m.hint&&(w=m.hint),m&&typeof m.code=="string"&&m.code&&(R=m.code)}catch{}throw new $e(r.status,h,w,R)}if(r.status===204)return;const e=await r.text();return e?JSON.parse(e):void 0}const rt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ce(s,i){const r=s&&i&&i!==rt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(rt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function Pn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{const w=h==="machine"?"targets":h;for(const R of r)R.classList.toggle("active",R.dataset.nav===w)}}}function ie(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function En(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function In(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),e=i%60;if(r===0)return`~${e}m`;if(r<48)return`~${r}h ${e}m`;const h=Math.floor(r/24),w=r%24;return`~${h}d ${w}h`}function U(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function ve(s){return`<span class="dot dot-${s}"></span>`}const it=["B","KB","MB","GB","TB","PB"];function Se(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<it.length-1;)i/=1024,r++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${it[r]}`}async function De(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ye(s,i){s.addEventListener("click",r=>{const e=r.target.closest("[data-action]");if(!e||!s.contains(e))return;const h=e.dataset.action;h&&i(h,e,r)})}function Ye(s,i,r){const e=i.find(w=>w.value===r),h=i.map(w=>`
      <li class="dropdown-option${w.value===r?" selected":""}" role="option"
          aria-selected="${w.value===r}" data-value="${n(w.value)}">
        ${n(w.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Ae(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function Qe(s,i){s.addEventListener("click",h=>{const w=h.target,R=w.closest(".dropdown-trigger");if(R&&s.contains(R)){const B=R.closest(".dropdown"),F=!!B&&!B.classList.contains("open");Ae(s),B&&F&&(B.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const m=w.closest(".dropdown-option");if(m&&s.contains(m)){const B=m.closest(".dropdown");Ae(s),i((B==null?void 0:B.dataset.dropdown)??"",m.dataset.value??"");return}Ae(s)});const r=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}const w=h.target;(!w.closest(".dropdown")||!s.contains(w))&&Ae(s)},e=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",e);return}h.key==="Escape"&&Ae(s)};document.addEventListener("click",r),document.addEventListener("keydown",e)}const _e="app-modal";let We=null;function re(s,i){Y();const r=document.createElement("div");r.className="modal-overlay",r.id=_e,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",h=>{const w=h.target.closest("[data-modal-action]");w!=null&&w.dataset.modalAction?i(w.dataset.modalAction):h.target===r&&i("cancel")});const e=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),We=e,document.body.appendChild(r)}function Y(){var s;(s=document.getElementById(_e))==null||s.remove(),We&&(document.removeEventListener("keydown",We),We=null)}function He(){return document.querySelector(`#${_e} .modal`)}function Be(s){return new Promise(i=>{var h;let r=!1;const e=w=>{r||(r=!0,Y(),i(w))};re(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,w=>e(w==="confirm")),(h=document.querySelector(`#${_e} [data-modal-action="confirm"]`))==null||h.focus()})}const Ve=5e3,Rn=60;function Ln(s,i){let r=!1,e=null,h=null,w=null,R=null;const m=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ce()}</div>`;const B=s.querySelector("#an-body");ye(s,(y,d)=>{var k;y==="toggle-endpoint"&&((k=d.closest(".an-endpoint"))==null||k.classList.toggle("expanded"))}),F();async function F(){try{e=((await ut()).gateways??[]).find(d=>d.id===i)??null}catch(y){if(r)return;w=String(y instanceof Error?y.message:y),D();return}if(!r){if(!e){D();return}await j(),R=window.setInterval(()=>void j(),Ve)}}async function j(){try{const y=await bn(i);if(r)return;E(y),h=y,w=null}catch(y){if(r)return;w=String(y instanceof Error?y.message:y)}D()}function E(y){if(!y.enabled||y.error)return;const d=m[m.length-1];d&&d.since!==y.since&&(m.length=0);const k=new Map;for(const I of y.networks??[])k.set(I.chainId,I.received);m.push({t:Date.now(),since:y.since,received:k}),m.length>Rn&&m.shift()}function D(){r||(B.innerHTML=O())}function O(){return w&&!h?`<h1>Analytics</h1><p class="error">${n(w)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${L(e)}
      ${h?u(h):`<p class="muted">Reading the gateway's counters…</p>`}
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
        <div class="an-head-right muted small">${S()}</div>
      </div>
    `}function S(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const y=h.since?new Date(h.since):null;return y&&!Number.isNaN(y.getTime())?`totals since the gateway started, ${n(y.toLocaleString())}<br />re-read every ${Ve/1e3}s`:`re-read every ${Ve/1e3}s`}function u(y){return y.enabled?y.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(y.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:g(y)+le(y):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function g(y){const d=y.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(k=>x(k)).join("")}
      </section>
    `}function x(y){const d=y.methods??[],k=y.endpoints??[],I=y.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${y.chainId}</span>
          <span class="band-name">${n(y.name)}</span>
          ${q(y)}
        </div>
        <div class="an-stats">
          ${N("Received",ie(y.received),"what clients asked this chain for")}
          ${N("Answered",ie(y.answered),"returned by one of your endpoints")}
          ${N("From cache",ie(y.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${N("Failed",ie(y.failed),"asked for and never answered",y.failed>0?"bad":"")}
        </div>
        ${ee(y.chainId)}
        ${I?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':z("Method",d.map(H=>({label:H.method,l:H})))+z("Endpoint",k.map(H=>({label:H.upstream,l:H})))+G(y)}
      </div>
    `}function N(y,d,k,I=""){return`
      <div class="an-stat${I?" an-stat-"+I:""}" title="${n(k)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(y)}</span>
      </div>
    `}function q(y){const d=X(y.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const k=Math.round((m[m.length-1].t-m[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${k}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${k}s</span>
    </span>`}function X(y){if(m.length<2)return null;const d=m[0],k=m[m.length-1],I=(k.t-d.t)/1e3;if(I<=0)return null;const H=(k.received.get(y)??0)-(d.received.get(y)??0);return H<0?null:H/I}function ee(y){if(m.length<3)return"";const d=[];for(let v=1;v<m.length;v++){const P=m[v-1],W=m[v],c=(W.t-P.t)/1e3,b=(W.received.get(y)??0)-(P.received.get(y)??0);d.push(c>0&&b>=0?b/c:0)}const k=Math.max(...d);if(k<=0)return"";const I=240,H=28,_=d.length>1?I/(d.length-1):I,f=d.map((v,P)=>`${(P*_).toFixed(1)},${(H-v/k*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${k.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${I} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${f}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(k.toFixed(2))} req/s</span>
      </div>
    `}function G(y){const d=[];return y.cached.count>0&&d.push(`${n(ie(y.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${y.cached.mean===null?"":`, in ${n(Ne(y.cached.mean))} on average`}.`),y.failedLatency.count>0&&y.failedLatency.mean!==null&&d.push(`The ${n(ie(y.failedLatency.count))} that failed took
         ${n(Ne(y.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function z(y,d){return d.length===0?"":`
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
            ${d.map(k=>de(k.label,k.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function de(y,d){return`
      <tr>
        <td><code>${n(y)}</code></td>
        <td class="an-num">${ie(d.count)}</td>
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(Ne(d.mean))}</td>
        <td>${te(d)}</td>
      </tr>
    `}function te(y){const d=y.buckets??[];if(d.length===0||y.count===0)return'<span class="muted small">—</span>';let k=0;const I=[];for(const _ of d){const f=_.count-k;k=_.count,I.push({label:se(_.le),n:Math.max(0,f)})}return I.reduce((_,f)=>_+f.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(I.filter(_=>_.n>0).map(_=>`${_.n} ${_.label}`).join(" · "))}">
        ${I.map((_,f)=>_.n===0?"":`<span class="an-band an-band-${Math.min(f,4)}" style="flex:${_.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ae(I))}</span>
    `}function ae(y){for(let d=y.length-1;d>=0;d--)if(y[d].n>0)return`slowest ${y[d].label}`;return""}function se(y){if(y==="+Inf")return"30s or more";const d=Number(y);return Number.isFinite(d)?`under ${Ne(d)}`:`under ${y}`}function le(y){const d=y.endpoints??[];return`
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
    `}function pe(y){const d=y.errors??[],k=d.reduce((H,_)=>H+_.count,0),I=d.length>0;return`
      <tr class="an-endpoint${I?" expandable":""}" ${I?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(y.upstream)}</code>
          ${y.chainId?`<span class="muted small">chain ${y.chainId}</span>`:""}
          ${y.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ie(y.requests)}</td>
        <td class="an-num${k>0?" bad":""}">${k>0?ie(k):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${y.headLag>0?ie(y.headLag):'<span class="muted">0</span>'}</td>
        <td>${me(y)}</td>
      </tr>
      ${I?be(y,d):""}
    `}function me(y){const d=[];return y.scored?(d.push(y.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(y.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(y.score.toFixed(3))}</span>`),y.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ie(y.primarySwitches)} switches</span>`),y.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Ne(y.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function be(y,d){return`
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
            Errors the gateway saw when it called <code>${n(y.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{r=!0,R!==null&&window.clearInterval(R)}}function Ne(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}function An(s,i){let r=!1,e=null,h=null,w=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ce()}</div>`;const m=s.querySelector("#diag-body"),B=s.querySelector("#diag-footer");ye(s,(u,g)=>{var x;if(u==="run")j();else if(u==="toggle")(x=g.closest(".check-item"))==null||x.classList.toggle("expanded");else if(u==="copy"){const N=g.dataset.copy;N&&S(g,N)}}),F();async function F(){let u,g;try{const[N,q]=await Promise.all([xe(),Ce()]);u=N.find(X=>X.id===i),g=q}catch(N){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String(N))}</p>`;return}if(r)return;if(!u){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const x=g==null?void 0:g.networks.find(N=>N.ChainID===u.wire.ChainID);x&&(B.innerHTML=ce(x.Name,x.LearnURL));try{e=await sn(i),R=!0}catch(N){h=String(N instanceof Error?N.message:N)}r||E()}async function j(){w=!0,h=null,E();try{e=await an(i),R=!0}catch(u){h=String(u instanceof Error?u.message:u)}w=!1,r||E()}function E(){m.innerHTML=`
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
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${D()}
    `}function D(){if(!R&&!h)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const u=new Date(e.at).toLocaleString(),g=e.failedId?`<p><strong>Failed at: ${n(O(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(u)} — trigger: ${n(e.trigger)}</p>
      ${g}
      <ul class="check-list">${e.items.map(L).join("")}</ul>
    `}function O(u){var g;return((g=e==null?void 0:e.items.find(x=>x.ID===u))==null?void 0:g.Title)??u}function L(u){const g=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral",x=u.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${x?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${U(x?"failed here":u.Status,g)}
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
    `}async function S(u,g){const x=await De(g),N=u.textContent;u.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{r||(u.textContent=N)},1500)}return()=>{r=!0}}const Nn=85,ze={exec:"Execution",beacon:"Beacon"};function Bn(s,i){let r=!1,e=null,h=null,w=null,R=null,m=null,B=null,F=null,j=null;const E={exec:null,beacon:null};let D=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ce()}</div>`;const O=s.querySelector("#dash-body"),L=s.querySelector("#dash-footer");O.addEventListener("click",d=>{const k=d.target.closest("[data-action]");if(!k||!O.contains(k))return;const I=k.dataset.action;if(I==="svc-action"){const H=k.dataset.svc,_=k.dataset.kind;H&&_&&pe(H,_)}else if(I==="open-clear"){const H=k.dataset.svc;H&&be(H)}else if(I==="copy"){const H=k.dataset.copy;H&&me(k,H)}else I==="retry-du"?u():I==="retry-endpoints"&&g()}),S();async function S(){let d,k;try{const[H,_]=await Promise.all([xe(),Ce()]);d=H.find(f=>f.id===i),k=_}catch(H){if(r)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(r)return;if(!d){O.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const I=k==null?void 0:k.networks.find(H=>H.ChainID===d.wire.ChainID);I&&(L.innerHTML=ce(I.Name,I.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',e=Jt(i,H=>{r||(x(H),h=H,w=H,N())}),u(),g()}async function u(){B=null;try{m=await en(i)}catch(d){m=null,B=String(d instanceof Error?d.message:d)}r||N()}async function g(){j=null;try{F=await tn(i)}catch(d){F=null,j=String(d instanceof Error?d.message:d)}r||N()}function x(d){if(!h)return;const k=(new Date(d.at).getTime()-new Date(h.at).getTime())/1e3,I=d.execHead-h.execHead;if(k>0&&I>=0){const H=I/k;R=R===null?H:R*.7+H*.3}}function N(){if(!w)return;const d=w;O.innerHTML=`
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
    `}function q(d){return!d.execActive&&!d.beaconActive?U("Node not running","bad"):d.execSyncing||d.beaconDistance>0?U("Syncing","warn"):U("Running · synced","ok")}function X(d){const I=d.refHead>0?d.refHead-d.execHead:null,H=I!==null&&I>0&&R&&R>0?In(I/R):I!==null&&I<=0?"caught up":"—";return{lag:I,eta:H}}function ee(d){const{lag:k,eta:I}=X(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?U("syncing","warn"):d.execHead===0?U("no data","neutral"):U("synced","ok"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ie(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${k!==null?ie(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${k!==null?ie(Math.max(k,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${I}</dd></div>
        </dl>
      </div>
    `}function G(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?U("no data","neutral"):d.beaconDistance===0?U("synced","ok"):U("syncing","warn"):U("stopped","bad")}</p>
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
    `}function de(d){const k=d.diskUsedPct>=Nn,I=`
      <div class="meter"><div class="meter-fill ${k?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${En(d.diskUsedPct)} used</p>
    `;if(B)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${I}
          <p class="error small">${n(B)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${I}
          <p class="muted">Loading…</p>
        </div>
      `;const H=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,_=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:f,eta:v}=X(d),P=f!==null&&f>0&&R!==null&&R>0;return`
      <div class="card ${k?"card-warn":""}">
        <h3>Storage</h3>
        ${I}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Se(m.ExecBytes)} of ~${Se(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${P?`<p class="muted small">Estimated time remaining: ${n(v)}</p>`:""}
        <p class="muted small">Beacon — ${Se(m.BeaconBytes)} of ~${Se(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${_}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Se(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function te(){if(j)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(j)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!F)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=F,k=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",I=d.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(d.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ve(d.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ve(d.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(d.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(d.BeaconHTTP)}">Copy</button>
        </div>
        ${k}
        ${I}
      </div>
    `}function ae(d,k){const I=ze[d],H=E[d],_=(f,v,P)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${f}" ${H!==null||P?"disabled":""}>${H===f?le():n(v)}</button>`;return`
      <div class="service-row">
        <span>${n(I)} ${k?U("active","ok"):U("down","bad")}</span>
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
        ${D?`<p class="error small">${n(D)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function le(){return'<span class="spinner" aria-label="working"></span>'}async function pe(d,k){if(E[d]===null){E[d]=k,D=null,N();try{await Xt(i,d,k)}catch(I){D=`${ze[d]} ${k} failed: ${I instanceof Error?I.message:String(I)}`}E[d]=null,r||N()}}async function me(d,k){const I=await De(k),H=d.textContent;d.textContent=I?"Copied!":"Copy failed",setTimeout(()=>{r||(d.textContent=H)},1500)}function be(d){const k=ze[d],I=m?Se(d==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";re(`
        <h2>Clear ${n(k)} data</h2>
        <p class="error">
          This stops the ${n(k.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(I)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,f=>{if(f==="cancel"){Y();return}f==="confirm"&&y(d)});const H=document.getElementById("clear-confirm-input"),_=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{_&&(_.disabled=H.value.trim()!==d)}),H==null||H.focus()}async function y(d){const k=document.getElementById("clear-confirm-btn");k&&(k.disabled=!0,k.textContent="Clearing…");try{await Qt(i,d),Y(),u()}catch(I){const H=He();if(H){const _=document.createElement("p");_.className="error small",_.textContent=`Clear failed: ${I instanceof Error?I.message:String(I)}`,H.appendChild(_)}k&&(k.disabled=!1,k.textContent="Clear and resync")}}return()=>{r=!0,e==null||e(),Y()}}const ct=500,lt="valve-node-app.explain-consent";function Hn(s,i){let r=!1,e=null;const h=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ce()}</div>
  `;const w=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");ye(s,S=>{S==="explain"&&j()}),m();async function m(){let S,u;try{const[x,N]=await Promise.all([xe(),Ce()]);S=x.find(q=>q.id===i),u=N}catch(x){if(r)return;w.innerHTML=`<p class="error">Failed to load target: ${n(String(x))}</p>`;return}if(r)return;if(!S){w.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!S.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const g=u==null?void 0:u.networks.find(x=>x.ChainID===S.wire.ChainID);g&&(R.innerHTML=ce(g.Name,g.LearnURL));try{const x=await Yt(i,200);if(r)return;h.push(...x)}catch(x){if(r)return;w.innerHTML=`<p class="error">Failed to load logs: ${n(String(x))}</p>`;return}B(),e=Zt(i,x=>{r||(h.push(x),h.length>ct&&h.splice(0,h.length-ct),B())})}function B(){const S=h.filter(g=>g.severity==="error"||g.severity==="critical");w.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${h.map(F).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${U(String(S.length),S.length?"bad":"neutral")}</h2>
          <div class="log-lines">${S.length?S.slice().reverse().map(F).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=w.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function F(S){const u=S.severity||"info",g=S.learnUrl?` <a href="${n(S.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(u)}">
        <span class="log-time">${n(new Date(S.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(S.unit)}</span>
        <span class="log-sev">${n(u)}</span>
        <span class="log-text">${n(S.line)}</span>
        ${S.explain?`<div class="log-explain">${n(S.explain)}${g}</div>`:""}
      </div>
    `}async function j(){const S=h.filter(g=>g.severity==="error"||g.severity==="critical").map(g=>g.line).slice(-40);if(!(localStorage.getItem(lt)==="1")){E(S);return}await D(S)}function E(S){const u=S.length?`<pre class="explain-excerpt">${S.map(g=>n(g)).join(`
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
    `,g=>{g==="proceed"?(localStorage.setItem(lt,"1"),L(),D(S)):L()})}async function D(S){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=S.length?await ot(i,S):await ot(i);if(r)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map(g=>n(g)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&L()})}catch(u){if(r)return;if(u instanceof $e&&u.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,g=>{g==="close"&&L()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,g=>{g==="close"&&L()})}}function O(S,u){L();const g=document.createElement("div");g.className="modal-overlay",g.id="explain-modal",g.innerHTML=`<div class="modal">${S}</div>`,g.addEventListener("click",x=>{const N=x.target.closest("[data-modal-action]");N!=null&&N.dataset.modalAction&&u(N.dataset.modalAction),x.target===g&&u("cancel")}),document.body.appendChild(g)}function L(){var S;(S=document.getElementById("explain-modal"))==null||S.remove()}return()=>{r=!0,e==null||e(),L()}}const Dn="run",Un={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Mn={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function On(s,i){let r=!1,e=null,h=null;const w={devnet:null},R={devnet:null},m={devnet:[]};let B=null;const F={devnet:!1};let j=null;const E={devnet:null},D={devnet:null};s.innerHTML=`
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
  `;const O=s.querySelector("#services-body");ye(s,(c,b)=>{be(c,b)}),L();async function L(){try{const c=await on(i);if(r)return;e=c,h=null}catch(c){if(r)return;e=null,h=P(c)}u()}function S(c){return e==null?void 0:e.services.find(b=>b.id===c)}function u(){if(!r){if(h){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(h)}</p>`;return}if(!e){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${g(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(x).join("")}
      </div>
    `}}function g(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const b=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(b)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function x(c){const b=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${N(c)}
        </div>
        <p class="muted small">${n(Un[c.id]??"")}</p>

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
        ${R[c.id]?`<p class="error small">${n(R[c.id])}</p>`:""}
        ${z(c)}

        ${de(c)}
      </div>
    `}function N(c){switch(c.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function q(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function X(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const b=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${b}.</p>`}function ee(c){const b=c.endpoints??[];return b.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":b.map(A=>`
        <div class="endpoint-row">
          ${ve("ok")}
          <span class="muted small">${n(A.label)}</span>
          <code class="endpoint-url">${n(A.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(A.url)}">Copy</button>
        </div>`).join("")}function G(c,b){const A=Mn[b];if(!A)return"";const K=w[c.id],Z=b==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:A.label;return`
      <button class="${A.className}" data-action="svc-${b}" data-svc="${n(c.id)}"
              title="${n(A.title)}" ${K?"disabled":""}>
        ${K===b?'<span class="spinner" aria-label="working"></span>':n(Z)}
      </button>
    `}function z(c){const b=m[c.id]??[];return b.length===0?"":`
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
        ${E[c.id]?`<p class="error small">${n(E[c.id])}</p>`:""}
        ${D[c.id]?`<p class="muted small">${n(D[c.id])}</p>`:""}
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
    `:""}function le(){F.devnet&&j&&(j.BlockTime=pe("#dev-blocktime",j.BlockTime),j.HTTPPort=me("#dev-http",j.HTTPPort),j.WSPort=me("#dev-ws",j.WSPort),j.BindAddr=pe("#dev-bind",j.BindAddr))}function pe(c,b){const A=s.querySelector(c);return A?A.value.trim():b}function me(c,b){const A=s.querySelector(c);if(!A)return b;const K=Number.parseInt(A.value.trim(),10);return Number.isFinite(K)?K:b}async function be(c,b){const A=b.dataset.svc??"";switch(c){case"refresh":await L();return;case"copy":b.dataset.copy&&await v(b,b.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await y(A,c.slice(4));return;case"svc-create":case"svc-recreate":await d(A);return;case"svc-wipe":H(A);return;case"toggle-config":k(A);return;case"save-config":await I(A);return;default:return}}async function y(c,b){if(!w[c]){w[c]=b,R[c]=null,u();try{await rn(i,c,b)}catch(A){R[c]=`${b} failed: ${P(A)}${W(A)}`}w[c]=null,await L()}}async function d(c){if(!w[c]){w[c]="create",R[c]=null,m[c]=["starting…"],u();try{await ln(i,c)}catch(b){R[c]=`${P(b)}${W(b)}`,m[c]=[],w[c]=null,u();return}B==null||B(),B=Xe(i,b=>{if(r)return;const A=b.err?`${b.stepId}: ${b.err}`:b.line?`${b.stepId}: ${b.line}`:`${b.stepId}: done`;if(m[c]=[...(m[c]??[]).filter(Z=>Z!=="starting…"),A],!!b.err||b.stepId===Dn&&!!b.done){B==null||B(),B=null,w[c]=null,b.err&&(R[c]="Provisioning failed — see the log below."),L();return}u()})}}function k(c){if(le(),F[c]=!F[c],E[c]=null,D[c]=null,F[c]){const b=S(c);b!=null&&b.devnet&&(j={...b.devnet})}u()}async function I(c){var K;le(),E[c]=null,D[c]=null;const b=j;if(!b)return;if(b.HTTPPort===b.WSPort){E[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await un(i,c,b)}catch(Z){E[c]=P(Z),u();return}const A=((K=S(c))==null?void 0:K.status.State)==="running";F[c]=!1,D[c]=A?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function H(c){const b=S(c);if(!b)return;const A=(b.restartsOnWipe??[]).map(M=>{var oe;return((oe=S(M))==null?void 0:oe.label)??M});re(`
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
      `,M=>{if(M==="cancel"||M==="close"){Y(),L();return}M==="confirm"&&_(c)});const K=document.getElementById("wipe-confirm-input"),Z=document.getElementById("wipe-confirm-btn");K==null||K.addEventListener("input",()=>{Z&&(Z.disabled=K.value.trim()!==c)}),K==null||K.focus()}async function _(c){const b=document.getElementById("wipe-confirm-btn");b&&(b.disabled=!0,b.textContent="Wiping…");let A;try{A=await cn(i,c)}catch(K){const Z=He();if(Z){const M=document.createElement("p");M.className="error small",M.textContent=`Wipe failed: ${P(K)}${W(K)}`,Z.appendChild(M)}b&&(b.disabled=!1,b.textContent=`Wipe ${c}`);return}f(c,A)}function f(c,b){const A=S(c),K=ne=>{var Pe;return((Pe=S(ne))==null?void 0:Pe.label)??ne},Z=[];Z.push(b.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ne of b.report.VolumesRemoved??[])Z.push(`Volume ${ne} deleted.`);for(const ne of b.report.VolumesAbsent??[])Z.push(`Volume ${ne} was already gone.`);b.report.Recreated&&Z.push("Container re-created from your saved configuration.");const M=(b.report.Cascaded??[]).map(K),oe=(b.report.CascadeSkipped??[]).map(K);re(`
        <h2>${n((A==null?void 0:A.label)??c)} wiped</h2>
        <ul class="plain-list">${Z.map(ne=>`<li>${n(ne)}</li>`).join("")}</ul>
        ${M.length?`<p class="ok">Restarted in front of it: ${n(M.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${oe.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(oe.join(", "))}.</p>`:""}
        ${b.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(b.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ne=>{(ne==="close"||ne==="cancel")&&(Y(),L())})}async function v(c,b){const A=await De(b),K=c.textContent;c.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{r||(c.textContent=K)},1500)}function P(c){return c instanceof Error?c.message:String(c)}function W(c){return c instanceof $e&&c.hint?` — ${c.hint}`:""}return()=>{r=!0,B==null||B(),Y()}}const Je=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],qe=8545,Fe=5052,je=30303,qn=[369,943,1],dt={369:"default",943:"practise here first"};function Fn(s,i){let r=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ce()}</div>`;const h=s.querySelector("#wizard-body"),w=s.querySelector("#wizard-footer");ye(s,(f,v)=>{me(f,v)}),Qe(s,(f,v)=>{f==="exec-select"?e.execId=v:f==="beacon-select"&&(e.beaconId=v),m()}),s.addEventListener("change",f=>{const v=f.target;v instanceof HTMLInputElement&&(v.id==="data-dir-input"?(be(),G()):v.id==="checkpoint-toggle"?(e.checkpoint=v.checked,m()):v.id==="exec-snapshot-toggle"&&(e.execSnapshot=v.checked,m()))}),R();async function R(){try{const[f,v]=await Promise.all([Ce(),xe()]);if(r)return;e.catalog=f;const P=v.find(W=>W.id===i);P!=null&&P.wire&&(e.chainId=P.wire.ChainID,e.execId=P.wire.ExecID,e.beaconId=P.wire.BeaconID,e.archive=P.wire.Archive,P.wire.ExecHTTPPort&&(e.execHTTPPort=String(P.wire.ExecHTTPPort)),P.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(P.wire.BeaconHTTPPort)),P.wire.ExecP2PPort&&(e.execP2PPort=String(P.wire.ExecP2PPort)),P.wire.RPCBindAddr&&(e.rpcBindAddr=P.wire.RPCBindAddr)),m()}catch(f){if(r)return;e.loadError=String(f instanceof Error?f.message:f),m()}}function m(){if(e.loadError){h.innerHTML=`<p class="error">Failed to load: ${n(e.loadError)}</p>`;return}e.catalog&&(h.innerHTML=`
      ${_(e.step)}
      ${F()}
    `,B())}function B(){var v;const f=(v=e.catalog)==null?void 0:v.networks.find(P=>P.ChainID===e.chainId);w.innerHTML=f?ce(f.Name,f.LearnURL):ce()}function F(){switch(e.step){case"network":return j();case"clients":return E();case"mode":return se();case"review":return le();case"run":return pe()}}function j(){const f=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${qn.map(P=>{const W=f.networks.find(A=>A.ChainID===P);if(!W)return"";const c=e.chainId===P,b=dt[P]?U(dt[P],P===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${P}" type="button">
          <h3>${n(W.Name)} <span class="muted">(chain ${P})</span></h3>
          ${b}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function E(){const f=e.catalog,v=f.networks.find(c=>c.ChainID===e.chainId);if(!v)return'<p class="error">Unknown network.</p>';(e.execId===null||!v.ExecClients.includes(e.execId))&&(e.execId=v.ExecClients[0]??null),(e.beaconId===null||!v.BeaconClients.includes(e.beaconId))&&(e.beaconId=v.BeaconClients[0]??null);const P=v.ExecClients.map(c=>de(c,f)),W=v.BeaconClients.map(c=>de(c,f));return`
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
          ${Ye("exec-select",P,e.execId)}
        </label>
        ${ae(e.execId,f)}
        <label>
          Beacon client
          ${Ye("beacon-select",W,e.beaconId)}
        </label>
        ${ae(e.beaconId,f)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function D(f){return f<=0?"—":f>=1?`~${f.toFixed(1)} TB`:`~${Math.round(f*1e3)} GB`}const O=1.1,L=.5,S="Valve reth snapshot",u="rough estimate";function g(f){return f.SnapshotSizeTB}function x(f){return f.SnapshotSizeTB*L}function N(f){return`<p class="muted small">${D(g(f))} is the measured size of Valve's reth snapshot for ${n(f.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function q(f){return{archive:g(f)*1e12*O,full:x(f)*1e12*O}}function X(f,v){if(!f)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${n(v)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${n(v)}</code>: ${n(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==v)return"";const P=q(f),W=e.freeBytes>=P.archive,c=e.freeBytes>=P.full,b=`<p class="muted small">Free at <code>${n(v)}</code>: <strong>${Se(e.freeBytes)}</strong> — archive ${W?"fits":"won't fit"} (${D(g(f))}, ${S}), full ${c?"fits":"won't fit"} (${D(x(f))}, ${u}).</p>`;let A="";return e.downgradeNote?A=`<p class="banner banner-warn">${n(e.downgradeNote)}</p>`:c||(A=`<p class="banner banner-warn">Neither full (${D(x(f))}, ${u}) nor archive (${D(g(f))}, ${S}) fits the free space here — choose a location with more room.</p>`),b+A}function ee(f,v){if(e.downgradeNote=null,!f||e.freeBytes===null)return;const P=q(f);e.archive&&e.freeBytes<P.archive&&e.freeBytes>=P.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${v} for archive (${D(g(f))}, ${S}) — switched to Full (${D(x(f))}, ${u}). Pick a location with more room to run archive.`)}async function G(){var P;if(e.chainId===null)return;const f=(P=e.catalog)==null?void 0:P.networks.find(W=>W.ChainID===e.chainId),v=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,m();try{const{freeBytes:W}=await Vt(e.targetId,v);if(r)return;e.freeBytes=W,e.probedPath=v,ee(f,v)}catch(W){if(r)return;e.freeBytes=null,e.probedPath=v,e.diskError=String(W instanceof Error?W.message:W)}e.diskProbing=!1,m()}function z(f){return f?/^https?:\/\/.+/i.test(f)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(f,v){const P=v.clients.find(W=>W.id===f);return{value:f,label:P?`${P.id} — ${te(P.repo)}`:f}}function te(f){const v=f.split("/");return v.length>=4?v[3]:f}function ae(f,v){const P=f?v.clients.find(c=>c.id===f):void 0;if(!P)return"";const W=P.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(P.repo)}" target="_blank" rel="noopener noreferrer">${n(W)}</a></p>`}function se(){var K,Z,M;const f=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",v=(K=e.catalog)==null?void 0:K.networks.find(oe=>oe.ChainID===e.chainId),P=((M=(Z=e.catalog)==null?void 0:Z.clients.find(oe=>oe.id===e.execId))==null?void 0:M.snapshotSupported)??!1,W=v?`${D(x(v))} (${u})`:"Smaller",c=v?`${D(g(v))} (${S})`:"Much larger",b=v?` on ${n(v.Name)}`:"",A=v?e.checkpoint?v.SyncLabel:v.GenesisSyncLabel:"";return`
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
          ${v?`<p class="sync-estimate">⏱ Estimated initial sync${b}: <strong>${n(A)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((v==null?void 0:v.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((v==null?void 0:v.CheckpointURL)??"")}" value="${n(e.checkpointUrl)}" />
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
          ${v?N(v):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${v?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${W}${v?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(f)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(f)}" value="${n(e.dataDir)}" />
          </label>
          ${X(v,e.dataDir||f)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(f)}/jwt.hex" value="${n(e.jwtPath)}" />
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
    `}function le(){const v=e.catalog.networks.find(ne=>ne.ChainID===e.chainId),P=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,W=e.jwtPath||`${P}/jwt.hex`,c=Je.map(ne=>`<li>${n(ne.title)}</li>`).join(""),b=I(e.execHTTPPort,qe),A=I(e.beaconHTTPPort,Fe),K=I(e.execP2PPort,je),Z=b||A||K?`<tr><th>Non-default ports</th><td>${[b?`exec HTTP ${b}`:null,A?`beacon HTTP ${A}`:null,K?`exec p2p ${K}`:null].filter(ne=>ne!==null).map(n).join(", ")}</td></tr>`:"",{addr:M}=y(e.rpcBindAddr),oe=M?`<tr><th>RPC bind address</th><td><code>${n(M)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((v==null?void 0:v.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(P)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(W)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${n(e.checkpointUrl||(v==null?void 0:v.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
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
    `}function pe(){const v=e.catalog.networks.find(M=>M.ChainID===e.chainId),P=v==null?void 0:v.LearnURL,W=new Set(e.events.filter(M=>M.done).map(M=>M.stepId)),c=new Set(e.events.filter(M=>M.err).map(M=>M.stepId)),b=new Map;for(const M of e.events){if(!M.line)continue;const oe=b.get(M.stepId)??[];oe.push(M.line),b.set(M.stepId,oe)}const A=Je.map(M=>{var Me;const oe=W.has(M.id),ne=c.has(M.id),Pe=ne?U("failed","bad"):oe?U("done","ok"):U("pending","neutral"),Re=(b.get(M.id)??[]).slice(-5),Ue=(Me=e.events.find(Ee=>Ee.stepId===M.id&&Ee.err))==null?void 0:Me.err,Ke=M.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${P?` <a href="${n(P)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
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
    `}function me(f,v){switch(f){case"pick-network":e.chainId=Number(v.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m(),G();break;case"goto-review":if(be(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){m();break}e.step="review",m();break;case"start-setup":H();break}}function be(){const f=s.querySelectorAll('input[name="mode"]');for(const M of Array.from(f))M.checked&&(e.archive=M.value==="archive");const v=s.querySelector("#data-dir-input"),P=s.querySelector("#jwt-path-input");v&&(e.dataDir=v.value.trim()),P&&(e.jwtPath=P.value.trim());const W=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),b=s.querySelector("#exec-p2p-port-input");W&&(e.execHTTPPort=W.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),b&&(e.execP2PPort=b.value.trim());const A=s.querySelector("#rpc-bind-addr-input");A&&(e.rpcBindAddr=A.value.trim());const K=s.querySelector("#checkpoint-url-input");K&&(e.checkpointUrl=K.value.trim());const Z=s.querySelector("#snapshot-key-input");Z&&(e.snapshotKey=Z.value.trim()),e.execHTTPPortError=k(e.execHTTPPort).error??null,e.beaconHTTPPortError=k(e.beaconHTTPPort).error??null,e.execP2PPortError=k(e.execP2PPort).error??null,e.rpcBindAddrError=y(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?z(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function y(f){if(!f)return{};const v=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(f);return v?v.slice(1).every(P=>Number(P)<=255)?{addr:f}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(f)&&f.includes(":")?{addr:f}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function k(f){if(!f)return{};if(!d.test(f))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const v=Number(f);return!Number.isInteger(v)||v<1||v>65535?{error:"Port must be between 1 and 65535."}:{port:v}}function I(f,v){const{port:P}=k(f);if(!(P===void 0||P===v))return P}async function H(){var b;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(b=e.streamStop)==null||b.call(e),e.streamStop=null,m();const f={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(f.DataDir=e.dataDir),e.jwtPath&&(f.JWTPath=e.jwtPath);const v=I(e.execHTTPPort,qe),P=I(e.beaconHTTPPort,Fe),W=I(e.execP2PPort,je);v!==void 0&&(f.ExecHTTPPort=v),P!==void 0&&(f.BeaconHTTPPort=P),W!==void 0&&(f.ExecP2PPort=W);const{addr:c}=y(e.rpcBindAddr);c!==void 0&&(f.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(f.CheckpointURL=e.checkpointUrl):f.NoCheckpoint=!0,e.execSnapshot&&(f.ExecSnapshot=!0,f.SnapshotKey=e.snapshotKey);try{await zt(e.targetId,f)}catch(A){if(!(A instanceof $e&&A.status===409)){e.starting=!1,e.startError=String(A instanceof Error?A.message:A),m();return}}e.starting=!1,e.step="run",m(),e.streamStop=Xe(e.targetId,A=>{r||(e.events.push(A),e.step==="run"&&m())})}function _(f){const v=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],W=v.map(c=>c.id).indexOf(f);return`
      <ol class="wizard-progress">
        ${v.map((c,b)=>`<li class="${b===W?"current":b<W?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var f;r=!0,(f=e.streamStop)==null||f.call(e)}}function jn(s,i){let r=!1;const e=new Map;s.innerHTML=`<h1>${n(i)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const h=s.querySelector("#machine-body");ye(s,(E,D)=>{E==="toggle-section"&&F(D.dataset.section??"")}),w();async function w(){let E,D;try{const[O,L]=await Promise.all([xe(),Ce()]);E=O.find(S=>S.id===i),D=L}catch(O){if(r)return;h.innerHTML=`<p class="error">Failed to load machine: ${n(String(O))}</p>`;return}if(!r){if(!E){location.hash="#/targets";return}R(E,D)}}function R(E,D){const O=E.mode==="local"?"this machine":"SSH",L=E.mode==="ssh"&&E.ssh?`${n(E.ssh.User)}@${n(E.ssh.Host)}`:O;h.innerHTML=`
      <p class="muted">${L}</p>
      <p>${m(E,D)}</p>
      <div class="machine-sections">
        ${j.map(S=>B(S,E,D)).join("")}
      </div>
      ${ce()}
    `}function m(E,D){const O=E.wire;if(!O)return U("not set up","neutral");const L=D.networks.find(u=>u.ChainID===O.ChainID),S=L?L.Name:`chain ${O.ChainID}`;return`${U(S,"ok")} ${U(O.ExecID,"neutral")} ${U(O.BeaconID,"neutral")}${O.Archive?" "+U("archive","warn"):""}`}function B(E,D,O){return`
      <section class="card machine-section" data-section-card="${n(E.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(E.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(E.title)}</span>
          <span class="machine-section-status">${E.status(D,O)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(E.key)}" hidden></div>
      </section>
    `}function F(E){const D=j.find(g=>g.key===E);if(!D)return;const O=s.querySelector(`[data-section-card="${E}"]`),L=s.querySelector(`[data-section-body="${E}"]`),S=s.querySelector(`.machine-section-head[data-section="${E}"]`);if(!O||!L||!S)return;const u=L.hidden;if(u&&!e.has(E)){const g=document.createElement("div");L.appendChild(g),e.set(E,D.mount(g))}L.hidden=!u,O.classList.toggle("open",u),S.setAttribute("aria-expanded",String(u))}const j=[{key:"setup",title:"Setup",status:E=>E.wire?U("set up","ok"):U("not set up","neutral"),mount:E=>Fn(E,i)},{key:"dashboard",title:"Dashboard",status:E=>E.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>Bn(E,i)},{key:"logs",title:"Logs",status:E=>E.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>Hn(E,i)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:E=>On(E,i)}];return()=>{r=!0;for(const E of e.values())try{E()}catch{}e.clear()}}function Wn(s,i){let r=!1,e=[],h=null,w=!1,R=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ce()}</div>`;const m=s.querySelector("#sec-body"),B=s.querySelector("#sec-footer");ye(s,(L,S)=>{var u;if(L==="rerun")j();else if(L==="toggle")(u=S.closest(".check-item"))==null||u.classList.toggle("expanded");else if(L==="copy"){const g=S.dataset.copy;g&&O(S,g)}}),F();async function F(){let L,S;try{const[g,x]=await Promise.all([xe(),Ce()]);L=g.find(N=>N.id===i),S=x}catch(g){if(r)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String(g))}</p>`;return}if(r)return;if(!L){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=S==null?void 0:S.networks.find(g=>g.ChainID===L.wire.ChainID);u&&(B.innerHTML=ce(u.Name,u.LearnURL)),await j()}async function j(){w=!0,h=null,E();try{e=await nn(i),R=!0}catch(L){h=String(L instanceof Error?L.message:L)}w=!1,r||E()}function E(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${w?"disabled":""}>${w?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${!R&&w?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(D).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function D(L){const S=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${U(L.Status,S)}
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
    `}async function O(L,S){const u=await De(S),g=L.textContent;L.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{r||(L.textContent=g)},1500)}return()=>{r=!0}}const _n=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],Ze="VALVE_API_KEY";function Kn(s){return s===Ze?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(s)}}</code> slot wherever an endpoint URL carries one.`}function Gn(s){let i=!1,r=!1,e=!1,h=null,w=!1,R=null,m=null;const B=new Set,F=new Map;let j="",E="";s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ce()}`;const D=s.querySelector("#settings-body");ye(s,(x,N)=>{if(x==="save"&&g(),x==="clear-key"){if(!R)return;r=!0;const q=s.querySelector("#ai-key");q&&(q.value=""),u(R)}if(x==="clear-provider-key"){const q=N.dataset.key;if(!R||!q)return;B.add(q),F.set(q,""),w=!1,u(R)}}),Qe(s,(x,N)=>{x!=="ai-provider"||!R||(m=N,w=!1,u(R))}),O();async function O(){try{const x=await Cn();if(i)return;R=x,u(x)}catch(x){if(i)return;D.innerHTML=`<p class="error">Failed to load settings: ${n(String(x))}</p>`}}function L(x){const q=(Array.isArray(x.providerKeysSet)?x.providerKeysSet:[]).filter(X=>X!==Ze).sort();return[Ze,...q]}function S(x,N){const q=n(x);return`
      <div class="pk-row">
        <label>
          <code>${q}</code>
          <input class="provider-key" data-key="${q}" type="password" autocomplete="off"
                 placeholder="${N?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Kn(x)}</p>
        ${N?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${q}">Clear saved key</button>`:""}
      </div>`}function u(x){var de;const N=m??x.aiProvider,q=Array.isArray(x.providerKeysSet)?x.providerKeysSet:[],X=L(x).map(te=>S(te,q.includes(te))).join("");D.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ye("ai-provider",_n.map(te=>({value:te.value,label:te.label})),N)}
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
        ${h?`<p class="error">${n(h)}</p>`:""}
        ${w?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const ee=s.querySelector("#ai-key");ee==null||ee.addEventListener("input",()=>{r=!0,w=!1}),(de=s.querySelector("#ref-rpc-base"))==null||de.addEventListener("input",()=>{w=!1}),s.querySelectorAll("input.provider-key").forEach(te=>{const ae=te.dataset.key;if(!ae)return;const se=F.get(ae);se!==void 0&&(te.value=se),te.addEventListener("input",()=>{B.add(ae),F.set(ae,te.value),w=!1})});const G=s.querySelector("#pk-new-value");G&&(G.value=E),G==null||G.addEventListener("input",()=>{E=G.value,w=!1});const z=s.querySelector("#pk-new-name");z==null||z.addEventListener("input",()=>{j=z.value,w=!1})}async function g(){const x=s.querySelector("#ai-key"),N=s.querySelector("#ref-rpc-base");if(!x||!N||!R)return;const q={aiProvider:m??R.aiProvider,refRpcBase:N.value.trim()};r&&(q.aiKey=x.value);const X={};for(const G of B)X[G]=F.get(G)??"";const ee=j.trim();ee&&(X[ee]=E),Object.keys(X).length>0&&(q.providerKeys=X),e=!0,h=null,w=!1,u(R);try{const G=await xn(q);if(i)return;R=G,r=!1,B.clear(),F.clear(),j="",E="",e=!1,w=!0,u(G)}catch(G){if(i)return;e=!1,h=String(G instanceof Error?G.message:G),u(R)}}return()=>{i=!0}}const Vn=["http","ws","archive","trace"],zn={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Jn="run",Yn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Zn(s){let i=!1,r=null,e=null;const h={},w={},R={},m={},B={},F={},j={},E={},D={},O={},L={};let S=null;s.innerHTML=`
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
  `;const u=s.querySelector("#rpc-body");ye(s,(t,a)=>{bt(t,a)}),Qe(s,()=>{}),g();async function g(){try{const t=await ut();if(i)return;r=t,e=null}catch(t){if(i)return;r=null,e=he(t)}z();for(const t of(r==null?void 0:r.gateways)??[])x(t.id),N(t.id,!1)}async function x(t){try{const a=await mn(t);if(i)return;h[t]=a}catch{if(i)return;h[t]=null}z()}async function N(t,a){R[t]=a,a&&z();try{const o=await gn(t,a);if(i)return;w[t]=o}catch{if(i)return;w[t]=null}R[t]=!1,z()}function q(t){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===t)}function X(t,a){return(t.networks??[]).find(o=>o.chainId===a)}function ee(t,a,o){var p;const l=(((p=h[t])==null?void 0:p.networks)??[]).find(T=>T.chainId===a);return((l==null?void 0:l.upstreams)??[]).find(T=>T.upstream===o)}function G(t,a,o){var l;return(((l=w[t])==null?void 0:l.endpoints)??[]).find(p=>p.chainId===a&&p.upstream===o)}function z(){if(i)return;if(e){u.innerHTML=`<p class="error">Could not read the gateways: ${n(e)}</p>`;return}if(!r){u.innerHTML='<p class="muted">Loading…</p>';return}const t=r.gateways??[],a=t.length>1,o=(r.targets??[]).some(l=>at(l.id,t));u.innerHTML=`
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
      ${I(t)}
    `}function se(t){var p;const a=t.status.State==="running",o=t.tls,l=[`on <strong>${n(t.placement.targetId)}</strong>`];return t.status.Image&&l.push(`<code>${n(t.status.Image)}</code>`),l.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-head">
        <div class="rpc-head-id">
          ${y(t)}
          <strong>${n(t.label)}</strong>
          ${be(t)}
          <span class="muted small">${l.join(" · ")}</span>
        </div>
        <div class="rpc-head-actions">
          ${(t.actions??[]).map(T=>d(t,T)).join("")}
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
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((p=(t.networks??[])[0])==null?void 0:p.path)??"/main/evm/<chainId>")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${n(t.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function le(t){const a=[];t.error&&a.push({tone:"bad",text:`This gateway could not be read: ${t.error}${t.hint?` — ${t.hint}`:""}`}),t.blocked&&a.push({tone:"warn",text:t.blocked});for(const l of t.warnings??[])a.push({tone:"warn",text:l});a.push(...me(t));const o=B[t.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(pe).join("")}</div>`}function pe(t){return`
      <div class="strip-line strip-${t.tone}">
        <span class="strip-text">${n(t.text)}</span>
        ${t.cmd?`<code class="strip-cmd">${n(t.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(t.cmd)}">Copy</button>`:""}
      </div>
    `}function me(t){var p,T;const a=t.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((p=a.status)==null?void 0:p.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((T=a.status)==null?void 0:T.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const l=E[t.id]??a.verification??null;return l&&(!l.ok||!l.subscriptionsOk)&&o.push({tone:l.ok?"warn":"bad",text:`${l.summary} Checked ${new Date(l.at).toLocaleString()} — open Settings for the full check.`}),l!=null&&l.expiryWarning&&o.push({tone:"warn",text:l.expiryWarning}),a.rootCaPath&&a.effectiveCertSource==="internal"&&o.push({tone:"note",text:`Served by Caddy's own certificate authority. Install this file (on ${t.placement.targetId}) into the trust store of every device that will call it and the browser warning goes away:`,cmd:a.rootCaPath}),o}function be(t){switch(t.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function y(t){return t.status.State==="running"?ve("ok"):t.status.State==="unknown"?ve("bad"):ve("neutral")}function d(t,a){const o=Yn[a];if(!o)return"";const l=m[t.id];return`
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
    `}function I(t){const a=t.networks??[];return a.length===0?`
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
    `}function H(t,a){const o=a.upstreams??[],l=f(a);return`
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
        ${(a.warnings??[]).map(p=>`<p class="chain-note">${n(p)}</p>`).join("")}
      </section>
    `}function _(t,a,o){const l=o>0,p=l?o:t,T=Math.min(t,p);let C="";for(let Ie=0;Ie<p;Ie++)C+=`<span class="seg${Ie<T?` seg-on seg-${a}`:""}"></span>`;const $=l&&t>o,V=l?$?`${t} (set is ${o})`:`${t} of ${o}`:`${t}`,Q=`${t} upstream${t===1?"":"s"} configured`,ue=l?`${Q}${$?`, ${t-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${Q}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(ue)}">${C}</span>
      <span class="segs-n">${V}</span>
    `}function f(t){const a=t.upstreams??[];if(a.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!t.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!a.some(v)){const l=P(a);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${l.length?` — every upstream here is configured as ${l.map(T=>`<code>${n(T)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(a.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!a.some(l=>l.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=a.filter(l=>!!l.problem);if(o.length>0){const l=a.length-o.length;return{tone:"warn",html:`${o.length} of these ${a.length} endpoints ${o.length===1?"is":"are"} unusable, so ${l===1?"only one can":`only ${l} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function v(t){return/^wss?:\/\//i.test((t.endpoint??"").trim())}function P(t){const a=new Set;for(const o of t){const l=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());l&&a.add(l[1].toLowerCase())}return[...a].sort()}function W(t){const a=w[t.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(Re(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[t.id]?"disabled":""}>
          ${R[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(t.id)}">+ Network</button>
      </div>
    `}function c(t,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(l=>b(t,a,l)).join("")}</ul>`}function b(t,a,o){const l=`${t.id}|${a.chainId}|${o.id}`,p=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?ve("bad"):ve("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${A(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${Z(t,a,o)}</div>
        <div class="up-share">${ne(t,a,o)}</div>
        <div class="up-acts">
          ${p.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(l)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${m[t.id]?"disabled":""}>
                   ${m[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(l)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function A(t){return t.problem?U("unusable","bad"):t.recentOnly?U("recent blocks","warn"):t.local?U("yours","ok"):U("public","neutral")}function K(t,a){var o;if(t)return a==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(o=(t.capabilities??[]).find(l=>l.key===a))==null?void 0:o.status}function Z(t,a,o){const l=G(t.id,a.chainId,o.id);return l?l.unprobeable?`<span class="caps-none" title="${n(l.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Vn.map(p=>M(t,a,l,p)).join("")}</span>`:`<span class="muted small">${w[t.id]===void 0?"probing…":"—"}</span>`}function M(t,a,o,l){const p=(o.capabilities??[]).find(Q=>Q.key===l),T=K(o,l)??"inconclusive",C=zn[l]??l.toUpperCase();let $="cap";T==="unsupported"?$=oe(t,a,l)?"cap missing":"cap off":T==="inconclusive"?$="cap unknown":T==="inconsistent"&&($="cap mixed");const V=p!=null&&p.detail?`${p.label}: ${p.detail}`:l==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${C}: no verdict`;return`<span class="${$}" title="${n(V)}">${n(C)}</span>`}function oe(t,a,o){const l=(a.upstreams??[]).map(p=>G(t.id,a.chainId,p.id)).filter(p=>!!p&&!p.unprobeable);return l.length>0&&l.every(p=>K(p,o)==="unsupported")}function ne(t,a,o){const l=h[t.id];if(l===void 0)return'<span class="muted small">reading…</span>';if(l===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!l.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const p=ee(t.id,a.chainId,o.id),T=(l.networks??[]).find(ue=>ue.chainId===a.chainId);if(!p||!T||T.attributed===0)return'<span class="muted small">no traffic yet</span>';const C=Math.round(p.actual*100),$=Math.round(p.intended*100),V=p.diverged?o.local?"warn":"":"ok",Q=`${p.succeeded.toLocaleString()} of ${T.attributed.toLocaleString()} answered requests · routing intends ${$}%`+(p.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(Q)}">
        <span class="bar">
          <span class="fill${V?" "+V:""}" style="width:${C}%"></span>
          <span class="tick" style="left:${$}%"></span>
        </span>
        <span class="share-n${p.diverged?" warn":""}">${C}%</span>
        ${p.unconfigured?U("not in config","warn"):""}
      </span>
    `}function Pe(t){const a=h[t.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
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
    `}function Me(t){var C;const a=n(t.id),o=t.config.TLS??null,l=(o==null?void 0:o.Enabled)??!1,p=(o==null?void 0:o.CertSource)||"internal",T=((C=t.tls)==null?void 0:C.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${a}-tls-host" value="${n((o==null?void 0:o.Hostname)??T)}"
               placeholder="${n(T||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${T?`<p class="muted small">
               The default is <code>${n(T)}</code>. That whole domain's wildcard resolves to
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
          <option value="internal" ${p==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${p==="files"?"selected":""}>A certificate file on this machine</option>
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
    `}function Ee(t){var C,$;const a=n(t.id),o=((C=t.config.TLS)==null?void 0:C.Enabled)??!1,l=E[t.id]??(($=t.tls)==null?void 0:$.verification)??null,p=D[t.id]??!1,T=O[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!p?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${p?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${T?`<p class="error small">${n(T)}</p>`:""}
      ${l?ht(l):""}
    `}function ht(t){const a=(t.assertions??[]).map(o=>`
          <li class="small">
            ${ft(o.status)}
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
    `}function ft(t){switch(t){case"pass":return U("pass","ok");case"fail":return U("fail","bad");case"unavailable":return U("unavailable","warn");default:return U("skipped","neutral")}}async function mt(t){D[t]=!0,O[t]=null,z();try{E[t]=await fn(t)}catch(a){O[t]=`${he(a)}${Le(a)}`}finally{D[t]=!1,z()}}function we(t){return{...t.config,Networks:(t.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function ke(t,a,o){B[t]=null;try{await vn(t,a)}catch(l){return B[t]=`${o?o+": ":""}${he(l)}`,z(),!1}return await g(),!0}async function bt(t,a){const o=a.dataset.gid??"";switch(t){case"refresh":await g();return;case"copy":a.dataset.copy&&await Ft(a,a.dataset.copy);return;case"reprobe":await N(o,!0);return;case"toggle-settings":j[o]=!j[o],z();return;case"save-settings":await gt(o);return;case"verify-tls":await mt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await $t(o,t.slice(3));return;case"gw-create":case"gw-recreate":await wt(o);return;case"gw-wipe":Ut(o);return;case"add-gateway":Ot();return;case"forget-gateway":await kt(o);return;case"dismiss-orphan":await Tt(a.dataset.name??"");return;case"add-chain":St(o);return;case"remove-chain":await Et(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":nt(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await It(a.dataset.key??"");return;case"reset-devnet":await Ht(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function gt(t){const a=q(t);if(!a)return;const o=we(a),l=s.querySelector(`#gw-${CSS.escape(t)}-port`),p=s.querySelector(`#gw-${CSS.escape(t)}-bind`);if(l){const $=Number.parseInt(l.value.trim(),10);Number.isFinite($)&&(o.Port=$)}p&&(o.BindAddr=p.value.trim());const T=s.querySelector(`#gw-${CSS.escape(t)}-metrics`);T&&(o.MetricsOff=!T.checked),o.TLS=yt(t,a);const C=a.status.State==="running";await ke(t,o,"Saving settings")&&(j[t]=!1,C&&(B[t]=null,vt(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),z())}function yt(t,a){var T,C,$,V,Q,ue,Ie;const o=jt=>s.querySelector(`#gw-${CSS.escape(t)}-${jt}`),l=o("tls");if(!l)return a.config.TLS??null;const p=Number.parseInt(((T=o("tls-port"))==null?void 0:T.value.trim())??"",10);return{Enabled:l.checked,Hostname:((C=o("tls-host"))==null?void 0:C.value.trim())??"",CertSource:(($=o("tls-source"))==null?void 0:$.value)??"internal",CertFile:((V=o("tls-cert"))==null?void 0:V.value.trim())??"",KeyFile:((Q=o("tls-key"))==null?void 0:Q.value.trim())??"",HTTPSPort:Number.isFinite(p)?p:443,BindAddr:((ue=a.config.TLS)==null?void 0:ue.BindAddr)??"",ImageRef:((Ie=a.config.TLS)==null?void 0:Ie.ImageRef)??""}}function vt(t,a){F[t]=[a]}async function $t(t,a){if(!m[t]){m[t]=a,B[t]=null,z();try{await $n(t,a)}catch(o){B[t]=`${a} failed: ${he(o)}${Le(o)}`}m[t]=null,await g()}}async function wt(t){if(m[t])return;m[t]="create",B[t]=null,F[t]=["starting…"],z();let a;try{a=await wn(t)}catch(o){B[t]=`${he(o)}${Le(o)}`,F[t]=[],m[t]=null,z();return}S==null||S(),S=Xe(a.targetId,o=>{if(i)return;const l=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(F[t]=[...(F[t]??[]).filter(T=>T!=="starting…"),l],!!o.err||o.stepId===Jn&&!!o.done){S==null||S(),S=null,m[t]=null,o.err&&(B[t]="Provisioning failed — see the log below."),g();return}z()})}async function kt(t){const a=q(t);if(!(!a||!await Be({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await yn(t)}catch(l){B[t]=he(l),z();return}await g()}}async function Tt(t){if(t){L[t]=null;try{await pn(t)}catch(a){L[t]=he(a),z();return}await g()}}function St(t){const a=q(t);if(!a)return;const o=new Set((a.networks??[]).map($=>$.chainId)),l=(r==null?void 0:r.presets)??[],p=l.filter($=>!o.has($.chainId)),T=l.filter($=>o.has($.chainId)),C=((r==null?void 0:r.targets)??[]).some($=>$.id===a.placement.targetId&&$.hasDevnet);re(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${p.map($=>`
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
        ${T.length?`<p class="muted small">Already fronted: ${n(T.map($=>$.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,$=>{if($==="cancel"){Y();return}if($==="custom"){Ct(t);return}if($.startsWith("preset:")){const V=Number.parseInt($.slice(7),10),Q=l.find(ue=>ue.chainId===V);Y(),Q!=null&&Q.devnet?Pt(t,V,C):et(t,V)}})}function Ct(t){var a;re(`
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
      `,o=>{if(o==="cancel"){Y();return}if(o!=="add")return;const l=document.getElementById("custom-chain-id"),p=document.getElementById("custom-chain-err"),T=Number.parseInt((l==null?void 0:l.value.trim())??"",10);if(!Number.isFinite(T)||T<=0){p&&(p.className="error small"),p&&(p.textContent="A chain id is a positive whole number.");return}Y(),et(t,T)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function et(t,a){const o=q(t);if(!o)return;const l=we(o),p=l.Networks??[];p.some(T=>T.ChainID===a)||(p.push({ChainID:a,Upstreams:[]}),l.Networks=p,await xt(t,l)&&(z(),nt(t,a)))}async function xt(t,a){var T;const o={...a,Networks:(a.Networks??[]).filter(C=>C.Upstreams.length>0)};if(!await ke(t,o))return!1;const p=q(t);if(p)for(const C of a.Networks??[])C.Upstreams.length===0&&!(p.networks??[]).some($=>$.chainId===C.ChainID)&&(p.config.Networks=[...p.config.Networks??[],{ChainID:C.ChainID,Upstreams:[]}],p.networks=[...p.networks??[],{chainId:C.ChainID,name:((T=((r==null?void 0:r.presets)??[]).find($=>$.chainId===C.ChainID))==null?void 0:T.name)??`Chain ${C.ChainID}`,path:`/${p.config.ProjectID}/evm/${C.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function Pt(t,a,o){const l=q(t);if(!l)return;if(!o){re(`
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
        `,()=>Y());return}const p=we(l),T=p.Networks??[],C={ID:"devnet",Kind:"managed-devnet",TargetID:l.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},$=T.find(V=>V.ChainID===a);$?$.Upstreams.push(C):T.push({ChainID:a,Upstreams:[C]}),p.Networks=T,await ke(t,p,"Adding the devnet")}async function Et(t,a){const o=q(t);if(!o||!Number.isFinite(a))return;const l=X(o,a);if(!await Be({title:`Remove ${(l==null?void 0:l.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(l==null?void 0:l.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const T=we(o);T.Networks=(T.Networks??[]).filter(C=>C.ChainID!==a),await ke(t,T,"Removing the network")}function tt(t){const a=t.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function It(t){const a=tt(t);if(!a)return;const o=q(a.gid);if(!o)return;const l=we(o),p=(l.Networks??[]).find($=>$.ChainID===a.chainId);if(!p)return;const T=p.Upstreams.findIndex(($,V)=>($.ID||`${a.chainId}-${V}`)===a.upstreamId);T<0||!await Be({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(p.Upstreams.splice(T,1),await ke(a.gid,l,"Removing the endpoint"))}function nt(t,a){const o=q(t);if(!o||!Number.isFinite(a))return;const l=((r==null?void 0:r.sources)??[]).filter($=>$.chainId===a),p=X(o,a),T=new Set(((p==null?void 0:p.upstreams)??[]).filter($=>$.kind!=="external").map($=>`${$.kind}|${$.targetId??""}`)),C=l.filter($=>!T.has(`${$.kind}|${$.targetId}`));re(`
        <h2>Add an endpoint for ${n((p==null?void 0:p.name)??`chain ${a}`)}</h2>
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
      `,$=>{if($==="cancel"){Y();return}if($==="known-set"){At(t,a);return}if($==="manual"){Bt(t,a);return}if($.startsWith("source:")){const[,V,Q]=$.split(":");Y(),Rt(t,a,V,Q)}})}async function Rt(t,a,o,l){const p=q(t);if(!p)return;const T=we(p),C=T.Networks??[],$={ID:`${o==="managed-devnet"?"devnet":"node"}-${l}`,Kind:o,TargetID:l,Endpoint:"",Local:!0,RecentOnly:!1},V=C.find(Q=>Q.ChainID===a);V?V.Upstreams.push($):C.push({ChainID:a,Upstreams:[$]}),T.Networks=C,await ke(t,T,"Adding the endpoint")}function Lt(t){const a=[...t].sort((p,T)=>(p.latencyMs??1e9)-(T.latencyMs??1e9)),o=a.slice(0,3),l=a.find(p=>p.url.startsWith("wss://")||p.url.startsWith("ws://"));return l&&!o.some(p=>p.url===l.url)&&(o.length===3&&o.pop(),o.push(l)),new Set(o.map(p=>p.url))}async function At(t,a){let o;try{o=await Sn(t,a)}catch($){re(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(he($))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Y());return}if(i)return;const l=o.endpoints??[],p=l.filter($=>!$.alreadyAdded).map($=>$.url),T=new Set(l.map($=>$.provider)).size,C=l.map($=>{const V=[$.websocket?'<span class="t ws">websocket</span>':"",$.archive?'<span class="t ar">archive</span>':"",$.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n($.url)}</code>
                  <span class="muted small">${n($.provider)}</span> ${V}</li>`}).join("");re(`<h2>Endpoints for chain ${a}</h2>
       ${l.length?`<p class="muted small">${T} providers valve has measured, in the order the gateway
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
         <button class="btn" data-modal-action="add"${p.length?"":" disabled"}>
           ${p.length?`Add ${p.length}`:"Nothing to add"}</button>
       </div>`,$=>{Y(),$==="add"&&Ge(t,a,p),$==="discover"&&Nt(t,a)})}async function Nt(t,a){re(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,C=>{C==="cancel"&&Y()});let o;try{o=await Tn(a)}catch(C){const $=He();if($){const V=document.createElement("p");V.className="error small",V.textContent=`Could not discover endpoints: ${he(C)}`,$.appendChild(V)}return}if(i)return;const l=(o.endpoints??[]).filter(C=>C.status==="live"||C.status==="unprobed"),p=(o.endpoints??[]).filter(C=>C.status==="rejected"),T=Lt(l);re(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${l.length?`<p class="muted small">${l.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${l.map(C=>{const $=T.has(C.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(C.url)}"${$}>
                       <span><code>${n(C.url)}</code></span>
                       <span class="muted small">${C.status==="live"?`answered in ${C.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${p.length?`<details class="rpc-rejected">
                 <summary class="muted small">${p.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${p.map(C=>`<li class="muted small"><code>${n(C.url)}</code> — ${n(C.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${l.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,C=>{if(C==="cancel"){Y();return}if(C==="add"){const $=He(),V=$?Array.from($.querySelectorAll('input[type="checkbox"]:checked')).map(Q=>Q.value):[];Y(),Ge(t,a,V);return}})}function Bt(t,a){var o;re(`
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
      `,l=>{if(l==="cancel"){Y();return}if(l!=="add")return;const p=document.getElementById("manual-endpoint"),T=document.getElementById("manual-recent"),C=document.getElementById("manual-err"),$=(p==null?void 0:p.value.trim())??"";if(!/^(https?|wss?):\/\//i.test($)){C&&(C.className="error small",C.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Y(),Ge(t,a,[$],(T==null?void 0:T.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ge(t,a,o,l=!1){if(!o.length)return;const p=q(t);if(!p)return;const T=we(p),C=T.Networks??[];let $=C.find(Q=>Q.ChainID===a);$||($={ChainID:a,Upstreams:[]},C.push($));let V=1;for(const Q of $.Upstreams){const ue=/^public-\d+-(\d+)$/.exec(Q.ID??"");ue&&(V=Math.max(V,Number(ue[1])+1))}for(const Q of o)$.Upstreams.some(ue=>ue.Endpoint===Q)||$.Upstreams.push({ID:`public-${a}-${V++}`,Kind:"external",Endpoint:Q,Local:!1,RecentOnly:l});T.Networks=C,await ke(t,T,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function Ht(t,a){const o=tt(t);if(!o||!a||!await Be({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;m[o.gid]="reset",B[o.gid]=null,z();let p;try{p=await dn(a)}catch(T){B[o.gid]=`Reset failed: ${he(T)}${Le(T)}`,m[o.gid]=null,z();return}m[o.gid]=null,Dt(a,p),await g()}function Dt(t,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const l=a.report.Cascaded??[],p=a.report.CascadeSkipped??[];re(`
        <h2>Devnet on ${n(t)} reset</h2>
        <ul class="plain-list">${o.map(T=>`<li>${n(T)}</li>`).join("")}</ul>
        ${l.length?`<p class="ok">Restarted in front of it: ${n(l.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${p.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(p.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Y())}function Ut(t){const a=q(t);if(!a)return;re(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(t)}</button>
        </div>
      `,p=>{if(p==="cancel"||p==="close"){Y(),g();return}p==="confirm"&&Mt(t)});const o=document.getElementById("wipe-confirm-input"),l=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{l&&(l.disabled=o.value.trim()!==t)}),o==null||o.focus()}async function Mt(t){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await kn(t)}catch(l){const p=He();if(p){const T=document.createElement("p");T.className="error small",T.textContent=`Wipe failed: ${he(l)}${Le(l)}`,p.appendChild(T)}a&&(a.disabled=!1,a.textContent=`Wipe ${t}`);return}re(`
        <h2>${n(t)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Y(),g()})}function at(t,a){return!a.some(o=>{var l;return((l=o.placement)==null?void 0:l.targetId)===t})}function Ot(){var T;const t=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=t.filter(C=>at(C.id,a)),l=new Set(a.map(C=>C.id));if(t.length===0){re(`
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
        `,()=>Y());return}const p=l.has("default")?"":"default";re(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(p)}" placeholder="edge" />
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
      `,C=>{if(C==="cancel"){Y();return}C==="create"&&qt()}),(T=document.getElementById("new-gw-id"))==null||T.focus()}async function qt(){const t=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),l=document.getElementById("new-gw-err"),p=(t==null?void 0:t.value.trim())??"",T=(a==null?void 0:a.value)??"",C=Number.parseInt((o==null?void 0:o.value.trim())??"",10),$=V=>{l&&(l.className="error small",l.textContent=V)};if(!p){$("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!T){$("Pick the machine it runs on.");return}try{await hn({id:p,placement:{targetId:T,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(C)?C:4e3,Networks:[]}})}catch(V){$(he(V));return}Y(),await g()}async function Ft(t,a){const o=await De(a),l=t.textContent;t.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(t.textContent=l)},1500)}function he(t){return t instanceof Error?t.message:String(t)}function Le(t){return t instanceof $e&&t.hint?` — ${t.hint}`:""}return()=>{i=!0,S==null||S(),Y()}}function Xn(s,i){if(s.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=s.filter(m=>!m.wire);if(r.length>0){const m=r.map(F=>F.id);return{level:"attention",sentence:m.length===1?"1 machine still needs setup.":`${m.length} machines still need setup.`,machines:m}}const e=i.networks??[],h=m=>{const B=e.find(F=>F.ChainID===m);return B?B.Name:`chain ${m}`},w=ea(s.map(m=>h(m.wire.ChainID))),R=s.length===1?"machine":"machines";return{level:"ok",sentence:`All ${s.length} ${R} healthy — ${ta(w)}.`,machines:[]}}function Qn(s,i){const r=i.machines.length?` <span class="verdict-machines">${i.machines.map(e=>`<a href="#/setup/${encodeURIComponent(e)}">${n(e)}</a>`).join(" ")}</span>`:"";s.innerHTML=`
    <div class="verdict-line verdict-${i.level}">
      ${U(i.level==="ok"?"OK":"Attention",i.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(i.sentence)}</strong>${r}
    </div>
  `}function ea(s){return[...new Set(s)]}function ta(s){return s.length<=1?s[0]??"":s.length===2?`${s[0]} and ${s[1]}`:`${s.slice(0,-1).join(", ")} and ${s[s.length-1]}`}const na="local";function aa(s){let i=!1,r=!1,e="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ce()}
  `;const w=s.querySelector("#targets-body");ye(s,(u,g)=>{j(u,g)}),R();async function R(){try{const[u,g,x]=await Promise.all([xe(),Ce(),Kt()]);if(i)return;e=x.os,B(u,g)}catch(u){if(i)return;w.innerHTML=`<p class="error">Failed to load machines: ${n(String(u))}</p>`}}function m(){h&&B(h.targets,h.catalog)}function B(u,g){h={targets:u,catalog:g};const x=e==="linux",N=[...u].sort((G,z)=>(G.mode==="local"?-1:0)-(z.mode==="local"?-1:0)),q=N.length?`<div class="card-grid">${N.map(G=>sa(G,g,G.mode!=="local"||x,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=u.some(G=>G.mode==="local");w.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${q}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${F(x,X)}
        ${r?oa():""}
      </section>
    `;const ee=w.querySelector("#fleet-verdict");ee&&Qn(ee,Xn(u,g))}function F(u,g){const x=`
      <div class="card">
        <h3>A server over SSH ${U("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${u?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${u?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,N=u?`
        <div class="card">
          <h3>This machine ${U("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${n(e)})`:""} ${U("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return g?`<div class="card-grid card-grid-wide">${x}</div>`:`<div class="card-grid card-grid-wide">${u?N+x:x+N}</div>`}async function j(u,g){var x;if(u==="add-local"){await E();return}if(u==="delete-target"){const N=g.dataset.id;if(!N||!await Be({title:"Remove machine",body:`Remove "${N}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await D(N);return}if(u==="toggle-ssh"){r=!r,S(),m(),r&&((x=s.querySelector("#ssh-host"))==null||x.focus());return}u==="add-ssh"&&await O()}async function E(){S();try{await st({id:na,mode:"local"}),await R()}catch(u){L(u)}}async function D(u){try{await Gt(u),await R()}catch(g){L(g)}}async function O(){const u=s.querySelector("#ssh-host"),g=s.querySelector("#ssh-user"),x=s.querySelector("#ssh-key"),N=s.querySelector("#ssh-port"),q=s.querySelector("#ssh-id");if(!u||!g||!x||!N||!q)return;const X=u.value.trim(),ee=g.value.trim(),G=x.value.trim(),z=N.value.trim(),de=q.value.trim();if(S(),!X||!ee||!G){L(new Error("host, user, and key path are required"));return}const te=de||ra(X),ae={Host:X,User:ee,KeyPath:G};if(z){const le=Number.parseInt(z,10);if(!Number.isFinite(le)||le<=0){L(new Error("port must be a positive number"));return}ae.Port=le}const se=s.querySelector("#ssh-submit");se&&(se.disabled=!0,se.textContent="Connecting…");try{await st({id:te,mode:"ssh",ssh:ae}),r=!1,await R()}catch(le){L(le),se&&(se.disabled=!1,se.textContent="Add server")}}function L(u){let g=s.querySelector("#targets-error");g||(w.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),g=s.querySelector("#targets-error")),g.textContent=String(u instanceof Error?u.message:u)}function S(){var u;(u=s.querySelector("#targets-error"))==null||u.remove()}return()=>{i=!0}}function sa(s,i,r,e){const h=s.wire,w=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:w;let m;if(!h&&!r)m=`${U("can't run a node","warn")} ${U(e||"not Linux","neutral")}`;else if(!h)m=U("not set up","neutral");else{const B=i.networks.find(j=>j.ChainID===h.ChainID),F=B?B.Name:`chain ${h.ChainID}`;m=`${U(F,"ok")} ${U(h.ExecID,"neutral")} ${U(h.BeaconID,"neutral")}${h.Archive?" "+U("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${R}</p>
      <p>${m}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(s.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function oa(){return`
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
  `}function ra(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ia=document.querySelector("#app"),{contentEl:ca,setActiveNav:la}=Pn(ia);let ge=null;function da(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[r,e]=i;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:e?decodeURIComponent(e):void 0}:{screen:r??"targets"}}function Te(s){const i=document.createElement("div");return ca.replaceChildren(i),s(i)}function pt(){if(ge){try{ge()}catch{}ge=null}const{screen:s,id:i}=da();switch(la(s),s){case"machine":if(!i){location.hash="#/targets";return}ge=Te(r=>jn(r,i));break;case"setup":case"dash":case"logs":case"services":if(!i){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(i)}`;return;case"security":if(!i){location.hash="#/targets";return}ge=Te(r=>Wn(r,i));break;case"diag":if(!i){location.hash="#/targets";return}ge=Te(r=>An(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}ge=Te(r=>Ln(r,i));break;case"rpc":ge=Te(r=>Zn(r));break;case"settings":ge=Te(r=>Gn(r));break;case"targets":default:ge=Te(r=>aa(r));break}}window.addEventListener("hashchange",pt);pt();
