import { ref, reactive, computed, type Ref } from "vue";
import {
  useServers,
  type StoredServer,
} from "~/composables/useServers";
import { useJobs } from "~/composables/useJobs";
import { useAgentStatus } from "~/composables/useAgentStatus";
import {
  destructiveServerStatuses,
  mutationBlockedServerStatuses,
  type NodeStatus,
} from "~/lib/platform-contract.generated";
import { errorMessage } from "~/lib/utils";

export function useServerDetail(serverId: Ref<string | null>) {
  const { fetchServer } = useServers();
  const { createJob } = useJobs();

  const server = ref<StoredServer | null>(null);
  const loading = ref(true);
  const error = ref("");

  // Agent status
  const {
    agentInfo,
    fetch: fetchAgentStatus,
    startPolling: startAgentPolling,
    stopPolling: stopAgentPolling,
  } = useAgentStatus(serverId, { autoStart: false });

  const agentConnected = computed(() => agentInfo.value?.connected ?? false);
  const agentStatus = computed<NodeStatus>(
    () => agentInfo.value?.status ?? "unknown",
  );

  const agentStatusLabel = computed(() => {
    switch (agentStatus.value) {
      case "online":
        return "Online";
      case "unhealthy":
        return "Unhealthy";
      case "offline":
        return "Offline";
      default:
        return "Unknown";
    }
  });

  const serverIsDeleted = computed(() => server.value?.status === "deleted");
  const serverBlocksMutations = computed(() => {
    const status = server.value?.status;
    return status ? mutationBlockedServerStatuses.includes(status) : false;
  });
  const destructiveActionInProgress = computed(() => {
    const status = server.value?.status;
    return status ? destructiveServerStatuses.includes(status) : false;
  });
  const settingsDisabled = computed(
    () => serverBlocksMutations.value || destructiveActionInProgress.value,
  );
  const settingsBlockedReason = computed(() => {
    if (server.value?.status === "deleting")
      return "This server is deleting. New actions are blocked until deletion succeeds or fails.";
    if (server.value?.status === "deleted")
      return "This server is deleted. The record is retained only as a tombstone.";
    if (server.value?.status === "rebuilding")
      return "A rebuild job is already in progress.";
    if (server.value?.status === "resizing")
      return "A resize job is already in progress.";
    return "";
  });

  const setupRetryState = reactive({ loading: false, error: "", success: "" });

  const retrySetup = async () => {
    if (!serverId.value) return;
    setupRetryState.loading = true;
    setupRetryState.error = "";
    setupRetryState.success = "";
    try {
      const job = await createJob({
        kind: "configure_server",
        server_id: serverId.value,
        payload: {},
      });
      setupRetryState.success = `Setup job #${job.id} queued`;
      await refreshServer();
    } catch (e: unknown) {
      setupRetryState.error = errorMessage(e) || "Failed to queue setup retry";
    } finally {
      setupRetryState.loading = false;
    }
  };

  const refreshServer = async () => {
    if (!serverId.value) return;
    server.value = await fetchServer(serverId.value);
  };

  const loadServer = async () => {
    if (!serverId.value) {
      error.value = "Invalid server ID";
      loading.value = false;
      return;
    }

    try {
      server.value = await fetchServer(serverId.value);
      if (server.value?.setup_state === "ready") {
        startAgentPolling();
      }
    } catch (e: unknown) {
      error.value = errorMessage(e) || "Failed to load server";
    } finally {
      loading.value = false;
    }
  };

  return {
    server,
    loading,
    error,
    agentInfo,
    agentConnected,
    agentStatus,
    agentStatusLabel,
    serverIsDeleted,
    serverBlocksMutations,
    destructiveActionInProgress,
    settingsDisabled,
    settingsBlockedReason,
    setupRetryState,
    retrySetup,
    refreshServer,
    loadServer,
    stopAgentPolling,
  };
}
