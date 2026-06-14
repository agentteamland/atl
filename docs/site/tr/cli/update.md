# `atl update`

Bir tane veya kurulu / önbelleklenmiş tüm agentteamland depolarının son sürümünü çek **ve** yerel olarak değiştirilmemiş proje-yerel kopyaları kendiliğinden yenile.

## Kullanım

```bash
atl update                          # her önbelleklenmiş depoyu güncelle + değiştirilmemiş kopyaları kendiliğinden yenile
atl update <team>                   # yalnızca tek takımın zincirini güncelle (eski kullanım)
atl update --silent-if-clean        # bir şey değişmediyse çıktı vermez (hook'ların kullandığı)
atl update --check-only             # kuru çalıştırma: ne güncellenirdi raporla, hiçbir şey çekme
atl update --throttle=30m           # son başarılı çalışma 30 dk'dan yakınsa atla
atl update --skip-self-check        # daha yeni atl sürüm yayınını denetleme
atl update -v                       # ayrıntılı (her git komutunu yazdır)
```

## Neyi günceller {#what-it-updates}

Takım adı verilmediğinde `atl update` üç adım uygular:

1. **Önbellek çekimi.** `~/.claude/repos/agentteamland/` altındaki her Git deposunda dolaşır (`core`, `brainstorm`, `rule`, `team-manager` ve kurulu her takım) — `git fetch origin main` → geride kaldıysa ileri-sarmalı `git pull` → güncelse işlem yapmaz.
2. **Sembolik bağdan kopyaya sessiz geçiş (atl ≥ 1.0.0).** Proje başına, `.claude/agents/` ve `.claude/rules/` altındaki global önbelleğe işaret eden v1.0.0 öncesi sembolik bağlar proje-yerel kopyalarla değiştirilir. Proje başına tek bir bilgi satırı olarak görünür; yıkıcı değildir (önce sembolik bağın hedefi okunur, ardından bağ aynı içeriği taşıyan gerçek bir dosyayla değiştirilir).
3. **Değiştirilmemiş kopyaların kendiliğinden yenilenmesi (atl ≥ 1.0.0).** Her kurulu takım için her ajan / kural / beceri kaynağının proje-yerel kopyası üç-yönlü SHA-256 karşılaştırmasıyla denetlenir:
   - **kurulum zamanındaki referans hat** (kurulum sırasında `.team-installs.json` dosyasına işlenmiş).
   - **mevcut proje kopyası** (şu an `.claude/...` içinde olan).
   - **mevcut önbellek içeriği** (az önce çekilmiş olan).

   `mevcut proje = referans hat ≠ önbellek` olduğunda kaynak yerelde değiştirilmemiştir → sessizce yeni önbellek içeriğiyle üzerine yazılır. `mevcut proje ≠ referans hat` olduğunda kullanıcı düzenleme yapmıştır → atlanır, takım başına bir ipucu açık zorla üzerine yazma için `atl install <team> --refresh` komutuna işaret eder.

