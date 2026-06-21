# Claude Code sözleşmeleri

`atl` ve onunla birlikte gelen becerilerin oturumlar arasında durumu eşgüdümlemek için `CLAUDE.md` içinde kullandığı işaretçi blok sözleşmeleri. Bunlar, Claude Code'un proje yönergesi mekanizması tarafından kendiliğinden yüklenen, HTML yorumlarıyla sınırlanan bloklardır — görüntülenmiş Markdown içinde görünmezler, Claude dosyayı okuduğunda kaçırılması olanaksızdır.

Aynı deseni kendi `CLAUDE.md` dosyalarında sen de kullanabilirsin. Bloklar yalnızca bir sözleşmedir — özel bir ayrıştırıcı yoktur. Yorum sınırlayıcıları, blokları program yoluyla bulmak, güncellemek ve kaldırmak için güvenli kılar.

## Üç blok

| Blok | Yazan | Amaç |
|---|---|---|
| `<!-- wiki:index -->` | [`/save-learnings`](/tr/skills/drain) | `.atl/wiki/` sayfaları için kendiliğinden yeniden inşa edilen içindekiler tablosu. Proje bağlamıyla yüklenir, Claude'a sıfır maliyetle bilgi haritasını sunar. |
| `<!-- brainstorm:active -->` | [`/brainstorm start`](/tr/skills/brainstorm) ve [`/brainstorm done`](/tr/skills/brainstorm) | Etkin beyin fırtınası konularını proje bağlamına sabitler; bir sonraki oturum bunları kaçıramaz. |
| `<!-- pending-implementation -->` | Beyin fırtınası `done` akışı | Bir beyin fırtınasının X kararını verdiğini ama uygulamasının henüz yayımlanmadığını bir sonraki oturuma anımsatır. |

Üçü de aynı `<!-- block:start --> ... <!-- block:end -->` sınırlayıcı desenini kullanır. Hiçbirinin katı anlamda bir ayrıştırıcısı yoktur — sözdizim değil, sözleşmedir. Ama sözleşme, gerektiğinde basit `sed` ya da düzenli ifadeyle bulmak, güncellemek ve kaldırmak için yeterince tutarlıdır.

> **Not — bu sayfadaki örnek blok içerikleri neden İngilizce?** Aşağıdaki üç şablon (`wiki:index`, `brainstorm:active`, `pending-implementation`) `/save-learnings` ve `/brainstorm` becerileri tarafından otomatik üretilir. Bu beceriler `feedback_speak_turkish` kuralı gereği her zaman İngilizce çıktı verir (taahhüt edilen tüm dosyalar İngilizce olmalıdır). Bu nedenle TR projelerde bile `CLAUDE.md` içindeki bu bloklar İngilizce görünür — örneklerin İngilizce gösterilmesi fiili çıktıyı yansıtır.

## `<!-- wiki:index -->` — bilgi haritası

`.atl/wiki/` dizinindeki her değişiklikten sonra `/save-learnings` tarafından yeniden inşa edilir. `CLAUDE.md` dosyasının üst kısmına yakın, H1 ve giriş paragrafından sonra yaşar:

```markdown
<!-- wiki:index:start -->
## 📚 Knowledge map

Knowledge lives in `.atl/wiki/` (current truth, topic-organized) and `.atl/journal/` (historical record, date-based). Before working on a topic, scan this list — if a page looks relevant, read it before deciding.

**Wiki topics:**
- [docs-audit-false-positive-rate](.atl/wiki/docs-audit-false-positive-rate.md) — ~40% of multi-agent docs-drift audit reports include hallucinated findings
- [pr-merge-discipline](.atl/wiki/pr-merge-discipline.md) — never `gh pr merge` from Claude; surface URL and stop
- [save-learnings-timing](.atl/wiki/save-learnings-timing.md) — run before PR creation as feature branch's last commit
- ...

**Discipline:** Before working on a topic, scan this list. If a topic looks relevant, read the page.
<!-- wiki:index:end -->
```

Her madde tek satırdır: `- [topic](.atl/wiki/topic.md) — tek satırlık özet` (dosya adına göre alfabetik sıralı). Özet, her wiki sayfasının frontmatter ve başlık dışındaki ilk satırından alınır.

**Neden normal bir bölüm değil de işaretçi bloku:** blok program yoluyla yeniden inşa ediliyor. İşaretçilerin içine yapılan elle düzenlemeler bir sonraki `/save-learnings` çalıştırmasında üzerine yazılır. Bir konu eklemek için: `CLAUDE.md` dosyasını doğrudan DÜZENLEME — wiki sayfasını konu başlığıyla oluştur; ardından `/save-learnings` dizini yeniden inşa eder.

## `<!-- brainstorm:active -->` — etkin konu sabitleyici

`/brainstorm start` tarafından yazılır, `/brainstorm done` tarafından kaldırılır. Kapsamın `CLAUDE.md` dosyasının (proje), `~/.claude/CLAUDE.md` dosyasının (global) ya da takım `README.md` dosyasının (takım kapsamı) üst kısmına yakın yaşar:

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[docs-sync-automation](.atl/brain-storms/docs-sync-automation.md)** (project, 2026-05-03) — closing the README + docs-site drift gap; save-learnings extension vs. /docs-sync skill vs. doc-agent
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

Takım depolarında (`~/.claude/repos/agentteamland/{team}/`) aynı bloklar `CLAUDE.md` yerine `README.md` dosyasında yaşayabilir (takım kapsamlı iş için takım `README.md`, Claude tarafından yüklenen aynı rolü üstlenir).

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
- [`/save-learnings`](/tr/skills/drain) — `<!-- wiki:index -->` bloğunu yazar.
- [Bilgi sistemi](/tr/guide/knowledge-system) — wiki:index bloğunun neyi indekslediği.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — bu sözleşmelerin geniş resme nereye oturduğu.
