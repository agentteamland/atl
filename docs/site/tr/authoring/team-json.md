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
| `capabilities` | nesne | — | Çoğunlukla kurulum ayrıştırıcısının değil, platformun becerilerinin okuduğu isteğe bağlı sözleşmeler. `capabilities.review: "<agent>"`, [`/create-pr`](/tr/skills/create-pr)'in bu takımın uzman gözden geçireni olarak başlattığı ajanı adlandırır; `capabilities.profile`, profil katmanı sağlayıcı/tüketici rolünü bildirir ([profile-team](/tr/teams/profile-team)'e bakın). **CLI**'ın okuduğu tek anahtar `store` — aşağıya bakın. |
| `backends` | dize[] | — | `backends/<name>/` altında arka uca özel bağdaştırıcı paketleri gönderen takımlar için (ör. delivery-team'in `["azure", "github"]` değeri): takımın hangi arka uçları desteklediğini bildirir. Bugün yalnızca bilgilendirme amaçlıdır — kurulum ayrıştırıcısı bunu okumaz. |

::: tip Açıklamayı kısa tut
`description`, `atl search` çıktısında tek satır olarak gösterilir; uzun bir açıklama garip biçimde kırılır. Bir tanıtım cümlesini hedefle — paragraf değil.
:::

## Kalıcı depo bildirmek {#kalici-depo-bildirmek}

Çoğu takım her şeyi, ATL'nin zaten izlediği yansıtılmış `.claude` ağacının içinde tutar. Kendi uzun ömürlü verisini bunun dışında bir yerde tutan bir takım — ilki profile-team'in `~/.atl/profiles/`'i — o konumu bildirir:

```json
{
  "capabilities": {
    "profile": { "role": "provider", "store": "~/.atl/profiles" }
  }
}
```

`store` herhangi bir yetenek adının altında yer alabilir, tek bir yol tutar ve ev dizini için `~` kullanabilir. `atl install` bildirilen her depoyu kurulum manifest'ine kaydeder; bundan sonra **oturum başlangıcı ve `atl tick`, o dizini yerel git altında tutar** — oturum başına bir kez ve tick'in throttle penceresi başına bir kez — ve son geçişten bu yana değişen ne varsa commit'ler.

Amaç geri getirilebilirliktir. Yazma politikası "son yazan kazanır" olan bir depo, her üzerine yazmada önceki değeri kaybeder: çıkarımla konmuş bir geçici değerin yerine doğrulanmış bir yanıt geldiğinde, yerine geçtiği şeyden hiçbir iz kalmaz. Dizin sürümlendiğinde eski değer bir `git show HEAD~1` uzaklıktadır.

::: tip Bu özellik gelmeden önce mi kurmuştunuz?
Bildirim kurulum anında okunur, dolayısıyla `stores` alanından önceki bir kurulumda bu kayıt yoktur. `atl update` bunu bir kez geri doldurur — sabitlenmiş kaynağı yeniden çekerek — ve kendiliğinden çalıştığı için sizin bir şey yapmanız gerekmez. O çalışana kadar depo henüz sürümlenmiyor demektir.
:::

Bunun bilinçli olarak **yapmadıkları**:

- **Dizini asla oluşturmaz.** Olmayan bir depo, özelliğin bu makinede kullanılmadığı anlamına gelir; onu oluşturmak hem diski kirletir hem de özelliği etkinmiş gibi gösterir.
- **Asla remote tanımlamaz ve asla push etmez.** Bir depo genellikle kullanıcının en hassas verisini tutar; bir kopyayı makinenin dışına taşımak, kullanıcının ayrıca istemesi gereken ayrı bir eylemdir (profile-team'in [`/profile-backup`](/tr/teams/profile-team)'ı böyle bir yoldur ve herkese açık bir depoya yazmayı reddeder).
- **Yalnızca kendi oluşturduğu repo'ya yazar.** Depoyu zaten kendi sürüm kontrolünüzde tutuyorsanız ATL ona hiç dokunmaz — amacı siz zaten karşılamışsınızdır ve dalınızı ilerletmek, devam eden çalışmanızın altından `HEAD`'i kaydırırdı. ATL kendi oluşturduğu repo'ları `.git/atl-store` dosyasıyla işaretler; o dosyayı silerseniz o depoya commit'lemeyi bırakır.
- **Başka bir repo'nun içinde duran bir depoya asla dokunmaz.** Orada `git init` yapmak dıştaki repo'yu gölgelerdi.
- **Git durumunuzu asla bozmaz.** Anlık görüntü, tek kullanımlık bir index üzerinde plumbing komutlarıyla yazılır; staging alanınıza dokunulmaz ve hiçbir hook çalışmaz. Merge, rebase veya cherry-pick ortasındaki bir repo, o durumdan çıkana kadar atlanır.
- **Kimseye erişim vermez.** ATL bildirilen *yolu* yalnızca bu tek mekanik amaç için okur. Bir depoyu kimin okuyup yazabileceği, platformun henüz uygulamadığı ayrı bir sözleşmedir.

Bildirilen yollar bunların hiçbiri çalışmadan önce denetlenir: önce sembolik bağlar çözülür, ardından hedefin ev dizininizin en az iki seviye altında olması ve içinde bulunduğunuz çalışma dizinini kapsamaması aranır. Yani `~`, en üst seviyedeki tek bir dizin veya üzerinde çalıştığınız projenin kendisi repo'ya çevrilmez, reddedilir. Bu bir kum havuzu (sandbox) değildir — bilerek kendi başka bir dizininize yönlendirdiğiniz bir yol kabul edilir, çünkü bu bir takımın deposunu meşru şekilde orada tutmasından ayırt edilemez. Tüm geçişi kapatmak için `ATL_NO_STORE_GIT=1`.

Bilinen bir sınır: depo kendi içinde bir git repo'su barındırıyorsa, o alt ağaç gitlink olarak kaydedilir ve içeriği anlık görüntüye girmez. Bir depo makine tarafından yazılan veridir, dolayısıyla pratikte bu durum oluşmaz — ama oluşursa sessizdir.

CLI bundan takımın hakkında hiçbir şey öğrenmez — ne ad, ne anlam, ne de dizinin ne tuttuğu bilgisi. Bir bildirime uyar; gelecekteki herhangi bir takımın aynı davranışı bedava almasının nedeni budur.

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
