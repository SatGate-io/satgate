import fetch from 'node-fetch';
import { SatGate, SatGateError } from '../index';

jest.mock('node-fetch', () => jest.fn());

const mockedFetch = fetch as unknown as jest.Mock;
const satgate = new SatGate({ apiKey: 'sg_test', baseUrl: 'https://api.example.test' });

beforeEach(() => {
  mockedFetch.mockReset();
  mockedFetch.mockResolvedValue({ ok: true, status: 200, json: async () => ({ decision: 'allow' }) });
});

it.each([
  { schema_version: 'satgate.receipt_bundle.v2' },
  { schema_version: 'satgate.receipt_bundle.v99' },
  { record_type: 'private_bundle' },
  { openings: null },
  { capsules: {} },
  { opening: '' },
  { schema_version: 'satgate.exact_call.opening.v1' },
  { schema_version: 'satgate.execution_result.opening.v1' },
  { schemaVersion: 'satgate.receipt_bundle.v2' },
  { recordType: 'private_bundle' },
  { schema_version: 'satgate.receipt_bundle.v2', schemaVersion: 'satgate.receipt.v1' },
  { schemaVersion: 'satgate.receipt_bundle.v2', schema_version: 'satgate.receipt.v1' },
])('refuses %p before POST', async (marker) => {
  const refusal = satgate.verify({ ...marker, dummy_private_field: 'dummy-private-value' });
  await expect(refusal).rejects.toThrow(SatGateError);
  await expect(refusal).rejects.toThrow(/^private_evidence_not_posted$/);
  expect(mockedFetch).not.toHaveBeenCalled();
  expect(JSON.stringify(mockedFetch.mock.calls)).not.toContain('dummy_private_field');
});

it('refuses a private bundle before POST', async () => {
  await expect(satgate.verify({
    schema_version: 'satgate.receipt_bundle.v2',
    record_type: 'private_bundle',
    manifest: {},
    artifacts: {},
    capsules: {},
    openings: { 'dummy-digest': 'dummy-private-value' },
  })).rejects.toThrow(/^private_evidence_not_posted$/);
  expect(mockedFetch).not.toHaveBeenCalled();
  expect(JSON.stringify(mockedFetch.mock.calls)).not.toContain('dummy-private-value');
});

it('still posts a receipt id once', async () => {
  await satgate.verify({ receiptId: 'rcpt_dummy' });
  expect(mockedFetch).toHaveBeenCalledTimes(1);
  expect(mockedFetch).toHaveBeenCalledWith('https://api.example.test/v1/verify', expect.objectContaining({
    method: 'POST',
    body: JSON.stringify({ receipt: { receipt_id: 'rcpt_dummy' } }),
  }));
});
