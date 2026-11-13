package utils

import "testing"

func TestSQLUtilEscapeLike(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc", "abc"},
		{"a_b", `a\_b`},
		{"100%", `100\%`},
		{`a\b`, `a\\b`},
		{`\%`, `\\\%`},
	}

	for _, c := range cases {
		if got := SQLUtil.EscapeLike(c.in); got != c.want {
			t.Errorf("EscapeLike(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSQLUtilLikeContains(t *testing.T) {
	if got := SQLUtil.LikeContains("a_b"); got != `%a\_b%` {
		t.Errorf("LikeContains = %q", got)
	}
}
