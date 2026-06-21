# `/wiki`

Wiki, projenin bilgi tabanıdır — yaşayan, çapraz başvurulu, daima güncel. Tek bir soruyu yanıtlar: **"Bu projede X hakkındaki güncel doğru nedir?"** Journal kayıtlarının (yalnızca eklemeli tarihsel anlatı) ya da yerleşmiş karar belgelerinin (durağan kayıtlar) aksine, wiki sayfaları **etkin biçimde bakım görür** — bilgi değiştiğinde eski doğru yan yana yığılmaz, değiştirilir.

::: warning v2'de bağımsız bir `/wiki` becerisi yok
v2'de **`/wiki` becerisi yoktur**. v1'de `/wiki`, elle çağırdığın açık dört kipli bir beceriydi (`init` / `ingest` / `query` / `lint`). v2'de wiki bir komut değil, öğrenme döngüsünün bir **hedefidir**: [`/drain`](/tr/skills/drain) öğrenme kuyruğunu bilgi tabanına katlarken wiki sayfalarını yazar ve günceller; sen de sayfaları doğrudan okursun (`.atl/wiki/` altında düz Markdown olarak yaşarlar). Bu sayfa wiki'nin **ne olduğunu** ve sayfalarının nasıl biçimlendiğini belgeler; yazma/güncelleme mekaniği için [`/drain`](/tr/skills/drain) sayfasına bak.
:::

## Wiki nerede yaşar?

```
.atl/wiki/
├── index.md                ← İçindekiler tablosu
├── {topic-1}.md            ← Bilgi sayfaları (kebab-case, sayfa başına tek kavram)
├── {topic-2}.md
└── ...
```

Sayfalar projenin kök `CLAUDE.md` dosyasında bir `<!-- wiki:index -->` işaretçi bloğu üzerinden de dizinlenir; böylece her Claude oturumu açılışta wiki haritasını yükler.

## Sayfalar nasıl yazılır: `/drain`

Wiki **kendiliğinden bakım görür** — insanlar sayfaları elle nadiren düzenler. v2 öğrenme döngüsü onu güncel tutar:

1. Claude konuşma sırasında satır içi sessiz `<!-- learning -->` işaretçileri düşürür (learning-capture kuralı gereği).
2. [`atl tick`](/tr/cli/tick) her işaretçiyi tam olarak bir kez kalıcı bir kuyruğa aktarır. `atl` ardından bir sonraki oturum başlangıcında **"N öğrenme bekliyor"** raporlar.
3. [`/drain`](/tr/skills/drain) komutunu çalıştırırsın. Kuyruktaki her öğe için `/drain` kebab-case bir konu çıkarır ve onu yönlendirir: konu biçimli *güncel doğru*, `<proj>/.atl/wiki/<topic>.md` dosyasına iner (sayfa varsa bayatlamış kısmın yerine yazar ya da onunla birleştirir) ve journal'a tarihli bir madde de eklenir.

Yani *"Redis önbellek TTL'si 15 değil, 30 dakika olmalı"* gibi bir öğrenme, `wiki/redis-ttl.md` dosyasını "TTL 30 dakikadır" diyecek biçimde günceller — eski "15 dakika"nın yerine yazar, ikinci bir satır eklemez.

`/drain` wiki'yi yazdığı için v1'deki bağımsız `ingest` / `lint` / `init` fiilleri ortadan kalktı — alma işlemi kuyruk üzerinden gerçekleşir ve ayrı bir lint komutu yoktur. Wiki'yi okumak için `index.md` ile ilgili sayfaları doğrudan aç ya da oturum içinde Claude'a sor.

## Sayfa biçimi

Wiki sayfaları tutarlı bir yapı izler; böylece hem insanlar hem ajanlar onları hızlıca okuyabilir:

```markdown
# {Topic Title}

> Last updated: {date}
> Sources: [journal](../journal/...), [docs](../docs/...)

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

## Önemli kurallar

1. **Wiki = güncel doğru.** Tarih değil, plan değil. ŞU AN doğru olan.
2. **Yerine yaz, eklemeyle birikme.** Bir bilgi değiştiğinde eski sürüm değiştirilir. (Tarih journal'da yaşar.)
3. **Daima çapraz başvur.** Her sayfa ilgili sayfalara bağ verir.
4. **Kendiliğinden bakım görür.** İnsanlar wiki'yi elle nadiren düzenler — [`/drain`](/tr/skills/drain) onu öğrenme kuyruğundan güncel tutar.
5. **Ajan tarafından okunabilir.** Sayfalar hem insan hem yapay zekâ tüketimine göre yapılandırılmıştır — net bölümler, belirsizlik yok.
6. **Konu tabanlıdır, tarih tabanlı değildir.** Journal'ın (tarih tabanlı) aksine wiki konuya göre düzenlenmiştir. Kavram başına tek sayfa.
7. **Daima NEDEN'i ekle.** Gerekçesiz bir bilgi çürür — yalnızca sonucu değil, akıl yürütmeyi de kaydet.

## Wiki, journal ve yerleşmiş belgeler karşılaştırması

| Katman | Biçim | Nasıl düzenlenir |
|---|---|---|
| **Wiki** (`.atl/wiki/`) | Güncel doğru, konuya göre düzenli | `/drain` tarafından yerinde değiştirilir/birleştirilir |
| **Journal** (`.atl/journal/`) | Tarihsel anlatı, tarihe göre düzenli | `/drain` tarafından yalnızca eklemeli |
| **Yerleşmiş belgeler** (`.atl/docs/`) | Tamamlanmış kararların durağan kayıtları | Bir [`/brainstorm`](/tr/skills/brainstorm) tamamlandığında bir kez yazılır |

## İlgili

- [`/drain`](/tr/skills/drain) — wiki sayfalarını öğrenme kuyruğundan yazar ve günceller.
- [`atl tick`](/tr/cli/tick) — öğrenme işaretçilerini `/drain`'in tükettiği kuyruğa taşır.
- [`/brainstorm`](/tr/skills/brainstorm) — `done` kipi, wiki'nin çapraz başvurduğu yerleşmiş belgeleri üretir.

## Kaynak

v2'de wiki'nin kendine ait bir becerisi yoktur; sayfaları `/drain` tarafından üretilir.

- Belirtim: [core/skills/drain/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/drain/SKILL.md)
