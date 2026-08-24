-- Import the wezterm API
local wezterm = require("wezterm")

-- Launch zellij when it is installed, from wherever it is installed
-- (cargo, Homebrew on macOS or Linux, system package). On a machine
-- without zellij, fall back to the default shell instead of erroring.
local function find_zellij()
	local home = wezterm.home_dir
	local candidates = {
		home .. "/.cargo/bin/zellij",
		"/opt/homebrew/bin/zellij",
		"/home/linuxbrew/.linuxbrew/bin/zellij",
		"/usr/local/bin/zellij",
		"/usr/bin/zellij",
	}
	for _, path in ipairs(candidates) do
		local f = io.open(path, "r")
		if f then
			f:close()
			return path
		end
	end
	return nil
end

-- Initialize the configuration table
local config = {}
local zellij = find_zellij()
if zellij then
	config.default_prog = { zellij }
end

-- Removing window padding
config.window_padding = {
	top = 0,
	right = 0,
	left = 0,
}

-- Set terminal font
config.font = wezterm.font("IosevkaTerm NF")
config.font_size = 16.0

-- Hide the tab bar if only one tab is open
config.hide_tab_bar_if_only_one_tab = true
config.max_fps = 240
config.enable_kitty_graphics = true -- Enables support for the "Kitty graphics protocol", which is a way of displaying images inside the terminal

config.color_schemes = {
	["Monokai Octagon"] = {
		foreground = "#FCFCFA",
		background = "#1C1E1F",
		cursor_bg = "#FCFCFA",
		cursor_fg = "#1C1E1F",
		cursor_border = "#FCFCFA",
		selection_fg = "#1C1E1F",
		selection_bg = "#FCFCFA",
		ansi = {
			"#403E41", -- black
			"#FF6188", -- red
			"#A9DC76", -- green
			"#FFD866", -- yellow
			"#78DCE8", -- blue
			"#AB9DF2", -- magenta
			"#78DCE8", -- cyan
			"#FCFCFA", -- white
		},
		brights = {
			"#727072", -- bright black
			"#FF6188", -- bright red
			"#A9DC76", -- bright green
			"#FFD866", -- bright yellow
			"#78DCE8", -- bright blue
			"#AB9DF2", -- bright magenta
			"#78DCE8", -- bright cyan
			"#FCFCFA", -- bright white
		},
	},
	["Ayu dark"] = {
		foreground = "#e6e1cf",
		background = "#0f1419",
		cursor_bg = "#e6e1cf",
		cursor_fg = "#0f1419",
		cursor_border = "#e6e1cf",
		selection_fg = "#e6e1cf",
		selection_bg = "#253340",
		ansi = {
			"#000000", -- black
			"#ff3333", -- red
			"#b8cc52", -- green
			"#e6c446", -- yellow
			"#36a3d9", -- blue
			"#f07078", -- magenta
			"#95e6cb", -- cyan
			"#bfbdb6", -- white
		},
		brights = {
			"#4c4f69", -- bright black
			"#ff3333", -- bright red
			"#b8cc52", -- bright green
			"#e6c446", -- bright yellow
			"#36a3d9", -- bright blue
			"#f07078", -- bright magenta
			"#95e6cb", -- bright cyan
			"#e6e1cf", -- bright white
		},
	},
}

-- Color scheme
config.color_scheme = "Ayu dark"

-- Background with transparency
config.window_background_opacity = 1
config.macos_window_background_blur = 0

-- Disable Scroll bar
config.enable_scroll_bar = false

return config
