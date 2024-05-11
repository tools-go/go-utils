package utils

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"go/build"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")

	rndPool = sync.Pool{
		New: func() interface{} {
			return rand.New(rand.NewSource(time.Now().UnixNano()))
		},
	}
)

// 获取随机长数字串id
func GenerateId() int64 {
	i, _ := strconv.ParseInt(RandomStrings(8, "1234567890"), 10, 64)
	return i
}

func StringsToInterfaces(src []string) []interface{} {
	ret := make([]interface{}, len(src))
	for i, v := range src {
		ret[i] = v
	}
	return ret
}

// stack returns a nicely formatted stack frame, skipping skip frames.
func Stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

func IsNil(src interface{}) bool {
	if src == nil {
		return true
	}
	switch reflect.TypeOf(src).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Interface:
		if reflect.ValueOf(src).IsNil() {
			return true
		}
	case reflect.Slice, reflect.Array:
		if reflect.ValueOf(src).IsNil() {
			return true
		}
	}
	return false
}

func GetFirstNotEmptyString(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}

func XMLDecode(data []byte, obj interface{}) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(obj)
}

func XMLEncode(obj interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := xml.NewEncoder(buf)
	err := encoder.Encode(obj)
	s := strings.Trim(buf.String(), "\n")
	return []byte(s), err
}

func ToInt64(value interface{}) (d int64) {
	val := reflect.ValueOf(value)
	switch value.(type) {
	case int, int8, int16, int32, int64:
		d = val.Int()

	case uint, uint8, uint16, uint32, uint64:
		d = int64(val.Uint())
	case string:
		d, _ = strconv.ParseInt(val.String(), 10, 64)
	case float64:
		d = int64(val.Float())
	}
	return
}

/*
四舍五入
f 待四舍五入数字
prec 小数点后prec位
*/
func Float64Round(f float64, prec int) float64 {
	pow10_n := math.Pow10(prec)
	return math.Trunc((f+0.5/pow10_n)*pow10_n) / pow10_n
}

/*
	截取float
	f 待截取数字
	prec 小数点后prec位
*/

func Float64Cut(f float64, prec int) float64 {
	pow10_n := math.Pow10(prec)
	return float64(int64(f*pow10_n)) / pow10_n
}

func Int64ToString(input int64) string {
	return strconv.FormatInt(input, 10)
}

func SliceCut(v interface{}, offset int, limit int) error {
	if limit <= 0 {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("second param none ptr")
	}
	if rv.Elem().Kind() != reflect.Slice {
		return errors.New("second param none ptr of slice")
	}
	if rv.Elem().Len() <= 0 {
		return nil
	}
	if offset < 0 {
		offset = rv.Elem().Len() + offset%rv.Elem().Len()
	} else {
		offset = offset % rv.Elem().Len()
	}
	endset := offset + limit
	if endset > rv.Elem().Len() {
		endset = rv.Elem().Len()
	}
	instead := rv.Elem().Slice(offset, endset)
	rv.Elem().Set(instead)
	return nil
}

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

var (
	ByteMap = map[float32]string{
		BYTE:     "B",
		KILOBYTE: "KB",
		MEGABYTE: "MB",
		GIGABYTE: "GB",
		TERABYTE: "TB",
	}

	BitPerSecMap = map[float32]string{
		BYTE:     "bps",
		KILOBYTE: "Kbps",
		MEGABYTE: "Mbps",
		GIGABYTE: "Gbps",
		TERABYTE: "Tbps",
	}
)

func StorageUnitConvert(bytes uint64, unitMap ...map[float32]string) string {
	units := ByteMap
	if len(unitMap) > 0 {
		units = unitMap[0]
	}
	value := float32(bytes)
	unit := ""
	switch {
	case bytes >= TERABYTE:
		unit = units[TERABYTE]
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = units[GIGABYTE]
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = units[MEGABYTE]
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = units[KILOBYTE]
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = units[BYTE]
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

// GetGOPATH get the env gopath
func GetGOPATH() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}

	return gopath
}

func CopyMap(originMap map[string]interface{}) map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range originMap {
		newMap[k] = v
	}
	return newMap
}

// 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}
