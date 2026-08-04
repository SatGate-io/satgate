import { redirect } from 'next/navigation';

export const metadata = {
  title: 'Governed Paid API Access for Agents',
  description: 'Redirects to the buyer-safe SatGate guide for delegated paid API access by agents.',
  alternates: { canonical: 'https://satgate.io/robot-customer-payments' },
};

export default function LegacyPaidAccessRedirect() {
  redirect('/paid-agent-payments');
}
