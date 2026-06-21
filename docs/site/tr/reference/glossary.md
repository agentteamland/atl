# Sözlük

**Ajan (agent)** — Claude Code için uzmanlaşmış bir rolü tanımlayan Markdown dosyası. Bir takımın parçası olarak yayımlanır. Takım deposunun `agents/` dizininde yaşar; kurulum kapsamındaki `.claude/agents/` dizinine kopyalanır.

**atl** — CLI (`atl install`, `atl list`, …). Katalogdan takımları çözümler ve kurar. Go ile yazılmış bir ikilidir.

**Katalog / Dizin** — takımların keşfedilme biçimi. GitHub'da [`atl-team`](https://github.com/topics/atl-team) konusuyla etiketlenmiş herkese açık depolardan oluşturulan bir dizin. `atl search` bu dizini sorgular; `atl install` bir tanıtıcıyı buna göre çözümler. Önbelleğe alınmış kopya `~/.atl/index.json` konumunda yaşar. Bkz. [`atl search`](/tr/cli/search).

**Children deseni** — karmaşık ajanlar için bir sözleşme: üst düzey `agent.md` kısa kalır (kimlik, kapsam, ilkeler, Knowledge Base); ayrıntılı bilgi `children/` altında konu başına bir dosya olarak yaşar. Her çocuk dosya, [`/drain`](/tr/skills/drain)'in üst `agent.md` dosyasının Knowledge Base bölümünü kendiliğinden yeniden inşa etmek için kullandığı `knowledge-base-summary` frontmatter alanını taşır. Becerilerde aynı desen `learnings/` olarak yansıtılır (`skill.md`'nin Accumulated Learnings bölümünü kendiliğinden yeniden inşa eder).

**Bağımlılıklar** — bir takımın gereksinim duyduğu ek takımlar; `team.json` içindeki `dependencies` alanıyla belirtilir (takım adı → sürüm kısıtı eşlemesi). Takımın kendisiyle birlikte çözülür ve kurulur.

**Tanıtıcı (Handle)** — katalogda kullanılan kurulum kimliği; `<tanıtıcı>/<takım>` biçimindedir (örn. `agentteamland/software-project-team`). Tanıtıcı yayıncı ad alanıdır; `atl install <tanıtıcı>/<takım>` onu dizinde arar ve eşleşen depodan kurar.

**Manifesto (Manifest)** — her takım için kapsamına göre ayrı tutulan kurulum kaydı; `<katman>/.atl/installed/<tanıtıcı>__<ad>.json` konumundadır (`<katman>` global için `~/.atl`, proje için `<proje>/.atl`). `schemaVersion`, tanıtıcı, ad, sürüm, kapsam, kaynak (`repo`, `subpath`, `ref`), `installedAt` ve kurulu yol → SHA-256 eşlemesini içeren `files` haritasını kaydeder. `atl remove`, `atl update` ve `atl doctor` tarafından kullanılır.

**Proje** — `atl`'yi çalıştırdığın bir dizin. Proje kapsamlı kurulumlar takımın varlıklarıyla `.claude/` dizinini doldurur; ATL'nin proje kapsamlı kayıtları `<proje>/.atl/` altında tutulur.

**Kural (rule)** — Claude Code tarafından her zaman yüklenen Markdown dosyası (çağrılmayı bekleyen becerilerin aksine). Takımın `rules/` dizinindedir; kurulum kapsamındaki `.claude/rules/` altına kopyalanır.

**İskele (scaffolder)** — takımın yığınında yeni bir projeyi başlatan, `/create-new-project` adıyla anılan takım kapsamlı beceri. [İskele belirtimine](/tr/authoring/scaffolder-spec) uymalıdır.

**Kapsam (Scope)** — bir varlığın kurulduğu katman. İki kapsam vardır: **global** (`~/.claude` içindeki varlıklar, `~/.atl` içindeki ATL durumu) ve **proje** (`<proje>/.claude` içindeki varlıklar, `<proje>/.atl` içindeki ATL durumu). Bir takımın yayıncısı `team.json` içinde varsayılan kapsamı (`project`, `global` ya da `both`; varsayılan `project`) bildirir; kullanıcı `--global` / `--project` ile geçersiz kılar. Bir yetenek her iki katmanda da mevcutsa proje katmanı globali **gölgeler** — en yakın olan kazanır.

