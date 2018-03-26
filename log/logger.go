package log

import (
	"fmt"
	"os"
	"time"

	"github.com/go-stack/stack"
)

const timeKey = "t"
const lvlKey = "lvl"
const msgKey = "msg"
const errorKey = "LOG15_ERROR"

type Lvl int

const (
	LvlCrit Lvl = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
	LvlTrace
	LvlNeighbors
	LvlHello
	LvlDiscProto
	LvlDiscPeer
	LvlStatus
	LvlDaoFork
	LvlTxRx
	LvlTxTx
	LvlNewBlockRx
	LvlNewBlockTx
	LvlNewBlockHashesRx
	LvlNewBlockHashesTx
)

// Aligned returns a 5-character string containing the name of a Lvl.
func (l Lvl) AlignedString() string {
	switch l {
	case LvlTrace:
		return "TRACE"
	case LvlDebug:
		return "DEBUG"
	case LvlInfo:
		return "INFO"
	case LvlWarn:
		return "WARN"
	case LvlError:
		return "ERROR"
	case LvlCrit:
		return "CRIT"
	case LvlNeighbors:
		return "NEIGHBORS"
	case LvlHello:
		return "HELLO"
	case LvlDiscProto:
		return "DISCPROTO"
	case LvlDiscPeer:
		return "DISCPEER"
	case LvlStatus:
		return "STATUS"
	case LvlDaoFork:
		return "DAOFORK"
	case LvlTxRx:
		return "TXRX"
	case LvlTxTx:
		return "TXTX"
	case LvlNewBlockRx:
		return "NBRX"
	case LvlNewBlockTx:
		return "NBTX"
	case LvlNewBlockHashesRx:
		return "NHRX"
	case LvlNewBlockHashesTx:
		return "NHTX"
	default:
		panic("bad level")
	}
}

// Strings returns the name of a Lvl.
func (l Lvl) String() string {
	switch l {
	case LvlTrace:
		return "trce"
	case LvlDebug:
		return "dbug"
	case LvlInfo:
		return "info"
	case LvlWarn:
		return "warn"
	case LvlError:
		return "eror"
	case LvlCrit:
		return "crit"
	case LvlNeighbors:
		return "neighbors"
	case LvlHello:
		return "hello"
	case LvlDiscProto:
		return "disc-proto"
	case LvlDiscPeer:
		return "disc-peer"
	case LvlStatus:
		return "status"
	case LvlDaoFork:
		return "daofork"
	case LvlTxRx:
		return "tx-rx"
	case LvlTxTx:
		return "tx-tx"
	case LvlNewBlockRx:
		return "newblock-rx"
	case LvlNewBlockTx:
		return "newblock-tx"
	case LvlNewBlockHashesRx:
		return "newblockhashes-rx"
	case LvlNewBlockHashesTx:
		return "newblockhashes-tx"
	default:
		panic("bad level")
	}
}

// Returns the appropriate Lvl from a string name.
// Useful for parsing command line args and configuration files.
func LvlFromString(lvlString string) (Lvl, error) {
	switch lvlString {
	case "trace", "trce":
		return LvlTrace, nil
	case "debug", "dbug":
		return LvlDebug, nil
	case "info":
		return LvlInfo, nil
	case "warn":
		return LvlWarn, nil
	case "error", "eror":
		return LvlError, nil
	case "crit":
		return LvlCrit, nil
	case "neighbors":
		return LvlNeighbors, nil
	case "hello":
		return LvlHello, nil
	case "disc-proto":
		return LvlDiscProto, nil
	case "disc-peer":
		return LvlDiscPeer, nil
	case "status":
		return LvlStatus, nil
	case "daofork":
		return LvlDaoFork, nil
	case "tx-rx":
		return LvlTxRx, nil
	case "tx-tx":
		return LvlTxTx, nil
	case "newblock-rx":
		return LvlNewBlockRx, nil
	case "newblock-tx":
		return LvlNewBlockTx, nil
	case "newblockhashes-rx":
		return LvlNewBlockHashesRx, nil
	case "newblockhashes-tx":
		return LvlNewBlockHashesTx, nil
	default:
		return LvlDebug, fmt.Errorf("Unknown level: %v", lvlString)
	}
}

// A Record is what a Logger asks its handler to write
type Record struct {
	Time     time.Time
	Lvl      Lvl
	Msg      string
	Ctx      []interface{}
	Call     stack.Call
	KeyNames RecordKeyNames
}

type RecordKeyNames struct {
	Time string
	Msg  string
	Lvl  string
}

// A Logger writes key/value pairs to a Handler
type Logger interface {
	// New returns a new Logger that has this logger's context plus the given context
	New(ctx ...interface{}) Logger

	// GetHandler gets the handler associated with the logger.
	GetHandler() Handler

	// SetHandler updates the logger to write records to the specified handler.
	SetHandler(h Handler)

	SetGlogger(glogger *GlogHandler)

	GetGlogger() *GlogHandler

	// Log a message at the given level with context key/value pairs
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
	Neighbors(msg string, ctx ...interface{})
	Hello(msg string, ctx ...interface{})
	DiscProto(msg string, ctx ...interface{})
	DiscPeer(msg string, ctx ...interface{})
	Status(msg string, ctx ...interface{})
	DaoFork(msg string, ctx ...interface{})
	TxRx(msg string, ctx ...interface{})
	TxTx(msg string, ctx ...interface{})
	NewBlockRx(msg string, ctx ...interface{})
	NewBlockTx(msg string, ctx ...interface{})
	NewBlockHashesRx(msg string, ctx ...interface{})
	NewBlockHashesTx(msg string, ctx ...interface{})
}

