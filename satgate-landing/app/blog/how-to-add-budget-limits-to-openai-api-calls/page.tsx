import Link from 'next/link';
import RoiCta from '../../components/RoiCta';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: "How to Set Budget Limits on OpenAI API Calls (Hard Caps, Not Just Alerts)",
  description: "Stop runaway OpenAI bills before they happen. Set hard spend caps per agent, per session, and per team — enforced at the request layer, not the dashboard.",
  alternates: { canonical: 'https://satgate.io/blog/how-to-add-budget-limits-to-openai-api-calls' },
  keywords: ['OpenAI API budget limits', 'OpenAI cost control', 'API gateway OpenAI', 'GPT-4 spending limits', 'OpenAI API costs', 'prevent OpenAI overspending', 'hard cap OpenAI spend', 'per-agent OpenAI budget']
};

export default function HowToAddBudgetLimitsToOpenAIAPICallsPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
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
          
          <h1 className="text-4xl font-bold mb-4">How to Set Hard Budget Limits on OpenAI API Calls</h1>
          <div className="mb-6 rounded-2xl border border-green-900/60 bg-green-950/20 p-5">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-green-300">Direct answer</p>
            <p className="text-gray-300">The safest way to set OpenAI API budget limits is to enforce spend before each request reaches OpenAI. Put a gateway in the request path, assign budgets per agent/team/session, estimate the call cost, and block or downgrade requests before runaway loops burn budget.</p>
          </div>
          
          <p className="text-xl text-gray-400 mb-6">
            OpenAI&apos;s dashboard shows you costs after they happen. By then, it&apos;s too late. Learn how to enforce hard budget limits that block requests before they overspend.
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
            <code>{`# Install SatGate
npm install -g @satgate/gateway

# Start with OpenAI proxy
satgate start --proxy openai`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 2: Create Budget-Limited Tokens</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Instead of using your OpenAI API key directly, create derivative tokens with spending limits:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Development token: $10/day for testing
satgate token create \\
  --name "dev-token" \\
  --daily-limit 10 \\
  --upstream openai

# Production token: $100/day with alerts at 80%
satgate token create \\
  --name "prod-token" \\
  --daily-limit 100 \\
  --alert-threshold 0.8 \\
  --upstream openai

# High-priority token: $500/day for critical paths
satgate token create \\
  --name "priority-token" \\
  --daily-limit 500 \\
  --hourly-limit 50 \\
  --upstream openai`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 3: Update Your Code</h2>
          
          <p className="text-gray-300 leading-relaxed">
            The beautiful part? Your application code barely changes:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`import OpenAI from 'openai';

// Before: Direct OpenAI connection
// const openai = new OpenAI({
//   apiKey: process.env.OPENAI_API_KEY
// });

// After: Route through SatGate
const openai = new OpenAI({
  apiKey: process.env.SATGATE_TOKEN,  // Your budget-limited token
  baseURL: 'http://localhost:8000/v1' // SatGate proxy
});

// Everything else stays the same
const response = await openai.chat.completions.create({
  model: "gpt-4",
  messages: [{ role: "user", content: "Hello" }]
});`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 4: Configure Team Budgets</h2>
          
          <p className="text-gray-300 leading-relaxed">
            For larger teams, create hierarchical budgets:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Create team buckets
satgate budget create --name "engineering" --monthly 5000
satgate budget create --name "marketing" --monthly 2000
satgate budget create --name "support" --monthly 1000

# Create tokens within team budgets
satgate token create \\
  --name "eng-dev" \\
  --budget "engineering" \\
  --daily-limit 50

satgate token create \\
  --name "marketing-automation" \\
  --budget "marketing" \\
  --daily-limit 100 \\
  --model "gpt-3.5-turbo" # Restrict to cheaper models`}</code>
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
            <code>{`// Middleware to inject user-specific tokens
app.use(async (req, res, next) => {
  const userId = req.user.id;
  
  // Get or create user token
  let token = await cache.get(\`token:\${userId}\`);
  if (!token) {
    token = await satgate.tokens.create({
      name: \`user-\${userId}\`,
      daily_limit: 10,  // $10/day per user
      upstream: 'openai'
    });
    await cache.set(\`token:\${userId}\`, token, 86400);
  }
  
  // Inject token for OpenAI client
  req.openaiToken = token;
  next();
});

// Route handler uses user-specific token
app.post('/chat', async (req, res) => {
  const openai = new OpenAI({
    apiKey: req.openaiToken,
    baseURL: 'http://localhost:8000/v1'
  });
  
  try {
    const response = await openai.chat.completions.create({
      model: "gpt-4",
      messages: req.body.messages
    });
    res.json(response);
  } catch (error) {
    if (error.type === 'budget_exceeded') {
      res.status(429).json({
        error: "Daily limit reached. Upgrade for more credits."
      });
    }
  }
});`}</code>
          </pre>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Monitoring and Alerts</h2>
          
          <p className="text-gray-300 leading-relaxed">
            Unlike OpenAI&apos;s &ldquo;email after overspend&rdquo; approach, SatGate alerts you before problems:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Configure alerts
