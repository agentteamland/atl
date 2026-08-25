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
atl digest projects           # bu makinedeki her digest ve kime ait olduğu
```

Ve yazma tarafı — elle değil, bir süpürme tarafından kullanılır:

```bash
printf '<kanıt, neden önemli, önerilen sonraki adım>' \
  | atl digest add --sweep observe --title '<tek satırlık iddia>'
```

> ⚠ **Kendi sweep'inizin çıktısını mı okuyorsunuz? `--all` kullanın.** Oturum-başlangıcı sinyali **okunmamış sayısını** raporlar; bulguların düştüğünü doğrulamak için çıplak formu çalıştırmak onları okundu işaretler ve o sayıyı sıfırlar — sweep "başarıyla" biter ve bir sonraki oturuma hiçbir şeyin beklemediği söylenir. Hasar hata olarak değil, **sessizlik** olarak görünür. Kurtarma elle yapılır: her bulgunun id'sini, başlığını ve gövdesini alın, `drop` edin, sonra yeniden `add` edin — `add` okundu durumunu bilerek korur, yani tek başına yeniden eklemek okunmamışa döndürmez.

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

### Bölünme doğru. Sessizliği değildi.

Proje başına bir depo doğru şekildir ve onları birleştirmek, tam da önlediği hatayı geri getirirdi. Ama bölünme eskiden **görünmezdi**, ve bu ayrı bir şey.

Altına başka depolar klonlayan bir depo — bir bakım hub'ı, checkout'lardan oluşan bir monorepo — her birine kendi digest'ini verir. İçlerinden birinde koşan bir süpürme oraya yazar, üstteki ise normal cevap vermeye devam eder ve **fark edilecek bir yokluk oluşmaz**. Hiçbir şey mahsur kalmaz, hiçbir şey hata vermez; bulgulara sadece hiç ulaşılmaz.

Bir makinede ölçüldü (2026-08-25): **altı depo, 70 bulgu** — hub'daki bir oturum bunların 14'ünü görüyordu, platformun kendi becerileri hakkındaki dokuz bulgu ise `<hub>/repos/atl` içinde duruyordu: ulaşılabilir, ve hiç ulaşılmamış.

Artık `atl digest` diğerlerinin var olduğunu söylüyor:

```
atl digest: 5 other project digest(s) on this machine hold 56 finding(s), 50 unread.
            They are not shown here — a digest answers for its own project.
            `atl digest projects` lists them.
```

**Yalnızca başka bir depo varsa** yazar — her koşuda çıkan bir dipnot, sıradan tek-projeli makinede duvar kâğıdı olurdu; oturum sinyalinin yalnızca bir sayaç taşımasının sebebi de aynı.

### `atl digest projects`

Her depoyu, sayılarını ve ait olduğu projeyi listeler; `*` içinde bulunduğunuz projeyi işaretler.

Proje **dosyanın içine kaydedilir**, çünkü dosya adı bunu söyleyemez: `Path` kökü hash'ler ve hash tek yönlüdür. Bu alan olmadan hiçbir araç digest'leri listeleyip adlandıramaz — bir makinede altısını teşhis etmek 5.596 dizini hash'lemeye mal oldu, ve ikisi hiç teşhis edilemedi.

Kök kaydedilmeden önce yazılmış bir depo `(project not recorded)` olarak görünür. Bu bilinçli ve ters aramayla doldurulmaz: yokluk, ayrımın ne zaman tutulmaya başlandığına dair bir olgudur, ve tahmin etmek tam da alanın kazanmak için var olduğu güveni taşıyan bir yol uydururdu.

## İlgili

- [`atl observe`](/tr/cli/observe) — bunların çoğunu yazan süpürme.
- [`/observe`](/tr/skills/observe) — bulguları üreten LLM yarısı.
