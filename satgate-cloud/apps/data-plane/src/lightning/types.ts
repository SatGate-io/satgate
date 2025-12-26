/**
 * Lightning provider types
 */

export interface Invoice {
  paymentHash: string;
  paymentRequest: string;  // BOLT11 invoice
  amountSats: number;
  memo: string;
  expiresAt: number;       // Unix timestamp in seconds
}

export interface InvoiceStatus {
  paid: boolean;
  preimage?: string;       // Hex-encoded preimage (if paid)
  paidAt?: number;         // Unix timestamp (if paid)
}

export interface CreateInvoiceParams {
  amountSats: number;
  memo: string;
  expirySecs?: number;     // Default: 3600 (1 hour)
}

export interface LightningProvider {
  /**
   * Provider name (for logging)
   */
  name: string;
  
  /**
   * Check if provider is available
   */
  isAvailable(): Promise<boolean>;
  
  /**
   * Create a new invoice
   */
  createInvoice(params: CreateInvoiceParams): Promise<Invoice>;
  
  /**
   * Check invoice payment status
   */
  getInvoiceStatus(paymentHash: string): Promise<InvoiceStatus>;
}

