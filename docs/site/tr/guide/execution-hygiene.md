# İcra hijyeni

ATL ajanları küçük bir **icra varsayılanları** kümesini paylaşır — çok adımlı mühendislik işini *nasıl* temiz yapacaklarına dair alışkanlıklar. Bunlar eskiden yalnızca tek bir bakımcının kişisel kurulumunda yaşıyordu; artık bir çekirdek kural, dolayısıyla her takım ve her otonom teslim worker'ı aynı disiplinli biçimde çalışıyor. Bu sayfa, o duruşun kullanıcı tarafıdır.

## Kaputun altında ne oluyor

[`execution-hygiene` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/execution-hygiene.md) her oturumda otomatik yüklenir (ve delivery-team'in başlattığı otonom `claude -p` worker'larına da, aynı global-kural yansıması yoluyla). Bir ajanın işini incelenebilir ve güvenilir kılan üç alışkanlığı kurallaştırır — "bir diff üretti" ile "kıdemli bir mühendisin onaylayacağı bir diff üretti" arasındaki fark.

Yanındaki iki kuralı tekrarlamak yerine **tamamlar**: [Karpathy ilkeleri](/tr/guide/karpathy-guidelines) kodlamadan önce *düşünmeyi* yönetir; branch hijyeni ise *branch yaşam döngüsünü*. İcra hijyeni ise aradaki anlık-mekaniği ele alır.

## Üç alışkanlık

**1. Subagent hijyeni.** Geniş bir arama ya da çok dosyalı bir inceleme için ajan bir *subagent* (alt-ajan) başlatır — her birine tek odaklı görev — ve kendi context'ini yalın tutar. Subagent **dosya dökümlerini değil, sonucu** döndürür: okuduğu her şeyin dökümünü değil, bulguyu alırsın. Bu, asıl akıl yürütmeyi tutarlı tutar (şişmiş bir context kendi başına bir karmaşıklık türüdür).

**2. Otonom bug-fix.** Ajana reproduce edilebilir bir bug, bir hata veya patlayan (failing) bir test ver; **kendisi araştırıp düzeltir** — log'u, stack trace'i, testi okur, hipotez kurar, düzeltir, doğrular — sana ne yapacağını sormak için geri sekmez. Bir bug raporu bir hedeftir, onay talebi değil. Yüzeye çıkaracağı tek şey gerçek belirsizliktir — o zaman bile, bir "ne yapayım?" değil, bir teşhis olarak.

**3. Atomic commit'ler.** Her commit tek tutarlı, doğrulanmış bir iş birimidir — o birim çalışır çalışmaz commit'lenir, ilgisiz değişikliklerle batch'lenmez. Değişen her satır tek bir niyete izlenir, böylece bir inceleme (ve gelecekteki bir `git bisect`) okunur kalır. Bu *commit granülaritesidir*; o commit'lerin nerede yaşadığı branch hijyeninin işidir.

## Neden çekirdek kural

Bunlar ATL'nin **otonom teslimini** güvenilir kılan disiplinlerdir: bir `claude -p` worker'ı bir backlog item'ını gözetimsiz inşa ederken atomik commit'ler, kendi test hatalarını düzeltir ve temiz delege eder — çünkü kural yalnızca interaktif bir oturuma değil, ona da ulaşır. Bunları tek bir kişinin persona'sından platforma terfi ettirmek, "backlog kendini teslim ediyor"un *temiz* anlamına gelmesini sağlayan şeydir.
