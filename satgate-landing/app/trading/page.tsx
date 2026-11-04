import type { Metadata } from 'next';
import Link from 'next/link';
import Image from 'next/image';
import { ArrowRight, Receipt, Users } from 'lucide-react';

const title = 'Financial guardrail for your AI agents';
const tagline = 'Control what agents can do and spend. Prove every action.';
const lead =
  'Your limits sit outside the AI, so it can\'t loosen them or talk its way past them. SatGate checks them before an order reaches your broker.';
const worksToday = 'Works with Robinhood today, with more brokerages to follow.';
const metaTitle = 'Dollar limits and signed receipts for AI trading agents | SatGate';
const description =
  'Dollar limits for your AI trading agent. Approve large orders, give each agent its own key, get signed receipts, and stop any agent with one button.';

export const metadata: Metadata = {
  title: { absolute: metaTitle },
  description,
  alternates: { canonical: 'https://satgate.io/trading' },
  openGraph: {
    title: metaTitle,
    description,
    url: 'https://satgate.io/trading',
    type: 'website',
    images: [{ url: '/og.png', width: 1200, height: 630, alt: 'SatGate: limits for your AI agent' }],
  },
  twitter: {
    card: 'summary_large_image',
    title: metaTitle,
    description,
  },
};

const signupUrl = 'https://cloud.satgate.io/cloud/signup?intent=trading';

const whatItDoes = [
  { icon: Users, text: 'Each agent gets its own key.' },
  { icon: Receipt, text: 'You can stop your agent with one button.' },
];

const checkerUrl = 'https://github.com/SatGate-io/satgate/tree/main/tools';

const auditTitle = 'Built for audit';

const auditItems = [
  {
    lead: 'Every action on the record.',
    text: 'Every action your agent takes through SatGate is recorded in a signed receipt, including orders SatGate refused.',
  },
  {
    lead: 'Checked before the money moves.',
    text: 'SatGate checks each order against your limits before it reaches your broker.',
  },
  {
    lead: 'No need to trust the AI company.',
    text: (
      <>
        Your agent never writes its own record. Anyone with your receipts can check them against SatGate&apos;s published keys using{' '}
        <a href={checkerUrl} className="text-cyan-300 underline underline-offset-2 hover:text-cyan-200">
          our open-source checker
        </a>
        .
      </>
    ),
  },
];

const auditDetail =
  'SatGate\'s receipts show what SatGate decided. Your broker\'s statement shows what was filled. Calls from an agent you\'ve stopped are turned away.';

const generalTitle = 'What SatGate does and doesn\'t do';

const generalItems = [
  'SatGate does not trade for anyone or open accounts. Your own AI agent trades in your own account. SatGate is your tool that sets limits and keeps receipts, like a firewall.',
  'SatGate signs a receipt when it refuses your agent\'s request for breaking your rules, with the tool name and a refusal code.',
  'You can stop your agent in SatGate at any time.',
  'SatGate\'s bill is a separate subscription and never comes out of your brokerage account.',
];

const robinhoodTitle = 'SatGate with Robinhood';
const robinhoodIntro = 'Robinhood is the brokerage SatGate works with today. Everything in this section is about Robinhood.';

const comparisonTitle = 'What Robinhood gives you, and what SatGate adds';
const comparisonIntro =
  'Robinhood gives your agent its own account that you fund, and shows its activity in the Robinhood app. SatGate adds the following on top.';

const comparisonRows = [
  {
    topic: 'Which accounts your agent can read',
    robinhood: 'Your agent can read all your Robinhood accounts, with their numbers, positions and balances.',
    satgate:
      'Your agent can see only the Agentic account. SatGate refuses the request that lists all your accounts, and any request that names another account.',
  },
  {
    topic: 'How much your agent can trade',
    robinhood: 'You choose how much money to put in the agent\'s account, and the agent can trade with what is in it.',
    satgate:
      'You set the most it can trade, in dollars, per order, per day and per week, and an order over a limit is refused before it reaches Robinhood. Your agent can trade stocks and crypto, in dollar amounts.',
  },
  {
    topic: 'Approving orders',
    robinhood:
      'Robinhood is still rolling out trade approvals for agents you connect from other apps, and not every customer has them. With them on, you review and place each trade your agent proposes in the Robinhood app.',
    satgate:
      'You pick a dollar amount. An order over it waits for your yes by email or in SatGate, and smaller orders go through within your limits.',
  },
  {
    topic: 'Orders that were stopped',
    robinhood: 'Orders your agent places show up in the Activity section of its account in the Robinhood app.',
    satgate:
      'Every action your agent takes is recorded in a signed receipt, including orders SatGate refuses for being over a limit, outside the Agentic account or not allowed.',
  },
  {
    topic: 'More than one agent',
    robinhood: 'Every agent you connect gets the same access. Its one safety setting is trade approvals.',
    satgate:
      'Connect agents from the AI apps you already use and give each its own limits. One can look only while another makes small trades, all in your Agentic account.',
  },
];

