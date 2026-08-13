const { stdin, stdout } = process;
const readline = require('readline');

// ── Colors (zero deps, ANSI escape codes) ───────────────────────────────
const c = {
  reset: '\x1b[0m',
  bold: '\x1b[1m',
  dim: '\x1b[2m',
  italic: '\x1b[3m',
  underline: '\x1b[4m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',
  gray: '\x1b[90m',
  bgRed: '\x1b[41m',
  bgGreen: '\x1b[42m',
  bgBlue: '\x1b[44m',
  bgMagenta: '\x1b[45m',
  bgCyan: '\x1b[46m',
};

function colorize(color, text) {
  return `${color}${text}${c.reset}`;
}

// ── Box Drawing ─────────────────────────────────────────────────────────
function box(lines, opts = {}) {
  const { title, width = 56, border = c.cyan, titleColor = c.bold + c.cyan } = opts;
  const top = `${border}╔${'═'.repeat(width - 2)}╗${c.reset}`;
  const bot = `${border}╚${'═'.repeat(width - 2)}╝${c.reset}`;
  const empty = `${border}║${' '.repeat(width - 2)}║${c.reset}`;

  const out = [top];
  if (title) {
    const pad = width - 2 - visibleWidth(title);
    const lp = Math.floor(pad / 2);
    const rp = pad - lp;
    out.push(`${border}║${' '.repeat(lp)}${titleColor}${title}${c.reset}${' '.repeat(rp)}${border}║${c.reset}`);
    out.push(`${border}╠${'═'.repeat(width - 2)}╣${c.reset}`);
  }
  for (const line of lines) {
    const vwidth = visibleWidth(line);
    const padding = width - 3 - vwidth;
    if (padding > 0) {
      out.push(`${border}║ ${line}${' '.repeat(padding)}${border}║${c.reset}`);
    } else {
      out.push(`${border}║ ${line.substring(0, width - 5)} ${border}║${c.reset}`);
    }
  }
  out.push(bot);
  return out.join('\n');
}

