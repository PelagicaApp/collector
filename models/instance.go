package models

type PingRequest struct {
	InstanceID string `json:"instance_id"`
	Version    string `json:"version"`
	Token      string `json:"token"`
}

type StatsResponse struct {
	TotalInstalls   int `json:"total_installs"`
	ActiveInstances int `json:"active_instances"`
}
