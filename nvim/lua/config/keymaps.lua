-- Keymaps are automatically loaded on the VeryLazy event
-- Default keymaps that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/keymaps.lua
-- Add any additional keymaps here

vim.keymap.set("n", "gf", function()
  return require("obsidian").util.gf_passthrough()
end, { noremap = false, expr = false, buffer = true })

vim.keymap.set("n", "<leader>ch", function()
  return require("obsidian").util.toggle_checkbox()
end, { buffer = true })

vim.keymap.set("n", "<cr>", function()
  return require("obsidian").util.smart_action()
end, { expr = true, buffer = true })
