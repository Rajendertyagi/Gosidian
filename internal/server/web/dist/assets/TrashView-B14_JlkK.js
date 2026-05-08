import{z as y,d as w,x as M,c as s,a as c,t as u,f as T,F as C,s as V,r as p,n as R,M as F,m as h,k as f,e as v,o as t}from"./index-CrclG0B9.js";import{u as I}from"./tree-CeFxOZNa.js";import{c as x}from"./createLucideIcon-CTqQzvbB.js";/**
 * @license lucide-vue-next v0.453.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const H=x("FileTextIcon",[["path",{d:"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z",key:"1rqfz7"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M10 9H8",key:"b1mrlr"}],["path",{d:"M16 13H8",key:"t4e002"}],["path",{d:"M16 17H8",key:"z1uh3a"}]]);/**
 * @license lucide-vue-next v0.453.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const $=x("FolderIcon",[["path",{d:"M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z",key:"1kt360"}]]);/**
 * @license lucide-vue-next v0.453.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const L=x("RotateCcwIcon",[["path",{d:"M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8",key:"1357e3"}],["path",{d:"M3 3v5h5",key:"1xhq8a"}]]);/**
 * @license lucide-vue-next v0.453.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const B=x("Trash2Icon",[["path",{d:"M3 6h18",key:"d0wm0j"}],["path",{d:"M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6",key:"4alrt4"}],["path",{d:"M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2",key:"v07s0e"}],["line",{x1:"10",x2:"10",y1:"11",y2:"17",key:"1uufr5"}],["line",{x1:"14",x2:"14",y1:"11",y2:"17",key:"xtxkd"}]]);async function E(){const{data:o}=await y.get("/trash");return o.items}async function N(o){const{data:l}=await y.post(`/trash/${encodeURIComponent(o)}/restore`,{});return l}async function P(o){await y.delete(`/trash/${encodeURIComponent(o)}`)}const S={class:"p-8 max-w-4xl mx-auto"},q={key:0,class:"text-text-muted"},z={key:1,class:"text-danger"},A={key:2,class:"text-success text-sm mb-3"},D={key:3,class:"text-text-muted text-sm"},U={key:4,class:"space-y-2"},Z={class:"flex-1 font-mono text-sm truncate"},j={class:"text-xs text-text-muted whitespace-nowrap"},G=["onClick"],J=["onClick"],W=w({__name:"TrashView",setup(o){const l=p([]),d=p(!1),n=p(null),i=p(null),k=I();async function m(){d.value=!0,n.value=null;try{l.value=await E()}catch(a){n.value=a instanceof Error?a.message:"Failed to load"}finally{d.value=!1}}async function g(a){try{const e=await N(a.id);i.value=`Restored to ${e.restored}`,k.invalidateAll(),await m()}catch(e){n.value=e instanceof Error?e.message:"Restore failed"}}async function _(a){if(confirm(`Permanently delete "${a.origin_path}"? This cannot be undone.`))try{await P(a.id),i.value="Purged.",await m()}catch(e){n.value=e instanceof Error?e.message:"Purge failed"}}return M(m),(a,e)=>(t(),s("div",S,[e[2]||(e[2]=c("h1",{class:"text-2xl font-semibold mb-1"},"Trash",-1)),e[3]||(e[3]=c("p",{class:"text-sm text-text-muted mb-6"}," Soft-deleted notes and folders. Restore puts them back at their original path; purge wipes them for good. ",-1)),d.value?(t(),s("p",q,"Loading…")):n.value?(t(),s("p",z,u(n.value),1)):i.value?(t(),s("p",A,u(i.value),1)):T("",!0),!d.value&&!l.value.length?(t(),s("p",D,"Trash is empty.")):(t(),s("ul",U,[(t(!0),s(C,null,V(l.value,r=>(t(),s("li",{key:r.id,class:"rounded border border-border bg-surface px-4 py-3 flex items-center gap-3"},[(t(),R(F(r.is_dir?h($):h(H)),{class:"w-4 h-4 text-text-muted shrink-0"})),c("span",Z,u(r.origin_path),1),c("span",j,u(r.discarded_at),1),c("button",{type:"button",class:"text-xs px-2 py-1 rounded border border-border hover:bg-surface-hover inline-flex items-center gap-1",onClick:b=>g(r)},[f(h(L),{class:"w-3 h-3"}),e[0]||(e[0]=v(" Restore",-1))],8,G),c("button",{type:"button",class:"text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover inline-flex items-center gap-1",onClick:b=>_(r)},[f(h(B),{class:"w-3 h-3"}),e[1]||(e[1]=v(" Purge",-1))],8,J)]))),128))]))]))}});export{W as default};
//# sourceMappingURL=TrashView-B14_JlkK.js.map
