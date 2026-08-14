const fs = require('fs');
const path = require('path');
const root = 'D:/Dev/Tracking App/Mobile/node_modules';
let found = [];
function walk(dir) {
  let entries;
  try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch { return; }
  for (const e of entries) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) {
      if (e.name.startsWith('.')) continue;
      walk(p);
    } else if (e.name.endsWith('.toml') || e.name.endsWith('.gradle')) {
      try {
        const c = fs.readFileSync(p, 'utf8');
        if (c.includes('1.9.24')) found.push(p);
      } catch {}
    }
  }
}
walk(root);
console.log(found.join('\n') || 'none');
