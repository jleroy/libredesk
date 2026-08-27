// The first page is stubbed empty, so these pass or fail on the ids call alone whatever the DB holds.

const stamp = Date.now()
const agentFirst = `Zzlate${stamp}`
const agentName = `${agentFirst} Agent`
const teamName = `zzz-late-team-${stamp}`
const viewName = `ZZ late view ${stamp}`
const sharedViewName = `ZZ late shared view ${stamp}`
const ruleName = `ZZ late rule ${stamp}`
const macroName = `ZZ late macro ${stamp}`

const leafFilters = (agentId) => ({
  logic: 'AND',
  rules: [
    {
      logic: 'AND',
      rules: [
        {
          model: 'conversations',
          field: 'assigned_user_id',
          operator: 'equals',
          value: String(agentId)
        }
      ]
    }
  ]
})

const blankFirstPage = (resource) => {
  cy.intercept({ method: 'GET', url: `**/api/v1/${resource}*`, query: { page: '1' } }, { body: { data: [] } })
}

const valueCombobox = () => cy.get('[data-cy="filter-value"] button[role="combobox"]')

// Opening the picker focuses its input; a dialog can hold a second popper, so target focus, not the wrapper.
const typeInPicker = (text) => cy.focused().type(text)

describe('Saved lookup values outside the first page', () => {
  const ids = {}

  // radix-vue focuses a trigger it just unmounted, and Cypress fails the test on uncaught errors.
  Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/agents', {
      first_name: agentFirst,
      last_name: 'Agent',
      email: `zz-late-agent-${stamp}@example.com`,
      roles: ['Agent'],
      send_welcome_email: false
    }).then(({ body }) => {
      ids.agent = body.data.id

      cy.api('POST', '/api/v1/teams', {
        name: teamName,
        emoji: '🛰️',
        conversation_assignment_type: 'Round robin',
        timezone: 'UTC',
        max_auto_assigned_conversations: 0
      }).then(({ body: teamBody }) => {
        ids.team = teamBody.data.id

        cy.api('POST', '/api/v1/views/me', {
          name: viewName,
          filters: leafFilters(ids.agent)
        }).then(({ body: viewBody }) => (ids.view = viewBody.data.id))

        cy.api('POST', '/api/v1/shared-views', {
          name: sharedViewName,
          visibility: 'all',
          filters: leafFilters(ids.agent)
        }).then(({ body: sharedBody }) => (ids.sharedView = sharedBody.data.id))

        cy.api('POST', '/api/v1/automations/rules', {
          name: ruleName,
          description: ruleName,
          type: 'new_conversation',
          enabled: false,
          events: [],
          rules: [
            {
              group_operator: 'AND',
              groups: [
                {
                  logical_op: 'AND',
                  rules: [
                    {
                      field: 'assigned_user',
                      value: String(ids.agent),
                      operator: 'equals',
                      field_type: 'conversation'
                    },
                    {
                      field: 'assigned_team',
                      value: String(ids.team),
                      operator: 'equals',
                      field_type: 'conversation'
                    }
                  ]
                }
              ],
              actions: [
                { type: 'assign_user', value: [String(ids.agent)] },
                { type: 'assign_team', value: [String(ids.team)] },
                {
                  type: 'notify',
                  value: [],
                  recipients: [`user:${ids.agent}`, `team:${ids.team}`],
                  subject: 'x',
                  message: 'y'
                }
              ]
            }
          ]
        }).then(({ body: ruleBody }) => (ids.rule = ruleBody.data.id))

        cy.api('POST', '/api/v1/macros', {
          name: macroName,
          message_content: 'probe',
          visibility: 'all',
          visible_when: ['replying'],
          actions: [{ type: 'assign_user', value: [String(ids.agent)] }]
        }).then(({ body: macroBody }) => (ids.macro = macroBody.data.id))
      })
    })
  })

  beforeEach(() => {
    cy.viewport(1400, 1000)
    cy.login()
  })

  describe('user view filters', () => {
    const openEditDialog = () => {
      cy.visit('/inboxes/assigned')
      cy.contains('button', viewName).parent().find('button[aria-haspopup="menu"]').click()
      cy.get('[role="menuitem"]').contains('Edit').click()
      cy.get('[data-cy="filter-row"]').should('have.length', 1)
    }

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      openEditDialog()

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      valueCombobox().should('contain.text', agentName)
    })

    // Inside a dialog, radix closes the picker on Cypress's synthetic typing after the first key,
    // so this pins the search call reaching the server. Real typing lists the results.
    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      openEditDialog()
      valueCombobox().should('contain.text', agentName).click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
    })
  })

  describe('shared view filters', () => {
    const editPath = () => `/admin/conversations/shared-views/${ids.sharedView}/edit`

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      cy.visit(editPath())

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      valueCombobox().should('contain.text', agentName)
    })

    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      valueCombobox().click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  describe('automation rule', () => {
    const editPath = () => `/admin/automations/${ids.rule}/edit`

    it('labels the saved agent and team in conditions and actions', () => {
      blankFirstPage('agents/compact')
      blankFirstPage('teams/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')
      cy.intercept('GET', '**/api/v1/teams/compact?*ids=*').as('teamById')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      cy.wait('@teamById').its('response.statusCode').should('eq', 200)

      cy.get('button[role="combobox"]')
        .filter((_, el) => el.innerText.includes(agentName))
        .should('have.length.at.least', 2)
      cy.get('button[role="combobox"]')
        .filter((_, el) => el.innerText.includes(teamName))
        .should('have.length.at.least', 2)
    })

    it('labels notify recipients instead of showing the raw id', () => {
      blankFirstPage('agents/compact')
      blankFirstPage('teams/compact')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)

      cy.contains(`Agent: ${agentName}`).should('exist')
      cy.contains(`Team: ${teamName}`).should('exist')
      cy.contains(`user:${ids.agent}`).should('not.exist')
      cy.contains(`team:${ids.team}`).should('not.exist')
    })

    it('searches agents from a condition value', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)
      cy.contains('button[role="combobox"]', agentName).first().click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  describe('macro actions', () => {
    const editPath = () => `/admin/conversations/macros/${ids.macro}/edit`

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', macroName)

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      cy.contains('button[role="combobox"]', agentName).should('exist')
    })

    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', macroName)
      cy.contains('button[role="combobox"]', agentName).click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  after(() => {
    cy.login()
    if (ids.view) cy.api('DELETE', `/api/v1/views/me/${ids.view}`)
    if (ids.sharedView) cy.api('DELETE', `/api/v1/shared-views/${ids.sharedView}`)
    if (ids.rule) cy.api('DELETE', `/api/v1/automations/rules/${ids.rule}`)
    if (ids.macro) cy.api('DELETE', `/api/v1/macros/${ids.macro}`)
    if (ids.team) cy.api('DELETE', `/api/v1/teams/${ids.team}`)
    if (ids.agent) cy.api('DELETE', `/api/v1/agents/${ids.agent}`)
  })
})
