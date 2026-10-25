package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/locales"
	"github.com/knadh/stuffbin"
	"github.com/leonelquinteros/gotext"
	"github.com/zerodha/fastglue"
	"golang.org/x/text/language"
)

const (
	defLang = "en-US"
)

var (
	localeI18nMu    sync.Mutex
	localeI18nCache = map[string]*gotext.Locale{}

	// Keyed by resolved language code
	i18nJSONMu    sync.Mutex
	i18nJSONCache = map[string][]byte{}
)

// handleGetI18nLang returns the JSON language pack for the given language code.
func handleGetI18nLang(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		lang = r.RequestCtx.UserValue("lang").(string)
	)
	b, err := i18nLangJSON(app, lang)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.SetContentType("application/json")
	r.RequestCtx.Response.SetStatusCode(http.StatusOK)
	r.RequestCtx.Response.SetBodyRaw(b)
	return nil
}

// handleGetAvailableLanguages returns the list of available languages
// by reading all JSON files from the /i18n/ directory in the embedded filesystem.
func handleGetAvailableLanguages(r *fastglue.Request) error {
	app := r.Context.(*App)

	files, err := app.fs.Glob("/i18n/*.json")
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "error listing language files", nil))
	}

	type langInfo struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	var langs []langInfo
	for _, f := range files {
		code := strings.TrimSuffix(filepath.Base(f), ".json")
		b, err := app.fs.Read(f)
		if err != nil {
			continue
		}

		var meta map[string]string
		if err := json.Unmarshal(b, &meta); err != nil {
			continue
		}

		name := meta["_.name"]
		if name == "" {
			name = code
		}
		langs = append(langs, langInfo{Code: code, Name: name})
	}

	sort.Slice(langs, func(i, j int) bool {
		return langs[i].Name < langs[j].Name
	})

	return r.SendEnvelope(langs)
}

// localeI18n returns an i18n instance for the given locale code, matching available
// language packs by exact code or language prefix, falling back to the app's i18n.
func localeI18n(app *App, locale string) *gotext.Locale {
	loc := strings.ToLower(strings.TrimSpace(locale))
	localeI18nMu.Lock()
	defer localeI18nMu.Unlock()
	if i, ok := localeI18nCache[loc]; ok {
		return i
	}
	i, err := locales.New(loc)
	if err != nil || i == nil {
		i = app.i18n
	}
	localeI18nCache[loc] = i
	return i
}

// i18nLangJSON returns the marshaled language pack for a language code, cached per resolved code.
func i18nLangJSON(app *App, lang string) ([]byte, error) {
	code := matchLangFile(app.fs, strings.ToLower(strings.TrimSpace(lang)))
	i18nJSONMu.Lock()
	defer i18nJSONMu.Unlock()
	if b, ok := i18nJSONCache[code]; ok {
		return b, nil
	}
	// These catalogs are exclusively for Vue. Keep their English-overlay behavior
	// independent of the backend's Gettext catalogs.
	b, err := app.fs.Read("/i18n/" + defLang + ".json")
	if err != nil {
		return nil, err
	}
	var messages map[string]string
	if err := json.Unmarshal(b, &messages); err != nil {
		return nil, err
	}
	if selected, err := app.fs.Read("/i18n/" + code + ".json"); err == nil {
		if err := json.Unmarshal(selected, &messages); err != nil {
			return nil, err
		}
	}
	b, err = json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	i18nJSONCache[code] = b
	return b, nil
}

func matchLangFile(fs stuffbin.FileSystem, loc string) string {
	wanted, err := language.All.Parse(strings.TrimSpace(loc))
	if err != nil || wanted.IsRoot() {
		return defLang
	}
	files, err := fs.Glob("/i18n/*.json")
	if err != nil {
		return defLang
	}
	wantedBase, _ := wanted.Base()
	wantedScript, _ := wanted.Script()
	prefixMatch := ""
	for _, f := range files {
		code := strings.TrimSuffix(filepath.Base(f), ".json")
		available, err := language.All.Parse(code)
		if err != nil {
			continue
		}
		if available.String() == wanted.String() {
			return code
		}
		base, _ := available.Base()
		script, _ := available.Script()
		if prefixMatch == "" && base == wantedBase && script == wantedScript {
			prefixMatch = code
		}
	}
	if prefixMatch != "" {
		return prefixMatch
	}
	return defLang
}
