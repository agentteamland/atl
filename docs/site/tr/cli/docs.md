# `atl docs`

Dokümantasyon sitesini koda karşı drift için denetler — docs-correctness'ın deterministik, LLM'siz yarısı. Anlamsal yarı (prose hâlâ kodun yaptığını mı anlatıyor?) `/docs-audit` skill'inin işidir; bu komut, bir makinenin sıfır yanlış-pozitifle doğrulayabileceği her şeydir.

## Kullanım

```bash
atl docs check [--external] [--record-audit]
```

`atl docs check`, çalışma dizininden yukarı doğru yürüyerek `docs/site/.vitepress` barındıran repo'yu bulur. Böyle bir repo dışında hiçbir şey yapmaz ve 0 ile çıkar — her yerde güvenle çalıştırılabilir (pre-flight atlaması). İçeride ise tüm deterministik kontrolleri çalıştırır ve **hata seviyesinde** herhangi bir bulgu varsa sıfırdan farklı kodla çıkar (uyarılar komutu asla başarısız kılmaz).

| Bayrak | Etki |
|---|---|
| `--external` | Dış bağlantıların HTTP üzerinden çözülüp çözülmediğini de kontrol eder. Yavaş, ağ bağımlı ve geçici kesintilere duyarlı olduğundan opsiyoneldir ve yalnızca uyarı üretir. |
| `--record-audit` | Çalışma hatasızsa, mevcut commit'i son denetlenen olarak damgalar (`~/.atl/docs-audit-state.json`). `/docs-audit` backstop'u taze bir taramanın gerekip gerekmediğini bilmek için bunu okur. |

## Ne kontrol eder

Her bulgu `[FAIL|warn] kontrol · sayfa — ayrıntı` biçimindedir. Hatalar CI kapısını kırar; uyarılar yüzeye çıkar ama asla başarısız olmaz.

- **`coverage`** (FAIL) — her CLI komutunun bir `cli/<ad>.md` sayfası vardır ve her `cli/*.md` bir shipping komuta karşılık gelir; benzer şekilde her core skill ↔ `skills/<ad>.md`. Komut listesi doğrudan canlı CLI'dan gelir; böylece sayfası olmayan yeni bir komut yakalanır — elle güncellenen bir envanter tutmaya gerek yok.
- **`parity`** (FAIL) — her İngilizce sayfanın `tr/` altında bir Türkçe aynası vardır.
- **`tokens`** (FAIL) — dar bir bayat *talimat* denylist'i: emekliye ayrılan Homebrew / Scoop / winget kanalları için canlı adım olarak yazılmış kurulum komutları. Bilinçli olarak yalnızca talimat odaklı — bir kanalın emekliye ayrıldığını *anlatan* salt tarihsel bir anış işaretlenmez. Prose'daki kavram-yeniden-adlandırma drift'i bu kontrolün değil, `/docs-audit` skill'inin işidir.
- **`links`** (warn) — bir dosyaya çözülmeyen iç görece bağlantılar. Ölü bağlantılarda otorite VitePress'in kendi derlemesidir; bu, Node olmadan çalışabilen hızlı ön-izlemedir.
- **`flags`** (warn) — bir komutun her uzun bayrağı, doküman sayfasında bir yerde geçer.
- **`external`** (warn, yalnızca `--external`) — dış URL'ler `< 400` döner.

## CLI / Skill ayrımı

`atl docs check` tasarımı gereği deterministik ve sıfır yanlış-pozitiftir: yalnızca bir makinenin kanıtlayabileceği drift'i raporlar (eksik bir sayfa, olmayan bir ayna, bayat bir kurulum adımı). Muhakeme gerektiren her şey — "bu paragraf hâlâ kodun yaptığını mı anlatıyor?" — burada kapsam dışıdır ve grep-temelli, çekişmeli-doğrulanan `/docs-audit` skill'ine aittir. Bu, platformun geri kalanının izlediği aynı CLI (deterministik) / Skill (LLM) sınırıdır.

## Örnekler

Temiz bir site:

```bash
$ atl docs check
atl docs: clean
```

Drift bulundu:

```bash
$ atl docs check
  [FAIL] coverage · cli/export.md — command `atl export` has no docs page
  [warn] flags · cli/install.md — flag --force not documented
atl: 1 documentation drift item(s), 1 warning(s) — fix before shipping
```

## İlgili

- [`atl doctor`](/tr/cli/doctor) — kardeş deterministik self-heal; docs yerine kurulu asset'ler için.
- [Yayın hattı](/tr/contributing/release-pipeline) — docs-drift CI kapısının `atl docs check`'i çalıştırdığı yer.
- [CLI genel bakış](/tr/cli/overview)
