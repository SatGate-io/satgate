/**
 * Usage event persistence
 */

import { logger } from '@satgate/common';

export type EventType = 'challenge' | 'paid' | 'allowed' | 'denied';

export interface UsageEvent {
  projectId: string;
  routeName: string;
  eventType: EventType;
  priceSats?: number;
  requestId?: string;
}

// Buffer events for batch insert (reduces DB load)
const eventBuffer: UsageEvent[] = [];
const BUFFER_SIZE = 100;
const FLUSH_INTERVAL_MS = 5000;

let flushTimer: NodeJS.Timeout | null = null;

/**
 * Log a usage event
 */
export function logUsageEvent(event: UsageEvent): void {
  eventBuffer.push(event);
  
  // Flush if buffer is full
  if (eventBuffer.length >= BUFFER_SIZE) {
    flushEvents();
  }
  
  // Set up periodic flush if not already scheduled
  if (!flushTimer) {
    flushTimer = setTimeout(() => {
      flushTimer = null;
      flushEvents();
    }, FLUSH_INTERVAL_MS);
  }
}

/**
 * Flush buffered events to database
 */
async function flushEvents(): Promise<void> {
  if (eventBuffer.length === 0) return;
  
  // Take all events from buffer
  const events = eventBuffer.splice(0, eventBuffer.length);
  
  try {
    const { query } = await import('../db');
    
    // Build batch insert
    const values: any[] = [];
    const placeholders: string[] = [];
    
    let paramIndex = 1;
    for (const event of events) {
      placeholders.push(
        `($${paramIndex++}, $${paramIndex++}, $${paramIndex++}, $${paramIndex++}, $${paramIndex++})`
      );
      values.push(
        event.projectId,
        event.routeName,
        event.eventType,
        event.priceSats ?? null,
        event.requestId ?? null
      );
    }
    
    await query(
      `INSERT INTO usage_events (project_id, route_name, event_type, price_sats, request_id)
       VALUES ${placeholders.join(', ')}`,
      values
    );
    
    logger.debug('Flushed usage events', { count: events.length });
  } catch (err) {
    logger.error('Failed to flush usage events', { 
      count: events.length, 
      error: (err as Error).message 
    });
    
    // Put events back if insert failed (will retry on next flush)
    eventBuffer.unshift(...events);
  }
}

/**
 * Flush any remaining events (call on shutdown)
 */
export async function shutdown(): Promise<void> {
  if (flushTimer) {
    clearTimeout(flushTimer);
    flushTimer = null;
  }
  await flushEvents();
}

