<script setup lang="ts">
/**
 * Top-level shell. The router handles layout selection: routes
 * marked meta.layout='bare' (login) render their own full-page
 * layouts; everything else mounts inside AppShell, which is itself
 * a router child component for the authenticated routes.
 *
 * UI-prefs hydration runs here (not in main.ts) so Pinia is fully
 * active by the time we touch the store — calling useStore() in
 * main.ts before app.mount() can race with Pinia's internal active
 * instance setup and crash the bundle, blanking the page.
 */
import { useUIStore } from '@/stores/ui'

useUIStore().hydrate()
</script>

<template>
  <RouterView />
</template>
