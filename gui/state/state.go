package state

type AppState struct {
	Subsidy     int
	TargetBits  int
	MempoolSize int
}

func NewAppState() *AppState {
	return &AppState{
		Subsidy:     10,
		TargetBits:  16,
		MempoolSize: 10,
	}
}
