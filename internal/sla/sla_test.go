package sla

import (
	"context"
	"database/sql"
	"testing"
	"time"

	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/sla/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

type stubTeamStore struct{}

type stubUserStore struct{}

type stubAppSettingsStore struct{}

type stubBusinessHrsStore struct{}

type appliedRow struct {
	ID          int          `db:"id"`
	Status      string       `db:"status"`
	FRDeadline  sql.NullTime `db:"first_response_deadline_at"`
	ResDeadline sql.NullTime `db:"resolution_deadline_at"`
	FRMetAt     sql.NullTime `db:"first_response_met_at"`
	FRBreached  sql.NullTime `db:"first_response_breached_at"`
	ResMetAt    sql.NullTime `db:"resolution_met_at"`
	ResBreached sql.NullTime `db:"resolution_breached_at"`
}

func (stubTeamStore) Get(id int) (tmodels.Team, error) { return tmodels.Team{}, nil }

func (stubUserStore) GetAgent(int, string) (umodels.User, error) { return umodels.User{}, nil }

func (stubAppSettingsStore) GetByPrefix(prefix string) (types.JSONText, error) {
	return types.JSONText(`{"app.business_hours_id":"1","app.timezone":"UTC"}`), nil
}

func (stubBusinessHrsStore) Get(id int) (bmodels.BusinessHours, error) {
	return bmodels.BusinessHours{ID: 1, IsAlwaysOpen: true}, nil
}

func TestApplySLASetsDeadlinesAndConversation(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, policy)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 1 || rows[0].Status != "pending" {
		t.Fatalf("expected one pending applied sla, got %+v", rows)
	}
	if !rows[0].FRDeadline.Valid || !rows[0].ResDeadline.Valid {
		t.Fatalf("expected both deadlines set, got %+v", rows[0])
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || !d.Time.Equal(rows[0].FRDeadline.Time) {
		t.Fatalf("expected conversation deadline = first response deadline, got %v vs %v", d, rows[0].FRDeadline)
	}
}

func TestApplySLAReplacesUntouchedPending(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "2h", "")
	p2 := insertPolicy(t, db, "p2", "3h", "4h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 1 || rows[0].Status != "pending" {
		t.Fatalf("expected old untouched sla deleted and one pending row, got %+v", rows)
	}
}

func TestApplySLAClosesSettledPendingAndCleansChildren(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	p2 := insertPolicy(t, db, "p2", "3h", "4h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	old := fetchApplied(t, db, conv)[0]
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW() WHERE id = $1`, old.ID)
	db.MustExec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at, status) VALUES ($1, $2, 'next_response', NOW() + INTERVAL '30 min', 'pending')`, old.ID, p1)
	db.MustExec(`INSERT INTO scheduled_sla_notifications (applied_sla_id, metric, notification_type, recipients, send_at) VALUES ($1, 'resolution', 'breach', '{1}', NOW() + INTERVAL '2h')`, old.ID)

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 {
		t.Fatalf("expected settled sla kept plus new pending, got %+v", rows)
	}
	if rows[0].ID != old.ID || rows[0].Status == "pending" {
		t.Fatalf("expected old sla closed, got %+v", rows[0])
	}
	if rows[1].Status != "pending" {
		t.Fatalf("expected new pending sla, got %+v", rows[1])
	}
	var events, notifs int
	db.QueryRow(`SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND status = 'pending'`, old.ID).Scan(&events)
	db.QueryRow(`SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND processed_at IS NULL`, old.ID).Scan(&notifs)
	if events != 0 || notifs != 0 {
		t.Fatalf("expected superseded sla children cleaned, got %d events and %d notifications", events, notifs)
	}
}

