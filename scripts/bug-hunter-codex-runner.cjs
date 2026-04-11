#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, 'utf8');
}

function buildSchema() {
  return {
    type: 'object',
    properties: {
      status: {
        type: 'string',
        enum: ['completed', 'blocked', 'failed']
      },
      domain: { type: 'string' },
      summary: { type: 'string' },
      confirmedBugCount: { type: 'integer', minimum: 0 },
      artifactsUpdated: { type: 'boolean' },
      testsRun: {
        type: 'array',
        items: { type: 'string' }
      },
      notes: {
        type: 'array',
        items: { type: 'string' }
      }
    },
    required: [
      'status',
      'domain',
      'summary',
      'confirmedBugCount',
      'artifactsUpdated',
      'testsRun',
      'notes'
    ],
    additionalProperties: false
  };
}

function ensureCodexOutput(outputPath) {
  if (!fs.existsSync(outputPath)) {
    throw new Error(`Codex did not write output file: ${outputPath}`);
  }
  const raw = fs.readFileSync(outputPath, 'utf8').trim();
  if (!raw) {
    throw new Error(`Codex wrote an empty output file: ${outputPath}`);
  }
  return JSON.parse(raw);
}

function runCodex(args, payload, stdoutPath, stderrPath) {
  return new Promise((resolve, reject) => {
    const child = spawn('codex', args, {
      cwd: payload.target,
      stdio: ['pipe', 'pipe', 'pipe']
    });
    const stdoutStream = fs.createWriteStream(stdoutPath, { encoding: 'utf8' });
    const stderrStream = fs.createWriteStream(stderrPath, { encoding: 'utf8' });
    let stdout = '';
    let stderr = '';
    let settled = false;
    let timedOut = false;

    const timeout = setTimeout(() => {
      timedOut = true;
      child.kill('SIGTERM');
      setTimeout(() => {
        if (!settled) {
          child.kill('SIGKILL');
        }
      }, 5000).unref();
    }, 15 * 60 * 1000);

    function finish(callback) {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timeout);
      stdoutStream.end();
      stderrStream.end();
      callback();
    }

    child.stdout.on('data', (chunk) => {
      const text = chunk.toString('utf8');
      stdout += text;
      stdoutStream.write(text);
    });

    child.stderr.on('data', (chunk) => {
      const text = chunk.toString('utf8');
      stderr += text;
      stderrStream.write(text);
    });

    child.on('error', (error) => {
      finish(() => reject(error));
    });

    child.on('close', (code, signal) => {
      finish(() => resolve({
        code,
        signal,
        stdout,
        stderr,
        timedOut
      }));
    });

    child.stdin.end(payload.prompt);
  });
}

async function main() {
  const payloadPath = process.argv[2];
  if (!payloadPath) {
    throw new Error('Usage: bug-hunter-codex-runner.cjs <payload-path>');
  }

  const payload = readJson(path.resolve(payloadPath));
  const runnerDir = path.join(payload.target, '.bug-hunter', 'sdk-loop', 'runner');
  const schemaPath = path.join(runnerDir, 'output-schema.json');
  const outputPath = path.join(runnerDir, `${payload.domain.slug || 'domain'}-response.json`);
  const stdoutPath = path.join(runnerDir, `${payload.domain.slug || 'domain'}-codex.stdout.log`);
  const stderrPath = path.join(runnerDir, `${payload.domain.slug || 'domain'}-codex.stderr.log`);

  writeJson(schemaPath, buildSchema());
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });

  const args = ['exec'];
  if (payload.options && payload.options.model) {
    args.push('--model', payload.options.model);
  }
  args.push(
    '-C', payload.target,
    '--ephemeral',
    '--sandbox', 'workspace-write',
    '--skip-git-repo-check',
    '--color', 'never',
    '--output-schema', schemaPath,
    '--output-last-message', outputPath,
    '-'
  );

  const result = await runCodex(args, payload, stdoutPath, stderrPath);
  if (result.code !== 0) {
    const stderrTail = (result.stderr || '').trim().split('\n').slice(-20).join('\n');
    throw new Error([
      result.timedOut
        ? 'codex exec timed out after 15 minutes'
        : `codex exec exited with status ${result.code}${result.signal ? ` (signal: ${result.signal})` : ''}`,
      `stdout log: ${stdoutPath}`,
      `stderr log: ${stderrPath}`,
      stderrTail ? `stderr tail:\n${stderrTail}` : null
    ].filter(Boolean).join('\n'));
  }

  const response = ensureCodexOutput(outputPath);
  process.stdout.write(JSON.stringify({
    threadId: payload.threadId || `ephemeral-${payload.domain.slug || 'domain'}`,
    response,
    usage: null
  }));
}

main().catch((error) => {
  console.error(error.stack || error.message);
  process.exit(1);
});
