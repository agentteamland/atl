# Bir takım yazma

Bir takım, `atl install` ile kurulabilen, yeniden kullanılabilir bir AI **ajanı**, **becerisi** ve **kuralı** paketidir. Bu sayfa, boş bir dizinden kataloglanmış bir takıma kadar her şeyi adım adım anlatır.

## Bir takım nedir (ve ne değildir)?

Bir takım, yalnızca bir `team.json` dosyası ve birkaç Markdown içeren bir Git deposudur. `atl install <handle>/<team>` çalıştırdığında CLI, handle'ı GitHub tabanlı katalog (catalog) ile eşleştirir, depoyu geçici bir HTTPS tarball'ı olarak indirir ve takımın `agents/`, `skills/`, `rules/` içeriğini hedef `.claude/` dizinine kopyalar. Hepsi bu — eklenti sistemi yok, JavaScript çalışma zamanı yok, özel ikili yok. Her şey metin dosyaları ve kopyalardır.

Bir takımın içinde şunlar olabilir:

- **Tek bir ajan** (Claude'un izleyeceği yönergeleri içeren tek bir Markdown dosyası)
- **Bir ya da daha çok beceri** (Claude'un çağırabileceği eğik çizgili komutlar)
- **Kurallar** (her oturumda yüklenen, davranışı biçimlendiren yönergeler)
- **Yukarıdakilerin herhangi bir bileşimi**

Bir takım, **kataloglandığında** kurulabilir hâle gelir: herkese açık GitHub deposunu [`atl-team`](https://github.com/topics/atl-team) konusuyla etiketle ya da repodan [`atl publish`](/tr/cli/publish) çalıştır; oluşturulan dizin (index) bunu alır. Artık herkes `atl install <handle>/<team>` ile kurabilir — handle, reponun GitHub sahibidir. Kayıt defteri deposu ve gönderim PR'ı yoktur.

---

## Tam adım adım anlatım

Sıfırdan küçük gerçek bir takım kuralım. Bir `my-team` deposu oluşturacaksın, bir ajan ekleyeceksin ve onu yayımlamaya hazır hâle getireceksin.

### Adım 1 — Takım deposunu oluştur

```bash
mkdir ~/projects/my-team
cd ~/projects/my-team
git init -b main
```

Klasör adının takımın katalog adıyla aynı olması gerekmez — o aşağıdaki `team.json` dosyasında belirlenir.

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
  "rules": []
}
```

Tüm alanlar için: [team.json](./team-json).

**Dikkat edilecek tuzaklar:**

- `name`, takımın kısa adıdır. Bir kez belirlendiğinde değiştirme — kullanıcılar buna göre başvuracak. Kebab-case olmalıdır (küçük harfler, rakamlar, tireler).
- `version`, SemVer biçimindedir (major.minor.patch). Değişiklik yayımladığında artır — `atl update` bu alana bakarak çekim yapıp yapmayacağına karar verir.
- `author` bir **nesnedir**, dize değildir. En azından `{ "name": "Your Name" }`. `"author": "You"` gibi düz bir dize ayrıştırma hatası değildir — kurulum ayrıştırıcısı `author` diye bir alan modellemez, dolayısıyla sessizce yok sayılır — yine de açıklık ve gelecekteki uyumluluk için nesne biçimini kullan.
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

`agent.md`, giriş noktasıdır — Claude her çağrıda onu okur. Kısa tut. Ayrıntılı desenleri `children/*.md` dosyalarına koy; ajanın `## Knowledge Base` bölümü onlara bağlantı verir ve Claude gerektiğinde okur.

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

::: tip Derinlemesine
Çocuklar deseni, [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md) kuralında anlatılır; özeti [Children + learnings](/tr/guide/children-and-learnings) sayfasındadır. Ana fikir: `agent.md` kısa kalır, konuya özgü ayrıntı her konu bir dosyada olmak üzere `children/*.md` dosyalarına gider.
:::

### Adım 4 — Commit at

Çalışmanı commit'le — `atl` kurulumları commit'lenmiş ref üzerinden yapar, çalışma ağacından değil:

```bash
git add .
git commit -m "feat: initial team"
```

### Adım 5 — Başkalarının kurabilmesi için yayımla

`atl install` handle'ları katalog üzerinden çözer, dolayısıyla bir takımın handle ile kurulabilmesi için önce kataloglanması gerekir. Repoyu herkese açık bir GitHub deposuna push'la ve `atl-team` konusuyla etiketle:

```bash
# Hesabın ya da org'un altında herkese açık bir GitHub deposuna push'la:
gh repo create you/my-team --public --source=. --push

# Katalogun dizine alması için etiketle:
gh repo edit you/my-team --add-topic atl-team
```

Saatlik çalışan bir dizin işi (index job), herkese açık `atl-team` etiketli depoları keşfeder ve kataloğa karşı bir katalog yenileme PR'ı açar; bir maintainer bunu birleştirdiğinde takımın `you/my-team` olarak keşfedilebilir olur — yani anında değil, o inceleme için bir gecikme bekle.

### Adım 6 — Kur

```bash
mkdir /tmp/demo-app && cd /tmp/demo-app
atl install you/my-team
# → atl: installed you/my-team@0.1.0 at project scope

atl list
# project:
#   you/my-team@0.1.0

ls .claude/agents/web-agent/
# → agent.md
```

Çıktı buna uyuyorsa takımın kurulmuştur. Ajan artık `/tmp/demo-app/` içinde Claude tarafından kullanılabilir.

Takım, yayıncısının bildirdiği kapsamda (scope) kurulur (varsayılan project — bkz. [team.json](./team-json) içindeki `scope` alanı). Her kurulum için `--global` ya da `--project` ile geçersiz kılabilirsin.

### Adım 7 — Yinele

`~/projects/my-team/` altındaki dosyaları düzenle, `team.json` içindeki `version` alanını artır, sonra commit'le ve push'la. Katalog yeni sürüme göre yeniden dizinlenir ve takımı kurulu olan her proje `atl update` ile alır:

```bash
cd ~/projects/my-team
vim agents/web-agent/agent.md           # ya da herhangi bir düzenleme
# team.json içindeki "version" alanını artır, sonra:
git commit -am "tweak web-agent guidance"
git push

cd /tmp/demo-app
atl update
# → atl yayımlanmış sürümü yeniden çeker, değiştirilmemiş kopyaları yeniler
```

`atl update`, yerel olarak değiştirmediğin kopyaları yeniler; kendi düzenlemelerine dokunmaz.

::: warning Release'ler kurulumları sabitler
Reponun GitHub Release'leri varsa katalog, kurulumları en son release'in etiketine (tag) sabitler — gönderdiğin her sürüm için yeni bir GitHub Release oluştur (yalnızca bir git tag değil). `main`'e düz bir push yalnızca repoda hiç release yokken kurucularınıza ulaşır.
:::

### Adım 8 — (İsteğe bağlı) Beceri ve kural ekle

**Beceriler** eğik çizgili komutlardır. Her biri bir frontmatter ile birlikte `skills/<skill-name>/SKILL.md` dosyasına yazılır:

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

**Kurallar**, Claude'un davranışını biçimlendiren, her oturumda yüklenen Markdown dosyalarıdır. `rules/<rule-name>.md` konumuna koy:

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

Herhangi bir değişiklikten sonra — ajan, beceri ya da kural — sürümü artır, commit at, push'la; ardından `atl update` ile değişikliği al.

### Adım 9 — Sonraki adımlar

- **İskele becerisi ekle.** Takımın yeni proje açma amacındaysa bir `/create-new-project` becerisi ekle. Bkz. [İskele belirtimi](./scaffolder-spec).
- **Başka bir takıma bağımlı ol.** Takımın başka birinin takımı üzerine inşa ediliyorsa `team.json` içindeki `dependencies` altında bildir — `atl install`, bağımlılığı seninki ile birlikte indirir.
- **Doğrulanmış rozet kazan.** AgentTeamLand maintainer'larınca incelenen takımlar (ve `agentteamland/*` altındaki her şey) `atl search` çıktısında `[verified]` rozeti gösterir. Rozetin olmaması yalnızca takımın kendi kendine yayımlandığı anlamına gelir.

---

## Takım düzeni başvurusu

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
│   └── backend-agent/
│       ├── agent.md
│       └── children/ ...
│
├── skills/                        ← beceri başına bir dizin
│   ├── lint-page/
│   │   └── SKILL.md              ← frontmatter (name, description, argument-hint) + gövde
│   └── run-e2e/
│       └── SKILL.md
│
└── rules/                         ← kural başına bir .md (düz, dizin değil)
    ├── react-19-defaults.md
    └── file-naming.md
```

`atl`, takımın varlık dizinleri (`agents/`, `skills/`, `rules/` ve ayrıca `knowledge/`, `backends/`, `scripts/`, `packs/`) altındaki her dosyayı kullanıcının `.claude/` dizinine kopyalar. `team.json` içindeki `agents[]`/`skills[]`/`rules[]` dizileri katalog üst bilgisidir — takımı `atl search` çıktısında tanıtırlar, neyin kopyalanacağına karar vermezler. Yalnızca o varlık dizinlerinin dışındaki dosyalar (`team.json`, `README`, `LICENSE`) geride kalır.

---

## Kurulum arka planda nasıl çalışır?

`atl install you/my-team` çalıştırıldığında:

1. **Çöz.** Handle, GitHub tabanlı katalogda aranır (herkese açık `atl-team` etiketli depolardan üretilen dizin). Monorepo alt yolundan yayımlanan bir takım o alt yola; bağımsız bir takım kendi deposunun köküne çözülür.
2. **İndir.** Takım, geçici bir dizine ref-sabitli HTTPS tarball'ı olarak indirilir — `git` ikili dosyası gerekmez. Geçici dizin kurulumdan sonra silinir.
3. **Doğrula.** `atl`, `team.json` dosyasını ayrıştırır ve bir `name` alanı olduğunu doğrular. Bildirilen ajan/beceri/kuralları teker teker diske karşı kontrol etmez — kurulum yalnızca takım hiç varlık dosyası göndermiyorsa burada hata verir (`team ships no installable assets`).
4. **Yaz.** Ajanlar, beceriler ve kurallar kapsamın `.claude/` dizinine **kopyalanır** — global kurulum için `~/.claude`, proje kurulumu için `<proje>/.claude`.
5. **Kaydet.** `<katman>/.atl/installed/<handle>__<name>.json` konumundaki takıma özgü manifesto, kaynak ref ve dosya başına SHA-256 değerlerini kaydeder; `atl update`'in otomatik yenileme ve `atl doctor`'ın bütünlük denetimi bu verilere dayanır.

Kalıcı klonlama önbelleği yoktur, ayrı bir ATL varlık deposu da yoktur. Takım varlıkları `.claude/` altında yaşar; ATL'nin kendi durum verisi (katalog önbelleği, öğrenme kuyruğu, pin'ler, kurulum manifestoları) `~/.atl` ve `<proje>/.atl` altında yaşar.

---

## Sık karşılaşılan tuzaklar

**Takımı düzenledim ve `atl update` çalıştırdım ama etki yok**
→ Commit attın mı, sürümü artırdın mı, push'ladın mı? `atl update`, yayımlanmış sürümü çeker; commit'lenmemiş veya push'lanmamış düzenlemeler akmaz. Commit at + sürümü artır + push'la, sonra `atl update`.

**`atl install` "team not found" diyor**
→ Handle henüz katalogda yok. Depo herkese açık olmalı ve `atl-team` konusuyla etiketlenmiş olmalı (ya da `atl publish` çalıştırılmış olmalı). Neyin dizinde olduğunu `atl search` ile doğrula.

**Bir takımı temiz biçimde silmek istiyorum**
→ `atl remove you/my-team` çalıştırmak, takımın manifest kayıtlı dosyalarını kapsamdan kaldırır (varsayılan olarak project; global katman için `--global`) ve artık boş olan dizinleri temizler.

---

## Sıkça sorulan sorular

**Kullanmak için takımı bir yere push'lamak zorunda mıyım?**
Evet. `atl install`, handle'ları GitHub tabanlı katalog üzerinden çözer; dolayısıyla takımın `atl-team` etiketli herkese açık bir deposu olması (ya da `atl publish` çalıştırılmış olması) gerekir.

**Tek bir projede birden çok takım yan yana yaşayabilir mi?**
Evet — istediğin kadar kur. Her takımın öğeleri paylaşılan `.claude/` dizinine kopyalanır. İki takım aynı adda bir öğe bildirirse, ilk kurulumda en son kurulan takımın dosyası öncekini sessizce üzerine yazar — `atl` bu çakışma için şu an uyarı vermez. `atl update` (ve yeniden kurulum) sırasında ise fan-out (dağıtım) disiplini, çakışan bir yoldaki yerel olarak farklılaşmış dosyayı ezmek yerine korur; dolayısıyla orada yeni gelen kazanmaz. Belirsizliği önlemek için benzersiz ajan/beceri/kural adları tercih et.

**`atl` hangi Markdown biçimini kullanır?**
İsteğe bağlı YAML frontmatter ile düz Markdown. Claude Code'un ajan ve beceri biçimi yerel olarak desteklenir.

**Becerileri takımdan bağımsız sürümleyebilir miyim?**
Bugün hayır. Sürümleme takım düzeyindedir; `team.json` içindeki `version` alanı üzerinden yapılır.

**Boyut sınırları var mı?**
Sert sınır yoktur. Pratikte takım depoları 10 MB'ın altındadır. Büyük ikili dosyalar eklerseniz README'de belirtin ki kullanıcılar ne indirdiklerini bilsin.

---

## Ayrıca bkz.

- [team.json alan başvurusu](./team-json)
- [İskele belirtimi](./scaffolder-spec) — `/create-new-project` becerileri ekleme
- [`atl install`](/tr/cli/install) — tam CLI başvurusu
- [`atl publish`](/tr/cli/publish) — takımının biriken kazanımlarını upstream'e taşı
- [Children + learnings](/tr/guide/children-and-learnings) — ajan/beceri bilgi tabanı deseni
