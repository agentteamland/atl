# atl nedir?

`atl`, bir projeye **AI ajan takımları** kuran komut satırı aracıdır — tıpkı `npm`'in JavaScript paketlerini ya da `brew`'un Unix ikililerini kurması gibi.

## Sorun

Claude Code'u iyi kullanabilmek yapılandırma gerektirir: modelin kod tabanın üzerinde nasıl akıl yürüteceğini biçimlendiren ajanlar, beceriler ve kurallar. Sonunda dosyaları projeler arasında elle kopyalıyor, başka birinin kurulumunu çatallıyor (fork) ve zamanla bunların birbirinden uzaklaştığını görüyorsun. Her yeni proje, bir öncekinin çoktan çözdüğü sorunları yeniden çözüyor.

## Yanıt

Bir **takım**, belirli bir iş türünün etrafında kurulmuş ajan, beceri ve kural paketidir. Bir takım, Docker Compose üretim düzeniyle bir .NET + Flutter + React yığınına yönelik olabilir. Bir başkası, Next.js + Sanity + Vercel blog yığınına yönelik olabilir. Üçüncüsü, Airflow ve dbt ile veri boru hatlarına.

`atl install some-team` takımı indirir, önbelleğe alır ve içindeki ajanları, becerileri ve kuralları içinde bulunduğun projenin `.claude/` dizinine kopyalar. Editörü açtığın an Claude Code takımı görür.

Takımın yazarı bir düzeltme yayımladığında `atl update` komutunu çalıştırırsın; o takımı kullanan her proje değişikliği alır. Projelerinin birbirinden uzaklaşması durur.

## Çatallama (fork) değil

Takımlar başka takımları **genişletebilir**. `software-project-team`'in %95'ini istiyor ama tek bir ajanı farklı olsun ve bir beceri daha eklensin diyorsan, onu `extends` ile genişleten küçük bir takım yazarsın, o tek ajanı bastırırsın ve beceriyi eklersin. Üst takım kendi deposunda gelişmeye devam eder; alt takım güncellemeleri bedava alır.

## Kapalı bir bahçe de değil

Her takım, kök dizininde `team.json` dosyası bulunan bir Git deposundan ibarettir. Takımlara `software-project-team` gibi kısa adlar verilebilmesi için herkese açık bir [kayıt defteri](https://github.com/agentteamland/registry) var, ama istediğin herhangi bir Git URL'sinden de kurulum yapabilirsin. Şema herkese açık. CLI, MIT lisanslı Go ile yazıldı. Spesifikasyonun tamamı burada belgelenmiş durumda.

## Kim için?

- Her proje için Claude Code kurulumunu elle hazırlamak istemeyen **geliştiriciler**.
- Şirketlerinin Claude kullanımını tüm depolarda standartlaştırmak ve yeni mühendisleri dakikalar içinde sürece dahil etmek isteyen **takım liderleri**.
- Çerçeve yazarlarının bugün CLI yayımladığı gibi, fikir sahibi ajan takımları yayımlamak isteyen **yığın yazarları**.

## Bugün nerede?

`atl` **v1.1.x** sürümünde — kararlı sürüm. Kurulum topolojisi proje-yerel kopyalardan oluşur (kaynak doğruluk olarak global önbellek kullanılır), otomatik güncelleme yolu Claude Code'un `SessionStart` ve `UserPromptSubmit` hook'larından geçer, kendini güncelleyen öğrenme döngüsü oturum bilgisini günlük, wiki, ajan çocukları ve beceri öğrenimleri katmanlarına yazar.

Kayıt defterinde bugün iki doğrulanmış takım yayında: `software-project-team` (13 ajan — .NET API + Flutter + React + Docker yığını) ve `design-system-team` (2 ajan + tasarım sistemi ve prototip araçları için 10 `/dst-*` becerisi). Ekosistemin tamamı MIT lisanslı ve PR'lara açık.

Sıradakiler:
- **[`atl`'yi kur](/tr/guide/install)**
- **[Hızlı başlangıç — 60 saniyede çalışan bir takım](/tr/guide/quickstart)**
