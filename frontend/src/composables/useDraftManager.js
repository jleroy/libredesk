import { ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { useDraftStore } from '@/stores/draftStore'

/**
 * Composable for managing draft state and persistence
 * @param key - Reactive reference to current draft key
 */
export function useDraftManager (key) {
  const draftStore = useDraftStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoadingDraft = ref(false)
  const isDirty = ref(false)

  /**
   * Load draft from backend for a given key
   */
  const loadDraft = async (key) => {
    if (!key) return
    isLoadingDraft.value = true
    const draft = await draftStore.getDraft(key)
    htmlContent.value = draft.htmlContent
    textContent.value = draft.textContent
    isDirty.value = false
    isLoadingDraft.value = false
  }

  /**
   * Save draft to store
   */
  const saveDraft = async (key) => {
    if (!key || isLoadingDraft.value) return
    await draftStore.setDraft(key, htmlContent.value, textContent.value)
    isDirty.value = false
  }

  /**
   * Clear draft and local state
   */
  const clearDraft = (key) => {
    if (!key) return
    isLoadingDraft.value = true
    draftStore.clearDraft(key)
    htmlContent.value = ''
    textContent.value = ''
    isDirty.value = false
    isLoadingDraft.value = false
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
        draftStore.setDraft(oldKey, htmlContent.value, textContent.value)
        isDirty.value = false
      }

      // Load new draft.
      if (newKey && newKey !== oldKey) {
        await loadDraft(newKey)
      } else if (!newKey && oldKey) {
        // Clear state.
        isLoadingDraft.value = true
        htmlContent.value = ''
        textContent.value = ''
        isDirty.value = false
        isLoadingDraft.value = false
      }
    },
    { immediate: true }
  )

  // Auto-save draft when content changes (debounced to avoid excessive writes)
  watchDebounced(
    [htmlContent, textContent],
    async () => {
      if (!isLoadingDraft.value && key.value) {
        isDirty.value = true
        await saveDraft(key.value)
      }
    },
    { debounce: 500 }
  )

  return {
    htmlContent,
    textContent,
    isLoadingDraft,
    loadDraft,
    saveDraft,
    clearDraft,
    hasDraftContent
  }
}