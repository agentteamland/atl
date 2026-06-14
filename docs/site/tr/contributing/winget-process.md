# winget üst akış-PR süreci

AgentTeamLand'in yeni `atl` sürümlerini [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs) deposundaki resmi Windows Package Manager kataloğuna nasıl yayımladığı.

Bu sayfa **bakımcılar içindir**. Son kullanıcılar bir sürüm üst akış kataloğuna indikten sonra `winget install AgentTeamLand.atl` ile kurulum yapar.

## Bu neden elle yapılır?

Diğer iki dağıtım kanalı (Homebrew + Scoop) goreleaser tarafından tümüyle otomatikleştirilmiştir — [`agentteamland/cli`](https://github.com/agentteamland/cli) deposundaki her Git etiketi, organizasyonumuzun tap ve bucket depolarındaki formula / manifestoyu ~5 dakika içinde kendiliğinden artırır.

winget farklıdır. Microsoft'un resmi kataloğu winget'in okuduğu **tek** yerdir — Scoop'taki gibi "özel paket kovası ekle" düzeneği yoktur. Kataloğa yayım yapmak için kendi çatalından `microsoft/winget-pkgs:master` dalına bir PR açarsın, Microsoft'un doğrulama hattı çalışır, bir Microsoft inceleyicisi birleştirir. **Bu kapıyı atlatan bir otomasyon yoktur.**

Bu yüzden hattımızın iki aşaması vardır:

1. **Otomatik** — goreleaser yeni manifestoları [`agentteamland/winget-pkgs`](https://github.com/agentteamland/winget-pkgs) çatalımıza push'lar.
2. **Elle** — çatalımızdan `microsoft/winget-pkgs:master` dalına bir PR aç.

Bu sayfa 2. aşamayı kapsar.

## Goreleaser çatalımıza ne koyar?

Etiketlenmiş her `atl` sürümü için goreleaser, çatalımızın `master` dalında `manifests/a/AgentTeamLand/atl/<version>/` altında bir dizin ekler. Dizin, winget manifesto şemasına göre üç YAML dosyası içerir:

```
manifests/a/AgentTeamLand/atl/1.1.4/
├── AgentTeamLand.atl.yaml              # sürüm manifestosu
├── AgentTeamLand.atl.installer.yaml    # kurulumcunun üst bilgisi + URL'leri + SHA-256
└── AgentTeamLand.atl.locale.en-US.yaml # açıklama, lisans, paket adı (en-US)
```

Bu dosyalar şunları açıklar: hangi sürüm, kurulumcuyu nereden indireceğiz (cli GitHub Release yapıtlarını gösterir), SHA-256 ne, lisans bilgisi, paket adı.

Goreleaser bu dosyaları her etiketde çatalımızın `master` dalına kendiliğinden commit'ler ve zorla push'lar. **Bu noktada üst akış kataloğu sürümden hâlâ habersizdir.**

## Üst akış PR'ını açma

```bash
cd repos/winget-pkgs   # workspace'in ÇATALIMIZA olan klonu

# Çatalımızın master'ının goreleaser'ın push'ladığı içerikle güncel olduğunu garanti et:
git checkout master
git pull origin master

# Bir PR dalı oluştur:
git checkout -b atl-<version>
# (düzenleme gerekmez — goreleaser zaten manifestoları master'a commit'ledi)

# Dalı push'la:
git push -u origin atl-<version>

# Üst akış microsoft/winget-pkgs'a PR aç:
gh pr create \
  --repo microsoft/winget-pkgs \
  --base master \
  --head agentteamland:atl-<version> \
  --title "New version: AgentTeamLand.atl version <X.Y.Z>" \
  --body "Adds manifests for AgentTeamLand.atl <X.Y.Z>"
```

Microsoft'un doğrulama hattı hemen devreye girer. Yaygın denetimler:

- Manifesto şema doğrulaması.
- Hash doğrulaması (kurulumcu URL'sini yeniden indirir, SHA-256 hesaplar, karşılaştırır).
- Kurulumcu biçim denetimi (MSIX, MSI, APPX, MSIXBundle, APPXBundle ya da .exe olmalı).
- Tüm URL'lerin erişilebilirliği.
- Lisans uyumluluğu.
- Paket üst bilgisinin akıl sağlığı.

Doğrulama geçerse bir Microsoft inceleyicisi göz atar (tipik olarak 24-72 saat içinde). Birleştirir → sürüm, Microsoft'un CDN'i yayıldığında (genellikle birleşmenin ardından dakikalar içinde) `winget install AgentTeamLand.atl` ile erişilebilir hâle gelir.

## Doğrulama başarısız olursa ne olur?

Doğrulama botu PR'a başarısızlık nedenini içeren bir yorum bırakır. Yaygın olanları:

- **Hash uyuşmazlığı** — kurulumcu manifestosundaki SHA-256'nın goreleaser'ın hesapladığıyla eşleştiğini yeniden denetle. Genelde, çatalımızın manifestosu commit'lendikten sonra goreleaser'ın yapıtı yüklediği anlamına gelir; çatalımızın `master` dalını taze çek ve PR'ı yeniden aç.
- **Erişilemeyen URL** — cli GitHub Release'inin beklenen dosya adıyla var olduğunu denetle. Bazen goreleaser, tüm yapıtların yüklenmesi tamamlanmadan manifestoyu yükler.
- **MSIX/EXE reddi** — `.exe` desteklenir ama kurulumcu [winget'in kurulumcu gereksinimlerini](https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/schema/1.6.0/singleton.md) karşılamalıdır. `atl` için doğrulamadan temiz geçen taşınabilir bir `.exe` yayımlıyoruz; ileride kurulumcu biçimi değişirse doğrulama yakalar.

Düzeltme gerekirse çatalımızın dalındaki manifestoları düzenle, push'la, bot kendiliğinden yeniden doğrular.

## cli sürümü ile winget erişilebilirliği arasındaki gecikme

Tipik: cli etiketinden winget kataloğunda görünürlüğe 1-3 gün.

Ayrıntılı zaman çizelgesi:

1. **t = 0**: cli deposuna etiket push'landı.
2. **t + 5 dk**: goreleaser bitti; manifestolar çatalımızın `master` dalında; brew + scoop zaten güncellendi.
3. **t + 10 dk**: bakımcı üst akış PR'ını açar (elle adım).
4. **t + 10 dk - 24 saat**: Microsoft doğrulama hattı çalışır.
5. **t + 24 saat - 72 saat**: Microsoft inceleyicisi birleştirir (kuyruğa bağlı olarak bazen daha hızlı, bazen daha yavaş).
6. **t + ~72 saat**: `winget upgrade atl` yeni sürümü çeker.

Bu gecikme [Kurulum sayfasında](../guide/install) **beklenir ve belgelenmiştir** — belgeler açıkça "winget bir-iki `v*` etiketin gerisinde olabilir" der. Mutlak en son sürüme ihtiyacı olan kullanıcılar PowerShell tek-satırlığına ya da scoop'a yönlendirilir.

## Tarihçe — geçmiş üst akış PR'ları

| atl sürümü | PR | Durum | Notlar |
|---|---|---|---|
| v0.1.1 | [#361975](https://github.com/microsoft/winget-pkgs/pull/361975) | 2026-04-24'te birleştirildi | Katalogdaki ilk sürüm. |
| v0.2.0 | [#364841](https://github.com/microsoft/winget-pkgs/pull/364841) | 2026-04-25'te birleştirildi | İkinci sürüm. |
| v0.3.0 / v1.0.0 / v1.1.0 / v1.1.1 / v1.1.2 / v1.1.3 | _(ayrı geri PR yapılmadı)_ | çatalda korundu | winget sözleşmesi gereği ara sürümler çatalın dallarında tutulur ama ayrı bir üst akış PR'ı yapılmaz — bir sonraki "gerçek" üst akış PR'ı onları üstü kapalı taşır. |
| v1.1.4 | [#367931](https://github.com/microsoft/winget-pkgs/pull/367931) | inceleme bekliyor | Üst akış kataloğunu mevcut yayımlanan sürüme yetiştirir. |

## İlgili

- [Sürüm yayım hattı](release-pipeline) — tam goreleaser akışı (brew + scoop + winget).
- [`atl`'yi kur](../guide/install) — kullanıcıya yönelik yönergeler, "winget gerisinde kalabilir" notu dâhil.
- [Microsoft'un winget-pkgs README'si](https://github.com/microsoft/winget-pkgs) — üst akış kuralları ve biçimi.
- [winget-cli sürümleri](https://github.com/microsoft/winget-cli/releases) — kataloğdan çekim yapan istemci tarafı.
