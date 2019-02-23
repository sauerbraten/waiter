package main

import (
	"log"
	"time"

	"github.com/ivahaev/timer"

	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type flag struct {
	index         int32
	team          int32
	owner         *Client
	version       int32
	dropTime      time.Time
	dropLocation  *geom.Vector
	spawnLocation *geom.Vector
	pendingReset  *timer.Timer
}

type flagMode struct {
	flagsInitialized bool
}

type ctfMode struct {
	timedMode
	teamMode
	flagMode
	good         flag
	evil         flag
	flagsByIndex map[int32]*flag
}

func newCTFMode() ctfMode {
	return ctfMode{
		timedMode:    newTimedMode(),
		teamMode:     newTeamMode(false, "good", "evil"),
		flagsByIndex: map[int32]*flag{},
	}
}

func (ctf *ctfMode) Pause(c *Client) {
	if ctf.good.pendingReset != nil {
		ctf.good.pendingReset.Pause()
	}
	if ctf.evil.pendingReset != nil {
		ctf.evil.pendingReset.Pause()
	}
	ctf.timedMode.Pause(c)
}

func (ctf *ctfMode) Resume(c *Client) {
	if ctf.good.pendingReset != nil {
		ctf.good.pendingReset.Start()
	}
	if ctf.evil.pendingReset != nil {
		ctf.evil.pendingReset.Start()
	}
	ctf.timedMode.Resume(c)
}

func (ctf *ctfMode) flagByTeamID(team int32) *flag {
	switch team {
	case 1:
		return &ctf.good
	case 2:
		return &ctf.evil
	default:
		return nil
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
		ctf.initFlags(p)

	case nmc.TakeFlag:
		i, ok := p.GetInt()
		if !ok {
			log.Println("could not read flag ID from takeflag packet (packet too short):", p)
			break
		}
		version, ok := p.GetInt()
		if !ok {
			log.Println("could not read flag version from takeflag packet (packet too short):", p)
			break
		}
		ctf.touchFlag(client, i, version)

	case nmc.TryDropFlag:
		ctf.dropFlag(client)

	default:
		return false
	}

	return true
}

func (ctf *ctfMode) initFlags(p *protocol.Packet) (f1, f2 *flag) {
	numFlags, ok := p.GetInt()
	if !ok {
		log.Println("could not read number of flags from initflags packet (packet too short):", p)
		return
	}
	if numFlags != 2 {
		log.Println("received", numFlags, "flags in CTF mode")
		return
	}

	for i := int32(0); i < numFlags; i++ {
		team, ok := p.GetInt()
		if !ok {
			log.Println("could not read flag team from initflags packet (packet too short):", p)
			return
		}

		spawnLocation, ok := parseVector(p)
		if !ok {
			log.Println("could not read flag spawn location from initflags packet (packet too short):", p)
			return
		}
		spawnLocation = spawnLocation.Mul(1 / geom.DMF)

		flag := ctf.flagByTeamID(team)
		if flag == nil {
			log.Printf("received invalid team ID '%d' in CTF mode", team)
			continue
		}

		flag.index, flag.team, flag.spawnLocation = i, team, spawnLocation
		ctf.flagsByIndex[i] = flag
	}

	ctf.flagsInitialized = true

	return
}

func (ctf *ctfMode) touchFlag(client *Client, i int32, version int32) {
	if !ctf.flagsInitialized {
		return
	}

	flag := ctf.flagsByIndex[i]
	if flag == nil {
		log.Printf("received invalid flag index '%d' in CTF mode", i)
		return
	}

	if flag.owner != nil || flag.version != version || client.GameState.State != playerstate.Alive {
		return
	}

	team := ctf.teamByFlag(flag)

	if client.Team.Name != team {
		// player stealing enemy flag
		ctf.takeFlag(client, flag)
	} else if !flag.dropTime.IsZero() {
		// player touches her own, dropped flag
		ctf.returnFlag(flag)
		flag.version++
		s.Clients.Broadcast(nil, nmc.ReturnFlag, client.CN, flag.index, flag.version)
		return
	} else {
		// player touches her own flag at its base
		enemyFlag := ctf.flagsByIndex[1-flag.index]
		if enemyFlag == nil {
			log.Println("could not get other flag in CTF mode")
			return
		}

		if enemyFlag.owner == client {
			ctf.returnFlag(enemyFlag)
			client.GameState.Flags++
			ctf.teams[team].Score++
			flag.version++
			enemyFlag.version++
			s.Clients.Broadcast(nil, nmc.ScoreFlag, client.CN, enemyFlag.index, enemyFlag.version, flag.index, flag.version, 0, flag.team, ctf.teams[team].Score, client.GameState.Flags)
			if ctf.teams[team].Score >= 10 {
				s.Intermission()
			}
		}
	}
}

func (ctf *ctfMode) takeFlag(client *Client, f *flag) {
	// cancel reset
	if f.pendingReset != nil {
		f.pendingReset.Stop()
		f.pendingReset = nil
	}

	f.version++
	s.Clients.Broadcast(nil, nmc.TakeFlag, client.CN, f.index, f.version)
	f.owner = client
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

	f.dropLocation = client.CurrentPos
	f.dropTime = time.Now()
	f.owner = nil
	f.version++

	s.Clients.Broadcast(nil, nmc.DropFlag, client.CN, f.index, f.version, f.dropLocation.Mul(geom.DMF))
	f.pendingReset = timer.AfterFunc(10*time.Second, func() {
		ctf.returnFlag(f)
		f.version++
		s.Clients.Broadcast(nil, nmc.ResetFlag, f.index, f.version, 0, f.team, ctf.teams[ctf.teamByFlag(f)].Score)
	})
	f.pendingReset.Start()
}

func (ctf *ctfMode) returnFlag(f *flag) {
	f.dropTime = time.Time{}
	f.owner = nil
}

func (ctf *ctfMode) NeedMapInfo() bool { return !ctf.flagsInitialized }

func (ctf *ctfMode) Init(client *Client) {
	q := []interface{}{
		nmc.InitFlags,
		ctf.teams["good"].Score,
		ctf.teams["evil"].Score,
	}

	if ctf.flagsInitialized {
		q = append(q, len(ctf.flagsByIndex))
		for _, i := range []int32{0, 1} {
			f := ctf.flagsByIndex[i]
			if f == nil {
				continue
			}

			var ownerCN int32 = -1
			if f.owner != nil {
				ownerCN = int32(f.owner.CN)
			}
			q = append(q, f.version, 0, ownerCN, 0)
			if f.owner == nil {
				dropped := !f.dropTime.IsZero()
				q = append(q, dropped)
				if dropped {
					q = append(q, f.dropLocation.Mul(geom.DMF))
				}
			}
		}
	} else {
		q = append(q, 0)
	}

	client.Send(q...)
}

func (ctf *ctfMode) Leave(client *Client) {
	ctf.dropFlag(client)
	ctf.teamMode.Leave(client)
}

func (ctf *ctfMode) CanSpawn(c *Client) bool {
	return c.GameState.LastDeath.IsZero() || time.Since(c.GameState.LastDeath) > 5*time.Second
}

func (ctf *ctfMode) HandleDeath(_, victim *Client) {
	ctf.dropFlag(victim)
}

func (ctf *ctfMode) CleanUp() {
	if ctf.good.pendingReset != nil {
		ctf.good.pendingReset.Stop()
	}
	if ctf.evil.pendingReset != nil {
		ctf.evil.pendingReset.Stop()
	}
	ctf.timedMode.CleanUp()
}

type EfficCTF struct {
	casualMode
	efficMode
	ctfMode
}

// assert interface implementations at compile time
var (
	_ GameMode = &EfficCTF{}
	_ TeamMode = &EfficCTF{}
)

func NewEfficCTF() *EfficCTF {
	var ectf *EfficCTF
	ectf = &EfficCTF{
		ctfMode: newCTFMode(),
	}
	return ectf
}

func (*EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }

type InstaCTF struct {
	casualMode
	instaMode
	ctfMode
}

// assert interface implementations at compile time
var (
	_ GameMode = &InstaCTF{}
	_ TeamMode = &InstaCTF{}
)

func NewInstaCTF() *InstaCTF {
	var ictf *InstaCTF
	ictf = &InstaCTF{
		ctfMode: newCTFMode(),
	}
	return ictf
}

func (*InstaCTF) ID() gamemode.ID { return gamemode.InstaCTF }
