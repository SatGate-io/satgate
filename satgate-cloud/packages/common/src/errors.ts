/**
 * Error classes
 */

export class SatGateError extends Error {
  public readonly code: string;
  public readonly statusCode: number;
  
  constructor(message: string, code: string, statusCode: number = 500) {
    super(message);
    this.name = 'SatGateError';
    this.code = code;
    this.statusCode = statusCode;
  }
  
  toJSON() {
    return {
      error: this.message,
      code: this.code,
    };
  }
}

export class ValidationError extends SatGateError {
  public readonly details: string[];
  
  constructor(message: string, details: string[] = []) {
    super(message, 'VALIDATION_ERROR', 400);
    this.name = 'ValidationError';
    this.details = details;
  }
  
  toJSON() {
    return {
      error: this.message,
      code: this.code,
      details: this.details,
    };
  }
}

export class NotFoundError extends SatGateError {
  constructor(resource: string) {
    super(`${resource} not found`, 'NOT_FOUND', 404);
    this.name = 'NotFoundError';
  }
}

export class UnauthorizedError extends SatGateError {
  constructor(message: string = 'Unauthorized') {
    super(message, 'UNAUTHORIZED', 401);
    this.name = 'UnauthorizedError';
  }
}

export class ForbiddenError extends SatGateError {
  constructor(message: string = 'Forbidden') {
    super(message, 'FORBIDDEN', 403);
    this.name = 'ForbiddenError';
  }
}

