# delivery-team

**delivery-team**, **takılabilir bir backend (Azure DevOps ya da GitHub) üzerinde iş-öğesi güdümlü,
sprint tabanlı otonom bir yazılım-teslim organizasyonudur** — gerçek bir tracker'da (Azure Boards +
Repos ya da GitHub Issues + Projects + Pull Requests) iş öğelerini planlayan, ayrıştıran, geliştiren,
doğrulayan, gözden geçiren ve teslim eden bir rol-ajan ekibi; Product Owner rolünde bir insanla. Bu bir
**proje-kapsamlı** (project-scope) takımdır: teslim ettiği depoya kurulur.

```bash
atl install agentteamland/delivery-team
```

Kurulum, rol-ajanları, seremoni skill'lerini, bilgi paketlerini (knowledge packs) ve her iki backend
adaptör paketini (`backends/azure/`, `backends/github/`) projenin `.claude/` dizinine yerleştirir;
ardından `/delivery-init`, seremonilerin ve orkestrasyon motorunun okuduğu `.delivery/` config'ini yazar.

## Organizasyon

delivery-team, her biri kendi reflekslerine sahip birer uzman olan **rol-ajanlardan** oluşur:

| Rol | Ne yapar |
|---|---|
| `intake` | Ham bir isteği şekillenmiş bir Epic/Feature backlog öğesine ayıklar. |
| `business-analyst` | İş analizini yazar — Description'daki `## Problem / Business Value / Scope / Acceptance Criteria / Out of Scope`. |
| `technical-analyst` | `**[Technical Analysis]**` sentinel yorumunu yazar — yaklaşım, fizibilite, NFR'ler, bağımlılıklar, önerilen alanlar. |
| `project-manager` | Sprint temposunu yürütür — kapasite, iterasyon ataması, velocity. |
| `tech-lead` | Feature'ları iş-birimlerine ayrıştırır, her birimin `**[Canonical Brief]**`'ini yazar, proje wiki'sinin (`Architecture/`, `Conventions/`, ADR'ler) sahibidir ve **tek review kapısıdır** — her PR'ı gözden geçirir ve yeşilse tamamlar (= merge) ve Done'a set eder. |
| `tester` | Bağımsız Level-2 doğrulama — niyeti yeniden türetir, doğru yüzeyde test-gate'leri koşar, kanıt ekler, bir verdict yayınlar. |
| `developer` | İş-birimi başına spawn edilen, stack'ten bağımsız, dinamik bir worker; etiketli `area:<name>` bilgi-paketini yükler ve birimi implement eder. |

Belirli bir stack için bir **software team**, jenerik `developer`'ın yüklediği alan-anahtarlı bilgi
paketlerinden (`packs/<area>/`) ibarettir — M1 "knowledge-as-data" dikişi; böylece bir React ya da .NET
ekibi yeni bir ajan olmadan takılır.

## Seremoniler

Sprint, her biri doğru rol olarak davranan, senin çağırdığın skill'lerle işler:

```bash
/delivery-init      # backend'i seç (azure | github) + projenin koordinatlarını + metodolojiyi bağla
/kickoff            # intake + business-analyst Epic/Feature backlog'unu şekillendirir
/refine             # technical-analyst + tech-lead Feature'ları brief'li iş-birimlerine ayrıştırır
/sprint-plan        # project-manager sprint'in birimlerini kapasiteye göre seçer
/sprint-start       # iş-birimi DAG'ını materialize et → motora devret
/sprint-review      # velocity, review sonucu wiki sayfası, sprint kapanışı
/request            # (her an) proje-ortası istek → triyaj → fizibilite → dürüst PO kapısı → kabul/ertele/ret
```

Metodoloji **kod değil, config'tir**: `methodology.json` (v1'de Scrum) seremonilerin okuduğu tempoyu
bildirir — bakımı gereken bir workflow motoru yoktur.

## Motor — `atl work dispatch`

`/sprint-start`, seçilen birimleri bir `.delivery/plan.json` bağımlılık DAG'ına materialize eder, sonra
**deterministik Go motoru** `atl work dispatch` devralır. **Sıfır LLM context tutar ve sıfır Azure çağrısı
yapar**: hazır birimleri bir eşzamanlılık sınırına kadar admit eder ve her biri için **tek bir git
worktree'de üç izole `claude -p` worker'dan oluşan bir pipeline** spawn eder —

