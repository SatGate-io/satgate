/**
 * ID generation utilities
 */

import { customAlphabet } from 'nanoid';

// URL-safe alphabet for slugs
const slugAlphabet = 'abcdefghijklmnopqrstuvwxyz0123456789';
const slugGenerator = customAlphabet(slugAlphabet, 8);

// Hex alphabet for API keys
const hexAlphabet = 'abcdef0123456789';
const apiKeyGenerator = customAlphabet(hexAlphabet, 32);

// Alphanumeric for general IDs
const alphanumeric = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
const idGenerator = customAlphabet(alphanumeric, 21);

/**
 * Generate a unique ID (21 chars, alphanumeric)
 */
export function generateId(): string {
  return idGenerator();
}

/**
 * Generate a URL-safe slug (8 chars, lowercase alphanumeric)
 */
export function generateSlug(): string {
  return slugGenerator();
}

/**
 * Generate an API key (32 hex chars)
 */
export function generateApiKey(): string {
  return apiKeyGenerator();
}

