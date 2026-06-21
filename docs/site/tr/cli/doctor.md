# `atl doctor`

Platformu tanıla ve güvenle düzeltebileceklerini kendiliğinden onar — ATL'nin her oturumda otomatik çalıştırdığı aynı denetimlerin istek üzerine erişilen yüzü.

## Kullanım

```bash
atl doctor
```

Bayrak yoktur. `atl doctor`, mevcut projeyi (çalışma dizini proje anahtarıdır) ve global katmanı inceler, üç denetimi sırayla çalıştırır, deterministik bir düzeltmeyle onarılabilecek olanı onarır ve her denetim için bir satır yazdırır.

## Ne zaman kullanılır

Çoğu zaman kullanmazsın — bu denetimler zaten her oturum başlangıcında ([`session-start` hook'u](/tr/cli/setup-hooks) aracılığıyla) çalışır, sessizce kendiliğinden onarır ve yalnızca bir şey ters gittiğinde su yüzüne çıkar. `atl doctor`'a bilerek bakmak istediğinde başvur:

- `.claude/` altındaki dosyaları kazara sildikten sonra (ya da bu dosyaların hiç inmediği taze bir checkout'ta) — varlıkların geri gelip gelmediğine bak,
- öğrenimler takılmış gibi hissettirdiğinde — kuyruğun boşaldığını ve döngünün hâlâ tıkırdadığını doğrula,
- bir çalışma oturumundan önce ya da sonra hızlı bir "platform burada sağlıklı mı?" denetimi olarak.

## Neyi denetler

Her satır `STATUS  check-name — detay` biçimindedir; burada `STATUS`, `OK`, `WARN` ya da `FAIL` olur. Çalışma sırasında deterministik bir düzeltme uygulayan bir denetim ` (self-healed)` etiketiyle işaretlenir.

### `asset-integrity` — eksik dosya geri yükleme

Kurulum manifestosu bir sözleşmedir: bu dosyalar bu kapsamda var olmalıdır. `doctor`, her manifestoyu diskte gerçekte bulunanla karşılaştırır — hem proje katmanında (`<project>/.claude`) hem de global katmanda (`~/.claude`) — ve **manifestonun listelediği ama diskte eksik olan her dosyayı geri yükler** — sabitlenmiş kaynağından yeniden çekilir ve sağlama toplamı doğrulanır.

Yalnızca *eksik* dosyalar geri yüklenir. Var olan ama değişmiş bir dosya, bir kullanıcı düzenlemesi (ya da bir öğrenme-döngüsü evrimi) olarak değerlendirilir ve **asla üzerine yazılmaz**. Bir takımı tümüyle kaldırmak için, manifestoyu silen [`atl remove`](/tr/cli/remove) komutunu kullan. Tamamlanamayan bir geri yükleme (örneğin ağ çevrimdışıysa) oturumu engelleyici değil, bir `WARN`'dır.

### `queue-backlog` — öğrenme kuyruğu boşalıyor mu?

Bu proje için öğrenme kuyruğundaki bekleyen öğeleri sayar. Kuyruk boş ya da rahatça küçük olduğunda `OK`; biriken iş eşiği (şu an 50) aştığında ise `WARN` olur ki bu, bir [`/drain`](/tr/cli/learnings) geçişinin yetişemediğine işaret eder. Doctor kuyruğu kendisi **boşaltmaz** — bir kuyruk öğesini bir bilgi tabanına katmak bir LLM gerektirir ve bu da bir becerinin işidir; dolayısıyla doctor biriken işi işlemek yerine *işaret eder*.

### `tick-freshness` — döngü hâlâ çalışıyor mu?

Son kuyruk tıkırtısından (tick) bu yana ne kadar geçtiğine bakar. Öğeler kuyruğa alınmışken tıkırtılar 24 saatten uzun süredir çalışmadıysa (ya da kuyruğa yazılmış ama hiç tıkırtı olmamışsa) `WARN` olur — bu, oturum-içi temponun çalışmadığının bir işaretidir. Aksi durumda `OK` olur ve son tıkırtının ne kadar önce gerçekleştiğini bildirir.

## CLI / Beceri ayrımı

`atl doctor` yalnızca deterministik onarımlar yapar — eksik bir dosyayı yeniden çek, mekanik bir adımı yeniden dene. Bir LLM gerektiren her şey (kuyruğa alınmış bir öğrenimi bilgi tabanına işlemek) tasarım gereği kapsam dışıdır; doctor sayıyı su yüzüne çıkarır ve seni beceriye yönlendirir. İşte bu yüzden büyük bir biriken iş burada bir uyarı olarak görünür ama aslında [`/drain`](/tr/cli/learnings) çalıştırılarak temizlenir.

## Örnekler

Sağlıklı bir proje:

```bash
$ atl doctor
OK    queue-backlog — queue empty
OK    tick-freshness — last tick 3m12s ago
OK    asset-integrity — all installed files present

doctor: all healthy
```

Bir dosya silinmiş ve doctor onu geri yüklemiş, kuyruk ise geride kalmış:

```bash
$ atl doctor
WARN  queue-backlog — 63 pending items — a drain skill should process them
OK    tick-freshness — last tick 8s ago
OK    asset-integrity — restored 1 missing file(s) — `atl remove <handle>/<team>` removes a team for good (self-healed)

doctor: warnings above (not fatal)
```

Çıkış mesajı, en ağır satıra göre `doctor: all healthy`, `doctor: warnings above (not fatal)` ya da `doctor: failures above` olur.

## İlgili

- [`atl learnings`](/tr/cli/learnings) — biriken iş denetiminin raporladığı kuyruğu incele; temizlemek için `/drain` çalıştır.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — bu aynı denetimleri otomatik çalıştıran `session-start` hook'unu bağlar.
- [`atl install`](/tr/cli/install) / [`atl remove`](/tr/cli/remove) — `asset-integrity`'nin onardığı manifestoları yazar ve siler.
- [CLI genel bakışı](/tr/cli/overview)
