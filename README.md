# sshkm

Профессиональный CLI-менеджер SSH-ключей для macOS, Debian и Ubuntu.

`sshkm` хранит управляемые ключи в изолированной директории `~/.ssh/sshkm`, генерирует/импортирует пары ключей, выводит fingerprint и интегрируется с `ssh-agent`.

## Возможности

- Безопасное хранение ключей с корректными правами (`0700` для каталога, `0600`/`0644` для ключей)
- Генерация ключей (`ed25519`, `rsa`)
- Импорт существующих ключей
- Просмотр публичного ключа и fingerprint
- Добавление/удаление ключа из `ssh-agent`
- Автоматическая сборка релизов для Linux/macOS

## Установка

### Homebrew (macOS/Linux)

После публикации релиза:

```bash
brew tap RTHeLL/tap
brew install sshkm
```

### Debian/Ubuntu (APT/.deb)

После публикации релиза:

```bash
sudo apt install ./sshkm_<version>_linux_amd64.deb
```

### Ручная установка

Скачайте архив из Releases, распакуйте и переместите бинарник в `PATH`:

```bash
install -m 0755 sshkm /usr/local/bin/sshkm
```

## Быстрый старт

```bash
sshkm init
sshkm generate work --comment "work@laptop"
sshkm list
sshkm public work
sshkm fingerprint work
sshkm agent add work
```

## Команды

- `sshkm init` — инициализировать каталог менеджера
- `sshkm list` — показать управляемые ключи
- `sshkm generate <name>` — сгенерировать ключ
- `sshkm import <name> --from <private-key-path>` — импортировать ключ
- `sshkm public <name>` — вывести публичный ключ
- `sshkm fingerprint <name>` — вывести fingerprint
- `sshkm agent add <name>` — добавить ключ в `ssh-agent`
- `sshkm agent remove <name>` — удалить ключ из `ssh-agent`
- `sshkm delete <name>` — удалить ключ из менеджера
- `sshkm version` — версия сборки

## Публикация в пакетные менеджеры

Проект использует `GoReleaser`:

- `Homebrew` formula публикуется в `RTHeLL/homebrew-tap`
- `.deb` пакет публикуется как артефакт релиза (для Debian/Ubuntu)

Триггер публикации:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Перед запуском релиза добавьте GitHub Actions secret:

- `HOMEBREW_TAP_GITHUB_TOKEN` — токен с доступом на запись в tap-репозиторий.

## Локальная разработка

```bash
go test ./...
go run ./cmd/sshkm --help
```
