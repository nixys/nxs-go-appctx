package appctx

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type logFormatter struct{}

func DefaultLogInit(logfile *os.File, loglevel logrus.Level, formatter logrus.Formatter) (*logrus.Logger, error) {
	return &logrus.Logger{
		Out:   logfile,
		Level: loglevel,
		Formatter: func() logrus.Formatter {
			if formatter == nil {
				return &logFormatter{}
			}
			return formatter
		}(),
	}, nil
}

// Set default log format function
func (f *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {

	var (
		o string
		s []string
	)

	for k, v := range entry.Data {
		s = append(s, fmt.Sprintf("%s: %v", k, v))
	}

	if len(s) > 0 {
		o = fmt.Sprintf("[%s] %s: %s (%s)\n",
			entry.Time.Format(time.RFC3339),
			strings.ToUpper(entry.Level.String()),
			entry.Message,
			strings.Join(s, ", "))
	} else {
		o = fmt.Sprintf("[%s] %s: %s\n",
			entry.Time.Format(time.RFC3339),
			strings.ToUpper(entry.Level.String()),
			entry.Message)
	}

	return []byte(o), nil
}
