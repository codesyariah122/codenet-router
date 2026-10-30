package providers

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

// parseMemInfo memisahkan parsing data kernel dari akses filesystem agar dapat diuji lintas platform.
func parseMemInfo(data []byte) map[string]uint64 {
	values := map[string]uint64{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimSuffix(fields[0], ":")] = value * 1024
	}
	return values
}

func parseCPUStat(data []byte) (busy, total uint64, ok bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() {
		return 0, 0, false
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		values = append(values, value)
	}
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	if total < idle {
		return 0, 0, false
	}
	return total - idle, total, true
}
