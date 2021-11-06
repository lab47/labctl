package types

import "time"

type AccountInfo struct {
	Email       string `json:"email"`
	Namespace   string `json:"namespace"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password,omitempty"`
}

type AccountResponse struct {
	Token string `json:"token"`
}

type RepoDetails struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	TotalTags int       `json:"num_tags"`
}

type NamespaceInfo struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type NamespaceDetails struct {
	Name   string        `json:"name"`
	Credit string        `json:"credit"`
	Repos  []RepoDetails `json:"repositories"`
}

type ListNamespaces struct {
	Namespaces []NamespaceDetails `json:"namespaces"`
}

type CreditAddRequest struct {
	Namespace string `json:"namespace"`
	Credits   int64  `json:"credits"`
	LocalPort int    `json:"local_port"`
}

type CreditAddResponse struct {
	URL string `json:"url"`
}

type MachineAccountCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Write       bool   `json:"enable_write"`
}

type MachineAccountCreateResponse struct {
	Token string `json:"tokn"`
}

type RepoSettingsApply struct {
	Public *bool `json:"public"`
}

type PersonalTokenRequest struct {
	JWT       bool   `json:"jwt"`
	PublicKey []byte `json:"public_key,omitempty"`
	TTL       int64  `json:"ttl"`
}

type PersonalTokenResponse struct {
	JWT  string `json:"jwt,omitempty"`
	X509 string `json:"x509,omitempty"`
}
