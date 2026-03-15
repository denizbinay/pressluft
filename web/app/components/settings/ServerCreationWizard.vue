<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogFooter,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { cn, errorMessage } from "@/lib/utils";
import { useProviders } from "~/composables/useProviders";
import { useServers, type ServerTypePrice } from "~/composables/useServers";
import type { Job } from "~/composables/useJobs";
import type { SupportLevel } from "~/lib/platform-contract.generated";

const props = defineProps<{
  isOpen: boolean;
}>();

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "created"): void;
}>();

const { providers } = useProviders();
const {
  profiles,
  catalog,
  saving,
  fetchCatalog,
  createServer,
  fetchServers,
} = useServers();

const formStep = ref<"configure" | "review" | "provisioning">("configure");
const formError = ref("");
const formLoadingCatalog = ref(false);

const formProviderId = ref("");
const formName = ref("");
const formLocation = ref("");
const formServerType = ref("");
const formProfileKey = ref("");

// Job tracking for provisioning step
const activeJobId = ref<number | null>(null);

const providerOptions = computed(() =>
  providers.value.map((p) => ({
    value: String(p.id),
    label: `${p.name} (${p.type})`,
  })),
);

const locationOptions = computed(() =>
  (catalog.value?.locations || []).map((loc) => ({
    value: loc.name,
    label: `${loc.name} - ${loc.description}`,
  })),
);

const selectableProfiles = computed(() =>
  profiles.value.filter((profile) => profile.support_level !== "unavailable"),
);

// Filter server types to only show those available at the selected location
const serverTypeOptions = computed(() => {
  const selectedLocation = formLocation.value;
  return (catalog.value?.server_types || [])
    .filter((type_) => {
      // If no location selected, show all
      if (!selectedLocation) return true;
      // Only show server types available at the selected location
      return type_.available_at?.includes(selectedLocation) ?? false;
    })
    .map((type_) => {
      const priceLabel = formatMonthlyPrice(type_, selectedLocation);
      const detail = `${type_.cores} vCPU · ${type_.memory_gb}GB RAM · ${type_.disk_gb}GB SSD`;
      return {
        value: type_.name,
        label: priceLabel
          ? `${type_.name} (${detail}, ${priceLabel}/mo)`
          : `${type_.name} (${detail})`,
      };
    });
});

const selectedProfile = computed(() =>
  profiles.value.find((profile) => profile.key === formProfileKey.value),
);

const defaultProfileKey = computed(
  () => selectableProfiles.value[0]?.key || "",
);

const selectedTypeLabel = computed(
  () =>
    serverTypeOptions.value.find(
      (option) => option.value === formServerType.value,
    )?.label || formServerType.value,
);

const buttonBaseClass =
  "rounded-lg focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const resetForm = () => {
  formStep.value = "configure";
  formError.value = "";
  formLoadingCatalog.value = false;
  formProviderId.value = providerOptions.value[0]?.value || "";
  formName.value = "";
  formLocation.value = "";
  formServerType.value = "";
  formProfileKey.value = defaultProfileKey.value;
  activeJobId.value = null;
};

const loadCatalogForSelectedProvider = async () => {
  if (!formProviderId.value) {
    return;
  }
  formLoadingCatalog.value = true;
  formError.value = "";
  try {
    await fetchCatalog(formProviderId.value);
    formLocation.value = locationOptions.value[0]?.value || "";
    // Server type will be set after location is selected (filtered by availability)
    formServerType.value = "";
    formProfileKey.value = defaultProfileKey.value;
  } catch (e: unknown) {
    formError.value = errorMessage(e);
  } finally {
    formLoadingCatalog.value = false;
  }
};

// Reset and load catalog when dialog opens
watch(
  () => props.isOpen,
  async (open) => {
    if (open) {
      resetForm();
      if (formProviderId.value) {
        await loadCatalogForSelectedProvider();
      }
    }
  },
);

watch(formProviderId, async () => {
  if (!props.isOpen) {
    return;
  }
  await loadCatalogForSelectedProvider();
});

// When location changes, reset server type to first available option
watch(formLocation, () => {
  formServerType.value = serverTypeOptions.value[0]?.value || "";
});

const goToReview = () => {
  if (!isFormValid()) {
    formError.value = "Please fill all required fields before continuing.";
    return;
  }
  formError.value = "";
  formStep.value = "review";
};

const goBack = () => {
  formStep.value = "configure";
};

const submit = async () => {
  if (!isFormValid()) {
    formError.value = "Please complete the form before creating a server.";
    return;
  }

  formError.value = "";
  try {
    const result = await createServer({
      provider_id: formProviderId.value,
      name: formName.value.trim(),
      location: formLocation.value,
      server_type: formServerType.value,
      profile_key: formProfileKey.value,
    });

    // Show provisioning progress instead of closing modal
    activeJobId.value = result.job_id;
    formStep.value = "provisioning";

    // Refresh server list in background
    fetchServers();
    emit("created");
  } catch (e: unknown) {
    formError.value = errorMessage(e);
  }
};

const handleJobCompleted = (_job: Job) => {
  // Refresh servers to show updated status
  fetchServers();
};

const handleJobFailed = (_job: Job, _error: string) => {
  // Refresh servers to show failed status
  fetchServers();
};

