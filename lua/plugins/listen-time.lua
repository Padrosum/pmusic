-- Session listen-time tracker for pmusic
-- Measures actual listening time (excluding paused periods) and displays
-- a running total via pmusic.notify() whenever the track changes.
--
-- Usage in init.lua:
--   require("plugins/listen-time")
--
-- The counter resets each time init.lua is reloaded (Ctrl+R).

local session_start = os.time()
local pause_start   = nil
local total_paused  = 0

pmusic.on_state_change(function(state)
    if state == "paused" then
        pause_start = os.time()
    elseif state == "playing" and pause_start then
        total_paused = total_paused + (os.time() - pause_start)
        pause_start  = nil
    elseif state == "stopped" and pause_start then
        -- Stopped while paused — stop accumulating pause time.
        total_paused = total_paused + (os.time() - pause_start)
        pause_start  = nil
    end
end)

pmusic.on_song_change(function(_)
    local elapsed = os.time() - session_start - total_paused
    if elapsed < 0 then elapsed = 0 end
    local h = math.floor(elapsed / 3600)
    local m = math.floor((elapsed % 3600) / 60)
    local s = elapsed % 60
    if h > 0 then
        pmusic.notify(string.format("Listened: %d h %02d m", h, m))
    elseif m > 0 then
        pmusic.notify(string.format("Listened: %d m %02d s", m, s))
    else
        pmusic.notify(string.format("Listened: %d s", s))
    end
end)
