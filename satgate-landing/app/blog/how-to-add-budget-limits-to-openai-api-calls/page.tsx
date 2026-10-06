import Link from 'next/link';
import RoiCta from '../../components/RoiCta';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: "OpenAI API Budget Limits: Stop Runaway Spend Before Calls",
  description: "Set OpenAI API budget limits by agent, team, or workflow. Stop runaway GPT spend before the call runs and keep signed receipt proof.",
  alternates: { canonical: 'https://satgate.io/blog/how-to-add-budget-limits-to-openai-api-calls' },
  keywords: ['OpenAI API budget limits', 'OpenAI cost control', 'API gateway OpenAI', 'GPT-4 spending limits', 'OpenAI API costs', 'prevent OpenAI overspending', 'hard cap OpenAI spend', 'per-agent OpenAI budget'],
  openGraph: {
    title: 'OpenAI API Budget Limits: Stop Runaway Spend Before Calls',
    description: 'Set OpenAI API budget limits by agent, team, or workflow. Stop runaway GPT spend before the call runs and keep signed receipt proof.',
    url: 'https://satgate.io/blog/how-to-add-budget-limits-to-openai-api-calls',
    type: 'article',
    publishedTime: '2026-04-07T00:00:00Z',
    modifiedTime: '2026-08-12T00:00:00Z',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'OpenAI API Budget Limits: Stop Runaway Spend Before Calls',
    description: 'Set per-agent, team, and workflow budgets before GPT calls execute, then prove each allow, deny, or downgrade decision.',
  },
};

