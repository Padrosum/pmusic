# pmusic Haberler ve Değişiklik Notları

## 2026-07-16 — Güvenilirlik ve güvenlik sağlamlaştırması

Bu güncelleme mevcut TUI, klavye kısayolları ve Vim-benzeri komut sistemini
korurken player yaşam döngüsü, arka plan olayları, kalıcı veriler, plugin sync
ve dağıtım sürecindeki kritik sorunları giderir.

### Öne çıkanlar

- Player kilit sırası yeniden düzenlendi; parça tamamlanması ile pause/toggle
  işlemlerinin aynı anda gerçekleşmesi halinde oluşabilecek deadlock giderildi.
- Audio backend başlatma işlemi package-level `init()` yerine açık constructor
  akışına taşındı. Audio cihazı başlatılamıyorsa hata artık çağırana ulaşıyor.
- Açılan audio stream ve dosyaların doğal EOF, stop, yeni parçayla değiştirme,
  decode hatası ve uygulama kapanışında tam bir kez kapatılması sağlandı.
- Bütün playback başlangıçları ortak bir typed Bubble Tea mesaj akışında
  birleştirildi. Başarısız playback işlemleri parça adı, dosya yolu ve hata
  nedeni ile kullanıcıya bildiriliyor.
- Bozuk bir parçanın sonsuz auto-advance döngüsü oluşturması engellendi.

### TUI ve olay yönlendirmesi

- Command line, command help ve music search overlay'leri artık yalnız kullanıcı
  input'unun sahipliğini alıyor.
- Overlay açıkken tick, playback completion/failure, Lua reload, library reload
  ve terminal resize mesajları kaybolmuyor.
- Command mode aktifken normal player ve quit kısayollarının tetiklenmemesi
  korunuyor.
- Queue ekranında yalnız gezinmek veya ekranı kapatmak artık diske JSON yazmıyor.
  Queue yalnız gerçek bir ekleme, silme, taşıma, temizleme veya tüketme işleminde
  kaydediliyor.
- Queue kayıt hataları status mesajı olarak gösteriliyor.

### Kalıcı veri güvenliği

- Config, queue, enabled-plugin state ve listening statistics için ortak atomik
  JSON persistence katmanı eklendi.
- Veriler önce geçici bir değere decode ediliyor; yalnız başarılı decode ve
  doğrulama sonrasında canlı state'e aktarılıyor.
- Dosyalar aynı dizindeki geçici dosyaya yazılıyor, sync ediliyor ve atomic
  rename ile değiştiriliyor.
- State/config dosyaları kullanıcıya özel `0600` izinleriyle oluşturuluyor.
- Eksik dosya, bozuk JSON, permission ve genel I/O hataları birbirinden
  ayrılıyor.
- Bozuk JSON artık sessizce boş state olarak kabul edilmiyor ve sağlam dosyanın
  üzerine yazılmıyor; kullanıcıya ilgili dosya yolu gösteriliyor.

### Plugin ve Lua store güvenliği

- Plugin/theme indirmeleri mutable `main` branch yerine immutable repository
  commit'ine sabitlendi.
- Her dosya kurulumdan önce SHA-256 ile doğrulanıyor.
- HTTP client timeout'u, başarılı status kontrolü ve 1 MiB dosya boyutu sınırı
  eklendi.
- İndirilen içerik yalnız doğrulama başarılı olduktan sonra atomik olarak
  kuruluyor. Timeout, partial body, oversize veya hash mismatch durumunda mevcut
  plugin korunuyor.
- Sync çıktısı release ve kaynak URL bilgisini gösteriyor.
- Lua eklentilerinin sandbox olmadığı ve trusted code olarak değerlendirilmesi
  gerektiği `docs/security.md` içinde belgelendi.

### yt-dlp ve URL güvenliği

- URL'ler `net/url` tabanlı ortak doğrulamadan geçiriliyor.
- Yalnız `http` ve `https` scheme'leri ile host içeren URL'ler kabul ediliyor.
- Control character, user-info, boş ve anlamsız URL'ler reddediliyor.
- Positional URL'den önce `--` option terminator ekleniyor.
- Query ve URL tek bir argv öğesi olarak korunuyor; shell kullanılmıyor.
- yt-dlp hata mesajlarında çözümlenen executable yolu gösteriliyor.

### Watcher ve kapanış

- fsnotify error channel artık tüketiliyor ve hatalar TUI status alanına
  aktarılıyor.
- Sonradan oluşturulan iç içe dizin ağaçları recursive olarak izlemeye ekleniyor.
- Player, watcher, Lua engine, search/download context ve listening statistics
  için tek ve idempotent `Model.Close()` yaşam döngüsü oluşturuldu.
- Normal quit, command quit ve program dönüşü aynı cleanup yolunu kullanıyor.

### Build ve dağıtım

- Go module yolu canonical repository kimliğiyle eşleştirildi:
  `github.com/Padrosum/pmusic`.
- README içindeki Go sürümü `go.mod` ile uyumlu hale getirildi.
- `pmusic --version`, `pmusic -v` ve `pmusic version` komutları eklendi.
- Version ve commit değerleri release sırasında `ldflags` ile atanabiliyor.
- Makefile'a `fmt-check`, `test`, `race`, `vet`, `build` ve küçültülmüş
  `release` hedefleri eklendi.
- GitHub Actions üzerinde format, vet, test, race ve build kalite kapıları
  eklendi.
- Build, coverage, profile, trace ve editor çıktıları için `.gitignore` eklendi.
- Derlenmiş `pmusic` binary'si kaynak takibinden çıkarıldı; çalışma kopyası
  silinmedi.
- `-trimpath -ldflags="-s -w"` release binary'si ölçümde 13.091.504 bayttan
  9.220.992 bayta düştü; yaklaşık `%29,6` küçülme sağlandı.

### Test kapsamı

- Player deadlock, init failure, idempotent stop/close ve stream lifecycle
  regresyon testleri eklendi.
- Playback başarı/hata mesajları ve bütün playback origin'leri test edildi.
- Overlay × global-message davranış matrisi eklendi.
- Queue persistence, atomik JSON persistence ve corruption davranışları test
  edildi.
- Plugin sync için hash mismatch, timeout, oversize, partial body, non-2xx ve
  başarılı atomik kurulum senaryoları eklendi.
- URL/yt-dlp argüman güvenliği ve recursive watcher davranışı test edildi.
- Tam test paketi normal ve race detector altında; ayrıca `go vet` ve
  `go build ./...` ile doğrulandı.

### Bilinen sınırlamalar

- Gerçek ALSA/audio-device smoke testi geliştirme ortamında uygun ses cihazı
  bulunmadığı için çalıştırılmadı; player testleri kontrollü fake backend ile
  gerçekleştirildi.
- Plugin manifest hash'leri içerik bütünlüğü sağlar ancak bağımsız bir
  kriptografik imza ve key-distribution sistemi değildir.
- macOS ve Windows audio/process davranışı bu Linux doğrulamasında çalıştırılmadı.
- Büyük müzik kitaplıkları için scan/search benchmark ve async index çalışması
  bu değişiklik setine dahil edilmedi.
