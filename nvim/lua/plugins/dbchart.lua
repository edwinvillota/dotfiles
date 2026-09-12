-- Plot a dadbod result set without leaving Neovim.
--
-- ~/.psqlrc already emits CSV for dadbod queries (it keys off ON_ERROR_STOP,
-- which dadbod is the only caller to set), which is what lets csvview render
-- the dbout buffer as a grid. That same CSV is exactly what `chart` wants, so
-- plotting a result is just piping the buffer through it.
--
--   <leader>Dc  chart the result buffer (auto-detects the chart type)
--   <leader>DC  chart it, prompting for the type
--
-- Bound under <leader>D, the database namespace this config already uses
-- (Dd/Dv/Dl). NOT <leader>uc/uC: <leader>uC is CsvViewToggle in csv.lua.
--
-- The chart is ANSI-coloured, so it goes to a terminal buffer rather than a
-- normal one -- a plain buffer would show the escape codes as text.

local CHART = vim.fn.expand("~/.config/chart/chart.py")
local VENV = vim.fn.expand("~/.local/share/chart/venv/bin/python")

local function chart_buffer(ctype)
  local buf = vim.api.nvim_get_current_buf()
  if vim.bo[buf].filetype ~= "dbout" and not vim.api.nvim_buf_get_name(buf):match("%.csv$") then
    vim.notify("dbchart: not a dbout or csv buffer", vim.log.levels.WARN)
    return
  end
  if vim.fn.executable(VENV) == 0 then
    vim.notify("dbchart: chart venv missing -- run `chart --setup` in a shell", vim.log.levels.ERROR)
    return
  end

  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  -- dadbod prints the row count as a trailing "(N rows)"; it is not CSV.
  while #lines > 0 and (lines[#lines] == "" or lines[#lines]:match("^%(%d+ rows?%)$")) do
    table.remove(lines)
  end
  if #lines < 2 then
    vim.notify("dbchart: need a header and at least one row", vim.log.levels.WARN)
    return
  end

  local tmp = vim.fn.tempname() .. ".csv"
  vim.fn.writefile(lines, tmp)

  local cmd = { VENV, CHART, "-i", tmp }
  if ctype and ctype ~= "" then
    vim.list_extend(cmd, { "-t", ctype })
  end
  -- size the plot to the split we are about to open
  local width = math.max(40, vim.o.columns - 4)
  local height = math.max(12, math.floor(vim.o.lines * 0.45) - 4)
  vim.list_extend(cmd, { "-W", tostring(width), "-H", tostring(height) })

  vim.cmd("botright new")
  local out = vim.api.nvim_get_current_buf()
  vim.bo[out].buftype = "nofile"
  vim.bo[out].bufhidden = "wipe"
  vim.bo[out].swapfile = false
  vim.api.nvim_buf_set_name(out, "dbchart://" .. (ctype or "auto"))
  vim.cmd("resize " .. (height + 3))

  -- termopen renders the ANSI colour the chart emits
  vim.fn.termopen(cmd, {
    on_exit = function(_, code)
      vim.fn.delete(tmp)
      if code ~= 0 then
        vim.notify("dbchart: chart exited " .. code, vim.log.levels.ERROR)
      end
    end,
  })
  vim.keymap.set("n", "q", "<cmd>close<cr>", { buffer = out, nowait = true })
  vim.cmd("stopinsert")
end

return {
  "tpope/vim-dadbod",
  optional = true,
  keys = {
    {
      "<leader>Dc",
      function() chart_buffer(nil) end,
      desc = "Chart query result",
      ft = { "dbout", "csv" },
    },
    {
      "<leader>DC",
      function()
        vim.ui.select({ "auto", "line", "bar", "barh", "scatter", "hist", "box" },
          { prompt = "chart type" },
          function(choice) if choice then chart_buffer(choice ~= "auto" and choice or nil) end end)
      end,
      desc = "Chart query result (pick type)",
      ft = { "dbout", "csv" },
    },
  },
}
