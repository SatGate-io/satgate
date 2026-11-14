export const tradingCopy = {
  T3: 'Example: Your agent has a $100 limit per order and sends a $5,000 order by mistake. SatGate refuses it, the order never reaches your broker, and you get a signed receipt of the refusal.',
  T4: 'Before it places an order, your agent can ask SatGate whether it would go through and how much of its limit is left.',
  T5: {
    title: 'How limits are counted',
    lines: [
      'An order counts against your limits when your agent places it, at the most it could cost.',
      "Cancelling an order doesn't give its amount back.",
      'Stock orders go in dollar amounts. An order for a number of shares must be a limit buy for the day, and counts as shares times the limit price. Any other order in shares is refused.',
      'Weekly limits run Monday to Sunday.',
    ],
  },
  T6: {
    intro: 'Running agents for clients or a team? ',
    link: 'See plans for teams',
    punctuation: '.',
    href: '/pricing',
  },
  T7: "You'll find each receipt under Receipts in your SatGate dashboard, and your agent's answer links to it.",
  T8: "Robinhood's side, as of November 13, 2026.",
  T9: {
    text: 'Powered by the SatGate economic firewall.',
    href: '/platform',
  },
} as const;
