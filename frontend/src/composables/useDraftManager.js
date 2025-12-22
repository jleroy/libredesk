import { ref, watch } from 'vue'
import { watchDebounced, useStorage, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@/stores/conversation'
import { MACRO_CONTEXT } from '@/constants/conversation'
import api from '@/api'

/**
 * Validate macro actions have required structure
 */
const validateMacroActions = (actions) => {
  if (!Array.isArray(actions)) return []
  return actions.filter(action =>
    action &&
    'type' in action &&
    'value' in action &&
    Array.isArray(action.value) &&
    'display_value' in action &&
    Array.isArray(action.display_value)
  )
}

/**
 * Validate attachments have required structure
 */
const validateAttachments = (attachments) => {
  if (!Array.isArray(attachments)) return []
  return attachments.filter(attachment =>
    attachment &&
    'id' in attachment &&
    'size' in attachment &&
    'uuid' in attachment &&
    'filename' in attachment
  )
}

/**
 * Composable for managing draft state and persistence
 * Saves to localStorage immediately, syncs to backend on conversation switch/send/unload
 * @param key - Reactive reference to current draft key
 * @param uploadedFiles - Optional reactive reference to uploaded files array
 */
export function useDraftManager (key, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const isDirty = ref(false)
  const skipNextSave = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])

  // Reactive localStorage for all drafts
  const localDrafts = useStorage('libredesk_drafts', {})

  /**
   * Save draft to localStorage only
   */
  const saveDraftLocal = (draftKey) => {
    if (!draftKey) return
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    const draftMeta = {}
    if (macroActions.length > 0) draftMeta.macro_actions = macroActions
    if (uploadedFiles?.value?.length > 0) draftMeta.attachments = uploadedFiles.value

    localDrafts.value[draftKey] = { content: htmlContent.value, meta: draftMeta }
    isDirty.value = true
  }

  /**
   * Get draft from localStorage
   */
  const getLocalDraft = (draftKey) => localDrafts.value[draftKey] || null

  /**
   * Remove draft from localStorage
   */
  const removeLocalDraft = (draftKey) => {
    if (localDrafts.value[draftKey]) {
      delete localDrafts.value[draftKey]
    }
  }

  /**
   * Sync localStorage draft to backend
   */
  const syncDraftToBackend = async (draftKey) => {
    if (!draftKey || !isDirty.value) return
    const localDraft = getLocalDraft(draftKey)
    if (!localDraft) return

    try {
      await api.saveDraft(draftKey, localDraft)
      isDirty.value = false
    } catch (error) {
      // Silent fail - will retry on next sync
    }
  }

  /**
   * Reset all draft state to initial values
   */
  const resetState = () => {
    htmlContent.value = ''
    textContent.value = ''
    isLoading.value = false
    isDirty.value = false
    loadedAttachments.value = []
    loadedMacroActions.value = []
  }

  /**
   * Load draft from backend
   */
  const loadDraft = async (draftKey) => {
    if (!draftKey) return
    isLoading.value = true
    isDirty.value = false
    skipNextSave.value = true
    try {
      // Check if there's an unsynced localStorage draft (e.g., from page refresh)
      const localDraft = getLocalDraft(draftKey)
      if (localDraft) {
        await api.saveDraft(draftKey, localDraft)
        removeLocalDraft(draftKey)
      }

      // Load from backend (source of truth)
      const response = await api.getDraft(draftKey)
      const draft = response.data.data
      htmlContent.value = draft.content || ''
      textContent.value = ''
      loadedAttachments.value = validateAttachments(draft.meta?.attachments)
      loadedMacroActions.value = validateMacroActions(draft.meta?.macro_actions)
    } catch (error) {
      resetState()
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Clear draft from both localStorage and backend
   */
  const clearDraft = async (draftKey) => {
    if (!draftKey) return
    removeLocalDraft(draftKey)
    isLoading.value = true
    try {
      await api.deleteDraft(draftKey)
      resetState()
    } catch (error) {
      // Silent fail
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Check if draft has content
   */
  const hasDraftContent = () => {
    return textContent.value?.trim() !== ''
  }

  // Watch for key changes - sync to backend before switching
  watch(
    key,
    async (newKey, oldKey) => {
      // Sync old draft to backend before switching
      if (newKey !== oldKey && isDirty.value && hasDraftContent()) {
        await syncDraftToBackend(oldKey)
        removeLocalDraft(oldKey)
      }

      // Load new draft from backend
      if (newKey && newKey !== oldKey) {
        await loadDraft(newKey)
      } else if (!newKey && oldKey) {
        resetState()
      }
    },
    { immediate: true }
  )

  // Debounced watcher - save to localStorage only
  const watchSources = [
    htmlContent,
    textContent,
    () => conversationStore.macros[MACRO_CONTEXT.REPLY]
  ]
  if (uploadedFiles) {
    watchSources.push(uploadedFiles)
  }

  watchDebounced(
    watchSources,
    () => {
      if (skipNextSave.value) {
        skipNextSave.value = false
        return
      }
      if (!isLoading.value && key.value) {
        saveDraftLocal(key.value)
      }
    },
    { debounce: 250, deep: true }
  )

  // Sync to backend when page is hidden (tab switch)
  useEventListener(document, 'visibilitychange', async () => {
    if (document.visibilityState === 'hidden' && isDirty.value && key.value) {
      await syncDraftToBackend(key.value)
    }
  })

  return {
    htmlContent,
    textContent,
    isLoading,
    clearDraft,
    loadedAttachments,
    loadedMacroActions
  }
}
