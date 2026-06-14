# `/save-learnings`

Bir konuşmada (ya da birden çok transkriptin işaretçili bölgelerinde) öğrenilenleri journal, wiki, ajan `children/` ve beceri `learnings/` katmanlarına kalıcılaştırır. Ardından durum dosyasını günceller; böylece bir sonraki oturum aynı işaretçileri yeniden işlemez.

Bu beceri, kendiliğinden tetiklenen öğrenme döngüsünün **işleme yarısıdır**:

```
[atl session-start hook'u] → işlenmemiş işaretçileri raporlar
        ↓
[Bir sonraki turda Claude] → /save-learnings --from-markers --transcripts ... çağırır
        ↓
[beceri] → işaretçiler journal / wiki / ajan children / beceri learnings'e iner
        ↓
[beceri döngüyü kapatır] → atl learning-capture --commit-from-transcripts
```

Global beceri olarak [core](https://github.com/agentteamland/core) içinde, `core@1.0.0` sürümünden bu yana yayımlanır. `core@1.8.0` sürümünde, SessionStart güdümlü hook akışını ve kendiliğinden büyüyen `children/` + `learnings/` desenlerini desteklemek üzere yeniden yazıldı.

## Üç çağırma kipi

| Kip | Çağrı | Ne zaman kullanılır |
|---|---|---|
| **Hook kipi** (kendiliğinden tetikleme) | `/save-learnings --from-markers --transcripts a.jsonl,b.jsonl` | En yaygın yol. `atl session-start` işaretçileri raporlar; Claude bir sonraki turda bu komutu çağırır. |
| **Tek transkript kipi** | `/save-learnings --from-markers` | Eski kullanım: yalnızca mevcut oturumun kendi transkriptini tarar. Hook akışı yayımlandığından bu yana nadiren gerekir. |
| **Elle kip** | `/save-learnings [agent-name]` | Kullanıcı doğrudan çağırır; işaretçi gerekmez. Canlı konuşmayı çözümler. |

## Ne yazar?

Yazımların büyük bölümü sessizce gerçekleşir — kullanıcı her yazımda bir soru görmez; yalnızca son özeti görür. `AskUserQuestion` üzerinden geçen beş istisna vardır (çalıştırma başına TEK soruda toplanır):

1. **Yeni beceri oluşturma** (bir iş akışı deseni 2+ kez tekrarlandığında).
2. **Yeni kural oluşturma** (kesin bir "daima X" / "asla Y" sözleşmesi billurlaştığında).
3. **Yeni ajan oluşturma** (bir alan açıkça ayrı bir ajan olduğunda).
4. **Mevcut ajan kimliğinin değişmesi** (sorumluluk / ilkelerin kayması gerektiğinde).
5. **Mevcut becerinin çekirdeğinin değişmesi** (becerinin adımlarının değişmesi gerektiğinde).

Geri kalan her şey — journal, wiki, ajan `children/`, beceri `learnings/` ve Knowledge Base + Accumulated Learnings bölümlerinin kendiliğinden yeniden inşası — soru sormadan yapılır.

### Öğrenme başına hedef matrisi

Her öğrenmenin *biçimi* nereye ineceğini belirler. Sınıflandırma, işaretçinin `kind` alanından kendiliğinden yapılır (elle kipte konuşma incelenerek):

| Öğrenme biçimi | Hedef |
|---|---|
| Tarih damgalı anlatı ("X denedik, sonra Y, sonunda Z işe yaradı") | Yalnızca journal kaydı. |
| Konu biçimli güncel doğru ("kimlik doğrulamanın doğru yolu …") | Wiki sayfası (varsa yerine yazılır) + journal kaydı. |
| Belirli bir ajan için alan bilgisi | Ajanın `children/{topic}.md` dosyası + journal kaydı. |
| Belirli bir beceri için alan bilgisi | Becerinin `learnings/{topic}.md` dosyası + journal kaydı. |
| Tekrar eden iş akışı (2+ kez) | **AskUserQuestion → yeni beceri.** |
| Billurlaşmış sözleşme ("asla X", "daima Y") | **AskUserQuestion → `/rule` üzerinden yeni kural.** |
| Sahibi olmayan alan | **AskUserQuestion → yeni ajan.** |
| Mevcut ajanın kimliği genişledi | **AskUserQuestion → `agent.md` çekirdek güncellemesi.** |
| Mevcut becerinin çekirdek akışı değişmeli | **AskUserQuestion → `skill.md` çekirdek güncellemesi.** |

## Neye dokunur?

| Yüzey | Ne değişir | Biçim |
|---|---|---|
| `.atl/journal/{YYYY-MM-DD}_{agent}.md` | Ajan ve tarih başına bir kayıt. Mevcuda eklenir, hash ile yinelenenler ayıklanır. | Frontmatter (`date`, `agent`, `tags`) + `## Summary` + `## Learnings` + `## Auto-Created` + `## User-Approved Structural Changes` + `## Notes for Other Agents`. |
| `.atl/wiki/{topic}.md` | Güncel doğru için yerine yazma biçimli güncelleme. Konu yepyeniyse şablondan yeni sayfa. | Standart wiki sayfa biçimi (Last updated / Current state / Sources). |
| `CLAUDE.md` | `<!-- wiki:index -->` işaretçi bloğunu mevcut wiki sayfa kümesinden yeniden inşa eder. | Dosya adına göre sıralı; sayfa başına tek satırlık özet. |
| Ajan `children/{topic}.md` | Zorunlu `knowledge-base-summary` frontmatter alanıyla ekleme veya oluşturma. | Frontmatter + gövde. |
| Ajan `agent.md` Knowledge Base bölümü | Her çocuk dosyanın `knowledge-base-summary` frontmatter alanından kendiliğinden yeniden inşa. Bu bölüme yapılan elle düzenlemeler üzerine yazılır. | Yeniden inşa edilen blokla aynı şekil — bkz. [agent-structure kuralı](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md). |
| Beceri `learnings/{topic}.md` | Ajan `children/` deseninin yansıması — aynı frontmatter sözleşmesi. | Frontmatter + gövde. |
| Beceri `skill.md` Accumulated Learnings bölümü | Her learnings dosyasının frontmatter alanından kendiliğinden yeniden inşa. | Ajan KB bölümüyle aynı şekil. |

Her başarılı çalıştırmadan sonra beceri, işaretçi başına hash'leri `~/.claude/state/learning-capture-state.json` dosyasına (5000 girişle FIFO sınırlı) işlemek için `atl learning-capture --commit-from-transcripts` komutunu çağırır. Döngüyü kapatan adım budur — bir sonraki [`atl session-start`](/tr/cli/setup-hooks), yeni bir şey yokken sıfır işaretçi raporlar.

## Hook kipi örneği

Yeni bir oturum açılır. `atl session-start`, `SessionStart` hook'u olarak çalışır, önceki oturumun transkriptlerinde 5 işlenmemiş işaretçi görür ve (Claude'un `additionalContext` alanına) yazdırır:

