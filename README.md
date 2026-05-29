# User Proxy Portal

Outil de configuration du proxy d'entreprise pour Ubuntu intégré à un domaine Active Directory.

Un **binaire unique** qui installe un proxy local authentifiant (Basic Auth) entre vos applications et le proxy d'entreprise, sans exposer vos credentials en clair dans les variables d'environnement.

## Comment ça fonctionne

```
Application / CLI
      ↓  http_proxy=http://127.0.0.1:3128
userproxyportal proxy  (daemon systemd)
      ↓  Proxy-Authorization: Basic <creds depuis GNOME Keyring>
proxy.entreprise.com:8080
      ↓
Internet
```

Le daemon proxy local lit les credentials depuis le **GNOME Keyring** (chiffré) et les injecte à la volée dans chaque requête. Les variables d'environnement système pointent sur `127.0.0.1:3128` sans aucun credential.

---

## Prérequis

- Ubuntu 22.04+ avec environnement GNOME
- Machine intégrée à un domaine Active Directory (via `sssd`)
- Proxy d'entreprise avec authentification **Basic**
- Certificat SSL d'inspection (`.cer` / `.crt`) fourni par votre administrateur

---

## Installation

### 1. Télécharger le binaire

Téléchargez la dernière version depuis les [releases GitHub](../../releases/latest) selon votre architecture :

| Architecture | Fichier |
|---|---|
| Intel / AMD (x86_64) | `userproxyportal-linux-amd64` |
| ARM 64-bit | `userproxyportal-linux-arm64` |

```bash
# Vérifier l'intégrité (optionnel)
sha256sum -c userproxyportal-linux-amd64.sha256

# Rendre exécutable
chmod +x userproxyportal-linux-amd64
```

### 2. Lancer l'installateur

```bash
./userproxyportal-linux-amd64 install
```

Cette commande (via `pkexec`) :
- Copie le binaire dans `/usr/local/bin/userproxyportal`
- Installe le service systemd dans `/etc/systemd/user/`
- Crée le répertoire de configuration `/etc/userproxyportal/`

### 3. Configurer le proxy

```bash
userproxyportal setup
```

Ouvre l'interface d'administration. Dans l'onglet **Configuration** :

| Champ | Description |
|---|---|
| Hôte proxy | Adresse du proxy d'entreprise (ex: `proxy.entreprise.local`) |
| Port proxy | Port du proxy (ex: `8080`) |
| Port local | Port d'écoute local (défaut: `3128`) |
| Exclusions | Hôtes à ne pas proxifier (ex: `localhost,127.0.0.1,.interne.local`) |
| URL PAC | Optionnel — URL du fichier PAC d'entreprise |
| Certificat SSL | Chemin vers le `.cer` d'inspection SSL |

Cochez **Installer le certificat dans le trust store** pour l'ajouter aux CA système.

### 4. Activer le service

```bash
systemctl --user enable --now userproxyportal.service
```

Le daemon proxy démarre automatiquement à chaque ouverture de session.

### 5. Renseigner vos credentials

```bash
userproxyportal
```

Saisissez votre nom d'utilisateur et mot de passe AD. Vos credentials sont stockés dans le **GNOME Keyring** (jamais en clair sur le disque).

Cliquez **Appliquer** — le proxy est actif immédiatement.

---

## Utilisation quotidienne

Le proxy est transparent une fois configuré. Le daemon redémarre automatiquement à chaque session.

### Mettre à jour ses credentials (changement de mot de passe)

```bash
userproxyportal
```

Saisissez le nouveau mot de passe et cliquez **Appliquer**. Le daemon recharge les credentials à la volée via `SIGHUP`.

### Désactiver temporairement le proxy

```bash
userproxyportal
# → Cliquer "Désactiver"
```

### Vérifier l'état du service

```bash
systemctl --user status userproxyportal.service
```

---

## Référence des sous-commandes

```
userproxyportal              Interface utilisateur (credentials AD)
userproxyportal setup        Interface d'administration
userproxyportal proxy        Démarrer le daemon proxy (usage systemd)
userproxyportal apply        Appliquer la configuration sans GUI
userproxyportal install      Installer le programme dans /usr/local/bin
userproxyportal version      Afficher la version
userproxyportal help         Afficher cette aide
```

### `userproxyportal apply`

```
Flags:
  --config string     Chemin vers config.yaml (défaut: /etc/userproxyportal/config.yaml)
  --privileged        Met aussi à jour /etc/environment et installe le certificat
  --clear             Supprime la configuration proxy
```

---

## Configuration

Fichier : `/etc/userproxyportal/config.yaml`

```yaml
proxy:
  host: proxy.entreprise.local   # Adresse du proxy d'entreprise
  port: 8080                     # Port du proxy d'entreprise
  local_port: 3128               # Port d'écoute local (défaut: 3128)
  pac_url: ""                    # URL PAC (optionnel)
  no_proxy: localhost,127.0.0.1,::1,.entreprise.local

certificate:
  path: /etc/userproxyportal/entreprise-ca.cer   # Certificat d'inspection SSL
```

Ce fichier est géré par l'administrateur et déployé sur toutes les machines du parc.

---

## Build depuis les sources

### Prérequis

- Go 1.22+
- Dépendances système :

```bash
sudo apt-get install -y gcc libgl1-mesa-dev libx11-dev \
  libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libxxf86vm-dev
```

### Compiler

```bash
git clone https://github.com/wisper/userproxyportal
cd userproxyportal
make build
# → bin/userproxyportal
```

### Installer directement depuis les sources

```bash
make build
./bin/userproxyportal install
```

---

## Déploiement en parc

Processus recommandé pour déployer sur plusieurs machines :

1. Préparer `/etc/userproxyportal/config.yaml` et le certificat `.cer`
2. Distribuer le binaire (via Ansible, script, clé USB...)
3. Sur chaque machine, en tant qu'utilisateur :
   ```bash
   ./userproxyportal install
   systemctl --user enable --now userproxyportal.service
   userproxyportal   # saisir ses credentials AD
   ```

Le fichier `config.yaml` peut être pré-déployé par l'administration avant que l'utilisateur ne saisisse ses credentials.

---

## Architecture technique

```
/usr/local/bin/
  userproxyportal          ← binaire unique (tout-en-un)

/etc/userproxyportal/
  config.yaml              ← configuration proxy (admin)
  entreprise-ca.cer        ← certificat SSL (optionnel)

/etc/systemd/user/
  userproxyportal.service  ← service utilisateur (auto-démarrage)

/usr/local/share/ca-certificates/
  entreprise-ca.crt        ← CA installée dans le trust store système

~/.local/share/keyrings/   ← credentials (GNOME Keyring, chiffré)
```

### Flux d'authentification

1. Le daemon lit les credentials depuis le GNOME Keyring au démarrage
2. Pour chaque requête HTTP : ajoute l'en-tête `Proxy-Authorization: Basic <base64>`
3. Pour chaque tunnel HTTPS (`CONNECT`) : négocie l'auth avec le proxy upstream, puis crée un tunnel transparent

### Sécurité

- Les credentials ne transitent jamais en clair dans les fichiers système
- `/etc/environment` contient uniquement `http_proxy=http://127.0.0.1:3128`
- Le stockage est délégué au GNOME Keyring (chiffré avec le mot de passe de session)
- Les opérations root (installation, certificat, `/etc/environment`) passent par `pkexec` avec une seule élévation de privilèges
