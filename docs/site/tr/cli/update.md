# `atl update`

Kurulu takımları yenile: güncel bir katalog indeksi çek, yeni sürüm yayınlanmış her takımı yükselt, global katmandaki kazanımları proje kopyalarına yay ve platform çekirdeğinin son sürümünü global katmana yansıt — yerel olarak düzenlediğin dosyalara her zaman dokunmadan (çek, asla itme).

`atl update`, ağ yenilemesinin **elle** çalıştırılan yüzeyidir. Günlük iş [oturum içi kadans](#otomatik-güncellemeler-oturum-içi-kadans) aracılığıyla otomatik olarak yürütülür; bu komutu yalnızca bir geçişi zorlamak istediğinde ya da ikili dosyayı elle kurduğunda çalıştırmak gerekir.

## Kullanım

```bash
atl update
```

Hiçbir argüman ve bayrak almaz. Her zaman **mevcut proje** (çalıştırdığın dizin) ile **global** katman üzerinde çalışır.

## Ne yapar {#what-it-does}

`atl update` sırayla dört adım çalıştırır:

1. **İndeks önbelleğini yeniler.** Katalogdan (GitHub destekli takım indeksi) `~/.atl/index.json` konumuna en iyi çaba ağ çekimi yapar. Çevrimdışıysan çekim sessizce başarısız olur ve çözümleme önbelleğe alınmış ya da gömülü indekse geri döner — çevrimdışı olmak normaldir; başka hiçbir şey engellenmez.
2. **Takımları daha yeni yayınlanan sürümlere yükseltir.** **Proje** ve **global** katmanlarda kurulu her takım için `atl update`, takımı çözümlenen indekste arar. Yayınlanan sürüm kurulu olandan yeniyse takımın kaynağını tek kullanımlık bir HTTPS tar dosyası olarak yeniden çeker, açar ve kurulu kopyaya [yayma disiplini](#yayma-disiplini-düzenlemelerinin-nasıl-korunur) kapsamında yansıtır: değiştirilmemiş dosyalar yeni sürüme yenilenir, düzenlediğin dosyalar korunur, sürümde bulunan yeni dosyalar eklenir. Ardından kurulum bildirimi yeni sürümle yeniden yazılır. İndekste bulunmayan takımlar (örneğin yerel bir takım) olduğu gibi bırakılır.
3. **Global kazanımları projeye yayar.** Hem **global** hem **proje** katmanında kurulu her takım için proje-yerel dosyaların her biri üç yönlü karşılaştırmayla denetlenir (aşağıya bkz.). Değiştirilmemiş proje kopyaları global kopyadan yenilenir; yerel olarak düzenlediğin dosyalar korunur. Bir kazanımın global katmana yükseltildikten sonra projelerine ulaşması böyle sağlanır.
4. **Platform çekirdeğini global katmana yansıtır.** Çekirdek kurallar ve beceriler `atl` ikili dosyasının içinde paketlenir; bu adım onları `~/.claude` konumuna yenileyerek global katmanı ikili sürüm numaranla eşzamanlı tutar.

Yükseltme veya yayma işleminin ardından, global katmana kurulu olup global kopyasında henüz yukarı akışa gönderilmemiş kazanımı bulunan her takım için bir satırlık bir öneri de gösterir (bkz. [Yayınlama önerileri](#yayınlama-önerileri)).

### Çıktı

Özet satırı ne olduğunu yansıtır:

```text
atl update: upgraded 1 team(s), refreshed 14 file(s) from global
```

```text
atl update: upgraded 1 team(s)
```

```text
atl update: refreshed 14 file(s) from the global layer
```

Bekleyen bir şey yoksa:

```text
atl update: everything up to date
```

Bekleyen bir şey **yoksa ve** dizin yenilemesi ağa ulaşamadıysa (çevrimdışıysanız), ağdan doğrulanmış bir sonuç varmış gibi davranmak yerine bunu açıkça belirtir — çözümleme adımı önbellekteki/gömülü dizine geri düşer:

```text
atl update: up to date (offline — using cached index)
```

Çekirdek dosyalar değiştiyse özetten önce ayrı bir satır görünür:

```text
atl update: refreshed 3 core file(s)
```

## Yayma disiplini — düzenlemelerinin nasıl korunur {#yayma-disiplini-düzenlemelerinin-nasıl-korunur}

Hem sürüm yükseltmesi (2. adım) hem de global→proje yayması (3. adım), her dosyayı **kurulum anındaki** hash ile karşılaştıran ([kurulum bildirimi](#kurulum-bildirimi) içinde kayıtlı) aynı üç yönlü SHA-256 karşılaştırmasıyla değerlendirir:

| Karşılaştırma | Anlam | İşlem |
|---|---|---|
| yerel **=** yukarı akış | zaten güncel | yapılacak bir şey yok |
| yerel **=** kurulum referans hattı | hiç dokunmadın | yukarı akış/global sürüme **yenile** |
| yerel **≠** referans hattı | düzenledin | **koru** — kopyat saklanır |

"Değiştirilmiş" demek "kurduğumuzdan bu yana ayrışmış" demektir, salt "yukarı akıştan farklı" değil. Hiç değiştirmediğin dosya yenilenir; değiştirdiğin dosya hiçbir zaman sessizce üzerine yazılmaz. Bir kopya yenilendiğinde referans hattı yeni içeriğe ilerler; böylece bir sonraki geçiş temiz başlar.

Zorla üzerine yazma bayrağı yoktur. Yerel düzenlemeleri kasıtlı olarak atmak ve yayınlanan sürümü almak istersen takımı kaldırıp yeniden kurabilirsin:

```bash
atl remove <handle>/<team>
atl install <handle>/<team>
```

## Kurulum bildirimi {#kurulum-bildirimi}

Yaymanın karşılaştırdığı referans hattı takımın **kurulum bildirimi**nde yaşar — kapsam başına bir JSON dosyasında:

- `~/.atl/installed/<handle>__<name>.json` (global)
- `<project>/.atl/installed/<handle>__<name>.json` (proje)

Her bildirim `schemaVersion`, `handle`, `name`, `version`, `scope`, çekildiği `source` (`repo`, `subpath`, `ref`), `installedAt` ve kurulum anındaki SHA-256 değerleriyle her kurulu yolun eşlendiği `files` haritasını kaydeder. `atl update` bu haritayı "değiştirilmemiş" ile "düzenlenmiş" arasında ayrım yapmak için okur ve bir takımı değiştirdiğinde onu (sürümü, kaynak ref'ini ve referans hash'lerini ilerleterek) yeniden yazar.

## Otomatik güncellemeler — oturum içi kadans {#otomatik-güncellemeler-oturum-içi-kadans}

`atl update`'i elle nadiren çalıştırsın çünkü ATL her şeyi otomatik güncel tutar. [`atl setup-hooks`](/tr/cli/setup-hooks) ([`atl install`](/tr/cli/install)'ın zorunlu bir parçası olarak çalıştırılır) iki Claude Code hook'u bağlar:

- `SessionStart` → [`atl session-start`](/tr/cli/setup-hooks) — önceki oturumun öğrenmelerini boşaltır, doktor öz denetimini çalıştırır, platform çekirdeğini global katmana yansıtır ve (proje başına günde bir kez kısıtlanmış olarak) arka planda bir `atl update` başlatır; böylece daha yeni *yayınlanmış* takım sürümleri sen istemeden çekilir.
- `UserPromptSubmit` → [`atl tick --throttle=10m`](/tr/cli/tick) — ucuz bir istem başına **yayma** (global→proje) işlemi; kısıtlamalı bir boşaltma + doktor + yükseltme geçişi içerir.

İstem başına [`atl tick`](/tr/cli/tick), yerel yaymayı sürekli halleder; bu sayede global katmanına yükseltilen kazanımlar sen hiçbir şey yapmadan projelerine ulaşır. **Ağ** kısmı — indeksi yeniden çözmek ve daha yeni *yayınlanmış* takım sürümlerini çekmek — de otomatik çalışır: `atl session-start`, proje başına günde en çok bir kez ayrık (detached) bir `atl update` başlatır; böylece indirme arka planda yürür ve bir sonraki oturum daha yeni takımları görür. `atl update`'i elle çalıştırmak yalnızca o ağ geçişini, kısıtlamayı beklemeden şimdi zorlar.

### Otomatik takım güncellemesini devre dışı bırakma

Otomatik session-start takım güncellemesini kapatmak için `ATL_NO_TEAM_UPDATE` değişkenini (herhangi bir değere) ayarla; elle çalıştırılan `atl update` komutu yine çalışır. Bu, ikili için [`ATL_NO_SELF_UPDATE`](/tr/cli/upgrade) neyse takım varlıkları için onun karşılığıdır. Özellik aksi hâlde her zaman açıktır.

```bash
ATL_NO_TEAM_UPDATE=1   # session-start artık takımları otomatik güncellemez
```

## Yayınlama önerileri {#yayınlama-önerileri}

Bittikten sonra `atl update`, **global** olarak kurulu her takımda global kopyanda henüz yayınlanan sürüme geçmemiş kazanımları denetler ve takım başına bir uyarı yazdırır:

```text
atl update: gains in <handle>/<team> not yet upstream (3 file(s)) — run `atl publish <handle>/<team>` to contribute them
```

Bu yalnızca bir öneridir — hiçbir şey otomatik olarak yayınlanmaz. Yayınlamak açık, onay gerektiren bir eylem olarak kalır; bkz. [`atl publish`](/tr/cli/publish). Denetim en iyi çaba bazındadır ve takımın yayınlanan kaynağı çekilemezse sessiz kalır.

## Çevrimdışı davranış {#çevrimdışı-davranış}

`atl update` çevrimdışında sorunsuz degrade eder. İndeks yenilemesi ve tüm tar dosyası çekimleri sessizce başarısız olur, çözümleme önbelleğe alınmış/gömülü indekse geri döner ve yeniden çekimi yapılamayan takımlar yükseltilmez. Yerel global→proje yayması (3. adım) ağ bağlantısı gerektirmez ve yine de çalışır.

## İlgili

- [`atl install`](/tr/cli/install) — takımın ilk kurulumu
- [`atl tick`](/tr/cli/tick) — projeleri otomatik güncel tutan istem başına yayma + boşaltma geçişi
- [`atl setup-hooks`](/tr/cli/setup-hooks) — otomasyon hook'larını bağla
- [`atl promote`](/tr/cli/promote) — projenin kazanımlarını global katmana yükselt (yaymanın dağıttığı içeriğin kaynağı)
- [`atl publish`](/tr/cli/publish) — global kazanımları yukarı akışa gönder
- [`atl list`](/tr/cli/list) — neyin kurulu olduğunu ve hangi kapsamda olduğunu gör
