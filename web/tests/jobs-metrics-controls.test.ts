import { beforeEach, describe, expect, it, vi } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import JobDetailPage from "~/pages/app/jobs/[id].vue";
import EnvironmentDetailPage from "~/pages/app/environments/[id].vue";
import AppLayout from "~/layouts/app.vue";

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve, 0));
};

const settle = async (steps = 8): Promise<void> => {
  for (let i = 0; i < steps; i += 1) {
    await flush();
  }
};

const now = "2026-02-21T00:00:00Z";

const { navigateToMock } = vi.hoisted(() => ({
  navigateToMock: vi.fn(async () => undefined as any),
}));

vi.mock("#app/composables/router", async (importOriginal) => {
  const mod = await importOriginal<any>();
  return {
    ...mod,
    navigateTo: navigateToMock,
  };
});

beforeEach(() => {
  navigateToMock.mockClear();
});

describe("jobs, metrics, and control surfaces", () => {
  it("renders metrics snapshot in authenticated shell", async () => {
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 1,
      jobs_queued: 2,
      nodes_active: 3,
      sites_total: 4,
    }));

    const wrapper = await mountSuspended(AppLayout, {
      route: "/app/jobs",
      slots: { default: "<div />" },
    });
    await settle();

    expect(wrapper.find('[data-testid="metrics"]').text()).toContain("1 running / 2 queued");
    expect(wrapper.find('[data-testid="metrics"]').text()).toContain("3");
    expect(wrapper.find('[data-testid="metrics"]').text()).toContain("4");
  });

  it("surfaces cancel conflict errors deterministically", async () => {
    const jobId = "11111111-1111-1111-1111-111111111111";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/jobs/${jobId}`, () => ({
      id: jobId,
      job_type: "env_deploy",
      status: "running",
      attempt_count: 0,
      max_attempts: 3,
      created_at: now,
      updated_at: now,
    }));

    registerEndpoint(`/api/jobs/${jobId}/cancel`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 409);
        return { code: "job_not_cancellable", message: "Job is already finished" };
      },
    });

    const wrapper = await mountSuspended(JobDetailPage, { route: `/app/jobs/${jobId}` });
    await settle();

    await wrapper.find('[data-testid="cancel-job"]').trigger("click");
    await settle();

    expect(wrapper.text()).toContain("Job is already finished");
  });

  it("redirects to login when metrics returns 401", async () => {
    registerEndpoint("/api/metrics", (event) => {
      setResponseStatus(event, 401);
      return { code: "unauthorized", message: "Authentication required" };
    });

    await mountSuspended(AppLayout, {
      route: "/app/jobs",
      slots: { default: "<div />" },
    });
    await settle();

    expect(navigateToMock).toHaveBeenCalledWith("/login", { replace: true });
  });

  it("surfaces reset conflict errors for failed environment", async () => {
    const envId = "22222222-2222-2222-2222-222222222222";
    const siteId = "33333333-3333-3333-3333-333333333333";

    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 1,
      sites_total: 1,
    }));

    registerEndpoint(`/api/environments/${envId}`, () => ({
      id: envId,
      site_id: siteId,
      name: "Failed",
      slug: "failed",
      environment_type: "staging",
      status: "failed",
      node_id: "44444444-4444-4444-4444-444444444444",
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
    }));

    registerEndpoint(`/api/sites/${siteId}/environments`, () => []);
    registerEndpoint(`/api/environments/${envId}/backups`, () => []);
    registerEndpoint(`/api/environments/${envId}/domains`, () => []);

    registerEndpoint(`/api/environments/${envId}/reset`, {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 409);
        return { code: "reset_validation_failed", message: "Reset blocked by server validation" };
      },
    });

    const wrapper = await mountSuspended(EnvironmentDetailPage, { route: `/app/environments/${envId}` });
    await settle();

    await wrapper.find('[data-testid="reset-env"]').trigger("click");
    await settle();

    expect(wrapper.text()).toContain("Reset blocked by server validation");
  });
});
