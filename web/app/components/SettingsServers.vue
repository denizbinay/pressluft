<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useProviders } from "~/composables/useProviders";
import { useServers } from "~/composables/useServers";
import { parseHealthResponse } from "~/lib/api-runtime";
import type { CallbackURLMode } from "~/lib/platform-contract.generated";
import ServerCreationWizard from "~/components/settings/ServerCreationWizard.vue";
import ServerList from "~/components/settings/ServerList.vue";

const { fetchProviders } = useProviders();
const { fetchServers } = useServers();

const { apiFetch } = useApiClient();

const callbackMode = ref<CallbackURLMode>("unknown");
const callbackWarning = ref("");
const wizardOpen = ref(false);

onMounted(async () => {
  await Promise.all([fetchProviders(), fetchServers()]);
  apiFetch("/health")
    .then((payload) => parseHealthResponse(payload))
    .then((body) => {
      callbackMode.value = body.callback_url_mode || "unknown";
      callbackWarning.value = body.callback_url_warning || "";
    })
    .catch(() => {});
});

const openWizard = () => {
  wizardOpen.value = true;
};

const handleWizardUpdate = (value: boolean) => {
  wizardOpen.value = value;
};

const handleCreated = () => {
  fetchServers();
};
</script>

<template>
  <div class="space-y-6">
    <Alert
      v-if="callbackMode === 'ephemeral' && callbackWarning"
      class="border-accent/30 bg-accent/10 text-accent"
    >
      <AlertDescription>
        {{ callbackWarning }}
      </AlertDescription>
    </Alert>

    <ServerList @create="openWizard" />

    <ServerCreationWizard
      :is-open="wizardOpen"
      @update:open="handleWizardUpdate"
      @created="handleCreated"
    />
  </div>
</template>
