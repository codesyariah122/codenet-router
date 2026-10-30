package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/auth"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	databasePath := flags.String("db", env("CODENET_DB_PATH", "data/codenet.db"), "path database autentikasi")
	username := flags.String("username", "", "username administrator")
	passwordStdin := flags.Bool("password-stdin", false, "baca password dan konfirmasi dari dua baris stdin")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	store, err := auth.OpenStore(*databasePath)
	if err != nil {
		fatal("Database tidak dapat dibuka", err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch command {
	case "init":
		if *username == "" {
			fatal("--username wajib diisi", nil)
		}
		password := mustReadPassword(*passwordStdin)
		if err := store.CreateAdmin(ctx, *username, password); err != nil {
			fatal("Inisialisasi administrator gagal", err)
		}
		fmt.Println("Administrator CODENET berhasil diinisialisasi.")
	case "reset-password":
		if *username == "" {
			fatal("--username wajib diisi", nil)
		}
		password := mustReadPassword(*passwordStdin)
		if err := store.UpdatePassword(ctx, *username, password); err != nil {
			fatal("Reset password gagal", err)
		}
		fmt.Println("Password administrator berhasil diubah dan seluruh session lama dibatalkan.")
	case "status":
		count, err := store.AdminCount(ctx)
		if err != nil {
			fatal("Status administrator tidak dapat dibaca", err)
		}
		if count == 0 {
			fmt.Println("Administrator belum diinisialisasi.")
			return
		}
		fmt.Println("Administrator sudah diinisialisasi.")
	default:
		usage()
		os.Exit(2)
	}
}

func mustReadPassword(fromStdin bool) string {
	var first, second string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		first = readHidden("Password baru: ")
		second = readHidden("Konfirmasi password: ")
	} else if fromStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			first = scanner.Text()
		}
		if scanner.Scan() {
			second = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fatal("Password tidak dapat dibaca", err)
		}
	} else {
		fatal("stdin bukan terminal; gunakan --password-stdin dengan dua baris input", nil)
	}
	if first == "" || first != second {
		fatal("Password dan konfirmasi tidak cocok", nil)
	}
	return first
}

func readHidden(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	value, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fatal("Password tidak dapat dibaca", err)
	}
	return string(value)
}

func usage() {
	fmt.Fprintln(os.Stderr, "Penggunaan: codenet-admin <init|reset-password|status> [--db path] [--username nama] [--password-stdin]")
}

func fatal(message string, err error) {
	if err != nil && !errors.Is(err, auth.ErrInvalidCredentials) {
		fmt.Fprintf(os.Stderr, "%s: %s\n", message, strings.TrimSpace(err.Error()))
	} else {
		fmt.Fprintln(os.Stderr, message)
	}
	os.Exit(1)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
