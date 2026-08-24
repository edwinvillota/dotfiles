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
