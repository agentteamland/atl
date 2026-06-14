# Sürüm yayım hattı (goreleaser → brew + scoop + winget)

`atl` sürümlerinin [`agentteamland/cli`](https://github.com/agentteamland/cli) deposundaki bir Git etiketinden, desteklenen her platformda `brew install` / `scoop install` / `winget install` ile kuruluma hazır bir ikiliye nasıl ulaştığı.

Bu sayfa **bakımcılar içindir**. Yalnızca `atl`'yi kurmak istiyorsan bkz. [Kurulum](../guide/install).

## Hatta bir bakış

```
[cli deposunda etiket push'landı]          git tag v1.1.4 && git push origin v1.1.4
        ↓
[GitHub Actions iş akışı tetiklendi]       .github/workflows/release.yml
        ↓
[goreleaser 6 ikili derler]                darwin/amd64, darwin/arm64,
                                           linux/amd64, linux/arm64,
                                           windows/amd64, windows/arm64
        ↓
[goreleaser her kanal için yayımlar]
   ├── GitHub Release (yapıt dosyaları eklenmiş)
   ├── homebrew-tap deposu (Formula/atl.rb kendiliğinden artırılır + zorla push'lanır)
   ├── scoop-bucket deposu (atl.json manifesto kendiliğinden artırılır)
   └── winget-pkgs ÇATALI (manifests/a/AgentTeamLand/atl/<version>/ eklenir)
        ↓
[elle yapılan üst akış PR'ı — yalnızca winget]   bkz. /tr/contributing/winget-process
```

Diğer her kanal (brew + scoop) tümüyle otomatiktir. winget, sürümün üst akış kataloğunda gerçekten görünmesi için `microsoft/winget-pkgs` deposuna elle bir PR gerektirir.

## Etiketleme nasıl yapılıyor?

Sürümler `cli` deposundaki `internal/config/config.go` dosyasında yönetilir:

```go
// internal/config/config.go
const Version = "0.1.0-dev"  // derleme sırasında ldflags ile bastırılır
```

`dev` son eki çalışma ağacının varsayılanıdır. Goreleaser derleme yaparken Git etiketi adını kullanarak ldflags ile bunu bastırır:

```bash
go build -ldflags "-X 'github.com/agentteamland/cli/internal/config.Version=v1.1.4'"
```

Yani `atl --version`, derlemenin yapıldığı etiketin değerini yazdırır.

## Etiketten sürüme akış

Sürüm üretmeye değer bir PR'ı birleştirdikten sonra:

```bash
cd repos/cli
git checkout main && git pull
git tag v1.1.4         # birleştirme sonrası main'deki config.go değerini kullan
git push origin v1.1.4 # .github/workflows/release.yml dosyasını tetikler
```

Etiket push'u:

1. `cli` üzerinde GitHub Actions'ı tetikler.
2. goreleaser 6 ikiliyi çapraz derler.
3. Commit başlıklarından kendiliğinden üretilen değişiklik günlüğüyle birlikte ikilileri içeren bir GitHub Release yayımlar.
4. [`agentteamland/homebrew-tap`](https://github.com/agentteamland/homebrew-tap) deposuna Formula güncellemelerini zorla push'lar.
5. [`agentteamland/scoop-bucket`](https://github.com/agentteamland/scoop-bucket) deposuna manifesto güncellemelerini zorla push'lar.
6. [winget-pkgs çatalımıza](https://github.com/agentteamland/winget-pkgs) yeni manifestolar ekler.

Etiket push'undan ~5 dakika içinde `brew upgrade atl` ve `scoop update atl` yeni sürümü çekecektir. winget kanalı bir üst akış PR'ı ister (aşağıya bak).

## Kanal: Homebrew (macOS + Linux)

[`agentteamland/homebrew-tap`](https://github.com/agentteamland/homebrew-tap) `Formula/atl.rb` konumunda tek bir Ruby Formula barındırır. Goreleaser her etiketde Formula'yı değiştirir — elle düzenlemeye gerek kalmaz.

Buraya dal koruması bilinçli olarak uygulanmaz (goreleaser'ın zorla push'unu engellerdi).

`Formula/atl.rb` şunları taşır:

- Mevcut sürüm.
- Platform başına ikili URL'leri (cli deposunun GitHub Release yapıtlarını gösterir).
- Platform başına SHA-256 sağlama toplamları.

Şu komutla kurulum yapan bir kullanıcı:

```bash
brew install agentteamland/tap/atl
```

tap deposunu klonlar, `Formula/atl.rb` dosyasını okur, kendi platformuna uygun ikiliyi cli deposunun GitHub Release'inden indirir, sağlama toplamını doğrular ve `/opt/homebrew/bin/atl` (Intel macOS / Linuxbrew için `/usr/local/bin/atl`) konumuna kurar.

> **Brew tap bayatlamış formula uyarısı:** `brew upgrade atl`, üçüncü taraf tap klonunu kendiliğinden tazelemez — yalnızca homebrew-core'un merkezi dizini kendiliğinden tazelenir. `brew update && brew upgrade atl` kullan (ya da 24 saatlik kısıtlamayı kapatmak için `HOMEBREW_AUTO_UPDATE_SECS=1` ayarla). Bu tuzakla ilgili ayrıntılı tarihçe için [brew-tap-stale-formula](https://github.com/agentteamland/workspace/blob/main/.atl/wiki/brew-tap-stale-formula.md) wiki kaydına bak.

## Kanal: Scoop (Windows)

[`agentteamland/scoop-bucket`](https://github.com/agentteamland/scoop-bucket) `bucket/atl.json` konumunda tek bir JSON manifestosu barındırır. Homebrew ile aynı kendiliğinden güncelleme düzeneği — goreleaser her etiketde üzerine yazar.

Şu komutla kurulum yapan bir kullanıcı:

```powershell
scoop bucket add agentteamland https://github.com/agentteamland/scoop-bucket
scoop install atl
```

paket kovasını uzak depo olarak ekler, `atl.json` dosyasını indirir, Windows ikilisini cli GitHub Release'inden alır ve `~/scoop/apps/atl/` konumuna kurar.

Dal koruması: uygulanmaz (Homebrew tap'iyle aynı sebep).

## Kanal: winget (Windows, resmi katalog)

[`agentteamland/winget-pkgs`](https://github.com/agentteamland/winget-pkgs), [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs) deposunun bir **çatalıdır**. winget, diğer kanalların aksine herhangi bir depodan çekim YAPMAZ — paketlerin Microsoft'un resmi kataloğunda olması gerekir.

Akış:

1. **Goreleaser kendiliğinden push** — yeni manifestoları ÇATALIMIZIN `manifests/a/AgentTeamLand/atl/<version>/` dizinine push'lar.
2. **Üst akışa elle PR** gereklidir: çatalımızdan `microsoft/winget-pkgs:master` dalına bir PR aç.
3. Microsoft'un doğrulama hattı çalışır (sertifika denetimi, MSIX/MSI/EXE biçim denetimi, hash doğrulaması vb.).
4. Microsoft inceleyicisi birleştirir (tipik olarak birkaç gün içinde).
5. Birleştirme sonrasında `winget upgrade atl` yeni sürümü çeker.

Üst akış PR adımı [winget üst akış-PR süreci](winget-process) sayfasında ayrıntılıdır. Üst akış PR'ı birleşene kadar winget kullanıcıları önceki sürümü görür — tipik gecikme bir-iki `v*` etiketidir.

İlk üç winget sürümü:

- v0.1.1 — [PR #361975](https://github.com/microsoft/winget-pkgs/pull/361975), 2026-04-24'te birleştirildi.
- v0.2.0 — [PR #364841](https://github.com/microsoft/winget-pkgs/pull/364841), 2026-04-25'te birleştirildi.
- v1.1.4 — [PR #367931](https://github.com/microsoft/winget-pkgs/pull/367931), inceleme bekliyor (ara v0.3.0 / v1.0.0 / v1.1.0 / v1.1.1 / v1.1.2 / v1.1.3 manifestoları çatalımızın dallarında korundu ama winget sözleşmesi gereği ayrı geri PR'ları yapılmadı).

## Bu neden böyle kurulu?

Üç kanallı bölünme, **çapraz platform CLI'lar için standart**tır. Her platformun kendi baskın paket yöneticisi vardır ve kullanıcılar tanıdık araçlarıyla kurmayı bekler. Goreleaser orkestratördür — tek yapılandırma (cli deposundaki `.goreleaser.yaml`), bir etiket üçünü birden tetikler.

İki otomatikleştirilmiş kanal (brew + scoop) bakım maliyetini neredeyse sıfırda tutar. winget kanalı sürüm başına elle PR ister çünkü Microsoft öyle ister — bu, resmi Windows kataloğunda olmanın bedelidir.

Sürüm yayım hattı depoları (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) bilinçli olarak dal koruması ALTINDA değildir. Goreleaser zorla push'lar; koruma bunu engellerdi. Hangi depoların korunduğu / korunmadığı için bkz. [Yönetişim](../guide/governance).

## İlgili

- [`atl`'yi kur](../guide/install) — üç kanal için kullanıcıya yönelik kurulum yönergeleri.
- [winget üst akış-PR süreci](winget-process) — winget hattındaki elle yapılan adım.
- [Yönetişim](../guide/governance) — organizasyon genelinde dal koruması.
- cli deposunun [`.goreleaser.yaml`](https://github.com/agentteamland/cli/blob/main/.goreleaser.yaml) dosyası — tüm hattı orkestre eden goreleaser yapılandırması.
