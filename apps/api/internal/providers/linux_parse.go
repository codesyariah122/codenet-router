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
