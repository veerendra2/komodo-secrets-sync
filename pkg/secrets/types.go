package secrets

import "time"

// BitwardenSyncResponse represents the response from Bitwarden Secrets Sync
type BitwardenSyncResponse struct {
	HasChanges bool                  `json:"hasChanges"`
	Secrets    []BitwardenSecretData `json:"secrets"`
}

// BitwardenSecretData represents a single secret from Bitwarden
type BitwardenSecretData struct {
	CreationDate   time.Time `json:"creationDate"`
	ID             string    `json:"id"`
	Key            string    `json:"key"`
	Note           string    `json:"note"`
	OrganizationID string    `json:"organizationId"`
	ProjectID      string    `json:"projectId"`
	RevisionDate   time.Time `json:"revisionDate"`
	Value          string    `json:"value"`
}

type Secret struct {
	Key   string
	Note  string
	Value string
}

type Dump struct {
	Secrets []Secret
}
