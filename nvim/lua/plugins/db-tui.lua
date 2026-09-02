-- lazysql (TUI database client) alongside vim-dadbod.
-- dadbod stays for authoring/running queries; these are for browsing,
-- navigating result sets and quick single-cell edits, which dadbod's plain
-- `dbout` buffer cannot do.
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

-- Each client takes the connection URL differently, and lazysql has a local
-- patched build, so resolve both per client instead of assuming `<bin> <url>`.
local clients = {
  lazysql = {
    brew = "lazysql",
    bin = function()
      -- Prefer the locally patched build (autocomplete fixes) when present.
      local patched = vim.fn.expand("~/.local/bin/lazysql-patched")
      return vim.fn.executable(patched) == 1 and patched or "lazysql"
    end,
    args = function(url)
      return { url }
    end,
  },
}

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

local function open(name, url)
  local client = clients[name]
  local bin = client.bin()
  if vim.fn.executable(bin) == 0 then
    vim.notify(name .. " is not installed (brew install " .. client.brew .. ")", vim.log.levels.ERROR)
    return
  end
  local cmd = { bin }
  vim.list_extend(cmd, client.args(url))
  Snacks.terminal.open(cmd, {
    interactive = true,
    win = {
      style = "terminal",
      width = 0.95,
      height = 0.95,
      border = "rounded",
      title = " " .. name .. " ",
      title_pos = "center",
    },
  })
end

-- Pipe the current dbout buffer (CSV, per ~/.psqlrc) into VisiData.
local function visidata(buf)
  if vim.fn.executable("vd") == 0 then
    vim.notify("visidata is not installed (brew install visidata)", vim.log.levels.ERROR)
    return
  end
  local tmp = vim.fn.tempname() .. ".csv"
  vim.fn.writefile(vim.api.nvim_buf_get_lines(buf, 0, -1, false), tmp)
  Snacks.terminal.open({ "vd", "-f", "csv", tmp }, {
    interactive = true,
    win = {
      style = "terminal",
      width = 0.95,
      height = 0.95,
      border = "rounded",
      title = " visidata ",
      title_pos = "center",
    },
  })
end

local function launch(name)
  return function()
    local conns = connections()
    if #conns == 0 then
      vim.ui.input({ prompt = "Postgres connection string: " }, function(url)
        if url and url ~= "" then
          open(name, url)
        end
      end)
      return
    end
    vim.ui.select(conns, {
      prompt = name .. ": select database",
      format_item = function(item)
        return item.name
      end,
    }, function(choice)
      if choice then
        open(name, choice.url)
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
    init = function()
      -- Make the results buffer usable. ~/.psqlrc emits CSV for dadbod (and
      -- only for dadbod), which is what lets the two viewers below treat these
      -- results as real tabular data instead of pre-formatted text.
      vim.api.nvim_create_autocmd("FileType", {
        pattern = "dbout",
        group = vim.api.nvim_create_augroup("dbout_view", { clear = true }),
        callback = function(ev)
          -- dadbod sizes this split to the output, which for anything wider
          -- than a couple of columns is too short to read. <C-w>_ still zooms.
          vim.api.nvim_win_set_height(0, math.floor(vim.o.lines * 0.6))
          vim.wo.wrap = false
          vim.wo.cursorline = true
          vim.wo.number = false
          vim.wo.signcolumn = "no"
          vim.wo.sidescrolloff = 8

          -- Inline grid: aligned columns, a header that stays put while
          -- scrolling, and <Tab>/<S-Tab> to move between fields. All of that
          -- comes from the csvview config in lua/plugins/csv.lua.
          local ok, csvview = pcall(require, "csvview")
          if ok and not csvview.is_enabled(ev.buf) then
            pcall(csvview.enable, ev.buf, {
              parser = { delimiter = "," },
              view = {
                display_mode = "border",
                header_lnum = 1,
                sticky_header = { enabled = true },
              },
            })
          end

          -- Snacks zoom (<leader>wm) wraps this buffer in a zen window and
          -- keeps owning it: teardown of the zoom indicator and the saved
          -- toggle states hangs off that window's WinClosed (snacks/zen.lua
          -- :195), and a BufWinEnter handler writes the zen buffer back into
          -- the parent window (:208). LazyVim's `q` for dbout closes the
          -- window and force-deletes the buffer, which destroys what Snacks is
          -- built around before that teardown runs -- leaving an orphaned
          -- float that cannot be focused, because the indicator style sets
          -- focusable = false (:78).
          --
          -- So while zoomed, `q` only leaves zoom. The results stay open and a
          -- second `q` closes them, with Snacks no longer holding the buffer.
          local function zen_win()
            local ok, zen = pcall(function()
              return Snacks.zen
            end)
            if ok and zen and zen.win and zen.win:valid() then
              return zen.win
            end
          end

          local function close_results()
            local zen = zen_win()
            if zen then
              pcall(function()
                zen:close()
              end)
              return
            end
            pcall(vim.cmd, "close")
            pcall(vim.api.nvim_buf_delete, ev.buf, { force = true })
          end

          -- Double schedule: LazyVim maps `q` for dbout from its own FileType
          -- autocmd via vim.schedule (config/autocmds.lua:80). Whichever
          -- autocmd runs last would otherwise win; deferring one extra tick
          -- puts this mapping after theirs regardless of registration order.
          vim.schedule(vim.schedule_wrap(function()
            if vim.api.nvim_buf_is_valid(ev.buf) then
              vim.keymap.set("n", "q", close_results, {
                buffer = ev.buf,
                silent = true,
                desc = "Leave zoom, else quit results",
              })
            end
          end))

          -- Net for paths that drop the buffer without going through `q`
          -- (:bd, or the next query replacing it) while zoom is still active.
          vim.api.nvim_create_autocmd({ "BufUnload", "BufWipeout" }, {
            buffer = ev.buf,
            callback = function()
              local zen = zen_win()
              if zen then
                vim.schedule(function()
                  pcall(function()
                    zen:close()
                  end)
                end)
              end
            end,
          })

          -- For anything csvview is too small for -- sorting, filtering,
          -- frequency tables, wide result sets -- hand the same CSV to
          -- VisiData. It is already a tracked tool (see ~/.visidatarc).
          vim.keymap.set("n", "<leader>Dv", function()
            visidata(ev.buf)
          end, { buffer = ev.buf, desc = "Results in VisiData" })
        end,
      })
    end,
  },
  {
    "folke/snacks.nvim",
    keys = {
      { "<leader>Dl", launch("lazysql"), desc = "LazySQL (TUI)" },
    },
  },
}
