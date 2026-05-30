package validator_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/duakc/mt/common/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertValidError(t *testing.T, err error, wantField string, wantValue any) {
	t.Helper()
	var ve *validator.ValidError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, wantField, ve.Field)
	assert.Equal(t, wantValue, ve.Value)
}

func TestNonEmpty(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.NonEmpty(1, "n"))
	require.NoError(t, validator.NonEmpty("x", "s"))

	assertValidError(t, validator.NonEmpty(0, "n"), "n", 0)
	assertValidError(t, validator.NonEmpty("", "s"), "s", "")
}

func TestNonEmptySlice(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.NonEmptySlice([]int{1}, "xs"))
	require.Error(t, validator.NonEmptySlice([]int{}, "xs"))
	require.Error(t, validator.NonEmptySlice[[]int](nil, "xs"))
}

func TestNonEmptyMap(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.NonEmptyMap(map[string]int{"a": 1}, "m"))
	require.Error(t, validator.NonEmptyMap(map[string]int{}, "m"))
	require.Error(t, validator.NonEmptyMap[map[string]int](nil, "m"))
}

func TestNotNil(t *testing.T) {
	t.Parallel()

	v := 1
	require.NoError(t, validator.NotNil(&v, "p"))
	require.Error(t, validator.NotNil[int](nil, "p"))
}

func TestGreaterThan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v, b int
		ok   bool
	}{
		{"greater", 2, 1, true},
		{"equal", 1, 1, false},
		{"less", 0, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.GreaterThan(tc.v, tc.b, "Num")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "Num", tc.v)
		})
	}
}

func TestGreaterOrEqual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v, b int
		ok   bool
	}{
		{"greater", 2, 1, true},
		{"equal", 1, 1, true},
		{"less", 0, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.GreaterOrEqual(tc.v, tc.b, "Num")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "Num", tc.v)
		})
	}
}

func TestLessThan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v, b int
		ok   bool
	}{
		{"less", 0, 1, true},
		{"equal", 1, 1, false},
		{"greater", 2, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.LessThan(tc.v, tc.b, "Num")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "Num", tc.v)
		})
	}
}

func TestLessOrEqual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v, b int
		ok   bool
	}{
		{"less", 0, 1, true},
		{"equal", 1, 1, true},
		{"greater", 2, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.LessOrEqual(tc.v, tc.b, "Num")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "Num", tc.v)
		})
	}
}

func TestBetween(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		v, min, max int
		ok          bool
	}{
		{"in range", 5, 1, 10, true},
		{"at lower bound", 1, 1, 10, true},
		{"at upper bound", 10, 1, 10, true},
		{"below", 0, 1, 10, false},
		{"above", 11, 1, 10, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.Between(tc.v, tc.min, tc.max, "Num")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "Num", tc.v)
		})
	}
}

func TestEqualWith(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.EqualWith(1, 1, "n"))
	require.NoError(t, validator.EqualWith("x", "x", "s"))
	assertValidError(t, validator.EqualWith(1, 2, "n"), "n", 1)
	assertValidError(t, validator.EqualWith("x", "y", "s"), "s", "x")
}

func TestNotEqualWith(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.NotEqualWith(1, 2, "n"))
	assertValidError(t, validator.NotEqualWith(1, 1, "n"), "n", 1)
}

func TestContains(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.Contains(2, []int{1, 2, 3}, "n"))
	assertValidError(t, validator.Contains(4, []int{1, 2, 3}, "n"), "n", 4)
	assertValidError(t, validator.Contains(1, []int{}, "n"), "n", 1)
}

func TestNotContains(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.NotContains(4, []int{1, 2, 3}, "n"))
	require.NoError(t, validator.NotContains(1, []int{}, "n"))
	assertValidError(t, validator.NotContains(2, []int{1, 2, 3}, "n"), "n", 2)
}

func TestMinRune(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    string
		n    int
		ok   bool
	}{
		{"equal", "abc", 3, true},
		{"greater", "abcd", 3, true},
		{"less", "ab", 3, false},
		{"empty fails on positive bound", "", 1, false},
		{"counts runes not bytes", "中文你", 3, true}, // 3 runes, 9 bytes
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.MinRune(tc.s, tc.n, "S")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "S", tc.s)
		})
	}
}

func TestMaxRune(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    string
		n    int
		ok   bool
	}{
		{"equal", "abc", 3, true},
		{"less", "ab", 3, true},
		{"greater", "abcd", 3, false},
		{"counts runes not bytes", "中文你", 3, true}, // 3 runes, would fail on bytes
		{"runes over bound", "中文你好", 3, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validator.MaxRune(tc.s, tc.n, "S")
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assertValidError(t, err, "S", tc.s)
		})
	}
}

func TestStringStartsWith(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.StringStartsWith("hello world", "hello", "s"))
	require.NoError(t, validator.StringStartsWith("abc", "", "s")) // empty prefix matches
	assertValidError(t, validator.StringStartsWith("hello", "world", "s"), "s", "hello")
}

func TestStringEndsWith(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.StringEndsWith("hello world", "world", "s"))
	require.NoError(t, validator.StringEndsWith("abc", "", "s"))
	assertValidError(t, validator.StringEndsWith("hello", "world", "s"), "s", "hello")
}

func TestStringContains(t *testing.T) {
	t.Parallel()

	require.NoError(t, validator.StringContains("hello world", "lo wo", "s"))
	require.NoError(t, validator.StringContains("abc", "", "s"))
	assertValidError(t, validator.StringContains("hello", "xyz", "s"), "s", "hello")
}

func TestImplements(t *testing.T) {
	t.Parallel()

	t.Run("error interface satisfied", func(t *testing.T) {
		t.Parallel()
		var v any = errors.New("boom")
		require.NoError(t, validator.Implements[error](v, "e"))
	})

	t.Run("error interface not satisfied", func(t *testing.T) {
		t.Parallel()
		var v any = 42
		err := validator.Implements[error](v, "e")
		assertValidError(t, err, "e", 42)
	})

	t.Run("stringer satisfied", func(t *testing.T) {
		t.Parallel()
		var v any = stringerImpl{}
		require.NoError(t, validator.Implements[fmt.Stringer](v, "s"))
	})
}

type stringerImpl struct{}

func (stringerImpl) String() string { return "ok" }

func TestErrorsJoinAccumulation(t *testing.T) {
	t.Parallel()

	err := errors.Join(
		validator.GreaterThan(0, 0, "Num"),
		validator.NonEmpty("", "Str"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Num")
	assert.Contains(t, err.Error(), "Str")
}

func TestValidErrorUnwrap(t *testing.T) {
	t.Parallel()

	err := validator.GreaterThan(3, 5, "Age")
	var ve *validator.ValidError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "Age", ve.Field)
	assert.Equal(t, 3, ve.Value)
	assert.NotNil(t, ve.Unwrap())
}
