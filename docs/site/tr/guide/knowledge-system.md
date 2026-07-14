# Bilgi sistemi

`atl` kullanan bir projede bilginin nasıl düzenlendiği. İki katman: **journal** (tarih tabanlı tarihsel kayıt) ve **wiki** (konu tabanlı güncel doğru). Hepsi bu. İki katman. Üzerine ekleme.

Kanonik kuralın kendisi [`core/rules/knowledge-system.md`](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

Ayrı bir **memory** katmanı yok. v1'in üçü vardı (agent-memory + journal + wiki); ilk ikisi de tarih tabanlı, yalnızca eklemeli ve pratikte yedekliydi; bu yüzden artık tek bir `journal/` altında birleşmiştir. "memory"nin uzandığı şey, ajanın kendi bilgi tabanı (`children/` dizini) artı kullanıcı-genel katman tarafından karşılanır.

## İki katmana bir bakış

| Katman | Konum | Amaç | Güncelleme biçimi |
|---|---|---|---|
| **Journal** | `.atl/journal/{YYYY-MM-DD}.md` | Tarih tabanlı tarihsel kayıt: ne oldu, neyin işe yaradığı, neyin yaramadığı ve neden. Gün başına bir dosya. | Yalnızca eklemeli |
| **Wiki** | `.atl/wiki/{topic}.md` | Konu tabanlı güncel doğru. ŞU AN doğru olanı yansıtır; eski doğrular eklenmez, değiştirilir. | Yerine yazma / güncelleme |

Farklı paradigmalar, farklı amaçlar:

- **Journal** "zaman içinde ne oldu?" sorusunu yanıtlar (kronolojik anlatı).
- **Wiki** "şu an ne doğru?" sorusunu yanıtlar (konu tabanlı anlık görüntü).

İkisini de okuyabilirsin; birbirini dışlamazlar. Ama farklı yazılırlar.

## Journal — ekle, asla düzenleme

Dosya adı: `{YYYY-MM-DD}.md` — gün başına bir dosya, o gün ne çalıştıysa hepsi tarafından paylaşılır (v1'in ajan başına `_{agent}` eki kaldırıldı).

Buraya şunlar girer:

- Olup biteni tarihleyen anlatı: keşifler, kararlar, hata düzeltmeleri, neyin işe yaradığı, neyin yaramadığı.
- Çapraz kesen notlar ("X'e sıradaki dokunan için: …").
- Her drain'in ne ürettiğinin kaydı (yeni wiki sayfaları, yeni ajan bilgisi).
- Kullanıcı onaylı yapısal değişiklikler (yeni beceri / kural / ajan kararları ve reddedilenleri).

Kurallar:

- **Yalnızca eklemeli.** Mevcut kayıtlar düzenlenmez; yenileri sona eklenir.
- **Asla silinmez** (tarihsel kayıt).
- **`*.local.md` dosya adı kalıbı `.gitignore` kapsamındadır** — gerçekten özel olan içerik için kullanılır (seyrek).

Journal katmanı, eskiden `.atl/agent-memory/` olan şeyin (ajan başına geçmiş) özgün journal katmanıyla (çapraz kesen sinyaller) birleşmiş halidir. Pratikte ikisinin de biçimi aynıydı (tarih + anlatı) ve sıkça birbirine atıf yapıyorlardı; bu yüzden artık tek bir katmandır.

## Wiki — yerine yaz, yalnızca güncel doğru

Dosya adı: `{topic}.md` (kebab-case, sayfa başına bir kavram).

Projenin yaşayan bilgi tabanıdır. Journal'ın (tarihsel kayıt) aksine, wiki **güncel doğruyu** yansıtır — bir bilgi değiştiğinde sayfa eklenmez, güncellenir.

Kurallar:

- **Konuya göre düzenli, tarihe göre değil** (kavram başına bir sayfa).
- **İçeri aldığın `<!-- learning -->` işaretçilerinden [`/drain`](/tr/skills/drain) tarafından yazılır** — konu biçimli güncel doğru buraya iner, tarihli anlatı journal'a gider.
- **Sayfalar ŞU AN doğru olanı yansıtır** — eski bilgi yerine yenisi yazılır.
- **Çapraz başvurulu:** ilgili sayfalar birbirine bağ verir.
- **`index.md` içindekiler tablosudur.**
- **`CLAUDE.md` üst kısmındaki `<!-- wiki:index -->` işaretçi bloğu** konu listesini kendiliğinden derler; böylece ajanlar sayfaları sıfır maliyetle keşfeder.

## Bilgi nasıl yazılır: öğrenme döngüsü

Journal'ı ya da wiki'yi elle yazmazsın. Bunlar, CLI/Skill sınırı boyunca temizce bölünmüş v2 öğrenme döngüsüyle beslenir:

1. **Yakalama (otomatik, deterministik).** Bir konuşma sırasında, öğrenme anı geldiğinde Claude sessiz `<!-- learning -->` işaretçileri düşürür. Kanonik yakalama kuralı [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md) dosyasıdır.
2. **Kuyruğa alma (CLI).** [`atl tick`](/tr/cli/tick) — [`atl setup-hooks`](/tr/cli/setup-hooks) tarafından `UserPromptSubmit` hook'una bağlanır — her işaretçiyi `~/.atl/queue.db` konumundaki dayanıklı bir [bbolt](https://github.com/etcd-io/bbolt) kuyruğuna **tam olarak bir kez** aktarır (işaretçi-hash tekilleştirmesi). Bir oturum açıldığında [`atl session-start`](/tr/cli/setup-hooks) bekleyen sayıyı yüzeye çıkarır.
3. **Drain (Skill, LLM).** [`/drain`](/tr/skills/drain) kuyruğa alınmış maddeleri [`atl learnings`](/tr/cli/learnings) (`status` / `peek` / `ack`) üzerinden okur, her birini wiki'ye (konu doğrusu), journal'a (geçmiş) ya da bir ajanın bilgi tabanına yönlendirir, ardından `ack`'ler.

Deterministik yarı (yakalama + kuyruğa alma) CLI'dir; yargı yarısı (bir öğrenmenin *nereye* ait olduğuna karar vermek) `/drain` becerisidir — CLI o kısmı yapamaz. `ack`'lenen bir madde kuyruktan **silinir**, böylece bir daha asla yeniden raporlanamaz: bu işle-sonra-sil tasarımı, v1'in uzun-oturum yeniden-raporlama hata sınıfını yapısal olarak ortadan kaldırır.

Hook'lar kurulu değilse, işaretçiler zararsızdır (görünmez HTML yorumlarıdır) — bunları işlemek için [`atl tick`](/tr/cli/tick) ve [`/drain`](/tr/skills/drain) komutlarını elle çalıştır.

## Ajanın açılış rutini

Her konuşmanın başında ajan şunları okur (geçerli olduğu durumda):

1. **Kendi ajan dosyası** — takımdan, proje-yerel kopya üzerinden. `agent.md`, `children/*.md` frontmatter'ından kendiliğinden derlenmiş bir Knowledge Base bölümüyle birlikte gelir (bkz. [Children + learnings](/tr/guide/children-and-learnings)).
2. **`CLAUDE.md` `<!-- wiki:index -->` bloğu** — kendiliğinden yüklenir; bilgi haritasını sıfır maliyetle verir. Ajanlar `.atl/wiki/` dizinini doğrudan taramak yerine ilgili wiki sayfalarını bu listeden keşfeder.
3. **Yakın tarihli journal kayıtları** — görev önceki çalışmayla örtüşüyorsa `.atl/journal/` dizininden (varsayılan olarak son 3–5; görev uzun soluklu bir konuya dokunduğunda kapsamı genişlet).
4. **Projeye özgü kurallar** — varsa `.atl/` altında.

Ajan bütün wiki sayfalarını okumaz. Dizini okur (kendiliğinden yüklenir) ve yalnızca görev o alana dokunduğunda ayrıntı sayfalarına olan bağları izler. Bu, bağlamı sıkı tutarken keşfedilebilirliği korur.

## Neden iki katman, üç değil?

v1 üç katman tanımlıyordu: **memory** (proje başına, ajan başına, yalnızca eklemeli geçmiş), **journal** (proje başına, ajanlar arası sinyaller, yalnızca eklemeli) ve **wiki** (proje başına, konu tabanlı, yerine yazma / güncelleme).

İlk ikisi de tarih tabanlı, yalnızca eklemeli ve anlatı biçimliydi. Her çalışma alanında birbirine atıfta bulunarak ya da aynı olayları yedekli olarak yakalayarak son buluyorlardı. "Ajanın kendine özel hafızası vs. başkalarına yayın" ayrımı asla zorlanmadı — herkes iki katmanı da okuyabiliyordu.

Tek bir `journal/` katmanında birleştirildiler çünkü:

- Aynı biçim → anlamsal ayrım yok.
- Aynı kitle (tüm ajanlar her ikisini de okur).
- Aynı yazma deseni (tarihe göre eklemeli).
- Bölünme, farklı içerik üretmeden zihinsel yük getiriyordu ("bu benim için mi yoksa başkaları için mi?").

Birleşen katmanın adı yalnızca `journal/`. Wiki ayrı kalır çünkü paradigması (konu tabanlı güncel doğru) journal'ınkinden (tarih tabanlı geçmiş) gerçekten farklıdır.

## Ajan tarafındaki yansıma: iki eksen

Aynı güncel-doğru-vs-geçmiş bölünmesi takım tarafında da vardır; tek bir projeye sıkıştırılmak yerine *ajanla birlikte taşınır*. Bu iki eksen verir:

- **güncel-doğru vs geçmiş** — wiki + bir ajanın `children/` dizini (güncel) vs journal (geçmiş).
- **proje vs ajan** — `.atl/` (yalnızca bu proje) vs bir ajanın `children/` dizini (ajanın kurulu olduğu her proje).

Somut olarak:

- **Ajan çocuk dosyaları** (ajanın dizinindeki `children/{topic}.md`) wiki'nin ajan tarafındaki karşılığıdır — konu tabanlı, yerine yazma / güncelleme, ajan için projeler arası alan bilgisi. (Beceriler bilgi deposu değil, yordamdır — böyle bir dizinleri yoktur.)

Her çocuk dosyada bir `knowledge-base-summary:` frontmatter alanı bulunur ve bu alan `agent.md`'nin Knowledge Base bölümüne kendiliğinden derlenir. Tüm desen için bkz. [Children + learnings](/tr/guide/children-and-learnings).

## İlgili

- [`/drain`](/tr/skills/drain) — öğrenme kuyruğunu journal kayıtlarına ve wiki sayfalarına katlar.
- [`atl learnings`](/tr/cli/learnings) — `/drain`'in sürdüğü deterministik kuyruk tesisatı (`status` / `peek` / `ack`).
- [`atl tick`](/tr/cli/tick) — yakalanan işaretçileri kuyruğa aktarır (döngünün yakalama yarısı).
- [Children + learnings](/tr/guide/children-and-learnings) — journal + wiki'nin ajan tarafındaki yansıması.
- Kanonik kural: [`core/rules/knowledge-system.md`](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md).
