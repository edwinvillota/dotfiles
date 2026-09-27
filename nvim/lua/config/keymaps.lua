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

-- Mute Harper in the current buffer, e.g. while writing in Spanish.
local function harper_ns()
  local client = vim.lsp.get_clients({ bufnr = 0, name = "harper_ls" })[1]
  return client and vim.lsp.diagnostic.get_namespace(client.id)
end

Snacks.toggle({
  name = "Harper Grammar",
  get = function()
    local ns = harper_ns()
    return ns ~= nil and vim.diagnostic.is_enabled({ bufnr = 0, ns_id = ns })
  end,
  set = function(state)
    local ns = harper_ns()
    if ns then
      vim.diagnostic.enable(state, { bufnr = 0, ns_id = ns })
    end
  end,
}):map("<leader>uH")
