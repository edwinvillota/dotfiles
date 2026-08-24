-- Trigger workflow_dispatch workflows from nvim.
--
-- Flow: workflow -> ref -> typed inputs (validated) -> confirm -> dispatch.
-- Deliberately scoped to the current repo: if cwd is not a GitHub repo this
-- refuses to run, so a workflow can never be dispatched against the wrong repo.
local M = {}

local function sh(cmd)
  local out = vim.fn.system(cmd)
  if vim.v.shell_error ~= 0 then return nil, vim.trim(out) end
  return vim.trim(out)
end

local function err(msg) vim.notify(msg, vim.log.levels.ERROR, { title = "gh workflow" }) end
local function info(msg) vim.notify(msg, vim.log.levels.INFO, { title = "gh workflow" }) end

--------------------------------------------------------------------------- parse

--- Parse `workflow_dispatch` out of a workflow YAML.
--- Small indentation scanner: we only need names, types, defaults, options.
---@return table[] inputs, boolean dispatchable
function M.parse_inputs(text)
  local lines = vim.split(text, "\n", { plain = true })
  local inputs, in_dispatch, dispatchable, inputs_indent, cur = {}, false, false, nil, nil
  local dispatch_indent

  local function indent_of(l) return #(l:match("^(%s*)") or "") end

  for _, line in ipairs(lines) do
    if line:match("^%s*#") or line:match("^%s*$") then goto continue end

    local ind = indent_of(line)

    if line:match("^%s*workflow_dispatch:%s*$") or line:match("^%s*workflow_dispatch:%s*{%s*}%s*$") then
      in_dispatch, dispatchable, dispatch_indent = true, true, ind
      goto continue
    end

    if in_dispatch then
      -- any key at or shallower than workflow_dispatch ends the block
      if ind <= dispatch_indent and line:match("^%s*[%w_-]+:") then
        in_dispatch, cur, inputs_indent = false, nil, nil
        goto continue
      end

      if line:match("^%s*inputs:%s*$") then
        inputs_indent = ind
        goto continue
      end

      if inputs_indent then
        local name = line:match("^%s*([%w_%-%.]+):%s*$")
        if name and ind == inputs_indent + 2 then
          cur = { name = name, options = {}, type = "string" }
          table.insert(inputs, cur)
          goto continue
        end

        if cur then
          local k, v = line:match("^%s*([%w_-]+):%s*(.-)%s*$")
          if k and v and v ~= "" and not line:match("^%s*%-") then
            v = v:gsub('^"(.*)"$', "%1"):gsub("^'(.*)'$", "%1")
            if k == "type" then cur.type = v
            elseif k == "default" then cur.default = v
            elseif k == "required" then cur.required = (v == "true")
            elseif k == "description" then cur.description = v
            end
          end
          local opt = line:match("^%s*%-%s*(.+)$")
          if opt then
            opt = vim.trim(opt):gsub('^"(.*)"$', "%1"):gsub("^'(.*)'$", "%1")
            table.insert(cur.options, opt)
          end
        end
      end
    end
    ::continue::
  end

  return inputs, dispatchable
end

------------------------------------------------------------------------ validate

--- Validate a raw string against an input spec.
---@return boolean ok, string? errmsg
function M.validate(inp, value)
  local required = inp.required == true

  if value == nil or value == "" then
    if required then return false, ("%s is required"):format(inp.name) end
    return true
  end

  if inp.type == "choice" then
    if not vim.tbl_contains(inp.options, value) then
      return false, ("%s must be one of: %s"):format(inp.name, table.concat(inp.options, ", "))
    end
  elseif inp.type == "boolean" then
    if value ~= "true" and value ~= "false" then
      return false, ("%s must be true or false"):format(inp.name)
    end
  elseif inp.type == "number" then
    if not tonumber(value) then
      return false, ("%s must be a number (got %q)"):format(inp.name, value)
    end
  end

  return true
end

---------------------------------------------------------------------------- ui

-- Prompt for one input, re-prompting until it validates or the user aborts.
local function ask(inp, done)
  local label = inp.name
  if inp.required then label = label .. " *" end
  if inp.description then label = label .. " — " .. inp.description end

  local function finish(value)
    local ok, msg = M.validate(inp, value)
    if not ok then
      err(msg)
      return vim.schedule(function() ask(inp, done) end)
    end
    done(value)
  end

  if inp.type == "choice" and #inp.options > 0 then
    local items = vim.deepcopy(inp.options)
    if not inp.required then table.insert(items, "<skip>") end
    vim.ui.select(items, {
      prompt = label,
      format_item = function(o)
        return (o == inp.default) and (o .. "  (default)") or o
      end,
    }, function(choice)
      if choice == nil then return done(nil, true) end
      finish(choice == "<skip>" and "" or choice)
    end)
  elseif inp.type == "boolean" then
    vim.ui.select({ "true", "false" }, { prompt = label }, function(choice)
      if choice == nil then return done(nil, true) end
      finish(choice)
    end)
  else
    vim.ui.input({ prompt = label .. ": ", default = inp.default or "" }, function(val)
      if val == nil then return done(nil, true) end
      finish(val)
    end)
  end
end

