# `atl learnings`

**Kalıcı öğrenme kuyruğunu** incele ve boşalt — kendi kendini süren öğrenme döngüsünün üstünde çalıştığı taban katman.

Konuşma sırasında yakalanan işaretçiler (Claude'un oturum ortasında düşürdüğü satır içi `<!-- learning ... -->` notları) kuyruğa **tam olarak bir kez** aktarılır; içerik özetine (hash) göre yinelenenler ayıklanır. [`/drain`](/tr/skills/drain) skill'i bekleyen her öğeyi bilgi tabanına (wiki / journal / ajan bilgi tabanı) katlar, ardından onaylar (ack) — böylece işlenen öğe silinir ve **bir daha asla yeniden raporlanamaz**. Bu işle-sonra-sil tasarımı, v1'in uzun-oturum yeniden-rapor hata sınıfını yapısal olarak ortadan kaldıran şeydir: raporlar kuyruktan gelir, sürekli büyüyen bir transkripti yeniden tarayarak değil.

Kuyruk, `~/.atl/queue.db` konumundaki tek bir gömülü [bbolt](https://github.com/etcd-io/bbolt) dosyasıdır — sunucu yok, daemon yok. Her projenin kuyruğu o tek dosyada yaşar; çalışma diziniyle anahtarlanan, proje başına ayrı kovalara (bucket) yalıtılmıştır. Bu alt komutların hepsi de **mevcut proje** üzerinde işler (komutları çalıştırdığın dizin).

## Ne zaman kullanılır?

Bunları elle pek nadiren çalıştırırsın — döngü onları kendiliğinden sürer. Şu durumlarda başvur:

- **`status`** — bilgi tabanına katlanmak üzere ne kadar öğenin beklediğine bir göz atmak için (bu, `SessionStart` hook'unun yüzeye çıkardığı sayının aynısıdır).
- **`peek`** — bekleyen öğeleri gerçekten görmek ya da makine-okunur listeyi bir betiğe vermek için. Bu, [`/drain`](/tr/skills/drain) skill'inin tükettiği belirlenimci okuma yüzeyidir.
- **`ack`** — döngünün normalde katlayacağı bir şeyi atlamak istiyorsan bir öğeyi elle işlenmiş olarak işaretlemek (silmek) için.
- **`transcript`** — konuşma akışını (yalnızca düz metin) yazdırmak için. Bu, bir drain'in madenleme adımının, ajanın işaretlemeyi unuttuğu öğrenmeleri geri kazanmak için kullandığı okuma yüzeyidir; `--channel` ile o kanalın kaldığı yerden devam eden ileri taramasına dönüşür.

## Kullanım

```bash
atl learnings status                 # kanal başına bekleyen sayılar
atl learnings status --json          # bekleyen sayılar JSON olarak (kanal→sayı)
atl learnings peek                   # bekleyen öğeleri listele (insan-okunur)
atl learnings peek --json            # tam makine-okunur liste
atl learnings peek --channel learning  # tek bir kanala filtrele
atl learnings recover                # silinmiş projelerde sıkışmış öğeleri listele (kuru çalışma)
atl learnings recover --apply        # onları bu projeye taşı, böylece bir drain ulaşabilsin
atl learnings ack <id>               # bir öğeyi işlenmiş işaretle (sil)
atl learnings transcript             # son konuşma akışı (düz bir okuma)
atl learnings transcript --json      # aynı akış, rol/metin kayıtları olarak
atl learnings transcript --channel learning   # ileri tara, o kanalın imlecini ilerlet
```

## Alt komutlar

### `atl learnings status`

Her kanal için bekleyen öğe sayısını yazdırır; doğrudan kuyruktan okur (yapısı gereği doğru, asla çıkarımla bulunmaz). Kanallar, çekirdeğin kendi `learning` kanalı ve kurulu bir takımın bildirdiği her kanaldır — profile-team'in `profile-fact`'i gönderilen örnektir ([yakalama kanalı bildirmek](/tr/authoring/team-json#declaring-a-capture-channel)). Kuyrukta hiçbir şey yokken şunu yazdırır:

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
| `--channel <name>` | string | *(tümü)* | Tek bir kanala filtreler (ör. `learning`). Kurulu hiçbir takımın bildirmediği bir kanal reddedilir ve hata mesajı gerçekten etkin olanları listeler — böylece bir yazım hatası sessizce hiçbir şeyle eşleşmek yerine gürültülü şekilde başarısız olur. |
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

Mevcut proje için **kullanıcı + asistan konuşma akışını** yazdırır — yalnızca düz metin; araç çağrıları ve sonuçları gürültü olarak ayıklanır. Bu, bir drain'in madenleme adımının üstünde çalıştığı okuma yüzeyidir: akışı, ajanın hiç işaretlemediği düzeltmeler, geri almalar ve kalıcı olgular için tarar, sonra her birini kuyruğa ekler (kuyruğun içerik özetiyle yinelemesi ayıklandığı için yeniden okumak her zaman güvenlidir).

**İki kipi** vardır ve aradaki fark, bir imleç tutup tutmadığıdır.

| Bayrak | Tip | Varsayılan | Ne yapar |
|---|---|---|---|
| `--channel <ad>` | string | *(yok)* | Bu yakalama kanalı için ileri tarar ve kanalın imlecini ilerletir. Etkin bir kanal olmalıdır. |
| `--limit <n>` | int | `2` | En son N transkripti okur. Yalnızca imleçsiz okuma için geçerlidir. |
| `--json` | bool | `false` | Turları `[rol] metin` satırları yerine JSON olarak (`role`, `text`) verir. |

**Çıplak hâli — düz bir okuma.** En son `--limit` transkriptten en son 256 KB düz metni verir ve hiçbir şey kaydetmez; dolayısıyla göz atmak için çalıştırmak bir drain'in henüz madenlenmemiş malzemesini asla tüketemez. Daha eski turlar kesildiğinde bunu bildiren bir not yazılır (stderr'e, böylece `--json` ayrıştırılabilir bir dizi olarak kalır).

**`--channel <ad>` — o kanalın taraması.** Kanalın en son madenlediği yerden devam eder, sonraki 256 KB düz metni **ileri** doğru verir, imleci ilerletir ve hâlâ bekleyeni raporlar:

```
atl: 12.4 MB of transcript still unmined for channel "learning" — sweep again to continue
```

Böylece ardışık taramalar bir oturumu baştan sona kapsar; kuyruğun ucunu tekrar tekrar okuyup iki çalıştırma arasında biriken her şeyi ikisine de okutmamak yerine. Bütçe çağrı başınadır ve bilinçli olarak bir madenleme alt-ajanının bağlamına göre boyutlanmıştır; bu yüzden büyük bir birikim tek seferde değil, birkaç çalıştırmada erir. `--limit` burada geçerli değildir: madenlenmemiş turu olan bir transkript, eski olduğu için asla atlanmaz.

İmleç **kanal başına** tutulur (`~/.atl/mine-cursor/` altında), çünkü madenlemenin birden çok tüketicisi vardır — [`/drain`](/tr/skills/drain) `learning` için, profile-team'in `/profile-drain`'i `profile-fact` için tarar ve ikisi aynı turda çalışabilir. Tek bir ortak konum, önce çalışanın diğerinin hâlâ ihtiyaç duyduğu pencereyi tüketmesine izin verirdi. Bir kanalın **ilk** taramasında devam edecek bir konum yoktur; bu yüzden son kuyruğu okur ve mevcut her transkripti o anki sonundan işaretler — imleci benimsemek, projenin şimdiye kadarki tüm oturumlarını yeniden oynatmaz.

Her iki kip de aşırı uzun olduğu için atlanan transkript kayıtlarını bildirir; böylece akışta eksik kalan turlar hiçbir zaman sessiz kalmaz.

İnsan-okunur çıktı, tur başına bir satırdır:

```
[user] hayır, oturum değil yenileme jetonu kullan
[assistant] Haklısın — yenileme jetonlarına geçiyorum.
```

### `atl learnings recover`

Proje dizini **artık var olmayan** kovalardaki bekleyen öğeleri, bir drain'in ulaşabileceği şekilde mevcut projenin kovasına taşır.

Kuyruk projeye göre bölümlenir ve her okuma yüzeyi proje-kapsamlıdır — yani silinmiş bir dizine anahtarlanmış kova her yerden görünmezdir, hiç kova olmamasından ayırt edilemez. `atl work dispatch` her birim için bir git worktree açıp tamamlanınca siler; anahtar depo köküne dönmeden önce otonom bir worker'ın işaretleri, sonradan yok olan bir yola kuyruklanıyordu. 2026-08-08 ölçümü: **6 kaybolmuş kovada 13 öğe**, yedisi üç hafta önce delivery worker'larının yakaladığı gerçek öğrenmeler. İçerikler baştan sona sağlamdı; kaybolan yalnızca adresleriydi.

Depo köküne anahtarlamak yeni kayıpları durdurur. Zaten sıkışmış olanı yüzeye çıkaramaz — yeniden anahtarlamadan sonra eski adresleri kimse aramaz — bu komut da bu yüzden onun yanında durur.

**Varsayılan kuru çalışmadır.** Taşımayı `--apply` yapar. İçerik, id ve özgün yakalama zamanı korunur; böylece drain bugün yakalanmış bir şey değil, üç haftalık bir kurtarma görür. Hedefte zaten bulunan bir id'ye dokunulmaz, yani iki kez kurtarmak asla kopyalamaz. Kaynak için tombstone yazılmaz: tombstone *işlendi* demektir, sıkışmış bir öğe ise hiç işlenmemiştir.

`atl doctor` aynı durumu `queue-stranded` olarak raporlar.

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
