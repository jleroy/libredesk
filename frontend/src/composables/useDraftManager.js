import { ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { useDraftStore } from '@/stores/draftStore'
import { useConversationStore } from '@/stores/conversation'
import { MACRO_CONTEXT } from '@/constants/conversation'

/**
 * Composable for managing draft state and persistence
 * @param key - Reactive reference to current draft key
 * @param uploadedFiles - Optional reactive reference to uploaded files array
 */
export function useDraftManager (key, uploadedFiles = null) {
  const draftStore = useDraftStore()
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const meta = ref({})
  const isLoading = ref(false)
  const isDirty = ref(false)
  const loadedAttachments = ref([])

  /**
   * Reset all draft state to initial values
   */
  const resetState = () => {
    htmlContent.value = ''
    textContent.value = ''
    meta.value = {}
    isLoading.value = false
    isDirty.value = false
    loadedAttachments.value = []
  }

  /**
   * Load draft from backend for a given key
   */
  const loadDraft = async (key) => {
    if (!key) return
    isLoading.value = true
    isDirty.value = false
    try {
      const draft = await draftStore.getDraft(key)
      console.log("Loaded draft:", draft)
      htmlContent.value = draft.htmlContent
      textContent.value = draft.textContent
      meta.value = draft.meta || {}
      loadedAttachments.value = draft.meta?.attachments || []
    } catch (error) {
      resetState()
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Save draft to store
   */
  const saveDraft = async (key) => {
    if (!key || isLoading.value) return
    if (!isDirty.value) return
    try {
      const macroActions = getCurrentMacroActions()
      let meta = {}
      if (macroActions.length > 0) {
        meta.macro_actions = macroActions
      }
      if (uploadedFiles && uploadedFiles.value && uploadedFiles.value.length > 0) {
        meta.attachments = uploadedFiles.value
      }
      await draftStore.setDraft(key, htmlContent.value, textContent.value, meta)
      isDirty.value = false
    } catch (error) {
      // pass
    }
  }

  /**
   * Clear draft and local state
   */
  const clearDraft = async (key) => {
    if (!key) return
    isLoading.value = true
    try {
      await draftStore.clearDraft(key)
      resetState()
    } catch (error) {
      // pass
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Returns current set macro ID from draft meta
   */
  const getCurrentMacroActions = () => {
    const macro = conversationStore.getMacro(MACRO_CONTEXT.REPLY)
    return macro ? macro.actions : []
  }

  /**
   * Check if draft has content
   */
  const hasDraftContent = () => {
    return textContent.value?.trim() !== ""
  }

  // Watch for key changes to save / load draft.
  watch(
    key,
    async (newKey, oldKey) => {
      // Save old draft first if content has changed.
      if (newKey != oldKey && isDirty.value && hasDraftContent()) {
        try {
          await saveDraft(oldKey)
          isDirty.value = false
        } catch (error) {
          // pass
        }
      }

      // Load new draft.
      if (newKey && newKey !== oldKey) {
        await loadDraft(newKey)
      } else if (!newKey && oldKey) {
        // Clear state.
        isLoading.value = true
        resetState()
        isLoading.value = false
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
    meta,
    htmlContent,
    textContent,
    isLoading,
    loadDraft,
    saveDraft,
    clearDraft,
    hasDraftContent,
    loadedAttachments
  }
}