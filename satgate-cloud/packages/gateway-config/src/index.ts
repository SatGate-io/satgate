/**
 * @satgate/gateway-config
 * 
 * Shared gateway configuration utilities for SatGate Cloud.
 * Used by both control-plane (validation) and data-plane (runtime).
 */

export { loadConfig, parseYaml } from './loader';
export { validateSchema, validateCloudPolicy, ValidationResult } from './validator';
export { normalizeConfig } from './normalizer';
export { generateRouteSummary, RouteSummary } from './summary';
export { GatewayConfig, Route, Upstream, Policy, L402Policy } from './types';
export { schema } from './schema';

