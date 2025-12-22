import { ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
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
 * @param key - Reactive reference to current draft key
 * @param uploadedFiles - Optional reactive reference to uploaded files array
 */
export function useDraftManager (key, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const isDirty = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])

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
   * Load draft from backend for a given key
   */
  const loadDraft = async (draftKey) => {
    if (!draftKey) return
    isLoading.value = true
    isDirty.value = false
    try {
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
   * Save draft to backend
   */
  const saveDraft = async (draftKey) => {
    if (!draftKey || isLoading.value || !isDirty.value) return
    try {
      const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
      const draftMeta = {}
      if (macroActions.length > 0) {
        draftMeta.macro_actions = macroActions
      }
      if (uploadedFiles?.value?.length > 0) {
        draftMeta.attachments = uploadedFiles.value
      }
      await api.saveDraft(draftKey, { content: htmlContent.value, meta: draftMeta })
      isDirty.value = false
    } catch (error) {
      // Silent fail for drafts
    }
  }

  /**
   * Clear draft and local state
   */
  const clearDraft = async (draftKey) => {
    if (!draftKey) return
    isLoading.value = true
    try {
      await api.deleteDraft(draftKey)
      resetState()
    } catch (error) {
      // Silent fail for drafts
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

  // Watch for key changes to save / load draft
  watch(
    key,
    async (newKey, oldKey) => {
      // Save old draft if dirty
      if (newKey !== oldKey && isDirty.value && hasDraftContent()) {
        await saveDraft(oldKey)
      }

      // Load new draft or clear state
      if (newKey && newKey !== oldKey) {
        await loadDraft(newKey)
      } else if (!newKey && oldKey) {
        resetState()
      }
    },
    { immediate: true }
  )

  // Auto-save draft when content, macro, or uploaded files change (debounced)
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
    async () => {
      if (!isLoading.value && key.value) {
        isDirty.value = true
        await saveDraft(key.value)
      }
    },
    { debounce: 250, deep: true }
  )

  return {
    htmlContent,
    textContent,
    isLoading,
    clearDraft,
    loadedAttachments,
    loadedMacroActions
  }
}
