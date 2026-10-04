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
    opts = function(_, opts)
      -- neotest parses test files in a child `nvim -u NONE`, which never sources
      -- nvim-treesitter's plugin/filetypes.lua. Without its filetype->parser map,
      -- "typescriptreact" has no parser there and .tsx/.jsx discovery fails with
      -- `No parser for language "typescriptreact"`. Register the map in the child.
      local subprocess = require("neotest.lib.subprocess")
      local init = subprocess.init
      subprocess.init = function(...)
        init(...)
        if subprocess.enabled() then
          subprocess.request(
            "nvim_exec_lua",
            [[
              vim.treesitter.language.register("tsx", { "typescriptreact", "typescript.tsx" })
              vim.treesitter.language.register("javascript", { "javascriptreact" })
            ]],
            {}
          )
        end
      end

      opts.adapters = vim.tbl_extend("force", opts.adapters or {}, {
        ["neotest-vitest"] = {},
        ["neotest-jest"] = {},
      })
    end,
  },
}
