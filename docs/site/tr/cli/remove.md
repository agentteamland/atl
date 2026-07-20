# `atl remove`

Bir takımı kaldırır; yalnızca kurulumda yazılan dosyaları — geri alınabilir biçimde — kaldırır.

## Kullanım

```bash
atl remove <handle>/<team>            # proje kapsamından kaldır (varsayılan)
atl remove <handle>/<team> --global   # kullanıcı-global katmandan kaldır
```

`<handle>/<team>`, takımın referansıdır — GitHub sahibi artı takım adı; [`atl install`](/tr/cli/install)'a verdiğinle aynı biçim. Neyin hangi kapsamda kurulu olduğunu görmek için [`atl list`](/tr/cli/list) komutunu çalıştır.

## Örnek

```bash
$ atl remove acme/example-team
atl remove: removed acme/example-team (17 files) from project scope — reversible with `atl gc --undo`
```

Takım o kapsamda kurulu değilse:

```bash
$ atl remove acme/example-team
acme/example-team is not installed at project scope
```

## Ne olur?

1. Seçilen kapsamdaki takımın kurulum manifestosu `<layer>/.atl/installed/<handle>__<name>.json` konumundan okunur — `<layer>`, `--global` için `~/.atl`, proje kapsamı için `<proje>/.atl`'dir.
2. Manifestonun kaydettiği her dosya (kurulu varlık dizinleri altındakiler — `.claude/agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/`, `packs/`) kalıcı silinmez, `~/.atl/gc-trash` içine **yumuşak-silinir** — böylece manifestoya girmiş bir promote kazancı her zaman geri alınabilir.
3. Bu dosyaları barındıran dizinler en derinden başlayarak budanır — yalnızca artık boş olanlar. Başka bir takımın dosyalarını ya da kendi içeriğini barındıran bir dizine dokunulmaz.
4. Manifestonun kendisi kaldırılır.

Dosyalar yumuşak-silindiyse kaldırma geri alınabilir: [`atl gc --undo`](/tr/cli/gc) en son grubu geri yükler, [`atl gc --purge`](/tr/cli/gc) çöpü kalıcı temizler. Çıktı, kaç dosyanın kaldırıldığını ve hangi kapsamdan kaldırıldığını raporlar:

```
atl remove: removed <handle>/<name> (N files) from <scope> scope — reversible with `atl gc --undo`
```

Manifestonun dosyaları diskten zaten kaybolmuşsa `~/.atl/gc-trash` içine hiçbir şey taşınmaz — bu yüzden çıktı geri-alınabilirlik vaadini atlar ve dosyaların zaten yok olduğunu bildirir (yalnızca manifesto düşürülür):

```
atl remove: dropped <handle>/<name> manifest from <scope> scope — no files were soft-deleted (they were already absent)
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
