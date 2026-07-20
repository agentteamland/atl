# `atl tick`

Tek bir oturum-içi bakım turu çalıştır — üç-hızlı ritmin sen bir oturumdayken birkaç dakikada bir tetiklediği iş: ucuz bir fan-out (dağıtım), artı kısıtlamalı bir drain (boşaltma) + doctor öz-denetimi.

Bunu neredeyse hiçbir zaman elle çalıştırmazsın. [`atl setup-hooks`](/tr/cli/setup-hooks) bunu `UserPromptSubmit` hook'una `atl tick --throttle=10m` olarak bağlar; böylece mesajlarına kendiliğinden eşlik eder. Elle erişim yüzeyi kurulum, hata ayıklama ve bir turu zorlamak için vardır.

## Ne zaman kullanılır

- **Normalde: doğrudan asla.** Hook üzerinden her mesajda çalışır; ağır kısım yalnızca ~10 dakikada bir tetiklenecek biçimde kısıtlanmıştır.
- **Şimdi bir turu zorlamak için** — örneğin [`/drain`](/tr/skills/drain) çalıştırmadan önce bekleyen yakalama işaretçilerinin kuyruğa taşınmasını istiyorsan — `atl tick` komutunu `--throttle` olmadan çalıştır.
- **Tek bir dosyayı boşaltmak için** (işaretçi ayrıştırıcısını hata ayıklarken) — `atl tick --file <path>`.

## Kullanım

```bash
atl tick                        # şimdi tam bir turu zorla (kısıtlama yok)
atl tick --throttle=10m         # son tick 10 dakikadan kısa süre önceyse drain+doctor turunu atla
atl tick --file <path>          # transkript keşfetmek yerine tek bir dosyayı boşalt (hata ayıklama)
```

`atl tick`, **mevcut proje** üzerinde — yani onu çalıştırdığın dizinde — işler.

## Bir tur ne yapar?

Sırasıyla:

1. **Fan-out (her çağrıda, ~bedava).** Yeni değişmiş dosyaları kullanıcı-global katmandan bu projeye çeker. Bir global nesil sayacıyla (`~/.atl/generation`) korunur: bu proje en son fan-out yaptığından beri global katman değişmemişse, tek bir küçük dosya okumasıdır ve hiçbir şey yapmaz. Bu adım, **kısıtlama diğer her şeyi atladığında bile** çalışır — her mesaja eşlik edecek kadar zaten ucuzdur.
2. **Otomatik-drain sinyali (her çağrıda).** Kalıcı kuyruk bekleyen öğrenmeler ya da profile-fact'ler tutuyorsa, kanal başına tek satırlık bir otomatik-drain sinyali yazdırır (`atl: N learning(s) pending — auto-drain them now in a background subagent …`); böylece ajan arka planda bir [`/drain`](/tr/skills/drain) (ya da `/profile-drain`) alt-ajanı başlatır. Tek bir ucuz sayaç okumasıdır ve kısıtlama kapısından **önce** çalışır; bu yüzden kuyruk boş olmadığı her mesajda tetiklenir — kısıtlama onu asla bastırmaz.
3. **Kısıtlama kapısı.** `--throttle=<dur>` ile, son tick `<dur>` içindeyse tur burada durur — yalnızca fan-out ve otomatik-drain sinyali çalışmıştır (mesaj başına hook'u ucuz tutan hızlı yol). Damga proje-başınadır, `~/.atl/cache/last-tick-<project-hash>` konumunda; böylece farklı projelerdeki eşzamanlı oturumlar birbirinin turunu aç bırakmaz. `--throttle` olmadan (ya da `--throttle=0` ile) kapı her zaman geçer.
4. **Drain.** Bu projenin son tick'ten beri değiştirilmiş Claude Code transkriptlerini keşfeder, assistant metnini çıkarır ve yakalama işaretçilerini kalıcı kuyruğa aktarır — **tam olarak bir kez**. İdempotenlik kuyruğun işaretçi-hash tekrar-elemesinden gelir; bu yüzden aynı metni yeniden boşaltmak kuyruğa yeni hiçbir şey eklemez.
5. **Doctor öz-denetimi.** [`atl doctor`](/tr/cli/doctor) ve [`atl session-start`](/tr/cli/setup-hooks) ile aynı kuyruk-sağlığı + varlık-bütünlüğü denetimlerini çalıştırır; yalnızca OK olmayan (ya da kendini onaran) satırları yazdırır. Her şey sağlıklıyken sessizdir.
6. **Kazanımları yükselt (1→2. halka).** Bu projenin biriken kazanımlarını kullanıcı-global katmana çıkarır. Eklemeli ve çakışma-arşivlidir; bu yüzden elle bir [`atl promote`](/tr/cli/promote) beklemek yerine tick'e eşlik etmesi güvenlidir. Çıkarılacak bir şey olmadığında sessizdir.

Drain adımı öğrenmeleri yalnızca **kuyruğa alır**. Onları bilgi tabanına katmak LLM işidir; bu yüzden CLI/Skill sınırının beceri tarafında kalır — [`/drain`](/tr/skills/drain) tam da bunu yapar; kuyruk boş olmadığında yukarıdaki otomatik-drain sinyaliyle arka planda bir alt-ajan olarak kendiliğinden başlatılır.

## Flag'ler

| Flag | Varsayılan | Etki |
|---|---|---|
| `--throttle <dur>` | `0` | Son tick bu süre içindeyse drain + doctor turunu atla (örneğin `10m`). Fan-out ve otomatik-drain sinyali yine de çalışır. Sıfır/yok bir değer her zaman tam turu çalıştırır. |
| `--file <path>` | `""` | Transkript keşfetmek yerine tek bir dosyayı boşalt. Yalnızca elle/hata ayıklama — kısıtlamayı ve imleci atlar; drain konumunu ilerletmez. |

## Örnek — bir turu zorla

```bash
$ atl tick
tick: scanned 2 transcript(s) — 5 marker(s), 3 new, 2 already queued
```

Son tick'ten beri yeni bir şey olmadığında:

```bash
$ atl tick
tick: no new transcripts to drain
```

Bu proje en son eşitlendiğinden beri global katman değiştiğinde, fan-out satırı da görünür:

```bash
$ atl tick
tick: fanned out 4 file(s) from the global layer
tick: scanned 1 transcript(s) — 2 marker(s), 2 new, 0 already queued
```

## Örnek — tek bir dosyayı boşalt (hata ayıklama)

```bash
$ atl tick --file ./transcript.txt
tick: drained ./transcript.txt — 3 marker(s), 1 new, 2 already queued
```

## İlgili

- [`atl setup-hooks`](/tr/cli/setup-hooks) — `tick`'i `UserPromptSubmit` hook'una bağlar (normalde böyle çalışır).
- [`atl doctor`](/tr/cli/doctor) — tick'in çalıştırdığı aynı sağlık denetimlerinin istek-üzerine yüzeyi.
- [`/drain`](/tr/skills/drain) — kuyruğa alınan öğrenmeleri bilgi tabanına katar (döngünün LLM yarısı).
- [`atl promote`](/tr/cli/promote) — tick'in kendiliğinden yaptığı kazanım-çıkarmanın elle sürümü.
