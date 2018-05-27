package log

import (
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

func TxTx(t time.Time, connInfoCtx []interface{}, size uint32, encodedSize uint32, txHash string) {
	ctx := append(connInfoCtx,
		"size", size,
		"encodedSize", encodedSize,
		"txHash", txHash,
	)
	root.writeTime(LvlTxTx, t, ctx)
}

func TxData(t time.Time, connInfoCtx []interface{}, size uint32, encodedSize uint32, tx string) {
	ctx := append(connInfoCtx,
		"size", size,
		"encodedSize", encodedSize,
		"tx", tx,
	)
	root.writeTime(LvlTxData, t, ctx)
}

func Task(msg string, taskInfoCtx []interface{}) {
	root.write(msg, LvlTask, taskInfoCtx)
}

func MessageTx(t time.Time, msgType string, size uint32, encodedSize uint32, connInfoCtx []interface{}, err error) {
	root.writeTimeMsgType(LvlMessageTx, t, msgType, size, encodedSize, connInfoCtx, err)
}

func MessageRx(t time.Time, msgType string, size uint32, encodedSize uint32, connInfoCtx []interface{}, err error) {
	root.writeTimeMsgType(LvlMessageRx, t, msgType, size, encodedSize, connInfoCtx, err)
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
