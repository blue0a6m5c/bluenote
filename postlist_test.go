package writefreely

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/writefreely/writefreely/config"
)

func TestManagementPostsPaging(t *testing.T) {
	app, router := newTemplateTestApp(t, nil)
	u, c, _ := createTemplateTestUser(t, app, "listpaging")
	_, err := app.db.Exec("DELETE FROM posts WHERE collection_id = ?", c.ID)
	require.NoError(t, err)
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 51; i++ {
		_, err = app.db.Exec("INSERT INTO posts (id, owner_id, collection_id, title, content, created, privacy, view_count) VALUES (?, ?, ?, ?, ?, ?, 0, 0)", fmt.Sprintf("list%06d", i), u.ID, c.ID, "title", "body excluded", created)
		require.NoError(t, err)
	}
	first, total, err := app.db.GetManagementPosts(c.ID, u.ID, 1)
	require.NoError(t, err)
	require.Equal(t, 51, total)
	require.Len(t, first, 50)
	for i, p := range first {
		require.Equal(t, fmt.Sprintf("list%06d", 50-i), p.ID)
	}
	second, _, err := app.db.GetManagementPosts(c.ID, u.ID, 2)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, "list000000", second[0].ID)
	for _, page := range []string{"1", "2"} {
		rec := assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts?p="+page, []*http.Cookie{loginCookie(t, app, u)}, 200)
		if page == "1" {
			require.Contains(t, rec.Body.String(), `rel="next" href="?p=2"`)
		} else {
			require.NotContains(t, rec.Body.String(), `rel="next"`)
		}
	}
	absent, _, err := app.db.GetManagementPosts(c.ID, u.ID, int(^uint(0)>>1))
	require.NoError(t, err)
	require.Empty(t, absent)
	denied, total, err := app.db.GetManagementPosts(c.ID, u.ID+999, 1)
	require.NoError(t, err)
	require.Empty(t, denied)
	require.Zero(t, total)
}

func TestManagementPostsExcludeDraftsAndOtherCollections(t *testing.T) {
	app, router := newTemplateTestApp(t, nil)
	u, c, p := createTemplateTestUser(t, app, "listdrafts")
	_, _, _ = createTemplateTestUser(t, app, "anotherlist")
	_, err := app.db.Exec("UPDATE posts SET collection_id = NULL WHERE id = ?", p.ID)
	require.NoError(t, err)
	posts, total, err := app.db.GetManagementPosts(c.ID, u.ID, 1)
	require.NoError(t, err)
	require.Empty(t, posts)
	require.Zero(t, total)
	rec := assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts", []*http.Cookie{loginCookie(t, app, u)}, 200)
	require.Contains(t, rec.Body.String(), "No posts on this page.")
	require.NotContains(t, rec.Body.String(), `rel="next"`)
	_, err = app.db.Exec("UPDATE posts SET collection_id = ?, title = '' WHERE id = ?", c.ID, p.ID)
	require.NoError(t, err)
	rec = assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts", []*http.Cookie{loginCookie(t, app, u)}, 200)
	require.Contains(t, rec.Body.String(), "Untitled")
}

func TestManagementPostsPage(t *testing.T) {
	for _, single := range []bool{true, false} {
		t.Run(fmt.Sprint(single), func(t *testing.T) {
			app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.SingleUser = single })
			u, c, p := createTemplateTestUser(t, app, "listview")
			future := time.Now().UTC().Add(48 * time.Hour)
			_, err := app.db.Exec("UPDATE posts SET title = ?, slug = ?, created = ?, pinned_position = 1 WHERE id = ?", "<script>alert(1)</script>", "20260907010203", future, p.ID)
			require.NoError(t, err)
			for _, privacy := range []int{0, int(CollPrivate), int(CollProtected)} {
				_, err = app.db.Exec("UPDATE collections SET privacy = ? WHERE id = ?", privacy, c.ID)
				require.NoError(t, err)
				rec := assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts", []*http.Cookie{loginCookie(t, app, u)}, 200)
				body := rec.Body.String()
				require.Contains(t, body, "&lt;script&gt;")
				require.NotContains(t, body, "<script>alert(1)</script>")
				require.NotContains(t, body, "This is a **test** post")
				require.Contains(t, body, "Pinned")
				require.Contains(t, body, "Scheduled")
				require.Regexp(t, regexp.MustCompile(`data-csrf-token="[^"]+"`), body)
				prefix := "/"
				if !single {
					prefix += c.Alias + "/"
				}
				require.Contains(t, body, `href="`+prefix+`20260907010203/edit"`)
				require.NotContains(t, body, `rel="next"`)
			}
			for _, query := range []string{"?p=0", "?p=-1", "?p=abc", "?p=99999999999999999999999999999"} {
				rec := assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts"+query, []*http.Cookie{loginCookie(t, app, u)}, 200)
				require.Contains(t, rec.Body.String(), "Page 1")
			}
			rec := assertRendersCleanly(t, router, "GET", "/me/c/"+c.Alias+"/posts?p=2", []*http.Cookie{loginCookie(t, app, u)}, 200)
			require.Contains(t, rec.Body.String(), "No posts on this page.")
			require.NotContains(t, rec.Body.String(), `rel="next"`)
			other, _, _ := createTemplateTestUser(t, app, "listother")
			rec, _ = renderedRequest(t, router, "GET", "/me/c/"+c.Alias+"/posts", []*http.Cookie{loginCookie(t, app, other)})
			require.Equal(t, 404, rec.Code)
			rec, _ = renderedRequest(t, router, "GET", "/me/c/"+c.Alias+"/posts", nil)
			require.NotEqual(t, 200, rec.Code)
		})
	}
}

