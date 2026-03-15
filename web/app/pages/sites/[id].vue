<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import JobTimeline from "~/components/JobTimeline.vue";
import NotImplementedDialog from "~/components/NotImplementedDialog.vue";
import SiteOverview from "~/components/site/SiteOverview.vue";
import SiteHostnames from "~/components/site/SiteHostnames.vue";
import SiteSettings from "~/components/site/SiteSettings.vue";
import { useActivity } from "~/composables/useActivity";
import { useDomains, type StoredDomain } from "~/composables/useDomains";
import { useNotImplemented } from "~/composables/useNotImplemented";
import { useServers } from "~/composables/useServers";
import { useSites, type SiteHealthResponse, type StoredSite } from "~/composables/useSites";
import { siteDeploymentMeta, siteRuntimeMeta, formatSiteDate } from "~/composables/useSiteHelpers";
import { errorMessage } from "~/lib/utils";

interface SiteSection {
  key: string;
  label: string;
  icon: string;
  description: string;
}

const sections: SiteSection[] = [
  {
    key: "overview",
    label: "Overview",
    icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6",
    description: "Deployment truth, runtime health, and current hosting context",
  },
  {
    key: "hostnames",
    label: "Hostnames",
    icon: "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9",
    description: "Primary and additional hostnames attached to this site",
  },
  {
    key: "settings",
    label: "Settings",
    icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826 3.31 2.37 2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
    description: "Editable site details and managed defaults for this install",
  },
  {
    key: "activity",
    label: "Activity",
    icon: "M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z",
    description: "Recent site activity and the latest deployment timeline",
  },
];

const route = useRoute();
const router = useRouter();

const siteId = computed(() => {
  const raw = route.params.id;
  return typeof raw === "string" ? raw.trim() : "";
});

const activeSection = computed(() => {
  const tab = typeof route.query.tab === "string" ? route.query.tab : "";
  return sections.some((section) => section.key === tab) ? tab : "overview";
});

const currentSection = computed(
  () => sections.find((section) => section.key === activeSection.value) || sections[0],
);

const isMobileSidebarOpen = ref(false);

const navigateTo = (key: string) => {
  router.push({ query: { ...route.query, tab: key } });
};

const toggleMobileSidebar = () => {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value;
};

const selectSection = (key: string) => {
  navigateTo(key);
  isMobileSidebarOpen.value = false;
};

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

const { trigger: triggerNotImplemented } = useNotImplemented();

const currentServer = computed(() => servers.value.find((server) => server.id === site.value?.server_id) || null);
const currentServerIPv4 = computed(() => currentServer.value?.ipv4 || "");
const serverLocation = computed(() => currentServer.value?.location || "");
const serverProfile = computed(() => currentServer.value?.profile_key || "");

const canOpenSite = computed(() => !!site.value?.primary_domain && site.value.deployment_state === "ready");

const openSite = () => {
  if (site.value?.primary_domain) {
    window.open(`https://${site.value.primary_domain}`, "_blank");
  }
};

const settingsName = computed(() => site.value?.name || "");
const settingsEmail = computed(() => site.value?.wordpress_admin_email || "");

const formatDate = formatSiteDate;

const loadActivity = async () => {
  if (!siteId.value) return;
  try {
    await listSiteActivity(siteId.value, { limit: 8 });
  } catch {
    // Timeline degrades quietly on detail page.
  }
};

const refreshDomains = async () => {
  if (!siteId.value) return;
  siteDomains.value = await fetchSiteDomains(siteId.value);
};

const refreshHealth = async () => {
  if (!siteId.value) return;
  try {
    siteHealth.value = await fetchSiteHealth(siteId.value);
  } catch {
    siteHealth.value = null;
  }
};

const refreshSite = async () => {
  if (!siteId.value) return;
  const loadedSite = await fetchSite(siteId.value);
  site.value = loadedSite;
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
    await Promise.all([loadActivity(), refreshDomains(), refreshHealth()]);
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to load site";
  } finally {
    loading.value = false;
  }
};

