import { ref, readonly } from "vue";
import type {
  CreateSiteRequest,
  DeleteSiteResponse,
  SiteHealthResponse,
  StoredSite,
  UpdateSiteRequest,
} from "~/lib/api-types";
import {
  parseDeleteSiteResponse,
  parseSiteHealthResponse,
  parseStoredSite,
  parseStoredSites,
} from "~/lib/api-runtime";

export type { CreateSiteRequest, DeleteSiteResponse, SiteHealthResponse, StoredSite, UpdateSiteRequest } from "~/lib/api-types";

export function useSites() {
  const { apiFetch } = useApiClient();
  const sites = ref<StoredSite[]>([]);
  const loading = ref(false);
  const saving = ref(false);
  const error = ref("");

  const fetchSites = async () => {
    loading.value = true;
    error.value = "";
    try {
      sites.value = parseStoredSites(await apiFetch("/sites"));
      return sites.value;
    } catch (e: any) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  };

  const fetchSite = async (siteId: string): Promise<StoredSite> => {
    error.value = "";
    return parseStoredSite(await apiFetch(`/sites/${siteId}`));
  };

  const fetchServerSites = async (serverId: string): Promise<StoredSite[]> => {
    error.value = "";
    return parseStoredSites(await apiFetch(`/servers/${serverId}/sites`));
  };

  const fetchSiteHealth = async (siteId: string): Promise<SiteHealthResponse> => {
    error.value = "";
    return parseSiteHealthResponse(await apiFetch(`/sites/${siteId}/health`));
  };

  const createSite = async (payload: CreateSiteRequest): Promise<StoredSite> => {
    saving.value = true;
    error.value = "";
    try {
      return parseStoredSite(
        await apiFetch("/sites", {
          method: "POST",
          body: payload,
        }),
      );
    } finally {
      saving.value = false;
    }
  };

  const updateSite = async (
    siteId: string,
    payload: UpdateSiteRequest,
  ): Promise<StoredSite> => {
    saving.value = true;
    error.value = "";
    try {
      return parseStoredSite(
        await apiFetch(`/sites/${siteId}`, {
          method: "PATCH",
          body: payload,
        }),
      );
    } finally {
      saving.value = false;
    }
  };

  const deleteSite = async (siteId: string): Promise<DeleteSiteResponse> => {
    error.value = "";
    return parseDeleteSiteResponse(
      await apiFetch(`/sites/${siteId}`, {
        method: "DELETE",
      }),
    );
  };

  return {
    sites: readonly(sites),
    loading: readonly(loading),
    saving: readonly(saving),
    error: readonly(error),
    fetchSites,
    fetchSite,
    fetchSiteHealth,
    fetchServerSites,
    createSite,
    updateSite,
    deleteSite,
  };
}
