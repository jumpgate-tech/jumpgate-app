var An=Object.defineProperty;var Nn=(n,o,i)=>o in n?An(n,o,{enumerable:!0,configurable:!0,writable:!0,value:i}):n[o]=i;var _e=(n,o,i)=>Nn(n,typeof o!="symbol"?o+"":o,i);(function(){const o=document.createElement("link").relList;if(o&&o.supports&&o.supports("modulepreload"))return;for(const d of document.querySelectorAll('link[rel="modulepreload"]'))t(d);new MutationObserver(d=>{for(const m of d)if(m.type==="childList")for(const g of m.addedNodes)g.tagName==="LINK"&&g.rel==="modulepreload"&&t(g)}).observe(document,{childList:!0,subtree:!0});function i(d){const m={};return d.integrity&&(m.integrity=d.integrity),d.referrerPolicy&&(m.referrerPolicy=d.referrerPolicy),d.crossOrigin==="use-credentials"?m.credentials="include":d.crossOrigin==="anonymous"?m.credentials="omit":m.credentials="same-origin",m}function t(d){if(d.ep)return;d.ep=!0;const m=i(d);fetch(d.href,m)}})();function Pt(){return te("/api/host")}function Ie(){return te("/api/catalog")}function Te(){return te("/api/targets")}function tt(n){return te("/api/targets",{method:"POST",headers:ve,body:JSON.stringify(n)})}function Bn(n){return te(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function Hn(n,o){return te(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(o)}`)}function Dn(n,o){return te(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:ve,body:JSON.stringify(o)})}function Fe(n,o){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return i.onmessage=t=>{try{o(JSON.parse(t.data))}catch{}},()=>i.close()}function Mn(n,o){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return i.onmessage=t=>{try{o(JSON.parse(t.data))}catch{}},()=>i.close()}function Un(n,o=200){return te(`/api/targets/${encodeURIComponent(n)}/logs?n=${o}`)}function On(n,o){const i=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return i.onmessage=t=>{try{o(JSON.parse(t.data))}catch{}},()=>i.close()}function yt(n,o){const i=o===void 0?{}:{lines:o};return te(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:ve,body:JSON.stringify(i)})}function Fn(n,o,i){return te(`/api/targets/${encodeURIComponent(n)}/services/${o}/${i}`,{method:"POST"})}function jn(n,o){return te(`/api/targets/${encodeURIComponent(n)}/services/${o}/clear`,{method:"POST",headers:ve,body:JSON.stringify({Confirm:o})})}function qn(n){return te(`/api/targets/${encodeURIComponent(n)}/du`)}function Wn(n){return te(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function _n(n){return te(`/api/targets/${encodeURIComponent(n)}/firewall`)}function Kn(n){return te(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function Vn(n){return te(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function Et(n){return te(`/api/targets/${encodeURIComponent(n)}/containers`)}function Gn(n,o,i){return te(`/api/targets/${encodeURIComponent(n)}/containers/${o}/${i}`,{method:"POST"})}async function zn(n,o){const i=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${o}/wipe`,{method:"POST",headers:ve,body:JSON.stringify({Confirm:o})}),t=await i.text();let d=null;try{d=t?JSON.parse(t):null}catch{}if(d&&typeof d=="object"&&"report"in d)return d;const m=d&&typeof d=="object"&&typeof d.error=="string"?d.error:i.statusText||`HTTP ${i.status}`;throw new ke(i.status,m)}function Jn(n,o){return te(`/api/targets/${encodeURIComponent(n)}/containers/${o}/provision`,{method:"POST"})}async function Yn(n){const o=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:ve}),i=await o.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const d=t&&typeof t=="object"&&typeof t.error=="string"?t.error:o.statusText||`HTTP ${o.status}`;throw new ke(o.status,d)}function Zn(n,o,i){return te(`/api/targets/${encodeURIComponent(n)}/containers/${o}/config`,{method:"PUT",headers:ve,body:JSON.stringify(i)})}function it(){return te("/api/gateways")}async function Xn(n){await te(`/api/orphans/${encodeURIComponent(n)}`,{method:"DELETE"})}function It(n){return te("/api/gateways",{method:"POST",headers:ve,body:JSON.stringify(n)})}function Rt(n){return te(`/api/gateways/${encodeURIComponent(n)}/tls/verify`)}function Qn(n){return te(`/api/gateways/${encodeURIComponent(n)}/traffic`)}function ea(n){return te(`/api/gateways/${encodeURIComponent(n)}/analytics`)}function Lt(n,o=!1){const i=o?"?refresh=1":"";return te(`/api/gateways/${encodeURIComponent(n)}/capabilities${i}`)}function ta(n){return te(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function nt(n,o){return te(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:ve,body:JSON.stringify(o)})}function At(n,o){return te(`/api/gateways/${encodeURIComponent(n)}/${o}`,{method:"POST"})}function na(n){return te(`/api/gateways/${encodeURIComponent(n)}/trust-cert`,{method:"POST"})}function at(n){return te(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Nt(n){const o=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:ve,body:JSON.stringify({Confirm:n})}),i=await o.text();let t=null;try{t=i?JSON.parse(i):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const d=t&&typeof t=="object"&&typeof t.error=="string"?t.error:o.statusText||`HTTP ${o.status}`;throw new ke(o.status,d)}function aa(n){return te(`/api/chainlist/${n}`)}function Bt(n,o){return te(`/api/gateways/${encodeURIComponent(n)}/knownset/${o}`)}function sa(){return te("/api/settings")}function oa(n){return te("/api/settings",{method:"PUT",headers:ve,body:JSON.stringify(n)})}class ke extends Error{constructor(i,t,d,m){super(t);_e(this,"status");_e(this,"hint");_e(this,"code");this.name="ApiError",this.status=i,this.hint=d,this.code=m}}const ve={"Content-Type":"application/json"};async function te(n,o){const i=await fetch(n,o);if(!i.ok){let d=i.statusText||`HTTP ${i.status}`,m,g;try{const p=await i.json();p&&typeof p.error=="string"&&p.error&&(d=p.error),p&&typeof p.hint=="string"&&p.hint&&(m=p.hint),p&&typeof p.code=="string"&&p.code&&(g=p.code)}catch{}throw new ke(i.status,d,m,g)}if(i.status===204)return;const t=await i.text();return t?JSON.parse(t):void 0}const gt="https://learn.valve.city/rpc";function a(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function he(n,o){const i=n&&o&&o!==gt?` <span class="footer-sep">·</span> <a href="${a(o)}" target="_blank" rel="noopener noreferrer">${a(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(gt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${i}
    </footer>
  `}function ra(n){n.innerHTML=`
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
  `;const o=n.querySelector("#content"),i=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:o,setActiveNav:d=>{const m=d==="machine"?"targets":d==="home"||d==="panel"?"rpc":d;for(const g of i)g.classList.toggle("active",g.dataset.nav===m)}}}function de(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function ia(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function ca(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const o=Math.round(n/60),i=Math.floor(o/60),t=o%60;if(i===0)return`~${t}m`;if(i<48)return`~${i}h ${t}m`;const d=Math.floor(i/24),m=i%24;return`~${d}d ${m}h`}function q(n,o){return`<span class="badge badge-${o}">${a(n)}</span>`}function Ce(n){return`<span class="dot dot-${n}"></span>`}const vt=["B","KB","MB","GB","TB","PB"];function Pe(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let o=n,i=0;for(;o>=1024&&i<vt.length-1;)o/=1024,i++;const t=o<10?2:o<100?1:0;return`${o.toFixed(t)} ${vt[i]}`}async function De(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function $e(n,o){n.addEventListener("click",i=>{const t=i.target.closest("[data-action]");if(!t||!n.contains(t))return;const d=t.dataset.action;d&&o(d,t,i)})}function st(n,o,i){const t=o.find(m=>m.value===i),d=o.map(m=>`
      <li class="dropdown-option${m.value===i?" selected":""}" role="option"
          aria-selected="${m.value===i}" data-value="${a(m.value)}">
        ${a(m.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${d}</ul>
    </div>
  `}function Me(n){n.querySelectorAll(".dropdown.open").forEach(o=>{var i;o.classList.remove("open"),(i=o.querySelector(".dropdown-trigger"))==null||i.setAttribute("aria-expanded","false")})}function ct(n,o){n.addEventListener("click",d=>{const m=d.target,g=m.closest(".dropdown-trigger");if(g&&n.contains(g)){const A=g.closest(".dropdown"),j=!!A&&!A.classList.contains("open");Me(n),A&&j&&(A.classList.add("open"),g.setAttribute("aria-expanded","true"));return}const p=m.closest(".dropdown-option");if(p&&n.contains(p)){const A=p.closest(".dropdown");Me(n),o((A==null?void 0:A.dataset.dropdown)??"",p.dataset.value??"");return}Me(n)});const i=d=>{if(!n.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}const m=d.target;(!m.closest(".dropdown")||!n.contains(m))&&Me(n)},t=d=>{if(!n.isConnected){document.removeEventListener("click",i),document.removeEventListener("keydown",t);return}d.key==="Escape"&&Me(n)};document.addEventListener("click",i),document.addEventListener("keydown",t)}const Je="app-modal";let ze=null;function le(n,o){ne();const i=document.createElement("div");i.className="modal-overlay",i.id=Je,i.innerHTML=`<div class="modal">${n}</div>`,i.addEventListener("click",d=>{const m=d.target.closest("[data-modal-action]");m!=null&&m.dataset.modalAction?o(m.dataset.modalAction):d.target===i&&o("cancel")});const t=d=>{d.key==="Escape"&&o("cancel")};document.addEventListener("keydown",t),ze=t,document.body.appendChild(i)}function ne(){var n;(n=document.getElementById(Je))==null||n.remove(),ze&&(document.removeEventListener("keydown",ze),ze=null)}function Oe(){return document.querySelector(`#${Je} .modal`)}function Ee(n){return new Promise(o=>{var d;let i=!1;const t=m=>{i||(i=!0,ne(),o(m))};le(`
        <h2>${a(n.title)}</h2>
        <p>${a(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${a(n.confirmLabel)}</button>
        </div>
      `,m=>t(m==="confirm")),(d=document.querySelector(`#${Je} [data-modal-action="confirm"]`))==null||d.focus()})}const Xe=5e3,la=60;function da(n,o){let i=!1,t=null,d=null,m=null,g=null;const p=[];n.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${he()}</div>`;const A=n.querySelector("#an-body");$e(n,(y,u)=>{var S;y==="toggle-endpoint"&&((S=u.closest(".an-endpoint"))==null||S.classList.toggle("expanded"))}),j();async function j(){try{t=((await it()).gateways??[]).find(u=>u.id===o)??null}catch(y){if(i)return;m=String(y instanceof Error?y.message:y),D();return}if(!i){if(!t){D();return}await _(),g=window.setInterval(()=>void _(),Xe)}}async function _(){try{const y=await ea(o);if(i)return;E(y),d=y,m=null}catch(y){if(i)return;m=String(y instanceof Error?y.message:y)}D()}function E(y){if(!y.enabled||y.error)return;const u=p[p.length-1];u&&u.since!==y.since&&(p.length=0);const S=new Map;for(const L of y.networks??[])S.set(L.chainId,L.received);p.push({t:Date.now(),since:y.since,received:S}),p.length>la&&p.shift()}function D(){i||(A.innerHTML=M())}function M(){return m&&!d?`<h1>Analytics</h1><p class="error">${a(m)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${R(t)}
      ${d?h(d):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${a(o)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function R(y){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${a(y.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${x()}</div>
      </div>
    `}function x(){if(!d)return"";if(!d.enabled)return"counters off";if(d.error)return"could not be read";const y=d.since?new Date(d.since):null;return y&&!Number.isNaN(y.getTime())?`totals since the gateway started, ${a(y.toLocaleString())}<br />re-read every ${Xe/1e3}s`:`re-read every ${Xe/1e3}s`}function h(y){return y.enabled?y.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a(y.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:$(y)+U(y):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function $(y){const u=y.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${u.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':u.map(S=>w(S)).join("")}
      </section>
    `}function w(y){const u=y.methods??[],S=y.endpoints??[],L=y.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${y.chainId}</span>
          <span class="band-name">${a(y.name)}</span>
          ${W(y)}
        </div>
        <div class="an-stats">
          ${B("Received",de(y.received),"what clients asked this chain for")}
          ${B("Answered",de(y.answered),"returned by one of your endpoints")}
          ${B("From cache",de(y.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${B("Failed",de(y.failed),"asked for and never answered",y.failed>0?"bad":"")}
        </div>
        ${se(y.chainId)}
        ${L?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':re("Method",u.map(H=>({label:H.method,l:H})))+re("Endpoint",S.map(H=>({label:H.upstream,l:H})))+K(y)}
      </div>
    `}function B(y,u,S,L=""){return`
      <div class="an-stat${L?" an-stat-"+L:""}" title="${a(S)}">
        <span class="an-stat-n">${a(u)}</span>
        <span class="an-stat-l">${a(y)}</span>
      </div>
    `}function W(y){const u=J(y.chainId);if(u===null)return'<span class="an-rate muted small">measuring rate…</span>';const S=Math.round((p[p.length-1].t-p[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${S}s apart.">
      ${a(u.toFixed(u<10?2:0))} req/s <span class="muted">over the last ${S}s</span>
    </span>`}function J(y){if(p.length<2)return null;const u=p[0],S=p[p.length-1],L=(S.t-u.t)/1e3;if(L<=0)return null;const H=(S.received.get(y)??0)-(u.received.get(y)??0);return H<0?null:H/L}function se(y){if(p.length<3)return"";const u=[];for(let k=1;k<p.length;k++){const I=p[k-1],z=p[k],l=(z.t-I.t)/1e3,v=(z.received.get(y)??0)-(I.received.get(y)??0);u.push(l>0&&v>=0?v/l:0)}const S=Math.max(...u);if(S<=0)return"";const L=240,H=28,Z=u.length>1?L/(u.length-1):L,b=u.map((k,I)=>`${(I*Z).toFixed(1)},${(H-k/S*H).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${S.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${L} ${H}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${b}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(S.toFixed(2))} req/s</span>
      </div>
    `}function K(y){const u=[];return y.cached.count>0&&u.push(`${a(de(y.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${y.cached.mean===null?"":`, in ${a(Ue(y.cached.mean))} on average`}.`),y.failedLatency.count>0&&y.failedLatency.mean!==null&&u.push(`The ${a(de(y.failedLatency.count))} that failed took
         ${a(Ue(y.failedLatency.mean))} on average to fail.`),u.length===0?"":`<p class="muted small">${u.join(" ")}</p>`}function re(y,u){return u.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${a(y)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${u.map(S=>ue(S.label,S.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function ue(y,u){return`
      <tr>
        <td><code>${a(y)}</code></td>
        <td class="an-num">${de(u.count)}</td>
        <td class="an-num">${u.mean===null?'<span class="muted">—</span>':a(Ue(u.mean))}</td>
        <td>${ee(u)}</td>
      </tr>
    `}function ee(y){const u=y.buckets??[];if(u.length===0||y.count===0)return'<span class="muted small">—</span>';let S=0;const L=[];for(const Z of u){const b=Z.count-S;S=Z.count,L.push({label:G(Z.le),n:Math.max(0,b)})}return L.reduce((Z,b)=>Z+b.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(L.filter(Z=>Z.n>0).map(Z=>`${Z.n} ${Z.label}`).join(" · "))}">
        ${L.map((Z,b)=>Z.n===0?"":`<span class="an-band an-band-${Math.min(b,4)}" style="flex:${Z.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(O(L))}</span>
    `}function O(y){for(let u=y.length-1;u>=0;u--)if(y[u].n>0)return`slowest ${y[u].label}`;return""}function G(y){if(y==="+Inf")return"30s or more";const u=Number(y);return Number.isFinite(u)?`under ${Ue(u)}`:`under ${y}`}function U(y){const u=y.endpoints??[];return`
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
                     <tbody>${u.map(S=>Y(S)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function Y(y){const u=y.errors??[],S=u.reduce((H,Z)=>H+Z.count,0),L=u.length>0;return`
      <tr class="an-endpoint${L?" expandable":""}" ${L?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${a(y.upstream)}</code>
          ${y.chainId?`<span class="muted small">chain ${y.chainId}</span>`:""}
          ${y.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${de(y.requests)}</td>
        <td class="an-num${S>0?" bad":""}">${S>0?de(S):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${y.headLag>0?de(y.headLag):'<span class="muted">0</span>'}</td>
        <td>${pe(y)}</td>
      </tr>
      ${L?F(y,u):""}
    `}function pe(y){const u=[];return y.scored?(u.push(y.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String(y.position))}</span>`),u.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a(y.score.toFixed(3))}</span>`),y.primarySwitches>1&&u.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${de(y.primarySwitches)} switches</span>`),y.excludedSeconds>0&&u.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(Ue(y.excludedSeconds))}</span>`),`<span class="an-selection">${u.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function F(y,u){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${u.map(S=>`
                    <tr>
                      <td class="an-num">${de(S.count)}</td>
                      <td><code>${a(S.class)}</code></td>
                      <td>${S.severity?`<span class="badge badge-${S.severity==="critical"?"bad":"warn"}">${a(S.severity)}</span>`:""}</td>
                      <td class="muted small">${a(S.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${a(y.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{i=!0,g!==null&&window.clearInterval(g)}}function Ue(n){return!Number.isFinite(n)||n<0?"—":n>0&&n<5e-4?"<1ms":n<1?`${Math.round(n*1e3)}ms`:n<60?`${n<10?n.toFixed(1):Math.round(n)}s`:`${Math.round(n/60)}m`}function ua(n,o){let i=!1,t=null,d=null,m=!1,g=!1;n.innerHTML=`<h1>Network diagnostics: ${a(o)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${he()}</div>`;const p=n.querySelector("#diag-body"),A=n.querySelector("#diag-footer");$e(n,(h,$)=>{var w;if(h==="run")_();else if(h==="toggle")(w=$.closest(".check-item"))==null||w.classList.toggle("expanded");else if(h==="copy"){const B=$.dataset.copy;B&&x($,B)}}),j();async function j(){let h,$;try{const[B,W]=await Promise.all([Te(),Ie()]);h=B.find(J=>J.id===o),$=W}catch(B){if(i)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(B))}</p>`;return}if(i)return;if(!h){p.innerHTML=`<p class="error">Target "${a(o)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(o)}">Run the setup wizard →</a></p>`;return}const w=$==null?void 0:$.networks.find(B=>B.ChainID===h.wire.ChainID);w&&(A.innerHTML=he(w.Name,w.LearnURL));try{t=await Vn(o),g=!0}catch(B){d=String(B instanceof Error?B.message:B)}i||E()}async function _(){m=!0,d=null,E();try{t=await Kn(o),g=!0}catch(h){d=String(h instanceof Error?h.message:h)}m=!1,i||E()}function E(){p.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(o)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${m?"disabled":""}>${m?"Running…":"Run diagnostics"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${D()}
    `}function D(){if(!g&&!d)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const h=new Date(t.at).toLocaleString(),$=t.failedId?`<p><strong>Failed at: ${a(M(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${a(h)} — trigger: ${a(t.trigger)}</p>
      ${$}
      <ul class="check-list">${t.items.map(R).join("")}</ul>
    `}function M(h){var $;return(($=t==null?void 0:t.items.find(w=>w.ID===h))==null?void 0:$.Title)??h}function R(h){const $=h.Status==="pass"?"ok":h.Status==="fail"?"bad":h.Status==="warn"?"warn":"neutral",w=h.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${w?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${q(w?"failed here":h.Status,$)}
          <strong>${a(h.Title)}</strong>
          <span class="muted small check-detail-inline">${a(h.Detail)}</span>
        </button>
        <div class="check-body">
          <details${w?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(h.Why)}</p>
          </details>
          ${h.Fix?`
                <details${w?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(h.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function x(h,$){const w=await De($),B=h.textContent;h.textContent=w?"Copied!":"Copy failed",setTimeout(()=>{i||(h.textContent=B)},1500)}return()=>{i=!0}}const pa=85,Qe={exec:"Execution",beacon:"Beacon"};function ha(n,o){let i=!1,t=null,d=null,m=null,g=null,p=null,A=null,j=null,_=null;const E={exec:null,beacon:null};let D=null;n.innerHTML=`<h1>Dashboard: ${a(o)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${he()}</div>`;const M=n.querySelector("#dash-body"),R=n.querySelector("#dash-footer");M.addEventListener("click",u=>{const S=u.target.closest("[data-action]");if(!S||!M.contains(S))return;const L=S.dataset.action;if(L==="svc-action"){const H=S.dataset.svc,Z=S.dataset.kind;H&&Z&&Y(H,Z)}else if(L==="open-clear"){const H=S.dataset.svc;H&&F(H)}else if(L==="copy"){const H=S.dataset.copy;H&&pe(S,H)}else L==="retry-du"?h():L==="retry-endpoints"&&$()}),x();async function x(){let u,S;try{const[H,Z]=await Promise.all([Te(),Ie()]);u=H.find(b=>b.id===o),S=Z}catch(H){if(i)return;M.innerHTML=`<p class="error">Failed to load target: ${a(String(H))}</p>`;return}if(i)return;if(!u){M.innerHTML=`<p class="error">Target "${a(o)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){M.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(o)}">Run the setup wizard →</a></p>`;return}const L=S==null?void 0:S.networks.find(H=>H.ChainID===u.wire.ChainID);L&&(R.innerHTML=he(L.Name,L.LearnURL)),M.innerHTML='<p class="muted">Connecting…</p>',t=Mn(o,H=>{i||(w(H),d=H,m=H,B())}),h(),$()}async function h(){A=null;try{p=await qn(o)}catch(u){p=null,A=String(u instanceof Error?u.message:u)}i||B()}async function $(){_=null;try{j=await Wn(o)}catch(u){j=null,_=String(u instanceof Error?u.message:u)}i||B()}function w(u){if(!d)return;const S=(new Date(u.at).getTime()-new Date(d.at).getTime())/1e3,L=u.execHead-d.execHead;if(S>0&&L>=0){const H=L/S;g=g===null?H:g*.7+H*.3}}function B(){if(!m)return;const u=m;M.innerHTML=`
      <p class="dash-status">${W(u)}</p>
      <div class="card-grid">
        ${G(u)}
        ${se(u)}
        ${K(u)}
        ${re(u)}
        ${ue(u)}
        ${ee()}
      </div>
      <p class="muted small">Last updated ${a(new Date(u.at).toLocaleTimeString())}</p>
    `}function W(u){return!u.execActive&&!u.beaconActive?q("Node not running","bad"):u.execSyncing||u.beaconDistance>0?q("Syncing","warn"):q("Running · synced","ok")}function J(u){const L=u.refHead>0?u.refHead-u.execHead:null,H=L!==null&&L>0&&g&&g>0?ca(L/g):L!==null&&L<=0?"caught up":"—";return{lag:L,eta:H}}function se(u){const{lag:S,eta:L}=J(u);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${u.execActive?u.execSyncing?q("syncing","warn"):u.execHead===0?q("no data","neutral"):q("synced","ok"):q("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${de(u.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${S!==null?de(u.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${S!==null?de(Math.max(S,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${L}</dd></div>
        </dl>
      </div>
    `}function K(u){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${u.beaconActive?u.beaconSlot===0?q("no data","neutral"):u.beaconDistance===0?q("synced","ok"):q("syncing","warn"):q("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${de(u.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${de(u.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function re(u){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${de(u.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${de(u.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function ue(u){const S=u.diskUsedPct>=pa,L=`
      <div class="meter"><div class="meter-fill ${S?"meter-warn":""}" style="width:${Math.min(u.diskUsedPct,100)}%"></div></div>
      <p>${ia(u.diskUsedPct)} used</p>
    `;if(A)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="error small">${a(A)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!p)return`
        <div class="card ${S?"card-warn":""}">
          <h3>Storage</h3>
          ${L}
          <p class="muted">Loading…</p>
        </div>
      `;const H=p.ExpectedExecBytes>0?Math.min(p.ExecBytes/p.ExpectedExecBytes*100,100):0,Z=p.ExpectedBeaconBytes>0?Math.min(p.BeaconBytes/p.ExpectedBeaconBytes*100,100):0,{lag:b,eta:k}=J(u),I=b!==null&&b>0&&g!==null&&g>0;return`
      <div class="card ${S?"card-warn":""}">
        <h3>Storage</h3>
        ${L}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Pe(p.ExecBytes)} of ~${Pe(p.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${H}%"></div></div>
        ${I?`<p class="muted small">Estimated time remaining: ${a(k)}</p>`:""}
        <p class="muted small">Beacon — ${Pe(p.BeaconBytes)} of ~${Pe(p.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${Z}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Pe(p.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(p.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(p.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function ee(){if(_)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(_)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const u=j,S=u.ExecReachable&&!u.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",L=u.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(u.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${Ce(u.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(u.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${Ce(u.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(u.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(u.BeaconHTTP)}">Copy</button>
        </div>
        ${S}
        ${L}
      </div>
    `}function O(u,S){const L=Qe[u],H=E[u],Z=(b,k,I)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${u}" data-kind="${b}" ${H!==null||I?"disabled":""}>${H===b?U():a(k)}</button>`;return`
      <div class="service-row">
        <span>${a(L)} ${S?q("active","ok"):q("down","bad")}</span>
        <div class="service-actions">
          ${Z("start","Start",S)}
          ${Z("stop","Stop",!S)}
          ${Z("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${u}" ${H!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function G(u){return`
      <div class="card">
        <h3>Services</h3>
        ${O("exec",u.execActive)}
        ${O("beacon",u.beaconActive)}
        ${D?`<p class="error small">${a(D)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(o)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(o)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(o)}">Diagnostics →</a>
        </p>
      </div>
    `}function U(){return'<span class="spinner" aria-label="working"></span>'}async function Y(u,S){if(E[u]===null){E[u]=S,D=null,B();try{await Fn(o,u,S)}catch(L){D=`${Qe[u]} ${S} failed: ${L instanceof Error?L.message:String(L)}`}E[u]=null,i||B()}}async function pe(u,S){const L=await De(S),H=u.textContent;u.textContent=L?"Copied!":"Copy failed",setTimeout(()=>{i||(u.textContent=H)},1500)}function F(u){const S=Qe[u],L=p?Pe(u==="exec"?p.ExecBytes:p.BeaconBytes):"unknown (disk usage hasn't loaded)";le(`
        <h2>Clear ${a(S)} data</h2>
        <p class="error">
          This stops the ${a(S.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(L)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(u)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,b=>{if(b==="cancel"){ne();return}b==="confirm"&&y(u)});const H=document.getElementById("clear-confirm-input"),Z=document.getElementById("clear-confirm-btn");H==null||H.addEventListener("input",()=>{Z&&(Z.disabled=H.value.trim()!==u)}),H==null||H.focus()}async function y(u){const S=document.getElementById("clear-confirm-btn");S&&(S.disabled=!0,S.textContent="Clearing…");try{await jn(o,u),ne(),h()}catch(L){const H=Oe();if(H){const Z=document.createElement("p");Z.className="error small",Z.textContent=`Clear failed: ${L instanceof Error?L.message:String(L)}`,H.appendChild(Z)}S&&(S.disabled=!1,S.textContent="Clear and resync")}}return()=>{i=!0,t==null||t(),ne()}}const $t=500,wt="valve-node-app.explain-consent";function fa(n,o){let i=!1,t=null;const d=[];n.innerHTML=`
    <h1>Logs: ${a(o)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${he()}</div>
  `;const m=n.querySelector("#logs-body"),g=n.querySelector("#logs-footer");$e(n,x=>{x==="explain"&&_()}),p();async function p(){let x,h;try{const[w,B]=await Promise.all([Te(),Ie()]);x=w.find(W=>W.id===o),h=B}catch(w){if(i)return;m.innerHTML=`<p class="error">Failed to load target: ${a(String(w))}</p>`;return}if(i)return;if(!x){m.innerHTML=`<p class="error">Target "${a(o)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!x.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(o)}">Run the setup wizard →</a></p>`;return}const $=h==null?void 0:h.networks.find(w=>w.ChainID===x.wire.ChainID);$&&(g.innerHTML=he($.Name,$.LearnURL));try{const w=await Un(o,200);if(i)return;d.push(...w)}catch(w){if(i)return;m.innerHTML=`<p class="error">Failed to load logs: ${a(String(w))}</p>`;return}A(),t=On(o,w=>{i||(d.push(w),d.length>$t&&d.splice(0,d.length-$t),A())})}function A(){const x=d.filter($=>$.severity==="error"||$.severity==="critical");m.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${d.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${q(String(x.length),x.length?"bad":"neutral")}</h2>
          <div class="log-lines">${x.length?x.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const h=m.querySelector(".log-lines");h&&(h.scrollTop=h.scrollHeight)}function j(x){const h=x.severity||"info",$=x.learnUrl?` <a href="${a(x.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(h)}">
        <span class="log-time">${a(new Date(x.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(x.unit)}</span>
        <span class="log-sev">${a(h)}</span>
        <span class="log-text">${a(x.line)}</span>
        ${x.explain?`<div class="log-explain">${a(x.explain)}${$}</div>`:""}
      </div>
    `}async function _(){const x=d.filter($=>$.severity==="error"||$.severity==="critical").map($=>$.line).slice(-40);if(!(localStorage.getItem(wt)==="1")){E(x);return}await D(x)}function E(x){const h=x.length?`<pre class="explain-excerpt">${x.map($=>a($)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';M(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${h}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,$=>{$==="proceed"?(localStorage.setItem(wt,"1"),R(),D(x)):R()})}async function D(x){M('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const h=x.length?await yt(o,x):await yt(o);if(i)return;M(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(h.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${h.sentExcerpt.map($=>a($)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&R()})}catch(h){if(i)return;if(h instanceof ke&&h.status===409){M(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,$=>{$==="close"&&R()});return}M(`
        <h2>Explain failed</h2>
        <p class="error">${a(h instanceof Error?h.message:String(h))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&R()})}}function M(x,h){R();const $=document.createElement("div");$.className="modal-overlay",$.id="explain-modal",$.innerHTML=`<div class="modal">${x}</div>`,$.addEventListener("click",w=>{const B=w.target.closest("[data-modal-action]");B!=null&&B.dataset.modalAction&&h(B.dataset.modalAction),w.target===$&&h("cancel")}),document.body.appendChild($)}function R(){var x;(x=document.getElementById("explain-modal"))==null||x.remove()}return()=>{i=!0,t==null||t(),R()}}const ma="run",ba={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},ya={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function ga(n,o){let i=!1,t=null,d=null;const m={devnet:null},g={devnet:null},p={devnet:[]};let A=null;const j={devnet:!1};let _=null;const E={devnet:null},D={devnet:null};n.innerHTML=`
    <div class="page-head">
      <h1>Services: ${a(o)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${he()}
  `;const M=n.querySelector("#services-body");$e(n,(l,v)=>{F(l,v)}),R();async function R(){try{const l=await Et(o);if(i)return;t=l,d=null}catch(l){if(i)return;t=null,d=I(l)}h()}function x(l){return t==null?void 0:t.services.find(v=>v.id===l)}function h(){if(!i){if(d){M.innerHTML=`<p class="error">Could not read this machine's services: ${a(d)}</p>`;return}if(!t){M.innerHTML='<p class="muted">Loading…</p>';return}M.innerHTML=`
      ${$(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(w).join("")}
      </div>
    `}}function $(l){if(l.present&&l.reachable&&!l.hint)return`<p class="muted small">Docker: ${a(l.flavor)}${l.serverVersion?` ${a(l.serverVersion)}`:""} · reachable</p>`;const v=l.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(v)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${l.detail?`<div class="small">${a(l.detail)}</div>`:""}
        ${l.hint?`<div class="small">${a(l.hint)}</div>`:""}
      </div>
    `}function w(l){const v=l.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(l.label)}</h2>
          ${B(l)}
        </div>
        <p class="muted small">${a(ba[l.id]??"")}</p>

        ${l.error?W(l):""}
        ${l.blocked?`<div class="banner banner-warn">${a(l.blocked)}</div>`:""}
        ${v.map(N=>`<div class="banner banner-warn">${a(N)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(l.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${l.status.Image?`<code>${a(l.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${J(l)}

        ${se(l)}

        <div class="card-actions">
          ${(l.actions??[]).map(N=>K(l,N)).join("")}
        </div>
        ${g[l.id]?`<p class="error small">${a(g[l.id])}</p>`:""}
        ${re(l)}

        ${ue(l)}
      </div>
    `}function B(l){switch(l.status.State){case"running":return q("running","ok");case"created-but-stopped":return q("stopped","warn");case"not-created":return q("not created","neutral");default:return q("unknown","bad")}}function W(l){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(l.error??"")}</div>
        ${l.hint?`<div class="small">${a(l.hint)}</div>`:""}
      </div>
    `}function J(l){if(l.status.State!=="created-but-stopped"||l.status.ExitCode===0)return"";const v=l.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${l.status.ExitCode}${v}.</p>`}function se(l){const v=l.endpoints??[];return v.length===0?l.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":v.map(N=>`
        <div class="endpoint-row">
          ${Ce("ok")}
          <span class="muted small">${a(N.label)}</span>
          <code class="endpoint-url">${a(N.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(N.url)}">Copy</button>
        </div>`).join("")}function K(l,v){const N=ya[v];if(!N)return"";const X=m[l.id],ae=v==="create"?`Create ${l.id==="devnet"?"devnet":"gateway"}`:N.label;return`
      <button class="${N.className}" data-action="svc-${v}" data-svc="${a(l.id)}"
              title="${a(N.title)}" ${X?"disabled":""}>
        ${X===v?'<span class="spinner" aria-label="working"></span>':a(ae)}
      </button>
    `}function re(l){const v=p[l.id]??[];return v.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(v.join(`
`))}</pre>
      </div>
    `}function ue(l){const v=j[l.id],N=ee(l);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${l.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(l.id)}">
            ${v?"Close":"Edit"}
          </button>
        </div>
        ${v?O():`<p class="small">${N}</p>`}
        ${E[l.id]?`<p class="error small">${a(E[l.id])}</p>`:""}
        ${D[l.id]?`<p class="muted small">${a(D[l.id])}</p>`:""}
      </div>
    `}function ee(l){const v=l.devnet;return v?`Chain ${v.ChainID} · a block every ${a(v.BlockTime)} · JSON-RPC on ${a(v.BindAddr)}:${v.HTTPPort} · WebSocket on ${a(v.BindAddr)}:${v.WSPort}`:"—"}function O(l){return G()}function G(){const l=_;return l?`
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
    `:""}function U(){j.devnet&&_&&(_.BlockTime=Y("#dev-blocktime",_.BlockTime),_.HTTPPort=pe("#dev-http",_.HTTPPort),_.WSPort=pe("#dev-ws",_.WSPort),_.BindAddr=Y("#dev-bind",_.BindAddr))}function Y(l,v){const N=n.querySelector(l);return N?N.value.trim():v}function pe(l,v){const N=n.querySelector(l);if(!N)return v;const X=Number.parseInt(N.value.trim(),10);return Number.isFinite(X)?X:v}async function F(l,v){const N=v.dataset.svc??"";switch(l){case"refresh":await R();return;case"copy":v.dataset.copy&&await k(v,v.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await y(N,l.slice(4));return;case"svc-create":case"svc-recreate":await u(N);return;case"svc-wipe":H(N);return;case"toggle-config":S(N);return;case"save-config":await L(N);return;default:return}}async function y(l,v){if(!m[l]){m[l]=v,g[l]=null,h();try{await Gn(o,l,v)}catch(N){g[l]=`${v} failed: ${I(N)}${z(N)}`}m[l]=null,await R()}}async function u(l){if(!m[l]){m[l]="create",g[l]=null,p[l]=["starting…"],h();try{await Jn(o,l)}catch(v){g[l]=`${I(v)}${z(v)}`,p[l]=[],m[l]=null,h();return}A==null||A(),A=Fe(o,v=>{if(i)return;const N=v.err?`${v.stepId}: ${v.err}`:v.line?`${v.stepId}: ${v.line}`:`${v.stepId}: done`;if(p[l]=[...(p[l]??[]).filter(ae=>ae!=="starting…"),N],!!v.err||v.stepId===ma&&!!v.done){A==null||A(),A=null,m[l]=null,v.err&&(g[l]="Provisioning failed — see the log below."),R();return}h()})}}function S(l){if(U(),j[l]=!j[l],E[l]=null,D[l]=null,j[l]){const v=x(l);v!=null&&v.devnet&&(_={...v.devnet})}h()}async function L(l){var X;U(),E[l]=null,D[l]=null;const v=_;if(!v)return;if(v.HTTPPort===v.WSPort){E[l]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",h();return}try{await Zn(o,l,v)}catch(ae){E[l]=I(ae),h();return}const N=((X=x(l))==null?void 0:X.status.State)==="running";j[l]=!1,D[l]=N?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await R()}function H(l){const v=x(l);if(!v)return;const N=(v.restartsOnWipe??[]).map(V=>{var ce;return((ce=x(V))==null?void 0:ce.label)??V});le(`
        <h2>Wipe ${a(v.label)}</h2>
        <p class="error">This deletes ${a(v.wipeDiscards)}</p>
        ${N.length?`<p>It also restarts what sits in front of it: ${a(N.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${a(l)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(l)}</button>
        </div>
      `,V=>{if(V==="cancel"||V==="close"){ne(),R();return}V==="confirm"&&Z(l)});const X=document.getElementById("wipe-confirm-input"),ae=document.getElementById("wipe-confirm-btn");X==null||X.addEventListener("input",()=>{ae&&(ae.disabled=X.value.trim()!==l)}),X==null||X.focus()}async function Z(l){const v=document.getElementById("wipe-confirm-btn");v&&(v.disabled=!0,v.textContent="Wiping…");let N;try{N=await zn(o,l)}catch(X){const ae=Oe();if(ae){const V=document.createElement("p");V.className="error small",V.textContent=`Wipe failed: ${I(X)}${z(X)}`,ae.appendChild(V)}v&&(v.disabled=!1,v.textContent=`Wipe ${l}`);return}b(l,N)}function b(l,v){const N=x(l),X=ie=>{var Re;return((Re=x(ie))==null?void 0:Re.label)??ie},ae=[];ae.push(v.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const ie of v.report.VolumesRemoved??[])ae.push(`Volume ${ie} deleted.`);for(const ie of v.report.VolumesAbsent??[])ae.push(`Volume ${ie} was already gone.`);v.report.Recreated&&ae.push("Container re-created from your saved configuration.");const V=(v.report.Cascaded??[]).map(X),ce=(v.report.CascadeSkipped??[]).map(X);le(`
        <h2>${a((N==null?void 0:N.label)??l)} wiped</h2>
        <ul class="plain-list">${ae.map(ie=>`<li>${a(ie)}</li>`).join("")}</ul>
        ${V.length?`<p class="ok">Restarted in front of it: ${a(V.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${ce.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(ce.join(", "))}.</p>`:""}
        ${v.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(v.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,ie=>{(ie==="close"||ie==="cancel")&&(ne(),R())})}async function k(l,v){const N=await De(v),X=l.textContent;l.textContent=N?"Copied!":"Copy failed",setTimeout(()=>{i||(l.textContent=X)},1500)}function I(l){return l instanceof Error?l.message:String(l)}function z(l){return l instanceof ke&&l.hint?` — ${l.hint}`:""}return()=>{i=!0,A==null||A(),ne()}}const et=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ke=8545,Ve=5052,Ge=30303,va=[369,943,1],kt={369:"default",943:"practise here first"};function $a(n,o){let i=!1;const t={targetId:o,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${a(o)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${he()}</div>`;const d=n.querySelector("#wizard-body"),m=n.querySelector("#wizard-footer");$e(n,(b,k)=>{pe(b,k)}),ct(n,(b,k)=>{b==="exec-select"?t.execId=k:b==="beacon-select"&&(t.beaconId=k),p()}),n.addEventListener("change",b=>{const k=b.target;k instanceof HTMLInputElement&&(k.id==="data-dir-input"?(F(),K()):k.id==="checkpoint-toggle"?(t.checkpoint=k.checked,p()):k.id==="exec-snapshot-toggle"&&(t.execSnapshot=k.checked,p()))}),g();async function g(){try{const[b,k]=await Promise.all([Ie(),Te()]);if(i)return;t.catalog=b;const I=k.find(z=>z.id===o);I!=null&&I.wire&&(t.chainId=I.wire.ChainID,t.execId=I.wire.ExecID,t.beaconId=I.wire.BeaconID,t.archive=I.wire.Archive,I.wire.ExecHTTPPort&&(t.execHTTPPort=String(I.wire.ExecHTTPPort)),I.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(I.wire.BeaconHTTPPort)),I.wire.ExecP2PPort&&(t.execP2PPort=String(I.wire.ExecP2PPort)),I.wire.RPCBindAddr&&(t.rpcBindAddr=I.wire.RPCBindAddr)),p()}catch(b){if(i)return;t.loadError=String(b instanceof Error?b.message:b),p()}}function p(){if(t.loadError){d.innerHTML=`<p class="error">Failed to load: ${a(t.loadError)}</p>`;return}t.catalog&&(d.innerHTML=`
      ${Z(t.step)}
      ${j()}
    `,A())}function A(){var k;const b=(k=t.catalog)==null?void 0:k.networks.find(I=>I.ChainID===t.chainId);m.innerHTML=b?he(b.Name,b.LearnURL):he()}function j(){switch(t.step){case"network":return _();case"clients":return E();case"mode":return G();case"review":return U();case"run":return Y()}}function _(){const b=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${va.map(I=>{const z=b.networks.find(N=>N.ChainID===I);if(!z)return"";const l=t.chainId===I,v=kt[I]?q(kt[I],I===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${l?"selected":""}" data-action="pick-network" data-chain-id="${I}" type="button">
          <h3>${a(z.Name)} <span class="muted">(chain ${I})</span></h3>
          ${v}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function E(){const b=t.catalog,k=b.networks.find(l=>l.ChainID===t.chainId);if(!k)return'<p class="error">Unknown network.</p>';(t.execId===null||!k.ExecClients.includes(t.execId))&&(t.execId=k.ExecClients[0]??null),(t.beaconId===null||!k.BeaconClients.includes(t.beaconId))&&(t.beaconId=k.BeaconClients[0]??null);const I=k.ExecClients.map(l=>ue(l,b)),z=k.BeaconClients.map(l=>ue(l,b));return`
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
          ${st("exec-select",I,t.execId)}
        </label>
        ${O(t.execId,b)}
        <label>
          Beacon client
          ${st("beacon-select",z,t.beaconId)}
        </label>
        ${O(t.beaconId,b)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function D(b){return b<=0?"—":b>=1?`~${b.toFixed(1)} TB`:`~${Math.round(b*1e3)} GB`}const M=1.1,R=.5,x="Valve reth snapshot",h="rough estimate";function $(b){return b.SnapshotSizeTB}function w(b){return b.SnapshotSizeTB*R}function B(b){return`<p class="muted small">${D($(b))} is the measured size of Valve's reth snapshot for ${a(b.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function W(b){return{archive:$(b)*1e12*M,full:w(b)*1e12*M}}function J(b,k){if(!b)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${a(k)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${a(k)}</code>: ${a(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==k)return"";const I=W(b),z=t.freeBytes>=I.archive,l=t.freeBytes>=I.full,v=`<p class="muted small">Free at <code>${a(k)}</code>: <strong>${Pe(t.freeBytes)}</strong> — archive ${z?"fits":"won't fit"} (${D($(b))}, ${x}), full ${l?"fits":"won't fit"} (${D(w(b))}, ${h}).</p>`;let N="";return t.downgradeNote?N=`<p class="banner banner-warn">${a(t.downgradeNote)}</p>`:l||(N=`<p class="banner banner-warn">Neither full (${D(w(b))}, ${h}) nor archive (${D($(b))}, ${x}) fits the free space here — choose a location with more room.</p>`),v+N}function se(b,k){if(t.downgradeNote=null,!b||t.freeBytes===null)return;const I=W(b);t.archive&&t.freeBytes<I.archive&&t.freeBytes>=I.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${k} for archive (${D($(b))}, ${x}) — switched to Full (${D(w(b))}, ${h}). Pick a location with more room to run archive.`)}async function K(){var I;if(t.chainId===null)return;const b=(I=t.catalog)==null?void 0:I.networks.find(z=>z.ChainID===t.chainId),k=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,p();try{const{freeBytes:z}=await Hn(t.targetId,k);if(i)return;t.freeBytes=z,t.probedPath=k,se(b,k)}catch(z){if(i)return;t.freeBytes=null,t.probedPath=k,t.diskError=String(z instanceof Error?z.message:z)}t.diskProbing=!1,p()}function re(b){return b?/^https?:\/\/.+/i.test(b)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function ue(b,k){const I=k.clients.find(z=>z.id===b);return{value:b,label:I?`${I.id} — ${ee(I.repo)}`:b}}function ee(b){const k=b.split("/");return k.length>=4?k[3]:b}function O(b,k){const I=b?k.clients.find(l=>l.id===b):void 0;if(!I)return"";const z=I.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(I.repo)}" target="_blank" rel="noopener noreferrer">${a(z)}</a></p>`}function G(){var X,ae,V;const b=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",k=(X=t.catalog)==null?void 0:X.networks.find(ce=>ce.ChainID===t.chainId),I=((V=(ae=t.catalog)==null?void 0:ae.clients.find(ce=>ce.id===t.execId))==null?void 0:V.snapshotSupported)??!1,z=k?`${D(w(k))} (${h})`:"Smaller",l=k?`${D($(k))} (${x})`:"Much larger",v=k?` on ${a(k.Name)}`:"",N=k?t.checkpoint?k.SyncLabel:k.GenesisSyncLabel:"";return`
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
          ${k?`<p class="sync-estimate">⏱ Estimated initial sync${v}: <strong>${a(N)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${a((k==null?void 0:k.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${a((k==null?void 0:k.CheckpointURL)??"")}" value="${a(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${a(t.checkpointUrlError)}</p>`:""}
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
              <tr><th>Approx. disk footprint${v}</th><td class="yes">${z}</td><td class="limited">${l}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${k?B(k):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${l}${k?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${z}${k?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(b)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(b)}" value="${a(t.dataDir)}" />
          </label>
          ${J(k,t.dataDir||b)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(b)}/jwt.hex" value="${a(t.jwtPath)}" />
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
    `}function U(){const k=t.catalog.networks.find(ie=>ie.ChainID===t.chainId),I=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,z=t.jwtPath||`${I}/jwt.hex`,l=et.map(ie=>`<li>${a(ie.title)}</li>`).join(""),v=L(t.execHTTPPort,Ke),N=L(t.beaconHTTPPort,Ve),X=L(t.execP2PPort,Ge),ae=v||N||X?`<tr><th>Non-default ports</th><td>${[v?`exec HTTP ${v}`:null,N?`beacon HTTP ${N}`:null,X?`exec p2p ${X}`:null].filter(ie=>ie!==null).map(a).join(", ")}</td></tr>`:"",{addr:V}=y(t.rpcBindAddr),ce=V?`<tr><th>RPC bind address</th><td><code>${a(V)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${a(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${a((k==null?void 0:k.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${a(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${a(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${a(I)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${a(z)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${a(t.checkpointUrl||(k==null?void 0:k.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${ae}
            ${ce}
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
    `}function Y(){const k=t.catalog.networks.find(V=>V.ChainID===t.chainId),I=k==null?void 0:k.LearnURL,z=new Set(t.events.filter(V=>V.done).map(V=>V.stepId)),l=new Set(t.events.filter(V=>V.err).map(V=>V.stepId)),v=new Map;for(const V of t.events){if(!V.line)continue;const ce=v.get(V.stepId)??[];ce.push(V.line),v.set(V.stepId,ce)}const N=et.map(V=>{var We;const ce=z.has(V.id),ie=l.has(V.id),Re=ie?q("failed","bad"):ce?q("done","ok"):q("pending","neutral"),je=(v.get(V.id)??[]).slice(-5),qe=(We=t.events.find(Le=>Le.stepId===V.id&&Le.err))==null?void 0:We.err,Ye=V.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${I?` <a href="${a(I)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${ce?"step-done":""} ${ie?"step-error":""}">
          <div class="step-head">${Re} <strong>${a(V.title)}</strong></div>
          ${Ye}
          ${je.length?`<pre class="step-log">${je.map(Le=>a(Le)).join(`
`)}</pre>`:""}
          ${qe?`<p class="error small">${a(qe)}</p>`:""}
        </li>
      `}).join(""),X=t.events.some(V=>V.err),ae=et.every(V=>z.has(V.id))||t.events.some(V=>V.stepId==="handshake"&&V.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${ae&&!X?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${a(t.startError)}</p>`:""}
        ${X?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function pe(b,k){switch(b){case"pick-network":t.chainId=Number(k.dataset.chainId),t.execId=null,t.beaconId=null,p();break;case"goto-network":t.step="network",p();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",p();break;case"goto-mode":t.step="mode",p(),K();break;case"goto-review":if(F(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){p();break}t.step="review",p();break;case"start-setup":H();break}}function F(){const b=n.querySelectorAll('input[name="mode"]');for(const V of Array.from(b))V.checked&&(t.archive=V.value==="archive");const k=n.querySelector("#data-dir-input"),I=n.querySelector("#jwt-path-input");k&&(t.dataDir=k.value.trim()),I&&(t.jwtPath=I.value.trim());const z=n.querySelector("#exec-http-port-input"),l=n.querySelector("#beacon-http-port-input"),v=n.querySelector("#exec-p2p-port-input");z&&(t.execHTTPPort=z.value.trim()),l&&(t.beaconHTTPPort=l.value.trim()),v&&(t.execP2PPort=v.value.trim());const N=n.querySelector("#rpc-bind-addr-input");N&&(t.rpcBindAddr=N.value.trim());const X=n.querySelector("#checkpoint-url-input");X&&(t.checkpointUrl=X.value.trim());const ae=n.querySelector("#snapshot-key-input");ae&&(t.snapshotKey=ae.value.trim()),t.execHTTPPortError=S(t.execHTTPPort).error??null,t.beaconHTTPPortError=S(t.beaconHTTPPort).error??null,t.execP2PPortError=S(t.execP2PPort).error??null,t.rpcBindAddrError=y(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?re(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function y(b){if(!b)return{};const k=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(b);return k?k.slice(1).every(I=>Number(I)<=255)?{addr:b}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(b)&&b.includes(":")?{addr:b}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const u=/^\d+$/;function S(b){if(!b)return{};if(!u.test(b))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const k=Number(b);return!Number.isInteger(k)||k<1||k>65535?{error:"Port must be between 1 and 65535."}:{port:k}}function L(b,k){const{port:I}=S(b);if(!(I===void 0||I===k))return I}async function H(){var v;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(v=t.streamStop)==null||v.call(t),t.streamStop=null,p();const b={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(b.DataDir=t.dataDir),t.jwtPath&&(b.JWTPath=t.jwtPath);const k=L(t.execHTTPPort,Ke),I=L(t.beaconHTTPPort,Ve),z=L(t.execP2PPort,Ge);k!==void 0&&(b.ExecHTTPPort=k),I!==void 0&&(b.BeaconHTTPPort=I),z!==void 0&&(b.ExecP2PPort=z);const{addr:l}=y(t.rpcBindAddr);l!==void 0&&(b.RPCBindAddr=l),t.checkpoint?t.checkpointUrl&&(b.CheckpointURL=t.checkpointUrl):b.NoCheckpoint=!0,t.execSnapshot&&(b.ExecSnapshot=!0,b.SnapshotKey=t.snapshotKey);try{await Dn(t.targetId,b)}catch(N){if(!(N instanceof ke&&N.status===409)){t.starting=!1,t.startError=String(N instanceof Error?N.message:N),p();return}}t.starting=!1,t.step="run",p(),t.streamStop=Fe(t.targetId,N=>{i||(t.events.push(N),t.step==="run"&&p())})}function Z(b){const k=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],z=k.map(l=>l.id).indexOf(b);return`
      <ol class="wizard-progress">
        ${k.map((l,v)=>`<li class="${v===z?"current":v<z?"past":"future"}">${a(l.label)}</li>`).join("")}
      </ol>
    `}return()=>{var b;i=!0,(b=t.streamStop)==null||b.call(t)}}function wa(n,o){let i=!1;const t=new Map;n.innerHTML=`<h1>${a(o)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const d=n.querySelector("#machine-body");$e(n,(E,D)=>{E==="toggle-section"&&j(D.dataset.section??"")}),m();async function m(){let E,D;try{const[M,R]=await Promise.all([Te(),Ie()]);E=M.find(x=>x.id===o),D=R}catch(M){if(i)return;d.innerHTML=`<p class="error">Failed to load machine: ${a(String(M))}</p>`;return}if(!i){if(!E){location.hash="#/targets";return}g(E,D)}}function g(E,D){const M=E.mode==="local"?"this machine":"SSH",R=E.mode==="ssh"&&E.ssh?`${a(E.ssh.User)}@${a(E.ssh.Host)}`:M;d.innerHTML=`
      <p class="muted">${R}</p>
      <p>${p(E,D)}</p>
      <div class="machine-sections">
        ${_.map(x=>A(x,E,D)).join("")}
      </div>
      ${he()}
    `}function p(E,D){const M=E.wire;if(!M)return q("not set up","neutral");const R=D.networks.find(h=>h.ChainID===M.ChainID),x=R?R.Name:`chain ${M.ChainID}`;return`${q(x,"ok")} ${q(M.ExecID,"neutral")} ${q(M.BeaconID,"neutral")}${M.Archive?" "+q("archive","warn"):""}`}function A(E,D,M){return`
      <section class="card machine-section" data-section-card="${a(E.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(E.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(E.title)}</span>
          <span class="machine-section-status">${E.status(D,M)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(E.key)}" hidden></div>
      </section>
    `}function j(E){const D=_.find($=>$.key===E);if(!D)return;const M=n.querySelector(`[data-section-card="${E}"]`),R=n.querySelector(`[data-section-body="${E}"]`),x=n.querySelector(`.machine-section-head[data-section="${E}"]`);if(!M||!R||!x)return;const h=R.hidden;if(h&&!t.has(E)){const $=document.createElement("div");R.appendChild($),t.set(E,D.mount($))}R.hidden=!h,M.classList.toggle("open",h),x.setAttribute("aria-expanded",String(h))}const _=[{key:"setup",title:"Setup",status:E=>E.wire?q("set up","ok"):q("not set up","neutral"),mount:E=>$a(E,o)},{key:"dashboard",title:"Dashboard",status:E=>E.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>ha(E,o)},{key:"logs",title:"Logs",status:E=>E.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:E=>fa(E,o)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:E=>ga(E,o)}];return()=>{i=!0;for(const E of t.values())try{E()}catch{}t.clear()}}function Ht(n){var t;if(!n)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const o=n.actions??[];if(n.blocked)return{tone:"blocked",label:"Unavailable",sub:n.blocked,actions:o,blocked:n.blocked};const i=((t=n.networks)==null?void 0:t.length)??0;return n.status.State==="running"?{tone:"on",label:"Running",sub:`${i} network${i===1?"":"s"} served`,actions:o}:{tone:"off",label:"Stopped",sub:i?`${i} network${i===1?"":"s"} configured`:"Press to start",actions:o}}function ot(n){if(!n.running)return"off";if(!n.serviceable)return"frequent";const o=n.slowRate??0;return o>.4?"frequent":o>=.1?"occasional":"stable"}const ka=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function Dt(n){return ka.map(({key:o,label:i,hot:t})=>{const d=n[o]==="supported";return{key:o,label:i,lit:d,hot:!!t&&d}})}function Ca(n,o){const i=n.Networks??[];return{...n,Networks:i.filter(t=>t.ChainID!==o)}}function Ta(n,o){if(n.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const i=n.filter(p=>!p.wire);if(i.length>0){const p=i.map(j=>j.id);return{level:"attention",sentence:p.length===1?"1 machine still needs setup.":`${p.length} machines still need setup.`,machines:p}}const t=o.networks??[],d=p=>{const A=t.find(j=>j.ChainID===p);return A?A.Name:`chain ${p}`},m=xa(n.map(p=>d(p.wire.ChainID))),g=n.length===1?"machine":"machines";return{level:"ok",sentence:`All ${n.length} ${g} healthy — ${Pa(m)}.`,machines:[]}}function Sa(n,o){const i=o.machines.length?` <span class="verdict-machines">${o.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${a(t)}</a>`).join(" ")}</span>`:"";n.innerHTML=`
    <div class="verdict-line verdict-${o.level}">
      ${q(o.level==="ok"?"OK":"Attention",o.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(o.sentence)}</strong>${i}
    </div>
  `}function xa(n){return[...new Set(n)]}function Pa(n){return n.length<=1?n[0]??"":n.length===2?`${n[0]} and ${n[1]}`:`${n.slice(0,-1).join(", ")} and ${n[n.length-1]}`}const Ea=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function Ct(n){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:n,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Ia=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`,fe=n=>`<svg class="p-i"><use href="#p-${n}"/></svg>`,Tt="run";function St(n){let o=null,i={name:"list"},t=null,d=null,m=null,g=null,p=[],A=null,j=null,_=!1,E=null,D=!1,M=null,R=!1,x=null;n.innerHTML=Ia+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const h=n.querySelector("#p-card");async function $(){try{const O=await it();o=Ra(O.gateways),t=null}catch(O){t=ye(O)}w()}function w(){h.innerHTML=B()}function B(){return t?La(t):i.name==="network"?qa(o,i.chainId,{caps:A,capsBusy:_,tls:E,tlsBusy:D,tlsErr:M,copyFlash:R,error:x}):i.name==="endpoint"?Wa(o,i.chainId,i.upstreamId):Aa(o,d,m,p)}async function W(O,G){_=!0,w();try{A=await Lt(O,G),j=O}catch{A=null,j=O}_=!1,w()}async function J(O,G){var pe;const U=(pe=O.networks)==null?void 0:pe.find(F=>F.chainId===G);if(await Ee({title:"Remove network",body:`Stop serving ${(U==null?void 0:U.name)??`chain ${G}`}?`,confirmLabel:"Remove",danger:!0})){x=null,w();try{await nt(O.id,Ca(O.config,G))}catch(F){x=`Could not remove the network: ${ye(F)}`,w();return}i={name:"list"},w(),await re(O.id)}}$e(h,(O,G)=>{se(O,G)});async function se(O,G){if(O==="setup"){if(d)return;await ue();return}if(O==="power"){if(!o||d)return;const U=Ht(o);if(U.tone==="blocked")return;if(o.status.State==="running"&&U.actions.includes("stop")){await K(o.id,"stop");return}if(U.actions.includes("start")){await K(o.id,"start");return}if(U.actions.includes("create")){await re(o.id);return}return}if(O==="open-network"){i={name:"network",chainId:Number(G.dataset.chainId)},x=null,E=null,M=null,w(),o&&j!==o.id&&W(o.id,!1);return}if(O==="back"){i={name:"list"},w();return}if(O==="back-to-network"){const U=Number(G.dataset.chainId);i=Number.isFinite(U)?{name:"network",chainId:U}:{name:"list"},w();return}if(O!=="add-network")switch(O){case"gw-start":case"gw-stop":case"gw-restart":o&&!d&&await K(o.id,O.slice(3));return;case"gw-create":case"gw-recreate":o&&!d&&await re(o.id);return;case"gw-wipe":o&&!d&&await ee(o);return;case"copy-url":{const U=G.dataset.url??"";if(!U)return;await De(U)&&(R=!0,w(),window.setTimeout(()=>{R=!1,w()},1200));return}case"verify-tls":{if(!o||D)return;D=!0,M=null,w();try{E=await Rt(o.id)}catch(U){M=ye(U)}D=!1,w();return}case"open-endpoint":{const U=Number(G.dataset.chainId),Y=G.dataset.upstreamId??"";if(!Number.isFinite(U)||!Y)return;i={name:"endpoint",chainId:U,upstreamId:Y},w();return}case"add-endpoint":return;case"remove-network":{if(!o||d||i.name!=="network")return;await J(o,i.chainId);return}case"recheck":{if(!o)return;await Promise.all([W(o.id,!0),$()]);return}default:return}}async function K(O,G){if(!d){d=G,m=null,w();try{await At(O,G)}catch(U){m=`${G} failed: ${ye(U)}`}d=null,await $()}}async function re(O){if(d)return;d="create",m=null,w();let G;try{G=await at(O)}catch(U){m=ye(U),d=null,w();return}g==null||g(),g=Fe(G.targetId,U=>{(U.err||U.stepId===Tt&&U.done)&&(g==null||g(),g=null,d=null,U.err&&(m=`Provisioning failed: ${U.err}`),$())})}async function ue(){if(d)return;d="setup",m=null,p=[],w();const O=F=>{p=[...p,F],w()},G=(F,y)=>{d=null,m=y?`${F} — ${y}`:F,w()};O("Preparing your endpoint…");try{(await Te()).some(y=>y.id==="local")||await tt({id:"local",mode:"local"})}catch(F){G(`Could not register this machine: ${ye(F)}`,Be(F));return}try{const F=await Et("local");if(!F.docker.reachable){G(F.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",F.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(F){G(`Could not check Docker on this machine: ${ye(F)}`,Be(F));return}O("Creating the gateway…");let U="default";try{U=(await It({id:U,placement:{targetId:"local",backend:"docker"},config:Ct([])})).id}catch(F){G(`Could not create the gateway: ${ye(F)}`,Be(F));return}O("Adding Ethereum and PulseChain endpoints…");const Y=[];for(const{chainId:F}of Ea)try{const u=((await Bt(U,F)).endpoints??[]).filter(S=>!S.alreadyAdded).map(S=>S.url);if(u.length===0)continue;Y.push({ChainID:F,Upstreams:u.map((S,L)=>({ID:`public-${F}-${L+1}`,Kind:"external",Endpoint:S,Local:!1,RecentOnly:!1}))})}catch(y){G(`Could not read valve's set for chain ${F}: ${ye(y)}`,Be(y));return}if(Y.length===0){G("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await nt(U,Ct(Y))}catch(F){G(`Could not save the endpoints: ${ye(F)}`,Be(F));return}O("Starting the gateway… the first run pulls the eRPC and Caddy images.");let pe;try{pe=await at(U)}catch(F){G(`Could not start the gateway: ${ye(F)}`,Be(F));return}g==null||g(),g=Fe(pe.targetId,F=>{const y=F.err?`${F.stepId}: ${F.err}`:F.line?`${F.stepId}: ${F.line}`:`${F.stepId}: done`;O(y),(F.err||F.stepId===Tt&&F.done)&&(g==null||g(),g=null,d=null,F.err&&(m=`Provisioning failed: ${F.err}`),p=[],$())})}async function ee(O){if(await Ee({title:`Wipe ${O.label}`,body:`This destroys ${O.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){d="wipe",m=null,w();try{const U=await Nt(O.id);U.error&&(m=U.error)}catch(U){m=`wipe failed: ${ye(U)}`}d=null,await $()}}return $(),()=>{g==null||g()}}function Ra(n){return!n||n.length===0?null:n.find(o=>o.placement.targetId==="local")??n[0]}function ye(n){return n instanceof Error?n.message:String(n)}function Be(n){return n instanceof ke?n.hint:void 0}function La(n){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(n)}</div>`}function Aa(n,o,i,t){var g;if(n===null)return Na(o,i,t);const d=Ht(n),m=(g=n==null?void 0:n.networks)!=null&&g.length?n.networks.map((p,A)=>Oa(n,p,A>0)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${a(d.sub)}</span>
    </div>
    <div class="p-band">
      ${Da(n,d,o,i)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${m}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${fe("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function Na(n,o,i){const t=n==="setup",d=o?`<div class="p-emptyerr">${a(o)}</div>`:"",m=i.length?`<div class="p-setup-log" aria-live="polite">${i.map(g=>`<div>${a(g)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${t?" disabled":""}>
        <div class="p-pbtn off big${t?" busy":""}">${fe("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${d}
      ${m}
    </div>
  `}function Ba(n,o){return o.tone==="blocked"?null:n.status.State==="running"&&o.actions.includes("stop")?"stop":o.actions.includes("start")?"start":o.actions.includes("create")?"create":null}const Ha={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},xt={restart:"refresh",recreate:"refresh",wipe:"trash"};function Da(n,o,i,t){const d=o.tone==="blocked"?o.blocked??"":o.sub,m=i?" busy":"",g=t?`<div class="p-ps" style="color:var(--red)">${a(t)}</div>`:"",p=o.tone==="blocked"&&(n!=null&&n.hint)?`<div class="p-ps">${a(n.hint)}</div>`:"",A=`
    <div class="p-power${m}" data-action="power">
      <div class="p-pbtn ${o.tone}">${fe("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(o.label)}</div>
        <div class="p-ps"${o.tone==="blocked"?' style="color:var(--red)"':""}>${a(d)}</div>
        ${p}
        ${g}
      </div>
    </div>
  `,j=n?Ma(n,o,i):"";return A+j}function Ma(n,o,i){const t=Ba(n,o),d=(n.actions??[]).filter(g=>g!==t);return d.length===0?"":`<div class="p-chips">${d.map(g=>{const p=Ha[g]??g,A=xt[g]?fe(xt[g]):"";return`<button type="button" class="p-chip${g==="wipe"?" danger":""}" data-action="gw-${g}" data-gid="${a(n.id)}"${i?" disabled":""}>${A}${a(p)}</button>`}).join("")}</div>`}const Mt={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function Ua(n){return n.map(o=>`<svg class="p-i${o.hot?" hot":o.lit?" on":""}"><use href="#p-${Mt[o.key]}"/></svg>`).join("")}function Oa(n,o,i){const t=ot({running:n.status.State==="running",serviceable:o.serviceable}),d=Dt({});return`
    <div class="p-row${i?" p-rowdiv":""}" data-action="open-network" data-chain-id="${o.chainId}">
      <span class="p-lead"><span class="p-dot ${t}"></span></span>
      <span class="p-nm">${a(o.name)}</span>
      <span class="p-caps">${Ua(d)}</span>
      <span class="p-chev">${fe("chevR")}</span>
    </div>
  `}function Fa(n,o){var i;return o==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(i=(n.capabilities??[]).find(t=>t.key===o))==null?void 0:i.status}function ja(n,o,i){const t=((n==null?void 0:n.endpoints)??[]).filter(m=>m.chainId===o&&i.includes(m.upstream)),d={};for(const m of["http","ws","archive","trace"])t.some(g=>Fa(g,m)==="supported")&&(d[m]="supported");return d}function qa(n,o,i){var W;const t=(W=n==null?void 0:n.networks)==null?void 0:W.find(J=>J.chainId===o);if(!n||!t)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${fe("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${o}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;const d=n.status.State==="running",m=ot({running:d,serviceable:t.serviceable}),g=t.upstreams??[],p=i.tls??n.tls.verification??null,A=(p==null?void 0:p.ok)===!0,j=i.tlsBusy?"Verifying…":A?`Verified ${p?new Date(p.at).toLocaleString():""}`:"Verify HTTPS now",_=i.tlsErr?`<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${a(i.tlsErr)}</div>`:"",E=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${A?"green":"dim"}" data-action="verify-tls" title="${a(j)}">${fe("lock")}</span>
          <span class="p-ic ${i.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(t.url??"")}" title="Copy the gateway URL">${fe("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${a(t.url||"—")}</div>
      ${_}
    </div>
  `,D=g.map((J,se)=>{const K=ot({running:d,serviceable:!J.problem});return`
        <div class="p-row${se>0?" p-rowdiv":""}" data-action="open-endpoint" data-chain-id="${t.chainId}" data-upstream-id="${a(J.id)}">
          <span class="p-lead"><span class="p-dot ${K}"></span></span>
          <span class="p-nm">${a(J.label)}</span>
          <span class="p-chev">${fe("chevR")}</span>
        </div>
      `}).join(""),M=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${g.length}</span></div>
      ${D}
      <div class="p-row${g.length>0?" p-rowdiv":""} addr" data-action="add-endpoint">
        <span class="p-lead">${fe("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `,R=ja(i.caps,o,g.map(J=>J.id)),x=Dt(R),h=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${i.capsBusy&&!i.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':`<div class="p-caprow">${x.map(J=>`<span class="p-capitem${J.lit?" lit":""}">${fe(Mt[J.key])}${a(J.label)}</span>`).join("")}</div>`}
    </div>
  `,$=d?t.serviceable?"Healthy":"Unserviceable":"Stopped",w=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${fe("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${m}"></span> ${a($)}</span></div>
    </div>
  `,B=i.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(i.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${fe("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${m}"></span> <span class="p-nmtxt">${a(t.name)}</span></span>
    </div>
    ${E}
    ${M}
    ${h}
    ${w}
    ${B}
    <div class="p-band p-remove" data-action="remove-network">${fe("trash")} Remove network</div>
  `}function Wa(n,o,i){var m,g;const t=(m=n==null?void 0:n.networks)==null?void 0:m.find(p=>p.chainId===o),d=(g=t==null?void 0:t.upstreams)==null?void 0:g.find(p=>p.id===i);return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${o}">${fe("chevL")}</span>
      <span class="p-dtitle"><span class="p-nmtxt">${a((d==null?void 0:d.label)??"Endpoint")}</span></span>
    </div>
    <div class="p-band" style="padding:16px;color:var(--dim)">Endpoint detail is coming soon.</div>
  `}function _a(n,o){let i=!1,t=[],d=null,m=!1,g=!1;n.innerHTML=`<h1>Security: ${a(o)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${he()}</div>`;const p=n.querySelector("#sec-body"),A=n.querySelector("#sec-footer");$e(n,(R,x)=>{var h;if(R==="rerun")_();else if(R==="toggle")(h=x.closest(".check-item"))==null||h.classList.toggle("expanded");else if(R==="copy"){const $=x.dataset.copy;$&&M(x,$)}}),j();async function j(){let R,x;try{const[$,w]=await Promise.all([Te(),Ie()]);R=$.find(B=>B.id===o),x=w}catch($){if(i)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String($))}</p>`;return}if(i)return;if(!R){p.innerHTML=`<p class="error">Target "${a(o)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!R.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(o)}">Run the setup wizard →</a></p>`;return}const h=x==null?void 0:x.networks.find($=>$.ChainID===R.wire.ChainID);h&&(A.innerHTML=he(h.Name,h.LearnURL)),await _()}async function _(){m=!0,d=null,E();try{t=await _n(o),g=!0}catch(R){d=String(R instanceof Error?R.message:R)}m=!1,i||E()}function E(){p.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(o)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${m?"disabled":""}>${m?"Re-running…":"Re-run checks"}</button>
      </div>
      ${d?`<p class="error">${a(d)}</p>`:""}
      ${!g&&m?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(D).join("")}</ul>`:g?'<p class="muted">No checks returned.</p>':""}
    `}function D(R){const x=R.Status==="pass"?"ok":R.Status==="fail"?"bad":R.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${q(R.Status,x)}
          <strong>${a(R.Title)}</strong>
          <span class="muted small check-detail-inline">${a(R.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${a(R.Why)}</p>
          </details>
          ${R.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(R.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(R.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function M(R,x){const h=await De(x),$=R.textContent;R.textContent=h?"Copied!":"Copy failed",setTimeout(()=>{i||(R.textContent=$)},1500)}return()=>{i=!0}}const Ka=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],rt="VALVE_API_KEY";function Va(n){return n===rt?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(n)}}</code> slot wherever an endpoint URL carries one.`}function Ga(n){let o=!1,i=!1,t=!1,d=null,m=!1,g=null,p=null;const A=new Set,j=new Map;let _="",E="";n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${he()}`;const D=n.querySelector("#settings-body");$e(n,(w,B)=>{if(w==="save"&&$(),w==="clear-key"){if(!g)return;i=!0;const W=n.querySelector("#ai-key");W&&(W.value=""),h(g)}if(w==="clear-provider-key"){const W=B.dataset.key;if(!g||!W)return;A.add(W),j.set(W,""),m=!1,h(g)}}),ct(n,(w,B)=>{w!=="ai-provider"||!g||(p=B,m=!1,h(g))}),M();async function M(){try{const w=await sa();if(o)return;g=w,h(w)}catch(w){if(o)return;D.innerHTML=`<p class="error">Failed to load settings: ${a(String(w))}</p>`}}function R(w){const W=(Array.isArray(w.providerKeysSet)?w.providerKeysSet:[]).filter(J=>J!==rt).sort();return[rt,...W]}function x(w,B){const W=a(w);return`
      <div class="pk-row">
        <label>
          <code>${W}</code>
          <input class="provider-key" data-key="${W}" type="password" autocomplete="off"
                 placeholder="${B?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Va(w)}</p>
        ${B?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${W}">Clear saved key</button>`:""}
      </div>`}function h(w){var ue;const B=p??w.aiProvider,W=Array.isArray(w.providerKeysSet)?w.providerKeysSet:[],J=R(w).map(ee=>x(ee,W.includes(ee))).join("");D.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${st("ai-provider",Ka.map(ee=>({value:ee.value,label:ee.label})),B)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${w.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${w.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${J}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${a(_)}" />
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
            <input id="ref-rpc-base" type="text" value="${a(w.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${d?`<p class="error">${a(d)}</p>`:""}
        ${m?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const se=n.querySelector("#ai-key");se==null||se.addEventListener("input",()=>{i=!0,m=!1}),(ue=n.querySelector("#ref-rpc-base"))==null||ue.addEventListener("input",()=>{m=!1}),n.querySelectorAll("input.provider-key").forEach(ee=>{const O=ee.dataset.key;if(!O)return;const G=j.get(O);G!==void 0&&(ee.value=G),ee.addEventListener("input",()=>{A.add(O),j.set(O,ee.value),m=!1})});const K=n.querySelector("#pk-new-value");K&&(K.value=E),K==null||K.addEventListener("input",()=>{E=K.value,m=!1});const re=n.querySelector("#pk-new-name");re==null||re.addEventListener("input",()=>{_=re.value,m=!1})}async function $(){const w=n.querySelector("#ai-key"),B=n.querySelector("#ref-rpc-base");if(!w||!B||!g)return;const W={aiProvider:p??g.aiProvider,refRpcBase:B.value.trim()};i&&(W.aiKey=w.value);const J={};for(const K of A)J[K]=j.get(K)??"";const se=_.trim();se&&(J[se]=E),Object.keys(J).length>0&&(W.providerKeys=J),t=!0,d=null,m=!1,h(g);try{const K=await oa(W);if(o)return;g=K,i=!1,A.clear(),j.clear(),_="",E="",t=!1,m=!0,h(K)}catch(K){if(o)return;t=!1,d=String(K instanceof Error?K.message:K),h(g)}}return()=>{o=!0}}const za=["http","ws","archive","trace"],Ja={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},He=1337,Ya="run",Za={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Xa(n){let o=!1,i=null,t=null;const d={},m={},g={},p={},A={},j={},_={},E={},D={},M={},R={},x={},h={},$={},w={};let B="",W=null;n.innerHTML=`
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
    ${he()}
  `;const J=n.querySelector("#rpc-body");$e(n,(e,s)=>{sn(e,s)}),ct(n,()=>{}),K(),se();async function se(){try{const e=await Pt();if(o)return;B=e.os,Y()}catch{}}async function K(){try{const e=await it();if(o)return;i=e,t=null}catch(e){if(o)return;i=null,t=me(e)}Y();for(const e of(i==null?void 0:i.gateways)??[])re(e.id),ue(e.id,!1)}async function re(e){try{const s=await Qn(e);if(o)return;d[e]=s}catch{if(o)return;d[e]=null}Y()}async function ue(e,s){g[e]=s,s&&Y();try{const r=await Lt(e,s);if(o)return;m[e]=r}catch{if(o)return;m[e]=null}g[e]=!1,Y()}function ee(e){return((i==null?void 0:i.gateways)??[]).find(s=>s.id===e)}function O(e,s){return(e.networks??[]).find(r=>r.chainId===s)}function G(e,s,r){var f;const c=(((f=d[e])==null?void 0:f.networks)??[]).find(T=>T.chainId===s);return((c==null?void 0:c.upstreams)??[]).find(T=>T.upstream===r)}function U(e,s,r){var c;return(((c=m[e])==null?void 0:c.endpoints)??[]).find(f=>f.chainId===s&&f.upstream===r)}function Y(){if(o)return;if(t){J.innerHTML=`<p class="error">Could not read the gateways: ${a(t)}</p>`;return}if(!i){J.innerHTML='<p class="muted">Loading…</p>';return}const e=i.gateways??[],s=e.length>1,r=(i.targets??[]).some(T=>bt(T.id,e)),c=new Set(e.map(T=>T.placement.targetId)),f=(i.orphans??[]).filter(T=>!c.has(T.targetId));J.innerHTML=`
      ${e.map(T=>y(T,s)).join("")}
      ${e.length===0?F():""}
      ${f.map(pe).join("")}
      ${r?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function pe(e){const s=`docker rm -f ${e.containerName}`,r=h[e.containerName];return`
      <div class="strip">
        ${N({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:s})}
        ${r?N({tone:"bad",text:r}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function F(){return((i==null?void 0:i.targets)??[]).length===0?`
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
    `}function y(e,s){return`
      ${s?`<h2 class="rpc-machine">${a(e.placement.targetId)}</h2>`:""}
      ${u(e)}
      ${v(e)}
      ${ie(e)}
      ${Re(e)}
      ${b(e)}
    `}function u(e){const s=e.status.State==="running",r=e.tls,c=[`on <strong>${a(e.placement.targetId)}</strong>`];return e.status.Image&&c.push(`<code>${a(e.status.Image)}</code>`),c.push(r!=null&&r.enabled?`HTTPS front <code>${a(r.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${V(e)}
        <strong>${a(e.label)}</strong>
        ${ae(e)}
        <span class="muted small">${c.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${s?`base <code>${a(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function S(e){const s=e.tls;return s!=null&&s.enabled&&s.rootCaPath&&s.effectiveCertSource==="internal"?s.rootCaPath:null}function L(e){var s;return((s=((i==null?void 0:i.targets)??[]).find(r=>r.id===e.placement.targetId))==null?void 0:s.mode)??""}function H(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function Z(e,s,r){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${s}"`;case"windows":return`certutil -addstore -f ROOT "${s}"`;case"linux":default:return`sudo cp "${s}" /usr/local/share/ca-certificates/valve-node-app-${r}.crt && sudo update-ca-certificates`}}function b(e){const s=D[e.id]??!1,r=((i==null?void 0:i.orphans)??[]).filter(c=>c.targetId===e.placement.targetId);return`
      <section class="card manage-section${s?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(e.id)}" aria-expanded="${s}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${k(e,r.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${s?I(e,r):""}
      </section>
    `}function k(e,s){const r=[];return e.status.State!=="running"&&r.push("gateway not running"),s>0&&r.push(`${s} leftover container${s===1?"":"s"}`),r.length===0?"container, settings, certificate":r.join(" · ")}function I(e,s){var r;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(c=>ce(e,c)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${a(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${g[e.id]?"disabled":""}>
            ${g[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
          </button>
          <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${a(e.id)}">
            ${_[e.id]?"Close settings":"Settings"}
          </button>
          <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${a(e.id)}"
                  title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
        </div>
        ${e.status.State==="running"?`<div class="rpc-head-url">
                 <code class="endpoint-url">${a(e.baseUrl)}</code>
                 <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.baseUrl)}">Copy base</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${a(((r=(e.networks??[])[0])==null?void 0:r.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${a(e.baseUrl)}</code> once it is running.</p>`}
        ${z(e)}
        ${s.map(pe).join("")}
        ${_[e.id]?Zt(e):""}
      </div>
    `}function z(e){const s=S(e);if(!s)return"";const r=L(e)==="local",c=Z(B,s,e.id),f=w[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(s)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(s)}">Copy path</button>
        </div>
        ${r?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${$[e.id]?"disabled":""}>
                   ${$[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${f?l(f):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${r?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${B?` (${a(H(B))})`:""}:</span>
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
    `}function v(e){const s=[];e.error&&s.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&s.push({tone:"warn",text:e.blocked});for(const c of e.warnings??[])s.push({tone:"warn",text:c});s.push(...X(e));const r=A[e.id];return r&&s.push({tone:"bad",text:r}),s.length===0?"":`<div class="strip">${s.map(N).join("")}</div>`}function N(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${a(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${a(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.cmd)}">Copy</button>`:""}
      </div>
    `}function X(e){var f,T;const s=e.tls;if(!(s!=null&&s.enabled))return[];const r=[];s.fallback&&r.push({tone:"warn",text:s.fallback}),s.error?r.push({tone:"warn",text:`HTTPS front: ${s.error}`}):((f=s.status)==null?void 0:f.State)!=="running"&&r.push({tone:"warn",text:`The HTTPS front is ${((T=s.status)==null?void 0:T.State)??"unknown"}, so nothing answers on ${s.url??"its https URL"} even if the gateway itself is up.`,cmd:s.containerName?`docker start ${s.containerName}`:void 0});const c=M[e.id]??s.verification??null;return c&&(!c.ok||!c.subscriptionsOk)&&r.push({tone:c.ok?"warn":"bad",text:`${c.summary} Checked ${new Date(c.at).toLocaleString()} — open Settings for the full check.`}),c!=null&&c.expiryWarning&&r.push({tone:"warn",text:c.expiryWarning}),r}function ae(e){switch(e.status.State){case"running":return q("running","ok");case"created-but-stopped":return q("stopped","warn");case"not-created":return q("not created","neutral");default:return q("unknown","bad")}}function V(e){return e.status.State==="running"?Ce("ok"):e.status.State==="unknown"?Ce("bad"):Ce("neutral")}function ce(e,s){const r=Za[s];if(!r)return"";const c=p[e.id];return`
      <button class="${r.className}" data-action="gw-${s}" data-gid="${a(e.id)}"
              title="${a(r.title)}" ${c?"disabled":""}>
        ${c===s?'<span class="spinner" aria-label="working"></span>':a(r.label)}
      </button>
    `}function ie(e){const s=j[e.id]??[];return s.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(e.placement.targetId)}</p>
        <pre class="step-log">${a(s.join(`
`))}</pre>
      </div>
    `}function Re(e){const s=je(e.networks??[]),r=s.some(c=>c.chainId===He);return s.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(e.id)}">Add a network</button>
            ${lt(e,r)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${s.map(c=>qe(e,c)).join("")}
        </div>
        ${Le(e,r)}
        ${Yt(e)}
      </div>
    `}function je(e){const s=e.filter(c=>c.chainId!==He),r=e.filter(c=>c.chainId===He);return[...s,...r]}function qe(e,s){const r=Ft(s),c=s.chainId===He,f=`${e.id}:${s.chainId}`,T=E[f]??!1,P=r.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${r.tone}${c?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(s.name)}</span>
          <code class="chain-key">evm:${s.chainId}</code>
          ${c?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${q(P,r.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(f)}" aria-expanded="${T}">
              ${T?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Ye(e,s)}
        ${T?We(e,s,r):""}
      </section>
    `}function Ye(e,s){if(!s.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const r=S(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(s.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(s.url)}"
                title="Copy ${a(s.url)}">Copy URL</button>
        ${r?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${L(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${$[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${$[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(r)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${w[e.id]?`<span class="chain-cert muted small">${a(w[e.id].ok?"Trusted — reload your wallet or browser.":w[e.id].message)}</span>`:""}`:""}
      </div>
    `}function We(e,s,r){const c=s.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${r.why?" chain-verdict-why":""}"${r.why?` title="${a(r.why)}"`:""}>${r.html}</p>
        <div class="chain-detail-bar">
          ${Ot(c.length,r.tone,s.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(e.id)}" data-chain="${s.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(e.id)}" data-chain="${s.chainId}">Remove</button>
        </div>
        ${Wt(e,s)}
        ${(s.warnings??[]).map(f=>`<p class="chain-note">${a(f)}</p>`).join("")}
      </div>
    `}function Le(e,s){const r=m[e.id],c=r!=null&&r.at?`probed ${a(ut(r.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(e.id)}">+ Network</button>
        ${lt(e,s)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${c}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${g[e.id]?"disabled":""}>
          ${g[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function lt(e,s){return s?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(e.id)}"
                    title="Add a throwaway local test chain (evm:${He}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Ot(e,s,r){const c=r>0,f=c?r:e,T=Math.min(e,f);let P="";for(let Ne=0;Ne<f;Ne++)P+=`<span class="seg${Ne<T?` seg-on seg-${s}`:""}"></span>`;const C=c&&e>r,Q=c?C?`${e} (set is ${r})`:`${e} of ${r}`:`${e}`,oe=`${e} upstream${e===1?"":"s"} configured`,be=c?`${oe}${C?`, ${e-r} beyond the set`:""}. valve's set for this chain is ${r}.`:`${oe}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(be)}">${P}</span>
      <span class="segs-n">${Q}</span>
    `}function Ft(e){const s=e.upstreams??[];if(s.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!s.some(jt)){const c=qt(s);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${c.length?` — every upstream here is configured as ${c.map(T=>`<code>${a(T)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(s.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!s.some(c=>c.local))return{tone:"warn",html:"No node of your own serves this chain."};const r=s.filter(c=>!!c.problem);if(r.length>0){const c=s.length-r.length;return{tone:"warn",html:`${r.length} of these ${s.length} endpoints ${r.length===1?"is":"are"} unusable, so ${c===1?"only one can":`only ${c} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${s.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function jt(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function qt(e){const s=new Set;for(const r of e){const c=/^([a-z][a-z0-9+.-]*):\/\//i.exec((r.endpoint??"").trim());c&&s.add(c[1].toLowerCase())}return[...s].sort()}function Wt(e,s){const r=s.upstreams??[];return r.length===0?"":`<ul class="ups">${r.map(c=>_t(e,s,c)).join("")}</ul>`}function _t(e,s,r){const c=`${e.id}|${s.chainId}|${r.id}`,f=r.actions??[];return`
      <li class="up${r.problem?" up-bad":""}">
        <div class="up-what">
          ${r.problem?Ce("bad"):Ce("ok")}
          <span class="up-label">${a(r.label)}</span>
          ${Kt(r)}
        </div>
        <code class="up-url">${a(r.endpoint||"—")}</code>
        <div class="up-caps">${Vt(e,s,r)}</div>
        <div class="up-share">${Jt(e,s,r)}</div>
        <div class="up-acts">
          ${f.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${a(c)}"
                         data-target="${a(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${p[e.id]?"disabled":""}>
                   ${p[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${a(c)}">Remove</button>
        </div>
        ${r.problem?`<div class="up-problem error small">${a(r.problem)}</div>`:""}
      </li>
    `}function Kt(e){return e.problem?q("unusable","bad"):e.recentOnly?q("recent blocks","warn"):e.local?q("yours","ok"):q("public","neutral")}function dt(e,s){var r;if(e)return s==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(r=(e.capabilities??[]).find(c=>c.key===s))==null?void 0:r.status}function Vt(e,s,r){const c=U(e.id,s.chainId,r.id);return c?c.unprobeable?`<span class="caps-none" title="${a(c.unprobeable)}">not probeable from here</span>`:`<span class="caps">${za.map(f=>Gt(e,s,c,f)).join("")}</span>`:`<span class="muted small">${m[e.id]===void 0?"probing…":"—"}</span>`}function Gt(e,s,r,c){const f=(r.capabilities??[]).find(oe=>oe.key===c),T=dt(r,c)??"inconclusive",P=Ja[c]??c.toUpperCase();let C="cap";T==="unsupported"?C=zt(e,s,c)?"cap missing":"cap off":T==="inconclusive"?C="cap unknown":T==="inconsistent"&&(C="cap mixed");const Q=f!=null&&f.detail?`${f.label}: ${f.detail}`:c==="http"&&r.reachDetail?`Answers JSON-RPC over HTTP: ${r.reachDetail}`:`${P}: no verdict`;return`<span class="${C}" title="${a(Q)}">${a(P)}</span>`}function zt(e,s,r){const c=(s.upstreams??[]).map(f=>U(e.id,s.chainId,f.id)).filter(f=>!!f&&!f.unprobeable);return c.length>0&&c.every(f=>dt(f,r)==="unsupported")}function Jt(e,s,r){const c=d[e.id];if(c===void 0)return'<span class="muted small">reading…</span>';if(c===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!c.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const f=G(e.id,s.chainId,r.id),T=(c.networks??[]).find(be=>be.chainId===s.chainId);if(!f||!T||T.attributed===0)return'<span class="muted small">no traffic yet</span>';const P=Math.round(f.actual*100),C=Math.round(f.intended*100),Q=f.diverged?r.local?"warn":"":"ok",oe=`${f.succeeded.toLocaleString()} of ${T.attributed.toLocaleString()} answered requests · routing intends ${C}%`+(f.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(oe)}">
        <span class="bar">
          <span class="fill${Q?" "+Q:""}" style="width:${P}%"></span>
          <span class="tick" style="left:${C}%"></span>
        </span>
        <span class="share-n${f.diverged?" warn":""}">${P}%</span>
        ${f.unconfigured?q("not in config","warn"):""}
      </span>
    `}function Yt(e){const s=d[e.id];return s?s.enabled?s.error?`<p class="muted small">The request counters could not be read: ${a(s.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${s.since?` (${a(ut(s.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function ut(e){const s=new Date(e);return Number.isNaN(s.getTime())?e:s.toLocaleString()}function Zt(e){const s=e.config;return`
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
        ${Xt(e)}
        ${Qt(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(e.id)}">Save settings</button>
        </div>
      </div>
    `}function Xt(e){const s=!e.config.MetricsOff;return`
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
    `}function Qt(e){var P;const s=a(e.id),r=e.config.TLS??null,c=(r==null?void 0:r.Enabled)??!1,f=(r==null?void 0:r.CertSource)||"internal",T=((P=e.tls)==null?void 0:P.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${s}-tls-host" value="${a((r==null?void 0:r.Hostname)??T)}"
               placeholder="${a(T||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${T?`<p class="muted small">
               The default is <code>${a(T)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${s}-tls-port" value="${(r==null?void 0:r.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${s}-tls-source">
          <option value="internal" ${f==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${f==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${s}-tls-cert" value="${a((r==null?void 0:r.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${s}-tls-key" value="${a((r==null?void 0:r.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${en(e)}
    `}function en(e){var P,C;const s=a(e.id),r=((P=e.config.TLS)==null?void 0:P.Enabled)??!1,c=M[e.id]??((C=e.tls)==null?void 0:C.verification)??null,f=R[e.id]??!1,T=x[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${s}" ${r&&!f?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${f?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${T?`<p class="error small">${a(T)}</p>`:""}
      ${c?tn(c):""}
    `}function tn(e){const s=(e.assertions??[]).map(r=>`
          <li class="small">
            ${nn(r.status)}
            <strong>${a(r.title)}</strong>
            <div class="muted">${a(r.detail)}</div>
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
    `}function nn(e){switch(e){case"pass":return q("pass","ok");case"fail":return q("fail","bad");case"unavailable":return q("unavailable","warn");default:return q("skipped","neutral")}}async function an(e){R[e]=!0,x[e]=null,Y();try{M[e]=await Rt(e)}catch(s){x[e]=`${me(s)}${Ae(s)}`}finally{R[e]=!1,Y()}}function Se(e){return{...e.config,Networks:(e.config.Networks??[]).map(s=>({ChainID:s.ChainID,Upstreams:s.Upstreams.map(r=>({...r}))}))}}async function xe(e,s,r){A[e]=null;try{await nt(e,s)}catch(c){return A[e]=`${r?r+": ":""}${me(c)}`,Y(),!1}return await K(),!0}async function sn(e,s){const r=s.dataset.gid??"";switch(e){case"refresh":await K();return;case"copy":s.dataset.copy&&await Rn(s,s.dataset.copy);return;case"reprobe":await ue(r,!0);return;case"toggle-settings":_[r]=!_[r],Y();return;case"toggle-manage":D[r]=!D[r],Y();return;case"toggle-chain-detail":{const c=s.dataset.key??"";c&&(E[c]=!E[c]),Y();return}case"save-settings":await on(r);return;case"verify-tls":await an(r);return;case"trust-cert":await ln(r);return;case"gw-start":case"gw-stop":case"gw-restart":await dn(r,e.slice(3));return;case"gw-create":case"gw-recreate":await un(r);return;case"gw-wipe":xn(r);return;case"add-gateway":En();return;case"forget-gateway":await pn(r);return;case"dismiss-orphan":await hn(s.dataset.name??"");return;case"add-chain":fn(r);return;case"add-devnet":{const c=ee(r);if(c){const f=((i==null?void 0:i.targets)??[]).some(T=>T.id===c.placement.targetId&&T.hasDevnet);ht(r,He,f)}return}case"remove-chain":await yn(r,Number.parseInt(s.dataset.chain??"",10));return;case"add-endpoint":mt(r,Number.parseInt(s.dataset.chain??"",10));return;case"remove-endpoint":await gn(s.dataset.key??"");return;case"reset-devnet":await Tn(s.dataset.key??"",s.dataset.target??"");return;default:return}}async function on(e){const s=ee(e);if(!s)return;const r=Se(s),c=n.querySelector(`#gw-${CSS.escape(e)}-port`),f=n.querySelector(`#gw-${CSS.escape(e)}-bind`);if(c){const C=Number.parseInt(c.value.trim(),10);Number.isFinite(C)&&(r.Port=C)}f&&(r.BindAddr=f.value.trim());const T=n.querySelector(`#gw-${CSS.escape(e)}-metrics`);T&&(r.MetricsOff=!T.checked),r.TLS=rn(e,s);const P=s.status.State==="running";await xe(e,r,"Saving settings")&&(_[e]=!1,P&&(A[e]=null,cn(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),Y())}function rn(e,s){var T,P,C,Q,oe,be,Ne;const r=Ln=>n.querySelector(`#gw-${CSS.escape(e)}-${Ln}`),c=r("tls");if(!c)return s.config.TLS??null;const f=Number.parseInt(((T=r("tls-port"))==null?void 0:T.value.trim())??"",10);return{Enabled:c.checked,Hostname:((P=r("tls-host"))==null?void 0:P.value.trim())??"",CertSource:((C=r("tls-source"))==null?void 0:C.value)??"internal",CertFile:((Q=r("tls-cert"))==null?void 0:Q.value.trim())??"",KeyFile:((oe=r("tls-key"))==null?void 0:oe.value.trim())??"",HTTPSPort:Number.isFinite(f)?f:443,BindAddr:((be=s.config.TLS)==null?void 0:be.BindAddr)??"",ImageRef:((Ne=s.config.TLS)==null?void 0:Ne.ImageRef)??""}}function cn(e,s){j[e]=[s]}async function ln(e){if(!$[e]){$[e]=!0,w[e]=null,Y();try{w[e]=await na(e)}catch(s){w[e]={ok:!1,message:`${me(s)}${Ae(s)}`}}$[e]=!1,Y()}}async function dn(e,s){if(!p[e]){p[e]=s,A[e]=null,Y();try{await At(e,s)}catch(r){A[e]=`${s} failed: ${me(r)}${Ae(r)}`}p[e]=null,await K()}}async function un(e){if(p[e])return;p[e]="create",A[e]=null,j[e]=["starting…"],Y();let s;try{s=await at(e)}catch(r){A[e]=`${me(r)}${Ae(r)}`,j[e]=[],p[e]=null,Y();return}W==null||W(),W=Fe(s.targetId,r=>{if(o)return;const c=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(j[e]=[...(j[e]??[]).filter(T=>T!=="starting…"),c],!!r.err||r.stepId===Ya&&!!r.done){W==null||W(),W=null,p[e]=null,r.err&&(A[e]="Provisioning failed — see the log below."),K();return}Y()})}async function pn(e){const s=ee(e);if(!(!s||!await Ee({title:`Forget ${s.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${s.containerName}" on ${s.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await ta(e)}catch(c){A[e]=me(c),Y();return}await K()}}async function hn(e){if(e){h[e]=null;try{await Xn(e)}catch(s){h[e]=me(s),Y();return}await K()}}function fn(e){const s=ee(e);if(!s)return;const r=new Set((s.networks??[]).map(C=>C.chainId)),c=(i==null?void 0:i.presets)??[],f=c.filter(C=>!r.has(C.chainId)),T=c.filter(C=>r.has(C.chainId)),P=((i==null?void 0:i.targets)??[]).some(C=>C.id===s.placement.targetId&&C.hasDevnet);le(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${a(s.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${f.map(C=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${C.chainId}">
                <span>${a(C.name)}</span>
                <span class="muted small">chain ${C.chainId}${C.devnet?P?" · uses the devnet on "+a(s.placement.targetId):" · will create a devnet on "+a(s.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${T.length?`<p class="muted small">Already fronted: ${a(T.map(C=>C.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,C=>{if(C==="cancel"){ne();return}if(C==="custom"){mn(e);return}if(C.startsWith("preset:")){const Q=Number.parseInt(C.slice(7),10),oe=c.find(be=>be.chainId===Q);ne(),oe!=null&&oe.devnet?ht(e,Q,P):pt(e,Q)}})}function mn(e){var s;le(`
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
      `,r=>{if(r==="cancel"){ne();return}if(r!=="add")return;const c=document.getElementById("custom-chain-id"),f=document.getElementById("custom-chain-err"),T=Number.parseInt((c==null?void 0:c.value.trim())??"",10);if(!Number.isFinite(T)||T<=0){f&&(f.className="error small"),f&&(f.textContent="A chain id is a positive whole number.");return}ne(),pt(e,T)}),(s=document.getElementById("custom-chain-id"))==null||s.focus()}async function pt(e,s){const r=ee(e);if(!r)return;const c=Se(r),f=c.Networks??[];f.some(T=>T.ChainID===s)||(f.push({ChainID:s,Upstreams:[]}),c.Networks=f,await bn(e,c)&&(Y(),mt(e,s)))}async function bn(e,s){var T;const r={...s,Networks:(s.Networks??[]).filter(P=>P.Upstreams.length>0)};if(!await xe(e,r))return!1;const f=ee(e);if(f)for(const P of s.Networks??[])P.Upstreams.length===0&&!(f.networks??[]).some(C=>C.chainId===P.ChainID)&&(f.config.Networks=[...f.config.Networks??[],{ChainID:P.ChainID,Upstreams:[]}],f.networks=[...f.networks??[],{chainId:P.ChainID,name:((T=((i==null?void 0:i.presets)??[]).find(C=>C.chainId===P.ChainID))==null?void 0:T.name)??`Chain ${P.ChainID}`,path:`/${f.config.ProjectID}/evm/${P.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function ht(e,s,r){const c=ee(e);if(!c)return;if(!r){le(`
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
        `,()=>ne());return}const f=Se(c),T=f.Networks??[],P={ID:"devnet",Kind:"managed-devnet",TargetID:c.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},C=T.find(Q=>Q.ChainID===s);C?C.Upstreams.push(P):T.push({ChainID:s,Upstreams:[P]}),f.Networks=T,await xe(e,f,"Adding the devnet")}async function yn(e,s){const r=ee(e);if(!r||!Number.isFinite(s))return;const c=O(r,s);if(!await Ee({title:`Remove ${(c==null?void 0:c.name)??`chain ${s}`}`,body:`This gateway will stop serving ${(c==null?void 0:c.path)??`chain ${s}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const T=Se(r);T.Networks=(T.Networks??[]).filter(P=>P.ChainID!==s),await xe(e,T,"Removing the network")}function ft(e){const s=e.split("|");return s.length!==3?null:{gid:s[0],chainId:Number.parseInt(s[1],10),upstreamId:s[2]}}async function gn(e){const s=ft(e);if(!s)return;const r=ee(s.gid);if(!r)return;const c=Se(r),f=(c.Networks??[]).find(C=>C.ChainID===s.chainId);if(!f)return;const T=f.Upstreams.findIndex((C,Q)=>(C.ID||`${s.chainId}-${Q}`)===s.upstreamId);T<0||!await Ee({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(f.Upstreams.splice(T,1),await xe(s.gid,c,"Removing the endpoint"))}function mt(e,s){const r=ee(e);if(!r||!Number.isFinite(s))return;const c=((i==null?void 0:i.sources)??[]).filter(C=>C.chainId===s),f=O(r,s),T=new Set(((f==null?void 0:f.upstreams)??[]).filter(C=>C.kind!=="external").map(C=>`${C.kind}|${C.targetId??""}`)),P=c.filter(C=>!T.has(`${C.kind}|${C.targetId}`));le(`
        <h2>Add an endpoint for ${a((f==null?void 0:f.name)??`chain ${s}`)}</h2>
        ${P.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${P.map(C=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${a(C.kind)}:${a(C.targetId)}">
                       <span>${a(C.label)}</span>
                       <span class="muted small">${a(C.endpoint)}</span>
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
      `,C=>{if(C==="cancel"){ne();return}if(C==="known-set"){wn(e,s);return}if(C==="manual"){Cn(e,s);return}if(C.startsWith("source:")){const[,Q,oe]=C.split(":");ne(),vn(e,s,Q,oe)}})}async function vn(e,s,r,c){const f=ee(e);if(!f)return;const T=Se(f),P=T.Networks??[],C={ID:`${r==="managed-devnet"?"devnet":"node"}-${c}`,Kind:r,TargetID:c,Endpoint:"",Local:!0,RecentOnly:!1},Q=P.find(oe=>oe.ChainID===s);Q?Q.Upstreams.push(C):P.push({ChainID:s,Upstreams:[C]}),T.Networks=P,await xe(e,T,"Adding the endpoint")}function $n(e){const s=[...e].sort((f,T)=>(f.latencyMs??1e9)-(T.latencyMs??1e9)),r=s.slice(0,3),c=s.find(f=>f.url.startsWith("wss://")||f.url.startsWith("ws://"));return c&&!r.some(f=>f.url===c.url)&&(r.length===3&&r.pop(),r.push(c)),new Set(r.map(f=>f.url))}async function wn(e,s){let r;try{r=await Bt(e,s)}catch(C){le(`<h2>Endpoints for chain ${s}</h2>
         <p class="error small">Could not read the set: ${a(me(C))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>ne());return}if(o)return;const c=r.endpoints??[],f=c.filter(C=>!C.alreadyAdded).map(C=>C.url),T=new Set(c.map(C=>C.provider)).size,P=c.map(C=>{const Q=[C.websocket?'<span class="t ws">websocket</span>':"",C.archive?'<span class="t ar">archive</span>':"",C.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(C.url)}</code>
                  <span class="muted small">${a(C.provider)}</span> ${Q}</li>`}).join("");le(`<h2>Endpoints for chain ${s}</h2>
       ${c.length?`<p class="muted small">${T} providers valve has measured, in the order the gateway
                should prefer them — ${c.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${P}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${r.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${f.length?"":" disabled"}>
           ${f.length?`Add ${f.length}`:"Nothing to add"}</button>
       </div>`,C=>{ne(),C==="add"&&Ze(e,s,f),C==="discover"&&kn(e,s)})}async function kn(e,s){le(`
        <h2>Public endpoints for chain ${s}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,P=>{P==="cancel"&&ne()});let r;try{r=await aa(s)}catch(P){const C=Oe();if(C){const Q=document.createElement("p");Q.className="error small",Q.textContent=`Could not discover endpoints: ${me(P)}`,C.appendChild(Q)}return}if(o)return;const c=(r.endpoints??[]).filter(P=>P.status==="live"||P.status==="unprobed"),f=(r.endpoints??[]).filter(P=>P.status==="rejected"),T=$n(c);le(`
        <h2>Public endpoints for chain ${s}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${a(r.fetchError)}</div>`:""}</div>`:""}
        ${c.length?`<p class="muted small">${c.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${c.map(P=>{const C=T.has(P.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${a(P.url)}"${C}>
                       <span><code>${a(P.url)}</code></span>
                       <span class="muted small">${P.status==="live"?`answered in ${P.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${s} right now.</p>`}
        ${f.length?`<details class="rpc-rejected">
                 <summary class="muted small">${f.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${f.map(P=>`<li class="muted small"><code>${a(P.url)}</code> — ${a(P.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${c.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,P=>{if(P==="cancel"){ne();return}if(P==="add"){const C=Oe(),Q=C?Array.from(C.querySelectorAll('input[type="checkbox"]:checked')).map(oe=>oe.value):[];ne(),Ze(e,s,Q);return}})}function Cn(e,s){var r;le(`
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
      `,c=>{if(c==="cancel"){ne();return}if(c!=="add")return;const f=document.getElementById("manual-endpoint"),T=document.getElementById("manual-recent"),P=document.getElementById("manual-err"),C=(f==null?void 0:f.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(C)){P&&(P.className="error small",P.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}ne(),Ze(e,s,[C],(T==null?void 0:T.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Ze(e,s,r,c=!1){if(!r.length)return;const f=ee(e);if(!f)return;const T=Se(f),P=T.Networks??[];let C=P.find(oe=>oe.ChainID===s);C||(C={ChainID:s,Upstreams:[]},P.push(C));let Q=1;for(const oe of C.Upstreams){const be=/^public-\d+-(\d+)$/.exec(oe.ID??"");be&&(Q=Math.max(Q,Number(be[1])+1))}for(const oe of r)C.Upstreams.some(be=>be.Endpoint===oe)||C.Upstreams.push({ID:`public-${s}-${Q++}`,Kind:"external",Endpoint:oe,Local:!1,RecentOnly:c});T.Networks=P,await xe(e,T,r.length===1?"Adding the endpoint":`Adding ${r.length} endpoints`)}async function Tn(e,s){const r=ft(e);if(!r||!s||!await Ee({title:"Reset this devnet",body:`The chain on ${s} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;p[r.gid]="reset",A[r.gid]=null,Y();let f;try{f=await Yn(s)}catch(T){A[r.gid]=`Reset failed: ${me(T)}${Ae(T)}`,p[r.gid]=null,Y();return}p[r.gid]=null,Sn(s,f),await K()}function Sn(e,s){const r=[];r.push(s.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),s.report.Recreated&&r.push("A fresh chain was started from genesis.");const c=s.report.Cascaded??[],f=s.report.CascadeSkipped??[];le(`
        <h2>Devnet on ${a(e)} reset</h2>
        <ul class="plain-list">${r.map(T=>`<li>${a(T)}</li>`).join("")}</ul>
        ${c.length?`<p class="ok">Restarted in front of it: ${a(c.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${f.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(f.join(", "))}.</p>`:""}
        ${s.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(s.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>ne())}function xn(e){const s=ee(e);if(!s)return;le(`
        <h2>Wipe ${a(s.label)}</h2>
        <p class="error">This destroys ${a(s.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(e)}</button>
        </div>
      `,f=>{if(f==="cancel"||f==="close"){ne(),K();return}f==="confirm"&&Pn(e)});const r=document.getElementById("wipe-confirm-input"),c=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{c&&(c.disabled=r.value.trim()!==e)}),r==null||r.focus()}async function Pn(e){const s=document.getElementById("wipe-confirm-btn");s&&(s.disabled=!0,s.textContent="Wiping…");let r;try{r=await Nt(e)}catch(c){const f=Oe();if(f){const T=document.createElement("p");T.className="error small",T.textContent=`Wipe failed: ${me(c)}${Ae(c)}`,f.appendChild(T)}s&&(s.disabled=!1,s.textContent=`Wipe ${e}`);return}le(`
        <h2>${a(e)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${a(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{ne(),K()})}function bt(e,s){return!s.some(r=>{var c;return((c=r.placement)==null?void 0:c.targetId)===e})}function En(){var T;const e=(i==null?void 0:i.targets)??[],s=(i==null?void 0:i.gateways)??[],r=e.filter(P=>bt(P.id,s)),c=new Set(s.map(P=>P.id));if(e.length===0){le(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>ne());return}if(r.length===0){le(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>ne());return}const f=c.has("default")?"":"default";le(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${a(f)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${r.map(P=>`<option value="${a(P.id)}">${a(P.id)} (${a(P.mode)})</option>`).join("")}
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
      `,P=>{if(P==="cancel"){ne();return}P==="create"&&In()}),(T=document.getElementById("new-gw-id"))==null||T.focus()}async function In(){const e=document.getElementById("new-gw-id"),s=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),c=document.getElementById("new-gw-err"),f=(e==null?void 0:e.value.trim())??"",T=(s==null?void 0:s.value)??"",P=Number.parseInt((r==null?void 0:r.value.trim())??"",10),C=Q=>{c&&(c.className="error small",c.textContent=Q)};if(!f){C("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!T){C("Pick the machine it runs on.");return}try{await It({id:f,placement:{targetId:T,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(P)?P:4e3,Networks:[]}})}catch(Q){C(me(Q));return}ne(),await K()}async function Rn(e,s){const r=await De(s),c=e.textContent;e.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{o||(e.textContent=c)},1500)}function me(e){return e instanceof Error?e.message:String(e)}function Ae(e){return e instanceof ke&&e.hint?` — ${e.hint}`:""}return()=>{o=!0,W==null||W(),ne()}}const Qa="local";function es(n){let o=!1,i=!1,t="",d=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${he()}
  `;const m=n.querySelector("#targets-body");$e(n,(h,$)=>{_(h,$)}),g();async function g(){try{const[h,$,w]=await Promise.all([Te(),Ie(),Pt()]);if(o)return;t=w.os,A(h,$)}catch(h){if(o)return;m.innerHTML=`<p class="error">Failed to load machines: ${a(String(h))}</p>`}}function p(){d&&A(d.targets,d.catalog)}function A(h,$){d={targets:h,catalog:$};const w=t==="linux",B=[...h].sort((K,re)=>(K.mode==="local"?-1:0)-(re.mode==="local"?-1:0)),W=B.length?`<div class="card-grid">${B.map(K=>ts(K,$,K.mode!=="local"||w,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',J=h.some(K=>K.mode==="local");m.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${W}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(w,J)}
        ${i?ns():""}
      </section>
    `;const se=m.querySelector("#fleet-verdict");se&&Sa(se,Ta(h,$))}function j(h,$){const w=`
      <div class="card">
        <h3>A server over SSH ${q("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${h?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${h?" btn-ghost":""}" data-action="toggle-ssh">
            ${i?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,B=h?`
        <div class="card">
          <h3>This machine ${q("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${a(t)})`:""} ${q("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return $?`<div class="card-grid card-grid-wide">${w}</div>`:`<div class="card-grid card-grid-wide">${h?B+w:w+B}</div>`}async function _(h,$){var w;if(h==="add-local"){await E();return}if(h==="delete-target"){const B=$.dataset.id;if(!B||!await Ee({title:"Remove machine",body:`Remove "${B}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await D(B);return}if(h==="toggle-ssh"){i=!i,x(),p(),i&&((w=n.querySelector("#ssh-host"))==null||w.focus());return}h==="add-ssh"&&await M()}async function E(){x();try{await tt({id:Qa,mode:"local"}),await g()}catch(h){R(h)}}async function D(h){try{await Bn(h),await g()}catch($){R($)}}async function M(){const h=n.querySelector("#ssh-host"),$=n.querySelector("#ssh-user"),w=n.querySelector("#ssh-key"),B=n.querySelector("#ssh-port"),W=n.querySelector("#ssh-id");if(!h||!$||!w||!B||!W)return;const J=h.value.trim(),se=$.value.trim(),K=w.value.trim(),re=B.value.trim(),ue=W.value.trim();if(x(),!J||!se||!K){R(new Error("host, user, and key path are required"));return}const ee=ue||as(J),O={Host:J,User:se,KeyPath:K};if(re){const U=Number.parseInt(re,10);if(!Number.isFinite(U)||U<=0){R(new Error("port must be a positive number"));return}O.Port=U}const G=n.querySelector("#ssh-submit");G&&(G.disabled=!0,G.textContent="Connecting…");try{await tt({id:ee,mode:"ssh",ssh:O}),i=!1,await g()}catch(U){R(U),G&&(G.disabled=!1,G.textContent="Add server")}}function R(h){let $=n.querySelector("#targets-error");$||(m.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=n.querySelector("#targets-error")),$.textContent=String(h instanceof Error?h.message:h)}function x(){var h;(h=n.querySelector("#targets-error"))==null||h.remove()}return()=>{o=!0}}function ts(n,o,i,t){const d=n.wire,m=n.mode==="local"?"this machine":"SSH",g=n.mode==="ssh"&&n.ssh?`${a(n.ssh.User)}@${a(n.ssh.Host)}`:m;let p;if(!d&&!i)p=`${q("can't run a node","warn")} ${q(t||"not Linux","neutral")}`;else if(!d)p=q("not set up","neutral");else{const A=o.networks.find(_=>_.ChainID===d.ChainID),j=A?A.Name:`chain ${d.ChainID}`;p=`${q(j,"ok")} ${q(d.ExecID,"neutral")} ${q(d.BeaconID,"neutral")}${d.Archive?" "+q("archive","warn"):""}`}return`
    <div class="card">
      <h2>${a(n.id)}</h2>
      <p class="muted">${g}</p>
      <p>${p}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(n.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(n.id)}">Remove</button>
      </div>
    </div>
  `}function ns(){return`
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
  `}function as(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ss=document.querySelector("#app"),{contentEl:os,setActiveNav:rs}=ra(ss);let ge=null;function is(){const o=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(o.length===0)return{screen:"home"};const[i,t]=o;return i==="machine"||i==="setup"||i==="dash"||i==="logs"||i==="security"||i==="diag"||i==="services"||i==="analytics"?{screen:i,id:t?decodeURIComponent(t):void 0}:{screen:i??"targets"}}function we(n){const o=document.createElement("div");return os.replaceChildren(o),n(o)}function Ut(){if(ge){try{ge()}catch{}ge=null}const{screen:n,id:o}=is();switch(rs(n),n){case"machine":if(!o){location.hash="#/targets";return}ge=we(i=>wa(i,o));break;case"setup":case"dash":case"logs":case"services":if(!o){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(o)}`;return;case"security":if(!o){location.hash="#/targets";return}ge=we(i=>_a(i,o));break;case"diag":if(!o){location.hash="#/targets";return}ge=we(i=>ua(i,o));break;case"analytics":if(!o){location.hash="#/rpc";return}ge=we(i=>da(i,o));break;case"rpc":ge=we(i=>Xa(i));break;case"settings":ge=we(i=>Ga(i));break;case"targets":ge=we(i=>es(i));break;case"panel":ge=we(i=>St(i));break;case"home":default:ge=we(i=>St(i));break}}window.addEventListener("hashchange",Ut);Ut();
