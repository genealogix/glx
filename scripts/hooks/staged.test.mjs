import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync, symlinkSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const hooks = resolve(import.meta.dirname)
const dependencies = resolve(hooks, '../../website/node_modules')

function fixture(t) {
  const dir = mkdtempSync(join(tmpdir(), 'glx-staged-test-'))
  t.after(() => rmSync(dir, { recursive: true, force: true }))
  function run(cmd, args, options = {}) {
    return spawnSync(cmd, args, { cwd: dir, encoding: 'utf8', ...options })
  }
  function git(...args) {
    const result = run('git', args)
    assert.equal(result.status, 0, result.stderr)
    return result.stdout.trim()
  }
  function put(file, text) {
    mkdirSync(resolve(dir, file, '..'), { recursive: true })
    writeFileSync(join(dir, file), text)
  }
  git('init', '-q', '-b', 'main')
  git('config', 'user.name', 'Hook Test')
  git('config', 'user.email', 'hook-test@example.invalid')
  return { dir, run, git, put }
}

function goFixture(t) {
  const f = fixture(t)
  f.put('.golangci-lint-version', 'v2.14.0\n')
  f.put('go.mod', 'module example.invalid/hook\n\ngo 1.27\n')
  f.put('pkg/keep.go', 'package pkg\n')
  f.put('pkg/remove.go', 'package pkg\n')
  f.git('add', '.')
  f.git('commit', '-qm', 'baseline')
  f.git('update-ref', 'refs/remotes/origin/main', 'HEAD')
  f.git('checkout', '-qb', 'feature')
  f.put('bin/golangci-lint', `#!/bin/sh
if [ "$1" = version ]; then echo 2.14.0; exit; fi
printf '%s\\n' "$@" > "$HOOK_ARGS"
cat pkg/keep.go > "$HOOK_SOURCE"
`)
  f.run('chmod', ['+x', 'bin/golangci-lint'])
  function hook() {
    return f.run('bash', [join(hooks, 'golangci-lint-staged.sh')], {
      env: { ...process.env, PATH: `${f.dir}/bin:${process.env.PATH}`, HOOK_ARGS: `${f.dir}/args`, HOOK_SOURCE: `${f.dir}/source` }
    })
  }
  return { ...f, hook }
}

test('Go deletion still lints the surviving package', t => {
  const f = goFixture(t)
  f.git('rm', '-q', 'pkg/remove.go')
  const result = f.hook()
  assert.equal(result.status, 0, result.stderr)
  assert.match(readFileSync(join(f.dir, 'args'), 'utf8'), /\.\/pkg/)
})

test('Go snapshot uses staged content and staged version pin', t => {
  const f = goFixture(t)
  f.put('pkg/keep.go', 'package pkg\n// staged\n')
  f.git('add', 'pkg/keep.go')
  f.put('pkg/keep.go', 'invalid unstaged source\n')
  f.put('.golangci-lint-version', 'v0.0.0\n')
  const result = f.hook()
  assert.equal(result.status, 0, result.stderr)
  assert.equal(readFileSync(join(f.dir, 'source'), 'utf8'), 'package pkg\n// staged\n')
  assert.equal(existsSync(join(f.dir, '.git/glx-lint-snapshot')), false)
})

test('Go findings committed with hooks skipped remain in the branch diff', t => {
  const f = goFixture(t)
  f.put('pkg/keep.go', 'package pkg\n// branch change\n')
  f.git('add', 'pkg/keep.go')
  f.git('commit', '-qm', 'branch change')
  const result = f.hook()
  assert.equal(result.status, 0, result.stderr)
  assert.match(readFileSync(join(f.dir, 'args'), 'utf8'), /--new-from-patch=/)
  assert.match(readFileSync(join(f.dir, 'args'), 'utf8'), /\.\/pkg/)
})

test('ESLint checks staged configuration and sources', { skip: !existsSync(join(dependencies, '.bin/eslint')) }, t => {
  const f = fixture(t)
  f.put('website/eslint.config.mjs', "export default [{ rules: { 'no-unused-vars': 'error' } }]\n")
  f.put('website/space name.js', 'const unused = 1\n')
  f.git('add', 'website')
  symlinkSync(dependencies, join(f.dir, 'website/node_modules'), 'dir')
  f.put('website/eslint.config.mjs', 'export default [{}]\n')
  f.put('website/space name.js', '// valid unstaged source\n')
  let result = f.run('bash', [join(hooks, 'eslint-staged.sh'), 'space name.js'], { cwd: join(f.dir, 'website') })
  assert.equal(result.status, 1, result.stdout + result.stderr)
  assert.match(result.stdout, /no-unused-vars/)
  f.put('website/space name.js', '// valid staged source\n')
  f.git('add', 'website/space name.js')
  f.put('website/space name.js', 'const unused = 1\n')
  result = f.run('bash', [join(hooks, 'eslint-staged.sh'), 'space name.js'], { cwd: join(f.dir, 'website') })
  assert.equal(result.status, 0, result.stdout + result.stderr)
})


test('ESLint still reads LF blobs when checkout-index converts to CRLF', { skip: !existsSync(join(dependencies, '.bin/eslint')) }, t => {
  const f = fixture(t)
  f.put('website/eslint.config.mjs', "export default [{ rules: { 'linebreak-style': ['error', 'unix'] } }]\n")
  f.put('website/test.js', '// clean LF source\n')
  f.git('add', 'website')
  f.git('config', 'core.autocrlf', 'true')
  symlinkSync(dependencies, join(f.dir, 'website/node_modules'), 'dir')
  const result = f.run('bash', [join(hooks, 'eslint-staged.sh'), 'test.js'], { cwd: join(f.dir, 'website') })
  assert.equal(result.status, 0, result.stdout + result.stderr)
})
