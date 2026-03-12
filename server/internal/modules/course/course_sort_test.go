package course

import "testing"

func TestNormalizeCourseSort(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "default empty", in: "", want: CourseSortName},
		{name: "name", in: CourseSortName, want: CourseSortName},
		{name: "credits", in: CourseSortCredits, want: CourseSortCredits},
		{name: "reviewCount", in: CourseSortReviewCount, want: CourseSortReviewCount},
		{name: "invalid", in: "drop table", want: CourseSortName},
	}

	for _, tc := range cases {
		if got := normalizeCourseSort(tc.in); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}
