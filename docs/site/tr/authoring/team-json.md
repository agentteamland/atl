# `team.json`

Her takım, kökünde bir `team.json` bulunan bir Git deposudur. Bu dosya tüm sözleşmedir: takımın adı, ne yayımladığı, neye bağlı olduğu ve varsayılan olarak nereye kurulduğu.

## En küçük örnek

```json
{
  "schemaVersion": 1,
  "name": "my-team",
  "version": "0.1.0",
  "description": "A starter team for small Next.js projects.",
  "author": { "name": "Your Name", "url": "https://github.com/you" },
  "agents": [
    { "name": "web-agent", "description": "Next.js + Tailwind reviewer and builder." }
  ]
}
```

Bu kadarı kuruluma yeter. CLI manifest dosyasını çözümler, `agents/web-agent.md` (ya da `agents/web-agent/agent.md`) dosyasını ilgili kapsamın `.claude/agents/` dizinine kopyalar ve kurulumu kapsama özgü bir manifest dosyasına kaydeder.

## Tam alan başvurusu

| Alan | Tür | Zorunlu | Açıklama |
|---|---|---|---|
| `schemaVersion` | tam sayı | ✅ | Şu an `1`. Yalnızca manifest yapısında geriye dönük uyumsuz bir değişiklik olduğunda artırılır. |
| `name` | dize | ✅ | Takımın katalog adı. Küçük harf kebab-case. GitHub kullanıcı adınızla birleşerek `<handle>/<name>` kurulum referansını oluşturur. |
| `version` | semver dizesi | ✅ | SemVer 2.0.0 (`1.2.3`, `1.2.3-beta.1`). `atl update` güncelleme gerekip gerekmediğine bunu karşılaştırarak karar verir. |
| `description` | dize | ✅ | `atl search` çıktısında görünen tek cümlelik tanıtım. Kısa tut — katalog çıktısında tek satırdır. |
| `author` | nesne | — | Kurulum ayrıştırıcısının şu an okumadığı isteğe bağlı üst veri. Verilirse `{ "name": "...", "url": "...", "email": "..." }` nesnesi geleneksel biçimdir; düz bir dize de kabul edilir (sessizce yoksayılır), reddedilmez. |
| `license` | SPDX dizesi | — | `"MIT"`, `"Apache-2.0"` vb. Geleneksel üst veri — CLI ve katalog onu okumaz. Depoda yanına bir LICENSE dosyası koyun. |
| `keywords` | dize[] | — | `atl search` eşleşmesi için. `["nextjs", "tailwind", "blog"]`. |
| `repository` | dize | — | Takımın kaynak URL'si. Geleneksel üst veri — katalog, kaynak depoyu bu alandan değil keşfedilen GitHub deposunun kendisinden türetir. |
| `homepage` | dize | — | Belge / açılış URL'si. |
| `agents` | nesne[] | — | Her biri: `{ name, description }`. Adlar `agents/` altındaki dosya ya da dizinlerle eşleşmelidir. |
| `skills` | nesne[] | — | Her biri: `{ name, description }`. Adlar `skills/` altındaki dizinlerle eşleşmelidir. |
| `rules` | nesne[] | — | Her biri: `{ name, description }`. Adlar `rules/` altındaki dosyalarla eşleşmelidir. |
| `scope` | dize | — | Yayıncı varsayılan kurulum katmanı: `"project"`, `"global"` ya da `"both"`. Varsayılan `"project"`. Kullanıcı kurulum sırasında `--global` / `--project` ile her zaman geçersiz kılabilir. |
| `dependencies` | nesne | — | CLI'nin bu takımın yanına kurması gereken diğer takımlar için `team-name → version-constraint` eşlemesi. |
| `requires.atl` | dize | — | Bildirilen en düşük `atl` sürümü. Örneğin `">=2.0.0"`. Geleneksel üst veri — kurulum ayrıştırıcısı şu an bunu dayatmaz. |
| `capabilities` | nesne | — | Çoğunlukla kurulum ayrıştırıcısının değil, platformun becerilerinin okuduğu isteğe bağlı sözleşmeler. `capabilities.review: "<agent>"`, [`/create-pr`](/tr/skills/create-pr)'in bu takımın uzman gözden geçireni olarak başlattığı ajanı adlandırır; `capabilities.profile`, profil katmanı sağlayıcı/tüketici rolünü bildirir ([profile-team](/tr/teams/profile-team)'e bakın). **CLI**'ın kendisinin okuduğu üç anahtar var: `store`, `channel` ve `sessionScript` — aşağıya bakın. |
| `backends` | dize[] | — | `backends/<name>/` altında arka uca özel bağdaştırıcı paketleri gönderen takımlar için — `backends/stripe/` ve `backends/paddle/` taşıyan bir takım `["stripe", "paddle"]` bildirir. Bugün yalnızca bilgilendirme amaçlıdır — kurulum ayrıştırıcısı bunu okumaz. |

