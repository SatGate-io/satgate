import Link from 'next/link';
import { ArrowLeft, ArrowRight, Calendar, Clock } from 'lucide-react';

const title = 'Always-On Agents Need More Than Identity: Economic Authority';
const description =
  'Microsoft Scout shows where enterprise agents are headed. Observe, Control, and Prove agent actions with economic authority, spend ownership, and evidence.';
const url = 'https://satgate.io/blog/always-on-agents-economic-authority';

export const metadata = {
  title: 'Always-On Agents Need More Than Identity: Economic Authority',
  description:
    'Microsoft Scout shows where enterprise agents are headed. Observe, Control, and Prove agent actions with economic authority, spend ownership, and evidence.',
  alternates: { canonical: 'https://satgate.io/blog/always-on-agents-economic-authority' },
  keywords: [
    'always-on agents',
    'AI agent governance',
    'enterprise AI agent governance',
    'economic authority',
    'economic firewall',
    'MCP governance',
    'AI agent spend controls',
    'agent policy enforcement',
    'agent audit evidence',
    'Microsoft Scout',
  ],
  openGraph: {
    title,
    description,
    url,
    type: 'article',
    publishedTime: '2026-06-04T00:00:00Z',
    modifiedTime: '2026-06-04T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title,
    description,
  },
};

const tagClass = 'px-2 py-1 rounded-full border text-xs font-mono';
const paragraphClass = 'text-gray-300 leading-relaxed';

