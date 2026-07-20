# `/rule-wizard`

Bir kural eklemeden önce **seçenek tabanlı soru-yanıt turları** kullanan bir netleştirme sihirbazı. [`/rule`](/tr/skills/rule)'u doğrudan kullanırken kolayca gözden kaçan ayrıntıları yakalar — sınır durumları, istisnalar, alternatif ifade biçimleri, kapsam, gerekçe, örnek varyantları — ve konuşmanın aslında *birden çok* kuralla ilgili olduğunu devingen biçimde algılar.

Tartışma tamamlanınca sihirbaz, sonlandırılmış metni `/rule`'a verir; böylece kural doğru dosyaya yazılır.

Global beceri olarak [atl monorepo](https://github.com/agentteamland/atl) içinde yayımlanır.

## Hangi durumda hangisi?

| Beceri | Ne zaman |
|---|---|
| [`/rule`](/tr/skills/rule) | Kural açıktır; tek bir cümleyle, bilinen bir kapsam ve bilinen bir gerekçeyle ifade edebiliyorsun. |
| `/rule-wizard` | Kural bulanık. Alanı biliyorsun ama tam ifadeyi bilmiyorsun. Bir istisnayı, ilgili bir kuralı atlamadığından ya da aslında *iki* kuralın tek bir şemsiyenin altına gizlenmediğinden emin olmak istiyorsun. |

## Zorunlu argüman

`/rule-wizard` bir **bağlam** ister — konuyu, başlangıç fikrini ya da karşılaşılan sorunu anlatan kısa bir metin (herhangi bir dilde):

- ✅ `/rule-wizard API'de logging kullanımı`
- ✅ `/rule-wizard Worker doğrudan DB'ye bağlanmamalı`
- ✅ `/rule-wizard Controller'lar try-catch yazmamalı; global handler devralır`
- ❌ `/rule-wizard` — bağlam olmadan çağrıldı, sihirbaz reddeder ve bağlam ister.

## İki kapsam

[`/rule`](/tr/skills/rule#two-scopes) ile aynıdır: proje (varsayılan) ve `--global`. Bayrak verildiğinde sihirbaz kapsam sorusunu atlar.

## Aşama 1 — anlama ve hazırlık

### 1.1 Mevcut kuralları oku (zorunlu)

Soru sormadan **önce** şunları oku:

- `.atl/rules/coding-common.md`.
- `.atl/docs/coding-standards/` altındaki tüm `.md` dosyalar (devingen biçimde listelenir).

Amaç:

- **Yinelemeden kaçınma** — aynı ya da çok benzer bir kural zaten var mı?
- **Çelişki algılama** — yeni kural mevcut bir kuralla çelişiyor mu?
- **Genişletme fırsatı** — mevcut bir kurala madde olarak eklenebilir mi?
- **Çapraz başvuru** — son metnin `Related` alanını besleyebilecek kural kimlikleri.

### 1.2 Bağlamı çözümle (sessizce)

Kullanıcının bağlamından içsel olarak türet:

- Olası kapsam (hangi uygulama(lar)? ortak mı tekil mi?).
- Olası niyet (zorunlu `must` / yasaklayıcı `must not` / öneri `should`?).
- Etkilenen katmanlar (Controller, Service, Repository, Consumer, Hub, Job?).
- `Apply when`, `Why` ve `Examples` için ilk varsayımlar.
- 1.1'den çıkan benzer mevcut kuralların kimlikleri.

### 1.3 Çözümleme özetini sun

Kullanıcıya kısa bir paragraf:

> "Anladığım kadarıyla API controller'larının try-catch bloğu içermesini istemiyorsun ve hata yönetiminin üst katmandaki global handler'a devredilmesini istiyorsun. Bu `coding-standards/api.md` kapsamına düşüyor ve mevcut `no-logic-in-bridges` kuralını tamamlıyor. Şimdi birkaç soruyla ayrıntıları netleştireceğim."

Ardından Aşama 2'ye geç.

## Aşama 2 — seçenek tabanlı sorgulama

### Temel ilkeler (hepsi bağlayıcı)

1. **Her soru `AskUserQuestion` kullanır.** Düz metin, açık uçlu soru yok.
2. **Her sorunun 2-4 seçeneği vardır.** Platform sınırı 4'tür. Daha makul seçenek varsa soruyu **böl**; "en iyi 4" ile kendini sınırlama.
3. **"Other" seçeneği kendiliğinden eklenir.** Araç bunu ekler; açıkça yazma.
4. **Önerilen bir seçenek varsa onu başa koy** — etiketinde `(Recommended)`, `description` alanında kısa bir gerekçe.
5. **Soruları ve seçenekleri kullanıcının diliyle eşle.** `/rule` son adımda İngilizceye çevirir; sihirbaz kullanıcının dilini aynalar.
6. **Tur başına en çok 4 soru.** 4'ten fazlaysa → turlara böl; sonraki turlar önceki yanıtları kullanabilir.
7. **Bağlamdan açıkça türetilen alanları yeniden sorma.** Onun yerine doğrulama sorusu sor: "Bunu X olarak anladım — doğru mu?" (evet/hayır).
8. **Seçenekler ayrı ve net olmalı.** İki seçenek neredeyse aynıysa birini kaldır.

### Kapsanacak alanlar

Her kural için (soru-yanıt yoluyla) netleştir:

#### A) Kapsam — nereye yazılacak?

(`--global` ayarlıysa atlanır.)

Proje kapsamı seçilirse alt soru: hangi uygulama? (Mevcut `.atl/docs/coding-standards/{app}.md` dosyalarını devingen biçimde listeler artı bir "common" seçeneği.)

#### B) Tek cümlelik kural ifadesi

Bağlamdan üç alternatif ifade; her biri farklı tonda / kısıtlamada:

- **Kesin yasak** ("X asla yapılmamalı").
- **Öneri** ("X için Y kullan").
- **Koşullu** ("X yalnızca Y olduğunda yapılabilir").

Kullanıcı kendi cümlesini "Other" üzerinden yazabilir.

#### C) Gerekçe (Why)

Bağlama göre üretilen listeden çoklu seçim: geçmiş hatadan çıkan ders (belirt), mimari tutarlılık, test edilebilirlik, başarım, güvenlik, okunabilirlik, mevzuat. Birincil gerekçe başa konur.

#### D) Apply when (tetikleme koşulları)

Bağlamdan 2-4 belirgin tetikleyici. Her seçenek **somut bir dosya yolu ya da kod deseni** içerir (örneğin controller eylemleri için `api/Controllers/*.cs`). Tetikleyiciler üst üste binebiliyorsa çoklu seçim olur.

#### E) Don't apply when (istisnalar, isteğe bağlı)

Seçenekler şunlardır: istisna yok / test kodu muaf / eski ya da otomatik üretilen kod muaf / Other. "İstisna yok" seçilirse alan son kuraldan çıkarılır.

#### F) Örnekler

İki soru: ✅ doğru örnek + ❌ yanlış örnek. Her biri 2-3 kısa senaryo sunar; kullanıcı birini seçer ya da kendi yazar.

#### G) İlgili kurallar (isteğe bağlı)

1.1'de bulunan benzer kural kimliklerini ve bir "None" seçeneğini listeler.

### Önerilen turlar

- **Tur 1 (temeller):** Kapsam + Kural ifadesi + Gerekçe — 3 soru.
- **Tur 2 (davranış):** Apply when + İstisna + ✅ Örnek — 3 soru.
- **Tur 3 (cila):** ❌ Örnek + İlgili kurallar + (gerekirse) sınır durumu — 2-3 soru.

Alanlar açıkça türetildikçe turlar küçülür; belirsizlik sürdükçe ek sorular eklenir. Kullanıcı bir tur içinde aynı yanıtı iki kez vermek zorunda bırakılmaz.

## Aşama 3 — devingen çoklu kural algılama

Sihirbaz **tek kural varsayımıyla başlar.** Ama sorgulama sırasında şu sinyallerden biri görülürse hemen kullanıcıya bir ayırt etme sorusu sor:

| Sinyal | Önerdiği |
|---|---|
| Kapsamda iki farklı uygulama seçildi ve doğaları farklı. | Tek görünen ama aslında iki kural. |
| Apply-when tetikleyicileri birbirinden bağımsız iki kod katmanına işaret ediyor. | İki kural. |
| Kural ifadesi iki bağımsız yasağı içeriyor ("X yapılmamalı, Y de yapılmamalı"). | İki kural. |
| Örnekler tek bir kuralla açıklanamıyor. | İki kural. |
| Gerekçe çoklu seçiminde iki bağımsız gerekçe seçildi. | İki kural. |

### Ayırt etme sorusu

```
"Bu bağlam aslında iki farklı kural gibi duruyor. Nasıl ilerleyelim?"
```

Seçenekler:

- **(Recommended)** İki ayrı kural olarak ekle — her birini ayrı ayrı netleştirelim.
- Tek kural olarak tut — Kural ifadesini iki maddeyi de kapsayacak biçimde genişlet.
- Şimdilik birine odaklan, ötekini sonra ele al.
- Yanlış algılandı — bu aslında tek kural.

### Karar sonrası akış

- **İki ayrı kural** → her biri için Aşama 2'yi bağımsız olarak yinele. Soru turlarını karıştırma.
- **Tek kural olarak tut** → Kural ifadesi sorusunu iki maddeyi birleştiren ifadelerle yeniden sor.
- **Yalnızca birine odaklan** → Aşama 4 sonunda "Öteki kurala şimdi geçelim mi?" diye sor.
- **Yanlış algılandı** → olağan akışa dön, sinyali yoksay.

## Aşama 4 — bütünleştirme ve son onay

### 4.1 Son kural metnini üret

Toplanan yanıtlardan kullanıcının dilinde **doğal dilde** bir kural metni oluştur. Bu metin, **`/rule` becerisi için girdidir** — `/rule` İngilizceye çeviriyi ve yapılandırılmış biçim ayrıştırmasını yapar.

Metin, `/rule`'un ayrıştıracağı her şeyi içermelidir: Kapsam, Rule, Why, Apply when, Don't apply when (uygulanabilirse), Examples (✅ + ❌), Related (uygulanabilirse).

Örnek (TR dilinde sihirbaz, İngilizce sonlu kural):

> "Controller actions in the API project should not write try-catch blocks — error handling must be delegated to the global exception handler at the upper layer. This rule exists to preserve architectural consistency and to keep controllers as thin bridges; try-catch is the responsibility of services or the global handler. Applies to: all controller actions in `.cs` files under `api/Controllers/`. Test code is exempt. Correct example: `[HttpPost] public async Task<IActionResult> Create(CreateProductRequest req) { var result = await _productService.CreateAsync(req); return Ok(result); }` — no try-catch. Wrong example: writing `try { ... } catch (Exception ex) { return BadRequest(ex.Message); }` inside a controller. Related rule: `no-logic-in-bridges`."

### 4.2 Kullanıcıya göster ve onay al

Son kural metnini göster ve `AskUserQuestion` ile sor:

```
"Bu metin kuralın son hâli mi? Şimdi /rule ile ekleyeyim mi?"
```

Seçenekler:

- **(Recommended)** Evet, `/rule` ile ekle.
- Metnin bir bölümünü düzeltmem gerekiyor — hangi bölüm söyleyeyim.
- Eksik bir alan var sanırım — başka bir soru turu yapalım.
- İptal et, şimdilik ekleme.

### 4.3 `/rule`'u çağır

Kullanıcı onay verdiğinde:

- **Tek kural** → `/rule <son metin>` komutunu çağır.
- **Birden çok kural** → her birini **sırayla** çağır; kısa ilerleme bildirimi ile: "İlk kural yazıldı (`{id}` → `{file}`). Şimdi ikinci kurala geçiyorum."
- Her `/rule` çağrısından sonra sonucu özetle.

Kullanıcı düzeltme isterse → ilgili soruyu yeniden sor, son metni güncelle, 4.2'ye dön.

Kullanıcı ek bir tur isterse → eksik alan için bir soru turu çalıştır, 4.1'e dön.

Kullanıcı iptal ederse → temiz biçimde sonlandır. Hiçbir dosyaya yazma. "Kural yazılmadı. İstediğin zaman `/rule-wizard` ile yeniden başlayabilirsin." diye bildir.

### 4.4 Son özet

Sonunda tek bir mesaj:

- Kaç kural yazıldı.
- Her kuralın kimliği ve dosyası.
- İlgili olarak işaretlenen mevcut kurallar (varsa).
- Aşama 3'te sonraya bırakılan ertelenmiş bir kural varsa anımsatma.

## Kritik ilkeler

1. **Bağlam zorunludur.** Beceri argümansız çalışmaz.
2. **Her sorunun seçenekleri vardır.** Yalnızca `AskUserQuestion`.
3. **4 seçenek yetmiyorsa böl.** Asla "en iyi 4" ile sınırlama.
4. **Mevcut kuralları önce oku.** Zorunlu önkoşul.
5. **Asla varsayma.** Bağlamdan açıkça türetilemeyen her alan soru ister.
6. **Çoklu kuralları devingen biçimde algıla.** Tek başla; ayrışma sinyalleri belirdiğinde sor.
7. **Son metin kullanıcının dilinde olur.** `/rule` İngilizceye çevirir.
8. **Onay olmadan `/rule` çağrılmaz.** Kullanıcı son metni görür ve onaylar.
9. **Eksik alan, hiç olmamış alandan kötüdür.** Zorunlu alanlar (Rule, Why, Apply when, Examples) eksiksiz temsil edilmelidir.
10. **Beceri birden çok kural için yinelemeli çalışabilir.** Aşama 3'te bölme kipi seçildiyse her kural Aşama 2-4 döngüsünden ayrı geçer.

## İlgili

- [`/rule`](/tr/skills/rule) — Aşama 4'te sonlandırılan kuralı yazmak için çağrılır.
- [Kavramlar: Kural](/tr/guide/concepts#rule) — kuralların ne olduğu ve nasıl yüklendiği.

## Kaynak

- Belirtim: [core/skills/rule-wizard/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rule-wizard/SKILL.md).
