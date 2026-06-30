# `/brainstorm`

Beyin fırtınası oturumlarını başlat ve tamamla. Beyin fırtınaları, kod yazılmadan önce önemli bir kararı düşünmenin kanonik yeridir — sonuçta oluşan dosya, hem tarihsel kayıttır hem de konuyu sonradan ele alacak herhangi bir oturum için aktarımdır.

İki kip: `start` yeni bir beyin fırtınası açar; `done` etkin beyin fırtınasını tamamlar ve kararlarını belge zincirine yayar.

Global beceri olarak [ATL monoreposunda](https://github.com/agentteamland/atl) yayımlanır.

## Üç kapsam

Bir beyin fırtınası üç düzeyden birinde yaşar — *kimin* karara önem vereceğine göre kapsamı seç:

| Bayrak | Hedef dizin | Ne zaman |
|---|---|---|
| *(yok)* | `.atl/brain-storms/` | Projeye özgü konular (varsayılan). |
| `--global` | `~/.atl/brain-storms/` | Projeler arası, kişisel konular. |
| `--team` | `<proje>/.atl/brain-storms/` (takım kapsamlı alt dizin) | Kurulu bir takımla ilgili konular (ajan kuralları, takım stratejisi, katkıcı yönetişimi). |

`--team` için etkin takım, kurulu `.claude/agents/` girişlerinden algılanır. Tek takım → kendiliğinden kullanılır; birden çok takım → beceri `AskUserQuestion` ile sorar; kurulu takım yok → `atl install` ipucuyla bir hata verir.

## `start` kipi

```
/brainstorm start [--global|--team] <konuyu açıklayan ilk mesaj>
```

Akış:

1. **Konuyu çıkar** — kullanıcının mesajından kebab-case bir dosya adı türet. Kullanıcı dosya adını VERMEZ.
2. **Kapsamı belirle** — bayraktan al (yoksa varsayılan olarak proje).
3. **Dizini oluştur** — yoksa (`{scope-base}/brain-storms/`) dizinini aç.
4. **Beyin fırtınası dosyasını oluştur** — frontmatter (`status: active`, `scope`, `team`, `date`, `participants`) + Context + Discussion + Open Items bölümleriyle.
5. **Beyin fırtınasını kapsamın `CLAUDE.md` / `README.md` dosyasına sabitle** — bir `<!-- brainstorm:active -->` işaretçi bloğu üzerinden. Bu, etkin beyin fırtınasının bir sonraki oturumda kaçırılmasını olanaksız kılar — proje bağlamıyla birlikte kendiliğinden yüklenir.
6. **Kullanıcıya bildir** — dosya adı, kapsam ve sabitlenen konum; ardından konunun içine dal.

### Etkin beyin fırtınası sabitleyicisi

Her etkin beyin fırtınası kendisini, kapsamın `CLAUDE.md` (ya da takım `README.md`) dosyasında bir `<!-- brainstorm:active:start --> ... <!-- brainstorm:active:end -->` bloğu içine sabitler:

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

1. **Etkin beyin fırtınasını bul.** Üç kapsamı da tarar (`.atl/brain-storms/`, `~/.atl/brain-storms/` ve `.atl/brain-storms/` altındaki takım kapsamlı alt dizinler). Birden çoksa bunları kapsamlarıyla birlikte listeler ve hangisinin tamamlanacağını sorar.
2. **Beyin fırtınası dosyasını tamamla.** `status: active` → `status: completed`. Son notları sona ekle. Open Items bölümünü güncelle (çözülmemişler kalır). Final Decisions bölümünü ekle.
3. **Belge dosyasını oluştur ya da güncelle.** Yerleşmiş kararlar şu yerlere gider:
   - **Proje beyin fırtınası** → `.atl/docs/`.
   - **Global beyin fırtınası** → `~/.atl/docs/`.
   - **Takım beyin fırtınası** → `<proje>/.atl/docs/` (takım kapsamlı alt dizin).
4. **`CLAUDE.md` / `README` güncellenir.** En fazla üç şey olur:
   - Tamamlanmış beyin fırtınası özeti uygun bölüme eklenir.
   - Bu beyin fırtınasının maddesi `<!-- brainstorm:active -->` işaretçi bloğundan kaldırılır. Madde listesi boşalırsa blok tümüyle kaldırılır (geride bayatlamış bir "Active brainstorms" başlığı kalmaz).
   - Karar implementasyonsuz iş bırakıyorsa, bir sonraki oturumun kuyruğu görmesi için `<!-- pending-implementation -->` bloğuna bir madde eklenir (saf-karar beyin fırtınalarında atlanır; implementasyon yayımlanınca kaldırılır).
5. **Takım beyin fırtınalarını doğrudan push'la değil, PR ile yayımla.** Takım beyin fırtınaları takımın yerel klonunda yaşar ve takım depoları dal korumalıdır. `done` akışı dosyayı yerelde yazar ve kullanıcıya bir PR açmasını söyler (elle ya da [`/create-pr`](/tr/skills/create-pr) ile).

## Belge zinciri

Her tartışma ve karar üç katmandan akar:

```
brain-storms/ (süreç) → docs/ (sonuç) → CLAUDE.md (özet)
                     \
                       backlog.md (ertelenmiş öğeler)
```

- Beyin fırtınası olmadan karar verilmez.
- Beyin fırtınası dosyaları **asla silinmez** — tarihsel kayıttır.
- Kararlar değişirse YENİ bir beyin fırtınası açılır ve eskisine `superseded by X` notu eklenir.

## Backlog disiplini

Bir beyin fırtınası sırasında "şimdi yapmıyoruz, sonraya" diye işaretlenen her öğe `.atl/backlog.md` dosyasına yansır:

- **Başa eklenir** (en yenisi en üstte).
- Her öğe için: tarih + kategori başlığı + bağlam bağı + ayrıntılı konu açıklaması + "ne zaman gündeme gelir" notu + ilgili kaynaklar.
- `/brainstorm done` sırasında beyin fırtınasındaki her "ertelenmiş" not backlog'a karşı denetlenir — eksik olanlar beyin fırtınası kapanmadan önce eklenir.
- Bir ertelenmiş öğe daha sonra hayata geçirildiğinde backlog'dan **silinir** (tamamlandı işareti konulup bırakılmaz).

## Önemli kurallar

1. **Birden çok etkin beyin fırtınası var olabilir.** Her biri kendi dosyasında yaşar. Kapsamlar arasında eş zamanlı etkin olabilirler.
2. **Bağlam kopukluklarına dayanıklılık.** Beyin fırtınası dosyası kalıcı durumdur. Yeni bir oturum, etkin beyin fırtınalarını işaretçi bloğu ile dizin taraması yoluyla algılar ve dosyayı okuyarak sürdürür.
3. **Dosya adı kullanıcıdan istenmez.** Mesajdan çıkarılır ve uygun bir kebab-case ad atanır.
4. **Beyin fırtınası dosyaları asla silinmez.** Tarihsel kayıttır.
5. **Her beyin fırtınası tek konuya odaklanır.** Farklı konular → farklı dosyalar.
6. **Etkin beyin fırtınası araması üç kapsamı da kapsar.** `done` kipinde proje, global ve takım kapsamlı alt dizinler taranır.
7. **Kapsam frontmatter'da yer alır.** `scope: project|global|team`, `team: {ad}` — `done` kipinin hedeflerini belirler.
8. **Takım beyin fırtınaları doğrudan push'la değil, PR ile yayımlanır.** Takım depoları dal korumalıdır; `done` akışı yerelde yazar ve PR oluşturmaya yönlendirir.

## İlgili

- [`/drain`](/tr/skills/drain) — düzenli öğrenme yakalama (beyin fırtınasına paralel; beyin fırtınaları kasıtlıdır, öğrenmeler kendiliğinden olur).
- [`/create-pr`](/tr/skills/create-pr) — bir takım beyin fırtınası değişikliğini PR olarak paketler.
- [Kavramlar: Beceri](/tr/guide/concepts#skill) — beyin fırtınalarının bilgi modelinde nereye oturduğu.

## Kaynak

- Belirtim: [core/skills/brainstorm/skill.md](https://github.com/agentteamland/atl/blob/main/core/skills/brainstorm/skill.md)
- Kural: [core/rules/brainstorm.md](https://github.com/agentteamland/atl/blob/main/core/rules/brainstorm.md)
