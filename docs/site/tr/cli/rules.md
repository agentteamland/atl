# `atl rules`

rules-distill'in **belirlenimci toplama** yarısı: skill + agent korpusundaki normatif / emir cümlelerini derler, böylece [`/rules-distill`](/tr/skills/rules-distill) skill'i hangi tekrar eden ilkelerin bir core rule olmayı hak ettiğini yargılayabilir. Yargı — *hangi* adayın gerçek bir ilke olduğu — skill'e aittir; bu, CLI/Skill sınırıdır.

Bu, monorepo'nun `core/` + `teams/`'ine karşı çalışan bir **maintainer-tarafı** yüzeydir. Monorepo dışında hiçbir şey yapmaz ve 0 ile çıkar.

## Kullanım

```bash
atl rules scan            # skill korpusundaki normatif cümleleri yazdır
atl rules scan --json     # aynısı, makine-okunur (file, line, text)
atl rules scan --record   # HEAD'i son distill olarak damgala (bir /rules-distill taramasından sonra)
```

## `atl rules scan`

`core/` ve `teams/` içindeki skill + agent markdown'ını gezer ve **güçlü bir normatif/emir tetikleyicisi** (`always`, `never`, `must`, `don't`, `avoid`, düzenle-öncesi-grep deyimi) taşıyan her satırı, `file:line`'ıyla birlikte yazdırır:

```
core/skills/drain/SKILL.md:49  Be strict — mine only what's worth never-repeating.
teams/software-project-team/agents/api-agent/agent.md:88  Never expose the domain entity directly …
```

**Bilerek fazla toplar** — toplama adımı yalnızca zeminli adayları derler; hangilerinin gerçek tekrar eden ilke olduğuna LLM karar verir. `rules/` alt ağaçları atlanır, çünkü kurallar distill'in *hedefidir*, kaynağı değil.

`--record`, HEAD'i son-distill-edilen commit olarak damgalar (`~/.atl/rules-distill-state.json` içinde); bu, oturum-başı "bir distill gerekiyor" kaçak-korumasını ~1 gün sıfırlar. `/rules-distill` bunu bir taramanın sonunda çağırır.

## İlgili

- [`/rules-distill`](/tr/skills/rules-distill) — LLM yarısı: adayları kümele, mevcut kuralları grep'le, tekrar edenleri core rule olarak öner (insan-onaylı).
- [`/rule`](/tr/skills/rule) — aklında zaten olan tek bir kuralı yazar; rules-distill korpusun *hangi* kuralları istediğini keşfeder.
- [`atl skills`](/tr/cli/skills) — kardeş belirlenimci kapı (varlık içerik-kalitesi; bu ise kural keşfini besler).
