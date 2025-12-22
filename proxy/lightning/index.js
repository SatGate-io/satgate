/**
 * SatGate Multi-Backend Lightning Module
 * 
 * Provides a unified interface for multiple Lightning backends.
 * Configure via environment variables or programmatically.
 * 
 * Environment Variables:
 *   LIGHTNING_PROVIDER=phoenixd|lnd|opennode|aperture
 *   
 *   For phoenixd:
 *     PHOENIXD_URL=http://localhost:9740
 *     PHOENIXD_PASSWORD=your-password
 *   
 *   For lnd:
 *     LND_URL=https://localhost:8080
 *     LND_MACAROON=hex-encoded-macaroon
 *   
 *   For opennode:
 *     OPENNODE_API_KEY=your-api-key
 *     OPENNODE_ENV=live|dev
 *   
 *   For aperture (legacy):
 *     Continue using LNC env vars - Aperture handles Lightning
 */

const { LightningProvider } = require('./provider');
const { PhoenixdProvider } = require('./phoenixd');
const { LNDProvider } = require('./lnd');
const { OpenNodeProvider } = require('./opennode');

/**
 * Create a Lightning provider from environment variables
 * @returns {LightningProvider}
 */
function createProvider() {
  const providerType = (process.env.LIGHTNING_PROVIDER || 'aperture').toLowerCase();

  console.log(`[Lightning] Initializing provider: ${providerType}`);

  switch (providerType) {
    case 'phoenixd':
    case 'phoenix':
      if (!process.env.PHOENIXD_PASSWORD) {
        console.warn('[Lightning] PHOENIXD_PASSWORD not set');
      }
      return new PhoenixdProvider({
        url: process.env.PHOENIXD_URL || 'http://localhost:9740',
        password: process.env.PHOENIXD_PASSWORD || '',
      });

    case 'lnd':
    case 'lnd-rest':
      if (!process.env.LND_MACAROON) {
        throw new Error('LND_MACAROON environment variable required for LND provider');
      }
      return new LNDProvider({
        url: process.env.LND_URL || 'https://localhost:8080',
        macaroon: process.env.LND_MACAROON,
        cert: process.env.LND_CERT || null,
      });

    case 'opennode':
      if (!process.env.OPENNODE_API_KEY) {
        throw new Error('OPENNODE_API_KEY environment variable required for OpenNode provider');
      }
      return new OpenNodeProvider({
        apiKey: process.env.OPENNODE_API_KEY,
        environment: process.env.OPENNODE_ENV || 'live',
      });

    case 'aperture':
    case 'lnc':
    default:
      // Aperture mode - Lightning is handled by Aperture via LNC
      // Return a stub provider that indicates Aperture is handling L402
      return new ApertureStubProvider();
  }
}

/**
 * Stub provider for when Aperture handles Lightning
 * This indicates that L402 challenges should be left to Aperture
 */
class ApertureStubProvider extends LightningProvider {
  constructor() {
    super({});
    this.name = 'aperture';
    this.isStub = true;
  }

  async createInvoice() {
    throw new Error(
      'Invoice creation handled by Aperture. ' +
      'Set LIGHTNING_PROVIDER to use native L402.'
    );
  }

  async checkPayment() {
    throw new Error('Payment verification handled by Aperture.');
  }

  async getStatus() {
    return {
      ok: true,
      info: {
        provider: 'aperture',
        mode: 'passthrough',
        note: 'Lightning handled by Aperture via LNC',
      },
    };
  }
}

// Singleton instance
let _provider = null;

/**
 * Get the configured Lightning provider (singleton)
 * @returns {LightningProvider}
 */
function getProvider() {
  if (!_provider) {
    _provider = createProvider();
  }
  return _provider;
}

/**
 * Check if we're in Aperture mode (L402 handled externally)
 * @returns {boolean}
 */
function isApertureMode() {
  const provider = getProvider();
  return provider.isStub === true;
}

module.exports = {
  // Factory
  createProvider,
  getProvider,
  isApertureMode,
  
  // Classes for direct use
  LightningProvider,
  PhoenixdProvider,
  LNDProvider,
  OpenNodeProvider,
  ApertureStubProvider,
};

