# `atl setup-hooks`

Takımların, global depoların (`core`, `brainstorm`, `rule`, `team-manager`) ve `atl` ikilisinin güncellemeleri kendiliğinden denetlenir hâlde tutulması için Claude Code hook'larını yapılandır — VE bir konuşma sırasında düşürülen satır içi öğrenme işaretçilerinin *bir sonraki* oturumun başında yakalanmasını sağla.

Hepsi senden hiçbir elle iş beklemeden.

`atl` ≥ 1.1.0 gerektirir.

## Kullanım

```bash
atl setup-hooks                    # 30 dakikalık kısıtlamayla kur (önerilen varsayılan)
atl setup-hooks --throttle=5m      # daha sıkı güncelleme denetimi (her 5 dakikalık etkinlikte)
atl setup-hooks --throttle=1h      # daha gevşek güncelleme denetimi
atl setup-hooks --remove           # atl hook'larını kaldır
```

İlk `atl install` çalıştırmasında bunu kendiliğinden açıp açmayacağın sorulur. Evet dersen senin yerine `atl setup-hooks` çalıştırılır. O an reddettiysen ya da sonradan açmak istiyorsan komutu elle çalıştır.

## Ne yapar?

`~/.claude/settings.json` dosyasına iki giriş yazar:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [
          { "type": "command", "command": "atl session-start --silent-if-clean" }
      ]}
    ],
    "UserPromptSubmit": [
      { "hooks": [
          { "type": "command", "command": "atl update --silent-if-clean --throttle=30m" }
      ]}
    ]
  }
}
```

v1.1.0 öncesinde dört giriş kaydeden (`SessionStart` + `UserPromptSubmit` + `SessionEnd` + `PreCompact`) kurulumlardaki eski `SessionEnd` ve `PreCompact` atl girişleri, bir sonraki `atl setup-hooks` çalıştırmasında sessizce kaldırılır. Eski yapı altında komutları çalışmaya devam ediyordu ama çıktıları Claude'a hiç ulaşmıyordu (bkz. [aşağıdaki Tarihçe notu](#history-from-four-hooks-to-two)); silinmesi güvenli ve kayıpsızdır.

Claude Code şunları kendiliğinden çalıştırır:

### `SessionStart` — birleşik açılış zamanı sarmalayıcısı (v1.1.0'da gelen yeni biçim)

Yeni bir Claude Code oturumu açtığında bir kez çalışır. Tek komut olan `atl session-start`, açılış zamanına ait üç görevi sırayla yapar:

1. **Otomatik güncelleme**: `atl update --silent-if-clean` — `~/.claude/repos/agentteamland/` altındaki her önbelleklenmiş depoyu çeker. Güncellemeler Claude'un `additionalContext` alanına `🔄 <team> <oldVer> → <newVer>` biçiminde tek satırlık bir blok olarak yansır.
2. **Önceki transkript işaretçi taraması**: `atl learning-capture --previous-transcripts` — mevcut projenin son başarılı `/save-learnings` çalıştırmasından sonra değişen her transkript dosyasını tarar (durum `~/.claude/state/learning-capture-state.json` içinde tutulur, ilk kullanımda 7 günle sınırlanır). İşaretçi bulunduğunda tek bir `🧠 learning-capture: N unprocessed markers across M transcripts → /save-learnings --from-markers --transcripts ...` bloğu yazdırır.
3. **atl sürüm denetimi**: GitHub Releases API'sine 24 saatte en fazla bir kez sorgu atar. Daha yeni bir `atl` mevcutsa `⬆ atl X.Y.Z → X.Y.Z+1 available` satırını çıkarır.

Hiçbir şey değişmemiş ve işaretçi yoksa çıktı boştur (sıfır jeton maliyeti).

### `UserPromptSubmit` — kısıtlamalı, mesaj başına yenileme

Claude'a gönderdiğin her mesajdan önce çalışır. `<duration>` başına bir kez (varsayılan 30 dakika) kısıtlanmıştır; böylece mesaj başına maliyet tek bir dosya bilgisi denetimine düşer (~1 ms). Yavaş yol (gerçek `git fetch` + `pull`) saatte en fazla iki kez çalışır.

Bir şey değiştiğinde Claude bağlamında aynı `🔄 <team> ...` satırını görür ve bunu kısaca anabilir. Hiçbir şey değişmediyse hiçbir şey görmezsin.

## İşaretçi güdümlü öğrenme işlemesi Claude'a nasıl ulaşır?

Akış uçtan uca otomatiktir; yalnızca tek bir Claude turu için elle adım vardır:

```
[N. oturumu kapatırsın]   işaretçiler transkript dosyasında oturur
        ↓
