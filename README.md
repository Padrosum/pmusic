# pmusic

Go ile yazılmış, terminal tabanlı (TUI) yerel müzik oynatıcı.

```
┌── Folders ──────────────┬── Jazz ────────────────────────────────────┐
│  Classic Rock           │    1.  ▶ Kind of Blue - Miles Davis        │
│  Electronic             │    2.    So What                           │
│> Jazz                   │    3.    Freddie Freeloader               │
│  Lo-fi                  │    4.    Blue in Green                     │
└─────────────────────────┴────────────────────────────────────────────┘
  ▶ Kind of Blue - Miles Davis ↺                           2:14 / 9:22
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━────────────────────────────────────────
  j/k:move  h/l:panel  enter:play  spc:pause  n/p:next/prev  r:loop  q:quit
```

## Özellikler

- **İki panelli arayüz** — Sol tarafta klasörler, sağ tarafta parçalar
- **MP3, FLAC ve WAV** formatları desteklenir
- **İlerleme çubuğu** ile geçen süre / toplam süre gösterimi
- **Döngü modu** — mevcut parçayı tekrarlar
- **Otomatik geçiş** — parça bitince sıradaki parça başlar
- **Canlı dizin izleme** — müzik klasörüne dosya eklenince liste kendiliğinden güncellenir
- **Yapılandırma kalıcılığı** — seçilen dizin kaydedilir, bir daha sorulmaz

## Kurulum

### ppd ile (önerilen)

```sh
ppd install pmusic
```

> ppd: https://github.com/Padrosum/ppd

### Go ile

```sh
go install github.com/Padrosum/pmusic@latest
```

### Kaynaktan derleme

```sh
git clone https://github.com/Padrosum/pmusic
cd pmusic
go build -o pmusic .
```

## Kullanım

```sh
# İlk çalıştırmada müzik dizini sorulur ve kaydedilir
pmusic

# Doğrudan dizin belirtmek için
pmusic ~/Music
```

İlk açılışta bir kurulum ekranı gelir; müzik klasörünüzün yolunu girin. Bu ayar `~/.config/pmusic/config.json` dosyasına kaydedilir ve bir daha sorulmaz.

## Klavye Kısayolları

| Tuş | İşlev |
|-----|-------|
| `j` / `↓` | Aşağı git |
| `k` / `↑` | Yukarı git |
| `h` / `←` | Klasörler paneline geç |
| `l` / `→` | Parçalar paneline geç |
| `Enter` | Seçili parçayı oynat |
| `Space` | Duraklat / Devam et |
| `n` | Sonraki parça |
| `p` | Önceki parça |
| `r` | Döngü modunu aç/kapat |
| `q` / `Ctrl+C` | Çıkış |

## Gereksinimler

- Go 1.21+

- Ses çıkışı için sistem ses sürücüsü (ALSA / CoreAudio / DirectSound)

## Neden pmusic?

Terminal'den çıkmadan müzik dinlemek isteyenler için tasarlandı. Grafik arayüz gerektirmez, hafiftir ve Vim benzeri klavye kısayolları ile hızla kullanılabilir. Herhangi bir metadata veritabanı veya harici servis bağlantısı gerekmez — sadece bir müzik dizini yeterlidir.
