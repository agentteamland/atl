# `atl install`

GitHub tabanlı katalogdan bir takımı çözümle ve bir kapsama kur.

## Kullanım

```bash
atl install <handle>/<team>              # yayıncının varsayılan kapsamında kur
atl install <handle>/<team> --global     # kullanıcı-global kapsam (her proje)
atl install <handle>/<team> --project    # proje kapsamı (yalnızca bu proje)
```

`<handle>/<team>`, katalog referansıdır — handle takımın GitHub sahibidir (sahiplik, yazarlıktır) ve team o handle içindeki addır. `@version` sabitleme, Git URL'si ya da yerel dosya sistemi yolu yoktur: install her zaman katalog üzerinden çözümlenir. Bir referans bulmak için [`atl search`](/tr/cli/search) komutunu kullan.

```bash
atl install acme/example-team
```

`--global` ve `--project` birbirini dışlar. İkisi de verilmezse takım, yayıncısının bildirdiği kapsamda kurulur (aşağıdaki [Kapsam](#kapsam) bölümüne bak).

## Ne olur?

1. **Çözümleme.** `<handle>/<team>` referansı, GitHub'daki [`atl-team`](https://github.com/topics/atl-team) konusuyla etiketlenmiş herkese açık depolardan üretilen GitHub tabanlı katalogda aranır. Katalog, `~/.atl/index.json` önbelleğinden çevrimdışı-önce (offline-first) çözümlenir; bu nedenle arama asla ağı beklemez. Monorepo alt yolundan yayımlanan bir takım o alt yola, bağımsız bir takım kendi deposunun köküne çözümlenir.
2. **Getirme.** Takımın kaynağı, **`git` binary'si gerekmeden** doğrudan GitHub'dan ref'e sabitlenmiş tek bir HTTPS tarball olarak geçici bir dizine indirilir; kurulum tamamlanır tamamlanmaz bu dizin silinir. Diskte kalıcı bir klon önbelleği yoktur.
3. **Okuma.** `team.json` ayrıştırılır. Doğrulama minimaldir: geçerli JSON olmalı ve bir `name` alanı içermelidir. JSON-Schema doğrulayıcısı yoktur — CLI'nin tam olarak neyi denetlediği için [Şema](/tr/reference/schema), tam alan sözleşmesi için [`team.json`](/tr/authoring/team-json) sayfalarına bak.
4. **Varlıkları yaz.** Takımın `agents/`, `skills/` ve `rules/` ağaçları doğrudan kapsamın Claude Code dizinine kopyalanır — global kurulum için `~/.claude`, proje kurulumu için `<project>/.claude`. Depodan başka hiçbir şey (`team.json`, `README`, `LICENSE`) kopyalanmaz.
5. **Manifest kaydet.** Kapsam katmanının `.atl/` dizinine `<layer>/.atl/installed/<handle>__<name>.json` konumunda takım başına bir manifest yazılır. Çözümlenen kaynak ref'ini ve kopyalanan her dosyanın SHA-256 temel değerini kaydeder; `atl update`'in yenileme ve `atl doctor`'ın bütünlük denetimi bu bilgilere dayanır.
6. **Otomasyon hook'larını bağla.** Otomasyon hook'ları (`SessionStart → atl session-start`, `UserPromptSubmit → atl tick` ve `PreToolUse (Bash|Edit|Write) → atl guard`) kurulumun zorunlu bir parçası olarak Claude Code'a eklenir — otomasyon varsayılan olarak açıktır, isteğe bağlı değil. Hook bağlama başarısız olursa uyarı olarak gösterilir; kurulum başarısız olmaz.
7. **Platform çekirdeğini yansıt.** Platformun kendi kuralları ve becerileri (`/drain`, `/create-pr`, `/brainstorm` ve diğerleri) binary içinde taşınır ve global katmana yansıtılarak her projede kullanılabilir hale getirilir. En iyi çaba esasıyla çalışır; bir hata fatal değildir.
8. **Bir proje `CLAUDE.md`'si oluştur.** Projede `CLAUDE.md` yoksa kurulum project-tier başlangıcını düşürür (bkz. [`atl init`](/tr/cli/init)) ki `/brainstorm` ve `/drain` bloklarının bir evi olsun. Yalnızca-yoksa — zaten sahip olduğun dosyaya asla dokunulmaz — ve en iyi çaba esasıyla, yani bir hata kurulumu asla başarısız kılmaz.

Başarıda CLI şunu yazdırır:

```
atl: installed <handle>/<name>@<version> at <scope> scope
```

`<scope>` değeri `global`, `project` ya da `both` olabilir.

## Kapsam {#kapsam}

Bir takım iki katmandan birinde yaşar:

- **global** — varlıklar `~/.claude` altında, ATL durumu `~/.atl` altında. Makinedeki her projede kullanılabilir.
- **project** — varlıklar `<project>/.claude` altında, ATL durumu `<project>/.atl` altında. Yalnızca o projede kullanılabilir.

Her takımın yayıncısı `team.json`'da varsayılan bir kapsam bildirir — `project` (varsayılan), `global` ya da `both`. Bunu kurulum sırasında `--global` veya `--project` ile geçersiz kılabilirsin; geçersiz kılma her zaman kazanır. `both` kurulumu **iki** katmana da yazar.

Aynı yetenek her iki katmanda da mevcutsa **proje katmanı global'i gölgeler** — en yakın kazanır; bu zihniyet modeli Claude Code'un kendi `CLAUDE.md` katmanlamasıyla aynıdır. Tam kapsam eksenini [Kavramlar](/tr/guide/concepts#scope-global-and-project) sayfasında incele.

```bash
atl install acme/example-team            # yayıncı varsayılanı (project)
atl install acme/example-team --global   # makinedeki her proje
```

## Kurulum manifestası

Her kurulum, `<layer>/.atl/installed/<handle>__<name>.json` konumunda kapsam başına takım başına bir JSON dosyası yazar (`<layer>`, global için `~/.atl`, proje için `<project>/.atl`'dir). Şunları kaydeder:

- `schemaVersion`, `handle`, `name`, `version` ve geçerli `scope`,
- `source` — kurulumun çözümlendiği `repo`, `subpath` ve `ref`; tam olarak getirilen byte'lara sabitlenmiş,
- `installedAt`,
- `files` — kopyalanan her dosyanın (`.claude` dizinine göreli yol) kurulum anındaki SHA-256 değeriyle eşleşmesi.

`files` haritası çift amaçlıdır: `atl update`, geçerli byte'ları bununla karşılaştırarak düzenlemelerini upstream değişikliklerden ayırt eder (bu sayede güncellemeler yerel değişikliklerinizi asla ezmez); `atl doctor` ise silinen ya da bozulan bir kopyayı tespit edip geri yüklemek için bu bütünlük kümesini kullanır.

## Bir projede birden çok takım

Bir projede birkaç takım birlikte yaşayabilir — her kurulum varlıklarını aynı `.claude/` dizinine kopyalar ve kendi manifestasını yazar. Kurulu iki takım aynı adda bir varlık getirirse en son yazılan kopya diskte kalır. Bir takımı [`atl remove`](/tr/cli/remove) ile kaldır; yalnızca o takımın manifesta kayıtlı dosyaları silinir.

Bir takım ayrıca `team.json`'da başka takımları `dependencies` (bağımlılıklar) olarak bildirebilir; bunlar birlikte kurulur. Bağımlılık alanı için [`team.json`](/tr/authoring/team-json) sayfasına bak.

## Sorun giderme

- **`team … not found in index`** — referans katalogda yok. [`atl search`](/tr/cli/search) ile kontrol et. Katalog, `atl-team` konusuyla etiketlenmiş herkese açık depolardan üretilir; yeni eklenen bir takım henüz listelenmemiş olabilir.
- **`invalid team reference`** — argüman `<handle>/<team>` biçiminde değil (her iki parça da gerekli).
- **`fetch … HTTP 404`** — takımın kaynak deposu ya da ref'i erişilemez durumda. Tarball getirme ağa ihtiyaç duyar; katalog çözümlemenin aksine çevrimdışı geri düşme yoktur.
- **`team ships no installable assets`** — çözümlenen takımda `agents/`, `skills/` veya `rules/` dizini yok.
- **`team.json has no name`** — takımın `team.json` dosyası hatalı biçimlendirilmiş. Yazardan düzeltmesini iste.

## İlgili

- [`atl search`](/tr/cli/search) — bir takımın `<handle>/<team>` referansını bul.
- [`atl list`](/tr/cli/list) — kurulu takımları kapsama göre gruplandırılmış gör.
- [`atl update`](/tr/cli/update) — kurulu takımları yenile; değiştirilmemiş kopyalar yerinde güncellenir, yerel düzenlemeler korunur.
- [`atl remove`](/tr/cli/remove) — bir takımı kapsamdan kaldır.
- [`atl setup-hooks`](/tr/cli/setup-hooks) — kurulumun otomatik olarak bağladığı otomasyon hook'ları.
- [Kavramlar](/tr/guide/concepts#scope-global-and-project) — global/proje kapsam ekseni.
