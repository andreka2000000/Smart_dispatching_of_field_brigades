/* global require */
/* eslint-disable @typescript-eslint/no-require-imports -- CommonJS CLI used with the existing js-yaml dependency. */
// Run from repository root: node backend/tools/generate.cjs
const fs = require('node:fs');
const process = require('node:process');
const yaml = require('js-yaml');
const spec = yaml.load(fs.readFileSync('contracts/api_spec.yaml', 'utf8'));
const names = Object.keys(spec.components.schemas);
const field = s => s.split('_').map(x => x[0].toUpperCase()+x.slice(1)).join('');
function type(s) {
  if (s.$ref) return s.$ref.split('/').pop();
  if (s.allOf?.length === 1) return type(s.allOf[0]);
  if (s.allOf?.length > 1) {
    const merged = { type: 'object', properties: {}, required: [] };
    for (let part of s.allOf) {
      if (part.$ref) part = spec.components.schemas[part.$ref.split('/').pop()];
      Object.assign(merged.properties, part.properties);
      merged.required.push(...(part.required || []));
    }
    return type(merged);
  }
  if (s.oneOf) return 'json.RawMessage';
  switch(s.type) {
    case 'string': return 'string';
    case 'integer': return 'int64';
    case 'number': return 'float64';
    case 'boolean': return 'bool';
    case 'array': return '[]'+type(s.items);
    case 'object':
      if (!s.properties) return 'map[string]'+(typeof s.additionalProperties === 'object' ? type(s.additionalProperties) : 'json.RawMessage');
      return 'struct {\n'+Object.entries(s.properties).filter(([k]) => k !== 'fact').map(([k,v]) => {
        const optional = !(s.required||[]).includes(k);
        let t = type(v);
        if ((optional || v.nullable) && !t.startsWith('[]') && !t.startsWith('map[')) t = '*'+t;
        return `${field(k)} ${t} \`json:"${k}${optional?',omitempty':''}"\``;
      }).join('\n')+'\n}';
    default:return 'json.RawMessage';
  }
}
fs.mkdirSync('backend/internal/dto',{recursive:true});
let generated='// Code generated from contracts/api_spec.yaml; DO NOT EDIT.\npackage dto\nimport "encoding/json"\n'+names.map(n=>'type '+n+' '+type(spec.components.schemas[n])).join('\n');
// Team-owned optional response extension, kept outside the organizer contract.
generated=generated.replace('type Plan struct {\n','type Plan struct {\nOptimization *PlanOptimization `json:"optimization,omitempty"`\n');
fs.writeFileSync('backend/internal/dto/types.gen.go',generated);
if (process.argv.includes('--types-only')) process.exit(0);
fs.copyFileSync('contracts/api_spec.yaml','backend/internal/contract/api_spec.yaml');
for(const [file,key] of [['reference','ReferenceDataExample'],['matrix','TravelMatrixExample'],['day','DayInputExample']]) {
  let value=spec.components.examples[key].value;
  if(file==='day') value.absent_staff_ids=[]; // shortened reference omits ST-017
  fs.writeFileSync(`backend/testdata/${file}.json`,JSON.stringify(value,null,2)+'\n');
}
