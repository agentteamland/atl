# `/rule`

Bir kodlama ya da mimari kuralı ekle. Kullanıcı kuralı doğal dilde (herhangi bir dilde) anlatır; beceri bunu doğru dosyaya **İngilizce yapılandırılmış biçimde** yazar.

Birden çok ifadenin olası olduğu karmaşık ya da belirsiz kurallar için doğrudan [`/rule-wizard`](/tr/skills/rule-wizard) kullan — son biçimi yazmak için `/rule` becerisini çağırmadan önce seçenek tabanlı soru-yanıt turlarından geçirir.

Global beceri olarak [atl monorepo](https://github.com/agentteamland/atl) içinde yayımlanır.

## Üç kapsam

| Bayrak | Hedef | Ne zaman |
|---|---|---|
| *(yok)* | Proje `.claude/` dosyaları | Bu projeye özgü kurallar (varsayılan). |
| `--global` | `~/.claude/rules/` | Her projeye uygulanan kişisel kurallar. |
| `--team` | Etkin kapsamın `.claude/` dizini altındaki takım dosyaları | Kurulu bir takıma ait ajan ya da takım kuralı dosyaları. |

`--team` için etkin takım, kurulu `.claude/agents/` dosyalarından algılanır. Tek takım → kendiliğinden kullanılır; birden çok takım → `AskUserQuestion` ile sorulur.

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
| Tüm uygulamalar için ortak | `.claude/rules/coding-common.md`. |
| Belirli bir uygulama | `.atl/docs/coding-standards/{app}.md` (var olan dosyalardan seçilir). |

**Global kapsam (`--global`):**

| Uygulanabilirlik | Dosya |
|---|---|
| Genel kural | `~/.claude/rules/{topic}.md` (varsa eklenir, yoksa oluşturulur). |

**Takım kapsamı (`--team`):**

| İlgili alan | Dosya |
|---|---|
| Bir ajanın bilgi tabanı | `.claude/agents/{agent}.md` (etkin kapsam altında). |
| Takım çapında kural | `.claude/rules/{topic}.md` (etkin kapsam altında). |

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

### 7. Takım kapsamlı kuralları kalıcılaştırma

`--team` ile yazılan takım kuralları, takımın kurulu varlık dosyalarına — etkin kapsamın `.claude/` dizinine — kaydedilir. Bunlar yerel kopyalardır; takımın kaynak deposuna otomatik olarak geri gönderilmez.

Bir kuralı upstream'e katkı olarak sunmak için doğrudan takımın kaynak deposuna karşı bir PR aç. Kuruluysa [`/create-pr`](/tr/skills/create-pr) bunu otomatikleştirir.

## Önemli kurallar

1. **Dil:** Kullanıcı beceriyi herhangi bir dilde çağırabilir; beceri kuralı **daima İngilizce yazar.**
2. **Bilgi eksikse sor.** Boşlukları kendi başına doldurma.
3. **Yineleme yapma.** Önce mevcut kuralları oku.
4. **Dosya yollarını doğrula.** Yanlış kapsam → yanlış dosya.
5. **Biçim sapması yok.** Tüm zorunlu alanlar doldurulur: Rule, Why, Apply when, Examples.
6. **Takım kapsamlı kurallar doğrudan push'la değil, PR ile yayımlanır.** Dal korumalıdır; beceri yerelde yazar ve PR oluşturmaya yönlendirir.

## İlgili

- [`/rule-wizard`](/tr/skills/rule-wizard) — belirsiz kurallar için seçenek tabanlı netleştirme sihirbazı; sonunda `/rule`'u çağırır.
- [Kavramlar: Kural](/tr/guide/concepts#rule) — kuralların ne olduğu ve nasıl yüklendiği.

## Kaynak

- Belirtim: [core/skills/rule/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rule/SKILL.md)
