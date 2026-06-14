# Kalıtım

Bir takım başka bir takımı genişletebilir. Mevcut bir takımın işini çatallamadan (fork) yeniden kullanmanın yolu budur.

## Temeller

```json
{
  "name": "my-team",
  "version": "0.1.0",
  "extends": "software-project-team@^1.0.0"
}
```

Hepsi bu kadar. Biri `atl install my-team` çalıştırdığında:

1. `atl` üst takımı (`software-project-team@^1.0.0`) kayıt defterinden çözer.
2. Üst takım kurulur (özyinelemeli olarak — kendi `extends` zinciri de çözülür).
3. Alt takımın kendi ajanları, becerileri ve kuralları üstüne kurulur.

Kullanıcı, üst takımdan gelen **ve** alt takımdan gelen tüm ajanlarla son bulur.

## Ad bazında bastırma

Alt takım, üst takımın ajanıyla aynı adda bir ajan yayımlarsa alt takım kazanır. Dosya birleştirme yoktur — tümüyle yerine yazma.

```json
{
  "extends": "software-project-team@^1.0.0",
  "agents": [
    { "name": "api-agent", "description": "Custom API conventions for our stack." }
  ]
}
```

Kurulum sırasında alt takımdaki `agents/api-agent.md`, üst takımın kopyasını işaret eden dosyanın üzerine yazar.

Aynı kural beceriler ve kurallar için de geçerlidir: ad çakışması ⇒ alt takım kazanır.

## İstemediklerini dışarıda bırak

```json
{
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"]
}
```

Üst takımdaki `ux-agent` asla kopyalanmaz. Üst takım dosyası önbellekte durmaya devam eder (üst takım olağan biçimde kurulur) — alt takım yalnızca onu yüzeye çıkarmamayı seçer.

Ajanları, becerileri ya da kuralları adlarına göre dışarıda bırakabilirsin.

## Sınırsız derinlik

Zincirler istediğin kadar derin olabilir:

```
grandchild-team
  └─ extends  child-team@^1.0.0
      └─ extends  software-project-team@^1.0.0
```

Kurulumda yükleme sırası **en derin atadan en yakın takıma**dır — bu nedenle sana en yakın takım her ad çakışmasını kazanır. Sınıf kalıtımıyla aynı zihinsel model.

## Döngü algılama

`A extends B extends A`, kurulum sırasında yakalanır ve hata mesajında zincirin tamamı yer alarak başarısız olur. Sessiz sonsuz döngü olmaz.

## Yalnızca tek üst takım

Bir takımın en çok bir `extends` alanı olabilir. Çoklu kalıtım yoktur, elmas sorunu yoktur. İki takımdan parça istiyorsan her ikisini kur — ya da birini `extends` et, ötekini `dependencies` altında bildir.

## Sürüm kısıtları

`extends` değeri `name` ya da `name@version-constraint` biçimindedir. Desteklenen kısıtlar:

| Sözdizim | Eşleşir |
|---|---|
| `software-project-team` | En son. |
| `software-project-team@^1.0.0` | `>=1.0.0 <2.0.0` — **önerilen**. |
| `software-project-team@~1.2.0` | `>=1.2.0 <1.3.0`. |
| `software-project-team@1.2.3` | Kesin olarak `1.2.3`. |

Caret önerilen varsayılandır — yama düzeltmelerini ve yeni küçük sürümleri alırsın ama geriye uyumsuz `2.0.0` sürümü sessizce sızmaz.

## Öncelik sırası

`atl install` kopyaları çözerken sonraki girişler öncekileri bastırır:

1. Atalar (en derin önce, en yakın sonra).
2. Mevcut takım.
3. Uygulanan dışarıda bırakmalar (listede olan her şey düşürülür).

## Uygulamalı örnek

Üst takım `software-project-team@1.0.0`:

```
agents: api-agent, flutter-agent, react-agent, ux-agent, ...
skills: create-new-project, verify-system
rules:  commit-style
```

Alt takım `my-team@0.1.0`:

```json
{
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    { "name": "api-agent", "description": "Override — my API conventions." },
    { "name": "analytics-agent", "description": "New — Mixpanel + GA instrumentation." }
  ]
}
```

`atl install my-team` sonrası:

```
.claude/agents/
├── api-agent.md         → my-team/agents/api-agent.md         (bastırma)
├── flutter-agent.md     → software-project-team/agents/flutter-agent.md
├── react-agent.md       → software-project-team/agents/react-agent.md
├── analytics-agent.md   → my-team/agents/analytics-agent.md   (yeni)
└── (ux-agent yok)                                             (dışarıda bırakıldı)
```

Beceriler ve kurallar aynı süreci izler.

## Genişletme mi yoksa çatallama mı?

- Üst takımın kararlarının %70'inden fazlasına katılıyorsan ve üst akış güncellemelerini istiyorsan **genişlet**.
- Temel kararlarla aynı fikirde değilsen (farklı dil, farklı mimari) **çatalla** — o noktada genişletmek, sağladığından çok yük getirir.

## Sıradaki

- **[team.json başvurusu](./team-json)** — her alan.
- **[Bir takım yazma](./creating-a-team)** — adım adım.
