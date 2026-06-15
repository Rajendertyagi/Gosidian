import{n}from"./index-ooM2UP9z.js";async function c(){const{data:t}=await n.get("/auth-config");return t}async function i(){const{data:t}=await n.post("/totp/enroll",{});return t}async function s(t,a){await n.post("/totp/confirm",{secret:t,code:a})}async function e(){await n.delete("/totp")}export{s as c,e as d,i as e,c as g};
//# sourceMappingURL=totp-BoWn2mBK.js.map
