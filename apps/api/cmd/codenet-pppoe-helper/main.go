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
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := pppoeusers.Serve(ctx, *socket, *secret, *chap); err != nil {
		log.Fatal(err)
	}
}
