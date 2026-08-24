return {
  {
    "nvim-neotest/neotest",
    -- Was pinned to 52fca67 (2025-08-08) because of a Neovim 0.12 incompatibility.
    -- Upstream fixed that in #596 ("Fix tests discovery with latest Neovim 0.12
    -- and Treesitter"), so the pin is removed. Re-pin only if discovery breaks.
    lazy = true,
    dependencies = {
      "marilari88/neotest-vitest",
      "nvim-neotest/neotest-jest",
    },
    opts = {
      adapters = {
        ["neotest-vitest"] = {},
        ["neotest-jest"] = {},
      },
    },
  },
}
