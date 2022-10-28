package logger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogFormatter struct{}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {

	var (
		out, job, storage string
		s                 []string
	)

	for k, v := range entry.Data {
		switch k {
		case "job":
			job = fmt.Sprintf("%s", v)
		case "storage":
			storage = fmt.Sprintf("%s", v)
		default:
			s = append(s, fmt.Sprintf("%s: %v", k, v))
		}
	}

	out = fmt.Sprintf("%s [%s]", strings.ToUpper(entry.Level.String()), entry.Time.Format("2006-01-02 15:04:05.000"))
	if job != "" {
		out += fmt.Sprintf("[%s]", job)
	}
	if storage != "" {
		out += fmt.Sprintf("(%s)", storage)
	}
	out += fmt.Sprintf(": %s", entry.Message)
	if len(s) > 0 {
		out += fmt.Sprintf(" (%s)\n", strings.Join(s, ", "))
	} else {
		out += "\n"
	}

	return []byte(out), nil
}
