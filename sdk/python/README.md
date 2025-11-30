# SatGate Python SDK

**Stripe for AI Agents** — L402 micropayments for APIs.

[![PyPI version](https://badge.fury.io/py/satgate.svg)](https://pypi.org/project/satgate/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Installation

```bash
pip install satgate
```

For LangChain integration:

```bash
pip install satgate[langchain]
```

## Quick Start

```python
from satgate import SatGateSession, LightningWallet

# Implement your wallet (or use a library like lndgrpc, pyln-client)
class MyWallet(LightningWallet):
    def pay_invoice(self, invoice: str) -> str:
        # Pay the invoice and return the preimage
        return your_lightning_node.pay(invoice)

# Create a session that auto-pays 402 responses
session = SatGateSession(wallet=MyWallet())

# Use like requests - payments happen automatically!
response = session.get("https://api.example.com/premium/data")
print(response.json())
```

## LangChain Integration

```python
from satgate import SatGateTool

# Create a tool your AI agent can use
tool = SatGateTool(wallet=MyWallet())

# Add to your LangChain agent
agent = initialize_agent(
    tools=[tool],
    llm=ChatOpenAI(),
    agent=AgentType.OPENAI_FUNCTIONS
)

# The agent can now access paid APIs automatically!
agent.run("Fetch premium market data from https://api.example.com/insights")
```

## How It Works

1. Your code makes a request to a paid API
2. The API returns `402 Payment Required` with a Lightning invoice
3. SatGate automatically pays the invoice via your wallet
4. The request is retried with the L402 token
5. You get your data ✨

## Links

- 🌐 Website: [satgate.io](https://satgate.io)
- 📖 Playground: [satgate.io/playground](https://satgate.io/playground)
- 💻 GitHub: [github.com/SatGate-io/satgate](https://github.com/SatGate-io/satgate)
- 📧 Contact: contact@satgate.io

## License

MIT License - © 2025 SatGate. Patent Pending.

