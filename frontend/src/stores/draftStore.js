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
        textContent: ''
      }
    } catch (error) {
      // If draft doesn't exist (404), return empty
      if (error.response?.status === 404 || error.response?.status === 500) {
        return { htmlContent: '', textContent: '' }
      }
      console.error('Failed to fetch draft:', error)
      return { htmlContent: '', textContent: '' }
    }
  }

  /**
   * Save draft to backend 
   */
  const setDraft = async (uuid, htmlContent, textContent) => {
    if (!uuid) return
    
    const isEmpty = (!htmlContent || htmlContent.trim() === '') &&
                    (!textContent || textContent.trim() === '')

    if (isEmpty) return

    try {
      await api.saveDraft(uuid, { content: htmlContent })
    } catch (error) {
      console.error('Failed to save draft:', error)
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
      // 404 is fine - draft doesn't exist
      if (error.response?.status !== 404) {
        console.error('Failed to delete draft:', error)
      }
    }
  }

  return {
    getDraft,
    setDraft,
    clearDraft,
  }
})