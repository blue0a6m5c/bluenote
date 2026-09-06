/*
 * Copyright © 2026 Joseph Quigley.
 * Copyright © 2026 BlueNote contributors.
 *
 * This file is part of WriteFreely.
 *
 * WriteFreely is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, included
 * in the LICENSE file in this source code package.
 */

package writefreely

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/writeas/impart"
	"github.com/writefreely/writefreely/config"
)

func saveVerificationLinks(t *testing.T, app *App, owner *User, coll *Collection, submitted string) error {
	t.Helper()
	title := coll.Title
	return app.db.UpdateCollection(app, &SubmittedCollection{
		OwnerID:      uint64(owner.ID),
		Alias:        &coll.Alias,
		Title:        &title,
		Verification: &submitted,
	}, coll.Alias)
}

func TestParseVerificationLinks(t *testing.T) {
	assert.Equal(t, []string{}, parseVerificationLinks(""))
	assert.Equal(t,
		[]string{"https://a.example", "https://b.example"},
		parseVerificationLinks(" https://a.example \r\n\nhttps://b.example\nhttps://a.example "),
	)
}

func TestNormalizeVerificationURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"absolute HTTPS", "https://example.com/@writer", "https://example.com/@writer"},
		{"absolute HTTP", "http://example.com/profile", "http://example.com/profile"},
		{"scheme omitted", "example.com/profile", "https://example.com/profile"},
		{"protocol relative", "//example.com/profile", "https://example.com/profile"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeVerificationURL(test.input)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestNormalizeVerificationURLRejectsUnsafeOrInvalidValues(t *testing.T) {
	for _, value := range []string{
		"javascript:alert(1)",
		"mailto:writer@example.com",
		"ftp://example.com/profile",
		"https:///missing-host",
		"https://user:password@example.com/profile",
		"https://example.com/%zz",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := normalizeVerificationURL(value)
			assert.Error(t, err)
		})
	}
}

func TestNormalizeVerificationLinksLimitsCountAfterDeduplication(t *testing.T) {
	five := strings.Join([]string{
		"https://a.example", "https://b.example", "https://c.example",
		"https://d.example", "https://e.example",
	}, "\n")
	links, err := normalizeVerificationLinks(nil, five+"\nhttps://a.example")
	require.NoError(t, err)
	assert.Len(t, links, maxVerificationLinks)

	_, err = normalizeVerificationLinks(nil, five+"\nhttps://f.example")
	assertHTTPErrorStatus(t, err, http.StatusBadRequest)
}

func TestNormalizeVerificationLinksEnforcesMySQLAttributeLimit(t *testing.T) {
	prefix := "https://example.com/"
	exact := prefix + strings.Repeat("a", maxVerificationLinksStoredLength-utf8.RuneCountInString(prefix))
	links, err := normalizeVerificationLinks(nil, exact)
	require.NoError(t, err)
	assert.Equal(t, maxVerificationLinksStoredLength, utf8.RuneCountInString(serializeVerificationLinks(links)))

	_, err = normalizeVerificationLinks(nil, exact+"a")
	assertHTTPErrorStatus(t, err, http.StatusBadRequest)
}

func assertHTTPErrorStatus(t *testing.T, err error, status int) {
	t.Helper()
	require.Error(t, err)
	httpErr, ok := err.(impart.HTTPError)
	require.True(t, ok, "error type = %T", err)
	assert.Equal(t, status, httpErr.Status)
}

func TestVerificationLinksPersistWithoutMigration(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	owner, coll, _ := createTemplateTestUser(t, app, "verifier")

	err := saveVerificationLinks(t, app, owner, coll, "example.com/me\nhttps://social.example/@me")
	require.NoError(t, err)

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/me", "https://social.example/@me"}, loaded.VerificationLinks)
	assert.Equal(t, "https://example.com/me", loaded.Verification())
	assert.Equal(t, loaded.Verification(), loaded.VerificationLink)

	stored := app.db.GetCollectionAttribute(coll.ID, "verification_link")
	assert.Equal(t, "https://example.com/me\nhttps://social.example/@me", stored)
}

