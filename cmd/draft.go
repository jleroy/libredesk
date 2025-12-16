package main

import (
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type draftReq struct {
	Content string `json:"content"`
}

// handleUpsertConversationDraft saves or updates a draft for a conversation.
func handleUpsertConversationDraft(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		req   = draftReq{}
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding draft request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Validate content is not empty
	if strings.TrimSpace(req.Content) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "content"), nil, envelope.InputError)
	}

	draft, err := app.conversation.UpsertConversationDraft(conv.ID, user.ID, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(draft)
}

// handleGetConversationDraft retrieves a draft for a conversation.
func handleGetConversationDraft(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	draft, err := app.conversation.GetConversationDraft(conv.ID, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(draft)
}

// handleDeleteConversationDraft deletes a draft for a conversation.
func handleDeleteConversationDraft(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.DeleteConversationDraft(conv.ID, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
