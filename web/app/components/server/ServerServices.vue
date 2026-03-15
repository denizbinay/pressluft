<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  useServers,
  type Service,
} from "~/composables/useServers";
import { useJobs } from "~/composables/useJobs";
import {
  jobTerminalStatuses,
  type JobTerminalStatus,
} from "~/lib/platform-contract.generated";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  serverId: string;
  agentConnected: boolean;
  agentStatusLabel: string;
}>();

const { fetchServices } = useServers();
const { createJob, fetchJob } = useJobs();

const services = ref<Service[]>([]);
const servicesLoading = ref(false);
const servicesError = ref("");

type ServiceActionStatus =
  | "idle"
  | "queued"
  | "running"
  | "succeeded"
  | "failed";

interface ServiceActionState {
  status: ServiceActionStatus;
  message: string;
  jobId?: string;
}

const serviceActions = reactive<Record<string, ServiceActionState>>({});
const serviceMonitors = new Map<string, ReturnType<typeof setInterval>>();

const terminalStatuses = new Set<string>(jobTerminalStatuses);

const setServiceAction = (
  serviceName: string,
  next: Partial<ServiceActionState>,
) => {
  const current = serviceActions[serviceName] || {
    status: "idle",
    message: "",
  };
  serviceActions[serviceName] = { ...current, ...next };
};

const clearServiceMonitor = (serviceName: string) => {
  const monitor = serviceMonitors.get(serviceName);
  if (monitor) clearInterval(monitor);
  serviceMonitors.delete(serviceName);
};

const loadServices = async () => {
  if (!props.serverId) return;
  servicesLoading.value = true;
  servicesError.value = "";
  try {
    const res = await fetchServices(props.serverId);
    services.value = res.services;
  } catch (e: unknown) {
    servicesError.value = errorMessage(e);
  } finally {
    servicesLoading.value = false;
  }
};

const monitorServiceJob = (serviceName: string, jobId: string) => {
  clearServiceMonitor(serviceName);

  const monitor = setInterval(async () => {
    try {
      const job = await fetchJob(jobId);
      if (terminalStatuses.has(job.status as JobTerminalStatus)) {
        clearServiceMonitor(serviceName);

        if (job.status === "succeeded") {
          setServiceAction(serviceName, {
            status: "succeeded",
            message: "Service restarted",
            jobId,
          });
          loadServices();
          return;
        }

        const errorMessage = job.last_error
          ? `Failed: ${job.last_error}`
          : "Restart failed";

        setServiceAction(serviceName, {
          status: "failed",
          message: errorMessage,
          jobId,
        });
        return;
      }

      if (job.status === "running") {
        setServiceAction(serviceName, {
          status: "running",
          message: "Restarting...",
          jobId,
        });
      } else if (job.status === "queued") {
        setServiceAction(serviceName, {
          status: "queued",
          message: "Queued for restart",
          jobId,
        });
      }
    } catch (e: unknown) {
      clearServiceMonitor(serviceName);
      setServiceAction(serviceName, {
        status: "failed",
        message: errorMessage(e) || "Failed to fetch job status",
        jobId,
      });
    }
  }, 1200);

  serviceMonitors.set(serviceName, monitor);
};

const restartService = async (serviceName: string) => {
  if (!props.serverId) return;
  if (
    serviceActions[serviceName]?.status === "queued" ||
    serviceActions[serviceName]?.status === "running"
  ) {
    return;
  }
  setServiceAction(serviceName, {
    status: "queued",
    message: "Queuing restart...",
  });
  try {
    const job = await createJob({
      kind: "restart_service",
      server_id: props.serverId,
      payload: { service_name: serviceName },
    });
    setServiceAction(serviceName, {
      status: "queued",
      message: "Queued for restart",
      jobId: job.id,
    });
    monitorServiceJob(serviceName, job.id);
  } catch (e: unknown) {
    setServiceAction(serviceName, {
      status: "failed",
      message: errorMessage(e) || "Failed to restart service",
    });
  }
};

// Load services when agent connects
watch(
  () => props.agentConnected,
  (connected) => {
    if (connected) loadServices();
  },
  { immediate: true },
);

onUnmounted(() => {
  for (const monitor of serviceMonitors.values()) {
    clearInterval(monitor);
  }
  serviceMonitors.clear();
});
</script>

<template>
  <div class="space-y-4">
    <!-- Agent not connected state -->
    <div
      v-if="!agentConnected"
      class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center"
    >
      <h3 class="text-sm font-medium text-foreground">
        Agent not connected
      </h3>
      <p class="mt-1 text-sm text-muted-foreground">
        Service management requires an active agent connection.
      </p>
      <p class="mt-3 text-xs text-muted-foreground">
        Agent status: {{ agentStatusLabel }}
      </p>
    </div>

    <!-- Services list -->
    <template v-else>
      <div class="flex items-center justify-between">
        <p class="text-sm text-muted-foreground">
          Running systemd services on this server.
        </p>
        <Button
          variant="ghost"
          size="sm"
          :disabled="servicesLoading"
          class="text-muted-foreground hover:text-foreground"
          @click="loadServices"
        >
          <svg
            class="h-4 w-4"
            :class="{ 'animate-spin': servicesLoading }"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          Refresh
        </Button>
      </div>

      <div
        v-if="servicesError"
        class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ servicesError }}
      </div>

      <div
        v-if="servicesLoading && services.length === 0"
        class="flex items-center justify-center py-8"
      >
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

      <div
        v-else-if="services.length === 0"
        class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center"
      >
        <p class="text-sm text-muted-foreground">
          No services found.
        </p>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="service in services"
          :key="service.name"
          class="flex items-center justify-between rounded-lg border border-border/60 bg-card/40 px-4 py-3"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span
                class="h-2 w-2 rounded-full"
                :class="{
                  'bg-primary':
                    service.active_state === 'running',
                  'bg-muted-foreground':
                    service.active_state !== 'running',
                }"
              />
              <span
                class="font-mono text-sm font-medium text-foreground"
                >{{ service.name }}</span
              >
            </div>
            <p
              class="mt-0.5 truncate text-xs text-muted-foreground"
            >
              {{ service.description }}
            </p>
            <p
              v-if="serviceActions[service.name]?.message"
              class="mt-0.5 text-xs"
              :class="{
                'text-destructive':
                  serviceActions[service.name]?.status ===
                  'failed',
                'text-primary':
                  serviceActions[service.name]?.status ===
                  'succeeded',
                'text-muted-foreground':
                  serviceActions[service.name]?.status !==
                    'failed' &&
                  serviceActions[service.name]?.status !==
                    'succeeded',
              }"
            >
              {{ serviceActions[service.name]?.message }}
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            :disabled="
              serviceActions[service.name]?.status === 'queued' ||
              serviceActions[service.name]?.status === 'running'
            "
            class="ml-4 text-muted-foreground hover:text-foreground"
            @click="restartService(service.name)"
          >
            <svg
              class="h-4 w-4"
              :class="{
                'animate-spin':
                  serviceActions[service.name]?.status ===
                    'queued' ||
                  serviceActions[service.name]?.status ===
                    'running',
              }"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
            {{
              serviceActions[service.name]?.status === "queued" ||
              serviceActions[service.name]?.status === "running"
                ? "Restarting..."
                : "Restart"
            }}
          </Button>
        </div>
      </div>
    </template>
  </div>
</template>
