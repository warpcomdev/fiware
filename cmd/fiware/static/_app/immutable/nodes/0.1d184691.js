import{S as l,i,s as u,C as c,D as _,E as f,F as d,g as p,d as m,G as b}from"../chunks/index.61187ccf.js";import{p as g}from"../chunks/preferDark.4a1c90e5.js";const y=!0,D=Object.freeze(Object.defineProperty({__proto__:null,prerender:y},Symbol.toStringTag,{value:"Module"}));function $(o){let s;const a=o[2].default,e=c(a,o,o[1],null);return{c(){e&&e.c()},l(t){e&&e.l(t)},m(t,r){e&&e.m(t,r),s=!0},p(t,[r]){e&&e.p&&(!s||r&2)&&_(e,a,t,t[1],s?d(a,t[1],r,null):f(t[1]),null)},i(t){s||(p(e,t),s=!0)},o(t){m(e,t),s=!1},d(t){e&&e.d(t)}}}function k(o,s,a){let e;b(o,g,n=>a(0,e=n));let{$$slots:t={},$$scope:r}=s;return o.$$set=n=>{"$$scope"in n&&a(1,r=n.$$scope)},o.$$.update=()=>{o.$$.dirty&1&&(e?document.body.classList.add("dark"):document.body.classList.remove("dark"))},[e,r,t]}class L extends l{constructor(s){super(),i(this,s,k,$,u,{})}}export{L as component,D as universal};