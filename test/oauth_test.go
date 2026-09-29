package api_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// RFC 7636 appendix B test vector.
func TestPKCEChallengeS256(t *testing.T) {
	assert.Equal(t,
		"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		api.PKCEChallengeS256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
	)
}

func TestGeneratePKCEVerifier(t *testing.T) {
	v1, err := api.GeneratePKCEVerifier()
	require.NoError(t, err)
	v2, err := api.GeneratePKCEVerifier()
	require.NoError(t, err)

	assert.Len(t, v1, 43) // 32 bytes, unpadded base64url
	assert.NotEqual(t, v1, v2)
	// RFC 7636 unreserved charset (base64url alphabet is a subset).
	const allowed = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for _, r := range v1 {
		assert.Contains(t, allowed, string(r))
	}
}

// TestOAuthDiscovery verifies the unauthenticated discovery chain against the live harness: ring config → Hub URL + YouTrack service ID → OAuth endpoints.
func TestOAuthDiscovery(t *testing.T) {
	requireE2EEnv(t)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	ring, err := api.GetRingConfigContext(t.Context(), baseURL)
	require.NoError(t, err)
	require.NotEmpty(t, ring.ServiceID, "ring.serviceId must be discoverable unauthenticated")

	hubURL := strings.TrimRight(ring.URL, "/")
	if hubURL == "" {
		hubURL = strings.TrimRight(baseURL, "/") + "/hub"
	}

	eps, err := api.DiscoverOAuthEndpointsContext(t.Context(), hubURL)
	require.NoError(t, err)
	assert.NotEmpty(t, eps.AuthorizationEndpoint)
	assert.NotEmpty(t, eps.TokenEndpoint)
}

// TestInstanceLogin_FallsBackWithoutOAuthClient checks that with no "localhost-auth" client, `instance login` pre-flight falls back to the paste-token flow (fed via stdin); the full browser flow can't run headlessly.
func TestInstanceLogin_FallsBackWithoutOAuthClient(t *testing.T) {
	requireE2EEnv(t)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	tmpHome := t.TempDir()

	out := runCmdWithEnv(t, token+"\n",
		[]string{"HOME=" + tmpHome},
		"--plain", "instance", "login", baseURL,
		"--alias", "ssotest",
		"--no-browser",
		"--insecure", // the E2E harness serves plain http on loopback
		"--timeout", "10s",
	)

	assert.Contains(t, out, "Token ready:")

	data, err := os.ReadFile(tmpHome + "/.youtrack-cli-config.json")
	require.NoError(t, err)
	assert.Contains(t, string(data), `"ssotest"`)
}
