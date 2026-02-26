<script setup lang="ts">
import { Card, CardContent } from "@/components/ui/card"
</script>

<template>
  <div class="space-y-8">
    <div>
      <h1 class="text-3xl font-semibold text-foreground">Servers</h1>
      <p class="mt-2 text-base text-muted-foreground">
        Provision and manage agency-ready infrastructure profiles.
      </p>
    </div>

    <Card class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none">
      <CardContent class="px-6 py-5">
        <SettingsServers />
      </CardContent>
    </Card>
  </div>
</template>
