# Kavramlar

AgentTeamLand'i oluşturan parçalar ve bunların birbirine nasıl oturduğu.

## Takım

Bir **takım** bir pakettir. Claude Code ile belirli bir iş türünü yapmak için gereken her şeyi bir araya getirir:

- **Ajanlar** — kendi bağlamı ve sorumlulukları olan uzmanlaşmış kişilikler.
- **Beceriler** — Claude Code içinde sunulan, kullanıcının çağırabileceği komutlar (slash komutları).
- **Kurallar** — her zaman yüklü olan davranışsal kısıtlar ve sözleşmeler.

Bir takım, kök dizininde `team.json` bulunan bir Git deposunda yaşar. O dosya takımı tanımlar: adını, sürümünü ve neleri paketlediğini.

Bir takımı kur, içeriği `.claude/` dizininde kopyalar olarak belirir. Claude Code bunları anında görür.

## Ajan

Bir ajan, bir rolü tanımlayan bir Markdown dosyasıdır. `backend-agent`, `code-reviewer` — her biri kendi sorumluluk alanı ve kendi bilgi tabanı olan, odaklanmış bir kişiliktir.

Karmaşık ajanlar için benimsenen yaklaşım **children deseni**dir: üst düzey `agent.md` dosyası kısadır (kimlik, kapsam, ilkeler) ve ayrıntılı bilgi `children/` altında, dosya başına bir konu olarak yaşar. Bu, üst düzey dosyayı sıkı tutar ve tek bir konuyu, gerisine dokunmadan ucuza güncellemeyi sağlar. Her çocuk dosyası, `/drain`'in `agent.md`'nin kendiliğinden yeniden inşa edilen **Knowledge Base** bölümüne taşıdığı bir `knowledge-base-summary` frontmatter satırı taşır — böylece üst dosyadaki dizin daima çocuklardan türetilir, asla elle düzenlenmez.

Tüm yapı için bkz. [Children + learnings](/tr/guide/children-and-learnings).

## Beceri {#skill}

Bir beceri, kullanıcının çağırabileceği bir slash komutudur. `/drain`, `/create-pr`, `/docs-audit`. Beceriler, kök dizininde bir `SKILL.md` bulunan dizinler olarak gelir; o dosya, becerinin ne zaman kullanılacağını ve ne yapması gerektiğini anlatır.

Beceriler **bilgi deposu değil, yordamdır** — bir beceri, çalıştırılacak adımlardır, dolayısıyla birikmiş-bilgi dizini taşımaz. Bilgi tabanı ajanın `children/` dizininde birleştirilmiştir (v1, bu şekli becerilere `learnings/` dizini olarak yansıtıyordu; v2 bunu kaldırdı, [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md) uyarınca).