[N+1. oturumu açarsın]
        ↓
SessionStart hook tetiklenir → atl session-start --silent-if-clean
        ↓
   adım 2: atl learning-capture --previous-transcripts
        → ~/.claude/state/learning-capture-state.json dosyasını okur
        → kesim noktasından sonra değişen proje transkriptlerini sıralar
        → <!-- learning --> blokları için grep ile tarama yapar (yalnızca assistant rolü)
        → `🧠 learning-capture: N markers ... → Run: /save-learnings ...` yazdırır
        ↓
Claude Code, stdout çıktısını Claude'un ilk additionalContext alanına enjekte eder
        ↓
[N+1. oturumda ilk turun]
        ↓
Claude raporu görür, /save-learnings --from-markers --transcripts <paths> komutunu çağırır
        ↓
/save-learnings, işaretçileri journal + wiki + ajan children'ı + beceri learnings'ine
        kalıcılaştırır + durum dosyasındaki lastProcessedAt değerini ilerletir →
        bir sonraki oturum 0 işaretçi görür
```

İşaretçi biçimi ve gürültü filtresinin ayrıntıları için [`atl learning-capture`](/tr/cli/learnings) sayfasına ve davranış belirtimi için core içindeki [learning-capture kuralı](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) ile [docs-sync kuralına](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md) bak.

## Neden iki hook (dört değil)

| Hook | Yanıtladığı soru |
|---|---|
| `SessionStart` (`atl session-start` üzerinden) | "Claude Code'u taze açıyorum; üst akışta ne değişti, önceki oturum geride hangi öğrenmeleri bıraktı, daha yeni bir `atl` var mı?" |
| `UserPromptSubmit` (`atl update` üzerinden) | "Saatlerdir bu oturumdayım; daha yeni bir sürüm var mı?" |

İki hook tüm güvenceyi karşılar. v1.1.0 öncesindeki dört hook'lu tasarım, güncelleme ile öğrenme yakalamayı `SessionStart` / `SessionEnd` / `PreCompact` arasında ayırıyordu; ama `SessionEnd` ve `PreCompact` hook'larının stdout çıktısı Claude'un `additionalContext` alanına ulaştırılmıyordu — bkz. [aşağıdaki Tarihçe notu](#history-from-four-hooks-to-two). Her şeyi birleşik bir sarmalayıcı üzerinden `SessionStart` altına toplamak, tüm davranışı korur ve çıktının gerçekten Claude'a ulaşmasını güvence altına alır.

## İdempotenlik — yeniden çalıştırması güvenlidir

Birleştirme, sahip olduğun diğer hook'ları korur. `atl setup-hooks` yeniden çalıştırıldığında yalnızca `atl`'ye ait girişlere (`atl ` ile başlayan herhangi bir komuta) dokunur. `settings.json` dosyasındaki diğer tüm hook'lar, izinler, model ayarları ve `extraKnownMarketplaces` el değmemiş kalır.

`--remove` benzer biçimde yalnızca `atl`'ye ait hook girişlerini söker; geri kalanı yerinde bırakır. İki komut da önceki kurulumlardan kalma eski `SessionEnd` / `PreCompact` atl girişlerini düşürür.

## Bunu ne zaman çalıştırmalısın?

- Etkileşimli Claude Code kullanıcıları için **her zaman önerilir**.
- CI ya da betikli `atl install` için **önerilmez** (hook'lar CI'da gereksiz yere tetiklenir). İlk kurulumdaki onay sorusu zaten etkileşimsiz bağlamlarda atlanır.

## Tam olarak ne denetlenir?

### Her `atl session-start` çalıştırması

1. **Otomatik güncelleme** (1. adım): `~/.claude/repos/agentteamland/*/` altında dolaşır, paralel `git fetch origin main` çalıştırır, geride kaldıysa ileri-sarmalı çekim yapar, `team.json` dosyasını öncesinden ve sonrasından ayrıştırarak `<oldVer> → <newVer>` çıktısını üretir.
2. **İşaretçi taraması** (2. adım): `~/.claude/state/learning-capture-state.json` dosyasını proje başına `lastProcessedAt` kesim noktası için okur (ilk çalıştırmada son 7 gün), o noktadan sonra değişen transkriptleri sıralar ve yalnızca assistant turlarının ürettiği `<!-- learning -->` bloklarını tarar (v1.1.1 gürültü filtresi düz metindeki anmaları, araç girdi-çıktılarını, özet olaylarını ve kebab-case düzenli ifadesini geçemeyen konuları reddeder). Ağ çağrısı yapılmaz.
3. **atl kendi sürüm denetimi** (3. adım): `api.github.com/repos/agentteamland/cli/releases/latest` adresine tek bir HTTPS GET, 24 saatte bir kısıtlamayla. Daha yeni bir sürüm yayını varsa `⬆` satırını çıkarır.

Yavaş yol: tipik kurulumlarda (5-10 önbelleklenmiş depo) ~2-3 saniye. Hızlı yol (kısıtlama penceresi geçmiş ve transkriptler değişmemişse): ~1 ms.

### Her `atl update --silent-if-clean --throttle=30m` çalıştırması

`atl session-start` komutunun 1. adımıyla aynı; ama son başarılı çalıştırma 30 dakikadan kısa süre önceyse atlanır (`--throttle` ile yapılandırılabilir).

## Çevrimdışı davranış

Çevrimdışıysan her deponun `git fetch` çağrısı sessizce başarısız olur; o depo bir `⚠` uyarı satırı alır (susturulmuş değilse) ve kalanlar sürer. İşaretçi taramasının ağa hiç gereksinimi yoktur — yalnızca yerel dosyaları okur. Claude Code'un istemi olağan biçimde sürer; hook'ların başarısız olması işini engellemez.

## Tarihçe — dört hook'tan iki hook'a {#history-from-four-hooks-to-two}

`atl v0.2.0` (2026-04-24) dört hook ile yayımladı: otomatik güncelleme için `SessionStart` + `UserPromptSubmit`, öğrenme yakalama için `SessionEnd` + `PreCompact`. Yakalama tarafı **çalışıyor gibi görünüyordu** (ikili çalışıyor, işaretçileri tarıyor, raporları yazdırıyordu); ama Claude Code v2.1.x'e göre `SessionEnd` ve `PreCompact` hook stdout çıktısı Claude'un `additionalContext` alanına teslim edilmez. v0.2.0 sonrasındaki ay boyunca bakımcı çalışma alanında 9 oturumda biriken 324 işaretçi **sıfır** otomatik işleme üretti — o dönemdeki gerçek `/save-learnings` çalıştırmalarının tümü kullanıcının elle yaptığı çağrılarla geldi.

`atl v1.1.0` (2026-05-02) akışı yeniden yapılandırdı:

- Yeni `atl session-start` birleşik sarmalayıcısı, güncellemeyi, önceki transkript işaretçi taramasını ve `atl` kendi sürüm denetimini bir araya getiriyor.
- Yeni `atl learning-capture --previous-transcripts` kipi, durum dosyasındaki kesim noktasından sonra değişen transkriptleri okuyor (mevcut oturumun `SessionEnd` hook'unun tetiklenmesine gerek kalmadan).
- `atl setup-hooks` v1.1.0, önceki kurulumlardan kalan eski `SessionEnd` / `PreCompact` atl girişlerini sessizce kaldırıyor. O olaylar altındaki başkalarına ait hook'lara dokunmuyor.

İşaretçi protokolünün kendisi değişmedi — v0.2.0 işaretçi biçimi yine geçerlidir. Yalnızca tetikleme yolu yer değiştirdi.

`atl v1.1.1` (2026-05-02), işaretçi tarayıcısına bir gürültü filtresi ekledi: yalnızca assistant rolü artı kebab-case konu düzenli ifadesi. Bu, `SessionStart` üzerinden yaşanan fazla raporlama hatasını kapattı — işaretçi biçimini *tartışan* her oturum, bir sonraki oturumun sayısını 10-25× şişiriyordu (doğrulama taramasında 5 çalışma alanı transkriptinde 149 ham alt-dize eşleşmesi → 16 gerçek işaretçi).

## İlgili

- [`atl update`](/tr/cli/update) — elle yapılan güncelleme (otomatik güncelleme hook'unun sessizce çağırdığı komut).
- [`atl learning-capture`](/tr/cli/learnings) — elle yapılan tarayıcı (`atl session-start` komutunun sessizce çağırdığı komut).
- [`atl install`](/tr/cli/install) — ilk kurulum (onay sorusunu içerir).
- [CLI'yi kur](/tr/guide/install) — `atl`'yi makinene almak.
