# `atl learning-capture`

Claude Code transkriptlerini, Claude'un önceki konuşmalarda düşürdüğü satır içi `<!-- learning ... -->` işaretçileri için tara ve bir sonraki oturumun ilk turunda harekete geçilebilecek kısa bir rapor yazdır.

[`atl setup-hooks`](/tr/cli/setup-hooks) komutunun kurduğu `SessionStart` hook'u tarafından sürülür — `atl session-start` birleşik komutunun içine sarılmıştır. Test ya da anlık tarama için elle de çağrılabilir.

`atl` ≥ 1.1.0 gerektirir.

## Bu neden var?

Otomatik bir yakalama adımı olmadan iki tür bilgi her zaman elden kayıp gider:

1. **Öğrenmeler kaydedilmez.** Kullanıcılar `/save-learnings` çalıştırmayı unutur, ajan da her zaman kendiliğinden teklif etmez. Kararlar, hata düzeltmeleri ve keşifler konuşmayla birlikte kaybolur.
2. **Belgeler bayatlar.** Bir özellik yayımlandığında ya da bir davranış değiştiğinde README ve dokümantasyon sitesi günler hatta haftalar geride kalır — ya da sonsuza dek.

`atl learning-capture` ile birlikte iki eşli core kuralı ([learning-capture](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) + [docs-sync](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md)) bu boşlukların ikisini de kapatır:

- Claude, konuşma sırasında gerçek bir öğrenme anı geçtiğinde satır içi bir `<!-- learning -->` işaretçisi düşürür. İşaretçiler ~50 jeton, HTML yorumu olarak biçimlenmiş (görüntülenmiş çıktıda görünmez) ve oturumda ilginç hiçbir şey yoksa bedavaya görmezden gelinir.
- *Bir sonraki* oturumun başında çalıştırma ortamı `atl session-start` komutunu çağırır; o da `atl learning-capture --previous-transcripts` komutunu çağırır. Son başarılı `/save-learnings` çalıştırmasından bu yana değiştirilmiş her proje transkriptini tarar, işaretçileri bulur ve stdout üzerine kısa bir rapor yazdırır — `SessionStart` bu çıktıyı Claude'un `additionalContext` alanına doğru biçimde ulaştırır.
- Claude'un ilk turu `/save-learnings --from-markers --transcripts <paths>` komutunu çağırır; bu, işaretçileri journal + wiki + ajan children'ı + beceri learnings'ine işler — ARTI `doc-impact` alanı belirlenmiş her işaretçi için README / dokümantasyon sitesi taslakları hazırlar.

Hiçbir şey herkese açık depolara kendiliğinden push'lanmaz. Taslakları sen ya da Claude inceler.

## Kipler

