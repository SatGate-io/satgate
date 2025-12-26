/**
 * HTTP utilities
 */

/**
 * Hop-by-hop headers that should not be forwarded
 */
export const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailers',
  'transfer-encoding',
  'upgrade',
  'proxy-connection',
]);

/**
 * Check if a header is hop-by-hop
 */
export function isHopByHopHeader(header: string): boolean {
  return HOP_BY_HOP_HEADERS.has(header.toLowerCase());
}

/**
 * Sanitize headers by removing hop-by-hop headers
 */
export function sanitizeHeaders(
  headers: Record<string, string | string[] | undefined>,
  options: { deny?: string[] } = {}
): Record<string, string> {
  const result: Record<string, string> = {};
  const denySet = new Set((options.deny || []).map(h => h.toLowerCase()));
  
  for (const [key, value] of Object.entries(headers)) {
    const lowerKey = key.toLowerCase();
    
    // Skip hop-by-hop headers
    if (isHopByHopHeader(lowerKey)) continue;
    
    // Skip denied headers
    if (denySet.has(lowerKey)) continue;
    
    // Skip undefined/null
    if (value === undefined || value === null) continue;
    
    // Convert arrays to comma-separated string
    result[key] = Array.isArray(value) ? value.join(', ') : value;
  }
  
  return result;
}

