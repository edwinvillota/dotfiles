-- Options are automatically loaded before lazy.nvim startup
-- Default options that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/options.lua
-- Add any additional options here


vim.opt.autoindent = true -- Maintain indent of current line
vim.opt.smartindent = true -- Smart autoindenting for C-like programs
vim.opt.expandtab = true -- Use spaces instead of tabs
vim.opt.shiftwidth = 2 -- Size of an indent
vim.opt.tabstop = 2 -- Number of spaces that a <Tab> counts for

-- Database connections for vim-dadbod-ui and the TUI clients
-- (lazysql -- see lua/plugins/db-tui.lua).
--
-- Credentials never live in this repo: any environment variable named
-- NVIM_DB_<NAME> is picked up automatically as a connection called <name>.
-- Export them from your shell profile, e.g.:
--
--   export NVIM_DB_DEV="postgres://user:pass@localhost:5432/dev"
--   export NVIM_DB_PROD="postgres://user:pass@host:5432/prod"
--
-- ...gives you connections "dev" and "prod". Adding a database is one
-- export -- no edit here.
do
  local dbs = {}
  for key, url in pairs(vim.fn.environ()) do
    local name = key:match("^NVIM_DB_(.+)$")
    if name and url ~= "" then
      dbs[name:lower()] = url
    end
  end
  vim.g.dbs = dbs
end
