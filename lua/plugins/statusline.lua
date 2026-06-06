-- Statusline bridge plugin for pmusic
-- Writes current track and playback state to a JSON file that external
-- status bars (waybar, polybar, i3blocks, tmux, eww…) can poll.
--
-- Usage in init.lua:
--   require("plugins/statusline")
--
-- Output file: /tmp/pmusic-status.json  (override with PMUSIC_STATUS_FILE)
-- JSON shape:
--   {"name":"Track Name","folder":"Artist","path":"/full/path.mp3","state":"playing"}
--
-- ── Waybar example ────────────────────────────────────────────────────────────
-- In waybar config.json:
--   "custom/pmusic": {
--       "exec": "cat /tmp/pmusic-status.json | jq -r '\"\\(.folder) — \\(.name)\"'",
--       "interval": 2,
--       "format": " {}"
--   }
-- ─────────────────────────────────────────────────────────────────────────────

local STATUS_FILE = PMUSIC_STATUS_FILE or "/tmp/pmusic-status.json"

local current = { name = "", folder = "", path = "", state = "stopped" }

local function escape_json(s)
    -- Minimal JSON string escaping (backslash and double-quote).
    return s:gsub('\\', '\\\\'):gsub('"', '\\"')
end

local function write_status()
    local f = io.open(STATUS_FILE, "w")
    if not f then return end
    f:write(string.format(
        '{"name":"%s","folder":"%s","path":"%s","state":"%s"}\n',
        escape_json(current.name),
        escape_json(current.folder),
        escape_json(current.path),
        escape_json(current.state)))
    f:close()
end

pmusic.on_song_change(function(track)
    current.name   = track.name
    current.folder = track.folder
    current.path   = track.path
    write_status()
end)

pmusic.on_state_change(function(state)
    current.state = state
    write_status()
end)

-- Write an initial "stopped" snapshot so pollers see a valid file immediately.
write_status()
