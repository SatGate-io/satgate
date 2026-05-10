import CapabilityLifecycleDemo from './CapabilityLifecycleDemo';

export const metadata = {
  title: 'Capability Lifecycle Control | Issue, Delegate, Revoke, Prove',
  description: 'Observe, control, and prove the capability lifecycle: issue, delegate, attenuate, revoke, enforce caveats, cap child spend, and export Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/capability-lifecycle-demo' },
  keywords: [
    'capability lifecycle control',
    'agent capability delegation',
    'macaroon caveats for AI agents',
    'delegation depth AI agents',
    'revocable agent capability',
    'child spend caps',
    'Evidence Pack proof',
  ],
  openGraph: {
    title: 'SatGate Capability Lifecycle Control',
    description: 'Issue capability → delegate → attenuate → revoke → prove, with customer-visible caveats, delegation depth, child spend caps, next-request revocation, and Evidence Pack audit records.',
    url: 'https://satgate.io/capability-lifecycle-demo',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate Capability Lifecycle Control',
    description: 'Issue capability → delegate → attenuate → revoke → prove, with scoped authority, child spend caps, next-request revocation, and Evidence Pack proof.',
  },
};

export default function CapabilityLifecycleDemoPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Capability Lifecycle Control',
    url: 'https://satgate.io/capability-lifecycle-demo',
    description: metadata.description,
    datePublished: '2026-05-10',
    dateModified: '2026-05-10',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'agent capability lifecycle' },
      { '@type': 'Thing', name: 'macaroon caveats' },
      { '@type': 'Thing', name: 'delegation depth' },
      { '@type': 'Thing', name: 'child spend caps' },
      { '@type': 'Thing', name: 'next-request revocation' },
      { '@type': 'Thing', name: 'Evidence Pack proof' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is a capability lifecycle for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A capability lifecycle shows how agent authority is issued, delegated to a child, attenuated with caveats, revoked before the next request, and preserved as Evidence Pack proof.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do caveats translate into customer controls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Caveats are enforceable limits: tenant, task, route, tool, budget, child spend cap, delegation depth, expiry, and revocation rules evaluated before execution.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why does delegation depth matter?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Delegation depth helps limit governed sub-agent chains. A child can receive narrower authority and reaches the configured delegation limit unless policy explicitly allows another handoff.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Capability Lifecycle Control', item: 'https://satgate.io/capability-lifecycle-demo' },
    ],
  };

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <CapabilityLifecycleDemo />
    </>
  );
}
