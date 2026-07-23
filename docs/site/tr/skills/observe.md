# `/observe`

**Proaktif gözlemci** — kendi teyakkuzunuz üzerinde bir kuvvet çarpanı. "Bu yalnızca kendi durumunu ele alıyor, diğerlerini değil" ya da "bu dosya büyüdükçe sessizce kırpılacak" gibi bir şeyi eninde sonunda fark eden *sizin* olmanız yerine, `/observe` bu tür boşluğu **önce** arar ve size sıralanmış, önceden doğrulanmış bir özet sunar.

İki boyutu vardır ve varsayılan olarak ikisini de çalıştırır:

- **Tetikleyici-gözcüsü** — ertelenmiş backlog'un `_Trigger:_` koşullarını gerçekte biriken şeye (kullanım, son işler, bilgi katmanı) karşı yürür ve **artık olgunlaşmış ertelenmiş öğeleri** yüzeye çıkarır — "hangisi hazır?" nöbeti, sizin üzerinizden alınmış.
- **Gizli-boşluk denetçisi** (yük taşıyan yarı) — **izlenmeyen** boşluklar için proaktif bir denetim: tasarım niyetine artık uymayan sevk edilmiş davranış, proje büyüdükçe sessizce kırılmak üzere olan şeyler, alınmış ama hiç sevk edilmemiş kararlar ve kendi global kurulumunuzdaki (`~/.atl/`) kayma.

[`/docs-audit`](/tr/skills/docs-audit) ile aynı disiplin: bulgular **grep'e dayalıdır** (birebir kaynak alıntısı olmadan iddia yok) ve **çelişki testinden geçirilir** (her aday, tutulmadan önce sınanır). Çok-ajanlı denetimler ~%40 oranında halüsinasyon üretir — bu koruma, özeti gürültü değil sinyal tutan şeydir.

## Ne zaman çalıştırılır

- `atl` oturum başında **"a proactive observer sweep is due — run /observe"** bildirdiğinde.
- Olgunlaşmış öğeleri ya da gizli boşlukları proaktif olarak kontrol etmek istediğiniz her an.

Yalnızca bir boyutu çalıştırmak için `--triggers-only` ya da `--gaps-only` ile kapsamı daraltın; varsayılan ikisidir.

## Ne yapar

1. **Yönel** — projenin `CLAUDE.md`'sini, son `.atl/journal/` kayıtlarını ve erteleme yüzeyini (`.atl/backlog.md` ya da yapılandırılmış bir teslimat panosu) oku.
2. **Olgun tetikleyiciler** — her ertelenmiş öğenin `_Trigger:_`'ini gerçek kanıta karşı değerlendir; bir öğe yalnızca onu ateşleyen sinyalin birebir alıntısı varsa olgundur.
3. **Gizli-boşluk taraması** — merceklere göre bulucuları dağıt (sevk-edilen-vs-tasarlanan, büyüme/ölçek, alınmış-ama-sevk-edilmemiş, kullanıcı kurulumu), her biri bulgusunu bir kaynak alıntısıyla temellendirir.
4. **Çelişkili doğrula** — her adayı *çürütmeye* çalış; yalnızca ayakta kalanı tut, ciddiyeti yeniden tart.
5. **Yüzeye çıkar** — doğrulanmış işaretlerin sıralanmış bir özeti, en eylemlenebilir olan başta, her biri kanıtı ve önerilen bir sonraki adımıyla. Boş bir tarama geçerli bir sonuçtur.
6. **Kaydet** — `atl observe --record` imleci damgalar, böylece sinyal ~1 gün yeniden ateşlenmez.

## Sınırlar

- **Önerir, kendiliğinden eylemez.** Bir gizli-boşluk bulgusu çoğu zaman bir karar gerektirir (bir brainstorm) — `/observe` onu yüzeye çıkarır ve seçmenize bırakır; sessizce PR açmaz ya da iş öğesi oluşturmaz.
- **Dürüst sınır.** Bir *sınıf* boşluğu güvenilir biçimde yakalar — sevk-edilen-vs-tasarlanan uyumsuzlukları, büyüme/ölçek riskleri, sevk edilmemiş kararlar, olgun tetikleyiciler — "önce siz yakaladınız" döngüsünü durduracak kadar. Her şeyi yakalama garantisi **değildir**; teyakkuzunuzun yerine geçen değil, onu çoğaltan bir araçtır.
- **Advisor sınırı.** Advisor'ın *kurulumunu ve çıktısını* dışarıdan denetler; asla onun içinde ya da onun *olarak* çalışmaz — o sohbet saf kalır.

## İlgili

- [`atl observe`](/tr/cli/observe) — deterministik yarı: "taramanın zamanı geldi mi?" sinyali ve bu skill'in damgaladığı imleç.
- [`/docs-audit`](/tr/skills/docs-audit) — docs sitesine yöneltilmiş aynı grep'e dayalı + çelişkili disiplin.
