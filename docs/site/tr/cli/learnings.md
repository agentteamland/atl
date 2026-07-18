# `atl learnings`

**Kalıcı öğrenme kuyruğunu** incele ve boşalt — kendi kendini süren öğrenme döngüsünün üstünde çalıştığı taban katman.

Konuşma sırasında yakalanan işaretçiler (Claude'un oturum ortasında düşürdüğü satır içi `<!-- learning ... -->` notları) kuyruğa **tam olarak bir kez** aktarılır; içerik özetine (hash) göre yinelenenler ayıklanır. [`/drain`](/tr/skills/drain) skill'i bekleyen her öğeyi bilgi tabanına (wiki / journal / ajan bilgi tabanı) katlar, ardından onaylar (ack) — böylece işlenen öğe silinir ve **bir daha asla yeniden raporlanamaz**. Bu işle-sonra-sil tasarımı, v1'in uzun-oturum yeniden-rapor hata sınıfını yapısal olarak ortadan kaldıran şeydir: raporlar kuyruktan gelir, sürekli büyüyen bir transkripti yeniden tarayarak değil.

Kuyruk, `~/.atl/queue.db` konumundaki tek bir gömülü [bbolt](https://github.com/etcd-io/bbolt) dosyasıdır — sunucu yok, daemon yok. Her projenin kuyruğu o tek dosyada yaşar; çalışma diziniyle anahtarlanan, proje başına ayrı kovalara (bucket) yalıtılmıştır. Bu alt komutların hepsi de **mevcut proje** üzerinde işler (komutları çalıştırdığın dizin).

## Ne zaman kullanılır?

Bunları elle pek nadiren çalıştırırsın — döngü onları kendiliğinden sürer. Şu durumlarda başvur:

- **`status`** — bilgi tabanına katlanmak üzere ne kadar öğenin beklediğine bir göz atmak için (bu, `SessionStart` hook'unun yüzeye çıkardığı sayının aynısıdır).
- **`peek`** — bekleyen öğeleri gerçekten görmek ya da makine-okunur listeyi bir betiğe vermek için. Bu, [`/drain`](/tr/skills/drain) skill'inin tükettiği belirlenimci okuma yüzeyidir.
- **`ack`** — döngünün normalde katlayacağı bir şeyi atlamak istiyorsan bir öğeyi elle işlenmiş olarak işaretlemek (silmek) için.
- **`transcript`** — son konuşma akışını (yalnızca düz metin) yazdırmak için. Bu, [`/drain`](/tr/skills/drain) skill'inin düzeltme-madenleme adımının, ajanın işaretlemeyi unuttuğu öğrenmeleri geri kazanmak için kullandığı okuma yüzeyidir.

## Kullanım

```bash
atl learnings status                 # kanal başına bekleyen sayılar
atl learnings status --json          # bekleyen sayılar JSON olarak (kanal→sayı)
atl learnings peek                   # bekleyen öğeleri listele (insan-okunur)
atl learnings peek --json            # tam makine-okunur liste
atl learnings peek --channel learning  # tek bir kanala filtrele
atl learnings ack <id>               # bir öğeyi işlenmiş işaretle (sil)
atl learnings transcript             # son konuşma akışı (/drain madenlemesi için)
atl learnings transcript --json      # aynı akış, rol/metin kayıtları olarak
```

## Alt komutlar

### `atl learnings status`

Her kanal için bekleyen öğe sayısını yazdırır; doğrudan kuyruktan okur (yapısı gereği doğru, asla çıkarımla bulunmaz). Kanallar `learning` ve `profile-fact`'tir. Kuyrukta hiçbir şey yokken şunu yazdırır:

```
learning queue: empty (nothing pending)
```

Aksi hâlde:

```
learning queue — pending by channel:
  learning       3
  profile-fact   1
```

`--json` ile aynı sayıları bunun yerine kararlı bir JSON nesnesi (`kanal→sayı`) olarak verir — kardeşleri `peek` ve `transcript`'in zaten sunduğu hafif, makine-okunur görünüm. Anahtarlar sıralıdır ve boş kuyruk `null` değil `{}`'dir; böylece çıktı betikler için kararlıdır.

| Bayrak | Tip | Varsayılan | Ne yapar |
|---|---|---|---|
| `--json` | bool | `false` | Bekleyen sayıları bir JSON nesnesi (`kanal→sayı`) olarak verir; boşken `{}`. |

```bash
$ atl learnings status --json
{"learning":3,"profile-fact":1}
```

### `atl learnings peek`

[`/drain`](/tr/skills/drain) skill'inin işleyip geçtiği bekleyen öğeleri listeler — `id`, `channel` ve yükün (payload) ilk satırı. Kuyrukta hiçbir şey yokken `no pending items` yazdırır.

| Bayrak | Tip | Varsayılan | Ne yapar |
|---|---|---|---|
| `--channel <name>` | string | *(tümü)* | Tek bir kanala filtreler (ör. `learning`). |
| `--json` | bool | `false` | Bekleyen listenin tamamını JSON olarak verir (id, channel, payload, enqueued_at) — `/drain` skill'inin üstünde çalıştığı biçim. |

İnsan-okunur çıktı, 12 karaktere kısaltılmış bir id'yi, kanalı ve yükün ilk satırını gösterir:

```
a1b2c3d4e5f6  learning      BSD sed requires escaped pipes for alternation …
```

### `atl learnings ack <id>`

İşlenmiş bir öğeyi kuyruktan siler — işle-sonra-sil olduğundan asla yeniden ortaya çıkamaz. Tam olarak bir id alır — tam id ya da onun belirsiz olmayan herhangi bir ön eki, `peek`'in yazdırdığı 12 karakterlik biçim dahil (git-short-SHA tarzında çözülür; bilinmeyen ya da belirsiz bir ön ek tahmin etmek yerine hata verir). Hiçbir bekleyen öğeyle eşleşmeyen bir id — bir yazım hatası ya da zaten onayladığın bir tanesi — sessizce başarılı olmak yerine bir hatayla reddedilir; böylece yanlış bir id, çalışıyormuş gibi yapmak yerine gürültülü biçimde başarısız olur. [`/drain`](/tr/skills/drain) skill'i her öğeyi tümleştirdikten sonra onu tam olarak bir kez onaylar (ack).

```
acked a1b2c3d4e5f6...
```

### `atl learnings transcript`

Mevcut proje için son **kullanıcı + asistan konuşma akışını** yazdırır — yalnızca düz metin; araç çağrıları ve sonuçları gürültü olarak ayıklanır. Bu, [`/drain`](/tr/skills/drain) skill'inin düzeltme-madenleme adımının üstünde çalıştığı okuma yüzeyidir: akışı, ajanın hiç işaretlemediği kullanıcı düzeltmeleri, geri almaları ve tekrarlanan hataları için tarar, sonra her birini bir öğrenme olarak kuyruğa ekler (kuyruğun içerik özetiyle yinelemesi ayıklanır; yani ilerletilecek bir imleci olmayan düz bir okumadır).

| Bayrak | Tip | Varsayılan | Ne yapar |
|---|---|---|---|
| `--limit <n>` | int | `2` | Bu proje için en son N transkripti okur. |
| `--json` | bool | `false` | Turları `[rol] metin` satırları yerine JSON olarak (`role`, `text`) verir. |

İnsan-okunur çıktı, tur başına bir satırdır:

```
[user] hayır, oturum değil yenileme jetonu kullan
[assistant] Haklısın — yenileme jetonlarına geçiyorum.
```

## Örnekler

**Neyin beklediğini denetle, sonra ona bak:**

```bash
atl learnings status
atl learnings peek
```

**Kuyruğu bir betikten sür** — JSON'u oku, her öğeyi tümleştir, onayla:

```bash
atl learnings peek --channel learning --json
# ... her öğeyi işle ...
atl learnings ack <id>
```

## İlgili

- [`/drain`](/tr/skills/drain) — `peek`'i okuyan, her öğeyi bilgi tabanına katlayan ve `ack`'leyen skill. Kuyruğun her günkü boşaltılma yolu; `learnings` alt komutları onun belirlenimci tesisatıdır.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — bekleyen sayıyı yüzeye çıkaran ve yakalanan işaretçileri kuyruğa aktaran `SessionStart` hook'unu kurar.
