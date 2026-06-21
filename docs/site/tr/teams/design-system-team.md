# design-system-team

> Herhangi bir projenin içinde tasarım sistemleri ve UI prototipleri — yerel, dosya tabanlı, tarayıcıda görüntülenebilir.

**En son sürüm:** `0.8.1`
**Durum:** Doğrulanmış
**Depo:** [github.com/agentteamland/design-system-team](https://github.com/agentteamland/design-system-team)

## Ne yapar?

Claude Design tarzı tasarım sistemi ve ekran prototipi çalışmasını her projeye yerel olarak getirir — dış bir uygulamaya, API anahtarına ya da arka plan sürecine gerek kalmadan. Tüm LLM işi senin Claude Code oturumunun içinde yapılır (mevcut Pro/Max yetkini kullanır); çıktılar JSON durum dosyaları ve Tailwind ile işlenmiş HTML sayfaları olarak projenin `.dst/` dizinine düşer ve herhangi bir tarayıcıda görüntülenebilir.

## Kur

```bash
cd your-project
atl install agentteamland/design-system-team
```

[`atl`](/tr/cli/overview) ≥ 0.1.2 gerekir (`team.json` içindeki `requires.atl` alanı).

## Hızlı başlangıç

```bash
# 1. Mevcut projede .dst/ iskelesini kur
/dst-init

# 2. Bir tasarım sistemi oluştur (etkileşimli soru-yanıt)
/dst-new-ds primary

# 3. Bu tasarım sistemine bağlı bir prototip oluştur
/dst-new-prototype --ds primary login-screen

# 4. Tarayıcında her şeyi görüntüle
/dst-open
```

## Düzen

`/dst-init` sonrası projen şunları kazanır:

```
.dst/
  index.html               ← stüdyo açılış sayfası (tarayıcıda aç)
  styles.css               ← paylaşımlı Tailwind yardımcıları
  state.json               ← hafif manifesto

  design-systems/
    primary/
      ds.json              ← renk paleti, tipografi, boşluk, bileşenler, marka, ses
      detail.html          ← bu tasarım sisteminin zengin görsel sayfası
      assets/              ← logolar, simgeler

  prototypes/
    login-screen/
      prototype.json       ← ekran durumu (bir DS'e bağlı)
      preview.html         ← çoklu durumlu görsel (idle/loading/error/success)
      handoff.zip          ← isteğe bağlı dışarı aktarma paketi
```

`.dst/` Git dostudur — commit et, bedavaya tasarım geçmişi kazan.

## Ajanlar (2)

- **ds-architect-agent** — Kapsamlı tasarım sistemleri tasarlar (renk paleti kuramı, tipografi rampaları, boşluk, bileşenler, marka kimliği, ses). 7 çocuk dosyası şemayı, renk paleti kuramını, tipografiyi, boşluğu, bileşenleri, markayı, proje bağlamı okumayı ve şablon işlemeyi kapsar.
- **prototype-agent** — Seçilen tasarım sisteminin jetonlarına kesin uyan ekran prototipleri tasarlar (jeton sadakati, durum kapsamı, erişilebilirlik — pazarlığa kapalı). 7 çocuk dosyası.

## Beceriler (10)

| Beceri | Amacı |
|---|---|
| `/dst-init` | Mevcut projede `.dst/` iskelesini kurar ya da tazeler (kendi kendini onaran). |
| `/dst-new-ds <name>` | Etkileşimli soru-yanıt ile yeni tasarım sistemi oluşturur. |
| `/dst-edit-ds <name> "<change>"` | Metinsel bir değişiklik uygular; yapısal bir değişiklikse bağlı prototiplere şelaleyle yansıtır. |
| `/dst-delete-ds <name>` | Bir DS'i kaldırır (bağımlı prototip varsa reddeder; `--force` ile yetim bırakılır). |
| `/dst-new-prototype --ds <name> <prototype>` | Bir DS'e bağlı ekran prototipi oluşturur. |
| `/dst-edit-prototype <name> "<change>"` | Metinsel bir değişiklik uygular, jeton sadakatini korur. |
| `/dst-delete-prototype <name>` | Bir prototipi kaldırır. |
| `/dst-open` | `.dst/index.html` dosyasını varsayılan tarayıcıda açar. |
| `/dst-handoff <prototype> [--target …]` | Prototip + DS + varlıkları paketler; `flutter-agent` / `react-agent` ajanını kaynak koda bütünleştirme için brifler. |
| `/dst-questions [init\|sync\|add\|resolve\|list]` | `.atl/wiki/open-questions.md` dosyasını yönetir (ekran çalışmasını engelleyen sorular); etkin listeyi `CLAUDE.md` dosyasına kendiliğinden sabitler. |

## Neden var?

Nisan 2026'da üç güç bir araya geldi:

1. Claude Design'ın hız sınırları Pro/Max planlarda bile sıkıştırıyor.
2. Anthropic, üçüncü taraf uygulamaların Pro/Max OAuth kullanmasını engelledi ([17 Nisan zorlaması](https://venturebeat.com/technology/anthropic-cuts-off-the-ability-to-use-claude-subscriptions-with-openclaw-and-third-party-ai-agents)).
3. MCP örnekleme (sunucuların LLM çağrılarını devretmesinin resmi yolu) Claude Code'da henüz hayata geçirilmedi.

Yön değişimi: **ayrı bir uygulama yapma.** Bir takım yap. Beceriler tüm LLM işini oturumun içinde yapar. Çıktılar durağan dosyalardır. Uygulama benzeri değer ayrı bir süreçten değil, dosyalar artı tarayıcı üzerinden gelir.

## İyi eşleşir

- [`software-project-team`](/tr/teams/software-project-team) — `/dst-*` becerileriyle tasarla, `flutter-agent` / `react-agent` / `api-agent` ile hayata geçir.

## Yayımlananlar

- **v0.4.0** — `dst-init` kendi kendini onarır hâle geldi (mevcut `.dst/` dizinleri güvenle yükseltilebilir; korunan JSON durumundan HTML/CSS yeniden işlenir); `dst-handoff` becerisi (tek komutla tasarım → kod; `flutter-agent` / `react-agent` üzerinden); prototip `target` alanı.
- **v0.5.0** — koyu kip jeton kapsamı + ilk prototip boru hattı iyileştirmeleri.
- **v0.6.0** — parça temelli şablon mimarisi (bir bileşen = bir dosya; 600 satırlık tek parça bitti); katalog **7 grupta 57 bileşene** büyüdü (yedinci grup olarak Charts eklendi + 12 Tier 1 ilkel bileşen).
- **v0.7.0** — `/dst-questions` becerisi + `open-questions-pinning` kuralı: ekran çalışmasını engelleyen ürün soruları için standart bir yer ile `CLAUDE.md` dosyasına kendiliğinden sabitleme (`brainstorm@1.1.0` sabitleme desenini aynalar).

## Sıradakiler (öngörü)

- Canlı yenileme (dosya izleyici → SSE ile tarayıcıda kendiliğinden yenileme).
- Projeler arası DS paylaşımı için `/dst-export` / `/dst-import`.
