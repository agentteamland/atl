# Hızlı başlangıç

Sıfırdan üretime hazır bir ajan takımına bir dakikadan kısa sürede — CLI'ı kur, ilk takımını kur, bir oturum aç.

## 1. `atl`'yi kur

`atl`, hiçbir çalışma-zamanı bağımlılığı olmayan tek statik bir Go ikili dosyasıdır. Platformuna uygun tek-satırlık betikle kur:

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Elle kurmayı mı tercih edersin? [GitHub Releases](https://github.com/agentteamland/atl/releases/latest) sayfasından önceden derlenmiş bir ikili al ve `PATH`'ine koy. Homebrew, Scoop ya da winget kanalı yok — desteklenen yol betik (ya da release ZIP'i). Tüm ayrıntılar: [Kurulum rehberi](/tr/guide/install).

Doğrula:

```bash
atl --version
```

## 2. Bir proje dizini oluştur

```bash
mkdir my-new-app && cd my-new-app
```

`atl` bir projenin içinde çalışır. Bir takım kurduğunda, takımın ajanlarını, becerilerini ve kurallarını bu projenin `.claude/` dizinine yazar — tam da Claude Code'un onları okuduğu yere.

## 3. Bir takım bul

```bash
atl search dotnet
```

[`atl search`](/tr/cli/search), [`atl-team`](https://github.com/topics/atl-team) konu etiketiyle işaretlenmiş herkese açık depolardan üretilen GitHub tabanlı kataloğu sorgular. Her sonuç, `<handle>/<team>@<version>` referansını (handle, takımın GitHub sahibidir — sahiplik yazarlıktır) ve kopyalanacak tam `atl install` komutunu yazdırır. Artık PR açılacak merkezi bir kayıt-defteri deposu yok; bir deponun listelenmesi, onu `atl-team` ile etiketlemek (ya da içinde [`atl publish`](/tr/cli/publish) çalıştırmak) yoluyla olur.

## 4. Takımı kur

```bash
atl install agentteamland/software-project-team
```

Birkaç saniye içinde:

- Takım, indeksten çözümlenir ve indirilir (yeniden kullanım için önbelleğe alınır).
- 13 ajan ve 3 beceri (`create-new-project`, `verify-system`, `design-screen`) `.claude/` dizinine yazılır.
- Temel dosya hash'lerinden oluşan bir manifest `.atl/` altına kaydedilir; böylece ileride güncellemeler senin düzenlemelerini yukarı-akış (upstream) değişikliklerinden ayırt edebilir.
- Otomasyon hook'ları, kurulumun bir parçası olarak Claude Code'a bağlanır — otomasyon, opt-in değil, varsayılan olarak açıktır.

Artık projeye bağlı tam bir .NET + Flutter + React + Docker ajan takımın var.

::: tip Global ve proje kapsamı
Bir takım, yayıncısının varsayılanının işaret ettiği yere kurulur. `--global` (her proje) ya da `--project` (yalnızca bu proje) ile bastır; bir takım her iki katmanda da bulunduğunda, proje kopyası global olanı gölgeler. Kapsam ekseni için [Kavramlar](/tr/guide/concepts) sayfasına bak.
:::

## 5. Ne kurduğunu gör

```bash
atl list
```

[`atl list`](/tr/cli/list), her kapsamda kurulu takımları gösterir — global (`~/.claude`) ve proje (`<cwd>/.claude`). Her ikisinde de bulunan bir takım, her birinin altında listelenir.

## 6. Claude Code'da kullan

Bu dizinde Claude Code'u aç. Takımın becerileri eğik çizgili komut olarak hazırdır:

- `/create-new-project MyApp` — tam bir yığını iskeletler (topla → iskeletle → derle → doğrula → commit'le).
- `/verify-system` — konteynerler, portlar, uygulamalar ve boru hatları üzerinde uçtan uca bir sağlık denetimi çalıştırır.

Takımın gönderdiği her ajan (api-agent, socket-agent, worker-agent, flutter-agent, react-agent, infra-agent, database-agent, redis-agent, rmq-agent, code-reviewer, project-reviewer, design-system-agent, ux-agent) Claude'un devredebilmesi için hazırdır.

Platformun kendi global becerileri de orada — `/drain`, `/create-pr`, `/create-code-diagram`, `/brainstorm`, `/rule`, `/rule-wizard` — hangi takımı kurduğundan bağımsız olarak her projede kullanılabilir.

## 7. Bırak öğrenme döngüsü kendini çalıştırsın

İşte kurulumunu zamanla bozulmak yerine sürekli daha iyi yapan kısım bu. Sen çalışırken ajanlar öğrendiklerini dayanıklı bir kuyruğa yazar. 4. adımdaki otomasyon hook'ları sayesinde bunların hiçbirini elle yönetmezsin:

- Oturum içinde (ve `atl tick` ile) bir bakım **tick**'i çalışır ve kuyruktaki öğrenmeleri bilgi tabanına katar.
- `atl doctor` kurulumu kendiliğinden onarır — hatırlaman gereken bir komut değil, daima açık çalışan sağlık daemon'udur.
- Bir şey seni beklediğinde `atl`, `N learning(s) pending` (N öğrenme bekliyor) der; `/drain` becerisi (oturumunda onu çalıştır) her öğeyi doğru yere yönlendirir — bir wiki sayfasına, günlüğe ya da bir ajanın bilgi tabanına — ardından öğeyi kuyruktan siler.

Kuyruğa istediğin zaman göz at:

```bash
atl learnings status
```

`atl learnings peek` bekleyen öğeleri listeler, `atl learnings ack <id>` ise bir öğeyi işlenmiş olarak işaretler.

## 8. Güncel kal

Bir takım yazarı iyileştirmeler gönderdiğinde:

```bash
atl update
```

Kurulu tüm takımlar tazelenir; değiştirmediğin kopyalar yerinde güncellenir ve yerel düzenlemelerin korunur. Projenin kendi kodunda hiçbir şey değişmez.

## Az önce ne oldu?

Tek bir komutla, özenle seçilmiş ve sürüme sabitlenmiş bir ajan kümesini bir projeye kurdun ve kendi kendine çalışan bir bakım döngüsünü açtın. Aynı takımı kuran diğer her proje aynı yapılandırmayı alır — ve yazar gönderdiğinde aynı güncellemeleri — ajanlarının öğrendiği kazanımlar ise `atl promote` ve `atl publish` aracılığıyla geri dolaşıma girer.

## Tasarım araçları ekle (isteğe bağlı)

Tasarım-sistemi ve ekran-prototipi araçları için `design-system-team`'i kur:

```bash
atl install agentteamland/design-system-team
```

Ardından Claude Code sohbetinde:

```
/dst-init
/dst-new-ds primary
/dst-new-prototype --ds primary login-screen
/dst-open
```

`.dst/` altında token-hizalı tasarım sistemleri ve çok-durumlu HTML prototipler elde edersin; herhangi bir tarayıcıda görüntülenebilir. Tüm beceri kümesi için [design-system-team](/tr/teams/design-system-team) sayfasına bak.

## Sıradaki

- **[Takımlara göz at](/tr/teams/)** — kurabileceğin kataloglanmış takımlar.
- **[Kavramlar](/tr/guide/concepts)** — takımlar, ajanlar, beceriler, kurallar ve global/proje kapsam ekseni.
- **[CLI referansı](/tr/cli/overview)** — her komut ayrıntısıyla.
- **[Takım yazma](/tr/authoring/creating-a-team)** — kendi takımını yaz ve yayımla.
