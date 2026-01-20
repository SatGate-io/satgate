/**
 * SatGate Gateway Node.js SDK
 *
 * ## Admin Client (for operators)
 * 
 * @example
 * ```typescript
 * import { SatGateClient } from '@satgate/sdk';
 *
 * const client = new SatGateClient({
 *   url: 'http://localhost:9090',
 *   token: 'your-admin-token'
 * });
 *
 * // Mint a new token
 * const token = await client.tokens.mint({ scope: 'api:*', expiresIn: 3600 });
 * console.log('Token:', token.token);
 * ```
 * 
 * ## Agent Client (for AI agents)
 * 
 * @example
 * ```typescript
 * import { SatGateAgentClient } from '@satgate/sdk';
 * 
 * // Auto-detect environment (K8s/AWS)
 * const client = new SatGateAgentClient({
 *   gatewayUrl: 'https://gateway.internal',
 *   identity: 'auto'
 * });
 * 
 * // Make requests - 402 handling is automatic
 * const response = await client.get('/api/v1/data');
 * console.log(`Cost: $${response.cost}`);
 * ```
 * 
 * ## With Lightning wallet for external APIs
 * 
 * @example
 * ```typescript
 * import { SatGateAgentClient, LNDWallet } from '@satgate/sdk';
 * 
 * const client = new SatGateAgentClient({
 *   gatewayUrl: 'https://api.external.com',
 *   wallet: new LNDWallet({ macaroonPath: '~/.lnd/admin.macaroon' })
 * });
 * ```
 */

// Admin client
export { SatGateClient, SatGateClientOptions } from './client';

// Agent client
export {
  SatGateAgentClient,
  SatGateAgentClientOptions,
  AgentResponse,
  CachedToken,
  IdentityProvider,
  KubernetesIdentity,
  AWSIdentity,
  OIDCIdentity,
  LightningWallet,
  LNDWallet,
  LNDWalletOptions,
  AlbyWallet,
  AlbyWalletOptions,
} from './agent-client';

// Types
export { Token, TokenInfo, BanRecord, Stats, MintRequest, DelegateRequest } from './types';

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
  NotFoundError,
  ValidationError,
  PaymentRequiredError,
  PaymentFailedError,
  BudgetExceededError,
  TokenExpiredError,
  DelegationError,
} from './errors';

