# `atl skills`

Platformun kendi skill'leri, agent'ları ve takım manifestleri için belirlenimci, LLM'siz **içerik-kalitesi kontrolleri** — [`atl docs check`](/tr/cli/docs)'in kardeşi. docs-check docs *sitesini* koda karşı doğrularken, skills-check **varlıkların kendisini** doğrular.

Bu, monorepo'nun `core/` ve `teams/` ağaçlarına karşı çalışan bir **maintainer-tarafı** kapıdır. Monorepo dışında hiçbir şey yapmaz ve 0 ile çıkar (ön-uçuş atlaması), böylece son-kullanıcı oturumları onu hiç görmez.

## Kullanım

```bash
atl skills check                      # frontmatter, team.json tutarlılığı, bildirilen yakalama kanalları, agent-KB çocukları, skill shell gövdelerini doğrula
atl skills check --record-stocktake   # HEAD'i son-stocktake-yapılan commit olarak damgala (/skill-stocktake bir taramanın sonunda çalıştırır)
```

## Neleri kontrol eder

Dört yapısal kontrol **yapısı gereği sıfır-yanlış-pozitiftir** — bulgu, dosya hakkında bir olgudur (bir frontmatter anahtarı ya vardır ya yoktur). Beşincisi, `shell`, yapısal bir olgu değil, adlandırılmış yapılar üzerinde bilinçli olarak dar tutulmuş bir desen eşleştirmesidir; depodaki her skill üzerinde ölçülerek temiz bulunmuştur ve **neyi kapsamadığı** aşağıda yazılıdır. Beşi de bir PR'ı bağlamak için güvenlidir:

