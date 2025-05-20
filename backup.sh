#!/bin/bash

function delete_folders {
  folders=("${@}")
  for folder in "${folders[@]}"; do
    rm -rf "$folder"
  done
}

function create_if_not_exist {
  object="$1"
  isFolder="$2"

  if [[ "$isFolder" == true && ! -d "$object" ]]; then
    echo "Creating folder $object ..."
    mkdir -p "$object"
  elif [[ "$isFolder" == false && ! -f "$object" ]]; then
    echo "Creating file $object ..."
    touch "$object"
  fi
}

function copy_if_exists {
  object="$1"
  isFolder="$2"
  target="$3"

  if [[ "$isFolder" == true && -d "$object" ]]; then
    echo "Copying folder $object ..."
    cp -R "$object" "$target" && echo "$object copied successfully" || echo "Error copying $object"
  elif [[ "$isFolder" == false && -f "$object" ]]; then
    echo "Copying file $object ..."
    mkdir -p "$target"
    cp "$object" "$target" && echo "$object copied successfully" || echo "Error copying $object"
  else
    echo "Warning: $object does not exist or type mismatch"
  fi
}

# Delete and recreate folders
folders=("zsh" "nvim" "powerlevel10k" "zellij" "wezterm")

delete_folders "${folders[@]}"

for folder in "${folders[@]}"; do
  create_if_not_exist "$folder" true
done

# Copy configuration files
config_files=(
  "file:$HOME/.zshrc:./zsh/"
  "folder:$HOME/.config/nvim:./"
  "file:$HOME/.p10k.zsh:./powerlevel10k/"
  "folder:$HOME/.config/zellij:./"
  "folder:$HOME/.config/wezterm:./"
)

for config in "${config_files[@]}"; do
  IFS=":" read -r type path target <<<"$config"
  echo "Processing $path ..."

  if [[ "$type" == "file" ]]; then
    copy_if_exists "$path" false "$target"
  elif [[ "$type" == "folder" ]]; then
    copy_if_exists "$path" true "$target"
  else
    echo "Invalid config type: $type"
  fi
done
