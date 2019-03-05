package main

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
)

type Config struct {
	ListenAddress string `json:"listen_address"`
	ListenPort    int    `json:"listen_port"`

	MasterServerAddress     string        `json:"master_server_address"`
	StatsServerAddress      string        `json:"stats_server_address"`
	StatsServerAuthDomain   string        `json:"stats_server_auth_domain"`
	FallbackGameMode        gamemode.ID   `json:"fallback_game_mode"`
	ServerDescription       string        `json:"server_description"`
	MaxClients              int           `json:"max_clients"`
	SendClientIPsViaExtinfo bool          `json:"send_client_ips_via_extinfo"`
	MessageOfTheDay         string        `json:"message_of_the_day"`
	GameDurationInMinutes   time.Duration `json:"game_duration_in_minutes"`
	PrimaryAuthDomain       string        `json:"primary_auth_domain"`
}
