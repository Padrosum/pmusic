-- Session play-count stats for pmusic
-- Tracks how many times each song was played in the current session
-- and shows the count in the notification bar.
--
-- Usage in init.lua:
--   require("plugins/stats")

local counts = {}   -- path → integer
local session_total = 0

pmusic.on_song_change(function(track)
    local n = (counts[track.path] or 0) + 1
    counts[track.path] = n
    session_total = session_total + 1

    if n > 1 then
        pmusic.notify(string.format("♻  played %d× this session", n))
    end
end)

-- Bind 'S' to display a session summary notification.
pmusic.register_keymap("S", "toggle_pause")   -- placeholder — replace with a real hook when the API exposes custom actions

-- For now, show total count whenever a song changes (remove if too noisy).
-- pmusic.on_song_change(function()
--     pmusic.notify(string.format("session: %d tracks played", session_total))
-- end)
