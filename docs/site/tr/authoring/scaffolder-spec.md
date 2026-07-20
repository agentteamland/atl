# İskele belirtimi

Bir **iskele**, takımın yığınında yeni bir projeyi başlatan, `/create-new-project` adıyla anılan takım kapsamlı bir beceridir. Bu sayfa, her iskelenin uyması gereken standart şekli tanımlar.

::: tip Kanonik kaynak
Bu sayfa, belirtimin kanonik sürümüdür. (Daha eski bir kopyası arşivlenmiş v1 `agentteamland/core` deposunda yaşıyordu; o kopya yalnızca tarihseldir.)
:::

## Neden bir belirtim?

Farklı takımlar çok farklı iskeleler kuracaktır (.NET + Docker yığını ile Next.js + Sanity blog ile Python + Jupyter veri projesi karşılaştırılırsa). Ama UX'in **şekli** tutarlı olmalı. Bir takımın iskelesini öğrenen kullanıcı bir başkasını kullanırken kendini evinde hissetmeli.

## Beceri konumu

Takım deposunun içinde:

```
{team-repo}/skills/create-new-project/SKILL.md
```

Ve `team.json` içinde:

```json
{
  "skills": [
    { "name": "create-new-project", "description": "Scaffold a new project on this team's stack." }
  ]
}
```

Kullanıcı `atl install <team>` çalıştırdığında beceri kendiliğinden `.claude/skills/` dizinine kopyalanır.

## Beş aşama

Her iskele şu aşamalardan sırayla geçmek ZORUNDADIR:

### Aşama 1 — Bilgi topla

Gereksinimleri `AskUserQuestion` aracıyla topla. Tipik sorular:

- Proje adı (argüman olarak verildiyse sormayı atla).
- Hangi uygulamaların / modüllerin / özelliklerin dâhil edileceği.
- Dağıtım hedefleri, port ofsetleri.
- Lisans seçimi.
- Yığına özgü açma-kapama düğmeleri (SaaS mi? Çoklu kiracı mı? Çerçeve sürümü?).

**Kurallar:**

- Soruları odaklı tut — çoğu iskele için 4-6 yeterlidir.
- Makul varsayılanlar sağla; en üst seçeneği "Önerilen" olarak işaretle.
- Kullanıcı bir soruyu yanıtlayan bir argüman geçirdiyse o soruyu sorma.

### Aşama 2 — Projeyi iskeleyle kur

Yeni projenin ihtiyaç duyduğu her dosyayı yaz. Büyük iskeleler için paralel alt ajanlara devret — her biri bir ana konuya (API, ön uç, altyapı, mobil vb.) odaklanır.

**Kontrol listesi:**

- Kök dosyalar (`README.md`, `.gitignore`, dile özgü kilit dosyaları).
- Proje yapılandırması (`CLAUDE.md`, `.mcp.json`, `.env.example`).
- `.claude/` proje dizini (`agents/`, `skills/`, `rules/`).
- `.atl/` proje dizini (`docs/`, `brain-storms/`, `wiki/`, `journal/`, `backlog.md`, `tasks.md`) — bilgi + karar katmanı.
- Kaynak ağacı (uygulamanın gerçekten çalışmasını sağlayan her şey).
- Konteyner / dağıtım yapılandırması (varsa).

### Aşama 3 — Derle ve başlat (isteğe bağlı)

Yığında bir derleme adımı varsa (`compile`, `npm install`, `docker compose up`) onu çalıştır:

- Derle; hata olursa yüksek sesle başarısız ol.
- Yerel servisleri başlat.
- Sağlık denetimlerinin geçmesini bekle (tipik olarak 30-60 saniye).

Yalnızca şablon iskeleler için bu aşamayı atla.

### Aşama 4 — Doğrula (ZORUNLU)

`/verify-system` becerisini bir `Skill` aracı çağrısıyla çalıştır. **Bu pazarlığa kapalıdır.**

```
Skill(skill="verify-system")
```

Aynı takım, yığını uçtan uca sınamayı bilen kendi `/verify-system` becerisini yayımlar. İskele şunları yapmalı:

1. Beceriyi `Skill` aracıyla çağırmalı (satır içi bash ile değil).
2. Sonucu beklemeli.
3. Doğrulama geçmezse görünür biçimde başarısız olmalı.

### Aşama 5 — Commit at

Doğrulama geçtikten sonra:

```bash
git init -b main
git add .
git commit -m "chore: initial scaffold via create-new-project"
```

Uzak depoyu boş bırak — uzak depoya push'lamak kullanıcının kararıdır.

## Çıktı sözleşmesi

Başarılı bir çalıştırmanın sonunda kullanıcı şunları içeren bir **son rapor** görür:

- Proje yolu.
- Ne oluşturuldu (sayılar: dosya, servis, ajan, beceri, kural).
- Doğrulama sonucu (✅ tamamı geçti / ❌ ne başarısız oldu).
- Sonraki adımlar (projenin nasıl açılacağı, nasıl çalıştırılacağı, belgelerin nerede olduğu).

## Başarısızlık kipleri

- **Aşama 1 iptal** — kullanıcı sorgu sırasında vazgeçer. Temiz çıkış, dosya yazılmaz.
- **Aşama 2 hatası** — kısmi iskele. Beceri ya geri almalı ya da kısmi ağacı açık bir notla bırakmalı.
- **Aşama 3 derleme hatası** — iskele yerinde kalır; derleme hatası raporlanır; Aşama 4'e **geçilmez**.
- **Aşama 4 doğrulama hatası** — iskele yerinde kalır; kullanıcı neyin başarısız olduğuna ve nasıl düzeltileceğine dair açık bilgi alır.

## Yeni başlama deneyimi

Hiç takım kurmadan `/create-new-project` yazan ilk kez kullanan biri şunu görür:

```
Skill not found: create-new-project
```

Bu kasıtlıdır. `/create-new-project` her zaman yığına özgüdür; bu yüzden genel bir global sürümü yoktur. Kullanıcı önce bir takım kurması gerektiğini öğrenir — `npm create react-app`'in önce `npm`'i gerektirmesi gibi.

İleride yapılacak iş: `atl new-project <team> <name>` — önceden var olan bir proje dizini gerektirmeden doğrudan takımın iskelesine yönlendirir.

## İlgili

- **[Bir takım yazma](./creating-a-team)** — iskele becerilerinin nerede yaşadığı.
- **[team.json](./team-json)** — beceriyi nasıl kaydedeceğin.
- **[Kavramlar](/tr/guide/concepts)** — bir becerinin ne olduğu.
