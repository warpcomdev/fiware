package main

import (
	"errors"
	"fmt"

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
	fmt.Printf("Using context %s", s.Current.Name)
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

func listContext(s *config.Store, c *cli.Context) error {
	names, err := s.List()
	if err != nil {
		return err
	}
	for index, curr := range names {
		if index == 0 {
			fmt.Printf("* %s\n", curr)
		} else {
			fmt.Println(curr)
		}
	}
	return nil
}

func useContext(s *config.Store, c *cli.Context) error {
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
	fmt.Printf("using context %s\n", s.Current.Name)
	return nil
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

func setContext(s *config.Store, c *cli.Context, pairs []string) error {
	if err := s.Set(pairs); err != nil {
		return err
	}
	fmt.Printf("using context %s\n", s.Current.Name)
	fmt.Println(s.Current.String())
	return nil
}

func setParamsContext(s *config.Store, c *cli.Context, pairs []string) error {
	if err := s.SetParams(pairs); err != nil {
		return err
	}
	fmt.Printf("using context %s\n", s.Current.Name)
	fmt.Println(s.Current.String())
	return nil
}
