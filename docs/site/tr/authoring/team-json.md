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
| `author` | nesne | — | Kurulum ayrıştırıcısının şu an okumadığı isteğe bağlı üst veri. Verilirse `{ "name": "...", "url": "...", "email": "..." }` nesnesi geleneksel biçimdir; düz bir dize de kabul edilir (sessizce yoksayılır), reddedilmez. |
| `license` | SPDX dizesi | — | `"MIT"`, `"Apache-2.0"` vb. Geleneksel üst veri — CLI ve katalog onu okumaz. Depoda yanına bir LICENSE dosyası koyun. |
| `keywords` | dize[] | — | `atl search` eşleşmesi için. `["nextjs", "tailwind", "blog"]`. |
| `repository` | dize | — | Takımın kaynak URL'si. Geleneksel üst veri — katalog, kaynak depoyu bu alandan değil keşfedilen GitHub deposunun kendisinden türetir. |
| `homepage` | dize | — | Belge / açılış URL'si. |
| `agents` | nesne[] | — | Her biri: `{ name, description }`. Adlar `agents/` altındaki dosya ya da dizinlerle eşleşmelidir. |
| `skills` | nesne[] | — | Her biri: `{ name, description }`. Adlar `skills/` altındaki dizinlerle eşleşmelidir. |
| `rules` | nesne[] | — | Her biri: `{ name, description }`. Adlar `rules/` altındaki dosyalarla eşleşmelidir. |
| `scope` | dize | — | Yayıncı varsayılan kurulum katmanı: `"project"`, `"global"` ya da `"both"`. Varsayılan `"project"`. Kullanıcı kurulum sırasında `--global` / `--project` ile her zaman geçersiz kılabilir. |
| `dependencies` | nesne | — | CLI'nin bu takımın yanına kurması gereken diğer takımlar için `team-name → version-constraint` eşlemesi. |
| `requires.atl` | dize | — | Bildirilen en düşük `atl` sürümü. Örneğin `">=2.0.0"`. Geleneksel üst veri — kurulum ayrıştırıcısı şu an bunu dayatmaz. |
| `capabilities` | nesne | — | Platformun becerilerinin (kurulum ayrıştırıcısının değil) okuduğu isteğe bağlı sözleşmeler. `capabilities.review: "<agent>"`, [`/create-pr`](/tr/skills/create-pr)'in bu takımın uzman gözden geçireni olarak başlattığı ajanı adlandırır; `capabilities.profile`, profil katmanı sağlayıcı/tüketici rolünü bildirir ([profile-team](/tr/teams/profile-team)'e bakın). |
| `backends` | dize[] | — | `backends/<name>/` altında arka uca özel bağdaştırıcı paketleri gönderen takımlar için (ör. delivery-team'in `["azure", "github"]` değeri): takımın hangi arka uçları desteklediğini bildirir. Bugün yalnızca bilgilendirme amaçlıdır — kurulum ayrıştırıcısı bunu okumaz. |

::: tip Açıklamayı kısa tut
`description`, `atl search` çıktısında tek satır olarak gösterilir; uzun bir açıklama garip biçimde kırılır. Bir tanıtım cümlesini hedefle — paragraf değil.
:::

## Sürüm kısıtları {#version-constraints}

`dependencies` değerleri ve `requires.atl`, gelenek gereği standart SemVer aralık sözdizimiyle yazılır:

| Sözdizim | Anlamı |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — önerilen varsayılan) |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde) |
| `1.2.3` | Kesin sabitleme |
| `>=1.2.0` | Açık uçlu en düşük sürüm |

Caret (`^`) geleneksel öneridir — anlamca yama ve küçük sürüm güncellemelerini alır, geriye uyumsuz ana sürüm artırımlarını engeller. Ancak bugün CLI bu aralıkları değerlendirmez: `atl install` her bağımlılığı adına göre çözümler ve katalogdaki mevcut sürümü kurar, `requires.atl` de dayatılmaz. Yine de bunları bildirin — niyeti belgelerler ve aralık dayatması manifest değişikliği olmadan gelebilir.

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
│       └── SKILL.md
└── rules/
    └── commit-style.md
```

Kurulabilir varlık dizinleri şunlardır: `agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/` ve `packs/` (`teampkg.AssetDirs` kümesi). `agents/`/`skills/`/`rules/` Claude Code'un doğrudan okuduğu dizinlerdir; `knowledge/`/`scripts/`/`packs/` ise takımın çalışma zamanı referans belgelerini, yardımcı betiklerini ve alan paketlerini taşır; `backends/` ise takımın arka uca özel bağdaştırıcı sözleşmelerini taşır (ör. delivery-team'in `backends/{azure,github}/` dizini). Diğer her şey (`team.json`, `README`, `LICENSE`) geride kalır.

Bir takımın bir varlık dizini altında en az bir dosya göndermesi gerekir, yoksa `atl install` başarısız olur (`team ships no installable assets`). Bildirilen tek tek `agents[]`/`skills[]`/`rules[]` girişleri katalog üst verisidir ve kurulum sırasında diske karşı doğrulanmaz — bildirilen `agents[]` ve `skills[]` girişlerini, birinci taraf takımlar için `atl skills check` geliştirici komutu çapraz kontrol eder.

## Doğrulama

v2'de ayrı bir JSON Şeması dosyası ve şema doğrulama CI adımı yoktur. Doğrulama minimumdur ve CLI'nin kendisinde yaşar:

- `team.json` geçerli JSON olarak ayrıştırılabilmelidir.
- `name` alanı bulunmalıdır.
- Takım, bir varlık dizini altında en az bir dosya göndermelidir — `atl install`, kurulabilir varlık olmayan bir takımda hata verir.

Sözleşmenin tamamı budur. `atl install` takımını kabul ederse geçerlidir; yerel ya da CI'da çalıştırılacak başka bir şey yoktur.

## Sıradaki

- **[Bir takım oluşturma](./creating-a-team)** — adım adım.
- **[`atl install`](/tr/cli/install)** — bir takımın nasıl çözümlendiği ve kurulduğu.
