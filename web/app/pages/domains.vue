<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useDomains } from "~/composables/useDomains";

const {
  domains,
  loading,
  saving,
  error,
  fetchDomains,
  createDomain,
  deleteDomain,
} = useDomains();

const pageError = ref("");
const successMessage = ref("");

const form = reactive({
  hostname: "",
  isWildcard: false,
});

const allDomains = computed(() => domains.value);

const domainStats = computed(() => ({
  total: allDomains.value.length,
  attached: allDomains.value.filter((domain) => Boolean(domain.site_id)).length,
  unassigned: allDomains.value.filter((domain) => !domain.site_id).length,
}));

const domainKindLabel = (domain: (typeof allDomains.value)[number]) => {
  if (domain.kind === "wildcard") {
    return "Wildcard domain";
  }
  if (domain.parent_domain_id) {
    return "Wildcard child";
  }
  return "Direct domain";
};

const domainRoleLabel = (domain: (typeof allDomains.value)[number]) => {
  if (domain.kind === "wildcard") {
    if (domain.ownership !== "platform") {
      return "Wildcard";
    }
    return domain.status === "active" ? "Active" : "Coming soon";
  }
  return domain.is_primary ? "Primary" : "Additional";
};

const domainOwnershipLabel = (domain: (typeof allDomains.value)[number]) =>
  domain.ownership === "platform" ? "Platform" : "User";

const loadPage = async () => {
  pageError.value = "";
  try {
    await fetchDomains();
  } catch (e: any) {
    pageError.value = e.message || "Failed to load domains";
  }
};

const resetForm = () => {
  form.hostname = "";
  form.isWildcard = false;
};

const handleCreateDomain = async () => {
  pageError.value = "";
  successMessage.value = "";
  try {
    const created = await createDomain({
      hostname: form.hostname,
      kind: form.isWildcard ? "wildcard" : "direct",
    });
    successMessage.value = `Added ${created.hostname}.`;
    resetForm();
    await fetchDomains();
  } catch (e: any) {
    pageError.value = e.message || "Failed to add domain";
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
  } catch (e: any) {
    pageError.value = e.message || "Failed to delete domain";
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
              Keep every domain and temporary URL in one place.
            </h1>
            <p class="mt-3 max-w-2xl text-base leading-7 text-muted-foreground">
              Store standalone hostnames and reusable wildcard roots in one inventory. Concrete child hostnames get minted from site actions when Pressluft needs them.
            </p>
          </div>
        </div>

        <div class="grid grid-cols-3 gap-3 sm:min-w-[420px]">
          <div class="rounded-2xl border border-border/60 bg-background/75 px-4 py-4 backdrop-blur">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">All domains</p>
            <p class="mt-3 text-3xl font-semibold text-foreground">{{ domainStats.total }}</p>
          </div>
          <div class="rounded-2xl border border-primary/20 bg-primary/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-primary/80">Attached</p>
            <p class="mt-3 text-3xl font-semibold text-primary">{{ domainStats.attached }}</p>
          </div>
          <div class="rounded-2xl border border-accent/20 bg-accent/10 px-4 py-4">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-accent/80">Unassigned</p>
            <p class="mt-3 text-3xl font-semibold text-accent">{{ domainStats.unassigned }}</p>
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
          <h2 class="mt-1 text-xl font-semibold text-foreground">Bring in a new domain</h2>
        </CardHeader>
        <CardContent class="px-6 py-5">
          <form class="space-y-4" @submit.prevent="handleCreateDomain">
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Domain</Label>
              <Input v-model="form.hostname" placeholder="www.client-example.com" />
            </div>
            <div class="rounded-2xl border border-border/60 bg-muted/20 p-4 text-sm leading-6 text-muted-foreground">
              Add a direct hostname if you only need one exact address. Add a wildcard root if you want Pressluft to generate child hostnames later for previews, rollbacks, staging, and similar workflows.
            </div>
            <label class="flex items-start gap-3 rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
              <input v-model="form.isWildcard" type="checkbox" class="mt-1 h-4 w-4 rounded border-border text-accent focus:ring-accent" />
              <span>
                Store this as a wildcard domain root. Enter `agency.dev`, not `*.agency.dev`. Pressluft will treat it as the reusable parent for future child hostnames, without enabling any DNS or TLS workflow yet.
              </span>
            </label>
            <Button type="submit" class="w-full bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving || !form.hostname.trim()">
              {{ saving ? "Adding domain..." : "Add domain" }}
            </Button>
          </form>
        </CardContent>
      </Card>

      <div class="space-y-6">
        <Card class="rounded-[24px] border border-border/60 bg-card/70 py-0 shadow-none">
          <CardHeader class="border-b border-border/50 px-6 py-5">
            <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Inventory</p>
            <h2 class="mt-1 text-xl font-semibold text-foreground">All tracked domains</h2>
          </CardHeader>
          <CardContent class="px-6 py-5">
            <div v-if="loading" class="py-10 text-sm text-muted-foreground">Loading domains...</div>
             <div v-else-if="allDomains.length === 0" class="rounded-2xl border border-dashed border-border/60 bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
                No domains tracked yet.
             </div>
             <div v-else class="space-y-3">
              <div v-for="domain in allDomains" :key="domain.id" class="rounded-2xl border border-border/60 bg-background/70 p-4">
                 <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                   <div>
                     <div class="flex flex-wrap items-center gap-2">
                       <p class="text-sm font-semibold text-foreground">{{ domain.hostname }}</p>
                       <Badge variant="outline" class="border-primary/30 bg-primary/10 text-primary">{{ domainRoleLabel(domain) }}</Badge>
                       <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domainKindLabel(domain) }}</Badge>
                        <Badge variant="outline" class="border-border/60 bg-muted/40 text-muted-foreground">{{ domainOwnershipLabel(domain) }}</Badge>
                      </div>
                      <p class="mt-1 text-xs text-muted-foreground">
                         {{ domain.kind === "wildcard" ? "Reusable root for future generated child hostnames" : domain.site_name ? `Attached to ${domain.site_name}` : "Unassigned" }}
                         <span v-if="domain.parent_hostname"> · via {{ domain.parent_hostname }}</span>
                       </p>
                   </div>
                    <div class="flex items-center gap-3 text-xs text-muted-foreground">
                       <NuxtLink v-if="domain.site_id" :to="`/sites/${domain.site_id}`" class="font-medium text-accent transition hover:text-accent/80">
                         Open site
                       </NuxtLink>
                    <Button v-if="domain.ownership !== 'platform'" type="button" variant="ghost" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" @click="handleDelete(domain.id, domain.hostname)">
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
  </div>
</template>
