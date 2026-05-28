# pmusic Lua Scripting Reference

> **English** · [Türkçe](#türkçe)

---

## Overview

pmusic loads `~/.config/pmusic/lua/init.lua` at startup. From that file you can override the color theme, register custom key bindings, react to playback events with hook functions, and split logic into reusable modules (plugins).

The scripting layer uses **Lua 5.1** (via [gopher-lua](https://github.com/yuin/gopher-lua)). All standard Lua libraries are available: `io`, `os`, `math`, `string`, `table`, `package`.

---

## Directory layout

```
~/.config/pmusic/lua/
├── init.lua          ← entry point — loaded on startup and on Ctrl+R
├── themes/
│   ├── gruvbox.lua
│   ├── catppuccin.lua
│   └── tokyo-night.lua
└── plugins/
    ├── logger.lua
    ├── stats.lua
    └── keymaps.lua
```

`init.lua` is the only file pmusic loads automatically. Everything else is pulled in with `require()` from inside `init.lua`.

---

## Hot-reload

Press **Ctrl+R** inside pmusic at any time to reload the Lua config.

What happens on reload:
1. The current Lua VM is destroyed.
2. All Lua-managed state is reset to defaults: theme → Nord, keymaps → empty, hooks → nil.
3. A fresh VM is created and `init.lua` is executed from scratch.
4. The new theme is applied to the UI immediately.

Errors in `init.lua` (syntax errors, runtime errors) are shown as a status-bar notification. The player keeps running — a bad config never crashes pmusic.

---

## API reference

The entire API lives in the global table `pmusic`. No imports needed.

---

### `pmusic.set_theme(t)`

Overrides UI colors. `t` is a Lua table — any subset of the keys below is accepted; unspecified keys keep their current value.

```lua
pmusic.set_theme({
    accent        = "#88C0D0",
    dim           = "#4C566A",
    selected_bg   = "#434C5E",
    now_playing   = "#A3BE8C",
    border        = "#4C566A",
    border_active = "#88C0D0",
    title         = "#88C0D0",
    status_bg     = "#3B4252",
    panel_bg      = "#2E3440",
    key           = "#81A1C1",
})
```

#### Color key reference

```
┌─[border]──────────────────┬─[border_active]──────────────────────────┐
│  [title] Folders           │  [title] Jazz                            │
│  Classic Rock  [dim]       │    1.  ▶ Kind of Blue  [now_playing]     │
│> Jazz          [selected]  │    2.    So What       [dim]             │
└────────────────────────────┴──────────────────────────────────────────┘
  ▶ Kind of Blue   2:14 / 9:22           ← [accent] or state color
  ━━━━━━━━━━━━━━━━━────────────────────  ← [accent] fill · [dim] empty
  [key]j[dim]:move  [key]q[dim]:quit     ← [status_bg] background
```

| Key | What it colors |
|-----|----------------|
| `accent` | Active panel border · progress bar fill · title text |
| `dim` | Inactive text · empty progress track · hint descriptions |
| `selected_bg` | Background of the cursor row |
| `now_playing` | Name of the track currently playing |
| `border` | Border of the inactive (non-focused) panel |
| `border_active` | Border of the active (focused) panel |
| `title` | Panel header ("Folders", album name) |
| `status_bg` | Background of the bottom status bar |
| `panel_bg` | Background of both panels and the canvas |
| `key` | Key labels in the hint line (`j`, `k`, `q` …) |

Values must be `#RRGGBB` hex strings. ANSI color names (`"red"`, `"blue"` …) are also accepted where the terminal supports them.

---

### `pmusic.get_theme()` → table

Returns the currently active theme as a Lua table with the same keys as `set_theme`. Useful for building themes that modify the existing palette instead of replacing it entirely.

```lua
local t = pmusic.get_theme()
-- make the active border brighter than the inactive one
pmusic.set_theme({ border_active = t.accent })
```

---

### `pmusic.register_keymap(key, action)`

Binds a key string to a built-in action. Bindings are **additive** — core keys (`j`, `k`, `h`, `l`, `q`, …) continue to work.

```lua
pmusic.register_keymap("f",      "next")
pmusic.register_keymap("b",      "prev")
pmusic.register_keymap("ctrl+l", "reload_lua")
```

#### Key string format

The key string is what BubbleTea's `KeyMsg.String()` returns:

| Key | String |
|-----|--------|
| Letters | `"a"`, `"A"`, `"z"` … |
| Space | `" "` |
| Enter | `"enter"` |
| Tab | `"tab"` |
| Escape | `"esc"` |
| Backspace | `"backspace"` |
| Arrows | `"up"`, `"down"`, `"left"`, `"right"` |
| Ctrl combos | `"ctrl+a"` … `"ctrl+z"`, `"ctrl+r"` |
| Function keys | `"f1"` … `"f12"` |

In a terminal, Shift+letter typically sends the uppercase letter (`"A"`, `"B"` …).

#### Available actions

| Action | Effect |
|--------|--------|
| `toggle_pause` | Toggle play/pause |
| `next` | Skip to the next track |
| `prev` | Go back to the previous track |
| `loop` | Toggle loop mode for the current track |
| `focus_folders` | Move focus to the folders panel |
| `focus_tracks` | Move focus to the tracks panel |
| `reload_lua` | Hot-reload `init.lua` (same as Ctrl+R) |
| `quit` | Stop playback and exit |

---

### `pmusic.on_song_change(fn)`

Registers a callback that fires each time a new track starts playing. Only one callback can be active; calling this twice replaces the first.

```lua
pmusic.on_song_change(function(track)
    -- track.name   → "Kind of Blue"
    -- track.folder → "Jazz"
    -- track.path   → "/home/user/Music/Jazz/Kind of Blue.flac"
    pmusic.notify("Now: " .. track.name)
end)
```

The callback receives a table with three string fields:

| Field | Content |
|-------|---------|
| `track.name` | Filename without extension |
| `track.folder` | Name of the containing folder |
| `track.path` | Full absolute file path |

**When it fires:** triggered by `playSelected`, `playNext`, `playPrev`, and `replayCurrent` (loop). It fires as soon as the model dispatches the track, before audio actually starts — typically within one tick (≤ 250 ms).

---

### `pmusic.on_state_change(fn)`

Registers a callback that fires whenever the player transitions between states. Only one callback can be active.

```lua
pmusic.on_state_change(function(state)
    -- state = "playing" | "paused" | "stopped"
    if state == "paused" then
        pmusic.notify("Paused")
    end
end)
```

| State | When |
|-------|------|
| `"playing"` | Playback starts or resumes |
| `"paused"` | User pauses with Space |
| `"stopped"` | Track finishes naturally |

---

### `pmusic.notify(msg)`

Displays a short message in the status bar for approximately 5 seconds. Replaces whatever was showing (now-playing info or a previous notification).

```lua
pmusic.notify("Config loaded.")
```

- Only the **last** call wins if called multiple times before the UI tick (≤ 250 ms).
- Notifications from `init.lua` top-level code appear on the first tick after startup.
- Errors inside hook callbacks automatically produce a notification (the error message is prefixed with `"lua on_song_change:"` or `"lua on_state_change:"`).

---

### `pmusic.config_dir()` → string

Returns the absolute path of the Lua config directory (`~/.config/pmusic/lua/`). Useful for building paths to data files inside a plugin.

```lua
local log = pmusic.config_dir() .. "/plays.log"
local f = io.open(log, "a")
```

---

### `pmusic.version` (string)

The current API version. Check this if a plugin requires a minimum version.

```lua
assert(pmusic.version >= "0.1.0", "pmusic API too old")
```

---

## Writing a theme

A theme file is a plain Lua script that calls `pmusic.set_theme`. No return value is needed.

```lua
-- themes/my-theme.lua
pmusic.set_theme({
    accent        = "#ff79c6",
    dim           = "#44475a",
    selected_bg   = "#44475a",
    now_playing   = "#50fa7b",
    border        = "#44475a",
    border_active = "#ff79c6",
    title         = "#bd93f9",
    status_bg     = "#21222c",
    panel_bg      = "#282a36",
    key           = "#ffb86c",
})
```

Load it from `init.lua`:

```lua
require("themes/my-theme")
```

`require` looks for files relative to `~/.config/pmusic/lua/`, so `require("themes/my-theme")` loads `~/.config/pmusic/lua/themes/my-theme.lua`.

**Tip:** Call `pmusic.set_theme` only with the keys you actually want to change. The rest will keep the current values (Nord defaults, or whatever a previously loaded theme set).

---

## Writing a plugin

A plugin is a Lua module that uses `pmusic.*` to react to events or add behavior.

### Minimal plugin template

```lua
-- plugins/my-plugin.lua

-- Module-local state (persists for the lifetime of the session, reset on Ctrl+R)
local play_count = 0

pmusic.on_song_change(function(track)
    play_count = play_count + 1
    pmusic.notify(string.format("Track #%d: %s", play_count, track.name))
end)
```

### Plugin with file I/O

```lua
-- plugins/my-plugin.lua
local data_file = os.getenv("HOME") .. "/.local/share/pmusic/my-data.txt"

-- Single-quote a path so spaces and special characters are safe in shell commands.
local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

-- Ensure the directory exists before writing.
local dir = data_file:match("(.+)/[^/]+$")
if dir then os.execute("mkdir -p " .. shell_quote(dir)) end

pmusic.on_song_change(function(track)
    local f = io.open(data_file, "a")
    if not f then return end
    f:write(os.date("%F %T") .. " " .. track.name .. "\n")
    f:close()
end)
```

### Important constraints

| Constraint | Reason |
|------------|--------|
| Only **one** `on_song_change` callback is active at a time. | Last call to `on_song_change` wins. If multiple plugins register, chain them manually. |
| Only **one** `on_state_change` callback is active at a time. | Same rule. |
| Hook callbacks **cannot return values** to control pmusic. | Hooks are fire-and-forget. |
| `pmusic.set_theme` and `pmusic.register_keymap` called inside hook callbacks **modify internal state immediately but the UI does not re-render until the next tick (≤ 250 ms)**. | For a live theme switch, trigger `reload_lua` action instead; calling `set_theme` from a hook is valid but unusual. |
| No network libraries are included. | gopher-lua ships without LuaSocket. Use `os.execute("curl ...")` for HTTP if needed. |
| `require` only searches `~/.config/pmusic/lua/`. | Standard `require "socket"` / `require "json"` will fail unless you provide the file. |

### Chaining multiple hooks

If two plugins both need `on_song_change`, compose them in `init.lua`:

```lua
local logger  = require("plugins/logger")   -- returns nothing
local stats   = require("plugins/stats")    -- returns nothing

-- Both registered separately — last one wins. Chain manually instead:
local function my_on_song_change(track)
    -- call logger logic
    local f = io.open(log_path, "a")
    if f then f:write(track.name .. "\n"); f:close() end
    -- call stats logic
    counts[track.path] = (counts[track.path] or 0) + 1
end

pmusic.on_song_change(my_on_song_change)
```

Alternatively, design each plugin to export a function rather than registering the hook directly:

```lua
-- plugins/logger.lua
local M = {}
function M.on_song_change(track)
    -- ...
end
return M
```

```lua
-- init.lua
local logger = require("plugins/logger")
local stats  = require("plugins/stats")

pmusic.on_song_change(function(track)
    logger.on_song_change(track)
    stats.on_song_change(track)
end)
```

---

## Execution model

Understanding when code runs helps avoid subtle bugs.

```
pmusic starts
    └─ Load() called
        ├─ Lua VM created
        ├─ pmusic.* global registered
        ├─ package.path extended to include ~/.config/pmusic/lua/
        └─ init.lua executed top-to-bottom
               ├─ set_theme → stored, applied after Load() returns
               ├─ register_keymap → stored
               ├─ on_song_change(fn) → fn stored, called later
               └─ on_state_change(fn) → fn stored, called later

User presses Ctrl+R
    └─ Load() called again
        ├─ Old VM closed
        ├─ theme reset to Nord defaults
        ├─ keymaps cleared
        ├─ hooks cleared
        └─ init.lua executed from scratch  (same as above)

Song changes / state changes
    └─ CallOnSongChange / CallOnStateChange called (from BubbleTea Update)
        └─ stored fn executed with Protect: true
               └─ Lua error → shown as notify, does NOT crash pmusic
```

---

## Default Nord colors (reference)

| Key | Default value | Nord name |
|-----|---------------|-----------|
| `panel_bg` | `#2E3440` | nord0 |
| `status_bg` | `#3B4252` | nord1 |
| `selected_bg` | `#434C5E` | nord2 |
| `dim` / `border` | `#4C566A` | nord3 |
| `accent` / `border_active` / `title` | `#88C0D0` | nord8 |
| `key` | `#81A1C1` | nord9 |
| `now_playing` | `#A3BE8C` | nord14 |

---

---

# Türkçe

> [English](#pmusic-lua-scripting-reference) · **Türkçe**

---

## Genel Bakış

pmusic başlarken `~/.config/pmusic/lua/init.lua` dosyasını yükler. Bu dosya üzerinden renk temasını değiştirebilir, özel tuş atamaları yapabilir, çalma olaylarına hook fonksiyonlarıyla tepki verebilir ve mantığı tekrar kullanılabilir modüllere (plugin) bölebilirsin.

Scripting katmanı **Lua 5.1** kullanır ([gopher-lua](https://github.com/yuin/gopher-lua) aracılığıyla). Standart Lua kütüphanelerinin tamamı mevcuttur: `io`, `os`, `math`, `string`, `table`, `package`.

---

## Dizin yapısı

```
~/.config/pmusic/lua/
├── init.lua          ← giriş noktası — başlangıçta ve Ctrl+R ile yüklenir
├── themes/
│   ├── gruvbox.lua
│   ├── catppuccin.lua
│   └── tokyo-night.lua
└── plugins/
    ├── logger.lua
    ├── stats.lua
    └── keymaps.lua
```

pmusic yalnızca `init.lua` dosyasını otomatik olarak yükler. Geri kalan her şey `init.lua` içinden `require()` ile çekilir.

---

## Hot-reload (Sıcak Yeniden Yükleme)

pmusic içindeyken istediğin zaman **Ctrl+R** tuşuna bas.

Yeniden yüklemede neler olur:
1. Mevcut Lua VM yok edilir.
2. Lua tarafından yönetilen tüm durum sıfırlanır: tema → Nord, keymap'ler → boş, hook'lar → nil.
3. Yeni bir VM oluşturulur ve `init.lua` sıfırdan çalıştırılır.
4. Yeni tema hemen UI'ya uygulanır.

`init.lua`'daki hatalar (sözdizimi veya çalışma zamanı hataları) status bar'da bildirim olarak gösterilir. Player çalışmaya devam eder — hatalı bir config pmusic'i hiçbir zaman çökertemez.

---

## API Referansı

API'nin tamamı global `pmusic` tablosundadır. Herhangi bir import gerekmez.

---

### `pmusic.set_theme(t)`

UI renklerini override eder. `t` bir Lua tablosudur — aşağıdaki anahtarların herhangi bir alt kümesi kabul edilir; belirtilmeyen anahtarlar mevcut değerini korur.

```lua
pmusic.set_theme({
    accent        = "#88C0D0",
    dim           = "#4C566A",
    selected_bg   = "#434C5E",
    now_playing   = "#A3BE8C",
    border        = "#4C566A",
    border_active = "#88C0D0",
    title         = "#88C0D0",
    status_bg     = "#3B4252",
    panel_bg      = "#2E3440",
    key           = "#81A1C1",
})
```

#### Renk anahtarı referansı

```
┌─[border]──────────────────┬─[border_active]──────────────────────────┐
│  [title] Klasörler         │  [title] Jazz                            │
│  Classic Rock  [dim]       │    1.  ▶ Kind of Blue  [now_playing]     │
│> Jazz          [selected]  │    2.    So What       [dim]             │
└────────────────────────────┴──────────────────────────────────────────┘
  ▶ Kind of Blue   2:14 / 9:22           ← [accent] veya durum rengi
  ━━━━━━━━━━━━━━━━━────────────────────  ← [accent] dolu · [dim] boş
  [key]j[dim]:move  [key]q[dim]:quit     ← [status_bg] arkaplan
```

| Anahtar | Ne renklendirir |
|---------|-----------------|
| `accent` | Aktif panel kenarlığı · ilerleme çubuğu dolumu · başlık metni |
| `dim` | Pasif metin · boş ilerleme çubuğu · ipucu açıklamaları |
| `selected_bg` | İmleç satırının arkaplanı |
| `now_playing` | O an çalan şarkının adı |
| `border` | Pasif (odaklanılmamış) panel kenarlığı |
| `border_active` | Aktif (odaklanılmış) panel kenarlığı |
| `title` | Panel başlığı ("Klasörler", albüm adı) |
| `status_bg` | Alt status bar'ın arkaplanı |
| `panel_bg` | Her iki panelin ve tuvalin arkaplanı |
| `key` | İpucu satırındaki tuş etiketleri (`j`, `k`, `q` …) |

Değerler `#RRGGBB` hex string olmalıdır. Terminal destekliyorsa ANSI renk adları da (`"red"`, `"blue"` …) kabul edilir.

---

### `pmusic.get_theme()` → tablo

Mevcut temayı `set_theme` ile aynı anahtarlara sahip bir Lua tablosu olarak döndürür. Mevcut paleti tamamen değiştirmek yerine üzerine inşa eden temalar yazmak için kullanışlıdır.

```lua
local t = pmusic.get_theme()
pmusic.set_theme({ border_active = t.accent })
```

---

### `pmusic.register_keymap(key, action)`

Bir tuş dizesini yerleşik bir eyleme bağlar. Bağlamalar **eklemeli**dir — temel tuşlar (`j`, `k`, `h`, `l`, `q`, …) çalışmaya devam eder.

```lua
pmusic.register_keymap("f",      "next")
pmusic.register_keymap("b",      "prev")
pmusic.register_keymap("ctrl+l", "reload_lua")
```

#### Tuş dizesi formatı

Tuş dizesi, BubbleTea'nın `KeyMsg.String()` metodunun döndürdüğü şeydir:

| Tuş | Dize |
|-----|------|
| Harfler | `"a"`, `"A"`, `"z"` … |
| Boşluk | `" "` |
| Enter | `"enter"` |
| Tab | `"tab"` |
| Escape | `"esc"` |
| Geri al | `"backspace"` |
| Ok tuşları | `"up"`, `"down"`, `"left"`, `"right"` |
| Ctrl kombinasyonları | `"ctrl+a"` … `"ctrl+z"`, `"ctrl+r"` |
| Fonksiyon tuşları | `"f1"` … `"f12"` |

Terminalde Shift+harf genellikle büyük harf gönderir (`"A"`, `"B"` …).

#### Mevcut eylemler

| Eylem | Etki |
|-------|------|
| `toggle_pause` | Oynat/duraklat'ı değiştir |
| `next` | Sonraki parçaya geç |
| `prev` | Önceki parçaya dön |
| `loop` | Mevcut parça için döngü modunu değiştir |
| `focus_folders` | Odağı klasörler paneline taşı |
| `focus_tracks` | Odağı parçalar paneline taşı |
| `reload_lua` | `init.lua`'yı hot-reload et (Ctrl+R ile aynı) |
| `quit` | Çalmayı durdur ve çık |

---

### `pmusic.on_song_change(fn)`

Her yeni parça çalmaya başladığında tetiklenecek bir callback kaydeder. Yalnızca tek bir callback aktif olabilir; bu fonksiyon iki kez çağrılırsa birincisi silinir.

```lua
pmusic.on_song_change(function(track)
    -- track.name   → "Kind of Blue"
    -- track.folder → "Jazz"
    -- track.path   → "/home/user/Müzik/Jazz/Kind of Blue.flac"
    pmusic.notify("Şimdi: " .. track.name)
end)
```

Callback üç string alanı olan bir tablo alır:

| Alan | İçerik |
|------|--------|
| `track.name` | Uzantısız dosya adı |
| `track.folder` | İçinde bulunduğu klasörün adı |
| `track.path` | Tam mutlak dosya yolu |

**Ne zaman tetiklenir:** `playSelected`, `playNext`, `playPrev` ve `replayCurrent` (döngü) tarafından tetiklenir. Model parçayı dispatch eder etmez ateşlenir, ses gerçekte başlamadan önce — genellikle tek bir tick içinde (≤ 250 ms).

---

### `pmusic.on_state_change(fn)`

Player durumlar arasında geçiş yaptığında tetiklenecek bir callback kaydeder. Yalnızca tek bir callback aktif olabilir.

```lua
pmusic.on_state_change(function(state)
    -- state = "playing" | "paused" | "stopped"
    if state == "paused" then
        pmusic.notify("Duraklatıldı")
    end
end)
```

| Durum | Ne zaman |
|-------|----------|
| `"playing"` | Çalma başlar veya devam eder |
| `"paused"` | Kullanıcı Space ile duraklatır |
| `"stopped"` | Parça doğal olarak biter |

---

### `pmusic.notify(msg)`

Status bar'da yaklaşık 5 saniye boyunca kısa bir mesaj gösterir. O anda gösterilen her şeyin (çalan şarkı bilgisi veya önceki bildirim) yerini alır.

```lua
pmusic.notify("Ayarlar yüklendi.")
```

- UI tick'inden önce (≤ 250 ms) birden fazla kez çağrılırsa yalnızca **son** çağrı geçerli olur.
- `init.lua` üst düzey kodundan gelen bildirimler başlangıçtan sonraki ilk tick'te görünür.
- Hook callback'lerindeki hatalar otomatik olarak bildirim üretir (hata mesajı `"lua on_song_change:"` veya `"lua on_state_change:"` önekiyle gelir).

---

### `pmusic.config_dir()` → string

Lua config dizininin mutlak yolunu döndürür (`~/.config/pmusic/lua/`). Bir plugin içinde veri dosyalarına yollar oluşturmak için kullanışlıdır.

```lua
local log = pmusic.config_dir() .. "/plays.log"
local f = io.open(log, "a")
```

---

### `pmusic.version` (string)

Mevcut API sürümü. Bir plugin minimum sürüm gerektiriyorsa kontrol et.

```lua
assert(pmusic.version >= "0.1.0", "pmusic API çok eski")
```

---

## Tema Yazma

Tema dosyası, `pmusic.set_theme` çağıran düz bir Lua scriptidir. Dönüş değeri gerekmez.

```lua
-- themes/kendi-temam.lua
pmusic.set_theme({
    accent        = "#ff79c6",
    dim           = "#44475a",
    selected_bg   = "#44475a",
    now_playing   = "#50fa7b",
    border        = "#44475a",
    border_active = "#ff79c6",
    title         = "#bd93f9",
    status_bg     = "#21222c",
    panel_bg      = "#282a36",
    key           = "#ffb86c",
})
```

`init.lua`'dan yükle:

```lua
require("themes/kendi-temam")
```

`require`, `~/.config/pmusic/lua/`'ya göre dosya arar; yani `require("themes/kendi-temam")`, `~/.config/pmusic/lua/themes/kendi-temam.lua` dosyasını yükler.

**İpucu:** `pmusic.set_theme`'i yalnızca değiştirmek istediğin anahtarlarla çağır. Geri kalanlar mevcut değerlerini korur (Nord varsayılanları veya önceden yüklenmiş bir temanın belirlediği değerler).

---

## Plugin Yazma

Plugin, olaylara tepki vermek veya davranış eklemek için `pmusic.*` kullanan bir Lua modülüdür.

### Minimal plugin şablonu

```lua
-- plugins/kendi-pluginim.lua

-- Modül-yerel durum (oturum boyunca kalır, Ctrl+R ile sıfırlanır)
local calis_sayisi = 0

pmusic.on_song_change(function(track)
    calis_sayisi = calis_sayisi + 1
    pmusic.notify(string.format("#%d: %s", calis_sayisi, track.name))
end)
```

### Dosya I/O kullanan plugin

```lua
-- plugins/kendi-pluginim.lua
local veri_dosyasi = os.getenv("HOME") .. "/.local/share/pmusic/verim.txt"

-- Kabuk komutlarında güvenli kullanım için path'i single-quote ile koru.
local function shell_quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

-- Yazmadan önce dizinin var olduğundan emin ol.
local dir = veri_dosyasi:match("(.+)/[^/]+$")
if dir then os.execute("mkdir -p " .. shell_quote(dir)) end

pmusic.on_song_change(function(track)
    local f = io.open(veri_dosyasi, "a")
    if not f then return end
    f:write(os.date("%F %T") .. " " .. track.name .. "\n")
    f:close()
end)
```

### Önemli kısıtlamalar

| Kısıtlama | Nedeni |
|-----------|--------|
| Yalnızca **tek** bir `on_song_change` callback'i aktif olabilir. | Son `on_song_change` çağrısı öncekinin yerini alır. Birden fazla plugin kayıt yapmak istiyorsa bunları manuel olarak zincirle. |
| Yalnızca **tek** bir `on_state_change` callback'i aktif olabilir. | Aynı kural. |
| Hook callback'leri pmusic'i kontrol etmek için **değer döndüremez**. | Hook'lar fire-and-forget (çağrı ve unut) şeklinde çalışır. |
| `pmusic.set_theme` ve `pmusic.register_keymap` hook callback'lerinden çağrıldığında **iç durumu anında değiştirir; ancak UI bir sonraki tick'e (≤ 250 ms) kadar yeniden render edilmez**. | Canlı tema geçişi için `reload_lua` eylemini tetikle; hook içinde `set_theme` çağırmak geçerlidir ancak alışılmadık bir kullanımdır. |
| Ağ kütüphanesi dahil değildir. | gopher-lua, LuaSocket olmadan gelir. HTTP için gerekirse `os.execute("curl ...")` kullan. |
| `require` yalnızca `~/.config/pmusic/lua/`'da arar. | `require "socket"` / `require "json"` dosyayı kendin sağlamadıkça başarısız olur. |

### Birden fazla hook'u zincirleme

İki plugin de `on_song_change`'e ihtiyaç duyuyorsa `init.lua`'da birleştir:

```lua
-- Seçenek A: Direkt zincir
local function hepsini_cagir(track)
    -- logger mantığı
    local f = io.open(log_yolu, "a")
    if f then f:write(track.name .. "\n"); f:close() end
    -- stats mantığı
    sayac[track.path] = (sayac[track.path] or 0) + 1
end
pmusic.on_song_change(hepsini_cagir)

-- Seçenek B: Her plugin bir fonksiyon export eder
local logger = require("plugins/logger")   -- { on_song_change = fn }
local stats  = require("plugins/stats")    -- { on_song_change = fn }

pmusic.on_song_change(function(track)
    logger.on_song_change(track)
    stats.on_song_change(track)
end)
```

---

## Çalışma Modeli

Kodun ne zaman çalıştığını anlamak ince hataları önlemeye yardımcı olur.

```
pmusic başlar
    └─ Load() çağrılır
        ├─ Lua VM oluşturulur
        ├─ pmusic.* global'i kaydedilir
        ├─ package.path, ~/.config/pmusic/lua/ içerecek şekilde genişletilir
        └─ init.lua yukarıdan aşağıya çalıştırılır
               ├─ set_theme → saklanır, Load() döndükten sonra uygulanır
               ├─ register_keymap → saklanır
               ├─ on_song_change(fn) → fn saklanır, sonra çağrılır
               └─ on_state_change(fn) → fn saklanır, sonra çağrılır

Kullanıcı Ctrl+R'ye basar
    └─ Load() tekrar çağrılır
        ├─ Eski VM kapatılır
        ├─ Tema Nord varsayılanlarına sıfırlanır
        ├─ Keymap'ler temizlenir
        ├─ Hook'lar temizlenir
        └─ init.lua sıfırdan çalıştırılır

Şarkı değişir / durum değişir
    └─ CallOnSongChange / CallOnStateChange çağrılır (BubbleTea Update'ten)
        └─ Saklanan fn, Protect: true ile çalıştırılır
               └─ Lua hatası → notify olarak gösterilir, pmusic ÇÖKMEZ
```

---

## Varsayılan Nord Renkleri (Referans)

| Anahtar | Varsayılan değer | Nord adı |
|---------|-----------------|----------|
| `panel_bg` | `#2E3440` | nord0 |
| `status_bg` | `#3B4252` | nord1 |
| `selected_bg` | `#434C5E` | nord2 |
| `dim` / `border` | `#4C566A` | nord3 |
| `accent` / `border_active` / `title` | `#88C0D0` | nord8 |
| `key` | `#81A1C1` | nord9 |
| `now_playing` | `#A3BE8C` | nord14 |
