package writefreely

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/guregu/null/zero"
	"github.com/writeas/web-core/activitystreams"
	"github.com/writefreely/writefreely/config"
)

type activityTagWire struct {
	Type string `json:"type"`
	HRef string `json:"href"`
	Name string `json:"name"`
}

type activityObjectWire struct {
	Type       string                   `json:"type"`
	Summary    *string                  `json:"summary"`
	Content    string                   `json:"content"`
	Tag        []activityTagWire        `json:"tag"`
	Attachment []activityAttachmentWire `json:"attachment"`
	Preview    struct {
		Content    string                   `json:"content"`
		Tag        []activityTagWire        `json:"tag"`
		Attachment []activityAttachmentWire `json:"attachment"`
	} `json:"preview"`
}

type activityAttachmentWire struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

func newActivityHashtagTestPost(host, content string) *PublicPost {
	return &PublicPost{
		Post: &Post{
			ID:      "testpost01",
			Slug:    null.NewString("article-slug", true),
			Title:   zero.NewString("Article title", true),
			Content: content,
			Created: time.Date(2026, 9, 6, 1, 2, 3, 0, time.UTC),
		},
		Collection: &CollectionObj{Collection: Collection{Alias: "blog", hostName: host}},
	}
}

func marshalActivityObject(t *testing.T, value interface{}) activityObjectWire {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var envelope struct {
		Object *activityObjectWire `json:"object"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal serialized ActivityPub JSON: %v\n%s", err, data)
	}
	if envelope.Object != nil {
		return *envelope.Object
	}

	var object activityObjectWire
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("unmarshal serialized ActivityPub object: %v\n%s", err, data)
	}
	return object
}

func requireHashtags(t *testing.T, got []activityTagWire, want map[string]string) {
	t.Helper()
	actual := map[string]string{}
	for _, tag := range got {
		if tag.Type != string(activitystreams.TagHashtag) {
			continue
		}
		if _, exists := actual[tag.Name]; exists {
			t.Fatalf("duplicate ActivityPub Hashtag %q: %+v", tag.Name, got)
		}
		actual[tag.Name] = tag.HRef
	}
	if len(actual) != len(want) {
		t.Fatalf("Hashtags = %+v, want %+v", actual, want)
	}
	for name, href := range want {
		if actual[name] != href {
			t.Errorf("Hashtag %q href = %q, want %q", name, actual[name], href)
		}
	}
}

func TestActivityObjectSerializesHashtagsFromContent(t *testing.T) {
	cfg := config.New()
	cfg.App.Host = "https://example.com"
	app := &App{cfg: cfg}
	content := "Tags: #BlueNote #日本語 #go123 #CamelCase.\n\nDuplicate: #BlueNote."
	post := newActivityHashtagTestPost(cfg.App.Host, content)
	if post.Tags != nil {
		t.Fatal("test requires Post.Tags to be unset")
	}

	want := map[string]string{
		"#BlueNote":  "https://example.com/blog/tag:BlueNote",
		"#日本語":       "https://example.com/blog/tag:日本語",
		"#go123":     "https://example.com/blog/tag:go123",
		"#CamelCase": "https://example.com/blog/tag:CamelCase",
	}

	object := marshalActivityObject(t, post.ActivityObject(app))
	if object.Type != "Article" {
		t.Fatalf("type = %q, want Article", object.Type)
	}
	requireHashtags(t, object.Tag, want)
	requireHashtags(t, object.Preview.Tag, want)
}

func TestActivityHashtagsAreIndependentOfCallingPath(t *testing.T) {
	cfg := config.New()
	cfg.App.Host = "https://example.com"
	app := &App{cfg: cfg}
	content := "A #BlueNote article.\n\nMore content."
	want := map[string]string{"#BlueNote": "https://example.com/blog/tag:BlueNote"}

	// These initial Tags values model what each caller currently passes to
	// ActivityObject. ActivityPub generation must produce the same result for all
	// of them, including callers that never invoked extractData.
	paths := []struct {
		name string
		tags []string
	}{
		{"new post", []string{"BlueNote"}},
		{"web form update", nil},
		{"JSON API update", []string{"BlueNote"}},
		{"file import", nil},
		{"collection claim", nil},
		{"outbox", []string{"BlueNote"}},
		{"ActivityPub GET", []string{"BlueNote"}},
		{"stale precomputed tags", []string{"OldTag"}},
	}

	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			post := newActivityHashtagTestPost(cfg.App.Host, content)
			post.Tags = path.tags
			requireHashtags(t, marshalActivityObject(t, post.ActivityObject(app)).Tag, want)
		})
	}
}

func TestCreateAndUpdateActivitiesSerializeHashtags(t *testing.T) {
	cfg := config.New()
	cfg.App.Host = "https://example.com"
	app := &App{cfg: cfg}
	want := map[string]string{"#BlueNote": "https://example.com/blog/tag:BlueNote"}

	for _, activityType := range []string{"Create", "Update"} {
		t.Run(activityType, func(t *testing.T) {
			post := newActivityHashtagTestPost(cfg.App.Host, "A #BlueNote article.\n\nMore content.")
			object := post.ActivityObject(app)
			var activity interface{}
			if activityType == "Create" {
				activity = activitystreams.NewCreateActivity(object)
			} else {
				activity = activitystreams.NewUpdateActivity(object)
			}
			data, err := json.Marshal(activity)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Type   string             `json:"type"`
				Object activityObjectWire `json:"object"`
			}
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatalf("unmarshal serialized %s Activity: %v\n%s", activityType, err, data)
			}
			if wire.Type != activityType {
				t.Fatalf("Activity type = %q, want %q", wire.Type, activityType)
			}
			requireHashtags(t, wire.Object.Tag, want)
		})
	}
}

func TestActivityHashtagURLModes(t *testing.T) {
	originalSingleUser := isSingleUser
	t.Cleanup(func() { isSingleUser = originalSingleUser })

	tests := []struct {
		name       string
		singleUser bool
		chorus     bool
		want       string
	}{
		{"single-user", true, false, "https://example.com/tag:BlueNote"},
		{"multi-user", false, false, "https://example.com/blog/tag:BlueNote"},
		{"chorus", false, true, "https://example.com/read/t/BlueNote"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isSingleUser = tc.singleUser
			cfg := config.New()
			cfg.App.Host = "https://example.com"
			cfg.App.SingleUser = tc.singleUser
			cfg.App.Chorus = tc.chorus
			app := &App{cfg: cfg}
			post := newActivityHashtagTestPost(cfg.App.Host, "A #BlueNote article.\n\nMore content.")
			requireHashtags(t, marshalActivityObject(t, post.ActivityObject(app)).Tag, map[string]string{"#BlueNote": tc.want})
		})
	}
}

func TestActivityObjectWithoutHashtagsAndNoteType(t *testing.T) {
	cfg := config.New()
	cfg.App.Host = "https://example.com"
	app := &App{cfg: cfg}

	article := newActivityHashtagTestPost(cfg.App.Host, "An article without tags.\n\nMore content.")
	articleWire := marshalActivityObject(t, article.ActivityObject(app))
	if articleWire.Type != "Article" || len(articleWire.Tag) != 0 || len(articleWire.Preview.Tag) != 0 {
		t.Fatalf("unexpected hashtag-free Article: %+v", articleWire)
	}

	note := newActivityHashtagTestPost(cfg.App.Host, "A short #BlueNote note.")
	noteWire := marshalActivityObject(t, note.ActivityObject(app))
	if noteWire.Type != "Note" {
		t.Fatalf("type = %q, want Note", noteWire.Type)
	}
	requireHashtags(t, noteWire.Tag, map[string]string{"#BlueNote": "https://example.com/blog/tag:BlueNote"})
}

func TestActivityObjectPreservesMentionTags(t *testing.T) {
	app, _ := newTemplateTestApp(t, func(cfg *config.Config) {
		cfg.App.Host = "https://example.com"
	})
	actor := "https://remote.example/users/alice"
	if _, err := app.db.Exec(
		"INSERT INTO remoteusers (actor_id, inbox, shared_inbox, url, handle) VALUES (?, ?, ?, ?, ?)",
		actor, actor+"/inbox", "https://remote.example/inbox", actor, "alice@remote.example",
	); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}

	post := newActivityHashtagTestPost(app.cfg.App.Host, "Hello @alice@remote.example and #BlueNote.\n\nMore content.")
	tags := marshalActivityObject(t, post.ActivityObject(app)).Tag
	requireHashtags(t, tags, map[string]string{"#BlueNote": "https://example.com/blog/tag:BlueNote"})

	found := false
	for _, tag := range tags {
		if tag.Type == "Mention" && tag.Name == "@alice@remote.example" && tag.HRef == actor {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Mention tag missing after hashtag generation: %+v", tags)
	}
}

func TestActivityArticleFeaturesCoexist(t *testing.T) {
	app, _ := newTemplateTestApp(t, func(cfg *config.Config) {
		cfg.App.Host = "https://example.com"
	})
	actor := "https://remote.example/users/alice"
	if _, err := app.db.Exec(
		"INSERT INTO remoteusers (actor_id, inbox, shared_inbox, url, handle) VALUES (?, ?, ?, ?, ?)",
		actor, actor+"/inbox", "https://remote.example/inbox", actor, "alice@remote.example",
	); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}

	imageURL := "https://example.com/sky.png"
	post := newActivityHashtagTestPost(app.cfg.App.Host,
		"Hello **BlueNote** @alice@remote.example #BlueNote.\n\n![A blue sky]("+imageURL+")")
	post.extractData()
	object := marshalActivityObject(t, post.ActivityObject(app))

	if object.Type != "Article" || object.Summary == nil {
		t.Fatalf("expected Article with summary: %+v", object)
	}
	if strings.Contains(*object.Summary, "<") || *object.Summary != "Hello BlueNote @alice@remote.example #BlueNote. [...]" {
		t.Fatalf("summary = %q", *object.Summary)
	}
	if object.Preview.Content == "" || !strings.Contains(object.Content, "<strong>BlueNote</strong>") {
		t.Fatalf("Article content or preview missing: %+v", object)
	}
	requireHashtags(t, object.Tag, map[string]string{
		"#BlueNote": "https://example.com/blog/tag:BlueNote",
	})
	requireHashtags(t, object.Preview.Tag, map[string]string{
		"#BlueNote": "https://example.com/blog/tag:BlueNote",
	})

	foundMention := false
	for _, tag := range object.Tag {
		if tag.Type == "Mention" && tag.Name == "@alice@remote.example" && tag.HRef == actor {
			foundMention = true
		}
	}
	if !foundMention {
		t.Fatalf("Mention tag missing: %+v", object.Tag)
	}
	for location, attachments := range map[string][]activityAttachmentWire{
		"object":  object.Attachment,
		"preview": object.Preview.Attachment,
	} {
		if len(attachments) != 1 || attachments[0].Type != "Image" || attachments[0].URL != imageURL || attachments[0].Name != "A blue sky" {
			t.Fatalf("%s attachment = %+v", location, attachments)
		}
	}
}
