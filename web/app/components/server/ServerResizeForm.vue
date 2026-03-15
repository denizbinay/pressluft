<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import type { StoredServer } from "~/composables/useServers";
import { useJobs } from "~/composables/useJobs";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  serverId: string;
  server: StoredServer;
  settingsDisabled: boolean;
  serverTypeOptions: ReadonlyArray<{ value: string; label: string }>;
  optionsLoading: boolean;
  optionsError: string;
}>();

const emit = defineEmits<{
  (e: "refresh"): void;
}>();

const { createJob } = useJobs();

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const resizeServerType = ref("");
const resizeUpgradeDisk = ref(false);

const resizeState = reactive({ loading: false, error: "", success: "" });

const normalizeText = (value: string) => value.trim();

const submitResize = async () => {
  if (!props.serverId) return;
  resizeState.loading = true;
  resizeState.error = "";
  resizeState.success = "";
  try {
    const serverType = normalizeText(resizeServerType.value);
    if (!serverType) {
      throw new Error("Server type is required");
    }
    const job = await createJob({
      kind: "resize_server",
      server_id: props.serverId,
      payload: {
        server_type: serverType,
        upgrade_disk: resizeUpgradeDisk.value,
      },
    });
    resizeState.success = `Resize job #${job.id} queued`;
    emit("refresh");
  } catch (e: unknown) {
    resizeState.error = errorMessage(e) || "Failed to create resize job";
  } finally {
    resizeState.loading = false;
  }
};

// Initialize from server prop
watch(
  () => props.server,
  (value) => {
    if (!value) return;
    if (!resizeServerType.value) resizeServerType.value = value.server_type || "";
  },
  { immediate: true },
);

watch(
  () => props.serverTypeOptions,
  (options) => {
    if (!resizeServerType.value && props.server?.server_type) {
      resizeServerType.value = props.server.server_type;
    }
    const firstOption = options[0];
    if (
      firstOption &&
      !options.some((option) => option.value === resizeServerType.value)
    ) {
      resizeServerType.value = firstOption.value;
    }
  },
);
</script>

<template>
  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
    <div>
      <h3 class="text-sm font-semibold text-foreground">
        Resize server
      </h3>
      <p class="mt-1 text-xs text-muted-foreground">
        Update the server type and disk upgrade preference.
      </p>
    </div>
    <form class="mt-4 space-y-3" @submit.prevent="submitResize">
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Server type</Label>
          <Select
            v-if="serverTypeOptions.length"
            v-model="resizeServerType"
            :disabled="optionsLoading"
          >
            <SelectTrigger :class="controlClass">
              <SelectValue placeholder="Select server type" />
            </SelectTrigger>
            <SelectContent class="border-border/60 bg-popover text-popover-foreground">
              <SelectItem
                v-for="option in serverTypeOptions"
                :key="option.value"
                :value="option.value"
                class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
              >
                <SelectItemText>{{ option.label }}</SelectItemText>
              </SelectItem>
            </SelectContent>
          </Select>
          <Input
            v-else
            v-model="resizeServerType"
            :disabled="true"
            placeholder="Current server type"
          />
          <p v-if="optionsLoading" class="text-xs text-muted-foreground">
            Loading server types...
          </p>
          <p v-else-if="!serverTypeOptions.length" class="text-xs text-muted-foreground">
            Resize options unavailable.
          </p>
        </div>
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Upgrade disk</Label>
          <div class="flex items-center gap-2">
            <Switch v-model:checked="resizeUpgradeDisk" />
            <span class="text-xs text-muted-foreground">
              {{ resizeUpgradeDisk ? "Yes" : "No" }}
            </span>
          </div>
        </div>
      </div>
      <p v-if="optionsError" class="text-xs text-destructive">
        {{ optionsError }}
      </p>
      <div class="flex flex-wrap items-center gap-3">
        <Button
          type="submit"
          size="sm"
          :disabled="
            settingsDisabled ||
            resizeState.loading ||
            !serverTypeOptions.length
          "
        >
          Queue resize job
        </Button>
        <span v-if="resizeState.loading" class="text-xs text-muted-foreground">Submitting...</span>
        <span v-if="resizeState.success" class="text-xs text-primary">{{ resizeState.success }}</span>
        <span v-if="resizeState.error" class="text-xs text-destructive">{{ resizeState.error }}</span>
      </div>
    </form>
  </div>
</template>
