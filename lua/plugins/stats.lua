-- Session play-count stats for pmusic
-- Tracks how many times each song was played in the current session
-- and shows the count in the notification bar.
--
-- Usage in init.lua:
--   require("plugins/stats")

local counts = {}  -- path → integer (counts per session, reset on Ctrl+R)

pmusic.on_song_change(function(track)
    local n = (counts[track.path] or 0) + 1
    counts[track.path] = n
    if n > 1 then
        pmusic.notify(string.format("♻  played %d× this session", n))
    end
end)
