export interface ApiErrorResponse {
  code: string;
  message: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
}

export interface LogoutResponse {
  success: boolean;
}

export interface JobResponse {
  job_id: string;
}

export interface CreateSiteRequest {
  name: string;
  slug: string;
}

export type SiteStatus = "active" | "cloning" | "deploying" | "restoring" | "failed";

export interface Site {
  id: string;
  name: string;
  slug: string;
  status: SiteStatus;
  primary_environment_id: string | null;
  created_at: string;
  updated_at: string;
  state_version: number;
}

export type EnvironmentType = "production" | "staging" | "clone";
export type EnvironmentStatus = "active" | "cloning" | "deploying" | "restoring" | "failed";
export type PromotionPreset = "content-protect" | "commerce-protect";
export type DriftStatus = "unknown" | "clean" | "drifted";

export type EnvironmentCreateType = "staging" | "clone";

export interface CreateEnvironmentRequest {
  name: string;
  slug: string;
  type: EnvironmentCreateType;
  source_environment_id?: string | null;
  promotion_preset: PromotionPreset;
}

export interface Environment {
  id: string;
  site_id: string;
  name: string;
  slug: string;
  environment_type: EnvironmentType;
  status: EnvironmentStatus;
  node_id: string;
  source_environment_id: string | null;
  promotion_preset: PromotionPreset;
  preview_url: string;
  primary_domain_id: string | null;
  current_release_id: string | null;
  drift_status: DriftStatus;
  drift_checked_at: string | null;
  last_drift_check_id: string | null;
  fastcgi_cache_enabled: boolean;
  redis_cache_enabled: boolean;
  created_at: string;
  updated_at: string;
  state_version: number;
}

export type JobStatus = "queued" | "running" | "succeeded" | "failed" | "cancelled";

export interface JobStatusResponse {
  id: string;
  job_type?: string;
  status: JobStatus;
  site_id?: string | null;
  environment_id?: string | null;
  node_id?: string | null;
  attempt_count: number;
  max_attempts: number;
  run_after?: string | null;
  locked_at?: string | null;
  locked_by?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  error_code?: string | null;
  error_message?: string | null;
  created_at: string;
  updated_at: string;
}

export interface JobControlResponse {
  success: boolean;
  status: JobStatus;
}

export type ResourceStatus = "active" | "cloning" | "deploying" | "restoring" | "failed";

export interface ResourceResetResponse {
  success: boolean;
  status: ResourceStatus;
}

export interface MetricsResponse {
  jobs_running: number;
  jobs_queued: number;
  nodes_active: number;
  sites_total: number;
}

export type DeploySourceType = "git" | "upload";

export interface DeployRequest {
  source_type: DeploySourceType;
  source_ref: string;
}

export type UpdateScope = "core" | "plugins" | "themes" | "all";

export interface UpdatesRequest {
  scope: UpdateScope;
}

export interface RestoreRequest {
  backup_id: string;
}

export interface PromoteRequest {
  target_environment_id: string;
}

export type BackupScope = "db" | "files" | "full";
export type BackupStatus = "pending" | "running" | "completed" | "failed" | "expired";
export type BackupStorageType = "s3";

export interface CreateBackupRequest {
  backup_scope: BackupScope;
}

export interface Backup {
  id: string;
  environment_id: string;
  backup_scope: BackupScope;
  status: BackupStatus;
  storage_type: BackupStorageType;
  storage_path: string;
  retention_until: string;
  checksum: string | null;
  size_bytes: number | null;
  created_at: string;
  completed_at: string | null;
}

export interface AddDomainRequest {
  hostname: string;
}

export type TlsStatus = "pending" | "active" | "failed" | "disabled";
export type TlsIssuer = "letsencrypt";

export interface Domain {
  id: string;
  environment_id: string;
  hostname: string;
  tls_status: TlsStatus;
  tls_issuer: TlsIssuer;
  created_at: string;
  updated_at: string;
}

export interface PatchCacheRequest {
  fastcgi_cache_enabled?: boolean;
  redis_cache_enabled?: boolean;
}

export interface MagicLoginResponse {
  login_url: string;
  expires_at: string;
}
