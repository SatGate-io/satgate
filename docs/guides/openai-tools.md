# OpenAI tools example

Wrap OpenAI tool calls with SatGate so the agent gets scoped authority before tool execution and a receipt after the decision.

```ts
import OpenAI from "openai";
import { SatGate } from "@satgate/sdk";

const openai = new OpenAI({ apiKey: process.env.OPENAI_API_KEY });
const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

const capability = await satgate.issue({
  agent: "support-agent",
  task: "lookup customer order status",
  allow: ["tool:orders.lookup"],
  budgetUsd: 2,
  expiresIn: "15m",
});

const receipt = await satgate.pay({
  capability,
  upstream: "tool:orders.lookup",
  maxUsd: 0.10,
});

const verified = await satgate.verify(receipt);
if (verified.decision !== "allowed") throw new Error("Tool call denied by SatGate");

await openai.responses.create({
  model: "gpt-4.1-mini",
  input: "Lookup order ORD-123",
  tools: [{ type: "function", name: "orders.lookup", parameters: { type: "object" } }],
});
```

Related: [Raw HTTP](raw-http.md), [Receipt schema](../reference/receipt-schema.md).