::: tip Açıklamayı kısa tut
`description`, `atl search` çıktısında tek satır olarak gösterilir; uzun bir açıklama garip biçimde kırılır. Bir tanıtım cümlesini hedefle — paragraf değil.
:::

## Kalıcı depo bildirmek {#kalici-depo-bildirmek}

Çoğu takım her şeyi, ATL'nin zaten izlediği yansıtılmış `.claude` ağacının içinde tutar. Kendi uzun ömürlü verisini bunun dışında bir yerde tutan bir takım — ilki profile-team'in `~/.atl/profiles/`'i — o konumu bildirir:

```json
{
  "capabilities": {
    "profile": { "role": "provider", "store": "~/.atl/profiles" }
  }
}
```

`store` herhangi bir yetenek adının altında yer alabilir, tek bir yol tutar ve ev dizini için `~` kullanabilir. `atl install` bildirilen her depoyu kurulum manifest'ine kaydeder; bundan sonra **oturum başlangıcı ve `atl tick`, o dizini yerel git altında tutar** — oturum başına bir kez ve tick'in throttle penceresi başına bir kez — ve son geçişten bu yana değişen ne varsa commit'ler.

Amaç geri getirilebilirliktir. Yazma politikası "son yazan kazanır" olan bir depo, her üzerine yazmada önceki değeri kaybeder: çıkarımla konmuş bir geçici değerin yerine doğrulanmış bir yanıt geldiğinde, yerine geçtiği şeyden hiçbir iz kalmaz. Dizin sürümlendiğinde eski değer bir `git show HEAD~1` uzaklıktadır.

::: tip Bu özellik gelmeden önce mi kurmuştunuz?
Bildirim kurulum anında okunur, dolayısıyla `stores` alanından önceki bir kurulumda bu kayıt yoktur. `atl update` bunu bir kez geri doldurur — sabitlenmiş kaynağı yeniden çekerek — ve kendiliğinden çalıştığı için sizin bir şey yapmanız gerekmez. O çalışana kadar depo henüz sürümlenmiyor demektir.
:::

Bunun bilinçli olarak **yapmadıkları**:

- **Dizini asla oluşturmaz.** Olmayan bir depo, özelliğin bu makinede kullanılmadığı anlamına gelir; onu oluşturmak hem diski kirletir hem de özelliği etkinmiş gibi gösterir.
- **Asla remote tanımlamaz ve asla push etmez.** Bir depo genellikle kullanıcının en hassas verisini tutar; bir kopyayı makinenin dışına taşımak, kullanıcının ayrıca istemesi gereken ayrı bir eylemdir (profile-team'in [`/profile-backup`](/tr/teams/profile-team)'ı böyle bir yoldur ve herkese açık bir depoya yazmayı reddeder).
- **Yalnızca kendi oluşturduğu repo'ya yazar.** Depoyu zaten kendi sürüm kontrolünüzde tutuyorsanız ATL ona hiç dokunmaz — amacı siz zaten karşılamışsınızdır ve dalınızı ilerletmek, devam eden çalışmanızın altından `HEAD`'i kaydırırdı. ATL kendi oluşturduğu repo'ları `.git/atl-store` dosyasıyla işaretler; o dosyayı silerseniz o depoya commit'lemeyi bırakır.
- **Başka bir repo'nun içinde duran bir depoya asla dokunmaz.** Orada `git init` yapmak dıştaki repo'yu gölgelerdi.
- **Git durumunuzu asla bozmaz.** Anlık görüntü, tek kullanımlık bir index üzerinde plumbing komutlarıyla yazılır; staging alanınıza dokunulmaz ve hiçbir hook çalışmaz. Merge, rebase veya cherry-pick ortasındaki bir repo, o durumdan çıkana kadar atlanır.
- **Kimseye erişim vermez.** ATL bildirilen *yolu* yalnızca bu tek mekanik amaç için okur. Bir depoyu kimin okuyup yazabileceği, platformun henüz uygulamadığı ayrı bir sözleşmedir.

