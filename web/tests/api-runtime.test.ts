import { describe, it, expect } from 'vitest'
import {
  parseAuthActor,
  parseJob,
  parseServerCatalogResponse,
  parseStoredServer,
  parseStoredServers,
  parseStoredSite,
  parseStoredDomain,
  parseDeleteSiteResponse,
  parseDeleteDomainResponse,
  parseDeleteServerResponse,
  parseCreateServerResponse,
  parseAgentInfo,
  parseAgentStatusMapResponse,
  parseJobEvents,
  parseActivity,
  parseActivityListResponse,
  parseUnreadCountResponse,
  parseServicesResponse,
  parseSiteHealthResponse,
  parseHealthResponse,
} from '~/lib/api-runtime'

describe('parseAuthActor', () => {
  it('accepts a valid actor', () => {
    const actor = parseAuthActor({
      id: '1',
      type: 'operator',
      email: 'admin@example.com',
      role: 'admin',
      authenticated: true,
      capabilities: ['manage_servers'],
    })

    expect(actor.email).toBe('admin@example.com')
    expect(actor.role).toBe('admin')
    expect(actor.authenticated).toBe(true)
  })

  it('rejects actor with missing required fields', () => {
    expect(() =>
      parseAuthActor({ id: '1', type: 'operator' }),
    ).toThrow(/Invalid auth actor response/)
  })

  it('rejects actor with invalid role', () => {
    expect(() =>
      parseAuthActor({
        id: '1',
        type: 'operator',
        email: 'admin@example.com',
        role: 'superadmin',
        authenticated: true,
      }),
    ).toThrow(/Invalid auth actor response/)
  })

  it('accepts actor without optional fields', () => {
    const actor = parseAuthActor({
      id: '1',
      type: 'operator',
      email: 'admin@example.com',
      role: 'admin',
      authenticated: false,
    })

    expect(actor.capabilities).toBeUndefined()
    expect(actor.auth_source).toBeUndefined()
  })
})

describe('parseJob', () => {
  const validJob = {
    id: 'job-1',
    server_id: 'srv-1',
    kind: 'configure_server',
    status: 'queued',
    current_step: 'validate',
    retry_count: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }

  it('accepts a valid job', () => {
    const job = parseJob(validJob)
    expect(job.id).toBe('job-1')
    expect(job.kind).toBe('configure_server')
    expect(job.status).toBe('queued')
  })

  it('rejects invalid job kind', () => {
    expect(() =>
      parseJob({
        id: 1,
        status: 'queued',
        kind: 'unknown_kind',
        current_step: '',
        retry_count: 0,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }),
    ).toThrow(/Invalid job response/)
  })

  it('rejects invalid job status', () => {
    expect(() =>
      parseJob({ ...validJob, status: 'invalid_status' }),
    ).toThrow(/Invalid job response/)
  })
})

describe('parseServerCatalogResponse', () => {
  it('accepts valid catalog with profiles', () => {
    const catalog = parseServerCatalogResponse({
      catalog: {
        locations: [
          { name: 'fsn1', description: 'Falkenstein DC1' },
        ],
        server_types: [
          {
            name: 'cx22',
            description: 'CX22',
            cores: 2,
            memory_gb: 4,
            disk_gb: 40,
            architecture: 'x86',
            available_at: ['fsn1'],
            prices: [
              {
                location_name: 'fsn1',
                hourly_gross: '0.0065',
                monthly_gross: '3.85',
                currency: 'EUR',
              },
            ],
          },
        ],
      },
      profiles: [
        {
          key: 'nginx-stack',
          name: 'Nginx Stack',
          description: 'Nginx + PHP-FPM',
          artifact_path: '/artifacts/nginx',
          support_level: 'supported',
          configure_guarantee: 'full',
        },
      ],
    })

    expect(catalog.catalog.locations).toHaveLength(1)
    expect(catalog.profiles[0].key).toBe('nginx-stack')
  })

  it('rejects malformed server catalog payload', () => {
    expect(() =>
      parseServerCatalogResponse({
        catalog: { locations: [], server_types: [] },
        profiles: [{ key: 'nginx-stack' }],
      }),
    ).toThrow(/Invalid server catalog response/)
  })
})

describe('parseStoredServer', () => {
  const validServer = {
    id: 'srv-1',
    provider_id: 'prov-1',
    provider_type: 'hetzner',
    name: 'web-1',
    location: 'fsn1',
    server_type: 'cx22',
    image: 'ubuntu-24.04',
    profile_key: 'nginx-stack',
    status: 'ready',
    setup_state: 'ready',
    has_key: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }

  it('accepts a valid server', () => {
    const server = parseStoredServer(validServer)
    expect(server.name).toBe('web-1')
    expect(server.status).toBe('ready')
  })

  it('rejects invalid server status', () => {
    expect(() =>
      parseStoredServer({ ...validServer, status: 'bogus' }),
    ).toThrow(/Invalid server response/)
  })
})

describe('parseStoredServers', () => {
  it('accepts an empty array', () => {
    expect(parseStoredServers([])).toEqual([])
  })
})

describe('parseStoredSite', () => {
  it('accepts a valid site', () => {
    const site = parseStoredSite({
      id: 'site-1',
      server_id: 'srv-1',
      server_name: 'web-1',
      name: 'my-site',
      status: 'active',
      deployment_state: 'deployed',
      runtime_health_state: 'healthy',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })
    expect(site.name).toBe('my-site')
  })
})

