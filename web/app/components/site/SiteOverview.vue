<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { StoredSite, SiteHealthResponse } from "~/composables/useSites";
import type { StoredDomain } from "~/composables/useDomains";
import type { Activity } from "~/composables/useActivity";
import { useNotImplemented } from "~/composables/useNotImplemented";
import { formatSiteDate, formatRelativeTime } from "~/composables/useSiteHelpers";

const props = defineProps<{
  site: StoredSite;
  siteHealth: SiteHealthResponse | null;
  siteDomains: StoredDomain[];
  activities: Activity[];
  serverName: string;
  serverId: string;
}>();

const { trigger: triggerNotImplemented } = useNotImplemented();

const router = useRouter();
const route = useRoute();

const overallHealthy = computed(() => {
  return props.site.runtime_health_state === "healthy" && props.site.deployment_state === "ready";
});

const healthChecks = computed(() => {
  return props.siteHealth?.snapshot?.checks || [];
});

const healthServices = computed(() => {
  return props.siteHealth?.snapshot?.services || [];
});

const redisService = computed(() => {
  return healthServices.value.find((s) => s.name === "redis-server");
});

const recentActivities = computed(() => {
  return props.activities.slice(0, 5);
});

const goToActivity = () => {
  router.push({ query: { ...route.query, tab: "activity" } });
};

const activityLevelClass = (level: string) => {
  switch (level) {
    case "error":
      return "border-destructive/30 bg-destructive/10 text-destructive";
    case "warning":
      return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200";
    default:
      return "border-border/60 bg-muted/50 text-muted-foreground";
  }
};
</script>

<template>
  <div class="space-y-4">
    <!-- Top row: Site Health + Updates -->
    <div class="grid gap-4 md:grid-cols-2">

      <!-- Site Health -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center justify-between">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Site Health</p>
          <span v-if="siteHealth" :class="overallHealthy ? 'text-primary' : 'text-amber-500'">
            <svg v-if="overallHealthy" xmlns="http://www.w3.org/2000/svg" class="size-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" /></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" /></svg>
          </span>
        </div>

        <div v-if="healthChecks.length" class="mt-4 space-y-2">
          <div
            v-for="check in healthChecks"
            :key="check.name"
            class="flex items-center justify-between gap-3 text-sm"
          >
            <span class="text-foreground">{{ check.name }}</span>
            <span class="flex items-center gap-1.5" :class="check.ok ? 'text-primary' : 'text-destructive'">
              <svg v-if="check.ok" xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
              {{ check.ok ? "OK" : (check.detail || "Issue") }}
            </span>
          </div>
        </div>

        <div v-else-if="siteHealth?.agent_connected" class="mt-4 text-sm text-muted-foreground">
          Agent connected. Waiting for health snapshot...
        </div>

        <div v-else class="mt-4 space-y-2">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Agent connected</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <p class="text-xs text-muted-foreground">
            Health checks require an active agent connection.
          </p>
        </div>

        <div v-if="healthServices.length" class="mt-4 flex flex-wrap gap-1.5">
          <span
            v-for="service in healthServices"
            :key="service.name"
            :class="[
              'rounded-full border px-2 py-0.5 text-xs',
              service.active_state === 'active'
                ? 'border-primary/20 bg-primary/5 text-primary'
                : 'border-destructive/20 bg-destructive/5 text-destructive',
            ]"
          >
            {{ service.name }}
          </span>
        </div>

        <p v-if="siteHealth?.last_health_check_at" class="mt-3 text-xs text-muted-foreground">
          Checked {{ formatRelativeTime(siteHealth.last_health_check_at) }}
        </p>
      </div>

      <!-- Updates -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Updates</p>

        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">WordPress Core</span>
            <span class="text-muted-foreground">{{ site.wordpress_version || "--" }}</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Plugin updates</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Theme updates</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>

        <div class="mt-5 flex gap-2">
          <Button size="sm" variant="outline" @click="triggerNotImplemented">Update All</Button>
          <Button size="sm" variant="ghost" @click="triggerNotImplemented">View Updates</Button>
        </div>
      </div>
    </div>

    <!-- Second row: Backups + Performance -->
    <div class="grid gap-4 md:grid-cols-2">

      <!-- Backups -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Backups</p>

        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Last backup</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Next scheduled</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>

        <div class="mt-5 flex gap-2">
          <Button size="sm" variant="outline" @click="triggerNotImplemented">Create Backup</Button>
          <Button size="sm" variant="ghost" @click="triggerNotImplemented">Restore</Button>
        </div>
      </div>

      <!-- Performance -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Performance</p>

        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Page cache</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Redis</span>
            <span v-if="redisService" :class="redisService.active_state === 'active' ? 'text-primary' : 'text-destructive'">
              {{ redisService.active_state === "active" ? "Active" : redisService.active_state }}
            </span>
            <span v-else class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">CDN</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>

        <div class="mt-5">
          <Button size="sm" variant="outline" @click="triggerNotImplemented">Clear Cache</Button>
        </div>
      </div>
    </div>

    <!-- Full-width: Recent Activity -->
    <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
      <div class="flex items-center justify-between">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent Activity</p>
        <button
          v-if="activities.length > 0"
          class="text-xs font-medium text-accent transition hover:text-accent/80"
          @click="goToActivity"
        >
          View all
        </button>
      </div>

      <div v-if="recentActivities.length === 0" class="mt-4 text-center text-sm text-muted-foreground py-4">
        No site activity recorded yet.
      </div>

      <div v-else class="mt-4 space-y-2">
        <div
          v-for="activity in recentActivities"
          :key="activity.id"
          class="flex items-start justify-between gap-3 rounded-xl border border-border/50 bg-muted/10 px-3.5 py-2.5"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="truncate text-sm font-medium text-foreground">{{ activity.title }}</p>
              <Badge variant="outline" :class="[activityLevelClass(activity.level), 'text-[10px] shrink-0']">{{ activity.level }}</Badge>
            </div>
            <p v-if="activity.message" class="mt-0.5 truncate text-xs text-muted-foreground">{{ activity.message }}</p>
          </div>
          <span class="shrink-0 text-xs text-muted-foreground">{{ formatRelativeTime(activity.created_at) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
