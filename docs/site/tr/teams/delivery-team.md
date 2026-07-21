# delivery-team

**delivery-team**, **takılabilir bir backend (Azure DevOps ya da GitHub) üzerinde iş-öğesi güdümlü,
sprint tabanlı otonom bir yazılım-teslim organizasyonudur** — gerçek bir tracker'da (Azure Boards +
Repos ya da GitHub Issues + Projects + Pull Requests) iş öğelerini planlayan, ayrıştıran, geliştiren,
doğrulayan, gözden geçiren ve teslim eden bir rol-ajan ekibi; Product Owner rolünde bir insanla. Bu bir
**proje-kapsamlı** (project-scope) takımdır: teslim ettiği depoya kurulur.

```bash
atl install agentteamland/delivery-team
```

Kurulum, rol-ajanları, seremoni skill'lerini, bilgi paketlerini (knowledge packs) ve her iki backend
adaptör paketini (`backends/azure/`, `backends/github/`) projenin `.claude/` dizinine yerleştirir;
ardından `/delivery-init`, seremonilerin ve orkestrasyon motorunun okuduğu `.delivery/` config'ini yazar.

## Organizasyon

delivery-team, her biri kendi reflekslerine sahip birer uzman olan **rol-ajanlardan** oluşur:

| Rol | Ne yapar |
|---|---|
| `intake` | Ham bir isteği şekillenmiş bir Epic/Feature backlog öğesine ayıklar. |
| `business-analyst` | İş analizini yazar — Description'daki `## Problem / Business Value / Scope / Acceptance Criteria / Out of Scope`. |
| `technical-analyst` | `**[Technical Analysis]**` sentinel yorumunu yazar — yaklaşım, fizibilite, NFR'ler, bağımlılıklar, önerilen alanlar. |
| `project-manager` | Sprint temposunu yürütür — kapasite, iterasyon ataması, velocity. |
| `tech-lead` | Feature'ları iş-birimlerine ayrıştırır, her birimin `**[Canonical Brief]**`'ini yazar, proje wiki'sinin (`Architecture/`, `Conventions/`, ADR'ler) sahibidir ve **tek review kapısıdır** — her PR'ı gözden geçirir ve yeşilse tamamlar (= merge) ve Done'a set eder. |
| `tester` | Bağımsız Level-2 doğrulama — niyeti yeniden türetir, doğru yüzeyde test-gate'leri koşar, kanıt ekler, bir verdict yayınlar. |
| `developer` | İş-birimi başına spawn edilen, stack'ten bağımsız, dinamik bir worker; etiketli `area:<name>` bilgi-paketini yükler ve birimi implement eder. |

Belirli bir stack için bir **software team**, jenerik `developer`'ın yüklediği alan-anahtarlı bilgi
paketlerinden (`packs/<area>/`) ibarettir — M1 "knowledge-as-data" dikişi; böylece bir React ya da .NET
ekibi yeni bir ajan olmadan takılır.

## Seremoniler

Sprint, her biri doğru rol olarak davranan, senin çağırdığın skill'lerle işler:

```bash
/delivery-init      # backend'i seç (azure | github) + projenin koordinatlarını + metodolojiyi bağla
/kickoff            # intake + business-analyst Epic/Feature backlog'unu şekillendirir
/refine             # technical-analyst + tech-lead Feature'ları brief'li iş-birimlerine ayrıştırır
/sprint-plan        # project-manager sprint'in birimlerini kapasiteye göre seçer
/sprint-start       # iş-birimi DAG'ını materialize et → motora devret
/sprint-review      # velocity, review sonucu wiki sayfası, sprint kapanışı
/request            # (her an) proje-ortası istek → triyaj → fizibilite → dürüst PO kapısı → kabul/ertele/ret
```

Metodoloji **kod değil, config'tir**: `methodology.json` (v1'de Scrum) seremonilerin okuduğu tempoyu
bildirir — bakımı gereken bir workflow motoru yoktur.

## Motor — `atl work dispatch`

`/sprint-start`, seçilen birimleri bir `.delivery/plan.json` bağımlılık DAG'ına materialize eder, sonra
**deterministik Go motoru** `atl work dispatch` devralır. **Sıfır LLM context tutar ve sıfır Azure çağrısı
yapar**: hazır birimleri bir eşzamanlılık sınırına kadar admit eder ve her biri için **tek bir git
worktree'de üç izole `claude -p` worker'dan oluşan bir pipeline** spawn eder —

