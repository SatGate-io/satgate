import json

import pytest

from satgate import SatGate, SatGateError
from satgate.private_beta import AttrDict
from test_issue_pay_verify import FakeResponse, RecordingSession


@pytest.fixture
def client():
    satgate = SatGate(api_key='sg_test', base_url='https://api.example.test')
    session = RecordingSession()
    session.responses = [FakeResponse(body={'decision': 'allow'})]
    satgate.session = session
    return satgate, session


@pytest.mark.parametrize('marker', [
    {'schema_version': 'satgate.receipt_bundle.v2'},
    {'schema_version': 'satgate.receipt_bundle.v99'},
    {'record_type': 'private_bundle'},
    {'openings': None},
    {'capsules': {}},
    {'opening': ''},
    {'schema_version': 'satgate.exact_call.opening.v1'},
    {'schema_version': 'satgate.execution_result.opening.v1'},
])
def test_private_evidence_is_refused_before_post(client, marker):
    satgate, session = client
    with pytest.raises(SatGateError, match='^private_evidence_not_posted$'):
        satgate.verify({**marker, 'dummy_private_field': 'dummy-private-value'})
    assert session.calls == []
    assert 'dummy_private_field' not in json.dumps(session.calls)


def test_private_bundle_response_wrapper_is_refused(client):
    satgate, session = client
    bundle = AttrDict.from_value({'schema_version': 'satgate.receipt_bundle.v2',
                                 'record_type': 'private_bundle', 'manifest': {},
                                 'artifacts': {}, 'capsules': {},
                                 'openings': {'dummy-digest': 'dummy-private-value'}})
    with pytest.raises(SatGateError, match='^private_evidence_not_posted$'):
        satgate.verify(bundle)
    assert session.calls == []
    assert 'dummy-private-value' not in json.dumps(session.calls)


def test_receipt_id_still_posts_once(client):
    satgate, session = client
    satgate.verify({'receipt_id': 'rcpt_dummy'})
    assert session.calls == [{
        'url': 'https://api.example.test/v1/verify',
        'json': {'receipt': {'receipt_id': 'rcpt_dummy'}},
        'timeout': 30,
    }]
