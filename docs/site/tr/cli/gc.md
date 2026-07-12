# `atl gc`

Sahipsiz varlıkları geri kazan — **kurulumun geri-alınabilir tersi**. Kurulu varlık dizinleri (`.claude/agents|skills|rules|knowledge|scripts|packs`) altında hiçbir kurulum manifestinin sahiplenmediği dosyaları ve bayatlamış promote çakışma arşivlerini bulur; hiçbir şeyi geri-alınamaz biçimde yok etmeden temizler.

`atl install` / `update` / `promote`, varlıkları `~/.claude` ve `<proj>/.claude` içine yazar ve her birini bir kurulum manifestine kaydeder. Bu sözleşmenin dışına düşeni ise hiçbir şey budamaz — güncellemede upstream'den düşen bir dosya (tasarım gereği diskte bırakılır), bir takım kaldırıldıktan sonra geride kalan bir öğrenme kazancı, ya da elle oluşturduğun bir dizin. Zamanla bunlar birikir. `atl gc`, eksik olan temizlik yarısıdır.

**`doctor` iyileştirir; `gc` budar.** [`atl doctor`](/tr/cli/doctor) *eksik kalan* manifest-listeli dosyaları geri getirir; `atl gc` ise hiçbir manifestin *sahiplenmediği* dosyaları kaldırır. İkisi bilinçli olarak zıttır — ve gc asla geri-alınamaz biçimde silmez.

## Kullanım

```bash
atl gc                          # yalnızca raporla — hiçbir şeye dokunmayan kuru çalışma (varsayılan)
atl gc --apply                  # sahipsizleri ~/.atl/gc-trash içine yumuşak-sil (geri alınabilir)
atl gc --apply --include-gains  # kurulu birimlerin yanındaki kazançları da geri kazan (opt-in)
atl gc --undo                   # en son yumuşak-silme grubunu geri yükle
atl gc --purge                  # süresi dolmuş çöp gruplarını kalıcı sil — tek gerçek silme
```

**Kazançlar varsayılan olarak korunur.** Kurulu bir birimin yanında duran ama hiçbir manifestin listelemediği bir dosya — bir `/drain`'in büyüttüğü bir `children/` öğrenmesi ya da elle bir düzenleme — düz bir `atl gc --apply` ile *asla* süpürülmez; raporlanır ve tutulur, böylece otomatik gc farkındalık geçişi birikmiş öğrenmeyi asla silemez. Bunların da geri kazanılmasını bilinçli olarak istediğinde `--include-gains` geç. [`atl pin`](/tr/cli/pin) ettiğin dosyalar sahipli sayılır ve hiç işaretlenmez.

## Neyler sahipsiz sayılır

`atl gc` her iki katmanı da (`~/.claude` global ve `<proj>/.claude` proje) gezer, her varlık dosyasını o katmanın kurulum manifestlerine karşı sorgular ve **hiçbir manifestin sahiplenmediğini** işaretler. Her biri tahmini bir kaynakla etiketlenir — bir ipucu, asla kesinlik değil:

| Kaynak | Genellikle ne demek |
|---|---|
| *kurulu bir birimin yanında kazanç ya da düzenleme* | Kurulu bir agent/skill altında, manifestte olmayan bir dosya (ör. bir `children/` öğrenmesi) — çoğunlukla bir öğrenme kazancı, bazen elle düzenleme. **`--include-gains` geçmedikçe `--apply` tarafından korunur.** |
| *sahipsiz birim (kaldırılmış bir takım ya da elle yapılmış bir dizin)* | Hiçbir manifestin sahiplenmediği bütün bir `agents/x` veya `skills/x` dizini — geride dosya bırakarak kaldırılmış bir takım ya da senin kendi ATL-dışı Claude Code varlıkların. |
| *süresi dolmuş çakışma arşivi* | `~/.atl/history/` altında 30 günden eski bir promote çakışma arşivi (bunlar içerik-adresli ve başka türlü hiç budanmaz). |

"Hiçbir manifest sahiplenmiyor" senin kendi ATL-dışı varlıklarına da uyduğu için, gc **varsayılan olarak kuru-çalışmadır** ve asla geri-alınamaz biçimde silmez — herhangi bir şey taşınmadan önce listeyi daima görürsün.

## Geri-alınabilir güvenlik modeli

Silme, ATL'nin sessizce otomatik olamayacağı tek yerdir; bu yüzden gc işlemi manuel yapmak yerine **geri-alınabilir** yapar:

1. **`atl gc`** (varsayılan) — sahipsizleri kapsam, kaynak ve boyuta göre raporlar. Hiçbir şeye dokunmaz.
2. **`atl gc --apply`** — **yumuşak-silme**: her *sahipsiz* öğeyi `~/.atl/gc-trash/` altında tarih damgalı bir gruba taşır ve bir geri-al manifesti yazar. Kurulu birimlerin yanındaki kazançlar korunur (yukarıya bak); hiçbir şey yok edilmez.
3. **`atl gc --apply --include-gains`** — bilinçli olarak istediğinde korunan kazançları da yumuşak-siler.
4. **`atl gc --undo`** — en son grubu orijinal yollarına geri yükler.
5. **`atl gc --purge`** — tek gerçek silme: 30 günden eski çöp gruplarını kalıcı kaldırır.

Yani hiçbir adımda geri-alınamaz veri kaybı yoktur. Eylem manuel kalır (sen `atl gc` çalıştırırsın), ama farkındalık otomatiktir: bir oturum-başı notu yüksek-sinyalli sahipsizleri yüzeye çıkarır (`atl: N orphaned file(s) beside installed units — run atl gc to review`), böylece kontrol etmeyi hatırlamak zorunda kalmazsın.

## İlgili

- [`atl doctor`](/tr/cli/doctor) — iyileştirme yarısı: eksik kalan manifest-listeli dosyaları geri getirir (gc budama yarısıdır)
- [`atl remove`](/tr/cli/remove) — bir takımın manifest-listeli dosyalarını kaldırır; gc, remove'un geride bıraktığını yakalar
- [`atl promote`](/tr/cli/promote) — gc'nin 30 gün sonra süresini dolduracağı çakışma arşivlerini yazar
