package user

import (
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type userRow struct {
	FirstName string
	LastName  string
	Email     string
	ExtID     string
	Type      string
}

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "user_contact")
	lo := logf.New(logf.Opts{})
	mgr, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatalf("creating user manager: %v", err)
	}
	return mgr, db
}

func newContact(email, extID, firstName, lastName string) *models.User {
	return &models.User{
		Email:          null.NewString(email, email != ""),
		ExternalUserID: null.NewString(extID, extID != ""),
		FirstName:      firstName,
		LastName:       lastName,
	}
}

func resolve(t *testing.T, u *Manager, c *models.User, policy models.ContactPolicy) {
	t.Helper()
	if err := u.ResolveContact(c, policy); err != nil {
		t.Fatalf("ResolveContact: %v", err)
	}
	if c.ID <= 0 {
		t.Fatalf("ResolveContact returned no ID")
	}
}

func fetchRow(t *testing.T, db *sqlx.DB, id int) userRow {
	t.Helper()
	var r userRow
	err := db.QueryRow(`SELECT first_name, COALESCE(last_name, ''), COALESCE(email, ''), COALESCE(external_user_id, ''), type FROM users WHERE id = $1`,
		id).Scan(&r.FirstName, &r.LastName, &r.Email, &r.ExtID, &r.Type)
	if err != nil {
		t.Fatalf("fetching user %d: %v", id, err)
	}
	return r
}

func countByEmail(t *testing.T, db *sqlx.DB, email string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&n); err != nil {
		t.Fatalf("counting users with email %s: %v", email, err)
	}
	return n
}

