# `/drain`

Bekleyen öğrenme kuyruğunu bilgi tabanına katar — her öğeyi wiki'ye, journal'a ya da bir ajanın bilgi tabanına yönlendirir, ardından onaylayarak (ack) kuyruktan siler.

`/drain`, v2 öğrenme döngüsünün **tüketen yarısıdır**. Yakalama kendiliğinden ve belirlenimcidir: Claude bir konuşma sırasında sessiz `<!-- learning -->` işaretçileri düşürür, [`atl tick`](/tr/cli/tick) ise her birini dayanıklı bir bbolt kuyruğuna tam olarak bir kez aktarır. Bu beceri, CLI'nin tek başına yapamayacağı LLM yarısıdır — kuyruğa alınmış her öğrenmeyi okur, nereye ait olduğuna karar verir, onu birleştirir ve onaylar.

## Ne zaman kullanılır?

- `atl`, oturum başında **"N learning(s) pending"** raporladığında.
- Öğrenme kuyruğunu elle işlemek istediğin her an.

CLI yarısı [`atl learnings`](#cli-yarisi) tarafından sunulur: `status` bekleyen sayıları gösterir, `peek` bu becerinin tükettiği öğeleri listeler, `ack` ise işlenmiş bir öğeyi siler. Kuyruk mevcut proje dizinine göre anahtarlanır; bu yüzden beceriyi, öğrenmelerini boşaltmak istediğin projeden çalıştır.

## Neden ack = silme?

Kuyruk, tam-olarak-bir-kez teslim ve yineleme ayıklama güvencesi verir. Transkriptleri hiç yeniden taramaz ve durum izlemezsin — yalnızca `peek` (gözat), birleştir ve `ack` (onayla) yaparsın. Onaylanmış bir öğe kuyruktan **silinir**, böylece bir daha asla yeniden raporlanamaz. v1'in yeniden-raporlama hata sınıfı yapısal olarak ortadan kalkmıştır: ilerletilecek bir durum dosyası ve karşı yineleme ayıklayacak hiçbir şey yoktur; bu yüzden boş bir kuyrukta `/drain` komutunu yeniden çalıştırmak işlem yapmaz.

## Yordam

### 1. Kuyruğa gözat

Proje dizininde çalıştır:

```bash
atl learnings peek --channel learning --json
```

Her öğe `{id, channel, payload, enqueued_at}` biçimindedir. `payload` serbest metindir — yakalanan işaretçinin gövdesidir. Liste boşsa "boşaltılacak bir şey yok" diye raporla ve dur.

### 2. Her öğeyi payload'ının biçimine göre yönlendir

v2 işaretçisi `topic`/`kind` üst verisi taşımaz — hedefi payload'dan **sen** çıkarırsın ve içerikten kebab-case bir `topic` türetirsin (tek kavram: `auth-refresh`, `redis-ttl`).

| Payload biçimi | Hedef |
|---|---|
| Konu biçimli güncel doğru ("kimlik doğrulamanın doğru yolu …") | **Wiki** — `<proj>/.atl/wiki/<topic>.md` (varsa yerine yaz / birleştir) **+ journal** |
| Tarih damgalı anlatı ("X denedik, sonra Y, sonunda Y işe yaradı") | **Yalnızca journal** — `<proj>/.atl/journal/<YYYY-MM-DD>.md` (ekle) |
| Belirli bir kurulu ajan için alan bilgisi | **Ajan KB** — `<scope>/.claude/agents/<agent>/children/<topic>.md` + o ajanın `## Knowledge Base` bölümünü yeniden inşa et **+ journal** |
| Tekrar eden bir iş akışı, billurlaşmış bir sözleşme, sahibi olan ajanı bulunmayan yeni bir alan ya da bir ajanın/becerinin kimlik genişlemesi | **Yapısal** — özerk biçimde YAZMA; topla ve öner |

Sahibi olan ajanı bulmak için `<proj>/.claude/agents/` ve `~/.claude/agents/` altındaki kurulu ajanlara bak (proje, global'i gölgeler). Hiçbir ajan onu açıkça sahiplenmiyorsa, bunun yerine wiki'ye yönlendir. Yazdığın şeye daima **NEDEN**'i kat — gerekçesiz bir gerçek çürür.

### 3. Yaz, sonra onayla — birer birer

Yapısal olmayan yazımlar sessizdir (onay yok). Her öğe birleştirildikten sonra, kuyruktan çıkması için onayla:

```bash
atl learnings ack <id>
```

**Yalnızca** yazım başarıyla tamamlandıktan sonra onayla. Bir öğeyi birleştiremiyorsan, onu bırak (onaylama) ve raporda not düş.

### 4. Yapısal değişiklikler — öner, asla kendiliğinden uygulama

"Yapısal" satırı için ajanları/becerileri/kuralları sessizce yazma. Onları topla ve en sonunda her birini `AskUserQuestion` üzerinden öner (yeni ajan / yeni beceri / yeni kural / kimlik değişikliği). Bu, reaktif-yaratım sınırıdır: yapısal büyümeyi bir insan onaylar. Yapısal bir öğeyi yalnızca önerisi çözüme bağlandıktan sonra onayla.

### 5. Raporla

Neyin nereye indiğini özetle: öğe başına, konu → hedef; oluşturulan yeni dosyaları ve varsa yapısal önerileri listele. Kısa tut.

## CLI yarısı

`/drain`, [`atl learnings`](/tr/cli/learnings) altındaki üç belirlenimci eylemi sürer:

```bash
atl learnings status          # bu proje için kanal başına bekleyen sayılar
atl learnings peek            # bekleyen öğeleri listele (insan tarafından okunabilir)
atl learnings peek --json     # becerinin tükettiği tam, makinece okunabilir liste
atl learnings peek --channel learning   # tek bir kanala süz
atl learnings ack <id>        # işlenmiş bir öğeyi kuyruktan sil
```

Bayraklar:

- `peek --json` — bekleyen öğeleri JSON olarak yayımlar (id, channel, payload, enqueued_at).
- `peek --channel <name>` — tek bir kanala süzer (ör. `learning`).

`status` ve `ack` bayrak almaz. `ack` tam olarak bir argüman alır — öğenin `id`'si.

### Kanallar

Kuyruk çok kanallıdır. `/drain` yalnızca **`learning`** kanalını işler. `profile-fact` kanalı, ileride gelecek bir birinci-taraf profil takımı için ayrılmıştır ve burada işlenmez.

## Ajan KB yeniden inşası

Bir ajanın bilgi tabanı `agent.md` + bir `children/` dizinidir. Her çocuk tek bir konudur ve `knowledge-base-summary` frontmatter alanı taşır:

```markdown
---
knowledge-base-summary: "<agent.md'nin Knowledge Base bölümünde kullanılan tek satırlık özet>"
---

# <Konu Başlığı>

<asıl bilgi — desenler, örnekler, gerekçe>
```

Bir çocuğu yazdıktan ya da güncelledikten sonra, `agent.md`'nin `## Knowledge Base` bölümünü çocukların frontmatter alanlarından (dosya adına göre sıralı) **tümüyle yeniden inşa et**. O bölüm türetilmiştir, elle düzenlenmez — her çalıştırmada baştan sona değiştirilir. Aynı desen, bir öğrenme bir beceriyi hedef alıyorsa, becerinin `learnings/` dizini ve `## Accumulated Learnings` bölümü için de geçerlidir.

## Wiki dizini yeniden inşası

Bir `/drain` çalıştırması bir `.atl/wiki/` sayfasını yazdığında ya da güncellediğinde, projenin `CLAUDE.md`'sindeki `<!-- wiki:index -->` bloğunu yeniden inşa eder ki bilgi haritası eşzamanlı kalsın — sayfa başına bir `- [topic](.atl/wiki/topic.md) — özet` satırı, dosya adına göre sıralı, türetilmiş (elle düzenlenmez). Projede `CLAUDE.md` yoksa yeniden inşa atlanır (`atl init` / `atl install` dosyayı oluşturur). Bloğun biçimi ve yerleşimi için [Claude Code kuralları](/tr/guide/claude-code-conventions) sayfasına bak.

## Örnekler

### Oturum başı bir komut isteminin ardından boşaltma

Yeni bir oturum açılır ve `atl` iki bekleyen öğrenme raporlar. Onları işle:

```bash
atl learnings peek --channel learning --json
```

```json
[
  {
    "id": "9f1c2a3b4d5e",
    "channel": "learning",
    "payload": "Redis cache TTL should be 30 minutes, not 15 — 15 caused cold-start thrash under load.",
    "enqueued_at": "2026-06-21T09:14:02Z"
  }
]
```

`redis-ttl` konusunu wiki'ye (güncel doğru) yaz ve bugünün journal'ına tarihli bir madde ekle, sonra onayla:

```bash
atl learnings ack 9f1c2a3b4d5e
```

### Boşaltmadan kuyruğu denetle

```bash
atl learnings status
```

```
learning queue — pending by channel:
  learning       2
```

## Kapsam

Wiki ve journal, `<proj>/.atl/` altındaki proje bilgisidir. Ajan KB'si ajanın kurulum kapsamını izler — bir proje `.claude/` dizini global `~/.claude/` dizinini gölgeler.

## Kaynak

- Belirtim: [core/skills/drain/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/drain/SKILL.md)
- CLI: [cli/cmd/atl/commands/learnings.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/learnings.go)
