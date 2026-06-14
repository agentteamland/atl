# Doğrulanmış takımlar

Resmi AgentTeamLand kayıt defteri şu anda **2 doğrulanmış takım** içerir. Her takım bağımsız sürümlenir, topluluk katkısına açıktır ve [team.json şemasına](/tr/authoring/team-json) uyar.

> **Doğrulanmış durumu**, takımın AgentTeamLand bakımcıları tarafından kalite, kapsam ve bağımlılık hijyeni açısından incelendiğini ifade eder. Topluluk katkılı takımlar (`status: community`) kurulum sırasında bir uyarıyla gelir.

## Göz at

| Takım | Sürüm | Açıklama |
|---|---|---|
| [`software-project-team`](/tr/teams/software-project-team) | 1.2.1 | Tam yığın yazılım projeleri için 13 uzman ajan (.NET 9 + Flutter + React + Postgres + RabbitMQ + Redis + Elasticsearch + MinIO). Phase 2.C: ajan Knowledge Base bölümleri çocuk dosyaların frontmatter alanlarından kendiliğinden yeniden inşa edilir. |
| [`design-system-team`](/tr/teams/design-system-team) | 0.8.1 | Herhangi bir projenin içinde tasarım sistemleri ve UI prototipleri — yerelde, dosya tabanlı, tarayıcıda görüntülenebilir. `/dst-*` becerileri JSON durum dosyaları üretir ve `.dst/` altında Tailwind ile işlenmiş HTML sayfaları üretir. |

## Herhangi bir takımı kur

```bash
atl install <team-name>
```

Yukarıdaki her iki takım da kısa adıyla kurulabilir. `atl` adı herkese açık kayıt defterinden çözer ve depoyu paylaşılan önbelleğe (`~/.claude/repos/agentteamland/`) klonlar.

## Tek bir projeye birden çok takım kur

İki takım da aynı projede temiz biçimde yan yana yaşar — `atl` v0.1.2+ çakışma uyarılarıyla birlikte çoklu takım kurulumunu destekler (iki takım aynı adda bir öğe bildirdiğinde en son kurulan kazanır, tek satırlık bir uyarı yazdırılır).

```bash
cd your-project
atl install software-project-team       # tam yığın ajanlar + iskele
atl install design-system-team          # tasarım sistemi + prototip araçları ekle

atl list
# ✓ software-project-team@1.2.1    13 agents, 3 skills
# ✓ design-system-team@0.8.1        2 agents, 10 skills (dst-*)
```

İki takım birbirini tamamlamak üzere tasarlanmıştır: `/dst-*` becerileriyle tasarla, software-project-team ajanlarıyla (`flutter-agent`, `react-agent` vb.) hayata geçir.

## Takım katkısında bulunma

Kendi takımını yayımlamak ister misin? [Takım yazma rehberine](/tr/authoring/creating-a-team) bak — bir `team.json` yaz, herkese açık bir Git deposuna push et ve [kayıt defterine](https://github.com/agentteamland/registry) bir PR gönder.
