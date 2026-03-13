<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import JobTimeline from "~/components/JobTimeline.vue";
import { useActivity } from "~/composables/useActivity";
import { useDomains, type StoredDomain } from "~/composables/useDomains";
import { useServers } from "~/composables/useServers";
import { useSites, type SiteHealthResponse, type StoredSite } from "~/composables/useSites";

const route = useRoute();
const router = useRouter();

const siteId = computed(() => {
  const raw = route.params.id;
  return typeof raw === "string" ? raw.trim() : "";
});

const { servers, fetchServers } = useServers();
const { fetchSite, fetchSiteHealth, updateSite, deleteSite, saving } = useSites();
const { activities, listSiteActivity } = useActivity();
const { fetchSiteDomains, createSiteDomain, updateDomain, deleteDomain } = useDomains();

const site = ref<StoredSite | null>(null);
const siteHealth = ref<SiteHealthResponse | null>(null);
const siteDomains = ref<StoredDomain[]>([]);
const loading = ref(true);
const pageError = ref("");
const successMessage = ref("");

const form = reactive({
  name: "",
  wordpressAdminEmail: "",
});

const hostnameForm = reactive({
  source: "preview",
  fallbackLabel: "",
  hostname: "",
});

const siteStatusMeta = (status: StoredSite["status"]) => {
  switch (status) {
    case "active":
      return { label: "Active", className: "border-primary/30 bg-primary/10 text-primary" };
    case "attention":
      return { label: "Attention", className: "border-accent/30 bg-accent/10 text-accent" };
    case "archived":
      return { label: "Archived", className: "border-border/60 bg-muted/70 text-muted-foreground" };
    default:
      return { label: "Draft", className: "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300" };
  }
};

const siteDeploymentMeta = (state: StoredSite["deployment_state"]) => {
  switch (state) {
    case "ready":
      return { label: "Live", className: "border-primary/30 bg-primary/10 text-primary" };
    case "failed":
      return { label: "Failed", className: "border-destructive/30 bg-destructive/10 text-destructive" };
    case "deploying":
      return { label: "Deploying", className: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200" };
    default:
      return { label: "Pending", className: "border-border/60 bg-muted/60 text-muted-foreground" };
  }
};

const siteRuntimeMeta = (state: StoredSite["runtime_health_state"]) => {
  switch (state) {
    case "healthy":
      return { label: "Healthy", className: "border-primary/30 bg-primary/10 text-primary" };
    case "issue":
      return { label: "Runtime issue", className: "border-destructive/30 bg-destructive/10 text-destructive" };
    case "unknown":
      return { label: "Unknown", className: "border-border/60 bg-muted/60 text-muted-foreground" };
    default:
      return { label: "Checking", className: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200" };
  }
};

const currentServer = computed(() => servers.value.find((server) => server.id === site.value?.server_id) || null);
const currentServerIPv4 = computed(() => currentServer.value?.ipv4 || "");

const hydrateForm = (value: StoredSite) => {
  form.name = value.name;
  form.wordpressAdminEmail = value.wordpress_admin_email || "";
};

const formatDate = (iso: string) => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
};

const loadActivity = async () => {
  if (!siteId.value) return;
  try {
    await listSiteActivity(siteId.value, { limit: 8 });
  } catch {
    // Timeline degrades quietly on detail page.
  }
};

const normalizeLabel = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/_/g, "-")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "");

const buildFallbackHostname = () => {
  const label = normalizeLabel(hostnameForm.fallbackLabel);
  if (!label || !currentServerIPv4.value) {
    return "";
  }
  return `${label}.${currentServerIPv4.value.replace(/\./g, "-")}.sslip.io`;
};

const refreshDomains = async () => {
  if (!siteId.value) return;
  siteDomains.value = await fetchSiteDomains(siteId.value);
  if (!currentServerIPv4.value && hostnameForm.source === "preview") {
    hostnameForm.source = "domain";
  }
};

const refreshHealth = async () => {
  if (!siteId.value) return;
  try {
    siteHealth.value = await fetchSiteHealth(siteId.value);
  } catch {
    siteHealth.value = null;
  }
};

const loadPage = async () => {
  if (!siteId.value) {
    pageError.value = "Invalid site ID";
    loading.value = false;
    return;
  }
  loading.value = true;
  pageError.value = "";
  try {
    const [loadedSite] = await Promise.all([fetchSite(siteId.value), fetchServers()]);
    site.value = loadedSite;
    hydrateForm(loadedSite);
    await Promise.all([loadActivity(), refreshDomains(), refreshHealth()]);
  } catch (e: any) {
    pageError.value = e.message || "Failed to load site";
  } finally {
    loading.value = false;
  }
};

