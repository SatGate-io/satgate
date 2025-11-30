
/**
 * SatGate Client SDK
 * Handles L402 payment loops automatically.
 * 
 * IMPORTANT: Aperture expects Authorization header format:
 *   LSAT <macaroon>:<preimage>
 * Where macaroon is base64 and preimage is hex.
 * Do NOT double-encode!
 */

export class SatGateClient {
  constructor(config = {}) {
    this.wallet = this._resolveWallet(config.wallet);
    this.baseUrl = config.baseUrl || '';
    this.onChallenge = config.onChallenge || (() => {});
    this.onPaymentStart = config.onPaymentStart || (() => {});
    this.onPayment = config.onPayment || (() => {});
  }

  _resolveWallet(walletConfig) {
    if (!walletConfig || walletConfig === 'webln' || walletConfig === 'alby') {
      return new WebLNWallet();
    }
    if (typeof walletConfig.payInvoice === 'function') {
      return walletConfig;
    }
    throw new Error('Unsupported wallet configuration');
  }

  async fetch(input, init = {}) {
    const url = this.baseUrl ? new URL(input, this.baseUrl).toString() : input;
    
    // Disable caching to avoid CORS preflight cache issues
    const fetchInit = {
      ...init,
      cache: 'no-store',
      mode: 'cors'
    };
    
    let response = await fetch(url, fetchInit);

    if (response.status === 402) {
      return this._handlePaymentChallenge(response, url, fetchInit);
    }

    return response;
  }

  async get(url, init = {}) {
    return this.fetch(url, { ...init, method: 'GET' });
  }

  async post(url, body, init = {}) {
    return this.fetch(url, { 
      ...init, 
      method: 'POST',
      body: JSON.stringify(body),
      headers: {
        'Content-Type': 'application/json',
        ...init.headers
      }
    });
  }

  async _handlePaymentChallenge(response, url, init) {
    const authHeader = response.headers.get('WWW-Authenticate');
    if (!authHeader) throw new Error('402 Response missing WWW-Authenticate header');

    const challenge = this._parseL402Header(authHeader);
    if (!challenge) throw new Error('Invalid L402 WWW-Authenticate header format');

    console.log(`[SatGate] Payment Required`);
    console.log(`[SatGate] Invoice: ${challenge.invoice.substring(0, 40)}...`);
    console.log(`[SatGate] Macaroon: ${challenge.macaroon.substring(0, 40)}...`);
    
    this.onChallenge({ invoice: challenge.invoice, macaroon: challenge.macaroon });

    try {
      // Signal that payment is starting (wallet popup about to appear)
      this.onPaymentStart({ invoice: challenge.invoice });
      
      const paymentResult = await this.wallet.payInvoice(challenge.invoice);
      const preimage = paymentResult.preimage;

      console.log(`[SatGate] Payment successful!`);
      console.log(`[SatGate] Preimage: ${preimage}`);

      this.onPayment({ url, amount: null, preimage });

      // CRITICAL: Aperture expects this EXACT format:
      // Authorization: LSAT <base64_macaroon>:<hex_preimage>
      // The macaroon from the header is ALREADY base64 encoded.
      // The preimage from WebLN is hex.
      // Do NOT re-encode anything!
      
      const authValue = `LSAT ${challenge.macaroon}:${preimage}`;
      
      console.log(`[SatGate] Auth header: LSAT ${challenge.macaroon.substring(0, 30)}...:${preimage.substring(0, 16)}...`);

      let retryResponse;
      try {
        retryResponse = await fetch(url, {
          ...init,
          method: 'GET',
          headers: {
            ...init.headers,
            'Authorization': authValue
          }
        });
      } catch (fetchError) {
        console.error('[SatGate] Retry fetch failed:', fetchError);
        console.error('[SatGate] This is usually a CORS or network error.');
        console.error('[SatGate] URL:', url);
        console.error('[SatGate] Auth header length:', authValue.length);
        throw new Error(`Retry request failed: ${fetchError.message}. Check browser console for CORS errors.`);
      }

      console.log(`[SatGate] Retry response: ${retryResponse.status}`);

      return retryResponse;

    } catch (error) {
      console.error('[SatGate] Error:', error);
      throw error;
    }
  }

  _parseL402Header(header) {
    // Aperture returns TWO WWW-Authenticate headers (LSAT and L402)
    // Browser combines them: "LSAT macaroon="...", invoice="...", L402 macaroon="...", invoice="...""
    // We extract the first valid pair.
    
    const macaroonMatch = header.match(/macaroon="([^"]+)"/);
    const invoiceMatch = header.match(/invoice="([^"]+)"/);

    if (macaroonMatch && invoiceMatch) {
      return {
        macaroon: macaroonMatch[1],
        invoice: invoiceMatch[1]
      };
    }
    return null;
  }
}

class WebLNWallet {
  async payInvoice(invoice) {
    if (typeof window === 'undefined' || !window.webln) {
      throw new Error('WebLN not available. Install Alby or use a supported environment.');
    }
    await window.webln.enable();
    return await window.webln.sendPayment(invoice);
  }
}
