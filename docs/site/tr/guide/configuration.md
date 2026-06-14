# Yapılandırma

`atl`'nin kendi yapılandırma sistemine kısa bir rehber — neyi ayarladığı, Claude Code'un `settings.json` dosyasıyla nasıl ilişkilendiği ve onu nasıl inceleyip düzenleyeceğin.

## İki yapılandırma yüzeyi {#two-configuration-surfaces}

atl ekosistemi iki farklı yapılandırma dosyası taşır. Aralarında çakışma yok.

| Yüzey | Yöneten | Varsayılan konum | Düzenleme komutu | Şema doğrulaması | Proje seviyesi desteği |
| --- | --- | --- | --- | --- | --- |
| **Claude Code ayarları** | Claude Code | `~/.claude/settings.json` | `claude` `/config` slash komutu | Yok (serbest biçimli) | `.claude/settings.local.json` |
| **atl yapılandırması** | atl CLI | `~/.atl/config.json` | `atl config edit` | JSON Schema (`atl-config.schema.json`) | `./.atl/config.json` |

**Claude Code'un `settings.json` dosyası** Claude Code'un davranışını yönetir: hangi hook'lar tetiklenir, editör kısayolları, IDE entegrasyonu, tema, kullanılan model.

**atl'nin `config.json` dosyası** atl'nin kendi davranışını yönetir: otomatik güncelleme throttle'ı, learning-capture geriye-bakma süresi, brainstorm marker üst sınırı, dil seçimi.

İkisi yalnız tek bir noktada kesişir: atl'nin otomatik güncelleme akışı, [`atl setup-hooks`](/tr/cli/setup-hooks) tarafından `~/.claude/settings.json` içine kaydedilen `SessionStart` ve `UserPromptSubmit` hook'ları üzerinden Claude Code'a bağlanır. Hook'lar yüklüyse her zaman `atl session-start` / `atl update` çağırır; `~/.atl/config.json` ise bu çağrıların ne yapacağını (marker tarayıp taramayacağı, throttle penceresi vb.) ayarlar.

## "Hangisini düzenliyorum?" {#which-one-am-i-editing}

- "atl'nin yeni-sürüm-var bildirimini kapatmak istiyorum." → **atl yapılandırması** — `autoUpdate.selfCheckEnabled = false` yap.
- "Claude Code'un editör tamamlama önerilerini kapatmak istiyorum." → **Claude Code ayarları** (atl IDE davranışını kontrol etmez).
- "atl'nin ilk taramada daha az transcript taramasını istiyorum." → **atl yapılandırması** — `learningCapture.firstRunLookbackDays` değerini düşür.
- "Hook'ların her promptta tetiklenmesini durdurmak istiyorum." → **her ikisi**: `atl setup-hooks --remove` (hook'ları `settings.json`'dan kaldırır), ya da hook'ları kayıtlı tutup atl yapılandırmasında `autoUpdate.promptSubmitEnabled = false` yap (hook tetiklenmeye devam eder ama no-op olur).
- "atl'nin Türkçe konuşmasını istiyorum." → **atl yapılandırması** — `cli.locale = "tr"` yap. (cli-localization işi tamamlanana kadar şu an `en` ile aynı davranır.)

## 9 kullanıcı-ayarlanabilir anahtar {#the-9-user-tunable-keys}

### `cli.locale`

`enum["en", "tr"]` · varsayılan `"en"`

atl prompt'larının kullanıcıya görünen dili. v1 sadece İngilizce davranır; `"tr"` cli-localization işi için ayrılmıştır ve şu anda `"en"` ile aynı şekilde çalışır.

### `autoUpdate.sessionStartEnabled`

`boolean` · varsayılan `true` (önerilen)

SessionStart hook'u tetiklendiğinde `atl session-start`'ı çalıştırır (cache çekme + önceki transcript marker tarama + atl-sürüm kontrolü). Hook'u kaldırmadan opt-out etmek için `false` yap.

### `autoUpdate.promptSubmitEnabled`

`boolean` · varsayılan `true`

UserPromptSubmit hook'u tetiklendiğinde `atl update`'i (throttle ile) çalıştırır. Uzun oturumlarda cache'i sürekli taze tutar; mesaj başına küçük bir git-fetch maliyeti getirir.

### `autoUpdate.throttleMinutes`

`integer`, `[1, 1440]` aralığında · varsayılan `30`

Ardışık prompt-submit otomatik güncellemeleri arasındaki minimum dakika. Düşük değerler cache'i daha taze tutar ama git-fetch sayısını artırır; varsayılan 30 ikisini dengeler.

### `autoUpdate.selfCheckEnabled`

`boolean` · varsayılan `true`