func TestEvaluateMetAndClose(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW(), resolved_at = NOW() WHERE id = $1`, conv)
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "met" || !row.FRMetAt.Valid || !row.ResMetAt.Valid {
		t.Fatalf("expected sla met and closed, got %+v", row)
	}
}

func TestEvaluateBreachAndClose(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '2h', resolution_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "breached" || !row.FRBreached.Valid || !row.ResBreached.Valid {
		t.Fatalf("expected sla breached and closed, got %+v", row)
	}
}

func TestEvaluatePartiallyMet(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET resolution_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "partially_met" || !row.FRMetAt.Valid || !row.ResBreached.Valid {
		t.Fatalf("expected partially met, got %+v", row)
	}
}

func TestEvaluateLeavesUnsettledPending(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" || !row.FRMetAt.Valid || row.ResMetAt.Valid {
		t.Fatalf("expected first response met but sla still pending, got %+v", row)
	}
}

func TestEvaluateSingleMetricPolicy(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "fr-only", "1h", "", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "met" || !row.FRMetAt.Valid {
		t.Fatalf("expected first-response-only sla closed as met, got %+v", row)
	}
}

func TestSweepSkipsNextResponseOnlyPolicy(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "nr-only", "", "", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	deadline, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0)
	if err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" {
		t.Fatalf("expected next-response-only sla to stay pending, got %+v", row)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(deadline).Abs() > time.Millisecond {
		t.Fatalf("expected next response deadline kept on conversation, got %v want %v", d, deadline)
	}
}

func TestBreachSchedulesNotification(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"breach","time_delay_type":"immediately","time_delay":"","recipients":["assigned_user"]}]' WHERE id = $1`, policy)
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND notification_type = 'breach'`, fetchApplied(t, db, conv)[0].ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected one scheduled breach notification, got %d", n)
	}
}

func TestNextResponseEventLifecycle(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != ErrUnmetSLAEventAlreadyExists {
		t.Fatalf("expected duplicate unmet event to be rejected, got %v", err)
	}
	if _, err := m.SetLatestSLAEventMetAt(appliedID, MetricNextResponse); err != nil {
		t.Fatalf("SetLatestSLAEventMetAt: %v", err)
	}
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}

	var status string
	db.QueryRow(`SELECT status FROM sla_events WHERE applied_sla_id = $1`, appliedID).Scan(&status)
	if status != "met" {
		t.Fatalf("expected event met, got %s", status)
	}
	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("expected new event allowed after previous met, got %v", err)
	}
}

func TestNextResponseEventBreachSchedulesNotification(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"breach","time_delay_type":"immediately","time_delay":"","recipients":["assigned_user"]}]' WHERE id = $1`, policy)
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	db.MustExec(`UPDATE sla_events SET deadline_at = NOW() - INTERVAL '1h' WHERE applied_sla_id = $1`, appliedID)
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}

	var status string
	db.QueryRow(`SELECT status FROM sla_events WHERE applied_sla_id = $1`, appliedID).Scan(&status)
	if status != "breached" {
		t.Fatalf("expected event breached, got %s", status)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND metric = 'next_response' AND notification_type = 'breach'`, appliedID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected one scheduled next-response breach notification, got %d", n)
	}
}

func TestSendNotificationSkipsMetMetric(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW() WHERE id = $1`, appliedID)
	db.MustExec(`INSERT INTO scheduled_sla_notifications (applied_sla_id, metric, notification_type, recipients, send_at) VALUES ($1, 'first_response', 'breach', '{1}', NOW() - INTERVAL '1 min')`, appliedID)

	var pending []models.ScheduledSLANotification
	if err := m.q.GetScheduledSLANotifications.Select(&pending); err != nil {
		t.Fatalf("fetching scheduled notifications: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected one due notification, got %d", len(pending))
	}
	if err := m.SendNotification(pending[0]); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	var processed bool
	db.QueryRow(`SELECT processed_at IS NOT NULL FROM scheduled_sla_notifications WHERE id = $1`, pending[0].ID).Scan(&processed)
	if !processed {
		t.Fatal("expected met-metric notification marked processed without sending")
	}
}

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "sla")
	lo := logf.New(logf.Opts{})
	mgr, err := New(
		Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)},
		stubTeamStore{},
		stubAppSettingsStore{},
		stubBusinessHrsStore{},
		nil,
		stubUserStore{},
		nil,
	)
	if err != nil {
		t.Fatalf("creating sla manager: %v", err)
	}
	return mgr, db
}

func insertPolicy(t *testing.T, db *sqlx.DB, name, fr, res, nr string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time) VALUES ($1, '', $2, $3, $4) RETURNING id`,
		name, fr, res, nr).Scan(&id)
	if err != nil {
		t.Fatalf("inserting sla policy: %v", err)
	}
	return id
}

func insertConversation(t *testing.T, db *sqlx.DB, ref string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`
		WITH contact AS (
			INSERT INTO users (type, email, first_name) VALUES ('contact', $1 || '@example.com', 'C') RETURNING id
		), inbox AS (
			INSERT INTO inboxes (channel, config, name) VALUES ('email', '{}', 'inb-' || $1) RETURNING id
		)
		INSERT INTO conversations (contact_id, inbox_id, status_id, reference_number, subject)
		SELECT contact.id, inbox.id, (SELECT id FROM conversation_statuses WHERE category != 'resolved' LIMIT 1), $1, 'subject'
		FROM contact, inbox RETURNING id`, ref).Scan(&id)
	if err != nil {
		t.Fatalf("inserting conversation: %v", err)
	}
	return id
}

func fetchApplied(t *testing.T, db *sqlx.DB, conversationID int) []appliedRow {
	t.Helper()
	var rows []appliedRow
	err := db.Select(&rows, `SELECT id, status, first_response_deadline_at, resolution_deadline_at,
		first_response_met_at, first_response_breached_at, resolution_met_at, resolution_breached_at
		FROM applied_slas WHERE conversation_id = $1 ORDER BY id`, conversationID)
	if err != nil {
		t.Fatalf("fetching applied slas: %v", err)
	}
	return rows
}

func conversationDeadline(t *testing.T, db *sqlx.DB, conversationID int) sql.NullTime {
	t.Helper()
	var d sql.NullTime
	if err := db.QueryRow(`SELECT next_sla_deadline_at FROM conversations WHERE id = $1`, conversationID).Scan(&d); err != nil {
		t.Fatalf("fetching conversation deadline: %v", err)
	}
	return d
}

func applySLA(t *testing.T, m *Manager, conversationID, policyID int) {
	t.Helper()
	if _, err := m.ApplySLA(time.Now(), conversationID, 0, policyID); err != nil {
		t.Fatalf("ApplySLA: %v", err)
	}
}
