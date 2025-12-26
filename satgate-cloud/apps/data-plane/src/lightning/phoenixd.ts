/**
 * Phoenixd Lightning provider
 * 
 * phoenixd is a lightweight Lightning node by ACINQ.
 * https://phoenix.acinq.co/server
 */

import * as crypto from 'crypto';
import { logger } from '@satgate/common';
import { LightningProvider, Invoice, InvoiceStatus, CreateInvoiceParams } from './types';

const PHOENIXD_URL = process.env.PHOENIXD_URL || 'http://localhost:9740';
const PHOENIXD_PASSWORD = process.env.PHOENIXD_PASSWORD || '';

/**
 * Make authenticated request to phoenixd
 */
async function phoenixdRequest<T>(
  method: string,
  path: string,
  body?: Record<string, any>
): Promise<T> {
  const url = `${PHOENIXD_URL}${path}`;
  
  // phoenixd uses HTTP Basic auth with empty username
  const auth = Buffer.from(`:${PHOENIXD_PASSWORD}`).toString('base64');
  
  const headers: Record<string, string> = {
    'Authorization': `Basic ${auth}`,
  };
  
  let requestBody: string | undefined;
  if (body) {
    // phoenixd expects form-urlencoded for POST
    headers['Content-Type'] = 'application/x-www-form-urlencoded';
    const params = new URLSearchParams();
    for (const [k, v] of Object.entries(body)) {
      params.set(k, String(v));
    }
    requestBody = params.toString();
  }
  
  const res = await fetch(url, {
    method,
    headers,
    body: requestBody,
  });
  
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`phoenixd ${method} ${path} failed: ${res.status} ${text}`);
  }
  
  return res.json() as Promise<T>;
}

/**
 * Phoenixd provider implementation
 */
export const phoenixdProvider: LightningProvider = {
  name: 'phoenixd',
  
  async isAvailable(): Promise<boolean> {
    try {
      // Check node info endpoint
      await phoenixdRequest<any>('GET', '/getinfo');
      return true;
    } catch (err) {
      logger.warn('phoenixd not available', { error: (err as Error).message });
      return false;
    }
  },
  
  async createInvoice(params: CreateInvoiceParams): Promise<Invoice> {
    const { amountSats, memo, expirySecs = 3600 } = params;
    
    // phoenixd /createinvoice endpoint
    // https://phoenix.acinq.co/server/api
    const response = await phoenixdRequest<{
      amountSat: number;
      paymentHash: string;
      serialized: string;  // BOLT11 invoice
    }>('POST', '/createinvoice', {
      amountSat: amountSats,
      description: memo,
      // Note: phoenixd may not support custom expiry
    });
    
    const expiresAt = Math.floor(Date.now() / 1000) + expirySecs;
    
    return {
      paymentHash: response.paymentHash,
      paymentRequest: response.serialized,
      amountSats: response.amountSat,
      memo,
      expiresAt,
    };
  },
  
  async getInvoiceStatus(paymentHash: string): Promise<InvoiceStatus> {
    try {
      // phoenixd /getincomingpayment endpoint
      const response = await phoenixdRequest<{
        paymentHash: string;
        preimage: string;
        isPaid: boolean;
        receivedSat: number;
        completedAt?: number;
      }>('GET', `/getincomingpayment?paymentHash=${paymentHash}`);
      
      return {
        paid: response.isPaid,
        preimage: response.isPaid ? response.preimage : undefined,
        paidAt: response.completedAt,
      };
    } catch (err) {
      // Invoice not found or not paid yet
      return { paid: false };
    }
  },
};

export default phoenixdProvider;

