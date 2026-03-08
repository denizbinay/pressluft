import test from 'node:test'
import assert from 'node:assert/strict'
import { createJiti } from 'jiti'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const testDir = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(testDir, '..')
const jiti = createJiti(import.meta.url, {
  alias: {
    '~': resolve(webRoot, 'app'),
    '@': resolve(webRoot, 'app'),
  },
})
const runtime = await jiti.import('../app/lib/api-runtime.ts')

test('parseJob rejects invalid job kind', () => {
  assert.throws(
    () =>
      runtime.parseJob({
        id: 1,
        status: 'queued',
        kind: 'unknown_kind',
        current_step: '',
        retry_count: 0,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }),
    /Invalid job response/,
  )
})

test('parseAuthActor accepts valid actor', () => {
  const actor = runtime.parseAuthActor({
    id: '1',
    type: 'operator',
    email: 'admin@example.com',
    role: 'admin',
    authenticated: true,
    capabilities: ['manage_servers'],
  })

  assert.equal(actor.email, 'admin@example.com')
  assert.equal(actor.role, 'admin')
})

test('parseServerCatalogResponse rejects malformed server catalog payload', () => {
  assert.throws(
    () =>
      runtime.parseServerCatalogResponse({
        catalog: { locations: [], server_types: [] },
        profiles: [{ key: 'nginx-stack' }],
      }),
    /Invalid server catalog response/,
  )
})
