import { ref, computed } from 'vue'

const PAGE_SIZE = 50

// Backs a store whose list lives on the server, ids outside the current page are pinned so their labels still render.
export function createRemoteLookup ({ fetchRows, onError }) {
  const byID = ref({})
  const listIDs = ref([])
  const pinnedIDs = ref([])
  const searching = ref(false)
  let firstPage = null
  let firstPageIDs = null
  let searchSeq = 0

  const rows = computed(() => {
    const seen = new Set()
    const out = []
    for (const id of [...listIDs.value, ...pinnedIDs.value]) {
      if (seen.has(id)) continue
      seen.add(id)
      const row = byID.value[id]
      if (row) out.push(row)
    }
    return out
  })

  const merge = (fetched) => {
    for (const row of fetched) byID.value[row.id] = row
    return fetched
  }

  const load = async (params) => {
    const response = await fetchRows({ page_size: PAGE_SIZE, ...params })
    return merge(response?.data?.data || [])
  }

  const fetchFirstPage = () => {
    if (firstPage) return firstPage
    firstPage = load({ page: 1 })
      .then((fetched) => {
        firstPageIDs = fetched.map((row) => row.id)
        listIDs.value = firstPageIDs
      })
      .catch((error) => {
        firstPage = null
        onError(error)
      })
    return firstPage
  }

  const search = async (query) => {
    const term = (query || '').trim()
    const seq = ++searchSeq
    if (!term && firstPageIDs) {
      listIDs.value = firstPageIDs
      searching.value = false
      return
    }
    searching.value = true
    try {
      const fetched = await load(term ? { q: term, page: 1 } : { page: 1 })
      if (seq !== searchSeq) return
      listIDs.value = fetched.map((row) => row.id)
    } catch (error) {
      if (seq === searchSeq) onError(error)
    } finally {
      if (seq === searchSeq) searching.value = false
    }
  }

  const ensureIDs = async (ids) => {
    const wanted = [...new Set((ids || []).map(Number).filter((id) => Number.isInteger(id) && id > 0))]
    if (!wanted.length) return
    pinnedIDs.value = [...new Set([...pinnedIDs.value, ...wanted])]
    if (wanted.some((id) => !byID.value[id])) await fetchFirstPage()
    const missing = wanted.filter((id) => !byID.value[id])
    if (!missing.length) return
    try {
      await load({ ids: missing.join(',') })
    } catch (error) {
      onError(error)
    }
  }

  const patch = (id, changes) => {
    const row = byID.value[id]
    if (row) byID.value[id] = { ...row, ...changes }
  }

  return { rows, searching, fetchFirstPage, search, ensureIDs, patch }
}
