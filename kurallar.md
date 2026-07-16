# pmusic Proje Kuralları

## Projenin Amacı

pmusic; kişisel, minimalist ve özgür yazılım felsefesine sadık bir TUI/CLI
müzik oynatıcısıdır. Ana amacı, kullanıcının yerel müzik arşivini ve yerel
playlist'lerini terminalden hızlı, sade ve güvenilir biçimde çalmaktır.

Çevrim içi arama, indirme, eklentiler ve mini oyunlar ikincil özelliklerdir.
Bu özellikler uygulamanın yerel müzik oynatma odağını gölgelememeli, zorunlu
hale gelmemeli veya temel kullanım deneyimini karmaşıklaştırmamalıdır.

## Temel Kurallar

1. **Yerel müzik önce gelir.** Uygulamanın temel işlevleri internet bağlantısı,
   hesap, abonelik veya harici bir servis gerektirmeden çalışmalıdır.

2. **Minimalizm korunmalıdır.** Yeni bir özellik eklenmeden önce gerçekten
   gerekli olup olmadığı, mevcut bir özellik ile çözülebilip çözülemeyeceği ve
   arayüzü gereksiz yere karmaşıklaştırıp karmaşıklaştırmayacağı değerlendirilmelidir.

3. **Özgür yazılım ilkelerine sadık kalınmalıdır.** Kullanıcının verisi ve
   müzik arşivi kullanıcıya aittir. Gizli telemetri, zorunlu bulut hizmeti,
   kullanıcı takibi veya kapalı bir servise bağımlılık eklenmemelidir.

4. **Her değişiklik `news.md` dosyasına yazılmalıdır.** Özellik, hata düzeltmesi,
   güvenlik iyileştirmesi, davranış değişikliği, bağımlılık güncellemesi ve
   dağıtım/CI değişiklikleri aynı değişiklik seti içinde `news.md` dosyasına
   açık ve anlaşılır biçimde eklenmelidir.

5. **Dokümantasyon kodla birlikte güncellenmelidir.** Kurulum, kullanım,
   kısayol, komut veya gereksinimleri etkileyen değişikliklerde `README.md` ve
   ilgili diğer belgeler aynı anda güncellenmelidir.

6. **Mevcut kullanım bozulmamalıdır.** Klavye kısayolları, komutlar, config ve
   kalıcı veri biçimleri değiştirilirken geriye uyumluluk gözetilmelidir. Kırıcı
   bir değişiklik zorunluysa geçiş yolu belgelenmelidir.

7. **Kullanıcı verisi güvenle korunmalıdır.** Config, kuyruk, istatistik ve
   benzeri kalıcı veriler doğrulanmalı ve mümkün olduğunda atomik olarak
   yazılmalıdır. Bozuk veya okunamayan veri sessizce ezilmemelidir.

8. **Temel özellikler isteğe bağlı araçlardan bağımsız olmalıdır.** `yt-dlp`,
   FFmpeg, Lua eklentileri veya çevrim içi servisler bulunmadığında yerel müzik
   oynatma ve kütüphane gezintisi çalışmaya devam etmelidir.

9. **Terminal deneyimi önceliklidir.** Arayüz klavye ile verimli kullanılmalı,
   küçük terminallerde anlaşılır davranmalı ve gereksiz görsel kalabalıktan
   kaçınmalıdır.

10. **Hatalar görünür ve anlaşılır olmalıdır.** Uygulama hataları sessizce
    yutmamalı; kullanıcıya neyin başarısız olduğunu ve mümkünse nasıl
    düzeltilebileceğini kısa bir mesajla bildirmelidir.

11. **Değişiklikler test edilmelidir.** Davranış değişikliklerine uygun testler
    eklenmeli veya mevcut testler güncellenmelidir. Kod gönderilmeden önce en
    azından format, test, vet ve build kontrolleri çalıştırılmalıdır.

12. **Dağıtımlar yeniden üretilebilir olmalıdır.** Release binary'leri sürüm ve
    commit bilgisini taşımalı; otomatik derleme ve doğrulama adımları başarılı
    olmadan yayımlanmamalıdır.

13. **Bağımlılıklar dikkatle seçilmelidir.** Yeni bağımlılıklar yalnız somut bir
    ihtiyacı karşılıyorsa eklenmeli; lisansı, bakım durumu, boyutu ve güvenlik
    etkisi değerlendirilmelidir.

14. **Kapsam kontrolü korunmalıdır.** pmusic genel amaçlı bir medya platformuna
    dönüşmemelidir. Yeni fikirler, yerel müzik dinleme deneyimini doğrudan
    iyileştirmiyorsa isteğe bağlı tutulmalı veya proje kapsamı dışında
    bırakılmalıdır.

## Değişiklik Kontrol Listesi

Her değişiklik tamamlanmadan önce şunlar kontrol edilmelidir:

- Değişiklik projenin yerel-öncelikli ve minimalist amacına uygun mu?
- `news.md` güncellendi mi?
- Kullanıcı davranışı veya kurulum değiştiyse `README.md` güncellendi mi?
- İlgili testler eklendi veya çalıştırıldı mı?
- Build ve release süreci etkileniyorsa workflow'lar doğrulandı mı?
- Yeni bir bağımlılık veya ağ erişimi gerçekten gerekli mi?
- Kullanıcı verisi ve geriye uyumluluk korunuyor mu?
