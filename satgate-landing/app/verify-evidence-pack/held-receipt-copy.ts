// Held copy: replace only after proof of every action is enabled in production
// and the replacement has been reviewed against that shipped code.
export const HELD_RECEIPT_COPY = {
  lead: 'You can check a SatGate receipt here in a few seconds, without taking our word for it.',
  stepOne: 'In your SatGate dashboard, open Activity, pick an entry and choose Download. You get a small file called an Evidence Pack. It holds the signed receipt.',
  coverageHeading: 'What gets a receipt',
  coverage: [
    "SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a permission, and when an agent spends from a budget in Control mode, where SatGate enforces your limits. If SatGate can't make the receipt, the request is still stopped.",
    "Not every request gets a receipt. A refusal for going over a rate limit doesn't get one, and neither does a request that never goes through SatGate.",
    "A receipt shows which tool or service your agent called, whether SatGate allowed or refused it, and a refusal code. It doesn't name the agent or include what the agent sent.",
  ],
} as const;
