-- Cell-aware actions for dadbod result buffers (dbout) holding CSV.
--
-- ~/.psqlrc makes psql emit CSV for dadbod so csvview can draw a grid, but
-- vim-dadbod-ui's own dbout actions locate cells through psql's aligned
-- output (the `----+----` rule under the header), so they break on CSV.
-- These replace them:
--   * foreign_key(): open the row a foreign key value points at, or the
--                    rows pointing at a referenced key            (gd)
--   * jump()/close_jump(): move through, and close, the results those
--                    jumps stack up in the window     ([r, ]r, q)
--   * select_cell(): `ic` text object for the value under the cursor (vic/yic)
--   * yank_header(): yank the column names                       (yh)
--
-- The buffer is parsed here rather than through csvview's internals, so this
-- keeps working with the grid toggled off.

local M = {}

-- Parse RFC 4180 CSV as psql writes it: `,` separated, `"` quoted, `""` for a
-- literal quote, newlines allowed inside quotes. Returns records, each a list
-- of fields { srow, scol, erow, ecol, quoted, value } where rows are 1-based,
-- cols 0-based byte offsets, and the end is exclusive.
function M.parse(lines)
  local records, record = {}, {}
  local row, col = 1, 0
  local nrows = #lines

  local function at(r, c)
    local l = lines[r]
    if not l then
      return nil
    end
    if c < #l then
      return l:sub(c + 1, c + 1)
    end
    return r < nrows and "\n" or nil
  end
  local function advance(r, c)
    if c < #lines[r] then
      return r, c + 1
    end
    return r + 1, 0
  end

  while row <= nrows do
    local srow, scol = row, col
    local ch = at(row, col)
    local field
    if ch == '"' then
      local parts, seg_r, seg_c = {}, row, col + 1
      row, col = advance(row, col)
      while true do
        local c = at(row, col)
        if c == nil then
          break
        elseif c == '"' then
          if at(advance(row, col)) == '"' then
            row, col = advance(row, col)
            row, col = advance(row, col)
            table.insert(parts, '"')
          else
            row, col = advance(row, col)
            break
          end
        else
          table.insert(parts, c)
          row, col = advance(row, col)
        end
      end
      field = { srow = srow, scol = scol, erow = row, ecol = col, quoted = true, value = table.concat(parts) }
      -- skip anything stray between the closing quote and the delimiter
      while at(row, col) and at(row, col) ~= "," and at(row, col) ~= "\n" do
        row, col = advance(row, col)
      end
      field.erow, field.ecol = row, col
    else
      local l = lines[row]
      local stop = l:find(",", col + 1, true)
      local ecol = stop and stop - 1 or #l
      field = { srow = row, scol = col, erow = row, ecol = ecol, quoted = false, value = l:sub(col + 1, ecol) }
      col = ecol
    end
    table.insert(record, field)

    local sep = at(row, col)
    if sep == "," then
      row, col = advance(row, col)
    else
      table.insert(records, record)
      record = {}
      if row == nrows and col >= #lines[row] then
        break
      end
      row, col = advance(row, col)
    end
  end
  if #record > 0 then
    table.insert(records, record)
  end
  return records
end

