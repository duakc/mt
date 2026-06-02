package xtypes

import (
	urlpkg "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRFC3986Query_Encode(t *testing.T) {
	tests := []struct {
		name   string
		values urlpkg.Values
		want   string
	}{
		{
			name:   "empty",
			values: urlpkg.Values{},
			want:   "",
		},
		{
			// Original String: "Hello World ~*"
			// RFC 3986:         Hello%20World%20~%2A
			// Form-urlencoded:  Hello+World+%7E*
			name:   "space tilde asterisk",
			values: urlpkg.Values{"k": []string{"Hello World ~*"}},
			want:   "k=Hello%20World%20~%2A",
		},
		{
			name:   "space is %20 not +",
			values: urlpkg.Values{"k": []string{"a b"}},
			want:   "k=a%20b",
		},
		{
			name:   "tilde is not encoded",
			values: urlpkg.Values{"k": []string{"~"}},
			want:   "k=~",
		},
		{
			name:   "asterisk is encoded as %2A",
			values: urlpkg.Values{"k": []string{"*"}},
			want:   "k=%2A",
		},
		{
			name:   "unreserved characters are not encoded",
			values: urlpkg.Values{"k": []string{"AZaz09-_.~"}},
			want:   "k=AZaz09-_.~",
		},
		{
			name:   "reserved sub-delims are encoded",
			values: urlpkg.Values{"k": []string{"!$&'()*+,;="}},
			want:   "k=%21%24%26%27%28%29%2A%2B%2C%3B%3D",
		},
		{
			name:   "gen-delims are encoded",
			values: urlpkg.Values{"k": []string{":/?#[]@"}},
			want:   "k=%3A%2F%3F%23%5B%5D%40",
		},
		{
			name:   "keys are also RFC 3986 encoded",
			values: urlpkg.Values{"a *~": []string{"v"}},
			want:   "a%20%2A~=v",
		},
		{
			name: "keys are sorted",
			values: urlpkg.Values{
				"b": []string{"2"},
				"a": []string{"1"},
				"c": []string{"3"},
			},
			want: "a=1&b=2&c=3",
		},
		{
			name:   "multiple values for one key preserve order",
			values: urlpkg.Values{"k": []string{"v1", "v2"}},
			want:   "k=v1&k=v2",
		},
		{
			name:   "empty value",
			values: urlpkg.Values{"k": []string{""}},
			want:   "k=",
		},
		{
			name:   "unicode is percent-encoded as UTF-8",
			values: urlpkg.Values{"k": []string{"你好"}},
			want:   "k=%E4%BD%A0%E5%A5%BD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &RFC3986Query{Values: tt.values}
			got := q.Encode()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRFC3986Query_Encode_DeterministicOrder asserts that Encode produces a
// stable ordering across runs — required for canonical request construction
// (e.g. AWS SigV4), where any ordering drift breaks the signature.
func TestRFC3986Query_Encode_DeterministicOrder(t *testing.T) {
	values := urlpkg.Values{
		"z": []string{"1"},
		"a": []string{"2"},
		"m": []string{"3"},
		"b": []string{"4"},
	}
	q := &RFC3986Query{Values: values}
	first := q.Encode()
	for range 50 {
		assert.Equal(t, first, q.Encode(), "Encode must be deterministic across iterations")
	}
	assert.Equal(t, "a=2&b=4&m=3&z=1", first)
}

func TestRFC3986Path_Encode(t *testing.T) {
	tests := []struct {
		name string
		in   RFC3986Path
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "single slash",
			in:   "/",
			want: "/",
		},
		{
			// Original String per RFC 3986 rules:
			//   space -> %20, '*' -> %2A, '~' -> ~
			name: "space tilde asterisk",
			in:   "/Hello World ~*",
			want: "/Hello%20World%20~%2A",
		},
		{
			name: "slashes preserved as separators",
			in:   "/a/b/c",
			want: "/a/b/c",
		},
		{
			name: "leading and trailing slashes preserved",
			in:   "/a/b/",
			want: "/a/b/",
		},
		{
			name: "consecutive slashes preserved",
			in:   "/a//b",
			want: "/a//b",
		},
		{
			// url.PathEscape would keep '+' literal, so wrapping with
			// PathEscape would mangle '+' if we then replaced '+' -> '%20'.
			// Per RFC 3986 '+' is a sub-delim and must be percent-encoded.
			name: "plus is encoded as %2B (not kept literal)",
			in:   "/a+b",
			want: "/a%2Bb",
		},
		{
			name: "unreserved characters kept",
			in:   "/AZaz09-_.~",
			want: "/AZaz09-_.~",
		},
		{
			name: "reserved sub-delims encoded",
			in:   "/!$&'()*+,;=",
			want: "/%21%24%26%27%28%29%2A%2B%2C%3B%3D",
		},
		{
			name: "gen-delims (except '/') encoded",
			in:   "/:?#[]@",
			want: "/%3A%3F%23%5B%5D%40",
		},
		{
			name: "unicode encoded as UTF-8",
			in:   "/你好",
			want: "/%E4%BD%A0%E5%A5%BD",
		},
		{
			name: "relative path without leading slash",
			in:   "a b/c d",
			want: "a%20b/c%20d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Encode()
			assert.Equal(t, tt.want, got)
		})
	}
}
