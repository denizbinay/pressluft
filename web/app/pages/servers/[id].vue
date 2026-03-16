<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  inProgressServerStatuses,
  type ServerStatus,
} from "~/lib/platform-contract.generated";
import { useServerDetail } from "~/composables/useServerDetail";
import { copyToClipboard, formatShortId } from "~/lib/utils";

interface ServerSection {
  key: string;
  label: string;
  icon: string;
  description: string;
}

const sections: ServerSection[] = [
  {
    key: "overview",
    label: "Overview",
    icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6",
    description: "Server status and quick actions",
  },
  {
    key: "services",
    label: "Services",
    icon: "M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01",
    description: "Running services and management",
  },
  {
    key: "sites",
    label: "Sites",
    icon: "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9",
    description: "Sites currently attached to this server",
  },
  {
    key: "settings",
    label: "Settings",
    icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
    description: "Server configuration and management",
  },
  {
    key: "activity",
    label: "Activity",
    icon: "M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z",
    description: "Recent events and job history",
  },
];

const visibleSectionKeys = new Set([
  "overview",
  "services",
  "sites",
  "settings",
  "activity",
]);

const visibleSections = sections.filter((section) =>
  visibleSectionKeys.has(section.key),
);

const route = useRoute();
const router = useRouter();

const serverId = computed(() => {
  const raw = route.params.id;
  if (typeof raw !== "string") return null;
  const id = raw.trim();
  return id.length > 0 ? id : null;
});

const {
  server,
  loading,
  error,
  agentInfo,
  agentConnected,
  agentStatus,
  agentStatusLabel,
  serverIsDeleted,
  serverBlocksMutations,
  settingsDisabled,
  settingsBlockedReason,
  setupRetryState,
  retrySetup,
  refreshServer,
  loadServer,
  stopAgentPolling,
} = useServerDetail(serverId);

const activeSection = computed(() => {
  const tab = route.query.tab as string;
  const isValid = visibleSections.some((section) => section.key === tab);
  return isValid ? tab : "overview";
});

const currentSection = computed(
  () => sections.find((s) => s.key === activeSection.value)!,
);

const navigateTo = (key: string) => {
  router.push({ query: { tab: key } });
};

const isMobileSidebarOpen = ref(false);
const copiedId = ref("");
let copyResetTimer: ReturnType<typeof setTimeout> | null = null;

const toggleMobileSidebar = () => {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value;
};

const selectSection = (key: string) => {
  navigateTo(key);
  isMobileSidebarOpen.value = false;
};

const copyIdentifier = async (kind: string, value?: string) => {
  if (!value) return;
  const copied = await copyToClipboard(value);
  if (!copied) return;
  copiedId.value = kind;
  if (copyResetTimer) clearTimeout(copyResetTimer);
  copyResetTimer = setTimeout(() => {
    copiedId.value = "";
    copyResetTimer = null;
  }, 1500);
};

const statusVariant = (
  status: ServerStatus,
): "success" | "warning" | "danger" | "default" => {
  if (status === "ready") return "success";
  if (status === "failed") return "danger";
  if (inProgressServerStatuses.includes(status)) return "warning";
  return "default";
};

const setupVariant = (
  setupState: string,
): "success" | "warning" | "danger" | "default" => {
  if (setupState === "ready") return "success";
  if (setupState === "degraded") return "danger";
  if (setupState === "running") return "warning";
  return "default";
};

onMounted(() => {
  loadServer();
});

onUnmounted(() => {
  stopAgentPolling();
  if (copyResetTimer) clearTimeout(copyResetTimer);
});
</script>

