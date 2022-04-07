package main

import (
	"errors"
	"fmt"
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
	fmt.Fprintf(os.Stderr, "Environment: %s\n", k.URL.String())
	fmt.Fprintf(os.Stderr, "Username@Service: %s@%s\n", k.Username, k.Service)
	fmt.Fprint(os.Stderr, "Password: ")
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr, "")
	if err != nil {
		return err
	}
	token, err := k.Login(httpClient(), string(bytepw))
	if err != nil {
		return err
	}
	if c.Bool(saveFlag.Name) {
		if err := store.Set([]string{"token", token}); err != nil {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		fmt.Printf("SET FIWARE_TOKEN=%s\n", token)
	} else {
		fmt.Println(token)
	}
	return nil
}
