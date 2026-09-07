/*
 * Copyright © 2026 BlueNote contributors.
 * This file is part of WriteFreely, licensed under the GNU AGPLv3.
 * Post management UI inspired by writefreely-wisp PR #5.
 */
package writefreely

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/writeas/impart"
)

const managementPageSize = 50

func handleCSRFToken(app *App, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "no-store")
	return impart.WriteSuccess(w, struct {
		Token string `json:"token"`
	}{Token: csrf.Token(r)}, http.StatusOK)
}

// Management rows intentionally exclude post bodies and public rendering state.
type managementPost struct {
	ID        string
	Slug      sql.NullString
	Title     sql.NullString
	Created   time.Time
	Pinned    sql.NullInt64
	URL       string
	Scheduled bool
}

func (p managementPost) Date() string { return p.Created.UTC().Format(time.RFC3339) }

func (db *datastore) GetManagementPosts(collectionID, ownerID int64, page int) ([]managementPost, int, error) {
	var total int
	// Repeat the collection ownership constraint at the data boundary.
	condition := " FROM posts p INNER JOIN collections c ON c.id = p.collection_id WHERE c.id = ? AND c.owner_id = ?"
	if err := db.QueryRow("SELECT COUNT(*)"+condition, collectionID, ownerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	last := (total + managementPageSize - 1) / managementPageSize
	if page < 1 {
		page = 1
	}
	if page > last && page != 1 {
		return []managementPost{}, total, nil
	}
	rows, err := db.Query("SELECT p.id, p.slug, p.title, p.created, p.pinned_position"+condition+" ORDER BY p.created DESC, p.id DESC LIMIT ? OFFSET ?", collectionID, ownerID, managementPageSize, (page-1)*managementPageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	posts := make([]managementPost, 0)
	for rows.Next() {
		var p managementPost
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Created, &p.Pinned); err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

type managementPostsPage struct {
	*UserPage
	Collection   *Collection
	Alias        string
	SingleUser   bool
	Posts        []managementPost
	CurrentPage  int
	PreviousPage int
	NextPage     int
	Silenced     bool
	CSRFToken    string
}

func viewManagementPosts(app *App, u *User, w http.ResponseWriter, r *http.Request) error {
	coll, err := app.db.GetCollection(mux.Vars(r)["collection"])
	if err != nil {
		return err
	}
	if coll.OwnerID != u.ID {
		return ErrCollectionNotFound
	}
	page, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil || page < 1 {
		page = 1
	}
	posts, total, err := app.db.GetManagementPosts(coll.ID, u.ID, page)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range posts {
		path := "/"
		if !app.cfg.App.SingleUser {
			path += url.PathEscape(coll.Alias) + "/"
		}
		posts[i].URL = path + url.PathEscape(posts[i].Slug.String)
		posts[i].Scheduled = posts[i].Created.After(now)
	}
	flashes, _ := getSessionFlashes(app, w, r, nil)
	data := managementPostsPage{UserPage: NewUserPage(app, r, u, coll.DisplayTitle()+" Posts", flashes), Collection: coll, Alias: coll.Alias, SingleUser: app.cfg.App.SingleUser, Posts: posts, CurrentPage: page, Silenced: u.IsSilenced(), CSRFToken: csrf.Token(r)}
	data.UserPage.CollAlias = coll.Alias
	if page > 1 {
		data.PreviousPage = page - 1
	}
	if page <= (total-1)/managementPageSize && total > 0 {
		data.NextPage = page + 1
	}
	showUserPage(w, "collection-posts", data)
	return nil
}

// deleteCollectionPost delegates authentication, collection validation, and
// deletion to the shared path so one-time access tokens are consumed once.
func deleteCollectionPost(app *App, w http.ResponseWriter, r *http.Request) error {
	return deletePostFromCollection(app, w, r, mux.Vars(r)["alias"])
}
