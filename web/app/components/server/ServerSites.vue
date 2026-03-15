<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { useSites, type StoredSite } from "~/composables/useSites";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  serverId: string;
}>();

const { fetchServerSites } = useSites();

const serverSites = ref<StoredSite[]>([]);
const sitesLoading = ref(false);
const sitesError = ref("");

const loadServerSites = async () => {
  if (!props.serverId) return;
  sitesLoading.value = true;
  sitesError.value = "";
  try {
    serverSites.value = await fetchServerSites(props.serverId);
  } catch (e: unknown) {
    sitesError.value = errorMessage(e) || "Failed to load sites";
  } finally {
    sitesLoading.value = false;
  }
};

const siteStatusClass = (status: StoredSite["status"]) => {
  switch (status) {
    case "active":
      return "border-primary/30 bg-primary/10 text-primary";
    case "attention":
      return "border-accent/30 bg-accent/10 text-accent";
    case "archived":
      return "border-border/60 bg-muted/70 text-muted-foreground";
    default:
      return "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300";
  }
};

const siteDeploymentClass = (state: StoredSite["deployment_state"]) => {
  switch (state) {
    case "ready":
      return "border-primary/30 bg-primary/10 text-primary";
    case "failed":
      return "border-destructive/30 bg-destructive/10 text-destructive";
    case "deploying":
      return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200";
    default:
      return "border-border/60 bg-muted/70 text-muted-foreground";
  }
};

const siteRuntimeClass = (state: StoredSite["runtime_health_state"]) => {
  switch (state) {
    case "healthy":
      return "border-primary/30 bg-primary/10 text-primary";
    case "issue":
      return "border-destructive/30 bg-destructive/10 text-destructive";
    case "unknown":
      return "border-border/60 bg-muted/70 text-muted-foreground";
    default:
      return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200";
  }
};

onMounted(() => {
  loadServerSites();
});
</script>

<template>
  <div class="space-y-4">
    <div
      class="flex flex-col gap-3 rounded-2xl border border-border/60 bg-[linear-gradient(135deg,rgba(255,255,255,0.95),rgba(224,246,241,0.88)_48%,rgba(237,239,255,0.86))] px-5 py-5 dark:bg-[linear-gradient(135deg,rgba(23,27,33,0.94),rgba(17,52,49,0.88)_50%,rgba(29,34,51,0.9))] lg:flex-row lg:items-center lg:justify-between"
    >
      <div>
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          Site inventory
        </p>
        <h3 class="mt-1 text-lg font-semibold text-foreground">
          WordPress workloads attached to this server
        </h3>
        <p class="mt-2 max-w-2xl text-sm text-muted-foreground">
          Sites live as top-level entities, but this view keeps the
          hosting relationship obvious when you are working from a
          server record.
        </p>
      </div>
      <p class="text-sm text-muted-foreground">
        Create new sites from the top-level `Sites` page, then track their deployment back to this server here.
      </p>
    </div>

    <div
      v-if="sitesError"
      class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ sitesError }}
    </div>

    <div
      v-else-if="sitesLoading"
      class="rounded-lg border border-border/50 px-4 py-8 text-center text-sm text-muted-foreground"
    >
      Loading sites...
    </div>

    <div
      v-else-if="serverSites.length === 0"
      class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center"
    >
      <h3 class="text-sm font-medium text-foreground">
        No sites attached yet
      </h3>
      <p class="mt-1 text-sm text-muted-foreground">
        Use the sites surface to create the first record mapped to
        this server.
      </p>
    </div>

    <div v-else class="grid gap-4 lg:grid-cols-2">
      <NuxtLink
        v-for="site in serverSites"
        :key="site.id"
        :to="`/sites/${site.id}`"
        class="rounded-2xl border border-border/60 bg-card/50 px-4 py-4 transition hover:-translate-y-0.5 hover:border-accent/40 hover:shadow-[0_18px_40px_-30px_rgba(22,92,85,0.45)]"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate text-sm font-semibold text-foreground">
                {{ site.name }}
              </h4>
              <Badge variant="outline" :class="siteStatusClass(site.status)">
                {{ site.status }}
              </Badge>
              <Badge variant="outline" :class="siteDeploymentClass(site.deployment_state)">
                {{ site.deployment_state }}
              </Badge>
              <Badge variant="outline" :class="siteRuntimeClass(site.runtime_health_state)">
                {{ site.runtime_health_state }}
              </Badge>
            </div>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ site.primary_domain || "No primary hostname recorded" }}
            </p>
            <p v-if="site.deployment_status_message" class="mt-2 text-xs text-muted-foreground">
              {{ site.deployment_status_message }}
            </p>
            <p v-if="site.runtime_health_status_message" class="mt-1 text-xs text-muted-foreground">
              {{ site.runtime_health_status_message }}
            </p>
          </div>
          <svg
            class="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M9 5l7 7-7 7"
            />
          </svg>
        </div>
        <p class="mt-3 text-xs text-muted-foreground">
          Managed WordPress install tracked from the top-level Sites inventory.
        </p>
      </NuxtLink>
    </div>
  </div>
</template>
