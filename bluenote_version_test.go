package writefreely

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/writefreely/writefreely/config"
	"github.com/writefreely/writefreely/internal/version"
	"github.com/writefreely/writefreely/page"
)

func TestBlueNoteVersionIdentity(t *testing.T) {
	old := version.Revision
	version.Revision = "abc1234"
	t.Cleanup(func() { version.Revision = old })
	if got := FormatVersion(); got != "BlueNote 0.1.0 (based on WriteFreely 0.17.2; commit abc1234)" {
		t.Fatal(got)
	}
	if got := ServerUserAgent("https://example.com"); got != "Go (WriteFreely/0.17.2+bluenote.0.1.0; +https://example.com)" {
		t.Fatal(got)
	}
	if got := ServerUserAgent(""); got != "Go (WriteFreely/0.17.2+bluenote.0.1.0)" {
		t.Fatal(got)
	}
	expectedUpstreamVersion := "v" + version.UpstreamVersion
	expectedCompatibilityVersion := version.CompatibilityVersion()
	cfg := config.New()
	cfg.App.SingleUser = false
	ni := nodeInfoConfig(nil, cfg)
	if ni.Software.Name != "writefreely" || ni.Software.Version != expectedCompatibilityVersion || ni.Metadata.Software.GitHub != version.SourceURL || ni.Metadata.Software.HomePage != version.SourceURL {
		t.Fatalf("unexpected NodeInfo: %+v", ni)
	}
	p := page.StaticPage{Version: expectedUpstreamVersion}
	if p.Version != "v0.17.2" || p.OfficialVersion() != "v0.17.2" || wfReleaseNotesURL(p.Version) != "https://blog.writefreely.org/version-0-17-2" {
		t.Fatalf("upstream URLs changed: %+v", p)
	}
	// Exercise comparison without creating a cache that launches a network request.
	uc := updatesCache{currentVersion: expectedUpstreamVersion, latestVersion: expectedUpstreamVersion, lastCheck: time.Now()}
	if uc.AreAvailableNoCheck() {
		t.Fatal("base release incorrectly reported as an update")
	}
	uc.latestVersion = "v0.17.3"
	if !uc.AreAvailableNoCheck() {
		t.Fatal("upstream update not detected")
	}
	if serverSoftware != "WriteFreely" {
		t.Fatal("Server header identity changed")
	}
}

func TestBlueNoteBrandingPages(t *testing.T) {
	for _, mode := range []struct {
		name                   string
		single, chorus, modest bool
	}{
		{"single", true, false, false},
		{"multi", false, false, false},
		{"modest", false, false, true},
		{"chorus", false, true, false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			app, router := newTemplateTestApp(t, func(c *config.Config) {
				c.App.SingleUser, c.App.Chorus, c.App.WFModesty = mode.single, mode.chorus, mode.modest
			})
			u, coll, post := createTemplateTestUser(t, app, "tester")
			cookie := loginCookie(t, app, u)
			prefix := "/" + coll.Alias
			if mode.single {
				prefix = ""
			}
			for _, path := range []string{prefix + "/", prefix + "/archive/", prefix + "/" + post.Slug.String, "/me/settings", "/admin/monitor", "/admin/updates"} {
				t.Run(path, func(t *testing.T) {
					rec := assertRendersCleanly(t, router, "GET", path, []*http.Cookie{cookie}, http.StatusOK)
					for _, want := range []string{"powered by", ">WriteFreely</a>", "BlueNote " + version.BlueNoteVersion, `href="` + version.SourceURL + `">Source</a>`} {
						if !strings.Contains(rec.Body.String(), want) {
							t.Errorf("missing %q in %s", want, path)
						}
					}
					if strings.HasPrefix(path, "/admin/") && !strings.Contains(rec.Body.String(), "WriteFreely base") {
						t.Error("missing upstream label")
					}
				})
			}
			// Exercise real NodeInfo serialization and discovery route.
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/nodeinfo", nil))
			var ni struct {
				Version  string `json:"version"`
				Software struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"software"`
				Metadata struct {
					Software struct {
						HomePage string `json:"homepage"`
						GitHub   string `json:"github"`
					} `json:"software"`
				} `json:"metadata"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &ni); err != nil {
				t.Fatal(err)
			}
			if ni.Version != "2.0" || ni.Software.Name != "writefreely" || ni.Software.Version != version.CompatibilityVersion() {
				t.Fatalf("unexpected NodeInfo: %+v", ni)
			}
			if ni.Metadata.Software.HomePage != version.SourceURL || ni.Metadata.Software.GitHub != version.SourceURL {
				t.Fatalf("unexpected NodeInfo source metadata: %+v", ni.Metadata.Software)
			}
			// Both footer layouts must render without depending on a request or JS.
			p := pageForReq(app, httptest.NewRequest("GET", "/", nil))
			for _, name := range []string{"footer", "branding"} {
				var b bytes.Buffer
				if err := templates["collection"].ExecuteTemplate(&b, name, p); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(b.String(), "BlueNote "+version.BlueNoteVersion) {
					t.Fatal("branding missing from " + name)
				}
			}
		})
	}
}
