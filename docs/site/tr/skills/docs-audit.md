# `/docs-audit`

Tüm doküman sitesini drift için tara — docs-sync v2'nin backstop'u. Değişiklik-anı kontrolleri ([`/create-pr`](/tr/skills/create-pr) docs-impact geçişi + deterministik CI kapısı) drift'i oluşurken yakalar; `/docs-audit`, onların kaçırdıklarının ağıdır: dış-dünya link çürümesi, atlanan bir docs-pass, birikmiş drift.

Hem elle çağrılabilir hem otomatik tetiklenir. [`atl session-start`](/tr/cli/setup-hooks), son kaydedilen denetimden bu yana doc-etkili commit'ler (`docs/` `core/` `cli/`) birikince **"a full audit is due"** sinyalini verir — ~1 günlük runaway-guard ile sınırlı — ve skill'i o zaman çalıştırırsın. Deterministik yarı [`atl docs check`](/tr/cli/docs)'tir; bu skill, bir makinenin yapamadığı anlamsal muhakemeyi ekler.

## Ne zaman kullanılır

- `atl`, oturum başında **"a full audit is due — run /docs-audit"** bildirdiğinde.
- Tüm doküman sitesini bilerek taramak istediğin her an.

## Nasıl çalışır

### Önce deterministik

Skill [`atl docs check`](/tr/cli/docs)'i çalıştırır ve her **FAIL**'i düzeltir (eksik sayfa, olmayan TR aynası, bayat kurulum talimatı) — mekanik, sıfır yanlış-pozitif. CLI'nın zaten kanıtladığını asla elle denetlemez.

Dış-dünya linklerini HTTP üzerinden de yoklamak için **`--external`** geç (`atl docs check --external`) — yavaş ve ağ-bağımlı olduğundan opt-in'dir; varsayılan tarama yalnızca site-içi kontrolleri ve prose-kod drift'ini kapsar.

### Anlamsal, grep-temelli, çekişmeli

Sonra sitenin her bölümünü (`cli/`, `guide/`, `skills/`, `teams/`, …) tarar ve her sayfayı, anlattığı kod / `SKILL.md` / `team.json`'a karşı okur. İki koruma, ~%40'lık çok-ajanlı denetim halüsinasyon oranını aşağıda tutar:

- **Grep-temelli** — hiçbir drift, verbatim bir kaynak alıntısı olmadan kaydedilmez. Kodda temellendirilemeyen bir iddia düşürülür.
- **Çekişmeli** — her aday bulgu sorgulanır ("prose aslında doğru mu? bu bilinçli tarihsel karşıtlık mı?") ve hayatta kalmazsa düşürülür.

Hayatta kalan düzeltmeler EN sayfaya uygulanır, TR aynası yeniden üretilir ve her şey, maintainer'ın incelemesi için bir **PR** olarak açılır — izin isteği değil, otonom taslak.

### Denetimi kaydeder

Bitince skill, denetim imlecini damgalar (`atl docs check --record-audit`); bu, runaway-guard'ı sıfırlar, böylece session-start ~1 gün boyunca tekrar sinyal vermez.

## CLI / Skill ayrımı

`/docs-audit`, docs-correctness'ın LLM yarısıdır. Deterministik yarı — coverage, parity, bayat-talimat denylist'i, link bütünlüğü — her PR'da CI kapısı olarak da çalışan [`atl docs check`](/tr/cli/docs)'tir. Skill, CLI'nın kanıtladığını asla yeniden türetmez; LLM çabasını yalnızca grep-temelli ve çekişmeli-doğrulanan anlamsal prose-kod drift'ine harcar.

## İlgili

- [`atl docs`](/tr/cli/docs) — bu skill'in üzerine kurulduğu deterministik yarı.
- [`/create-pr`](/tr/skills/create-pr) — docs-impact geçişi değişiklik-anı katmanıdır; `/docs-audit` backstop'tur.
- [Skill'lere genel bakış](/tr/skills/drain)

## Kaynak

- Şartname: [core/skills/docs-audit/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/docs-audit/SKILL.md)
