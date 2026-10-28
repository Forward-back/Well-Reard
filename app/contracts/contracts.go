// Package contracts exposes runtime-loaded contract files shared with the
// frontend. capabilities.json is the single source of truth for capability
// keys and the role-to-capability matrix.
package contracts

import _ "embed"

// CapabilitiesJSON is the embedded contents of capabilities.json.
//
//go:embed capabilities.json
var CapabilitiesJSON []byte
