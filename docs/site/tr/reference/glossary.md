# Sözlük

**Ajan (agent)** — Claude Code için uzmanlaşmış bir rolü tanımlayan Markdown dosyası. Bir takımın parçası olarak yayımlanır. Takım deposunun `agents/` dizininde yaşar; projedeki `.claude/agents/` dizinine kopyalanır.

**atl** — CLI (`atl install`, `atl list`, …). Takımları bir projeye kurar. Go ile yazılmış bir ikilidir.

**Önbellek** — `atl`'nin klonlanmış takım depolarını sakladığı paylaşılan disk dizini. Makine başına bir adettir: `~/.claude/repos/agentteamland/`. Projeler buraya kopya çıkarır.

**Children deseni** — karmaşık ajanlar için bir sözleşme: üst düzey `agent.md` kısa kalır (kimlik, kapsam, ilkeler, Knowledge Base); ayrıntılı bilgi `children/` altında konu başına bir dosya olarak yaşar. Her çocuk dosya, `/save-learnings`'in üst `agent.md` dosyasının Knowledge Base bölümünü kendiliğinden yeniden inşa etmek için kullandığı `knowledge-base-summary` frontmatter alanını taşır. Becerilerde aynı desen `learnings/` olarak yansıtılır (`skill.md`'nin Accumulated Learnings bölümünü kendiliğinden yeniden inşa eder).

**Döngülü zincir** — bir takımın `extends` zincirinin kendine geri dönmesi (`A → B → A`). Kurulum sırasında yakalanır; CLI tam zinciri yazdırarak başarısız olur.

**Bağımlılıklar** — bir takımın gereksinim duyduğu ek takımlar; `team.json` içindeki `dependencies` alanıyla belirtilir. Takımın kendisiyle birlikte çözülür ve kurulur.

**`excludes`** — `team.json` içinde, alt takımın kopyalanmasını istemediği üst takım sağlamalı ajan / beceri / kural adlarının listesi. Üst takım kurulduktan sonra etkili olur.

**`extends`** — tek üst takım kalıtımı. `extends: "parent@^1.0.0"` taşıyan bir takım önce üst takımı kurar ve kendi içeriğini üzerine katmanlar.

**Bastırma** — alt takım, üst takımdaki bir öğeyle aynı adda bir öğe yayımlarsa alt takımın sürümü kazanır. Tümüyle yerine yazma; birleştirme yok.

**Proje** — `atl`'yi çalıştırdığın bir dizin. Kurulu takımlara ait kopyalarla doldurulmuş bir `.claude/` alt dizini kazanır.

**Kayıt defteri** — takımların herkese açık kataloğu. [`agentteamland/registry`](https://github.com/agentteamland/registry) deposunda tek bir `teams.json` dosyası. PR güdümlüdür ve şema ile doğrulanır.

**Kural (rule)** — Claude Code tarafından her zaman yüklenen Markdown dosyası (çağrılmayı bekleyen becerilerin aksine). Takımın `rules/` dizinindedir; `.claude/rules/` altına kopyalanır.

**İskele (scaffolder)** — takımın yığınında yeni bir projeyi başlatan, `/create-new-project` adıyla anılan takım kapsamlı beceri. [İskele belirtimine](/tr/authoring/scaffolder-spec) uymalıdır.

**SemVer kısıtı** — `extends` ve `dependencies` alanlarında kullanılan sürüm aralığı sözdizimi. `^1.0.0` (caret), `~1.2.0` (tilde), `1.2.3` (kesin), `>=1.2.0` (açık uçlu).

**Beceri (skill)** — kullanıcı tarafından çağrılan eğik çizgili komut (örneğin `/verify-system`). Kök dizininde `skill.md` bulunan bir dizin olarak gelir. Global beceriler `~/.claude/skills/` altında yaşar; takım kapsamlı beceriler bir takımla birlikte gelir ve kurulumun ardından `.claude/skills/` altında görünür.

**Durum** — bir takımın kayıt defterindeki durumu: `verified`, `community` ya da `deprecated`.

**Takım (team)** — kökünde `team.json` bulunan bir Git deposu; belirli bir iş türü için ajanları, becerileri ve kuralları bir araya paketler.

**team.json** — her takım deposunun kökündeki manifesto dosyası. Adı, sürümü, açıklamayı, paketlenenleri, genişletilen üst takımı ve dışarıda bırakılanları bildirir.

**Çalışma alanı (workspace)** — `agentteamland/workspace`, tüm eş depoların geliştirme için bir araya getirildiği bakımcı merkezi. AgentTeamLand'i kullanmak için gerekmez; yalnızca platforma katkı veriyorsan ilgilenir.

**Journal** — `.atl/journal/{date}_{agent}.md` altındaki kronolojik, ajan başına öğrenme kaydı. Emekli edilmiş `agent-memory/` katmanının yerini alır. `/save-learnings` tarafından yazılır; ajan açılışında Claude tarafından [knowledge-system kuralı](https://github.com/agentteamland/core/blob/main/rules/knowledge-system.md) gereği okunur.

**knowledge-base-summary** — her `children/{topic}.md` (ve `learnings/{topic}.md`) dosyasında zorunlu olan YAML frontmatter alanı. `/save-learnings`'in üst `agent.md`'nin Knowledge Base (ya da `skill.md`'nin Accumulated Learnings) bölümünü yeniden inşa ederken çıkardığı bir-üç satırlık özet. Kaynak doğruluktur — yeniden inşa edilmiş bölüme yapılan elle düzenlemeler bir sonraki save-learnings çalıştırmasında üzerine yazılır.

**knowledge-system** — iki katmanlı bilgi modelini (`journal/` + `wiki/`) tanımlayan çekirdek kural. `agent-memory` katmanı journal'a katıldıktan sonra `core@1.8.0` sürümünde `memory-system` adından yeniden adlandırıldı.

**learnings/** — ajanların `children/` dizinini yansıtan, beceri başına alt dizin. Her `learnings/{topic}.md` dosyası `knowledge-base-summary` frontmatter taşır; becerinin `## Accumulated Learnings` bölümü bu dosyalardan kendiliğinden yeniden inşa edilir.

**Öğrenme işaretçisi** — bir öğrenme anı geçtiğinde Claude'un konuşma sırasında düşürdüğü satır içi HTML yorumu. Biçim: `<!-- learning topic: ... kind: ... doc-impact: ... body: ... -->`. Bir sonraki oturumun `SessionStart` adımında `atl learning-capture` tarafından taranır ve `/save-learnings --from-markers` tarafından işlenir.

**Wiki** — `.atl/wiki/{topic}.md` altında konuya göre düzenlenmiş güncel doğru bilgisi. Doğru değiştiğinde eklenmez, yerine yazılır; `CLAUDE.md` dosyasındaki `<!-- wiki:index -->` işaretçi bloğu canlı dizini her oturum başında Claude'a görünür kılar.
