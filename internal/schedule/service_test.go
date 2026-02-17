package schedule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Kxiandaoyan/Memoh-v2/internal/automation"
)

type mockTriggerer struct {
	called  bool
	botID   string
	payload TriggerPayload
	token   string
}

func (m *mockTriggerer) TriggerSchedule(_ context.Context, botID string, payload TriggerPayload, token string) error {
	m.called = true
	m.botID = botID
	m.payload = payload
	m.token = token
	return nil
}

func TestGenerateTriggerToken(t *testing.T) {
	secret := "test-secret-key-for-schedule"
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	tok, err := automation.GenerateTriggerToken(userID, secret, automation.DefaultTriggerTokenTTL)
	if err != nil {
		t.Fatalf("GenerateTriggerToken returned error: %v", err)
	}
	if !strings.HasPrefix(tok, "Bearer ") {
		t.Fatalf("expected Bearer prefix, got: %s", tok)
	}

	raw := strings.TrimPrefix(tok, "Bearer ")
	parsed, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse JWT: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	if sub, _ := claims["sub"].(string); sub != userID {
		t.Errorf("expected sub=%s, got=%s", userID, sub)
	}
	if uid, _ := claims["user_id"].(string); uid != userID {
		t.Errorf("expected user_id=%s, got=%s", userID, uid)
	}
	exp, _ := claims["exp"].(float64)
	if exp == 0 {
		t.Fatal("expected non-zero exp")
	}
	expTime := time.Unix(int64(exp), 0)
	if expTime.Before(time.Now().Add(9 * time.Minute)) {
		t.Error("token expires too soon")
	}
}

func TestGenerateTriggerToken_EmptySecret(t *testing.T) {
	_, err := automation.GenerateTriggerToken("user-123", "", automation.DefaultTriggerTokenTTL)
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestGenerateTriggerToken_EmptyUserID(t *testing.T) {
	_, err := automation.GenerateTriggerToken("", "some-secret", automation.DefaultTriggerTokenTTL)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}
