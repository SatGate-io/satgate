import os
import sys
import logging

# --- Setup Paths & Imports ---
# Ensure we can import the SDK locally (if not yet installed via pip)
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(__file__)), 'sdk/python'))

# We need to gracefully handle missing dependencies for the demo
try:
    from langchain.chat_models import ChatOpenAI
    from langchain.agents import initialize_agent, AgentType
    from satgate.client import LightningWallet
    from satgate.langchain_integrations import SatGateTool
except ImportError as e:
    print(f"⚠️  Missing dependency: {e}")
    print("   Please install: pip install langchain openai requests")
    sys.exit(1)

# --- Configuration ---
# 1. Get OpenAI Key (Critical for the Agent to "think")
if "OPENAI_API_KEY" not in os.environ:
    print("⚠️  Error: OPENAI_API_KEY environment variable not set.")
    print("   Please export it: export OPENAI_API_KEY='sk-...'")
    sys.exit(1)

# 2. Configure Logging to see the "Invisible Battle"
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
logger = logging.getLogger("SatGate-Agent")

# --- Component 1: The Wallet (The "Bank Account") ---
class MockAgentWallet(LightningWallet):
    """
    A specific wallet implementation for this Agent.
    In production, this would connect to LND, LNBits, or Alby.
    """
    def pay_invoice(self, invoice: str) -> str:
        logger.info(f"💰 WALLET: Received invoice for {len(invoice)} chars.")
        logger.info("       ... Deciding to pay ...")
        logger.info("       ... Signing transaction ...")
        
        # In a real scenario, we would call: lnd.pay(invoice)
        # For this demo, we return a mock preimage (or the one the mock server expects)
        fake_preimage = "a1b2c3d4e5f67890" * 4 
        
        logger.info("✅ WALLET: Payment Sent! Preimage secured.")
        return fake_preimage

# --- Component 2: The Agent Setup ---
def run_demo():
    print("\n🤖 --- SATGATE AGENT DEMO --- 🤖\n")

    # A. Initialize the "Credit Card" (Tool)
    wallet = MockAgentWallet()
    satgate_tool = SatGateTool(wallet=wallet)
    
    # B. Initialize the Brain (LLM)
    try:
        # Try GPT-4, fallback to 3.5-turbo if 4 isn't available to the key
        llm = ChatOpenAI(temperature=0, model_name="gpt-4")
    except Exception:
        print("⚠️  GPT-4 not available, falling back to gpt-3.5-turbo")
        llm = ChatOpenAI(temperature=0, model_name="gpt-3.5-turbo")

    # C. specific instructions for the agent
    # We give it the 'satgate_api_browser' tool which we built in the SDK
    agent = initialize_agent(
        tools=[satgate_tool], 
        llm=llm, 
        agent=AgentType.OPENAI_FUNCTIONS, # Modern "Tool Calling" agent
        verbose=True
    )

    # --- Component 3: The Mission ---
    # We give the agent a high-level goal. It must figure out:
    # 1. "I need data."
    # 2. "I should use the SatGate tool."
    # 3. "I got a 402 error? The tool handled it. I have the data now."
    
    # Use the port from the playground or default to 8081 (Aperture)
    target_url = "http://localhost:8081/api/premium/insights" 
    
    prompt = (
        f"I need the latest premium market report located at {target_url}. "
        "Fetch the content, analyze the JSON data, and tell me the 'outlook' summary."
    )

    print(f"🎯 Mission: {prompt}\n")
    
    try:
        result = agent.run(prompt)
        print(f"\n📝 AGENT FINAL REPORT:\n{result}")
    except Exception as e:
        print(f"\n❌ MISSION FAILED: {e}")
        print("   (Did you make sure your local SatGate server is running?)")

if __name__ == "__main__":
    run_demo()

