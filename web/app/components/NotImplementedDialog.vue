<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useNotImplemented } from "~/composables/useNotImplemented";

const { isOpen, close } = useNotImplemented();

const repoUrl = "https://github.com/bombig-net/pressluft";
</script>

<template>
  <Dialog :open="isOpen" @update:open="(v: boolean) => { if (!v) close() }">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Not implemented yet</DialogTitle>
        <DialogDescription>
          This feature is on the roadmap but hasn't been built yet. Pressluft is an open-source project and contributions are welcome.
        </DialogDescription>
      </DialogHeader>

      <div class="rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
        Want to help build this? Check out the repository, open an issue, or send a pull request.
      </div>

      <DialogFooter class="gap-2 sm:gap-0">
        <Button variant="outline" @click="close">Close</Button>
        <Button as="a" :href="repoUrl" target="_blank" rel="noopener noreferrer">
          View on GitHub
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
