import { existsSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const root = process.cwd();
const pagePath = join(root, 'app', 'agent-control-plane', 'page.tsx');
const assetDir = join(root, 'public', 'acp-demo');
const requiredAssets = [
  'satgate-acp-first-touch.mp4',
  'satgate-acp-walkthrough.mp4',
  'satgate-acp-thumbnail.jpg',
  'satgate-acp-ciso-proof-card.png',
  'satgate-acp-ciso-proof-card.pdf',
];

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

assert(existsSync(pagePath), `Missing ACP landing page: ${pagePath}`);
const page = readFileSync(pagePath, 'utf8');

const requiredText = [
  'SatGate Agent Control Plane',
  'A local AI agent has no standing authority',
  'Control the agent. Control delegation. Prove the lineage.',
  '/acp-demo/satgate-acp-first-touch.mp4',
  '/acp-demo/satgate-acp-walkthrough.mp4',
  '/acp-demo/satgate-acp-ciso-proof-card.png',
  '200 / 402 / 401 / 403 / 400',
  'Book a demo',
];

for (const text of requiredText) {
  assert(page.includes(text), `Landing page missing required text/reference: ${text}`);
}

for (const asset of requiredAssets) {
  const path = join(assetDir, asset);
  assert(existsSync(path), `Missing ACP demo asset: ${path}`);
  assert(statSync(path).size > 1024, `ACP demo asset is unexpectedly small: ${path}`);
}

console.log('ACP landing page verification passed');
