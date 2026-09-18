-- Snacks picker colors.
--
-- The picker paints file rows, git status letters and match spans with its
-- own highlight groups, and most colorschemes never define them. Hard-coding
-- hexes here is what made every theme except ayu-dark look wrong, so the
-- colors now come from the active palette: `dotfiles theme` writes them into
-- lua/config/theme-active.lua alongside the colorscheme name. See
-- internal/theme/palette.go (Palette.derivePicker) for the derivation, and
-- themes/<name>/palette.toml [picker] to override a single color.
--
-- The fallback below is ayu-dark, matching plugins/colorscheme.lua, so a
-- checkout with no theme applied yet still looks the way it always has.
local tag_fallback = { tag = "#a3ccc7", attribute = "#e6b450" }

local fallback = {
  file = "#a3b7cc",
  folder = "#a3ccc7",
  hidden = "#a3c7ff",
  ignored = "#5c6370",
  match = "#cc7e7e",
  selection = "#b7c7a3",
  prompt = "#a3b7cc",
  git_added = "#b7c7a3",
  git_modified = "#a3b7cc",
  git_deleted = "#cc7e7e",
  git_renamed = "#a3ccc7",
  git_untracked = "#d4d4d4",
}

local c = fallback
local tag = tag_fallback
local ok, active = pcall(require, "config.theme-active")
if ok and type(active) == "table" then
  if type(active.picker) == "table" then
    c = vim.tbl_extend("force", fallback, active.picker)
  end
  if type(active.tag) == "table" then
    tag = vim.tbl_extend("force", tag_fallback, active.tag)
  end
end

-- Group names are the ones Snacks actually paints with; the earlier list
-- here spelled several of them wrong (SnacksPickerFolder, ...Selection,
-- ...FileHidden, ...GitStatusIgnored), so those lines never did anything.
-- SnacksPickerDir, the dim path prefix, is deliberately left to the
-- colorscheme -- every theme already renders it sensibly.
local groups = {
  SnacksPickerFile = { fg = c.file },
  SnacksPickerDirectory = { fg = c.folder },
  SnacksPickerPathHidden = { fg = c.hidden },
  SnacksPickerPathIgnored = { fg = c.ignored },
  SnacksPickerMatch = { fg = c.match, bold = true },
  SnacksPickerSelected = { fg = c.selection, bold = true },
  SnacksPickerPrompt = { fg = c.prompt, italic = true },
  SnacksPickerGitStatusAdded = { fg = c.git_added },
  SnacksPickerGitStatusStaged = { fg = c.git_added },
  SnacksPickerGitStatusModified = { fg = c.git_modified },
  SnacksPickerGitStatusDeleted = { fg = c.git_deleted },
  SnacksPickerGitStatusUnmerged = { fg = c.git_deleted },
  SnacksPickerGitStatusRenamed = { fg = c.git_renamed },
  SnacksPickerGitStatusCopied = { fg = c.git_renamed },
  SnacksPickerGitStatusUntracked = { fg = c.git_untracked },
}

-- JSX markup.
--
-- Treesitter gives the element name, the attribute name and the brackets their
-- own captures, but a colorscheme that predates treesitter defines none of
-- them and they fall through to plain text -- so `<div data-slot="card">` is
-- one flat color and the markup stops being readable as markup. Nord leaves
-- the element and attribute names at the foreground; Iceberg leaves the
-- element name unset and component names at the foreground; the Jellybeans
-- port leaves attribute names at the foreground.
--
-- Only fill a capture the colorscheme actually left unstyled: unset, or the
-- exact foreground color. Every theme that styles them properly is untouched,
-- and no threshold has to be guessed -- the broken cases are exact matches.
local function fg_of(name)
  local h = vim.api.nvim_get_hl(0, { name = name, link = false })
  return h and h.fg
end

local function paint_tags()
  local plain = fg_of("Normal")
  if not plain then
    return
  end
  local unstyled = function(name)
    local v = fg_of(name)
    return v == nil or v == plain
  end

  -- the element name, and the two captures that conventionally follow it
  if unstyled("@tag") then
    for _, g in ipairs({ "@tag", "@tag.builtin", "@constructor" }) do
      if unstyled(g) then
        vim.api.nvim_set_hl(0, g, { fg = tag.tag })
      end
    end
  else
    for _, g in ipairs({ "@tag.builtin", "@constructor" }) do
      if unstyled(g) then
        vim.api.nvim_set_hl(0, g, { fg = fg_of("@tag") and string.format("#%06x", fg_of("@tag")) or tag.tag })
      end
    end
  end

  -- the attribute name has to differ from the element name as well, or
  -- `data-slot` reads as part of the tag
  local tagfg = fg_of("@tag")
  local attr = fg_of("@tag.attribute")
  if attr == nil or attr == plain or (tagfg ~= nil and attr == tagfg) then
    vim.api.nvim_set_hl(0, "@tag.attribute", { fg = tag.attribute })
  end
end

local function paint()
  for name, opts in pairs(groups) do
    vim.api.nvim_set_hl(0, name, opts)
  end
  paint_tags()
end

-- Loading a colorscheme clears every highlight, so repaint after each one
-- rather than only at startup.
vim.api.nvim_create_autocmd("ColorScheme", {
  group = vim.api.nvim_create_augroup("dotfiles_picker_highlights", { clear = true }),
  callback = paint,
})
paint()
