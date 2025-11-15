import test from 'node:test'
import assert from 'node:assert/strict'

import { runPrechecks } from '../../tools/cli/src/lib/publish/precheck'

test('runPrechecks rejects when stable notes missing', async () => {
  const manifest = { id: 'demo', version: '1.2.3', permissions: [] }
  await assert.rejects(
    () =>
      runPrechecks({
        manifest,
        channel: 'stable',
        notes: '',
      }),
    /stable releases must include release notes/i,
  )
})

test('runPrechecks passes for beta without notes', async () => {
  const manifest = { id: 'demo', version: '1.2.3', permissions: [] }
  await assert.doesNotReject(() =>
    runPrechecks({
      manifest,
      channel: 'beta',
      notes: '',
    }),
  )
})
