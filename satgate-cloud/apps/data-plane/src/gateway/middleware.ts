/**
 * Gateway middleware - ties together routing, L402, and proxy
 */

import { Request, Response, NextFunction } from 'express';
import { GatewayConfig } from '@satgate/gateway-config';
import { logger } from '@satgate/common';
import { matchRoute } from './router';
import { enforceL402 } from './l402';
import { proxyRequest } from './proxy';

/**
 * Gateway middleware for Cloud data plane
 */
export async function gatewayMiddleware(
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> {
  const tenantSlug = (req as any).tenantSlug as string;
  const config = (req as any).gatewayConfig as GatewayConfig;
  
  if (!tenantSlug || !config) {
    res.status(500).json({ error: 'Missing tenant context' });
    return;
  }
  
  const path = req.path || '/';
  const method = req.method;
  
  // Match route
  const route = matchRoute(config, path, method, req.headers);
  
  if (!route) {
    // No matching route - default deny (fail-closed)
    res.status(403).json({
      error: 'Forbidden',
      message: 'No matching route',
    });
    return;
  }
  
  // Apply policy
  const policy = route.policy;
  
  switch (policy.kind) {
    case 'public':
      // No auth required - proxy directly
      await proxyRequest(req, res, config, route, tenantSlug);
      break;
      
    case 'deny':
      // Explicit deny
      res.status(policy.status || 403).json({
        error: 'Forbidden',
        message: 'Access denied',
      });
      break;
      
    case 'l402':
      // L402 enforcement
      const result = await enforceL402(req, res, policy, tenantSlug);
      
      if (result.allowed) {
        // Log successful L402 access
        logUsageEvent(tenantSlug, route.name, 'allowed', policy.priceSats);
        
        // Attach token info for metering
        (req as any).l402TokenId = result.tokenId;
        (req as any).l402Caveats = result.caveats;
        
        await proxyRequest(req, res, config, route, tenantSlug);
      } else {
        // Send challenge or error
        if (result.statusCode === 402) {
          logUsageEvent(tenantSlug, route.name, 'challenge', policy.priceSats);
        }
        
        if (result.headers) {
          for (const [key, value] of Object.entries(result.headers)) {
            res.setHeader(key, value);
          }
        }
        
        res.status(result.statusCode || 401).json(result.body);
      }
      break;
      
    case 'capability':
      // Capability token enforcement (similar to L402 but pre-issued)
      // TODO: Implement capability validation
      res.status(501).json({
        error: 'Not Implemented',
        message: 'Capability tokens not yet supported in Cloud',
      });
      break;
      
    default:
      res.status(500).json({
        error: 'Unknown policy kind',
        kind: (policy as any).kind,
      });
  }
}

/**
 * Log a usage event
 */
function logUsageEvent(
  tenantSlug: string,
  routeName: string,
  eventType: 'challenge' | 'paid' | 'allowed' | 'denied',
  priceSats?: number
): void {
  // For v1, just log to console
  // TODO: Write to usage_events table
  logger.debug('Usage event', {
    tenant: tenantSlug,
    route: routeName,
    event: eventType,
    price: priceSats,
  });
}

