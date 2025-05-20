return {
  "obsidian-nvim/obsidian.nvim",
  lazy = false,
  ft = "markdown",
  dependencies = {
    "nvim-lua/plenary.nvim",
  },
  opts = {
    workspaces = {
      {
        name = "random",
        path = os.getenv("HOME") .. "/Documents/notes/random",
      },
      {
        name = "university",
        path = os.getenv("HOME") .. "/Documents/notes/university",
      },
      {
        name = "work",
        path = os.getenv("HOME") .. "/Documents/notes/work",
      },
      {
        name = "dev",
        path = os.getenv("HOME") .. "/Documents/notes/dev",
      },
    },
    {
      picker = {
        name = "snacks.pick",
      },
    },
    completion = {
      blink = true,
    },
    mappings = {
      ["gf"] = {
        action = function()
          return require("obsidian").util.gf_passthrough()
        end,
        opts = { noremap = false, expr = false, buffer = true },
      },
      ["<leader>ch"] = {
        action = function()
          return require("obsidian").util.toggle_checkbox()
        end,
        opts = { buffer = true },
      },
      ["<cr>"] = {
        action = function()
          return require("obsidian").util.smart_action()
        end,
        opts = { expr = true, buffer = true },
      },
    },
  },
}
