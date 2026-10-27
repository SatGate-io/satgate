import Link from 'next/link';
import { ArrowRight } from 'lucide-react';

// Four recordings from production, with captions and the receipt check.
// Rendered by the home page and by /sandbox so both show the same files and the same words.
export default function ProductionRecordings({ heading }: { heading: string }) {
  return (
    <section id="see-it-live" className="py-16 px-6 border-b border-gray-800 scroll-mt-20">
      <div className="max-w-4xl mx-auto text-center">
        <h2 className="text-3xl font-bold mb-3">{heading}</h2>
        <p className="text-gray-400 mb-10 max-w-2xl mx-auto">Four recordings from production, with narration.</p>
        <h3 className="text-xl font-bold mb-2">Your agents&apos; API calls: a budget they can&apos;t overspend</h3>
        <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Connect an API, set a route to Control, and mint a token with a 100-credit budget. Without the token the route returns 401; with it, 200. A demo agent spends the budget at 10 credits a call until SatGate blocks its next call. Then the open-source verifier checks the signed receipt for that refusal.</p>
        <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
          <video
            controls
            preload="none"
            poster="/satgate-demo-http-poster.jpg"
            className="w-full"
            playsInline
            aria-label="Narrated demo: a token with a 100-credit budget is minted for one API route, a demo agent spends it 10 credits per call until SatGate blocks its next call, and the signed receipts are checked with the open-source verifier"
          >
            <source src="/satgate-demo-http.mp4" type="video/mp4" />
            Your browser does not support the video tag.
          </video>
        </div>
        <p className="text-xs text-gray-500 mt-3">The agent in the recording is the dashboard&apos;s built-in demo agent.</p>

        <h3 className="text-xl font-bold mt-16 mb-2">Your agents&apos; tool calls: only the tools you allow, within budget</h3>
        <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Add an MCP server, mint a token with a 3-credit budget that can call one tool, and copy its config into Claude Code, Cursor, Hermes, OpenClaw or any other MCP client. A tool outside the list is refused before it runs. Once the budget is spent, the next call is refused too. The open-source verifier checks the receipts.</p>
        <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
          <video
            controls
            preload="none"
            poster="/satgate-demo-mcp-poster.jpg"
            className="w-full"
            playsInline
            aria-label="Narrated demo: an MCP token with a 3-credit budget and one allowed tool; a call to another tool is refused before it runs, the budget runs out and the next call is refused, and the signed receipts are checked with the open-source verifier"
          >
            <source src="/satgate-demo.mp4" type="video/mp4" />
            Your browser does not support the video tag.
          </video>
        </div>
        <p className="text-xs text-gray-500 mt-3">The agent in the recording is a scripted MCP client using the public <code className="text-gray-400">satgate-mcp-bridge</code> npm package.</p>

        <h3 className="text-xl font-bold mt-16 mb-2">Your agents on a real brokerage account: each with its own limits</h3>
        <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Two agents trade one real brokerage account through SatGate, each with its own token. The trading agent can spend $200 a day, needs the owner&apos;s OK for any order over $4, and can only use its own account. It&apos;s refused on the owner&apos;s main account, buys $1 of Bitcoin, and is refused a $250 order the account could pay for. A $5 order waits until the owner approves it from the link in the email, then fills. The owner revokes the trading agent&apos;s token. Its next connection is refused, and the savings agent keeps trading. The refusals, the hold and each order get a signed receipt, and the open-source verifier checks them.</p>
        <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
          <video
            controls
            preload="none"
            poster="/satgate-demo-trading-poster.jpg"
            className="w-full"
            playsInline
            aria-label="Narrated demo: two agents with their own tokens trade one real brokerage account; one is refused on the owner's main account and over its $200 daily limit, a $5 order waits for the owner's approval, the owner revokes one agent while the other keeps trading, and the signed receipts pass the open-source verifier"
          >
            <source src="/satgate-demo-trading.mp4" type="video/mp4" />
            Your browser does not support the video tag.
          </video>
        </div>
        <div className="mt-6">
          <Link href="/trading" className="inline-flex items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 px-8 py-3 font-bold text-white shadow-lg shadow-purple-500/20 transition hover:from-purple-500 hover:to-cyan-500">
            Set up my agent&apos;s limits <ArrowRight size={16} />
          </Link>
        </div>
        <p className="text-xs text-gray-500 mt-3">The agents in the recording are scripted MCP clients. The orders are real: three Bitcoin buys, $7 in total. Account numbers are masked.</p>

        <h3 id="usdc-recording" className="text-xl font-bold mt-16 mb-2 scroll-mt-20">External agents: pay before they get in</h3>
        <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Set a route to Admit and choose how external agents pay: USDC on Base, sats over Lightning, or both, each with its own price. An agent with no wallet gets 402 Payment Required and never reaches your API. In the recording, a second agent sends one cent in USDC, the payment settles on Base, and the call goes through. Sending the same payment again is refused. With USDC, each payment buys one request. On Lightning you choose how many requests one payment buys, and on sat-priced routes the price can rise under load, up to a ceiling you set; unpaid 402s don&apos;t raise it. Paying never gets an agent past your access rules. SatGate signs a receipt when it refuses an agent for a missing payment.</p>
        <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-yellow-500/10">
          <video
            controls
            preload="none"
            poster="/satgate-admit-poster.jpg"
            className="w-full"
            playsInline
            aria-label="Narrated demo: the owner chooses USDC, sats, or both for a paid route; an external agent with no wallet is refused with 402 Payment Required, a second external agent sends one cent in USDC on Base and is allowed through, and the signed receipts pass the open-source verifier"
          >
            <source src="/satgate-admit-demo.mp4" type="video/mp4" />
            Your browser does not support the video tag.
          </video>
        </div>
        <p className="text-xs text-gray-500 mt-3">The agents in the recording are scripted HTTP clients. The USDC payment is real: one cent, settled on Base.</p>
        <div className="mt-6 text-left max-w-2xl mx-auto rounded-lg border border-gray-800 bg-black/60 p-4">
          <p className="text-sm text-gray-300 mb-3">
            Check the receipts from the video yourself. The paid call:{' '}
            <a href="https://api.satgate.io/v1/evidence/evid_e-suLq8DGymE6LvIJ2D1mTN95Oavy_4E" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">live receipt</a>{' '}
            or{' '}
            <a href="/evidence/admit-usdc-paid-20261005.json" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">downloaded copy</a>. A call refused for no payment:{' '}
            <a href="https://api.satgate.io/v1/evidence/evid_XIZBDiyJjM7dPU6Y0TZF7ToRkVzRrRBi" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">live receipt</a>{' '}
            or{' '}
            <a href="/evidence/admit-payment-refusal-20261005.json" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">downloaded copy</a>. Then run the{' '}
            <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">open-source verifier</a>{' '}
            against SatGate&apos;s public key:
          </p>
          <pre className="text-xs text-gray-400 font-mono whitespace-pre-wrap break-all">python3 tools/verify_evidence_pack.py https://api.satgate.io/v1/evidence/evid_e-suLq8DGymE6LvIJ2D1mTN95Oavy_4E --jwks-url https://api.satgate.io/.well-known/jwks.json --require-trusted-issuer</pre>
          <p className="text-xs text-gray-500 mt-3">
            <Link href="/verify-evidence-pack" className="hover:text-gray-300 underline underline-offset-2">What the verifier checks</Link>
          </p>
        </div>
      </div>
    </section>
  );
}