```
developer  →  tester  →  tech-lead
(implement    (Level-2     (review → vote →
 + PR aç)      verify)      PR-complete = dev'e merge → Done)
```

Motor, bir worker'ın temiz çıkışında stage'i ilerletir, tech-lead'in merge'inin `dev`'e indiğini saf bir
git okumasıyla doğrular (worker'ın exit code'una asla güvenmez), worktree'yi geri alır ve DAG'ı doldurur.
Stall eden ya da çöken bir worker geri alınıp bir kez retry edilir, sonra mark-blocked olur — bunu
`/sprint-review`'ın backend'e yansıttığı (`blocked` tag'i ya da label'ı + tanı yorumu) ve temizlediği
kalıcı bir rapor. Her worker tracker'a yalnız motorun ona bağladığı şey üzerinden erişir — Azure
backend'inde proje-kapsamlı `azureDevOps` MCP, ya da GitHub backend'inde motorun enjekte ettiği bir
`GH_TOKEN` (`config.credential.ref`'ten çözümlenir) ile `gh` CLI — asla operatörün ortam MCP config'i ya
da kimlik bilgileri değil.

## Backend tek gerçek kaynaktır

Yerel bir veritabanı yoktur. **İş-öğeleri geçici yürütme durumudur** ve **kalıcı-bilgi deposu kalıcı
bilgiyi tutar** (ATL wiki/journal ayrımının backend'de yaşayan hali: Azure'da proje wiki'si, GitHub'da
repo-içi bir `docs/` ağacı). Her rol, backend'e tek bir belgelenmiş **sağlayıcıdan-bağımsız
operasyon-sözleşmesi** (`knowledge/backend-interface.md`) üzerinden erişir; bu, sağlayıcı başına bir
adaptör paketiyle bağlanır — `backends/azure/adapter.md` (`azureDevOps` MCP: iş-öğeleri için `wit_*`,
PR'lar için `repo_*`, bilgi için `wiki_*`, MCP'nin eksik olduğu tek operasyon için, kanıt ekleme, ince
bir REST carve-out ile) ya da `backends/github/adapter.md` (`gh` CLI: Issues, Projects v2, Pull Requests
ve repo-içi `docs/` deposu). İçerik **makine-bulunabilir sentinel'lerle** yerleştirilir: iş analizi Description'da,
`**[Technical Analysis]**` ve `**[Canonical Brief]**` yorumları her biri tam ilk satırıyla eşleşerek
("en yeni yorum" değil), alan bağlama `System.Tags: area:<name>` ile.

## İşi teslim etmek — iki-branch akışı

İş **`dev`**'e entegre olur (tech-lead her birimin PR'ını yeşilde tamamlar — platformun never-merge
kuralının kapsamlı istisnası) ve Product Owner onaylanmış bir sprint'i `dev`'den **`release`**'e promote
eder — asla sohbette verilen bir onayla değil, `atl work promote`'un backend'den geri okuyup karşısında
merge ettiği, commit'e bağlı bir onay kaydıyla (aşağıdaki kapı; v1 onu **GitHub**'da bağlar, Azure'da ise
eksik okuma bağlanana kadar promote'u **bekletir**). Review **delivery-native**'dir:
tech-lead adversarial review desenini (evidence gate + refute-to-keep) doğrudan backend'in PR'ı üzerinde
koşar — Azure'da `repo_*` thread'leri ve vote, GitHub'da `gh pr comment` / `gh pr review` — `/create-pr`
değil.

## Promote kapısı — commit'e bağlı onay

`dev` → `release`, geri alınamaz tek adımdır; bu yüzden sohbette verilmez. `/sprint-review` raporu derler,
`dev` → `release` **promote PR'ını** açar (ya da bulur) ve kararı **`atl work promote`**'a devreder —
backend'den kalıcı bir **onay kaydını** geri okuyan, onu PR'ın güncel head'iyle karşılaştıran ve yalnız
birebir eşleşmede merge eden deterministik bir komut. O PR'ı açmak artık promote etmek değildir —
**promote etmek onu merge etmektir**. v1'de bu kapı yalnız **GitHub** backend'inde bağlıdır — Azure'da
hâlâ neyin eksik olduğu için bu bölümün sonundaki **v1'de yalnızca GitHub** bloğuna bak.

