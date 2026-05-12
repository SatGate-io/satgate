"""Private-beta issue/pay/verify facade for SatGate Cloud."""

from types import SimpleNamespace
from typing import Any, Dict, Optional

import requests

from .exceptions import SatGateAuthError, SatGateError

BETA_ACCESS_MESSAGE = (
    "This API namespace requires private beta access. "
    "Visit cloud.satgate.io/docs to request access."
)


class AttrDict(SimpleNamespace):
    """Small response wrapper so examples can use receipt.foo attributes."""

    @classmethod
    def from_value(cls, value: Any) -> Any:
        if isinstance(value, dict):
            return cls(**{key: cls.from_value(val) for key, val in value.items()})
        if isinstance(value, list):
            return [cls.from_value(item) for item in value]
        return value

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {}
        for key, value in self.__dict__.items():
            if isinstance(value, AttrDict):
                result[key] = value.to_dict()
            elif isinstance(value, list):
                result[key] = [item.to_dict() if isinstance(item, AttrDict) else item for item in value]
            else:
                result[key] = value
        return result


class SatGate:
    """SatGate Cloud issue/pay/verify client.

    The public package installs today. The issue/pay/verify API namespace is
    private beta; unauthenticated or non-beta credentials raise SatGateAuthError
    with a docs CTA instead of a stack trace or mocked receipt.
    """

    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: str = "https://api.satgate.io",
        tenant: Optional[str] = None,
        timeout: int = 30,
    ):
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.tenant = tenant
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})
        if api_key:
            self.session.headers.update({"Authorization": f"Bearer {api_key}"})
        if tenant:
            self.session.headers.update({"X-SatGate-Tenant": tenant})

    def issue(self, **payload: Any) -> AttrDict:
        """Issue a scoped capability for an agent task."""
        return self._post("/v1/capabilities", payload)

    def pay(self, **payload: Any) -> AttrDict:
        """Route a value-bearing call under a scoped capability."""
        return self._post("/v1/pay", self._serialize(payload))

    def verify(self, receipt: Any) -> AttrDict:
        """Verify a SatGate receipt and return proof metadata."""
        payload = receipt.to_dict() if hasattr(receipt, "to_dict") else receipt
        return self._post("/v1/verify", {"receipt": self._serialize(payload)})

    def _post(self, path: str, payload: Dict[str, Any]) -> AttrDict:
        if not self.api_key:
            raise SatGateAuthError(BETA_ACCESS_MESSAGE, status_code=401)

        try:
            response = self.session.post(
                f"{self.base_url}{path}",
                json=payload,
                timeout=self.timeout,
            )
        except requests.RequestException as exc:
            raise SatGateError(f"Request failed: {exc}") from exc

        if response.status_code in {401, 403, 404, 503}:
            raise SatGateAuthError(BETA_ACCESS_MESSAGE, status_code=response.status_code)

        if response.status_code >= 400:
            try:
                body = response.json()
                message = body.get("message") or body.get("error") or response.text
            except Exception:
                message = response.text
            raise SatGateError(f"API error {response.status_code}: {message}")

        if not response.content:
            return AttrDict()
        return AttrDict.from_value(response.json())

    def _serialize(self, value: Any) -> Any:
        if isinstance(value, AttrDict):
            return value.to_dict()
        if isinstance(value, dict):
            return {key: self._serialize(val) for key, val in value.items()}
        if isinstance(value, list):
            return [self._serialize(item) for item in value]
        return value