Bildirilen yollar bunların hiçbiri çalışmadan önce denetlenir: önce sembolik bağlar çözülür, ardından hedefin ev dizininizin en az iki seviye altında olması ve içinde bulunduğunuz çalışma dizinini kapsamaması aranır. Yani `~`, en üst seviyedeki tek bir dizin, ev dizininizin tamamen dışındaki bir yer veya üzerinde çalıştığınız projenin kendisi repo'ya çevrilmez, reddedilir — sessizce, hiçbir uyarı olmadan. Bu bir kum havuzu (sandbox) değildir: bu kuralları sağlayan bir dizin, içeriği ne olursa olsun kabul edilir; çünkü deposunu meşru şekilde orada tutan bir takım da tam olarak böyle görünür. Tüm geçişi kapatmak için `ATL_NO_STORE_GIT=1`.

::: warning Ev dizininizin dışındaki bir depo sürümlenmez
Depoyu harici bir diskte veya bir senkronizasyon klasöründe tutuyorsanız — doğrudan bildirilmiş ya da sembolik bağla ulaşılıyor olsun — denetlenen bölgenin dışında kalır ve hiç sürümlenmez. Bugün bunun için bir uyarı yok.
:::

Bilinen bir sınır: depo kendi içinde bir git repo'su barındırıyorsa, o alt ağaç gitlink olarak kaydedilir ve içeriği anlık görüntüye girmez. Bir depo makine tarafından yazılan veridir, dolayısıyla pratikte bu durum oluşmaz — ama oluşursa sessizdir.

CLI bundan takımın hakkında hiçbir şey öğrenmez — ne ad, ne anlam, ne de dizinin ne tuttuğu bilgisi. Bir bildirime uyar; gelecekteki herhangi bir takımın aynı davranışı bedava almasının nedeni budur.

## Yakalama kanalı bildirmek {#declaring-a-capture-channel}

ATL'nin yakalama döngüsü — konuşma sırasında düşürülen bir işaretçi, kalıcı kuyruğa aktarılması, arka planda bir beceri tarafından drain edilmesi — yalnızca çekirdeğe ait değildir. Çekirdeğin sahip olduğu tek kanal `learning`'dir. Bir takım, bildirerek kendi kanalına sahip olabilir:

```json
{
  "capabilities": {
    "profile": {
      "role": "provider",
      "channel": {
        "name": "profile-fact",
        "drain": "/profile-drain",
        "rule": "profile-capture",
        "describes": "durable entity facts"
      }
    }
  }
}
```

`store` gibi `channel` da herhangi bir yetenek adının altında yer alabilir. Dört alanının her biri bir yerde tüketilir:

| Alan | Neyi besler |
|---|---|
| `name` | kuyruk kanalı ve yakalama geçişinin aradığı işaretçi öneki — `<!-- profile-fact: … -->`. Ayrıca `atl learnings peek --channel <name>`'in kabul ettiği değer. |
| `drain` | ajana arka planda bir alt-ajan olarak başlatması söylenen beceri. |
| `rule` | her iki sinyalin de "ne yapılacağını söyleyen şey" olarak adlandırdığı kural. |
| `describes` | bekçinin "son turları şu kaçırılanlar için gözden geçir: **…**" cümlesindeki insan-okur etiket. |

`atl install` bildirimi kurulum manifest'ine kaydeder; bundan sonra [`atl tick`](/tr/cli/tick) ve oturum başlangıcı o kanalın sinyallerini çekirdeğinkinin yanında basar:

```
atl: 2 profile-fact(s) pending — auto-drain them now in a background subagent (per the profile-capture rule)
atl: capture-watchdog (profile-fact) — no profile-fact markers for 4 assistant turn(s) / ~2100 chars of user input; review the recent turns for missed durable entity facts and mark them, and spawn ONE background /profile-drain subagent to mine the stretch (per the profile-capture rule, valid even with an empty queue)
```

İş bölümü işin bütün özüdür. **Sinyali platform basar; ona göre davranan, takımın kuralıdır.** ATL bir kanalın dört sözcüğünü bilir, fazlasını değil — hangi takımın gönderdiğini değil, `profile-fact`'in ne olduğunu değil, `/profile-drain`'in onunla ne yaptığını da değil. Bir sinyali davranışa çeviren yönerge, bildirimin adlandırdığı kuralda gelir; o kural da kanalın sahibi olan takımla birlikte kurulur. Dolayısıyla o takımın kurulu olmadığı bir makine sinyali hiç görmez: bildirim yoksa kanal yoktur, davranılacak bir şey de yoktur.

