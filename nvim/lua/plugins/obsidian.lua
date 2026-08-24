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
    legacy_commands = false,
    picker = {
      name = "snacks.pick",
    },
  },
}
