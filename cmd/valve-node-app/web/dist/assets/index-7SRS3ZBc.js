var At=Object.defineProperty;var Bt=(s,i,o)=>i in s?At(s,i,{enumerable:!0,configurable:!0,writable:!0,value:o}):s[i]=o;var De=(s,i,o)=>Bt(s,typeof i!="symbol"?i+"":i,o);(function(){const i=document.createElement("link").relList;if(i&&i.supports&&i.supports("modulepreload"))return;for(const h of document.querySelectorAll('link[rel="modulepreload"]'))e(h);new MutationObserver(h=>{for(const $ of h)if($.type==="childList")for(const R of $.addedNodes)R.tagName==="LINK"&&R.rel==="modulepreload"&&e(R)}).observe(document,{childList:!0,subtree:!0});function o(h){const $={};return h.integrity&&($.integrity=h.integrity),h.referrerPolicy&&($.referrerPolicy=h.referrerPolicy),h.crossOrigin==="use-credentials"?$.credentials="include":h.crossOrigin==="anonymous"?$.credentials="omit":$.credentials="same-origin",$}function e(h){if(h.ep)return;h.ep=!0;const $=o(h);fetch(h.href,$)}})();function Ht(){return K("/api/host")}function Pe(){return K("/api/catalog")}function Ee(){return K("/api/targets")}function et(s){return K("/api/targets",{method:"POST",headers:pe,body:JSON.stringify(s)})}function Dt(s){return K(`/api/targets/${encodeURIComponent(s)}`,{method:"DELETE"})}function Ut(s,i){return K(`/api/targets/${encodeURIComponent(s)}/disk?path=${encodeURIComponent(i)}`)}function Mt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/setup`,{method:"POST",headers:pe,body:JSON.stringify(i)})}function Ge(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/setup/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Ot(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/monitor/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function Ft(s,i=200){return K(`/api/targets/${encodeURIComponent(s)}/logs?n=${i}`)}function qt(s,i){const o=new EventSource(`/api/targets/${encodeURIComponent(s)}/logs/stream`);return o.onmessage=e=>{try{i(JSON.parse(e.data))}catch{}},()=>o.close()}function tt(s,i){const o=i===void 0?{}:{lines:i};return K(`/api/targets/${encodeURIComponent(s)}/explain`,{method:"POST",headers:pe,body:JSON.stringify(o)})}function jt(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/${o}`,{method:"POST"})}function Wt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/services/${i}/clear`,{method:"POST",headers:pe,body:JSON.stringify({Confirm:i})})}function _t(s){return K(`/api/targets/${encodeURIComponent(s)}/du`)}function Kt(s){return K(`/api/targets/${encodeURIComponent(s)}/endpoints`)}function zt(s){return K(`/api/targets/${encodeURIComponent(s)}/firewall`)}function Gt(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics`)}function Jt(s){return K(`/api/targets/${encodeURIComponent(s)}/diagnostics/latest`)}function Vt(s){return K(`/api/targets/${encodeURIComponent(s)}/containers`)}function Yt(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/${o}`,{method:"POST"})}async function Zt(s,i){const o=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/${i}/wipe`,{method:"POST",headers:pe,body:JSON.stringify({Confirm:i})}),e=await o.text();let h=null;try{h=e?JSON.parse(e):null}catch{}if(h&&typeof h=="object"&&"report"in h)return h;const $=h&&typeof h=="object"&&typeof h.error=="string"?h.error:o.statusText||`HTTP ${o.status}`;throw new Te(o.status,$)}function Xt(s,i){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/provision`,{method:"POST"})}async function Qt(s){const i=await fetch(`/api/targets/${encodeURIComponent(s)}/containers/devnet/reset`,{method:"POST",headers:pe}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new Te(i.status,h)}function en(s,i,o){return K(`/api/targets/${encodeURIComponent(s)}/containers/${i}/config`,{method:"PUT",headers:pe,body:JSON.stringify(o)})}function it(){return K("/api/gateways")}function tn(s){return K("/api/gateways",{method:"POST",headers:pe,body:JSON.stringify(s)})}function nn(s){return K(`/api/gateways/${encodeURIComponent(s)}/tls/verify`)}function an(s){return K(`/api/gateways/${encodeURIComponent(s)}/traffic`)}function sn(s){return K(`/api/gateways/${encodeURIComponent(s)}/analytics`)}function rn(s,i=!1){const o=i?"?refresh=1":"";return K(`/api/gateways/${encodeURIComponent(s)}/capabilities${o}`)}function on(s){return K(`/api/gateways/${encodeURIComponent(s)}`,{method:"DELETE"})}function cn(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/config`,{method:"PUT",headers:pe,body:JSON.stringify(i)})}function ln(s,i){return K(`/api/gateways/${encodeURIComponent(s)}/${i}`,{method:"POST"})}function dn(s){return K(`/api/gateways/${encodeURIComponent(s)}/provision`,{method:"POST"})}async function un(s){const i=await fetch(`/api/gateways/${encodeURIComponent(s)}/wipe`,{method:"POST",headers:pe,body:JSON.stringify({Confirm:s})}),o=await i.text();let e=null;try{e=o?JSON.parse(o):null}catch{}if(e&&typeof e=="object"&&"report"in e)return e;const h=e&&typeof e=="object"&&typeof e.error=="string"?e.error:i.statusText||`HTTP ${i.status}`;throw new Te(i.status,h)}function pn(s){return K(`/api/chainlist/${s}`)}function hn(){return K("/api/settings")}function fn(s){return K("/api/settings",{method:"PUT",headers:pe,body:JSON.stringify(s)})}class Te extends Error{constructor(o,e,h,$){super(e);De(this,"status");De(this,"hint");De(this,"code");this.name="ApiError",this.status=o,this.hint=h,this.code=$}}const pe={"Content-Type":"application/json"};async function K(s,i){const o=await fetch(s,i);if(!o.ok){let h=o.statusText||`HTTP ${o.status}`,$,R;try{const m=await o.json();m&&typeof m.error=="string"&&m.error&&(h=m.error),m&&typeof m.hint=="string"&&m.hint&&($=m.hint),m&&typeof m.code=="string"&&m.code&&(R=m.code)}catch{}throw new Te(o.status,h,$,R)}if(o.status===204)return;const e=await o.text();return e?JSON.parse(e):void 0}const nt="https://learn.valve.city/rpc";function t(s){return s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function oe(s,i){const o=s&&i&&i!==nt?` <span class="footer-sep">·</span> <a href="${t(i)}" target="_blank" rel="noopener noreferrer">${t(s)}</a>`:"";return`
    <footer class="footer">
      <a href="${t(nt)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${o}
    </footer>
  `}function mn(s){s.innerHTML=`
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
  `;const i=s.querySelector("#content"),o=Array.from(s.querySelectorAll("[data-nav]"));return{contentEl:i,setActiveNav:h=>{for(const $ of o)$.classList.toggle("active",$.dataset.nav===h)}}}function te(s){return Number.isFinite(s)?s.toLocaleString("en-US"):"—"}function bn(s){return Number.isFinite(s)?`${s.toFixed(1)}%`:"—"}function gn(s){if(!Number.isFinite(s)||s<0)return"—";if(s<60)return`~${Math.round(s)}s`;const i=Math.round(s/60),o=Math.floor(i/60),e=i%60;if(o===0)return`~${e}m`;if(o<48)return`~${o}h ${e}m`;const h=Math.floor(o/24),$=o%24;return`~${h}d ${$}h`}function U(s,i){return`<span class="badge badge-${i}">${t(s)}</span>`}function ke(s){return`<span class="dot dot-${s}"></span>`}const at=["B","KB","MB","GB","TB","PB"];function Ce(s){if(!Number.isFinite(s)||s<0)return"—";if(s===0)return"0 B";let i=s,o=0;for(;i>=1024&&o<at.length-1;)i/=1024,o++;const e=i<10?2:i<100?1:0;return`${i.toFixed(e)} ${at[o]}`}async function Ae(s){try{return await navigator.clipboard.writeText(s),!0}catch{return!1}}function $e(s,i){s.addEventListener("click",o=>{const e=o.target.closest("[data-action]");if(!e||!s.contains(e))return;const h=e.dataset.action;h&&i(h,e,o)})}function ze(s,i,o){const e=i.find($=>$.value===o),h=i.map($=>`
      <li class="dropdown-option${$.value===o?" selected":""}" role="option"
          aria-selected="${$.value===o}" data-value="${t($.value)}">
        ${t($.label)}
      </li>`).join("");return`
    <div class="dropdown" data-dropdown="${t(s)}">
      <button type="button" class="dropdown-trigger" aria-haspopup="listbox" aria-expanded="false">
        <span class="dropdown-value">${t(e?e.label:"Select…")}</span>
        <span class="dropdown-caret" aria-hidden="true">▾</span>
      </button>
      <ul class="dropdown-menu" role="listbox">${h}</ul>
    </div>
  `}function Re(s){s.querySelectorAll(".dropdown.open").forEach(i=>{var o;i.classList.remove("open"),(o=i.querySelector(".dropdown-trigger"))==null||o.setAttribute("aria-expanded","false")})}function Je(s,i){s.addEventListener("click",h=>{const $=h.target,R=$.closest(".dropdown-trigger");if(R&&s.contains(R)){const B=R.closest(".dropdown"),j=!!B&&!B.classList.contains("open");Re(s),B&&j&&(B.classList.add("open"),R.setAttribute("aria-expanded","true"));return}const m=$.closest(".dropdown-option");if(m&&s.contains(m)){const B=m.closest(".dropdown");Re(s),i((B==null?void 0:B.dataset.dropdown)??"",m.dataset.value??"");return}Re(s)});const o=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}const $=h.target;(!$.closest(".dropdown")||!s.contains($))&&Re(s)},e=h=>{if(!s.isConnected){document.removeEventListener("click",o),document.removeEventListener("keydown",e);return}h.key==="Escape"&&Re(s)};document.addEventListener("click",o),document.addEventListener("keydown",e)}const je="app-modal";let Fe=null;function ae(s,i){J();const o=document.createElement("div");o.className="modal-overlay",o.id=je,o.innerHTML=`<div class="modal">${s}</div>`,o.addEventListener("click",h=>{const $=h.target.closest("[data-modal-action]");$!=null&&$.dataset.modalAction?i($.dataset.modalAction):h.target===o&&i("cancel")});const e=h=>{h.key==="Escape"&&i("cancel")};document.addEventListener("keydown",e),Fe=e,document.body.appendChild(o)}function J(){var s;(s=document.getElementById(je))==null||s.remove(),Fe&&(document.removeEventListener("keydown",Fe),Fe=null)}function qe(){return document.querySelector(`#${je} .modal`)}function Ne(s){return new Promise(i=>{var h;let o=!1;const e=$=>{o||(o=!0,J(),i($))};ae(`
        <h2>${t(s.title)}</h2>
        <p>${t(s.body)}</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn${s.danger?" btn-danger":""}" data-modal-action="confirm">${t(s.confirmLabel)}</button>
        </div>
      `,$=>e($==="confirm")),(h=document.querySelector(`#${je} [data-modal-action="confirm"]`))==null||h.focus()})}const We=5e3,yn=60;function vn(s,i){let o=!1,e=null,h=null,$=null,R=null;const m=[];s.innerHTML=`<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${oe()}</div>`;const B=s.querySelector("#an-body");$e(s,(b,l)=>{var w;b==="toggle-endpoint"&&((w=l.closest(".an-endpoint"))==null||w.classList.toggle("expanded"))}),j();async function j(){try{e=((await it()).gateways??[]).find(l=>l.id===i)??null}catch(b){if(o)return;$=String(b instanceof Error?b.message:b),H();return}if(!o){if(!e){H();return}await M(),R=window.setInterval(()=>void M(),We)}}async function M(){try{const b=await sn(i);if(o)return;_(b),h=b,$=null}catch(b){if(o)return;$=String(b instanceof Error?b.message:b)}H()}function _(b){if(!b.enabled||b.error)return;const l=m[m.length-1];l&&l.since!==b.since&&(m.length=0);const w=new Map;for(const x of b.networks??[])w.set(x.chainId,x.received);m.push({t:Date.now(),since:b.since,received:w}),m.length>yn&&m.shift()}function H(){o||(B.innerHTML=F())}function F(){return $&&!h?`<h1>Analytics</h1><p class="error">${t($)}</p><p><a href="#/rpc">← Back to RPC</a></p>`:e?`
      ${P(e)}
      ${h?d(h):`<p class="muted">Reading the gateway's counters…</p>`}
    `:`
        <h1>Analytics</h1>
        <p class="error">No gateway called “${t(i)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `}function P(b){return`
      <div class="an-head">
        <div>
          <h1>Analytics: ${t(b.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${C()}</div>
      </div>
    `}function C(){if(!h)return"";if(!h.enabled)return"counters off";if(h.error)return"could not be read";const b=h.since?new Date(h.since):null;return b&&!Number.isNaN(b.getTime())?`totals since the gateway started, ${t(b.toLocaleString())}<br />re-read every ${We/1e3}s`:`re-read every ${We/1e3}s`}function d(b){return b.enabled?b.error?`
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${t(b.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `:k(b)+re(b):`
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `}function k(b){const l=b.networks??[];return`
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${l.length===0?'<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>':l.map(w=>A(w)).join("")}
      </section>
    `}function A(b){const l=b.methods??[],w=b.endpoints??[],x=b.received===0;return`
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${b.chainId}</span>
          <span class="band-name">${t(b.name)}</span>
          ${Z(b)}
        </div>
        <div class="an-stats">
          ${L("Received",te(b.received),"what clients asked this chain for")}
          ${L("Answered",te(b.answered),"returned by one of your endpoints")}
          ${L("From cache",te(b.unattributed),"answered by the gateway itself, without calling any endpoint")}
          ${L("Failed",te(b.failed),"asked for and never answered",b.failed>0?"bad":"")}
        </div>
        ${X(b.chainId)}
        ${x?'<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>':ce("Method",l.map(N=>({label:N.method,l:N})))+ce("Endpoint",w.map(N=>({label:N.upstream,l:N})))+z(b)}
      </div>
    `}function L(b,l,w,x=""){return`
      <div class="an-stat${x?" an-stat-"+x:""}" title="${t(w)}">
        <span class="an-stat-n">${t(l)}</span>
        <span class="an-stat-l">${t(b)}</span>
      </div>
    `}function Z(b){const l=Q(b.chainId);if(l===null)return'<span class="an-rate muted small">measuring rate…</span>';const w=Math.round((m[m.length-1].t-m[0].t)/1e3);return`<span class="an-rate" title="Measured from this page's own readings, ${w}s apart.">
      ${t(l.toFixed(l<10?2:0))} req/s <span class="muted">over the last ${w}s</span>
    </span>`}function Q(b){if(m.length<2)return null;const l=m[0],w=m[m.length-1],x=(w.t-l.t)/1e3;if(x<=0)return null;const N=(w.received.get(b)??0)-(l.received.get(b)??0);return N<0?null:N/x}function X(b){if(m.length<3)return"";const l=[];for(let v=1;v<m.length;v++){const S=m[v-1],O=m[v],c=(O.t-S.t)/1e3,f=(O.received.get(b)??0)-(S.received.get(b)??0);l.push(c>0&&f>=0?f/c:0)}const w=Math.max(...l);if(w<=0)return"";const x=240,N=28,q=l.length>1?x/(l.length-1):x,p=l.map((v,S)=>`${(S*q).toFixed(1)},${(N-v/w*N).toFixed(1)}`).join(" ");return`
      <div class="an-spark" title="Request rate since you opened this page. Peak ${w.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${x} ${N}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${p}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${t(w.toFixed(2))} req/s</span>
      </div>
    `}function z(b){const l=[];return b.cached.count>0&&l.push(`${t(te(b.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${b.cached.mean===null?"":`, in ${t(Le(b.cached.mean))} on average`}.`),b.failedLatency.count>0&&b.failedLatency.mean!==null&&l.push(`The ${t(te(b.failedLatency.count))} that failed took
         ${t(Le(b.failedLatency.mean))} on average to fail.`),l.length===0?"":`<p class="muted small">${l.join(" ")}</p>`}function ce(b,l){return l.length===0?"":`
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${t(b)}</th>
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
    `}function de(b,l){return`
      <tr>
        <td><code>${t(b)}</code></td>
        <td class="an-num">${te(l.count)}</td>
        <td class="an-num">${l.mean===null?'<span class="muted">—</span>':t(Le(l.mean))}</td>
        <td>${ue(l)}</td>
      </tr>
    `}function ue(b){const l=b.buckets??[];if(l.length===0||b.count===0)return'<span class="muted small">—</span>';let w=0;const x=[];for(const q of l){const p=q.count-w;w=q.count,x.push({label:se(q.le),n:Math.max(0,p)})}return x.reduce((q,p)=>q+p.n,0)===0?'<span class="muted small">—</span>':`
      <span class="an-dist" title="${t(x.filter(q=>q.n>0).map(q=>`${q.n} ${q.label}`).join(" · "))}">
        ${x.map((q,p)=>q.n===0?"":`<span class="an-band an-band-${Math.min(p,4)}" style="flex:${q.n}"></span>`).join("")}
      </span>
      <span class="muted small">${t(ie(x))}</span>
    `}function ie(b){for(let l=b.length-1;l>=0;l--)if(b[l].n>0)return`slowest ${b[l].label}`;return""}function se(b){if(b==="+Inf")return"30s or more";const l=Number(b);return Number.isFinite(l)?`under ${Le(l)}`:`under ${b}`}function re(b){const l=b.endpoints??[];return`
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
                     <tbody>${l.map(w=>he(w)).join("")}</tbody>
                   </table>
                 </div>
               </div>`}
      </section>
    `}function he(b){const l=b.errors??[],w=l.reduce((N,q)=>N+q.count,0),x=l.length>0;return`
      <tr class="an-endpoint${x?" expandable":""}" ${x?'data-action="toggle-endpoint"':""}>
        <td>
          <code>${t(b.upstream)}</code>
          ${b.chainId?`<span class="muted small">chain ${b.chainId}</span>`:""}
          ${b.configured?"":`<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`}
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${te(b.requests)}</td>
        <td class="an-num${w>0?" bad":""}">${w>0?te(w):'<span class="muted">0</span>'}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${b.headLag>0?te(b.headLag):'<span class="muted">0</span>'}</td>
        <td>${fe(b)}</td>
      </tr>
      ${x?me(b,l):""}
    `}function fe(b){const l=[];return b.scored?(l.push(b.position===0?'<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>':`<span class="muted small">position ${t(String(b.position))}</span>`),l.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${t(b.score.toFixed(3))}</span>`),b.primarySwitches>1&&l.push(`<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${te(b.primarySwitches)} switches</span>`),b.excludedSeconds>0&&l.push(`<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${t(Le(b.excludedSeconds))}</span>`),`<span class="an-selection">${l.join(" ")}</span>`):'<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>'}function me(b,l){return`
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${l.map(w=>`
                    <tr>
                      <td class="an-num">${te(w.count)}</td>
                      <td><code>${t(w.class)}</code></td>
                      <td>${w.severity?`<span class="badge badge-${w.severity==="critical"?"bad":"warn"}">${t(w.severity)}</span>`:""}</td>
                      <td class="muted small">${t(w.method||"")}</td>
                    </tr>`).join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${t(b.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `}return()=>{o=!0,R!==null&&window.clearInterval(R)}}function Le(s){return!Number.isFinite(s)||s<0?"—":s>0&&s<5e-4?"<1ms":s<1?`${Math.round(s*1e3)}ms`:s<60?`${s<10?s.toFixed(1):Math.round(s)}s`:`${Math.round(s/60)}m`}const $n=85,_e={exec:"Execution",beacon:"Beacon"};function wn(s,i){let o=!1,e=null,h=null,$=null,R=null,m=null,B=null,j=null,M=null;const _={exec:null,beacon:null};let H=null;s.innerHTML=`<h1>Dashboard: ${t(i)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${oe()}</div>`;const F=s.querySelector("#dash-body"),P=s.querySelector("#dash-footer");F.addEventListener("click",l=>{const w=l.target.closest("[data-action]");if(!w||!F.contains(w))return;const x=w.dataset.action;if(x==="svc-action"){const N=w.dataset.svc,q=w.dataset.kind;N&&q&&he(N,q)}else if(x==="open-clear"){const N=w.dataset.svc;N&&me(N)}else if(x==="copy"){const N=w.dataset.copy;N&&fe(w,N)}else x==="retry-du"?d():x==="retry-endpoints"&&k()}),C();async function C(){let l,w;try{const[N,q]=await Promise.all([Ee(),Pe()]);l=N.find(p=>p.id===i),w=q}catch(N){if(o)return;F.innerHTML=`<p class="error">Failed to load target: ${t(String(N))}</p>`;return}if(o)return;if(!l){F.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){F.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const x=w==null?void 0:w.networks.find(N=>N.ChainID===l.wire.ChainID);x&&(P.innerHTML=oe(x.Name,x.LearnURL)),F.innerHTML='<p class="muted">Connecting…</p>',e=Ot(i,N=>{o||(A(N),h=N,$=N,L())}),d(),k()}async function d(){B=null;try{m=await _t(i)}catch(l){m=null,B=String(l instanceof Error?l.message:l)}o||L()}async function k(){M=null;try{j=await Kt(i)}catch(l){j=null,M=String(l instanceof Error?l.message:l)}o||L()}function A(l){if(!h)return;const w=(new Date(l.at).getTime()-new Date(h.at).getTime())/1e3,x=l.execHead-h.execHead;if(w>0&&x>=0){const N=x/w;R=R===null?N:R*.7+N*.3}}function L(){if(!$)return;const l=$;F.innerHTML=`
      <p class="dash-status">${Z(l)}</p>
      <div class="card-grid">
        ${se(l)}
        ${X(l)}
        ${z(l)}
        ${ce(l)}
        ${de(l)}
        ${ue()}
      </div>
      <p class="muted small">Last updated ${t(new Date(l.at).toLocaleTimeString())}</p>
    `}function Z(l){return!l.execActive&&!l.beaconActive?U("Node not running","bad"):l.execSyncing||l.beaconDistance>0?U("Syncing","warn"):U("Running · synced","ok")}function Q(l){const x=l.refHead>0?l.refHead-l.execHead:null,N=x!==null&&x>0&&R&&R>0?gn(x/R):x!==null&&x<=0?"caught up":"—";return{lag:x,eta:N}}function X(l){const{lag:w,eta:x}=Q(l);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${l.execActive?l.execSyncing?U("syncing","warn"):l.execHead===0?U("no data","neutral"):U("synced","ok"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${te(l.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${w!==null?te(l.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${w!==null?te(Math.max(w,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${x}</dd></div>
        </dl>
      </div>
    `}function z(l){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${l.beaconActive?l.beaconSlot===0?U("no data","neutral"):l.beaconDistance===0?U("synced","ok"):U("syncing","warn"):U("stopped","bad")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${te(l.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${te(l.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function ce(l){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${te(l.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${te(l.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function de(l){const w=l.diskUsedPct>=$n,x=`
      <div class="meter"><div class="meter-fill ${w?"meter-warn":""}" style="width:${Math.min(l.diskUsedPct,100)}%"></div></div>
      <p>${bn(l.diskUsedPct)} used</p>
    `;if(B)return`
        <div class="card ${w?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="error small">${t(B)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return`
        <div class="card ${w?"card-warn":""}">
          <h3>Storage</h3>
          ${x}
          <p class="muted">Loading…</p>
        </div>
      `;const N=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,q=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:p,eta:v}=Q(l),S=p!==null&&p>0&&R!==null&&R>0;return`
      <div class="card ${w?"card-warn":""}">
        <h3>Storage</h3>
        ${x}
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${Ce(m.ExecBytes)} of ~${Ce(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${N}%"></div></div>
        ${S?`<p class="muted small">Estimated time remaining: ${t(v)}</p>`:""}
        <p class="muted small">Beacon — ${Ce(m.BeaconBytes)} of ~${Ce(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${q}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${Ce(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${t(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${t(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function ue(){if(M)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${t(M)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!j)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const l=j,w=l.ExecReachable&&!l.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",x=l.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${t(l.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${t(l.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ke(l.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${t(l.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(l.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ke(l.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${t(l.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(l.BeaconHTTP)}">Copy</button>
        </div>
        ${w}
        ${x}
      </div>
    `}function ie(l,w){const x=_e[l],N=_[l],q=(p,v,S)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${l}" data-kind="${p}" ${N!==null||S?"disabled":""}>${N===p?re():t(v)}</button>`;return`
      <div class="service-row">
        <span>${t(x)} ${w?U("active","ok"):U("down","bad")}</span>
        <div class="service-actions">
          ${q("start","Start",w)}
          ${q("stop","Stop",!w)}
          ${q("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${l}" ${N!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function se(l){return`
      <div class="card">
        <h3>Services</h3>
        ${ie("exec",l.execActive)}
        ${ie("beacon",l.beaconActive)}
        ${H?`<p class="error small">${t(H)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(i)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(i)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(i)}">Diagnostics →</a>
        </p>
      </div>
    `}function re(){return'<span class="spinner" aria-label="working"></span>'}async function he(l,w){if(_[l]===null){_[l]=w,H=null,L();try{await jt(i,l,w)}catch(x){H=`${_e[l]} ${w} failed: ${x instanceof Error?x.message:String(x)}`}_[l]=null,o||L()}}async function fe(l,w){const x=await Ae(w),N=l.textContent;l.textContent=x?"Copied!":"Copy failed",setTimeout(()=>{o||(l.textContent=N)},1500)}function me(l){const w=_e[l],x=m?Ce(l==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";ae(`
        <h2>Clear ${t(w)} data</h2>
        <p class="error">
          This stops the ${t(w.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${t(x)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${t(l)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,p=>{if(p==="cancel"){J();return}p==="confirm"&&b(l)});const N=document.getElementById("clear-confirm-input"),q=document.getElementById("clear-confirm-btn");N==null||N.addEventListener("input",()=>{q&&(q.disabled=N.value.trim()!==l)}),N==null||N.focus()}async function b(l){const w=document.getElementById("clear-confirm-btn");w&&(w.disabled=!0,w.textContent="Clearing…");try{await Wt(i,l),J(),d()}catch(x){const N=qe();if(N){const q=document.createElement("p");q.className="error small",q.textContent=`Clear failed: ${x instanceof Error?x.message:String(x)}`,N.appendChild(q)}w&&(w.disabled=!1,w.textContent="Clear and resync")}}return()=>{o=!0,e==null||e(),J()}}const st=500,rt="valve-node-app.explain-consent";function kn(s,i){let o=!1,e=null;const h=[];s.innerHTML=`
    <h1>Logs: ${t(i)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${oe()}</div>
  `;const $=s.querySelector("#logs-body"),R=s.querySelector("#logs-footer");$e(s,C=>{C==="explain"&&M()}),m();async function m(){let C,d;try{const[A,L]=await Promise.all([Ee(),Pe()]);C=A.find(Z=>Z.id===i),d=L}catch(A){if(o)return;$.innerHTML=`<p class="error">Failed to load target: ${t(String(A))}</p>`;return}if(o)return;if(!C){$.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!C.wire){$.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const k=d==null?void 0:d.networks.find(A=>A.ChainID===C.wire.ChainID);k&&(R.innerHTML=oe(k.Name,k.LearnURL));try{const A=await Ft(i,200);if(o)return;h.push(...A)}catch(A){if(o)return;$.innerHTML=`<p class="error">Failed to load logs: ${t(String(A))}</p>`;return}B(),e=qt(i,A=>{o||(h.push(A),h.length>st&&h.splice(0,h.length-st),B())})}function B(){const C=h.filter(k=>k.severity==="error"||k.severity==="critical");$.innerHTML=`
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
    `;const d=$.querySelector(".log-lines");d&&(d.scrollTop=d.scrollHeight)}function j(C){const d=C.severity||"info",k=C.learnUrl?` <a href="${t(C.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${t(d)}">
        <span class="log-time">${t(new Date(C.at).toLocaleTimeString())}</span>
        <span class="log-unit">${t(C.unit)}</span>
        <span class="log-sev">${t(d)}</span>
        <span class="log-text">${t(C.line)}</span>
        ${C.explain?`<div class="log-explain">${t(C.explain)}${k}</div>`:""}
      </div>
    `}async function M(){const C=h.filter(k=>k.severity==="error"||k.severity==="critical").map(k=>k.line).slice(-40);if(!(localStorage.getItem(rt)==="1")){_(C);return}await H(C)}function _(C){const d=C.length?`<pre class="explain-excerpt">${C.map(k=>t(k)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';F(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${d}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,k=>{k==="proceed"?(localStorage.setItem(rt,"1"),P(),H(C)):P()})}async function H(C){F('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const d=C.length?await tt(i,C):await tt(i);if(o)return;F(`
        <h2>Explanation</h2>
        <div class="explain-text">${t(d.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${d.sentExcerpt.map(k=>t(k)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,k=>{k==="close"&&P()})}catch(d){if(o)return;if(d instanceof Te&&d.status===409){F(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,k=>{k==="close"&&P()});return}F(`
        <h2>Explain failed</h2>
        <p class="error">${t(d instanceof Error?d.message:String(d))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,k=>{k==="close"&&P()})}}function F(C,d){P();const k=document.createElement("div");k.className="modal-overlay",k.id="explain-modal",k.innerHTML=`<div class="modal">${C}</div>`,k.addEventListener("click",A=>{const L=A.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&d(L.dataset.modalAction),A.target===k&&d("cancel")}),document.body.appendChild(k)}function P(){var C;(C=document.getElementById("explain-modal"))==null||C.remove()}return()=>{o=!0,e==null||e(),P()}}function Tn(s,i){let o=!1,e=null,h=null,$=!1,R=!1;s.innerHTML=`<h1>Network diagnostics: ${t(i)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${oe()}</div>`;const m=s.querySelector("#diag-body"),B=s.querySelector("#diag-footer");$e(s,(d,k)=>{var A;if(d==="run")M();else if(d==="toggle")(A=k.closest(".check-item"))==null||A.classList.toggle("expanded");else if(d==="copy"){const L=k.dataset.copy;L&&C(k,L)}}),j();async function j(){let d,k;try{const[L,Z]=await Promise.all([Ee(),Pe()]);d=L.find(Q=>Q.id===i),k=Z}catch(L){if(o)return;m.innerHTML=`<p class="error">Failed to load target: ${t(String(L))}</p>`;return}if(o)return;if(!d){m.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!d.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const A=k==null?void 0:k.networks.find(L=>L.ChainID===d.wire.ChainID);A&&(B.innerHTML=oe(A.Name,A.LearnURL));try{e=await Jt(i),R=!0}catch(L){h=String(L instanceof Error?L.message:L)}o||_()}async function M(){$=!0,h=null,_();try{e=await Gt(i),R=!0}catch(d){h=String(d instanceof Error?d.message:d)}$=!1,o||_()}function _(){m.innerHTML=`
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
      ${h?`<p class="error">${t(h)}</p>`:""}
      ${H()}
    `}function H(){if(!R&&!h)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const d=new Date(e.at).toLocaleString(),k=e.failedId?`<p><strong>Failed at: ${t(F(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${t(d)} — trigger: ${t(e.trigger)}</p>
      ${k}
      <ul class="check-list">${e.items.map(P).join("")}</ul>
    `}function F(d){var k;return((k=e==null?void 0:e.items.find(A=>A.ID===d))==null?void 0:k.Title)??d}function P(d){const k=d.Status==="pass"?"ok":d.Status==="fail"?"bad":d.Status==="warn"?"warn":"neutral",A=d.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${A?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${U(A?"failed here":d.Status,k)}
          <strong>${t(d.Title)}</strong>
          <span class="muted small check-detail-inline">${t(d.Detail)}</span>
        </button>
        <div class="check-body">
          <details${A?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${t(d.Why)}</p>
          </details>
          ${d.Fix?`
                <details${A?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${t(d.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${t(d.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function C(d,k){const A=await Ae(k),L=d.textContent;d.textContent=A?"Copied!":"Copy failed",setTimeout(()=>{o||(d.textContent=L)},1500)}return()=>{o=!0}}function Cn(s,i){let o=!1,e=[],h=null,$=!1,R=!1;s.innerHTML=`<h1>Security: ${t(i)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${oe()}</div>`;const m=s.querySelector("#sec-body"),B=s.querySelector("#sec-footer");$e(s,(P,C)=>{var d;if(P==="rerun")M();else if(P==="toggle")(d=C.closest(".check-item"))==null||d.classList.toggle("expanded");else if(P==="copy"){const k=C.dataset.copy;k&&F(C,k)}}),j();async function j(){let P,C;try{const[k,A]=await Promise.all([Ee(),Pe()]);P=k.find(L=>L.id===i),C=A}catch(k){if(o)return;m.innerHTML=`<p class="error">Failed to load target: ${t(String(k))}</p>`;return}if(o)return;if(!P){m.innerHTML=`<p class="error">Target "${t(i)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!P.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(i)}">Run the setup wizard →</a></p>`;return}const d=C==null?void 0:C.networks.find(k=>k.ChainID===P.wire.ChainID);d&&(B.innerHTML=oe(d.Name,d.LearnURL)),await M()}async function M(){$=!0,h=null,_();try{e=await zt(i),R=!0}catch(P){h=String(P instanceof Error?P.message:P)}$=!1,o||_()}function _(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(i)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${$?"disabled":""}>${$?"Re-running…":"Re-run checks"}</button>
      </div>
      ${h?`<p class="error">${t(h)}</p>`:""}
      ${!R&&$?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map(H).join("")}</ul>`:R?'<p class="muted">No checks returned.</p>':""}
    `}function H(P){const C=P.Status==="pass"?"ok":P.Status==="fail"?"bad":P.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${U(P.Status,C)}
          <strong>${t(P.Title)}</strong>
          <span class="muted small check-detail-inline">${t(P.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${t(P.Why)}</p>
          </details>
          ${P.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${t(P.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${t(P.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function F(P,C){const d=await Ae(C),k=P.textContent;P.textContent=d?"Copied!":"Copy failed",setTimeout(()=>{o||(P.textContent=k)},1500)}return()=>{o=!0}}const Sn=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function xn(s){let i=!1,o=!1,e=!1,h=null,$=!1,R=null,m=null;s.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${oe()}`;const B=s.querySelector("#settings-body");$e(s,H=>{if(H==="save"&&_(),H==="clear-key"){if(!R)return;o=!0;const F=s.querySelector("#ai-key");F&&(F.value=""),M(R)}}),Je(s,(H,F)=>{H!=="ai-provider"||!R||(m=F,$=!1,M(R))}),j();async function j(){try{const H=await hn();if(i)return;R=H,M(H)}catch(H){if(i)return;B.innerHTML=`<p class="error">Failed to load settings: ${t(String(H))}</p>`}}function M(H){var C;const F=m??H.aiProvider;B.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${ze("ai-provider",Sn.map(d=>({value:d.value,label:d.label})),F)}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${H.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${H.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${t(H.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${h?`<p class="error">${t(h)}</p>`:""}
        ${$?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const P=s.querySelector("#ai-key");P==null||P.addEventListener("input",()=>{o=!0,$=!1}),(C=s.querySelector("#ref-rpc-base"))==null||C.addEventListener("input",()=>{$=!1})}async function _(){const H=s.querySelector("#ai-key"),F=s.querySelector("#ref-rpc-base");if(!H||!F||!R)return;const P={aiProvider:m??R.aiProvider,refRpcBase:F.value.trim()};o&&(P.aiKey=H.value),e=!0,h=null,$=!1,M(R);try{const C=await fn(P);if(i)return;R=C,o=!1,e=!1,$=!0,M(C)}catch(C){if(i)return;e=!1,h=String(C instanceof Error?C.message:C),M(R)}}return()=>{i=!0}}const Pn=["http","ws","archive","trace"],En={http:"HTTP",ws:"WS",archive:"ARCHIVE",trace:"TRACE"},In="run",Rn={start:{label:"Start",title:"Start the existing gateway container",className:"btn"},stop:{label:"Stop",title:"Stop the gateway. Its configuration is kept.",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",className:"btn btn-ghost"},create:{label:"Create gateway",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy the gateway container and rebuild it",className:"btn btn-danger"}};function Ln(s){let i=!1,o=null,e=null;const h={},$={},R={},m={},B={},j={},M={},_={},H={},F={};let P=null;s.innerHTML=`
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
    ${oe()}
  `;const C=s.querySelector("#rpc-body");$e(s,(n,a)=>{xe(n,a)}),Je(s,()=>{}),d();async function d(){try{const n=await it();if(i)return;o=n,e=null}catch(n){if(i)return;o=null,e=ge(n)}z();for(const n of(o==null?void 0:o.gateways)??[])k(n.id),A(n.id,!1)}async function k(n){try{const a=await an(n);if(i)return;h[n]=a}catch{if(i)return;h[n]=null}z()}async function A(n,a){R[n]=a,a&&z();try{const r=await rn(n,a);if(i)return;$[n]=r}catch{if(i)return;$[n]=null}R[n]=!1,z()}function L(n){return((o==null?void 0:o.gateways)??[]).find(a=>a.id===n)}function Z(n,a){return(n.networks??[]).find(r=>r.chainId===a)}function Q(n,a,r){var g;const u=(((g=h[n])==null?void 0:g.networks)??[]).find(y=>y.chainId===a);return((u==null?void 0:u.upstreams)??[]).find(y=>y.upstream===r)}function X(n,a,r){var u;return(((u=$[n])==null?void 0:u.endpoints)??[]).find(g=>g.chainId===a&&g.upstream===r)}function z(){if(i)return;if(e){C.innerHTML=`<p class="error">Could not read the gateways: ${t(e)}</p>`;return}if(!o){C.innerHTML='<p class="muted">Loading…</p>';return}const n=o.gateways??[];C.innerHTML=`
      ${n.map(de).join("")}
      ${n.length===0?ce():""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${n.length?" btn-ghost":""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `}function ce(){return((o==null?void 0:o.targets)??[]).length===0?`
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
    `}function de(n){return`
      <section class="rpc-gateway">
        ${ue(n)}
        ${n.error?re(n):""}
        ${n.blocked?`<div class="banner banner-warn">${t(n.blocked)}</div>`:""}
        ${(n.warnings??[]).map(a=>`<div class="banner banner-warn">${t(a)}</div>`).join("")}
        ${He(n)}
        ${B[n.id]?`<p class="error small">${t(B[n.id])}</p>`:""}
        ${fe(n)}
        ${M[n.id]?W(n):""}
        ${me(n)}
      </section>
    `}function ue(n){var r;const a=n.status.State==="running";return`
      <div class="rpc-bar${a?"":" rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${se(n)}
            <strong>${t(n.label)}</strong>
            ${ie(n)}
            <span class="muted small">on ${t(n.placement.targetId)} · ${t(n.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(n.actions??[]).map(u=>he(n,u)).join("")}
            <a class="btn btn-ghost" href="#/analytics/${encodeURIComponent(n.id)}"
               title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what.">Analytics</a>
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${t(n.id)}">
              ${M[n.id]?"Close":"Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${t(n.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${a?`<code class="endpoint-url">${t(n.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${t(n.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${t(((r=(n.networks??[])[0])==null?void 0:r.path)??"/main/evm/&lt;chainId&gt;")}</code></span>`:`<span class="muted small">Not serving — it will answer on <code>${t(n.baseUrl)}</code> once it is running.</span>`}
        </div>
      </div>
    `}function ie(n){switch(n.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function se(n){return n.status.State==="running"?ke("ok"):n.status.State==="unknown"?ke("bad"):ke("neutral")}function re(n){return`
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${t(n.error??"")}</div>
        ${n.hint?`<div class="small">${t(n.hint)}</div>`:""}
      </div>
    `}function he(n,a){const r=Rn[a];if(!r)return"";const u=m[n.id];return`
      <button class="${r.className}" data-action="gw-${a}" data-gid="${t(n.id)}"
              title="${t(r.title)}" ${u?"disabled":""}>
        ${u===a?'<span class="spinner" aria-label="working"></span>':t(r.label)}
      </button>
    `}function fe(n){const a=j[n.id]??[];return a.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning on ${t(n.placement.targetId)}</p>
        <pre class="step-log">${t(a.join(`
`))}</pre>
      </div>
    `}function me(n){const a=n.networks??[];return a.length===0?`
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${t(n.id)}">Add a network</button>
          </div>
        </div>
      `:`
      <div class="card rpc-surface">
        ${b(n)}
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
              ${a.map(r=>l(n,r)+x(n,r)).join("")}
            </tbody>
          </table>
        </div>
        ${f(n)}
      </div>
    `}function b(n){const a=$[n.id];return`
      <div class="surface-head">
        <span class="muted small">${a!=null&&a.at?`probed ${t(E(a.at))}`:"not probed yet"}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${t(n.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${R[n.id]?"disabled":""}>
          ${R[n.id]?'<span class="spinner" aria-label="probing"></span>':"Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${t(n.id)}">+ Network</button>
      </div>
    `}function l(n,a){return`
      <tr class="band${!a.serviceable?" band-bad":""}">
        <td colspan="6">
          <div class="band-inner">
            <span class="band-id">${a.chainId}</span>
            <span class="band-name">${t(a.name)}</span>
            <code class="band-path">${t(a.path)}</code>
            ${a.url?`<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${t(a.url)}"
                           title="Copy ${t(a.url)}">Copy URL</button>`:""}
            <span class="band-right">
              ${w(n,a)}
              <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                      data-gid="${t(n.id)}" data-chain="${a.chainId}">+ Endpoint</button>
              <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                      data-gid="${t(n.id)}" data-chain="${a.chainId}">Remove</button>
            </span>
          </div>
          ${(a.warnings??[]).map(u=>`<div class="band-warn">${t(u)}</div>`).join("")}
        </td>
      </tr>
    `}function w(n,a){if(!a.serviceable)return U("no usable endpoint","bad");const r=a.upstreams??[],u=r.map(y=>X(n.id,a.chainId,y.id)).filter(y=>!!y&&!y.unprobeable);return u.length>0&&u.every(y=>p(y,"ws")==="unsupported")?U("subscriptions unavailable","bad"):r.map(y=>Q(n.id,a.chainId,y.id)).some((y,I)=>{var T;return y&&y.diverged&&(((T=r[I])==null?void 0:T.local)??!1)})?U("your endpoint is under-used","warn"):U(`${r.length} endpoint${r.length===1?"":"s"}`,"ok")}function x(n,a){const r=a.upstreams??[];return r.length===0?`
        <tr class="ep"><td colspan="6" class="muted small">
          No endpoint yet, so there is nowhere for calls on this path to go.
        </td></tr>
      `:r.map(u=>N(n,a,u)).join("")}function N(n,a,r){const u=`${n.id}|${a.chainId}|${r.id}`,g=r.actions??[];return`
      <tr class="ep${r.problem?" ep-bad":""}">
        <td class="col-endpoint">
          <div class="ep-what">
            ${r.problem?ke("bad"):ke("ok")}
            <span class="ep-label">${t(r.label)}</span>
          </div>
          <code class="ep-url">${t(r.endpoint||"—")}</code>
          ${r.problem?`<div class="error small">${t(r.problem)}</div>`:""}
        </td>
        <td>${r.local?"Yours":"Public"}</td>
        <td>${q(r)}</td>
        <td>${v(n,a,r)}</td>
        <td class="col-share">${c(n,a,r)}</td>
        <td class="col-act">
          ${g.includes("reset")?`<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${t(u)}"
                         data-target="${t(r.targetId??"")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${m[n.id]?"disabled":""}>
                   ${m[n.id]==="reset"?'<span class="spinner" aria-label="working"></span>':"Reset"}
                 </button>`:""}
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${t(u)}">Remove</button>
        </td>
      </tr>
    `}function q(n){return n.problem?U("unusable","bad"):n.recentOnly?U("recent blocks","warn"):n.local?U("serving","ok"):U("fallback","neutral")}function p(n,a){var r;if(n)return a==="http"?n.unprobeable?"inconclusive":n.reachable?"supported":"unsupported":(r=(n.capabilities??[]).find(u=>u.key===a))==null?void 0:r.status}function v(n,a,r){const u=X(n.id,a.chainId,r.id);return u?u.unprobeable?`<span class="caps-none" title="${t(u.unprobeable)}">not probeable from here</span>`:`<span class="caps">${Pn.map(g=>S(n,a,u,g)).join("")}</span>`:`<span class="muted small">${$[n.id]===void 0?"probing…":"—"}</span>`}function S(n,a,r,u){const g=(r.capabilities??[]).find(ne=>ne.key===u),y=p(r,u)??"inconclusive",I=En[u]??u.toUpperCase();let T="cap";y==="unsupported"?T=O(n,a,u)?"cap missing":"cap off":y==="inconclusive"?T="cap unknown":y==="inconsistent"&&(T="cap mixed");const V=g!=null&&g.detail?`${g.label}: ${g.detail}`:u==="http"&&r.reachDetail?`Answers JSON-RPC over HTTP: ${r.reachDetail}`:`${I}: no verdict`;return`<span class="${T}" title="${t(V)}">${t(I)}</span>`}function O(n,a,r){const u=(a.upstreams??[]).map(g=>X(n.id,a.chainId,g.id)).filter(g=>!!g&&!g.unprobeable);return u.length>0&&u.every(g=>p(g,r)==="unsupported")}function c(n,a,r){const u=h[n.id];if(u===void 0)return'<span class="muted small">reading…</span>';if(u===null)return'<span class="muted small" title="The counters could not be read.">—</span>';if(!u.enabled)return`<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;const g=Q(n.id,a.chainId,r.id),y=(u.networks??[]).find(we=>we.chainId===a.chainId);if(!g||!y||y.attributed===0)return'<span class="muted small">no traffic yet</span>';const I=Math.round(g.actual*100),T=Math.round(g.intended*100),V=g.diverged?r.local?"warn":"":"ok",ne=`${g.succeeded.toLocaleString()} of ${y.attributed.toLocaleString()} answered requests · routing intends ${T}%`+(g.unconfigured?" · this endpoint is no longer in the saved configuration":"");return`
      <span class="share" title="${t(ne)}">
        <span class="bar">
          <span class="fill${V?" "+V:""}" style="width:${I}%"></span>
          <span class="tick" style="left:${T}%"></span>
        </span>
        <span class="share-n${g.diverged?" warn":""}">${I}%</span>
        ${g.unconfigured?U("not in config","warn"):""}
      </span>
    `}function f(n){const a=h[n.id];return a?a.enabled?a.error?`<p class="muted small">The request counters could not be read: ${t(a.error)}</p>`:`<p class="muted small">
      Share is measured from the gateway's own counters since it started${a.since?` (${t(E(a.since))})`:""}. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`:`<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`:""}function E(n){const a=new Date(n);return Number.isNaN(a.getTime())?n:a.toLocaleString()}function W(n){const a=n.config;return`
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${t(n.id)}-port" value="${a.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${t(n.id)}-bind" value="${t(a.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${t(a.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${G(n)}
        ${D(n)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${t(n.id)}">Save settings</button>
        </div>
      </div>
    `}function G(n){const a=!n.config.MetricsOff;return`
      <label class="check">
        <input type="checkbox" id="gw-${t(n.id)}-metrics" ${a?"checked":""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `}function D(n){var I;const a=t(n.id),r=n.config.TLS??null,u=(r==null?void 0:r.Enabled)??!1,g=(r==null?void 0:r.CertSource)||"internal",y=((I=n.tls)==null?void 0:I.suggestedHostname)??"";return`
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${a}-tls" ${u?"checked":""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${a}-tls-host" value="${t((r==null?void 0:r.Hostname)??y)}"
               placeholder="${t(y||"gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${y?`<p class="muted small">
               The default is <code>${t(y)}</code>. That whole domain's wildcard resolves to
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
        <input type="text" id="gw-${a}-tls-cert" value="${t((r==null?void 0:r.CertFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${a}-tls-key" value="${t((r==null?void 0:r.KeyFile)??"")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${ee(n)}
    `}function ee(n){var I,T;const a=t(n.id),r=((I=n.config.TLS)==null?void 0:I.Enabled)??!1,u=_[n.id]??((T=n.tls)==null?void 0:T.verification)??null,g=H[n.id]??!1,y=F[n.id]??null;return`
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${a}" ${r&&!g?"":"disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${g?'<span class="spinner" aria-label="verifying"></span> Verifying…':"Verify HTTPS now"}
        </button>
        ${r?"":'<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>'}
      </div>
      ${y?`<p class="error small">${t(y)}</p>`:""}
      ${u?Y(u):""}
    `}function Y(n){const a=(n.assertions??[]).map(r=>`
          <li class="small">
            ${Se(r.status)}
            <strong>${t(r.title)}</strong>
            <div class="muted">${t(r.detail)}</div>
          </li>`).join("");return`
      <div class="banner ${n.ok?n.subscriptionsOk?"banner-ok":"banner-warn":"banner-bad"}">
        ${t(n.summary)}
      </div>
      <ul class="verify-list">${a}</ul>
      <p class="muted small">
        Checked ${t(new Date(n.at).toLocaleString())} against <code>${t(n.address)}</code>
        ${n.notAfter?`· certificate valid until <code>${t(new Date(n.notAfter).toLocaleString())}</code> (${t(n.expiresIn??"")})`:""}
      </p>
      ${n.expiryWarning?`<div class="banner banner-warn">${t(n.expiryWarning)}</div>`:""}
    `}function Se(n){switch(n){case"pass":return U("pass","ok");case"fail":return U("fail","bad");case"unavailable":return U("unavailable","warn");default:return U("skipped","neutral")}}async function Be(n){H[n]=!0,F[n]=null,z();try{_[n]=await nn(n)}catch(a){F[n]=`${ge(a)}${Ie(a)}`}finally{H[n]=!1,z()}}function He(n){var g,y;const a=n.tls;if(!(a!=null&&a.enabled))return"";const r=[];a.fallback&&r.push(`<div class="banner banner-warn">${t(a.fallback)}</div>`),a.error?r.push(`<div class="banner banner-warn">HTTPS front: ${t(a.error)}</div>`):((g=a.status)==null?void 0:g.State)!=="running"&&r.push(`<div class="banner banner-warn">The HTTPS front (<code>${t(a.containerName??"")}</code>) is
         ${t(((y=a.status)==null?void 0:y.State)??"unknown")}, so nothing is answering on
         <code>${t(a.url??"")}</code> even if the gateway itself is up.</div>`);const u=_[n.id]??a.verification??null;return u&&(!u.ok||!u.subscriptionsOk)&&r.push(`<div class="banner ${u.ok?"banner-warn":"banner-bad"}">${t(u.summary)}
         <div class="small">Checked ${t(new Date(u.at).toLocaleString())} — open Settings for the full check.</div></div>`),u!=null&&u.expiryWarning&&r.push(`<div class="banner banner-warn">${t(u.expiryWarning)}</div>`),a.rootCaPath&&a.effectiveCertSource==="internal"&&r.push(`<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${t(a.rootCaPath)}</code> (on ${t(n.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`),r.join("")}function ve(n){return{...n.config,Networks:(n.config.Networks??[]).map(a=>({ChainID:a.ChainID,Upstreams:a.Upstreams.map(r=>({...r}))}))}}async function be(n,a,r){B[n]=null;try{await cn(n,a)}catch(u){return B[n]=`${r?r+": ":""}${ge(u)}`,z(),!1}return await d(),!0}async function xe(n,a){const r=a.dataset.gid??"";switch(n){case"refresh":await d();return;case"copy":a.dataset.copy&&await Lt(a,a.dataset.copy);return;case"reprobe":await A(r,!0);return;case"toggle-settings":M[r]=!M[r],z();return;case"save-settings":await lt(r);return;case"verify-tls":await Be(r);return;case"gw-start":case"gw-stop":case"gw-restart":await pt(r,n.slice(3));return;case"gw-create":case"gw-recreate":await ht(r);return;case"gw-wipe":xt(r);return;case"add-gateway":It();return;case"forget-gateway":await ft(r);return;case"add-chain":mt(r);return;case"remove-chain":await vt(r,Number.parseInt(a.dataset.chain??"",10));return;case"add-endpoint":Ze(r,Number.parseInt(a.dataset.chain??"",10));return;case"remove-endpoint":await $t(a.dataset.key??"");return;case"reset-devnet":await Ct(a.dataset.key??"",a.dataset.target??"");return;default:return}}async function lt(n){const a=L(n);if(!a)return;const r=ve(a),u=s.querySelector(`#gw-${CSS.escape(n)}-port`),g=s.querySelector(`#gw-${CSS.escape(n)}-bind`);if(u){const T=Number.parseInt(u.value.trim(),10);Number.isFinite(T)&&(r.Port=T)}g&&(r.BindAddr=g.value.trim());const y=s.querySelector(`#gw-${CSS.escape(n)}-metrics`);y&&(r.MetricsOff=!y.checked),r.TLS=dt(n,a);const I=a.status.State==="running";await be(n,r,"Saving settings")&&(M[n]=!1,I&&(B[n]=null,ut(n,"Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.")),z())}function dt(n,a){var y,I,T,V,ne,we,Qe;const r=Nt=>s.querySelector(`#gw-${CSS.escape(n)}-${Nt}`),u=r("tls");if(!u)return a.config.TLS??null;const g=Number.parseInt(((y=r("tls-port"))==null?void 0:y.value.trim())??"",10);return{Enabled:u.checked,Hostname:((I=r("tls-host"))==null?void 0:I.value.trim())??"",CertSource:((T=r("tls-source"))==null?void 0:T.value)??"internal",CertFile:((V=r("tls-cert"))==null?void 0:V.value.trim())??"",KeyFile:((ne=r("tls-key"))==null?void 0:ne.value.trim())??"",HTTPSPort:Number.isFinite(g)?g:443,BindAddr:((we=a.config.TLS)==null?void 0:we.BindAddr)??"",ImageRef:((Qe=a.config.TLS)==null?void 0:Qe.ImageRef)??""}}function ut(n,a){j[n]=[a]}async function pt(n,a){if(!m[n]){m[n]=a,B[n]=null,z();try{await ln(n,a)}catch(r){B[n]=`${a} failed: ${ge(r)}${Ie(r)}`}m[n]=null,await d()}}async function ht(n){if(m[n])return;m[n]="create",B[n]=null,j[n]=["starting…"],z();let a;try{a=await dn(n)}catch(r){B[n]=`${ge(r)}${Ie(r)}`,j[n]=[],m[n]=null,z();return}P==null||P(),P=Ge(a.targetId,r=>{if(i)return;const u=r.err?`${r.stepId}: ${r.err}`:r.line?`${r.stepId}: ${r.line}`:`${r.stepId}: done`;if(j[n]=[...(j[n]??[]).filter(y=>y!=="starting…"),u],!!r.err||r.stepId===In&&!!r.done){P==null||P(),P=null,m[n]=null,r.err&&(B[n]="Provisioning failed — see the log below."),d();return}z()})}async function ft(n){const a=L(n);if(!(!a||!await Ne({title:`Forget ${a.label}`,body:`valve-node-app will forget this gateway's configuration. Its container "${a.containerName}" on ${a.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,confirmLabel:"Forget it",danger:!0}))){try{await on(n)}catch(u){B[n]=ge(u),z();return}await d()}}function mt(n){const a=L(n);if(!a)return;const r=new Set((a.networks??[]).map(T=>T.chainId)),u=(o==null?void 0:o.presets)??[],g=u.filter(T=>!r.has(T.chainId)),y=u.filter(T=>r.has(T.chainId)),I=((o==null?void 0:o.targets)??[]).some(T=>T.id===a.placement.targetId&&T.hasDevnet);ae(`
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${t(a.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${g.map(T=>`
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${T.chainId}">
                <span>${t(T.name)}</span>
                <span class="muted small">chain ${T.chainId}${T.devnet?I?" · uses the devnet on "+t(a.placement.targetId):" · will create a devnet on "+t(a.placement.targetId):""}</span>
              </button>
            </li>`).join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${y.length?`<p class="muted small">Already fronted: ${t(y.map(T=>T.name).join(", "))}.</p>`:""}
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,T=>{if(T==="cancel"){J();return}if(T==="custom"){bt(n);return}if(T.startsWith("preset:")){const V=Number.parseInt(T.slice(7),10),ne=u.find(we=>we.chainId===V);J(),ne!=null&&ne.devnet?yt(n,V,I):Ve(n,V)}})}function bt(n){var a;ae(`
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
      `,r=>{if(r==="cancel"){J();return}if(r!=="add")return;const u=document.getElementById("custom-chain-id"),g=document.getElementById("custom-chain-err"),y=Number.parseInt((u==null?void 0:u.value.trim())??"",10);if(!Number.isFinite(y)||y<=0){g&&(g.className="error small"),g&&(g.textContent="A chain id is a positive whole number.");return}J(),Ve(n,y)}),(a=document.getElementById("custom-chain-id"))==null||a.focus()}async function Ve(n,a){const r=L(n);if(!r)return;const u=ve(r),g=u.Networks??[];g.some(y=>y.ChainID===a)||(g.push({ChainID:a,Upstreams:[]}),u.Networks=g,await gt(n,u)&&(z(),Ze(n,a)))}async function gt(n,a){var y;const r={...a,Networks:(a.Networks??[]).filter(I=>I.Upstreams.length>0)};if(!await be(n,r))return!1;const g=L(n);if(g)for(const I of a.Networks??[])I.Upstreams.length===0&&!(g.networks??[]).some(T=>T.chainId===I.ChainID)&&(g.config.Networks=[...g.config.Networks??[],{ChainID:I.ChainID,Upstreams:[]}],g.networks=[...g.networks??[],{chainId:I.ChainID,name:((y=((o==null?void 0:o.presets)??[]).find(T=>T.chainId===I.ChainID))==null?void 0:y.name)??`Chain ${I.ChainID}`,path:`/${g.config.ProjectID}/evm/${I.ChainID}`,upstreams:[],serviceable:!1,warnings:["This network has no endpoint yet, so it is not saved on the gateway until you add one."]}]);return!0}async function yt(n,a,r){const u=L(n);if(!u)return;if(!r){ae(`
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${t(u.placement.targetId)}</code>, so adding chain ${a} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(u.placement.targetId)}" data-modal-action="go">Create a devnet on ${t(u.placement.targetId)}</a>
          </div>
        `,()=>J());return}const g=ve(u),y=g.Networks??[],I={ID:"devnet",Kind:"managed-devnet",TargetID:u.placement.targetId,Endpoint:"",Local:!0,RecentOnly:!1},T=y.find(V=>V.ChainID===a);T?T.Upstreams.push(I):y.push({ChainID:a,Upstreams:[I]}),g.Networks=y,await be(n,g,"Adding the devnet")}async function vt(n,a){const r=L(n);if(!r||!Number.isFinite(a))return;const u=Z(r,a);if(!await Ne({title:`Remove ${(u==null?void 0:u.name)??`chain ${a}`}`,body:`This gateway will stop serving ${(u==null?void 0:u.path)??`chain ${a}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,confirmLabel:"Remove network",danger:!0}))return;const y=ve(r);y.Networks=(y.Networks??[]).filter(I=>I.ChainID!==a),await be(n,y,"Removing the network")}function Ye(n){const a=n.split("|");return a.length!==3?null:{gid:a[0],chainId:Number.parseInt(a[1],10),upstreamId:a[2]}}async function $t(n){const a=Ye(n);if(!a)return;const r=L(a.gid);if(!r)return;const u=ve(r),g=(u.Networks??[]).find(T=>T.ChainID===a.chainId);if(!g)return;const y=g.Upstreams.findIndex((T,V)=>(T.ID||`${a.chainId}-${V}`)===a.upstreamId);y<0||!await Ne({title:"Remove this endpoint",body:"The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",confirmLabel:"Remove",danger:!0})||(g.Upstreams.splice(y,1),await be(a.gid,u,"Removing the endpoint"))}function Ze(n,a){const r=L(n);if(!r||!Number.isFinite(a))return;const u=((o==null?void 0:o.sources)??[]).filter(T=>T.chainId===a),g=Z(r,a),y=new Set(((g==null?void 0:g.upstreams)??[]).filter(T=>T.kind!=="external").map(T=>`${T.kind}|${T.targetId??""}`)),I=u.filter(T=>!y.has(`${T.kind}|${T.targetId}`));ae(`
        <h2>Add an endpoint for ${t((g==null?void 0:g.name)??`chain ${a}`)}</h2>
        ${I.length?`<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${I.map(T=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${t(T.kind)}:${t(T.targetId)}">
                       <span>${t(T.label)}</span>
                       <span class="muted small">${t(T.endpoint)}</span>
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
      `,T=>{if(T==="cancel"){J();return}if(T==="discover"){kt(n,a);return}if(T==="manual"){Tt(n,a);return}if(T.startsWith("source:")){const[,V,ne]=T.split(":");J(),wt(n,a,V,ne)}})}async function wt(n,a,r,u){const g=L(n);if(!g)return;const y=ve(g),I=y.Networks??[],T={ID:`${r==="managed-devnet"?"devnet":"node"}-${u}`,Kind:r,TargetID:u,Endpoint:"",Local:!0,RecentOnly:!1},V=I.find(ne=>ne.ChainID===a);V?V.Upstreams.push(T):I.push({ChainID:a,Upstreams:[T]}),y.Networks=I,await be(n,y,"Adding the endpoint")}async function kt(n,a){ae(`
        <h2>Public endpoints for chain ${a}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,y=>{y==="cancel"&&J()});let r;try{r=await pn(a)}catch(y){const I=qe();if(I){const T=document.createElement("p");T.className="error small",T.textContent=`Could not discover endpoints: ${ge(y)}`,I.appendChild(T)}return}if(i)return;const u=(r.endpoints??[]).filter(y=>y.status==="live"||y.status==="unprobed"),g=(r.endpoints??[]).filter(y=>y.status==="rejected");ae(`
        <h2>Public endpoints for chain ${a}</h2>
        ${r.source==="vendored"?`<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${r.fetchError?`<div class="small">${t(r.fetchError)}</div>`:""}</div>`:""}
        ${u.length?`<p class="muted small">${u.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${u.map(y=>`
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(y.url)}">
                       <span><code>${t(y.url)}</code></span>
                       <span class="muted small">${y.status==="live"?`answered in ${y.latencyMs??0} ms`:"not probed (WebSocket)"}</span>
                     </button>
                   </li>`).join("")}
               </ul>`:`<p class="error small">Nothing in the feed answered for chain ${a} right now.</p>`}
        ${g.length?`<details class="rpc-rejected">
                 <summary class="muted small">${g.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${g.map(y=>`<li class="muted small"><code>${t(y.url)}</code> — ${t(y.reason??"rejected")}</li>`).join("")}
                 </ul>
               </details>`:""}
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,y=>{if(y==="cancel"){J();return}y.startsWith("add:")&&(J(),Xe(n,a,decodeURIComponent(y.slice(4))))})}function Tt(n,a){var r;ae(`
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
      `,u=>{if(u==="cancel"){J();return}if(u!=="add")return;const g=document.getElementById("manual-endpoint"),y=document.getElementById("manual-recent"),I=document.getElementById("manual-err"),T=(g==null?void 0:g.value.trim())??"";if(!/^(https?|wss?):\/\//i.test(T)){I&&(I.className="error small",I.textContent="It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");return}J(),Xe(n,a,T,(y==null?void 0:y.checked)??!1)}),(r=document.getElementById("manual-endpoint"))==null||r.focus()}async function Xe(n,a,r,u=!1){const g=L(n);if(!g)return;const y=ve(g),I=y.Networks??[],T=I.find(we=>we.ChainID===a),V=((T==null?void 0:T.Upstreams.length)??0)+1,ne={ID:`public-${a}-${V}`,Kind:"external",Endpoint:r,Local:!1,RecentOnly:u};T?T.Upstreams.push(ne):I.push({ChainID:a,Upstreams:[ne]}),y.Networks=I,await be(n,y,"Adding the endpoint")}async function Ct(n,a){const r=Ye(n);if(!r||!a||!await Ne({title:"Reset this devnet",body:`The chain on ${a} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,confirmLabel:"Reset the chain"}))return;m[r.gid]="reset",B[r.gid]=null,z();let g;try{g=await Qt(a)}catch(y){B[r.gid]=`Reset failed: ${ge(y)}${Ie(y)}`,m[r.gid]=null,z();return}m[r.gid]=null,St(a,g),await d()}function St(n,a){const r=[];r.push(a.report.ContainerRemoved?"The old chain was removed.":"There was no devnet container to remove."),a.report.Recreated&&r.push("A fresh chain was started from genesis.");const u=a.report.Cascaded??[],g=a.report.CascadeSkipped??[];ae(`
        <h2>Devnet on ${t(n)} reset</h2>
        <ul class="plain-list">${r.map(y=>`<li>${t(y)}</li>`).join("")}</ul>
        ${u.length?`<p class="ok">Restarted in front of it: ${t(u.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`:'<p class="muted small">Nothing needed restarting in front of it.</p>'}
        ${g.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(g.join(", "))}.</p>`:""}
        ${a.error?`<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(a.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>J())}function xt(n){const a=L(n);if(!a)return;ae(`
        <h2>Wipe ${t(a.label)}</h2>
        <p class="error">This destroys ${t(a.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${t(n)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(n)}</button>
        </div>
      `,g=>{if(g==="cancel"||g==="close"){J(),d();return}g==="confirm"&&Pt(n)});const r=document.getElementById("wipe-confirm-input"),u=document.getElementById("wipe-confirm-btn");r==null||r.addEventListener("input",()=>{u&&(u.disabled=r.value.trim()!==n)}),r==null||r.focus()}async function Pt(n){const a=document.getElementById("wipe-confirm-btn");a&&(a.disabled=!0,a.textContent="Wiping…");let r;try{r=await un(n)}catch(u){const g=qe();if(g){const y=document.createElement("p");y.className="error small",y.textContent=`Wipe failed: ${ge(u)}${Ie(u)}`,g.appendChild(y)}a&&(a.disabled=!1,a.textContent=`Wipe ${n}`);return}ae(`
        <h2>${t(n)} wiped</h2>
        <ul class="plain-list">
          <li>${r.report.ContainerRemoved?"Container removed.":"There was no container to remove."}</li>
          ${r.report.Recreated?"<li>Container re-created from your saved configuration.</li>":""}
        </ul>
        ${r.error?`<p class="error small">${t(r.error)}</p>`:""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,()=>{J(),d()})}function Et(n,a){return!a.some(r=>{var u;return((u=r.placement)==null?void 0:u.targetId)===n})}function It(){var y;const n=(o==null?void 0:o.targets)??[],a=(o==null?void 0:o.gateways)??[],r=n.filter(I=>Et(I.id,a)),u=new Set(a.map(I=>I.id));if(n.length===0){ae(`
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,()=>J());return}if(r.length===0){ae(`
          <h2>Every machine already has a gateway</h2>
          <p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
          <div class="modal-actions">
            <button class="btn" data-modal-action="cancel">Close</button>
          </div>
        `,()=>J());return}const g=u.has("default")?"":"default";ae(`
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${t(g)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${r.map(I=>`<option value="${t(I.id)}">${t(I.id)} (${t(I.mode)})</option>`).join("")}
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
      `,I=>{if(I==="cancel"){J();return}I==="create"&&Rt()}),(y=document.getElementById("new-gw-id"))==null||y.focus()}async function Rt(){const n=document.getElementById("new-gw-id"),a=document.getElementById("new-gw-target"),r=document.getElementById("new-gw-port"),u=document.getElementById("new-gw-err"),g=(n==null?void 0:n.value.trim())??"",y=(a==null?void 0:a.value)??"",I=Number.parseInt((r==null?void 0:r.value.trim())??"",10),T=V=>{u&&(u.className="error small",u.textContent=V)};if(!g){T("Give it a name — it becomes the container's name, which is how it is found again.");return}if(!y){T("Pick the machine it runs on.");return}try{await tn({id:g,placement:{targetId:y,backend:"docker"},config:{ProjectID:"main",BindAddr:"127.0.0.1",Port:Number.isFinite(I)?I:4e3,Networks:[]}})}catch(V){T(ge(V));return}J(),await d()}async function Lt(n,a){const r=await Ae(a),u=n.textContent;n.textContent=r?"Copied!":"Copy failed",setTimeout(()=>{i||(n.textContent=u)},1500)}function ge(n){return n instanceof Error?n.message:String(n)}function Ie(n){return n instanceof Te&&n.hint?` — ${n.hint}`:""}return()=>{i=!0,P==null||P(),J()}}const Nn="run",An={devnet:"A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container."},Bn={start:{label:"Start",title:"Start the existing container",className:"btn"},stop:{label:"Stop",title:"Stop the container; it keeps its data",className:"btn btn-ghost"},restart:{label:"Restart",title:"Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",className:"btn btn-ghost"},create:{label:"Create",title:"Create the container from the configuration below",className:"btn"},recreate:{label:"Re-create (apply config)",title:"Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",className:"btn btn-ghost"},wipe:{label:"Wipe…",title:"Destroy this service's data and rebuild it",className:"btn btn-danger"}};function Hn(s,i){let o=!1,e=null,h=null;const $={devnet:null},R={devnet:null},m={devnet:[]};let B=null;const j={devnet:!1};let M=null;const _={devnet:null},H={devnet:null};s.innerHTML=`
    <div class="page-head">
      <h1>Services: ${t(i)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${oe()}
  `;const F=s.querySelector("#services-body");$e(s,(c,f)=>{me(c,f)}),P();async function P(){try{const c=await Vt(i);if(o)return;e=c,h=null}catch(c){if(o)return;e=null,h=S(c)}d()}function C(c){return e==null?void 0:e.services.find(f=>f.id===c)}function d(){if(!o){if(h){F.innerHTML=`<p class="error">Could not read this machine's services: ${t(h)}</p>`;return}if(!e){F.innerHTML='<p class="muted">Loading…</p>';return}F.innerHTML=`
      ${k(e.docker)}
      <div class="card-grid card-grid-wide">
        ${e.services.map(A).join("")}
      </div>
    `}}function k(c){if(c.present&&c.reachable&&!c.hint)return`<p class="muted small">Docker: ${t(c.flavor)}${c.serverVersion?` ${t(c.serverVersion)}`:""} · reachable</p>`;const f=c.present?"Docker is installed, but no engine answered":"No docker engine on this machine";return`
      <div class="banner banner-bad">
        <strong>${t(f)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${c.detail?`<div class="small">${t(c.detail)}</div>`:""}
        ${c.hint?`<div class="small">${t(c.hint)}</div>`:""}
      </div>
    `}function A(c){const f=c.warnings??[];return`
      <div class="card">
        <div class="service-head">
          <h2>${t(c.label)}</h2>
          ${L(c)}
        </div>
        <p class="muted small">${t(An[c.id]??"")}</p>

        ${c.error?Z(c):""}
        ${c.blocked?`<div class="banner banner-warn">${t(c.blocked)}</div>`:""}
        ${f.map(E=>`<div class="banner banner-warn">${t(E)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${t(c.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${c.status.Image?`<code>${t(c.status.Image)}</code>`:"—"}</dd></div>
        </dl>
        ${Q(c)}

        ${X(c)}

        <div class="card-actions">
          ${(c.actions??[]).map(E=>z(c,E)).join("")}
        </div>
        ${R[c.id]?`<p class="error small">${t(R[c.id])}</p>`:""}
        ${ce(c)}

        ${de(c)}
      </div>
    `}function L(c){switch(c.status.State){case"running":return U("running","ok");case"created-but-stopped":return U("stopped","warn");case"not-created":return U("not created","neutral");default:return U("unknown","bad")}}function Z(c){return`
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${t(c.error??"")}</div>
        ${c.hint?`<div class="small">${t(c.hint)}</div>`:""}
      </div>
    `}function Q(c){if(c.status.State!=="created-but-stopped"||c.status.ExitCode===0)return"";const f=c.status.ExitCode===137?" (137 is a kill — most often the machine ran out of memory)":"";return`<p class="muted small">It exited with code ${c.status.ExitCode}${f}.</p>`}function X(c){const f=c.endpoints??[];return f.length===0?c.status.State==="running"?'<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>':"":f.map(E=>`
        <div class="endpoint-row">
          ${ke("ok")}
          <span class="muted small">${t(E.label)}</span>
          <code class="endpoint-url">${t(E.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${t(E.url)}">Copy</button>
        </div>`).join("")}function z(c,f){const E=Bn[f];if(!E)return"";const W=$[c.id],G=f==="create"?`Create ${c.id==="devnet"?"devnet":"gateway"}`:E.label;return`
      <button class="${E.className}" data-action="svc-${f}" data-svc="${t(c.id)}"
              title="${t(E.title)}" ${W?"disabled":""}>
        ${W===f?'<span class="spinner" aria-label="working"></span>':t(G)}
      </button>
    `}function ce(c){const f=m[c.id]??[];return f.length===0?"":`
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${t(f.join(`
`))}</pre>
      </div>
    `}function de(c){const f=j[c.id],E=ue(c);return`
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${c.configured?"":" (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${t(c.id)}">
            ${f?"Close":"Edit"}
          </button>
        </div>
        ${f?ie():`<p class="small">${E}</p>`}
        ${_[c.id]?`<p class="error small">${t(_[c.id])}</p>`:""}
        ${H[c.id]?`<p class="muted small">${t(H[c.id])}</p>`:""}
      </div>
    `}function ue(c){const f=c.devnet;return f?`Chain ${f.ChainID} · a block every ${t(f.BlockTime)} · JSON-RPC on ${t(f.BindAddr)}:${f.HTTPPort} · WebSocket on ${t(f.BindAddr)}:${f.WSPort}`:"—"}function ie(c){return se()}function se(){const c=M;return c?`
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${t(c.BlockTime)}" autocomplete="off" spellcheck="false" />
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
        <input type="text" id="dev-bind" value="${t(c.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${c.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `:""}function re(){j.devnet&&M&&(M.BlockTime=he("#dev-blocktime",M.BlockTime),M.HTTPPort=fe("#dev-http",M.HTTPPort),M.WSPort=fe("#dev-ws",M.WSPort),M.BindAddr=he("#dev-bind",M.BindAddr))}function he(c,f){const E=s.querySelector(c);return E?E.value.trim():f}function fe(c,f){const E=s.querySelector(c);if(!E)return f;const W=Number.parseInt(E.value.trim(),10);return Number.isFinite(W)?W:f}async function me(c,f){const E=f.dataset.svc??"";switch(c){case"refresh":await P();return;case"copy":f.dataset.copy&&await v(f,f.dataset.copy);return;case"svc-start":case"svc-stop":case"svc-restart":await b(E,c.slice(4));return;case"svc-create":case"svc-recreate":await l(E);return;case"svc-wipe":N(E);return;case"toggle-config":w(E);return;case"save-config":await x(E);return;default:return}}async function b(c,f){if(!$[c]){$[c]=f,R[c]=null,d();try{await Yt(i,c,f)}catch(E){R[c]=`${f} failed: ${S(E)}${O(E)}`}$[c]=null,await P()}}async function l(c){if(!$[c]){$[c]="create",R[c]=null,m[c]=["starting…"],d();try{await Xt(i,c)}catch(f){R[c]=`${S(f)}${O(f)}`,m[c]=[],$[c]=null,d();return}B==null||B(),B=Ge(i,f=>{if(o)return;const E=f.err?`${f.stepId}: ${f.err}`:f.line?`${f.stepId}: ${f.line}`:`${f.stepId}: done`;if(m[c]=[...(m[c]??[]).filter(G=>G!=="starting…"),E],!!f.err||f.stepId===Nn&&!!f.done){B==null||B(),B=null,$[c]=null,f.err&&(R[c]="Provisioning failed — see the log below."),P();return}d()})}}function w(c){if(re(),j[c]=!j[c],_[c]=null,H[c]=null,j[c]){const f=C(c);f!=null&&f.devnet&&(M={...f.devnet})}d()}async function x(c){var W;re(),_[c]=null,H[c]=null;const f=M;if(!f)return;if(f.HTTPPort===f.WSPort){_[c]="JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.",d();return}try{await en(i,c,f)}catch(G){_[c]=S(G),d();return}const E=((W=C(c))==null?void 0:W.status.State)==="running";j[c]=!1,H[c]=E?"Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect.":"Saved.",await P()}function N(c){const f=C(c);if(!f)return;const E=(f.restartsOnWipe??[]).map(D=>{var ee;return((ee=C(D))==null?void 0:ee.label)??D});ae(`
        <h2>Wipe ${t(f.label)}</h2>
        <p class="error">This deletes ${t(f.wipeDiscards)}</p>
        ${E.length?`<p>It also restarts what sits in front of it: ${t(E.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`:""}
        <p>Type <code>${t(c)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${t(c)}</button>
        </div>
      `,D=>{if(D==="cancel"||D==="close"){J(),P();return}D==="confirm"&&q(c)});const W=document.getElementById("wipe-confirm-input"),G=document.getElementById("wipe-confirm-btn");W==null||W.addEventListener("input",()=>{G&&(G.disabled=W.value.trim()!==c)}),W==null||W.focus()}async function q(c){const f=document.getElementById("wipe-confirm-btn");f&&(f.disabled=!0,f.textContent="Wiping…");let E;try{E=await Zt(i,c)}catch(W){const G=qe();if(G){const D=document.createElement("p");D.className="error small",D.textContent=`Wipe failed: ${S(W)}${O(W)}`,G.appendChild(D)}f&&(f.disabled=!1,f.textContent=`Wipe ${c}`);return}p(c,E)}function p(c,f){const E=C(c),W=Y=>{var Se;return((Se=C(Y))==null?void 0:Se.label)??Y},G=[];G.push(f.report.ContainerRemoved?"Container removed.":"There was no container to remove.");for(const Y of f.report.VolumesRemoved??[])G.push(`Volume ${Y} deleted.`);for(const Y of f.report.VolumesAbsent??[])G.push(`Volume ${Y} was already gone.`);f.report.Recreated&&G.push("Container re-created from your saved configuration.");const D=(f.report.Cascaded??[]).map(W),ee=(f.report.CascadeSkipped??[]).map(W);ae(`
        <h2>${t((E==null?void 0:E.label)??c)} wiped</h2>
        <ul class="plain-list">${G.map(Y=>`<li>${t(Y)}</li>`).join("")}</ul>
        ${D.length?`<p class="ok">Restarted in front of it: ${t(D.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`:""}
        ${ee.length?`<p class="muted small">Not restarted (they were not running, so they held no stale head): ${t(ee.join(", "))}.</p>`:""}
        ${f.error?`<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${t(f.error)}</p>`:""}
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,Y=>{(Y==="close"||Y==="cancel")&&(J(),P())})}async function v(c,f){const E=await Ae(f),W=c.textContent;c.textContent=E?"Copied!":"Copy failed",setTimeout(()=>{o||(c.textContent=W)},1500)}function S(c){return c instanceof Error?c.message:String(c)}function O(c){return c instanceof Te&&c.hint?` — ${c.hint}`:""}return()=>{o=!0,B==null||B(),J()}}const Dn="local";function Un(s){let i=!1,o=!1,e="",h=null;s.innerHTML=`
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${oe()}
  `;const $=s.querySelector("#targets-body");$e(s,(d,k)=>{M(d,k)}),R();async function R(){try{const[d,k,A]=await Promise.all([Ee(),Pe(),Ht()]);if(i)return;e=A.os,B(d,k)}catch(d){if(i)return;$.innerHTML=`<p class="error">Failed to load machines: ${t(String(d))}</p>`}}function m(){h&&B(h.targets,h.catalog)}function B(d,k){h={targets:d,catalog:k};const A=e==="linux",L=[...d].sort((X,z)=>(X.mode==="local"?-1:0)-(z.mode==="local"?-1:0)),Z=L.length?`<div class="card-grid">${L.map(X=>Mn(X,k,X.mode!=="local"||A,e)).join("")}</div>`:'<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>',Q=d.some(X=>X.mode==="local");$.innerHTML=`
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${Z}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${j(A,Q)}
        ${o?On():""}
      </section>
    `}function j(d,k){const A=`
      <div class="card">
        <h3>A server over SSH ${U("Available","ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${d?"":" The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${d?" btn-ghost":""}" data-action="toggle-ssh">
            ${o?"Cancel":"Add a server"}
          </button>
        </div>
      </div>
    `,L=d?`
        <div class="card">
          <h3>This machine ${U("Available","ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `:`
        <div class="card card-warn">
          <h3>This machine${e?` (${t(e)})`:""} ${U("Can't run a node","warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;return k?`<div class="card-grid card-grid-wide">${A}</div>`:`<div class="card-grid card-grid-wide">${d?L+A:A+L}</div>`}async function M(d,k){var A;if(d==="add-local"){await _();return}if(d==="delete-target"){const L=k.dataset.id;if(!L||!await Ne({title:"Remove machine",body:`Remove "${L}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,confirmLabel:"Remove",danger:!0}))return;await H(L);return}if(d==="toggle-ssh"){o=!o,C(),m(),o&&((A=s.querySelector("#ssh-host"))==null||A.focus());return}d==="add-ssh"&&await F()}async function _(){C();try{await et({id:Dn,mode:"local"}),await R()}catch(d){P(d)}}async function H(d){try{await Dt(d),await R()}catch(k){P(k)}}async function F(){const d=s.querySelector("#ssh-host"),k=s.querySelector("#ssh-user"),A=s.querySelector("#ssh-key"),L=s.querySelector("#ssh-port"),Z=s.querySelector("#ssh-id");if(!d||!k||!A||!L||!Z)return;const Q=d.value.trim(),X=k.value.trim(),z=A.value.trim(),ce=L.value.trim(),de=Z.value.trim();if(C(),!Q||!X||!z){P(new Error("host, user, and key path are required"));return}const ue=de||Fn(Q),ie={Host:Q,User:X,KeyPath:z};if(ce){const re=Number.parseInt(ce,10);if(!Number.isFinite(re)||re<=0){P(new Error("port must be a positive number"));return}ie.Port=re}const se=s.querySelector("#ssh-submit");se&&(se.disabled=!0,se.textContent="Connecting…");try{await et({id:ue,mode:"ssh",ssh:ie}),o=!1,await R()}catch(re){P(re),se&&(se.disabled=!1,se.textContent="Add server")}}function P(d){let k=s.querySelector("#targets-error");k||($.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),k=s.querySelector("#targets-error")),k.textContent=String(d instanceof Error?d.message:d)}function C(){var d;(d=s.querySelector("#targets-error"))==null||d.remove()}return()=>{i=!0}}function Mn(s,i,o,e){const h=s.wire,$=s.mode==="local"?"this machine":"SSH",R=s.mode==="ssh"&&s.ssh?`${t(s.ssh.User)}@${t(s.ssh.Host)}`:$,m=`<a class="btn btn-ghost" href="#/services/${encodeURIComponent(s.id)}">Devnet</a>`;let B,j;if(!h&&!o)B=`${U("can't run a node","warn")} ${U(e||"not Linux","neutral")}`,j=`
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Preview setup wizard</a>
    `;else if(!h)B=U("not set up","neutral"),j=`
      <a class="btn" href="#/setup/${encodeURIComponent(s.id)}">Run setup wizard</a>
      ${m}
    `;else{const M=i.networks.find(H=>H.ChainID===h.ChainID),_=M?M.Name:`chain ${h.ChainID}`;B=`${U(_,"ok")} ${U(h.ExecID,"neutral")} ${U(h.BeaconID,"neutral")}${h.Archive?" "+U("archive","warn"):""}`,j=`
      <a class="btn" href="#/dash/${encodeURIComponent(s.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(s.id)}">Logs</a>
      ${m}
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(s.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${t(s.id)}</h2>
      <p class="muted">${R}</p>
      <p>${B}</p>
      <div class="card-actions">
        ${j}
        <button class="btn btn-danger" data-action="delete-target" data-id="${t(s.id)}">Remove</button>
      </div>
    </div>
  `}function On(){return`
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
  `}function Fn(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const Ke=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ue=8545,Me=5052,Oe=30303,qn=[369,943,1],ot={369:"default",943:"practise here first"};function jn(s,i){let o=!1;const e={targetId:i,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!0,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,rpcBindAddr:"",rpcBindAddrError:null,freeBytes:null,probedPath:null,diskProbing:!1,diskError:null,downgradeNote:null,checkpoint:!0,checkpointUrl:"",checkpointUrlError:null,execSnapshot:!1,snapshotKey:"",snapshotKeyError:null,starting:!1,startError:null,events:[],streamStop:null};s.innerHTML=`<h1>Setup: ${t(i)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${oe()}</div>`;const h=s.querySelector("#wizard-body"),$=s.querySelector("#wizard-footer");$e(s,(p,v)=>{fe(p,v)}),Je(s,(p,v)=>{p==="exec-select"?e.execId=v:p==="beacon-select"&&(e.beaconId=v),m()}),s.addEventListener("change",p=>{const v=p.target;v instanceof HTMLInputElement&&(v.id==="data-dir-input"?(me(),z()):v.id==="checkpoint-toggle"?(e.checkpoint=v.checked,m()):v.id==="exec-snapshot-toggle"&&(e.execSnapshot=v.checked,m()))}),R();async function R(){try{const[p,v]=await Promise.all([Pe(),Ee()]);if(o)return;e.catalog=p;const S=v.find(O=>O.id===i);S!=null&&S.wire&&(e.chainId=S.wire.ChainID,e.execId=S.wire.ExecID,e.beaconId=S.wire.BeaconID,e.archive=S.wire.Archive,S.wire.ExecHTTPPort&&(e.execHTTPPort=String(S.wire.ExecHTTPPort)),S.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(S.wire.BeaconHTTPPort)),S.wire.ExecP2PPort&&(e.execP2PPort=String(S.wire.ExecP2PPort)),S.wire.RPCBindAddr&&(e.rpcBindAddr=S.wire.RPCBindAddr)),m()}catch(p){if(o)return;e.loadError=String(p instanceof Error?p.message:p),m()}}function m(){if(e.loadError){h.innerHTML=`<p class="error">Failed to load: ${t(e.loadError)}</p>`;return}e.catalog&&(h.innerHTML=`
      ${q(e.step)}
      ${j()}
    `,B())}function B(){var v;const p=(v=e.catalog)==null?void 0:v.networks.find(S=>S.ChainID===e.chainId);$.innerHTML=p?oe(p.Name,p.LearnURL):oe()}function j(){switch(e.step){case"network":return M();case"clients":return _();case"mode":return se();case"review":return re();case"run":return he()}}function M(){const p=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${qn.map(S=>{const O=p.networks.find(E=>E.ChainID===S);if(!O)return"";const c=e.chainId===S,f=ot[S]?U(ot[S],S===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${c?"selected":""}" data-action="pick-network" data-chain-id="${S}" type="button">
          <h3>${t(O.Name)} <span class="muted">(chain ${S})</span></h3>
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
        <p class="muted">Only combinations known to work on ${t(v.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${ze("exec-select",S,e.execId)}
        </label>
        ${ie(e.execId,p)}
        <label>
          Beacon client
          ${ze("beacon-select",O,e.beaconId)}
        </label>
        ${ie(e.beaconId,p)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function H(p){return p<=0?"—":p>=1?`~${p.toFixed(1)} TB`:`~${Math.round(p*1e3)} GB`}const F=1.1,P=.5,C="Valve reth snapshot",d="rough estimate";function k(p){return p.SnapshotSizeTB}function A(p){return p.SnapshotSizeTB*P}function L(p){return`<p class="muted small">${H(k(p))} is the measured size of Valve's reth snapshot for ${t(p.Name)}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.</p>`}function Z(p){return{archive:k(p)*1e12*F,full:A(p)*1e12*F}}function Q(p,v){if(!p)return"";if(e.diskProbing)return`<p class="muted small">Checking free space at <code>${t(v)}</code>…</p>`;if(e.diskError)return`<p class="error small">Couldn't read free space at <code>${t(v)}</code>: ${t(e.diskError)}</p>`;if(e.freeBytes===null||e.probedPath!==v)return"";const S=Z(p),O=e.freeBytes>=S.archive,c=e.freeBytes>=S.full,f=`<p class="muted small">Free at <code>${t(v)}</code>: <strong>${Ce(e.freeBytes)}</strong> — archive ${O?"fits":"won't fit"} (${H(k(p))}, ${C}), full ${c?"fits":"won't fit"} (${H(A(p))}, ${d}).</p>`;let E="";return e.downgradeNote?E=`<p class="banner banner-warn">${t(e.downgradeNote)}</p>`:c||(E=`<p class="banner banner-warn">Neither full (${H(A(p))}, ${d}) nor archive (${H(k(p))}, ${C}) fits the free space here — choose a location with more room.</p>`),f+E}function X(p,v){if(e.downgradeNote=null,!p||e.freeBytes===null)return;const S=Z(p);e.archive&&e.freeBytes<S.archive&&e.freeBytes>=S.full&&(e.archive=!1,e.downgradeNote=`Not enough space at ${v} for archive (${H(k(p))}, ${C}) — switched to Full (${H(A(p))}, ${d}). Pick a location with more room to run archive.`)}async function z(){var S;if(e.chainId===null)return;const p=(S=e.catalog)==null?void 0:S.networks.find(O=>O.ChainID===e.chainId),v=(e.dataDir||`/var/lib/valve-node-app/${e.chainId}`).trim();e.diskProbing=!0,e.diskError=null,m();try{const{freeBytes:O}=await Ut(e.targetId,v);if(o)return;e.freeBytes=O,e.probedPath=v,X(p,v)}catch(O){if(o)return;e.freeBytes=null,e.probedPath=v,e.diskError=String(O instanceof Error?O.message:O)}e.diskProbing=!1,m()}function ce(p){return p?/^https?:\/\/.+/i.test(p)?null:"Enter an http(s) URL, or leave blank for the network default.":null}function de(p,v){const S=v.clients.find(O=>O.id===p);return{value:p,label:S?`${S.id} — ${ue(S.repo)}`:p}}function ue(p){const v=p.split("/");return v.length>=4?v[3]:p}function ie(p,v){const S=p?v.clients.find(c=>c.id===p):void 0;if(!S)return"";const O=S.repo.replace(/^https?:\/\//,"");return`<p class="muted small">Source: <a href="${t(S.repo)}" target="_blank" rel="noopener noreferrer">${t(O)}</a></p>`}function se(){var W,G,D;const p=e.chainId!==null?`/var/lib/valve-node-app/${e.chainId}`:"",v=(W=e.catalog)==null?void 0:W.networks.find(ee=>ee.ChainID===e.chainId),S=((D=(G=e.catalog)==null?void 0:G.clients.find(ee=>ee.id===e.execId))==null?void 0:D.snapshotSupported)??!1,O=v?`${H(A(v))} (${d})`:"Smaller",c=v?`${H(k(v))} (${C})`:"Much larger",f=v?` on ${t(v.Name)}`:"",E=v?e.checkpoint?v.SyncLabel:v.GenesisSyncLabel:"";return`
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
          ${v?`<p class="sync-estimate">⏱ Estimated initial sync${f}: <strong>${t(E)}</strong></p>
                   <p class="muted small">Scales with the target's CPU and disk speed.</p>`:""}
          ${e.checkpoint?`<label>
                   Checkpoint URL <span class="muted">(default: ${t((v==null?void 0:v.CheckpointURL)??"")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${t((v==null?void 0:v.CheckpointURL)??"")}" value="${t(e.checkpointUrl)}" />
                 </label>
                 ${e.checkpointUrlError?`<p class="error small">${t(e.checkpointUrlError)}</p>`:""}
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
                   <input id="snapshot-key-input" type="text" placeholder="vk_…" value="${t(e.snapshotKey)}" />
                 </label>
                 ${e.snapshotKeyError?`<p class="error small">${t(e.snapshotKeyError)}</p>`:""}
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
          ${v?L(v):'<p class="muted small">Disk sizes depend on the network, the execution client and the sync mode — pick a network to see figures.</p>'}
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
            Data location <span class="muted">(default: ${t(p)})</span>
            <input id="data-dir-input" type="text" placeholder="${t(p)}" value="${t(e.dataDir)}" />
          </label>
          ${Q(v,e.dataDir||p)}
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${t(p)}/jwt.hex" value="${t(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Ue})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Ue}" value="${t(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${t(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${Me})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${Me}" value="${t(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${t(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${Oe})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${Oe}" value="${t(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${t(e.execP2PPortError)}</p>`:""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${t(e.rpcBindAddr)}" />
          </label>
          ${e.rpcBindAddrError?`<p class="error small">${t(e.rpcBindAddrError)}</p>`:""}
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
    `}function re(){const v=e.catalog.networks.find(Y=>Y.ChainID===e.chainId),S=e.dataDir||`/var/lib/valve-node-app/${e.chainId}`,O=e.jwtPath||`${S}/jwt.hex`,c=Ke.map(Y=>`<li>${t(Y.title)}</li>`).join(""),f=x(e.execHTTPPort,Ue),E=x(e.beaconHTTPPort,Me),W=x(e.execP2PPort,Oe),G=f||E||W?`<tr><th>Non-default ports</th><td>${[f?`exec HTTP ${f}`:null,E?`beacon HTTP ${E}`:null,W?`exec p2p ${W}`:null].filter(Y=>Y!==null).map(t).join(", ")}</td></tr>`:"",{addr:D}=b(e.rpcBindAddr),ee=D?`<tr><th>RPC bind address</th><td><code>${t(D)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${t(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${t((v==null?void 0:v.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${t(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${t(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${t(S)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${t(O)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${e.checkpoint?`<code>${t(e.checkpointUrl||(v==null?void 0:v.CheckpointURL)||"")}</code>`:"off — syncing from genesis"}</td></tr>
            ${G}
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
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function he(){const v=e.catalog.networks.find(D=>D.ChainID===e.chainId),S=v==null?void 0:v.LearnURL,O=new Set(e.events.filter(D=>D.done).map(D=>D.stepId)),c=new Set(e.events.filter(D=>D.err).map(D=>D.stepId)),f=new Map;for(const D of e.events){if(!D.line)continue;const ee=f.get(D.stepId)??[];ee.push(D.line),f.set(D.stepId,ee)}const E=Ke.map(D=>{var be;const ee=O.has(D.id),Y=c.has(D.id),Se=Y?U("failed","bad"):ee?U("done","ok"):U("pending","neutral"),Be=(f.get(D.id)??[]).slice(-5),He=(be=e.events.find(xe=>xe.stepId===D.id&&xe.err))==null?void 0:be.err,ve=D.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${S?` <a href="${t(S)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${ee?"step-done":""} ${Y?"step-error":""}">
          <div class="step-head">${Se} <strong>${t(D.title)}</strong></div>
          ${ve}
          ${Be.length?`<pre class="step-log">${Be.map(xe=>t(xe)).join(`
`)}</pre>`:""}
          ${He?`<p class="error small">${t(He)}</p>`:""}
        </li>
      `}).join(""),W=e.events.some(D=>D.err),G=Ke.every(D=>O.has(D.id))||e.events.some(D=>D.stepId==="handshake"&&D.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${E}</ol>
        ${G&&!W?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${e.startError?`<p class="error">${t(e.startError)}</p>`:""}
        ${W?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function fe(p,v){switch(p){case"pick-network":e.chainId=Number(v.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":e.step="mode",m(),z();break;case"goto-review":if(me(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError||e.rpcBindAddrError||e.checkpointUrlError||e.snapshotKeyError){m();break}e.step="review",m();break;case"start-setup":N();break}}function me(){const p=s.querySelectorAll('input[name="mode"]');for(const D of Array.from(p))D.checked&&(e.archive=D.value==="archive");const v=s.querySelector("#data-dir-input"),S=s.querySelector("#jwt-path-input");v&&(e.dataDir=v.value.trim()),S&&(e.jwtPath=S.value.trim());const O=s.querySelector("#exec-http-port-input"),c=s.querySelector("#beacon-http-port-input"),f=s.querySelector("#exec-p2p-port-input");O&&(e.execHTTPPort=O.value.trim()),c&&(e.beaconHTTPPort=c.value.trim()),f&&(e.execP2PPort=f.value.trim());const E=s.querySelector("#rpc-bind-addr-input");E&&(e.rpcBindAddr=E.value.trim());const W=s.querySelector("#checkpoint-url-input");W&&(e.checkpointUrl=W.value.trim());const G=s.querySelector("#snapshot-key-input");G&&(e.snapshotKey=G.value.trim()),e.execHTTPPortError=w(e.execHTTPPort).error??null,e.beaconHTTPPortError=w(e.beaconHTTPPort).error??null,e.execP2PPortError=w(e.execP2PPort).error??null,e.rpcBindAddrError=b(e.rpcBindAddr).error??null,e.checkpointUrlError=e.checkpoint?ce(e.checkpointUrl):null,e.snapshotKeyError=e.execSnapshot&&!e.snapshotKey.trim()?"A free snapshot key is required (get one at valve.city).":null}function b(p){if(!p)return{};const v=/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(p);return v?v.slice(1).every(S=>Number(S)<=255)?{addr:p}:{error:"Each part of an IPv4 address must be 0–255."}:/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(p)&&p.includes(":")?{addr:p}:{error:"Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback."}}const l=/^\d+$/;function w(p){if(!p)return{};if(!l.test(p))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const v=Number(p);return!Number.isInteger(v)||v<1||v>65535?{error:"Port must be between 1 and 65535."}:{port:v}}function x(p,v){const{port:S}=w(p);if(!(S===void 0||S===v))return S}async function N(){var f;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,e.events=[],(f=e.streamStop)==null||f.call(e),e.streamStop=null,m();const p={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(p.DataDir=e.dataDir),e.jwtPath&&(p.JWTPath=e.jwtPath);const v=x(e.execHTTPPort,Ue),S=x(e.beaconHTTPPort,Me),O=x(e.execP2PPort,Oe);v!==void 0&&(p.ExecHTTPPort=v),S!==void 0&&(p.BeaconHTTPPort=S),O!==void 0&&(p.ExecP2PPort=O);const{addr:c}=b(e.rpcBindAddr);c!==void 0&&(p.RPCBindAddr=c),e.checkpoint?e.checkpointUrl&&(p.CheckpointURL=e.checkpointUrl):p.NoCheckpoint=!0,e.execSnapshot&&(p.ExecSnapshot=!0,p.SnapshotKey=e.snapshotKey);try{await Mt(e.targetId,p)}catch(E){if(!(E instanceof Te&&E.status===409)){e.starting=!1,e.startError=String(E instanceof Error?E.message:E),m();return}}e.starting=!1,e.step="run",m(),e.streamStop=Ge(e.targetId,E=>{o||(e.events.push(E),e.step==="run"&&m())})}function q(p){const v=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],O=v.map(c=>c.id).indexOf(p);return`
      <ol class="wizard-progress">
        ${v.map((c,f)=>`<li class="${f===O?"current":f<O?"past":"future"}">${t(c.label)}</li>`).join("")}
      </ol>
    `}return()=>{var p;o=!0,(p=e.streamStop)==null||p.call(e)}}const Wn=document.querySelector("#app"),{contentEl:_n,setActiveNav:Kn}=mn(Wn);let le=null;function zn(){const i=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(i.length===0)return{screen:"targets"};const[o,e]=i;return o==="setup"||o==="dash"||o==="logs"||o==="security"||o==="diag"||o==="services"||o==="analytics"?{screen:o,id:e?decodeURIComponent(e):void 0}:{screen:o??"targets"}}function ye(s){const i=document.createElement("div");return _n.replaceChildren(i),s(i)}function ct(){if(le){try{le()}catch{}le=null}const{screen:s,id:i}=zn();switch(Kn(s),s){case"setup":if(!i){location.hash="#/targets";return}le=ye(o=>jn(o,i));break;case"dash":if(!i){location.hash="#/targets";return}le=ye(o=>wn(o,i));break;case"logs":if(!i){location.hash="#/targets";return}le=ye(o=>kn(o,i));break;case"security":if(!i){location.hash="#/targets";return}le=ye(o=>Cn(o,i));break;case"diag":if(!i){location.hash="#/targets";return}le=ye(o=>Tn(o,i));break;case"services":if(!i){location.hash="#/targets";return}le=ye(o=>Hn(o,i));break;case"analytics":if(!i){location.hash="#/rpc";return}le=ye(o=>vn(o,i));break;case"rpc":le=ye(o=>Ln(o));break;case"settings":le=ye(o=>xn(o));break;case"targets":default:le=ye(o=>Un(o));break}}window.addEventListener("hashchange",ct);ct();
