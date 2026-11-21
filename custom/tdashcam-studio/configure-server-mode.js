const fs = require('fs');
const path = require('path');

const HTML_DIR = path.join(__dirname, 'html');
const NGINX_DIR = '/etc/nginx/server-mode';
const NGINX_CONF = path.join(NGINX_DIR, 'media.conf');
const CONFIG_JS = path.join(HTML_DIR, 'server-config.js');
const MEDIA_URL = '/media/';

function parseList(value, fallback) {
  if (value === undefined) return fallback;
  return value.split(',').map((item) => item.trim().replace(/^\/+|\/+$/g, '')).filter(Boolean);
}

function fail(message) {
  console.error(`[ServerMode] ${message}`);
  process.exit(1);
}

const mediaRoot = (process.env.MEDIA_ROOT || '').trim().replace(/\/+$/, '');

fs.mkdirSync(NGINX_DIR, { recursive: true });

if (!mediaRoot) {
  fs.rmSync(NGINX_CONF, { force: true });
  fs.writeFileSync(CONFIG_JS, 'window.TDASHCAM_SERVER = null;\n');
  console.log('[ServerMode] MEDIA_ROOT not set, serving the local-folder player only.');
  process.exit(0);
}

if (!/^\/[A-Za-z0-9._\/-]+$/.test(mediaRoot) || mediaRoot.split('/').includes('..')) {
  fail(`MEDIA_ROOT must be an absolute path made of letters, digits, ".", "_", "-" and "/": ${mediaRoot}`);
}
if (!fs.existsSync(mediaRoot) || !fs.statSync(mediaRoot).isDirectory()) {
  fail(`MEDIA_ROOT is not a directory inside the container: ${mediaRoot}`);
}

const libraryDirs = parseList(process.env.LIBRARY_DIRS, []);
const excludeDirs = parseList(process.env.LIBRARY_EXCLUDE_DIRS, ['EncryptedClips']);

fs.writeFileSync(NGINX_CONF, [
  `location ${MEDIA_URL} {`,
  `    alias ${mediaRoot}/;`,
  '    autoindex on;',
  '    autoindex_format json;',
  '}',
  '',
].join('\n'));

fs.writeFileSync(CONFIG_JS, `window.TDASHCAM_SERVER = ${JSON.stringify({ mediaUrl: MEDIA_URL, libraryDirs, excludeDirs })};\n`);

console.log(`[ServerMode] Serving ${mediaRoot} at ${MEDIA_URL}`);
console.log(`[ServerMode] Library dirs: ${libraryDirs.length ? libraryDirs.join(', ') : 'auto-discover'}; excluded: ${excludeDirs.join(', ') || 'none'}`);
