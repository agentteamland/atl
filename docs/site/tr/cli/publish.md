# `atl publish`

Global katmanının bir takım için biriktirdiği kazanımları geldikleri yere geri paylaş — kazanım dolaşımının 2→3. halkası.

## Kullanım

```bash
atl publish <handle>/<team>
```

`<handle>/<team>`, takım referansıdır (handle, takımın GitHub sahibidir). Takımın **global** katmanında kurulu olması gerekir — `publish`, bir projenin değil, global kazanımlarından çalışır.

Takımın global kopyanı, **yayımlanmış** sürümüyle (kaynak deponun taze bir çekimi) karşılaştırır. Farklılık gösteren — ya da senin eklediğin — her dosya bir *yayımlanabilir kazanımdır*. Varsayılan olarak `publish` yalnızca **planı gösterir**; `--apply` geçilene kadar hiçbir şey forklanmaz, commit'lenmez veya push'lanmaz.

## Ne zaman kullanılır

Bir takım, gerçek kullanım üzerinden iyileştirmeler kazandıktan sonra kullan — daha net bir yönerge, daha iyi bir desen, o takımı kuran herkesle paylaşmaya değer bir öğrenme. Global kopyanın yayımlanmış sürümün önüne geçtiğini fark ettiğinde `atl update` de seni dürter (`gains in X not yet upstream — run atl publish X …`).

`publish` **tasarım gereği bilinçlidir**: yazar sınırını aştığı için asla otomatik çalışmaz. Sen onu çağırırsın (dışarı paylaşma rızan); sahibi olmadığın bir takım içinse PR'ı sahibi kabul eder (onun rızası). Her iki durumda da kendi yerel ve global kazanımların, kabule asla bağımlı olmaz.

## Sahipliğe göre ne yapar

`publish`, takım deposunun sahibinin senin kimliği doğrulanmış GitHub login'inle eşleşip eşleşmediğini kontrol eder:

- **Sahibi sensin** → **yeniden-yayımlama**: depoyu klonlar, kazanımları takımın alt yolu altında hazırlar (stage), `team.json`'un sürümünü patch düzeyinde yükseltir, ardından commit + tag + push yapar ve index'in yeniden indekslemesi için deponun `atl-team` topic'ini taşıdığından emin olur.
- **Sahibi sen değilsin** → **upstream'e öner**: kaynak depoyu forklar, varsayılan dalından bir dal açar, kazanımları hazırlar, fork'una push'lar ve kaynak depoya karşı bir PR açar — sahibinin kabul ya da reddedebileceği, elinden gelenin en iyisi bir katkı.

CLI yalnızca mekanikleri yapar. Muhakeme (hangi sapmaların paylaşmaya değer olduğu — projeye ya da kullanıcıya özgü olanlar değil, genel iyileştirmeler) ve metin (PR gövdesi ya da commit mesajı), bu komutu süren ve yazılmış metni `--body-file` ile ona devreden `/publish` skill'inden gelir.

## Bayraklar

| Bayrak | Etki |
|---|---|
| `--apply` | Plana göre harekete geç (fork + push + PR açma, ya da commit + tag + push). O olmadan `publish` yalnızca planı yazdırır. |
| `--body-file <path>` | PR gövdesini (upstream'e öner) ya da commit mesajını (yeniden-yayımlama) tutan dosya. `/publish` skill'i tarafından yazılır. **`--apply` ile zorunludur.** |
| `--dry-run` | `--apply` ile birlikte, GitHub'a dokunmadan tam olarak ne olacağını yazdırır (fork/branch/stage/push, ya da commit/tag/push). `--body-file` olmadan da çalışır. |
| `--only <paths>` | Bunu yalnızca şu `.claude`-göreli yollarla sınırla — skill'in projeye/kullanıcıya özgü kazanımları eledikten sonra tuttuğu alt küme. Tekrarlanabilir / virgülle ayrılabilir. |

`--apply`, `--body-file` gerektirir; bir gövde yazmadan bir apply'ı önizlemek için `--dry-run` geç.

## Örnekler

Neyin yayımlanabilir olduğunu göster (yan etkisiz):

```bash
atl publish mesut/my-team
```

```
atl publish: 2 publishable gain(s) in mesut/my-team (vs published main):
  modified  agents/reviewer.md
  new       rules/commit-style.md

You own github.com/mesut/my-team — these would re-publish to it (commit + version bump + tag).
```

Sahibi olmadığın bir takım için bir apply'ı önizle, ardından `/publish` skill'i PR gövdesini yazdıktan sonra harekete geç:

```bash
atl publish acme/example-team --dry-run --apply
atl publish acme/example-team --apply --body-file pr-body.md
```

```
atl publish: opened https://github.com/acme/example-team/pull/42
```

## Notlar

- **Bütün-dosya kazanımları** (`promote` ve fan-out ile tutarlı olarak) — her dosya, parçaların elle birleştirilmesi olarak değil, global katmanında durduğu haliyle önerilir.
- Upstream'e öner, deterministik bir dal (`atl-publish/<handle>-<name>`) kullanır ve force-push yapar; böylece aynı takımı yeniden yayımlamak, eski bir dalla çakışmak yerine kendi dalını ve açık olan herhangi bir PR'ı günceller.
- Mekanikler, `PATH`'inde `gh`'ye (kimliği doğrulanmış) ihtiyaç duyar. Varsayılan plan görünümü duymaz.

## İlgili

- [`atl search`](/tr/cli/search) — `publish`'in taze tuttuğu aynı `atl-team`-topic'li index üzerinden takımları keşfet.
- [`atl update`](/tr/cli/update) — bir `publish`'i tetikleyen "henüz upstream'e gitmemiş kazanımlar" önerisini yüzeye çıkarır.

---

## `atl` kurulumu

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Ya da [GitHub Releases](https://github.com/agentteamland/atl/releases/latest) üzerinden önceden derlenmiş bir ikili edin.
