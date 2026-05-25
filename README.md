# kalama

The package manager for the [shaw](https://github.com/justin06lee/shaw) terminal
arcade engine.

`kalama` installs, removes, and lists games for the shaw arcade. It pulls games
from the [hegale](https://github.com/justin06lee/hegale) registry and drops the
matching per-OS binary plus a `manifest.json` into `~/.kalama/games/<game>/`, so
you only download the games you want and can delete them anytime.

> **Status: stub.** Nothing is built yet. The design lives in the shaw repo at
> `docs/superpowers/specs/2026-05-23-arcade-engine-design.md`. This repo reserves
> the name and will hold the package-manager CLI.

## Planned commands

```
kalama install <game>   # fetch from hegale, install to ~/.kalama/games/<game>/
kalama remove  <game>   # delete an installed game
kalama list             # list installed games
```

## Related

- [shaw](https://github.com/justin06lee/shaw) — the arcade engine games are built on.
- [hegale](https://github.com/justin06lee/hegale) — the registry kalama pulls games from.
