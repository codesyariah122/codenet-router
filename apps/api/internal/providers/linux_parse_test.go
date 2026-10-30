package providers

import "testing"

func TestParseMemInfo(t *testing.T) {
	values := parseMemInfo([]byte("MemTotal:       2048000 kB\nMemAvailable:   1024000 kB\nInvalid: nope kB\n"))
	if values["MemTotal"] != 2048000*1024 {
		t.Fatalf("MemTotal salah: %d", values["MemTotal"])
	}
	if values["MemAvailable"] != 1024000*1024 {
		t.Fatalf("MemAvailable salah: %d", values["MemAvailable"])
	}
	if _, exists := values["Invalid"]; exists {
		t.Fatal("nilai invalid tidak boleh diparsing")
	}
}
