package matchmake

import (
	"encoding/json"
	"testing"

	"github.com/bkohler93/game-backend/internal/message"
	"github.com/google/uuid"
)

func TestJsonMarshallingRequest(t *testing.T) {
	jsonBytes := []byte("{\"name\": \"JohnnyRocket\"}")
	var req message.MatchmakingRequest
	err := json.Unmarshal(jsonBytes, &req)
	if err != nil {
		t.Errorf("error unmarshalling match request - %v", err)
	}
	if req.UserId.UUID() != uuid.Nil {
		t.Errorf("userId does not equal uuid.Nil. req.UserId=%v", req.UserId)
	}
	if req.MatchedWith.UUID() != uuid.Nil {
		t.Errorf("userId does not equal uuid.Nil. req.MatchedWith=%v", req.MatchedWith)
	}
}
