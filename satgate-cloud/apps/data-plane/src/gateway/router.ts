/**
 * Route matcher for gateway requests
 */

import { Route, GatewayConfig } from '@satgate/gateway-config';

/**
 * Match a request against gateway routes
 */
export function matchRoute(
  config: GatewayConfig,
  path: string,
  method: string,
  headers: Record<string, string | string[] | undefined>
): Route | null {
  for (const route of config.routes) {
    if (matchesRoute(route, path, method, headers)) {
      return route;
    }
  }
  return null;
}

/**
 * Check if request matches a route
 */
function matchesRoute(
  route: Route,
  path: string,
  method: string,
  headers: Record<string, string | string[] | undefined>
): boolean {
  const match = route.match;
  
  // Check exact path match
  if (match.exactPath && path !== match.exactPath) {
    return false;
  }
  
  // Check path prefix match
  if (match.pathPrefix && !path.startsWith(match.pathPrefix)) {
    return false;
  }
  
  // Check method match
  if (match.methods && match.methods.length > 0) {
    const upperMethod = method.toUpperCase();
    if (!match.methods.some(m => m.toUpperCase() === upperMethod)) {
      return false;
    }
  }
  
  // Check header matches
  if (match.headers) {
    for (const [key, expected] of Object.entries(match.headers)) {
      const actual = headers[key.toLowerCase()];
      if (!actual) return false;
      
      const actualValue = Array.isArray(actual) ? actual[0] : actual;
      if (actualValue !== expected) return false;
    }
  }
  
  return true;
}

