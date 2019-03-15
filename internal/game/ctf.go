package game

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
	owner         *Player
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
	s Server
	timedMode
	teamMode
	flagMode
	good         flag
	evil         flag
	flagsByIndex map[int32]*flag
}

func newCTFMode(s Server, keepTeams bool) ctfMode {
	return ctfMode{
		s:            s,
		timedMode:    newTimedMode(s),
		teamMode:     newTeamMode(s, false, keepTeams, "good", "evil"),
		flagsByIndex: map[int32]*flag{},
	}
}

func (ctf *ctfMode) Pause(p *Player) {
	if ctf.good.pendingReset != nil {
		ctf.good.pendingReset.Pause()
	}
	if ctf.evil.pendingReset != nil {
		ctf.evil.pendingReset.Pause()
	}
	ctf.timedMode.Pause(p)
}

func (ctf *ctfMode) Resume(p *Player) {
	if ctf.good.pendingReset != nil {
		ctf.good.pendingReset.Start()
	}
	if ctf.evil.pendingReset != nil {
		ctf.evil.pendingReset.Start()
	}
	ctf.timedMode.Resume(p)
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

func (ctf *ctfMode) HandlePacket(p *Player, packetType nmc.ID, pkt *protocol.Packet) bool {
	switch packetType {
	case nmc.InitFlags:
		ctf.initFlags(pkt)

	case nmc.TakeFlag:
		i, ok := pkt.GetInt()
		if !ok {
			log.Println("could not read flag ID from takeflag packet (packet too short):", pkt)
			break
		}
		version, ok := pkt.GetInt()
		if !ok {
			log.Println("could not read flag version from takeflag packet (packet too short):", pkt)
			break
		}
		ctf.touchFlag(p, i, version)

	case nmc.TryDropFlag:
		ctf.dropFlag(p)

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

func (ctf *ctfMode) touchFlag(p *Player, i int32, version int32) {
	if !ctf.flagsInitialized {
		return
	}

	flag := ctf.flagsByIndex[i]
	if flag == nil {
		log.Printf("received invalid flag index '%d' in CTF mode", i)
		return
	}

	if flag.owner != nil || flag.version != version || p.State != playerstate.Alive {
		return
	}

	team := ctf.teamByFlag(flag)

	if p.Team.Name != team {
		// player stealing enemy flag
		ctf.takeFlag(p, flag)
	} else if !flag.dropTime.IsZero() {
		// player touches her own, dropped flag
		flag.pendingReset.Stop()
		ctf.returnFlag(flag)
		ctf.s.Broadcast(nmc.ReturnFlag, p.CN, flag.index, flag.version)
		return
	} else {
		// player touches her own flag at its base
		enemyFlag := ctf.flagsByIndex[1-flag.index]
		if enemyFlag == nil {
			log.Println("could not get other flag in CTF mode")
			return
		}

		if enemyFlag.owner == p {
			ctf.returnFlag(enemyFlag)
			p.Flags++
			ctf.teams[team].Score++
			flag.version++
			ctf.s.Broadcast(nmc.ScoreFlag, p.CN, enemyFlag.index, enemyFlag.version, flag.index, flag.version, 0, flag.team, ctf.teams[team].Score, p.Flags)
			if ctf.teams[team].Score >= 10 {
				ctf.s.Intermission()
			}
		}
	}
}

func (ctf *ctfMode) takeFlag(p *Player, f *flag) {
	// cancel reset
	if f.pendingReset != nil {
		f.pendingReset.Stop()
		f.pendingReset = nil
	}

	f.version++
	ctf.s.Broadcast(nmc.TakeFlag, p.CN, f.index, f.version)
	f.owner = p
}

func (ctf *ctfMode) dropFlag(p *Player) {
	if !ctf.flagsInitialized {
		return
	}

	var f *flag
	switch p.Team.Name {
	case "good":
		f = &ctf.evil
	case "evil":
		f = &ctf.good
	default:
		return
	}

	if f.owner != p {
		return
	}

	f.dropLocation = p.Position
	f.dropTime = time.Now()
	f.owner = nil
	f.version++

	ctf.s.Broadcast(nmc.DropFlag, p.CN, f.index, f.version, f.dropLocation.Mul(geom.DMF))
	f.pendingReset = timer.AfterFunc(10*time.Second, func() {
		ctf.returnFlag(f)
		ctf.s.Broadcast(nmc.ResetFlag, f.index, f.version, 0, f.team, ctf.teams[ctf.teamByFlag(f)].Score)
	})
	f.pendingReset.Start()
}

func (ctf *ctfMode) returnFlag(f *flag) {
	f.dropTime = time.Time{}
	f.owner = nil
	f.version++
}

func (ctf *ctfMode) NeedMapInfo() bool { return !ctf.flagsInitialized }

func (ctf *ctfMode) Init(p *Player) (nmc.ID, []interface{}) {
	typ, q := nmc.InitFlags, []interface{}{
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

	return typ, q
}

func (ctf *ctfMode) Leave(p *Player) {
	ctf.dropFlag(p)
	ctf.teamMode.Leave(p)
}

func (ctf *ctfMode) CanSpawn(p *Player) bool {
	return p.LastDeath.IsZero() || time.Since(p.LastDeath) > 5*time.Second
}

func (ctf *ctfMode) HandleFrag(actor, victim *Player) {
	ctf.dropFlag(victim)
	ctf.teamMode.HandleFrag(actor, victim)
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
	_ Mode      = &EfficCTF{}
	_ TeamMode  = &EfficCTF{}
	_ TimedMode = &EfficCTF{}
)

func NewEfficCTF(s Server, keepTeams bool) *EfficCTF {
	var ectf *EfficCTF
	ectf = &EfficCTF{
		ctfMode: newCTFMode(s, keepTeams),
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
	_ Mode      = &InstaCTF{}
	_ TeamMode  = &InstaCTF{}
	_ TimedMode = &InstaCTF{}
)

func NewInstaCTF(s Server, keepTeams bool) *InstaCTF {
	var ictf *InstaCTF
	ictf = &InstaCTF{
		ctfMode: newCTFMode(s, keepTeams),
	}
	return ictf
}

func (*InstaCTF) ID() gamemode.ID { return gamemode.InstaCTF }

func parseVector(p *protocol.Packet) (*geom.Vector, bool) {
	xyz := [3]float64{}
	for i := range xyz {
		coord, ok := p.GetInt()
		if !ok {
			log.Printf("could not read %s coordinate from packet: %v", string("xzy"[i]), p)
			return nil, false
		}
		xyz[i] = float64(coord)
	}
	return geom.NewVector(xyz[0], xyz[1], xyz[2]), true
}
