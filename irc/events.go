package irc

import "time"

type Event interface{}

type ErrorEvent struct {
	Severity Severity
	Code     string
	Message  string
}

type RegisteredEvent struct{}

type SelfNickEvent struct {
	FormerNick string
}

type UserNickEvent struct {
	User       string
	FormerNick string
}

type SelfJoinEvent struct {
	Channel string
}

type UserJoinEvent struct {
	User    string
	Channel string
}

type SelfPartEvent struct {
	Channel string
}

type UserPartEvent struct {
	User    string
	Channel string
}

type UserQuitEvent struct {
	User     string
	Channels []string
}

type TopicChangeEvent struct {
	Channel string
	Topic   string
}

type MessageEvent struct {
	User            string
	Target          string
	TargetIsChannel bool
	Command         string
	Content         string
	Time            time.Time
}

type HistoryEvent struct {
	Target   string
	Messages []Event
}

type NewNetworkEvent struct {
	NetworkID string
}
