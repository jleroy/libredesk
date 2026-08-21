const openSockets = []
const createdInboxes = []

const baseConfig = () => ({
  brand_name: 'Cypress',
  colors: { primary: '#112233' },
  launcher: { position: 'right', spacing: { side: 20, bottom: 20 } },
  session_duration: '4h',
  visitors: { allow_start_conversation: true },
  users: { allow_start_conversation: true }
})

const merge = (base, extra) => {
  const out = { ...base }
  Object.entries(extra || {}).forEach(([key, value]) => {
    out[key] =
      value && typeof value === 'object' && !Array.isArray(value)
        ? merge(base[key] || {}, value)
        : value
  })
  return out
}

Cypress.Commands.add('createLivechatInbox', (configOverrides = {}, inboxOverrides = {}) => {
  const stamp = `${Date.now()}${Cypress._.random(1000, 9999)}`
  const payload = {
    name: `Livechat ${stamp}`,
    channel: 'livechat',
    enabled: true,
    from: `Livechat <livechat+${stamp}@cypress.test>`,
    config: merge(baseConfig(), configOverrides),
    ...inboxOverrides
  }
  cy.login()
  return cy.api('POST', '/api/v1/inboxes', payload).then(({ body }) => {
    const inbox = { id: body.data.id, uuid: body.data.uuid, payload }
    expect(inbox.uuid, 'inbox uuid').to.be.a('string').and.not.be.empty
    createdInboxes.push(inbox.id)
    return inbox
  })
})

Cypress.Commands.add('saveLivechatInbox', (inbox, configOverrides = {}) => {
  const payload = { ...inbox.payload, config: merge(inbox.payload.config, configOverrides) }
  inbox.payload = payload
  return cy.api('PUT', `/api/v1/inboxes/${inbox.id}`, payload).its('status').should('eq', 200)
})

Cypress.Commands.add('widgetInit', (inboxUuid, body = {}, options = {}) =>
  cy
    .request({
      method: 'POST',
      url: '/api/v1/widget/chat/conversations/init',
      headers: { 'X-Libredesk-Inbox-ID': inboxUuid, ...(options.headers || {}) },
      body: { message: body.message || `Visitor hello ${Date.now()}`, ...body },
      failOnStatusCode: options.failOnStatusCode !== false
    })
    .then((res) => {
      if (res.status !== 200) return res
      return {
        status: res.status,
        body: res.body,
        message: res.requestBody?.message,
        sessionToken: res.body.data.session_token,
        conversationUuid: res.body.data.conversation.uuid
      }
    })
)

Cypress.Commands.add('widgetApi', (method, path, sessionToken, inboxUuid, body, options = {}) =>
  cy.request({
    method,
    url: path,
    body,
    failOnStatusCode: options.failOnStatusCode !== false,
    headers: {
      'X-Libredesk-Inbox-ID': inboxUuid,
      ...(sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {}),
      ...(options.headers || {})
    }
  })
)

// keepAlive false lets the server's 20s read deadline expire.
Cypress.Commands.add('openWidgetSocket', (sessionToken, inboxUuid, { keepAlive = true } = {}) =>
  cy.window().then((win) => {
    const scheme = win.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new win.WebSocket(`${scheme}//${win.location.host}/widget/ws`)
    const socket = { ws, received: [], closed: false, pinger: null }
    ws.addEventListener('message', (event) => socket.received.push(JSON.parse(event.data)))
    ws.addEventListener('close', () => {
      socket.closed = true
      win.clearInterval(socket.pinger)
    })
    ws.addEventListener('open', () => {
      ws.send(JSON.stringify({ type: 'join', token: sessionToken, data: { inbox_id: inboxUuid } }))
      if (keepAlive) {
        socket.pinger = win.setInterval(() => ws.send(JSON.stringify({ type: 'ping' })), 5000)
      }
    })
    openSockets.push(socket)
    return socket
  })
)

// cy.wrap(null).should(fn) retries the callback until it passes or times out.
Cypress.Commands.add('waitForFrame', (predicate, errorMsg, timeout = 20000) =>
  cy.wrap(null, { timeout, log: false }).should(() => {
    expect(predicate(), errorMsg).to.be.true
  })
)

Cypress.Commands.add('frameOfType', (socket, type) => {
  const frame = socket.received.find((m) => m.type === type)
  return cy.wrap(frame, { log: false })
})

export const joined = (socket) => () => socket.received.some((m) => m.type === 'joined')

export const gotMessage = (socket, bodyText) => () =>
  socket.received.some(
    (m) =>
      m.type === 'new_message' &&
      `${m.data?.content || ''}${m.data?.text_content || ''}`.includes(bodyText)
  )

// cy.then defers the URL until conversationUuid has actually been assigned.
Cypress.Commands.add('agentReply', (conversationUuidRef, body) =>
  cy.then(() => {
    const uuid = typeof conversationUuidRef === 'function' ? conversationUuidRef() : conversationUuidRef
    return cy
      .api('POST', `/api/v1/conversations/${uuid}/messages`, {
        message: `<p>${body}</p>`,
        private: false,
        sender_type: 'agent'
      })
      .its('status')
      .should('eq', 200)
  })
)

afterEach(() => {
  openSockets.splice(0).forEach((socket) => {
    if (socket.ws) socket.ws.close()
  })
})

after(() => {
  if (!createdInboxes.length) return
  cy.login()
  createdInboxes.splice(0).forEach((id) => {
    cy.api('DELETE', `/api/v1/inboxes/${id}`, null, { failOnStatusCode: false })
  })
})
