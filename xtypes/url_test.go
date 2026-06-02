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
