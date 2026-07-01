# `/skill-stocktake`

Skill + agent korpusunu, belirlenimci bir kapının yargılayamayacağı içerik-kalitesi sorunları için tara — **skill-kalitesinin yargı yarısı** (docs-sync v2 deseni). [`atl skills check`](/tr/cli/skills) korpusun yapısal olarak sağlam olduğunu kanıtlar; `/skill-stocktake` ise **itaat**i (bir skill kendi belgelenmiş akışına uyuyor mu?) ve **fazlalığı** (iki skill aynı işi mi yapıyor ya da birbiriyle çelişiyor mu?) yargılar.

Hem elle çağrılabilir **hem de** otomatik tetiklenir. [`atl session-start`](/tr/cli/setup-hooks), son kaydedilen stocktake'ten bu yana varlık-etkileyen commit'ler (`core/` `teams/`) biriktiğinde **"bir stocktake gerekiyor"** sinyali verir — ~1-günlük kaçak-koruma ile sınırlanır — ve skill'i o zaman çalıştırırsın.

## Ne zaman kullanılır?

- `atl`, oturum başında **"bir stocktake gerekiyor — /skill-stocktake çalıştır"** raporladığında.
- Skill korpusunu bilerek taramak istediğin her an (`--all` tüm korpusu zorlar).

## Nasıl çalışır?

### Önce belirlenimci

Skill, [`atl skills check`](/tr/cli/skills) çalıştırır ve her **FAIL**'i düzeltir (eksik bir frontmatter bloğu, bir team.json↔disk uyumsuzluğu, özeti olmayan bir çocuk) — mekanik, sıfır-yanlış-pozitif. CLI'nin zaten kanıtladığını asla elle yargılamaz.

### Değişim-farkında

Her seferinde tüm korpusu taramak israftır; bu yüzden son stocktake'ten bu yana dokunulan skill/agent'lara odaklanır (imleç `~/.atl/skill-stocktake-state.json`'da yaşar). Hiçbir şey değişmediyse ve `--all` yoksa, taranacak bir şey yoktur.

### Anlamsal, grep-zeminli, çekişmeli

Sonra iki geçiş, her biri ~%40'lık çok-ajanlı denetim halüsinasyon oranına karşı korumalı:

- **İtaat** — her kapsamdaki skill'in `SKILL.md`'sini kendine karşı oku: hiç tanımlamadığı bir bayrağa atıfta bulunan bir adım, açıklama ile gövde arasında bir çelişki, sarkan bir "bkz. adım N", akışın hiç üretmediği vaat edilen bir çıktı.
- **Fazlalık** — her kapsamdaki skill'i geri kalanla karşılaştır: aynı tetikleyiciyi iddia eden, aynı işi yapan ya da çelişen talimatlar veren iki skill.

Her bulgu **grep-zeminlidir** (birebir alıntılanır, yoksa düşürülür) ve **çekişmeli doğrulanır** (sorgulanır, ayakta kalmazsa düşürülür).

### Önerir — asla sessizce yeniden yazmaz

Ayakta kalan bulgular AskUserQuestion ile **önerilir** (bozuk bir akışı düzelt / örtüşen iki skill'i birleştir / bir çelişkiyi uzlaştır). Bir skill'in yeniden yazımı kimliğe dokunur, bu yüzden insan onaylar — bu, `/skill-stocktake`'in [`/docs-audit`](/tr/skills/docs-audit)'ten daha katı olduğu tek yerdir; docs-audit'in düzyazı düzeltmeleri otomatik uygulanabilir.

### Stocktake'i kaydeder

Tamamlanınca skill imleci damgalar (`atl skills check --record-stocktake`); bu, kaçak-korumayı sıfırlar, böylece session-start ~1 gün boyunca yeniden sinyal vermez.

## CLI / Skill ayrımı

`/skill-stocktake`, skill-kalitesinin LLM yarısıdır. Belirlenimci yarı — frontmatter geçerliliği, team.json tutarlılığı, agent-KB çocukları — her PR'da CI kapısı olarak da çalışan [`atl skills check`](/tr/cli/skills)'tir. Skill, CLI'nin kanıtladığını asla yeniden türetmez; LLM çabasını yalnızca itaat + fazlalığa harcar, grep-zeminli ve çekişmeli doğrulanmış.

## İlgili

- [`atl skills`](/tr/cli/skills) — bu skill'in üstüne kurulduğu belirlenimci yarı.
- [`/docs-audit`](/tr/skills/docs-audit) — docs sitesi için aynı backstop şekli (onun düzyazı düzeltmeleri özerktir; bu skill'inkiler önerilir).

## Kaynak

- Belirtim: [core/skills/skill-stocktake/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/skill-stocktake/SKILL.md)
