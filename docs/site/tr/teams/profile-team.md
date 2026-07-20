# profile-team

**profile-team**, dünyandaki insanların **ve şeylerin** paylaşımlı, projeler-arası bir
profilini yönetir — anlam taşıyan varlıklardan oluşan iç dünyanın: insanlar, organizasyonlar,
hayvanlar, yerler, nesneler ve projeler. Yeniden inşa edilen ilk birinci-parti takım ve bir
**global-kapsamlı** takımdır: aynı varlık, çalıştığın her projede tek bir profildir.
Duraklatılmış kişisel-danışman yığınının üstüne kurulduğu temeldir — danışman lens'leri onun
yönettiği profilleri okur.

```bash
atl install agentteamland/profile-team
```

Varsayılan olarak global kurulur (`team.json`'ı `scope: global` bildirir); `profile-curator`
ajanını, `/profile-drain`, `/profile-backup` ve `/profile-restore` becerilerini ve
`profile-capture` kuralını `~/.claude`'a yerleştirir, profilleri `~/.atl/profiles/` altında
saklar.

## Profil dünyası

Her şey global ATL katmanı altında `~/.atl/profiles/`'te yaşar:

```
~/.atl/profiles/
├── _index.md                     # keşif: kim var, önem (salience), rol
├── _interfaces/                  # kendini-tanımlayan şemalar — altı tür seed'lenir
│   ├── person.md                 #   (+ yeni türler için ajan-yazımı arayüzler)
│   ├── org.md · animal.md · place.md · object.md · project.md
│   └── migrations/               # kırıcı-değişiklik migration dosyaları, dokununca uygulanır
└── <type-dir>/                   # tür başına bir dizin: people · orgs · animals ·
    └── <slug>/                   #   places · objects · projects (+ taslaklar için unknown/)
        ├── profile.md            # frontmatter çekirdek + anlatı gövdesi
        ├── wiki/                 # bu varlık için konu-organizasyonlu güncel gerçek
        └── learnings/            # örüntü-organizasyonlu, KB'den yeniden derlenen
```

Bu dünya **varlık-organizasyonludur** ve bir projenin **konu-organizasyonlu** `.atl/wiki/` ve
`.atl/journal/`'ından bilinçli olarak ayrıdır. İkisi yalnızca serbest göreli markdown
bağlantılarıyla birbirine referans verir. Profiller global olduğundan asla bir projenin içinde
yaşamaz — profil-kör projeler (saf bir yazılım deposu) sıfır maliyet öder ve bir projede
andığın kişi her yerde aynı profildir.

## Nasıl öğrenir — önce yakalama, sonra tahliye (drain)

profile-team, ATL'nin marker → kuyruk → drain düzeneğini özel bir `profile-fact` kanalında
yeniden kullanır — [öğrenme döngüsünün](/tr/guide/learning-marker-lifecycle) kardeşi:

1. **Yakalama.** `profile-capture` kuralı, bir kişi hakkında kalıcı bir bilgi ortaya çıktığında
   asistana sessiz bir marker düşürmeyi öğretir:

   ```html
   <!-- profile-fact:
     entity: alex
     kind: friend
     fields:
       identity.name: Alex Doe
       traits.fears: [confrontation]
       state.emotional: anxious about the new job
     source: user-confirmed
   -->
   ```

2. **Kuyruk.** `atl tick` (ve oturum başlangıcı) marker'ları kuyruğun `profile-fact` kanalına tam
   olarak bir kez aktarır — deterministik, LLM yok. `atl learnings status` onları sayar;
   `atl learnings peek --channel profile-fact` inceler.

3. **Sinyal.** `profile-fact` kanalı boş olmadığında, `atl tick` (her turda) ve oturum
   başlangıcı bir auto-drain sinyali basar — `N profile-fact(s) pending — auto-drain …`,
   öğrenme sinyalinin kardeşi. `profile-capture` kuralı buna göre davranır: ajan arka planda
   **tek** bir `/profile-drain` subagent'ı başlatır (single-in-flight), böylece entegrasyon
   otomatiktir ve onu asla elle çalıştırmazsınız.

4. **Drain.** `/profile-drain`, bekleyen bilgileri `profile-curator` ajanına devreder; ajan her
   birini doğru kişiye çözer, uygular (gizlilik-kapılı, kaynak-etiketli), şemayı evrimleştirir,
   `_index.md`'yi yeniden kurar ve onaylar (ack). **Yeni** bir kişi oluşturmadan önce bir
   **gerçeklik kapısı** uygular: capture taramasının süpürdüğü bir doküman örneği ya da format
   placeholder'ı (gerçek ilişki/durum olmayan, yalnız bir kalıp-trait taşıyan çıplak isim)
   uydurma bir kişiye dönüştürülmez, düşürülür — mevcut bir profil gerçeklik-kanıtıdır ve asla
   kapıya tabi tutulmaz. Çekirdek `/drain` yalnızca `learning` kanalında kalır — `profile-fact`
   profile-team'in kanalıdır.

## interface'ler

Her varlık **türü**nün kendi **kendini-tanımlayan** interface dosyası vardır
(`_interfaces/<type>.md`). Altı tanesi tohumlanmıştır — **person, org, animal, place, object,
project** — ve dünya tür-açıktır (aşağıda). Bir interface'in frontmatter'ı ne olduğunu
(`matches` + örnekler, tür tespiti için), `schema-version` + `changelog`'unu (evrim için),
`tier-defaults`'unu (gizlilik), `thresholds`'unu (tür-eşleşmesi + salience) ve izinli enum'larını
taşır. `fields:` bloğu **hibrit** bir yapıdır: her türün paylaştığı ortak bir **çekirdek**
(`meta`, `identity`, `salience` dahil `relation-to-user`, `emotional-tags`) artı bir **tür
uzantısı** — person dokuz trait alanı + skills + `state.{emotional,goals,financial}` +
relationships ekler; animal species + `adopted`/`passed` anchors + history-tracked health;
place bond + sensory-memories; object provenance + history-tracked status; project
status/motivation/stakes; org standing + key-people bağlantıları.

**Tür tespiti.** Bir bilgi bir varlığı adlandırdığında, curator marker'ın opsiyonel `type:`
ipucunu alır, yoksa varlığı her interface'in `matches` + örneklerine karşı fit-scorlar ve
0.80 eşiğinde/üstünde en iyi uyanı yeniden kullanır.

**Tür-açık (auto-creation).** Bir varlık tohumlanmış türlerin hiçbirine iyi uymadığında ve
*tutarlı, tekrar eden bir tür* olduğunda, curator onun için **anında yeni bir interface yazar**
— sessiz ama guardrail'li (çekirdek üstünde küçük bir uzantı, konservatif varsayılan tier'lar,
`authored: agent-<date>` damgalı, incelenebilir kalsın diye). Gerçek bir tek-seferlik varlık
hafif bir `unknown` taslağı olarak tutulur. Bu, profil dünyasının hiç öğretilmediği türleri de
tutabilmesini sağlar.

