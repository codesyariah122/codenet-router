//go:build linux

package providers

import (
	"testing"
)

func TestParseIPRoutes(t *testing.T) {
	data := []byte(`[{"dst":"10.50.0.0/24","dev":"enp0s8","protocol":"kernel","scope":"link","table":254,"type":"unicast"},{"gateway":"10.0.2.2","dev":"enp0s3","metric":100,"protocol":"dhcp","table":"main","type":"unicast"}]`)
	routes, err := parseIPRoutes(data, 4)
	if err != nil {
		t.Fatalf("parseIPRoutes() error = %v", err)
	}
	if len(routes) != 2 || routes[0].Destination != "10.50.0.0/24" || routes[1].Destination != "default" || routes[1].Metric != 100 || routes[1].Table != "main" {
		t.Fatalf("routes = %#v", routes)
	}
}

func TestParseIPv6DefaultRoute(t *testing.T) {
	routes, err := parseIPRoutes([]byte(`[{"gateway":"fe80::1","dev":"enp0s3","protocol":"ra","scope":"global","table":254}]`), 6)
	if err != nil || len(routes) != 1 {
		t.Fatalf("routes = %#v, err = %v", routes, err)
	}
	if routes[0].Destination != "::/0" || routes[0].Version != 6 {
		t.Fatalf("route = %#v", routes[0])
	}
}

func TestParseIPRoutesRejectsInvalidJSON(t *testing.T) {
	if _, err := parseIPRoutes([]byte("not-json"), 4); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestParseIPRoutesEmptyTable(t *testing.T) {
	routes, err := parseIPRoutes([]byte(`[]`), 4)
	if err != nil || len(routes) != 0 {
		t.Fatalf("routes = %#v, err = %v", routes, err)
	}
}
