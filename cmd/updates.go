// Copyright Kailash Nadh (https://github.com/knadh/listmonk)
// SPDX-License-Identifier: AGPL-3.0
// Adapted from listmonk for Libredesk.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/mod/semver"
)

const updateCheckURL = "https://updates.libredesk.io/updates.json"

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

// checkUpdates is a blocking function that checks for updates to the app
// at the given intervals. On detecting a new update (new semver), it
// sets the global update status that renders a prompt on the UI.
func checkUpdates(curVersion string, interval time.Duration, app *App) {
	client := &http.Client{Timeout: updateCheckTimeout}

	fnCheck := func() {
		out, err := fetchAppUpdate(client, updateCheckURL)
		if err != nil {
			app.lo.Error("error checking for app updates", "err", err)
			return
		}

		// There is an update. Set it on the global app state.
		// semver ranks v2.9.0-rc.10 below v2.9.0, so RC builds are offered the stable release.
		if semver.IsValid(out.Update.ReleaseVersion) {
			if semver.Compare(out.Update.ReleaseVersion, curVersion) > 0 {
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

func fetchAppUpdate(client *http.Client, url string) (*AppUpdate, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var out AppUpdate
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