function stripAnsi(str) {
  return str.replace(/\x1b\[[0-9;]*m/g, '');
}

function visibleWidth(str) {
  const stripped = stripAnsi(str);
  let width = 0;
  for (const char of stripped) {
    const code = char.codePointAt(0);
    // Double-width: CJK, emoji, symbols, arrows, etc.
    if (
      code > 0x1f000 ||          // Emoji
      (code >= 0x20000 && code <= 0x2ffff) || // CJK Extension B+
      (code >= 0x30000 && code <= 0x3ffff) ||
      (code >= 0xf900 && code <= 0xfaff) ||   // CJK Compatibility Ideographs
      (code >= 0xfe30 && code <= 0xfe4f) ||   // CJK Compatibility Forms
      (code >= 0xff00 && code <= 0xffef) ||   // Fullwidth Forms
      (code >= 0x20000 && code <= 0x2a6df) || // CJK Unified Extension B
      (code >= 0x2a700 && code <= 0x2b73f) || // CJK Unified Extension C
      (code >= 0x2b740 && code <= 0x2b81f) || // CJK Unified Extension D
      (code >= 0x2b820 && code <= 0x2ceaf) || // CJK Unified Extension E
      (code >= 0x2ceb0 && code <= 0x2ebef) || // CJK Unified Extension F
      (code >= 0x30000 && code <= 0x3134f) || // CJK Unified Extension G
      (code >= 0x1f300 && code <= 0x1f9ff) || // Misc Symbols and Pictographs
      (code >= 0x1fa00 && code <= 0x1fa6f) || // Chess Symbols
      (code >= 0x1fa70 && code <= 0x1faff) || // Symbols and Pictographs Extended-A
      (code >= 0x20000 && code <= 0x2fa1f)    // CJK Compatibility Supplement
    ) {
      width += 2;
    } else {
      width += 1;
    }
  }
  return width;
}

// ── Banner ──────────────────────────────────────────────────────────────
function banner() {
  const lines = [
    '',
    colorize(c.bold + c.magenta, '  ██╗  ██╗██╗████████╗██████╗  ██████╗ ███████╗'),
    colorize(c.bold + c.magenta, '  ██║ ██╔╝██║╚══██╔══╝██╔══██╗██╔═══██╗██╔════╝'),
    colorize(c.bold + c.cyan,   '  █████╔╝ ██║   ██║   ██████╔╝██║   ██║███████╗'),
    colorize(c.bold + c.cyan,   '  ██╔═██╗ ██║   ██║   ██╔══██╗██║   ██║╚════██║'),
    colorize(c.bold + c.blue,   '  ██║  ██╗██║   ██║   ██████╔╝╚██████╔╝███████║'),
    colorize(c.bold + c.blue,   '  ╚═╝  ╚═╝╚═╝   ╚═╝   ╚═════╝  ╚═════╝ ╚══════╝'),
    '',
    colorize(c.dim + c.white,  '       One endpoint. Ten providers. Zero budget.'),
    '',
  ];
  return lines.join('\n');
}

// ── Status Icons ────────────────────────────────────────────────────────
const icon = {
  ok: colorize(c.green, '✔'),
  fail: colorize(c.red, '✘'),
  warn: colorize(c.yellow, '⚠'),
  info: colorize(c.cyan, 'ℹ'),
  dot: colorize(c.magenta, '●'),
  arrow: colorize(c.cyan, '→'),
  spark: colorize(c.yellow, '✦'),
  key: colorize(c.cyan, '🔐'),
  rocket: colorize(c.green, '🚀'),
  globe: colorize(c.blue, '🌐'),
  lock: colorize(c.yellow, '🔒'),
  fire: colorize(c.red, '🔥'),
  star: colorize(c.yellow, '★'),
};

// ── Spinner ─────────────────────────────────────────────────────────────
const spinners = {
  dots: { frames: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'], interval: 80 },
  moon: { frames: ['🌑', '🌒', '🌓', '🌔', '🌕', '🌖', '🌗', 'loon'], interval: 100 },
  fire: { frames: ['🔥', '🔥', '🔥'], interval: 150 },
};

function spinner(text, opts = {}) {
  const frames = opts.frames || spinners.dots.frames;
  const interval = opts.interval || spinners.dots.interval;
  let i = 0;
  let running = true;

  const id = setInterval(() => {
    if (!running) return;
    stdout.write(`\r  ${frames[i % frames.length]} ${text}   `);
    i++;
  }, interval);

  return {
    stop(suffix = '') {
      running = false;
      clearInterval(id);
      stdout.write(`\r${' '.repeat(text.length + 10)}\r`);
      if (suffix) stdout.write(`  ${suffix}\n`);
    },
    succeed(text) { this.stop(`${icon.ok} ${colorize(c.green, text)}`); },
    fail(text) { this.stop(`${icon.fail} ${colorize(c.red, text)}`); },
    warn(text) { this.stop(`${icon.warn} ${colorize(c.yellow, text)}`); },
  };
}

// ── Progress Bar ────────────────────────────────────────────────────────
function progressBar(pct, opts = {}) {
  const width = opts.width || 30;
  const filled = Math.round(width * pct / 100);
  const empty = width - filled;
  const bar = colorize(c.cyan, '█'.repeat(filled)) + colorize(c.gray, '░'.repeat(empty));
  const pctStr = colorize(c.bold + c.white, `${Math.round(pct)}%`);
  return `[${bar}] ${pctStr}`;
}

// ── Interactive Input ───────────────────────────────────────────────────
function createInterface() {
  return readline.createInterface({ input: stdin, output: stdout });
}

function ask(question, opts = {}) {
  return new Promise((resolve) => {
    const rl = createInterface();
    const prefix = opts.prefix || `  ${icon.arrow} `;
    const suffix = opts.suffix || '';
    rl.question(`${prefix}${colorize(c.bold + c.white, question)}${suffix} `, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

async function confirm(question, defaultVal = true) {
  const hint = defaultVal ? colorize(c.dim, '(Y/n)') : colorize(c.dim, '(y/N)');
  const answer = await ask(`${question} ${hint}`, { suffix: '' });
  if (!answer) return defaultVal;
  return answer.toLowerCase() === 'y' || answer.toLowerCase() === 'yes';
}

async function password(question) {
  return new Promise((resolve) => {
    const rl = createInterface();
    const prefix = `  ${icon.key} `;
    rl.question(`${prefix}${colorize(c.bold + c.white, question)} `, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

function select(question, options) {
  return new Promise((resolve) => {
    const rl = createInterface();
    console.log(`\n  ${colorize(c.bold + c.white, question)}`);
    options.forEach((opt, i) => {
      console.log(`    ${colorize(c.cyan, String(i + 1))}) ${opt.label}`);
    });
    console.log();
    rl.question(`  ${icon.arrow} ${colorize(c.dim, 'Pick [1-' + options.length + ']')}: `, (answer) => {
      rl.close();
      const idx = parseInt(answer) - 1;
      if (idx >= 0 && idx < options.length) {
        resolve(options[idx]);
      } else {
        resolve(options[0]);
      }
    });
  });
}

// ── Section Divider ─────────────────────────────────────────────────────
function divider(label) {
  const pad = 54 - label.length;
  const left = Math.floor(pad / 2);
  const right = pad - left;
  return `\n${c.dim}${'─'.repeat(left)}${c.reset} ${colorize(c.bold + c.white, label)} ${c.dim}${'─'.repeat(right)}${c.reset}\n`;
}

// ── Summary Table ───────────────────────────────────────────────────────
function table(rows, opts = {}) {
  const colWidths = opts.colWidths || [20, 30];
  const lines = [];
  for (const row of rows) {
    let line = '  ';
    for (let i = 0; i < row.length; i++) {
      const cell = String(row[i]);
      const width = colWidths[i] || 20;
      const stripped = stripAnsi(cell);
      const padding = width - stripped.length;
      line += cell + ' '.repeat(Math.max(0, padding)) + '  ';
    }
    lines.push(line);
  }
  return lines.join('\n');
}

// ── Clear Screen ────────────────────────────────────────────────────────
function clear() {
  stdout.write('\x1b[2J\x1b[H');
}

module.exports = {
  c, colorize, stripAnsi, visibleWidth, box, banner, icon, spinner, progressBar,
  ask, confirm, password, select, divider, table, clear, createInterface,
};
