package game

import (
	"log"

	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type CTF struct {
	ctfSpawnState
	*teamMode
	*flagMode
	*handlesPickups
}

func NewCTF(s Server, keepTeams bool) *CTF {
	good, evil := NewTeam("good"), NewTeam("evil")
	return &CTF{
		teamMode:       withTeams(s, false, keepTeams, good, evil),
		flagMode:       handlingFlags(newCTF(s, good, evil)),
		handlesPickups: handlingPickups(s),
	}
}

func (m *CTF) NeedsMapInfo() bool {
	return m.handlesPickups.NeedsMapInfo() || m.flagMode.NeedsMapInfo()
}

func (m *CTF) InitPlayer() []any {
	return append(m.handlesPickups.InitPlayer(), m.flagMode.InitPlayer()...)
}

func (m *CTF) HandlePacket(p *Player, packetType nmc.ID, pkt *protocol.Packet) bool {
	switch packetType {
	case nmc.InitFlags,
		nmc.TouchFlag,
		nmc.TryDropFlag:
		return m.flagMode.HandlePacket(p, packetType, pkt)
	case nmc.PickupList,
		nmc.PickupTry:
		return m.handlesPickups.HandlePacket(p, packetType, pkt)
	default:
		log.Println("received unrelated packet", packetType, pkt)
		return false
	}
}

func (m *CTF) HandleFrag(fragger *Player, victim *Player) {
	m.flagMode.HandleFrag(fragger, victim)
	m.teamMode.HandleFrag(fragger, victim)
}

func (m *CTF) Leave(p *Player) {
	m.flagMode.Leave(p)
	m.teamMode.Leave(p)
}

func (m *CTF) ChangeTeam(p *Player, newTeamName string, forced bool) {
	m.flagMode.ChangeTeam(p, newTeamName, forced)
	m.teamMode.ChangeTeam(p, newTeamName, forced)
}

func (m *CTF) Pause() {
	m.flagMode.Pause()
	m.handlesPickups.Pause()
}

func (m *CTF) Resume() {
	m.flagMode.Resume()
	m.handlesPickups.Resume()
}

func (m *CTF) Stop() {
	m.flagMode.Stop()
	m.handlesPickups.Stop()
}

func (m *CTF) CleanUp() {
	m.flagMode.CleanUp()
	m.handlesPickups.CleanUp()
}

func (*CTF) ID() gamemode.ID { return gamemode.CTF }

type EfficCTF struct {
	efficSpawnState
	*teamMode
	*flagMode
}

func NewEfficCTF(s Server, keepTeams bool) *EfficCTF {
	good, evil := NewTeam("good"), NewTeam("evil")
	return &EfficCTF{
		teamMode: withTeams(s, false, keepTeams, good, evil),
		flagMode: handlingFlags(newCTF(s, good, evil)),
	}
}

func (m *EfficCTF) HandleFrag(fragger *Player, victim *Player) {
	m.flagMode.HandleFrag(fragger, victim)
	m.teamMode.HandleFrag(fragger, victim)
}

func (m *EfficCTF) Leave(p *Player) {
	m.flagMode.Leave(p)
	m.teamMode.Leave(p)
}

func (m *EfficCTF) ChangeTeam(p *Player, newTeamName string, forced bool) {
	m.flagMode.ChangeTeam(p, newTeamName, forced)
	m.teamMode.ChangeTeam(p, newTeamName, forced)
}

func (*EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }

type InstaCTF struct {
	instaSpawnState
	*teamMode
	*flagMode
}

func NewInstaCTF(s Server, keepTeams bool) *InstaCTF {
	good, evil := NewTeam("good"), NewTeam("evil")
	return &InstaCTF{
		teamMode: withTeams(s, false, keepTeams, good, evil),
		flagMode: handlingFlags(newCTF(s, good, evil)),
	}
}

func (m *InstaCTF) HandleFrag(fragger *Player, victim *Player) {
	m.flagMode.HandleFrag(fragger, victim)
	m.teamMode.HandleFrag(fragger, victim)
}

func (m *InstaCTF) Leave(p *Player) {
	m.flagMode.Leave(p)
	m.teamMode.Leave(p)
}

func (m *InstaCTF) ChangeTeam(p *Player, newTeamName string, forced bool) {
	m.flagMode.ChangeTeam(p, newTeamName, forced)
	m.teamMode.ChangeTeam(p, newTeamName, forced)
}

func (*InstaCTF) ID() gamemode.ID { return gamemode.InstaCTF }
