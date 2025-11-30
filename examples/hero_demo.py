#!/usr/bin/env python3
"""
🎬 SatGate Hero Demo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
An AI agent with no bank account, no credit card, and no API keys
autonomously pays for premium data using Lightning micropayments.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"""

import time
import json
import sys

# ANSI colors for terminal output
class Colors:
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    GREEN = '\033[92m'
    RED = '\033[91m'
    CYAN = '\033[96m'
    BOLD = '\033[1m'
    END = '\033[0m'

def log(emoji, msg, color=Colors.END, delay=0.3):
    """Print a log line with color and timing."""
    print(f"{color}{emoji} {msg}{Colors.END}")
    time.sleep(delay)

def main():
    print()
    print(f"{Colors.BOLD}{'━' * 60}{Colors.END}")
    print(f"{Colors.CYAN}{Colors.BOLD}  🤖 SATGATE AUTONOMOUS AGENT DEMO{Colors.END}")
    print(f"{Colors.BOLD}{'━' * 60}{Colors.END}")
    print()
    
    # Step 1: Agent Introduction
    log("🤖", "Initializing AI Agent...", Colors.CYAN, 0.5)
    log("🔑", "No API keys configured", Colors.YELLOW, 0.3)
    log("💳", "No credit card on file", Colors.YELLOW, 0.3)
    log("🏦", "No bank account linked", Colors.YELLOW, 0.3)
    log("⚡", "Lightning Wallet: CONNECTED", Colors.GREEN, 0.5)
    print()
    
    # Step 2: Mission Assignment
    log("🎯", "Mission: Fetch market data", Colors.BOLD, 0.5)
    log("📡", "Target: http://localhost:8081/api/micro/data", Colors.CYAN, 0.3)
    print()
    
    # Step 3: The Request
    log("→", "Sending GET request...", Colors.CYAN, 0.8)
    print()
    
    # Step 4: The 402 Challenge (THE CRITICAL MOMENT)
    print(f"{Colors.YELLOW}{Colors.BOLD}⚡ 402 PAYMENT REQUIRED{Colors.END}")
    log("  ", "Price: 1 sat ($0.001)", Colors.YELLOW, 0.3)
    log("  ", "Invoice: lnbc10n1p5jn...q4378au", Colors.YELLOW, 0.5)
    print()
    
    # Step 5: Autonomous Payment (THE MAGIC)
    log("💰", "Agent deciding to pay...", Colors.BLUE, 0.4)
    log("💰", "Signing Lightning transaction...", Colors.BLUE, 0.4)
    log("💰", "Broadcasting to network...", Colors.BLUE, 0.6)
    print()
    
    # Step 6: Payment Confirmation (THE MONEY SHOT)
    print(f"{Colors.GREEN}{Colors.BOLD}✅ PAYMENT SENT!{Colors.END}")
    log("  ", "Preimage: 4a289b3691f8c2e5...secured", Colors.GREEN, 0.3)
    log("  ", "Settlement: < 1 second", Colors.GREEN, 0.5)
    print()
    
    # Step 7: Retry with Token
    log("🔄", "Retrying request with L402 token...", Colors.CYAN, 0.5)
    log("→", "Authorization: L402 <macaroon>:<preimage>", Colors.CYAN, 0.3)
    print()
    
    # Step 8: Success!
    print(f"{Colors.GREEN}{Colors.BOLD}✅ 200 OK - ACCESS GRANTED{Colors.END}")
    print()
    
    # Step 9: The Data (THE PAYOFF)
    data = {
        "ok": True,
        "tier": "micro",
        "price": "1 sat ($0.001)",
        "data": {
            "market_sentiment": "bullish",
            "confidence": 0.87,
            "note": "This request cost $0.001 — true micropayments!"
        }
    }
    
    print(f"{Colors.CYAN}📦 Response Data:{Colors.END}")
    print(f"{Colors.GREEN}{json.dumps(data, indent=2)}{Colors.END}")
    print()
    
    # Final Summary
    print(f"{Colors.BOLD}{'━' * 60}{Colors.END}")
    print(f"{Colors.GREEN}{Colors.BOLD}  ✅ MISSION COMPLETE{Colors.END}")
    print(f"{Colors.BOLD}{'━' * 60}{Colors.END}")
    print()
    print(f"  {Colors.CYAN}Frictionless, machine-to-machine commerce.{Colors.END}")
    print(f"  {Colors.YELLOW}{Colors.BOLD}This is SatGate.{Colors.END}")
    print()

if __name__ == "__main__":
    main()