| Kontrol | Ne sağlanmalı |
|---|---|
| **frontmatter** | Her skill'in `SKILL.md`'si ve her agent'ın `agent.md`'si bir `name` + `description` frontmatter bloğu taşır. |
| **manifest** | Her `team.json`'ın `agents[]` / `skills[]` adları diskteki dizinlerle eşleşir — **her iki yönde** (bildirilmiş-ama-yok yok, diskte-ama-bildirilmemiş yok). |
| **channel** | [Bildirilen her yakalama kanalı](/tr/authoring/team-json#declaring-a-capture-channel) dört alanının hepsini taşır ve `rule` ile `drain` alanları takımın gerçekten yayımladığı bir kurala ve bir skill'e işaret eder. |
| **children** | **Yayımlanan** her agent-KB çocuğu (`teams/<team>/agents/<x>/children/*.md`) boş olmayan bir `knowledge-base-summary` frontmatter'ı bildirir — KB-yeniden-inşa sözleşmesi. |
| **shell** | Bir `SKILL.md` içindeki hiçbir shell bloğu, bilinen iki yalnızca-bash yapısından birini kullanmaz — eşleşmeyen bir glob, skill gövdesini gerçekten çalıştıran kabuk olan zsh'te **ölümcüldür**. Tam kapsam ve sınırları aşağıdadır. |

### Bildirilen bir kanal neden diske karşı çözümleniyor

Bir yakalama kanalının sinyalleri, bildirimin verdiği dört kelimeden kurulur ve çalışma zamanında hiçbir şey bu kelimelerin gerçek bir yere işaret ettiğini denetlemez. Dolayısıyla takımın yayımlamadığı bir kurala işaret eden kanal her yerde kabul edilir: kanal etkinleşir, işaretleri kuyruğa *gerçekten* alınır — ve hiçbir kural bir agent'a onları boşaltmasını söylemez. Sonsuza dek birikirler ve tek belirti, açıklaması olmayan bir birikimdir.

Çalışma zamanındaki doctor kontrolü bu yarıyı yakalayamaz. O, takımın kaynak ağacının çoktan gitmiş olduğu *kurulu* bir manifesti inceler; yalnızca dört alanın var olduğunu doğrulayabilir. Burada, monorepo'da varlıklar tam oradadır — bu da bozuk bir birinci-parti bildiriminin çözümlenebileceği tek yer olmasını sağlar ve yayımlanmak yerine CI'ı kırar.

### Bir skill'in shell gövdesi neden denetleniyor

Bir skill gövdesi **çalıştırılabilirdir**: agent bloğu, Bash aracının çalıştırdığı kabuğa yapıştırır; yani ` ```bash ` çiti **bir etikettir, çalıştırıcı değil** (macOS'te bu zsh'tir). zsh'in varsayılan `nomatch` ayarı, eşleşmeyen bir glob'u **ölümcül** yapar — bash onu düz metin bırakırken. Böyle bir kesilme bir kez `rm -rf`'ten *sonra* gerçekleşti: yedeği sildi ve skill'in sonuç işaretlerinden hiçbirini yazdırmadı. Kusur bir Markdown dosyasının içindeki bash parçacığında yaşadığı için her kapıdan yeşil geçti.

Üç yapı işaretlenir:

1. **`for` listesinde çıplak glob** (`for entry in "$SRC"/*`) — bunun yerine bir `find` sonucunu dolaşın ya da `cp -R "$SRC/."` ile kopyalayın.
2. **Betiğin veya bir fonksiyonun son ifadesi olarak `[ … ] && cmd`**, `set -e` altında — yanlış bir test tüm birimin sıfırdan farklı çıkmasına yol açar. `if … then … fi` kullanın ya da `|| true` ekleyin. (Yalnızca *son* konum işaretlenir: başka yerlerde POSIX, bir AND-OR listesinin sonuncusu dışındaki her komutu errexit'ten muaf tutar; orada deyim güvenlidir ve raporlanmaz.)
3. **Yıkıcı sıralama** — yukarıdakilerden birinin, `trap` olmadan bir `rm -r` sonrasında durması; taşınabilirlik pürüzünü veri kaybına çeviren şey budur.

Bu bir **desen eşleştirmesidir, ayrıştırma değil**. `sh -n`, `bash -n` ve `zsh -n` yukarıdaki yapıların hepsini kabul eder — `nomatch` çalışma-zamanı davranışıdır — dolayısıyla bir sözdizimi denetimi hiçbirini yakalamaz. Değer tümüyle bilinen taşınamaz yapıları tanımaktadır.

**Neyi kapsamaz.** Bu yapıları bilir; bir gövdenin genel olarak kabuktan bağımsız olup olmadığına karar vermez. Kaçırdığı en yakın kardeş, bir `for` listesi *dışındaki* çıplak glob'dur — `cp -R "$SRC"/* "$DEST/"` zsh altında tıpatıp aynı şekilde kesilir. Bu bir gözden kaçırma değil, bilinçli bir sınırdır: glob kuralını her komut sözcüğüne genişletmek korpus üzerinde ölçüldü ve **her** isabet yanlış pozitif çıktı (`?` ile biten satır sonu yorumu, bir `case` deseni, shell çiti içine alıntılanmış JavaScript); yalancı çoban olan bir kapı da kapatılan kapıdır. Yine de kabuktan bağımsız biçimi yazın — bu kontrol kuralın kendisi değil, kuralın altındaki ağdır.

### Kurulu katman burada değil, oturum başında denetlenir

Yukarıdaki `children` denetimi bir PR içinde yazılan kopyaları gezer. Ama `/drain`, agent-KB çocuklarını **kurulu** katmana yazar — `<proje>/.claude/agents/<agent>/children/` ve `~/.claude/agents/<agent>/children/` — ki CI bu katmanı göremez; oraya kapı koymak, koşucunun kopyası bile olmayan bir şeye kapı koymak olurdu.

Bu yarı, onun yerine bir **oturum-başı uyarısı**dır ve yalnız monorepo'da değil, her projede çalışır: oradaki bir çocukta `knowledge-base-summary` yoksa `atl session-start` bunu söyler. `## Knowledge Base` bölümü tam da o frontmatter'dan türetildiği için, onsuz yazılan bir çocuk yeniden inşaya kendi girdisini türetecek hiçbir şey bırakmaz. Uyarı ihlali bildirir — `/drain`'e frontmatter'ı yazdıramaz; o sözleşme skill'in kendi metninde yaşar (aynı metin, böyle bir çocuğu geri-doldurmak yerine düşürmeyi de yasaklar).

`atl skills check` herhangi bir başarısızlıkta sıfırdan farklı çıkar; bu yüzden docs-drift kapısının yanında **her PR'ı CI'da kapılar**. Yargı yarısı — bir skill kendi belgelenmiş akışına uyuyor mu? iki skill örtüşüyor mu? — bu belirlenimci ağın değil, eşlik eden [`/skill-stocktake`](/tr/skills/skill-stocktake) skill'inin (LLM) işidir. Bu ayrım CLI/Skill sınırıdır: belirlenimci kontroller burada, zeminli yargı skill'de.

`--record-stocktake`, çalıştırma hatasız tamamlandığında HEAD'i son-stocktake-yapılan commit olarak (`~/.atl` durumunda) damgalar — `/skill-stocktake` skill'i bunu bir taramanın sonunda, oturum-başındaki "stocktake zamanı geldi" sinyalini sıfırlamak için çağırır; `atl rules scan --record`'un kardeşidir.

## İlgili

- [`/skill-stocktake`](/tr/skills/skill-stocktake) — LLM yarısı: itaat + fazlalık, grep-zeminli, değişim-farkında
- [`atl docs check`](/tr/cli/docs) — kardeş kapı: docs-sitesi driftı (bu, varlık içerik-kalitesi)
- [`atl doctor`](/tr/cli/doctor) — çalışma-zamanı kendini-iyileştirme (bu, derleme-zamanı kalite kapısı)
