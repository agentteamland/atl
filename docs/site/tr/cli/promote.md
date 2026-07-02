# `atl promote`

Projenin yerel kazanımlarını kullanıcı-global katmana yükselt — kazanım dolaşımının 1→2. halkası, [`atl update`](/tr/cli/update) fan-out'unun (dağıtımının) yukarı doğru aynası.

## Ne zaman kullanılır

Bunu neredeyse hiçbir zaman elle çalıştırmazsın. `atl promote`, arka planda [`atl tick`](/tr/cli/tick) içinde **kendiliğinden** çalışır; böylece bir ajanın tek bir projenin akışında biriktirdiği kazanımlar kendiliğinden global katmana yükselir ve *diğer* projelerin bir sonraki tick'inde onlara dağıtılır. Halka, sen düşünmeden kapanır.

Elle çalıştırılan komut, hemen şimdi yükseltmek istediğin durumlar için vardır — bir ajanın çokça büyüdüğü bir oturumun ardından ya da neyin yükseltileceğini doğrulamak için.

## Neyi yükseltir

Bir takım kurduğunda ATL, onun ajanlarını, becerilerini ve kurallarını projeye kopyalar. Sen çalıştıkça [öğrenme döngüsü](/tr/cli/learnings) bu varlıkları kurulum hatlarının ötesine büyütür — ezici çoğunlukla bir ajanın `children/` dizini ve yeniden kurulan bilgi tabanı. Fan-out, global→proje yönünde çeker; projenin hiç dokunmadığı dosyaları tazelerken dokunduklarını korur. **Promote bunun tersini yapar:** projenin *gerçekten* evrimleştirdiği dosyaları aynı takımın global kopyasına yükseltir.

Tasarımı gereği dar ve güvenlidir:

- **Yalnızca her iki kapsamda da kurulu takımlar.** Yükseltilecek bir global kopya bulunmalıdır — yalnızca projede kurulu olan bir takıma dokunulmaz (`atl promote` onu sessizce atlar).
- **Eklemeli.** Projenin kurulum hattının ötesine değiştirdiği dosyalar ile takımın kendi birimleri altındaki yepyeni dosyalar (örneğin yeni büyütülmüş bir `children/` girdisi) yukarı kopyalanır. Bir yükseltmenin ardından hem proje hem de global hatlar yükseltilen sürüme ilerler; böylece aynı kazanım asla iki kez yükseltilmez.
- **Çakışmaya karşı güvenli.** Global katman bir dosyayı *bağımsız olarak* da değiştirmişse, bu gerçek bir çakışmadır: proje değeri kazanır ve önceki global değer önce `~/.atl/history` altında arşivlenir (içerik-adresli, geri alınabilir).
- **Sabitleme-farkında.** [`atl pin`](/tr/cli/pin) ile sabitlediğin dosyalar yalnızca-projede tutulur ve asla yükseltilmez.

Başarılı bir geçiş, global nesil sayacını artırır; bu da diğer projelerine bir sonraki tick'lerinde kazanımları dağıtmalarını söyleyen şeydir.

## Kullanım

```bash
atl promote [handle/team]
```

Argümansız çalıştırıldığında promote, mevcut projedeki her takımda dolaşır ve her birinin uygun kazanımlarını yükseltir. Geçişi tek bir takımla sınırlamak için isteğe bağlı bir `handle/team` referansı ver (örneğin `acme/example-team`).

`atl promote` hiçbir flag almaz.

## Örnekler

Mevcut projeden uygun her takımın kazanımlarını yükselt:

```bash
atl promote
```

```
atl promote: lifted 3 file(s) to the global layer
```

Yükseltmeyi tek bir takımla sınırla:

```bash
atl promote acme/example-team
```

Global katmanın da değiştirdiği bir dosya yükseltildiğinde çakışma raporlanır ve önceki değer arşivlenir:

```
atl promote: lifted 2 file(s) to the global layer (1 conflict(s) — project won, prior global archived to ~/.atl/history)
```

Global katman zaten güncelse promote sessiz bir no-op'tur:

```
atl promote: nothing to lift — the global layer is already current
```

## İlgili

- [`atl tick`](/tr/cli/tick) — promote'u kendiliğinden çalıştıran oturum-içi ritim.
- [`atl update`](/tr/cli/update) — fan-out, aşağı doğru ayna (global→proje).
- [`atl pin`](/tr/cli/pin) — bir dosyayı yalnızca-projede tut, böylece promote onu asla yükseltmez.
