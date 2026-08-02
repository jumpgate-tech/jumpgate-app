var hn=Object.defineProperty;var fn=(s,i,r)=>i in s?hn(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var qe=(s,i,r)=>fn(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))t(h);new MutationObserver(h=>{for(const $ of h)if($.type==="childList")for(const P of $.addedNodes)P.tagName==="LINK"&&P.rel==="modulepreload"&&t(P)}).observe(document,{childList:!0,subtree:!0});function r(h){const $={};return h.integrity&&($.integrity=h.integrity),h.referrerPolicy&&($.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?$.credentials="include":h.crossOrigin==="anonymous"?$.credentials="omit":$.credentials="same-origin",$}function t(h){if(h.ep)return;h.ep=!0;const $=r(h);fetch(h.href,$)}})();function gt(){return z("/api/host")}function xe(){return z("/api/catalog")}function Ce(){return z("/api/targets")}function dt(s){return z("/api/targets",{method:"POST",headers:be,body:JSON.stringify(s)})}function mn(s){return z(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function bn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function yn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:be,body:JSON.stringify(i)})}function Qe(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function gn(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function vn(s,i=200){return z(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function $n(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function ut(s,i){const r=i===void 0?{}:{lines:i};return z(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:be,body:JSON.stringify(r)})}function wn(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function kn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:be,body:JSON.stringify({Confirm:i})})}function Tn(s){return z(`/api/targets/${encodeURIComponent(s)}/du`)}function Sn(s){return z(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function xn(s){return z(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Cn(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function En(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function Pn(s){return z(`/api/targets/${encodeURIComponent(s)}/containers`)}function In(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function Rn(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:i})}),t=await r.text();let h=null;try{h=t?JSON.parse(t):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const $=h&&typeof h=="object"&&typeof h.error=="string"?h.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,$)}function Ln(s,i){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function An(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:be}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const h=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,h)}function Nn(s,i,r){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:be,body:JSON.stringify(r)})}function et(){return z("/api/gateways")}async function Bn(s){await z(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function Hn(s){return z("/api/gateways",{method:"POST",headers:be,body:JSON.stringify(s)})}function Dn(s){return z(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function Mn(s){return z(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function Un(s){return z(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function On(s,i=!1){const r=i?"?refresh=1":"";return z(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function qn(s){return z(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function Fn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:be,body:JSON.stringify(i)})}function jn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function Wn(s){return z(`/api/gateways/${encodeURIComponent(s)}/trust-cert`,{method:"POST"})}function _n(s){return z(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function Kn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:be,body:JSON.stringify({Confirm:s})}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const h=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,h)}function Vn(s){return z(`/api/chainlist/${s}`)}function Gn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function zn(){return z("/api/settings")}function Jn(s){return z("/api/settings",{method:"PUT",headers:be,body:JSON.stringify(s)})}class we extends Error{constructor(r,t,h,$){super(t);qe(this,"status");qe(this,"hint");qe(this,"code");this.name="ApiError",this.status=r,this.hint=h,this.code=$}}const be={"Content-Type":"application/json"};async function z(s,i){const r=await fetch(s,i);if(!r.ok){let h=r.statusText||`HTTP ${r.status}`,$,P;try{const f=await r.json();f&&typeof f.error=="string"&&f.error&&(h=f.error),f&&typeof f.hint=="string"&&f.hint&&($=f.hint),f&&typeof f.code=="string"&&f.code&&(P=f.code)}catch{}throw new we(r.status,h,$,P)}if(r.status===204)return;const t=await r.text();return t?JSON.parse(t):void 0}const pt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function de(s,i){const r=s&&i&&i!==pt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(pt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function Yn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{const $=h==="machine"?"targets":h==="home"?"rpc":h;for(const P of r)P.classList.toggle("active",P.dataset.nav===$)}}}function ce(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function Zn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function Xn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),t=i%60;if(r===0)return`~${t}m`;if(r<48)return`~${r}h ${t}m`;const h=Math.floor(r/24),$=r%24;return`~${h}d ${$}h`}function D(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function $e(s){return`<span class="dot dot-${s}"></span>`}const ht=["B","KB","MB","GB","TB","PB"];function Se(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<ht.length-1;)i/=1024,r++;const t=i<10?2:i<100?1:0;return`${i.toFixed(t)} ${ht[r]}`}async function De(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ye(s,i){s.addEventListener("click",r=>{const t=r.target.closest("[data-action]");if(!t||!s.contains(t))return;const h=t.dataset.action;h&&i(h,t,r)})}function Ze(s,i,r){const t=i.find($=>$.value===r),h=i.map($=>`
      <li class="dropdown-option${$.value===r?" selected":""}" role="option"
          aria-selected="${$.value===r}" data-value="${n($.value)}">
        ${n($.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Ae(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function tt(s,i){s.addEventListener("click",h=>{const $=h.target,P=$.closest(".dropdown-trigger");if(P&&s.contains(P)){const N=P.closest(".dropdown"),q=!!N&&!N.classList.contains("open");Ae(s),N&&q&&(N.classList.add("open"),P.setAttribute("aria-expanded","true"));return}const f=$.closest(".dropdown-option");if(f&&s.contains(f)){const N=f.closest(".dropdown");Ae(s),i((N==null?void 0:N.dataset.dropdown)??"",f.dataset.value??"");return}Ae(s)});const r=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}const $=h.target;(!$.closest(".dropdown")||!s.contains($))&&Ae(s)},t=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}h.key==="Escape"&&Ae(s)};document.addEventListener("click",r),document.addEventListener("keydown",t)}const Ke="app-modal";let _e=null;function ie(s,i){Z();const r=document.createElement("div");r.className="modal-overlay",r.id=Ke,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",h=>{const $=h.target.closest("[data-modal-action]");$!=null&&$.dataset.modalAction?i($.dataset.modalAction):h.target===r&&i("cancel")});const t=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",t),_e=t,document.body.appendChild(r)}function Z(){var s;(s=document.getElementById(Ke))==null||s.remove(),_e&&(document.removeEventListener("keydown",_e),_e=null)}function He(){return document.querySelector(`#${Ke} .modal`)}function Be(s){return new Promise(i=>{var h;let r=!1;const t=$=>{r||(r=!0,Z(),i($))};ie(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,$=>t($==="confirm")),(h=document.querySelector(`#${Ke} [data-modal-action="confirm"]`))==null||h.focus()})}const ze=5e3,Qn=60;function ea(s,i){let r=!1,t=null,h=null,$=null,P=null;const f=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${de()}</div>`;const N=s.querySelector("#an-body");ye(s,(y,d)=>{var T;y==="toggle-endpoint"&&((T=d.closest(".an-endpoint"))==null||T.classList.toggle("expanded"))}),q();async function q(){try{t=((await et()).gateways??[]).find(d=>d.id===i)??null}catch(y){if(r)return;$=String(y instanceof Error?y.message:y),M();return}if(!r){if(!t){M();return}await j(),P=window.setInterval(()=>void j(),ze)}}async function j(){try{const y=await Un(i);if(r)return;I(y),h=y,$=null}catch(y){if(r)return;$=String(y instanceof Error?y.message:y)}M()}function I(y){if(!y.enabled||y.error)return;const d=f[f.length-1];d&&d.since!==y.since&&(f.length=0);const T=new Map;for(const R of y.networks??[])T.set(R.chainId,R.received);f.push({t:Date.now(),since:y.since,received:T}),f.length>Qn&&f.shift()}function M(){r||(N.innerHTML=O())}function O(){return $&&!h?`<h1>Analytics</h1><p class="error">${n($)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${L(t)}
      ${h?p(h):`<p class="muted">Reading the gateway's counters…</p>`}
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
        <div class="an-head-right muted small">${C()}</div>
      </div>
    `}function C(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const y=h.since?new Date(h.since):null;return y&&!Number.isNaN(y.getTime())?`totals since the gateway started, ${n(y.toLocaleString())}<br />re-read every ${ze/1e3}s`:`re-read every ${ze/1e3}s`}function p(y){return y.enabled?y.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(y.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:w(y)+le(y):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function w(y){const d=y.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(T=>S(T)).join("")}
      </section>
    `}function S(y){const d=y.methods??[],T=y.endpoints??[],R=y.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${y.chainId}</span>
          <span class="band-name">${n(y.name)}</span>
          ${F(y)}
        </div>
        <div class="an-stats">
          ${B("Received",ce(y.received),"what clients asked this chain for")}
          ${B("Answered",ce(y.answered),"returned by one of your endpoints")}
          ${B("From cache",ce(y.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${B("Failed",ce(y.failed),"asked for and never answered",y.failed>0?"bad":"")}
        </div>
        ${te(y.chainId)}
        ${R?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ae("Method",d.map(H=>({label:H.method,l:H})))+ae("Endpoint",T.map(H=>({label:H.upstream,l:H})))+W(y)}
      </div>
    `}function B(y,d,T,R=""){return`
      <div class="an-stat${R?" an-stat-"+R:""}" title="${n(T)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(y)}</span>
      </div>
    `}function F(y){const d=X(y.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const T=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${T}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${T}s</span>
    </span>`}function X(y){if(f.length<2)return null;const d=f[0],T=f[f.length-1],R=(T.t-d.t)/1e3;if(R<=0)return null;const H=(T.received.get(y)??0)-(d.received.get(y)??0);return H<0?null:H/R}function te(y){if(f.length<3)return"";const d=[];for(let g=1;g<f.length;g++){const E=f[g-1],_=f[g],l=(_.t-E.t)/1e3,b=(_.received.get(y)??0)-(E.received.get(y)??0);d.push(l>0&&b>=0?b/l:0)}const T=Math.max(...d);if(T<=0)return"";const R=240,H=28,K=d.length>1?R/(d.length-1):R,m=d.map((g,E)=>`${(E*K).toFixed(1)},${(H-g/T*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${T.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${R} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${m}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(T.toFixed(2))} req/s</span>
      </div>
    `}function W(y){const d=[];return y.cached.count>0&&d.push(`${n(ce(y.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${y.cached.mean===null?"":`, in ${n(Ne(y.cached.mean))} on average`}.`),y.failedLatency.count>0&&y.failedLatency.mean!==null&&d.push(`The ${n(ce(y.failedLatency.count))} that failed took
         ${n(Ne(y.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function ae(y,d){return d.length===0?"":`
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
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(Ne(d.mean))}</td>
        <td>${J(d)}</td>
      </tr>
    `}function J(y){const d=y.buckets??[];if(d.length===0||y.count===0)return'<span class="muted small">—</span>';let T=0;const R=[];for(const K of d){const m=K.count-T;T=K.count,R.push({label:oe(K.le),n:Math.max(0,m)})}return R.reduce((K,m)=>K+m.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(R.filter(K=>K.n>0).map(K=>`${K.n} ${K.label}`).join(" · "))}">
        ${R.map((K,m)=>K.n===0?"":`<span class="an-band an-band-${Math.min(m,4)}" style="flex:${K.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(se(R))}</span>
    `}function se(y){for(let d=y.length-1;d>=0;d--)if(y[d].n>0)return`slowest ${y[d].label}`;return""}function oe(y){if(y==="+Inf")return"30s or more";const d=Number(y);return Number.isFinite(d)?`under ${Ne(d)}`:`under ${y}`}function le(y){const d=y.endpoints??[];return`
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
    `}function Y(y){const d=y.errors??[],T=d.reduce((H,K)=>H+K.count,0),R=d.length>0;return`
      <tr class="an-endpoint${R?" expandable":""}" ${R?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(y.upstream)}</code>
          ${y.chainId?`<span class="muted small">chain ${y.chainId}</span>`:""}
          ${y.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ce(y.requests)}</td>
        <td class="an-num${T>0?" bad":""}">${T>0?ce(T):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${y.headLag>0?ce(y.headLag):'<span class="muted">0</span>'}</td>
        <td>${me(y)}</td>
      </tr>
      ${R?ge(y,d):""}
    `}function me(y){const d=[];return y.scored?(d.push(y.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(y.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(y.score.toFixed(3))}</span>`),y.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ce(y.primarySwitches)} switches</span>`),y.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Ne(y.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(y,d){return`
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
    `}return()=>{r=!0,P!==null&&window.clearInterval(P)}}function Ne(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}function ta(s,i){let r=!1,t=null,h=null,$=!1,P=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${de()}</div>`;const f=s.querySelector("#diag-body"),N=s.querySelector("#diag-footer");ye(s,(p,w)=>{var S;if(p==="run")j();else if(p==="toggle")(S=w.closest(".check-item"))==null||S.classList.toggle("expanded");else if(p==="copy"){const B=w.dataset.copy;B&&C(w,B)}}),q();async function q(){let p,w;try{const[B,F]=await Promise.all([Ce(),xe()]);p=B.find(X=>X.id===i),w=F}catch(B){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(B))}</p>`;return}if(r)return;if(!p){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const S=w==null?void 0:w.networks.find(B=>B.ChainID===p.wire.ChainID);S&&(N.innerHTML=de(S.Name,S.LearnURL));try{t=await En(i),P=!0}catch(B){h=String(B instanceof Error?B.message:B)}r||I()}async function j(){$=!0,h=null,I();try{t=await Cn(i),P=!0}catch(p){h=String(p instanceof Error?p.message:p)}$=!1,r||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${$?"disabled":""}>${$?"Running…":"Run diagnostics"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${M()}
    `}function M(){if(!P&&!h)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const p=new Date(t.at).toLocaleString(),w=t.failedId?`<p><strong>Failed at: ${n(O(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(p)} — trigger: ${n(t.trigger)}</p>
      ${w}
      <ul class="check-list">${t.items.map(L).join("")}</ul>
    `}function O(p){var w;return((w=t==null?void 0:t.items.find(S=>S.ID===p))==null?void 0:w.Title)??p}function L(p){const w=p.Status==="pass"?"ok":p.Status==="fail"?"bad":p.Status==="warn"?"warn":"neutral",S=p.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${S?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${D(S?"failed here":p.Status,w)}
          <strong>${n(p.Title)}</strong>
          <span class="muted small check-detail-inline">${n(p.Detail)}</span>
        </button>
        <div class="check-body">
          <details${S?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(p.Why)}</p>
          </details>
          ${p.Fix?`
                <details${S?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(p.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(p.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function C(p,w){const S=await De(w),B=p.textContent;p.textContent=S?"Copied!":"Copy failed",setTimeout(()=>{r||(p.textContent=B)},1500)}return()=>{r=!0}}const na=85,Je={exec:"Execution",beacon:"Beacon"};function aa(s,i){let r=!1,t=null,h=null,$=null,P=null,f=null,N=null,q=null,j=null;const I={exec:null,beacon:null};let M=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${de()}</div>`;const O=s.querySelector("#dash-body"),L=s.querySelector("#dash-footer");O.addEventListener("click",d=>{const T=d.target.closest("[data-action]");if(!T||!O.contains(T))return;const R=T.dataset.action;if(R==="svc-action"){const H=T.dataset.svc,K=T.dataset.kind;H&&K&&Y(H,K)}else if(R==="open-clear"){const H=T.dataset.svc;H&&ge(H)}else if(R==="copy"){const H=T.dataset.copy;H&&me(T,H)}else R==="retry-du"?p():R==="retry-endpoints"&&w()}),C();async function C(){let d,T;try{const[H,K]=await Promise.all([Ce(),xe()]);d=H.find(m=>m.id===i),T=K}catch(H){if(r)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(r)return;if(!d){O.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const R=T==null?void 0:T.networks.find(H=>H.ChainID===d.wire.ChainID);R&&(L.innerHTML=de(R.Name,R.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',t=gn(i,H=>{r||(S(H),h=H,$=H,B())}),p(),w()}async function p(){N=null;try{f=await Tn(i)}catch(d){f=null,N=String(d instanceof Error?d.message:d)}r||B()}async function w(){j=null;try{q=await Sn(i)}catch(d){q=null,j=String(d instanceof Error?d.message:d)}r||B()}function S(d){if(!h)return;const T=(new Date(d.at).getTime()-new Date(h.at).getTime())/1e3,R=d.execHead-h.execHead;if(T>0&&R>=0){const H=R/T;P=P===null?H:P*.7+H*.3}}function B(){if(!$)return;const d=$;O.innerHTML=`
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
    `}function F(d){return!d.execActive&&!d.beaconActive?D("Node not running","bad"):d.execSyncing||d.beaconDistance>0?D("Syncing","warn"):D("Running · synced","ok")}function X(d){const R=d.refHead>0?d.refHead-d.execHead:null,H=R!==null&&R>0&&P&&P>0?Xn(R/P):R!==null&&R<=0?"caught up":"—";return{lag:R,eta:H}}function te(d){const{lag:T,eta:R}=X(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?D("syncing","warn"):d.execHead===0?D("no data","neutral"):D("synced","ok"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ce(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${T!==null?ce(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${T!==null?ce(Math.max(T,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${R}</dd></div>
        </dl>
      </div>
    `}function W(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?D("no data","neutral"):d.beaconDistance===0?D("synced","ok"):D("syncing","warn"):D("stopped","bad")}</p>
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
    `}function ue(d){const T=d.diskUsedPct>=na,R=`
      <div class="meter"><div class="meter-fill ${T?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${Zn(d.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="error small">${n(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${T?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="muted">Loading…</p>
        </div>
      `;const H=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,K=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:m,eta:g}=X(d),E=m!==null&&m>0&&P!==null&&P>0;return`
      <div class="card ${T?"card-warn":""}">
        <h3>Storage</h3>
        ${R}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Se(f.ExecBytes)} of ~${Se(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${E?`<p class="muted small">Estimated time remaining: ${n(g)}</p>`:""}
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
      `;if(!q)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=q,T=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",R=d.Access==="ssh"?`
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
        ${T}
        ${R}
      </div>
    `}function se(d,T){const R=Je[d],H=I[d],K=(m,g,E)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${m}" ${H!==null||E?"disabled":""}>${H===m?le():n(g)}</button>`;return`
      <div class="service-row">
        <span>${n(R)} ${T?D("active","ok"):D("down","bad")}</span>
        <div class="service-actions">
          ${K("start","Start",T)}
          ${K("stop","Stop",!T)}
          ${K("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${d}" ${H!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(d){return`
      <div class="card">
        <h3>Services</h3>
        ${se("exec",d.execActive)}
        ${se("beacon",d.beaconActive)}
        ${M?`<p class="error small">${n(M)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function le(){return'<span class="spinner" aria-label="working"></span>'}async function Y(d,T){if(I[d]===null){I[d]=T,M=null,B();try{await wn(i,d,T)}catch(R){M=`${Je[d]} ${T} failed: ${R instanceof Error?R.message:String(R)}`}I[d]=null,r||B()}}async function me(d,T){const R=await De(T),H=d.textContent;d.textContent=R?"Copied!":"Copy failed",setTimeout(()=>{r||(d.textContent=H)},1500)}function ge(d){const T=Je[d],R=f?Se(d==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";ie(`
        <h2>Clear ${n(T)} data</h2>
        <p class="error">
          This stops the ${n(T.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(R)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,m=>{if(m==="cancel"){Z();return}m==="confirm"&&y(d)});const H=document.getElementById("clear-confirm-input"),K=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{K&&(K.disabled=H.value.trim()!==d)}),H==null||H.focus()}async function y(d){const T=document.getElementById("clear-confirm-btn");T&&(T.disabled=!0,T.textContent="Clearing…");try{await kn(i,d),Z(),p()}catch(R){const H=He();if(H){const K=document.createElement("p");K.className="error small",K.textContent=`Clear failed: ${R instanceof Error?R.message:String(R)}`,H.appendChild(K)}T&&(T.disabled=!1,T.textContent="Clear and resync")}}return()=>{r=!0,t==null||t(),Z()}}const ft=500,mt="valve-node-app.explain-consent";function sa(s,i){let r=!1,t=null;const h=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${de()}</div>
  `;const $=s.querySelector("#logs-body"),P=s.querySelector("#logs-footer");ye(s,C=>{C==="explain"&&j()}),f();async function f(){let C,p;try{const[S,B]=await Promise.all([Ce(),xe()]);C=S.find(F=>F.id===i),p=B}catch(S){if(r)return;$.innerHTML=`<p class="error">Failed to load target: ${n(String(S))}</p>`;return}if(r)return;if(!C){$.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){$.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const w=p==null?void 0:p.networks.find(S=>S.ChainID===C.wire.ChainID);w&&(P.innerHTML=de(w.Name,w.LearnURL));try{const S=await vn(i,200);if(r)return;h.push(...S)}catch(S){if(r)return;$.innerHTML=`<p class="error">Failed to load logs: ${n(String(S))}</p>`;return}N(),t=$n(i,S=>{r||(h.push(S),h.length>ft&&h.splice(0,h.length-ft),N())})}function N(){const C=h.filter(w=>w.severity==="error"||w.severity==="critical");$.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${h.map(q).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${D(String(C.length),C.length?"bad":"neutral")}</h2>
          <div class="log-lines">${C.length?C.slice().reverse().map(q).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const p=$.querySelector(".log-lines");p&&(p.scrollTop=p.scrollHeight)}function q(C){const p=C.severity||"info",w=C.learnUrl?` <a href="${n(C.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(p)}">
        <span class="log-time">${n(new Date(C.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(C.unit)}</span>
        <span class="log-sev">${n(p)}</span>
        <span class="log-text">${n(C.line)}</span>
        ${C.explain?`<div class="log-explain">${n(C.explain)}${w}</div>`:""}
      </div>
    `}async function j(){const C=h.filter(w=>w.severity==="error"||w.severity==="critical").map(w=>w.line).slice(-40);if(!(localStorage.getItem(mt)==="1")){I(C);return}await M(C)}function I(C){const p=C.length?`<pre class="explain-excerpt">${C.map(w=>n(w)).join(`
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
    `,w=>{w==="proceed"?(localStorage.setItem(mt,"1"),L(),M(C)):L()})}async function M(C){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const p=C.length?await ut(i,C):await ut(i);if(r)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(p.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${p.sentExcerpt.map(w=>n(w)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,w=>{w==="close"&&L()})}catch(p){if(r)return;if(p instanceof we&&p.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,w=>{w==="close"&&L()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(p instanceof Error?p.message:String(p))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,w=>{w==="close"&&L()})}}function O(C,p){L();const w=document.createElement("div");w.className="modal-overlay",w.id="explain-modal",w.innerHTML=`<div class="modal">${C}</div>`,w.addEventListener("click",S=>{const B=S.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&p(B.dataset.modalAction),S.target===w&&p("cancel")}),document.body.appendChild(w)}function L(){var C;(C=document.getElementById("explain-modal"))==null||C.remove()}return()=>{r=!0,t==null||t(),L()}}const oa="run",ra={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},ia={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function ca(s,i){let r=!1,t=null,h=null;const $={devnet:null},P={devnet:null},f={devnet:[]};let N=null;const q={devnet:!1};let j=null;const I={devnet:null},M={devnet:null};s.innerHTML=`
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
    ${de()}
  `;const O=s.querySelector("#services-body");ye(s,(l,b)=>{ge(l,b)}),L();async function L(){try{const l=await Pn(i);if(r)return;t=l,h=null}catch(l){if(r)return;t=null,h=E(l)}p()}function C(l){return t==null?void 0:t.services.find(b=>b.id===l)}function p(){if(!r){if(h){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(h)}</p>`;return}if(!t){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${w(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(S).join("")}
      </div>
    `}}function w(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${n(l.flavor)}${l.serverVersion?` ${n(l.serverVersion)}`:""} · reachable</p>`;const b=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(b)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${n(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function S(l){const b=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(l.label)}</h2>
          ${B(l)}
        </div>
        <p class="muted small">${n(ra[l.id]??"")}</p>

        ${l.error?F(l):""}
        ${l.blocked?`<div class="banner banner-warn">${n(l.blocked)}</div>`:""}
        ${b.map(A=>`<div class="banner banner-warn">${n(A)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${n(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${X(l)}

        ${te(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(A=>W(l,A)).join("")}
        </div>
        ${P[l.id]?`<p class="error small">${n(P[l.id])}</p>`:""}
        ${ae(l)}

        ${ue(l)}
      </div>
    `}function B(l){switch(l.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function F(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(l.error??"")}</div>
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function X(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const b=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${b}.</p>`}function te(l){const b=l.endpoints??[];return b.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":b.map(A=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(A.label)}</span>
          <code class="endpoint-url">${n(A.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(A.url)}">Copy</button>
        </div>`).join("")}function W(l,b){const A=ia[b];if(!A)return"";const V=$[l.id],Q=b==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:A.label;return`
      <button class="${A.className}" data-action="svc-${b}" data-svc="${n(l.id)}"
              title="${n(A.title)}" ${V?"disabled":""}>
        ${V===b?'<span class="spinner" aria-label="working"></span>':n(Q)}
      </button>
    `}function ae(l){const b=f[l.id]??[];return b.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(b.join(`
`))}</pre>
      </div>
    `}function ue(l){const b=q[l.id],A=J(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(l.id)}">
            ${b?"Close":"Edit"}
          </button>
        </div>
        ${b?se():`<p class="small">${A}</p>`}
        ${I[l.id]?`<p class="error small">${n(I[l.id])}</p>`:""}
        ${M[l.id]?`<p class="muted small">${n(M[l.id])}</p>`:""}
      </div>
    `}function J(l){const b=l.devnet;return b?`Chain ${b.ChainID} · a block every ${n(b.BlockTime)} · JSON-RPC on ${n(b.BindAddr)}:${b.HTTPPort} · WebSocket on ${n(b.BindAddr)}:${b.WSPort}`:"—"}function se(l){return oe()}function oe(){const l=j;return l?`
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
    `:""}function le(){q.devnet&&j&&(j.BlockTime=Y("#dev-blocktime",j.BlockTime),j.HTTPPort=me("#dev-http",j.HTTPPort),j.WSPort=me("#dev-ws",j.WSPort),j.BindAddr=Y("#dev-bind",j.BindAddr))}function Y(l,b){const A=s.querySelector(l);return A?A.value.trim():b}function me(l,b){const A=s.querySelector(l);if(!A)return b;const V=Number.parseInt(A.value.trim(),10);return Number.isFinite(V)?V:b}async function ge(l,b){const A=b.dataset.svc??"";switch(l){case"refresh":await L();return;case"copy":b.dataset.copy&&await g(b,b.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await y(A,l.slice(4));return;case"svc-create":case"svc-recreate":await d(A);return;case"svc-wipe":H(A);return;case"toggle-config":T(A);return;case"save-config":await R(A);return;default:return}}async function y(l,b){if(!$[l]){$[l]=b,P[l]=null,p();try{await In(i,l,b)}catch(A){P[l]=`${b} failed: ${E(A)}${_(A)}`}$[l]=null,await L()}}async function d(l){if(!$[l]){$[l]="create",P[l]=null,f[l]=["starting…"],p();try{await Ln(i,l)}catch(b){P[l]=`${E(b)}${_(b)}`,f[l]=[],$[l]=null,p();return}N==null||N(),N=Qe(i,b=>{if(r)return;const A=b.err?`${b.stepId}: ${b.err}`:b.line?`${b.stepId}: ${b.line}`:`${b.stepId}: done`;if(f[l]=[...(f[l]??[]).filter(Q=>Q!=="starting…"),A],!!b.err||b.stepId===oa&&!!b.done){N==null||N(),N=null,$[l]=null,b.err&&(P[l]="Provisioning failed — see the log below."),L();return}p()})}}function T(l){if(le(),q[l]=!q[l],I[l]=null,M[l]=null,q[l]){const b=C(l);b!=null&&b.devnet&&(j={...b.devnet})}p()}async function R(l){var V;le(),I[l]=null,M[l]=null;const b=j;if(!b)return;if(b.HTTPPort===b.WSPort){I[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",p();return}try{await Nn(i,l,b)}catch(Q){I[l]=E(Q),p();return}const A=((V=C(l))==null?void 0:V.status.State)==="running";q[l]=!1,M[l]=A?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await L()}function H(l){const b=C(l);if(!b)return;const A=(b.restartsOnWipe??[]).map(U=>{var re;return((re=C(U))==null?void 0:re.label)??U});ie(`
        <h2>Wipe ${n(b.label)}</h2>
        <p class="error">This deletes ${n(b.wipeDiscards)}</p>
        ${A.length?`<p>It also restarts what sits in front of it: ${n(A.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(l)}</button>
        </div>
      `,U=>{if(U==="cancel"||U==="close"){Z(),L();return}U==="confirm"&&K(l)});const V=document.getElementById("wipe-confirm-input"),Q=document.getElementById("wipe-confirm-btn");V==null||V.addEventListener("input",()=>{Q&&(Q.disabled=V.value.trim()!==l)}),V==null||V.focus()}async function K(l){const b=document.getElementById("wipe-confirm-btn");b&&(b.disabled=!0,b.textContent="Wiping…");let A;try{A=await Rn(i,l)}catch(V){const Q=He();if(Q){const U=document.createElement("p");U.className="error small",U.textContent=`Wipe failed: ${E(V)}${_(V)}`,Q.appendChild(U)}b&&(b.disabled=!1,b.textContent=`Wipe ${l}`);return}m(l,A)}function m(l,b){const A=C(l),V=ne=>{var Ee;return((Ee=C(ne))==null?void 0:Ee.label)??ne},Q=[];Q.push(b.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ne of b.report.VolumesRemoved??[])Q.push(`Volume ${ne} deleted.`);for(const ne of b.report.VolumesAbsent??[])Q.push(`Volume ${ne} was already gone.`);b.report.Recreated&&Q.push("Container re-created from your saved configuration.");const U=(b.report.Cascaded??[]).map(V),re=(b.report.CascadeSkipped??[]).map(V);ie(`
        <h2>${n((A==null?void 0:A.label)??l)} wiped</h2>
        <ul class="plain-list">${Q.map(ne=>`<li>${n(ne)}</li>`).join("")}</ul>
        ${U.length?`<p class="ok">Restarted in front of it: ${n(U.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${re.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(re.join(", "))}.</p>`:""}
        ${b.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(b.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ne=>{(ne==="close"||ne==="cancel")&&(Z(),L())})}async function g(l,b){const A=await De(b),V=l.textContent;l.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{r||(l.textContent=V)},1500)}function E(l){return l instanceof Error?l.message:String(l)}function _(l){return l instanceof we&&l.hint?` — ${l.hint}`:""}return()=>{r=!0,N==null||N(),Z()}}const Ye=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Fe=8545,je=5052,We=30303,la=[369,943,1],bt={369:"default",943:"practise here first"};function da(s,i){let r=!1;const t={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${de()}</div>`;const h=s.querySelector("#wizard-body"),$=s.querySelector("#wizard-footer");ye(s,(m,g)=>{me(m,g)}),tt(s,(m,g)=>{m==="exec-select"?t.execId=g:m==="beacon-select"&&(t.beaconId=g),f()}),s.addEventListener("change",m=>{const g=m.target;g instanceof HTMLInputElement&&(g.id==="data-dir-input"?(ge(),W()):g.id==="checkpoint-toggle"?(t.checkpoint=g.checked,f()):g.id==="exec-snapshot-toggle"&&(t.execSnapshot=g.checked,f()))}),P();async function P(){try{const[m,g]=await Promise.all([xe(),Ce()]);if(r)return;t.catalog=m;const E=g.find(_=>_.id===i);E!=null&&E.wire&&(t.chainId=E.wire.ChainID,t.execId=E.wire.ExecID,t.beaconId=E.wire.BeaconID,t.archive=E.wire.Archive,E.wire.ExecHTTPPort&&(t.execHTTPPort=String(E.wire.ExecHTTPPort)),E.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(E.wire.BeaconHTTPPort)),E.wire.ExecP2PPort&&(t.execP2PPort=String(E.wire.ExecP2PPort)),E.wire.RPCBindAddr&&(t.rpcBindAddr=E.wire.RPCBindAddr)),f()}catch(m){if(r)return;t.loadError=String(m instanceof Error?m.message:m),f()}}function f(){if(t.loadError){h.innerHTML=`<p class="error">Failed to load: ${n(t.loadError)}</p>`;return}t.catalog&&(h.innerHTML=`
      ${K(t.step)}
      ${q()}
    `,N())}function N(){var g;const m=(g=t.catalog)==null?void 0:g.networks.find(E=>E.ChainID===t.chainId);$.innerHTML=m?de(m.Name,m.LearnURL):de()}function q(){switch(t.step){case"network":return j();case"clients":return I();case"mode":return oe();case"review":return le();case"run":return Y()}}function j(){const m=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${la.map(E=>{const _=m.networks.find(A=>A.ChainID===E);if(!_)return"";const l=t.chainId===E,b=bt[E]?D(bt[E],E===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${E}" type="button">
          <h3>${n(_.Name)} <span class="muted">(chain ${E})</span></h3>
          ${b}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function I(){const m=t.catalog,g=m.networks.find(l=>l.ChainID===t.chainId);if(!g)return'<p class="error">Unknown network.</p>';(t.execId===null||!g.ExecClients.includes(t.execId))&&(t.execId=g.ExecClients[0]??null),(t.beaconId===null||!g.BeaconClients.includes(t.beaconId))&&(t.beaconId=g.BeaconClients[0]??null);const E=g.ExecClients.map(l=>ue(l,m)),_=g.BeaconClients.map(l=>ue(l,m));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${n(g.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${Ze("exec-select",E,t.execId)}
        </label>
        ${se(t.execId,m)}
        <label>
          Beacon client
          ${Ze("beacon-select",_,t.beaconId)}
        </label>
        ${se(t.beaconId,m)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function M(m){return m<=0?"—":m>=1?`~${m.toFixed(1)} TB`:`~${Math.round(m*1e3)} GB`}const O=1.1,L=.5,C="Valve reth snapshot",p="rough estimate";function w(m){return m.SnapshotSizeTB}function S(m){return m.SnapshotSizeTB*L}function B(m){return`<p class="muted small">${M(w(m))} is the measured size of Valve's reth snapshot for ${n(m.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function F(m){return{archive:w(m)*1e12*O,full:S(m)*1e12*O}}function X(m,g){if(!m)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${n(g)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${n(g)}</code>: ${n(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==g)return"";const E=F(m),_=t.freeBytes>=E.archive,l=t.freeBytes>=E.full,b=`<p class="muted small">Free at <code>${n(g)}</code>: <strong>${Se(t.freeBytes)}</strong> — archive ${_?"fits":"won't fit"} (${M(w(m))}, ${C}), full ${l?"fits":"won't fit"} (${M(S(m))}, ${p}).</p>`;let A="";return t.downgradeNote?A=`<p class="banner banner-warn">${n(t.downgradeNote)}</p>`:l||(A=`<p class="banner banner-warn">Neither full (${M(S(m))}, ${p}) nor archive (${M(w(m))}, ${C}) fits the free space here — choose a location with more room.</p>`),b+A}function te(m,g){if(t.downgradeNote=null,!m||t.freeBytes===null)return;const E=F(m);t.archive&&t.freeBytes<E.archive&&t.freeBytes>=E.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${g} for archive (${M(w(m))}, ${C}) — switched to Full (${M(S(m))}, ${p}). Pick a location with more room to run archive.`)}async function W(){var E;if(t.chainId===null)return;const m=(E=t.catalog)==null?void 0:E.networks.find(_=>_.ChainID===t.chainId),g=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,f();try{const{freeBytes:_}=await bn(t.targetId,g);if(r)return;t.freeBytes=_,t.probedPath=g,te(m,g)}catch(_){if(r)return;t.freeBytes=null,t.probedPath=g,t.diskError=String(_ instanceof Error?_.message:_)}t.diskProbing=!1,f()}function ae(m){return m?/^https?:\/\/.+/i.test(m)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ue(m,g){const E=g.clients.find(_=>_.id===m);return{value:m,label:E?`${E.id} — ${J(E.repo)}`:m}}function J(m){const g=m.split("/");return g.length>=4?g[3]:m}function se(m,g){const E=m?g.clients.find(l=>l.id===m):void 0;if(!E)return"";const _=E.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(E.repo)}" target="_blank" rel="noopener noreferrer">${n(_)}</a></p>`}function oe(){var V,Q,U;const m=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",g=(V=t.catalog)==null?void 0:V.networks.find(re=>re.ChainID===t.chainId),E=((U=(Q=t.catalog)==null?void 0:Q.clients.find(re=>re.id===t.execId))==null?void 0:U.snapshotSupported)??!1,_=g?`${M(S(g))} (${p})`:"Smaller",l=g?`${M(w(g))} (${C})`:"Much larger",b=g?` on ${n(g.Name)}`:"",A=g?t.checkpoint?g.SyncLabel:g.GenesisSyncLabel:"";return`
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
          ${g?`<p class="sync-estimate">⏱ Estimated initial sync${b}: <strong>${n(A)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((g==null?void 0:g.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((g==null?void 0:g.CheckpointURL)??"")}" value="${n(t.checkpointUrl)}" />
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
              <tr><th>Approx. disk footprint${b}</th><td class="yes">${_}</td><td class="limited">${l}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${g?B(g):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${g?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${_}${g?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(m)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(m)}" value="${n(t.dataDir)}" />
          </label>
          ${X(g,t.dataDir||m)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(m)}/jwt.hex" value="${n(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Fe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Fe}" value="${n(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${n(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${je})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${je}" value="${n(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${n(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${We})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${We}" value="${n(t.execP2PPort)}" />
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
    `}function le(){const g=t.catalog.networks.find(ne=>ne.ChainID===t.chainId),E=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,_=t.jwtPath||`${E}/jwt.hex`,l=Ye.map(ne=>`<li>${n(ne.title)}</li>`).join(""),b=R(t.execHTTPPort,Fe),A=R(t.beaconHTTPPort,je),V=R(t.execP2PPort,We),Q=b||A||V?`<tr><th>Non-default ports</th><td>${[b?`exec HTTP ${b}`:null,A?`beacon HTTP ${A}`:null,V?`exec p2p ${V}`:null].filter(ne=>ne!==null).map(n).join(", ")}</td></tr>`:"",{addr:U}=y(t.rpcBindAddr),re=U?`<tr><th>RPC bind address</th><td><code>${n(U)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((g==null?void 0:g.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(E)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(_)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${n(t.checkpointUrl||(g==null?void 0:g.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
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
    `}function Y(){const g=t.catalog.networks.find(U=>U.ChainID===t.chainId),E=g==null?void 0:g.LearnURL,_=new Set(t.events.filter(U=>U.done).map(U=>U.stepId)),l=new Set(t.events.filter(U=>U.err).map(U=>U.stepId)),b=new Map;for(const U of t.events){if(!U.line)continue;const re=b.get(U.stepId)??[];re.push(U.line),b.set(U.stepId,re)}const A=Ye.map(U=>{var Oe;const re=_.has(U.id),ne=l.has(U.id),Ee=ne?D("failed","bad"):re?D("done","ok"):D("pending","neutral"),Me=(b.get(U.id)??[]).slice(-5),Ue=(Oe=t.events.find(Pe=>Pe.stepId===U.id&&Pe.err))==null?void 0:Oe.err,Ve=U.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${E?` <a href="${n(E)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${re?"step-done":""} ${ne?"step-error":""}">
          <div class="step-head">${Ee} <strong>${n(U.title)}</strong></div>
          ${Ve}
          ${Me.length?`<pre class="step-log">${Me.map(Pe=>n(Pe)).join(`
`)}</pre>`:""}
          ${Ue?`<p class="error small">${n(Ue)}</p>`:""}
        </li>
      `}).join(""),V=t.events.some(U=>U.err),Q=Ye.every(U=>_.has(U.id))||t.events.some(U=>U.stepId==="handshake"&&U.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${A}</ol>
        ${Q&&!V?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        ${V?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function me(m,g){switch(m){case"pick-network":t.chainId=Number(g.dataset.chainId),t.execId=null,t.beaconId=null,f();break;case"goto-network":t.step="network",f();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",f();break;case"goto-mode":t.step="mode",f(),W();break;case"goto-review":if(ge(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){f();break}t.step="review",f();break;case"start-setup":H();break}}function ge(){const m=s.querySelectorAll('input[name="mode"]');for(const U of Array.from(m))U.checked&&(t.archive=U.value==="archive");const g=s.querySelector("#data-dir-input"),E=s.querySelector("#jwt-path-input");g&&(t.dataDir=g.value.trim()),E&&(t.jwtPath=E.value.trim());const _=s.querySelector("#exec-http-port-input"),l=s.querySelector("#beacon-http-port-input"),b=s.querySelector("#exec-p2p-port-input");_&&(t.execHTTPPort=_.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),b&&(t.execP2PPort=b.value.trim());const A=s.querySelector("#rpc-bind-addr-input");A&&(t.rpcBindAddr=A.value.trim());const V=s.querySelector("#checkpoint-url-input");V&&(t.checkpointUrl=V.value.trim());const Q=s.querySelector("#snapshot-key-input");Q&&(t.snapshotKey=Q.value.trim()),t.execHTTPPortError=T(t.execHTTPPort).error??null,t.beaconHTTPPortError=T(t.beaconHTTPPort).error??null,t.execP2PPortError=T(t.execP2PPort).error??null,t.rpcBindAddrError=y(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?ae(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function y(m){if(!m)return{};const g=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(m);return g?g.slice(1).every(E=>Number(E)<=255)?{addr:m}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(m)&&m.includes(":")?{addr:m}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function T(m){if(!m)return{};if(!d.test(m))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const g=Number(m);return!Number.isInteger(g)||g<1||g>65535?{error:"Port must be between 1 and 65535."}:{port:g}}function R(m,g){const{port:E}=T(m);if(!(E===void 0||E===g))return E}async function H(){var b;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(b=t.streamStop)==null||b.call(t),t.streamStop=null,f();const m={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(m.DataDir=t.dataDir),t.jwtPath&&(m.JWTPath=t.jwtPath);const g=R(t.execHTTPPort,Fe),E=R(t.beaconHTTPPort,je),_=R(t.execP2PPort,We);g!==void 0&&(m.ExecHTTPPort=g),E!==void 0&&(m.BeaconHTTPPort=E),_!==void 0&&(m.ExecP2PPort=_);const{addr:l}=y(t.rpcBindAddr);l!==void 0&&(m.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(m.CheckpointURL=t.checkpointUrl):m.NoCheckpoint=!0,t.execSnapshot&&(m.ExecSnapshot=!0,m.SnapshotKey=t.snapshotKey);try{await yn(t.targetId,m)}catch(A){if(!(A instanceof we&&A.status===409)){t.starting=!1,t.startError=String(A instanceof Error?A.message:A),f();return}}t.starting=!1,t.step="run",f(),t.streamStop=Qe(t.targetId,A=>{r||(t.events.push(A),t.step==="run"&&f())})}function K(m){const g=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],_=g.map(l=>l.id).indexOf(m);return`
      <ol class="wizard-progress">
        ${g.map((l,b)=>`<li class="${b===_?"current":b<_?"past":"future"}">${n(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var m;r=!0,(m=t.streamStop)==null||m.call(t)}}function ua(s,i){let r=!1;const t=new Map;s.innerHTML=`<h1>${n(i)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const h=s.querySelector("#machine-body");ye(s,(I,M)=>{I==="toggle-section"&&q(M.dataset.section??"")}),$();async function $(){let I,M;try{const[O,L]=await Promise.all([Ce(),xe()]);I=O.find(C=>C.id===i),M=L}catch(O){if(r)return;h.innerHTML=`<p class="error">Failed to load machine: ${n(String(O))}</p>`;return}if(!r){if(!I){location.hash="#/targets";return}P(I,M)}}function P(I,M){const O=I.mode==="local"?"this machine":"SSH",L=I.mode==="ssh"&&I.ssh?`${n(I.ssh.User)}@${n(I.ssh.Host)}`:O;h.innerHTML=`
      <p class="muted">${L}</p>
      <p>${f(I,M)}</p>
      <div class="machine-sections">
        ${j.map(C=>N(C,I,M)).join("")}
      </div>
      ${de()}
    `}function f(I,M){const O=I.wire;if(!O)return D("not set up","neutral");const L=M.networks.find(p=>p.ChainID===O.ChainID),C=L?L.Name:`chain ${O.ChainID}`;return`${D(C,"ok")} ${D(O.ExecID,"neutral")} ${D(O.BeaconID,"neutral")}${O.Archive?" "+D("archive","warn"):""}`}function N(I,M,O){return`
      <section class="card machine-section" data-section-card="${n(I.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(I.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(I.title)}</span>
          <span class="machine-section-status">${I.status(M,O)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(I.key)}" hidden></div>
      </section>
    `}function q(I){const M=j.find(w=>w.key===I);if(!M)return;const O=s.querySelector(`[data-section-card="${I}"]`),L=s.querySelector(`[data-section-body="${I}"]`),C=s.querySelector(`.machine-section-head[data-section="${I}"]`);if(!O||!L||!C)return;const p=L.hidden;if(p&&!t.has(I)){const w=document.createElement("div");L.appendChild(w),t.set(I,M.mount(w))}L.hidden=!p,O.classList.toggle("open",p),C.setAttribute("aria-expanded",String(p))}const j=[{key:"setup",title:"Setup",status:I=>I.wire?D("set up","ok"):D("not set up","neutral"),mount:I=>da(I,i)},{key:"dashboard",title:"Dashboard",status:I=>I.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>aa(I,i)},{key:"logs",title:"Logs",status:I=>I.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>sa(I,i)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:I=>ca(I,i)}];return()=>{r=!0;for(const I of t.values())try{I()}catch{}t.clear()}}const pa=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`;function yt(s){let i=null,r=null;s.innerHTML=pa+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const t=s.querySelector("#p-card");async function h(){try{const N=await et();i=ha(N.gateways),r=null}catch(N){r=fa(N)}$()}function $(){t.innerHTML=P()}function P(){return r?ma(r):ba()}ye(t,(N,q)=>{f()});async function f(N,q){}return h(),()=>{}}function ha(s){return!s||s.length===0?null:s.find(i=>i.placement.targetId==="local")??s[0]}function fa(s){return s instanceof Error?s.message:String(s)}function ma(s){return`<div class="p-band" style="padding:16px;color:var(--red)">${n(s)}</div>`}function ba(s){return'<div class="p-band" style="padding:16px">Panel loading…</div>'}function ya(s,i){let r=!1,t=[],h=null,$=!1,P=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${de()}</div>`;const f=s.querySelector("#sec-body"),N=s.querySelector("#sec-footer");ye(s,(L,C)=>{var p;if(L==="rerun")j();else if(L==="toggle")(p=C.closest(".check-item"))==null||p.classList.toggle("expanded");else if(L==="copy"){const w=C.dataset.copy;w&&O(C,w)}}),q();async function q(){let L,C;try{const[w,S]=await Promise.all([Ce(),xe()]);L=w.find(B=>B.id===i),C=S}catch(w){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(w))}</p>`;return}if(r)return;if(!L){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!L.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const p=C==null?void 0:C.networks.find(w=>w.ChainID===L.wire.ChainID);p&&(N.innerHTML=de(p.Name,p.LearnURL)),await j()}async function j(){$=!0,h=null,I();try{t=await xn(i),P=!0}catch(L){h=String(L instanceof Error?L.message:L)}$=!1,r||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${$?"disabled":""}>${$?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${!P&&$?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(M).join("")}</ul>`:P?'<p class="muted">No checks returned.</p>':""}
    `}function M(L){const C=L.Status==="pass"?"ok":L.Status==="fail"?"bad":L.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${D(L.Status,C)}
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
    `}async function O(L,C){const p=await De(C),w=L.textContent;L.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{r||(L.textContent=w)},1500)}return()=>{r=!0}}const ga=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],Xe="VALVE_API_KEY";function va(s){return s===Xe?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(s)}}</code> slot wherever an endpoint URL carries one.`}function $a(s){let i=!1,r=!1,t=!1,h=null,$=!1,P=null,f=null;const N=new Set,q=new Map;let j="",I="";s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${de()}`;const M=s.querySelector("#settings-body");ye(s,(S,B)=>{if(S==="save"&&w(),S==="clear-key"){if(!P)return;r=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),p(P)}if(S==="clear-provider-key"){const F=B.dataset.key;if(!P||!F)return;N.add(F),q.set(F,""),$=!1,p(P)}}),tt(s,(S,B)=>{S!=="ai-provider"||!P||(f=B,$=!1,p(P))}),O();async function O(){try{const S=await zn();if(i)return;P=S,p(S)}catch(S){if(i)return;M.innerHTML=`<p class="error">Failed to load settings: ${n(String(S))}</p>`}}function L(S){const F=(Array.isArray(S.providerKeysSet)?S.providerKeysSet:[]).filter(X=>X!==Xe).sort();return[Xe,...F]}function C(S,B){const F=n(S);return`
      <div class="pk-row">
        <label>
          <code>${F}</code>
          <input class="provider-key" data-key="${F}" type="password" autocomplete="off"
                 placeholder="${B?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${va(S)}</p>
        ${B?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${F}">Clear saved key</button>`:""}
      </div>`}function p(S){var ue;const B=f??S.aiProvider,F=Array.isArray(S.providerKeysSet)?S.providerKeysSet:[],X=L(S).map(J=>C(J,F.includes(J))).join("");M.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ze("ai-provider",ga.map(J=>({value:J.value,label:J.label})),B)}
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
            <input id="ref-rpc-base" type="text" value="${n(S.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${h?`<p class="error">${n(h)}</p>`:""}
        ${$?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const te=s.querySelector("#ai-key");te==null||te.addEventListener("input",()=>{r=!0,$=!1}),(ue=s.querySelector("#ref-rpc-base"))==null||ue.addEventListener("input",()=>{$=!1}),s.querySelectorAll("input.provider-key").forEach(J=>{const se=J.dataset.key;if(!se)return;const oe=q.get(se);oe!==void 0&&(J.value=oe),J.addEventListener("input",()=>{N.add(se),q.set(se,J.value),$=!1})});const W=s.querySelector("#pk-new-value");W&&(W.value=I),W==null||W.addEventListener("input",()=>{I=W.value,$=!1});const ae=s.querySelector("#pk-new-name");ae==null||ae.addEventListener("input",()=>{j=ae.value,$=!1})}async function w(){const S=s.querySelector("#ai-key"),B=s.querySelector("#ref-rpc-base");if(!S||!B||!P)return;const F={aiProvider:f??P.aiProvider,refRpcBase:B.value.trim()};r&&(F.aiKey=S.value);const X={};for(const W of N)X[W]=q.get(W)??"";const te=j.trim();te&&(X[te]=I),Object.keys(X).length>0&&(F.providerKeys=X),t=!0,h=null,$=!1,p(P);try{const W=await Jn(F);if(i)return;P=W,r=!1,N.clear(),q.clear(),j="",I="",t=!1,$=!0,p(W)}catch(W){if(i)return;t=!1,h=String(W instanceof Error?W.message:W),p(P)}}return()=>{i=!0}}const wa=["http","ws","archive","trace"],ka={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Le=1337,Ta="run",Sa={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function xa(s){let i=!1,r=null,t=null;const h={},$={},P={},f={},N={},q={},j={},I={},M={},O={},L={},C={},p={},w={},S={};let B="",F=null;s.innerHTML=`
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
  `;const X=s.querySelector("#rpc-body");ye(s,(e,a)=>{Ot(e,a)}),tt(s,()=>{}),W(),te();async function te(){try{const e=await gt();if(i)return;B=e.os,Y()}catch{}}async function W(){try{const e=await et();if(i)return;r=e,t=null}catch(e){if(i)return;r=null,t=pe(e)}Y();for(const e of(r==null?void 0:r.gateways)??[])ae(e.id),ue(e.id,!1)}async function ae(e){try{const a=await Mn(e);if(i)return;h[e]=a}catch{if(i)return;h[e]=null}Y()}async function ue(e,a){P[e]=a,a&&Y();try{const o=await On(e,a);if(i)return;$[e]=o}catch{if(i)return;$[e]=null}P[e]=!1,Y()}function J(e){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===e)}function se(e,a){return(e.networks??[]).find(o=>o.chainId===a)}function oe(e,a,o){var u;const c=(((u=h[e])==null?void 0:u.networks)??[]).find(k=>k.chainId===a);return((c==null?void 0:c.upstreams)??[]).find(k=>k.upstream===o)}function le(e,a,o){var c;return(((c=$[e])==null?void 0:c.endpoints)??[]).find(u=>u.chainId===a&&u.upstream===o)}function Y(){if(i)return;if(t){X.innerHTML=`<p class="error">Could not read the gateways: ${n(t)}</p>`;return}if(!r){X.innerHTML='<p class="muted">Loading…</p>';return}const e=r.gateways??[],a=e.length>1,o=(r.targets??[]).some(k=>lt(k.id,e)),c=new Set(e.map(k=>k.placement.targetId)),u=(r.orphans??[]).filter(k=>!c.has(k.targetId));X.innerHTML=`
      ${e.map(k=>y(k,a)).join("")}
      ${e.length===0?ge():""}
      ${u.map(me).join("")}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function me(e){const a=`docker rm -f ${e.containerName}`,o=p[e.containerName];return`
      <div class="strip">
        ${A({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:a})}
        ${o?A({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function ge(){return((r==null?void 0:r.targets)??[]).length===0?`
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
      ${b(e)}
      ${ne(e)}
      ${Ee(e)}
      ${m(e)}
    `}function d(e){const a=e.status.State==="running",o=e.tls,c=[`on <strong>${n(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${n(e.status.Image)}</code>`),c.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${U(e)}
        <strong>${n(e.label)}</strong>
        ${Q(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${a?`base <code>${n(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function T(e){const a=e.tls;return a!=null&&a.enabled&&a.rootCaPath&&a.effectiveCertSource==="internal"?a.rootCaPath:null}function R(e){var a;return((a=((r==null?void 0:r.targets)??[]).find(o=>o.id===e.placement.targetId))==null?void 0:a.mode)??""}function H(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function K(e,a,o){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${a}"`;case"windows":return`certutil -addstore -f ROOT "${a}"`;case"linux":default:return`sudo cp "${a}" /usr/local/share/ca-certificates/valve-node-app-${o}.crt && sudo update-ca-certificates`}}function m(e){const a=M[e.id]??!1,o=((r==null?void 0:r.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${a?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${n(e.id)}" aria-expanded="${a}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${g(e,o.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${a?E(e,o):""}
      </section>
    `}function g(e,a){const o=[];return e.status.State!=="running"&&o.push("gateway not running"),a>0&&o.push(`${a} leftover container${a===1?"":"s"}`),o.length===0?"container, settings, certificate":o.join(" · ")}function E(e,a){var o;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>re(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${P[e.id]?"disabled":""}>
            ${P[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
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
        ${a.map(me).join("")}
        ${j[e.id]?At(e):""}
      </div>
    `}function _(e){const a=T(e);if(!a)return"";const o=R(e)==="local",c=K(B,a,e.id),u=S[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${n(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${n(a)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a)}">Copy path</button>
        </div>
        ${o?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${w[e.id]?"disabled":""}>
                   ${w[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${u?l(u):""}
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
    `}function b(e){const a=[];e.error&&a.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&a.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])a.push({tone:"warn",text:c});a.push(...V(e));const o=N[e.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(A).join("")}</div>`}function A(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${n(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${n(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.cmd)}">Copy</button>`:""}
      </div>
    `}function V(e){var u,k;const a=e.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((u=a.status)==null?void 0:u.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((k=a.status)==null?void 0:k.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const c=O[e.id]??a.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&o.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&o.push({tone:"warn",text:c.expiryWarning}),o}function Q(e){switch(e.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function U(e){return e.status.State==="running"?$e("ok"):e.status.State==="unknown"?$e("bad"):$e("neutral")}function re(e,a){const o=Sa[a];if(!o)return"";const c=f[e.id];return`
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
    `}function Ee(e){const a=Me(e.networks??[]),o=a.some(c=>c.chainId===Le);return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(e.id)}">Add a network</button>
            ${nt(e,o)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${a.map(c=>Ue(e,c)).join("")}
        </div>
        ${Pe(e,o)}
        ${Lt(e)}
      </div>
    `}function Me(e){const a=e.filter(c=>c.chainId!==Le),o=e.filter(c=>c.chainId===Le);return[...a,...o]}function Ue(e,a){const o=wt(a),c=a.chainId===Le,u=`${e.id}:${a.chainId}`,k=I[u]??!1,x=o.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${o.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${n(a.name)}</span>
          <code class="chain-key">evm:${a.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${D(x,o.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${n(u)}" aria-expanded="${k}">
              ${k?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Ve(e,a)}
        ${k?Oe(e,a,o):""}
      </section>
    `}function Ve(e,a){if(!a.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const o=T(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${n(a.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                title="Copy ${n(a.url)}">Copy URL</button>
        ${o?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${R(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${n(e.id)}" ${w[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${w[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(o)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${n(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${S[e.id]?`<span class="chain-cert muted small">${n(S[e.id].ok?"Trusted — reload your wallet or browser.":S[e.id].message)}</span>`:""}`:""}
      </div>
    `}function Oe(e,a,o){const c=a.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${o.why?" chain-verdict-why":""}"${o.why?` title="${n(o.why)}"`:""}>${o.html}</p>
        <div class="chain-detail-bar">
          ${$t(c.length,o.tone,a.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">Remove</button>
        </div>
        ${St(e,a)}
        ${(a.warnings??[]).map(u=>`<p class="chain-note">${n(u)}</p>`).join("")}
      </div>
    `}function Pe(e,a){const o=$[e.id],c=o!=null&&o.at?`probed ${n(st(o.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${n(e.id)}">+ Network</button>
        ${nt(e,a)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${n(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${P[e.id]?"disabled":""}>
          ${P[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function nt(e,a){return a?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${n(e.id)}"
                    title="Add a throwaway local test chain (evm:${Le}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function $t(e,a,o){const c=o>0,u=c?o:e,k=Math.min(e,u);let x="";for(let Re=0;Re<u;Re++)x+=`<span class="seg${Re<k?` seg-on seg-${a}`:""}"></span>`;const v=c&&e>o,G=c?v?`${e} (set is ${o})`:`${e} of ${o}`:`${e}`,ee=`${e} upstream${e===1?"":"s"} configured`,he=c?`${ee}${v?`, ${e-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${ee}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(he)}">${x}</span>
      <span class="segs-n">${G}</span>
    `}function wt(e){const a=e.upstreams??[];if(a.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!a.some(kt)){const c=Tt(a);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(k=>`<code>${n(k)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(a.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!a.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=a.filter(c=>!!c.problem);if(o.length>0){const c=a.length-o.length;return{tone:"warn",html:`${o.length} of these ${a.length} endpoints ${o.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function kt(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Tt(e){const a=new Set;for(const o of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());c&&a.add(c[1].toLowerCase())}return[...a].sort()}function St(e,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(c=>xt(e,a,c)).join("")}</ul>`}function xt(e,a,o){const c=`${e.id}|${a.chainId}|${o.id}`,u=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?$e("bad"):$e("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${Ct(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${Et(e,a,o)}</div>
        <div class="up-share">${Rt(e,a,o)}</div>
        <div class="up-acts">
          ${u.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(c)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[e.id]?"disabled":""}>
                   ${f[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(c)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function Ct(e){return e.problem?D("unusable","bad"):e.recentOnly?D("recent blocks","warn"):e.local?D("yours","ok"):D("public","neutral")}function at(e,a){var o;if(e)return a==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(c=>c.key===a))==null?void 0:o.status}function Et(e,a,o){const c=le(e.id,a.chainId,o.id);return c?c.unprobeable?`<span class="caps-none" title="${n(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${wa.map(u=>Pt(e,a,c,u)).join("")}</span>`:`<span class="muted small">${$[e.id]===void 0?"probing…":"—"}</span>`}function Pt(e,a,o,c){const u=(o.capabilities??[]).find(ee=>ee.key===c),k=at(o,c)??"inconclusive",x=ka[c]??c.toUpperCase();let v="cap";k==="unsupported"?v=It(e,a,c)?"cap missing":"cap off":k==="inconclusive"?v="cap unknown":k==="inconsistent"&&(v="cap mixed");const G=u!=null&&u.detail?`${u.label}: ${u.detail}`:c==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${x}: no verdict`;return`<span class="${v}" title="${n(G)}">${n(x)}</span>`}function It(e,a,o){const c=(a.upstreams??[]).map(u=>le(e.id,a.chainId,u.id)).filter(u=>!!u&&!u.unprobeable);return c.length>0&&c.every(u=>at(u,o)==="unsupported")}function Rt(e,a,o){const c=h[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const u=oe(e.id,a.chainId,o.id),k=(c.networks??[]).find(he=>he.chainId===a.chainId);if(!u||!k||k.attributed===0)return'<span class="muted small">no traffic yet</span>';const x=Math.round(u.actual*100),v=Math.round(u.intended*100),G=u.diverged?o.local?"warn":"":"ok",ee=`${u.succeeded.toLocaleString()} of ${k.attributed.toLocaleString()} answered requests · routing intends ${v}%`+(u.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ee)}">
        <span class="bar">
          <span class="fill${G?" "+G:""}" style="width:${x}%"></span>
          <span class="tick" style="left:${v}%"></span>
        </span>
        <span class="share-n${u.diverged?" warn":""}">${x}%</span>
        ${u.unconfigured?D("not in config","warn"):""}
      </span>
    `}function Lt(e){const a=h[e.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(st(a.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function st(e){const a=new Date(e);return Number.isNaN(a.getTime())?e:a.toLocaleString()}function At(e){const a=e.config;return`
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
        ${Nt(e)}
        ${Bt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(e.id)}">Save settings</button>
        </div>
      </div>
    `}function Nt(e){const a=!e.config.MetricsOff;return`
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
    `}function Bt(e){var x;const a=n(e.id),o=e.config.TLS??null,c=(o==null?void 0:o.Enabled)??!1,u=(o==null?void 0:o.CertSource)||"internal",k=((x=e.tls)==null?void 0:x.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${a}-tls-host" value="${n((o==null?void 0:o.Hostname)??k)}"
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
        <input type="text" inputmode="numeric" id="gw-${a}-tls-port" value="${(o==null?void 0:o.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${a}-tls-source">
          <option value="internal" ${u==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${u==="files"?"selected":""}>A certificate file on this machine</option>
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
      ${Ht(e)}
    `}function Ht(e){var x,v;const a=n(e.id),o=((x=e.config.TLS)==null?void 0:x.Enabled)??!1,c=O[e.id]??((v=e.tls)==null?void 0:v.verification)??null,u=L[e.id]??!1,k=C[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!u?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${u?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${k?`<p class="error small">${n(k)}</p>`:""}
      ${c?Dt(c):""}
    `}function Dt(e){const a=(e.assertions??[]).map(o=>`
          <li class="small">
            ${Mt(o.status)}
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
    `}function Mt(e){switch(e){case"pass":return D("pass","ok");case"fail":return D("fail","bad");case"unavailable":return D("unavailable","warn");default:return D("skipped","neutral")}}async function Ut(e){L[e]=!0,C[e]=null,Y();try{O[e]=await Dn(e)}catch(a){C[e]=`${pe(a)}${Ie(a)}`}finally{L[e]=!1,Y()}}function ke(e){return{...e.config,Networks:(e.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function Te(e,a,o){N[e]=null;try{await Fn(e,a)}catch(c){return N[e]=`${o?o+": ":""}${pe(c)}`,Y(),!1}return await W(),!0}async function Ot(e,a){const o=a.dataset.gid??"";switch(e){case"refresh":await W();return;case"copy":a.dataset.copy&&await un(a,a.dataset.copy);return;case"reprobe":await ue(o,!0);return;case"toggle-settings":j[o]=!j[o],Y();return;case"toggle-manage":M[o]=!M[o],Y();return;case"toggle-chain-detail":{const c=a.dataset.key??"";c&&(I[c]=!I[c]),Y();return}case"save-settings":await qt(o);return;case"verify-tls":await Ut(o);return;case"trust-cert":await Wt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await _t(o,e.slice(3));return;case"gw-create":case"gw-recreate":await Kt(o);return;case"gw-wipe":rn(o);return;case"add-gateway":ln();return;case"forget-gateway":await Vt(o);return;case"dismiss-orphan":await Gt(a.dataset.name??"");return;case"add-chain":zt(o);return;case"add-devnet":{const c=J(o);if(c){const u=((r==null?void 0:r.targets)??[]).some(k=>k.id===c.placement.targetId&&k.hasDevnet);rt(o,Le,u)}return}case"remove-chain":await Zt(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":ct(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await Xt(a.dataset.key??"");return;case"reset-devnet":await sn(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function qt(e){const a=J(e);if(!a)return;const o=ke(a),c=s.querySelector(`#gw-${CSS.escape(e)}-port`),u=s.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const v=Number.parseInt(c.value.trim(),10);Number.isFinite(v)&&(o.Port=v)}u&&(o.BindAddr=u.value.trim());const k=s.querySelector(`#gw-${CSS.escape(e)}-metrics`);k&&(o.MetricsOff=!k.checked),o.TLS=Ft(e,a);const x=a.status.State==="running";await Te(e,o,"Saving settings")&&(j[e]=!1,x&&(N[e]=null,jt(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),Y())}function Ft(e,a){var k,x,v,G,ee,he,Re;const o=pn=>s.querySelector(`#gw-${CSS.escape(e)}-${pn}`),c=o("tls");if(!c)return a.config.TLS??null;const u=Number.parseInt(((k=o("tls-port"))==null?void 0:k.value.trim())??"",10);return{Enabled:c.checked,Hostname:((x=o("tls-host"))==null?void 0:x.value.trim())??"",CertSource:((v=o("tls-source"))==null?void 0:v.value)??"internal",CertFile:((G=o("tls-cert"))==null?void 0:G.value.trim())??"",KeyFile:((ee=o("tls-key"))==null?void 0:ee.value.trim())??"",HTTPSPort:Number.isFinite(u)?u:443,BindAddr:((he=a.config.TLS)==null?void 0:he.BindAddr)??"",ImageRef:((Re=a.config.TLS)==null?void 0:Re.ImageRef)??""}}function jt(e,a){q[e]=[a]}async function Wt(e){if(!w[e]){w[e]=!0,S[e]=null,Y();try{S[e]=await Wn(e)}catch(a){S[e]={ok:!1,message:`${pe(a)}${Ie(a)}`}}w[e]=!1,Y()}}async function _t(e,a){if(!f[e]){f[e]=a,N[e]=null,Y();try{await jn(e,a)}catch(o){N[e]=`${a} failed: ${pe(o)}${Ie(o)}`}f[e]=null,await W()}}async function Kt(e){if(f[e])return;f[e]="create",N[e]=null,q[e]=["starting…"],Y();let a;try{a=await _n(e)}catch(o){N[e]=`${pe(o)}${Ie(o)}`,q[e]=[],f[e]=null,Y();return}F==null||F(),F=Qe(a.targetId,o=>{if(i)return;const c=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(q[e]=[...(q[e]??[]).filter(k=>k!=="starting…"),c],!!o.err||o.stepId===Ta&&!!o.done){F==null||F(),F=null,f[e]=null,o.err&&(N[e]="Provisioning failed — see the log below."),W();return}Y()})}async function Vt(e){const a=J(e);if(!(!a||!await Be({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await qn(e)}catch(c){N[e]=pe(c),Y();return}await W()}}async function Gt(e){if(e){p[e]=null;try{await Bn(e)}catch(a){p[e]=pe(a),Y();return}await W()}}function zt(e){const a=J(e);if(!a)return;const o=new Set((a.networks??[]).map(v=>v.chainId)),c=(r==null?void 0:r.presets)??[],u=c.filter(v=>!o.has(v.chainId)),k=c.filter(v=>o.has(v.chainId)),x=((r==null?void 0:r.targets)??[]).some(v=>v.id===a.placement.targetId&&v.hasDevnet);ie(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${u.map(v=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${v.chainId}">
                <span>${n(v.name)}</span>
                <span class="muted small">chain ${v.chainId}${v.devnet?x?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${k.length?`<p class="muted small">Already fronted: ${n(k.map(v=>v.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,v=>{if(v==="cancel"){Z();return}if(v==="custom"){Jt(e);return}if(v.startsWith("preset:")){const G=Number.parseInt(v.slice(7),10),ee=c.find(he=>he.chainId===G);Z(),ee!=null&&ee.devnet?rt(e,G,x):ot(e,G)}})}function Jt(e){var a;ie(`
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
      `,o=>{if(o==="cancel"){Z();return}if(o!=="add")return;const c=document.getElementById("custom-chain-id"),u=document.getElementById("custom-chain-err"),k=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(k)||k<=0){u&&(u.className="error small"),u&&(u.textContent="A chain id is a positive whole number.");return}Z(),ot(e,k)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function ot(e,a){const o=J(e);if(!o)return;const c=ke(o),u=c.Networks??[];u.some(k=>k.ChainID===a)||(u.push({ChainID:a,Upstreams:[]}),c.Networks=u,await Yt(e,c)&&(Y(),ct(e,a)))}async function Yt(e,a){var k;const o={...a,Networks:(a.Networks??[]).filter(x=>x.Upstreams.length>0)};if(!await Te(e,o))return!1;const u=J(e);if(u)for(const x of a.Networks??[])x.Upstreams.length===0&&!(u.networks??[]).some(v=>v.chainId===x.ChainID)&&(u.config.Networks=[...u.config.Networks??[],{ChainID:x.ChainID,Upstreams:[]}],u.networks=[...u.networks??[],{chainId:x.ChainID,name:((k=((r==null?void 0:r.presets)??[]).find(v=>v.chainId===x.ChainID))==null?void 0:k.name)??`Chain ${x.ChainID}`,path:`/${u.config.ProjectID}/evm/${x.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function rt(e,a,o){const c=J(e);if(!c)return;if(!o){ie(`
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
        `,()=>Z());return}const u=ke(c),k=u.Networks??[],x={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},v=k.find(G=>G.ChainID===a);v?v.Upstreams.push(x):k.push({ChainID:a,Upstreams:[x]}),u.Networks=k,await Te(e,u,"Adding the devnet")}async function Zt(e,a){const o=J(e);if(!o||!Number.isFinite(a))return;const c=se(o,a);if(!await Be({title:`Remove ${(c==null?void 0:c.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const k=ke(o);k.Networks=(k.Networks??[]).filter(x=>x.ChainID!==a),await Te(e,k,"Removing the network")}function it(e){const a=e.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function Xt(e){const a=it(e);if(!a)return;const o=J(a.gid);if(!o)return;const c=ke(o),u=(c.Networks??[]).find(v=>v.ChainID===a.chainId);if(!u)return;const k=u.Upstreams.findIndex((v,G)=>(v.ID||`${a.chainId}-${G}`)===a.upstreamId);k<0||!await Be({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(u.Upstreams.splice(k,1),await Te(a.gid,c,"Removing the endpoint"))}function ct(e,a){const o=J(e);if(!o||!Number.isFinite(a))return;const c=((r==null?void 0:r.sources)??[]).filter(v=>v.chainId===a),u=se(o,a),k=new Set(((u==null?void 0:u.upstreams)??[]).filter(v=>v.kind!=="external").map(v=>`${v.kind}|${v.targetId??""}`)),x=c.filter(v=>!k.has(`${v.kind}|${v.targetId}`));ie(`
        <h2>Add an endpoint for ${n((u==null?void 0:u.name)??`chain ${a}`)}</h2>
        ${x.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${x.map(v=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(v.kind)}:${n(v.targetId)}">
                       <span>${n(v.label)}</span>
                       <span class="muted small">${n(v.endpoint)}</span>
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
      `,v=>{if(v==="cancel"){Z();return}if(v==="known-set"){tn(e,a);return}if(v==="manual"){an(e,a);return}if(v.startsWith("source:")){const[,G,ee]=v.split(":");Z(),Qt(e,a,G,ee)}})}async function Qt(e,a,o,c){const u=J(e);if(!u)return;const k=ke(u),x=k.Networks??[],v={ID:`${o==="managed-devnet"?"devnet":"node"}-${c}`,Kind:o,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},G=x.find(ee=>ee.ChainID===a);G?G.Upstreams.push(v):x.push({ChainID:a,Upstreams:[v]}),k.Networks=x,await Te(e,k,"Adding the endpoint")}function en(e){const a=[...e].sort((u,k)=>(u.latencyMs??1e9)-(k.latencyMs??1e9)),o=a.slice(0,3),c=a.find(u=>u.url.startsWith("wss://")||u.url.startsWith("ws://"));return c&&!o.some(u=>u.url===c.url)&&(o.length===3&&o.pop(),o.push(c)),new Set(o.map(u=>u.url))}async function tn(e,a){let o;try{o=await Gn(e,a)}catch(v){ie(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(pe(v))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Z());return}if(i)return;const c=o.endpoints??[],u=c.filter(v=>!v.alreadyAdded).map(v=>v.url),k=new Set(c.map(v=>v.provider)).size,x=c.map(v=>{const G=[v.websocket?'<span class="t ws">websocket</span>':"",v.archive?'<span class="t ar">archive</span>':"",v.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n(v.url)}</code>
                  <span class="muted small">${n(v.provider)}</span> ${G}</li>`}).join("");ie(`<h2>Endpoints for chain ${a}</h2>
       ${c.length?`<p class="muted small">${k} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${x}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${o.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${u.length?"":" disabled"}>
           ${u.length?`Add ${u.length}`:"Nothing to add"}</button>
       </div>`,v=>{Z(),v==="add"&&Ge(e,a,u),v==="discover"&&nn(e,a)})}async function nn(e,a){ie(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,x=>{x==="cancel"&&Z()});let o;try{o=await Vn(a)}catch(x){const v=He();if(v){const G=document.createElement("p");G.className="error small",G.textContent=`Could not discover endpoints: ${pe(x)}`,v.appendChild(G)}return}if(i)return;const c=(o.endpoints??[]).filter(x=>x.status==="live"||x.status==="unprobed"),u=(o.endpoints??[]).filter(x=>x.status==="rejected"),k=en(c);ie(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(x=>{const v=k.has(x.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(x.url)}"${v}>
                       <span><code>${n(x.url)}</code></span>
                       <span class="muted small">${x.status==="live"?`answered in ${x.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${u.length?`<details class="rpc-rejected">
                 <summary class="muted small">${u.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${u.map(x=>`<li class="muted small"><code>${n(x.url)}</code> — ${n(x.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,x=>{if(x==="cancel"){Z();return}if(x==="add"){const v=He(),G=v?Array.from(v.querySelectorAll('input[type="checkbox"]:checked')).map(ee=>ee.value):[];Z(),Ge(e,a,G);return}})}function an(e,a){var o;ie(`
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
      `,c=>{if(c==="cancel"){Z();return}if(c!=="add")return;const u=document.getElementById("manual-endpoint"),k=document.getElementById("manual-recent"),x=document.getElementById("manual-err"),v=(u==null?void 0:u.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(v)){x&&(x.className="error small",x.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Z(),Ge(e,a,[v],(k==null?void 0:k.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ge(e,a,o,c=!1){if(!o.length)return;const u=J(e);if(!u)return;const k=ke(u),x=k.Networks??[];let v=x.find(ee=>ee.ChainID===a);v||(v={ChainID:a,Upstreams:[]},x.push(v));let G=1;for(const ee of v.Upstreams){const he=/^public-\d+-(\d+)$/.exec(ee.ID??"");he&&(G=Math.max(G,Number(he[1])+1))}for(const ee of o)v.Upstreams.some(he=>he.Endpoint===ee)||v.Upstreams.push({ID:`public-${a}-${G++}`,Kind:"external",Endpoint:ee,Local:!1,RecentOnly:c});k.Networks=x,await Te(e,k,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function sn(e,a){const o=it(e);if(!o||!a||!await Be({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[o.gid]="reset",N[o.gid]=null,Y();let u;try{u=await An(a)}catch(k){N[o.gid]=`Reset failed: ${pe(k)}${Ie(k)}`,f[o.gid]=null,Y();return}f[o.gid]=null,on(a,u),await W()}function on(e,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const c=a.report.Cascaded??[],u=a.report.CascadeSkipped??[];ie(`
        <h2>Devnet on ${n(e)} reset</h2>
        <ul class="plain-list">${o.map(k=>`<li>${n(k)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${n(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${u.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(u.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Z())}function rn(e){const a=J(e);if(!a)return;ie(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(e)}</button>
        </div>
      `,u=>{if(u==="cancel"||u==="close"){Z(),W();return}u==="confirm"&&cn(e)});const o=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{c&&(c.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function cn(e){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await Kn(e)}catch(c){const u=He();if(u){const k=document.createElement("p");k.className="error small",k.textContent=`Wipe failed: ${pe(c)}${Ie(c)}`,u.appendChild(k)}a&&(a.disabled=!1,a.textContent=`Wipe ${e}`);return}ie(`
        <h2>${n(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Z(),W()})}function lt(e,a){return!a.some(o=>{var c;return((c=o.placement)==null?void 0:c.targetId)===e})}function ln(){var k;const e=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=e.filter(x=>lt(x.id,a)),c=new Set(a.map(x=>x.id));if(e.length===0){ie(`
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
        `,()=>Z());return}const u=c.has("default")?"":"default";ie(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(u)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${o.map(x=>`<option value="${n(x.id)}">${n(x.id)} (${n(x.mode)})</option>`).join("")}
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
      `,x=>{if(x==="cancel"){Z();return}x==="create"&&dn()}),(k=document.getElementById("new-gw-id"))==null||k.focus()}async function dn(){const e=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),u=(e==null?void 0:e.value.trim())??"",k=(a==null?void 0:a.value)??"",x=Number.parseInt((o==null?void 0:o.value.trim())??"",10),v=G=>{c&&(c.className="error small",c.textContent=G)};if(!u){v("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!k){v("Pick the machine it runs on.");return}try{await Hn({id:u,placement:{targetId:k,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(x)?x:4e3,Networks:[]}})}catch(G){v(pe(G));return}Z(),await W()}async function un(e,a){const o=await De(a),c=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(e.textContent=c)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Ie(e){return e instanceof we&&e.hint?` — ${e.hint}`:""}return()=>{i=!0,F==null||F(),Z()}}function Ca(s,i){if(s.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=s.filter(f=>!f.wire);if(r.length>0){const f=r.map(q=>q.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const t=i.networks??[],h=f=>{const N=t.find(q=>q.ChainID===f);return N?N.Name:`chain ${f}`},$=Pa(s.map(f=>h(f.wire.ChainID))),P=s.length===1?"machine":"machines";return{level:"ok",sentence:`All ${s.length} ${P} healthy — ${Ia($)}.`,machines:[]}}function Ea(s,i){const r=i.machines.length?` <span class="verdict-machines">${i.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${n(t)}</a>`).join(" ")}</span>`:"";s.innerHTML=`
    <div class="verdict-line verdict-${i.level}">
      ${D(i.level==="ok"?"OK":"Attention",i.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(i.sentence)}</strong>${r}
    </div>
  `}function Pa(s){return[...new Set(s)]}function Ia(s){return s.length<=1?s[0]??"":s.length===2?`${s[0]} and ${s[1]}`:`${s.slice(0,-1).join(", ")} and ${s[s.length-1]}`}const Ra="local";function La(s){let i=!1,r=!1,t="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${de()}
  `;const $=s.querySelector("#targets-body");ye(s,(p,w)=>{j(p,w)}),P();async function P(){try{const[p,w,S]=await Promise.all([Ce(),xe(),gt()]);if(i)return;t=S.os,N(p,w)}catch(p){if(i)return;$.innerHTML=`<p class="error">Failed to load machines: ${n(String(p))}</p>`}}function f(){h&&N(h.targets,h.catalog)}function N(p,w){h={targets:p,catalog:w};const S=t==="linux",B=[...p].sort((W,ae)=>(W.mode==="local"?-1:0)-(ae.mode==="local"?-1:0)),F=B.length?`<div class="card-grid">${B.map(W=>Aa(W,w,W.mode!=="local"||S,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',X=p.some(W=>W.mode==="local");$.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${F}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${q(S,X)}
        ${r?Na():""}
      </section>
    `;const te=$.querySelector("#fleet-verdict");te&&Ea(te,Ca(p,w))}function q(p,w){const S=`
      <div class="card">
        <h3>A server over SSH ${D("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${p?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${p?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,B=p?`
        <div class="card">
          <h3>This machine ${D("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${n(t)})`:""} ${D("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return w?`<div class="card-grid card-grid-wide">${S}</div>`:`<div class="card-grid card-grid-wide">${p?B+S:S+B}</div>`}async function j(p,w){var S;if(p==="add-local"){await I();return}if(p==="delete-target"){const B=w.dataset.id;if(!B||!await Be({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await M(B);return}if(p==="toggle-ssh"){r=!r,C(),f(),r&&((S=s.querySelector("#ssh-host"))==null||S.focus());return}p==="add-ssh"&&await O()}async function I(){C();try{await dt({id:Ra,mode:"local"}),await P()}catch(p){L(p)}}async function M(p){try{await mn(p),await P()}catch(w){L(w)}}async function O(){const p=s.querySelector("#ssh-host"),w=s.querySelector("#ssh-user"),S=s.querySelector("#ssh-key"),B=s.querySelector("#ssh-port"),F=s.querySelector("#ssh-id");if(!p||!w||!S||!B||!F)return;const X=p.value.trim(),te=w.value.trim(),W=S.value.trim(),ae=B.value.trim(),ue=F.value.trim();if(C(),!X||!te||!W){L(new Error("host, user, and key path are required"));return}const J=ue||Ba(X),se={Host:X,User:te,KeyPath:W};if(ae){const le=Number.parseInt(ae,10);if(!Number.isFinite(le)||le<=0){L(new Error("port must be a positive number"));return}se.Port=le}const oe=s.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await dt({id:J,mode:"ssh",ssh:se}),r=!1,await P()}catch(le){L(le),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function L(p){let w=s.querySelector("#targets-error");w||($.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),w=s.querySelector("#targets-error")),w.textContent=String(p instanceof Error?p.message:p)}function C(){var p;(p=s.querySelector("#targets-error"))==null||p.remove()}return()=>{i=!0}}function Aa(s,i,r,t){const h=s.wire,$=s.mode==="local"?"this machine":"SSH",P=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:$;let f;if(!h&&!r)f=`${D("can't run a node","warn")} ${D(t||"not Linux","neutral")}`;else if(!h)f=D("not set up","neutral");else{const N=i.networks.find(j=>j.ChainID===h.ChainID),q=N?N.Name:`chain ${h.ChainID}`;f=`${D(q,"ok")} ${D(h.ExecID,"neutral")} ${D(h.BeaconID,"neutral")}${h.Archive?" "+D("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${P}</p>
      <p>${f}</p>
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
  `}function Ba(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Ha=document.querySelector("#app"),{contentEl:Da,setActiveNav:Ma}=Yn(Ha);let fe=null;function Ua(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"home"};const[r,t]=i;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:t?decodeURIComponent(t):void 0}:{screen:r??"targets"}}function ve(s){const i=document.createElement("div");return Da.replaceChildren(i),s(i)}function vt(){if(fe){try{fe()}catch{}fe=null}const{screen:s,id:i}=Ua();switch(Ma(s),s){case"machine":if(!i){location.hash="#/targets";return}fe=ve(r=>ua(r,i));break;case"setup":case"dash":case"logs":case"services":if(!i){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(i)}`;return;case"security":if(!i){location.hash="#/targets";return}fe=ve(r=>ya(r,i));break;case"diag":if(!i){location.hash="#/targets";return}fe=ve(r=>ta(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}fe=ve(r=>ea(r,i));break;case"rpc":fe=ve(r=>xa(r));break;case"settings":fe=ve(r=>$a(r));break;case"targets":fe=ve(r=>La(r));break;case"panel":fe=ve(r=>yt(r));break;case"home":default:fe=ve(r=>yt(r));break}}window.addEventListener("hashchange",vt);vt();
