<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const props = defineProps<{
  name: string;
  wordpressAdminEmail: string;
  saving: boolean;
}>();

const emit = defineEmits<{
  (e: "save", payload: { name: string; wordpressAdminEmail: string }): void;
  (e: "delete"): void;
}>();

const form = reactive({
  name: props.name,
  wordpressAdminEmail: props.wordpressAdminEmail,
});

// Sync props into local form when parent re-hydrates
watch(
  () => [props.name, props.wordpressAdminEmail],
  ([name, email]) => {
    form.name = name;
    form.wordpressAdminEmail = email;
  },
);

const handleSubmit = () => {
  emit("save", {
    name: form.name,
    wordpressAdminEmail: form.wordpressAdminEmail,
  });
};
</script>

<template>
  <div class="space-y-6">
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div class="space-y-1.5">
        <Label class="text-sm font-medium text-muted-foreground">Site name</Label>
        <Input v-model="form.name" />
      </div>

      <div class="space-y-1.5">
        <Label class="text-sm font-medium text-muted-foreground">WordPress admin email</Label>
        <Input v-model="form.wordpressAdminEmail" type="email" />
      </div>

      <div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-sm text-muted-foreground">
        Pressluft keeps the install path, PHP version, WordPress version, and certificate contact flow under managed defaults during this first live deployment phase.
      </div>

      <div class="flex flex-col gap-3 border-t border-border/50 pt-4 sm:flex-row sm:justify-between">
        <Button type="button" variant="ghost" class="justify-start text-destructive hover:bg-destructive/10 hover:text-destructive" @click="emit('delete')">Delete site record</Button>
        <Button type="submit" class="bg-accent text-accent-foreground hover:bg-accent/85" :disabled="saving">{{ saving ? "Saving..." : "Save changes" }}</Button>
      </div>
    </form>
  </div>
</template>
