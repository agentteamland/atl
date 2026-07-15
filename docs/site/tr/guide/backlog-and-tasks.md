# Backlog ve görevler

`.atl/` altındaki iki küçük dosya, bir projenin ertelenmiş kararlarını ve yakın vadeli niyetlerini tutar. Bunlar karar *durumudur* — [bilgi sistemi](./knowledge-system.md)nin journal'ı ve wiki'sinin bir kardeşi; üçüncü bir bilgi katmanı değil — ve [`/brainstorm`](../skills/brainstorm.md) becerisi tarafından yazılıp güncel tutulur.

## İki katman

| Dosya | Ne olduğu | Nasıl tutulduğu |
|---|---|---|
| **`.atl/backlog.md`** | Ertelenen, savuşturulan ya da belirsiz bırakılan her şeyin pasif, **tetik-kapılı üst kümesi**. Taranabilir bir dizin, bir yapılacaklar listesi değil. | Alana göre gruplu; bir öğe gönderildiğinde ya da terfi edildiğinde silinir |
| **`.atl/tasks.md`** | **Aktif-niyet alt kümesi** — sırada gerçekten yapmayı düşündüğün şeylerin kısa, önceliklendirilmiş listesi. | `Now` / `Next`; bir öğe gönderildiğinde silinir |

Bir öğe, onu öne çekmeye karar verdiğinde **backlog → tasks** yönünde taşınır — bir tetik ateşlendiğinde ya da yalnızca onu önceliklendirmeyi seçtiğinde. İlişkinin tamamı budur: backlog, şu an yapmamayı bilinçli olarak seçtiğin her şeydir; tasks ise yapmayı seçtiğin dilimdir.

## backlog.md

Backlog, kapsam ilerlediğinde hiçbir şeyin kaybolmaması için vardır. Bir brainstorm bir alt konuyu her ertelediğinde, bir şeyi "bu adımda değil" diye işaretlediğinde ya da bir soruyu açık bıraktığında, o öğe hemen buraya düşer — böylece aylar önce alınmış bir karar, zamanı geldiğinde hâlâ keşfedilebilir olur.

- **Tarihe göre değil, alana göre gruplu.** Başlıklar temalardır (`## Learning loop`, `## Distribution`, …); dosya, taradığın bir dizindir, bir kronoloji değil.
- **Öğe başına tek satır:** `- **Title** — one sentence. _Trigger:_ when it resurfaces. ↳ [source](...)`.
- **Tetik-kapılı.** Çoğu öğe bir `_Trigger:_` taşır — geri döneceği koşul. Backlog bir yapılacaklar listesi değildir; neyi ertelediğinin ve neden geri döneceğinin belleğidir.
- **Ayrıntı kaynakta yaşar.** Zengin "neden ertelendi / tam bağlam" açıklaması, bağlantılı brainstorm'da kalır — backlog dizin, brainstorm ise kayıttır. Onu tekrarlama.

## tasks.md

Tasks, aktif niyetin dürüst kısa listesidir.

- **Biçim:** `- [ ] **Title** — one sentence. ↳ [source](...)`, `## Now` / `## Next` altında gruplu.
- **Kısa ve dürüst.** Aktif olarak planlanmış bir şey yoksa, `tasks.md` neredeyse boştur — bu doğru durumdur, doldurulacak bir boşluk değil. Görev uydurma; planlanmamış ertelenmiş iş `backlog.md`'ye aittir.
- **Gönderildiğinde silinir.** Biten bir görev kaldırılır (`docs/` ve CLAUDE.md doğrunun kaynağı olur), asla işaretlenmiş bir öğe olarak geride bırakılmaz.

## Yaşam döngüsü

```
brainstorm defers  →  backlog.md  →(pull forward)→  tasks.md  →(ship)→  deleted
                          └──────────────(ship directly)──────────────→  deleted
```

- **Backlog'a giriş:** [`/brainstorm done`](../skills/brainstorm.md), kapatılan brainstorm'u tarar ve ertelenen ya da belirsiz her öğeyi doğru alan grubunun altına yerleştirir. Bunu atlamak, ertelenmiş kapsamın sessizce yok olma yoludur.
- **Backlog → tasks:** bir öğe üzerinde harekete geçmeye karar verdiğinde onu terfi ettir.
- **Çıkış:** bir öğe, gönderildiği anda her iki dosyadan da çıkar — dokümanlar doğru hâline gelir ve hiçbir şey "yapıldı" olarak öylece kalmaz.

## İskele

[`atl init`](../cli/init.md) (ve `atl install`), `.atl/` altına boş `backlog.md` ve `tasks.md` iskeletlerini yalnızca henüz yoklarsa bırakır — kendi dosyaların asla üzerine yazılmaz. Global katmanda proje `.atl/`'si olmadığından atlanır.
