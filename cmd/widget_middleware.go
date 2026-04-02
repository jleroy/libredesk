package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	// Context keys for storing authenticated widget data
	ctxWidgetClaims    = "widget_claims"
	ctxWidgetContactID = "widget_contact_id"
	ctxWidgetInbox     = "widget_inbox"
	ctxWidgetConfig    = "widget_config"

	hdrWidgetInboxID    = "X-Libredesk-Inbox-ID"
	hdrWidgetVisitorJWT = "X-Libredesk-Visitor-JWT"
	hdrClearVisitorJWT  = "X-Libredesk-Clear-Visitor"
)

// widgetAuth middleware authenticates widget requests using JWT and inbox validation.
// It always validates the inbox from X-Libredesk-Inbox-ID header, and conditionally validates JWT.
// For /conversations/init without JWT, it allows visitor creation while still validating inbox.
func widgetAuth(next func(*fastglue.Request) error) func(*fastglue.Request) error {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Always extract and validate inbox UUID from custom header.
		inboxUUID := string(r.RequestCtx.Request.Header.Peek(hdrWidgetInboxID))
		if inboxUUID == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Fetch and validate inbox by UUID.
		inbox, err := app.inbox.GetDBRecord(inboxUUID)
		if err != nil {
			app.lo.Error("error fetching inbox", "inbox_uuid", inboxUUID, "error", err)
			return sendErrorEnvelope(r, err)
		}

		if !inbox.Enabled {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("status.disabledInbox"), nil, envelope.InputError)
		}

		// Check if inbox is the correct type for widget requests
		if inbox.Channel != livechat.ChannelLiveChat {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
		}

		// Check if the client's IP is blocked.
		var config livechat.Config
		if err := json.Unmarshal(inbox.Config, &config); err != nil {
			app.lo.Error("error parsing live chat config", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
		}
		if len(config.BlockedIPs) > 0 {
			clientIP := realip.FromRequest(r.RequestCtx)
			if httputil.IsIPBlocked(clientIP, config.BlockedIPs) {
				return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("widget.ipBlocked"), nil, envelope.PermissionError)
			}
		}

		// Store inbox and parsed config in context for downstream handlers.
		r.RequestCtx.SetUserValue(ctxWidgetInbox, inbox)
		r.RequestCtx.SetUserValue(ctxWidgetConfig, config)

		// Extract JWT from Authorization header (Bearer token)
		authHeader := string(r.RequestCtx.Request.Header.Peek("Authorization"))

		// For init endpoint, allow requests without JWT (visitor creation)
		if authHeader == "" && strings.HasSuffix(string(r.RequestCtx.Path()), "/conversations/init") {
			return next(r)
		}

		// For all other requests, require JWT
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}
		jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify JWT using inbox secret
		claims, err := verifyStandardJWT(jwtToken, inbox.Secret.String)
		if err != nil {
			app.lo.Error("invalid widget JWT", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}

		// Validate required fields for external user JWT.
		if claims.ExternalUserID != "" {
			if claims.Email == "" {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "email"), nil, envelope.InputError)
			}
			if claims.FirstName == "" {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "first_name"), nil, envelope.InputError)
			}
		}

		// Resolve or create contact from JWT claims.
		contactID, err := resolveOrCreateExternalContact(app, claims)
		if err != nil {
			app.lo.Error("error resolving or creating contact", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
		}

		// Store authenticated data in request context for downstream handlers
		r.RequestCtx.SetUserValue(ctxWidgetClaims, claims)
		r.RequestCtx.SetUserValue(ctxWidgetContactID, contactID)

		// Merge visitor to contact if visitor JWT is provided.
		visitorJWT := string(r.RequestCtx.Request.Header.Peek(hdrWidgetVisitorJWT))
		if visitorJWT != "" && claims.ExternalUserID != "" && contactID > 0 {
			visitorClaims, err := verifyStandardJWT(visitorJWT, inbox.Secret.String)
			if err == nil && visitorClaims.IsVisitor && visitorClaims.UserID > 0 {
				visitorID := visitorClaims.UserID
				if visitorID != contactID {
					if err := app.user.MergeVisitorToContact(visitorID, contactID); err != nil {
						app.lo.Error("error merging visitor to contact", "visitor_id", visitorID, "contact_id", contactID, "error", err)
					} else {
						app.lo.Info("merged visitor to contact", "visitor_id", visitorID, "contact_id", contactID)
						r.RequestCtx.Response.Header.Set(hdrClearVisitorJWT, "true")
					}
				}
			}
		}

		return next(r)
	}
}

// Helper functions to extract authenticated data from request context

// getWidgetContactID extracts contact ID from request context
func getWidgetContactID(r *fastglue.Request) (int, error) {
	val := r.RequestCtx.UserValue(ctxWidgetContactID)
	if val == nil {
		return 0, fmt.Errorf("widget middleware not applied: missing contact ID in context")
	}
	contactID, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("invalid contact ID type in context")
	}
	return contactID, nil
}

// getWidgetInbox extracts inbox model from request context
func getWidgetInbox(r *fastglue.Request) (imodels.Inbox, error) {
	val := r.RequestCtx.UserValue(ctxWidgetInbox)
	if val == nil {
		return imodels.Inbox{}, fmt.Errorf("widget middleware not applied: missing inbox in context")
	}
	inbox, ok := val.(imodels.Inbox)
	if !ok {
		return imodels.Inbox{}, fmt.Errorf("invalid inbox type in context")
	}
	return inbox, nil
}

// getWidgetConfig extracts parsed livechat config from request context
func getWidgetConfig(r *fastglue.Request) (livechat.Config, error) {
	val := r.RequestCtx.UserValue(ctxWidgetConfig)
	if val == nil {
		return livechat.Config{}, fmt.Errorf("widget middleware not applied: missing config in context")
	}
	config, ok := val.(livechat.Config)
	if !ok {
		return livechat.Config{}, fmt.Errorf("invalid config type in context")
	}
	return config, nil
}

// getWidgetClaimsOptional extracts JWT claims from request context, returns nil if not set
func getWidgetClaimsOptional(r *fastglue.Request) *Claims {
	val := r.RequestCtx.UserValue(ctxWidgetClaims)
	if val == nil {
		return nil
	}
	if claims, ok := val.(Claims); ok {
		return &claims
	}
	return nil
}

