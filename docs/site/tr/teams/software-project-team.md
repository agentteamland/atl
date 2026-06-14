# software-project-team

> Üretim seviyesinde tam yığın yazılım projeleri için 13 uzmanlaşmış ajan.

**En son sürüm:** `1.2.1`
**Durum:** Doğrulanmış
**Depo:** [github.com/agentteamland/software-project-team](https://github.com/agentteamland/software-project-team)

## Kur

```bash
cd your-project
atl install software-project-team
```

## Yığın

- **API**: .NET 9, Minimal API, [martinothamar/Mediator](https://github.com/martinothamar/Mediator) (kaynak üreteç), FluentValidation.
- **Veritabanı**: PostgreSQL 17, EF Core 9.
- **Mesajlaşma**: RabbitMQ 3 (fanout exchange'ler).
- **Önbellek**: Redis 7.
- **Günlükleme**: Serilog → RMQ → Elasticsearch 8 + Kibana.
- **E-posta**: MailSender tüketicisi → Mailpit (geliştirme).
- **Kimlik doğrulama**: JWT HS256, BCrypt, X-Internal-Token.
- **Depolama**: MinIO (geliştirme) / S3 (üretim).
- **Mobil**: Flutter (Riverpod, go_router).
- **Web**: React + TypeScript + Vite (ya da Next.js).
- **Altyapı**: Docker Compose, dotnet watch.

## Ajanlar (13)

| Ajan | Rol |
|---|---|
| `api-agent` | .NET Minimal API uzmanı (Vertical Slice + Clean Arch + Mediator). |
| `socket-agent` | SignalR köprüsü (gerçek zamanlı WebSocket + JWT). |
| `worker-agent` | Cronos güdümlü arka plan işleri. |
| `flutter-agent` | Mobil / tablet uzmanı (Riverpod + go_router + dio + i18n). |
| `react-agent` | Web UI uzmanı (Tailwind + Zustand + React Query + i18next). |
| `infra-agent` | Docker Compose topolojisi, sağlık denetimleri, ortam değişkeni bağlama. |
| `database-agent` | PostgreSQL + EF Core 9 uzmanı. |
| `redis-agent` | Yenileme jetonları, önbellek, idempotenlik, devingen ayarlar. |
| `rmq-agent` | Fanout exchange'ler, DLX topolojisi, tüketici desenleri. |
| `code-reviewer` | Birleştirme öncesi kod incelemesi. |
| `project-reviewer` | Çapraz kesen proje incelemesi. |
| `design-system-agent` | Tasarım jetonları, bileşen kütüphanesi, tema. |
| `ux-agent` | Kullanıcı deneyimi, akışlar, erişilebilirlik, mikro metin. |

## Beceriler

- `/create-new-project [Name]` — 5 aşamalı iskele (bilgi toplama → yapı oluşturma → derleme → `/verify-system` → `git init`).
- `/verify-system` — 4 düzeyli uçtan uca sağlık denetimi (konteynerler, portlar, uygulama sağlığı, boru hatları).
- `/design-screen` — Claude Design döngüsü orkestratörü (akış belirtimi ile kod arasındaki görsel-prototip aşaması).

## Yerleşmiş mimari desenler

Takım, ürettiği tüm kodda şu sistemli yetenekleri zorunlu kılar:

- **Yalnızca UI'da i18n** (1.0.3+) — Backend yalnızca İngilizcedir. UI uygulamaları `messageKey + placeholders + fallback` zarfı üzerinden yerelleştirir. E-postalar ve push bildirimleri MailSender içindeki yerel başına şablonlarla yerelleştirilir.
- **Claude Design tümleştirmesi** (1.1.0+) — `/design-screen` üzerinden isteğe bağlı görsel prototip aşaması. Flutter, React, design-system ve UX için ajan başına aktarım kılavuzları.
- **DST aktarım bilgisi** (1.1.3+) — `flutter-agent` ve `react-agent`, design-system-team paketinin biçimini ve prototipleri kaynak kodla bütünleştirmeden önceki zorunlu tema eşzamanlama adımını bilir.
- **Soğuk derleme iskele disiplini** (1.1.4+) — `/create-new-project` Aşama 2.2, soğuk derlemelerin elle yapılan csproj düzeltmesi olmadan ilk denemede `/verify-system`'i yeşil geçmesini sağlamak için dört zorunlu csproj / migration gereksinimini (Infrastructure `<FrameworkReference Include="Microsoft.AspNetCore.App"/>`, Api `Microsoft.EntityFrameworkCore.Design`, Worker / LogIngest / MailSender `Microsoft.Extensions.Hosting`, başlangıç EF migration'ı) zorunlu kılar. Tam keşif bağlamı: `agents/api-agent/children/known-issues.md`.
- **Kendini güncelleyen öğrenme döngüsüyle hizalama** (1.2.0+) — Her ajanın `agent.md` Knowledge Base bölümü `/save-learnings` tarafından her `children/{topic}.md` dosyasının `knowledge-base-summary` frontmatter alanından kendiliğinden yeniden inşa edilir. Platform genelindeki Phase 2.C migration betiğiyle 169 çocuk dosyası yeni sözleşmeye taşındı (kayıpsız; özetler mevcut elle hazırlanmış KB bölümlerinden olduğu gibi alındı). Her beceri, `/save-learnings`'in doldurmasına hazır boş bir `learnings/` alt dizini ve `## Accumulated Learnings` bölümüyle birlikte yayımlanır.

Tüm ayrıntılar için [takımın README dosyasına](https://github.com/agentteamland/software-project-team) ve `agent.md` dosyalarına bak.

## Nerede kullanılır?

- Bir referans test projesi — her sürümden önce uçtan uca (`/create-new-project` + `/verify-system` tamamı yeşil) doğrulanır.
