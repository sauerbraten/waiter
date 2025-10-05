package game

type Capture struct {
	teamMode
	bases map[*Team][]int32
}

func (m *Capture) Bases(t *Team) []int32 {
	if bases, ok := m.bases[t]; ok {
		return bases
	}
	return nil
}

func (m *Capture) CanSpawn(*Player) bool { return true }
