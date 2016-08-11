package gossip

import (
	"fmt"
	"log"
)

type Logger interface {
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

// If we want no noise, don't care about it even logging to file then
// we can use this
type SilentLogger struct{}

func NewSilentLogger() *SilentLogger {
	return &SilentLogger{}
}

func (SilentLogger) Debugf(format string, v ...interface{}) {
}

func (SilentLogger) Errorf(format string, v ...interface{}) {
}

func (SilentLogger) Fatalf(format string, v ...interface{}) {
}

// the StdoutLogger simply leverages go's log.Println
type StdoutLogger struct{}

func NewStdoutLogger() *StdoutLogger {
	return &StdoutLogger{}
}

func (StdoutLogger) Debugf(format string, v ...interface{}) {
	log.Println("[DEBUG] " + fmt.Sprintf(format, v...))
}

func (StdoutLogger) Errorf(format string, v ...interface{}) {
	log.Println("[ERROR] " + fmt.Sprintf(format, v...))
}

func (StdoutLogger) Fatalf(format string, v ...interface{}) {
	log.Println("[FATAL] " + fmt.Sprintf(format, v...))
}
