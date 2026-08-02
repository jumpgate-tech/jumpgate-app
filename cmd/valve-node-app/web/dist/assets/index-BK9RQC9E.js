var Hn=Object.defineProperty;var Un=(n,s,r)=>s in n?Hn(n,s,{enumerable:!0,configurable:!0,writable:!0,value:r}):n[s]=r;var Ke=(n,s,r)=>Un(n,typeof s!="symbol"?s+"":s,r);(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))t(l);new MutationObserver(l=>{for(const p of l)if(p.type==="childList")for(const $ of p.addedNodes)$.tagName==="LINK"&&$.rel==="modulepreload"&&t($)}).observe(document,{childList:!0,subtree:!0});function r(l){const p={};return l.integrity&&(p.integrity=l.integrity),l.referrerPolicy&&(p.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?p.credentials="include":l.crossOrigin==="anonymous"?p.credentials="omit":p.credentials="same-origin",p}function t(l){if(l.ep)return;l.ep=!0;const p=r(l);fetch(l.href,p)}})();function Nt(){return ee("/api/host")}function Ie(){return ee("/api/catalog")}function Ee(){return ee("/api/targets")}function st(n){return ee("/api/targets",{method:"POST",headers:$e,body:JSON.stringify(n)})}function Mn(n){return ee(`/api/targets/${encodeURIComponent(n)}`,{method:"DELETE"})}function On(n,s){return ee(`/api/targets/${encodeURIComponent(n)}/disk?path=${encodeURIComponent(s)}`)}function Fn(n,s){return ee(`/api/targets/${encodeURIComponent(n)}/setup`,{method:"POST",headers:$e,body:JSON.stringify(s)})}function qe(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/setup/stream`);return r.onmessage=t=>{try{s(JSON.parse(t.data))}catch{}},()=>r.close()}function jn(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/monitor/stream`);return r.onmessage=t=>{try{s(JSON.parse(t.data))}catch{}},()=>r.close()}function qn(n,s=200){return ee(`/api/targets/${encodeURIComponent(n)}/logs?n=${s}`)}function Wn(n,s){const r=new EventSource(`/api/targets/${encodeURIComponent(n)}/logs/stream`);return r.onmessage=t=>{try{s(JSON.parse(t.data))}catch{}},()=>r.close()}function wt(n,s){const r=s===void 0?{}:{lines:s};return ee(`/api/targets/${encodeURIComponent(n)}/explain`,{method:"POST",headers:$e,body:JSON.stringify(r)})}function _n(n,s,r){return ee(`/api/targets/${encodeURIComponent(n)}/services/${s}/${r}`,{method:"POST"})}function Kn(n,s){return ee(`/api/targets/${encodeURIComponent(n)}/services/${s}/clear`,{method:"POST",headers:$e,body:JSON.stringify({Confirm:s})})}function Vn(n){return ee(`/api/targets/${encodeURIComponent(n)}/du`)}function Gn(n){return ee(`/api/targets/${encodeURIComponent(n)}/endpoints`)}function zn(n){return ee(`/api/targets/${encodeURIComponent(n)}/firewall`)}function Jn(n){return ee(`/api/targets/${encodeURIComponent(n)}/diagnostics`)}function Yn(n){return ee(`/api/targets/${encodeURIComponent(n)}/diagnostics/latest`)}function At(n){return ee(`/api/targets/${encodeURIComponent(n)}/containers`)}function Zn(n,s,r){return ee(`/api/targets/${encodeURIComponent(n)}/containers/${s}/${r}`,{method:"POST"})}async function Xn(n,s){const r=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/${s}/wipe`,{method:"POST",headers:$e,body:JSON.stringify({Confirm:s})}),t=await r.text();let l=null;try{l=t?JSON.parse(t):null}catch{}if(l&&typeof l=="object"&&"report"in l)return l;const p=l&&typeof l=="object"&&typeof l.error=="string"?l.error:r.statusText||`HTTP ${r.status}`;throw new Ce(r.status,p)}function Qn(n,s){return ee(`/api/targets/${encodeURIComponent(n)}/containers/${s}/provision`,{method:"POST"})}async function ea(n){const s=await fetch(`/api/targets/${encodeURIComponent(n)}/containers/devnet/reset`,{method:"POST",headers:$e}),r=await s.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const l=t&&typeof t=="object"&&typeof t.error=="string"?t.error:s.statusText||`HTTP ${s.status}`;throw new Ce(s.status,l)}function ta(n,s,r){return ee(`/api/targets/${encodeURIComponent(n)}/containers/${s}/config`,{method:"PUT",headers:$e,body:JSON.stringify(r)})}function lt(){return ee("/api/gateways")}async function na(n){await ee(`/api/orphans/${encodeURIComponent(n)}`,{method:"DELETE"})}function Bt(n){return ee("/api/gateways",{method:"POST",headers:$e,body:JSON.stringify(n)})}function Dt(n){return ee(`/api/gateways/${encodeURIComponent(n)}/tls/verify`)}function aa(n){return ee(`/api/gateways/${encodeURIComponent(n)}/traffic`)}function Ht(n){return ee(`/api/gateways/${encodeURIComponent(n)}/analytics`)}function Ut(n,s=!1){const r=s?"?refresh=1":"";return ee(`/api/gateways/${encodeURIComponent(n)}/capabilities${r}`)}function sa(n){return ee(`/api/gateways/${encodeURIComponent(n)}`,{method:"DELETE"})}function Se(n,s){return ee(`/api/gateways/${encodeURIComponent(n)}/config`,{method:"PUT",headers:$e,body:JSON.stringify(s)})}function Mt(n,s){return ee(`/api/gateways/${encodeURIComponent(n)}/${s}`,{method:"POST"})}function oa(n){return ee(`/api/gateways/${encodeURIComponent(n)}/trust-cert`,{method:"POST"})}function ot(n){return ee(`/api/gateways/${encodeURIComponent(n)}/provision`,{method:"POST"})}async function Ot(n){const s=await fetch(`/api/gateways/${encodeURIComponent(n)}/wipe`,{method:"POST",headers:$e,body:JSON.stringify({Confirm:n})}),r=await s.text();let t=null;try{t=r?JSON.parse(r):null}catch{}if(t&&typeof t=="object"&&"report"in t)return t;const l=t&&typeof t=="object"&&typeof t.error=="string"?t.error:s.statusText||`HTTP ${s.status}`;throw new Ce(s.status,l)}function ra(n){return ee(`/api/chainlist/${n}`)}function rt(n,s){return ee(`/api/gateways/${encodeURIComponent(n)}/knownset/${s}`)}function ia(){return ee("/api/settings")}function ca(n){return ee("/api/settings",{method:"PUT",headers:$e,body:JSON.stringify(n)})}class Ce extends Error{constructor(r,t,l,p){super(t);Ke(this,"status");Ke(this,"hint");Ke(this,"code");this.name="ApiError",this.status=r,this.hint=l,this.code=p}}const $e={"Content-Type":"application/json"};async function ee(n,s){const r=await fetch(n,s);if(!r.ok){let l=r.statusText||`HTTP ${r.status}`,p,$;try{const f=await r.json();f&&typeof f.error=="string"&&f.error&&(l=f.error),f&&typeof f.hint=="string"&&f.hint&&(p=f.hint),f&&typeof f.code=="string"&&f.code&&($=f.code)}catch{}throw new Ce(r.status,l,p,$)}if(r.status===204)return;const t=await r.text();return t?JSON.parse(t):void 0}const kt="https://learn.valve.city/rpc";function a(n){return n.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function he(n,s){const r=n&&s&&s!==kt?` <span class="footer-sep">·</span> <a href="${a(s)}" target="_blank" rel="noopener noreferrer">${a(n)}</a>`:"";return`
    <footer class="footer">
      <a href="${a(kt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${r}
    </footer>
  `}function la(n){n.innerHTML=`
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
  `;const s=n.querySelector("#content"),r=Array.from(n.querySelectorAll("[data-nav]"));return{contentEl:s,setActiveNav:l=>{const p=l==="machine"?"targets":l==="home"||l==="panel"?"rpc":l;for(const $ of r)$.classList.toggle("active",$.dataset.nav===p)}}}function de(n){return Number.isFinite(n)?n.toLocaleString("en-US"):"—"}function da(n){return Number.isFinite(n)?`${n.toFixed(1)}%`:"—"}function ua(n){if(!Number.isFinite(n)||n<0)return"—";if(n<60)return`~${Math.round(n)}s`;const s=Math.round(n/60),r=Math.floor(s/60),t=s%60;if(r===0)return`~${t}m`;if(r<48)return`~${r}h ${t}m`;const l=Math.floor(r/24),p=r%24;return`~${l}d ${p}h`}function K(n,s){return`<span class="badge badge-${s}">${a(n)}</span>`}function xe(n){return`<span class="dot dot-${n}"></span>`}const Ct=["B","KB","MB","GB","TB","PB"];function Le(n){if(!Number.isFinite(n)||n<0)return"—";if(n===0)return"0 B";let s=n,r=0;for(;s>=1024&&r<Ct.length-1;)s/=1024,r++;const t=s<10?2:s<100?1:0;return`${s.toFixed(t)} ${Ct[r]}`}async function Oe(n){try{return await navigator.clipboard.writeText(n),!0}catch{return!1}}function we(n,s){n.addEventListener("click",r=>{const t=r.target.closest("[data-action]");if(!t||!n.contains(t))return;const l=t.dataset.action;l&&s(l,t,r)})}function it(n,s,r){const t=s.find(p=>p.value===r),l=s.map(p=>`
      <li class="dropdown-option${p.value===r?" selected":""}" role="option"
          aria-selected="${p.value===r}" data-value="${a(p.value)}">
        ${a(p.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${a(n)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${a(t?t.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${l}</ul>
    </div>
  `}function Fe(n){n.querySelectorAll(".dropdown.open").forEach(s=>{var r;s.classList.remove("open"),(r=s.querySelector(".dropdown-trigger"))==null||r.setAttribute("aria-expanded","false")})}function dt(n,s){n.addEventListener("click",l=>{const p=l.target,$=p.closest(".dropdown-trigger");if($&&n.contains($)){const N=$.closest(".dropdown"),O=!!N&&!N.classList.contains("open");Fe(n),N&&O&&(N.classList.add("open"),$.setAttribute("aria-expanded","true"));return}const f=p.closest(".dropdown-option");if(f&&n.contains(f)){const N=f.closest(".dropdown");Fe(n),s((N==null?void 0:N.dataset.dropdown)??"",f.dataset.value??"");return}Fe(n)});const r=l=>{if(!n.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}const p=l.target;(!p.closest(".dropdown")||!n.contains(p))&&Fe(n)},t=l=>{if(!n.isConnected){document.removeEventListener("click",r),document.removeEventListener("keydown",t);return}l.key==="Escape"&&Fe(n)};document.addEventListener("click",r),document.addEventListener("keydown",t)}const Ze="app-modal";let Je=null;function re(n,s){Y();const r=document.createElement("div");r.className="modal-overlay",r.id=Ze,r.innerHTML=`<div class="modal">${n}</div>`,r.addEventListener("click",l=>{const p=l.target.closest("[data-modal-action]");p!=null&&p.dataset.modalAction?s(p.dataset.modalAction):l.target===r&&s("cancel")});const t=l=>{l.key==="Escape"&&s("cancel")};document.addEventListener("keydown",t),Je=t,document.body.appendChild(r)}function Y(){var n;(n=document.getElementById(Ze))==null||n.remove(),Je&&(document.removeEventListener("keydown",Je),Je=null)}function Me(){return document.querySelector(`#${Ze} .modal`)}function Te(n){return new Promise(s=>{var l;let r=!1;const t=p=>{r||(r=!0,Y(),s(p))};re(`
        <h2>${a(n.title)}</h2>
        <p>${a(n.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${n.danger?" btn-danger":""}" data-modal-action="confirm">${a(n.confirmLabel)}</button>
        </div>
      `,p=>t(p==="confirm")),(l=document.querySelector(`#${Ze} [data-modal-action="confirm"]`))==null||l.focus()})}const et=5e3,pa=60;function ha(n,s){let r=!1,t=null,l=null,p=null,$=null;const f=[];n.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${he()}</div>`;const N=n.querySelector("#an-body");we(n,(C,h)=>{var E;C==="toggle-endpoint"&&((E=h.closest(".an-endpoint"))==null||E.classList.toggle("expanded"))}),O();async function O(){try{t=((await lt()).gateways??[]).find(h=>h.id===s)??null}catch(C){if(r)return;p=String(C instanceof Error?C.message:C),j();return}if(!r){if(!t){j();return}await _(),$=window.setInterval(()=>void _(),et)}}async function _(){try{const C=await Ht(s);if(r)return;A(C),l=C,p=null}catch(C){if(r)return;p=String(C instanceof Error?C.message:C)}j()}function A(C){if(!C.enabled||C.error)return;const h=f[f.length-1];h&&h.since!==C.since&&(f.length=0);const E=new Map;for(const U of C.networks??[])E.set(U.chainId,U.received);f.push({t:Date.now(),since:C.since,received:E}),f.length>pa&&f.shift()}function j(){r||(N.innerHTML=W())}function W(){return p&&!l?`<h1>Analytics</h1><p class="error">${a(p)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:t?`
      ${B(t)}
      ${l?b(l):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${a(s)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function B(C){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${a(C.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${R()}</div>
      </div>
    `}function R(){if(!l)return"";if(!l.enabled)return"counters off";if(l.error)return"could not be read";const C=l.since?new Date(l.since):null;return C&&!Number.isNaN(C.getTime())?`totals since the gateway started, ${a(C.toLocaleString())}<br />re-read every ${et/1e3}s`:`re-read every ${et/1e3}s`}function b(C){return C.enabled?C.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${a(C.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:k(C)+le(C):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function k(C){const h=C.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${h.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':h.map(E=>T(E)).join("")}
      </section>
    `}function T(C){const h=C.methods??[],E=C.endpoints??[],U=C.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${C.chainId}</span>
          <span class="band-name">${a(C.name)}</span>
          ${q(C)}
        </div>
        <div class="an-stats">
          ${M("Received",de(C.received),"what clients asked this chain for")}
          ${M("Answered",de(C.answered),"returned by one of your endpoints")}
          ${M("From cache",de(C.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${M("Failed",de(C.failed),"asked for and never answered",C.failed>0?"bad":"")}
        </div>
        ${ne(C.chainId)}
        ${U?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':se("Method",h.map(F=>({label:F.method,l:F})))+se("Endpoint",E.map(F=>({label:F.upstream,l:F})))+V(C)}
      </div>
    `}function M(C,h,E,U=""){return`
      <div class="an-stat${U?" an-stat-"+U:""}" title="${a(E)}">
        <span class="an-stat-n">${a(h)}</span>
        <span class="an-stat-l">${a(C)}</span>
      </div>
    `}function q(C){const h=D(C.chainId);if(h===null)return'<span class="an-rate muted small">measuring rate…</span>';const E=Math.round((f[f.length-1].t-f[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${E}s apart.">
      ${a(h.toFixed(h<10?2:0))} req/s <span class="muted">over the last ${E}s</span>
    </span>`}function D(C){if(f.length<2)return null;const h=f[0],E=f[f.length-1],U=(E.t-h.t)/1e3;if(U<=0)return null;const F=(E.received.get(C)??0)-(h.received.get(C)??0);return F<0?null:F/U}function ne(C){if(f.length<3)return"";const h=[];for(let w=1;w<f.length;w++){const H=f[w-1],G=f[w],c=(G.t-H.t)/1e3,u=(G.received.get(C)??0)-(H.received.get(C)??0);h.push(c>0&&u>=0?u/c:0)}const E=Math.max(...h);if(E<=0)return"";const U=240,F=28,J=h.length>1?U/(h.length-1):U,g=h.map((w,H)=>`${(H*J).toFixed(1)},${(F-w/E*F).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${E.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${U} ${F}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${g}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${a(E.toFixed(2))} req/s</span>
      </div>
    `}function V(C){const h=[];return C.cached.count>0&&h.push(`${a(de(C.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${C.cached.mean===null?"":`, in ${a(je(C.cached.mean))} on average`}.`),C.failedLatency.count>0&&C.failedLatency.mean!==null&&h.push(`The ${a(de(C.failedLatency.count))} that failed took
         ${a(je(C.failedLatency.mean))} on average to fail.`),h.length===0?"":`<p class="muted small">${h.join(" ")}</p>`}function se(C,h){return h.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${a(C)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${h.map(E=>pe(E.label,E.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function pe(C,h){return`
      <tr>
        <td><code>${a(C)}</code></td>
        <td class="an-num">${de(h.count)}</td>
        <td class="an-num">${h.mean===null?'<span class="muted">—</span>':a(je(h.mean))}</td>
        <td>${Q(h)}</td>
      </tr>
    `}function Q(C){const h=C.buckets??[];if(h.length===0||C.count===0)return'<span class="muted small">—</span>';let E=0;const U=[];for(const J of h){const g=J.count-E;E=J.count,U.push({label:oe(J.le),n:Math.max(0,g)})}return U.reduce((J,g)=>J+g.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${a(U.filter(J=>J.n>0).map(J=>`${J.n} ${J.label}`).join(" · "))}">
        ${U.map((J,g)=>J.n===0?"":`<span class="an-band an-band-${Math.min(g,4)}" style="flex:${J.n}"></span>`).join("")}
      </span>
      <span class="muted small">${a(ie(U))}</span>
    `}function ie(C){for(let h=C.length-1;h>=0;h--)if(C[h].n>0)return`slowest ${C[h].label}`;return""}function oe(C){if(C==="+Inf")return"30s or more";const h=Number(C);return Number.isFinite(h)?`under ${je(h)}`:`under ${C}`}function le(C){const h=C.endpoints??[];return`
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
                     <tbody>${h.map(E=>Z(E)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function Z(C){const h=C.errors??[],E=h.reduce((F,J)=>F+J.count,0),U=h.length>0;return`
      <tr class="an-endpoint${U?" expandable":""}" ${U?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${a(C.upstream)}</code>
          ${C.chainId?`<span class="muted small">chain ${C.chainId}</span>`:""}
          ${C.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${de(C.requests)}</td>
        <td class="an-num${E>0?" bad":""}">${E>0?de(E):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${C.headLag>0?de(C.headLag):'<span class="muted">0</span>'}</td>
        <td>${me(C)}</td>
      </tr>
      ${U?ve(C,h):""}
    `}function me(C){const h=[];return C.scored?(h.push(C.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${a(String(C.position))}</span>`),h.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${a(C.score.toFixed(3))}</span>`),C.primarySwitches>1&&h.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${de(C.primarySwitches)} switches</span>`),C.excludedSeconds>0&&h.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${a(je(C.excludedSeconds))}</span>`),`<span class="an-selection">${h.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ve(C,h){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${h.map(E=>`
                    <tr>
                      <td class="an-num">${de(E.count)}</td>
                      <td><code>${a(E.class)}</code></td>
                      <td>${E.severity?`<span class="badge badge-${E.severity==="critical"?"bad":"warn"}">${a(E.severity)}</span>`:""}</td>
                      <td class="muted small">${a(E.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${a(C.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{r=!0,$!==null&&window.clearInterval($)}}function je(n){return!Number.isFinite(n)||n<0?"—":n>0&&n<5e-4?"<1ms":n<1?`${Math.round(n*1e3)}ms`:n<60?`${n<10?n.toFixed(1):Math.round(n)}s`:`${Math.round(n/60)}m`}function fa(n,s){let r=!1,t=null,l=null,p=!1,$=!1;n.innerHTML=`<h1>Network diagnostics: ${a(s)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${he()}</div>`;const f=n.querySelector("#diag-body"),N=n.querySelector("#diag-footer");we(n,(b,k)=>{var T;if(b==="run")_();else if(b==="toggle")(T=k.closest(".check-item"))==null||T.classList.toggle("expanded");else if(b==="copy"){const M=k.dataset.copy;M&&R(k,M)}}),O();async function O(){let b,k;try{const[M,q]=await Promise.all([Ee(),Ie()]);b=M.find(D=>D.id===s),k=q}catch(M){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${a(String(M))}</p>`;return}if(r)return;if(!b){f.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!b.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const T=k==null?void 0:k.networks.find(M=>M.ChainID===b.wire.ChainID);T&&(N.innerHTML=he(T.Name,T.LearnURL));try{t=await Yn(s),$=!0}catch(M){l=String(M instanceof Error?M.message:M)}r||A()}async function _(){p=!0,l=null,A();try{t=await Jn(s),$=!0}catch(b){l=String(b instanceof Error?b.message:b)}p=!1,r||A()}function A(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${p?"disabled":""}>${p?"Running…":"Run diagnostics"}</button>
      </div>
      ${l?`<p class="error">${a(l)}</p>`:""}
      ${j()}
    `}function j(){if(!$&&!l)return'<p class="muted">Loading…</p>';if(!t)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const b=new Date(t.at).toLocaleString(),k=t.failedId?`<p><strong>Failed at: ${a(W(t.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${a(b)} — trigger: ${a(t.trigger)}</p>
      ${k}
      <ul class="check-list">${t.items.map(B).join("")}</ul>
    `}function W(b){var k;return((k=t==null?void 0:t.items.find(T=>T.ID===b))==null?void 0:k.Title)??b}function B(b){const k=b.Status==="pass"?"ok":b.Status==="fail"?"bad":b.Status==="warn"?"warn":"neutral",T=b.ID===(t==null?void 0:t.failedId);return`
      <li class="check-item${T?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${K(T?"failed here":b.Status,k)}
          <strong>${a(b.Title)}</strong>
          <span class="muted small check-detail-inline">${a(b.Detail)}</span>
        </button>
        <div class="check-body">
          <details${T?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${a(b.Why)}</p>
          </details>
          ${b.Fix?`
                <details${T?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${a(b.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${a(b.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function R(b,k){const T=await Oe(k),M=b.textContent;b.textContent=T?"Copied!":"Copy failed",setTimeout(()=>{r||(b.textContent=M)},1500)}return()=>{r=!0}}const ma=85,tt={exec:"Execution",beacon:"Beacon"};function ba(n,s){let r=!1,t=null,l=null,p=null,$=null,f=null,N=null,O=null,_=null;const A={exec:null,beacon:null};let j=null;n.innerHTML=`<h1>Dashboard: ${a(s)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${he()}</div>`;const W=n.querySelector("#dash-body"),B=n.querySelector("#dash-footer");W.addEventListener("click",h=>{const E=h.target.closest("[data-action]");if(!E||!W.contains(E))return;const U=E.dataset.action;if(U==="svc-action"){const F=E.dataset.svc,J=E.dataset.kind;F&&J&&Z(F,J)}else if(U==="open-clear"){const F=E.dataset.svc;F&&ve(F)}else if(U==="copy"){const F=E.dataset.copy;F&&me(E,F)}else U==="retry-du"?b():U==="retry-endpoints"&&k()}),R();async function R(){let h,E;try{const[F,J]=await Promise.all([Ee(),Ie()]);h=F.find(g=>g.id===s),E=J}catch(F){if(r)return;W.innerHTML=`<p class="error">Failed to load target: ${a(String(F))}</p>`;return}if(r)return;if(!h){W.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){W.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const U=E==null?void 0:E.networks.find(F=>F.ChainID===h.wire.ChainID);U&&(B.innerHTML=he(U.Name,U.LearnURL)),W.innerHTML='<p class="muted">Connecting…</p>',t=jn(s,F=>{r||(T(F),l=F,p=F,M())}),b(),k()}async function b(){N=null;try{f=await Vn(s)}catch(h){f=null,N=String(h instanceof Error?h.message:h)}r||M()}async function k(){_=null;try{O=await Gn(s)}catch(h){O=null,_=String(h instanceof Error?h.message:h)}r||M()}function T(h){if(!l)return;const E=(new Date(h.at).getTime()-new Date(l.at).getTime())/1e3,U=h.execHead-l.execHead;if(E>0&&U>=0){const F=U/E;$=$===null?F:$*.7+F*.3}}function M(){if(!p)return;const h=p;W.innerHTML=`
      <p class="dash-status">${q(h)}</p>
      <div class="card-grid">
        ${oe(h)}
        ${ne(h)}
        ${V(h)}
        ${se(h)}
        ${pe(h)}
        ${Q()}
      </div>
      <p class="muted small">Last updated ${a(new Date(h.at).toLocaleTimeString())}</p>
    `}function q(h){return!h.execActive&&!h.beaconActive?K("Node not running","bad"):h.execSyncing||h.beaconDistance>0?K("Syncing","warn"):K("Running · synced","ok")}function D(h){const U=h.refHead>0?h.refHead-h.execHead:null,F=U!==null&&U>0&&$&&$>0?ua(U/$):U!==null&&U<=0?"caught up":"—";return{lag:U,eta:F}}function ne(h){const{lag:E,eta:U}=D(h);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${h.execActive?h.execSyncing?K("syncing","warn"):h.execHead===0?K("no data","neutral"):K("synced","ok"):K("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${de(h.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${E!==null?de(h.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${E!==null?de(Math.max(E,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${U}</dd></div>
        </dl>
      </div>
    `}function V(h){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${h.beaconActive?h.beaconSlot===0?K("no data","neutral"):h.beaconDistance===0?K("synced","ok"):K("syncing","warn"):K("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${de(h.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${de(h.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function se(h){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${de(h.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${de(h.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function pe(h){const E=h.diskUsedPct>=ma,U=`
      <div class="meter"><div class="meter-fill ${E?"meter-warn":""}" style="width:${Math.min(h.diskUsedPct,100)}%"></div></div>
      <p>${da(h.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${E?"card-warn":""}">
          <h3>Storage</h3>
          ${U}
          <p class="error small">${a(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!f)return`
        <div class="card ${E?"card-warn":""}">
          <h3>Storage</h3>
          ${U}
          <p class="muted">Loading…</p>
        </div>
      `;const F=f.ExpectedExecBytes>0?Math.min(f.ExecBytes/f.ExpectedExecBytes*100,100):0,J=f.ExpectedBeaconBytes>0?Math.min(f.BeaconBytes/f.ExpectedBeaconBytes*100,100):0,{lag:g,eta:w}=D(h),H=g!==null&&g>0&&$!==null&&$>0;return`
      <div class="card ${E?"card-warn":""}">
        <h3>Storage</h3>
        ${U}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Le(f.ExecBytes)} of ~${Le(f.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${F}%"></div></div>
        ${H?`<p class="muted small">Estimated time remaining: ${a(w)}</p>`:""}
        <p class="muted small">Beacon — ${Le(f.BeaconBytes)} of ~${Le(f.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${J}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Le(f.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${a(f.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${a(f.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function Q(){if(_)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${a(_)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!O)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const h=O,E=h.ExecReachable&&!h.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",U=h.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${a(h.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${xe(h.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${xe(h.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${a(h.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(h.BeaconHTTP)}">Copy</button>
        </div>
        ${E}
        ${U}
      </div>
    `}function ie(h,E){const U=tt[h],F=A[h],J=(g,w,H)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${h}" data-kind="${g}" ${F!==null||H?"disabled":""}>${F===g?le():a(w)}</button>`;return`
      <div class="service-row">
        <span>${a(U)} ${E?K("active","ok"):K("down","bad")}</span>
        <div class="service-actions">
          ${J("start","Start",E)}
          ${J("stop","Stop",!E)}
          ${J("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${h}" ${F!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function oe(h){return`
      <div class="card">
        <h3>Services</h3>
        ${ie("exec",h.execActive)}
        ${ie("beacon",h.beaconActive)}
        ${j?`<p class="error small">${a(j)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(s)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(s)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(s)}">Diagnostics →</a>
        </p>
      </div>
    `}function le(){return'<span class="spinner" aria-label="working"></span>'}async function Z(h,E){if(A[h]===null){A[h]=E,j=null,M();try{await _n(s,h,E)}catch(U){j=`${tt[h]} ${E} failed: ${U instanceof Error?U.message:String(U)}`}A[h]=null,r||M()}}async function me(h,E){const U=await Oe(E),F=h.textContent;h.textContent=U?"Copied!":"Copy failed",setTimeout(()=>{r||(h.textContent=F)},1500)}function ve(h){const E=tt[h],U=f?Le(h==="exec"?f.ExecBytes:f.BeaconBytes):"unknown (disk usage hasn't loaded)";re(`
        <h2>Clear ${a(E)} data</h2>
        <p class="error">
          This stops the ${a(E.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${a(U)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${a(h)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,g=>{if(g==="cancel"){Y();return}g==="confirm"&&C(h)});const F=document.getElementById("clear-confirm-input"),J=document.getElementById("clear-confirm-btn");F==null||F.addEventListener("input",()=>{J&&(J.disabled=F.value.trim()!==h)}),F==null||F.focus()}async function C(h){const E=document.getElementById("clear-confirm-btn");E&&(E.disabled=!0,E.textContent="Clearing…");try{await Kn(s,h),Y(),b()}catch(U){const F=Me();if(F){const J=document.createElement("p");J.className="error small",J.textContent=`Clear failed: ${U instanceof Error?U.message:String(U)}`,F.appendChild(J)}E&&(E.disabled=!1,E.textContent="Clear and resync")}}return()=>{r=!0,t==null||t(),Y()}}const St=500,xt="valve-node-app.explain-consent";function ya(n,s){let r=!1,t=null;const l=[];n.innerHTML=`
    <h1>Logs: ${a(s)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${he()}</div>
  `;const p=n.querySelector("#logs-body"),$=n.querySelector("#logs-footer");we(n,R=>{R==="explain"&&_()}),f();async function f(){let R,b;try{const[T,M]=await Promise.all([Ee(),Ie()]);R=T.find(q=>q.id===s),b=M}catch(T){if(r)return;p.innerHTML=`<p class="error">Failed to load target: ${a(String(T))}</p>`;return}if(r)return;if(!R){p.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!R.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const k=b==null?void 0:b.networks.find(T=>T.ChainID===R.wire.ChainID);k&&($.innerHTML=he(k.Name,k.LearnURL));try{const T=await qn(s,200);if(r)return;l.push(...T)}catch(T){if(r)return;p.innerHTML=`<p class="error">Failed to load logs: ${a(String(T))}</p>`;return}N(),t=Wn(s,T=>{r||(l.push(T),l.length>St&&l.splice(0,l.length-St),N())})}function N(){const R=l.filter(k=>k.severity==="error"||k.severity==="critical");p.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(O).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${K(String(R.length),R.length?"bad":"neutral")}</h2>
          <div class="log-lines">${R.length?R.slice().reverse().map(O).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const b=p.querySelector(".log-lines");b&&(b.scrollTop=b.scrollHeight)}function O(R){const b=R.severity||"info",k=R.learnUrl?` <a href="${a(R.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${a(b)}">
        <span class="log-time">${a(new Date(R.at).toLocaleTimeString())}</span>
        <span class="log-unit">${a(R.unit)}</span>
        <span class="log-sev">${a(b)}</span>
        <span class="log-text">${a(R.line)}</span>
        ${R.explain?`<div class="log-explain">${a(R.explain)}${k}</div>`:""}
      </div>
    `}async function _(){const R=l.filter(k=>k.severity==="error"||k.severity==="critical").map(k=>k.line).slice(-40);if(!(localStorage.getItem(xt)==="1")){A(R);return}await j(R)}function A(R){const b=R.length?`<pre class="explain-excerpt">${R.map(k=>a(k)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';W(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${b}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,k=>{k==="proceed"?(localStorage.setItem(xt,"1"),B(),j(R)):B()})}async function j(R){W('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const b=R.length?await wt(s,R):await wt(s);if(r)return;W(`
        <h2>Explanation</h2>
        <div class="explain-text">${a(b.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${b.sentExcerpt.map(k=>a(k)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,k=>{k==="close"&&B()})}catch(b){if(r)return;if(b instanceof Ce&&b.status===409){W(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,k=>{k==="close"&&B()});return}W(`
        <h2>Explain failed</h2>
        <p class="error">${a(b instanceof Error?b.message:String(b))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,k=>{k==="close"&&B()})}}function W(R,b){B();const k=document.createElement("div");k.className="modal-overlay",k.id="explain-modal",k.innerHTML=`<div class="modal">${R}</div>`,k.addEventListener("click",T=>{const M=T.target.closest("[data-modal-action]");M!=null&&M.dataset.modalAction&&b(M.dataset.modalAction),T.target===k&&b("cancel")}),document.body.appendChild(k)}function B(){var R;(R=document.getElementById("explain-modal"))==null||R.remove()}return()=>{r=!0,t==null||t(),B()}}const va="run",ga={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},$a={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function wa(n,s){let r=!1,t=null,l=null;const p={devnet:null},$={devnet:null},f={devnet:[]};let N=null;const O={devnet:!1};let _=null;const A={devnet:null},j={devnet:null};n.innerHTML=`
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
    ${he()}
  `;const W=n.querySelector("#services-body");we(n,(c,u)=>{ve(c,u)}),B();async function B(){try{const c=await At(s);if(r)return;t=c,l=null}catch(c){if(r)return;t=null,l=H(c)}b()}function R(c){return t==null?void 0:t.services.find(u=>u.id===c)}function b(){if(!r){if(l){W.innerHTML=`<p class="error">Could not read this machine's services: ${a(l)}</p>`;return}if(!t){W.innerHTML='<p class="muted">Loading…</p>';return}W.innerHTML=`
      ${k(t.docker)}
      <div class="card-grid card-grid-wide">
        ${t.services.map(T).join("")}
      </div>
    `}}function k(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${a(c.flavor)}${c.serverVersion?` ${a(c.serverVersion)}`:""} · reachable</p>`;const u=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${a(u)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${a(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${a(c.hint)}</div>`:""}
      </div>
    `}function T(c){const u=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${a(c.label)}</h2>
          ${M(c)}
        </div>
        <p class="muted small">${a(ga[c.id]??"")}</p>

        ${c.error?q(c):""}
        ${c.blocked?`<div class="banner banner-warn">${a(c.blocked)}</div>`:""}
        ${u.map(m=>`<div class="banner banner-warn">${a(m)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${a(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${a(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${D(c)}

        ${ne(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(m=>V(c,m)).join("")}
        </div>
        ${$[c.id]?`<p class="error small">${a($[c.id])}</p>`:""}
        ${se(c)}

        ${pe(c)}
      </div>
    `}function M(c){switch(c.status.State){case"running":return K("running","ok");case"created-but-stopped":return K("stopped","warn");case"not-created":return K("not created","neutral");default:return K("unknown","bad")}}function q(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${a(c.error??"")}</div>
        ${c.hint?`<div class="small">${a(c.hint)}</div>`:""}
      </div>
    `}function D(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const u=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${u}.</p>`}function ne(c){const u=c.endpoints??[];return u.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":u.map(m=>`
        <div class="endpoint-row">
          ${xe("ok")}
          <span class="muted small">${a(m.label)}</span>
          <code class="endpoint-url">${a(m.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${a(m.url)}">Copy</button>
        </div>`).join("")}function V(c,u){const m=$a[u];if(!m)return"";const P=p[c.id],I=u==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:m.label;return`
      <button class="${m.className}" data-action="svc-${u}" data-svc="${a(c.id)}"
              title="${a(m.title)}" ${P?"disabled":""}>
        ${P===u?'<span class="spinner" aria-label="working"></span>':a(I)}
      </button>
    `}function se(c){const u=f[c.id]??[];return u.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${a(u.join(`
`))}</pre>
      </div>
    `}function pe(c){const u=O[c.id],m=Q(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${a(c.id)}">
            ${u?"Close":"Edit"}
          </button>
        </div>
        ${u?ie():`<p class="small">${m}</p>`}
        ${A[c.id]?`<p class="error small">${a(A[c.id])}</p>`:""}
        ${j[c.id]?`<p class="muted small">${a(j[c.id])}</p>`:""}
      </div>
    `}function Q(c){const u=c.devnet;return u?`Chain ${u.ChainID} · a block every ${a(u.BlockTime)} · JSON-RPC on ${a(u.BindAddr)}:${u.HTTPPort} · WebSocket on ${a(u.BindAddr)}:${u.WSPort}`:"—"}function ie(c){return oe()}function oe(){const c=_;return c?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${a(c.BlockTime)}" autocomplete="off" spellcheck="false" />
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
        <input type="text" id="dev-bind" value="${a(c.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${c.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function le(){O.devnet&&_&&(_.BlockTime=Z("#dev-blocktime",_.BlockTime),_.HTTPPort=me("#dev-http",_.HTTPPort),_.WSPort=me("#dev-ws",_.WSPort),_.BindAddr=Z("#dev-bind",_.BindAddr))}function Z(c,u){const m=n.querySelector(c);return m?m.value.trim():u}function me(c,u){const m=n.querySelector(c);if(!m)return u;const P=Number.parseInt(m.value.trim(),10);return Number.isFinite(P)?P:u}async function ve(c,u){const m=u.dataset.svc??"";switch(c){case"refresh":await B();return;case"copy":u.dataset.copy&&await w(u,u.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await C(m,c.slice(4));return;case"svc-create":case"svc-recreate":await h(m);return;case"svc-wipe":F(m);return;case"toggle-config":E(m);return;case"save-config":await U(m);return;default:return}}async function C(c,u){if(!p[c]){p[c]=u,$[c]=null,b();try{await Zn(s,c,u)}catch(m){$[c]=`${u} failed: ${H(m)}${G(m)}`}p[c]=null,await B()}}async function h(c){if(!p[c]){p[c]="create",$[c]=null,f[c]=["starting…"],b();try{await Qn(s,c)}catch(u){$[c]=`${H(u)}${G(u)}`,f[c]=[],p[c]=null,b();return}N==null||N(),N=qe(s,u=>{if(r)return;const m=u.err?`${u.stepId}: ${u.err}`:u.line?`${u.stepId}: ${u.line}`:`${u.stepId}: done`;if(f[c]=[...(f[c]??[]).filter(I=>I!=="starting…"),m],!!u.err||u.stepId===va&&!!u.done){N==null||N(),N=null,p[c]=null,u.err&&($[c]="Provisioning failed — see the log below."),B();return}b()})}}function E(c){if(le(),O[c]=!O[c],A[c]=null,j[c]=null,O[c]){const u=R(c);u!=null&&u.devnet&&(_={...u.devnet})}b()}async function U(c){var P;le(),A[c]=null,j[c]=null;const u=_;if(!u)return;if(u.HTTPPort===u.WSPort){A[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",b();return}try{await ta(s,c,u)}catch(I){A[c]=H(I),b();return}const m=((P=R(c))==null?void 0:P.status.State)==="running";O[c]=!1,j[c]=m?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await B()}function F(c){const u=R(c);if(!u)return;const m=(u.restartsOnWipe??[]).map(v=>{var z;return((z=R(v))==null?void 0:z.label)??v});re(`
        <h2>Wipe ${a(u.label)}</h2>
        <p class="error">This deletes ${a(u.wipeDiscards)}</p>
        ${m.length?`<p>It also restarts what sits in front of it: ${a(m.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${a(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(c)}</button>
        </div>
      `,v=>{if(v==="cancel"||v==="close"){Y(),B();return}v==="confirm"&&J(c)});const P=document.getElementById("wipe-confirm-input"),I=document.getElementById("wipe-confirm-btn");P==null||P.addEventListener("input",()=>{I&&(I.disabled=P.value.trim()!==c)}),P==null||P.focus()}async function J(c){const u=document.getElementById("wipe-confirm-btn");u&&(u.disabled=!0,u.textContent="Wiping…");let m;try{m=await Xn(s,c)}catch(P){const I=Me();if(I){const v=document.createElement("p");v.className="error small",v.textContent=`Wipe failed: ${H(P)}${G(P)}`,I.appendChild(v)}u&&(u.disabled=!1,u.textContent=`Wipe ${c}`);return}g(c,m)}function g(c,u){const m=R(c),P=te=>{var fe;return((fe=R(te))==null?void 0:fe.label)??te},I=[];I.push(u.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const te of u.report.VolumesRemoved??[])I.push(`Volume ${te} deleted.`);for(const te of u.report.VolumesAbsent??[])I.push(`Volume ${te} was already gone.`);u.report.Recreated&&I.push("Container re-created from your saved configuration.");const v=(u.report.Cascaded??[]).map(P),z=(u.report.CascadeSkipped??[]).map(P);re(`
        <h2>${a((m==null?void 0:m.label)??c)} wiped</h2>
        <ul class="plain-list">${I.map(te=>`<li>${a(te)}</li>`).join("")}</ul>
        ${v.length?`<p class="ok">Restarted in front of it: ${a(v.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${z.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(z.join(", "))}.</p>`:""}
        ${u.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(u.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,te=>{(te==="close"||te==="cancel")&&(Y(),B())})}async function w(c,u){const m=await Oe(u),P=c.textContent;c.textContent=m?"Copied!":"Copy failed",setTimeout(()=>{r||(c.textContent=P)},1500)}function H(c){return c instanceof Error?c.message:String(c)}function G(c){return c instanceof Ce&&c.hint?` — ${c.hint}`:""}return()=>{r=!0,N==null||N(),Y()}}const nt=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ve=8545,Ge=5052,ze=30303,ka=[369,943,1],Tt={369:"default",943:"practise here first"};function Ca(n,s){let r=!1;const t={targetId:s,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};n.innerHTML=`<h1>Setup: ${a(s)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${he()}</div>`;const l=n.querySelector("#wizard-body"),p=n.querySelector("#wizard-footer");we(n,(g,w)=>{me(g,w)}),dt(n,(g,w)=>{g==="exec-select"?t.execId=w:g==="beacon-select"&&(t.beaconId=w),f()}),n.addEventListener("change",g=>{const w=g.target;w instanceof HTMLInputElement&&(w.id==="data-dir-input"?(ve(),V()):w.id==="checkpoint-toggle"?(t.checkpoint=w.checked,f()):w.id==="exec-snapshot-toggle"&&(t.execSnapshot=w.checked,f()))}),$();async function $(){try{const[g,w]=await Promise.all([Ie(),Ee()]);if(r)return;t.catalog=g;const H=w.find(G=>G.id===s);H!=null&&H.wire&&(t.chainId=H.wire.ChainID,t.execId=H.wire.ExecID,t.beaconId=H.wire.BeaconID,t.archive=H.wire.Archive,H.wire.ExecHTTPPort&&(t.execHTTPPort=String(H.wire.ExecHTTPPort)),H.wire.BeaconHTTPPort&&(t.beaconHTTPPort=String(H.wire.BeaconHTTPPort)),H.wire.ExecP2PPort&&(t.execP2PPort=String(H.wire.ExecP2PPort)),H.wire.RPCBindAddr&&(t.rpcBindAddr=H.wire.RPCBindAddr)),f()}catch(g){if(r)return;t.loadError=String(g instanceof Error?g.message:g),f()}}function f(){if(t.loadError){l.innerHTML=`<p class="error">Failed to load: ${a(t.loadError)}</p>`;return}t.catalog&&(l.innerHTML=`
      ${J(t.step)}
      ${O()}
    `,N())}function N(){var w;const g=(w=t.catalog)==null?void 0:w.networks.find(H=>H.ChainID===t.chainId);p.innerHTML=g?he(g.Name,g.LearnURL):he()}function O(){switch(t.step){case"network":return _();case"clients":return A();case"mode":return oe();case"review":return le();case"run":return Z()}}function _(){const g=t.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${ka.map(H=>{const G=g.networks.find(m=>m.ChainID===H);if(!G)return"";const c=t.chainId===H,u=Tt[H]?K(Tt[H],H===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${H}" type="button">
          <h3>${a(G.Name)} <span class="muted">(chain ${H})</span></h3>
          ${u}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${t.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function A(){const g=t.catalog,w=g.networks.find(c=>c.ChainID===t.chainId);if(!w)return'<p class="error">Unknown network.</p>';(t.execId===null||!w.ExecClients.includes(t.execId))&&(t.execId=w.ExecClients[0]??null),(t.beaconId===null||!w.BeaconClients.includes(t.beaconId))&&(t.beaconId=w.BeaconClients[0]??null);const H=w.ExecClients.map(c=>pe(c,g)),G=w.BeaconClients.map(c=>pe(c,g));return`
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
          ${it("exec-select",H,t.execId)}
        </label>
        ${ie(t.execId,g)}
        <label>
          Beacon client
          ${it("beacon-select",G,t.beaconId)}
        </label>
        ${ie(t.beaconId,g)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function j(g){return g<=0?"—":g>=1?`~${g.toFixed(1)} TB`:`~${Math.round(g*1e3)} GB`}const W=1.1,B=.5,R="Valve reth snapshot",b="rough estimate";function k(g){return g.SnapshotSizeTB}function T(g){return g.SnapshotSizeTB*B}function M(g){return`<p class="muted small">${j(k(g))} is the measured size of Valve's reth snapshot for ${a(g.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function q(g){return{archive:k(g)*1e12*W,full:T(g)*1e12*W}}function D(g,w){if(!g)return"";if(t.diskProbing)return`<p class="muted small">Checking free space at <code>${a(w)}</code>…</p>`;if(t.diskError)return`<p class="error small">Couldn't read free space at <code>${a(w)}</code>: ${a(t.diskError)}</p>`;if(t.freeBytes===null||t.probedPath!==w)return"";const H=q(g),G=t.freeBytes>=H.archive,c=t.freeBytes>=H.full,u=`<p class="muted small">Free at <code>${a(w)}</code>: <strong>${Le(t.freeBytes)}</strong> — archive ${G?"fits":"won't fit"} (${j(k(g))}, ${R}), full ${c?"fits":"won't fit"} (${j(T(g))}, ${b}).</p>`;let m="";return t.downgradeNote?m=`<p class="banner banner-warn">${a(t.downgradeNote)}</p>`:c||(m=`<p class="banner banner-warn">Neither full (${j(T(g))}, ${b}) nor archive (${j(k(g))}, ${R}) fits the free space here — choose a location with more room.</p>`),u+m}function ne(g,w){if(t.downgradeNote=null,!g||t.freeBytes===null)return;const H=q(g);t.archive&&t.freeBytes<H.archive&&t.freeBytes>=H.full&&(t.archive=!1,t.downgradeNote=`Not enough space at ${w} for archive (${j(k(g))}, ${R}) — switched to Full (${j(T(g))}, ${b}). Pick a location with more room to run archive.`)}async function V(){var H;if(t.chainId===null)return;const g=(H=t.catalog)==null?void 0:H.networks.find(G=>G.ChainID===t.chainId),w=(t.dataDir||`/var/lib/valve-node-app/${t.chainId}`).trim();t.diskProbing=!0,t.diskError=null,f();try{const{freeBytes:G}=await On(t.targetId,w);if(r)return;t.freeBytes=G,t.probedPath=w,ne(g,w)}catch(G){if(r)return;t.freeBytes=null,t.probedPath=w,t.diskError=String(G instanceof Error?G.message:G)}t.diskProbing=!1,f()}function se(g){return g?/^https?:\/\/.+/i.test(g)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function pe(g,w){const H=w.clients.find(G=>G.id===g);return{value:g,label:H?`${H.id} — ${Q(H.repo)}`:g}}function Q(g){const w=g.split("/");return w.length>=4?w[3]:g}function ie(g,w){const H=g?w.clients.find(c=>c.id===g):void 0;if(!H)return"";const G=H.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${a(H.repo)}" target="_blank" rel="noopener noreferrer">${a(G)}</a></p>`}function oe(){var P,I,v;const g=t.chainId!==null?`/var/lib/valve-node-app/${t.chainId}`:"",w=(P=t.catalog)==null?void 0:P.networks.find(z=>z.ChainID===t.chainId),H=((v=(I=t.catalog)==null?void 0:I.clients.find(z=>z.id===t.execId))==null?void 0:v.snapshotSupported)??!1,G=w?`${j(T(w))} (${b})`:"Smaller",c=w?`${j(k(w))} (${R})`:"Much larger",u=w?` on ${a(w.Name)}`:"",m=w?t.checkpoint?w.SyncLabel:w.GenesisSyncLabel:"";return`
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
          ${w?`<p class="sync-estimate">⏱ Estimated initial sync${u}: <strong>${a(m)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${t.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${a((w==null?void 0:w.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${a((w==null?void 0:w.CheckpointURL)??"")}" value="${a(t.checkpointUrl)}" />
                 </label>
                 ${t.checkpointUrlError?`<p class="error small">${a(t.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${H?`
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
              <tr><th>Approx. disk footprint${u}</th><td class="yes">${G}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${w?M(w):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${t.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${w?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${t.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${G}${w?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${a(g)})</span>
            <input id="data-dir-input" type="text" placeholder="${a(g)}" value="${a(t.dataDir)}" />
          </label>
          ${D(w,t.dataDir||g)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${a(g)}/jwt.hex" value="${a(t.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Ve})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Ve}" value="${a(t.execHTTPPort)}" />
          </label>
          ${t.execHTTPPortError?`<p class="error small">${a(t.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Ge})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Ge}" value="${a(t.beaconHTTPPort)}" />
          </label>
          ${t.beaconHTTPPortError?`<p class="error small">${a(t.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${ze})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${ze}" value="${a(t.execP2PPort)}" />
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
    `}function le(){const w=t.catalog.networks.find(te=>te.ChainID===t.chainId),H=t.dataDir||`/var/lib/valve-node-app/${t.chainId}`,G=t.jwtPath||`${H}/jwt.hex`,c=nt.map(te=>`<li>${a(te.title)}</li>`).join(""),u=U(t.execHTTPPort,Ve),m=U(t.beaconHTTPPort,Ge),P=U(t.execP2PPort,ze),I=u||m||P?`<tr><th>Non-default ports</th><td>${[u?`exec HTTP ${u}`:null,m?`beacon HTTP ${m}`:null,P?`exec p2p ${P}`:null].filter(te=>te!==null).map(a).join(", ")}</td></tr>`:"",{addr:v}=C(t.rpcBindAddr),z=v?`<tr><th>RPC bind address</th><td><code>${a(v)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${a(t.targetId)}</td></tr>
            <tr><th>Network</th><td>${a((w==null?void 0:w.Name)??String(t.chainId))} (chain ${t.chainId})</td></tr>
            <tr><th>Execution client</th><td>${a(t.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${a(t.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${t.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${a(H)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${a(G)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${t.checkpoint?`<code>${a(t.checkpointUrl||(w==null?void 0:w.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${I}
            ${z}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${c}</ol>
        ${t.startError?`<p class="error">${a(t.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${t.starting?"disabled":""}>
            ${t.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function Z(){const w=t.catalog.networks.find(v=>v.ChainID===t.chainId),H=w==null?void 0:w.LearnURL,G=new Set(t.events.filter(v=>v.done).map(v=>v.stepId)),c=new Set(t.events.filter(v=>v.err).map(v=>v.stepId)),u=new Map;for(const v of t.events){if(!v.line)continue;const z=u.get(v.stepId)??[];z.push(v.line),u.set(v.stepId,z)}const m=nt.map(v=>{var _e;const z=G.has(v.id),te=c.has(v.id),fe=te?K("failed","bad"):z?K("done","ok"):K("pending","neutral"),Ne=(u.get(v.id)??[]).slice(-5),We=(_e=t.events.find(Ae=>Ae.stepId===v.id&&Ae.err))==null?void 0:_e.err,Xe=v.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${H?` <a href="${a(H)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${z?"step-done":""} ${te?"step-error":""}">
          <div class="step-head">${fe} <strong>${a(v.title)}</strong></div>
          ${Xe}
          ${Ne.length?`<pre class="step-log">${Ne.map(Ae=>a(Ae)).join(`
`)}</pre>`:""}
          ${We?`<p class="error small">${a(We)}</p>`:""}
        </li>
      `}).join(""),P=t.events.some(v=>v.err),I=nt.every(v=>G.has(v.id))||t.events.some(v=>v.stepId==="handshake"&&v.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${m}</ol>
        ${I&&!P?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(t.targetId)}">Open the dashboard →</a></p>`:""}
        ${t.startError?`<p class="error">${a(t.startError)}</p>`:""}
        ${P?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function me(g,w){switch(g){case"pick-network":t.chainId=Number(w.dataset.chainId),t.execId=null,t.beaconId=null,f();break;case"goto-network":t.step="network",f();break;case"goto-clients":if(t.chainId===null)return;t.step="clients",f();break;case"goto-mode":t.step="mode",f(),V();break;case"goto-review":if(ve(),t.execHTTPPortError||t.beaconHTTPPortError||t.execP2PPortError||t.rpcBindAddrError||t.checkpointUrlError||t.snapshotKeyError){f();break}t.step="review",f();break;case"start-setup":F();break}}function ve(){const g=n.querySelectorAll('input[name="mode"]');for(const v of Array.from(g))v.checked&&(t.archive=v.value==="archive");const w=n.querySelector("#data-dir-input"),H=n.querySelector("#jwt-path-input");w&&(t.dataDir=w.value.trim()),H&&(t.jwtPath=H.value.trim());const G=n.querySelector("#exec-http-port-input"),c=n.querySelector("#beacon-http-port-input"),u=n.querySelector("#exec-p2p-port-input");G&&(t.execHTTPPort=G.value.trim()),c&&(t.beaconHTTPPort=c.value.trim()),u&&(t.execP2PPort=u.value.trim());const m=n.querySelector("#rpc-bind-addr-input");m&&(t.rpcBindAddr=m.value.trim());const P=n.querySelector("#checkpoint-url-input");P&&(t.checkpointUrl=P.value.trim());const I=n.querySelector("#snapshot-key-input");I&&(t.snapshotKey=I.value.trim()),t.execHTTPPortError=E(t.execHTTPPort).error??null,t.beaconHTTPPortError=E(t.beaconHTTPPort).error??null,t.execP2PPortError=E(t.execP2PPort).error??null,t.rpcBindAddrError=C(t.rpcBindAddr).error??null,t.checkpointUrlError=t.checkpoint?se(t.checkpointUrl):null,t.snapshotKeyError=t.execSnapshot&&!t.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function C(g){if(!g)return{};const w=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(g);return w?w.slice(1).every(H=>Number(H)<=255)?{addr:g}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(g)&&g.includes(":")?{addr:g}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const h=/^\d+$/;function E(g){if(!g)return{};if(!h.test(g))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const w=Number(g);return!Number.isInteger(w)||w<1||w>65535?{error:"Port must be between 1 and 65535."}:{port:w}}function U(g,w){const{port:H}=E(g);if(!(H===void 0||H===w))return H}async function F(){var u;if(t.chainId===null||!t.execId||!t.beaconId)return;t.starting=!0,t.startError=null,t.events=[],(u=t.streamStop)==null||u.call(t),t.streamStop=null,f();const g={ChainID:t.chainId,ExecID:t.execId,BeaconID:t.beaconId,Archive:t.archive};t.dataDir&&(g.DataDir=t.dataDir),t.jwtPath&&(g.JWTPath=t.jwtPath);const w=U(t.execHTTPPort,Ve),H=U(t.beaconHTTPPort,Ge),G=U(t.execP2PPort,ze);w!==void 0&&(g.ExecHTTPPort=w),H!==void 0&&(g.BeaconHTTPPort=H),G!==void 0&&(g.ExecP2PPort=G);const{addr:c}=C(t.rpcBindAddr);c!==void 0&&(g.RPCBindAddr=c),t.checkpoint?t.checkpointUrl&&(g.CheckpointURL=t.checkpointUrl):g.NoCheckpoint=!0,t.execSnapshot&&(g.ExecSnapshot=!0,g.SnapshotKey=t.snapshotKey);try{await Fn(t.targetId,g)}catch(m){if(!(m instanceof Ce&&m.status===409)){t.starting=!1,t.startError=String(m instanceof Error?m.message:m),f();return}}t.starting=!1,t.step="run",f(),t.streamStop=qe(t.targetId,m=>{r||(t.events.push(m),t.step==="run"&&f())})}function J(g){const w=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],G=w.map(c=>c.id).indexOf(g);return`
      <ol class="wizard-progress">
        ${w.map((c,u)=>`<li class="${u===G?"current":u<G?"past":"future"}">${a(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var g;r=!0,(g=t.streamStop)==null||g.call(t)}}function Sa(n,s){let r=!1;const t=new Map;n.innerHTML=`<h1>${a(s)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;const l=n.querySelector("#machine-body");we(n,(A,j)=>{A==="toggle-section"&&O(j.dataset.section??"")}),p();async function p(){let A,j;try{const[W,B]=await Promise.all([Ee(),Ie()]);A=W.find(R=>R.id===s),j=B}catch(W){if(r)return;l.innerHTML=`<p class="error">Failed to load machine: ${a(String(W))}</p>`;return}if(!r){if(!A){location.hash="#/targets";return}$(A,j)}}function $(A,j){const W=A.mode==="local"?"this machine":"SSH",B=A.mode==="ssh"&&A.ssh?`${a(A.ssh.User)}@${a(A.ssh.Host)}`:W;l.innerHTML=`
      <p class="muted">${B}</p>
      <p>${f(A,j)}</p>
      <div class="machine-sections">
        ${_.map(R=>N(R,A,j)).join("")}
      </div>
      ${he()}
    `}function f(A,j){const W=A.wire;if(!W)return K("not set up","neutral");const B=j.networks.find(b=>b.ChainID===W.ChainID),R=B?B.Name:`chain ${W.ChainID}`;return`${K(R,"ok")} ${K(W.ExecID,"neutral")} ${K(W.BeaconID,"neutral")}${W.Archive?" "+K("archive","warn"):""}`}function N(A,j,W){return`
      <section class="card machine-section" data-section-card="${a(A.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${a(A.key)}" aria-expanded="false">
          <span class="machine-section-title">${a(A.title)}</span>
          <span class="machine-section-status">${A.status(j,W)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${a(A.key)}" hidden></div>
      </section>
    `}function O(A){const j=_.find(k=>k.key===A);if(!j)return;const W=n.querySelector(`[data-section-card="${A}"]`),B=n.querySelector(`[data-section-body="${A}"]`),R=n.querySelector(`.machine-section-head[data-section="${A}"]`);if(!W||!B||!R)return;const b=B.hidden;if(b&&!t.has(A)){const k=document.createElement("div");B.appendChild(k),t.set(A,j.mount(k))}B.hidden=!b,W.classList.toggle("open",b),R.setAttribute("aria-expanded",String(b))}const _=[{key:"setup",title:"Setup",status:A=>A.wire?K("set up","ok"):K("not set up","neutral"),mount:A=>Ca(A,s)},{key:"dashboard",title:"Dashboard",status:A=>A.wire?'<span class="muted small">sync, peers, storage and endpoints — live</span>':'<span class="muted small">available once this machine is set up</span>',mount:A=>ba(A,s)},{key:"logs",title:"Logs",status:A=>A.wire?'<span class="muted small">live tail and error feed</span>':'<span class="muted small">available once this machine is set up</span>',mount:A=>ya(A,s)},{key:"services",title:"Devnet",status:()=>'<span class="muted small">throwaway chain — always available on this machine</span>',mount:A=>wa(A,s)}];return()=>{r=!0;for(const A of t.values())try{A()}catch{}t.clear()}}function xa(n){let s;try{s=new URL(n).hostname}catch{return"endpoint"}if(!s)return"endpoint";if(s==="localhost"||/^[0-9.]+$/.test(s)||/^\[.*\]$/.test(s))return s;const r=s.split(".").filter(Boolean);return r.length<=1?s:r[r.length-2]}function Ft(n){var t;if(!n)return{tone:"off",label:"Not set up",sub:"Press to set up your endpoint",actions:[]};const s=n.actions??[];if(n.blocked)return{tone:"blocked",label:"Unavailable",sub:n.blocked,actions:s,blocked:n.blocked};const r=((t=n.networks)==null?void 0:t.length)??0;return n.status.State==="running"?{tone:"on",label:"Running",sub:`${r} network${r===1?"":"s"} served`,actions:s}:{tone:"off",label:"Stopped",sub:r?`${r} network${r===1?"":"s"} configured`:"Press to start",actions:s}}function Ye(n){if(!n.running)return"off";if(!n.serviceable)return"frequent";const s=n.slowRate??0;return s>.4?"frequent":s>=.1?"occasional":"stable"}const Ta=[{key:"http",label:"HTTP"},{key:"ws",label:"WS"},{key:"archive",label:"Archive",hot:!0},{key:"trace",label:"Trace"}];function ut(n){return Ta.map(({key:s,label:r,hot:t})=>{const l=n[s]==="supported";return{key:s,label:r,lit:l,hot:!!t&&l}})}function Ia(n,s,r){const t=n.Networks??[],l=t.findIndex(f=>f.ChainID===s),p={ChainID:s,Upstreams:r},$=l===-1?[...t,p]:t.map((f,N)=>N===l?p:f);return{...n,Networks:$}}function Ea(n,s){const r=n.Networks??[];return{...n,Networks:r.filter(t=>t.ChainID!==s)}}function at(n,s,r){const t=n.Networks??[],l=t.findIndex(O=>O.ChainID===s);if(l===-1)return{...n,Networks:[...t,{ChainID:s,Upstreams:[r]}]};const p=t[l],$=p.Upstreams.findIndex(O=>O.ID===r.ID),f=$===-1?[...p.Upstreams,r]:p.Upstreams.map((O,_)=>_===$?r:O),N={...p,Upstreams:f};return{...n,Networks:t.map((O,_)=>_===l?N:O)}}function Pa(n,s,r){const t=n.Networks??[],l=t.findIndex(f=>f.ChainID===s);if(l===-1)return{...n,Networks:t};const p=t[l],$={...p,Upstreams:p.Upstreams.filter(f=>f.ID!==r)};return{...n,Networks:t.map((f,N)=>N===l?$:f)}}function Ra(n,s){if(n.length===0)return{level:"ok",sentence:"No machines yet.",machines:[]};const r=n.filter(f=>!f.wire);if(r.length>0){const f=r.map(O=>O.id);return{level:"attention",sentence:f.length===1?"1 machine still needs setup.":`${f.length} machines still need setup.`,machines:f}}const t=s.networks??[],l=f=>{const N=t.find(O=>O.ChainID===f);return N?N.Name:`chain ${f}`},p=Na(n.map(f=>l(f.wire.ChainID))),$=n.length===1?"machine":"machines";return{level:"ok",sentence:`All ${n.length} ${$} healthy — ${Aa(p)}.`,machines:[]}}function La(n,s){const r=s.machines.length?` <span class="verdict-machines">${s.machines.map(t=>`<a href="#/setup/${encodeURIComponent(t)}">${a(t)}</a>`).join(" ")}</span>`:"";n.innerHTML=`
    <div class="verdict-line verdict-${s.level}">
      ${K(s.level==="ok"?"OK":"Attention",s.level==="ok"?"ok":"warn")}
      <strong class="verdict-sentence">${a(s.sentence)}</strong>${r}
    </div>
  `}function Na(n){return[...new Set(n)]}function Aa(n){return n.length<=1?n[0]??"":n.length===2?`${n[0]} and ${n[1]}`:`${n.slice(0,-1).join(", ")} and ${n[n.length-1]}`}const Ba=[{chainId:1,name:"Ethereum"},{chainId:369,name:"PulseChain"}];function It(n){return{ProjectID:"main",BindAddr:"127.0.0.1",Port:4e3,Networks:n,TLS:{Enabled:!0,Hostname:"",CertSource:"internal",CertFile:"",KeyFile:"",HTTPSPort:0,BindAddr:"",ImageRef:""}}}const Da=`<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
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
</defs></svg>`,ce=n=>`<svg class="p-i"><use href="#p-${n}"/></svg>`,Et="run",Pt=1337;function Rt(n){let s=null,r={name:"list"},t=null,l=null,p=null,$=null,f=[],N=null,O=null,_=!1,A=null,j=!1,W=null,B=!1,R=null,b=null,k=null,T=null;n.innerHTML=Da+'<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>';const M=n.querySelector("#p-card");async function q(){try{const c=await lt();s=Ha(c.gateways),t=null}catch(c){t=ue(c)}D()}function D(){M.innerHTML=ne()}function ne(){return t?Ua(t):r.name==="network"?Ga(s,r.chainId,{caps:N,capsBusy:_,tls:A,tlsBusy:j,tlsErr:W,copyFlash:B,error:R}):r.name==="endpoint"?Ja(s,r.chainId,r.upstreamId,{caps:N,capsBusy:_,health:k,copyFlash:B,error:b}):Ma(s,l,p,f)}async function V(c,u){_=!0,D();try{N=await Ut(c,u),O=c}catch{N=null,O=c}_=!1,D()}async function se(c,u){if(!(!u&&T===c&&k)){D();try{k=await Ht(c),T=c}catch{k=null,T=c}D()}}async function pe(c,u){var I;const m=(I=c.networks)==null?void 0:I.find(v=>v.chainId===u);if(await Te({title:"Remove network",body:`Stop serving ${(m==null?void 0:m.name)??`chain ${u}`}?`,confirmLabel:"Remove",danger:!0})){R=null,D();try{await Se(c.id,Ea(c.config,u))}catch(v){R=`Could not remove the network: ${ue(v)}`,D();return}r={name:"list"},D(),await w(c.id)}}async function Q(c,u,m){var z;const P=(z=c.networks)==null?void 0:z.find(te=>te.chainId===u),I=oe(c,u,m);if(await Te({title:"Remove endpoint",body:`Stop routing to ${(I==null?void 0:I.label)??"this endpoint"}? The gateway keeps balancing across whatever else remains on ${(P==null?void 0:P.name)??`chain ${u}`}.`,confirmLabel:"Remove",danger:!0})){b=null,D();try{await Se(c.id,Pa(c.config,u,m))}catch(te){b=`Could not remove the endpoint: ${ue(te)}`,D();return}r={name:"network",chainId:u},D(),await w(c.id)}}we(M,(c,u)=>{ie(c,u)});async function ie(c,u){if(c==="setup"){if(l)return;await H();return}if(c==="power"){if(!s||l)return;const m=Ft(s);if(m.tone==="blocked")return;if(s.status.State==="running"&&m.actions.includes("stop")){await g(s.id,"stop");return}if(m.actions.includes("start")){await g(s.id,"start");return}if(m.actions.includes("create")){await w(s.id);return}return}if(c==="open-network"){r={name:"network",chainId:Number(u.dataset.chainId)},R=null,A=null,W=null,D(),s&&O!==s.id&&V(s.id,!1);return}if(c==="back"){r={name:"list"},D();return}if(c==="back-to-network"){const m=Number(u.dataset.chainId);r=Number.isFinite(m)?{name:"network",chainId:m}:{name:"list"},b=null,D();return}if(c==="add-network"){if(!s||l)return;await F(s);return}switch(c){case"gw-start":case"gw-stop":case"gw-restart":s&&!l&&await g(s.id,c.slice(3));return;case"gw-create":case"gw-recreate":s&&!l&&await w(s.id);return;case"gw-wipe":s&&!l&&await G(s);return;case"copy-url":{const m=u.dataset.url??"";if(!m)return;await Oe(m)&&(B=!0,D(),window.setTimeout(()=>{B=!1,D()},1200));return}case"verify-tls":{if(!s||j)return;j=!0,W=null,D();try{A=await Dt(s.id)}catch(m){W=ue(m)}j=!1,D();return}case"open-endpoint":{const m=Number(u.dataset.chainId),P=u.dataset.upstreamId??"";if(!Number.isFinite(m)||!P)return;r={name:"endpoint",chainId:m,upstreamId:P},b=null,D(),s&&O!==s.id&&V(s.id,!1),s&&T!==s.id&&se(s.id,!1);return}case"add-endpoint":{if(!s||l||r.name!=="network")return;E(s,r.chainId);return}case"remove-network":{if(!s||l||r.name!=="network")return;await pe(s,r.chainId);return}case"rename-endpoint":{if(!s||l||r.name!=="endpoint")return;const m=oe(s,r.chainId,r.upstreamId);if(!m)return;me(s.id,r.chainId,m.id,m.label);return}case"edit-address":{if(!s||l||r.name!=="endpoint")return;const m=oe(s,r.chainId,r.upstreamId);if(!m||m.kind!=="external")return;C(s.id,r.chainId,m.id,m.endpoint);return}case"remove-endpoint":{if(!s||l||r.name!=="endpoint")return;await Q(s,r.chainId,r.upstreamId);return}case"recheck":{if(!s)return;const m=[V(s.id,!0),q()];r.name==="endpoint"&&m.push(se(s.id,!0)),await Promise.all(m);return}default:return}}function oe(c,u,m){var P,I,v;return(v=(I=(P=c.networks)==null?void 0:P.find(z=>z.chainId===u))==null?void 0:I.upstreams)==null?void 0:v.find(z=>z.id===m)}function le(c,u,m){var P,I;return(I=(P=c.config.Networks)==null?void 0:P.find(v=>v.ChainID===u))==null?void 0:I.Upstreams.find(v=>v.ID===m)}function Z(c){const u=Me();if(!u)return;const m=document.createElement("p");m.className="error small",m.textContent=c,u.appendChild(m)}function me(c,u,m,P){re(`
        <h2>Rename endpoint</h2>
        <label>
          Name
          <input type="text" id="ep-rename-input" autocomplete="off" spellcheck="false" value="${a(P)}" />
        </label>
        <p class="muted small">Clear it to fall back to the automatic name.</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-rename-save">Save</button>
        </div>
      `,v=>{if(v==="cancel"){Y();return}v==="save"&&ve(c,u,m)});const I=document.getElementById("ep-rename-input");I==null||I.focus(),I==null||I.select()}async function ve(c,u,m){if(!s)return;const P=le(s,u,m);if(!P){Y();return}const I=document.getElementById("ep-rename-input"),v=document.getElementById("ep-rename-save"),z=(I==null?void 0:I.value.trim())??"";I&&(I.disabled=!0),v&&(v.disabled=!0,v.textContent="Saving…");const te={...P,Name:z||void 0};try{await Se(c,at(s.config,u,te))}catch(fe){Z(`Could not rename the endpoint: ${ue(fe)}`),I&&(I.disabled=!1),v&&(v.disabled=!1,v.textContent="Save");return}Y(),await w(c)}function C(c,u,m,P){re(`
        <h2>Edit endpoint address</h2>
        <p class="muted small">http://, https://, ws:// or wss://.</p>
        <label>
          URL
          <input type="text" id="ep-addr-input" autocomplete="off" spellcheck="false" value="${a(P)}" placeholder="https://rpc.example.com" />
        </label>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-addr-save">Save</button>
        </div>
      `,v=>{if(v==="cancel"){Y();return}v==="save"&&h(c,u,m)});const I=document.getElementById("ep-addr-input");I==null||I.focus(),I==null||I.select()}async function h(c,u,m){if(!s)return;const P=document.getElementById("ep-addr-input"),I=document.getElementById("ep-addr-save"),v=(P==null?void 0:P.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(v)){Z("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}const z=le(s,u,m);if(!z){Y();return}P&&(P.disabled=!0),I&&(I.disabled=!0,I.textContent="Saving…");const te={...z,Endpoint:v};try{await Se(c,at(s.config,u,te))}catch(fe){Z(`Could not save the address: ${ue(fe)}`),P&&(P.disabled=!1),I&&(I.disabled=!1,I.textContent="Save");return}Y(),await w(c)}function E(c,u){var m;re(`
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
      `,P=>{if(P==="cancel"){Y();return}P==="add"&&U(c.id,u)}),(m=document.getElementById("ep-add-input"))==null||m.focus()}async function U(c,u){if(!s)return;const m=document.getElementById("ep-add-input"),P=document.getElementById("ep-add-save"),I=(m==null?void 0:m.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(I)){Z("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}m&&(m.disabled=!0),P&&(P.disabled=!0,P.textContent="Adding…");const v={ID:crypto.randomUUID(),Kind:"external",Endpoint:I,Local:!1,RecentOnly:!1,Name:xa(I)};try{await Se(c,at(s.config,u,v))}catch(z){Z(`Could not add the endpoint: ${ue(z)}`),m&&(m.disabled=!1),P&&(P.disabled=!1,P.textContent="Add endpoint");return}Y(),await w(c)}async function F(c){l="add-network",p=null,D();let u;try{u=(await Ie()).networks??[]}catch(I){l=null,p=`Could not load the network catalog: ${ue(I)}`,D();return}l=null,D();const m=new Set((c.networks??[]).map(I=>I.chainId)),P=u.filter(I=>!m.has(I.ChainID)).map(I=>({chainId:I.ChainID,name:I.Name}));if(m.has(Pt)||P.push({chainId:Pt,name:"Devnet"}),P.length===0){p="Every network valve's catalog knows about is already configured on this gateway.",D();return}re(`
        <h2>Add a network</h2>
        <ul class="plain-list rpc-picker">
          ${P.map(I=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="pick:${I.chainId}">
                <span>${a(I.name)}</span>
                <span class="muted small">chain ${I.chainId}</span>
              </button>
            </li>`).join("")}
        </ul>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,I=>{if(I==="cancel"){Y();return}if(I.startsWith("pick:")){const v=Number.parseInt(I.slice(5),10);if(!Number.isFinite(v))return;Y(),J(c.id,v)}})}async function J(c,u){if(!s||l)return;l="create",p=null,D();let m;try{m=((await rt(c,u)).endpoints??[]).filter(v=>!v.alreadyAdded).map(v=>v.url)}catch(I){l=null,p=`Could not read valve's known set for chain ${u}: ${ue(I)}`,D();return}if(m.length===0){l=null,p=`valve has no measured endpoints for chain ${u} yet, so there was nothing to add.`,D();return}const P=m.map((I,v)=>({ID:`public-${u}-${v+1}`,Kind:"external",Endpoint:I,Local:!1,RecentOnly:!1}));try{await Se(c,Ia(s.config,u,P))}catch(I){l=null,p=`Could not add the network: ${ue(I)}`,D();return}l=null,await w(c,()=>{r={name:"network",chainId:u},D()})}async function g(c,u){if(!l){l=u,p=null,D();try{await Mt(c,u)}catch(m){p=`${u} failed: ${ue(m)}`}l=null,await q()}}async function w(c,u){if(l)return;l="create",p=null,D();let m;try{m=await ot(c)}catch(P){p=ue(P),l=null,D();return}$==null||$(),$=qe(m.targetId,P=>{(P.err||P.stepId===Et&&P.done)&&($==null||$(),$=null,l=null,P.err&&(p=`Provisioning failed: ${P.err}`),q().then(()=>{P.err||u==null||u()}))})}async function H(){if(l)return;l="setup",p=null,f=[],D();const c=v=>{f=[...f,v],D()},u=(v,z)=>{l=null,p=z?`${v} — ${z}`:v,D()};c("Preparing your endpoint…");try{(await Ee()).some(z=>z.id==="local")||await st({id:"local",mode:"local"})}catch(v){u(`Could not register this machine: ${ue(v)}`,He(v));return}try{const v=await At("local");if(!v.docker.reachable){u(v.docker.detail||"A gateway runs as a container, and no Docker engine answered on this machine.",v.docker.hint||"Start Docker Desktop, OrbStack or colima, then try again.");return}}catch(v){u(`Could not check Docker on this machine: ${ue(v)}`,He(v));return}c("Creating the gateway…");let m="default";try{m=(await Bt({id:m,placement:{targetId:"local",backend:"docker"},config:It([])})).id}catch(v){u(`Could not create the gateway: ${ue(v)}`,He(v));return}c("Adding Ethereum and PulseChain endpoints…");const P=[];for(const{chainId:v}of Ba)try{const te=((await rt(m,v)).endpoints??[]).filter(fe=>!fe.alreadyAdded).map(fe=>fe.url);if(te.length===0)continue;P.push({ChainID:v,Upstreams:te.map((fe,Ne)=>({ID:`public-${v}-${Ne+1}`,Kind:"external",Endpoint:fe,Local:!1,RecentOnly:!1}))})}catch(z){u(`Could not read valve's set for chain ${v}: ${ue(z)}`,He(z));return}if(P.length===0){u("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");return}try{await Se(m,It(P))}catch(v){u(`Could not save the endpoints: ${ue(v)}`,He(v));return}c("Starting the gateway… the first run pulls the eRPC and Caddy images.");let I;try{I=await ot(m)}catch(v){u(`Could not start the gateway: ${ue(v)}`,He(v));return}$==null||$(),$=qe(I.targetId,v=>{const z=v.err?`${v.stepId}: ${v.err}`:v.line?`${v.stepId}: ${v.line}`:`${v.stepId}: done`;c(z),(v.err||v.stepId===Et&&v.done)&&($==null||$(),$=null,l=null,v.err&&(p=`Provisioning failed: ${v.err}`),f=[],q())})}async function G(c){if(await Te({title:`Wipe ${c.label}`,body:`This destroys ${c.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,confirmLabel:"Wipe",danger:!0})){l="wipe",p=null,D();try{const m=await Ot(c.id);m.error&&(p=m.error)}catch(m){p=`wipe failed: ${ue(m)}`}l=null,await q()}}return q(),()=>{$==null||$()}}function Ha(n){return!n||n.length===0?null:n.find(s=>s.placement.targetId==="local")??n[0]}function ue(n){return n instanceof Error?n.message:String(n)}function He(n){return n instanceof Ce?n.hint:void 0}function Ua(n){return`<div class="p-band" style="padding:16px;color:var(--red)">${a(n)}</div>`}function Ma(n,s,r,t){var $;if(n===null)return Oa(s,r,t);const l=Ft(n),p=($=n==null?void 0:n.networks)!=null&&$.length?n.networks.map((f,N)=>Ka(n,f,N>0)).join(""):"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${a(l.sub)}</span>
    </div>
    <div class="p-band">
      ${qa(n,l,s,r)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${p}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${ce("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `}function Oa(n,s,r){const t=n==="setup",l=s?`<div class="p-emptyerr">${a(s)}</div>`:"",p=r.length?`<div class="p-setup-log" aria-live="polite">${r.map($=>`<div>${a($)}</div>`).join("")}</div>`:"";return`
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${t?" disabled":""}>
        <div class="p-pbtn off big${t?" busy":""}">${ce("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${l}
      ${p}
    </div>
  `}function Fa(n,s){return s.tone==="blocked"?null:n.status.State==="running"&&s.actions.includes("stop")?"stop":s.actions.includes("start")?"start":s.actions.includes("create")?"create":null}const ja={start:"Start",stop:"Stop",restart:"Restart",create:"Create",recreate:"Recreate",wipe:"Wipe"},Lt={restart:"refresh",recreate:"refresh",wipe:"trash"};function qa(n,s,r,t){const l=s.tone==="blocked"?s.blocked??"":s.sub,p=r?" busy":"",$=t?`<div class="p-ps" style="color:var(--red)">${a(t)}</div>`:"",f=s.tone==="blocked"&&(n!=null&&n.hint)?`<div class="p-ps">${a(n.hint)}</div>`:"",N=`
    <div class="p-power${p}" data-action="power">
      <div class="p-pbtn ${s.tone}">${ce("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${a(s.label)}</div>
        <div class="p-ps"${s.tone==="blocked"?' style="color:var(--red)"':""}>${a(l)}</div>
        ${f}
        ${$}
      </div>
    </div>
  `,O=n?Wa(n,s,r):"";return N+O}function Wa(n,s,r){const t=Fa(n,s),l=(n.actions??[]).filter($=>$!==t);return l.length===0?"":`<div class="p-chips">${l.map($=>{const f=ja[$]??$,N=Lt[$]?ce(Lt[$]):"";return`<button type="button" class="p-chip${$==="wipe"?" danger":""}" data-action="gw-${$}" data-gid="${a(n.id)}"${r?" disabled":""}>${N}${a(f)}</button>`}).join("")}</div>`}const pt={http:"globe",ws:"ws",archive:"archive",trace:"trace"};function _a(n){return n.map(s=>`<svg class="p-i${s.hot?" hot":s.lit?" on":""}"><use href="#p-${pt[s.key]}"/></svg>`).join("")}function Ka(n,s,r){const t=Ye({running:n.status.State==="running",serviceable:s.serviceable}),l=ut({});return`
    <div class="p-row${r?" p-rowdiv":""}" data-action="open-network" data-chain-id="${s.chainId}">
      <span class="p-lead"><span class="p-dot ${t}"></span></span>
      <span class="p-nm">${a(s.name)}</span>
      <span class="p-caps">${_a(l)}</span>
      <span class="p-chev">${ce("chevR")}</span>
    </div>
  `}function jt(n,s){var r;return s==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(r=(n.capabilities??[]).find(t=>t.key===s))==null?void 0:r.status}function Va(n,s,r){const t=((n==null?void 0:n.endpoints)??[]).filter(p=>p.chainId===s&&r.includes(p.upstream)),l={};for(const p of["http","ws","archive","trace"])t.some($=>jt($,p)==="supported")&&(l[p]="supported");return l}function Ga(n,s,r){var q;const t=(q=n==null?void 0:n.networks)==null?void 0:q.find(D=>D.chainId===s);if(!n||!t)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${ce("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${s}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;const l=n.status.State==="running",p=Ye({running:l,serviceable:t.serviceable}),$=t.upstreams??[],f=r.tls??n.tls.verification??null,N=(f==null?void 0:f.ok)===!0,O=r.tlsBusy?"Verifying…":N?`Verified ${f?new Date(f.at).toLocaleString():""}`:"Verify HTTPS now",_=r.tlsErr?`<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${a(r.tlsErr)}</div>`:"",A=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${N?"green":"dim"}" data-action="verify-tls" title="${a(O)}">${ce("lock")}</span>
          <span class="p-ic ${r.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(t.url??"")}" title="Copy the gateway URL">${ce("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${a(t.url||"—")}</div>
      ${_}
    </div>
  `,j=$.map((D,ne)=>{const V=Ye({running:l,serviceable:!D.problem});return`
        <div class="p-row${ne>0?" p-rowdiv":""}" data-action="open-endpoint" data-chain-id="${t.chainId}" data-upstream-id="${a(D.id)}">
          <span class="p-lead"><span class="p-dot ${V}"></span></span>
          <span class="p-nm">${a(D.label)}</span>
          <span class="p-chev">${ce("chevR")}</span>
        </div>
      `}).join(""),W=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${$.length}</span></div>
      ${j}
      <div class="p-row${$.length>0?" p-rowdiv":""} addr" data-action="add-endpoint">
        <span class="p-lead">${ce("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `,B=Va(r.caps,s,$.map(D=>D.id)),R=ut(B),b=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${r.capsBusy&&!r.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':`<div class="p-caprow">${R.map(D=>`<span class="p-capitem${D.lit?" lit":""}">${ce(pt[D.key])}${a(D.label)}</span>`).join("")}</div>`}
    </div>
  `,k=l?t.serviceable?"Healthy":"Unserviceable":"Stopped",T=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${ce("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${p}"></span> ${a(k)}</span></div>
    </div>
  `,M=r.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(r.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${ce("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${p}"></span> <span class="p-nmtxt">${a(t.name)}</span></span>
    </div>
    ${A}
    ${W}
    ${b}
    ${T}
    ${M}
    <div class="p-band p-remove" data-action="remove-network">${ce("trash")} Remove network</div>
  `}function za(n,s,r){const t=((n==null?void 0:n.endpoints)??[]).find(p=>p.chainId===s&&p.upstream===r);if(!t)return{};const l={};for(const p of["http","ws","archive","trace"])jt(t,p)==="supported"&&(l[p]="supported");return l}function Ja(n,s,r,t){var T,M,q;const l=(T=n==null?void 0:n.networks)==null?void 0:T.find(D=>D.chainId===s),p=(M=l==null?void 0:l.upstreams)==null?void 0:M.find(D=>D.id===r);if(!n||!l||!p)return`
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${ce("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Endpoint</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This endpoint is no longer configured.</div>
    `;const $=n.status.State==="running",f=Ye({running:$,serviceable:!p.problem}),N=p.kind==="external",O=`
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Address</span>
        <span class="p-acts"><span class="p-ic ${t.copyFlash?"green":"accent"}" data-action="copy-url" data-url="${a(p.endpoint)}" title="Copy the endpoint URL">${ce("copy")}</span></span>
      </div>
      <div class="p-gwurl"${N?' data-action="edit-address" style="cursor:text"':""}>${a(p.endpoint||"—")}</div>
    </div>
  `,_=za(t.caps,s,r),A=ut(_),j=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${t.capsBusy&&!t.caps?'<div class="p-caprow" style="color:var(--dim2)">probing…</div>':`<div class="p-caprow">${A.map(D=>`<span class="p-capitem${D.lit?" lit":""}">${ce(pt[D.key])}${a(D.label)}</span>`).join("")}</div>`}
    </div>
  `,W=$?p.problem?p.problem:"Healthy":"Stopped",B=(((q=t.health)==null?void 0:q.endpoints)??[]).find(D=>D.chainId===s&&D.upstream===r),R=B&&B.scored&&B.headLag>0?`<div class="p-srow"><span class="p-k">Chain head</span><span class="p-v">behind ${de(B.headLag)} block${B.headLag===1?"":"s"}</span></div>`:"",b=`
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${ce("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${f}"></span> ${a(W)}</span></div>
      ${R}
    </div>
  `,k=t.error?`<div class="p-band" style="padding:10px 16px;color:var(--red)">${a(t.error)}</div>`:"";return`
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${s}">${ce("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${f}"></span> <span class="p-nmtxt">${a(p.label)}</span> <span class="p-pen" data-action="rename-endpoint">${ce("pencil")}</span></span>
    </div>
    ${O}
    ${j}
    ${b}
    ${k}
    <div class="p-band p-remove" data-action="remove-endpoint">${ce("trash")} Remove endpoint</div>
  `}function Ya(n,s){let r=!1,t=[],l=null,p=!1,$=!1;n.innerHTML=`<h1>Security: ${a(s)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${he()}</div>`;const f=n.querySelector("#sec-body"),N=n.querySelector("#sec-footer");we(n,(B,R)=>{var b;if(B==="rerun")_();else if(B==="toggle")(b=R.closest(".check-item"))==null||b.classList.toggle("expanded");else if(B==="copy"){const k=R.dataset.copy;k&&W(R,k)}}),O();async function O(){let B,R;try{const[k,T]=await Promise.all([Ee(),Ie()]);B=k.find(M=>M.id===s),R=T}catch(k){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${a(String(k))}</p>`;return}if(r)return;if(!B){f.innerHTML=`<p class="error">Target "${a(s)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!B.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(s)}">Run the setup wizard →</a></p>`;return}const b=R==null?void 0:R.networks.find(k=>k.ChainID===B.wire.ChainID);b&&(N.innerHTML=he(b.Name,b.LearnURL)),await _()}async function _(){p=!0,l=null,A();try{t=await zn(s),$=!0}catch(B){l=String(B instanceof Error?B.message:B)}p=!1,r||A()}function A(){f.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(s)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${p?"disabled":""}>${p?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${a(l)}</p>`:""}
      ${!$&&p?'<p class="muted">Loading…</p>':t.length?`<ul class="check-list">${t.map(j).join("")}</ul>`:$?'<p class="muted">No checks returned.</p>':""}
    `}function j(B){const R=B.Status==="pass"?"ok":B.Status==="fail"?"bad":B.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${K(B.Status,R)}
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
    `}async function W(B,R){const b=await Oe(R),k=B.textContent;B.textContent=b?"Copied!":"Copy failed",setTimeout(()=>{r||(B.textContent=k)},1500)}return()=>{r=!0}}const Za=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}],ct="VALVE_API_KEY";function Xa(n){return n===ct?"Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.":`Fills the <code>\${${a(n)}}</code> slot wherever an endpoint URL carries one.`}function Qa(n){let s=!1,r=!1,t=!1,l=null,p=!1,$=null,f=null;const N=new Set,O=new Map;let _="",A="";n.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${he()}`;const j=n.querySelector("#settings-body");we(n,(T,M)=>{if(T==="save"&&k(),T==="clear-key"){if(!$)return;r=!0;const q=n.querySelector("#ai-key");q&&(q.value=""),b($)}if(T==="clear-provider-key"){const q=M.dataset.key;if(!$||!q)return;N.add(q),O.set(q,""),p=!1,b($)}}),dt(n,(T,M)=>{T!=="ai-provider"||!$||(f=M,p=!1,b($))}),W();async function W(){try{const T=await ia();if(s)return;$=T,b(T)}catch(T){if(s)return;j.innerHTML=`<p class="error">Failed to load settings: ${a(String(T))}</p>`}}function B(T){const q=(Array.isArray(T.providerKeysSet)?T.providerKeysSet:[]).filter(D=>D!==ct).sort();return[ct,...q]}function R(T,M){const q=a(T);return`
      <div class="pk-row">
        <label>
          <code>${q}</code>
          <input class="provider-key" data-key="${q}" type="password" autocomplete="off"
                 placeholder="${M?"•••••••• (leave blank to keep)":"no key set"}" />
        </label>
        <p class="muted small">${Xa(T)}</p>
        ${M?`<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${q}">Clear saved key</button>`:""}
      </div>`}function b(T){var pe;const M=f??T.aiProvider,q=Array.isArray(T.providerKeysSet)?T.providerKeysSet:[],D=B(T).map(Q=>R(Q,q.includes(Q))).join("");j.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${it("ai-provider",Za.map(Q=>({value:Q.value,label:Q.label})),M)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${T.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${T.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${D}
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
            <input id="ref-rpc-base" type="text" value="${a(T.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${l?`<p class="error">${a(l)}</p>`:""}
        ${p?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${t?"disabled":""}>${t?"Saving…":"Save"}</button>
      </form>
    `;const ne=n.querySelector("#ai-key");ne==null||ne.addEventListener("input",()=>{r=!0,p=!1}),(pe=n.querySelector("#ref-rpc-base"))==null||pe.addEventListener("input",()=>{p=!1}),n.querySelectorAll("input.provider-key").forEach(Q=>{const ie=Q.dataset.key;if(!ie)return;const oe=O.get(ie);oe!==void 0&&(Q.value=oe),Q.addEventListener("input",()=>{N.add(ie),O.set(ie,Q.value),p=!1})});const V=n.querySelector("#pk-new-value");V&&(V.value=A),V==null||V.addEventListener("input",()=>{A=V.value,p=!1});const se=n.querySelector("#pk-new-name");se==null||se.addEventListener("input",()=>{_=se.value,p=!1})}async function k(){const T=n.querySelector("#ai-key"),M=n.querySelector("#ref-rpc-base");if(!T||!M||!$)return;const q={aiProvider:f??$.aiProvider,refRpcBase:M.value.trim()};r&&(q.aiKey=T.value);const D={};for(const V of N)D[V]=O.get(V)??"";const ne=_.trim();ne&&(D[ne]=A),Object.keys(D).length>0&&(q.providerKeys=D),t=!0,l=null,p=!1,b($);try{const V=await ca(q);if(s)return;$=V,r=!1,N.clear(),O.clear(),_="",A="",t=!1,p=!0,b(V)}catch(V){if(s)return;t=!1,l=String(V instanceof Error?V.message:V),b($)}}return()=>{s=!0}}const es=["http","ws","archive","trace"],ts={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Ue=1337,ns="run",as={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function ss(n){let s=!1,r=null,t=null;const l={},p={},$={},f={},N={},O={},_={},A={},j={},W={},B={},R={},b={},k={},T={};let M="",q=null;n.innerHTML=`
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
  `;const D=n.querySelector("#rpc-body");we(n,(e,o)=>{ln(e,o)}),dt(n,()=>{}),V(),ne();async function ne(){try{const e=await Nt();if(s)return;M=e.os,Z()}catch{}}async function V(){try{const e=await lt();if(s)return;r=e,t=null}catch(e){if(s)return;r=null,t=be(e)}Z();for(const e of(r==null?void 0:r.gateways)??[])se(e.id),pe(e.id,!1)}async function se(e){try{const o=await aa(e);if(s)return;l[e]=o}catch{if(s)return;l[e]=null}Z()}async function pe(e,o){$[e]=o,o&&Z();try{const i=await Ut(e,o);if(s)return;p[e]=i}catch{if(s)return;p[e]=null}$[e]=!1,Z()}function Q(e){return((r==null?void 0:r.gateways)??[]).find(o=>o.id===e)}function ie(e,o){return(e.networks??[]).find(i=>i.chainId===o)}function oe(e,o,i){var y;const d=(((y=l[e])==null?void 0:y.networks)??[]).find(x=>x.chainId===o);return((d==null?void 0:d.upstreams)??[]).find(x=>x.upstream===i)}function le(e,o,i){var d;return(((d=p[e])==null?void 0:d.endpoints)??[]).find(y=>y.chainId===o&&y.upstream===i)}function Z(){if(s)return;if(t){D.innerHTML=`<p class="error">Could not read the gateways: ${a(t)}</p>`;return}if(!r){D.innerHTML='<p class="muted">Loading…</p>';return}const e=r.gateways??[],o=e.length>1,i=(r.targets??[]).some(x=>$t(x.id,e)),d=new Set(e.map(x=>x.placement.targetId)),y=(r.orphans??[]).filter(x=>!d.has(x.targetId));D.innerHTML=`
      ${e.map(x=>C(x,o)).join("")}
      ${e.length===0?ve():""}
      ${y.map(me).join("")}
      ${i?`<div class="card-actions rpc-add-gateway">
               <button class="btn${e.length?" btn-ghost":""}" data-action="add-gateway">
                 Add a gateway${e.length?" on another machine":""}
               </button>
             </div>`:""}
    `}function me(e){const o=`docker rm -f ${e.containerName}`,i=b[e.containerName];return`
      <div class="strip">
        ${m({tone:"warn",text:`${e.containerName} is still running on ${e.targetId}. Its chains were folded into ${e.mergedInto}, but valve-node-app does not stop containers it did not start.`,cmd:o})}
        ${i?m({tone:"bad",text:i}):""}
        <div class="strip-line strip-note">
          <button class="btn btn-ghost btn-tiny" data-action="dismiss-orphan"
                  data-name="${a(e.containerName)}">Dismiss this record</button>
          <span class="muted small">Forgets the record only — the container is never touched from here.</span>
        </div>
      </div>
    `}function ve(){return((r==null?void 0:r.targets)??[]).length===0?`
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
    `}function C(e,o){return`
      ${o?`<h2 class="rpc-machine">${a(e.placement.targetId)}</h2>`:""}
      ${h(e)}
      ${u(e)}
      ${te(e)}
      ${fe(e)}
      ${g(e)}
    `}function h(e){const o=e.status.State==="running",i=e.tls,d=[`on <strong>${a(e.placement.targetId)}</strong>`];return e.status.Image&&d.push(`<code>${a(e.status.Image)}</code>`),d.push(i!=null&&i.enabled?`HTTPS front <code>${a(i.containerName||"caddy")}</code>`:"no HTTPS front"),`
      <div class="rpc-ident">
        ${v(e)}
        <strong>${a(e.label)}</strong>
        ${I(e)}
        <span class="muted small">${d.join(" · ")}</span>
        <span class="rpc-ident-base muted small">${o?`base <code>${a(e.baseUrl)}</code>`:"not serving"}</span>
      </div>
    `}function E(e){const o=e.tls;return o!=null&&o.enabled&&o.rootCaPath&&o.effectiveCertSource==="internal"?o.rootCaPath:null}function U(e){var o;return((o=((r==null?void 0:r.targets)??[]).find(i=>i.id===e.placement.targetId))==null?void 0:o.mode)??""}function F(e){switch(e){case"darwin":return"macOS";case"windows":return"Windows";case"linux":return"Linux";default:return e||"this device"}}function J(e,o,i){switch(e){case"darwin":return`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${o}"`;case"windows":return`certutil -addstore -f ROOT "${o}"`;case"linux":default:return`sudo cp "${o}" /usr/local/share/ca-certificates/valve-node-app-${i}.crt && sudo update-ca-certificates`}}function g(e){const o=j[e.id]??!1,i=((r==null?void 0:r.orphans)??[]).filter(d=>d.targetId===e.placement.targetId);return`
      <section class="card manage-section${o?" open":""}">
        <button type="button" class="manage-head" data-action="toggle-manage"
                data-gid="${a(e.id)}" aria-expanded="${o}">
          <span class="manage-title">Manage gateway</span>
          <span class="manage-status muted small">${w(e,i.length)}</span>
          <span class="manage-caret" aria-hidden="true">▸</span>
        </button>
        ${o?H(e,i):""}
      </section>
    `}function w(e,o){const i=[];return e.status.State!=="running"&&i.push("gateway not running"),o>0&&i.push(`${o} leftover container${o===1?"":"s"}`),i.length===0?"container, settings, certificate":i.join(" · ")}function H(e,o){var i;return`
      <div class="manage-body">
        <div class="rpc-head-actions">
          ${(e.actions??[]).map(d=>z(e,d)).join("")}
          <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(e.id)}"
             title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
          <button class="btn btn-ghost" data-action="reprobe" data-gid="${a(e.id)}"
                  title="Ask every endpoint what it can do, again. This opens real connections to them."
                  ${$[e.id]?"disabled":""}>
            ${$[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
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
                 <span class="muted small">a chain is addressed by path, e.g. <code>${a(((i=(e.networks??[])[0])==null?void 0:i.path)??"/main/evm/<chainId>")}</code></span>
               </div>`:`<p class="muted small">Not serving — it will answer on <code>${a(e.baseUrl)}</code> once it is running.</p>`}
        ${G(e)}
        ${o.map(me).join("")}
        ${_[e.id]?tn(e):""}
      </div>
    `}function G(e){const o=E(e);if(!o)return"";const i=U(e)==="local",d=J(M,o,e.id),y=T[e.id];return`
      <div class="strip">
        <div class="strip-line strip-note">
          <span class="strip-text">Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until that authority's root is trusted. The root is on ${a(e.placement.targetId)} at:</span>
          <code class="strip-cmd">${a(o)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(o)}">Copy path</button>
        </div>
        ${i?`<div class="strip-line strip-note">
                 <span class="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
                 <button class="btn btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${k[e.id]?"disabled":""}>
                   ${k[e.id]?'<span class="spinner" aria-label="installing"></span>':"Trust on this machine"}
                 </button>
               </div>`:""}
        ${y?c(y):""}
        <div class="strip-line strip-note">
          <span class="strip-text">The certificate must be trusted on whatever device opens the URL — ${i?"if that is a different device (a phone, another laptop), copy the root above to it and run":"this gateway runs elsewhere, so on the device you browse from run"}${M?` (${a(F(M))})`:""}:</span>
          <code class="strip-cmd">${a(d)}</code>
          <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(d)}">Copy command</button>
        </div>
      </div>
    `}function c(e){return e.ok?`<div class="strip-line strip-note"><span class="strip-text">${a(e.message)}</span></div>`:`
      <div class="strip-line strip-warn">
        <span class="strip-text">${a(e.message)}</span>
        ${e.ranCommand?`<code class="strip-cmd">${a(e.ranCommand)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.ranCommand)}">Copy</button>`:""}
      </div>
    `}function u(e){const o=[];e.error&&o.push({tone:"bad",text:`This gateway could not be read: ${e.error}${e.hint?` — ${e.hint}`:""}`}),e.blocked&&o.push({tone:"warn",text:e.blocked});for(const d of e.warnings??[])o.push({tone:"warn",text:d});o.push(...P(e));const i=N[e.id];return i&&o.push({tone:"bad",text:i}),o.length===0?"":`<div class="strip">${o.map(m).join("")}</div>`}function m(e){return`
      <div class="strip-line strip-${e.tone}">
        <span class="strip-text">${a(e.text)}</span>
        ${e.cmd?`<code class="strip-cmd">${a(e.cmd)}</code>
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(e.cmd)}">Copy</button>`:""}
      </div>
    `}function P(e){var y,x;const o=e.tls;if(!(o!=null&&o.enabled))return[];const i=[];o.fallback&&i.push({tone:"warn",text:o.fallback}),o.error?i.push({tone:"warn",text:`HTTPS front: ${o.error}`}):((y=o.status)==null?void 0:y.State)!=="running"&&i.push({tone:"warn",text:`The HTTPS front is ${((x=o.status)==null?void 0:x.State)??"unknown"}, so nothing answers on ${o.url??"its https URL"} even if the gateway itself is up.`,cmd:o.containerName?`docker start ${o.containerName}`:void 0});const d=W[e.id]??o.verification??null;return d&&(!d.ok||!d.subscriptionsOk)&&i.push({tone:d.ok?"warn":"bad",text:`${d.summary} Checked ${new Date(d.at).toLocaleString()} — open Settings for the full check.`}),d!=null&&d.expiryWarning&&i.push({tone:"warn",text:d.expiryWarning}),i}function I(e){switch(e.status.State){case"running":return K("running","ok");case"created-but-stopped":return K("stopped","warn");case"not-created":return K("not created","neutral");default:return K("unknown","bad")}}function v(e){return e.status.State==="running"?xe("ok"):e.status.State==="unknown"?xe("bad"):xe("neutral")}function z(e,o){const i=as[o];if(!i)return"";const d=f[e.id];return`
      <button class="${i.className}" data-action="gw-${o}" data-gid="${a(e.id)}"
              title="${a(i.title)}" ${d?"disabled":""}>
        ${d===o?'<span class="spinner" aria-label="working"></span>':a(i.label)}
      </button>
    `}function te(e){const o=O[e.id]??[];return o.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${a(e.placement.targetId)}</p>
        <pre class="step-log">${a(o.join(`
`))}</pre>
      </div>
    `}function fe(e){const o=Ne(e.networks??[]),i=o.some(d=>d.chainId===Ue);return o.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${a(e.id)}">Add a network</button>
            ${ht(e,i)}
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        <div class="chains">
          ${o.map(d=>We(e,d)).join("")}
        </div>
        ${Ae(e,i)}
        ${en(e)}
      </div>
    `}function Ne(e){const o=e.filter(d=>d.chainId!==Ue),i=e.filter(d=>d.chainId===Ue);return[...o,...i]}function We(e,o){const i=_t(o),d=o.chainId===Ue,y=`${e.id}:${o.chainId}`,x=A[y]??!1,L=i.tone==="ok"?"healthy":"attention";return`
      <section class="chain chain-${i.tone}${d?" chain-devnet":""}">
        <div class="chain-head">
          <span class="chain-name">${a(o.name)}</span>
          <code class="chain-key">evm:${o.chainId}</code>
          ${d?'<span class="chain-tag">local test chain (devnet)</span>':""}
          ${K(L,i.tone)}
          <span class="chain-right">
            <button class="btn btn-ghost btn-tiny" data-action="toggle-chain-detail"
                    data-key="${a(y)}" aria-expanded="${x}">
              ${x?"Hide details":"Details"}
            </button>
          </span>
        </div>
        ${Xe(e,o)}
        ${x?_e(e,o,i):""}
      </section>
    `}function Xe(e,o){if(!o.url)return`<p class="chain-connect-none muted small">${e.status.State!=="running"?"No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”.":"Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}</p>`;const i=E(e);return`
      <div class="chain-connect">
        <code class="endpoint-url">${a(o.url)}</code>
        <button class="btn btn-tiny" data-action="copy" data-copy="${a(o.url)}"
                title="Copy ${a(o.url)}">Copy URL</button>
        ${i?`<span class="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
               ${U(e)==="local"?`<button class="btn btn-ghost btn-tiny" data-action="trust-cert" data-gid="${a(e.id)}" ${k[e.id]?"disabled":""}
                              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet.">${k[e.id]?"Trusting…":"Trust on this machine"}</button>`:""}
               <button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${a(i)}"
                       title="Copy the path to Caddy's root certificate. Install it on ${a(e.placement.targetId)} and in the trust store of any device that will call this URL, and the warning goes away.">Copy cert path</button>
               ${T[e.id]?`<span class="chain-cert muted small">${a(T[e.id].ok?"Trusted — reload your wallet or browser.":T[e.id].message)}</span>`:""}`:""}
      </div>
    `}function _e(e,o,i){const d=o.upstreams??[];return`
      <div class="chain-detail">
        <p class="chain-verdict${i.why?" chain-verdict-why":""}"${i.why?` title="${a(i.why)}"`:""}>${i.html}</p>
        <div class="chain-detail-bar">
          ${Wt(d.length,i.tone,o.knownSetSize)}
          <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                  data-gid="${a(e.id)}" data-chain="${o.chainId}">+ Endpoint</button>
          <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                  data-gid="${a(e.id)}" data-chain="${o.chainId}">Remove</button>
        </div>
        ${Gt(e,o)}
        ${(o.warnings??[]).map(y=>`<p class="chain-note">${a(y)}</p>`).join("")}
      </div>
    `}function Ae(e,o){const i=p[e.id],d=i!=null&&i.at?`probed ${a(mt(i.at))}`:"not probed yet";return`
      <div class="chains-foot">
        <button class="btn btn-ghost btn-tiny" data-action="add-chain" data-gid="${a(e.id)}">+ Network</button>
        ${ht(e,o)}
        <span class="chains-foot-gap"></span>
        <span class="muted small">${d}</span>
        <button class="btn btn-ghost btn-tiny" data-action="reprobe" data-gid="${a(e.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${$[e.id]?"disabled":""}>
          ${$[e.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
      </div>
    `}function ht(e,o){return o?"":`<button class="btn btn-ghost btn-tiny" data-action="add-devnet" data-gid="${a(e.id)}"
                    title="Add a throwaway local test chain (evm:${Ue}) fronted by this gateway. Optional — real chains only by default.">Add a local devnet</button>`}function Wt(e,o,i){const d=i>0,y=d?i:e,x=Math.min(e,y);let L="";for(let De=0;De<y;De++)L+=`<span class="seg${De<x?` seg-on seg-${o}`:""}"></span>`;const S=d&&e>i,X=d?S?`${e} (set is ${i})`:`${e} of ${i}`:`${e}`,ae=`${e} upstream${e===1?"":"s"} configured`,ye=d?`${ae}${S?`, ${e-i} beyond the set`:""}. valve's set for this chain is ${i}.`:`${ae}. valve has not measured a set for this chain, so there is nothing to count it against.`;return`
      <span class="segs" title="${a(ye)}">${L}</span>
      <span class="segs-n">${X}</span>
    `}function _t(e){const o=e.upstreams??[];if(o.length===0)return{tone:"bad",html:"No endpoint yet, so there is nowhere for calls on this path to go."};if(!e.serviceable)return{tone:"bad",html:"No upstream here can be dialed, so every call on this path fails."};if(!o.some(Kt)){const d=Vt(o);return{tone:"warn",html:`No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain${d.length?` — every upstream here is configured as ${d.map(x=>`<code>${a(x)}://</code>`).join(" or ")}.`:"."}`,why:"eRPC infers WebSocket from the endpoint's scheme and has no separate setting, so a chain configured entirely with http:// or https:// upstreams refuses every eth_subscribe — even where the same host would accept a wss:// connection. That is why an endpoint below can be tagged WS and this still be true."}}if(o.length===1)return{tone:"warn",html:"One endpoint, so this chain stops when it does."};if(!o.some(d=>d.local))return{tone:"warn",html:"No node of your own serves this chain."};const i=o.filter(d=>!!d.problem);if(i.length>0){const d=o.length-i.length;return{tone:"warn",html:`${i.length} of these ${o.length} endpoints ${i.length===1?"is":"are"} unusable, so ${d===1?"only one can":`only ${d} can`} actually answer — the segments above count what is configured, not what is working.`}}return{tone:"ok",html:`${o.length} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still answer.`}}function Kt(e){return/^wss?:\/\//i.test((e.endpoint??"").trim())}function Vt(e){const o=new Set;for(const i of e){const d=/^([a-z][a-z0-9+.-]*):\/\//i.exec((i.endpoint??"").trim());d&&o.add(d[1].toLowerCase())}return[...o].sort()}function Gt(e,o){const i=o.upstreams??[];return i.length===0?"":`<ul class="ups">${i.map(d=>zt(e,o,d)).join("")}</ul>`}function zt(e,o,i){const d=`${e.id}|${o.chainId}|${i.id}`,y=i.actions??[];return`
      <li class="up${i.problem?" up-bad":""}">
        <div class="up-what">
          ${i.problem?xe("bad"):xe("ok")}
          <span class="up-label">${a(i.label)}</span>
          ${Jt(i)}
        </div>
        <code class="up-url">${a(i.endpoint||"—")}</code>
        <div class="up-caps">${Yt(e,o,i)}</div>
        <div class="up-share">${Qt(e,o,i)}</div>
        <div class="up-acts">
          ${y.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${a(d)}"
                         data-target="${a(i.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${f[e.id]?"disabled":""}>
                   ${f[e.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${a(d)}">Remove</button>
        </div>
        ${i.problem?`<div class="up-problem error small">${a(i.problem)}</div>`:""}
      </li>
    `}function Jt(e){return e.problem?K("unusable","bad"):e.recentOnly?K("recent blocks","warn"):e.local?K("yours","ok"):K("public","neutral")}function ft(e,o){var i;if(e)return o==="http"?e.unprobeable?"inconclusive":e.reachable?"supported":"unsupported":(i=(e.capabilities??[]).find(d=>d.key===o))==null?void 0:i.status}function Yt(e,o,i){const d=le(e.id,o.chainId,i.id);return d?d.unprobeable?`<span class="caps-none" title="${a(d.unprobeable)}">not probeable from here</span>`:`<span class="caps">${es.map(y=>Zt(e,o,d,y)).join("")}</span>`:`<span class="muted small">${p[e.id]===void 0?"probing…":"—"}</span>`}function Zt(e,o,i,d){const y=(i.capabilities??[]).find(ae=>ae.key===d),x=ft(i,d)??"inconclusive",L=ts[d]??d.toUpperCase();let S="cap";x==="unsupported"?S=Xt(e,o,d)?"cap missing":"cap off":x==="inconclusive"?S="cap unknown":x==="inconsistent"&&(S="cap mixed");const X=y!=null&&y.detail?`${y.label}: ${y.detail}`:d==="http"&&i.reachDetail?`Answers JSON-RPC over HTTP: ${i.reachDetail}`:`${L}: no verdict`;return`<span class="${S}" title="${a(X)}">${a(L)}</span>`}function Xt(e,o,i){const d=(o.upstreams??[]).map(y=>le(e.id,o.chainId,y.id)).filter(y=>!!y&&!y.unprobeable);return d.length>0&&d.every(y=>ft(y,i)==="unsupported")}function Qt(e,o,i){const d=l[e.id];if(d===void 0)return'<span class="muted small">reading…</span>';if(d===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!d.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const y=oe(e.id,o.chainId,i.id),x=(d.networks??[]).find(ye=>ye.chainId===o.chainId);if(!y||!x||x.attributed===0)return'<span class="muted small">no traffic yet</span>';const L=Math.round(y.actual*100),S=Math.round(y.intended*100),X=y.diverged?i.local?"warn":"":"ok",ae=`${y.succeeded.toLocaleString()} of ${x.attributed.toLocaleString()} answered requests · routing intends ${S}%`+(y.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${a(ae)}">
        <span class="bar">
          <span class="fill${X?" "+X:""}" style="width:${L}%"></span>
          <span class="tick" style="left:${S}%"></span>
        </span>
        <span class="share-n${y.diverged?" warn":""}">${L}%</span>
        ${y.unconfigured?K("not in config","warn"):""}
      </span>
    `}function en(e){const o=l[e.id];return o?o.enabled?o.error?`<p class="muted small">The request counters could not be read: ${a(o.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${o.since?` (${a(mt(o.since))})`:""}. The tick is the share routing intends: on a chain where you run a node, yours
      carries it and the public endpoints are there for when it cannot; on a chain served
      only by public endpoints there is nothing to prefer, so the intent is an even split
      across all of them.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function mt(e){const o=new Date(e);return Number.isNaN(o.getTime())?e:o.toLocaleString()}function tn(e){const o=e.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${a(e.id)}-port" value="${o.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${a(e.id)}-bind" value="${a(o.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${a(o.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${nn(e)}
        ${an(e)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${a(e.id)}">Save settings</button>
        </div>
      </div>
    `}function nn(e){const o=!e.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${a(e.id)}-metrics" ${o?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function an(e){var L;const o=a(e.id),i=e.config.TLS??null,d=(i==null?void 0:i.Enabled)??!1,y=(i==null?void 0:i.CertSource)||"internal",x=((L=e.tls)==null?void 0:L.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${o}-tls" ${d?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${o}-tls-host" value="${a((i==null?void 0:i.Hostname)??x)}"
               placeholder="${a(x||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${x?`<p class="muted small">
               The default is <code>${a(x)}</code>. That whole domain's wildcard resolves to
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
          <option value="internal" ${y==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${y==="files"?"selected":""}>A certificate file on this machine</option>
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
      ${sn(e)}
    `}function sn(e){var L,S;const o=a(e.id),i=((L=e.config.TLS)==null?void 0:L.Enabled)??!1,d=W[e.id]??((S=e.tls)==null?void 0:S.verification)??null,y=B[e.id]??!1,x=R[e.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${o}" ${i&&!y?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${y?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${i?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${x?`<p class="error small">${a(x)}</p>`:""}
      ${d?on(d):""}
    `}function on(e){const o=(e.assertions??[]).map(i=>`
          <li class="small">
            ${rn(i.status)}
            <strong>${a(i.title)}</strong>
            <div class="muted">${a(i.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${e.ok?e.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${a(e.summary)}
      </div>
      <ul class="verify-list">${o}</ul>
      <p class="muted small">
        Checked ${a(new Date(e.at).toLocaleString())} against <code>${a(e.address)}</code>
        ${e.notAfter?`· certificate valid until <code>${a(new Date(e.notAfter).toLocaleString())}</code> (${a(e.expiresIn??"")})`:""}
      </p>
      ${e.expiryWarning?`<div class="banner banner-warn">${a(e.expiryWarning)}</div>`:""}
    `}function rn(e){switch(e){case"pass":return K("pass","ok");case"fail":return K("fail","bad");case"unavailable":return K("unavailable","warn");default:return K("skipped","neutral")}}async function cn(e){B[e]=!0,R[e]=null,Z();try{W[e]=await Dt(e)}catch(o){R[e]=`${be(o)}${Be(o)}`}finally{B[e]=!1,Z()}}function Pe(e){return{...e.config,Networks:(e.config.Networks??[]).map(o=>({ChainID:o.ChainID,Upstreams:o.Upstreams.map(i=>({...i}))}))}}async function Re(e,o,i){N[e]=null;try{await Se(e,o)}catch(d){return N[e]=`${i?i+": ":""}${be(d)}`,Z(),!1}return await V(),!0}async function ln(e,o){const i=o.dataset.gid??"";switch(e){case"refresh":await V();return;case"copy":o.dataset.copy&&await Bn(o,o.dataset.copy);return;case"reprobe":await pe(i,!0);return;case"toggle-settings":_[i]=!_[i],Z();return;case"toggle-manage":j[i]=!j[i],Z();return;case"toggle-chain-detail":{const d=o.dataset.key??"";d&&(A[d]=!A[d]),Z();return}case"save-settings":await dn(i);return;case"verify-tls":await cn(i);return;case"trust-cert":await hn(i);return;case"gw-start":case"gw-stop":case"gw-restart":await fn(i,e.slice(3));return;case"gw-create":case"gw-recreate":await mn(i);return;case"gw-wipe":Rn(i);return;case"add-gateway":Nn();return;case"forget-gateway":await bn(i);return;case"dismiss-orphan":await yn(o.dataset.name??"");return;case"add-chain":vn(i);return;case"add-devnet":{const d=Q(i);if(d){const y=((r==null?void 0:r.targets)??[]).some(x=>x.id===d.placement.targetId&&x.hasDevnet);yt(i,Ue,y)}return}case"remove-chain":await wn(i,Number.parseInt(o.dataset.chain??"",10));return;case"add-endpoint":gt(i,Number.parseInt(o.dataset.chain??"",10));return;case"remove-endpoint":await kn(o.dataset.key??"");return;case"reset-devnet":await En(o.dataset.key??"",o.dataset.target??"");return;default:return}}async function dn(e){const o=Q(e);if(!o)return;const i=Pe(o),d=n.querySelector(`#gw-${CSS.escape(e)}-port`),y=n.querySelector(`#gw-${CSS.escape(e)}-bind`);if(d){const S=Number.parseInt(d.value.trim(),10);Number.isFinite(S)&&(i.Port=S)}y&&(i.BindAddr=y.value.trim());const x=n.querySelector(`#gw-${CSS.escape(e)}-metrics`);x&&(i.MetricsOff=!x.checked),i.TLS=un(e,o);const L=o.status.State==="running";await Re(e,i,"Saving settings")&&(_[e]=!1,L&&(N[e]=null,pn(e,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),Z())}function un(e,o){var x,L,S,X,ae,ye,De;const i=Dn=>n.querySelector(`#gw-${CSS.escape(e)}-${Dn}`),d=i("tls");if(!d)return o.config.TLS??null;const y=Number.parseInt(((x=i("tls-port"))==null?void 0:x.value.trim())??"",10);return{Enabled:d.checked,Hostname:((L=i("tls-host"))==null?void 0:L.value.trim())??"",CertSource:((S=i("tls-source"))==null?void 0:S.value)??"internal",CertFile:((X=i("tls-cert"))==null?void 0:X.value.trim())??"",KeyFile:((ae=i("tls-key"))==null?void 0:ae.value.trim())??"",HTTPSPort:Number.isFinite(y)?y:443,BindAddr:((ye=o.config.TLS)==null?void 0:ye.BindAddr)??"",ImageRef:((De=o.config.TLS)==null?void 0:De.ImageRef)??""}}function pn(e,o){O[e]=[o]}async function hn(e){if(!k[e]){k[e]=!0,T[e]=null,Z();try{T[e]=await oa(e)}catch(o){T[e]={ok:!1,message:`${be(o)}${Be(o)}`}}k[e]=!1,Z()}}async function fn(e,o){if(!f[e]){f[e]=o,N[e]=null,Z();try{await Mt(e,o)}catch(i){N[e]=`${o} failed: ${be(i)}${Be(i)}`}f[e]=null,await V()}}async function mn(e){if(f[e])return;f[e]="create",N[e]=null,O[e]=["starting…"],Z();let o;try{o=await ot(e)}catch(i){N[e]=`${be(i)}${Be(i)}`,O[e]=[],f[e]=null,Z();return}q==null||q(),q=qe(o.targetId,i=>{if(s)return;const d=i.err?`${i.stepId}: ${i.err}`:i.line?`${i.stepId}: ${i.line}`:`${i.stepId}: done`;if(O[e]=[...(O[e]??[]).filter(x=>x!=="starting…"),d],!!i.err||i.stepId===ns&&!!i.done){q==null||q(),q=null,f[e]=null,i.err&&(N[e]="Provisioning failed — see the log below."),V();return}Z()})}async function bn(e){const o=Q(e);if(!(!o||!await Te({title:`Forget ${o.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${o.containerName}" on ${o.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await sa(e)}catch(d){N[e]=be(d),Z();return}await V()}}async function yn(e){if(e){b[e]=null;try{await na(e)}catch(o){b[e]=be(o),Z();return}await V()}}function vn(e){const o=Q(e);if(!o)return;const i=new Set((o.networks??[]).map(S=>S.chainId)),d=(r==null?void 0:r.presets)??[],y=d.filter(S=>!i.has(S.chainId)),x=d.filter(S=>i.has(S.chainId)),L=((r==null?void 0:r.targets)??[]).some(S=>S.id===o.placement.targetId&&S.hasDevnet);re(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${a(o.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${y.map(S=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${S.chainId}">
                <span>${a(S.name)}</span>
                <span class="muted small">chain ${S.chainId}${S.devnet?L?" · uses the devnet on "+a(o.placement.targetId):" · will create a devnet on "+a(o.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${x.length?`<p class="muted small">Already fronted: ${a(x.map(S=>S.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,S=>{if(S==="cancel"){Y();return}if(S==="custom"){gn(e);return}if(S.startsWith("preset:")){const X=Number.parseInt(S.slice(7),10),ae=d.find(ye=>ye.chainId===X);Y(),ae!=null&&ae.devnet?yt(e,X,L):bt(e,X)}})}function gn(e){var o;re(`
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
      `,i=>{if(i==="cancel"){Y();return}if(i!=="add")return;const d=document.getElementById("custom-chain-id"),y=document.getElementById("custom-chain-err"),x=Number.parseInt((d==null?void 0:d.value.trim())??"",10);if(!Number.isFinite(x)||x<=0){y&&(y.className="error small"),y&&(y.textContent="A chain id is a positive whole number.");return}Y(),bt(e,x)}),(o=document.getElementById("custom-chain-id"))==null||o.focus()}async function bt(e,o){const i=Q(e);if(!i)return;const d=Pe(i),y=d.Networks??[];y.some(x=>x.ChainID===o)||(y.push({ChainID:o,Upstreams:[]}),d.Networks=y,await $n(e,d)&&(Z(),gt(e,o)))}async function $n(e,o){var x;const i={...o,Networks:(o.Networks??[]).filter(L=>L.Upstreams.length>0)};if(!await Re(e,i))return!1;const y=Q(e);if(y)for(const L of o.Networks??[])L.Upstreams.length===0&&!(y.networks??[]).some(S=>S.chainId===L.ChainID)&&(y.config.Networks=[...y.config.Networks??[],{ChainID:L.ChainID,Upstreams:[]}],y.networks=[...y.networks??[],{chainId:L.ChainID,name:((x=((r==null?void 0:r.presets)??[]).find(S=>S.chainId===L.ChainID))==null?void 0:x.name)??`Chain ${L.ChainID}`,path:`/${y.config.ProjectID}/evm/${L.ChainID}`,upstreams:[],knownSetSize:0,serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function yt(e,o,i){const d=Q(e);if(!d)return;if(!i){re(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${a(d.placement.targetId)}</code>, so adding chain ${o} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(d.placement.targetId)}" data-modal-action="go">Create a devnet on ${a(d.placement.targetId)}</a>
          </div>
        `,()=>Y());return}const y=Pe(d),x=y.Networks??[],L={ID:"devnet",Kind:"managed-devnet",TargetID:d.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},S=x.find(X=>X.ChainID===o);S?S.Upstreams.push(L):x.push({ChainID:o,Upstreams:[L]}),y.Networks=x,await Re(e,y,"Adding the devnet")}async function wn(e,o){const i=Q(e);if(!i||!Number.isFinite(o))return;const d=ie(i,o);if(!await Te({title:`Remove ${(d==null?void 0:d.name)??`chain ${o}`}`,body:`This gateway will stop serving ${(d==null?void 0:d.path)??`chain ${o}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const x=Pe(i);x.Networks=(x.Networks??[]).filter(L=>L.ChainID!==o),await Re(e,x,"Removing the network")}function vt(e){const o=e.split("|");return o.length!==3?null:{gid:o[0],chainId:Number.parseInt(o[1],10),upstreamId:o[2]}}async function kn(e){const o=vt(e);if(!o)return;const i=Q(o.gid);if(!i)return;const d=Pe(i),y=(d.Networks??[]).find(S=>S.ChainID===o.chainId);if(!y)return;const x=y.Upstreams.findIndex((S,X)=>(S.ID||`${o.chainId}-${X}`)===o.upstreamId);x<0||!await Te({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(y.Upstreams.splice(x,1),await Re(o.gid,d,"Removing the endpoint"))}function gt(e,o){const i=Q(e);if(!i||!Number.isFinite(o))return;const d=((r==null?void 0:r.sources)??[]).filter(S=>S.chainId===o),y=ie(i,o),x=new Set(((y==null?void 0:y.upstreams)??[]).filter(S=>S.kind!=="external").map(S=>`${S.kind}|${S.targetId??""}`)),L=d.filter(S=>!x.has(`${S.kind}|${S.targetId}`));re(`
        <h2>Add an endpoint for ${a((y==null?void 0:y.name)??`chain ${o}`)}</h2>
        ${L.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${L.map(S=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${a(S.kind)}:${a(S.targetId)}">
                       <span>${a(S.label)}</span>
                       <span class="muted small">${a(S.endpoint)}</span>
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
      `,S=>{if(S==="cancel"){Y();return}if(S==="known-set"){xn(e,o);return}if(S==="manual"){In(e,o);return}if(S.startsWith("source:")){const[,X,ae]=S.split(":");Y(),Cn(e,o,X,ae)}})}async function Cn(e,o,i,d){const y=Q(e);if(!y)return;const x=Pe(y),L=x.Networks??[],S={ID:`${i==="managed-devnet"?"devnet":"node"}-${d}`,Kind:i,TargetID:d,Endpoint:"",Local:!0,RecentOnly:!1},X=L.find(ae=>ae.ChainID===o);X?X.Upstreams.push(S):L.push({ChainID:o,Upstreams:[S]}),x.Networks=L,await Re(e,x,"Adding the endpoint")}function Sn(e){const o=[...e].sort((y,x)=>(y.latencyMs??1e9)-(x.latencyMs??1e9)),i=o.slice(0,3),d=o.find(y=>y.url.startsWith("wss://")||y.url.startsWith("ws://"));return d&&!i.some(y=>y.url===d.url)&&(i.length===3&&i.pop(),i.push(d)),new Set(i.map(y=>y.url))}async function xn(e,o){let i;try{i=await rt(e,o)}catch(S){re(`<h2>Endpoints for chain ${o}</h2>
         <p class="error small">Could not read the set: ${a(be(S))}</p>
         <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,()=>Y());return}if(s)return;const d=i.endpoints??[],y=d.filter(S=>!S.alreadyAdded).map(S=>S.url),x=new Set(d.map(S=>S.provider)).size,L=d.map(S=>{const X=[S.websocket?'<span class="t ws">websocket</span>':"",S.archive?'<span class="t ar">archive</span>':"",S.alreadyAdded?'<span class="t dup">already added</span>':""].join("");return`<li><code>${a(S.url)}</code>
                  <span class="muted small">${a(S.provider)}</span> ${X}</li>`}).join("");re(`<h2>Endpoints for chain ${o}</h2>
       ${d.length?`<p class="muted small">${x} providers valve has measured, in the order the gateway
                should prefer them — ${d.length} entries, because a provider that serves both schemes
                appears twice: eRPC reads WebSocket off the scheme, so an <code>https://</code> upstream
                never answers <code>eth_subscribe</code> however well the host speaks it.</p>
              <ul class="plain-list">${L}</ul>`:'<p class="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>'}
       ${i.usingDefaultKey?`<p class="muted small">valve's entries here are resolved with the key that ships with the app, so
                this works with no setup. To use an account of your own instead, put it in Settings under
                <code>VALVE_API_KEY</code>.</p>`:`<p class="muted small">valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${y.length?"":" disabled"}>
           ${y.length?`Add ${y.length}`:"Nothing to add"}</button>
       </div>`,S=>{Y(),S==="add"&&Qe(e,o,y),S==="discover"&&Tn(e,o)})}async function Tn(e,o){re(`
        <h2>Public endpoints for chain ${o}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,L=>{L==="cancel"&&Y()});let i;try{i=await ra(o)}catch(L){const S=Me();if(S){const X=document.createElement("p");X.className="error small",X.textContent=`Could not discover endpoints: ${be(L)}`,S.appendChild(X)}return}if(s)return;const d=(i.endpoints??[]).filter(L=>L.status==="live"||L.status==="unprobed"),y=(i.endpoints??[]).filter(L=>L.status==="rejected"),x=Sn(d);re(`
        <h2>Public endpoints for chain ${o}</h2>
        ${i.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${i.fetchError?`<div class="small">${a(i.fetchError)}</div>`:""}</div>`:""}
        ${d.length?`<p class="muted small">${d.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${d.map(L=>{const S=x.has(L.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${a(L.url)}"${S}>
                       <span><code>${a(L.url)}</code></span>
                       <span class="muted small">${L.status==="live"?`answered in ${L.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${o} right now.</p>`}
        ${y.length?`<details class="rpc-rejected">
                 <summary class="muted small">${y.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${y.map(L=>`<li class="muted small"><code>${a(L.url)}</code> — ${a(L.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${d.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,L=>{if(L==="cancel"){Y();return}if(L==="add"){const S=Me(),X=S?Array.from(S.querySelectorAll('input[type="checkbox"]:checked')).map(ae=>ae.value):[];Y(),Qe(e,o,X);return}})}function In(e,o){var i;re(`
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
      `,d=>{if(d==="cancel"){Y();return}if(d!=="add")return;const y=document.getElementById("manual-endpoint"),x=document.getElementById("manual-recent"),L=document.getElementById("manual-err"),S=(y==null?void 0:y.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(S)){L&&(L.className="error small",L.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Y(),Qe(e,o,[S],(x==null?void 0:x.checked)??!1)}),(i=document.getElementById("manual-endpoint"))==null||i.focus()}async function Qe(e,o,i,d=!1){if(!i.length)return;const y=Q(e);if(!y)return;const x=Pe(y),L=x.Networks??[];let S=L.find(ae=>ae.ChainID===o);S||(S={ChainID:o,Upstreams:[]},L.push(S));let X=1;for(const ae of S.Upstreams){const ye=/^public-\d+-(\d+)$/.exec(ae.ID??"");ye&&(X=Math.max(X,Number(ye[1])+1))}for(const ae of i)S.Upstreams.some(ye=>ye.Endpoint===ae)||S.Upstreams.push({ID:`public-${o}-${X++}`,Kind:"external",Endpoint:ae,Local:!1,RecentOnly:d});x.Networks=L,await Re(e,x,i.length===1?"Adding the endpoint":`Adding ${i.length} endpoints`)}async function En(e,o){const i=vt(e);if(!i||!o||!await Te({title:"Reset this devnet",body:`The chain on ${o} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;f[i.gid]="reset",N[i.gid]=null,Z();let y;try{y=await ea(o)}catch(x){N[i.gid]=`Reset failed: ${be(x)}${Be(x)}`,f[i.gid]=null,Z();return}f[i.gid]=null,Pn(o,y),await V()}function Pn(e,o){const i=[];i.push(o.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),o.report.Recreated&&i.push("A fresh chain was started from genesis.");const d=o.report.Cascaded??[],y=o.report.CascadeSkipped??[];re(`
        <h2>Devnet on ${a(e)} reset</h2>
        <ul class="plain-list">${i.map(x=>`<li>${a(x)}</li>`).join("")}</ul>
        ${d.length?`<p class="ok">Restarted in front of it: ${a(d.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${y.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${a(y.join(", "))}.</p>`:""}
        ${o.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${a(o.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Y())}function Rn(e){const o=Q(e);if(!o)return;re(`
        <h2>Wipe ${a(o.label)}</h2>
        <p class="error">This destroys ${a(o.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${a(e)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${a(e)}</button>
        </div>
      `,y=>{if(y==="cancel"||y==="close"){Y(),V();return}y==="confirm"&&Ln(e)});const i=document.getElementById("wipe-confirm-input"),d=document.getElementById("wipe-confirm-btn");i==null||i.addEventListener("input",()=>{d&&(d.disabled=i.value.trim()!==e)}),i==null||i.focus()}async function Ln(e){const o=document.getElementById("wipe-confirm-btn");o&&(o.disabled=!0,o.textContent="Wiping…");let i;try{i=await Ot(e)}catch(d){const y=Me();if(y){const x=document.createElement("p");x.className="error small",x.textContent=`Wipe failed: ${be(d)}${Be(d)}`,y.appendChild(x)}o&&(o.disabled=!1,o.textContent=`Wipe ${e}`);return}re(`
        <h2>${a(e)} wiped</h2>
        <ul class="plain-list">
          <li>${i.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${i.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${i.error?`<p class="error small">${a(i.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Y(),V()})}function $t(e,o){return!o.some(i=>{var d;return((d=i.placement)==null?void 0:d.targetId)===e})}function Nn(){var x;const e=(r==null?void 0:r.targets)??[],o=(r==null?void 0:r.gateways)??[],i=e.filter(L=>$t(L.id,o)),d=new Set(o.map(L=>L.id));if(e.length===0){re(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Y());return}if(i.length===0){re(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Y());return}const y=d.has("default")?"":"default";re(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${a(y)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${i.map(L=>`<option value="${a(L.id)}">${a(L.id)} (${a(L.mode)})</option>`).join("")}
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
      `,L=>{if(L==="cancel"){Y();return}L==="create"&&An()}),(x=document.getElementById("new-gw-id"))==null||x.focus()}async function An(){const e=document.getElementById("new-gw-id"),o=document.getElementById("new-gw-target"),i=document.getElementById("new-gw-port"),d=document.getElementById("new-gw-err"),y=(e==null?void 0:e.value.trim())??"",x=(o==null?void 0:o.value)??"",L=Number.parseInt((i==null?void 0:i.value.trim())??"",10),S=X=>{d&&(d.className="error small",d.textContent=X)};if(!y){S("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!x){S("Pick the machine it runs on.");return}try{await Bt({id:y,placement:{targetId:x,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(L)?L:4e3,Networks:[]}})}catch(X){S(be(X));return}Y(),await V()}async function Bn(e,o){const i=await Oe(o),d=e.textContent;e.textContent=i?"Copied!":"Copy failed",setTimeout(()=>{s||(e.textContent=d)},1500)}function be(e){return e instanceof Error?e.message:String(e)}function Be(e){return e instanceof Ce&&e.hint?` — ${e.hint}`:""}return()=>{s=!0,q==null||q(),Y()}}const os="local";function rs(n){let s=!1,r=!1,t="",l=null;n.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${he()}
  `;const p=n.querySelector("#targets-body");we(n,(b,k)=>{_(b,k)}),$();async function $(){try{const[b,k,T]=await Promise.all([Ee(),Ie(),Nt()]);if(s)return;t=T.os,N(b,k)}catch(b){if(s)return;p.innerHTML=`<p class="error">Failed to load machines: ${a(String(b))}</p>`}}function f(){l&&N(l.targets,l.catalog)}function N(b,k){l={targets:b,catalog:k};const T=t==="linux",M=[...b].sort((V,se)=>(V.mode==="local"?-1:0)-(se.mode==="local"?-1:0)),q=M.length?`<div class="card-grid">${M.map(V=>is(V,k,V.mode!=="local"||T,t)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',D=b.some(V=>V.mode==="local");p.innerHTML=`
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${q}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${O(T,D)}
        ${r?cs():""}
      </section>
    `;const ne=p.querySelector("#fleet-verdict");ne&&La(ne,Ra(b,k))}function O(b,k){const T=`
      <div class="card">
        <h3>A server over SSH ${K("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${b?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${b?" btn-ghost":""}" data-action="toggle-ssh">
            ${r?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,M=b?`
        <div class="card">
          <h3>This machine ${K("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${t?` (${a(t)})`:""} ${K("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return k?`<div class="card-grid card-grid-wide">${T}</div>`:`<div class="card-grid card-grid-wide">${b?M+T:T+M}</div>`}async function _(b,k){var T;if(b==="add-local"){await A();return}if(b==="delete-target"){const M=k.dataset.id;if(!M||!await Te({title:"Remove machine",body:`Remove "${M}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await j(M);return}if(b==="toggle-ssh"){r=!r,R(),f(),r&&((T=n.querySelector("#ssh-host"))==null||T.focus());return}b==="add-ssh"&&await W()}async function A(){R();try{await st({id:os,mode:"local"}),await $()}catch(b){B(b)}}async function j(b){try{await Mn(b),await $()}catch(k){B(k)}}async function W(){const b=n.querySelector("#ssh-host"),k=n.querySelector("#ssh-user"),T=n.querySelector("#ssh-key"),M=n.querySelector("#ssh-port"),q=n.querySelector("#ssh-id");if(!b||!k||!T||!M||!q)return;const D=b.value.trim(),ne=k.value.trim(),V=T.value.trim(),se=M.value.trim(),pe=q.value.trim();if(R(),!D||!ne||!V){B(new Error("host, user, and key path are required"));return}const Q=pe||ls(D),ie={Host:D,User:ne,KeyPath:V};if(se){const le=Number.parseInt(se,10);if(!Number.isFinite(le)||le<=0){B(new Error("port must be a positive number"));return}ie.Port=le}const oe=n.querySelector("#ssh-submit");oe&&(oe.disabled=!0,oe.textContent="Connecting…");try{await st({id:Q,mode:"ssh",ssh:ie}),r=!1,await $()}catch(le){B(le),oe&&(oe.disabled=!1,oe.textContent="Add server")}}function B(b){let k=n.querySelector("#targets-error");k||(p.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),k=n.querySelector("#targets-error")),k.textContent=String(b instanceof Error?b.message:b)}function R(){var b;(b=n.querySelector("#targets-error"))==null||b.remove()}return()=>{s=!0}}function is(n,s,r,t){const l=n.wire,p=n.mode==="local"?"this machine":"SSH",$=n.mode==="ssh"&&n.ssh?`${a(n.ssh.User)}@${a(n.ssh.Host)}`:p;let f;if(!l&&!r)f=`${K("can't run a node","warn")} ${K(t||"not Linux","neutral")}`;else if(!l)f=K("not set up","neutral");else{const N=s.networks.find(_=>_.ChainID===l.ChainID),O=N?N.Name:`chain ${l.ChainID}`;f=`${K(O,"ok")} ${K(l.ExecID,"neutral")} ${K(l.BeaconID,"neutral")}${l.Archive?" "+K("archive","warn"):""}`}return`
    <div class="card">
      <h2>${a(n.id)}</h2>
      <p class="muted">${$}</p>
      <p>${f}</p>
      <div class="card-actions">
        <a class="btn" href="#/machine/${encodeURIComponent(n.id)}">Open</a>
        <button class="btn btn-danger" data-action="delete-target" data-id="${a(n.id)}">Remove</button>
      </div>
    </div>
  `}function cs(){return`
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
  `}function ls(n){return n.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ds=document.querySelector("#app"),{contentEl:us,setActiveNav:ps}=la(ds);let ge=null;function hs(){const s=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(s.length===0)return{screen:"home"};const[r,t]=s;return r==="machine"||r==="setup"||r==="dash"||r==="logs"||r==="security"||r==="diag"||r==="services"||r==="analytics"?{screen:r,id:t?decodeURIComponent(t):void 0}:{screen:r??"targets"}}function ke(n){const s=document.createElement("div");return us.replaceChildren(s),n(s)}function qt(){if(ge){try{ge()}catch{}ge=null}const{screen:n,id:s}=hs();switch(ps(n),n){case"machine":if(!s){location.hash="#/targets";return}ge=ke(r=>Sa(r,s));break;case"setup":case"dash":case"logs":case"services":if(!s){location.hash="#/targets";return}location.hash=`#/machine/${encodeURIComponent(s)}`;return;case"security":if(!s){location.hash="#/targets";return}ge=ke(r=>Ya(r,s));break;case"diag":if(!s){location.hash="#/targets";return}ge=ke(r=>fa(r,s));break;case"analytics":if(!s){location.hash="#/rpc";return}ge=ke(r=>ha(r,s));break;case"rpc":ge=ke(r=>ss(r));break;case"settings":ge=ke(r=>Qa(r));break;case"targets":ge=ke(r=>rs(r));break;case"panel":ge=ke(r=>Rt(r));break;case"home":default:ge=ke(r=>Rt(r));break}}window.addEventListener("hashchange",qt);qt();
