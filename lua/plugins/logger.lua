-- Track play logger for pmusic
-- Appends every played track (with timestamp) to a log file.
--
-- Usage in init.lua:
--   require("plugins/logger")
--
-- Default log path: ~/.local/share/pmusic/plays.log
-- Override before requiring:
--   PMUSIC_LOG = "/tmp/pmusic.log"
--   require("plugins/logger")

local log_file = PMUSIC_LOG or (os.getenv("HOME") .. "/.local/share/pmusic/plays.log")

-- Single-quote a path for safe use in a shell command.
local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

-- Create parent directory if it doesn't exist.
local dir = log_file:match("(.+)/[^/]+$")
if dir then os.execute("mkdir -p " .. shell_quote(dir)) end

pmusic.on_song_change(function(track)
    local f = io.open(log_file, "a")
    if not f then
        pmusic.notify("logger: cannot open " .. log_file)
        return
    end
    f:write(string.format("[%s] %s / %s\n",
        os.date("%Y-%m-%d %H:%M:%S"),
        track.folder,
        track.name))
    f:close()
end)
