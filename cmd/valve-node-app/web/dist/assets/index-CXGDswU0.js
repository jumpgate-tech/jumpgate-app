var sn=Object.defineProperty;var on=(s,i,r)=>i in s?sn(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var qe=(s,i,r)=>on(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))t(h);new MutationObserver(h=>{for(const g of h)if(g.type==="childList")for(const P of g.addedNodes)P.tagName==="LINK"&&P.rel==="modulepreload"&&t(P)}).observe(document,{childList:!0,subtree:!0});function r(h){const g={};return h.integrity&&(g.integrity=h.integrity),h.referrerPolicy&&(g.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?g.credentials="include":h.crossOrigin==="anonymous"?g.credentials="omit":g.credentials="same-origin",g}function t(h){if(h.ep)return;h.ep=!0;const g=r(h);fetch(h.href,g)}})();function mt(){return J("/api/host")}function we(){return J("/api/catalog")}function ke(){return J("/api/targets")}function ct(s){return J("/api/targets",{method:"POST",headers:me,body:JSON.stringify(s)})}function rn(s){return J(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function cn(s,i){return J(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function ln(s,i){return J(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:me,body:JSON.stringify(i)})}function Qe(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function dn(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function un(s,i=200){return J(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function pn(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function lt(s,i){const r=i===void 0?{}:{lines:i};return J(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:me,body:JSON.stringify(r)})}function hn(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function fn(s,i){return J(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})})}function mn(s){return J(`/api/targets/${encodeURIComponent(s)}/du`)}function bn(s){return J(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function gn(s){return J(`/api/targets/${encodeURIComponent(s)}/firewall`)}function yn(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function vn(s){return J(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function $n(s){return J(`/api/targets/${encodeURIComponent(s)}/containers`)}function wn(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function kn(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})}),t=await r.text();let h=null;try{h=t?JSON.parse(t):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const g=h&&typeof h=="object"&&typeof h.error=="string"?h.error:r.statusText||`HTTP ${r.status}`;throw new Se(r.status,g)}function Sn(s,i){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function Tn(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:me}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const h=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new Se(i.status,h)}function Cn(s,i,r){return J(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:me,body:JSON.stringify(r)})}function bt(){return J("/api/gateways")}async function xn(s){await J(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function Pn(s){return J("/api/gateways",{method:"POST",headers:me,body:JSON.stringify(s)})}function En(s){return J(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function In(s){return J(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function Rn(s){return J(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function Ln(s,i=!1){const r=i?"?refresh=1":"";return J(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function An(s){return J(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function Nn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:me,body:JSON.stringify(i)})}function Bn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function Hn(s){return J(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function Dn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:s})}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const h=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new Se(i.status,h)}function Un(s){return J(`/api/chainlist/${s}`)}function Mn(s,i){return J(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function On(){return J("/api/settings")}function qn(s){return J("/api/settings",{method:"PUT",headers:me,body:JSON.stringify(s)})}class Se extends Error{constructor(r,t,h,g){super(t);qe(this,"status");qe(this,"hint");qe(this,"code");this.name="ApiError",this.status=r,this.hint=h,this.code=g}}const me={"Content-Type":"application/json"};async function J(s,i){const r=await fetch(s,i);if(!r.ok){let h=r.statusText||`HTTP ${r.status}`,g,P;try{const f=await r.json();f&&typeof f.error=="string"&&f.error&&(h=f.error),f&&typeof f.hint=="string"&&f.hint&&(g=f.hint),f&&typeof f.code=="string"&&f.code&&(P=f.code)}catch{}throw new Se(r.status,h,g,P)}if(r.status===204)return;const t=await r.text();return t?JSON.parse(t):void 0}const dt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ce(s,i){const r=s&&i&&i!==dt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(dt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function Fn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{const g=h==="machine"?"targets":h==="home"?"rpc":h;for(const P of r)P.classList.toggle("active",P.dataset.nav===g)}}}function ie(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function jn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function Wn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),t=i%60;if(r===0)return`~${t}m`;if(r<48)return`~${r}h ${t}m`;const h=Math.floor(r/24),g=r%24;return`~${h}d ${g}h`}function D(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function $e(s){return`<span class="dot dot-${s}"></span>`}const ut=["B","KB","MB","GB","TB","PB"];function xe(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<ut.length-1;)i/=1024,r++;const t=i<10?2:i<100?1:0;return`${i.toFixed(t)} ${ut[r]}`}async function De(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ye(s,i){s.addEventListener("click",r=>{const t=r.target.closest("[data-action]");if(!t||!s.contains(t))return;const h=t.dataset.action;h&&i(h,t,r)})}function Ze(s,i,r){const t=i.find(g=>g.value===r),h=i.map(g=>`
      <li class="dropdown-option${g.value===r?" selected":""}" role="option"
          aria-selected="${g.value===r}" data-value="${n(g.value)}">
        ${n(g.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Ae(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function et(s,i){s.addEventListener("click",h=>{const g=h.target,P=g.closest(".dropdown-trigger");if(P&&s.contains(P)){const B=P.closest(".dropdown"),q=!!B&&!B.classList.contains("open");Ae(s),B&&q&&(B.classList.add("open"),P.setAttribute("aria-expanded","true"));return}const f=g.closest(".dropdown-option");if(f&&s.contains(f)){const B=f.closest(".dropdown");Ae(s),i((B==null?void 0:B.dataset.dropdown)??"",f.dataset.value??"");return}Ae(s)});const r=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}const g=h.target;(!g.closest(".dropdown")||!s.contains(g))&&Ae(s)},t=h=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}h.key==="Escape"&&Ae(s)};document.addEventListener("click",r),document.addEventListener("keydown",t)}const Ke="app-modal";let _e=null;function re(s,i){Z();const r=document.createElement("div");r.className="modal-overlay",r.id=Ke,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",h=>{const g=h.target.closest("[data-modal-action]");g!=null&&g.dataset.modalAction?i(g.dataset.modalAction):h.target===r&&i("cancel")});const t=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",t),_e=t,document.body.appendChild(r)}function Z(){var s;(s=document.getElementById(Ke))==null||s.remove(),_e&&(document.removeEventListener("keydown",_e),_e=null)}function He(){return document.querySelector(`#${Ke} .modal`)}function Be(s){return new Promise(i=>{var h;let r=!1;const t=g=>{r||(r=!0,Z(),i(g))};re(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,g=>t(g==="confirm")),(h=document.querySelector(`#${Ke} [data-modal-action="confirm"]`))==null||h.focus()})}const ze=5e3,_n=60;function Kn(s,i){let r=!1,t=null,h=null,g=null,P=null;const f=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ce()}</div>`;const B=s.querySelector("#an-body");ye(s,(v,d)=>{var S;v==="toggle-endpoint"&&((S=d.closest(".an-endpoint"))==null||S.classList.toggle("expanded"))}),q();async function q(){try{t=((await bt()).gateways??[]).find(d=>d.id===i)??null}catch(v){if(r)return;g=String(v instanceof Error?v.message:v),U();return}if(!r){if(!t){U();return}await F(),P=window.setInterval(()=>void F(),ze)}}async function F(){try{const v=await Rn(i);if(r)return;I(v),h=v,g=null}catch(v){if(r)return;g=String(v instanceof Error?v.message:v)}U()}function I(v){if(!v.enabled||v.error)return;const d=f[f.length-1];d&&d.since!==v.since&&(f.length=0);const S=new Map;for(const R of v.networks??[])S.set(R.chainId,R.received);f.push({t:Date.now(),since:v.since,received:S}),f.length>_n&&f.shift()}function U(){r||(B.innerHTML=O())}function O(){return g&&!h?`<h1>Analytics</h1><p class="error">${n(g)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${A(t)}
      ${h?u(h):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(i)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function A(v){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${n(v.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${x()}</div>
      </div>
    `}function x(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const v=h.since?new Date(h.since):null;return v&&!Number.isNaN(v.getTime())?`totals since the gateway started, ${n(v.toLocaleString())}<br />re-read every ${ze/1e3}s`:`re-read every ${ze/1e3}s`}function u(v){return v.enabled?v.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(v.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:y(v)+de(v):`
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
        ${d.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':d.map(S=>C(S)).join("")}
      </section>
    `}function C(v){const d=v.methods??[],S=v.endpoints??[],R=v.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${v.chainId}</span>
          <span class="band-name">${n(v.name)}</span>
          ${G(v)}
        </div>
        <div class="an-stats">
          ${L("Received",ie(v.received),"what clients asked this chain for")}
          ${L("Answered",ie(v.answered),"returned by one of your endpoints")}
          ${L("From cache",ie(v.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${L("Failed",ie(v.failed),"asked for and never answered",v.failed>0?"bad":"")}
        </div>
        ${Y(v.chainId)}
        ${R?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ne("Method",d.map(H=>({label:H.method,l:H})))+ne("Endpoint",S.map(H=>({label:H.upstream,l:H})))+V(v)}
      </div>
    `}function L(v,d,S,R=""){return`
      <div class="an-stat${R?" an-stat-"+R:""}" title="${n(S)}">
        <span class="an-stat-n">${n(d)}</span>
        <span class="an-stat-l">${n(v)}</span>
      </div>
    `}function G(v){const d=Q(v.chainId);if(d===null)return'<span class="an-rate muted small">measuring rate…</span>';const S=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${S}s apart.">
      ${n(d.toFixed(d<10?2:0))} req/s <span class="muted">over the last ${S}s</span>
    </span>`}function Q(v){if(f.length<2)return null;const d=f[0],S=f[f.length-1],R=(S.t-d.t)/1e3;if(R<=0)return null;const H=(S.received.get(v)??0)-(d.received.get(v)??0);return H<0?null:H/R}function Y(v){if(f.length<3)return"";const d=[];for(let $=1;$<f.length;$++){const E=f[$-1],j=f[$],l=(j.t-E.t)/1e3,b=(j.received.get(v)??0)-(E.received.get(v)??0);d.push(l>0&&b>=0?b/l:0)}const S=Math.max(...d);if(S<=0)return"";const R=240,H=28,_=d.length>1?R/(d.length-1):R,m=d.map(($,E)=>`${(E*_).toFixed(1)},${(H-$/S*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${S.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${R} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${m}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(S.toFixed(2))} req/s</span>
      </div>
    `}function V(v){const d=[];return v.cached.count>0&&d.push(`${n(ie(v.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${v.cached.mean===null?"":`, in ${n(Ne(v.cached.mean))} on average`}.`),v.failedLatency.count>0&&v.failedLatency.mean!==null&&d.push(`The ${n(ie(v.failedLatency.count))} that failed took
         ${n(Ne(v.failedLatency.mean))} on average to fail.`),d.length===0?"":`<p class="muted small">${d.join(" ")}</p>`}function ne(v,d){return d.length===0?"":`
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
            ${d.map(S=>le(S.label,S.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function le(v,d){return`
      <tr>
        <td><code>${n(v)}</code></td>
        <td class="an-num">${ie(d.count)}</td>
        <td class="an-num">${d.mean===null?'<span class="muted">—</span>':n(Ne(d.mean))}</td>
        <td>${W(d)}</td>
      </tr>
    `}function W(v){const d=v.buckets??[];if(d.length===0||v.count===0)return'<span class="muted small">—</span>';let S=0;const R=[];for(const _ of d){const m=_.count-S;S=_.count,R.push({label:oe(_.le),n:Math.max(0,m)})}return R.reduce((_,m)=>_+m.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(R.filter(_=>_.n>0).map(_=>`${_.n} ${_.label}`).join(" · "))}">
        ${R.map((_,m)=>_.n===0?"":`<span class="an-band an-band-${Math.min(m,4)}" style="flex:${_.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ae(R))}</span>
    `}function ae(v){for(let d=v.length-1;d>=0;d--)if(v[d].n>0)return`slowest ${v[d].label}`;return""}function oe(v){if(v==="+Inf")return"30s or more";const d=Number(v);return Number.isFinite(d)?`under ${Ne(d)}`:`under ${v}`}function de(v){const d=v.endpoints??[];return`
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
                     <tbody>${d.map(S=>be(S)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function be(v){const d=v.errors??[],S=d.reduce((H,_)=>H+_.count,0),R=d.length>0;return`
      <tr class="an-endpoint${R?" expandable":""}" ${R?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(v.upstream)}</code>
          ${v.chainId?`<span class="muted small">chain ${v.chainId}</span>`:""}
          ${v.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ie(v.requests)}</td>
        <td class="an-num${S>0?" bad":""}">${S>0?ie(S):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${v.headLag>0?ie(v.headLag):'<span class="muted">0</span>'}</td>
        <td>${he(v)}</td>
      </tr>
      ${R?ge(v,d):""}
    `}function he(v){const d=[];return v.scored?(d.push(v.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(v.position))}</span>`),d.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(v.score.toFixed(3))}</span>`),v.primarySwitches>1&&d.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ie(v.primarySwitches)} switches</span>`),v.excludedSeconds>0&&d.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Ne(v.excludedSeconds))}</span>`),`<span class="an-selection">${d.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(v,d){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${d.map(S=>`
                    <tr>
                      <td class="an-num">${ie(S.count)}</td>
                      <td><code>${n(S.class)}</code></td>
                      <td>${S.severity?`<span class="badge badge-${S.severity==="critical"?"bad":"warn"}">${n(S.severity)}</span>`:""}</td>
                      <td class="muted small">${n(S.method||"")}</td>
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
    `}return()=>{r=!0,P!==null&&window.clearInterval(P)}}function Ne(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}function Gn(s,i){let r=!1,t=null,h=null,g=!1,P=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ce()}</div>`;const f=s.querySelector("#diag-body"),B=s.querySelector("#diag-footer");ye(s,(u,y)=>{var C;if(u==="run")F();else if(u==="toggle")(C=y.closest(".check-item"))==null||C.classList.toggle("expanded");else if(u==="copy"){const L=y.dataset.copy;L&&x(y,L)}}),q();async function q(){let u,y;try{const[L,G]=await Promise.all([ke(),we()]);u=L.find(Q=>Q.id===i),y=G}catch(L){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(L))}</p>`;return}if(r)return;if(!u){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const C=y==null?void 0:y.networks.find(L=>L.ChainID===u.wire.ChainID);C&&(B.innerHTML=ce(C.Name,C.LearnURL));try{t=await vn(i),P=!0}catch(L){h=String(L instanceof Error?L.message:L)}r||I()}async function F(){g=!0,h=null,I();try{t=await yn(i),P=!0}catch(u){h=String(u instanceof Error?u.message:u)}g=!1,r||I()}function I(){f.innerHTML=`
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
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${U()}
    `}function U(){if(!P&&!h)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const u=new Date(t.at).toLocaleString(),y=t.failedId?`<p><strong>Failed at: ${n(O(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(u)} — trigger: ${n(t.trigger)}</p>
      ${y}
      <ul class="check-list">${t.items.map(A).join("")}</ul>
    `}function O(u){var y;return((y=t==null?void 0:t.items.find(C=>C.ID===u))==null?void 0:y.Title)??u}function A(u){const y=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral",C=u.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${C?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${D(C?"failed here":u.Status,y)}
          <strong>${n(u.Title)}</strong>
          <span class="muted small check-detail-inline">${n(u.Detail)}</span>
        </button>
        <div class="check-body">
          <details${C?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(u.Why)}</p>
          </details>
          ${u.Fix?`
                <details${C?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(u.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(u.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function x(u,y){const C=await De(y),L=u.textContent;u.textContent=C?"Copied!":"Copy failed",setTimeout(()=>{r||(u.textContent=L)},1500)}return()=>{r=!0}}function gt(s,i){if(s.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=s.filter(f=>!f.wire);if(r.length>0){const f=r.map(q=>q.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const t=i.networks??[],h=f=>{const B=t.find(q=>q.ChainID===f);return B?B.Name:`chain ${f}`},g=Vn(s.map(f=>h(f.wire.ChainID))),P=s.length===1?"machine":"machines";return{level:"ok",sentence:`All ${s.length} ${P} healthy — ${zn(g)}.`,machines:[]}}function yt(s,i){const r=i.machines.length?` <span class="verdict-machines">${i.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${n(t)}</a>`).join(" ")}</span>`:"";s.innerHTML=`
    <div class="verdict-line verdict-${i.level}">
      ${D(i.level==="ok"?"OK":"Attention",i.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${n(i.sentence)}</strong>${r}
    </div>
  `}function Vn(s){return[...new Set(s)]}function zn(s){return s.length<=1?s[0]??"":s.length===2?`${s[0]} and ${s[1]}`:`${s.slice(0,-1).join(", ")} and ${s[s.length-1]}`}function Jn(s,i){const r=i==="linux";return s.some(h=>h.mode==="ssh"||h.mode==="local"&&r)||r}function Yn(s){let i=!1;s.innerHTML='<div id="home-body"><p class="muted">Loading…</p></div>';const r=s.querySelector("#home-body");t();async function t(){let g,P,f;try{const[B,q,F]=await Promise.all([ke(),we(),mt()]);g=B,P=q,f=F.os}catch(B){if(i)return;r.innerHTML=`<p class="error">Failed to load: ${n(String(B))}</p>`;return}if(!i){if(Jn(g,f)){location.hash="#/targets";return}h(g,P)}}function h(g,P){r.innerHTML=`
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
    `;const f=r.querySelector("#fleet-verdict");f&&yt(f,gt(g,P))}return()=>{i=!0}}const Zn=85,Je={exec:"Execution",beacon:"Beacon"};function Xn(s,i){let r=!1,t=null,h=null,g=null,P=null,f=null,B=null,q=null,F=null;const I={exec:null,beacon:null};let U=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ce()}</div>`;const O=s.querySelector("#dash-body"),A=s.querySelector("#dash-footer");O.addEventListener("click",d=>{const S=d.target.closest("[data-action]");if(!S||!O.contains(S))return;const R=S.dataset.action;if(R==="svc-action"){const H=S.dataset.svc,_=S.dataset.kind;H&&_&&be(H,_)}else if(R==="open-clear"){const H=S.dataset.svc;H&&ge(H)}else if(R==="copy"){const H=S.dataset.copy;H&&he(S,H)}else R==="retry-du"?u():R==="retry-endpoints"&&y()}),x();async function x(){let d,S;try{const[H,_]=await Promise.all([ke(),we()]);d=H.find(m=>m.id===i),S=_}catch(H){if(r)return;O.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(r)return;if(!d){O.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){O.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const R=S==null?void 0:S.networks.find(H=>H.ChainID===d.wire.ChainID);R&&(A.innerHTML=ce(R.Name,R.LearnURL)),O.innerHTML='<p class="muted">Connecting…</p>',t=dn(i,H=>{r||(C(H),h=H,g=H,L())}),u(),y()}async function u(){B=null;try{f=await mn(i)}catch(d){f=null,B=String(d instanceof Error?d.message:d)}r||L()}async function y(){F=null;try{q=await bn(i)}catch(d){q=null,F=String(d instanceof Error?d.message:d)}r||L()}function C(d){if(!h)return;const S=(new Date(d.at).getTime()-new Date(h.at).getTime())/1e3,R=d.execHead-h.execHead;if(S>0&&R>=0){const H=R/S;P=P===null?H:P*.7+H*.3}}function L(){if(!g)return;const d=g;O.innerHTML=`
      <p class="dash-status">${G(d)}</p>
      <div class="card-grid">
        ${oe(d)}
        ${Y(d)}
        ${V(d)}
        ${ne(d)}
        ${le(d)}
        ${W()}
      </div>
      <p class="muted small">Last updated ${n(new Date(d.at).toLocaleTimeString())}</p>
    `}function G(d){return!d.execActive&&!d.beaconActive?D("Node not running","bad"):d.execSyncing||d.beaconDistance>0?D("Syncing","warn"):D("Running · synced","ok")}function Q(d){const R=d.refHead>0?d.refHead-d.execHead:null,H=R!==null&&R>0&&P&&P>0?Wn(R/P):R!==null&&R<=0?"caught up":"—";return{lag:R,eta:H}}function Y(d){const{lag:S,eta:R}=Q(d);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${d.execActive?d.execSyncing?D("syncing","warn"):d.execHead===0?D("no data","neutral"):D("synced","ok"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ie(d.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${S!==null?ie(d.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${S!==null?ie(Math.max(S,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${R}</dd></div>
        </dl>
      </div>
    `}function V(d){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${d.beaconActive?d.beaconSlot===0?D("no data","neutral"):d.beaconDistance===0?D("synced","ok"):D("syncing","warn"):D("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ie(d.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ie(d.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ne(d){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ie(d.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ie(d.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function le(d){const S=d.diskUsedPct>=Zn,R=`
      <div class="meter"><div class="meter-fill ${S?"meter-warn":""}" style="width:${Math.min(d.diskUsedPct,100)}%"></div></div>
      <p>${jn(d.diskUsedPct)} used</p>
    `;if(B)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="error small">${n(B)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${R}
          <p class="muted">Loading…</p>
        </div>
      `;const H=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,_=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:m,eta:$}=Q(d),E=m!==null&&m>0&&P!==null&&P>0;return`
      <div class="card ${S?"card-warn":""}">
        <h3>Storage</h3>
        ${R}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${xe(f.ExecBytes)} of ~${xe(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${E?`<p class="muted small">Estimated time remaining: ${n($)}</p>`:""}
        <p class="muted small">Beacon — ${xe(f.BeaconBytes)} of ~${xe(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${_}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${xe(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function W(){if(F)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(F)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!q)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const d=q,S=d.ExecReachable&&!d.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",R=d.Access==="ssh"?`
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
        ${S}
        ${R}
      </div>
    `}function ae(d,S){const R=Je[d],H=I[d],_=(m,$,E)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${d}" data-kind="${m}" ${H!==null||E?"disabled":""}>${H===m?de():n($)}</button>`;return`
      <div class="service-row">
        <span>${n(R)} ${S?D("active","ok"):D("down","bad")}</span>
        <div class="service-actions">
          ${_("start","Start",S)}
          ${_("stop","Stop",!S)}
          ${_("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${d}" ${H!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(d){return`
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
    `}function de(){return'<span class="spinner" aria-label="working"></span>'}async function be(d,S){if(I[d]===null){I[d]=S,U=null,L();try{await hn(i,d,S)}catch(R){U=`${Je[d]} ${S} failed: ${R instanceof Error?R.message:String(R)}`}I[d]=null,r||L()}}async function he(d,S){const R=await De(S),H=d.textContent;d.textContent=R?"Copied!":"Copy failed",setTimeout(()=>{r||(d.textContent=H)},1500)}function ge(d){const S=Je[d],R=f?xe(d==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";re(`
        <h2>Clear ${n(S)} data</h2>
        <p class="error">
          This stops the ${n(S.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(R)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(d)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,m=>{if(m==="cancel"){Z();return}m==="confirm"&&v(d)});const H=document.getElementById("clear-confirm-input"),_=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{_&&(_.disabled=H.value.trim()!==d)}),H==null||H.focus()}async function v(d){const S=document.getElementById("clear-confirm-btn");S&&(S.disabled=!0,S.textContent="Clearing…");try{await fn(i,d),Z(),u()}catch(R){const H=He();if(H){const _=document.createElement("p");_.className="error small",_.textContent=`Clear failed: ${R instanceof Error?R.message:String(R)}`,H.appendChild(_)}S&&(S.disabled=!1,S.textContent="Clear and resync")}}return()=>{r=!0,t==null||t(),Z()}}const pt=500,ht="valve-node-app.explain-consent";function Qn(s,i){let r=!1,t=null;const h=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ce()}</div>
  `;const g=s.querySelector("#logs-body"),P=s.querySelector("#logs-footer");ye(s,x=>{x==="explain"&&F()}),f();async function f(){let x,u;try{const[C,L]=await Promise.all([ke(),we()]);x=C.find(G=>G.id===i),u=L}catch(C){if(r)return;g.innerHTML=`<p class="error">Failed to load target: ${n(String(C))}</p>`;return}if(r)return;if(!x){g.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!x.wire){g.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const y=u==null?void 0:u.networks.find(C=>C.ChainID===x.wire.ChainID);y&&(P.innerHTML=ce(y.Name,y.LearnURL));try{const C=await un(i,200);if(r)return;h.push(...C)}catch(C){if(r)return;g.innerHTML=`<p class="error">Failed to load logs: ${n(String(C))}</p>`;return}B(),t=pn(i,C=>{r||(h.push(C),h.length>pt&&h.splice(0,h.length-pt),B())})}function B(){const x=h.filter(y=>y.severity==="error"||y.severity==="critical");g.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${h.map(q).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${D(String(x.length),x.length?"bad":"neutral")}</h2>
          <div class="log-lines">${x.length?x.slice().reverse().map(q).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=g.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function q(x){const u=x.severity||"info",y=x.learnUrl?` <a href="${n(x.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(u)}">
        <span class="log-time">${n(new Date(x.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(x.unit)}</span>
        <span class="log-sev">${n(u)}</span>
        <span class="log-text">${n(x.line)}</span>
        ${x.explain?`<div class="log-explain">${n(x.explain)}${y}</div>`:""}
      </div>
    `}async function F(){const x=h.filter(y=>y.severity==="error"||y.severity==="critical").map(y=>y.line).slice(-40);if(!(localStorage.getItem(ht)==="1")){I(x);return}await U(x)}function I(x){const u=x.length?`<pre class="explain-excerpt">${x.map(y=>n(y)).join(`
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
    `,y=>{y==="proceed"?(localStorage.setItem(ht,"1"),A(),U(x)):A()})}async function U(x){O('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=x.length?await lt(i,x):await lt(i);if(r)return;O(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map(y=>n(y)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,y=>{y==="close"&&A()})}catch(u){if(r)return;if(u instanceof Se&&u.status===409){O(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,y=>{y==="close"&&A()});return}O(`
        <h2>Explain failed</h2>
        <p class="error">${n(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,y=>{y==="close"&&A()})}}function O(x,u){A();const y=document.createElement("div");y.className="modal-overlay",y.id="explain-modal",y.innerHTML=`<div class="modal">${x}</div>`,y.addEventListener("click",C=>{const L=C.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&u(L.dataset.modalAction),C.target===y&&u("cancel")}),document.body.appendChild(y)}function A(){var x;(x=document.getElementById("explain-modal"))==null||x.remove()}return()=>{r=!0,t==null||t(),A()}}const ea="run",ta={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},na={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function aa(s,i){let r=!1,t=null,h=null;const g={devnet:null},P={devnet:null},f={devnet:[]};let B=null;const q={devnet:!1};let F=null;const I={devnet:null},U={devnet:null};s.innerHTML=`
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
  `;const O=s.querySelector("#services-body");ye(s,(l,b)=>{ge(l,b)}),A();async function A(){try{const l=await $n(i);if(r)return;t=l,h=null}catch(l){if(r)return;t=null,h=E(l)}u()}function x(l){return t==null?void 0:t.services.find(b=>b.id===l)}function u(){if(!r){if(h){O.innerHTML=`<p class="error">Could not read this machine's services: ${n(h)}</p>`;return}if(!t){O.innerHTML='<p class="muted">Loading…</p>';return}O.innerHTML=`
      ${y(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(C).join("")}
      </div>
    `}}function y(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${n(l.flavor)}${l.serverVersion?` ${n(l.serverVersion)}`:""} · reachable</p>`;const b=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(b)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${n(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function C(l){const b=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(l.label)}</h2>
          ${L(l)}
        </div>
        <p class="muted small">${n(ta[l.id]??"")}</p>

        ${l.error?G(l):""}
        ${l.blocked?`<div class="banner banner-warn">${n(l.blocked)}</div>`:""}
        ${b.map(N=>`<div class="banner banner-warn">${n(N)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${n(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Q(l)}

        ${Y(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(N=>V(l,N)).join("")}
        </div>
        ${P[l.id]?`<p class="error small">${n(P[l.id])}</p>`:""}
        ${ne(l)}

        ${le(l)}
      </div>
    `}function L(l){switch(l.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function G(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(l.error??"")}</div>
        ${l.hint?`<div class="small">${n(l.hint)}</div>`:""}
      </div>
    `}function Q(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const b=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${b}.</p>`}function Y(l){const b=l.endpoints??[];return b.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":b.map(N=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(N.label)}</span>
          <code class="endpoint-url">${n(N.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(N.url)}">Copy</button>
        </div>`).join("")}function V(l,b){const N=na[b];if(!N)return"";const K=g[l.id],X=b==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:N.label;return`
      <button class="${N.className}" data-action="svc-${b}" data-svc="${n(l.id)}"
              title="${n(N.title)}" ${K?"disabled":""}>
        ${K===b?'<span class="spinner" aria-label="working"></span>':n(X)}
      </button>
    `}function ne(l){const b=f[l.id]??[];return b.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(b.join(`
`))}</pre>
      </div>
    `}function le(l){const b=q[l.id],N=W(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(l.id)}">
            ${b?"Close":"Edit"}
          </button>
        </div>
        ${b?ae():`<p class="small">${N}</p>`}
        ${I[l.id]?`<p class="error small">${n(I[l.id])}</p>`:""}
        ${U[l.id]?`<p class="muted small">${n(U[l.id])}</p>`:""}
      </div>
    `}function W(l){const b=l.devnet;return b?`Chain ${b.ChainID} · a block every ${n(b.BlockTime)} · JSON-RPC on ${n(b.BindAddr)}:${b.HTTPPort} · WebSocket on ${n(b.BindAddr)}:${b.WSPort}`:"—"}function ae(l){return oe()}function oe(){const l=F;return l?`
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
    `:""}function de(){q.devnet&&F&&(F.BlockTime=be("#dev-blocktime",F.BlockTime),F.HTTPPort=he("#dev-http",F.HTTPPort),F.WSPort=he("#dev-ws",F.WSPort),F.BindAddr=be("#dev-bind",F.BindAddr))}function be(l,b){const N=s.querySelector(l);return N?N.value.trim():b}function he(l,b){const N=s.querySelector(l);if(!N)return b;const K=Number.parseInt(N.value.trim(),10);return Number.isFinite(K)?K:b}async function ge(l,b){const N=b.dataset.svc??"";switch(l){case"refresh":await A();return;case"copy":b.dataset.copy&&await $(b,b.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await v(N,l.slice(4));return;case"svc-create":case"svc-recreate":await d(N);return;case"svc-wipe":H(N);return;case"toggle-config":S(N);return;case"save-config":await R(N);return;default:return}}async function v(l,b){if(!g[l]){g[l]=b,P[l]=null,u();try{await wn(i,l,b)}catch(N){P[l]=`${b} failed: ${E(N)}${j(N)}`}g[l]=null,await A()}}async function d(l){if(!g[l]){g[l]="create",P[l]=null,f[l]=["starting…"],u();try{await Sn(i,l)}catch(b){P[l]=`${E(b)}${j(b)}`,f[l]=[],g[l]=null,u();return}B==null||B(),B=Qe(i,b=>{if(r)return;const N=b.err?`${b.stepId}: ${b.err}`:b.line?`${b.stepId}: ${b.line}`:`${b.stepId}: done`;if(f[l]=[...(f[l]??[]).filter(X=>X!=="starting…"),N],!!b.err||b.stepId===ea&&!!b.done){B==null||B(),B=null,g[l]=null,b.err&&(P[l]="Provisioning failed — see the log below."),A();return}u()})}}function S(l){if(de(),q[l]=!q[l],I[l]=null,U[l]=null,q[l]){const b=x(l);b!=null&&b.devnet&&(F={...b.devnet})}u()}async function R(l){var K;de(),I[l]=null,U[l]=null;const b=F;if(!b)return;if(b.HTTPPort===b.WSPort){I[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await Cn(i,l,b)}catch(X){I[l]=E(X),u();return}const N=((K=x(l))==null?void 0:K.status.State)==="running";q[l]=!1,U[l]=N?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await A()}function H(l){const b=x(l);if(!b)return;const N=(b.restartsOnWipe??[]).map(M=>{var se;return((se=x(M))==null?void 0:se.label)??M});re(`
        <h2>Wipe ${n(b.label)}</h2>
        <p class="error">This deletes ${n(b.wipeDiscards)}</p>
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
      `,M=>{if(M==="cancel"||M==="close"){Z(),A();return}M==="confirm"&&_(l)});const K=document.getElementById("wipe-confirm-input"),X=document.getElementById("wipe-confirm-btn");K==null||K.addEventListener("input",()=>{X&&(X.disabled=K.value.trim()!==l)}),K==null||K.focus()}async function _(l){const b=document.getElementById("wipe-confirm-btn");b&&(b.disabled=!0,b.textContent="Wiping…");let N;try{N=await kn(i,l)}catch(K){const X=He();if(X){const M=document.createElement("p");M.className="error small",M.textContent=`Wipe failed: ${E(K)}${j(K)}`,X.appendChild(M)}b&&(b.disabled=!1,b.textContent=`Wipe ${l}`);return}m(l,N)}function m(l,b){const N=x(l),K=te=>{var Pe;return((Pe=x(te))==null?void 0:Pe.label)??te},X=[];X.push(b.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const te of b.report.VolumesRemoved??[])X.push(`Volume ${te} deleted.`);for(const te of b.report.VolumesAbsent??[])X.push(`Volume ${te} was already gone.`);b.report.Recreated&&X.push("Container re-created from your saved configuration.");const M=(b.report.Cascaded??[]).map(K),se=(b.report.CascadeSkipped??[]).map(K);re(`
        <h2>${n((N==null?void 0:N.label)??l)} wiped</h2>
        <ul class="plain-list">${X.map(te=>`<li>${n(te)}</li>`).join("")}</ul>
        ${M.length?`<p class="ok">Restarted in front of it: ${n(M.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${se.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(se.join(", "))}.</p>`:""}
        ${b.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(b.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,te=>{(te==="close"||te==="cancel")&&(Z(),A())})}async function $(l,b){const N=await De(b),K=l.textContent;l.textContent=N?"Copied!":"Copy failed",setTimeout(()=>{r||(l.textContent=K)},1500)}function E(l){return l instanceof Error?l.message:String(l)}function j(l){return l instanceof Se&&l.hint?` — ${l.hint}`:""}return()=>{r=!0,B==null||B(),Z()}}const Ye=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Fe=8545,je=5052,We=30303,sa=[369,943,1],ft={369:"default",943:"practise here first"};function oa(s,i){let r=!1;const t={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ce()}</div>`;const h=s.querySelector("#wizard-body"),g=s.querySelector("#wizard-footer");ye(s,(m,$)=>{he(m,$)}),et(s,(m,$)=>{m==="exec-select"?t.execId=$:m==="beacon-select"&&(t.beaconId=$),f()}),s.addEventListener("change",m=>{const $=m.target;$ instanceof HTMLInputElement&&($.id==="data-dir-input"?(ge(),V()):$.id==="checkpoint-toggle"?(t.checkpoint=$.checked,f()):$.id==="exec-snapshot-toggle"&&(t.execSnapshot=$.checked,f()))}),P();async function P(){try{const[m,$]=await Promise.all([we(),ke()]);if(r)return;t.catalog=m;const E=$.find(j=>j.id===i);E!=null&&E.wire&&(t.chainId=E.wire.ChainID,t.execId=E.wire.ExecID,t.beaconId=E.wire.BeaconID,t.archive=E.wire.Archive,E.wire.ExecHTTPPort&&(t.execHTTPPort=String(E.wire.ExecHTTPPort)),E.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(E.wire.BeaconHTTPPort)),E.wire.ExecP2PPort&&(t.execP2PPort=String(E.wire.ExecP2PPort)),E.wire.RPCBindAddr&&(t.rpcBindAddr=E.wire.RPCBindAddr)),f()}catch(m){if(r)return;t.loadError=String(m instanceof Error?m.message:m),f()}}function f(){if(t.loadError){h.innerHTML=`<p class="error">Failed to load: ${n(t.loadError)}</p>`;return}t.catalog&&(h.innerHTML=`
      ${_(t.step)}
      ${q()}
    `,B())}function B(){var $;const m=($=t.catalog)==null?void 0:$.networks.find(E=>E.ChainID===t.chainId);g.innerHTML=m?ce(m.Name,m.LearnURL):ce()}function q(){switch(t.step){case"network":return F();case"clients":return I();case"mode":return oe();case"review":return de();case"run":return be()}}function F(){const m=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${sa.map(E=>{const j=m.networks.find(N=>N.ChainID===E);if(!j)return"";const l=t.chainId===E,b=ft[E]?D(ft[E],E===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${E}" type="button">
          <h3>${n(j.Name)} <span class="muted">(chain ${E})</span></h3>
          ${b}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function I(){const m=t.catalog,$=m.networks.find(l=>l.ChainID===t.chainId);if(!$)return'<p class="error">Unknown network.</p>';(t.execId===null||!$.ExecClients.includes(t.execId))&&(t.execId=$.ExecClients[0]??null),(t.beaconId===null||!$.BeaconClients.includes(t.beaconId))&&(t.beaconId=$.BeaconClients[0]??null);const E=$.ExecClients.map(l=>le(l,m)),j=$.BeaconClients.map(l=>le(l,m));return`
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
          ${Ze("exec-select",E,t.execId)}
        </label>
        ${ae(t.execId,m)}
        <label>
          Beacon client
          ${Ze("beacon-select",j,t.beaconId)}
        </label>
        ${ae(t.beaconId,m)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function U(m){return m<=0?"—":m>=1?`~${m.toFixed(1)} TB`:`~${Math.round(m*1e3)} GB`}const O=1.1,A=.5,x="Valve reth snapshot",u="rough estimate";function y(m){return m.SnapshotSizeTB}function C(m){return m.SnapshotSizeTB*A}function L(m){return`<p class="muted small">${U(y(m))} is the measured size of Valve's reth snapshot for ${n(m.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function G(m){return{archive:y(m)*1e12*O,full:C(m)*1e12*O}}function Q(m,$){if(!m)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${n($)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${n($)}</code>: ${n(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==$)return"";const E=G(m),j=t.freeBytes>=E.archive,l=t.freeBytes>=E.full,b=`<p class="muted small">Free at <code>${n($)}</code>: <strong>${xe(t.freeBytes)}</strong> — archive ${j?"fits":"won't fit"} (${U(y(m))}, ${x}), full ${l?"fits":"won't fit"} (${U(C(m))}, ${u}).</p>`;let N="";return t.downgradeNote?N=`<p class="banner banner-warn">${n(t.downgradeNote)}</p>`:l||(N=`<p class="banner banner-warn">Neither full (${U(C(m))}, ${u}) nor archive (${U(y(m))}, ${x}) fits the free space here — choose a location with more room.</p>`),b+N}function Y(m,$){if(t.downgradeNote=null,!m||t.freeBytes===null)return;const E=G(m);t.archive&&t.freeBytes<E.archive&&t.freeBytes>=E.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${$} for archive (${U(y(m))}, ${x}) — switched to Full (${U(C(m))}, ${u}). Pick a location with more room to run archive.`)}async function V(){var E;if(t.chainId===null)return;const m=(E=t.catalog)==null?void 0:E.networks.find(j=>j.ChainID===t.chainId),$=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,f();try{const{freeBytes:j}=await cn(t.targetId,$);if(r)return;t.freeBytes=j,t.probedPath=$,Y(m,$)}catch(j){if(r)return;t.freeBytes=null,t.probedPath=$,t.diskError=String(j instanceof Error?j.message:j)}t.diskProbing=!1,f()}function ne(m){return m?/^https?:\/\/.+/i.test(m)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function le(m,$){const E=$.clients.find(j=>j.id===m);return{value:m,label:E?`${E.id} — ${W(E.repo)}`:m}}function W(m){const $=m.split("/");return $.length>=4?$[3]:m}function ae(m,$){const E=m?$.clients.find(l=>l.id===m):void 0;if(!E)return"";const j=E.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(E.repo)}" target="_blank" rel="noopener noreferrer">${n(j)}</a></p>`}function oe(){var K,X,M;const m=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",$=(K=t.catalog)==null?void 0:K.networks.find(se=>se.ChainID===t.chainId),E=((M=(X=t.catalog)==null?void 0:X.clients.find(se=>se.id===t.execId))==null?void 0:M.snapshotSupported)??!1,j=$?`${U(C($))} (${u})`:"Smaller",l=$?`${U(y($))} (${x})`:"Much larger",b=$?` on ${n($.Name)}`:"",N=$?t.checkpoint?$.SyncLabel:$.GenesisSyncLabel:"";return`
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
          ${$?`<p class="sync-estimate">⏱ Estimated initial sync${b}: <strong>${n(N)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n(($==null?void 0:$.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n(($==null?void 0:$.CheckpointURL)??"")}" value="${n(t.checkpointUrl)}" />
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
              <tr><th>Approx. disk footprint${b}</th><td class="yes">${j}</td><td class="limited">${l}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${$?L($):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${$?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${j}${$?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(m)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(m)}" value="${n(t.dataDir)}" />
          </label>
          ${Q($,t.dataDir||m)}
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
    `}function de(){const $=t.catalog.networks.find(te=>te.ChainID===t.chainId),E=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,j=t.jwtPath||`${E}/jwt.hex`,l=Ye.map(te=>`<li>${n(te.title)}</li>`).join(""),b=R(t.execHTTPPort,Fe),N=R(t.beaconHTTPPort,je),K=R(t.execP2PPort,We),X=b||N||K?`<tr><th>Non-default ports</th><td>${[b?`exec HTTP ${b}`:null,N?`beacon HTTP ${N}`:null,K?`exec p2p ${K}`:null].filter(te=>te!==null).map(n).join(", ")}</td></tr>`:"",{addr:M}=v(t.rpcBindAddr),se=M?`<tr><th>RPC bind address</th><td><code>${n(M)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${n(($==null?void 0:$.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(E)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(j)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${n(t.checkpointUrl||($==null?void 0:$.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${X}
            ${se}
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
    `}function be(){const $=t.catalog.networks.find(M=>M.ChainID===t.chainId),E=$==null?void 0:$.LearnURL,j=new Set(t.events.filter(M=>M.done).map(M=>M.stepId)),l=new Set(t.events.filter(M=>M.err).map(M=>M.stepId)),b=new Map;for(const M of t.events){if(!M.line)continue;const se=b.get(M.stepId)??[];se.push(M.line),b.set(M.stepId,se)}const N=Ye.map(M=>{var Oe;const se=j.has(M.id),te=l.has(M.id),Pe=te?D("failed","bad"):se?D("done","ok"):D("pending","neutral"),Ue=(b.get(M.id)??[]).slice(-5),Me=(Oe=t.events.find(Ee=>Ee.stepId===M.id&&Ee.err))==null?void 0:Oe.err,Ge=M.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${E?` <a href="${n(E)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${se?"step-done":""} ${te?"step-error":""}">
          <div class="step-head">${Pe} <strong>${n(M.title)}</strong></div>
          ${Ge}
          ${Ue.length?`<pre class="step-log">${Ue.map(Ee=>n(Ee)).join(`
`)}</pre>`:""}
          ${Me?`<p class="error small">${n(Me)}</p>`:""}
        </li>
      `}).join(""),K=t.events.some(M=>M.err),X=Ye.every(M=>j.has(M.id))||t.events.some(M=>M.stepId==="handshake"&&M.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${X&&!K?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        ${K?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function he(m,$){switch(m){case"pick-network":t.chainId=Number($.dataset.chainId),t.execId=null,t.beaconId=null,f();break;case"goto-network":t.step="network",f();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",f();break;case"goto-mode":t.step="mode",f(),V();break;case"goto-review":if(ge(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){f();break}t.step="review",f();break;case"start-setup":H();break}}function ge(){const m=s.querySelectorAll('input[name="mode"]');for(const M of Array.from(m))M.checked&&(t.archive=M.value==="archive");const $=s.querySelector("#data-dir-input"),E=s.querySelector("#jwt-path-input");$&&(t.dataDir=$.value.trim()),E&&(t.jwtPath=E.value.trim());const j=s.querySelector("#exec-http-port-input"),l=s.querySelector("#beacon-http-port-input"),b=s.querySelector("#exec-p2p-port-input");j&&(t.execHTTPPort=j.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),b&&(t.execP2PPort=b.value.trim());const N=s.querySelector("#rpc-bind-addr-input");N&&(t.rpcBindAddr=N.value.trim());const K=s.querySelector("#checkpoint-url-input");K&&(t.checkpointUrl=K.value.trim());const X=s.querySelector("#snapshot-key-input");X&&(t.snapshotKey=X.value.trim()),t.execHTTPPortError=S(t.execHTTPPort).error??null,t.beaconHTTPPortError=S(t.beaconHTTPPort).error??null,t.execP2PPortError=S(t.execP2PPort).error??null,t.rpcBindAddrError=v(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?ne(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function v(m){if(!m)return{};const $=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(m);return $?$.slice(1).every(E=>Number(E)<=255)?{addr:m}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(m)&&m.includes(":")?{addr:m}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const d=/^\d+$/;function S(m){if(!m)return{};if(!d.test(m))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const $=Number(m);return!Number.isInteger($)||$<1||$>65535?{error:"Port must be between 1 and 65535."}:{port:$}}function R(m,$){const{port:E}=S(m);if(!(E===void 0||E===$))return E}async function H(){var b;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(b=t.streamStop)==null||b.call(t),t.streamStop=null,f();const m={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(m.DataDir=t.dataDir),t.jwtPath&&(m.JWTPath=t.jwtPath);const $=R(t.execHTTPPort,Fe),E=R(t.beaconHTTPPort,je),j=R(t.execP2PPort,We);$!==void 0&&(m.ExecHTTPPort=$),E!==void 0&&(m.BeaconHTTPPort=E),j!==void 0&&(m.ExecP2PPort=j);const{addr:l}=v(t.rpcBindAddr);l!==void 0&&(m.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(m.CheckpointURL=t.checkpointUrl):m.NoCheckpoint=!0,t.execSnapshot&&(m.ExecSnapshot=!0,m.SnapshotKey=t.snapshotKey);try{await ln(t.targetId,m)}catch(N){if(!(N instanceof Se&&N.status===409)){t.starting=!1,t.startError=String(N instanceof Error?N.message:N),f();return}}t.starting=!1,t.step="run",f(),t.streamStop=Qe(t.targetId,N=>{r||(t.events.push(N),t.step==="run"&&f())})}function _(m){const $=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],j=$.map(l=>l.id).indexOf(m);return`
      <ol class="wizard-progress">
        ${$.map((l,b)=>`<li class="${b===j?"current":b<j?"past":"future"}">${n(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var m;r=!0,(m=t.streamStop)==null||m.call(t)}}function ra(s,i){let r=!1;const t=new Map;s.innerHTML=`<h1>${n(i)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const h=s.querySelector("#machine-body");ye(s,(I,U)=>{I==="toggle-section"&&q(U.dataset.section??"")}),g();async function g(){let I,U;try{const[O,A]=await Promise.all([ke(),we()]);I=O.find(x=>x.id===i),U=A}catch(O){if(r)return;h.innerHTML=`<p class="error">Failed to load machine: ${n(String(O))}</p>`;return}if(!r){if(!I){location.hash="#/targets";return}P(I,U)}}function P(I,U){const O=I.mode==="local"?"this machine":"SSH",A=I.mode==="ssh"&&I.ssh?`${n(I.ssh.User)}@${n(I.ssh.Host)}`:O;h.innerHTML=`
      <p class="muted">${A}</p>
      <p>${f(I,U)}</p>
      <div class="machine-sections">
        ${F.map(x=>B(x,I,U)).join("")}
      </div>
      ${ce()}
    `}function f(I,U){const O=I.wire;if(!O)return D("not set up","neutral");const A=U.networks.find(u=>u.ChainID===O.ChainID),x=A?A.Name:`chain ${O.ChainID}`;return`${D(x,"ok")} ${D(O.ExecID,"neutral")} ${D(O.BeaconID,"neutral")}${O.Archive?" "+D("archive","warn"):""}`}function B(I,U,O){return`
      <section class="card machine-section" data-section-card="${n(I.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${n(I.key)}" aria-expanded="false">
          <span class="machine-section-title">${n(I.title)}</span>
          <span class="machine-section-status">${I.status(U,O)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${n(I.key)}" hidden></div>
      </section>
    `}function q(I){const U=F.find(y=>y.key===I);if(!U)return;const O=s.querySelector(`[data-section-card="${I}"]`),A=s.querySelector(`[data-section-body="${I}"]`),x=s.querySelector(`.machine-section-head[data-section="${I}"]`);if(!O||!A||!x)return;const u=A.hidden;if(u&&!t.has(I)){const y=document.createElement("div");A.appendChild(y),t.set(I,U.mount(y))}A.hidden=!u,O.classList.toggle("open",u),x.setAttribute("aria-expanded",String(u))}const F=[{key:"setup",title:"Setup",status:I=>I.wire?D("set up","ok"):D("not set up","neutral"),mount:I=>oa(I,i)},{key:"dashboard",title:"Dashboard",status:I=>I.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>Xn(I,i)},{key:"logs",title:"Logs",status:I=>I.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:I=>Qn(I,i)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:I=>aa(I,i)}];return()=>{r=!0;for(const I of t.values())try{I()}catch{}t.clear()}}function ia(s,i){let r=!1,t=[],h=null,g=!1,P=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ce()}</div>`;const f=s.querySelector("#sec-body"),B=s.querySelector("#sec-footer");ye(s,(A,x)=>{var u;if(A==="rerun")F();else if(A==="toggle")(u=x.closest(".check-item"))==null||u.classList.toggle("expanded");else if(A==="copy"){const y=x.dataset.copy;y&&O(x,y)}}),q();async function q(){let A,x;try{const[y,C]=await Promise.all([ke(),we()]);A=y.find(L=>L.id===i),x=C}catch(y){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${n(String(y))}</p>`;return}if(r)return;if(!A){f.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!A.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=x==null?void 0:x.networks.find(y=>y.ChainID===A.wire.ChainID);u&&(B.innerHTML=ce(u.Name,u.LearnURL)),await F()}async function F(){g=!0,h=null,I();try{t=await gn(i),P=!0}catch(A){h=String(A instanceof Error?A.message:A)}g=!1,r||I()}function I(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${g?"disabled":""}>${g?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${!P&&g?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(U).join("")}</ul>`:P?'<p class="muted">No checks returned.</p>':""}
    `}function U(A){const x=A.Status==="pass"?"ok":A.Status==="fail"?"bad":A.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${D(A.Status,x)}
          <strong>${n(A.Title)}</strong>
          <span class="muted small check-detail-inline">${n(A.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${n(A.Why)}</p>
          </details>
          ${A.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(A.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(A.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function O(A,x){const u=await De(x),y=A.textContent;A.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{r||(A.textContent=y)},1500)}return()=>{r=!0}}const ca=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],Xe="VALVE_API_KEY";function la(s){return s===Xe?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${n(s)}}</code> slot wherever an endpoint URL carries one.`}function da(s){let i=!1,r=!1,t=!1,h=null,g=!1,P=null,f=null;const B=new Set,q=new Map;let F="",I="";s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ce()}`;const U=s.querySelector("#settings-body");ye(s,(C,L)=>{if(C==="save"&&y(),C==="clear-key"){if(!P)return;r=!0;const G=s.querySelector("#ai-key");G&&(G.value=""),u(P)}if(C==="clear-provider-key"){const G=L.dataset.key;if(!P||!G)return;B.add(G),q.set(G,""),g=!1,u(P)}}),et(s,(C,L)=>{C!=="ai-provider"||!P||(f=L,g=!1,u(P))}),O();async function O(){try{const C=await On();if(i)return;P=C,u(C)}catch(C){if(i)return;U.innerHTML=`<p class="error">Failed to load settings: ${n(String(C))}</p>`}}function A(C){const G=(Array.isArray(C.providerKeysSet)?C.providerKeysSet:[]).filter(Q=>Q!==Xe).sort();return[Xe,...G]}function x(C,L){const G=n(C);return`
      <div class="pk-row">
        <label>
          <code>${G}</code>
          <input class="provider-key" data-key="${G}" type="password" autocomplete="off"
                 placeholder="${L?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${la(C)}</p>
        ${L?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${G}">Clear saved key</button>`:""}
      </div>`}function u(C){var le;const L=f??C.aiProvider,G=Array.isArray(C.providerKeysSet)?C.providerKeysSet:[],Q=A(C).map(W=>x(W,G.includes(W))).join("");U.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ze("ai-provider",ca.map(W=>({value:W.value,label:W.label})),L)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${C.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${C.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${Q}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${n(F)}" />
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
            <input id="ref-rpc-base" type="text" value="${n(C.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${h?`<p class="error">${n(h)}</p>`:""}
        ${g?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const Y=s.querySelector("#ai-key");Y==null||Y.addEventListener("input",()=>{r=!0,g=!1}),(le=s.querySelector("#ref-rpc-base"))==null||le.addEventListener("input",()=>{g=!1}),s.querySelectorAll("input.provider-key").forEach(W=>{const ae=W.dataset.key;if(!ae)return;const oe=q.get(ae);oe!==void 0&&(W.value=oe),W.addEventListener("input",()=>{B.add(ae),q.set(ae,W.value),g=!1})});const V=s.querySelector("#pk-new-value");V&&(V.value=I),V==null||V.addEventListener("input",()=>{I=V.value,g=!1});const ne=s.querySelector("#pk-new-name");ne==null||ne.addEventListener("input",()=>{F=ne.value,g=!1})}async function y(){const C=s.querySelector("#ai-key"),L=s.querySelector("#ref-rpc-base");if(!C||!L||!P)return;const G={aiProvider:f??P.aiProvider,refRpcBase:L.value.trim()};r&&(G.aiKey=C.value);const Q={};for(const V of B)Q[V]=q.get(V)??"";const Y=F.trim();Y&&(Q[Y]=I),Object.keys(Q).length>0&&(G.providerKeys=Q),t=!0,h=null,g=!1,u(P);try{const V=await qn(G);if(i)return;P=V,r=!1,B.clear(),q.clear(),F="",I="",t=!1,g=!0,u(V)}catch(V){if(i)return;t=!1,h=String(V instanceof Error?V.message:V),u(P)}}return()=>{i=!0}}const ua=["http","ws","archive","trace"],pa={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Re=1337,ha="run",fa={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function ma(s){let i=!1,r=null,t=null;const h={},g={},P={},f={},B={},q={},F={},I={},U={},O={},A={},x={},u={};let y=null;s.innerHTML=`
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
    ${ce()}
  `;const C=s.querySelector("#rpc-body");ye(s,(e,a)=>{At(e,a)}),et(s,()=>{}),L();async function L(){try{const e=await bt();if(i)return;r=e,t=null}catch(e){if(i)return;r=null,t=pe(e)}W();for(const e of(r==null?void 0:r.gateways)??[])G(e.id),Q(e.id,!1)}async function G(e){try{const a=await In(e);if(i)return;h[e]=a}catch{if(i)return;h[e]=null}W()}async function Q(e,a){P[e]=a,a&&W();try{const o=await Ln(e,a);if(i)return;g[e]=o}catch{if(i)return;g[e]=null}P[e]=!1,W()}function Y(e){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===e)}function V(e,a){return(e.networks??[]).find(o=>o.chainId===a)}function ne(e,a,o){var p;const c=(((p=h[e])==null?void 0:p.networks)??[]).find(k=>k.chainId===a);return((c==null?void 0:c.upstreams)??[]).find(k=>k.upstream===o)}function le(e,a,o){var c;return(((c=g[e])==null?void 0:c.endpoints)??[]).find(p=>p.chainId===a&&p.upstream===o)}function W(){if(i)return;if(t){C.innerHTML=`<p class="error">Could not read the gateways: ${n(t)}</p>`;return}if(!r){C.innerHTML='<p class="muted">Loading…</p>';return}const e=r.gateways??[],a=e.length>1,o=(r.targets??[]).some(k=>it(k.id,e)),c=new Set(e.map(k=>k.placement.targetId)),p=(r.orphans??[]).filter(k=>!c.has(k.targetId));C.innerHTML=`
      ${e.map(k=>de(k,a)).join("")}
      ${e.length===0?oe():""}
      ${p.map(ae).join("")}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function ae(e){const a=`docker rm -f ${e.containerName}`,o=u[e.containerName];return`
      <div class="strip">
        ${H({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:a})}
        ${o?H({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function oe(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function de(e,a){return`
      ${a?`<h2 class="rpc-machine">${n(e.placement.targetId)}</h2>`:""}
      ${be(e)}
      ${R(e)}
      ${j(e)}
      ${l(e)}
      ${ge(e)}
    `}function be(e){const a=e.status.State==="running",o=e.tls,c=[`on <strong>${n(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${n(e.status.Image)}</code>`),c.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${$(e)}
        <strong>${n(e.label)}</strong>
        ${m(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${a?`base <code>${n(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function he(e){const a=e.tls;return a!=null&&a.enabled&&a.rootCaPath&&a.effectiveCertSource==="internal"?a.rootCaPath:null}function ge(e){const a=U[e.id]??!1,o=((r==null?void 0:r.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${a?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${n(e.id)}" aria-expanded="${a}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${v(e,o.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${a?d(e,o):""}
      </section>
    `}function v(e,a){const o=[];return e.status.State!=="running"&&o.push("gateway not running"),a>0&&o.push(`${a} leftover container${a===1?"":"s"}`),o.length===0?"container, settings, certificate":o.join(" · ")}function d(e,a){var o;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>E(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${P[e.id]?"disabled":""}>
            ${P[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(e.id)}">
            ${F[e.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(e.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${e.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${n(e.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((o=(e.networks??[])[0])==null?void 0:o.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${n(e.baseUrl)}</code> once it is running.</p>`}
        ${S(e)}
        ${a.map(ae).join("")}
        ${F[e.id]?Ct(e):""}
      </div>
    `}function S(e){const a=he(e);return a?`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority. Install this file (on ${n(e.placement.targetId)}) into the trust store of every device that will call it and the browser warning goes away:</span>
          <code class="strip-cmd">${n(a)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a)}">Copy</button>
        </div>
      </div>
    `:""}function R(e){const a=[];e.error&&a.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&a.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])a.push({tone:"warn",text:c});a.push(..._(e));const o=B[e.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(H).join("")}</div>`}function H(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${n(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${n(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.cmd)}">Copy</button>`:""}
      </div>
    `}function _(e){var p,k;const a=e.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((p=a.status)==null?void 0:p.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((k=a.status)==null?void 0:k.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const c=O[e.id]??a.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&o.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&o.push({tone:"warn",text:c.expiryWarning}),o}function m(e){switch(e.status.State){case"running":return D("running","ok");case"created-but-stopped":return D("stopped","warn");case"not-created":return D("not created","neutral");default:return D("unknown","bad")}}function $(e){return e.status.State==="running"?$e("ok"):e.status.State==="unknown"?$e("bad"):$e("neutral")}function E(e,a){const o=fa[a];if(!o)return"";const c=f[e.id];return`
      <button class="${o.className}" data-action="gw-${a}" data-gid="${n(e.id)}"
              title="${n(o.title)}" ${c?"disabled":""}>
        ${c===a?'<span class="spinner" aria-label="working"></span>':n(o.label)}
      </button>
    `}function j(e){const a=q[e.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(e.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function l(e){const a=b(e.networks??[]),o=a.some(c=>c.chainId===Re);return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(e.id)}">Add a network</button>
            ${se(e,o)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${a.map(c=>N(e,c)).join("")}
        </div>
        ${M(e,o)}
        ${Tt(e)}
      </div>
    `}function b(e){const a=e.filter(c=>c.chainId!==Re),o=e.filter(c=>c.chainId===Re);return[...a,...o]}function N(e,a){const o=Pe(a),c=a.chainId===Re,p=`${e.id}:${a.chainId}`,k=I[p]??!1,T=o.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${o.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${n(a.name)}</span>
          <code class="chain-key">evm:${a.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${D(T,o.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${n(p)}" aria-expanded="${k}">
              ${k?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${K(e,a)}
        ${k?X(e,a,o):""}
      </section>
    `}function K(e,a){if(!a.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const o=he(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${n(a.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                title="Copy ${n(a.url)}">Copy URL</button>
        ${o?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(o)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${n(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>`:""}
      </div>
    `}function X(e,a,o){const c=a.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${o.why?" chain-verdict-why":""}"${o.why?` title="${n(o.why)}"`:""}>${o.html}</p>
        <div class="chain-detail-bar">
          ${te(c.length,o.tone,a.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${n(e.id)}" data-chain="${a.chainId}">Remove</button>
        </div>
        ${Ge(e,a)}
        ${(a.warnings??[]).map(p=>`<p class="chain-note">${n(p)}</p>`).join("")}
      </div>
    `}function M(e,a){const o=g[e.id],c=o!=null&&o.at?`probed ${n(nt(o.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${n(e.id)}">+ Network</button>
        ${se(e,a)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${n(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${P[e.id]?"disabled":""}>
          ${P[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function se(e,a){return a?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${n(e.id)}"
                    title="Add a throwaway local test chain (evm:${Re}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function te(e,a,o){const c=o>0,p=c?o:e,k=Math.min(e,p);let T="";for(let Ie=0;Ie<p;Ie++)T+=`<span class="seg${Ie<k?` seg-on seg-${a}`:""}"></span>`;const w=c&&e>o,z=c?w?`${e} (set is ${o})`:`${e} of ${o}`:`${e}`,ee=`${e} upstream${e===1?"":"s"} configured`,ue=c?`${ee}${w?`, ${e-o} beyond the set`:""}. valve's set for this chain is ${o}.`:`${ee}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${n(ue)}">${T}</span>
      <span class="segs-n">${z}</span>
    `}function Pe(e){const a=e.upstreams??[];if(a.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!a.some(Ue)){const c=Me(a);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(k=>`<code>${n(k)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(a.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!a.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const o=a.filter(c=>!!c.problem);if(o.length>0){const c=a.length-o.length;return{tone:"warn",html:`${o.length} of these ${a.length} endpoints ${o.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function Ue(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Me(e){const a=new Set;for(const o of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((o.endpoint??"").trim());c&&a.add(c[1].toLowerCase())}return[...a].sort()}function Ge(e,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(c=>Oe(e,a,c)).join("")}</ul>`}function Oe(e,a,o){const c=`${e.id}|${a.chainId}|${o.id}`,p=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?$e("bad"):$e("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${Ee(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${$t(e,a,o)}</div>
        <div class="up-share">${St(e,a,o)}</div>
        <div class="up-acts">
          ${p.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(c)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[e.id]?"disabled":""}>
                   ${f[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(c)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function Ee(e){return e.problem?D("unusable","bad"):e.recentOnly?D("recent blocks","warn"):e.local?D("yours","ok"):D("public","neutral")}function tt(e,a){var o;if(e)return a==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(c=>c.key===a))==null?void 0:o.status}function $t(e,a,o){const c=le(e.id,a.chainId,o.id);return c?c.unprobeable?`<span class="caps-none" title="${n(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${ua.map(p=>wt(e,a,c,p)).join("")}</span>`:`<span class="muted small">${g[e.id]===void 0?"probing…":"—"}</span>`}function wt(e,a,o,c){const p=(o.capabilities??[]).find(ee=>ee.key===c),k=tt(o,c)??"inconclusive",T=pa[c]??c.toUpperCase();let w="cap";k==="unsupported"?w=kt(e,a,c)?"cap missing":"cap off":k==="inconclusive"?w="cap unknown":k==="inconsistent"&&(w="cap mixed");const z=p!=null&&p.detail?`${p.label}: ${p.detail}`:c==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${T}: no verdict`;return`<span class="${w}" title="${n(z)}">${n(T)}</span>`}function kt(e,a,o){const c=(a.upstreams??[]).map(p=>le(e.id,a.chainId,p.id)).filter(p=>!!p&&!p.unprobeable);return c.length>0&&c.every(p=>tt(p,o)==="unsupported")}function St(e,a,o){const c=h[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const p=ne(e.id,a.chainId,o.id),k=(c.networks??[]).find(ue=>ue.chainId===a.chainId);if(!p||!k||k.attributed===0)return'<span class="muted small">no traffic yet</span>';const T=Math.round(p.actual*100),w=Math.round(p.intended*100),z=p.diverged?o.local?"warn":"":"ok",ee=`${p.succeeded.toLocaleString()} of ${k.attributed.toLocaleString()} answered requests · routing intends ${w}%`+(p.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ee)}">
        <span class="bar">
          <span class="fill${z?" "+z:""}" style="width:${T}%"></span>
          <span class="tick" style="left:${w}%"></span>
        </span>
        <span class="share-n${p.diverged?" warn":""}">${T}%</span>
        ${p.unconfigured?D("not in config","warn"):""}
      </span>
    `}function Tt(e){const a=h[e.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(nt(a.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function nt(e){const a=new Date(e);return Number.isNaN(a.getTime())?e:a.toLocaleString()}function Ct(e){const a=e.config;return`
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
        ${xt(e)}
        ${Pt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(e.id)}">Save settings</button>
        </div>
      </div>
    `}function xt(e){const a=!e.config.MetricsOff;return`
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
    `}function Pt(e){var T;const a=n(e.id),o=e.config.TLS??null,c=(o==null?void 0:o.Enabled)??!1,p=(o==null?void 0:o.CertSource)||"internal",k=((T=e.tls)==null?void 0:T.suggestedHostname)??"";return`
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
      ${Et(e)}
    `}function Et(e){var T,w;const a=n(e.id),o=((T=e.config.TLS)==null?void 0:T.Enabled)??!1,c=O[e.id]??((w=e.tls)==null?void 0:w.verification)??null,p=A[e.id]??!1,k=x[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!p?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${p?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${k?`<p class="error small">${n(k)}</p>`:""}
      ${c?It(c):""}
    `}function It(e){const a=(e.assertions??[]).map(o=>`
          <li class="small">
            ${Rt(o.status)}
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
    `}function Rt(e){switch(e){case"pass":return D("pass","ok");case"fail":return D("fail","bad");case"unavailable":return D("unavailable","warn");default:return D("skipped","neutral")}}async function Lt(e){A[e]=!0,x[e]=null,W();try{O[e]=await En(e)}catch(a){x[e]=`${pe(a)}${Le(a)}`}finally{A[e]=!1,W()}}function Te(e){return{...e.config,Networks:(e.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function Ce(e,a,o){B[e]=null;try{await Nn(e,a)}catch(c){return B[e]=`${o?o+": ":""}${pe(c)}`,W(),!1}return await L(),!0}async function At(e,a){const o=a.dataset.gid??"";switch(e){case"refresh":await L();return;case"copy":a.dataset.copy&&await nn(a,a.dataset.copy);return;case"reprobe":await Q(o,!0);return;case"toggle-settings":F[o]=!F[o],W();return;case"toggle-manage":U[o]=!U[o],W();return;case"toggle-chain-detail":{const c=a.dataset.key??"";c&&(I[c]=!I[c]),W();return}case"save-settings":await Nt(o);return;case"verify-tls":await Lt(o);return;case"gw-start":case"gw-stop":case"gw-restart":await Dt(o,e.slice(3));return;case"gw-create":case"gw-recreate":await Ut(o);return;case"gw-wipe":Xt(o);return;case"add-gateway":en();return;case"forget-gateway":await Mt(o);return;case"dismiss-orphan":await Ot(a.dataset.name??"");return;case"add-chain":qt(o);return;case"add-devnet":{const c=Y(o);if(c){const p=((r==null?void 0:r.targets)??[]).some(k=>k.id===c.placement.targetId&&k.hasDevnet);st(o,Re,p)}return}case"remove-chain":await Wt(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":rt(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await _t(a.dataset.key??"");return;case"reset-devnet":await Yt(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function Nt(e){const a=Y(e);if(!a)return;const o=Te(a),c=s.querySelector(`#gw-${CSS.escape(e)}-port`),p=s.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const w=Number.parseInt(c.value.trim(),10);Number.isFinite(w)&&(o.Port=w)}p&&(o.BindAddr=p.value.trim());const k=s.querySelector(`#gw-${CSS.escape(e)}-metrics`);k&&(o.MetricsOff=!k.checked),o.TLS=Bt(e,a);const T=a.status.State==="running";await Ce(e,o,"Saving settings")&&(F[e]=!1,T&&(B[e]=null,Ht(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),W())}function Bt(e,a){var k,T,w,z,ee,ue,Ie;const o=an=>s.querySelector(`#gw-${CSS.escape(e)}-${an}`),c=o("tls");if(!c)return a.config.TLS??null;const p=Number.parseInt(((k=o("tls-port"))==null?void 0:k.value.trim())??"",10);return{Enabled:c.checked,Hostname:((T=o("tls-host"))==null?void 0:T.value.trim())??"",CertSource:((w=o("tls-source"))==null?void 0:w.value)??"internal",CertFile:((z=o("tls-cert"))==null?void 0:z.value.trim())??"",KeyFile:((ee=o("tls-key"))==null?void 0:ee.value.trim())??"",HTTPSPort:Number.isFinite(p)?p:443,BindAddr:((ue=a.config.TLS)==null?void 0:ue.BindAddr)??"",ImageRef:((Ie=a.config.TLS)==null?void 0:Ie.ImageRef)??""}}function Ht(e,a){q[e]=[a]}async function Dt(e,a){if(!f[e]){f[e]=a,B[e]=null,W();try{await Bn(e,a)}catch(o){B[e]=`${a} failed: ${pe(o)}${Le(o)}`}f[e]=null,await L()}}async function Ut(e){if(f[e])return;f[e]="create",B[e]=null,q[e]=["starting…"],W();let a;try{a=await Hn(e)}catch(o){B[e]=`${pe(o)}${Le(o)}`,q[e]=[],f[e]=null,W();return}y==null||y(),y=Qe(a.targetId,o=>{if(i)return;const c=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(q[e]=[...(q[e]??[]).filter(k=>k!=="starting…"),c],!!o.err||o.stepId===ha&&!!o.done){y==null||y(),y=null,f[e]=null,o.err&&(B[e]="Provisioning failed — see the log below."),L();return}W()})}async function Mt(e){const a=Y(e);if(!(!a||!await Be({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await An(e)}catch(c){B[e]=pe(c),W();return}await L()}}async function Ot(e){if(e){u[e]=null;try{await xn(e)}catch(a){u[e]=pe(a),W();return}await L()}}function qt(e){const a=Y(e);if(!a)return;const o=new Set((a.networks??[]).map(w=>w.chainId)),c=(r==null?void 0:r.presets)??[],p=c.filter(w=>!o.has(w.chainId)),k=c.filter(w=>o.has(w.chainId)),T=((r==null?void 0:r.targets)??[]).some(w=>w.id===a.placement.targetId&&w.hasDevnet);re(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${p.map(w=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${w.chainId}">
                <span>${n(w.name)}</span>
                <span class="muted small">chain ${w.chainId}${w.devnet?T?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
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
      `,w=>{if(w==="cancel"){Z();return}if(w==="custom"){Ft(e);return}if(w.startsWith("preset:")){const z=Number.parseInt(w.slice(7),10),ee=c.find(ue=>ue.chainId===z);Z(),ee!=null&&ee.devnet?st(e,z,T):at(e,z)}})}function Ft(e){var a;re(`
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
      `,o=>{if(o==="cancel"){Z();return}if(o!=="add")return;const c=document.getElementById("custom-chain-id"),p=document.getElementById("custom-chain-err"),k=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(k)||k<=0){p&&(p.className="error small"),p&&(p.textContent="A chain id is a positive whole number.");return}Z(),at(e,k)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function at(e,a){const o=Y(e);if(!o)return;const c=Te(o),p=c.Networks??[];p.some(k=>k.ChainID===a)||(p.push({ChainID:a,Upstreams:[]}),c.Networks=p,await jt(e,c)&&(W(),rt(e,a)))}async function jt(e,a){var k;const o={...a,Networks:(a.Networks??[]).filter(T=>T.Upstreams.length>0)};if(!await Ce(e,o))return!1;const p=Y(e);if(p)for(const T of a.Networks??[])T.Upstreams.length===0&&!(p.networks??[]).some(w=>w.chainId===T.ChainID)&&(p.config.Networks=[...p.config.Networks??[],{ChainID:T.ChainID,Upstreams:[]}],p.networks=[...p.networks??[],{chainId:T.ChainID,name:((k=((r==null?void 0:r.presets)??[]).find(w=>w.chainId===T.ChainID))==null?void 0:k.name)??`Chain ${T.ChainID}`,path:`/${p.config.ProjectID}/evm/${T.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function st(e,a,o){const c=Y(e);if(!c)return;if(!o){re(`
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
        `,()=>Z());return}const p=Te(c),k=p.Networks??[],T={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},w=k.find(z=>z.ChainID===a);w?w.Upstreams.push(T):k.push({ChainID:a,Upstreams:[T]}),p.Networks=k,await Ce(e,p,"Adding the devnet")}async function Wt(e,a){const o=Y(e);if(!o||!Number.isFinite(a))return;const c=V(o,a);if(!await Be({title:`Remove ${(c==null?void 0:c.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const k=Te(o);k.Networks=(k.Networks??[]).filter(T=>T.ChainID!==a),await Ce(e,k,"Removing the network")}function ot(e){const a=e.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function _t(e){const a=ot(e);if(!a)return;const o=Y(a.gid);if(!o)return;const c=Te(o),p=(c.Networks??[]).find(w=>w.ChainID===a.chainId);if(!p)return;const k=p.Upstreams.findIndex((w,z)=>(w.ID||`${a.chainId}-${z}`)===a.upstreamId);k<0||!await Be({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(p.Upstreams.splice(k,1),await Ce(a.gid,c,"Removing the endpoint"))}function rt(e,a){const o=Y(e);if(!o||!Number.isFinite(a))return;const c=((r==null?void 0:r.sources)??[]).filter(w=>w.chainId===a),p=V(o,a),k=new Set(((p==null?void 0:p.upstreams)??[]).filter(w=>w.kind!=="external").map(w=>`${w.kind}|${w.targetId??""}`)),T=c.filter(w=>!k.has(`${w.kind}|${w.targetId}`));re(`
        <h2>Add an endpoint for ${n((p==null?void 0:p.name)??`chain ${a}`)}</h2>
        ${T.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${T.map(w=>`
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
      `,w=>{if(w==="cancel"){Z();return}if(w==="known-set"){Vt(e,a);return}if(w==="manual"){Jt(e,a);return}if(w.startsWith("source:")){const[,z,ee]=w.split(":");Z(),Kt(e,a,z,ee)}})}async function Kt(e,a,o,c){const p=Y(e);if(!p)return;const k=Te(p),T=k.Networks??[],w={ID:`${o==="managed-devnet"?"devnet":"node"}-${c}`,Kind:o,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},z=T.find(ee=>ee.ChainID===a);z?z.Upstreams.push(w):T.push({ChainID:a,Upstreams:[w]}),k.Networks=T,await Ce(e,k,"Adding the endpoint")}function Gt(e){const a=[...e].sort((p,k)=>(p.latencyMs??1e9)-(k.latencyMs??1e9)),o=a.slice(0,3),c=a.find(p=>p.url.startsWith("wss://")||p.url.startsWith("ws://"));return c&&!o.some(p=>p.url===c.url)&&(o.length===3&&o.pop(),o.push(c)),new Set(o.map(p=>p.url))}async function Vt(e,a){let o;try{o=await Mn(e,a)}catch(w){re(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(pe(w))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Z());return}if(i)return;const c=o.endpoints??[],p=c.filter(w=>!w.alreadyAdded).map(w=>w.url),k=new Set(c.map(w=>w.provider)).size,T=c.map(w=>{const z=[w.websocket?'<span class="t ws">websocket</span>':"",w.archive?'<span class="t ar">archive</span>':"",w.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n(w.url)}</code>
                  <span class="muted small">${n(w.provider)}</span> ${z}</li>`}).join("");re(`<h2>Endpoints for chain ${a}</h2>
       ${c.length?`<p class="muted small">${k} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${T}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${o.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${p.length?"":" disabled"}>
           ${p.length?`Add ${p.length}`:"Nothing to add"}</button>
       </div>`,w=>{Z(),w==="add"&&Ve(e,a,p),w==="discover"&&zt(e,a)})}async function zt(e,a){re(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,T=>{T==="cancel"&&Z()});let o;try{o=await Un(a)}catch(T){const w=He();if(w){const z=document.createElement("p");z.className="error small",z.textContent=`Could not discover endpoints: ${pe(T)}`,w.appendChild(z)}return}if(i)return;const c=(o.endpoints??[]).filter(T=>T.status==="live"||T.status==="unprobed"),p=(o.endpoints??[]).filter(T=>T.status==="rejected"),k=Gt(c);re(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(T=>{const w=k.has(T.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(T.url)}"${w}>
                       <span><code>${n(T.url)}</code></span>
                       <span class="muted small">${T.status==="live"?`answered in ${T.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${p.length?`<details class="rpc-rejected">
                 <summary class="muted small">${p.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${p.map(T=>`<li class="muted small"><code>${n(T.url)}</code> — ${n(T.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,T=>{if(T==="cancel"){Z();return}if(T==="add"){const w=He(),z=w?Array.from(w.querySelectorAll('input[type="checkbox"]:checked')).map(ee=>ee.value):[];Z(),Ve(e,a,z);return}})}function Jt(e,a){var o;re(`
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
      `,c=>{if(c==="cancel"){Z();return}if(c!=="add")return;const p=document.getElementById("manual-endpoint"),k=document.getElementById("manual-recent"),T=document.getElementById("manual-err"),w=(p==null?void 0:p.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(w)){T&&(T.className="error small",T.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Z(),Ve(e,a,[w],(k==null?void 0:k.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function Ve(e,a,o,c=!1){if(!o.length)return;const p=Y(e);if(!p)return;const k=Te(p),T=k.Networks??[];let w=T.find(ee=>ee.ChainID===a);w||(w={ChainID:a,Upstreams:[]},T.push(w));let z=1;for(const ee of w.Upstreams){const ue=/^public-\d+-(\d+)$/.exec(ee.ID??"");ue&&(z=Math.max(z,Number(ue[1])+1))}for(const ee of o)w.Upstreams.some(ue=>ue.Endpoint===ee)||w.Upstreams.push({ID:`public-${a}-${z++}`,Kind:"external",Endpoint:ee,Local:!1,RecentOnly:c});k.Networks=T,await Ce(e,k,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function Yt(e,a){const o=ot(e);if(!o||!a||!await Be({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[o.gid]="reset",B[o.gid]=null,W();let p;try{p=await Tn(a)}catch(k){B[o.gid]=`Reset failed: ${pe(k)}${Le(k)}`,f[o.gid]=null,W();return}f[o.gid]=null,Zt(a,p),await L()}function Zt(e,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const c=a.report.Cascaded??[],p=a.report.CascadeSkipped??[];re(`
        <h2>Devnet on ${n(e)} reset</h2>
        <ul class="plain-list">${o.map(k=>`<li>${n(k)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${n(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${p.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(p.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Z())}function Xt(e){const a=Y(e);if(!a)return;re(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(e)}</button>
        </div>
      `,p=>{if(p==="cancel"||p==="close"){Z(),L();return}p==="confirm"&&Qt(e)});const o=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{c&&(c.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function Qt(e){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await Dn(e)}catch(c){const p=He();if(p){const k=document.createElement("p");k.className="error small",k.textContent=`Wipe failed: ${pe(c)}${Le(c)}`,p.appendChild(k)}a&&(a.disabled=!1,a.textContent=`Wipe ${e}`);return}re(`
        <h2>${n(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Z(),L()})}function it(e,a){return!a.some(o=>{var c;return((c=o.placement)==null?void 0:c.targetId)===e})}function en(){var k;const e=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=e.filter(T=>it(T.id,a)),c=new Set(a.map(T=>T.id));if(e.length===0){re(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Z());return}if(o.length===0){re(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Z());return}const p=c.has("default")?"":"default";re(`
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
            ${o.map(T=>`<option value="${n(T.id)}">${n(T.id)} (${n(T.mode)})</option>`).join("")}
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
      `,T=>{if(T==="cancel"){Z();return}T==="create"&&tn()}),(k=document.getElementById("new-gw-id"))==null||k.focus()}async function tn(){const e=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),p=(e==null?void 0:e.value.trim())??"",k=(a==null?void 0:a.value)??"",T=Number.parseInt((o==null?void 0:o.value.trim())??"",10),w=z=>{c&&(c.className="error small",c.textContent=z)};if(!p){w("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!k){w("Pick the machine it runs on.");return}try{await Pn({id:p,placement:{targetId:k,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(T)?T:4e3,Networks:[]}})}catch(z){w(pe(z));return}Z(),await L()}async function nn(e,a){const o=await De(a),c=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(e.textContent=c)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Le(e){return e instanceof Se&&e.hint?` — ${e.hint}`:""}return()=>{i=!0,y==null||y(),Z()}}const ba="local";function ga(s){let i=!1,r=!1,t="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ce()}
  `;const g=s.querySelector("#targets-body");ye(s,(u,y)=>{F(u,y)}),P();async function P(){try{const[u,y,C]=await Promise.all([ke(),we(),mt()]);if(i)return;t=C.os,B(u,y)}catch(u){if(i)return;g.innerHTML=`<p class="error">Failed to load machines: ${n(String(u))}</p>`}}function f(){h&&B(h.targets,h.catalog)}function B(u,y){h={targets:u,catalog:y};const C=t==="linux",L=[...u].sort((V,ne)=>(V.mode==="local"?-1:0)-(ne.mode==="local"?-1:0)),G=L.length?`<div class="card-grid">${L.map(V=>ya(V,y,V.mode!=="local"||C,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Q=u.some(V=>V.mode==="local");g.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${G}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${q(C,Q)}
        ${r?va():""}
      </section>
    `;const Y=g.querySelector("#fleet-verdict");Y&&yt(Y,gt(u,y))}function q(u,y){const C=`
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
    `,L=u?`
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
      `;return y?`<div class="card-grid card-grid-wide">${C}</div>`:`<div class="card-grid card-grid-wide">${u?L+C:C+L}</div>`}async function F(u,y){var C;if(u==="add-local"){await I();return}if(u==="delete-target"){const L=y.dataset.id;if(!L||!await Be({title:"Remove machine",body:`Remove "${L}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await U(L);return}if(u==="toggle-ssh"){r=!r,x(),f(),r&&((C=s.querySelector("#ssh-host"))==null||C.focus());return}u==="add-ssh"&&await O()}async function I(){x();try{await ct({id:ba,mode:"local"}),await P()}catch(u){A(u)}}async function U(u){try{await rn(u),await P()}catch(y){A(y)}}async function O(){const u=s.querySelector("#ssh-host"),y=s.querySelector("#ssh-user"),C=s.querySelector("#ssh-key"),L=s.querySelector("#ssh-port"),G=s.querySelector("#ssh-id");if(!u||!y||!C||!L||!G)return;const Q=u.value.trim(),Y=y.value.trim(),V=C.value.trim(),ne=L.value.trim(),le=G.value.trim();if(x(),!Q||!Y||!V){A(new Error("host, user, and key path are required"));return}const W=le||$a(Q),ae={Host:Q,User:Y,KeyPath:V};if(ne){const de=Number.parseInt(ne,10);if(!Number.isFinite(de)||de<=0){A(new Error("port must be a positive number"));return}ae.Port=de}const oe=s.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await ct({id:W,mode:"ssh",ssh:ae}),r=!1,await P()}catch(de){A(de),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function A(u){let y=s.querySelector("#targets-error");y||(g.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),y=s.querySelector("#targets-error")),y.textContent=String(u instanceof Error?u.message:u)}function x(){var u;(u=s.querySelector("#targets-error"))==null||u.remove()}return()=>{i=!0}}function ya(s,i,r,t){const h=s.wire,g=s.mode==="local"?"this machine":"SSH",P=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:g;let f;if(!h&&!r)f=`${D("can't run a node","warn")} ${D(t||"not Linux","neutral")}`;else if(!h)f=D("not set up","neutral");else{const B=i.networks.find(F=>F.ChainID===h.ChainID),q=B?B.Name:`chain ${h.ChainID}`;f=`${D(q,"ok")} ${D(h.ExecID,"neutral")} ${D(h.BeaconID,"neutral")}${h.Archive?" "+D("archive","warn"):""}`}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${P}</p>
      <p>${f}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(s.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function va(){return`
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
  `}function $a(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const wa=document.querySelector("#app"),{contentEl:ka,setActiveNav:Sa}=Fn(wa);let fe=null;function Ta(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"home"};const[r,t]=i;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:t?decodeURIComponent(t):void 0}:{screen:r??"targets"}}function ve(s){const i=document.createElement("div");return ka.replaceChildren(i),s(i)}function vt(){if(fe){try{fe()}catch{}fe=null}const{screen:s,id:i}=Ta();switch(Sa(s),s){case"machine":if(!i){location.hash="#/targets";return}fe=ve(r=>ra(r,i));break;case"setup":case"dash":case"logs":case"services":if(!i){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(i)}`;return;case"security":if(!i){location.hash="#/targets";return}fe=ve(r=>ia(r,i));break;case"diag":if(!i){location.hash="#/targets";return}fe=ve(r=>Gn(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}fe=ve(r=>Kn(r,i));break;case"rpc":fe=ve(r=>ma(r));break;case"settings":fe=ve(r=>da(r));break;case"targets":fe=ve(r=>ga(r));break;case"home":default:fe=ve(r=>Yn(r));break}}window.addEventListener("hashchange",vt);vt();
