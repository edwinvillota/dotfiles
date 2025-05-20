-- Import the wezterm API
local wezterm = require("wezterm")

-- Initialize an empty configuration table
local config = {
	default_prog = { "/Users/edwinvillota/.cargo/bin/zellij" },
}

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
}

-- Color scheme
config.color_scheme = "Monokai Octagon"

-- Background with transparency
config.window_background_opacity = 1
config.macos_window_background_blur = 0

-- Disable Scroll bar
config.enable_scroll_bar = false

return config
