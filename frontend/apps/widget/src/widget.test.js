// @vitest-environment jsdom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

const source = readFileSync(resolve(import.meta.dirname, '../../../../static/widget.js'), 'utf8')
let widget

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'setTimeout', 'clearTimeout', 'performance'] })
  window.history.replaceState({}, '', '/pricing')
  delete window.__libredeskWidgetLoaded
  window.eval(source)
  vi.spyOn(window.Libredesk.prototype, 'init').mockImplementation(() => {})
  widget = new window.Libredesk({ baseURL: window.location.origin, inboxID: 'test' })
  vi.spyOn(widget, 'renderPreviews').mockImplementation(() => { widget.previewHost = {} })
  vi.spyOn(widget, 'postToIframe').mockImplementation(() => {})
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
})

afterEach(() => {
  widget.stopPageTracking()
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('campaign navigation', () => {
  const deliver = url => {
    const contentWindow = {}
    widget.iframe = { contentWindow }
    widget.handleMessage({
      source: contentWindow,
      origin: window.location.origin,
      data: { type: 'SHOW_CAMPAIGN', delivery: { id: 'pricing-message' }, url }
    })
  }

  test('drops a delayed response after navigation before the next timer tick', () => {
    const requestedURL = window.location.href
    window.history.pushState({}, '', '/checkout')

    deliver(requestedURL)

    expect(widget.campaignData).toBeNull()
    expect(widget.renderPreviews).not.toHaveBeenCalled()
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'dropped', id: 'pricing-message' })
  })

  test('shows responses for the current page', () => {
    deliver(window.location.href)

    expect(widget.campaignData).toEqual({ id: 'pricing-message' })
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'displayed', id: 'pricing-message' })
  })

  test.each([false, true])('preserves an immediate event with page tracking enabled: %s', pageTracking => {
    if (pageTracking) widget.startPageTracking()
    widget.campaignDelay = 3
    widget.startCampaignTracking()
    widget.trackEvent('old-page')
    vi.advanceTimersByTime(60000)
    widget.postToIframe.mockClear()

    window.history.pushState({}, '', '/checkout')
    widget.trackEvent('checkout-viewed')
    vi.advanceTimersByTime(3000)

    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_CONTEXT', context: {
      url: window.location.href, mobile: widget.isMobile, active_seconds: 3, event: 'checkout-viewed'
    } })
  })

  test('clears old page events and visible messages on navigation', () => {
    widget.startPageTracking()
    widget.startCampaignTracking()
    widget.trackEvent('pricing-viewed')
    widget.campaignData = { id: 'pricing-message' }

    window.history.pushState({}, '', '/checkout')
    vi.advanceTimersByTime(2000)

    expect(widget.campaignData).toBeNull()
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_CONTEXT', context: {
      url: window.location.href, mobile: widget.isMobile, active_seconds: 2, event: ''
    } })
  })
})
