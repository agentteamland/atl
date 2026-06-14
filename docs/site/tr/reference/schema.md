# Şema

`team.json` şeması şu konumda yayımlanır:

**[`agentteamland/core/schemas/team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json)**

**JSON Schema Draft 2020-12** standardını izler.

## Hızlı başvuru

| Alan | Tür | Zorunlu | Varsayılan | Not |
|---|---|---|---|---|
| `schemaVersion` | tam sayı | ✅ | — | Şu an `1`. |
| `name` | dize | ✅ | — | Küçük harf kebab-case. Kayıt defterinde benzersiz olmalıdır. |
| `version` | dize | ✅ | — | SemVer 2.0.0. |
| `description` | dize | ✅ | — | Tek cümlelik özet. |
| `author` | dize | — | — | `"Name <email>"` biçimi önerilir. |
| `license` | dize | — | `"MIT"` | SPDX tanımlayıcısı. |
| `keywords` | dize[] | — | `[]` | `atl search` için kullanılır. |
| `repository` | dize | — | — | Git URL'si. |
| `homepage` | dize | — | — | Belge / açılış URL'si. |
| `agents` | nesne[] | — | `[]` | Her biri: `{ name: string, description: string }`. |
| `skills` | nesne[] | — | `[]` | `agents` ile aynı biçim. |
| `rules` | nesne[] | — | `[]` | `agents` ile aynı biçim. |
| `extends` | dize | — | — | `"team-name"` ya da `"team-name@constraint"`. |
| `excludes` | dize[] | — | `[]` | Üst takımdan düşürülecek öğe adları. |
| `dependencies` | nesne | — | `{}` | `"team-name"` → `"version-constraint"` eşlemesi. |
| `requires` | nesne | — | `{}` | `{ atl: string }` — en düşük CLI sürümü. |

## CI'de şemayı kullanma

Herhangi bir JSON Şema doğrulayıcısı işe yarar. [Bir takım yazma](/tr/authoring/creating-a-team) sayfası tam bir `ajv-cli` GitHub Actions iş akışı gösterir. Kısa sürüm:

```bash
npm install -g ajv-cli
curl -sSfL https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json -o team.schema.json
ajv -s team.schema.json -d team.json --strict=false
```

## Düzenleyici tümleştirmesi

Düzenleyicin `$schema` üzerinden JSON Şemasını destekliyorsa `team.json` dosyasının üstüne şunu ekle:

```json
{
  "$schema": "https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json",
  "schemaVersion": 1,
  ...
}
```

VS Code, JetBrains ve çoğu JSON düzenleyicisi otomatik tamamlama ile satır içi doğrulama sağlar.

## Şema sürümleme

- **Eklemeli değişiklikler** (yeni isteğe bağlı alan) → sürüm artırımı yok.
- **Geriye uyumsuz değişiklikler** (alan silme, tür değiştirme, zorunlu kümeyi sıkılaştırma) → `schemaVersion` `2`'ye çıkar. CLI, eski `schemaVersion` değerlerini bir kullanımdan kaldırma penceresi boyunca kabul etmeye devam eder (en az iki küçük CLI sürümü).

Mevcut şema `schemaVersion: 1` durumundadır.

## İlgili

- **[team.json](/tr/authoring/team-json)** — örneklerle insan odaklı başvuru.
- **[Sözlük](./glossary)** — şema boyunca kullanılan terimler.
