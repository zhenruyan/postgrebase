package tygoja

import (
	"fmt"
	"strconv"
	"strings"
)

func isProbablyIotaType(groupType string) bool {
	groupType = strings.Trim(groupType, "()")
	return groupType == "iota" || strings.HasPrefix(groupType, "iota +") || strings.HasSuffix(groupType, "+ iota")
}

func basicIotaOffsetValueParse(groupType string) (int, error) {
	if !isProbablyIotaType(groupType) {
		panic("can't parse non-iota type")
	}

	groupType = strings.Trim(groupType, "()")
	if groupType == "iota" {
		return 0, nil
	}
	parts := strings.Split(groupType, " + ")

	var numPart string
	if parts[0] == "iota" {
		numPart = parts[1]
	} else {
		numPart = parts[0]
	}

	addValue, err := strconv.ParseInt(numPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Failed to guesstimate initial iota value for \"%s\": %w", groupType, err)
	}

	return int(addValue), nil
}
