-- Run the SQL statement under the cursor, so one file can hold many queries.
--
-- Statements are split on `;`, skipping semicolons inside 'strings',
-- "identifiers", $tag$ dollar quotes $tag$, -- line and /* block */ comments
-- (Postgres block comments nest). Leading comments/whitespace are not sent.

local M = {}

-- Returns a list of { start = byte, stop = byte } (1-based, inclusive) covering
-- the code of each statement in `text`, `stop` being its `;` when it has one.
function M.statements(text)
  local stmts = {}
  local n = #text
  local i = 1
  local first, last -- first/last code byte of the statement being scanned

  local function code(pos_start, pos_stop)
    first = first or pos_start
    last = pos_stop
  end

  while i <= n do
    local c = text:sub(i, i)
    local c2 = text:sub(i, i + 1)

    if c2 == "--" then
      local nl = text:find("\n", i + 2, true)
      i = nl and nl + 1 or n + 1
    elseif c2 == "/*" then
      local depth, j = 1, i + 2
      while j <= n and depth > 0 do
        local t = text:sub(j, j + 1)
        if t == "/*" then
          depth, j = depth + 1, j + 2
        elseif t == "*/" then
          depth, j = depth - 1, j + 2
        else
          j = j + 1
        end
      end
      i = j
    elseif c == "'" or c == '"' then
      -- E'...' strings honour backslash escapes; '' / "" doubling is handled
      -- by closing and immediately reopening the quote.
      local escapes = c == "'" and i > 1 and text:sub(i - 1, i - 1):match("[Ee]")
        and (i == 2 or not text:sub(i - 2, i - 2):match("[%w_]"))
      local j = i + 1
      while j <= n do
        local t = text:sub(j, j)
        if escapes and t == "\\" then
          j = j + 2
        elseif t == c then
          break
        else
          j = j + 1
        end
      end
      code(i, math.min(j, n))
      i = j + 1
    elseif c == "$" and text:match("^%$[%a_][%w_]*%$", i) or c2 == "$$" then
      local tag = text:match("^%$[%a_]?[%w_]*%$", i)
      local close = text:find(tag, i + #tag, true)
      local stop = close and close + #tag - 1 or n
      code(i, stop)
      i = stop + 1
    elseif c == ";" then
      code(i, i)
      table.insert(stmts, { start = first, stop = last })
      first, last = nil, nil
      i = i + 1
    elseif c:match("%s") then
      i = i + 1
    else
      code(i, i)
      i = i + 1
    end
  end

  if first then
    table.insert(stmts, { start = first, stop = last })
  end
  return stmts
end

-- The statement whose span (from the previous `;` up to and including its own)
-- contains `offset`. Past the last statement, fall back to the last one.
function M.at(stmts, offset)
  for _, s in ipairs(stmts) do
    if offset <= s.stop then
      return s
    end
  end
  return stmts[#stmts]
end

local function offset_to_pos(line_starts, offset)
  local row = 1
  for r, start in ipairs(line_starts) do
    if start > offset then
      break
    end
    row = r
  end
  return row, offset - line_starts[row] -- (1-based row, 0-based col)
end

function M.run()
  local buf = vim.api.nvim_get_current_buf()
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local line_starts, pos = {}, 1
  for r, l in ipairs(lines) do
    line_starts[r] = pos
    pos = pos + #l + 1
  end
  local text = table.concat(lines, "\n")

  local cursor = vim.api.nvim_win_get_cursor(0)
  local stmt = M.at(M.statements(text), line_starts[cursor[1]] + cursor[2])
  if not stmt then
    vim.notify("No SQL statement under the cursor", vim.log.levels.WARN)
    return
  end
  local sql = text:sub(stmt.start, stmt.stop)

  -- Bind parameters (:name) are vim-dadbod-ui's to fill in, so hand it the
  -- statement as a charwise selection. Its own visual path yanks exactly the
  -- selection before injecting values. Without parameters that path falls
  -- back to a linewise '<,'>DB, which would also run anything sharing the
  -- first/last line, so run the exact text from a file instead.
  local pattern = vim.g.db_ui_bind_param_pattern or [[:\w\+]]
  local has_params = vim.fn.match(sql, [=[\(^\|[[:blank:]]\|[^:]\)\(]=] .. pattern .. [[\)]]) > -1
  if has_params and vim.b[buf].dbui_db_key_name then
    local srow, scol = offset_to_pos(line_starts, stmt.start)
    local erow, ecol = offset_to_pos(line_starts, stmt.stop)
    vim.api.nvim_win_set_cursor(0, { srow, scol })
    vim.cmd("normal! v")
    vim.api.nvim_win_set_cursor(0, { erow, ecol })
    -- feedkeys, not :normal, so its parameter prompts can read real input.
    vim.api.nvim_feedkeys(vim.keycode("<Plug>(DBUI_ExecuteQuery)"), "m", false)
    return
  end

  local file = vim.fn.tempname() .. ".sql"
  vim.fn.writefile(vim.split(sql, "\n", { plain = true }), file)
  vim.cmd("DB < " .. vim.fn.fnameescape(file))
end

return M
