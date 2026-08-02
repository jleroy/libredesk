<template>
  <Spinner v-if="loading" />
  <div v-else class="flex flex-col">
    <CustomBreadcrumb :links="breadcrumbLinks" class="mb-5" />

    <div class="flex gap-6">
      <div class="w-[420px] shrink-0 pr-2">
        <HelpCenterForm
          :help-center="helpCenter"
          :submit-form="handleSave"
          :is-loading="isSubmitting"
          @cancel="goBack"
          @change="onFormChange"
        />
      </div>

      <!-- The preview page pins its footer to the bottom of the frame; an unbounded frame strands it below the content. -->
      <div class="flex-1 min-w-0">
        <div
          class="sticky top-0 h-[calc(100dvh-8rem)] rounded-md border overflow-hidden bg-muted"
        >
          <iframe
            ref="previewFrame"
            class="w-full h-full border-0 bg-background"
            :title="t('globals.terms.helpCenter')"
            sandbox="allow-same-origin"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import HelpCenterForm from '@main/features/admin/help-center/HelpCenterForm.vue'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const props = defineProps({
  id: {
    type: [String, Number],
    required: true
  }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()

const loading = ref(true)
const isSubmitting = ref(false)
const helpCenter = ref(null)
const previewFrame = ref(null)

let previewTimer = null

const breadcrumbLinks = computed(() => [
  { path: 'help-center-list', label: t('globals.terms.helpCenter', 2) },
  { path: '', label: helpCenter.value?.name || '' }
])

const goBack = () => router.push({ name: 'help-center-list' })

const renderPreview = async (values) => {
  try {
    const { data } = await api.previewHelpCenter(props.id, values)
    if (previewFrame.value) previewFrame.value.srcdoc = data
  } catch {
    // A half-filled form can fail validation while typing; the last good preview stays up.
  }
}

const onFormChange = (values) => {
  clearTimeout(previewTimer)
  previewTimer = setTimeout(() => renderPreview(values), 300)
}

const handleSave = async (formData) => {
  isSubmitting.value = true
  try {
    await api.updateHelpCenter(props.id, formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  try {
    const { data } = await api.getHelpCenter(props.id)
    helpCenter.value = data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => clearTimeout(previewTimer))
</script>
