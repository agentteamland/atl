# `/publish`

Bir takım için global-katman kazanımlarını upstream'e paylaş — sahip olmadığın bir takıma katkı öner ya da sahip olduğun bir takımı yeniden yayınla. [`atl publish`](/tr/cli/publish)'in LLM yarısı (kazanım dolaşımının 2→3 halkası).

`/publish`, CLI'nın yapamadığı muhakemeyi ve prose'u sağlar: *neyin* paylaşmaya değer olduğu ve PR gövdesi ya da commit mesajı. CLI planı hesaplar (global-katmandaki hangi dosyalar yayınlanmış sürümden farklı ve repo'nun sahibi sen misin) ve git/PR mekaniğini yürütür.

**publish tasarımı gereği bilinçlidir** — yazar sınırını aştığı için asla otomatik çalışmaz. Sen onu çağırırsın (dışarı paylaşma rızan); sahibi PR'ı kabul eder (onun rızası). Senin yerel ve global kazanımların, sahibinin bir şeyi kabul etmesine asla bağlı değildir.

## Ne zaman kullanılır

- `atl` bir throttled ağ kontrolü sırasında **"gains in X not yet upstream"** uyarısı verdiğinde.
- Bir takımın birikmiş kazanımlarını elle paylaşmak istediğin her an.

## Yordam

### 1. Planı al

```bash
atl publish <handle>/<team>
```

Yayınlanabilir kazanımları (her biri `modified` veya `new`) ve repo'nun sahibi olup olmadığını listeler. "nothing to publish" derse, bunu bildir ve dur.

### 2. Neyin paylaşmaya değer olduğuna karar ver

Her sapma upstream'e ait değildir. *Genel* olan kazanımları **tut** — daha net bir talimat, daha iyi bir desen, takımı kullanan herkese yardımcı olan yeniden kullanılabilir bir öğrenme. *Projeye* veya *kullanıcıya özgü* her şeyi **bırak** — o, paylaşılan takıma değil, proje kapsamına (global'i gölgeleyen) aittir. Bir kazanım sınırdaysa, dahil etmeden önce kullanıcıya sor.

### 3a. Sahip OLMADIĞIN takım → upstream'e öner

Mekaniği CLI yapar (fork + branch + tutulan kazanımları uygula + push + kaynak repo'ya PR aç). Sen **PR gövdesini** yazarsın:

- **Ne değişti ve NEDEN** — kazanım başına kısa bir bölüm, gerekçeyle başla.
- Gerçek kullanımdan gelen en-iyi-çaba bir katkı olarak çerçevele.
- Baskı yok: sahibi kabul eder ya da etmez; her durumda kazanımları yerel ve global olarak elinde tutarsın.

### 3b. Sahip OLDUĞUN takım → yeniden yayınla

CLI, tutulan kazanımları repo'na commit'ler, takım sürümünü yükseltir ve etiketler; konu-keşfi (index CI) oradan yeniden indeksler. Sen **commit mesajını** yazarsın (conventional, gerekçeyle).

### 4. Raporla

PR bağlantısını (upstream-öner) veya yeni etiketi (yeniden-yayınla) yüzeye çıkar, artı neyin paylaşıldığını ve neyin bilinçli olarak dışarıda bırakıldığını tek satırla özetle.

## Notlar

- **Bütün-dosya kazanımları** ([`atl promote`](/tr/cli/promote) ve fan-out ile tutarlı) — her dosyayı, parçaların elle birleştirmesi olarak değil, global katmanındaki haliyle önerirsin.
- **CLI / Skill sınırı** — deterministik uygula (fork/branch/push/PR-aç ya da commit/tag) CLI'nın işidir; muhakeme (adım 2) ve prose (PR gövdesi ya da commit mesajı) bu skill'in işidir. Skill, CLI'nın uygula ilkelini sürer ve yazılı gövdeyi ona verir.
- **Öneri, otomasyon değil** — atl, throttled bir ağ kontrolü sırasında "gains in X not yet upstream" uyarısını yüzeye çıkarabilir; bu, bunu çalıştırmanın işaretidir ama eylem senindir.

## İlgili

- [`atl publish`](/tr/cli/publish) — bu skill'in sürdüğü deterministik yarı.
- [`atl promote`](/tr/cli/promote) — publish'in üzerine kurulduğu 1→2 halkası (proje → global).
- [Skill'lere genel bakış](/tr/skills/drain)

## Kaynak

- Şartname: [core/skills/publish/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/publish/SKILL.md)
- CLI: [cli/cmd/atl/commands/publish.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/publish.go)
