<script setup lang="ts">
import type { StoredServer } from "~/composables/useServers";
import { useServerOptions } from "~/composables/useServerOptions";

const props = defineProps<{
  server: StoredServer;
  serverId: string;
  agentConnected: boolean;
  settingsDisabled: boolean;
  settingsBlockedReason: string;
}>();

const emit = defineEmits<{
  (e: "refresh"): void;
}>();

const {
  images: imageOptions,
  serverTypes: serverTypeOptions,
  firewalls: firewallOptions,
  volumes: volumeOptions,
  loading: optionsLoading,
  error: optionsError,
  fetchAll: fetchServerOptions,
} = useServerOptions();

// Load server options on mount
onMounted(() => {
  fetchServerOptions(props.serverId);
});
</script>

<template>
  <div class="space-y-4">
    <div
      v-if="settingsBlockedReason"
      class="rounded-lg border border-accent/30 bg-accent/10 px-3 py-2 text-xs text-accent"
    >
      {{ settingsBlockedReason }}
    </div>
    <!-- Execution mode indicator -->
    <div
      class="flex items-center gap-2 rounded-lg border border-border/40 bg-muted/30 px-3 py-2"
    >
      <span
        class="h-2 w-2 rounded-full"
        :class="
          agentConnected
            ? 'bg-primary animate-pulse'
            : 'bg-muted-foreground'
        "
      />
      <span class="text-xs text-muted-foreground">
        {{
          agentConnected
            ? "Runtime diagnostics use the agent; provisioning and deploy workflows still run via Ansible over SSH."
            : "Provisioning and deploy workflows run via Ansible over SSH; live diagnostics improve when the agent is connected."
        }}
      </span>
    </div>
    <div class="grid gap-4 lg:grid-cols-2">
      <ServerRebuildForm
        :server-id="serverId"
        :server="server"
        :settings-disabled="settingsDisabled"
        :image-options="imageOptions"
        :options-loading="optionsLoading"
        :options-error="optionsError"
        @refresh="emit('refresh')"
      />
      <ServerResizeForm
        :server-id="serverId"
        :server="server"
        :settings-disabled="settingsDisabled"
        :server-type-options="serverTypeOptions"
        :options-loading="optionsLoading"
        :options-error="optionsError"
        @refresh="emit('refresh')"
      />
      <ServerFirewallForm
        :server-id="serverId"
        :settings-disabled="settingsDisabled"
        :firewall-options="firewallOptions"
        :options-loading="optionsLoading"
        :options-error="optionsError"
      />
      <ServerVolumeForm
        :server-id="serverId"
        :server="server"
        :settings-disabled="settingsDisabled"
        :volume-options="volumeOptions"
        :options-loading="optionsLoading"
        :options-error="optionsError"
      />
    </div>
  </div>
</template>
