<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { StoredServer } from "~/composables/useServers";
import { useSites, type StoredSite } from "~/composables/useSites";
import { useAuth } from "~/composables/useAuth";
import { errorMessage } from "~/lib/utils";

const props = defineProps<{
  open: boolean;
  servers: StoredServer[];
  initialServerId?: string;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  created: [site: StoredSite];
}>();

const { createSite, saving } = useSites();
const { user } = useAuth();

const form = reactive({
  serverId: "",
  name: "",
  wordpressAdminEmail: "",
  mode: "preview",
  hostname: "",
});

const formError = ref("");

const deployableServers = computed(() =>
  props.servers.filter(
    (server) =>
      server.status === "ready" &&
      server.setup_state === "ready" &&
      server.profile_key === "nginx-stack",
  ),
);

const selectedServer = computed(
  () => deployableServers.value.find((server) => server.id === form.serverId) || null,
);

const previewLabel = computed(() => {
  const label = form.name
    .trim()
    .toLowerCase()
    .replace(/_/g, "-")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40);
  return label || "site";
});

const previewHostname = computed(() => {
  const ipv4 = selectedServer.value?.ipv4?.trim();
  if (!ipv4) {
    return "";
  }
  return `${previewLabel.value}.${ipv4.replace(/\./g, "-")}.sslip.io`;
});

const canCreate = computed(() => {
  if (!form.serverId || !form.name.trim() || !form.wordpressAdminEmail.trim()) {
    return false;
  }
  if (form.mode === "preview") {
    return Boolean(previewHostname.value);
  }
  return Boolean(form.hostname.trim());
});

const resetForm = () => {
  formError.value = "";
  form.name = "";
  form.wordpressAdminEmail = user.value?.email?.trim() || "";
  form.mode = "preview";
  form.hostname = "";
  const preferred = props.initialServerId?.trim();
  form.serverId =
    deployableServers.value.find((server) => server.id === preferred)?.id ||
    deployableServers.value[0]?.id ||
    "";
};

const handleOpenChange = (value: boolean) => {
  emit("update:open", value);
};

const close = () => {
  emit("update:open", false);
};

const handleSubmit = async () => {
  formError.value = "";
  try {
      const created = await createSite({
        server_id: form.serverId,
        name: form.name.trim(),
        wordpress_admin_email: form.wordpressAdminEmail.trim(),
        primary_hostname_config:
        form.mode === "preview"
          ? {
              source: "fallback_resolver",
              label: previewLabel.value,
            }
          : {
              source: "user",
              hostname: form.hostname.trim(),
            },
    });
    emit("created", created);
    close();
  } catch (e: unknown) {
    formError.value = errorMessage(e) || "Failed to create site";
  }
};

watch(
  () => props.open,
  (value) => {
    if (value) {
      resetForm();
    }
  },
  { immediate: true },
);

watch(
  () => props.initialServerId,
  () => {
    if (props.open) {
      resetForm();
    }
  },
);
</script>

<template>
  <Dialog :open="open" @update:open="handleOpenChange">
    <DialogContent class="border-border/60 bg-popover/95 sm:max-w-lg">
      <DialogHeader class="text-left">
        <DialogTitle class="text-xl font-semibold text-foreground">Create site</DialogTitle>
        <DialogDescription class="text-sm leading-6 text-muted-foreground">
          Start with the minimum: a site name, a deployable server, and either a preview URL or the real hostname.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <Alert v-if="formError" class="border-destructive/30 bg-destructive/10 text-destructive">
          <AlertDescription>{{ formError }}</AlertDescription>
        </Alert>

        <Alert
          v-if="deployableServers.length === 0"
          class="border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-200"
        >
          <AlertDescription>
            You need a ready `nginx-stack` server before Pressluft can create and deploy a WordPress site.
          </AlertDescription>
        </Alert>

        <template v-else>
          <div class="space-y-1.5">
            <Label for="site-name" class="text-sm font-medium text-muted-foreground">Site name</Label>
            <Input id="site-name" v-model="form.name" placeholder="e.g. Northwind Marketing" />
          </div>

          <div class="space-y-1.5">
            <Label for="site-admin-email" class="text-sm font-medium text-muted-foreground">WordPress admin email</Label>
            <Input id="site-admin-email" v-model="form.wordpressAdminEmail" type="email" placeholder="owner@client-example.com" />
            <p class="text-xs text-muted-foreground">
              Prefilled from your Pressluft account, but editable for the website owner or inbox you want WordPress to use.
            </p>
          </div>

          <div class="space-y-1.5">
            <Label for="site-server" class="text-sm font-medium text-muted-foreground">Server</Label>
            <select
              id="site-server"
              v-model="form.serverId"
              class="flex h-10 w-full rounded-lg border border-border/60 bg-background/70 px-3 text-sm text-foreground outline-none transition focus:border-accent/40"
            >
              <option v-for="server in deployableServers" :key="server.id" :value="server.id">
                {{ server.name }} - {{ server.location }} - {{ server.server_type }}
              </option>
            </select>
          </div>

          <div class="space-y-3 rounded-2xl border border-border/60 bg-muted/25 p-4">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Destination</p>
              <p class="mt-1 text-sm text-muted-foreground">Choose a temporary preview URL or the real hostname you want to launch.</p>
            </div>

            <label class="flex cursor-pointer items-start gap-3 rounded-xl border border-border/60 bg-background/80 p-3">
              <input v-model="form.mode" type="radio" class="mt-1" value="preview" />
              <div>
                <p class="text-sm font-medium text-foreground">Preview URL</p>
                <p class="mt-1 text-sm text-muted-foreground">
                  Pressluft generates a temporary `sslip.io` URL from the selected server. Good for onboarding and evaluation.
                </p>
              </div>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-xl border border-border/60 bg-background/80 p-3">
              <input v-model="form.mode" type="radio" class="mt-1" value="domain" />
              <div>
                <p class="text-sm font-medium text-foreground">Real domain</p>
                <p class="mt-1 text-sm text-muted-foreground">
                  Use the real hostname you want this WordPress site to answer on.
                </p>
              </div>
            </label>

            <div v-if="form.mode === 'preview'" class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">Preview URL</Label>
              <Input :model-value="previewHostname" readonly placeholder="Choose a ready server first" />
              <p class="text-xs text-muted-foreground">
                The resolver details stay internal to Pressluft. This URL is not intended for production.
              </p>
            </div>

            <div v-else class="space-y-1.5">
              <Label for="site-hostname" class="text-sm font-medium text-muted-foreground">Hostname</Label>
              <Input id="site-hostname" v-model="form.hostname" placeholder="www.client-example.com" />
            </div>
          </div>

          <div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
            Pressluft will create the WordPress install, provision its database, configure nginx and TLS, and apply the managed baseline automatically.
          </div>
        </template>
      </div>

      <DialogFooter class="gap-2 sm:justify-end">
        <Button variant="ghost" @click="close">Cancel</Button>
        <Button class="bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving || !canCreate" @click="handleSubmit">
          {{ saving ? "Creating site..." : "Create and deploy site" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
