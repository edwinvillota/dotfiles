-- Floating-terminal launchers for the GitHub TUIs.
-- <leader>gO -> gh-dash (PRs/issues), <leader>gR -> dispatch, <leader>gW -> watch a run.
-- gh-dash moved off <leader>gD: diffview.nvim owns that for staged changes.
local function gh_term(cmd, title)
  return function()
    require("snacks").terminal(cmd, {
      cwd = vim.fn.getcwd(),
      win = {
        style = "terminal",
        border = "rounded",
        width = 0.9,
        height = 0.9,
        title = " " .. title .. " ",
        title_pos = "center",
      },
    })
  end
end

return {
  "folke/snacks.nvim",
  keys = {
    { "<leader>gO", gh_term("gh dash", "GitHub Dash"), desc = "GitHub Dashboard (gh-dash)" },
    {
      "<leader>gR",
      function() require("util.gh_workflow").run() end,
      desc = "Run GitHub workflow (dispatch)",
    },
    {
      "<leader>gW",
      function() require("util.gh_workflow").watch() end,
      desc = "Watch GitHub workflow run",
    },
  },
}
