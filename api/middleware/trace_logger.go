package middleware

import (
	"fmt"
	"log"
)

type TraceLogger struct {
	TraceID string
}

func (t *TraceLogger) Log(message string, args ...interface{}) {
	log.Printf("[TRACE-%s] %s", t.TraceID, fmt.Sprintf(message, args...))
}
