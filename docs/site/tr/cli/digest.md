# `atl digest`

**İnsan kararı** bekleyen süpürme bulguları.

## Neden var

`sweep-dispatch` çekirdek kuralı bir süpürmenin çıktısını, bulgunun karar gerektirip gerektirmediğine göre ayırır:

| bulgu | nereye gider |
|---|---|
| eyleme dönük ve kararı verilmiş — olgunlaşmış bir erteleme tetiği, deterministik bir kayma | zaten işi kapanışa taşıyan **tahta** |
| yargı gerektiren — gizli bir boşluk, önerilen bir kural, iki beceri arasındaki fazlalık | **buraya** |

Karar gerektiren bir bulgunun kararı, biletinden *önce* gerekir; bu yüzden ikinci tür asla otomatik kartlanmaz. Ama sadece söylenmesi de yeterli değildir: bir süpürme genellikle arka plan alt-ajanı olarak koşar, yani söyleyeceği canlı bir tur çoğu zaman yoktur — ve her koştuğunda konuşan bir mekanizma okuyucuya atlamayı öğretir.

Bu yüzden digest, kalıcı bir depo **artı** oturum sinyalindeki okunmadı sayacıdır. Hiçbir yarısı tek başına işe yaramaz: okunması zorunlu olmayan bir dosya, hiçbir şeyi gönderemeyen bir durumdur; içeriğini her oturumda tekrarlayan bir sinyal ise ayrımın önlemek için var olduğu gürültünün ta kendisidir.

## Kullanım

```bash
atl digest                    # bekleyeni yazdır ve okundu işaretle
atl digest --all              # okunmuşlar dahil hepsini yazdır; hiçbirini işaretleme
atl digest drop <id>          # kararı verilmiş bir bulguyu kaldır
```

Ve yazma tarafı — elle değil, bir süpürme tarafından kullanılır:

```bash
printf '<kanıt, neden önemli, önerilen sonraki adım>' \
  | atl digest add --sweep observe --title '<tek satırlık iddia>'
```

Gövde stdin'den okunur; böylece kanıt — dosya yolları, alıntılanan satırlar — kabuk tırnaklamasına takılmadan taşınabilir.

## Etkisizlik (idempotence)

Bir bulgu gövdesiyle değil, `(sweep, title)` ile anahtarlanır. Aynı bulgu tekrar bildirildiğinde:

- **gövdesi** en son ifadeyle tazelenir, ve
- **okundu durumu korunur.**

İkinci kısım günlük bir süpürmeyi katlanılır kılan şeydir. Bir süpürme taradığı yollar her hareket ettiğinde ateşlenir ve gizli bir boşluk, dün bildirildi diye gizli bir boşluk olmaktan çıkmaz — bu olmasa her süpürme aynı şey hakkında yeni bir kesinti olurdu.

Gövde yerine başlıkla anahtarlamak da aynı nedenle bilinçlidir: koşular arasında kanıtını farklı ifade eden bir süpürme, aynı bulguyu bildiriyordur.

## Okumak karar vermek değildir

Bir bulgu gösterildikten sonra digest'te kalır. Okuyucunun gördüğü ama henüz üzerine gitmediği bir bulgu hâlâ gerçektir; okununca kendini boşaltan bir depo tam da düşünülmesi gerekenleri düşürürdü.

Bir bulgu gerçekten sonuçlandığında — bir brainstorm açıldı, bir kart oluşturuldu ya da önemsiz olduğuna karar verildi — `atl digest drop <id>` kullanın. Sayacı dürüst tutan şey budur.

## Depolama

`~/.atl/digest/<proje-hash>.json`, proje başına bir dosya — bir süpürme `.atl/` dizini olan her projede ateşlenir, dolayısıyla tek bir ortak dosya, ilk açılan projenin diğerlerinin hepsi adına cevap vermesine yol açardı.

Bozuk bir digest boş okunur ve bir sonraki `add` ile yeniden yazılır: bir bulguyu kaybetmek telafi edilebilir — süpürme onu tekrar bildirir — kalıcı olarak başarısız bir okuma ise edilemez.

## İlgili

- [`atl observe`](/tr/cli/observe) — bunların çoğunu yazan süpürme.
- [`/observe`](/tr/skills/observe) — bulguları üreten LLM yarısı.
