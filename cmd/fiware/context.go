package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
)

func createContext(s *config.Store, c *cli.Context) error {
	if c.NArg() <= 0 {
		return errors.New("please provide the name of the context to create")
	}
	cname := c.Args().Get(0)
	if err := s.Create(cname); err != nil {
		return err
	}
	fmt.Printf("Using context %s\n", s.Current.Name)
	return nil
}

func deleteContext(s *config.Store, c *cli.Context) error {
	if c.NArg() <= 0 {
		return errors.New("please provide the name of the context to remove")
	}
	cname := c.Args().Get(0)
	if err := s.Delete(cname); err != nil {
		return err
	}
	if s.Current.Name == "" {
		fmt.Println("no more contexts remaining")
	}
	fmt.Printf("Using context %s now\n", s.Current.Name)
	return nil
}

func listContext(s *config.Store, c *cli.Context, ignoreMissing bool) error {
	names, err := s.List(ignoreMissing)
	if err != nil {
		return err
	}
	for _, curr := range names {
		if curr == s.Current.Name {
			fmt.Printf("* %s\n", curr)
		} else {
			fmt.Println(curr)
		}
	}
	return nil
}

func useContext(s *config.Store, c *cli.Context) error {
	var (
		name       string
		subservice string
	)
	if c.NArg() > 0 {
		name = c.Args().Get(0)
	}
	if c.NArg() > 1 {
		subservice = c.Args().Get(1)
	}
	if err := s.Use(name); err != nil {
		return err
	}
	cfg := s.Current
	if cfg.Name == "" {
		fmt.Println("no contexts available")
		return nil
	}
	if subservice != "" {
		cfg.Params["subservice"] = subservice
		if err := s.Save(cfg); err != nil {
			return err
		}
	}
	summaryContext(cfg)
	return nil
}

// summaryContext prints a summary of the current context selections
func summaryContext(cfg config.Config) {
	ss := cfg.Params["subservice"]
	if ss != "" {
		fmt.Printf("using context '%s' subservice '%s'\n", cfg.Name, ss)
	} else {
		fmt.Printf("using context '%s' without subservice\n", cfg.Name)
	}
}

func infoContext(s *config.Store, c *cli.Context) error {
	var name string
	if c.NArg() > 0 {
		name = c.Args().Get(0)
	}
	selected, err := s.Info(name)
	if err != nil {
		return err
	}
	if selected.Name == "" {
		fmt.Println("no contexts available")
		return nil
	}
	fmt.Println(selected.String())
	return nil
}

func dupContext(s *config.Store, c *cli.Context) error {
	if c.NArg() <= 0 {
		return errors.New("please provide the name of the new context")
	}
	cname := c.Args().Get(0)
	if err := s.Dup(cname); err != nil {
		return err
	}
	fmt.Printf("Using context %s\n", s.Current.Name)
	return nil
}

func envContext(s *config.Store, c *cli.Context) error {
	var name string
	if c.NArg() > 0 {
		name = c.Args().Get(0)
	}
	if err := s.Use(name); err != nil {
		return err
	}
	if s.Current.Name == "" {
		fmt.Println("no contexts available")
		return nil
	}
	env, err := json.Marshal(config.FromConfig(s.Current))
	if err != nil {
		return err
	}
	var dst bytes.Buffer
	if err := json.Indent(&dst, env, "", "  "); err != nil {
		return err
	}
	fmt.Println(dst.String())
	return nil
}

func makePairs(pairs []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(pairs)%2 != 0 {
		return nil, config.ErrParametersNumber
	}
	for i := 0; i < len(pairs); i += 2 {
		result[pairs[i]] = pairs[i+1]
	}
	return result, nil
}

func setContext(s *config.Store, c *cli.Context, contextName string, pairs []string) error {
	pairMap, err := makePairs(pairs)
	if err != nil {
		return err
	}
	keys, err := s.Set(contextName, pairMap)
	if err != nil {
		return err
	}
	fmt.Printf("using context %s\nupdated fields: {\n", s.Current.Name)
	sort.Strings(keys)
	finalPairs := s.Current.Pairs()
	for _, k := range keys {
		fmt.Printf("  %s: %s\n", k, finalPairs[k])
	}
	fmt.Println("}")
	return nil
}

func setParamsContext(s *config.Store, c *cli.Context, contextName string, pairs []string) error {
	pairMap, err := makePairs(pairs)
	if err != nil {
		return err
	}
	if err := s.SetParams(contextName, pairMap); err != nil {
		return err
	}
	fmt.Printf("using context %s\nupdated params: {\n", s.Current.Name)
	finalPairs := s.Current.Params
	keys := config.SortedKeys(finalPairs)
	for _, k := range keys {
		if _, found := pairMap[k]; found {
			fmt.Printf("    %s: %s\n", k, finalPairs[k])
		}
	}
	fmt.Println("}")
	return nil
}
