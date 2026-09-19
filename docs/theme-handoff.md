# Theme pass — handoff

Working notes for continuing the theme work in a fresh session. The pass is
committed on branch `theme-pass-visidata-ansi`, cut from `improve-dadbob-goto`
(which is unrelated to this work and where it was all sitting uncommitted).

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

11. **VisiData takes the palette's ANSI slots where the hex matches exactly.**
    Every color used to come from the fixed cube on the rule that 16-255 is
    immune to the terminal's remapping, but the cube has no cell within deltaE
    5 of most palette accents: seven of nine themes painted theirs as one of
    two near-identical golds (179/180, error up to 16.2). `Palette.Paint` now
    returns a slot 0-15 when the hex is *exactly* one of the palette's 16 ANSI
    colors — the terminal then hands back the color we asked for — and the cube
    otherwise. 58 of 63 role/theme pairs are slot-exact; the accent is exact on
    eight of nine (ayu-dark pins an accent that is not one of its ANSI colors).
    Contrast floors now measure `PaintedHex`, which is what reaches the screen.
    That exposed one real case: nord's true accent2 is darker than the cube
    cell that stood in for it, dropping text on the active status bar from 4.6
    to 4.23, so a solid role bar now lifts toward the foreground until its text
    clears — a no-op on the other eight.
12. **ayu-dark's picker ladder descends.** Its pinned `hidden` was louder than
    its file names (11.2 vs 9.4). Now `#7d99c5`, contrast 6.56, between file
    9.4 and ignored 3.2. Measured in a real explorer: `.github` 11.2 -> 6.65.
13. **github-dark's background matches its colorscheme.** The palette painted
    `#010409` where the colorscheme paints `#0d1117` (confirmed from a live
    nvim session), which showed as a seam against the zellij bars. Committed
    zellij/btop/yazi assets regenerated.

### Tests

`internal/theme` has ~32 tests. The ones added this pass:
`TestPickerColorsComplete`, `TestAyuPickerPinned`, `TestGitStatusColorsDistinct`,
`TestPickerMatchStandsOut`, `TestVisiDataHeaderCarriesAccent`,
`TestVisiDataSlotsAreExact` (replaced `TestVisiDataAvoidsRemappedANSI`),
`TestPickerLadderDescends`,
`TestVisiDataIsNotMonochrome`, `TestVisiDataSeparatorsAreVisible`,
`TestVisiDataPanelTextIsLegible`, `TestTagColorsAreDistinct`.

Helpers worth knowing: `deltaE` / `labOf` (perceptual distance — use this, not
contrast, when asking "do these read as different colors"), `contrastHex` /
`contrastOnBody` (WCAG against the true terminal background), `hasHue`,
`idxHex`, `Xterm256`.

## Open items, in the order I would do them

Items 1-3 of the previous list are done (11-13 above). Items 4 and 5 were
measured and closed with a decision; 6 is upstream. What is left:

### 1. tokyo-night draws JSX tags and attributes 24.8 apart

Measured on the same JSX file as the nord comparison, LSP off: tokyo-night
paints `@tag` `#2ac3de` and `@tag.attribute` `#73daca`, deltaE 24.8 — just
under the line where two colors read apart at text size. The tag fill-in
(item 10) deliberately does not touch it, because the colorscheme *does* style
both; the rule only fills what is unset or exactly the plain foreground.
Widening the rule to "also fill when the attribute is within deltaE 25 of the
tag" would fix it, and would be the same change needed to adopt gbprod's nord.
It reaches every theme, so it wants its own measured pass.

### 2. Snacks tags hidden and ignored in the explorer source only

Confirmed by measurement this pass, not just from the source: in
`Snacks.picker.files({hidden=true, ignored=true})` a dotfile and an ordinary
file are both painted `SnacksPickerFile` (`.gitignore` and `README.md` both
`#a3b7cc` on ayu-dark), while the same items in `Snacks.picker.explorer()` do
get `hidden` and `ignored`. Not reachable from the palette; it needs an
upstream change or a custom matcher.

### 3. zellij paints the keybinding letter in `red` on its light ribbon

The bottom bar's `<n>` / `<f>` characters are drawn in the theme's `red` on the
`white` ribbon: 2.25 on jellybeans and 1.62 on ayu-dark, so this is
cross-theme, not a jellybeans defect. Fixing it means darkening `red` in the
zellij theme when it does not clear the ribbon, which repaints ayu-dark and
github-dark, so it needs a decision first. Measure it with the zellij frame in
`ptyshot.py` — the numbers above come from real sessions.

### Closed this pass

