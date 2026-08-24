-- One spec per theme supported by `dotfiles theme`; all lazy. The active
-- colorscheme comes from lua/config/theme-active.lua, which `dotfiles theme`
-- writes on this machine (never synced; see unit.nvim ignore). Fallback when
-- it is absent: ayu-dark. Running instances keep their colors; new ones pick
-- up the active theme.
local active = "ayu-dark"
local ok, t = pcall(require, "config.theme-active")
if ok and type(t) == "table" and t.colorscheme then
  active = t.colorscheme
end

return {
  { "Shatur/neovim-ayu", name = "ayu", lazy = true, opts = { mirage = true } },
  { "cocopon/iceberg.vim", lazy = true },
  { "nanotech/jellybeans.vim", lazy = true },
  { "rebelot/kanagawa.nvim", lazy = true },
  { "projekt0n/github-nvim-theme", name = "github-theme", lazy = true },
  { "shaunsingh/nord.nvim", lazy = true },
  { "folke/tokyonight.nvim", lazy = true },
  { "LazyVim/LazyVim", opts = { colorscheme = active } },
}
