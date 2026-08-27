import { describe, it, expect, vi } from 'vitest'
import { createRemoteLookup } from './remote-lookup'

const rows = (values) => ({ data: { data: values } })

describe('createRemoteLookup', () => {
  it('seeds the list from the first page and only fetches it once', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await Promise.all([lookup.fetchFirstPage(), lookup.fetchFirstPage()])

    expect(fetchRows).toHaveBeenCalledTimes(1)
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('replaces the list with server results when searching', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'zebra' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    await lookup.search('zeb')

    expect(fetchRows).toHaveBeenLastCalledWith({ page_size: 50, q: 'zeb', page: 1 })
    expect(lookup.rows.value).toEqual([{ id: 9, name: 'zebra' }])
    expect(lookup.searching.value).toBe(false)
  })

  it('does not let a late first page replace newer search results', async () => {
    let resolveFirstPage
    const fetchRows = vi.fn()
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirstPage = resolve }))
      .mockResolvedValueOnce(rows([{ id: 2, name: 'search result' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    const firstPage = lookup.fetchFirstPage()
    await lookup.search('search')
    resolveFirstPage(rows([{ id: 1, name: 'first page' }]))
    await firstPage

    expect(lookup.rows.value).toEqual([{ id: 2, name: 'search result' }])
  })

  it('drops results of a search that a newer one has superseded', async () => {
    let resolveFirst
    const fetchRows = vi.fn()
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockResolvedValueOnce(rows([{ id: 2, name: 'second' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    const stale = lookup.search('a')
    await lookup.search('b')
    resolveFirst(rows([{ id: 1, name: 'first' }]))
    await stale

    expect(lookup.rows.value).toEqual([{ id: 2, name: 'second' }])
  })

  it('keeps pinned ids in the list across searches', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 7, name: 'pinned' }]))
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([7])
    await lookup.search('a')

    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }, { id: 7, name: 'pinned' }])
  })

  it('resolves ids from the first page instead of fetching them separately', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 3, name: 'c' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([3])
    await lookup.ensureIDs([3])

    expect(fetchRows).toHaveBeenCalledTimes(1)
    expect(fetchRows).toHaveBeenCalledWith({ page_size: 50, page: 1 })
  })

  it('fetches only the ids the first page does not hold', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'z' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([1, 9])

    expect(fetchRows).toHaveBeenCalledTimes(2)
    expect(fetchRows).toHaveBeenLastCalledWith({ page_size: 50, ids: '9' })
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }, { id: 9, name: 'z' }])
  })

  it('restores the cached first page when a search is cleared', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'zebra' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    await lookup.search('zeb')
    await lookup.search('')

    expect(fetchRows).toHaveBeenCalledTimes(2)
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('reports a failed first page and allows a retry', async () => {
    const onError = vi.fn()
    const fetchRows = vi.fn()
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError })

    await lookup.fetchFirstPage()
    expect(onError).toHaveBeenCalledTimes(1)
    expect(lookup.rows.value).toEqual([])

    await lookup.fetchFirstPage()
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('patches a cached row in place', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 1, availability_status: 'offline' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    lookup.patch(1, { availability_status: 'online' })

    expect(lookup.rows.value).toEqual([{ id: 1, availability_status: 'online' }])
  })
})
