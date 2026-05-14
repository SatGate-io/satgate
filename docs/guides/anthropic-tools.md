# Anthropic tools example

Wrap Anthropic tool use with the same issue/pay/verify primitive. SatGate governs authority outside the model provider boundary.

```ts
import Anthropic from "@anthropic-ai/sdk";
import { SatGate } from "@satgate/sdk";

const anthropic = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

const capability = await satgate.issue({
  agent: "analyst-agent",
  task: "query financial dataset",
  allow: ["tool:finance.query"],
  budgetUsd: 5,
  expiresIn: "30m",
});

const receipt = await satgate.pay({
  capability,
  upstream: "tool:finance.query",
  maxUsd: 0.25,
});

const verified = await satgate.verify(receipt);
if (verified.decision !== "allowed") throw new Error("Tool call denied by SatGate");

await anthropic.messages.create({
  model: "claude-sonnet-4-5",
  max_tokens: 512,
  messages: [{ role: "user", content: "Query the approved finance tool." }],
});
```

Related: [Raw HTTP](raw-http.md), [Capability schema](../reference/capability-schema.md).
