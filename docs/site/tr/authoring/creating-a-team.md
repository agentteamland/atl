# Bir takım yazma

Bir takım, `atl install` ile bir projeye kurduğun, yeniden kullanılabilir bir AI **ajanı**, **becerisi** ve **kuralı** paketidir. Bu sayfa, boş bir dizinden kurulu bir takıma kadar gerekli olan her şeyi anlatır — "Dizüstüne kuruyorum, henüz bir yere push'lamadım" durumu da dâhil.

## Bir takım nedir (ve ne değildir)?

Bir takım, yalnızca bir `team.json` dosyası ve birkaç Markdown içeren bir Git deposudur. `atl install` çalıştırdığında CLI o depoyu yerel önbelleğine klonlar ve içeriğini projendeki `.claude/` dizinine kopyalar. Hepsi bu — eklenti sistemi yok, JavaScript çalışma zamanı yok, özel ikili yok. Her şey metin dosyaları ve kopyalardır.

Bir takımın içinde şunlar olabilir:

- **Tek bir ajan** (Claude'un izleyeceği yönergeleri içeren tek bir Markdown dosyası).
- **Bir ya da daha çok beceri** (Claude'un çağırabileceği eğik çizgili komutlar).
- **Kurallar** (global yüklenen, davranışı biçimlendiren yönergeler).
- **Yukarıdakilerin herhangi bir bileşimi.**
- **Başka bir takımın uzantısı** (tüm parçalarını miras alır, belirli öğeleri bastırır ya da dışarıda bırakır) — bkz. [Kalıtım](./inheritance).

Bir takımı şuralardan kurabilirsin:

1. Herkese açık **AgentTeamLand kayıt defteri** (`atl install software-project-team`).
2. Herhangi bir **GitHub deposu** (`atl install agentteamland/starter-extended` ya da tam URL).
3. **Herhangi bir Git URL'si** — GitHub, GitLab, Bitbucket, kendi barındırılan (herkese açık ya da SSH anahtarıyla özel).
4. **Yerel dosya sistemin** (`atl install ./my-team` ya da mutlak yol) — uzak depo gerekmez; takım dizininde bir `git init` yeterlidir.

Sonuncusu, kendi takımını yazan çoğu kişinin ilk uzandığı yoldur. Aşağıda ayrıntılı olarak ele alınıyor.

---

## Bölüm 1 — Tam adım adım anlatım

Sıfırdan küçük gerçek bir takım kuralım. Makinende bir `my-team` dizini oluşturacaksın, bir ajan ekleyeceksin ve onu hiçbir Git sunucusuna push'lamadan bir test projesine kuracaksın.

### Adım 1 — Takım dizinini oluştur

```bash
mkdir ~/projects/my-team
cd ~/projects/my-team

git init -b main                    # atl kurulum hattı bir Git deposu gerektirir
```

Konum önemli değil. Klasör adının takımın kayıt defteri adıyla aynı olması da gerekmez — o aşağıdaki `team.json` dosyasında belirlenir.

### Adım 2 — `team.json` yaz

Bu, takımın manifesto dosyasıdır. En küçük geçerli hâli:

```json
{
  "schemaVersion": 1,
  "name": "my-team",
  "version": "0.1.0",
  "description": "Opinionated setup for Next.js + Tailwind projects.",
  "author": { "name": "Your Name", "url": "https://github.com/you" },
  "license": "MIT",
  "keywords": ["nextjs", "tailwind", "typescript"],
  "agents": [
    { "name": "web-agent", "description": "Reviews and builds Next.js pages." }
  ],
  "skills": [],
  "rules": [],
  "extends": null,
  "excludes": []
}
```

Tüm alanlar için: [team.json](./team-json).

**Dikkat edilecek tuzaklar:**

- `name`, **kayıt defterindeki kısa addır**. Bir kez belirlendiğinde değiştirme — kullanıcılar buna göre başvuracak. Kebab-case olmalıdır (küçük harfler, rakamlar, tireler).
- `version`, SemVer biçimindedir (major.minor.patch). Yerelde yinelerken bile değişiklik yayımladığında artır — `atl update` çekim yapıp yapmayacağına buna göre karar verir.
- `author` bir **nesnedir**, dize değildir. En azından `{ "name": "Your Name" }`. Kayıt defteri başvuruları ayrıca `url` ister.
- `agents`, **üst bilgi** dizisidir, ajan içeriği değildir. Asıl ajan Markdown'ı `agents/<name>/agent.md` altında yaşar (bkz. Adım 3).

### Adım 3 — Ajanını ekle

`team.json` dosyasının bildirdiği her ajan, `agents/` altında **çocuklar deseniyle** bir dizine ihtiyaç duyar:

```
my-team/
├── team.json
└── agents/
    └── web-agent/
        ├── agent.md              ← kısa: kimlik, kapsam, ilkeler (<300 satır)
        └── children/             ← isteğe bağlı: derinlemesine konular
            ├── routing.md
            ├── data-fetching.md
            └── testing.md
```

`agent.md`, giriş noktasıdır — Claude her çağrıda onu okur. Kısa tut. Ayrıntılı desenleri `children/*.md` dosyalarına koy; ajanın `## Knowledge Base` bölümü onlara bağ verir ve Claude gerektiğinde okur.

En küçük `agent.md`:

```markdown
---
name: web-agent
description: "Reviews and builds Next.js pages."
---

# Web Agent

## Identity
I build and review Next.js pages for this project.

## Area of Responsibility (Positive List)
I ONLY touch:
- `app/` — Next.js App Router pages + layouts + routes
- `components/` — shared UI primitives
- `lib/` — data-fetching + utility functions

I do NOT touch:
- `api/` — that's the backend's concern
- Build config (`next.config.js`, `tsconfig.json`) without explicit approval

## Core Principles
1. Server components by default; client components only when interactive.
2. Co-locate styles with their component; no global CSS.
3. Loading UI for every async boundary.
```

İşte çalışan bir ajan. Ajan büyüdükçe `children/` dizinine daha fazla ayrıntı ekle.

> 📘 **Derinlemesine:** çocuklar deseni, `agentteamland/core` deposundaki [agent-structure kuralında](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md) anlatılır. Ana fikir: `agent.md` kısa kalır, konuya özgü ayrıntı her konu bir dosyada olmak üzere `children/*.md` dosyalarına gider.

### Adım 4 — Commit at

`atl`'nin kurulum hattı arka planda `git clone` kullanır; bu nedenle takım dizini en az bir commit'i olan bir Git deposu olmalıdır:

```bash
git add .
git commit -m "feat: initial team"
```

Bu noktada uzak depo gerekmez.

### Adım 5 — Test projesine kur

```bash
mkdir /tmp/demo-app && cd /tmp/demo-app
atl install ~/projects/my-team         # mutlak yol ✓
# ya da:  atl install ../my-team       # göreli yol ✓
# ya da:  atl install file:///Users/you/projects/my-team   # açık file:// URL'si ✓

atl list
# → my-team@0.1.0    (effective: 1 agents, 0 skills, 0 rules)

ls -la .claude/agents/
# → web-agent.md → ~/.claude/repos/agentteamland/my-team/agents/web-agent/agent.md
```

Çıktı buna uyuyorsa takımın kurulmuştur. Ajan artık `/tmp/demo-app/` içinde Claude tarafından kullanılabilir.

> 💡 **Özel takımımın önbellek yolunda neden "agentteamland" var?** `atl`, takımın nereden geldiğine bakmaksızın tek bir paylaşılan yerel önbellek dizini kullanır. Özel takımın, herkese açık olanların yanında `~/.claude/repos/agentteamland/` altında yaşar; onları ayıran `team.json` içindeki `name` alanıdır. Bu, takımının organizasyona push'landığı ya da paylaşıldığı anlamına GELMEZ — yalnızca bir önbellek sözleşmesidir.

### Adım 6 — Yinele

`~/projects/my-team/` altındaki dosyaları düzenle, commit at (`atl` çalışma ağacını değil commit'leri okur), ardından kurulu takımı yenile:

```bash
cd ~/projects/my-team
vim agents/web-agent/agent.md           # ya da herhangi bir düzenleme
git commit -am "tweak web-agent guidance"

cd /tmp/demo-app
atl update my-team
# → atl yeniden çekim yapar, kopyaları yeniler
```

Gidiş-dönüş yaklaşık 1 saniye sürer. Test projesine karşı hızla yinelersin.

### Adım 7 — (İsteğe bağlı) Beceri ve kural ekle

**Beceriler** eğik çizgili komutlardır. Her biri bir frontmatter ile birlikte `skills/<skill-name>/skill.md` dosyasına yazılır:

```markdown
---
name: lint-page
description: "/lint-page <path> — run the project's lint config against a Next.js page file."
argument-hint: "<path-to-page>"
---

# /lint-page Skill

## Purpose
Lint a single Next.js page file using the project's ESLint + Prettier.

## Flow
1. Validate the path exists and matches `app/**/*.tsx` or `pages/**/*.tsx`.
2. Run `npm run lint -- --file <path>`.
3. Parse the output; if violations exist, print them with file:line:column citations.
4. Offer to auto-fix where safe.
```

`team.json` dosyasında bildir:

```json
"skills": [
  { "name": "lint-page", "description": "/lint-page <path> — run lint against a page file." }
]
```

**Kurallar**, Claude'un davranışını biçimlendiren, global olarak yüklenen Markdown dosyalarıdır. `rules/<rule-name>.md` konumuna koy:

```markdown
# React 19 defaults

- Server components unless interactivity is needed
- Never use `"use client"` at the top of a shared lib
- `useActionState` replaces manual form-state boilerplate
```

Bildir:

```json
"rules": [
  { "name": "react-19-defaults", "description": "Default to server components; avoid client boundary creep." }
]
```

Herhangi bir değişiklikten sonra — ajan, beceri ya da kural — commit at ve test projesinde `atl update` çalıştırarak değişikliği al.

### Adım 8 — Sonraki adımlar

Takımın yerelde çalıştığına göre:

- **Özel olarak tut.** Sonsuza dek bu şekilde kullanabilirsin — `atl install`'u yerel yola yönlendir ya da özel bir Git sunucusuna push'la (GitHub özel deposu, GitLab, kendi barındırılan Gitea). Başvuru gerekmez.
- **Bir takımla paylaş.** Özel bir depoya push'la ve takımdaşlarına URL'yi ver: `atl install git@github.com:your-org/your-team.git`. SSH üzerinden kurarlar (kayıt defteri işin içinde değildir).
- **Var olan bir takımı genişlet.** Bkz. [Kalıtım](./inheritance) — `"extends": "software-project-team@^1.1.0"` bildir; o takımın ajanlarını miras al, belirli öğeleri bastır ya da dışarıda bırak.
- **Herkese açık kayıt defterine başvur.** Yalnızca başkalarının kısa adıyla bulmasını istiyorsan. Bkz. [Kayıt defteri başvurusu](./registry-submission).
- **Bir iskele becerisi ekle.** Takımın yeni proje açma amacındaysa bir `/create-*` becerisi ekle. Bkz. [İskele belirtimi](./scaffolder-spec).

---

## Bölüm 2 — Kurulum kipleri

Hepsi çalışır. Takımın nerede yaşadığına uygun olanı seç.

### Kayıt defteri (herkese açık, doğrulanmış)

```bash
atl install software-project-team
```

Kısa ad [herkese açık kayıt defterinde](https://github.com/agentteamland/registry) aranır. Yalnızca başvurusu yapılmış ve doğrulanmış takımlar için işe yarar.

### GitHub `owner/repo` kısa biçimi

```bash
atl install agentteamland/starter-extended
```

`https://github.com/agentteamland/starter-extended.git` adresine genişler. Kayıt defterinde olmayan herkese açık GitHub takımları için kullanışlıdır.

### Tam Git URL'si (herhangi bir barındırıcı, herkese açık ya da özel)

```bash
atl install https://github.com/you/your-team.git      # herkese açık, HTTPS
atl install git@github.com:you/your-team.git          # özel, SSH
atl install ssh://git@gitlab.com/you/your-team.git    # GitLab SSH
atl install https://gitea.example.com/you/team.git    # kendi barındırılan Gitea
```

Özel depolar için barındırıcıda Git kimlik bilgileri / SSH anahtarları kurulu olmalıdır (`atl` `git clone`'a kabuk üzerinden devreder, o da Git yapılandırmanı kullanır). `atl`'ye özgü bir kimlik doğrulama yoktur.

### Yerel dosya sistemi (uzak depo gerekmez)

```bash
atl install ~/projects/my-team                   # mutlak ya da ev dizini yolu
atl install ./my-team                            # göreli
atl install file:///Users/you/projects/my-team   # açık file:// URL'si
```

Gereksinimler:

- Hedef, kökünde `team.json` bulunan bir dizin olmalı.
- Dizin, en az bir commit'i olan bir Git deposu olmalı (`git init` + `git add . && git commit` yeterli).

Uzak push gerekmez. Yerelde yineleme, özel takımlar ya da henüz paylaşmaya hazır olmadığın takımlar için idealdir.

> ⚠️ **Sürüm:** yerel dosya sistemi kurulumu `atl` ≥ 0.1.4 gerektirir. Daha eski sürümler yalnızca URL kabul ediyordu.

---

## Bölüm 3 — Takım düzeni başvurusu

```
my-team/
├── team.json                      ← manifesto (zorunlu)
├── README.md                      ← takım belgeleri (kuvvetle önerilir)
├── LICENSE                        ← genellikle MIT
│
├── agents/                        ← ajan başına bir dizin
│   ├── web-agent/
│   │   ├── agent.md              ← kısa: kimlik + kapsam + ilkeler + Knowledge Base dizini
│   │   └── children/             ← isteğe bağlı: derinlemesine konular
│   │       ├── routing.md
│   │       ├── data-fetching.md
│   │       └── testing.md
│   └── api-agent/
│       ├── agent.md
│       └── children/ ...
│
├── skills/                        ← beceri başına bir dizin
│   ├── lint-page/
│   │   └── skill.md              ← frontmatter (name, description, argument-hint) + gövde
│   └── run-e2e/
│       └── skill.md
│
├── rules/                         ← kural başına bir .md (düz, dizin değil)
│   ├── react-19-defaults.md
│   └── file-naming.md
│
├── schemas/                       ← isteğe bağlı: takımının doğruladığı JSON şemalar
│   └── config.schema.json
│
└── .github/workflows/
    └── validate.yml               ← her push'ta team.json'u şemayla doğrula
```

`team.json` tarafından listelenen `agents/`, `skills/` ve `rules/` altındaki her dosya, kullanıcı kurulum yaptığında onun `.claude/` dizinine bir kopya olarak gelir. Listelenmeyen dosyalar yok sayılır.

---

## Bölüm 4 — CI'de şema doğrulaması (önerilir)

Bunu `.github/workflows/validate.yml` dosyasına yerleştir; bozuk bir `team.json` kuruluma kadar gitmeden yakalanır:

```yaml
name: Validate team.json
on:
  push:
  pull_request:

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - run: npm install -g ajv-cli

      - name: Download schema
        run: curl -sSfL https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json -o team.schema.json

      - name: Validate
        run: ajv -s team.schema.json -d team.json --strict=false
```

İsteğe bağlıdır ama `team.json` hatalarının çoğunu kurulumda değil PR sırasında yakalar.

---

## Bölüm 5 — Özel takım iş akışları

Üç tür "özel takım" vardır ve her birinin en temiz yolu farklıdır:

### 🟢 (a) Tümüyle yerel, yalnızca ben

Tek kullanıcı sensin. Takım dizüstünde yaşıyor.

```bash
# Yaz:
mkdir ~/projects/my-team && cd ~/projects/my-team
git init -b main
vim team.json agents/main-agent/agent.md
git add . && git commit -m "init"

# Herhangi bir projeye kur:
cd ~/projects/my-real-app
atl install ~/projects/my-team

# Yinele (düzenle → commit → yenile):
cd ~/projects/my-team
vim agents/main-agent/agent.md
git commit -am "tighter scope"

cd ~/projects/my-real-app
atl update my-team
```

Tüm iş akışı budur. Uzak depo işin içinde değildir.

### 🟡 (b) Birkaç takımdaşla paylaşılan

2-10 kişinin kurmasını istersin, ama herkese açık OLMASIN.

```bash
# Özel bir GitHub (ya da GitLab / Gitea) deposuna push'la:
gh repo create you/your-team --private --source=. --push

# Takımdaşlar SSH ile kursun (SSH anahtarları depoya eklenmiş olmalı):
atl install git@github.com:you/your-team.git

# Güncellemeler:
atl update your-team
```

Kayıt defteri başvurusu yok; yabancılar tarafından bulunamaz. SSH her güncel Git barındırıcısında çalışır.

### 🔵 (c) Şirket içi / kurumsal, kendi barındırılan Git sunucusunun arkasında

(b) ile aynı; URL kendi barındırdığın Git'i gösterir:

```bash
atl install git@git.your-company.com:platform/your-team.git
```

`atl` barındırıcıyı umursamaz; `git clone` kimlik doğrulamayı senin yapılandırdığın SSH anahtarları ya da kimlik yardımcılarıyla halleder.

### 🟣 (d) Tüm dünyaya açık

Herkesin `atl install your-team` diyebilmesi için [AgentTeamLand kayıt defterine](https://github.com/agentteamland/registry) gönder. Bkz. [Kayıt defteri başvurusu](./registry-submission).

---

## Bölüm 6 — Sık karşılaşılan tuzaklar

**`atl install ./my-team` "could not find team" diyor**
→ `./my-team` yerel yol olarak tanınmadı. Şunu denetle: bu bir dizin mi? Kökünde `team.json` var mı? `atl` sürümü ≥ 0.1.4 mü? (`atl --version`).

**`Error: agent source missing: .../agents/foo/agent.md`**
→ `team.json` dosyan `agents: [{"name": "foo"}]` olarak listeliyor ama dosya sisteminde `agents/foo.md` (düz) var, `agents/foo/agent.md` (çocuklar deseni) yok. `atl`, çocuklar deseni yapısını gerektirir.

**`Error: parse team.json: json: cannot unmarshal string into Go struct field TeamManifest.author`**
→ `author` bir nesne olmalı, dize değil. `"author": "You"` yerine `"author": { "name": "You" }`.

**Takımı düzenledim, `atl update` çalıştırdım ama etki yok**
→ Commit attın mı? `atl update` Git üzerinden çekim yapar; commit'lenmemiş düzenlemeler akmaz. Takıma commit at, ardından `atl update`.

**Bir takımı temiz biçimde silmek istiyorum**
→ Projede `atl remove my-team` çalıştırmak `.claude/` dizinindeki kopyaları kaldırır ama önbelleklenmiş depoyu korur. Önbelleği de silmek için: `rm -rf ~/.claude/repos/agentteamland/my-team`.

**Takım `extends` kullanıyor; üst takımı yerelde değiştirdim ama `atl install` yanlış üst takımı çekiyor**
→ `extends`, alt takımın `team.json` dosyasında yazılı olan kayıt defteri / URL belirtimine göre çözülür. Alt takımın `extends` alanı bir yerel yola işaret etmedikçe yerel üst takımını okumaz. Tümüyle yerel zincirler için `"extends": "/abs/path/to/parent-team"` gibi bir yerel yolla sabitle.

---

## Bölüm 7 — Sıkça sorulan sorular

**Kullanmak için takımı bir yere push'lamak zorunda mıyım?**
Hayır. `atl install`'u yerel yoluna yönlendir (atl ≥ 0.1.4). Yerel Git deponda commit'lerin olması yeterlidir.

**Kayıt defterine başvurmak zorunda mıyım?**
Hayır. Kayıt defteri yalnızca kısa adla bulunabilirlik içindir. Çoğu özel / kurumsal takım hiç başvurmaz.

**Tek bir projede birden çok takım yan yana yaşayabilir mi?**
Evet — `atl install a && atl install b && atl install c`. Her takımın öğeleri paylaşılan `.claude/` dizinine kopyalanır. Adlar çakışırsa `atl` seni uyarır (alt takım üst takımı bastırır; sonraki kurulumlar öncekileri bastırır).

**`atl` hangi Markdown biçimini kullanır?**
İsteğe bağlı YAML frontmatter ile düz Markdown. Claude'un ajan ve beceri biçimi yerel olarak desteklenir.

**Becerileri takımdan bağımsız sürümleyebilir miyim?**
Bugün hayır. Sürümleme takım düzeyindedir; `team.json.version` üzerinden yapılır. Beceri başına sürümleme önerilen bir gelecek özelliğidir.

**Boyut sınırları var mı?**
Sert sınır yoktur. Pratikte takım depoları 10 MB'ın altındadır. Büyük ikili dosyalar (CSS paketleri, SVG sprite'lar) gömerseniz README'de söyleyin ki kullanıcı ne çektiğini bilsin.

---

## Ayrıca bkz.

- [team.json alan başvurusu](./team-json).
- [Kalıtım](./inheritance) — var olan bir takımı genişletme.
- [İskele belirtimi](./scaffolder-spec) — `/create-new-*` becerileri ekleme.
- [Kayıt defteri başvurusu](./registry-submission) — herkese açık yayımlama.
- [`atl install` komutu](../cli/install) — tam CLI başvurusu.
- [Başvuru şeması](../reference/schema) — `team.json` için JSON Şeması.
