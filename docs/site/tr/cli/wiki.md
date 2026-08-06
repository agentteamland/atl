# `atl wiki`

Bir projenin `.atl/wiki` dizini — **güncel doğruluk** bilgi katmanı — için bütünlük denetimleri. [`atl docs check`](/tr/cli/docs) ve [`atl skills check`](/tr/cli/skills) komutlarının kardeşi; docs sitesi ya da varlıklar yerine bilgi tabanı üzerinde çalışır.

`.atl/wiki` dizini olan her projede çalışır. Olmayan bir projede hiçbir şey yapmaz ve 0 ile çıkar.

## Kullanım

```bash
atl wiki check   # wiki genelinde erişilebilirliği ve bağlantı bütünlüğünü doğrula
```

## Neyi denetler

| denetim | neyi bildirir |
|---|---|
| `index-targets` | `CLAUDE.md` içinde `.atl/wiki`'ye giden, diskte karşılığı olmayan bir bağlantı |
| `reachability` | `CLAUDE.md`'nin hiç bağlantı vermediği bir wiki sayfası |
| `links` | Bir wiki sayfasının içinde, diskte karşılığı olmayan göreli bir bağlantı |

Üçü de **Fail** seviyesindedir. Uyarı katmanı yoktur: her bulgu bir dosya hakkında bir olgudur ve emin olamayan bir denetimin bu komutta yeri yoktur.

## Erişilebilirlik neden düzenlilik meselesi değil

`CLAUDE.md` indeksi, bir ajanın ürettiği arama sorgusunun kelime dağarcığını aldığı yerdir — ölçüm: indeks yüklüyken **%100 recall@5**, indeks görülmeden **%58**. Dolayısıyla hiçbir yerden bağlantı almayan bir sayfa yalnızca zor bulunur değildir: consult mekanizmasının bağlı olduğu ölçülen kelime dağarcığının **dışındadır**, yani onu bulmak için kurulmuş şey tarafından erişilemez.

## Bilerek denetlemediği şey

**Bir sayfanın hâlâ doğru olup olmadığı.**

Docs sitesi drift için denetlenebilir, çünkü karşılaştırılacak bir ground truth'u vardır — kod. Bir wiki sayfasının iddiasının (*"bu repo herkese açık"*, *"deponun uzak deposu yok"*) tek bir göndergesi yoktur; bu yüzden doğruluk burada mekanik olarak kararlaştırılabilir değildir.

Bu varsayılmadı, ölçüldü. Akla ilk gelen aday — *bir sayfanın atıf yaptığı depo yolu hâlâ var mı?* — gerçek bir korpus üzerinde **%76–90 yanlış pozitif** veriyor, ve nedeni ayarla düzelecek cinsten değil, yapısal: bir bilgi korpusunun türü ağırlıklı olarak **bir şeyin öldüğünü belgelemektir**, ve bu **ölü bir şeye atıf yapmak**la bayt bayt aynıdır. Denetim, düzeltme işini en iyi yapmış sayfalarda en çok ateşler.

Dolayısıyla bu komutun iki dürüst işi vardır: keşfedilebilirlik ve bağlantı bütünlüğü için bir **regresyon koruması**, ve yargı yarısı için **öncelikli bir hedef listesi** — o yarı burada değil, [`/observe`](/tr/skills/observe) becerisinin güncel-doğruluk lensinde yaşar. Bu ayrım CLI/Skill sınırıdır: CLI deterministiktir, beceri yargıdır.

## Çıkış kodları

- `0` — temiz, ya da bu projede `.atl/wiki` yok
- sıfır dışı — bir veya daha fazla bulgu, satır başına bir tane yazdırılır