**Interface evrimi.** Zorunlu alan yoktur — her profil her zaman geçerlidir ve curator'ın
disiplini, *kanıtın desteklediği ölçüde alanları doldurmaktır*, doğrulamak değil. Interface
büyüdüğünde (minor bir sürüm alan eklediğinde) eski profiller toplu olarak taşınmaz; her biri,
drain bir sonraki kez dokunduğunda **tembel (lazy)** yakalar — changelog'un `added` listeleri
deterministik bir doldurmayı yönetir. **Kırıcı** bir değişiklik (bir alanı yeniden adlandıran,
kaldıran ya da yeniden şekillendiren major sürüm) bunun yerine, curator'ın dokununca çalıştırdığı
bir **göç dosyasıyla** (`_interfaces/migrations/<type>/<from>-to-<to>.md`) uygulanır — bir gizlilik
kapısını asla zayıflatmayacak şekilde doğrulanır ve her değerin kaynağını taşıma boyunca korur;
dosya eksikse profil, tahmin edilmeden, eski şemasında bırakılır ve işaretlenir. Çıkarım
(inference) tolere edilir ama etiketlenir (`agent-inferred-<date>`), böylece yanlış bir tahmin
gerçeğe sertleşmek yerine sonraki bir konuşmada kendini düzeltir. Eşikler interface frontmatter'ında yaşar (v2'de tasarım gereği
config sistemi yoktur — bunlar türe özgüdür, yani interface onların evidir).

