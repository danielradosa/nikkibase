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
| Love Nikki Wiki, by its editors | `-fandom` | Item names, slots and places, letter grades and style tags (~18,400 items); the style codes of Template:S; Maiden's tags on Story 5-12, 6-7 and 6-9; the items 57 story stages require; how to get 19,382 items, from item and suit pages, the event or shop behind 1,403 items' vaguer nikkiup2u3 lines, and the pack behind 343 lines that say only "Recharge"; from event pages, the event of 185 items whose lines are all vague, and from the Events timeline, the recharge channels 760 items were rerun on; the suit of 29,337 items, and the Chinese name of each suit | CC BY-SA 3.0 | redistributable |
| nikkiup2u3 by 傲娇攻略组 | `-packed`, `-stage-values` | Every spirit's flat bonus; grades, tags and places for ~14,800 items no other source has; with Nikki Calc, the grade on 406 grades the wiki's item pages give otherwise; exact weights and tag awards for every stage except co-op; how to get ~14,600 items the wiki has no page for, and the base of 31 customizations and evolutions the wiki names none for; the suit of 137 items the wiki places in none | None located | permission-required |
| nikkiup2u by lovenikkiusa | `-stage-names`, `-items` | English names of arena and co-op stages; Maiden's tag sizes on Story 5-12 | None located | permission-required |
| nikkiup2u3_data by seal100x | `-stages` | The stage list, modes and rules; co-op weights and tag awards; the Maiden notes; the items 25 stages require where the wiki names none | None located | permission-required |
| Nikki Calc (nikkicalc.com) | `-names`, `-keys`, `-subgrades`, `-calc-grades`, `-calc-recipes`, `-calc-suits` | Letter grades, sides, places and style tags for the 982 items no other source grades; the ingredients of 1,755 items the other sources say are crafted without naming them; with nikkiup2u3, the grade on 406 grades the wiki's item pages give otherwise; the English names of 47 suits the wiki has no page for (720 items); English item names where no other source has one, and one that `data/id-corrections.json` shows in place of the wiki's (181600); every item's rarity; the Global item-ID list; the sub-grades (+ and −) that set each item's stats within its letter grade; its spelling of item names, which decides which hyphenated words in item names stay joined and which hyphens are spaced as separators | None; written permission | permitted |

The three permission-required sources ship only under the exceptions recorded
in [`data/production-exceptions.json`](data/production-exceptions.json). Nikki
Calc ships with its maintainer's written permission, on the conditions in
[DATA-LICENSE.md](DATA-LICENSE.md).

## Rebuilding the bundle

The bundle `cloud` rebuilds byte-for-byte from the files below
(verified 2026-09-29, provenance included, with the build time fixed by
`SOURCE_DATE_EPOCH`). With the source files in `.ai/research/sources/`, and the
wiki dump extracted to `/tmp/fandom` and brought up to date by
`fandom/patch-2026-09-29/merge.py` (see [Love Nikki Wiki](#love-nikki-wiki)):

```sh
SOURCE_DATE_EPOCH=1790640000 go run ./cmd/bundle \
  -fandom  /tmp/fandom/lovenikki673_pages_current-2026-09-29.xml \
  -stages       .ai/research/sources/community/seal100x/levels.js \
  -stage-values .ai/research/sources/community/aojiao/levels.js \
  -stage-names  .ai/research/sources/community/nikkiup2u/data/levels.js \
  -packed  .ai/research/sources/community/aojiao/wardrobe.js \
  -names   .ai/research/sources/ids/items-v0.14.json \
  -keys    .ai/research/sources/ids/ni-ids-v0.14.json \
  -subgrades '.ai/research/sources/nikkicalc/item-batches/item-batch-v0.14-*.json' \
  -calc-grades -calc-recipes \
  -calc-suits .ai/research/sources/nikkicalc/suits-v0.14.json \
  -out     web/public/data \
  -version cloud
```

A version directory is served as immutable once deployed. Versions are named
after the seven nations of Miraland, in the order the wiki's 7 Nations page
lists them: `apple`, `lilith`, `cloud`, `pigeon`, `north`, `wasteland`,
`ruin`; after `ruin` the list starts again with a number (`apple-2`,
`lilith-2`, …). `SOURCE_DATE_EPOCH` is the build day's midnight UTC. Versions
built before `apple` are named after their build day (`2026-09-29b`).

`cloud` changes `items.json` and `items.bin` from `lilith` (and the version
in `acquire.json`), after a check in the game. Crisis in the Mist (91095) takes
the grades the game shows, Gorgeous S, Elegant SS, Mature SS, Sexy A and
Cool A, which Nikki Calc also gives, in place of the packed table's Simple S,
Elegant SS, Mature A, Pure S and Cool SS (`gradeOverrides`, whose
`wasAttribute` moves a grade to the other side of its pair). All 19 Nightfall
Menace pieces, which the packed table tags Evening Gown, take Chic, the tag
the game shows on Crisis in the Mist and Nikki Calc gives them all
(`tagOverrides`, both in `data/id-corrections.json`). Four items take the
name the game shows (`displayNames`): Army of Discipline and Dawn of
Independence, as Nikki Calc names them, in place of the wiki's Discipline Army
and Independent Light; Colorful Illusion, as the wiki names it, in place of
Nikki Calc's Colorful Illusions; and the Moon Leaning on High Mountain, with
the lower-case first word the game uses. The best possible score changes on 22
of 605 stage versions, from −0.7% to +2.1%.

`lilith` changes only `acquire.json` from `apple`: the 789 rerun lines on
760 items give the month of their latest rerun in a field of their own
(`"last":"2023-06"`) instead of in their text ("Abyssal Island (last Jun
2023)"), so the site lists the pieces rerun on one channel together and
shows each one's month beside it.

`apple` changes `items.json`, `items.bin` and `acquire.json` from
`2026-09-29b`; its `stages.json`, `tags.json` and `positions.json` are the same
byte for byte, and `items.bin` gives every item the same stats, sides, places
and tags, in another row order. The wiki is read as its dump of 2026-07-29
brought up to 2026-09-22 by the 56 pages changed since it (30 edited, 24 new,
1 moved and 1 deleted): seven of the eight new item pages grade their items
as the other sources already did
(the eighth gives no grades yet), and a new suit page names Metallic Crisis. `items.json` names the suit of 30,194 items
instead of 30,193: Goldfish's Summer Fanstasy joins Goldfish Girl, whose suit
page spells it otherwise (`data/suit-part-aliases.json`). `acquire.json` says
how to get 34,000 items instead of 33,998, and 964 items' lines change. 185
items whose lines were all vague take the event of the one wiki event page
that lists their suit: 109 the $100 Recharge event, 76 one of eight rebate
recharge events. 760 items whose only line was a plain "Recharge" or "Event
recharge" take in its place the recharge channels the wiki's Events timeline
reruns them on, with the month of the latest rerun (Abyssal Island on 709 items,
Cumulative Recharge on 56, the One-Dollar Sale on 23 and Lucky Bags on 1). Neither reaches the items a suit
page lists only as additional items or gives in its gift box: those were
sold on their own or come with the suit's completion. Two items gain lines
through the suit part spellings. 17 change with the newer pages: the Metallic
Crisis suit's ten pieces name the Diamond Consumption event and its two gift
box items the suit, three crafted items name Mailbox Gift as their recipe's
source, Orchid Touch's recipe takes 5 Azure Flower and 6 British Shoes-Brown,
and Horn of Cloud gains the Deluge Dragon event and its page's recipe, whose
"Bow Braclet-Red" is Bow Bracelet-Red (`data/ingredient-aliases.json`). Items
whose lines are all vague fall from 1,694 to 737. Best possible scores are
unchanged.

`2026-09-29b` holds the same data as `2026-10-01`, which was named ahead of
its build day; only `provenance.json` (version and build time) and the version
in `acquire.json` differ. `2026-10-01`
changes `items.json`, `items.bin` and `acquire.json` from `2026-09-30`; its
`stages.json`, `tags.json` and `positions.json` are the same byte for byte.
982 items no other source grades take their grades, sides, places and style
tags from Nikki Calc (`-calc-grades`; 611 of them have a style tag), so the
catalogue holds 34,005 items. Seven stay listed by name only: five whose ID
names another slot than the one Nikki Calc and the wiki's suit pages give, and
two never released (`excluded` in `data/id-corrections.json`). Where the
wiki's item page gives one grade and nikkiup2u3 and Nikki Calc agree on
another, theirs is kept: 406 grades on 285 items, 21 of them on the other side
of their pair. `items.json` names the suit of 30,193 items instead of 29,444:
16 take the suit their other pieces are in, and 733 take Nikki Calc's name for
48 suits the wiki has no page for (`-calc-suits`). `acquire.json` says how to
get 33,998 items instead of 33,990: 1,396 items whose lines were only
nikkiup2u3's "Limited event", "Recharge" and the like take the event or shop
their suit's wiki page names, 343 wiki lines that say only "Recharge" take the
pack their page's categories name, 1,759 items that nikkiup2u3 says only are crafted
take the ingredients of Nikki Calc's recipe for them (`-calc-recipes`), 31
customizations and evolutions gain the base nikkiup2u3 gives, three pack names are spelled one way, and 8 items take a
hand-checked line from `data/acquisition-extra.json`.

`2026-09-30` changes only names from `2026-09-29`, in `items.json` and `acquire.json`: a
name keeps its source's own spelling instead of being respaced, as the game
spells it. 1,978 names the wiki writes with a tight hyphen ("Red
Satin-Epic") lose the spaces added around it, 843 Nikki Calc names take a
tight "-" for its "·", and 2 keep a tight "&"; the 1,958 the wiki writes with
a spaced " - " keep it. Gift box lines name a suit without the wiki's
"(Hidden Suit)"-style qualifier.

`2026-09-29` changes only `items.json`, `items.bin` and `acquire.json` from `2026-09-28`.
The wiki's item pages are read whole: 182 pages whose Attributes template
starts on a new line, or whose empty infobox field hid the wardrobe number,
were skipped before, so the wiki grades 18,405 items instead of 18,223, six
items listed by name only gain grades and the catalogue holds 33,023 items.
Where the wiki's letter differs from both nikkiup2u3's and Nikki Calc's
sub-grade (three stats), theirs is kept (`gradeOverrides` in
`data/id-corrections.json`). Names are shown as the game shows them: the
bracket the wiki adds to tell items apart ("Musical Sound (Coat)") is dropped
from 1,543 names, night forms show their plain name, and six names take the
wiki's " - " for Nikki Calc's " · ".

`2026-09-27`, the bundle before `2026-09-28`, holds the same `stages.json`, `tags.json` and
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
codes, whenever `-packed` is given. `data/acquisition-extra.json` holds
hand-checked lines for items the sources give none for; the build refuses an
entry once the sources give that item a line. `data/suit-part-aliases.json`
(`-suit-part-aliases`) names the item a suit page means where it spells a part
unlike any item; the build refuses an entry once the page no longer lists the
part or the part matches an item without it. `data/ingredient-aliases.json`
(`-ingredient-aliases`) names the item an item page means where its recipe,
evolution or customization spells an ingredient or base unlike any item; the
build refuses an entry once the page no longer gives that spelling unmatched or
it matches an item without the entry.
Pass `-exceptions ""` to build without the recorded exceptions; it then refuses
the three above. `-allow-unlicensed` admits anything but marks the bundle
research-only, and `deploy.sh` will not ship it.

## Love Nikki Wiki

- **URL:** https://lovenikki.fandom.com (MediaWiki `lovenikki673`)
- **File:** `fandom/lovenikki673_dump.7z` — pages-current XML dump, latest revision 2026-07-29
- **SHA-256:** `d06b9fd7339cbad6b69ee4fb6677b23f503ee092e1c1f840a8165c67c95865e8` (14,346,916 bytes)
- **Pages changed since:** the 69 pages created, edited, moved or deleted from 2026-07-28 to 2026-09-22 (latest revision 2026-09-22), exported through the wiki's API (`api.php`) on 2026-09-29 and put in place of their dump revisions, matched by page ID and then title, by `fandom/patch-2026-09-29/merge.py`. 13 of them are already in the dump at the same revision, so 56 pages change: 30 edited, 24 new, 1 moved and 1 deleted. The result, `lovenikki673_pages_current-2026-09-29.xml`, is what `-fandom` reads: SHA-256 `372f607a83e200dbe49a2f0a76904114f59947fa4651abedb3281e70ff35d6d4` (131,786,443 bytes), recorded in `provenance.json`.
- **Licence:** CC BY-SA 3.0 Unported, per https://www.fandom.com/licensing (archived 2026). Attribution may be given by link, stable copy, or author list; the site links the wiki, gives each item's page as lovenikki.fandom.com/wiki/<item name>, and lists the 428 named editors of the pages' latest revisions.
- **Used for:** item names, slots and places, letter grades and style tags; the style codes of Template:S, kept in `pipeline/tagtable.go`; from the stage pages of Story 5-12, 6-7 and 6-9, which tags Maiden pays; from the quest field of 57 story stage pages, the items those stages require, kept in `data/stage-rules.json`; and how each item is obtained, written to `acquire.json` as short lines of NikkiBase's own: the infobox's "how to obtain", the Crafted from, Evolved from and Reconstructed from sections, the Customization sections for dye costs and for the item a customization starts from where the opening paragraph names an impossible one, Template:Currency and Template:Items for the names of currencies and materials, and, for items without a page, the "how to obtain" and gift box of their suit's page, which also names the event, shop or gift box behind an item's nikkiup2u3 lines where those say only "Limited event", "Recharge", "Event recharge", "Styling Gift Box" or "Dream Weaver". An item page's categories name the pack (Abyssal Island, Lucky Bags, Zodiac Lucky Pack, Time-limited Pack, First Recharge Giftpack) behind a line that says only "Recharge". Event pages (those with an Event Infobox, other than recurring events) name the event of an item whose lines are all vague, where exactly one of them lists its suit, and only in place of vague lines of the same kind: "Limited event" by an event, "Recharge" and "Event recharge" by a recharge event such as the $100 Recharge event. The Events timeline pages (Events/2017 to Events/2026) name the recharge channels (Abyssal Island, Cumulative Recharge, the One-Dollar Sale, Lucky Bags) an item whose only lines are a plain recharge was rerun on, with the month of the latest rerun; "Recharge" gives way to them and "Event recharge" is kept. Neither an event page nor the timeline reaches, through its suit, an item the suit page lists only as an additional item or gives in its gift box. Where two items share a name, the suit each item page and suit page gives, and the number the item list pages (Hairs, Coats, Earrings and the others) give each title, decide which item a line or ingredient names. The same suit pages give each item its suit in `items.json`, named as the suit's page is titled (so "Wish of Snow (Gallery Suit)" and "Wish of Snow (Pigeon Suit)" stay two suits): the suit its own page's "part of suit" field names, unless that suit's page leaves it out and another suit page lists it, or else the suit page that lists it, the first title in alphabetical order (29 items are listed on more than one). A part listed as "X (Day)" with its night form places both items named X in the part's slot. A part a suit page spells unlike any item is placed through `data/suit-part-aliases.json`, which names the item each such spelling means (2 today), and an ingredient an item page's recipe spells unlike any item is linked through `data/ingredient-aliases.json` in the same way (1 today). Pack pages (type Pack) are not suits and give no item its suit. Each suit page's "cnwiki" field and its link to the Chinese wiki, the suit's Chinese name, let nikkiup2u3's suit column be read in English. The spirit pages' Skill Bonus sections are read as a check only; nothing from them ships.
- **Not read:** item descriptions and captions (verbatim game text). The opening paragraph of an item page is read only for prices, the shops it says the item can now be bought in, the suit a Styling Gift Box completes and the item a customization starts from; none of its wording ships.
- **Note:** the wiki community moved to Miraheze (https://lovenikki.miraheze.org) on 2026-08-15. The dump predates the move; the pages changed since it were read from lovenikki.fandom.com.

## nikkiup2u3 by 傲娇攻略组

- **URL:** https://github.com/aojiaogongluezu/nikkiup2u3 — `gh-pages` branch, `data/wardrobe.js` and `data/levels.js`
- **Files:**
  - `community/aojiao/wardrobe.js` — identical to upstream; self-dated `wardrobe_lastupd = '2026/8/24'`. SHA-256 `53c2b294d0237631a93785a351ed3302fad4c167407727adaa2d9784245ce0d2` (1,537,899 bytes).
  - `community/aojiao/levels.js` (`-stage-values`) — identical to upstream at commit `0459ba134b9d6dc35f466b1419daf5be29731c6f` (2026-08-29); last changed 2026-08-28. SHA-256 `12aaf5cfb686380563cab8bd23d71f7af82ae46a7d2fe769d32c8d657cdddfe3` (60,791 bytes).
- **Used for:** every spirit's flat bonus (its only source); letter grades, style tags and places for ~14,800 items no other source has; the stage values below; and, for the ~14,600 items the wiki has no page for, how to get them, from the source column of `wardrobe.js`, translated through `data/acquisition-cn.json` (a code missing there fails the build) and marked `"cn":1` in `acquire.json`. A customization or evolution line the wiki gives without its base takes the base this table gives (31 items). The table describes the Chinese server, so these lines can differ from Global; event names are given only where the wiki names the same event on at least three quarters of the items both cover, a count the build checks against both sources every time, and an evolution or customization base the table files under another garment's family is left out. Where the wiki's item page gives one grade and this table and Nikki Calc give another alike, theirs is kept (406 grades on 285 items). Its suit column gives the suit of 122 items the wiki places in none, under the title of the wiki's suit page whose "cnwiki" field or link to the Chinese wiki gives the same Chinese name, and of 15 more whose other pieces the wiki places, all in one suit; where both place an item they agree on 25,667 of the 25,682, and the wiki is kept on the other 15. Items it files only as a suit's base (1,355, the pieces a suit's evolved items start from) are left without one. Where no wiki suit page gives an item's Chinese suit name, Nikki Calc's name for that suit is used (see below); one item, To Eternity, is in a suit no source names in English.
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
- **File (`-calc-suits`):** `nikkicalc/suits-v0.14.json`, its suit table (each suit's name and the items it holds), fetched 2026-09-21. SHA-256 `ff451d84dab62acdbf76078b5a7445d14e5121170789736a8b49f6d6f1ab3918` (220,450 bytes).
- **Licence:** none. **Permission:** in writing, from the maintainer, on 2026-09-26, on the conditions in [DATA-LICENSE.md](DATA-LICENSE.md).
- **Used for:** English item names where no other source has one (~15,800 items: those nikkiup2u3 names in Chinese only, and the 989 no other source has, 982 of which it also grades), and the name of 181600, Cookie Sweet Dream, which `data/id-corrections.json` shows in place of the wiki's Biscuits & Sweet Dream; every item's rarity (34,012 items), from the rarity code each record holds after its index, where 1 to 6 and 7 to 12 both mean 1 to 6 hearts; the set of valid Global item IDs, which keeps unreleased items out; and each item's sub-grades (+ and −), which set its stats within its letter grade. A sub-grade is used only where its letter and side agree with the item's own grade (169,994 of 170,025 graded stats); the rest keep their letter's value. For the 982 items no other source grades (`-calc-grades`), it also gives the letter grades (each sub-grade without its + or −), sides, wearable places (from each record's slot code) and style tags (from its style list; a mark outside that list, on 38 items, is left out). Where the wiki's item page gives one grade and nikkiup2u3 and Nikki Calc give another alike, theirs is kept (406 grades). Its suit table names the suits the wiki has no page for: the pieces nikkiup2u3 files under one Chinese suit take Nikki Calc's name for it when every piece Nikki Calc places in one suit gives the same name, and an item no other source places in a suit takes the one suit Nikki Calc lists it in (720 items, 47 suits); a name Nikki Calc gives two suits is never used. Where the other sources say only that an item is crafted, its recipe gives the ingredients (`-calc-recipes`, 1,755 items); where they name the ingredients, theirs are kept (its recipes agree with the wiki's on 3,336 of the 3,382 items both give). Tag awards stay priced on the letter grades.
- **Spelling:** its spelling of item names decides which hyphenated words in item names stay joined and which hyphens are spaced as separators. A hyphen written with a space on one side only is spaced on both (Moonlight - White). A hyphenated word in an item's name stays joined where Nikki Calc spells it joined for the same item. Otherwise a hyphen is spaced as a separator where Nikki Calc writes a separator there instead (a middot, or a hyphen with a space beside it) for the same item, whatever the length of the words or the number of hyphens (Fox Talk - Me, Passers-By - Red). Otherwise it stays joined where a lowercase letter follows it, and a word made of two or more letters, one hyphen and three or more letters is spaced as a separator (Doll Dress - Blue), unless it starts with a common prefix such as anti- or re-; every other hyphen is kept as written.
- **Not read:** item descriptions and every other field of the item batches except each item's attribute sides, sub-grades, slot code, style marks and recipe; the suit table's icons.
