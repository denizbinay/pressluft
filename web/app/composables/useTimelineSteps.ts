import { computed, type Ref } from "vue";
import type { Job, JobEvent } from "~/composables/useJobs";
import {
  jobKindLabels,
  jobKindSteps,
  type JobKind,
} from "~/lib/platform-contract.generated";

export interface TimelineStep {
  key: string;
  label: string;
  status: "pending" | "running" | "completed" | "failed";
  message?: string;
  timestamp?: string;
}

export function useTimelineSteps(
  activeJob: Ref<Job | null>,
  events: Ref<readonly JobEvent[]>,
) {
  const steps = computed<TimelineStep[]>(() => {
    const kind = activeJob.value?.kind as JobKind | undefined;
    const eventStepOrder: string[] = [];
    for (const event of events.value) {
      if (event.step_key && !eventStepOrder.includes(event.step_key)) {
        eventStepOrder.push(event.step_key);
      }
    }
    const workflowSteps = kind ? jobKindSteps[kind] || [] : [];
    const stepOrder =
      workflowSteps.length > 0
        ? workflowSteps.map((step) => step.key)
        : eventStepOrder;
    const eventsByStep = new Map<string, JobEvent[]>();

    // Group events by step_key
    for (const event of events.value) {
      if (event.step_key) {
        const existing = eventsByStep.get(event.step_key) || [];
        existing.push(event);
        eventsByStep.set(event.step_key, existing);
      }
    }

    // Build timeline steps
    return stepOrder.map((key) => {
      const stepEvents = eventsByStep.get(key) || [];
      const latestEvent = stepEvents[stepEvents.length - 1];

      let status: TimelineStep["status"] = "pending";
      if (latestEvent) {
        if (
          latestEvent.status === "completed" ||
          latestEvent.event_type === "step_completed"
        ) {
          status = "completed";
        } else if (
          latestEvent.status === "failed" ||
          latestEvent.event_type === "step_failed"
        ) {
          status = "failed";
        } else if (
          latestEvent.status === "running" ||
          latestEvent.event_type === "step_started"
        ) {
          status = "running";
        }
      }

      // Check if this is the current step from the job
      if (activeJob.value?.current_step === key && status === "pending") {
        status = "running";
      }

      return {
        key,
        label:
          workflowSteps.find((step) => step.key === key)?.label || key,
        status,
        message: latestEvent?.message,
        timestamp: latestEvent?.occurred_at,
      };
    });
  });

  const jobKindLabel = computed(() => {
    const kind = activeJob.value?.kind;
    return kind ? jobKindLabels[kind as JobKind] || kind : "";
  });

  function formatPayloadValue(value: unknown): string {
    if (value === null) return "null";
    if (typeof value === "string") return value;
    if (typeof value === "number" || typeof value === "boolean")
      return String(value);
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }

  const payloadSummary = computed(() => {
    const payload = activeJob.value?.payload;
    if (!payload) return "";
    if (typeof payload === "string") {
      try {
        const parsed = JSON.parse(payload) as Record<string, unknown>;
        const entries = Object.entries(parsed);
        if (entries.length === 0) return "";
        const summary = entries
          .map(
            ([key, value]) => `${key}=${formatPayloadValue(value)}`,
          )
          .join(", ");
        return summary.length > 160
          ? `${summary.slice(0, 157)}...`
          : summary;
      } catch {
        return payload;
      }
    }
    return "";
  });

  return {
    steps,
    jobKindLabel,
    payloadSummary,
    formatPayloadValue,
  };
}
