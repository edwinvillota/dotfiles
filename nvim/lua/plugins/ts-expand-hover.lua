-- Expandable TypeScript hovers: K opens a float where + / - expand or collapse
-- type aliases one level at a time. Needs TypeScript 5.9+ in the project; vtsls
-- uses the workspace TypeScript (autoUseWorkspaceTsdk), and its bundled 5.7
-- cannot expand, so older projects get the plain hover.
return {
  {
    "nemanjamalesija/ts-expand-hover.nvim",
    -- Pinned to the reviewed commit (no license, single maintainer).
    commit = "629277cc4b9f9179b3b4439d8a7b820c843a4ee7",
    lazy = true,
    -- LazyVim owns K per buffer, so the plugin must not map it globally.
    opts = { keymaps = { hover = false } },
  },
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        ["*"] = {
          keys = {
            -- Replaces LazyVim's K. Falls back to vim.lsp.buf.hover() when vtsls
            -- is not attached, so other languages keep the normal hover.
            {
              "K",
              function() require("ts_expand_hover").hover() end,
              desc = "Hover (expandable TS types)",
            },
          },
        },
      },
    },
  },
}
