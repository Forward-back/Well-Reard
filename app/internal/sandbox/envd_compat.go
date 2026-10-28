package sandbox

import (
	"context"
	"encoding/base64"
)

// maintenanceFilesystemKey marks a request as a maintenance filesystem
// operation. Only the server's skill installer may select the maintenance
// identity. Ordinary filesystem operations use the same account as
// shell/script exec.
type maintenanceFilesystemKey struct{}

func withMaintenanceFilesystem(ctx context.Context) context.Context {
	return context.WithValue(ctx, maintenanceFilesystemKey{}, true)
}

// remoteFileUser returns the sandbox user for filesystem operations:
// the maintenance identity when marked, otherwise the default exec user.
func remoteFileUser(ctx context.Context) string {
	if maintenance, _ := ctx.Value(maintenanceFilesystemKey{}).(bool); maintenance {
		return "root"
	}
	return DefaultSandboxExecUser
}

// basicAuthorizationFor builds the Basic auth header value for the given user
// (envd-compatible data-plane authentication).
func basicAuthorizationFor(user string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"))
}
