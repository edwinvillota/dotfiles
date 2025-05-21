# Purpose

This folder contains modular extensions for your `.zshrc` configuration. Each
file in this directory is intended to be sourced from your main `.zshrc`,
allowing you to organize and maintain your shell configuration in a clean,
scalable way.

## How to Add a New Extension

1. Create a new `.zsh` file in this folder (e.g., `my-extension.zsh`).
2. Add your custom Zsh configuration, aliases, functions, or environment
   variables to this file.
3. In your main `.zshrc`, source the extension by adding:

   ```sh
   source /path/to/this/folder/my-extension.zsh
   ```

4. Reload your shell or run `source ~/.zshrc` to apply the changes.

## Best Practices

- Use descriptive filenames for each extension (e.g., `git-aliases.zsh`, `docker.zsh`).
- Keep each extension focused on a single topic or tool.
- Add comments to explain non-obvious configurations.
- Avoid duplicating settings across extensions.
- Test each extension independently before sourcing it in your main `.zshrc`.
- Use version control to track changes.

  ```zsh
  for config in ~/.zshrc.d/*.zsh; do
    [ -r "$config" ] && source "$config"
  done
  ```