const closeAndReset = () => {
  emit("update:open", false);
  // Small delay to let modal animation complete before resetting
  setTimeout(resetForm, 200);
};

const handleClose = () => {
  emit("update:open", false);
};

const handleDialogUpdate = (value: boolean) => {
  emit("update:open", value);
};

const isFormValid = () => {
  return (
    !!formProviderId.value &&
    !!formName.value.trim() &&
    !!formLocation.value &&
    !!formServerType.value &&
    !!formProfileKey.value &&
    selectedProfile.value?.support_level !== "unavailable"
  );
};

const formatMonthlyPrice = (
  serverType: { prices: ReadonlyArray<ServerTypePrice> },
  location: string,
): string => {
  const price = serverType.prices.find(
    (entry) => entry.location_name === location,
  );
  if (!price) return "";

  const amount = Number(price.monthly_gross);
  if (Number.isNaN(amount)) {
    return `${price.monthly_gross} ${price.currency}`;
  }

  const formattedAmount = new Intl.NumberFormat("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);
  return `${formattedAmount} ${price.currency}`;
};
</script>

<template>
  <Dialog
    v-if="isOpen"
    :open="isOpen"
    @update:open="handleDialogUpdate"
  >
    <DialogScrollContent
      class="max-h-[85vh] border-border/60 bg-popover/90 p-5 text-popover-foreground sm:max-w-xl sm:p-6"
    >
      <DialogHeader class="text-left">
        <DialogTitle class="text-base font-semibold text-foreground">
          Create Managed Server
        </DialogTitle>
      </DialogHeader>

      <div class="space-y-4">
        <Alert
          v-if="formError"
          variant="destructive"
          class="border-destructive/30 bg-destructive/10 text-destructive"
        >
          <AlertDescription class="text-destructive">
            {{ formError }}
          </AlertDescription>
        </Alert>

        <template v-if="formStep === 'configure'">
          <WizardStepConfigure
            :form-provider-id="formProviderId"
            :form-name="formName"
            :form-profile-key="formProfileKey"
            :form-location="formLocation"
            :form-server-type="formServerType"
            :form-loading-catalog="formLoadingCatalog"
            :provider-options="providerOptions"
            :profiles="profiles"
            :selectable-profiles="selectableProfiles"
            :selected-profile="selectedProfile"
            :location-options="locationOptions"
            :server-type-options="serverTypeOptions"
            @update:form-provider-id="formProviderId = $event"
            @update:form-name="formName = $event"
            @update:form-profile-key="formProfileKey = $event"
            @update:form-location="formLocation = $event"
            @update:form-server-type="formServerType = $event"
          />
        </template>

        <template v-else-if="formStep === 'review'">
          <WizardStepReview
            :form-name="formName"
            :form-location="formLocation"
            :form-profile-key="formProfileKey"
            :selected-type-label="selectedTypeLabel"
            :selected-profile="selectedProfile"
          />
        </template>

        <!-- Provisioning step: show job timeline -->
        <template v-else-if="formStep === 'provisioning' && activeJobId">
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <div class="h-2 w-2 animate-pulse rounded-full bg-accent" />
              <span class="text-sm font-medium text-foreground/80"
                >Provisioning {{ formName }}</span
              >
            </div>
            <JobTimeline
              :job-id="activeJobId"
              compact
              @completed="handleJobCompleted"
              @failed="handleJobFailed"
            />
          </div>
        </template>
      </div>

      <DialogFooter class="flex-row items-center justify-end gap-2">
        <!-- Cancel/Close button -->
        <Button
          v-if="formStep !== 'provisioning'"
          variant="ghost"
          size="sm"
          :class="
            cn(buttonBaseClass, 'text-muted-foreground hover:bg-muted/50')
          "
          @click="handleClose"
        >
          Cancel
        </Button>

        <!-- Back button (review step only) -->
        <Button
          v-if="formStep === 'review'"
          variant="ghost"
          size="sm"
          :class="
            cn(buttonBaseClass, 'text-muted-foreground hover:bg-muted/50')
          "
          @click="goBack"
        >
          Back
        </Button>

        <!-- Review button (configure step) -->
        <Button
          v-if="formStep === 'configure'"
          size="sm"
          :disabled="!isFormValid() || formLoadingCatalog"
          :class="
            cn(
              buttonBaseClass,
              'bg-primary text-primary-foreground hover:bg-primary/90',
            )
          "
          @click="goToReview"
        >
          Review
        </Button>

        <!-- Create button (review step) -->
        <Button
          v-if="formStep === 'review'"
          size="sm"
          :disabled="saving"
          :class="
            cn(
              buttonBaseClass,
              'bg-primary text-primary-foreground hover:bg-primary/90',
            )
          "
          @click="submit"
        >
          <Spinner v-if="saving" class="text-primary-foreground" />
          Create Server
        </Button>

        <!-- Done button (provisioning step) -->
        <Button
          v-if="formStep === 'provisioning'"
          size="sm"
          :class="
            cn(
              buttonBaseClass,
              'bg-primary text-primary-foreground hover:bg-primary/90',
            )
          "
          @click="closeAndReset"
        >
          Done
        </Button>
      </DialogFooter>
    </DialogScrollContent>
  </Dialog>
</template>
