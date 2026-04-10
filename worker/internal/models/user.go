package models

type UserInfo struct {
	Username string `json:"username,omitempty"`
	Domain   string `json:"domain,omitempty"`
	SID      string `json:"sid,omitempty"`
}
