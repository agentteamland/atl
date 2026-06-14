# Bilgi sistemi

`atl` kullanan bir projede bilginin nasıl düzenlendiği. İki katman: **Journal** (tarih tabanlı tarihsel kayıt) ve **Wiki** (konu tabanlı güncel doğru). Hepsi bu. İki katman. Üzerine ekleme.

Kanonik kuralın kendisi [`core/rules/knowledge-system.md`](https://github.com/agentteamland/core/blob/main/rules/knowledge-system.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

(`core@1.8.0` sürümünde `memory-system` adından yeniden adlandırıldı; Q4 sonrası gerçekliği yansıtmak için: artık ayrı bir "memory" katmanı yok. İki tarih tabanlı katman — `agent-memory` ve `journal` — [self-updating-learning-loop](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) Q4'te tek bir `journal/` katmanına birleştirildi. Önceki dosya adı yanıltıcıydı.)

## İki katmana bir bakış

| Katman | Konum | Amaç | Güncelleme biçimi |
|---|---|---|---|
| **Journal** | `.atl/journal/{YYYY-MM-DD}_{agent}.md` | Tarih tabanlı tarihsel kayıt. Hem ajan başına öğrenme geçmişi HEM de ajanlar arası sinyaller (Q4'te birleştirildi — pratikte yedekliydiler). | Yalnızca eklemeli |
| **Wiki** | `.atl/wiki/{topic}.md` | Konu tabanlı güncel doğru. ŞU AN doğru olanı yansıtır; eski doğrular eklenmez, değiştirilir. | Yerine yazma / güncelleme |

Farklı paradigmalar, farklı amaçlar:

- **Journal** "zaman içinde ne oldu?" sorusunu yanıtlar (kronolojik anlatı).
- **Wiki** "şu an ne doğru?" sorusunu yanıtlar (konu tabanlı anlık görüntü).

İkisini de okuyabilirsin; birbirini dışlamazlar. Ama farklı yazılırlar.

## Journal — ekle, asla düzenleme

Dosya adı: `{YYYY-MM-DD}_{agent-name}.md`. Aynı tarihte birden çok ajan → birden çok dosya. Aynı ajan gün içinde birkaç kez → tek dosya, alt başlıklarla.

Buraya şunlar girer:

- Olup biteni tarihleyen anlatı: keşifler, kararlar, hata düzeltmeleri, neyin işe yaradığı, neyin yaramadığı.
- Ajanlar arası notlar ("X'e sıradaki dokunan için: …").
- Otomatik oluşturulan üretim listeleri ("bu oturum Y wiki sayfasını ve Z ajan-çocuk dosyasını oluşturdu").
- Kullanıcı onaylı yapısal değişiklikler (yeni beceri / kural / ajan kararları ve reddedilenleri).

Kurallar:

- **Yalnızca eklemeli.** Mevcut kayıtlar düzenlenmez; yenileri sona eklenir.
- **İdempotenlik:** [`/save-learnings`](/tr/skills/save-learnings) bir günlük maddesi yazdığında `(kind + topic + body)` üçlüsünü hashler ve aynı dosyada ya da aynı tarihli `.atl/journal/*.md` dosyalarında zaten bulunan yinelenenleri atlar.
- **Asla silinmez** (tarihsel kayıt).
- **`*.local.md` dosya adı kalıbı `.gitignore` kapsamındadır** — gerçekten özel olan içerik için kullanılır (seyrek).

Journal katmanı, eskiden `.atl/agent-memory/` olan şeyi (ajan başına geçmiş) ARTI özgün journal katmanını (ajanlar arası sinyaller) tek başına kapsar. Self-updating-learning-loop Q4 bunları birleştirdi çünkü pratikte iki katmanın da biçimi aynıydı (tarih + ajan + anlatı) ve sıkça birbirine atıf yapıyorlardı.

## Wiki — yerine yaz, yalnızca güncel doğru

Dosya adı: `{topic}.md` (kebab-case, sayfa başına bir kavram).

Projenin yaşayan bilgi tabanıdır. Journal'ın (tarihsel kayıt) aksine, wiki **güncel doğruyu** yansıtır — bir bilgi değiştiğinde sayfa eklenmez, güncellenir.

Kurallar:

- **Konuya göre düzenli, tarihe göre değil** (kavram başına bir sayfa).
- **`<!-- learning -->` işaretçilerinden güncellenir** ([`/save-learnings`](/tr/skills/save-learnings) yoluyla) ya da doğrudan [`/wiki ingest`](/tr/skills/wiki) ile.
- **Sayfalar ŞU AN doğru olanı yansıtır** — eski bilgi yerine yenisi yazılır.
- **Çapraz başvurulu:** ilgili sayfalar birbirine bağ verir.
- **`index.md` kendiliğinden bakım görür** — içindekiler tablosu olarak.
- **`CLAUDE.md` üst kısmındaki `<!-- wiki:index -->` işaretçi bloğu** konu listesini kendiliğinden derler ([self-updating-learning-loop Q5](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) gereği).
- **İlk kurulum:** `.atl/wiki/` bulunmayan bir projede [`/wiki init`](/tr/skills/wiki) komutuyla iskeleyi kur.
- **Düzenli denetim** için [`/wiki lint`](/tr/skills/wiki) komutunu çalıştır.

## Ajanın açılış rutini

Her konuşmanın başında ajan şunları okur (geçerli olduğu durumda):

1. **Kendi ajan dosyası** — takımdan, proje-yerel kopya üzerinden. `agent.md`, `children/*.md` frontmatter'ından kendiliğinden derlenmiş bir Knowledge Base bölümüyle birlikte gelir (bkz. [Children + learnings](/tr/guide/children-and-learnings)).
2. **`CLAUDE.md` `<!-- wiki:index -->` bloğu** — kendiliğinden yüklenir; bilgi haritasını sıfır maliyetle verir. Ajanlar `.atl/wiki/` dizinini doğrudan taramak yerine ilgili wiki sayfalarını bu listeden keşfeder.
3. **Yakın tarihli journal kayıtları** — görev önceki çalışmayla örtüşüyorsa `.atl/journal/` dizininden (genellikle son birkaç kayıt yeter).
4. **Projeye özgü kurallar** — varsa `.atl/docs/coding-standards/{app}.md` dosyasından.

Ajan bütün wiki sayfalarını okumaz. Yalnızca dizini okur (kendiliğinden yüklenir) ve görev o alana dokunduğunda ayrıntı sayfasına olan bağı izler. Bu, bağlamı sıkı tutarken keşfedilebilirliği korur.

## Konuşma sonu rutini — `SessionStart` üzerinden otomatik tetikleme

"Oturum sonunda kaydet" anlamı, **bir sonraki oturum başında tara** olarak hayata geçirildi; çünkü Claude Code'un `SessionEnd` hook çıktısı bir sonraki oturumun Claude'una asla ulaşmaz (bkz. [Öğrenme işaretçisi yaşam döngüsü](/tr/guide/learning-marker-lifecycle)). Otomatik tetikleme akışı:

- **Konuşma sırasında:** öğrenme anları gerçekleştikçe `<!-- learning -->` işaretçileri düşür. İşaretçi biçimi ve disiplini için [işaretçi yaşam döngüsü sayfasına](/tr/guide/learning-marker-lifecycle) bak.
- **Bir sonraki oturum başında:** [`atl session-start`](/tr/cli/setup-hooks) sarmalayıcısı → [`atl learning-capture --previous-transcripts`](/tr/cli/learning-capture) → çıktı `additionalContext` içinde belirir → sen [`/save-learnings --from-markers --transcripts ...`](/tr/skills/save-learnings) komutunu çalıştırırsın → döngü kapanır.
- **Bir değişiklik kullanıcıya görünür olduğunda:** değişikliğin yapıldığı turun içinde, eşleşen README / dokümantasyon sayfasını da güncelle. Bkz. [`docs-sync` kuralı](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md).

`atl setup-hooks` kurulu değilse de işaretçiler transkriptte birikmeye devam eder ve elle yapılan `/save-learnings` çağrısı için kullanılabilir kalır. Hook akışı otomasyondur; işaretçi disiplini ve elle yapılan çağrı temeldir.

## Neden iki katman, üç değil?

Bu kuralın daha eski sürümleri üç katman tanımlıyordu: **memory** (proje başına, ajan başına, yalnızca eklemeli geçmiş), **journal** (proje başına, ajanlar arası sinyaller, yalnızca eklemeli) ve **wiki** (proje başına, konu tabanlı, yerine yazma / güncelleme).

İlk ikisi de tarih tabanlı, yalnızca eklemeli ve anlatı biçimliydi. Her çalışma alanında birbirine atıfta bulunarak ya da aynı olayları yedekli olarak yakalayarak son buluyorlardı. "Ajanın kendine özel hafızası vs. başkalarına yayın" ayrımı asla zorlanmadı — herkes iki katmanı da okuyabiliyordu.

Self-updating-learning-loop Q4 bunları birleştirdi çünkü:

- Aynı biçim → anlamsal ayrım yok.
- Aynı kitle (tüm ajanlar her ikisini de okur).
- Aynı yazma deseni (tarihe göre eklemeli).
- Bölünme, farklı içerik üretmeden zihinsel yük getiriyordu ("bu benim için mi yoksa başkaları için mi?").

Birleşen katmanın adı `journal/`. Wiki ayrı kalır çünkü paradigması (konu tabanlı güncel doğru) journal'ınkinden (tarih tabanlı geçmiş) gerçekten farklıdır.

## Takım başına / proje başına yansımalar

Aynı iki-katmanlı sistem hem kullanıcının projesi içinde (`.atl/journal/`, `.atl/wiki/`) hem de projeler arası bilgi için takım deposu tarafında uygulanır:

- **Ajan çocuk dosyaları** (`children/{topic}.md`, takım deposundaki ajan dizininde) wiki'nin takım tarafındaki karşılığıdır — konu tabanlı, yerine yazma / güncelleme, ajan için projeler arası alan bilgisi.
- **Beceri öğrenim dosyaları** (`learnings/{topic}.md`, takım deposundaki beceri dizininde) beceri başına karşılıktır — aynı biçim, kapsamı beceriye sıkıştırılmış.

Her ikisinde de bir `knowledge-base-summary:` frontmatter alanı bulunur ve bu alan `agent.md` (Knowledge Base bölümü) ya da `skill.md` (Accumulated Learnings bölümü) içine kendiliğinden derlenir. Tüm desen için bkz. [Children + learnings](/tr/guide/children-and-learnings).

## İlgili

- [`/save-learnings`](/tr/skills/save-learnings) — journal kayıtları ve wiki sayfaları yazar.
- [`/wiki`](/tr/skills/wiki) — wiki bakımı (init / ingest / query / lint).
- [Children + learnings](/tr/guide/children-and-learnings) — bu desenin takım tarafındaki yansıması.
- [Öğrenme işaretçisi yaşam döngüsü](/tr/guide/learning-marker-lifecycle) — bilginin konuşmadan işaretçilere ve oradan journal/wiki'ye nasıl aktığı.
- Kanonik kural: [`core/rules/knowledge-system.md`](https://github.com/agentteamland/core/blob/main/rules/knowledge-system.md).
