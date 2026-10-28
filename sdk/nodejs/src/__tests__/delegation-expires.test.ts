import { Caveats } from '../delegation';

describe('Caveats.expires', () => {
  it('writes seconds from now as an absolute time in Unix milliseconds', () => {
    const before = Date.now();
    const caveat = Caveats.expires(3600);
    const after = Date.now();
    const match = /^expires = (\d+)$/.exec(caveat);
    expect(match).not.toBeNull();
    const value = Number(match![1]);
    expect(value).toBeGreaterThanOrEqual(before + 3600 * 1000);
    expect(value).toBeLessThanOrEqual(after + 3600 * 1000);
  });

  it('writes a Date as Unix milliseconds', () => {
    const at = new Date('2026-11-01T00:00:00Z');
    expect(Caveats.expires(at)).toBe(`expires = ${at.getTime()}`);
  });

  it('refuses an invalid Date', () => {
    expect(() => Caveats.expires(new Date('not a date'))).toThrow();
  });
});
