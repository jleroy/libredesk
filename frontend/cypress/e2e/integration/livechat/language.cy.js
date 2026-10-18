const home = { en: 'Home', de: 'Startseite', zh: '主页' }

describe('Live chat widget language chosen by the host page', () => {
  it('loads in the language the host page passes', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de-DE' })
      cy.get('iframe[src*="/widget?inbox_id="]')
        .should('have.attr', 'src')
        .and('contain', 'lang=de-DE')
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('matches a bare language tag to a shipped translation', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('switches language when the host page calls setLanguage', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-US' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.window().then((win) => win.Libredesk.setLanguage('de-DE'))
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('keeps the latest language when switches overlap', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.intercept('GET', '**/api/v1/lang/de-DE', (req) => {
        req.on('response', (res) => res.setDelay(1000))
      })
      cy.visitWidgetHost(inbox.uuid, { language: 'en-US' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.window().then((win) => {
        win.Libredesk.setLanguage('de-DE')
        win.Libredesk.setLanguage('zh-CN')
      })
      // eslint-disable-next-line cypress/no-unnecessary-waiting -- the delayed German response must land and be ignored
      cy.wait(1500)
      cy.widgetBody().contains(home.zh).should('be.visible')
      cy.widgetBody().contains(home.de).should('not.exist')
    })
  })

  it('keeps a language the admin fixed for the inbox', () => {
    cy.createLivechatInbox({ language: 'en-US' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de-DE' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
    })
  })
})

describe('Widget and help center languages', () => {
  let helpCenter

  before(() => {
    const locales = ['de', 'en-US', 'en-GB', 'zh-Hans', 'zh-Hant']
    cy.login()
    cy.api('POST', '/api/v1/help-centers', {
      name: 'Language regression articles',
      page_title: 'Language regression articles',
      slug: `language-regression-${Date.now()}`,
      default_locale: 'de',
      allowed_locales: locales
    }).then(({ body }) => {
      helpCenter = body.data
      cy.wrap(locales).each((locale) =>
        cy.api('POST', `/api/v1/help-centers/${helpCenter.id}/collections`, {
          name: `Articles ${locale}`,
          locale,
          is_published: true
        }).then(({ body: collection }) =>
          cy.api('POST', `/api/v1/collections/${collection.data.id}/articles`, {
            title: `Guide ${locale}`,
            content: `<p>Article content ${locale}</p>`,
            locale,
            status: 'published'
          })
        )
      )
    })
  })

  const inboxConfig = () => ({
    language: 'auto',
    help: { help_center_id: helpCenter.id, visitors: { tab: true }, users: { tab: true } }
  })

  it('keeps article regions when labels match and closes the previous article', () => {
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-GB' })
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-GB')
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()
      cy.widgetBody().contains('Articles en-GB').click()
      cy.widgetBody().contains('Guide en-GB').click()
      cy.widgetBody().find('iframe').its('0.contentDocument.body')
        .should('contain', 'Article content en-GB')
      cy.window().its('Libredesk.isExpanded').should('eq', true)

      cy.window().then((win) => win.Libredesk.setLanguage('en-US'))
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-US')
      cy.widgetBody().contains('Articles en-US').should('be.visible')
      cy.widgetBody().find('iframe').should('not.exist')
      cy.window().its('Libredesk.isExpanded').should('eq', false)

      cy.request(`/api/v1/public/help-centers/${helpCenter.slug}/tree?locale=en-AU`)
        .its('body.data.tree.0.locale').should('eq', 'en-US')
    })
  })

  it('uses available HC fallbacks and keeps Chinese writing systems separate', () => {
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'ko-KR' })
      cy.wait('@localizedHelp').then(({ request, response }) => {
        const params = new URL(request.url).searchParams
        expect(params.get('locale')).to.eq('ko-KR')
        expect(params.get('fallback_locale')).to.eq('en-US')
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()

      for (const [requested, resolved] of [['zh-TW', 'zh-Hant'], ['zh-CN', 'zh-Hans'], ['fr-CA', 'de']]) {
        cy.window().then((win) => win.Libredesk.setLanguage(requested))
        cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', resolved)
        cy.widgetBody().contains(`Articles ${resolved}`).should('be.visible')
      }

      cy.window().then((win) => win.Libredesk.setLanguage(null))
      cy.wait('@localizedHelp').then(({ request, response }) => {
        expect(new URL(request.url).searchParams.get('fallback_locale')).to.eq(null)
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetBody().contains('Articles en-US').should('be.visible')
    })
  })
})
