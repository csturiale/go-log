// The colorful and simple logging library
// Copyright (c) 2017 Fadhli Dzil Ikram

package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/csturiale/go-log/colorful"
)

// FdWriter interface extends existing io.Writer with file descriptor function
// support
type FdWriter interface {
	io.Writer
	Fd() uintptr
}
type Config struct {
	Color     bool
	Out       FdWriter
	Debug     bool
	Timestamp bool
	Quiet     bool
	Prefix    string
}

// Logger struct define the underlying storage for single logger
type Logger struct {
	mu     sync.RWMutex
	config Config
	buf    colorful.ColorBuffer
}

// Prefix struct define plain and Color byte
type Prefix struct {
	Plain []byte
	Color []byte
	File  bool
}

var (
	// Plain prefix template
	plainFatal = []byte("[FATAL] ")
	plainError = []byte("[ERROR] ")
	plainWarn  = []byte("[WARN]  ")
	plainInfo  = []byte("[INFO]  ")
	plainDebug = []byte("[DEBUG] ")
	plainTrace = []byte("[TRACE] ")

	// FatalPrefix show fatal prefix
	FatalPrefix = Prefix{
		Plain: plainFatal,
		Color: colorful.Red(plainFatal),
		File:  true,
	}

	// ErrorPrefix show error prefix
	ErrorPrefix = Prefix{
		Plain: plainError,
		Color: colorful.Red(plainError),
		File:  true,
	}

	// WarnPrefix show warn prefix
	WarnPrefix = Prefix{
		Plain: plainWarn,
		Color: colorful.Orange(plainWarn),
	}

	// InfoPrefix show info prefix
	InfoPrefix = Prefix{
		Plain: plainInfo,
		Color: colorful.Green(plainInfo),
	}

	// DebugPrefix show info prefix
	DebugPrefix = Prefix{
		Plain: plainDebug,
		Color: colorful.Purple(plainDebug),
		File:  true,
	}

	// TracePrefix show info prefix
	TracePrefix = Prefix{
		Plain: plainTrace,
		Color: colorful.Cyan(plainTrace),
	}
	logger *Logger
)

// Init returns single logger instance with predefined writer output and
// automatically detect terminal coloring support
func Init(config Config) (*Logger, error) {
	if config.Out == nil {
		return nil, errors.New("config.out is a mandatory field")
	}
	if logger == nil {
		logger = newLogger(config)
	}
	return logger, nil
}

// newLogger returns newLogger Logger instance with predefined writer output and
// automatically detect terminal coloring support
func newLogger(config Config) *Logger {
	return &Logger{
		config: config,
	}
}

// WithColor explicitly turn on colorful features on the log
func (l *Logger) WithColor() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Color = true
	return l
}

// WithoutColor explicitly turn off colorful features on the log
func (l *Logger) WithoutColor() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Color = false
	return l
}

// WithDebug turn on debugging output on the log to reveal Debug and trace level
func (l *Logger) WithDebug() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Debug = true
	return l
}

// WithoutDebug turn off debugging output on the log
func (l *Logger) WithoutDebug() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Debug = false
	return l
}

// IsDebug check the state of debugging output
func (l *Logger) IsDebug() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Debug
}

// WithTimestamp turn on Timestamp output on the log
func (l *Logger) WithTimestamp() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Timestamp = true
	return l
}

// WithoutTimestamp turn off Timestamp output on the log
func (l *Logger) WithoutTimestamp() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Timestamp = false
	return l
}

// Quiet turn off all log output
func (l *Logger) Quiet() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Quiet = true
	return l
}

// NoQuiet turn on all log output
func (l *Logger) NoQuiet() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Quiet = false
	return l
}

// IsQuiet check for Quiet state
func (l *Logger) IsQuiet() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Quiet
}

