// internal/provider/provider.go
package provider

import "context"

// Provider define a interface para interagir com serviços como GitHub, GitLab, etc.
type Provider interface {
	CreateRelease(ctx context.Context, tag, changelog string) (string, error)
}
