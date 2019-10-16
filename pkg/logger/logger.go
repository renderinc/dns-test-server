package logger

import (
	"fmt"
	"time"
)

type Logger interface {
	Info(args ...interface{})
	Infof(f string, args ...interface{})
}

type StdLogger struct{}

func NewStdLogger() *StdLogger {
	return &StdLogger{}
}

func (s *StdLogger) Info(args ...interface{}) {
	fmt.Println(append([]interface{}{time.Now().Format(time.RFC3339 + ":")}, args...)...)
}

func (s *StdLogger) Infof(f string, args ...interface{}) {
	fmt.Printf("%v: "+f+"\n", append([]interface{}{time.Now().Format(time.RFC3339)}, args...)...)
}