```
developer  →  tester  →  tech-lead
(implement    (Level-2     (review → vote →
 + PR aç)      verify)      PR-complete = dev'e merge → Done)
```

Motor, bir worker'ın temiz çıkışında stage'i ilerletir, tech-lead'in merge'inin `dev`'e indiğini saf bir
git okumasıyla doğrular (worker'ın exit code'una asla güvenmez), worktree'yi geri alır ve DAG'ı doldurur.
Stall eden ya da çöken bir worker geri alınıp bir kez retry edilir, sonra mark-blocked olur — bunu
`/sprint-review`'ın backend'e yansıttığı (`blocked` tag'i ya da label'ı + tanı yorumu) ve temizlediği
kalıcı bir rapor. Her worker tracker'a yalnız motorun ona bağladığı şey üzerinden erişir — Azure
backend'inde proje-kapsamlı `azureDevOps` MCP, ya da GitHub backend'inde motorun enjekte ettiği bir
`GH_TOKEN` (`config.credential.ref`'ten çözümlenir) ile `gh` CLI — asla operatörün ortam MCP config'i ya
da kimlik bilgileri değil.

## Backend tek gerçek kaynaktır

Yerel bir veritabanı yoktur. **İş-öğeleri geçici yürütme durumudur** ve **kalıcı-bilgi deposu kalıcı
bilgiyi tutar** (ATL wiki/journal ayrımının backend'de yaşayan hali: Azure'da proje wiki'si, GitHub'da
repo-içi bir `docs/` ağacı). Her rol, backend'e tek bir belgelenmiş **sağlayıcıdan-bağımsız
operasyon-sözleşmesi** (`knowledge/backend-interface.md`) üzerinden erişir; bu, sağlayıcı başına bir
adaptör paketiyle bağlanır — `backends/azure/adapter.md` (`azureDevOps` MCP: iş-öğeleri için `wit_*`,
PR'lar için `repo_*`, bilgi için `wiki_*`, MCP'nin eksik olduğu tek operasyon için, kanıt ekleme, ince
bir REST carve-out ile) ya da `backends/github/adapter.md` (`gh` CLI: Issues, Projects v2, Pull Requests
ve repo-içi `docs/` deposu). İçerik **makine-bulunabilir sentinel'lerle** yerleştirilir: iş analizi Description'da,
`**[Technical Analysis]**` ve `**[Canonical Brief]**` yorumları her biri tam ilk satırıyla eşleşerek
("en yeni yorum" değil), alan bağlama `System.Tags: area:<name>` ile.

## İşi teslim etmek — iki-branch akışı

İş **`dev`**'e entegre olur (tech-lead her birimin PR'ını yeşilde tamamlar — platformun never-merge
kuralının kapsamlı istisnası) ve Product Owner onaylanmış bir sprint'i `dev`'den **`release`**'e promote
eder. Review **delivery-native**'dir: tech-lead adversarial review desenini (evidence gate + refute-to-keep)
doğrudan backend'in PR'ı üzerinde koşar — Azure'da `repo_*` thread'leri ve vote, GitHub'da `gh pr comment`
/ `gh pr review` — `/create-pr` değil.

## Neler geliyor

Tam rol-ajan organizasyonu, altı seremoni skill'i, `atl work dispatch` motoru, Azure DevOps ve GitHub
adaptör paketleriyle sağlayıcıdan-bağımsız backend arayüzü, bir Scrum `methodology.json`'ı ve dört-alanlı
bir referans paketi (web / mobile / api / go-cli). Otonom developer→tester→tech-lead döngüsü, canlı bir
Azure DevOps projesine karşı uçtan uca kanıtlanmıştır.

Ertelenenler (tasarım yakalandı, tetik-kapılı): Scrum ötesi **çoklu-metodoloji** desteği, jenerik
developer'ın **stack-özel override'ı**, **dinamik-kapasite** eşzamanlılığı, bir **hotfix akışı** ve
**device-farm** emulator'ları. **mobile-emulator** test hattı yapıldı ama canlı doğrulaması bir masaüstü
(GUI) oturumuna kapılı.

## Ayrıca bakın

- [`atl install`](/tr/cli/install) — bir takımın nasıl çözümlenip kurulduğu
- [Takımlar](/tr/teams/) — katalog ve ilk-parti yeniden inşa
- [Kavramlar: scope](/tr/guide/concepts#scope-global-and-project) — proje vs. global takımlar
