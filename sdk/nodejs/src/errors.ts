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

export class SatGateAuthError extends AuthenticationError {
  public readonly docsUrl: string;

  constructor(
    message: string = 'This API namespace requires private beta access. Visit cloud.satgate.io/docs to request access.',
    statusCode: number = 401,
    docsUrl: string = 'https://cloud.satgate.io/docs'
  ) {
    super(message);
    this.name = 'SatGateAuthError';
    this.statusCode = statusCode;
    this.docsUrl = docsUrl;
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

/**
 * An SDK request was answered with a redirect. The SDK never follows a
 * redirect, so a request (and any credential it carries) is sent only to the
 * configured URL, once. `host` and the message name only the status and the
 * host and port of the request URL; the SDK passes that URL's host with user
 * information, path and query removed.
 */
export class RedirectRefusedError extends SatGateError {
  public readonly host: string;

  constructor(status: number, host: string, label: string = 'host') {
    super(
      `${label} ${host} answered HTTP ${status || 'redirect'}; the SDK does not follow ` +
      'redirects, and the request was not sent again',
      status || undefined,
    );
    this.name = 'RedirectRefusedError';
    this.host = host;
  }
}

/**
 * The SDK refused a request URL before sending anything: the URL has user
 * information (a user name or password), or it cannot be parsed. `host` and
 * the message name only the host and port of the URL, never its user
 * information, path or query.
 */
export class RequestUrlRefusedError extends SatGateError {
  public readonly host: string;

  constructor(host: string, label: string = 'host', reason: 'user-info' | 'unparseable' = 'user-info') {
    super(
      reason === 'user-info'
        ? `${label} URL for ${host} has a user name or password in it; the SDK does not ` +
          'send a request to a URL with user information, and no request was sent'
        : `${label} URL could not be parsed; no request was sent`,
    );
    this.name = 'RequestUrlRefusedError';
    this.host = host;
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

