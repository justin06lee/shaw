// Command kalama is the package manager and launcher for the shaw terminal arcade.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/justin06lee/kalama/internal/launcher"
	"github.com/justin06lee/kalama/internal/registry"
	"github.com/justin06lee/kalama/internal/store"
)

const usage = `kalama — package manager and launcher for the shaw terminal arcade

Usage:
  kalama install <game>   fetch from hegale and install to ~/.kalama/games/<game>/
  kalama remove  <game>   delete an installed game
  kalama list             list installed games
  kalama play [game]      launch a game (menu if no name given)
  kalama help             show this help

Environment:
  KALAMA_HOME      install location (default ~/.kalama)
  KALAMA_REGISTRY  registry index URL (default hegale on GitHub)
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "kalama: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "install":
		if len(args) < 2 {
			return fmt.Errorf("install requires a game name")
		}
		return cmdInstall(args[1])
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("remove requires a game name")
		}
		return cmdRemove(args[1])
	case "list":
		return cmdList()
	case "play":
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		return launcher.Play(name)
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdInstall(name string) error {
	ix, err := registry.Fetch()
	if err != nil {
		return err
	}
	g, ok := ix.Find(name)
	if !ok {
		return fmt.Errorf("game %q not found in registry", name)
	}
	url, ok := g.AssetURL(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return fmt.Errorf("no build of %s for %s/%s", name, runtime.GOOS, runtime.GOARCH)
	}
	m := store.Manifest{Name: g.Name, Description: g.Description, Version: g.Version, Binary: g.Binary}
	if err := store.Install(m, url); err != nil {
		return err
	}
	fmt.Printf("installed %s %s\n", g.Name, g.Version)
	return nil
}

func cmdRemove(name string) error {
	if err := store.Remove(name); err != nil {
		return err
	}
	fmt.Printf("removed %s\n", name)
	return nil
}

func cmdList() error {
	games, err := store.List()
	if err != nil {
		return err
	}
	if len(games) == 0 {
		fmt.Println("no games installed")
		return nil
	}
	for _, g := range games {
		fmt.Printf("%s  %s  %s\n", g.Name, g.Version, g.Description)
	}
	return nil
}
