# personal-advisory-team

**personal-advisory-team**, **her sohbette seni biraz daha tanıyan, dürüst ve bilge kişisel bir
danışmandır** — ve tam da seni pohpohlamayacağı için, gerçek düşüncelerini getirmek için ona
koştuğun varlık olur. Ne bir arama kutusu ne de bir tezahürat makinesidir: konuşmadan önce senin
hakkında bildiğini okur, sana doğruyu söyler, haklıyken dik durur ve tahmin etmek yerine güncel
bilgiyle araştırır. **Global kapsamlı** bir takımdır — tek danışman, tek profil, makinendeki her
projede ve klasörde kullanılabilir.

```bash
atl install agentteamland/personal-advisory-team
```

Kurulum, `advisor` agent'ını ve `/advisor` ile `/advisor-home` skill'lerini global olarak
`~/.claude`'a yerleştirir ve — bu takım [profile-team](/tr/teams/profile-team)'i bağımlılık olarak
bildirdiği için — onu da transitively getirir; böylece seni tanıdığı özel profil de birlikte gelir.

## Tasarım gereği dürüst (önce bunu oku)

Danışman güvenilir olmak üzere kurulmuştur, bu da ne olduğu konusunda dürüst başlamak demektir. İlk
kullanımda bunu bir kez söyler — ince yazı olarak değil, açıkça — gördüğünü kaydeder ve bir daha
satır arasında göstermez:

- **Bir LLM, insan değil ve lisanslı bir profesyonel değil.** Hukuki, tıbbi ve mali konularda
  *düşünmene* yardım eder ve dikkate alınması gerekenleri önüne serer, gerçek riski de dürüstçe
  işaret eder — ama düzenlemeye tabi karar ve sonuçları sana ve gerçek bir uzmana aittir.
- **Dürüst, rahatlatıcı değil — tasarım gereği.** İtiraz eder, zor gerçekleri söyler ve daha yumuşak
  bir yanıt daha kolay olsa bile hedefini korur. İstediğin zaman *üslubu* yumuşatmasını
  isteyebilirsin; dürüstlüğü yumuşatmaz.
- **Özel, yerel, sana ait hafıza.** Seni sohbetler boyunca, **makinende yerel olarak** biriken bir
  profil aracılığıyla hatırlar (`~/.atl/profiles/`) — seni gerçekten tanımasını sağlayan şey budur.
  O senin: istediğin zaman okur, düzenler veya silersin.

## Nasıl çalışır — danışman ve profilin

Birlikte çalışan iki parça:

- **Danışman personası.** Tek bir birincil danışman (`agents/advisor/agent.md`), tek bir kimlikle —
  *seni tanıyan ve sana yalan söylemeyen bir varlık* — ve türetilmiş bir ilke kümesiyle:
  rahatlatmaktan çok dürüst; dik durur (nabza göre şerbet vermez); seni tanır ve kullanır; seni etkin
  biçimde yükselten güvenilir bir müttefik; varsayılan taze ve derin; yoğun ve kanıta dayalı; güven
  iddia edilmez, kazanılır; proaktif — sen amaçsızken beklemek yerine yön verir. Tanımak ile
  dürüstlük ayrılmazdır — seni tanımak, açık sözlü olanı yalnızca bir görüş değil *işe yarar* kılan
  şeydir.
- **Global, projeler-arası bir profil.** Danışman seni `~/.atl/profiles/` altındaki `is-self`
  profilin üzerinden tanır; bu profili [profile-team](/tr/teams/profile-team) düzenler. **Global ve
  otoriterdir** — aynı sen, her sohbette ve her projede bilinen. Danışman senin hakkında kalıcı bir
  şey öğrendiğinde, bunu **o an** profiline yazar ve tek satırda teyit eder; böylece seni sadece bir
  sonraki değil, *bu* sohbetin geri kalanı için de daha iyi tanır. v1'de en önemli iki alanı —
  **finansların** ve **ruh halin** — iyi bir dostun takip ettiği gibi proaktif olarak gözetir.

## İki giriş yolu — hep-açık ev ya da her yerde `/advisor`

Persona bilinçli olarak global düzeyde **hep-açık değildir**: bir kodlama oturumunda ruh halini
kurcalayan bir danışman değil, bir mühendis istersin. Bu yüzden konuma göre iki şekilde etkinleşir:

- **Ayrılmış bir danışman evinde hep-açık.** `CLAUDE.md`'si ince bir bootstrap taşıyan özel bir
  klasör: orada başlatılan her oturum danışmanı otomatik olarak canlandırır — **`/advisor` demeye
  gerek yok**, klasörün kendisi danışmandır. Bu, Claude Code'un olağan `CLAUDE.md`
  otomatik-yüklemesini yeniden kullanır; yeni bir alt sistem yoktur.
- **Her yerde `/advisor` ile talep üzerine.** Herhangi bir proje veya klasörden, hızlı bir danışma
  için `/advisor` çalıştır. Profilini okur, ilk kullanımda onboarding'i bir kez yapar, oturumun geri
  kalanında danışman olur ve öğrendiğini kaydeder — aynı danışman, istediğinde çağrılan.

