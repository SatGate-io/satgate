"""parse_rfc3339 accepts every RFC 3339 date-time shape and nothing else.

Round 2: the string is checked against the RFC 3339 section 5.6 grammar (strict
regex, ASCII digits, then range checks) BEFORE any normalisation or
datetime.fromisoformat. fromisoformat tolerates non-RFC3339 shapes (missing
seconds, space separator, offsets with seconds, +00:60); those are now rejected.
Round 1 wrongly widened that hole by accepting lowercase "z" (Astra, NO-GO).

Go's time.RFC3339Nano trims trailing zeros, so receipts carry 1-9 fractional
digits. Python < 3.11 datetime.fromisoformat accepts only 3 or 6 digits, which
made valid receipts fail with malformed_issued_at on Python 3.9/3.10 (the
default python3 on macOS). Run:
  python -B -m unittest discover -s tools -p 'test_rfc3339_fraction.py' -v
"""
import copy
import importlib.util
import re
import sys
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

import test_paid_rail as paid  # noqa: E402

COPIES = {
    "tools": ROOT / "tools" / "verify_evidence_pack.py",
    "demo": ROOT / "demo" / "verify_evidence_pack.py",
}

# What Python 3.9/3.10 datetime.fromisoformat accepts for an aware timestamp:
# a fraction of exactly 3 or 6 digits, and a numeric offset (no "Z").
_PY39_ISO = re.compile(
    r"^\d{4}-\d\d-\d\d[T ]\d\d:\d\d:\d\d(\.(\d{3}|\d{6}))?([+-]\d\d:\d\d)?$"
)


class _Py39DateTime(datetime):
    """datetime whose fromisoformat enforces the Python 3.9/3.10 fraction rule."""

    @classmethod
    def fromisoformat(cls, value):
        if not _PY39_ISO.match(value):
            raise ValueError(f"Invalid isoformat string: {value!r}")
        return datetime.fromisoformat(value)


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def utc(y, mo, d, h, mi, s, us=0, offset_minutes=0):
    local = datetime(y, mo, d, h, mi, s, us, tzinfo=timezone(timedelta(minutes=offset_minutes)))
    return local.astimezone(timezone.utc)


