import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = process.cwd();
const pagePath = join(root, 'app', 'agent-control-plane', 'page.tsx');

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

assert(existsSync(pagePath), `Missing ACP landing page: ${pagePath}`);
const page = readFileSync(pagePath, 'utf8');

const requiredText = [
  'SatGate Agent Control Plane',
  'Treat every model like a capable insider that can make mistakes or be tricked.',
  'Book a demo',
  'Why zero trust in the model',
  'Give every agent its own limits, and keep the off switch.',
  'A SatGate Cloud receipt can be checked with the open-source verifier against our published keys.',
];

for (const text of requiredText) {
  assert(page.includes(text), `Landing page missing required text: ${text}`);
}

console.log('ACP landing page verification passed');
