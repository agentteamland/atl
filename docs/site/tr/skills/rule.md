# `/rule`

Bir kodlama ya da mimari kuralı ekle. Kullanıcı kuralı doğal dilde (herhangi bir dilde) anlatır; beceri bunu doğru dosyaya **İngilizce yapılandırılmış biçimde** yazar.

Birden çok ifadenin olası olduğu karmaşık ya da belirsiz kurallar için doğrudan [`/rule-wizard`](/tr/skills/rule-wizard) kullan — son biçimi yazmak için `/rule` becerisini çağırmadan önce seçenek tabanlı soru-yanıt turlarından geçirir.

Global beceri olarak [atl monorepo](https://github.com/agentteamland/atl) içinde yayımlanır.

## İki kapsam {#two-scopes}

| Bayrak | Hedef | Ne zaman |
|---|---|---|
| *(yok)* | Proje `.atl/` dosyaları | Bu projeye özgü kurallar (varsayılan). |
| `--global` | `~/.atl/rules/` | Her projeye uygulanan kişisel kurallar. |

::: tip Kural gerçekte nasıl yüklenir
`.atl/rules/` kalıcı kaynaktır. `atl session-start`, `.atl/rules/*.md` dosyalarını eşleşen `.claude/rules/` dizinine — Claude Code'un yüklediği yüzeye — yansıtır; böylece `.atl/rules/` içine yazılan bir kural bir sonraki oturumdan itibaren etkin olur. `.claude/rules/` kopyası türetilmiştir; `atl gc` onu ancak sen `.atl/rules/` kaynağını sildikten sonra geri kazanır. (`.atl/docs/coding-standards/` altındaki uygulamaya özgü dosyalar, her zaman yüklenen kurallar değil talep üzerine başvuru belgeleridir ve yansıtılmaz.)
:::

## Akış

### 1. Kuralı çözümle

Kullanıcının doğal dildeki ifadesinden çıkar:

- **Konu** — kodlama, mimari, adlandırma, hata yönetimi vb.
- **Kapsam** — hangi uygulama(ları) etkiler.
- **Gerekçe** — bu kuralın *neden*i (söylenmediyse makul bir Why türet; emin değilsen sor).

### 2. Hedef dosyayı belirle

**Proje kapsamı (varsayılan):**

| Uygulanabilirlik | Dosya |
|---|---|
| Tüm uygulamalar için ortak | `.atl/rules/coding-common.md`. |
| Belirli bir uygulama | `.atl/docs/coding-standards/{app}.md` (var olan dosyalardan seçilir). |

**Global kapsam (`--global`):**

| Uygulanabilirlik | Dosya |
|---|---|
| Genel kural | `~/.atl/rules/{topic}.md` (varsa eklenir, yoksa oluşturulur). |

Kural birden çoğuna uyuyor ama hepsine uymuyorsa beceri sorar.

### 3. Mevcut kuralları denetle

Hedef dosyayı **daima oku**. Üç durum vardır:

- **Tümüyle yeni bir kural** → yeni bir bölüm olarak ekle.
- **Var olan bir kuralı genişletme / güncelleme** → yerinde güncelle; yinelemeyi yapma.
- **Çelişki** (iki kural birbiriyle çelişiyorsa) → kullanıcıya sor; varsayma.

### 4. Yapılandırılmış biçimde yaz

Ayrıntılı ve açık biçimde, İngilizce yaz. **Eksik bir kural, hiç olmamış bir kuraldan daha tehlikelidir.**

```markdown
### {kebab-case-rule-id}
**Rule:** {Kuralın tek bir cümleyle açık ifadesi}

**Why:** {Gerekçe. Hangi sorunu önler? Hangi ilkeyi destekler?
Uygulanabiliyorsa geçmiş hatalardan çıkarılan dersler de eklenir. Bu alan boş ya da muğlak bırakılamaz.}

**Apply when:** {Hangi koşullarda — dosya yolları, kod desenleri,
ne tür değişiklikler? Belirgin ol.}

**Don't apply when:** {(İsteğe bağlı) İstisnaları açıkça belirt.}

**Examples:**
- ✅ Correct: {kod örneği ya da somut senaryo}
- ❌ Wrong: {kod örneği ya da somut senaryo}

**Related:** {(İsteğe bağlı) İlgili kural kimlikleri}
```

### 5. Kural yazımı (kritik)

- **Asla varsayma.** Bilgi eksikse sor.
- **Kısa tutma — açıkla.** Atlanan ayrıntı = uygulanmamış kural.
- **Sınır durumlarını yakala.** Uygulanabilir olduğunda `Don't apply when` alanını ekle.
- **Örnek ver.** Hem ✅ hem ❌.
- **Benzersiz bir kimlik ata.** Çakışmayı önlemek için önce dosyayı oku.

### 6. Yaz ve doğrula

Hedef dosyayı `Edit` ile güncelle. Kullanıcıya kısa bir özet ver: hangi dosya, hangi kimlik.

## Önemli kurallar

1. **Dil:** Kullanıcı beceriyi herhangi bir dilde çağırabilir; beceri kuralı **daima İngilizce yazar.**
2. **Bilgi eksikse sor.** Boşlukları kendi başına doldurma.
3. **Yineleme yapma.** Önce mevcut kuralları oku.
4. **Dosya yollarını doğrula.** Yanlış kapsam → yanlış dosya.
5. **Biçim sapması yok.** Tüm zorunlu alanlar doldurulur: Rule, Why, Apply when, Examples.

## İlgili

- [`/rule-wizard`](/tr/skills/rule-wizard) — belirsiz kurallar için seçenek tabanlı netleştirme sihirbazı; sonunda `/rule`'u çağırır.
- [Kavramlar: Kural](/tr/guide/concepts#rule) — kuralların ne olduğu ve nasıl yüklendiği.

## Kaynak

- Belirtim: [core/skills/rule/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rule/SKILL.md)
