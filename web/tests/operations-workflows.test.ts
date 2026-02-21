import { beforeEach, describe, expect, it } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import { defineComponent } from "vue";
import type { Backup, Environment } from "~/lib/api/types";
import EnvironmentDetailPage from "~/pages/app/environments/[id].vue";

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve, 0));
};

const settle = async (steps = 8): Promise<void> => {
  for (let i = 0; i < steps; i += 1) {
    await flush();
  }
};

const now = "2026-02-21T00:00:00Z";

const baseEnv = (id: string, siteId: string): Environment => ({
  id,
  site_id: siteId,
  name: "Staging",
  slug: "staging",
  environment_type: "staging",
  status: "active",
  node_id: "55555555-5555-5555-5555-555555555555",
  source_environment_id: null,
  promotion_preset: "content-protect",
  preview_url: "https://preview.example.test",
  primary_domain_id: null,
  current_release_id: null,
  drift_status: "unknown",
  drift_checked_at: null,
  last_drift_check_id: null,
  fastcgi_cache_enabled: true,
  redis_cache_enabled: true,
  created_at: now,
  updated_at: now,
  state_version: 1,
});

describe("operations workflows", () => {
  const AuthInit = defineComponent({
    setup() {
      useAuthSession();
      return {};
    },
    template: "<div />",
  });

  beforeEach(async () => {
    await mountSuspended(AuthInit, { route: false });
    useAuthSession().status.value = "unknown";
  });

  it("creates a backup and refreshes list", async () => {
    const envId = "11111111-1111-1111-1111-111111111111";
    const siteId = "22222222-2222-2222-2222-222222222222";
    const jobId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";

    const backups: Backup[] = [];

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => baseEnv(envId, siteId));
    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);

    registerEndpoint(`/api/environments/${envId}/backups`, () => backups);

    registerEndpoint(`/api/environments/${envId}/backups`, {
      method: "POST",
      handler: () => ({ job_id: jobId }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      once: true,
      handler: () => ({
        id: jobId,
        job_type: "backup_create",
        status: "queued",
        attempt_count: 0,
        max_attempts: 3,
        created_at: now,
        updated_at: now,
      }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      handler: () => {
        if (backups.length === 0) {
          backups.push({
            id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
            environment_id: envId,
            backup_scope: "full",
            status: "completed",
            storage_type: "s3",
            storage_path: "s3://bucket/backups/test.tar.gz",
            retention_until: "2026-03-01T00:00:00Z",
            checksum: null,
            size_bytes: null,
            created_at: now,
            completed_at: now,
          });
        }
        return {
          id: jobId,
          job_type: "backup_create",
          status: "succeeded",
          attempt_count: 1,
          max_attempts: 3,
          created_at: now,
          updated_at: now,
        };
      },
    });

    registerEndpoint(`/api/environments/${envId}/domains`, () => []);

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find("#backup-scope").setValue("full");
    await wrapper.find('[data-testid="backup-form"]').trigger("submit");
    await settle(12);

    expect(wrapper.text()).toContain("completed");
    expect(wrapper.text()).toContain("retention 2026-03-01T00:00:00Z");
  });

  it("shows conflict error when adding domain is blocked", async () => {
    const envId = "33333333-3333-3333-3333-333333333333";
    const siteId = "44444444-4444-4444-4444-444444444444";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => baseEnv(envId, siteId));
    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);
    registerEndpoint(`/api/environments/${envId}/backups`, () => []);
    registerEndpoint(`/api/environments/${envId}/domains`, () => []);

    registerEndpoint(`/api/environments/${envId}/domains`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 409);
        return { code: "domain_conflict", message: "Domain already exists" };
      },
    });

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find("#domain-host").setValue("example.com");
    await wrapper.find('[data-testid="domain-form"]').trigger("submit");
    await settle();

    expect(wrapper.text()).toContain("Domain already exists");
  });

  it("maps magic login node errors deterministically", async () => {
    const envId = "55555555-5555-5555-5555-555555555555";
    const siteId = "66666666-6666-6666-6666-666666666666";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => baseEnv(envId, siteId));
    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);
    registerEndpoint(`/api/environments/${envId}/backups`, () => []);
    registerEndpoint(`/api/environments/${envId}/domains`, () => []);

    registerEndpoint(`/api/environments/${envId}/magic-login`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 502);
        return { code: "node_unreachable", message: "SSH failed" };
      },
    });

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find('[data-testid="magic-login-button"]').trigger("click");
    await settle();

    expect(wrapper.text()).toContain("Node unreachable");
  });
});
