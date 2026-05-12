#!/usr/bin/env node
/**
 * Minimal SatGate issue/pay/verify private-beta example.
 *
 * Run from sdk/nodejs after building local SDK source:
 *
 *   npm install
 *   npm run build
 *   SATGATE_API_KEY=sg_live_... node ../../examples/node/issue-pay-verify.mjs
 *
 * Without a beta key this exits cleanly with SatGateAuthError and the docs CTA.
 */

import { SatGate, SatGateAuthError } from '../../sdk/nodejs/dist/index.js';

const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

try {
  const capability = await satgate.issue({
    task: 'research market prices',
    agent: 'research-agent',
    allow: ['https://api.example.com/search'],
    budgetUsd: 2.00,
    expiresIn: '10m',
  });

  const receipt = await satgate.pay({
    upstream: 'https://api.example.com/search',
    capability,
    maxUsd: 0.25,
  });

  const verified = await satgate.verify(receipt);
  console.log(verified.decision, verified.evidencePackId ?? verified.evidence_pack_id);
} catch (err) {
  if (err instanceof SatGateAuthError || err?.name === 'SatGateAuthError') {
    console.error(`SatGateAuthError: ${err.message}`);
    process.exit(1);
  }
  throw err;
}
