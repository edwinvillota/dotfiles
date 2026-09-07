-- SQL tooling overrides for LazyVim's lang.sql extra.

return {
  -- LazyVim runs `sqlfluff format --dialect=ansi`. Two changes:
  --   * `fix` instead of `format`: format only applies layout/capitalisation,
  --     while fix also applies the other auto-fixable rules (explicit `as`,
  --     dropping unnecessary quotes, join-condition order, ...). Style policy
  --     lives in ~/.sqlfluff. fix exits 1 when unfixable violations remain
  --     but still writes the fixed SQL to stdout, so treat 1 as success.
  --   * postgres dialect: ansi rejects Postgres-only syntax.
  {
    "stevearc/conform.nvim",
    optional = true,
    opts = function(_, opts)
      opts.formatters = opts.formatters or {}
      opts.formatters.sqlfluff = {
        args = { "fix", "--dialect=postgres", "-" },
        exit_codes = { 0, 1 },
        -- conform's builtin requires a project-local .sqlfluff before it will
        -- run. The config lives in ~/.sqlfluff, so drop that requirement and
        -- let sqlfluff resolve its own config as it normally does.
        require_cwd = false,
      }
    end,
  },

  -- Format-on-save runs `sqlfluff fix`, so any violation sqlfluff can fix
  -- itself will be gone by the time the file is written. Reporting those
  -- (layout, capitalisation, aliasing style, ...) between edits is noise.
  -- sqlfluff's JSON output carries a `fixes` list per violation; keep only the
  -- ones it is empty for -- the problems that need a human (duplicate output
  -- names, ambiguous references, parse errors, ...).
  {
    "mfussenegger/nvim-lint",
    optional = true,
    opts = function(_, opts)
      local original = require("lint.linters.sqlfluff").parser
      opts.linters = opts.linters or {}
      opts.linters.sqlfluff = {
        parser = function(output, bufnr)
          local ok, decoded = pcall(vim.json.decode, output)
          if ok and type(decoded) == "table" then
            for _, file in ipairs(decoded) do
              file.violations = vim.tbl_filter(function(v)
                return v.fixes == nil or #v.fixes == 0
              end, file.violations or {})
            end
            output = vim.json.encode(decoded)
          end
          return original(output, bufnr)
        end,
      }
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

      -- The plugin fetches tables/columns once per connection and keeps them
      -- for the whole session, so a table created with :DB is invisible to
      -- completion until :DBCompletionClearCache (or a restart). dadbod fires
      -- `User <outfile>/DBExecutePost` when a query finishes; if the query
      -- that ran was DDL, drop the cache and re-fetch for every open SQL
      -- buffer. Hooking the results buffer instead would be too early: dadbod
      -- opens it before the query has actually executed.
      vim.api.nvim_create_autocmd("User", {
        pattern = "*/DBExecutePost",
        group = vim.api.nvim_create_augroup("dadbod_completion_refresh", { clear = true }),
        callback = function(args)
          local outfile = args.match:gsub("/DBExecutePost$", "")
          local outbuf = vim.fn.bufnr(outfile)
          local input = outbuf > 0 and vim.b[outbuf].db_input or nil
          if not input or vim.fn.filereadable(input) == 0 then
            return
          end
          local sql = table.concat(vim.fn.readfile(input), "\n"):lower()
          local ddl = false
          for _, kw in ipairs({ "create", "drop", "alter", "rename" }) do
            if sql:find("%f[%w]" .. kw .. "%f[%W]") then
              ddl = true
              break
            end
          end
          if not ddl then
            return
          end
          if vim.fn.exists("*vim_dadbod_completion#clear_cache") == 0 then
            return
          end
          vim.cmd("silent call vim_dadbod_completion#clear_cache()")
          for _, buf in ipairs(vim.api.nvim_list_bufs()) do
            if vim.api.nvim_buf_is_loaded(buf) then
              local ft = vim.bo[buf].filetype
              if ft == "sql" or ft == "mysql" or ft == "plsql" then
                pcall(vim.fn["vim_dadbod_completion#fetch"], buf)
              end
            end
          end
        end,
      })
    end,

    -- The plugin quotes a completed identifier when it is camelCase, has
    -- whitespace, or is in its reserved-word list. That list is a union of
    -- every dialect it supports (~1400 words), so on Postgres it quotes
    -- ordinary columns like name, status, type or value that Postgres never
    -- reserves. Postgres only needs quotes for identifiers that would not
    -- survive its lowercase folding (any uppercase or non [a-z0-9_] char) and
    -- for its own reserved words (pg_get_keywords() catcode R/T, PG 16).
    -- `schemas#get` returns the live dict, so the predicate is swapped in
    -- place; `postgres` and `postgresql` share that dict. The predicate has to
    -- be Vimscript because the plugin calls it as a dict method.
    config = function()
      vim.cmd([[
        let s:pg_reserved = {}
        for s:kw in split('ALL ANALYSE ANALYZE AND ANY ARRAY AS ASC ASYMMETRIC
              \ AUTHORIZATION BINARY BOTH CASE CAST CHECK COLLATE COLLATION COLUMN
              \ CONCURRENTLY CONSTRAINT CREATE CROSS CURRENT_CATALOG CURRENT_DATE
              \ CURRENT_ROLE CURRENT_SCHEMA CURRENT_TIME CURRENT_TIMESTAMP
              \ CURRENT_USER DEFAULT DEFERRABLE DESC DISTINCT DO ELSE END EXCEPT
              \ FALSE FETCH FOR FOREIGN FREEZE FROM FULL GRANT GROUP HAVING ILIKE
              \ IN INITIALLY INNER INTERSECT INTO IS ISNULL JOIN LATERAL LEADING
              \ LEFT LIKE LIMIT LOCALTIME LOCALTIMESTAMP NATURAL NOT NOTNULL NULL
              \ OFFSET ON ONLY OR ORDER OUTER OVERLAPS PLACING PRIMARY REFERENCES
              \ RETURNING RIGHT SELECT SESSION_USER SIMILAR SOME SYMMETRIC
              \ SYSTEM_USER TABLE TABLESAMPLE THEN TO TRAILING TRUE UNION UNIQUE
              \ USER USING VARIADIC VERBOSE WHEN WHERE WINDOW WITH')
          let s:pg_reserved[s:kw] = 1
        endfor

        function! PgShouldQuote(val) abort
          let val = trim(a:val)
          if empty(val)
            return 0
          endif
          return val !~# '^[a-z_][a-z0-9_]*$' || has_key(s:pg_reserved, toupper(val))
        endfunction

        let s:pg = vim_dadbod_completion#schemas#get('postgres')
        let s:pg.should_quote = function('PgShouldQuote')
      ]])
    end,
  },
}
