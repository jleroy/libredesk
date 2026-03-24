<template>
  <div>
    <div
      v-if="pageVisits.length === 0"
      class="text-center text-sm text-muted-foreground py-4"
    >
      {{ t('globals.messages.noResults') }}
    </div>
    <div v-else class="space-y-1">
      <a
        v-for="(page, idx) in pageVisits"
        :key="idx"
        :href="page.url"
        target="_blank"
        rel="noopener"
        class="block p-2 rounded hover:bg-muted"
      >
        <div class="flex items-start justify-between gap-2">
          <span class="sidebar-value font-medium truncate" :title="page.url">
            {{ page.title || page.url }}
          </span>
          <span v-if="page.time" class="sidebar-label flex-shrink-0">
            {{ formatDate(page.time) }}
          </span>
        </div>
      </a>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import api from '../../../api'

const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()

const pageVisits = computed(() => conversation.value?.contact?.page_visits || [])

function formatDate (dateStr) {
  try {
    return format(new Date(dateStr), 'd MMM yyyy')
  } catch {
    return ''
  }
}

watch(
  () => conversation.value?.uuid,
  async (uuid) => {
    if (uuid && conversation.value?.inbox_channel === 'livechat') {
      try {
        const resp = await api.getContactPageVisits(uuid)
        if (resp.data?.data) {
          conversationStore.mergeContactUpdate({
            contact_id: conversation.value?.contact_id,
            page_visits: resp.data.data
          })
        }
      } catch {
        // Page visits are optional.
      }
    }
  },
  { immediate: true }
)
</script>
