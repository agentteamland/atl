# Claude Code sözleşmeleri

ATL'nin `CLAUDE.md`'i — Claude Code'un talimat olarak otomatik yüklediği dosya — nasıl şekillendirdiği. İki yarım: **üç tier** (her katman için tek yalın dosya, token bütçeleri ve bir ownership modeliyle) ve proje tier'inin oturumlar arası durumu eşgüdümlemek için taşıdığı **marker bloklar**.

[`atl init`](/tr/cli/init) her tier için bir başlangıç oluşturur; `atl install` da bir projede yoksa project başlangıcını otomatik düşürür. Bu sayfa, o scaffold'ların somutlaştırdığı şekildir.

## Üç tier

Tier başına tek, her zaman yüklü dosya — her biri büyüteceğin yalın bir başlangıç. Yetki katmanları (kimlik → kurallar → yönlendirme) ayrı dosyalar değil, **tek dosya içindeki sıralı bölümlerdir** — `SOUL`/`RULES`/`AGENTS` çoğaltması yok.

| Tier | Dosya | Ne tutar | Yumuşak bütçe |
|---|---|---|---|
| **global** | `~/.claude/CLAUDE.md` | Saf kullanıcı **persona'sı** — Claude'un her yerde nasıl çalışmasını istediğin (dil, stil, her projede istediğin mühendislik varsayılanları). **ATL burada hiçbir şey yönetmez.** | ≤ ~80 satır |
| **project** | `<proje>/CLAUDE.md` | **Hibrit:** ATL'nin yönettiği marker bloklar (aşağıda) + kendi kanıttan-doldurulan gerçeklerin (stack, komutlar, kurallar) + opsiyonel ≤5 satırlık glob→skill yönlendirme tablosu. | ≤ ~60 satır |
| **monorepo** | `<repo>/CLAUDE.md` | Proje şeklinin **özelleşmiş + yalın** hâli: bir layout tablosu ve kurallar **pointer** olarak, içeri gömülmeden. | ~30 satır |

Her zaman yüklü zincir (global + project) ~300 satırın altında kalmalı — her satır her oturumda context'e mal olur, bu yüzden her bölüm bütçeye karşı yerini hak etmeli.

### Ownership — yönetilen vs. kullanıcıya ait

**project** tier'inde, bir marker bloğunun içindeki içerik (aşağıdaki üç blok) **ATL-yönetimlidir**: `/brainstorm` ve `/drain` becerileri bu blokları yazar ve yeniden yazar, dolayısıyla marker'ların içine yapılan elle düzenlemeler kalıcı olmaz. Marker'ların **dışındaki** her şey **senindir** — `atl init` onu bir kez tohumlar ve bir daha dokunmaz (yalnızca var olmayan bir `CLAUDE.md`'i yazar). **global** tier'in hiç yönetilen bloğu yoktur: tamamen senindir.

Kullanıcıya ait gerçekleri **kodun gerçekte nasıl davrandığından** doldur — stack'i, gerçek build/test komutlarını, kullanımdaki kuralları yakala. İş kuralları uydurma; sözdizimini tekrar etmek yerine meta-mimariyi hizala.

### Volatilite ayrımı

Oynak çalışma/sprint durumu her-zaman-yüklü dosyaya **ait değildir** — her oturumda context'e mal olur ve sürekli çalkalanır. Onu ayrı, talep üzerine okunan bir tracker'da tut ve `CLAUDE.md`'de yalnızca bir pointer bırak (proje başlangıcının `## State` bölümü bu pointer'dır). Pratik kural: yalnızca güncel sprint için ayrıntılı; artık icrayı şekillendirmediğinde docs/journal'a özetleyip çıkar.

## Üç blok

| Blok | Yazan | Amaç |
|---|---|---|
| `<!-- wiki:index -->` | [`/drain`](/tr/skills/drain) | `.atl/wiki/` sayfaları için kendiliğinden yeniden inşa edilen içindekiler tablosu. Proje bağlamıyla yüklenir, Claude'a sıfır maliyetle bilgi haritasını sunar. |
| `<!-- brainstorm:active -->` | [`/brainstorm start`](/tr/skills/brainstorm) ve [`/brainstorm done`](/tr/skills/brainstorm) | Etkin beyin fırtınası konularını proje bağlamına sabitler; bir sonraki oturum bunları kaçıramaz. |
| `<!-- pending-implementation -->` | Beyin fırtınası `done` akışı | Bir beyin fırtınasının X kararını verdiğini ama uygulamasının henüz yayımlanmadığını bir sonraki oturuma anımsatır. |

