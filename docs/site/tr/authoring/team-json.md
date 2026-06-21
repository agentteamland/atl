# `team.json`

Her takım, kökünde bir `team.json` bulunan bir Git deposudur. Bu dosya tüm sözleşmedir: takımın adı, ne yayımladığı, neye bağlı olduğu ve varsayılan olarak nereye kurulduğu.

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

Bu kadarı kuruluma yeter. CLI manifest dosyasını çözümler, `agents/web-agent.md` (ya da `agents/web-agent/agent.md`) dosyasını ilgili kapsamın `.claude/agents/` dizinine kopyalar ve kurulumu kapsama özgü bir manifest dosyasına kaydeder.

## Tam alan başvurusu

| Alan | Tür | Zorunlu | Açıklama |
|---|---|---|---|
| `schemaVersion` | tam sayı | ✅ | Şu an `1`. Yalnızca manifest yapısında geriye dönük uyumsuz bir değişiklik olduğunda artırılır. |
| `name` | dize | ✅ | Takımın katalog adı. Küçük harf kebab-case. GitHub kullanıcı adınızla birleşerek `<handle>/<name>` kurulum referansını oluşturur. |
| `version` | semver dizesi | ✅ | SemVer 2.0.0 (`1.2.3`, `1.2.3-beta.1`). `atl update` güncelleme gerekip gerekmediğine bunu karşılaştırarak karar verir. |
| `description` | dize | ✅ | `atl search` çıktısında görünen tek cümlelik tanıtım. Kısa tut — katalog çıktısında tek satırdır. |
| `author` | nesne | — | `{ "name": "...", "url": "...", "email": "..." }`. **Nesne olmalıdır, dize değil** — `"Your Name <you@example.com>"` gibi düz bir dize ayrıştırmada başarısız olur. `author` verildiğinde yalnızca `name` zorunludur. |
| `license` | SPDX dizesi | — | `"MIT"`, `"Apache-2.0"` vb. Verilmezse `"MIT"` varsayılır. |
| `keywords` | dize[] | — | `atl search` eşleşmesi için. `["nextjs", "tailwind", "blog"]`. |
| `repository` | dize | — | Katalogda gösterilen takım kaynak URL'si. |
| `homepage` | dize | — | Belge / açılış URL'si. |
| `agents` | nesne[] | — | Her biri: `{ name, description }`. Adlar `agents/` altındaki dosya ya da dizinlerle eşleşmelidir. |
| `skills` | nesne[] | — | Her biri: `{ name, description }`. Adlar `skills/` altındaki dizinlerle eşleşmelidir. |
| `rules` | nesne[] | — | Her biri: `{ name, description }`. Adlar `rules/` altındaki dosyalarla eşleşmelidir. |
| `scope` | dize | — | Yayıncı varsayılan kurulum katmanı: `"project"`, `"global"` ya da `"both"`. Varsayılan `"project"`. Kullanıcı kurulum sırasında `--global` / `--project` ile her zaman geçersiz kılabilir. |
| `dependencies` | nesne | — | CLI'nin bu takımın yanına kurması gereken diğer takımlar için `team-name → version-constraint` eşlemesi. |
| `requires.atl` | dize | — | En düşük `atl` sürümü. Örneğin `">=2.0.0"`. |

::: tip Açıklamayı kısa tut
`description`, `atl search` çıktısında tek satır olarak gösterilir; uzun bir açıklama garip biçimde kırılır. Bir tanıtım cümlesini hedefle — paragraf değil.
:::

## Sürüm kısıtları {#version-constraints}

`dependencies` eşlemesi ve `requires.atl` alanı standart SemVer aralık sözdizimini kabul eder:

| Sözdizim | Anlamı |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — önerilen varsayılan) |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde) |
| `1.2.3` | Kesin sabitleme |
| `>=1.2.0` | Açık uçlu en düşük sürüm |

Caret (`^`) önerilen varsayılandır — yama ve küçük sürüm güncellemelerini alır, geriye uyumsuz ana sürüm artırımlarını engeller.

## Dizin sözleşmeleri

`atl`, paketlediğin dosyaları `team.json` dosyasını okuyarak ve eşleşen yolları arayarak keşfeder:

```
my-team/
├── team.json
├── agents/
│   ├── web-agent.md             ← basit ajan (tek dosya)
│   └── db-agent/
│       ├── agent.md             ← karmaşık ajan (children deseni)
│       └── children/
│           ├── migrations.md
│           └── rls.md
├── skills/
│   └── create-new-project/
│       └── skill.md
└── rules/
    └── commit-style.md
```

Yalnızca `agents/`, `skills/` ve `rules/` kurulabilir varlıklardır — Claude Code'un okuduğu dizinler bunlardır. Repodaki diğer her şey (`team.json`, `README`, `LICENSE`) geride kalır ve tüketicinin `.claude/` dizinine kopyalanmaz.

`team.json` içindeki her giriş (`agents[]`, `skills[]`, `rules[]` altında) diskte gerçek bir dosya ya da dizine karşılık gelmek zorundadır. Varlık bildiren ama hiçbirini göndermeyen bir takım kurulumda hata verir.

## Doğrulama

v2'de ayrı bir JSON Şeması dosyası ve şema doğrulama CI adımı yoktur. Doğrulama minimumdur ve CLI'nin kendisinde yaşar:

- `team.json` geçerli JSON olarak ayrıştırılabilmelidir.
- `name` alanı bulunmalıdır.
- Bildirilen `agents/`, `skills/`, `rules/` girişleri diskte var olmalıdır — `atl install`, kurulabilir varlık olmayan bir takımda hata verir.

Sözleşmenin tamamı budur. `atl install` takımını kabul ederse geçerlidir; yerel ya da CI'da çalıştırılacak başka bir şey yoktur.

## Sıradaki

- **[Bir takım oluşturma](./creating-a-team)** — adım adım.
- **[`atl install`](/tr/cli/install)** — bir takımın nasıl çözümlendiği ve kurulduğu.
