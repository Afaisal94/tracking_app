const { execSync } = require('child_process');
try {
  const out = execSync('grep -r "1.9.24" "D:/Dev/Tracking App/Mobile/node_modules" --include="*.toml" --include="*.gradle" 2>&1', { encoding: 'utf8' });
  console.log(out.slice(0, 5000));
} catch (e) {
  console.log(e.stdout?.slice(0, 5000) || e.message);
}
