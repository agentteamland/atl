# `atl list`

Mevcut projede kurulu takımları gösterir.

## Kullanım

```bash
atl list
```

## Çıktı

```
Installed teams in /home/you/projects/your-app:

  ✓ software-project-team@1.2.1
     effective: 13 agents, 3 skills, 1 rules

  ↑ starter-extended@0.2.0 [community]  (latest: 0.3.0)
     extends: software-project-team@^1.0.0
     effective: 12 agents, 2 skills, 1 rules
```

Takım başına biçim:

- **`✓`** — bilinen en son sürümde kurulu (yeşil onay işareti).
- **`↑`** — kurulu, ama kayıt defterinde daha yeni bir sürüm var (sarı yukarı oku). En yeni sürüm parantez içinde gösterilir; yenilemek için `atl update`.
- **`name@version`** — kurulu takımın adı ve sürümü (`team.json` dosyasından).
- **`[community]`** — takım topluluk durumundaysa (henüz doğrulanmamışsa) rozet olarak görünür. Doğrulanmış takımlarda rozet yoktur.
- **`extends:`** — yalnızca takımın bir üst takımı varsa yazdırılır. Kalıtım zincirini gösterir (kök → ... → kurulu takım).
- **`effective:`** — kalıtım çözümünden sonraki sayılar: üst ve alt takım kaynakları, eksi `excludes`, alt takım bastırma çakışmaları sıkıştırılmış hâliyle.

## Projede takım yoksa

```
No teams installed in this project.
Run: atl install <team-name>
```

## Notlar

- `atl list` okumayı `.claude/.team-installs.json` dosyasından ve global kayıt defteri önbelleğinden yapar. Kararlı durumda ağa gitmez — eski sürüm tespiti (`↑`) önbelleklenmiş kayıt defteri kopyasını kullanır; o kopya da `atl update` tarafından yenilenir.
- Komutun şu anda hiçbir bayrağı yoktur. Önceki belgeler `--json` ve `--chain` bayraklarından söz ediyordu; bu bayraklar hiç hayata geçirilmedi. Betiklenebilir çıktıya gereksinimin varsa `.claude/.team-installs.json` dosyasını doğrudan ayrıştır — kararlı bir JSON'dur ve `name`, `version`, `extendsChain` ve `effective` alanlarını içerir.

## İlgili

- [`atl install`](/tr/cli/install).
- [`atl remove`](/tr/cli/remove).
- [`atl update`](/tr/cli/update).
