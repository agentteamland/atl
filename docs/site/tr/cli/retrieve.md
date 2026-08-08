# `atl retrieve`

ATL bilgi döngüsünün **okuma** tarafı. Yazma tarafı [öğrenme yakalama](/tr/guide/learning-marker-lifecycle) ve [`/drain`](/tr/skills/drain); bu ise o bilgiyi geri çıkaran şey.

Projenin bilgi sayfalarını bir sorguya karşı sıralar — BM25 (sözcüksel) ile yerel bir anlamsal gömücü birleştirilir — ve en iyi eşleşmeleri yazdırır. Her yerde **hataya açık** (fail-open): herhangi bir hata hiçbir şey yazdırmaz ve bir istemi asla engellemez.

## Kullanım

```bash
atl retrieve --query "nomatch zsh glob"   # kendi yazdığın bir sorguyla ara
atl retrieve stats                        # kanalın burada gerçekte ne yaptığı
atl retrieve index                        # bu projenin dizinini (yeniden) kur
atl retrieve warm                         # gömme modelini indir, hattı ısıt
```

Alt komut olmadan çalıştırıldığında `UserPromptSubmit` hook gövdesidir: istemi stdin'den okur ve eşleşmeleri bağlam olarak sunar. Bu biçimi elle yazmazsın — [`atl setup-hooks`](/tr/cli/setup-hooks) onu bağlar.

## `--query` — danışma, ve neden hook'tan iyi

Hook, kullanıcının yazdığı cümleyi gömer. `--query` ise ajanın sorguyu **kendi durumundan** yazmasına izin verir. Bu fark ölçüldü, aynı on iki soru üzerinde:

| sorgu | recall@5 |
|---|---|
| kullanıcının ham istemi (hook'un yaptığı) | %83 İngilizce · %75 Türkçe |
| ajanın yazdığı, **bilgi haritası bağlamdayken** | **%100**, ve iki dil aynı skoru alıyor |
| ajanın yazdığı, **harita olmadan** | **%58** — hiçbir şey yapmamaktan kötü |

Kör kol kontrol grubudur ve talimatın kendisidir: `CLAUDE.md` haritası yüklüyken ajan korpusun kendi kelime dağarcığını üretir; haritasızken korpusun hiç kullanmadığı makul kelimeler uydurur. Bu yüzden sorgu haritadan, terimlerle ve İngilizce yazılır — *ne zaman* sorulacağına karar veren beceri için bkz. [`/consult`](/tr/skills/consult).

## `stats` — kanalın kendi sayıları

Retrieval kanalının ölçülebildiği tek yer. Danışma çalışmasındaki her rakam buradan geldi.

```
fires          440
  ranked       329   74.8%
    offered    319   72.5%   (1619 page refs)
    silent      10    2.3%
  suppressed   111   25.2%   (machine 104, short 7)
  translated    30    6.8%   (of the above — a query with no lexical hit)

turns           62
  consulted      8   12.9%   (agent wrote its own query)
    no match     1          (asked, corpus had nothing — a gap, not a miss)
```

İki satır dikkatle okunmayı hak ediyor:

- **`suppressed`** kayıp iş değil, sağlık işaretidir. İstemlerin dörtte biri makine üretimi metin ya da birkaç karakterdir — kimsenin sormadığı sorular. Susamayan bir kanal sinyal değildir.
- **`consulted`**, ajanın **bilerek** sorduğu turları sayar. Paydası turlardır; `Stop` hook'unun (`atl retrieve turn-end`) varlık sebebi tam da budur: tur işareti olmadan günlük neyin **sunulduğunu** kaydeder, sonrasında ne olduğunu değil.

`translated` satırının kendisi bir istem olarak sayılmaz — kendisinden sonra gelen fire üzerinde bir niteleyicidir ve sayılması yukarıdaki her yüzdenin bölündüğü paydayı şişirirdi. Kardeşi `translate-skipped` aynı gerekçeyle dışarıda tutulur ama kendine ait bir satırı yoktur: çalışmış, ancak cevabı bilerek kullanılmamış bir çeviriyi işaretler — metin zaten İngilizceydi ya da model, sorgu beklenen yerde düz metin döndürdü. `translated` satırı hiç görünmüyorsa burada hiçbir şey çevrilmemiş demektir — genellikle bir kimlik bilgisi tanımlı olmadığı için. Çevirmenin kendine ait bir kimlik bilgisine ihtiyacı vardır: ya `~/.atl/claude-token` ya da dışa aktarılmış bir ortam değişkeni — ve onu **nerede** dışa aktardığın, bir hook'un onu görüp göremeyeceğini belirler; bkz. [bilgi sistemi kılavuzu](/tr/guide/knowledge-system).

## `index` ve `warm`

`index` korpus dizinini yeniden kurar. Nadiren gerekir — korpus değiştiğinde session-start arka planda yeniden kurar, ve silinen bir sayfa artık tespit edilir (yalnızca-silme içeren bir değişiklik eskiden dizini var olmayan bir dosyayı sunar halde bırakıyordu).

**Soğuk** bir kurulum pahalıdır — büyük bir korpusta onlarca dakika — artımlı olan ise saniyeler sürer, çünkü parçalar `(yol, metin)` ile anahtarlanır. Dolayısıyla `touch` yeniden kurulumu zorlamaz. Tek-uçuş kilidi iki kurulumun yarışmasını engeller; kilit bir build'in ne kadar süreceğine dair tahminle değil, kalp atışıyla tutulur.

`warm` gömme modelini indirir ve hattı ısıtır; böylece ilk gerçek istem bu bedeli ödemez.

## Neyi dizinler

`.atl/{wiki,journal,docs}` ve `.claude/{agents,knowledge,skills,backends,packs}`, artı bir takımın **kaynağını** barındıran repo'da `teams/`.

Sonuncusu kurulu kopyalarla aynı şey değildir: kurulu bir kopya kaynağının bir sürüm gerisinde kalabilir ve kaynağı barındıran repo'da otorite kaynaktır. O repo'daki oturumların çoğu bir takımı düzenliyor ve bu eklenene kadar düzenledikleri şeyi getiremiyorlardı.

İlk girdiyi tam olarak oku: orada yazan `.atl/docs`, bir repo'nun kendi üst düzey `docs/` ağacı değil. O ağaç **hiçbir zaman** dizinlenmez — genellikle bu projenin bilgisi değil, üretilmiş ya da dışarıdan alınmış büyük bir sitedir ve onu dizinlemek asıl bilgi sayfalarını altında bırakır.

`.atl/brain-storms` **bilerek ve kalıcı olarak dışarıda**: brainstorm'ların çoğu reddedilen seçenekleri gereği kaydeder, ve kararından koparılmış bir parça birebir bir karar gibi okunur. Müzakereyi değil, hükmü dizinle.

## İlgili

- [`/consult`](/tr/skills/consult) — ne zaman sorulacağına ve sorgunun nasıl yazılacağına karar veren beceri
- [`atl setup-hooks`](/tr/cli/setup-hooks) — istem başına hook'u ve tur işaretini bağlar
- [`atl wiki`](/tr/cli/wiki) — bunun aradığı bilgi katmanı üzerindeki bütünlük denetimleri
