# Takım deposu bakımı

Paylaşılan **`agentteamland/` herkese açık depolarındaki** değişiklikler için yönetişim — takım depoları (cli, core, brainstorm, rule, team-manager, software-project-team, design-system-team, registry, docs, starter-extended, create-project, workspace, .github). Kural, 2026-04-24 tarihinde tüm herkese açık depolara dal koruması eklenmeden önce var olan disiplin boşluğunu kapatır.

Dal koruması **güvenlik ağıdır** — organizasyondaki her herkese açık deponun `main` dalına yapılan doğrudan commit'leri reddeder. Bu kural ise **yöntemdir**: güvenlik ağını karşılayacak ve git günlüğünü okuyan bir sonraki bakımcıya yararlı olacak temiz bir değişikliği nasıl üretebileceğin.

Kanonik kuralın kendisi [`core/rules/team-repo-maintenance.md`](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md) dosyasında yaşar. Bu sayfa kullanıcıya yönelik özettir.

## Bu ne zaman geçerlidir?

Sen (ya da senin adına Claude) `~/.claude/repos/agentteamland/{team}/` altındaki önbelleklenmiş bir agentteamland deposundaki bir dosyayı her değiştirdiğinde. Hem takım depoları (software-project-team, design-system-team vb.) hem de global depolar (core, brainstorm, rule, team-manager, cli, docs, registry, workspace) buna dâhildir.

**Şunlar için geçerli DEĞİLDİR:**

