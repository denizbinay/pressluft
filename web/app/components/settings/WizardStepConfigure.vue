<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
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
import { cn } from "@/lib/utils";
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
  formProviderId: string;
  formName: string;
  formProfileKey: string;
  formLocation: string;
  formServerType: string;
  formLoadingCatalog: boolean;
  providerOptions: Array<{ value: string; label: string }>;
  profiles: ProfileOption[];
  selectableProfiles: ProfileOption[];
  selectedProfile: ProfileOption | undefined;
  locationOptions: Array<{ value: string; label: string }>;
  serverTypeOptions: Array<{ value: string; label: string }>;
}>();

const emit = defineEmits<{
  "update:formProviderId": [value: string];
  "update:formName": [value: string];
  "update:formProfileKey": [value: string];
  "update:formLocation": [value: string];
  "update:formServerType": [value: string];
}>();

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background";

const selectTriggerClass = cn(
  controlClass,
  "hover:border-border data-[placeholder]:text-muted-foreground",
);

const inputClass = cn(controlClass, "hover:border-border");

const selectContentClass =
  "border-border/60 bg-popover text-popover-foreground";

const selectItemClass =
  "text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground";

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

const selectedProfileStatusClass = computed(() =>
  props.selectedProfile
    ? profileSupportClass(props.selectedProfile.support_level)
    : "border-border/60 bg-muted/40 text-muted-foreground",
);

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
  <div class="space-y-1.5">
    <Label class="text-sm font-medium text-muted-foreground">
      Provider Connection
    </Label>
    <Select
      :model-value="formProviderId"
      @update:model-value="emit('update:formProviderId', String($event))"
    >
      <SelectTrigger :class="selectTriggerClass">
        <SelectValue placeholder="Select provider" />
      </SelectTrigger>
      <SelectContent :class="selectContentClass">
        <SelectItem
          v-for="option in providerOptions"
          :key="option.value"
          :value="option.value"
          :class="selectItemClass"
        >
          <SelectItemText>{{ option.label }}</SelectItemText>
        </SelectItem>
      </SelectContent>
    </Select>
  </div>

  <div class="space-y-1.5">
    <Label class="text-sm font-medium text-muted-foreground">
      Server Name
    </Label>
    <Input
      :model-value="formName"
      placeholder="e.g. agency-prod-eu-1"
      :class="inputClass"
      @update:model-value="emit('update:formName', String($event))"
    />
  </div>

  <div class="space-y-1.5">
    <Label class="text-sm font-medium text-muted-foreground">
      Server Profile
    </Label>
    <Select
      :model-value="formProfileKey"
      :disabled="formLoadingCatalog"
      @update:model-value="emit('update:formProfileKey', String($event))"
    >
      <SelectTrigger :class="selectTriggerClass">
        <SelectValue placeholder="Select server profile">
          <div
            v-if="selectedProfile"
            class="flex min-w-0 items-center justify-between gap-2"
          >
            <span class="truncate text-sm text-foreground">{{
              selectedProfile.name
            }}</span>
            <Badge
              :class="profileSupportClass(selectedProfile.support_level)"
            >
              {{ supportLevelLabel(selectedProfile.support_level) }}
            </Badge>
          </div>
        </SelectValue>
      </SelectTrigger>
      <SelectContent :class="selectContentClass">
        <SelectItem
          v-for="profile in profiles"
          :key="profile.key"
          :value="profile.key"
          :disabled="profile.support_level === 'unavailable'"
          :class="selectItemClass"
        >
          <div
            class="flex w-full min-w-0 items-center justify-between gap-2"
          >
            <span class="truncate">{{ profile.name }}</span>
            <Badge :class="profileSupportClass(profile.support_level)">
              {{ supportLevelLabel(profile.support_level) }}
            </Badge>
          </div>
        </SelectItem>
      </SelectContent>
    </Select>
    <div
      v-if="selectedProfile"
      class="rounded-lg border px-3 py-3"
      :class="selectedProfileStatusClass"
    >
      <ul class="space-y-2 text-sm text-muted-foreground">
        <li class="flex items-start gap-2">
          <span
            class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70"
          />
          <span>
            <strong class="text-foreground">Description:</strong>
            {{ selectedProfile.description }}
          </span>
        </li>
        <li class="flex items-start gap-2">
          <span
            class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70"
          />
          <span>
            <strong class="text-foreground">Configure guarantee:</strong>
            {{ selectedProfile.configure_guarantee }}
          </span>
        </li>
        <li class="flex items-start gap-2">
          <span
            class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70"
          />
          <span>
            <strong class="text-foreground">Support:</strong>
            {{ selectedProfileSupportText }}
          </span>
        </li>
      </ul>
    </div>
    <p v-else class="text-xs text-muted-foreground">
      No selectable server profile is available for this provider yet.
    </p>
  </div>

  <div class="space-y-1.5">
    <Label class="text-sm font-medium text-muted-foreground">
      Region
    </Label>
    <Select
      :model-value="formLocation"
      :disabled="formLoadingCatalog"
      @update:model-value="emit('update:formLocation', String($event))"
    >
      <SelectTrigger :class="selectTriggerClass">
        <SelectValue placeholder="Select region" />
      </SelectTrigger>
      <SelectContent :class="selectContentClass">
        <SelectItem
          v-for="option in locationOptions"
          :key="option.value"
          :value="option.value"
          :class="selectItemClass"
        >
          <SelectItemText>{{ option.label }}</SelectItemText>
        </SelectItem>
      </SelectContent>
    </Select>
  </div>

  <div class="space-y-1.5">
    <Label class="text-sm font-medium text-muted-foreground">
      Size
    </Label>
    <Select
      :model-value="formServerType"
      :disabled="formLoadingCatalog || !formLocation"
      @update:model-value="emit('update:formServerType', String($event))"
    >
      <SelectTrigger :class="selectTriggerClass">
        <SelectValue placeholder="Select size" />
      </SelectTrigger>
      <SelectContent :class="selectContentClass">
        <SelectItem
          v-for="option in serverTypeOptions"
          :key="option.value"
          :value="option.value"
          :class="selectItemClass"
        >
          <SelectItemText>{{ option.label }}</SelectItemText>
        </SelectItem>
      </SelectContent>
    </Select>
  </div>

  <p v-if="formLoadingCatalog" class="text-xs text-muted-foreground">
    Loading provider catalog...
  </p>
</template>
