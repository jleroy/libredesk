import { joined } from '../../../support/livechat'

describe('Live chat widget session and auth', () => {
  let inbox
  let session

  beforeEach(() => {
    cy.createLivechatInbox().then((created) => {
      inbox = created
    })
    cy.then(() => cy.widgetInit(inbox.uuid)).then((res) => {
      session = res
    })
  })

  it('issues a session token that identifies the visitor', () => {
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/auth/me', session.sessionToken, inbox.uuid)
        .then(({ status, body }) => {
          expect(status).to.eq(200)
          expect(body.data, 'auth/me returned no user').to.not.be.null
        })
    )
  })

  it('rejects a missing, garbage, or foreign-inbox token', () => {
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/auth/me', null, inbox.uuid, null, {
          failOnStatusCode: false
        })
        .its('status')
        .should('eq', 401)
    )
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/auth/me', 'not-a-real-token', inbox.uuid, null, {
          failOnStatusCode: false
        })
        .its('status')
        .should('eq', 401)
    )

    let other
    cy.createLivechatInbox().then((created) => {
      other = created
    })
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/auth/me', session.sessionToken, other.uuid, null, {
          failOnStatusCode: false
        })
        .its('status')
        .should('eq', 401)
    )
  })

  it('refuses a socket join carrying a bad token', () => {
    cy.visit('/inboxes/all')
    let socket
    cy.then(() => cy.openWidgetSocket('not-a-real-token', inbox.uuid)).then((s) => {
      socket = s
    })
    cy.then(() =>
      cy.waitForFrame(
        () => socket.received.some((m) => m.type === 'error'),
        'bad token was allowed to join'
      )
    )
    cy.then(() => expect(joined(socket)(), 'bad token still joined').to.be.false)
  })

  it('rejects a numeric or unknown inbox id', () => {
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/settings', null, '1', null, { failOnStatusCode: false })
        .its('status')
        .should('eq', 400)
    )
    cy.then(() =>
      cy
        .widgetApi(
          'GET',
          '/api/v1/widget/chat/settings',
          null,
          '00000000-0000-0000-0000-000000000000',
          null,
          { failOnStatusCode: false }
        )
        .its('status')
        .should('eq', 400)
    )
  })

  it('locks out the widget once the inbox is disabled', () => {
    cy.then(() => cy.api('PUT', `/api/v1/inboxes/${inbox.id}`, { ...inbox.payload, enabled: false }))
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/settings', null, inbox.uuid, null, {
          failOnStatusCode: false
        })
        .its('status')
        .should('eq', 400)
    )
    cy.then(() =>
      cy.widgetInit(inbox.uuid, {}, { failOnStatusCode: false }).its('status').should('eq', 400)
    )
  })

  it('blocks a visitor whose IP is on the inbox blocklist', () => {
    cy.then(() => cy.saveLivechatInbox(inbox, { blocked_ips: ['0.0.0.0/0'] }))
    cy.then(() =>
      cy
        .widgetApi('GET', '/api/v1/widget/chat/settings', null, inbox.uuid, null, {
          failOnStatusCode: false
        })
        .its('status')
        .should('eq', 403)
    )
  })

  it('keeps the session usable across a page reload', () => {
    cy.visit('/inboxes/all')
    cy.reload()
    let socket
    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      socket = s
    })
    cy.then(() => cy.waitForFrame(joined(socket), 'session did not survive a reload'))
  })
})