-- (record index, field index, field) under a 1-based row / 0-based col.
local function field_at(records, row, col)
  for ri, record in ipairs(records) do
    local last = record[#record]
    if row <= last.erow then
      for fi, f in ipairs(record) do
        -- a field owns its trailing delimiter, so the cursor on `,` counts
        local before_end = row < f.erow or (row == f.erow and col <= f.ecol)
        local after_start = row > f.srow or (row == f.srow and col >= f.scol)
        if after_start and before_end then
          return ri, fi, f
        end
      end
      return ri, #record, last
    end
  end
end

local function under_cursor(buf)
  local records = M.parse(vim.api.nvim_buf_get_lines(buf, 0, -1, false))
  local cursor = vim.api.nvim_win_get_cursor(0)
  local ri, fi, f = field_at(records, cursor[1], cursor[2])
  return records, ri, fi, f
end

-- Region of the value under the cursor, in mini.ai's format: 1-based lines
-- and columns, both ends inclusive. Quotes are left out. nil when empty.
function M.cell_region()
  local _, _, _, f = under_cursor(0)
  if not f then
    return nil
  end
  local srow, scol, erow, ecol = f.srow, f.scol, f.erow, f.ecol -- 0-based cols, end exclusive
  if f.quoted then
    scol, ecol = scol + 1, ecol - 1
  end
  if erow == srow and ecol <= scol then
    return nil
  end
  if ecol == 0 then
    -- ends right after a newline inside the value: stop at the previous line's end
    erow = erow - 1
    ecol = #vim.api.nvim_buf_get_lines(0, erow - 1, erow, false)[1]
  end

  -- The region is raw CSV text, where a quote inside a value is doubled.
  -- When yanking one, swap in the decoded value afterwards.
  if vim.fn.mode(1):sub(1, 2) == "no" and vim.v.operator == "y" and f.quoted and f.value:find('"', 1, true) then
    vim.api.nvim_create_autocmd("TextYankPost", {
      buffer = vim.api.nvim_get_current_buf(),
      once = true,
      callback = function()
        local ev = vim.v.event
        local reg = ev.regname ~= "" and ev.regname or '"'
        vim.fn.setreg(reg, f.value, ev.regtype)
        if reg == '"' then
          vim.fn.setreg("0", f.value, ev.regtype)
        end
      end,
    })
  end

  return { from = { line = srow, col = scol + 1 }, to = { line = erow, col = ecol } }
end

-- `ic` for setups without mini.ai: select the region in charwise visual.
function M.select_cell()
  local region = M.cell_region()
  if not region then
    return
  end
  local mode = vim.fn.mode()
  if mode == "v" or mode == "V" or mode == "\22" then
    vim.cmd("normal! " .. mode) -- leave visual so `v` below starts fresh
  end
  vim.api.nvim_win_set_cursor(0, { region.from.line, region.from.col - 1 })
  vim.cmd("normal! v")
  vim.api.nvim_win_set_cursor(0, { region.to.line, region.to.col - 1 })
end

function M.yank_header()
  local records = M.parse(vim.api.nvim_buf_get_lines(0, 0, 1, false))
  if not records[1] then
    return
  end
  local names = vim.tbl_map(function(f)
    return f.value
  end, records[1])
  local reg = vim.v.register
  vim.fn.setreg(reg, table.concat(names, ", "))
  vim.notify(("Yanked %d column names"):format(#names))
end

local function sql_literal(s)
  return "'" .. s:gsub("'", "''") .. "'"
end

local function ellipsis(s, n)
  if vim.fn.strchars(s) <= n then
    return s
  end
  return vim.fn.strcharpart(s, 0, n - 1) .. "…"
end

-- Jumps stay inside the results window. Each one takes the window over and is
-- pushed onto a stack kept on that window, which the winbar draws as a strip
-- of tabs -- so the surrounding layout (the drawer, the query buffer) is left
-- alone, and the query a jump came from is still there to go back to.
--
-- That is only possible because a stacked buffer is flipped to
-- `bufhidden=hide`. dadbod gives every dbout buffer `bufhidden=delete`
-- (db.vim:410), so the moment a jump took the window, the results it was
-- followed from were unloaded -- rows, `b:db_input` and all -- rather than
-- merely displaced. That is the real reason going back never brought the same
-- query back. It is done here, on the buffers actually stacked, rather than
-- for every dbout buffer from a FileType autocmd: those run before dadbod's
-- own BufReadPost sets the option, so it would simply be overwritten, and
-- results nobody jumped from are better left to dadbod to clean up.
--
-- In exchange these buffers are ours to delete: when closed, dropped from the
-- stack, or left behind by an unrelated query landing in the window.

local function discard(buf)
  if vim.api.nvim_buf_is_loaded(buf) and vim.fn.bufwinid(buf) == -1 then
    pcall(vim.api.nvim_buf_delete, buf, { force = true })
  end
end

-- What a tab is called: the step that opened it, or for a root query the first
-- table it reads.
local function label_of(buf)
  local label = vim.b[buf].dbout_label
  if not label or label == "" then
    label = "results"
    local input = vim.b[buf].db_input
    if input and vim.fn.filereadable(input) == 1 then
      local tables = M.query_tables(table.concat(vim.fn.readfile(input), "\n"))
      label = tables[1] or label
    end
    vim.b[buf].dbout_label = label
  end
  return label
end

-- The window's stack and where in it we are, dropping buffers that are gone.
-- A buffer that is not on the stack at all is an unrelated query that has just
-- landed in the window, which makes it a new root and the old stack rubbish.
local function sync(win)
  local cur = vim.api.nvim_win_get_buf(win)
  local stack, idx = {}, nil
  for _, buf in ipairs(vim.w[win].dbout_stack or {}) do
    if vim.api.nvim_buf_is_loaded(buf) then
      table.insert(stack, buf)
      if buf == cur then
        idx = #stack
      end
    end
  end
  if not idx then
    for _, buf in ipairs(stack) do
      discard(buf)
    end
    stack, idx = { cur }, 1
  end
  for _, buf in ipairs(stack) do
    vim.bo[buf].bufhidden = "hide"
  end
  vim.w[win].dbout_stack = stack
  return stack, idx
end

-- One tab per stack entry, the current one highlighted. Hidden entirely while
-- there is nothing to move between, so an ordinary query keeps the whole
-- window.
local function render(win)
  local stack, idx = sync(win)
  if #stack < 2 then
    vim.wo[win][0].winbar = ""
    return
  end
  local tabs = {}
  for i, buf in ipairs(stack) do
    local hl = i == idx and "%#TabLineSel#" or "%#TabLine#"
    table.insert(tabs, hl .. " " .. label_of(buf):gsub("%%", "%%%%") .. " ")
  end
  vim.wo[win][0].winbar = table.concat(tabs, "%#WinBar#▏") .. "%#WinBar#"
end

-- Move `delta` tabs along the stack. The buffers are all still loaded, so this
-- is a plain buffer swap -- no query is re-run.
function M.jump(delta)
  local win = vim.api.nvim_get_current_win()
  local stack, idx = sync(win)
  local target = stack[idx + delta]
  if not target then
    vim.notify(
      delta < 0 and "Results: already at the first" or "Results: already at the last",
      vim.log.levels.WARN
    )
    return false
  end
  vim.api.nvim_win_set_buf(win, target)
  render(win)
  return true
end

-- Close the current jump and land back on the results it was followed from.
-- False at a root query, which is the caller's cue to close the window itself.
function M.close_jump()
  local win = vim.api.nvim_get_current_win()
  local stack, idx = sync(win)
  if idx == 1 then
    return false
  end
  local gone = table.remove(stack, idx)
  vim.w[win].dbout_stack = stack
  vim.api.nvim_win_set_buf(win, stack[idx - 1])
  discard(gone)
  render(win)
  return true
end

-- Every results buffer this window is holding, for the caller to clean up when
-- it closes the window for good.
function M.stack_bufs(win)
  win = win or vim.api.nvim_get_current_win()
  if not vim.api.nvim_win_is_valid(win) then
    return {}
  end
  return (sync(win))
end

-- Tables named after FROM / JOIN in the query that produced this buffer, used
-- to prefer the right foreign key when several tables share a column name.
function M.query_tables(sql)
  sql = sql:gsub("%-%-[^\n]*", " "):gsub("/%*.-%*/", " "):gsub("'[^']*'", "''")
  local tables, seen = {}, {}
  for _, kw in ipairs({ "[Ff][Rr][Oo][Mm]", "[Jj][Oo][Ii][Nn]" }) do
    for name in sql:gmatch("%f[%w_]" .. kw .. "%s+([%w_%.\"]+)") do
      if not seen[name] then
        seen[name] = true
        table.insert(tables, name)
      end
    end
  end
  return tables
end

local function psql(url, sql)
  local cmd = vim.fn["db#adapter#dispatch"](url, "interactive")
  vim.list_extend(cmd, { "-X", "-At", "-F", "\t", "-v", "ON_ERROR_STOP=1", "-c", sql })
  local res = vim.system(cmd, { text = true }):wait(10000)
  if res.code ~= 0 then
    error(vim.trim(res.stderr ~= "" and res.stderr or "psql failed"), 0)
  end
  local rows = {}
  for _, line in ipairs(vim.split(vim.trim(res.stdout), "\n", { trimempty = true })) do
    table.insert(rows, vim.split(line, "\t", { plain = true }))
  end
  return rows
end

function M.foreign_key()
  local buf = vim.api.nvim_get_current_buf()
  local db = vim.b[buf].db
  local url = type(db) == "table" and db.db_url or db
  if type(url) ~= "string" or not url:match("^postgres") then
    vim.notify("Foreign key jump: only Postgres results are supported", vim.log.levels.WARN)
    return
  end

  local records, ri, fi, f = under_cursor(buf)
  if not f or ri == 1 then
    vim.notify("Foreign key jump: put the cursor on a value, not the header", vim.log.levels.WARN)
    return
  end
  local header = records[1][fi]
  if not header then
    vim.notify("Foreign key jump: this row has more fields than the header", vim.log.levels.WARN)
    return
  end
  if f.value == "" and not f.quoted then
    vim.notify("Foreign key jump: value is NULL", vim.log.levels.WARN)
    return
  end
  local column = header.value

  local candidates = {}
  local input = vim.b[buf].db_input
  if input and vim.fn.filereadable(input) == 1 then
    for _, t in ipairs(M.query_tables(table.concat(vim.fn.readfile(input), "\n"))) do
      table.insert(candidates, sql_literal(t))
    end
  end

  -- Single-column foreign keys touching a column with this name, both ways:
  --   fk  -- declared on the column: open the row it points at
  --   ref -- the column is what a key points at: open the rows pointing here
  -- `local_` says whether the side holding the column is a table the query
  -- reads from; `known` counts how many query names are real tables
  -- (to_regclass() yields NULL for CTEs, functions and the like).
  local lookup = ([[
    with q as (select to_regclass(t) oid from unnest(array[%s]::text[]) t),
    keys as (
      select c.conrelid, c.confrelid, a.attname src_col, fa.attname dst_col
      from pg_constraint c
      cross join lateral unnest(c.conkey, c.confkey) k(attnum, fattnum)
      join pg_attribute a on a.attrelid = c.conrelid and a.attnum = k.attnum
      join pg_attribute fa on fa.attrelid = c.confrelid and fa.attnum = k.fattnum
      where c.contype = 'f' and cardinality(c.conkey) = 1
    )
    select 'fk', conrelid::regclass::text, confrelid::regclass::text, quote_ident(dst_col),
           conrelid in (select oid from q) local_, (select count(oid) from q)
    from keys where src_col = %s
    union all
    select 'ref', confrelid::regclass::text, conrelid::regclass::text, quote_ident(src_col),
           confrelid in (select oid from q), (select count(oid) from q)
    from keys where dst_col = %s
    order by 1, 3, 4]]):format(table.concat(candidates, ","), sql_literal(column), sql_literal(column))

  local ok, rows = pcall(psql, url, lookup)
  if not ok then
    vim.notify("Foreign key jump: " .. rows, vim.log.levels.ERROR)
    return
  end

  -- When the query's tables are known, only keys on their side count: a
  -- same-named key elsewhere would, on `select * from departments`, follow
  -- employees.dept_id back into departments itself. With no known table
  -- (`select 7 as customer_id`) guess forward keys by name; which rows point
  -- at an anonymous value cannot be guessed.
  local matches, seen = {}, {}
  local scoped = rows[1] and tonumber(rows[1][6]) > 0
  for _, r in ipairs(rows) do
    local kind, from, target, target_col, is_local = r[1], r[2], r[3], r[4], r[5] == "t"
    local wanted = scoped and is_local or (not scoped and kind == "fk")
    local key = kind .. target .. "." .. target_col
    if wanted and not seen[key] then
      seen[key] = true
      table.insert(matches, { kind = kind, from = from, table = target, column = target_col })
    end
  end
  if #matches == 0 then
    vim.notify(
      ("Foreign key jump: no foreign key on or referencing column %s"):format(column),
      vim.log.levels.WARN
    )
    return
  end

  -- `:DB` pedits, which lands in the tab's preview window -- the very window
  -- these results are in. That is what we want: the jump takes the window over
  -- and is pushed onto its stack, leaving the rest of the layout alone. The
  -- results it came from stay loaded and one `q` away.
  local function open(m)
    local win = vim.api.nvim_get_current_win()
    local file = vim.fn.tempname() .. ".sql"
    vim.fn.writefile(
      { ("select * from %s where %s = %s;"):format(m.table, m.column, sql_literal(f.value)) },
      file
    )

    -- Jumping from the middle of the stack drops whatever was ahead, the way
    -- following a link partway through a history does.
    local stack, idx = sync(win)
    for i = #stack, idx + 1, -1 do
      discard(table.remove(stack, i))
    end

    -- Pass the URL by variable: fnameescape() would turn `?` into `\?`, which
    -- :DB no longer recognises as a URL, and then silently runs nothing. It is
    -- global rather than buffer-local because vim.ui.select's callback may not
    -- have restored the buffer this was started from.
    vim.g.dbout_fk_url = url
    local opened, err = pcall(vim.cmd, "DB g:dbout_fk_url < " .. vim.fn.fnameescape(file))
    vim.g.dbout_fk_url = nil
    if not opened then
      vim.notify("Foreign key jump: " .. err, vim.log.levels.ERROR)
      return
    end

    -- pedit does not move the cursor when it is run from outside the preview
    -- window, so make sure the jump is what is focused either way.
    vim.api.nvim_set_current_win(win)
    local landed = vim.api.nvim_win_get_buf(win)
    vim.bo[landed].bufhidden = "hide"
    vim.b[landed].dbout_label = ("%s.%s = %s"):format(m.table, m.column, ellipsis(f.value, 40))
    table.insert(stack, landed)
    vim.w[win].dbout_stack = stack
    render(win)
  end

  if #matches == 1 then
    return open(matches[1])
  end
  vim.ui.select(matches, {
    prompt = ("Follow %s = %s"):format(column, f.value),
    format_item = function(m)
      if m.kind == "fk" then
        return ("→ %s.%s  (key on %s)"):format(m.table, m.column, m.from)
      end
      return ("← %s.%s  (rows referencing %s)"):format(m.table, m.column, m.from)
    end,
  }, function(choice)
    if choice then
      open(choice)
    end
  end)
end

return M
