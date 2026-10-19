// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useDraftManager } from './useDraftManager'

const { drafts, api, store } = vi.hoisted(() => {
  const drafts = new Map()
  return {
    drafts,
    api: { saveDraft: vi.fn().mockResolvedValue({}), deleteDraft: vi.fn().mockResolvedValue({}) },
    store: {
      macros: {},
      draftsReady: Promise.resolve(),
      getMacro: () => null,
      getDraft: (uuid, type) => drafts.get(`${uuid}:${type}`),
      setDraft: (uuid, type, draft) => drafts.set(`${uuid}:${type}`, draft),
      removeDraft: (uuid, type) => drafts.delete(`${uuid}:${type}`)
    }
  }
})
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => store }))
vi.mock('@main/api', () => ({ default: api }))

let scopes
const settle = async () => {
  await nextTick()
  await vi.advanceTimersByTimeAsync(600)
}
const openDraft = () => {
  const scope = effectScope()
  scopes.push(scope)
  const uuid = ref('conversation-a')
  const type = ref('reply')
  const draft = scope.run(() => useDraftManager(uuid, type))
  return { ...draft, uuid, type }
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  drafts.clear()
  scopes = []
})
afterEach(() => {
  scopes.forEach(scope => scope.stop())
  vi.useRealTimers()
})

describe('reply draft recipients', () => {
  it('restores recipient edits with the body after remount', async () => {
    const draft = openDraft()
    await settle()
    draft.htmlContent.value = '<p>Keep this reply.</p>'
    draft.recipients.value = { to: 'other@example.com', cc: '', bcc: 'audit@example.com', send_from: 'alias@example.com' }
    await settle()
    const restored = openDraft()
    await settle()
    expect(restored.htmlContent.value).toBe('<p>Keep this reply.</p>')
    expect(restored.recipients.value).toEqual(draft.recipients.value)
    expect(api.saveDraft).toHaveBeenCalledTimes(1)
  })

  it('preserves an explicitly removed recipient without message text', async () => {
    const draft = openDraft()
    await settle()
    draft.recipients.value = { cc: '' }
    await settle()
    expect(drafts.get('conversation-a:reply').meta.recipients).toEqual({ cc: '' })
    expect(api.deleteDraft).not.toHaveBeenCalled()
  })

  it('isolates recipients across conversations and private notes', async () => {
    const draft = openDraft()
    await settle()
    draft.recipients.value = { bcc: 'audit@example.com' }
    draft.type.value = 'private_note'
    await settle()
    expect(draft.recipients.value).toEqual({})
    draft.htmlContent.value = '<p>Private note</p>'
    await settle()
    expect(drafts.get('conversation-a:private_note').meta).not.toHaveProperty('recipients')
    draft.type.value = 'reply'
    await settle()
    expect(draft.recipients.value).toEqual({ bcc: 'audit@example.com' })
    draft.uuid.value = 'conversation-b'
    await settle()
    expect(draft.recipients.value).toEqual({})
    draft.recipients.value = { to: 'second@example.com' }
    draft.uuid.value = 'conversation-a'
    await settle()
    expect(draft.recipients.value).toEqual({ bcc: 'audit@example.com' })
    expect(drafts.get('conversation-b:reply').meta.recipients).toEqual({ to: 'second@example.com' })
  })

  it('clears recipients after a successful send without resurrecting the draft', async () => {
    const draft = openDraft()
    await settle()
    draft.recipients.value = { to: '', cc: 'copy@example.com' }
    await settle()
    draft.clearDraft()
    await settle()
    expect(draft.recipients.value).toEqual({})
    expect(drafts.size).toBe(0)
    expect(api.deleteDraft).toHaveBeenCalledTimes(1)
  })

  it('loads older drafts and ignores malformed recipient metadata', async () => {
    drafts.set('conversation-a:reply', { content: '<p>Older draft</p>', meta: {} })
    const draft = openDraft()
    await settle()
    expect(draft.recipients.value).toEqual({})
    drafts.set('conversation-b:reply', { content: '<p>Reply</p>', meta: { recipients: { to: [], cc: '', bcc: null, extra: 'ignored' } } })
    draft.uuid.value = 'conversation-b'
    await settle()
    expect(draft.recipients.value).toEqual({ cc: '' })
  })
})
