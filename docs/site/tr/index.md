---
layout: home

hero:
  name: AgentTeamLand
  text: Paket gibi kurulan AI agent takımları.
  tagline: AI agent takımlarının paket yöneticisi ve registry'si — tüm bir stack'i tek komutla kur, başkasının takımını miras al, ürünü çıkar.
  image:
    src: /logo.svg
    alt: AgentTeamLand
  actions:
    - theme: brand
      text: Başla
      link: /tr/guide/quickstart
    - theme: alt
      text: atl'yi kur
      link: /tr/guide/install
    - theme: alt
      text: GitHub
      link: https://github.com/agentteamland

features:
  - icon: 📦
    title: Takımlar = paketler
    details: Bir takım; belirli bir tür iş için uzmanlaşmış agent'ları, skill'leri ve rule'ları bir arada paketler. Tek komutla kur, projenin .claude/ dizinine kopya ile yerleşsin.
  - icon: 🧬
    title: Doğru kurgulanmış miras
    details: Tek parent'lı extends; override ve excludes desteği. Sınırsız derinlik, circular detection, caret version constraint'leri — npm seviyesinde semantik, diamond problem yok.
  - icon: ⚡
    title: Tek static binary
    details: atl ~7 MB'lık bir Go binary'si. Homebrew, Scoop, winget veya curl ile kur — runtime bağımlılığı sıfır.
  - icon: 🧪
    title: İki onaylı takım, daha fazlası yolda
    details: software-project-team (13 agent — .NET, Flutter, React, Docker) ve design-system-team (DS + prototype araçları, /dst-* skill'leri). Registry'ye göz at.
  - icon: 🔍
    title: Herkese açık registry
    details: Takımları kısa adıyla keşfet. Registry; CI'da schema ile doğrulanan, PR ile yönetilen tek bir JSON dosyası.
  - icon: 🛠️
    title: Açık ve programlanabilir
    details: Her şey MIT lisanslı. team.json açık bir schema. Kendi takımını yaz ve registry'ye gönder.
---

<div style="text-align:center; margin: 3rem 0 1rem;">

## Çalışırken gör

<img src="https://raw.githubusercontent.com/agentteamland/workspace/main/assets/demo.gif" alt="atl demo" width="820" style="max-width:100%; border-radius:8px;"/>

</div>

## 30 saniyede

```bash
# macOS / Linux
brew install agentteamland/tap/atl
```

```powershell
# Windows (PowerShell, paket yöneticisi şart değil)
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

```bash
# Sonra, herhangi bir projede:
atl install software-project-team
atl install design-system-team    # opsiyonel: design-system + prototype araçları
atl setup-hooks                   # opsiyonel: her Claude Code session'ında auto-update
```

13 agent'lık tam bir stack (API, web, mobil, veritabanı, altyapı, kod incelemesi) projenin `.claude/` dizinine bağlanmış, Claude Code'un hemen kullanımına hazır. Yanına design-system-team eklersen `/dst-*` design araçlarını da kazanırsın.

## Sıradaki

- **[`atl` nedir?](/tr/guide/what-is-atl)** — beş dakikada büyük resim.
- **[Hızlı başlangıç](/tr/guide/quickstart)** — ilk takım 60 saniyeden kısa sürede kurulu.
- **[Takımlara göz at](/tr/teams/)** — registry'deki onaylı takımlar.
- **[Takım yazımı](/tr/authoring/team-json)** — kendi takımını yayınla.
