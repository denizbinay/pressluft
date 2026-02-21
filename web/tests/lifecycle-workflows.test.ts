import { describe, expect, it } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import type { Environment } from "~/lib/api/types";
import EnvironmentDetailPage from "~/pages/app/environments/[id].vue";

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve, 0));
};

const settle = async (steps = 6): Promise<void> => {
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

describe("lifecycle workflows", () => {
  it("validates deploy form deterministically", async () => {
    const envId = "11111111-1111-1111-1111-111111111111";
    const siteId = "22222222-2222-2222-2222-222222222222";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => baseEnv(envId, siteId));
    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find("form").trigger("submit");
    await settle();

    expect(wrapper.text()).toContain("Source reference is required.");
  });

  it("starts deploy and renders terminal job status", async () => {
    const envId = "33333333-3333-3333-3333-333333333333";
    const siteId = "44444444-4444-4444-4444-444444444444";
    const jobId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => baseEnv(envId, siteId));
    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);

    registerEndpoint(`/api/environments/${envId}/deploy`, {
      method: "POST",
      handler: () => ({ job_id: jobId }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      once: true,
      handler: () => ({
        id: jobId,
        job_type: "env_deploy",
        status: "queued",
        attempt_count: 0,
        max_attempts: 3,
        created_at: now,
        updated_at: now,
      }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      handler: () => ({
        id: jobId,
        job_type: "env_deploy",
        status: "succeeded",
        attempt_count: 1,
        max_attempts: 3,
        created_at: now,
        updated_at: now,
      }),
    });

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find("#deploy-ref").setValue("git@github.com:acme/site.git#main");
    await wrapper.find("form").trigger("submit");
    await settle(10);

    expect(wrapper.text()).toContain("succeeded");
  });

  it("renders conflict errors for promote", async () => {
    const envId = "55555555-5555-5555-5555-555555555555";
    const siteId = "66666666-6666-6666-6666-666666666666";
    const targetEnvId = "77777777-7777-7777-7777-777777777777";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => ({
      ...baseEnv(envId, siteId),
      drift_status: "drifted",
    }));

    registerEndpoint(`/api/sites/${siteId}/environments`, () => [
      baseEnv(envId, siteId),
      {
        ...baseEnv(targetEnvId, siteId),
        id: targetEnvId,
        name: "Production",
        slug: "production",
        environment_type: "production",
      },
    ]);

    registerEndpoint(`/api/environments/${envId}/promote`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 409);
        return { code: "drift_not_clean", message: "Drift gate must be clean before promotion" };
      },
    });

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find("#promote-target").setValue(targetEnvId);
    await wrapper.find('[data-testid="promote-form"]').trigger("submit");
    await settle();

    expect(wrapper.text()).toContain("Drift gate must be clean before promotion");
  });
});
