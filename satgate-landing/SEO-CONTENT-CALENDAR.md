# SEO Content Calendar — SatGate Blog

## Publishing Cadence
- **2 posts per week** (Tuesday + Thursday)
- Cross-post to dev.to with canonical URL back to satgate.io
- Target competitive keywords that competitors rank for

## Keyword Targets (prioritized)

### High Priority — Direct competitors ranking
- [x] "API gateway for AI agents" (Solo.io, Gravitee ranking) — `/blog/api-gateway-for-ai-agents` (3/12)
- [x] "AI agent spending limits" (TrueFoundry ranking) — `/blog/ai-agent-spending-limits` (3/10)
- [x] "LLM cost management" (Radicalbit, TDS ranking) — `/blog/llm-cost-management` (3/17)
- [x] "AI governance API teams" (Zuplo ranking) — `/blog/ai-governance-api-teams` (3/19)
- [x] "MCP gateway guide" (Composio ranking) — `/blog/mcp-gateway-guide` (3/24)
- [x] "API monetization AI" (general gap) — `/blog/api-monetization-ai` (3/26)

### Medium Priority — Category creation
- [x] "Macaroon tokens vs API keys" (own the comparison) — `/blog/macaroon-tokens-vs-api-keys` (3/31)
- [x] "L402 protocol explained" (own the protocol) — `/blog/l402-protocol-explained` (4/2)
- [x] "HTTP 402 Payment Required use cases" (technical SEO) — `/blog/http-402-payment-required-use-cases` (4/2)
- [x] "AI agent delegation patterns" (architectural content) — `/blog/deepmind-intelligent-delegation-satgate` (3/11)
- [x] "Capability-based security for APIs" (academic SEO) — `/blog/deepmind-intelligent-delegation-satgate` (3/11)
- [x] "Zero Trust for AI agents" (Wayne's LinkedIn angle) — `/blog/zero-trust-for-ai-agents` (4/3)

### Long-tail — Tutorial/how-to
- [x] "How to add budget limits to OpenAI API calls" — `/blog/how-to-add-budget-limits-to-openai-api-calls` (4/7)
- [x] "Cursor MCP proxy setup guide" — `/blog/cursor-mcp-proxy-setup-guide` (4/9)
- [ ] "Multi-agent cost tracking with SatGate"
- [ ] "API cost attribution by team and department"
- [ ] "How to monetize your API with Lightning payments"
- [ ] "Prevent AI agent runaway costs"
- [ ] "AI agent audit trail compliance"
- [ ] "Compare API gateways for AI workloads"

### Thought Leadership
- [ ] "The agent economy needs economic infrastructure"
- [ ] "Why identity-based security fails for autonomous agents"
- [ ] "From API keys to capability tokens: the next evolution"
- [ ] "The CFO's guide to AI agent cost control"
- [ ] "Security as a profit center (enterprise version)"

## Published Posts
- 2026-02-06: "Why Routing Isn't Governance"
- 2026-02-12: "Beyond Connection: Economic Governance in MCP" (also on dev.to)
- 2026-02-13: "How We Built Budget Enforcement for MCP"
- 2026-02-14: "Hard-Capping MCP Tool Spend"
- 2026-02-14: "Security as a Profit Center"
- 2026-03-05: "How to Control AI Agent API Costs" (also on dev.to)
- 2026-03-05: "What Is an Economic Firewall?" (also on dev.to)
- 2026-03-05: "MCP Budget Enforcement Guide" (also on dev.to)
- 2026-03-05: "Agent Swarms Cost Governance" (also on dev.to)
- 2026-03-12: "API Gateway for AI Agents: Why Traditional Gateways Fall Short" (also on dev.to)
- 2026-03-17: "LLM Cost Management: From Monitoring Dashboards to Real-Time Enforcement" (also on dev.to)
- 2026-03-19: "AI Governance for API Teams: Why Your Gateway Needs Policy, Not Just Routing" (also on dev.to)
- 2026-03-24: "MCP Gateway Guide: From Traffic Routing to Economic Governance" (also on dev.to)
- 2026-03-26: "API Monetization for AI: How to Charge Agents, Not Just Developers" (also on dev.to)
- 2026-03-31: "Macaroon Tokens vs API Keys" 
- 2026-04-02: "L402 Protocol Explained"
- 2026-04-02: "HTTP 402 Payment Required: The Dormant Status Code That Powers the Agent Economy" (also on dev.to)
- 2026-04-03: "Zero Trust for AI Agents: Why Identity-Based Security Collapses When Machines Call the Shots" (also on dev.to)
- 2026-04-09: "Cursor MCP Proxy Setup Guide: Add Budget Controls and Audit Trails to Your Tools"

## dev.to Account
- Handle: @mattdeangit
- API key: stored in memory (2026-02-12 daily note)
- Always set canonical_url to satgate.io
- Tags: ai, api, security, opensource, mcp, tutorial (pick 4 per post)

## Rules
- Every post must have: metadata export, SEO title, description (150-160 chars), keywords array, canonical URL
- Every post gets added to: blog index (app/blog/page.tsx), sitemap (app/sitemap.ts)
- Build + verify before commit
- Push to main (auto-deploys via Vercel)
- Cross-post to dev.to same day
