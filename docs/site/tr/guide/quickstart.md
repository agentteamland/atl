# Hızlı başlangıç

Sıfırdan üretime hazır bir ajan takımına bir dakikadan kısa sürede.

## 1. `atl`'yi kur

```bash
# macOS / Linux
brew install agentteamland/tap/atl
```

```powershell
# Windows (PowerShell; paket yöneticisi gerekmez)
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

Tam liste (`scoop`, `winget`, manuel ZIP, tek-satırlık yedek): [Kurulum rehberi](/tr/guide/install).

## 2. Bir proje dizini oluştur

```bash
mkdir my-new-app && cd my-new-app
```

`atl` bir projenin içinde çalışmayı bekler — takımın ajanları, becerileri ve kuralları için burada bir `.claude/` dizini oluşturacak.

## 3. Bir takım keşfet

```bash
atl search dotnet
```

Herkese açık [kayıt defterinden](https://github.com/agentteamland/registry) eşleşen kayıtları görürsün; aralarında `software-project-team` da var.

## 4. Takımı kur

```bash
atl install software-project-team
```

Birkaç saniye içinde:

- Takım paylaşımlı önbelleğe klonlanır (yalnızca ilk seferinde).
- 13 ajan ve 3 beceri (`create-new-project`, `verify-system`, `design-screen`) `.claude/` dizinine kopyalanır.
- `team.json` kaydı `.claude/.team-installs.json` dosyasına işlenir.

Projenin içine bağlanmış tam bir .NET + Flutter + React + Docker ajan takımına sahipsin artık.

## 5. Ne kurulduğunu incele

```bash
atl list
```

Bu projede kurulu takımları, etkili ajan sayılarını ve kalıtım zincirini gösterir.

## 6. Claude Code içinde kullan

Bu dizinde Claude Code'u aç. Takımın becerileri eğik çizgili komutlar olarak hazırdır:

- `/create-new-project MyApp` — tam bir yığını iskeleyle kurar (Aşama 1–5: toplama → iskele → derleme → doğrulama → commit).
- `/verify-system` — konteynerler, portlar, uygulamalar ve boru hatları üzerinde uçtan uca sağlık denetimi yapar.

Ve her ajan (`api-agent`, `socket-agent`, `worker-agent`, `flutter-agent`, `react-agent`, `database-agent`, `redis-agent`, `rmq-agent`, `infra-agent`, `code-reviewer`, `project-reviewer`, `design-system-agent`, `ux-agent`) Claude'un yetki devredeceği biçimde kullanılabilir.

## 7. Güncel kal

Takımın yazarı iyileştirmeler yayımladığında:

```bash
atl update
```

Kurulu tüm takımlar çekilir, bağımlılıklar çözülür, kopyalar yenilenir. Projenin kodunda hiçbir şey değişmez.

## Az önce ne oldu?

Tek bir komutla, özenle seçilmiş, sürümü sabitlenmiş ve bağımlılıkları bilinen bir ajan kümesini projene kurdun. Aynı takımı kuran her proje aynı yapılandırmayı alır — yazar yeni iyileştirmeler yayımladığında da aynı güncellemeleri alır.

## Yanına tasarım araçları ekle (isteğe bağlı)

Tasarım sistemi ve ekran prototipi araçlarını da istiyorsan `design-system-team`'i kur:

```bash
atl install design-system-team
```

Ardından Claude Code sohbetinde:

```
/dst-init
/dst-new-ds primary
/dst-new-prototype --ds primary login-screen
/dst-open
```

`.dst/` altında jeton hizalı tasarım sistemleri ve çoklu durumlu HTML prototipleri elde edersin; herhangi bir tarayıcıda açılır. Tam beceri kümesi için [design-system-team](/tr/teams/design-system-team) sayfasına bak.

## Sıradaki

- **[Takımlara göz at](/tr/teams/)** — kayıt defterindeki doğrulanmış takımlar.
- **[Kavramlar](/tr/guide/concepts)** — takım, ajan, beceri ve kuralların arkasındaki zihinsel model.
- **[CLI başvurusu](/tr/cli/overview)** — her komut ayrıntısıyla.
- **[Kendi takımını yaz](/tr/authoring/creating-a-team)** — takımı kayıt defterine yayımla.