const handleSave = async (payload: { name: string; wordpressAdminEmail: string }) => {
  if (!siteId.value) return;
  successMessage.value = "";
  pageError.value = "";
  try {
    const updated = await updateSite(siteId.value, {
      name: payload.name,
      wordpress_admin_email: payload.wordpressAdminEmail,
    });
    site.value = updated;
    await Promise.all([refreshDomains(), refreshHealth(), loadActivity()]);
    successMessage.value = "Site details updated.";
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to update site";
  }
};

const handleAssignHostname = async (payload: { hostname: string; source: string; is_primary: boolean }) => {
  if (!siteId.value) return;
  pageError.value = "";
  successMessage.value = "";
  try {
    const created = await createSiteDomain(siteId.value, payload);
    successMessage.value = `Assigned ${created.hostname}.`;
    await Promise.all([refreshDomains(), refreshSite(), refreshHealth(), loadActivity()]);
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to assign hostname";
  }
};

const handleSetPrimary = async (domainId: string) => {
  pageError.value = "";
  successMessage.value = "";
  try {
    await updateDomain(domainId, { is_primary: true });
    successMessage.value = "Primary hostname updated.";
    await Promise.all([refreshDomains(), refreshSite(), refreshHealth(), loadActivity()]);
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to update primary hostname";
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
    await Promise.all([refreshDomains(), refreshSite(), refreshHealth(), loadActivity()]);
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to remove hostname";
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
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to delete site";
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
    <div v-if="loading" class="flex items-center justify-center py-24 text-sm text-muted-foreground">Loading site record...</div>

    <template v-else-if="site">
      <div class="flex flex-col gap-5 rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(18,34,42,0.96),rgba(18,58,56,0.9)_52%,rgba(28,38,61,0.92))] px-7 py-7 text-white shadow-[0_32px_120px_-52px_rgba(9,18,32,0.85)]">
        <NuxtLink to="/sites" class="text-xs font-semibold uppercase tracking-[0.22em] text-white/65 transition hover:text-white">Back to sites</NuxtLink>

        <div class="flex flex-col gap-1">
          <div class="flex flex-wrap items-center gap-3">
            <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ site.name }}</h1>
            <Badge variant="outline" :class="siteDeploymentMeta(site.deployment_state).className">{{ siteDeploymentMeta(site.deployment_state).label }}</Badge>
            <Badge variant="outline" :class="siteRuntimeMeta(site.runtime_health_state).className">{{ siteRuntimeMeta(site.runtime_health_state).label }}</Badge>
          </div>
          <p class="flex flex-wrap items-center gap-x-2 text-sm text-white/55">
            <span>{{ site.primary_domain || "No hostname" }}</span>
            <span v-if="site.php_version" class="before:mr-2 before:content-['·']">PHP {{ site.php_version }}</span>
            <span v-if="serverLocation" class="before:mr-2 before:content-['·']">{{ serverLocation }}</span>
            <span v-if="serverProfile" class="before:mr-2 before:content-['·']">{{ serverProfile }}</span>
          </p>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-wrap gap-2">
            <Button size="sm" class="bg-white/15 text-white hover:bg-white/25 border-white/10 border shadow-none" :disabled="!canOpenSite" @click="openSite">
              <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" /></svg>
              Open Site
            </Button>
            <Button size="sm" class="bg-white/15 text-white hover:bg-white/25 border-white/10 border shadow-none" @click="triggerNotImplemented">
              <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" /></svg>
              WP Admin
            </Button>
            <Button size="sm" class="bg-white/15 text-white hover:bg-white/25 border-white/10 border shadow-none" @click="triggerNotImplemented">
              <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" /></svg>
              Create Staging
            </Button>
            <Button size="sm" class="bg-white/15 text-white hover:bg-white/25 border-white/10 border shadow-none" @click="triggerNotImplemented">
              <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182M4.031 9.865" /></svg>
              Run Updates
            </Button>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <Button size="sm" class="bg-white/10 text-white/70 hover:bg-white/20 hover:text-white border-white/10 border shadow-none" @click="triggerNotImplemented">
              <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
              Clear Cache
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button size="sm" class="bg-white/10 text-white/70 hover:bg-white/20 hover:text-white border-white/10 border shadow-none">
                  <svg xmlns="http://www.w3.org/2000/svg" class="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM12.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM18.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" /></svg>
                  More
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-48">
                <DropdownMenuItem @click="triggerNotImplemented">
                  <svg xmlns="http://www.w3.org/2000/svg" class="mr-2 size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.1 3.01 1.36-5.68L2.3 8.39l5.83-.5L10.5 2.5l2.37 5.39 5.83.5-4.38 4.11 1.36 5.68z" /></svg>
                  Maintenance Mode
                </DropdownMenuItem>
                <DropdownMenuItem @click="triggerNotImplemented">
                  <svg xmlns="http://www.w3.org/2000/svg" class="mr-2 size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" /></svg>
                  Switch Environment
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>

      <Alert v-if="pageError" class="border-destructive/30 bg-destructive/10 text-destructive">
        <AlertDescription>{{ pageError }}</AlertDescription>
      </Alert>
      <Alert v-if="successMessage" class="border-primary/30 bg-primary/10 text-primary">
        <AlertDescription>{{ successMessage }}</AlertDescription>
      </Alert>

      <div class="lg:hidden">
        <button
          class="flex w-full items-center justify-between rounded-lg border border-border/60 bg-card/50 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-card/70"
          @click="toggleMobileSidebar"
        >
          <span class="flex items-center gap-2.5">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" :d="currentSection.icon" />
            </svg>
            {{ currentSection.label }}
          </span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4 text-muted-foreground transition-transform"
            :class="{ 'rotate-180': isMobileSidebarOpen }"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 -translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-1"
        >
          <div v-if="isMobileSidebarOpen" class="mt-1 overflow-hidden rounded-lg border border-border/60 bg-card/80 backdrop-blur-sm">
            <nav aria-label="Site sections">
              <button
                v-for="section in sections"
                :key="section.key"
                :class="[
                  'flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-sm transition-colors',
                  activeSection === section.key
                    ? 'bg-accent/10 text-accent'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                ]"
                @click="selectSection(section.key)"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
                </svg>
                {{ section.label }}
              </button>
            </nav>
          </div>
        </Transition>
      </div>

      <div class="flex gap-6">
        <aside class="hidden w-56 shrink-0 lg:block">
          <nav aria-label="Site sections" class="space-y-0.5">
            <button
              v-for="section in sections"
              :key="section.key"
              :class="[
                'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                activeSection === section.key
                  ? 'bg-accent/10 text-accent'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
              ]"
              @click="navigateTo(section.key)"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
              </svg>
              {{ section.label }}
            </button>
          </nav>
        </aside>

        <div class="min-w-0 flex-1">
          <Card class="rounded-xl border border-border/60 bg-card/50 py-0 shadow-none backdrop-blur-sm">
            <CardHeader class="border-b border-border/40 px-6 py-5">
              <div>
                <h2 class="text-lg font-semibold text-foreground">{{ currentSection.label }}</h2>
                <p class="mt-0.5 text-sm text-muted-foreground">{{ currentSection.description }}</p>
              </div>
            </CardHeader>

            <CardContent class="px-6 py-5">
              <SiteOverview
                v-if="activeSection === 'overview'"
                :site="site"
                :site-health="siteHealth"
                :site-domains="siteDomains"
                :activities="activities"
                :server-name="currentServer?.name || site.server_name"
                :server-id="site.server_id"
              />

              <SiteHostnames
                v-else-if="activeSection === 'hostnames'"
                :domains="siteDomains"
                :current-server-i-pv4="currentServerIPv4"
                :saving="saving"
                @assign="handleAssignHostname"
                @set-primary="handleSetPrimary"
                @remove="handleRemoveHostname"
              />

              <SiteSettings
                v-else-if="activeSection === 'settings'"
                :name="settingsName"
                :wordpress-admin-email="settingsEmail"
                :saving="saving"
                @save="handleSave"
                @delete="handleDelete"
              />

              <div v-else class="space-y-6">
                <div v-if="site.last_deploy_job_id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                  <p class="mb-4 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Latest deploy</p>
                  <JobTimeline :job-id="site.last_deploy_job_id" :compact="true" />
                </div>
                <div v-else class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
                  No deployment job has been recorded yet.
                </div>

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
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>

    <Alert v-else class="border-destructive/30 bg-destructive/10 text-destructive">
      <AlertDescription>{{ pageError || "Site not found" }}</AlertDescription>
    </Alert>

    <NotImplementedDialog />
  </div>
</template>