const handleSave = async () => {
  if (!siteId.value) return;
  successMessage.value = "";
  pageError.value = "";
  try {
    const updated = await updateSite(siteId.value, {
      name: form.name,
      wordpress_admin_email: form.wordpressAdminEmail,
    });
    site.value = updated;
    hydrateForm(updated);
    await Promise.all([refreshDomains(), refreshHealth()]);
    successMessage.value = "Site details updated.";
  } catch (e: any) {
    pageError.value = e.message || "Failed to update site";
  }
};

const handleAssignHostname = async () => {
  if (!siteId.value) return;
  pageError.value = "";
  successMessage.value = "";
    try {
      const payload =
      hostnameForm.source === "preview"
        ? {
            hostname: buildFallbackHostname(),
            source: "fallback_resolver",
            is_primary: siteDomains.value.length === 0,
          }
        : {
            hostname: hostnameForm.hostname.trim(),
            source: "user",
            is_primary: siteDomains.value.length === 0,
          };
    const created = await createSiteDomain(siteId.value, payload);
    successMessage.value = `Assigned ${created.hostname}.`;
    hostnameForm.fallbackLabel = "";
    hostnameForm.hostname = "";
    await Promise.all([refreshDomains(), loadPage(), refreshHealth()]);
  } catch (e: any) {
    pageError.value = e.message || "Failed to assign hostname";
  }
};

const handleSetPrimary = async (domainId: string) => {
  pageError.value = "";
  successMessage.value = "";
  try {
    await updateDomain(domainId, { is_primary: true });
    successMessage.value = "Primary hostname updated.";
    await Promise.all([refreshDomains(), loadPage(), refreshHealth()]);
  } catch (e: any) {
    pageError.value = e.message || "Failed to update primary hostname";
  }
};

const handleRemoveHostname = async (domain: StoredDomain) => {
  if (!window.confirm(`Remove ${domain.hostname} from this site?`)) {
    return;
  }
  pageError.value = "";
  successMessage.value = "";
  try {
    await deleteDomain(domain.id);
    successMessage.value = `Removed ${domain.hostname}.`;
    await Promise.all([refreshDomains(), loadPage(), refreshHealth()]);
  } catch (e: any) {
    pageError.value = e.message || "Failed to remove hostname";
  }
};

const handleDelete = async () => {
  if (!site.value) return;
  if (!window.confirm(`Delete ${site.value.name}? This only removes the site record.`)) {
    return;
  }
  pageError.value = "";
  try {
    await deleteSite(site.value.id);
    router.push("/sites");
  } catch (e: any) {
    pageError.value = e.message || "Failed to delete site";
  }
};

onMounted(loadPage);

watch(siteId, async (value, previous) => {
  if (!value || value === previous) return;
  await loadPage();
});

watch(currentServerIPv4, (value) => {
  if (!value && hostnameForm.source === "preview") {
    hostnameForm.source = "domain";
  }
});
</script>

