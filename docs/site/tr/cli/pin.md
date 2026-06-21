# `atl pin` / `atl unpin`

Proje-yerel bir özelleştirmeyi yalnızca o projeye ait tut. Bir **pin**, bu projenin `.claude` dizini altındaki bir yolu işaretler; böylece [`atl promote`](/tr/cli/promote) o yolu — ya da onun alt ağacını — global katmana asla yükseltmez.

Pin, *bildirimsel* bir devre-dışı bırakmadır; bir `.gitignore` girdisi gibi: yükseltme yine de otomatik olarak çalışır, pin yalnızca kapsamını daraltır. `atl pin` dışlamayı kaydeder; `atl unpin` onu temizler.

## Ne zaman kullanılır?

Varsayılan olarak, bir projede elde ettiğin bir kazanç (ayarlanmış bir agent, yeni bir skill, kendine özgü bir house-style kuralı) global katmanına yükseltilmeye uygundur; böylece diğer projelerine de dolaşır. İstediğin bunun *tersi* olduğunda — özelleştirme bilerek yalnızca bu projeye özgüyse ve başka yerlere sızmaması gerekiyorsa — bir yolu pin'le.

Bir pin, fan-out'a (dağıtım) **dokunmaz**. Fan-out zaten yerel olarak değiştirdiğin her dosyayı korur; dolayısıyla pin'lenmiş bir ayrışma, alıcı tarafta ne olursa olsun olduğu yerde kalır. Pin yalnızca global'e doğru olan *yukarı* yükseltmeyi durdurur.

## Kullanım

```bash
atl pin [path]        # bir pin ekler, ya da yol verilmediğinde pinleri listeler
atl unpin <path>      # bir pini kaldırır
```

`path`, bu projenin `.claude` dizinine göredir ve eğik çizgiyle ayrılır. Ya tek bir dosyayı ya da bir alt ağacı adlandırır — tamamını pin'lemek için bir agent/skill/rule birimine yönlendir:

```bash
atl pin agents/api-agent       # tüm api-agent alt ağacını pinler
atl pin rules/house-style.md   # tek bir rule dosyasını pinler
```

Yollar normalize edilir (baştaki `./`, çevreleyen eğik çizgiler ve `.` / `..` parçaları temizlenir); böylece `./rules/house-style.md/` ile `rules/house-style.md` aynı pini kaydeder. Bir alt ağaç pini, altında yuvalanmış her dosyayı kapsar — `agents/api-agent` pin'lendiğinde `agents/api-agent/agent.md`, onun `children/` dizini, `learnings/` dizini vb. hepsi muaf tutulur.

Pinler `<project>/.atl/pins.json` içinde yaşar — proje başına tek dosya, atomik biçimde yazılır ve liste sıralı tutulur. Dosyanın bulunmaması yalnızca "pin yok" demektir.

`atl pin` ve `atl unpin` **proje kapsamlıdır**: her zaman geçerli çalışma dizininin proje katmanı üzerinde işlem yapar.

## Örnekler

Mevcut pinleri listele (argümansız):

```bash
atl pin
```

```
atl pin — project-only paths (never promoted):
  agents/api-agent
  rules/house-style.md
```

Hiçbir şey pin'lenmemişken:

```
atl pin: no pins — every gain promotes to global
```

Projeye özgü bir kuralı pin'le, sonra yeniden dolaşmasına izin ver:

```bash
atl pin rules/house-style.md
```

```
atl pin: rules/house-style.md is now project-only (won't be promoted)
```

```bash
atl unpin rules/house-style.md
```

```
atl unpin: rules/house-style.md will be promoted again
```

Var olan bir pini yeniden pin'lemek ya da hiç pin'lenmemiş bir şeyin pinini kaldırmak işlemsiz (no-op) bir eylemdir ve bunu söyler (`… is already pinned` / `… was not pinned`); ikisi de hata değildir.

## Notlar

- **Bayrak yok.** Her iki komut da yalnızca yol argümanını alır: `atl pin` sıfır ya da bir, `atl unpin` tam olarak bir argüman kabul eder.
- Pin anahtarı, `.claude` dizinine göreli yoldur; mutlak ya da depo köküne göreli bir yol değil. Pini etkili kılan şey, `atl promote`'un başka türlü yükselteceği bir dosyaya çözümlenen bir yoldur.

## İlgili

- [`atl promote`](/tr/cli/promote) — proje kazançlarını global'e yükseltir; pinlere uyar.
- [`atl list`](/tr/cli/list) — bu projede neyin kurulu olduğu.
