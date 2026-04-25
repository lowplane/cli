#!/usr/bin/env node
// Wrapper that invokes the platform-specific costify binary fetched by
// postinstall.js. Stays small and dependency-free on purpose.

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const binaryName = process.platform === 'win32' ? 'costify.exe' : 'costify';
const binaryPath = path.join(__dirname, '..', 'vendor', binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error('costify: binary not found at', binaryPath);
  console.error('costify: try `npm install -g @costify/cost` again, or build from source:');
  console.error('  go install github.com/lowplane/cli/cmd/costify@latest');
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: true,
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 1);
  }
});

child.on('error', (err) => {
  console.error('costify: failed to launch binary:', err.message);
  process.exit(1);
});
