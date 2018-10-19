package main

import "time"

const (
	GLOBAL_AUTH_DOMAIN = ""
)

type Config struct {
	ListenAddress string `json:"listen_address"`
	ListenPort    int    `json:"listen_port"`

	MasterServerAddress string `json:"master_server_address"`
	MasterServerPort    int    `json:"master_server_port"`

	ServerDescription       string        `json:"server_description"`
	MaxClients              int           `json:"max_clients"`
	SendClientIPsViaExtinfo bool          `json:"send_client_ips_via_extinfo"`
	MessageOfTheDay         string        `json:"message_of_the_day"`
	GameDuration            time.Duration `json:"game_duration_in_minutes"`
	PrimaryAuthDomain       string        `json:"primary_auth_domain"`
}
