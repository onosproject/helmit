package logging

type Logger interface {
	Log(message string)
	Logf(message string, args ...any)
}
