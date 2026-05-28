-- Keymap presets for pmusic
-- Usage in init.lua:
--   require("plugins/keymaps")
--
-- All bindings are additive — core keys (j/k/h/l/q…) remain active.

-- ── Media key aliases ─────────────────────────────────────────────────────────
-- Uncomment the lines you want.

-- pmusic.register_keymap("f",   "next")           -- f → next track
-- pmusic.register_keymap("b",   "prev")           -- b → previous track
-- pmusic.register_keymap("x",   "toggle_pause")   -- x → pause/resume (MPD-style)
-- pmusic.register_keymap("z",   "prev")           -- z → previous     (MPD-style)
-- pmusic.register_keymap("c",   "toggle_pause")   -- c → pause        (MPD-style)
-- pmusic.register_keymap("v",   "quit")           -- v → stop / quit  (MPD-style)

-- ── Panel navigation ──────────────────────────────────────────────────────────
-- pmusic.register_keymap("tab",   "focus_tracks")
-- pmusic.register_keymap("S-tab", "focus_folders")  -- shift+tab (terminal may send different string)

-- ── Loop ──────────────────────────────────────────────────────────────────────
-- pmusic.register_keymap("L", "loop")   -- capital L → toggle loop

-- ── Reload ────────────────────────────────────────────────────────────────────
pmusic.register_keymap("ctrl+l", "reload_lua")   -- ctrl+l → reload Lua config
