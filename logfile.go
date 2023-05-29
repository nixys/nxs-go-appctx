package appctx

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type logFile struct {
	*os.File
	m       sync.Mutex
	path    string
	sigchan chan os.Signal
}

type logFormatter struct{}

// LogfileInit initializes logging
func LogfileInit(logfile, loglevel string, logrotateSignals []os.Signal, formatter logrus.Formatter) (*logrus.Logger, error) {

	// Validate log level
	level, err := logrus.ParseLevel(loglevel)
	if err != nil {
		return nil, fmt.Errorf("wrong loglevel value: %s", loglevel)
	}

	switch logfile {
	case "", "stdout":
		return &logrus.Logger{
			Out:       os.Stdout,
			Level:     level,
			Formatter: logFormat(formatter),
		}, nil
	case "stderr":
		return &logrus.Logger{
			Out:       os.Stderr,
			Level:     level,
			Formatter: logFormat(formatter),
		}, nil
	}

	log := &logrus.Logger{
		Level:     level,
		Formatter: logFormat(formatter),
	}

	// Open log file
	l, err := logfileOpen(logfile, logrotateSignals)
	if err != nil {
		return log, fmt.Errorf("can't open log file: %v", err)
	}

	log.SetOutput(l)

	return log, nil
}

// LogfileChange changes logging settings.
// It opens new log file, sets new log level and log rotation signals
func LogfileChange(log *logrus.Logger, logfileNew, loglevelNew string, logrotateSignalsNew []os.Signal) error {

	var l io.Writer

	// Validate log level
	level, err := logrus.ParseLevel(loglevelNew)
	if err != nil {
		return fmt.Errorf("wrong loglevel value: %s", loglevelNew)
	}

	switch logfileNew {
	case "", "stdout":
		l = os.Stdout
	case "stderr":
		l = os.Stderr
	default:
		l, err = logfileOpen(logfileNew, logrotateSignalsNew)
		if err != nil {
			return fmt.Errorf("can't open log file: %v", err)
		}
	}

	// Get current output setting
	v, isLogFile := log.Out.(*logFile)

	// Apply new logging settings
	log.SetLevel(level)
	log.SetOutput(l)

	// Close old log file
	if isLogFile {
		v.Close()
	}

	return nil
}

// LogfileClose closes log file.
func LogfileClose(log *logrus.Logger) error {

	// Close the log file
	if v, isLogFile := log.Out.(*logFile); isLogFile {
		v.Close()
	}
	log.SetOutput(os.Stdout)

	return nil
}

// logFileOpen opens specified log file and waits signal SIGUSR1 to reopen that file (used for log rotate)
func logfileOpen(path string, signals []os.Signal) (*logFile, error) {

	l := &logFile{
		m:       sync.Mutex{},
		path:    path,
		sigchan: make(chan os.Signal, 1),
	}

	if err := l.reopen(); err != nil {
		return nil, err
	}

	// If at least one signal is set
	if len(signals) > 0 {
		// Wait for signal and reopen log file
		go func() {
			signal.Notify(l.sigchan, signals...)
			for range l.sigchan {
				if err := l.reopen(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: error reopening log file: %v\n", time.Now(), err)
				}
			}
		}()
	}

	return l, nil
}

func (l *logFile) reopen() (err error) {

	l.m.Lock()
	defer l.m.Unlock()

	l.File.Close()
	l.File, err = os.OpenFile(l.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	return err
}

func (l *logFile) Write(b []byte) (int, error) {

	l.m.Lock()
	defer l.m.Unlock()

	return l.File.Write(b)
}

func (l *logFile) Close() error {

	l.m.Lock()
	defer l.m.Unlock()

	signal.Stop(l.sigchan)
	close(l.sigchan)

	return l.File.Close()
}

func logFormat(formatter logrus.Formatter) logrus.Formatter {
	if formatter == nil {
		return &logFormatter{}
	}
	return formatter
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
