import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'
import api from '@/api'

const STORAGE_KEY = 'libredesk-conversation-drafts'
const MAX_ENTRIES = 10

export const useDraftStore = defineStore('drafts', () => {
  const drafts = useStorage(STORAGE_KEY, {}, localStorage, {
    serializer: {
      read: (v) => {
        try {
          return v ? JSON.parse(v) : {}
        } catch (error) {
          console.error('Failed to parse drafts:', error)
          return {}
        }
      },
      write: (v) => JSON.stringify(v)
    }
  })
  
  const getDraft = async (uuid) => {
    if (!uuid) return { htmlContent: '', textContent: '' }
    
    // First check localStorage
    const localDraft = drafts.value[uuid]
    
    // Then fetch from backend
    try {
      const response = await api.getDraft(uuid)
      const backendDraft = response.data.data
      
      // If backend has a draft, use it and update localStorage
      if (backendDraft?.content) {
        const draft = {
          htmlContent: backendDraft.content || '',
          textContent: '',
          timestamp: new Date(backendDraft.updated_at).getTime()
        }
        
        // Update localStorage with backend data
        drafts.value[uuid] = draft
        return draft
      }
    } catch (error) {
      // If backend fails or returns 404, fall back to localStorage
      if (error.response?.status !== 404) {
        console.error('Failed to fetch draft from backend:', error)
      }
    }
    
    // Return localStorage draft or empty
    return localDraft || { htmlContent: '', textContent: '' }
  }

  const setDraft = (uuid, htmlContent, textContent) => {
    if (!uuid) return
    
    const isEmpty = (!htmlContent || htmlContent.trim() === '') &&
                  (!textContent || textContent.trim() === '')

    if (isEmpty) return
    
    drafts.value[uuid] = {
      htmlContent,
      textContent,
      timestamp: Date.now()
    }
    
    // Sync to backend
    api.saveDraft(uuid, { content: htmlContent }).catch(err => {
      console.error('Failed to sync draft to backend:', err)
    })
    
    const keys = Object.keys(drafts.value)
    if (keys.length > MAX_ENTRIES) {
      const sorted = keys
        .map(k => [k, drafts.value[k].timestamp])
        .sort((a, b) => a[1] - b[1])
      const removeCount = keys.length - MAX_ENTRIES
      for (let i = 0; i < removeCount; i++) {
        delete drafts.value[sorted[i][0]]
      }
    }
  }

  const clearDraft = (uuid) => {
    if (!uuid) return
    delete drafts.value[uuid]
    
    api.deleteDraft(uuid).catch(err => {
      if (error.response?.status !== 404) {
        console.error('Failed to delete draft from backend:', err)
      }
    })
  }
  
  const clearAllDrafts = () => {
    drafts.value = {}
  }

  return {
    drafts,
    getDraft,
    setDraft,
    clearDraft,
    clearAllDrafts
  }
})