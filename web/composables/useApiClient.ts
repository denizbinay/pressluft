import { ApiClient } from "~/lib/api/client";

export const useApiClient = (): ApiClient => {
  const config = useRuntimeConfig();
  return new ApiClient({ baseUrl: config.public.apiBase });
};
