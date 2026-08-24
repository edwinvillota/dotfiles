--- chafa.yazi — render image previews as terminal text instead of using a
--- graphics protocol.
---
--- Why this exists: inside a multiplexer, yazi has no image driver of its own
--- (`B::Zellij => vec![]`), so it falls through to sixel passthrough. Zellij
--- relays the escape sequence to the terminal but does not track the image
--- placement, so yazi's `image_erase()` never reaches it and previews bleed
--- through modals (delete confirmation, input, etc).
---
--- chafa renders the image as coloured glyphs, which live in the normal text
--- layer. Occlusion then works like any other widget. Lower fidelity, but
--- correct — and it works inside Zellij.

local M = {}

local function msg(job, s)
	ya.preview_widget(job, ui.Text(ui.Line(s)):area(job.area):wrap(ui.Wrap.YES))
end

function M:peek(job)
	local w, h = math.max(1, job.area.w), math.max(1, job.area.h)

	-- Debounce exactly like yazi's built-in image previewer: sleep first, so
	-- scrolling quickly through a directory cancels the task before chafa is
	-- ever spawned. Without this, every cursor move spawns a process.
	local start = os.clock()
	ya.sleep(math.max(0, rt.preview.image_delay / 1000 + start - os.clock()))

	local output, err = Command("chafa")
		:arg({
			"--format=symbols",
			-- don't emit cursor-hide / terminal-specific control sequences
			"--polite=on",
			"--animate=off",
			-- Truecolor: the debounce above is what fixes latency, so we can
			-- afford the larger payload and keep full colour fidelity.
			-- (Dropping to --colors=256 would cut the ANSI payload ~62% if
			-- scrolling ever feels slow again -- that is the knob to turn.)
			"--colors=full",
			"--color-space=din99d",
			"--work=9",
			-- quad + half give 4 and 2 sub-cells per character respectively,
			-- which is the finest detail available; chafa 1.18 has no
			-- sextant/octant classes despite what some docs suggest.
			"--symbols=block+quad+half+vhalf",
			"--dither=ordered",
			"--size=" .. w .. "x" .. h,
			tostring(job.file.path),
		})
		:stdout(Command.PIPED)
		:stderr(Command.PIPED)
		:output()

	if not output then
		return msg(job, "Failed to run chafa: " .. tostring(err))
	elseif not output.status.success then
		return msg(job, "chafa exited " .. tostring(output.status.code) .. ": " .. output.stderr)
	end

	-- chafa writes one line per glyph row; skip supports scrolling the preview.
	-- Reassemble into a single string because ui.Text.parse (not ui.Text) is what
	-- interprets chafa's ANSI colour sequences -- plain ui.Text prints them raw.
	local kept, n = {}, 0
	for line in output.stdout:gmatch("[^\n]+") do
		n = n + 1
		if n > job.skip then
			kept[#kept + 1] = line
		end
		if #kept >= h then
			break
		end
	end

	ya.preview_widget(job, ui.Text.parse(table.concat(kept, "\n")):area(job.area))
end

function M:seek(job)
	local h = math.max(1, job.area.h)
	local step = math.floor(h / 2)
	ya.emit("peek", {
		math.max(0, job.skip + (job.units > 0 and step or -step)),
		only_if = job.file.url,
	})
end

return M