Ayrıca daha yeni bir `atl` ikili sürüm yayını için denetim yapar (GitHub Releases API'si, 24 saatlik kısıtlamayla):

```
⬆  atl 1.1.1 → 1.1.2 available — run: brew upgrade atl
```

İkili kendiliğinden yükseltilmez — mesaj, `atl`'yi nasıl kurduğuna göre doğru paket yöneticisi komutuna işaret eder.

## Örnek — silent-if-clean (hook'larda kullanılan)

Hiçbir şey değişmemiş:

```bash
$ atl update --silent-if-clean
$                               # sıfır çıktı, çıkış kodu 0
```

Bir şeyler değişmiş:

```bash
$ atl update --silent-if-clean
🔄 software-project-team 1.2.0 → 1.2.1 (auto-updated)
🔄 core 1.8.0 → 1.9.0 (auto-updated)
   ↪ refreshed 14 unmodified copies in current project
```

## Örnek — kuru çalıştırma

```bash
$ atl update --check-only
🔄 software-project-team 1.2.0 → 1.2.1 (auto-updated)
   ↪ would refresh 14 unmodified copies in current project
   ↪ would skip 2 modified copies (run: atl install software-project-team --refresh)
```

Neyin güncellenebileceğini yazdırır; `git pull` çalıştırmaz, hiçbir kopyaya dokunmaz.

## Hook'lar üzerinden kendiliğinden güncelleme (önerilen)

Bir kez ayarla, sonsuza kadar unut:

```bash
atl setup-hooks                 # varsayılan: UserPromptSubmit 30 dakikaya kısıtlanır
atl setup-hooks --throttle=5m   # daha sıkı
atl setup-hooks --remove        # devre dışı bırak
```

Bu komut `~/.claude/settings.json` dosyasına iki Claude Code hook'u ekler:

- `SessionStart` → `atl session-start --silent-if-clean` (birleşik: güncelleme + önceki transkript işaretçi taraması + `atl` kendi sürüm denetimi).
- `UserPromptSubmit` → `atl update --silent-if-clean --throttle=30m` (mesaj başına yenileme, kısıtlamalı).

Claude Code bir oturum başlattığında ya da bir istem gönderdiğinde hook, her önbelleklenmiş depoyu sessizce yeniler ve değiştirilmemiş proje kopyalarını kendiliğinden tazeler. Bir şey değiştiyse Claude tek bir `🔄` satırı görür ve güncellemeye göre davranır. Hiçbir şey değişmediyse hook anında geri döner (~1 ms, yalnızca dosya bilgisi denetimi).

İlk `atl install` çalıştırmasında bunu açmak isteyip istemediğin sorulur. Evet de; istediğin zaman `atl setup-hooks --remove` ile geri alabilirsin.

Ayrıntılar için bkz. [`atl setup-hooks`](/tr/cli/setup-hooks).

## Sürüm kısıtları (takım başına) yine geçerli

`software-project-team@^1.0.0` kurduysan, `atl update` en son `1.x.x` sürümüne kadar çeker — `2.0.0` sürümüne **çekmez**. Ana sürüm artırımları açık biçimde `atl install software-project-team@^2.0.0` çalıştırmanı gerektirir.

## Kısıtlama iç işleyişi

İki zaman damgası dosyası şurada yaşar:

- `~/.claude/cache/atl-last-repo-check` — depo çekimi kısıtlaması.
- `~/.claude/cache/atl-last-self-check` — `atl` sürüm yayını API kısıtlaması (24 saat, sabit).

`--throttle=<dur>` depo çekimi zaman damgasını denetler. Dosyanın değiştirilme zamanı `<dur>` içindeyse depo taraması tamamen atlanır (hızlı yol ~1 ms). Aksi durumda tarama çalışır ve başarıda dosyaya damga vurulur. Başarısız olursa (örneğin çevrimdışı), damga güncellenmez; bir sonraki çağrı yeniden dener.

## Çevrimdışı davranış

Ağ erişilemezse tek tek `git fetch` çağrıları sessizce başarısız olur ve o depo `⚠ <name>: fetch: <error>` olarak raporlanır (`--silent-if-clean` bunu susturmaz — bilmen gerekir). Denetimin geri kalanı sürer. Çevrimdışı çalıştırmalarda proje kopyalarına asla dokunulmaz.

## İlgili

- [`atl install`](/tr/cli/install) — ilk kurulum (kendiliğinden güncelleme onayı sorulur).
- [`atl install <team> --refresh`](/tr/cli/install#idempotency-atl-v100) — bir proje için açık zorla üzerine yazma (yerel değişiklikler nedeniyle kendiliğinden yenileme seni atladığında).
- [`atl setup-hooks`](/tr/cli/setup-hooks) — hook'ları elle yapılandır.
- [`atl list`](/tr/cli/list) — neyin kurulu olduğunu gör.
- [Sürüm kısıtları](/tr/authoring/team-json#version-constraints) — `^`, `~` ve kesin sabitlemelerin nasıl çözüldüğü.
