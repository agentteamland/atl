# atl nedir?

`atl`, bir projeye **AI ajan takımları** kuran komut satırı aracıdır — tıpkı `npm`'in JavaScript paketlerini ya da `brew`'un Unix ikililerini kurması gibi.

## Sorun

Claude Code'u iyi kullanabilmek yapılandırma gerektirir: modelin kod tabanın üzerinde nasıl akıl yürüteceğini biçimlendiren ajanlar, beceriler ve kurallar. Sonunda dosyaları projeler arasında elle kopyalıyor, başka birinin kurulumunu çatallıyor (fork) ve zamanla bunların birbirinden uzaklaştığını görüyorsun. Her yeni proje, bir öncekinin çoktan çözdüğü sorunları yeniden çözüyor.

## Yanıt

Bir **takım**, belirli bir iş türünün etrafında kurulmuş ajan, beceri ve kural paketidir. Bir takım, Docker Compose üretim düzeniyle bir .NET + Flutter + React yığınına yönelik olabilir. Bir başkası, Next.js + Sanity + Vercel blog yığınına yönelik olabilir. Üçüncüsü, Airflow ve dbt ile veri boru hatlarına.

`atl install some-team` takımı GitHub destekli bir katalog üzerinden çözümler, kaynağını getirir ve içindeki ajanları, becerileri ve kuralları içinde bulunduğun projenin `.claude/` dizinine kopyalar. Editörü açtığın an Claude Code takımı görür.

Takımın yazarı bir düzeltme yayımladığında `atl update` komutunu çalıştırırsın; o takımı kullanan her proje değişikliği alır. Projelerinin birbirinden uzaklaşması durur.

## Kapalı bir bahçe değil

Her takım, kök dizininde `team.json` dosyası bulunan herkese açık bir GitHub deposundan ibarettir. Başvurulacak merkezi bir kayıt defteri yoktur: bir depoyu [`atl-team`](https://github.com/topics/atl-team) GitHub konusuyla etiketle, oluşturulan **katalogda** görünsün; oradan herkes `<handle>/<name>` biçimiyle bulup kurabilsin. `atl search` o katalogu sorgular; `atl install` onu çözümler. CLI, MIT lisanslı Go ile yazıldı. Takım sözleşmesi burada belgelenmiş — bkz. [`team.json` referansı](/tr/authoring/team-json).

## Kim için?

- Her proje için Claude Code kurulumunu elle hazırlamak istemeyen **geliştiriciler**.
- Şirketlerinin Claude kullanımını tüm depolarda standartlaştırmak ve yeni mühendisleri dakikalar içinde sürece dahil etmek isteyen **takım liderleri**.
- Çerçeve yazarlarının bugün CLI yayımladığı gibi, fikir sahibi ajan takımları yayımlamak isteyen **yığın yazarları**.

## Bugün nerede?

`atl` **v2** sürümünde — tek bir monorepo ([`agentteamland/atl`](https://github.com/agentteamland/atl)), şu an **alpha** aşamasında. Kurulum topolojisi proje-yerel kopyalardan oluşur (diskte kalıcı klon önbelleği yoktur — kaynaklar kurulumdan sonra atılır), otomatik güncelleme yolu Claude Code'un `SessionStart` ve `UserPromptSubmit` hook'larından geçer, öğrenme döngüsü ise oturum bilgisini kalıcı biçimde saklar: satır içi işaretçiler (marker) kuyruğa alınır ve `/drain` becerisi her birini günlük, wiki ve ajan bilgi tabanlarına işler.

v1 dönemi first-party takımlar Temmuz 2026'da emekliye ayrıldı; katalog bugün açık yayımlamayla büyüyor — [`atl-team`](https://github.com/topics/atl-team) topic'iyle etiketlenmiş herkese açık her depo kataloğa girer. Platformun tamamı MIT lisanslı ve katkılara açık.

Sıradakiler:
- **[`atl`'yi kur](/tr/guide/install)**
- **[Hızlı başlangıç — 60 saniyede çalışan bir takım](/tr/guide/quickstart)**
