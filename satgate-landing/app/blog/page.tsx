import Link from 'next/link';
import { ArrowLeft, ArrowRight, Calendar, Clock, User } from 'lucide-react';

export const metadata = {
  title: 'Blog - SatGate',
  description: 'Insights on API economics, agent governance, and the future of machine-to-machine commerce.',
};

// Blog post data - in a real setup this would come from a CMS or markdown files
const posts = [
  {
    slug: 'cursor-mcp-proxy-setup-guide',
    title: 'Cursor MCP Proxy Setup Guide: Add Budget Controls and Audit Trails to Your Tools',
    description: 'Learn how to set up a Cursor MCP proxy with SatGate to enforce budgets, meter tool usage, and add audit trails without changing your MCP servers.',
    date: '2026-04-09',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['Cursor', 'MCP', 'Tutorial', 'Governance'],
  },
  {
    slug: 'start-at-1-credit-economic-policy',
    title: 'Start at 1 Credit: A Smarter Way to Price AI Agent Tools',
    description: 'Stop guessing tool costs on day one. Start everything at 1 credit in Observe mode, measure real usage, then use that data to shape behavior with intentional pricing.',
    date: '2026-04-07',
    readTime: '5 min read',
    author: 'Wayne Mattadeen',
    tags: ['Economic Policy', 'Best Practices', 'AI Agents'],
  },
  {
    slug: 'how-to-add-budget-limits-to-openai-api-calls',
    title: 'OpenAI API Budget Limits: Stop Runaway GPT Spend Before It Happens',
    description: 'Learn how to enforce OpenAI API budget limits before each request, then use the ROI calculator to quantify avoided agent-loop spend.',
    date: '2026-04-07',
    readTime: '8 min read',
    author: 'Matt Dean',
    tags: ['OpenAI', 'Cost Control', 'Tutorial', 'API Gateway'],
  },
  {
    slug: 'zero-trust-for-ai-agents',
    title: 'Zero Trust for AI Agents: Why Identity-Based Security Collapses When Machines Call the Shots',
    description: 'Zero Trust was built for human users and managed devices. AI agents break every assumption it makes. Here\'s how capability-based security fixes what Zero Trust can\'t.',
    date: '2026-04-03',
    readTime: '12 min read',
    author: 'Matt Dean',
    tags: ['Zero Trust', 'Security', 'AI Agents'],
  },
  {
    slug: 'http-402-payment-required-use-cases',
    title: 'HTTP 402 Payment Required: The Dormant Status Code That Powers the Agent Economy',
    description: 'HTTP 402 Payment Required has been "reserved for future use" since 1997. AI agents finally make it essential. Here are the real-world use cases unlocking it.',
    date: '2026-04-02',
    readTime: '11 min read',
    author: 'Matt Dean',
    tags: ['HTTP', 'Payments', 'AI Agents'],
  },
  {
    slug: 'l402-protocol-explained',
    title: 'L402 Protocol Explained: How HTTP 402 Enables Machine-Native API Payments',
    description: 'L402 combines HTTP 402 Payment Required with Lightning Network micropayments and macaroon tokens. Learn how this protocol enables AI agents to pay for API access in real time.',
    date: '2026-04-02',
    readTime: '11 min read',
    author: 'Matt Dean',
    tags: ['L402', 'Payments', 'API Economics'],
  },
  {
    slug: 'macaroon-tokens-vs-api-keys',
    title: 'Macaroon Tokens vs API Keys: Why Capability-Based Auth Beats Identity-Based Auth for AI Agents',
    description: 'API keys tie identity to unlimited access. Macaroon tokens embed capabilities and constraints. For AI agents that need delegation and budget limits, the difference is everything.',
    date: '2026-03-31',
    readTime: '12 min read',
    author: 'Matt Dean',
    tags: ['Authentication', 'AI Agents', 'Security'],
  },
  {
    slug: 'why-process-wont-scale-for-ai-agent-costs',
    title: "InformationWeek Says Control AI Agent Costs With Process. Here's Why That Won't Scale.",
    description: "InformationWeek's 9 recommendations for AI agent cost control are solid for 2025. But when agents make 1,500 calls per prompt and operate 24/7, process can't keep pace. 7 of 9 are automatable at the infrastructure layer.",
    date: '2026-03-28',
    readTime: '10 min read',
    author: 'SatGate Team',
    tags: ['Cost Control', 'AI Agents', 'Thought Leadership'],
  },
  {
    slug: 'api-monetization-ai',
    title: 'API Monetization for AI: How to Charge Agents, Not Just Developers',
    description: 'AI agents are your next API customers. Traditional API monetization fails for autonomous workloads. Learn how to price, meter, and collect from machine consumers.',
    date: '2026-03-26',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['Monetization', 'AI Agents', 'API Economics'],
  },
  {
    slug: 'mcp-gateway-guide',
    title: 'MCP Gateway Guide: From Traffic Routing to Economic Governance',
    description: 'A complete MCP gateway guide covering architecture, auth, tool aggregation, and the economic governance layer most guides miss. Deploy with budget enforcement.',
    date: '2026-03-24',
    readTime: '11 min read',
    author: 'Matt Dean',
    tags: ['MCP', 'Gateway', 'Guide', 'Economic Governance'],
  },
  {
    slug: 'can-adversaries-game-your-economic-firewall',
    title: 'Can Adversaries Game Your Economic Firewall?',
    description: 'Economic firewalls protect against runaway AI agent costs — but what happens when attackers deliberately try to exploit them? We examine four adversarial attack vectors and the cryptographic defenses that stop them.',
    date: '2026-03-23',
    readTime: '14 min read',
    author: 'Wayne Mattadeen',
    tags: ['Security', 'Economic Firewall', 'Adversarial AI', 'Macaroons'],
  },
  {
    slug: 'the-enterprise-adoption-playbook-observe-control-charge',
    title: 'The Enterprise Adoption Playbook: Observe, Control, Charge',
    description: 'Observe, Control, Charge isn\'t just a product taxonomy — it\'s an enterprise change management strategy for adopting economic governance incrementally, building trust at each stage.',
    date: '2026-03-20',
    readTime: '11 min read',
    author: 'Matt Dean',
    tags: ['Enterprise', 'Governance', 'Change Management', 'Strategy'],
  },
  {
    slug: 'why-economic-firewalls-are-the-prerequisite-for-autonomous-ai-agents',
    title: 'Why Economic Firewalls Are the Prerequisite for Autonomous AI Agents',
    description: 'Economic firewalls aren\'t just safety tools — they\'re the enablers that unlock fully autonomous AI agents. By solving unbounded cost, they create the trust infrastructure for agents to make independent purchasing decisions.',
    date: '2026-03-20',
    readTime: '11 min read',
    author: 'Matt Dean',
    tags: ['Agent Economy', 'Economic Firewall', 'Autonomous Agents'],
  },
  {
    slug: 'ai-governance-api-teams',
    title: 'AI Governance for API Teams: Why Your Gateway Needs Policy, Not Just Routing',
    description: 'API teams need AI governance that enforces budgets, permissions, and audit trails — not just traffic routing. Learn why traditional API management falls short.',
    date: '2026-03-19',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['Governance', 'API Teams', 'Policy Enforcement'],
  },
  {
    slug: 'llm-cost-management',
    title: 'LLM Cost Management: Dashboards vs Real-Time Budget Enforcement',
    description: 'LLM cost dashboards show what happened. Real-time enforcement controls what agents can spend before the bill arrives.',
    date: '2026-03-17',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['Cost Control', 'LLM', 'Economic Firewall'],
  },
  {
    slug: 'api-gateway-for-ai-agents',
    title: 'API Gateway for AI Agents: Why Traditional Gateways Fall Short',
    description: 'Traditional API gateways route traffic. AI agents need economic governance. Learn why Solo.io, Kong, and Gravitee weren\'t built for autonomous agent workloads.',
    date: '2026-03-12',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['Gateway', 'AI Agents', 'Architecture'],
  },
  {
    slug: 'deepmind-intelligent-delegation-satgate',
    title: 'What Google DeepMind Gets Right About Agent Delegation — And What SatGate Already Built',
    description: 'DeepMind\'s Intelligent AI Delegation paper proposes macaroon-based capability tokens for safe agent delegation. Here\'s how their framework maps to SatGate\'s architecture.',
    date: '2026-03-11',
    readTime: '8 min read',
    author: 'Matt Dean',
    tags: ['Research', 'Delegation', 'Macaroons'],
  },
  {
    slug: 'ai-agent-spending-limits',
    title: 'AI Agent Spending Limits: Why API Keys Aren\'t Enough',
    description: 'API rate limits don\'t control agent costs. Learn how economic firewalls enforce real-time budget limits on autonomous AI spend — per agent, per tool.',
    date: '2026-03-10',
    readTime: '9 min read',
    author: 'Matt Dean',
    tags: ['Cost Control', 'AI Agents', 'Budget Enforcement'],
  },
  {
    slug: 'ai-agent-api-cost-control',
    title: 'How to Control AI Agent API Costs: Rate Limiting vs Economic Firewalls',
    description: 'Rate limiting doesn\'t understand money. Learn how economic firewalls give you real budget enforcement for AI agent API spend — per agent, per tool, in real time.',
    date: '2026-03-05',
    readTime: '8 min read',
    author: 'Matt Dean',
    tags: ['Cost Control', 'AI Agents', 'Gateway'],
  },
  {
    slug: 'what-is-an-economic-firewall',
    title: 'What Is an Economic Firewall? The Security Primitive for the Agent Economy',
    description: 'An economic firewall enforces budget limits on AI agent API calls at the gateway layer. Learn how it differs from traditional API security and why agents need it.',
    date: '2026-03-05',
    readTime: '6 min read',
    author: 'Matt Dean',
    tags: ['Concepts', 'Economic Firewall', 'Agent Economy'],
  },
  {
    slug: 'mcp-budget-enforcement-guide',
    title: 'MCP Budget Enforcement: A Practical Guide to Controlling AI Tool Spend',
    description: 'How to enforce per-tool budgets on MCP servers. Assign costs per tool call, cap agent spending, and attribute costs across teams.',
    date: '2026-03-05',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['MCP', 'Budget Enforcement', 'Tutorial'],
  },
  {
    slug: 'agent-swarms-cost-governance',
    title: 'Agent Swarms Are Here. Who\'s Controlling the Spend?',
    description: 'Multi-agent AI systems multiply API costs exponentially. Without economic governance, agent swarms can burn through budgets in minutes.',
    date: '2026-03-05',
    readTime: '7 min read',
    author: 'Matt Dean',
    tags: ['Agent Economy', 'Cost Control', 'Multi-Agent'],
  },
  {
    slug: 'security-as-a-profit-center',
    title: 'Security as a Profit Center: Why Your Economic Firewall Pays for Itself',
    description: 'Every security tool tells you how many attacks it stopped. SatGate tells you how many dollars it saved. Here\'s the CFO math on why economic governance is a profit center.',
    date: '2026-02-14',
    readTime: '12 min read',
    author: 'Matt Dean',
    tags: ['ROI', 'Governance', 'Enterprise', 'Cost Management'],
  },
  {
    slug: 'hard-capping-mcp-tool-spend',
    title: 'Hard-Capping MCP Tool Spend with SatGate Proxy',
    description: 'Your AI agent burned $500 overnight calling tools in a loop. Here\'s how to enforce real-time budget hard caps at the protocol level.',
    date: '2026-02-14',
    readTime: '10 min read',
    author: 'Matt Dean',
    tags: ['MCP', 'Budget Control', 'L402', 'Cost Management'],
  },
  {
    slug: 'how-we-built-budget-enforcement-mcp',
    title: 'How We Built Budget Enforcement for MCP Tool Calls',
    description: '2,164 lines of Go, 28 tests, and one evening. An open-source MCP proxy that enforces per-tool budgets with cryptographic delegation.',
    date: '2026-02-13',
    readTime: '8 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'Go', 'Macaroons', 'Engineering'],
  },
  {
    slug: 'beyond-connection-economic-governance-mcp',
    title: 'Beyond Connection: The Case for Economic Governance in MCP',
    description: 'The MCP ecosystem talks about capability. Nobody talks about cost. Here\'s why economic policy is the missing layer — and how to enforce it at the protocol level with L402 and macaroons.',
    date: '2026-02-12',
    readTime: '12 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'L402', 'Macaroons', 'Agent Economy'],
  },
  {
    slug: 'why-routing-isnt-governance',
    title: 'Why Routing Isn\'t Governance',
    description: 'AI gateways excel at routing LLM calls. But when agents control spend autonomously, routing isn\'t enough. You need economic governance.',
    date: '2026-02-06',
    readTime: '5 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'Governance', 'Agent Economy'],
  },
];

