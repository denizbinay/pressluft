import { beforeEach, describe, expect, it, vi } from "vitest";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";
import LoginPage from "~/pages/login.vue";

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve, 0));
};

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

describe("login page", () => {
  it("navigates to protected shell on successful login", async () => {
    navigateToMock.mockClear();
    registerEndpoint("/api/metrics", {
      once: true,
      handler: (event) => {
        setResponseStatus(event, 401);
        return { code: "unauthorized", message: "Authentication required" };
      },
    });
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 0,
      sites_total: 0,
    }));

    registerEndpoint("/api/login", {
      method: "POST",
      handler: () => ({ success: true }),
    });

    const wrapper = await mountSuspended(LoginPage, { route: "/login?redirect=/app" });

    await wrapper.find("#email").setValue("operator@example.com");
    await wrapper.find("#password").setValue("pw");
    await wrapper.find("form").trigger("submit");
    await flush();
    await flush();

    expect(navigateToMock).toHaveBeenCalledWith("/app", { replace: true });
  });

  it("shows deterministic message on invalid credentials", async () => {
    navigateToMock.mockClear();
    registerEndpoint("/api/metrics", (event) => {
      setResponseStatus(event, 401);
      return { code: "unauthorized", message: "Authentication required" };
    });
    registerEndpoint("/api/login", {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 401);
        return { code: "unauthorized", message: "Invalid credentials" };
      },
    });

    const wrapper = await mountSuspended(LoginPage, { route: "/login" });

    await wrapper.find("#email").setValue("operator@example.com");
    await wrapper.find("#password").setValue("wrong");
    await wrapper.find("form").trigger("submit");
    await flush();
    await flush();

    expect(wrapper.text()).toContain("Invalid email or password.");
    expect(navigateToMock).not.toHaveBeenCalled();
  });
});
