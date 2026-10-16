import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

const title = "Zero trust in the model: Nadella's insider-risk rules and where SatGate fits";
const description = 'Satya Nadella says to treat AI models as insider risks and keep the controls outside the model. What that asks of agent builders, and what SatGate does today.';
const socialTitle = "Zero trust in the model: Nadella's insider-risk rules";

export const metadata = {
  title: socialTitle,
  description,
  alternates: { canonical: 'https://satgate.io/blog/zero-trust-in-the-model' },
  keywords: ['Zero Trust', 'AI Agents', 'Security', 'insider risk', 'model controls'],
  openGraph: {
    title: socialTitle,
    description,
    url: 'https://satgate.io/blog/zero-trust-in-the-model',
    type: 'article',
    publishedTime: '2026-10-16T00:00:00Z',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: socialTitle,
    description,
  },
};

export default function ZeroTrustInTheModelBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: title,
    description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-10-16',
    dateModified: '2026-10-16',
    mainEntityOfPage: 'https://satgate.io/blog/zero-trust-in-the-model',
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>

        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono">Zero Trust</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">AI Agents</span>
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">Security</span>
          </div>

          <h1 className="text-4xl font-bold mb-4">{title}</h1>

          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> October 16, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 4 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          <p className="text-gray-300 text-lg leading-relaxed">
            On October 10, Microsoft&apos;s CEO Satya Nadella published &quot;Models as Insider Risks in the Super Intelligence Era.&quot; The argument is short. Nobody can trace why a model does what it does, yet companies are giving models their most sensitive data and letting them act. So don&apos;t take the model&apos;s word for what it will do, and don&apos;t take the model provider&apos;s word either. As he puts it:
          </p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            A model provider&apos;s assurances do not relieve us of that responsibility.
          </blockquote>

          <p className="text-gray-300 leading-relaxed">
            His answer is an old security idea applied to a new kind of insider:
          </p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            we need to separate the supply of intelligence from the authority over it.
          </blockquote>
          <p className="text-gray-300 leading-relaxed">
            The model supplies the thinking. The company running it keeps the authority over what the model can reach, what it can do and when it stops.
          </p>

          <p className="text-gray-300 leading-relaxed">
            That split is the job SatGate does for AI agents. SatGate sits between your agents and the services they use, and every request an agent sends through it is checked against limits you set. Here is how his principles line up with what SatGate does today, and where it stops.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The controls sit outside the model</h2>

          <p className="text-gray-300 leading-relaxed">Nadella:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            the controls that govern what a model can access and what actions it can take must sit outside the model.
          </blockquote>
          <p className="text-gray-300 leading-relaxed">And:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            Orgs should be able to independently determine what a model can access and what actions it can take.
          </blockquote>

          <p className="text-gray-300 leading-relaxed">
            In SatGate you set each agent&apos;s budget, the tools it may use, rules on what those tools may do (buy only, or one account only) and an end date. The agent gets a token with those limits built in. SatGate checks each request against them before it reaches the service. The model can&apos;t change those limits. Neither can a prompt that tricks it, and neither can the model provider, because the limits live in your SatGate account and not in the model or the provider&apos;s settings.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Evidence the model didn&apos;t write</h2>

          <p className="text-gray-300 leading-relaxed">Nadella:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            Every meaningful model action must leave tamper-proof human readable evidence. If it can&apos;t be observed, it can&apos;t be trusted!
          </blockquote>
          <p className="text-gray-300 leading-relaxed">
            He also asks that no single model control both what a system does and the evidence of what it did.
          </p>

          <p className="text-gray-300 leading-relaxed">
            SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a permission. In Control mode, where SatGate enforces your limits, it also signs one when an agent spends from a budget. The receipt names the tool or route and says whether SatGate allowed or refused it and why. It doesn&apos;t name the agent or include the call&apos;s arguments. A SatGate Cloud receipt can be checked with the open-source verifier against SatGate&apos;s published keys. The agent never writes its own record, so the record doesn&apos;t depend on the model&apos;s account of itself.
          </p>

          <p className="text-gray-300 leading-relaxed">
            There are limits to this. Not every request gets a receipt. A refusal for going over a rate limit doesn&apos;t get one, and neither does a request that never goes through SatGate.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">An emergency brake</h2>

          <p className="text-gray-300 leading-relaxed">Nadella:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            We must assume a model is compromised and contain it from the start. Think of it like an emergency brake. An authorized person should always be able to pause or shut down a model mid-task.
          </blockquote>

          <p className="text-gray-300 leading-relaxed">
            Revoke an agent in the dashboard and its next request is refused, along with the requests of any helper agents it handed work to. Since this week, a revoke that SatGate can&apos;t confirm is shown as failed rather than done, so the off switch never claims to have worked when it didn&apos;t. You can also make an agent&apos;s riskiest calls wait until you approve them by email.
          </p>

          <p className="text-gray-300 leading-relaxed">
            What SatGate doesn&apos;t do is pause and resume. A stop is a stop, and a request already running when you press it finishes.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">No single model in charge</h2>

          <p className="text-gray-300 leading-relaxed">Nadella:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            No one model should become the sole dependency for an important outcome or be responsible for verifying its own work.
          </blockquote>

          <p className="text-gray-300 leading-relaxed">
            SatGate doesn&apos;t care which model runs an agent. The same limits and receipts apply whichever model or vendor you use, so switching models doesn&apos;t mean rebuilding your controls. Picking and testing the models is still your job.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">What this doesn&apos;t cover</h2>

          <p className="text-gray-300 leading-relaxed">
            Several of his points are about the models and the people who run them: reasoning that can be read, models testing each other, telling people when something fails. SatGate doesn&apos;t do those. It covers the actions an agent takes through it, and only those. An agent that holds its own keys to a service can go around any gateway, so give agents your services through SatGate rather than handing them the keys.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Trust the model the least</h2>

          <p className="text-gray-300 leading-relaxed">Nadella ends the essay with this:</p>
          <blockquote className="border-l-4 border-purple-500 pl-4 py-2 my-6 text-white italic text-lg">
            The most trustworthy Super Intelligence system will not be the one with the model we trust most. It will be the one that enables us to trust the model the least.
          </blockquote>

          <p className="text-gray-300 leading-relaxed">
            That is a good test for any agent setup. Assume the agent will be wrong or tricked sooner or later, and make sure the damage stops at the limits you set.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Read his full post:{' '}
            <a href="https://x.com/satyanadella/status/2108931348857827686" target="_blank" rel="noopener noreferrer" className="text-purple-400 hover:text-purple-300 underline underline-offset-2 break-all">
              https://x.com/satyanadella/status/2108931348857827686
            </a>
          </p>

          <p className="text-gray-300 leading-relaxed">
            Related: <Link href="/blog/zero-trust-for-ai-agents" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">&quot;Zero Trust for AI Agents&quot;</Link> and <Link href="/govern" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">SatGate for enterprises</Link>.
          </p>
        </article>
      </div>
    </div>
  );
}
