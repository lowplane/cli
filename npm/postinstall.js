#!/usr/bin/env node
// Downloads the platform-specific Go binary from the matching GitHub Release
// and places it at vendor/costify. Designed to fail loudly on unsupported
// platforms instead of silently degrading.

const fs = require('fs');
const path = require('path');
const https = require('https');
const zlib = require('zlib');
const { pipeline } = require('stream');
const { promisify } = require('util');
const { execFileSync } = require('child_process');

const pkg = require('../package.json');
const VERSION = pkg.version;

// Skip download in CI/dev when the user is building from source.
if (process.env.COSTIFY_SKIP_POSTINSTALL === '1') {
  console.log('costify: COSTIFY_SKIP_POSTINSTALL=1, skipping binary download.');
  process.exit(0);
}

const platform = process.platform;
const arch = process.arch;

const supported = {
  'darwin-x64': 'darwin_amd64',
  'darwin-arm64': 'darwin_arm64',
  'linux-x64': 'linux_amd64',
  'linux-arm64': 'linux_arm64',
};

const key = `${platform}-${arch}`;
const target = supported[key];

if (!target) {
  console.error(`costify: unsupported platform ${key}.`);
  console.error('costify: build from source instead:');
  console.error('  go install github.com/lowplane/cli/cmd/costify@latest');
  process.exit(1);
}

const url = `https://github.com/lowplane/cli/releases/download/v${VERSION}/costify_${VERSION}_${target}.tar.gz`;
const vendorDir = path.join(__dirname, '..', 'vendor');
fs.mkdirSync(vendorDir, { recursive: true });

const tarballPath = path.join(vendorDir, 'costify.tar.gz');

const get = (u) =>
  new Promise((resolve, reject) => {
    https
      .get(u, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return resolve(get(res.headers.location));
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode} fetching ${u}`));
        }
        resolve(res);
      })
      .on('error', reject);
  });

const pipelineP = promisify(pipeline);

(async () => {
  try {
    console.log(`costify: downloading binary for ${key}...`);
    const res = await get(url);
    await pipelineP(res, fs.createWriteStream(tarballPath));
    // tar -xzf using system tar (avoids adding tar npm dep).
    execFileSync('tar', ['-xzf', tarballPath, '-C', vendorDir], { stdio: 'inherit' });
    fs.unlinkSync(tarballPath);
    console.log('costify: ready. Run `costify --version` to verify.');
  } catch (err) {
    console.error('costify: failed to install binary:', err.message);
    console.error('costify: this is non-fatal — build from source if needed:');
    console.error('  go install github.com/lowplane/cli/cmd/costify@latest');
    // Exit 0 so npm install does not abort entirely.
    process.exit(0);
  }
})();
