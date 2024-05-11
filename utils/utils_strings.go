package utils

import (
	"bytes"
	"math/rand"
	"strings"
	"unsafe"

	"github.com/iancoleman/strcase"
)

// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// Deprecated, use github.com/duke-git/lancet/v2/slice instead.
func StringArrayUnique(m []string) []string {
	d := make([]string, 0)
	tempMap := make(map[string]bool, len(m))
	for _, v := range m {
		if !tempMap[v] {
			tempMap[v] = true
			d = append(d, v)
		}
	}
	return d
}

// Deprecated, use github.com/duke-git/lancet/v2/slice instead.
func StringArrayContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Deprecated, use github.com/duke-git/lancet/v2/slice instead.
func StringContainsOne(src string, items ...string) bool {
	for _, item := range items {
		if strings.Contains(src, item) {
			return true
		}
	}
	return false
}

// Deprecated, use github.com/duke-git/lancet/v2/slice instead.
func StringArrayDiff(m, n []string) []string {
	d := make([]string, 0)
	tempMap := make(map[string]bool, len(m))
	for _, v := range m {
		if !tempMap[v] {
			tempMap[v] = true
		}
	}

	for _, v := range n {
		if tempMap[v] {
			delete(tempMap, v)
		}
	}

	for v := range tempMap {
		d = append(d, v)
	}

	return d
}

func StringReplace(base, oldS, newS string, n int) string {
	if n == 0 {
		return base
	}
	b := []byte(base)
	old := []byte(oldS)
	new := []byte(newS)
	if len(oldS) < len(newS) {
		result := bytes.Replace(b, old, new, n)
		return string(result)
	}

	if n < 0 {
		n = len(b)
	}

	var wid, i, j, w int
	for i, j = 0, 0; i < len(b) && j < n; j++ {
		wid = bytes.Index(b[i:], old)
		if wid < 0 {
			break
		}

		w += copy(b[w:], b[i:i+wid])
		w += copy(b[w:], new)
		i += wid + len(old)
	}

	w += copy(b[w:], b[i:])
	return string(b[0:w])
}

func ByteReplace(s, old, new []byte, n int) []byte {
	if n == 0 {
		return s
	}

	if len(old) < len(new) {
		return bytes.Replace(s, old, new, n)
	}

	if n < 0 {
		n = len(s)
	}

	var wid, i, j, w int
	for i, j = 0, 0; i < len(s) && j < n; j++ {
		wid = bytes.Index(s[i:], old)
		if wid < 0 {
			break
		}

		w += copy(s[w:], s[i:i+wid])
		w += copy(s[w:], new)
		i += wid + len(old)
	}

	w += copy(s[w:], s[i:])
	return s[0:w]
}

func RandomStrings(length int, from string) string {
	l := len(from)
	if l < length {
		return ""
	}

	ret := make([]byte, length)
	rnd := rndPool.Get().(*rand.Rand)
	defer rndPool.Put(rnd)

	for i := 0; i < length; i++ {
		pos := rnd.Intn(l)
		ret[i] = from[pos]
	}

	return string(ret)
}

// RandomPassword 生成随机密码
// pwdLength 为密码长度，pwdCodes为密码生成字符范围，不传使用默认值
func RandomPassword(pwdLength int, pwdCodes ...byte) string {
	if len(pwdCodes) < pwdLength {
		pwdCodes = []byte{
			'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
			'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
			'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
			'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '+', '-', '=',
			'~', '!', '?', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_',
			';', '.', ':', '<', '>',
		}
	}

	pwd := make([]byte, pwdLength)
	rnd := rndPool.Get().(*rand.Rand)
	defer rndPool.Put(rnd)

	for j := 0; j < pwdLength; j++ {
		index := rnd.Int() % len(pwdCodes)

		pwd[j] = pwdCodes[index]
	}

	return string(pwd)
}

// Converts a string to CamelCase
func ToCamel(s string) string {
	return strcase.ToCamel(s)
}

// Converts a string to lowerCamelCase
func ToLowerCamel(s string) string {
	return strcase.ToLowerCamel(s)
}

// Converts a string to snake_case
func ToSnake(s string) string {
	return ToDelimited(s, '_')
}

// Converts a string to SCREAMING_SNAKE_CASE
func ToScreamingSnake(s string) string {
	return ToScreamingDelimited(s, '_', true)
}

// Converts a string to kebab-case
func ToKebab(s string) string {
	return ToDelimited(s, '-')
}

// Converts a string to SCREAMING-KEBAB-CASE
func ToScreamingKebab(s string) string {
	return ToScreamingDelimited(s, '-', true)
}

// Converts a string to delimited.snake.case (in this case `del = '.'`)
func ToDelimited(s string, del uint8) string {
	return ToScreamingDelimited(s, del, false)
}

// Converts a string to SCREAMING.DELIMITED.SNAKE.CASE (in this case `del = '.'; screaming = true`) or delimited.snake.case (in this case `del = '.'; screaming = false`)
func ToScreamingDelimited(s string, del uint8, screaming bool) string {
	return strcase.ToScreamingDelimited(s, del, "", screaming)
}

// Remove blank lines from a string
func TrimBlankLines(text string) string {
	lines := strings.Split(text, "\n")
	nonBlankLines := []string{}

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonBlankLines = append(nonBlankLines, line)
		}
	}

	return strings.Join(nonBlankLines, "\n")
}