<template>
  <div>
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <svg
        class="h-6 w-6 animate-spin text-muted-foreground"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          class="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="4"
        />
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="space-y-4">
      <div>
        <h1 class="text-2xl font-semibold text-foreground">Server Not Found</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ error }}</p>
      </div>
      <NuxtLink
        to="/servers"
        class="inline-flex items-center gap-1 text-sm text-accent transition-colors hover:text-accent/80"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M10 19l-7-7m0 0l7-7m-7 7h18"
          />
        </svg>
        Back to Servers
      </NuxtLink>
    </div>

    <!-- Server detail -->
    <template v-else-if="server">
      <!-- Page header -->
      <div class="mb-6">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <NuxtLink
            to="/servers"
            class="hover:text-foreground/80 transition-colors"
            >Servers</NuxtLink
          >
          <span>/</span>
          <span class="text-foreground/80">{{ server.name }}</span>
        </div>
        <div class="mt-2 flex items-center gap-3">
          <h1 class="text-2xl font-semibold text-foreground">
            {{ server.name }}
          </h1>
          <Badge
            variant="outline"
            :class="[
              'px-2.5 py-1 text-sm border',
              statusVariant(server.status) === 'success' &&
                'border-primary/30 bg-primary/10 text-primary',
              statusVariant(server.status) === 'warning' &&
                'border-accent/30 bg-accent/10 text-accent',
              statusVariant(server.status) === 'danger' &&
                'border-destructive/30 bg-destructive/10 text-destructive',
              statusVariant(server.status) === 'default' &&
                'border-border/60 bg-muted/60 text-foreground',
            ]"
          >
            {{ server.status }}
          </Badge>
          <Badge
            variant="outline"
            :class="[
              'px-2.5 py-1 text-sm border',
              setupVariant(server.setup_state) === 'success' &&
                'border-primary/30 bg-primary/10 text-primary',
              setupVariant(server.setup_state) === 'warning' &&
                'border-accent/30 bg-accent/10 text-accent',
              setupVariant(server.setup_state) === 'danger' &&
                'border-destructive/30 bg-destructive/10 text-destructive',
              setupVariant(server.setup_state) === 'default' &&
                'border-border/60 bg-muted/60 text-foreground',
            ]"
          >
            setup {{ server.setup_state }}
          </Badge>
          <!-- Agent status badge -->
          <Badge
            v-if="server.setup_state === 'ready'"
            variant="outline"
            :class="[
              'px-2.5 py-1 text-sm border flex items-center gap-1.5',
              agentStatus === 'online' &&
                'border-primary/30 bg-primary/10 text-primary',
              agentStatus === 'unhealthy' &&
                'border-amber-500/30 bg-amber-500/10 text-amber-600',
              agentStatus === 'offline' &&
                'border-border/60 bg-muted/60 text-muted-foreground',
              agentStatus === 'unknown' &&
                'border-border/40 bg-muted/40 text-muted-foreground',
            ]"
          >
            <span
              class="h-1.5 w-1.5 rounded-full"
              :class="{
                'bg-primary animate-pulse': agentStatus === 'online',
                'bg-amber-500': agentStatus === 'unhealthy',
                'bg-muted-foreground/50': agentStatus === 'offline',
                'bg-muted-foreground/30': agentStatus === 'unknown',
              }"
            />
            Agent {{ agentStatusLabel }}
          </Badge>
        </div>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ server.location }} · {{ server.server_type }} ·
          {{ server.profile_key }}
        </p>
        <div class="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <button type="button" class="rounded-full border border-border/60 bg-background/70 px-2.5 py-1 font-mono transition hover:bg-muted/50 hover:text-foreground" :title="server.id" @click="copyIdentifier('server', server.id)">
            {{ copiedId === 'server' ? 'Server ID copied' : `Server ${formatShortId(server.id)}` }}
          </button>
          <button v-if="server.provider_server_id" type="button" class="rounded-full border border-border/60 bg-background/70 px-2.5 py-1 font-mono transition hover:bg-muted/50 hover:text-foreground" :title="server.provider_server_id" @click="copyIdentifier('provider', server.provider_server_id)">
            {{ copiedId === 'provider' ? 'Provider ID copied' : `Provider ${formatShortId(server.provider_server_id)}` }}
          </button>
        </div>
      </div>

      <!-- Mobile section selector -->
      <div class="mb-4 lg:hidden">
        <button
          class="flex w-full items-center justify-between rounded-lg border border-border/60 bg-card/50 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-card/70"
          @click="toggleMobileSidebar"
        >
          <span class="flex items-center gap-2.5">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-4 w-4 text-muted-foreground"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                :d="currentSection.icon"
              />
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
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>

        <!-- Mobile dropdown -->
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 -translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-1"
        >
          <div
            v-if="isMobileSidebarOpen"
            class="mt-1 overflow-hidden rounded-lg border border-border/60 bg-card/80 backdrop-blur-sm"
          >
            <nav aria-label="Server sections">
              <button
                v-for="section in visibleSections"
                :key="section.key"
                :class="[
                  'flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-sm transition-colors',
                  activeSection === section.key
                    ? 'bg-accent/10 text-accent'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                ]"
                @click="selectSection(section.key)"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 shrink-0"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    :d="section.icon"
                  />
                </svg>
                {{ section.label }}
              </button>
            </nav>
          </div>
        </Transition>
      </div>

      <!-- Desktop layout: sidebar + content -->
      <div class="flex gap-6">
        <!-- Sidebar (desktop) -->
        <aside class="hidden w-56 shrink-0 lg:block">
          <nav aria-label="Server sections" class="space-y-0.5">
            <button
              v-for="section in visibleSections"
              :key="section.key"
              :class="[
                'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                activeSection === section.key
                  ? 'bg-accent/10 text-accent'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
              ]"
              @click="navigateTo(section.key)"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4 shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  :d="section.icon"
                />
              </svg>
              {{ section.label }}
            </button>
          </nav>
        </aside>

        <!-- Content area -->
        <div class="min-w-0 flex-1">
          <Card
            class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none"
          >
            <CardHeader class="border-b border-border/40 px-6 py-5">
              <div>
                <h2 class="text-lg font-semibold text-foreground">
                  {{ currentSection.label }}
                </h2>
                <p class="mt-0.5 text-sm text-muted-foreground">
                  {{ currentSection.description }}
                </p>
              </div>
            </CardHeader>

            <CardContent class="px-6 py-5">
              <!-- Section content -->
              <div class="space-y-6">
                <!-- Overview -->
                <div v-if="activeSection === 'overview'" class="space-y-4">
                  <ServerOverview
                    :server="server"
                    :agent-info="agentInfo"
                    :agent-connected="agentConnected"
                    :setup-retry-state="setupRetryState"
                    :server-blocks-mutations="serverBlocksMutations"
                    :server-is-deleted="serverIsDeleted"
                    @retry-setup="retrySetup"
                    @refresh="refreshServer"
                  />
                </div>

                <!-- Services -->
                <div v-if="activeSection === 'services'" class="space-y-4">
                  <ServerServices
                    :server-id="serverId!"
                    :agent-connected="agentConnected"
                    :agent-status-label="agentStatusLabel"
                  />
                </div>

                <!-- Sites -->
                <div v-if="activeSection === 'sites'" class="space-y-4">
                  <ServerSites :server-id="serverId!" />
                </div>

                <!-- Settings -->
                <div v-if="activeSection === 'settings'" class="space-y-4">
                  <ServerSettings
                    :server="server"
                    :server-id="serverId!"
                    :agent-connected="agentConnected"
                    :settings-disabled="settingsDisabled"
                    :settings-blocked-reason="settingsBlockedReason"
                    @refresh="refreshServer"
                  />
                </div>

                <!-- Activity -->
                <div v-if="activeSection === 'activity'" class="space-y-4">
                  <ServerActivity :server-id="server.id" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>
  </div>
</template>
