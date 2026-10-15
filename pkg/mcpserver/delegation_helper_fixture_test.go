package mcpserver

import (
	"os"
	"testing"
)

func os_getenv(k string) string { return os.Getenv(k) }

// emitHelperFixture writes a new helper token to a file so the old public
// module and the enterprise gateway at b28c999c can be asked to verify it.
func emitHelperFixture(t *testing.T, path, token string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
}
