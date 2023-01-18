package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
	"golang.org/x/term"
)

// Auth inicia sesión y vuelca el token por consola
func auth(c *cli.Context, store *config.Store, backoff keystone.Backoff) error {
	selectedContext := c.String(selectedContextFlag.Name)
	if err := store.Read(selectedContext); err != nil {
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
	verbose := c.Bool(verboseFlag.Name)
	fiwareToken, urboToken, err := getTokens(k, selected, string(bytepw), backoff, verbose)
	if err != nil {
		return err
	}
	save := c.Bool(saveFlag.Name)
	if save {
		if _, err := store.Set(selectedContext, map[string]string{
			"token":     fiwareToken,
			"urbotoken": urboToken,
		}); err != nil {
			return err
		}
	}
	if save && !verbose {
		fmt.Printf("tokens for context %s cached\n", selected.Name)
	} else {
		if runtime.GOOS == "windows" {
			fmt.Printf("SET FIWARE_TOKEN=%s\nSET URBO_TOKEN=%s\n", fiwareToken, urboToken)
		} else {
			fmt.Printf("export FIWARE_TOKEN=%s\nexport URBO_TOKEN=%s\n", fiwareToken, urboToken)
		}
	}
	return nil
}

func getTokens(api *keystone.Keystone, selected config.Config, password string, backoff keystone.Backoff, verbose bool) (string, string, error) {
	client := httpClient(verbose)
	var fiwareToken, urboToken string
	var fiwareError, urboError error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fiwareToken, fiwareError = api.Login(client, password, backoff)
	}()
	if selected.UrboURL != "" {
		wg.Add(1)
		u, err := urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
		if err != nil {
			return "", "", err
		}
		go func() {
			defer wg.Done()
			urboToken, urboError = u.Login(client, password, backoff)
		}()
	}
	wg.Wait()
	if fiwareError != nil {
		return "", "", fiwareError
	}
	if urboError != nil {
		return "", "", urboError
	}
	return fiwareToken, urboToken, nil
}

func serve(store *config.Store, backoff keystone.Backoff) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			http.Error(w, "unsupported content type", http.StatusNotAcceptable)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		context := r.Form.Get("context")
		if context == "" {
			http.Error(w, "must provide context name", http.StatusBadRequest)
			return
		}
		password := r.Form.Get("password")
		if password == "" {
			http.Error(w, "must provide password", http.StatusBadRequest)
			return
		}
		selected, err := store.Info(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if selected.KeystoneURL == "" || selected.Service == "" || selected.Username == "" {
			http.Error(w, "context is not properly configured", http.StatusNotFound)
			return
		}
		k, err := keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fiwareToken, urboToken, err := getTokens(k, selected, password, backoff, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		selected.Token = fiwareToken
		selected.UrboToken = urboToken
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		if err := enc.Encode(selected); err != nil {
			log.Println(err.Error())
		}
	})
}
