# `/create-pr`

Çalışma ağacı değişikliklerini al (commit edilmemiş ya da yakın zamanda varsayılan dala commit edilmiş), farktan uygun bir dal adı + commit mesajı + PR başlığı türet, [`/drain`](/tr/skills/drain) komutunu çalıştır ki birikmiş deneyim aynı PR'ın içinde yolculuk etsin, AI inceleme zincirini çalıştır (genel temel + takım tarafından bildirilen uzmanlar), commit + push yap, bir PR aç. İsteğe bağlı olarak GitHub auto-merge düzeneğini, sınırlı bir yoklama ve kendiliğinden düzeltme döngüsüyle etkinleştir. İş bitiminde kullanıcıyı daima hedef dala döndür.

Bu beceri, "bir parça işi yayımla" akışının belirlenimci hâlidir — `team-repo-maintenance`, `branch-hygiene`, `learning-capture`, `docs-sync` ve `karpathy-guidelines` ile tanımlı disiplinleri tüketir; kullanıcı bunları her PR'da yeniden üretmek zorunda kalmaz.

Global beceri olarak [atl monoreposunda](https://github.com/agentteamland/atl) yayımlanır.

## Bayraklar

| Bayrak | Varsayılan | Etkisi |
|---|---|---|
| `--auto-merge` | KAPALI | GitHub auto-merge düzeneğini etkinleştirir (`gh pr merge --auto --merge`); birleşene ya da kalıcı bir başarısızlık olana kadar yoklama + kendiliğinden düzeltme yapılır. |
| `--no-review` | KAPALI (inceleme açık) | Tüm inceleme zincirini atlar (genel + her takım inceleyicisi). |
| `--no-auto-fix` | KAPALI (düzeltme açık) | Yoklama döngüsü sırasında CI / birleştirme başarısızlıklarını düzeltmeye çalışmaz; bunun yerine kullanıcıya bildirir. |
| `--no-learning` | KAPALI (öğrenme açık) | `/drain` ve doc-impact hattını atlar. |
| `--timeout {min}` | 10 | Dakika cinsinden yoklama zaman aşımı; 1 dakikalık aralık; hem `--auto-merge` hem elle birleştirme beklemesi için geçerli. |

## Akış

Akış sıralı çalışır. Her adımın net bir önkoşulu ve ardkoşulu vardır; bir önkoşul karşılanmazsa beceri sorunu yüzeye çıkarır ve devam etmek yerine durur.

### Adım 1 — Ön denetimler

- Mevcut dizin bir Git deposunun içindedir.
- Çalışma ağacında değişiklik VARDIR ya da mevcut dalın push edilmemiş commit'leri vardır.
- Deponun varsayılan dalı (`main`/`master`) belirlenir.

### Adım 2 — Hedef dalı belirle

"Hedef dal", PR'ın birleşeceği VE iş bitiminde kullanıcının döneceği daldır.

- **Varsayılan daldaysan** → hedef = varsayılan dal.
- **Varsayılan dışı bir daldaysan** → `AskUserQuestion` üç seçenekle: üst dal (kendiliğinden algılanır), varsayılan dal ya da serbest metinli Other.

### Adım 3 — Dal adı ve commit mesajını üret

Stage'lenmiş + stage'lenmemiş + izlenmeyen değişiklikleri çözümle:

- **Tür** — `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `style` arasından (farktan sezgisel olarak çıkarılır: `skills/agents/rules/` altında yeni dosya → `feat`; hata düzeltme dili → `fix`; yalnızca `*.md` → `docs`; vb.).
- **Kapsam** — değişikliği kapsayan en belirgin kapsam (beceri adı, kural adı, ajan adı, CLI komutu, depo alanı).
- **Slug** — kebab-case, ≤ 50 karakter, ASCII.

Çıktılar:

- **Dal adı** — `{type}/{slug}` (örneğin `feat/create-pr-skill`, `fix/winget-403`, `docs/translate-trk-en`).
- **Commit konusu** — `{type}({scope}): {tek satırlık özet}`, 70 karakterin altında.
- **Commit gövdesi** — değişikliği anlatan 2-4 madde. Takım deposu bağlamında çağrılıyorsa son satır bir "Discovered via" bağlamıdır ([team-repo-maintenance §3](https://github.com/agentteamland/atl/blob/main/core/rules/team-repo-maintenance.md) gereği).

Beceri kullanıcıya ad onayı için **sormaz** — adları üretir ve devam eder.

### Adım 4 — Öğrenmeleri kaydet (`--no-learning` verilmedikçe)

[`/drain`](/tr/skills/drain) komutunu elle kipte çalıştırır (canlı konuşmayı çözümler):

- Wiki / journal / ajan children / beceri learnings güncellemelerini yazar (proje-yerel).
- Her `<!-- learning doc-impact: readme/docs/breaking -->` işaretçisi için bir doküman taslağı hazırlar.
- Her taslak kullanıcıya kabul / reddet / düzenle için satır içi sunulur:

```
📝 Doc draft for README.md:
<diff>
Accept? (y/n/edit)
```

Kabul edilen taslaklar stage'lenir; reddedilen taslaklar atılır.

### Adım 5 — İnceleme zinciri (`--no-review` verilmedikçe)

İki katman, sıralı olarak çalıştırılır:

**5a — Genel inceleyici (daima)**

Karpathy temelli bir inceleme istemiyle taze bağlamlı bir alt ajan (`subagent_type: general-purpose`) çağırır:

- Kodlamadan Önce Düşün (varsayımlar açık mı?).
- Önce Sadelik (fazla mühendislik var mı?).
- Cerrahi Değişiklikler (geçerken yapılan düzenlemeler? yetimler?).
- Hedef Odaklı Yürütme (hedefe karşı doğrulanıyor mu? başarı ölçütleri?).

Buna ek olarak genel kod kalitesi (adlandırma, kapsam kayması, güvenlik kokuları, ölü kod, test kapsamı). Sonuç 🔴 Sorunlar / 🟡 Endişeler / 🟢 Sorunsuz görünüyor olarak raporlanır.

**5b — Takım inceleyicileri (kurulu takım başına)**

Kurulu her takım için beceri `team.json` dosyasını okur ve `capabilities.review` alanına bakar:

- Bildirilmişse (örneğin `capabilities.review: "code-reviewer"`), adı geçen takım ajanı aynı farka karşı çalıştırılır ve alana özgü bir inceleme üretir.
- Bildirilmemişse sessizce atlanır — takım başına yedek yoktur. Genel inceleyici platform genelinde temeldir.

Bütünleştirilmiş rapor kullanıcıya gösterilir. Devam et / iptal et / düzenle.

### Adım 6 — Commit + push

```bash
git checkout -b {branch-name}
git add -A
git commit -m "{commit-subject}

{commit-body}

{discovered-via-line if applicable}"

git push -u origin {branch-name}
```

### Adım 7 — PR aç

```bash
gh pr create \
  --base {target-branch} \
  --title "{commit-subject}" \
  --body "..."
```

Gövdede Summary, Discovered via, Sürüm artırımı (uygulanabilirse) ve Test plan bulunur. [team-repo-maintenance §4](https://github.com/agentteamland/atl/blob/main/core/rules/team-repo-maintenance.md) gereği beceri `--assignee` ya da `--reviewer` **geçirmez**.

### Adım 8 — `--auto-merge` etkinleştirme (yalnızca bayrak verilmişse)

```bash
gh pr merge {N} --auto --merge
```

Bu, **tüm beceri kümesindeki tek izin verilen birleştirme çağrısıdır.** Hemen birleştirmez — GitHub zorunlu denetimleri bekler ve sonra kendiliğinden birleştirir. Dal korumasının denetim kapısı korunur.

### Adım 9 — Yoklama + kendiliğinden düzeltme döngüsü (yalnızca `--auto-merge` verildiyse)

PR durumunu 1 dakikalık aralıklarla, en çok `{timeout}` deneme boyunca (varsayılan 10) yoklar. Durum makinesi:

| Durum | Eylem |
|---|---|
| `MERGED` | Başarı — iş bitimine geç. |
| `CLOSED` | Kullanıcı birleştirmeden kapattı — temiz çık, iş bitimi yok. |
| `*CLEAN` / `*HAS_HOOKS` | Sağlıklı durum, yalnızca denetimleri bekliyor — yoklamaya devam. |
| `*BLOCKED` / `*UNSTABLE` / `*DIRTY` / `*BEHIND` | CI başarısızlığı veya birleştirme çakışması — `handle_failure`. |

#### `handle_failure` sınıflandırması

**Kapsam içinde (kendiliğinden düzeltme denenir):**

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
git pull origin {target-branch}
```

Kullanıcı beceriyi hedef dalda, birleştirilmiş değişiklik dahil edilmiş hâlde, bir sonraki göreve hazır olarak bitirir.

### Adım 12 — Son rapor

```
✅ /create-pr complete
   Branch:      feat/create-pr-skill
   PR:          https://github.com/.../pull/N
   Review:      generic + 1 team reviewer (software-project-team)
                3 issues, 1 concern, all addressed
   Learnings:   /drain ran — 2 wiki pages updated, 1 README draft accepted
   Auto-merge:  enabled, merged after 4 min (1 auto-fix: prettier formatting)
   End-of-work: returned to main, pulled latest
```

## Önemli kısıtlar

1. **Asla doğrudan birleştirme.** Bu beceri `gh pr merge --auto --merge` (auto-merge etkinleştirme) komutunu yalnızca `--auto-merge` bayrağı verildiğinde kullanır. Doğrudan birleştirme (`--auto` olmadan `--merge`/`--squash`/`--rebase`) **daima yasaktır** — bkz. [team-repo-maintenance "PR merge discipline"](https://github.com/agentteamland/atl/blob/main/core/rules/team-repo-maintenance.md). Kullanıcı bayrağı yazarak auto-merge'i açıkça yetkilendirmiştir; bu belgelenmiş istisnadır.
2. **Discovered-via bağlamı.** Paylaşılan / takım deposundan çağrıldığında beceri team-repo-maintenance disiplinine uyar: PR gövdesine "Discovered via" eklenir, sürüm artırılır, conventional commit kullanılır. Algılama: uzak URL'nin bilinen bir paylaşılan depo desenine (örneğin `agentteamland/` altında) uyması.
3. **İdempotent drain.** Burada `/drain`'i yeniden çalıştırmak güvenlidir — yalnızca onaylanmamış kuyruk girdilerini işler.
4. **team.json doğrulaması.** Stage'lenmiş fark bir `team.json` dosyasına dokunuyorsa, push'tan önce beceri dosyanın ayrıştırılabilir olduğunu, bir `name` alanı içerdiğini ve bildirilen tüm varlıkların diskte var olduğunu doğrular.
5. **Başlamadan önce dal hijyeni.** Yeni dalı türetmeden önce beceri yerel varsayılan dalın `origin` ile güncel olduğunu doğrular. Değilse önce ileri-sarma yapar.
6. **Sessiz, kısmi başarısızlık yok.** Herhangi bir adım başarısız olursa beceri durur ve raporlar.

## İlgili

- [`/drain`](/tr/skills/drain) — Adım 4'te çağrılır.
- [team-repo-maintenance kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/team-repo-maintenance.md) — paylaşılan depolar için yönetişim.
- [karpathy-guidelines kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/karpathy-guidelines.md) — inceleme isteminin temeli.

## Gelecek evrim

- **Alana duyarlı inceleme yönlendirmesi** — her takım ajanı `domains: ["*.tsx", ...]` glob'u bildirir; beceri farkın dosya türlerini eşler ve yalnızca ilgili ajanları çağırır.
- **Paralel takım incelemesi** — takım inceleyicileri eş zamanlı çalıştırılır.
- **Kendiliğinden düzeltme kapsamının genişletilmesi** — kapsam, testin aynı farkta eklendiği test başarısızlıklarına kadar genişletilir.

## Kaynak

- Belirtim: [core/skills/create-pr/skill.md](https://github.com/agentteamland/atl/blob/main/core/skills/create-pr/skill.md).
