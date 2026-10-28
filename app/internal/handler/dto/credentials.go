package dto

// CredentialFieldMetadata describes whether a secret field is populated
// without disclosing the value.
type CredentialFieldMetadata struct {
	Configured bool `json:"configured"`
}

type CredentialsResponse struct {
	Fields map[string]CredentialFieldMetadata `json:"fields"`
}
