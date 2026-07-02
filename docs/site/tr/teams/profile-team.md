# profile-team

**profile-team**, dünyandaki insanların paylaşımlı, projeler-arası bir profilini yönetir.
Yeniden inşa edilen ilk birinci-parti takım ve bir **global-kapsamlı** takımdır: aynı kişi,
çalıştığın her projede tek bir profildir. Duraklatılmış kişisel-danışman yığınının üstüne
kurulduğu temeldir — danışman lens'leri onun yönettiği profilleri okur.

```bash
atl install profile-team
```

Varsayılan olarak global kurulur (`team.json`'ı `scope: global` bildirir); `profile-curator`
ajanını, `/profile-drain` becerisini ve `profile-capture` kuralını `~/.claude`'a yerleştirir,
kişi profillerini `~/.atl/profiles/` altında saklar.

## Profil dünyası

Her şey global ATL katmanı altında `~/.atl/profiles/`'te yaşar:

```
~/.atl/profiles/
├── _index.md                     # keşif: kim var, önem (salience), rol
├── _interfaces/
│   └── person.md                 # kendini-tanımlayan şema (v1'de person)
└── people/
    └── <slug>/
        ├── profile.md            # frontmatter çekirdek + anlatı gövdesi
        ├── wiki/                 # bu kişi için konu-organizasyonlu güncel gerçek
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

3. **Sinyal.** Oturum başlangıcında `atl`, `N profile-fact(s) pending — run /profile-drain`
   satırını yüzeye çıkarır (öğrenme sinyalinin kardeşi).

4. **Drain.** `/profile-drain`, bekleyen bilgileri `profile-curator` ajanına devreder; ajan her
   birini doğru kişiye çözer, uygular (gizlilik-kapılı, kaynak-etiketli), şemayı evrimleştirir,
   `_index.md`'yi yeniden kurar ve onaylar (ack). Çekirdek `/drain` yalnızca `learning`
   kanalında kalır — `profile-fact` profile-team'in kanalıdır.

## person interface

Şema tek bir **kendini-tanımlayan** interface dosyasıdır (`_interfaces/person.md`). Kendi
frontmatter'ı ne olduğunu (`matches` + örnekler, tür tespiti için), `schema-version` +
`changelog`'unu (evrim için), `tier-defaults`'unu (gizlilik), `thresholds`'unu (tür-eşleşmesi +
salience) ve izinli `kind`/`role` enum'larını taşır. `fields:` bloğu **hibrit** bir yapıdır: her
varlığın paylaştığı ortak bir **çekirdek** (`meta`, `identity`, `salience` dahil
`relation-to-user`, `emotional-tags`) artı bir **person uzantısı** (dokuz trait alanı + skills,
`identity-extension`, `anchors`, `state.{emotional,goals,financial}`, `relationships`).

**Interface evrimi.** Zorunlu alan yoktur — her profil her zaman geçerlidir ve curator'ın
disiplini, *kanıtın desteklediği ölçüde alanları doldurmaktır*, doğrulamak değil. Interface
büyüdüğünde (minor bir sürüm alan eklediğinde) eski profiller toplu olarak taşınmaz; her biri,
drain bir sonraki kez dokunduğunda **tembel (lazy)** yakalar — changelog'un `added` listeleri
deterministik bir doldurmayı yönetir. Çıkarım (inference) tolere edilir ama etiketlenir
(`agent-inferred-<date>`), böylece yanlış bir tahmin gerçeğe sertleşmek yerine sonraki bir
konuşmada kendini düzeltir. Eşikler interface frontmatter'ında yaşar (v2'de tasarım gereği
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

Tüketen bir takımın lens'i profilleri doğrudan okur: kimin var olduğunu görmek için `_index.md`'yi
istek üzerine yükler, sonra ihtiyaç duyduğu belirli `~/.atl/profiles/people/<slug>/profile.md`'yi
okur (profiller, wiki gibi düz markdown'dır). Index asla `CLAUDE.md`'ye enjekte edilmez — yalnızca
bir lens gerçekten kullanıcının insanları üzerine akıl yürütürken çekilir.

**Takımlar-arası erişim** açık değil, bildirimlidir. Üçüncü-taraf danışman takımlar geldiğinde,
her biri profil erişimini kendi `team.json`'ında `capabilities.profile` altında bildirir (`reads`
/ `writes` alan listeleri), kurulum anında yüzeye çıkarılır — böylece bir hukuk-danışma takımına,
bir esenlik takımından farklı erişim verilebilir. v1'de tek tüketici personal-advisory-team'dir;
sözleşme şimdi vardır ki üçüncü-taraf takımlar serbestçe okumadan önce yerinde olsun.

## v1'de ne geliyor

v1, **person** interface'ini ve tam döngüyü sağlar; kuzey-yıldızı tüketicisi
(personal-advisory-team) üzerinde doğrulanır. Mimari tür-açıktır — açıkça person olmayan bir varlık,
person'a uydurulmak yerine minimal bir `unknown` taslağı olarak tutulur.

Daha sonraki bir sürüme ertelenen (tasarımı yakalanmış): diğer varlık türleri (organizasyon,
hayvan, proje, yer, nesne) ve sıfırdan yeni bir interface yazma; zamanlanmış/aralıklı drain (bugün
oturum başlangıcında çalışır); profil ve proje dünyaları arasında yapılandırılmış çapraz-bağlantılar;
ve kırıcı şema değişiklikleri için tembel-*göç (migration)* uygulaması.

## Ayrıca bkz.

- [Öğrenme marker yaşam döngüsü](/tr/guide/learning-marker-lifecycle) — `profile-fact`'in yansıttığı kardeş döngü
- [Kavramlar](/tr/guide/concepts) — global ve proje katmanlarının nasıl etkileştiği
- [`atl learnings`](/tr/cli/learnings) — kuyruğu incele (`--channel profile-fact`)
- [`atl install`](/tr/cli/install) — bir takımın nasıl çözülüp kurulduğu
