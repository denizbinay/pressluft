<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useDomains } from "~/composables/useDomains";
import { useServers } from "~/composables/useServers";
import { useSites, type StoredSite } from "~/composables/useSites";

const route = useRoute();

const { servers, fetchServers } = useServers();
const { domains, fetchDomains } = useDomains();
const { sites, loading, saving, error, fetchSites, createSite } = useSites();

const pageError = ref("");
const successMessage = ref("");

const form = reactive({
  serverId: "",
  name: "",
  status: "draft",
  wordpressPath: "/srv/www/",
  phpVersion: "8.3",
  wordpressVersion: "6.8",
  hostnameSource: "fallback_resolver",
  fallbackLabel: "",
  userDomainMode: "base_domain",
  baseDomainId: "",
  userDomainLabel: "",
  exactHostname: "",
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
      return { label: "Pending", className: "border-border/60 bg-muted/70 text-muted-foreground" };
  }
};

const siteStats = computed(() => {
  const items = sites.value;
  return {
    total: items.length,
    active: items.filter((site) => site.status === "active").length,
    draft: items.filter((site) => site.status === "draft").length,
    attention: items.filter((site) => site.status === "attention").length,
  };
});

const serverOptions = computed(() => [...servers.value].sort((a, b) => a.name.localeCompare(b.name)));
const selectedServer = computed(() => serverOptions.value.find((server) => server.id === form.serverId) || null);
const selectedServerName = computed(() => selectedServer.value?.name || "Pick a server");
const selectedServerIPv4 = computed(() => selectedServer.value?.ipv4 || "");

const userBaseDomains = computed(() =>
  domains.value.filter((domain) => domain.kind === "base_domain" && domain.source === "user"),
);

const selectedBaseDomain = computed(() =>
  userBaseDomains.value.find((domain) => domain.id === form.baseDomainId) || null,
);

const normalizeLabel = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/_/g, "-")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "");

const buildFallbackHostname = () => {
  const label = normalizeLabel(form.fallbackLabel);
  if (!label || !selectedServerIPv4.value) {
    return "";
  }
  return `${label}.${selectedServerIPv4.value.replace(/\./g, "-")}.sslip.io`;
};

const buildBaseDomainHostname = () => {
  const label = normalizeLabel(form.userDomainLabel);
  if (!label || !selectedBaseDomain.value) {
    return "";
  }
  return `${label}.${selectedBaseDomain.value.hostname}`;
};

const canCreateSite = computed(() => {
  if (!form.serverId || !form.name.trim()) {
    return false;
  }
  if (form.hostnameSource === "fallback_resolver") {
    return Boolean(buildFallbackHostname());
  }
  if (form.userDomainMode === "base_domain") {
    return Boolean(buildBaseDomainHostname() && form.baseDomainId);
  }
  return Boolean(form.exactHostname.trim());
});

const loadPage = async () => {
  pageError.value = "";
  try {
    await Promise.all([fetchServers(), fetchSites(), fetchDomains()]);
    const requestedServer = route.query.serverId;
    if (typeof requestedServer === "string" && requestedServer.trim()) {
      form.serverId = requestedServer.trim();
    }
    if (!form.serverId && serverOptions.value[0]) {
      form.serverId = serverOptions.value[0].id;
    }
    if (!form.baseDomainId && userBaseDomains.value[0]) {
      form.baseDomainId = userBaseDomains.value[0].id;
    }
    if (!selectedServerIPv4.value) {
      form.hostnameSource = "user";
    }
  } catch (e: any) {
    pageError.value = e.message || "Failed to load sites";
  }
};

const resetForm = () => {
  form.name = "";
  form.status = "draft";
  form.wordpressPath = "/srv/www/";
  form.phpVersion = "8.3";
  form.wordpressVersion = "6.8";
  form.fallbackLabel = "";
  form.userDomainLabel = "";
  form.exactHostname = "";
  form.userDomainMode = "base_domain";
  form.baseDomainId = userBaseDomains.value[0]?.id || "";
  form.hostnameSource = selectedServerIPv4.value ? "fallback_resolver" : "user";
};

