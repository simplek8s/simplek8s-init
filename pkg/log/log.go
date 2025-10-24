// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package log provides simple logging functionality.
// Log messages have levels that determine verbosity and importance.
package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
)

// callerFrame returns file, line, and function name of the first frame outside
// this log package.
func callerFrame() (file string, line int, fnName string) {
	// NOTE: skip=0 is callerFrame, 1 is print/printf/printFn, etc.
	for skip := 2; skip < 15; skip++ {
		var pc uintptr
		var ok bool
		pc, file, line, ok = runtime.Caller(skip)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		fnName = fn.Name()

		// skip frames of this package
		if strings.Contains(fnName, "simplek8s/pkg/log") {
			continue
		}

		file = strings.TrimPrefix(file, "simplek8s/")
		fnName = filepath.Base(fnName)

		return
	}
	return
}

var Output io.Writer = os.Stdout

func init() {
	if !common.IsPathExists("/dev/kmsg") {
		mount.Mount(mount.Mountpoints.Dev)
		defer mount.Unmount(mount.Mountpoints.Dev.Target, 0)
	}

	w, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0o644)
	if err == nil {
		Output = w
	}
}

// defaultFormat returns the log message formatted according to the output type.
// If writing to the kernel log, it uses the kernel log format, otherwise stdout.
func defaultFormat(level LogLevel, msg string) string {
	if Output != os.Stdout {
		return FormatKmsg(level, msg)
	} else {
		return FormatStdout(level, msg)
	}
}

// FormatKmsg formats a message for the kernel log (/dev/kmsg).
func FormatKmsg(level LogLevel, msg string) string {
	pid := os.Getpid()
	file, line, fnName := callerFrame()

	return fmt.Sprintf("<%d>simplek8s[%d]: %s:%d %s(): %s\n", level, pid, file, line, fnName, msg)
}

// FormatStdout formats a message for standard output.
func FormatStdout(level LogLevel, msg string) string {
	var sLevel string
	switch level {
	case LevelTrace:
		sLevel = "\033[0;34mTRACE\033[0m"
	case LevelDebug:
		sLevel = "\033[0;37mDEBUG\033[0m"
	case LevelInfo:
		sLevel = "\033[0;36mINFO\033[0m"
	case LevelWarn:
		sLevel = "\033[0;33mWARN\033[0m"
	case LevelError:
		sLevel = "\033[0;31mERROR\033[0m]"
	case LevelCritical:
		sLevel = "\033[2;31mCRITICAL\033[0m"
	case LevelAlert:
		sLevel = "\033[4;31mALERT\033[0m"
	}

	file, line, fnName := callerFrame()

	return fmt.Sprintf("%s: %s:%d %s(): %s\n", sLevel, file, line, fnName, msg)
}

type jsonEntry struct {
	Level     LogLevel
	Msg       string
	File      string
	Line      int
	Function  string
	Timestamp int64
}

// FormatJson formats a message as a JSON object with level and timestamp.
func FormatJson(level LogLevel, msg string) string {
	file, line, fnName := callerFrame()
	timestamp := time.Now().Unix()

	jEntry := jsonEntry{
		Level:     level,
		Msg:       msg,
		File:      file,
		Line:      line,
		Function:  fnName,
		Timestamp: timestamp,
	}
	d, _ := json.Marshal(jEntry)
	return string(d)
}

var Format = defaultFormat

// LogLevel represents the severity or verbosity of a log message.
type LogLevel int

const (
	LevelTrace     LogLevel = 8 // Trace is for detailed messages tracking function flow.
	LevelDebug     LogLevel = 7 // Debug is for debug-level messages.
	LevelInfo      LogLevel = 6 // Info is for informational messages.
	LevelNotice    LogLevel = 5 // Notice is for normal but significant condition.
	LevelWarn      LogLevel = 4 // Warn is for warning conditions.
	LevelError     LogLevel = 3 // Error is for error conditions.
	LevelCritical  LogLevel = 2 // Critical is for critical conditions.
	LevelAlert     LogLevel = 1 // Alert is for conditions that require immediate action.
	LevelEmergency LogLevel = 0 // Emergency is for when the system is unusable.
)

// Level sets the minimum level of messages that will be printed.
//
// Messages below this level will be discarded.
var Level LogLevel = LevelInfo

// print writes a message to the log with the given level and format.
//
// Messages below LogLevel are ignored.
// The output format depends on the configured Format function.
func print(level LogLevel, msg string) {
	if level > Level {
		return
	}

	s := Format(level, msg)
	fmt.Fprint(Output, s)
}

// printf formats a message according to a printf-style format string and
// writes it to the log.
func printf(level LogLevel, format string, args ...any) {
	if level > Level {
		return
	}

	s := fmt.Sprintf(format, args...)
	s = Format(level, s)
	fmt.Fprint(Output, s)
}

// Println writes whatever fn returns to the log with the given level.
//
// fn will not be called if level is less than LogLevel.
func printFn(level LogLevel, fn func() string) {
	if level > Level {
		return
	}

	s := fn()
	s = Format(level, s)
	fmt.Fprint(Output, s)
}

func Trace(msg string)                  { print(LevelTrace, msg) }
func Tracef(format string, args ...any) { printf(LevelTrace, format, args...) }
func TraceFn(fn func() string)          { printFn(LevelTrace, fn) }

func Debug(msg string)                  { print(LevelDebug, msg) }
func Debugf(format string, args ...any) { printf(LevelDebug, format, args...) }
func DebugFn(fn func() string)          { printFn(LevelDebug, fn) }

func Info(msg string)                  { print(LevelInfo, msg) }
func Infof(format string, args ...any) { printf(LevelInfo, format, args...) }
func InfoFn(fn func() string)          { printFn(LevelInfo, fn) }

func Notice(msg string)                  { print(LevelNotice, msg) }
func Noticef(format string, args ...any) { printf(LevelNotice, format, args...) }
func NoticeFn(fn func() string)          { printFn(LevelNotice, fn) }

func Warn(msg string)                  { print(LevelWarn, msg) }
func Warnf(format string, args ...any) { printf(LevelWarn, format, args...) }
func WarnFn(fn func() string)          { printFn(LevelWarn, fn) }

func Error(msg string)                  { print(LevelError, msg) }
func Errorf(format string, args ...any) { printf(LevelError, format, args...) }
func ErrorFn(fn func() string)          { printFn(LevelError, fn) }

func Critical(msg string)                  { print(LevelCritical, msg) }
func Criticalf(format string, args ...any) { printf(LevelCritical, format, args...) }
func CriticalFn(fn func() string)          { printFn(LevelCritical, fn) }

func Alert(msg string)                  { print(LevelAlert, msg) }
func Alertf(format string, args ...any) { printf(LevelAlert, format, args...) }
func AlertFn(fn func() string)          { printFn(LevelAlert, fn) }

func Emergency(msg string)                  { print(LevelEmergency, msg) }
func Emergencyf(format string, args ...any) { printf(LevelEmergency, format, args...) }
func EmergencyFn(fn func() string)          { printFn(LevelEmergency, fn) }
