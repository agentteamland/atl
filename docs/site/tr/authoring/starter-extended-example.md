# Uygulamalı örnek: `starter-extended`

[`agentteamland/starter-extended`](https://github.com/agentteamland/starter-extended), [kalıtım](inheritance) düzeneğini uçtan uca gösteren küçük bir örnek takımdır. Kendi uzantı takımını yazmak üzereysen ve geçerli en küçük örneğin nasıl göründüğünü görmek istiyorsan bu sayfayı oku.

Kalıtım düzeneğinin kendisi (extends, excludes, bastırma anlamı, yükleme sırası) için önce [Kalıtım](inheritance) sayfasına bak.

## `starter-extended` neyi gösterir?

Yaklaşık 50 satırlık bir depo içinde:

- **`extends`** — `team.json` içinde bir üst takımı bildirme (`extends: software-project-team@^1.0.0`).
- **`excludes`** — miras alınmış bir üyeyi adına göre dışarıda bırakma (`excludes: ["ux-agent"]`).
- Üst takımın üzerine yeni bir ajan **ekleme** (`stripe-agent`).
- Kanonik ajan düzeni ([`agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md) gereği) — Identity / Area of Responsibility / Core Principles / Knowledge Base; konu başına bir dosya olarak `children/` altında, `knowledge-base-summary` frontmatter alanıyla.

Toplam: 1 `team.json` + 1 ajan (`stripe-agent`, 1 çocuk dosyasıyla).

## Depo düzeni

```
starter-extended/
├── README.md                                # taslak — bu belge sayfasına işaret eder
├── LICENSE                                  # MIT
├── team.json                                # bildirir: extends + excludes + yeni ajan
└── agents/
    └── stripe-agent/
        ├── agent.md                         # Identity, Responsibility, Core Principles, Knowledge Base
        └── children/
            └── webhook-topology.md          # knowledge-base-summary frontmatter'ı ile
```

## `team.json`

```json
{
  "name": "starter-extended",
  "version": "0.2.0",
  "description": "Minimal example team — demonstrates extending software-project-team, excluding ux-agent, and adding stripe-agent.",
  "author": "agentteamland",
  "license": "MIT",
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    {
      "name": "stripe-agent",
      "description": "Stripe billing integration: webhook topology, idempotency, refund flows.",
      "path": "agents/stripe-agent/"
    }
  ]
}
```

Biri `atl install starter-extended` çalıştırdığında:

1. `atl` üst takımı (`software-project-team@^1.0.0`) kayıt defterinden çözer → 13 ajan + 3 beceri kurar.
2. `excludes: ["ux-agent"]` kuralı `ux-agent`'ı kurulu kümeden çıkarır → 12 ajan + 3 beceri kalır.
3. Alt takımın kendi ajanı (`stripe-agent`) üstüne kurulur → toplam 13 ajan + 3 beceri.

Kullanıcı, üst takımın tam yığını eksi `ux-agent` artı özel bir Stripe odaklı ajanla son bulur. Bu en yaygın "software-project-team'in çoğunu istiyorum ama kendi dokunuşumla" desenidir.

## Ajan (`stripe-agent`)

`agents/stripe-agent/agent.md` dosyası kanonik yapıyı izler ([`agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md) gereği):

```markdown
# Stripe Agent

## Identity
Stripe billing integration specialist. Handles webhook events,
idempotency, refunds, dispute flows, and Stripe-specific error semantics.

## Area of Responsibility
- Stripe webhook endpoint design + signature verification
- Webhook idempotency (event ID dedup, replay protection)
- Refund + partial-refund flows
- Dispute handling
- Stripe-specific error mapping to internal exceptions

## Core Principles
- Never trust webhook payloads without signature verification
- Always idempotent — Stripe retries aggressively
- Refund logic lives in API; Stripe SDK calls go through a dedicated service

## Knowledge Base

(Auto-rebuilt by /save-learnings from children/*.md frontmatter.)

### Webhook Topology
Stripe's webhook event taxonomy + the exchange/queue topology we
use to fan out events to consumers. Idempotency is enforced at
event-ID level via Redis SETNX with 7-day TTL.
→ [Details](children/webhook-topology.md)
```

