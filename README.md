# shaw

The launcher and package manager for the shaw terminal arcade. Type `shaw` to
pick a game and play; install more with `shaw install`.

Games are built on the [kalama](https://github.com/justin06lee/kalama) engine and
pulled from the [hegale](https://github.com/justin06lee/hegale) registry. shaw
drops the matching per-OS binary plus a `manifest.json` into `~/.shaw/games/<game>/`.

## Install

```
go install github.com/justin06lee/shaw/cmd/shaw@latest
```

## Quickstart

```
shaw install snake.shaw   # download and install the snake game
shaw                      # open the menu and play
```

## Commands

```
shaw                    open the game menu
shaw install <game>     fetch from hegale and install to ~/.shaw/games/<game>/
shaw remove  <game>     delete an installed game
shaw list               list installed games
shaw play [game]        launch a game directly, or open the menu
shaw help               show usage
```

In the menu: `↑`/`↓` (or `k`/`j`) move, `enter` plays the highlighted game,
`q`/`esc` quits. Games are listed by their friendly name (the `.shaw` suffix is
hidden), e.g. `snake.shaw` shows as `snake`.

## Environment

| Variable        | Default                           | Purpose                   |
| --------------- | --------------------------------- | ------------------------- |
| `SHAW_HOME`     | `~/.shaw`                         | where games are installed |
| `SHAW_REGISTRY` | the hegale `index.json` on GitHub | registry index URL        |

## Install layout

```
~/.shaw/
  games/
    snake.shaw/
      manifest.json   {"name","description","version","binary"}
      snake.shaw      the executable (chmod 0755)
```

## Related

- [kalama](https://github.com/justin06lee/kalama) — the engine games are built on.
- [hegale](https://github.com/justin06lee/hegale) — the registry shaw pulls games from.