- Senin kendi yerel projendeki `.claude/` dizini (bu proje belleğidir, paylaşılan değildir).
- `homebrew-tap` / `scoop-bucket` / `winget-pkgs` (goreleaser ile yönetilir; sürüm yayım hattı için doğrudan push'a izin verilir).

## Beş sabit adım

Paylaşılan depo değişikliklerinin tamamı aynı beş adımdan geçer. Adımlar sıralıdır — her biri bir öncekinin üstüne kurulur.

### 0. PR açmadan önce `/save-learnings` çalıştır

1-4. adımlardan önce, [`/save-learnings`](/tr/skills/save-learnings) komutunu **özellik dalının son commit'i** olarak çalıştır. Bu, yayımlanmak üzere olan işten birikmiş deneyimi yakalar ve aynı PR'ın içinde yolculuk etmesine olanak tanır.

Bu zamanlamanın nedeni:

- PR sınırı doğal bir billurlaşma anıdır — kararlar somutlaşır.
- `save-learnings`'in **mevcut PR'ın deposuna** dokunan çıktıları (agent.md güncellemeleri, doc-impact taslakları, yeni çocuk dosyaları, bilinen sorunlar girişleri) kendiliğinden özellik dalında commit'lenir ve PR'ın içinde yolculuk eder.
- İnceleme hem işi hem de çıkarılan deneyimi tek atomik birim olarak görür.

Çoklu depo uyarısı: `/save-learnings` çoğu zaman birden çok depoya yayılan çıktılar üretir. Yalnızca **mevcut PR'ın deposuna** dokunan çıktılar yolculuk eder; diğer depolara ait çıktıların kendi PR akışına ihtiyacı vardır.

Bu kural, satır içi `<!-- learning -->` işaretçileri ve `SessionStart` hook'uyla eşlidir (bkz. [learning-capture kuralı](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md)). PR zamanında çalıştırılan `/save-learnings`, yayımlanmak üzere olan işten billurlaşmış öğrenmeleri yakalar; *bir sonraki* oturumun `SessionStart` hook'u ise zaten yayımlanmış işten kalan işaretçileri (inceleme geri bildirimleri, çakışma çözümü içgörüleri, birleştirme sonrası keşifler) yakalar. Tamamlayıcıdırlar, gereksiz değildir — ikisini de tut.

### 1. Sürümü artır (takımlar için `team.json`; CLI için `internal/config.Version`)

Kesin SemVer'i izle:

| Artırma | Ne zaman | Örnek |
|---|---|---|
| **Yama** (0.4.1 → 0.4.2) | Hata düzeltmesi, API değişikliği yok, davranış ilan edilen biçime geri getirildi. | `fix(dst-new-ds): Q3 cap`. |
| **Küçük** (0.4.2 → 0.5.0) | Yeni beceri / ajan / kural / komut, geriye uyumlu. | `feat(core): new rule learning-capture`. |
| **Ana** (0.4.2 → 1.0.0) | Geriye uyumsuz: kaldırılmış / yeniden adlandırılmış komut, uyumsuz yapılandırma, kullanıcıların güvendiği bir davranış değişikliği. | `feat(cli)!: rename atl install-team → atl install`. |

CLI için sürüm `internal/config/config.go` içinde yaşar (goreleaser etiketleri üzerinden derleme zamanında ldflags ile bastırılır). Takımlar için sürüm `team.json` içindedir.

Bir davranış değişikliğini sürüm artırımı olmadan **asla** yayımlama — bu, `atl update`'in `X → Y` bildirimini sessizce kırar ve tüm güncelleme hattını anlamsızlaştırır.

#### Ne zaman ARTIRILMAMALIDIR?

Gerçekten yalnızca belge olan PR'lar (çeviriler, yazım hatası düzeltmeleri, README güncellemeleri, yorum değişiklikleri) bir davranış farkı taşımaz ve `atl update`'in `X → Y` bildiriminde görünmesi gerekmez. Eğer `main` üzerinde önceden var olan şema ihlalleri (genellikle 200 karakterlik açıklama tavanı) varsa, sürümü artırmak aynı PR'da bir kırpmayı zorunlu kılar ve kapsamı şişirir. Pragmatik yol:

1. Yalnızca belge olan PR'da **sürüm artırımını atla**.
2. **Atlamayı PR gövdesinde açıkça not et** — örneğin: "Sürüm artırımı: atlandı — `main` önceden var olan açıklama uzunluğu ihlalleri taşıyor; izleyen bir `chore: trim ...` PR'ında ele alınacak."
3. Sürüm artırımıyla birlikte **özel bir** `chore: trim team.json descriptions` PR'ı aç.

"Her değişiklik için sürüm artırımı" kuralı, kullanıcıların anlamlı davranış farklarını görmesini sağlamakla ilgilidir — kozmetik düzenlemelerde tören temposuyla değil.

#### `team.json` biçim sözleşmeleri

`team.json` dosyasını düzenlerken:

1. **Hoş biçimli, çok satırlı nesneler** (2 boşluklu girinti). Her ajan / beceri / anahtar sözcük kendi satırında.
2. **Tire (em-dash) karakterleri için Unicode kaçışı `—`** açıklama dizelerinde (gerçek `—` yerine).

Yeniden biçimlendir:

```bash
python3 -m json.tool team.json > tmp && mv tmp team.json
```

Bir özellik dalı ile `main` her ikisi de `team.json` dosyasını düzenleyip biçim üzerinde çakıştığında (örneğin yoğun biçim ile hoş biçim arasında), **`main`'in biçimini benimse ve özellik dalının içeriğini onun üstüne yeniden uygula.** Sözleşmeyle savaşma — onun içine kıvrıl.

#### Push'tan ÖNCE `team.json` doğrulaması — pazarlığa kapalı

Şema kısıtları ([`core/schemas/team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json) içinde tanımlı):

| Alan | Kural |
|---|---|
| `description` (üst düzey) | 10-200 karakter. |
| `agents[*].description`, `skills[*].description`, `rules[*].description` | en çok 200 karakter. |
| `name` (üst düzey + agents/skills/rules) | kebab-case deseni `^[a-z][a-z0-9-]*$`. |
| `keywords[*]` | 1-40 karakter, en çok 20 anahtar sözcük, benzersiz. |
| `version` | kesin SemVer `MAJOR.MINOR.PATCH`. |

200 karakterlik `description` üst sınırı üretimde üç kez tökezletti. Her seferinde düzeltme bir takip eden "açıklama kırp" commit'iydi. **Daha fazla yok.**

`team.json` dosyasına dokunan her push'tan önce:

```bash
~/.claude/repos/agentteamland/core/scripts/validate-team-json.sh path/to/team.json
```

Ya da `core` deposunun içinden:

```bash
./scripts/validate-team-json.sh team.json
```

Betik, yalnızca Python standart kütüphanesini kullanarak hızlı bir uzunluk denetimi yapar; ayrıca `ajv-cli` PATH üzerindeyse CI'nin çalıştırdığı tam `ajv validate` komutunu da çalıştırır — böylece GitHub Actions'ın yapacağı denetimle eşitlik elde edersin. Atlanan bir uzunluk denetimi nedeniyle CI'de başarısız olmak, 2 saniyelik yerel çalıştırmadan çok daha pahalıya mal olur.

`ajv` olmasa bile betiğin Python uzunluk denetimi tarihsel olarak hata veren kısıtı yakalar. Çalıştır. Daima.

### 2. Conventional commit biçimi

```
<type>(<scope>): <70 karakterin altında tek satırlık özet>

<gövde — değişikliğin NEDEN'i, NE'si değil (farkı diff gösterir)>
<bağlam — ihtiyacı hangi proje / oturum ortaya çıkardı>

<ayak — eş yazar, sorun başvuruları, geriye uyumsuzluk notları>
```

Türler: `fix`, `feat`, `docs`, `chore`, `style`, `refactor`, `test`, `perf`. Geriye uyumsuzluk için türden sonra `!` ekle: `feat(cli)!: …`.

Kapsam, değiştirilen alt modüldür (ajan adı, beceri adı, CLI komutu, depo alanı).

### 3. Gövdede "Discovered via" bağlamı

Paylaşılan bir depodaki bir düzeltme, başka bir proje üzerinde çalışırken bulunduysa o bağlamı **daima** yüzeye çıkar:

```
Discovered while scaffolding a design system for a downstream project.
The bug is not project-specific; every project running /dst-new-ds
hits the same wall.
```

Bu denetim izi, gelecekteki sana (ya da başka bir bakımcıya) güdülenmeyi belleğinden yeniden inşa etmek zorunda kalmadan anlama olanağı tanır. Takım deposu git günlüğü kendi kendini belgeler.

### 4. PR akışı (varsayılan, dal koruması ile zorunlu)

Tüm herkese açık `agentteamland/` depoları, `main` dalına birleştirme için bir pull request gerektirir. Doğrudan push'lar dal koruması tarafından reddedilir.

```bash
cd ~/.claude/repos/agentteamland/{team}
git checkout -b <fix|feat|chore>/<short-description>
# … değişiklik yap, sürümü artır …
git add <files>
git commit -m "<conventional message>"
git push -u origin <branch-name>
gh pr create \
  --title "<type>(<scope>): <summary>" \
  --body  "<aşağıdaki PR gövdesi şablonuna bak>"
```

**Kendi PR'larında `--assignee` ya da `--reviewer` EKLEME.** Mevcut tek bakımcılı kurulumda Claude, bakımcının GitHub hesabı üzerinden push yapar; bu da bakımcıyı kendiliğinden PR yazarı yapar:

- Yazar alanı, PR'ı bakımcının "Created by me" / "Involves me" panolarında zaten gösterir.
- Açık `--assignee @me` gereksizdir (yazar = atanan) ve "Assigned to me" kuyruğunu kirletir.
- GitHub, PR yazarından inceleme istenmesini engeller; bu nedenle kendi PR'larda `--add-reviewer mkurak` sessizce başarısız olur.

(Claude'un push'ları için ayrı bir bot hesabı kurulduğunda — yazar ≠ bakımcı olduğunda — `--reviewer mkurak` olası ve uygun hâle gelir.)

#### PR gövdesi şablonu

```markdown
## Summary
<Ne değişti ve neden — 2-4 madde>

## Discovered via
<Hangi proje / oturum / senaryo bunu ortaya çıkardı>

## Version bump
<sürüm: X.Y.Z → X.Y.Z+1> (patch | minor | major — gerekçe)

## Test plan
- [ ] <düzeltmenin işe yaradığını doğrulama yolu>
- [ ] <gerileme denetimi>
```

Tek bakımcılı akış için onaylar zorunlu değildir (sayı: 0) — PR, dış bir kapı olarak değil **ritüel ve denetim izi** olarak vardır.

## 🚫 PR birleştirme disiplini — mutlak, istisnasız {#pr-merge-discipline-absolute-no-exceptions}

**Claude pull request'leri asla birleştirmez.** Bu pazarlığa kapalıdır ve hiçbir kapsam sınırı yoktur.

Yasak, bir PR'ı `main` dalına indiren her eylemi kapsar:

- Herhangi bir biçimde `gh pr merge` (`--squash`, `--rebase`, `--merge`).
- `gh pr review --approve`.
- Herhangi bir MCP güdümlü tarayıcıyla "Merge pull request" tıklama.
- GitHub REST / GraphQL API üzerinden eşdeğer bir sunucu tarafı eylem.

Şu durumlarda bile:

- PR önemsiz olsa bile (tek satırlık yazım hatası, biçim düzeltmesi, kırık bağlantı).
- `required_approving_review_count` 0 olsa bile (tek bakımcı).
- PR'ı Claude'un kendisi yazmış olsa bile.
- Kullanıcı sohbette daha önce "push'la" ya da "yapalım" demiş olsa bile.
- Dal koruması yönetici atlatmasına izin veriyor olsa bile.
- Bakımcı ulaşılamaz ve acil bir yama acil görünüyor olsa bile.

→ Yanıt **yine de hayır**. Birleştirme insan inceleyiciye aittir. Bir şey gerçekten acilse PR URL'sini yüzeye çıkar ve kullanıcıya açıkça engelleyici olduğunu söyle — birleştirme düğmesine 10 saniyede basar.

### PR'larda neye izin var?

- `gh pr create` — PR açma.
- `gh pr edit` — başlık / gövdedeki yazım hatalarını düzeltme, etiket ekleme / kaldırma.
- `gh pr list` / `gh pr view` / `gh pr diff` / `gh pr checkout` — yalnızca okuma niteliğinde inceleme.
- `gh pr review --comment` — bir geri bildirim yorumu bırakma (ONAYLAMA, DEĞİŞİKLİK İSTEME).

### Neye izin yok?

- Birleştirme (yukarıya bak).
- Onaylama (`--approve`).
- Başkasının PR'ında değişiklik isteme (`--request-changes`).
- Kapatma (`gh pr close`) — yıkıcıdır; PR'ları yalnızca yazar ya da kullanıcı kapatır.
- Açık talimat olmadan kapanmış bir PR'ı yeniden açma.

### İstisna: araçlar üzerinden GitHub yerel auto-merge

**Tek dar bir istisna** vardır: kullanıcı, araçlar üzerinden auto-merge'i açıkça yetkilendirdiğinde — en sık [`/create-pr --auto-merge`](/tr/skills/create-pr) ile — Claude, GitHub'ın yerel auto-merge düzeneğini etkinleştirmek için `gh pr merge --auto --merge` komutunu çalıştırabilir.

Bu istisna sınırlıdır:

- **Bayrak aynı turda kullanıcıdan gelmek zorundadır.** "Önceden yetkilendirme" ya da "geniş yetkilendirme" sayılmaz. Kullanıcının `--auto-merge` (ya da eşdeğer açık talimat) yazması kapıdır.
- **`--auto` zorunludur.** Anında birleştiren her şey (`--auto` olmadan `gh pr merge`) yine yasaktır.
- **Dal korumasının denetim kapısı korunur.** GitHub, gerekli denetimlerin geçmesini bekler; denetimler başarısız olursa birleştirme gerçekleşmez. "İnceleme kapısı" CI'ye devredilir — atlanmaz.
- **`--approve`, `--auto` olmadan `--squash` ya da herhangi başka bir birleştirme türevi için istisna yoktur.** Yalnızca `gh pr merge --auto --merge` (ya da deponun ayarları öyle gerektiriyorsa `--auto --squash` / `--auto --rebase`) kapsamdadır.

Kullanıcı araçlar üzerinden açık bir auto-merge bayrağı geçirmediyse özgün yasak yürürlüktedir: PR URL'sini yüzeye çıkar ve dur.

### PR açtıktan sonra teslim

`gh pr create` başarılı olduktan sonra URL'yi yüzeye çıkar ve o PR'da dur:

> PR opened: https://github.com/.../pull/N — incelediğin ve birleştirdiğinde haber ver, sonra devam ederim.

CI'nin yeşil sonrası kendiliğinden birleştirmesi için bekleme. Kendini onaylama. "5 dakikadır bir şey olmadı" diye `gh pr merge`'i yeniden çağırma. Birleştirme eylemi, kullanıcının PR'ı incelediğine dair sinyalidir; o sinyali atlamak kapıyı yıkar.

## Kaçış kapakları (yalnızca doğrudan push, asla auto-merge)

### Doğrudan push için yönetici atlatması (yalnızca acil durum)

Dal koruması, `enforce_admins` `false` (varsayılanımız) olduğunda yöneticinin doğrudan push yapmasına izin verir. Bu, PR akışı atlatmasının **tek** biçimidir — ve bir commit push'lar, bir birleştirme değil. Yalnızca şu durumlarda kullan:

- Sürüm yayım hattını kıran sorun `brew upgrade atl` / `scoop install atl`'i engelliyor.
- Kamuya açık bir gerilemeyi durdurmak için bir geri alma dakikalar içinde inmek zorunda.
- Bakımcı açıkça talimat veriyor: "doğrudan push'la, PR yok."

Bunu kullanırken yine de:

- Sürümü artır.
- Conventional commit kullan.
- Bir geriye dönük değerlendirmeyle takip et: `chore(postmortem): ...` commit'i ya da issue'su.

Kullanıcı tarafının beklentisi: yönetici atlatması **onun** aracıdır. Claude bunu başlatmaz.

### Önemsiz değişiklikler de PR'dan geçer

En küçük değişiklikte bile — bir yazım hatası, kırık bir bağlantı, bir `gofmt` çalıştırması — yol şudur:

1. Özellik dalı → commit → push → `gh pr create`.
2. Kullanıcı inceler, birleştirme düğmesine basar.

"PR için fazla küçük" diye bir sınıf yoktur. PR töreni ucuzdur (30 saniyelik iş); inceleme kapısı, önemsiz görünüp gerçekte öyle olmayan hataları yakalar.

## Bu kuralın KAPSAMADIĞI şeyler

- **Özel proje depoları** — kendi projenin Git iş akışı sana aittir. Bu kural yalnızca `agentteamland/` herkese açık depoları içindir.
- **Sürüm yayım hattı depoları** (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) — goreleaser kendiliğinden push yapar; dal koruması bilinçli olarak uygulanmaz.
- **Etiket tabanlı sürümler** — `cli v0.2.1` etiketlenirken etiket push'u goreleaser'ı tetikler. Etiket oluşturmak için PR gerekmez (etiket zaten birleştirilmiş bir commit'i `main` dalında gösterir).

## İlgili

- [`/create-pr`](/tr/skills/create-pr) — bu disiplini otomatikleştiren beceri.
- [`/save-learnings`](/tr/skills/save-learnings) — Adım 0 çağrısı.
- [karpathy-guidelines](/tr/guide/karpathy-guidelines) — bu yönetişim kuralının kodlama-rehberi karşılığı.
- Kanonik kaynak: [`core/rules/team-repo-maintenance.md`](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md).

## Tarihçe

2026-04-24'ten önce, takım depolarına `/save-learnings` üzerinden yapılan yazımlar `main` dalına gelişigüzel bir commit mesajıyla doğrudan inebiliyordu. Bu, gerçek hata düzeltmelerinin hızla yayımlanmasına olanak tanıyordu ama sürüm artırımları sıkça unutuluyor (`atl update`'in fark bildirimleri kırılıyordu) ve commit mesajı disiplini klavyede o sırada kim varsa ona kalıyordu.

2026-04-24 tarihinde bakımcı organizasyondaki her herkese açık depoya dal koruması ekledi ve ilkeli bir iş akışı talep etti; bu kural o iş akışıdır. Doğrudan push reddedilir; PR töreni hafiftir (tek bakımcı için dış onay gerekmez) ama zorunludur — her takım deposu değişikliğinin bir sürüm artırımı, bir conventional mesaj ve bir "Discovered via" bağlamı taşımasını güvence altına alır.