- **nord's colorscheme plugin stays `shaunsingh/nord.nvim`.** `gbprod/nord.nvim`
  (HEAD 2026-04-14, versus shaunsingh's 2023-12-19) does style more captures —
  plain-foreground share on the JSX sample drops from 27.5% to 21.2% — but it
  collapses the markup separation the last pass bought: tag vs attribute goes
  from deltaE 59.2 to 21.2 and the `<` delimiter merges into the tag color.
  The maintenance argument did not outweigh the visible regression.
- **iceberg needs no port swap.** After the tag fill-in it reads tag vs
  attribute at deltaE 62.5, the widest of the nine, and its plain-foreground
  share (27.3%) is ordinary identifiers, in line with nord. `oahlen/iceberg.nvim`
  is small and stale; `cocopon/iceberg.vim` plus the fill-in is the better deal.
- **yazi's icon colors.** yazi paints the glyph in front of every row from its
  own [icon] table, which hard-codes hexes -- the generic folder icon is
  #03a9f4 under every theme. The flavor now restates its fallback icon rules
  with palette colors (`TestYaziIconsComeFromPalette`). Directory *names* were
  already palette-driven: they take accent2, which every palette derives from
  its bright blue, so folders stay blue under all nine themes by design, the
  same convention as yazi's own default and LS_COLORS. A palette-level
  `folder` role would be the change if that should vary per theme.
- **jellybeans' zellij bars.** Its ANSI black (#929292) and bright black
  (#bdbdbd) are light, so zellij's bars had no dark end (1.86-2.31 contrast)
  and inactive tabs were as loud as the active one. `Palette.uiDark` and
  `Palette.dimmed` fix both; see `TestZellijUIHasDarkAnchor`, `TestDimReadsDimmer`.

## The screenshot harness

Rebuilt this pass, in the session scratchpad again (`vdshot.py`, `nvimshot.py`,
`measure.py`, `montage.py`, a `venv` with pyte/pillow/pynvim). Worth rebuilding
if you continue this work. Two renderers:

- **nvim** (`nvimshot.py`): attaches a UI to a headless `nvim --embed` over
  msgpack-rpc, drives real keystrokes and ex commands, and renders the exact
  per-cell RGB to PNG from nvim's own `hl_attr_define` attributes.
- **yazi cannot be captured in the pty harness at all**: yazi 26 silently falls
  back to its built-in theme there, official flavors included, so every colour
  it paints is a default and looks identical under every palette. Drive the
  real terminal instead: `wezterm cli spawn --new-window -- yazi <path>`, poll
  `wezterm cli get-text --pane-id <id> --escapes` until the listing appears,
  then `wezterm cli kill-pane`. Note it emits truecolor with *colon*
  subparameters (`38:2::r:g:b`), which a `38;2;` regex misses, and always
  confirm which flavor is live before reading colors off a capture -- two of
  mine showed another theme's palette because the probe raced the config.
- **any TUI** (`ptyshot.py`): the same pty renderer as `vdshot.py`, but it
  takes a command, so it also captures zellij (run it against a sandbox `HOME`
  with the repo config; on macOS zellij's plugin permissions live in
  `~/Library/Caches/org.Zellij-Contributors.Zellij/permissions.kdl`, not under
  `XDG_CACHE_HOME`, and zjstatus will block on a permission prompt without it).
- **VisiData** (`vdshot.py`): runs `vd` in a real pty, parses output with
  `pyte`, and paints each cell with what the terminal would show — indices 0-15
  from the palette's ANSI slots, 16-255 from the fixed cube. This is what makes
  the quantization traps visible.

Both write a `<out>.json` beside the PNG with every cell's text and attributes,
which is what the measurements are computed from — look at the picture, but
measure the dump.

Both run against a sandbox `XDG_CONFIG_HOME` / `HOME` so the live config is
never touched. A scratch test hook (`GEN_THEME` / `GEN_OUT` / `GEN_KIND` writing
`NvimActive` or `VisiData` output to a file) is what lets the harness generate a
theme's config without running `dotfiles theme` for real; it is deleted after
each use to keep the tree clean. For before/after work, copy the repo to the
scratchpad and force the old behavior there (this pass: `Palette.Paint` returning
`Xterm256(hex)` unconditionally) — then both sides come out of the same code path.

Gotchas that cost time:

- Snacks pickers close when another window takes focus. Drive them with
  `auto_close=false`, or accept flakiness.
- The picker's background is `SnacksPickerList`, **not** `NormalFloat`. I
  measured a false regression against the wrong group once.
- `hidden` and `ignored` only reach the screen in `Snacks.picker.explorer()`.
  In `files()` a dotfile is painted `SnacksPickerFile` like any other, so an
  ayu-dark ladder fix is invisible there.
- Snacks is lazy-loaded: `require("lazy").load({plugins={"snacks.nvim"}})`
  before calling into it, or `Snacks` is nil.
- pynvim only accepts requests from its loop thread. Run `nvim.run_loop` on the
  main thread with a `setup_cb` that attaches the UI and starts a driver
  thread, and push every action through `nvim.async_call`.
- `nvimshot.py` chdirs into the scenario's cwd, so resolve output paths to
  absolute *before* that. Getting this wrong silently wrote a sandbox into the
  repo root once.
- Copying `~/.local/share/nvim/lazy` into a sandbox needs the `.git` dirs;
  without them lazy cannot install or update anything.
- Comparing colorschemes: stop the LSP clients and disable inlay hints first
  (`vim.lsp.stop_client`, `vim.lsp.inlay_hint.enable(false)`), or semantic
  tokens make one run look better styled than the other for no theme reason.
- `Snacks.notifier.hide()` clears the mason popups that otherwise sit on top of
  the code you are measuring.
- pyte spells SGR 33 `brown`, not `yellow`.
- Counting distinct colors across a capture set is a bad metric — it hid nord's
  JSX bug, because the count was padded by the string and delimiter colors while
  the three highest-volume captures all sat on plain foreground. Measure against
  the plain foreground instead. A second version of the same trap this pass:
  "min distance between any two colors in a captured row" read as a column-cycle
  regression on iceberg, but the offending pair was the *cursor column's*
  foreground against a column color — `color_current_col` outranks the
  colorizer. Measure the thing you changed, not whatever is on screen near it.

## Review pages

Regenerate rather than trust any of these; they are scratchpad files that go
away with the session. This pass produced `visidata-ansi.html` (all nine
themes, VisiData before/after), `shots/compare-header.png` (the same, as one
image) and the nvim frames `shots/nvim-explorer-{old,new}.png` and
`shots/jsx-*.png`.
