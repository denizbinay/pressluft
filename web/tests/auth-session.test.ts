import { describe, expect, it } from "vitest";
import { defineComponent } from "vue";
import { setResponseStatus } from "h3";
import { mountSuspended, registerEndpoint } from "@nuxt/test-utils/runtime";

const AuthProbe = defineComponent({
  setup() {
    const auth = useAuthSession();
    return { auth };
  },
  template: "<div />",
});

describe("auth session", () => {
  it("restoreSession marks authenticated on 200", async () => {
    registerEndpoint("/api/metrics", () => ({
      jobs_running: 0,
      jobs_queued: 0,
      nodes_active: 0,
      sites_total: 0,
    }));

    const wrapper = await mountSuspended(AuthProbe, { route: false });
    wrapper.vm.auth.status.value = "unknown";

    const ok = await wrapper.vm.auth.restoreSession({ force: true });
    expect(ok).toBe(true);
    expect(wrapper.vm.auth.isAuthenticated.value).toBe(true);
  });

  it("restoreSession marks guest on 401", async () => {
    registerEndpoint("/api/metrics", (event) => {
      setResponseStatus(event, 401);
      return { code: "unauthorized", message: "Authentication required" };
    });

    const wrapper = await mountSuspended(AuthProbe, { route: false });
    wrapper.vm.auth.status.value = "unknown";

    const ok = await wrapper.vm.auth.restoreSession({ force: true });
    expect(ok).toBe(false);
    expect(wrapper.vm.auth.status.value).toBe("guest");
  });

  it("login + logout update local auth state", async () => {
    registerEndpoint("/api/login", {
      method: "POST",
      handler: () => ({ success: true }),
    });
    registerEndpoint("/api/logout", {
      method: "POST",
      handler: (event) => {
        setResponseStatus(event, 401);
        return { code: "unauthorized", message: "Already signed out" };
      },
    });

    const wrapper = await mountSuspended(AuthProbe, { route: false });
    wrapper.vm.auth.status.value = "guest";

    await wrapper.vm.auth.login({ email: "test@example.com", password: "pw" });
    expect(wrapper.vm.auth.status.value).toBe("authenticated");

    await wrapper.vm.auth.logout();
    expect(wrapper.vm.auth.status.value).toBe("guest");
  });
});