const canAndCant = [
  'You sign in on Robinhood\'s own page. SatGate never sees your Robinhood password.',
  'SatGate can\'t move money. Robinhood\'s agent connection has no deposit, withdrawal, transfer or bank-link action, and trades can only happen in the Agentic account you fund.',
  'SatGate stores the access Robinhood gives it, encrypted, and uses it to pass along the calls your agent is allowed to make.',
  'On its own, SatGate only asks Robinhood for its list of tools, plus one read of your account list during setup, when you click, to find your Agentic account. It never reads your balances or places orders by itself.',
  'Robinhood lets a connected agent read all your Robinhood accounts. SatGate locks yours to the Agentic account.',
  'We don\'t keep Robinhood\'s replies, so we don\'t keep your balances or positions.',
  'You can also disconnect SatGate in the Robinhood app at any time.',
];

export default function TradingPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: metaTitle,
    url: 'https://satgate.io/trading',
    description,
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Trading', item: 'https://satgate.io/trading' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <nav className="border-b border-gray-800">
        <div className="mx-auto flex h-16 max-w-4xl items-center px-6">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="h-7 w-7 sm:h-8 sm:w-8" />
            <span className="whitespace-nowrap text-lg font-bold text-white sm:text-xl">SatGate</span>
          </Link>
        </div>
      </nav>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_24%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-4xl px-6 py-16 md:py-24">
          <h1 className="mb-4 text-4xl font-extrabold tracking-tight md:text-6xl">{title}</h1>
          <p className="mb-4 text-2xl font-semibold text-white md:text-3xl">{tagline}</p>
          <p className="mb-8 text-lg text-gray-300 md:text-xl">{lead}</p>
          <ul className="space-y-4">
            {whatItDoes.map(({ icon: Icon, text }) => (
              <li key={text} className="flex items-start gap-3 text-lg text-gray-300 md:text-xl">
                <Icon size={22} className="mt-1 shrink-0 text-cyan-300" aria-hidden="true" />
                <span>{text}</span>
              </li>
            ))}
          </ul>
          <div className="mt-8">
            <a
              href={signupUrl}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 px-8 py-3 font-bold text-white shadow-lg shadow-purple-500/20 transition hover:from-purple-500 hover:to-cyan-500"
            >
              Set up my agent&apos;s limits <ArrowRight size={16} />
            </a>
            <p className="mt-3 text-sm text-gray-400">Free for 30 days, no card needed. Continue with Personal for <span className="whitespace-nowrap">$9.99 a month.</span></p>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 pt-16" aria-labelledby="how-it-works-title">
        <div className="rounded-lg border border-gray-800 bg-black/60 p-6 md:p-8">
          <h2 id="how-it-works-title" className="mb-5 text-2xl font-bold text-white">How it works</h2>
          <ul className="list-disc space-y-3 pl-5 text-gray-300 marker:text-cyan-300">
            <li>
              <strong className="font-semibold text-white">What can my agent do?</strong>{' '}
              Only the tools and accounts you allow. Everything else is blocked before it reaches your broker.
            </li>
            <li>
              <strong className="font-semibold text-white">How much can it spend?</strong>{' '}
              A hard dollar limit it can&apos;t go past. Orders above an amount you choose wait for your OK by email.
            </li>
            <li>
              <strong className="font-semibold text-white">Can I prove what happened?</strong>{' '}
              Every action leaves a signed receipt. You, your accountant or anyone you show it to can check it against SatGate&apos;s published keys, with no need to trust the AI company.
            </li>
          </ul>
          <p className="mt-6 text-gray-300">Example: Say your agent tries to buy $2 of Bitcoin and crypto isn&apos;t on its list. SatGate refuses it, the order never reaches your broker, and you get a signed receipt of the refusal.</p>
        </div>
      </section>

      <div className="border-b border-gray-900">
        <p className="mx-auto max-w-4xl px-6 py-4 text-base font-medium text-cyan-300">{worksToday}</p>
      </div>

      <section className="mx-auto max-w-4xl px-6 pt-16" aria-labelledby="audit-title">
        <div className="rounded-lg border border-gray-800 bg-black/60 p-6 md:p-8">
          <h2 id="audit-title" className="mb-3 text-2xl font-bold text-white">{auditTitle}</h2>
          <ul className="list-disc space-y-3 pl-5 text-gray-300 marker:text-cyan-300">
            {auditItems.map(({ lead, text }) => (
              <li key={lead}>
                <span className="font-semibold text-white">{lead}</span> {text}
              </li>
            ))}
          </ul>
          <p className="mt-6 text-sm text-gray-400">{auditDetail}</p>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 pt-16" aria-labelledby="general-title">
        <div className="rounded-lg border border-gray-800 bg-black/60 p-6 md:p-8">
          <h2 id="general-title" className="mb-5 text-2xl font-bold text-white">{generalTitle}</h2>
          <ul className="list-disc space-y-3 pl-5 text-gray-300 marker:text-cyan-300">
            {generalItems.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 pt-16" aria-labelledby="robinhood-title">
        <h2 id="robinhood-title" className="mb-3 text-3xl font-bold text-white">{robinhoodTitle}</h2>
        <p className="mb-8 text-gray-300">{robinhoodIntro}</p>
        <div className="rounded-lg border border-gray-800 bg-black/60 p-6 md:p-8">
          <h3 id="comparison-title" className="mb-3 text-2xl font-bold text-white">{comparisonTitle}</h3>
          <p className="mb-6 text-gray-300">{comparisonIntro}</p>
          <div className="hidden gap-8 border-b border-gray-800 pb-3 text-sm font-semibold text-gray-400 md:grid md:grid-cols-2" aria-hidden="true">
            <span>Robinhood on its own</span>
            <span>With SatGate</span>
          </div>
          <ul>
            {comparisonRows.map((row) => (
              <li key={row.topic} className="border-b border-gray-800 py-5 last:border-b-0 last:pb-0">
                <h4 className="mb-3 text-base font-semibold text-white">{row.topic}</h4>
                <div className="grid gap-4 md:grid-cols-2 md:gap-8">
                  <div>
                    <p className="mb-1 text-sm font-semibold text-gray-400 md:hidden">Robinhood on its own</p>
                    <p className="text-gray-300">{row.robinhood}</p>
                  </div>
                  <div className="border-l-2 border-cyan-300 pl-4">
                    <p className="mb-1 text-sm font-semibold text-cyan-300 md:hidden">With SatGate</p>
                    <p className="text-gray-100">{row.satgate}</p>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <div className="mt-8 rounded-lg border border-gray-800 bg-black/60 p-6 md:p-8">
          <h3 className="mb-5 text-2xl font-bold text-white">What SatGate can and can&apos;t do with your Robinhood connection</h3>
          <ul className="list-disc space-y-3 pl-5 text-gray-300 marker:text-cyan-300">
            {canAndCant.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </div>

        <p className="mt-6 text-sm text-gray-500">SatGate is not affiliated with or endorsed by Robinhood.</p>
      </section>

      <section className="mx-auto max-w-4xl px-6 py-16">
        <div>
          <a
            href={signupUrl}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 px-8 py-3 font-bold text-white shadow-lg shadow-purple-500/20 transition hover:from-purple-500 hover:to-cyan-500"
          >
            Set up my agent&apos;s limits <ArrowRight size={16} />
          </a>
          <p className="mt-3 text-sm text-gray-400">Free for 30 days, no card needed. Continue with Personal for <span className="whitespace-nowrap">$9.99 a month.</span></p>
          <p className="mt-4 text-sm text-gray-500">
            SatGate sets limits and keeps receipts. It does not pick trades or give investment advice.
          </p>
        </div>
      </section>
    </main>
  );
}
