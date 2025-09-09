-- Monokay pro highlight groups
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusUntracked", { fg = "#727072" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusModified", { fg = "#A9DC76" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusAdded", { fg = "#FFD866" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusDeleted", { fg = "#FF6188" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusRenamed", { fg = "#78DCE8" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusStaged", { fg = "#A9DC76" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusIgnored", { fg = "#727072" })
-- vim.api.nvim_set_hl(0, "SnacksPickerGitStatusHidden", { fg = "#727072" })
-- vim.api.nvim_set_hl(0, "SnacksPickerPathHidden", { fg = "#727072" })
--

-- neovim-yua highlight groups
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusUntracked", { fg = "#d4d4d4" }) -- yua.gray
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusModified", { fg = "#a3b7cc" }) -- yua.blue
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusAdded", { fg = "#b7c7a3" }) -- yua.green
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusDeleted", { fg = "#cc7e7e" }) -- yua.red
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusRenamed", { fg = "#a3ccc7" }) -- yua.cyan
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusStaged", { fg = "#b7c7a3" }) -- yua.green
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusIgnored", { fg = "#e0e0ff" }) -- brighter gray for ignored
vim.api.nvim_set_hl(0, "SnacksPickerGitStatusHidden", { fg = "#a3c7ff" }) -- brighter cyan for hidden
vim.api.nvim_set_hl(0, "SnacksPickerPathHidden", { fg = "#a3c7ff" }) -- brighter cyan for hidden path

vim.api.nvim_set_hl(0, "SnacksPickerFile", { fg = "#a3b7cc" }) -- yua.blue
vim.api.nvim_set_hl(0, "SnacksPickerFileHidden", { fg = "#7e7eae" }) -- yua.gray
vim.api.nvim_set_hl(0, "SnacksPickerFolder", { fg = "#a3ccc7" }) -- yua.cyan
vim.api.nvim_set_hl(0, "SnacksPickerFolderHidden", { fg = "#7e7eae" }) -- yua.gray
vim.api.nvim_set_hl(0, "SnacksPickerPathIgnored", { fg = "#5c6370" })

vim.api.nvim_set_hl(0, "SnacksPickerMatch", { fg = "#cc7e7e", bold = true }) -- yua.red, for search matches
vim.api.nvim_set_hl(0, "SnacksPickerSelection", { fg = "#b7c7a3", bold = true }) -- yua.green, for selection
vim.api.nvim_set_hl(0, "SnacksPickerPrompt", { fg = "#a3b7cc", italic = true }) -- yua.blue, for prompt
