import{S as ve,i as ye,s as ge,N as we,e as s,w as f,b as d,c as at,f as h,g as r,h as e,m as st,x as Mt,O as ue,P as $e,k as Pe,Q as Re,n as Ce,t as Z,a as x,o as c,d as nt,T as Oe,C as fe,p as Te,r as it,u as Ae}from"./index-9b3e256b.js";import{S as Ue}from"./SdkTabs-00028c2e.js";import{F as Me}from"./FieldsQueryParam-96f2e561.js";function pe(n,l,o){const i=n.slice();return i[8]=l[o],i}function be(n,l,o){const i=n.slice();return i[8]=l[o],i}function De(n){let l;return{c(){l=f("email")},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function Ee(n){let l;return{c(){l=f("username")},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function We(n){let l;return{c(){l=f("username/email")},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function me(n){let l;return{c(){l=s("strong"),l.textContent="username"},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function he(n){let l;return{c(){l=f("or")},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function _e(n){let l;return{c(){l=s("strong"),l.textContent="email"},m(o,i){r(o,l,i)},d(o){o&&c(l)}}}function ke(n,l){let o,i=l[8].code+"",S,m,p,u;function _(){return l[7](l[8])}return{key:n,first:null,c(){o=s("button"),S=f(i),m=d(),h(o,"class","tab-item"),it(o,"active",l[3]===l[8].code),this.first=o},m(R,C){r(R,o,C),e(o,S),e(o,m),p||(u=Ae(o,"click",_),p=!0)},p(R,C){l=R,C&16&&i!==(i=l[8].code+"")&&Mt(S,i),C&24&&it(o,"active",l[3]===l[8].code)},d(R){R&&c(o),p=!1,u()}}}function Se(n,l){let o,i,S,m;return i=new we({props:{content:l[8].body}}),{key:n,first:null,c(){o=s("div"),at(i.$$.fragment),S=d(),h(o,"class","tab-item"),it(o,"active",l[3]===l[8].code),this.first=o},m(p,u){r(p,o,u),st(i,o,null),e(o,S),m=!0},p(p,u){l=p;const _={};u&16&&(_.content=l[8].body),i.$set(_),(!m||u&24)&&it(o,"active",l[3]===l[8].code)},i(p){m||(Z(i.$$.fragment,p),m=!0)},o(p){x(i.$$.fragment,p),m=!1},d(p){p&&c(o),nt(i)}}}function Le(n){var ie,re;let l,o,i=n[0].name+"",S,m,p,u,_,R,C,O,B,Dt,rt,A,ct,N,dt,U,tt,Et,et,I,Wt,ut,lt=n[0].name+"",ft,Lt,pt,V,bt,M,mt,Bt,Q,D,ht,qt,_t,Ft,$,Ht,kt,St,wt,Yt,vt,yt,j,gt,E,$t,Nt,J,W,Pt,It,Rt,Vt,k,Qt,q,jt,Jt,Kt,Ct,zt,Ot,Gt,Xt,Zt,Tt,xt,te,F,At,K,Ut,L,z,T=[],ee=new Map,le,G,w=[],oe=new Map,H;function ae(t,a){if(t[1]&&t[2])return We;if(t[1])return Ee;if(t[2])return De}let Y=ae(n),P=Y&&Y(n);A=new Ue({props:{js:`
        import PocketBase from 'pocketbase';

        const pb = new PocketBase('${n[6]}');

        ...

        const authData = await pb.collection('${(ie=n[0])==null?void 0:ie.name}').authWithPassword(
            '${n[5]}',
            'YOUR_PASSWORD',
        );

        // after the above you can also access the auth data from the authStore
        console.log(pb.authStore.isValid);
        console.log(pb.authStore.token);
        console.log(pb.authStore.model.id);

        // "logout" the last authenticated account
        pb.authStore.clear();
    `,dart:`
        import 'package:pocketbase/pocketbase.dart';

        final pb = PocketBase('${n[6]}');

        ...

        final authData = await pb.collection('${(re=n[0])==null?void 0:re.name}').authWithPassword(
          '${n[5]}',
          'YOUR_PASSWORD',
        );

        // after the above you can also access the auth data from the authStore
        print(pb.authStore.isValid);
        print(pb.authStore.token);
        print(pb.authStore.model.id);

        // "logout" the last authenticated account
        pb.authStore.clear();
    `}});let v=n[1]&&me(),y=n[1]&&n[2]&&he(),g=n[2]&&_e();q=new we({props:{content:"?expand=relField1,relField2.subRelField"}}),F=new Me({});let ot=n[4];const se=t=>t[8].code;for(let t=0;t<ot.length;t+=1){let a=be(n,ot,t),b=se(a);ee.set(b,T[t]=ke(b,a))}let X=n[4];const ne=t=>t[8].code;for(let t=0;t<X.length;t+=1){let a=pe(n,X,t),b=ne(a);oe.set(b,w[t]=Se(b,a))}return{c(){l=s("h3"),o=f("Auth with password ("),S=f(i),m=f(")"),p=d(),u=s("div"),_=s("p"),R=f(`Returns new auth token and account data by a combination of
        `),C=s("strong"),P&&P.c(),O=f(`
        and `),B=s("strong"),B.textContent="password",Dt=f("."),rt=d(),at(A.$$.fragment),ct=d(),N=s("h6"),N.textContent="API details",dt=d(),U=s("div"),tt=s("strong"),tt.textContent="POST",Et=d(),et=s("div"),I=s("p"),Wt=f("/api/collections/"),ut=s("strong"),ft=f(lt),Lt=f("/auth-with-password"),pt=d(),V=s("div"),V.textContent="Body Parameters",bt=d(),M=s("table"),mt=s("thead"),mt.innerHTML=`<tr><th>Param</th> 
            <th>Type</th> 
            <th width="50%">Description</th></tr>`,Bt=d(),Q=s("tbody"),D=s("tr"),ht=s("td"),ht.innerHTML=`<div class="inline-flex"><span class="label label-success">Required</span> 
                    <span>identity</span></div>`,qt=d(),_t=s("td"),_t.innerHTML='<span class="label">String</span>',Ft=d(),$=s("td"),Ht=f(`The
                `),v&&v.c(),kt=d(),y&&y.c(),St=d(),g&&g.c(),wt=f(`
                of the record to authenticate.`),Yt=d(),vt=s("tr"),vt.innerHTML=`<td><div class="inline-flex"><span class="label label-success">Required</span> 
                    <span>password</span></div></td> 
            <td><span class="label">String</span></td> 
            <td>The auth record password.</td>`,yt=d(),j=s("div"),j.textContent="Query parameters",gt=d(),E=s("table"),$t=s("thead"),$t.innerHTML=`<tr><th>Param</th> 
            <th>Type</th> 
            <th width="60%">Description</th></tr>`,Nt=d(),J=s("tbody"),W=s("tr"),Pt=s("td"),Pt.textContent="expand",It=d(),Rt=s("td"),Rt.innerHTML='<span class="label">String</span>',Vt=d(),k=s("td"),Qt=f(`Auto expand record relations. Ex.:
                `),at(q.$$.fragment),jt=f(`
                Supports up to 6-levels depth nested relations expansion. `),Jt=s("br"),Kt=f(`
                The expanded relations will be appended to the record under the
                `),Ct=s("code"),Ct.textContent="expand",zt=f(" property (eg. "),Ot=s("code"),Ot.textContent='"expand": {"relField1": {...}, ...}',Gt=f(`).
                `),Xt=s("br"),Zt=f(`
                Only the relations to which the request user has permissions to `),Tt=s("strong"),Tt.textContent="view",xt=f(" will be expanded."),te=d(),at(F.$$.fragment),At=d(),K=s("div"),K.textContent="Responses",Ut=d(),L=s("div"),z=s("div");for(let t=0;t<T.length;t+=1)T[t].c();le=d(),G=s("div");for(let t=0;t<w.length;t+=1)w[t].c();h(l,"class","m-b-sm"),h(u,"class","content txt-lg m-b-sm"),h(N,"class","m-b-xs"),h(tt,"class","label label-primary"),h(et,"class","content"),h(U,"class","alert alert-success"),h(V,"class","section-title"),h(M,"class","table-compact table-border m-b-base"),h(j,"class","section-title"),h(E,"class","table-compact table-border m-b-base"),h(K,"class","section-title"),h(z,"class","tabs-header compact left"),h(G,"class","tabs-content"),h(L,"class","tabs")},m(t,a){r(t,l,a),e(l,o),e(l,S),e(l,m),r(t,p,a),r(t,u,a),e(u,_),e(_,R),e(_,C),P&&P.m(C,null),e(_,O),e(_,B),e(_,Dt),r(t,rt,a),st(A,t,a),r(t,ct,a),r(t,N,a),r(t,dt,a),r(t,U,a),e(U,tt),e(U,Et),e(U,et),e(et,I),e(I,Wt),e(I,ut),e(ut,ft),e(I,Lt),r(t,pt,a),r(t,V,a),r(t,bt,a),r(t,M,a),e(M,mt),e(M,Bt),e(M,Q),e(Q,D),e(D,ht),e(D,qt),e(D,_t),e(D,Ft),e(D,$),e($,Ht),v&&v.m($,null),e($,kt),y&&y.m($,null),e($,St),g&&g.m($,null),e($,wt),e(Q,Yt),e(Q,vt),r(t,yt,a),r(t,j,a),r(t,gt,a),r(t,E,a),e(E,$t),e(E,Nt),e(E,J),e(J,W),e(W,Pt),e(W,It),e(W,Rt),e(W,Vt),e(W,k),e(k,Qt),st(q,k,null),e(k,jt),e(k,Jt),e(k,Kt),e(k,Ct),e(k,zt),e(k,Ot),e(k,Gt),e(k,Xt),e(k,Zt),e(k,Tt),e(k,xt),e(J,te),st(F,J,null),r(t,At,a),r(t,K,a),r(t,Ut,a),r(t,L,a),e(L,z);for(let b=0;b<T.length;b+=1)T[b]&&T[b].m(z,null);e(L,le),e(L,G);for(let b=0;b<w.length;b+=1)w[b]&&w[b].m(G,null);H=!0},p(t,[a]){var ce,de;(!H||a&1)&&i!==(i=t[0].name+"")&&Mt(S,i),Y!==(Y=ae(t))&&(P&&P.d(1),P=Y&&Y(t),P&&(P.c(),P.m(C,null)));const b={};a&97&&(b.js=`
        import PocketBase from 'pocketbase';

        const pb = new PocketBase('${t[6]}');

        ...

        const authData = await pb.collection('${(ce=t[0])==null?void 0:ce.name}').authWithPassword(
            '${t[5]}',
            'YOUR_PASSWORD',
        );

        // after the above you can also access the auth data from the authStore
        console.log(pb.authStore.isValid);
        console.log(pb.authStore.token);
        console.log(pb.authStore.model.id);

        // "logout" the last authenticated account
        pb.authStore.clear();
    `),a&97&&(b.dart=`
        import 'package:pocketbase/pocketbase.dart';

        final pb = PocketBase('${t[6]}');

        ...

        final authData = await pb.collection('${(de=t[0])==null?void 0:de.name}').authWithPassword(
          '${t[5]}',
          'YOUR_PASSWORD',
        );

        // after the above you can also access the auth data from the authStore
        print(pb.authStore.isValid);
        print(pb.authStore.token);
        print(pb.authStore.model.id);

        // "logout" the last authenticated account
        pb.authStore.clear();
    `),A.$set(b),(!H||a&1)&&lt!==(lt=t[0].name+"")&&Mt(ft,lt),t[1]?v||(v=me(),v.c(),v.m($,kt)):v&&(v.d(1),v=null),t[1]&&t[2]?y||(y=he(),y.c(),y.m($,St)):y&&(y.d(1),y=null),t[2]?g||(g=_e(),g.c(),g.m($,wt)):g&&(g.d(1),g=null),a&24&&(ot=t[4],T=ue(T,a,se,1,t,ot,ee,z,$e,ke,null,be)),a&24&&(X=t[4],Pe(),w=ue(w,a,ne,1,t,X,oe,G,Re,Se,null,pe),Ce())},i(t){if(!H){Z(A.$$.fragment,t),Z(q.$$.fragment,t),Z(F.$$.fragment,t);for(let a=0;a<X.length;a+=1)Z(w[a]);H=!0}},o(t){x(A.$$.fragment,t),x(q.$$.fragment,t),x(F.$$.fragment,t);for(let a=0;a<w.length;a+=1)x(w[a]);H=!1},d(t){t&&c(l),t&&c(p),t&&c(u),P&&P.d(),t&&c(rt),nt(A,t),t&&c(ct),t&&c(N),t&&c(dt),t&&c(U),t&&c(pt),t&&c(V),t&&c(bt),t&&c(M),v&&v.d(),y&&y.d(),g&&g.d(),t&&c(yt),t&&c(j),t&&c(gt),t&&c(E),nt(q),nt(F),t&&c(At),t&&c(K),t&&c(Ut),t&&c(L);for(let a=0;a<T.length;a+=1)T[a].d();for(let a=0;a<w.length;a+=1)w[a].d()}}}function Be(n,l,o){let i,S,m,p,{collection:u=new Oe}=l,_=200,R=[];const C=O=>o(3,_=O.code);return n.$$set=O=>{"collection"in O&&o(0,u=O.collection)},n.$$.update=()=>{var O,B;n.$$.dirty&1&&o(2,S=(O=u==null?void 0:u.options)==null?void 0:O.allowEmailAuth),n.$$.dirty&1&&o(1,m=(B=u==null?void 0:u.options)==null?void 0:B.allowUsernameAuth),n.$$.dirty&6&&o(5,p=m&&S?"YOUR_USERNAME_OR_EMAIL":m?"YOUR_USERNAME":"YOUR_EMAIL"),n.$$.dirty&1&&o(4,R=[{code:200,body:JSON.stringify({token:"JWT_TOKEN",record:fe.dummyCollectionRecord(u)},null,2)},{code:400,body:`
                {
                  "code": 400,
                  "message": "Failed to authenticate.",
                  "data": {
                    "identity": {
                      "code": "validation_required",
                      "message": "Missing required value."
                    }
                  }
                }
            `}])},o(6,i=fe.getApiExampleUrl(Te.baseUrl)),[u,m,S,_,R,p,i,C]}class Ye extends ve{constructor(l){super(),ye(this,l,Be,Le,ge,{collection:0})}}export{Ye as default};
