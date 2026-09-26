# Data sources

Every input `cmd/bundle` reads, where it came from, and on what terms. The
files themselves are kept locally and are not committed.

Each input was verified byte-for-byte against its upstream on 2026-09-23 where
the upstream still publishes it. Original retrieval dates were not recorded;
the checksums below identify exactly what the current bundle was built from.

The machine-readable registry is [`data/sources.json`](data/sources.json);
what each status allows is in [DATA-LICENSE.md](DATA-LICENSE.md).

| Source | Flag | What NikkiBase uses | Licence | Status |
|---|---|---|---|---|
| Love Nikki Wiki, by its editors | `-fandom` | Item names, slots and places, letter grades and style tags (~18,200 items); the style codes of Template:S; Maiden's tags on Story 5-12, 6-7 and 6-9; the items 57 story stages require; how to get 19,373 items, from item and suit pages; the suit of 29,323 items, and the Chinese name of each suit | CC BY-SA 3.0 | redistributable |
| nikkiup2u3 by 傲娇攻略组 | `-packed`, `-stage-values` | Every spirit's flat bonus; grades, tags and places for ~14,800 items no other source has; exact weights and tag awards for every stage except co-op; how to get ~14,600 items the wiki has no page for; the suit of 121 items the wiki places in none | None located | permission-required |
| nikkiup2u by lovenikkiusa | `-stage-names`, `-items` | English names of arena and co-op stages; Maiden's tag sizes on Story 5-12 | None located | permission-required |
| nikkiup2u3_data by seal100x | `-stages` | The stage list, modes and rules; co-op weights and tag awards; the Maiden notes; the items 25 stages require where the wiki names none | None located | permission-required |
| Nikki Calc (nikkicalc.com) | `-names`, `-keys`, `-subgrades` | English item names where no other source has one, and one that `data/id-corrections.json` shows in place of the wiki's (181600); every item's rarity; the Global item-ID list; the sub-grades (+ and −) that set each item's stats within its letter grade; its spelling of item names, which decides which hyphenated words in item names stay joined and which hyphens are spaced as separators | None; written permission | permitted |

The three permission-required sources ship only under the exceptions recorded
in [`data/production-exceptions.json`](data/production-exceptions.json). Nikki
Calc ships with its maintainer's written permission, on the conditions in
[DATA-LICENSE.md](DATA-LICENSE.md).

## Rebuilding the bundle

The bundle `2026-09-28` rebuilds byte-for-byte from the files below
(verified 2026-09-26, provenance included, with the build time fixed by
`SOURCE_DATE_EPOCH`). With the source files in `.ai/research/sources/` and the
wiki dump extracted to `/tmp/fandom`:

```sh
SOURCE_DATE_EPOCH=1790553600 go run ./cmd/bundle \
  -fandom  /tmp/fandom/lovenikki673_pages_current.xml \
  -stages       .ai/research/sources/community/seal100x/levels.js \
  -stage-values .ai/research/sources/community/aojiao/levels.js \
  -stage-names  .ai/research/sources/community/nikkiup2u/data/levels.js \
  -packed  .ai/research/sources/community/aojiao/wardrobe.js \
  -names   .ai/research/sources/ids/items-v0.14.json \
  -keys    .ai/research/sources/ids/ni-ids-v0.14.json \
  -subgrades '.ai/research/sources/nikkicalc/item-batches/item-batch-v0.14-*.json' \
  -out     web/public/data \
  -version 2026-09-28
```

A version directory is served as immutable once deployed. `2026-09-27`, the
bundle before this one, holds the same `stages.json`, `tags.json` and
`positions.json` byte for byte. Its catalogue graded 978 items that no source
of this build grades (498 accessories, 106 dresses, 90 hairs, 77 shoes, 53
coats, 51 hosiery, 44 tops, 38 bottoms and 21 makeup); `2026-09-28` lists them
in `items.json` by Nikki Calc's name alone, so its catalogue holds 33,017
items. Of the items both grade, 40 differ in at least one letter and 31 in a
side, 77 in their style tags (19 lose them all) and 23 in their place, all now
as nikkiup2u3 or the wiki gives them. `items.json` names the suit of 29,444
items, as the wiki's suit pages are titled (`2026-09-27` named 24,706, from a
source this build no longer reads), 30 rarities change to Nikki Calc's, and 238
names change: 222 only in spacing, punctuation or case, and 16 in wording (Deer
Fence is now Deer Haven). `acquire.json` says how to get 33,990 items instead
of 33,987: four items gain lines and one loses them, and two craft lines gain a
link to an ingredient. The suit a gift box completes is now the one the wiki
gives: 119 items' gift box lines name it where they did not, 10 no longer do,
and 7 name it as the wiki's suit page is titled. The bundles before
`2026-09-28` were built from sources this tree no longer reads and do not
rebuild from it. Never rebuild into a version that has been served -- give the
new build a new `-version`.

