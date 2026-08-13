const os = require('os');
const path = require('path');
const fs = require('fs');

const HOME = process.env.HOME || process.env.USERPROFILE || os.homedir();
const INSTALL_DIR = path.join(HOME, '.keyparty', 'bin');

const PLATFORM_MAP = {
  linux: 'linux',
  darwin: 'darwin',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
  arm: 'armv6',
};

function getPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  if (!platform) throw new Error(`Unsupported platform: ${process.platform}`);
  return platform;
}

function getArch() {
  const arch = ARCH_MAP[os.arch()];
  if (!arch) throw new Error(`Unsupported architecture: ${os.arch()}`);
  return arch;
}

function getBinaryName() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === 'windows' ? '.exe' : '';
  return `keyparty-${platform}-${arch}${ext}`;
}

function getBinaryPath() {
  return path.join(INSTALL_DIR, getBinaryName());
}

function getCloudflaredName() {
  const platform = getPlatform();
  const arch = getArch();
  if (platform === 'windows') return `cloudflared-windows-${arch}.exe`;
  if (platform === 'darwin') return `cloudflared-darwin-${arch}`;
  return `cloudflared-linux-${arch}`;
}

function getCloudflaredPath() {
  return path.join(INSTALL_DIR, getCloudflaredName());
}

function getReleaseUrl(repo, filename) {
  return `https://github.com/${repo}/releases/latest/download/${filename}`;
}

function isInstalled() {
  const bin = getBinaryPath();
  return fs.existsSync(bin);
}

function ensureInstallDir() {
  if (!fs.existsSync(INSTALL_DIR)) {
    fs.mkdirSync(INSTALL_DIR, { recursive: true });
  }
}

function makeExecutable(filepath) {
  try {
    fs.chmodSync(filepath, 0o755);
  } catch (e) {
    // Windows doesn't need chmod
  }
}

module.exports = {
  HOME,
  INSTALL_DIR,
  getPlatform,
  getArch,
  getBinaryName,
  getBinaryPath,
  getCloudflaredName,
  getCloudflaredPath,
  getReleaseUrl,
  isInstalled,
  ensureInstallDir,
  makeExecutable,
};