`atl session-start`'ın yeni atl binary için GitHub releases'i yoklamasını sağlar. brew/scoop'un yükseltmeleri merkezi olarak yönettiği makinelerde devre dışı bırak.

### `autoUpdate.selfCheckHours`

`integer`, `[1, 168]` aralığında · varsayılan `24`

Ardışık self-check yoklamaları arasındaki minimum saat. Varsayılan binary release temposuna uyar.

### `learningCapture.autoScanEnabled`

`boolean` · varsayılan `true`

`atl session-start`'ın önceki-oturum transcript'lerini `<!-- learning -->` markerları için taramasını ve işlenmemiş olanları yeni oturumun `additionalContext`'ine raporlamasını sağlar.

### `learningCapture.firstRunLookbackDays`

`integer`, `[1, 365]` aralığında · varsayılan `7`

Bir proje için ilk taramada (henüz state dosyası yok), kaç günlük transcript dikkate alınacak. Sonraki taramalar `~/.atl/state/learning-capture-state.json` içindeki proje-bazlı `lastProcessedAt` zaman damgasını kullanır.

### `brainstorm.markerBulletCap`

`integer`, `[1, 50]` aralığında · varsayılan `8`

Bir kapsamın `CLAUDE.md` / `README` dosyasına aynı anda iliştirilen aktif brainstorm madde sayısının üst sınırı. Sayı bunu aştığında, en eski maddeler marker bloğundan düşer (brainstorm dosyalarının kendisi asla silinmez).

## Komutlar {#commands}

### İncele {#inspect}

```bash
atl config show              # etkin yapılandırma (defaults <- global <- project), JSON olarak
atl config show --table      # anahtar / değer / kaynak tablosu
atl config show --global     # ham ~/.atl/config.json içeriği
atl config show --project    # ham ./.atl/config.json içeriği
```

### Düzenle {#edit}

```bash
atl config init              # ilk-defa karşılama + Q&A → ~/.atl/config.json yaz
atl config edit              # mevcut global yapılandırma üzerinde Q&A
atl config edit --project    # projenin ./.atl/config.json dosyası üzerinde Q&A
```

Q&A 9 anahtarı tek tek ekran ekran gezdirir, sonunda bir özet sunar. Boolean sorular `Y`/`N` kısayollarını kabul eder; integer sorular aralığı satır içinde doğrular. Bir önceki soruya dönmek için `Esc`; vazgeçme onayını açmak için `Ctrl+C`.

### Sıfırla {#reset}

```bash
atl config reset              # interaktif: onay + global'e varsayılanları yaz
atl config reset --yes        # script-uyumlu: onayı atla
atl config reset --project    # interaktif: proje overlay'ini sil
```

## Dosya düzeni {#file-layout}

```
~/.atl/                                # global atl dizini
├── config.json                        # global kullanıcı yapılandırması
├── state/
│   ├── learning-capture-state.json    # tarama ilerlemesi + işlenen marker hash'leri
│   └── docs-sync-state.json           # docs-sync skill state
├── cache/
│   ├── last-repo-check                # otomatik güncelleme throttle damgası
│   └── last-self-check                # binary sürüm-kontrolü throttle damgası
└── install-marker.json                # ilk-kurulum onboarding kapısı

./.atl/                                # proje overlay'i (opsiyonel)
└── config.json                        # projeye özel override'lar
```

Birleştirme sırası `defaults <- global <- project`, alan seviyesinde derin birleşim. Yalnız `{"schemaVersion": 1, "cli": {"locale": "tr"}}` içeren bir proje dosyası sadece `cli.locale`'i override eder; diğer her anahtar önce global'e, sonra defaults'a düşer.

## Şema sürümleme {#schema-versioning}

Diskteki dosya biçimi zorunlu `schemaVersion` alanıyla sürümlenir (şu an `1`). İleride yapılacak bumplar `~/.atl/config.json.bak.v<N>` yedeği yazıp sessiz auto-migration çalıştırır. Kanonik şema [`core/schemas/atl-config.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/atl-config.schema.json) altında yaşar; `atl migrate` tarafından otomatik uygulanır (`atl update` / `atl session-start` üzerinden de tetiklenir).

## Karar bağlamı {#decision-context}

İki yüzey ayrımı, neyin yapılandırma (vs kural, vs hardcode) sayılacağını belirleyen keystone testi ve migration mekaniği [`atl-config-system`](https://github.com/agentteamland/workspace/blob/main/.atl/docs/atl-config-system.md) brainstorm'unda kararlaştırıldı. Keystone testi tek cümlede: *iki kullanıcı bunu farklı yapılandırsa hâlâ aynı atl'yi mi kullanıyor olurlar?* — yapılandırma için evet, kural için hayır.