local function collect(inputs, idx, args, done)
  if idx > #inputs then return done(args) end
  local inp = inputs[idx]
  ask(inp, function(value, aborted)
    if aborted then return info("Cancelled") end
    if value and value ~= "" then
      table.insert(args, "-f")
      table.insert(args, inp.name .. "=" .. value)
    end
    collect(inputs, idx + 1, args, done)
  end)
end

-------------------------------------------------------------------------- steps

--- Resolve the repo from cwd. No picker fallback on purpose: dispatching is a
--- write action, so it must target the repo you are actually working in.
local function current_repo()
  local repo = sh({ "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner" })
  if repo and repo ~= "" then return repo end
  return nil
end

local function pick_ref(repo, cb)
  local default = sh({ "gh", "api", "repos/" .. repo, "--jq", ".default_branch" }) or "main"
  local raw = sh({ "gh", "api", "repos/" .. repo .. "/branches?per_page=100", "--jq", "[.[].name]" })
  local branches = {}
  if raw then
    local ok, b = pcall(vim.json.decode, raw)
    if ok then branches = b end
  end

  -- default branch first
  table.sort(branches, function(a, b)
    if a == default then return true end
    if b == default then return false end
    return a < b
  end)
  if #branches == 0 then branches = { default } end

  vim.ui.select(branches, {
    prompt = "Ref",
    format_item = function(b) return (b == default) and (b .. "  (default)") or b end,
  }, function(ref)
    if ref then cb(ref) end
  end)
end

function M.run()
  local repo = current_repo()
  if not repo then
    return err("Not inside a GitHub repository — cd into the repo first")
  end

  local raw = sh({ "gh", "workflow", "list", "-R", repo, "--json", "name,id,path" })
    if not raw then return err("gh workflow list failed for " .. repo) end

    local ok, workflows = pcall(vim.json.decode, raw)
    if not ok or #workflows == 0 then return err("No workflows in " .. repo) end

    vim.ui.select(workflows, {
      prompt = "Workflow in " .. repo,
      format_item = function(w) return w.name end,
    }, function(wf)
      if not wf then return end

      pick_ref(repo, function(ref)
        local body = sh({ "gh", "api",
          ("repos/%s/contents/%s?ref=%s"):format(repo, wf.path, ref), "--jq", ".content" })
        if not body then
          return err(("%s not found on ref %q"):format(wf.path, ref))
        end

        local decoded = vim.base64.decode((body:gsub("%s", "")))
        local inputs, dispatchable = M.parse_inputs(decoded or "")

        if not dispatchable then
          return err(("%q has no workflow_dispatch trigger on %s — not dispatchable")
            :format(wf.name, ref))
        end

        collect(inputs, 1, {}, function(args)
          local cmd = { "gh", "workflow", "run", wf.path:match("[^/]+$"),
            "-R", repo, "--ref", ref }
          vim.list_extend(cmd, args)

          vim.ui.select({ "Dispatch", "Cancel" }, {
            prompt = table.concat(cmd, " "),
          }, function(choice)
            if choice ~= "Dispatch" then return info("Cancelled") end
            local out, e = sh(cmd)
            if not out then return err("Dispatch failed: " .. (e or "")) end
            info(("Dispatched %s on %s"):format(wf.name, ref))
          end)
        end)
      end)
    end)
end

--------------------------------------------------------------------------- watch

local STATUS_ICON = {
  completed = "",
  in_progress = "",
  queued = "",
  requested = "",
  waiting = "",
  pending = "",
}

local CONCLUSION_ICON = {
  success = "",
  failure = "",
  cancelled = "",
  skipped = "",
  timed_out = "",
  action_required = "",
  startup_failure = "",
  neutral = "",
}

--- Open `gh run watch` for a run picked from the current repo's recent runs.
--- Finished runs are shown too — watching one just prints its final state,
--- so they are opened with `view` instead.
function M.watch()
  local repo = current_repo()
  if not repo then
    return err("Not inside a GitHub repository — cd into the repo first")
  end

  local raw = sh({ "gh", "run", "list", "-R", repo, "-L", "25",
    "--json", "databaseId,displayTitle,workflowName,headBranch,status,conclusion,createdAt" })
  if not raw then return err("gh run list failed for " .. repo) end

  local ok, runs = pcall(vim.json.decode, raw)
  if not ok or #runs == 0 then return err("No recent runs in " .. repo) end

  vim.ui.select(runs, {
    prompt = "Watch run in " .. repo,
    format_item = function(r)
      local icon = (r.status == "completed")
        and (CONCLUSION_ICON[r.conclusion] or "")
        or (STATUS_ICON[r.status] or "")
      local when = (r.createdAt or ""):gsub("T", " "):sub(1, 16)
      return ("%s %-18s %-16s %s"):format(icon, r.workflowName or "?", r.headBranch or "?", when)
    end,
  }, function(run)
    if not run then return end

    -- A finished run has nothing to stream; show its result instead.
    local sub = (run.status == "completed") and "view" or "watch"
    local cmd = ("gh run %s %d -R %s"):format(sub, run.databaseId, repo)
    if sub == "view" then cmd = cmd .. " --log-failed" end

    require("snacks").terminal(cmd, {
      cwd = vim.fn.getcwd(),
      win = {
        style = "terminal",
        border = "rounded",
        width = 0.9,
        height = 0.9,
        title = (" %s — %s "):format(sub == "watch" and "Watching" or "Run", run.workflowName or ""),
        title_pos = "center",
      },
    })
  end)
end

return M
