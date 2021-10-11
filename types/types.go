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
