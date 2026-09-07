/*
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
	"os"
	"strings"
	"testing"
)

func TestBlueNoteAssetIncludedOnceInPublicTemplates(t *testing.T) {
	for _, path := range []string{"templates/collection.tmpl", "templates/collection-post.tmpl"} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		template := string(contents)
		if got := strings.Count(template, `<script src="/js/bluenote.js"></script>`); got != 1 {
			t.Errorf("%s contains %d BlueNote asset tags, want exactly one", path, got)
		}
		if strings.Contains(template, "/local/blue-note.js") {
			t.Errorf("%s still references the unmanaged /local/blue-note.js asset", path)
		}
		csrfIndex := strings.Index(template, `<script src="/js/csrf.js"></script>`)
		assetIndex := strings.Index(template, `<script src="/js/bluenote.js"></script>`)
		if csrfIndex == -1 || assetIndex == -1 || csrfIndex > assetIndex {
			t.Errorf("%s loads BlueNote asset before csrf.js", path)
		}
	}
}
