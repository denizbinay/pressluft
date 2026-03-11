<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useActivity } from "~/composables/useActivity";
import { useDomains, type StoredDomain } from "~/composables/useDomains";
import { useServers } from "~/composables/useServers";
import { useSites, type StoredSite } from "~/composables/useSites";

const route = useRoute();
const router = useRouter();

const siteId = computed(() => {
  const raw = route.params.id;
  return typeof raw === "string" ? raw.trim() : "";
});

const { servers, fetchServers } = useServers();
const { fetchSite, updateSite, deleteSite, saving } = useSites();
const { activities, listSiteActivity } = useActivity();
const {
  fetchDomains,
  fetchSiteDomains,
  createSiteDomain,
  updateDomain,
  deleteDomain,
} = useDomains();

const site = ref<StoredSite | null>(null);
const siteDomains = ref<StoredDomain[]>([]);
const loading = ref(true);
const pageError = ref("");
const successMessage = ref("");

const form = reactive({
  serverId: "",
  name: "",
  status: "draft",
  wordpressPath: "",
  phpVersion: "",
  wordpressVersion: "",
});

const hostnameForm = reactive({
  mode: "wildcard",
  label: "",
  hostname: "",
  parentDomainId: "",
});

const siteStatusMeta = (status: StoredSite["status"]) => {
  switch (status) {
    case "active":
      return {
        label: "Active",
        className: "border-primary/30 bg-primary/10 text-primary",
      };
    case "attention":
      return {
        label: "Attention",
        className: "border-accent/30 bg-accent/10 text-accent",
      };
    case "archived":
      return {
        label: "Archived",
        className: "border-border/60 bg-muted/70 text-muted-foreground",
      };
    default:
      return {
        label: "Draft",
        className: "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300",
      };
  }
};

const currentServer = computed(() =>
  servers.value.find((server) => server.id === form.serverId),
);

const hydrateForm = (value: StoredSite) => {
  form.serverId = value.server_id;
  form.name = value.name;
  form.status = value.status;
  form.wordpressPath = value.wordpress_path || "";
  form.phpVersion = value.php_version || "";
  form.wordpressVersion = value.wordpress_version || "";
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

const allWildcardDomains = ref<StoredDomain[]>([]);
const availableWildcardDomains = ref<StoredDomain[]>([]);
const hasMultipleWildcardDomains = computed(() => availableWildcardDomains.value.length > 1);

const refreshDomains = async () => {
  if (!siteId.value) return;
  const [allDomains, assignedDomains] = await Promise.all([
    fetchDomains(),
    fetchSiteDomains(siteId.value),
  ]);
  allWildcardDomains.value = allDomains.filter(
    (domain) => domain.kind === "wildcard",
  );
  availableWildcardDomains.value = allWildcardDomains.value.filter((domain) => domain.status === "active");
  siteDomains.value = assignedDomains;
  if (!hostnameForm.parentDomainId && availableWildcardDomains.value[0]) {
    hostnameForm.parentDomainId = availableWildcardDomains.value[0].id;
  }
  if (!availableWildcardDomains.value.length) {
    hostnameForm.mode = "direct";
  }
};

const futureWildcardDomains = computed(() =>
  allWildcardDomains.value.filter((domain) => domain.status !== "active"),
);

const selectedWildcardDomain = computed(() =>
  availableWildcardDomains.value.find((domain) => domain.id === hostnameForm.parentDomainId) || null,
);

const wildcardDomainLabel = (domain: StoredDomain | null) => {
  if (!domain) return "";
  return domain.ownership === "platform" ? "Pressluft" : "Your domain";
};

const buildWildcardHostname = () => {
  const base = selectedWildcardDomain.value;
  if (!base) return "";
  const label = hostnameForm.label.trim().toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/^-+|-+$/g, "");
  if (!label) return "";
  return `${label}.${base.hostname}`;
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
    await Promise.all([loadActivity(), refreshDomains()]);
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
      server_id: form.serverId,
      name: form.name,
      status: form.status,
      wordpress_path: form.wordpressPath,
      php_version: form.phpVersion,
      wordpress_version: form.wordpressVersion,
    });
    site.value = updated;
    hydrateForm(updated);
    await refreshDomains();
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
    const hostname = hostnameForm.mode === "wildcard" ? buildWildcardHostname() : hostnameForm.hostname.trim();
    const created = await createSiteDomain(siteId.value, {
      hostname,
      parent_domain_id: hostnameForm.mode === "wildcard" ? hostnameForm.parentDomainId : undefined,
      is_primary: siteDomains.value.length === 0,
    });
    successMessage.value = `Assigned ${created.hostname}.`;
    hostnameForm.label = "";
    hostnameForm.hostname = "";
    await Promise.all([refreshDomains(), loadPage()]);
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
    await Promise.all([refreshDomains(), loadPage()]);
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
    await Promise.all([refreshDomains(), loadPage()]);
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
</script>

