return {
  "neovim/nvim-lspconfig",
  opts = {
    servers = {
      vtsls = {
        settings = {
          -- "shortest" uses tsconfig path aliases (@/...) when they beat the
          -- relative path, and keeps ./ for nearby files.
          -- "minimal" drops the extension even when the project sets
          -- allowImportingTsExtensions ("auto" can switch to "./Button.tsx").
          javascript = {
            preferences = {
              importModuleSpecifier = "shortest",
              importModuleSpecifierEnding = "minimal",
            },
          },
          typescript = {
            preferences = {
              importModuleSpecifier = "shortest",
              importModuleSpecifierEnding = "minimal",
            },
          },
        },
      },
      -- Grammar/spelling checker (English only). Words added through its code
      -- action land in spell/harper.txt, so they sync with the dotfiles.
      -- isolateEnglish is left off: it treats a sentence with a typo as
      -- non-English and skips it. Mute Spanish buffers with <leader>uH instead.
      harper_ls = {
        settings = {
          ["harper-ls"] = {
            userDictPath = vim.fn.stdpath("config") .. "/spell/harper.txt",
            diagnosticSeverity = "hint",
            linters = {
              -- Noisy in code comments and commit bodies.
              SentenceCapitalization = false,
              LongSentences = false,
              ExpandConfiguration = false,
            },
            markdown = {
              IgnoreLinkTitle = true,
            },
          },
        },
      },
    },
  },
}
