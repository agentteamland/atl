# `/consult`

Projenin daha önce kaydettiğine bakar — sorguyu, kullanıcının yazdığı cümleden türetilmiş bir metin yerine, **ajanın kendisi**, içinde bulunduğu durumdan yazar.

## Hangi sorun için var?

ATL'nin zaten bir okuma yolu vardı: her turda kullanıcının prompt'unu gömen ve en iyi beş bilgi sayfasını enjekte eden bir `UserPromptSubmit` hook'u. Gerçek bir projede iki gün boyunca ölçüldüğünde bu kanalın neredeyse atıl olduğu görüldü — üstelik göründüğü sebepten değil.

**49 gerçek prompt** içinde yalnızca **2–3'ü** bilgi tabanının cevaplayabileceği bir soruydu. **%53'ü** düz talimattı ("birleştir", "sürüm kes"). Yani hook, zamanın yaklaşık %95'inde kimsenin sormadığı bir soruyu cevaplıyor ve sunduğu sayfaların **%88'i** hiç açılmıyor.

Başarısızlık ne sıralama kalitesinde ne de iyi sonuçları görmezden gelen bir okuyucuda. Soru basitçe hiç sorulmamış.

## Mekanizma

Tetikleyici, kullanıcının mesajının dilbilgisine değil, **ajanın** durumuna aittir. "Sürüm kes" cümlesi soru içermez ama kayıtlı bir kontrol listesi vardır; ona ihtiyacı olan ajandır, sormak zorunda olan da ajandır.

`/consult` bu yüzden eski tasarımın iki yarısını da tersine çevirir:

| | her-prompt hook'u | `/consult` |
|---|---|---|
| sorguyu kim yazar | harness, ham prompt'tan | ajan, durumdan |
| ne zaman çalışır | her tur | ajan kayda ihtiyaç duyduğuna karar verdiğinde |
| neyi optimize eder | kapsama | isabet |

İkisi de çalışmaya devam eder. Hangisinin kalacağı ölçümün sorusudur, bu sayfanın değil.

## Ölçüm

Commit edilmiş bir cevap anahtarına karşı (`.atl/eval/retrieval-answer-key.json` — 12 soru, iki dilde, gerçek sıralama yolu üzerinde):

| sorgu | recall@1 | recall@5 |
|---|---|---|
| ham prompt, İngilizce (hook'un yaptığı) | %50 | %83 |
| ham prompt, Türkçe | %42 | %75 |
| **modelin yazdığı sorgu** | **%58** | **%100** |
| modelin yazdığı sorgu, **`CLAUDE.md`'ye kör** | %25 | %58 |

Buradan iki sonuç çıkar; ikincisi önemli olandır.

**Dil farkı kaynağında kapanır.** Modelin yazdığı sorguyla İngilizce ve Türkçe *aynı* skoru alır, çünkü sorgu doğrudan korpusun dilinde doğar. Sonradan çevrilecek bir şey kalmaz.

**Kazanç modelin değil, index'in.** Aynı model, `CLAUDE.md` bulunmayan bir dizinde 12'de 7 alır — *hiçbir şey yapmamaktan kötü*. Kör hâlde hiçbir sayfada geçmeyen makul kelimeler uydurur (`failglob`, `nullopt`); bilgi haritası bağlamındayken korpusun kendi kelimelerini üretir (`nomatch`, `tombstone`, `gitlink`). Beceri bu yüzden ajana sorgu terimlerini her zaman yüklü olan bilgi haritasından almasını söyler; ve bu yüzden o haritayı küçültmek artık planlanacak bir temizlik değil, ölçülmesi gereken bir değişikliktir.

## CLI yarısı

```
atl retrieve --query "<terimler>"
```

Projenin index'inde açık bir sorguyla arar ve eşleşmeleri yazdırır. Sıralama yolunu hook ile paylaşır — aynı eşikler, aynı `topK` — böylece ikisi asla ayrışamaz ve karşılaştırılabilir kalır.

Hook'un aksine hata durumunda **sessiz değildir**. Fail-open (hata durumunda sessizce geçme), davetsiz ateşlenen bir şeyin özelliğidir; ajanın kendi çağırdığı bir araç ne bulduğunu bildirir — `No page … matched` dahil, ki bu bir hata değil gerçek bir cevaptır.

## Ölçülebilir kalması

Modelin çağırdığı bir mekanizma, modelin çağırmayı hatırlamasına bağlıdır — ve hiç danışılmamış bir tur, danışmaya ihtiyaç duymamış bir turla birebir aynı görünür. Ölçümün aynı değişiklikte, sonrasında değil, birlikte gelmesinin sebebi bu risktir:

- Her danışma, döndürdüğü sayfalarla birlikte fire log'una bir `consulted` sonucu yazar.
- Bir `Stop` hook'u (`atl retrieve turn-end`) tamamlanan her tur için tek satır kaydeder ve modelin bağlamına hiçbir şey yazmaz. Amacı danışmaya dürüst bir **payda** vermektir — "ajan kaç turda kayda baktı?" — sunulan-sayfa-başına-okuma yerine; ki o oran, sayfalar her promptta sunulduğunda anlamsızdır.

İkisi de `atl retrieve stats` çıktısında görünür:

```
turns           40
  consulted      9   22.5%   (agent wrote its own query)
    no match     2           (asked, corpus had nothing — a gap, not a miss)
```

`no match` ayrı okunmayı hak eder: ajan sordu ve korpus sessiz kaldı demektir — bakmama başarısızlığına değil, yazılmamış bir boşluğa işaret eder.

## Ne zaman ateşlenir?

Becerinin kendi yönergesi: bir şeyin nasıl çalıştığını iddia etmeden önce, bir öneri ya da seçim yapmadan önce, konvansiyonlarını doğrulamadığın bir dosyayı düzenlemeden önce, tanıdık gelen bir hatada, kayıtlı bir prosedürü olan bir işi adlandıran talimatta ve geçmişi son mesajından önemli olan bir işe geri dönerken.

Ve açıkça, bunların dışında **hayır**. Sabit bir kanal okunmaz olur — bu becerinin düzeltmek için var olduğu ölçülmüş başarısızlık tam olarak budur ve düzeltmenin kendisi için de geçerlidir.
