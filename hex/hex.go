package hex

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
)

// hexifySingle creates a single hex digit from a number 0 to 15
// Note that there are no error checks, so don't misuse it
func hexifySingle(num uint8) string {
	switch num {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
		return strconv.Itoa(int(num))
	case 10:
		return "a"
	case 11:
		return "b"
	case 12:
		return "c"
	case 13:
		return "d"
	case 14:
		return "e"
	case 15:
		return "f"
	default:
		println("Programmer error on use of `hexifySingle()`")
		os.Exit(999)         // TODO: Make note of 999 as the exit code for programmer errors
		return "unreachable" // TODO: Refactor so this isn't needed, even though it's unreachable
	}
}

func hexify(b byte) string {
	divCount := 2
	//radix := 16
	var res string

	for i := divCount - 1; i >= 0; i-- {
		placeVal := int(1) << uint(4*i)
		if int(b) >= placeVal && placeVal > 0 { // If remaining value is no smaller than value of current radix "place"
			res += hexifySingle(byte(int(b) / placeVal)) // Count of current "place" in new radix
			b = byte(int(b) % placeVal)                  // Subtract extracted place value from remaining value
		} else {
			res += "0" // Absolutely necessary to prevent lost digits on pairs under 16
		}
	}

	return res
}

// GetRand generates a random hex value `len` digits long
func GetRand(len uint8) string {
	var randLen uint8
	if len%2 == 0 {
		randLen = len / 2
	} else {
		randLen = len/2 + 1
	}

	bs := make([]byte, randLen) // TODO: Check if this is producing a value of the correct length given rounding
	_, err := rand.Read(bs)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1236) // TODO: Fix arbitrary use of error code 1236 for random number generation errors
	}

	var res string
	for i, b := range bs {
		// If this is the last of an odd-digited hex, convert it to only one hex character
		if i == int(randLen-1) && len%2 == 1 {
			res += hexifySingle(b % 16)
			// Otherwise, convert it to two
		} else {
			res += hexify(b)
		}
	}

	return res
}
