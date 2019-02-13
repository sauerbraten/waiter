package main

import (
	"log"
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type flag struct {
	id           int32
	team         int32
	owner        *Client
	version      int32
	dropTime     time.Time
	dropLocation *geom.Vector
	//spawnIndex    int32
	spawnLocation *geom.Vector
}

type flagMode struct {
	flagsInitialized bool
}

type ctfMode struct {
	teamMode
	flagMode
	good flag
	evil flag
}

func (ctf *ctfMode) flagByID(id int32) (*flag, bool) {
	switch id {
	case 0:
		return &ctf.good, true
	case 1:
		return &ctf.evil, true
	default:
		return nil, false
	}
}

func (ctf *ctfMode) teamByFlag(f *flag) string {
	switch f.team {
	case 1:
		return "good"
	case 2:
		return "evil"
	default:
		return ""
	}
}

func (ctf *ctfMode) HandlePacket(client *Client, packetType nmc.ID, p *protocol.Packet) bool {
	switch packetType {
	case nmc.InitFlags:
		ctf.initFlags(ctf.parseFlags(p))

	case nmc.TakeFlag:
		id, ok := p.GetInt()
		if !ok {
			log.Println("could not read flag ID from takeflag packet (packet too short):", p)
			break
		}
		version, ok := p.GetInt()
		if !ok {
			log.Println("could not read flag version from takeflag packet (packet too short):", p)
			break
		}
		ctf.takeFlag(client, id, version)

	case nmc.TryDropFlag:
		ctf.dropFlag(client)

	default:
		return false
	}

	return true
}

func (*ctfMode) parseFlags(p *protocol.Packet) (f1, f2 *flag) {
	numFlags, ok := p.GetInt()
	if !ok {
		log.Println("could not read number of flags from initflags packet (packet too short):", p)
		return
	}
	if numFlags != 2 {
		log.Println("received", numFlags, "flags in CTF mode")
		return
	}

	f1, f2 = &flag{}, &flag{}
	for id, flag := range []*flag{f1, f2} {
		flag.id = int32(id)

		flag.team, ok = p.GetInt()
		if !ok {
			log.Println("could not read flag team from initflags packet (packet too short):", p)
			return
		}

		flag.spawnLocation, ok = parseVector(p)
		if !ok {
			log.Println("could not read flag spawn location from initflags packet (packet too short):", p)
			return
		}
		flag.spawnLocation = flag.spawnLocation.Mul(1 / geom.DMF)
	}

	return
}

func (ctf *ctfMode) initFlags(f1, f2 *flag) {
	if ctf.flagsInitialized || f1 == nil || f2 == nil {
		return
	}

	for _, f := range []*flag{f1, f2} {
		flag, ok := ctf.flagByID(f.id)
		if !ok {
			log.Printf("received invalid flag ID '%d' in CTF mode", f.id)
			continue
		}

		*flag = *f
	}

	ctf.flagsInitialized = true
}

func (ctf *ctfMode) takeFlag(client *Client, id int32, version int32) {
	if !ctf.flagsInitialized {
		return
	}

	flag, ok := ctf.flagByID(id)
	if !ok {
		log.Printf("received invalid flag id '%d' in CTF mode", id)
		return
	}

	if flag.owner != nil || flag.version != version || client.GameState.State != playerstate.Alive {
		return
	}

	team := ctf.teamByFlag(flag)
	log.Println(client.Team, team)

	if client.Team.Name != team {
		// player stealing enemy flag
		flag.version++
		s.Clients.Broadcast(nil, nmc.TakeFlag, client.CN, id, flag.version)
		flag.owner = client
	} else if !flag.dropTime.IsZero() {
		// player touches her own, dropped flag
		ctf.returnFlag(flag)
		flag.version++
		s.Clients.Broadcast(nil, nmc.ReturnFlag, client.CN, flag.id, flag.version)
		return
	} else {
		// player touches her own, spawned flag
		enemyFlag, ok := ctf.flagByID(1 - id)
		if !ok {
			log.Println("could not get other flag in CTF mode")
			return
		}

		if enemyFlag.owner == client {
			ctf.returnFlag(enemyFlag)
			client.GameState.Flags++
			ctf.Teams[team].Score++
			flag.version++
			enemyFlag.version++
			s.Clients.Broadcast(nil, nmc.ScoreFlag, client.CN, enemyFlag.id, enemyFlag.version, flag.id, flag.version, 0, flag.team, ctf.Teams[team].Score, client.GameState.Flags)
			if ctf.Teams[team].Score >= 10 {
				// todo: trigger intermission
			}
		}
	}
}

func (ctf *ctfMode) dropFlag(client *Client) {
	if !ctf.flagsInitialized {
		return
	}

	var f *flag
	switch client.Team.Name {
	case "good":
		f = &ctf.evil
	case "evil":
		f = &ctf.good
	default:
		return
	}

	if f.owner != client {
		return
	}

	// TODO: use client position
}

func (ctf *ctfMode) returnFlag(f *flag) {
	f.dropTime = time.Time{}
	f.owner = nil
}

func (ctf *ctfMode) Init() {
	ctf.teamMode.Init()

	ctf.Teams["good"] = NewTeam("good")
	ctf.Teams["evil"] = NewTeam("evil")

	s.Clients.ForEach(func(c *Client) { ctf.Join(c) })
}

func (ctf *ctfMode) Join(client *Client) {
	ctf.teamMode.Join(client)
	// TODO: init flags for client
}

func (ctf *ctfMode) Leave(client *Client) {
	// TODO: drop flag
	ctf.teamMode.Leave(client)
}

func (ctf *ctfMode) CanSpawn(c *Client) bool {
	return !c.Joined || c.GameState.LastDeath.IsZero() || time.Since(c.GameState.LastDeath) > 5*time.Second
}

type EfficCTF struct {
	ctfMode
}

func NewEfficCTF() *EfficCTF {
	return &EfficCTF{
		ctfMode{
			teamMode: teamMode{
				Teams: map[string]*Team{},
			},
		},
	}
}

func (*EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }