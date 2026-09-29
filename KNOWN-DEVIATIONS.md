# Known deviations

Where NikkiBase knowingly differs from the game, or from what a source states.
"The stage source" below is nikkiup2u3_data by seal100x; "the exact stage
values" are nikkiup2u3 by 傲娇攻略组 (see [SOURCES.md](SOURCES.md)).
Each entry says what it costs a player. Nothing here is a silent allowance: the
build fails if one of these gaps widens, and the ceilings live in
`data/coverage.json`.

## Top ranked in-game scores

These are real contest results, reported on Reddit in September 2026, set
against NikkiBase's whole-catalogue best possible score. They are not a
like-for-like comparison, so they are not used to judge accuracy: ranked
players use skills (Charming Smile multiplies one attribute by up to 1.78; the
table below is without them), their items have exact stats where NikkiBase
prices each item from its sub-grade, and a ranked score may come from the
other difficulty. They are
recorded so the gaps stay visible. `golden/` fails if a gap over 10% is missing
from this list. The like-for-like check is against Nikki Calc's best outfit,
which NikkiBase matches to within 1% on Story 1-1.

| Stage | NikkiBase best possible | #1 ranked | Gap |
|---|---|---|---|
| Story 1-1 | 119,118 | 183,260 | -35% |
| Story 1-2 | 139,269 | 194,384 | -28% |
| Story 4-1 | 125,052 | 239,979 | -48% |
| Story 7-7 | 163,665 | 212,890 | -23% |

Best possible is at Princess difficulty, which on these stages other than 4-1
scores the same as Maiden.

- **Story 1-1 and 1-2**: mostly skills. With Skills switched on at max level, the
  best possible is 163,333 (-11%) and 184,694 (-5%).
- **Story 7-7**: most of its score is tag points, which skills don't raise:
  166,617 (-22%) with them. The rest is unexplained.
- **Story 4-1**: at Princess its Rain tag pays C x0.5; at Maiden it pays SS x1,
  and the best possible is 186,680 (-22%), or 216,903 (-10%) with skills. The
  Princess gap is larger than skills can explain, so the ranked score was
  probably set at Maiden.

## Tag stages (12)

Story 2-7, 3-6, 4-8, 4-12, 5-7, 5-11, 5-Side 3, 6-8, 6-Side 3, 7-7, 7-8 and
7-Side 3 have weights a fraction of an ordinary stage's (7 to 11 against about
130) and fixed tag awards of 6,000 to 35,000, so wearing the tagged items
decides the outfit. That is how the game builds them: the exact stage values
state these figures. Each is acknowledged in `data/stage-corrections.json` at
its weight sum and number of awards, and the build fails if either moves.

**Cost:** none; recorded because the build's outlier check would otherwise
refuse them.

## Two stages far outside the usual range

- **Story 15-9** — its weights sum to 1,518 where the median Story stage sums
  to about 130, with ordinary tag multipliers, and it pays 150,000 and 50,000
  for its tags. The exact stage values state the same, and the stage source
  describes a stage passed by wearing required items and accumulating points
  over repeated clears.
  **Cost:** every score here is roughly ten times its neighbours'. The required
  items and repeated clears are not modelled.
- **Story II-4-7** — ordinary weights, but both tags pay 149,996, a multiplier of
  10 where the other Volume II stages use 0.1 to 1. The exact stage values
  state the same size.
  **Cost:** none beyond the stage's own design: the tagged items decide the
  outfit.

Each is excused from one check at the value recorded in
`data/stage-corrections.json`; if a source moves it by more than 1%, the build
fails until someone looks again.

## Maiden and Princess

Story stages score the same at both difficulties except ten: 2-Side 2, 3-10,
3-11, 4-1, 5-12, 6-7, 6-9, 6-10, 7-9 and 7-Side 3, whose Maiden tags or
weights differ. Both are modelled; the difference and its basis are recorded in
`data/stage-difficulty.json`. Three Maiden figures are chosen rather than stated:

- **6-9** — the source prices Maiden's two tags by grade, C x0.5 each. They
  are paid at Princess's exact factors instead: 35.5 for Republic of China, and
  for Cheongsam 31.4, Princess's factor for Multicultural, the tag it replaces.
- **6-10** — the source gives Maiden's weights and says its tags weigh more,
  with no figure. Princess's awards are kept on Maiden's lighter weights, which
  makes them weigh about twice as much against the weights.
- **7-Side 3** — the source says Maiden's tag is halved; the multiplier is
  halved (SS x10 against SS x20) and priced on Maiden's weights.

Where the difficulties differ only in which items score F, the difference is
not modelled (see below).

**Cost:** on those three stages Maiden's tag award may be off by more than the
usual estimate.

