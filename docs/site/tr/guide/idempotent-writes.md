# Idempotent yazımlar

ATL ajanları ve seremonileri sürekli **yeniden çalışır** — bir restart'ta, bir retry'da, yarıda kalmış bir işe dönüşte. Bu çekirdek kural o yeniden-çalışmaları güvenli kılar: kalıcı bir depoya (durable store) yazma, işi çoğaltmak ya da daha yeni bir düzenlemeyi ezmek yerine aynı son-duruma **yakınsamalı (converge)**. Bu sayfa, o disiplinin kullanıcı tarafıdır.

## Kaputun altında ne oluyor

[`idempotent-writes` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/idempotent-writes.md) her oturumda otomatik yüklenir (ve her otonom `claude -p` oturumuna da, aynı global-kural yansıması yoluyla). Tek bir alışkanlığı kurallaştırır: **stabil bir anahtarla önce-kontrol-sonra-oluştur, mevcut-gerçeği yerinde ez, asla körlemesine yazma.**

Corpus'un kendisinden damıtıldı — aynı prensip bir iş-takip döngüsünde (adı konmuş bir `check-first-by-key` kontratı olarak), `/brainstorm`'da, `/create-code-diagram`'da ve `/profile-restore`'da bağımsız olarak yeniden türetilmişti. Bir disiplin bu kadar çok ilgisiz yerde tekrar ediyorsa, tek bir kurala aittir.

## Pratikte ne demek

**Stabil bir anahtarla önce-kontrol-sonra-oluştur.** Kalıcı bir öğe — bir work-item, bir sayfa, bir config — oluşturmadan önce ajan onu stabil girdilerden türetilmiş bir anahtarla (parent + ordinal, bir content hash) arar; per-run bir id ile değil. Bulundu → yeniden kullanıp günceller; bulunamadı → oluşturup anahtarı damgalar. Bir çakışma, hata vermek yerine var olan öğeye *çözümlenir*.

**Mevcut-gerçeği yerinde ez.** "Şu an ne doğru" tutan bir depo — bir wiki sayfası, üretilmiş bir diyagram, bir review raporu, bir config dosyası — yeniden-çalışmada eklenmez, değiştirilir. Aynı seremoniyi iki kez çalıştırmak iki değil, tek bir yakınsanmış sonuç bırakır.

**Daha yeni bir düzenlemeyi asla körlemesine ezme.** Bir üzerine-yazma, ajanın hedefi son okuyuşundan beri yazılmış veriyi yok edebilecekse, yazımı korur — timestamp/sürüm karşılaştırır ya da diff'leyip onaylar — çünkü daha yeni bir düzenlemeyi kaybetmek, duraklamaktan kötüdür. `/profile-restore`'un global hafızana dokunmadan önce yaptığı tam da budur.

## Neden çekirdek kural

Her **gözetimsiz yeniden-çalışma** buna bağlıdır: yarıda kalmış bir işe dönüş kalıcı duruma yakınsamalı, onu yeniden oluşturmamalı — yoksa bir restart, çoğaltılmış bir work-item, ikizlenmiş bir PR ya da kaybolmuş bir düzenleme demektir. Yakınsamayı varsayılan yapmak (özel bir kurtarma modu değil), gözetimsiz bir `claude -p` oturumunun güvenle retry/resume yapmasını sağlayan şeydir. Tek kasıtlı istisna **append-only** depolardır — her çalışmada yeni tarihli bir giriş eklemenin amaç olduğu bir journal ya da audit log, ki orada bu bir duplicate değildir.
