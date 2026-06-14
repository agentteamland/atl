# Öğrenme işaretçisi yaşam döngüsü

Bilginin bir konuşmadan projenin bilgi tabanına nasıl aktığının uçtan uca resmi. Desen şudur: **satır içi işaretçiler + bir sonraki oturum başında tarama** — yazması ucuz, işlenmesi otomatik.

Kanonik kural [`core/rules/learning-capture.md`](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

## Akışa bir bakış

```
[Oturum N]                                     Claude konuştukça satır içi
                                               <!-- learning --> işaretçileri düşürür.
                                               Araç çağrısı yok, ek maliyet yok.
        ↓
[Oturum N biter]                               İşaretçiler transkriptte oturur.
                                               Oturum sonunda hiçbir hook tetiklenmez.
        ↓
[Oturum N+1 başlar]                            SessionStart hook tetiklenir:
                                               atl session-start --silent-if-clean
        ↓
   adım 3: atl learning-capture                ~/.claude/state/
           --previous-transcripts              learning-capture-state.json dosyasını okur,
                                               kesim noktasından sonra değişen proje
                                               transkriptlerini sıralar,
                                               işaretçi bloklarını grep ile tarar.
        ↓
[Çıktı Claude'a ulaşır]                        🧠 learning-capture: N işaretçi
                                               → Çalıştır: /save-learnings ...
        ↓
[Claude'un ilk turu]                           /save-learnings çağrılır
                                               --from-markers --transcripts ...
        ↓
[/save-learnings kalıcılaştırır]               Journal kaydı, wiki sayfa(ları),
                                               ajan children'ı, beceri learnings'i.
                                               Durum dosyasını ilerletmek için
                                               atl learning-capture
                                               --commit-from-transcripts çağrılır.
        ↓
[Döngü kapandı]                                Bir sonraki SessionStart sıfır
                                               işlenmemiş işaretçi görür.
```

Uçtan uca otomatiktir; yalnızca **iki insan dokunuş noktası** vardır:

1. **Sen (ajan)** `additionalContext` önerisini gördükten sonra `/save-learnings --from-markers --transcripts ...` komutunu çalıştırırsın. Bakımcının tasarımı gereği, bu tek bir komut çağrısıdır — işaretçi başına elle inceleme yoktur.
2. **Kullanıcı**, yeni yapılar (beceri / kural / ajan / kimlik / beceri çekirdek değişikliği) önerildiğinde `AskUserQuestion` onay kapısını yanıtlar. Çalışma başına tek bir çoklu seçim sorusu.

Geri kalan her şey (journal, wiki, children, learnings, dizin yeniden inşaları, durum ilerlemesi) sessizce gerçekleşir.

## Ne öğrenme anı sayılır?

Şunlardan herhangi biri bir konuşma sırasında olduğunda öğrenme anıdır:

- **Hata düzeltme** — gerçek bir hata yeniden üretildi ve düzeltildi.
- **Karar** — alternatifler arasında bir seçim yapıldı (JWT vs oturum, Redis vs memcached, 7 günlük vs 15 günlük yenileme).
- **Desen** — bir yaklaşım temiz ve yeniden kullanılabilir çıktı.
- **Anti-desen** — bir şey denendi, başarısız oldu, nedenini biliyoruz.
- **Keşif** — sistem, kütüphane ya da dış servis hakkında apaçık olmayan bir gerçek.
- **Sözleşme** — "şu andan itibaren X'i daima / asla yapmayız."

Sıradan soru-yanıt, dosya bakışları ve mekanik düzenlemeler öğrenme anı DEĞİLDİR. Her yanıtı işaretçileme.

## İşaretçi biçimi

Bir öğrenme anı meydana geldiğinde yanıt metnine bir HTML yorumu düşür. Görüntülenmiş çıktıda görünmez, hook'un taradığı transkriptte korunur, ~40 jeton:

```html
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7 günlük JWT yenilemesi seçildi çünkü uzun oturum istiyoruz; kullanıcı haftada en fazla bir kez giriş yapar.
-->
```

### Alanlar

| Alan | Zorunlu | Açıklama |
|---|---|---|
| `topic` | ✅ | kebab-case, tek kavram (wiki sayfasının adı olur). Örnek: `auth-refresh`, `redis-ttl`, `build-pipeline`. |
| `kind` | ✅ | Şunlardan biri: `bug-fix \| decision \| pattern \| anti-pattern \| discovery \| convention`. |
| `doc-impact` | ✅ | Şunlardan biri: `none \| readme \| docs \| both \| breaking`. Emin değilsen varsayılan `none`. [docs-sync kuralını](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md) tetikler. |
| `body` | ✅ | Bir-üç cümle. **Daima NEDEN'ini içer.** "X seçtik" diyen, gerekçesi olmayan altı aylık bir not işe yaramaz. |

### Yanıt başına birden çok işaretçi

Birden çok öğrenme aynı yanıtta gerçekleşiyorsa olur. İlişkisiz öğrenmeleri tek bir işaretçide **toplama** — her konu kendi işaretçisini hak eder.

## Neden satır içi işaretçi, araç çağrısı değil?

Öğrenme başına bir araç çağrısı, jeton maliyetini ikiye katlar ve konuşmayı yavaşlatır. Satır içi işaretçiler, ajanın zaten üretmek üzere olduğu metnin içine gömülüdür. Grep düzeyindeki bir hook bunları sıfıra yakın maliyetle bulur; AI yoğun olan `/save-learnings` işi yalnızca işaretçi varsa çalışır — sıkıcı oturumlar bedava kalır.

## İşaretçilemeyi ne zaman atla?

- Salt sohbet niteliğindeki turlar (selamlaşma, netleştirme, durum soruları).
- Bir dosyayı okuyup içeriğini özetlemek (karar yok, keşif yok).
- Hiçbir sürpriz olmayan sıradan düzenlemeler.
- Aynı oturumda yakın bir işaretçi tarafından zaten yakalanmış öğrenmeler (yinelenenleri yazma).

## Adım adım sahne arkası

### 1. SessionStart hook

Hook'u [`atl setup-hooks`](/tr/cli/setup-hooks) komutu kurar:

```
SessionStart → atl session-start --silent-if-clean
```

`atl session-start`, açılış zamanında üç görevi sırayla çalıştırır:

1. **Otomatik güncelleme**: önbelleklenmiş her agentteamland deposunu çek.
2. **Önceki transkript işaretçi taraması**: işlenmemiş öğrenmeleri yüzeye çıkaran adım.
3. **`atl` kendi sürüm denetimi**: GitHub Releases API, 24 saatlik kısıtlamayla.

### 2. İşaretçi taraması

`atl session-start` içindeki işaretçi taraması tam olarak şudur:

```
atl learning-capture --previous-transcripts
```

Yaptığı:

- `~/.claude/state/learning-capture-state.json` dosyasını proje başına `lastProcessedAt` kesim noktası için okur (ilk çalıştırmada son 7 gün).
- O kesim noktasından sonra değişen transkriptleri sıralar.
- Yalnızca **assistant** turları tarafından üretilen `<!-- learning -->` blokları için grep ile tarama yapar (v1.1.1 gürültü filtresi düz metindeki anmaları, araç girdi ve çıktılarını, özet olaylarını ve kebab-case düzenli ifadesini geçemeyen konuları reddeder).
- Durum dosyasının `processedMarkers` kümesine karşı işaretçi başına hash karşılaştırmasıyla yinelenenleri ayıklar (5000 girişle FIFO sınırlı) — bu, `atl v1.1.3` ve `core@1.10.0` sürümlerinde düzeltilen uzun-oturum tekrar-raporlama hatasını kapatır.
- Derli toplu bir rapor yazdırır.

### 3. Çıktı Claude'a ulaşır

`SessionStart` ve `UserPromptSubmit`, stdout çıktısı Claude'un `additionalContext` alanına ulaşan tek Claude Code hook'larıdır ([Claude Code v2.1.x belgeleri](https://docs.claude.com/en/docs/claude-code/hooks) gereği). Önceki v0.2.0 tasarımı işaretçi taraması için `SessionEnd` ve `PreCompact` hook'larını kullanıyordu — ve yaklaşık 7 hafta boyunca sessizce çıktı kaybetti çünkü o olaylar `additionalContext` alanına ulaştırılmaz.

Mevcut yalnızca `SessionStart`'a dayalı tasarım bu boşluğu kapatır. Tüm geçiş öyküsü için bkz. [`atl setup-hooks` Tarihçe notu](/tr/cli/setup-hooks#history-from-four-hooks-to-two).

### 4. `/save-learnings` işler

Ajan (sen) `additionalContext` raporunu okur ve şunu çağırır:

```
/save-learnings --from-markers --transcripts <path1>,<path2>,...
```

Beceri:

1. Listelenen transkriptlerden her `<!-- learning -->` bloğunu çıkarır.
2. `(kind + topic + body)` üçlüsünü hashler ve aynı tarihte journal'da bulunanları atlar.
3. Her öğrenmeyi `kind` ve gövde biçimine göre sınıflandırır.
4. Gerekli yerlere journal + wiki + ajan children'ı + beceri learnings'i yazar.
5. (5 onay kapılı değişiklikten herhangi biri öneriliyorsa) tek bir `AskUserQuestion` içinde toplar.
6. Durum dosyasına işaretçi başına hash kaydetmek için `atl learning-capture --commit-from-transcripts` çağrısı yapar.
7. Tek bir özet bloğu raporlar.

### 5. Durum ilerler; döngü kapanır

6. adımdaki durum dosyası yazımı, döngüyü kapatan adımdır. Bir sonraki `atl session-start` aynı durumu okur ve yeni bir şey olmadığında sıfır işlenmemiş işaretçi görür. Sıkıcı oturumlar bedava kalır.

`/save-learnings` yarı yolda başarısız olursa durum dosyası **güncellenmez** — işaretçiler bir sonraki oturumda yeniden raporlanır ve işleme yeniden denenir. Başarısızlık biçimleri veri kaybettirmez.

## Hook kurulu değilken

İşaretçiler işleme yapacak bir hook olmadan da zararsızdır — HTML yorumlarıdır, görüntülenmiş çıktıda görünmezler, metin olarak etkisizdirler. Yakalama alışkanlığı yine de değerlidir (işaretçiler transkriptin bir insan okuyucusu için bile okunaklıdır).

Otomatik işleme için kullanıcı `atl setup-hooks` komutunu çalıştırır. Bu hook'lar olmadan, kullanıcının oturum sınırlarında `/save-learnings` komutunu elle çağırması gerekir; işaretçiler yine de transkriptlerde birikir ve işleme ne zaman yapılırsa yapılsın kullanılabilir kalır.

## Tarihçe

Bu kural üç biçimden geçti:

1. **Özgün hâl (`atl`'den önceki sürümler):** "Claude her oturum sonunda öngörülü biçimde öğrenmeleri kaydetmeli." Zaman zaman işe yaradı; Claude'un bir düz metin yönergesini hatırlamasına bağlıydı. Güvenilmez.
2. **İlk `atl` sürümü (v0.2.0 — `core@1.3.0`):** Satır içi işaretçiler + `SessionEnd` ve `PreCompact` hook'larına bağlı `atl learning-capture`. **Sessizce kırık** — o olaylar hook stdout çıktısını Claude'un `additionalContext` alanına ulaştırmaz. Bakımcının çalışma alanında 9 oturum boyunca 324 işaretçi, üretimde olduğu ay süresince **sıfır** otomatik işleme üretti. O dönemdeki tüm gerçek `/save-learnings` işi kullanıcının elle yaptığı çağrılarla tetiklendi, hook çıktısıyla değil.
3. **Mevcut hâl (v1.1.0+ — `core@1.8.0`):** Hook, yeni `atl session-start` sarmalayıcısı üzerinden `SessionStart`'a taşındı; önceki oturumun transkriptlerini yeni `--previous-transcripts` kipi üzerinden tarıyor. Çıktı `additionalContext` alanına ulaşıyor. Durum dosyası proje başına `lastProcessedAt` ve işaretçi başına hash'leri tutuyor (sonuncusu, uzun-oturum tekrar-raporlama hatasını düzeltmek için `atl v1.1.3` + `core@1.10.0` sürümlerinde yayımlandı). Döngü kararlı biçimde kapanıyor.

## İlgili

- [`atl learning-capture`](/tr/cli/learning-capture) — CLI tarayıcısı.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — `SessionStart` hook'unu bağlar.
- [`/save-learnings`](/tr/skills/save-learnings) — işaretçileri işler.
- [Bilgi sistemi](/tr/guide/knowledge-system) — journal ve wiki nerede yaşar.
- [Children + learnings](/tr/guide/children-and-learnings) — ajan / beceri alan bilgisi nereye iner.
- [Claude Code sözleşmeleri](/tr/guide/claude-code-conventions) — boyunca kullanılan işaretçi blok sözleşmeleri.
- Kanonik kural: [`core/rules/learning-capture.md`](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md).