export default function AlwaysOnAgentsEconomicAuthorityBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: title,
    description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-06-04',
    dateModified: '2026-06-04',
    mainEntityOfPage: url,
    about: [
      { '@type': 'Thing', name: 'always-on agents' },
      { '@type': 'Thing', name: 'AI agent governance' },
      { '@type': 'Thing', name: 'economic authority' },
      { '@type': 'Thing', name: 'Economic Firewall' },
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'Policy-to-Proof' },
      { '@type': 'Thing', name: 'Evidence Packs' },
      { '@type': 'Thing', name: 'Microsoft Scout' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Why do always-on agents need more than identity?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Identity tells an enterprise who an agent is. Always-on agents also need economic authority: runtime controls that decide which actions are allowed, what they can cost, who pays, and what evidence proves the decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is economic authority for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Economic authority is the right for an agent to cause value to move, including money, API credits, paid data, cloud usage, SaaS actions, delegated work, or other resources the business pays for.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does MCP governance relate to economic authority?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP gives agents a common way to call tools. MCP governance adds policy, budget, delegation, route, tenant, and evidence checks before those tool calls execute.',
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
            <span className={`${tagClass} bg-purple-900/30 border-purple-500/30 text-purple-300`}>AI Agent Governance</span>
            <span className={`${tagClass} bg-cyan-900/30 border-cyan-500/30 text-cyan-300`}>Economic Authority</span>
            <span className={`${tagClass} bg-yellow-900/30 border-yellow-500/30 text-yellow-300`}>MCP Governance</span>
          </div>

          <h1 className="text-4xl font-bold mb-4">Always-on agents need more than identity. They need economic authority.</h1>

          <div className="mb-6 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-5">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">Short answer</p>
            <p className="text-gray-300">
              Microsoft Scout shows the category is real: enterprises are starting to govern autonomous agents as real actors. Identity and access matter. But when agents call external APIs, use MCP tools, spend credits, trigger SaaS actions, or delegate work, they also need economic authority that can be enforced and proven.
            </p>
          </div>

          <div className="mb-6 flex flex-col gap-3 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center rounded-lg bg-white px-5 py-3 text-sm font-bold text-black transition hover:bg-gray-200">
              Govern agent action <ArrowRight size={16} className="ml-2" />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              Economic Firewall
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              MCP governance
            </Link>
          </div>

          <p className="text-xl text-gray-400 mb-6 italic">
            The next control layer is deciding what agents are economically allowed to do, who pays, and what proof exists. Observe, Control, Prove is the operating model.
          </p>

          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> Jun 4, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 8 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          <p className="text-gray-300 text-lg leading-relaxed">
            Microsoft&apos;s Scout announcement is worth paying attention to, but not because every company suddenly needs a Microsoft-flavored personal assistant.
          </p>

          <p className={paragraphClass}>The important part is quieter than that.</p>

          <p className={paragraphClass}>
            Microsoft is treating an AI agent as a real enterprise actor. Scout is always on. It has its own identity. It works in the background. It can operate across Teams, Outlook, OneDrive, SharePoint, email, calendar, contacts, the browser, local resources, and MCP servers. It is governed through Entra, Purview, Intune policy, approvals, and Microsoft 365 controls.
          </p>

          <p className={paragraphClass}>That is the shift.</p>

          <p className={paragraphClass}>
            We are moving from assistants that answer questions to agents that keep working after the conversation ends. Once an agent can act without being prompted every time, the governance problem changes completely.
          </p>

          <p className={paragraphClass}>It is no longer enough to ask, &quot;Who is this agent?&quot;</p>

          <p className={paragraphClass}>
            You also have to ask, &quot;What is it allowed to do? What can that action cost? Who is paying for it? Who approved the authority? Can the agent delegate that authority? And what proof exists after the fact?&quot;
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Identity is necessary. It is not the whole job.</h2>

          <p className={paragraphClass}>Scout gets one big thing right: agents need identity.</p>

          <p className={paragraphClass}>
            Shared service accounts were already a bad habit for software. They are worse for autonomous agents. If an agent schedules a meeting, sends a file, updates a system, calls an API, or triggers a workflow, the enterprise needs to know which agent acted, on whose behalf, and under which policy.
          </p>

          <p className={paragraphClass}>That matters.</p>

          <p className={paragraphClass}>But identity and access control mostly answer one class of questions:</p>

          <ul className="text-gray-300 space-y-2">
            <li>Who is acting?</li>
            <li>What data can they reach?</li>
            <li>Which internal resources are they allowed to touch?</li>
          </ul>

          <p className={paragraphClass}>Those are Microsoft 365 questions. Entra and Purview are strong tools for that world.</p>

          <p className={paragraphClass}>The next problem starts when the agent crosses the boundary.</p>

          <p className={paragraphClass}>
            What happens when that same agent calls an external API? Uses a third-party MCP server? Consumes paid credits? Kicks off a SaaS admin action? Delegates a task to another agent? Buys data? Runs a model that costs real money? Opens a support workflow that commits the company to something?
          </p>

          <p className={paragraphClass}>Now the question is not just access. It is authority with economic consequences.</p>

          <div className="my-8 rounded-2xl border border-yellow-900/60 bg-yellow-950/20 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">The clean split</h2>
            <p className="text-gray-300 leading-relaxed mb-4">
              Scout handles identity, access, and data movement inside Microsoft 365.
            </p>
            <p className="text-gray-300 leading-relaxed mb-0">
              The Economic Firewall handles economic authority across external APIs, MCP tools, SaaS actions, paid rails, and delegation chains: what actions are allowed, who pays, and what proof exists.
            </p>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The missing layer is economic authority.</h2>

          <p className={paragraphClass}>Economic authority is the right to cause value to move.</p>

          <p className={paragraphClass}>
            That value might be money. It might be API credits, cloud usage, paid data, inventory, compute, support obligations, contract changes, or anything else the business ultimately pays for.
          </p>

          <p className={paragraphClass}>
            Humans deal with this through messy but familiar systems: budgets, approvals, procurement rules, delegated signing authority, reimbursement policies, vendor contracts, audit records.
          </p>

          <p className={paragraphClass}>Agents need the same kind of boundary, but enforced at machine speed.</p>

          <p className={paragraphClass}>Before an agent acts, the system should be able to answer:</p>

          <ul className="text-gray-300 space-y-2">
            <li>Is this action allowed for this agent, this user, this tenant, and this task?</li>
            <li>Does the agent have authority to spend or consume this resource?</li>
            <li>What budget applies?</li>
            <li>Is there a per-call, per-route, or per-vendor limit?</li>
            <li>Can this authority be delegated to another agent?</li>
            <li>When does the grant expire?</li>
            <li>What happens if the action is denied?</li>
            <li>What receipt proves the decision later?</li>
          </ul>

          <p className={paragraphClass}>This is where a normal identity stack starts to run out of road.</p>

          <p className={paragraphClass}>
            Identity tells you who the agent is. Access control tells you what it can reach. Economic authority tells you what it is allowed to cause.
          </p>

          <p className={paragraphClass}>That last part is the new governance problem.</p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">MCP makes this urgent.</h2>

          <p className={paragraphClass}>
            MCP is useful because it gives agents a common way to discover and use tools. That is also what makes it risky.
          </p>

          <p className={paragraphClass}>A tool call is not just a technical event. It can be a business event.</p>

          <p className={paragraphClass}>
            One MCP call might read a document. Another might update a CRM record. Another might query a paid data source. Another might initiate a refund, provision infrastructure, send a message to a customer, or call a model that costs $20 in one shot.
          </p>

          <p className={paragraphClass}>From the agent&apos;s point of view, these are all just tools.</p>

          <p className={paragraphClass}>From the business&apos;s point of view, they are very different kinds of authority.</p>

          <p className={paragraphClass}>
            That difference needs to be enforced before execution, not reconstructed from logs after something expensive or embarrassing happens.
          </p>

          <p className={paragraphClass}>
            This is why <Link href="/mcp-governance" className="text-cyan-300 hover:text-cyan-200">MCP governance</Link> matters. The gateway between agents and tools is becoming the natural place to check policy, budget, tenant, delegation depth, and evidence requirements before the action goes out.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Logs are not proof.</h2>

          <p className={paragraphClass}>
            A lot of agent governance will start with observability. That is fine. Teams need traces, logs, analytics, and dashboards.
          </p>

          <p className={paragraphClass}>But logs answer a weaker question: &quot;What happened?&quot;</p>

          <p className={paragraphClass}>The harder question is: &quot;Was it allowed to happen?&quot;</p>

          <p className={paragraphClass}>And after that: &quot;Can we prove which policy allowed or denied it?&quot;</p>

          <p className={paragraphClass}>
            For always-on agents, that distinction matters. A record of an action is useful. A record of the authority decision is more useful. A portable receipt that shows the agent, user, tenant, policy, budget, request, decision, and evidence is better still.
          </p>

          <div className="my-8 rounded-2xl border border-purple-900/60 bg-purple-950/20 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Policy-to-Proof for agent action</h2>
            <ol className="text-gray-300 space-y-2">
              <li>Define authority before execution.</li>
              <li>Enforce it at runtime.</li>
              <li>Preserve proof after the decision.</li>
            </ol>
            <p className="mt-5 text-gray-300 leading-relaxed mb-0">
              Every meaningful agent action should leave a receipt.
            </p>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Where Microsoft ends and the wider agent economy begins.</h2>

          <p className={paragraphClass}>
            Microsoft is going to be strongest inside the Microsoft 365 trust boundary. That is the obvious place for Scout, Work IQ, Entra, Purview, Teams, Outlook, SharePoint, and Copilot to come together.
          </p>

          <p className={paragraphClass}>Enterprises will still need something else for the messy world outside that boundary.</p>

          <p className={paragraphClass}>
            They will have agents calling external APIs, using third-party MCP servers, touching SaaS tools, consuming paid services, delegating work to other agents, and operating across hybrid environments. Some of those actions will be harmless. Some will move data. Some will cost money. Some will create obligations.
          </p>

          <p className={paragraphClass}>
            That is the space SatGate calls the <Link href="/economic-firewall" className="text-cyan-300 hover:text-cyan-200">Economic Firewall</Link> for agents.
          </p>

          <p className={paragraphClass}>Not a payment rail. Not another chatbot. Not a dashboard that notices problems after the fact.</p>

          <p className={paragraphClass}>An Economic Firewall sits at the point where an agent tries to act and asks a simple set of questions:</p>

          <ul className="text-gray-300 space-y-2">
            <li>Is this action allowed?</li>
            <li>Is this spend allowed?</li>
            <li>Is this delegation allowed?</li>
            <li>Is this route allowed?</li>
            <li>What proof should be produced?</li>
          </ul>

          <p className={paragraphClass}>
            If the answer is yes, the action proceeds with a receipt. If the answer is no, the action is denied with a receipt.
          </p>

          <p className={paragraphClass}>
            That sounds boring until you imagine thousands of agents running in the background across enterprise systems. Then it becomes basic plumbing.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The split is clean.</h2>

          <p className={paragraphClass}>Scout handles identity, access, and data movement inside Microsoft 365.</p>

          <p className={paragraphClass}>
            The Economic Firewall handles economic authority across external APIs, MCP tools, SaaS actions, paid rails, and delegation chains.
          </p>

          <p className={paragraphClass}>One answers who can act and what data can move.</p>

          <p className={paragraphClass}>The other answers what actions are economically allowed, who pays, and what proof exists.</p>

          <p className={paragraphClass}>Both matter.</p>

          <p className={paragraphClass}>
            But the second problem is the one enterprises are about to feel as agents move from copilots to coworkers.
          </p>

          <p className={paragraphClass}>Microsoft Scout makes the agent actor real.</p>

          <p className={paragraphClass}>Now the question is whether that actor has authority before it acts.</p>

          <div className="my-10 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-6">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">Next step</p>
            <h2 className="text-2xl font-bold text-white mb-3">Govern external agent actions before they execute</h2>
            <p className="text-gray-300 leading-relaxed mb-5">
              If your agents are starting to call MCP tools, paid APIs, SaaS workflows, or delegated agent systems, put policy and proof in the request path before autonomy scales.
            </p>
            <div className="flex flex-col gap-3 sm:flex-row">
              <Link href="/govern" className="inline-flex items-center justify-center rounded-lg bg-white px-5 py-3 text-sm font-bold text-black transition hover:bg-gray-200">
                See agent governance <ArrowRight size={16} className="ml-2" />
              </Link>
              <Link href="/policy-to-proof" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
                See Policy-to-Proof
              </Link>
            </div>
          </div>
        </article>
      </div>
    </div>
  );
}
