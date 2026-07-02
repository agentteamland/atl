---
layout: home

hero:
  name: AgentTeamLand
  text: Paket gibi kurulan AI agent takımları.
  tagline: AI agent takımlarının paket yöneticisi — tüm bir stack'i tek komutla kur, güncel tut, ürünü çıkar.
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
      link: https://github.com/agentteamland/atl

features:
  - icon: 📦
    title: Takımlar = paketler
    details: Bir takım; belirli bir tür iş için uzmanlaşmış agent'ları, skill'leri ve rule'ları bir arada paketler — full-stack uygulamalar, design system'ler ve dahası. Tek komutla kur, projenin Claude Code dizinine kopyalansın.
  - icon: ⚡
    title: Tek static binary
    details: atl, runtime bağımlılığı olmayan ~7 MB'lık bir Go binary'si. Tek bir curl (macOS/Linux) ya da PowerShell (Windows) komutuyla kur.
  - icon: 🔄
    title: Kendi kendini süren güncelleme + öğrenme
    details: Hook'lar takımlarını güncel tutar ve session içi öğrenimleri kendiliğinden bilgi tabanına işler — kazanımları global katmanına promote et, üst kaynağa publish et.
  - icon: 🧪
    title: Katalogdan keşfet
    details: Takımlar, atl-team topic'iyle etiketlenmiş herkese açık GitHub depolarından üretilen katalogda yaşar. atl search ile keşfet, tek install komutuyla kur — yayımlamak da aynı ölçüde açık.
  - icon: 🔍
    title: Açık self-publish
    details: Deponu atl-team GitHub topic'iyle etiketle ve atl publish çalıştır — merkezi bir kapı bekçisi yok. Herhangi bir takımı adıyla atl search ile keşfet.
  - icon: 🛠️
    title: Açık ve programlanabilir
    details: Her şey MIT lisanslı. team.json açık bir schema. Kendi takımını yaz ve publish et.
---

<div style="text-align:center; margin: 3rem 0 1rem;">

## Çalışırken gör

<img src="https://raw.githubusercontent.com/agentteamland/workspace/main/assets/demo.gif" alt="atl demo" width="820" style="max-width:100%; border-radius:8px;"/>

</div>

## 30 saniyede

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

```bash
# Sonra, herhangi bir projede:
atl install <handle>/<takım>      # takımı atl search ile keşfet
atl setup-hooks                   # opsiyonel: her Claude Code session'ında auto-update + öğrenme yakalama
```

Kurduğun takımın agent'ları, skill'leri ve rule'ları projenin `.claude/` dizinine kopyalanır — Claude Code hepsini anında görür, hemen kullanıma hazırdır.

## Sıradaki

- **[`atl` nedir?](/tr/guide/what-is-atl)** — beş dakikada büyük resim.
- **[Hızlı başlangıç](/tr/guide/quickstart)** — ilk takım 60 saniyeden kısa sürede kurulu.
- **[Takımlara göz at](/tr/teams/)** — takım kataloğu ve keşif.
- **[Takım yazımı](/tr/authoring/team-json)** — kendi takımını yayınla.