```
🧠 learning-capture: 5 unprocessed markers across 4 transcripts
  by kind: 1 convention, 1 decision, 2 discovery, 1 pattern

→ Run: /save-learnings --from-markers --transcripts <path1>,<path2>,<path3>,<path4>
```

Claude bunu okur, beceriyi olduğu gibi çağırır ve beceri:

1. Transkriptlerden her `<!-- learning -->` bloğunu çıkarır.
2. `(topic, body)` ikilisini hashler ve aynı tarihte journal'da bulunanları atlar.
3. Her öğrenmeyi `kind` ve gövde biçimine göre sınıflandırır.
4. Gerekli yerlere journal + wiki + children + learnings yazar.
5. (5 onay kapılı değişiklikten herhangi biri öneriliyorsa) tek bir `AskUserQuestion` içinde toplar.
6. Durum dosyasını ilerletmek için `atl learning-capture --commit-from-transcripts` çalıştırır.
7. Tek bir özet bloğu raporlar.

Sıkıcı oturumlar (işaretçi yok) sıfır jeton harcar — hook hiçbir şey yazdırmaz, beceri hiç çağrılmaz.

## Elle kip örneği

```
/save-learnings api-agent
```

Satır içi işaretçi düşürülmemiş, elle kodlama içeren bir konuşmanın sonunda kullanılır. Beceri canlı konuşmayı öğrenmeler (desenler, sözleşmeler, kararlar, keşifler) için tarar ve hook kipiyle aynı hedef matrisini uygular. Durum dosyası ilerletilmez — elle kip işaretçi-durum-dosyası sözleşmesine bağlı değildir.

