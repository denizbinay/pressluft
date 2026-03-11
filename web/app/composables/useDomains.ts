import { ref, readonly } from "vue";
import type {
  CreateDomainRequest,
  DeleteDomainResponse,
  StoredDomain,
  UpdateDomainRequest,
} from "~/lib/api-types";
import {
  parseDeleteDomainResponse,
  parseStoredDomain,
  parseStoredDomains,
} from "~/lib/api-runtime";

export type {
  CreateDomainRequest,
  DeleteDomainResponse,
  StoredDomain,
  UpdateDomainRequest,
} from "~/lib/api-types";

export function useDomains() {
  const { apiFetch } = useApiClient();
  const domains = ref<StoredDomain[]>([]);
  const loading = ref(false);
  const saving = ref(false);
  const error = ref("");

  const fetchDomains = async () => {
    loading.value = true;
    error.value = "";
    try {
      domains.value = parseStoredDomains(await apiFetch("/domains"));
      return domains.value;
    } catch (e: any) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  };

  const fetchSiteDomains = async (siteId: string): Promise<StoredDomain[]> => {
    error.value = "";
    return parseStoredDomains(await apiFetch(`/sites/${siteId}/domains`));
  };

  const createDomain = async (payload: CreateDomainRequest): Promise<StoredDomain> => {
    saving.value = true;
    error.value = "";
    try {
      return parseStoredDomain(
        await apiFetch("/domains", {
          method: "POST",
          body: payload,
        }),
      );
    } finally {
      saving.value = false;
    }
  };

  const createSiteDomain = async (
    siteId: string,
    payload: CreateDomainRequest,
  ): Promise<StoredDomain> => {
    saving.value = true;
    error.value = "";
    try {
      return parseStoredDomain(
        await apiFetch(`/sites/${siteId}/domains`, {
          method: "POST",
          body: payload,
        }),
      );
    } finally {
      saving.value = false;
    }
  };

  const updateDomain = async (
    domainId: string,
    payload: UpdateDomainRequest,
  ): Promise<StoredDomain> => {
    saving.value = true;
    error.value = "";
    try {
      return parseStoredDomain(
        await apiFetch(`/domains/${domainId}`, {
          method: "PATCH",
          body: payload,
        }),
      );
    } finally {
      saving.value = false;
    }
  };

  const deleteDomain = async (domainId: string): Promise<DeleteDomainResponse> => {
    error.value = "";
    return parseDeleteDomainResponse(
      await apiFetch(`/domains/${domainId}`, {
        method: "DELETE",
      }),
    );
  };

  return {
    domains: readonly(domains),
    loading: readonly(loading),
    saving: readonly(saving),
    error: readonly(error),
    fetchDomains,
    fetchSiteDomains,
    createDomain,
    createSiteDomain,
    updateDomain,
    deleteDomain,
  };
}
