<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useDomains } from "~/composables/useDomains";
import { errorMessage } from "~/lib/utils";

const {
  domains,
  loading,
  saving,
  error,
  fetchDomains,
  createDomain,
  updateDomain,
  deleteDomain,
} = useDomains();

const pageError = ref("");
const successMessage = ref("");

const form = reactive({
  hostname: "",
  kind: "base_domain",
  dnsState: "pending",
});

const allDomains = computed(() => domains.value);

const domainStats = computed(() => ({
  total: allDomains.value.length,
  baseDomains: allDomains.value.filter((domain) => domain.kind === "base_domain").length,
  attached: allDomains.value.filter((domain) => Boolean(domain.site_id)).length,
  dnsPending: allDomains.value.filter((domain) => domain.dns_state === "pending").length,
}));

const domainKindLabel = (domain: (typeof allDomains.value)[number]) =>
  domain.kind === "base_domain" ? "Reusable base domain" : "Hostname";

const domainSourceLabel = (domain: (typeof allDomains.value)[number]) =>
  domain.source === "fallback_resolver" ? "Preview URL" : "User-managed";

const dnsStateLabel = (state: string) => {
  switch (state) {
    case "ready":
      return "DNS ready";
    case "issue":
      return "Needs attention";
    case "disabled":
      return "Disabled";
    default:
      return "Pending DNS";
  }
};

const routingStateLabel = (state: string) => {
  switch (state) {
    case "ready":
      return "Routing ready";
    case "issue":
      return "Routing issue";
    case "pending":
      return "Routing pending";
    default:
      return "Not routed";
  }
};

const loadPage = async () => {
  pageError.value = "";
  try {
    await fetchDomains();
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to load domains";
  }
};

const resetForm = () => {
  form.hostname = "";
  form.kind = "base_domain";
  form.dnsState = "pending";
};

const handleCreateDomain = async () => {
  pageError.value = "";
  successMessage.value = "";
  try {
    const created = await createDomain({
      hostname: form.hostname,
      kind: form.kind,
      source: "user",
      dns_state: form.dnsState,
    });
    successMessage.value = `Added ${created.hostname}.`;
    resetForm();
    await fetchDomains();
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to add domain";
  }
};

const handleDNSStateChange = async (domainId: string, dnsState: string) => {
  pageError.value = "";
  successMessage.value = "";
  try {
    await updateDomain(domainId, { dns_state: dnsState });
    successMessage.value = "DNS state updated.";
    await fetchDomains();
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to update DNS state";
  }
};

const handleDelete = async (domainId: string, hostname: string) => {
  if (!window.confirm(`Delete ${hostname}?`)) {
    return;
  }
  pageError.value = "";
  successMessage.value = "";
  try {
    await deleteDomain(domainId);
    successMessage.value = `Deleted ${hostname}.`;
    await fetchDomains();
  } catch (e: unknown) {
    pageError.value = errorMessage(e) || "Failed to delete domain";
  }
};

onMounted(loadPage);
</script>