export default function BlogPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-4xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <div className="mb-12">
          <h1 className="text-4xl font-bold mb-4">Blog</h1>
          <p className="text-xl text-gray-400">
            Insights on API economics, agent governance, and the future of machine-to-machine commerce.
          </p>
        </div>

        <div className="space-y-6">
          {posts.map((post) => (
            <Link 
              key={post.slug}
              href={`/blog/${post.slug}`}
              className="block bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-purple-600/50 transition group"
            >
              <div className="flex flex-wrap gap-2 mb-3">
                {post.tags.map((tag) => (
                  <span 
                    key={tag}
                    className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono"
                  >
                    {tag}
                  </span>
                ))}
              </div>
              
              <h2 className="text-xl font-bold mb-2 group-hover:text-purple-400 transition">
                {post.title}
              </h2>
              
              <p className="text-gray-400 mb-4">
                {post.description}
              </p>
              
              <div className="flex items-center gap-4 text-sm text-gray-500">
                <span className="flex items-center gap-1">
                  <Calendar size={14} />
                  {new Date(post.date).toLocaleDateString('en-US', { 
                    month: 'short', 
                    day: 'numeric', 
                    year: 'numeric' 
                  })}
                </span>
                <span className="flex items-center gap-1">
                  <Clock size={14} />
                  {post.readTime}
                </span>
              </div>
              
              <div className="mt-4 flex items-center text-purple-400 text-sm font-medium group-hover:gap-2 transition-all">
                Read more <ArrowRight size={16} className="ml-1" />
              </div>
            </Link>
          ))}
        </div>

        {/* Newsletter CTA */}
        <section className="mt-16 bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-800/30 rounded-xl p-8 text-center">
          <h2 className="text-2xl font-bold mb-3">Stay Updated</h2>
          <p className="text-gray-400 mb-6">
            Get insights on API economics and agent governance delivered to your inbox.
          </p>
          <a 
            href="mailto:contact@satgate.io?subject=Newsletter%20Subscription"
            className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition"
          >
            Subscribe to Updates
          </a>
        </section>
      </div>
    </div>
  );
}