It reads `data/sources.json`, `data/production-exceptions.json`,
`data/stage-scope.json`, `data/stage-difficulty.json` and `data/stage-rules.json` by default,
and `data/acquisition-cn.json`, the English lines for the packed table's source
codes, whenever `-packed` is given.
Pass `-exceptions ""` to build without the recorded exceptions; it then refuses
the three above. `-allow-unlicensed` admits anything but marks the bundle
research-only, and `deploy.sh` will not ship it.

## Love Nikki Wiki

- **URL:** https://lovenikki.fandom.com (MediaWiki `lovenikki673`)
- **File:** `fandom/lovenikki673_dump.7z` — pages-current XML dump, latest revision 2026-07-29
- **SHA-256:** `d06b9fd7339cbad6b69ee4fb6677b23f503ee092e1c1f840a8165c67c95865e8` (14,346,916 bytes)
- **Licence:** CC BY-SA 3.0 Unported, per https://www.fandom.com/licensing (archived 2026). Attribution may be given by link, stable copy, or author list; the site links the wiki, gives each item's page as lovenikki.fandom.com/wiki/<item name>, and lists the dump's 424 named editors.
- **Used for:** item names, slots and places, letter grades and style tags; the style codes of Template:S, kept in `pipeline/tagtable.go`; from the stage pages of Story 5-12, 6-7 and 6-9, which tags Maiden pays; from the quest field of 57 story stage pages, the items those stages require, kept in `data/stage-rules.json`; and how each item is obtained, written to `acquire.json` as short lines of NikkiBase's own: the infobox's "how to obtain", the Crafted from, Evolved from and Reconstructed from sections, the Customization sections for dye costs and for the item a customization starts from where the opening paragraph names an impossible one, Template:Currency and Template:Items for the names of currencies and materials, and, for items without a page, the "how to obtain" and gift box of their suit's page. Where two items share a name, the suit each item page and suit page gives, and the number the item list pages (Hairs, Coats, Earrings and the others) give each title, decide which item a line or ingredient names. The same suit pages give each item its suit in `items.json`, named as the suit's page is titled (so "Wish of Snow (Gallery Suit)" and "Wish of Snow (Pigeon Suit)" stay two suits): the suit its own page's "part of suit" field names, unless that suit's page leaves it out and another suit page lists it, or else the suit page that lists it, the first title in alphabetical order (29 items are listed on more than one). A part listed as "X (Day)" with its night form places both items named X in the part's slot. Pack pages (type Pack) are not suits and give no item its suit. Each suit page's "cnwiki" field, the suit's Chinese name, lets nikkiup2u3's suit column be read in English. The spirit pages' Skill Bonus sections are read as a check only; nothing from them ships.
- **Not read:** item descriptions and captions (verbatim game text). The opening paragraph of an item page is read only for prices, the shops it says the item can now be bought in, the suit a Styling Gift Box completes and the item a customization starts from; none of its wording ships.
- **Note:** the wiki community moved to Miraheze (https://lovenikki.miraheze.org) on 2026-08-15. This dump predates the move.

## nikkiup2u3 by 傲娇攻略组

- **URL:** https://github.com/aojiaogongluezu/nikkiup2u3 — `gh-pages` branch, `data/wardrobe.js` and `data/levels.js`
- **Files:**
  - `community/aojiao/wardrobe.js` — identical to upstream; self-dated `wardrobe_lastupd = '2026/8/24'`. SHA-256 `53c2b294d0237631a93785a351ed3302fad4c167407727adaa2d9784245ce0d2` (1,537,899 bytes).
  - `community/aojiao/levels.js` (`-stage-values`) — identical to upstream at commit `0459ba134b9d6dc35f466b1419daf5be29731c6f` (2026-08-29); last changed 2026-08-28. SHA-256 `12aaf5cfb686380563cab8bd23d71f7af82ae46a7d2fe769d32c8d657cdddfe3` (60,791 bytes).
- **Used for:** every spirit's flat bonus (its only source); letter grades, style tags and places for ~14,800 items no other source has; the stage values below; and, for the ~14,600 items the wiki has no page for, how to get them, from the source column of `wardrobe.js`, translated through `data/acquisition-cn.json` (a code missing there fails the build) and marked `"cn":1` in `acquire.json`. The table describes the Chinese server, so these lines can differ from Global; event names are given only where the wiki names the same event on at least three quarters of the items both cover, a count the build checks against both sources every time, and an evolution or customization base the table files under another garment's family is left out. Its suit column gives the suit of 121 items the wiki places in none, under the title of the wiki's suit page whose "cnwiki" field gives the same Chinese name; where both place an item they agree on 23,658 of the 23,665, and the wiki is kept on the other 7. Items it files only as a suit's base (1,355, the pieces a suit's evolved items start from) are left without one, and so are 740 whose Chinese suit name no wiki suit page gives, or two pages give.
- **Stage values:** weights to four decimals, and every tag award as a factor of the stage's weight sum (grade `F`). Its numbers are Princess's where the two difficulties differ. They replace the stage source's on every stage it carries: all but the co-op stages, which it does not have.
- **Licence:** none located.
- **Filtering:** rows for items not released in the Global game (~4,900) are dropped against the Nikki Calc ID list.

## nikkiup2u by lovenikkiusa

- **URL:** https://github.com/lovenikkiusa/nikkiup2u — commit `3eeff01ece97d95d4f47244e38160d88d7193c45` (2020-11-08)
- **File:** `community/nikkiup2u/data/levels.js`
- **SHA-256:** `8ed9c40eaa249a7f7bf7eaba93010a09449998269216cdeb33836de1f04c8e4b` (135,853 bytes)
- **Licence:** none located (the repo's `LICENSE.md` contains git commands, not a licence).
- **Used for:** the English names of the arena and co-op stages, which the stage source names in Chinese only, joined on the Chinese half of this file's bilingual keys. Two misspellings are corrected on the way: "Neve" (Neva) and "Ofiice" (Office).
- **Also used:** Maiden's tag sizes on Story 5-12, which its table holds.
- **Not used:** weights, rules, other tag bonuses. Unmaintained since 2020, and its stages end at Volume 2 Chapter 5. `-items` (its `wardrobe.js`) is a fallback for builds without `-fandom`; the current bundle does not use it.

## nikkiup2u3_data by seal100x

- **URL:** https://github.com/seal100x/nikkiup2u3_data — `gh-pages` branch, commit `4ab7207ae14dc1255d6331c923be119fbb3349a5` (2026-09-19); `levels.js` last changed 2026-08-24
- **File:** `community/seal100x/levels.js` — identical to upstream at that commit
- **SHA-256:** `54113933897a932f9826998d86d9b27c6ca8a6370c6af05cf4c68ff8426da08d` (172,223 bytes)
- **Licence:** none located.
- **Used for:** the stage list, each stage's mode and rules (`levelFilters` and `addHintInfo`), limited to the Global stages in `data/stage-scope.json`, and the weights and tag awards of the 19 co-op stages; the exact stage values replace them on every other stage. Its notes and tables give Maiden's numbers for 9 of the 10 stages in `data/stage-difficulty.json`, and the build holds that file to them. Its notes give the items 25 stages require where the wiki names none, kept in `data/stage-rules.json`, which the build holds to their wording.
- **Not used:** opponent skills, and the stages not yet released on Global (Commission 21-25, Volume III past chapter 3, events, Dream Weaver).

## Nikki Calc

- **URL:** https://nikkicalc.com/data/items-v0.14.json and https://nikkicalc.com/data/ni-ids-v0.14.json — both identical to upstream; and the item batches at https://nikkicalc.com/data/items/item-batch-v0.14-<n>.json
- **SHA-256:** items `4b991ac68bf43b70c0068ff18a1b726e940f5b0c60bc467d6c6cb83ef9f1c5aa` (1,418,602 bytes); ids `7916e730db0a37bbbc486026175acc135a8c8e2f14b38b6973f01242a517f7f0` (462,771 bytes)
- **Files (`-subgrades`):** `nikkicalc/item-batches/item-batch-v0.14-<n>.json`, 69 files of 500 items each (n = 0, 500, … 34000; data version v0.14, built 2026-09-08), fetched 2026-09-21 and 2026-09-24, 15,012,873 bytes in all. Each file's SHA-256 is recorded in the bundle's `provenance.json`.
- **Licence:** none. **Permission:** in writing, from the maintainer, on 2026-09-26, on the conditions in [DATA-LICENSE.md](DATA-LICENSE.md).
- **Used for:** English item names where no other source has one (~15,800 items: those nikkiup2u3 names in Chinese only, and the 995 that reach `items.json` by Nikki Calc's name alone), and the name of 181600, Cookie Sweet Dream, which `data/id-corrections.json` shows in place of the wiki's Biscuits & Sweet Dream; every item's rarity (34,012 items), from the rarity code each record holds after its index, where 1 to 6 and 7 to 12 both mean 1 to 6 hearts; the set of valid Global item IDs, which keeps unreleased items out; and each item's sub-grades (+ and −), which set its stats within its letter grade. A sub-grade is used only where its letter and side agree with the item's own grade (164,644 of 165,085 graded stats); the rest keep their letter's value. Letters and sides are never taken from it, and tag awards stay priced on the letter grades.
- **Spelling:** its spelling of item names decides which hyphenated words in item names stay joined and which hyphens are spaced as separators. A hyphen written with a space on one side only is spaced on both (Moonlight - White). A hyphenated word in an item's name stays joined where Nikki Calc spells it joined for the same item. Otherwise a hyphen is spaced as a separator where Nikki Calc writes a separator there instead (a middot, or a hyphen with a space beside it) for the same item, whatever the length of the words or the number of hyphens (Fox Talk - Me, Passers-By - Red). Otherwise it stays joined where a lowercase letter follows it, and a word made of two or more letters, one hyphen and three or more letters is spaced as a separator (Doll Dress - Blue), unless it starts with a common prefix such as anti- or re-; every other hyphen is kept as written.
- **Not read:** item descriptions and every other field of the item batches except each item's attribute sides and sub-grades.
