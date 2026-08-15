const { spawn, execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const readline = require('readline');
const { c, colorize, box, banner, icon, ask, confirm, password, select, divider, table, clear, spinner } = require('./tui');
const { getBinaryPath, INSTALL_DIR } = require('./platform');
const { installCloudflared, hasCloudflared } = require('./install');
const { startTunnel, stopTunnel } = require('./tunnel');

const CONFIG_FILE = path.join(process.env.HOME || process.env.USERPROFILE || '.', '.keyparty', 'gateway.env');

// ── Load/Save Config ────────────────────────────────────────────────────
function loadConfig() {
  if (!fs.existsSync(CONFIG_FILE)) return {};
  const data = fs.readFileSync(CONFIG_FILE, 'utf8');
  const config = {};
  for (const line of data.split('\n')) {
    const match = line.match(/^([^#=]+)=(.*)$/);
    if (match) config[match[1].trim()] = match[2].trim();
  }
  return config;
}

function saveConfig(config) {
  const dir = path.dirname(CONFIG_FILE);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
  const lines = Object.entries(config).map(([k, v]) => `${k}=${v}`).join('\n');
  fs.writeFileSync(CONFIG_FILE, lines + '\n');
}

// ── Welcome Screen ──────────────────────────────────────────────────────
function showWelcome() {
  clear();
  console.log(banner());

  const lines = [
    `${icon.rocket} ${colorize(c.bold + c.green, 'Welcome to KeyParty!')}`,
    '',
    `${icon.arrow} ${colorize(c.white, 'One endpoint. Ten providers. Zero budget.')}`,
    '',
    `${icon.info} ${colorize(c.dim, 'This wizard sets up your gateway in 60 seconds.')}`,
    `${icon.info} ${colorize(c.dim, 'You need one API key (Groq has a free tier).')}`,
  ];

  console.log(box(lines, { title: 'Setup Wizard', width: 60 }));
}

// ── Step 1: Admin Password ──────────────────────────────────────────────
async function setupPassword() {
  console.log(divider('Step 1: Admin Password'));

  const config = loadConfig();
  if (config.ADMIN_PASSWORD) {
    console.log(`  ${icon.ok} ${colorize(c.dim, 'Password already set.')}`);
    const change = await confirm('Change it?', false);
    if (!change) return config.ADMIN_PASSWORD;
  }

  console.log(`  ${colorize(c.dim, 'Protects the dashboard and admin API.')}`);
  console.log(`  ${colorize(c.dim, 'Leave empty to disable auth (not recommended).')}\n`);

  const pass = await password('Set admin password:');
  if (pass) {
    console.log(`  ${icon.ok} ${colorize(c.green, 'Password set.')}`);
  } else {
    console.log(`  ${icon.warn} ${colorize(c.yellow, 'Admin auth disabled.')}`);
  }
  return pass;
}

// ── Step 2: Cloudflared ─────────────────────────────────────────────────
async function setupCloudflared() {
  console.log(divider('Step 2: Cloudflare Tunnel'));

  const has = hasCloudflared();
  if (has) {
    console.log(`  ${icon.ok} ${colorize(c.dim, 'Cloudflared already installed.')}`);
    return true;
  }

  console.log(`  ${colorize(c.dim, 'A tunnel hides your real IP behind a *.trycloudflare.com URL.')}`);
  console.log(`  ${colorize(c.dim, 'Free. No account needed.')}\n`);

  const install = await confirm('Install cloudflared?', true);
  if (install) {
    return await installCloudflared();
  }
  return false;
}

// ── Step 3: Port ────────────────────────────────────────────────────────
async function setupPort() {
  console.log(divider('Step 3: Port'));

  const config = loadConfig();
  const defaultPort = config.PORT || '8080';

  const port = await ask(`Port [${defaultPort}]:`);
  return port || defaultPort;
}

// ── Step 4: Start Gateway ───────────────────────────────────────────────
function startGateway(port, password) {
  const bin = getBinaryPath();
  const args = ['-port', port];
  if (password) args.push('-admin-pass', password);

  const proc = spawn(bin, args, {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, ADMIN_PASSWORD: password || '' },
  });

  return proc;
}

// ── Step 5: Tunnel ──────────────────────────────────────────────────────
async function setupTunnel(port) {
  console.log(divider('Step 5: Tunnel'));

  if (!hasCloudflared()) {
    console.log(`  ${icon.info} ${colorize(c.dim, 'Skipping tunnel (cloudflared not available).')}`);
    return null;
  }

  const enable = await confirm('Start a Cloudflare tunnel?', false);
  if (!enable) return null;

  try {
    const tunnel = await startTunnel(port);
    return tunnel;
  } catch (err) {
    console.log(`  ${icon.warn} ${colorize(c.yellow, 'Tunnel failed: ' + err.message)}`);
    return null;
  }
}

// ── Final Summary ───────────────────────────────────────────────────────
function showSummary(port, password, tunnel) {
  console.log('\n' + '═'.repeat(78));

  const lines = [
    '',
    `  ${colorize(c.bold + c.green, 'KeyParty is running!')}`,
    '',
    `  Dashboard:  http://localhost:${port}`,
    `  Proxy:      http://localhost:${port}/v1/chat/completions`,
    `  Health:     http://localhost:${port}/health`,
  ];

  if (tunnel) {
    lines.push('');
    lines.push(`  ${colorize(c.bold + c.yellow, 'TUNNEL ACTIVE')}`);
    lines.push(`  Public:     ${colorize(c.green, tunnel.url)}`);
    lines.push(`  Proxy:      ${colorize(c.cyan, tunnel.url + '/v1/chat/completions')}`);
  }

  if (password) {
    lines.push('');
    lines.push(`  Admin auth: enabled`);
  }

  lines.push('');
  lines.push(`  ${colorize(c.dim, 'Press Ctrl+C to stop.')}`);
  lines.push('');

  console.log(box(lines, { title: 'Status', width: 78, border: c.green, titleColor: c.bold + c.green }));
}

// ── Main Setup Flow ─────────────────────────────────────────────────────
async function setup() {
  showWelcome();

  const password = await setupPassword();
  const hasCF = await setupCloudflared();
  const port = await setupPort();

  // Save config
  const config = loadConfig();
  if (password) config.ADMIN_PASSWORD = password;
  config.PORT = port;
  saveConfig(config);

  console.log(divider('Starting Gateway'));

  const sp = spinner('Starting KeyParty on port ' + port);
  const gwProc = startGateway(port, password);

  // Wait for gateway to start
  await new Promise((resolve) => setTimeout(resolve, 1500));

  if (gwProc.killed || gwProc.exitCode !== null) {
    sp.fail('Gateway failed to start');
    console.log(`  ${icon.info} ${colorize(c.dim, 'Check if port ' + port + ' is already in use.')}`);
    process.exit(1);
  }
  sp.succeed(`Gateway running (PID: ${gwProc.pid})`);

  const tunnel = await setupTunnel(port);

  showSummary(port, password, tunnel);

  // Handle shutdown
  const shutdown = () => {
    console.log(`\n  ${icon.arrow} ${colorize(c.dim, 'Shutting down...')}`);
    if (tunnel) stopTunnel(tunnel.process);
    gwProc.kill('SIGTERM');
    process.exit(0);
  };

  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);

  // Keep alive
  gwProc.on('exit', (code) => {
    console.log(`\n  ${icon.warn} ${colorize(c.yellow, 'Gateway exited with code ' + code)}`);
    if (tunnel) stopTunnel(tunnel.process);
    process.exit(code);
  });
}

module.exports = { setup };
