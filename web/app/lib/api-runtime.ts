import { z } from "zod";
import { platformContract } from "~/lib/platform-contract.generated";
import type {
  Activity,
  ActivityListResponse,
  AgentInfo,
  AgentStatusMapResponse,
  DeleteDomainResponse,
  DeleteSiteResponse,
  CreateServerResponse,
  DeleteServerResponse,
  Job,
  JobEvent,
  ServerCatalogResponse,
  SiteHealthResponse,
  ServicesResponse,
  StoredDomain,
  StoredServer,
  StoredSite,
  UnreadCountResponse,
} from "~/lib/api-types";
import type { AuthActor } from "~/lib/api-contract";

const serverStatusSchema = z.enum(platformContract.server_statuses);
const setupStateSchema = z.enum(platformContract.setup_states);
const nodeStatusSchema = z.enum(platformContract.node_statuses);
const jobStatusSchema = z.enum(platformContract.job_statuses);
const jobKindSchema = z.enum(
  platformContract.job_kinds.map((spec) => spec.kind) as [string, ...string[]],
);
const supportLevelSchema = z.enum(platformContract.support_levels);
const callbackURLModeSchema = z.enum(platformContract.callback_url_modes);

const authActorSchema = z.object({
  id: z.string(),
  type: z.string(),
  email: z.string(),
  role: z.literal("admin"),
  capabilities: z.array(z.string()).optional(),
  authenticated: z.boolean(),
  auth_source: z.string().optional(),
});

