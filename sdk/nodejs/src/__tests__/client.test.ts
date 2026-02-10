import { SatGateClient } from '../client';

describe('SatGateClient', () => {
  it('should create a client with base URL', () => {
    const client = new SatGateClient({
      url: 'http://localhost:8080',
      token: 'test-token',
    });
    expect(client).toBeDefined();
  });
});
