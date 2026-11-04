Landing security controls

Design-partner delivery

The POST route accepts JSON only, with an actual streamed-body limit of 16 KiB.
Required fields: name (100 characters), email (254, local part at most 64),
company (200). Optional: role (100), agentCount (50), challenge (4000), apis
(at most 20 strings of at most 100 characters). The hidden website field must
be empty. All submitted HTML values are escaped. Confirmation content is fixed;
only its envelope recipient comes from the application. Notification has no cc.
Provider response/error bodies and submitted personal data are not logged.

Before enabling delivery, configure RESEND_API_KEY and a dedicated shared
Upstash-compatible Redis REST store through UPSTASH_REDIS_REST_URL (HTTPS) and
UPSTASH_REDIS_REST_TOKEN. Do not expose these variables as NEXT_PUBLIC_*.
The store needs EVAL, INCR and EXPIRE permissions. The route atomically reserves
at most three attempts per normalized recipient for 3600 seconds from the first
attempt, including failed mail attempts. Keys contain only a SHA-256 recipient
digest, never a raw address. Missing/unavailable rate storage fails closed; there
is deliberately no per-process fallback. This prevents serverless restarts or
parallel instances from resetting the limit. These are code/configuration
requirements, not a deployment instruction. No deployment is part of this PR.

CSP

proxy.ts generates a fresh 192-bit nonce per page request, replaces untrusted
request CSP headers, passes the policy to Next and returns the same policy to
the browser. Next applies the nonce to its bootstrap and streamed hydration
scripts. RootLayout uses connection() so HTML is rendered at request time;
HTML responses are private/no-store. This trades static-page caching for nonce
correctness. JSON-LD is non-executable data. Script attributes, eval, objects,
framing and cross-origin form posts are disallowed. Inline style attributes
remain allowed for existing React pages. The allowlist retains the existing
live Protect/Pay API origin; no real API calls are made in the browser gate.

Local checks

npm ci
npm run lint
npm run test:design-partner
npm run build
npx playwright install chromium
npm start -- --hostname 127.0.0.1 --port 3114

From a second shell in this directory: npm run test:csp
The browser gate discovers every page.tsx plus retired demo redirect URLs. It
blocks external requests and non-GET requests, asserts a successful local page
response, a fresh nonce on repeat fetches, matching hydration-script nonces,
and zero CSP console errors/securitypolicyviolation events. Route tests replace
all HTTP calls, including Redis and Resend. They cannot send email.
