/**
 * Structured logging
 */

export interface Logger {
  info(message: string, meta?: Record<string, unknown>): void;
  warn(message: string, meta?: Record<string, unknown>): void;
  error(message: string, meta?: Record<string, unknown>): void;
  debug(message: string, meta?: Record<string, unknown>): void;
}

function formatLog(level: string, message: string, meta?: Record<string, unknown>): string {
  const timestamp = new Date().toISOString();
  const metaStr = meta ? ` ${JSON.stringify(meta)}` : '';
  return `[${timestamp}] [${level.toUpperCase()}] ${message}${metaStr}`;
}

export const logger: Logger = {
  info(message: string, meta?: Record<string, unknown>) {
    console.log(formatLog('info', message, meta));
  },
  
  warn(message: string, meta?: Record<string, unknown>) {
    console.warn(formatLog('warn', message, meta));
  },
  
  error(message: string, meta?: Record<string, unknown>) {
    console.error(formatLog('error', message, meta));
  },
  
  debug(message: string, meta?: Record<string, unknown>) {
    if (process.env.DEBUG) {
      console.log(formatLog('debug', message, meta));
    }
  },
};

