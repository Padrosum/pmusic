-- Desktop notification plugin for pmusic
-- Sends a system notification whenever the track changes.
--
-- Usage in init.lua:
--   require("plugins/notify-send")
--
-- Requires one of:
--   Linux  : notify-send  (libnotify) or dunstify (dunst)
--   macOS  : osascript    (built-in)
--
-- Override the notifier before requiring:
--   PMUSIC_NOTIFIER = "dunstify"
--   require("plugins/notify-send")

-- Single-quote a string for safe use in a shell command.
local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

-- Detect platform and pick a default notifier.
local notifier = PMUSIC_NOTIFIER

if not notifier then
    -- Prefer dunstify when dunst is running; fall back to notify-send.
    if os.execute("command -v dunstify >/dev/null 2>&1") == 0 then
        notifier = "dunstify"
    elseif os.execute("command -v notify-send >/dev/null 2>&1") == 0 then
        notifier = "notify-send"
    elseif os.execute("command -v osascript >/dev/null 2>&1") == 0 then
        notifier = "osascript"
    end
end

local function send(title, body)
    if not notifier then return end

    local cmd
    if notifier == "osascript" then
        -- macOS
        cmd = string.format(
            "osascript -e 'display notification %s with title %s'",
            shell_quote(body), shell_quote(title))
    else
        -- notify-send / dunstify  (same flags)
        cmd = string.format(
            "%s -t 4000 --icon=audio-x-generic %s %s",
            notifier, shell_quote(title), shell_quote(body))
    end

    os.execute(cmd .. " &")   -- fire-and-forget; don't block Lua
end

pmusic.on_song_change(function(track)
    send("pmusic", track.folder .. " — " .. track.name)
end)
