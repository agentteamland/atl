# Karpathy ilkeleri

Platform tarafından her Claude Code oturumuna yüklenen davranış rehberleri. Yaygın LLM kodlama hatalarını azaltır — gizlenmiş varsayımlar, fazla karmaşıklaştırma, geçerken yapılan düzenlemeler ve muğlak yürütme.

2026-04-22 tarihinde AgentTeamLand `core@1.1.0` sürümüne platform genelinde bir kural olarak eklendi; böylece mevcut ve gelecekteki her takım onu kendiliğinden miras alır (takım başına yinelenen tanım yok). Kaynak: [forrestchang/andrej-karpathy-skills](https://github.com/forrestchang/andrej-karpathy-skills) — [Andrej Karpathy'nin LLM kodlama tuzakları üzerine gözlemlerinden](https://x.com/karpathy/status/2015883857489522876) türetildi. MIT lisanslı.

## Neden platform seviyesinde bir kural?

Kendi başına bırakılan LLM'ler şu eğilimleri gösterir:

- Sormak yerine **kafa karışıklığını gizlemek** ("birini seçeyim").
- **Fazla karmaşıklaştırmak** ("ileride lazım olur diye esnek bir soyutlama").
- İstenen değişikliği yaparken yan koda da dokunarak **geçerken düzenleme yapmak**.
- Net başarı kriteri olmadan **muğlak yürütmek**.

Bu dört desen gözlenebilir başarısızlık kalıplarıdır — havadan değil. Karpathy'nin ilkeleri bunları doğrudan hedefler. Onları her oturuma yüklemek, sürtünmenin yerini değiştirir: önemsiz görevlerde biraz hız maliyeti getirir ama önemsiz olmayanlarda gerçek zaman kazandırır (daha az yeniden yazım, daha az kapsam kayması, daha az "dur, neden o dosyayı da değiştirdin?" anı).

> **Ödünleşim:** Bu rehberler hız yerine dikkati öne çıkarır. Önemsiz görevlerde kendi yargını kullan.

## Dört ilke

### 1. Kodlamadan Önce Düşün

> **Varsayma. Kafa karışıklığını gizleme. Ödünleşimleri yüzeye çıkar.**

Uygulamaya geçmeden önce:

- Varsayımlarını açıkça belirt. Emin değilsen sor.
- Birden çok yorum mümkünse hepsini sun — sessizce birini seçme.
- Daha basit bir yaklaşım varsa söyle. Gerektiğinde geri it.
- Bir şey net değilse dur. Neyin kafa karıştırıcı olduğunu adlandır. Sor.

### 2. Önce Sadelik

> **Sorunu çözen en az kod. Speküle hiçbir şey yok.**

- İstenenin ötesinde özellik yok.
- Tek kullanımlık kod için soyutlama yok.
- İstenmemiş "esneklik" ya da "yapılandırılabilirlik" yok.
- İmkânsız senaryolar için hata yönetimi yok.
- 200 satır yazıyorsan ve 50 satırla olabiliyorsa, yeniden yaz.

Kendi kendine test: "Kıdemli bir mühendis buna fazla karmaşık der miydi?" Evet ise sadeleştir.

### 3. Cerrahi Değişiklikler

> **Yalnızca dokunman gerekene dokun. Sadece kendi pisliğini temizle.**

Mevcut kodu düzenlerken:

- Yan koda, yorumlara ya da biçimlendirmeye "iyileştirme" yapma.
- Bozuk olmayan şeyleri yeniden tasarlama.
- Sen farklı yapacak olsan bile mevcut biçeme uy.
- İlgisiz ölü kod fark edersen söyle — silme.

Değişikliklerin yetim parçalar bıraktığında:

- SENİN değişikliklerinin kullanılmaz hale getirdiği `import`, değişken ya da işlevleri kaldır.
- İstenmedikçe önceden var olan ölü kodu silme.

Sınama: değişen her satır doğrudan kullanıcının isteğine kadar izlenebilmeli.

### 4. Hedef Odaklı Yürütme

> **Başarı kriterini tanımla. Doğrulanana kadar döngüde kal.**

Görevleri doğrulanabilir hedeflere dönüştür:

| Muğlak | Hedef odaklı |
|---|---|
| "Doğrulama ekle" | "Geçersiz girdiler için test yaz, sonra geçmesini sağla" |
| "Hatayı düzelt" | "Hatayı yeniden üreten bir test yaz, sonra geçmesini sağla" |
| "X'i yeniden tasarla" | "Öncesinde ve sonrasında testlerin geçtiğinden emin ol" |

Çok adımlı görevlerde kısa bir plan belirt:

```
1. [Adım] → doğrulama: [denetim]
2. [Adım] → doğrulama: [denetim]
3. [Adım] → doğrulama: [denetim]
```

Güçlü başarı kriterleri ajanın bağımsız döngü kurmasına izin verir. Zayıf kriterler ("çalıştır işte") sürekli netleştirme gerektirir.

## Etkili olduğunun sinyalleri

Bu rehberlerin işe yaradığını şuradan anlarsın:

- **Farklar küçülür.** PR başına daha az gereksiz değişiklik.
- **Yeniden yazımlar kaybolur.** "Aslında, lütfen sadeleştir" türünden daha az gidip gelme.
- **Netleştirme soruları** önden gelir; yanlış uygulamadan sonra değil.

Atlandığını şuradan anlarsın:

- Ajan varsayım yapar ve sonradan özür diler.
- Ajan, tek çağrı yeri olan bir soyutlama yayımlar.
- Ajanın farkı ilgisiz biçimlendirme değişiklikleri içerir.
- Ajanın planı "yol giderken çözerim" şeklindedir.

## Projene nasıl ulaşır?

Kuralın kendisi [`core/rules/karpathy-guidelines.md`](https://github.com/agentteamland/core/blob/main/rules/karpathy-guidelines.md) dosyasında yaşar. [`atl install`](/tr/cli/install) komutunu (herhangi bir takım için) çalıştırmış her proje, `core` önbelleği üzerinden bu kuralı kendiliğinden alır; `atl update` ise [proje-yerel kopya yenileme modeli](/tr/cli/update#what-it-updates) üzerinden onu güncel tutar.

Kural her oturum başında Claude'un bağlamına yüklenir (bkz. [knowledge-system mimarisi](https://github.com/agentteamland/core/blob/main/rules/knowledge-system.md)) — istem başına ayrı bir çağrıya gerek yoktur.

## Daha derinine

- Üst kaynaktaki yan yana doğru-yanlış kod örnekleri (her ilke için): [EXAMPLES.md](https://github.com/forrestchang/andrej-karpathy-skills/blob/main/EXAMPLES.md)
- Karpathy'nin orijinal ileti dizisi: [@karpathy on X](https://x.com/karpathy/status/2015883857489522876)
- Kuralın kaynak doğruluğu (oturumlarına yüklenen sürüm): [core/rules/karpathy-guidelines.md](https://github.com/agentteamland/core/blob/main/rules/karpathy-guidelines.md)

## İlgili

- [Kavramlar: Kural](/tr/guide/concepts#rule) — kuralların ne olduğu ve nasıl yüklendiği.
