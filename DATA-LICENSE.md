# Data licence

NikkiBase's [MIT licence](LICENSE) covers its **source code** only: `core/`,
`pipeline/`, `cmd/`, `wasm/`, `main.go` and `web/`, with two exceptions:
`web/src/wasm_exec.js` is the Go authors' (BSD-3-Clause, see [NOTICE](NOTICE)),
and the style-code table `styleName` in `pipeline/tagtable.go` follows the
wiki's Template:S (CC BY-SA 3.0, below). The MIT licence does not cover data,
and it grants no rights in it.

The data is built by `cmd/bundle` from the sources registered in
[`data/sources.json`](data/sources.json); [SOURCES.md](SOURCES.md) says what
NikkiBase takes from each. The source files are not committed. The repository
holds only facts derived from them — the scope lists and corrections in
`data/`, including ten stages' Maiden numbers, the items 82 stages require and
the English lines for nikkiup2u3's source codes, the scores pinned in
`golden/testdata/`, and the editor list in `web/public/credits/` — and these
are data, not MIT. Every bundle carries a `provenance.json` recording which
sources went into it and on what basis.

## Sources

| Source | ID | Licence stated? | Redistribution permission established? | Included in production? | Research-only? |
|---|---|---|---|---|---|
| Love Nikki Wiki | `love-nikki-wiki` | Yes — CC BY-SA 3.0 Unported | Yes — under that licence | Yes — under licence | No |
| nikkiup2u3 (傲娇攻略组) | `aojiao-nikkiup2u3` | No | No | Yes — recorded exception | No |
| nikkiup2u (lovenikkiusa) | `lovenikkiusa-nikkiup2u` | No | No | Yes — recorded exception | No |
| nikkiup2u3_data (seal100x) | `seal100x-nikkiup2u3-data` | No | No | Yes — recorded exception | No |
| Nikki Calc | `nikki-calc` | No | Yes — in writing, from the maintainer | Yes — with permission | No |
| Hand-supplied known-ID list | `known-id-list` | No | No | No | Yes |

A test (`pipeline/datalicence_test.go`) fails the build if this table and
`data/sources.json` disagree.

## Love Nikki Wiki — CC BY-SA 3.0

Text on the [Love Nikki Wiki](https://lovenikki.fandom.com) is licensed under
[CC BY-SA 3.0 Unported](https://creativecommons.org/licenses/by-sa/3.0/), per
[Fandom's licensing terms](https://www.fandom.com/licensing).

NikkiBase adapts the wiki's item pages (pages-current dump, latest revision
2026-07-29): it extracts item names, slots and places, letter grades and style
tags, respaces names and corrects some, reads misnumbered pages at the item
they describe or drops them, renames pages titled after another item, and
merges the result with the other sources. From the same pages, and from suit
pages for items that have no page of their own, it takes how each item is
obtained (shop and price, crafting recipe, evolution, customization, stage,
event or pack) and writes it as short lines of its own in `acquire.json`; the
entries there not marked `"cn":1` all come from the wiki. From its suit
pages, and the suit field of its item pages, it takes the suit each item
belongs to, named as the suit's page is titled, for the suit column of
`items.json`, and each suit page's Chinese name, which gives the English name
of the suits nikkiup2u3 names in Chinese. It also uses the style codes of
Template:S; from the stage pages of Story 5-12, 6-7 and 6-9, which tags Maiden
pays; and, from the quest field of 57 story stage pages, the items those stages
require. All of this is offered under the same licence, CC BY-SA 3.0.
Attribution: the Love Nikki Wiki and its editors, who are listed on the site
under *Credits & licences*; each item's page is
lovenikki.fandom.com/wiki/<item name>.

NikkiBase does not take item descriptions, captions, story text, images or the
wording of its pages from the wiki.

## Nikki Calc — written permission

Nikki Calc has no licence. Its maintainer claims only the work of compiling
it, not the game data it holds. On 2026-09-26 the maintainer
[permitted NikkiBase to use it](https://github.com/nikki-calc/nikki-calc-project/issues/33#issuecomment-5843594161)
on three conditions:

1. NikkiBase stays free for the community. Ads and Patreon are allowed; the
   data itself is not sold.
2. Nikki Calc is credited.
3. Nothing NikkiBase does puts load on nikkicalc.com or its hosting.

NikkiBase is free to use, credits Nikki Calc on the site under
*Credits & licences*, and uses only the JSON files Nikki Calc publishes
(`items-v0.14.json`, `ni-ids-v0.14.json` and the item batches), downloaded when
the data is built. The site never loads anything from nikkicalc.com.
[SOURCES.md](SOURCES.md) lists what NikkiBase takes from it.

This permission was given to NikkiBase. It is not a licence, and it grants you
no right to reuse Nikki Calc's data.

## Sources with no licence

For nikkiup2u3, nikkiup2u and nikkiup2u3_data:

**No explicit redistribution licence located. Not included in production
bundles unless permission is established — or, until then, a deliberate
exception is recorded.**

NikkiBase currently ships data from all three under exceptions recorded in
[`data/production-exceptions.json`](data/production-exceptions.json). An
exception is a dated decision by the project to ship the data while permission
is requested. It is **not** permission, and neither is crediting the source:
nothing here grants you, or NikkiBase, any right to reuse that data. If a
maintainer refuses permission or asks for removal, the exception is withdrawn
and the data is taken out of the next bundle.

`cmd/bundle` enforces this. It refuses to build from a source with no licence
unless an exception for it is recorded, or `-allow-unlicensed` is passed —
which marks the bundle research-only, and `deploy.sh` then refuses to ship it.

## Love Nikki itself

Love Nikki and its game content are proprietary to their respective rights
holders. NikkiBase is an unofficial fan project and is not affiliated with or
endorsed by them. It reproduces item, suit, stage, style, event, shop,
pavilion, character (Dream Weaver), currency and material names only, as the
minimum needed for a player to recognise an item and find it. It does not
reproduce artwork, item descriptions or story text, and it filters out items
that have not been released in the Global game.

## Contact

To have something credited differently or removed,
[open an issue](https://github.com/danielradosa/nikkibase/issues) or email
[nikkibaseproject@gmail.com](mailto:nikkibaseproject@gmail.com).
