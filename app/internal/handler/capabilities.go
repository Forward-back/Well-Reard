package handler

import (
	"gaze/internal/capabilities"
	"gaze/internal/types"
)

// capsForMembership derives UI capability keys from a synthesized membership.
// Membership projections built by the user service already carry Caps; this
// helper covers handler-side projections (auto-setup, invite registration).
func capsForMembership(role types.TenantRole, departmentRole types.DepartmentRole) []string {
	return capabilities.MustDefault().CapsFor(string(role), string(departmentRole), false)
}

func mergeCapabilityKeys(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, keys := range [][]string{base, extra} {
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, key)
		}
	}
	return out
}
