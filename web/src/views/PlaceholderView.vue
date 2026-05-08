<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <section class="mt-8 rounded border border-border bg-surface p-6">
    <p class="font-mono text-sm text-text-muted">route: {{ route.fullPath }}</p>
    <p class="mt-2">View not yet implemented in Phase 0.</p>
  </section>
</template>
