const { spawn } = require('child_process');
const fs = require('fs');
const { spinner } = require('./tui');
const { getCloudflaredPath } = require('./platform');

// ── Start Cloudflared Tunnel ────────────────────────────────────────────
function startTunnel(port = 8080) {
  return new Promise((resolve, reject) => {
    const cfPath = getCloudflaredPath();
    if (!fs.existsSync(cfPath)) {
      return reject(new Error('cloudflared not installed'));
    }

    const sp = spinner('Starting cloudflared tunnel', { frames: ['🌐', '🌍', '🌎', '🌏'] });

    const proc = spawn(cfPath, ['tunnel', '--url', `http://localhost:${port}`], {
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    let resolved = false;
    const timeout = setTimeout(() => {
      if (!resolved) {
        proc.kill();
        sp.fail('Tunnel timed out (30s)');
        reject(new Error('Tunnel timeout'));
      }
    }, 30000);

    let buffer = '';
    proc.stdout.on('data', (data) => {
      buffer += data.toString();
      const match = buffer.match(/https:\/\/[a-z0-9-]+\.trycloudflare\.com/);
      if (match && !resolved) {
        resolved = true;
        clearTimeout(timeout);
        const url = match[0];
        sp.succeed(`Tunnel active!`);
        resolve({ url, process: proc });
      }
    });

    let stderrBuffer = '';
    proc.stderr.on('data', (data) => {
      stderrBuffer += data.toString();
      const match = stderrBuffer.match(/https:\/\/[a-z0-9-]+\.trycloudflare\.com/);
      if (match && !resolved) {
        resolved = true;
        clearTimeout(timeout);
        const url = match[0];
        sp.succeed(`Tunnel active!`);
        resolve({ url, process: proc });
      }
    });

    proc.on('error', (err) => {
      if (!resolved) {
        clearTimeout(timeout);
        sp.fail('Failed to start tunnel');
        reject(err);
      }
    });

    proc.on('exit', (code) => {
      if (!resolved) {
        clearTimeout(timeout);
        sp.fail(`Tunnel exited with code ${code}`);
        reject(new Error(`cloudflared exited: ${code}`));
      }
    });
  });
}

// ── Stop Tunnel ─────────────────────────────────────────────────────────
function stopTunnel(proc) {
  if (proc && !proc.killed && proc.exitCode === null) {
    proc.kill('SIGTERM');
  }
}

module.exports = { startTunnel, stopTunnel };
