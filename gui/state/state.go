package state

type AppState struct {
	Subsidy    int
	TargetBits int
}

func NewAppState() *AppState {
	return &AppState{
		Subsidy:    10,
		TargetBits: 16,
	}
}
