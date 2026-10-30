package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMockProviderTidakMengeksposPassword(t *testing.T) {
	p := NewMock()
	sessions, err := p.PPPoESessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) == 0 {
		t.Fatal("sesi mock harus tersedia")
	}
	if sessions[0].Username == "" || sessions[0].IP == "" {
		t.Fatal("data sesi mock tidak lengkap")
	}
	payload, err := json.Marshal(sessions)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(payload)), "password") {
		t.Fatal("respons sesi tidak boleh memiliki field password")
	}
}

func TestMockProviderInterfaceNotFound(t *testing.T) {
	_, err := NewMock().Interface(context.Background(), "tidak-ada")
	if err == nil {
		t.Fatal("mengharapkan error not found")
	}
	if _, ok := err.(NotFoundError); !ok {
		t.Fatalf("tipe error salah: %T", err)
	}
}
