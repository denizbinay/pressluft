<script setup lang="ts">
import { Button } from "@/components/ui/button"
import { HelpCircle } from "lucide-vue-next"
</script>

<template>
  <Button
    variant="ghost"
    class="rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    type="button"
    aria-label="Help"
  >
    <HelpCircle class="h-4 w-4" aria-hidden="true" />
  </Button>
</template>