Bu yön, **bildirilmemiş** bir kanaldaki işaretçinin başına ne geleceğini de belirler: hiçbir şey. Kuyruğa hiç girmez; böylece bir yazım hatası (`profile-fact` yerine `profile-fct`) bir olgunun yakalanmış *göründüğü* ama hiçbir drain'in asla sahiplenmeyeceği hayalet bir kanal açamaz. Etkin bir kanala çok yakın düşen bir yazım, hata sessizce yutulmak yerine raporlanır; `atl learnings peek --channel` ise hiçbir şeyle eşleşmeyen bilinmeyen bir kanalı, yazım hatasına "bekleyen öğe yok" yanıtını vermek yerine reddeder. (Kuyrukta öğesi *olan* ama artık etkin olmayan bir kanalı — örneğin öğeleri beklerken kaldırılmış bir takımın kanalını — yine de okur. O öğeler gerçektir ve `atl learnings status` da aynı nedenle onları listeler.)

`drain`, `rule` ya da `describes` alanı eksik bir bildirim, içi boş bir cümle basılmaktansa reddedilir; adı zaten alınmış bir kanal — çekirdeğin `learning`'i ya da kurulu başka bir takım tarafından — yok sayılır. İkisi de [`atl doctor`](/tr/cli/doctor)'da bir uyarı olarak görünür; çünkü aksi hâlde her ikisi de sessiz kayıptır: o kanala yazılan işaretçiler hiç yakalanmaz ve nedenini kimse söylemez.

`atl doctor`'ın denetleyemediği şey, `rule` ve `drain` alanlarının var olan varlıklara işaret edip etmediğidir: o, takımın kaynak ağacının çoktan gitmiş olduğu *kurulu* bir manifesti okur. Takımın hiç yayımlamadığı bir kurala işaret eden bir bildirim, ikisinden daha kötü olanıdır — kanal etkinleşir, işaretleri *gerçekten* yakalanır, ama hiçbir şey bir agent'a onları boşaltmasını söylemez. Bu monorepo'daki birinci-parti takımlar için [`atl skills check`](/tr/cli/skills) her iki adı da `rules/` ve `skills/` altında çözümler ve CI'ı kırar. Kendi deposundan yayımlanan bir takımın böyle bir kapısı yoktur; o yüzden kendiniz denetleyin: adını verdiğiniz kural, yayımladığınız bir dosya olmalı.

::: tip Bu özellik gelmeden önce mi kurmuştunuz?
`store` ile aynı hikâye: bildirim kurulum anında okunur, dolayısıyla `channels` alanından önceki bir kurulumda bu kayıt yoktur ve tam olarak hiç kanal bildirmeyen bir takım gibi davranır. `atl update` sabitlenmiş kaynağı yeniden çekerek bunu bir kez geri doldurur ve kendiliğinden çalışır — o çalışana kadar o kanalın işaretçileri yakalanmaz.
:::

**Kimseye erişim vermez.** ATL bildirilen *sözcükleri* yalnızca bu tek mekanik amaç için okur — bir sinyali sözcüklendirmek ve hangi kanalların var olduğuna karar vermek. Bir yeteneğin adlandırdığı şeyi kimin okuyup yazabileceği, platformun henüz uygulamadığı ayrı bir sözleşmedir.

## Oturum betiği bildirmek {#declaring-a-session-script}

Yukarıdaki iki bildirim ATL'ye bir *yol* ve bir *sözcük kümesi* verir. Üçüncüsü ise çalıştırılacak bir şey verir: ATL'nin oturum başlangıcında çalıştırdığı ve yazdırdığı her şeyi oturumun bağlamına ilettiği bir betik.

```json
{
  "capabilities": {
    "migrations": { "sessionScript": "scripts/session-brief.sh" }
  }
}
```

`store` ve `channel` gibi `sessionScript` de herhangi bir yetenek adının altında durabilir. Değeri, **takımınızın varlıklarına göreli** bir yoldur — `scripts/session-brief.sh`, takım deponuzda o yoldaki dosyadır; kurulumdan sonra ise kapsamın `.claude/scripts/` dizinindeki kopyasıdır. Mutlak bir yol ya da `..` ile dışarı tırmanan bir yol reddedilir: bildirim, sizin dağıttığınız bir dosyayı adlandırır, kullanıcının makinesindeki bir dosyayı değil.

