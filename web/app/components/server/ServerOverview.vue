<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { StoredServer, AgentInfo } from "~/composables/useServers";
import { inProgressServerStatuses } from "~/lib/platform-contract.generated";
import type { ServerStatus } from "~/lib/platform-contract.generated";

interface SetupRetryState {
  loading: boolean;
  error: string;
  success: string;
}

const props = defineProps<{
  server: StoredServer;
  agentInfo: AgentInfo | null;
  agentConnected: boolean;
  setupRetryState: SetupRetryState;
  serverBlocksMutations: boolean;
  serverIsDeleted: boolean;
}>();

const emit = defineEmits<{
  (e: "retry-setup"): void;
  (e: "refresh"): void;
}>();

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

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
};

const memPercent = computed(() => {
  if (!props.agentInfo?.mem_total_mb) return 0;
  return Math.round(
    ((props.agentInfo.mem_used_mb ?? 0) / props.agentInfo.mem_total_mb) * 100,
  );
});
</script>

<template>
  <div class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2">
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Status
        </p>
        <div class="mt-1 flex items-center gap-2">
          <span
            class="h-2 w-2 rounded-full"
            :class="{
              'bg-primary': server.status === 'ready',
              'bg-destructive': server.status === 'failed',
              'bg-accent animate-pulse':
                inProgressServerStatuses.includes(server.status),
              'bg-muted-foreground': ![
                'ready',
                'failed',
                'pending',
                'provisioning',
                'configuring',
                'rebuilding',
                'resizing',
                'deleting',
              ].includes(server.status),
            }"
          />
          <span
            class="text-sm font-medium text-foreground/80 capitalize"
            >{{ server.status }}</span
          >
        </div>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Setup
        </p>
        <div class="mt-1 flex items-center gap-2">
          <span
            class="h-2 w-2 rounded-full"
            :class="{
              'bg-primary': server.setup_state === 'ready',
              'bg-destructive': server.setup_state === 'degraded',
              'bg-accent animate-pulse':
                server.setup_state === 'running',
              'bg-muted-foreground': ![
                'ready',
                'degraded',
                'running',
              ].includes(server.setup_state),
            }"
          />
          <span
            class="text-sm font-medium text-foreground/80 capitalize"
            >{{ server.setup_state }}</span
          >
        </div>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Provider
        </p>
        <p class="mt-1 text-sm font-medium text-foreground/80">
          {{ server.provider_type }}
        </p>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Location
        </p>
        <p class="mt-1 text-sm font-medium text-foreground/80">
          {{ server.location }}
        </p>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Server Type
        </p>
        <p class="mt-1 text-sm font-medium text-foreground/80">
          {{ server.server_type }}
        </p>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Profile
        </p>
        <p class="mt-1 text-sm font-medium text-foreground/80">
          {{ server.profile_key }}
        </p>
      </div>
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Created
        </p>
        <p class="mt-1 text-sm font-medium text-foreground/80">
          {{ formatDate(server.created_at) }}
        </p>
      </div>
    </div>

    <!-- Provider Server ID (if available) -->
    <div
      v-if="server.provider_server_id"
      class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
    >
      <p class="text-xs font-medium text-muted-foreground">
        Provider Server ID
      </p>
      <p class="mt-1 font-mono text-sm text-foreground/80">
        {{ server.provider_server_id }}
      </p>
    </div>

    <div
      v-if="server.status === 'deleting'"
      class="rounded-lg border border-accent/30 bg-accent/10 px-4 py-3 text-sm text-accent"
    >
      Deletion is in progress asynchronously. The record stays
      visible until provider-side removal finishes and the server
      becomes a tombstone.
    </div>
    <div
      v-else-if="serverIsDeleted"
      class="rounded-lg border border-border/60 bg-muted/60 px-4 py-3 text-sm text-muted-foreground"
    >
      This server has been deleted. Pressluft keeps this record as
      a tombstone for audit history.
    </div>
    <div
      v-else-if="server.setup_state === 'degraded'"
      class="space-y-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      <p>
        Setup needs attention<span v-if="server.setup_last_error"
          >: {{ server.setup_last_error }}</span
        >
      </p>
      <div class="flex flex-wrap items-center gap-3">
        <Button
          :disabled="
            setupRetryState.loading || serverBlocksMutations
          "
          @click="emit('retry-setup')"
        >
          {{
            setupRetryState.loading
              ? "Queuing setup..."
              : "Retry setup"
          }}
        </Button>
        <span
          v-if="setupRetryState.success"
          class="text-xs text-primary"
          >{{ setupRetryState.success }}</span
        >
        <span
          v-if="setupRetryState.error"
          class="text-xs text-destructive"
          >{{ setupRetryState.error }}</span
        >
      </div>
    </div>

    <!-- Agent Metrics (only show when server is ready) -->
    <template v-if="server.setup_state === 'ready'">
      <div class="grid gap-4 sm:grid-cols-2">
        <!-- CPU Usage -->
        <div
          class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
        >
          <div class="flex items-center justify-between">
            <p class="text-xs font-medium text-muted-foreground">
              CPU Usage
            </p>
            <span
              v-if="agentConnected"
              class="flex items-center gap-1 text-xs text-primary"
            >
              <span
                class="h-1 w-1 animate-pulse rounded-full bg-primary"
              />
              Live
            </span>
          </div>
          <template v-if="agentConnected">
            <div class="mt-2 flex items-center gap-3">
              <div
                class="h-2 flex-1 overflow-hidden rounded-full bg-muted"
              >
                <div
                  class="h-2 rounded-full bg-accent transition-all duration-500"
                  :style="{
                    width: `${Math.min(agentInfo?.cpu_percent ?? 0, 100)}%`,
                  }"
                />
              </div>
              <span
                class="w-12 text-right text-sm font-medium text-foreground/80"
              >
                {{ (agentInfo?.cpu_percent ?? 0).toFixed(1) }}%
              </span>
            </div>
          </template>
          <div
            v-else
            class="mt-1 text-sm text-muted-foreground/60"
          >
            Agent not connected
          </div>
        </div>

        <!-- Memory Usage -->
        <div
          class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
        >
          <div class="flex items-center justify-between">
            <p class="text-xs font-medium text-muted-foreground">
              Memory
            </p>
            <span
              v-if="agentConnected"
              class="flex items-center gap-1 text-xs text-primary"
            >
              <span
                class="h-1 w-1 animate-pulse rounded-full bg-primary"
              />
              Live
            </span>
          </div>
          <template v-if="agentConnected">
            <div class="mt-2 flex items-center gap-3">
              <div
                class="h-2 flex-1 overflow-hidden rounded-full bg-muted"
              >
                <div
                  class="h-2 rounded-full bg-primary transition-all duration-500"
                  :style="{ width: `${memPercent}%` }"
                />
              </div>
              <span
                class="w-24 text-right text-sm font-medium text-foreground/80"
              >
                {{ agentInfo?.mem_used_mb ?? 0 }} /
                {{ agentInfo?.mem_total_mb ?? 0 }} MB
              </span>
            </div>
          </template>
          <div
            v-else
            class="mt-1 text-sm text-muted-foreground/60"
          >
            Agent not connected
          </div>
        </div>
      </div>

      <!-- Agent info -->
      <div
        v-if="agentConnected && agentInfo?.version"
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <p class="text-xs font-medium text-muted-foreground">
          Agent Version
        </p>
        <p class="mt-1 font-mono text-sm text-foreground/80">
          {{ agentInfo.version }}
        </p>
      </div>
    </template>

    <!-- Quick actions placeholder -->
    <div
      class="rounded-lg border border-dashed border-border/50 px-4 py-6 text-center"
    >
      <p class="text-sm text-muted-foreground">
        Quick actions (reboot, stop, start, SSH access) will be
        available here.
      </p>
    </div>
  </div>
</template>