**Bu kontrol seremoni talimatı değil, koddur — ve asıl mesele budur.** İlk sürümünde kontrol
`/sprint-review` becerisinin içine yazılmış bir prosedürdü ve gerçek bir koşuda aynı seremoni onu bir turda
uygulayıp bir sonraki turda sessizce atladı: sohbette "Approve or Reject?" diye sormaya, yani tasarımın
ortadan kaldırdığı kapıya geri döndü. Hatırlanmaya bağlı bir adım er geç hatırlanmaz ve bu sessizce olur.
Bu yüzden komut **doğrulamayı ve merge'ü tek çağrıda** yapar: bir seremoninin kontrolü atlayarak
ulaşabileceği ayrı bir merge adımı yoktur; seremoninin işi komutu çalıştırıp verdiği kararı aktarmaya iner.

Onaylamak için Product Owner, promote PR'ına, **ilk satırı tam olarak** `**[Promotion Approval]**` olan ve
onaylanan commit'i adıyla veren bir yorum ekler:

```markdown
**[Promotion Approval]**

## Approved Commit
<40 karakterlik küçük harfli hex commit id'si>

## Sprint
Sprint <n> · <iterasyon-adı>

## Decision
APPROVE
```

Yalnızca `## Approved Commit` taşıyıcıdır — gerisi denetim bağlamıdır. Bunu PR'ın yorum kutusuna yapıştır
ya da CLI'dan gönder:

```bash
gh pr comment <PR#> --repo <owner>/<repo> --body-file approval.md
```

PO'nun işi bundan ibarettir. `atl work promote` ardından PR'ın head'ini ve üzerindeki her kaydı tek
çağrıda okur ve yalnız bir kayıt tam olarak o head'i verdiğinde merge eder — SHA'ya sabitlenmiş biçimde
(`gh pr merge --merge --match-head-commit <onaylanan-commit>`), yani head arada oynadıysa merge'ü
sağlayıcının kendisi reddeder. Sonuç, sprint'in review sayfasına `## Promotion decision` başlığı altında
yazılır: onaylanan commit, onaylayan, zaman damgası, PR bağlantısı. Diğer her sonuç bir **HOLD**'dur —
komut sıfırdan farklı bir kodla çıkar, hiçbir şey merge olmaz, hiçbir iş-öğesi durum değiştirmez ve mesaj
tam olarak neyi set etmen gerektiğini yazar:

| Kapı ne bulur | Ne yapar (`reason`) |
|---|---|
| PR'da `**[Promotion Approval]**` kaydı yok | Bekletir; PR bağlantısını + onaylanacak head commit'i yazar (`no-record`). |
| Kayıt var ama `## Approved Commit` altında 40-hex id yok | Bekletir; güncel head'i veren yeni bir kayıt ister (`unparseable-record`). |
| Kayıt, PR'ın head'i olmayan bir commit'i veriyor | Bekletir — onaylanan durum, teslim edilecek durum değildir (`superseded`). |
| Kayıt okunamadı | Bekletir. Doğrulanmamış, onaylanmış değildir (`read-failed`). |
| Kayıt eşleşti ama GitHub SHA'ya sabitlenmiş merge'ü reddetti | Bekletir (`merge-refused`) — kontrol ile merge arasında `dev` ilerlediği için sağlayıcı reddetti. Hiçbir şey promote edilmedi; yeni head'i onayla. |
| Üzerinde çalışılacak açık bir `dev` → `release` PR'ı yok | Bekletir — önce onu aç (`no-open-pr`). Zaten promote edilmiş bir sprint de böyle yakınsar: yeniden koşu hiçbir şeyi merge etmez. |
| Backend head-commit okumasını bağlamıyor | Bekletir (`backend-unbound`) — aşağıdaki **v1'de yalnızca GitHub** bloğuna bak. |

