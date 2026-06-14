# Yönetişim

Paylaşılan `agentteamland/` depolarının nasıl yönetildiği: GitHub düzeyinde dal koruması artı ajan düzeyinde [team-repo-maintenance kuralı](/tr/authoring/team-repo-maintenance). Birlikte, paylaşılan bir depodaki her değişikliğin sürüm artırımı, conventional commit, "Discovered via" bağlamı taşımasını ve bir PR'dan geçmesini güvence altına alırlar — anımsamaya ya da iyi niyete dayanmadan.

Bu, üst düzey "politika" görünümüdür. Yordamsal ayrıntı (PR'ın gerçekte nasıl açılacağı, commit mesajının nasıl görüneceği) için bkz. [`team-repo-maintenance`](/tr/authoring/team-repo-maintenance).

## İki katman

| Katman | Ne | Neden |
|---|---|---|
| **Dal koruması** | Sunucu tarafında kötü biçimi reddeder. | Güvenlik ağıdır — kuralı anımsamaya bağlı değildir. |
| **`team-repo-maintenance` kuralı** | Yerelde iyi biçim üretir. | Yöntemdir — Claude'a PR'a hazır bir değişikliği nasıl yazacağını söyler. |

İkisi tek başına yeterli değildir:

- **Kural olmadan koruma** → Claude doğrudan push yapmaya çalışır, git reddeder, kullanıcı her seferinde nedenini ayıklamak ve Claude'a PR akışını öğretmek zorunda kalır.
- **Koruma olmadan kural** → Claude kuralı izler, ta ki izlemediği güne kadar; kaçırılan ilk adım `main` dalına kötü bir commit yayımlar.

Birlikte: kural, commit'i doğru kurar; koruma, ne olursa olsun PR'dan geçtiğini güvence altına alır.

## Dal koruması — 12 üretim deposuna uygulanır

GitHub dal koruması kuralları, `agentteamland/` organizasyonundaki her herkese açık üretim deposuna uygulanır:

**Korunan** (PR zorunlu, doğrudan push reddedilir):

- Kod depoları: `core`, `cli`, `docs`, `brainstorm`, `rule`, `team-manager`, `registry`, `create-project`, `workspace`, `.github`.
- Takım depoları: `software-project-team`, `design-system-team`, `starter-extended`.

**Korunmayan** (goreleaser kendiliğinden push yapar; koruma yayım hattını bozardı):

- Yayım hattı depoları: `homebrew-tap`, `scoop-bucket`, `winget-pkgs`.

### Ayarlar

| Ayar | Değer | Neden |
|---|---|---|
| `required_approving_review_count` | `0` | Tek bakımcı — PR, dış bir kapı olarak değil, ritüel ve denetim izi olarak gereklidir. (İkinci bir bakımcı katılırsa `1`'e çıkar.) |
| `enforce_admins` | `false` | Yönetici (bakımcı), gerçek acil durumlarda atlatabilir (yayım hattı kırılması, ivedi geri alım). |
| `allow_force_pushes` | `false` | Tarih yeniden yazılmaz. |
| `allow_deletions` | `false` | Dal silinmez. |

GitHub dal koruması API'si üzerinden uygulanır. Uygulandıktan sonra `main` dalına yapılan her doğrudan push sunucu düzeyinde reddedilir — Claude'un takdirine kalmaz.

## Kural — beş sabit adım

[`team-repo-maintenance`](/tr/authoring/team-repo-maintenance) kuralı, Claude (ya da herhangi biri) önbelleklenmiş paylaşılan bir depoyu düzenlediğinde beş adımı zorunlu kılar:

0. **PR açmadan önce `/save-learnings` komutunu çalıştır** — yayımlanmak üzere olan işten birikmiş deneyimi yakala; aynı PR'ın içinde yolculuk etsin.
1. **Sürümü artır** (takımlar için `team.json`; CLI için `internal/config.Version`), kesin SemVer kurallarına uyarak.
2. **Conventional commit** biçimi: `type(scope): summary`; gövde *neden*'i açıklar.
3. **Gövdeye "Discovered via" bağlamı** ekle — ihtiyacı hangi proje / oturum ortaya çıkardı; böylece takım deposunun günlüğü kendi kendini belgeler.
4. **PR akışı** (varsayılan, dal koruması tarafından zorlanır): özellik dalı → push → `gh pr create` → kullanıcı inceler ve birleştirme düğmesine basar.

Her adımın ayrıntısı, `team.json` doğrulama sözleşmesi ve mutlak ASLA-MERGE-ETME disiplini için bkz. [team-repo-maintenance sayfası](/tr/authoring/team-repo-maintenance).

## İki mutlak kısıt

Beş sabit adımın ötesinde, iki kısıt pazarlığa kapalıdır:

### 1. Claude PR'ları asla birleştirmez

İstisna yoktur. Önemsiz değişiklikler için, "0 onay gereksinimi" ritüeli için, kendi yazdığı PR'lar için ya da acil yamalar için bile değil. **Birleştirme insan inceleyiciye aittir.**

Tek dar istisna: kullanıcı, araçlar üzerinden auto-merge'i açıkça yetkilendirdiğinde — en sık [`/create-pr --auto-merge`](/tr/skills/create-pr) ile — Claude, GitHub'ın yerel auto-merge düzeneğini etkinleştirmek için `gh pr merge --auto --merge` çalıştırabilir. Dal korumasının denetim kapısı yine korunur (GitHub, birleştirmeden önce zorunlu denetimleri bekler). Tam yasak ve istisna belirtimi için bkz. [team-repo-maintenance "PR merge discipline" bölümü](/tr/authoring/team-repo-maintenance#pr-merge-discipline-absolute-no-exceptions).

Gerçek acil durumlarda doğrudan push atlatması yine geçerlidir (yönetici doğrudan commit yapabilir), çünkü bu farklı bir işlemdir — ama Claude bunu başlatmaz; kullanıcı başlatır.

### 2. Claude'un kendi PR'larında `--assignee` ya da `--reviewer` kullanmaz

Mevcut tek bakımcılı kurulumda Claude, bakımcının kendi GitHub hesabı üzerinden push yapar; bu da bakımcıyı otomatik olarak PR yazarı yapar:

- Yazar alanı, PR'ı bakımcının "Created by me" ve "Involves me" panolarında zaten gösterir.
- Açık `--assignee @me` gereksizdir (yazar = atanan) ve "Assigned to me" kuyruğunu kirletir.
- GitHub, PR yazarından inceleme istenmesini engeller; bu nedenle kendi PR'larda `--add-reviewer mkurak` sessizce başarısız olur.

(Claude'un push'ları için ayrı bir bot hesabı sağlandığında — yazar ≠ bakımcı olduğunda — `--reviewer mkurak` olası ve uygun hâle gelir. Bu, ileride yapılacak iştir; mevcut politika değildir.)

## Tek bakımcılı düzene özgü düşünceler

**Neden `required_approving_review_count = 0`?** Çünkü tek etkin bakımcıyla dış bir onaylayıcıyı zorunlu kılmak her PR'ı 24 saatlik bir bekleme bloğuna çevirir (başka bir hesabın onaylamasını beklemek). Sıfır onay = PR belgeleme amacıyla vardır, ama bakımcı kendisi birleştirebilir. İkinci bir bakımcı katılırsa 1'e çıkar.

**Neden yönetici atlatmasına izin var?** Yayım hattı kırılması, ivedi geri alım ya da PR ritüelini beklemenin gerçek bir kullanıcı maliyetine yol açtığı durumlar için. Kullanıldığında commit yine conventional biçimi izler ve bir retrospektifle eşlenmelidir.

**Solo vs takım ödünleşimi:** Bazı yönetişim katılıkçıları ikinci bir makine / kimlikten bile onay ister. Bu ekosistem için, bu aşamada, bu fazla tasarımdır. Dal koruması ile kural birleşimi durumların %95'ini yakalar (commit hijyeni, sürüm artırımları, denetim izi); kalan %5 (kendi kendini inceleme kalitesi) zaten tek bakımcının disiplinle üstlenmesi gereken bir şeydir.

## Bu yönetişimin KAPSAMADIĞI şeyler

- **Özel proje depoları** — kendi projenin Git iş akışı sana aittir. Bu yönetişim özellikle `agentteamland/` herkese açık depoları içindir.
- **Yayım hattı depoları** (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) — goreleaser kendiliğinden push yapar; dal koruması bilinçli olarak uygulanmaz.
- **Etiket tabanlı yayımlar** — `cli v0.2.1` etiketlenirken, etiket push'u goreleaser'ı tetikler. Etiket oluşturmak için PR gerekmez (etiket zaten birleştirilmiş bir commit'i `main` üzerinde işaret eder).

## Uygulama

2026-04-24 tarihinde uygulandı:

1. **GitHub API üzerinden dal koruması** 13 depoya (12 üretim + workspace) — anında etkili.
2. **`core@1.3.1` PR**: [feat(core): team-repo-maintenance rule](https://github.com/agentteamland/core/pull/1) — yeni akıştan geçen ilk değişiklik.
3. **`design-system-team@0.4.2` PR**: [fix(dst-new-ds): Q3 single-select](https://github.com/agentteamland/design-system-team/pull/1) — yeni akışın eş zamanlı ilk müşterisi.
4. **Workspace `CLAUDE.md` durum anlık görüntüsü** + [yerleşmiş karar belgesi](https://github.com/agentteamland/workspace/blob/main/.atl/docs/branch-protection-and-team-repo-governance.md) — yeni normali yansıtacak biçimde güncellendi.

## İlgili

- [Team-repo maintenance](/tr/authoring/team-repo-maintenance) — yordamsal ayrıntı (PR'ın gerçekte nasıl açılacağı, commit mesajının nasıl görüneceği).
- [`/create-pr`](/tr/skills/create-pr) — bu disiplini otomatikleştiren beceri.
- [Karpathy ilkeleri](/tr/guide/karpathy-guidelines) — bu yönetişimin kodlama-rehberi karşılığı.
- Yerleşmiş karar belgesi: [branch-protection-and-team-repo-governance.md](https://github.com/agentteamland/workspace/blob/main/.atl/docs/branch-protection-and-team-repo-governance.md).
- Dal koruması API başvurusu: [docs.github.com/en/rest/branches/branch-protection](https://docs.github.com/en/rest/branches/branch-protection).
