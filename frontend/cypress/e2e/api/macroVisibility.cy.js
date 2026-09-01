const stamp = Date.now()
const password = 'StrongPass!123'
const prefix = `macro-vis-${stamp}`
const teamOneName = `${prefix}-team-one`
const teamTwoName = `${prefix}-team-two`
const agentOneEmail = `macro.vis.one.${stamp}@example.com`
const agentTwoEmail = `macro.vis.two.${stamp}@example.com`

const macroName = (suffix) => `${prefix} ${suffix}`

const createTeam = (name) =>
  cy.api('POST', '/api/v1/teams', {
    name,
    emoji: '🧪',
    timezone: 'UTC',
    conversation_assignment_type: 'Manual'
  })

const createAgent = (firstName, email, teamName) =>
  cy
    .api('POST', '/api/v1/agents', {
      first_name: firstName,
      email,
      roles: ['Agent'],
      send_welcome_email: false,
      teams: [teamName]
    })
    .then(({ body }) => {
      const id = body.data.id
      // Create leaves the agent without a password, so set one to log in as them.
      return cy
        .api('PUT', `/api/v1/agents/${id}`, {
          first_name: firstName,
          email,
          roles: ['Agent'],
          enabled: true,
          availability_status: 'offline',
          teams: [teamName],
          new_password: password
        })
        .then(() => id)
    })

const loginAs = (email) => {
  cy.session(
    email,
    () => {
      cy.visit('/')
      cy.get('#email').clear().type(email)
      cy.get('#password').clear().type(password, { log: false })
      cy.contains('button', 'Sign in').click()
      cy.url().should('include', '/inboxes')
    },
    {
      validate () {
        cy.request('/api/v1/agents/me').its('status').should('eq', 200)
      }
    }
  )
}

const searchNames = () =>
  cy
    .api('GET', `/api/v1/macros/search?q=${encodeURIComponent(prefix)}`)
    .then(({ status, body }) => {
      expect(status).to.eq(200)
      return body.data.map((m) => m.name).sort()
    })

describe('API: macro visibility scoping', () => {
  const macroIds = []
  let agentOneId, agentTwoId, teamOneId, teamTwoId

  before(() => {
    cy.login()
    createTeam(teamOneName).then(({ body }) => {
      teamOneId = body.data.id
    })
    createTeam(teamTwoName).then(({ body }) => {
      teamTwoId = body.data.id
    })
    createAgent('MacroVisOne', agentOneEmail, teamOneName).then((id) => {
      agentOneId = id
    })
    createAgent('MacroVisTwo', agentTwoEmail, teamTwoName).then((id) => {
      agentTwoId = id
    })

    cy.then(() => {
      const macros = [
        { suffix: 'everyone', visibility: 'all' },
        { suffix: 'agent one', visibility: 'user', user_id: String(agentOneId) },
        { suffix: 'agent two', visibility: 'user', user_id: String(agentTwoId) },
        { suffix: 'team one', visibility: 'team', team_id: String(teamOneId) },
        { suffix: 'team two', visibility: 'team', team_id: String(teamTwoId) }
      ]
      macros.forEach(({ suffix, ...rest }) => {
        cy.api('POST', '/api/v1/macros', {
          name: macroName(suffix),
          message_content: `<p>${suffix}</p>`,
          visible_when: ['replying'],
          actions: [],
          ...rest
        }).then(({ body }) => macroIds.push(body.data.id))
      })
    })
  })

  after(() => {
    cy.login()
    cy.then(() => {
      macroIds.forEach((id) => cy.api('DELETE', `/api/v1/macros/${id}`))
      if (agentOneId) cy.api('DELETE', `/api/v1/agents/${agentOneId}`)
      if (agentTwoId) cy.api('DELETE', `/api/v1/agents/${agentTwoId}`)
      if (teamOneId) cy.api('DELETE', `/api/v1/teams/${teamOneId}`)
      if (teamTwoId) cy.api('DELETE', `/api/v1/teams/${teamTwoId}`)
    })
  })

  it('gives the first agent the shared macro, their own macro and their team macro', () => {
    loginAs(agentOneEmail)
    searchNames().should('deep.eq', [
      macroName('agent one'),
      macroName('everyone'),
      macroName('team one')
    ])
  })

  it('gives the second agent a different set from the same macros', () => {
    loginAs(agentTwoEmail)
    searchNames().should('deep.eq', [
      macroName('agent two'),
      macroName('everyone'),
      macroName('team two')
    ])
  })

  it('hides every scoped macro from an agent who owns none of them', () => {
    cy.login()
    searchNames().should('deep.eq', [macroName('everyone')])
  })
})
