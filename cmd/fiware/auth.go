package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"golang.org/x/term"
)

// Auth inicia sesión y vuelca el token por consola
func auth(c *cli.Context, store *config.Store) error {
	if err := store.Read(); err != nil {
		return err
	}
	if store.Current.Name == "" {
		return errors.New("no contexts defined")
	}
	selected := store.Current
	if selected.KeystoneURL == "" || selected.Service == "" || selected.Username == "" {
		return errors.New("current context is not properly configured")
	}
	k, err := keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
	if err != nil {
		return err
	}
	var console io.Writer = os.Stdout
	isTerminal := term.IsTerminal(int(syscall.Stdout))
	if !isTerminal {
		// Display prompt through stderr, so stdout is only used
		// for token.
		console = os.Stderr
	}
	fmt.Fprintf(console, "Environment: %s\n", k.LoginURL.String())
	fmt.Fprintf(console, "Username@Service: %s@%s\n", k.Username, k.Service)
	fmt.Fprint(console, "Password: ")
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	if isTerminal {
		fmt.Println() // in case the stdin is a terminal
	}
	if err != nil {
		return err
	}
	token, err := k.Login(http.DefaultClient, string(bytepw))
	if err != nil {
		return err
	}
	if isTerminal {
		if runtime.GOOS == "windows" {
			fmt.Printf("SET FIWARE_TOKEN=%s\n", token)
		} else {
			fmt.Printf("export FIWARE_TOKEN=%s\n", token)
		}
	} else {
		fmt.Println(token)
	}
	return nil
}
