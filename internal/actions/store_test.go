package actions

import (
	"errors"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

func TestStoreUpdateStatus(t *testing.T) {
	store := NewStore()
	action := types.ActionRequest{
		ID:        "action-1",
		Type:      "message.send",
		Risk:      types.ActionRiskMedium,
		Status:    types.ActionStatusConfirmed,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	store.Save(action)

	updated, err := store.UpdateStatus(action.ID, types.ActionStatusDenied)
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if updated.Status != types.ActionStatusDenied {
		t.Fatalf("expected denied, got %s", updated.Status)
	}

	if _, err := store.UpdateStatus("missing", types.ActionStatusDenied); !errors.Is(err, ErrActionNotFound) {
		t.Fatalf("expected action not found, got %v", err)
	}
}