<template>
  <div class="space-y-8">
    <section
      class="relative overflow-hidden rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(255,255,255,0.95),rgba(226,244,240,0.88)_45%,rgba(237,233,254,0.82))] p-7 shadow-[0_32px_120px_-56px_rgba(18,95,84,0.45)] dark:bg-[linear-gradient(135deg,rgba(21,28,31,0.95),rgba(17,57,53,0.88)_50%,rgba(36,33,52,0.88))]"
    >
      <div class="absolute inset-y-0 right-0 hidden w-80 bg-[radial-gradient(circle_at_top,rgba(69,198,214,0.28),transparent_62%)] lg:block" />
      <div class="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-4">
          <div class="inline-flex items-center gap-2 rounded-full border border-foreground/10 bg-background/70 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground backdrop-blur">
            Domains
          </div>
          <div>
            <h1 class="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
              Track the domains you actually control.
            </h1>
            <p class="mt-3 max-w-2xl text-base leading-7 text-muted-foreground">
              Keep reusable base domains and concrete hostnames in one truthful inventory, with separate DNS and routing states for later deployment work.
            </p>
          </div>
        </div>

        <div class="grid grid-cols-4 gap-3 sm:min-w-[520px]">
          <div class="rounded-2xl border border-border/60 bg-background/75 px-4 py-4 backdrop-blur">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">All</p>
            <p class="mt-3 text-3xl font-semibold text-foreground">{{ domainStats.total }}</p>
          </div>
          <div class="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-primary/80">Base domains</p>
            <p class="mt-3 text-3xl font-semibold text-primary">{{ domainStats.baseDomains }}</p>
          </div>
          <div class="rounded-2xl border border-accent/20 bg-accent/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-accent/80">Attached</p>
            <p class="mt-3 text-3xl font-semibold text-accent">{{ domainStats.attached }}</p>
          </div>
          <div class="rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-amber-700 dark:text-amber-300">Pending DNS</p>
            <p class="mt-3 text-3xl font-semibold text-amber-700 dark:text-amber-300">{{ domainStats.dnsPending }}</p>
          </div>
        </div>
      </div>
    </section>

    <Alert v-if="pageError || error" class="border-destructive/30 bg-destructive/10 text-destructive">
      <AlertDescription>{{ pageError || error }}</AlertDescription>
    </Alert>
    <Alert v-if="successMessage" class="border-primary/30 bg-primary/10 text-primary">
      <AlertDescription>{{ successMessage }}</AlertDescription>
    </Alert>

    <div class="grid gap-6 xl:grid-cols-[0.95fr_1.25fr]">
      <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
        <CardHeader class="border-b border-border/50 px-6 py-5">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Add domain</p>
          <h2 class="mt-1 text-xl font-semibold text-foreground">Bring in a user-managed domain</h2>
        </CardHeader>
        <CardContent class="space-y-4 px-6 py-5">
          <form class="space-y-4" @submit.prevent="handleCreateDomain">
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Hostname or base domain</Label>
              <Input v-model="form.hostname" :placeholder="form.kind === 'base_domain' ? 'agency.example.com' : 'www.client-example.com'" />
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-muted-foreground">Inventory type</Label>
                <select
                  v-model="form.kind"
                  class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                >
                  <option value="base_domain">Reusable base domain</option>
                  <option value="hostname">Exact hostname</option>
                </select>
              </div>
              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-muted-foreground">Initial DNS state</Label>
                <select
                  v-model="form.dnsState"
                  class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                >
                  <option value="pending">Pending DNS</option>
                  <option value="ready">DNS ready</option>
                  <option value="issue">Needs attention</option>
                  <option value="disabled">Disabled</option>
                </select>
              </div>
            </div>

            <div class="rounded-2xl border border-border/60 bg-muted/20 p-4 text-sm leading-6 text-muted-foreground">
              Add a reusable base domain when you want to mint child hostnames later. Add an exact hostname when DNS will point to a single concrete address.
            </div>

            <Button type="submit" class="w-full bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving || !form.hostname.trim()">
              {{ saving ? "Adding domain..." : "Add domain" }}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
        <CardHeader class="border-b border-border/50 px-6 py-5">
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Inventory</p>
          <h2 class="mt-1 text-xl font-semibold text-foreground">All tracked domains and hostnames</h2>
        </CardHeader>
        <CardContent class="px-6 py-5">
          <div v-if="loading" class="py-10 text-sm text-muted-foreground">Loading domains...</div>
          <div v-else-if="allDomains.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
            No domains tracked yet.
          </div>
          <div v-else class="space-y-3">
            <div v-for="domain in allDomains" :key="domain.id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
              <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="text-sm font-semibold text-foreground">{{ domain.hostname }}</p>
                    <Badge variant="outline" class="border-primary/30 bg-primary/10 text-primary">{{ domainKindLabel(domain) }}</Badge>
                    <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domainSourceLabel(domain) }}</Badge>
                    <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ dnsStateLabel(domain.dns_state) }}</Badge>
                    <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ routingStateLabel(domain.routing_state) }}</Badge>
                  </div>
                  <p class="mt-1 text-xs text-muted-foreground">
                    {{ domain.kind === "base_domain" ? "Reusable parent for future child hostnames." : domain.site_name ? `Attached to ${domain.site_name}.` : "Unassigned exact hostname." }}
                    <span v-if="domain.parent_hostname"> Child of {{ domain.parent_hostname }}.</span>
                  </p>
                </div>

                <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
                  <select
                    :value="domain.dns_state"
                    class="flex h-9 rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
                    @change="handleDNSStateChange(domain.id, ($event.target as HTMLSelectElement).value)"
                  >
                    <option value="pending">Pending DNS</option>
                    <option value="ready">DNS ready</option>
                    <option value="issue">Needs attention</option>
                    <option value="disabled">Disabled</option>
                  </select>
                  <NuxtLink v-if="domain.site_id" :to="`/sites/${domain.site_id}`" class="text-sm font-medium text-accent transition hover:text-accent/80">
                    Open site
                  </NuxtLink>
                  <Button type="button" variant="ghost" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleDelete(domain.id, domain.hostname)">
                    Delete
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