Beceriler **global** (`atl`'nin kendisiyle birlikte gelen) ya da **takım kapsamlı** (belirli bir takımca getirilen ve yalnızca o takım kurulduktan sonra görünen) olabilir. Yaptığı iş yığına özgü olan beceriler — örneğin bir takımın proje iskeleti kuran becerisi — takım kapsamlıdır. `/drain`, `/create-pr`, `/create-code-diagram`, `/brainstorm`, `/rule` ve `/rule-wizard` ise globaldir çünkü her yere uygulanır.

Beceri ile CLI arasındaki ayrım bilinçlidir: **CLI deterministiktir** (aynı girdiler, aynı sonuç, LLM yok); **beceriler LLM güdümlüdür** (senin özel kodun üzerinde akıl yürütürler). `/drain`, öğrenme döngüsünün muhakeme yarısıdır; `atl learnings` ise deterministik yarısı. Aşağıdaki [CLI](#the-cli) bölümüne bak.

## Kural

Bir kural, her Claude Code oturumuna yüklenen bir Markdown dosyasıdır. Bir beceriden farklı olarak (beceri çağrılmayı bekler), bir kural daima etkindir — daha sen bir şey sormadan, Claude'un projeyi nasıl düşüneceğini biçimlendirir.

Global kurallar `~/.claude/rules/` dizininde yaşar. Takımın sağladığı kurallar, takım kurulduğunda projenin `.claude/rules/` dizinine kopyalanır.

## Kapsam: global ve proje

`atl`'nin kurduğu her şey iki **kapsam**tan birinde var olur:

- **Global** (`~/.claude/`) — makinendeki her projede paylaşılan ajanlar, beceriler ve kurallar.
- **Proje** (`<project>/.claude/`) — aynı tür varlıklar, ama tek bir projeye kapsanmış.

Her iki kapsam da aynı adı taşıyan bir varlık barındırdığında **proje kopyası kazanır** — en yakın olan, globali gölgeler. Diğer her kavramın üzerine asıldığı eksen budur: bir kapsamda kurarsın, kazanımlar kapsamlar _arasında_ dolaşır ve Claude Code birleştirilmiş sonucu okur.

ATL'ye ait ayrı bir varlık deposu yoktur. Varlıklar Claude Code'un kendi dizinlerinde yaşar; `atl`'nin defter tutması (öğrenme kuyruğu, önbelleğe alınmış katalog, pin'ler, kurulum manifestoları) `~/.atl/` ve `<project>/.atl/` altında yaşar.

## Takım kataloğu

Takımlar bir **katalog** üzerinden keşfedilir — [`atl-team`](https://github.com/topics/atl-team) konusuyla etiketlenmiş herkese açık GitHub depolarından oluşturulan, üretilmiş bir dizin. `atl install <handle>/<takım>` çalıştırmak, referansı o dizine karşı çözer ve eşleşen depodan kurar.

Kayıt defteri deposu da yok, gönderim PR'ı da. Bir takımın listelenmesi için deposunu `atl-team` ile etiketlersin (ya da takım deposundan `atl publish` çalıştırırsın) ve dizin onu alır. `atl search` dizini sorgular; `atl install` ona karşı çözer. Bir **`[verified]`** rozeti, AgentTeamLand bakımcıları tarafından incelenmiş takımları işaretler — yokluğu yalnızca takımın kendi kendine yayımlandığı anlamına gelir, güvensiz olduğu değil.

Dizinin nasıl sorgulanıp tazelendiği için bkz. [`atl search`](/tr/cli/search).

## Kazanım dolaşımı

Ajanların çalıştıkça **kazanımlar** biriktirir — yeni öğrenimler, keskinleşmiş beceriler, proje-yerel kurallar. AgentTeamLand bu kazanımları üç-halkalı bir merdivenle dışa doğru taşır, böylece kimse çözülmüş bir sorunu yeniden çözmez:

1. **Proje → global** — `atl promote`, proje-yerel bir kazanımı global katmana yükseltir, böylece her proje yararlanır. `atl pin`, bir özelleştirme yalnızca projeye özel kalmalıyken bir yolu geride tutar; `atl unpin` onu serbest bırakır.
2. **Global → upstream** — `atl publish`, global katmandaki kazanımlarını takımın kaynak deposuna geri paylaşır: sahibi olduğun bir takımı yeniden yayımla ya da sahibi olmadığın bir takım için kazanımları bir GitHub PR'ı olarak öner. Yazar sınırını aştığı için asla otomatik çalışmaz — sen çağırırsın; sahibi kabul eder.
3. **Upstream → herkes** — `atl update`, kurulu her takımın en son yayımlanan sürümünü çeker ve paylaşılan kazanımları her kuruluma geri dağıtır.

`promote` ve `update`'teki dağıtım rutin işlerdir; `publish` ise tasarım gereği bilinçlidir.

## Öğrenme kuyruğu

Öğrenme yakalama otomatiktir. Bir oturuma bir öğrenme işaretçisi düştüğünde, satır içinde işlenmek yerine **kalıcı bir kuyruğa** (`~/.atl/queue.db`, bir bbolt deposu) eklenir — uzun bir oturumun aynı maddeyi neden asla yeniden raporlamadığının nedeni budur. `/drain` becerisi kuyruğu ajan bilgi tabanlarına işler; `atl learnings` ise ona açılan deterministik penceredir (bekleyenler için `status`, maddeleri listelemek için `peek`, birini işlenmiş olarak işaretlemek için `ack`).

Bu, v1'in `/save-learnings`'ini (artık `/drain`) değiştirir ve ayrı `memory` kavramını tamamen kaldırır: bağımsız bir bellek katmanı yoktur. Ajan başına öğrenimler ajanın bilgi tabanında yaşar ([children + learnings](/tr/guide/children-and-learnings)); kesişen bilgi ise projenin [bilgi sisteminde](/tr/guide/knowledge-system) yaşar (journal + wiki).

## CLI {#the-cli}

`atl`, deterministik, kullanıcıya yönelik araçtır. Komutları üç gruba ayrılır:

**Takım komutları** (elle çalıştırılır):

- `atl install [team]` — bir takımı (katalog adıyla) geçerli kapsama kurar.
- `atl list` — burada neyin kurulu olduğunu gösterir.
- `atl remove [team]` — kurulumu kaldırır.
- `atl update [team]` — bir ya da tüm kurulu takımlar için en sonu çeker.
- `atl search [query]` — takım kataloğunda arar.

**Kazanım-dolaşımı komutları** (ajanlarının öğrendiklerini dışa taşır):

- `atl promote` — proje-yerel kazanımları global katmana yükseltir.
- `atl publish` — kendi takımını yeniden yayımlar ya da kazanımları upstream'e önerir.
- `atl pin` / `atl unpin` — bir yolu yükseltmeden geride tutar ya da serbest bırakır.
- `atl learnings` — kalıcı öğrenme kuyruğunu inceler (`status` / `peek` / `ack`).

**Otomasyon komutları** (Claude Code hook'larına bağlıdır; bunları nadiren yazarsın):

- `atl setup-hooks` — `SessionStart` + `UserPromptSubmit` hook'larının tek seferlik kurulumu.
- `atl session-start` — açılış zamanı bakımı (çekirdek tazeleme + işaretçi taraması + doctor öz-onarımı + günde bir binary self-update kontrolü).
- `atl tick` — oturum içi bakım tıkırtısı (kısıtlanmış arka plan işini birkaç dakikada bir boşaltır).
- `atl doctor` — kendini iyileştiren artalan süreci: sapmayı teşhis eder ve kurulumu otomatik onarır.

v1 ile karşılaştırıldığında `config`, `migrate` ya da `learning-capture` komutu yoktur. Bkz. [CLI'ye genel bakış](/tr/cli/overview).

## Claude Code ile nasıl çalışır

Claude Code her oturumun başında `.claude/` dizinini okur. Bir takımın o dizine kattığı her şey anında belirir — devir için hazır ajanlar, slash komutu olarak hazır beceriler, her komut istemine yüklenen kurallar. Proje kapsamı global kapsamı gölgeler, dolayısıyla birleştirilmiş görünüm daima "en yakın kazanır" şeklindedir.

AgentTeamLand, Claude Code'un yerine geçmez ya da onu genişletmez. O bir teslim katmanıdır: Claude Code'un zaten okuduğu dosyalar için paket yönetimi — artı bir öğrenme döngüsü.
