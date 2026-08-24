-- Quick access to personal cheatsheets kept in the Obsidian `dev` vault.
-- <leader>sK sits next to LazyVim's <leader>sk (keymaps), which is the same
-- "how do I do this again?" reflex.
local CHEATS = vim.fn.expand("~/Documents/notes/dev/cheatsheets")

return {
  "folke/snacks.nvim",
  keys = {
    {
      "<leader>sK",
      function()
        if vim.fn.isdirectory(CHEATS) == 0 then
          return vim.notify("No cheatsheets dir: " .. CHEATS, vim.log.levels.WARN)
        end
        require("snacks").picker.files({
          cwd = CHEATS,
          title = "Cheatsheets",
          -- these are notes, not code: preview matters more than the list
          layout = { preset = "default" },
        })
      end,
      desc = "Cheatsheets",
    },
    {
      "<leader>sk",
      function() require("snacks").picker.keymaps() end,
      desc = "Keymaps",
    },
  },
}
