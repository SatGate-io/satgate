"""
SatGate LangChain Integration - Budget-Aware Tools for AI Agents

This module provides LangChain-compatible tools that automatically handle
SatGate's economic gates (402 challenges, budgets, payments).

Example usage:
    from langchain.agents import initialize_agent, AgentType
    from langchain.chat_models import ChatOpenAI
    from satgate.langchain import SatGateTool, SatGateToolkit
    
    # Create a budget-aware tool
    market_tool = SatGateTool(
        name="corporate_market_data",
        description="Fetch market data from the corporate database. Costs $0.50 per query.",
        gateway_url="https://gateway.internal",
        endpoint="/api/v1/market/query",
        budget_limit=50.00
    )
    
    # Use in a LangChain agent
    llm = ChatOpenAI(temperature=0)
    agent = initialize_agent(
        tools=[market_tool],
        llm=llm,
        agent=AgentType.OPENAI_FUNCTIONS,
        verbose=True
    )
    
    result = agent.run("Get the Q4 sales forecast")
"""

from typing import Any, Callable, Dict, List, Optional, Type, Union
from dataclasses import dataclass
import json

from .agent_client import (
    SatGateAgentClient,
    IdentityProvider,
    LightningWallet,
)
from .exceptions import (
    SatGateError,
    BudgetExceededError,
    PaymentRequiredError,
    PaymentFailedError,
)

# Try to import LangChain - make it optional
try:
    from langchain.tools import BaseTool
    from langchain.callbacks.manager import CallbackManagerForToolRun
    from pydantic import BaseModel, Field
    LANGCHAIN_AVAILABLE = True
except ImportError:
    LANGCHAIN_AVAILABLE = False
    BaseTool = object
    BaseModel = object
    Field = lambda *args, **kwargs: None
    CallbackManagerForToolRun = None


@dataclass
class ToolResult:
    """Result from a SatGate tool invocation"""
    success: bool
    data: Any
    cost: float
    budget_remaining: Optional[float]
    error: Optional[str] = None


def require_langchain():
    """Raise an error if LangChain is not installed"""
    if not LANGCHAIN_AVAILABLE:
        raise ImportError(
            "LangChain is required for this feature. "
            "Install it with: pip install langchain"
        )


class SatGateToolInput(BaseModel if LANGCHAIN_AVAILABLE else object):
    """Input schema for SatGate tools"""
    query: str = Field(description="The query or request to send to the API")
    params: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Optional parameters to include in the request"
    )