const handleCreateSite = async () => {
  successMessage.value = "";
  pageError.value = "";
  try {
    const created = await createSite({
      server_id: form.serverId,
      name: form.name,
      status: form.status,
      wordpress_path: form.wordpressPath || undefined,
      php_version: form.phpVersion || undefined,
      wordpress_version: form.wordpressVersion || undefined,
      primary_hostname_config:
        form.hostnameSource === "fallback_resolver"
          ? {
              source: "fallback_resolver",
              label: form.fallbackLabel,
            }
          : form.userDomainMode === "base_domain"
            ? {
                source: "user",
                label: form.userDomainLabel,
                domain_id: form.baseDomainId,
              }
            : {
                source: "user",
                hostname: form.exactHostname,
              },
    });
    successMessage.value = `Created ${created.name} on ${created.server_name} with ${created.primary_domain}.`;
    resetForm();
    await fetchSites();
  } catch (e: any) {
    pageError.value = e.message || "Failed to create site";
  }
};

const formatDate = (iso: string) => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return iso;
  }
};

onMounted(loadPage);

watch(
  () => route.query.serverId,
  (value) => {
    if (typeof value === "string" && value.trim()) {
      form.serverId = value.trim();
    }
  },
);

watch(selectedServerIPv4, (value) => {
  if (!value && form.hostnameSource === "fallback_resolver") {
    form.hostnameSource = "user";
  }
});

watch(userBaseDomains, (value) => {
  if (!value.some((domain) => domain.id === form.baseDomainId)) {
    form.baseDomainId = value[0]?.id || "";
  }
});
</script>

