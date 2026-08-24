-- lazysql (TUI database client) alongside vim-dadbod.
-- dadbod stays for quick inline queries; these are for browsing/exploring.
--
-- Connections are read from `vim.g.dbs` (the same table vim-dadbod-ui uses),
-- so there is a single source of truth. Define it in lua/config/options.lua:
--
--   vim.g.dbs = {
--     dev  = "postgres://user:pass@localhost:5432/dev",
--     prod = "postgres://user:pass@host:5432/prod",
--   }
--
-- If `vim.g.dbs` is empty you get prompted for a connection string.

local function connections()
  local dbs = vim.g.dbs or {}
  local list = {}
  -- vim.g.dbs supports both a map and a list of {name=, url=}
  if vim.islist(dbs) then
    for _, e in ipairs(dbs) do
      table.insert(list, { name = e.name, url = e.url })
    end
  else
    for name, url in pairs(dbs) do
      table.insert(list, { name = name, url = url })
    end
  end
  table.sort(list, function(a, b)
    return a.name < b.name
  end)
  return list
end

local function open(client, url)
  if vim.fn.executable(client) == 0 then
    vim.notify(client .. " is not installed (brew install " .. client .. ")", vim.log.levels.ERROR)
    return
  end
  -- Prefer the locally patched build (autocomplete fixes) when present.
  local bin = vim.fn.expand("~/.local/bin/lazysql-patched")
  if vim.fn.executable(bin) == 0 then
    bin = client
  end
  Snacks.terminal.open({ bin, url }, {
    interactive = true,
    win = {
      style = "terminal",
      width = 0.95,
      height = 0.95,
      border = "rounded",
      title = " " .. client .. " ",
      title_pos = "center",
    },
  })
end

local function launch(client)
  return function()
    local conns = connections()
    if #conns == 0 then
      vim.ui.input({ prompt = "Postgres connection string: " }, function(url)
        if url and url ~= "" then
          open(client, url)
        end
      end)
      return
    end
    vim.ui.select(conns, {
      prompt = client .. ": select database",
      format_item = function(item)
        return item.name
      end,
    }, function(choice)
      if choice then
        open(client, choice.url)
      end
    end)
  end
end

return {
  {
    "folke/which-key.nvim",
    opts = {
      spec = {
        { "<leader>D", group = "database" },
      },
    },
  },
  -- Free up <leader>D so it can be a group; DBUI moves to <leader>Dd.
  {
    "kristijanhusak/vim-dadbod-ui",
    keys = {
      { "<leader>D", false },
      { "<leader>Dd", "<cmd>DBUIToggle<cr>", desc = "Toggle DBUI (dadbod)" },
    },
  },
  {
    "folke/snacks.nvim",
    keys = {
      { "<leader>Dl", launch("lazysql"), desc = "LazySQL (TUI)" },
    },
  },
}
