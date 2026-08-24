return {
  "obsidian-nvim/obsidian.nvim",
  lazy = false,
  ft = "markdown",
  dependencies = {
    "nvim-lua/plenary.nvim",
  },
  opts = function()
    local workspaces = {
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
    }
    -- The vault folders must exist or obsidian.nvim errors on startup.
    -- Create them so a fresh machine works out of the box.
    for _, ws in ipairs(workspaces) do
      vim.fn.mkdir(ws.path, "p")
    end
    return {
      workspaces = workspaces,
      legacy_commands = false,
      picker = {
        name = "snacks.pick",
      },
    }
  end,
}
