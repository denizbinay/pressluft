<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import type { StoredSite, SiteHealthResponse } from "~/composables/useSites";

const props = defineProps<{
  site: StoredSite;
  siteHealth: SiteHealthResponse | null;
  serverName: string;
  serverId: string;
}>();

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
</script>

<template>
  <div class="space-y-6">
    <div class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
      <div class="space-y-4">
        <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Deployment</p>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <Badge variant="outline" :class="siteDeploymentMeta(site.deployment_state).className">{{ siteDeploymentMeta(site.deployment_state).label }}</Badge>
            <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
            <span class="text-sm text-muted-foreground">{{ site.deployment_status_message || "Waiting for deployment activity." }}</span>
          </div>
          <p v-if="site.last_deployed_at" class="mt-3 text-xs text-muted-foreground">Last deployed {{ formatDate(site.last_deployed_at) }}</p>
        </div>

        <div class="rounded-2xl border border-border/60 bg-background/70 p-4">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Runtime</p>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
            <span class="text-sm text-muted-foreground">{{ site.runtime_health_status_message || "Waiting for the first runtime health check." }}</span>
          </div>
          <p v-if="site.last_health_check_at" class="mt-3 text-xs text-muted-foreground">Last checked {{ formatDate(site.last_health_check_at) }}</p>
        </div>
      </div>

      <div class="space-y-4">
        <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Placement</p>
          <p class="mt-2 text-lg font-semibold text-foreground">{{ serverName }}</p>
          <p class="mt-1 text-sm text-muted-foreground">
            Sites stay top-level resources, but the current hosting target remains explicit while deployment is still server-bound.
          </p>
          <NuxtLink :to="`/servers/${serverId}?tab=sites`" class="mt-4 inline-flex text-sm font-medium text-accent transition hover:text-accent/80">Open server view</NuxtLink>
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
      </div>
    </div>

    <div v-if="siteHealth?.snapshot" class="rounded-2xl border border-border/60 bg-background/70 p-4">
      <p class="text-sm font-semibold text-foreground">{{ siteHealth.snapshot.summary }}</p>
      <p class="mt-1 text-xs text-muted-foreground">Agent snapshot captured {{ formatDate(siteHealth.snapshot.generated_at) }}</p>
      <div v-if="siteHealth.snapshot.checks?.length" class="mt-4 grid gap-2 md:grid-cols-2">
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
  </div>
</template>
