module.exports = [
"[externals]/next/dist/compiled/next-server/app-page-turbo.runtime.dev.js [external] (next/dist/compiled/next-server/app-page-turbo.runtime.dev.js, cjs)", ((__turbopack_context__, module, exports) => {

const mod = __turbopack_context__.x("next/dist/compiled/next-server/app-page-turbo.runtime.dev.js", () => require("next/dist/compiled/next-server/app-page-turbo.runtime.dev.js"));

module.exports = mod;
}),
"[project]/satgate-landing/app/playground/page.tsx [app-ssr] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "default",
    ()=>PlaygroundPage
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/next/dist/server/route-modules/app-page/vendored/ssr/react-jsx-dev-runtime.js [app-ssr] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/next/dist/server/route-modules/app-page/vendored/ssr/react.js [app-ssr] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$zap$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Zap$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/zap.js [app-ssr] (ecmascript) <export default as Zap>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$shield$2d$alert$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__ShieldAlert$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/shield-alert.js [app-ssr] (ecmascript) <export default as ShieldAlert>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$circle$2d$check$2d$big$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__CheckCircle$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/circle-check-big.js [app-ssr] (ecmascript) <export default as CheckCircle>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$play$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Play$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/play.js [app-ssr] (ecmascript) <export default as Play>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$arrow$2d$left$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__ArrowLeft$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/arrow-left.js [app-ssr] (ecmascript) <export default as ArrowLeft>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wifi$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Wifi$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/wifi.js [app-ssr] (ecmascript) <export default as Wifi>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wifi$2d$off$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__WifiOff$3e$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/lucide-react/dist/esm/icons/wifi-off.js [app-ssr] (ecmascript) <export default as WifiOff>");
var __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$client$2f$app$2d$dir$2f$link$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/satgate-landing/node_modules/next/dist/client/app-dir/link.js [app-ssr] (ecmascript)");
'use client';
;
;
;
;
// --- ENDPOINT OPTIONS ---
const ENDPOINTS = [
    {
        path: '/api/micro/ping',
        price: 1,
        label: '/api/micro/ping (1 sat = $0.001)'
    },
    {
        path: '/api/basic/quote',
        price: 10,
        label: '/api/basic/quote (10 sats = $0.01)'
    },
    {
        path: '/api/standard/analytics',
        price: 100,
        label: '/api/standard/analytics (100 sats = $0.10)'
    },
    {
        path: '/api/premium/insights',
        price: 1000,
        label: '/api/premium/insights (1000 sats = $1.00)'
    }
];
// --- 1. MOCK CLIENT (Simulation) ---
class MockSatGateClient {
    async get() {
        await new Promise((r)=>setTimeout(r, 800));
        const error = new Error("Payment Required");
        error.status = 402;
        error.headers = {
            get: ()=>'L402 macaroon="mock_mac", invoice="lnbc10u1..."'
        };
        throw error;
    }
}
// --- 2. REAL CLIENT (WebLN / Alby) ---
class RealSatGateClient {
    async get(url, token) {
        const headers = {};
        if (token) headers['Authorization'] = token;
        const res = await fetch(url, {
            headers,
            cache: 'no-store'
        });
        if (res.status === 402) {
            const error = new Error("Payment Required");
            error.status = 402;
            error.headers = res.headers;
            try {
                error.body = await res.text();
            } catch (e) {}
            throw error;
        }
        if (!res.ok) {
            const txt = await res.text();
            throw new Error(`Server Error ${res.status}: ${txt}`);
        }
        return res.json();
    }
}
function PlaygroundPage() {
    const [useRealNetwork, setUseRealNetwork] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useState"])(false);
    const [selectedEndpoint, setSelectedEndpoint] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useState"])(ENDPOINTS[0]); // Default to cheapest
    const [logs, setLogs] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useState"])([]);
    const [status, setStatus] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useState"])('idle');
    const [isLoading, setIsLoading] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useState"])(false);
    const scrollRef = (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useRef"])(null);
    const addLog = (msg, type = 'info')=>{
        setLogs((prev)=>[
                ...prev,
                {
                    msg,
                    type
                }
            ]);
    };
    (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["useEffect"])(()=>{
        if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }, [
        logs
    ]);
    const handleFetch = async ()=>{
        setIsLoading(true);
        setLogs([]);
        setStatus('idle');
        const BASE_URL = 'http://localhost:8081';
        const TARGET_URL = `${BASE_URL}${selectedEndpoint.path}`;
        try {
            addLog(`🚀 Initializing Client (${useRealNetwork ? 'REAL NETWORK' : 'SIMULATION'})...`, 'info');
            addLog(`📡 GET ${selectedEndpoint.path}`, 'info');
            addLog(`💵 Price: ${selectedEndpoint.price} sats`, 'info');
            let invoice = "";
            let macaroon = "";
            // --- STEP A: INITIAL REQUEST ---
            try {
                if (useRealNetwork) {
                    const realClient = new RealSatGateClient();
                    await realClient.get(TARGET_URL);
                } else {
                    const mockClient = new MockSatGateClient();
                    await mockClient.get();
                }
                addLog("⚠️ Endpoint is already open (No 402 received).", 'warn');
                setStatus('success');
                return;
            } catch (err) {
                if (err.status === 402) {
                    addLog('🛑 402 Payment Required received', 'warn');
                    setStatus('blocked');
                    const authHeader = err.headers.get('www-authenticate') || err.headers.get('WWW-Authenticate');
                    if (useRealNetwork && !authHeader) {
                        console.error("Full Headers:", err.headers);
                        throw new Error("No L402 header found from backend.");
                    }
                    addLog('⚡ L402 Header detected. Parsing invoice...', 'info');
                    if (useRealNetwork) {
                        const macMatch = authHeader.match(/macaroon="([^"]+)"/);
                        const invMatch = authHeader.match(/invoice="([^"]+)"/);
                        if (macMatch && invMatch) {
                            macaroon = macMatch[1];
                            invoice = invMatch[1];
                            // Show truncated invoice
                            addLog(`📜 Invoice: ${invoice.substring(0, 20)}...${invoice.substring(invoice.length - 10)}`, 'info');
                        } else {
                            throw new Error(`Invalid header format: ${authHeader}`);
                        }
                    } else {
                        invoice = "lnbc10u1p3qj...xyz";
                        addLog(`📜 Invoice: ${invoice}`, 'info');
                        await new Promise((r)=>setTimeout(r, 800));
                    }
                } else {
                    throw err;
                }
            }
            // --- STEP B: PAYMENT ---
            setStatus('paying');
            let preimage = "";
            if (useRealNetwork) {
                addLog(`💸 Launching WebLN (Alby) to pay ${selectedEndpoint.price} sats...`, 'info');
                // @ts-ignore
                if (typeof window.webln === 'undefined') {
                    throw new Error("WebLN not found. Please install Alby extension.");
                }
                // @ts-ignore
                await window.webln.enable();
                // @ts-ignore
                const payment = await window.webln.sendPayment(invoice);
                preimage = payment.preimage;
                addLog(`✅ Payment Sent! Preimage: ${preimage.substring(0, 10)}...`, 'success');
            } else {
                addLog(`💸 Paying Invoice (${selectedEndpoint.price} sats)...`, 'info');
                await new Promise((r)=>setTimeout(r, 1500));
                preimage = "mock_preimage_123";
                addLog('✅ Payment Confirmed. Preimage secured.', 'success');
            }
            // --- STEP C: RETRY WITH AUTH DISCOVERY ---
            addLog('🔄 Retrying request with L402 Token...', 'info');
            if (useRealNetwork) {
                const realClient = new RealSatGateClient();
                const candidates = [
                    `LSAT ${macaroon}:${preimage}`,
                    `L402 ${macaroon}:${preimage}`,
                    `LSAT ${btoa(macaroon + ':' + preimage)}`,
                    `L402 ${btoa(macaroon + ':' + preimage)}`
                ];
                let success = false;
                let lastError;
                for (const token of candidates){
                    try {
                        const finalRes = await realClient.get(TARGET_URL, token);
                        addLog('✅ 200 OK: Request Authorized.', 'success');
                        addLog(`📦 Payload: ${JSON.stringify(finalRes)}`, 'success');
                        success = true;
                        break;
                    } catch (e) {
                        if (e.status === 402) {
                            lastError = e;
                            continue;
                        } else {
                            throw e;
                        }
                    }
                }
                if (!success) {
                    throw new Error(`All Auth formats failed. Last error: ${lastError?.message}`);
                }
            } else {
                await new Promise((r)=>setTimeout(r, 800));
                addLog('✅ 200 OK: Request Authorized.', 'success');
                addLog('📦 Payload: { "market_sentiment": "bullish", "confidence": 0.98 }', 'success');
            }
            setStatus('success');
        } catch (err) {
            setStatus('blocked');
            addLog(`❌ Error: ${err.message || err}`, 'error');
        } finally{
            setIsLoading(false);
        }
    };
    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        className: "min-h-screen bg-black text-gray-100 font-sans flex flex-col items-center py-12 px-4",
        children: [
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                className: "w-full max-w-3xl mb-8 flex items-center justify-between",
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$client$2f$app$2d$dir$2f$link$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["default"], {
                        href: "/",
                        className: "text-gray-500 hover:text-white flex items-center gap-2 transition",
                        children: [
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$arrow$2d$left$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__ArrowLeft$3e$__["ArrowLeft"], {
                                size: 18
                            }, void 0, false, {
                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                lineNumber: 214,
                                columnNumber: 11
                            }, this),
                            " Back to Home"
                        ]
                    }, void 0, true, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 213,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("h1", {
                        className: "text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400",
                        children: "⚡ SatGate Playground"
                    }, void 0, false, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 216,
                        columnNumber: 9
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                lineNumber: 212,
                columnNumber: 7
            }, this),
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                className: "w-full max-w-3xl bg-gray-900 rounded-xl border border-gray-800 shadow-2xl overflow-hidden relative",
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        className: `absolute top-0 left-0 w-full h-1 bg-gradient-to-r transition-all duration-500
          ${status === 'idle' ? 'from-gray-800 to-gray-800' : ''}
          ${status === 'blocked' ? 'from-red-500 via-orange-500 to-red-500 animate-pulse' : ''}
          ${status === 'paying' ? 'from-purple-500 via-cyan-500 to-purple-500 animate-pulse' : ''}
          ${status === 'success' ? 'from-green-500 to-emerald-400' : ''}
        `
                    }, void 0, false, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 225,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        className: "p-4 border-b border-gray-800 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-gray-900/50 backdrop-blur",
                        children: [
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                className: "flex flex-col sm:flex-row items-start sm:items-center gap-3 w-full sm:w-auto",
                                children: [
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("button", {
                                        onClick: ()=>setUseRealNetwork(!useRealNetwork),
                                        className: `flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-bold transition-all border
                ${useRealNetwork ? 'bg-purple-900/30 border-purple-500 text-purple-300' : 'bg-gray-800 border-gray-600 text-gray-400'}`,
                                        children: [
                                            useRealNetwork ? /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wifi$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Wifi$3e$__["Wifi"], {
                                                size: 14
                                            }, void 0, false, {
                                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                                lineNumber: 245,
                                                columnNumber: 35
                                            }, this) : /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wifi$2d$off$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__WifiOff$3e$__["WifiOff"], {
                                                size: 14
                                            }, void 0, false, {
                                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                                lineNumber: 245,
                                                columnNumber: 56
                                            }, this),
                                            useRealNetwork ? "REAL NETWORK" : "SIMULATION"
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                        lineNumber: 238,
                                        columnNumber: 13
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "flex items-center gap-2",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                                                className: "px-2 py-1 bg-emerald-900/50 text-emerald-300 text-xs font-bold rounded border border-emerald-800",
                                                children: "GET"
                                            }, void 0, false, {
                                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                                lineNumber: 251,
                                                columnNumber: 15
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("select", {
                                                value: selectedEndpoint.path,
                                                onChange: (e)=>{
                                                    const ep = ENDPOINTS.find((ep)=>ep.path === e.target.value);
                                                    if (ep) setSelectedEndpoint(ep);
                                                },
                                                className: "bg-gray-800 text-gray-300 border border-gray-700 rounded-lg px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-purple-500",
                                                children: ENDPOINTS.map((ep)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("option", {
                                                        value: ep.path,
                                                        children: ep.label
                                                    }, ep.path, false, {
                                                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                                        lineNumber: 261,
                                                        columnNumber: 19
                                                    }, this))
                                            }, void 0, false, {
                                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                                lineNumber: 252,
                                                columnNumber: 15
                                            }, this)
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                        lineNumber: 250,
                                        columnNumber: 13
                                    }, this)
                                ]
                            }, void 0, true, {
                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                lineNumber: 236,
                                columnNumber: 11
                            }, this),
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("button", {
                                onClick: handleFetch,
                                disabled: isLoading,
                                className: `
              flex items-center gap-2 px-6 py-2 rounded-lg font-bold text-sm transition-all
              ${isLoading ? 'bg-gray-800 text-gray-500 cursor-not-allowed' : 'bg-white text-black hover:bg-gray-200'}
            `,
                                children: isLoading ? 'Processing...' : /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["Fragment"], {
                                    children: [
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$play$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Play$3e$__["Play"], {
                                            size: 16,
                                            fill: "black"
                                        }, void 0, false, {
                                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                            lineNumber: 276,
                                            columnNumber: 46
                                        }, this),
                                        " Run Request"
                                    ]
                                }, void 0, true)
                            }, void 0, false, {
                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                lineNumber: 268,
                                columnNumber: 11
                            }, this)
                        ]
                    }, void 0, true, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 233,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        ref: scrollRef,
                        className: "h-96 bg-black p-6 font-mono text-sm overflow-y-auto space-y-2",
                        children: [
                            logs.length === 0 && /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                className: "text-gray-600 italic",
                                children: useRealNetwork ? "⚠️ REAL MODE: Ensure Aperture is running on localhost:8081 and Alby is installed." : "Ready to simulate... Click 'Run Request' to start."
                            }, void 0, false, {
                                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                lineNumber: 283,
                                columnNumber: 17
                            }, this),
                            logs.map((log, i)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                    className: `flex gap-3 ${log.type === 'error' ? 'text-red-400' : log.type === 'success' ? 'text-green-400' : log.type === 'warn' ? 'text-yellow-400' : 'text-gray-300'}`,
                                    children: [
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                                            className: "opacity-30",
                                            children: [
                                                "[",
                                                new Date().toLocaleTimeString(),
                                                "]"
                                            ]
                                        }, void 0, true, {
                                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                            lineNumber: 295,
                                            columnNumber: 17
                                        }, this),
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                                            children: log.msg
                                        }, void 0, false, {
                                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                            lineNumber: 296,
                                            columnNumber: 17
                                        }, this)
                                    ]
                                }, i, true, {
                                    fileName: "[project]/satgate-landing/app/playground/page.tsx",
                                    lineNumber: 290,
                                    columnNumber: 17
                                }, this))
                        ]
                    }, void 0, true, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 281,
                        columnNumber: 9
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                lineNumber: 222,
                columnNumber: 7
            }, this),
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                className: "w-full max-w-3xl mt-8 grid grid-cols-3 gap-4",
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(StatusStep, {
                        active: status === 'blocked',
                        completed: status === 'paying' || status === 'success',
                        icon: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$shield$2d$alert$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__ShieldAlert$3e$__["ShieldAlert"], {
                            size: 20
                        }, void 0, false, {
                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                            lineNumber: 304,
                            columnNumber: 113
                        }, void 0),
                        label: "1. 402 Blocked"
                    }, void 0, false, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 304,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(StatusStep, {
                        active: status === 'paying',
                        completed: status === 'success',
                        icon: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$zap$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__Zap$3e$__["Zap"], {
                            size: 20
                        }, void 0, false, {
                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                            lineNumber: 305,
                            columnNumber: 89
                        }, void 0),
                        label: "2. Lightning Payment"
                    }, void 0, false, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 305,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(StatusStep, {
                        active: status === 'success',
                        completed: status === 'success',
                        icon: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$circle$2d$check$2d$big$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__$3c$export__default__as__CheckCircle$3e$__["CheckCircle"], {
                            size: 20
                        }, void 0, false, {
                            fileName: "[project]/satgate-landing/app/playground/page.tsx",
                            lineNumber: 306,
                            columnNumber: 90
                        }, void 0),
                        label: "3. Data Unlocked"
                    }, void 0, false, {
                        fileName: "[project]/satgate-landing/app/playground/page.tsx",
                        lineNumber: 306,
                        columnNumber: 9
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                lineNumber: 303,
                columnNumber: 7
            }, this)
        ]
    }, void 0, true, {
        fileName: "[project]/satgate-landing/app/playground/page.tsx",
        lineNumber: 209,
        columnNumber: 5
    }, this);
}
const StatusStep = ({ active, completed, icon, label })=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        className: `
    p-4 rounded-xl border flex items-center justify-center gap-3 transition-all duration-500
    ${active ? 'bg-gray-800 border-white text-white scale-105 shadow-lg shadow-purple-500/20' : ''}
    ${completed ? 'bg-gray-900 border-green-900 text-green-500' : ''}
    ${!active && !completed ? 'bg-black border-gray-800 text-gray-600' : ''}
  `,
        children: [
            icon,
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$satgate$2d$landing$2f$node_modules$2f$next$2f$dist$2f$server$2f$route$2d$modules$2f$app$2d$page$2f$vendored$2f$ssr$2f$react$2d$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$ssr$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                className: "font-bold",
                children: label
            }, void 0, false, {
                fileName: "[project]/satgate-landing/app/playground/page.tsx",
                lineNumber: 321,
                columnNumber: 5
            }, ("TURBOPACK compile-time value", void 0))
        ]
    }, void 0, true, {
        fileName: "[project]/satgate-landing/app/playground/page.tsx",
        lineNumber: 314,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0));
}),
];

//# sourceMappingURL=%5Broot-of-the-server%5D__cd3ecebf._.js.map