// Output print the actual value
func (l *Logger) Output(depth int, prefix Prefix, data string) error {
	// Check if Quiet is requested, and try to return no error and be Quiet
	if l.IsQuiet() {
		return nil
	}
	// Get current time
	now := time.Now()
	// Temporary storage for file and line tracing
	var file string
	var line int
	var fn string
	// Check if the specified prefix needs to be included with file logging
	if prefix.File {
		var ok bool
		var pc uintptr

		// Get the caller filename and line
		if pc, file, line, ok = runtime.Caller(depth + 1); !ok {
			file = "<unknown file>"
			fn = "<unknown function>"
			line = 0
		} else {
			file = filepath.Base(file)
			fn = runtime.FuncForPC(pc).Name()
		}
	}
	// Acquire exclusive access to the shared buffer
	l.mu.Lock()
	defer l.mu.Unlock()
	// Reset buffer so it start from the begining
	l.buf.Reset()
	// Write prefix to the buffer
	if l.config.Color {
		l.buf.Off()
		l.buf.Append([]byte("[" + l.config.Prefix + "]"))
		l.buf.Append(prefix.Color)
	} else {
		l.buf.Append([]byte("[" + l.config.Prefix + "]"))
		l.buf.Append(prefix.Plain)
	}
	// Check if the log require timestamping
	if l.config.Timestamp {
		// Print Timestamp Color if Color enabled
		if l.config.Color {
			l.buf.Blue()
		}
		// Print date and time
		year, month, day := now.Date()
		l.buf.AppendInt(year, 4)
		l.buf.AppendByte('/')
		l.buf.AppendInt(int(month), 2)
		l.buf.AppendByte('/')
		l.buf.AppendInt(day, 2)
		l.buf.AppendByte(' ')
		hour, min, sec := now.Clock()
		l.buf.AppendInt(hour, 2)
		l.buf.AppendByte(':')
		l.buf.AppendInt(min, 2)
		l.buf.AppendByte(':')
		l.buf.AppendInt(sec, 2)
		l.buf.AppendByte(' ')
		// Print reset Color if Color enabled
		if l.config.Color {
			l.buf.Off()
		}
	}
	// Add caller filename and line if enabled
	if prefix.File {
		// Print Color start if enabled
		if l.config.Color {
			l.buf.Orange()
		}
		// Print filename and line
		l.buf.Append([]byte(fn))
		l.buf.AppendByte(':')
		l.buf.Append([]byte(file))
		l.buf.AppendByte(':')
		l.buf.AppendInt(line, 0)
		l.buf.AppendByte(' ')
		// Print Color stop
		if l.config.Color {
			l.buf.Off()
		}
	}
	// Print the actual string data from caller
	l.buf.Append([]byte(data))
	if len(data) == 0 || data[len(data)-1] != '\n' {
		l.buf.AppendByte('\n')
	}
	// Flush buffer to output
	_, err := l.config.Out.Write(l.buf.Buffer)
	return err
}

// Fatal print fatal message to output and quit the application with status 1
func (l *Logger) Fatal(v ...interface{}) {
	l.Output(1, FatalPrefix, fmt.Sprintln(v...))
	os.Exit(1)
}

// Fatalf print formatted fatal message to output and quit the application
// with status 1
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(1, FatalPrefix, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Error print error message to output
func (l *Logger) Error(v ...interface{}) {
	l.Output(1, ErrorPrefix, fmt.Sprintln(v...))
}

// Errorf print formatted error message to output
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Output(1, ErrorPrefix, fmt.Sprintf(format, v...))
}

// Warn print warning message to output
func (l *Logger) Warn(v ...interface{}) {
	l.Output(1, WarnPrefix, fmt.Sprintln(v...))
}

// Warnf print formatted warning message to output
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Output(1, WarnPrefix, fmt.Sprintf(format, v...))
}

// Info print informational message to output
func (l *Logger) Info(v ...interface{}) {
	l.Output(1, InfoPrefix, fmt.Sprintln(v...))
}

// Infof print formatted informational message to output
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Output(1, InfoPrefix, fmt.Sprintf(format, v...))
}

// Debug print Debug message to output if Debug output enabled
func (l *Logger) Debug(v ...interface{}) {
	if l.IsDebug() {
		l.Output(1, DebugPrefix, fmt.Sprintln(v...))
	}
}

// Debugf print formatted Debug message to output if Debug output enabled
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.IsDebug() {
		l.Output(1, DebugPrefix, fmt.Sprintf(format, v...))
	}
}

// Trace print trace message to output if Debug output enabled
func (l *Logger) Trace(v ...interface{}) {
	if l.IsDebug() {
		l.Output(1, TracePrefix, fmt.Sprintln(v...))
	}
}

// Tracef print formatted trace message to output if Debug output enabled
func (l *Logger) Tracef(format string, v ...interface{}) {
	if l.IsDebug() {
		l.Output(1, TracePrefix, fmt.Sprintf(format, v...))
	}
}
