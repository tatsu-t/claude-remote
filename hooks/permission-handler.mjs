#!/usr/bin/env node
/**
 * PermissionRequest hook for claude-remote plugin.
 * Auto-approves only the specific commands needed by /install:
 *   - go version / go install github.com/tatsu-t/claude-remote
 *   - claude-remote (any subcommand)
 *   - install.sh inside the plugin directory
 * All other Bash requests are deferred to the default handler.
 */

// Shell metacharacters that must never appear in approved commands.
const SHELL_META = /[;&|`$><\n(){}\\'"]/;

// Strictly anchored allowlist. Each pattern must match the ENTIRE command.
const ALLOWED = [
  // go version
  /^go version$/,
  // go install of this specific module (optional @version tag)
  /^go install github\.com\/tatsu-t\/claude-remote\/cmd\/claude-remote(@[\w.\-/]+)?$/,
  // claude-remote with a finite set of known safe subcommands / flags
  /^claude-remote (upload|attach|pull|config|--version|--help)([ \t][\w.\-=/:@]+)*$/,
  // install.sh inside the plugin directory (absolute path, no traversal)
  /^[^\0]*claude-remote[/\\]install\.sh$/,
];

async function readStdin() {
  return new Promise((resolve) => {
    let buf = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => { buf += chunk; });
    process.stdin.on('end', () => resolve(buf));
    // Timeout guard: don't block Claude Code if stdin stalls.
    setTimeout(() => resolve(buf || '{}'), 3000);
  });
}

async function main() {
  const raw = await readStdin();
  try {
    const data = JSON.parse(raw);
    const cmd = (data?.tool_input?.command ?? '').trim();
    // Reject any command containing shell metacharacters before allowlist check.
    if (SHELL_META.test(cmd)) {
      process.stdout.write(JSON.stringify({}) + '\n');
      return;
    }
    const allowed = ALLOWED.some((re) => re.test(cmd));
    if (allowed) {
      process.stdout.write(JSON.stringify({ behavior: 'allow' }) + '\n');
    } else {
      // Defer — let Claude Code's own permission logic decide.
      process.stdout.write(JSON.stringify({}) + '\n');
    }
  } catch {
    // Parse error: defer rather than block.
    process.stdout.write(JSON.stringify({}) + '\n');
  }
}

main();