Üçü de aynı `<!-- block:start --> ... <!-- block:end -->` sınırlayıcı desenini kullanır. Hiçbirinin katı anlamda bir ayrıştırıcısı yoktur — sözdizim değil, sözleşmedir. Ama sözleşme, gerektiğinde basit `sed` ya da düzenli ifadeyle bulmak, güncellemek ve kaldırmak için yeterince tutarlıdır.

> **Not — bu sayfadaki örnek blok içerikleri neden İngilizce?** Aşağıdaki üç şablon (`wiki:index`, `brainstorm:active`, `pending-implementation`) `/drain` ve `/brainstorm` becerileri tarafından otomatik üretilir. Bu beceriler, platformun `communication-style` kuralı gereği (taahhüt edilen artefaktlar yalnızca İngilizcedir) her zaman İngilizce çıktı verir — konuşma dili ne olursa olsun taahhüt edilen dosyalar İngilizce olmalıdır. Bu nedenle TR projelerde bile `CLAUDE.md` içindeki bu bloklar İngilizce görünür — örneklerin İngilizce gösterilmesi fiili çıktıyı yansıtır.

## `<!-- wiki:index -->` — bilgi haritası

Bir wiki sayfası yazdığında ya da güncellediğinde [`/drain`](/tr/skills/drain) tarafından yeniden inşa edilir. `.atl/wiki/` sayfaları için bir içindekiler tablosu; `CLAUDE.md` dosyasının üst kısmına yakın, H1 ve giriş paragrafından sonra yaşar:

```markdown
<!-- wiki:index:start -->
## 📚 Knowledge map

Knowledge lives in `.atl/wiki/` (current truth, topic-organized) and `.atl/journal/` (historical record, date-based). Before working on a topic, scan this list — if a page looks relevant, read it before deciding.

- [docs-audit-false-positive-rate](.atl/wiki/docs-audit-false-positive-rate.md) — ~40% of multi-agent docs-drift audit reports include hallucinated findings
- [pr-merge-discipline](.atl/wiki/pr-merge-discipline.md) — never `gh pr merge` from Claude; surface URL and stop
- [complexity-resistance](.atl/wiki/complexity-resistance.md) — when a proposal needs paragraphs to defend, that's a smell
<!-- wiki:index:end -->
```

Her madde tek satırdır: `- [topic](.atl/wiki/topic.md) — tek satırlık özet` (dosya adına göre alfabetik sıralı). Özet, her wiki sayfasının frontmatter ve başlık dışındaki ilk satırından alınır. Blok program yoluyla yeniden inşa edilir — marker'ların içine yapılan elle düzenlemeler bir sonraki `/drain` çalıştırmasında üzerine yazılır; yani bir konu eklemek için wiki sayfasını oluşturursun (ya da `/drain`'in oluşturmasına bırakırsın), dizin onu izler.

## `<!-- brainstorm:active -->` — etkin konu sabitleyici

`/brainstorm start` tarafından yazılır, `/brainstorm done` tarafından kaldırılır. Kapsamın `CLAUDE.md` dosyasının (proje), `~/.claude/CLAUDE.md` dosyasının (global) ya da takım `README.md` dosyasının (takım kapsamı) üst kısmına yakın yaşar:

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[profile-team](.atl/brain-storms/profile-team.md)** (project, 2026-05-08) — schema, storage, privacy, and initial scope for the new profile-team package
<!-- brainstorm:active:end -->
```

Birden çok etkin beyin fırtınası, aynı blok içinde madde olarak yan yana yaşar. `done` akışı yalnızca tamamlanmakta olan beyin fırtınasının maddesini kaldırır; madde listesi boşalırsa blok tümüyle kaldırılır (ortada bayatlamış bir "Active brainstorms" başlığı kalmaz).

**Bu sözleşme neden var:** beyin fırtınası kuralının "her oturum başında `.atl/brain-storms/` dizinini `status: active` dosyalar için tara" adımı, Claude'un her oturum başında bunu yapmayı anımsamasına bağlıydı. Etkin beyin fırtınasını `CLAUDE.md` dosyasına sabitlemek, kendiliğinden yüklenmesini sağlar — kaçırılması olanaksızdır. Dizin taraması artık birincil sinyal değil, bir yedeklilik düzeneğidir.

`brainstorm@1.1.0` ile yayımlandı.

## `<!-- pending-implementation -->` — kararı verilmiş ama yayımlanmamış anımsatıcı

Bir beyin fırtınasının `done` akışı henüz hayata geçirilmemiş bir değişikliğe karar verdiğinde yazılır. Bir sonraki oturuma kararın var olduğunu ve işin sıraya alındığını anımsatır:

```markdown
<!-- pending-implementation:start -->
## 🚧 Pending implementation

