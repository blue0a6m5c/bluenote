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
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/writeas/impart"
)

const (
	// MySQL stores collection attribute values in VARCHAR(255). Keep the
	// serialized list within that existing cross-database contract so this
	// feature does not require a schema migration.
	maxVerificationLinksStoredLength = 255
	maxVerificationLinks             = 5
)

var verificationURLScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// parseVerificationLinks splits the newline-separated value stored in the
// existing verification_link collection attribute. It trims blank entries
// and removes exact duplicates while retaining their first occurrence.
func parseVerificationLinks(stored string) []string {
	links := make([]string, 0)
	seen := map[string]bool{}
	for _, line := range strings.Split(stored, "\n") {
		link := strings.TrimSpace(line)
		if link == "" || seen[link] {
			continue
		}
		seen[link] = true
		links = append(links, link)
	}
	return links
}

func serializeVerificationLinks(links []string) string {
	return strings.Join(parseVerificationLinks(strings.Join(links, "\n")), "\n")
}

// normalizeVerificationLinks validates and normalizes user-submitted rel=me
// links. A legacy single URL becomes a one-element list, while fediverse
// handles continue to resolve through the existing profile lookup path.
func normalizeVerificationLinks(app *App, submitted string) ([]string, error) {
	normalized := make([]string, 0)
	for _, link := range parseVerificationLinks(submitted) {
		var normalizedLink string
		var err error
		if strings.HasPrefix(link, "@") {
			if strings.Count(link, "@") != 2 {
				return nil, invalidVerificationLink(link)
			}
			normalizedLink, err = GetProfileURLFromHandle(app, link)
			if err != nil || normalizedLink == "" {
				return nil, impart.HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("Couldn't resolve verification handle %q.", link)}
			}
		} else {
			normalizedLink = link
		}

		normalizedLink, err = normalizeVerificationURL(normalizedLink)
		if err != nil {
			return nil, invalidVerificationLink(link)
		}
		normalized = append(normalized, normalizedLink)
	}

	// Normalize first so equivalent submitted forms do not consume multiple
	// slots or unnecessary storage.
	normalized = parseVerificationLinks(serializeVerificationLinks(normalized))
	if len(normalized) > maxVerificationLinks {
		return nil, impart.HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("Add no more than %d verification links.", maxVerificationLinks)}
	}
	stored := serializeVerificationLinks(normalized)
	if utf8.RuneCountInString(stored) > maxVerificationLinksStoredLength {
		return nil, impart.HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("Verification links must use at most %d characters in total.", maxVerificationLinksStoredLength)}
	}
	return normalized, nil
}

func normalizeVerificationURL(value string) (string, error) {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return "", fmt.Errorf("empty URL")
	}

	switch {
	case strings.HasPrefix(candidate, "//"):
		candidate = "https:" + candidate
	case !verificationURLScheme.MatchString(candidate):
		candidate = "https://" + candidate
	}

	u, err := url.Parse(candidate)
	if err != nil {
		return "", err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Hostname() == "" || u.User != nil {
		return "", fmt.Errorf("verification URL must be an absolute HTTP(S) URL without credentials")
	}
	return u.String(), nil
}

func invalidVerificationLink(value string) error {
	return impart.HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("Invalid verification link %q. Use an absolute HTTP(S) URL or a fediverse handle.", value)}
}
