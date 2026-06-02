package xtypes

import (
	urlpkg "net/url"
	"sort"
	"strings"
)

// RFC3986Query wraps url.Values and provides an Encode method that produces
// a query string encoded according to RFC 3986 (§2.2, §2.3).
//
// The two encodings differ on a handful of characters. For example, the
// string "Hello World ~*" serializes as:
//
//	RFC 3986:                       Hello%20World%20~%2A
//	application/x-www-form-urlencoded: Hello+World+%7E*
//
// Differences relative to url.Values.Encode (which follows
// application/x-www-form-urlencoded, per the WHATWG URL spec):
//  1. Space is '%20', not '+'.
//  2. '*' is '%2A' (reserved sub-delim per RFC 3986 §2.2), not literal '*'.
//  3. '~' stays as '~' (unreserved per §2.3); form-urlencoded escapes it to '%7E'.
//
// This type overrides Encode to apply RFC 3986 encoding, making it suitable
// for canonical request construction (e.g. AWS SigV4).
type RFC3986Query struct {
	urlpkg.Values
}

// Encode encodes the query parameters into a string following RFC 3986.
// Parameters are sorted by key, and each key-value pair is joined with '&'.
// If there are no parameters, an empty string is returned.
func (q *RFC3986Query) Encode() string {
	if len(q.Values) == 0 {
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
// It builds on url.QueryEscape and patches the two differences from
// RFC 3986 that matter in practice:
//  1. Space is encoded as "+"; RFC 3986 requires "%20".
//  2. As a defensive measure, "%7E" is unescaped back to "~". Modern Go
//     (see net/url shouldEscape: "case '-', '_', '.', '~'" returns false)
//     already leaves "~" alone, so this is a no-op today but guards
//     against regressions or older toolchains.
//
// "*" is already encoded as "%2A" by url.QueryEscape in encodeQueryComponent
// mode, matching RFC 3986 §2.2 (sub-delim).
func encodeComponent(s string) string {
	encoded := urlpkg.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
