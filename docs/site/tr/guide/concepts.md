# Kavramlar

AgentTeamLand'i oluşturan parçalar ve nasıl bir araya geldikleri.

## Takım

Bir **takım** bir pakettir. Claude Code ile belirli bir tür işi yapmak için gereken her şeyi bir araya getirir:

- **Ajanlar** — kendi bağlamı ve sorumluluk alanı olan uzmanlaşmış kişilikler.
- **Beceriler** — kullanıcı tarafından çağrılabilen, Claude Code'da eğik çizgili komut olarak sunulan araçlar.
- **Kurallar** — her zaman yüklü olan davranış sınırları ve sözleşmeler.

Bir takım, kök dizininde `team.json` bulunan bir Git deposunda yaşar. Bu dosya takımı tanımlar: adı, sürümü, neyi kapsadığı, neye bağımlı olduğu, neyi genişlettiği.

Takımı bir projeye kurarsın; içeriği `.claude/` dizinine kopya olarak düşer. Claude Code onları anında görür.

## Ajan

Bir ajan, bir rolü tanımlayan bir Markdown dosyasıdır. `api-agent`, `flutter-agent`, `code-reviewer` — her biri kendi sorumluluk alanı ve kendi bilgi tabanı olan odaklı bir kişiliktir.

Karmaşık ajanlar için sözleşme **çocuklar deseni**dir: üst düzey `agent.md` dosyası kısa kalır (kimlik, kapsam, ilkeler) ve ayrıntılı bilgi `children/` altında konu başına bir dosya olarak yaşar. Bu, üst dosyayı sıkı tutar ve tek bir konuyu, gerisine dokunmadan ucuza güncellemeyi sağlar. Her çocuk dosyası `knowledge-base-summary` adında bir frontmatter satırı taşır; `/save-learnings`, `agent.md` içindeki otomatik yeniden inşa edilen **Knowledge Base** bölümüne bu satırları taşır — yani üst dosyadaki dizin daima çocuklardan türetilir, asla elle düzenlenmez.

## Beceri {#skill}

Bir beceri, kullanıcının çağırdığı eğik çizgili bir komuttur. `/create-new-project`, `/verify-system`, `/save-learnings`. Beceriler, kök dizininde `skill.md` bulunan bir dizin olarak gelir; bu dosya, becerinin ne zaman kullanılacağını ve ne yapacağını anlatır.

Beceriler ajanın biçimini aynalar: karmaşık beceriler, konu başına bir dosya olarak edge case ve birikmiş deneyim taşıyan bir `learnings/` alt dizini barındırır; aynı `knowledge-base-summary` frontmatter sözleşmesiyle. `/save-learnings`, `skill.md` içindeki **Accumulated Learnings** bölümünü o frontmatter satırlarından yeniden kurar — `learnings/`, beceriler için neyse `children/` da ajanlar için odur.

Beceriler **global** (bootstrap ile yayımlanır) ya da **takım kapsamlı** (belirli bir takımla yayımlanır ve yalnızca o takım kurulduktan sonra görünür) olabilir. `/create-new-project`, `/verify-system` ve `/design-screen` takım kapsamlıdır çünkü yaptıkları iş her zaman yığına özgüdür. `/brainstorm`, `/rule`, `/save-learnings`, `/wiki`, `/create-pr`, `/create-code-diagram` ise globaldir çünkü her yerde geçerlidir.

## Kural {#rule}

Bir kural, her Claude Code oturumuna yüklenen bir Markdown dosyasıdır. Bir beceriden farklı olarak (beceri çağrılmayı bekler), bir kural daima etkindir — daha sen bir şey sormadan Claude'un projeyi nasıl düşüneceğini biçimlendirir.

Global kurallar `~/.claude/rules/` dizininde yaşar. Takımın sağladığı kurallar, takım kurulduğunda projenin `.claude/rules/` dizinine kopyalanır.

## Kayıt defteri

**Kayıt defteri**, [`agentteamland/registry`](https://github.com/agentteamland/registry) adresindeki tek bir JSON dosyasıdır; kısa takım adlarını Git URL'lerine eşler. `atl install software-project-team` komutu adı arar, URL'yi bulur ve kurulumu oradan yapar.

Kayıt defterine eklemeler PR ile olur. CI her kaydı JSON şemasına göre doğrular, URL'nin erişilebilirliğini denetler ve yinelenenleri işaretler.

## Kalıtım

Bir takım başka bir takımı **genişletebilir**. Alt takım, üst takımın ajanlarını, becerilerini ve kurallarını miras alır; herhangi birini ad bazında **bastırabilir** ve istemediklerini **dışarıda bırakabilir**.

```json
{
  "name": "my-team",
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    { "name": "api-agent", "description": "API desenlerimize özel bastırmalar." }
  ]
}
```

Kısıtlar:

- **Tek üst takım** — çoklu kalıtım yoktur.
- **Sınırsız derinlik** — zincirler istendiği kadar uzun olabilir.
- **Döngü algılama** — `A extends B extends A` tam zinciri hata mesajına koyarak hızla başarısız olur.
- **Yükleme sırası** — en derin atadan başlanır, mevcut takım en sona kalır. Sana en yakın olan kazanır.

Tüm ayrıntılar: [Kalıtım](/tr/authoring/inheritance).

## CLI

`atl` kullanıcının kullandığı araçtır. Beş iş yapar:

- `atl install [team]` — bir takımı (kayıt defteri adı veya Git URL'si ile) mevcut projeye kurar.
- `atl list` — burada neyin kurulu olduğunu gösterir.
- `atl remove [team]` — kaldırır.
- `atl update [team]` — bir takımın ya da kurulu tüm takımların son sürümünü çeker.
- `atl search [query]` — kayıt defterinde arama yapar.

Bakınız: [CLI genel bakışı](/tr/cli/overview).

## Çalışma alanları

Bir **proje**, `atl`'yi çalıştırdığın bir dizindir. İçine takım içeriğinin kopyalandığı bir `.claude/` alt dizini kazanır.

**Önbellek** (`~/.claude/repos/agentteamland/`) gerçek takım depolarını barındırır — yalnızca bir kez klonlanır, aynı takımı kuran her projede yeniden kullanılır. Önbelleği silmek güvenlidir; `atl update` onu yeniden doldurur.

## Claude Code ile birlikte nasıl çalışır?

Claude Code her oturumun başında `.claude/` dizinini okur. Bir takımın bu dizine getirdiği her şey anında ortaya çıkar: yetki devri için hazır ajanlar, eğik çizgili komut olarak hazır beceriler, her isteme yüklenmiş kurallar.

AgentTeamLand, Claude Code'un yerine geçmez ve onu genişletmez. O bir teslim katmanıdır: Claude Code'un zaten okuduğu dosyalar için paket yönetimi.
