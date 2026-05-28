-- pmusic Lua config
-- Location: ~/.config/pmusic/lua/init.lua
--
-- Copy this file (and the themes/ and plugins/ folders) to
-- ~/.config/pmusic/lua/ to activate scripting.
--
-- Hot-reload: press ctrl+r inside pmusic to apply changes without restarting.

-- ── Theme ─────────────────────────────────────────────────────────────────────
-- Option A: use a bundled theme (copy the file to themes/ first)
-- require("themes/gruvbox")
-- require("themes/catppuccin")
-- require("themes/tokyo-night")

-- Option B: define colors inline (any subset of keys is fine)
pmusic.set_theme({
    -- Accent: active border, progress bar, title
    accent        = "#88C0D0",  -- Nord cyan

    -- Inactive text and empty progress track
    dim           = "#4C566A",

    -- Cursor row background
    selected_bg   = "#434C5E",

    -- Color of the currently playing track name
    now_playing   = "#A3BE8C",  -- Nord green

    -- Panel borders
    border        = "#4C566A",
    border_active = "#88C0D0",

    -- Status bar
    title         = "#88C0D0",
    status_bg     = "#3B4252",

    -- Canvas / panel background
    panel_bg      = "#2E3440",

    -- Key-hint labels
    key           = "#81A1C1",
})

-- ── Keymaps ───────────────────────────────────────────────────────────────────
-- Bind extra keys to built-in actions (additive — core bindings stay).
--
-- Available actions:
--   toggle_pause  next  prev  loop
--   focus_folders  focus_tracks
--   reload_lua  quit

-- require("plugins/keymaps")   -- load the preset keymap file

-- Or define individual bindings:
-- pmusic.register_keymap("f",      "next")
-- pmusic.register_keymap("b",      "prev")
-- pmusic.register_keymap("ctrl+l", "reload_lua")

-- ── Plugins ───────────────────────────────────────────────────────────────────
-- require("plugins/logger")   -- log played tracks to ~/.local/share/pmusic/plays.log
-- require("plugins/stats")    -- show play-count notifications

-- ── Hooks ─────────────────────────────────────────────────────────────────────
-- on_song_change(fn) → fires when a new track starts.
-- track = { name: string, path: string, folder: string }

pmusic.on_song_change(function(track)
    pmusic.notify("▶  " .. track.name)
end)

-- on_state_change(fn) → fires on play / pause / stop transitions.
-- state = "playing" | "paused" | "stopped"

pmusic.on_state_change(function(state)
    -- pmusic.notify("state → " .. state)
end)