type logger struct {
	ctx     []interface{}
	h       *swapHandler
	datadir string
	glogger *GlogHandler
}

func (l *logger) write(msg string, lvl Lvl, ctx []interface{}) {
	l.h.Log(&Record{
		Time: time.Now(),
		Lvl:  lvl,
		Msg:  msg,
		Ctx:  newContext(l.ctx, ctx),
		Call: stack.Caller(2),
		KeyNames: RecordKeyNames{
			Time: timeKey,
			Msg:  msgKey,
			Lvl:  lvlKey,
		},
	})
}

func (l *logger) New(ctx ...interface{}) Logger {
	child := &logger{ctx: newContext(l.ctx, ctx), h: new(swapHandler)}
	child.SetHandler(l.h)
	return child
}

func newContext(prefix []interface{}, suffix []interface{}) []interface{} {
	normalizedSuffix := normalize(suffix)
	newCtx := make([]interface{}, len(prefix)+len(normalizedSuffix))
	n := copy(newCtx, prefix)
	copy(newCtx[n:], normalizedSuffix)
	return newCtx
}

func (l *logger) Trace(msg string, ctx ...interface{}) {
	l.write(msg, LvlTrace, ctx)
}

func (l *logger) Debug(msg string, ctx ...interface{}) {
	l.write(msg, LvlDebug, ctx)
}

func (l *logger) Info(msg string, ctx ...interface{}) {
	l.write(msg, LvlInfo, ctx)
}

func (l *logger) Warn(msg string, ctx ...interface{}) {
	l.write(msg, LvlWarn, ctx)
}

func (l *logger) Error(msg string, ctx ...interface{}) {
	l.write(msg, LvlError, ctx)
}

func (l *logger) Crit(msg string, ctx ...interface{}) {
	l.write(msg, LvlCrit, ctx)
	os.Exit(1)
}

func (l *logger) Neighbors(msg string, ctx ...interface{}) {
	l.write(msg, LvlNeighbors, ctx)
}

func (l *logger) Hello(msg string, ctx ...interface{}) {
	l.write(msg, LvlHello, ctx)
}

func (l *logger) DiscProto(msg string, ctx ...interface{}) {
	l.write(msg, LvlDiscProto, ctx)
}

func (l *logger) DiscPeer(msg string, ctx ...interface{}) {
	l.write(msg, LvlDiscPeer, ctx)
}

func (l *logger) Status(msg string, ctx ...interface{}) {
	l.write(msg, LvlStatus, ctx)
}

func (l *logger) DaoFork(msg string, ctx ...interface{}) {
	l.write(msg, LvlDaoFork, ctx)
}

func (l *logger) TxRx(msg string, ctx ...interface{}) {
	l.write(msg, LvlTxRx, ctx)
}

func (l *logger) TxTx(msg string, ctx ...interface{}) {
	l.write(msg, LvlTxTx, ctx)
}

func (l *logger) NewBlockRx(msg string, ctx ...interface{}) {
	l.write(msg, LvlNewBlockRx, ctx)
}

func (l *logger) NewBlockTx(msg string, ctx ...interface{}) {
	l.write(msg, LvlNewBlockTx, ctx)
}

func (l *logger) NewBlockHashesRx(msg string, ctx ...interface{}) {
	l.write(msg, LvlNewBlockHashesRx, ctx)
}

func (l *logger) NewBlockHashesTx(msg string, ctx ...interface{}) {
	l.write(msg, LvlNewBlockHashesTx, ctx)
}

func (l *logger) GetHandler() Handler {
	return l.h.Get()
}

func (l *logger) SetHandler(h Handler) {
	l.h.Swap(h)
}

func (l *logger) GetGlogger() *GlogHandler {
	return l.glogger
}

func (l *logger) SetGlogger(glogger *GlogHandler) {
	l.glogger = glogger
}

func normalize(ctx []interface{}) []interface{} {
	// if the caller passed a Ctx object, then expand it
	if len(ctx) == 1 {
		if ctxMap, ok := ctx[0].(Ctx); ok {
			ctx = ctxMap.toArray()
		}
	}

	// ctx needs to be even because it's a series of key/value pairs
	// no one wants to check for errors on logging functions,
	// so instead of erroring on bad input, we'll just make sure
	// that things are the right length and users can fix bugs
	// when they see the output looks wrong
	if len(ctx)%2 != 0 {
		ctx = append(ctx, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	return ctx
}

// Lazy allows you to defer calculation of a logged value that is expensive
// to compute until it is certain that it must be evaluated with the given filters.
//
// Lazy may also be used in conjunction with a Logger's New() function
// to generate a child logger which always reports the current value of changing
// state.
//
// You may wrap any function which takes no arguments to Lazy. It may return any
// number of values of any type.
type Lazy struct {
	Fn interface{}
}

// Ctx is a map of key/value pairs to pass as context to a log function
// Use this only if you really need greater safety around the arguments you pass
// to the logging functions.
type Ctx map[string]interface{}

func (c Ctx) toArray() []interface{} {
	arr := make([]interface{}, len(c)*2)

	i := 0
	for k, v := range c {
		arr[i] = k
		arr[i+1] = v
		i += 2
	}

	return arr
}
