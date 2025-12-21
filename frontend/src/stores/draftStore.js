import { defineStore } from 'pinia'
import api from '@/api'

export const useDraftStore = defineStore('drafts', () => {
  /**
   * Get draft from backend 
   */
  const getDraft = async (uuid) => {
    if (!uuid) return { htmlContent: '', textContent: '' }

    try {
      const response = await api.getDraft(uuid)
      const draft = response.data.data
      return {
        htmlContent: draft.content || '',
        // We only store HTML in backend.
        textContent: '',
        meta: draft.meta || {}
      }
    } catch (error) {
      return { htmlContent: '', textContent: '' }
    }
  }

  /**
   * Save draft to backend 
   */
  const setDraft = async (uuid, htmlContent, textContent, meta = {}) => {
    if (!uuid) return

    try {
      await api.saveDraft(uuid, { content: htmlContent, meta })
    } catch (error) {
      // pass
    }
  }

  /**
   * Delete draft from backend 
   */
  const clearDraft = async (uuid) => {
    if (!uuid) return

    try {
      await api.deleteDraft(uuid)
    } catch (error) {
      // pass
    }
  }

  return {
    getDraft,
    setDraft,
    clearDraft,
  }
})