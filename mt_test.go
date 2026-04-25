package mt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyValue(t *testing.T) {
	cases := []struct {
		in    string
		key   string
		val   string
		found bool
	}{
		{"t=0", "t", "0", true},
		{"t=", "", "", false},
		{"t", "", "", false},
		{"t t=0", "t t", "0", true},
		{"t_t==0", "t_t", "=0", true},
		{`t\=1`, `t\`, "1", true},
		{"测=试", "测", "试", true},
		{"テスト=テスト", "テスト", "テスト", true},
		{"시험=시험", "시험", "시험", true},
	}
	for _, cc := range cases {
		key, val, found := KeyValue(cc.in)
		assert.Equal(t, cc.key, key)
		assert.Equal(t, cc.val, val)
		assert.Equal(t, cc.found, found)
	}
}

func TestKeyValueMulti(t *testing.T) {
	cases := []struct {
		in    string
		key   string
		val   string
		found bool
	}{
		{"t=0", "t", "0", true},
		{"t=", "", "", false},
		{"t", "", "", false},
		{"t t=0", "t t", "0", true},
		// diff
		{"t_t==0", "t_t", "0", true},
		{`t\=1`, `t\`, "1", true},
		{`t\=\=\1`, `t\`, `\=\1`, true},
		{`t\==\=\1`, `t\`, `\=\1`, true},
		{"测=试", "测", "试", true},
		{"测==试", "测", "试", true},
		{"テスト=テスト", "テスト", "テスト", true},
		{"시험=시험", "시험", "시험", true},
	}
	for _, cc := range cases {
		key, val, found := KeyValueMulti(cc.in)
		assert.Equal(t, cc.key, key)
		assert.Equal(t, cc.val, val)
		assert.Equal(t, cc.found, found)
	}
}
