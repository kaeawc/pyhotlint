package suppress

import "testing"

func TestParse_Empty(t *testing.T) {
	s := Parse([]byte("x = 1\n"))
	if s.IsSuppressed("any-rule", 1) {
		t.Fatal("empty file should suppress nothing")
	}
}

func TestParse_LineIgnoreAll(t *testing.T) {
	src := "x = 1\ntime.sleep(1)  # pyhotlint: ignore\ny = 2\n"
	s := Parse([]byte(src))
	if !s.IsSuppressed("any-rule", 2) {
		t.Fatal("line 2 should suppress every rule")
	}
	if s.IsSuppressed("any-rule", 1) {
		t.Fatal("line 1 should not be suppressed")
	}
	if s.IsSuppressed("any-rule", 3) {
		t.Fatal("line 3 should not be suppressed")
	}
}

func TestParse_LineIgnoreSpecific(t *testing.T) {
	src := "time.sleep(1)  # pyhotlint: ignore[sync-io-in-async-fn]\n"
	s := Parse([]byte(src))
	if !s.IsSuppressed("sync-io-in-async-fn", 1) {
		t.Fatal("named rule should be suppressed on line 1")
	}
	if s.IsSuppressed("other-rule", 1) {
		t.Fatal("other rule should not be suppressed")
	}
}

func TestParse_LineIgnoreMultiple(t *testing.T) {
	src := "do_thing()  # pyhotlint: ignore[rule-a, rule-b]\n"
	s := Parse([]byte(src))
	if !s.IsSuppressed("rule-a", 1) || !s.IsSuppressed("rule-b", 1) {
		t.Fatal("both listed rules should be suppressed")
	}
	if s.IsSuppressed("rule-c", 1) {
		t.Fatal("unlisted rule should not be suppressed")
	}
}

func TestParse_FileIgnoreAll(t *testing.T) {
	src := "# pyhotlint: ignore-file\nimport time\ntime.sleep(1)\n"
	s := Parse([]byte(src))
	if !s.IsSuppressed("any-rule", 1) || !s.IsSuppressed("any-rule", 3) {
		t.Fatal("file-level ignore should cover every line")
	}
}

func TestParse_FileIgnoreSpecific(t *testing.T) {
	src := "# pyhotlint: ignore-file[sync-io-in-async-fn]\nimport time\ntime.sleep(1)\n"
	s := Parse([]byte(src))
	if !s.IsSuppressed("sync-io-in-async-fn", 3) {
		t.Fatal("named rule should be suppressed throughout file")
	}
	if s.IsSuppressed("other-rule", 3) {
		t.Fatal("unlisted rule should still fire")
	}
}

func TestParse_ToleratesWhitespace(t *testing.T) {
	cases := []string{
		"#pyhotlint:ignore",
		"#pyhotlint: ignore",
		"# pyhotlint :ignore",
		"# pyhotlint  :  ignore[rule-a]",
		"#  pyhotlint:ignore[ rule-a , rule-b ]",
	}
	for _, c := range cases {
		s := Parse([]byte(c + "\n"))
		if !s.IsSuppressed("rule-a", 1) {
			t.Fatalf("expected suppression for %q", c)
		}
	}
}

func TestParse_NotAPragma(t *testing.T) {
	src := "x = 1  # ordinary comment\ny = 2  # noqa: E501\nz = 3  # pyhotlinter: ignore\n"
	s := Parse([]byte(src))
	for line := 1; line <= 3; line++ {
		if s.IsSuppressed("rule-a", line) {
			t.Fatalf("line %d should not be a pragma", line)
		}
	}
}

func TestIsSuppressed_NilSafe(t *testing.T) {
	var s *Set
	if s.IsSuppressed("rule", 1) {
		t.Fatal("nil Set should return false")
	}
}