func insertAgent(t *testing.T, db *sqlx.DB, email, extID string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO users (type, email, first_name, "password", external_user_id) VALUES ('agent', $1, 'Agent', 'x', $2) RETURNING id`,
		email, null.NewString(extID, extID != "")).Scan(&id)
	if err != nil {
		t.Fatalf("inserting agent: %v", err)
	}
	return id
}

func TestResolveContactReuse(t *testing.T) {
	u, db := newTestManager(t)

	// Creates a contact when none exists.
	alice := newContact("alice@example.com", "", "Alice", "Original")
	resolve(t, u, alice, models.ContactReuse)
	if r := fetchRow(t, db, alice.ID); r.Type != "contact" || r.FirstName != "Alice" {
		t.Fatalf("unexpected created row: %+v", r)
	}

	// A later reuse with a different name resolves to the same contact untouched.
	again := newContact("alice@example.com", "", "Changed", "Name")
	resolve(t, u, again, models.ContactReuse)
	if again.ID != alice.ID {
		t.Fatalf("expected reuse of contact %d, got %d", alice.ID, again.ID)
	}
	if r := fetchRow(t, db, alice.ID); r.FirstName != "Alice" || r.LastName != "Original" {
		t.Fatalf("reuse policy modified the contact: %+v", r)
	}

	// Email matching is case-insensitive.
	upper := newContact("ALICE@Example.COM", "", "X", "Y")
	resolve(t, u, upper, models.ContactReuse)
	if upper.ID != alice.ID {
		t.Fatalf("expected case-insensitive match of contact %d, got %d", alice.ID, upper.ID)
	}

	// Creates a contact with external ID when none matches.
	bob := newContact("bob@example.com", "ext-bob", "Bob", "")
	resolve(t, u, bob, models.ContactReuse)
	if r := fetchRow(t, db, bob.ID); r.ExtID != "ext-bob" {
		t.Fatalf("expected ext_id ext-bob, got %+v", r)
	}

	// External ID match wins over the passed email, and the stored email is returned for use as recipient.
	byExt := newContact("different@example.com", "ext-bob", "B", "")
	resolve(t, u, byExt, models.ContactReuse)
	if byExt.ID != bob.ID {
		t.Fatalf("expected ext_id match of contact %d, got %d", bob.ID, byExt.ID)
	}
	if byExt.Email.String != "bob@example.com" {
		t.Fatalf("expected stored email bob@example.com back, got %q", byExt.Email.String)
	}
	if r := fetchRow(t, db, bob.ID); r.Email != "bob@example.com" {
		t.Fatalf("reuse policy modified stored email: %+v", r)
	}

	// An unknown ext_id with an existing contact's email falls back to the email match instead of inserting a duplicate.
	staleExt := newContact("alice@example.com", "ext-unknown", "A", "")
	resolve(t, u, staleExt, models.ContactReuse)
	if staleExt.ID != alice.ID {
		t.Fatalf("expected email fallback to contact %d, got %d", alice.ID, staleExt.ID)
	}
	if n := countByEmail(t, db, "alice@example.com"); n != 1 {
		t.Fatalf("unknown ext_id created a duplicate contact, got %d rows", n)
	}

	// Plain email reuse also matches a contact that has an ext_id.
	byEmail := newContact("bob@example.com", "", "B", "")
	resolve(t, u, byEmail, models.ContactReuse)
	if byEmail.ID != bob.ID {
		t.Fatalf("expected email match of contact %d, got %d", bob.ID, byEmail.ID)
	}

	// An agent carrying the same external ID must never be resolved as the contact.
	agentID := insertAgent(t, db, "agent@example.com", "ext-agent")
	carol := newContact("carol@example.com", "ext-agent", "Carol", "")
	resolve(t, u, carol, models.ContactReuse)
	if carol.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact", agentID)
	}
	if r := fetchRow(t, db, carol.ID); r.Type != "contact" {
		t.Fatalf("expected new contact, got %+v", r)
	}

	// An agent's email must never be resolved as the contact either.
	agentEmail := newContact("agent@example.com", "", "NotAgent", "")
	resolve(t, u, agentEmail, models.ContactReuse)
	if agentEmail.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact by email", agentID)
	}

	// Soft-deleted contacts are not matched; a fresh contact is created.
	dave := newContact("dave@example.com", "", "Dave", "")
	resolve(t, u, dave, models.ContactReuse)
	db.MustExec(`UPDATE users SET deleted_at = now() WHERE id = $1`, dave.ID)
	dave2 := newContact("dave@example.com", "", "Dave", "")
	resolve(t, u, dave2, models.ContactReuse)
	if dave2.ID == dave.ID {
		t.Fatalf("resolved a soft-deleted contact %d", dave.ID)
	}
}

func TestResolveContactSync(t *testing.T) {
	u, db := newTestManager(t)

	// Creates a contact, normalizing the email to lowercase.
	eve := newContact("EVE@Example.com", "", "Eve", "One")
	resolve(t, u, eve, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.Email != "eve@example.com" {
		t.Fatalf("expected normalized email, got %+v", r)
	}

	// Sync by email updates the name on the matched contact.
	renamed := newContact("eve@example.com", "", "Eva", "Two")
	resolve(t, u, renamed, models.ContactSync)
	if renamed.ID != eve.ID {
		t.Fatalf("expected match of contact %d, got %d", eve.ID, renamed.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Eva" || r.LastName != "Two" {
		t.Fatalf("expected updated name, got %+v", r)
	}

	// Blank names do not clobber stored ones.
	blank := newContact("eve@example.com", "", "", "")
	resolve(t, u, blank, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Eva" || r.LastName != "Two" {
		t.Fatalf("blank names clobbered stored ones: %+v", r)
	}

	// Sync with an ext_id enriches the matched no-ext contact.
	enrich := newContact("eve@example.com", "ext-eve", "Eva", "")
	resolve(t, u, enrich, models.ContactSync)
	if enrich.ID != eve.ID {
		t.Fatalf("expected enrichment of contact %d, got %d", eve.ID, enrich.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.ExtID != "ext-eve" {
		t.Fatalf("expected enriched ext_id, got %+v", r)
	}

	// Sync by ext_id updates email and name.
	updated := newContact("eve2@example.com", "ext-eve", "Evelyn", "Three")
	resolve(t, u, updated, models.ContactSync)
	if updated.ID != eve.ID {
		t.Fatalf("expected ext_id match of contact %d, got %d", eve.ID, updated.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.Email != "eve2@example.com" || r.FirstName != "Evelyn" {
		t.Fatalf("expected updated email/name, got %+v", r)
	}

	// An empty email on an ext_id sync keeps the stored email.
	noEmail := newContact("", "ext-eve", "Evelyn", "")
	resolve(t, u, noEmail, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.Email != "eve2@example.com" {
		t.Fatalf("empty email clobbered stored one: %+v", r)
	}

	// Sync without ext_id on an email owned by an ext_id contact reuses it without updating the name.
	plain := newContact("eve2@example.com", "", "Someone", "Else")
	resolve(t, u, plain, models.ContactSync)
	if plain.ID != eve.ID {
		t.Fatalf("expected reuse of ext_id contact %d, got %d", eve.ID, plain.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Evelyn" {
		t.Fatalf("no-ext sync modified an ext_id contact: %+v", r)
	}

	// The same email with a different ext_id creates a second contact - one email can map to multiple external IDs.
	other := newContact("eve2@example.com", "ext-other", "Other", "")
	resolve(t, u, other, models.ContactSync)
	if other.ID == eve.ID {
		t.Fatalf("expected a second contact for a different ext_id, got the same one")
	}
	if n := countByEmail(t, db, "eve2@example.com"); n != 2 {
		t.Fatalf("expected 2 contacts with the shared email, got %d", n)
	}

	// An ext_id already owned by another contact wins over an email-matched contact.
	frank := newContact("frank@example.com", "", "Frank", "")
	resolve(t, u, frank, models.ContactSync)
	taken := newContact("frank@example.com", "ext-eve", "Franklin", "")
	resolve(t, u, taken, models.ContactSync)
	if taken.ID != eve.ID {
		t.Fatalf("expected ext_id owner %d to win, got %d", eve.ID, taken.ID)
	}
	if r := fetchRow(t, db, frank.ID); r.ExtID != "" || r.FirstName != "Frank" {
		t.Fatalf("email-matched contact was modified: %+v", r)
	}

	// An agent's email must never be matched.
	agentID := insertAgent(t, db, "agent2@example.com", "")
	notAgent := newContact("agent2@example.com", "", "NotAgent", "")
	resolve(t, u, notAgent, models.ContactSync)
	if notAgent.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact", agentID)
	}
	if r := fetchRow(t, db, agentID); r.FirstName != "Agent" {
		t.Fatalf("sync modified an agent row: %+v", r)
	}
}

func TestResolveContactConcurrent(t *testing.T) {
	u, db := newTestManager(t)

	race := func(policy models.ContactPolicy, email, extID string) {
		t.Helper()
		var (
			wg  sync.WaitGroup
			mu  sync.Mutex
			ids = map[int]struct{}{}
		)
		for range 10 {
			wg.Go(func() {
				c := newContact(email, extID, "Race", "")
				if err := u.ResolveContact(c, policy); err != nil {
					t.Errorf("ResolveContact: %v", err)
					return
				}
				mu.Lock()
				ids[c.ID] = struct{}{}
				mu.Unlock()
			})
		}
		wg.Wait()
		if len(ids) != 1 {
			t.Fatalf("concurrent resolves produced %d distinct contacts: %v", len(ids), ids)
		}
		if n := countByEmail(t, db, email); n != 1 {
			t.Fatalf("expected 1 contact with email %s, got %d", email, n)
		}
	}

	race(models.ContactReuse, "race-reuse@example.com", "")
	race(models.ContactReuse, "race-reuse-ext@example.com", "ext-race-reuse")
	race(models.ContactSync, "race-sync@example.com", "")
	race(models.ContactSync, "race-sync-ext@example.com", "ext-race-sync")
}
