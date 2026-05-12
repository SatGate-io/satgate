import fetch from 'node-fetch';
import { SatGate, SatGateAuthError } from '../index';

jest.mock('node-fetch', () => jest.fn());

const mockedFetch = fetch as unknown as jest.Mock;

describe('SatGate issue/pay/verify primitive facade', () => {
  beforeEach(() => {
    mockedFetch.mockReset();
  });

  it('throws a clear beta-access error without credentials', async () => {
    const satgate = new SatGate();

    await expect(satgate.issue({
      task: 'summarize vendor invoice',
      agent: 'invoice-agent',
      allow: ['POST /v1/invoices/*'],
      budgetUsd: 0.25,
      expiresIn: '10m',
    })).rejects.toThrow(SatGateAuthError);

    await expect(satgate.issue({
      task: 'summarize vendor invoice',
      agent: 'invoice-agent',
      allow: ['POST /v1/invoices/*'],
    })).rejects.toThrow('cloud.satgate.io/docs');
    expect(mockedFetch).not.toHaveBeenCalled();
  });

  it('falls back to the early capabilities endpoint when /v1/issue is absent', async () => {
    mockedFetch
      .mockResolvedValueOnce({ ok: false, status: 404, json: async () => ({}) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ id: 'cap_compat' }) });

    const satgate = new SatGate({ apiKey: 'sg_test', baseUrl: 'https://api.example.test' });
    const capability = await satgate.issue({
      task: 'summarize vendor invoice',
      agent: 'invoice-agent',
      allow: ['POST /v1/invoices/*'],
    });

    expect(mockedFetch).toHaveBeenNthCalledWith(1, 'https://api.example.test/v1/issue', expect.any(Object));
    expect(mockedFetch).toHaveBeenNthCalledWith(2, 'https://api.example.test/v1/capabilities', expect.any(Object));
    expect(capability.id).toBe('cap_compat');
  });

  it('posts copy-pasteable issue/pay/verify payloads with primitive endpoint names', async () => {
    mockedFetch
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ id: 'cap_123' }) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ receiptId: 'rcpt_123' }) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ decision: 'allow', evidencePackId: 'ep_123' }) });

    const satgate = new SatGate({ apiKey: 'sg_test', baseUrl: 'https://api.example.test', tenant: 'tenant_123' });
    const capability = await satgate.issue({
      task: 'summarize vendor invoice',
      agent: 'invoice-agent',
      allow: ['POST /v1/invoices/*'],
      budgetUsd: 0.25,
      expiresIn: '10m',
    });
    const receipt = await satgate.pay({
      upstream: 'https://api.vendor.test/v1/invoices/42',
      capability,
      maxUsd: 0.10,
    });
    await satgate.verify(receipt);

    expect(mockedFetch).toHaveBeenNthCalledWith(1, 'https://api.example.test/v1/issue', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({
        Authorization: 'Bearer sg_test',
        'Content-Type': 'application/json',
        'X-SatGate-Tenant': 'tenant_123',
      }),
      body: JSON.stringify({
        task: 'summarize vendor invoice',
        agent: 'invoice-agent',
        allow: ['POST /v1/invoices/*'],
        budget_usd: 0.25,
        expires_in: '10m',
      }),
    }));
    expect(mockedFetch).toHaveBeenNthCalledWith(2, 'https://api.example.test/v1/pay', expect.objectContaining({
      body: JSON.stringify({
        upstream: 'https://api.vendor.test/v1/invoices/42',
        capability: { id: 'cap_123' },
        max_usd: 0.10,
      }),
    }));
    expect(mockedFetch).toHaveBeenNthCalledWith(3, 'https://api.example.test/v1/verify', expect.objectContaining({
      body: JSON.stringify({ receipt: { receipt_id: 'rcpt_123' } }),
    }));
  });
});