## İşaretçi biçimi

Satır içi işaretçiler, assistant turlarına gömülü HTML yorumlarıdır. Görüntülenmiş çıktıda görünmezler, transkriptte korunurlar ve her biri ~40 jetondur:

```html
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7-day JWT refresh chosen because we want long sessions; user logs in once a week max.
-->
```

Zorunlu alanlar:

- `topic` — kebab-case, tek kavram (wiki sayfasının adı olur).
- `kind` — şunlardan biri: `bug-fix | decision | pattern | anti-pattern | discovery | convention`.
- `doc-impact` — şunlardan biri: `none | readme | docs | both | breaking` (varsayılan `none`).
- `body` — 1-3 cümle. **Daima NEDEN'i içer.**

Tüm işaretçi belirtimi için [learning-capture kuralına](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) ve onları yüzeye çıkaran tarayıcı için [`atl learning-capture`](/tr/cli/learning-capture) sayfasına bak.

## Önemli kurallar

1. **Yapısal olmayan yazımlar için onay alınmaz.** Journal / wiki / ajan children / beceri learnings yazımlarının tümü sessizce gerçekleşir.
2. **`AskUserQuestion` YALNIZCA yeni yapılar veya kimlik değişiklikleri için kullanılır.** Yeni beceri, yeni kural, yeni ajan, ajan kimliği, beceri çekirdeği.
3. **Durum dosyası yazımı kapanış parantezidir.** Durum dosyası güncellenmediği sürece işaretçiler işlenmemiş sayılır ve bir sonraki `SessionStart`'ta yeniden raporlanır.
4. **Hassas bilgi filtresi.** Parolalar, jetonlar ve API anahtarları journal / wiki / takım depolarına ASLA yazılmaz. Olası kimlik bilgileri sansürlenir.
5. **Her yerde idempotent.** Aynı işaretçilerle yeniden çalıştırmak artımlı bir değişiklik üretmez (hash ile yinelenenler ayıklanır, aynı içerikle yerine yazma işlem yapmaz, KB yeniden inşası belirlenimcidir).
6. **Takım deposuna push'lama bakımcılar için kendiliğindendir; kullanıcılar için zarif biçimde başarısız olur.** Push iznine sahip olmayan kullanıcılar değişiklikleri yerelde tutar; üst kaynağa katkı akışı bunları zaman içinde PR olarak paketler.
7. **Onay kapısı toplulaştırması.** Birden çok yapısal değişiklik **tek bir** `AskUserQuestion` içinde çoklu seçimle sunulur, N ayrı soru olarak değil.
8. **Beceri oluşturma eşiği = 2 örnek.** Tek bir iş akışı oluşumunda kendiliğinden yeni beceri önerme.
9. **Kural oluşturma ölçütü = açık "daima X" / "asla Y" anlatımı.** Çekincenin olduğu ifade kurala değil, wiki'ye gider.

## İlgili

- [`atl learning-capture`](/tr/cli/learning-capture) — işaretçileri yüzeye çıkaran ve hash'leri duruma işleyen CLI tarayıcısı.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — bu beceriyi tetikleyen `SessionStart` hook'unu bağlar.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — bu becerinin sürdürdüğü `learnings/` deseni.
- [`/wiki`](/tr/skills/wiki) — eşlik eden bilgi tabanı becerisi (bu beceri wiki sayfaları yazar; `/wiki` onları sorgular ve denetler).

## Kaynak

- Belirtim: [core/skills/save-learnings/skill.md](https://github.com/agentteamland/core/blob/main/skills/save-learnings/skill.md).
- Kural: [core/rules/learning-capture.md](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md).
- Kural: [core/rules/agent-structure.md](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md).
