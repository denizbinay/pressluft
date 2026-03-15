<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import type { SupportLevel } from "~/lib/platform-contract.generated";

interface ProfileOption {
  key: string;
  name: string;
  description: string;
  configure_guarantee: string;
  support_level: SupportLevel;
  support_reason?: string;
}

const props = defineProps<{
  formName: string;
  formLocation: string;
  formProfileKey: string;
  selectedTypeLabel: string;
  selectedProfile: ProfileOption | undefined;
}>();

const supportLevelLabel = (supportLevel: SupportLevel): string => {
  if (supportLevel === "supported") return "Supported";
  if (supportLevel === "experimental") return "Experimental";
  return "Not Ready";
};

const profileSupportClass = (supportLevel: SupportLevel): string => {
  if (supportLevel === "supported")
    return "border-primary/30 bg-primary/10 text-primary";
  if (supportLevel === "experimental")
    return "border-accent/30 bg-accent/10 text-accent";
  return "border-border/60 bg-muted/60 text-muted-foreground";
};

const selectedProfileSupportText = computed(() => {
  if (!props.selectedProfile) {
    return "";
  }
  return (
    props.selectedProfile.support_reason ||
    supportLevelLabel(props.selectedProfile.support_level)
  );
});
</script>

<template>
  <div
    class="rounded-lg border border-border/60 bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
  >
    <p>
      <strong class="text-foreground">Name:</strong> {{ formName }}
    </p>
    <p>
      <strong class="text-foreground">Region:</strong>
      {{ formLocation }}
    </p>
    <p>
      <strong class="text-foreground">Size:</strong>
      {{ selectedTypeLabel }}
    </p>
    <div class="mt-1">
      <p>
        <strong class="text-foreground">Profile:</strong>
        {{ selectedProfile?.name || formProfileKey }}
      </p>
      <div
        v-if="selectedProfile"
        class="mt-2 flex flex-wrap items-center gap-2"
      >
        <Badge
          :class="profileSupportClass(selectedProfile.support_level)"
        >
          {{ supportLevelLabel(selectedProfile.support_level) }}
        </Badge>
      </div>
      <p
        v-if="selectedProfile"
        class="mt-2 text-xs text-muted-foreground"
      >
        {{ selectedProfileSupportText }}
      </p>
    </div>
  </div>
  <p class="text-xs text-muted-foreground">
    The base image is determined by the selected profile. Advanced
    networking, firewalls, and storage options are intentionally
    hidden for this managed flow.
  </p>
</template>