<template>
  <div class="space-y-8">
    <div v-if="loading" class="flex items-center justify-center py-24 text-sm text-muted-foreground">Loading site record...</div>

    <template v-else-if="site">
      <div class="flex flex-col gap-5 rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(18,34,42,0.96),rgba(18,58,56,0.9)_52%,rgba(28,38,61,0.92))] px-7 py-7 text-white shadow-[0_32px_120px_-52px_rgba(9,18,32,0.85)]">
        <NuxtLink to="/sites" class="text-xs font-semibold uppercase tracking-[0.22em] text-white/65 transition hover:text-white">Back to sites</NuxtLink>
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ site.name }}</h1>
              <Badge variant="outline" :class="siteStatusMeta(site.status).className">{{ siteStatusMeta(site.status).label }}</Badge>
              <Badge variant="outline" :class="siteDeploymentMeta(site.deployment_state).className">{{ siteDeploymentMeta(site.deployment_state).label }}</Badge>
              <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
            </div>
            <p class="mt-3 max-w-2xl text-base leading-7 text-white/72">
              {{ site.primary_domain || "No primary hostname assigned yet." }}
              {{ site.runtime_health_status_message || site.deployment_status_message || "Pressluft keeps deployment, routing, and runtime health truthfully in sync." }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-3 sm:min-w-[360px]">
            <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-white/55">Hostname</p>
              <p class="mt-2 text-lg font-semibold">{{ site.primary_domain || "Pending" }}</p>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-white/55">Install</p>
              <p class="mt-2 text-lg font-semibold">Managed baseline</p>
            </div>
          </div>
        </div>
      </div>

      <Alert v-if="pageError" class="border-destructive/30 bg-destructive/10 text-destructive">
        <AlertDescription>{{ pageError }}</AlertDescription>
      </Alert>
      <Alert v-if="successMessage" class="border-primary/30 bg-primary/10 text-primary">
        <AlertDescription>{{ successMessage }}</AlertDescription>
      </Alert>

      <div class="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Site</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Basic site details</h2>
            </CardHeader>
            <CardContent class="px-6 py-5">
              <form class="space-y-4" @submit.prevent="handleSave">
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">Site name</Label>
                  <Input v-model="form.name" />
                </div>

                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">WordPress admin email</Label>
                  <Input v-model="form.wordpressAdminEmail" type="email" />
                </div>

                <div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
                  Pressluft keeps the install path, PHP version, WordPress version, and certificate contact flow under managed defaults during this first live deployment phase.
                </div>

                <div class="flex flex-col gap-3 border-t border-border/50 pt-4 sm:flex-row sm:justify-between">
                <Button type="button" variant="ghost" class="justify-start text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleDelete">Delete site record</Button>
                <Button type="submit" class="bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving">{{ saving ? "Saving..." : "Save changes" }}</Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <div class="space-y-6">
          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Deployment</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Live deployment status</h2>
            </CardHeader>
            <CardContent class="space-y-4 px-6 py-5">
              <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" :class="siteDeploymentMeta(site.deployment_state).className">{{ siteDeploymentMeta(site.deployment_state).label }}</Badge>
                  <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
                  <span class="text-sm text-muted-foreground">{{ site.deployment_status_message || "Waiting for deployment activity." }}</span>
                </div>
                <p v-if="site.last_deployed_at" class="mt-3 text-xs text-muted-foreground">Last deployed {{ formatDate(site.last_deployed_at) }}</p>
              </div>
              <div v-if="site.last_deploy_job_id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <JobTimeline :job-id="site.last_deploy_job_id" :compact="true" />
              </div>
            </CardContent>
          </Card>

          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Runtime</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Managed runtime health</h2>
            </CardHeader>
            <CardContent class="space-y-4 px-6 py-5">
              <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
                  <span class="text-sm text-muted-foreground">{{ site.runtime_health_status_message || "Waiting for the first runtime health check." }}</span>
                </div>
                <p v-if="site.last_health_check_at" class="mt-3 text-xs text-muted-foreground">Last checked {{ formatDate(site.last_health_check_at) }}</p>
              </div>
              <div v-if="siteHealth?.snapshot" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <p class="text-sm font-semibold text-foreground">{{ siteHealth.snapshot.summary }}</p>
                <p class="mt-1 text-xs text-muted-foreground">Agent snapshot captured {{ formatDate(siteHealth.snapshot.generated_at) }}</p>
                <div v-if="siteHealth.snapshot.checks?.length" class="mt-4 space-y-2">
                  <div v-for="check in siteHealth.snapshot.checks" :key="check.name" class="flex items-start justify-between gap-3 rounded-xl border border-border/50 bg-muted/20 px-3 py-2 text-sm">
                    <span class="font-medium text-foreground">{{ check.name }}</span>
                    <span :class="check.ok ? 'text-primary' : 'text-destructive'">{{ check.ok ? 'OK' : (check.detail || 'Issue') }}</span>
                  </div>
                </div>
                <div v-if="siteHealth.snapshot.services?.length" class="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <span v-for="service in siteHealth.snapshot.services" :key="service.name" class="rounded-full border border-border/60 bg-muted/30 px-2.5 py-1">
                    {{ service.name }} {{ service.active_state }}
                  </span>
                </div>
                <div v-if="siteHealth.snapshot.recent_errors?.length" class="mt-4 rounded-xl border border-border/60 bg-muted/20 p-3">
                  <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent diagnostics</p>
                  <p v-for="entry in siteHealth.snapshot.recent_errors" :key="entry" class="mt-2 break-words text-xs text-muted-foreground">{{ entry }}</p>
                </div>
              </div>
              <div v-else class="rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
                {{ siteHealth?.agent_connected ? 'Agent diagnostics are available but no live snapshot was returned yet.' : 'Live agent diagnostics are unavailable right now, so this view falls back to the cached runtime health state.' }}
              </div>
            </CardContent>
          </Card>

          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Hostnames</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Primary and additional hostnames</h2>
            </CardHeader>
            <CardContent class="space-y-4 px-6 py-5">
              <div v-if="siteDomains.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">No hostnames assigned yet.</div>
              <div v-else class="space-y-3">
                <div v-for="domain in siteDomains" :key="domain.id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                  <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div>
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-semibold text-foreground">{{ domain.hostname }}</p>
                        <Badge v-if="domain.is_primary" variant="outline" class="border-primary/30 bg-primary/10 text-primary">Primary</Badge>
                        <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domain.source === 'fallback_resolver' ? 'Preview URL' : domain.parent_domain_id ? 'Child hostname' : 'Exact hostname' }}</Badge>
                        <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">DNS {{ domain.dns_state }}</Badge>
                        <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">Routing {{ domain.routing_state }}</Badge>
                      </div>
                       <p class="mt-1 text-xs text-muted-foreground">
                          {{ domain.parent_hostname || (domain.source === "fallback_resolver" ? "Pressluft generated this preview URL from the current server IP." : "User-managed hostname record.") }}
                        </p>
                       <p v-if="domain.routing_status_message || domain.dns_status_message" class="mt-2 text-xs text-muted-foreground">
                         {{ domain.routing_status_message || domain.dns_status_message }}
                       </p>
                     </div>
                     <div class="flex flex-wrap items-center gap-2">
                       <Button v-if="!domain.is_primary" type="button" variant="outline" size="sm" @click="handleSetPrimary(domain.id)">Make primary</Button>
                       <Button type="button" variant="ghost" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleRemoveHostname(domain)">Remove</Button>
                     </div>
                  </div>
                </div>
              </div>

              <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
                <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Attach hostname</p>
                <div class="mt-4 space-y-4">
                  <div class="space-y-1.5">
                    <Label class="text-sm font-medium text-muted-foreground">Destination</Label>
                    <select v-model="hostnameForm.source" class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40">
                      <option value="preview" :disabled="!currentServerIPv4">Preview URL</option>
                      <option value="domain">Exact hostname</option>
                    </select>
                  </div>

                  <template v-if="hostnameForm.source === 'preview'">
                    <div class="space-y-1.5">
                      <Label class="text-sm font-medium text-muted-foreground">Preview label</Label>
                      <Input v-model="hostnameForm.fallbackLabel" placeholder="preview" />
                    </div>
                    <div class="space-y-1.5">
                      <Label class="text-sm font-medium text-muted-foreground">Preview URL</Label>
                      <Input :model-value="buildFallbackHostname()" readonly placeholder="Current server needs an IPv4 address" />
                    </div>
                    <Alert class="border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-200">
                      <AlertDescription>
                        Preview URLs are fine for onboarding and evaluation, but they are not recommended for production.
                      </AlertDescription>
                    </Alert>
                  </template>

                  <template v-else>
                    <div class="space-y-1.5">
                      <Label class="text-sm font-medium text-muted-foreground">Exact hostname</Label>
                      <Input v-model="hostnameForm.hostname" placeholder="www.client-example.com" />
                    </div>
                  </template>
                </div>
                <Button
                  type="button"
                  class="mt-4 bg-accent text-accent-foreground hover:bg-accent/85"
                  :disabled="saving || (hostnameForm.source === 'preview' ? !buildFallbackHostname() : !hostnameForm.hostname.trim())"
                  @click="handleAssignHostname"
                >
                  Add hostname
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Placement</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Current hosting context</h2>
            </CardHeader>
            <CardContent class="space-y-4 px-6 py-5">
              <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
                <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Server</p>
                <p class="mt-2 text-lg font-semibold text-foreground">{{ currentServer?.name || site.server_name }}</p>
                <p class="mt-1 text-sm text-muted-foreground">
                  Sites stay top-level resources, but the current hosting target remains explicit while deployment is still server-bound.
                </p>
                <NuxtLink :to="`/servers/${site.server_id}?tab=sites`" class="mt-4 inline-flex text-sm font-medium text-accent transition hover:text-accent/80">Open server view</NuxtLink>
              </div>
              <div class="grid grid-cols-2 gap-3 text-sm">
                <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
                  <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Created</p>
                  <p class="mt-2 font-medium text-foreground">{{ formatDate(site.created_at) }}</p>
                </div>
                <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
                  <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Updated</p>
                  <p class="mt-2 font-medium text-foreground">{{ formatDate(site.updated_at) }}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent activity</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Timeline</h2>
            </CardHeader>
            <CardContent class="px-6 py-5">
              <div v-if="activities.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">No site activity recorded yet.</div>
              <div v-else class="space-y-3">
                <div v-for="activity in activities" :key="activity.id" class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                  <div class="flex items-center justify-between gap-3">
                    <p class="text-sm font-medium text-foreground">{{ activity.title }}</p>
                    <Badge variant="outline" class="border-border/60 bg-muted/50 text-xs text-muted-foreground">{{ activity.level }}</Badge>
                  </div>
                  <p v-if="activity.message" class="mt-1 text-sm text-muted-foreground">{{ activity.message }}</p>
                  <p class="mt-2 text-xs text-muted-foreground">{{ formatDate(activity.created_at) }}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>

    <Alert v-else class="border-destructive/30 bg-destructive/10 text-destructive">
      <AlertDescription>{{ pageError || "Site not found" }}</AlertDescription>
    </Alert>
  </div>
</template>
