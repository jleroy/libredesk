package main

import (
	"bytes"
	"encoding/json"
	"html"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/importer"
	"github.com/abhinavxd/libredesk/internal/locales"
	"github.com/abhinavxd/libredesk/internal/setting"
	smodels "github.com/abhinavxd/libredesk/internal/setting/models"
	tmpl "github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestLanguageReloadReachesExistingManagers(t *testing.T) {
	db := testutil.NewDB(t, "i18n_settings")
	// The handler reloads global configuration. Keep this test nonparallel and
	// restore the previous instance so other cmd tests cannot inherit settings.
	previousConfig := ko
	ko = koanf.New(".")
	t.Cleanup(func() { ko = previousConfig })
	if err := ko.Set("upload.provider", "fs"); err != nil {
		t.Fatal(err)
	}
	logger := logf.New(logf.Opts{Writer: io.Discard})
	settings, err := setting.New(setting.Opts{DB: db, Lo: &logger})
	if err != nil {
		t.Fatal(err)
	}
	general := smodels.General{Lang: "fr-FR", RootURL: "https://example.com", SiteName: "Test"}
	if err := settings.Update(general); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", root+"/static:static")
	if err != nil {
		t.Fatal(err)
	}
	locale, err := locales.New("fr")
	if err != nil {
		t.Fatal(err)
	}
	app := &App{setting: settings, fs: fs, i18n: locale, lo: &logger, tmpl: &tmpl.Manager{}}
	if err := reloadSettings(app); err != nil {
		t.Fatal(err)
	}
	manager := importer.New(importer.Opts{I18n: app.i18n, Lo: &logger})
	t.Cleanup(manager.Close)
	_, before := manager.GetStatus("missing")
	if before == nil || before.Error() != "Importation introuvable" {
		t.Fatalf("expected a French error before reload, got %v", before)
	}
	for _, tc := range []struct{ language, message string }{
		{"en-US", "Import not found"},
		{"fr-FR", "Importation introuvable"},
	} {
		general.Lang = tc.language
		body, err := json.Marshal(general)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(fasthttp.MethodPut)
		ctx.Request.SetRequestURI("/api/v1/settings/general")
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBody(body)
		if err := handleUpdateGeneralSettings(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != fasthttp.StatusOK {
			t.Fatalf("updating language to %s: HTTP %d: %s", tc.language, ctx.Response.StatusCode(), ctx.Response.Body())
		}
		stored, err := settings.GetAll()
		if err != nil {
			t.Fatal(err)
		}
		if stored.Lang != tc.language || ko.String("app.lang") != tc.language {
			t.Fatalf("language was not persisted and reloaded: stored=%q runtime=%q", stored.Lang, ko.String("app.lang"))
		}
		_, after := manager.GetStatus("missing")
		if after == nil || after.Error() != tc.message {
			t.Fatalf("%s: existing manager returned %v, want %q", tc.language, after, tc.message)
		}
	}
}

// The API overlays the selected frontend JSON catalog on English defaults.
// This checks the merge contract, not a historical snapshot of translations.
func TestFrontendCatalogMergesEnglishFallback(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", root+"/i18n:i18n")
	if err != nil {
		t.Fatal(err)
	}
	app := &App{fs: fs}
	for _, code := range []string{"en-US", "fr-FR", "ja-JP", "pt-PT"} {
		want := map[string]string{}
		for _, lang := range []string{"en-US", code} {
			b, err := fs.Read("/i18n/" + lang + ".json")
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &want); err != nil {
				t.Fatal(err)
			}
		}
		b, err := i18nLangJSON(app, code)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s: merged catalog has %d entries, want %d", code, len(got), len(want))
		}
		for key, value := range want {
			if actual, ok := got[key]; !ok {
				t.Errorf("%s: merged catalog is missing key %q", code, key)
			} else if actual != value {
				t.Errorf("%s: merged key %q = %q, want %q", code, key, actual, value)
			}
		}
	}
}

func TestGettextWebTemplates(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", root+"/static:static")
	if err != nil {
		t.Fatal(err)
	}
	locale, err := locales.New("fr")
	if err != nil {
		t.Fatal(err)
	}
	funcs := getTmplFuncs(&constants{}, locale, fs)
	if _, err := parseWebTemplates(funcs, fs); err != nil {
		t.Fatal(err)
	}
	// Exercise HTML/JSON escaping and server-side plural selection used by typeahead.
	tpl, err := template.New("search-labels").Funcs(funcs).Parse(`<div data-labels="{{ HelpSearchLabels L }}">{{ L.GetD "templates" "Rate your recent interaction" }}</div>`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 résultat") || !strings.Contains(out.String(), "2 résultats") || strings.Contains(out.String(), "Rate your recent interaction") {
		t.Fatalf("unexpected rendered translations: %s", out.String())
	}
	// The browser parses this attribute as JSON, not Go's default slice format.
	start := strings.Index(out.String(), `data-labels="`) + len(`data-labels="`)
	end := strings.Index(out.String()[start:], `"`) + start
	var labels []string
	if err := json.Unmarshal([]byte(html.UnescapeString(out.String()[start:end])), &labels); err != nil {
		t.Fatalf("invalid search-label JSON: %v (%s)", err, out.String())
	}
	if len(labels) != publicSearchLimit+1 || labels[0] != "0 résultat" || labels[1] != "1 résultat" || labels[2] != "2 résultats" {
		t.Fatalf("incorrect search labels: %v", labels)
	}
}
