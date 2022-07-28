//go:build installer

package installer_bundle

import "embed"

//go:embed bundle
var Bundle embed.FS
