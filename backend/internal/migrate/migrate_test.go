package migrate

import (
	"testing"

	"github.com/google/uuid"
)

func TestGoogleUUIDAvailable(t *testing.T) {
	t.Helper()
	if uuid.New() == uuid.Nil {
		t.Fatal("expected non-nil uuid")
	}
}