**SemVer kısıtı** — `dependencies` ve `requires.atl` alanlarında kullanılan sürüm aralığı sözdizimi. `^1.0.0` (caret), `~1.2.0` (tilde), `1.2.3` (kesin), `>=1.2.0` (açık uçlu).

**Beceri (skill)** — kullanıcı tarafından çağrılan eğik çizgili komut (örneğin `/verify-system`). Kök dizininde `skill.md` bulunan bir dizin olarak gelir. Global beceriler `~/.claude/skills/` altında yaşar; takım kapsamlı beceriler bir takımla birlikte gelir ve kurulumun ardından `.claude/skills/` altında görünür.

**Takım (team)** — kökünde `team.json` bulunan bir Git deposu; belirli bir iş türü için ajanları, becerileri ve kuralları bir araya paketler.

**team.json** — her takım deposunun kökündeki manifesto dosyası. Takımın adını, sürümünü, açıklamasını, yazarını, lisansını, paketlediklerini (ajanlar/beceriler/kurallar), `dependencies`'i, minimum `requires.atl` sürümünü ve isteğe bağlı varsayılan `scope`'u bildirir. Bkz. [team.json sözleşmesi](/tr/authoring/team-json).

**Çalışma alanı (workspace)** — `agentteamland/workspace`, tüm eş depoların geliştirme için bir araya getirildiği bakımcı merkezi. AgentTeamLand'i kullanmak için gerekmez; yalnızca platforma katkı veriyorsan ilgilenir.

**Journal** — `.atl/journal/{date}_{agent}.md` altındaki kronolojik, ajan başına öğrenme kaydı. [`/drain`](/tr/skills/drain) öğrenme kuyruğunu bilgi tabanına işlerken yazar; ajan açılışında Claude tarafından [knowledge-system kuralı](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md) gereği okunur.

**knowledge-base-summary** — her `children/{topic}.md` (ve `learnings/{topic}.md`) dosyasında zorunlu olan YAML frontmatter alanı. [`/drain`](/tr/skills/drain)'in üst `agent.md`'nin Knowledge Base (ya da `skill.md`'nin Accumulated Learnings) bölümünü yeniden inşa ederken çıkardığı bir-üç satırlık özet. Kaynak doğruluktur — yeniden inşa edilmiş bölüme yapılan elle düzenlemeler bir sonraki `/drain` çalıştırmasında üzerine yazılır.

**knowledge-system** — iki katmanlı bilgi modelini (`journal/` + `wiki/`) tanımlayan çekirdek kural. `agent-memory` katmanı journal'a katıldıktan sonra `memory-system` adından yeniden adlandırıldı.

**learnings/** — ajanların `children/` dizinini yansıtan, beceri başına alt dizin. Her `learnings/{topic}.md` dosyası `knowledge-base-summary` frontmatter taşır; becerinin `## Accumulated Learnings` bölümü bu dosyalardan kendiliğinden yeniden inşa edilir.

**Öğrenme işaretçisi** — bir öğrenme anı geçtiğinde Claude'un konuşma sırasında düşürdüğü satır içi HTML yorumu. Biçim: `<!-- learning: serbest metin -->`. `atl tick` tarafından dayanıklı kuyruğa (`~/.atl/queue.db`) alınır (içerik hash'iyle tekilleştirilerek, tam bir kez), ardından [`/drain`](/tr/skills/drain) tarafından işlenir ve silinir.

**Wiki** — `.atl/wiki/{topic}.md` altında konuya göre düzenlenmiş güncel doğru bilgisi. Doğru değiştiğinde eklenmez, yerine yazılır; `CLAUDE.md` dosyasındaki `<!-- wiki:index -->` işaretçi bloğu canlı dizini her oturum başında Claude'a görünür kılar.
