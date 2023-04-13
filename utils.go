package main

import "strings"

func removeHexPrefix(str string) string {
	return strings.Replace(str, "0x", "", -1)
}