| Kip | Çağrı | Ne zaman |
|---|---|---|
| **Önceki transkriptler** (önerilen) | `atl learning-capture --previous-transcripts` | `SessionStart` hook'u tetiklendiğinde `atl session-start` tarafından kullanılır. Çoklu transkript taraması, durum dosyası güdümlü kesim noktası. |
| **Tek transkript** (eski kullanım) | `atl learning-capture --transcript-path <path>` | Belirli bir transkript dosyasının elle taranması. |
| **Stdin yükü** (eski kullanım) | `atl learning-capture < hook-stdin.json` | `transcript_path`'i, bir Claude Code hook'unun stdin JSON yükünden okur. v0.2.0 `SessionEnd` / `PreCompact` kayıtlarıyla geriye dönük uyum için tutuluyor (o olaylar stdout çıktısını Claude'a hiç teslim etmedi — bkz. [setup-hooks tarihçesi](/tr/cli/setup-hooks#history-from-four-hooks-to-two) — ama ikili çağrıyı yine de kabul ediyor). |

## Önceki transkriptler kipi

`atl session-start`'ın çağırdığı kiptir. Durum `~/.claude/state/learning-capture-state.json` dosyasında tutulur:

```json
{
  "projects": {
    "-Users-you-projects-my-app": {
      "lastProcessedAt": "2026-05-02T14:00:31Z"
    }
  }
}
```

Slug, geçerli çalışma dizini yolundaki `/` karakterlerinin `-` ile değiştirilmiş hâlidir. Her çalıştırmada komut:

1. Durum dosyasından slug'ın `lastProcessedAt` değerini okur (ilk kullanımda "7 gün öncesine" varsayılan olur).
2. `~/.claude/projects/{slug}/*.jsonl` altındaki, o kesim noktasından sonra değişen her transkript dosyasını sıralar.
3. Her birinde `<!-- learning -->` blokları için tarama yapar (v1.1.1 gürültü filtresi ile — aşağıya bak).
4. İşaretçi bulunduğunda stdout üzerine tek birleşik bir rapor yazdırır; bulunmadığında sessizdir.

Durum dosyasını başarılı bir `/save-learnings` çalıştırmasından sonra `/save-learnings` yazar. CLI dosyaya doğrudan asla yazmaz — bu, çöken bir taramadan doğacak yarım yazma bozulmasını önler.

## İşaretçi biçimi

İşaretçiler, gevşek YAML alanları içeren satır içi HTML yorumlarıdır:

```
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7-day JWT refresh chosen because we want long sessions; user logs in once a week max.
-->
```

| Alan | Zorunlu | İzin verilen değerler | Amacı |
|---|---|---|---|
| `topic` | evet | kebab-case (küçük harfler / rakamlar; ayraç olarak tire ya da nokta) | Wiki / children / learnings sayfasının adı olur. |
| `kind` | evet | `bug-fix`, `decision`, `pattern`, `anti-pattern`, `discovery`, `convention` | Öğrenmeyi sınıflandırır. |
| `doc-impact` | hayır | `none`, `readme`, `docs`, `both`, `breaking` | `none` dışında olduğunda doc-sync taslaklarını tetikler. |
| `body` | evet | bir ya da daha çok cümle | Asıl öğrenme içeriği — DAİMA NEDEN'i içer. |

HTML yorumları görüntülenmiş Markdown çıktısında görünmez; bu nedenle işaretçiler konuşma arayüzünü kirletmez; yalnızca tarayıcının gördüğü transkriptte yaşar.

## v1.1.1 gürültü filtresi

Tarayıcının önceki sürümleri, transkriptin herhangi bir yerindeki `<!-- learning` çıplak alt-dizesini eşleştiriyordu. İşaretçi biçimini *tartışan* oturumlar — kural yeniden yazımları, beceri yeniden yazımları, öğrenme yakalama üzerine beyin fırtınaları — bir sonraki oturumun sayısını 10-25× şişiriyordu çünkü araç çıktısı metinleri okudukları her işaretçi parçasını yansıtıyordu. v1.1.1 filtresi şunları reddeder:

- Assistant olmayan turlar (araç kullanımı girdisi, araç sonucu çıktısı, kullanıcı mesajları, özet olayları). Yalnızca `message.role == "assistant"` metin içeriği taranır.
- Kebab-case bir konu içermeyen işaretçiler. Düzenli ifade `^[a-z0-9]+([-.][a-z0-9]+)*$` biçimindedir — büyük harfleri, boşlukları, üç nokta yer tutucularını (`topic: ... doc-impact ...`) ve `bug-fix | decision | pattern | ...` gibi düz alan-belirtim metnini reddeder.

Doğrulama taraması: 5 çalışma alanı transkriptinde 149 ham alt-dize eşleşmesi → filtre sonrası 16 gerçek işaretçi. Geriye kalan 133 tanesi biçim belgelendirmesi, düz metin anmaları ve araç alıntılamasıydı.

## Çıktı

### Boş (değişen transkript yok ya da işaretçi bulunamadı)

```
(--silent-if-empty geçildiğinde sessiz — SessionStart hook yolu bunu kullanır)
```

`--silent-if-empty` olmadan:

```
📝 learning-capture: scanned 3 transcripts, no markers found
```

### İşaretçi bulundu

```
🧠 learning-capture: 7 unprocessed markers across 2 transcripts
  by kind: 3 decision, 2 pattern, 1 discovery, 1 bug-fix
  3 markers require doc drafts (README / doc site) — see docs-sync rule

→ Run: /save-learnings --from-markers --transcripts <path1>,<path2>
```

`SessionStart` bu metni stdout üzerinden Claude'un `additionalContext` alanına enjekte eder. `learning-capture` core kuralı, Claude'a ilk turda adı geçen `/save-learnings` komutunu çağırmasını söyler; bu komut da işaretçileri şunlara işler:

- **journal/{date}_{agent}.md** kayıtları (kronolojik, ajan başına kayıt).
- **wiki/{topic}.md** güncellemeleri (güncel doğru, yerine yazma biçimli).
- **agents/{agent}/children/{topic}.md** kendiliğinden büyüyen içerik (`knowledge-base-summary` frontmatter alanıyla; `agent.md` Knowledge Base bölümü kendiliğinden yeniden inşa edilir).
- **skills/{skill}/learnings/{topic}.md** kendiliğinden büyüyen içerik (`skill.md` Accumulated Learnings bölümü kendiliğinden yeniden inşa edilir).
- `doc-impact` taşıyan işaretçiler için **doküman taslakları** (incelenmek üzere sunulur, kendiliğinden push'lanmaz).

Başarılı işlemenin ardından `/save-learnings`, durum dosyasındaki `lastProcessedAt` değerini ilerletir; böylece aynı işaretçiler bir sonraki `SessionStart`'ta yeniden raporlanmaz.

## Bayraklar

| Bayrak | Varsayılan | Amaç |
|---|---|---|
| `--previous-transcripts` | (kapalı) | Durum dosyasının sürdüğü çoklu transkript taraması (`atl session-start` tarafından kullanılır). |
| `--silent-if-empty` | `false` | İşaretçi bulunmadığında çıktı vermez (hook'lar için). |
| `--transcript-path <path>` | (stdin JSON'dan) | Açık tek dosyalık tarama; hem durum dosyasını hem stdin yükünü atlar. |
| `--help` | — | Komut yardımını gösterir. |

## Maliyet modeli

| Senaryo | Claude'a jeton maliyeti | Zaman maliyeti |
|---|---|---|
| `--previous-transcripts` ile değişen transkript yok | 0 | <1 ms (tek dosya bilgisi çağrısı). |
| `--previous-transcripts` ile N transkript, işaretçi yok | 0 (silent-if-empty ile) | Transkript MB'si başına ~10 ms. |
| `--previous-transcripts` ile N işaretçi | Rapor için ~80 jeton. | Transkript MB'si başına ~10 ms. |
| `/save-learnings --from-markers` (işleme) | Maliyet işaretçi sayısıyla orantılı, transkript boyutuyla değil. | Saniyeler. |

Tasarım bilinçli olarak sıkıcı oturumları bedava kılar. Yalnızca gerçek öğrenmeler maliyet üretir ve maliyet, gerçekten öğrenilenle orantılı ölçeklenir — asla konuşma uzunluğuyla değil.

## Elle test

```bash
# Tek bir assistant işaretçisi içeren bir sentetik transkript oluştur (rol+içerik biçimine dikkat et)
cat > /tmp/test.jsonl <<'EOF'
{"message":{"role":"assistant","content":[{"type":"text","text":"<!-- learning\ntopic: my-topic\nkind: decision\ndoc-impact: none\nbody: test.\n-->"}]}}
EOF

# Tara
atl learning-capture --transcript-path /tmp/test.jsonl
```

Beklenen çıktı:

```
🧠 learning-capture: 1 unprocessed marker across 1 transcript
  by kind: 1 decision

→ Run: /save-learnings --from-markers --transcripts /tmp/test.jsonl
```

## Tarihçe

`atl v0.2.0` (2026-04-24) işaretçi protokolünü `SessionEnd` ve `PreCompact` hook'larıyla tanıttı. Claude Code v2.1.x'e göre o hook'lar stdout çıktısını Claude'un `additionalContext` alanına teslim ETMEZ. v0.2.0 sonrasındaki ay boyunca bakımcı çalışma alanında 9 oturumda biriken 324 işaretçi **sıfır** otomatik işleme üretti — her gerçek `/save-learnings` çalıştırması elle yapılan çağrıdan geldi. Yakalama ikilisi çalışıyordu; tetikleme yolu yanlıştı.

`atl v1.1.0` (2026-05-02) `--previous-transcripts` kipini, `~/.claude/state/learning-capture-state.json` durum dosyasını ve bu komutu `SessionStart` hook'undan çağıran `atl session-start` birleşik sarmalayıcısını tanıttı (stdout'u Claude'a güvenilir biçimde teslim eden *tek* hook olayı). İşaretçi protokolü değişmedi.

`atl v1.1.1` (2026-05-02) gürültü filtresini ekledi (assistant rolü + kebab-case konu).

## İlgili

- [`atl setup-hooks`](/tr/cli/setup-hooks) — bu komutu `atl session-start` üzerinden süren `SessionStart` hook'unu kurar.
- [`atl update`](/tr/cli/update) — `atl session-start` tarafından çağrılan diğer parça.
- [learning-capture kuralı](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) — Claude'un ne zaman işaretçi düşürmesi gerektiğine dair davranış belirtimi.
- [docs-sync kuralı](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md) — öngörülü doküman güncellemeleri için eşli kural (`doc-impact` alanını kullanır).
- [`/save-learnings` becerisi](https://github.com/agentteamland/core/blob/main/skills/save-learnings/skill.md) — sistemin işleme yarısı.
