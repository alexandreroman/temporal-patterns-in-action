package events

import "testing"

func TestSubject(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		workflow string
		category string
		want     string
	}{
		{"saga progress", "saga", "wf-123", CategoryProgress, "patterns.saga.wf-123.progress"},
		{"saga business", "saga", "wf-abc", CategoryBusiness, "patterns.saga.wf-abc.business"},
		{"empty workflow", "saga", "", CategoryProgress, "patterns.saga..progress"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Subject(tc.pattern, tc.workflow, tc.category); got != tc.want {
				t.Errorf("Subject() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCategoryOf(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{TypeWorkflowStarted, CategoryProgress},
		{TypeStepFailed, CategoryProgress},
		{TypeCompensationCompleted, CategoryProgress},
		{"saga.car.reserved", CategoryBusiness},
		{"", ""},
		{"nodot", CategoryBusiness},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := CategoryOf(tc.in); got != tc.want {
				t.Errorf("CategoryOf(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
