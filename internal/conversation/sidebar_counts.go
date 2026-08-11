package conversation

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
)

const sidebarCountsMaxViews = 50

// conversationsCountBaseQuery counts conversations matching list-type/view filters.
// Unlike the standard inbox badges, view counts intentionally do NOT force
// category='open' — the badge must match whatever the view's own filters return.
const conversationsCountBaseQuery = `
SELECT COUNT(*)
FROM conversations
JOIN users ON contact_id = users.id
JOIN inboxes ON inbox_id = inboxes.id
LEFT JOIN conversation_statuses ON status_id = conversation_statuses.id
WHERE TRUE
%s`

// ListsForUserPermissions returns conversation list types the user may access.
func ListsForUserPermissions(permissions []string) []string {
	lists := []string{}
	hasTeamAll := slices.Contains(permissions, authzModels.PermConversationsReadTeamAll)

	for _, perm := range permissions {
		if perm == authzModels.PermConversationsReadAll {
			return []string{models.AllConversations}
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, models.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, models.AssignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamInbox && !hasTeamAll {
			lists = append(lists, models.TeamUnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamAll {
			lists = append(lists, models.TeamAllConversations)
		}
	}
	return lists
}

// UserCanAccessView reports whether a user can access a view.
func UserCanAccessView(view vmodels.View, userID int, teamIDs []int) bool {
	switch view.Visibility {
	case vmodels.VisibilityUser:
		return view.UserID != nil && *view.UserID == userID
	case vmodels.VisibilityAll:
		return true
	case vmodels.VisibilityTeam:
		if view.TeamID == nil {
			return false
		}
		return slices.Contains(teamIDs, *view.TeamID)
	default:
		return false
	}
}

// GetSidebarCounts returns sidebar badge counts for standard inboxes (open only)
// and for each accessible view (matching the view filters exactly).
func (c *Manager) GetSidebarCounts(viewingUserID int, permissions []string, teamIDs []int, views []vmodels.View) (models.SidebarCounts, error) {
	out := models.SidebarCounts{Views: map[string]int{}}

	standard, err := c.getStandardSidebarCounts(viewingUserID, permissions)
	if err != nil {
		return out, err
	}
	out.Assigned = standard.Assigned
	out.Mentioned = standard.Mentioned
	out.Unassigned = standard.Unassigned
	out.All = standard.All

	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return out, nil
	}

	accessible := make([]vmodels.View, 0, len(views))
	for _, view := range views {
		if UserCanAccessView(view, viewingUserID, teamIDs) {
			accessible = append(accessible, view)
		}
	}
	if len(accessible) > sidebarCountsMaxViews {
		accessible = accessible[:sidebarCountsMaxViews]
	}

	viewCounts, err := c.getViewOpenCounts(viewingUserID, teamIDs, lists, accessible)
	if err != nil {
		return out, err
	}
	out.Views = viewCounts

	return out, nil
}

func (c *Manager) getStandardSidebarCounts(userID int, permissions []string) (models.SidebarCounts, error) {
	var row struct {
		Assigned   int `db:"assigned"`
		Unassigned int `db:"unassigned"`
		Mentioned  int `db:"mentioned"`
		All        int `db:"all"`
	}

	if err := c.q.GetSidebarStandardCounts.Get(&row, userID); err != nil {
		c.lo.Error("error fetching sidebar standard counts", "error", err)
		return models.SidebarCounts{}, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	out := models.SidebarCounts{}
	if slices.Contains(permissions, authzModels.PermConversationsReadAssigned) {
		out.Assigned = row.Assigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadUnassigned) {
		out.Unassigned = row.Unassigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsRead) {
		out.Mentioned = row.Mentioned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadAll) {
		out.All = row.All
	}
	return out, nil
}

// getViewOpenCounts returns the conversation count per view, keyed by view ID, using a
// single round trip so the number of views does not drive the number of queries.
// Counts follow each view's filters (no forced open category).
func (c *Manager) getViewOpenCounts(userID int, teamIDs []int, listTypes []string, views []vmodels.View) (map[string]int, error) {
	counts := map[string]int{}
	if len(views) == 0 {
		return counts, nil
	}

	query, qArgs, err := c.makeViewCountsQuery(userID, teamIDs, listTypes, views)
	if err != nil {
		c.lo.Error("error making view counts query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var rows []struct {
		ViewID    int `db:"view_id"`
		OpenCount int `db:"open_count"`
	}
	if err := c.db.SelectContext(context.Background(), &rows, query, qArgs...); err != nil {
		c.lo.Error("error fetching view open counts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	for _, row := range rows {
		counts[strconv.Itoa(row.ViewID)] = row.OpenCount
	}
	return counts, nil
}

// makeViewCountsQuery unions one counting subquery per view into a single statement
// returning (view_id, open_count) rows.
func (c *Manager) makeViewCountsQuery(userID int, teamIDs []int, listTypes []string, views []vmodels.View) (string, []any, error) {
	var (
		args  = []any{}
		parts = make([]string, 0, len(views))
	)

	for _, view := range views {
		countQuery, nextArgs, err := c.makeConversationsCountQuery(args, userID, teamIDs, listTypes, string(view.Filters))
		if err != nil {
			return "", nil, err
		}
		// view.ID comes from the database, so it is safe to inline as a literal.
		parts = append(parts, fmt.Sprintf("SELECT %d AS view_id, (%s) AS open_count", view.ID, countQuery))
		args = nextArgs
	}

	return strings.Join(parts, " UNION ALL "), args, nil
}

// makeConversationsCountQuery prepares a scalar query counting conversations that match
// the given list types and filters. Bind parameters continue from existingArgs so several
// count queries can share one statement.
func (c *Manager) makeConversationsCountQuery(existingArgs []any, userID int, teamIDs []int, listTypes []string, filtersJSON string) (string, []any, error) {
	if filtersJSON == "" {
		filtersJSON = "[]"
	}

	if len(listTypes) == 0 {
		return "", nil, fmt.Errorf("no conversation list types specified")
	}

	qArgs := existingArgs
	conditions, err := appendListTypeConditions(listTypes, userID, userID, teamIDs, &qArgs)
	if err != nil {
		return "", nil, err
	}

	baseQuery := fmt.Sprintf(conversationsCountBaseQuery, listTypeWhereClause(conditions))

	return dbutil.BuildFilterQuery(baseQuery, qArgs, filtersJSON, conversationListAllowedFields, conversationFilterRenderers, c.filterLocation())
}
