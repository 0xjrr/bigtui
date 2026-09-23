package format

import (
	"fmt"
	"strings"
)

var byteUnits = []string{"KB", "MB", "GB", "TB"}

func Bytes(value int64) string {
	if value < 1000 {
		return fmt.Sprintf("%d B", value)
	}
	amount := float64(value)
	for _, unit := range byteUnits {
		amount /= 1000
		if amount < 1000 {
			return fmt.Sprintf("%.1f %s", amount, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", amount/1000)
}

func Labels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for key, value := range labels {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ", ")
}

func ValidationError(err error) string {
	message := strings.TrimPrefix(err.Error(), "googleapi: Error 400: ")
	return strings.TrimSpace(message)
}
