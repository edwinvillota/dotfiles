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
    keymaps = {
      -- Excel-like field navigation
      textobject_field_inner = { "if", mode = { "o", "x" } },
      textobject_field_outer = { "af", mode = { "o", "x" } },
      jump_next_field_end = { "<Tab>", mode = { "n", "v" } },
      jump_prev_field_end = { "<S-Tab>", mode = { "n", "v" } },
      jump_next_row = { "<Enter>", mode = { "n", "v" } },
      jump_prev_row = { "<S-Enter>", mode = { "n", "v" } },
    },
  },
  config = function(_, opts)
    require("csvview").setup(opts)

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
