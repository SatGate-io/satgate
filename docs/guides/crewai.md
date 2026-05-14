# CrewAI example

Use SatGate at the tool boundary. CrewAI decides task orchestration; SatGate governs whether an agent has authority to spend, call, delegate, or reach an upstream tool.

```python
import os
from satgate import SatGate

satgate = SatGate(api_key=os.environ["SATGATE_API_KEY"])

def governed_tool_call(agent: str, task: str, upstream: str, max_usd: float):
    capability = satgate.issue(
        agent=agent,
        task=task,
        allow=[upstream],
        budget_usd=max_usd,
        expires_in="30m",
    )
    receipt = satgate.pay(upstream=upstream, capability=capability, max_usd=max_usd)
    verified = satgate.verify(receipt)
    if verified.decision != "allowed":
        raise RuntimeError(f"SatGate denied tool call: {verified.decision_reason}")
    return verified

# Inside a CrewAI tool wrapper:
proof = governed_tool_call(
    agent="research-crew.market-agent",
    task="fetch market price evidence",
    upstream="api:prices.read",
    max_usd=1.00,
)
print(proof.evidence_pack_id)
```

Related: [LangChain integration](langchain-integration.md), [Raw HTTP](raw-http.md), [Receipt schema](../reference/receipt-schema.md).
