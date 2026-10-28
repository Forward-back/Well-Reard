package service

import "errors"

var (
	ErrOrgNotFound       = errors.New("organization not found")
	ErrTenantNotInOrg    = errors.New("tenant is not a member of this organization")
	ErrInvalidRole       = errors.New("invalid organization role")
	ErrOrgMemberNotFound = errors.New("organization member not found")
)
