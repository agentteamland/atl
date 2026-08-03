# `atl decided`

Bir kararın kayıtlı olabileceği her yüzeyde arama yapar ve **hiçbir şey bulamadığında bunu dürüstçe söyler**.

```bash
atl decided "brief and stop"
```

## Cevapladığı soru

Bir gerekçe yazmadan önce — *davranış neden böyle, bilerek ne değil* — sorulmaya değer bir soru vardır: **kayıt, öne sürmek üzere olduğun kararı gerçekten taşıyor mu?**

Bu soru kolayca atlanır, çünkü yazılmış bir gerekçe otoriter okunur ve hiçbir yeri yanlış görünmez. Kimsenin vermediği bir karar iddiası, incelemeden gerçekten verilmiş bir karar kadar rahat geçer.

`atl decided` bu kontrolün mümkün olan en ucuz hali: tek komut, durum yok, indeks yok — karar yüzeylerinde doğrudan arama.

## Neyi arar

Genişten dara, ve yalnızca **var olan** kökleri:

| kök | orada ne yaşar |
|---|---|
| `.atl/docs` | yerleşmiş kararlar |
| `.atl/brain-storms` | onları üreten tartışmalar, reddedilen seçenekler dahil |
| `.atl/wiki` · `.atl/journal` | şu-an-doğru ve tarihçe |
| `.claude` | kurulu takım bilgisi |
| `docs` · `cli` · `core` · `teams` | site, ve kararları uygulayan kod |

`cli/` listede **bilerek** var. Bir karar bazen yalnızca onu uygulayan kodda kayıtlıdır — gerçek bir vakada mesele tek bir komutun yardım metnindeki tek satırla çözüldü, başka hiçbir yerde yazmıyordu.

Bu, her prompt'taki erişim hook'unun indekslediğinden daha geniştir ve bu kasıtlıdır: erişimin seçici olması gerekir, buranınsa tüm işi kapsayıcı olmaktır ve pahalı sonuç **yanlış negatiftir**.

Satıcı ve üretilmiş ağaçlar (`node_modules`, `vendor`, `.git`, `worktrees`) atlanır.

## Sıfır sonucu okumak

```
0 matches for "brief and stop"
searched: .atl/docs .atl/brain-storms .atl/wiki .claude docs cli core teams
```

Boş sonuç, bu komutun **var olma sebebidir** — bu yüzden hiçbir şey basmak yerine hangi kökleri aradığını yazar.

Ama uyarıyı harfiyen oku. **Bir metin araması sana hiçbir kararın olmadığını söyleyemez** — yalnızca bu kelimelerin bu dosyalarda olmadığını söyler. Aklına gelmeyen bir terim, kimsenin vermediği bir karar gibi görünür. Sonuca varmadan önce diğer ifadeleri dene.

Çıktı ayrıca yalnızca **gerçekten mevcut olan** kökleri adlandırır. Var olmayan bir dizin ne aranır ne de iddia edilir; böylece sıfır sonucu, sahip olmadığı bir kapsamayı ima etmez.

## İlgili

- [`atl learnings`](/tr/cli/learnings) — bu komutun aradığı bilgi döngüsünün yazma tarafı.
- [Kavramlar](/tr/guide/concepts) — karar yüzeylerinin birbiriyle ilişkisi.

Her prompt'ta sıralanan yüzey (`atl retrieve`) seçici karşılıktır: hiçbir şey döndürmeyebilir ve kapsayıcı değildir. Bu komut ise bilerek başvurduğun, sıfır sonucun cevap olduğu haldir.