func TestManagementPostActions(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	u, c, p := createTemplateTestUser(t, app, "listactions")
	session := loginCookie(t, app, u)
	baseURL := strings.TrimRight(app.cfg.App.Host, "/")
	csrfToken, csrfCookie := fetchCSRFToken(t, router, baseURL)

	request := func(method, path, body, token string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, baseURL+path, strings.NewReader(body))
		req.Header.Set("Referer", baseURL+"/me/c/"+c.Alias+"/posts")
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.AddCookie(session)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if token != "" {
			req.Header.Set("X-CSRF-Token", token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		pinned interface{}
	}{
		{"pin", "POST", "/api/collections/" + c.Alias + "/pin", `[{"id":"` + p.ID + `"}]`, nil},
		{"unpin", "POST", "/api/collections/" + c.Alias + "/unpin", `[{"id":"` + p.ID + `"}]`, 1},
		{"delete", "DELETE", "/api/collections/" + c.Alias + "/posts/" + p.ID + "/delete", "", nil},
	} {
		t.Run(tc.name+" rejects missing and invalid CSRF tokens", func(t *testing.T) {
			_, err := app.db.Exec("UPDATE posts SET pinned_position = ? WHERE id = ?", tc.pinned, p.ID)
			require.NoError(t, err)
			for _, credentials := range []struct {
				name   string
				token  string
				cookie *http.Cookie
			}{
				{"missing", "", nil},
				{"invalid", "invalid-token", csrfCookie},
			} {
				rec := request(tc.method, tc.path, tc.body, credentials.token, credentials.cookie)
				require.Equal(t, http.StatusForbidden, rec.Code, credentials.name)
				var count int
				require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
				require.Equal(t, 1, count)
				require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND pinned_position IS NOT NULL", p.ID).Scan(&count))
				if tc.pinned == nil {
					require.Zero(t, count)
				} else {
					require.Equal(t, 1, count)
				}
			}
		})
	}
	rec := request("POST", "/api/collections/"+c.Alias+"/pin?token=not-a-pin-credential", `[{"id":"`+p.ID+`"}]`, "", nil)
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = request("POST", "/api/collections/"+c.Alias+"/pin", `[{"id":"`+p.ID+`"}]`, csrfToken, csrfCookie)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"code":200`)
	var count int
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND pinned_position IS NOT NULL", p.ID).Scan(&count))
	require.Equal(t, 1, count)
	rec = request("POST", "/api/collections/"+c.Alias+"/unpin", `[{"id":"`+p.ID+`"}]`, csrfToken, csrfCookie)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND pinned_position IS NOT NULL", p.ID).Scan(&count))
	require.Zero(t, count)
	rec = request("DELETE", "/api/collections/"+c.Alias+"/posts/"+p.ID+"/delete", "", csrfToken, csrfCookie)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
	require.Zero(t, count)
}

func TestManagementPostActionsWithAuthorization(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	u, c, p := createTemplateTestUser(t, app, "listtoken")
	token, err := app.db.GetAccessToken(u.ID)
	require.NoError(t, err)

	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Token "+token)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	require.Equal(t, http.StatusOK, request("POST", "/api/collections/"+c.Alias+"/pin", `[{"id":"`+p.ID+`"}]`).Code)
	var count int
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND pinned_position IS NOT NULL", p.ID).Scan(&count))
	require.Equal(t, 1, count)
	require.Equal(t, http.StatusOK, request("POST", "/api/collections/"+c.Alias+"/unpin", `[{"id":"`+p.ID+`"}]`).Code)
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND pinned_position IS NOT NULL", p.ID).Scan(&count))
	require.Zero(t, count)
	require.Equal(t, http.StatusNoContent, request("DELETE", "/api/posts/"+p.ID, "").Code)
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
	require.Zero(t, count)
}

func TestScopedDeleteAuthorizationTokens(t *testing.T) {
	for _, tc := range []struct {
		name    string
		oneTime bool
	}{
		{"persistent", false},
		{"one-time", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
			u, c, p := createTemplateTestUser(t, app, "listscopedtoken"+strings.ReplaceAll(tc.name, "-", ""))
			token, err := app.db.GetTemporaryOneTimeAccessToken(u.ID, 60, tc.oneTime)
			require.NoError(t, err)
			req := httptest.NewRequest("DELETE", "/api/collections/"+c.Alias+"/posts/"+p.ID+"/delete", nil)
			req.Header.Set("Authorization", "Token "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
			var count int
			require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
			require.Zero(t, count)
		})
	}
}

func TestAnonymousModifyTokenDeleteCompatibility(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	const modifyToken = "12345678901234567890123456789012"
	for _, tc := range []struct {
		name        string
		postID      string
		requestPath string
		body        string
	}{
		{"query", "anonquery1", "/api/posts/anonquery1?token=" + modifyToken, ""},
		{"form body", "anonform01", "/api/posts/anonform01", "token=" + modifyToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.db.Exec("INSERT INTO posts (id, modify_token, title, content, created, privacy, view_count) VALUES (?, ?, ?, ?, ?, 0, 0)", tc.postID, modifyToken, "Anonymous", "body", time.Now().UTC())
			require.NoError(t, err)
			req := httptest.NewRequest("DELETE", tc.requestPath, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
			var count int
			require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", tc.postID).Scan(&count))
			require.Zero(t, count)
		})
	}
}

func TestModifyTokenCannotUseCookieAuthority(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	u, c, p := createTemplateTestUser(t, app, "listmodifybypass")
	const suppliedToken = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	for _, tc := range []struct {
		name        string
		requestPath string
		body        string
		wantStatus  int
	}{
		{"generic query uses modify-token auth", "/api/posts/" + p.ID + "?token=" + suppliedToken, "", http.StatusConflict},
		{"generic form uses modify-token auth", "/api/posts/" + p.ID, "token=" + suppliedToken, http.StatusConflict},
		{"scoped form cannot bypass CSRF", "/api/collections/" + c.Alias + "/posts/" + p.ID + "/delete", "token=" + suppliedToken, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", tc.requestPath, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(loginCookie(t, app, u))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, tc.wantStatus, rec.Code)
			var count int
			require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}

func TestManagementDeleteIsCollectionScoped(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	u, c, _ := createTemplateTestUser(t, app, "listscope")
	other, err := app.db.CreateCollection(app.cfg, "listscopeother", "Other", u.ID)
	require.NoError(t, err)
	const postID = "scopepost1"
	_, err = app.db.Exec("INSERT INTO posts (id, owner_id, collection_id, title, content, created, privacy, view_count) VALUES (?, ?, ?, ?, ?, ?, 0, 0)", postID, u.ID, other.ID, "Other post", "body", time.Now().UTC())
	require.NoError(t, err)

	baseURL := strings.TrimRight(app.cfg.App.Host, "/")
	token, csrfCookie := fetchCSRFToken(t, router, baseURL)
	deletePost := func(alias string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("DELETE", baseURL+"/api/collections/"+alias+"/posts/"+postID+"/delete", nil)
		req.Header.Set("Referer", baseURL+"/me/c/"+alias+"/posts")
		req.AddCookie(loginCookie(t, app, u))
		req.AddCookie(csrfCookie)
		req.Header.Set("X-CSRF-Token", token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	denied := deletePost(c.Alias)
	require.Equal(t, http.StatusUnauthorized, denied.Code, denied.Body.String())
	var count int
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND collection_id = ?", postID, other.ID).Scan(&count))
	require.Equal(t, 1, count)
	require.Equal(t, http.StatusNoContent, deletePost(other.Alias).Code)
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", postID).Scan(&count))
	require.Zero(t, count)
}

func TestScopedDeleteRejectsOtherMethods(t *testing.T) {
	app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Federation = false })
	_, c, p := createTemplateTestUser(t, app, "listmethods")
	path := "/api/collections/" + c.Alias + "/posts/" + p.ID + "/delete"
	for _, method := range []string{"GET", "PATCH", "POST", "PUT"} {
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusMethodNotAllowed, rec.Code, method)
		require.Equal(t, "DELETE", rec.Header().Get("Allow"), method)
	}
	var count int
	require.NoError(t, app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", p.ID).Scan(&count))
	require.Equal(t, 1, count)
}

func TestCSRFCookieSecureFollowsConfiguredHost(t *testing.T) {
	for _, tc := range []struct {
		name   string
		host   string
		secure bool
	}{
		{"HTTP development", "http://localhost:8080", false},
		{"HTTPS production", "https://example.com", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, router := newTemplateTestApp(t, func(c *config.Config) { c.App.Host = tc.host })
			_, cookie := fetchCSRFToken(t, router, strings.TrimRight(app.cfg.App.Host, "/"))
			require.Equal(t, tc.secure, cookie.Secure)
		})
	}
}

func fetchCSRFToken(t *testing.T, router http.Handler, baseURL string) (string, *http.Cookie) {
	t.Helper()
	req := httptest.NewRequest("GET", baseURL+"/api/csrf", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.NotEmpty(t, response.Data.Token)
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "_gorilla_csrf" {
			return response.Data.Token, cookie
		}
	}
	t.Fatal("CSRF cookie was not set")
	return "", nil
}
