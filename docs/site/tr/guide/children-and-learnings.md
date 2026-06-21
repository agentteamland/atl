# Children + learnings

Karmaşık ajanların ve becerilerin birikmiş alan bilgisi için kullandığı şekil: kısa bir üst düzey dosya, üstüne konu başına bir dosya barındıran bir alt dizin; her dosya, üst dosyanın dizin bölümünü kendiliğinden yeniden inşa eden tek satırlık bir `knowledge-base-summary` frontmatter taşır.

Aynı desen, iki ad — ajanlar için **`children/`**, beceriler için **`learnings/`**. İkisi arasında tek bir zihinsel model.

Kanonik kural [`core/rules/agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

## Bu desen neden var?

Onsuz, karmaşık ajanlar ve beceriler iki kötü şekilden birinde son bulur:

1. **Tek parça dosyalar** — her şey tek bir `agent.md` ya da `skill.md` içine yığılmış. Bir parçayı diğerlerine dokunmadan güncellemek zor. Farklar gürültülü olur. Her yeniden okuma jeton yakar.
2. **Elle hazırlanmış dizin bölümleri** — bir insanın konu dosyalarına paralel bakım yaptığı ayrı bir `agent.md` içindekiler tablosu. Biri güncellemeyi unuttuğu an gerçeklikten kayar.

Children + learnings deseni ikisini birden çözer:

- **Konu başına bir dosya** — bir parçayı diğerlerine dokunmadan güncellersin.
- **Kendiliğinden yeniden inşa edilen dizin** — üst düzey dosyanın "Knowledge Base" / "Accumulated Learnings" bölümü her [`/drain`](/tr/skills/drain) çalıştırmasında frontmatter'dan yeniden oluşturulur. Elle yapılan düzenlemeler üzerine yazılır — kaynak doğruluk her çocuk dosyanın frontmatter'ıdır.

Sonuç: bilgi sürtünmesizce birikir, üst düzey dosya sıkı kalır ve dizin asla bayatlamaz.

## Children — ajanlar için

Her karmaşık ajan şu yapıyla düzenlenir:

```
~/.claude/repos/agentteamland/{team}/agents/{agent-name}/
├── agent.md              ← Kimlik, sorumluluk alanı, çekirdek ilkeler (kısa, gömülü)
└── children/             ← Ayrıntılı bilgi, desenler, stratejiler (her konu ayrı bir dosyada)
    ├── topic-1.md
    ├── topic-2.md
    └── ...
```

[`atl install`](/tr/cli/install) sonrasında aynı yapı projenin `.claude/agents/{agent-name}/` dizinine kopyalanır.

### Kurallar

1. **`agent.md` kısa kalır.** Yalnızca: kimlik, sorumluluk alanı (olumlu liste), çekirdek ilkeler (değişmeyen, kısa maddeler), Knowledge Base bölümü (kendiliğinden derlenir), "`children/` dizinini oku" yönergesi.
2. **Ayrıntılı her şey `children/` altına gider.** Stratejiler, desenler, iş akışları, sözleşmeler — her biri ayrı bir `.md` dosyasında.
3. **Yeni konu = yeni dosya.** `agent.md`'ye elle dokunmadan, `children/` altına bir `.md` dosyası ekle. Knowledge Base bölümünü `/drain` her çocuk dosyanın frontmatter'ından kendiliğinden yeniden inşa eder.
4. **Güncelleme = tek dosya.** Bir konuyu güncellemek için yalnızca ilgili `children/` dosyasına dokunulur.
5. **Tek parça ajan dosyaları yasaktır.**
6. **Bu desen tüm ajanlar için geçerlidir.** API, Socket, Worker, Flutter, React, Mail, Log, Infra — hepsi aynı yapıyı izler.

## Learnings — beceriler için

Her karmaşık beceri ajan biçimini aynalar. İki konum önemlidir:

```
# Kaynak doğruluk (atl tarafından sunulan takım / core deposu):
~/.claude/repos/agentteamland/{team-or-core}/skills/{skill-name}/
├── skill.md              ← Becerinin yordamı (adımlar, kimlik, akış). Kısa kalır.
└── learnings/            ← Birikmiş sınır durumları, başarılı desenler, kötü desenler
    ├── topic-1.md
    ├── topic-2.md
    └── ...

# Proje-yerel kopya (atl-v1.0.0 sonrası kurulum topolojisi):
{project}/.claude/skills/{skill-name}/
├── skill.md              ← Aynı kopya; değiştirilmemişse `atl update` yeniler
└── learnings/            ← Aynı desen; kendiliğinden büyüyen kopyalar
```

[`atl install`](/tr/cli/install) becerileri (ve ajanlar ile kuralları) projeye kopyalar. [`atl update`](/tr/cli/update) değiştirilmemiş kopyaları üç-yönlü SHA-256 karşılaştırmasıyla yeniler. `/drain` önce proje-yerel kopyaya yazar; ardından `atl promote` kazanımları global katmanına taşır ve `atl publish` bunları takımın deposuna üst kaynak olarak önerebilir.

Aynı şekil, aynı kurallar, aynı `knowledge-base-summary` frontmatter sözleşmesi. Becerinin `skill.md` dosyası, `learnings/*.md` frontmatter'ından kendiliğinden derlenen bir "Accumulated Learnings" bölümüyle birlikte gelir — `agent.md` Knowledge Base mekanizmasının aynısı.

**Beceriyi neden ajan üzerine aynalayalım?** "Kendini iyileştiren beceri" çerçevelemesi, ajanların (Claude) beceriyi çağırırken görebileceği birikmiş deneyim için yapılandırılmış bir yer kazandırır. `learnings/` olmadan her beceri kullanımı, daha önce karşılaşılmış sınır durumlarında sıfırdan başlar.

## Frontmatter sözleşmesi

Her `children/*.md` ve `learnings/*.md` dosyası bir `knowledge-base-summary` frontmatter alanı taşımak ZORUNDADIR:

```markdown
---
knowledge-base-summary: "<kendiliğinden yeniden inşa edilen dizin bölümünde kullanılan bir-üç satırlık özet>"
---

# <Konu Başlığı>

<asıl içerik — desenler, stratejiler, örnekler — gerektiği kadar uzun>
```

Bu özet, üst dosyanın Knowledge Base / Accumulated Learnings bölümünü besler. Bu alan olmadan `/drain` ya konuyu yeniden inşada atlar YA DA (kendi oluşturduğu yeni dosyalar için) alanı türetilmiş bir özetle yazar; her iki durumda da dosyada bir adet bulunmalıdır.

## Kendiliğinden yeniden inşa edilen dizin bölümleri

`/drain` çalıştığında, üst dosyanın dizin bölümünü her `children/*.md` (ajanlar için) ya da `learnings/*.md` (beceriler için) frontmatter'ından yeniden inşa eder. Şekil her ikisi için aynıdır:

```markdown
## Knowledge Base                     ← (beceriler için "Accumulated Learnings")

### <Konu 1 (dosya adından başlık biçimine getirilmiş)>
<knowledge-base-summary>
→ [Details](children/topic-1.md)     ← (beceriler için learnings/topic-1.md)

### <Konu 2>
<knowledge-base-summary>
→ [Details](children/topic-2.md)

...
```

Bu bölüme yapılan elle düzenlemeler bir sonraki `/drain` çalıştırmasında **üzerine yazılır** — kaynak doğruluk her çocuk dosyanın frontmatter'ıdır. `agent.md` / `skill.md` dosyasının geri kalanı (kimlik, sorumluluk, ilkeler, akış) yeniden inşa tarafından **değiştirilmez**.

## Üç güncelleme katmanı

Bu bölünme, "bilgi birikir" davranışının kendiliğinden ve sürtünmesiz olmasını sağlarken üst düzey dosyanın kimliğini kaymaya karşı korur:

| Katman | Ne değişir | Nasıl |
|---|---|---|
| **A — otomatik** | Bir `children/{topic}.md` ya da `learnings/{topic}.md` dosyası oluşturulur veya güncellenir. | `/drain` doğrudan yazar. Soru sormaz. |
| **B — otomatik** | Üst dosyanın Knowledge Base / Accumulated Learnings bölümü yeni frontmatter kümesinden yeniden inşa edilir. | `/drain` yeniden inşa eder. Soru sormaz. |
| **C — onay kapılı** | Üst dosyanın kimliği / sorumluluğu / ilkeleri / beceri akışı değişmek zorundadır. | `/drain` bir `AskUserQuestion` onay kapısı açar. Kullanıcı onaylar; dosya güncellenir. Kullanıcı reddeder; öneri journal'a "reddedildi" olarak yazılır. |

C katmanı, üst düzey kimliği kendiliğinden kaymaya karşı korur. Kullanıcı bir değişikliği onayladıktan sonra dosya güncellenir.

## Blueprint deseni (yalnızca ajanlar)

Her ajanın bir **birincil üretim birimi** vardır — tekrar tekrar ürettiği ana şey. Bu birimin `children/` dizininde, şunları içeren bir blueprint dosyası bulunmak ZORUNDADIR:

1. **Şablon** — üretim biriminin yapısal iskeleti (kod taslağı).
2. **Kontrol listesi** — birim tamamlanmadan önce doğrulanması gereken her şey.
3. **Adlandırma sözleşmeleri** — dosyaların, sınıfların ve yöntemlerin nasıl adlandırılacağı.
4. **Yaşam döngüsü** — oluşturma → kayıt → test akışı.

Ajanın üretim biriminin yeni bir örneğini oluşturması gerektiğinde blueprint'i okur ve adım adım izler.

| Ajan | Birincil üretim birimi | Blueprint dosyası |
|---|---|---|
| API Agent | Feature (Command/Query/Handler/Validator) | `children/workflows.md` |
| Socket Agent | Hub method + Event | `children/hub-method-blueprint.md` |
| Worker Agent | Scheduled Job | `children/job-blueprint.md` |
| Flutter Agent | Screen / Widget | `children/screen-blueprint.md` |
| React Agent | Component / Page | `children/component-blueprint.md` |

Blueprint olmadan ajan, yeni birimleri nasıl oluşturacağını tahmin eder. Blueprint ile:

- Her birim aynı yapıyı izler.
- Hiçbir şey unutulmaz (kontrol listesi eksiksizliği garanti eder).
- Yeni takım üyeleri (ya da yeni Claude oturumları) tutarlı çıktı üretir.
- Kalite kazara değil, tekrarlanabilirdir.

(Becerilerin bir blueprint deseni yoktur — bir beceri, şablon güdümlü bir birim değil, yordamın kendisidir. Accumulated Learnings bölümü, beceri için blueprint kontrol listesinin karşılığıdır: hatırlanacak şeyler, dikkat edilecek sınır durumları.)

## İlgili

- [Bilgi sistemi](/tr/guide/knowledge-system) — bu takım tarafı desenin proje tarafındaki yansıması (journal + wiki).
- [`/drain`](/tr/skills/drain) — `children/` ve `learnings/` dosyalarını yazar; üst dizin bölümlerini yeniden inşa eder.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — `learnings/` deseninin nereye oturduğu.
- Kanonik kural: [`core/rules/agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md).

## Tarihçe

- `core@1.0.0`: ajan children deseni tanıtıldı. Knowledge Base bölümü elle bakım görüyordu.
- `core@1.8.0`: [self-updating-learning-loop](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) Q3'ü, children desenini becerilere genişletti (`learnings/`, `children/` yansıması). Knowledge Base ve Accumulated Learnings bölümleri frontmatter'dan kendiliğinden yeniden inşa edilir oldu. Kimlik / çekirdek değişiklikleri için C-katmanı onay kapısı kuralın bir parçası olarak biçimlendi. "Agent Configuration Rules" adından "Agent + skill structure rules" adına geçilerek genişleyen kapsamı yansıtıldı.
