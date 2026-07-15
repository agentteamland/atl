# Öğrenme işaretçisi yaşam döngüsü

Bilginin bir konuşmadan projenin bilgi tabanına nasıl aktığının uçtan uca resmi. v2 deseni **satır içi işaretçiler → kalıcı kuyruk → otomatik drain → ack** — yazması ucuz, kendiliğinden yakalanan, arka planda kendiliğinden drain edilen, tam olarak bir kez işlenen ve yeniden raporlanması imkânsız.

Kanonik kural [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

## Akışa bir bakış

```
[konuşma ortasında]         Claude bir öğrenme işaretler: kullanıcının gördüğü
                            görünür bir "📝 Öğrenildi: …" satırı, artı boru hattının
                            yakaladığı gizli bir <!-- learning: … --> işaretçisi.
                            Araç çağrısı yok, ek maliyet yok.
        ↓
atl tick                    Bir hook her promptta `atl tick` çalıştırır. Bu projenin
(UserPromptSubmit hook,     transkriptlerinden gizli işaretçileri ayrıştırır ve her
 her tur + oturum başı)     birini kalıcı kuyruğa sokar — tam olarak bir kez, içerik
                            hash'iyle yinelenenler ayıklanarak. Kuyruk sayısını da okur.
        ↓
~/.atl/queue.db             Tek bir gömülü bbolt dosyası, çalışma dizinine göre
                            anahtarlanmış proje başına kovalar. Sunucu yok, daemon yok.
        ↓
[kuyruk boş değil]          tick, Claude'un bağlamına bir OTOMATİK-DRAIN sinyali
                            yazdırır: "N learning(s) pending — auto-drain them now …".
        ↓
[aynı tur, arka plan]       Claude TEK bir arka plan drain alt-ajanı başlatır (çalışan
                            oturumun kimlik doğrulamasını kullanarak). Her öğeyi
                            wiki / journal / ajan bilgi tabanına yönlendirir ve ack'ler.
                            /drain'i kimse elle çalıştırmaz.
        ↓
atl learnings ack <id>      Ack'lenmiş bir öğe kuyruktan SİLİNİR.
        ↓
[döngü kapandı]             İşlenmiş bir öğe gitmiştir — asla yeniden raporlanamaz.
                            İlerletilecek bir durum dosyası yoktur.
```

Bu bölünme bilinçlidir: **yakalama kendiliğinden ve deterministiktir** (işaretçiler → kuyruk, CLI tarafından yapılır) ve **entegrasyon da kendiliğindendir** — hook, kuyruk boş olmadığı her turda sinyal verir ve ajan onu arka planda drain eder ([`/drain`](/tr/skills/drain) yönlendirmesi LLM yarısıdır). Geriye kalan tek insan dokunuş noktası:

- **Kullanıcı**, yalnızca bir drain *yapısal* bir değişiklik önerdiğinde (yeni bir ajan / beceri / kural ya da bir kimlik genişletmesi) bir `AskUserQuestion` kapısını yanıtlar. Wiki / journal / ajan KB'sine yapılan sıradan yazmalar arka planda sessizce gerçekleşir.

Kullanıcının (ya da ajanın) `/drain` çalıştırmayı hatırlaması gereken hiçbir şey yok — o elle atılan adım ortadan kalktı.

## Ne öğrenme anı sayılır?

Şunlardan herhangi biri bir konuşma sırasında olduğunda öğrenme anıdır:

- **Hata düzeltme** — gerçek bir hata yeniden üretildi ve düzeltildi
- **Karar** — alternatifler arasında bir seçim yapıldı (JWT vs oturum, Redis vs memcached, 7 günlük vs 15 günlük yenileme)
- **Desen** — bir yaklaşım temiz ve yeniden kullanılabilir çıktı
- **Anti-desen** — bir şey denendi, başarısız oldu ve nedenini biliyoruz
- **Keşif** — sistem, kütüphane ya da dış servis hakkında apaçık olmayan bir gerçek
- **Sözleşme** — "şu andan itibaren X'i daima / asla yaparız"

Sıradan soru-yanıt, dosya bakışları ve mekanik düzenlemeler öğrenme anı DEĞİLDİR. Her yanıtı işaretçileme.

## İşaretçi biçimi — görünür bir satır + gizli bir işaretçi

Bir öğrenme anı meydana geldiğinde Claude **her ikisini** de yazar: kullanıcının o an ne öğrenildiğini görmesi için görünür bir satır ve hook'un yakaladığı gizli bir HTML yorumu.

```
📝 Öğrenildi: 7-day JWT refresh chosen — we want long sessions; the user logs in ~weekly.
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

- **Görünür satır** (`📝 Öğrenildi: …`) sohbette görüntülenir — kullanıcının, yakalama anında neyin alındığını görme biçimidir. Bu, eski ayrı "işlendi" günlüğünün yerini alır: işaretin kendisi görünürlüktür.
- **Gizli işaretçi** (`<!-- learning: … -->`) görüntülenmiş çıktıda görünmezdir ama hook'un taradığı transkriptte korunur. Aynı olguyu taşır, her zaman **NEDEN'i içererek**.

İşaretçi, **bütün** yakalama biçimidir:

```
<!-- learning: <her zaman NEDEN'i içeren bir-üç cümle> -->
```

Alan yok, şema yok — yalnızca düz metinle olgunun kendisi ve gerekçesi. [`/drain`](/tr/skills/drain) yönlendirmesi yükü okur ve nereye ait olduğunu (bir wiki konusu, bir journal kaydı ya da bir ajanın bilgi tabanı) çıkarsar, içerikten kebab-case bir konu türetir. Daha uzun bir düşünce için çok satırlı kullanım da olur:

```html
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Her zaman NEDEN'i ekle.** Gerekçesi olmayan, altı aylık bir "X seçtik" işe yaramaz. İşaret başına tek öğrenme — ilişkisiz öğrenmeleri tek pakette toplama; her biri kendi satırını + işaretçisini hak eder.

> **v1'den değişti.** Eski işaretçi yapılandırılmış YAML alanları taşıyordu (`topic`, `kind`, `doc-impact`, `body`). v2 bunların hepsini bırakır: yük düz nesirdir ve eskiden alanların kodladığı yönlendirmeyi drain yapar.

### `profile-fact` kanalı

Kuyruk çok kanallıdır. İkinci bir kanal, `profile-fact`, kullanıcı ya da birlikte çalıştığı kişiler hakkındaki kalıcı olguları yakalar — aynı gizli-yorum şekli, `profile-fact:` öneki:

```html
<!-- profile-fact: Prefers TypeScript over JavaScript for all new services. -->
```

Her iki kanal da aynı şekilde auto-drain olur — `atl tick` her biri için sinyali basar ve ajan arka planda bir drain subagent'ı başlatır. `learning` kanalı `/drain` ile (`learning-capture` kuralına göre) drain edilir; `profile-fact` ise profile-team'in `/profile-drain`'i ile (kendi `profile-capture` kuralına göre, takımla birlikte kurulur) drain edilir — yani profile-team kurulu olmayan bir oturum `profile-fact` sinyaline hiç davranmaz.

## Neden satır içi işaretçi, araç çağrısı değil?

Öğrenme başına bir araç çağrısı, jeton maliyetini ikiye katlar ve konuşmayı yavaşlatır. Satır içi işaretçiler, ajanın zaten üretecek olduğu metnin içine gömülüdür. [`atl tick`](/tr/cli/tick) içindeki grep düzeyinde bir geçiş gizli işaretçileri sıfıra yakın maliyetle bulur; AI yoğun olan drain yalnızca kuyrukta madde olduğunda çalışır — sıkıcı oturumlar bedava kalır.

## İşaretçilemeyi ne zaman atla?

- Salt sohbet niteliğindeki turlar (selamlaşma, netleştirme, durum soruları)
- Bir dosyayı okuyup içeriğini özetlemek (karar yok, keşif yok)
- Hiçbir sürpriz olmayan sıradan düzenlemeler
- Aynı oturumda daha önce bir işaretçiyle zaten yakalanmış öğrenmeler (yineleme)

## Adım adım sahne arkası

### 1. `atl tick` işaretçileri yakalar ve sinyal verir

[`atl setup-hooks`](/tr/cli/setup-hooks), [`atl tick`](/tr/cli/tick) komutunu `UserPromptSubmit` hook'una bağlar ve `atl session-start` oturum başında bir geçiş çalıştırır. Her çalıştırmada `tick`:

- bu projenin son tick'ten beri değişen Claude Code transkriptlerini keşfeder,
- assistant metnini çıkarır ve `<!-- learning: ... -->` (ve `<!-- profile-fact: ... -->`) gizli işaretçilerini ayrıştırır,
- **her birini kalıcı kuyruğa tam olarak bir kez sokar** — idempotenlik kuyruğun içerik-hash yineleme ayıklamasından gelir, dolayısıyla aynı metni yeniden drain etmek yeni hiçbir şey eklemez,
- kuyruk sayısını okur ve boş olmadığında **otomatik-drain sinyalini** Claude'un bağlamına yazdırır (kısıtlamasız, dolayısıyla bekleyen iş olan her turda tetiklenir — `--throttle`'ın kapıladığı, daha ağır olan yakalama geçişidir).

