local script_filetypes = {
  javascript = true,
  javascriptreact = true,
  typescript = true,
  typescriptreact = true,
}

local script_extensions = { "ts", "tsx", "mts", "cts", "js", "jsx", "mjs", "cjs" }

-- True when the cursor sits inside the specifier of an import/export/require.
local function in_module_specifier(ctx)
  local before = ctx.line:sub(1, ctx.cursor[2])
  return before:match("from%s*['\"][^'\"]*$")
    or before:match("import%s*['\"][^'\"]*$")
    or before:match("import%s*%(%s*['\"][^'\"]*$")
    or before:match("require%s*%(%s*['\"][^'\"]*$")
end

local function is_script_file(name)
  local ext = name:match("%.([%w]+)$")
  return ext ~= nil and vim.tbl_contains(script_extensions, ext)
end

return {
  "saghen/blink.cmp",
  opts = {
    keymap = {
      preset = "enter", -- keep the default preset
      ["<M-Space>"] = { "show" }, -- <C-Space> opens the completion menu
    },
    sources = {
      providers = {
        path = {
          -- In JS/TS import strings the LSP already completes folders and
          -- modules (extensionless, with tsconfig aliases). Leave those to it
          -- so the path source doesn't insert "Button.tsx"; keep assets
          -- (.css, .svg, .jpg...), which need their extension.
          transform_items = function(ctx, items)
            if not script_filetypes[vim.bo[ctx.bufnr].filetype] or not in_module_specifier(ctx) then
              return items
            end
            return vim.tbl_filter(function(item)
              local data = item.data or {}
              return data.type ~= "directory" and not is_script_file(data.path or item.label)
            end, items)
          end,
        },
      },
    },
  },
}
