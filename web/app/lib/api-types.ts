import type {
  Activity as GeneratedActivity,
  AgentInfo,
  CreateDomainRequest,
  CreateSiteRequest,
  CreateJobRequest as GeneratedCreateJobRequest,
  CreateServerRequest,
  CreateServerResponse as GeneratedCreateServerResponse,
  DeleteDomainResponse as GeneratedDeleteDomainResponse,
  DeleteServerResponse as GeneratedDeleteServerResponse,
  DeleteSiteResponse as GeneratedDeleteSiteResponse,
  Job as GeneratedJob,
  JobEvent,
  ServerCatalogResponse as GeneratedServerCatalogResponse,
  ServerProfile as GeneratedServerProfile,
  Service,
  SiteHealthCheck,
  SiteHealthResponse as GeneratedSiteHealthResponse,
  SiteHealthSnapshot,
  ServerTypePrice,
  ServicesResponse as GeneratedServicesResponse,
  StoredDomain as GeneratedStoredDomain,
  StoredServer as GeneratedStoredServer,
  StoredSite as GeneratedStoredSite,
  UnreadCountResponse,
  UpdateDomainRequest,
  UpdateSiteRequest,
} from "~/lib/api-contract";

export type {
  AgentInfo,
  CreateDomainRequest,
  CreateSiteRequest,
  CreateServerRequest,
  JobEvent,
  Service,
  SiteHealthCheck,
  SiteHealthSnapshot,
  ServerTypePrice,
  UnreadCountResponse,
  UpdateDomainRequest,
  UpdateSiteRequest,
};

export type StoredServer = Omit<GeneratedStoredServer, "id"> & { id: string };
export type StoredDomain = Omit<GeneratedStoredDomain, "id" | "site_id" | "parent_domain_id"> & {
  id: string;
  site_id?: string;
  parent_domain_id?: string;
};
export type StoredSite = Omit<GeneratedStoredSite, "id" | "server_id"> & {
  id: string;
  server_id: string;
};
export type CreateServerResponse = Omit<
  GeneratedCreateServerResponse,
  "server_id"
> & { server_id: string };
export type DeleteServerResponse = Omit<
  GeneratedDeleteServerResponse,
  "server_id"
> & { server_id: string };
export type DeleteSiteResponse = Omit<GeneratedDeleteSiteResponse, "site_id"> & {
  site_id: string;
};
export type DeleteDomainResponse = Omit<GeneratedDeleteDomainResponse, "domain_id"> & {
  domain_id: string;
};
export type ServicesResponse = Omit<GeneratedServicesResponse, "server_id"> & {
  server_id: string;
};
export type SiteHealthResponse = Omit<GeneratedSiteHealthResponse, "site_id"> & {
  site_id: string;
};
export type AgentStatusMapResponse = Record<string, AgentInfo>;
export type ServerProfile = Omit<GeneratedServerProfile, "image">;
export type ServerCatalogResponse = Omit<
  GeneratedServerCatalogResponse,
  "profiles"
> & {
  profiles: ServerProfile[];
};
export type CreateJobRequest = Omit<GeneratedCreateJobRequest, "server_id"> & {
  server_id?: string;
};
export type Job = Omit<GeneratedJob, "server_id"> & { server_id?: string };
export type Activity = Omit<
  GeneratedActivity,
  "resource_id" | "parent_resource_id"
> & {
  resource_id?: string;
  parent_resource_id?: string;
};
export interface ActivityListResponse {
  data: Activity[];
  next_cursor?: string;
}
