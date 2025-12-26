/**
 * JSON Schema for gateway config
 * Exported for use in validation
 */

export const schema = {
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "upstreams", "routes"],
  "properties": {
    "version": {
      "type": "integer",
      "minimum": 1,
      "maximum": 1
    },
    "server": {
      "type": "object",
      "properties": {
        "listen": { "type": "string" },
        "trustProxy": { "type": "integer" }
      }
    },
    "admin": {
      "type": "object",
      "properties": {
        "listen": { "type": "string" },
        "requireAdminToken": { "type": "boolean" }
      }
    },
    "limits": {
      "type": "object",
      "properties": {
        "maxRequestBodyBytes": { "type": "integer" },
        "maxHeadersBytes": { "type": "integer" },
        "upstreamTimeoutMs": { "type": "integer" },
        "upstreamConnectTimeoutMs": { "type": "integer" }
      }
    },
    "cors": {
      "type": "object",
      "properties": {
        "origins": {
          "type": "array",
          "items": { "type": "string" }
        },
        "allowCredentials": { "type": "boolean" }
      }
    },
    "l402": {
      "type": "object",
      "properties": {
        "mode": { "type": "string", "enum": ["native", "aperture"] },
        "rootKeyEnv": { "type": "string" },
        "defaultTTLSeconds": { "type": "integer" },
        "defaultMaxCalls": { "type": "integer" },
        "defaultBudgetSats": { "type": "integer" }
      }
    },
    "metering": {
      "type": "object",
      "properties": {
        "backend": { "type": "string", "enum": ["redis", "memory"] },
        "redisUrlEnv": { "type": "string" }
      }
    },
    "upstreams": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "required": ["url"],
        "properties": {
          "url": { "type": "string", "format": "uri" },
          "passHostHeader": { "type": "boolean" },
          "addHeaders": {
            "type": "object",
            "additionalProperties": { "type": "string" }
          },
          "allowRequestHeaders": {
            "type": "array",
            "items": { "type": "string" }
          },
          "denyRequestHeaders": {
            "type": "array",
            "items": { "type": "string" }
          },
          "timeoutMs": { "type": "integer" }
        }
      }
    },
    "routes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "match", "policy"],
        "properties": {
          "name": {
            "type": "string",
            "pattern": "^[a-z0-9][a-z0-9_-]*$"
          },
          "match": {
            "type": "object",
            "properties": {
              "pathPrefix": { "type": "string" },
              "exactPath": { "type": "string" },
              "methods": {
                "type": "array",
                "items": { "type": "string" }
              },
              "headers": {
                "type": "object",
                "additionalProperties": { "type": "string" }
              }
            }
          },
          "upstream": { "type": "string" },
          "policy": {
            "type": "object",
            "required": ["kind"],
            "properties": {
              "kind": {
                "type": "string",
                "enum": ["public", "deny", "l402", "capability"]
              }
            },
            "allOf": [
              {
                "if": { "properties": { "kind": { "const": "l402" } } },
                "then": {
                  "required": ["tier", "priceSats", "scope"],
                  "properties": {
                    "tier": { "type": "string" },
                    "priceSats": { "type": "integer", "minimum": 1 },
                    "scope": { "type": "string" },
                    "ttlSeconds": { "type": "integer" },
                    "maxCalls": { "type": "integer" },
                    "budgetSats": { "type": "integer" }
                  }
                }
              },
              {
                "if": { "properties": { "kind": { "const": "deny" } } },
                "then": {
                  "properties": {
                    "status": { "type": "integer", "minimum": 400, "maximum": 599 }
                  }
                }
              },
              {
                "if": { "properties": { "kind": { "const": "capability" } } },
                "then": {
                  "required": ["scope"],
                  "properties": {
                    "scope": { "type": "string" },
                    "maxCalls": { "type": "integer" },
                    "budgetSats": { "type": "integer" }
                  }
                }
              }
            ]
          }
        }
      }
    }
  }
};

