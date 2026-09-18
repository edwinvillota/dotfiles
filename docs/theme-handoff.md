# Theme pass — handoff

Working notes for continuing the theme work in a fresh session. Everything
below is uncommitted on branch `improve-dadbob-goto` (which is unrelated to
this work — branch before committing).

## Where things stand

`dotfiles theme` drives nine palettes into wezterm, zellij, nvim, fzf/bat,
yazi, btop, gh-dash and VisiData. This pass fixed a class of bug where a tool
paints its own colors instead of asking the colorscheme, so every theme
inherited whatever was hard-coded for the one theme it was tuned on.

### Done and verified

1. **Snacks picker colors come from the palette.** `nvim/lua/config/highlights.lua`
   hard-coded eighteen hexes. `dotfiles theme` now writes them into
   `lua/config/theme-active.lua`; `highlights.lua` reads them and repaints on
   every `ColorScheme` event. Derivation: `Palette.derivePicker`. A palette may
   override any single color under `[picker]`; `ayu-dark` pins the whole set to
   its hand-tuned values.
2. **Six picker group names were wrong** and never did anything
   (`SnacksPickerFolder`, `...Selection`, `...FileHidden`, `...FolderHidden`,
   `...GitStatusIgnored`, `...GitStatusHidden`). Corrected to the groups Snacks
   actually paints.
3. **Match spans must be findable.** Contrast ratio is blind to hue: on
   kanagawa-wave the accent scored 1.16 against the foreground and was
   invisible. `deltaE` (CIE76) now enforces a perceptual floor, `minMatchDelta`.
4. **Git status colors must differ from each other.** github-dark-colorblind
   moves red to orange, landing it beside its yellow; both kanagawa palettes put
   added and renamed in the same green. `minGitDelta` separates them.
5. **VisiData drew every header white.** It composites `color_bottom_hdr` at
   precedence 5 over a one-line column header, so `color_default_hdr` — where
   the accent was — never reaches the screen.
6. **VisiData had no hue anywhere.** The 256-color cube has almost no dark
   saturated cells, so chrome mixed from background toward foreground quantized
   onto the grayscale ramp. Separators now walk toward accent2 instead.
7. **VisiData's background is the terminal's.** It calls
   `curses.use_default_colors()`, so `color_default`, `color_column_sep` and
   `color_guide_unwritten` carry no background and inherit the real theme color.
   Naming an index quantized it: nord's `#2e3440` became `#3a3a3a`,
   tokyo-night's `#1a1b26` became `#1c1c1c`, and every theme's sheet was the
   same gray.
8. **VisiData colors by column**, mirroring what `nvim/lua/plugins/csv.lua`
   already does for csvview. `color_col0..5` come from the palette; the
   colorizer lives in `visidata/.visidatarc`. Separation is checked in painted
   index space, not source hex — nord's bright blue and bright cyan both
   quantize to 109.
9. **Jellybeans migrated** from `nanotech/jellybeans.vim` to `WTFox/jellybeans.nvim`.
   The original is a classic Vim colorscheme with no treesitter groups, so every
   JSX capture fell through to one green.
10. **JSX markup is filled in when the colorscheme leaves it unstyled.**
    `highlights.lua` fills `@tag`, `@tag.builtin`, `@constructor` and
    `@tag.attribute` only when they are unset or exactly the `Normal`
    foreground. Fixes nord (element and attribute names were plain), iceberg
    (element name unset, component names plain) and the jellybeans port
    (attribute names plain). Themes that style them properly are untouched —
    verified on tokyo-night, ayu-dark and kanagawa-wave.

### Tests

`internal/theme` has ~32 tests. The ones added this pass:
`TestPickerColorsComplete`, `TestAyuPickerPinned`, `TestGitStatusColorsDistinct`,
`TestPickerMatchStandsOut`, `TestVisiDataHeaderCarriesAccent`,
`TestVisiDataIsNotMonochrome`, `TestVisiDataSeparatorsAreVisible`,
`TestVisiDataPanelTextIsLegible`, `TestTagColorsAreDistinct`.

Helpers worth knowing: `deltaE` / `labOf` (perceptual distance — use this, not
contrast, when asking "do these read as different colors"), `contrastHex` /
`contrastOnBody` (WCAG against the true terminal background), `hasHue`,
`idxHex`, `Xterm256`.

## Open items, in the order I would do them

### 1. VisiData accents converge — needs a decision

Seven of nine themes paint their accent as one of two near-identical golds,
because the cube quantizes them all to index 179 or 180:

```
179  ayu-dark, github-dark, github-dark-colorblind, tokyo-night
180  iceberg, kanagawa-dragon, kanagawa-wave
```

Error from the true palette accent runs deltaE 5.5 to 16.2. The accent is the
dominant colored element (headers, cursor cell, selected rows, sidebar titles,
keystrokes), so this is most of why VisiData still reads similarly across
themes.

**The fix** is ANSI slots 0-15, which the terminal remaps to the exact palette
colors — zero quantization for eight of nine themes. Map a role to a slot only
when its hex exactly matches one of the palette's 16 ANSI colors, and fall back
to `Xterm256` otherwise (ayu-dark overrides `accent`, so it would keep today's
behavior).

**Why it needs a decision:** this reverses a documented rule. The README says
VisiData uses only cube indices 16-255 "so it is immune to the remapping", and
`TestVisiDataAvoidsRemappedANSI` enforces it. The original reasoning was that
VisiData's *stock* defaults use ANSI names blindly; using them deliberately,
from the palette that defines the remapping, is a different case. The real cost
is that it only holds in a terminal carrying our palette — though item 7 above
already introduced that same dependency, so this is arguably now consistent
rather than new risk.

If accepted: update the README bullet and replace the guard test with one that
asserts slots are only used where the hex matches the palette exactly.

### 2. Ayu Dark's hidden files are brighter than its normal files

Its picker colors are pinned to hand-tuned values where hidden paths sit at
11.2 contrast against the picker background and normal file names at 9.4. Every
derived theme runs the ladder the other way (file > hidden > ignored). One line
in `themes/ayu-dark/palette.toml` fixes it. Left alone because it changes a look
the user approved.

### 3. GitHub Dark's terminal and editor backgrounds disagree

Palette paints `#010409`, the colorscheme paints `#0d1117`, deltaE 4.4. Seven
of nine themes match exactly and jellybeans is within 1.3, so this is a defect
by the repo's own standard — it shows as a seam between the editor and the
zellij bars. Changing `primary.background` fixes it but shifts an approved
theme, and ripples into the committed assets (`dotfiles theme render`).

### 4. Nord's colorscheme plugin is stale

`shaunsingh/nord.nvim` was last pushed 2024-06-25. `gbprod/nord.nvim` (313
stars, 2026-04-14) is maintained. Item 10 papers over the JSX symptom; swapping
would fix the cause and likely other unstyled captures. Same tradeoff as the
jellybeans migration: a different author's reading shifts the whole theme.

### 5. Iceberg has no good Lua port

`cocopon/iceberg.vim` is maintained (2385 stars) but vimscript.
`oahlen/iceberg.nvim` is small and stale (35 stars, 2025-03). Item 10 covers the
JSX gap; there may be other unstyled captures worth auditing.

### 6. Ignored files only dim in the explorer

Snacks tags an item as hidden or ignored in its explorer source only, so a
`node_modules` hit in the file picker renders at full brightness. Upstream
behavior, not reachable from the palette. Would need an upstream change or a
custom matcher.

## The screenshot harness

Not in the repo — it lives in the session scratchpad and is worth rebuilding if
you continue this work. Two renderers:

- **nvim**: attaches a UI to a headless nvim over its msgpack UI protocol,
  drives real keystrokes, and renders the exact per-cell RGB to PNG. Colors come
  from nvim's own `rgb` highlight attributes, so the PNG is what the editor
  would paint.
- **VisiData**: runs `vd` in a real pty, parses output with `pyte`, and paints
  each cell with what the terminal would show — indices 0-15 from the palette's
  ANSI slots, 16-255 from the fixed cube. This is what makes the quantization
  traps visible.

Both run against a sandbox `XDG_CONFIG_HOME` / `HOME` so the live config is
never touched. A scratch test hook (`GEN_THEME` / `GEN_OUT` / `GEN_KIND` writing
`NvimActive` or `VisiData` output to a file) is what lets the harness generate a
theme's config without running `dotfiles theme` for real; it was deleted after
each use to keep the tree clean.

Gotchas that cost time:

- Snacks pickers close when another window takes focus. Drive them with
  `auto_close=false`, or accept flakiness.
- The picker's background is `SnacksPickerList`, **not** `NormalFloat`. I
  measured a false regression against the wrong group once.
- `Snacks.picker.git_status()` closes with "No results" on a clean worktree, so
  point the git scenario at a repo that has changes.
- pyte spells SGR 33 `brown`, not `yellow`.
- Counting distinct colors across a capture set is a bad metric — it hid nord's
  JSX bug, because the count was padded by the string and delimiter colors while
  the three highest-volume captures all sat on plain foreground. Measure against
  the plain foreground instead.

## Review pages

`/tmp/theme-shots/themes.html` (all nine themes, four frames each) and
`/tmp/theme-shots/compare.html` (the github-dark before/after). Both are local
files with images alongside; regenerate rather than trust them, since they
predate items 8, 9 and 10.
