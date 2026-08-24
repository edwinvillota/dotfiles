-- SQL tooling overrides for LazyVim's lang.sql extra.

return {
  -- LazyVim hardcodes --dialect=ansi for the formatter. Everything here is
  -- Postgres, and ansi rejects Postgres-only syntax, so switch it. The linter
  -- takes its dialect from ~/.sqlfluff (it is invoked without a flag).
  {
    "stevearc/conform.nvim",
    optional = true,
    opts = function(_, opts)
      opts.formatters = opts.formatters or {}
      opts.formatters.sqlfluff = {
        args = { "format", "--dialect=postgres", "-" },
        -- conform's builtin requires a project-local .sqlfluff before it will
        -- run. The config lives in ~/.sqlfluff, so drop that requirement and
        -- let sqlfluff resolve its own config as it normally does.
        require_cwd = false,
      }
    end,
  },

  -- sqlfluff's rule set is written for versioned SQL in a repo, so on ad-hoc
  -- queries it flags nearly every line (select-target layout, aliasing style,
  -- join order). The rules stay enabled -- `sqlfluff format` needs them to do
  -- anything -- but they are dropped as linter diagnostics: format-on-save
  -- fixes the layout ones automatically, so reporting them is redundant.
  {
    "mfussenegger/nvim-lint",
    optional = true,
    opts = function(_, opts)
      for _, ft in ipairs({ "sql", "mysql", "plsql" }) do
        if opts.linters_by_ft and opts.linters_by_ft[ft] then
          opts.linters_by_ft[ft] = vim.tbl_filter(function(l)
            return l ~= "sqlfluff"
          end, opts.linters_by_ft[ft])
        end
      end
    end,
  },

  -- Buffers opened from lazysql's "open in external editor" (Ctrl-O) are temp
  -- files named lazysql-*.sql with no database attached, so
  -- vim-dadbod-completion has no schema to complete against. lazysql exports
  -- LAZYSQL_CONNECTION_URL while the editor runs -- adopt it as the buffer's
  -- dadbod connection so table/column completion works there.
  {
    "kristijanhusak/vim-dadbod-completion",
    optional = true,
    init = function()
      vim.api.nvim_create_autocmd({ "BufRead", "BufNewFile" }, {
        pattern = "lazysql-*.sql",
        group = vim.api.nvim_create_augroup("lazysql_dadbod", { clear = true }),
        callback = function(args)
          local url = vim.env.LAZYSQL_CONNECTION_URL
          if url and url ~= "" then
            vim.b[args.buf].db = url
          end
        end,
      })
    end,
  },
}
