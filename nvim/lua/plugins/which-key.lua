return {
  "folke/which-key.nvim",
  keys = {
    {
      "<leader>bc",
      function()
        local git_root = vim.fn.system("git rev-parse --show-toplevel 2>/dev/null"):gsub("\n", "")
        local full_path = vim.fn.expand("%:p")

        if git_root ~= "" then
          local relative_path = full_path:gsub("^" .. vim.pesc(git_root .. "/"), "")
          vim.fn.setreg("+", relative_path)
          print("Copied relative path: " .. relative_path)
        else
          -- Fallback to current working directory
          local path = vim.fn.expand("%:.")
          vim.fn.setreg("+", path)
          print("Copied relative path: " .. path)
        end
      end,
      desc = "Copy relative path",
    },
  },
}
