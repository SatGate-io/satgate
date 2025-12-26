/**
 * Streaming HTTP proxy for upstream requests
 */

import * as http from 'http';
import * as https from 'https';
import { Request, Response } from 'express';
import { GatewayConfig, Route, Upstream } from '@satgate/gateway-config';
import { logger, HOP_BY_HOP_HEADERS, isHopByHopHeader } from '@satgate/common';

// Headers to never forward
const BLOCKED_HEADERS = new Set([
  'host',
  'x-forwarded-for',
  'x-forwarded-proto',
  'x-forwarded-host',
  ...Array.from(HOP_BY_HOP_HEADERS).map((h: string) => h.toLowerCase()),
]);

/**
 * Proxy a request to upstream
 */
export async function proxyRequest(
  req: Request,
  res: Response,
  config: GatewayConfig,
  route: Route,
  tenantSlug: string
): Promise<void> {
  const upstreamName = route.upstream;
  
  if (!upstreamName) {
    res.status(502).json({ error: 'No upstream configured for route' });
    return;
  }
  
  const upstream = config.upstreams?.[upstreamName];
  
  if (!upstream) {
    res.status(502).json({ error: `Upstream '${upstreamName}' not found` });
    return;
  }
  
  const url = new URL(upstream.url);
  const isHttps = url.protocol === 'https:';
  const httpModule = isHttps ? https : http;
  
  // Build request path
  const requestPath = req.originalUrl || req.url;
  
  // Build headers
  const headers: Record<string, string | string[]> = {};
  
  for (const [key, value] of Object.entries(req.headers)) {
    const lower = key.toLowerCase();
    
    // Skip blocked headers
    if (BLOCKED_HEADERS.has(lower)) continue;
    if (isHopByHopHeader(key)) continue;
    
    // Check denylist
    if (upstream.denyRequestHeaders?.some(h => h.toLowerCase() === lower)) {
      continue;
    }
    
    // Check allowlist (if specified, only allow listed headers)
    if (upstream.allowRequestHeaders && upstream.allowRequestHeaders.length > 0) {
      if (!upstream.allowRequestHeaders.some(h => h.toLowerCase() === lower)) {
        continue;
      }
    }
    
    if (value) {
      headers[key] = value;
    }
  }
  
  // Set/override headers
  headers['host'] = upstream.passHostHeader ? (req.headers.host || url.host) : url.host;
  headers['x-forwarded-for'] = req.ip || req.socket.remoteAddress || '';
  headers['x-forwarded-proto'] = req.protocol;
  headers['x-forwarded-host'] = req.headers.host || '';
  headers['x-satgate-tenant'] = tenantSlug;
  headers['x-request-id'] = generateRequestId();
  
  // Add configured headers
  if (upstream.addHeaders) {
    for (const [key, value] of Object.entries(upstream.addHeaders)) {
      headers[key] = value;
    }
  }
  
  const options: http.RequestOptions = {
    hostname: url.hostname,
    port: url.port || (isHttps ? 443 : 80),
    path: requestPath,
    method: req.method,
    headers,
    timeout: upstream.timeoutMs || config.limits?.upstreamTimeoutMs || 30000,
  };
  
  return new Promise((resolve) => {
    const proxyReq = httpModule.request(options, (proxyRes) => {
      // Forward status
      res.status(proxyRes.statusCode || 502);
      
      // Forward headers (except hop-by-hop)
      for (const [key, value] of Object.entries(proxyRes.headers)) {
        if (!isHopByHopHeader(key) && value) {
          res.setHeader(key, value);
        }
      }
      
      // Stream response body
      proxyRes.pipe(res);
      
      proxyRes.on('end', resolve);
    });
    
    proxyReq.on('error', (err) => {
      logger.error('Upstream request error', { 
        upstream: upstreamName, 
        error: err.message 
      });
      
      if (!res.headersSent) {
        res.status(502).json({ 
          error: 'Bad Gateway',
          message: 'Failed to connect to upstream',
        });
      }
      resolve();
    });
    
    proxyReq.on('timeout', () => {
      proxyReq.destroy();
      
      if (!res.headersSent) {
        res.status(504).json({
          error: 'Gateway Timeout',
          message: 'Upstream did not respond in time',
        });
      }
      resolve();
    });
    
    // Stream request body
    if (req.readable) {
      req.pipe(proxyReq);
    } else {
      proxyReq.end();
    }
  });
}

/**
 * Generate a unique request ID
 */
function generateRequestId(): string {
  return `${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 10)}`;
}

