import type {
  AddDomainRequest,
  ApiErrorResponse,
  Backup,
  CreateBackupRequest,
  CreateEnvironmentRequest,
  CreateSiteRequest,
  DeployRequest,
  Domain,
  Environment,
  JobControlResponse,
  JobResponse,
  JobStatusResponse,
  LoginRequest,
  LoginResponse,
  LogoutResponse,
  MagicLoginResponse,
  MetricsResponse,
  PatchCacheRequest,
  PromoteRequest,
  Site,
  ResourceResetResponse,
  RestoreRequest,
  UpdatesRequest,
} from "./types";

export class ApiClientError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiClientError";
    this.status = status;
    this.code = code;
  }
}

interface ApiClientOptions {
  baseUrl: string;
  fetchImpl?: typeof fetch;
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly fetchImpl: typeof fetch;

  constructor(options: ApiClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  async login(payload: LoginRequest): Promise<LoginResponse> {
    return this.request<LoginResponse>("/login", {
      method: "POST",
      body: payload,
    });
  }

  async logout(): Promise<LogoutResponse> {
    return this.request<LogoutResponse>("/logout", {
      method: "POST",
    });
  }

  async listSites(): Promise<Site[]> {
    return this.request<Site[]>("/sites");
  }

  async createSite(payload: CreateSiteRequest): Promise<JobResponse> {
    return this.request<JobResponse>("/sites", {
      method: "POST",
      body: payload,
    });
  }

  async getSite(id: string): Promise<Site> {
    return this.request<Site>(`/sites/${id}`);
  }

  async listSiteEnvironments(siteId: string): Promise<Environment[]> {
    return this.request<Environment[]>(`/sites/${siteId}/environments`);
  }

  async createEnvironment(siteId: string, payload: CreateEnvironmentRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/sites/${siteId}/environments`, {
      method: "POST",
      body: payload,
    });
  }

  async getEnvironment(id: string): Promise<Environment> {
    return this.request<Environment>(`/environments/${id}`);
  }

  async deployEnvironment(id: string, payload: DeployRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/deploy`, {
      method: "POST",
      body: payload,
    });
  }

  async applyEnvironmentUpdates(id: string, payload: UpdatesRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/updates`, {
      method: "POST",
      body: payload,
    });
  }

  async restoreEnvironment(id: string, payload: RestoreRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/restore`, {
      method: "POST",
      body: payload,
    });
  }

  async runDriftCheck(id: string): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/drift-check`, {
      method: "POST",
    });
  }

  async promoteEnvironment(id: string, payload: PromoteRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/promote`, {
      method: "POST",
      body: payload,
    });
  }

  async listEnvironmentBackups(id: string): Promise<Backup[]> {
    return this.request<Backup[]>(`/environments/${id}/backups`);
  }

  async createEnvironmentBackup(id: string, payload: CreateBackupRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/backups`, {
      method: "POST",
      body: payload,
    });
  }

  async listEnvironmentDomains(id: string): Promise<Domain[]> {
    return this.request<Domain[]>(`/environments/${id}/domains`);
  }

  async addEnvironmentDomain(id: string, payload: AddDomainRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/domains`, {
      method: "POST",
      body: payload,
    });
  }

  async deleteDomain(domainId: string): Promise<JobResponse> {
    return this.request<JobResponse>(`/domains/${domainId}`, {
      method: "DELETE",
    });
  }

  async updateEnvironmentCache(id: string, payload: PatchCacheRequest): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/cache`, {
      method: "PATCH",
      body: payload,
    });
  }

  async purgeEnvironmentCache(id: string): Promise<JobResponse> {
    return this.request<JobResponse>(`/environments/${id}/cache/purge`, {
      method: "POST",
    });
  }

  async createMagicLogin(id: string): Promise<MagicLoginResponse> {
    return this.request<MagicLoginResponse>(`/environments/${id}/magic-login`, {
      method: "POST",
    });
  }

  async listJobs(): Promise<JobStatusResponse[]> {
    return this.request<JobStatusResponse[]>("/jobs");
  }

  async getJob(id: string): Promise<JobStatusResponse> {
    return this.request<JobStatusResponse>(`/jobs/${id}`);
  }

  async cancelJob(id: string): Promise<JobControlResponse> {
    return this.request<JobControlResponse>(`/jobs/${id}/cancel`, {
      method: "POST",
    });
  }

  async resetSite(id: string): Promise<ResourceResetResponse> {
    return this.request<ResourceResetResponse>(`/sites/${id}/reset`, {
      method: "POST",
    });
  }

  async resetEnvironment(id: string): Promise<ResourceResetResponse> {
    return this.request<ResourceResetResponse>(`/environments/${id}/reset`, {
      method: "POST",
    });
  }

  async getMetrics(): Promise<MetricsResponse> {
    return this.request<MetricsResponse>("/metrics");
  }

  private async request<T>(path: string, init?: Omit<RequestInit, "body"> & { body?: unknown }): Promise<T> {
    const response = await this.fetchImpl(`${this.baseUrl}${path}`, {
      method: init?.method ?? "GET",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: init?.body === undefined ? undefined : JSON.stringify(init.body),
    });

    if (!response.ok) {
      const errorPayload = await this.tryParseError(response);
      throw new ApiClientError(
        response.status,
        errorPayload?.message ?? response.statusText,
        errorPayload?.code,
      );
    }

    return (await response.json()) as T;
  }

  private async tryParseError(response: Response): Promise<ApiErrorResponse | null> {
    try {
      const payload = (await response.json()) as Partial<ApiErrorResponse>;
      if (typeof payload.message === "string") {
        return {
          code: typeof payload.code === "string" ? payload.code : "unknown_error",
          message: payload.message,
        };
      }
      return null;
    } catch {
      return null;
    }
  }
}
