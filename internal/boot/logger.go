package boot

import (
	"fmt"
	"io"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/asaidimu/hestia/internal/core"
)

// UserOutput prints plain-text informational messages to the user on stdout.
// It is NOT a structured logger — output is meant for human eyes.
type UserOutput struct {
	w io.Writer
}

func NewUserOutput() *UserOutput {
	return &UserOutput{w: os.Stdout}
}

func (u *UserOutput) Printf(format string, args ...any) {
	fmt.Fprintf(u.w, format, args...)
}

func (u *UserOutput) Print(args ...any) {
	fmt.Fprint(u.w, args...)
}

func (u *UserOutput) Banner() {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                    Hestia Platform                           ║
║                                                              ║
║                  ERP Template Server                         ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Fprint(u.w, banner)
}

type Loggers struct {
	File       *zap.Logger
	Stdout     *UserOutput
	lumberjack *lumberjack.Logger
}

func NewLoggers(cfg *core.Config) *Loggers {
	stdout := NewUserOutput()

	rotator := &lumberjack.Logger{
		Filename:   cfg.LogPath,
		MaxSize:    cfg.LogMaxSize,
		MaxAge:     cfg.LogMaxAge,
		MaxBackups: cfg.LogMaxBackups,
		LocalTime:  true,
	}

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(rotator),
		zap.InfoLevel,
	)

	fileLogger := zap.New(fileCore, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return &Loggers{File: fileLogger, Stdout: stdout, lumberjack: rotator}
}

func (l *Loggers) Close() error {
	_ = l.File.Sync()
	return l.lumberjack.Close()
}
