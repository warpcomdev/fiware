package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
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
	client := httpClient(c.Bool(verboseFlag.Name))
	var fiwareToken, urboToken string
	var fiwareError, urboError error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fiwareToken, fiwareError = k.Login(client, string(bytepw))
	}()
	if selected.UrboURL != "" {
		wg.Add(1)
		u, err := urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
		if err != nil {
			return err
		}
		go func() {
			defer wg.Done()
			urboToken, urboError = u.Login(client, string(bytepw))
		}()
	}
	wg.Wait()
	if fiwareError != nil {
		return fiwareError
	}
	if urboError != nil {
		return urboError
	}
	if c.Bool(saveFlag.Name) {
		if err := store.Set([]string{
			"token", fiwareToken, "urbotoken", urboToken,
		}); err != nil {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		fmt.Printf("SET FIWARE_TOKEN=%s\nSET URBO_TOKEN=%s\n", fiwareToken, urboToken)
	} else {
		fmt.Printf("export FIWARE_TOKEN=%s\nexport URBO_TOKEN=%s\n", fiwareToken, urboToken)
	}
	return nil
}
