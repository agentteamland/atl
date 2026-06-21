# Birinci Taraf Takımlar

AgentTeamLand **2 birinci taraf takım** yayımlar. Her biri bağımsız olarak sürümlenir ve [`team.json` sözleşmesine](/tr/authoring/team-json) uyar. Her ikisi de takım kataloğu aracılığıyla keşfedilebilir ve referans yoluyla kurulabilir.

> Katalog, [`atl-team`](https://github.com/topics/atl-team) GitHub konusuyla etiketlenen herkese açık depolardan üretilir. Gönderi yapılacak bir kayıt defteri yoktur — depoyu etiketlemek (ya da [`atl publish`](/tr/cli/publish) çalıştırmak) takımı listeler.

## Göz at

| Takım | Sürüm | Açıklama |
|------|---------|-------------|
| [`software-project-team`](/tr/teams/software-project-team) | 1.2.1 | Tam yığın yazılım projeleri için 13 uzman ajan (.NET 9 + Flutter + React + Postgres + RabbitMQ + Redis + Elasticsearch + MinIO). Phase 2.C: ajan KB bölümleri çocuk dosyaların frontmatter alanlarından kendiliğinden yeniden inşa edilir. |
| [`design-system-team`](/tr/teams/design-system-team) | 0.8.1 | Herhangi bir projenin içinde tasarım sistemleri ve UI prototipleri — yerelde, dosya tabanlı, tarayıcıda görüntülenebilir. `/dst-*` becerileri JSON durum dosyaları ve `.dst/` altında Tailwind ile işlenmiş HTML sayfaları üretir. |

Her ikisi de `agentteamland/` tanıtıcısı altında yayımlandığından [`atl search`](/tr/cli/search) çıktısında **`[verified]`** (doğrulanmış) rozetini taşır. Rozet, AgentTeamLand bakımcıları tarafından incelenmiş takımları işaretler (`agentteamland/*` ve bir bakımcı izin listesi); bir takımdaki durum alanı değildir; kendi yayımladığın bir takımda bu rozetin bulunmaması o takımın güvensiz olduğu anlamına gelmez.

## Herhangi bir takımı kur

```bash
atl install agentteamland/software-project-team
```

Takımlar `<tanıtıcı>/<isim>` referansıyla kurulur. [`atl install`](/tr/cli/install), referansı GitHub destekli katalogda çözer, kaynağı HTTPS üzerinden geçici bir tarball olarak getirir ve takımın `agents/`, `skills/` ile `rules/` dizinlerini kapsamın `.claude/` dizinine kopyalar:

```
atl: installed agentteamland/software-project-team@1.2.1 at project scope
```

Varsayılan olarak takım, yayımcısının bildirdiği kapsamda (proje, global veya her ikisi) kurulur. Geçersiz kılmak için `--global` ya da `--project` kullan. İki katmanın nasıl etkileşime girdiğini öğrenmek için bkz. [kapsamlar](/tr/guide/concepts#kapsam-global-ve-proje).

## Tek bir projeye birden çok takım kur

Her iki takım da aynı projede sorunsuz biçimde yan yana bulunabilir. İki takım aynı adda bir varlık bildirdiğinde en son kurulan kazanır ve `atl` tek satırlık bir çakışma uyarısı yazdırır:

```bash
cd your-project
atl install agentteamland/software-project-team    # tam yığın ajanlar + iskele
atl install agentteamland/design-system-team       # tasarım sistemi + prototip araçları ekle

atl list
# project:
#   agentteamland/software-project-team@1.2.1
#   agentteamland/design-system-team@0.8.1
```

İki takım birbirini tamamlamak üzere tasarlanmıştır: `/dst-*` becerileriyle tasarla, software-project-team ajanlarıyla (`flutter-agent`, `react-agent` vb.) hayata geçir.

## Takım katkısında bulunma

Kendi takımını yayımlamak ister misin? [Takım yazma rehberine](/tr/authoring/creating-a-team) bak — bir `team.json` yaz, herkese açık bir GitHub deposuna push et, ardından o depoyu `atl-team` konusuyla etiketle ya da [`atl publish`](/tr/cli/publish) komutunu çalıştır. Katalog buradan devralır; PR göndermene gerek yoktur.
