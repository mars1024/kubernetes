package sigma

type EventMessage struct {
	UpdateTime  string
	ContainerId string
	EventType   EventType
	EventData   EventData
}

type EventData map[string]string

type EventType string

const (
	EventOomKill  = "oomkill"
	EventCoreDump = "coredump"
)
