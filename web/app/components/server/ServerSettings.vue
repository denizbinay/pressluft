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
import { useServers, type StoredServer } from "~/composables/useServers";
import { useJobs } from "~/composables/useJobs";
import { useServerOptions } from "~/composables/useServerOptions";
import { errorMessage } from "~/lib/utils";

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

const { fetchServer } = useServers();
const { createJob } = useJobs();

const {
  images: imageOptions,
  serverTypes: serverTypeOptions,
  firewalls: firewallOptions,
  volumes: volumeOptions,
  loading: optionsLoading,
  error: optionsError,
  fetchAll: fetchServerOptions,
} = useServerOptions();

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const rebuildServerImage = ref("");
const resizeServerType = ref("");
const resizeUpgradeDisk = ref(false);
const firewallsSelected = ref<string[]>([]);
const firewallsCustom = ref("");
const firewallsUseCustom = ref(false);
const volumeName = ref("");
const volumeSizeGb = ref("");
const volumeState = ref("present");
const volumeAutomount = ref(false);

const rebuildState = reactive({ loading: false, error: "", success: "" });
const resizeState = reactive({ loading: false, error: "", success: "" });
const firewallsState = reactive({ loading: false, error: "", success: "" });
const volumeStateUi = reactive({ loading: false, error: "", success: "" });

const normalizeText = (value: string) => value.trim();
const normalizeList = (value: string) =>
  value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

const showFirewallCustomInput = computed(
  () => firewallOptions.value.length === 0 || firewallsUseCustom.value,
);

const toggleFirewallSelection = (value: string) => {
  if (firewallsSelected.value.includes(value)) {
    firewallsSelected.value = firewallsSelected.value.filter(
      (item) => item !== value,
    );
    return;
  }
  firewallsSelected.value = [...firewallsSelected.value, value];
};

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

const submitUpdateFirewalls = async () => {
  if (!props.serverId) return;
  firewallsState.loading = true;
  firewallsState.error = "";
  firewallsState.success = "";
  try {
    const customFirewalls = showFirewallCustomInput.value
      ? normalizeList(firewallsCustom.value)
      : [];
    const firewalls = Array.from(
      new Set([...firewallsSelected.value, ...customFirewalls]),
    );
    if (firewalls.length === 0) {
      throw new Error("At least one firewall is required");
    }
    const job = await createJob({
      kind: "update_firewalls",
      server_id: props.serverId,
      payload: { firewalls },
    });
    firewallsState.success = `Job #${job.id} created`;
  } catch (e: unknown) {
    firewallsState.error = errorMessage(e) || "Failed to create firewall update job";
  } finally {
    firewallsState.loading = false;
  }
};

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

const selectedVolume = computed(() =>
  volumeOptions.value.find((option) => option.value === volumeName.value),
);

// Initialize form values from server prop
watch(
  () => props.server,
  (value) => {
    if (!value) return;
    if (!rebuildServerImage.value) rebuildServerImage.value = value.image || "";
    if (!resizeServerType.value) resizeServerType.value = value.server_type || "";
  },
  { immediate: true },
);

watch(imageOptions, (options) => {
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
});

watch(serverTypeOptions, (options) => {
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
});

watch(volumeOptions, (options) => {
  const firstOption = options[0];
  if (!volumeName.value && firstOption) {
    volumeName.value = firstOption.value;
  }
});

