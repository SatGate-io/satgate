/**
 * SimpleMacaroon - JSON-based macaroon implementation
 * 
 * Uses HMAC-SHA256 for signatures, avoids binary encoding issues.
 */

import * as crypto from 'crypto';

interface MacaroonData {
  v: number;          // version
  l: string;          // location
  i: string;          // identifier
  c: string[];        // caveats
  s: string;          // signature (hex)
}

export class SimpleMacaroon {
  private location: string;
  private identifier: string;
  private caveats: string[];
  private signature: Buffer;

  constructor(location: string, identifier: string, rootKey: string) {
    this.location = location;
    this.identifier = identifier;
    this.caveats = [];
    
    // Initial signature = HMAC(rootKey, identifier)
    this.signature = this.hmac(rootKey, identifier);
  }

  /**
   * Add a first-party caveat
   */
  addFirstPartyCaveat(caveat: string): void {
    this.caveats.push(caveat);
    // Chain signature: sig = HMAC(sig, caveat)
    this.signature = this.hmac(this.signature.toString('hex'), caveat);
  }

  /**
   * Serialize to base64 JSON
   */
  serialize(): string {
    const data: MacaroonData = {
      v: 1,
      l: this.location,
      i: this.identifier,
      c: this.caveats,
      s: this.signature.toString('hex'),
    };
    return Buffer.from(JSON.stringify(data)).toString('base64');
  }

  /**
   * Get the signature (for token binding)
   */
  getSignature(): string {
    return this.signature.toString('hex');
  }

  /**
   * Import from base64 JSON
   */
  static import(base64: string): SimpleMacaroon {
    const json = Buffer.from(base64, 'base64').toString('utf8');
    const data: MacaroonData = JSON.parse(json);
    
    const mac = Object.create(SimpleMacaroon.prototype);
    mac.location = data.l;
    mac.identifier = data.i;
    mac.caveats = data.c;
    mac.signature = Buffer.from(data.s, 'hex');
    
    return mac;
  }

  /**
   * Verify signature against root key
   */
  verify(rootKey: string): boolean {
    // Recompute signature from scratch
    let sig = this.hmac(rootKey, this.identifier);
    
    for (const caveat of this.caveats) {
      sig = this.hmac(sig.toString('hex'), caveat);
    }
    
    // Constant-time compare
    return crypto.timingSafeEqual(sig, this.signature);
  }

  /**
   * Get all caveats
   */
  getCaveats(): string[] {
    return [...this.caveats];
  }

  /**
   * Get identifier
   */
  getIdentifier(): string {
    return this.identifier;
  }

  /**
   * Get location
   */
  getLocation(): string {
    return this.location;
  }

  private hmac(key: string, data: string): Buffer {
    return crypto.createHmac('sha256', key).update(data).digest();
  }
}

