"""Generate synthetic UI cases and export every verifier call from the public corpus.
Run from satgate-landing: python3 scripts/receipt-checker-corpus.py OUTPUT_DIRECTORY
Only synthetic keys, never private key serialization.
"""
import copy, importlib, json, subprocess, sys, unittest
from pathlib import Path
ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT/'tools'))
OUT = Path(sys.argv[1]); OUT.mkdir(parents=True, exist_ok=True)
import test_paid_rail as paid
import test_receipt_record as records

f = paid.Fixture()
b = records.Builder(f)
# Existing corpus builders, using an allowlisted issuer and no account identifiers.
def clean_receipt(self, *args, **kwargs):
    r = original_receipt(*args, **kwargs)
    r['issuer'] = 'https://api.satgate.io'
    r['evidence_url'] = 'https://api.satgate.io/synthetic-proof'
    r['metadata'].pop('tenant_id',None)
    return r
original_receipt = f.receipt
f.receipt = clean_receipt.__get__(f)
p = f.pack(f.receipt(decision='denied',reason='budget_exhausted'))
rpack = b.allowed(10)
m = b.manifest([rpack],issuer='https://api.satgate.io')
record = '\n'.join(json.dumps(o) for o in [rpack,m])+'\n'
other = paid.Fixture()
cases = []
def add(name,text,keys=f.jwks,expected='Valid',sentence=None):
    data = text if isinstance(text,str) else json.dumps(text)
    cases.append({'name':name,'text':data,'keys':keys,'expected':expected,'sentence':sentence})
    if expected in ('Valid','Not valid'):
        source = OUT/(name + ('.ndjson' if name.startswith('record') else '.json'))
        source.write_text(data)
        jwks = OUT/(name+'-keys.json'); jwks.write_text(json.dumps(keys))
        tool = 'verify_receipt_record.py' if name.startswith('record') else 'verify_evidence_pack.py'
        cli = subprocess.run([sys.executable,str(ROOT/'tools'/tool),str(source),'--jwks-file',str(jwks),'--require-trusted-issuer'],capture_output=True,text=True)
        verdict = json.loads(cli.stdout)
        assert verdict['valid'] == (expected=='Valid'), (name,cli.stdout)
        (OUT/(name+'-cli.json')).write_text(cli.stdout)
        print('CLI',name,cli.returncode)
add('pack-valid',p)
expired = f.receipt(decision='denied',reason='budget_exhausted'); expired['expires_at']='2026-01-02T00:00:00Z'
add('pack-expired',f.pack(expired),expected='Not valid',sentence="The receipt's dates do not pass the check.")
changed = copy.deepcopy(p); changed['receipts'][0]['route_or_tool'] += 'x'
add('pack-tampered',changed,expected='Not valid',sentence="The receipt hash doesn't match the receipt's contents.")
add('pack-other-key',p,other.jwks,expected='Not valid',sentence="The receipt's signature is missing or doesn't match the issuer's published key.")
add('pack-embedded-only',p,{'keys':[dict(other.jwks['keys'][0],kid='other-key')]},expected='Not valid')
add('record-valid',record)
bad_count = b.sign(dict(m,count=m['count']+1))
add('record-count-mismatch','\n'.join(json.dumps(o) for o in [rpack,bad_count])+'\n',expected='Not valid',sentence='This receipt record does not pass all its checks.')
record_objects = [json.loads(line) for line in record.splitlines()]
record_objects[0]['receipts'][0]['route_or_tool'] += 'x'
changed_record = '\n'.join(json.dumps(o) for o in record_objects)+'\n'
add('record-tampered',changed_record,expected='Not valid',sentence="The receipt hash doesn't match the receipt's contents.")
add('record-other-key',record,other.jwks,expected='Not valid')
add('record-embedded-only',record,{'keys':[dict(other.jwks['keys'][0],kid='other-key')]},expected='Not valid')
add('not-json','not JSON',expected='This file is not a receipt in JSON or a receipt record.')
add('empty','',expected='Choose a receipt file that is not empty.')
add('large','x'*(10*1024*1024+1),expected='Choose a receipt file smaller than 10 MiB.')
unknown = copy.deepcopy(p); unknown['issuer']='https://self-hosted.example'; unknown['receipts'][0]['issuer']='https://self-hosted.example'
add('self-hosted',unknown,expected='This checker can only check SatGate Cloud receipts.')
add('keys-blocked',p,expected="SatGate couldn't reach the issuer's published signing keys")
add('keys-malformed',p,expected="SatGate couldn't read the issuer's published signing keys")
malicious = copy.deepcopy(p); malicious['receipts'][0]['jwks_url']='https://attacker.example/jwks.json'
add('file-key-url-ignored',malicious,expected='Not valid')
(OUT/'ui-cases.json').write_text(json.dumps(cases))

