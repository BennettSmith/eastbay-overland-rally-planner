package domain

import "testing"

func TestNormalizeHumanName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"Alice", "Alice"},
		{"  Alice  ", "Alice"},
		{"Alice   Smith", "Alice Smith"},
		{"\tAlice\tSmith\n", "Alice Smith"},
		{"", ""},
		{"   \n\t  ", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeHumanName(tc.in); got != tc.want {
				t.Fatalf("NormalizeHumanName(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}
