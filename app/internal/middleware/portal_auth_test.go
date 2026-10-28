package middleware

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssuePortalConsumerToken_RoundTrip(t *testing.T) {
	secret := "test-secret"
	token, err := IssuePortalConsumerToken(secret, "phone-hash-abc", time.Hour)
	require.NoError(t, err)

	payload, err := VerifyPortalToken(secret, token)
	require.NoError(t, err)
	assert.Equal(t, "consumer", payload.Kind)
	assert.Equal(t, "phone-hash-abc", payload.IdentityKey)
	assert.Empty(t, payload.SessionID, "consumer 令牌不绑定单会话")
}

func TestIssuePortalConsumerToken_RequiresIdentity(t *testing.T) {
	_, err := IssuePortalConsumerToken("secret", "", time.Hour)
	require.Error(t, err)
}

func TestIssuePortalConsumerToken_TamperedSignatureRejected(t *testing.T) {
	secret := "test-secret"
	token, err := IssuePortalConsumerToken(secret, "phone-hash-abc", time.Hour)
	require.NoError(t, err)

	// 篡改 payload 但保留原签名
	parts := strings.SplitN(strings.TrimPrefix(token, portalTokenPrefix), ".", 2)
	require.Len(t, parts, 2)
	tampered := portalTokenPrefix + "eyJraW5kIjoiY29uc3VtZXIifQ." + parts[1]
	_, err = VerifyPortalToken(secret, tampered)
	require.Error(t, err)
}

func TestIssuePortalConsumerToken_ExpiredRejected(t *testing.T) {
	token, err := IssuePortalConsumerToken("secret", "phone-hash-abc", -time.Minute)
	require.NoError(t, err)
	_, err = VerifyPortalToken("secret", token)
	require.Error(t, err)
}

func TestIssuePortalAnonymousToken_StillTiesSession(t *testing.T) {
	token, err := IssuePortalAnonymousToken("secret", "sess-1", time.Hour)
	require.NoError(t, err)
	payload, err := VerifyPortalToken("secret", token)
	require.NoError(t, err)
	assert.Equal(t, "anonymous", payload.Kind)
	assert.Equal(t, "sess-1", payload.SessionID)
	assert.Empty(t, payload.IdentityKey)
}