Brainstorms have decided these but the work hasn't shipped yet:

- **[install-mechanism-redesign](.atl/docs/install-mechanism-redesign.md)** — symlink → project-local copy migration. Atomic write helper + auto-refresh logic queued for `atl v1.0.0`.
<!-- pending-implementation:end -->
```

Uygulama yayımlandığında kaldırılır (tipik olarak değişikliği yayımlayan PR tarafından).

**Bu neden önemli:** anımsatıcı olmadan, tamamlanmış beyin fırtınaları `.atl/docs/` dizininde haftalarca otururken uygulama başka işlerin arkasında sırada bekleyebilir. Sabitleme, sırayı görünür kılar.

## Bloklar nerede yaşar

Bir projenin kök `CLAUDE.md` dosyasında:

```markdown
# Project Name

Kısa giriş paragrafı.

<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms
...
<!-- brainstorm:active:end -->

<!-- pending-implementation:start -->
## 🚧 Pending implementation
...
<!-- pending-implementation:end -->

<!-- wiki:index:start -->
## 📚 Knowledge map
...
<!-- wiki:index:end -->

## What this is
... (normal CLAUDE.md içeriğinin geri kalanı)
```

Sıra, görsel hiyerarşi için önemlidir (en aciliyle başla: etkin beyin fırtınaları → kararı verilmiş ama yayımlanmamış kuyruğu → genel bilgi haritası → serbest biçimli içerik), ama ayrıştırıcı için sıranın önemi yoktur — yalnızca yorum sınırlayıcıları sayılır.

Takım depolarında (ilgili kapsamda `.claude/` altına yüklenen varlıklar) aynı bloklar `CLAUDE.md` yerine `README.md` dosyasında yaşayabilir (takım kapsamlı iş için takım `README.md`, Claude tarafından yüklenen aynı rolü üstlenir).

## Kendi işaretçi bloğunu ekle

Bu desen yalnızca bir sözleşmedir. Kendi otomatik bölümünü eklemek için:

1. Benzersiz bir blok adı seç (örneğin `<!-- ci-status -->`).
2. Yeniden inşa edilecek içeriğini `<!-- block:start --> ... <!-- block:end -->` arasına sar.
3. Betiğinin her yeniden inşada bloğu bulup yerine yazmasını sağla.

Örneğin, basit bir "geçerli sprint" bloğu:

```markdown
<!-- sprint:start -->
## 🏃 Geçerli sprint

Sprint 5 — Phase 1.D-η — kavram sayfaları.
- [ ] knowledge-system sayfası (EN + TR)
- [ ] children-and-learnings sayfası (EN + TR)
- ...
<!-- sprint:end -->
```

Sprint adını ve kontrol listesini girdi olarak alıp `CLAUDE.md` içindeki blok içeriğini değiştiren bir betikle güncelle. Proje bağlamıyla birlikte kendiliğinden yüklenir.

## Neden HTML yorumu?

Düz Markdown başlıkları (`## Active brainstorms`) görsel bölüm olarak iş görürdü, ama:

- İçinde elle düzenleme yapmak yeniden inşayı bozabilir.
- Düzenli ifade tabanlı bul-değiştir, başlık duyarlı olmak zorunda kalırdı (kırılgan).
- Kullanıcı, ilgili ama farklı içerikle gerçek bir "Active brainstorms" bölümü yazabilir.

HTML yorumları:

- Görüntülenmiş Markdown'da görünmezler (blok boş ya da ilgisiz olduğunda görsel kalabalık olmaz).
- Basit düzenli ifadeyle bulmak, güncellemek ve kaldırmak kolaydır (başlık ayrıştırıcısı gerekmez).
- İnsan eliyle yazılan bölümlerden ayrıdır (ad alanı çakışması olmaz).
- Claude Code'un proje yönergesi mekanizması tarafından kendiliğinden yüklenir (Claude, `<!-- -->` çerçevesine karşın okur).

## İlgili

- [`/brainstorm`](/tr/skills/brainstorm) — `<!-- brainstorm:active -->` bloğunu yazar ve kaldırır.
- [`/drain`](/tr/skills/drain) — `<!-- wiki:index -->` bloğunu yeniden inşa eder.
- [Bilgi sistemi](/tr/guide/knowledge-system) — wiki:index bloğunun neyi indekslediği.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — bu sözleşmelerin geniş resme nereye oturduğu.
