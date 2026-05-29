package xtypes

import (
	urlpkg "net/url"
	"sort"
	"strings"
)

// RFC3986Query wraps url.Values and provides an Encode method that produces
// a query string encoded according to RFC 3986.
//
// The go standard url.Values.Encode() method uses application/x-www-form-urlencoded
// encoding, which differs from RFC 3986 in two ways:
//  1. Spaces are encoded as '+' instead of '%20'.
//  2. The unreserved character '~' is encoded as '%7E'.
//
// This type overrides the Encode method to apply RFC 3986 encoding, making it
// suitable for use in canonical request construction.
type RFC3986Query struct {
	urlpkg.Values
}

// Encode encodes the query parameters into a string following RFC 3986.
// Parameters are sorted by key, and each key-value pair is joined with '&'.
// If there are no parameters, an empty string is returned.
func (q *RFC3986Query) Encode() string {
	if q.Values == nil || len(q.Values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(q.Values))
	for k := range q.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		for _, v := range q.Values[k] {
			parts = append(parts, encodeComponent(k)+"="+encodeComponent(v))
		}
	}
	return strings.Join(parts, "&")
}

// encodeComponent applies RFC 3986 percent-encoding to the input string.
//
// Go's standard url.QueryEscape encodes according to the
// "application/x-www-form-urlencoded" convention, which differs from
// RFC 3986 in two important ways:
//  1. It encodes a space as "+" instead of "%20".
//  2. It encodes the unreserved character "~" as "%7E", whereas
//     RFC 3986 requires "~" to be left as-is.
func encodeComponent(s string) string {
	encoded := urlpkg.QueryEscape(s)
	encoded = strings.Replace(encoded, "+", "%20", -1)
	encoded = strings.Replace(encoded, "%7E", "~", -1)
	return encoded
}