func TestLegacySingleVerificationLinkStillLoadsAndSerializes(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	_, coll, _ := createTemplateTestUser(t, app, "legacyverify")
	require.NoError(t, app.db.SetCollectionAttribute(coll.ID, "verification_link", "https://only.example/me"))

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://only.example/me"}, loaded.VerificationLinks)

	b, err := json.Marshal(loaded)
	require.NoError(t, err)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &data))
	assert.Equal(t, "https://only.example/me", data["verification_link"])
	assert.Equal(t, []interface{}{"https://only.example/me"}, data["verification_links"])
}

func TestVerificationHandleNormalizationRemainsSupported(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	owner, coll, _ := createTemplateTestUser(t, app, "handleverify")

	require.NoError(t, saveVerificationLinks(t, app, owner, coll, "@blue0a6m5c@github.com"))
	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://github.com/blue0a6m5c"}, loaded.VerificationLinks)
}

func TestClearingVerificationLinksRemovesAllLinks(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	owner, coll, _ := createTemplateTestUser(t, app, "clearverify")
	require.NoError(t, saveVerificationLinks(t, app, owner, coll, "https://a.example\nhttps://b.example"))
	require.NoError(t, saveVerificationLinks(t, app, owner, coll, ""))

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{}, loaded.VerificationLinks)
	assert.Equal(t, "", loaded.VerificationLink)
}

func TestInvalidVerificationLinkDoesNotReplaceStoredLinks(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	owner, coll, _ := createTemplateTestUser(t, app, "invalidverify")
	require.NoError(t, saveVerificationLinks(t, app, owner, coll, "https://owner.example/me"))

	err := saveVerificationLinks(t, app, owner, coll, "javascript:alert(1)")
	assertHTTPErrorStatus(t, err, http.StatusBadRequest)

	loaded, loadErr := app.db.GetCollection(coll.Alias)
	require.NoError(t, loadErr)
	assert.Equal(t, []string{"https://owner.example/me"}, loaded.VerificationLinks)
}

