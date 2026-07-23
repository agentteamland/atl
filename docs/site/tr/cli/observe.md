# `atl observe`

Proaktif gözlemcinin **deterministik yarısı**: bir taramanın zamanı gelip gelmediğini bildirir ve [`/observe`](/tr/skills/observe) skill'inin çalıştıktan sonra damgaladığı imleci damgalar. Denetimin kendisi — olgunlaşmış backlog tetikleyicileri ile grep'e dayalı, çelişki testinden geçmiş gizli-boşluk taraması — skill'in LLM işidir. Bu, CLI/Skill sınırıdır: CLI deterministik, skill ise muhakemedir.

ATL karar yüzeyi (bir `.atl/` dizini) olan her projede çalışır. Dışında hiçbir şey yapmaz ve 0 ile çıkar.

## Kullanım

```bash
atl observe            # proaktif gözlemci taramasının zamanı geldi mi bildir
atl observe --record   # HEAD'i son gözlemci taraması olarak damgala (bir /observe koşusundan sonra)
```

## "Zamanı geldi" ne demek

Bir tarama, projenin `.atl/` karar + bilgi yüzeyi son kaydedilen taramadan bu yana hareket ettiğinde — bir karar, sevk edilen bir iş, bir günlük kaydı — **zamanı gelmiş** sayılır; bu, günde birden fazla sinyal vermemesi için **~1 günlük bir kaçış-engeliyle** sınırlanır. Bu ritim, gözlemciyi tam da "sevk edilenle tasarlanan" arasındaki kaymanın en olası olduğu anda tetikler: iş yeni landığında.

Oturum başında `atl session-start`, bu koşulda **"a proactive observer sweep is due — run /observe"** yazdırır; böylece kontrol etmeyi hatırlamak zorunda kalmazsınız. Sinyal ucuzdur (bir git-log ritim kontrolü); pahalı LLM taraması yalnızca buna karşılık `/observe` çağırırsanız çalışır.

`--record`, HEAD + zamanı son tarama olarak damgalar (`~/.atl/observe-state.json` içinde) ve bu kaçış-engelini ~1 gün sıfırlar. `/observe` bunu bir taramanın sonunda çağırır.

## İlgili

- [`/observe`](/tr/skills/observe) — LLM yarısı: olgunlaşmış backlog tetikleyicilerini izle ve gizli boşlukları tara (sevk-edilen-vs-tasarlanan, büyüme/ölçek riskleri, sevk edilmemiş kararlar, kurulumun), grep'e dayalı ve çelişki testinden geçmiş.
- [`atl docs`](/tr/cli/docs) / [`atl rules`](/tr/cli/rules) — aynı imleç mekanizması üzerindeki kardeş deterministik-CLI + LLM-skill destekleri.
