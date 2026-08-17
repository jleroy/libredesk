// Copyright Kailash Nadh (https://github.com/knadh/listmonk)
// SPDX-License-Identifier: AGPL-3.0
// Adapted from listmonk for Libredesk.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/mod/semver"
)

const updateCheckURL = "https://updates.libredesk.io/updates.json"

// updateCheckTimeout bounds a single update check so a hung or blackholing
// endpoint cannot block the checker loop forever (see #445).
const updateCheckTimeout = 10 * time.Second

type AppUpdate struct {
	Update struct {
		ReleaseVersion string `json:"release_version"`
		ReleaseDate    string `json:"release_date"`
		URL            string `json:"url"`
		Description    string `json:"description"`

		// This is computed and set locally based on the local version.
		IsNew bool `json:"is_new"`
	} `json:"update"`
	Messages []struct {
		Date        string `json:"date"`
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Priority    string `json:"priority"`
	} `json:"messages"`
}

var reSemver = regexp.MustCompile(`-(.*)`)

// checkUpdates is a blocking function that checks for updates to the app
// at the given intervals. On detecting a new update (new semver), it
// sets the global update status that renders a prompt on the UI.
func checkUpdates(curVersion string, interval time.Duration, app *App) {
	// Strip -* suffix.
	curVersion = reSemver.ReplaceAllString(curVersion, "")

	fnCheck := func() {
		out, err := fetchAppUpdate(&http.Client{Timeout: updateCheckTimeout}, updateCheckURL)
		if err != nil {
			app.lo.Error("error checking for app updates", "err", err)
			return
		}

		// There is an update. Set it on the global app state.
		if semver.IsValid(out.Update.ReleaseVersion) {
			v := reSemver.ReplaceAllString(out.Update.ReleaseVersion, "")
			if semver.Compare(v, curVersion) > 0 {
				out.Update.IsNew = true
				app.lo.Info("new update available", "version", out.Update.ReleaseVersion)
			}
		}

		app.Lock()
		app.update = out
		app.Unlock()
	}

	// Give a 5 minute buffer after app start in case the admin wants to disable
	// update checks entirely and not make a request to upstream.
	time.Sleep(time.Minute * 5)
	fnCheck()

	// Thereafter, check every $interval.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		fnCheck()
	}
}

// fetchAppUpdate fetches and parses the update manifest using the given client.
// The response body is always closed, including on the non-200 and read-error
// paths that previously leaked it (see #445). The caller supplies the client so
// a timeout can be enforced (and so this can be tested against a stub server).
func fetchAppUpdate(client *http.Client, url string) (*AppUpdate, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status code checking for app updates: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var out AppUpdate
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return &out, nil
}
