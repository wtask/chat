package chat

// MessageHistory - interface to access ordered history of chat messages
type MessageHistory interface {
	// Push - push new message into history
	Push(string)
	// Tail - get a number of latest messages from history in chronological order
	Tail(n int) []string
}

func historyPush(h MessageHistory, message string) {
	if h == nil {
		return
	}
	h.Push(message)
}

func historyTail(h MessageHistory, n int) []string {
	if h == nil {
		return []string{}
	}
	return h.Tail(n)
}
