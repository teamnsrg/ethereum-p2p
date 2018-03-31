package log

import (
	"fmt"
	"os"
	"time"
)

var (
	root          = &logger{ctx: []interface{}{}, h: new(swapHandler)}
	StdoutHandler = StreamHandler(os.Stdout, LogfmtFormat())
	StderrHandler = StreamHandler(os.Stderr, LogfmtFormat())
)

func init() {
	root.SetHandler(DiscardHandler())
}

// New returns a new logger with the given context.
// New is a convenient alias for Root().New
func New(ctx ...interface{}) Logger {
	return root.New(ctx...)
}

// Root returns the root logger
func Root() Logger {
	return root
}

// The following functions bypass the exported logger methods (logger.Debug,
// etc.) to keep the call depth the same for all paths to logger.write so
// runtime.Caller(2) always refers to the call site in client code.

func DaoFork(t time.Time, connInfoCtx []interface{}, support bool) {
	ctx := []interface{}{
		"support", support,
	}
	root.write(fmt.Sprintf("%.6f", float64(t.UnixNano())/1e9), LvlDaoFork, ctx)
}

func EthTx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlEthTx, t, connInfoCtx, msgType, size, data, err)
}

func EthRx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlEthRx, t, connInfoCtx, msgType, size, data, err)
}

func DEVp2pTx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlDEVp2pTx, t, connInfoCtx, msgType, size, data, err)
}

func DEVp2pRx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlDEVp2pRx, t, connInfoCtx, msgType, size, data, err)
}

func RLPXTx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlRLPXTx, t, connInfoCtx, msgType, size, data, err)
}

func RLPXRx(t time.Time, connInfoCtx []interface{}, msgType string, size int, data interface{}, err error) {
	root.writeMsgType(LvlRLPXRx, t, connInfoCtx, msgType, size, data, err)
}

// Trace is a convenient alias for Root().Trace
func Trace(msg string, ctx ...interface{}) {
	root.write(msg, LvlTrace, ctx)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, ctx ...interface{}) {
	root.write(msg, LvlDebug, ctx)
}

// Info is a convenient alias for Root().Info
func Info(msg string, ctx ...interface{}) {
	root.write(msg, LvlInfo, ctx)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, ctx ...interface{}) {
	root.write(msg, LvlWarn, ctx)
}

// Error is a convenient alias for Root().Error
func Error(msg string, ctx ...interface{}) {
	root.write(msg, LvlError, ctx)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, ctx ...interface{}) {
	root.write(msg, LvlCrit, ctx)
	os.Exit(1)
}
