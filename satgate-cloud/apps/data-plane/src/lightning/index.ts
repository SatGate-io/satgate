/**
 * Lightning provider factory
 * 
 * Selects provider based on environment configuration.
 */

import { logger } from '@satgate/common';
import { LightningProvider } from './types';
import { phoenixdProvider } from './phoenixd';
import { mockProvider, mockPayInvoice } from './mock';

export * from './types';
export { mockPayInvoice };

const LIGHTNING_ENABLED = process.env.LIGHTNING_ENABLED === 'true';
const LIGHTNING_BACKEND = process.env.LIGHTNING_BACKEND || 'mock';

let provider: LightningProvider | null = null;

/**
 * Get the configured Lightning provider
 */
export async function getLightningProvider(): Promise<LightningProvider> {
  if (provider) {
    return provider;
  }
  
  if (!LIGHTNING_ENABLED) {
    logger.info('[Lightning] Disabled, using mock provider');
    provider = mockProvider;
    return provider;
  }
  
  switch (LIGHTNING_BACKEND.toLowerCase()) {
    case 'phoenixd':
      if (await phoenixdProvider.isAvailable()) {
        logger.info('[Lightning] Using phoenixd provider');
        provider = phoenixdProvider;
      } else {
        logger.warn('[Lightning] phoenixd unavailable, falling back to mock');
        provider = mockProvider;
      }
      break;
      
    case 'mock':
    default:
      logger.info('[Lightning] Using mock provider');
      provider = mockProvider;
      break;
  }
  
  return provider;
}

/**
 * Get current provider (sync, must call getLightningProvider first)
 */
export function getCurrentProvider(): LightningProvider {
  if (!provider) {
    throw new Error('Lightning provider not initialized. Call getLightningProvider() first.');
  }
  return provider;
}