<template>
  <div class="space-y-8">
    <section
      class="relative overflow-hidden rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(255,255,255,0.95),rgba(226,244,240,0.88)_45%,rgba(237,233,254,0.82))] p-7 shadow-[0_32px_120px_-56px_rgba(18,95,84,0.45)] dark:bg-[linear-gradient(135deg,rgba(21,28,31,0.95),rgba(17,57,53,0.88)_50%,rgba(36,33,52,0.88))]"
    >
      <div class="absolute inset-y-0 right-0 hidden w-80 bg-[radial-gradient(circle_at_top,rgba(69,198,214,0.28),transparent_62%)] lg:block" />
      <div class="relative flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-4">
          <div class="inline-flex items-center gap-2 rounded-full border border-foreground/10 bg-background/70 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground backdrop-blur">
            Sites Inventory
          </div>
          <div>
            <h1 class="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
              Track each WordPress site with the hostname path it will really use.
            </h1>
            <p class="mt-3 max-w-2xl text-base leading-7 text-muted-foreground">
              Choose either a fallback resolver hostname for onboarding and evaluation, or a user-managed domain path that matches production reality.
            </p>
          </div>
        </div>

        <div class="grid grid-cols-3 gap-3 sm:min-w-[420px]">
          <div class="rounded-2xl border border-border/60 bg-background/75 px-4 py-4 backdrop-blur">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Total</p>
            <p class="mt-3 text-3xl font-semibold text-foreground">{{ siteStats.total }}</p>
          </div>
          <div class="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-primary/80">Active</p>
            <p class="mt-3 text-3xl font-semibold text-primary">{{ siteStats.active }}</p>
          </div>
          <div class="rounded-2xl border border-accent/20 bg-accent/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-accent/80">Attention</p>
            <p class="mt-3 text-3xl font-semibold text-accent">{{ siteStats.attention }}</p>
          </div>
        </div>
      </div>
    </section>

    <Alert v-if="pageError || error" class="border-destructive/30 bg-destructive/10 text-destructive">
      <AlertDescription>{{ pageError || error }}</AlertDescription>
    </Alert>

    <Alert v-if="successMessage" class="border-primary/30 bg-primary/10 text-primary">
      <AlertDescription>{{ successMessage }}</AlertDescription>
    </Alert>

    <div class="grid gap-6 xl:grid-cols-[1.4fr_minmax(360px,0.9fr)]">
      <Card class="overflow-hidden rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
        <CardHeader class="border-b border-border/50 px-6 py-5">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Inventory</p>
              <h2 class="mt-1 text-xl font-semibold text-foreground">Managed sites</h2>
            </div>
            <Badge variant="outline" class="border-border/60 bg-muted/50 text-xs text-muted-foreground">
              {{ siteStats.draft }} drafts
            </Badge>
          </div>
        </CardHeader>
        <CardContent class="px-6 py-5">
          <div v-if="loading" class="flex items-center justify-center py-16 text-sm text-muted-foreground">Loading sites...</div>

          <div v-else-if="sites.length === 0" class="rounded-3xl border border-dashed border-border/60 bg-muted/20 px-6 py-16 text-center">
            <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-background/80 shadow-sm">
              <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6">
                <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9" />
              </svg>
            </div>
            <h3 class="mt-5 text-xl font-semibold text-foreground">No sites tracked yet</h3>
            <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-muted-foreground">
              Start with one site record and choose a hostname source that matches how you plan to bring it online.
            </p>
          </div>

          <div v-else class="space-y-4">
            <NuxtLink
              v-for="site in sites"
              :key="site.id"
              :to="`/sites/${site.id}`"
              class="group block rounded-[22px] border border-border/60 bg-background/70 px-5 py-4 transition-transform duration-200 hover:-translate-y-0.5 hover:border-accent/40 hover:shadow-[0_18px_50px_-28px_rgba(21,95,84,0.4)]"
            >
              <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="truncate text-lg font-semibold text-foreground">{{ site.name }}</h3>
                    <Badge variant="outline" :class="siteStatusMeta(site.status).className">
                      {{ siteStatusMeta(site.status).label }}
                    </Badge>
                    <Badge variant="outline" :class="siteDeploymentMeta(site.deployment_state).className">
                      {{ siteDeploymentMeta(site.deployment_state).label }}
                    </Badge>
                  </div>
                  <p class="mt-1 text-sm text-muted-foreground">{{ site.primary_domain || "No primary hostname yet" }}</p>
                  <p v-if="site.deployment_status_message" class="mt-2 text-sm text-muted-foreground">
                    {{ site.deployment_status_message }}
                  </p>
                  <div class="mt-3 flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <span class="rounded-full border border-border/60 bg-muted/40 px-2.5 py-1">{{ site.server_name }}</span>
                    <span class="rounded-full border border-border/60 bg-muted/40 px-2.5 py-1">PHP {{ site.php_version || "TBD" }}</span>
                    <span class="rounded-full border border-border/60 bg-muted/40 px-2.5 py-1">WordPress {{ site.wordpress_version || "TBD" }}</span>
                    <span v-if="site.wordpress_path" class="rounded-full border border-border/60 bg-muted/40 px-2.5 py-1">{{ site.wordpress_path }}</span>
                  </div>
                </div>

                <div class="shrink-0 text-sm text-muted-foreground lg:text-right">
                  <p class="font-medium text-foreground">Open site record</p>
                  <p class="mt-1">Created {{ formatDate(site.created_at) }}</p>
                </div>
              </div>
            </NuxtLink>
          </div>
        </CardContent>
      </Card>

      <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
        <CardHeader class="border-b border-border/50 px-6 py-5">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">New Site</p>
          <h2 class="mt-1 text-xl font-semibold text-foreground">Create a site with its primary hostname</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">
            Capture the site record first, then choose whether it starts on a fallback resolver hostname or a user-managed domain path.
          </p>
        </CardHeader>
        <CardContent class="px-6 py-5">
          <form class="space-y-4" @submit.prevent="handleCreateSite">
            <div class="space-y-1.5">
              <Label for="site-server" class="text-sm font-medium text-muted-foreground">Current server</Label>
              <select
                id="site-server"
                v-model="form.serverId"
                class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
              >
                <option v-for="server in serverOptions" :key="server.id" :value="server.id">{{ server.name }}</option>
              </select>
              <p class="text-xs text-muted-foreground">Attached to {{ selectedServerName }} today, with room for future moves and routing changes.</p>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div class="space-y-1.5 sm:col-span-2">
                <Label for="site-name" class="text-sm font-medium text-muted-foreground">Site name</Label>
                <Input id="site-name" v-model="form.name" placeholder="e.g. Northwind Marketing" />
              </div>
              <div class="space-y-1.5">
                <Label for="site-status" class="text-sm font-medium text-muted-foreground">Lifecycle state</Label>
                <select
                  id="site-status"
                  v-model="form.status"
                  class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                >
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
                  <option value="attention">Attention</option>
                  <option value="archived">Archived</option>
                </select>
              </div>
              <div class="space-y-1.5 sm:col-span-2">
                <Label for="site-path" class="text-sm font-medium text-muted-foreground">WordPress path</Label>
                <Input id="site-path" v-model="form.wordpressPath" placeholder="/srv/www/client/current" />
              </div>
              <div class="space-y-1.5">
                <Label for="site-php" class="text-sm font-medium text-muted-foreground">PHP version</Label>
                <Input id="site-php" v-model="form.phpVersion" placeholder="8.3" />
              </div>
              <div class="space-y-1.5">
                <Label for="site-wp" class="text-sm font-medium text-muted-foreground">WordPress version</Label>
                <Input id="site-wp" v-model="form.wordpressVersion" placeholder="6.8" />
              </div>
            </div>

            <div class="space-y-4 rounded-2xl border border-border/60 bg-muted/25 p-4">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Primary Hostname</p>

              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-muted-foreground">Hostname source</Label>
                <select
                  v-model="form.hostnameSource"
                  class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                >
                  <option value="fallback_resolver" :disabled="!selectedServerIPv4">Fallback resolver hostname (sslip.io-style)</option>
                  <option value="user">User-added domain/base domain</option>
                </select>
              </div>

              <template v-if="form.hostnameSource === 'fallback_resolver'">
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">Hostname label</Label>
                  <Input v-model="form.fallbackLabel" placeholder="northwind-preview" />
                </div>
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">Result</Label>
                  <Input :model-value="buildFallbackHostname()" readonly placeholder="Select a server with IPv4 and enter a label" />
                </div>
                <Alert class="border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-200">
                  <AlertDescription>
                    Fallback resolver hostnames are useful for onboarding, development, and evaluation, but they are not recommended for production.
                  </AlertDescription>
                </Alert>
              </template>

              <template v-else>
                <div class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">User domain path</Label>
                  <select
                    v-model="form.userDomainMode"
                    class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                  >
                    <option value="base_domain">Mint from a reusable base domain</option>
                    <option value="hostname">Use an exact manual hostname</option>
                  </select>
                </div>

                <div v-if="form.userDomainMode === 'base_domain'" class="grid gap-4 sm:grid-cols-2">
                  <div class="space-y-1.5">
                    <Label class="text-sm font-medium text-muted-foreground">Base domain</Label>
                    <select
                      v-model="form.baseDomainId"
                      class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                    >
                      <option v-for="domain in userBaseDomains" :key="domain.id" :value="domain.id">
                        {{ domain.hostname }} ({{ domain.dns_state }})
                      </option>
                    </select>
                  </div>
                  <div class="space-y-1.5">
                    <Label class="text-sm font-medium text-muted-foreground">Hostname label</Label>
                    <Input v-model="form.userDomainLabel" placeholder="northwind-live" />
                  </div>
                  <div class="space-y-1.5 sm:col-span-2">
                    <Label class="text-sm font-medium text-muted-foreground">Result</Label>
                    <Input :model-value="buildBaseDomainHostname()" readonly placeholder="Choose a base domain and enter a label" />
                  </div>
                  <p class="sm:col-span-2 text-sm text-muted-foreground">
                    Reusable base domains are first-class inventory records. Wildcard-ready domains can mint child hostnames later, while still keeping DNS readiness visible.
                  </p>
                </div>

                <div v-else class="space-y-1.5">
                  <Label class="text-sm font-medium text-muted-foreground">Exact hostname</Label>
                  <Input v-model="form.exactHostname" placeholder="www.client-example.com" />
                  <p class="text-sm text-muted-foreground">
                    Use this when the site needs a manually managed concrete hostname instead of a reusable base domain.
                  </p>
                </div>
              </template>
            </div>

            <Button type="submit" class="w-full rounded-xl bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving || !canCreateSite">
              {{ saving ? "Creating site..." : "Create and deploy site" }}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