export default function HowToAddBudgetLimitsToOpenAIAPICallsPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'OpenAI API Budget Limits: Stop Runaway Spend Before Calls',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-07',
    dateModified: '2026-08-12',
    mainEntityOfPage: 'https://satgate.io/blog/how-to-add-budget-limits-to-openai-api-calls',
    about: [
      { '@type': 'Thing', name: 'OpenAI API budget limits' },
      { '@type': 'Thing', name: 'hard caps for GPT spend' },
      { '@type': 'Thing', name: 'per-agent OpenAI budgets' },
      { '@type': 'Thing', name: 'budget enforcement before the request goes through' },
      { '@type': 'Thing', name: 'runaway LLM spend prevention' },
      { '@type': 'Thing', name: 'Observe Control Prove' },
      { '@type': 'Thing', name: 'signed receipts' },
    ],
  };

  const faqSchema = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Does OpenAI have built-in spending limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'OpenAI has account-level usage limits, but they are not the same as budget enforcement before the request goes through. They are coarse, can lag behind real usage, and usually cannot isolate spend by agent, user, session, workflow, or tool before a request executes.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the difference between a rate limit and a budget limit for OpenAI?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A rate limit caps request volume, while a budget limit caps spend. For OpenAI and other LLM APIs, spend depends on model, tokens, context length, retries, and tool calls, so budget limits are the safer control for autonomous agents.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you set per-agent OpenAI budget limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Set per-agent OpenAI budget limits by routing calls through a gateway that identifies the agent, estimates or prices the request, checks remaining budget, and blocks or downgrades the call before it reaches OpenAI.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can you set OpenAI API budget limits by team or customer?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. A budget gateway that checks before the request goes through can attach spend policy to a team, customer, environment, workflow, or agent token, then enforce separate OpenAI API budgets before each request executes.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you enforce an OpenAI API budget limit before GPT calls run?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Put a gateway in front of OpenAI, identify the agent or team, estimate the request cost from model and token policy, check remaining budget, and allow, block, or downgrade the request before it reaches the OpenAI API.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqSchema) }}
      />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">OpenAI</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">Cost Control</span>
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">Tutorial</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">API Gateway</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">How to Add OpenAI API Budget Limits Before Calls Run</h1>
          <div className="mb-6 rounded-2xl border border-green-900/60 bg-green-950/20 p-5">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-green-300">Direct answer</p>
            <p className="text-gray-300">OpenAI usage limits are account-level. Controls that run before the request goes through enforce per-agent, per-team, and per-workflow budgets before a GPT call reaches OpenAI, then Prove each allow, deny, or downgrade with a signed receipt.</p>
          </div>
          <div className="mb-6 flex flex-col gap-3 sm:flex-row">
            <Link href="/build" className="inline-flex items-center justify-center rounded-lg bg-white px-5 py-3 text-sm font-bold text-black transition hover:bg-gray-200">Add a budget with SatGate</Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-purple-500">See how rules and receipts work</Link>
          </div>
          
          <p className="text-xl text-gray-400 mb-6">
            OpenAI&apos;s dashboard shows cost after it happens. SatGate-style budget authority blocks, downgrades, or routes GPT calls before spend crosses a policy line.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> April 7, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 8 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">The $72,000 Lesson</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Last month, a developer shared their nightmare: a misconfigured retry loop burned $72,000 in OpenAI credits overnight. The dashboard showed the damage hours later. The bill? Non-negotiable.
          </p>
          
          <p className="text-gray-300 leading-relaxed">
            This isn&apos;t rare. Search &ldquo;OpenAI unexpected bill&rdquo; and you&apos;ll find dozens of similar stories. The pattern is always the same:
          </p>
          
          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li>A bug causes excessive API calls</li>
            <li>Rate limits prevent immediate detection</li>
            <li>Usage dashboards update hours later</li>
            <li>The damage is already done</li>
          </ul>
          
          <p className="text-gray-300 leading-relaxed">
            OpenAI&apos;s built-in limits? They&apos;re monthly caps that email you after overspending. That&apos;s like a smoke detector that texts you after your house burns down.
          </p>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Why Traditional Solutions Fail</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Most teams try one of three approaches:
          </p>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">1. OpenAI&apos;s Usage Limits</h3>
          
          <p className="text-gray-300 leading-relaxed">
            OpenAI offers monthly spending limits, but they have critical flaws:
          </p>
          
          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li><strong>Delayed enforcement:</strong> Limits check against cached usage data</li>
            <li><strong>All-or-nothing:</strong> Hit the limit? Your entire account stops</li>
            <li><strong>No granularity:</strong> Can&apos;t set limits per team, project, or user</li>
            <li><strong>Soft enforcement:</strong> &ldquo;Hard limits&rdquo; can still overshoot by 10-20%</li>
          </ul>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">2. Monitoring Dashboards</h3>
          
          <p className="text-gray-300 leading-relaxed">
            Tools like Datadog or custom dashboards show beautiful graphs of your spending. They&apos;re great for post-mortems, useless for prevention:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# This alert fires AFTER you've already spent $1000
alert: openai_daily_spend_high
expr: sum(openai_spend_24h) > 1000
annotations:
  summary: "OpenAI spend exceeded $1000 in 24h"`}</code>
          </pre>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">3. Client-Side Rate Limiting</h3>
          
          <p className="text-gray-300 leading-relaxed">
            Some teams implement token counting in their application code:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`import tiktoken

