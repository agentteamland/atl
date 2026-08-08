# `atl list`

Her kapsamda kurulu takımları gösterir.

## Kullanım

```bash
atl list          # insan tarafından okunabilir, kapsama göre gruplanmış
atl list --json   # aynı küme, bir betik ya da beceri için JSON olarak
```

`atl list` argüman almaz. Her katmanın `.atl/installed/` dizinindeki kurulum manifestolarını okur; ağa gitmez.

## Çıktı

Takımlar [kapsama](/tr/guide/concepts#scope-global-and-project) göre gruplanır; her takım iki boşluk girintili `<handle>/<name>@<version>` biçiminde yazdırılır:

```
global:
  acme/example-team@1.2.0
project:
  acme/proto-team@0.3.0
```

Her iki kapsamda da kurulu bir takım her ikisinin altında görünür. `<handle>` takımın GitHub sahibidir; `<name>` ve `<version>` takımın `team.json` dosyasından gelir.

Bir **inceleme ajanı** beyan eden takım (`team.json` içinde `capabilities.review`) bunu sürümünün yanında gösterir:

```
project:
  acme/proto-team@0.3.0  (review: code-reviewer)
```

## `--json`

Aynı kümeyi bir dizi olarak yazdırır; [`/create-pr`](/tr/skills/create-pr) inceleme zinciri için hangi takımların alan uzmanı sunduğunu böyle keşfeder:

```json
[
  {
    "handle": "acme",
    "name": "proto-team",
    "version": "0.3.0",
    "scope": "project",
    "reviewer": "code-reviewer"
  }
]
```

Hiç beyan etmeyen takım için `reviewer` alanı yazılmaz — yaygın durum budur. Kurulu takım yoksa çıktı `null` değil `[]` olur.

Bu, bilerek **ham manifesto değildir**: bir manifesto ayrıca yüzlerce girdiye ulaşan dosya-başına sağlama haritası taşır ve *hangi takımlar kurulu, her biri ne beyan ediyor?* diye soran bir çağıranın onun içinden geçmesi gerekmemelidir.

::: tip v2.26.0 öncesinde kurulmuş bir takım
`reviewer` kurulum anında kaydedilir; alan var olmadan önce kurulmuş bir takım, [`atl update`](/tr/cli/update) manifestosunu tazeleyene kadar bunu göstermez.
:::

## Hiç takım kurulu değilse

Her iki kapsamda da kurulu takım yoksa:

```
atl list: no teams installed
```

## İlgili

- [`atl install`](/tr/cli/install) — takım kur.
- [`atl remove`](/tr/cli/remove) — takım kaldır.
- [`atl search`](/tr/cli/search) — kurulacak takım bul.
