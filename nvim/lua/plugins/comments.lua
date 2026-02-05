return {
  {
    "bennypowers/template-literal-comments.nvim",
    ft = {
      "javascript",
      "typescript",
      "javascriptreact",
      "typescriptreact",
    },
    opts = true,
  },

  -- Ensure the GLSL parser is installed for Treesitter
  {
    "nvim-treesitter/nvim-treesitter",
    opts = function(_, opts)
      if type(opts.ensure_installed) == "table" then
        vim.list_extend(opts.ensure_installed, { "glsl" })
      end
    end,
  },
}
