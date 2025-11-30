from typing import Optional, Type, Any
from langchain.tools import BaseTool
from pydantic import BaseModel, Field

from .client import SatGateSession, LightningWallet

class SatGateToolInput(BaseModel):
    endpoint: str = Field(description="The full URL of the premium API endpoint to fetch data from.")

class SatGateTool(BaseTool):
    name: str = "satgate_api_browser"
    description: str = (
        "Useful for fetching data from paid/premium APIs that require Lightning Network payments. "
        "Use this tool when you need to access high-value data, reports, or analytics "
        "that are behind a paywall. The tool handles payment automatically."
    )
    args_schema: Type[BaseModel] = SatGateToolInput
    
    # We exclude session from Pydantic fields since it's not a model field
    # but an internal component. However, LangChain tools often want fields to be Pydantic compatible.
    # We'll mark it as PrivateAttr or just exclude it from init if possible, 
    # but BaseTool inherits from BaseModel.
    # The standard way is to treat it as a private attribute or configured via init.
    
    _session: SatGateSession

    def __init__(self, wallet: LightningWallet, **kwargs):
        super().__init__(**kwargs)
        self._session = SatGateSession(wallet=wallet)

    def _run(self, endpoint: str) -> str:
        """Synchronous execution"""
        try:
            response = self._session.get(endpoint)
            # Raise error for 4xx/5xx if not handled (though 402 is handled inside)
            response.raise_for_status()
            return response.text
        except Exception as e:
            return f"Error fetching data: {str(e)}"

    async def _arun(self, endpoint: str) -> str:
        """Async support (Critical for high-performance agents)"""
        # For MVP, we can just wrap the sync call or use aiohttp later
        # Since SatGateSession is sync (requests), we just call _run.
        return self._run(endpoint)