class SatGateTool(BaseTool if LANGCHAIN_AVAILABLE else object):
    """
    A LangChain tool that wraps a SatGate-protected API endpoint.
    
    Automatically handles:
    - Identity badge-in (K8s, AWS, OIDC)
    - 402 Payment Required challenges
    - Budget tracking and enforcement
    - Semantic error messages for LLM reasoning
    
    Args:
        name: Tool name (used by the agent)
        description: Tool description (helps the agent decide when to use it)
        gateway_url: SatGate gateway URL
        endpoint: API endpoint path
        method: HTTP method (GET, POST, etc.)
        identity: Identity provider ("auto", "k8s", "aws", "oidc", or IdentityProvider)
        wallet: Lightning wallet for L402 payments
        budget_limit: Maximum budget for this tool
        cost_per_call: Expected cost per call (for description)
        on_budget_alert: Callback when budget is low
    """
    
    name: str = "satgate_tool"
    description: str = "A SatGate-protected API endpoint"
    args_schema: Type[BaseModel] = SatGateToolInput if LANGCHAIN_AVAILABLE else None
    
    # SatGate configuration
    gateway_url: str = ""
    endpoint: str = "/"
    method: str = "POST"
    identity: Union[str, IdentityProvider] = "auto"
    wallet: Optional[LightningWallet] = None
    budget_limit: Optional[float] = None
    cost_per_call: Optional[float] = None
    on_budget_alert: Optional[Callable[[float, float], None]] = None
    
    # Internal state
    _client: Optional[SatGateAgentClient] = None
    _total_cost: float = 0.0
    
    def __init__(
        self,
        name: str,
        description: str,
        gateway_url: str,
        endpoint: str = "/",
        method: str = "POST",
        identity: Union[str, IdentityProvider] = "auto",
        wallet: Optional[LightningWallet] = None,
        budget_limit: Optional[float] = None,
        cost_per_call: Optional[float] = None,
        on_budget_alert: Optional[Callable[[float, float], None]] = None,
        **kwargs
    ):
        require_langchain()
        
        # Build enhanced description with cost info
        full_description = description
        if cost_per_call:
            full_description += f" (Estimated cost: ${cost_per_call:.2f} per call)"
        if budget_limit:
            full_description += f" (Budget limit: ${budget_limit:.2f})"
        
        super().__init__(
            name=name,
            description=full_description,
            **kwargs
        )
        
        self.gateway_url = gateway_url
        self.endpoint = endpoint
        self.method = method
        self.identity = identity
        self.wallet = wallet
        self.budget_limit = budget_limit
        self.cost_per_call = cost_per_call
        self.on_budget_alert = on_budget_alert
        self._total_cost = 0.0
    
    @property
    def client(self) -> SatGateAgentClient:
        """Lazy-initialize the SatGate client"""
        if self._client is None:
            self._client = SatGateAgentClient(
                gateway_url=self.gateway_url,
                identity=self.identity,
                wallet=self.wallet,
                budget_limit=self.budget_limit,
                on_budget_alert=self.on_budget_alert,
            )
        return self._client
    
    def _run(
        self,
        query: str,
        params: Optional[Dict[str, Any]] = None,
        run_manager: Optional[CallbackManagerForToolRun] = None,
    ) -> str:
        """Execute the tool synchronously"""
        try:
            # Build request body
            body = {"query": query}
            if params:
                body.update(params)
            
            # Make request
            if self.method.upper() == "GET":
                response = self.client.get(self.endpoint, params=body)
            else:
                response = self.client.post(self.endpoint, json=body)
            
            # Track cost
            self._total_cost += response.cost
            
            # Format response for LLM
            result = {
                "success": True,
                "data": response.json() if hasattr(response, 'json') else response.data,
                "cost": response.cost,
            }
            
            if response.budget_remaining is not None:
                result["budget_remaining"] = response.budget_remaining
            
            return json.dumps(result, indent=2)
            
        except BudgetExceededError as e:
            return json.dumps({
                "success": False,
                "error": "budget_exceeded",
                "message": f"I have exceeded my budget limit. Used: ${e.used:.2f}, Limit: ${e.limit:.2f}",
                "action_required": "Please approve additional budget or use a different approach.",
            }, indent=2)
            
        except PaymentRequiredError as e:
            return json.dumps({
                "success": False,
                "error": "payment_required",
                "message": f"This API requires payment: {e.amount} {e.unit}",
                "action_required": "Payment could not be completed automatically.",
            }, indent=2)
            
        except PaymentFailedError as e:
            return json.dumps({
                "success": False,
                "error": "payment_failed",
                "message": str(e),
                "action_required": "Check wallet balance or try again later.",
            }, indent=2)
            
        except SatGateError as e:
            return json.dumps({
                "success": False,
                "error": "api_error",
                "message": str(e),
            }, indent=2)
            
        except Exception as e:
            return json.dumps({
                "success": False,
                "error": "unexpected_error",
                "message": str(e),
            }, indent=2)
    
    async def _arun(
        self,
        query: str,
        params: Optional[Dict[str, Any]] = None,
        run_manager: Optional[CallbackManagerForToolRun] = None,
    ) -> str:
        """Execute the tool asynchronously (falls back to sync for now)"""
        return self._run(query, params, run_manager)
    
    @property
    def total_cost(self) -> float:
        """Total cost incurred by this tool"""
        return self._total_cost
    
    @property
    def budget_remaining(self) -> Optional[float]:
        """Remaining budget for this tool"""
        if self.budget_limit is None:
            return None
        return max(0, self.budget_limit - self._total_cost)


