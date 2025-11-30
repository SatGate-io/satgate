
export interface SatGateClientConfig {
  wallet?: 'alby' | 'webln' | WalletProvider;
  baseUrl?: string;
  onPayment?: (details: { url: string; amount: number | null; preimage: string }) => void;
}

export interface WalletProvider {
  payInvoice(invoice: string): Promise<{ preimage: string }>;
}

export class SatGateClient {
  constructor(config?: SatGateClientConfig);
  fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response>;
  get(url: string, init?: RequestInit): Promise<Response>;
  post(url: string, body: any, init?: RequestInit): Promise<Response>;
}