`tick` yalnızca **kuyruğa sokar ve sinyal verir**. Asla entegre etmez — bir öğrenmeyi bilgi tabanına katlamak LLM işidir, bu yüzden CLI/Beceri sınırının beceri tarafında kalır.

### 2. Kalıcı kuyruk

Kuyruk, `~/.atl/queue.db` konumundaki tek bir gömülü [bbolt](https://github.com/etcd-io/bbolt) dosyasıdır — sunucu yok, daemon yok. Her projenin kuyruğu o tek dosyada yaşar, çalışma dizinine göre anahtarlanmış proje başına kovalara yalıtılır. [`atl learnings`](/tr/cli/learnings) deterministik okuma/ack yüzeyidir:

```bash
atl learnings status                    # kanal başına bekleyen sayıları (bu proje)
atl learnings peek                      # bekleyen maddeleri listele (insan okunur)
atl learnings peek --channel learning --json   # drain'in tükettiği makine-okunur liste
atl learnings ack <id>                  # bir maddeyi işlenmiş olarak işaretle (sil)
```

### 3. Otomatik-drain sinyali

Kuyruk boş olmadığı her an — oturum başında ve sonraki her promptta — hook, bekleyen sayıyı Claude'un `additionalContext` alanında kısa bir sinyal olarak bildirir:

```
atl: 2 learning(s) pending — auto-drain them now in a background subagent (per the learning-capture rule)
```

Kuyrukta hiçbir şey yokken çıktı boştur (sıfır jeton maliyeti).

### 4. Ajan arka planda otomatik drain eder

Bu sinyali görünce ajan ([learning-capture kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md) uyarınca) **tek bir arka plan drain alt-ajanı başlatır** — kullanıcıdan `/drain` komutu yok, bekleme yok. Alt-ajan çalışan oturumun kimlik doğrulamasını devralır (dolayısıyla ayrı bir başsız (headless) `claude -p` yoktur ve kimlik doğrulama sorunu yaşanmaz) ve drain'i çalıştırır:

1. Bekleyen maddeleri `atl learnings peek --channel learning --json` ile okur (`{id, channel, payload, enqueued_at}`).
2. Her maddeyi yükünün biçimine göre yönlendirir, içerikten kebab-case bir konu türeterek:
   - **Konu biçimli güncel doğru** → wiki sayfası (`<proj>/.atl/wiki/<topic>.md`, yerine yaz/birleştir) + journal
   - **Zaman damgalı anlatı** → yalnızca journal (`<proj>/.atl/journal/<YYYY-MM-DD>.md`, ekle)
   - **Kurulu bir ajan için alan bilgisi** → o ajanın `children/<topic>.md` dosyası + `## Knowledge Base` bölümünü yeniden inşa et + journal
   - **Yapısal** (tekrarlayan iş akışı, kristalleşmiş sözleşme, sahibi ajan olmayan yeni bir alan, bir kimlik genişletmesi) → `AskUserQuestion` ile öner; asla otonom yazma
3. Her yapısal olmayan maddeyi sessizce yazar, ardından **yalnızca yazma başarılı olduktan sonra ack'ler**.
4. Neyin nereye indiğine dair kısa bir özet bildirir.

**Tek-uçuşta:** sinyal kuyruk boşalana dek belirmeyi sürdürür, dolayısıyla ajan aynı anda yalnızca bir drain alt-ajanı başlatır — çalışan biri kuyruğu temizler; ajan ikinciyi üst üste bindirmez. Bir drain başarısız olursa ya da bir tur atlanırsa, maddeler kuyrukta hayatta kalır ve bir sonraki turun sinyali onları yeniden dener, dolayısıyla **hiçbir şey asla kaybolmaz** — en kötü ihtimalle bir öğrenme bir tur sonra entegre edilir.

### 5. ack = sil; döngü yapısal olarak kapanır

`atl learnings ack <id>` maddeyi kuyruktan **siler**. İlerletilecek bir durum dosyası ve sonradan karşılaştırılacak bir şey yoktur — işlenmiş bir işaretçi fiziksel olarak geri dönemez.

v1'in uzun-oturum tekrar-raporlama hata sınıfını yapısal olarak öldüren şey budur: v1'de raporlar, sürekli büyüyen bir transkripti bir JSON durum dosyasına karşı süzerek yeniden taramaktan geliyordu ve süzgeç hatalı tetiklenebiliyordu. v2'de raporlar kuyruktan gelir ve işleme maddeyi kaldırır. Boş bir kuyrukta drain'i yeniden çalıştırmak bir no-op'tur.

## Hook kurulu değilken

Gizli işaretçiler hook olmadan da zararsızdır — HTML yorumlarıdır, görüntülenmiş çıktıda görünmezler, metin olarak etkisizdirler (görünür `📝 Öğrenildi:` satırı yine de kullanıcıya ne öğrenildiğini gösterir). Yakalama alışkanlığı yine de değerlidir.

Otomatik yakalama + otomatik drain için [`atl setup-hooks`](/tr/cli/setup-hooks) çalıştır. Onsuz hiçbir şey kendiliğinden kuyruğa girmez ya da sinyal vermez; bir yakalama geçişini yine de kendin [`atl tick`](/tr/cli/tick) ile zorlayabilir, ardından [`/drain`](/tr/skills/drain) becerisini elle çalıştırabilirsin. İşaretçiler transkriptlerde birikir ve bir `tick` geçişi ne zaman çalışırsa kullanılabilir kalır.

## Tarihçe

Bu akış dört biçimden geçti:

1. **Özgün hâl (`atl` öncesi):** "Claude her oturum sonunda öngörülü biçimde öğrenmeleri kaydetmeli." Claude'un bir düz metin yönergesini hatırlamasına bağlıydı. Güvenilmez.
2. **v1 (transkript taraması + `/save-learnings`):** Satır içi işaretçiler yapılandırılmış YAML alanları taşıyordu; bir `SessionStart` hook'u önceki oturumun transkriptlerini yeniden tarıyor, bir JSON durum dosyasına karşı süzüyor ve işlenmemiş işaretçileri raporluyordu. Süzgece-karşı-yeniden-tarama modeli uzun-oturum tekrar-raporlama hata sınıfının kaynağıydı ve işaretçi şeması yakalamayı bir docs-sync adımına bağlıyordu.
3. **v2 (işaretçi → bbolt kuyruğu → elle `/drain` → ack):** İşaretçi düz nesir oldu; [`atl tick`](/tr/cli/tick) her birini kalıcı bir kuyruğa tam olarak bir kez sokar; tekrar-raporlama hata sınıfı tasarım gereği yok oldu. Ama entegrasyon hâlâ, oturum başı sinyalinden sonra bir insanın `/drain` çalıştırmasını gerektiriyordu.
4. **Mevcut hâl (otomatik-drain + görünür işaretler):** İşaret artık görünür bir satır + gizli bir işaretçidir ve hook, kuyruk boş olmadığı her turda sinyal verir; böylece ajan onu **arka planda kendiliğinden** drain eder — elle `/drain` adımı ortadan kalktı. Kuyruğun kalıcılığı, atlanmış bir drain'i kendini iyileştirir kılar.

## İlgili

- [`atl tick`](/tr/cli/tick) — işaretçileri ayrıştıran, kuyruğa sokan ve otomatik-drain sinyalini yayan oturum içi geçiş.
- [`atl learnings`](/tr/cli/learnings) — kalıcı kuyruğu incele ve drain et (`status` / `peek` / `ack`).
- [`/drain`](/tr/skills/drain) — LLM yarısı: her kuyruktaki öğrenmeyi bilgi tabanına yönlendirir, sonra ack'ler.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — `tick` çalıştıran `UserPromptSubmit` + `SessionStart` hook'larını bağlar.
- [`atl doctor`](/tr/cli/doctor) — aynı bekleyen sayıyı talep üzerine yüzeye çıkarır.
- [Bilgi sistemi](/tr/guide/knowledge-system) — journal ve wiki nerede yaşar.
- [Children + learnings](/tr/guide/children-and-learnings) — ajan / beceri alan bilgisi nereye iner.
- Kanonik kural: [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md).
