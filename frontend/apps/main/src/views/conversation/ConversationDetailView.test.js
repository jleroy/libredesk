// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import mitt from 'mitt'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'

const containerWidth = ref(0)
const compactViewport = ref(false)

vi.mock('@vueuse/core', async (importOriginal) => ({
  ...await importOriginal(),
  useElementSize: () => ({ width: containerWidth, height: ref(600) }),
  useMediaQuery: () => compactViewport
}))
vi.mock('@main/stores/conversation', () => ({
  useConversationStore: () => ({
    current: {},
    conversation: { loading: false },
    messages: { loading: false }
  })
}))
vi.mock('@main/features/conversation/Conversation.vue', () => ({
  default: { props: ['detailsOpen'], setup: () => () => h('div') }
}))
vi.mock('@main/features/conversation/sidebar/ConversationSideBar.vue', () => ({
  default: { setup: () => () => h('div') }
}))

import ConversationDetailView from './ConversationDetailView.vue'

let app
let host
let emitter

const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}

const mount = async () => {
  host = document.createElement('div')
  document.body.append(host)
  emitter = mitt()
  app = createApp(ConversationDetailView)
  app.config.globalProperties.$t = (key) => key
  app.config.globalProperties.emitter = emitter
  app.mount(host)
  await settle()
}

const unmount = () => {
  app?.unmount()
  host?.remove()
  app = undefined
  host = undefined
}

const sidebar = () => host.querySelector('[data-panel-collapsible]')

beforeEach(() => {
  localStorage.clear()
  containerWidth.value = 0
  compactViewport.value = false
  vi.stubGlobal('ResizeObserver', class {
    observe() {}
    unobserve() {}
    disconnect() {}
  })
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => ({
    width: containerWidth.value || 4,
    height: 600,
    top: 0,
    left: 0,
    bottom: 600,
    right: containerWidth.value || 4
  }))
})

afterEach(() => {
  unmount()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('conversation sidebar preferences', () => {
  it.each([true, false])('keeps the saved open state %s when returning from search', async (open) => {
    localStorage.setItem('conversationSidebarOpen', String(open))
    localStorage.setItem('conversationSidebarWidth', '340')
    for (let visit = 0; visit < 2; visit++) {
      await mount()
      expect(localStorage.getItem('conversationSidebarOpen')).toBe(String(open))
      expect(localStorage.getItem('conversationSidebarWidth')).toBe('340')
      containerWidth.value = 1000
      await settle()
      expect(sidebar().getAttribute('data-state')).toBe(open ? 'expanded' : 'collapsed')
      expect(localStorage.getItem('conversationSidebarOpen')).toBe(String(open))
      expect(Number(localStorage.getItem('conversationSidebarWidth'))).toBeCloseTo(340)
      unmount()
      containerWidth.value = 0
    }
  })

  it('keeps the desktop preference when switching to a drawer and back', async () => {
    localStorage.setItem('conversationSidebarOpen', 'true')
    await mount()
    containerWidth.value = 1000
    await settle()
    compactViewport.value = true
    await settle()
    expect(sidebar()).toBeNull()
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('true')
    compactViewport.value = false
    await settle()
    expect(sidebar().getAttribute('data-state')).toBe('expanded')
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('true')
  })

  it('keeps the width when the user closes and reopens the sidebar', async () => {
    localStorage.setItem('conversationSidebarOpen', 'true')
    localStorage.setItem('conversationSidebarWidth', '340')
    await mount()
    containerWidth.value = 1000
    await settle()
    emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)
    await settle()
    expect(sidebar().getAttribute('data-state')).toBe('collapsed')
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('false')
    expect(Number(localStorage.getItem('conversationSidebarWidth'))).toBeCloseTo(340)
    emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)
    await settle()
    expect(sidebar().getAttribute('data-state')).toBe('expanded')
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('true')
    expect(Number(localStorage.getItem('conversationSidebarWidth'))).toBeCloseTo(340)
  })

  it('keeps the desktop preference when the conversation area becomes narrow', async () => {
    localStorage.setItem('conversationSidebarOpen', 'true')
    localStorage.setItem('conversationSidebarWidth', '340')
    await mount()
    containerWidth.value = 1000
    await settle()
    containerWidth.value = 600
    await settle()
    expect(sidebar()).toBeNull()
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('true')
    expect(Number(localStorage.getItem('conversationSidebarWidth'))).toBeCloseTo(340)
    containerWidth.value = 1000
    await settle()
    expect(sidebar().getAttribute('data-state')).toBe('expanded')
    expect(localStorage.getItem('conversationSidebarOpen')).toBe('true')
  })
})
