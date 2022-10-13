package log

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	mu             sync.Mutex
	lastLogCall    int64
	debugOverride  bool
	categoryFilter *regexp.Regexp
}

// NewNullLogger will create a logger where log lines will
// be discarded and not logged anywhere.
func NewNullLogger() *Logger {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	return New(log, false, nil)
}

// New creates a new logger.
func New(logger *logrus.Logger, debugOverride bool, categoryFilter *regexp.Regexp) *Logger {
	return &Logger{
		Logger:         logger,
		debugOverride:  debugOverride,
		categoryFilter: categoryFilter,
	}
}

func (l *Logger) Tracef(category string, msg string, args ...interface{}) {
	l.Logf(logrus.TraceLevel, category, msg, args...)
}

func (l *Logger) Debugf(category string, msg string, args ...interface{}) {
	l.Logf(logrus.DebugLevel, category, msg, args...)
}

func (l *Logger) Errorf(category string, msg string, args ...interface{}) {
	l.Logf(logrus.ErrorLevel, category, msg, args...)
}

func (l *Logger) Infof(category string, msg string, args ...interface{}) {
	l.Logf(logrus.InfoLevel, category, msg, args...)
}

func (l *Logger) Warnf(category string, msg string, args ...interface{}) {
	l.Logf(logrus.WarnLevel, category, msg, args...)
}

func (l *Logger) Logf(level logrus.Level, category string, msg string, args ...interface{}) {
	if l == nil {
		return
	}
	// don't log if the current log level isn't in the required level.
	if l.GetLevel() < level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano() / 1000000
	elapsed := now - l.lastLogCall
	if now == elapsed {
		elapsed = 0
	}
	defer func() {
		l.lastLogCall = now
	}()

	if l.categoryFilter != nil && !l.categoryFilter.Match([]byte(category)) {
		return
	}
	if l.Logger == nil {
		magenta := color.New(color.FgMagenta).SprintFunc()
		fmt.Printf("%s [%d]: %s - %s ms\n", magenta(category), goRoutineID(), string(msg), magenta(elapsed))
		return
	}
	entry := l.WithFields(logrus.Fields{
		"category":  category,
		"elapsed":   fmt.Sprintf("%d ms", elapsed),
		"goroutine": goRoutineID(),
	})
	if l.GetLevel() < level && l.debugOverride {
		entry.Printf(msg, args...)
		return
	}
	entry.Logf(level, msg, args...)
}

// SetLevel sets the logger level from a level string.
// Accepted values:
//  - "panic"
//  - "fatal"
//  - "error"
//  - "warn"
//  - "warning"
//  - "info"
//  - "debug"
//  - "trace"
func (l *Logger) SetLevel(level string) error {
	pl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	l.Logger.SetLevel(pl)
	return nil
}

// DebugMode returns true if the logger level is set to Debug or higher.
func (l *Logger) DebugMode() bool {
	return l.GetLevel() >= logrus.DebugLevel
}

// ReportCaller adds source file and function names to the log entries.
func (l *Logger) ReportCaller() {
	caller := func() func(*runtime.Frame) (string, string) {
		return func(f *runtime.Frame) (function string, file string) {
			return f.Func.Name(), fmt.Sprintf("%s:%d", f.File, f.Line)
		}
	}
	l.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: caller(),
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyFile: "caller",
		},
	})
	l.SetReportCaller(true)
}

// ConsoleLogFormatterSerializer creates a new logger that will
// correctly serialize RemoteObject instances.
func (l *Logger) ConsoleLogFormatterSerializer() *Logger {
	return &Logger{
		Logger: &logrus.Logger{
			Out:       l.Out,
			Level:     l.Level,
			Formatter: &consoleLogFormatter{l.Formatter},
		},
	}
}

func goRoutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("internal error while getting goroutine ID: %v", err))
	}
	return id
}

type consoleLogFormatter struct {
	logrus.Formatter
}

// Format assembles a message from marshalling elements in the "objects" field
// to JSON separated by space, and deletes the field when done.
func (f *consoleLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if objects, ok := entry.Data["objects"].([]interface{}); ok {
		var msg []string
		for _, obj := range objects {
			// TODO: Log error?
			if o, err := json.Marshal(obj); err == nil {
				msg = append(msg, string(o))
			}
		}
		entry.Message = strings.Join(msg, " ")
		delete(entry.Data, "objects")
	}
	return f.Formatter.Format(entry)
}
