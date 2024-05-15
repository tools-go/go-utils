package trace

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func init() {
	zap.RegisterEncoder("fc", func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return NewFCLogEncoder(config), nil
	})
}

// For JSON-escaping; see jsonEncoder.safeAddString below.
const _hex = "0123456789abcdef"

var bufferpool = buffer.NewPool()

var _pool = sync.Pool{New: func() interface{} {
	return &FCLogEncoder{}
}}

func getFCLogEncoder() *FCLogEncoder {
	return _pool.Get().(*FCLogEncoder)
}

func putFCLogEncoder(enc *FCLogEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_pool.Put(enc)
}

type FCLogEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool // include spaces after colons and commas
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

// NewJSONEncoder creates a fast, low-allocation JSON encoder. The encoder
// appropriately escapes all field keys and values.
//
// Note that the encoder doesn't deduplicate keys, so it's possible to produce
// a message like
//
//	{"foo":"bar","foo":"baz"}
//
// This is permitted by the JSON specification, but not encouraged. Many
// libraries will ignore duplicate key-value pairs (typically keeping the last
// pair) when unmarshaling, but users should attempt to avoid adding duplicate
// keys.
func NewFCLogEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return newFCLogEncoder(cfg, false)
}

func newFCLogEncoder(cfg zapcore.EncoderConfig, spaced bool) *FCLogEncoder {
	return &FCLogEncoder{
		EncoderConfig: &cfg,
		buf:           bufferpool.Get(),
		spaced:        spaced,
	}
}

func (enc *FCLogEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *FCLogEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *FCLogEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *FCLogEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *FCLogEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *FCLogEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *FCLogEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *FCLogEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *FCLogEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *FCLogEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferpool.Get()
		enc.reflectEnc = json.NewEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

func (enc *FCLogEncoder) AddReflected(key string, obj interface{}) error {
	enc.resetReflectBuf()
	err := enc.reflectEnc.Encode(obj)
	if err != nil {
		return err
	}
	enc.reflectBuf.TrimNewline()
	enc.addKey(key)
	_, err = enc.buf.Write(enc.reflectBuf.Bytes())
	return err
}

func (enc *FCLogEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *FCLogEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *FCLogEncoder) AddTime(key string, val time.Time) {
	enc.AppendTime(val)
}

func (enc *FCLogEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *FCLogEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	enc.buf.AppendByte('[')
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendByte(']')
	return err
}

func (enc *FCLogEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	enc.buf.AppendByte('{')
	err := obj.MarshalLogObject(enc)
	enc.buf.AppendByte('}')
	return err
}

func (enc *FCLogEncoder) AppendBool(val bool) {
	enc.buf.AppendBool(val)
}

func (enc *FCLogEncoder) AppendByteString(val []byte) {
	// enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	// enc.buf.AppendByte('"')
}

func (enc *FCLogEncoder) AppendComplex128(val complex128) {
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *FCLogEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	enc.EncodeDuration(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *FCLogEncoder) AppendInt64(val int64) {
	enc.buf.AppendInt(val)
}

func (enc *FCLogEncoder) AppendReflected(val interface{}) error {
	enc.resetReflectBuf()
	err := enc.reflectEnc.Encode(val)
	if err != nil {
		return err
	}
	enc.reflectBuf.TrimNewline()
	_, err = enc.buf.Write(enc.reflectBuf.Bytes())
	return err
}

func (enc *FCLogEncoder) AppendString(val string) {
	enc.safeAddString(val)
}

func (enc *FCLogEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	enc.EncodeTime(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *FCLogEncoder) AppendUint64(val uint64) {
	enc.buf.AppendUint(val)
}

func (enc *FCLogEncoder) AddComplex64(k string, v complex64) {
	enc.AddComplex128(k, complex128(v))
}
func (enc *FCLogEncoder) AddFloat32(k string, v float32) { enc.AddFloat64(k, float64(v)) }
func (enc *FCLogEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *FCLogEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *FCLogEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *FCLogEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *FCLogEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *FCLogEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *FCLogEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *FCLogEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *FCLogEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *FCLogEncoder) AppendComplex64(v complex64)    { enc.AppendComplex128(complex128(v)) }
func (enc *FCLogEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *FCLogEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *FCLogEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *FCLogEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *FCLogEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *FCLogEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *FCLogEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *FCLogEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *FCLogEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *FCLogEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *FCLogEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

func (enc *FCLogEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *FCLogEncoder) clone() *FCLogEncoder {
	clone := getFCLogEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = bufferpool.Get()
	return clone
}

// EncodeEntry 组织日志输出
func (enc *FCLogEncoder) EncodeEntry(ent zapcore.Entry, fields []zap.Field) (*buffer.Buffer, error) {
	final := enc.clone()
	if final.LevelKey != "" {
		cur := final.buf.Len()
		final.EncodeLevel(ent.Level, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeLevel was a no-op. Fall back to strings to keep
			// output JSON valid.
			final.AppendString(ent.Level.String())
		}
	}
	if final.TimeKey != "" {
		final.AddTime(final.TimeKey, ent.Time)
	}
	if ent.Caller.Defined && final.CallerKey != "" {
		cur := final.buf.Len()
		final.EncodeCaller(ent.Caller, final)
		if cur == final.buf.Len() {
			final.AppendString(ent.Caller.String())
		}
	}
	final.buf.AppendByte(' ')
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].Key == LOGKEY_TAG {
			return true
		} else if fields[j].Key == LOGKEY_TAG {
			return false
		} else {
			return fields[i].Key < fields[j].Key
		}
	})
	idx := 0
	if len(fields) > 0 && fields[0].Key == LOGKEY_TAG {
		idx = 1
		final.buf.AppendString(fields[0].String)
		final.addElementSeparator()
	}
	if final.MessageKey != "" {
		final.buf.AppendString(fmt.Sprintf("_msg=%s", ent.Message))
	}
	if enc.buf.Len() > 0 {
		final.addElementSeparator()
		final.buf.Write(enc.buf.Bytes())
	}
	addFields(final, fields[idx:])

	final.closeOpenNamespaces()
	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}
	if final.LineEnding != "" {
		final.buf.AppendString(final.LineEnding)
	} else {
		final.buf.AppendString(zapcore.DefaultLineEnding)
	}
	ret := final.buf
	putFCLogEncoder(final)
	return ret, nil
}

func (enc *FCLogEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
}

func (enc *FCLogEncoder) addKey(key string) {
	enc.addElementSeparator()
	enc.safeAddString(key)
	enc.buf.AppendByte('=')
	if enc.spaced {
		enc.buf.AppendByte(' ')
	}
}

func (enc *FCLogEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendByte('|')
		enc.buf.AppendByte('|')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *FCLogEncoder) appendFloat(val float64, bitSize int) {
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *FCLogEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *FCLogEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *FCLogEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '"':
		enc.buf.AppendByte(b)
	case '\\':
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *FCLogEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(LOG_TIME_FORMAT))
}

func LevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("[%s]", l.CapitalString()))
}

func DurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%vms", float64(d)/float64(time.Millisecond)))
}

func CallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	_, file, line, ok := caller.PC, caller.File, caller.Line, caller.Defined
	for i := 6; i < 15; i++ {
		if ok {
			_, file, line, ok = runtime.Caller(i)
			if !strings.Contains(file, "log/logger.go") {
				break
			}
		}
	}
	c := fmt.Sprintf("%s:%d", file, line)

	enc.AppendString(fmt.Sprintf("[%s]", c))
}
