-- yt-dlp plugin for pmusic
-- Downloads audio from YouTube using the current track name as a search query.
--
-- Usage in init.lua:
--   require("plugins/yt-dlp")
--
-- Requires: yt-dlp  (https://github.com/yt-dlp/yt-dlp)
--
-- Default key: Y  (Shift+y)
-- Override the key before requiring:
--   PMUSIC_YTDLP_KEY = "D"
--   require("plugins/yt-dlp")
--
-- Downloaded files are saved as MP3 to your music directory.
-- pmusic's filesystem watcher picks them up automatically — no restart needed.
--
-- WARNING: This plugin runs yt-dlp as an external process.
--          Only download content you have the right to obtain.

local bind_key = PMUSIC_YTDLP_KEY or "Y"

-- Remember the current track so we have a search query when Y is pressed.
local current = { name = "", folder = "", path = "" }

pmusic.on_song_change(function(track)
    current = track
end)

local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

pmusic.register_keymap(bind_key, function()
    if current.name == "" then
        pmusic.notify("yt-dlp: no track playing")
        return
    end

    local dir = pmusic.music_dir()
    if not dir or dir == "" then
        pmusic.notify("yt-dlp: music directory not set")
        return
    end

    -- Use "Folder TrackName" as the YouTube search query when the folder name
    -- looks like an artist (non-empty and not a bare path component).
    local query = current.name
    if current.folder ~= "" then
        query = current.folder .. " " .. current.name
    end

    -- Build the yt-dlp command.
    -- Flags:
    --   -x                  extract audio only (no video)
    --   --audio-format mp3  re-encode to MP3
    --   --audio-quality 0   best VBR quality
    --   -o <template>       save to music dir with yt title as filename
    --   "ytsearch1:QUERY"   pick the first YouTube result
    --   >/dev/null 2>&1 &   run in background; suppress all output
    local out_tpl = shell_quote(dir .. "/%(title)s.%(ext)s")
    local cmd = string.format(
        "yt-dlp -x --audio-format mp3 --audio-quality 0 -o %s %s >/dev/null 2>&1 &",
        out_tpl,
        shell_quote("ytsearch1:" .. query)
    )

    os.execute(cmd)
    pmusic.notify("↓ " .. query)
end)
