# Workspace — bakımcı merkezi

[`agentteamland/workspace`](https://github.com/agentteamland/workspace) deposu, AgentTeamLand ekosisteminin **bakımcı merkezidir**. Bir meta-depodur: klonlayıp tek bir betiği çalıştırınca her eş depo (`atl`, `docs`, `.github` vb.) tek bir ağaç altında, `./repos/` dizininde, kontrol edilmiş hâle gelir. Platformun her hareketli parçası bir `cd repos/<name>` uzaklıktadır.

Çalışma alanını, **birden çok depoyu kapsayan bakım işi** yaparken kullan: depolar arası yeniden tasarımlar, çoklu PR yayımları, yönetişim denetimleri ya da yalnızca birçok ayrı `cd` komutu olmadan organizasyon genelinde `git status` çalıştırmak.

`atl`'yi yalnızca KULLANMAK istiyorsan (kendi projelerine takım kurmak için) çalışma alanına ihtiyacın yoktur — [kurulum betiği](../guide/install) yeterlidir. Çalışma alanı, ekosistem tarafındaki iş içindir.

## İlk kurulum

```bash
git clone https://github.com/agentteamland/workspace.git
cd workspace
./scripts/sync.sh
```

`sync.sh`, `agentteamland/` altındaki her eş depoyu `./repos/<name>/` dizinine klonlar. İdempotent çalışır — yeniden çalıştırmak var olan klonları ileri-sarmalı çekimle günceller ve kanonik listesinde olup henüz kontrol edilmemiş depoları klonlar. Depo listesi `sync.sh` içinde elle tutulur; organizasyona yeni eklenen bir depo, sync onu alabilmeden önce oraya eklenmelidir.

Eşzamanlamadan sonra `./repos/`, v2 aktif depolarını ve arşivlenmiş v1 depolarını içerir (tarih için salt okunur olarak saklanmıştır):

```
repos/
├── atl/                       # v2 monorepo — cli + core + takımlar + belgeler
└── .github/                   # organizasyon profili

# Arşivlenmiş v1 depoları (salt okunur, tarih için saklanmış):
├── cli/                       # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── core/                      # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── brainstorm/                # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── rule/                      # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── team-manager/              # 🗄 ARCHIVED 2026-06-21 — önyükleme sarmalayıcısı; kurulum artık atl'de
├── software-project-team/     # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── design-system-team/        # 🗄 ARCHIVED 2026-06-21 — atl monoreposuna aktarıldı
├── starter-extended/          # 🗄 ARCHIVED 2026-06-21 — kalıtım v2'de kaldırıldı
├── registry/                  # 🗄 ARCHIVED 2026-06-21 — GitHub konu kataloğuyla değiştirildi
├── homebrew-tap/ scoop-bucket/ # 🗄 ARCHIVED 2026-06-21 — dağıtım artık yalnızca GitHub Releases üzerinden
└── docs/                      # 🗄 ARCHIVED 2026-06-22 — belgeler sitesi atl monoreposuna aktarıldı (docs/site/)
```

## Günlük komutlar

Çalışma alanı `./scripts/` altında üç betik ile gelir:

```bash
./scripts/sync.sh         # eksik depoları klonla; var olanları ileri-sarmalı çekimle güncelle
./scripts/status.sh       # tablolu genel görünüm — kim kirli, kim önde, kim geride
./scripts/push-all.sh     # push'lanmamış commit'lerin kuru çalıştırma listesi (gerçekten push'lamak için --force)
```

`status.sh`, her depo için tek satırlık bir tablo yazdırır — dal, önde / geride sayıları, kirli işareti. Organizasyonun mevcut durumunu bir bakışta görmek için her oturumun başında çalıştır.

`push-all.sh` varsayılan olarak kuru çalıştırma yapar — NE'nin push'lanacağını gösterir, gerçek push'lamayı yapmaz. Gerçekten push'lamak için `--force` geç. ("force" adı kuru çalıştırmayı bastırmaya işaret eder, `git push --force` değil — gerçek push olağan Git anlamlarını kullanır.)

## Bir eş depoda çalışmak

```bash
cd repos/<repo-name>
# Değişikliklerini yap, conventional-commit + branch-hygiene disiplinini izle
git checkout -b <type>/<short-description>
# ... dosyaları düzenle ...
git add <files> && git commit -m "<conventional message>"
git push -u origin <branch-name>
gh pr create
# Bakımcının inceleyip birleştirmesini bekle
```

Her eş depo kendi uzak deposu olan kendi Git klonudur. Genel üretim depolarındaki dal koruması PR akışını zorunlu kılar.

## Çalışma alanını Claude Code ile kullanmak

Çalışma alanının kökünde Claude Code aç:

```bash
cd ~/projects/my/agentteamland/workspace
claude    # ya da Claude Code'u nasıl çağırıyorsan
```

Claude Code burada başladığında kendiliğinden şunları görür:

- **`./repos/` altındaki her eş depo** doğrudan düzenleme için — ayrı `cd` gerekmez.
- **Tüm etkin beyin fırtınaları** ([brainstorm kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/brainstorm.md) gereği `CLAUDE.md`'ye kendiliğinden sabitlenmiş).
- **Çalışma alanının `CLAUDE.md` dosyası** — platform düzeyinde yönlendirme belgesi.
- **Yerleşmiş kararlar** `.atl/docs/` altında (tamamlanmış beyin fırtınalarından türeyen mimari kararlar).
- **Wiki + journal** — `.atl/wiki/` ve `.atl/journal/` içinde ([bilgi sistemi](../guide/knowledge-system) gereği).

Bu, depolar arası iş için doğal kurulumdur: Claude'un çalışma kümesi tüm organizasyondur.

## Bilgi haritası

Çalışma alanının `CLAUDE.md` dosyası, her wiki sayfasının başlığını ve özetini Claude'un bağlamına kendiliğinden yükleyen bir `<!-- wiki:index -->` işaretçi bloğu taşır. İşaretçi bloğunun nasıl çalıştığı ve neden var olduğu için bkz. [Claude Code sözleşmeleri](../guide/claude-code-conventions).

Wiki'nin kendisi (`.atl/wiki/*.md`), bakımcının depolar arası endişeler üzerinde çalışırken elinin altında bulundurması gereken platform genelindeki desenler, sözleşmeler, keşifler ve kötü desenlerin kanonik kaydıdır. Sayfalar güncel tutulur — [bilgi sistemi](../guide/knowledge-system), güncel doğru için yerine yazma biçimli, geçmiş için yalnızca eklemeli journal kullanır.

## Oturum sonu

Toparlanırken:

```bash
./scripts/status.sh        # her şeyin main'de ve temiz olduğunu doğrula
./scripts/push-all.sh      # push'lanmamış ne var, gör
```

Daha kapsamlı bir oturum sonu geçişi için [`/repo-cleanup`](https://github.com/agentteamland/workspace/blob/main/.claude/skills/repo-cleanup/SKILL.md) şunları otomatikleştirir: `/drain` → dal + commit + push + PR + auto-merge → etiket + dal budama. Çalışma alanında Claude Code'un içinden çalıştır.

## İlgili

- [`atl` CLI'yi kur](../guide/install) — yalnızca `atl`'yi KULLANMAK istiyorsan çalışma alanını atla.
- [Bilgi sistemi](../guide/knowledge-system) — çalışma alanının `.atl/` dizinindeki journal ve wiki katmanları.
