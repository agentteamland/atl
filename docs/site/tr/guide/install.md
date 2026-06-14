# Kurulum

`atl` tek statik bir Go ikili dosyası olarak gelir (~7 MB, hiçbir çalışma-zamanı bağımlılığı yok). Platformuna uygun kanalı seç.

---

## macOS / Linux — Homebrew (önerilen)

```bash
brew install agentteamland/tap/atl
```

Eşdeğer iki adımlı biçim:

```bash
brew tap agentteamland/tap
brew install atl
```

Yükseltme:

```bash
brew update && brew upgrade atl
```

### macOS / Linux — tek-satırlık yedek

`brew` yoksa (ya da standart kütüphaneye dayalı bir kurulum tercih ediyorsan):

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.sh | sh
```

En son `atl` ikilisini indirir, açar ve `/usr/local/bin/atl` konumuna taşır (gerekirse sudo ister). Sudo'suz bir kurulum için `ATL_INSTALL_DIR=$HOME/.local/bin` ile bastır.

---

## Windows — PowerShell tek-satırlık (önerilen)

```powershell
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

Hepsi bu kadar. PowerShell aç, yapıştır, Enter. Betik şunları yapar:

- En son `atl.exe` dosyasını GitHub Releases üzerinden indirir
- `%LOCALAPPDATA%\Programs\atl\` konumuna kurar (yönetici yetkisi gerekmez)
- O klasörü **kullanıcı PATH'ine** ekler
- `atl --version` çalıştırarak kurulumu doğrular

Yönetici hakkı gerekmez, paket yöneticisi ön koşulu yok, sıfırdan bir Windows makinesinde çalışır.

::: tip Neden scoop/winget yerine bu?
Bir paket yöneticisiyle rahatsan aşağıdaki scoop ve winget de iş görür. Ama henüz scoop ya da winget yapılandırman yoksa (ve "vibe coding" yapan birçok geliştiricide yoktur), tek-satırlık yöntem o uzun yolu tamamen atlatır.
:::

Yükseltme: aynı komutu yeniden çalıştır. Her zaman en son sürümü çeker.

Belirli bir sürüme sabitlemek:

```powershell
$env:ATL_VERSION = 'v1.1.4'
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

### Windows — Scoop

`scoop` zaten kurulu olan kullanıcılar için:

```powershell
scoop bucket add agentteamland https://github.com/agentteamland/scoop-bucket
scoop install atl
```

Yükseltme:

```powershell
scoop update atl
```

Henüz `scoop` yok mu? Yukarıdaki PowerShell tek-satırlığı kullan — önce `scoop` kurmaktan daha basit ve daha hızlı.

### Windows — winget

```powershell
winget install agentteamland.atl
```

::: tip En son sürümün gerisinde kalabilir
`atl` 2026-04-24 tarihinden beri winget kataloğunda. Her yeni sürüm `microsoft/winget-pkgs` üzerinde elle yapılan bir inceleme adımından geçer; bu yüzden winget bir-iki `v*` etiketin gerisinde olabilir. Mutlak en son sürüme ihtiyacın varsa yukarıdaki PowerShell tek-satırlığı veya `scoop`'u kullan.
:::

### Windows — Manuel ZIP

En elle yapılan yol, sıfır otomasyon, her yerde çalışır:

1. [**GitHub Releases**](https://github.com/agentteamland/cli/releases/latest) sayfasını aç → `atl_<version>_windows_amd64.zip` (ya da `arm64`) dosyasını indir.
2. ZIP'i denetimindeki bir klasöre aç, örneğin `C:\Users\<sen>\tools\atl\`.
3. O klasörü kullanıcı PATH'ine ekle: **Ayarlar → Sistem → Hakkında → Gelişmiş sistem ayarları → Ortam Değişkenleri → Path → Düzenle → Yeni**, klasör yolunu yapıştır, Tamam.
4. Yeni bir PowerShell / cmd penceresi aç → `atl --version` → sürümü yazdırmalı.

PowerShell betiklerinin kısıtlandığı kilitli makineler için uygundur.

---

## Manuel indirme (her platform)

Önceden derlenmiş bir ikiliyi [**GitHub Releases**](https://github.com/agentteamland/cli/releases/latest) üzerinden al. Dağıtılan yapı dosyaları:

- `darwin` (macOS): `amd64`, `arm64`
- `linux`: `amd64`, `arm64`
- `windows`: `amd64`, `arm64`

Aç, `atl` ikilisini `PATH` üzerindeki bir konuma koy, hazırsın.

---

## Doğrulama

```bash
atl --version
atl --help
```

Kurulu sürümü ve komut listesini (`install`, `list`, `remove`, `update`, `search`, `setup-hooks`) görmelisin.

## Ne kuruldu?

Tek bir ikili dosya. `atl` takım önbelleğini şurada tutar:

- macOS / Linux: `~/.claude/repos/agentteamland/`
- Windows: `%USERPROFILE%\.claude\repos\agentteamland\`

Kurulu takımlar bu önbellekte klonlanmış Git deposu olarak yaşar ve her projenin `.claude/` dizinine kopyalanır.

## Önerilen sonraki adım — otomatik güncelleme hook'ları

`atl` PATH üzerinde çalışır hale geldikten sonra:

```bash
atl setup-hooks
```

Bu komut Claude Code'un `SessionStart` ve `UserPromptSubmit` hook'larını bağlar; böylece her oturumda (ve her mesajda, 30 dakikada bir tetiklenecek biçimde kısıtlanarak) arka planda sessizce takım, çekirdek ve `atl` güncellemeleri kontrol edilir. `atl update` komutunu elle çalıştırmana gerek kalmaz — takımların kendiliğinden güncel kalır. Ayrıntı için [`atl setup-hooks`](/tr/cli/setup-hooks).

## Sıradaki

- **[Hızlı başlangıç](/tr/guide/quickstart)** — ilk takımını kur.
- **[Kavramlar](/tr/guide/concepts)** — takım, ajan, beceri, kural, kalıtım.
- **[Takım yazma](/tr/authoring/creating-a-team)** — kendi takımını yaz (herkese açık, özel ya da yerel).
