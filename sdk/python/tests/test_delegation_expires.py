"""Caveats.expires writes an absolute time in Unix milliseconds."""

import re
import time
from datetime import datetime, timezone

from satgate.delegation import Caveats


def _value(caveat):
    match = re.fullmatch(r"expires = (\d+)", caveat)
    assert match, caveat
    return int(match.group(1))


def test_seconds_from_now_are_unix_milliseconds():
    before = int(time.time() * 1000)
    value = _value(Caveats.expires(seconds=3600))
    after = int(time.time() * 1000)
    assert before + 3600 * 1000 - 1000 <= value <= after + 3600 * 1000 + 1000


def test_datetime_is_unix_milliseconds():
    at = datetime(2026, 11, 1, tzinfo=timezone.utc)
    assert Caveats.expires(dt=at) == f"expires = {int(at.timestamp() * 1000)}"
    assert _value(Caveats.expires(dt=at)) == 1793491200000
