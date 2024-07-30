package main

import (
	"fmt"

	"github.com/abhinavxd/artemis/internal/stringutil"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleOIDCLogin initializes an OIDC request and redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	state, _ := stringutil.RandomAlNumString(30)
	authURL := app.auth.LoginURL(state)
	fmt.Println("url ", authURL)
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

// handleOIDCCallback receives the redirect callback from the OIDC provider and completes the handshake.
func handleOIDCCallback(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		code  = string(r.RequestCtx.QueryArgs().Peek("code"))
		state = string(r.RequestCtx.QueryArgs().Peek("state"))
	)

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, code)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		return err
	}

	// Get user by e-mail received from OIDC.
	user, err := app.user.GetByEmail(claims.Email)
	if err != nil {
		return err
	}

	// Set the session.
	if err := app.auth.SaveSession(user, r); err != nil {
		return err
	}

	return r.Redirect(state, fasthttp.StatusFound, nil, "")
}
