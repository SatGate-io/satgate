// Names come from the actual unchanged Python AST, including dynamic mirror
// assignments. Adding a new check must fail here until explicitly mapped.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {spawnSync} from 'node:child_process';
import ts from 'typescript';
const require=createRequire(import.meta.url);
const filename=new URL('../app/verify-evidence-pack/ReceiptChecker.tsx',import.meta.url);
const source=readFileSync(filename,'utf8');
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
const exports={};new Function('exports','require',compiled)(exports,require);
const ast=spawnSync(process.env.RECEIPT_PYTHON||'python3',['-c',`
import ast,json
from pathlib import Path
root=ast.parse(Path('../tools/verify_evidence_pack.py').read_text())
def name(node):
    if isinstance(node,ast.Constant) and isinstance(node.value,str): return node.value
    if isinstance(node,ast.JoinedStr):
        return ''.join(part.value if isinstance(part,ast.Constant) else ('receipt_0' if isinstance(part.value,ast.Name) and part.value.id=='prefix' else 'dummy') for part in node.values)
    return None
keys=[]
for node in ast.walk(root):
    if isinstance(node,ast.Subscript) and isinstance(node.value,ast.Name) and node.value.id=='checks':
        if isinstance(node.slice,ast.Name) and node.slice.id=='key':
            scope=next(f for f in root.body if isinstance(f,ast.FunctionDef) and f.lineno <= node.lineno <= f.end_lineno)
            keys += [name(a.value) for a in ast.walk(scope) if isinstance(a,ast.Assign) and any(isinstance(t,ast.Name) and t.id=='key' for t in a.targets)]
        else: keys.append(name(node.slice))
    if isinstance(node,ast.Dict):
        for key,value in zip(node.keys,node.values):
            if isinstance(key,ast.Constant) and key.value=='checks' and isinstance(value,ast.Dict): keys += [name(k) for k in value.keys]
assert all(k is not None for k in keys), keys
print(json.dumps(sorted(set(keys))))
`],{encoding:'utf8'});
assert.equal(ast.status,0,ast.stderr);
const names=JSON.parse(ast.stdout);
const patterns=exports.checkGroups.flat();
const matches=(name,p)=>new RegExp('^'+p.split('*').join('[a-zA-Z0-9_]+')+'$').test(name);
for(const name of names) assert.ok(patterns.some(p=>matches(name,p)),'Unmapped Python check: '+name);
const reasons=['this file is not a SatGate receipt','it was not signed by a SatGate address','it was changed after it was signed','the signature does not match','this file is not a SatGate receipt'];
let assertions=names.length;
for(const [i,group] of exports.checkGroups.entries()) for(const p of group) {
  const name=p.replaceAll('*','0');
  assert.equal(exports.failureSentence({[name]:false}),`This receipt did not pass: ${reasons[i]}.`);
  // First group wins even if the lower-priority check is inserted first.
  for(let j=i+1;j<exports.checkGroups.length;j++) {
    const lower=exports.checkGroups[j][0].replaceAll('*','0');
    assert.equal(exports.failureSentence({[lower]:false,[name]:false}),`This receipt did not pass: ${reasons[i]}.`);assertions++;
  }
  assertions++;
}
assert.equal(exports.message({kind:'valid'}),'Signed by SatGate, and nothing in it has changed since.');
assert.equal(exports.message({kind:'runtime'}),'Check your connection and try again.');
assert.equal(exports.failureSentence({unknown:false}),'This receipt did not pass: this file is not a SatGate receipt.');
console.log(JSON.stringify({pythonNames:names,patterns:patterns.length,assertions:assertions+3}));
