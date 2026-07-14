# `atl upgrade`

`atl` binary'sinin kendisini en son kararlı (stable) sürüme günceller: en yeni yayınlanan sürümü çözer ve çalışan derlemeden daha yeniyse indirir, checksum'ını doğrular ve bu binary'yi yerinde atomik olarak değiştirir.

`atl upgrade`, binary'yi güncel tutmanın **elle** yüzeyidir. Buna nadiren ihtiyaç duyarsın — [hook'lar](/tr/cli/setup-hooks) kurulduktan sonra `atl session-start` aynı kontrolü otomatik yapar ([Otomatik yükseltmeler](#otomatik-yükseltmeler)). Bu komuta yalnızca anında bir güncelleme zorlamak için başvurursun.

::: tip Binary mi, takımlar mı
`atl upgrade` **binary'yi** günceller. [`atl update`](/tr/cli/update) kurulu **takımlarını** günceller ve platform çekirdeğini `~/.claude`'a yansıtır. Bunlar ayrı yüzeylerdir: binary yayınlanmış bir artefakt, takımlar ise içeriktir.
:::

## Kullanım

```bash
atl upgrade
```

Argüman ve bayrak almaz.

## Ne yapar

1. **En son kararlı sürümü çöz.** GitHub'dan en yeni kararlı sürümü sorgular (ön-sürümler hariç tutulur).
2. **Karşılaştır — yalnızca yükselt, asla düşürme.** Çalışan derleme en son sürümden kesinlikle daha eski değilse işlem yapılmaz (`already up to date`). Bir `dev` derlemesi (damgalanmamış yerel `go build`) olduğu gibi bırakılır.
3. **İndir + doğrula.** İşletim sistemin/mimarin için sürüm asset'ini ve yayınlanmış `checksums.txt`'sini indirir, ve indirmenin SHA-256'sını kurulum dizinine dokunmadan **önce** doğrular.
4. **Çalışan binary'yi atomik değiştir.** Yeni binary'yi mevcut olanın yanına yazar ve yerine rename eder; böylece yarıda kesilen bir güncelleme asla yarı-yazılmış bir binary bırakmaz. Çalışan süreç eski kopyayı çalıştırmaya devam eder; bir sonraki çağrı yeni sürümdür.

## Otomatik yükseltmeler

Hook'lar kuruluyken `atl session-start` aynı kontrolü senin adına yapar — en fazla **24 saatte bir** olacak şekilde kısıtlanmış. Daha yeni bir sürüm varsa, indirme ve değiştirmeyi **arka planda** başlatır (oturumu asla bloklamaz) ve tek satırlık bir bildirim yazar; yeni binary bir **sonraki** oturumdan itibaren etkindir. Bu otomatiktir ve zorunludur — proje bazlı bir opt-in yoktur.

## Devre dışı bırakma

`ATL_NO_SELF_UPDATE` (herhangi bir değere) ayarlanınca self-update tamamen devre dışı kalır — hem elle komut hem de otomatik session-start kontrolü işlem yapmaz olur. Bu bir acil-frendir (örneğin bir sürüm sorun çıkarıyorsa); özellik aksi halde her zaman açıktır.

```bash
ATL_NO_SELF_UPDATE=1 atl upgrade   # → devre dışı, hiçbir şey yapmaz
```

## Platform notları

- **macOS / Linux** — tam destek (yerinde atomik değiştirme).
- **Windows** — çalışan bir `.exe` kendini üzerine yazamaz, bu yüzden `atl upgrade` daha yeni sürümü bildirir ve bunun yerine [kurulum betiğini](/tr/guide/install) yeniden çalıştırmanı ister; otomatik kontrol de değiştirmek yerine bildirir.
- **İzinler** — kurulum dizini yazılabilir değilse (örneğin `sudo` ile kurulmuş bir sistem yolu), yükseltme sessizce yetki yükseltmek yerine net bir hata bildirir. [Kurulum betiğiyle](/tr/guide/install) yeniden kur ya da `ATL_INSTALL_DIR` ile kullanıcı-yazılabilir bir dizine yönlendir.

## Ayrıca bkz.

- [`atl update`](/tr/cli/update) — kurulu **takımları** tazele (binary'yi değil).
- [`atl setup-hooks`](/tr/cli/setup-hooks) — otomatik kontrolü çalıştıran hook'ları bağla.
- [Kurulum](/tr/guide/install) — kurulum betiği (binary'yi yükseltmenin diğer yolu).