func TestCollectionAPIEmitsNewAndLegacyVerificationFields(t *testing.T) {
	app, router := newTemplateTestApp(t, func(cfg *config.Config) { cfg.App.SingleUser = false })
	owner, coll, _ := createTemplateTestUser(t, app, "apiverify")
	require.NoError(t, saveVerificationLinks(t, app, owner, coll,
		"https://first.example/me\nhttps://second.example/me"))

	req := httptest.NewRequest("GET", "/api/collections/"+coll.Alias, nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response struct {
		Data struct {
			VerificationLinks []string `json:"verification_links"`
			VerificationLink  string   `json:"verification_link"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, []string{"https://first.example/me", "https://second.example/me"}, response.Data.VerificationLinks)
	assert.Equal(t, "https://first.example/me", response.Data.VerificationLink)
}

func TestCollectionJSONAPIUpdatesVerificationLinksWithoutOtherFields(t *testing.T) {
	app, router := newTemplateTestApp(t, func(cfg *config.Config) { cfg.App.SingleUser = false })
	owner, coll, _ := createTemplateTestUser(t, app, "apiverifyupdate")

	body := `{"verification_link":"https://first.example/me\nhttps://second.example/me"}`
	req := httptest.NewRequest("POST", "/api/collections/"+coll.Alias, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token, err := app.db.GetAccessToken(owner.ID)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://first.example/me", "https://second.example/me"}, loaded.VerificationLinks)
}

func TestRendersEveryVerificationLinkInSingleAndMultiUserModes(t *testing.T) {
	for _, singleUser := range []bool{true, false} {
		t.Run(map[bool]string{true: "single-user", false: "multi-user"}[singleUser], func(t *testing.T) {
			app, router := newTemplateTestApp(t, func(cfg *config.Config) {
				cfg.App.SingleUser = singleUser
			})
			owner, coll, _ := createTemplateTestUser(t, app, "htmlverify")
			require.NoError(t, saveVerificationLinks(t, app, owner, coll,
				"https://first.example/me?a=1&b=2\nhttps://second.example/@me"))

			path := "/" + coll.Alias + "/"
			if singleUser {
				path = "/"
			}
			rec := assertRendersCleanly(t, router, "GET", path, nil, http.StatusOK)
			body := rec.Body.String()
			first := `<link rel="me" href="https://first.example/me?a=1&amp;b=2" />`
			second := `<link rel="me" href="https://second.example/@me" />`
			assert.Contains(t, body, first)
			assert.Contains(t, body, second)
			assert.Less(t, strings.Index(body, first), strings.Index(body, second))
		})
	}
}

func TestFediverseCreatorUsesOnlyFirstVerificationLink(t *testing.T) {
	app, router := newTemplateTestApp(t, func(cfg *config.Config) { cfg.App.SingleUser = false })
	owner, coll, post := createTemplateTestUser(t, app, "fedicreator")
	for _, remote := range []struct{ profile, handle string }{
		{"https://first.example/@one", "one@first.example"},
		{"https://second.example/@two", "two@second.example"},
	} {
		_, err := app.db.Exec("INSERT INTO remoteusers (actor_id, inbox, shared_inbox, url, handle) VALUES (?, ?, ?, ?, ?)",
			remote.profile, remote.profile+"/inbox", remote.profile+"/inbox", remote.profile, remote.handle)
		require.NoError(t, err)
	}
	require.NoError(t, saveVerificationLinks(t, app, owner, coll,
		"https://first.example/@one\nhttps://second.example/@two"))

	rec := assertRendersCleanly(t, router, "GET", "/"+coll.Alias+"/"+post.Slug.String, nil, http.StatusOK)
	body := rec.Body.String()
	assert.Equal(t, 1, strings.Count(body, `name="fediverse:creator"`))
	assert.Contains(t, body, `<meta name="fediverse:creator" content="@one@first.example">`)
	assert.NotContains(t, body, "@two@second.example")
}

func TestSettingsRowsSubmitAllVerificationLinks(t *testing.T) {
	app, router := newTemplateTestApp(t, func(cfg *config.Config) { cfg.App.SingleUser = false })
	owner, coll, _ := createTemplateTestUser(t, app, "rowverify")
	cookie := loginCookie(t, app, owner)

	form := url.Values{
		"web":                   {"1"},
		"title":                 {coll.Title},
		"verification_link_row": {"https://a.example/me", " ", "https://b.example/me"},
	}
	req := httptest.NewRequest("POST", "/api/collections/"+coll.Alias, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code, rec.Body.String())

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.example/me", "https://b.example/me"}, loaded.VerificationLinks)

	rec = assertRendersCleanly(t, router, "GET", "/me/c/"+coll.Alias, []*http.Cookie{cookie}, http.StatusOK)
	assert.Equal(t, 2, strings.Count(rec.Body.String(), `name="verification_link_row"`))
}

func TestNonOwnerCannotChangeVerificationLinks(t *testing.T) {
	app, router := newTemplateTestApp(t, func(cfg *config.Config) { cfg.App.SingleUser = false })
	owner, coll, _ := createTemplateTestUser(t, app, "verifyowner")
	other, _, _ := createTemplateTestUser(t, app, "verifyother")
	require.NoError(t, saveVerificationLinks(t, app, owner, coll, "https://owner.example/me"))

	form := url.Values{
		"web":                   {"1"},
		"title":                 {coll.Title},
		"verification_link_row": {"https://attacker.example/me"},
	}
	req := httptest.NewRequest("POST", "/api/collections/"+coll.Alias, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(loginCookie(t, app, other))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://owner.example/me"}, loaded.VerificationLinks)
}

func TestVerificationLinksDoNotChangeActivityPubActor(t *testing.T) {
	app, _ := newTemplateTestApp(t, nil)
	owner, coll, _ := createTemplateTestUser(t, app, "actorverify")
	require.NoError(t, saveVerificationLinks(t, app, owner, coll, "https://profile.example/me"))

	loaded, err := app.db.GetCollection(coll.Alias)
	require.NoError(t, err)
	loaded.hostName = app.cfg.App.Host
	actorJSON, err := json.Marshal(loaded.PersonObject())
	require.NoError(t, err)
	assert.NotContains(t, string(actorJSON), "profile.example")
	assert.NotContains(t, string(actorJSON), "verification_link")
}
