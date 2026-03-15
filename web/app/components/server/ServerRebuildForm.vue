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
import type { StoredServer } from "~/composables/useServers";
import { useJobs } from "~/composables/useJobs";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  serverId: string;
  server: StoredServer;
  settingsDisabled: boolean;
  imageOptions: ReadonlyArray<{ value: string; label: string }>;
  optionsLoading: boolean;
  optionsError: string;
}>();

const emit = defineEmits<{
  (e: "refresh"): void;
}>();

const { createJob } = useJobs();

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const rebuildServerImage = ref("");

const rebuildState = reactive({ loading: false, error: "", success: "" });

const normalizeText = (value: string) => value.trim();

const submitRebuild = async () => {
  if (!props.serverId) return;
  rebuildState.loading = true;
  rebuildState.error = "";
  rebuildState.success = "";
  try {
    const serverImage = normalizeText(rebuildServerImage.value);
    if (!serverImage) {
      throw new Error("Server image is required");
    }
    const job = await createJob({
      kind: "rebuild_server",
      server_id: props.serverId,
      payload: {
        server_image: serverImage,
      },
    });
    rebuildState.success = `Rebuild job #${job.id} queued`;
    emit("refresh");
  } catch (e: unknown) {
    rebuildState.error = errorMessage(e) || "Failed to create rebuild job";
  } finally {
    rebuildState.loading = false;
  }
};

// Initialize from server prop
watch(
  () => props.server,
  (value) => {
    if (!value) return;
    if (!rebuildServerImage.value) rebuildServerImage.value = value.image || "";
  },
  { immediate: true },
);

watch(
  () => props.imageOptions,
  (options) => {
    if (!rebuildServerImage.value && props.server?.image) {
      rebuildServerImage.value = props.server.image;
    }
    const firstOption = options[0];
    if (
      firstOption &&
      !options.some((option) => option.value === rebuildServerImage.value)
    ) {
      rebuildServerImage.value = firstOption.value;
    }
  },
);
</script>

<template>
  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
    <div>
      <h3 class="text-sm font-semibold text-foreground">
        Rebuild server
      </h3>
      <p class="mt-1 text-xs text-muted-foreground">
        Reinstall the server with a new image.
      </p>
    </div>
    <form class="mt-4 space-y-3" @submit.prevent="submitRebuild">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-muted-foreground">Server image</Label>
        <Select
          v-if="imageOptions.length"
          v-model="rebuildServerImage"
          :disabled="optionsLoading"
        >
          <SelectTrigger :class="controlClass">
            <SelectValue placeholder="Select image" />
          </SelectTrigger>
          <SelectContent class="border-border/60 bg-popover text-popover-foreground">
            <SelectItem
              v-for="option in imageOptions"
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
          v-model="rebuildServerImage"
          :disabled="true"
          placeholder="Current image"
        />
        <p v-if="optionsLoading" class="text-xs text-muted-foreground">
          Loading images...
        </p>
        <p v-else-if="!imageOptions.length" class="text-xs text-muted-foreground">
          Using current image only.
        </p>
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
            rebuildState.loading ||
            (!rebuildServerImage && !imageOptions.length)
          "
        >
          Queue rebuild job
        </Button>
        <span v-if="rebuildState.loading" class="text-xs text-muted-foreground">Submitting...</span>
        <span v-if="rebuildState.success" class="text-xs text-primary">{{ rebuildState.success }}</span>
        <span v-if="rebuildState.error" class="text-xs text-destructive">{{ rebuildState.error }}</span>
      </div>
    </form>
  </div>
</template>
