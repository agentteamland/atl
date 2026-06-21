# `atl install`

Mevcut projeye bir takım kur.

## Kullanım

```bash
atl install <team>                # idempotent, yalnızca ilk kurulumda iş yapar (atl ≥ 1.0.0)
atl install <team> --refresh      # zorla üzerine yaz (yerel değişiklikleri uyarıyla siler)
```

`<team>` şu olabilir:

- **Kayıt defterindeki kısa ad** — `software-project-team`.
- **Kayıt defteri adı + sürüm** — `software-project-team@^1.2.0` (caret, tilde ya da kesin sabitleme).
- **GitHub `owner/repo` kısa biçimi** — `agentteamland/starter-extended`.
- **Git URL'si** — `https://github.com/you/your-team.git`, `git@github.com:you/team.git`, `ssh://...`, `file:///abs/path.git`.
- **Yerel dosya sistemi yolu** — `./my-team`, `~/projects/my-team`, `/abs/path/to/team` (atl ≥ 0.1.4; yol, kökünde `team.json` ve içinde `.git/` bulunan bir dizin olmalı).

## Örnekler

### Kayıt defteri (herkese açık, doğrulanmış)

```bash
atl install software-project-team
atl install software-project-team@^1.2.0
```

### GitHub kısa biçimi

```bash
atl install agentteamland/starter-extended
```

### Tam Git URL'si (herkese açık ya da özel)

```bash
atl install https://github.com/acme/acme-starter.git
atl install git@github.com:you/private-team.git          # SSH; senin git kimlik bilgilerini kullanır
atl install https://gitea.example.com/you/team.git       # kendi barındırdığın
```

### Yerel dosya sistemi — uzak depo gerekmez (atl ≥ 0.1.4)

Özel-yerel iş akışı: takımı dizüstüne kur, hiçbir Git sunucusuna push yapmadan kendi projene yükle.

```bash
# Tek seferlik: takımı bir Git deposuna dönüştür
cd ~/projects/my-team
git init -b main && git add . && git commit -m "init"

# Herhangi bir projeye kur:
cd ~/projects/some-app
atl install ~/projects/my-team                   # mutlak yol
atl install ./my-team                            # göreli yol
atl install file:///Users/you/projects/my-team   # açık file:// URL'si
```

