var Ut=Object.defineProperty;var Mt=(s,i,o)=>i in s?Ut(s,i,{enumerable:!0,configurable:!0,writable:!0,value:o}):s[i]=o;var Ue=(s,i,o)=>Mt(s,typeof i!="symbol"?i+"":i,o);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))e(h);new MutationObserver(h=>{for(const $ of h)if($.type==="childList")for(const R of $.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&e(R)}).observe(document,{childList:!0,subtree:!0});function o(h){const $={};return h.integrity&&($.integrity=h.integrity),h.referrerPolicy&&($.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?$.credentials="include":h.crossOrigin==="anonymous"?$.credentials="omit":$.credentials="same-origin",$}function e(h){if(h.ep)return;h.ep=!0;const $=o(h);fetch(h.href,$)}})();function Ot(){return z("/api/host")}function xe(){return z("/api/catalog")}function Pe(){return z("/api/targets")}function tt(s){return z("/api/targets",{method:"POST",headers:me,body:JSON.stringify(s)})}function Ft(s){return z(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function qt(s,i){return z(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function jt(s,i){return z(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:me,body:JSON.stringify(i)})}function Je(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Wt(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function _t(s,i=200){return z(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function Kt(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function nt(s,i){const o=i===void 0?{}:{lines:i};return z(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:me,body:JSON.stringify(o)})}function zt(s,i,o){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/${o}`,{method:"POST"})}function Gt(s,i){return z(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})})}function Jt(s){return z(`/api/targets/${encodeURIComponent(s)}/du`)}function Vt(s){return z(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function Yt(s){return z(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Zt(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function Xt(s){return z(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function Qt(s){return z(`/api/targets/${encodeURIComponent(s)}/containers`)}function en(s,i,o){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${o}`,{method:"POST"})}async function tn(s,i){const o=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:i})}),e=await o.text();let h=null;try{h=e?JSON.parse(e):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const $=h&&typeof h=="object"&&typeof h.error=="string"?h.error:o.statusText||`HTTP ${o.status}`;throw new ke(o.status,$)}function nn(s,i){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function an(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:me}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ke(i.status,h)}function sn(s,i,o){return z(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:me,body:JSON.stringify(o)})}function ct(){return z("/api/gateways")}async function rn(s){await z(`/api/orphans/${encodeURIComponent(s)}`,{method:"DELETE"})}function on(s){return z("/api/gateways",{method:"POST",headers:me,body:JSON.stringify(s)})}function cn(s){return z(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function ln(s){return z(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function dn(s){return z(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function un(s,i=!1){const o=i?"?refresh=1":"";return z(`/api/gateways/${encodeURIComponent(s)}/capabilities${o}`)}function pn(s){return z(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function hn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:me,body:JSON.stringify(i)})}function fn(s,i){return z(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function mn(s){return z(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function bn(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:me,body:JSON.stringify({Confirm:s})}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new ke(i.status,h)}function gn(s){return z(`/api/chainlist/${s}`)}function yn(){return z("/api/settings")}function vn(s){return z("/api/settings",{method:"PUT",headers:me,body:JSON.stringify(s)})}class ke extends Error{constructor(o,e,h,$){super(e);Ue(this,"status");Ue(this,"hint");Ue(this,"code");this.name="ApiError",this.status=o,this.hint=h,this.code=$}}const me={"Content-Type":"application/json"};async function z(s,i){const o=await fetch(s,i);if(!o.ok){let h=o.statusText||`HTTP ${o.status}`,$,R;try{const b=await o.json();b&&typeof b.error=="string"&&b.error&&(h=b.error),b&&typeof b.hint=="string"&&b.hint&&($=b.hint),b&&typeof b.code=="string"&&b.code&&(R=b.code)}catch{}throw new ke(o.status,h,$,R)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const at="https://learn.valve.city/rpc";function n(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function ie(s,i){const o=s&&i&&i!==at?` <span class="footer-sep">·</span> <a href="${n(i)}" target="_blank" rel="noopener noreferrer">${n(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${n(at)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function $n(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),o=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{for(const $ of o)$.classList.toggle("active",$.dataset.nav===h)}}}function ne(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function wn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function kn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),o=Math.floor(i/60),e=i%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const h=Math.floor(o/24),$=o%24;return`~${h}d ${$}h`}function U(s,i){return`<span class="badge badge-${i}">${n(s)}</span>`}function we(s){return`<span class="dot dot-${s}"></span>`}const st=["B","KB","MB","GB","TB","PB"];function Ce(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,o=0;for(;i>=1024&&o<st.length-1;)i/=1024,o++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${st[o]}`}async function Ae(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function $e(s,i){s.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!s.contains(e))return;const h=e.dataset.action;h&&i(h,e,o)})}function Ge(s,i,o){const e=i.find($=>$.value===o),h=i.map($=>`
      <li class="dropdown-option${$.value===o?" selected":""}" role="option"
          aria-selected="${$.value===o}" data-value="${n($.value)}">
        ${n($.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${n(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${n(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Ie(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var o;i.classList.remove("open"),(o=i.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function Ve(s,i){s.addEventListener("click",h=>{const $=h.target,R=$.closest(".dropdown-trigger");if(R&&s.contains(R)){const N=R.closest(".dropdown"),j=!!N&&!N.classList.contains("open");Ie(s),N&&j&&(N.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const b=$.closest(".dropdown-option");if(b&&s.contains(b)){const N=b.closest(".dropdown");Ie(s),i((N==null?void 0:N.dataset.dropdown)??"",b.dataset.value??"");return}Ie(s)});const o=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const $=h.target;(!$.closest(".dropdown")||!s.contains($))&&Ie(s)},e=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}h.key==="Escape"&&Ie(s)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const je="app-modal";let qe=null;function se(s,i){Y();const o=document.createElement("div");o.className="modal-overlay",o.id=je,o.innerHTML=`<div class="modal">${s}</div>`,o.addEventListener("click",h=>{const $=h.target.closest("[data-modal-action]");$!=null&&$.dataset.modalAction?i($.dataset.modalAction):h.target===o&&i("cancel")});const e=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),qe=e,document.body.appendChild(o)}function Y(){var s;(s=document.getElementById(je))==null||s.remove(),qe&&(document.removeEventListener("keydown",qe),qe=null)}function Ne(){return document.querySelector(`#${je} .modal`)}function Le(s){return new Promise(i=>{var h;let o=!1;const e=$=>{o||(o=!0,Y(),i($))};se(`
        <h2>${n(s.title)}</h2>
        <p>${n(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${n(s.confirmLabel)}</button>
        </div>
      `,$=>e($==="confirm")),(h=document.querySelector(`#${je} [data-modal-action="confirm"]`))==null||h.focus()})}const _e=5e3,Tn=60;function Cn(s,i){let o=!1,e=null,h=null,$=null,R=null;const b=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${ie()}</div>`;const N=s.querySelector("#an-body");$e(s,(g,l)=>{var w;g==="toggle-endpoint"&&((w=l.closest(".an-endpoint"))==null||w.classList.toggle("expanded"))}),j();async function j(){try{e=((await ct()).gateways??[]).find(l=>l.id===i)??null}catch(g){if(o)return;$=String(g instanceof Error?g.message:g),B();return}if(!o){if(!e){B();return}await M(),R=window.setInterval(()=>void M(),_e)}}async function M(){try{const g=await dn(i);if(o)return;_(g),h=g,$=null}catch(g){if(o)return;$=String(g instanceof Error?g.message:g)}B()}function _(g){if(!g.enabled||g.error)return;const l=b[b.length-1];l&&l.since!==g.since&&(b.length=0);const w=new Map;for(const P of g.networks??[])w.set(P.chainId,P.received);b.push({t:Date.now(),since:g.since,received:w}),b.length>Tn&&b.shift()}function B(){o||(N.innerHTML=F())}function F(){return $&&!h?`<h1>Analytics</h1><p class="error">${n($)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${I(e)}
      ${h?u(h):`<p class="muted">Reading the gateway's counters…</p>`}
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
    `}function C(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const g=h.since?new Date(h.since):null;return g&&!Number.isNaN(g.getTime())?`totals since the gateway started, ${n(g.toLocaleString())}<br />re-read every ${_e/1e3}s`:`re-read every ${_e/1e3}s`}function u(g){return g.enabled?g.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${n(g.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:v(g)+oe(g):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function v(g){const l=g.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${l.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':l.map(w=>A(w)).join("")}
      </section>
    `}function A(g){const l=g.methods??[],w=g.endpoints??[],P=g.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${g.chainId}</span>
          <span class="band-name">${n(g.name)}</span>
          ${G(g)}
        </div>
        <div class="an-stats">
          ${H("Received",ne(g.received),"what clients asked this chain for")}
          ${H("Answered",ne(g.answered),"returned by one of your endpoints")}
          ${H("From cache",ne(g.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${H("Failed",ne(g.failed),"asked for and never answered",g.failed>0?"bad":"")}
        </div>
        ${Q(g.chainId)}
        ${P?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':J("Method",l.map(L=>({label:L.method,l:L})))+J("Endpoint",w.map(L=>({label:L.upstream,l:L})))+ae(g)}
      </div>
    `}function H(g,l,w,P=""){return`
      <div class="an-stat${P?" an-stat-"+P:""}" title="${n(w)}">
        <span class="an-stat-n">${n(l)}</span>
        <span class="an-stat-l">${n(g)}</span>
      </div>
    `}function G(g){const l=ee(g.chainId);if(l===null)return'<span class="an-rate muted small">measuring rate…</span>';const w=Math.round((b[b.length-1].t-b[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${w}s apart.">
      ${n(l.toFixed(l<10?2:0))} req/s <span class="muted">over the last ${w}s</span>
    </span>`}function ee(g){if(b.length<2)return null;const l=b[0],w=b[b.length-1],P=(w.t-l.t)/1e3;if(P<=0)return null;const L=(w.received.get(g)??0)-(l.received.get(g)??0);return L<0?null:L/P}function Q(g){if(b.length<3)return"";const l=[];for(let y=1;y<b.length;y++){const x=b[y-1],O=b[y],c=(O.t-x.t)/1e3,m=(O.received.get(g)??0)-(x.received.get(g)??0);l.push(c>0&&m>=0?m/c:0)}const w=Math.max(...l);if(w<=0)return"";const P=240,L=28,q=l.length>1?P/(l.length-1):P,p=l.map((y,x)=>`${(x*q).toFixed(1)},${(L-y/w*L).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${w.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${P} ${L}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${p}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${n(w.toFixed(2))} req/s</span>
      </div>
    `}function ae(g){const l=[];return g.cached.count>0&&l.push(`${n(ne(g.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${g.cached.mean===null?"":`, in ${n(Re(g.cached.mean))} on average`}.`),g.failedLatency.count>0&&g.failedLatency.mean!==null&&l.push(`The ${n(ne(g.failedLatency.count))} that failed took
         ${n(Re(g.failedLatency.mean))} on average to fail.`),l.length===0?"":`<p class="muted small">${l.join(" ")}</p>`}function J(g,l){return l.length===0?"":`
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
            ${l.map(w=>de(w.label,w.l)).join("")}
          </tbody>
        </table>
      </div>
    `}function de(g,l){return`
      <tr>
        <td><code>${n(g)}</code></td>
        <td class="an-num">${ne(l.count)}</td>
        <td class="an-num">${l.mean===null?'<span class="muted">—</span>':n(Re(l.mean))}</td>
        <td>${pe(l)}</td>
      </tr>
    `}function pe(g){const l=g.buckets??[];if(l.length===0||g.count===0)return'<span class="muted small">—</span>';let w=0;const P=[];for(const q of l){const p=q.count-w;w=q.count,P.push({label:re(q.le),n:Math.max(0,p)})}return P.reduce((q,p)=>q+p.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${n(P.filter(q=>q.n>0).map(q=>`${q.n} ${q.label}`).join(" · "))}">
        ${P.map((q,p)=>q.n===0?"":`<span class="an-band an-band-${Math.min(p,4)}" style="flex:${q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${n(ce(P))}</span>
    `}function ce(g){for(let l=g.length-1;l>=0;l--)if(g[l].n>0)return`slowest ${g[l].label}`;return""}function re(g){if(g==="+Inf")return"30s or more";const l=Number(g);return Number.isFinite(l)?`under ${Re(l)}`:`under ${g}`}function oe(g){const l=g.endpoints??[];return`
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
                     <tbody>${l.map(w=>be(w)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function be(g){const l=g.errors??[],w=l.reduce((L,q)=>L+q.count,0),P=l.length>0;return`
      <tr class="an-endpoint${P?" expandable":""}" ${P?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${n(g.upstream)}</code>
          ${g.chainId?`<span class="muted small">chain ${g.chainId}</span>`:""}
          ${g.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${ne(g.requests)}</td>
        <td class="an-num${w>0?" bad":""}">${w>0?ne(w):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${g.headLag>0?ne(g.headLag):'<span class="muted">0</span>'}</td>
        <td>${ge(g)}</td>
      </tr>
      ${P?ye(g,l):""}
    `}function ge(g){const l=[];return g.scored?(l.push(g.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${n(String(g.position))}</span>`),l.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${n(g.score.toFixed(3))}</span>`),g.primarySwitches>1&&l.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${ne(g.primarySwitches)} switches</span>`),g.excludedSeconds>0&&l.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${n(Re(g.excludedSeconds))}</span>`),`<span class="an-selection">${l.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function ye(g,l){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${l.map(w=>`
                    <tr>
                      <td class="an-num">${ne(w.count)}</td>
                      <td><code>${n(w.class)}</code></td>
                      <td>${w.severity?`<span class="badge badge-${w.severity==="critical"?"bad":"warn"}">${n(w.severity)}</span>`:""}</td>
                      <td class="muted small">${n(w.method||"")}</td>
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
    `}return()=>{o=!0,R!==null&&window.clearInterval(R)}}function Re(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}const Sn=85,Ke={exec:"Execution",beacon:"Beacon"};function xn(s,i){let o=!1,e=null,h=null,$=null,R=null,b=null,N=null,j=null,M=null;const _={exec:null,beacon:null};let B=null;s.innerHTML=`<h1>Dashboard: ${n(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${ie()}</div>`;const F=s.querySelector("#dash-body"),I=s.querySelector("#dash-footer");F.addEventListener("click",l=>{const w=l.target.closest("[data-action]");if(!w||!F.contains(w))return;const P=w.dataset.action;if(P==="svc-action"){const L=w.dataset.svc,q=w.dataset.kind;L&&q&&be(L,q)}else if(P==="open-clear"){const L=w.dataset.svc;L&&ye(L)}else if(P==="copy"){const L=w.dataset.copy;L&&ge(w,L)}else P==="retry-du"?u():P==="retry-endpoints"&&v()}),C();async function C(){let l,w;try{const[L,q]=await Promise.all([Pe(),xe()]);l=L.find(p=>p.id===i),w=q}catch(L){if(o)return;F.innerHTML=`<p class="error">Failed to load target: ${n(String(L))}</p>`;return}if(o)return;if(!l){F.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){F.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const P=w==null?void 0:w.networks.find(L=>L.ChainID===l.wire.ChainID);P&&(I.innerHTML=ie(P.Name,P.LearnURL)),F.innerHTML='<p class="muted">Connecting…</p>',e=Wt(i,L=>{o||(A(L),h=L,$=L,H())}),u(),v()}async function u(){N=null;try{b=await Jt(i)}catch(l){b=null,N=String(l instanceof Error?l.message:l)}o||H()}async function v(){M=null;try{j=await Vt(i)}catch(l){j=null,M=String(l instanceof Error?l.message:l)}o||H()}function A(l){if(!h)return;const w=(new Date(l.at).getTime()-new Date(h.at).getTime())/1e3,P=l.execHead-h.execHead;if(w>0&&P>=0){const L=P/w;R=R===null?L:R*.7+L*.3}}function H(){if(!$)return;const l=$;F.innerHTML=`
      <p class="dash-status">${G(l)}</p>
      <div class="card-grid">
        ${re(l)}
        ${Q(l)}
        ${ae(l)}
        ${J(l)}
        ${de(l)}
        ${pe()}
      </div>
      <p class="muted small">Last updated ${n(new Date(l.at).toLocaleTimeString())}</p>
    `}function G(l){return!l.execActive&&!l.beaconActive?U("Node not running","bad"):l.execSyncing||l.beaconDistance>0?U("Syncing","warn"):U("Running · synced","ok")}function ee(l){const P=l.refHead>0?l.refHead-l.execHead:null,L=P!==null&&P>0&&R&&R>0?kn(P/R):P!==null&&P<=0?"caught up":"—";return{lag:P,eta:L}}function Q(l){const{lag:w,eta:P}=ee(l);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${l.execActive?l.execSyncing?U("syncing","warn"):l.execHead===0?U("no data","neutral"):U("synced","ok"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${ne(l.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${w!==null?ne(l.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${w!==null?ne(Math.max(w,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${P}</dd></div>
        </dl>
      </div>
    `}function ae(l){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${l.beaconActive?l.beaconSlot===0?U("no data","neutral"):l.beaconDistance===0?U("synced","ok"):U("syncing","warn"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${ne(l.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${ne(l.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function J(l){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${ne(l.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${ne(l.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(l){const w=l.diskUsedPct>=Sn,P=`
      <div class="meter"><div class="meter-fill ${w?"meter-warn":""}" style="width:${Math.min(l.diskUsedPct,100)}%"></div></div>
      <p>${wn(l.diskUsedPct)} used</p>
    `;if(N)return`
        <div class="card ${w?"card-warn":""}">
          <h3>Storage</h3>
          ${P}
          <p class="error small">${n(N)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!b)return`
        <div class="card ${w?"card-warn":""}">
          <h3>Storage</h3>
          ${P}
          <p class="muted">Loading…</p>
        </div>
      `;const L=b.ExpectedExecBytes>0?Math.min(b.ExecBytes/b.ExpectedExecBytes*100,100):0,q=b.ExpectedBeaconBytes>0?Math.min(b.BeaconBytes/b.ExpectedBeaconBytes*100,100):0,{lag:p,eta:y}=ee(l),x=p!==null&&p>0&&R!==null&&R>0;return`
      <div class="card ${w?"card-warn":""}">
        <h3>Storage</h3>
        ${P}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Ce(b.ExecBytes)} of ~${Ce(b.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${L}%"></div></div>
        ${x?`<p class="muted small">Estimated time remaining: ${n(y)}</p>`:""}
        <p class="muted small">Beacon — ${Ce(b.BeaconBytes)} of ~${Ce(b.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Ce(b.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${n(b.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${n(b.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function pe(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${n(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const l=j,w=l.ExecReachable&&!l.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",P=l.Access==="ssh"?`
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
        ${w}
        ${P}
      </div>
    `}function ce(l,w){const P=Ke[l],L=_[l],q=(p,y,x)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${l}" data-kind="${p}" ${L!==null||x?"disabled":""}>${L===p?oe():n(y)}</button>`;return`
      <div class="service-row">
        <span>${n(P)} ${w?U("active","ok"):U("down","bad")}</span>
        <div class="service-actions">
          ${q("start","Start",w)}
          ${q("stop","Stop",!w)}
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
    `}function oe(){return'<span class="spinner" aria-label="working"></span>'}async function be(l,w){if(_[l]===null){_[l]=w,B=null,H();try{await zt(i,l,w)}catch(P){B=`${Ke[l]} ${w} failed: ${P instanceof Error?P.message:String(P)}`}_[l]=null,o||H()}}async function ge(l,w){const P=await Ae(w),L=l.textContent;l.textContent=P?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=L)},1500)}function ye(l){const w=Ke[l],P=b?Ce(l==="exec"?b.ExecBytes:b.BeaconBytes):"unknown (disk usage hasn't loaded)";se(`
        <h2>Clear ${n(w)} data</h2>
        <p class="error">
          This stops the ${n(w.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${n(P)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${n(l)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,p=>{if(p==="cancel"){Y();return}p==="confirm"&&g(l)});const L=document.getElementById("clear-confirm-input"),q=document.getElementById("clear-confirm-btn");L==null||L.addEventListener("input",()=>{q&&(q.disabled=L.value.trim()!==l)}),L==null||L.focus()}async function g(l){const w=document.getElementById("clear-confirm-btn");w&&(w.disabled=!0,w.textContent="Clearing…");try{await Gt(i,l),Y(),u()}catch(P){const L=Ne();if(L){const q=document.createElement("p");q.className="error small",q.textContent=`Clear failed: ${P instanceof Error?P.message:String(P)}`,L.appendChild(q)}w&&(w.disabled=!1,w.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),Y()}}const rt=500,ot="valve-node-app.explain-consent";function Pn(s,i){let o=!1,e=null;const h=[];s.innerHTML=`
    <h1>Logs: ${n(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${ie()}</div>
  `;const $=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");$e(s,C=>{C==="explain"&&M()}),b();async function b(){let C,u;try{const[A,H]=await Promise.all([Pe(),xe()]);C=A.find(G=>G.id===i),u=H}catch(A){if(o)return;$.innerHTML=`<p class="error">Failed to load target: ${n(String(A))}</p>`;return}if(o)return;if(!C){$.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){$.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const v=u==null?void 0:u.networks.find(A=>A.ChainID===C.wire.ChainID);v&&(R.innerHTML=ie(v.Name,v.LearnURL));try{const A=await _t(i,200);if(o)return;h.push(...A)}catch(A){if(o)return;$.innerHTML=`<p class="error">Failed to load logs: ${n(String(A))}</p>`;return}N(),e=Kt(i,A=>{o||(h.push(A),h.length>rt&&h.splice(0,h.length-rt),N())})}function N(){const C=h.filter(v=>v.severity==="error"||v.severity==="critical");$.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${h.map(j).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${U(String(C.length),C.length?"bad":"neutral")}</h2>
          <div class="log-lines">${C.length?C.slice().reverse().map(j).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=$.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function j(C){const u=C.severity||"info",v=C.learnUrl?` <a href="${n(C.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${n(u)}">
        <span class="log-time">${n(new Date(C.at).toLocaleTimeString())}</span>
        <span class="log-unit">${n(C.unit)}</span>
        <span class="log-sev">${n(u)}</span>
        <span class="log-text">${n(C.line)}</span>
        ${C.explain?`<div class="log-explain">${n(C.explain)}${v}</div>`:""}
      </div>
    `}async function M(){const C=h.filter(v=>v.severity==="error"||v.severity==="critical").map(v=>v.line).slice(-40);if(!(localStorage.getItem(ot)==="1")){_(C);return}await B(C)}function _(C){const u=C.length?`<pre class="explain-excerpt">${C.map(v=>n(v)).join(`
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
    `,v=>{v==="proceed"?(localStorage.setItem(ot,"1"),I(),B(C)):I()})}async function B(C){F('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=C.length?await nt(i,C):await nt(i);if(o)return;F(`
        <h2>Explanation</h2>
        <div class="explain-text">${n(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map(v=>n(v)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,v=>{v==="close"&&I()})}catch(u){if(o)return;if(u instanceof ke&&u.status===409){F(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,v=>{v==="close"&&I()});return}F(`
        <h2>Explain failed</h2>
        <p class="error">${n(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,v=>{v==="close"&&I()})}}function F(C,u){I();const v=document.createElement("div");v.className="modal-overlay",v.id="explain-modal",v.innerHTML=`<div class="modal">${C}</div>`,v.addEventListener("click",A=>{const H=A.target.closest("[data-modal-action]");H!=null&&H.dataset.modalAction&&u(H.dataset.modalAction),A.target===v&&u("cancel")}),document.body.appendChild(v)}function I(){var C;(C=document.getElementById("explain-modal"))==null||C.remove()}return()=>{o=!0,e==null||e(),I()}}function En(s,i){let o=!1,e=null,h=null,$=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${n(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${ie()}</div>`;const b=s.querySelector("#diag-body"),N=s.querySelector("#diag-footer");$e(s,(u,v)=>{var A;if(u==="run")M();else if(u==="toggle")(A=v.closest(".check-item"))==null||A.classList.toggle("expanded");else if(u==="copy"){const H=v.dataset.copy;H&&C(v,H)}}),j();async function j(){let u,v;try{const[H,G]=await Promise.all([Pe(),xe()]);u=H.find(ee=>ee.id===i),v=G}catch(H){if(o)return;b.innerHTML=`<p class="error">Failed to load target: ${n(String(H))}</p>`;return}if(o)return;if(!u){b.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!u.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const A=v==null?void 0:v.networks.find(H=>H.ChainID===u.wire.ChainID);A&&(N.innerHTML=ie(A.Name,A.LearnURL));try{e=await Xt(i),R=!0}catch(H){h=String(H instanceof Error?H.message:H)}o||_()}async function M(){$=!0,h=null,_();try{e=await Zt(i),R=!0}catch(u){h=String(u instanceof Error?u.message:u)}$=!1,o||_()}function _(){b.innerHTML=`
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
      ${B()}
    `}function B(){if(!R&&!h)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const u=new Date(e.at).toLocaleString(),v=e.failedId?`<p><strong>Failed at: ${n(F(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${n(u)} — trigger: ${n(e.trigger)}</p>
      ${v}
      <ul class="check-list">${e.items.map(I).join("")}</ul>
    `}function F(u){var v;return((v=e==null?void 0:e.items.find(A=>A.ID===u))==null?void 0:v.Title)??u}function I(u){const v=u.Status==="pass"?"ok":u.Status==="fail"?"bad":u.Status==="warn"?"warn":"neutral",A=u.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${A?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${U(A?"failed here":u.Status,v)}
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
    `}async function C(u,v){const A=await Ae(v),H=u.textContent;u.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{o||(u.textContent=H)},1500)}return()=>{o=!0}}function In(s,i){let o=!1,e=[],h=null,$=!1,R=!1;s.innerHTML=`<h1>Security: ${n(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${ie()}</div>`;const b=s.querySelector("#sec-body"),N=s.querySelector("#sec-footer");$e(s,(I,C)=>{var u;if(I==="rerun")M();else if(I==="toggle")(u=C.closest(".check-item"))==null||u.classList.toggle("expanded");else if(I==="copy"){const v=C.dataset.copy;v&&F(C,v)}}),j();async function j(){let I,C;try{const[v,A]=await Promise.all([Pe(),xe()]);I=v.find(H=>H.id===i),C=A}catch(v){if(o)return;b.innerHTML=`<p class="error">Failed to load target: ${n(String(v))}</p>`;return}if(o)return;if(!I){b.innerHTML=`<p class="error">Target "${n(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!I.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const u=C==null?void 0:C.networks.find(v=>v.ChainID===I.wire.ChainID);u&&(N.innerHTML=ie(u.Name,u.LearnURL)),await M()}async function M(){$=!0,h=null,_();try{e=await Yt(i),R=!0}catch(I){h=String(I instanceof Error?I.message:I)}$=!1,o||_()}function _(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${$?"disabled":""}>${$?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${n(h)}</p>`:""}
      ${!R&&$?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(B).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function B(I){const C=I.Status==="pass"?"ok":I.Status==="fail"?"bad":I.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${U(I.Status,C)}
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
    `}async function F(I,C){const u=await Ae(C),v=I.textContent;I.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{o||(I.textContent=v)},1500)}return()=>{o=!0}}const Rn=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Ln(s){let i=!1,o=!1,e=!1,h=null,$=!1,R=null,b=null;s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${ie()}`;const N=s.querySelector("#settings-body");$e(s,B=>{if(B==="save"&&_(),B==="clear-key"){if(!R)return;o=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),M(R)}}),Ve(s,(B,F)=>{B!=="ai-provider"||!R||(b=F,$=!1,M(R))}),j();async function j(){try{const B=await yn();if(i)return;R=B,M(B)}catch(B){if(i)return;N.innerHTML=`<p class="error">Failed to load settings: ${n(String(B))}</p>`}}function M(B){var C;const F=b??B.aiProvider;N.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${Ge("ai-provider",Rn.map(u=>({value:u.value,label:u.label})),F)}
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
        ${$?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const I=s.querySelector("#ai-key");I==null||I.addEventListener("input",()=>{o=!0,$=!1}),(C=s.querySelector("#ref-rpc-base"))==null||C.addEventListener("input",()=>{$=!1})}async function _(){const B=s.querySelector("#ai-key"),F=s.querySelector("#ref-rpc-base");if(!B||!F||!R)return;const I={aiProvider:b??R.aiProvider,refRpcBase:F.value.trim()};o&&(I.aiKey=B.value),e=!0,h=null,$=!1,M(R);try{const C=await vn(I);if(i)return;R=C,o=!1,e=!1,$=!0,M(C)}catch(C){if(i)return;e=!1,h=String(C instanceof Error?C.message:C),M(R)}}return()=>{i=!0}}const Nn=["http","ws","archive","trace"],An={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},Bn="run",Hn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Dn(s){let i=!1,o=null,e=null;const h={},$={},R={},b={},N={},j={},M={},_={},B={},F={},I={};let C=null;s.innerHTML=`
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
  `;const u=s.querySelector("#rpc-body");$e(s,(t,a)=>{dt(t,a)}),Ve(s,()=>{}),v();async function v(){try{const t=await ct();if(i)return;o=t,e=null}catch(t){if(i)return;o=null,e=he(t)}J();for(const t of(o==null?void 0:o.gateways)??[])A(t.id),H(t.id,!1)}async function A(t){try{const a=await ln(t);if(i)return;h[t]=a}catch{if(i)return;h[t]=null}J()}async function H(t,a){R[t]=a,a&&J();try{const r=await un(t,a);if(i)return;$[t]=r}catch{if(i)return;$[t]=null}R[t]=!1,J()}function G(t){return((o==null?void 0:o.gateways)??[]).find(a=>a.id===t)}function ee(t,a){return(t.networks??[]).find(r=>r.chainId===a)}function Q(t,a,r){var f;const d=(((f=h[t])==null?void 0:f.networks)??[]).find(k=>k.chainId===a);return((d==null?void 0:d.upstreams)??[]).find(k=>k.upstream===r)}function ae(t,a,r){var d;return(((d=$[t])==null?void 0:d.endpoints)??[]).find(f=>f.chainId===a&&f.upstream===r)}function J(){if(i)return;if(e){u.innerHTML=`<p class="error">Could not read the gateways: ${n(e)}</p>`;return}if(!o){u.innerHTML='<p class="muted">Loading…</p>';return}const t=o.gateways??[];u.innerHTML=`
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
        ${I[t.containerName]?`<div class="error small">${n(I[t.containerName])}</div>`:""}
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
        ${t.error?ge(t):""}
        ${t.blocked?`<div class="banner banner-warn">${n(t.blocked)}</div>`:""}
        ${(t.warnings??[]).map(a=>`<div class="banner banner-warn">${n(a)}</div>`).join("")}
        ${De(t)}
        ${N[t.id]?`<p class="error small">${n(N[t.id])}</p>`:""}
        ${g(t)}
        ${M[t.id]?D(t):""}
        ${l(t)}
      </section>
    `}function re(t){var r;const a=t.status.State==="running";return`
      <div class="rpc-bar${a?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${be(t)}
            <strong>${n(t.label)}</strong>
            ${oe(t)}
            <span class="muted small">on ${n(t.placement.targetId)} · ${n(t.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(t.actions??[]).map(d=>ye(t,d)).join("")}
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
    `}function oe(t){switch(t.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function be(t){return t.status.State==="running"?we("ok"):t.status.State==="unknown"?we("bad"):we("neutral")}function ge(t){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${n(t.error??"")}</div>
        ${t.hint?`<div class="small">${n(t.hint)}</div>`:""}
      </div>
    `}function ye(t,a){const r=Hn[a];if(!r)return"";const d=b[t.id];return`
      <button class="${r.className}" data-action="gw-${a}" data-gid="${n(t.id)}"
              title="${n(r.title)}" ${d?"disabled":""}>
        ${d===a?'<span class="spinner" aria-label="working"></span>':n(r.label)}
      </button>
    `}function g(t){const a=j[t.id]??[];return a.length===0?"":`
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
        ${w(t)}
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
              ${a.map(r=>P(t,r)+q(t,r)).join("")}
            </tbody>
          </table>
        </div>
        ${W(t)}
      </div>
    `}function w(t){const a=$[t.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${n(V(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${n(t.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[t.id]?"disabled":""}>
          ${R[t.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${n(t.id)}">+ Network</button>
      </div>
    `}function P(t,a){return`
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
    `}function L(t,a){if(!a.serviceable)return U("no usable endpoint","bad");const r=a.upstreams??[],d=r.map(k=>ae(t.id,a.chainId,k.id)).filter(k=>!!k&&!k.unprobeable);return d.length>0&&d.every(k=>x(k,"ws")==="unsupported")?U("subscriptions unavailable","bad"):r.map(k=>Q(t.id,a.chainId,k.id)).some((k,S)=>{var T;return k&&k.diverged&&(((T=r[S])==null?void 0:T.local)??!1)})?U("your endpoint is under-used","warn"):U(`${r.length} endpoint${r.length===1?"":"s"}`,"ok")}function q(t,a){const r=a.upstreams??[];return r.length===0?`
        <tr class="ep"><td colspan="6" class="muted small">
          No endpoint yet, so there is nowhere for calls on this path to go.
        </td></tr>
      `:r.map(d=>p(t,a,d)).join("")}function p(t,a,r){const d=`${t.id}|${a.chainId}|${r.id}`,f=r.actions??[];return`
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
        <td>${y(r)}</td>
        <td>${O(t,a,r)}</td>
        <td class="col-share">${E(t,a,r)}</td>
        <td class="col-act">
          ${f.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${n(d)}"
                         data-target="${n(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${b[t.id]?"disabled":""}>
                   ${b[t.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${n(d)}">Remove</button>
        </td>
      </tr>
    `}function y(t){return t.problem?U("unusable","bad"):t.recentOnly?U("recent blocks","warn"):t.local?U("serving","ok"):U("fallback","neutral")}function x(t,a){var r;if(t)return a==="http"?t.unprobeable?"inconclusive":t.reachable?"supported":"unsupported":(r=(t.capabilities??[]).find(d=>d.key===a))==null?void 0:r.status}function O(t,a,r){const d=ae(t.id,a.chainId,r.id);return d?d.unprobeable?`<span class="caps-none" title="${n(d.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Nn.map(f=>c(t,a,d,f)).join("")}</span>`:`<span class="muted small">${$[t.id]===void 0?"probing…":"—"}</span>`}function c(t,a,r,d){const f=(r.capabilities??[]).find(Z=>Z.key===d),k=x(r,d)??"inconclusive",S=An[d]??d.toUpperCase();let T="cap";k==="unsupported"?T=m(t,a,d)?"cap missing":"cap off":k==="inconclusive"?T="cap unknown":k==="inconsistent"&&(T="cap mixed");const K=f!=null&&f.detail?`${f.label}: ${f.detail}`:d==="http"&&r.reachDetail?`Answers JSON-RPC over HTTP: ${r.reachDetail}`:`${S}: no verdict`;return`<span class="${T}" title="${n(K)}">${n(S)}</span>`}function m(t,a,r){const d=(a.upstreams??[]).map(f=>ae(t.id,a.chainId,f.id)).filter(f=>!!f&&!f.unprobeable);return d.length>0&&d.every(f=>x(f,r)==="unsupported")}function E(t,a,r){const d=h[t.id];if(d===void 0)return'<span class="muted small">reading…</span>';if(d===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!d.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const f=Q(t.id,a.chainId,r.id),k=(d.networks??[]).find(fe=>fe.chainId===a.chainId);if(!f||!k||k.attributed===0)return'<span class="muted small">no traffic yet</span>';const S=Math.round(f.actual*100),T=Math.round(f.intended*100),K=f.diverged?r.local?"warn":"":"ok",Z=`${f.succeeded.toLocaleString()} of ${k.attributed.toLocaleString()} answered requests · routing intends ${T}%`+(f.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${n(Z)}">
        <span class="bar">
          <span class="fill${K?" "+K:""}" style="width:${S}%"></span>
          <span class="tick" style="left:${T}%"></span>
        </span>
        <span class="share-n${f.diverged?" warn":""}">${S}%</span>
        ${f.unconfigured?U("not in config","warn"):""}
      </span>
    `}function W(t){const a=h[t.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${n(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${n(V(a.since))})`:""}. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function V(t){const a=new Date(t);return Number.isNaN(a.getTime())?t:a.toLocaleString()}function D(t){const a=t.config;return`
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
        ${te(t)}
        ${X(t)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${n(t.id)}">Save settings</button>
        </div>
      </div>
    `}function te(t){const a=!t.config.MetricsOff;return`
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
    `}function X(t){var S;const a=n(t.id),r=t.config.TLS??null,d=(r==null?void 0:r.Enabled)??!1,f=(r==null?void 0:r.CertSource)||"internal",k=((S=t.tls)==null?void 0:S.suggestedHostname)??"";return`
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
        <input type="text" id="gw-${a}-tls-host" value="${n((r==null?void 0:r.Hostname)??k)}"
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
        <input type="text" inputmode="numeric" id="gw-${a}-tls-port" value="${(r==null?void 0:r.HTTPSPort)||443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${a}-tls-source">
          <option value="internal" ${f==="internal"?"selected":""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${f==="files"?"selected":""}>A certificate file on this machine</option>
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
    `}function Se(t){var S,T;const a=n(t.id),r=((S=t.config.TLS)==null?void 0:S.Enabled)??!1,d=_[t.id]??((T=t.tls)==null?void 0:T.verification)??null,f=B[t.id]??!1,k=F[t.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${r&&!f?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${f?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${k?`<p class="error small">${n(k)}</p>`:""}
      ${d?Be(d):""}
    `}function Be(t){const a=(t.assertions??[]).map(r=>`
          <li class="small">
            ${He(r.status)}
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
    `}function He(t){switch(t){case"pass":return U("pass","ok");case"fail":return U("fail","bad");case"unavailable":return U("unavailable","warn");default:return U("skipped","neutral")}}async function We(t){B[t]=!0,F[t]=null,J();try{_[t]=await cn(t)}catch(a){F[t]=`${he(a)}${Ee(a)}`}finally{B[t]=!1,J()}}function De(t){var f,k;const a=t.tls;if(!(a!=null&&a.enabled))return"";const r=[];a.fallback&&r.push(`<div class="banner banner-warn">${n(a.fallback)}</div>`),a.error?r.push(`<div class="banner banner-warn">HTTPS front: ${n(a.error)}</div>`):((f=a.status)==null?void 0:f.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${n(a.containerName??"")}</code>) is
         ${n(((k=a.status)==null?void 0:k.State)??"unknown")}, so nothing is answering on
         <code>${n(a.url??"")}</code> even if the gateway itself is up.</div>`);const d=_[t.id]??a.verification??null;return d&&(!d.ok||!d.subscriptionsOk)&&r.push(`<div class="banner ${d.ok?"banner-warn":"banner-bad"}">${n(d.summary)}
         <div class="small">Checked ${n(new Date(d.at).toLocaleString())} — open Settings for the full check.</div></div>`),d!=null&&d.expiryWarning&&r.push(`<div class="banner banner-warn">${n(d.expiryWarning)}</div>`),a.rootCaPath&&a.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${n(a.rootCaPath)}</code> (on ${n(t.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function ue(t){return{...t.config,Networks:(t.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(r=>({...r}))}))}}async function Te(t,a,r){N[t]=null;try{await hn(t,a)}catch(d){return N[t]=`${r?r+": ":""}${he(d)}`,J(),!1}return await v(),!0}async function dt(t,a){const r=a.dataset.gid??"";switch(t){case"refresh":await v();return;case"copy":a.dataset.copy&&await Ht(a,a.dataset.copy);return;case"reprobe":await H(r,!0);return;case"toggle-settings":M[r]=!M[r],J();return;case"save-settings":await ut(r);return;case"verify-tls":await We(r);return;case"gw-start":case"gw-stop":case"gw-restart":await ft(r,t.slice(3));return;case"gw-create":case"gw-recreate":await mt(r);return;case"gw-wipe":Rt(r);return;case"add-gateway":At();return;case"forget-gateway":await bt(r);return;case"dismiss-orphan":await gt(a.dataset.name??"");return;case"add-chain":yt(r);return;case"remove-chain":await kt(r,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":Xe(r,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await Tt(a.dataset.key??"");return;case"reset-devnet":await Et(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function ut(t){const a=G(t);if(!a)return;const r=ue(a),d=s.querySelector(`#gw-${CSS.escape(t)}-port`),f=s.querySelector(`#gw-${CSS.escape(t)}-bind`);if(d){const T=Number.parseInt(d.value.trim(),10);Number.isFinite(T)&&(r.Port=T)}f&&(r.BindAddr=f.value.trim());const k=s.querySelector(`#gw-${CSS.escape(t)}-metrics`);k&&(r.MetricsOff=!k.checked),r.TLS=pt(t,a);const S=a.status.State==="running";await Te(t,r,"Saving settings")&&(M[t]=!1,S&&(N[t]=null,ht(t,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),J())}function pt(t,a){var k,S,T,K,Z,fe,et;const r=Dt=>s.querySelector(`#gw-${CSS.escape(t)}-${Dt}`),d=r("tls");if(!d)return a.config.TLS??null;const f=Number.parseInt(((k=r("tls-port"))==null?void 0:k.value.trim())??"",10);return{Enabled:d.checked,Hostname:((S=r("tls-host"))==null?void 0:S.value.trim())??"",CertSource:((T=r("tls-source"))==null?void 0:T.value)??"internal",CertFile:((K=r("tls-cert"))==null?void 0:K.value.trim())??"",KeyFile:((Z=r("tls-key"))==null?void 0:Z.value.trim())??"",HTTPSPort:Number.isFinite(f)?f:443,BindAddr:((fe=a.config.TLS)==null?void 0:fe.BindAddr)??"",ImageRef:((et=a.config.TLS)==null?void 0:et.ImageRef)??""}}function ht(t,a){j[t]=[a]}async function ft(t,a){if(!b[t]){b[t]=a,N[t]=null,J();try{await fn(t,a)}catch(r){N[t]=`${a} failed: ${he(r)}${Ee(r)}`}b[t]=null,await v()}}async function mt(t){if(b[t])return;b[t]="create",N[t]=null,j[t]=["starting…"],J();let a;try{a=await mn(t)}catch(r){N[t]=`${he(r)}${Ee(r)}`,j[t]=[],b[t]=null,J();return}C==null||C(),C=Je(a.targetId,r=>{if(i)return;const d=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(j[t]=[...(j[t]??[]).filter(k=>k!=="starting…"),d],!!r.err||r.stepId===Bn&&!!r.done){C==null||C(),C=null,b[t]=null,r.err&&(N[t]="Provisioning failed — see the log below."),v();return}J()})}async function bt(t){const a=G(t);if(!(!a||!await Le({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await pn(t)}catch(d){N[t]=he(d),J();return}await v()}}async function gt(t){if(t){I[t]=null;try{await rn(t)}catch(a){I[t]=he(a),J();return}await v()}}function yt(t){const a=G(t);if(!a)return;const r=new Set((a.networks??[]).map(T=>T.chainId)),d=(o==null?void 0:o.presets)??[],f=d.filter(T=>!r.has(T.chainId)),k=d.filter(T=>r.has(T.chainId)),S=((o==null?void 0:o.targets)??[]).some(T=>T.id===a.placement.targetId&&T.hasDevnet);se(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${n(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${f.map(T=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${T.chainId}">
                <span>${n(T.name)}</span>
                <span class="muted small">chain ${T.chainId}${T.devnet?S?" · uses the devnet on "+n(a.placement.targetId):" · will create a devnet on "+n(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${k.length?`<p class="muted small">Already fronted: ${n(k.map(T=>T.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,T=>{if(T==="cancel"){Y();return}if(T==="custom"){vt(t);return}if(T.startsWith("preset:")){const K=Number.parseInt(T.slice(7),10),Z=d.find(fe=>fe.chainId===K);Y(),Z!=null&&Z.devnet?wt(t,K,S):Ye(t,K)}})}function vt(t){var a;se(`
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
      `,r=>{if(r==="cancel"){Y();return}if(r!=="add")return;const d=document.getElementById("custom-chain-id"),f=document.getElementById("custom-chain-err"),k=Number.parseInt((d==null?void 0:d.value.trim())??"",10);if(!Number.isFinite(k)||k<=0){f&&(f.className="error small"),f&&(f.textContent="A chain id is a positive whole number.");return}Y(),Ye(t,k)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function Ye(t,a){const r=G(t);if(!r)return;const d=ue(r),f=d.Networks??[];f.some(k=>k.ChainID===a)||(f.push({ChainID:a,Upstreams:[]}),d.Networks=f,await $t(t,d)&&(J(),Xe(t,a)))}async function $t(t,a){var k;const r={...a,Networks:(a.Networks??[]).filter(S=>S.Upstreams.length>0)};if(!await Te(t,r))return!1;const f=G(t);if(f)for(const S of a.Networks??[])S.Upstreams.length===0&&!(f.networks??[]).some(T=>T.chainId===S.ChainID)&&(f.config.Networks=[...f.config.Networks??[],{ChainID:S.ChainID,Upstreams:[]}],f.networks=[...f.networks??[],{chainId:S.ChainID,name:((k=((o==null?void 0:o.presets)??[]).find(T=>T.chainId===S.ChainID))==null?void 0:k.name)??`Chain ${S.ChainID}`,path:`/${f.config.ProjectID}/evm/${S.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function wt(t,a,r){const d=G(t);if(!d)return;if(!r){se(`
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
        `,()=>Y());return}const f=ue(d),k=f.Networks??[],S={ID:"devnet",Kind:"managed-devnet",TargetID:d.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},T=k.find(K=>K.ChainID===a);T?T.Upstreams.push(S):k.push({ChainID:a,Upstreams:[S]}),f.Networks=k,await Te(t,f,"Adding the devnet")}async function kt(t,a){const r=G(t);if(!r||!Number.isFinite(a))return;const d=ee(r,a);if(!await Le({title:`Remove ${(d==null?void 0:d.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(d==null?void 0:d.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const k=ue(r);k.Networks=(k.Networks??[]).filter(S=>S.ChainID!==a),await Te(t,k,"Removing the network")}function Ze(t){const a=t.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function Tt(t){const a=Ze(t);if(!a)return;const r=G(a.gid);if(!r)return;const d=ue(r),f=(d.Networks??[]).find(T=>T.ChainID===a.chainId);if(!f)return;const k=f.Upstreams.findIndex((T,K)=>(T.ID||`${a.chainId}-${K}`)===a.upstreamId);k<0||!await Le({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(f.Upstreams.splice(k,1),await Te(a.gid,d,"Removing the endpoint"))}function Xe(t,a){const r=G(t);if(!r||!Number.isFinite(a))return;const d=((o==null?void 0:o.sources)??[]).filter(T=>T.chainId===a),f=ee(r,a),k=new Set(((f==null?void 0:f.upstreams)??[]).filter(T=>T.kind!=="external").map(T=>`${T.kind}|${T.targetId??""}`)),S=d.filter(T=>!k.has(`${T.kind}|${T.targetId}`));se(`
        <h2>Add an endpoint for ${n((f==null?void 0:f.name)??`chain ${a}`)}</h2>
        ${S.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${S.map(T=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${n(T.kind)}:${n(T.targetId)}">
                       <span>${n(T.label)}</span>
                       <span class="muted small">${n(T.endpoint)}</span>
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
      `,T=>{if(T==="cancel"){Y();return}if(T==="discover"){xt(t,a);return}if(T==="manual"){Pt(t,a);return}if(T.startsWith("source:")){const[,K,Z]=T.split(":");Y(),Ct(t,a,K,Z)}})}async function Ct(t,a,r,d){const f=G(t);if(!f)return;const k=ue(f),S=k.Networks??[],T={ID:`${r==="managed-devnet"?"devnet":"node"}-${d}`,Kind:r,TargetID:d,Endpoint:"",Local:!0,RecentOnly:!1},K=S.find(Z=>Z.ChainID===a);K?K.Upstreams.push(T):S.push({ChainID:a,Upstreams:[T]}),k.Networks=S,await Te(t,k,"Adding the endpoint")}function St(t){const a=[...t].sort((f,k)=>(f.latencyMs??1e9)-(k.latencyMs??1e9)),r=a.slice(0,3),d=a.find(f=>f.url.startsWith("wss://")||f.url.startsWith("ws://"));return d&&!r.some(f=>f.url===d.url)&&(r.length===3&&r.pop(),r.push(d)),new Set(r.map(f=>f.url))}async function xt(t,a){se(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,S=>{S==="cancel"&&Y()});let r;try{r=await gn(a)}catch(S){const T=Ne();if(T){const K=document.createElement("p");K.className="error small",K.textContent=`Could not discover endpoints: ${he(S)}`,T.appendChild(K)}return}if(i)return;const d=(r.endpoints??[]).filter(S=>S.status==="live"||S.status==="unprobed"),f=(r.endpoints??[]).filter(S=>S.status==="rejected"),k=St(d);se(`
        <h2>Public endpoints for chain ${a}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${n(r.fetchError)}</div>`:""}</div>`:""}
        ${d.length?`<p class="muted small">${d.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${d.map(S=>{const T=k.has(S.url)?" checked":"";return`
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${n(S.url)}"${T}>
                       <span><code>${n(S.url)}</code></span>
                       <span class="muted small">${S.status==="live"?`answered in ${S.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </label>
                   </li>`}).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${f.length?`<details class="rpc-rejected">
                 <summary class="muted small">${f.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${f.map(S=>`<li class="muted small"><code>${n(S.url)}</code> — ${n(S.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          ${d.length?'<button class="btn" data-modal-action="add">Add selected</button>':""}
        </div>
      `,S=>{if(S==="cancel"){Y();return}if(S==="add"){const T=Ne(),K=T?Array.from(T.querySelectorAll('input[type="checkbox"]:checked')).map(Z=>Z.value):[];Y(),Qe(t,a,K);return}})}function Pt(t,a){var r;se(`
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
      `,d=>{if(d==="cancel"){Y();return}if(d!=="add")return;const f=document.getElementById("manual-endpoint"),k=document.getElementById("manual-recent"),S=document.getElementById("manual-err"),T=(f==null?void 0:f.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(T)){S&&(S.className="error small",S.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}Y(),Qe(t,a,[T],(k==null?void 0:k.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Qe(t,a,r,d=!1){if(!r.length)return;const f=G(t);if(!f)return;const k=ue(f),S=k.Networks??[];let T=S.find(Z=>Z.ChainID===a);T||(T={ChainID:a,Upstreams:[]},S.push(T));let K=1;for(const Z of T.Upstreams){const fe=/^public-\d+-(\d+)$/.exec(Z.ID??"");fe&&(K=Math.max(K,Number(fe[1])+1))}for(const Z of r)T.Upstreams.some(fe=>fe.Endpoint===Z)||T.Upstreams.push({ID:`public-${a}-${K++}`,Kind:"external",Endpoint:Z,Local:!1,RecentOnly:d});k.Networks=S,await Te(t,k,r.length===1?"Adding the endpoint":`Adding ${r.length} endpoints`)}async function Et(t,a){const r=Ze(t);if(!r||!a||!await Le({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;b[r.gid]="reset",N[r.gid]=null,J();let f;try{f=await an(a)}catch(k){N[r.gid]=`Reset failed: ${he(k)}${Ee(k)}`,b[r.gid]=null,J();return}b[r.gid]=null,It(a,f),await v()}function It(t,a){const r=[];r.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&r.push("A fresh chain was started from genesis.");const d=a.report.Cascaded??[],f=a.report.CascadeSkipped??[];se(`
        <h2>Devnet on ${n(t)} reset</h2>
        <ul class="plain-list">${r.map(k=>`<li>${n(k)}</li>`).join("")}</ul>
        ${d.length?`<p class="ok">Restarted in front of it: ${n(d.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${f.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(f.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>Y())}function Rt(t){const a=G(t);if(!a)return;se(`
        <h2>Wipe ${n(a.label)}</h2>
        <p class="error">This destroys ${n(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${n(t)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${n(t)}</button>
        </div>
      `,f=>{if(f==="cancel"||f==="close"){Y(),v();return}f==="confirm"&&Lt(t)});const r=document.getElementById("wipe-confirm-input"),d=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{d&&(d.disabled=r.value.trim()!==t)}),r==null||r.focus()}async function Lt(t){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let r;try{r=await bn(t)}catch(d){const f=Ne();if(f){const k=document.createElement("p");k.className="error small",k.textContent=`Wipe failed: ${he(d)}${Ee(d)}`,f.appendChild(k)}a&&(a.disabled=!1,a.textContent=`Wipe ${t}`);return}se(`
        <h2>${n(t)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${n(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{Y(),v()})}function Nt(t,a){return!a.some(r=>{var d;return((d=r.placement)==null?void 0:d.targetId)===t})}function At(){var k;const t=(o==null?void 0:o.targets)??[],a=(o==null?void 0:o.gateways)??[],r=t.filter(S=>Nt(S.id,a)),d=new Set(a.map(S=>S.id));if(t.length===0){se(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>Y());return}if(r.length===0){se(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>Y());return}const f=d.has("default")?"":"default";se(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${n(f)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${r.map(S=>`<option value="${n(S.id)}">${n(S.id)} (${n(S.mode)})</option>`).join("")}
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
      `,S=>{if(S==="cancel"){Y();return}S==="create"&&Bt()}),(k=document.getElementById("new-gw-id"))==null||k.focus()}async function Bt(){const t=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),d=document.getElementById("new-gw-err"),f=(t==null?void 0:t.value.trim())??"",k=(a==null?void 0:a.value)??"",S=Number.parseInt((r==null?void 0:r.value.trim())??"",10),T=K=>{d&&(d.className="error small",d.textContent=K)};if(!f){T("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!k){T("Pick the machine it runs on.");return}try{await on({id:f,placement:{targetId:k,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(S)?S:4e3,Networks:[]}})}catch(K){T(he(K));return}Y(),await v()}async function Ht(t,a){const r=await Ae(a),d=t.textContent;t.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{i||(t.textContent=d)},1500)}function he(t){return t instanceof Error?t.message:String(t)}function Ee(t){return t instanceof ke&&t.hint?` — ${t.hint}`:""}return()=>{i=!0,C==null||C(),Y()}}const Un="run",Mn={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},On={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Fn(s,i){let o=!1,e=null,h=null;const $={devnet:null},R={devnet:null},b={devnet:[]};let N=null;const j={devnet:!1};let M=null;const _={devnet:null},B={devnet:null};s.innerHTML=`
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
  `;const F=s.querySelector("#services-body");$e(s,(c,m)=>{ye(c,m)}),I();async function I(){try{const c=await Qt(i);if(o)return;e=c,h=null}catch(c){if(o)return;e=null,h=x(c)}u()}function C(c){return e==null?void 0:e.services.find(m=>m.id===c)}function u(){if(!o){if(h){F.innerHTML=`<p class="error">Could not read this machine's services: ${n(h)}</p>`;return}if(!e){F.innerHTML='<p class="muted">Loading…</p>';return}F.innerHTML=`
      ${v(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(A).join("")}
      </div>
    `}}function v(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${n(c.flavor)}${c.serverVersion?` ${n(c.serverVersion)}`:""} · reachable</p>`;const m=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${n(m)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${n(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function A(c){const m=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${n(c.label)}</h2>
          ${H(c)}
        </div>
        <p class="muted small">${n(Mn[c.id]??"")}</p>

        ${c.error?G(c):""}
        ${c.blocked?`<div class="banner banner-warn">${n(c.blocked)}</div>`:""}
        ${m.map(E=>`<div class="banner banner-warn">${n(E)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${n(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${n(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${ee(c)}

        ${Q(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(E=>ae(c,E)).join("")}
        </div>
        ${R[c.id]?`<p class="error small">${n(R[c.id])}</p>`:""}
        ${J(c)}

        ${de(c)}
      </div>
    `}function H(c){switch(c.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function G(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${n(c.error??"")}</div>
        ${c.hint?`<div class="small">${n(c.hint)}</div>`:""}
      </div>
    `}function ee(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const m=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${m}.</p>`}function Q(c){const m=c.endpoints??[];return m.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":m.map(E=>`
        <div class="endpoint-row">
          ${we("ok")}
          <span class="muted small">${n(E.label)}</span>
          <code class="endpoint-url">${n(E.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${n(E.url)}">Copy</button>
        </div>`).join("")}function ae(c,m){const E=On[m];if(!E)return"";const W=$[c.id],V=m==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:E.label;return`
      <button class="${E.className}" data-action="svc-${m}" data-svc="${n(c.id)}"
              title="${n(E.title)}" ${W?"disabled":""}>
        ${W===m?'<span class="spinner" aria-label="working"></span>':n(V)}
      </button>
    `}function J(c){const m=b[c.id]??[];return m.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${n(m.join(`
`))}</pre>
      </div>
    `}function de(c){const m=j[c.id],E=pe(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${n(c.id)}">
            ${m?"Close":"Edit"}
          </button>
        </div>
        ${m?ce():`<p class="small">${E}</p>`}
        ${_[c.id]?`<p class="error small">${n(_[c.id])}</p>`:""}
        ${B[c.id]?`<p class="muted small">${n(B[c.id])}</p>`:""}
      </div>
    `}function pe(c){const m=c.devnet;return m?`Chain ${m.ChainID} · a block every ${n(m.BlockTime)} · JSON-RPC on ${n(m.BindAddr)}:${m.HTTPPort} · WebSocket on ${n(m.BindAddr)}:${m.WSPort}`:"—"}function ce(c){return re()}function re(){const c=M;return c?`
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
    `:""}function oe(){j.devnet&&M&&(M.BlockTime=be("#dev-blocktime",M.BlockTime),M.HTTPPort=ge("#dev-http",M.HTTPPort),M.WSPort=ge("#dev-ws",M.WSPort),M.BindAddr=be("#dev-bind",M.BindAddr))}function be(c,m){const E=s.querySelector(c);return E?E.value.trim():m}function ge(c,m){const E=s.querySelector(c);if(!E)return m;const W=Number.parseInt(E.value.trim(),10);return Number.isFinite(W)?W:m}async function ye(c,m){const E=m.dataset.svc??"";switch(c){case"refresh":await I();return;case"copy":m.dataset.copy&&await y(m,m.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await g(E,c.slice(4));return;case"svc-create":case"svc-recreate":await l(E);return;case"svc-wipe":L(E);return;case"toggle-config":w(E);return;case"save-config":await P(E);return;default:return}}async function g(c,m){if(!$[c]){$[c]=m,R[c]=null,u();try{await en(i,c,m)}catch(E){R[c]=`${m} failed: ${x(E)}${O(E)}`}$[c]=null,await I()}}async function l(c){if(!$[c]){$[c]="create",R[c]=null,b[c]=["starting…"],u();try{await nn(i,c)}catch(m){R[c]=`${x(m)}${O(m)}`,b[c]=[],$[c]=null,u();return}N==null||N(),N=Je(i,m=>{if(o)return;const E=m.err?`${m.stepId}: ${m.err}`:m.line?`${m.stepId}: ${m.line}`:`${m.stepId}: done`;if(b[c]=[...(b[c]??[]).filter(V=>V!=="starting…"),E],!!m.err||m.stepId===Un&&!!m.done){N==null||N(),N=null,$[c]=null,m.err&&(R[c]="Provisioning failed — see the log below."),I();return}u()})}}function w(c){if(oe(),j[c]=!j[c],_[c]=null,B[c]=null,j[c]){const m=C(c);m!=null&&m.devnet&&(M={...m.devnet})}u()}async function P(c){var W;oe(),_[c]=null,B[c]=null;const m=M;if(!m)return;if(m.HTTPPort===m.WSPort){_[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",u();return}try{await sn(i,c,m)}catch(V){_[c]=x(V),u();return}const E=((W=C(c))==null?void 0:W.status.State)==="running";j[c]=!1,B[c]=E?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await I()}function L(c){const m=C(c);if(!m)return;const E=(m.restartsOnWipe??[]).map(D=>{var te;return((te=C(D))==null?void 0:te.label)??D});se(`
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
      `,D=>{if(D==="cancel"||D==="close"){Y(),I();return}D==="confirm"&&q(c)});const W=document.getElementById("wipe-confirm-input"),V=document.getElementById("wipe-confirm-btn");W==null||W.addEventListener("input",()=>{V&&(V.disabled=W.value.trim()!==c)}),W==null||W.focus()}async function q(c){const m=document.getElementById("wipe-confirm-btn");m&&(m.disabled=!0,m.textContent="Wiping…");let E;try{E=await tn(i,c)}catch(W){const V=Ne();if(V){const D=document.createElement("p");D.className="error small",D.textContent=`Wipe failed: ${x(W)}${O(W)}`,V.appendChild(D)}m&&(m.disabled=!1,m.textContent=`Wipe ${c}`);return}p(c,E)}function p(c,m){const E=C(c),W=X=>{var Se;return((Se=C(X))==null?void 0:Se.label)??X},V=[];V.push(m.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const X of m.report.VolumesRemoved??[])V.push(`Volume ${X} deleted.`);for(const X of m.report.VolumesAbsent??[])V.push(`Volume ${X} was already gone.`);m.report.Recreated&&V.push("Container re-created from your saved configuration.");const D=(m.report.Cascaded??[]).map(W),te=(m.report.CascadeSkipped??[]).map(W);se(`
        <h2>${n((E==null?void 0:E.label)??c)} wiped</h2>
        <ul class="plain-list">${V.map(X=>`<li>${n(X)}</li>`).join("")}</ul>
        ${D.length?`<p class="ok">Restarted in front of it: ${n(D.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${te.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${n(te.join(", "))}.</p>`:""}
        ${m.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${n(m.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,X=>{(X==="close"||X==="cancel")&&(Y(),I())})}async function y(c,m){const E=await Ae(m),W=c.textContent;c.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{o||(c.textContent=W)},1500)}function x(c){return c instanceof Error?c.message:String(c)}function O(c){return c instanceof ke&&c.hint?` — ${c.hint}`:""}return()=>{o=!0,N==null||N(),Y()}}const qn="local";function jn(s){let i=!1,o=!1,e="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${ie()}
  `;const $=s.querySelector("#targets-body");$e(s,(u,v)=>{M(u,v)}),R();async function R(){try{const[u,v,A]=await Promise.all([Pe(),xe(),Ot()]);if(i)return;e=A.os,N(u,v)}catch(u){if(i)return;$.innerHTML=`<p class="error">Failed to load machines: ${n(String(u))}</p>`}}function b(){h&&N(h.targets,h.catalog)}function N(u,v){h={targets:u,catalog:v};const A=e==="linux",H=[...u].sort((Q,ae)=>(Q.mode==="local"?-1:0)-(ae.mode==="local"?-1:0)),G=H.length?`<div class="card-grid">${H.map(Q=>Wn(Q,v,Q.mode!=="local"||A,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',ee=u.some(Q=>Q.mode==="local");$.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${G}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(A,ee)}
        ${o?_n():""}
      </section>
    `}function j(u,v){const A=`
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
      `;return v?`<div class="card-grid card-grid-wide">${A}</div>`:`<div class="card-grid card-grid-wide">${u?H+A:A+H}</div>`}async function M(u,v){var A;if(u==="add-local"){await _();return}if(u==="delete-target"){const H=v.dataset.id;if(!H||!await Le({title:"Remove machine",body:`Remove "${H}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await B(H);return}if(u==="toggle-ssh"){o=!o,C(),b(),o&&((A=s.querySelector("#ssh-host"))==null||A.focus());return}u==="add-ssh"&&await F()}async function _(){C();try{await tt({id:qn,mode:"local"}),await R()}catch(u){I(u)}}async function B(u){try{await Ft(u),await R()}catch(v){I(v)}}async function F(){const u=s.querySelector("#ssh-host"),v=s.querySelector("#ssh-user"),A=s.querySelector("#ssh-key"),H=s.querySelector("#ssh-port"),G=s.querySelector("#ssh-id");if(!u||!v||!A||!H||!G)return;const ee=u.value.trim(),Q=v.value.trim(),ae=A.value.trim(),J=H.value.trim(),de=G.value.trim();if(C(),!ee||!Q||!ae){I(new Error("host, user, and key path are required"));return}const pe=de||Kn(ee),ce={Host:ee,User:Q,KeyPath:ae};if(J){const oe=Number.parseInt(J,10);if(!Number.isFinite(oe)||oe<=0){I(new Error("port must be a positive number"));return}ce.Port=oe}const re=s.querySelector("#ssh-submit");re&&(re.disabled=!0,re.textContent="Connecting…");try{await tt({id:pe,mode:"ssh",ssh:ce}),o=!1,await R()}catch(oe){I(oe),re&&(re.disabled=!1,re.textContent="Add server")}}function I(u){let v=s.querySelector("#targets-error");v||($.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),v=s.querySelector("#targets-error")),v.textContent=String(u instanceof Error?u.message:u)}function C(){var u;(u=s.querySelector("#targets-error"))==null||u.remove()}return()=>{i=!0}}function Wn(s,i,o,e){const h=s.wire,$=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${n(s.ssh.User)}@${n(s.ssh.Host)}`:$,b=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(s.id)}">Devnet</a>`;let N,j;if(!h&&!o)N=`${U("can't run a node","warn")} ${U(e||"not Linux","neutral")}`,j=`
      ${b}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Preview setup wizard</a>
    `;else if(!h)N=U("not set up","neutral"),j=`
      <a class="btn" href="#/setup/${encodeURIComponent(s.id)}">Run setup wizard</a>
      ${b}
    `;else{const M=i.networks.find(B=>B.ChainID===h.ChainID),_=M?M.Name:`chain ${h.ChainID}`;N=`${U(_,"ok")} ${U(h.ExecID,"neutral")} ${U(h.BeaconID,"neutral")}${h.Archive?" "+U("archive","warn"):""}`,j=`
      <a class="btn" href="#/dash/${encodeURIComponent(s.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(s.id)}">Logs</a>
      ${b}
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
  `}function _n(){return`
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
  `}function Kn(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ze=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Me=8545,Oe=5052,Fe=30303,zn=[369,943,1],it={369:"default",943:"practise here first"};function Gn(s,i){let o=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${n(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${ie()}</div>`;const h=s.querySelector("#wizard-body"),$=s.querySelector("#wizard-footer");$e(s,(p,y)=>{ge(p,y)}),Ve(s,(p,y)=>{p==="exec-select"?e.execId=y:p==="beacon-select"&&(e.beaconId=y),b()}),s.addEventListener("change",p=>{const y=p.target;y instanceof HTMLInputElement&&(y.id==="data-dir-input"?(ye(),ae()):y.id==="checkpoint-toggle"?(e.checkpoint=y.checked,b()):y.id==="exec-snapshot-toggle"&&(e.execSnapshot=y.checked,b()))}),R();async function R(){try{const[p,y]=await Promise.all([xe(),Pe()]);if(o)return;e.catalog=p;const x=y.find(O=>O.id===i);x!=null&&x.wire&&(e.chainId=x.wire.ChainID,e.execId=x.wire.ExecID,e.beaconId=x.wire.BeaconID,e.archive=x.wire.Archive,x.wire.ExecHTTPPort&&(e.execHTTPPort=String(x.wire.ExecHTTPPort)),x.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(x.wire.BeaconHTTPPort)),x.wire.ExecP2PPort&&(e.execP2PPort=String(x.wire.ExecP2PPort)),x.wire.RPCBindAddr&&(e.rpcBindAddr=x.wire.RPCBindAddr)),b()}catch(p){if(o)return;e.loadError=String(p instanceof Error?p.message:p),b()}}function b(){if(e.loadError){h.innerHTML=`<p class="error">Failed to load: ${n(e.loadError)}</p>`;return}e.catalog&&(h.innerHTML=`
      ${q(e.step)}
      ${j()}
    `,N())}function N(){var y;const p=(y=e.catalog)==null?void 0:y.networks.find(x=>x.ChainID===e.chainId);$.innerHTML=p?ie(p.Name,p.LearnURL):ie()}function j(){switch(e.step){case"network":return M();case"clients":return _();case"mode":return re();case"review":return oe();case"run":return be()}}function M(){const p=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${zn.map(x=>{const O=p.networks.find(E=>E.ChainID===x);if(!O)return"";const c=e.chainId===x,m=it[x]?U(it[x],x===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${x}" type="button">
          <h3>${n(O.Name)} <span class="muted">(chain ${x})</span></h3>
          ${m}
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function _(){const p=e.catalog,y=p.networks.find(c=>c.ChainID===e.chainId);if(!y)return'<p class="error">Unknown network.</p>';(e.execId===null||!y.ExecClients.includes(e.execId))&&(e.execId=y.ExecClients[0]??null),(e.beaconId===null||!y.BeaconClients.includes(e.beaconId))&&(e.beaconId=y.BeaconClients[0]??null);const x=y.ExecClients.map(c=>de(c,p)),O=y.BeaconClients.map(c=>de(c,p));return`
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
          ${Ge("exec-select",x,e.execId)}
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
    `}function B(p){return p<=0?"—":p>=1?`~${p.toFixed(1)} TB`:`~${Math.round(p*1e3)} GB`}const F=1.1,I=.5,C="Valve reth snapshot",u="rough estimate";function v(p){return p.SnapshotSizeTB}function A(p){return p.SnapshotSizeTB*I}function H(p){return`<p class="muted small">${B(v(p))} is the measured size of Valve's reth snapshot for ${n(p.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function G(p){return{archive:v(p)*1e12*F,full:A(p)*1e12*F}}function ee(p,y){if(!p)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${n(y)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${n(y)}</code>: ${n(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==y)return"";const x=G(p),O=e.freeBytes>=x.archive,c=e.freeBytes>=x.full,m=`<p class="muted small">Free at <code>${n(y)}</code>: <strong>${Ce(e.freeBytes)}</strong> — archive ${O?"fits":"won't fit"} (${B(v(p))}, ${C}), full ${c?"fits":"won't fit"} (${B(A(p))}, ${u}).</p>`;let E="";return e.downgradeNote?E=`<p class="banner banner-warn">${n(e.downgradeNote)}</p>`:c||(E=`<p class="banner banner-warn">Neither full (${B(A(p))}, ${u}) nor archive (${B(v(p))}, ${C}) fits the free space here — choose a location with more room.</p>`),m+E}function Q(p,y){if(e.downgradeNote=null,!p||e.freeBytes===null)return;const x=G(p);e.archive&&e.freeBytes<x.archive&&e.freeBytes>=x.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${y} for archive (${B(v(p))}, ${C}) — switched to Full (${B(A(p))}, ${u}). Pick a location with more room to run archive.`)}async function ae(){var x;if(e.chainId===null)return;const p=(x=e.catalog)==null?void 0:x.networks.find(O=>O.ChainID===e.chainId),y=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,b();try{const{freeBytes:O}=await qt(e.targetId,y);if(o)return;e.freeBytes=O,e.probedPath=y,Q(p,y)}catch(O){if(o)return;e.freeBytes=null,e.probedPath=y,e.diskError=String(O instanceof Error?O.message:O)}e.diskProbing=!1,b()}function J(p){return p?/^https?:\/\/.+/i.test(p)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(p,y){const x=y.clients.find(O=>O.id===p);return{value:p,label:x?`${x.id} — ${pe(x.repo)}`:p}}function pe(p){const y=p.split("/");return y.length>=4?y[3]:p}function ce(p,y){const x=p?y.clients.find(c=>c.id===p):void 0;if(!x)return"";const O=x.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${n(x.repo)}" target="_blank" rel="noopener noreferrer">${n(O)}</a></p>`}function re(){var W,V,D;const p=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",y=(W=e.catalog)==null?void 0:W.networks.find(te=>te.ChainID===e.chainId),x=((D=(V=e.catalog)==null?void 0:V.clients.find(te=>te.id===e.execId))==null?void 0:D.snapshotSupported)??!1,O=y?`${B(A(y))} (${u})`:"Smaller",c=y?`${B(v(y))} (${C})`:"Much larger",m=y?` on ${n(y.Name)}`:"",E=y?e.checkpoint?y.SyncLabel:y.GenesisSyncLabel:"";return`
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
          ${y?`<p class="sync-estimate">⏱ Estimated initial sync${m}: <strong>${n(E)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${n((y==null?void 0:y.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${n((y==null?void 0:y.CheckpointURL)??"")}" value="${n(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${n(e.checkpointUrlError)}</p>`:""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`:'<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>'}
        </div>

        ${x?`
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
              <tr><th>Approx. disk footprint${m}</th><td class="yes">${O}</td><td class="limited">${c}</td></tr>
              <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
            </tbody>
          </table>
          ${y?H(y):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
        </details>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          <span><strong>Archive</strong> — full historical state · ${c}${y?"":" disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${O}${y?"":" disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${n(p)})</span>
            <input id="data-dir-input" type="text" placeholder="${n(p)}" value="${n(e.dataDir)}" />
          </label>
          ${ee(y,e.dataDir||p)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${n(p)}/jwt.hex" value="${n(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Me})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Me}" value="${n(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${n(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Oe})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Oe}" value="${n(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${n(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Fe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Fe}" value="${n(e.execP2PPort)}" />
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
    `}function oe(){const y=e.catalog.networks.find(X=>X.ChainID===e.chainId),x=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,O=e.jwtPath||`${x}/jwt.hex`,c=ze.map(X=>`<li>${n(X.title)}</li>`).join(""),m=P(e.execHTTPPort,Me),E=P(e.beaconHTTPPort,Oe),W=P(e.execP2PPort,Fe),V=m||E||W?`<tr><th>Non-default ports</th><td>${[m?`exec HTTP ${m}`:null,E?`beacon HTTP ${E}`:null,W?`exec p2p ${W}`:null].filter(X=>X!==null).map(n).join(", ")}</td></tr>`:"",{addr:D}=g(e.rpcBindAddr),te=D?`<tr><th>RPC bind address</th><td><code>${n(D)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${n(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${n((y==null?void 0:y.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${n(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${n(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${n(x)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${n(O)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${n(e.checkpointUrl||(y==null?void 0:y.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${V}
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
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function be(){const y=e.catalog.networks.find(D=>D.ChainID===e.chainId),x=y==null?void 0:y.LearnURL,O=new Set(e.events.filter(D=>D.done).map(D=>D.stepId)),c=new Set(e.events.filter(D=>D.err).map(D=>D.stepId)),m=new Map;for(const D of e.events){if(!D.line)continue;const te=m.get(D.stepId)??[];te.push(D.line),m.set(D.stepId,te)}const E=ze.map(D=>{var De;const te=O.has(D.id),X=c.has(D.id),Se=X?U("failed","bad"):te?U("done","ok"):U("pending","neutral"),Be=(m.get(D.id)??[]).slice(-5),He=(De=e.events.find(ue=>ue.stepId===D.id&&ue.err))==null?void 0:De.err,We=D.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${x?` <a href="${n(x)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${te?"step-done":""} ${X?"step-error":""}">
          <div class="step-head">${Se} <strong>${n(D.title)}</strong></div>
          ${We}
          ${Be.length?`<pre class="step-log">${Be.map(ue=>n(ue)).join(`
`)}</pre>`:""}
          ${He?`<p class="error small">${n(He)}</p>`:""}
        </li>
      `}).join(""),W=e.events.some(D=>D.err),V=ze.every(D=>O.has(D.id))||e.events.some(D=>D.stepId==="handshake"&&D.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${E}</ol>
        ${V&&!W?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${n(e.startError)}</p>`:""}
        ${W?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function ge(p,y){switch(p){case"pick-network":e.chainId=Number(y.dataset.chainId),e.execId=null,e.beaconId=null,b();break;case"goto-network":e.step="network",b();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",b();break;case"goto-mode":e.step="mode",b(),ae();break;case"goto-review":if(ye(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){b();break}e.step="review",b();break;case"start-setup":L();break}}function ye(){const p=s.querySelectorAll('input[name="mode"]');for(const D of Array.from(p))D.checked&&(e.archive=D.value==="archive");const y=s.querySelector("#data-dir-input"),x=s.querySelector("#jwt-path-input");y&&(e.dataDir=y.value.trim()),x&&(e.jwtPath=x.value.trim());const O=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),m=s.querySelector("#exec-p2p-port-input");O&&(e.execHTTPPort=O.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),m&&(e.execP2PPort=m.value.trim());const E=s.querySelector("#rpc-bind-addr-input");E&&(e.rpcBindAddr=E.value.trim());const W=s.querySelector("#checkpoint-url-input");W&&(e.checkpointUrl=W.value.trim());const V=s.querySelector("#snapshot-key-input");V&&(e.snapshotKey=V.value.trim()),e.execHTTPPortError=w(e.execHTTPPort).error??null,e.beaconHTTPPortError=w(e.beaconHTTPPort).error??null,e.execP2PPortError=w(e.execP2PPort).error??null,e.rpcBindAddrError=g(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?J(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function g(p){if(!p)return{};const y=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(p);return y?y.slice(1).every(x=>Number(x)<=255)?{addr:p}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(p)&&p.includes(":")?{addr:p}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const l=/^\d+$/;function w(p){if(!p)return{};if(!l.test(p))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const y=Number(p);return!Number.isInteger(y)||y<1||y>65535?{error:"Port must be between 1 and 65535."}:{port:y}}function P(p,y){const{port:x}=w(p);if(!(x===void 0||x===y))return x}async function L(){var m;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(m=e.streamStop)==null||m.call(e),e.streamStop=null,b();const p={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(p.DataDir=e.dataDir),e.jwtPath&&(p.JWTPath=e.jwtPath);const y=P(e.execHTTPPort,Me),x=P(e.beaconHTTPPort,Oe),O=P(e.execP2PPort,Fe);y!==void 0&&(p.ExecHTTPPort=y),x!==void 0&&(p.BeaconHTTPPort=x),O!==void 0&&(p.ExecP2PPort=O);const{addr:c}=g(e.rpcBindAddr);c!==void 0&&(p.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(p.CheckpointURL=e.checkpointUrl):p.NoCheckpoint=!0,e.execSnapshot&&(p.ExecSnapshot=!0,p.SnapshotKey=e.snapshotKey);try{await jt(e.targetId,p)}catch(E){if(!(E instanceof ke&&E.status===409)){e.starting=!1,e.startError=String(E instanceof Error?E.message:E),b();return}}e.starting=!1,e.step="run",b(),e.streamStop=Je(e.targetId,E=>{o||(e.events.push(E),e.step==="run"&&b())})}function q(p){const y=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],O=y.map(c=>c.id).indexOf(p);return`
      <ol class="wizard-progress">
        ${y.map((c,m)=>`<li class="${m===O?"current":m<O?"past":"future"}">${n(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var p;o=!0,(p=e.streamStop)==null||p.call(e)}}const Jn=document.querySelector("#app"),{contentEl:Vn,setActiveNav:Yn}=$n(Jn);let le=null;function Zn(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[o,e]=i;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"||o==="analytics"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function ve(s){const i=document.createElement("div");return Vn.replaceChildren(i),s(i)}function lt(){if(le){try{le()}catch{}le=null}const{screen:s,id:i}=Zn();switch(Yn(s),s){case"setup":if(!i){location.hash="#/targets";return}le=ve(o=>Gn(o,i));break;case"dash":if(!i){location.hash="#/targets";return}le=ve(o=>xn(o,i));break;case"logs":if(!i){location.hash="#/targets";return}le=ve(o=>Pn(o,i));break;case"security":if(!i){location.hash="#/targets";return}le=ve(o=>In(o,i));break;case"diag":if(!i){location.hash="#/targets";return}le=ve(o=>En(o,i));break;case"services":if(!i){location.hash="#/targets";return}le=ve(o=>Fn(o,i));break;case"analytics":if(!i){location.hash="#/rpc";return}le=ve(o=>Cn(o,i));break;case"rpc":le=ve(o=>Dn(o));break;case"settings":le=ve(o=>Ln(o));break;case"targets":default:le=ve(o=>jn(o));break}}window.addEventListener("hashchange",lt);lt();
