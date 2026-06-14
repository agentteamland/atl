# CLI genel bakışı

`atl`'nin sekiz komutu vardır. Beş **kullanıcı komutu**, aksi belirtilmedikçe **mevcut proje** üzerinde (yani `atl`'yi çalıştırdığın dizinde) iş görür. Üç **otomasyon komutu** ise tipik olarak Claude Code hook'larına bağlanır ve gözetimsiz çalışır — bunları elle yalnızca kurulum ya da sorun gidermek için çalıştırırsın.

## Kullanıcı komutları

| Komut | Ne yapar |
|---|---|
| [`atl install`](/tr/cli/install) | Bir takımı kayıt defteri adı veya Git URL'si ile kurar. |
| [`atl list`](/tr/cli/list) | Bu projede kurulu takımları gösterir. |
| [`atl remove`](/tr/cli/remove) | Bir takımı kaldırır. |
| [`atl update`](/tr/cli/update) | Bir takımın ya da kurulu tüm takımların son sürümünü çeker. |
| [`atl search`](/tr/cli/search) | Herkese açık kayıt defterinde arama yapar. |

## Otomasyon komutları

| Komut | Ne yapar |
|---|---|
| [`atl setup-hooks`](/tr/cli/setup-hooks) | Otomatik güncelleme ve öğrenme yakalamayı bağlayan Claude Code hook'larını (`SessionStart`, `UserPromptSubmit`) tek seferlik kurar veya kaldırır. |
| `atl session-start` | `SessionStart` hook'unun çağırdığı, açılış zamanına ait birleşik sarmalayıcı (önbellek çekimi + sembolik bağdan kopyaya geçiş + kendiliğinden yenileme + önceki transkript işaretçi taraması + `atl` kendi sürüm denetimi). Normalde elle çalıştırılmaz. |
| [`atl learning-capture`](/tr/cli/learning-capture) | Claude Code transkriptlerini `<!-- learning -->` işaretçileri için tarar. `SessionStart` sarmalayıcısı tarafından çağrılır; test ya da geriye dönük tarama için elle de çalıştırılabilir. |

## Global bayraklar

| Bayrak | Etkisi |
|---|---|
| `--help`, `-h` | Kullanımı yazdırır ve çıkar. |
| `--version`, `-v` | Kurulu `atl` sürümünü yazdırır. |

Her komutun kendi `--help` sayfası vardır: `atl install --help`, `atl search --help` ve benzerleri.

## `atl` hangi durumu tutar?

**Paylaşılan önbellek** (makine başına bir adet):

```
~/.claude/repos/agentteamland/
└── <team-name>/          ← klonlanmış Git deposu; tüm projelerde yeniden kullanılır
```

**Proje başına durum** (`atl`'yi çalıştırdığın her dizin için):

```
<project>/
└── .claude/
    ├── .team-installs.json       ← hangi takımlar hangi sürümde kurulu
    ├── agents/                   ← takım ajanlarının kopyaları
    ├── skills/                   ← takım becerilerinin kopyaları
    ├── rules/                    ← takım kurallarının kopyaları
    └── ...
```

Kopyalar paylaşılan önbelleğe işaret eder. `atl update`'in aynı anda her projede etkili olmasının nedeni budur: projeyi değil önbelleği güncellersin.

## Çıkış kodları

| Kod | Anlamı |
|---|---|
| `0` | Başarılı. |
| `1` | Genel hata (geçersiz argüman, takım bulunamadı, ağ sorunu). |
| `2` | Doğrulama hatası (`team.json` şemaya uymadı). |
| `3` | Kalıtım hatası (döngülü zincir, üst takım yok). |

## Felsefe

- **Belirlenimci.** Aynı girdiler, aynı kopyalar. Gizli durum yok.
- **İdempotent.** Zaten kurulu bir takıma `atl install` komutunu yeniden çalıştırmak işlem yapmaz (ya da yalnızca çekim yapar).
- **Gözlemlenebilir.** Her eylem ne yaptığını yazdırır. Dönen ikon yerine çıktının kendisini kullan.

## Sıradaki

- **[`atl install`](/tr/cli/install)** — en sık çalıştıracağın komut.
- **[`atl search`](/tr/cli/search)** — kayıt defterinde ne olduğunu keşfet.
