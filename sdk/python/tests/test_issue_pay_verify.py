import pytest

from satgate import SatGate, SatGateAuthError


class FakeResponse:
    def __init__(self, status_code=200, body=None):
        self.status_code = status_code
        self._body = body or {}
        self.content = b"{}"
        self.text = ""

    def json(self):
        return self._body


class RecordingSession:
    def __init__(self):
        self.headers = {}
        self.calls = []
        self.responses = [
            FakeResponse(body={"id": "cap_123"}),
            FakeResponse(body={"receipt_id": "rcpt_123"}),
            FakeResponse(body={"decision": "allow", "evidence_pack_id": "ep_123"}),
        ]

    def post(self, url, json, timeout):
        self.calls.append({"url": url, "json": json, "timeout": timeout})
        return self.responses.pop(0)


def test_issue_without_credentials_returns_docs_cta_without_network():
    satgate = SatGate()

    with pytest.raises(SatGateAuthError) as exc:
        satgate.issue(
            task="summarize vendor invoice",
            agent="invoice-agent",
            allow=["POST /v1/invoices/*"],
            budget_usd=0.25,
            expires_in="10m",
        )

    assert "cloud.satgate.io/docs" in str(exc.value)


def test_issue_falls_back_to_early_capabilities_endpoint_on_404():
    satgate = SatGate(api_key="sg_test", base_url="https://api.example.test")
    session = RecordingSession()
    session.responses = [FakeResponse(status_code=404), FakeResponse(body={"id": "cap_compat"})]
    satgate.session = session

    capability = satgate.issue(
        task="summarize vendor invoice",
        agent="invoice-agent",
        allow=["POST /v1/invoices/*"],
    )

    assert [call["url"] for call in session.calls] == [
        "https://api.example.test/v1/issue",
        "https://api.example.test/v1/capabilities",
    ]
    assert capability.id == "cap_compat"


def test_issue_pay_verify_use_primitive_endpoints_and_copy_paste_payloads():
    satgate = SatGate(api_key="sg_test", base_url="https://api.example.test", tenant="tenant_123")
    session = RecordingSession()
    satgate.session = session

    capability = satgate.issue(
        task="summarize vendor invoice",
        agent="invoice-agent",
        allow=["POST /v1/invoices/*"],
        budget_usd=0.25,
        expires_in="10m",
    )
    receipt = satgate.pay(
        upstream="https://api.vendor.test/v1/invoices/42",
        capability=capability,
        max_usd=0.10,
    )
    verified = satgate.verify(receipt)

    assert session.calls[0] == {
        "url": "https://api.example.test/v1/issue",
        "json": {
            "task": "summarize vendor invoice",
            "agent": "invoice-agent",
            "allow": ["POST /v1/invoices/*"],
            "budget_usd": 0.25,
            "expires_in": "10m",
        },
        "timeout": 30,
    }
    assert session.calls[1]["url"] == "https://api.example.test/v1/pay"
    assert session.calls[1]["json"] == {
        "upstream": "https://api.vendor.test/v1/invoices/42",
        "capability": {"id": "cap_123"},
        "max_usd": 0.10,
    }
    assert session.calls[2]["url"] == "https://api.example.test/v1/verify"
    assert session.calls[2]["json"] == {"receipt": {"receipt_id": "rcpt_123"}}
    assert verified.decision == "allow"
    assert verified.evidence_pack_id == "ep_123"
