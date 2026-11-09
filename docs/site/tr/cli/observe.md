# `atl observe`

Proaktif gözlemcinin **deterministik yarısı**: bir taramanın zamanı gelip gelmediğini bildirir ve [`/observe`](/tr/skills/observe) skill'inin çalıştıktan sonra damgaladığı imleci damgalar. Denetimin kendisi — olgunlaşmış backlog tetikleyicileri ile grep'e dayalı, çelişki testinden geçmiş gizli-boşluk taraması — skill'in LLM işidir. Bu, CLI/Skill sınırıdır: CLI deterministik, skill ise muhakemedir.

ATL karar yüzeyi (bir `.atl/` dizini) olan her projede çalışır. Dışında hiçbir şey yapmaz ve 0 ile çıkar.

## Kullanım

```bash
atl observe            # proaktif gözlemci taramasının zamanı geldi mi bildir
atl observe --record   # HEAD'i son gözlemci taraması olarak damgala (bir /observe koşusundan sonra)
atl observe key <tür> <çapa>...   # bir bulgunun kararlı anahtarını yazdır
```

## "Zamanı geldi" ne demek

Bir tarama, projenin `.atl/` karar + bilgi yüzeyi son kaydedilen taramadan bu yana hareket ettiğinde — bir karar, sevk edilen bir iş, bir günlük kaydı — **zamanı gelmiş** sayılır; bu, günde birden fazla sinyal vermemesi için **~1 günlük bir kaçış-engeliyle** sınırlanır. Bu ritim, gözlemciyi tam da "sevk edilenle tasarlanan" arasındaki kaymanın en olası olduğu anda tetikler: iş yeni landığında.

Oturum başında `atl session-start`, bu koşulda **"a proactive observer sweep is due — run /observe now in a background subagent … (per the sweep-dispatch rule)"** yazdırır; böylece kontrol etmeyi hatırlamak zorunda kalmazsınız. Sinyal ucuzdur (bir git-log ritim kontrolü); pahalı LLM taraması kendi isteğinizin yanında bir arka plan subagent'ında çalışır ve bulguları size ya pano öğesi ya da bir [özet](/tr/cli/digest) olarak ulaşır.

## Otomatik yarıyı kapatmak

Bu sinyalin muhatabı ajandır ve `sweep-dispatch` çekirdek kuralı, ajana onu sormadan uygulamasını söyler. Bir taramanın ne zaman çalışacağına kendiniz karar vermek isterseniz:

```bash
export ATL_NO_SWEEP_DISPATCH=1
```

Bundan sonra her tarama sinyali **pasif biçimini** yazdırır — `a proactive observer sweep is due — run /observe to …`, hiçbir subagent adı geçmez, hiçbir kural anılmaz — ve kural açıkça bu biçime uygulanmaz. Raporu korursunuz (hangi taramanın zamanı geldiği ve onu çalıştıracak komut), sizin adınıza hiçbir şey başlatılmaz. Fren yalnız bu taramayı değil, dördünü birden kapsar.

Adında *sevkiyat* geçmesi bilinçlidir: otomatik işten vazgeçmek, işin zamanının geldiğini bilmekten vazgeçmekle aynı şey değildir.

`--record`, HEAD + zamanı son tarama olarak damgalar (`~/.atl/observe-state.json` içinde) ve bu kaçış-engelini ~1 gün sıfırlar. `/observe` bunu bir taramanın sonunda çağırır.

## Bir bulgunun kararlı anahtarı {#the-stable-key-of-a-finding}

`atl observe key`, bir süpürmenin daha önce bildirdiği bir bulguyu tanımak için kullandığı anahtarı yazdırır — [`atl digest add`](/tr/cli/digest) için `--key` olarak ve bir pano kartı oluşturulmadan önce aranan etiket olarak.

```bash
atl observe key stale-truth .atl/backlog.md AB#1201
# atl-observe:618a598a7a20
```

Anahtar, bulgunun nasıl **ifade edildiğinden** değil, neyle **ilgili** olduğundan türetilir. Süpürme, oturum hangi dilde yürüyorsa o dilde düzyazı yazan bir LLM'dir; anahtar süpürmenin uydurduğu bir slug iken, eskimiş tek bir backlog satırı bir kez `observe:backlog-virtual-account-shipped`, günler sonra yeniden `observe:backlog-sanal-hesap-shipped` olarak kartlandı ve önce-kontrol sorgusu eşleşecek hiçbir şey bulamadı. Tanımlayıcılar çevrilmez, olduğu gibi kopyalanır; bu yüzden her süpürmede aynı dizgedir.

- **tür** sabit bir sözlükten biridir, `/observe` mercekleriyle eşleşir: `ripe-trigger`, `shipped-vs-designed`, `growth-risk`, `decided-unshipped`, `stale-truth`, `global-setup`.
- **çapalar** bir kart referansı (`AB#2033`, `atl#434`; büyük/küçük harf duyarsız karşılaştırılır) ya da var olan bir yoldur — projeye göre göreli, mutlak veya `~/…` — isteğe bağlı olarak `yol#Sembol`.

Sıra, tekrarlar, `./` öneki ve kart referansının harf büyüklüğü anahtarı değiştirmez. Serbest bir ifade, bir slug ve bir satır numarası hash'lenmez, **reddedilir**: bunlar tam da koşudan koşuya değişen girdilerdir ve bir satır iğnesi, desteklediği iddia doğru kalırken kayar.

## İlgili

- [`/observe`](/tr/skills/observe) — LLM yarısı: olgunlaşmış backlog tetikleyicilerini izle ve gizli boşlukları tara (sevk-edilen-vs-tasarlanan, büyüme/ölçek riskleri, sevk edilmemiş kararlar, kurulumun), grep'e dayalı ve çelişki testinden geçmiş.
- [`atl docs`](/tr/cli/docs) / [`atl rules`](/tr/cli/rules) — aynı imleç mekanizması üzerindeki kardeş deterministik-CLI + LLM-skill destekleri.