<template>
  <div class="space-y-8">
    <div v-if="loading" class="flex items-center justify-center py-24 text-sm text-muted-foreground">
      Loading site record...
    </div>

    <template v-else-if="site">
      <div class="flex flex-col gap-5 rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(18,34,42,0.96),rgba(18,58,56,0.9)_52%,rgba(28,38,61,0.92))] px-7 py-7 text-white shadow-[0_32px_120px_-52px_rgba(9,18,32,0.85)]">
        <NuxtLink to="/sites" class="text-xs font-semibold uppercase tracking-[0.22em] text-white/65 transition hover:text-white">
          Back to sites
        </NuxtLink>
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ site.name }}</h1>
              <Badge variant="outline" :class="siteStatusMeta(site.status).className">
                {{ siteStatusMeta(site.status).label }}
              </Badge>
            </div>
             <p class="mt-3 max-w-2xl text-base leading-7 text-white/72">
               {{ site.primary_domain || "No primary domain assigned yet." }}
               This site lives on {{ site.server_name }} and keeps every direct domain or wildcard-derived hostname visible as its own record.
             </p>
          </div>

          <div class="grid grid-cols-2 gap-3 sm:min-w-[360px]">
            <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-white/55">PHP</p>
              <p class="mt-2 text-2xl font-semibold">{{ site.php_version || "TBD" }}</p>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-white/55">WordPress</p>
              <p class="mt-2 text-2xl font-semibold">{{ site.wordpress_version || "TBD" }}</p>
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
            <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Metadata</p>
            <h2 class="mt-1 text-xl font-semibold text-foreground">Edit site profile</h2>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <form class="space-y-4" @submit.prevent="handleSave">
              <div class="grid gap-4 sm:grid-cols-2">
                <div class="space-y-1.5 sm:col-span-2">
                  <Label class="text-sm font-medium text-muted-foreground">Site name</Label>
                  <Input v-model="form.name" />
                </div>
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">Lifecycle state</Label>
                  <select v-model="form.status" class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40">
                    <option value="draft">Draft</option>
                    <option value="active">Active</option>
                    <option value="attention">Attention</option>
                    <option value="archived">Archived</option>
                  </select>
                </div>
                <div class="space-y-1.5 sm:col-span-2">
                  <Label class="text-sm font-medium text-muted-foreground">Current server</Label>
                  <select v-model="form.serverId" class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40">
                    <option v-for="server in servers" :key="server.id" :value="server.id">
                      {{ server.name }}
                    </option>
                  </select>
                </div>
                <div class="space-y-1.5 sm:col-span-2">
                  <Label class="text-sm font-medium text-muted-foreground">WordPress path</Label>
                  <Input v-model="form.wordpressPath" placeholder="/srv/www/client/current" />
                </div>
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">PHP version</Label>
                  <Input v-model="form.phpVersion" />
                </div>
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">WordPress version</Label>
                  <Input v-model="form.wordpressVersion" />
                </div>
              </div>

              <div class="flex flex-col gap-3 border-t border-border/50 pt-4 sm:flex-row sm:justify-between">
                <Button type="button" variant="ghost" class="justify-start text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleDelete">
                  Delete site record
                </Button>
                <Button type="submit" class="bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving">
                  {{ saving ? "Saving..." : "Save changes" }}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <div class="space-y-6">
          <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
            <CardHeader class="border-b border-border/50 px-6 py-5">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Domains</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Primary and additional domains</h2>
            </CardHeader>
            <CardContent class="space-y-4 px-6 py-5">
              <div v-if="siteDomains.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
                No domains assigned yet.
              </div>
              <div v-else class="space-y-3">
                <div v-for="domain in siteDomains" :key="domain.id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-semibold text-foreground">{{ domain.hostname }}</p>
                        <Badge v-if="domain.is_primary" variant="outline" class="border-primary/30 bg-primary/10 text-primary">Primary</Badge>
                        <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domain.parent_domain_id ? 'Wildcard child' : 'Direct domain' }}</Badge>
                        <Badge v-if="domain.parent_domain_id" variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domain.ownership === 'platform' ? 'Pressluft' : 'Your domain' }}</Badge>
                      </div>
                      <p class="mt-1 text-xs text-muted-foreground">
                        {{ domain.parent_hostname || (domain.ownership === "platform" ? "Provided by Pressluft" : "Managed by the agency") }}
                      </p>
                    </div>
                    <div class="flex gap-2">
                      <Button v-if="!domain.is_primary" type="button" variant="outline" size="sm" @click="handleSetPrimary(domain.id)">
                        Make primary
                      </Button>
                      <Button type="button" variant="ghost" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleRemoveHostname(domain)">
                        Remove
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

                <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
                   <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                     Attach Domain
                   </p>
                   <div class="flex gap-2">
                    <Button type="button" size="sm" :variant="hostnameForm.mode === 'wildcard' ? 'default' : 'outline'" @click="hostnameForm.mode = 'wildcard'" :disabled="availableWildcardDomains.length === 0">
                      Wildcard domain
                    </Button>
                    <Button type="button" size="sm" :variant="hostnameForm.mode === 'direct' ? 'default' : 'outline'" @click="hostnameForm.mode = 'direct'">
                      Direct domain
                    </Button>
                  </div>
                <div class="mt-4 grid gap-4 sm:grid-cols-2">
                  <template v-if="hostnameForm.mode === 'wildcard'">
                    <div class="space-y-1.5">
                      <Label class="text-sm font-medium text-muted-foreground">Child label</Label>
                      <Input v-model="hostnameForm.label" placeholder="northwind-live" />
                    </div>
                    <div v-if="hasMultipleWildcardDomains" class="space-y-1.5">
                      <Label class="text-sm font-medium text-muted-foreground">Wildcard root</Label>
                      <select v-model="hostnameForm.parentDomainId" class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40">
                        <option v-for="domain in availableWildcardDomains" :key="domain.id" :value="domain.id">
                          {{ domain.hostname }} ({{ domain.ownership === 'platform' ? 'Pressluft' : 'Your domain' }})
                        </option>
                      </select>
                    </div>
                    <div class="space-y-1.5 sm:col-span-2">
                      <Label class="text-sm font-medium text-muted-foreground">Result</Label>
                      <Input :model-value="buildWildcardHostname()" readonly placeholder="Enter a label to preview the allocated hostname" />
                    </div>
                    <div v-if="selectedWildcardDomain" class="sm:col-span-2 flex items-center gap-2 text-sm text-muted-foreground">
                      <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">
                        {{ wildcardDomainLabel(selectedWildcardDomain) }}
                      </Badge>
                      <span>{{ selectedWildcardDomain.hostname }} stays reusable while this site gets a concrete child hostname row.</span>
                    </div>
                    <p class="sm:col-span-2 text-sm text-muted-foreground">
                      Wildcard roots are better when you want previews, staging, rollbacks, or future generated hostnames without pre-registering each child.
                    </p>
                    <p v-if="availableWildcardDomains.length === 0" class="sm:col-span-2 text-sm text-muted-foreground">
                      No wildcard domains are available right now, so attach a direct domain instead.
                    </p>
                    <p v-if="futureWildcardDomains.length > 0" class="sm:col-span-2 text-sm text-muted-foreground">
                      Not active yet: {{ futureWildcardDomains.map((domain) => domain.hostname).join(", ") }}.
                    </p>
                  </template>
                  <template v-else>
                    <div class="space-y-1.5 sm:col-span-2">
                      <Label class="text-sm font-medium text-muted-foreground">Hostname</Label>
                      <Input v-model="hostnameForm.hostname" placeholder="www.client-example.com" />
                    </div>
                  </template>
                </div>
                <Button type="button" class="mt-4 bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving || (hostnameForm.mode === 'wildcard' ? !buildWildcardHostname() : !hostnameForm.hostname.trim())" @click="handleAssignHostname">
                  Add domain
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
                  This relationship stays explicit so future staging moves and
                  migrations have a clear current-home reference.
                </p>
                <NuxtLink :to="`/servers/${form.serverId}?tab=sites`" class="mt-4 inline-flex text-sm font-medium text-accent transition hover:text-accent/80">
                  Open server view
                </NuxtLink>
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
              <div v-if="activities.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
                No site activity recorded yet.
              </div>
              <div v-else class="space-y-3">
                <div v-for="activity in activities" :key="activity.id" class="rounded-2xl border border-border/60 bg-background/70 px-4 py-3">
                  <div class="flex items-center justify-between gap-3">
                    <p class="text-sm font-medium text-foreground">{{ activity.title }}</p>
                    <Badge variant="outline" class="border-border/60 bg-muted/50 text-xs text-muted-foreground">
                      {{ activity.level }}
                    </Badge>
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
