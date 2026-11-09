package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/codesyariah/codenet-router/apps/api/internal/pppoeusers"
)

func main() {
	socket := flag.String("socket", "/run/codenet/pppoe-helper.sock", "lokasi Unix socket helper")
	secret := flag.String("secret-file", "/etc/codenet/helper.secret", "file secret IPC owner-only")
	chap := flag.String("chap-secrets", "/etc/codenet/chap-secrets", "file chap-secrets yang diizinkan")
	fixture := flag.Bool("fixture", false, "mode fixture development; hanya /var/lib/codenet-v04/helper-test dan non-root")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := pppoeusers.ServeWithConfig(ctx, pppoeusers.ServeConfig{SocketPath: *socket, SecretFile: *secret, ChapSecrets: *chap, Fixture: *fixture, SocketUID: -1, SocketGID: -1, SocketMode: 0o660}); err != nil {
		log.Fatal(err)
	}
}