## Co-op tag awards

The exact stage values do not cover co-op stages, so their weights and tag
awards come from the stage source's grade and multiplier.

**Cost:** a co-op tag award can be off by more than other stages' (about 15%
in either direction).

## Items listed by name only

7 items the game lists are listed by name only. 5 carry an ID of one slot
while Nikki Calc and the wiki's suit pages file them under another: 30961
Vanity Fair, a coat filed as a top, and the accessories 81327 Moonlight
Serenade, 82599 Bamboo Leaf Bracelet, 83221 Maple Viewing and 85735 Glistening
Aqua, filed as a coat, leglets, shoes and leglets. How the game sizes such an
item's stats is unknown. 2 were never released: 85647 Magic Bubble and 181794
Hymn of Dawn, both called unused by the wiki (`data/id-corrections.json`).
`items.json` lists each under Nikki Calc's name with its rarity and no stats.

**Cost:** these items are never recommended and score nothing. A player who
owns one of the five gets no credit for it.

## Items graded by Nikki Calc alone

982 items released on Global, most of them Global-only, have no grades in the
Love Nikki Wiki or nikkiup2u3. Their letters, sides, wearable places and style
tags come from Nikki Calc's item data: each sub-grade rounded to its letter
(S- is S), its slot code for the place, and its style list for the tags.
Where Nikki Calc and the other sources both grade an item, they agree on 99.7%
of grades and 99.98% of sides, and its place matches ours on 99.95% of items.
Nikki Calc marks 38 items in Korean-themed suits with a style its own list does
not name; that mark is left out, and no Global stage awards such a style.

**Cost:** one of these items can be a letter off on a stat, or miss a style
tag, somewhat more often than an item two sources grade.

## Item-page grades two sources contradict

On 406 grades of 285 items, the wiki's item page gives one grade while
nikkiup2u3 and Nikki Calc agree on another; the build takes theirs, 21 times on
the other side of the pair. On the 35 pages with three or more such grades,
most rows are another item's copied onto the page: a sibling in the same suit,
a recolour, or the spirit it evolves into.

**Cost:** none known. If both sources were wrong where the wiki is right, the
item would be a letter off on that stat.

## Items in no suit

`items.json` places 30,194 of 34,012 items in 2,293 suits: 29,337 as the
Love Nikki Wiki's suit and item pages place them, 122 the wiki places in none
that nikkiup2u3 files under a suit the wiki names in Chinese, 15 that nikkiup2u3
files with pieces the wiki already places, and 720 under Nikki Calc's name for
47 suits the wiki has no page for yet (the wiki is read as of 2026-09-22). The
other 3,818 are in no suit: 1,355 that nikkiup2u3 files only as a suit's base,
1 whose Chinese suit no source names in English (To Eternity), and 2,462 no
source places.
A suit is named as its wiki page is titled, and the site shows it without the
qualifier the wiki adds to tell suits apart ("Star Shadow", not "Star Shadow
(Hidden Suit)"); two suits with one name, such as the two Night Rose suits,
stay apart. The wiki's pack pages (event
items, chests and sales) are not suits: 579 items only a pack page lists, and
576 of them are in no suit (nikkiup2u3 files the other 3 under Green Wind).
29 items are listed on more than one suit page; each takes the suit its own
page names, unless that suit's page leaves it out, and otherwise the first
title in alphabetical order.

**Cost:** an item in no suit never shows in Worth getting's suit view, and a
suit that misses a piece is ranked without it.

## Scores are estimates

The game publishes letter grades, not per-item numbers. NikkiBase prices each
item from its sub-grade (S-, S, S+ and so on): a + or - moves a stat 43% of the
way to the next letter's value, and each letter keeps its average. On the few
single stats that have been measured exactly this is within about 1.5%, and
the best possible outfit on Story 1-1 is within 1% of Nikki Calc's. Two items
sharing a sub-grade cannot be told apart, and the 35 stats whose sub-grade
names another letter or side than our grade keep the letter's value. See the
footer note in the product.

## Content ceiling

Every stage released on the Global server as of 2026-09-23 is covered: Story
Volumes I and II, Volume III chapters 1-2 and the first seven stages of
chapter 3, Commission Acts 1-20, Arena and Co-op (`data/stage-scope.json`).
Event and Dream Weaver stages are not. The stage sources are kept against the
Chinese server, which runs ahead; the scope file is widened by hand as Global
releases more.

## Stage rules

Items a stage requires (`data/stage-rules.json`) are enforced: every best
outfit wears them, and the site says when a wardrobe lacks one. Stages where
some items or styles score F are flagged, but those rules are not modelled. A
best outfit on such a stage may include an item that scores F there.
