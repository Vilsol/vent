package utils

import "fmt"

func BytesToHex(b []byte) string {
	hex := ""
	chars := ""

	for _, a := range b {
		hex += fmt.Sprintf("%#4x", a) + " "
		chars += fmt.Sprintf("%s", safeChar(a))
	}

	return hex + chars
}

func safeChar(char byte) string {
	if char <= 0x1F {
		return "."
	}

	return string(char)
}
