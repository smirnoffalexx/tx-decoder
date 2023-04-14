package main

import "strings"

func removeHexPrefix(str string) string {
	return strings.Replace(str, "0x", "", -1)
}

func mapKeysToSlice(mapping map[string]string) []string {
	values := []string{}
	for key, _ := range mapping {
		values = append(values, key)
	}
	return values
}

func parseEvent(event string) (string, []string) {
	event = strings.Replace(event, ")", "", -1) // remove ")"
	eventSplitted := strings.Split(event, "(")  // split eventName and args
	name := eventSplitted[0]
	args := strings.Split(eventSplitted[1], ",")

	return name, args
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