HOLD bir ret değildir: hiçbir şey kapatılmaz, hiçbir şeye carryover etiketi konmaz ve kayıt olduğu yerde
bırakılır. Açık bir **ret** sohbette kalır ve mevcut carryover yolunu işletir — kapı yalnız geri alınamaz
yönü korur; bir promote'u geri çevirmek fazladan bir şey teslim edemez.

**`dev`, onaydan sonra ilerlediyse onay da onunla birlikte geçersizleşir.** Kapı onaylanan commit'i ve
güncel head'i bildirir ve onayı ileriye **taşımaz**. Güncel durumu promote
etmek için tazelenmiş raporu yeniden oku ve yeni head için yeni bir kayıt ekle; tam olarak onayladığın şeyi
promote etmek için önce `dev`'i o commit'e geri al. Eski kayıt denetim tarihçesi olarak yerinde kalır —
kanal ekle-ve-geçersiz-kıl (append-and-supersede) biçiminde çalışır ve yeni kayıt, daha yeni bir commit'i
vererek eskisinin yerini alır.

İki sınır, açıkça:

**Kontrol edilebilir, taklit edilemez değil.** Etkileşimli bir oturumda seremoni PO'nun kendi kimlik
bilgisini taşır; dolayısıyla bir yazar kontrolü, seremoninin yazdığı bir kaydı PO'nun yazdığından ayırt
edemez. Kapının kazandırdığı şey şudur: bir promote artık commit-kapsamlıdır (gözden geçirilen durumdan daha
yenisini sessizce teslim edemez) ve atfedilebilirdir (bir commit, bir yazar ve bir zaman damgası veren
kalıcı bir kayıt) — bu bir doğruluk kapısıdır, bir kimlik doğrulama kapısı değil.

**v1'de yalnızca GitHub.** Azure'da onay kaydının kendisi bağlıdır (PR thread'leri, hem okuma hem yazma),
ama head-commit okuması bağlı değildir: branch okumasında (`repo_branch`, `action: "get"`) commit id'sini
taşıyan yanıt alanı canlı bir sunucuya karşı çözümlenmedi ve takım, kanıtlayamadığı bir alan adını asla
yazmaz. **Dolayısıyla commit'e bağlı kapı Azure backend'inde henüz çalışmıyor.**

Bu bir **HOLD'dur, bir geri-dönüş (fallback) değil** — çözülemeyen bir head okuma hatasıdır: Azure'da
`/sprint-review` raporu derler, promote PR'ını açar (ya da bulur) ve sonra bekletir; `atl work promote`
`backend-unbound` bildirir, hiçbir Azure yüzeyini çağırmaz ve hiçbir şeyi merge etmez. Sohbette verilen
bir onayla promote etmeye **geri dönmez**. Okuma bağlanana kadar bu
sürümdeki bir Azure projesi, promote'u o PR'ı Azure DevOps üzerinde kendisi tamamlayarak yapar. O alanı
canlı sunucuya karşı çözümleyip bağlamak, adı konmuş bir sonraki adımdır.

## Neler geliyor

Tam rol-ajan organizasyonu, altı seremoni skill'i, `atl work dispatch` motoru, Azure DevOps ve GitHub
adaptör paketleriyle sağlayıcıdan-bağımsız backend arayüzü, bir Scrum `methodology.json`'ı ve dört-alanlı
bir referans paketi (web / mobile / api / go-cli). Otonom developer→tester→tech-lead döngüsü, canlı bir
Azure DevOps projesine karşı uçtan uca kanıtlanmıştır.

Ertelenenler (tasarım yakalandı, tetik-kapılı): Scrum ötesi **çoklu-metodoloji** desteği, jenerik
developer'ın **stack-özel override'ı**, **dinamik-kapasite** eşzamanlılığı, bir **hotfix akışı** ve
**device-farm** emulator'ları. **mobile-emulator** test hattı yapıldı ama canlı doğrulaması bir masaüstü
(GUI) oturumuna kapılı.

## Ayrıca bakın

- [`atl install`](/tr/cli/install) — bir takımın nasıl çözümlenip kurulduğu
- [Takımlar](/tr/teams/) — katalog ve ilk-parti yeniden inşa
- [Kavramlar: scope](/tr/guide/concepts#scope-global-and-project) — proje vs. global takımlar
