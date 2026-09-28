package redis

import (
	"context"
	"testing"
	"time"
)

func TestRelationStatusCache(t *testing.T) {
	ctx := context.Background()
	userID := uint(time.Now().UnixNano())
	toUserID := uint(80001)
	defer DeleteRelationStatus(ctx, userID, toUserID)

	if err := SetRelationStatus(ctx, userID, toUserID, true); err != nil {
		t.Fatalf("set relation cache failed: %v", err)
	}
	following, found, err := GetRelationStatus(ctx, userID, toUserID)
	if err != nil {
		t.Fatalf("get relation cache failed: %v", err)
	}
	if !found || !following {
		t.Fatalf("unexpected relation cache: found=%v following=%v", found, following)
	}
	_, reverseFound, err := GetRelationStatus(ctx, toUserID, userID)
	if err != nil {
		t.Fatalf("get reverse relation cache failed: %v", err)
	}
	if reverseFound {
		t.Fatal("relation cache must be directional")
	}
}
