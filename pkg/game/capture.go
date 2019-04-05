package game

type CaptureMode interface {
	Teamed
	Bases(*Team) []int32
}

type captureMode struct {
	teamed
	bases map[*Team][]int32
}

func (cm *captureMode) Bases(t *Team) []int32 {
	if bases, ok := cm.bases[t]; ok {
		return bases
	}
	return nil
}
