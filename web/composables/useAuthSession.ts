import type { LoginRequest } from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

type AuthStatus = "unknown" | "authenticated" | "guest";

interface RestoreSessionOptions {
  force?: boolean;
}

export const useAuthSession = () => {
  const api = useApiClient();
  const status = useState<AuthStatus>("auth:status", () => "unknown");
  const isRestoring = useState<boolean>("auth:is-restoring", () => false);
  const isAuthenticated = computed(() => status.value === "authenticated");

  const restoreSession = async (options: RestoreSessionOptions = {}): Promise<boolean> => {
    const force = options.force ?? false;

    if (!force && status.value === "authenticated") return true;
    if (status.value === "guest") {
      return false;
    }
    if (isRestoring.value) {
      return status.value === "authenticated";
    }

    isRestoring.value = true;
    try {
      await api.getMetrics();
      status.value = "authenticated";
      return true;
    } catch (error) {
      if (error instanceof ApiClientError && error.status === 401) {
        status.value = "guest";
        return false;
      }
      throw error;
    } finally {
      isRestoring.value = false;
    }
  };

  const login = async (payload: LoginRequest): Promise<void> => {
    await api.login(payload);
    status.value = "authenticated";
  };

  const logout = async (): Promise<void> => {
    try {
      await api.logout();
    } catch (error) {
      if (!(error instanceof ApiClientError) || error.status !== 401) {
        throw error;
      }
    } finally {
      status.value = "guest";
    }
  };

  const setGuest = (): void => {
    status.value = "guest";
  };

  return {
    status,
    isAuthenticated,
    restoreSession,
    login,
    logout,
    setGuest,
  };
};