const storedServerSchema = z.object({
  id: z.string(),
  provider_id: z.string(),
  provider_type: z.string(),
  provider_server_id: z.string().optional(),
  ipv4: z.string().optional(),
  ipv6: z.string().optional(),
  name: z.string(),
  location: z.string(),
  server_type: z.string(),
  image: z.string(),
  profile_key: z.string(),
  status: serverStatusSchema,
  setup_state: setupStateSchema,
  setup_last_error: z.string().optional(),
  action_id: z.string().optional(),
  action_status: z.string().optional(),
  has_key: z.boolean(),
  node_status: nodeStatusSchema.optional(),
  node_last_seen: z.string().optional(),
  node_version: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

const agentInfoSchema = z.object({
  connected: z.boolean(),
  status: nodeStatusSchema,
  last_seen: z.string().optional(),
  version: z.string().optional(),
  cpu_percent: z.number().optional(),
  mem_used_mb: z.number().optional(),
  mem_total_mb: z.number().optional(),
});

const serverProfileSchema = z.object({
  key: z.string(),
  name: z.string(),
  description: z.string(),
  artifact_path: z.string(),
  support_level: supportLevelSchema,
  configure_guarantee: z.string(),
  support_reason: z.string().optional(),
});

const serverCatalogSchema = z.object({
  locations: z.array(
    z.object({
      name: z.string(),
      description: z.string(),
      country: z.string().optional(),
      city: z.string().optional(),
      network_zone: z.string().optional(),
    }),
  ),
  server_types: z.array(
    z.object({
      name: z.string(),
      description: z.string(),
      cores: z.number(),
      memory_gb: z.number(),
      disk_gb: z.number(),
      architecture: z.string(),
      available_at: z.array(z.string()),
      prices: z.array(
        z.object({
          location_name: z.string(),
          hourly_gross: z.string(),
          monthly_gross: z.string(),
          currency: z.string(),
        }),
      ),
    }),
  ),
});

const serverCatalogResponseSchema = z.object({
  catalog: serverCatalogSchema,
  profiles: z.array(serverProfileSchema),
});

const createServerResponseSchema = z.object({
  server_id: z.string(),
  job_id: z.string(),
  status: serverStatusSchema,
});

const deleteServerResponseSchema = z.object({
  server_id: z.string(),
  job_id: z.string(),
  status: serverStatusSchema,
  job_status: jobStatusSchema,
  async: z.boolean(),
  description: z.string(),
});

const siteStatusSchema = z.enum([
  "draft",
  "active",
  "attention",
  "archived",
]);

const storedSiteSchema = z.object({
  id: z.string(),
  server_id: z.string(),
  server_name: z.string(),
  name: z.string(),
  wordpress_admin_email: z.string().optional(),
  primary_domain: z.string().optional(),
  status: siteStatusSchema,
  deployment_state: z.string(),
  deployment_status_message: z.string().optional(),
  last_deploy_job_id: z.string().optional(),
  last_deployed_at: z.string().optional(),
  runtime_health_state: z.string(),
  runtime_health_status_message: z.string().optional(),
  last_health_check_at: z.string().optional(),
  wordpress_path: z.string().optional(),
  php_version: z.string().optional(),
  wordpress_version: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

const storedDomainSchema = z.object({
  id: z.string(),
  hostname: z.string(),
  kind: z.string(),
  source: z.string(),
  dns_state: z.string(),
  routing_state: z.string(),
  dns_status_message: z.string().optional(),
  routing_status_message: z.string().optional(),
  last_checked_at: z.string().optional(),
  site_id: z.string().optional(),
  site_name: z.string().optional(),
  parent_domain_id: z.string().optional(),
  parent_hostname: z.string().optional(),
  is_primary: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

const deleteSiteResponseSchema = z.object({
  site_id: z.string(),
  deleted: z.boolean(),
  description: z.string(),
});

const deleteDomainResponseSchema = z.object({
  domain_id: z.string(),
  deleted: z.boolean(),
  description: z.string(),
});

const jobSchema = z.object({
  id: z.string(),
  server_id: z.string().optional(),
  kind: jobKindSchema,
  status: jobStatusSchema,
  current_step: z.string(),
  retry_count: z.number(),
  last_error: z.string().optional(),
  payload: z.string().optional(),
  started_at: z.string().optional(),
  finished_at: z.string().optional(),
  timeout_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
  command_id: z.string().optional(),
});

const jobEventSchema = z.object({
  id: z.string(),
  job_id: z.string(),
  seq: z.number(),
  event_type: z.string(),
  level: z.string(),
  step_key: z.string().optional(),
  status: z.string().optional(),
  message: z.string(),
  payload: z.string().optional(),
  occurred_at: z.string(),
});

const activitySchema = z.object({
  id: z.string(),
  event_type: z.string(),
  category: z.string(),
  level: z.string(),
  resource_type: z.string().optional(),
  resource_id: z.string().optional(),
  parent_resource_type: z.string().optional(),
  parent_resource_id: z.string().optional(),
  actor_type: z.string(),
  actor_id: z.string().optional(),
  title: z.string(),
  message: z.string().optional(),
  payload: z.string().optional(),
  requires_attention: z.boolean(),
  read_at: z.string().optional(),
  created_at: z.string(),
});

const activityListResponseSchema = z.object({
  data: z.array(activitySchema),
  next_cursor: z.string().optional(),
});

const unreadCountResponseSchema = z.object({
  count: z.number(),
});

const servicesResponseSchema = z.object({
  server_id: z.string(),
  agent_connected: z.boolean(),
  services: z.array(
    z.object({
      name: z.string(),
      description: z.string(),
      active_state: z.string(),
      load_state: z.string(),
    }),
  ),
});

const siteHealthResponseSchema = z.object({
  site_id: z.string(),
  agent_connected: z.boolean(),
  runtime_health_state: z.string(),
  runtime_health_status_message: z.string().optional(),
  last_health_check_at: z.string().optional(),
  snapshot: z.object({
    site_id: z.string(),
    hostname: z.string(),
    generated_at: z.string(),
    healthy: z.boolean(),
    summary: z.string(),
    services: z.array(
      z.object({
        name: z.string(),
        description: z.string(),
        active_state: z.string(),
        load_state: z.string(),
      }),
    ).optional(),
    checks: z.array(
      z.object({
        name: z.string(),
        ok: z.boolean(),
        detail: z.string().optional(),
      }),
    ).optional(),
    recent_errors: z.array(z.string()).optional(),
  }).optional(),
});

const agentStatusMapResponseSchema = z.record(z.string(), agentInfoSchema);

const healthResponseSchema = z.object({
  status: z.string(),
  callback_url_mode: callbackURLModeSchema.optional(),
  callback_url_warning: z.string().optional(),
});

function decode<T>(schema: z.ZodType<T>, payload: unknown, label: string): T {
  const parsed = schema.safeParse(payload);
  if (parsed.success) {
    return parsed.data;
  }
  const detail = parsed.error.issues
    .map((issue) => `${issue.path.join(".")}: ${issue.message}`)
    .join("; ");
  throw new Error(`Invalid ${label} response${detail ? `: ${detail}` : ""}`);
}

export const parseAuthActor = (payload: unknown): AuthActor =>
  decode(authActorSchema, payload, "auth actor") as AuthActor;
export const parseStoredServer = (payload: unknown): StoredServer =>
  decode(storedServerSchema, payload, "server");
export const parseStoredServers = (payload: unknown): StoredServer[] =>
  decode(z.array(storedServerSchema), payload, "server list");
export const parseServerCatalogResponse = (
  payload: unknown,
): ServerCatalogResponse =>
  decode(serverCatalogResponseSchema, payload, "server catalog");
export const parseCreateServerResponse = (
  payload: unknown,
): CreateServerResponse =>
  decode(createServerResponseSchema, payload, "create server");
export const parseDeleteServerResponse = (
  payload: unknown,
): DeleteServerResponse =>
  decode(deleteServerResponseSchema, payload, "delete server");
export const parseStoredSite = (payload: unknown): StoredSite =>
  decode(storedSiteSchema, payload, "site");
export const parseStoredSites = (payload: unknown): StoredSite[] =>
  decode(z.array(storedSiteSchema), payload, "site list");
export const parseStoredDomain = (payload: unknown): StoredDomain =>
  decode(storedDomainSchema, payload, "domain");
export const parseStoredDomains = (payload: unknown): StoredDomain[] =>
  decode(z.array(storedDomainSchema), payload, "domain list");
export const parseDeleteSiteResponse = (payload: unknown): DeleteSiteResponse =>
  decode(deleteSiteResponseSchema, payload, "delete site");
export const parseDeleteDomainResponse = (payload: unknown): DeleteDomainResponse =>
  decode(deleteDomainResponseSchema, payload, "delete domain");
export const parseAgentInfo = (payload: unknown): AgentInfo =>
  decode(agentInfoSchema, payload, "agent info");
export const parseSiteHealthResponse = (payload: unknown): SiteHealthResponse =>
  decode(siteHealthResponseSchema, payload, "site health");
export const parseAgentStatusMapResponse = (
  payload: unknown,
): AgentStatusMapResponse =>
  decode(
    agentStatusMapResponseSchema,
    payload,
    "agent status map",
  ) as AgentStatusMapResponse;
export const parseJob = (payload: unknown): Job =>
  decode(jobSchema, payload, "job");
export const parseJobEvents = (payload: unknown): JobEvent[] =>
  decode(z.array(jobEventSchema), payload, "job events");
export const parseActivity = (payload: unknown): Activity =>
  decode(activitySchema, payload, "activity");
export const parseActivityListResponse = (
  payload: unknown,
): ActivityListResponse =>
  decode(activityListResponseSchema, payload, "activity list");
export const parseUnreadCountResponse = (
  payload: unknown,
): UnreadCountResponse =>
  decode(unreadCountResponseSchema, payload, "unread count");
export const parseServicesResponse = (payload: unknown): ServicesResponse =>
  decode(servicesResponseSchema, payload, "services");
export const parseHealthResponse = (payload: unknown) =>
  decode(healthResponseSchema, payload, "health");
