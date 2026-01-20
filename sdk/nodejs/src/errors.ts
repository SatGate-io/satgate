/**
 * SatGate Gateway Error Classes
 */

export class SatGateError extends Error {
  constructor(
    message: string,
    public statusCode?: number
  ) {
    super(message);
    this.name = 'SatGateError';
  }
}

export class AuthenticationError extends SatGateError {
  constructor(message: string = 'Authentication failed') {
    super(message, 401);
    this.name = 'AuthenticationError';
  }
}

export class NotFoundError extends SatGateError {
  constructor(message: string = 'Resource not found') {
    super(message, 404);
    this.name = 'NotFoundError';
  }
}

export class ValidationError extends SatGateError {
  constructor(message: string = 'Validation failed') {
    super(message, 400);
    this.name = 'ValidationError';
  }
}

export class RateLimitError extends SatGateError {
  constructor(
    message: string = 'Rate limit exceeded',
    public retryAfter?: number
  ) {
    super(message, 429);
    this.name = 'RateLimitError';
  }
}

/**
 * 402 Payment Required - economic gate triggered
 */
export class PaymentRequiredError extends SatGateError {
  constructor(
    message: string = 'Payment required',
    public challenge: Record<string, unknown> = {}
  ) {
    super(message, 402);
    this.name = 'PaymentRequiredError';
  }
  
  /** The amount required for payment */
  get amount(): number {
    return (this.challenge.amount as number) || 0;
  }
  
  /** The currency unit (USD, sats, credits) */
  get unit(): string {
    return (this.challenge.unit as string) || 'unknown';
  }
  
  /** The invoice ID for Fiat402 challenges */
  get invoiceId(): string | undefined {
    return this.challenge.invoice_id as string | undefined;
  }
  
  /** The Lightning invoice for L402 challenges */
  get invoice(): string | undefined {
    return this.challenge.invoice as string | undefined;
  }
}

/**
 * Payment attempt failed
 */
export class PaymentFailedError extends SatGateError {
  constructor(message: string = 'Payment failed') {
    super(message);
    this.name = 'PaymentFailedError';
  }
}

/**
 * Budget limit exceeded
 */
export class BudgetExceededError extends SatGateError {
  constructor(
    message: string = 'Budget exceeded',
    public used: number = 0,
    public limit: number = 0
  ) {
    super(message);
    this.name = 'BudgetExceededError';
  }
  
  /** Remaining budget (may be negative) */
  get remaining(): number {
    return this.limit - this.used;
  }
}

/**
 * Token has expired
 */
export class TokenExpiredError extends SatGateError {
  constructor(message: string = 'Token expired') {
    super(message, 401);
    this.name = 'TokenExpiredError';
  }
}

/**
 * Token delegation failed
 */
export class DelegationError extends SatGateError {
  constructor(message: string = 'Delegation failed') {
    super(message);
    this.name = 'DelegationError';
  }
}

