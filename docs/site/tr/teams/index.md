# Takımlar

Takımlar, ATL'nin dağıtım birimidir: tek komutla bir projeye (ya da global olarak) kurulan, versiyonlanmış bir ajan + beceri + kural paketi. Her takım [`team.json` sözleşmesini](/tr/authoring/team-json) izler ve takım kataloğu üzerinden keşfedilebilir.

> Katalog, [`atl-team`](https://github.com/topics/atl-team) topic'iyle etiketlenmiş herkese açık GitHub depolarından üretilir. Başvurulacak bir kayıt merkezi yoktur — depoyu etiketlemek (ya da [`atl publish`](/tr/cli/publish) çalıştırmak) takımı listeleyen şeydir.

## First-party takımlar: yeniden kuruluyor

v1 dönemi first-party takımlar (bir full-stack yazılım takımı ve bir design-system takımı) **Temmuz 2026'da emekliye ayrıldı** — v2 platformundan önce yazılmışlardı ve parça parça yamamak yerine v2 temeli üzerinde bilinçli bir yeniden kurulum tercih edilerek kaldırıldılar. Geçmişleri [atl deposunda](https://github.com/agentteamland/atl) korunuyor.

Yeniden kurulum **profile-team** ile başlıyor (danışmanlık tarzı takımların üzerine kurulacağı ortak kullanıcı-profili katmanı), ardından yeni bir yazılım geliştirici takımı geliyor. Takımlar yayımlandıkça bu sayfa yeniden katalog gezinme sayfası hâline gelecek.

## Göz at ve kur

```bash
atl search                      # kataloğa göz at
atl search <anahtar-kelime>     # ada, açıklamaya veya anahtar kelimeye göre takım bul
atl install <handle>/<takım>    # referansla kur
```

Takımlar `<handle>/<ad>` referansıyla kurulur. [`atl install`](/tr/cli/install) referansı GitHub-destekli kataloğa karşı çözer, kaynağı HTTPS üzerinden geçici bir tarball olarak indirir ve takımın `agents/`, `skills/` ve `rules/` içeriğini kapsamın `.claude/` dizinine kopyalar. Bir takım varsayılan olarak yayımcısının bildirdiği kapsamda kurulur (proje, global veya ikisi); `--global` ya da `--project` ile geçersiz kılabilirsin. İki katmanın nasıl etkileştiği için [kapsamlar](/tr/guide/concepts#scope-global-and-project) sayfasına bak.

`agentteamland/` handle'ı altında yayımlanan takımlar (ve bir bakımcı izin listesi) [`atl search`](/tr/cli/search) çıktısında **`[verified]`** rozeti taşır. Rozet, AgentTeamLand bakımcılarının incelediği takımları işaretler; kendin yayımladığın bir takımda olmaması takımın güvensiz olduğu anlamına gelmez.

## Kendi takımını yayımla

Herkes takım yayımlayabilir — [Takım oluşturma](/tr/authoring/creating-a-team) sayfasına bak. Geçerli bir `team.json` içeren ve `atl-team` topic'i taşıyan herkese açık bir depo bir saat içinde katalogda görünür.
