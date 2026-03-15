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
  volumeOptions: ReadonlyArray<{ value: string; label: string; size_gb?: number }>;
  optionsLoading: boolean;
  optionsError: string;
}>();

const { createJob } = useJobs();

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const volumeName = ref("");
const volumeSizeGb = ref("");
const volumeState = ref("present");
const volumeAutomount = ref(false);

const volumeStateUi = reactive({ loading: false, error: "", success: "" });

const normalizeText = (value: string) => value.trim();

const selectedVolume = computed(() =>
  props.volumeOptions.find((option) => option.value === volumeName.value),
);

const submitManageVolume = async () => {
  if (!props.serverId) return;
  volumeStateUi.loading = true;
  volumeStateUi.error = "";
  volumeStateUi.success = "";
  try {
    const name = normalizeText(volumeName.value);
    const state = normalizeText(volumeState.value);
    const sizeGb = Number.parseInt(volumeSizeGb.value, 10);
    if (!name || !state) {
      throw new Error("Volume name and state are required");
    }
    if (state === "present") {
      if (!Number.isFinite(sizeGb) || sizeGb <= 0) {
        throw new Error("Size must be a positive number");
      }
    }
    const payload: Record<string, unknown> = {
      volume_name: name,
      state,
    };
    if (state === "present") {
      payload.size_gb = sizeGb;
      payload.automount = volumeAutomount.value;
    }
    const job = await createJob({
      kind: "manage_volume",
      server_id: props.serverId,
      payload,
    });
    volumeStateUi.success = `Job #${job.id} created`;
  } catch (e: unknown) {
    volumeStateUi.error = errorMessage(e) || "Failed to create volume job";
  } finally {
    volumeStateUi.loading = false;
  }
};

watch(
  () => props.volumeOptions,
  (options) => {
    const firstOption = options[0];
    if (!volumeName.value && firstOption) {
      volumeName.value = firstOption.value;
    }
  },
);

watch([selectedVolume, volumeState], ([option, state]) => {
  if (state !== "present" || !option?.size_gb) return;
  const current = Number.parseInt(volumeSizeGb.value, 10);
  if (!Number.isFinite(current) || current < option.size_gb) {
    volumeSizeGb.value = String(option.size_gb);
  }
});
</script>

<template>
  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
    <div>
      <h3 class="text-sm font-semibold text-foreground">
        Manage volume
      </h3>
      <p class="mt-1 text-xs text-muted-foreground">
        Create &amp; attach a volume or delete an existing one.
      </p>
    </div>
    <form class="mt-4 space-y-3" @submit.prevent="submitManageVolume">
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Volume</Label>
          <Select
            v-if="volumeOptions.length"
            v-model="volumeName"
            :disabled="optionsLoading"
          >
            <SelectTrigger :class="controlClass">
              <SelectValue placeholder="Select volume" />
            </SelectTrigger>
            <SelectContent class="border-border/60 bg-popover text-popover-foreground">
              <SelectItem
                v-for="option in volumeOptions"
                :key="option.value"
                :value="option.value"
                class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
              >
                <SelectItemText>{{ option.label }}</SelectItemText>
              </SelectItem>
            </SelectContent>
          </Select>
          <div v-else class="space-y-1">
            <Input v-model="volumeName" placeholder="data-volume" />
            <p class="text-xs text-muted-foreground">No volume list available.</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Action</Label>
          <Select v-model="volumeState">
            <SelectTrigger :class="controlClass">
              <SelectValue placeholder="Select state" />
            </SelectTrigger>
            <SelectContent class="border-border/60 bg-popover text-popover-foreground">
              <SelectItem
                value="present"
                class="text-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
              >
                <SelectItemText>Create &amp; attach</SelectItemText>
              </SelectItem>
              <SelectItem
                value="absent"
                class="text-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
              >
                <SelectItemText>Delete volume</SelectItemText>
              </SelectItem>
            </SelectContent>
          </Select>
          <p v-if="volumeState === 'absent'" class="text-xs text-destructive">
            This deletes the volume from your cloud provider.
          </p>
        </div>
        <div v-if="volumeState === 'present'" class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Size (GB)</Label>
          <Input
            v-model="volumeSizeGb"
            type="number"
            :min="selectedVolume?.size_gb || 1"
            placeholder="50"
          />
          <p v-if="selectedVolume?.size_gb" class="text-xs text-muted-foreground">
            Existing volume size is {{ selectedVolume.size_gb }}GB. You can only increase it.
          </p>
        </div>
        <div v-if="volumeState === 'present'" class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Location</Label>
          <div class="rounded-lg border border-border/60 bg-background/50 px-3 py-2 text-sm text-foreground/80">
            {{ server.location }}
          </div>
          <p class="text-xs text-muted-foreground">
            Volumes are created in the server location.
          </p>
        </div>
        <div v-if="volumeState === 'present'" class="space-y-1.5">
          <Label class="text-xs font-medium text-muted-foreground">Automount</Label>
          <div class="flex items-center gap-2">
            <Switch v-model:checked="volumeAutomount" />
            <span class="text-xs text-muted-foreground">
              {{ volumeAutomount ? "Yes" : "No" }}
            </span>
          </div>
        </div>
      </div>
      <p v-if="optionsLoading" class="text-xs text-muted-foreground">
        Loading volume options...
      </p>
      <p v-if="optionsError" class="text-xs text-destructive">
        {{ optionsError }}
      </p>
      <div class="flex flex-wrap items-center gap-3">
        <Button
          type="submit"
          size="sm"
          :disabled="settingsDisabled || volumeStateUi.loading"
        >
          Create volume job
        </Button>
        <span v-if="volumeStateUi.loading" class="text-xs text-muted-foreground">Submitting...</span>
        <span v-if="volumeStateUi.success" class="text-xs text-primary">{{ volumeStateUi.success }}</span>
        <span v-if="volumeStateUi.error" class="text-xs text-destructive">{{ volumeStateUi.error }}</span>
      </div>
    </form>
  </div>
</template>
