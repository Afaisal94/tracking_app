const fs = require('fs');
const files = [
  'D:/Dev/Tracking App/Mobile/node_modules/@react-native/gradle-plugin/gradle/libs.versions.toml',
  'D:/Dev/Tracking App/Mobile/node_modules/expo-modules-core/android/build.gradle',
  'D:/Dev/Tracking App/Mobile/node_modules/expo-modules-core/android/ExpoModulesCorePlugin.gradle',
  'D:/Dev/Tracking App/Mobile/node_modules/react-native/gradle/libs.versions.toml',
];
for (const p of files) {
  try {
    let c = fs.readFileSync(p, 'utf8');
    if (p.endsWith('build.gradle')) {
      // Fix compose mapping: keep 1.9.24->1.5.14 and 1.9.25->1.5.15
      c = c.replace('"1.9.25": "1.5.14"', '"1.9.24": "1.5.14"');
      // Don't blindly replace 1.9.24 in build.gradle (it breaks the map)
    } else {
      const before = c.includes('1.9.24');
      c = c.replaceAll('1.9.24', '1.9.25');
      console.log(p, before ? 'patched' : 'already ok');
    }
    fs.writeFileSync(p, c);
    if (p.endsWith('build.gradle')) console.log(p, 'compose map fixed');
  } catch (e) { console.log(p, 'error', e.message); }
}
console.log('done');
