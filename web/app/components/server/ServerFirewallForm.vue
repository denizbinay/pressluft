<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useJobs } from "~/composables/useJobs";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  serverId: string;
  settingsDisabled: boolean;
  firewallOptions: ReadonlyArray<{ value: string; label: string }>;
  optionsLoading: boolean;
  optionsError: string;
}>();

const { createJob } = useJobs();

const firewallsSelected = ref<string[]>([]);
const firewallsCustom = ref("");
const firewallsUseCustom = ref(false);

const firewallsState = reactive({ loading: false, error: "", success: "" });

const normalizeList = (value: string) =>
  value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

const showFirewallCustomInput = computed(
  () => props.firewallOptions.length === 0 || firewallsUseCustom.value,
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
</script>

<template>
  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
    <div>
      <h3 class="text-sm font-semibold text-foreground">
        Update firewalls
      </h3>
      <p class="mt-1 text-xs text-muted-foreground">
        Replace firewall assignments using firewall names or IDs.
      </p>
    </div>
    <form class="mt-4 space-y-3" @submit.prevent="submitUpdateFirewalls">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-muted-foreground">Firewalls</Label>
        <div v-if="firewallOptions.length" class="grid gap-2 sm:grid-cols-2">
          <label
            v-for="option in firewallOptions"
            :key="option.value"
            class="flex items-start gap-2 rounded-lg border border-border/60 bg-background/50 px-3 py-2 text-sm text-foreground/80"
          >
            <input
              type="checkbox"
              class="mt-0.5 h-4 w-4 rounded border-border/60 text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
              :value="option.value"
              :checked="firewallsSelected.includes(option.value)"
              @change="toggleFirewallSelection(option.value)"
            />
            <span>{{ option.label }}</span>
          </label>
        </div>
        <p v-else class="text-xs text-muted-foreground">
          No firewall options available.
        </p>
      </div>
      <div v-if="firewallOptions.length" class="flex items-center gap-2">
        <Switch v-model:checked="firewallsUseCustom" />
        <span class="text-xs text-muted-foreground">Add custom firewall IDs</span>
      </div>
      <div v-if="showFirewallCustomInput" class="space-y-1.5">
        <Label class="text-xs font-medium text-muted-foreground">Custom firewalls</Label>
        <Input v-model="firewallsCustom" placeholder="fw-core, fw-web" />
        <p class="text-xs text-muted-foreground">Comma-separated list.</p>
      </div>
      <p v-if="optionsLoading" class="text-xs text-muted-foreground">
        Loading firewalls...
      </p>
      <p v-if="optionsError" class="text-xs text-destructive">
        {{ optionsError }}
      </p>
      <div class="flex flex-wrap items-center gap-3">
        <Button
          type="submit"
          size="sm"
          :disabled="settingsDisabled || firewallsState.loading"
        >
          Create firewall update job
        </Button>
        <span v-if="firewallsState.loading" class="text-xs text-muted-foreground">Submitting...</span>
        <span v-if="firewallsState.success" class="text-xs text-primary">{{ firewallsState.success }}</span>
        <span v-if="firewallsState.error" class="text-xs text-destructive">{{ firewallsState.error }}</span>
      </div>
    </form>
  </div>
</template>
