# kalama

The package manager and launcher for the
[shaw](https://github.com/justin06lee/shaw) terminal arcade engine.

`kalama` installs, removes, lists, and launches games for the shaw arcade. It
pulls games from the [hegale](https://github.com/justin06lee/hegale) registry and
drops the matching per-OS binary plus a `manifest.json` into
`~/.kalama/games/<game>/`, so you only download the games you want and can delete
them anytime.

## Install

```
go install github.com/justin06lee/kalama/cmd/kalama@latest
```

## Quickstart

```
kalama install luma   # download and install the luma game
kalama play           # pick a game from the menu and play
```

## Commands

```
kalama install <game>   fetch from hegale and install to ~/.kalama/games/<game>/
kalama remove  <game>   delete an installed game
kalama list             list installed games
kalama play [game]      launch a game directly, or show a menu of installed games
kalama help             show usage
```

In the `play` menu: `↑`/`↓` (or `k`/`j`) move, `enter` plays the highlighted
game, `q`/`esc` quits without playing.

## Environment

| Variable          | Default                           | Purpose                   |
| ----------------- | --------------------------------- | ------------------------- |
| `KALAMA_HOME`     | `~/.kalama`                       | where games are installed |
| `KALAMA_REGISTRY` | the hegale `index.json` on GitHub | registry index URL        |

## Install layout

Each installed game lives in its own directory under `KALAMA_HOME`:

```
~/.kalama/
  games/
    luma/
      manifest.json   {"name","description","version","binary"}
      luma            the executable (chmod 0755)
```

`kalama` reads the hegale registry index, picks the asset matching your
`GOOS/GOARCH`, downloads it atomically (temp file + rename), and records the
manifest alongside the binary.

## Related

- [shaw](https://github.com/justin06lee/shaw) — the arcade engine games are built on.
- [hegale](https://github.com/justin06lee/hegale) — the registry kalama pulls games from.
