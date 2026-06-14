# `team.json`

Her takım, kökünde bir `team.json` bulunan bir Git deposudur. Bu dosya tüm sözleşmedir: takımın adı, ne yayımladığı ve neye bağlı olduğu.

## En küçük örnek

```json
{
  "schemaVersion": 1,
  "name": "my-team",
  "version": "0.1.0",
  "description": "A starter team for small Next.js projects.",
  "author": { "name": "Your Name", "url": "https://github.com/you" },
  "agents": [
    { "name": "web-agent", "description": "Next.js + Tailwind reviewer and builder." }
  ]
}
```

Bu kadarı kuruluma yeter. CLI depoyu klonlar, `agents/web-agent.md` (ya da `agents/web-agent/agent.md`) dosyasını `.claude/agents/` altına kopyalar ve kurulumu kaydeder.

## Tam alan başvurusu

| Alan | Tür | Zorunlu | Açıklama |
|---|---|---|---|
| `schemaVersion` | tam sayı | ✅ | Şu an `1`. Geriye dönük uyumsuz şema değişikliklerinde artırılır. |
| `name` | dize | ✅ | Kayıt defteri adı. Küçük harf kebab-case, 3-40 karakter. Kayıt defterindeki dizin adıyla eşleşmelidir. |
| `version` | semver dizesi | ✅ | SemVer 2.0.0 (`1.2.3`, `1.2.3-beta.1`). |
| `description` | dize | ✅ | `atl search` çıktısında görünen tek cümlelik tanıtım. **`minLength: 10`, `maxLength: 200`.** 200 karakteri aşmak, kayıt defteri PR doğrulamasında en sık görülen başarısızlıktır — kısa tut. |
| `author` | nesne | — | `{ "name": "...", "url": "...", "email": "..." }`. Verildiyse `name` zorunludur. **Bir nesne olmalıdır, dize değil** — `"Your Name <you@example.com>"` gibi düz bir dize şema doğrulamasında başarısız olur. |
| `license` | SPDX dizesi | — | `"MIT"`, `"Apache-2.0"` gibi. Verilmezse `"MIT"` varsayılır. |
| `keywords` | dize[] | — | Arama eşleşmesi için. En çok 20 giriş; her biri ≤ 40 karakter. `["nextjs", "tailwind", "blog"]`. |
| `repository` | dize | — | Git URL'si (`https://`, `git@` ya da `ssh://`). Verilmezse CLI klonlama kaynağını kullanır. |
| `homepage` | dize | — | Belge / açılış URL'si. |
| `agents` | nesne[] | — | Her biri: `{ name, description, tags? }`. Adlar `agents/` altındaki dosya / dizinlerle eşleşmeli ve küçük harf kebab-case olmalıdır. |
| `skills` | nesne[] | — | Her biri: `{ name, description, tags? }`. Adlar `skills/` altındaki dizinlerle eşleşmelidir. |
| `rules` | nesne[] | — | Her biri: `{ name, description, tags? }`. Adlar `rules/` altındaki dosyalarla eşleşmelidir. |
| `extends` | dize | — | Üst takım belirtimi: `"name"` ya da `"name@version-constraint"`. Bkz. [Kalıtım](./inheritance). |
| `excludes` | dize[] | — | Miras alınan üst takımlardan dışarıda bırakılacak ajan / beceri / kural adları. |
| `dependencies` | nesne | — | CLI'nin yan yana kurması gereken ek takımlar için `team-name → version-constraint` eşlemesi. |
| `requires.atl` | dize | — | En düşük `atl` sürümü. Örneğin `">=1.0.0"`. |

::: tip Push'tan önce doğrula
`description.maxLength = 200` kısıtı ilk kez katkı verenlerin çoğunu çelmeler. Bir PR açmadan önce, kayıt defteri deposundaki `./scripts/validate.sh` betiğini ya da herhangi bir yerel Draft 2020-12 JSON Şeması doğrulayıcısını [`team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json) dosyasına karşı çalıştır. CI aynı denetimi yapar; yerelde başarısız olmak GitHub'da başarısız olmaktan çok daha hızlıdır.
:::

## Sürüm kısıtları {#version-constraints}

`extends` ve `dependencies` alanları standart SemVer aralık sözdizimini kabul eder:

| Sözdizim | Anlamı |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — önerilen varsayılan). |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde). |
| `1.2.3` | Kesin sabitleme. |
| `>=1.2.0` | Açık uçlu en düşük sürüm. |

Caret (`^`) önerilen varsayılandır — yama ve küçük sürüm güncellemelerini alır, geriye uyumsuz ana sürüm artırımlarını engeller.

## Dizin sözleşmeleri

`atl`, paketlediğin dosyaları `team.json` dosyasını okuyarak ve eşleşen yollara bakarak keşfeder:

```
my-team/
├── team.json
├── agents/
│   ├── web-agent.md             ← basit ajan (tek dosya)
│   └── db-agent/
│       ├── agent.md             ← karmaşık ajan (çocuklar deseni)
│       └── children/
│           ├── migrations.md
│           └── rls.md
├── skills/
│   └── create-new-project/
│       └── skill.md
└── rules/
    └── commit-style.md
```

`team.json` içindeki her giriş (`agents[]`, `skills[]`, `rules[]` altında) gerçek bir dosya ya da dizine karşılık gelmek zorundadır. Eksik girişler doğrulamada başarısız olur.

## CI'de doğrulama

Her takım deposu, her push ve PR'da `team.json` dosyasını [`agentteamland/core/schemas/team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json) şemasına göre doğrulayan bir GitHub Action ile gelir. Mevcut takımlardan birinden iş akışı dosyasını kopyalayarak bunu bedavaya kazanırsın.

## Sıradaki

- **[Bir takım yazma](./creating-a-team)** — adım adım.
- **[Kalıtım](./inheritance)** — `extends`, `excludes`, bastırma kuralları.
- **[Şema başvurusu](/tr/reference/schema)** — makine okunabilir sözleşme.
