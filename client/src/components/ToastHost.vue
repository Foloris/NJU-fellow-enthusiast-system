<script setup lang="ts">
import { CheckCircle2, CircleAlert, Info } from "lucide-vue-next";
import { toasts } from "@/stores/toast";
</script>

<template>
  <div class="toast-stack" aria-live="polite">
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast"
        :class="`toast-${toast.type}`"
      >
        <CheckCircle2 v-if="toast.type === 'success'" :size="18" />
        <CircleAlert v-else-if="toast.type === 'error'" :size="18" />
        <Info v-else :size="18" />
        <span>{{ toast.message }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>

