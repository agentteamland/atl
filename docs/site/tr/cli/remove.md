# `atl remove`

Bir takımı kaldırır; yalnızca kurulumda yazılan dosyaları siler.

## Kullanım

```bash
atl remove <handle>/<team>            # proje kapsamından kaldır (varsayılan)
atl remove <handle>/<team> --global   # kullanıcı-global katmandan kaldır
```

`<handle>/<team>`, takımın referansıdır — GitHub sahibi artı takım adı; [`atl install`](/tr/cli/install)'a verdiğinle aynı biçim. Neyin hangi kapsamda kurulu olduğunu görmek için [`atl list`](/tr/cli/list) komutunu çalıştır.

## Örnek

```bash
$ atl remove agentteamland/software-project-team
atl remove: removed agentteamland/software-project-team (17 files) from project scope
```

Takım o kapsamda kurulu değilse:

```bash
$ atl remove agentteamland/software-project-team
agentteamland/software-project-team is not installed at project scope
```

## Ne olur?

1. Seçilen kapsamdaki takımın kurulum manifestosu `<layer>/.atl/installed/<handle>__<name>.json` konumundan okunur — `<layer>`, `--global` için `~/.atl`, proje kapsamı için `<proje>/.atl`'dir.
2. Manifestonun kaydettiği her dosya (`.claude/agents/`, `.claude/skills/`, `.claude/rules/` altındakiler) silinir.
3. Bu dosyaları barındıran dizinler en derinden başlayarak budanır — yalnızca artık boş olanlar. Başka bir takımın dosyalarını ya da kendi içeriğini barındıran bir dizine dokunulmaz.
4. Manifestonun kendisi kaldırılır.

Çıktı, kaç dosyanın silindiğini ve hangi kapsamdan kaldırıldığını raporlar:

```
atl remove: removed <handle>/<name> (N files) from <scope> scope
```

::: tip Yalnızca manifesto kaydındaki dosyalar kaldırılır
`atl remove` tam olarak kurulum sırasında takımın kaydettiği dosyaları siler — bundan fazlası değil. `.claude/` altındaki diğer her şey yerinde kalır: kendiliğinden büyüyen agent `children/` ve `learnings/` dizinleri, kendi yazdığın beceriler, wiki sayfaları, journal kayıtları ve diğer içerikler. Bunlar manifesto kaydında olmadığı için kaldırma işleminden etkilenmez.
:::

## Kapsam

`atl remove` varsayılan olarak **proje kapsamında** çalışır — komutun çalıştırıldığı dizinin `.claude/` ve `.atl/` klasörleri. Kullanıcı-global katmandan (`~/.claude` varlıkları, `~/.atl` manifestosu) kaldırmak için `--global` bayrağını kullan.

Bir takım her iki kapsamda bağımsız olarak kurulu olabilir; birini kaldırmak diğerini etkilemez. Kaldırmadan önce takımın hangi kapsamda olduğunu görmek için [`atl list`](/tr/cli/list) komutunu çalıştır.

## Bayraklar

| Bayrak | Etkisi |
|---|---|
| `--global` | Proje yerine kullanıcı-global katmandan kaldırır. |

## İlgili

- [`atl list`](/tr/cli/list) — neyin hangi kapsamda kurulu olduğunu gör.
- [`atl install`](/tr/cli/install) — fikrini değiştirirsen yeniden kur.
- [`atl update`](/tr/cli/update) — kurulu takımları katalogdaki en son sürüme yenile.