Ne işe yaradığı, her oturumda yüklenen bir kuralın yapamadığı şeydir: *şu ana dair bir olguyu* bildirmek. Bir projenin veritabanı migration'larını üstlenen bir `acme/example-team` düşünün: oturum brifingi, yerel veritabanının gerçekte hangi migration'da olduğunu söyler ve az önce geçiş yaptığınız dal daha ilerisini bekliyorsa uyarır. Bunların hiçbiri bir ajanın okuyabileceği bir dosyadan bilinemez — hangi dalda olduğunuza ve yalnızca bu makinede bulunan bir veritabanının durumuna göre değişir.

Çıktı sözleşmesi kısadır ve tamamı şudur: **bir betik yalnızca başarılı olarak konuşur.**

- ATL **stdout**'u iletir, yalnızca **sıfır çıkış kodunda**. Sıfır olmayan bir çıkış, çıktıyı atar; böylece yarım kalmış bir okuma oturuma asla tamamlanmış gibi ulaşmaz.
- **stderr atılır.** Tanılamalarınız sizin içindir; bir kancanın çıktısı ise bir ajan tarafından olgu olarak okunur.
- Geçişin tamamı **süre sınırlıdır** ve çıktısı **boyut sınırlıdır**. Asılı kalan ya da taşıran bir betik sessizliğe dönüşür.
- Her başarısızlık — eksik dosya, kaybolan `+x`, sıfır olmayan çıkış, zaman aşımı — **sessizdir**, çünkü bu bir kanca içinde çalışır ve bir kanca bir oturumu asla engellememeli ya da başarısız etmemelidir.

Bu yüzden betiği, **söyleyecek bir şeyi yokken çıktısız biçimde 0 ile çıkacak** ve ağa ancak söyleyecek bir şeyi olduğunu bildikten sonra ulaşacak şekilde yazın. Çalışma dizini proje kökü olduğundan, buna karar vermek genellikle iki yerel okumadır.

::: warning Sessizlik aynı zamanda bozulma biçimidir
Bu başarısızlıkların her biri, çalışıp da söyleyecek bir şeyi olmayan bir betiğe birebir benzer. Farkı görünür kılmak için iki yüzey vardır, çünkü başka hiçbir şey kılamaz: [`atl skills check`](/tr/cli/skills) bildirimi takımın kendi ağacına karşı çözer ve birinci taraf bir takım için **CI'ı kırar**; [`atl doctor`](/tr/cli/doctor) ise bildirilen betiği eksik, dışarı taşan ya da çalıştırılabilir olmayan kurulu bir takım için **uyarır**. Brifinginiz hiç görünmüyorsa, betiği ayıklamadan önce doctor'ı çalıştırın.
:::

Bir tane yazarken bilmeye değer iki şey:

- **Çalıştırılabilir olarak dağıtın.** Kurulum, kaynak dosyanın kipini korur; dolayısıyla `+x` olmadan işlenen bir betik `+x` olmadan yansıtılır ve sonra exec'te başarısız olur — sessizce, her makinede. `atl skills check` bunu ATL tek deposundaki takımlar için yakalar; sizinki başka yerde yaşıyorsa işlemeden önce `chmod +x` yapın.
- **Bir git worktree içinde çalışmaz.** Bir bildirim *işlenmiş* bir dosyadır; aksi hâlde bir repo'nun herhangi bir worktree'sinde açılan her oturum betiğinizi çalıştırırdı — defalarca, betiğin okuduğu her şeyi tekrarlayarak ve içinde kimsenin oturmadığı bağlamlara yazdırarak. Oturum brifingi, bir insanın içinde oturduğu oturum içindir.

ATL bundan takımınız hakkında hiçbir şey öğrenmez. Betiği hangi takımın bildirdiğini, çıktının ne anlama geldiğini ya da betiğin bunu üretmek için ne okuduğunu — hangi hizmetle konuştuğu dâhil — bilmez. Nokta da budur: betik, önemsediği proje durumunu *kendisi* okur; dolayısıyla yeni bir tüketici — ya da bildirilecek yeni bir şey — CLI'da hiçbir değişiklik gerektirmez.

