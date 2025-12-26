/**
 * Config loader - parses YAML and validates
 */

import * as yaml from 'js-yaml';
import { GatewayConfig } from './types';
import { validateSchema, validateCloudPolicy } from './validator';
import { normalizeConfig } from './normalizer';

export interface LoadOptions {
  /** Skip Cloud policy validation (for self-hosted mode) */
  skipCloudPolicy?: boolean;
}

export type ParseResult = 
  | { success: true; data: unknown }
  | { success: false; error: string };

/**
 * Parse YAML string to config object
 */
export function parseYaml(yamlContent: string): ParseResult {
  try {
    const data = yaml.load(yamlContent);
    return { success: true, data };
  } catch (err) {
    return { success: false, error: (err as Error).message };
  }
}

/**
 * Load and validate config from YAML string
 */
export function loadConfig(yamlContent: string, options: LoadOptions = {}): GatewayConfig {
  // Parse YAML
  const parseResult = parseYaml(yamlContent);
  
  if (!parseResult.success) {
    throw new Error(`YAML parse error: ${parseResult.error}`);
  }
  
  const raw = parseResult.data;
  
  if (!raw || typeof raw !== 'object') {
    throw new Error('Invalid config: must be a YAML object');
  }
  
  // Schema validation
  const schemaResult = validateSchema(raw);
  if (!schemaResult.valid) {
    throw new Error(`Schema validation failed:\n${schemaResult.errors.join('\n')}`);
  }
  
  // Normalize (apply defaults, etc.)
  const config = normalizeConfig(raw as GatewayConfig);
  
  // Cloud policy validation (unless skipped)
  if (!options.skipCloudPolicy) {
    const cloudResult = validateCloudPolicy(config);
    if (!cloudResult.valid) {
      throw new Error(`Cloud policy validation failed:\n${cloudResult.errors.join('\n')}`);
    }
  }
  
  return config;
}

