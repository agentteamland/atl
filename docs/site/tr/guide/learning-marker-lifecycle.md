# Öğrenme işaretçisi yaşam döngüsü

Bilginin bir konuşmadan projenin bilgi tabanına nasıl aktığının uçtan uca resmi. v2 deseni **satır içi işaretçiler → kalıcı kuyruk → drain → ack** — yazması ucuz, kendiliğinden yakalanan, tam olarak bir kez işlenen ve yeniden raporlanması imkânsız.

Kanonik kural [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

## Akışa bir bakış

```
[konuşma ortasında]         Claude konuştukça <!-- learning: ... --> işaretçilerini
                            satır içi düşürür. Araç çağrısı yok, ek maliyet yok.
        ↓
atl tick                    Bir hook her promptta (kısıtlanmış) ve oturum başında
(hook-run, birkaç dakikada  `atl tick` çalıştırır. Bu projenin transkriptlerinden
 bir + oturum başında)      işaretçileri ayrıştırır ve her birini kalıcı kuyruğa
                            sokar — içerik hash'iyle yinelenenler ayıklanarak,
                            tam olarak bir kez.
        ↓
~/.atl/queue.db             Tek bir gömülü bbolt dosyası, çalışma dizinine göre
                            anahtarlanmış proje başına kovalar. Sunucu yok, daemon yok.
        ↓
[oturum başında]            SessionStart hook'u bir sayı yüzeye çıkarır:
                            "N öğrenme bekliyor" + bir /drain sinyali.
        ↓
[ilk turun]                 /drain komutunu çağırırsın. Bekleyen maddeleri okur
                            (atl learnings peek --json), her birini wiki / journal /
                            ajan bilgi tabanına yönlendirir ve ack'ler.
        ↓
atl learnings ack <id>      Ack'lenmiş bir madde kuyruktan SİLİNİR.
        ↓
[döngü kapandı]             İşlenmiş bir madde gitmiştir — asla yeniden raporlanamaz.
                            İlerletilecek bir durum dosyası yoktur.
```

Bu bölünme bilinçlidir: **yakalama kendiliğinden ve deterministiktir** (işaretçiler → kuyruk, tam olarak bir kez, CLI tarafından yapılır) ve **entegrasyon LLM yarısıdır** ([`/drain`](/tr/skills/drain) — her öğrenmenin nereye ait olduğuna karar verir). Tek insan dokunuş noktaları şunlardır:

1. **Sen (ajan)**, oturum başındaki "N bekliyor" sinyalini gördükten sonra [`/drain`](/tr/skills/drain) komutunu çağırırsın — tek bir komut.
2. **Kullanıcı**, yalnızca `/drain` *yapısal* bir değişiklik önerdiğinde (yeni bir ajan / beceri / kural ya da bir kimlik genişletmesi) bir `AskUserQuestion` kapısını yanıtlar. Wiki / journal / ajan bilgi tabanına yapılan sıradan yazmalar sessizce gerçekleşir.

## Ne öğrenme anı sayılır?

Şunlardan herhangi biri bir konuşma sırasında olduğunda öğrenme anıdır:

- **Hata düzeltme** — gerçek bir hata yeniden üretildi ve düzeltildi
- **Karar** — alternatifler arasında bir seçim yapıldı (JWT vs oturum, Redis vs memcached, 7 günlük vs 15 günlük yenileme)
- **Desen** — bir yaklaşım temiz ve yeniden kullanılabilir çıktı
- **Anti-desen** — bir şey denendi, başarısız oldu ve nedenini biliyoruz
- **Keşif** — sistem, kütüphane ya da dış servis hakkında apaçık olmayan bir gerçek
- **Sözleşme** — "şu andan itibaren X'i daima / asla yaparız"

Sıradan soru-yanıt, dosya bakışları ve mekanik düzenlemeler öğrenme anı DEĞİLDİR. Her yanıtı işaretçileme.

## İşaretçi biçimi

Bir öğrenme anı meydana geldiğinde yanıt metnine bir HTML yorumu düşür. Görüntülenmiş çıktıda görünmez, hook'un taradığı transkriptte korunur, ~20 jeton:

```html
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

**Bütün** biçim bundan ibarettir:

```
<!-- learning: <her zaman NEDEN'i içeren bir-üç cümle> -->
```

Alan yok, şema yok — yalnızca düz metinle olgunun kendisi ve gerekçesi. [`/drain`](/tr/skills/drain) becerisi yükü okur ve nereye ait olduğunu (bir wiki konusu, bir journal kaydı ya da bir ajanın bilgi tabanı) çıkarsar, içerikten kebab-case bir konu türetir. Daha uzun bir düşünce için çok satırlı kullanım da olur:

```html
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Her zaman NEDEN'i ekle.** Gerekçesi olmayan, altı aylık bir "X seçtik" işe yaramaz. Öğrenme başına tek işaretçi — ilişkisiz öğrenmeleri tek pakette toplama; her biri kendi işaretçisini hak eder.

> **v1'den değişti.** Eski işaretçi yapılandırılmış YAML alanları taşıyordu (`topic`, `kind`, `doc-impact`, `body`). v2 bunların hepsini bırakır: yük düz nesirdir ve eskiden alanların kodladığı yönlendirmeyi `/drain` yapar. `doc-impact` alanı kalktı çünkü v2'de docs-sync adımı yoktur.

### `profile-fact` kanalı

Kuyruk çok kanallıdır. İkinci bir kanal, `profile-fact`, kullanıcı ya da birlikte çalıştığı kişiler hakkındaki kalıcı olguları yakalar — aynı yorum şekli, `profile-fact:` öneki:

```html
<!-- profile-fact: Prefers TypeScript over JavaScript for all new services. -->
```

[`/drain`](/tr/skills/drain) yalnızca `learning` kanalını işler; `profile-fact`, gelecekteki bir birinci-taraf profil takımının kendi drain'ine ayrılmıştır ve burada ele alınmaz.

## Neden satır içi işaretçi, araç çağrısı değil?

Öğrenme başına bir araç çağrısı, jeton maliyetini ikiye katlar ve konuşmayı yavaşlatır. Satır içi işaretçiler, ajanın zaten üretecek olduğu metnin içine gömülüdür. [`atl tick`](/tr/cli/tick) içindeki grep düzeyinde bir geçiş onları sıfıra yakın maliyetle bulur; AI yoğun olan [`/drain`](/tr/skills/drain) işi yalnızca kuyrukta madde olduğunda çalışır — sıkıcı oturumlar bedava kalır.

## İşaretçilemeyi ne zaman atla?

- Salt sohbet niteliğindeki turlar (selamlaşma, netleştirme, durum soruları)
- Bir dosyayı okuyup içeriğini özetlemek (karar yok, keşif yok)
- Hiçbir sürpriz olmayan sıradan düzenlemeler
- Aynı oturumda daha önce bir işaretçiyle zaten yakalanmış öğrenmeler (yineleme)

## Adım adım sahne arkası

### 1. `atl tick` işaretçileri yakalar

[`atl setup-hooks`](/tr/cli/setup-hooks), [`atl tick`](/tr/cli/tick) komutunu `UserPromptSubmit` hook'una bağlar (kısıtlanmış, ör. `--throttle=10m`) ve `atl session-start` oturum başında bir geçiş çalıştırır. Her çalıştırmada `tick`:

- bu projenin son tick'ten beri değişen Claude Code transkriptlerini keşfeder,
- assistant metnini çıkarır ve `<!-- learning: ... -->` (ve `<!-- profile-fact: ... -->`) işaretçilerini ayrıştırır,
- **her birini kalıcı kuyruğa tam olarak bir kez sokar** — idempotenlik kuyruğun içerik-hash yineleme ayıklamasından gelir, dolayısıyla aynı metni yeniden drain etmek yeni hiçbir şey eklemez.

`tick` yalnızca **kuyruğa sokar**. Asla entegre etmez — bir öğrenmeyi bilgi tabanına katlamak LLM işidir, bu yüzden CLI/Beceri sınırının beceri tarafında kalır.

### 2. Kalıcı kuyruk

Kuyruk, `~/.atl/queue.db` konumundaki tek bir gömülü [bbolt](https://github.com/etcd-io/bbolt) dosyasıdır — sunucu yok, daemon yok. Her projenin kuyruğu o tek dosyada yaşar, çalışma dizinine göre anahtarlanmış proje başına kovalara yalıtılır. [`atl learnings`](/tr/cli/learnings) deterministik okuma/ack yüzeyidir:

```bash
atl learnings status                    # kanal başına bekleyen sayıları (bu proje)
atl learnings peek                      # bekleyen maddeleri listele (insan okunur)
atl learnings peek --channel learning --json   # /drain'in tükettiği makine-okunur liste
atl learnings ack <id>                  # bir maddeyi işlenmiş olarak işaretle (sil)
```

### 3. Oturum başlangıcı sayıyı yüzeye çıkarır

Yeni bir oturum açtığında, `SessionStart` hook'u ([`atl session-start`](/tr/cli/setup-hooks)) bir `tick` geçişi çalıştırır ve bekleyen sayıyı — [`atl doctor`](/tr/cli/doctor) komutunun raporladığı sayının aynısını — Claude'un `additionalContext` alanında kısa bir sinyal olarak bildirir:

```
🧠 2 learning(s) pending → run /drain
```

Kuyrukta hiçbir şey yokken çıktı boştur (sıfır jeton maliyeti).

### 4. `/drain` kuyruğu işler

Ajan (sen) sinyali okur ve şunu çağırır:

```
/drain
```

Beceri:

1. Bekleyen maddeleri okumak için `atl learnings peek --channel learning --json` çalıştırır (`{id, channel, payload, enqueued_at}`).
2. Her maddeyi yükünün biçimine göre yönlendirir, içerikten kebab-case bir konu türeterek:
   - **Konu biçimli güncel doğru** → wiki sayfası (`<proj>/.atl/wiki/<topic>.md`, yerine yaz/birleştir) + journal
   - **Zaman damgalı anlatı** → yalnızca journal (`<proj>/.atl/journal/<YYYY-MM-DD>.md`, ekle)
   - **Kurulu bir ajan için alan bilgisi** → o ajanın `children/<topic>.md` dosyası + `## Knowledge Base` bölümünü yeniden inşa et + journal
   - **Yapısal** (tekrarlayan bir iş akışı, kristalleşmiş bir sözleşme, sahibi ajan olmayan yeni bir alan, bir kimlik genişletmesi) → `AskUserQuestion` ile öner; asla otonom yazma
3. Her yapısal olmayan maddeyi sessizce yazar, ardından **yalnızca yazma başarılı olduktan sonra ack'ler**.
4. Yapısal maddeler için onları toplar ve her birini tek bir `AskUserQuestion` üzerinden önerir (reaktif-oluşturma sınırı — yapısal büyümeyi bir insan onaylar).
5. Neyin nereye indiğine dair kısa bir özet bildirir.

### 5. ack = sil; döngü yapısal olarak kapanır

`atl learnings ack <id>` maddeyi kuyruktan **siler**. İlerletilecek bir durum dosyası ve sonradan karşılaştırılacak bir şey yoktur — işlenmiş bir işaretçi fiziksel olarak geri dönemez.

v1'in uzun-oturum tekrar-raporlama hata sınıfını yapısal olarak öldüren şey budur: v1'de raporlar, sürekli büyüyen bir transkripti `~/.claude/state/learning-capture-state.json` dosyasına karşı süzerek yeniden taramaktan geliyordu ve süzgeç hatalı tetiklenebiliyordu. v2'de raporlar kuyruktan gelir ve işleme maddeyi kaldırır. Boş bir kuyrukta `/drain` komutunu yeniden çalıştırmak bir no-op'tur.

`/drain` bir maddeyi entegre edemezse, onu ack'lenmemiş bırakır ve raporda not eder — başarısızlık biçimleri veri kaybettirmez.

## Hook kurulu değilken

İşaretçiler hook olmadan da zararsızdır — HTML yorumlarıdır, görüntülenmiş çıktıda görünmezler, metin olarak etkisizdirler. Yakalama alışkanlığı yine de değerlidir (işaretçiler transkripti okuyan bir insan için bile okunaklıdır).

Otomatik yakalama için [`atl setup-hooks`](/tr/cli/setup-hooks) çalıştır. Onsuz hiçbir şey kendiliğinden kuyruğa girmez; bir yakalama geçişini yine de kendin [`atl tick`](/tr/cli/tick) ile (`--throttle` olmadan) zorlayabilir, ardından [`/drain`](/tr/skills/drain) çalıştırabilirsin. İşaretçiler transkriptlerde birikir ve bir `tick` geçişi ne zaman çalışırsa kullanılabilir kalır.

## Tarihçe

Bu akış üç biçimden geçti:

1. **Özgün hâl (`atl` öncesi):** "Claude her oturum sonunda öngörülü biçimde öğrenmeleri kaydetmeli." Claude'un bir düz metin yönergesini hatırlamasına bağlıydı. Güvenilmez.
2. **v1 (transkript taraması + `/save-learnings`):** Satır içi işaretçiler yapılandırılmış YAML alanları taşıyordu; bir `SessionStart` hook'u önceki oturumun transkriptlerini yeniden tarıyor, bir JSON durum dosyasına karşı süzüyor ve işlenmemiş işaretçileri `/save-learnings` becerisinin işlemesi için raporluyordu. Durum dosyası başarıda ilerletiliyordu. Model çalışıyordu, ama sürekli büyüyen bir transkripti bir süzgece karşı yeniden taramak uzun-oturum tekrar-raporlama hata sınıfının kaynağıydı ve işaretçi şeması (`topic`/`kind`/`doc-impact`/`body`) yakalamayı bir docs-sync adımına bağlıyordu.
3. **Mevcut hâl (v2 — işaretçi → bbolt kuyruğu → `/drain` → ack):** İşaretçi düz nesirdir. [`atl tick`](/tr/cli/tick) her birini kalıcı bir [bbolt](https://github.com/etcd-io/bbolt) kuyruğuna tam olarak bir kez sokar; [`/drain`](/tr/skills/drain) her birini bilgi tabanına katlar ve ack'ler (siler). Transkript yeniden-taraması yok, durum dosyası yok, docs-sync bağlaması yok — ve tekrar-raporlama hata sınıfı tasarım gereği yok oldu.

## İlgili

- [`atl tick`](/tr/cli/tick) — işaretçileri ayrıştıran ve kuyruğa sokan oturum içi geçiş.
- [`atl learnings`](/tr/cli/learnings) — kalıcı kuyruğu incele ve drain et (`status` / `peek` / `ack`).
- [`/drain`](/tr/skills/drain) — LLM yarısı: her kuyruktaki öğrenmeyi bilgi tabanına yönlendirir, sonra ack'ler.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — `tick` çalıştıran `UserPromptSubmit` + `SessionStart` hook'larını bağlar.
- [`atl doctor`](/tr/cli/doctor) — aynı bekleyen sayıyı talep üzerine yüzeye çıkarır.
- [Bilgi sistemi](/tr/guide/knowledge-system) — journal ve wiki nerede yaşar.
- [Children + learnings](/tr/guide/children-and-learnings) — ajan / beceri alan bilgisi nereye iner.
- [Claude Code sözleşmeleri](/tr/guide/claude-code-conventions) — boyunca kullanılan işaretçi blok sözleşmeleri.
- Kanonik kural: [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md).