::: tip Bu özellik gelmeden önce mi kurmuştunuz?
`store` ve `channel` ile aynı hikâye: bildirim kurulum anında okunur, dolayısıyla `sessionScripts` alanından önceki bir kurulumda bu kayıt yoktur ve hiç bildirmeyen bir takım gibi davranır. `atl update` sabitlenmiş kaynağı yeniden çekerek bunu bir kez geri doldurur ve kendiliğinden çalışır.
:::

::: danger Bu, kurulu kodu otomatik çalıştırır
Bildirilen bir betik, makinenizde her oturum başlangıcında, sizin izinlerinizle ve sorulmadan çalıştırılır. Bu bilinçli bir karardır — oturum başına bir onay, mekanizmayı var oluş amacı için kullanılamaz kılardı — ve tam da bu yüzden *bir takımın nereden geldiği* önemlidir. Güvendiğiniz takımları kurun. `ATL_NO_SESSION_SCRIPT=1` geçişin tamamını kapatır.
:::

## Sürüm kısıtları {#version-constraints}

`dependencies` değerleri ve `requires.atl`, gelenek gereği standart SemVer aralık sözdizimiyle yazılır:

| Sözdizim | Anlamı |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — önerilen varsayılan) |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde) |
| `1.2.3` | Kesin sabitleme |
| `>=1.2.0` | Açık uçlu en düşük sürüm |

Caret (`^`) geleneksel öneridir — anlamca yama ve küçük sürüm güncellemelerini alır, geriye uyumsuz ana sürüm artırımlarını engeller. Ancak bugün CLI bu aralıkları değerlendirmez: `atl install` her bağımlılığı adına göre çözümler ve katalogdaki mevcut sürümü kurar, `requires.atl` de dayatılmaz. Yine de bunları bildirin — niyeti belgelerler ve aralık dayatması manifest değişikliği olmadan gelebilir.

## Dizin sözleşmeleri

`atl`, paketlediğin dosyaları `team.json` dosyasını okuyarak ve eşleşen yolları arayarak keşfeder:

```
my-team/
├── team.json
├── agents/
│   ├── web-agent.md             ← basit ajan (tek dosya)
│   └── db-agent/
│       ├── agent.md             ← karmaşık ajan (children deseni)
│       └── children/
│           ├── migrations.md
│           └── rls.md
├── skills/
│   └── create-new-project/
│       └── SKILL.md
└── rules/
    └── commit-style.md
```

Kurulabilir varlık dizinleri şunlardır: `agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/` ve `packs/` (`teampkg.AssetDirs` kümesi). `agents/`/`skills/`/`rules/` Claude Code'un doğrudan okuduğu dizinlerdir; `knowledge/`/`scripts/`/`packs/` ise takımın çalışma zamanı referans belgelerini, yardımcı betiklerini ve alan paketlerini taşır; `backends/` ise takımın arka uca özel bağdaştırıcı sözleşmelerini taşır — `backends[]` içinde adlandırdığı her arka uç için bir alt dizin. Diğer her şey (`team.json`, `README`, `LICENSE`) geride kalır.

Bir takımın bir varlık dizini altında en az bir dosya göndermesi gerekir, yoksa `atl install` başarısız olur (`team ships no installable assets`). Bildirilen tek tek `agents[]`/`skills[]`/`rules[]` girişleri katalog üst verisidir ve kurulum sırasında diske karşı doğrulanmaz — bildirilen `agents[]` ve `skills[]` girişlerini, birinci taraf takımlar için `atl skills check` geliştirici komutu çapraz kontrol eder.

## Doğrulama

v2'de ayrı bir JSON Şeması dosyası ve şema doğrulama CI adımı yoktur. Doğrulama minimumdur ve CLI'nin kendisinde yaşar:

- `team.json` geçerli JSON olarak ayrıştırılabilmelidir.
- `name` alanı bulunmalıdır.
- Takım, bir varlık dizini altında en az bir dosya göndermelidir — `atl install`, kurulabilir varlık olmayan bir takımda hata verir.

Sözleşmenin tamamı budur. `atl install` takımını kabul ederse geçerlidir; yerel ya da CI'da çalıştırılacak başka bir şey yoktur.

## Sıradaki

- **[Bir takım oluşturma](./creating-a-team)** — adım adım.
- **[`atl install`](/tr/cli/install)** — bir takımın nasıl çözümlendiği ve kurulduğu.
