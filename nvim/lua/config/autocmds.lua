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

-- Stale swap files should not block a file from opening.
--
-- A swap file left behind by a crashed or killed nvim makes the next open
-- raise E325 ATTENTION and wait for an answer. Snacks' explorer and picker run
-- their `:buffer` inside `nvim_exec2`, where that prompt cannot be answered, so
-- the open fails outright with "E5108: ... Vim(buffer):E325: ATTENTION" and the
-- file never appears -- which reads as the filetype's plugin being broken
-- (kulala, on a .http file) rather than as a swap problem.
--
-- Decide instead of asking, but only when it is safe to:
--   * another *live* nvim holds the file -> still prompt, that warning is real
--   * dead process, no unsaved changes    -> delete the swap and open
--   * dead process, unsaved changes       -> open, keep the swap, and say so
local swap_augroup = vim.api.nvim_create_augroup("dotfiles_stale_swap", { clear = true })

vim.api.nvim_create_autocmd("SwapExists", {
  group = swap_augroup,
  callback = function()
    local ok, info = pcall(vim.fn.swapinfo, vim.v.swapname)
    if not ok or type(info) ~= "table" or info.error then
      return -- cannot tell what this is; let nvim ask
    end

    local pid = tonumber(info.pid) or 0
    if pid > 0 then
      local out = vim.fn.system({ "ps", "-p", tostring(pid), "-o", "comm=" })
      if vim.v.shell_error == 0 and out:gsub("%s+", "") ~= "" then
        return -- a running process holds it: prompt, the warning is real
      end
    end

    if info.dirty == 1 then
      vim.v.swapchoice = "e" -- unsaved changes: open, but keep them recoverable
      vim.schedule(function()
        vim.notify(
          ("Stale swap kept for %s -- it has unsaved changes (:recover to see them)"):format(
            vim.fn.fnamemodify(vim.v.swapname, ":t")
          ),
          vim.log.levels.WARN
        )
      end)
    else
      vim.v.swapchoice = "d" -- nothing to lose: delete it and open the file
    end
  end,
})
