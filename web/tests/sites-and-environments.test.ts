import { beforeEach, describe, expect, it } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import { defineComponent } from "vue";
import type { Environment, JobStatusResponse, Site } from "~/lib/api/types";
import SitesIndexPage from "~/pages/app/sites/index.vue";
import SiteDetailPage from "~/pages/app/sites/[id].vue";

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve, 0));
};

const now = "2026-02-21T00:00:00Z";

const job = (id: string, status: JobStatusResponse["status"], extra?: Partial<JobStatusResponse>): JobStatusResponse => {
  return {
    id,
    status,
    attempt_count: 0,
    max_attempts: 3,
    created_at: now,
    updated_at: now,
    ...extra,
  };
};

describe("sites and environments flows", () => {
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

  it("creates a site and refreshes list after job completion", async () => {
    const siteId = "11111111-1111-1111-1111-111111111111";
    const jobId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";

    const sites: Site[] = [];

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: sites.length,
    }));

    registerEndpoint("/api/sites", () => sites);

    registerEndpoint("/api/sites", {
      method: "POST",
      handler: () => ({ job_id: jobId }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      once: true,
      handler: () => job(jobId, "queued", { job_type: "site_create" }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      once: true,
      handler: () => job(jobId, "running", { job_type: "site_create" }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      handler: () => {
        if (sites.length === 0) {
          sites.push({
            id: siteId,
            name: "Acme",
            slug: "acme",
            status: "active",
            primary_environment_id: null,
            created_at: now,
            updated_at: now,
            state_version: 1,
          });
        }
        return job(jobId, "succeeded", { job_type: "site_create" });
      },
    });

    const wrapper = await mountSuspended(SitesIndexPage, { route: "/app/sites" });
    await flush();

    await wrapper.find("#site-name").setValue("Acme");
    await wrapper.find("#site-slug").setValue("acme");
    await wrapper.find("form").trigger("submit");
    await flush();
    await flush();
    await flush();

    expect(wrapper.text()).toContain("succeeded");
    expect(wrapper.text()).toContain("Acme");
  });

  it("shows validation error when site creation fails", async () => {
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 0,
    }));

    registerEndpoint("/api/sites", () => []);

    registerEndpoint("/api/sites", {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 400);
        return { code: "validation_failed", message: "slug must be unique" };
      },
    });

    const wrapper = await mountSuspended(SitesIndexPage, { route: "/app/sites" });
    await flush();

    await wrapper.find("#site-name").setValue("Acme");
    await wrapper.find("#site-slug").setValue("acme");
    await wrapper.find("form").trigger("submit");
    await flush();
    await flush();

    expect(wrapper.text()).toContain("slug must be unique");
  });

  it("creates an environment and refreshes list after job completion", async () => {
    const siteId = "22222222-2222-2222-2222-222222222222";
    const envId = "33333333-3333-3333-3333-333333333333";
    const jobId = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb";

    const envs: Environment[] = [
      {
        id: "44444444-4444-4444-4444-444444444444",
        site_id: siteId,
        name: "Production",
        slug: "production",
        environment_type: "production",
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
      },
    ];

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/sites/${siteId}`, () => ({
      id: siteId,
      name: "Example",
      slug: "example",
      status: "active",
      primary_environment_id: envs[0]?.id ?? null,
      created_at: now,
      updated_at: now,
      state_version: 1,
    } satisfies Site));

    registerEndpoint(`/api/sites/${siteId}/environments`, () => envs);

    registerEndpoint(`/api/sites/${siteId}/environments`, {
      method: "POST",
      handler: () => ({ job_id: jobId }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      once: true,
      handler: () => job(jobId, "queued", { job_type: "env_create" }),
    });

    registerEndpoint(`/api/jobs/${jobId}`, {
      handler: () => {
        if (!envs.some((e) => e.id === envId)) {
          envs.push({
            id: envId,
            site_id: siteId,
            name: "Staging",
            slug: "staging",
            environment_type: "staging",
            status: "active",
            node_id: envs[0].node_id,
            source_environment_id: envs[0].id,
            promotion_preset: "content-protect",
            preview_url: "https://staging-preview.example.test",
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
        }
        return job(jobId, "succeeded", { job_type: "env_create" });
      },
    });

    const wrapper = await mountSuspended(SiteDetailPage, { route: `/app/sites/${siteId}` });
    await flush();
    await flush();

    await wrapper.find("#env-type").setValue("staging");
    await wrapper.find("#env-name").setValue("Staging");
    await wrapper.find("#env-slug").setValue("staging");
    await wrapper.find("form").trigger("submit");

    await flush();
    await flush();
    await flush();

    expect(wrapper.text()).toContain("succeeded");
    expect(wrapper.text()).toContain("Staging");
  });

  it("shows validation error when environment creation fails", async () => {
    const siteId = "66666666-6666-6666-6666-666666666666";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/sites/${siteId}`, () => ({
      id: siteId,
      name: "Example",
      slug: "example",
      status: "active",
      primary_environment_id: null,
      created_at: now,
      updated_at: now,
      state_version: 1,
    } satisfies Site));

    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);

    registerEndpoint(`/api/sites/${siteId}/environments`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 400);
        return { code: "validation_failed", message: "slug must be unique" };
      },
    });

    const wrapper = await mountSuspended(SiteDetailPage, { route: `/app/sites/${siteId}` });
    await flush();

    await wrapper.find("#env-type").setValue("staging");
    await wrapper.find("#env-name").setValue("Staging");
    await wrapper.find("#env-slug").setValue("staging");
    await wrapper.find("form").trigger("submit");

    await flush();
    await flush();

    expect(wrapper.text()).toContain("slug must be unique");
  });
});
