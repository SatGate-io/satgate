import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, Shield, AlertTriangle, Lock, Eye, Target, Layers, CheckCircle } from 'lucide-react';

export const metadata = {
  title: 'Can Adversaries Game Your Economic Firewall? - SatGate Blog',
  description: 'Can attackers game an economic firewall? See four adversarial cost-control attacks and the cryptographic defenses that stop them.',
  openGraph: {
    title: 'Can Adversaries Game Your Economic Firewall?',
    description: 'The emerging threat landscape for AI agent cost governance — four attack vectors and the cryptographic defenses that stop them.',
    url: 'https://satgate.io/blog/can-adversaries-game-your-economic-firewall',
    type: 'article',
    publishedTime: '2026-03-23T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Can Adversaries Game Your Economic Firewall?',
    description: 'The emerging threat landscape for AI agent cost governance — four attack vectors and the cryptographic defenses that stop them.',
  },
  keywords: ['economic firewall security', 'AI agent adversarial attacks', 'macaroon tokens', 'cryptographic enforcement', 'budget jailbreak', 'AI cost governance', 'prompt injection defense', 'MCP security', 'agent economy', 'economic exfiltration'],
  alternates: { canonical: 'https://satgate.io/blog/can-adversaries-game-your-economic-firewall' },
};

