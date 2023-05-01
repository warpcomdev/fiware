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

	"github.com/warpcomdev/fiware"
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
	saveCreds := c.Bool(saveFlag.Name)
	if err := getCredentials(c, &selected, backoff, saveCreds, true); err != nil {
		return err
	}
	// Save config to update project cache, so we can autocomplete.
	// Notice this is only done when we authenticate through the CLI.
	if err := store.Save(selected); err != nil {
		return err
	}
	if saveCreds {
		fmt.Printf("tokens for context %s cached\n", selected.Name)
	}
	return nil
}

// impresonatePep get user ID for PEP user in admin_domain env
func authAsPep(c *cli.Context, store *config.Store, backoff keystone.Backoff) error {
	selectedContext := c.String(selectedContextFlag.Name)
	if err := store.Read(selectedContext); err != nil {
		return err
	}
	if store.Current.Name == "" {
		return errors.New("no contexts defined")
	}
	selected := store.Current
	selected.Service = "admin_domain"
	selected.Username = "pep"
	if err := getCredentials(c, &selected, backoff, false, false); err != nil {
		return err
	}
	return nil
}

// getCredentials updates credentials in the selected config object
// but does not save anything to any persistent store.
func getCredentials(c *cli.Context, selected *config.Config, backoff keystone.Backoff, saveCreds, getProjects bool) error {
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
	client := httpClient(verbosity(c), configuredTimeout(c))
	fiwareToken, urboToken, userId, err := getTokens(client, k, selected, string(bytepw), backoff, getProjects)
	if err != nil {
		return err
	}
	if saveCreds {
		selected.SetCredentials(fiwareToken, urboToken)
	} else {
		if runtime.GOOS == "windows" {
			fmt.Printf("SET FIWARE_USERID=%s\nSET FIWARE_TOKEN=%s\nSET URBO_TOKEN=%s\n", userId, fiwareToken, urboToken)
		} else {
			fmt.Printf("export FIWARE_USERID=%s\nexport FIWARE_TOKEN=%s\nexport URBO_TOKEN=%s\n", userId, fiwareToken, urboToken)
		}
	}
	return nil
}

func getAndCacheProjects(client keystone.HTTPClient, api *keystone.Keystone, selected *config.Config, fiwareToken string) error {
	headers := api.Headers("", fiwareToken)
	projects, err := api.Projects(client, headers)
	if err != nil {
		return err
	}
	return cacheProjects(selected, projects)
}

func cacheProjects(selected *config.Config, projects []fiware.Project) error {
	projectNames := make([]string, 0, len(projects))
	for _, project := range projects {
		if strings.HasPrefix(project.Name, "/") {
			projectNames = append(projectNames, strings.TrimPrefix(project.Name, "/"))
		}
	}
	selected.ProjectCache = projectNames
	return nil
}

func getTokens(client keystone.HTTPClient, api *keystone.Keystone, selected *config.Config, password string, backoff keystone.Backoff, getProjects bool) (string, string, string, error) {
	var fiwareToken, urboToken, userId string
	var fiwareError, urboError error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// get fiware token and user id
		defer wg.Done()
		fiwareToken, userId, fiwareError = api.Login(client, password, backoff)
		if fiwareError != nil {
			return
		}
		// when appropiate, get list of projects for autocomplete
		if getProjects {
			fiwareError = getAndCacheProjects(client, api, selected, fiwareToken)
		}
	}()
	if selected.UrboURL != "" {
		wg.Add(1)
		u, err := urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
		if err != nil {
			return "", "", "", err
		}
		go func() {
			defer wg.Done()
			urboToken, urboError = u.Login(client, password, backoff)
		}()
	}
	wg.Wait()
	if fiwareError != nil {
		return "", "", "", fiwareError
	}
	if urboError != nil {
		return "", "", "", urboError
	}
	return fiwareToken, urboToken, userId, nil
}

func authServe(client keystone.HTTPClient, store *config.Store, backoff keystone.Backoff) http.Handler {
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
		fiwareToken, urboToken, _, err := getTokens(client, k, &selected, password, backoff, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		selected.SetCredentials(fiwareToken, urboToken)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		if err := enc.Encode(selected); err != nil {
			log.Println(err.Error())
		}
	})
}
