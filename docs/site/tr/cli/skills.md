# `atl skills`

Platformun kendi skill'leri, agent'ları ve takım manifestleri için belirlenimci, LLM'siz **içerik-kalitesi kontrolleri** — [`atl docs check`](/tr/cli/docs)'in kardeşi. docs-check docs *sitesini* koda karşı doğrularken, skills-check **varlıkların kendisini** doğrular.

Bu, monorepo'nun `core/` ve `teams/` ağaçlarına karşı çalışan bir **maintainer-tarafı** kapıdır. Monorepo dışında hiçbir şey yapmaz ve 0 ile çıkar (ön-uçuş atlaması), böylece son-kullanıcı oturumları onu hiç görmez.

## Kullanım

```bash
atl skills check                      # frontmatter, team.json tutarlılığı, agent-KB çocuklarını doğrula
atl skills check --record-stocktake   # HEAD'i son-stocktake-yapılan commit olarak damgala (/skill-stocktake bir taramanın sonunda çalıştırır)
```

## Neleri kontrol eder

Her kontrol **yapısı gereği sıfır-yanlış-pozitiftir** — bir başarısızlık her zaman gerçek bir sorundur, bu yüzden bir PR'ı buna bağlamak güvenlidir:

| Kontrol | Ne sağlanmalı |
|---|---|
| **frontmatter** | Her skill'in `SKILL.md`'si ve her agent'ın `agent.md`'si bir `name` + `description` frontmatter bloğu taşır. |
| **manifest** | Her `team.json`'ın `agents[]` / `skills[]` adları diskteki dizinlerle eşleşir — **her iki yönde** (bildirilmiş-ama-yok yok, diskte-ama-bildirilmemiş yok). |
| **children** | Her agent-KB çocuğu (`agents/<x>/children/*.md`) boş olmayan bir `knowledge-base-summary` frontmatter'ı bildirir — KB-yeniden-inşa sözleşmesi. |

`atl skills check` herhangi bir başarısızlıkta sıfırdan farklı çıkar; bu yüzden docs-drift kapısının yanında **her PR'ı CI'da kapılar**. Yargı yarısı — bir skill kendi belgelenmiş akışına uyuyor mu? iki skill örtüşüyor mu? — bu belirlenimci ağın değil, eşlik eden [`/skill-stocktake`](/tr/skills/skill-stocktake) skill'inin (LLM) işidir. Bu ayrım CLI/Skill sınırıdır: belirlenimci kontroller burada, zeminli yargı skill'de.

`--record-stocktake`, çalıştırma hatasız tamamlandığında HEAD'i son-stocktake-yapılan commit olarak (`~/.atl` durumunda) damgalar — `/skill-stocktake` skill'i bunu bir taramanın sonunda, oturum-başındaki "stocktake zamanı geldi" sinyalini sıfırlamak için çağırır; `atl rules scan --record`'un kardeşidir.

## İlgili

- [`/skill-stocktake`](/tr/skills/skill-stocktake) — LLM yarısı: itaat + fazlalık, grep-zeminli, değişim-farkında
- [`atl docs check`](/tr/cli/docs) — kardeş kapı: docs-sitesi driftı (bu, varlık içerik-kalitesi)
- [`atl doctor`](/tr/cli/doctor) — çalışma-zamanı kendini-iyileştirme (bu, derleme-zamanı kalite kapısı)
