import { ref, onMounted, onUnmounted } from "vue";
import {
  useJobs,
  type Job,
  type JobEvent,
  type ConnectionMode,
} from "~/composables/useJobs";
import {
  jobTerminalStatuses,
  type JobTerminalStatus,
} from "~/lib/platform-contract.generated";
import { errorMessage } from "~/lib/utils";

export interface UseJobStreamOptions {
  jobId: string;
  autoConnect: boolean;
  onCompleted?: (job: Job) => void;
  onFailed?: (job: Job, error: string) => void;
}

export function useJobStream(options: UseJobStreamOptions) {
  const {
    activeJob,
    events,
    connectionMode,
    fetchJob,
    fetchJobEvents,
    streamJobEvents,
    clearEvents,
  } = useJobs();

  /** Whether we're showing a historical (already completed) job vs live */
  const isHistoricalView = ref(false);

  const loading = ref(true);
  const connectionError = ref("");
  const retryCount = ref(0);

  // SSE cleanup function
  let closeStream: (() => void) | null = null;

  // Watch for terminal states and emit events
  watch(
    () => activeJob.value?.status,
    (status) => {
      if (!activeJob.value) return;

      if (status === "succeeded") {
        options.onCompleted?.(activeJob.value);
      } else if (status === "failed") {
        options.onFailed?.(
          activeJob.value,
          activeJob.value.last_error || "Unknown error",
        );
      }
    },
  );

  // Handle event updates to refresh job status
  function handleEvent(event: JobEvent) {
    // Refresh job when we get terminal events
    if (
      event.event_type === "job_succeeded" ||
      event.event_type === "job_failed" ||
      event.event_type === "job_timed_out" ||
      event.event_type === "job_recovered"
    ) {
      fetchJob(options.jobId).catch(() => {});
    }
  }

  // Handle connection mode changes
  function handleModeChange(mode: ConnectionMode) {
    if (mode === "polling") {
      // Clear any previous connection error when we successfully fall back to polling
      connectionError.value = "";
    }
  }

  // Retry loading the job
  async function retryLoad() {
    retryCount.value++;
    connectionError.value = "";
    loading.value = true;
    clearEvents();

    try {
      const job = await fetchJob(options.jobId);

      // Check if job is already in terminal state (historical view)
      if (
        jobTerminalStatuses.includes(job.status as JobTerminalStatus)
      ) {
        isHistoricalView.value = true;
        await fetchJobEvents(options.jobId);
        loading.value = false;
        return;
      }

      loading.value = false;
      if (options.autoConnect) {
        if (closeStream) closeStream();
        closeStream = streamJobEvents(
          options.jobId,
          handleEvent,
          handleModeChange,
        );
      }
    } catch (e: unknown) {
      connectionError.value =
        errorMessage(e) || "Failed to load job";
      loading.value = false;
    }
  }

  onMounted(async () => {
    try {
      const job = await fetchJob(options.jobId);

      // Check if job is already in terminal state (historical view)
      if (
        jobTerminalStatuses.includes(job.status as JobTerminalStatus)
      ) {
        // Historical view: fetch all events at once, no streaming
        isHistoricalView.value = true;
        await fetchJobEvents(options.jobId);
        loading.value = false;
        return;
      }

      // Live view: stream events
      loading.value = false;
      if (options.autoConnect) {
        closeStream = streamJobEvents(
          options.jobId,
          handleEvent,
          handleModeChange,
        );
      }
    } catch (e: unknown) {
      connectionError.value =
        errorMessage(e) || "Failed to load job";
      loading.value = false;
    }
  });

  onUnmounted(() => {
    if (closeStream) {
      closeStream();
      closeStream = null;
    }
  });

  return {
    activeJob,
    events,
    connectionMode,
    isHistoricalView,
    loading,
    connectionError,
    retryCount,
    retryLoad,
  };
}
