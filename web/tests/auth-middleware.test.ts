import { beforeEach, describe, expect, it, vi } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import { defineComponent } from "vue";
import middleware from "~/middleware/auth.global";

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

const AuthInit = defineComponent({
  setup() {
    // Ensure Nuxt app context exists for useState.
    useAuthSession();
    return {};
  },
  template: "<div />",
});

describe("auth route middleware", () => {
  beforeEach(async () => {
    navigateToMock.mockClear();
    await mountSuspended(AuthInit, { route: false });
    useAuthSession().status.value = "unknown";
  });

  it("redirects guest from protected route to login", async () => {
    registerEndpoint("/api/metrics", (event) => {
      setResponseStatus(event, 401);
      return { code: "unauthorized", message: "Authentication required" };
    });

    await middleware(
      {
        path: "/app",
        fullPath: "/app",
      } as any,
      {} as any,
    );

    expect(navigateToMock).toHaveBeenCalledWith(
      {
        path: "/login",
        query: { redirect: "/app" },
      },
      { replace: true },
    );
  });

  it("redirects / to /app when authenticated", async () => {
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 0,
      sites_total: 0,
    }));

    await middleware({ path: "/", fullPath: "/" } as any, {} as any);

    expect(navigateToMock).toHaveBeenCalledWith("/app", { replace: true });
  });

  it("redirects /login to /app when authenticated", async () => {
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 0,
      sites_total: 0,
    }));

    await middleware({ path: "/login", fullPath: "/login" } as any, {} as any);

    expect(navigateToMock).toHaveBeenCalledWith("/app", { replace: true });
  });
});
