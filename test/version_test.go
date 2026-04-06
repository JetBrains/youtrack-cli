package api_test

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCmd_Version(t *testing.T) {
	cmd := exec.Command(binaryPath, "--version")

	out, err := cmd.Output()
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("youtrack %s (%s, %s)\n", binaryVersion, binaryCommit, binaryDate), string(out))
}