watch([selectedVolume, volumeState], ([option, state]) => {
  if (state !== "present" || !option?.size_gb) return;
  const current = Number.parseInt(volumeSizeGb.value, 10);
  if (!Number.isFinite(current) || current < option.size_gb) {
    volumeSizeGb.value = String(option.size_gb);
  }
});

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
      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-4"
      >
        <div>
          <h3 class="text-sm font-semibold text-foreground">
            Rebuild server
          </h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Reinstall the server with a new image.
          </p>
        </div>
        <form
          class="mt-4 space-y-3"
          @submit.prevent="submitRebuild"
        >
          <div class="space-y-1.5">
            <Label
              class="text-xs font-medium text-muted-foreground"
              >Server image</Label
            >
            <Select
              v-if="imageOptions.length"
              v-model="rebuildServerImage"
              :disabled="optionsLoading"
            >
              <SelectTrigger :class="controlClass">
                <SelectValue placeholder="Select image" />
              </SelectTrigger>
              <SelectContent
                class="border-border/60 bg-popover text-popover-foreground"
              >
                <SelectItem
                  v-for="option in imageOptions"
                  :key="option.value"
                  :value="option.value"
                  class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                >
                  <SelectItemText>{{
                    option.label
                  }}</SelectItemText>
                </SelectItem>
              </SelectContent>
            </Select>
            <Input
              v-else
              v-model="rebuildServerImage"
              :disabled="true"
              placeholder="Current image"
            />
            <p
              v-if="optionsLoading"
              class="text-xs text-muted-foreground"
            >
              Loading images...
            </p>
            <p
              v-else-if="!imageOptions.length"
              class="text-xs text-muted-foreground"
            >
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
            <span
              v-if="rebuildState.loading"
              class="text-xs text-muted-foreground"
              >Submitting...</span
            >
            <span
              v-if="rebuildState.success"
              class="text-xs text-primary"
              >{{ rebuildState.success }}</span
            >
            <span
              v-if="rebuildState.error"
              class="text-xs text-destructive"
              >{{ rebuildState.error }}</span
            >
          </div>
        </form>
      </div>

      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-4"
      >
        <div>
          <h3 class="text-sm font-semibold text-foreground">
            Resize server
          </h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Update the server type and disk upgrade preference.
          </p>
        </div>
        <form
          class="mt-4 space-y-3"
          @submit.prevent="submitResize"
        >
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Server type</Label
              >
              <Select
                v-if="serverTypeOptions.length"
                v-model="resizeServerType"
                :disabled="optionsLoading"
              >
                <SelectTrigger :class="controlClass">
                  <SelectValue placeholder="Select server type" />
                </SelectTrigger>
                <SelectContent
                  class="border-border/60 bg-popover text-popover-foreground"
                >
                  <SelectItem
                    v-for="option in serverTypeOptions"
                    :key="option.value"
                    :value="option.value"
                    class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                  >
                    <SelectItemText>{{
                      option.label
                    }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Input
                v-else
                v-model="resizeServerType"
                :disabled="true"
                placeholder="Current server type"
              />
              <p
                v-if="optionsLoading"
                class="text-xs text-muted-foreground"
              >
                Loading server types...
              </p>
              <p
                v-else-if="!serverTypeOptions.length"
                class="text-xs text-muted-foreground"
              >
                Resize options unavailable.
              </p>
            </div>
            <div class="space-y-1.5">
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Upgrade disk</Label
              >
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
            <span
              v-if="resizeState.loading"
              class="text-xs text-muted-foreground"
              >Submitting...</span
            >
            <span
              v-if="resizeState.success"
              class="text-xs text-primary"
              >{{ resizeState.success }}</span
            >
            <span
              v-if="resizeState.error"
              class="text-xs text-destructive"
              >{{ resizeState.error }}</span
            >
          </div>
        </form>
      </div>

      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-4"
      >
        <div>
          <h3 class="text-sm font-semibold text-foreground">
            Update firewalls
          </h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Replace firewall assignments using firewall names or
            IDs.
          </p>
        </div>
        <form
          class="mt-4 space-y-3"
          @submit.prevent="submitUpdateFirewalls"
        >
          <div class="space-y-1.5">
            <Label
              class="text-xs font-medium text-muted-foreground"
              >Firewalls</Label
            >
            <div
              v-if="firewallOptions.length"
              class="grid gap-2 sm:grid-cols-2"
            >
              <label
                v-for="option in firewallOptions"
                :key="option.value"
                class="flex items-start gap-2 rounded-lg border border-border/60 bg-background/50 px-3 py-2 text-sm text-foreground/80"
              >
                <input
                  type="checkbox"
                  class="mt-0.5 h-4 w-4 rounded border-border/60 text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
                  :value="option.value"
                  :checked="
                    firewallsSelected.includes(option.value)
                  "
                  @change="toggleFirewallSelection(option.value)"
                />
                <span>{{ option.label }}</span>
              </label>
            </div>
            <p v-else class="text-xs text-muted-foreground">
              No firewall options available.
            </p>
          </div>
          <div
            v-if="firewallOptions.length"
            class="flex items-center gap-2"
          >
            <Switch v-model:checked="firewallsUseCustom" />
            <span class="text-xs text-muted-foreground"
              >Add custom firewall IDs</span
            >
          </div>
          <div v-if="showFirewallCustomInput" class="space-y-1.5">
            <Label
              class="text-xs font-medium text-muted-foreground"
              >Custom firewalls</Label
            >
            <Input
              v-model="firewallsCustom"
              placeholder="fw-core, fw-web"
            />
            <p class="text-xs text-muted-foreground">
              Comma-separated list.
            </p>
          </div>
          <p
            v-if="optionsLoading"
            class="text-xs text-muted-foreground"
          >
            Loading firewalls...
          </p>
          <p v-if="optionsError" class="text-xs text-destructive">
            {{ optionsError }}
          </p>
          <div class="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              size="sm"
              :disabled="
                settingsDisabled || firewallsState.loading
              "
            >
              Create firewall update job
            </Button>
            <span
              v-if="firewallsState.loading"
              class="text-xs text-muted-foreground"
              >Submitting...</span
            >
            <span
              v-if="firewallsState.success"
              class="text-xs text-primary"
              >{{ firewallsState.success }}</span
            >
            <span
              v-if="firewallsState.error"
              class="text-xs text-destructive"
              >{{ firewallsState.error }}</span
            >
          </div>
        </form>
      </div>

      <div
        class="rounded-lg border border-border/60 bg-card/40 px-4 py-4"
      >
        <div>
          <h3 class="text-sm font-semibold text-foreground">
            Manage volume
          </h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Create & attach a volume or delete an existing one.
          </p>
        </div>
        <form
          class="mt-4 space-y-3"
          @submit.prevent="submitManageVolume"
        >
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Volume</Label
              >
              <Select
                v-if="volumeOptions.length"
                v-model="volumeName"
                :disabled="optionsLoading"
              >
                <SelectTrigger :class="controlClass">
                  <SelectValue placeholder="Select volume" />
                </SelectTrigger>
                <SelectContent
                  class="border-border/60 bg-popover text-popover-foreground"
                >
                  <SelectItem
                    v-for="option in volumeOptions"
                    :key="option.value"
                    :value="option.value"
                    class="text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                  >
                    <SelectItemText>{{
                      option.label
                    }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
              <div v-else class="space-y-1">
                <Input
                  v-model="volumeName"
                  placeholder="data-volume"
                />
                <p class="text-xs text-muted-foreground">
                  No volume list available.
                </p>
              </div>
            </div>
            <div class="space-y-1.5">
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Action</Label
              >
              <Select v-model="volumeState">
                <SelectTrigger :class="controlClass">
                  <SelectValue placeholder="Select state" />
                </SelectTrigger>
                <SelectContent
                  class="border-border/60 bg-popover text-popover-foreground"
                >
                  <SelectItem
                    value="present"
                    class="text-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                  >
                    <SelectItemText
                      >Create &amp; attach</SelectItemText
                    >
                  </SelectItem>
                  <SelectItem
                    value="absent"
                    class="text-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
                  >
                    <SelectItemText>Delete volume</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
              <p
                v-if="volumeState === 'absent'"
                class="text-xs text-destructive"
              >
                This deletes the volume from your cloud provider.
              </p>
            </div>
            <div
              v-if="volumeState === 'present'"
              class="space-y-1.5"
            >
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Size (GB)</Label
              >
              <Input
                v-model="volumeSizeGb"
                type="number"
                :min="selectedVolume?.size_gb || 1"
                placeholder="50"
              />
              <p
                v-if="selectedVolume?.size_gb"
                class="text-xs text-muted-foreground"
              >
                Existing volume size is
                {{ selectedVolume.size_gb }}GB. You can only
                increase it.
              </p>
            </div>
            <div
              v-if="volumeState === 'present'"
              class="space-y-1.5"
            >
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Location</Label
              >
              <div
                class="rounded-lg border border-border/60 bg-background/50 px-3 py-2 text-sm text-foreground/80"
              >
                {{ server.location }}
              </div>
              <p class="text-xs text-muted-foreground">
                Volumes are created in the server location.
              </p>
            </div>
            <div
              v-if="volumeState === 'present'"
              class="space-y-1.5"
            >
              <Label
                class="text-xs font-medium text-muted-foreground"
                >Automount</Label
              >
              <div class="flex items-center gap-2">
                <Switch v-model:checked="volumeAutomount" />
                <span class="text-xs text-muted-foreground">
                  {{ volumeAutomount ? "Yes" : "No" }}
                </span>
              </div>
            </div>
          </div>
          <p
            v-if="optionsLoading"
            class="text-xs text-muted-foreground"
          >
            Loading volume options...
          </p>
          <p v-if="optionsError" class="text-xs text-destructive">
            {{ optionsError }}
          </p>
          <div class="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              size="sm"
              :disabled="
                settingsDisabled || volumeStateUi.loading
              "
            >
              Create volume job
            </Button>
            <span
              v-if="volumeStateUi.loading"
              class="text-xs text-muted-foreground"
              >Submitting...</span
            >
            <span
              v-if="volumeStateUi.success"
              class="text-xs text-primary"
              >{{ volumeStateUi.success }}</span
            >
            <span
              v-if="volumeStateUi.error"
              class="text-xs text-destructive"
              >{{ volumeStateUi.error }}</span
            >
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