`children/webhook-topology.md` dosyası kanonik `knowledge-base-summary` frontmatter alanını taşır:

```markdown
---
knowledge-base-summary: "Stripe's webhook event taxonomy + the exchange/queue topology we use to fan out events to consumers. Idempotency is enforced at event-ID level via Redis SETNX with 7-day TTL."
---

# Webhook Topology

(Detaylı içerik: tüm şemalar, adlandırma sözleşmeleri, yeniden deneme davranışı, ...)
```

Bu, [Children + learnings](../guide/children-and-learnings) deseninin uygulamada görünüşüdür — üst `agent.md` dosyasının Knowledge Base bölümü her çocuk dosyanın frontmatter alanından kendiliğinden yeniden inşa edilir.

## Şablon olarak kullan

Depo bir GitHub Şablonu olarak ayarlanmıştır. Kendi uzantı takımını başlat:

```bash
gh repo create your-org/your-extension-team --template agentteamland/starter-extended
```

Ardından `team.json` dosyasını düzenle:

- Farklı bir üst takım seç (`extends: design-system-team@^0.8.0` gibi).
- İstemediğin üyeleri kaldırmak için `excludes` listesini ayarla.
- Kendi ajanlarını `agents/` altına, becerileri `skills/` altına, kuralları `rules/` altına ekle.

Push'tan önce yerelde doğrula:

```bash
~/.claude/repos/agentteamland/core/scripts/validate-team-json.sh team.json
```

Ya da doğrulayıcıyı `git push` üzerine bağla (önerilir):

```bash
git config core.hooksPath .githooks    # klon başına bir kez (.githooks'u da kopyaladıysan)
```

Doğrulama geçtikten sonra push'la, bir etiket aç ve kısa adla kurulum istiyorsan [kayıt defterine](registry-submission) başvur.

## Genişletmek mi yoksa sıfırdan yazmak mı?

Şu durumlarda genişlet:

- Üst takımın mimari seçimleriyle (i18n, Mediator, önce Docker vb.) hemfikirsen ve sadece üstüne uzman ajanlar eklemek istiyorsan.
- Güncellemelerin üst takımdan akmasını istiyorsan — `software-project-team` v1.3.0 sürümünü yayımladığında uzantın `^1.0.0` kısıtı gereği üst güncellemeleri kendiliğinden alır.
- Küçük ve odaklı bir kod tabanı istiyorsan (takımının deposu yalnızca farkları içerir).

Şu durumlarda sıfırdan yaz:

- Yığının temelden farklıysa (farklı dil, farklı desenler, farklı altyapı).
- Miras alınmış bir yüzey olmadan tam denetim istiyorsan.
- Farklı bir platformu hedefliyorsan (örneğin Rust servisleri için AgentTeamLand tarzı bir takım).

"Küçük uzmanlık eklemesi" durumu için genişletmek çok daha az iştir. `starter-extended` deposu toplamda ~50 satırdır — bu, ".NET yığını ama stripe-agent eklenmiş" durumunun tüm maliyetidir.

## İlgili

- [Kalıtım](inheritance) — bu örneğin gösterdiği düzenek.
- [Bir takım yazma](creating-a-team) — sıfırdan yazma yolu.
- [Children + learnings](../guide/children-and-learnings) — burada kullanılan `agent.md` / `children/` düzeni.
- [Kayıt defteri başvurusu](registry-submission) — takımının kısa adını katalog'a koyma.

## Tarihçe

- `v0.1.0` (2026-04-17) — kalıtımı gösteren ilk örnek.
- `v0.2.0` (2026-05-02) — platform genelindeki inceleme sırasında kurtarıldı (Phase 2.B/2.C migrasyonu uygulandı: `agent.md` bölümleri kanonik şemaya yeniden adlandırıldı + Knowledge Base bölümü eklendi; çocuk dosyalarına `knowledge-base-summary` frontmatter eklendi; LICENSE + README eklendi).
