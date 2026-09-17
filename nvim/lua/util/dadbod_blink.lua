-- vim-dadbod-completion's blink.cmp source, with a fix for completing right
-- after punctuation.
--
-- The upstream source finds the word being typed by walking back to the
-- previous whitespace or trigger character, then gives up (sends an empty
-- base) if that word contains anything but [0-9A-Za-z_]. So `sum(pop_`,
-- `within group (pop_`, `a*b` or `x>y` complete nothing: the word it sees is
-- `(pop_`. Blank out other punctuation in the line it reads, keeping byte
-- offsets intact, so the word starts after the `(`. Everything else,
-- including the item mapping, stays upstream's.

local upstream = require("vim_dadbod_completion.blink")

local M = setmetatable({}, { __index = upstream })

function M.new()
  return setmetatable({}, { __index = M })
end

function M:get_completions(ctx, callback)
  -- Quotes, brackets and `.` are upstream's trigger characters and `:` marks
  -- bind parameters; keep those, turn the rest into spaces.
  local line = ctx.line:gsub("[^%w_%s\"`%[%]%.:]", " ")
  local proxy = setmetatable({ line = line }, { __index = ctx })
  return upstream.get_completions(self, proxy, function(response)
    response.context = ctx
    callback(response)
  end)
end

return M
