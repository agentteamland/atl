# `/create-pr`

Çalışma ağacı değişikliklerini al (commit edilmemiş ya da yakın zamanda varsayılan dala commit edilmiş), farktan uygun bir dal adı + commit mesajı + PR başlığı türet, [`/drain`](/tr/skills/drain) komutunu çalıştır ki bekleyen öğrenmeler aynı PR'ın içinde yolculuk etsin, çekişmeli (adversarial) bir AI inceleme zinciri çalıştır (genel temel + takımın bildirdiği uzmanlar + koru-eğer-çürütülemezse doğrulama geçişi), commit + push yap, bir PR aç. İsteğe bağlı olarak GitHub auto-merge düzeneğini, sınırlı bir yoklama + kendiliğinden düzeltme döngüsüyle etkinleştir. İş bitiminde kullanıcıyı daima hedef dala döndür.

Bu beceri, "bir parça işi yayımla" akışının belirlenimci hâlidir — [`branch-hygiene`](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md), [`learning-capture`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md) ve [`karpathy-guidelines`](https://github.com/agentteamland/atl/blob/main/core/rules/karpathy-guidelines.md) kurallarını uygular; böylece bunları her PR'da yeniden üretmek zorunda kalmazsın.

Global beceri olarak [atl monoreposunda](https://github.com/agentteamland/atl) yayımlanır.

## Bayraklar

| Bayrak | Varsayılan | Etkisi |
|---|---|---|
| `--auto-merge` | KAPALI | GitHub auto-merge düzeneğini etkinleştirir (`gh pr merge --auto --squash`); birleşene ya da kalıcı bir başarısızlık olana kadar yoklama + kendiliğinden düzeltme yapar. |
| `--no-review` | KAPALI (inceleme açık) | Tüm inceleme zincirini atlar (genel + her takım inceleyicisi + çekişmeli doğrulama geçişi). |
| `--no-auto-fix` | KAPALI (düzeltme açık) | Yoklama döngüsü sırasında CI / birleştirme başarısızlıklarını düzeltmeye çalışmaz; bunun yerine kullanıcıya bildirir. |
| `--no-drain` | KAPALI (drain açık) | Bekleyen öğrenmeleri bilgi tabanına katlamayı atlar. |
| `--no-docs` | KAPALI (docs açık) | Doküman sitesini değişiklikle senkron tutan docs-impact geçişini atlar. |
| `--timeout {min}` | 10 | Dakika cinsinden yoklama zaman aşımı; 1 dakikalık aralık; hem `--auto-merge` hem elle birleştirme beklemesi için geçerli. |

## Akış

Akış sıralı çalışır. Her adımın net bir önkoşulu ve ardkoşulu vardır; bir önkoşul karşılanmazsa beceri sorunu yüzeye çıkarır ve devam etmek yerine durur.

### Adım 1 — Ön denetimler

- Mevcut dizin bir Git deposunun içindedir.
- Çalışma ağacında değişiklik VARDIR ya da mevcut dalın push edilmemiş commit'leri vardır (ikisi de yoksa: "Yapılacak bir şey yok — çalışma ağacı temiz ve dal güncel").
- Deponun varsayılan dalı (`main`/`master`) belirlenir.

### Adım 2 — Hedef dalı belirle

"Hedef dal", PR'ın birleşeceği VE iş bitiminde kullanıcının döneceği daldır.

- **Varsayılan daldaysan** → hedef = varsayılan dal.
- **Varsayılan dışı bir daldaysan** → `AskUserQuestion` üç seçenekle: üst dal (kendiliğinden algılanır), varsayılan dal ya da serbest metinli Other.

### Adım 3 — Dal adı + commit mesajını üret

Stage'lenmiş + stage'lenmemiş + izlenmeyen değişiklikleri çözümle:

- **Tür** — `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf` arasından biri (farktan sezgisel olarak çıkarılır: yeni ajan/beceri/kural ya da özellik → `feat`; hata düzeltme dili → `fix`; yalnızca `*.md` → `docs`; vb.).
- **Kapsam** — değişikliği kapsayan en belirgin kapsam (beceri adı, kural adı, ajan adı, CLI komutu, depo alanı).
- **Slug** — kebab-case, ≤ 50 karakter, ASCII.

Çıktılar:

- **Dal adı** — `{type}/{slug}` (örneğin `feat/create-pr-skill`, `fix/install-404`, `docs/translate-tr-en`).
- **Commit konusu** — `{type}({scope}): {tek satırlık özet}`, 70 karakterin altında.
- **Commit gövdesi** — değişikliği anlatan 2-4 madde.

Beceri kullanıcıya ad onayı için **sormaz** — adları üretir ve devam eder.

### Adım 4 — Bekleyen öğrenmeleri drain et (`--no-drain` verilmedikçe)

[`/drain`](/tr/skills/drain) komutunu çağırır; böylece oturum sırasında yakalanan her öğrenme aynı PR'ın içinde yolculuk eder:

- `/drain`, kalıcı öğrenme kuyruğunu okur, bekleyen her öğeyi wiki / journal / ajan bilgi tabanına yönlendirir ve onaylar (ack).
- Boş kuyruk bir işlem-yok (no-op) durumudur.
- `/drain` kurulu değilse adım tek satırlık bir bildirimle atlanır — beceriyi asla başarısız kılmaz.

v2'nin işaretçisi düz metindir (`<!-- learning: serbest metin -->`); ayrı bir doküman-etkisi / doküman-taslağı hattı yoktur — her öğrenmenin nereye ait olduğunu `/drain` çıkarır. Bkz. [`atl learnings`](/tr/cli/learnings) ve [`/drain` becerisi](/tr/skills/drain).

### Adım 4.5 — Docs-impact geçişi (`--no-docs` verilmedikçe)

Doküman sitesini değişiklikle aynı hizada tutar — [docs-sync](/tr/cli/docs)'in değişiklik-anı katmanı, böylece drift hiç oluşmaz. Ön-kontrol yapar ve **ikisi de** geçerli olmadıkça ucuzca atlar: repo'nun bir doküman sitesi vardır (`docs/site/.vitepress`) ve diff doc-etkili bir yüzeye dokunur (`cli/`, `core/`, `docs/` ya da bir komut/skill/rule/kavram).

Geçerli olduğunda:

1. **Önce deterministik** — [`atl docs check`](/tr/cli/docs)'i çalıştırır ve her FAIL'i düzeltir (eksik sayfa, olmayan TR aynası, bayat kurulum talimatı). Mekanik, sıfır yanlış-pozitif.
2. **Anlamsal, grep-temelli** — diff'in değiştirdiğini olası biçimde anlatan her sayfa için, onu yeni koda karşı okur ve drift iddia etmeden önce kaynağı verbatim alıntılar (~%40 halüsinasyon koruması). Etkilenen sayfaları günceller (EN + TR aynası).
3. Doküman düzenlemelerini **aynı PR**'da yolculuk edecek şekilde stage'ler — docs ve kod atomik olarak iner.

Sıradan diff'ler hiçbir şeye mal olmaz (ön-kontrol onları atlar). Bu, deterministik [docs CI kapısının](/tr/cli/docs) yapamadığı LLM yarısıdır. Tüm-site backstop için bkz. [`/docs-audit`](/tr/skills/docs-audit).

### Adım 5 — İnceleme zinciri (`--no-review` verilmedikçe)

Üç katman, sıralı olarak çalıştırılır — iki bulucu, sonra bir çekişmeli doğrulayıcı:

**5a — Genel inceleyici (daima)**

Stage'lenmiş fark üzerinde taze bağlamlı bir alt ajan çağırır (taze bağlam, böylece inceleme farkı yazan modelden etkilenmez); dört Karpathy ilkesiyle istemlenir:

- Kodlamadan Önce Düşün (varsayımlar açık mı?).
- Önce Sadelik (fazla mühendislik var mı?).
- Cerrahi Değişiklikler (geçerken yapılan düzenlemeler? yetimler?).
- Hedef Odaklı Yürütme (hedefe karşı doğrulanıyor mu? başarı ölçütleri?).

Buna ek olarak genel kod kalitesi (adlandırma, kapsam kayması, güvenlik kokuları — loglarda sırlar, enjeksiyon, sabit kodlanmış kimlik bilgileri — ölü kod, test kapsamı). Sonuç 🔴 sorunlar / 🟡 endişeler / 🟢 sorunsuz olarak raporlanır.

**5b — Takım uzmanları (kurulu takım başına)**

Kurulu her takım için (önce `.claude/agents/`, sonra `~/.claude/agents/` altına bakılır — proje global'i gölgeler), beceri `team.json` dosyasındaki `capabilities.review` alanını okur:

- Bir ajan adı veriyorsa (örneğin `capabilities.review: "code-reviewer"`), o takım ajanı aynı farka karşı çalıştırılır ve alana özgü bir inceleme üretir.
- Bildirilmemişse takım atlanır — 5a, platform genelindeki temeldir.

**5c — Çekişmeli doğrulama (daima)**

Bulucular yazara-yakın iyimserlerdir; ham bulguları doğrudan sunulmaz. Taze bağlamlı tek bir alt ajan, **bütünleştirilmiş 5a + 5b bulguları** üzerinde çalışır (tüm farkı yeniden değil, bulgu listesini) — iki görevle:

- **Kanıt kapısı** — her bulgu somut kanıt göstermeli (bir `dosya:satır`, bir grep örüntüsü ya da başarısız bir test/komut). Hiçbirini göstermeyen bulgu düşürülür, gösterilmez — [`/docs-audit`](/tr/skills/docs-audit)'in "verbatim alıntı olmadan iddia yok" disiplini, kod incelemesine uygulanmış hâli.
- **Koru-eğer-çürütülemezse** — hayatta kalan her bulgu için ajan, alıntılanan satırları okur ve çürütmeye çalışır; yalnızca hayatta kalan bulgular korunur, önem yeniden tartılır. 5a ile bir 5b uzmanı önemde anlaşamadığında bu geçiş hakemdir.

Bu, tüm farkın ikinci bir incelemesi değil, küçük bir bulgu listesi üzerinde tek ek ajandır. Yalnızca hayatta kalan, kanıtlı bulgular gösterilir; kaçının düşürüldüğü ya da çürütüldüğü sayısıyla birlikte.

Bütünleştirilmiş rapor kullanıcıya gösterilir: devam et / iptal et / düzenle.

### Adım 6 — Commit + push

```bash
git checkout -b {branch-name}
git add -A
git commit -m "{commit-subject}

{commit-body}"

git push -u origin {branch-name}
```

### Adım 7 — PR aç

```bash
gh pr create \
  --base {target-branch} \
  --title "{commit-subject}" \
  --body "<Özet maddeleri + Test planı kontrol listesi>"
```

Beceri `--assignee` ya da `--reviewer` **geçirmez**.

### Adım 8 — `--auto-merge` (yalnızca bayrak verilmişse)

```bash
gh pr merge {N} --auto --squash
```

Bu, **tüm beceri kümesindeki tek izin verilen birleştirme çağrısıdır.** Hemen birleştirmez — GitHub zorunlu denetimleri bekler ve sonra birleştirir; böylece dal koruması kapısı korunur. Kullanıcı bayrağı geçirerek bu işe açıkça katılmıştır.

### Adım 9 — Yoklama + kendiliğinden düzeltme döngüsü (yalnızca `--auto-merge` verildiyse)

PR durumunu 1 dakikalık aralıklarla, en çok `{timeout}` deneme boyunca (varsayılan 10) yoklar. Durum makinesi:

| Durum | Eylem |
|---|---|
| `MERGED` | Başarı — iş bitimine geç. |
| `CLOSED` | Kullanıcı birleştirmeden kapattı — temiz çık, iş bitimi yok. |
| `*CLEAN` / `*HAS_HOOKS` | Sağlıklı durum, yalnızca denetimleri bekliyor — yoklamaya devam. |
| `*BLOCKED` / `*UNSTABLE` / `*DIRTY` / `*BEHIND` | CI başarısızlığı veya birleştirme çakışması — `handle_failure`. |

#### `handle_failure` sınıflandırması

**Kapsam içinde (kendiliğinden düzeltme denenir, en çok 3):**

- Birleştirme çakışmaları — en güncel hedefi çek, üç yönlü birleştirme dene.
- Lint / biçim başarısızlıkları — projenin biçimleyicisini çalıştır (kendiliğinden algılanır: `package.json` içindeki `scripts.lint`, `.prettierrc`, `gofmt`, `cargo fmt` vb.).
- Önemsiz tür hataları / eksik içe aktarımlar — derleyicinin önerdiği düzeltmeleri uygula.

**Kapsam dışı (bildir ve dur):**

- Gerçek test başarısızlıkları (önermeler, mevcut testlerde gerileme).
- Önemsiz olmayan yapı hataları.
- Altyapı ya da CI yapılandırma sorunları.
- Eksik zorunlu incelemeler (insan inceleyiciler engelliyor).

Kapsam içi 3 düzeltme denemesinden sonra beceri durur ve raporlar.

### Adım 10 — Elle birleştirme yoklaması (yalnızca `--auto-merge` VERİLMEDİYSE)

Beceri yine de birleştirme için yoklama yapar — kullanıcı `{timeout}` dakika içinde elle birleştirebilir. Aynı MERGED / CLOSED / zaman aşımı çıkışları geçerlidir.

### Adım 11 — İş bitimi (evrensel)

Yalnızca PR başarıyla birleştiyse erişilir:

```bash
git checkout {target-branch}
git pull
```

Kullanıcı beceriyi hedef dalda, birleştirilmiş değişiklik dahil edilmiş hâlde, bir sonraki göreve hazır olarak bitirir.

### Adım 12 — Son rapor

```
✅ /create-pr complete
   Branch:      feat/create-pr-skill
   PR:          https://github.com/.../pull/N
   Review:      generic + 1 team reviewer (acme/example-team) + adversarial verify
                3 issues, 1 concern surviving (2 dropped: no evidence), all addressed
   Drain:       /drain ran — 2 wiki pages updated, 1 journal entry
   Auto-merge:  enabled, merged after 4 min (1 auto-fix: prettier formatting)
   End-of-work: returned to main, pulled latest
```

## Önemli kısıtlar

1. **Asla doğrudan birleştirme.** Bu beceri `gh pr merge --auto --squash` (auto-merge etkinleştirme) komutunu yalnızca `--auto-merge` verildiğinde kullanır. Hemen birleştiren bir `gh pr merge --squash`/`--merge`/`--rebase` (`--auto` olmadan) **daima yasaktır** — auto-merge zorunlu-denetim kapısını korur ve kullanıcı bayrağı yazarak bu işe katılmıştır.
2. **İdempotent drain.** Burada `/drain`'i çalıştırmak güvenlidir — yalnızca onaylanmamış kuyruk girdilerini işler.
3. **Başlamadan önce dal hijyeni.** Yeni dalı türetmeden önce beceri yerel varsayılan dalın `origin` ile güncel olduğunu doğrular; geride kalmışsa önce ileri-sarma yapar ([`branch-hygiene`](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md) gereği).
4. **Sessiz, kısmi başarısızlık yok.** Herhangi bir adım başarısız olursa beceri durur ve raporlar — kullanıcı nerede olduğunu daima bilir.

## İlgili

- [`/drain`](/tr/skills/drain) — bekleyen öğrenmeleri bilgi tabanına katlamak için Adım 4'te çağrılır.
- [`branch-hygiene` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md) — dallanmadan önce temel dalı güncel tut.
- [`karpathy-guidelines` kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/karpathy-guidelines.md) — inceleme isteminin temeli.

## Kaynak

- Belirtim: [core/skills/create-pr/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/create-pr/SKILL.md).
