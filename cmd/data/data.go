package data

import (
	"bytes"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	LetterIdxBits = 6                    // 6 bits to represent a letter index
	LetterIdxMask = 1<<LetterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	LetterIdxMax  = 63 / LetterIdxBits   // # of letter indices fitting in 63 bits
)

var NonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z]+`)
var NonNumericRegex = regexp.MustCompile(`[^0-9]+`)
var RandSource = rand.NewSource(time.Now().UTC().UnixNano())

func Cleanse(in string) string {
	return strings.ReplaceAll(in, `
`, `\n`)
}

func Base64(in string) string {
	b64String := base64.StdEncoding.EncodeToString([]byte(in))
	return b64String
}

func Base64JSON(in map[string]string) string {
	jsonBytes, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	b64String := base64.StdEncoding.EncodeToString(jsonBytes)
	return b64String
}

func ReplaceAllCaseInsensitive(input, old, new string) string {
	// Create a regular expression pattern for the old string
	pattern := "(?i)" + regexp.QuoteMeta(old)
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// If the regular expression couldn't be compiled, return the input string
		return input
	}
	return regex.ReplaceAllStringFunc(input, func(match string) string {
		if strings.EqualFold(match, old) {
			return new
		}
		return match
	})
}

func HumanReadableFileMode(fm fs.FileMode) (string, error) {
	if !fm.IsRegular() && !fm.IsDir() {
		return "", errors.New("unsupported file mode")
	}

	var mask string

	if fm.IsDir() {
		mask = "d"
	} else {
		mask = "-"
	}

	for i := 0; i < 3; i++ {
		shift := uint(6 - (i * 3))
		permissions := (fm >> shift) & 7

		if permissions&4 != 0 {
			mask += "r"
		} else {
			mask += "-"
		}

		if permissions&2 != 0 {
			mask += "w"
		} else {
			mask += "-"
		}

		if permissions&1 != 0 {
			mask += "x"
		} else {
			mask += "-"
		}
	}

	return mask, nil
}

func PadNumber(number string) string {
	num, err := strconv.Atoi(number)
	if err != nil {
		return number
	}

	if num > 999 {
		return number
	}

	if num > 99 {
		return fmt.Sprintf("%04d", num*10)
	}

	if num > 9 {
		return fmt.Sprintf("%04d", num*100)
	}

	return fmt.Sprintf("%04d", num*1000)
}

func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func WrapText(text string, maxWidth int) string {
	wrapped := ""
	for i := 0; i < len(text); i += maxWidth {
		if i+maxWidth > len(text) {
			wrapped += text[i:]
		} else {
			wrapped += text[i:i+maxWidth] + "\n"
		}
	}
	return wrapped
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func DuplicateString(s string, n int) string {
	var b []byte
	for i := 0; i < n; i++ {
		d := []byte(s)
		b = append(b, d[0])
	}
	return fmt.Sprint(string(b))
}

func MergeStringSlices(slices ...[]string) []string {
	var output []string
	for s := 0; s < len(slices); s++ {
		slice := slices[s]
		output = append(output, slice...)
	}
	return output
}

func MergeByteSlices(slices ...[]byte) []byte {
	var output []byte
	for s := 0; s < len(slices); s++ {
		slice := slices[s]
		output = append(output, slice...)
	}
	return output
}

func BB(s string) (b []byte) {
	return bytes.NewBufferString(s).Bytes()
}

func ReplaceNumbers(str string, repl string) string {
	return NonAlphanumericRegex.ReplaceAllString(str, repl)
}

func RemoveNumbers(str string) string {
	return ReplaceNumbers(str, "")
}

func ReplaceNonNumbers(str string, repl string) string {
	return NonNumericRegex.ReplaceAllString(str, repl)
}

func KeepNumbers(str string) string {
	return ReplaceNonNumbers(str, "")
}

func HasNumber(str string) bool {
	num := KeepNumbers(str)
	return len(num) > 0
}

func ExtractNumbers(str string) int {
	out, err := strconv.Atoi(ReplaceNonNumbers(str, ""))
	if err != nil {
		log.Println(fmt.Errorf("failed to extract numbers out of string provided: %v", err))
	}
	return out
}

func RandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, RandSource.Int63(), LetterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = RandSource.Int63(), LetterIdxMax
		}
		if idx := int(cache & LetterIdxMask); idx < len(LetterBytes) {
			sb.WriteByte(LetterBytes[idx])
			i--
		}
		cache >>= LetterIdxBits
		remain--
	}

	return sb.String()
}

func IsYes(s string) bool {
	return strings.EqualFold(s, "yes") ||
		strings.EqualFold(s, "1") ||
		strings.EqualFold(s, "ye") ||
		strings.EqualFold(s, "y") ||
		strings.EqualFold(s, "ja") ||
		strings.EqualFold(s, "da") ||
		strings.EqualFold(s, "si") ||
		strings.EqualFold(s, "oui")
}

func RandomBytes() []byte {
	str := RandomString(RandomInt(999))
	return bytes.NewBufferString(str).Bytes()
}

func ChunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

func Prepend(existing []any, items ...any) (out []any) {
	return append(append(out, items...), existing...)
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ContainsInt(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func Fib() func() int {
	a, b := 0, 1
	return func() int {
		a, b = b, a+b
		return b
	}
}

// HasNextIdx is a wrapped safety check for a slice to ensure if an action is about to occur on an index, that the index exists first as a bool
//
//lint:ignore U1000 unused currently
//goland:noinspection GoUnusedFunction
func HasNextIdx(s []string, i int) bool {
	return len(s) > i
}

// ReplaceIfInside looks inside a slice of strings for a given string
func ReplaceIfInside(insideThis []string, lookingAt string, replaceLookingForWith string) (string, bool) {
	newResult := make([]string, len(insideThis))
	inside := false
	for _, protected := range insideThis {
		if len(protected) < 1 {
			continue
		}
		if strings.Contains(lookingAt, protected) {
			inside = true
			newResult = append(newResult,
				strings.Replace(protected, lookingAt, replaceLookingForWith, len(insideThis)))
		} else {
			newResult = append(newResult, lookingAt)
		}
	}
	return strings.Join(newResult, ""), inside
}

// ReplaceInside searches for f inside s and replaces f with r and returns a modified s along with a bool if it worked
//
//lint:ignore U1000 unused currently
//goland:noinspection GoUnusedFunction
func ReplaceInside(s []string, f string, r string) ([]string, bool) {
	at, ok := AtInside(s, f)
	if ok {
		if len(s) > at {
			s[at] = strings.Replace(s[at], f, r, len(s))
		}
	}
	return s, ok
}

// AtInside returns the index location where f string exists inside s slice of strings
func AtInside(s []string, f string) (int, bool) {
	for i := 0; i < len(s); i++ {
		if strings.Contains(s[i], f) {
			return i, true
		}
	}
	return 0, false
}

// IsInside acts like strings.Contains but lets you pass in a slice of strings instead of just a string
func IsInside(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(a, e) {
			return true
		}
	}
	return false
}

// RandomInt generates a cryptographically safe random integer
func RandomInt(limit int) int {
	c := make(chan int, 1)
	go func(l int) int {
		// Seed the rand engine
		rand.Seed(time.Now().UTC().UnixNano())

		if l <= 0 {
			return 0
		}
		newInt, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(l)))
		if err != nil {
			return l
		}
		data := int(newInt.Int64())
		c <- data
		return data
	}(limit)
	return <-c
}

// RandomRangeInt return a random int that is between start and limit, will recursively run until match found
func RandomRangeInt(start int, limit int) int {
	i := RandomInt(limit)
	if i >= start && i <= limit {
		return i
	} else {
		return RandomRangeInt(start, limit) // retry
	}
}

func RemoveDirectorySlashes(directory string) string {
	directory = strings.ReplaceAll(directory, `\\\\`, `\\`) // go will escape these slashes
	directory = strings.ReplaceAll(directory, `\\`, `\`)    // go will escape these slashes
	return directory
}

func CombineSlices(first []string, second []string) []string {
	secondT := len(second)
	firstT := len(first)
	if firstT > secondT {
		for idx := 0; idx < secondT; idx++ {
			first = append(first, second[idx])
		}
		return first
	} else {
		for idx := 0; idx < firstT; idx++ {
			second = append(second, first[idx])
		}
		return second
	}
}
