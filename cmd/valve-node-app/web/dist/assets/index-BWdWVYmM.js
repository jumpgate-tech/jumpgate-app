var jt=Object.defineProperty;var Wt=(s,i,r)=>i in s?jt(s,i,{enumerable:!0,configurable:!0,writable:!0,value:r}):s[i]=r;var Oe=(s,i,r)=>Wt(s,typeof i!="symbol"?i+"":i,r);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const f of document.querySelectorAll('link[rel="modulepreload"]'))t(f);new MutationObserver(f=>{for(const w of f)if(w.type==="childList")for(const R of w.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&t(R)}).observe(document,{childList:!0,subtree:!0});function r(f){const w={};return f.integrity&&(w.integrity=f.integrity),f.referrerPolicy&&(w.referrerPolicy=f.referrerPolicy),f.crossOrigin==="use-credentials"?w.credentials="include":f.crossOrigin==="anonymous"?w.credentials="omit":w.credentials="same-origin",w}function t(f){if(f.ep)return;f.ep=!0;const w=r(f);fetch(f.href,w)}})();function _t(){return K("/api/host")}function Ee(){return K("/api/catalog")}function Ie(){return K("/api/targets")}function st(s){return K("/api/targets",{method:"POST",headers:me,body:JSON.stringify(s)})}function Kt(s){return K(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function zt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function Gt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:me,body:JSON.stringify(i)})}function Ze(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function Jt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function Vt(s,i=200){return K(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function Yt(s,i){const r=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return r.onmessage=t=>{try{i(JSON.parse(t.data))}catch{}},()=>r.close()}function ot(s,i){const r=i===void 0?{}:{lines:i};return K(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:me,body:JSON.stringify(r)})}function Zt(s,i,r){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/${r}`,{method:"POST"})}function Xt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})})}function Qt(s){return K(`/api/targets/${encodeURIComponent(s)}/du`)}function en(s){return K(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function tn(s){return K(`/api/targets/${encodeURIComponent(s)}/firewall`)}function nn(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function an(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function sn(s){return K(`/api/targets/${encodeURIComponent(s)}/containers`)}function on(s,i,r){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${r}`,{method:"POST"})}async function rn(s,i){const r=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})}),t=await r.text();let f=null;try{f=t?JSON.parse(t):null}catch{}if(f&&typeof f=="object"&&"report"in f)return f;const w=f&&typeof f=="object"&&typeof f.error=="string"?f.error:r.statusText||`HTTP ${r.status}`;throw new we(r.status,w)}function cn(s,i){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function ln(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:me}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const f=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,f)}function dn(s,i,r){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:me,body:JSON.stringify(r)})}function ut(){return K("/api/gateways")}async function un(s){await K(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function pn(s){return K("/api/gateways",{method:"POST",headers:me,body:JSON.stringify(s)})}function hn(s){return K(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function fn(s){return K(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function mn(s){return K(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function bn(s,i=!1){const r=i?"?refresh=1":"";return K(`/api/gateways/${encodeURIComponent(s)}/capabilities${r}`)}function gn(s){return K(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function yn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:me,body:JSON.stringify(i)})}function vn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function $n(s){return K(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function wn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:s})}),r=await i.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const f=t&&typeof t=="object"&&typeof t.error=="string"?t.error:i.statusText||`HTTP ${i.status}`;throw new we(i.status,f)}function kn(s){return K(`/api/chainlist/${s}`)}function Tn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/knownset/${i}`)}function Cn(){return K("/api/settings")}function Sn(s){return K("/api/settings",{method:"PUT",headers:me,body:JSON.stringify(s)})}class we extends Error{constructor(r,t,f,w){super(t);Oe(this,"status");Oe(this,"hint");Oe(this,"code");this.name="ApiError",this.status=r,this.hint=f,this.code=w}}const me={"Content-Type":"application/json"};async function K(s,i){const r=await fetch(s,i);if(!r.ok){let f=r.statusText||`HTTP ${r.status}`,w,R;try{const b=await r.json();b&&typeof b.error=="string"&&b.error&&(f=b.error),b&&typeof b.hint=="string"&&b.hint&&(w=b.hint),b&&typeof b.code=="string"&&b.code&&(R=b.code)}catch{}throw new we(r.status,f,w,R)}if(r.status===204)return;const t=await r.text();return t?JSON.parse(t):void 0}const rt="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ie(s,i){const r=s&&i&&i!==rt?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(rt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function xn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),r=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:f=>{for(const w of r)w.classList.toggle("active",w.dataset.nav===f)}}}function ae(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function Pn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function En(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),r=Math.floor(i/60),t=i%60;if(r===0)return`~${t}m`;if(r<48)return`~${r}h ${t}m`;const f=Math.floor(r/24),w=r%24;return`~${f}d ${w}h`}function O(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function $e(s){return`<span class="dot dot-${s}"></span>`}const it=["B","KB","MB","GB","TB","PB"];function Se(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,r=0;for(;i>=1024&&r<it.length-1;)i/=1024,r++;const t=i<10?2:i<100?1:0;return`${i.toFixed(t)} ${it[r]}`}async function He(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ve(s,i){s.addEventListener("click",r=>{const t=r.target.closest("[data-action]");if(!t||!s.contains(t))return;const f=t.dataset.action;f&&i(f,t,r)})}function Ye(s,i,r){const t=i.find(w=>w.value===r),f=i.map(w=>`
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
  `}function Le(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var r;i.classList.remove("open"),(r=i.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function Xe(s,i){s.addEventListener("click",f=>{const w=f.target,R=w.closest(".dropdown-trigger");if(R&&s.contains(R)){const A=R.closest(".dropdown"),j=!!A&&!A.classList.contains("open");Le(s),A&&j&&(A.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const b=w.closest(".dropdown-option");if(b&&s.contains(b)){const A=b.closest(".dropdown");Le(s),i((A==null?void 0:A.dataset.dropdown)??"",b.dataset.value??"");return}Le(s)});const r=f=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}const w=f.target;(!w.closest(".dropdown")||!s.contains(w))&&Le(s)},t=f=>{if(!s.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}f.key==="Escape"&&Le(s)};document.addEventListener("click",r),document.addEventListener("keydown",t)}const _e="app-modal";let We=null;function ne(s,i){V();const r=document.createElement("div");r.className="modal-overlay",r.id=_e,r.innerHTML=`<div class="modal">${s}</div>`,r.addEventListener("click",f=>{const w=f.target.closest("[data-modal-action]");w!=null&&w.dataset.modalAction?i(w.dataset.modalAction):f.target===r&&i("cancel")});const t=f=>{f.key==="Escape"&&i("cancel")};document.addEventListener("keydown",t),We=t,document.body.appendChild(r)}function V(){var s;(s=document.getElementById(_e))==null||s.remove(),We&&(document.removeEventListener("keydown",We),We=null)}function Be(){return document.querySelector(`#${_e} .modal`)}function Ae(s){return new Promise(i=>{var f;let r=!1;const t=w=>{r||(r=!0,V(),i(w))};ne(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,w=>t(w==="confirm")),(f=document.querySelector(`#${_e} [data-modal-action="confirm"]`))==null||f.focus()})}const Ge=5e3,In=60;function Rn(s,i){let r=!1,t=null,f=null,w=null,R=null;const b=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ie()}</div>`;const A=s.querySelector("#an-body");ve(s,(g,l)=>{var k;g==="toggle-endpoint"&&((k=l.closest(".an-endpoint"))==null||k.classList.toggle("expanded"))}),j();async function j(){try{t=((await ut()).gateways??[]).find(l=>l.id===i)??null}catch(g){if(r)return;w=String(g instanceof Error?g.message:g),B();return}if(!r){if(!t){B();return}await U(),R=window.setInterval(()=>void U(),Ge)}}async function U(){try{const g=await mn(i);if(r)return;z(g),f=g,w=null}catch(g){if(r)return;w=String(g instanceof Error?g.message:g)}B()}function z(g){if(!g.enabled||g.error)return;const l=b[b.length-1];l&&l.since!==g.since&&(b.length=0);const k=new Map;for(const P of g.networks??[])k.set(P.chainId,P.received);b.push({t:Date.now(),since:g.since,received:k}),b.length>In&&b.shift()}function B(){r||(A.innerHTML=F())}function F(){return w&&!f?`<h1>Analytics</h1><p class="error">${n(w)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${I(t)}
      ${f?p(f):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(i)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function I(g){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${n(g.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${C()}</div>
      </div>
    `}function C(){if(!f)return"";if(!f.enabled)return"counters off";if(f.error)return"could not be read";const g=f.since?new Date(f.since):null;return g&&!Number.isNaN(g.getTime())?`totals since the gateway started, ${n(g.toLocaleString())}<br />re-read every ${Ge/1e3}s`:`re-read every ${Ge/1e3}s`}function p(g){return g.enabled?g.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(g.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:$(g)+re(g):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function $(g){const l=g.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${l.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':l.map(k=>N(k)).join("")}
      </section>
    `}function N(g){const l=g.methods??[],k=g.endpoints??[],P=g.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${g.chainId}</span>
          <span class="band-name">${n(g.name)}</span>
          ${G(g)}
        </div>
        <div class="an-stats">
          ${H("Received",ae(g.received),"what clients asked this chain for")}
          ${H("Answered",ae(g.answered),"returned by one of your endpoints")}
          ${H("From cache",ae(g.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${H("Failed",ae(g.failed),"asked for and never answered",g.failed>0?"bad":"")}
        </div>
        ${ee(g.chainId)}
        ${P?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':J("Method",l.map(L=>({label:L.method,l:L})))+J("Endpoint",k.map(L=>({label:L.upstream,l:L})))+se(g)}
      </div>
    `}function H(g,l,k,P=""){return`
      <div class="an-stat${P?" an-stat-"+P:""}" title="${n(k)}">
        <span class="an-stat-n">${n(l)}</span>
        <span class="an-stat-l">${n(g)}</span>
      </div>
    `}function G(g){const l=Q(g.chainId);if(l===null)return'<span class="an-rate muted small">measuring rate…</span>';const k=Math.round((b[b.length-1].t-b[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${k}s apart.">
      ${n(l.toFixed(l<10?2:0))} req/s <span class="muted">over the last ${k}s</span>
    </span>`}function Q(g){if(b.length<2)return null;const l=b[0],k=b[b.length-1],P=(k.t-l.t)/1e3;if(P<=0)return null;const L=(k.received.get(g)??0)-(l.received.get(g)??0);return L<0?null:L/P}function ee(g){if(b.length<3)return"";const l=[];for(let y=1;y<b.length;y++){const x=b[y-1],M=b[y],c=(M.t-x.t)/1e3,m=(M.received.get(g)??0)-(x.received.get(g)??0);l.push(c>0&&m>=0?m/c:0)}const k=Math.max(...l);if(k<=0)return"";const P=240,L=28,q=l.length>1?P/(l.length-1):P,h=l.map((y,x)=>`${(x*q).toFixed(1)},${(L-y/k*L).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${k.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${P} ${L}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${h}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(k.toFixed(2))} req/s</span>
      </div>
    `}function se(g){const l=[];return g.cached.count>0&&l.push(`${n(ae(g.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${g.cached.mean===null?"":`, in ${n(Ne(g.cached.mean))} on average`}.`),g.failedLatency.count>0&&g.failedLatency.mean!==null&&l.push(`The ${n(ae(g.failedLatency.count))} that failed took
         ${n(Ne(g.failedLatency.mean))} on average to fail.`),l.length===0?"":`<p class="muted small">${l.join(" ")}</p>`}function J(g,l){return l.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${n(g)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${l.map(k=>de(k.label,k.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function de(g,l){return`
      <tr>
        <td><code>${n(g)}</code></td>
        <td class="an-num">${ae(l.count)}</td>
        <td class="an-num">${l.mean===null?'<span class="muted">—</span>':n(Ne(l.mean))}</td>
        <td>${he(l)}</td>
      </tr>
    `}function he(g){const l=g.buckets??[];if(l.length===0||g.count===0)return'<span class="muted small">—</span>';let k=0;const P=[];for(const q of l){const h=q.count-k;k=q.count,P.push({label:oe(q.le),n:Math.max(0,h)})}return P.reduce((q,h)=>q+h.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(P.filter(q=>q.n>0).map(q=>`${q.n} ${q.label}`).join(" · "))}">
        ${P.map((q,h)=>q.n===0?"":`<span class="an-band an-band-${Math.min(h,4)}" style="flex:${q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ce(P))}</span>
    `}function ce(g){for(let l=g.length-1;l>=0;l--)if(g[l].n>0)return`slowest ${g[l].label}`;return""}function oe(g){if(g==="+Inf")return"30s or more";const l=Number(g);return Number.isFinite(l)?`under ${Ne(l)}`:`under ${g}`}function re(g){const l=g.endpoints??[];return`
      <section class="an-section">
        <h2>What the gateway sees from your endpoints</h2>
        <p class="muted small">
          The gateway's own view, not a client's. Every count here <strong>includes the
          gateway's block-tracking poller</strong>, which calls each endpoint on a timer
          whether or not anyone is using it — on a quiet gateway it is nearly all of this.
          That is why these numbers are much larger than the ones above, and why they are
          not a measure of your traffic.
        </p>
        ${l.length===0?'<div class="card"><p class="muted">The gateway has not talked to any endpoint yet.</p></div>':`<div class="card">
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
                     <tbody>${l.map(k=>ue(k)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function ue(g){const l=g.errors??[],k=l.reduce((L,q)=>L+q.count,0),P=l.length>0;return`
      <tr class="an-endpoint${P?" expandable":""}" ${P?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(g.upstream)}</code>
          ${g.chainId?`<span class="muted small">chain ${g.chainId}</span>`:""}
          ${g.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ae(g.requests)}</td>
        <td class="an-num${k>0?" bad":""}">${k>0?ae(k):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${g.headLag>0?ae(g.headLag):'<span class="muted">0</span>'}</td>
        <td>${be(g)}</td>
      </tr>
      ${P?ge(g,l):""}
    `}function be(g){const l=[];return g.scored?(l.push(g.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(g.position))}</span>`),l.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(g.score.toFixed(3))}</span>`),g.primarySwitches>1&&l.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ae(g.primarySwitches)} switches</span>`),g.excludedSeconds>0&&l.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Ne(g.excludedSeconds))}</span>`),`<span class="an-selection">${l.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(g,l){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${l.map(k=>`
                    <tr>
                      <td class="an-num">${ae(k.count)}</td>
                      <td><code>${n(k.class)}</code></td>
                      <td>${k.severity?`<span class="badge badge-${k.severity==="critical"?"bad":"warn"}">${n(k.severity)}</span>`:""}</td>
                      <td class="muted small">${n(k.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${n(g.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{r=!0,R!==null&&window.clearInterval(R)}}function Ne(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}const Ln=85,Je={exec:"Execution",beacon:"Beacon"};function Nn(s,i){let r=!1,t=null,f=null,w=null,R=null,b=null,A=null,j=null,U=null;const z={exec:null,beacon:null};let B=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ie()}</div>`;const F=s.querySelector("#dash-body"),I=s.querySelector("#dash-footer");F.addEventListener("click",l=>{const k=l.target.closest("[data-action]");if(!k||!F.contains(k))return;const P=k.dataset.action;if(P==="svc-action"){const L=k.dataset.svc,q=k.dataset.kind;L&&q&&ue(L,q)}else if(P==="open-clear"){const L=k.dataset.svc;L&&ge(L)}else if(P==="copy"){const L=k.dataset.copy;L&&be(k,L)}else P==="retry-du"?p():P==="retry-endpoints"&&$()}),C();async function C(){let l,k;try{const[L,q]=await Promise.all([Ie(),Ee()]);l=L.find(h=>h.id===i),k=q}catch(L){if(r)return;F.innerHTML=`<p class="error">Failed to load target: ${n(String(L))}</p>`;return}if(r)return;if(!l){F.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){F.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const P=k==null?void 0:k.networks.find(L=>L.ChainID===l.wire.ChainID);P&&(I.innerHTML=ie(P.Name,P.LearnURL)),F.innerHTML='<p class="muted">Connecting…</p>',t=Jt(i,L=>{r||(N(L),f=L,w=L,H())}),p(),$()}async function p(){A=null;try{b=await Qt(i)}catch(l){b=null,A=String(l instanceof Error?l.message:l)}r||H()}async function $(){U=null;try{j=await en(i)}catch(l){j=null,U=String(l instanceof Error?l.message:l)}r||H()}function N(l){if(!f)return;const k=(new Date(l.at).getTime()-new Date(f.at).getTime())/1e3,P=l.execHead-f.execHead;if(k>0&&P>=0){const L=P/k;R=R===null?L:R*.7+L*.3}}function H(){if(!w)return;const l=w;F.innerHTML=`
      <p class="dash-status">${G(l)}</p>
      <div class="card-grid">
        ${oe(l)}
        ${ee(l)}
        ${se(l)}
        ${J(l)}
        ${de(l)}
        ${he()}
      </div>
      <p class="muted small">Last updated ${n(new Date(l.at).toLocaleTimeString())}</p>
    `}function G(l){return!l.execActive&&!l.beaconActive?O("Node not running","bad"):l.execSyncing||l.beaconDistance>0?O("Syncing","warn"):O("Running · synced","ok")}function Q(l){const P=l.refHead>0?l.refHead-l.execHead:null,L=P!==null&&P>0&&R&&R>0?En(P/R):P!==null&&P<=0?"caught up":"—";return{lag:P,eta:L}}function ee(l){const{lag:k,eta:P}=Q(l);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${l.execActive?l.execSyncing?O("syncing","warn"):l.execHead===0?O("no data","neutral"):O("synced","ok"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ae(l.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${k!==null?ae(l.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${k!==null?ae(Math.max(k,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${P}</dd></div>
        </dl>
      </div>
    `}function se(l){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${l.beaconActive?l.beaconSlot===0?O("no data","neutral"):l.beaconDistance===0?O("synced","ok"):O("syncing","warn"):O("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ae(l.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ae(l.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function J(l){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ae(l.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ae(l.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(l){const k=l.diskUsedPct>=Ln,P=`
      <div class="meter"><div class="meter-fill ${k?"meter-warn":""}" style="width:${Math.min(l.diskUsedPct,100)}%"></div></div>
      <p>${Pn(l.diskUsedPct)} used</p>
    `;if(A)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${P}
          <p class="error small">${n(A)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!b)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${P}
          <p class="muted">Loading…</p>
        </div>
      `;const L=b.ExpectedExecBytes>0?Math.min(b.ExecBytes/b.ExpectedExecBytes*100,100):0,q=b.ExpectedBeaconBytes>0?Math.min(b.BeaconBytes/b.ExpectedBeaconBytes*100,100):0,{lag:h,eta:y}=Q(l),x=h!==null&&h>0&&R!==null&&R>0;return`
      <div class="card ${k?"card-warn":""}">
        <h3>Storage</h3>
        ${P}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Se(b.ExecBytes)} of ~${Se(b.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${L}%"></div></div>
        ${x?`<p class="muted small">Estimated time remaining: ${n(y)}</p>`:""}
        <p class="muted small">Beacon — ${Se(b.BeaconBytes)} of ~${Se(b.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Se(b.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(b.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(b.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function he(){if(U)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(U)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const l=j,k=l.ExecReachable&&!l.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",P=l.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(l.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${$e(l.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(l.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${$e(l.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(l.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.BeaconHTTP)}">Copy</button>
        </div>
        ${k}
        ${P}
      </div>
    `}function ce(l,k){const P=Je[l],L=z[l],q=(h,y,x)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${l}" data-kind="${h}" ${L!==null||x?"disabled":""}>${L===h?re():n(y)}</button>`;return`
      <div class="service-row">
        <span>${n(P)} ${k?O("active","ok"):O("down","bad")}</span>
        <div class="service-actions">
          ${q("start","Start",k)}
          ${q("stop","Stop",!k)}
          ${q("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${l}" ${L!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(l){return`
      <div class="card">
        <h3>Services</h3>
        ${ce("exec",l.execActive)}
        ${ce("beacon",l.beaconActive)}
        ${B?`<p class="error small">${n(B)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function re(){return'<span class="spinner" aria-label="working"></span>'}async function ue(l,k){if(z[l]===null){z[l]=k,B=null,H();try{await Zt(i,l,k)}catch(P){B=`${Je[l]} ${k} failed: ${P instanceof Error?P.message:String(P)}`}z[l]=null,r||H()}}async function be(l,k){const P=await He(k),L=l.textContent;l.textContent=P?"Copied!":"Copy failed",setTimeout(()=>{r||(l.textContent=L)},1500)}function ge(l){const k=Je[l],P=b?Se(l==="exec"?b.ExecBytes:b.BeaconBytes):"unknown (disk usage hasn't loaded)";ne(`
        <h2>Clear ${n(k)} data</h2>
        <p class="error">
          This stops the ${n(k.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(P)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,h=>{if(h==="cancel"){V();return}h==="confirm"&&g(l)});const L=document.getElementById("clear-confirm-input"),q=document.getElementById("clear-confirm-btn");L==null||L.addEventListener("input",()=>{q&&(q.disabled=L.value.trim()!==l)}),L==null||L.focus()}async function g(l){const k=document.getElementById("clear-confirm-btn");k&&(k.disabled=!0,k.textContent="Clearing…");try{await Xt(i,l),V(),p()}catch(P){const L=Be();if(L){const q=document.createElement("p");q.className="error small",q.textContent=`Clear failed: ${P instanceof Error?P.message:String(P)}`,L.appendChild(q)}k&&(k.disabled=!1,k.textContent="Clear and resync")}}return()=>{r=!0,t==null||t(),V()}}const ct=500,lt="valve-node-app.explain-consent";function An(s,i){let r=!1,t=null;const f=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ie()}</div>
  `;const w=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");ve(s,C=>{C==="explain"&&U()}),b();async function b(){let C,p;try{const[N,H]=await Promise.all([Ie(),Ee()]);C=N.find(G=>G.id===i),p=H}catch(N){if(r)return;w.innerHTML=`<p class="error">Failed to load target: ${n(String(N))}</p>`;return}if(r)return;if(!C){w.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const $=p==null?void 0:p.networks.find(N=>N.ChainID===C.wire.ChainID);$&&(R.innerHTML=ie($.Name,$.LearnURL));try{const N=await Vt(i,200);if(r)return;f.push(...N)}catch(N){if(r)return;w.innerHTML=`<p class="error">Failed to load logs: ${n(String(N))}</p>`;return}A(),t=Yt(i,N=>{r||(f.push(N),f.length>ct&&f.splice(0,f.length-ct),A())})}function A(){const C=f.filter($=>$.severity==="error"||$.severity==="critical");w.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${f.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${O(String(C.length),C.length?"bad":"neutral")}</h2>
          <div class="log-lines">${C.length?C.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const p=w.querySelector(".log-lines");p&&(p.scrollTop=p.scrollHeight)}function j(C){const p=C.severity||"info",$=C.learnUrl?` <a href="${n(C.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(p)}">
        <span class="log-time">${n(new Date(C.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(C.unit)}</span>
        <span class="log-sev">${n(p)}</span>
        <span class="log-text">${n(C.line)}</span>
        ${C.explain?`<div class="log-explain">${n(C.explain)}${$}</div>`:""}
      </div>
    `}async function U(){const C=f.filter($=>$.severity==="error"||$.severity==="critical").map($=>$.line).slice(-40);if(!(localStorage.getItem(lt)==="1")){z(C);return}await B(C)}function z(C){const p=C.length?`<pre class="explain-excerpt">${C.map($=>n($)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';F(`
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
    `,$=>{$==="proceed"?(localStorage.setItem(lt,"1"),I(),B(C)):I()})}async function B(C){F('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const p=C.length?await ot(i,C):await ot(i);if(r)return;F(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(p.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${p.sentExcerpt.map($=>n($)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&I()})}catch(p){if(r)return;if(p instanceof we&&p.status===409){F(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,$=>{$==="close"&&I()});return}F(`
        <h2>Explain failed</h2>
        <p class="error">${n(p instanceof Error?p.message:String(p))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&I()})}}function F(C,p){I();const $=document.createElement("div");$.className="modal-overlay",$.id="explain-modal",$.innerHTML=`<div class="modal">${C}</div>`,$.addEventListener("click",N=>{const H=N.target.closest("[data-modal-action]");H!=null&&H.dataset.modalAction&&p(H.dataset.modalAction),N.target===$&&p("cancel")}),document.body.appendChild($)}function I(){var C;(C=document.getElementById("explain-modal"))==null||C.remove()}return()=>{r=!0,t==null||t(),I()}}function Bn(s,i){let r=!1,t=null,f=null,w=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ie()}</div>`;const b=s.querySelector("#diag-body"),A=s.querySelector("#diag-footer");ve(s,(p,$)=>{var N;if(p==="run")U();else if(p==="toggle")(N=$.closest(".check-item"))==null||N.classList.toggle("expanded");else if(p==="copy"){const H=$.dataset.copy;H&&C($,H)}}),j();async function j(){let p,$;try{const[H,G]=await Promise.all([Ie(),Ee()]);p=H.find(Q=>Q.id===i),$=G}catch(H){if(r)return;b.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(r)return;if(!p){b.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!p.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const N=$==null?void 0:$.networks.find(H=>H.ChainID===p.wire.ChainID);N&&(A.innerHTML=ie(N.Name,N.LearnURL));try{t=await an(i),R=!0}catch(H){f=String(H instanceof Error?H.message:H)}r||z()}async function U(){w=!0,f=null,z();try{t=await nn(i),R=!0}catch(p){f=String(p instanceof Error?p.message:p)}w=!1,r||z()}function z(){b.innerHTML=`
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
      ${B()}
    `}function B(){if(!R&&!f)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const p=new Date(t.at).toLocaleString(),$=t.failedId?`<p><strong>Failed at: ${n(F(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(p)} — trigger: ${n(t.trigger)}</p>
      ${$}
      <ul class="check-list">${t.items.map(I).join("")}</ul>
    `}function F(p){var $;return(($=t==null?void 0:t.items.find(N=>N.ID===p))==null?void 0:$.Title)??p}function I(p){const $=p.Status==="pass"?"ok":p.Status==="fail"?"bad":p.Status==="warn"?"warn":"neutral",N=p.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${N?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${O(N?"failed here":p.Status,$)}
          <strong>${n(p.Title)}</strong>
          <span class="muted small check-detail-inline">${n(p.Detail)}</span>
        </button>
        <div class="check-body">
          <details${N?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(p.Why)}</p>
          </details>
          ${p.Fix?`
                <details${N?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(p.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(p.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function C(p,$){const N=await He($),H=p.textContent;p.textContent=N?"Copied!":"Copy failed",setTimeout(()=>{r||(p.textContent=H)},1500)}return()=>{r=!0}}function Hn(s,i){let r=!1,t=[],f=null,w=!1,R=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ie()}</div>`;const b=s.querySelector("#sec-body"),A=s.querySelector("#sec-footer");ve(s,(I,C)=>{var p;if(I==="rerun")U();else if(I==="toggle")(p=C.closest(".check-item"))==null||p.classList.toggle("expanded");else if(I==="copy"){const $=C.dataset.copy;$&&F(C,$)}}),j();async function j(){let I,C;try{const[$,N]=await Promise.all([Ie(),Ee()]);I=$.find(H=>H.id===i),C=N}catch($){if(r)return;b.innerHTML=`<p class="error">Failed to load target: ${n(String($))}</p>`;return}if(r)return;if(!I){b.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!I.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const p=C==null?void 0:C.networks.find($=>$.ChainID===I.wire.ChainID);p&&(A.innerHTML=ie(p.Name,p.LearnURL)),await U()}async function U(){w=!0,f=null,z();try{t=await tn(i),R=!0}catch(I){f=String(I instanceof Error?I.message:I)}w=!1,r||z()}function z(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${w?"disabled":""}>${w?"Re-running…":"Re-run checks"}</button>
      </div>
      ${f?`<p class="error">${n(f)}</p>`:""}
      ${!R&&w?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(B).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function B(I){const C=I.Status==="pass"?"ok":I.Status==="fail"?"bad":I.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${O(I.Status,C)}
          <strong>${n(I.Title)}</strong>
          <span class="muted small check-detail-inline">${n(I.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${n(I.Why)}</p>
          </details>
          ${I.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(I.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(I.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function F(I,C){const p=await He(C),$=I.textContent;I.textContent=p?"Copied!":"Copy failed",setTimeout(()=>{r||(I.textContent=$)},1500)}return()=>{r=!0}}const Dn=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Un(s){let i=!1,r=!1,t=!1,f=null,w=!1,R=null,b=null;s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ie()}`;const A=s.querySelector("#settings-body");ve(s,B=>{if(B==="save"&&z(),B==="clear-key"){if(!R)return;r=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),U(R)}}),Xe(s,(B,F)=>{B!=="ai-provider"||!R||(b=F,w=!1,U(R))}),j();async function j(){try{const B=await Cn();if(i)return;R=B,U(B)}catch(B){if(i)return;A.innerHTML=`<p class="error">Failed to load settings: ${n(String(B))}</p>`}}function U(B){var C;const F=b??B.aiProvider;A.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ye("ai-provider",Dn.map(p=>({value:p.value,label:p.label})),F)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${B.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${B.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${n(B.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${f?`<p class="error">${n(f)}</p>`:""}
        ${w?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const I=s.querySelector("#ai-key");I==null||I.addEventListener("input",()=>{r=!0,w=!1}),(C=s.querySelector("#ref-rpc-base"))==null||C.addEventListener("input",()=>{w=!1})}async function z(){const B=s.querySelector("#ai-key"),F=s.querySelector("#ref-rpc-base");if(!B||!F||!R)return;const I={aiProvider:b??R.aiProvider,refRpcBase:F.value.trim()};r&&(I.aiKey=B.value),t=!0,f=null,w=!1,U(R);try{const C=await Sn(I);if(i)return;R=C,r=!1,t=!1,w=!0,U(C)}catch(C){if(i)return;t=!1,f=String(C instanceof Error?C.message:C),U(R)}}return()=>{i=!0}}const Mn=["http","ws","archive","trace"],On={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Pe=4,Fn="run",qn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function jn(s){let i=!1,r=null,t=null;const f={},w={},R={},b={},A={},j={},U={},z={},B={},F={},I={};let C=null;s.innerHTML=`
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
    ${ie()}
  `;const p=s.querySelector("#rpc-body");ve(s,(e,a)=>{mt(e,a)}),Xe(s,()=>{}),$();async function $(){try{const e=await ut();if(i)return;r=e,t=null}catch(e){if(i)return;r=null,t=pe(e)}J();for(const e of(r==null?void 0:r.gateways)??[])N(e.id),H(e.id,!1)}async function N(e){try{const a=await fn(e);if(i)return;f[e]=a}catch{if(i)return;f[e]=null}J()}async function H(e,a){R[e]=a,a&&J();try{const o=await bn(e,a);if(i)return;w[e]=o}catch{if(i)return;w[e]=null}R[e]=!1,J()}function G(e){return((r==null?void 0:r.gateways)??[]).find(a=>a.id===e)}function Q(e,a){return(e.networks??[]).find(o=>o.chainId===a)}function ee(e,a,o){var u;const d=(((u=f[e])==null?void 0:u.networks)??[]).find(T=>T.chainId===a);return((d==null?void 0:d.upstreams)??[]).find(T=>T.upstream===o)}function se(e,a,o){var d;return(((d=w[e])==null?void 0:d.endpoints)??[]).find(u=>u.chainId===a&&u.upstream===o)}function J(){if(i)return;if(t){p.innerHTML=`<p class="error">Could not read the gateways: ${n(t)}</p>`;return}if(!r){p.innerHTML='<p class="muted">Loading…</p>';return}const e=r.gateways??[],a=e.length>1,o=(r.targets??[]).some(d=>nt(d.id,e));p.innerHTML=`
      ${(r.orphans??[]).map(de).join("")}
      ${e.map(d=>ce(d,a)).join("")}
      ${e.length===0?he():""}
      ${o?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function de(e){const a=`docker rm -f ${e.containerName}`,o=I[e.containerName];return`
      <div class="strip">
        ${ue({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:a})}
        ${o?ue({tone:"bad",text:o}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${n(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function he(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function ce(e,a){return`
      ${a?`<h2 class="rpc-machine">${n(e.placement.targetId)}</h2>`:""}
      ${oe(e)}
      ${re(e)}
      ${k(e)}
      ${U[e.id]?De(e):""}
      ${P(e)}
    `}function oe(e){var u;const a=e.status.State==="running",o=e.tls,d=[`on <strong>${n(e.placement.targetId)}</strong>`];return e.status.Image&&d.push(`<code>${n(e.status.Image)}</code>`),d.push(o!=null&&o.enabled?`HTTPS front <code>${n(o.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-head">
        <div class="rpc-head-id">
          ${g(e)}
          <strong>${n(e.label)}</strong>
          ${ge(e)}
          <span class="muted small">${d.join(" · ")}</span>
        </div>
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(T=>l(e,T)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(e.id)}">
            ${U[e.id]?"Close":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(e.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        <div class="rpc-head-url">
          ${a?`<code class="endpoint-url">${n(e.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((u=(e.networks??[])[0])==null?void 0:u.path)??"/main/evm/<chainId>")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${n(e.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function re(e){const a=[];e.error&&a.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&a.push({tone:"warn",text:e.blocked});for(const d of e.warnings??[])a.push({tone:"warn",text:d});a.push(...be(e));const o=A[e.id];return o&&a.push({tone:"bad",text:o}),a.length===0?"":`<div class="strip">${a.map(ue).join("")}</div>`}function ue(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${n(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${n(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(e.cmd)}">Copy</button>`:""}
      </div>
    `}function be(e){var u,T;const a=e.tls;if(!(a!=null&&a.enabled))return[];const o=[];a.fallback&&o.push({tone:"warn",text:a.fallback}),a.error?o.push({tone:"warn",text:`HTTPS front: ${a.error}`}):((u=a.status)==null?void 0:u.State)!=="running"&&o.push({tone:"warn",text:`The HTTPS front is ${((T=a.status)==null?void 0:T.State)??"unknown"}, so nothing answers on ${a.url??"its https URL"} even if the gateway itself is up.`,cmd:a.containerName?`docker start ${a.containerName}`:void 0});const d=z[e.id]??a.verification??null;return d&&(!d.ok||!d.subscriptionsOk)&&o.push({tone:d.ok?"warn":"bad",text:`${d.summary} Checked ${new Date(d.at).toLocaleString()} — open Settings for the full check.`}),d!=null&&d.expiryWarning&&o.push({tone:"warn",text:d.expiryWarning}),a.rootCaPath&&a.effectiveCertSource==="internal"&&o.push({tone:"note",text:`Served by Caddy's own certificate authority. Install this file (on ${e.placement.targetId}) into the trust store of every device that will call it and the browser warning goes away:`,cmd:a.rootCaPath}),o}function ge(e){switch(e.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function g(e){return e.status.State==="running"?$e("ok"):e.status.State==="unknown"?$e("bad"):$e("neutral")}function l(e,a){const o=qn[a];if(!o)return"";const d=b[e.id];return`
      <button class="${o.className}" data-action="gw-${a}" data-gid="${n(e.id)}"
              title="${n(o.title)}" ${d?"disabled":""}>
        ${d===a?'<span class="spinner" aria-label="working"></span>':n(o.label)}
      </button>
    `}function k(e){const a=j[e.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(e.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function P(e){const a=e.networks??[];return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${n(e.id)}">Add a network</button>
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        ${x(e)}
        <div class="chains">
          ${a.map(o=>L(e,o)).join("")}
        </div>
        ${X(e)}
      </div>
    `}function L(e,a){const o=a.upstreams??[],d=h(a);return`
      <section class="chain chain-${d.tone}">
        <div class="chain-head">
          <span class="chain-name">${n(a.name)}</span>
          <code class="chain-key">evm:${a.chainId}</code>
          <code class="chain-path">${n(a.path)}</code>
          ${a.url?`<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                         title="Copy ${n(a.url)}">Copy URL</button>`:""}
          <span class="chain-right">
            ${q(o.length,d.tone)}
            <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                    data-gid="${n(e.id)}" data-chain="${a.chainId}">+ Endpoint</button>
            <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                    data-gid="${n(e.id)}" data-chain="${a.chainId}">Remove</button>
          </span>
        </div>
        <p class="chain-verdict${d.why?" chain-verdict-why":""}"${d.why?` title="${n(d.why)}"`:""}>${d.html}</p>
        ${M(e,a)}
        ${(a.warnings??[]).map(u=>`<p class="chain-note">${n(u)}</p>`).join("")}
      </section>
    `}function q(e,a){const o=Math.min(e,Pe);let d="";for(let T=0;T<Pe;T++)d+=`<span class="seg${T<o?` seg-on seg-${a}`:""}"></span>`;const u=e>Pe?`, ${e-Pe} beyond the set`:"";return`
      <span class="segs" title="${e} upstream${e===1?"":"s"} configured${u}. valve's set for a chain is ${Pe}.">${d}</span>
      <span class="segs-n">${e} of ${Pe}</span>
    `}function h(e){const a=e.upstreams??[];return a.length===0?{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."}:e.serviceable?a.some(y)?a.length===1?{tone:"warn",html:"One endpoint, so this chain stops when it does."}:a.some(o=>o.local)?{tone:"ok",html:`${a.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}:{tone:"warn",html:"No node of your own serves this chain."}:{tone:"warn",html:"No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain.",why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection."}:{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."}}function y(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function x(e){const a=w[e.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(ke(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[e.id]?"disabled":""}>
          ${R[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(e.id)}">+ Network</button>
      </div>
    `}function M(e,a){const o=a.upstreams??[];return o.length===0?"":`<ul class="ups">${o.map(d=>c(e,a,d)).join("")}</ul>`}function c(e,a,o){const d=`${e.id}|${a.chainId}|${o.id}`,u=o.actions??[];return`
      <li class="up${o.problem?" up-bad":""}">
        <div class="up-what">
          ${o.problem?$e("bad"):$e("ok")}
          <span class="up-label">${n(o.label)}</span>
          ${m(o)}
        </div>
        <code class="up-url">${n(o.endpoint||"—")}</code>
        <div class="up-caps">${W(e,a,o)}</div>
        <div class="up-share">${te(e,a,o)}</div>
        <div class="up-acts">
          ${u.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(d)}"
                         data-target="${n(o.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${b[e.id]?"disabled":""}>
                   ${b[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(d)}">Remove</button>
        </div>
        ${o.problem?`<div class="up-problem error small">${n(o.problem)}</div>`:""}
      </li>
    `}function m(e){return e.problem?O("unusable","bad"):e.recentOnly?O("recent blocks","warn"):e.local?O("yours","ok"):O("public","neutral")}function E(e,a){var o;if(e)return a==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(o=(e.capabilities??[]).find(d=>d.key===a))==null?void 0:o.status}function W(e,a,o){const d=se(e.id,a.chainId,o.id);return d?d.unprobeable?`<span class="caps-none" title="${n(d.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Mn.map(u=>Y(e,a,d,u)).join("")}</span>`:`<span class="muted small">${w[e.id]===void 0?"probing…":"—"}</span>`}function Y(e,a,o,d){const u=(o.capabilities??[]).find(Z=>Z.key===d),T=E(o,d)??"inconclusive",S=On[d]??d.toUpperCase();let v="cap";T==="unsupported"?v=D(e,a,d)?"cap missing":"cap off":T==="inconclusive"?v="cap unknown":T==="inconsistent"&&(v="cap mixed");const _=u!=null&&u.detail?`${u.label}: ${u.detail}`:d==="http"&&o.reachDetail?`Answers JSON-RPC over HTTP: ${o.reachDetail}`:`${S}: no verdict`;return`<span class="${v}" title="${n(_)}">${n(S)}</span>`}function D(e,a,o){const d=(a.upstreams??[]).map(u=>se(e.id,a.chainId,u.id)).filter(u=>!!u&&!u.unprobeable);return d.length>0&&d.every(u=>E(u,o)==="unsupported")}function te(e,a,o){const d=f[e.id];if(d===void 0)return'<span class="muted small">reading…</span>';if(d===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!d.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const u=ee(e.id,a.chainId,o.id),T=(d.networks??[]).find(fe=>fe.chainId===a.chainId);if(!u||!T||T.attributed===0)return'<span class="muted small">no traffic yet</span>';const S=Math.round(u.actual*100),v=Math.round(u.intended*100),_=u.diverged?o.local?"warn":"":"ok",Z=`${u.succeeded.toLocaleString()} of ${T.attributed.toLocaleString()} answered requests · routing intends ${v}%`+(u.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(Z)}">
        <span class="bar">
          <span class="fill${_?" "+_:""}" style="width:${S}%"></span>
          <span class="tick" style="left:${v}%"></span>
        </span>
        <span class="share-n${u.diverged?" warn":""}">${S}%</span>
        ${u.unconfigured?O("not in config","warn"):""}
      </span>
    `}function X(e){const a=f[e.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(ke(a.since))})`:""}. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function ke(e){const a=new Date(e);return Number.isNaN(a.getTime())?e:a.toLocaleString()}function De(e){const a=e.config;return`
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
        ${Ue(e)}
        ${Ke(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(e.id)}">Save settings</button>
        </div>
      </div>
    `}function Ue(e){const a=!e.config.MetricsOff;return`
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
    `}function Ke(e){var S;const a=n(e.id),o=e.config.TLS??null,d=(o==null?void 0:o.Enabled)??!1,u=(o==null?void 0:o.CertSource)||"internal",T=((S=e.tls)==null?void 0:S.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${a}-tls" ${d?"checked":""} />
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
      ${Me(e)}
    `}function Me(e){var S,v;const a=n(e.id),o=((S=e.config.TLS)==null?void 0:S.Enabled)??!1,d=z[e.id]??((v=e.tls)==null?void 0:v.verification)??null,u=B[e.id]??!1,T=F[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${o&&!u?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${u?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${o?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${T?`<p class="error small">${n(T)}</p>`:""}
      ${d?xe(d):""}
    `}function xe(e){const a=(e.assertions??[]).map(o=>`
          <li class="small">
            ${ht(o.status)}
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
    `}function ht(e){switch(e){case"pass":return O("pass","ok");case"fail":return O("fail","bad");case"unavailable":return O("unavailable","warn");default:return O("skipped","neutral")}}async function ft(e){B[e]=!0,F[e]=null,J();try{z[e]=await hn(e)}catch(a){F[e]=`${pe(a)}${Re(a)}`}finally{B[e]=!1,J()}}function Te(e){return{...e.config,Networks:(e.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(o=>({...o}))}))}}async function Ce(e,a,o){A[e]=null;try{await yn(e,a)}catch(d){return A[e]=`${o?o+": ":""}${pe(d)}`,J(),!1}return await $(),!0}async function mt(e,a){const o=a.dataset.gid??"";switch(e){case"refresh":await $();return;case"copy":a.dataset.copy&&await Ft(a,a.dataset.copy);return;case"reprobe":await H(o,!0);return;case"toggle-settings":U[o]=!U[o],J();return;case"save-settings":await bt(o);return;case"verify-tls":await ft(o);return;case"gw-start":case"gw-stop":case"gw-restart":await vt(o,e.slice(3));return;case"gw-create":case"gw-recreate":await $t(o);return;case"gw-wipe":Dt(o);return;case"add-gateway":Mt();return;case"forget-gateway":await wt(o);return;case"dismiss-orphan":await kt(a.dataset.name??"");return;case"add-chain":Tt(o);return;case"remove-chain":await Pt(o,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":tt(o,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await Et(a.dataset.key??"");return;case"reset-devnet":await Bt(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function bt(e){const a=G(e);if(!a)return;const o=Te(a),d=s.querySelector(`#gw-${CSS.escape(e)}-port`),u=s.querySelector(`#gw-${CSS.escape(e)}-bind`);if(d){const v=Number.parseInt(d.value.trim(),10);Number.isFinite(v)&&(o.Port=v)}u&&(o.BindAddr=u.value.trim());const T=s.querySelector(`#gw-${CSS.escape(e)}-metrics`);T&&(o.MetricsOff=!T.checked),o.TLS=gt(e,a);const S=a.status.State==="running";await Ce(e,o,"Saving settings")&&(U[e]=!1,S&&(A[e]=null,yt(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),J())}function gt(e,a){var T,S,v,_,Z,fe,at;const o=qt=>s.querySelector(`#gw-${CSS.escape(e)}-${qt}`),d=o("tls");if(!d)return a.config.TLS??null;const u=Number.parseInt(((T=o("tls-port"))==null?void 0:T.value.trim())??"",10);return{Enabled:d.checked,Hostname:((S=o("tls-host"))==null?void 0:S.value.trim())??"",CertSource:((v=o("tls-source"))==null?void 0:v.value)??"internal",CertFile:((_=o("tls-cert"))==null?void 0:_.value.trim())??"",KeyFile:((Z=o("tls-key"))==null?void 0:Z.value.trim())??"",HTTPSPort:Number.isFinite(u)?u:443,BindAddr:((fe=a.config.TLS)==null?void 0:fe.BindAddr)??"",ImageRef:((at=a.config.TLS)==null?void 0:at.ImageRef)??""}}function yt(e,a){j[e]=[a]}async function vt(e,a){if(!b[e]){b[e]=a,A[e]=null,J();try{await vn(e,a)}catch(o){A[e]=`${a} failed: ${pe(o)}${Re(o)}`}b[e]=null,await $()}}async function $t(e){if(b[e])return;b[e]="create",A[e]=null,j[e]=["starting…"],J();let a;try{a=await $n(e)}catch(o){A[e]=`${pe(o)}${Re(o)}`,j[e]=[],b[e]=null,J();return}C==null||C(),C=Ze(a.targetId,o=>{if(i)return;const d=o.err?`${o.stepId}: ${o.err}`:o.line?`${o.stepId}: ${o.line}`:`${o.stepId}: done`;if(j[e]=[...(j[e]??[]).filter(T=>T!=="starting…"),d],!!o.err||o.stepId===Fn&&!!o.done){C==null||C(),C=null,b[e]=null,o.err&&(A[e]="Provisioning failed — see the log below."),$();return}J()})}async function wt(e){const a=G(e);if(!(!a||!await Ae({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await gn(e)}catch(d){A[e]=pe(d),J();return}await $()}}async function kt(e){if(e){I[e]=null;try{await un(e)}catch(a){I[e]=pe(a),J();return}await $()}}function Tt(e){const a=G(e);if(!a)return;const o=new Set((a.networks??[]).map(v=>v.chainId)),d=(r==null?void 0:r.presets)??[],u=d.filter(v=>!o.has(v.chainId)),T=d.filter(v=>o.has(v.chainId)),S=((r==null?void 0:r.targets)??[]).some(v=>v.id===a.placement.targetId&&v.hasDevnet);ne(`
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
                <span class="muted small">chain ${v.chainId}${v.devnet?S?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${T.length?`<p class="muted small">Already fronted: ${n(T.map(v=>v.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,v=>{if(v==="cancel"){V();return}if(v==="custom"){Ct(e);return}if(v.startsWith("preset:")){const _=Number.parseInt(v.slice(7),10),Z=d.find(fe=>fe.chainId===_);V(),Z!=null&&Z.devnet?xt(e,_,S):Qe(e,_)}})}function Ct(e){var a;ne(`
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
      `,o=>{if(o==="cancel"){V();return}if(o!=="add")return;const d=document.getElementById("custom-chain-id"),u=document.getElementById("custom-chain-err"),T=Number.parseInt((d==null?void 0:d.value.trim())??"",10);if(!Number.isFinite(T)||T<=0){u&&(u.className="error small"),u&&(u.textContent="A chain id is a positive whole number.");return}V(),Qe(e,T)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function Qe(e,a){const o=G(e);if(!o)return;const d=Te(o),u=d.Networks??[];u.some(T=>T.ChainID===a)||(u.push({ChainID:a,Upstreams:[]}),d.Networks=u,await St(e,d)&&(J(),tt(e,a)))}async function St(e,a){var T;const o={...a,Networks:(a.Networks??[]).filter(S=>S.Upstreams.length>0)};if(!await Ce(e,o))return!1;const u=G(e);if(u)for(const S of a.Networks??[])S.Upstreams.length===0&&!(u.networks??[]).some(v=>v.chainId===S.ChainID)&&(u.config.Networks=[...u.config.Networks??[],{ChainID:S.ChainID,Upstreams:[]}],u.networks=[...u.networks??[],{chainId:S.ChainID,name:((T=((r==null?void 0:r.presets)??[]).find(v=>v.chainId===S.ChainID))==null?void 0:T.name)??`Chain ${S.ChainID}`,path:`/${u.config.ProjectID}/evm/${S.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function xt(e,a,o){const d=G(e);if(!d)return;if(!o){ne(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${n(d.placement.targetId)}</code>, so adding chain ${a} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(d.placement.targetId)}" data-modal-action="go">Create a devnet on ${n(d.placement.targetId)}</a>
          </div>
        `,()=>V());return}const u=Te(d),T=u.Networks??[],S={ID:"devnet",Kind:"managed-devnet",TargetID:d.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},v=T.find(_=>_.ChainID===a);v?v.Upstreams.push(S):T.push({ChainID:a,Upstreams:[S]}),u.Networks=T,await Ce(e,u,"Adding the devnet")}async function Pt(e,a){const o=G(e);if(!o||!Number.isFinite(a))return;const d=Q(o,a);if(!await Ae({title:`Remove ${(d==null?void 0:d.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(d==null?void 0:d.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const T=Te(o);T.Networks=(T.Networks??[]).filter(S=>S.ChainID!==a),await Ce(e,T,"Removing the network")}function et(e){const a=e.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function Et(e){const a=et(e);if(!a)return;const o=G(a.gid);if(!o)return;const d=Te(o),u=(d.Networks??[]).find(v=>v.ChainID===a.chainId);if(!u)return;const T=u.Upstreams.findIndex((v,_)=>(v.ID||`${a.chainId}-${_}`)===a.upstreamId);T<0||!await Ae({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(u.Upstreams.splice(T,1),await Ce(a.gid,d,"Removing the endpoint"))}function tt(e,a){const o=G(e);if(!o||!Number.isFinite(a))return;const d=((r==null?void 0:r.sources)??[]).filter(v=>v.chainId===a),u=Q(o,a),T=new Set(((u==null?void 0:u.upstreams)??[]).filter(v=>v.kind!=="external").map(v=>`${v.kind}|${v.targetId??""}`)),S=d.filter(v=>!T.has(`${v.kind}|${v.targetId}`));ne(`
        <h2>Add an endpoint for ${n((u==null?void 0:u.name)??`chain ${a}`)}</h2>
        ${S.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${S.map(v=>`
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
      `,v=>{if(v==="cancel"){V();return}if(v==="known-set"){Lt(e,a);return}if(v==="manual"){At(e,a);return}if(v.startsWith("source:")){const[,_,Z]=v.split(":");V(),It(e,a,_,Z)}})}async function It(e,a,o,d){const u=G(e);if(!u)return;const T=Te(u),S=T.Networks??[],v={ID:`${o==="managed-devnet"?"devnet":"node"}-${d}`,Kind:o,TargetID:d,Endpoint:"",Local:!0,RecentOnly:!1},_=S.find(Z=>Z.ChainID===a);_?_.Upstreams.push(v):S.push({ChainID:a,Upstreams:[v]}),T.Networks=S,await Ce(e,T,"Adding the endpoint")}function Rt(e){const a=[...e].sort((u,T)=>(u.latencyMs??1e9)-(T.latencyMs??1e9)),o=a.slice(0,3),d=a.find(u=>u.url.startsWith("wss://")||u.url.startsWith("ws://"));return d&&!o.some(u=>u.url===d.url)&&(o.length===3&&o.pop(),o.push(d)),new Set(o.map(u=>u.url))}async function Lt(e,a){let o;try{o=await Tn(e,a)}catch(v){ne(`<h2>Endpoints for chain ${a}</h2>
         <p class="error small">Could not read the set: ${n(pe(v))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>V());return}if(i)return;const d=o.endpoints??[],u=d.filter(v=>!v.alreadyAdded).map(v=>v.url),T=new Set(d.map(v=>v.provider)).size,S=d.map(v=>{const _=[v.websocket?'<span class="t ws">websocket</span>':"",v.archive?'<span class="t ar">archive</span>':"",v.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${n(v.url)}</code>
                  <span class="muted small">${n(v.provider)}</span> ${_}</li>`}).join("");ne(`<h2>Endpoints for chain ${a}</h2>
       ${d.length?`<p class="muted small">${T} providers valve has measured, in the order the gateway
                should prefer them.</p>
              <ul class="plain-list">${S}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${o.usingDefaultKey?`<p class="muted small">Using the shared <code>${n(o.key)}</code> key, so this
                works with no setup. A free key of your own removes the shared limit.</p>`:'<p class="muted small">Using your key for this chain.</p>'}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${u.length?"":" disabled"}>
           ${u.length?`Add ${u.length}`:"Nothing to add"}</button>
       </div>`,v=>{V(),v==="add"&&ze(e,a,u),v==="discover"&&Nt(e,a)})}async function Nt(e,a){ne(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,S=>{S==="cancel"&&V()});let o;try{o=await kn(a)}catch(S){const v=Be();if(v){const _=document.createElement("p");_.className="error small",_.textContent=`Could not discover endpoints: ${pe(S)}`,v.appendChild(_)}return}if(i)return;const d=(o.endpoints??[]).filter(S=>S.status==="live"||S.status==="unprobed"),u=(o.endpoints??[]).filter(S=>S.status==="rejected"),T=Rt(d);ne(`
        <h2>Public endpoints for chain ${a}</h2>
        ${o.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${o.fetchError?`<div class="small">${n(o.fetchError)}</div>`:""}</div>`:""}
        ${d.length?`<p class="muted small">${d.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${d.map(S=>{const v=T.has(S.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(S.url)}"${v}>
                       <span><code>${n(S.url)}</code></span>
                       <span class="muted small">${S.status==="live"?`answered in ${S.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${u.length?`<details class="rpc-rejected">
                 <summary class="muted small">${u.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${u.map(S=>`<li class="muted small"><code>${n(S.url)}</code> — ${n(S.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${d.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,S=>{if(S==="cancel"){V();return}if(S==="add"){const v=Be(),_=v?Array.from(v.querySelectorAll('input[type="checkbox"]:checked')).map(Z=>Z.value):[];V(),ze(e,a,_);return}})}function At(e,a){var o;ne(`
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
      `,d=>{if(d==="cancel"){V();return}if(d!=="add")return;const u=document.getElementById("manual-endpoint"),T=document.getElementById("manual-recent"),S=document.getElementById("manual-err"),v=(u==null?void 0:u.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(v)){S&&(S.className="error small",S.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}V(),ze(e,a,[v],(T==null?void 0:T.checked)??!1)}),(o=document.getElementById("manual-endpoint"))==null||o.focus()}async function ze(e,a,o,d=!1){if(!o.length)return;const u=G(e);if(!u)return;const T=Te(u),S=T.Networks??[];let v=S.find(Z=>Z.ChainID===a);v||(v={ChainID:a,Upstreams:[]},S.push(v));let _=1;for(const Z of v.Upstreams){const fe=/^public-\d+-(\d+)$/.exec(Z.ID??"");fe&&(_=Math.max(_,Number(fe[1])+1))}for(const Z of o)v.Upstreams.some(fe=>fe.Endpoint===Z)||v.Upstreams.push({ID:`public-${a}-${_++}`,Kind:"external",Endpoint:Z,Local:!1,RecentOnly:d});T.Networks=S,await Ce(e,T,o.length===1?"Adding the endpoint":`Adding ${o.length} endpoints`)}async function Bt(e,a){const o=et(e);if(!o||!a||!await Ae({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;b[o.gid]="reset",A[o.gid]=null,J();let u;try{u=await ln(a)}catch(T){A[o.gid]=`Reset failed: ${pe(T)}${Re(T)}`,b[o.gid]=null,J();return}b[o.gid]=null,Ht(a,u),await $()}function Ht(e,a){const o=[];o.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&o.push("A fresh chain was started from genesis.");const d=a.report.Cascaded??[],u=a.report.CascadeSkipped??[];ne(`
        <h2>Devnet on ${n(e)} reset</h2>
        <ul class="plain-list">${o.map(T=>`<li>${n(T)}</li>`).join("")}</ul>
        ${d.length?`<p class="ok">Restarted in front of it: ${n(d.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${u.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(u.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>V())}function Dt(e){const a=G(e);if(!a)return;ne(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(e)}</button>
        </div>
      `,u=>{if(u==="cancel"||u==="close"){V(),$();return}u==="confirm"&&Ut(e)});const o=document.getElementById("wipe-confirm-input"),d=document.getElementById("wipe-confirm-btn");o==null||o.addEventListener("input",()=>{d&&(d.disabled=o.value.trim()!==e)}),o==null||o.focus()}async function Ut(e){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let o;try{o=await wn(e)}catch(d){const u=Be();if(u){const T=document.createElement("p");T.className="error small",T.textContent=`Wipe failed: ${pe(d)}${Re(d)}`,u.appendChild(T)}a&&(a.disabled=!1,a.textContent=`Wipe ${e}`);return}ne(`
        <h2>${n(e)} wiped</h2>
        <ul class="plain-list">
          <li>${o.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${o.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${o.error?`<p class="error small">${n(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{V(),$()})}function nt(e,a){return!a.some(o=>{var d;return((d=o.placement)==null?void 0:d.targetId)===e})}function Mt(){var T;const e=(r==null?void 0:r.targets)??[],a=(r==null?void 0:r.gateways)??[],o=e.filter(S=>nt(S.id,a)),d=new Set(a.map(S=>S.id));if(e.length===0){ne(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>V());return}if(o.length===0){ne(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>V());return}const u=d.has("default")?"":"default";ne(`
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
            ${o.map(S=>`<option value="${n(S.id)}">${n(S.id)} (${n(S.mode)})</option>`).join("")}
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
      `,S=>{if(S==="cancel"){V();return}S==="create"&&Ot()}),(T=document.getElementById("new-gw-id"))==null||T.focus()}async function Ot(){const e=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),o=document.getElementById("new-gw-port"),d=document.getElementById("new-gw-err"),u=(e==null?void 0:e.value.trim())??"",T=(a==null?void 0:a.value)??"",S=Number.parseInt((o==null?void 0:o.value.trim())??"",10),v=_=>{d&&(d.className="error small",d.textContent=_)};if(!u){v("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!T){v("Pick the machine it runs on.");return}try{await pn({id:u,placement:{targetId:T,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(S)?S:4e3,Networks:[]}})}catch(_){v(pe(_));return}V(),await $()}async function Ft(e,a){const o=await He(a),d=e.textContent;e.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{i||(e.textContent=d)},1500)}function pe(e){return e instanceof Error?e.message:String(e)}function Re(e){return e instanceof we&&e.hint?` — ${e.hint}`:""}return()=>{i=!0,C==null||C(),V()}}const Wn="run",_n={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Kn={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function zn(s,i){let r=!1,t=null,f=null;const w={devnet:null},R={devnet:null},b={devnet:[]};let A=null;const j={devnet:!1};let U=null;const z={devnet:null},B={devnet:null};s.innerHTML=`
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
    ${ie()}
  `;const F=s.querySelector("#services-body");ve(s,(c,m)=>{ge(c,m)}),I();async function I(){try{const c=await sn(i);if(r)return;t=c,f=null}catch(c){if(r)return;t=null,f=x(c)}p()}function C(c){return t==null?void 0:t.services.find(m=>m.id===c)}function p(){if(!r){if(f){F.innerHTML=`<p class="error">Could not read this machine's services: ${n(f)}</p>`;return}if(!t){F.innerHTML='<p class="muted">Loading…</p>';return}F.innerHTML=`
      ${$(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(N).join("")}
      </div>
    `}}function $(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const m=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(m)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function N(c){const m=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${H(c)}
        </div>
        <p class="muted small">${n(_n[c.id]??"")}</p>

        ${c.error?G(c):""}
        ${c.blocked?`<div class="banner banner-warn">${n(c.blocked)}</div>`:""}
        ${m.map(E=>`<div class="banner banner-warn">${n(E)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${n(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Q(c)}

        ${ee(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(E=>se(c,E)).join("")}
        </div>
        ${R[c.id]?`<p class="error small">${n(R[c.id])}</p>`:""}
        ${J(c)}

        ${de(c)}
      </div>
    `}function H(c){switch(c.status.State){case"running":return O("running","ok");case"created-but-stopped":return O("stopped","warn");case"not-created":return O("not created","neutral");default:return O("unknown","bad")}}function G(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function Q(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const m=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${m}.</p>`}function ee(c){const m=c.endpoints??[];return m.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":m.map(E=>`
        <div class="endpoint-row">
          ${$e("ok")}
          <span class="muted small">${n(E.label)}</span>
          <code class="endpoint-url">${n(E.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(E.url)}">Copy</button>
        </div>`).join("")}function se(c,m){const E=Kn[m];if(!E)return"";const W=w[c.id],Y=m==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:E.label;return`
      <button class="${E.className}" data-action="svc-${m}" data-svc="${n(c.id)}"
              title="${n(E.title)}" ${W?"disabled":""}>
        ${W===m?'<span class="spinner" aria-label="working"></span>':n(Y)}
      </button>
    `}function J(c){const m=b[c.id]??[];return m.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(m.join(`
`))}</pre>
      </div>
    `}function de(c){const m=j[c.id],E=he(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(c.id)}">
            ${m?"Close":"Edit"}
          </button>
        </div>
        ${m?ce():`<p class="small">${E}</p>`}
        ${z[c.id]?`<p class="error small">${n(z[c.id])}</p>`:""}
        ${B[c.id]?`<p class="muted small">${n(B[c.id])}</p>`:""}
      </div>
    `}function he(c){const m=c.devnet;return m?`Chain ${m.ChainID} · a block every ${n(m.BlockTime)} · JSON-RPC on ${n(m.BindAddr)}:${m.HTTPPort} · WebSocket on ${n(m.BindAddr)}:${m.WSPort}`:"—"}function ce(c){return oe()}function oe(){const c=U;return c?`
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
    `:""}function re(){j.devnet&&U&&(U.BlockTime=ue("#dev-blocktime",U.BlockTime),U.HTTPPort=be("#dev-http",U.HTTPPort),U.WSPort=be("#dev-ws",U.WSPort),U.BindAddr=ue("#dev-bind",U.BindAddr))}function ue(c,m){const E=s.querySelector(c);return E?E.value.trim():m}function be(c,m){const E=s.querySelector(c);if(!E)return m;const W=Number.parseInt(E.value.trim(),10);return Number.isFinite(W)?W:m}async function ge(c,m){const E=m.dataset.svc??"";switch(c){case"refresh":await I();return;case"copy":m.dataset.copy&&await y(m,m.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await g(E,c.slice(4));return;case"svc-create":case"svc-recreate":await l(E);return;case"svc-wipe":L(E);return;case"toggle-config":k(E);return;case"save-config":await P(E);return;default:return}}async function g(c,m){if(!w[c]){w[c]=m,R[c]=null,p();try{await on(i,c,m)}catch(E){R[c]=`${m} failed: ${x(E)}${M(E)}`}w[c]=null,await I()}}async function l(c){if(!w[c]){w[c]="create",R[c]=null,b[c]=["starting…"],p();try{await cn(i,c)}catch(m){R[c]=`${x(m)}${M(m)}`,b[c]=[],w[c]=null,p();return}A==null||A(),A=Ze(i,m=>{if(r)return;const E=m.err?`${m.stepId}: ${m.err}`:m.line?`${m.stepId}: ${m.line}`:`${m.stepId}: done`;if(b[c]=[...(b[c]??[]).filter(Y=>Y!=="starting…"),E],!!m.err||m.stepId===Wn&&!!m.done){A==null||A(),A=null,w[c]=null,m.err&&(R[c]="Provisioning failed — see the log below."),I();return}p()})}}function k(c){if(re(),j[c]=!j[c],z[c]=null,B[c]=null,j[c]){const m=C(c);m!=null&&m.devnet&&(U={...m.devnet})}p()}async function P(c){var W;re(),z[c]=null,B[c]=null;const m=U;if(!m)return;if(m.HTTPPort===m.WSPort){z[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",p();return}try{await dn(i,c,m)}catch(Y){z[c]=x(Y),p();return}const E=((W=C(c))==null?void 0:W.status.State)==="running";j[c]=!1,B[c]=E?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await I()}function L(c){const m=C(c);if(!m)return;const E=(m.restartsOnWipe??[]).map(D=>{var te;return((te=C(D))==null?void 0:te.label)??D});ne(`
        <h2>Wipe ${n(m.label)}</h2>
        <p class="error">This deletes ${n(m.wipeDiscards)}</p>
        ${E.length?`<p>It also restarts what sits in front of it: ${n(E.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(c)}</button>
        </div>
      `,D=>{if(D==="cancel"||D==="close"){V(),I();return}D==="confirm"&&q(c)});const W=document.getElementById("wipe-confirm-input"),Y=document.getElementById("wipe-confirm-btn");W==null||W.addEventListener("input",()=>{Y&&(Y.disabled=W.value.trim()!==c)}),W==null||W.focus()}async function q(c){const m=document.getElementById("wipe-confirm-btn");m&&(m.disabled=!0,m.textContent="Wiping…");let E;try{E=await rn(i,c)}catch(W){const Y=Be();if(Y){const D=document.createElement("p");D.className="error small",D.textContent=`Wipe failed: ${x(W)}${M(W)}`,Y.appendChild(D)}m&&(m.disabled=!1,m.textContent=`Wipe ${c}`);return}h(c,E)}function h(c,m){const E=C(c),W=X=>{var ke;return((ke=C(X))==null?void 0:ke.label)??X},Y=[];Y.push(m.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const X of m.report.VolumesRemoved??[])Y.push(`Volume ${X} deleted.`);for(const X of m.report.VolumesAbsent??[])Y.push(`Volume ${X} was already gone.`);m.report.Recreated&&Y.push("Container re-created from your saved configuration.");const D=(m.report.Cascaded??[]).map(W),te=(m.report.CascadeSkipped??[]).map(W);ne(`
        <h2>${n((E==null?void 0:E.label)??c)} wiped</h2>
        <ul class="plain-list">${Y.map(X=>`<li>${n(X)}</li>`).join("")}</ul>
        ${D.length?`<p class="ok">Restarted in front of it: ${n(D.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${te.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(te.join(", "))}.</p>`:""}
        ${m.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(m.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,X=>{(X==="close"||X==="cancel")&&(V(),I())})}async function y(c,m){const E=await He(m),W=c.textContent;c.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{r||(c.textContent=W)},1500)}function x(c){return c instanceof Error?c.message:String(c)}function M(c){return c instanceof we&&c.hint?` — ${c.hint}`:""}return()=>{r=!0,A==null||A(),V()}}const Gn="local";function Jn(s){let i=!1,r=!1,t="",f=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ie()}
  `;const w=s.querySelector("#targets-body");ve(s,(p,$)=>{U(p,$)}),R();async function R(){try{const[p,$,N]=await Promise.all([Ie(),Ee(),_t()]);if(i)return;t=N.os,A(p,$)}catch(p){if(i)return;w.innerHTML=`<p class="error">Failed to load machines: ${n(String(p))}</p>`}}function b(){f&&A(f.targets,f.catalog)}function A(p,$){f={targets:p,catalog:$};const N=t==="linux",H=[...p].sort((ee,se)=>(ee.mode==="local"?-1:0)-(se.mode==="local"?-1:0)),G=H.length?`<div class="card-grid">${H.map(ee=>Vn(ee,$,ee.mode!=="local"||N,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Q=p.some(ee=>ee.mode==="local");w.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${G}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(N,Q)}
        ${r?Yn():""}
      </section>
    `}function j(p,$){const N=`
      <div class="card">
        <h3>A server over SSH ${O("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${p?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${p?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,H=p?`
        <div class="card">
          <h3>This machine ${O("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${n(t)})`:""} ${O("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return $?`<div class="card-grid card-grid-wide">${N}</div>`:`<div class="card-grid card-grid-wide">${p?H+N:N+H}</div>`}async function U(p,$){var N;if(p==="add-local"){await z();return}if(p==="delete-target"){const H=$.dataset.id;if(!H||!await Ae({title:"Remove machine",body:`Remove "${H}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await B(H);return}if(p==="toggle-ssh"){r=!r,C(),b(),r&&((N=s.querySelector("#ssh-host"))==null||N.focus());return}p==="add-ssh"&&await F()}async function z(){C();try{await st({id:Gn,mode:"local"}),await R()}catch(p){I(p)}}async function B(p){try{await Kt(p),await R()}catch($){I($)}}async function F(){const p=s.querySelector("#ssh-host"),$=s.querySelector("#ssh-user"),N=s.querySelector("#ssh-key"),H=s.querySelector("#ssh-port"),G=s.querySelector("#ssh-id");if(!p||!$||!N||!H||!G)return;const Q=p.value.trim(),ee=$.value.trim(),se=N.value.trim(),J=H.value.trim(),de=G.value.trim();if(C(),!Q||!ee||!se){I(new Error("host, user, and key path are required"));return}const he=de||Zn(Q),ce={Host:Q,User:ee,KeyPath:se};if(J){const re=Number.parseInt(J,10);if(!Number.isFinite(re)||re<=0){I(new Error("port must be a positive number"));return}ce.Port=re}const oe=s.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await st({id:he,mode:"ssh",ssh:ce}),r=!1,await R()}catch(re){I(re),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function I(p){let $=s.querySelector("#targets-error");$||(w.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=s.querySelector("#targets-error")),$.textContent=String(p instanceof Error?p.message:p)}function C(){var p;(p=s.querySelector("#targets-error"))==null||p.remove()}return()=>{i=!0}}function Vn(s,i,r,t){const f=s.wire,w=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:w,b=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(s.id)}">Devnet</a>`;let A,j;if(!f&&!r)A=`${O("can't run a node","warn")} ${O(t||"not Linux","neutral")}`,j=`
      ${b}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Preview setup wizard</a>
    `;else if(!f)A=O("not set up","neutral"),j=`
      <a class="btn" href="#/setup/${encodeURIComponent(s.id)}">Run setup wizard</a>
      ${b}
    `;else{const U=i.networks.find(B=>B.ChainID===f.ChainID),z=U?U.Name:`chain ${f.ChainID}`;A=`${O(z,"ok")} ${O(f.ExecID,"neutral")} ${O(f.BeaconID,"neutral")}${f.Archive?" "+O("archive","warn"):""}`,j=`
      <a class="btn" href="#/dash/${encodeURIComponent(s.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(s.id)}">Logs</a>
      ${b}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${R}</p>
      <p>${A}</p>
      <div class="card-actions">
        ${j}
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function Yn(){return`
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
  `}function Zn(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Ve=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Fe=8545,qe=5052,je=30303,Xn=[369,943,1],dt={369:"default",943:"practise here first"};function Qn(s,i){let r=!1;const t={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ie()}</div>`;const f=s.querySelector("#wizard-body"),w=s.querySelector("#wizard-footer");ve(s,(h,y)=>{be(h,y)}),Xe(s,(h,y)=>{h==="exec-select"?t.execId=y:h==="beacon-select"&&(t.beaconId=y),b()}),s.addEventListener("change",h=>{const y=h.target;y instanceof HTMLInputElement&&(y.id==="data-dir-input"?(ge(),se()):y.id==="checkpoint-toggle"?(t.checkpoint=y.checked,b()):y.id==="exec-snapshot-toggle"&&(t.execSnapshot=y.checked,b()))}),R();async function R(){try{const[h,y]=await Promise.all([Ee(),Ie()]);if(r)return;t.catalog=h;const x=y.find(M=>M.id===i);x!=null&&x.wire&&(t.chainId=x.wire.ChainID,t.execId=x.wire.ExecID,t.beaconId=x.wire.BeaconID,t.archive=x.wire.Archive,x.wire.ExecHTTPPort&&(t.execHTTPPort=String(x.wire.ExecHTTPPort)),x.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(x.wire.BeaconHTTPPort)),x.wire.ExecP2PPort&&(t.execP2PPort=String(x.wire.ExecP2PPort)),x.wire.RPCBindAddr&&(t.rpcBindAddr=x.wire.RPCBindAddr)),b()}catch(h){if(r)return;t.loadError=String(h instanceof Error?h.message:h),b()}}function b(){if(t.loadError){f.innerHTML=`<p class="error">Failed to load: ${n(t.loadError)}</p>`;return}t.catalog&&(f.innerHTML=`
      ${q(t.step)}
      ${j()}
    `,A())}function A(){var y;const h=(y=t.catalog)==null?void 0:y.networks.find(x=>x.ChainID===t.chainId);w.innerHTML=h?ie(h.Name,h.LearnURL):ie()}function j(){switch(t.step){case"network":return U();case"clients":return z();case"mode":return oe();case"review":return re();case"run":return ue()}}function U(){const h=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Xn.map(x=>{const M=h.networks.find(E=>E.ChainID===x);if(!M)return"";const c=t.chainId===x,m=dt[x]?O(dt[x],x===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${x}" type="button">
          <h3>${n(M.Name)} <span class="muted">(chain ${x})</span></h3>
          ${m}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function z(){const h=t.catalog,y=h.networks.find(c=>c.ChainID===t.chainId);if(!y)return'<p class="error">Unknown network.</p>';(t.execId===null||!y.ExecClients.includes(t.execId))&&(t.execId=y.ExecClients[0]??null),(t.beaconId===null||!y.BeaconClients.includes(t.beaconId))&&(t.beaconId=y.BeaconClients[0]??null);const x=y.ExecClients.map(c=>de(c,h)),M=y.BeaconClients.map(c=>de(c,h));return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${n(y.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${Ye("exec-select",x,t.execId)}
        </label>
        ${ce(t.execId,h)}
        <label>
          Beacon client
          ${Ye("beacon-select",M,t.beaconId)}
        </label>
        ${ce(t.beaconId,h)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function B(h){return h<=0?"—":h>=1?`~${h.toFixed(1)} TB`:`~${Math.round(h*1e3)} GB`}const F=1.1,I=.5,C="Valve reth snapshot",p="rough estimate";function $(h){return h.SnapshotSizeTB}function N(h){return h.SnapshotSizeTB*I}function H(h){return`<p class="muted small">${B($(h))} is the measured size of Valve's reth snapshot for ${n(h.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function G(h){return{archive:$(h)*1e12*F,full:N(h)*1e12*F}}function Q(h,y){if(!h)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${n(y)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${n(y)}</code>: ${n(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==y)return"";const x=G(h),M=t.freeBytes>=x.archive,c=t.freeBytes>=x.full,m=`<p class="muted small">Free at <code>${n(y)}</code>: <strong>${Se(t.freeBytes)}</strong> — archive ${M?"fits":"won't fit"} (${B($(h))}, ${C}), full ${c?"fits":"won't fit"} (${B(N(h))}, ${p}).</p>`;let E="";return t.downgradeNote?E=`<p class="banner banner-warn">${n(t.downgradeNote)}</p>`:c||(E=`<p class="banner banner-warn">Neither full (${B(N(h))}, ${p}) nor archive (${B($(h))}, ${C}) fits the free space here — choose a location with more room.</p>`),m+E}function ee(h,y){if(t.downgradeNote=null,!h||t.freeBytes===null)return;const x=G(h);t.archive&&t.freeBytes<x.archive&&t.freeBytes>=x.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${y} for archive (${B($(h))}, ${C}) — switched to Full (${B(N(h))}, ${p}). Pick a location with more room to run archive.`)}async function se(){var x;if(t.chainId===null)return;const h=(x=t.catalog)==null?void 0:x.networks.find(M=>M.ChainID===t.chainId),y=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,b();try{const{freeBytes:M}=await zt(t.targetId,y);if(r)return;t.freeBytes=M,t.probedPath=y,ee(h,y)}catch(M){if(r)return;t.freeBytes=null,t.probedPath=y,t.diskError=String(M instanceof Error?M.message:M)}t.diskProbing=!1,b()}function J(h){return h?/^https?:\/\/.+/i.test(h)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(h,y){const x=y.clients.find(M=>M.id===h);return{value:h,label:x?`${x.id} — ${he(x.repo)}`:h}}function he(h){const y=h.split("/");return y.length>=4?y[3]:h}function ce(h,y){const x=h?y.clients.find(c=>c.id===h):void 0;if(!x)return"";const M=x.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(x.repo)}" target="_blank" rel="noopener noreferrer">${n(M)}</a></p>`}function oe(){var W,Y,D;const h=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",y=(W=t.catalog)==null?void 0:W.networks.find(te=>te.ChainID===t.chainId),x=((D=(Y=t.catalog)==null?void 0:Y.clients.find(te=>te.id===t.execId))==null?void 0:D.snapshotSupported)??!1,M=y?`${B(N(y))} (${p})`:"Smaller",c=y?`${B($(y))} (${C})`:"Much larger",m=y?` on ${n(y.Name)}`:"",E=y?t.checkpoint?y.SyncLabel:y.GenesisSyncLabel:"";return`
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
          ${y?`<p class="sync-estimate">⏱ Estimated initial sync${m}: <strong>${n(E)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((y==null?void 0:y.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((y==null?void 0:y.CheckpointURL)??"")}" value="${n(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${n(t.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${x?`
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
              <tr><th>Approx. disk footprint${m}</th><td class="yes">${M}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${y?H(y):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${y?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${M}${y?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(h)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(h)}" value="${n(t.dataDir)}" />
          </label>
          ${Q(y,t.dataDir||h)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(h)}/jwt.hex" value="${n(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Fe})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Fe}" value="${n(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${n(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${qe})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${qe}" value="${n(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${n(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${je})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${je}" value="${n(t.execP2PPort)}" />
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
    `}function re(){const y=t.catalog.networks.find(X=>X.ChainID===t.chainId),x=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,M=t.jwtPath||`${x}/jwt.hex`,c=Ve.map(X=>`<li>${n(X.title)}</li>`).join(""),m=P(t.execHTTPPort,Fe),E=P(t.beaconHTTPPort,qe),W=P(t.execP2PPort,je),Y=m||E||W?`<tr><th>Non-default ports</th><td>${[m?`exec HTTP ${m}`:null,E?`beacon HTTP ${E}`:null,W?`exec p2p ${W}`:null].filter(X=>X!==null).map(n).join(", ")}</td></tr>`:"",{addr:D}=g(t.rpcBindAddr),te=D?`<tr><th>RPC bind address</th><td><code>${n(D)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((y==null?void 0:y.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(x)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(M)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${n(t.checkpointUrl||(y==null?void 0:y.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${Y}
            ${te}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${c}</ol>
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${t.starting?"disabled":""}>
            ${t.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function ue(){const y=t.catalog.networks.find(D=>D.ChainID===t.chainId),x=y==null?void 0:y.LearnURL,M=new Set(t.events.filter(D=>D.done).map(D=>D.stepId)),c=new Set(t.events.filter(D=>D.err).map(D=>D.stepId)),m=new Map;for(const D of t.events){if(!D.line)continue;const te=m.get(D.stepId)??[];te.push(D.line),m.set(D.stepId,te)}const E=Ve.map(D=>{var Me;const te=M.has(D.id),X=c.has(D.id),ke=X?O("failed","bad"):te?O("done","ok"):O("pending","neutral"),De=(m.get(D.id)??[]).slice(-5),Ue=(Me=t.events.find(xe=>xe.stepId===D.id&&xe.err))==null?void 0:Me.err,Ke=D.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${x?` <a href="${n(x)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${te?"step-done":""} ${X?"step-error":""}">
          <div class="step-head">${ke} <strong>${n(D.title)}</strong></div>
          ${Ke}
          ${De.length?`<pre class="step-log">${De.map(xe=>n(xe)).join(`
`)}</pre>`:""}
          ${Ue?`<p class="error small">${n(Ue)}</p>`:""}
        </li>
      `}).join(""),W=t.events.some(D=>D.err),Y=Ve.every(D=>M.has(D.id))||t.events.some(D=>D.stepId==="handshake"&&D.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${E}</ol>
        ${Y&&!W?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${n(t.startError)}</p>`:""}
        ${W?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(h,y){switch(h){case"pick-network":t.chainId=Number(y.dataset.chainId),t.execId=null,t.beaconId=null,b();break;case"goto-network":t.step="network",b();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",b();break;case"goto-mode":t.step="mode",b(),se();break;case"goto-review":if(ge(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){b();break}t.step="review",b();break;case"start-setup":L();break}}function ge(){const h=s.querySelectorAll('input[name="mode"]');for(const D of Array.from(h))D.checked&&(t.archive=D.value==="archive");const y=s.querySelector("#data-dir-input"),x=s.querySelector("#jwt-path-input");y&&(t.dataDir=y.value.trim()),x&&(t.jwtPath=x.value.trim());const M=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),m=s.querySelector("#exec-p2p-port-input");M&&(t.execHTTPPort=M.value.trim()),c&&(t.beaconHTTPPort=c.value.trim()),m&&(t.execP2PPort=m.value.trim());const E=s.querySelector("#rpc-bind-addr-input");E&&(t.rpcBindAddr=E.value.trim());const W=s.querySelector("#checkpoint-url-input");W&&(t.checkpointUrl=W.value.trim());const Y=s.querySelector("#snapshot-key-input");Y&&(t.snapshotKey=Y.value.trim()),t.execHTTPPortError=k(t.execHTTPPort).error??null,t.beaconHTTPPortError=k(t.beaconHTTPPort).error??null,t.execP2PPortError=k(t.execP2PPort).error??null,t.rpcBindAddrError=g(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?J(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function g(h){if(!h)return{};const y=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(h);return y?y.slice(1).every(x=>Number(x)<=255)?{addr:h}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(h)&&h.includes(":")?{addr:h}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const l=/^\d+$/;function k(h){if(!h)return{};if(!l.test(h))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const y=Number(h);return!Number.isInteger(y)||y<1||y>65535?{error:"Port must be between 1 and 65535."}:{port:y}}function P(h,y){const{port:x}=k(h);if(!(x===void 0||x===y))return x}async function L(){var m;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(m=t.streamStop)==null||m.call(t),t.streamStop=null,b();const h={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(h.DataDir=t.dataDir),t.jwtPath&&(h.JWTPath=t.jwtPath);const y=P(t.execHTTPPort,Fe),x=P(t.beaconHTTPPort,qe),M=P(t.execP2PPort,je);y!==void 0&&(h.ExecHTTPPort=y),x!==void 0&&(h.BeaconHTTPPort=x),M!==void 0&&(h.ExecP2PPort=M);const{addr:c}=g(t.rpcBindAddr);c!==void 0&&(h.RPCBindAddr=c),t.checkpoint?t.checkpointUrl&&(h.CheckpointURL=t.checkpointUrl):h.NoCheckpoint=!0,t.execSnapshot&&(h.ExecSnapshot=!0,h.SnapshotKey=t.snapshotKey);try{await Gt(t.targetId,h)}catch(E){if(!(E instanceof we&&E.status===409)){t.starting=!1,t.startError=String(E instanceof Error?E.message:E),b();return}}t.starting=!1,t.step="run",b(),t.streamStop=Ze(t.targetId,E=>{r||(t.events.push(E),t.step==="run"&&b())})}function q(h){const y=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],M=y.map(c=>c.id).indexOf(h);return`
      <ol class="wizard-progress">
        ${y.map((c,m)=>`<li class="${m===M?"current":m<M?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var h;r=!0,(h=t.streamStop)==null||h.call(t)}}const ea=document.querySelector("#app"),{contentEl:ta,setActiveNav:na}=xn(ea);let le=null;function aa(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[r,t]=i;return r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:t?decodeURIComponent(t):void 0}:{screen:r??"targets"}}function ye(s){const i=document.createElement("div");return ta.replaceChildren(i),s(i)}function pt(){if(le){try{le()}catch{}le=null}const{screen:s,id:i}=aa();switch(na(s),s){case"setup":if(!i){location.hash="#/targets";return}le=ye(r=>Qn(r,i));break;case"dash":if(!i){location.hash="#/targets";return}le=ye(r=>Nn(r,i));break;case"logs":if(!i){location.hash="#/targets";return}le=ye(r=>An(r,i));break;case"security":if(!i){location.hash="#/targets";return}le=ye(r=>Hn(r,i));break;case"diag":if(!i){location.hash="#/targets";return}le=ye(r=>Bn(r,i));break;case"services":if(!i){location.hash="#/targets";return}le=ye(r=>zn(r,i));break;case"analytics":if(!i){location.hash="#/rpc";return}le=ye(r=>Rn(r,i));break;case"rpc":le=ye(r=>jn(r));break;case"settings":le=ye(r=>Un(r));break;case"targets":default:le=ye(r=>Jn(r));break}}window.addEventListener("hashchange",pt);pt();
