import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

export const useTeamStore = defineStore('team', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getTeamsCompact(params),
        onError: showFetchError
    })

    const teams = lookup.rows
    const options = computed(() => teams.value.map(team => ({
        label: team.name,
        value: String(team.id),
        emoji: team.emoji,
    })))

    return {
        teams,
        options,
        searching: lookup.searching,
        fetchTeams: lookup.fetchFirstPage,
        searchTeams: lookup.search,
        ensureTeamIDs: lookup.ensureIDs,
    }
})
