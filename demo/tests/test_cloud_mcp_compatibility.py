"""Run the shared offline signed issuer corpus next to the demo tests."""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tools'))
from test_cloud_mcp_issuer import CloudMCPIssuerTests  # noqa: E402,F401
