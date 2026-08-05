package app

import (
	"fmt"
	"strings"
)

type logField struct {
	key   string
	value string
}

func formatLogMessage(level string, operation string, err error, fields ...logField) string {
	var b strings.Builder
	if level != "" {
		b.WriteString(level)
		b.WriteString(": ")
	}
	b.WriteString(operation)
	for _, field := range fields {
		if field.key == "" || field.value == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(field.key)
		b.WriteByte('=')
		b.WriteString(fmt.Sprintf("%q", field.value))
	}
	if err != nil {
		b.WriteString(": ")
		b.WriteString(err.Error())
	}
	return b.String()
}