class SatGateRESTTool(SatGateTool):
    """
    A SatGate tool for generic REST API calls.
    
    Allows the agent to make arbitrary HTTP requests to a SatGate-protected API.
    """
    
    def __init__(
        self,
        name: str = "satgate_rest_api",
        description: str = "Make HTTP requests to a SatGate-protected REST API",
        gateway_url: str = "",
        base_path: str = "/api/v1",
        **kwargs
    ):
        super().__init__(
            name=name,
            description=description,
            gateway_url=gateway_url,
            endpoint=base_path,
            **kwargs
        )
        self.base_path = base_path
    
    def _run(
        self,
        query: str,
        params: Optional[Dict[str, Any]] = None,
        run_manager: Optional[CallbackManagerForToolRun] = None,
    ) -> str:
        """Execute a REST API call"""
        try:
            # Parse query as JSON if possible
            try:
                request = json.loads(query)
            except json.JSONDecodeError:
                # Treat as simple endpoint path
                request = {"path": query, "method": "GET"}
            
            path = request.get("path", self.endpoint)
            method = request.get("method", "GET").upper()
            body = request.get("body", params)
            
            # Ensure path starts with base_path
            if not path.startswith(self.base_path):
                path = f"{self.base_path.rstrip('/')}/{path.lstrip('/')}"
            
            # Make request
            if method == "GET":
                response = self.client.get(path)
            elif method == "POST":
                response = self.client.post(path, json=body)
            elif method == "PUT":
                response = self.client.put(path, json=body)
            elif method == "DELETE":
                response = self.client.delete(path)
            else:
                return json.dumps({"error": f"Unsupported method: {method}"})
            
            self._total_cost += response.cost
            
            return json.dumps({
                "success": True,
                "status_code": response.status_code,
                "data": response.json() if hasattr(response, 'json') else response.data,
                "cost": response.cost,
            }, indent=2)
            
        except Exception as e:
            return json.dumps({
                "success": False,
                "error": str(e),
            }, indent=2)


class SatGateToolkit:
    """
    A collection of SatGate tools for a specific gateway.
    
    Provides a convenient way to create multiple tools that share
    the same gateway configuration and budget.
    
    Example:
        toolkit = SatGateToolkit(
            gateway_url="https://gateway.internal",
            budget_limit=100.00
        )
        
        # Create tools
        market_tool = toolkit.create_tool(
            name="market_data",
            description="Get market data",
            endpoint="/api/v1/market"
        )
        
        analytics_tool = toolkit.create_tool(
            name="analytics",
            description="Run analytics queries",
            endpoint="/api/v1/analytics"
        )
        
        # Get all tools for an agent
        tools = toolkit.get_tools()
    """
    
    def __init__(
        self,
        gateway_url: str,
        identity: Union[str, IdentityProvider] = "auto",
        wallet: Optional[LightningWallet] = None,
        budget_limit: Optional[float] = None,
        on_budget_alert: Optional[Callable[[float, float], None]] = None,
    ):
        require_langchain()
        
        self.gateway_url = gateway_url
        self.identity = identity
        self.wallet = wallet
        self.budget_limit = budget_limit
        self.on_budget_alert = on_budget_alert
        self._tools: List[SatGateTool] = []
        self._shared_client: Optional[SatGateAgentClient] = None
    
    @property
    def shared_client(self) -> SatGateAgentClient:
        """Get or create the shared client"""
        if self._shared_client is None:
            self._shared_client = SatGateAgentClient(
                gateway_url=self.gateway_url,
                identity=self.identity,
                wallet=self.wallet,
                budget_limit=self.budget_limit,
                on_budget_alert=self.on_budget_alert,
            )
        return self._shared_client
    
    def create_tool(
        self,
        name: str,
        description: str,
        endpoint: str,
        method: str = "POST",
        cost_per_call: Optional[float] = None,
    ) -> SatGateTool:
        """Create a new tool in this toolkit"""
        tool = SatGateTool(
            name=name,
            description=description,
            gateway_url=self.gateway_url,
            endpoint=endpoint,
            method=method,
            identity=self.identity,
            wallet=self.wallet,
            budget_limit=self.budget_limit,
            cost_per_call=cost_per_call,
            on_budget_alert=self.on_budget_alert,
        )
        # Share the client
        tool._client = self.shared_client
        self._tools.append(tool)
        return tool
    
    def get_tools(self) -> List[SatGateTool]:
        """Get all tools in this toolkit"""
        return self._tools.copy()
    
    @property
    def total_cost(self) -> float:
        """Total cost across all tools"""
        return self.shared_client.total_cost
    
    @property
    def budget_remaining(self) -> Optional[float]:
        """Remaining budget across all tools"""
        return self.shared_client.budget_remaining


# Convenience function for quick tool creation
def create_satgate_tool(
    name: str,
    description: str,
    gateway_url: str,
    endpoint: str,
    **kwargs
) -> SatGateTool:
    """
    Convenience function to create a SatGate tool.
    
    Example:
        from satgate.langchain import create_satgate_tool
        
        tool = create_satgate_tool(
            name="weather",
            description="Get weather data",
            gateway_url="https://api.weather.com",
            endpoint="/v1/forecast",
            budget_limit=10.00
        )
    """
    return SatGateTool(
        name=name,
        description=description,
        gateway_url=gateway_url,
        endpoint=endpoint,
        **kwargs
    )
