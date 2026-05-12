/**
 * SatGate Node.js SDK
 *
 * ## Developer primitive: issue/pay/verify
 *
 * @example
 * ```typescript
 * import { SatGate } from '@satgate/sdk';
 *
 * const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });
 * const capability = await satgate.issue({
 *   task: 'summarize vendor invoice',
 *   agent: 'invoice-agent',
 *   allow: ['POST /v1/invoices/*'],
 *   budgetUsd: 0.25,
 *   expiresIn: '10m',
 * });
 * const receipt = await satgate.pay({
 *   upstream: 'https://api.vendor.test/v1/invoices/42',
 *   capability,
 *   maxUsd: 0.10,
 * });
 * const verified = await satgate.verify(receipt);
 * console.log(verified.decision, verified.evidencePackId ?? verified.evidence_pack_id);
 * ```
 *
 * ## Compatibility: lower-level OSS Gateway clients
 *
 * SatGateClient and SatGateAgentClient preserve existing self-hosted gateway token,
 * delegation, and paid-rail APIs. New Cloud/private-beta app examples should lead
 * with issue/pay/verify.
 */

// SatGate Cloud private-beta facade
export {
  SatGate,
  SatGateOptions,
  CapabilityRequest,
  PayRequest,
  SatGateReceipt,
} from './private-beta';

// Admin client
export { SatGateClient, SatGateClientOptions } from './client';

// Agent client
export {
  SatGateAgentClient,
  SatGateAgentClientOptions,
  AgentResponse,
  CachedToken,
  LightningWallet,
  LNDWallet,
  LNDWalletOptions,
  AlbyWallet,
  AlbyWalletOptions,
} from './agent-client';

// Types
export {
  Token,
  TokenInfo,
  BanRecord,
  GraphData,
  GraphStats,
  DelegateRequest,
  BanRequest,
  PolicyKind,
  RouteConfig,
} from './types';

// Delegation helpers
export {
  Caveats,
  DelegationBuilder,
  DelegationPatterns,
  delegate,
  parseCaveats,
  isMoreRestrictive,
} from './delegation';

// Errors
export {
  SatGateError,
  AuthenticationError,
  NotFoundError,
  PaymentRequiredError,
  PaymentFailedError,
  BudgetExceededError,
  TokenExpiredError,
  DelegationError,
  SatGateAuthError,
} from './errors';
