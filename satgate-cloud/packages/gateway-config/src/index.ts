/**
 * @satgate/gateway-config
 * 
 * Shared gateway configuration utilities for SatGate Cloud.
 * Used by both control-plane (validation) and data-plane (runtime).
 */

export { loadConfig, parseYaml } from './loader';
export { validateSchema, validateCloudPolicy, ValidationResult } from './validator';
export { normalizeConfig } from './normalizer';
export { generateRouteSummary, generateTextSummary, RouteSummary } from './summary';
export { GatewayConfig, Route, Upstream, Policy, L402Policy } from './types';
export { schema } from './schema';

import { parseYaml } from './loader';
import { validateSchema } from './validator';
import { generateTextSummary } from './summary';
import { GatewayConfig } from './types';

/**
 * Validate a YAML config string
 */
export function validateConfig(yaml: string): { valid: boolean; errors?: string[]; config?: GatewayConfig } {
  const parseResult = parseYaml(yaml);
  if (!parseResult.success) {
    return { valid: false, errors: [parseResult.error || 'Invalid YAML'] };
  }
  
  const schemaResult = validateSchema(parseResult.data);
  if (!schemaResult.valid) {
    return { valid: false, errors: schemaResult.errors };
  }
  
  return { valid: true, config: parseResult.data as GatewayConfig };
}

/**
 * Print a text summary of a config
 */
export function printSummary(config: GatewayConfig): string {
  return generateTextSummary(config);
}

