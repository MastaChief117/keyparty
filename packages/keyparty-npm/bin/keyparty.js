#!/usr/bin/env node

const { install } = require('../lib/install');
const { setup } = require('../lib/setup');
const { isInstalled } = require('../lib/platform');
const { c, colorize, icon, banner } = require('../lib/tui');

// ── CLI Flags ───────────────────────────────────────────────────────────
const args = process.argv.slice(2);

if (args.includes('--help') || args.includes('-h')) {
  console.log(banner());
  console.log(`
  ${colorize(c.bold + c.white, 'Usage:')}
    keyparty                 ${colorize(c.dim, 'Start interactive setup wizard')}
    keyparty --port ${colorize(c.dim, '8080')}         ${colorize(c.dim, 'Start on custom port')}
    keyparty --password ${colorize(c.dim, 'xxx')}     ${colorize(c.dim, 'Set admin password')}
    keyparty --tunnel        ${colorize(c.dim, 'Enable cloudflare tunnel')}
    keyparty --version       ${colorize(c.dim, 'Show version')}
    keyparty --help          ${colorize(c.dim, 'Show this help')}
  `);
  console.log(`  ${colorize(c.dim, 'Docs:')} https://mastachief117.github.io/keyparty/`);
  console.log(`  ${colorize(c.dim, 'GitHub:')} https://github.com/MastaChief117/keyparty`);
  console.log();
  process.exit(0);
}

if (args.includes('--version') || args.includes('-v')) {
  const pkg = require('../package.json');
  console.log(`keyparty v${pkg.version}`);
  process.exit(0);
}

// ── Main ────────────────────────────────────────────────────────────────
async function main() {
  console.log(banner());

  // Check if binary is installed
  if (!isInstalled()) {
    console.log(`  ${icon.info} ${colorize(c.dim, 'First run detected. Installing KeyParty binary...')}`);
    await install();
    console.log();
  }

  // If flags provided, skip wizard and go straight to start
  if (args.includes('--port') || args.includes('--password') || args.includes('--tunnel')) {
    const { getBinaryPath } = require('../lib/platform');
    const { spawn } = require('child_process');

    const portIdx = args.indexOf('--port');
    const passIdx = args.indexOf('--password');
    const port = portIdx !== -1 ? args[portIdx + 1] : '8080';
    const pass = passIdx !== -1 ? args[passIdx + 1] : '';

    const bin = getBinaryPath();
    const binArgs = ['-port', port];
    if (pass) binArgs.push('-admin-pass', pass);

    const proc = spawn(bin, binArgs, {
      stdio: 'inherit',
      env: { ...process.env, ADMIN_PASSWORD: pass },
    });

    proc.on('exit', (code) => process.exit(code));
    return;
  }

  // Interactive setup
  await setup();
}

main().catch((err) => {
  console.error(`\n  ${icon.fail} ${colorize(c.red, err.message)}`);
  process.exit(1);
});
