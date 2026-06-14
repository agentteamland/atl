# `/wiki`

Proje bilgi tabanı — yaşayan, çapraz başvurulu, daima güncel. Kurma, yeni bilgi alma, var olanı sorgulama, bayatlama denetimi.

Wiki şu soruyu yanıtlar: **"Bu projede X hakkında güncel doğru nedir?"** Journal kayıtlarının (yalnızca eklemeli tarihsel anlatı) ya da yerleşmiş karar belgelerinin (durağan kayıtlar) aksine, wiki sayfaları **etkin biçimde bakım görür** — bilgi değiştiğinde eski doğru yan yana yığılmaz, değiştirilir.

Global beceri olarak [core](https://github.com/agentteamland/core) içinde yayımlanır.

## Wiki nerede yaşar?

```
.atl/wiki/
├── index.md                ← Kendiliğinden bakım gören içindekiler tablosu
├── {topic-1}.md            ← Bilgi sayfaları (kebab-case, sayfa başına tek kavram)
├── {topic-2}.md
└── ...
```

Sayfalar projenin kök `CLAUDE.md` dosyasında bir `<!-- wiki:index -->` işaretçi bloğu üzerinden de dizinlenir — [`/save-learnings`](/tr/skills/save-learnings) tarafından kendiliğinden yeniden inşa edilir; böylece her Claude oturumu açılışta wiki haritasını yükler.

## Dört kip

### `init` — boş wiki iskelesi kur

```
/wiki init
```

`.atl/wiki/` dizinini ve bir `index.md` şablonunu oluşturur. İdempotent — zaten kurulmuş bir wiki üzerinde yeniden çalıştırmak işlem yapmaz (`wiki: already initialized (N pages)`).

`init`, `atl setup-hooks` yapılandırılmış ama henüz `.atl/wiki/` dizini olmayan bir projenin ilk oturumunda kendiliğinden de tetiklenir.

### `ingest` — proje kaynaklarındaki bilgiyi wiki sayfalarına çek

```
/wiki ingest
```

Her proje bilgi kaynağını tarar ve wiki sayfalarını günceller:

| Kaynak | Önemi |
|---|---|
| Mevcut oturum transkriptindeki `<!-- learning -->` işaretçileri | Birincil kaynak — canlı konuşma bilgisi. |
| `.atl/journal/*.md` | Ajan başına ve tarih başına öğrenme kaydı (Q4 sonrası tek katman). |
| `.atl/docs/*.md` | Tamamlanmış beyin fırtınalarından gelen yerleşmiş karar belgeleri. |
| `.atl/brain-storms/*.md` (yalnızca `status: completed` olanlar) | Karar bağlamı. |
| Yakın konuşma bağlamı | Az önce ne tartışıldı / ne inşa edildi. |

Her bilgi parçası için: konuyu belirler, wiki sayfasını bulur ya da oluşturur, **çelişkileri çözerek yeni bilgiyi birleştirir**, çapraz başvuruları ve `index.md` dosyasını günceller.

> Wiki sayfaları **güncel doğruyu** yansıtır. Eski bir kayıt "X desenini kullanıyoruz" diyor ama daha sonraki bir kayıt "X sorun çıkardı, Y'ye geçtik" diyorsa, wiki sayfası "Y kullanıyoruz" der — ikisi birden değil.

### `query` — bir soru için wiki'de arama yap

```
/wiki query bu projede önbellekleme nasıl çalışır?
```

İlgili sayfaları bulmak için `index.md` dosyasını okur, o sayfaları okur ve kaynak sayfalara atıflar yaparak bir yanıt sentezler.

Şu durumlarda yararlıdır:

- Yeni bir geliştirici (ya da yeni bir Claude oturumu) bütün depoyu yeniden okumadan bir konuyu anlamak istediğinde.
- Bir şeyin nasıl karara bağlandığını ya da hayata geçirildiğini unuttuğunda.
- Bir değişiklik yapmadan önce hızlı bir hatırlatma istediğinde.

### `lint` — tüm wiki için sağlık denetimi

```
/wiki lint
```

Yapılan denetimler:

| Denetim | Eylem |
|---|---|
| Bayatlamış sayfalar (>30 gün, kaynak dosyalar bu arada değişmişse) | İnceleme için işaretler. |
| Sayfalar arası çelişkiler | Çözüm için işaretler. |
| Yetim sayfalar (gelen bağ yok) | Bağlantılar önerir. |
| Eksik sayfalar (atıf var ama dosya yok) | Oluşturmayı önerir; bir taslak oluşturur. |
| Yinelenen konular | Birleştirmeyi önerir. |
| Dizin eşzamanlaması (`index.md` ↔ gerçek dosyalar) | Eşzaman dışıysa kendiliğinden düzeltir. |

Kendiliğinden düzeltilebilir sorunlar sessizce düzeltilir. Çelişkiler ve bayatlamış içerik insan incelemesi için raporlanır.

Örnek çıktı:

```
Wiki Health Report:
──────────────────────────
📊 Total pages: 12
✅ Healthy: 9
⚠️  Stale (>30 days): 2 (caching-patterns.md, email-setup.md)
❌ Contradiction found: auth.md says "15min token" but jwt-config.md says "30min token"
🔗 Orphan: database-indexes.md (no incoming links)
📝 Missing: "rate-limiting" referenced in api-endpoints.md but no page exists
──────────────────────────

Fixing automatically...
✅ index.md synced
✅ Created rate-limiting.md (stub)
⚠️  Review needed: auth.md vs jwt-config.md contradiction
⚠️  Review needed: 2 stale pages
```

## Sayfa biçimi

Wiki sayfaları tutarlı bir yapı izler; böylece hem insanlar hem ajanlar onları hızlıca okuyabilir:

```markdown
# {Topic Title}

> Last updated: {date}
> Sources: [journal](../journal/...), [brainstorm](../brain-storms/...)

## Summary
{2-3 cümlelik genel bakış}

## Current State
{ŞU AN doğru olan — tarih değil, plan değil, yalnızca güncel gerçeklik}

## Key Decisions
{Bu konudaki önemli kararlar, kısa gerekçesiyle}

## Patterns & Rules
{Bu konu için yerleşik sözleşmeler}

## Known Issues
{Mevcut sorunlar ya da kısıtlar}

## Related
- [{related-topic-1}]({related-topic-1}.md)
- [{related-topic-2}]({related-topic-2}.md)
```

## Diğer becerilerle nasıl etkileşir?

Wiki büyük ölçüde **kendiliğinden bakım görür** — insanlar sayfaları elle nadiren düzenler. `atl setup-hooks` yapılandırıldığında olağan akış:

1. Claude konuşma sırasında satır içi `<!-- learning topic=... -->` işaretçileri düşürür ([learning-capture kuralı](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) gereği).
2. **Bir sonraki oturumun `SessionStart` adımında** → `atl session-start` çalışır, `atl learning-capture --previous-transcripts` ile işaretçileri tarar.
3. İşaretçi bulunursa → Claude [`/save-learnings --from-markers`](/tr/skills/save-learnings) komutunu çağırır.
4. `/save-learnings`, `journal/`, ajan `children/`, beceri `learnings/` ve **wiki sayfalarını** günceller (yerine yazma / güncelleme).

Örnek işaretçi yayılımı:

```
<!-- learning topic: redis-cache; body: TTL 30 dk olmalı, 15 değil -->
  → journal/{date}_{agent}.md: tarihiyle birlikte tarihsel not eklenir
  → wiki/redis-cache.md: GÜNCELLENİR — "TTL 30 dakika" (eski "15 dakika" yerine)
  → (alana özgüyse) agents/{agent}/children/redis-cache.md
```

[`/brainstorm done`](/tr/skills/brainstorm) benzer biçimde, beyin fırtınası tamamlandığında kararlarını wiki'ye alır.

İşaretçiler (ya da hook'lar) olmadan da `/wiki ingest` ve `/save-learnings` (elle kip) çalışır — ikisi de doğrudan çağrılabilir.

## Önemli kurallar

1. **Wiki = güncel doğru.** Tarih değil, plan değil. ŞU AN doğru olan.
2. **Yerine yaz, eklemeyle birikme.** Bir bilgi değiştiğinde eski sürüm değiştirilir.
3. **Daima çapraz başvur.** Her sayfa ilgili sayfalara bağ verir. Yetimler `lint` tarafından işaretlenir.
4. **Kendiliğinden bakım görür.** İnsanlar wiki'yi elle nadiren düzenler — `/save-learnings`, `/brainstorm done` ve `/wiki ingest` onu güncel tutar.
5. **Ajan tarafından okunabilir.** Sayfalar hem insan hem yapay zekâ için yapılandırılmıştır — net bölümler, belirsizlik yok.
6. **Konu tabanlıdır, tarih tabanlı değildir.** Journal'ın (tarih tabanlı) aksine wiki konuya göre düzenlenmiştir. Kavram başına tek sayfa.
7. **Düzenli denetim.** `/wiki lint` komutunu zaman zaman (aylık ya da bir şey eğri göründüğünde) çalıştır.

## İlgili

- [`/save-learnings`](/tr/skills/save-learnings) — bu becerinin sorguladığı ve denetlediği wiki sayfalarını yazar.
- [`/brainstorm`](/tr/skills/brainstorm) — `done` kipi kararları wiki'ye alır.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — wiki ile journal ve yerleşmiş belgelerin karşılaştırması.

## Kaynak

- Belirtim: [core/skills/wiki/skill.md](https://github.com/agentteamland/core/blob/main/skills/wiki/skill.md).
