/*
from list of 62 char choose 7 chars for short url (62^7 = 3.5 trillion possible combinations)
-> 62 characters (a-z, A-Z, 0-9) for short URL generation (note: no symbols since they could have conflicting meaning)
*/
package main

import (
	"math/rand/v2"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const base = uint64(62) //base for encoding/decoding

func encode(num uint64) string {
	var result string
	if num == 0 {
		return string(charset[0]) //return a for 0
	}
	for num > 0 {
		//we want to get the the last digit in num in base 62, which is num % 62
		//instead of looping through we can dir calc the index to append to result string
		index := num % base
		result = string(charset[index]) + result
		nextNum := num / base //remainder after removing last digit in base 62
		num = nextNum
	}
	return result
}

func GenerateCode() string {
	var min uint64 = 62 * 62 * 62 * 62 * 62 * 62 * 62
	var max uint64 = (62 * 62 * 62 * 62 * 62 * 62 * 62 * 62) - 1 //max is 62^8 - 1 since we want 7 chars
	randNum := min + rand.Uint64N(max-min+1)                     //generate random number between min and max (inclusive)
	encoded := encode(randNum)
	return encoded[:7] //first 7 chars
}
