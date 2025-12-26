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

/**
 * Parse YAML string to config object
 */
export function parseYaml(yamlContent: string): unknown {
  return yaml.load(yamlContent);
}

/**
 * Load and validate config from YAML string
 */
export function loadConfig(yamlContent: string, options: LoadOptions = {}): GatewayConfig {
  // Parse YAML
  const raw = parseYaml(yamlContent);
  
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

