-- yt-dlp plugin for pmusic
-- Downloads audio from YouTube using the current track as the search query.
--
-- Usage in init.lua:
--   require("plugins/yt-dlp")
--
-- Requires: yt-dlp  (https://github.com/yt-dlp/yt-dlp)
-- Downloaded files are saved to your music directory as MP3.
-- The filesystem watcher picks them up automatically.
--
-- Override the trigger key before requiring this plugin:
--   PMUSIC_YTDLP_KEY = "D"
--   require("plugins/yt-dlp")

local bind_key = PMUSIC_YTDLP_KEY or "Y"

local function ytdlp_available()
    local h = io.popen("yt-dlp --version 2>/dev/null")
    if not h then return false end
    local out = h:read("*l")
    h:close()
    return out ~= nil and out ~= ""
end

local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

pmusic.register_keymap(bind_key, function()
    local track = pmusic.current_track()
    if track.name == "" then
        pmusic.notify("yt-dlp: no track playing")
        return
    end

    local dir = pmusic.music_dir()
    if not dir or dir == "" then
        pmusic.notify("yt-dlp: music directory not set")
        return
    end

    if not ytdlp_available() then
        pmusic.notify("yt-dlp: not found — install yt-dlp and add to PATH")
        return
    end

    local query = track.name
    if track.folder ~= "" then
        query = track.folder .. " " .. track.name
    end

    local out_tpl = shell_quote(dir .. "/%(title)s.%(ext)s")
    local cmd = string.format(
        "yt-dlp -x --audio-format mp3 --audio-quality 0 -o %s %s >/dev/null 2>&1 &",
        out_tpl,
        shell_quote("ytsearch1:" .. query)
    )

    os.execute(cmd)
    pmusic.notify("↓ " .. query)
end)
