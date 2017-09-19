package server

import "github.com/sauerbraten/waiter/internal/server/config"

type Server struct {
	State  *State
	Config *config.Config
}
