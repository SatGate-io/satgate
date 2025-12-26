/**
 * Database connection (read-only for data plane)
 */

import { Pool } from 'pg';
import { logger } from '@satgate/common';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 10,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

pool.on('error', (err) => {
  logger.error('Database pool error', { error: err.message });
});

/**
 * Execute a read-only query
 */
export async function query<T = any>(
  text: string,
  params?: any[]
): Promise<{ rows: T[]; rowCount: number }> {
  const start = Date.now();
  const result = await pool.query(text, params);
  const duration = Date.now() - start;
  
  if (process.env.DEBUG) {
    logger.debug('Query executed', { duration, rowCount: result.rowCount });
  }
  
  return { rows: result.rows as T[], rowCount: result.rowCount || 0 };
}

/**
 * Health check
 */
export async function healthCheck(): Promise<boolean> {
  try {
    await pool.query('SELECT 1');
    return true;
  } catch {
    return false;
  }
}

export default { query, healthCheck };

