# Kurulum

`atl` tek statik bir Go ikili dosyası olarak gelir (~18 MB, hiçbir çalışma-zamanı bağımlılığı yok). Her platform için tek bir betik — önce kurulması gereken bir paket yöneticisi yok.

---

## macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

Hepsi bu kadar. Betik şunları yapar:

- En son sürümü GitHub üzerinden çözer
- İşletim sistemine ve mimarine uygun `atl` ikilisini indirir (`darwin`/`linux`, `amd64`/`arm64`)
- Açar ve `/usr/local/bin/atl` konumuna taşır (yalnızca o dizin yazılabilir değilse `sudo` ister)

Sudo'suz kurulum — denetimindeki bir dizini hedef göster:

```bash
ATL_INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

Yükseltme: aynı komutu yeniden çalıştır — her zaman en son sürümü çeker. (Buna nadiren ihtiyaç duyarsın: hook'lar kurulduktan sonra `atl` hem kendini hem de takımlarını arka planda güncel tutar. Bkz. [otomatik güncelleme hook'ları](#onerilen-sonraki-adim-otomatik-guncelleme-hook-lari).)

Belirli bir sürüme sabitlemek:

```bash
ATL_VERSION=v2.0.0 curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

---

## Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

PowerShell aç, yapıştır, Enter. Betik şunları yapar:

- En son `atl.exe` dosyasını GitHub Releases üzerinden indirir (`amd64` ya da `arm64`)
- `%LOCALAPPDATA%\Programs\atl\` konumuna kurar (yönetici yetkisi gerekmez)
- O klasörü **kullanıcı PATH'ine** ekler
- `atl --version` çalıştırarak kurulumu doğrular

Yönetici hakkı gerekmez, paket yöneticisi ön koşulu yok, sıfırdan bir Windows makinesinde çalışır.

Yükseltme: aynı komutu yeniden çalıştır. Her zaman en son sürümü çeker.

Özel kurulum dizini:

```powershell
$env:ATL_INSTALL_DIR = 'C:\Users\<sen>\bin'
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Belirli bir sürüme sabitlemek:

```powershell
$env:ATL_VERSION = 'v2.0.0'
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

---

## Manuel indirme (her platform)

Kilitli makineler için (ya da bir betiği boru hattından geçirmemeyi tercih ediyorsan) önceden derlenmiş bir ikiliyi doğrudan [**GitHub Releases**](https://github.com/agentteamland/atl/releases/latest) üzerinden al. Yapı dosyaları şunlar için dağıtılır:

- `darwin` (macOS): `amd64`, `arm64`
- `linux`: `amd64`, `arm64`
- `windows`: `amd64`, `arm64`

Arşivi aç (macOS/Linux'ta `.tar.gz`, Windows'ta `.zip`), `atl` dosyasını `PATH` üzerindeki bir konuma koy, hazırsın. Windows'ta klasörü kullanıcı PATH'ine **Ayarlar → Sistem → Hakkında → Gelişmiş sistem ayarları → Ortam Değişkenleri → Path** üzerinden ekle, ardından yeni bir terminal aç.

::: tip brew / scoop / winget yok
`atl` v2 yalnızca kurulum betikleri ve GitHub Releases üzerinden dağıtılır. Homebrew, Scoop ve winget kanalları v2 yeniden inşasında emekliye ayrıldı — tek-satırlık komut paket-yöneticisi kurulumunu tamamen atlar ve senkron tutulması gereken üçüncü-taraf bir tap yoktur.
:::

---

## Doğrulama

```bash
atl --version
atl --help
```

Kurulu sürümü ve üst düzey komutları görmelisin; bunlara `install`, `update`, `upgrade`, `remove`, `list`, `search`, `promote`, `publish`, `pin`, `unpin`, `learnings`, `tick`, `session-start`, `setup-hooks`, `doctor`, `gc`, `skills`, `rules`, `docs` ve `init` dahildir.

## Ne kuruldu?

Tek bir ikili dosya. `atl` kendi durumunu — indeks önbelleği, kalıcı öğrenme kuyruğu, kısıtlama (throttle) damgaları ve global-katman kazanımların — şurada tutar:

- macOS / Linux: `~/.atl/`
- Windows: `%USERPROFILE%\.atl\`

Takım varlıkları (ajanlar, beceriler, kurallar) Claude Code'un kendi dizinine kopyalanır; editör onları oradan alır:

- **Global katman:** `~/.claude/`
- **Proje katmanı:** `<project>/.claude/` (bu, mevcut proje için global katmanı gölgeler)

Yani `.atl/` ATL'nin operasyonel deposu, `.claude/` ise ajanların/becerilerin/kuralların Claude Code'un yüklemesi için gerçekten yaşadığı yerdir.

## Önerilen sonraki adım — otomatik güncelleme hook'ları

`atl` PATH üzerinde çalışır hale geldikten sonra:

```bash
atl setup-hooks
```

Bu komut Claude Code'un hook'larını bağlar; böylece platform kendini arka planda çalıştırır:

- **`SessionStart` → `atl session-start`** — önceki oturumun öğrenmelerini boşaltır, kendini onarmak için `doctor` çalıştırır ve `atl` binary'sini ve kurulu takımlarını güncel tutar (günde bir [self-update](/tr/cli/upgrade) + [takım güncellemesi](/tr/cli/update) kontrolü).
- **`UserPromptSubmit` → `atl tick --throttle=10m`** — oturum-içi bir bakım tıklaması (kısıtlanmış), böylece güncellemeler, fan-out (dağıtım) ve öğrenme yakalama parmağını kıpırdatmadan gerçekleşir.
- **`UserPromptSubmit` → `atl retrieve`** — istem başına bilgi getirme (retrieval): projenin wiki/journal sayfalarını her isteme göre sıralar (BM25 ile yerel bir anlamsal gömme (semantic embedder) modelinin harmanı) ve en iyi eşleşmeleri bağlam olarak sunar. Hataya-açık (fail-open) — bir istemi asla engellemez.
- **`PreToolUse` → `atl guard`** — her `Bash`, `Edit` ve `Write` araç çağrısından önce çalışır: felaket düzeyinde geri-alınamaz komutları ve en bariz sır-sızdırma (secret-exfiltration) çağrılarını deterministik olarak engeller ve mevcut bir dosyanın ilk düzenlemesinde engellemeyen bir düzenle-önce-grep hatırlatması ekler.

v2'de bunun açık olması beklenir, isteğe bağlı değil — otomasyon işin özüdür. `atl update` komutunu elle çalıştırmazsın; takımların ve `atl`'nin kendisi kendiliğinden güncel kalır. Ayrıntı için [`atl setup-hooks`](/tr/cli/setup-hooks).

## Sıradaki

- **[Hızlı başlangıç](/tr/guide/quickstart)** — ilk takımını kur.
- **[Kavramlar](/tr/guide/concepts)** — takım, ajan, beceri, kural ve global/proje kapsam ekseni.
- **[CLI başvurusu](/tr/cli/overview)** — her komut ayrıntısıyla.
- **[Takım yazma](/tr/authoring/creating-a-team)** — kendi takımını yaz ve yayımla.
