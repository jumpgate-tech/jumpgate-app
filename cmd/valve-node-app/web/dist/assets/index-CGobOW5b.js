var Dt=Object.defineProperty;var Ut=(s,i,o)=>i in s?Dt(s,i,{enumerable:!0,configurable:!0,writable:!0,value:o}):s[i]=o;var De=(s,i,o)=>Ut(s,typeof i!="symbol"?i+"":i,o);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))e(h);new MutationObserver(h=>{for(const w of h)if(w.type==="childList")for(const R of w.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&e(R)}).observe(document,{childList:!0,subtree:!0});function o(h){const w={};return h.integrity&&(w.integrity=h.integrity),h.referrerPolicy&&(w.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?w.credentials="include":h.crossOrigin==="anonymous"?w.credentials="omit":w.credentials="same-origin",w}function e(h){if(h.ep)return;h.ep=!0;const w=o(h);fetch(h.href,w)}})();function Mt(){return K("/api/host")}function xe(){return K("/api/catalog")}function Pe(){return K("/api/targets")}function tt(s){return K("/api/targets",{method:"POST",headers:fe,body:JSON.stringify(s)})}function Ot(s){return K(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Ft(s,i){return K(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function qt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:fe,body:JSON.stringify(i)})}function Je(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function jt(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Wt(s,i=200){return K(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function _t(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function nt(s,i){const o=i===void 0?{}:{lines:i};return K(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:fe,body:JSON.stringify(o)})}function Kt(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/${o}`,{method:"POST"})}function zt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:i})})}function Gt(s){return K(`/api/targets/${encodeURIComponent(s)}/du`)}function Jt(s){return K(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function Vt(s){return K(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Yt(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function Zt(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function Xt(s){return K(`/api/targets/${encodeURIComponent(s)}/containers`)}function Qt(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${o}`,{method:"POST"})}async function en(s,i){const o=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:i})}),e=await o.text();let h=null;try{h=e?JSON.parse(e):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const w=h&&typeof h=="object"&&typeof h.error=="string"?h.error:o.statusText||`HTTP ${o.status}`;throw new ke(o.status,w)}function tn(s,i){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function nn(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:fe}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ke(i.status,h)}function an(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:fe,body:JSON.stringify(o)})}function ct(){return K("/api/gateways")}async function sn(s){await K(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function rn(s){return K("/api/gateways",{method:"POST",headers:fe,body:JSON.stringify(s)})}function on(s){return K(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function cn(s){return K(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function ln(s){return K(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function dn(s,i=!1){const o=i?"?refresh=1":"";return K(`/api/gateways/${encodeURIComponent(s)}/capabilities${o}`)}function un(s){return K(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function pn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:fe,body:JSON.stringify(i)})}function hn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function fn(s){return K(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function mn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:fe,body:JSON.stringify({Confirm:s})}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ke(i.status,h)}function bn(s){return K(`/api/chainlist/${s}`)}function gn(){return K("/api/settings")}function yn(s){return K("/api/settings",{method:"PUT",headers:fe,body:JSON.stringify(s)})}class ke extends Error{constructor(o,e,h,w){super(e);De(this,"status");De(this,"hint");De(this,"code");this.name="ApiError",this.status=o,this.hint=h,this.code=w}}const fe={"Content-Type":"application/json"};async function K(s,i){const o=await fetch(s,i);if(!o.ok){let h=o.statusText||`HTTP ${o.status}`,w,R;try{const m=await o.json();m&&typeof m.error=="string"&&m.error&&(h=m.error),m&&typeof m.hint=="string"&&m.hint&&(w=m.hint),m&&typeof m.code=="string"&&m.code&&(R=m.code)}catch{}throw new ke(o.status,h,w,R)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const at="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ie(s,i){const o=s&&i&&i!==at?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(at)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function vn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),o=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{for(const w of o)w.classList.toggle("active",w.dataset.nav===h)}}}function te(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function $n(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function wn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),o=Math.floor(i/60),e=i%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const h=Math.floor(o/24),w=o%24;return`~${h}d ${w}h`}function U(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function we(s){return`<span class="dot dot-${s}"></span>`}const st=["B","KB","MB","GB","TB","PB"];function Ce(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,o=0;for(;i>=1024&&o<st.length-1;)i/=1024,o++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${st[o]}`}async function Ne(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function ve(s,i){s.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!s.contains(e))return;const h=e.dataset.action;h&&i(h,e,o)})}function Ge(s,i,o){const e=i.find(w=>w.value===o),h=i.map(w=>`
      <li class="dropdown-option${w.value===o?" selected":""}" role="option"
          aria-selected="${w.value===o}" data-value="${n(w.value)}">
        ${n(w.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Ie(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var o;i.classList.remove("open"),(o=i.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function Ve(s,i){s.addEventListener("click",h=>{const w=h.target,R=w.closest(".dropdown-trigger");if(R&&s.contains(R)){const N=R.closest(".dropdown"),j=!!N&&!N.classList.contains("open");Ie(s),N&&j&&(N.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const m=w.closest(".dropdown-option");if(m&&s.contains(m)){const N=m.closest(".dropdown");Ie(s),i((N==null?void 0:N.dataset.dropdown)??"",m.dataset.value??"");return}Ie(s)});const o=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const w=h.target;(!w.closest(".dropdown")||!s.contains(w))&&Ie(s)},e=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}h.key==="Escape"&&Ie(s)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const je="app-modal";let Fe=null;function se(s,i){V();const o=document.createElement("div");o.className="modal-overlay",o.id=je,o.innerHTML=`<div class="modal">${s}</div>`,o.addEventListener("click",h=>{const w=h.target.closest("[data-modal-action]");w!=null&&w.dataset.modalAction?i(w.dataset.modalAction):h.target===o&&i("cancel")});const e=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),Fe=e,document.body.appendChild(o)}function V(){var s;(s=document.getElementById(je))==null||s.remove(),Fe&&(document.removeEventListener("keydown",Fe),Fe=null)}function qe(){return document.querySelector(`#${je} .modal`)}function Le(s){return new Promise(i=>{var h;let o=!1;const e=w=>{o||(o=!0,V(),i(w))};se(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,w=>e(w==="confirm")),(h=document.querySelector(`#${je} [data-modal-action="confirm"]`))==null||h.focus()})}const _e=5e3,kn=60;function Tn(s,i){let o=!1,e=null,h=null,w=null,R=null;const m=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ie()}</div>`;const N=s.querySelector("#an-body");ve(s,(b,l)=>{var k;b==="toggle-endpoint"&&((k=l.closest(".an-endpoint"))==null||k.classList.toggle("expanded"))}),j();async function j(){try{e=((await ct()).gateways??[]).find(l=>l.id===i)??null}catch(b){if(o)return;w=String(b instanceof Error?b.message:b),B();return}if(!o){if(!e){B();return}await M(),R=window.setInterval(()=>void M(),_e)}}async function M(){try{const b=await ln(i);if(o)return;_(b),h=b,w=null}catch(b){if(o)return;w=String(b instanceof Error?b.message:b)}B()}function _(b){if(!b.enabled||b.error)return;const l=m[m.length-1];l&&l.since!==b.since&&(m.length=0);const k=new Map;for(const x of b.networks??[])k.set(x.chainId,x.received);m.push({t:Date.now(),since:b.since,received:k}),m.length>kn&&m.shift()}function B(){o||(N.innerHTML=F())}function F(){return w&&!h?`<h1>Analytics</h1><p class="error">${n(w)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${E(e)}
      ${h?u(h):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${n(i)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function E(b){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${n(b.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${T()}</div>
      </div>
    `}function T(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const b=h.since?new Date(h.since):null;return b&&!Number.isNaN(b.getTime())?`totals since the gateway started, ${n(b.toLocaleString())}<br />re-read every ${_e/1e3}s`:`re-read every ${_e/1e3}s`}function u(b){return b.enabled?b.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(b.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:$(b)+oe(b):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function $(b){const l=b.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${l.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':l.map(k=>A(k)).join("")}
      </section>
    `}function A(b){const l=b.methods??[],k=b.endpoints??[],x=b.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${b.chainId}</span>
          <span class="band-name">${n(b.name)}</span>
          ${z(b)}
        </div>
        <div class="an-stats">
          ${H("Received",te(b.received),"what clients asked this chain for")}
          ${H("Answered",te(b.answered),"returned by one of your endpoints")}
          ${H("From cache",te(b.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${H("Failed",te(b.failed),"asked for and never answered",b.failed>0?"bad":"")}
        </div>
        ${X(b.chainId)}
        ${x?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':G("Method",l.map(L=>({label:L.method,l:L})))+G("Endpoint",k.map(L=>({label:L.upstream,l:L})))+ne(b)}
      </div>
    `}function H(b,l,k,x=""){return`
      <div class="an-stat${x?" an-stat-"+x:""}" title="${n(k)}">
        <span class="an-stat-n">${n(l)}</span>
        <span class="an-stat-l">${n(b)}</span>
      </div>
    `}function z(b){const l=Q(b.chainId);if(l===null)return'<span class="an-rate muted small">measuring rate…</span>';const k=Math.round((m[m.length-1].t-m[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${k}s apart.">
      ${n(l.toFixed(l<10?2:0))} req/s <span class="muted">over the last ${k}s</span>
    </span>`}function Q(b){if(m.length<2)return null;const l=m[0],k=m[m.length-1],x=(k.t-l.t)/1e3;if(x<=0)return null;const L=(k.received.get(b)??0)-(l.received.get(b)??0);return L<0?null:L/x}function X(b){if(m.length<3)return"";const l=[];for(let v=1;v<m.length;v++){const S=m[v-1],O=m[v],c=(O.t-S.t)/1e3,f=(O.received.get(b)??0)-(S.received.get(b)??0);l.push(c>0&&f>=0?f/c:0)}const k=Math.max(...l);if(k<=0)return"";const x=240,L=28,q=l.length>1?x/(l.length-1):x,p=l.map((v,S)=>`${(S*q).toFixed(1)},${(L-v/k*L).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${k.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${x} ${L}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${p}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(k.toFixed(2))} req/s</span>
      </div>
    `}function ne(b){const l=[];return b.cached.count>0&&l.push(`${n(te(b.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${b.cached.mean===null?"":`, in ${n(Re(b.cached.mean))} on average`}.`),b.failedLatency.count>0&&b.failedLatency.mean!==null&&l.push(`The ${n(te(b.failedLatency.count))} that failed took
         ${n(Re(b.failedLatency.mean))} on average to fail.`),l.length===0?"":`<p class="muted small">${l.join(" ")}</p>`}function G(b,l){return l.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${n(b)}</th>
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
    `}function de(b,l){return`
      <tr>
        <td><code>${n(b)}</code></td>
        <td class="an-num">${te(l.count)}</td>
        <td class="an-num">${l.mean===null?'<span class="muted">—</span>':n(Re(l.mean))}</td>
        <td>${pe(l)}</td>
      </tr>
    `}function pe(b){const l=b.buckets??[];if(l.length===0||b.count===0)return'<span class="muted small">—</span>';let k=0;const x=[];for(const q of l){const p=q.count-k;k=q.count,x.push({label:re(q.le),n:Math.max(0,p)})}return x.reduce((q,p)=>q+p.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(x.filter(q=>q.n>0).map(q=>`${q.n} ${q.label}`).join(" · "))}">
        ${x.map((q,p)=>q.n===0?"":`<span class="an-band an-band-${Math.min(p,4)}" style="flex:${q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ce(x))}</span>
    `}function ce(b){for(let l=b.length-1;l>=0;l--)if(b[l].n>0)return`slowest ${b[l].label}`;return""}function re(b){if(b==="+Inf")return"30s or more";const l=Number(b);return Number.isFinite(l)?`under ${Re(l)}`:`under ${b}`}function oe(b){const l=b.endpoints??[];return`
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
                     <tbody>${l.map(k=>me(k)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function me(b){const l=b.errors??[],k=l.reduce((L,q)=>L+q.count,0),x=l.length>0;return`
      <tr class="an-endpoint${x?" expandable":""}" ${x?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(b.upstream)}</code>
          ${b.chainId?`<span class="muted small">chain ${b.chainId}</span>`:""}
          ${b.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${te(b.requests)}</td>
        <td class="an-num${k>0?" bad":""}">${k>0?te(k):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${b.headLag>0?te(b.headLag):'<span class="muted">0</span>'}</td>
        <td>${be(b)}</td>
      </tr>
      ${x?ge(b,l):""}
    `}function be(b){const l=[];return b.scored?(l.push(b.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(b.position))}</span>`),l.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(b.score.toFixed(3))}</span>`),b.primarySwitches>1&&l.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${te(b.primarySwitches)} switches</span>`),b.excludedSeconds>0&&l.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Re(b.excludedSeconds))}</span>`),`<span class="an-selection">${l.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ge(b,l){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${l.map(k=>`
                    <tr>
                      <td class="an-num">${te(k.count)}</td>
                      <td><code>${n(k.class)}</code></td>
                      <td>${k.severity?`<span class="badge badge-${k.severity==="critical"?"bad":"warn"}">${n(k.severity)}</span>`:""}</td>
                      <td class="muted small">${n(k.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${n(b.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{o=!0,R!==null&&window.clearInterval(R)}}function Re(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}const Cn=85,Ke={exec:"Execution",beacon:"Beacon"};function Sn(s,i){let o=!1,e=null,h=null,w=null,R=null,m=null,N=null,j=null,M=null;const _={exec:null,beacon:null};let B=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ie()}</div>`;const F=s.querySelector("#dash-body"),E=s.querySelector("#dash-footer");F.addEventListener("click",l=>{const k=l.target.closest("[data-action]");if(!k||!F.contains(k))return;const x=k.dataset.action;if(x==="svc-action"){const L=k.dataset.svc,q=k.dataset.kind;L&&q&&me(L,q)}else if(x==="open-clear"){const L=k.dataset.svc;L&&ge(L)}else if(x==="copy"){const L=k.dataset.copy;L&&be(k,L)}else x==="retry-du"?u():x==="retry-endpoints"&&$()}),T();async function T(){let l,k;try{const[L,q]=await Promise.all([Pe(),xe()]);l=L.find(p=>p.id===i),k=q}catch(L){if(o)return;F.innerHTML=`<p class="error">Failed to load target: ${n(String(L))}</p>`;return}if(o)return;if(!l){F.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){F.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const x=k==null?void 0:k.networks.find(L=>L.ChainID===l.wire.ChainID);x&&(E.innerHTML=ie(x.Name,x.LearnURL)),F.innerHTML='<p class="muted">Connecting…</p>',e=jt(i,L=>{o||(A(L),h=L,w=L,H())}),u(),$()}async function u(){N=null;try{m=await Gt(i)}catch(l){m=null,N=String(l instanceof Error?l.message:l)}o||H()}async function $(){M=null;try{j=await Jt(i)}catch(l){j=null,M=String(l instanceof Error?l.message:l)}o||H()}function A(l){if(!h)return;const k=(new Date(l.at).getTime()-new Date(h.at).getTime())/1e3,x=l.execHead-h.execHead;if(k>0&&x>=0){const L=x/k;R=R===null?L:R*.7+L*.3}}function H(){if(!w)return;const l=w;F.innerHTML=`
      <p class="dash-status">${z(l)}</p>
      <div class="card-grid">
        ${re(l)}
        ${X(l)}
        ${ne(l)}
        ${G(l)}
        ${de(l)}
        ${pe()}
      </div>
      <p class="muted small">Last updated ${n(new Date(l.at).toLocaleTimeString())}</p>
    `}function z(l){return!l.execActive&&!l.beaconActive?U("Node not running","bad"):l.execSyncing||l.beaconDistance>0?U("Syncing","warn"):U("Running · synced","ok")}function Q(l){const x=l.refHead>0?l.refHead-l.execHead:null,L=x!==null&&x>0&&R&&R>0?wn(x/R):x!==null&&x<=0?"caught up":"—";return{lag:x,eta:L}}function X(l){const{lag:k,eta:x}=Q(l);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${l.execActive?l.execSyncing?U("syncing","warn"):l.execHead===0?U("no data","neutral"):U("synced","ok"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${te(l.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${k!==null?te(l.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${k!==null?te(Math.max(k,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${x}</dd></div>
        </dl>
      </div>
    `}function ne(l){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${l.beaconActive?l.beaconSlot===0?U("no data","neutral"):l.beaconDistance===0?U("synced","ok"):U("syncing","warn"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${te(l.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${te(l.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function G(l){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${te(l.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${te(l.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(l){const k=l.diskUsedPct>=Cn,x=`
      <div class="meter"><div class="meter-fill ${k?"meter-warn":""}" style="width:${Math.min(l.diskUsedPct,100)}%"></div></div>
      <p>${$n(l.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="error small">${n(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${k?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="muted">Loading…</p>
        </div>
      `;const L=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,q=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:p,eta:v}=Q(l),S=p!==null&&p>0&&R!==null&&R>0;return`
      <div class="card ${k?"card-warn":""}">
        <h3>Storage</h3>
        ${x}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Ce(m.ExecBytes)} of ~${Ce(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${L}%"></div></div>
        ${S?`<p class="muted small">Estimated time remaining: ${n(v)}</p>`:""}
        <p class="muted small">Beacon — ${Ce(m.BeaconBytes)} of ~${Ce(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Ce(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function pe(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const l=j,k=l.ExecReachable&&!l.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",x=l.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${n(l.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${we(l.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${n(l.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${we(l.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${n(l.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(l.BeaconHTTP)}">Copy</button>
        </div>
        ${k}
        ${x}
      </div>
    `}function ce(l,k){const x=Ke[l],L=_[l],q=(p,v,S)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${l}" data-kind="${p}" ${L!==null||S?"disabled":""}>${L===p?oe():n(v)}</button>`;return`
      <div class="service-row">
        <span>${n(x)} ${k?U("active","ok"):U("down","bad")}</span>
        <div class="service-actions">
          ${q("start","Start",k)}
          ${q("stop","Stop",!k)}
          ${q("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${l}" ${L!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function re(l){return`
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
    `}function oe(){return'<span class="spinner" aria-label="working"></span>'}async function me(l,k){if(_[l]===null){_[l]=k,B=null,H();try{await Kt(i,l,k)}catch(x){B=`${Ke[l]} ${k} failed: ${x instanceof Error?x.message:String(x)}`}_[l]=null,o||H()}}async function be(l,k){const x=await Ne(k),L=l.textContent;l.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=L)},1500)}function ge(l){const k=Ke[l],x=m?Ce(l==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";se(`
        <h2>Clear ${n(k)} data</h2>
        <p class="error">
          This stops the ${n(k.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(x)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,p=>{if(p==="cancel"){V();return}p==="confirm"&&b(l)});const L=document.getElementById("clear-confirm-input"),q=document.getElementById("clear-confirm-btn");L==null||L.addEventListener("input",()=>{q&&(q.disabled=L.value.trim()!==l)}),L==null||L.focus()}async function b(l){const k=document.getElementById("clear-confirm-btn");k&&(k.disabled=!0,k.textContent="Clearing…");try{await zt(i,l),V(),u()}catch(x){const L=qe();if(L){const q=document.createElement("p");q.className="error small",q.textContent=`Clear failed: ${x instanceof Error?x.message:String(x)}`,L.appendChild(q)}k&&(k.disabled=!1,k.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),V()}}const rt=500,ot="valve-node-app.explain-consent";function xn(s,i){let o=!1,e=null;const h=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ie()}</div>
  `;const w=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");ve(s,T=>{T==="explain"&&M()}),m();async function m(){let T,u;try{const[A,H]=await Promise.all([Pe(),xe()]);T=A.find(z=>z.id===i),u=H}catch(A){if(o)return;w.innerHTML=`<p class="error">Failed to load target: ${n(String(A))}</p>`;return}if(o)return;if(!T){w.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!T.wire){w.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const $=u==null?void 0:u.networks.find(A=>A.ChainID===T.wire.ChainID);$&&(R.innerHTML=ie($.Name,$.LearnURL));try{const A=await Wt(i,200);if(o)return;h.push(...A)}catch(A){if(o)return;w.innerHTML=`<p class="error">Failed to load logs: ${n(String(A))}</p>`;return}N(),e=_t(i,A=>{o||(h.push(A),h.length>rt&&h.splice(0,h.length-rt),N())})}function N(){const T=h.filter($=>$.severity==="error"||$.severity==="critical");w.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${h.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${U(String(T.length),T.length?"bad":"neutral")}</h2>
          <div class="log-lines">${T.length?T.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=w.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function j(T){const u=T.severity||"info",$=T.learnUrl?` <a href="${n(T.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(u)}">
        <span class="log-time">${n(new Date(T.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(T.unit)}</span>
        <span class="log-sev">${n(u)}</span>
        <span class="log-text">${n(T.line)}</span>
        ${T.explain?`<div class="log-explain">${n(T.explain)}${$}</div>`:""}
      </div>
    `}async function M(){const T=h.filter($=>$.severity==="error"||$.severity==="critical").map($=>$.line).slice(-40);if(!(localStorage.getItem(ot)==="1")){_(T);return}await B(T)}function _(T){const u=T.length?`<pre class="explain-excerpt">${T.map($=>n($)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';F(`
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
    `,$=>{$==="proceed"?(localStorage.setItem(ot,"1"),E(),B(T)):E()})}async function B(T){F('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=T.length?await nt(i,T):await nt(i);if(o)return;F(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map($=>n($)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&E()})}catch(u){if(o)return;if(u instanceof ke&&u.status===409){F(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,$=>{$==="close"&&E()});return}F(`
        <h2>Explain failed</h2>
        <p class="error">${n(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,$=>{$==="close"&&E()})}}function F(T,u){E();const $=document.createElement("div");$.className="modal-overlay",$.id="explain-modal",$.innerHTML=`<div class="modal">${T}</div>`,$.addEventListener("click",A=>{const H=A.target.closest("[data-modal-action]");H!=null&&H.dataset.modalAction&&u(H.dataset.modalAction),A.target===$&&u("cancel")}),document.body.appendChild($)}function E(){var T;(T=document.getElementById("explain-modal"))==null||T.remove()}return()=>{o=!0,e==null||e(),E()}}function Pn(s,i){let o=!1,e=null,h=null,w=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ie()}</div>`;const m=s.querySelector("#diag-body"),N=s.querySelector("#diag-footer");ve(s,(u,$)=>{var A;if(u==="run")M();else if(u==="toggle")(A=$.closest(".check-item"))==null||A.classList.toggle("expanded");else if(u==="copy"){const H=$.dataset.copy;H&&T($,H)}}),j();async function j(){let u,$;try{const[H,z]=await Promise.all([Pe(),xe()]);u=H.find(Q=>Q.id===i),$=z}catch(H){if(o)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(o)return;if(!u){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const A=$==null?void 0:$.networks.find(H=>H.ChainID===u.wire.ChainID);A&&(N.innerHTML=ie(A.Name,A.LearnURL));try{e=await Zt(i),R=!0}catch(H){h=String(H instanceof Error?H.message:H)}o||_()}async function M(){w=!0,h=null,_();try{e=await Yt(i),R=!0}catch(u){h=String(u instanceof Error?u.message:u)}w=!1,o||_()}function _(){m.innerHTML=`
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
      ${B()}
    `}function B(){if(!R&&!h)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const u=new Date(e.at).toLocaleString(),$=e.failedId?`<p><strong>Failed at: ${n(F(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(u)} — trigger: ${n(e.trigger)}</p>
      ${$}
      <ul class="check-list">${e.items.map(E).join("")}</ul>
    `}function F(u){var $;return(($=e==null?void 0:e.items.find(A=>A.ID===u))==null?void 0:$.Title)??u}function E(u){const $=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral",A=u.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${A?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${U(A?"failed here":u.Status,$)}
          <strong>${n(u.Title)}</strong>
          <span class="muted small check-detail-inline">${n(u.Detail)}</span>
        </button>
        <div class="check-body">
          <details${A?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${n(u.Why)}</p>
          </details>
          ${u.Fix?`
                <details${A?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(u.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(u.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(u,$){const A=await Ne($),H=u.textContent;u.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{o||(u.textContent=H)},1500)}return()=>{o=!0}}function En(s,i){let o=!1,e=[],h=null,w=!1,R=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ie()}</div>`;const m=s.querySelector("#sec-body"),N=s.querySelector("#sec-footer");ve(s,(E,T)=>{var u;if(E==="rerun")M();else if(E==="toggle")(u=T.closest(".check-item"))==null||u.classList.toggle("expanded");else if(E==="copy"){const $=T.dataset.copy;$&&F(T,$)}}),j();async function j(){let E,T;try{const[$,A]=await Promise.all([Pe(),xe()]);E=$.find(H=>H.id===i),T=A}catch($){if(o)return;m.innerHTML=`<p class="error">Failed to load target: ${n(String($))}</p>`;return}if(o)return;if(!E){m.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!E.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=T==null?void 0:T.networks.find($=>$.ChainID===E.wire.ChainID);u&&(N.innerHTML=ie(u.Name,u.LearnURL)),await M()}async function M(){w=!0,h=null,_();try{e=await Vt(i),R=!0}catch(E){h=String(E instanceof Error?E.message:E)}w=!1,o||_()}function _(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${w?"disabled":""}>${w?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${!R&&w?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(B).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function B(E){const T=E.Status==="pass"?"ok":E.Status==="fail"?"bad":E.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${U(E.Status,T)}
          <strong>${n(E.Title)}</strong>
          <span class="muted small check-detail-inline">${n(E.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${n(E.Why)}</p>
          </details>
          ${E.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${n(E.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${n(E.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function F(E,T){const u=await Ne(T),$=E.textContent;E.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{o||(E.textContent=$)},1500)}return()=>{o=!0}}const In=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Rn(s){let i=!1,o=!1,e=!1,h=null,w=!1,R=null,m=null;s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ie()}`;const N=s.querySelector("#settings-body");ve(s,B=>{if(B==="save"&&_(),B==="clear-key"){if(!R)return;o=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),M(R)}}),Ve(s,(B,F)=>{B!=="ai-provider"||!R||(m=F,w=!1,M(R))}),j();async function j(){try{const B=await gn();if(i)return;R=B,M(B)}catch(B){if(i)return;N.innerHTML=`<p class="error">Failed to load settings: ${n(String(B))}</p>`}}function M(B){var T;const F=m??B.aiProvider;N.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ge("ai-provider",In.map(u=>({value:u.value,label:u.label})),F)}
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
        ${h?`<p class="error">${n(h)}</p>`:""}
        ${w?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const E=s.querySelector("#ai-key");E==null||E.addEventListener("input",()=>{o=!0,w=!1}),(T=s.querySelector("#ref-rpc-base"))==null||T.addEventListener("input",()=>{w=!1})}async function _(){const B=s.querySelector("#ai-key"),F=s.querySelector("#ref-rpc-base");if(!B||!F||!R)return;const E={aiProvider:m??R.aiProvider,refRpcBase:F.value.trim()};o&&(E.aiKey=B.value),e=!0,h=null,w=!1,M(R);try{const T=await yn(E);if(i)return;R=T,o=!1,e=!1,w=!0,M(T)}catch(T){if(i)return;e=!1,h=String(T instanceof Error?T.message:T),M(R)}}return()=>{i=!0}}const Ln=["http","ws","archive","trace"],Nn={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},An="run",Bn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Hn(s){let i=!1,o=null,e=null;const h={},w={},R={},m={},N={},j={},M={},_={},B={},F={},E={};let T=null;s.innerHTML=`
    <div class="page-head">
      <h1>RPC</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      eRPC sits above everything else here. One gateway fronts as many chains as you
      list, and each chain can be served by a devnet on this machine, a node on any
      machine you manage, or a public endpoint — a gateway names the machine it runs
      on, it does not belong to it.
    </p>
    <div id="rpc-body"><p class="muted">Loading…</p></div>
    ${ie()}
  `;const u=s.querySelector("#rpc-body");ve(s,(t,a)=>{dt(t,a)}),Ve(s,()=>{}),$();async function $(){try{const t=await ct();if(i)return;o=t,e=null}catch(t){if(i)return;o=null,e=he(t)}G();for(const t of(o==null?void 0:o.gateways)??[])A(t.id),H(t.id,!1)}async function A(t){try{const a=await cn(t);if(i)return;h[t]=a}catch{if(i)return;h[t]=null}G()}async function H(t,a){R[t]=a,a&&G();try{const r=await dn(t,a);if(i)return;w[t]=r}catch{if(i)return;w[t]=null}R[t]=!1,G()}function z(t){return((o==null?void 0:o.gateways)??[]).find(a=>a.id===t)}function Q(t,a){return(t.networks??[]).find(r=>r.chainId===a)}function X(t,a,r){var g;const d=(((g=h[t])==null?void 0:g.networks)??[]).find(y=>y.chainId===a);return((d==null?void 0:d.upstreams)??[]).find(y=>y.upstream===r)}function ne(t,a,r){var d;return(((d=w[t])==null?void 0:d.endpoints)??[]).find(g=>g.chainId===a&&g.upstream===r)}function G(){if(i)return;if(e){u.innerHTML=`<p class="error">Could not read the gateways: ${n(e)}</p>`;return}if(!o){u.innerHTML='<p class="muted">Loading…</p>';return}const t=o.gateways??[];u.innerHTML=`
      ${(o.orphans??[]).map(de).join("")}
      ${t.map(ce).join("")}
      ${t.length===0?pe():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${t.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function de(t){return`
      <div class="banner banner-warn">
        <strong>${n(t.containerName)}</strong> is still running on
        ${n(t.targetId)}. Its chains were folded into
        <code>${n(t.mergedInto)}</code>, but valve-node-app does not stop
        containers it did not start. Remove it yourself:
        <code>docker rm -f ${n(t.containerName)}</code>
        ${E[t.containerName]?`<div class="error small">${n(E[t.containerName])}</div>`:""}
        <div class="card-actions">
          <button class="btn btn-ghost" data-action="dismiss-orphan" data-name="${n(t.containerName)}">Dismiss</button>
        </div>
      </div>
    `}function pe(){return((o==null?void 0:o.targets)??[]).length===0?`
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
    `}function ce(t){return`
      <section class="rpc-gateway">
        ${re(t)}
        ${t.error?be(t):""}
        ${t.blocked?`<div class="banner banner-warn">${n(t.blocked)}</div>`:""}
        ${(t.warnings??[]).map(a=>`<div class="banner banner-warn">${n(a)}</div>`).join("")}
        ${He(t)}
        ${N[t.id]?`<p class="error small">${n(N[t.id])}</p>`:""}
        ${b(t)}
        ${M[t.id]?D(t):""}
        ${l(t)}
      </section>
    `}function re(t){var r;const a=t.status.State==="running";return`
      <div class="rpc-bar${a?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${me(t)}
            <strong>${n(t.label)}</strong>
            ${oe(t)}
            <span class="muted small">on ${n(t.placement.targetId)} · ${n(t.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(t.actions??[]).map(d=>ge(t,d)).join("")}
            <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(t.id)}"
               title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${n(t.id)}">
              ${M[t.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${n(t.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${a?`<code class="endpoint-url">${n(t.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${n(t.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${n(((r=(t.networks??[])[0])==null?void 0:r.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${n(t.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function oe(t){switch(t.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function me(t){return t.status.State==="running"?we("ok"):t.status.State==="unknown"?we("bad"):we("neutral")}function be(t){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${n(t.error??"")}</div>
        ${t.hint?`<div class="small">${n(t.hint)}</div>`:""}
      </div>
    `}function ge(t,a){const r=Bn[a];if(!r)return"";const d=m[t.id];return`
      <button class="${r.className}" data-action="gw-${a}" data-gid="${n(t.id)}"
              title="${n(r.title)}" ${d?"disabled":""}>
        ${d===a?'<span class="spinner" aria-label="working"></span>':n(r.label)}
      </button>
    `}function b(t){const a=j[t.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${n(t.placement.targetId)}</p>
        <pre class="step-log">${n(a.join(`
`))}</pre>
      </div>
    `}function l(t){const a=t.networks??[];return a.length===0?`
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
        ${k(t)}
        <div class="surface-scroll">
          <table class="surface">
            <thead>
              <tr>
                <th class="col-endpoint">Endpoint</th>
                <th>Role</th>
                <th>State</th>
                <th>Capabilities</th>
                <th class="col-share">Share of traffic</th>
                <th class="col-act"></th>
              </tr>
            </thead>
            <tbody>
              ${a.map(r=>x(t,r)+q(t,r)).join("")}
            </tbody>
          </table>
        </div>
        ${W(t)}
      </div>
    `}function k(t){const a=w[t.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(J(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[t.id]?"disabled":""}>
          ${R[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(t.id)}">+ Network</button>
      </div>
    `}function x(t,a){return`
      <tr class="band${!a.serviceable?" band-bad":""}">
        <td colspan="6">
          <div class="band-inner">
            <span class="band-id">${a.chainId}</span>
            <span class="band-name">${n(a.name)}</span>
            <code class="band-path">${n(a.path)}</code>
            ${a.url?`<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${n(a.url)}"
                           title="Copy ${n(a.url)}">Copy URL</button>`:""}
            <span class="band-right">
              ${L(t,a)}
              <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                      data-gid="${n(t.id)}" data-chain="${a.chainId}">+ Endpoint</button>
              <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                      data-gid="${n(t.id)}" data-chain="${a.chainId}">Remove</button>
            </span>
          </div>
          ${(a.warnings??[]).map(d=>`<div class="band-warn">${n(d)}</div>`).join("")}
        </td>
      </tr>
    `}function L(t,a){if(!a.serviceable)return U("no usable endpoint","bad");const r=a.upstreams??[],d=r.map(y=>ne(t.id,a.chainId,y.id)).filter(y=>!!y&&!y.unprobeable);return d.length>0&&d.every(y=>S(y,"ws")==="unsupported")?U("subscriptions unavailable","bad"):r.map(y=>X(t.id,a.chainId,y.id)).some((y,I)=>{var C;return y&&y.diverged&&(((C=r[I])==null?void 0:C.local)??!1)})?U("your endpoint is under-used","warn"):U(`${r.length} endpoint${r.length===1?"":"s"}`,"ok")}function q(t,a){const r=a.upstreams??[];return r.length===0?`
        <tr class="ep"><td colspan="6" class="muted small">
          No endpoint yet, so there is nowhere for calls on this path to go.
        </td></tr>
      `:r.map(d=>p(t,a,d)).join("")}function p(t,a,r){const d=`${t.id}|${a.chainId}|${r.id}`,g=r.actions??[];return`
      <tr class="ep${r.problem?" ep-bad":""}">
        <td class="col-endpoint">
          <div class="ep-what">
            ${r.problem?we("bad"):we("ok")}
            <span class="ep-label">${n(r.label)}</span>
          </div>
          <code class="ep-url">${n(r.endpoint||"—")}</code>
          ${r.problem?`<div class="error small">${n(r.problem)}</div>`:""}
        </td>
        <td>${r.local?"Yours":"Public"}</td>
        <td>${v(r)}</td>
        <td>${O(t,a,r)}</td>
        <td class="col-share">${P(t,a,r)}</td>
        <td class="col-act">
          ${g.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(d)}"
                         data-target="${n(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${m[t.id]?"disabled":""}>
                   ${m[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(d)}">Remove</button>
        </td>
      </tr>
    `}function v(t){return t.problem?U("unusable","bad"):t.recentOnly?U("recent blocks","warn"):t.local?U("serving","ok"):U("fallback","neutral")}function S(t,a){var r;if(t)return a==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(r=(t.capabilities??[]).find(d=>d.key===a))==null?void 0:r.status}function O(t,a,r){const d=ne(t.id,a.chainId,r.id);return d?d.unprobeable?`<span class="caps-none" title="${n(d.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Ln.map(g=>c(t,a,d,g)).join("")}</span>`:`<span class="muted small">${w[t.id]===void 0?"probing…":"—"}</span>`}function c(t,a,r,d){const g=(r.capabilities??[]).find(ae=>ae.key===d),y=S(r,d)??"inconclusive",I=Nn[d]??d.toUpperCase();let C="cap";y==="unsupported"?C=f(t,a,d)?"cap missing":"cap off":y==="inconclusive"?C="cap unknown":y==="inconsistent"&&(C="cap mixed");const Y=g!=null&&g.detail?`${g.label}: ${g.detail}`:d==="http"&&r.reachDetail?`Answers JSON-RPC over HTTP: ${r.reachDetail}`:`${I}: no verdict`;return`<span class="${C}" title="${n(Y)}">${n(I)}</span>`}function f(t,a,r){const d=(a.upstreams??[]).map(g=>ne(t.id,a.chainId,g.id)).filter(g=>!!g&&!g.unprobeable);return d.length>0&&d.every(g=>S(g,r)==="unsupported")}function P(t,a,r){const d=h[t.id];if(d===void 0)return'<span class="muted small">reading…</span>';if(d===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!d.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const g=X(t.id,a.chainId,r.id),y=(d.networks??[]).find($e=>$e.chainId===a.chainId);if(!g||!y||y.attributed===0)return'<span class="muted small">no traffic yet</span>';const I=Math.round(g.actual*100),C=Math.round(g.intended*100),Y=g.diverged?r.local?"warn":"":"ok",ae=`${g.succeeded.toLocaleString()} of ${y.attributed.toLocaleString()} answered requests · routing intends ${C}%`+(g.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(ae)}">
        <span class="bar">
          <span class="fill${Y?" "+Y:""}" style="width:${I}%"></span>
          <span class="tick" style="left:${C}%"></span>
        </span>
        <span class="share-n${g.diverged?" warn":""}">${I}%</span>
        ${g.unconfigured?U("not in config","warn"):""}
      </span>
    `}function W(t){const a=h[t.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(J(a.since))})`:""}. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function J(t){const a=new Date(t);return Number.isNaN(a.getTime())?t:a.toLocaleString()}function D(t){const a=t.config;return`
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
        ${ee(t)}
        ${Z(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(t.id)}">Save settings</button>
        </div>
      </div>
    `}function ee(t){const a=!t.config.MetricsOff;return`
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
    `}function Z(t){var I;const a=n(t.id),r=t.config.TLS??null,d=(r==null?void 0:r.Enabled)??!1,g=(r==null?void 0:r.CertSource)||"internal",y=((I=t.tls)==null?void 0:I.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${a}-tls-host" value="${n((r==null?void 0:r.Hostname)??y)}"
               placeholder="${n(y||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${y?`<p class="muted small">
               The default is <code>${n(y)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`:""}
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${a}-tls-port" value="${(r==null?void 0:r.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${a}-tls-source">
          <option value="internal" ${g==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${g==="files"?"selected":""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${a}-tls-cert" value="${n((r==null?void 0:r.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${a}-tls-key" value="${n((r==null?void 0:r.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${Se(t)}
    `}function Se(t){var I,C;const a=n(t.id),r=((I=t.config.TLS)==null?void 0:I.Enabled)??!1,d=_[t.id]??((C=t.tls)==null?void 0:C.verification)??null,g=B[t.id]??!1,y=F[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${r&&!g?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${g?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${y?`<p class="error small">${n(y)}</p>`:""}
      ${d?Ae(d):""}
    `}function Ae(t){const a=(t.assertions??[]).map(r=>`
          <li class="small">
            ${Be(r.status)}
            <strong>${n(r.title)}</strong>
            <div class="muted">${n(r.detail)}</div>
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
    `}function Be(t){switch(t){case"pass":return U("pass","ok");case"fail":return U("fail","bad");case"unavailable":return U("unavailable","warn");default:return U("skipped","neutral")}}async function We(t){B[t]=!0,F[t]=null,G();try{_[t]=await on(t)}catch(a){F[t]=`${he(a)}${Ee(a)}`}finally{B[t]=!1,G()}}function He(t){var g,y;const a=t.tls;if(!(a!=null&&a.enabled))return"";const r=[];a.fallback&&r.push(`<div class="banner banner-warn">${n(a.fallback)}</div>`),a.error?r.push(`<div class="banner banner-warn">HTTPS front: ${n(a.error)}</div>`):((g=a.status)==null?void 0:g.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${n(a.containerName??"")}</code>) is
         ${n(((y=a.status)==null?void 0:y.State)??"unknown")}, so nothing is answering on
         <code>${n(a.url??"")}</code> even if the gateway itself is up.</div>`);const d=_[t.id]??a.verification??null;return d&&(!d.ok||!d.subscriptionsOk)&&r.push(`<div class="banner ${d.ok?"banner-warn":"banner-bad"}">${n(d.summary)}
         <div class="small">Checked ${n(new Date(d.at).toLocaleString())} — open Settings for the full check.</div></div>`),d!=null&&d.expiryWarning&&r.push(`<div class="banner banner-warn">${n(d.expiryWarning)}</div>`),a.rootCaPath&&a.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${n(a.rootCaPath)}</code> (on ${n(t.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function ue(t){return{...t.config,Networks:(t.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(r=>({...r}))}))}}async function Te(t,a,r){N[t]=null;try{await pn(t,a)}catch(d){return N[t]=`${r?r+": ":""}${he(d)}`,G(),!1}return await $(),!0}async function dt(t,a){const r=a.dataset.gid??"";switch(t){case"refresh":await $();return;case"copy":a.dataset.copy&&await Bt(a,a.dataset.copy);return;case"reprobe":await H(r,!0);return;case"toggle-settings":M[r]=!M[r],G();return;case"save-settings":await ut(r);return;case"verify-tls":await We(r);return;case"gw-start":case"gw-stop":case"gw-restart":await ft(r,t.slice(3));return;case"gw-create":case"gw-recreate":await mt(r);return;case"gw-wipe":It(r);return;case"add-gateway":Nt();return;case"forget-gateway":await bt(r);return;case"dismiss-orphan":await gt(a.dataset.name??"");return;case"add-chain":yt(r);return;case"remove-chain":await kt(r,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":Xe(r,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await Tt(a.dataset.key??"");return;case"reset-devnet":await Pt(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function ut(t){const a=z(t);if(!a)return;const r=ue(a),d=s.querySelector(`#gw-${CSS.escape(t)}-port`),g=s.querySelector(`#gw-${CSS.escape(t)}-bind`);if(d){const C=Number.parseInt(d.value.trim(),10);Number.isFinite(C)&&(r.Port=C)}g&&(r.BindAddr=g.value.trim());const y=s.querySelector(`#gw-${CSS.escape(t)}-metrics`);y&&(r.MetricsOff=!y.checked),r.TLS=pt(t,a);const I=a.status.State==="running";await Te(t,r,"Saving settings")&&(M[t]=!1,I&&(N[t]=null,ht(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),G())}function pt(t,a){var y,I,C,Y,ae,$e,et;const r=Ht=>s.querySelector(`#gw-${CSS.escape(t)}-${Ht}`),d=r("tls");if(!d)return a.config.TLS??null;const g=Number.parseInt(((y=r("tls-port"))==null?void 0:y.value.trim())??"",10);return{Enabled:d.checked,Hostname:((I=r("tls-host"))==null?void 0:I.value.trim())??"",CertSource:((C=r("tls-source"))==null?void 0:C.value)??"internal",CertFile:((Y=r("tls-cert"))==null?void 0:Y.value.trim())??"",KeyFile:((ae=r("tls-key"))==null?void 0:ae.value.trim())??"",HTTPSPort:Number.isFinite(g)?g:443,BindAddr:(($e=a.config.TLS)==null?void 0:$e.BindAddr)??"",ImageRef:((et=a.config.TLS)==null?void 0:et.ImageRef)??""}}function ht(t,a){j[t]=[a]}async function ft(t,a){if(!m[t]){m[t]=a,N[t]=null,G();try{await hn(t,a)}catch(r){N[t]=`${a} failed: ${he(r)}${Ee(r)}`}m[t]=null,await $()}}async function mt(t){if(m[t])return;m[t]="create",N[t]=null,j[t]=["starting…"],G();let a;try{a=await fn(t)}catch(r){N[t]=`${he(r)}${Ee(r)}`,j[t]=[],m[t]=null,G();return}T==null||T(),T=Je(a.targetId,r=>{if(i)return;const d=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(j[t]=[...(j[t]??[]).filter(y=>y!=="starting…"),d],!!r.err||r.stepId===An&&!!r.done){T==null||T(),T=null,m[t]=null,r.err&&(N[t]="Provisioning failed — see the log below."),$();return}G()})}async function bt(t){const a=z(t);if(!(!a||!await Le({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await un(t)}catch(d){N[t]=he(d),G();return}await $()}}async function gt(t){if(t){E[t]=null;try{await sn(t)}catch(a){E[t]=he(a),G();return}await $()}}function yt(t){const a=z(t);if(!a)return;const r=new Set((a.networks??[]).map(C=>C.chainId)),d=(o==null?void 0:o.presets)??[],g=d.filter(C=>!r.has(C.chainId)),y=d.filter(C=>r.has(C.chainId)),I=((o==null?void 0:o.targets)??[]).some(C=>C.id===a.placement.targetId&&C.hasDevnet);se(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${g.map(C=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${C.chainId}">
                <span>${n(C.name)}</span>
                <span class="muted small">chain ${C.chainId}${C.devnet?I?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${y.length?`<p class="muted small">Already fronted: ${n(y.map(C=>C.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,C=>{if(C==="cancel"){V();return}if(C==="custom"){vt(t);return}if(C.startsWith("preset:")){const Y=Number.parseInt(C.slice(7),10),ae=d.find($e=>$e.chainId===Y);V(),ae!=null&&ae.devnet?wt(t,Y,I):Ye(t,Y)}})}function vt(t){var a;se(`
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
      `,r=>{if(r==="cancel"){V();return}if(r!=="add")return;const d=document.getElementById("custom-chain-id"),g=document.getElementById("custom-chain-err"),y=Number.parseInt((d==null?void 0:d.value.trim())??"",10);if(!Number.isFinite(y)||y<=0){g&&(g.className="error small"),g&&(g.textContent="A chain id is a positive whole number.");return}V(),Ye(t,y)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function Ye(t,a){const r=z(t);if(!r)return;const d=ue(r),g=d.Networks??[];g.some(y=>y.ChainID===a)||(g.push({ChainID:a,Upstreams:[]}),d.Networks=g,await $t(t,d)&&(G(),Xe(t,a)))}async function $t(t,a){var y;const r={...a,Networks:(a.Networks??[]).filter(I=>I.Upstreams.length>0)};if(!await Te(t,r))return!1;const g=z(t);if(g)for(const I of a.Networks??[])I.Upstreams.length===0&&!(g.networks??[]).some(C=>C.chainId===I.ChainID)&&(g.config.Networks=[...g.config.Networks??[],{ChainID:I.ChainID,Upstreams:[]}],g.networks=[...g.networks??[],{chainId:I.ChainID,name:((y=((o==null?void 0:o.presets)??[]).find(C=>C.chainId===I.ChainID))==null?void 0:y.name)??`Chain ${I.ChainID}`,path:`/${g.config.ProjectID}/evm/${I.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function wt(t,a,r){const d=z(t);if(!d)return;if(!r){se(`
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
        `,()=>V());return}const g=ue(d),y=g.Networks??[],I={ID:"devnet",Kind:"managed-devnet",TargetID:d.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},C=y.find(Y=>Y.ChainID===a);C?C.Upstreams.push(I):y.push({ChainID:a,Upstreams:[I]}),g.Networks=y,await Te(t,g,"Adding the devnet")}async function kt(t,a){const r=z(t);if(!r||!Number.isFinite(a))return;const d=Q(r,a);if(!await Le({title:`Remove ${(d==null?void 0:d.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(d==null?void 0:d.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const y=ue(r);y.Networks=(y.Networks??[]).filter(I=>I.ChainID!==a),await Te(t,y,"Removing the network")}function Ze(t){const a=t.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function Tt(t){const a=Ze(t);if(!a)return;const r=z(a.gid);if(!r)return;const d=ue(r),g=(d.Networks??[]).find(C=>C.ChainID===a.chainId);if(!g)return;const y=g.Upstreams.findIndex((C,Y)=>(C.ID||`${a.chainId}-${Y}`)===a.upstreamId);y<0||!await Le({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(g.Upstreams.splice(y,1),await Te(a.gid,d,"Removing the endpoint"))}function Xe(t,a){const r=z(t);if(!r||!Number.isFinite(a))return;const d=((o==null?void 0:o.sources)??[]).filter(C=>C.chainId===a),g=Q(r,a),y=new Set(((g==null?void 0:g.upstreams)??[]).filter(C=>C.kind!=="external").map(C=>`${C.kind}|${C.targetId??""}`)),I=d.filter(C=>!y.has(`${C.kind}|${C.targetId}`));se(`
        <h2>Add an endpoint for ${n((g==null?void 0:g.name)??`chain ${a}`)}</h2>
        ${I.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${I.map(C=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(C.kind)}:${n(C.targetId)}">
                       <span>${n(C.label)}</span>
                       <span class="muted small">${n(C.endpoint)}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="muted small">No machine you manage serves chain ${a}.</p>`}
        <div class="modal-actions modal-actions-stack">
          <button class="btn btn-ghost" data-modal-action="discover">Find public endpoints…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,C=>{if(C==="cancel"){V();return}if(C==="discover"){St(t,a);return}if(C==="manual"){xt(t,a);return}if(C.startsWith("source:")){const[,Y,ae]=C.split(":");V(),Ct(t,a,Y,ae)}})}async function Ct(t,a,r,d){const g=z(t);if(!g)return;const y=ue(g),I=y.Networks??[],C={ID:`${r==="managed-devnet"?"devnet":"node"}-${d}`,Kind:r,TargetID:d,Endpoint:"",Local:!0,RecentOnly:!1},Y=I.find(ae=>ae.ChainID===a);Y?Y.Upstreams.push(C):I.push({ChainID:a,Upstreams:[C]}),y.Networks=I,await Te(t,y,"Adding the endpoint")}async function St(t,a){se(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,y=>{y==="cancel"&&V()});let r;try{r=await bn(a)}catch(y){const I=qe();if(I){const C=document.createElement("p");C.className="error small",C.textContent=`Could not discover endpoints: ${he(y)}`,I.appendChild(C)}return}if(i)return;const d=(r.endpoints??[]).filter(y=>y.status==="live"||y.status==="unprobed"),g=(r.endpoints??[]).filter(y=>y.status==="rejected");se(`
        <h2>Public endpoints for chain ${a}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${n(r.fetchError)}</div>`:""}</div>`:""}
        ${d.length?`<p class="muted small">${d.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${d.map(y=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(y.url)}">
                       <span><code>${n(y.url)}</code></span>
                       <span class="muted small">${y.status==="live"?`answered in ${y.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${g.length?`<details class="rpc-rejected">
                 <summary class="muted small">${g.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${g.map(y=>`<li class="muted small"><code>${n(y.url)}</code> — ${n(y.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,y=>{if(y==="cancel"){V();return}y.startsWith("add:")&&(V(),Qe(t,a,decodeURIComponent(y.slice(4))))})}function xt(t,a){var r;se(`
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
      `,d=>{if(d==="cancel"){V();return}if(d!=="add")return;const g=document.getElementById("manual-endpoint"),y=document.getElementById("manual-recent"),I=document.getElementById("manual-err"),C=(g==null?void 0:g.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(C)){I&&(I.className="error small",I.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}V(),Qe(t,a,C,(y==null?void 0:y.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Qe(t,a,r,d=!1){const g=z(t);if(!g)return;const y=ue(g),I=y.Networks??[],C=I.find($e=>$e.ChainID===a),Y=((C==null?void 0:C.Upstreams.length)??0)+1,ae={ID:`public-${a}-${Y}`,Kind:"external",Endpoint:r,Local:!1,RecentOnly:d};C?C.Upstreams.push(ae):I.push({ChainID:a,Upstreams:[ae]}),y.Networks=I,await Te(t,y,"Adding the endpoint")}async function Pt(t,a){const r=Ze(t);if(!r||!a||!await Le({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;m[r.gid]="reset",N[r.gid]=null,G();let g;try{g=await nn(a)}catch(y){N[r.gid]=`Reset failed: ${he(y)}${Ee(y)}`,m[r.gid]=null,G();return}m[r.gid]=null,Et(a,g),await $()}function Et(t,a){const r=[];r.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&r.push("A fresh chain was started from genesis.");const d=a.report.Cascaded??[],g=a.report.CascadeSkipped??[];se(`
        <h2>Devnet on ${n(t)} reset</h2>
        <ul class="plain-list">${r.map(y=>`<li>${n(y)}</li>`).join("")}</ul>
        ${d.length?`<p class="ok">Restarted in front of it: ${n(d.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${g.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(g.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>V())}function It(t){const a=z(t);if(!a)return;se(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(t)}</button>
        </div>
      `,g=>{if(g==="cancel"||g==="close"){V(),$();return}g==="confirm"&&Rt(t)});const r=document.getElementById("wipe-confirm-input"),d=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{d&&(d.disabled=r.value.trim()!==t)}),r==null||r.focus()}async function Rt(t){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let r;try{r=await mn(t)}catch(d){const g=qe();if(g){const y=document.createElement("p");y.className="error small",y.textContent=`Wipe failed: ${he(d)}${Ee(d)}`,g.appendChild(y)}a&&(a.disabled=!1,a.textContent=`Wipe ${t}`);return}se(`
        <h2>${n(t)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${n(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{V(),$()})}function Lt(t,a){return!a.some(r=>{var d;return((d=r.placement)==null?void 0:d.targetId)===t})}function Nt(){var y;const t=(o==null?void 0:o.targets)??[],a=(o==null?void 0:o.gateways)??[],r=t.filter(I=>Lt(I.id,a)),d=new Set(a.map(I=>I.id));if(t.length===0){se(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>V());return}if(r.length===0){se(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>V());return}const g=d.has("default")?"":"default";se(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(g)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${r.map(I=>`<option value="${n(I.id)}">${n(I.id)} (${n(I.mode)})</option>`).join("")}
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
      `,I=>{if(I==="cancel"){V();return}I==="create"&&At()}),(y=document.getElementById("new-gw-id"))==null||y.focus()}async function At(){const t=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),d=document.getElementById("new-gw-err"),g=(t==null?void 0:t.value.trim())??"",y=(a==null?void 0:a.value)??"",I=Number.parseInt((r==null?void 0:r.value.trim())??"",10),C=Y=>{d&&(d.className="error small",d.textContent=Y)};if(!g){C("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!y){C("Pick the machine it runs on.");return}try{await rn({id:g,placement:{targetId:y,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(I)?I:4e3,Networks:[]}})}catch(Y){C(he(Y));return}V(),await $()}async function Bt(t,a){const r=await Ne(a),d=t.textContent;t.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{i||(t.textContent=d)},1500)}function he(t){return t instanceof Error?t.message:String(t)}function Ee(t){return t instanceof ke&&t.hint?` — ${t.hint}`:""}return()=>{i=!0,T==null||T(),V()}}const Dn="run",Un={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Mn={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function On(s,i){let o=!1,e=null,h=null;const w={devnet:null},R={devnet:null},m={devnet:[]};let N=null;const j={devnet:!1};let M=null;const _={devnet:null},B={devnet:null};s.innerHTML=`
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
  `;const F=s.querySelector("#services-body");ve(s,(c,f)=>{ge(c,f)}),E();async function E(){try{const c=await Xt(i);if(o)return;e=c,h=null}catch(c){if(o)return;e=null,h=S(c)}u()}function T(c){return e==null?void 0:e.services.find(f=>f.id===c)}function u(){if(!o){if(h){F.innerHTML=`<p class="error">Could not read this machine's services: ${n(h)}</p>`;return}if(!e){F.innerHTML='<p class="muted">Loading…</p>';return}F.innerHTML=`
      ${$(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(A).join("")}
      </div>
    `}}function $(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const f=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(f)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function A(c){const f=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${H(c)}
        </div>
        <p class="muted small">${n(Un[c.id]??"")}</p>

        ${c.error?z(c):""}
        ${c.blocked?`<div class="banner banner-warn">${n(c.blocked)}</div>`:""}
        ${f.map(P=>`<div class="banner banner-warn">${n(P)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${n(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Q(c)}

        ${X(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(P=>ne(c,P)).join("")}
        </div>
        ${R[c.id]?`<p class="error small">${n(R[c.id])}</p>`:""}
        ${G(c)}

        ${de(c)}
      </div>
    `}function H(c){switch(c.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function z(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function Q(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const f=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${f}.</p>`}function X(c){const f=c.endpoints??[];return f.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":f.map(P=>`
        <div class="endpoint-row">
          ${we("ok")}
          <span class="muted small">${n(P.label)}</span>
          <code class="endpoint-url">${n(P.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(P.url)}">Copy</button>
        </div>`).join("")}function ne(c,f){const P=Mn[f];if(!P)return"";const W=w[c.id],J=f==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:P.label;return`
      <button class="${P.className}" data-action="svc-${f}" data-svc="${n(c.id)}"
              title="${n(P.title)}" ${W?"disabled":""}>
        ${W===f?'<span class="spinner" aria-label="working"></span>':n(J)}
      </button>
    `}function G(c){const f=m[c.id]??[];return f.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(f.join(`
`))}</pre>
      </div>
    `}function de(c){const f=j[c.id],P=pe(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(c.id)}">
            ${f?"Close":"Edit"}
          </button>
        </div>
        ${f?ce():`<p class="small">${P}</p>`}
        ${_[c.id]?`<p class="error small">${n(_[c.id])}</p>`:""}
        ${B[c.id]?`<p class="muted small">${n(B[c.id])}</p>`:""}
      </div>
    `}function pe(c){const f=c.devnet;return f?`Chain ${f.ChainID} · a block every ${n(f.BlockTime)} · JSON-RPC on ${n(f.BindAddr)}:${f.HTTPPort} · WebSocket on ${n(f.BindAddr)}:${f.WSPort}`:"—"}function ce(c){return re()}function re(){const c=M;return c?`
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
    `:""}function oe(){j.devnet&&M&&(M.BlockTime=me("#dev-blocktime",M.BlockTime),M.HTTPPort=be("#dev-http",M.HTTPPort),M.WSPort=be("#dev-ws",M.WSPort),M.BindAddr=me("#dev-bind",M.BindAddr))}function me(c,f){const P=s.querySelector(c);return P?P.value.trim():f}function be(c,f){const P=s.querySelector(c);if(!P)return f;const W=Number.parseInt(P.value.trim(),10);return Number.isFinite(W)?W:f}async function ge(c,f){const P=f.dataset.svc??"";switch(c){case"refresh":await E();return;case"copy":f.dataset.copy&&await v(f,f.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await b(P,c.slice(4));return;case"svc-create":case"svc-recreate":await l(P);return;case"svc-wipe":L(P);return;case"toggle-config":k(P);return;case"save-config":await x(P);return;default:return}}async function b(c,f){if(!w[c]){w[c]=f,R[c]=null,u();try{await Qt(i,c,f)}catch(P){R[c]=`${f} failed: ${S(P)}${O(P)}`}w[c]=null,await E()}}async function l(c){if(!w[c]){w[c]="create",R[c]=null,m[c]=["starting…"],u();try{await tn(i,c)}catch(f){R[c]=`${S(f)}${O(f)}`,m[c]=[],w[c]=null,u();return}N==null||N(),N=Je(i,f=>{if(o)return;const P=f.err?`${f.stepId}: ${f.err}`:f.line?`${f.stepId}: ${f.line}`:`${f.stepId}: done`;if(m[c]=[...(m[c]??[]).filter(J=>J!=="starting…"),P],!!f.err||f.stepId===Dn&&!!f.done){N==null||N(),N=null,w[c]=null,f.err&&(R[c]="Provisioning failed — see the log below."),E();return}u()})}}function k(c){if(oe(),j[c]=!j[c],_[c]=null,B[c]=null,j[c]){const f=T(c);f!=null&&f.devnet&&(M={...f.devnet})}u()}async function x(c){var W;oe(),_[c]=null,B[c]=null;const f=M;if(!f)return;if(f.HTTPPort===f.WSPort){_[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await an(i,c,f)}catch(J){_[c]=S(J),u();return}const P=((W=T(c))==null?void 0:W.status.State)==="running";j[c]=!1,B[c]=P?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await E()}function L(c){const f=T(c);if(!f)return;const P=(f.restartsOnWipe??[]).map(D=>{var ee;return((ee=T(D))==null?void 0:ee.label)??D});se(`
        <h2>Wipe ${n(f.label)}</h2>
        <p class="error">This deletes ${n(f.wipeDiscards)}</p>
        ${P.length?`<p>It also restarts what sits in front of it: ${n(P.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${n(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(c)}</button>
        </div>
      `,D=>{if(D==="cancel"||D==="close"){V(),E();return}D==="confirm"&&q(c)});const W=document.getElementById("wipe-confirm-input"),J=document.getElementById("wipe-confirm-btn");W==null||W.addEventListener("input",()=>{J&&(J.disabled=W.value.trim()!==c)}),W==null||W.focus()}async function q(c){const f=document.getElementById("wipe-confirm-btn");f&&(f.disabled=!0,f.textContent="Wiping…");let P;try{P=await en(i,c)}catch(W){const J=qe();if(J){const D=document.createElement("p");D.className="error small",D.textContent=`Wipe failed: ${S(W)}${O(W)}`,J.appendChild(D)}f&&(f.disabled=!1,f.textContent=`Wipe ${c}`);return}p(c,P)}function p(c,f){const P=T(c),W=Z=>{var Se;return((Se=T(Z))==null?void 0:Se.label)??Z},J=[];J.push(f.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const Z of f.report.VolumesRemoved??[])J.push(`Volume ${Z} deleted.`);for(const Z of f.report.VolumesAbsent??[])J.push(`Volume ${Z} was already gone.`);f.report.Recreated&&J.push("Container re-created from your saved configuration.");const D=(f.report.Cascaded??[]).map(W),ee=(f.report.CascadeSkipped??[]).map(W);se(`
        <h2>${n((P==null?void 0:P.label)??c)} wiped</h2>
        <ul class="plain-list">${J.map(Z=>`<li>${n(Z)}</li>`).join("")}</ul>
        ${D.length?`<p class="ok">Restarted in front of it: ${n(D.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${ee.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(ee.join(", "))}.</p>`:""}
        ${f.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(f.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,Z=>{(Z==="close"||Z==="cancel")&&(V(),E())})}async function v(c,f){const P=await Ne(f),W=c.textContent;c.textContent=P?"Copied!":"Copy failed",setTimeout(()=>{o||(c.textContent=W)},1500)}function S(c){return c instanceof Error?c.message:String(c)}function O(c){return c instanceof ke&&c.hint?` — ${c.hint}`:""}return()=>{o=!0,N==null||N(),V()}}const Fn="local";function qn(s){let i=!1,o=!1,e="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ie()}
  `;const w=s.querySelector("#targets-body");ve(s,(u,$)=>{M(u,$)}),R();async function R(){try{const[u,$,A]=await Promise.all([Pe(),xe(),Mt()]);if(i)return;e=A.os,N(u,$)}catch(u){if(i)return;w.innerHTML=`<p class="error">Failed to load machines: ${n(String(u))}</p>`}}function m(){h&&N(h.targets,h.catalog)}function N(u,$){h={targets:u,catalog:$};const A=e==="linux",H=[...u].sort((X,ne)=>(X.mode==="local"?-1:0)-(ne.mode==="local"?-1:0)),z=H.length?`<div class="card-grid">${H.map(X=>jn(X,$,X.mode!=="local"||A,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Q=u.some(X=>X.mode==="local");w.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${z}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(A,Q)}
        ${o?Wn():""}
      </section>
    `}function j(u,$){const A=`
      <div class="card">
        <h3>A server over SSH ${U("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${u?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${u?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,H=u?`
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
      `;return $?`<div class="card-grid card-grid-wide">${A}</div>`:`<div class="card-grid card-grid-wide">${u?H+A:A+H}</div>`}async function M(u,$){var A;if(u==="add-local"){await _();return}if(u==="delete-target"){const H=$.dataset.id;if(!H||!await Le({title:"Remove machine",body:`Remove "${H}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await B(H);return}if(u==="toggle-ssh"){o=!o,T(),m(),o&&((A=s.querySelector("#ssh-host"))==null||A.focus());return}u==="add-ssh"&&await F()}async function _(){T();try{await tt({id:Fn,mode:"local"}),await R()}catch(u){E(u)}}async function B(u){try{await Ot(u),await R()}catch($){E($)}}async function F(){const u=s.querySelector("#ssh-host"),$=s.querySelector("#ssh-user"),A=s.querySelector("#ssh-key"),H=s.querySelector("#ssh-port"),z=s.querySelector("#ssh-id");if(!u||!$||!A||!H||!z)return;const Q=u.value.trim(),X=$.value.trim(),ne=A.value.trim(),G=H.value.trim(),de=z.value.trim();if(T(),!Q||!X||!ne){E(new Error("host, user, and key path are required"));return}const pe=de||_n(Q),ce={Host:Q,User:X,KeyPath:ne};if(G){const oe=Number.parseInt(G,10);if(!Number.isFinite(oe)||oe<=0){E(new Error("port must be a positive number"));return}ce.Port=oe}const re=s.querySelector("#ssh-submit");re&&(re.disabled=!0,re.textContent="Connecting…");try{await tt({id:pe,mode:"ssh",ssh:ce}),o=!1,await R()}catch(oe){E(oe),re&&(re.disabled=!1,re.textContent="Add server")}}function E(u){let $=s.querySelector("#targets-error");$||(w.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=s.querySelector("#targets-error")),$.textContent=String(u instanceof Error?u.message:u)}function T(){var u;(u=s.querySelector("#targets-error"))==null||u.remove()}return()=>{i=!0}}function jn(s,i,o,e){const h=s.wire,w=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:w,m=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(s.id)}">Devnet</a>`;let N,j;if(!h&&!o)N=`${U("can't run a node","warn")} ${U(e||"not Linux","neutral")}`,j=`
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Preview setup wizard</a>
    `;else if(!h)N=U("not set up","neutral"),j=`
      <a class="btn" href="#/setup/${encodeURIComponent(s.id)}">Run setup wizard</a>
      ${m}
    `;else{const M=i.networks.find(B=>B.ChainID===h.ChainID),_=M?M.Name:`chain ${h.ChainID}`;N=`${U(_,"ok")} ${U(h.ExecID,"neutral")} ${U(h.BeaconID,"neutral")}${h.Archive?" "+U("archive","warn"):""}`,j=`
      <a class="btn" href="#/dash/${encodeURIComponent(s.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(s.id)}">Logs</a>
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${n(s.id)}</h2>
      <p class="muted">${R}</p>
      <p>${N}</p>
      <div class="card-actions">
        ${j}
        <button class="btn btn-danger" data-action="delete-target" data-id="${n(s.id)}">Remove</button>
      </div>
    </div>
  `}function Wn(){return`
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
  `}function _n(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ze=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ue=8545,Me=5052,Oe=30303,Kn=[369,943,1],it={369:"default",943:"practise here first"};function zn(s,i){let o=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ie()}</div>`;const h=s.querySelector("#wizard-body"),w=s.querySelector("#wizard-footer");ve(s,(p,v)=>{be(p,v)}),Ve(s,(p,v)=>{p==="exec-select"?e.execId=v:p==="beacon-select"&&(e.beaconId=v),m()}),s.addEventListener("change",p=>{const v=p.target;v instanceof HTMLInputElement&&(v.id==="data-dir-input"?(ge(),ne()):v.id==="checkpoint-toggle"?(e.checkpoint=v.checked,m()):v.id==="exec-snapshot-toggle"&&(e.execSnapshot=v.checked,m()))}),R();async function R(){try{const[p,v]=await Promise.all([xe(),Pe()]);if(o)return;e.catalog=p;const S=v.find(O=>O.id===i);S!=null&&S.wire&&(e.chainId=S.wire.ChainID,e.execId=S.wire.ExecID,e.beaconId=S.wire.BeaconID,e.archive=S.wire.Archive,S.wire.ExecHTTPPort&&(e.execHTTPPort=String(S.wire.ExecHTTPPort)),S.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(S.wire.BeaconHTTPPort)),S.wire.ExecP2PPort&&(e.execP2PPort=String(S.wire.ExecP2PPort)),S.wire.RPCBindAddr&&(e.rpcBindAddr=S.wire.RPCBindAddr)),m()}catch(p){if(o)return;e.loadError=String(p instanceof Error?p.message:p),m()}}function m(){if(e.loadError){h.innerHTML=`<p class="error">Failed to load: ${n(e.loadError)}</p>`;return}e.catalog&&(h.innerHTML=`
      ${q(e.step)}
      ${j()}
    `,N())}function N(){var v;const p=(v=e.catalog)==null?void 0:v.networks.find(S=>S.ChainID===e.chainId);w.innerHTML=p?ie(p.Name,p.LearnURL):ie()}function j(){switch(e.step){case"network":return M();case"clients":return _();case"mode":return re();case"review":return oe();case"run":return me()}}function M(){const p=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Kn.map(S=>{const O=p.networks.find(P=>P.ChainID===S);if(!O)return"";const c=e.chainId===S,f=it[S]?U(it[S],S===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${S}" type="button">
          <h3>${n(O.Name)} <span class="muted">(chain ${S})</span></h3>
          ${f}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function _(){const p=e.catalog,v=p.networks.find(c=>c.ChainID===e.chainId);if(!v)return'<p class="error">Unknown network.</p>';(e.execId===null||!v.ExecClients.includes(e.execId))&&(e.execId=v.ExecClients[0]??null),(e.beaconId===null||!v.BeaconClients.includes(e.beaconId))&&(e.beaconId=v.BeaconClients[0]??null);const S=v.ExecClients.map(c=>de(c,p)),O=v.BeaconClients.map(c=>de(c,p));return`
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
          ${Ge("exec-select",S,e.execId)}
        </label>
        ${ce(e.execId,p)}
        <label>
          Beacon client
          ${Ge("beacon-select",O,e.beaconId)}
        </label>
        ${ce(e.beaconId,p)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function B(p){return p<=0?"—":p>=1?`~${p.toFixed(1)} TB`:`~${Math.round(p*1e3)} GB`}const F=1.1,E=.5,T="Valve reth snapshot",u="rough estimate";function $(p){return p.SnapshotSizeTB}function A(p){return p.SnapshotSizeTB*E}function H(p){return`<p class="muted small">${B($(p))} is the measured size of Valve's reth snapshot for ${n(p.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function z(p){return{archive:$(p)*1e12*F,full:A(p)*1e12*F}}function Q(p,v){if(!p)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${n(v)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${n(v)}</code>: ${n(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==v)return"";const S=z(p),O=e.freeBytes>=S.archive,c=e.freeBytes>=S.full,f=`<p class="muted small">Free at <code>${n(v)}</code>: <strong>${Ce(e.freeBytes)}</strong> — archive ${O?"fits":"won't fit"} (${B($(p))}, ${T}), full ${c?"fits":"won't fit"} (${B(A(p))}, ${u}).</p>`;let P="";return e.downgradeNote?P=`<p class="banner banner-warn">${n(e.downgradeNote)}</p>`:c||(P=`<p class="banner banner-warn">Neither full (${B(A(p))}, ${u}) nor archive (${B($(p))}, ${T}) fits the free space here — choose a location with more room.</p>`),f+P}function X(p,v){if(e.downgradeNote=null,!p||e.freeBytes===null)return;const S=z(p);e.archive&&e.freeBytes<S.archive&&e.freeBytes>=S.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${v} for archive (${B($(p))}, ${T}) — switched to Full (${B(A(p))}, ${u}). Pick a location with more room to run archive.`)}async function ne(){var S;if(e.chainId===null)return;const p=(S=e.catalog)==null?void 0:S.networks.find(O=>O.ChainID===e.chainId),v=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,m();try{const{freeBytes:O}=await Ft(e.targetId,v);if(o)return;e.freeBytes=O,e.probedPath=v,X(p,v)}catch(O){if(o)return;e.freeBytes=null,e.probedPath=v,e.diskError=String(O instanceof Error?O.message:O)}e.diskProbing=!1,m()}function G(p){return p?/^https?:\/\/.+/i.test(p)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(p,v){const S=v.clients.find(O=>O.id===p);return{value:p,label:S?`${S.id} — ${pe(S.repo)}`:p}}function pe(p){const v=p.split("/");return v.length>=4?v[3]:p}function ce(p,v){const S=p?v.clients.find(c=>c.id===p):void 0;if(!S)return"";const O=S.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(S.repo)}" target="_blank" rel="noopener noreferrer">${n(O)}</a></p>`}function re(){var W,J,D;const p=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",v=(W=e.catalog)==null?void 0:W.networks.find(ee=>ee.ChainID===e.chainId),S=((D=(J=e.catalog)==null?void 0:J.clients.find(ee=>ee.id===e.execId))==null?void 0:D.snapshotSupported)??!1,O=v?`${B(A(v))} (${u})`:"Smaller",c=v?`${B($(v))} (${T})`:"Much larger",f=v?` on ${n(v.Name)}`:"",P=v?e.checkpoint?v.SyncLabel:v.GenesisSyncLabel:"";return`
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
          ${v?`<p class="sync-estimate">⏱ Estimated initial sync${f}: <strong>${n(P)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((v==null?void 0:v.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((v==null?void 0:v.CheckpointURL)??"")}" value="${n(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${n(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${S?`
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
              <tr><th>Approx. disk footprint${f}</th><td class="yes">${O}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${v?H(v):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${v?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${O}${v?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(p)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(p)}" value="${n(e.dataDir)}" />
          </label>
          ${Q(v,e.dataDir||p)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(p)}/jwt.hex" value="${n(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Ue})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Ue}" value="${n(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${n(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Me})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Me}" value="${n(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${n(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Oe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Oe}" value="${n(e.execP2PPort)}" />
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
    `}function oe(){const v=e.catalog.networks.find(Z=>Z.ChainID===e.chainId),S=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,O=e.jwtPath||`${S}/jwt.hex`,c=ze.map(Z=>`<li>${n(Z.title)}</li>`).join(""),f=x(e.execHTTPPort,Ue),P=x(e.beaconHTTPPort,Me),W=x(e.execP2PPort,Oe),J=f||P||W?`<tr><th>Non-default ports</th><td>${[f?`exec HTTP ${f}`:null,P?`beacon HTTP ${P}`:null,W?`exec p2p ${W}`:null].filter(Z=>Z!==null).map(n).join(", ")}</td></tr>`:"",{addr:D}=b(e.rpcBindAddr),ee=D?`<tr><th>RPC bind address</th><td><code>${n(D)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((v==null?void 0:v.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(S)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(O)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${n(e.checkpointUrl||(v==null?void 0:v.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${J}
            ${ee}
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
    `}function me(){const v=e.catalog.networks.find(D=>D.ChainID===e.chainId),S=v==null?void 0:v.LearnURL,O=new Set(e.events.filter(D=>D.done).map(D=>D.stepId)),c=new Set(e.events.filter(D=>D.err).map(D=>D.stepId)),f=new Map;for(const D of e.events){if(!D.line)continue;const ee=f.get(D.stepId)??[];ee.push(D.line),f.set(D.stepId,ee)}const P=ze.map(D=>{var He;const ee=O.has(D.id),Z=c.has(D.id),Se=Z?U("failed","bad"):ee?U("done","ok"):U("pending","neutral"),Ae=(f.get(D.id)??[]).slice(-5),Be=(He=e.events.find(ue=>ue.stepId===D.id&&ue.err))==null?void 0:He.err,We=D.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${S?` <a href="${n(S)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${ee?"step-done":""} ${Z?"step-error":""}">
          <div class="step-head">${Se} <strong>${n(D.title)}</strong></div>
          ${We}
          ${Ae.length?`<pre class="step-log">${Ae.map(ue=>n(ue)).join(`
`)}</pre>`:""}
          ${Be?`<p class="error small">${n(Be)}</p>`:""}
        </li>
      `}).join(""),W=e.events.some(D=>D.err),J=ze.every(D=>O.has(D.id))||e.events.some(D=>D.stepId==="handshake"&&D.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${P}</ol>
        ${J&&!W?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        ${W?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function be(p,v){switch(p){case"pick-network":e.chainId=Number(v.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m(),ne();break;case"goto-review":if(ge(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){m();break}e.step="review",m();break;case"start-setup":L();break}}function ge(){const p=s.querySelectorAll('input[name="mode"]');for(const D of Array.from(p))D.checked&&(e.archive=D.value==="archive");const v=s.querySelector("#data-dir-input"),S=s.querySelector("#jwt-path-input");v&&(e.dataDir=v.value.trim()),S&&(e.jwtPath=S.value.trim());const O=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),f=s.querySelector("#exec-p2p-port-input");O&&(e.execHTTPPort=O.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),f&&(e.execP2PPort=f.value.trim());const P=s.querySelector("#rpc-bind-addr-input");P&&(e.rpcBindAddr=P.value.trim());const W=s.querySelector("#checkpoint-url-input");W&&(e.checkpointUrl=W.value.trim());const J=s.querySelector("#snapshot-key-input");J&&(e.snapshotKey=J.value.trim()),e.execHTTPPortError=k(e.execHTTPPort).error??null,e.beaconHTTPPortError=k(e.beaconHTTPPort).error??null,e.execP2PPortError=k(e.execP2PPort).error??null,e.rpcBindAddrError=b(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?G(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function b(p){if(!p)return{};const v=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(p);return v?v.slice(1).every(S=>Number(S)<=255)?{addr:p}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(p)&&p.includes(":")?{addr:p}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const l=/^\d+$/;function k(p){if(!p)return{};if(!l.test(p))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const v=Number(p);return!Number.isInteger(v)||v<1||v>65535?{error:"Port must be between 1 and 65535."}:{port:v}}function x(p,v){const{port:S}=k(p);if(!(S===void 0||S===v))return S}async function L(){var f;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(f=e.streamStop)==null||f.call(e),e.streamStop=null,m();const p={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(p.DataDir=e.dataDir),e.jwtPath&&(p.JWTPath=e.jwtPath);const v=x(e.execHTTPPort,Ue),S=x(e.beaconHTTPPort,Me),O=x(e.execP2PPort,Oe);v!==void 0&&(p.ExecHTTPPort=v),S!==void 0&&(p.BeaconHTTPPort=S),O!==void 0&&(p.ExecP2PPort=O);const{addr:c}=b(e.rpcBindAddr);c!==void 0&&(p.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(p.CheckpointURL=e.checkpointUrl):p.NoCheckpoint=!0,e.execSnapshot&&(p.ExecSnapshot=!0,p.SnapshotKey=e.snapshotKey);try{await qt(e.targetId,p)}catch(P){if(!(P instanceof ke&&P.status===409)){e.starting=!1,e.startError=String(P instanceof Error?P.message:P),m();return}}e.starting=!1,e.step="run",m(),e.streamStop=Je(e.targetId,P=>{o||(e.events.push(P),e.step==="run"&&m())})}function q(p){const v=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],O=v.map(c=>c.id).indexOf(p);return`
      <ol class="wizard-progress">
        ${v.map((c,f)=>`<li class="${f===O?"current":f<O?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var p;o=!0,(p=e.streamStop)==null||p.call(e)}}const Gn=document.querySelector("#app"),{contentEl:Jn,setActiveNav:Vn}=vn(Gn);let le=null;function Yn(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[o,e]=i;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"||o==="analytics"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function ye(s){const i=document.createElement("div");return Jn.replaceChildren(i),s(i)}function lt(){if(le){try{le()}catch{}le=null}const{screen:s,id:i}=Yn();switch(Vn(s),s){case"setup":if(!i){location.hash="#/targets";return}le=ye(o=>zn(o,i));break;case"dash":if(!i){location.hash="#/targets";return}le=ye(o=>Sn(o,i));break;case"logs":if(!i){location.hash="#/targets";return}le=ye(o=>xn(o,i));break;case"security":if(!i){location.hash="#/targets";return}le=ye(o=>En(o,i));break;case"diag":if(!i){location.hash="#/targets";return}le=ye(o=>Pn(o,i));break;case"services":if(!i){location.hash="#/targets";return}le=ye(o=>On(o,i));break;case"analytics":if(!i){location.hash="#/rpc";return}le=ye(o=>Tn(o,i));break;case"rpc":le=ye(o=>Hn(o));break;case"settings":le=ye(o=>Rn(o));break;case"targets":default:le=ye(o=>qn(o));break}}window.addEventListener("hashchange",lt);lt();
