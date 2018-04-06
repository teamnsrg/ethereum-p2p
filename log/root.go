package log

import (
	"os"
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

// Neighbors is a convenient alias for Root().Neighbors
func Neighbors(msg string, ctx ...interface{}) {
	root.write(msg, LvlNeighbors, ctx)
}

// Hello is a convenient alias for Root().Hello
func Hello(msg string, ctx ...interface{}) {
	root.write(msg, LvlHello, ctx)
}

// DiscProto is a convenient alias for Root().DiscProto
func DiscProto(msg string, ctx ...interface{}) {
	root.write(msg, LvlDiscProto, ctx)
}

// DiscPeer is a convenient alias for Root().DiscPeer
func DiscPeer(msg string, ctx ...interface{}) {
	root.write(msg, LvlDiscPeer, ctx)
}

// Status is a convenient alias for Root().Status
func Status(msg string, ctx ...interface{}) {
	root.write(msg, LvlStatus, ctx)
}

// DaoFork is a convenient alias for Root().DaoFork
func DaoFork(msg string, ctx ...interface{}) {
	root.write(msg, LvlDaoFork, ctx)
}

// TxData is a convenient alias for Root().TxData
func TxData(msg string, ctx ...interface{}) {
	root.write(msg, LvlTxData, ctx)
}

// TxRx is a convenient alias for Root().TxRx
func TxRx(msg string, ctx ...interface{}) {
	root.write(msg, LvlTxRx, ctx)
}

// TxTx is a convenient alias for Root().TxTx
func TxTx(msg string, ctx ...interface{}) {
	root.write(msg, LvlTxTx, ctx)
}

// NewBlockData is a convenient alias for Root().NewBlockData
func NewBlockData(msg string, ctx ...interface{}) {
	root.write(msg, LvlNewBlockData, ctx)
}

// NewBlockRx is a convenient alias for Root().NewBlockRx
func NewBlockRx(msg string, ctx ...interface{}) {
	root.write(msg, LvlNewBlockRx, ctx)
}

// NewBlockTx is a convenient alias for Root().NewBlockTx
func NewBlockTx(msg string, ctx ...interface{}) {
	root.write(msg, LvlNewBlockTx, ctx)
}

// NewBlockHashesRx is a convenient alias for Root().NewBlockHashesRx
func NewBlockHashesRx(msg string, ctx ...interface{}) {
	root.write(msg, LvlNewBlockHashesRx, ctx)
}

// NewBlockHashesTx is a convenient alias for Root().NewBlockHashesTx
func NewBlockHashesTx(msg string, ctx ...interface{}) {
	root.write(msg, LvlNewBlockHashesTx, ctx)
}
