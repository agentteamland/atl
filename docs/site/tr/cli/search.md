# `atl search`

Takım kataloğunda arama yapar — [`atl install`](/tr/cli/install) komutunun çözümlediği, GitHub tabanlı index.

## Kullanım

```bash
atl search [anahtar-sözcük]
```

`[anahtar-sözcük]`, takım handle'larına, adlarına, açıklamalarına ve anahtar sözcüklerine karşı eşleşir. Eşleşme büyük-küçük harf duyarsız ve alt-dize tabanlıdır; düzenli ifade (regex) değildir. Anahtar sözcük vermeden `atl search` çalıştırmak kataloğun tamamına göz attırır.

## Örnek

```bash
atl search web
```

```
1 team(s) matching "web":

  acme/example-team@1.2.0
    Agents, skills and rules for a typical full-stack web project.
    keywords: web, full-stack, example
    install: atl install acme/example-team
```

Her sonuç şunları gösterir:

- `<handle>/<ad>@<sürüm>` referansı (handle, takımın GitHub sahibidir — sahiplik, yazarlıktır),
- açıklama ve anahtar sözcükler,
- kopyalanabilir tam `atl install` komutu.

**`[verified]`** rozeti, AgentTeamLand bakımcılarınca incelenmiş takımları işaretler (`agentteamland/*` artı bir bakımcı izin listesi). Rozetin yokluğu yalnızca takımın kendi-yayımı olduğu anlamına gelir — güvensiz olduğu değil.

## Kataloğun tamamına göz at

Anahtar sözcüğü atlayarak kataloglanmış her takımı listele:

```bash
atl search
```

## Çevrimdışı davranış

`atl search` ağ için asla beklemez. Index'i çevrimdışı-öncelikli çözer: varsa `~/.atl/index.json` konumundaki ağdan tazelenen önbellek, yoksa ikiliye gömülü kopya. Önbellek bant dışında (`atl update` ile) tazelenir; böylece `search`, bir çekim için hiç beklemeden sonuçlar güncel kalır.

## Sonuç yok mu?

Katalog, [`atl-team`](https://github.com/topics/atl-team) topic'iyle etiketlenmiş herkese açık GitHub depolarından üretilir ve gençtir — alanın henüz kapsanmıyorsa, bu büyük olasılıkla yalnızca "henüz değil" demektir. Bir takımı listeletmek için deposunu `atl-team` ile etiketle (ya da takım deposundan `atl publish` çalıştır); katalog onu alır. Bkz. [Bir takım yazma](/tr/authoring/creating-a-team).

## İlgili

- [`atl install`](/tr/cli/install) — bulduğunu kur.
