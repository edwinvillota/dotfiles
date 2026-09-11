return {
  "hat0uma/csvview.nvim",
  ft = { "csv", "tsv" },
  keys = {
    { "<leader>uC", "<cmd>CsvViewToggle<cr>", desc = "Toggle CSV view" },
  },
  opts = {
    parser = {
      comments = { "#", "//" },
      -- The default pins csv -> "," via `ft`, which short-circuits detection and
      -- breaks semicolon files. Leave csv out so `fallbacks` auto-detects instead.
      delimiter = {
        ft = { tsv = "\t" },
        fallbacks = { ",", ";", "\t", "|", ":" },
      },
    },
    view = {
      display_mode = "border",
      header_lnum = 1,
      sticky_header = { enabled = true },
    },
    -- Excel-like field navigation lives in `maps` below, not here: csvview's own
    -- keymap registration binds with `buffer = true`, i.e. whatever buffer is
    -- *current when it runs* -- and it runs from the async metrics callback
    -- (init.lua:160, parser.lua yields with vim.schedule). For a file you just
    -- opened that is still the right buffer, but for dadbod's `dbout` results
    -- focus is back in the query window by then, so <Tab> got mapped there and
    -- the results buffer got nothing. Pressing it then threw
    -- "CsvView is not enabled for this buffer".
    keymaps = {},
  },
  config = function(_, opts)
    require("csvview").setup(opts)

    -- Same actions csvview's keymap presets would install, bound to the buffer
    -- that actually attached (the CsvViewAttach event carries its bufnr).
    local function jump(fn)
      return function()
        for _ = 1, vim.v.count1 do
          require("csvview.jump")[fn]()
        end
      end
    end

    local function row(dir)
      return function()
        require("csvview.jump").field(0, { pos = { dir * vim.v.count1, 0 }, anchor = "end" })
      end
    end

    local function textobject(include_delimiter)
      return function()
        require("csvview.textobject").field(0, { include_delimiter = include_delimiter })
      end
    end

    local maps = {
      { { "o", "x" }, "if", textobject(false), "Select field" },
      { { "o", "x" }, "af", textobject(true), "Select field with delimiter" },
      { { "n", "v" }, "<Tab>", jump("next_field_end"), "Next field" },
      { { "n", "v" }, "<S-Tab>", jump("prev_field_end"), "Previous field" },
      { { "n", "v" }, "<Enter>", row(1), "Next row" },
      { { "n", "v" }, "<S-Enter>", row(-1), "Previous row" },
    }

    vim.api.nvim_create_autocmd("User", {
      pattern = "CsvViewAttach",
      group = vim.api.nvim_create_augroup("csvview_keymaps", { clear = true }),
      callback = function(ev)
        local buf = ev.data
        if type(buf) ~= "number" or not vim.api.nvim_buf_is_valid(buf) then
          return
        end
        for _, m in ipairs(maps) do
          vim.keymap.set(m[1], m[2], m[3], { buffer = buf, silent = true, desc = "[csvview] " .. m[4] })
        end
      end,
    })

    vim.api.nvim_create_autocmd("User", {
      pattern = "CsvViewDetach",
      group = "csvview_keymaps",
      callback = function(ev)
        local buf = ev.data
        if type(buf) ~= "number" or not vim.api.nvim_buf_is_valid(buf) then
          return
        end
        for _, m in ipairs(maps) do
          pcall(vim.keymap.del, m[1], m[2], { buffer = buf })
        end
      end,
    })

    -- csvview colours columns through CsvViewCol0..8, which it links to the
    -- built-in csvCol0..8 (csvview/config.lua). Neovim only defines those when
    -- runtime/syntax/csv.vim runs, i.e. for filetype=csv -- so in any other
    -- buffer csvview is enabled on (the dbout results buffer from dadbod, for
    -- one) the links dangle and every column renders in plain text.
    --
    -- Point them at standard syntax groups instead. Those are defined by the
    -- colorscheme, so the columns follow whatever theme is active with nothing
    -- to regenerate when it changes.
    -- Only these six standard groups reliably carry distinct colours; the rest
    -- (PreProc, Keyword, Special, Number, Boolean...) link back onto them in
    -- most colorschemes, which would give neighbouring columns the same hue.
    -- Nine slots, cycled so no two adjacent columns ever match.
    local col_links = {
      "Function", "String", "Type", "Identifier", "Statement", "Constant",
      "Function", "String", "Type",
    }
    local function link_csv_cols()
      for i, group in ipairs(col_links) do
        vim.api.nvim_set_hl(0, ("CsvViewCol%d"):format(i - 1), { link = group })
      end
    end

    link_csv_cols()
    -- Re-link after a colorscheme swap: `:colorscheme` clears highlight groups.
    vim.api.nvim_create_autocmd("ColorScheme", {
      group = vim.api.nvim_create_augroup("csvview_theme", { clear = true }),
      callback = link_csv_cols,
    })

    -- csvview ships disabled; turn it on automatically for csv/tsv buffers.
    vim.api.nvim_create_autocmd("FileType", {
      group = vim.api.nvim_create_augroup("csvview_auto", { clear = true }),
      pattern = { "csv", "tsv" },
      callback = function(ev)
        if not require("csvview").is_enabled(ev.buf) then
          require("csvview").enable(ev.buf)
        end
      end,
    })

    -- The autocmd above misses the buffer that lazy-loaded this plugin.
    local ft = vim.bo.filetype
    if ft == "csv" or ft == "tsv" then
      require("csvview").enable(0)
    end
  end,
}