Üç biçim de tıpatıp aynı çalışır. Kaynak, `team.json` içeren bir dizin ve bir Git deposu olmalı (en az bir commit'i bulunan). Tüm adım adım anlatım için bkz. [Bir takım yazma](/tr/authoring/creating-a-team).

## Çoklu takım kurulumu

Birden çok takım aynı projede yan yana yaşayabilir — `atl` v0.1.2+ bunu yerel olarak destekler. Her kaynak (ajan, kural, beceri) projenin `.claude/` dizinine **kopyalanır**; `~/.claude/repos/agentteamland/{team}/` adresindeki global önbellek kaynak doğruluktur, `atl update` ise değiştirilmemiş kopyaları eşzamanlı tutar.

```bash
atl install software-project-team
atl install design-system-team

atl list
# ✓ software-project-team@1.2.1
# ✓ design-system-team@0.8.1
```

İki takım aynı adda bir öğe bildirdiğinde (örneğin ikisinde de bir `code-reviewer` ajanı varsa), en son kurulan kazanır. `atl` tek satırlık bir uyarı yazdırır:

```
⚠ overriding agent "code-reviewer" (was from team-a, now from team-b)
```

Bu, npm / pip / GNU Stow sözleşmelerini yansıtır. Bir takımı kaldırmak güvenlidir — `atl remove` o takımın kopyalanmış kaynaklarını siler ve geriye kalan takımların kopyalarını yeniden uygular. Kaldırılan takımın çakışma yoluyla "kazandığı" öğeler doğru biçimde özgün sahibine geri düşer.

## Proje-yerel kopya kurulumu (atl v1.0.0+)

Her takım kaynağı — ajan, kural, beceri — `<project>/.claude/` dizinine bir **proje-yerel kopya** olarak kurulur. `~/.claude/repos/agentteamland/{team}/` adresindeki global önbellek, makinedeki tüm projeler arasında paylaşılan kaynak doğruluktur; her proje kendine yeten kopyasını tutar.

Bu, [install-mechanism-redesign kararının](https://github.com/agentteamland/workspace/blob/main/.atl/docs/install-mechanism-redesign.md) seçtiği topolojidir; v1.0.0 öncesi karışımın (ajan ve kural için sembolik bağ, yalnızca beceri için kopya) yerine geçti. İki neden bu değişikliği zorunlu kıldı:

1. **Değişimler yerelde kalır.** `/save-learnings` ve [kendini güncelleyen öğrenme döngüsünün](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) kendiliğinden büyüyen `children/` ve `learnings/` dizinleri projenin `.claude/` içine yazar. Sembolik bağlar olsaydı bu yazımlar global önbelleği kirletir ve bir sonraki `atl update` çekiminde çakışırdı. Kopyalarla, değişimler kendisini doğuran projeye yalıtık kalır.
2. **Claude Code'un beceri yükleyicisi.** Topoloji kararından bağımsız olarak Claude Code'un proje düzeyindeki beceri yükleyicisi `.claude/skills/` altındaki sembolik bağları izlemez (üst akış sorunları [#14836](https://github.com/anthropics/claude-code/issues/14836), [#25367](https://github.com/anthropics/claude-code/issues/25367), [#37590](https://github.com/anthropics/claude-code/issues/37590)). Beceriler nasıl olsa kopyalanmak zorundaydı; v1.0.0 yeniden tasarımı bunu tüm kaynaklara genelleştirdi.

`atl update` eşzamanlama düzeneğidir — aşağıya bak.

## İdempotenlik (atl v1.0.0+) {#idempotency-atl-v100}

Zaten kurulu bir takım üzerinde `atl install <team>` komutunu ikinci kez çalıştırmak, eskiden olduğu gibi sessizce yeniden kurmaz; tek satırlık bir bilgi mesajıyla **işlem yapmaz**. v1.0.0 öncesinde her kurulum yerel düzenlemelerin üzerine sessizce yazardı; v1.0.0+ bunu yapmak için `--refresh` ister.

```bash
atl install software-project-team           # ilk kez → kurar
atl install software-project-team           # yeniden → işlem yok + bilgi satırı
atl install software-project-team --refresh # zorla yeniden kurma (yerel düzenlemeler varsa uyarır)
```

Bu, `atl install`'ı yeniden çalıştırmanın betiklerde bile güvenli olduğu anlamına gelir. Kurulu bir takıma üst akış değişikliklerini almak için `atl update` komutunu kullan (değiştirilmemiş kopyaları kendiliğinden yeniler).

## Ne olur?

1. **Çözümleme.** Handle, GitHub tabanlı index'te aranır — first-party takım monorepo alt-yoluna, third-party takım kendi bağımsız deposuna çözülür. Git URL doğrudan kullanılır.
2. **Getirme.** Takım, ref'e sabitlenmiş bir HTTP tarball olarak (bir `git` binary'si gerekmeden) yerel önbelleğe indirilir.
3. **Doğrulama.** `team.json` [şemaya](/tr/reference/schema) göre denetlenir. Geçersiz takımlar burada başarısız olur.
4. **Yazma.** Ajanlar, kurallar ve beceriler kapsamın `.claude/` dizinine **kopyalanır** — global kurulum için `~/.claude`, proje kurulumu için `<project>/.claude`. Mevcut kopyalar korunur (idempotent varsayılan); üzerine yazmak için `--refresh`.
5. **Kaydetme.** `<layer>/.atl/installed/<handle>__<name>.json` konumundaki takım başına manifest, kaynak ref'i + `atl update`'in kendiliğinden yenilemesinin ve `atl doctor`'ın bütünlük denetiminin dayandığı dosya başına SHA-256 referans hat'larını kaydeder.

## Çevrimdışı davranış

Ağ erişilemezse `atl install` paylaşılan önbelleğe geri çekilir. En son çekilen sürüm ne ise onu alırsın. CLI bunu açıkça günlüğe yazar — sessiz bayatlama olmaz.

## Sorun giderme

- **"team not found"** — handle index'te yok. `atl search` ile dene.
- **"schema validation failed"** — takımın `team.json` dosyası bozuk. Yazardan düzeltmesini iste ya da daha eski bir sürüme sabitle.
- **"invalid team slug"** — ad, güvenlik düzenli ifadesini geçemeyen bir slug'a çözüldü (örneğin `-` ile başlıyor). Takımı `owner/repo` biçiminde geç ya da bir Git URL'si kullan.

## İlgili

- [`atl search`](/tr/cli/search) — bir takımın adını bul.
- [`atl list`](/tr/cli/list) — neyin kurulu olduğunu gör.
- [`atl update`](/tr/cli/update) — önbellek çekiminden sonra değiştirilmemiş kopyaları yenile.
- [`atl remove`](/tr/cli/remove) — kaldır (etkileşimsiz çalıştırmak için `--force`).