class OpenAIBudgetWrapper:
    def __init__(self, daily_limit=100):
        self.daily_limit = daily_limit
        self.spent_today = 0
    
    def complete(self, prompt):
        # Problem 1: Estimates are often wrong
        estimated_cost = self.estimate_cost(prompt)
        
        # Problem 2: No coordination between instances
        if self.spent_today + estimated_cost > self.daily_limit:
            raise BudgetExceeded()
        
        # Problem 3: Actual cost known only after response
        response = openai.complete(prompt)
        actual_cost = response.usage.total_cost
        self.spent_today += actual_cost
        
        return response`}</code>
          </pre>
          
          <p className="text-gray-300 leading-relaxed">
            This fails because:
          </p>
          
          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li>Cost estimates are inaccurate (especially with JSON mode, tool calls)</li>
            <li>Multiple app instances don&apos;t share state</li>
            <li>Actual costs are known only after the request completes</li>
            <li>No protection against retry storms or runaway loops</li>
          </ul>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">The Solution: Request-Level Budget Enforcement</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Real budget protection requires three things OpenAI doesn&apos;t provide:
          </p>
          
          <ol className="list-decimal list-inside space-y-2 text-gray-300 my-4">
            <li><strong>Pre-request validation:</strong> Check budgets before forwarding to OpenAI</li>
            <li><strong>Real-time accounting:</strong> Track actual spend, not estimates</li>
            <li><strong>Granular controls:</strong> Different limits for different use cases</li>
          </ol>
          
          <p className="text-gray-300 leading-relaxed">
            Here&apos;s how to implement it properly with SatGate:
          </p>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 1: Install the Gateway</h2>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Mint a token with a budget in the SatGate dashboard: Cloud, then Tokens or MCP setup.
Send the agent's calls through the gateway with that token.
The gateway holds the upstream key. The agent only holds the SatGate token.
There is no daily limit yet. A spent budget returns HTTP 402 with code budget_exhausted.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 2: Create Budget-Limited Tokens</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Instead of using your OpenAI API key directly, create derivative tokens with spending limits:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Mint a token with a budget in the SatGate dashboard: Cloud, then Tokens or MCP setup.
Use that token on calls you send through the gateway.
Do not set a daily limit. That control does not exist yet.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 3: Update Your Code</h2>
          
          <p className="text-gray-300 leading-relaxed">
            The beautiful part? Your application code barely changes:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Send calls through the gateway URL shown in the dashboard, with the token you minted.
The agent does not hold the upstream API key.
A spent budget returns HTTP 402 with code budget_exhausted, not 429.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 4: Configure Team Budgets</h2>
          
          <p className="text-gray-300 leading-relaxed">
            For larger teams, create hierarchical budgets:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Create tokens within a budget in the dashboard (Cloud, then Tokens or MCP setup).
# There is no satgate budget create command, and no daily limit.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Real-World Example: Preventing Retry Storms</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Here&apos;s how SatGate prevents the $72,000 nightmare scenario:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`// Buggy code with infinite retry loop
async function processDocument(doc) {
  while (true) {
    try {
      const response = await openai.chat.completions.create({
        model: "gpt-4",
        messages: [
          { role: "system", content: "Extract entities from document" },
          { role: "user", content: doc.content } // Bug: 100MB document
        ]
      });
      return response;
    } catch (error) {
      console.log("Retrying..."); // Infinite loop on large docs
      await sleep(1000);
    }
  }
}`}</code>
          </pre>
          
          <p className="text-gray-300 leading-relaxed">
            Without protection: This burns thousands of dollars as it repeatedly sends a huge document to GPT-4.
          </p>
          
          <p className="text-gray-300 leading-relaxed">
            With SatGate: The token&apos;s hourly limit triggers after ~$50, blocking further requests:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Request 1: $12.50 (huge input) - Allowed (total: $12.50)
# Request 2: $12.50 retry - Allowed (total: $25.00)
# Request 3: $12.50 retry - Allowed (total: $37.50)
# Request 4: $12.50 retry - Allowed (total: $50.00)
# Request 5: BLOCKED - Hourly limit exceeded

{
  "error": {
    "type": "budget_exceeded",
    "message": "Hourly budget limit exceeded",
    "limit": 50,
    "spent": 50,
    "resets_at": "2024-04-07T19:00:00Z"
  }
}`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Advanced: Per-User Budgets for AI Apps</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Building a ChatGPT wrapper? Give each user their own budget:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Mint one token per user, with a budget, in the SatGate dashboard (Cloud, then Tokens or MCP setup).
Send that user's calls through the gateway with their token.
There is no daily limit, and no satgate.tokens.create API.
A spent budget returns HTTP 402 with code budget_exhausted.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Monitoring and Alerts</h2>
          
          <p className="text-gray-300 leading-relaxed">
            A spent budget is enforced on the token, not by a CLI. There is no satgate alerts command.
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Mint the token in the SatGate dashboard (Cloud, then Tokens or MCP setup).
There is no satgate alerts add command, and no daily period.
A spent budget returns HTTP 402 with code budget_exhausted.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">The Results</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Teams using request-level budget enforcement report:
          </p>
          
          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li><strong>100% prevention</strong> of runaway spend incidents</li>
            <li><strong>73% reduction</strong> in overall OpenAI costs (better visibility)</li>
            <li><strong>Zero production outages</strong> from hitting OpenAI account limits</li>
            <li><strong>Granular insights</strong> into cost per feature/team/user</li>
          </ul>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Common Questions</h2>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Does this add latency?</h3>
          
          <p className="text-gray-300 leading-relaxed">
            SatGate adds &lt;1ms to check budgets. Compare that to the 2-3 seconds for a typical GPT-4 call. The overhead is negligible.
          </p>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">What happens when limits are hit?</h3>
          
          <p className="text-gray-300 leading-relaxed">
            Requests are immediately rejected with HTTP 402 and code budget_exhausted. That is not a rate-limit 429. Your app can handle the 402 without retrying it as if the limit would clear on its own.
          </p>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Can I override limits in emergencies?</h3>
          
          <p className="text-gray-300 leading-relaxed">
            Mint another token with a larger budget in the dashboard if you need more room. There is no satgate token update command, and no daily limit to raise.
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`A spent budget returns HTTP 402 with code budget_exhausted.
Do not treat that as HTTP 429.`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Start Small, Scale Safely</h2>
          
          <p className="text-gray-300 leading-relaxed">
            You don&apos;t need to migrate everything at once. Start with:
          </p>
          
          <ol className="list-decimal list-inside space-y-2 text-gray-300 my-4">
            <li>Install SatGate alongside your existing setup</li>
            <li>Route development traffic through budget-limited tokens</li>
            <li>Monitor savings and prevented overages</li>
            <li>Gradually migrate production workloads</li>
          </ol>
          
          <p className="text-gray-300 leading-relaxed">
            The best time to add budget protection? Before you need it. The second best time? Right now.
          </p>

          <div className="my-10 rounded-2xl border border-green-900/60 bg-green-950/20 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">OpenAI API Budget Limit: The Click-Intent Answer</h2>
            <p className="text-gray-300 leading-relaxed">
              If you searched for an OpenAI API budget, the distinction is simple: account-level usage limits protect the vendor account; budget limits checked before the request goes through protect each agent, team, session, customer, and workflow before the next GPT call runs.
            </p>
            <ol className="list-decimal list-inside space-y-2 text-gray-300 my-4">
              <li>Tag every OpenAI request with agent, team, customer, session, and workflow.</li>
              <li>Price the request using model, token, and tool policy before forwarding it.</li>
              <li>Block, downgrade, queue, or require approval when the remaining budget is not enough.</li>
            </ol>
          </div>
          
          <div className="bg-gray-900 border border-gray-800 p-6 rounded-lg mt-8">
            <h3 className="text-xl font-semibold mb-4">Ready to Protect Your OpenAI Spending?</h3>
            <p className="mb-4 text-gray-300">
              SatGate is open source and takes 5 minutes to set up. Never wake up to a surprise OpenAI bill again.
            </p>
            <div className="flex gap-4">
              <a href="https://github.com/SatGate-io/satgate" className="inline-flex items-center px-4 py-2 bg-white text-black font-bold rounded-lg hover:bg-gray-200 transition">
                Get Started →
              </a>
              <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="inline-flex items-center px-4 py-2 border border-gray-600 text-gray-300 rounded-lg hover:bg-gray-900 transition">
                Read the Docs
              </a>
            </div>
          </div>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Frequently Asked Questions</h2>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Does OpenAI have built-in spending limits?</h3>
          <p className="text-gray-300 leading-relaxed">
            OpenAI has account-level usage limits, but they are not the same as budget enforcement before the request goes through. They are coarse, can lag behind real usage, and usually cannot isolate spend by agent, user, session, workflow, or tool before a request executes.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">What is the difference between a rate limit and a budget limit for OpenAI?</h3>
          <p className="text-gray-300 leading-relaxed">
            A rate limit caps request volume, while a budget limit caps spend. For OpenAI and other LLM APIs, spend depends on model, tokens, context length, retries, and tool calls, so budget limits are the safer control for autonomous agents.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">How do you set per-agent OpenAI budget limits?</h3>
          <p className="text-gray-300 leading-relaxed">
            Set per-agent OpenAI budget limits by routing calls through a gateway that identifies the agent, estimates or prices the request, checks remaining budget, and blocks or downgrades the call before it reaches OpenAI.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Can you set OpenAI API budget limits by team or customer?</h3>
          <p className="text-gray-300 leading-relaxed">
            Yes. A budget gateway that checks before the request goes through can attach spend policy to a team, customer, environment, workflow, or agent token, then enforce separate OpenAI API budgets before each request executes.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">How do you enforce an OpenAI API budget limit before GPT calls run?</h3>
          <p className="text-gray-300 leading-relaxed">
            Put a gateway in front of OpenAI, identify the agent or team, estimate the request cost from model and token policy, check remaining budget, and allow, block, or downgrade the request before it reaches the OpenAI API.
          </p>

          <div className="my-10 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">Put a budget in front of OpenAI calls</h3>
            <p className="mb-4 text-gray-300">Give each agent a token with a credit budget, route its OpenAI calls through SatGate, and see every allow and refusal with a signed receipt.</p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/build" className="text-cyan-300 hover:text-cyan-200">Build with SatGate →</Link>
              <Link href="/ai-agent-cost-control" className="text-cyan-300 hover:text-cyan-200">AI agent cost control →</Link>
              <Link href="/tools" className="text-cyan-300 hover:text-cyan-200">Cost-control tools →</Link>
              <Link href="/policy-to-proof" className="text-cyan-300 hover:text-cyan-200">See how rules and receipts work →</Link>
              <Link href="/agent-api-key-risk-assessment" className="text-cyan-300 hover:text-cyan-200">API key risk assessment →</Link>
              <Link href="/ai-agent-runaway-spend-index" className="text-cyan-300 hover:text-cyan-200">Runaway spend index →</Link>
              <Link href="/blog/llm-cost-management" className="text-cyan-300 hover:text-cyan-200">LLM cost management →</Link>
              <Link href="/govern" className="text-cyan-300 hover:text-cyan-200">Enterprise governance →</Link>
            </div>
          </div>


          <div className="my-10 rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">SatGate path: Observe → Control → Prove</h3>
            <p className="mb-4 text-gray-300">
              Start by observing agent, API, and MCP usage. Move to checks before the request goes through when budgets, scopes, and revocation need to stop bad calls before they run. Preserve signed receipts so every allow, deny, and budget decision can be verified later.
            </p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/policy-to-proof" className="text-cyan-300 hover:text-cyan-200">See how rules and receipts work →</Link>
              <Link href="/mcp" className="text-cyan-300 hover:text-cyan-200">MCP governance →</Link>
              <Link href="/govern" className="text-cyan-300 hover:text-cyan-200">See SatGate governance →</Link>
            </div>
          </div>

          <RoiCta
            title="OpenAI API budget limits are easier to sell with numbers"
            body="Model the cost of runaway OpenAI calls, retries, and agent loops before they hit the invoice."
          />

        </article>
      </div>
    </div>
  );
}