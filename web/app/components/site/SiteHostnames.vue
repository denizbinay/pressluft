<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { StoredDomain } from "~/composables/useDomains";

const props = defineProps<{
  domains: StoredDomain[];
  currentServerIPv4: string;
  saving: boolean;
}>();

const emit = defineEmits<{
  (e: "assign", payload: { hostname: string; source: string; is_primary: boolean }): void;
  (e: "set-primary", domainId: string): void;
  (e: "remove", domain: StoredDomain): void;
}>();

const hostnameForm = reactive({
  source: "preview",
  fallbackLabel: "",
  hostname: "",
});

const normalizeLabel = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/_/g, "-")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "");

const buildFallbackHostname = () => {
  const label = normalizeLabel(hostnameForm.fallbackLabel);
  if (!label || !props.currentServerIPv4) {
    return "";
  }
  return `${label}.${props.currentServerIPv4.replace(/\./g, "-")}.sslip.io`;
};

const handleAssign = () => {
  const payload =
    hostnameForm.source === "preview"
      ? {
          hostname: buildFallbackHostname(),
          source: "fallback_resolver",
          is_primary: props.domains.length === 0,
        }
      : {
          hostname: hostnameForm.hostname.trim(),
          source: "user",
          is_primary: props.domains.length === 0,
        };
  emit("assign", payload);
  hostnameForm.fallbackLabel = "";
  hostnameForm.hostname = "";
};

const isAssignDisabled = computed(() => {
  if (props.saving) return true;
  if (hostnameForm.source === "preview") return !buildFallbackHostname();
  return !hostnameForm.hostname.trim();
});

// If server has no IPv4 and preview is selected, switch to domain
watch(
  () => props.currentServerIPv4,
  (value) => {
    if (!value && hostnameForm.source === "preview") {
      hostnameForm.source = "domain";
    }
  },
);

// Also check on mount
onMounted(() => {
  if (!props.currentServerIPv4 && hostnameForm.source === "preview") {
    hostnameForm.source = "domain";
  }
});
</script>

<template>
  <div class="space-y-6">
    <div v-if="domains.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">No hostnames assigned yet.</div>
    <div v-else class="space-y-3">
      <div v-for="domain in domains" :key="domain.id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <p class="text-sm font-semibold text-foreground">{{ domain.hostname }}</p>
              <Badge v-if="domain.is_primary" variant="outline" class="border-primary/30 bg-primary/10 text-primary">Primary</Badge>
              <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domain.source === 'fallback_resolver' ? 'Preview URL' : domain.parent_domain_id ? 'Child hostname' : 'Exact hostname' }}</Badge>
              <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">DNS {{ domain.dns_state }}</Badge>
              <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">Routing {{ domain.routing_state }}</Badge>
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ domain.parent_hostname || (domain.source === "fallback_resolver" ? "Pressluft generated this preview URL from the current server IP." : "User-managed hostname record.") }}
            </p>
            <p v-if="domain.routing_status_message || domain.dns_status_message" class="mt-2 text-xs text-muted-foreground">
              {{ domain.routing_status_message || domain.dns_status_message }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Button v-if="!domain.is_primary" type="button" variant="outline" size="sm" @click="emit('set-primary', domain.id)">Make primary</Button>
            <Button type="button" variant="ghost" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" @click="emit('remove', domain)">Remove</Button>
          </div>
        </div>
      </div>
    </div>

    <div class="rounded-2xl border border-border/60 bg-muted/20 p-4">
      <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Attach hostname</p>
      <div class="mt-4 space-y-4">
        <div class="space-y-1.5">
          <Label class="text-sm font-medium text-muted-foreground">Destination</Label>
          <select v-model="hostnameForm.source" class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40">
            <option value="preview" :disabled="!currentServerIPv4">Preview URL</option>
            <option value="domain">Exact hostname</option>
          </select>
        </div>

        <template v-if="hostnameForm.source === 'preview'">
          <div class="space-y-1.5">
            <Label class="text-sm font-medium text-muted-foreground">Preview label</Label>
            <Input v-model="hostnameForm.fallbackLabel" placeholder="preview" />
          </div>
          <div class="space-y-1.5">
            <Label class="text-sm font-medium text-muted-foreground">Preview URL</Label>
            <Input :model-value="buildFallbackHostname()" readonly placeholder="Current server needs an IPv4 address" />
          </div>
          <Alert class="border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-200">
            <AlertDescription>
              Preview URLs are fine for onboarding and evaluation, but they are not recommended for production.
            </AlertDescription>
          </Alert>
        </template>

        <template v-else>
          <div class="space-y-1.5">
            <Label class="text-sm font-medium text-muted-foreground">Exact hostname</Label>
            <Input v-model="hostnameForm.hostname" placeholder="www.client-example.com" />
          </div>
        </template>
      </div>
      <Button
        type="button"
        class="mt-4 bg-accent text-accent-foreground hover:bg-accent/85"
        :disabled="isAssignDisabled"
        @click="handleAssign"
      >
        Add hostname
      </Button>
    </div>
  </div>
</template>
