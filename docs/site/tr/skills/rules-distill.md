# `/rules-distill`

Skill + agent korpusundan tekrar eden ilkeleri çıkar ve onları core rule olarak öner — **[`/rule`](/tr/skills/rule)'un tamamlayıcısı**. `/rule` aklında zaten olan tek bir kuralı yazar; `/rules-distill` korpusun *kendisini* okur ve birçok skill/agent'ta tekrarlayan ama henüz kural olmayan ilkeleri yüzeye çıkarır.

Hem elle çağrılabilir **hem de** otomatik tetiklenir. [`atl session-start`](/tr/cli/setup-hooks), son distill'den bu yana korpus-etkileyen commit'ler (`core/` `teams/`) biriktiğinde **"bir distill gerekiyor"** sinyali verir — ~1-günlük kaçak-koruma ile sınırlanır. Yıkıcı olmadığından (yalnızca öneri), Lane 3 otomasyon kararı uyarınca otomatik sinyal verir.

## Ne zaman kullanılır?

- `atl`, oturum başında **"bir distill gerekiyor — /rules-distill çalıştır"** raporladığında.
- Korpusu bilerek madenlemek istediğin her an (`--all` tüm korpusu zorlar).

## Nasıl çalışır?

### Belirlenimci toplama

Skill, skill + agent korpusundaki her normatif/emir satırını (`always`, `never`, `must`, `don't`, `avoid`, düzenle-öncesi-grep deyimi) `file:line`'ıyla yazdıran [`atl rules scan`](/tr/cli/rules) çalıştırır. Bilerek fazla toplar — toplama yalnızca zeminli adayları derler; yargılayan skill'dir.

### Değişim-farkında

Her seferinde tüm korpusu distill etmek israftır; bu yüzden son distill'den bu yana değişene odaklanır (imleç `~/.atl/rules-distill-state.json`'da yaşar). `--all` tüm korpusu zorlar.

### Kümele + yargıla

LLM, toplanan cümleleri **tekrar eden ilkelere** gruplar — birkaç skill/agent'ta ifade edilen aynı disiplin (ör. birden çok agent'ta geçen "düzenlemeden önce grep'le"). Tek seferlik bir şey ilke değildir; asıl eşik gerçek tekrardır. **Önce mevcut kuralları grep'ler**, böylece zaten kural olanı asla yeniden önermez; ve her adayı tekrarladığı `file:line`'larla zeminler.

### Önerir — asla otomatik yazmaz

Ayakta kalan her aday AskUserQuestion ile **önerilir** ("ilke X, A, B, C'de tekrarlıyor — bir core rule'a yükseltelim mi?"). Yeni bir core rule yapısal büyümedir — insan onaylar ve onaylanan bir aday, new-rule-shipping-checklist'i taşıyan [`/rule`](/tr/skills/rule) ile yazılır. `/rules-distill` asla özerk biçimde core rule yazmaz.

### Distill'i kaydeder

Tamamlanınca skill imleci damgalar (`atl rules scan --record`); bu, kaçak-korumayı sıfırlar, böylece session-start ~1 gün yeniden sinyal vermez.

## CLI / Skill ayrımı

`/rules-distill`, kural keşfinin LLM yarısıdır. Belirlenimci yarı — zeminli aday cümleleri toplamak — [`atl rules scan`](/tr/cli/rules)'tir. Skill toplamayı asla yeniden türetmez; LLM çabasını yalnızca kümeleme, tekrar-yargısı ve öneri adımına harcar. distill korpusun *hangi* kuralı istediğini söyler; `/rule` *nasıl ship edileceğini* söyler.

## İlgili

- [`atl rules`](/tr/cli/rules) — bu skill'in üstüne kurulduğu belirlenimci toplama.
- [`/rule`](/tr/skills/rule) — onaylanan bir adayı yazar (rules-distill keşfeder, /rule ship eder).
- [`/skill-stocktake`](/tr/skills/skill-stocktake) — kardeş korpus-hijyeni backstop'u (skill kalitesi; bu ise kural keşfi).

## Kaynak

- Belirtim: [core/skills/rules-distill/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rules-distill/SKILL.md)
