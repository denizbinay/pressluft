import type {
  Activity as GeneratedActivity,
  AgentInfo,
  CreateJobRequest as GeneratedCreateJobRequest,
  CreateServerRequest,
  CreateServerResponse as GeneratedCreateServerResponse,
  DeleteServerResponse as GeneratedDeleteServerResponse,
  Job as GeneratedJob,
  JobEvent,
  ServerCatalogResponse as GeneratedServerCatalogResponse,
  ServerProfile as GeneratedServerProfile,
  Service,
  ServerTypePrice,
  ServicesResponse as GeneratedServicesResponse,
  StoredServer as GeneratedStoredServer,
  UnreadCountResponse,
} from "~/lib/api-contract";

export type {
  AgentInfo,
  CreateServerRequest,
  JobEvent,
  Service,
  ServerTypePrice,
  UnreadCountResponse,
};

export type StoredServer = Omit<GeneratedStoredServer, "id"> & { id: string };
export type CreateServerResponse = Omit<
  GeneratedCreateServerResponse,
  "server_id"
> & { server_id: string };
export type DeleteServerResponse = Omit<
  GeneratedDeleteServerResponse,
  "server_id"
> & { server_id: string };
export type ServicesResponse = Omit<GeneratedServicesResponse, "server_id"> & {
  server_id: string;
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
