<template>
  <BaseBanner
    v-if="!isOnline"
    :text="$t('globals.messages.noInternetConnection')"
    color-class="bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300"
  />
  <BaseBanner
    v-else-if="connectionFailed"
    :text="$t('globals.messages.connectionFailedRefresh')"
    color-class="bg-destructive text-destructive-foreground"
  />
  <BaseBanner
    v-else-if="showReconnecting"
    :text="$t('globals.messages.connecting')"
    color-class="bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300"
  />
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { useOnline } from '@vueuse/core'
import { storeToRefs } from 'pinia'
import { useConnectionStore } from '@/stores/connection'
import BaseBanner from '@shared-ui/components/BaseBanner.vue'

const RECONNECTING_DELAY_MS = 1500

const isOnline = useOnline()
const { connecting, connectionFailed } = storeToRefs(useConnectionStore())
const showReconnecting = ref(false)
let reconnectingTimer = null

// Most drops recover on the first retry, so hold the banner back instead of flashing it.
watch(connecting, (value) => {
  clearTimeout(reconnectingTimer)
  if (value) {
    reconnectingTimer = setTimeout(() => (showReconnecting.value = true), RECONNECTING_DELAY_MS)
  } else {
    showReconnecting.value = false
  }
})

onUnmounted(() => clearTimeout(reconnectingTimer))
</script>
