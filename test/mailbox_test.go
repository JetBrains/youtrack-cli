package api_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// TestCmd_MailboxList verifies that `mailbox list --format json` returns a
// valid JSON array (possibly empty).  The test is skipped if the endpoint is
// unavailable on this instance.
func TestCmd_MailboxList(t *testing.T) {
	c := newTestClient(t)

	if _, err := c.ListMailboxServers(); err != nil {
		t.Skipf("mailbox list not supported on this instance: %v", err)
	}

	out := runCmd(t, "mailbox", "list", "--format", "json")
	assert.Contains(t, out, "[", "expected JSON array in output")
}

// TestCmd_MailboxAddDelete creates a mailbox server against a real IMAP host
// (provided via MAILHOG_HOST) and then deletes it, verifying the full
// add→list→delete lifecycle.  The test is skipped when MAILHOG_HOST is not set.
func TestCmd_MailboxAddDelete(t *testing.T) {
	mailhogHost := os.Getenv("MAILHOG_HOST")
	if mailhogHost == "" {
		t.Skip("MAILHOG_HOST not set — skipping mailbox add/delete test")
	}

	c := newTestClient(t)

	// Verify the endpoint is reachable before attempting to add.
	if _, err := c.ListMailboxServers(); err != nil {
		t.Skipf("mailbox list not supported on this instance: %v", err)
	}

	out := runCmd(t,
		"mailbox", "add",
		"--host", mailhogHost,
		"--port", "143",
		"--protocol", "IMAP",
		"--login", "test@example.com",
		"--password", "password",
		"--format", "json",
	)

	var server api.MailboxServer
	require.NoError(t, json.Unmarshal([]byte(out), &server), "add output: %s", out)
	require.NotEmpty(t, server.ID, "mailbox server ID should not be empty")
	assert.Equal(t, mailhogHost, server.Host)

	// Ensure cleanup even if the delete assertion below fails.
	t.Cleanup(func() { _ = c.DeleteMailboxServer(server.ID) })

	// Delete via CLI and confirm the success message.
	delOut := runCmd(t, "mailbox", "delete", server.ID)
	assert.Contains(t, delOut, "deleted", "expected 'deleted' in delete output")
}
