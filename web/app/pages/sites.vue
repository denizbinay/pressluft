<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useServers } from "~/composables/useServers";
import { useSites, type StoredSite } from "~/composables/useSites";

const route = useRoute();
const router = useRouter();

const { servers, fetchServers } = useServers();
const { sites, loading, error, fetchSites } = useSites();

const pageError = ref("");
const successMessage = ref("");
const createDialogOpen = ref(false);

const initialServerId = computed(() => {
  const value = route.query.serverId;
  return typeof value === "string" ? value.trim() : "";
});

const deployableServers = computed(() =>
  servers.value.filter(
    (server) =>
      server.status === "ready" &&
      server.setup_state === "ready" &&
      server.profile_key === "nginx-stack",
  ),
);

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

const siteRuntimeMeta = (state: StoredSite["runtime_health_state"]) => {
  switch (state) {
    case "healthy":
      return { label: "Healthy", className: "border-primary/30 bg-primary/10 text-primary" };
    case "issue":
      return { label: "Runtime issue", className: "border-destructive/30 bg-destructive/10 text-destructive" };
    case "unknown":
      return { label: "Unknown", className: "border-border/60 bg-muted/70 text-muted-foreground" };
    default:
      return { label: "Checking", className: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200" };
  }
};

const siteStats = computed(() => {
  const items = sites.value;
  return {
    total: items.length,
    live: items.filter((site) => site.deployment_state === "ready").length,
    deploying: items.filter((site) => site.deployment_state === "deploying").length,
    needsAttention: items.filter((site) => site.deployment_state === "failed" || site.runtime_health_state === "issue" || site.status === "attention").length,
  };
});

const loadPage = async () => {
  pageError.value = "";
  try {
    await Promise.all([fetchServers(), fetchSites()]);
  } catch (e: any) {
    pageError.value = e.message || "Failed to load sites";
  }
};

const handleCreated = async (site: StoredSite) => {
  successMessage.value = `Created ${site.name}. Deployment is now running.`;
  await fetchSites();
  await router.push(`/sites/${site.id}`);
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
            Sites
          </div>
          <div>
            <h1 class="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
              Create WordPress sites without exposing the deploy contract.
            </h1>
            <p class="mt-3 max-w-2xl text-base leading-7 text-muted-foreground">
              Start with a site name, a ready server, and either a preview URL or the real hostname. Pressluft handles the rest.
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <Button class="rounded-xl bg-accent text-accent-foreground hover:bg-accent/85" @click="createDialogOpen = true">
              Create Site
            </Button>
            <p class="self-center text-sm text-muted-foreground">
              {{ deployableServers.length }} ready {{ deployableServers.length === 1 ? "server" : "servers" }} available for deployment
            </p>
          </div>
        </div>

        <div class="grid grid-cols-3 gap-3 sm:min-w-[420px]">
          <div class="rounded-2xl border border-border/60 bg-background/75 px-4 py-4 backdrop-blur">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Total</p>
            <p class="mt-3 text-3xl font-semibold text-foreground">{{ siteStats.total }}</p>
          </div>
          <div class="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-primary/80">Live</p>
            <p class="mt-3 text-3xl font-semibold text-primary">{{ siteStats.live }}</p>
          </div>
          <div class="rounded-2xl border border-accent/20 bg-accent/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-accent/80">Deploying</p>
            <p class="mt-3 text-3xl font-semibold text-accent">{{ siteStats.deploying }}</p>
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

    <Card class="overflow-hidden rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
      <CardHeader class="border-b border-border/50 px-6 py-5">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Inventory</p>
            <h2 class="mt-1 text-xl font-semibold text-foreground">Managed sites</h2>
          </div>
          <Badge variant="outline" class="border-border/60 bg-muted/50 text-xs text-muted-foreground">
            {{ siteStats.needsAttention }} need attention
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
            Create the first managed WordPress site with a preview URL or a real hostname.
          </p>
          <Button class="mt-6 rounded-xl bg-accent text-accent-foreground hover:bg-accent/85" @click="createDialogOpen = true">
            Create Site
          </Button>
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
                  <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">
                    {{ siteRuntimeMeta(site.runtime_health_state).label }}
                  </Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">{{ site.primary_domain || "No primary hostname yet" }}</p>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ site.runtime_health_status_message || site.deployment_status_message || "Waiting for the first runtime health check." }}
                </p>
                <div class="mt-3 flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <span class="rounded-full border border-border/60 bg-muted/40 px-2.5 py-1">{{ site.server_name }}</span>
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

    <CreateSiteDialog
      v-model:open="createDialogOpen"
      :servers="servers"
      :initial-server-id="initialServerId"
      @created="handleCreated"
    />
  </div>
</template>
