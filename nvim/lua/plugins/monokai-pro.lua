return {
  "loctvl842/monokai-pro.nvim",
  lazy = false, -- load it during startup
  priority = 1000, -- ensure it loads before other plugins
  config = function()
    require("monokai-pro").setup({
      transparent_background = false, -- optional
      terminal_colors = true, -- optional
      devicons = true, -- optional
      styles = {
        comment = { italic = true },
        keyword = { italic = true }, -- change to false to disable italics
        type = { italic = false },
        storageclass = { italic = false },
        structure = { italic = false },
        parameter = { italic = false },
        annotation = { italic = false },
        tag_attribute = { italic = false },
      },
      filter = "octagon", -- <<< this enables `monokai-octagon`
      inc_search = "background", -- or "none"
      background_clear = {},
    })
    vim.cmd("colorscheme monokai-pro")
  end,
}