# The task's tests/test_verify_evidence_pack.py does not exist at this main.
# Capture the actual tools/test_*.py corpus, without altering any verifier file.
modules = [importlib.import_module(p.stem) for p in sorted((ROOT/'tools').glob('test_*.py'))]
seen = set(); corpus = []; failures = []
def install(module, name):
    key=(id(module),name)
    if key in seen: return
    seen.add(key); original=getattr(module,name)
    def wrapped(*args,**kwargs):
        result=original(*args,**kwargs)
        try:
            item={'entry':name,'args':args,'kwargs':kwargs,'valid':result['valid']}
            encoded=json.dumps(item,ensure_ascii=True)
            corpus.append(encoded)
        except (TypeError,ValueError) as exc: failures.append(type(exc).__name__)
        return result
    setattr(module,name,wrapped)
for mod in modules:
    for obj in list(vars(mod).values()):
        if hasattr(obj,'verify_pack') and getattr(obj,'__file__','') == str(ROOT/'tools/verify_evidence_pack.py'): install(obj,'verify_pack')
        if hasattr(obj,'verify_record') and getattr(obj,'__file__','') == str(ROOT/'tools/verify_receipt_record.py'): install(obj,'verify_record')
        if isinstance(obj,dict):
            for candidate in obj.values():
                if getattr(candidate,'__file__',None)==str(ROOT/'tools/verify_evidence_pack.py'): install(candidate,'verify_pack')
suite=unittest.TestSuite(unittest.defaultTestLoader.loadTestsFromModule(m) for m in modules)
result=unittest.TextTestRunner(verbosity=1).run(suite)
# Exercise the exact CLI main functions on every exported call as well. URL
# loader interception is test-side only, for the four source-origin cases.
import contextlib, io, tempfile
cli_mismatches=[]
with tempfile.TemporaryDirectory() as tmp:
    import verify_evidence_pack as cli_pack
    import verify_receipt_record as cli_record
    for index, encoded in enumerate(corpus):
        case=json.loads(encoded); opts=case['kwargs']; entry=case['entry']
        module=cli_pack if entry=='verify_pack' else cli_record
        source=Path(tmp)/'input'; keys=Path(tmp)/'keys'
        source.write_text(json.dumps(case['args'][0]) if entry=='verify_pack' else case['args'][0])
        argv=[str(source)]
        if opts.get('jwks') is not None:
            keys.write_text(json.dumps(opts['jwks'])); argv+=['--jwks-file',str(keys)]
        if opts.get('require_trusted_issuer'): argv+=['--require-trusted-issuer']
        if opts.get('allow_mock'): argv+=['--allow-mock']
        if opts.get('now') is not None: argv+=['--now',str(opts['now'])]
        old_loader=module.load_jwks if entry=='verify_pack' else None
        if opts.get('jwks_source'):
            argv[argv.index('--jwks-file'):argv.index('--jwks-file')+2]=['--jwks-url',opts['jwks_source']]
            module.load_jwks=lambda source:opts['jwks']
        capture=io.StringIO()
        try:
            with contextlib.redirect_stdout(capture),contextlib.redirect_stderr(io.StringIO()): code=module.main(argv)
        finally:
            if old_loader: module.load_jwks=old_loader
        verdict=json.loads(capture.getvalue())
        if verdict['valid']!=case['valid']: cli_mismatches.append(index)
assert not cli_mismatches, cli_mismatches
(OUT/'parity-corpus.json').write_text(json.dumps(corpus))
(OUT/'corpus-summary.json').write_text(json.dumps({'tests':result.testsRun,'failures':len(result.failures),'errors':len(result.errors),'skipped':len(result.skipped),'captured':len(corpus),'cli_parity':len(corpus),'capture_errors':failures},indent=2))
print('Captured',len(corpus),'capture errors',failures)
if not result.wasSuccessful() or failures: sys.exit(1)
