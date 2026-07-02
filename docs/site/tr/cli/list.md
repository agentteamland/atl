# `atl list`

Her kapsamda kurulu takımları gösterir.

## Kullanım

```bash
atl list
```

`atl list` herhangi bir bayrak veya argüman almaz. Her katmanın `.atl/installed/` dizinindeki kurulum manifestolarını okur; ağa gitmez.

## Çıktı

Takımlar [kapsama](/tr/guide/concepts#scope-global-and-project) göre gruplanır; her takım iki boşluk girintili `<handle>/<name>@<version>` biçiminde yazdırılır:

```
global:
  acme/example-team@1.2.0
project:
  acme/proto-team@0.3.0
```

Her iki kapsamda da kurulu bir takım her ikisinin altında görünür. `<handle>` takımın GitHub sahibidir; `<name>` ve `<version>` takımın `team.json` dosyasından gelir.

## Hiç takım kurulu değilse

Her iki kapsamda da kurulu takım yoksa:

```
atl list: no teams installed
```

## İlgili

- [`atl install`](/tr/cli/install) — takım kur.
- [`atl remove`](/tr/cli/remove) — takım kaldır.
- [`atl search`](/tr/cli/search) — kurulacak takım bul.
