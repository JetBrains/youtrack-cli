package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_RawPaginateEmptyCollection(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	path := "/api/issues?query=" + url.QueryEscape("project: "+proj.ShortName)
	for _, flags := range [][]string{nil, {"--paginate", "--page-size", "1"}} {
		args := append([]string{"raw", path, "--fields", "id", "--json"}, flags...)
		out := runCmd(t, args...)
		assert.JSONEq(t, `[]`, out, "empty collections must remain arrays with or without pagination")
	}
}