describe('parseStoredDomain', () => {
  it('accepts a valid domain', () => {
    const domain = parseStoredDomain({
      id: 'dom-1',
      hostname: 'example.com',
      kind: 'primary',
      source: 'manual',
      dns_state: 'verified',
      routing_state: 'active',
      is_primary: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })
    expect(domain.hostname).toBe('example.com')
    expect(domain.is_primary).toBe(true)
  })
})

describe('parseDeleteSiteResponse', () => {
  it('accepts a valid response', () => {
    const resp = parseDeleteSiteResponse({
      site_id: 'site-1',
      deleted: true,
      description: 'Site deleted',
    })
    expect(resp.deleted).toBe(true)
  })
})

describe('parseDeleteDomainResponse', () => {
  it('accepts a valid response', () => {
    const resp = parseDeleteDomainResponse({
      domain_id: 'dom-1',
      deleted: true,
      description: 'Domain deleted',
    })
    expect(resp.deleted).toBe(true)
  })
})

describe('parseDeleteServerResponse', () => {
  it('accepts a valid response', () => {
    const resp = parseDeleteServerResponse({
      server_id: 'srv-1',
      job_id: 'job-1',
      status: 'deleting',
      job_status: 'queued',
      async: true,
      description: 'Server deletion queued',
    })
    expect(resp.async).toBe(true)
  })
})

describe('parseCreateServerResponse', () => {
  it('accepts a valid response', () => {
    const resp = parseCreateServerResponse({
      server_id: 'srv-1',
      job_id: 'job-1',
      status: 'pending',
    })
    expect(resp.server_id).toBe('srv-1')
  })
})

describe('parseAgentInfo', () => {
  it('accepts a valid agent info', () => {
    const info = parseAgentInfo({
      connected: true,
      status: 'online',
      last_seen: '2026-01-01T00:00:00Z',
      version: '1.0.0',
      cpu_percent: 5.2,
      mem_used_mb: 512,
      mem_total_mb: 2048,
    })
    expect(info.connected).toBe(true)
    expect(info.status).toBe('online')
  })
})

describe('parseAgentStatusMapResponse', () => {
  it('accepts a map of agent statuses', () => {
    const map = parseAgentStatusMapResponse({
      'srv-1': { connected: true, status: 'online' },
      'srv-2': { connected: false, status: 'offline' },
    })
    expect(map['srv-1'].connected).toBe(true)
    expect(map['srv-2'].status).toBe('offline')
  })
})

describe('parseJobEvents', () => {
  it('accepts valid job events', () => {
    const events = parseJobEvents([
      {
        id: 'evt-1',
        job_id: 'job-1',
        seq: 1,
        event_type: 'step_started',
        level: 'info',
        step_key: 'validate',
        message: 'Validating request',
        occurred_at: '2026-01-01T00:00:00Z',
      },
    ])
    expect(events).toHaveLength(1)
    expect(events[0].seq).toBe(1)
  })
})

describe('parseActivity', () => {
  it('accepts a valid activity', () => {
    const activity = parseActivity({
      id: 'act-1',
      event_type: 'server.created',
      category: 'server',
      level: 'info',
      actor_type: 'operator',
      title: 'Server created',
      requires_attention: false,
      created_at: '2026-01-01T00:00:00Z',
    })
    expect(activity.title).toBe('Server created')
  })
})

describe('parseActivityListResponse', () => {
  it('accepts a valid activity list', () => {
    const resp = parseActivityListResponse({
      data: [
        {
          id: 'act-1',
          event_type: 'server.created',
          category: 'server',
          level: 'info',
          actor_type: 'operator',
          title: 'Server created',
          requires_attention: false,
          created_at: '2026-01-01T00:00:00Z',
        },
      ],
      next_cursor: 'abc123',
    })
    expect(resp.data).toHaveLength(1)
    expect(resp.next_cursor).toBe('abc123')
  })
})

describe('parseUnreadCountResponse', () => {
  it('accepts a valid unread count', () => {
    const resp = parseUnreadCountResponse({ count: 5 })
    expect(resp.count).toBe(5)
  })

  it('rejects non-numeric count', () => {
    expect(() =>
      parseUnreadCountResponse({ count: 'many' }),
    ).toThrow(/Invalid unread count response/)
  })
})

describe('parseServicesResponse', () => {
  it('accepts a valid services response', () => {
    const resp = parseServicesResponse({
      server_id: 'srv-1',
      agent_connected: true,
      services: [
        {
          name: 'nginx',
          description: 'Nginx web server',
          active_state: 'active',
          load_state: 'loaded',
        },
      ],
    })
    expect(resp.services).toHaveLength(1)
    expect(resp.services[0].name).toBe('nginx')
  })
})

describe('parseSiteHealthResponse', () => {
  it('accepts a valid site health response', () => {
    const resp = parseSiteHealthResponse({
      site_id: 'site-1',
      agent_connected: true,
      runtime_health_state: 'healthy',
    })
    expect(resp.site_id).toBe('site-1')
    expect(resp.agent_connected).toBe(true)
  })
})

describe('parseHealthResponse', () => {
  it('accepts a valid health response', () => {
    const resp = parseHealthResponse({ status: 'ok' })
    expect(resp.status).toBe('ok')
  })

  it('accepts health with callback url mode', () => {
    const resp = parseHealthResponse({
      status: 'ok',
      callback_url_mode: 'stable',
      callback_url_warning: 'some warning',
    })
    expect(resp.callback_url_mode).toBe('stable')
  })
})
