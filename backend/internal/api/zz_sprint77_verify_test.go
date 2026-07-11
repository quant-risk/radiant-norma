package api

import "testing"

func TestSprint77VersionFix(t *testing.T) {
	// The bug: string comparison would say "3.9" > "3.10"
	cases := []struct {
		name string
		versions []string
		want string
	}{
		{"3.9 vs 3.10", []string{"3.9", "3.10"}, "3.10"},
		{"3.10 vs 3.9 reversed", []string{"3.10", "3.9"}, "3.10"},
		{"1.0 vs 10.0", []string{"1.0", "10.0"}, "10.0"},
		{"1.0 vs 2.0", []string{"1.0", "2.0"}, "2.0"},
		{"three versions mixed", []string{"3.9", "3.10", "3.11"}, "3.11"},
		{"single", []string{"3.0"}, "3.0"},
		{"empty", []string{}, ""},
		{"patch version 3.10.1 vs 3.9.5", []string{"3.9.5", "3.10.1"}, "3.10.1"},
		{"same version 3.10 vs 3.10", []string{"3.10", "3.10"}, "3.10"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := latestVersion(tc.versions)
			if got != tc.want {
				t.Errorf("latestVersion(%v) = %q, want %q", tc.versions, got, tc.want)
			}
		})
	}

	// compareVersion direct checks
	compCases := []struct {
		v1, v2 string
		want int // +1 if v1>v2, -1 if v1<v2, 0 equal
	}{
		{"3.10", "3.9", 1},
		{"3.9", "3.10", -1},
		{"10.0", "1.0", 1},
		{"1.0", "10.0", -1},
		{"3.10", "3.10", 0},
		{"3.11", "3.9", 1},
		{"3.9", "3.11", -1},
	}
	for _, tc := range compCases {
		t.Run("compare_"+tc.v1+"_"+tc.v2, func(t *testing.T) {
			got := compareVersion(tc.v1, tc.v2)
			// normalize sign
			sign := 0
			if got > 0 { sign = 1 } else if got < 0 { sign = -1 }
			if sign != tc.want {
				t.Errorf("compareVersion(%q,%q) = %d, want sign %d", tc.v1, tc.v2, got, tc.want)
			}
		})
	}

	// splitVersion edge cases
	svCases := []struct {
		v string
		want [3]int
	}{
		{"3.10", [3]int{3, 10, 0}},
		{"3.10.5", [3]int{3, 10, 5}},
		{"", [3]int{0, 0, 0}},
		{"3", [3]int{3, 0, 0}},
		{"3.", [3]int{3, 0, 0}}, // empty segment -> 0
		{".10", [3]int{0, 10, 0}},
		{"abc.def", [3]int{0, 0, 0}}, // non-numeric -> Atoi fails -> 0
	}
	for _, tc := range svCases {
		t.Run("split_"+tc.v, func(t *testing.T) {
			got := splitVersion(tc.v)
			if got != tc.want {
				t.Errorf("splitVersion(%q) = %v, want %v", tc.v, got, tc.want)
			}
		})
	}
}
