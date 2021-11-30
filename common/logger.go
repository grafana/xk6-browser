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
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func NullLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)
	return log
}

func NewLogger(logger *logrus.Logger) *Logger {
	return &Logger{logger}
}

// DebugMode returns true if the logger level is set to Debug or higher.
func (l *Logger) DebugMode() bool {
	return l.Logger.GetLevel() >= logrus.DebugLevel
}

// ReportCaller adds source file and function names to the log entries.
func (l *Logger) EnableReportCaller() {
	const mod = "github.com/grafana/xk6-browser"

	// strip the module informaiton
	strip := func(s string) string {
		if !strings.Contains(s, mod) {
			return s
		}
		s = strings.TrimPrefix(s, mod)
		s = s[strings.Index(s, "/")+1:]
		return s
	}
	caller := func() func(*runtime.Frame) (string, string) {
		return func(f *runtime.Frame) (fn string, loc string) {
			// find the caller of the log func
			fn = f.Func.Name()
			_, file, no, ok := runtime.Caller(8)
			if ok {
				fn = fmt.Sprintf("%s:%d", strip(file), no)
			}
			return fn, ""
		}
	}
	l.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: caller(),
	})
	l.SetReportCaller(true)
}
