package strBits

import (
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomStr Iterate over the chars in aDestStr, converting them to
// random chars chosen from aRandSource.
func GenerateRandomStr( aRandSource string, aDestStr string ) string {
	theRandLen := len(aRandSource)
	theDestAsRunes := []rune(aDestStr)
	theRandAsRunes := []rune(aRandSource)
	for k := range theDestAsRunes {
		idx := rand.Intn(theRandLen)
		theDestAsRunes[k] = theRandAsRunes[idx]
	}
	return string(theDestAsRunes)
}

const Base64Charset = "/.ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// UrlSafeRandomStr Random string with just ".", "0 thru 9", and "A-Z,a-z".
//
// Pass in 0 for "default length" which is 16.
func UrlSafeRandomStr( aLen int ) string {
	theRandSource := Base64Charset[1:]
	// min length is 1, default to 16 if less than 1
	if aLen < 1 {
		aLen = 16
	}
	return GenerateRandomStr(theRandSource, strings.Repeat(".", aLen))
}

// Base64RandomSalt Random string with the Base64Charset characters.
//
// Pass in 0 for "default length" which is 16.
func Base64RandomSalt( aLen int ) string {
	// min length is 1, default to 16 if less than 1
	if aLen < 1 {
		aLen = 16
	}
	return GenerateRandomStr(Base64Charset, strings.Repeat(".", aLen))
}
