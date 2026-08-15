const https = require('https');
const http = require('http');
const fs = require('fs');
const { c, colorize, spinner, progressBar, icon } = require('./tui');
const {
  getBinaryName, getBinaryPath, getCloudflaredName, getCloudflaredPath,
  getReleaseUrl, isInstalled, ensureInstallDir, makeExecutable, INSTALL_DIR,
} = require('./platform');

const KEPARTY_REPO = 'MastaChief117/keyparty';
const CLOUDFLARED_REPO = 'cloudflare/cloudflared';

// ── HTTP Download with Redirects ────────────────────────────────────────
function download(url, dest, redirectCount = 0) {
  if (redirectCount > 5) {
    return Promise.reject(new Error('Too many redirects'));
  }
  return new Promise((resolve, reject) => {
    const proto = url.startsWith('https') ? https : http;

    const req = proto.get(url, { headers: { 'User-Agent': 'keyparty-npm' } }, (res) => {
      // Follow redirects
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return download(res.headers.location, dest, redirectCount + 1).then(resolve).catch(reject);
      }

      if (res.statusCode !== 200) {
        reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        return;
      }

      const total = parseInt(res.headers['content-length'], 10);
      let downloaded = 0;
      const file = fs.createWriteStream(dest);

      res.on('data', (chunk) => {
        downloaded += chunk.length;
        if (total) {
          const pct = (downloaded / total * 100).toFixed(0);
          process.stdout.write(`\r  ⬇ ${colorize(c.cyan, progressBar(downloaded / total * 100))} ${colorize(c.dim, formatBytes(downloaded) + '/' + formatBytes(total))}  `);
        }
      });

      res.pipe(file);

      res.on('error', (err) => {
        file.close();
        fs.unlink(dest, () => {});
        reject(err);
      });

      file.on('finish', () => {
        file.close();
        process.stdout.write('\r' + ' '.repeat(80) + '\r');
        resolve();
      });

      file.on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    });

    req.on('error', reject);
    req.setTimeout(30000, () => {
      req.destroy();
      reject(new Error('Download timed out'));
    });
  });
}

function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / 1048576).toFixed(1) + ' MB';
}

// ── Install KeyParty Binary ─────────────────────────────────────────────
async function installBinary() {
  if (isInstalled()) {
    const sp = spinner('KeyParty binary found');
    sp.succeed(`Binary ready at ${colorize(c.dim, getBinaryPath())}`);
    return true;
  }

  console.log(`\n  ${icon.spark} ${colorize(c.bold + c.white, 'Installing KeyParty binary...')}`);

  ensureInstallDir();
  const binaryName = getBinaryName();
  const dest = getBinaryPath();
  const url = getReleaseUrl(KEPARTY_REPO, binaryName);

  const sp = spinner(`Downloading ${binaryName}`);
  try {
    await download(url, dest);
    makeExecutable(dest);
    sp.succeed(`Downloaded ${colorize(c.green, binaryName)}`);
    return true;
  } catch (err) {
    sp.fail(`Failed to download binary`);
    console.log(`\n  ${icon.info} ${colorize(c.dim, 'Manual install: go build -o keyparty . && cp keyparty ' + INSTALL_DIR + '/')}`);
    return false;
  }
}

// ── Install Cloudflared ─────────────────────────────────────────────────
async function installCloudflared() {
  const cfPath = getCloudflaredPath();
  if (fs.existsSync(cfPath)) {
    const sp = spinner('Cloudflared found');
    sp.succeed(`Cloudflared ready at ${colorize(c.dim, cfPath)}`);
    return true;
  }

  console.log(`\n  ${icon.globe} ${colorize(c.bold + c.white, 'Installing cloudflared for tunnel support...')}`);

  ensureInstallDir();
  const cfName = getCloudflaredName();
  const dest = cfPath;
  const url = getReleaseUrl(CLOUDFLARED_REPO, cfName);

  const sp = spinner(`Downloading ${cfName}`);
  try {
    await download(url, dest);
    makeExecutable(dest);
    sp.succeed(`Downloaded ${colorize(c.green, cfName)}`);
    return true;
  } catch (err) {
    sp.fail(`Failed to download cloudflared`);
    console.log(`  ${icon.info} ${colorize(c.dim, 'Install manually: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/')}`);
    return false;
  }
}

// ── Check Cloudflared Available ─────────────────────────────────────────
function hasCloudflared() {
  const cfPath = getCloudflaredPath();
  return fs.existsSync(cfPath);
}

// ── Main Install ────────────────────────────────────────────────────────
async function install() {
  const binaryOk = await installBinary();
  if (!binaryOk) {
    console.log(`\n  ${icon.fail} ${colorize(c.red + c.bold, 'Could not install KeyParty binary.')}`);
    console.log(`  ${colorize(c.dim, 'Install Go and run: go build -o keyparty .')}`);
    throw new Error('Could not install KeyParty binary');
  }
}

module.exports = { install, installBinary, installCloudflared, hasCloudflared };
