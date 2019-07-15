package chat

// Logger - interface for logging chat events
type Logger interface {
	Println(v ...interface{})
}

func logInfo(l Logger, v ...interface{}) {
	if l == nil {
		return
	}
	l.Println(v...)
}

func logError(l Logger, v ...interface{}) {
	if l == nil {
		return
	}
	l.Println(append([]interface{}{"ERR"}, v...)...)
}