# (input, expected UTC instant). Fractions are truncated (not rounded) to microseconds.
ACCEPT = [
    ("2026-07-15T12:56:55Z", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55z", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55+00:00", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55.1Z", utc(2026, 7, 15, 12, 56, 55, 100000)),
    ("2026-07-15T12:56:55.12Z", utc(2026, 7, 15, 12, 56, 55, 120000)),
    ("2026-07-15T12:56:55.123Z", utc(2026, 7, 15, 12, 56, 55, 123000)),
    ("2026-07-15T12:56:55.1234Z", utc(2026, 7, 15, 12, 56, 55, 123400)),
    ("2026-07-15T12:56:55.12345Z", utc(2026, 7, 15, 12, 56, 55, 123450)),
    ("2026-07-15T12:56:55.123456Z", utc(2026, 7, 15, 12, 56, 55, 123456)),
    ("2026-07-15T12:56:55.1234567Z", utc(2026, 7, 15, 12, 56, 55, 123456)),
    ("2026-07-15T12:56:55.12345678Z", utc(2026, 7, 15, 12, 56, 55, 123456)),
    ("2026-07-15T12:56:55.123456789Z", utc(2026, 7, 15, 12, 56, 55, 123456)),
    ("2026-07-15T12:56:55.999999999Z", utc(2026, 7, 15, 12, 56, 55, 999999)),
    ("2026-07-15T12:56:55.9z", utc(2026, 7, 15, 12, 56, 55, 900000)),
    ("2026-07-15T12:56:55.45966z", utc(2026, 7, 15, 12, 56, 55, 459660)),
    # Trailing zeros are not significant.
    ("2026-07-15T12:56:55.000Z", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55.0Z", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55.000000000Z", utc(2026, 7, 15, 12, 56, 55)),
    # Numeric offsets with every fraction length, positive and negative.
    ("2026-07-15T12:56:55.5+05:30", utc(2026, 7, 15, 12, 56, 55, 500000, 330)),
    ("2026-07-15T12:56:55.53-04:00", utc(2026, 7, 15, 12, 56, 55, 530000, -240)),
    ("2026-07-15T12:56:55.4567+00:00", utc(2026, 7, 15, 12, 56, 55, 456700)),
    ("2026-07-15T12:56:55.45678-07:00", utc(2026, 7, 15, 12, 56, 55, 456780, -420)),
    ("2026-07-15T12:56:55.456789+01:00", utc(2026, 7, 15, 12, 56, 55, 456789, 60)),
    ("2026-07-15T12:56:55.123456789-04:00", utc(2026, 7, 15, 12, 56, 55, 123456, -240)),
    ("2026-07-15T12:56:55.12345678+14:00", utc(2026, 7, 15, 12, 56, 55, 123456, 840)),
    ("2026-07-15T23:59:59.9999999-12:00", utc(2026, 7, 15, 23, 59, 59, 999999, -720)),
]

# Astra's matrix: shapes datetime.fromisoformat tolerates (or round 1 let in) that
# are NOT RFC 3339 section 5.6. Each is rejected as malformed_<field> on every
# Python. Comment = behaviour before this round on (3.9 / 3.11):
#   main = origin/main before PR #226, r1 = PR head before round 2, new = now.
NON_RFC3339_SHAPES = [
    "2026-07-15T12:56z",        # main: reject, r1: ACCEPT (Astra's finding), new: reject
    "2026-07-15T12:56Z",        # main/r1: accept (no seconds), new: reject
    "2026-07-15T12:56+00:00",   # main/r1: accept (no seconds), new: reject
    "2026-07-15 12:56:55Z",     # main/r1: accept (space separator), new: reject
    "2026-07-15 12:56:55+00:00",  # main/r1: accept, new: reject
    "2026-07-15 12:56:55z",
    "2026-07-15T12:56:55+00:00:30",  # main/r1: accept (seconds in offset), new: reject
    "2026-07-15T12:56:55+00:00:00",
    "2026-07-15T12:56:55+00",   # main/r1: reject on 3.9, accept on 3.11; new: reject on both
    "2026-07-15T12:56:55+0000",  # same: 3.9 reject, 3.11 accept; new: reject on both
    "20260715T125655Z",         # 3.11 accepts basic format; new: reject
    "2026-07-15T12:56:55,5Z",   # 3.11 accepts comma fraction; new: reject
    "2026-07-15T12:56:55.Z",    # main: accept (empty fraction), r1/new: reject
    "2026-07-15T12:56:55.+00:00",
    "2026-07-15T\u0661\u0662:56:55Z",   # non-ASCII digits
    "\uff12026-07-15T12:56:55Z",         # fullwidth digit
    "2026-07-15T12:56:55Z ",    # trailing whitespace
    " 2026-07-15T12:56:55Z",    # leading whitespace
    "2026-07-15T12:56:55Z\n",   # trailing newline ($ would allow it; fullmatch does not)
    "2026-07-15T12:56:55\tZ",
]

# Grammar matches but a field is out of range.
RANGE_ERRORS = [
    "2026-00-10T12:00:00Z",   # month 00
    "2026-13-01T12:00:00Z",   # month 13
    "2026-07-00T12:00:00Z",   # day 00
    "2026-07-32T12:00:00Z",
    "2026-04-31T12:00:00Z",   # 30-day month
    "2026-02-29T12:00:00Z",   # not a leap year
    "2100-02-29T12:00:00Z",   # century, not a leap year
    "2026-07-15T24:00:00Z",   # hour 24
    "2026-07-15T12:60:00Z",   # minute 60
    "2026-07-15T12:56:61Z",   # second 61
    "2026-07-15T12:56:99Z",
    # second 60 passes the section 5.6 range check (leap second) but datetime has
    # no leap seconds, so it is rejected (fail closed). Go's time.Parse rejects
    # it too, so gateway receipts never carry it.
    "2016-12-31T23:59:60Z",
    "2026-07-15T12:56:55+24:00",   # offset hour 24
    "2026-07-15T12:56:55-24:00",
    "2026-07-15T12:56:55+00:60",   # offset minute 60 (main/r1: accepted, read as +01:00)
    "2026-07-15T12:56:55-05:99",
    "0000-01-01T00:00:00Z",        # year 0
    # datetime overflow after offset conversion: main/r1 raised OverflowError
    # (uncaught crash); now a clean malformed_<field>.
    "9999-12-31T23:59:59-01:00",
    "0001-01-01T00:00:00+01:00",
]

# Valid RFC 3339 that must stay accepted (also in ACCEPT above).
ACCEPT += [
    ("2026-07-15t12:56:55z", utc(2026, 7, 15, 12, 56, 55)),             # lowercase t and z
    ("2026-07-15t12:56:55+00:00", utc(2026, 7, 15, 12, 56, 55)),
    ("2026-07-15T12:56:55-00:00", utc(2026, 7, 15, 12, 56, 55)),        # RFC 3339 "unknown local offset"
    ("2026-07-15T12:56:55.5-00:00", utc(2026, 7, 15, 12, 56, 55, 500000)),
    ("2026-07-15T12:56:55+23:59", utc(2026, 7, 15, 12, 56, 55, 0, 1439)),
    ("2026-07-15T12:56:55-23:59", utc(2026, 7, 15, 12, 56, 55, 0, -1439)),
    ("2024-02-29T12:00:00Z", utc(2024, 2, 29, 12, 0, 0)),               # leap day
    ("2000-02-29T12:00:00Z", utc(2000, 2, 29, 12, 0, 0)),               # 400-year leap
    ("2026-12-31T23:59:59Z", utc(2026, 12, 31, 23, 59, 59)),
    ("0001-01-01T00:00:00Z", utc(1, 1, 1, 0, 0, 0)),
    ("9999-12-31T23:59:59.999999999Z", utc(9999, 12, 31, 23, 59, 59, 999999)),
    ("2026-01-01T00:00:00.000000001Z", utc(2026, 1, 1, 0, 0, 0)),       # nanosecond truncates to 0us
]

REJECT = [
    "2026-07-15T12:56:55",            # no timezone
    "2026-07-15T12:56:55.5",          # fraction, no timezone
    "2026-07-15T12:56:55.123456789",  # nanosecond fraction, no timezone
    "2026-07-15",                     # date only
    "not-a-timestamp",
    "garbage",
    "2026-07-15T12:56:55.Z",          # empty fraction
    "2026-07-15T12:56:55.z",
    "2026-07-15T12:56:55.+00:00",
    "2026-07-15T12:56:55.-04:00",
    "2026-07-15T12:56:55.nopeZ",      # non-digit fraction
    "2026-07-15T12:56:55.1aZ",
    "2026-07-15T12:56:55.1234567aZ",  # non-digit after the truncation point
    "2026-07-15T12:56:55.12 3Z",
    "2026-07-15T12:56:55.+5Z",
    "2026-07-15T12:56:55.\u0663Z",     # non-ASCII digit
    "2026-07-15T12:56:55.5ZZ",
    "2026-07-15T12:56:55.5+25:00",    # impossible offset
    "2026-13-15T12:56:55.5Z",         # impossible date
    "2026-07-15T12:56:55.5Zjunk",
] + NON_RFC3339_SHAPES + RANGE_ERRORS


def parse(module, value):
    reasons, codes = [], []
    parsed = module.parse_rfc3339(value, reasons, codes, "issued_at")
    return parsed, reasons, codes


class FractionTable:
    """Mixin: run the accept/reject tables against one verifier copy."""

    module = None

    def check_tables(self):
        for value, expected in ACCEPT:
            with self.subTest(accept=value):
                parsed, reasons, codes = parse(self.module, value)
                self.assertEqual(parsed, expected, (value, reasons, codes))
                self.assertEqual(reasons, [])
                self.assertEqual(codes, [])
        for value in REJECT:
            with self.subTest(reject=value):
                parsed, reasons, codes = parse(self.module, value)
                self.assertIsNone(parsed, value)
                self.assertEqual(codes, ["malformed_issued_at"], value)
                self.assertEqual(reasons, ["receipt.issued_at is not RFC3339"], value)

    def test_every_fraction_length_zero_to_nine(self):
        for digits in range(0, 10):
            frac = "." + "".join(str(1 + i % 9) for i in range(digits)) if digits else ""
            for tz in ("Z", "z", "+00:00", "-04:00", "+05:30"):
                value = f"2026-07-15T12:56:55{frac}{tz}"
                with self.subTest(value=value):
                    parsed, reasons, codes = parse(self.module, value)
                    self.assertIsNotNone(parsed, (value, reasons, codes))
                    self.assertEqual(codes, [])
                    self.assertEqual(parsed.tzinfo, timezone.utc)

    def test_missing_value_is_missing_not_malformed(self):
        for value in (None, "", 123):
            with self.subTest(value=value):
                parsed, _, codes = parse(self.module, value)
                self.assertIsNone(parsed)
                self.assertEqual(codes, ["missing_issued_at"])

    def test_native_python_tables(self):
        self.check_tables()

    def test_python39_strict_fromisoformat(self):
        """Pin Python 3.9/3.10 behaviour on any interpreter.

        Swap the module's datetime for one whose fromisoformat only accepts a
        3- or 6-digit fraction, exactly as Python < 3.11 does.
        """
        # Sanity: the stand-in really rejects what real 3.9 rejects, so this
        # test would have failed against the old (truncate-only) code.
        for bad in ("2026-07-15T12:56:55.9+00:00", "2026-07-15T12:56:55.45966+00:00",
                    "2026-07-15T12:56:55.1234+00:00", "2026-07-15T12:56:55Z",
                    "2026-07-15T12:56:55.123456789+00:00"):
            with self.assertRaises(ValueError):
                _Py39DateTime.fromisoformat(bad)
        original = self.module.datetime
        self.module.datetime = _Py39DateTime
        try:
            self.check_tables()
        finally:
            self.module.datetime = original


class ToolsCopy(FractionTable, unittest.TestCase):
    module = load(COPIES["tools"], "verify_tools_fraction")


class DemoCopy(FractionTable, unittest.TestCase):
    module = load(COPIES["demo"], "verify_demo_fraction")


class CopiesStayInSync(unittest.TestCase):
    def test_parse_rfc3339_identical_across_copies(self):
        def body(path):
            text = path.read_text()
            start = text.index("def parse_rfc3339(")
            return text[start:text.index("def issuer_jwks_url(")]

        self.assertEqual(body(COPIES["tools"]), body(COPIES["demo"]))


class SignedReceiptRejectsNonRfc3339(unittest.TestCase):
    """Astra's scenario: a validly signed receipt whose issued_at is not RFC 3339.

    Round 1 verified `...T12:56z` as valid (lowercase z + fromisoformat's
    tolerance for a missing seconds field). The signature is fine; the
    timestamp grammar is what must fail.
    """

    SHAPES = (
        "2026-01-01T00:00z",           # Astra's case
        "2026-01-01T00:00Z",
        "2026-01-01 00:00:00Z",
        "2026-01-01T00:00:00+00:00:30",
        "2026-01-01T00:00:00+00:60",
    )

    def check(self, tools):
        for stamp in self.SHAPES:
            with self.subTest(issued_at=stamp):
                f = paid.Fixture()
                receipt = f.receipt("denied", "capability_invalid")
                receipt["issued_at"] = stamp
                signed = f.sign(receipt)
                self.assertEqual(signed["issued_at"], stamp)
                result = tools.verify_pack(f.pack(receipt), jwks=f.jwks, now=paid.NOW, allow_mock=True)
                self.assertFalse(result["valid"], result)
                self.assertIn("malformed_issued_at", result["reason_codes"])
                # The signature itself is genuine: only the grammar check fails.
                self.assertTrue(result["checks"]["receipt_0_signature_valid"], result["checks"])

    def test_tools_copy(self):
        self.check(load(COPIES["tools"], "verify_tools_astra"))

    def test_demo_copy(self):
        self.check(load(COPIES["demo"], "verify_demo_astra"))

    def test_valid_rfc3339_with_seconds_still_verifies(self):
        tools = load(COPIES["tools"], "verify_tools_astra_ok")
        for stamp in ("2026-01-01T00:00:00z", "2026-01-01t00:00:00Z", "2026-01-01T00:00:00-00:00",
                      "2026-01-01T00:00:00.123456789+00:00"):
            with self.subTest(issued_at=stamp):
                f = paid.Fixture()
                receipt = f.receipt("denied", "capability_invalid")
                receipt["issued_at"] = stamp
                receipt["timestamp"] = stamp
                result = tools.verify_pack(f.pack(receipt), jwks=f.jwks, now=paid.NOW, allow_mock=True)
                self.assertTrue(result["valid"], result)


class SignedReceiptKeepsOriginalString(unittest.TestCase):
    """Normalization is validation-only: hash and signature use the original."""

    def test_short_fraction_receipt_verifies_under_py39_rules(self):
        tools = load(COPIES["tools"], "verify_tools_fraction_signed")
        original = tools.datetime
        tools.datetime = _Py39DateTime
        try:
            for stamp in ("2026-01-01T00:00:00.5Z", "2026-01-01T00:00:00.12Z",
                          "2026-01-01T00:00:00.1234Z", "2026-01-01T00:00:00.12345Z",
                          "2026-01-01T00:00:00.123456789Z"):
                with self.subTest(issued_at=stamp):
                    f = paid.Fixture()
                    receipt = f.receipt("denied", "capability_invalid")
                    receipt["issued_at"] = stamp
                    receipt["timestamp"] = stamp
                    signed = f.sign(receipt)
                    # The signed bytes carry the original, un-padded string.
                    self.assertEqual(signed["issued_at"], stamp)
                    self.assertIn(stamp.encode(), tools.canonical_receipt_payload(signed))
                    pack = f.pack(receipt)
                    result = tools.verify_pack(pack, jwks=f.jwks, now=paid.NOW, allow_mock=True)
                    self.assertTrue(result["valid"], result)
                    self.assertNotIn("malformed_issued_at", result.get("reason_codes", []))
                    self.assertNotIn("malformed_timestamp", result.get("reason_codes", []))

                    # Rewriting the string to its "normalized" form breaks the signature.
                    tampered = copy.deepcopy(pack)
                    norm = stamp[:stamp.index(".")] + "." + stamp[stamp.index(".") + 1:-1][:6].ljust(6, "0") + "Z"
                    tampered["receipts"][0]["issued_at"] = norm
                    bad = tools.verify_pack(tampered, jwks=f.jwks, now=paid.NOW, allow_mock=True)
                    self.assertFalse(bad["valid"], bad)
        finally:
            tools.datetime = original


if __name__ == "__main__":
    unittest.main()
