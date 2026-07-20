# Uydurma yok

Makul görünen bir değer uyduran bir ATL ajanı — bir tool adı, bir work-item id'si, aslında hiç doğrulamadığı bir "yeşil" verdict'i — kendinden emin, yanlış çıktı üretir ve bunu downstream'de yakalamak pahalıdır. Bu çekirdek kural bunu yasaklar: **gerçekleri kaynaktan verbatim çöz; bir şeyi çözemiyor ya da doğrulayamıyorsan dur ve söyle — asla uydurma.** Bu sayfa, o disiplinin kullanıcı tarafıdır.

## Kaputun altında ne oluyor

[`no-fabrication` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/no-fabrication.md) her oturumda otomatik yüklenir (ve delivery-team'in başlattığı otonom `claude -p` worker'larına da, aynı global-kural yansıması yoluyla). Bir **çıktı-bütünlüğü (output-integrity)** alışkanlığını kurallaştırır: sahip olmadığın bir fact'i, identifier'ı ya da sonucu asla imal etme.

Corpus'un kendisinden damıtıldı — aynı kenar üç takımda ve çekirdek skill'lerde bağımsız yeniden türetilmişti: delivery `developer` ("bir tool adını, state literal'ini ya da path'i asla uydurmam"; "unverified asla pass değildir"), backend kontratı ("çalıştırılamayan bir yüzey → block, asla fake-green"), profile `curator` ("bir değeri asla fabricate etme"), `advisor` ("kanıtsız asla iddia edilmez") ve `/rule` ("asla varsayma — bilgi eksikse, sor").

## Pratikte ne demek

**Identifier'ları verbatim çöz.** Bir tool adı, bir API alanı, bir state literal'i, bir path, bir work-item id'si, bir sürüm — kaynaktan oku ve birebir üret; hafızadan makul bir tanesini yeniden kurmak yerine.

**Çözemiyorsan, boşluğu yüzeye çıkar.** Eksik bir analiz, henüz var olmayan bir id, ajanın bulamadığı bir değer — ilerlemek için boşluğu kendinden emin bir tahminle doldurmak yerine, söyler.

**Asla fake-green.** Timeout olan, atlanan ya da çalışamayan bir test/build/check **unverified**'dır — ve unverified asla "pass" olarak raporlanmaz. Ajan onu kanıtıyla birlikte blocked/unknown olarak raporlar.

## Neden çekirdek kural

ATL'nin **otonom teslimi**nin altındaki dürüstlük katmanıdır: bir merge state'ini uyduran ya da bir yeşil testi taklit eden gözetimsiz bir `claude -p` worker'ı, bozuk işi sessizce indirir — deterministik gate'lerin önlemek için var olduğu tam da o hata. [Karpathy ilkelerini](/tr/guide/karpathy-guidelines) **tamamlar**: onlar *girdi* tarafını yönetir (requirement'ı tahmin etme — sor); bu, *çıktı* tarafını (emit ettiğin artifact'i uydurma). Kural, gerçek olduğunu iddia ettiğin fact ve sonuçları hedefler — gerçekten üretken işi (prose taslağı, brainstorm, açıkça-etiketli tahminler) **kısıtlamaz**, ki orada uydurmak zaten işin kendisidir.
