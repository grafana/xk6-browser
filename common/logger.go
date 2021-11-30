/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package common

import (
	"context"
	"io/ioutil"
	"regexp"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	ctx context.Context
	*logrus.Logger
	// mu             sync.Mutex
	// lastLogCall    int64
	debugOverride  bool
	categoryFilter *regexp.Regexp
}

func NullLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)
	return log
}

func NewLogger(ctx context.Context, logger *logrus.Logger, debugOverride bool, categoryFilter *regexp.Regexp) *Logger {
	return &Logger{
		ctx:            ctx,
		Logger:         logger,
		debugOverride:  debugOverride,
		categoryFilter: categoryFilter,
	}
}

// func (l *Logger) Tracef(category string, msg string, args ...interface{}) {
// 	l.Logf(logrus.TraceLevel, category, msg, args...)
// }

// func (l *Logger) Debugf(category string, msg string, args ...interface{}) {
// 	l.Logf(logrus.DebugLevel, category, msg, args...)
// }

// func (l *Logger) Errorf(category string, msg string, args ...interface{}) {
// 	l.Logf(logrus.ErrorLevel, category, msg, args...)
// }

// func (l *Logger) Infof(category string, msg string, args ...interface{}) {
// 	l.Logf(logrus.InfoLevel, category, msg, args...)
// }

// func (l *Logger) Warnf(category string, msg string, args ...interface{}) {
// 	l.Logf(logrus.WarnLevel, category, msg, args...)
// }

// func (l *Logger) Logf(level logrus.Level, category string, msg string, args ...interface{}) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()

// 	// don't log if the current log level isn't in the required level.
// 	if l.log.GetLevel() < level {
// 		return
// 	}
// 	if l.categoryFilter != nil && !l.categoryFilter.Match([]byte(category)) {
// 		return
// 	}

// 	now := time.Now().UnixNano() / 1000000
// 	elapsed := now - l.lastLogCall
// 	if now == elapsed {
// 		elapsed = 0
// 	}
// 	defer func() {
// 		l.lastLogCall = now
// 	}()

// 	if l.log == nil {
// 		magenta := color.New(color.FgMagenta).SprintFunc()
// 		fmt.Printf("ZZZ%s [%d]: %s - %s ms\n", magenta(category), goRoutineID(), string(msg), magenta(elapsed))
// 		return
// 	}
// 	entry := l.log.WithFields(logrus.Fields{
// 		"category":  category,
// 		"elapsed":   fmt.Sprintf("%d ms", elapsed),
// 		"goroutine": goRoutineID(),
// 	})
// 	if l.log.GetLevel() < level && l.debugOverride {
// 		entry.Printf("XXX"+msg, args...)
// 		return
// 	}
// 	entry.Logf(level, "YYY"+msg, args...)
// }

// // SetLevel sets the logger level from a level string.
// // Accepted values:
// func (l *Logger) SetLevel(level string) error {
// 	pl, err := logrus.ParseLevel(level)
// 	if err != nil {
// 		return err
// 	}
// 	l.log.SetLevel(pl)
// 	return nil
// }

// DebugMode returns true if the logger level is set to Debug or higher.
func (l *Logger) DebugMode() bool {
	return l.Logger.GetLevel() >= logrus.DebugLevel
}

// // ReportCaller adds source file and function names to the log entries.
// func (l *Logger) ReportCaller() {
// 	const mod = "github.com/grafana/xk6-browser"

// 	// strip the module informaiton
// 	strip := func(s string) string {
// 		if !strings.Contains(s, mod) {
// 			return s
// 		}
// 		s = strings.TrimPrefix(s, mod)
// 		s = s[strings.Index(s, "/")+1:]
// 		return s
// 	}
// 	caller := func() func(*runtime.Frame) (string, string) {
// 		return func(f *runtime.Frame) (fn string, loc string) {
// 			// loc = fmt.Sprintf("%s:%d", f.File, f.Line)
// 			// // strip the module informaiton
// 			// loc = strip(loc)
// 			// find the caller of the log func
// 			fn = f.Func.Name()
// 			_, file, no, ok := runtime.Caller(8)
// 			if ok {
// 				fn = fmt.Sprintf("%s:%d", strip(file), no)
// 			}
// 			return fn, ""
// 		}
// 	}
// 	l.log.SetFormatter(&logrus.TextFormatter{
// 		CallerPrettyfier: caller(),
// 	})
// 	l.log.SetReportCaller(true)
// }

// func goRoutineID() int {
// 	var buf [64]byte
// 	n := runtime.Stack(buf[:], false)
// 	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
// 	id, err := strconv.Atoi(idField)
// 	if err != nil {
// 		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
// 	}
// 	return id
// }
