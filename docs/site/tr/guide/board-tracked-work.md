# Board'a-işlenen iş

Projende bir teslimat **board backend'i** varsa — delivery-team ile bağlanmış bir Azure ya da GitHub project'i — ATL board'u dürüst tutar: **board'a bir öğe düşmeden hiçbir shippable iş yapılmaz.** Bu sayfa, o disiplinin kullanıcı tarafıdır.

## Kaputun altında ne oluyor

[`board-tracked-work` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/board-tracked-work.md) her oturumda otomatik yüklenir (ve delivery-team'in başlattığı otonom `claude -p` worker'larına da). **Koşulludur**: yalnızca projenin kökünde bir `.delivery/config.json` varsa aktif olur. `atl session-start` o dosyayı tespit eder ve tek satırlık bir hatırlatma basar — *"this project is board-backed (github) — record every shippable unit on the board…"* — böylece ajan kuralın burada aktif olduğunu bilir. Board backend'i olmayan bir projede kural uykudadır, hiçbir şey eklemez.

Belirli bir açığı kapatmak için vardır. Board-backed bir projede durum okunabilir olsun diye bir tracker vardır — ne planlı, ne uçuşta, ne shipping olmuş. Ama bu, ancak iş gerçekten board'a düşerse geçerlidir. Hızlı bir oturumda bir yığın gerçek işi — fix'ler, release'ler, sweep'ler — yapıp ship edip hiçbirini kaydetmemek kolaydır; geriye kod epey ilerlemişken hiç ilerleme göstermeyen bir board kalır. O zaman tracker yalan söyler. Bu kural, board'un gerçeği yansıtmasını sağlar.

## Pratikte ne demek

**Bir shippable birim başlamadan önce bir board öğesi.** "Shippable birim", tutarlı bir teslimattır — commit/PR olan şey (bir fix, bir feature, bir doküman sweep'i, bir release). Tek bir board öğesi alır; yoksa iş başlamadan önce oluşturulur. Granülarite teslimattır, **her alt-edit değil** — bir birimin içindeki adımlar onun öğesini paylaşır.

**Görev sırasında çıkan yeni iş kendi öğesini alır.** Bir görev daha fazla iş doğurduğunda — biri düzeltilirken bulunan bir bug, gereken bir follow-up — o da bir board öğesi alır; orijinalin altında izlenmeyen scope ship etmek yerine.

**Board durumu yansıtır.** Öğeler başlarken *In Progress*'e, ship'te *Done*'a taşınır. (GitHub Projects v2'de başlangıç → *In Progress* adımı manueldir — platform yalnızca Done ucunu otomatikleştirir.)


**Resume'de önce board'a bak.** Bir oturum resume olduğunda (moladan sonra bir "devam et"), önce board'da *In Progress* olan bir şey var mı diye bakar ve yarım kalan işin döngüsünü kapatır — görevi yarıda biten bir oturum öğesini In Progress bırakır, bir sonraki onu kaybetmemeli. Bu, aynı disiplinin okuma tarafı (`atl session-start` hatırlatmayı basar). Board sana *ne uçuşta* olduğunu söyler; *sıradaki ne* olduğunu kendi başına dikte etmez — bir backlog çoğunlukla ertelenmiş tasarım işi olabilir, o yüzden yapılacak sıradaki şey için yine projenin kendi resume geleneğine bakarsın, en üst karta değil. Board-aware, board-driven değil.
## Neden çekirdek kural

Board-backed bir projenin tracker'ını güvenilir kılan şey budur — hem bir insan reviewer için hem de ATL'nin kendi **otonom teslimi** için, ki orada board tüm döngünün okuduğu tek doğru kaynaktır. Delivery-team'in "board = doğru" disiplinini resmi seremonilerden (`/kickoff`, `/refine`, `/sprint-plan` — zaten öğe oluştururlar) projedeki **tüm** işe genişletir — interaktif düzenlemeler, bakım, ad-hoc çalıştırmalar dahil — böylece hiçbir şey board'u atlamaz. Aşırı-düzeltmeye karşı tek koruma: **keystroke değil, teslimat izle** — her micro-edit'e bir board öğesi, board'u boş bir board kadar işe yaramaz kılar.
