import { ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useSharedViewStore = defineStore('sharedViewStore', () => {
    const sharedViewList = ref([])
    const emitter = useEmitter()
    const isLoaded = ref(false)

    const loadSharedViews = async (force = false) => {
        if (!force && isLoaded.value) return
        try {
            const response = await api.getSharedViews()
            sharedViewList.value = response?.data?.data || []
            isLoaded.value = true
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }

    const refresh = () => loadSharedViews(true)

    const reset = () => {
        sharedViewList.value = []
        isLoaded.value = false
    }

    return {
        sharedViewList,
        isLoaded,
        loadSharedViews,
        refresh,
        reset
    }
})
