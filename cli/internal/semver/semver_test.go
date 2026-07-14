package semver

import "testing"

func TestLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.2.0", "1.2.1", true},
		{"1.2.1", "1.2.1", false},
		{"1.2.1", "1.2.0", false},
		{"1.9.0", "1.10.0", true}, // numeric, not lexicographic
		{"v1.0.0", "v2.0.0", true},
		{"0.8.1", "0.8.1", false},
		{"1.0.0-beta", "1.0.0", true},  // a pre-release is older than its final release → upgrades
		{"1.0.0", "1.0.0-beta", false}, // and the final is never "older" than its own pre-release
		{"2.0.0-alpha.1", "2.0.0", true},
		{"1.0.0-beta", "1.0.1", true}, // numeric triple still wins first

		// self-update guard cases:
		{"2.3.1", "v2.3.1", false}, // v-prefix asymmetry: stamped "2.3.1" vs tag "v2.3.1" are equal
		{"v2.3.1", "2.3.1", false},
		{"2.3.1", "v2.3.2", true},  // installed behind the latest release → upgrade
		{"2.3.2", "v2.3.1", false}, // installed ahead of the resolved latest → never downgrade
		{"dev", "2.3.1", true},     // an un-stamped build parses to [0,0,0] → reads as older
		{"2.3.1", "dev", false},    // ...and a real release is never older than "dev"
	}
	for _, c := range cases {
		if got := Less(c.a, c.b); got != c.want {
			t.Errorf("Less(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		v    string
		want [3]int
	}{
		{"2.3.1", [3]int{2, 3, 1}},
		{"v2.3.1", [3]int{2, 3, 1}},
		{" v2.3.1 ", [3]int{2, 3, 1}},
		{"2.3.1-alpha.2", [3]int{2, 3, 1}},
		{"2.3.1+build.7", [3]int{2, 3, 1}},
		{"dev", [3]int{0, 0, 0}},
		{"1.2", [3]int{1, 2, 0}}, // missing patch → 0
	}
	for _, c := range cases {
		if got := Parse(c.v); got != c.want {
			t.Errorf("Parse(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestHasPrerelease(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"2.3.1", false},
		{"v2.3.1", false},
		{"2.3.1-alpha.2", true},
		{"2.3.1+build.7", false}, // build metadata is not a pre-release
		{"2.3.1-rc.1+build.7", true},
	}
	for _, c := range cases {
		if got := HasPrerelease(c.v); got != c.want {
			t.Errorf("HasPrerelease(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}
