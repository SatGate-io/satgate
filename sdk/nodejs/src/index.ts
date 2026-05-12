/**
 * SatGate Gateway Node.js SDK (OSS)
 *
 * ## Admin Client (for operators)
 * 
 * @example
 * ```typescript
 * import { SatGateClient } from '@satgate/sdk';
 *
 * const client = new SatGateClient({
 *   url: 'http://localhost:8080',
 *   token: 'your-admin-token'
 * });
 *
 * // Mint a new token
 * const token = await client.tokens.mint({ scope: 'api:*', duration: '1h' });
 * console.log('Token:', token.token);
 * ```
 * 
 * ## Agent Client (for AI agents)
 * 
 * @example
 * ```typescript
 * import { SatGateAgentClient } from '@satgate/sdk';
 * 
 * // With admin token (auto-mints capability tokens)
 * const client = new SatGateAgentClient({
 *   gatewayUrl: 'http://localhost:8080',
 *   adminToken: 'your-admin-token'
 * });
 * 
 * // Make requests - token management is automatic
 * const response = await client.get('/api/data');
 * console.log(response.data);
 * ```
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
  DelegationNode,
} from './delegation';

// Errors
export {
  SatGateError,
  AuthenticationError,
  SatGateAuthError,
  NotFoundError,
  ValidationError,
  PaymentRequiredError,
  PaymentFailedError,
  BudgetExceededError,
  TokenExpiredError,
  DelegationError,
} from './errors';
