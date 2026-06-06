-- Time-of-day theme scheduler for pmusic
-- Automatically applies a theme based on the current hour when
-- init.lua loads (startup or Ctrl+R).
--
-- Usage in init.lua:
--   require("plugins/theme-scheduler")
--
-- Schedule (24-hour clock, adjust to taste):
--   07:00 – 16:59  →  Catppuccin Latte  (light, easy on daytime eyes)
--   17:00 – 21:59  →  Tokyo Night Storm  (warm evening colours)
--   22:00 – 06:59  →  Nord               (dark & muted for night)
--
-- Tip: press Ctrl+R at any time to reapply the schedule for the current hour.

local hour = tonumber(os.date("%H"))

if hour >= 7 and hour < 17 then
    -- ── Catppuccin Latte (light) ──────────────────────────────────────────────
    pmusic.set_theme({
        accent       = "#1e66f5",   -- blue
        dim          = "#9ca0b0",   -- subtext0
        selected_bg  = "#ccd0da",   -- surface1
        now_playing  = "#40a02b",   -- green
        border       = "#bcc0cc",   -- surface2
        border_active= "#1e66f5",   -- blue
        title        = "#8839ef",   -- mauve
        status_bg    = "#e6e9ef",   -- mantle
        panel_bg     = "#eff1f5",   -- base
        key          = "#fe640b",   -- peach
    })
    pmusic.notify("Theme: Catppuccin Latte")

elseif hour >= 17 and hour < 22 then
    -- ── Tokyo Night Storm (evening) ───────────────────────────────────────────
    pmusic.set_theme({
        accent       = "#7aa2f7",   -- blue
        dim          = "#565f89",   -- comment
        selected_bg  = "#292e42",   -- selection
        now_playing  = "#9ece6a",   -- green
        border       = "#3b4261",   -- terminal black
        border_active= "#7aa2f7",   -- blue
        title        = "#bb9af7",   -- purple
        status_bg    = "#1f2335",   -- bg_dark
        panel_bg     = "#24283b",   -- bg
        key          = "#ff9e64",   -- orange
    })
    pmusic.notify("Theme: Tokyo Night Storm")

else
    -- ── Nord (night) ──────────────────────────────────────────────────────────
    pmusic.set_theme({
        accent       = "#88C0D0",   -- nord8
        dim          = "#4C566A",   -- nord3
        selected_bg  = "#3B4252",   -- nord1
        now_playing  = "#A3BE8C",   -- nord14
        border       = "#4C566A",   -- nord3
        border_active= "#88C0D0",   -- nord8
        title        = "#81A1C1",   -- nord9
        status_bg    = "#2E3440",   -- nord0
        panel_bg     = "#2E3440",   -- nord0
        key          = "#EBCB8B",   -- nord13
    })
    pmusic.notify("Theme: Nord")
end
