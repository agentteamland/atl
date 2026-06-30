# `atl init`

Seçtiğin tier için yalın bir başlangıç `CLAUDE.md`'si oluşturur — **yalnızca dosya yoksa**, böylece kendi `CLAUDE.md`'in asla üzerine yazılmaz.

`CLAUDE.md`, Claude Code'un proje (ve global) talimatı olarak otomatik yüklediği dosyadır. ATL her tier için bir başlangıç şekli sunar; böylece boş dosyadan başlamazsın — senin için işaretli kısımları doldurursun, `/brainstorm` ve `/drain` becerileri de proje dosyasındaki kendi marker bloklarını zamanla yönetir.

## Kullanım

```bash
atl init                 # proje köküne CLAUDE.md (varsayılan)
atl init --project       # aynısı, açıkça
atl init --global        # kişisel ~/.claude/CLAUDE.md persona'n
atl init --monorepo      # yalın ~30 satırlık yönlendirme dosyası
```

Üç bayrak birbirini dışlar. `atl install` da bir projede `CLAUDE.md` yoksa **project** başlangıcını otomatik düşürür (bkz. [Ne yapar?](#ne-yapar)); bu yüzden `atl init`'i elle genelde yalnızca **global** persona ya da **monorepo** yönlendirme dosyası için çalıştırırsın.

## Tier'ler

| Bayrak | Hedef yol | Şekil |
|---|---|---|
| `--project` (varsayılan) | `<proje>/CLAUDE.md` | Hibrit: ATL'nin yönettiği marker blokları (aktif brainstorm'lar, bilgi indeksi — `/brainstorm` + `/drain` tarafından bakılır) artı kullanıcının sahip olduğu, kanıttan doldurulan gerçekler (stack, komutlar, kurallar) ve opsiyonel bir skill-yönlendirme tablosu. Yumuşak bütçe ≤ ~60 satır. |
| `--global` | `~/.claude/CLAUDE.md` | Saf kullanıcı persona'sı — Claude'un her yerde nasıl çalışmasını istediğin. **ATL burada hiçbir şey yönetmez.** Yumuşak bütçe ≤ ~80 satır. |
| `--monorepo` | `<repo>/CLAUDE.md` | Proje şeklinin özelleşmiş + yalın hâli: bir layout tablosu ve kurallar **pointer** olarak, içeri gömülmeden. Yumuşak bütçe ~30 satır. |

Tier'ler, bütçeleri ve yönetilen-vs-sahip-olunan ownership modeli [Claude Code kuralları](/tr/guide/claude-code-conventions) sayfasında tam olarak anlatılır.

## Ne yapar?

`atl init`:

1. Seçilen tier için hedef yolu çözer (global → `~/.claude/CLAUDE.md`; project / monorepo → proje kökünün `CLAUDE.md`'si).
2. **Orada zaten bir `CLAUDE.md` varsa hiçbir şey yapmaz** — dosyan kullanıcıya aittir ve asla üzerine yazılmaz.
3. Aksi hâlde tier'in başlangıç iskeletini yazar (proje / repo adını doldurarak) ve oluşturduğu yolu yazdırır.

`atl install` aynı project-tier scaffold'ı en-iyi-çaba adımı olarak çalıştırır: `CLAUDE.md`'si olmayan bir projeye takım kurduğunda ATL project başlangıcını düşürür ki `/brainstorm` ve `/drain` bloklarının bir evi olsun. Yalnızca-yoksa çalışır ve kurulumu asla başarısız kılmaz.

## İdempotenlik — yeniden çalıştırması güvenlidir

`CLAUDE.md` zaten varken `atl init`'i (ya da `atl install`'ı) yeniden çalıştırmak bir no-op'tur — dosyanın var olduğunu bildirir ve el değmeden bırakır. `--force` yoktur; var olan bir `CLAUDE.md`'i değiştirmek, bir scaffold'ın yapması gereken değil, bilinçli bir elle eylemdir.

## İlgili

- [Claude Code kuralları](/tr/guide/claude-code-conventions) — üç tier, token bütçeleri, ownership modeli ve proje dosyasının taşıdığı marker bloklar
- [`atl install`](/tr/cli/install) — bir takım kurar ve yoksa project başlangıcını düşürür
- [`/brainstorm`](/tr/skills/brainstorm) — proje `CLAUDE.md`'sindeki `<!-- brainstorm:active -->` bloğunu yönetir
- [`/drain`](/tr/skills/drain) — `<!-- wiki:index -->` bilgi haritası bloğunu yönetir
