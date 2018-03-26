package log

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	nanoTermTimeFormat = "2006-01-02T15:04:05.999999999-0700"
	floatFormat        = 'f'
)

// locationTrims are trimmed for display to avoid unwieldy log lines.
var locationTrims = []string{
	"github.com/teamnsrg/go-ethereum/",
}

// PrintOrigins sets or unsets log location (file:line) printing for terminal
// format output.
func PrintOrigins(print bool) {
	if print {
		atomic.StoreUint32(&locationEnabled, 1)
	} else {
		atomic.StoreUint32(&locationEnabled, 0)
	}
}

// locationEnabled is an atomic flag controlling whether the terminal formatter
// should append the log locations too when printing entries.
var locationEnabled uint32

// locationLength is the maxmimum path length encountered, which all logs are
// padded to to aid in alignment.
var locationLength uint32

type Format interface {
	Format(r *Record) []byte
}

// FormatFunc returns a new Format object which uses
// the given function to perform record formatting.
func FormatFunc(f func(*Record) []byte) Format {
	return formatFunc(f)
}

type formatFunc func(*Record) []byte

func (f formatFunc) Format(r *Record) []byte {
	return f(r)
}

// TerminalStringer is an analogous interface to the stdlib stringer, allowing
// own types to have custom shortened serialization formats when printed to the
// screen.
type TerminalStringer interface {
	TerminalString() string
}

// TerminalFormat formats log records optimized for human readability on
// a terminal with color-coded level output and terser human friendly timestamp.
// This format should only be used for interactive programs or while developing.
func TerminalFormat(usecolor bool) Format {
	return FormatFunc(func(r *Record) []byte {
		var color = 0
		if usecolor {
			switch r.Lvl {
			case LvlCrit:
				color = 35
			case LvlError:
				color = 31
			case LvlWarn:
				color = 33
			case LvlInfo:
				color = 32
			case LvlDebug:
				color = 36
			case LvlTrace:
				color = 34
			default:
				color = 37
			}
		}

		b := &bytes.Buffer{}
		lvl := r.Lvl.AlignedString()

		unixTime := strconv.FormatFloat(float64(r.Time.UnixNano())/1000000000, floatFormat, 6, 64)

		if atomic.LoadUint32(&locationEnabled) != 0 {
			// Log origin printing was requested, format the location path and line number
			location := fmt.Sprintf("%+v", r.Call)
			for _, prefix := range locationTrims {
				location = strings.TrimPrefix(location, prefix)
			}
			// Maintain the maximum location length for fancyer alignment
			align := int(atomic.LoadUint32(&locationLength))
			if align < len(location) {
				align = len(location)
				atomic.StoreUint32(&locationLength, uint32(align))
			}

			// Assemble and print the log heading
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m|%s|%s|[%s]|%s", color, lvl, r.Time.Format(nanoTermTimeFormat), unixTime, location, r.Msg)
			} else {
				fmt.Fprintf(b, "%s|%s|%s|[%s]|%s", lvl, r.Time.Format(nanoTermTimeFormat), unixTime, location, r.Msg)
			}
		} else {
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m|%s|%s|%s", color, lvl, r.Time.Format(nanoTermTimeFormat), unixTime, r.Msg)
			} else {
				fmt.Fprintf(b, "%s|%s|%s|%s", lvl, r.Time.Format(nanoTermTimeFormat), unixTime, r.Msg)
			}
		}
		// print the keys logfmt style
		logfmt(b, r.Ctx, color, true)
		return b.Bytes()
	})
}

// LogfmtFormat prints records in logfmt format, an easy machine-parseable but human-readable
// format for key/value pairs.
//
// For more details see: http://godoc.org/github.com/kr/logfmt
//
func LogfmtFormat() Format {
	return FormatFunc(func(r *Record) []byte {
		common := []interface{}{r.KeyNames.Time, r.Time, r.KeyNames.Lvl, r.Lvl, r.KeyNames.Msg, r.Msg}
		buf := &bytes.Buffer{}
		logfmt(buf, append(common, r.Ctx...), 0, false)
		return buf.Bytes()
	})
}

func logfmt(buf *bytes.Buffer, ctx []interface{}, color int, term bool) {
	for i := 0; i < len(ctx); i += 2 {
		buf.WriteByte('|')

		k, ok := ctx[i].(string)
		v := formatLogfmtValue(ctx[i+1], term)
		if !ok {
			k, v = errorKey, formatLogfmtValue(k, term)
		}

		if color > 0 {
			fmt.Fprintf(buf, "\x1b[%dm%s\x1b[0m=", color, k)
		} else {
			buf.WriteString(k)
			buf.WriteByte('=')
		}
		buf.WriteString(v)
	}
	buf.WriteByte('\n')
}

func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return float64(v.UnixNano()) / 1000000000

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

// formatValue formats a value for serialization
func formatLogfmtValue(value interface{}, term bool) string {
	if value == nil {
		return "nil"
	}

	if t, ok := value.(time.Time); ok {
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return strconv.FormatFloat(float64(t.UnixNano())/1000000000, floatFormat, 6, 64)
	}
	if term {
		if s, ok := value.(TerminalStringer); ok {
			// Custom terminal stringer provided, use that
			return escapeString(s.TerminalString())
		}
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 6, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 6, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

var stringBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func escapeString(s string) string {
	needsQuotes := false
	needsEscape := false
	for _, r := range s {
		if r < ' ' || r == '=' || r == '"' {
			needsQuotes = true
		}
		if r == '\\' || r == '"' || r == '\n' || r == '\r' || r == '\t' {
			needsEscape = true
		}
	}
	if !needsEscape && !needsQuotes {
		return s
	}
	e := stringBufPool.Get().(*bytes.Buffer)
	e.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			e.WriteByte('\\')
			e.WriteByte(byte(r))
		case '\n':
			e.WriteString("\\n")
		case '\r':
			e.WriteString("\\r")
		case '\t':
			e.WriteString("\\t")
		default:
			e.WriteRune(r)
		}
	}
	e.WriteByte('"')
	var ret string
	if needsQuotes {
		ret = e.String()
	} else {
		ret = string(e.Bytes()[1 : e.Len()-1])
	}
	e.Reset()
	stringBufPool.Put(e)
	return ret
}
