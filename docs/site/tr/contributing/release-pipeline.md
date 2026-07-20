# Yayım hattı (goreleaser → GitHub Releases)

Bir `atl` yayımının, [`agentteamland/atl`](https://github.com/agentteamland/atl) monoreposundaki bir git etiketinden, desteklenen her platformda kuruluma hazır bir ikiliye nasıl ulaştığı.

Bu sayfa **bakımcılar** içindir. Yalnızca `atl`'yi kurmak istiyorsan bkz. [Kurulum](../guide/install).

## Hatta bir bakış

```
[atl monoreposunda etiket push'lanır]   git tag v2.0.0 && git push origin v2.0.0
        ↓
[GitHub Actions iş akışı tetiklenir]    .github/workflows/release.yml
        ↓
[go test ./...]                         cli modülü, hiçbir şey yayımlanmadan önce test edilir
        ↓
[goreleaser 6 ikili derler]             linux/amd64,   linux/arm64,
                                        darwin/amd64,  darwin/arm64,
                                        windows/amd64, windows/arm64
        ↓
[goreleaser TEK kanal yayımlar]
   └── GitHub Release
         ├── platform başına arşivler (.tar.gz; Windows'ta .zip)
         ├── bir checksums dosyası (atl_<version>_checksums.txt)
         └── commit türüne göre gruplanan otomatik changelog
```

v2 **yalnızca GitHub Releases** üzerinden dağıtılır. Homebrew, Scoop ya da winget kanalı yoktur — bunlar v2 yeniden inşasında emekliye ayrıldı. Kullanıcılar, doğru arşivi doğrudan en son GitHub Release'ten indiren tek satırlık betiklerle (`install.sh` / `install.ps1`) kurar.

## Sürümleme nasıl yapılır

Sürüm, derleme zamanında `cli/internal/buildinfo` üzerinden ikiliye gömülür:

```go
// cli/internal/buildinfo/buildinfo.go
var (
	Version = "dev" // yayım derlemesinde ldflags ile geçersiz kılınır
	Commit  = ""
	Date    = ""
)
```

`dev`, çalışma ağacının varsayılanıdır. goreleaser, git etiketi, commit ve derleme tarihini kullanarak üçünü de ldflags ile geçersiz kılar:

```
-X github.com/agentteamland/atl/cli/internal/buildinfo.Version={{.Version}}
-X github.com/agentteamland/atl/cli/internal/buildinfo.Commit={{.Commit}}
-X github.com/agentteamland/atl/cli/internal/buildinfo.Date={{.Date}}
```

Böylece `atl --version`, derlemenin kesildiği etiketi yazdırır.

## Etiket → yayım akışı

Yayıma değer bir PR'ı `main`'e birleştirdikten sonra:

```bash
cd repos/atl
git checkout main && git pull
git tag v2.0.0          # yayımlanacak sürüm
git push origin v2.0.0  # .github/workflows/release.yml'i tetikler
```

Etiket push'u:

1. `v*` etiketinde `release` iş akışını tetikler (`.github/workflows/release.yml`).
2. `cli/` içinde `go test ./...` çalıştırır — başarısız bir test yayımı durdurur.
3. `goreleaser release --clean` çalıştırır; bu da 6 ikiliyi çapraz derler.
4. Arşivler, bir checksums dosyası ve commit başlıklarından otomatik üretilen bir changelog (Features / Bug fixes / Documentation / Others) ile bir GitHub Release yayımlar.

Etiket push'undan bir-iki dakika sonra yeni sürüm en son GitHub Release olur — ve kurulum betikleri ona kendiliğinden çözümlenir.

## Tek kanal: GitHub Releases

[`.goreleaser.yaml`](https://github.com/agentteamland/atl/blob/main/.goreleaser.yaml), tek bir `builds` girdisi (cli modülü, `dir: cli`, `main: ./cmd/atl`) ve tek bir `archives` girdisi tanımlar:

- **Arşivler** — linux/darwin için `atl_<version>_<os>_<arch>.tar.gz`, windows için `.zip`. Her arşiv, `atl` ikilisini artı `README.md` + `LICENSE` dosyalarını paketler.
- **Checksums** — her arşivi kapsayan, doğrulama için tek bir `atl_<version>_checksums.txt`.
- **Changelog** — GitHub commit geçmişinden üretilir, conventional-commit türüne göre gruplanır.

Yayımın `header` alanı kurulum tek satırlıklarını gömer; böylece GitHub Release sayfasının kendisi kullanıcılara nasıl kurulacağını gösterir.

## Kullanıcılar bundan nasıl kurar

[`scripts/install.sh`](https://github.com/agentteamland/atl/blob/main/scripts/install.sh) (macOS/Linux) ve [`scripts/install.ps1`](https://github.com/agentteamland/atl/blob/main/scripts/install.ps1) (Windows) kurulum betikleri:

1. En son yayım etiketini GitHub API üzerinden çözer (ya da sabitlenmiş bir `ATL_VERSION`'a uyar).
2. İşletim sistemi + mimariyi algılar ve arşiv adını oluşturur (`atl_<version>_<os>_<arch>.tar.gz`).
3. O arşivi ve yayımın checksums dosyasını indirir, arşivin sha256'sını ona karşı doğrular — herhangi bir uyuşmazlık ya da eksik girdi kurulumu durdurur (fail-closed).
4. `atl`'yi çıkarır ve kullanıcının `PATH`'ine yerleştirir (`ATL_INSTALL_DIR`; varsayılan macOS/Linux'ta `/usr/local/bin`, Windows'ta `%LOCALAPPDATA%\Programs\atl` — betiğin ayrıca bu dizini kullanıcının PATH'ine eklediği yer).

Paket yöneticisi yok, tap yok, merkezî katalog yok — yayım çıktısı tek doğru kaynaktır. Kullanıcıya dönük yönergeler için bkz. [Kurulum](../guide/install).

## Neden tek kanal

v1, Homebrew, Scoop ve winget'e dağılıyordu. Her biri bakım maliyeti ekliyordu — özellikle winget, yayım başına `microsoft/winget-pkgs`'e elle bir PR, bir Microsoft inceleme kuyruğu ve denetlenmesi gereken bir fork-master disiplini gerektiriyordu. v2 yeniden inşası, dağıtımı kurulum betikleri + GitHub Releases'e indirgedi: tek goreleaser yapılandırması, tek etiket, tek çıktı kümesi, yayım başına sıfır elle adım. goreleaser, çapraz derleme + yayım orkestratörü olarak kalır; yalnızca alt akıştaki paket yöneticisi push'ları kaldırıldı.

## İlgili

- [`atl`'yi kur](../guide/install) — kullanıcıya dönük kurulum yönergeleri.
- Monoreponun [`.goreleaser.yaml`](https://github.com/agentteamland/atl/blob/main/.goreleaser.yaml) dosyası — bu hattı süren goreleaser yapılandırması.
- [`.github/workflows/release.yml`](https://github.com/agentteamland/atl/blob/main/.github/workflows/release.yml) — etiketle tetiklenen iş akışı.
