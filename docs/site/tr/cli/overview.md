# CLI'a Genel Bakış

`atl` ajan takımlarını kurar, güncel tutar, ajanlarının öğrendiği kazanımları dolaşıma sokar ve arka planda kendi kendini çalıştırır; böylece sen yalnızca projene odaklanırsın.

Komutlar üç gruba ayrılır: elle çalıştırdığın **takım komutları**, ajanlarının öğrendiklerini terfi ettirip paylaşan **kazanım-dolaşımı** halkası ve Claude Code'un senin yerine tetiklediği **otomasyon**. Aksi belirtilmedikçe her şey **mevcut proje** (`atl`'ı içinde çalıştırdığın dizin) üzerinde işler; bunun üstünde, projenin gölgelediği ikinci bir **kullanıcı-global** katman vardır (en yakın olan kazanır).

## Takım komutları

| Komut | Ne yapar |
|---|---|
| [`atl install`](/tr/cli/install) | Bir takımı kısa adıyla (GitHub destekli index'e göre çözümlenerek) mevcut kapsama (scope) kurar. |
| [`atl list`](/tr/cli/list) | Bu projede kurulu takımları gösterir. |
| [`atl remove`](/tr/cli/remove) | Bir takımı kaldırır. |
| [`atl update`](/tr/cli/update) | Bir ya da tüm kurulu takımlar için en günceli çeker. |
| [`atl search`](/tr/cli/search) | Takım kataloğunda (GitHub destekli index) arama yapar. |
| [`atl gc`](/tr/cli/gc) | Hiçbir manifestin sahiplenmediği sahipsiz varlıkları geri kazanır — kurulumun geri-alınabilir tersi (varsayılan kuru-çalışma; yumuşak-silme + geri-al). |

## Kazanım-dolaşımı komutları

Ajanların çalıştıkça **kazanım** biriktirir — yeni öğrenmeler, keskinleşen skill'ler, projeye özel kurallar. Bu komutlar o kazanımları üç-halkalı merdiven boyunca dışarı taşır.

| Komut | Ne yapar |
|---|---|
| `atl promote` | Projeye özel kazanımları kullanıcı-global katmana yükseltir (böylece her proje yararlanır). |
| `atl publish` | Global katmandaki kazanımlarını yayımlar — kendi takımını yeniden yayımlar ya da onları bir GitHub PR'ı olarak yukarı (upstream) önerir. |
| `atl pin` | Projeye özel bir yolun global katmana terfi ettirilmesini engeller. |
| `atl unpin` | Daha önce sabitlenmiş (pin) bir yolun yeniden terfi ettirilebilmesine izin verir. |
| `atl learnings` | Kalıcı öğrenme kuyruğunu inceler: `status` (kanal/proje başına bekleyenler), `peek` (öğeleri listeler; `/drain` skill'inin kullandığı), `ack <id>` (bir öğeyi işlenmiş olarak işaretler). |

## Otomasyon komutları

Bunlar [`atl setup-hooks`](/tr/cli/setup-hooks) tarafından Claude Code hook'larına bağlanır ve **gözetimsiz** çalışır — normalde bunları hiç yazmazsın. Yalnızca hook çıktısında tanıyabilesin ya da sorun giderirken başvurabilesin diye burada listelenmişlerdir.

| Komut | Ne yapar |
|---|---|
| [`atl setup-hooks`](/tr/cli/setup-hooks) | Aşağıdaki otomasyonu süren Claude Code hook'larının (`SessionStart`, `UserPromptSubmit`) tek seferlik kurulumu/kaldırılması. |
| `atl session-start` | `SessionStart` hook'unun çalıştırdığı, açılış anındaki bakım işi (önbellek yenileme + otomatik güncelleme + önceki-transkript işaretçi taraması + öz-sürüm denetimi). |
| `atl tick` | Oturum içi bakım tıkı (prompt'a iliştirilerek her 5–10 dakikada bir): kısılmış (throttled) arka plan işlerini boşaltır. |
| `atl doctor` | Kendi kendini onaran daemon — sapmayı (drift) teşhis eder ve kurulumu kendiliğinden onarır. |

> v1 ile karşılaştırıldığında **`config`, `migrate` ya da `learning-capture` komutu yoktur.** Öğrenme yakalama artık otomatiktir (işaretçiler, `atl learnings`'in incelediği ve `/drain` skill'inin işlediği kalıcı bir kuyruğa düşer); yapılandırma ve durum-dosyası taşıma v2 yüzeyinin parçası değildir.

## Eşlik eden skill'ler

`atl` platformun belirlenimci (deterministic) yarısıdır; muhakeme ağırlıklı yarı, takımların kurduğu Claude Code skill'lerinde yaşar:

- **`/drain`** — öğrenme kuyruğunu ajan bilgi tabanlarına işler (v1'deki `/save-learnings`).
- **`/create-pr`** — branch → review → commit → PR.
- **`/create-code-diagram`** — kod tabanının bir mimari/sınıf diyagramını üretir.
- **`/brainstorm`**, **`/rule`**, **`/rule-wizard`** — kuralları tasarla, yaz ve iskeletini kur.

Bu ayrım bilinçlidir: **CLI belirlenimcidir** (aynı girdiler, aynı sonuç, LLM yok), **skill'ler LLM güdümlüdür** (senin özgül kodun üzerinde muhakeme yaparlar).

## Genel bayraklar

| Bayrak | Etki |
|---|---|
| `--help`, `-h` | Kullanımı yazdırır ve çıkar. |
| `--version`, `-v` | Kurulu `atl` sürümünü yazdırır. |

Her komutun kendi `--help` sayfası vardır: `atl install --help`, `atl publish --help` ve benzeri.

## `atl`'ın tuttuğu durum (state)

Varlıklar (assets), iki kapsamdan birinde, **Claude Code'un kendi dizinlerinde** yaşar — ATL'a ait ayrı bir varlık deposu yoktur:

```
~/.claude/                 ← kullanıcı-global katman (tüm projeler arasında paylaşılan agents/skills/rules)
<project>/.claude/         ← proje katmanı (global'i gölgeler; en yakın kazanır)
```

`atl`'ın kendi defter tutması `~/.atl/` (kullanıcı-global) ve `<project>/.atl/` (proje başına) altında yaşar:

```
~/.atl/
├── queue.db               ← kalıcı öğrenme kuyruğu (bbolt)
├── index.json             ← önbelleğe alınmış takım kataloğu (atl update ile yenilenir)
├── generation             ← global katman değişim sayacı (her prompt fan-out'unu yönlendirir)
├── pins.json              ← terfiden alıkonan yollar
├── cache/                 ← önbellek damgaları
└── installed/             ← takım başına kurulum manifestoları + bütünlük (integrity) tabanları
```

## Felsefe

- **Belirlenimci.** Aynı girdiler, aynı sonuç. Gizli durum yok.
- **Gözlemlenebilir.** Her eylem ne yaptığını yazdırır. Bir spinner'a değil, çıktıya bak.
- **Eli boş bırak.** Otomasyon komutları, sen düşünmeden her şeyi güncel tutar.

## Sonraki adım

- **[`atl install`](/tr/cli/install)** — en çok çalıştıracağın komut.
- **[`atl search`](/tr/cli/search)** — katalogda ne olduğunu keşfet.
