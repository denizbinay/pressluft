<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { cn, errorMessage } from "@/lib/utils";
import { useServers } from "~/composables/useServers";
import { useAllAgentStatus } from "~/composables/useAgentStatus";
import {
  inProgressServerStatuses,
  mutationBlockedServerStatuses,
  type ServerStatus,
  type SetupState,
} from "~/lib/platform-contract.generated";

const emit = defineEmits<{
  (e: "create"): void;
}>();

const {
  servers,
  loading,
  deleteServer,
  fetchServers,
} = useServers();

const { getStatusType, isConnected } = useAllAgentStatus({
  pollInterval: 15000,
});

// Delete confirmation state
const deleteConfirmId = ref<string | null>(null);
const deleting = ref(false);
const deleteError = ref("");
const deleteSuccess = ref("");

const buttonBaseClass =
  "rounded-lg focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return iso;
  }
};

const statusBadgeClass = (status: ServerStatus): string => {
  if (status === "ready") return "border-primary/30 bg-primary/10 text-primary";
  if (status === "failed")
    return "border-destructive/30 bg-destructive/10 text-destructive";
  if (inProgressServerStatuses.includes(status)) {
    return "border-accent/30 bg-accent/10 text-accent";
  }
  if (status === "deleted")
    return "border-border/60 bg-muted/60 text-muted-foreground";
  return "border-border/60 bg-muted/60 text-foreground";
};

const setupBadgeClass = (setupState?: SetupState): string => {
  if (setupState === "ready")
    return "border-primary/30 bg-primary/10 text-primary";
  if (setupState === "degraded")
    return "border-destructive/30 bg-destructive/10 text-destructive";
  if (setupState === "running")
    return "border-accent/30 bg-accent/10 text-accent";
  return "border-border/60 bg-muted/60 text-muted-foreground";
};

const confirmDelete = (serverId: string) => {
  deleteError.value = "";
  deleteSuccess.value = "";
  deleteConfirmId.value = serverId;
};

const cancelDelete = () => {
  deleteConfirmId.value = null;
  deleteError.value = "";
};

const executeDelete = async (serverId: string) => {
  deleting.value = true;
  deleteError.value = "";
  deleteSuccess.value = "";
  try {
    const result = await deleteServer(serverId);
    deleteSuccess.value = `Delete job #${result.job_id} queued`;
    deleteConfirmId.value = null;
    await fetchServers();
  } catch (e: unknown) {
    deleteError.value = errorMessage(e) || "Failed to queue delete job";
  } finally {
    deleting.value = false;
  }
};
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">
        Provision managed servers for agency WordPress workloads.
      </p>
      <Button
        size="sm"
        :class="
          cn(
            buttonBaseClass,
            'bg-primary text-primary-foreground hover:bg-primary/90',
          )
        "
        @click="emit('create')"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 4v16m8-8H4"
          />
        </svg>
        Create New Server
      </Button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-10">
      <Spinner class="text-muted-foreground" />
    </div>

    <div
      v-else-if="servers.length === 0"
      class="rounded-lg border border-dashed border-border/50 px-4 py-10 text-center"
    >
      <h3 class="text-sm font-medium text-foreground">No servers yet</h3>
      <p class="mt-1 text-sm text-muted-foreground">
        Create your first managed server to start onboarding WordPress sites.
      </p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="server in servers"
        :key="server.id"
        class="group flex items-center justify-between rounded-lg border border-border/60 bg-card/30 px-4 py-3 transition-colors hover:border-border/80 hover:bg-card/50"
      >
        <NuxtLink :to="`/servers/${server.id}`" class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span
              class="text-sm font-medium text-foreground group-hover:text-foreground"
              >{{ server.name }}</span
            >
            <Badge :class="statusBadgeClass(server.status)">{{
              server.status
            }}</Badge>
            <Badge :class="setupBadgeClass(server.setup_state)"
              >setup {{ server.setup_state }}</Badge
            >
            <!-- Agent status indicator -->
            <span
              v-if="server.setup_state === 'ready'"
              class="flex items-center gap-1 text-xs"
              :title="'Agent: ' + getStatusType(server.id)"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="{
                  'bg-primary animate-pulse':
                    getStatusType(server.id) === 'online',
                  'bg-amber-500': getStatusType(server.id) === 'unhealthy',
                  'bg-muted-foreground/50':
                    getStatusType(server.id) === 'offline',
                  'bg-muted-foreground/30':
                    getStatusType(server.id) === 'unknown',
                }"
              />
              <span
                class="hidden sm:inline"
                :class="{
                  'text-primary': getStatusType(server.id) === 'online',
                  'text-amber-500': getStatusType(server.id) === 'unhealthy',
                  'text-muted-foreground/60':
                    getStatusType(server.id) === 'offline' ||
                    getStatusType(server.id) === 'unknown',
                }"
              >
                {{ isConnected(server.id) ? "Agent" : "" }}
              </span>
            </span>
          </div>
          <p class="text-xs text-muted-foreground">
            {{ server.location }} · {{ server.server_type }} ·
            {{ server.profile_key }} · Added {{ formatDate(server.created_at) }}
          </p>
          <p
            v-if="server.setup_state === 'degraded' && server.setup_last_error"
            class="mt-1 text-xs text-destructive"
          >
            Setup needs attention: {{ server.setup_last_error }}
          </p>
          <p
            v-if="server.status === 'deleting'"
            class="mt-1 text-xs text-accent"
          >
            Deletion is queued and runs asynchronously until provider-side
            removal completes.
          </p>
          <p
            v-else-if="server.status === 'deleted'"
            class="mt-1 text-xs text-muted-foreground"
          >
            Tombstone record retained after provider-side deletion.
          </p>
        </NuxtLink>
        <div class="flex items-center gap-3">
          <span class="text-xs text-muted-foreground">{{
            server.provider_type
          }}</span>
          <!-- Delete button (always visible for failed, hover for others) -->
          <Button
            v-if="deleteConfirmId !== server.id"
            variant="ghost"
            size="icon-sm"
            type="button"
            :class="
              cn(
                buttonBaseClass,
                'text-muted-foreground hover:bg-destructive/10 hover:text-destructive',
                !['failed', 'ready'].includes(server.status) &&
                  'opacity-0 group-hover:opacity-100',
              )
            "
            :disabled="mutationBlockedServerStatuses.includes(server.status)"
            title="Delete server"
            @click.prevent="confirmDelete(server.id)"
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
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </Button>
          <!-- Delete confirmation -->
          <div v-else class="flex items-center gap-1">
            <Button
              variant="ghost"
              type="button"
              :disabled="deleting"
              :class="
                cn(
                  buttonBaseClass,
                  'h-7 px-2 text-xs text-destructive hover:bg-destructive/10',
                )
              "
              @click.prevent="executeDelete(server.id)"
            >
              {{ deleting ? "Deleting..." : "Confirm" }}
            </Button>
            <Button
              variant="ghost"
              type="button"
              :disabled="deleting"
              :class="
                cn(
                  buttonBaseClass,
                  'h-7 px-2 text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                )
              "
              @click.prevent="cancelDelete"
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>
      <p v-if="deleteSuccess" class="text-xs text-primary">
        {{ deleteSuccess }}
      </p>
      <p v-if="deleteError" class="text-xs text-destructive">
        {{ deleteError }}
      </p>
    </div>
  </div>
</template>
