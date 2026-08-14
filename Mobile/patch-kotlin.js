const fs = require('fs');
const p1 = 'D:/Dev/Tracking App/Mobile/node_modules/react-native/gradle/libs.versions.toml';
let c1 = fs.readFileSync(p1, 'utf8');
if (c1.includes('1.9.24')) {
  c1 = c1.replaceAll('1.9.24', '1.9.25');
  fs.writeFileSync(p1, c1);
  console.log('patched libs.versions.toml');
} else {
  console.log('libs.versions.toml already patched:', c1.match(/kotlin.*/)[0]);
}

const p2 = 'D:/Dev/Tracking App/Mobile/node_modules/expo-modules-core/android/ExpoModulesCorePlugin.gradle';
let c2 = fs.readFileSync(p2, 'utf8');
if (c2.includes('"1.9.24"')) {
  c2 = c2.replaceAll('"1.9.24"', '"1.9.25"');
  fs.writeFileSync(p2, c2);
  console.log('patched ExpoModulesCorePlugin.gradle');
} else {
  console.log('ExpoModulesCorePlugin.gradle already patched');
}
console.log('done');
