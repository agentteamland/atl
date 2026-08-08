# `/brainstorm`

Beyin fırtınası oturumlarını başlat ve tamamla. Beyin fırtınaları, kod yazılmadan önce önemli bir kararı düşünmenin kanonik yeridir — sonuçta oluşan dosya, hem tarihsel kayıttır hem de konuyu sonradan ele alacak herhangi bir oturum için aktarımdır.

İki kip: `start` yeni bir beyin fırtınası açar; `done` etkin beyin fırtınasını tamamlar ve kararlarını belge zincirine yayar.

Global beceri olarak [ATL monoreposunda](https://github.com/agentteamland/atl) yayımlanır.

## İki kapsam

Bir beyin fırtınası iki düzeyden birinde yaşar — *kimin* karara önem vereceğine göre kapsamı seç:

| Bayrak | Hedef dizin | Ne zaman |
|---|---|---|
| *(yok)* | `.atl/brain-storms/` | Projeye özgü konular (varsayılan). |
| `--global` | `~/.atl/brain-storms/` | Projeler arası, kişisel konular. |

## `start` kipi

```
/brainstorm start [--global] <konuyu açıklayan ilk mesaj>
```

Akış:

1. **Konuyu çıkar** — kullanıcının mesajından kebab-case bir dosya adı türet. Kullanıcı dosya adını VERMEZ.
2. **Kapsamı belirle** — bayraktan al (yoksa varsayılan olarak proje).
3. **Dizini oluştur** — yoksa (`{scope-base}/brain-storms/`) dizinini aç.
4. **Beyin fırtınası dosyasını oluştur** — frontmatter (`status: active`, `scope`, `date`, `participants`) + Context + Discussion + Open Items bölümleriyle.
5. **Beyin fırtınasını kapsamın `CLAUDE.md` dosyasına sabitle** — bir `<!-- brainstorm:active -->` işaretçi bloğu üzerinden. Bu, etkin beyin fırtınasının bir sonraki oturumda kaçırılmasını olanaksız kılar — proje bağlamıyla birlikte kendiliğinden yüklenir.
6. **Kullanıcıya bildir** — dosya adı, kapsam ve sabitlenen konum; ardından konunun içine dal.

### Etkin beyin fırtınası sabitleyicisi

Her etkin beyin fırtınası kendisini, kapsamın `CLAUDE.md` dosyasında bir `<!-- brainstorm:active:start --> ... <!-- brainstorm:active:end -->` bloğu içine sabitler:

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[profile-team](.atl/brain-storms/profile-team.md)** (project, 2026-05-08) — schema, storage, privacy, and initial scope for the new profile-team package
<!-- brainstorm:active:end -->
```

Birden çok etkin beyin fırtınası, aynı blok içinde madde olarak yan yana yaşar. `brainstorm@1.1.0` ile yayımlandı.

### Beyin fırtınasını canlı tutmak (her mesaj turu)

Bir beyin fırtınası etkinken her mesajda:

- **Yanıt vermeden önce** — beyin fırtınası dosyasını oku (bağlamı yeniden hatırla).
- **Yanıt verdikten sonra** — yeni fikirleri, kararları, reddedilen alternatifleri ve gerekçelerini, kullanıcının önemli noktalardaki birebir ifadelerini, açık soruları ve kronolojik akışı dosyaya yaz.

Dosya, **yeterince ayrıntılı** olmalıdır; yeni bir bağlamda dosyayı okuyan bir Claude, özgün konuşmada bulunmuş gibi devam edebilmelidir.

## `done` kipi

```
/brainstorm done
```

Akış:

1. **Etkin beyin fırtınasını bul.** Her iki kapsamı da tarar (`.atl/brain-storms/` ve `~/.atl/brain-storms/`). Birden çoksa bunları kapsamlarıyla birlikte listeler ve hangisinin tamamlanacağını sorar.
2. **Beyin fırtınası dosyasını tamamla.** `status: active` → `status: completed`. Son notları sona ekle. Open Items bölümünü güncelle (çözülmemişler kalır). Final Decisions bölümünü ekle.
3. **Belge dosyasını oluştur ya da güncelle.** Yerleşmiş kararlar şu yerlere gider:
   - **Proje beyin fırtınası** → `.atl/docs/`.
   - **Global beyin fırtınası** → `~/.atl/docs/`.
4. **`CLAUDE.md` güncellenir.** En fazla üç şey olur:
   - Tamamlanmış beyin fırtınası özeti uygun bölüme eklenir.
   - Bu beyin fırtınasının maddesi `<!-- brainstorm:active -->` işaretçi bloğundan kaldırılır. Madde listesi boşalırsa blok tümüyle kaldırılır (geride bayatlamış bir "Active brainstorms" başlığı kalmaz).
   - Karar implementasyonsuz iş bırakıyorsa, bir sonraki oturumun kuyruğu görmesi için `<!-- pending-implementation -->` bloğuna bir madde eklenir (saf-karar beyin fırtınalarında atlanır; implementasyon yayımlanınca kaldırılır).

## Belge zinciri

Her tartışma ve karar üç katmandan akar; sürece iki karar-durumu dosyası da bağlanır:

```
brain-storms/ (süreç) → docs/ (sonuç) → CLAUDE.md (özet)
                     \
                       backlog.md (ertelenmiş üst-küme) → tasks.md (etkin-niyet alt-kümesi)
```

- Beyin fırtınası olmadan karar verilmez.
- Beyin fırtınası dosyaları **asla silinmez** — tarihsel kayıttır.
- Kararlar değişirse YENİ bir beyin fırtınası açılır ve eskisine `superseded by X` notu eklenir.

## Backlog ve tasks

Bir projenin `.atl/` dizini altındaki iki dosya **karar durumunu** tutar — bilgi sisteminin journal + wiki katmanının bir kardeşidir, üçüncü bir bilgi katmanı değil. Bu dosyaları `/brainstorm` becerisi yazar ve güncel tutar. Tam sözleşme için [Backlog ve tasks](../guide/backlog-and-tasks.md) rehberine bakın.

- **`backlog.md`** — ertelenen, bir kenara bırakılan ya da belirsiz kalan her şeyin edilgen, **tetik-kapılı üst-kümesi**. Yapılacaklar listesi değil, taranabilir bir dizin. `## Area` başlıklarına göre gruplanır (temaya göre, tarihe göre değil). Öğe başına tek satır: `- **Başlık** — tek cümle. _Trigger:_ ne zaman yeniden gündeme gelir. ↳ [kaynak](...)`. Zengin gerekçe/bağlam, bağlanan beyin fırtınasında kalır (backlog dizindir, beyin fırtınası ayrıntıdır — tekrar yok).
- **`tasks.md`** — **etkin-niyet alt-kümesi**: gerçekten sıradaki adım olarak yapmayı düşündüğümüz kısa, önceliklendirilmiş liste. `- [ ] **Başlık** — tek cümle. ↳ [kaynak](...)`, `## Now` / `## Next` altında gruplanır. Kısa ve dürüst tutulur: planlanan bir şey yoksa neredeyse boştur (doğru durum budur, bir eksiklik değil) — görev uydurmayın.

**Terfi.** Bir öğe, onu öne çektiğimize karar verdiğimizde (bir tetik ateşlendi ya da önceliklendirmeyi seçtik) `backlog.md` → `tasks.md` yönünde taşınır. Bir öğe `backlog.md`'den, yayımlandığında **ya da** `tasks.md`'ye terfi edildiğinde **çıkar**; bir görev `tasks.md`'den yayımlandığında **çıkar** (kaynak-doğru artık docs + CLAUDE.md olur) — silinir, işaretlenip bırakılmaz.

**`/brainstorm done` denetimleri.** Bir beyin fırtınasını kapatmak zorunlu bir **backlog denetimi** (her ertelenmiş öğe kendi `## Area` grubunun altına bir kayıt alır) ve bir **tasks denetimi** (şimdi yapmaya karar verilen her şeyi terfi ettir; bu beyin fırtınasının yayımladığı her şeyi kaldır) çalıştırır.

**İskele.** `atl init` (ve `atl install`), yalnızca yoksa `.atl/` altına boş `backlog.md` + `tasks.md` iskeletleri bırakır (kullanıcının mevcut dosyası asla üzerine yazılmaz). Global katmanın proje `.atl/` dizini olmadığından, orada atlanır.

## Önemli kurallar

1. **Birden çok etkin beyin fırtınası var olabilir.** Her biri kendi dosyasında yaşar. Kapsamlar arasında eş zamanlı etkin olabilirler.
2. **Bağlam kopukluklarına dayanıklılık.** Beyin fırtınası dosyası kalıcı durumdur. Yeni bir oturum, etkin beyin fırtınalarını işaretçi bloğu ile dizin taraması yoluyla algılar ve dosyayı okuyarak sürdürür.
3. **Dosya adı kullanıcıdan istenmez.** Mesajdan çıkarılır ve uygun bir kebab-case ad atanır.
4. **Beyin fırtınası dosyaları asla silinmez.** Tarihsel kayıttır.
5. **Her beyin fırtınası tek konuya odaklanır.** Farklı konular → farklı dosyalar.
6. **Etkin beyin fırtınası araması her iki kapsamı da kapsar.** `done` kipinde proje ve global taranır.
7. **Kapsam frontmatter'da yer alır.** `scope: project|global` — `done` kipinin hedeflerini belirler.

## İlgili

- [`/drain`](/tr/skills/drain) — düzenli öğrenme yakalama (beyin fırtınasına paralel; beyin fırtınaları kasıtlıdır, öğrenmeler kendiliğinden olur).
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — beyin fırtınalarının bilgi modelinde nereye oturduğu.

## Kaynak

- Belirtim: [core/skills/brainstorm/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/brainstorm/SKILL.md)
- Kural: [core/rules/brainstorm.md](https://github.com/agentteamland/atl/blob/main/core/rules/brainstorm.md)