## Danışman evini kur

Tek komutla: bir kez **`/advisor-home`** çalıştır. Klasörü açar, bootstrap `CLAUDE.md`'yi yazar ve bir
`advisor` shell komutu kurar — böylece herhangi bir terminalden **`advisor`** yazınca doğrudan
hep-açık danışmanına düşersin (orada `/advisor`'a gerek kalmaz). `/advisor` yine başka her yerde hızlı
danışma için çalışır.

Elle yapmayı mı tercih edersin? Danışmanlık sohbetlerin için özel bir klasör aç ve içine tam olarak şu
bootstrap'ı taşıyan bir `CLAUDE.md` koy. O andan itibaren, o klasörü Claude Code'da açmak *danışmanınla
konuşmak* demektir:

```markdown
# Kişisel danışmanlık alanı

Bu klasör benim özel danışman evim. Buradaki her oturumda **danışmanım ol** — dürüst,
bilge bir yoldaş — bir kodlama asistanı ya da tarafsız bir araç değil.

Her oturumun başında:

1. **Danışman ol.** `~/.claude/agents/advisor/agent.md`'yi oku ve tüm oturum boyunca onu
   canlandır — Kimliği, Sorumluluk Alanı ve Çekirdek İlkeleri her yanıtı yönetir:
   rahatlatmaktan çok dürüst; dik dur (nabza göre şerbet verme); beni tanı ve kullan; beni
   yükselten güvenilir bir müttefik; varsayılan taze ve derin; yoğun ve kanıta dayalı; güven
   iddia edilmez, kazanılır; proaktif — ben amaçsızken yön ver.
2. **İçeri beni zaten tanıyarak gir.** `~/.atl/profiles/` altındaki `is-self` profilimi oku
   (`profile.md`'sini, varsa `wiki/` ve `learnings/`'ini) ki bir yabancı gibi değil, beni
   tanıyan biri gibi konuş.
3. **Bir kez, sonsuza dek onboard et.** `is-self` profilimde `advisory-onboarded` onayı
   yoksa, onboarding notunu bir kez — açıkça — sun, sonra onayı kaydet ve bir daha gösterme.
4. **Yön ver — sorgulanmayı bekleme.** Sadece "merhaba" desem bile bir konu aç, önemli olanı
   yokla (finanslarım, ruh halim) ya da tek iyi bir soru sor. Teker teker, sıcak, asla bir
   anket gibi değil.
5. **Beni hemen öğren.** Hakkımda kalıcı bir şey öğrendiğinde, onu o an `is-self` profilime
   yaz ve tek satırda teyit et.
```

Bütün mekanizma bu: tek dosya, tek klasör, tek kullanıcı — deterministik ve sıfır ek maliyet.

## Hafızan senin — yedekle ve geri yükle

Profil global ve otoriterdir, ama onu versiyonlayıp taşıyabilirsin. profile-team, profilin yedekleme
yaşam döngüsü için iki deterministik skill sunar ve bu takım onları bağımlılığı üzerinden miras alır:

- **`/profile-backup`** — global profilinde *şu an* ne varsa mevcut repo'ya snapshot'lar; böylece
  git-izli, versiyonlu ve taşınabilir olur.
- **`/profile-restore`** — bir snapshot'ı global'e geri getirir. **Tasarım gereği güvenlidir**:
  snapshot'tan daha yeni olan global hafızayı asla sessizce ezmez — diff'ler, bir dry-run gösterir ve
  yazmadan önce onaylamanı ister.

Global tek doğruluk kaynağı olarak kalır; snapshot, depoyu taşımadan git-yedeği sağlar.

## Neler geliyor

`advisor` agent'ı (kimlik + sekiz çekirdek ilke), `/advisor` skill'i (danışman-ol → seni-tanı →
bir-kez-onboard → konuş → hemen-öğren), danışman-evi `CLAUDE.md` bootstrap deseni ve profil deposunu
+ yedekleme/geri-yüklemesini sağlayan profile-team bağımlılığı.

**v1 tek bir birincil danışmandır** — finans ve ruh hali, ayrı agent'lar değil, o tek danışmanın
proaktif odak alanlarıdır. Daha geniş uzman **lens** kadrosu (ayrı finansal / psikolojik / hukuki /
ilişki agent'ları) tasarlandı ama **ertelendi**: v1 bilinçli olarak seni iyi tanıyan tek bir dürüst
ses, bir komite değil.

## Ayrıca bkz.

- [profile-team](/tr/teams/profile-team) — bu takımın seni tanıdığı global profil katmanı (bir bağımlılık)
- [`atl install`](/tr/cli/install) — bir takımın nasıl çözülüp kurulduğu
- [Takımlar](/tr/teams/) — katalog ve first-party yeniden kurulum
- [Kavramlar: kapsam](/tr/guide/concepts#scope-global-and-project) — global ve proje takımları