satgate alerts add \\
  --type webhook \\
  --url https://your-app.com/webhooks/budget-alerts \\
  --events "budget.80_percent,budget.exceeded,anomaly.detected"

# Alert payload when 80% spent
{
  "event": "budget.80_percent",
  "token": "prod-token",
  "spent": 80.00,
  "limit": 100.00,
  "period": "daily",
  "top_consumers": [
    { "endpoint": "/api/chat", "spent": 45.00 },
    { "endpoint": "/api/summarize", "spent": 35.00 }
  ]
}`}</code>
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
            Requests are immediately rejected with a 429 status and clear error message. Your app can handle this gracefully - offer upgrades, queue for later, or fall back to cached responses.
          </p>
          
          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Can I override limits in emergencies?</h3>
          
          <p className="text-gray-300 leading-relaxed">
            Yes. Create emergency tokens with higher limits or use temporary overrides:
          </p>
          
          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# Temporary override for incident response
satgate token update incident-token --daily-limit 1000 --expires 1h`}</code>
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
          
          <div className="bg-gray-900 border border-gray-800 p-6 rounded-lg mt-8">
            <h3 className="text-xl font-semibold mb-4">Ready to Protect Your OpenAI Spending?</h3>
            <p className="mb-4 text-gray-300">
              SatGate is open source and takes 5 minutes to set up. Never wake up to a surprise OpenAI bill again.
            </p>
            <div className="flex gap-4">
              <a href="https://github.com/satgatelabs/satgate" className="inline-flex items-center px-4 py-2 bg-white text-black font-bold rounded-lg hover:bg-gray-200 transition">
                Get Started →
              </a>
              <a href="/docs" className="inline-flex items-center px-4 py-2 border border-gray-600 text-gray-300 rounded-lg hover:bg-gray-900 transition">
                Read the Docs
              </a>
            </div>
          </div>
          
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Frequently Asked Questions</h2>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Does OpenAI have built-in spending limits?</h3>
          <p className="text-gray-300 leading-relaxed">
            Yes, but they&apos;re monthly account-level caps with delayed enforcement. They don&apos;t support per-agent, per-user, or per-session granularity, and &ldquo;hard limits&rdquo; can still overshoot by 10–20% because usage data is cached. When your account limit is hit, all API access stops — not just the offending agent.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">What&apos;s the difference between a rate limit and a budget limit?</h3>
          <p className="text-gray-300 leading-relaxed">
            Rate limits cap requests-per-minute. Budget limits cap dollar spend. For AI agents, budget limits are the meaningful control — one GPT-4o call with a large context can cost 1,000× more than a cached lookup. Counting requests tells you nothing about cost exposure.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">How do I set a per-agent budget without changing my application code?</h3>
          <p className="text-gray-300 leading-relaxed">
            Route API calls through a request-path gateway. The gateway checks budget before forwarding each request — your agent code only changes the <code className="text-green-300 bg-black/50 px-1 rounded">baseURL</code> and token. Budget policy lives in the gateway config, not in the agent.
          </p>

          <div className="my-10 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">Turn OpenAI limits into enforceable policy</h3>
            <p className="mb-4 text-gray-300">Use the policy generator and spend template to convert this guide into per-agent, per-session, per-request, and model-route controls.</p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/openai-budget-policy-generator" className="text-cyan-300 hover:text-cyan-200">OpenAI budget generator →</Link>
              <Link href="/agent-spend-policy-template" className="text-cyan-300 hover:text-cyan-200">Agent spend policy template →</Link>
              <Link href="/ai-agent-runaway-spend-index" className="text-cyan-300 hover:text-cyan-200">Runaway spend index →</Link>
              <Link href="/blog/llm-cost-management" className="text-cyan-300 hover:text-cyan-200">LLM cost management →</Link>
              <Link href="/govern" className="text-cyan-300 hover:text-cyan-200">Enterprise governance →</Link>
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