-- Autocmds are automatically loaded on the VeryLazy event
-- Default autocmds that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/autocmds.lua
--
-- Add any additional autocmds here
-- with `vim.api.nvim_create_autocmd`
--
-- Or remove existing autocmds by their group name (which is prefixed with `lazyvim_` for the defaults)
-- e.g. vim.api.nvim_del_augroup_by_name("lazyvim_wrap_spell")

local function disable_copilot_on_startup()
  -- Check if Copilot is loaded before trying to disable it
  if pcall(require, "copilot") then
    -- It's important to use vim.cmd in a safe way
    -- Use vim.schedule to ensure the command runs after everything is settled
    vim.schedule(function()
      vim.cmd("Copilot disable")
      -- You can add a notification to confirm it worked (optional)
      vim.notify("Copilot disabled on startup.", vim.log.levels.INFO)
    end)
  end
end

-- Create an autocommand group to avoid duplicate autocommands if the file is sourced multiple times
local copilot_disable_augroup = vim.api.nvim_create_augroup("CopilotDisableGroup", { clear = true })

-- Run the disable_copilot_on_startup function on the 'VimEnter' event
-- 'VimEnter' is triggered after all startup files are sourced and initializations are done.
vim.api.nvim_create_autocmd("VimEnter", {
  group = copilot_disable_augroup,
  callback = disable_copilot_on_startup,
})

-- You might also consider 'BufReadPost' or 'VeryLazy' if VimEnter is too early for some reason,
-- but VimEnter is generally reliable for post-startup commands.
-- Example for BufReadPost (less robust for global disable):
-- vim.api.nvim_create_autocmd("BufReadPost", {
--   group = copilot_disable_augroup,
--   pattern = "*", -- Apply to all files
--   callback = disable_copilot_on_startup,
-- })
