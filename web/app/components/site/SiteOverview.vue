<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { StoredSite, SiteHealthResponse } from "~/composables/useSites";
import type { StoredDomain } from "~/composables/useDomains";
import type { Activity } from "~/composables/useActivity";
import { useNotImplemented } from "~/composables/useNotImplemented";
import { formatRelativeTime } from "~/composables/useSiteHelpers";

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

// ── Health card logic ────────────────────────────────────────

const healthChecks = computed(() => props.siteHealth?.snapshot?.checks || []);
const healthServices = computed(() => props.siteHealth?.snapshot?.services || []);
const recentErrors = computed(() => props.siteHealth?.snapshot?.recent_errors || []);

const findCheck = (name: string) => healthChecks.value.find((c) => c.name === name);

const availabilityChecks = computed(() => [
  { label: "Website reachable", check: findCheck("home-page") },
  { label: "Login page reachable", check: findCheck("login-page") },
]);

const securityChecks = computed(() => [
  { label: "SSL certificate valid", check: undefined as ReturnType<typeof findCheck> },
  { label: "Security scan passed", check: undefined as ReturnType<typeof findCheck> },
]);

const updateChecks = computed(() => [
  { label: "WordPress core", value: props.site.wordpress_version || undefined },
  { label: "Plugins up to date", check: undefined as ReturnType<typeof findCheck> },
  { label: "Themes up to date", check: undefined as ReturnType<typeof findCheck> },
]);

const protectionChecks = computed(() => [
  { label: "Automatic backups configured", check: undefined as ReturnType<typeof findCheck> },
  { label: "Malware protection active", check: undefined as ReturnType<typeof findCheck> },
]);

/** Health score: percentage of checks with real data that are passing */
const healthScore = computed(() => {
  if (!props.siteHealth?.snapshot) return null;

  const checks = healthChecks.value;
  if (checks.length === 0) return null;

  const passing = checks.filter((c) => c.ok).length;
  return Math.round((passing / checks.length) * 100);
});

const overallHealthy = computed(() => {
  return props.site.runtime_health_state === "healthy" && props.site.deployment_state === "ready";
});

// ── Performance card logic ───────────────────────────────────

const redisService = computed(() => healthServices.value.find((s) => s.name === "redis-server"));

// ── Activity card logic ──────────────────────────────────────

const recentActivities = computed(() => props.activities.slice(0, 5));

const goToActivity = () => {
  router.push({ query: { ...route.query, tab: "activity" } });
};

const goToLogs = () => {
  router.push({ query: { ...route.query, tab: "logs" } });
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
    <!--
      Main grid layout:
      Left column: Site Health (spans 2 rows)
      Right column row 1: Traffic
      Right column row 2: Backups
    -->
    <div class="grid gap-4 md:grid-cols-2 md:grid-rows-[auto_auto]">

      <!-- ═══ Site Health (spans 2 rows) ═══ -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5 md:row-span-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
            <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Site Health</p>
          </div>
          <div v-if="healthScore !== null" class="flex items-center gap-2">
            <span class="text-lg font-semibold" :class="healthScore >= 80 ? 'text-primary' : healthScore >= 50 ? 'text-amber-500' : 'text-destructive'">{{ healthScore }}%</span>
            <span v-if="siteHealth" :class="overallHealthy ? 'text-primary' : 'text-amber-500'">
              <svg v-if="overallHealthy" xmlns="http://www.w3.org/2000/svg" class="size-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" /></svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" /></svg>
            </span>
          </div>
        </div>

        <!-- Availability -->
        <div class="mt-5">
          <p class="text-xs font-medium text-muted-foreground">Availability</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in availabilityChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                <svg v-if="item.check.ok" xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <!-- Security -->
        <div class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">Security</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in securityChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                <svg v-if="item.check.ok" xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <!-- Updates -->
        <div class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">Updates</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in updateChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.value" class="text-muted-foreground">{{ item.value }}</span>
              <span v-else-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <!-- Protection -->
        <div class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">Protection</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in protectionChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <!-- Services -->
        <div v-if="healthServices.length" class="mt-5 flex flex-wrap gap-1.5">
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

        <!-- No agent state -->
        <div v-if="!siteHealth?.snapshot && !siteHealth?.agent_connected" class="mt-4 rounded-xl border border-border/40 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground">
          Health checks require an active agent connection.
        </div>
        <div v-else-if="siteHealth?.agent_connected && !siteHealth?.snapshot" class="mt-4 rounded-xl border border-border/40 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground">
          Agent connected. Waiting for health snapshot...
        </div>

        <p v-if="siteHealth?.last_health_check_at" class="mt-3 text-xs text-muted-foreground">
          Checked {{ formatRelativeTime(siteHealth.last_health_check_at) }}
        </p>
      </div>

      <!-- ═══ Traffic ═══ -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Traffic</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Requests (24h)</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Bandwidth</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Avg response time</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Server errors (5xx)</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>
      </div>

      <!-- ═══ Backups ═══ -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Backups</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Automatic backups</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Frequency</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Last backup</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Storage</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>
        <div class="mt-5">
          <Button size="sm" variant="outline" @click="triggerNotImplemented">Create Backup</Button>
        </div>
      </div>
    </div>

    <!-- Row 2: Performance + Environment -->
    <div class="grid gap-4 md:grid-cols-2">

      <!-- ═══ Performance ═══ -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Performance</p>
        </div>
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
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">PHP workers</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Avg response time</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>
      </div>

      <!-- ═══ Environment ═══ -->
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Environment</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">PHP version</span>
            <span class="font-medium text-foreground">{{ site.php_version || "--" }}</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">WordPress version</span>
            <span class="font-medium text-foreground">{{ site.wordpress_version || "--" }}</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Server</span>
            <span class="font-medium text-foreground">{{ serverName || "--" }}</span>
          </div>
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
      </div>
    </div>

    <!-- Full-width: Recent Logs -->
    <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent Logs</p>
        </div>
        <button
          class="text-xs font-medium text-accent transition hover:text-accent/80"
          @click="goToLogs"
        >
          View all
        </button>
      </div>

      <div v-if="recentErrors.length === 0" class="mt-4 py-4 text-center text-sm text-muted-foreground">
        No recent errors.
      </div>

      <div v-else class="mt-4 space-y-1.5">
        <div
          v-for="(error, index) in recentErrors.slice(0, 4)"
          :key="index"
          class="rounded-lg border border-border/40 bg-muted/10 px-3 py-2 font-mono text-xs text-muted-foreground leading-relaxed break-all"
        >
          {{ error }}
        </div>
      </div>
    </div>

    <!-- Full-width: Recent Activity -->
    <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent Activity</p>
        </div>
        <button
          v-if="activities.length > 0"
          class="text-xs font-medium text-accent transition hover:text-accent/80"
          @click="goToActivity"
        >
          View all
        </button>
      </div>

      <div v-if="recentActivities.length === 0" class="mt-4 py-4 text-center text-sm text-muted-foreground">
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
