/**
 * Mock Lightning provider for testing
 * 
 * Creates fake invoices that can be "paid" by providing any preimage.
 */

import * as crypto from 'crypto';
import { logger } from '@satgate/common';
import { LightningProvider, Invoice, InvoiceStatus, CreateInvoiceParams } from './types';

// In-memory storage for mock invoices
const mockInvoices = new Map<string, {
  invoice: Invoice;
  paid: boolean;
  preimage?: string;
  paidAt?: number;
}>();

/**
 * Generate a fake BOLT11 invoice
 * Starts with "lnbc" + amount + random data
 */
function generateFakeBolt11(amountSats: number): string {
  const prefix = amountSats >= 1000 
    ? `lnbc${Math.floor(amountSats / 1000)}m`  // milli-satoshi notation
    : `lnbc${amountSats}n`;                      // nano notation
  const randomPart = crypto.randomBytes(50).toString('base64url');
  return prefix + randomPart.slice(0, 100);
}

/**
 * Mock provider implementation
 */
export const mockProvider: LightningProvider = {
  name: 'mock',
  
  async isAvailable(): Promise<boolean> {
    return true;  // Always available
  },
  
  async createInvoice(params: CreateInvoiceParams): Promise<Invoice> {
    const { amountSats, memo, expirySecs = 3600 } = params;
    
    const paymentHash = crypto.randomBytes(32).toString('hex');
    const paymentRequest = generateFakeBolt11(amountSats);
    const expiresAt = Math.floor(Date.now() / 1000) + expirySecs;
    
    const invoice: Invoice = {
      paymentHash,
      paymentRequest,
      amountSats,
      memo,
      expiresAt,
    };
    
    mockInvoices.set(paymentHash, {
      invoice,
      paid: false,
    });
    
    logger.debug('[Mock LN] Created invoice', { 
      paymentHash: paymentHash.slice(0, 16) + '...',
      amountSats,
    });
    
    return invoice;
  },
  
  async getInvoiceStatus(paymentHash: string): Promise<InvoiceStatus> {
    const record = mockInvoices.get(paymentHash);
    
    if (!record) {
      return { paid: false };
    }
    
    return {
      paid: record.paid,
      preimage: record.preimage,
      paidAt: record.paidAt,
    };
  },
};

/**
 * Mark a mock invoice as paid (for testing)
 */
export function mockPayInvoice(paymentHash: string, preimage: string): boolean {
  const record = mockInvoices.get(paymentHash);
  if (!record) {
    return false;
  }
  
  // Verify preimage hashes to payment hash
  const computedHash = crypto.createHash('sha256')
    .update(Buffer.from(preimage, 'hex'))
    .digest('hex');
  
  if (computedHash !== paymentHash) {
    logger.warn('[Mock LN] Invalid preimage', {
      expected: paymentHash.slice(0, 16) + '...',
      got: computedHash.slice(0, 16) + '...',
    });
    return false;
  }
  
  record.paid = true;
  record.preimage = preimage;
  record.paidAt = Math.floor(Date.now() / 1000);
  
  logger.debug('[Mock LN] Invoice paid', { 
    paymentHash: paymentHash.slice(0, 16) + '...' 
  });
  
  return true;
}

export default mockProvider;

