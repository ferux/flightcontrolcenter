package fccgob

// MessageKind describes type of message encoded into body.
type MessageKind uint8

const (
	MessageKindUnknown = iota
	MessageKindLog
	MessageKindNotify
	MessageKindOK
	MessageKindFail
)

func (k MessageKind) String() string {
	switch k {
	case MessageKindUnknown:
		return "kind_unknown"
	case MessageKindLog:
		return "kind_log"
	case MessageKindNotify:
		return "kind_notify"
	case MessageKindOK:
		return "kind_ok"
	case MessageKindFail:
		return "kind_fail"
	default:
		return "undefind"
	}
}

// Message is transffered over the wire packet.
type Message struct {
	RequestID string
	Kind      MessageKind
	Data      []byte
}

type notifyTelegram struct {
	APIKey  string
	ChatID  string
	Message string
}

type logMessage struct {
	Text     string
	Severity logSeverity
}

type ok struct{}

type failure struct {
	Reason string
}