## Gizlilik

Her alan dört katmandan birine eşlenir ve curator'ın ne yazdığını kapılar:

| Katman | Örnek alanlar | Davranış |
|---|---|---|
| **1 — Açık** | identity, anchors, relation kind/role | Her zaman yazılır. |
| **2 — Algı-işaretli** | traits | Yazılır; üçüncü-taraf profilde (`is-self: false`) *kullanıcının algısı* olarak kaydedilir. |
| **3 — Açık sinyal** | state.emotional, state.goals | **Yalnızca** `user-confirmed` bir bilgiden yazılır; çıkarılmış değer reddedilir. |
| **4 — Onay-kapılı** | state.financial | **Yalnızca** kullanıcı onay verdiyse (`meta.consent.<field>`) yazılır, varsayılan kapalı. |

Yazılan her alan `source`'unu da kaydeder (`user-confirmed` / `agent-inferred-<date>` /
`lens-set`) ve `meta.is-self` kullanıcının kendi profilini işaretler — en hassas alanların
doğrudan kaydedilebildiği tek yer.

## Profilleri okuma

Tüketen bir takımın lens'i profilleri doğrudan okur: kimin ve neyin var olduğunu görmek için
`_index.md`'yi istek üzerine yükler, sonra ihtiyaç duyduğu belirli
`~/.atl/profiles/<type>/<slug>/profile.md`'yi okur (profiller, wiki gibi düz markdown'dır). Index
asla `CLAUDE.md`'ye enjekte edilmez — yalnızca bir lens gerçekten kullanıcının dünyası üzerine
akıl yürütürken çekilir.

**Takımlar-arası erişim** açık değil, bildirimlidir. Üçüncü-taraf danışman takımlar geldiğinde,
her biri profil erişimini kendi `team.json`'ında `capabilities.profile` altında bildirir (`reads`
/ `writes` alan listeleri), kurulum anında yüzeye çıkarılır — böylece bir hukuk-danışma takımına,
bir esenlik takımından farklı erişim verilebilir. v1'de tek tüketici personal-advisory-team'dir;
sözleşme şimdi vardır ki üçüncü-taraf takımlar serbestçe okumadan önce yerinde olsun.

## Ne geliyor

profile-team, **altı varlık türü** (person, org, animal, place, object, project) üzerinde tam
döngüyü sağlar — her biri kendi kendini-tanımlayan interface'iyle — artı tutarlı yeni bir tür
için **auto-creation** (yeni interface yazma) ve interface **evrimi** (add-only büyüme için
changelog-güdümlü lazy-fill, artı dokununca uygulanan kırıcı-değişiklik göç dosyaları). Tümü,
kuzey-yıldızı tüketicisi (personal-advisory-team) üzerinde doğrulanır.

Daha sonraki bir sürüme ertelenen (tasarımı yakalanmış, tetik-gözetimli): **zamanlanmış /
aralıklı drain** (bugün oturum başlangıcında çalışır — ayrı bir ATL zamanlama primitive'ine
bağlı); profil ve proje dünyaları arasında **yapılandırılmış çapraz-bağlantılar** (bugün serbest
markdown linkleri); ve ilk tüketen takımla gelen tüketici-tarafı erişim kapısı olan
**`capabilities.profile` enforcement**.

## Ayrıca bkz.

- [Öğrenme marker yaşam döngüsü](/tr/guide/learning-marker-lifecycle) — `profile-fact`'in yansıttığı kardeş döngü
- [Kavramlar](/tr/guide/concepts) — global ve proje katmanlarının nasıl etkileştiği
- [`atl learnings`](/tr/cli/learnings) — kuyruğu incele (`--channel profile-fact`)
- [`atl install`](/tr/cli/install) — bir takımın nasıl çözülüp kurulduğu
