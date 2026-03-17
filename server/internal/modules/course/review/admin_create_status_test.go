package review

import (
	"os"
	"strings"
	"testing"
)

func TestCreateTeacherUsesCreatedStatusCode(t *testing.T) {
	source, err := os.ReadFile("handler_teacher_admin.go")
	if err != nil {
		t.Fatalf("failed to read handler_teacher_admin.go: %v", err)
	}

	if !strings.Contains(string(source), "response.Created(c, teacher)") {
		t.Fatalf("expected CreateTeacher handler to use response.Created(c, teacher)")
	}
}

func TestCreateSensitiveWordUsesCreatedStatusCode(t *testing.T) {
	source, err := os.ReadFile("handler_sensitive_word_admin.go")
	if err != nil {
		t.Fatalf("failed to read handler_sensitive_word_admin.go: %v", err)
	}

	if !strings.Contains(string(source), "response.Created(c, w)") {
		t.Fatalf("expected CreateSensitiveWord handler to use response.Created(c, w)")
	}
}