export default function AdversarialBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Can Adversaries Game Your Economic Firewall?',
    description: 'Four adversarial cost-control attacks against AI agent economic firewalls and the cryptographic defenses that stop them.',
    url: 'https://satgate.io/blog/can-adversaries-game-your-economic-firewall',
    datePublished: '2026-03-23',
    dateModified: '2026-05-02',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'economic firewall security' },
      { '@type': 'Thing', name: 'adversarial AI agent attacks' },
      { '@type': 'Thing', name: 'cryptographic budget enforcement' },
      { '@type': 'Thing', name: 'macaroon capability tokens' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Can attackers game an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Attackers can try to game weak economic controls through prompt injection, tool confusion, budget spreading, or token misuse. A well-designed economic firewall resists this by enforcing policy below the agent layer in the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'What makes an economic firewall resistant to adversarial AI?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Adversarial resistance comes from per-tool cost attribution, cryptographic capability tokens, non-escalatable caveats, revocation, audit trails, and fail-closed enforcement before upstream calls execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are macaroons useful for economic firewall security?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Macaroons let teams encode scope, expiry, budget, delegation, and revocation constraints directly into credentials. Child tokens can only add stricter caveats, never expand authority.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono">Security</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">Economic Firewall</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Adversarial AI</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">Macaroons</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">Can Adversaries Game Your Economic Firewall?</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            The Emerging Threat Landscape for AI Agent Cost Governance
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 23, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 14 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          {/* Introduction */}
          <p className="text-gray-300 text-lg leading-relaxed">
            Economic firewalls are having a moment. As organizations deploy autonomous AI agents that make real API calls with real costs, the industry has converged on a simple truth: you need a budget enforcer between your agents and your wallet. Rate limits aren&apos;t enough. API keys aren&apos;t enough. You need something that understands cost, delegates authority, and fails closed.
          </p>
          <p className="text-gray-300 leading-relaxed">
            But here&apos;s the question nobody&apos;s asking loudly enough: <strong>what happens when the threat isn&apos;t a runaway agent — it&apos;s an adversary?</strong>
          </p>
          <p className="text-gray-300 leading-relaxed">
            We built economic firewalls for accidents. A coding agent that gets stuck in a loop and burns through $400 of GPT-4 calls. A data pipeline agent that retries indefinitely against a paid API. These are real problems, and economic firewalls solve them elegantly. Budget exceeded, request denied, crisis averted.
          </p>
          <p className="text-gray-300 leading-relaxed">
            That&apos;s the easy case. The hard case is an attacker who understands your controls and deliberately engineers around them.
          </p>

          {/* The Assumption */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Assumption We Need to Challenge</h2>

          <p className="text-gray-300 leading-relaxed">
            Every economic firewall makes an implicit assumption: the request metadata is trustworthy. The agent says it&apos;s making a text completion call, so we price it as a text completion call. The agent presents its token, so we check the token&apos;s budget. The agent stays under its limit, so we let it through.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This works when agents are honest — or at least predictably broken. It does not work when an adversary is actively manipulating the agent, the request, or the cost perception layer between them.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Adversarial AI changes the calculus. Prompt injection, tool confusion, multi-agent coordination attacks — these aren&apos;t theoretical. They&apos;re documented, reproducible, and getting more sophisticated. If your economic firewall only defends against accidents, you&apos;ve built a smoke detector that doesn&apos;t work during arson.
          </p>
          <p className="text-gray-300 leading-relaxed">
            The question isn&apos;t whether your firewall handles budget limits. It&apos;s whether your firewall&apos;s enforcement is architecturally resistant to manipulation. That distinction — between policy enforcement and cryptographic enforcement — is the entire ballgame.
          </p>

          {/* Attack Vector 1 */}
          <div className="my-12 p-6 bg-gradient-to-r from-red-900/20 to-red-800/10 border border-red-500/20 rounded-xl">
            <div className="flex items-center gap-3 mb-4">
              <Target className="text-red-400" size={28} />
              <h2 className="text-2xl font-bold text-white m-0">Attack Vector 1: Cost-Category Manipulation</h2>
            </div>
            <p className="text-red-200 text-lg font-medium mb-0">Trick the agent into misclassifying expensive operations as cheap ones.</p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            <strong>The attack:</strong> An adversary uses prompt injection to trick an agent into misclassifying a high-cost operation as a low-cost one. The agent believes it&apos;s making a simple text query. In reality, it&apos;s triggering an image generation call, a fine-tuning job, or an expensive third-party API.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This isn&apos;t far-fetched. Prompt injection can alter an agent&apos;s understanding of what tool it&apos;s calling, what parameters it&apos;s passing, or what category of work it&apos;s performing. If your cost governance relies on the agent&apos;s self-reported action type, you&apos;re trusting the thing that just got compromised.
          </p>

          <div className="my-8 p-6 bg-green-900/10 border border-green-500/20 rounded-xl">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="text-green-400" size={20} />
              <h3 className="text-lg font-bold text-green-300 m-0">The Defense: Per-Tool Cost Attribution</h3>
            </div>
            <p className="text-gray-300 leading-relaxed mb-0">
              In an MCP-based architecture, the economic firewall doesn&apos;t ask the agent what it thinks it&apos;s doing — it inspects the actual tool call. The firewall sits between the agent and the tool server. It sees the real method name, the real parameters, the real cost profile. The agent&apos;s confused perception is irrelevant because enforcement happens below the agent&apos;s abstraction layer.
            </p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            This is the difference between a security guard who asks &quot;what&apos;s in the bag?&quot; and an X-ray machine. One relies on the answer. The other doesn&apos;t need to ask.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Per-tool attribution also means you can set different budget thresholds per tool category. Text completions get one budget. Image generation gets another. Code execution gets a third. Even if an attacker manages to route a request to the wrong tool, the tool-level budget catches it.
          </p>

          {/* Attack Vector 2 */}
          <div className="my-12 p-6 bg-gradient-to-r from-orange-900/20 to-orange-800/10 border border-orange-500/20 rounded-xl">
            <div className="flex items-center gap-3 mb-4">
              <Layers className="text-orange-400" size={28} />
              <h2 className="text-2xl font-bold text-white m-0">Attack Vector 2: Budget Envelope Spreading</h2>
            </div>
            <p className="text-orange-200 text-lg font-medium mb-0">Distribute spend across many agents to stay under individual limits.</p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            <strong>The attack:</strong> Instead of one compromised agent blowing through a single budget, the adversary compromises — or simply provisions — multiple agents, each with its own modest budget. Individually, every agent stays well within its limits. Collectively, they drain ten or fifty times what any single budget would allow.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This is the distributed denial-of-wallet attack. Each agent looks compliant in isolation. The pattern only emerges when you correlate spend across the fleet.
          </p>

          <div className="my-8 p-6 bg-green-900/10 border border-green-500/20 rounded-xl">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="text-green-400" size={20} />
              <h3 className="text-lg font-bold text-green-300 m-0">The Defense: Delegation Hierarchies + Governance Graph</h3>
            </div>
            <p className="text-gray-300 leading-relaxed">
              First, <strong>delegation hierarchies with budget carving</strong>. When a parent agent delegates authority to child agents, the children&apos;s budgets are carved from the parent&apos;s total allocation — not created independently. If a parent has $100 and delegates $20 to each of five children, the total possible spend is still $100. You can&apos;t create budget out of thin air by spawning more agents. The math is subtractive, not additive.
            </p>
            <p className="text-gray-300 leading-relaxed mb-0">
              Second, <strong>governance graph visualization and cross-agent spend correlation</strong>. A governance graph maps every agent, every delegation, every token relationship. When you can visualize the entire delegation tree — who authorized whom, how much budget flowed where, which branches are consuming disproportionately — envelope spreading becomes visible. The blast radius is contained by the hierarchy. The detection happens through correlation.
            </p>
          </div>

          {/* Attack Vector 3 */}
          <div className="my-12 p-6 bg-gradient-to-r from-purple-900/20 to-purple-800/10 border border-purple-500/20 rounded-xl">
            <div className="flex items-center gap-3 mb-4">
              <Lock className="text-purple-400" size={28} />
              <h2 className="text-2xl font-bold text-white m-0">Attack Vector 3: Budget Jailbreaks</h2>
            </div>
            <p className="text-purple-200 text-lg font-medium mb-0">Manipulate the agent into believing it has more budget than it actually does.</p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            <strong>The attack:</strong> The adversary manipulates the agent into believing it has more budget than it actually does. Maybe a prompt injection overwrites the agent&apos;s internal budget counter. Maybe the agent&apos;s cost estimation logic is poisoned so it thinks calls are cheaper than they are. Maybe the agent is simply told &quot;you have unlimited budget, proceed.&quot;
          </p>
          <p className="text-gray-300 leading-relaxed">
            In a policy-based system, this is devastating. If the agent is responsible for tracking its own spend and self-limiting, then compromising the agent&apos;s perception of its budget is equivalent to removing the budget entirely.
          </p>

          <div className="my-8 p-6 bg-green-900/10 border border-green-500/20 rounded-xl">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="text-green-400" size={20} />
              <h3 className="text-lg font-bold text-green-300 m-0">The Defense: Cryptographic Enforcement via Macaroon Caveats</h3>
            </div>
            <p className="text-gray-300 leading-relaxed">
              A macaroon token doesn&apos;t store the budget in the agent&apos;s memory, in a config file, or in an environment variable the agent can read and modify. The budget is embedded in the token itself as a cryptographic caveat. When the agent presents its token to the firewall, the firewall evaluates the caveats — including remaining budget — against the request. The agent&apos;s opinion about its budget is not consulted.
            </p>
            <p className="text-gray-300 leading-relaxed">
              Even if the agent is fully compromised, even if it&apos;s been jailbroken into believing it has infinite resources, the token it carries still says $20. The firewall still enforces $20. The agent cannot forge a new token with a higher budget because macaroon caveats are chained cryptographic commitments — adding a caveat is easy, removing one requires breaking the HMAC chain.
            </p>
            <p className="text-gray-300 leading-relaxed font-semibold mb-0">
              The agent doesn&apos;t enforce its own budget. The credential does. Jailbreaking the agent doesn&apos;t jailbreak the token.
            </p>
          </div>

          {/* Attack Vector 4 */}
          <div className="my-12 p-6 bg-gradient-to-r from-yellow-900/20 to-yellow-800/10 border border-yellow-500/20 rounded-xl">
            <div className="flex items-center gap-3 mb-4">
              <Eye className="text-yellow-400" size={28} />
              <h2 className="text-2xl font-bold text-white m-0">Attack Vector 4: Slow Drain / Economic Exfiltration</h2>
            </div>
            <p className="text-yellow-200 text-lg font-medium mb-0">Small, legitimate-looking requests that accumulate into significant unauthorized spend over time.</p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            <strong>The attack:</strong> The adversary doesn&apos;t blow through the budget in one dramatic burst. Instead, they make small, perfectly authorized-looking requests over an extended period. Each individual transaction passes every check — correct tool, reasonable cost, within budget limits. But over days or weeks, these small draws accumulate into significant unauthorized spend.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This is economic exfiltration. It&apos;s the AI equivalent of salami slicing. And it&apos;s the hardest attack to detect because every single request, examined in isolation, looks legitimate.
          </p>

          <div className="my-8 p-6 bg-green-900/10 border border-green-500/20 rounded-xl">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="text-green-400" size={20} />
              <h3 className="text-lg font-bold text-green-300 m-0">The Defense: Operational Modes + Temporal Controls</h3>
            </div>
            <p className="text-gray-300 leading-relaxed">
              <strong>Shadow and Observe modes</strong> let you monitor agent spending patterns before you enforce hard limits. Both modes build a baseline of normal behavior. When spending deviates from that baseline — even if every individual request is within policy — the anomaly surfaces.
            </p>
            <p className="text-gray-300 leading-relaxed mb-0">
              <strong>Time-based budget refresh periods</strong> limit cumulative damage. Instead of a single lifetime budget of $500, you set $50 per day with automatic refresh. A slow drain that would take weeks to exhaust a lifetime budget now has to extract value within each refresh window. The economics of patience-based attacks get much worse when the budget resets.
            </p>
          </div>

          {/* Why Cryptographic Enforcement Wins */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Why Cryptographic Enforcement Beats Policy Enforcement</h2>

          <p className="text-gray-300 leading-relaxed">
            Every attack vector above shares a common thread: they exploit the gap between what the system <em>checks</em> and what the system <em>enforces</em>.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Traditional API key management is all-or-nothing. A valid key gets full access. A compromised key means full exposure. You can layer rate limits and monitoring on top, but the key itself carries no constraints. It&apos;s a skeleton key. You&apos;re relying on the lock to be smart, not the key.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Macaroon-based tokens invert this model. The token itself carries its constraints — budget limits, tool restrictions, time bounds, delegation depth. These constraints are cryptographically chained. A child token cannot have more authority than its parent. This isn&apos;t a policy check that can be bypassed. It&apos;s a mathematical guarantee.
          </p>

          <div className="my-8 grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-5 bg-red-900/10 border border-red-500/20 rounded-xl">
              <h3 className="text-lg font-bold text-red-300 mb-3 flex items-center gap-2">
                <AlertTriangle size={18} /> Policy Enforcement
              </h3>
              <p className="text-gray-400 text-sm mb-0">
                &quot;We check your budget in a database before approving the request.&quot; The database can be wrong. The check can be skipped. The logic can be fooled. The enforcement point is software that can have bugs, race conditions, or configuration errors.
              </p>
            </div>
            <div className="p-5 bg-green-900/10 border border-green-500/20 rounded-xl">
              <h3 className="text-lg font-bold text-green-300 mb-3 flex items-center gap-2">
                <Lock size={18} /> Cryptographic Enforcement
              </h3>
              <p className="text-gray-400 text-sm mb-0">
                &quot;Your budget is baked into your credential, and the credential can&apos;t be modified without invalidating it.&quot; The enforcement isn&apos;t in a separate system that can be circumvented — it&apos;s in the token the agent must present. The math doesn&apos;t have configuration errors.
              </p>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            For the CISO evaluating these systems: if the budget enforcement can be bypassed by compromising the agent, it&apos;s not security infrastructure. It&apos;s accounting software with aspirations.
          </p>

          {/* The Defensive Playbook */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Defensive Playbook</h2>

          <p className="text-gray-300 leading-relaxed">
            If you&apos;re building or evaluating an economic firewall for AI agents, here&apos;s what the architecture should include:
          </p>

          <div className="my-8 space-y-4">
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Per-tool cost attribution</h4>
                <p className="text-gray-400 text-sm mb-0">Don&apos;t trust the agent&apos;s description of its own actions. Attribute cost at the tool-call layer, below the agent&apos;s abstraction.</p>
              </div>
            </div>
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Delegation depth limits</h4>
                <p className="text-gray-400 text-sm mb-0">Cap how many layers deep a token can be delegated. Each layer is a potential point of compromise.</p>
              </div>
            </div>
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Budget refresh periods</h4>
                <p className="text-gray-400 text-sm mb-0">Time-bound budgets instead of lifetime allocations. Daily or hourly refresh windows limit cumulative damage from slow-drain attacks.</p>
              </div>
            </div>
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Cross-agent correlation via governance graph</h4>
                <p className="text-gray-400 text-sm mb-0">Visualize the entire delegation tree. Correlate spend across sibling agents, across branches, across time.</p>
              </div>
            </div>
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Fail-closed enforcement</h4>
                <p className="text-gray-400 text-sm mb-0">When the firewall can&apos;t verify a token, can&apos;t reach the budget ledger, or encounters any ambiguity — deny the request. Fail-open is a vulnerability.</p>
              </div>
            </div>
            <div className="flex items-start gap-3 p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
              <CheckCircle className="text-green-400 mt-1 flex-shrink-0" size={20} />
              <div>
                <h4 className="text-white font-bold mb-1">Shadow mode for anomaly detection</h4>
                <p className="text-gray-400 text-sm mb-0">Run in observation mode before enforcement mode. Build a behavioral baseline. Detect anomalies while they&apos;re still data points, not incidents.</p>
              </div>
            </div>
          </div>

          {/* The Bottom Line */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Bottom Line</h2>

          <p className="text-gray-300 leading-relaxed">
            Economic firewalls started as cost controls. That&apos;s fine — cost control is valuable. But the architecture you choose for cost control determines whether you&apos;ve also built a security boundary or just a dashboard with a kill switch.
          </p>
          <p className="text-gray-300 leading-relaxed">
            The adversarial threat to AI agent infrastructure is real and growing. Prompt injection, multi-agent coordination attacks, and economic exfiltration are not tomorrow&apos;s problems. They&apos;re today&apos;s problems that most organizations haven&apos;t tested for yet.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Cryptographic enforcement — tokens with embedded, non-escalatable constraints — is the foundation that makes economic firewalls defensible against intentional exploitation. Everything else is defense in depth on top of that foundation.
          </p>

          <div className="my-8 p-6 bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-500/20 rounded-xl">
            <p className="text-xl text-white font-bold mb-0">
              Build the firewall that works when someone&apos;s trying to break it. That&apos;s the only kind worth having.
            </p>
          </div>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-red-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">Economic firewall adversarial security questions</h2>
            <div className="space-y-5">
              {[
                ['Can attackers game an economic firewall?', 'Attackers can try to game weak economic controls through prompt injection, tool confusion, budget spreading, or token misuse. A well-designed economic firewall resists this by enforcing policy below the agent layer in the request path.'],
                ['What makes an economic firewall resistant to adversarial AI?', 'Adversarial resistance comes from per-tool cost attribution, cryptographic capability tokens, non-escalatable caveats, revocation, audit trails, and fail-closed enforcement before upstream calls execute.'],
                ['Why are macaroons useful for economic firewall security?', 'Macaroons let teams encode scope, expiry, budget, delegation, and revocation constraints directly into credentials. Child tokens can only add stricter caveats, never expand authority.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>
        </article>

        {/* CTA */}
        <section className="mt-16 bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-800/30 rounded-xl p-8 text-center">
          <h2 className="text-2xl font-bold mb-3">Ready to Secure Your Agent Economy?</h2>
          <p className="text-gray-400 mb-6">
            SatGate provides cryptographic budget enforcement for AI agents — not just cost tracking.
          </p>
          <div className="flex flex-wrap justify-center gap-4">
            <Link
              href="/govern"
              className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition"
            >
              See How It Works
            </Link>
            <Link
              href="/design-partners"
              className="inline-flex items-center gap-2 bg-transparent text-white border border-gray-600 px-6 py-3 rounded-lg font-bold hover:border-purple-500 transition"
            >
              Become a Design Partner
            </Link>
          </div>
        </section>

        {/* Navigation */}
        <div className="mt-8 pt-8 border-t border-gray-800">
          <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
            <ArrowLeft size={18} /> Back to all posts
          </Link>
        </div>
      </div>
    </div>
  );
}
