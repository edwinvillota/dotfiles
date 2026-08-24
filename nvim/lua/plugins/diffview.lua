return {
  -- Move LazyVim's snacks `git_diff` (hunks) picker off <leader>gd so diffview can own it.
  -- New key: <leader>gc ("git changes" – list of hunks across the repo).
  {
    "folke/snacks.nvim",
    keys = {
      { "<leader>gd", false }, -- disable LazyVim's default
      {
        "<leader>gc",
        function()
          Snacks.picker.git_diff()
        end,
        desc = "Git Diff (hunks)",
      },
    },
  },

  {
    "sindrets/diffview.nvim",
    dependencies = { "nvim-lua/plenary.nvim" },
    cmd = {
      "DiffviewOpen",
      "DiffviewClose",
      "DiffviewToggleFiles",
      "DiffviewFocusFiles",
      "DiffviewFileHistory",
    },
    opts = {
      enhanced_diff_hl = true,
      view = {
        merge_tool = {
          layout = "diff3_mixed", -- 3-way layout for conflicts: OURS | BASE+RESULT | THEIRS
        },
      },
      keymaps = {
        view = {
          { "n", "q", "<cmd>DiffviewClose<cr>", { desc = "Close Diffview" } },
        },
        file_panel = {
          { "n", "q", "<cmd>DiffviewClose<cr>", { desc = "Close Diffview" } },
        },
        file_history_panel = {
          { "n", "q", "<cmd>DiffviewClose<cr>", { desc = "Close Diffview" } },
        },
      },
    },
    keys = {
      { "<leader>gd", "<cmd>DiffviewOpen<cr>", desc = "Diffview: unstaged changes" },
      { "<leader>gD", "<cmd>DiffviewOpen --staged<cr>", desc = "Diffview: staged changes" },
      { "<leader>gm", "<cmd>DiffviewOpen<cr>", desc = "Diffview: resolve merge conflicts" },
      { "<leader>gH", "<cmd>DiffviewFileHistory %<cr>", desc = "Diffview: file history" },
      { "<leader>gA", "<cmd>DiffviewFileHistory<cr>", desc = "Diffview: repo history" },
    },
  },
}
