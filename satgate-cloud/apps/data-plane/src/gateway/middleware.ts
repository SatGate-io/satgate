/**
 * Gateway middleware - ties together routing, L402, and proxy
 */

import * as crypto from 'crypto';
import { Request, Response, NextFunction } from 'express';
import { GatewayConfig } from '@satgate/gateway-config';
import { logger } from '@satgate/common';
import { matchRoute } from './router';
import { enforceL402 } from './l402';
import { proxyRequest } from './proxy';
import { logUsageEvent, EventType } from '../usage/events';

/**
 * Extended request with tenant context
 */
interface TenantRequest extends Request {
  tenantSlug: string;
  gatewayConfig: GatewayConfig;
  projectId: string;
}

/**
 * Generate a request ID
 */
function generateRequestId(): string {
  return `${Date.now().toString(36)}-${crypto.randomBytes(4).toString('hex')}`;
}

/**
 * Gateway middleware for Cloud data plane
 */
export async function gatewayMiddleware(
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> {
  const tenantReq = req as unknown as TenantRequest;
  const { tenantSlug, gatewayConfig: config, projectId } = tenantReq;
  
  if (!tenantSlug || !config) {
    res.status(500).json({ error: 'Missing tenant context' });
    return;
  }
  
  const requestId = generateRequestId();
  const path = req.path || '/';
  const method = req.method;
  
  // Match route
  const route = matchRoute(config, path, method, req.headers);
  
  if (!route) {
    // No matching route - default deny (fail-closed)
    if (projectId) {
      logUsageEvent({
        projectId,
        routeName: '__no_match__',
        eventType: 'denied',
        requestId,
      });
    }
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
      if (projectId) {
        logUsageEvent({
          projectId,
          routeName: route.name,
          eventType: 'allowed',
          requestId,
        });
      }
      await proxyRequest(req, res, config, route, tenantSlug);
      break;
      
    case 'deny':
      // Explicit deny
      if (projectId) {
        logUsageEvent({
          projectId,
          routeName: route.name,
          eventType: 'denied',
          requestId,
        });
      }
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
        if (projectId) {
          logUsageEvent({
            projectId,
            routeName: route.name,
            eventType: 'allowed',
            priceSats: policy.priceSats,
            requestId,
          });
        }
        
        // Attach token info for metering
        (req as any).l402TokenId = result.tokenId;
        (req as any).l402Caveats = result.caveats;
        
        await proxyRequest(req, res, config, route, tenantSlug);
      } else {
        // Send challenge or error
        if (projectId) {
          logUsageEvent({
            projectId,
            routeName: route.name,
            eventType: result.statusCode === 402 ? 'challenge' : 'denied',
            priceSats: result.statusCode === 402 ? policy.priceSats : undefined,
            requestId,
          });
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
