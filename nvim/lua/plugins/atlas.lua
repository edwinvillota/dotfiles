-- atlas.nvim: browse and review pull requests from inside Neovim.
-- GitHub goes through the `gh` CLI, so it reuses the existing `gh auth login`.
-- <leader>ga -> PR dashboard, <leader>gr -> pick a PR in this repo and review it.
return {
  "emrearmagan/atlas.nvim",
  cmd = "Atlas",
  dependencies = {
    "MeanderingProgrammer/render-markdown.nvim",
  },
  keys = {
    { "<leader>ga", "<cmd>Atlas pulls github<cr>", desc = "Atlas: pull requests" },
    { "<leader>gr", "<cmd>Atlas review<cr>", desc = "Atlas: review pull request" },
  },
  ---@type AtlasConfig
  opts = {
    ui = {
      picker = "snacks",
    },
    -- Providers are opt-in: without this entry Atlas reports GitHub as not configured.
    providers = {
      github = {},
    },
    pulls = {
      github = {
        views = {
          { name = "Review requested", key = "1", layout = "compact", search = "is:open review-requested:@me" },
          { name = "Mine", key = "2", layout = "plain", search = "author:@me sort:updated-desc" },
          -- Scoped to the repo of the current working directory.
          { name = "This repo", key = "3", layout = "plain", current_repo = true, search = "is:pr sort:updated-desc" },
        },
      },
      repo_config = {
        -- Where Atlas finds local checkouts for diffs and checkout.
        paths = {
          ["edwinvillota/*"] = "~/Documents/dev/*",
        },
      },
    },
  },
}
