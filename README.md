# NikkiBase

[![checks](https://github.com/danielradosa/nikkibase/actions/workflows/checks.yml/badge.svg?branch=main)](https://github.com/danielradosa/nikkibase/actions/workflows/checks.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/danielradosa/nikkibase)](go.mod)
[![code licence: MIT](https://img.shields.io/badge/code_licence-MIT-blue)](LICENSE)
[![data: see DATA-LICENSE.md](https://img.shields.io/badge/data-see_DATA--LICENSE.md-lightgrey)](DATA-LICENSE.md)

> **Awaiting approvals.** Permission to use the game's content and the
> community data NikkiBase is built on is being requested from the Love Nikki
> team and each data maintainer. Until they reply, NikkiBase stays as it is. See
> [DATA-LICENSE.md](DATA-LICENSE.md) for what is and isn't licensed today.

NikkiBase suggests a stage outfit from the items you own, runs entirely in
your browser, and tells you what it doesn't know.

Your wardrobe is never uploaded.

**Status: early beta.** See [KNOWN-DEVIATIONS.md](KNOWN-DEVIATIONS.md) for
every place it knowingly differs from the game.

## What NikkiBase knows

- Letter grades and sub-grades (S−, S, S+ and so on) for about 33,000 Global
  items, gathered from community sources (see [SOURCES.md](SOURCES.md)).
- Stage weights and style-tag bonuses for every stage released on the Global
  server: Story Volumes I–II and Volume III to chapter 3, Commission Acts 1–20,
  Arena and Co-op, at Maiden and Princess where the two differ.
- The game's 34 wearable places, including which hand-held items exclude each
  other, and the accessory penalty.
- The items 82 Story stages require. Every suggested outfit wears them, and the
  site says when your wardrobe lacks one.

On Story 1-1 its best possible outfit scores within 1% of the best outfit
Nikki Calc finds, which its live scoring reproduces exactly; both are 32 items.
That comparison is pinned in the test suite. It checks NikkiBase against
another calculator, not against the game.

## What it doesn't know

- **Exact item stats.** The game publishes grades, not numbers, so every score
  is an estimate. Pricing each item from its sub-grade keeps the measured
  single stats within about 1.5%; two items with the same sub-grade can't be
  told apart.
- **Skills.** Scores assume no skills unless you switch Skills on. Smile and
  Charming are modelled at every level, on the two attributes your outfit
  scores most on, or where you choose. That is usually the best placement but
  not always (on some stages another one scores up to about 1% more). Lower
  levels use the same formula as the maximum, the only level the community has
  checked. The other skills act on your opponent or shield you, so they don't
  change your best outfit and aren't modelled.
- **Stage rules.** Stages where some items or styles score F are flagged, but
  not checked.
- **Whether you pass.** It doesn't know the opponent's score or the B/A/S
  thresholds.
- **Events.** Event and Dream Weaver stages aren't in the data, and new Global
  chapters are added as they're released.

## Importing a wardrobe

- Your game's `clothes_date` file.
- A [Nikki Calc](https://nikkicalc.com) selections file (`@SEL…`).
- Or tick the items you own by hand in the Items tab.

Either file is read in your browser and never sent anywhere.

## What to get next

The Worth getting tab ranks the items you don't own by how much each would
raise your best scores, as a share of the best possible score on each stage,
and says how to get them. Each item assumes you already got the ones above it.
Stages you can't pass yet, because you lack an item they require, are listed
apart.

## Licences

- **Code** — MIT, see [LICENSE](LICENSE). It covers the source in this
  repository only.
- **Data** — the site's item and stage data is *not* MIT. It comes from
  community sources, each on its own terms, and five of them publish no
  licence: see [SOURCES.md](SOURCES.md) for who made each and what NikkiBase
  uses from it, and [DATA-LICENSE.md](DATA-LICENSE.md) for the terms. The
  source files are not committed, and `cmd/bundle` refuses unlicensed sources
  unless an exception is recorded.
- **Third-party software** — see [NOTICE](NOTICE).
- **Privacy** — see [PRIVACY.md](PRIVACY.md).

Love Nikki and its game content are proprietary to their respective rights
holders. NikkiBase is an unofficial fan project and is not affiliated with or
endorsed by them.

## Corrections and takedowns

If you maintain one of the sources in [SOURCES.md](SOURCES.md), or hold rights in anything the site
shows, and want it credited differently or removed,
[open an issue](https://github.com/danielradosa/nikkibase/issues) or email
[nikkibaseproject@gmail.com](mailto:nikkibaseproject@gmail.com).
